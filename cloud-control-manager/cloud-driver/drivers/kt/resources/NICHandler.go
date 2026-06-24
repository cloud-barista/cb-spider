// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// KT Cloud VPC NIC Handler.
//
// by ETRI, 2025.

package resources

import (
	"fmt"
	"strings"
	"time"

	ktvpcsdk "github.com/cloud-barista/ktcloudvpc-sdk-go"
	attachinterfaces "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/compute/v2/extensions/attachinterfaces"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/compute/v2/servers"
	ktl3fips "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/networks"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/ports"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type KTVpcNICHandler struct {
	RegionInfo    idrv.RegionInfo
	NetworkClient *ktvpcsdk.ServiceClient
	VMClient      *ktvpcsdk.ServiceClient
}

// ListIID returns a list of IIDs for all NICs (ports).
func (nicHandler *KTVpcNICHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("KT Cloud VPC Driver: called NICHandler ListIID()")
	InitLog()

	callLogInfo := getCallLogScheme(nicHandler.RegionInfo.Zone, call.NIC, "ListIID()", "ListIID()")
	start := call.Start()

	allPages, err := ports.List(nicHandler.NetworkClient, ports.ListOpts{}).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to list NIC pages: [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	portList, err := ports.ExtractPorts(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to extract NICs: [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	loggingInfo(callLogInfo, start)

	var iidList []*irs.IID
	for _, port := range portList {
		iid := &irs.IID{
			NameId:   port.Name,
			SystemId: port.ID,
		}
		iidList = append(iidList, iid)
	}
	return iidList, nil
}

// CreateNIC creates a new NIC (port) in the specified subnet.
// Note: KT Cloud VPC does not support standalone Neutron port creation (POST /ports returns computeFault 500).
// NICs in KT Cloud are created implicitly when attaching an interface to a VM via the Nova os-interface API.
func (nicHandler *KTVpcNICHandler) CreateNIC(nicReqInfo irs.NICReqInfo) (irs.NICInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called NICHandler CreateNIC()")
	InitLog()

	callLogInfo := getCallLogScheme(nicHandler.RegionInfo.Zone, call.NIC, nicReqInfo.IId.NameId, "CreateNIC()")
	newErr := fmt.Errorf("KT Cloud does not support standalone NIC creation. NICs are created implicitly when attaching an interface to a VM.")
	loggingError(callLogInfo, newErr)
	return irs.NICInfo{}, newErr
}

// ListNIC returns a list of all NIC info.
func (nicHandler *KTVpcNICHandler) ListNIC() ([]*irs.NICInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called NICHandler ListNIC()")
	InitLog()

	callLogInfo := getCallLogScheme(nicHandler.RegionInfo.Zone, call.NIC, "ListNIC()", "ListNIC()")
	start := call.Start()

	allPages, err := ports.List(nicHandler.NetworkClient, ports.ListOpts{}).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to list NIC pages: [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	portList, err := ports.ExtractPorts(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to extract NICs: [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	loggingInfo(callLogInfo, start)

	var nicInfoList []*irs.NICInfo
	for _, port := range portList {
		portCopy := port
		nicInfo, err := nicHandler.mappingNICInfo(&portCopy)
		if err != nil {
			cblogger.Warningf("Failed to map NIC info for port [%s]: %v", port.ID, err)
			continue
		}
		nicInfoList = append(nicInfoList, &nicInfo)
	}
	return nicInfoList, nil
}

// GetNIC returns NIC info for the specified NIC IID.
func (nicHandler *KTVpcNICHandler) GetNIC(nicIID irs.IID) (irs.NICInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called NICHandler GetNIC()")
	InitLog()

	callLogInfo := getCallLogScheme(nicHandler.RegionInfo.Zone, call.NIC, nicIID.SystemId, "GetNIC()")
	start := call.Start()

	port, err := nicHandler.getPort(nicIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to get NIC: [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NICInfo{}, newErr
	}
	loggingInfo(callLogInfo, start)

	nicInfo, err := nicHandler.mappingNICInfo(port)
	if err != nil {
		newErr := fmt.Errorf("Failed to map NIC info: [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.NICInfo{}, newErr
	}
	return nicInfo, nil
}

// DeleteNIC deletes the specified NIC (port).
func (nicHandler *KTVpcNICHandler) DeleteNIC(nicIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called NICHandler DeleteNIC()")
	InitLog()

	callLogInfo := getCallLogScheme(nicHandler.RegionInfo.Zone, call.NIC, nicIID.SystemId, "DeleteNIC()")
	start := call.Start()

	port, err := nicHandler.getPort(nicIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to get NIC to delete: [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	// Check if port is still attached to a VM
	if strings.Contains(port.DeviceOwner, "compute") {
		newErr := fmt.Errorf("The NIC [%s] is attached to a VM. Detach first before deleting.", port.ID)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	err = ports.Delete(nicHandler.NetworkClient, port.ID).ExtractErr()
	if err != nil {
		newErr := fmt.Errorf("Failed to delete NIC [%s]: [%v]", port.ID, err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}
	loggingInfo(callLogInfo, start)
	return true, nil
}

// AttachNIC attaches the specified NIC to a VM using attachinterfaces.
func (nicHandler *KTVpcNICHandler) AttachNIC(nicIID irs.IID, vmIID irs.IID) (irs.NICInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called NICHandler AttachNIC()")
	InitLog()

	callLogInfo := getCallLogScheme(nicHandler.RegionInfo.Zone, call.NIC, nicIID.SystemId, "AttachNIC()")
	start := call.Start()

	port, err := nicHandler.getPort(nicIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to get NIC to attach: [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NICInfo{}, newErr
	}

	serverID := vmIID.SystemId
	if strings.EqualFold(serverID, "") {
		newErr := fmt.Errorf("Invalid VM SystemId!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NICInfo{}, newErr
	}

	_, err = attachinterfaces.Create(nicHandler.VMClient, serverID, attachinterfaces.CreateOpts{
		PortID: port.ID,
	}).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to attach NIC [%s] to VM [%s]: [%v]", port.ID, serverID, err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.NICInfo{}, newErr
	}
	loggingInfo(callLogInfo, start)

	// Re-fetch updated port info
	updatedPort, err := ports.Get(nicHandler.NetworkClient, port.ID).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to get updated NIC info after attach: [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.NICInfo{}, newErr
	}

	nicInfo, err := nicHandler.mappingNICInfo(updatedPort)
	if err != nil {
		newErr := fmt.Errorf("Failed to map NIC info: [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.NICInfo{}, newErr
	}
	return nicInfo, nil
}

// DetachNIC detaches the specified NIC from its attached VM.
func (nicHandler *KTVpcNICHandler) DetachNIC(nicIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called NICHandler DetachNIC()")
	InitLog()

	callLogInfo := getCallLogScheme(nicHandler.RegionInfo.Zone, call.NIC, nicIID.SystemId, "DetachNIC()")
	start := call.Start()

	port, err := nicHandler.getPort(nicIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to get NIC to detach: [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	if !strings.Contains(port.DeviceOwner, "compute") {
		newErr := fmt.Errorf("The NIC [%s] is not attached to any VM.", port.ID)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	serverID := port.DeviceID
	if strings.EqualFold(serverID, "") {
		newErr := fmt.Errorf("Failed to get VM ID from NIC [%s]", port.ID)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	err = attachinterfaces.Delete(nicHandler.VMClient, serverID, port.ID).ExtractErr()
	if err != nil {
		newErr := fmt.Errorf("Failed to detach NIC [%s] from VM [%s]: [%v]", port.ID, serverID, err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}
	loggingInfo(callLogInfo, start)
	return true, nil
}

// ==================== Internal Helpers ====================

// getPort fetches a port by NameId or SystemId.
func (nicHandler *KTVpcNICHandler) getPort(nicIID irs.IID) (*ports.Port, error) {
	if !strings.EqualFold(nicIID.SystemId, "") {
		port, err := ports.Get(nicHandler.NetworkClient, nicIID.SystemId).Extract()
		if err != nil {
			return nil, fmt.Errorf("Failed to get port with SystemId [%s]: [%v]", nicIID.SystemId, err)
		}
		return port, nil
	}

	// Search by NameId
	if strings.EqualFold(nicIID.NameId, "") {
		return nil, fmt.Errorf("Both NameId and SystemId are empty")
	}

	allPages, err := ports.List(nicHandler.NetworkClient, ports.ListOpts{Name: nicIID.NameId}).AllPages()
	if err != nil {
		return nil, fmt.Errorf("Failed to list ports by name: [%v]", err)
	}
	portList, err := ports.ExtractPorts(allPages)
	if err != nil {
		return nil, fmt.Errorf("Failed to extract ports: [%v]", err)
	}
	for _, port := range portList {
		if strings.EqualFold(port.Name, nicIID.NameId) {
			return &port, nil
		}
	}
	return nil, fmt.Errorf("Failed to find NIC with NameId [%s]", nicIID.NameId)
}

// mappingNICInfo converts a ports.Port to irs.NICInfo.
func (nicHandler *KTVpcNICHandler) mappingNICInfo(port *ports.Port) (irs.NICInfo, error) {
	nicInfo := irs.NICInfo{
		IId: irs.IID{
			NameId:   port.Name,
			SystemId: port.ID,
		},
		MACAddress: port.MACAddress,
	}

	// Determine status and resolve OwnerVM name
	if strings.Contains(port.DeviceOwner, "compute") {
		nicInfo.Status = irs.NICAttached
		ownerVMNameId := port.DeviceID
		if nicHandler.VMClient != nil && port.DeviceID != "" {
			if vm, err := servers.Get(nicHandler.VMClient, port.DeviceID).Extract(); err == nil && vm.Name != "" {
				ownerVMNameId = vm.Name
			}
		}
		nicInfo.OwnerVM = irs.IID{NameId: ownerVMNameId, SystemId: port.DeviceID}
	} else {
		nicInfo.Status = irs.NICAvailable
	}

	// Primary private IP, subnet, and VPC
	if len(port.FixedIPs) > 0 {
		subnetID := port.FixedIPs[0].SubnetID
		nicInfo.PrivateIP = port.FixedIPs[0].IPAddress
		nicInfo.SubnetIID = irs.IID{SystemId: subnetID}

		subnet, err := getSubnetWithId(nicHandler.NetworkClient, subnetID)
		if err == nil {
			nicInfo.SubnetIID.NameId = subnet.NetworkName // KT subnet name = tier name
			nicInfo.VpcIID = irs.IID{SystemId: subnet.NetworkID}
			if net, netErr := networks.Get(nicHandler.NetworkClient, subnet.NetworkID).ExtractVPC(); netErr == nil {
				nicInfo.VpcIID.NameId = net.Name
			}
		}

		// Build FloatingIP map (fixedIP → publicIP) for PublicIPs index-alignment
		fipMap := map[string]string{}
		fipPages, fipErr := ktl3fips.List(nicHandler.NetworkClient, ktl3fips.ListOpts{PortID: port.ID}).AllPages()
		if fipErr == nil {
			if fips, fipErr2 := ktl3fips.ExtractFloatingIPs(fipPages); fipErr2 == nil {
				for _, fip := range fips {
					fipMap[fip.FixedIP] = fip.FloatingIP
				}
			}
		}

		for _, fixedIP := range port.FixedIPs {
			if fixedIP.IPAddress != "" {
				nicInfo.PrivateIPs = append(nicInfo.PrivateIPs, fixedIP.IPAddress)
				nicInfo.PublicIPs = append(nicInfo.PublicIPs, fipMap[fixedIP.IPAddress])
			}
		}
	}

	// Security groups
	if len(port.SecurityGroups) > 0 {
		var sgIIDs []irs.IID
		for _, sgID := range port.SecurityGroups {
			sgIIDs = append(sgIIDs, irs.IID{SystemId: sgID})
		}
		nicInfo.SecurityGroupIIDs = sgIIDs
	}

	// KeyValueList
	nicInfo.KeyValueList = []irs.KeyValue{
		{Key: "NetworkID", Value: port.NetworkID},
		{Key: "DeviceOwner", Value: port.DeviceOwner},
		{Key: "DeviceID", Value: port.DeviceID},
		{Key: "AdminStateUp", Value: fmt.Sprintf("%v", port.AdminStateUp)},
		{Key: "PortStatus", Value: port.Status},
	}

	nicInfo.CreatedTime = time.Now()
	return nicInfo, nil
}


// AddPrivateIP adds a secondary private IP to a KT Cloud VPC NIC (port) using ports.Update.
func (h *KTVpcNICHandler) AddPrivateIP(nicIID irs.IID, privateIP string) (irs.NICInfo, error) {
	port, err := h.getPort(nicIID)
	if err != nil {
		return irs.NICInfo{}, fmt.Errorf("KTVpcNICHandler.AddPrivateIP: failed to get port: %w", err)
	}

	type fixedIPEntry struct {
		SubnetID  string `json:"subnet_id"`
		IPAddress string `json:"ip_address,omitempty"`
	}
	subnetID := port.FixedIPs[0].SubnetID
	newFixedIPs := make([]fixedIPEntry, 0, len(port.FixedIPs)+1)
	for _, fip := range port.FixedIPs {
		newFixedIPs = append(newFixedIPs, fixedIPEntry{SubnetID: fip.SubnetID, IPAddress: fip.IPAddress})
	}
	newFixedIPs = append(newFixedIPs, fixedIPEntry{SubnetID: subnetID, IPAddress: privateIP})

	updated, err := ports.Update(h.NetworkClient, port.ID, ports.UpdateOpts{
		FixedIPs: newFixedIPs,
	}).Extract()
	if err != nil {
		return irs.NICInfo{}, fmt.Errorf("KTVpcNICHandler.AddPrivateIP: failed to update port: %w", err)
	}

	return h.mappingNICInfo(updated)
}

// RemovePrivateIP removes a secondary private IP from a KT Cloud VPC NIC (port) using ports.Update.
func (h *KTVpcNICHandler) RemovePrivateIP(nicIID irs.IID, privateIP string) (bool, error) {
	port, err := h.getPort(nicIID)
	if err != nil {
		return false, fmt.Errorf("KTVpcNICHandler.RemovePrivateIP: failed to get port: %w", err)
	}

	// Disassociate FloatingIP bound to this privateIP before removing FixedIP
	h.disassociateFloatingIPByFixedIP(port.ID, privateIP)

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
		return false, fmt.Errorf("KTVpcNICHandler.RemovePrivateIP: IP %s not found on port %s", privateIP, port.ID)
	}

	_, err = ports.Update(h.NetworkClient, port.ID, ports.UpdateOpts{
		FixedIPs: newFixedIPs,
	}).Extract()
	if err != nil {
		return false, fmt.Errorf("KTVpcNICHandler.RemovePrivateIP: failed to update port: %w", err)
	}
	return true, nil
}

// disassociateFloatingIPByFixedIP disassociates any FloatingIP bound to the given fixedIP on portID.
func (h *KTVpcNICHandler) disassociateFloatingIPByFixedIP(portID, fixedIP string) {
	allPages, err := ktl3fips.List(h.NetworkClient, ktl3fips.ListOpts{PortID: portID}).AllPages()
	if err != nil {
		return
	}
	fips, err := ktl3fips.ExtractFloatingIPs(allPages)
	if err != nil {
		return
	}
	emptyPort := ""
	for _, fip := range fips {
		if fip.FixedIP == fixedIP || fixedIP == "" {
			ktl3fips.Update(h.NetworkClient, fip.ID, ktl3fips.UpdateOpts{PortID: &emptyPort})
		}
	}
}

// GetNICOSConfigScript returns an empty string for KT Cloud.
// KT Cloud NIC independent creation API is not supported (POST /ports → 500); no OS config script is provided.
func (h *KTVpcNICHandler) GetNICOSConfigScript(nicIID irs.IID) (string, error) {
	return "", nil
}
