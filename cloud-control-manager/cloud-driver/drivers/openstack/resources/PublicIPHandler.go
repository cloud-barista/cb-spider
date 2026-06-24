// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// OpenStack Floating IP Handler
//
// by CB-Spider Team, 2025.06.

package resources

import (
	"context"
	"fmt"
	"time"

	"github.com/gophercloud/gophercloud/v2"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/external"
	layer3floatingips "github.com/gophercloud/gophercloud/v2/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/v2/openstack/networking/v2/ports"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type OpenStackPublicIPHandler struct {
	Region        idrv.RegionInfo
	NetworkClient *gophercloud.ServiceClient
}

func (h *OpenStackPublicIPHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(h.NetworkClient.IdentityEndpoint, call.PUBLICIP, "ListIID", "floatingips.List()")
	start := call.Start()

	pager, err := layer3floatingips.List(h.NetworkClient, layer3floatingips.ListOpts{}).AllPages(context.TODO())
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	fips, err := layer3floatingips.ExtractFloatingIPs(pager)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	var iidList []*irs.IID
	for _, fip := range fips {
		iidList = append(iidList, &irs.IID{NameId: osPublicIPNameId(&fip), SystemId: fip.ID})
	}
	return iidList, nil
}

func (h *OpenStackPublicIPHandler) CreatePublicIP(reqInfo irs.PublicIPInfo) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.NetworkClient.IdentityEndpoint, call.PUBLICIP, reqInfo.IId.NameId, "floatingips.Create()")
	start := call.Start()

	extNetID, err := h.findExternalNetworkID()
	if err != nil {
		return irs.PublicIPInfo{}, fmt.Errorf("failed to find external network: %w", err)
	}

	createOpts := layer3floatingips.CreateOpts{
		FloatingNetworkID: extNetID,
		Description:       reqInfo.IId.NameId,
	}

	fip, err := layer3floatingips.Create(context.TODO(), h.NetworkClient, createOpts).Extract()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	info := extractOSPublicIPInfo(fip)
	info.IId.NameId = reqInfo.IId.NameId
	return info, nil
}

func (h *OpenStackPublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.NetworkClient.IdentityEndpoint, call.PUBLICIP, "All", "floatingips.List()")
	start := call.Start()

	pager, err := layer3floatingips.List(h.NetworkClient, layer3floatingips.ListOpts{}).AllPages(context.TODO())
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	fips, err := layer3floatingips.ExtractFloatingIPs(pager)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	var infoList []*irs.PublicIPInfo
	for _, fip := range fips {
		info := extractOSPublicIPInfo(&fip)
		h.resolvePublicIPOwner(&info)
		infoList = append(infoList, &info)
	}
	if infoList == nil {
		infoList = []*irs.PublicIPInfo{}
	}
	return infoList, nil
}

func (h *OpenStackPublicIPHandler) GetPublicIP(publicIPIID irs.IID) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.NetworkClient.IdentityEndpoint, call.PUBLICIP, publicIPIID.NameId, "floatingips.Get()")
	start := call.Start()

	var fip *layer3floatingips.FloatingIP
	var err error

	if publicIPIID.SystemId != "" {
		fip, err = layer3floatingips.Get(context.TODO(), h.NetworkClient, publicIPIID.SystemId).Extract()
	} else {
		pager, listErr := layer3floatingips.List(h.NetworkClient, layer3floatingips.ListOpts{}).AllPages(context.TODO())
		if listErr != nil {
			err = listErr
		} else {
			fips, extractErr := layer3floatingips.ExtractFloatingIPs(pager)
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

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	info := extractOSPublicIPInfo(fip)
	if publicIPIID.NameId != "" {
		info.IId.NameId = publicIPIID.NameId
	}
	h.resolvePublicIPOwner(&info)
	return info, nil
}

func (h *OpenStackPublicIPHandler) DeletePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.NetworkClient.IdentityEndpoint, call.PUBLICIP, publicIPIID.NameId, "floatingips.Delete()")
	start := call.Start()

	systemId := publicIPIID.SystemId
	if systemId == "" {
		info, err := h.GetPublicIP(publicIPIID)
		if err != nil {
			return false, err
		}
		systemId = info.IId.SystemId
	}

	err := layer3floatingips.Delete(context.TODO(), h.NetworkClient, systemId).ExtractErr()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

// NetworkWithExternalExt embeds the external extension fields.
type OSNetworkWithExt struct {
	networks.Network
	external.NetworkExternalExt
}

func (h *OpenStackPublicIPHandler) findExternalNetworkID() (string, error) {
	listOpts := external.ListOptsExt{
		ListOptsBuilder: networks.ListOpts{},
	}
	pager, err := networks.List(h.NetworkClient, listOpts).AllPages(context.TODO())
	if err != nil {
		return "", err
	}

	var netList []OSNetworkWithExt
	if err := networks.ExtractNetworksInto(pager, &netList); err != nil {
		return "", err
	}

	for _, net := range netList {
		if net.External {
			return net.ID, nil
		}
	}
	return "", fmt.Errorf("no external network found")
}

func extractOSPublicIPInfo(fip *layer3floatingips.FloatingIP) irs.PublicIPInfo {
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
		// OwnedNIC: SystemId=portID, NameId resolved by caller if possible
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

func osPublicIPNameId(fip *layer3floatingips.FloatingIP) string {
	if fip.Description != "" {
		return fip.Description
	}
	return fip.ID
}

// AssociatePublicIP links a Floating IP to a VM port.
func (h *OpenStackPublicIPHandler) AssociatePublicIP(publicIPIID irs.IID, vmIID irs.IID, nicIID irs.IID, privateIP string) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.NetworkClient.IdentityEndpoint, call.PUBLICIP, publicIPIID.NameId, "floatingips.Update()")
	start := call.Start()

	info, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return irs.PublicIPInfo{}, err
	}

	// Resolve port ID from NIC IID
	var portID string
	if nicIID.SystemId != "" {
		portID = nicIID.SystemId
	} else if nicIID.NameId != "" {
		// nicIID.NameId is a port name — resolve to port UUID
		pager, pErr := ports.List(h.NetworkClient, ports.ListOpts{Name: nicIID.NameId}).AllPages(context.TODO())
		if pErr == nil {
			portList, pErr2 := ports.ExtractPorts(pager)
			if pErr2 == nil {
				for _, p := range portList {
					if p.Name == nicIID.NameId {
						portID = p.ID
						break
					}
				}
			}
		}
		if portID == "" {
			return irs.PublicIPInfo{}, fmt.Errorf("AssociatePublicIP: NIC not found with name: %s", nicIID.NameId)
		}
	} else {
		portID, err = h.getVMPortID(vmIID)
		if err != nil {
			return irs.PublicIPInfo{}, err
		}
	}

	updateOpts := layer3floatingips.UpdateOpts{PortID: &portID}
	if privateIP != "" {
		updateOpts.FixedIP = privateIP
	}
	updated, err := layer3floatingips.Update(context.TODO(), h.NetworkClient, info.IId.SystemId, updateOpts).Extract()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	result := extractOSPublicIPInfo(updated)
	result.IId.NameId = publicIPIID.NameId
	h.resolvePublicIPOwner(&result)
	return result, nil
}

// DisassociatePublicIP clears the port association from a Floating IP.
func (h *OpenStackPublicIPHandler) DisassociatePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.NetworkClient.IdentityEndpoint, call.PUBLICIP, publicIPIID.NameId, "floatingips.Update()")
	start := call.Start()

	info, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return false, err
	}
	if info.Status != irs.PublicIPAssociated {
		return false, fmt.Errorf("PublicIP %s is not associated", publicIPIID.NameId)
	}

	emptyPort := ""
	_, err = layer3floatingips.Update(context.TODO(), h.NetworkClient, info.IId.SystemId, layer3floatingips.UpdateOpts{PortID: &emptyPort}).Extract()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

// resolvePublicIPOwner fills OwnedNIC.NameId and OwnedVM from a port lookup.
func (h *OpenStackPublicIPHandler) resolvePublicIPOwner(info *irs.PublicIPInfo) {
	if info.OwnedNIC.SystemId == "" {
		return
	}
	portID := info.OwnedNIC.SystemId
	port, err := ports.Get(context.TODO(), h.NetworkClient, portID).Extract()
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

func (h *OpenStackPublicIPHandler) getVMPortID(vmIID irs.IID) (string, error) {
	deviceId := vmIID.SystemId
	if deviceId == "" {
		deviceId = vmIID.NameId
	}
	pager, err := ports.List(h.NetworkClient, ports.ListOpts{DeviceID: deviceId}).AllPages(context.TODO())
	if err != nil {
		return "", fmt.Errorf("failed to list ports for VM %s: %w", deviceId, err)
	}
	portList, err := ports.ExtractPorts(pager)
	if err != nil {
		return "", fmt.Errorf("failed to extract ports: %w", err)
	}
	if len(portList) == 0 {
		return "", fmt.Errorf("no port found for VM %s", deviceId)
	}
	return portList[0].ID, nil
}
