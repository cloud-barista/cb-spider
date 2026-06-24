// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is NIC (Neutron Port) Handler for OpenStack.
//
// by CB-Spider Team, 2025.06.

package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/attachinterfaces"
	"github.com/gophercloud/gophercloud/v2/openstack/compute/v2/servers"
	layer3floatingips "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/v2/pagination"
)

type OpenStackNICHandler struct {
	Region        idrv.RegionInfo
	NetworkClient *gophercloud.ServiceClient
	ComputeClient *gophercloud.ServiceClient
}

// ListIID returns a list of IIDs for all NIC (Neutron port) resources.
func (nicHandler *OpenStackNICHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region.Region, call.NIC, "NIC", "ListIID()")
	start := call.Start()

	var iidList []*irs.IID
	err := ports.List(nicHandler.NetworkClient, ports.ListOpts{}).EachPage(context.TODO(), func(ctx context.Context, page pagination.Page) (bool, error) {
		portList, err := ports.ExtractPorts(page)
		if err != nil {
			return false, err
		}
		for _, p := range portList {
			iid := &irs.IID{
				NameId:   p.Name,
				SystemId: p.ID,
			}
			iidList = append(iidList, iid)
		}
		return true, nil
	})
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List NIC IID. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

// CreateNIC creates a new Neutron port (NIC).
func (nicHandler *OpenStackNICHandler) CreateNIC(nicReqInfo irs.NICReqInfo) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region.Region, call.NIC, nicReqInfo.IId.NameId, "CreateNIC()")
	start := call.Start()

	// Resolve NetworkID from VpcIID
	networkID := nicReqInfo.VpcIID.SystemId
	if networkID == "" {
		page, err := networks.List(nicHandler.NetworkClient, networks.ListOpts{Name: nicReqInfo.VpcIID.NameId}).AllPages(context.TODO())
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Create NIC. err = failed to look up VPC: %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.NICInfo{}, createErr
		}
		netList, err := networks.ExtractNetworks(page)
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Create NIC. err = failed to extract VPC list: %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.NICInfo{}, createErr
		}
		for _, net := range netList {
			if net.Name == nicReqInfo.VpcIID.NameId {
				networkID = net.ID
				break
			}
		}
		if networkID == "" {
			createErr := errors.New(fmt.Sprintf("Failed to Create NIC. err = VPC not found: %s", nicReqInfo.VpcIID.NameId))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.NICInfo{}, createErr
		}
	}

	// Resolve SubnetID from SubnetIID
	subnetID := nicReqInfo.SubnetIID.SystemId
	if subnetID == "" {
		sPage, err := subnets.List(nicHandler.NetworkClient, subnets.ListOpts{
			NetworkID: networkID,
			Name:      nicReqInfo.SubnetIID.NameId,
		}).AllPages(context.TODO())
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Create NIC. err = failed to look up Subnet: %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.NICInfo{}, createErr
		}
		subnetList, err := subnets.ExtractSubnets(sPage)
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Create NIC. err = failed to extract Subnet list: %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.NICInfo{}, createErr
		}
		for _, subnet := range subnetList {
			if subnet.Name == nicReqInfo.SubnetIID.NameId {
				subnetID = subnet.ID
				break
			}
		}
		if subnetID == "" {
			createErr := errors.New(fmt.Sprintf("Failed to Create NIC. err = Subnet not found: %s", nicReqInfo.SubnetIID.NameId))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.NICInfo{}, createErr
		}
	}

	// Build SecurityGroups list (neutron SG UUIDs — caller must provide SystemId)
	var sgIDs []string
	for _, sgIID := range nicReqInfo.SecurityGroupIIDs {
		if sgIID.SystemId != "" {
			sgIDs = append(sgIDs, sgIID.SystemId)
		}
		// Name-based lookup for neutron SGs is not implemented here;
		// callers should provide SystemId.
	}

	createOpts := ports.CreateOpts{
		NetworkID: networkID,
		Name:      nicReqInfo.IId.NameId,
		FixedIPs: []ports.IP{
			{SubnetID: subnetID},
		},
	}
	if len(sgIDs) > 0 {
		createOpts.SecurityGroups = &sgIDs
	}

	port, err := ports.Create(context.TODO(), nicHandler.NetworkClient, createOpts).Extract()
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create NIC. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NICInfo{}, createErr
	}

	nicInfo, err := nicHandler.portToNICInfo(port)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create NIC. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.NICInfo{}, createErr
	}

	LoggingInfo(hiscallInfo, start)
	return nicInfo, nil
}

// ListNIC returns a list of all NIC (Neutron port) resources.
func (nicHandler *OpenStackNICHandler) ListNIC() ([]*irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region.Region, call.NIC, "NIC", "ListNIC()")
	start := call.Start()

	var nicInfoList []*irs.NICInfo
	err := ports.List(nicHandler.NetworkClient, ports.ListOpts{}).EachPage(context.TODO(), func(ctx context.Context, page pagination.Page) (bool, error) {
		portList, err := ports.ExtractPorts(page)
		if err != nil {
			return false, err
		}
		for _, p := range portList {
			pCopy := p
			nicInfo, err := nicHandler.portToNICInfo(&pCopy)
			if err != nil {
				cblogger.Warn(fmt.Sprintf("Failed to convert port %s to NICInfo: %s", p.ID, err.Error()))
				continue
			}
			nicInfoList = append(nicInfoList, &nicInfo)
		}
		return true, nil
	})
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List NIC. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	LoggingInfo(hiscallInfo, start)
	return nicInfoList, nil
}

// GetNIC returns the NICInfo for the port identified by nicIID.
// If SystemId is set it performs a direct Get; otherwise it searches by name.
func (nicHandler *OpenStackNICHandler) GetNIC(nicIID irs.IID) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region.Region, call.NIC, nicIID.NameId, "GetNIC()")
	start := call.Start()

	port, err := nicHandler.getRawPort(nicIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get NIC. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.NICInfo{}, getErr
	}

	nicInfo, err := nicHandler.portToNICInfo(port)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get NIC. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.NICInfo{}, getErr
	}

	LoggingInfo(hiscallInfo, start)
	return nicInfo, nil
}

// DeleteNIC deletes the port identified by nicIID.
func (nicHandler *OpenStackNICHandler) DeleteNIC(nicIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region.Region, call.NIC, nicIID.NameId, "DeleteNIC()")
	start := call.Start()

	port, err := nicHandler.getRawPort(nicIID)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete NIC. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}

	err = ports.Delete(context.TODO(), nicHandler.NetworkClient, port.ID).ExtractErr()
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete NIC. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

// AttachNIC attaches the given NIC (port) to the specified VM (server).
func (nicHandler *OpenStackNICHandler) AttachNIC(nicIID irs.IID, vmIID irs.IID) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region.Region, call.NIC, nicIID.NameId, "AttachNIC()")
	start := call.Start()

	// Resolve port
	port, err := nicHandler.getRawPort(nicIID)
	if err != nil {
		attachErr := errors.New(fmt.Sprintf("Failed to Attach NIC. err = %s", err.Error()))
		cblogger.Error(attachErr.Error())
		LoggingError(hiscallInfo, attachErr)
		return irs.NICInfo{}, attachErr
	}

	// Resolve server ID
	serverID := vmIID.SystemId
	if serverID == "" {
		serverID, err = nicHandler.getServerIDByName(vmIID.NameId)
		if err != nil {
			attachErr := errors.New(fmt.Sprintf("Failed to Attach NIC. err = %s", err.Error()))
			cblogger.Error(attachErr.Error())
			LoggingError(hiscallInfo, attachErr)
			return irs.NICInfo{}, attachErr
		}
	}

	_, err = attachinterfaces.Create(context.TODO(), nicHandler.ComputeClient, serverID, attachinterfaces.CreateOpts{
		PortID: port.ID,
	}).Extract()
	if err != nil {
		attachErr := errors.New(fmt.Sprintf("Failed to Attach NIC. err = %s", err.Error()))
		cblogger.Error(attachErr.Error())
		LoggingError(hiscallInfo, attachErr)
		return irs.NICInfo{}, attachErr
	}

	// Poll until port DeviceID is set (ACTIVE)
	var updatedPort *ports.Port
	for i := 0; i < 20; i++ {
		time.Sleep(2 * time.Second)
		updatedPort, err = ports.Get(context.TODO(), nicHandler.NetworkClient, port.ID).Extract()
		if err != nil {
			break
		}
		if updatedPort.DeviceID != "" {
			break
		}
	}
	if err != nil {
		attachErr := errors.New(fmt.Sprintf("Failed to Attach NIC (port Get after attach). err = %s", err.Error()))
		cblogger.Error(attachErr.Error())
		LoggingError(hiscallInfo, attachErr)
		return irs.NICInfo{}, attachErr
	}

	nicInfo, err := nicHandler.portToNICInfo(updatedPort)
	if err != nil {
		attachErr := errors.New(fmt.Sprintf("Failed to Attach NIC. err = %s", err.Error()))
		cblogger.Error(attachErr.Error())
		LoggingError(hiscallInfo, attachErr)
		return irs.NICInfo{}, attachErr
	}

	LoggingInfo(hiscallInfo, start)
	return nicInfo, nil
}

// DetachNIC detaches the given NIC (port) from its currently attached VM.
func (nicHandler *OpenStackNICHandler) DetachNIC(nicIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(nicHandler.Region.Region, call.NIC, nicIID.NameId, "DetachNIC()")
	start := call.Start()

	port, err := nicHandler.getRawPort(nicIID)
	if err != nil {
		detachErr := errors.New(fmt.Sprintf("Failed to Detach NIC. err = %s", err.Error()))
		cblogger.Error(detachErr.Error())
		LoggingError(hiscallInfo, detachErr)
		return false, detachErr
	}

	if port.DeviceID == "" {
		detachErr := errors.New("Failed to Detach NIC. err = port is not attached to any VM")
		cblogger.Error(detachErr.Error())
		LoggingError(hiscallInfo, detachErr)
		return false, detachErr
	}

	serverID := port.DeviceID
	portID := port.ID

	err = attachinterfaces.Delete(context.TODO(), nicHandler.ComputeClient, serverID, portID).ExtractErr()
	if err != nil {
		detachErr := errors.New(fmt.Sprintf("Failed to Detach NIC. err = %s", err.Error()))
		cblogger.Error(detachErr.Error())
		LoggingError(hiscallInfo, detachErr)
		return false, detachErr
	}

	// Poll until port DeviceID is cleared
	for i := 0; i < 20; i++ {
		time.Sleep(2 * time.Second)
		p, pErr := ports.Get(context.TODO(), nicHandler.NetworkClient, portID).Extract()
		if pErr != nil {
			break
		}
		if p.DeviceID == "" {
			break
		}
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

// ---- internal helpers ----

// getRawPort fetches the raw gophercloud Port struct by IID.
// Uses SystemId for a direct Get; falls back to listing and matching by name.
func (nicHandler *OpenStackNICHandler) getRawPort(nicIID irs.IID) (*ports.Port, error) {
	if nicIID.SystemId != "" {
		return ports.Get(context.TODO(), nicHandler.NetworkClient, nicIID.SystemId).Extract()
	}
	// Search by name
	var found *ports.Port
	err := ports.List(nicHandler.NetworkClient, ports.ListOpts{Name: nicIID.NameId}).EachPage(context.TODO(), func(ctx context.Context, page pagination.Page) (bool, error) {
		portList, err := ports.ExtractPorts(page)
		if err != nil {
			return false, err
		}
		for _, p := range portList {
			if p.Name == nicIID.NameId {
				pCopy := p
				found = &pCopy
				return false, nil // stop paging
			}
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, fmt.Errorf("NIC not found with name: %s", nicIID.NameId)
	}
	return found, nil
}

// portToNICInfo converts a gophercloud Port to an irs.NICInfo.
func (nicHandler *OpenStackNICHandler) portToNICInfo(port *ports.Port) (irs.NICInfo, error) {
	nicInfo := irs.NICInfo{
		IId: irs.IID{
			NameId:   port.Name,
			SystemId: port.ID,
		},
		VpcIID: irs.IID{
			NameId:   "",
			SystemId: port.NetworkID,
		},
		MACAddress:  port.MACAddress,
		CreatedTime: port.CreatedAt,
	}

	// Resolve VPC name
	if net, err := GetNetworkByID(nicHandler.NetworkClient, port.NetworkID); err == nil {
		nicInfo.VpcIID.NameId = net.Name
	}

	// SubnetIID and PrivateIP from the first FixedIP entry
	if len(port.FixedIPs) > 0 {
		subnetID := port.FixedIPs[0].SubnetID
		nicInfo.SubnetIID = irs.IID{
			NameId:   "",
			SystemId: subnetID,
		}
		nicInfo.PrivateIP = port.FixedIPs[0].IPAddress
		if sub, err := GetSubnetByID(nicHandler.NetworkClient, subnetID); err == nil {
			nicInfo.SubnetIID.NameId = sub.Name
		}
	}

	// All private IPs + index-aligned public IPs from floating IPs on this port
	fipMap := map[string]string{} // fixedIP -> floatingIP
	fipPages, fipErr := layer3floatingips.List(nicHandler.NetworkClient, layer3floatingips.ListOpts{PortID: port.ID}).AllPages(context.TODO())
	if fipErr == nil {
		if fips, fipErr2 := layer3floatingips.ExtractFloatingIPs(fipPages); fipErr2 == nil {
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
	if len(fipMap) > 0 {
		for _, pip := range nicInfo.PublicIPs {
			if pip != "" {
				nicInfo.PublicIP = pip
				break
			}
		}
	}

	// SecurityGroups
	for _, sgID := range port.SecurityGroups {
		nicInfo.SecurityGroupIIDs = append(nicInfo.SecurityGroupIIDs, irs.IID{SystemId: sgID})
	}

	// Status: if DeviceOwner starts with "compute:" the port is attached to a VM
	if strings.HasPrefix(port.DeviceOwner, "compute:") {
		nicInfo.Status = irs.NICAttached
		nicInfo.OwnerVM = irs.IID{SystemId: port.DeviceID}
		// Resolve VM name
		if vmName, err := nicHandler.getServerNameByID(port.DeviceID); err == nil {
			nicInfo.OwnerVM.NameId = vmName
		} else {
			nicInfo.OwnerVM.NameId = port.DeviceID
		}
	} else {
		nicInfo.Status = irs.NICAvailable
	}

	// KeyValueList
	nicInfo.KeyValueList = []irs.KeyValue{
		{Key: "DeviceOwner", Value: port.DeviceOwner},
		{Key: "AdminStateUp", Value: fmt.Sprintf("%v", port.AdminStateUp)},
		{Key: "PortStatus", Value: port.Status},
		{Key: "TenantID", Value: port.TenantID},
	}

	return nicInfo, nil
}

// getServerNameByID returns the server name for a given server UUID.
func (nicHandler *OpenStackNICHandler) getServerNameByID(serverID string) (string, error) {
	var serverName string
	err := servers.List(nicHandler.ComputeClient, servers.ListOpts{}).EachPage(context.TODO(), func(ctx context.Context, page pagination.Page) (bool, error) {
		serverList, err := servers.ExtractServers(page)
		if err != nil {
			return false, err
		}
		for _, s := range serverList {
			if s.ID == serverID {
				serverName = s.Name
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to search for server by ID %s: %s", serverID, err.Error())
	}
	if serverName == "" {
		return "", fmt.Errorf("server not found with ID: %s", serverID)
	}
	return serverName, nil
}

// getServerIDByName finds the server UUID for a given server name.
func (nicHandler *OpenStackNICHandler) getServerIDByName(serverName string) (string, error) {
	var serverID string
	err := servers.List(nicHandler.ComputeClient, servers.ListOpts{Name: serverName}).EachPage(context.TODO(), func(ctx context.Context, page pagination.Page) (bool, error) {
		serverList, err := servers.ExtractServers(page)
		if err != nil {
			return false, err
		}
		for _, s := range serverList {
			if s.Name == serverName {
				serverID = s.ID
				return false, nil
			}
		}
		return true, nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to search for server by name %s: %s", serverName, err.Error())
	}
	if serverID == "" {
		return "", fmt.Errorf("server not found with name: %s", serverName)
	}
	return serverID, nil
}

// AddPrivateIP adds a secondary private IP to a NIC (Neutron port) using ports.Update.
func (h *OpenStackNICHandler) AddPrivateIP(nicIID irs.IID, privateIP string) (irs.NICInfo, error) {
	port, err := h.getRawPort(nicIID)
	if err != nil {
		return irs.NICInfo{}, fmt.Errorf("OpenStackNICHandler.AddPrivateIP: failed to get port: %w", err)
	}

	// Build updated FixedIPs list: keep existing + add new
	// SubnetID is required by Neutron — use the same subnet as the primary IP
	if len(port.FixedIPs) == 0 {
		return irs.NICInfo{}, fmt.Errorf("OpenStackNICHandler.AddPrivateIP: port has no existing FixedIPs to derive SubnetID")
	}
	subnetID := port.FixedIPs[0].SubnetID
	newFixedIPs := make([]ports.IP, 0, len(port.FixedIPs)+1)
	for _, fip := range port.FixedIPs {
		newFixedIPs = append(newFixedIPs, ports.IP{SubnetID: fip.SubnetID, IPAddress: fip.IPAddress})
	}
	if privateIP != "" {
		newFixedIPs = append(newFixedIPs, ports.IP{SubnetID: subnetID, IPAddress: privateIP})
	} else {
		newFixedIPs = append(newFixedIPs, ports.IP{SubnetID: subnetID})
	}

	updated, err := ports.Update(context.TODO(), h.NetworkClient, port.ID, ports.UpdateOpts{
		FixedIPs: newFixedIPs,
	}).Extract()
	if err != nil {
		return irs.NICInfo{}, fmt.Errorf("OpenStackNICHandler.AddPrivateIP: failed to update port: %w", err)
	}

	return h.portToNICInfo(updated)
}

// RemovePrivateIP removes a secondary private IP from a NIC (Neutron port) using ports.Update.
func (h *OpenStackNICHandler) RemovePrivateIP(nicIID irs.IID, privateIP string) (bool, error) {
	port, err := h.getRawPort(nicIID)
	if err != nil {
		return false, fmt.Errorf("OpenStackNICHandler.RemovePrivateIP: failed to get port: %w", err)
	}

	// Build updated FixedIPs list: exclude the specified IP
	newFixedIPs := make([]ports.IP, 0, len(port.FixedIPs))
	found := false
	for _, fip := range port.FixedIPs {
		if fip.IPAddress == privateIP {
			found = true
			continue
		}
		newFixedIPs = append(newFixedIPs, ports.IP{SubnetID: fip.SubnetID, IPAddress: fip.IPAddress})
	}
	if !found {
		return false, fmt.Errorf("OpenStackNICHandler.RemovePrivateIP: IP %s not found on port %s", privateIP, port.ID)
	}

	// If a floating IP is associated with this private IP, disassociate it first
	if err := h.disassociateFloatingIPByFixedIP(port.ID, privateIP); err != nil {
		return false, fmt.Errorf("OpenStackNICHandler.RemovePrivateIP: failed to disassociate floating IP: %w", err)
	}

	_, err = ports.Update(context.TODO(), h.NetworkClient, port.ID, ports.UpdateOpts{
		FixedIPs: newFixedIPs,
	}).Extract()
	if err != nil {
		return false, fmt.Errorf("OpenStackNICHandler.RemovePrivateIP: failed to update port: %w", err)
	}
	return true, nil
}

// disassociateFloatingIPByFixedIP disassociates any floating IP bound to the given port+fixedIP.
func (h *OpenStackNICHandler) disassociateFloatingIPByFixedIP(portID, fixedIP string) error {
	allPages, err := layer3floatingips.List(h.NetworkClient, layer3floatingips.ListOpts{PortID: portID}).AllPages(context.TODO())
	if err != nil {
		return err
	}
	fips, err := layer3floatingips.ExtractFloatingIPs(allPages)
	if err != nil {
		return err
	}
	emptyPort := ""
	for _, fip := range fips {
		if fip.FixedIP == fixedIP || fixedIP == "" {
			_, err = layer3floatingips.Update(context.TODO(), h.NetworkClient, fip.ID, layer3floatingips.UpdateOpts{PortID: &emptyPort}).Extract()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// GetNICOSConfigScript returns a bash script that must be run inside the OpenStack VM OS
// after a secondary port (NIC) is attached. OpenStack does not auto-configure secondary
// ports via cloud-init in most images; the guest OS must bring up the interface and
// configure policy-based routing for the new port's IP.
func (nicHandler *OpenStackNICHandler) GetNICOSConfigScript(nicIID irs.IID) (string, error) {
	nicInfo, err := nicHandler.GetNIC(nicIID)
	if err != nil {
		return "", fmt.Errorf("GetNICOSConfigScript: failed to get NIC info: %w", err)
	}

	privateIPs := nicInfo.PrivateIPs
	if len(privateIPs) == 0 && nicInfo.PrivateIP != "" {
		privateIPs = []string{nicInfo.PrivateIP}
	}
	if len(privateIPs) == 0 {
		return "", fmt.Errorf("GetNICOSConfigScript: NIC has no private IPs")
	}

	mac := strings.ToLower(nicInfo.MACAddress)
	ip0 := privateIPs[0]
	lastDot := strings.LastIndex(ip0, ".")
	gateway := ip0[:lastDot] + ".1"
	cidr := ip0[:lastDot] + ".0/24"

	script := "# 1. Identify interface name by MAC\n" +
		"IFACE=$(ip link | grep -B1 \"" + mac + "\" | head -1 | awk -F': ' '{print $2}')\n" +
		"echo \"Target interface: $IFACE\"\n"

	script += "\n# 2. Bring up secondary NIC (do NOT use dhclient — it adds a default route\n" +
		"#    to the main table and breaks primary NIC SSH)\n" +
		"sudo ip link set $IFACE up\n"
	for _, ip := range privateIPs {
		script += "sudo ip addr add " + ip + "/24 dev $IFACE 2>/dev/null || true\n"
	}

	script += "\n# 3. Configure Policy-based Routing (PBR)\n"
	script += "sudo ip route flush table 101 2>/dev/null || true\n"
	for _, ip := range privateIPs {
		script += "sudo ip rule del from " + ip + " lookup 101 2>/dev/null || true\n"
	}
	script += "sudo ip route add " + cidr + " dev $IFACE src " + ip0 + " table 101\n"
	script += "sudo ip route add default via " + gateway + " dev $IFACE table 101\n"
	for _, ip := range privateIPs {
		script += "sudo ip rule add from " + ip + " lookup 101\n"
	}

	return script, nil
}
