// Cloud Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by CB-Spider Team, 2025.06.

package resources

import (
	"fmt"
	"strings"
	"time"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/extensions/attachinterfaces"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/servers"
	nhnl3fips "github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/ports"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NhnCloudNICHandler struct {
	RegionInfo    idrv.RegionInfo
	NetworkClient *nhnsdk.ServiceClient
	VMClient      *nhnsdk.ServiceClient
}

// mappingNICInfo converts a NHN Cloud port to NICInfo.
func (nicHandler *NhnCloudNICHandler) mappingNICInfo(port *ports.Port) (irs.NICInfo, error) {
	nicInfo := irs.NICInfo{
		IId: irs.IID{
			NameId:   port.Name,
			SystemId: port.ID,
		},
		VpcIID: irs.IID{
			NameId:   "",
			SystemId: port.NetworkID,
		},
		MACAddress: port.MACAddress,
	}

	// Use description as NameId fallback if name is empty
	if port.Name == "" && port.Description != "" {
		nicInfo.IId.NameId = port.Description
	}

	// Resolve VPC name
	if vpc, err := getVPCWithId(nicHandler.NetworkClient, port.NetworkID); err == nil {
		nicInfo.VpcIID.NameId = vpc.Name
	}

	// Extract subnet and IP info from FixedIPs
	if len(port.FixedIPs) > 0 {
		subnetID := port.FixedIPs[0].SubnetID
		nicInfo.SubnetIID = irs.IID{
			NameId:   "",
			SystemId: subnetID,
		}
		nicInfo.PrivateIP = port.FixedIPs[0].IPAddress
		// Resolve Subnet name
		if sub, err := getVpcsubnetWithId(nicHandler.NetworkClient, subnetID); err == nil {
			nicInfo.SubnetIID.NameId = sub.Name
		}
	}

	// Build index-aligned PublicIPs from floating IPs on this port
	fipMap := map[string]string{} // fixedIP -> floatingIP
	allFipPages, fipErr := nhnl3fips.List(nicHandler.NetworkClient, nhnl3fips.ListOpts{PortID: port.ID}).AllPages()
	if fipErr == nil {
		if fips, fipErr2 := nhnl3fips.ExtractFloatingIPs(allFipPages); fipErr2 == nil {
			for _, fip := range fips {
				fipMap[fip.FixedIP] = fip.FloatingIP
			}
		}
	}
	for _, ip := range port.FixedIPs {
		nicInfo.PrivateIPs = append(nicInfo.PrivateIPs, ip.IPAddress)
		nicInfo.PublicIPs = append(nicInfo.PublicIPs, fipMap[ip.IPAddress])
	}
	for _, pip := range nicInfo.PublicIPs {
		if pip != "" {
			nicInfo.PublicIP = pip
			break
		}
	}

	// Security groups
	for _, sgID := range port.SecurityGroups {
		nicInfo.SecurityGroupIIDs = append(nicInfo.SecurityGroupIIDs, irs.IID{
			NameId:   "",
			SystemId: sgID,
		})
	}

	// Status
	if port.DeviceOwner == "compute:nova" || port.DeviceOwner == "compute:AZ1" ||
		(port.DeviceOwner != "" && port.DeviceID != "") {
		nicInfo.Status = irs.NICAttached
		nicInfo.OwnerVM = irs.IID{
			NameId:   port.DeviceID,
			SystemId: port.DeviceID,
		}
		// Resolve VM name
		if vmName, err := nicHandler.getServerNameByID(port.DeviceID); err == nil {
			nicInfo.OwnerVM.NameId = vmName
		}
	} else if port.Status == "ERROR" {
		nicInfo.Status = irs.NICError
	} else {
		nicInfo.Status = irs.NICAvailable
	}

	nicInfo.CreatedTime = time.Now() // NHN ports API does not return created_at in all versions

	return nicInfo, nil
}

// getServerNameByID returns the server name for a given server UUID.
func (nicHandler *NhnCloudNICHandler) getServerNameByID(serverID string) (string, error) {
	allPages, err := servers.List(nicHandler.VMClient, servers.ListOpts{}).AllPages()
	if err != nil {
		return "", err
	}
	serverList, err := servers.ExtractServers(allPages)
	if err != nil {
		return "", err
	}
	for _, s := range serverList {
		if s.ID == serverID {
			return s.Name, nil
		}
	}
	return "", fmt.Errorf("server not found with ID: %s", serverID)
}

// ListIID lists all NIC IIDs.
func (nicHandler *NhnCloudNICHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("NHN Cloud Driver: called NIC ListIID()")
	InitLog()

	callLogInfo := getCallLogScheme(nicHandler.RegionInfo.Region, call.NIC, "ListIID()", "ListIID()")
	start := call.Start()

	allPages, err := ports.List(nicHandler.NetworkClient, ports.ListOpts{}).AllPages()
	if err != nil {
		return nil, logAndReturnError(callLogInfo, "Failed to list NIC pages:", err)
	}
	portList, err := ports.ExtractPorts(allPages)
	if err != nil {
		return nil, logAndReturnError(callLogInfo, "Failed to extract NICs:", err)
	}
	LoggingInfo(callLogInfo, start)

	var iidList []*irs.IID
	for _, port := range portList {
		nameId := port.Name
		if nameId == "" {
			nameId = port.Description
		}
		iidList = append(iidList, &irs.IID{
			NameId:   nameId,
			SystemId: port.ID,
		})
	}
	return iidList, nil
}

// CreateNIC creates a new NIC (port).
func (nicHandler *NhnCloudNICHandler) CreateNIC(nicReqInfo irs.NICReqInfo) (irs.NICInfo, error) {
	cblogger.Info("NHN Cloud Driver: called NIC CreateNIC()")
	InitLog()

	callLogInfo := getCallLogScheme(nicHandler.RegionInfo.Region, call.NIC, nicReqInfo.IId.NameId, "CreateNIC()")
	start := call.Start()

	// Use VpcIID.SystemId as NetworkID (NHN uses network_id for VPC)
	networkID := nicReqInfo.VpcIID.SystemId
	if networkID == "" {
		networkID = nicReqInfo.SubnetIID.SystemId
	}

	createOpts := ports.CreateOpts{
		NetworkID:   networkID,
		Name:        nicReqInfo.IId.NameId,
		Description: nicReqInfo.IId.NameId,
	}

	// Set subnet via FixedIPs if SubnetIID provided
	if nicReqInfo.SubnetIID.SystemId != "" {
		type FixedIP struct {
			SubnetID string `json:"subnet_id"`
		}
		createOpts.FixedIPs = []FixedIP{{SubnetID: nicReqInfo.SubnetIID.SystemId}}
	}

	// Attach security groups
	if len(nicReqInfo.SecurityGroupIIDs) > 0 {
		sgIDs := make([]string, len(nicReqInfo.SecurityGroupIIDs))
		for i, sg := range nicReqInfo.SecurityGroupIIDs {
			sgIDs[i] = sg.SystemId
		}
		createOpts.SecurityGroups = &sgIDs
	}

	port, err := ports.Create(nicHandler.NetworkClient, createOpts).Extract()
	if err != nil {
		return irs.NICInfo{}, logAndReturnError(callLogInfo, "Failed to create NIC:", err)
	}
	LoggingInfo(callLogInfo, start)

	nicInfo, err := nicHandler.mappingNICInfo(port)
	if err != nil {
		return irs.NICInfo{}, err
	}
	// Set NameId from request
	nicInfo.IId.NameId = nicReqInfo.IId.NameId
	return nicInfo, nil
}

// ListNIC lists all NICs.
func (nicHandler *NhnCloudNICHandler) ListNIC() ([]*irs.NICInfo, error) {
	cblogger.Info("NHN Cloud Driver: called NIC ListNIC()")
	InitLog()

	callLogInfo := getCallLogScheme(nicHandler.RegionInfo.Region, call.NIC, "ListNIC()", "ListNIC()")
	start := call.Start()

	allPages, err := ports.List(nicHandler.NetworkClient, ports.ListOpts{}).AllPages()
	if err != nil {
		return nil, logAndReturnError(callLogInfo, "Failed to list NIC pages:", err)
	}
	portList, err := ports.ExtractPorts(allPages)
	if err != nil {
		return nil, logAndReturnError(callLogInfo, "Failed to extract NICs:", err)
	}
	LoggingInfo(callLogInfo, start)

	var nicInfoList []*irs.NICInfo
	for _, port := range portList {
		p := port
		nicInfo, err := nicHandler.mappingNICInfo(&p)
		if err != nil {
			cblogger.Warningf("Failed to map NIC info for port %s: %v", port.ID, err)
			continue
		}
		nicInfoList = append(nicInfoList, &nicInfo)
	}
	return nicInfoList, nil
}

// GetNIC retrieves a NIC by IID.
func (nicHandler *NhnCloudNICHandler) GetNIC(nicIID irs.IID) (irs.NICInfo, error) {
	cblogger.Info("NHN Cloud Driver: called NIC GetNIC()")
	InitLog()

	callLogInfo := getCallLogScheme(nicHandler.RegionInfo.Region, call.NIC, nicIID.SystemId, "GetNIC()")
	start := call.Start()

	// If SystemId provided, use ports.Get directly
	if nicIID.SystemId != "" {
		port, err := ports.Get(nicHandler.NetworkClient, nicIID.SystemId).Extract()
		if err != nil {
			return irs.NICInfo{}, logAndReturnError(callLogInfo, "Failed to get NIC:", err)
		}
		LoggingInfo(callLogInfo, start)
		return nicHandler.mappingNICInfo(port)
	}

	// Otherwise list and find by NameId or Description
	allPages, err := ports.List(nicHandler.NetworkClient, ports.ListOpts{}).AllPages()
	if err != nil {
		return irs.NICInfo{}, logAndReturnError(callLogInfo, "Failed to list NIC pages:", err)
	}
	portList, err := ports.ExtractPorts(allPages)
	if err != nil {
		return irs.NICInfo{}, logAndReturnError(callLogInfo, "Failed to extract NICs:", err)
	}
	LoggingInfo(callLogInfo, start)

	for _, port := range portList {
		if port.Name == nicIID.NameId || port.Description == nicIID.NameId {
			p := port
			return nicHandler.mappingNICInfo(&p)
		}
	}
	return irs.NICInfo{}, fmt.Errorf("NIC with NameId '%s' not found", nicIID.NameId)
}

// DeleteNIC deletes a NIC by IID.
func (nicHandler *NhnCloudNICHandler) DeleteNIC(nicIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called NIC DeleteNIC()")
	InitLog()

	callLogInfo := getCallLogScheme(nicHandler.RegionInfo.Region, call.NIC, nicIID.SystemId, "DeleteNIC()")
	start := call.Start()

	// Resolve SystemId if only NameId is given
	portID := nicIID.SystemId
	if portID == "" {
		nicInfo, err := nicHandler.GetNIC(nicIID)
		if err != nil {
			return false, logAndReturnError(callLogInfo, "Failed to find NIC:", err)
		}
		portID = nicInfo.IId.SystemId
	}

	err := ports.Delete(nicHandler.NetworkClient, portID).ExtractErr()
	if err != nil {
		return false, logAndReturnError(callLogInfo, "Failed to delete NIC:", err)
	}
	LoggingInfo(callLogInfo, start)
	return true, nil
}

// AttachNIC is not supported by NHN Cloud.
// NHN Cloud does not expose the Nova os-interface API (/v2/ and /v2.1/ both return 404),
// and direct Neutron port device_id update is also rejected (500).
func (nicHandler *NhnCloudNICHandler) AttachNIC(nicIID irs.IID, vmIID irs.IID) (irs.NICInfo, error) {
	return irs.NICInfo{}, fmt.Errorf("NHN Cloud does not support NIC Attach/Detach via API. Please use the NHN Cloud Console to manage network interfaces.")
}

func (nicHandler *NhnCloudNICHandler) attachNICUnsupported(nicIID irs.IID, vmIID irs.IID) (irs.NICInfo, error) {
	cblogger.Info("NHN Cloud Driver: called NIC AttachNIC()")
	InitLog()

	callLogInfo := getCallLogScheme(nicHandler.RegionInfo.Region, call.NIC, nicIID.SystemId, "AttachNIC()")
	start := call.Start()

	// Resolve port SystemId
	portID := nicIID.SystemId
	if portID == "" {
		nicInfo, err := nicHandler.GetNIC(nicIID)
		if err != nil {
			return irs.NICInfo{}, logAndReturnError(callLogInfo, "Failed to find NIC:", err)
		}
		portID = nicInfo.IId.SystemId
	}

	// Resolve VM SystemId
	serverID := vmIID.SystemId
	if serverID == "" {
		return irs.NICInfo{}, fmt.Errorf("VM SystemId is required for AttachNIC")
	}

	// NHN Cloud's service catalog registers /v2/ but os-interface is only available at /v2.1/
	vmClient21 := *nicHandler.VMClient
	vmClient21.Endpoint = strings.Replace(nicHandler.VMClient.Endpoint, "/v2/", "/v2.1/", 1)
	vmClient21.ResourceBase = ""

	_, err := attachinterfaces.Create(&vmClient21, serverID, attachinterfaces.CreateOpts{
		PortID: portID,
	}).Extract()
	if err != nil {
		return irs.NICInfo{}, logAndReturnError(callLogInfo, "Failed to attach NIC to VM:", err)
	}
	LoggingInfo(callLogInfo, start)

	// Poll until port DeviceID is set
	for i := 0; i < 20; i++ {
		time.Sleep(2 * time.Second)
		p, pErr := ports.Get(nicHandler.NetworkClient, portID).Extract()
		if pErr != nil {
			break
		}
		if p.DeviceID != "" {
			break
		}
	}

	return nicHandler.GetNIC(irs.IID{SystemId: portID})
}

// DetachNIC is not supported by NHN Cloud (same limitation as AttachNIC).
func (nicHandler *NhnCloudNICHandler) DetachNIC(nicIID irs.IID) (bool, error) {
	return false, fmt.Errorf("NHN Cloud does not support NIC Attach/Detach via API. Please use the NHN Cloud Console to manage network interfaces.")
}

func (nicHandler *NhnCloudNICHandler) detachNICUnsupported(nicIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called NIC DetachNIC()")
	InitLog()

	callLogInfo := getCallLogScheme(nicHandler.RegionInfo.Region, call.NIC, nicIID.SystemId, "DetachNIC()")
	start := call.Start()

	nicInfo, err := nicHandler.GetNIC(nicIID)
	if err != nil {
		return false, logAndReturnError(callLogInfo, "Failed to find NIC:", err)
	}

	portID := nicInfo.IId.SystemId
	serverID := nicInfo.OwnerVM.SystemId
	if serverID == "" {
		return false, fmt.Errorf("NIC '%s' is not attached to any VM", portID)
	}

	vmClient21 := *nicHandler.VMClient
	vmClient21.Endpoint = strings.Replace(nicHandler.VMClient.Endpoint, "/v2/", "/v2.1/", 1)
	vmClient21.ResourceBase = ""

	err = attachinterfaces.Delete(&vmClient21, serverID, portID).ExtractErr()
	if err != nil {
		return false, logAndReturnError(callLogInfo, "Failed to detach NIC from VM:", err)
	}
	LoggingInfo(callLogInfo, start)

	// Poll until port DeviceID is cleared
	for i := 0; i < 20; i++ {
		time.Sleep(2 * time.Second)
		p, pErr := ports.Get(nicHandler.NetworkClient, portID).Extract()
		if pErr != nil {
			break
		}
		if p.DeviceID == "" {
			break
		}
	}

	return true, nil
}

// getRawPort fetches a raw ports.Port by IID (SystemId preferred, then NameId scan).
func (h *NhnCloudNICHandler) getRawPort(nicIID irs.IID) (*ports.Port, error) {
	if nicIID.SystemId != "" {
		return ports.Get(h.NetworkClient, nicIID.SystemId).Extract()
	}
	allPages, err := ports.List(h.NetworkClient, ports.ListOpts{}).AllPages()
	if err != nil {
		return nil, err
	}
	portList, err := ports.ExtractPorts(allPages)
	if err != nil {
		return nil, err
	}
	for _, p := range portList {
		if p.Name == nicIID.NameId || p.Description == nicIID.NameId {
			pp := p
			return &pp, nil
		}
	}
	return nil, fmt.Errorf("NIC not found with NameId: %s", nicIID.NameId)
}

// AddPrivateIP adds a secondary private IP to a NHN Cloud NIC (Neutron port) using ports.Update.
func (h *NhnCloudNICHandler) AddPrivateIP(nicIID irs.IID, privateIP string) (irs.NICInfo, error) {
	port, err := h.getRawPort(nicIID)
	if err != nil {
		return irs.NICInfo{}, fmt.Errorf("NhnCloudNICHandler.AddPrivateIP: failed to get port: %w", err)
	}

	// Build updated FixedIPs list — SubnetID is required by Neutron
	if len(port.FixedIPs) == 0 {
		return irs.NICInfo{}, fmt.Errorf("NhnCloudNICHandler.AddPrivateIP: port has no existing FixedIPs to derive SubnetID")
	}
	subnetID := port.FixedIPs[0].SubnetID
	type fixedIPEntry struct {
		SubnetID  string `json:"subnet_id"`
		IPAddress string `json:"ip_address,omitempty"`
	}
	newFixedIPs := make([]fixedIPEntry, 0, len(port.FixedIPs)+1)
	for _, fip := range port.FixedIPs {
		newFixedIPs = append(newFixedIPs, fixedIPEntry{SubnetID: fip.SubnetID, IPAddress: fip.IPAddress})
	}
	newFixedIPs = append(newFixedIPs, fixedIPEntry{SubnetID: subnetID, IPAddress: privateIP})

	updated, err := ports.Update(h.NetworkClient, port.ID, ports.UpdateOpts{
		FixedIPs: newFixedIPs,
	}).Extract()
	if err != nil {
		return irs.NICInfo{}, fmt.Errorf("NhnCloudNICHandler.AddPrivateIP: failed to update port: %w", err)
	}

	return h.mappingNICInfo(updated)
}

// RemovePrivateIP removes a secondary private IP from a NHN Cloud NIC (Neutron port) using ports.Update.
func (h *NhnCloudNICHandler) RemovePrivateIP(nicIID irs.IID, privateIP string) (bool, error) {
	port, err := h.getRawPort(nicIID)
	if err != nil {
		return false, fmt.Errorf("NhnCloudNICHandler.RemovePrivateIP: failed to get port: %w", err)
	}

	type fixedIPEntry struct {
		SubnetID  string `json:"subnet_id"`
		IPAddress string `json:"ip_address,omitempty"`
	}
	newFixedIPs := make([]fixedIPEntry, 0, len(port.FixedIPs))
	found := false
	for _, fip := range port.FixedIPs {
		if fip.IPAddress == privateIP {
			found = true
			continue
		}
		newFixedIPs = append(newFixedIPs, fixedIPEntry{SubnetID: fip.SubnetID, IPAddress: fip.IPAddress})
	}
	if !found {
		return false, fmt.Errorf("NhnCloudNICHandler.RemovePrivateIP: IP %s not found on port %s", privateIP, port.ID)
	}

	_, err = ports.Update(h.NetworkClient, port.ID, ports.UpdateOpts{
		FixedIPs: newFixedIPs,
	}).Extract()
	if err != nil {
		return false, fmt.Errorf("NhnCloudNICHandler.RemovePrivateIP: failed to update port: %w", err)
	}
	return true, nil
}

// GetNICOSConfigScript returns an empty string for NHN Cloud.
// NHN Cloud does not support AttachNIC via API (console only); no OS config script is provided.
func (nicHandler *NhnCloudNICHandler) GetNICOSConfigScript(nicIID irs.IID) (string, error) {
	return "", nil
}
