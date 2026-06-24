// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NHN Cloud Floating IP Handler (OpenStack-compatible)
//
// by CB-Spider Team, 2025.06.

package resources

import (
	"fmt"
	"time"

	calllog "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	computefips "github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/extensions/floatingips"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/networks"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/ports"
)

type NhnCloudPublicIPHandler struct {
	RegionInfo    idrv.RegionInfo
	NetworkClient *nhnsdk.ServiceClient
	VMClient      *nhnsdk.ServiceClient
}

func (h *NhnCloudPublicIPHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, calllog.PUBLICIP, "ListIID", "floatingips.List()")
	start := calllog.Start()

	allPages, err := floatingips.List(h.NetworkClient, floatingips.ListOpts{}).AllPages()
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	fips, err := floatingips.ExtractFloatingIPs(allPages)
	hiscallInfo.ElapsedTime = calllog.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	var iidList []*irs.IID
	for _, fip := range fips {
		nameId := nhnPublicIPNameId(&fip)
		iidList = append(iidList, &irs.IID{NameId: nameId, SystemId: fip.ID})
	}
	return iidList, nil
}

func (h *NhnCloudPublicIPHandler) CreatePublicIP(reqInfo irs.PublicIPInfo) (irs.PublicIPInfo, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, calllog.PUBLICIP, reqInfo.IId.NameId, "floatingips.Create()")
	start := calllog.Start()

	extNetID, err := h.findExternalNetworkID()
	if err != nil {
		return irs.PublicIPInfo{}, fmt.Errorf("failed to find external network: %w", err)
	}

	createOpts := floatingips.CreateOpts{
		FloatingNetworkID: extNetID,
		Description:       reqInfo.IId.NameId,
	}

	fip, err := floatingips.Create(h.NetworkClient, createOpts).Extract()
	hiscallInfo.ElapsedTime = calllog.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	info := extractNhnPublicIPInfo(fip)
	info.IId.NameId = reqInfo.IId.NameId
	return info, nil
}

func (h *NhnCloudPublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, calllog.PUBLICIP, "All", "floatingips.List()")
	start := calllog.Start()

	allPages, err := floatingips.List(h.NetworkClient, floatingips.ListOpts{}).AllPages()
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	fips, err := floatingips.ExtractFloatingIPs(allPages)
	hiscallInfo.ElapsedTime = calllog.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	var infoList []*irs.PublicIPInfo
	for _, fip := range fips {
		info := extractNhnPublicIPInfo(&fip)
		h.resolveNhnPublicIPOwner(&info)
		infoList = append(infoList, &info)
	}
	if infoList == nil {
		infoList = []*irs.PublicIPInfo{}
	}
	return infoList, nil
}

func (h *NhnCloudPublicIPHandler) GetPublicIP(publicIPIID irs.IID) (irs.PublicIPInfo, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, calllog.PUBLICIP, publicIPIID.NameId, "floatingips.Get()")
	start := calllog.Start()

	var fip *floatingips.FloatingIP
	var err error

	if publicIPIID.SystemId != "" {
		fip, err = floatingips.Get(h.NetworkClient, publicIPIID.SystemId).Extract()
	} else {
		allPages, listErr := floatingips.List(h.NetworkClient, floatingips.ListOpts{}).AllPages()
		if listErr != nil {
			err = listErr
		} else {
			fips, extractErr := floatingips.ExtractFloatingIPs(allPages)
			if extractErr != nil {
				err = extractErr
			} else {
				for i, f := range fips {
					if f.Description == publicIPIID.NameId || f.FloatingIP == publicIPIID.NameId {
						fip = &fips[i]
						break
					}
				}
				if fip == nil {
					err = fmt.Errorf("PublicIP not found: %s", publicIPIID.NameId)
				}
			}
		}
	}

	hiscallInfo.ElapsedTime = calllog.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	info := extractNhnPublicIPInfo(fip)
	if publicIPIID.NameId != "" {
		info.IId.NameId = publicIPIID.NameId
	}
	h.resolveNhnPublicIPOwner(&info)
	return info, nil
}

func (h *NhnCloudPublicIPHandler) DeletePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, calllog.PUBLICIP, publicIPIID.NameId, "floatingips.Delete()")
	start := calllog.Start()

	systemId := publicIPIID.SystemId
	if systemId == "" {
		info, err := h.GetPublicIP(publicIPIID)
		if err != nil {
			return false, err
		}
		systemId = info.IId.SystemId
	}

	err := floatingips.Delete(h.NetworkClient, systemId).ExtractErr()
	hiscallInfo.ElapsedTime = calllog.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func (h *NhnCloudPublicIPHandler) findExternalNetworkID() (string, error) {
	allPages, err := networks.List(h.NetworkClient, networks.ListOpts{}).AllPages()
	if err != nil {
		return "", err
	}
	netList, err := networks.ExtractNetworks(allPages)
	if err != nil {
		return "", err
	}
	for _, net := range netList {
		if net.RouterExternal {
			return net.ID, nil
		}
	}
	return "", fmt.Errorf("no external network found")
}

func extractNhnPublicIPInfo(fip *floatingips.FloatingIP) irs.PublicIPInfo {
	status := irs.PublicIPAvailable
	if fip.PortID != "" {
		status = irs.PublicIPAssociated
	}

	nameId := fip.Description
	if nameId == "" {
		nameId = fip.ID
	}

	info := irs.PublicIPInfo{
		IId:             irs.IID{NameId: nameId, SystemId: fip.ID},
		PublicIPAddress: fip.FloatingIP,
		Status:          status,
		CreatedTime:     time.Time{},
	}

	if fip.PortID != "" {
		// OwnedNIC: SystemId=portID, NameId resolved by resolveNhnPublicIPOwner
		info.OwnedNIC = irs.IID{NameId: fip.PortID, SystemId: fip.PortID}
		info.OwnedPrivateIP = fip.FixedIP
	}

	info.KeyValueList = []irs.KeyValue{
		{Key: "FloatingNetworkID", Value: fip.FloatingNetworkID},
		{Key: "FixedIP", Value: fip.FixedIP},
		{Key: "PortID", Value: fip.PortID},
		{Key: "TenantID", Value: fip.TenantID},
		{Key: "Status", Value: fip.Status},
	}

	return info
}

// resolveNhnPublicIPOwner fills OwnedNIC.NameId and OwnedVM from a port lookup.
func (h *NhnCloudPublicIPHandler) resolveNhnPublicIPOwner(info *irs.PublicIPInfo) {
	if info.OwnedNIC.SystemId == "" {
		return
	}
	port, err := ports.Get(h.NetworkClient, info.OwnedNIC.SystemId).Extract()
	if err != nil {
		return
	}
	if port.Name != "" {
		info.OwnedNIC.NameId = port.Name
	}
	if port.DeviceID != "" {
		info.OwnedVM = irs.IID{SystemId: port.DeviceID, NameId: port.DeviceID}
	}
}

func nhnPublicIPNameId(fip *floatingips.FloatingIP) string {
	if fip.Description != "" {
		return fip.Description
	}
	return fip.ID
}

// AssociatePublicIP links a Floating IP to a VM.
// When nicIID is provided, uses Neutron port association.
// When only vmIID is provided (nicIID empty), uses Nova compute addFloatingIp action.
func (h *NhnCloudPublicIPHandler) AssociatePublicIP(publicIPIID irs.IID, vmIID irs.IID, nicIID irs.IID, privateIP string) (irs.PublicIPInfo, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, calllog.PUBLICIP, publicIPIID.NameId, "floatingips.AssociatePublicIP()")
	start := calllog.Start()

	info, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return irs.PublicIPInfo{}, err
	}

	nicIsEmpty := nicIID.SystemId == "" && nicIID.NameId == ""

	if nicIsEmpty && vmIID.SystemId != "" || nicIsEmpty && vmIID.NameId != "" {
		// Use Nova compute addFloatingIp action — works reliably on NHN Cloud
		serverID, sErr := h.resolveServerID(vmIID)
		if sErr != nil {
			return irs.PublicIPInfo{}, sErr
		}
		assocOpts := computefips.AssociateOpts{FloatingIP: info.PublicIPAddress}
		if privateIP != "" {
			assocOpts.FixedIP = privateIP
		}
		assocErr := computefips.AssociateInstance(h.VMClient, serverID, assocOpts).ExtractErr()
		hiscallInfo.ElapsedTime = calllog.Elapsed(start)
		if assocErr != nil {
			cblogger.Error(assocErr)
			LoggingError(hiscallInfo, assocErr)
			return irs.PublicIPInfo{}, assocErr
		}
		LoggingInfo(hiscallInfo, start)
		// Re-fetch to get updated port binding
		return h.GetPublicIP(irs.IID{NameId: publicIPIID.NameId, SystemId: info.IId.SystemId})
	}

	// NIC specified — use Neutron port association
	var portID string
	if nicIID.SystemId != "" {
		portID = nicIID.SystemId
	} else {
		// nicIID.NameId is a port name — resolve to port UUID
		allPages, pErr := ports.List(h.NetworkClient, ports.ListOpts{}).AllPages()
		if pErr == nil {
			portList, pErr2 := ports.ExtractPorts(allPages)
			if pErr2 == nil {
				for _, p := range portList {
					if p.Name == nicIID.NameId || p.Description == nicIID.NameId {
						portID = p.ID
						break
					}
				}
			}
		}
		if portID == "" {
			return irs.PublicIPInfo{}, fmt.Errorf("AssociatePublicIP: NIC not found with name: %s", nicIID.NameId)
		}
	}

	updateOpts := floatingips.UpdateOpts{PortID: &portID}
	if privateIP != "" {
		updateOpts.FixedIP = privateIP
	}
	updated, err := floatingips.Update(h.NetworkClient, info.IId.SystemId, updateOpts).Extract()
	hiscallInfo.ElapsedTime = calllog.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	result := extractNhnPublicIPInfo(updated)
	result.IId.NameId = publicIPIID.NameId
	h.resolveNhnPublicIPOwner(&result)
	return result, nil
}

func (h *NhnCloudPublicIPHandler) resolveServerID(vmIID irs.IID) (string, error) {
	if vmIID.SystemId != "" {
		return vmIID.SystemId, nil
	}
	// Look up server by name via ports
	allPages, err := ports.List(h.NetworkClient, ports.ListOpts{}).AllPages()
	if err != nil {
		return "", fmt.Errorf("resolveServerID: failed to list ports: %w", err)
	}
	portList, err := ports.ExtractPorts(allPages)
	if err != nil {
		return "", fmt.Errorf("resolveServerID: failed to extract ports: %w", err)
	}
	for _, p := range portList {
		if (p.Name == vmIID.NameId || p.Description == vmIID.NameId) && p.DeviceID != "" {
			return p.DeviceID, nil
		}
	}
	// NameId might already be the server UUID
	return vmIID.NameId, nil
}

// DisassociatePublicIP clears the port from a Floating IP.
func (h *NhnCloudPublicIPHandler) DisassociatePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, calllog.PUBLICIP, publicIPIID.NameId, "floatingips.Update()")
	start := calllog.Start()

	info, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return false, err
	}
	if info.Status != irs.PublicIPAssociated {
		return false, fmt.Errorf("PublicIP %s is not associated", publicIPIID.NameId)
	}

	emptyPort := ""
	_, err = floatingips.Update(h.NetworkClient, info.IId.SystemId, floatingips.UpdateOpts{PortID: &emptyPort}).Extract()
	hiscallInfo.ElapsedTime = calllog.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func (h *NhnCloudPublicIPHandler) getVMPortID(vmIID irs.IID) (string, error) {
	deviceId := vmIID.SystemId
	if deviceId == "" {
		deviceId = vmIID.NameId
	}
	allPages, err := ports.List(h.NetworkClient, ports.ListOpts{DeviceID: deviceId}).AllPages()
	if err != nil {
		return "", fmt.Errorf("failed to list ports for VM %s: %w", deviceId, err)
	}
	portList, err := ports.ExtractPorts(allPages)
	if err != nil {
		return "", fmt.Errorf("failed to extract ports: %w", err)
	}
	if len(portList) == 0 {
		return "", fmt.Errorf("no port found for VM %s", deviceId)
	}
	return portList[0].ID, nil
}
