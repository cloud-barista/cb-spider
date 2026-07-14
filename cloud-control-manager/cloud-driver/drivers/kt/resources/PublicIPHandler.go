// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// KT Cloud VPC Public IP (Floating IP) Handler
//
// by CB-Spider Team, 2025.06.

package resources

import (
	"fmt"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	ktvpcsdk "github.com/cloud-barista/ktcloudvpc-sdk-go"
	ips "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/compute/v2/extensions/floatingips"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/ports"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/extensions/layer3/staticnat"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/pagination"
)

type KTVpcPublicIPHandler struct {
	RegionInfo    idrv.RegionInfo
	NetworkClient *ktvpcsdk.ServiceClient
	VMClient      *ktvpcsdk.ServiceClient
}

func (h *KTVpcPublicIPHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, "ListIID", "floatingips.List()")
	start := call.Start()

	listOpts := ips.ListOpts{
		Page: 1, 
		Size: 2000, // Max page size, to list all data in a single page
	}	
	pager := ips.List(h.NetworkClient, listOpts)

	var iidList []*irs.IID
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		fipList, err := ips.ExtractFloatingIPs(page)
		if err != nil {
			return false, err
		}
		for _, fip := range fipList {
			iidList = append(iidList, &irs.IID{NameId: fip.PublicIpID, SystemId: fip.PublicIpID})
		}
		return true, nil
	})
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		loggingError(hiscallInfo, err)
		return nil, err
	}
	loggingInfo(hiscallInfo, start)

	return iidList, nil
}

func (h *KTVpcPublicIPHandler) CreatePublicIP(reqInfo irs.PublicIPInfo) (irs.PublicIPInfo, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, reqInfo.IId.NameId, "floatingips.Create()")
	start := call.Start()

	createOpts := ips.CreateOpts{}
	result, err := ips.Create(h.NetworkClient, createOpts).ExtractCreate()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		loggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	loggingInfo(hiscallInfo, start)

	// KT create only returns PublicIpID; get full info via list
	info, getErr := h.GetPublicIP(irs.IID{NameId: reqInfo.IId.NameId, SystemId: result.Data.PublicIpID})
	if getErr != nil {
		// Return minimal info if get fails
		return irs.PublicIPInfo{
			IId:             irs.IID{NameId: reqInfo.IId.NameId, SystemId: result.Data.PublicIpID},
			PublicIPAddress: "NA",
			Status:          irs.PublicIPAvailable,
			CreatedTime:     time.Time{},
		}, nil
	}
	info.IId.NameId = reqInfo.IId.NameId
	return info, nil
}

func (h *KTVpcPublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, "All", "floatingips.List()")
	start := call.Start()

	listOpts := ips.ListOpts{
		Page: 1,
		Size: 2000, // Max page size, to list all data in a single page
	}
	pager := ips.List(h.NetworkClient, listOpts)

	var infoList []*irs.PublicIPInfo
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		fipList, err := ips.ExtractFloatingIPs(page)
		if err != nil {
			return false, err
		}
		for _, fip := range fipList {
			info := extractKTPublicIPInfo(&fip)
			infoList = append(infoList, &info)
		}
		return true, nil
	})
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		loggingError(hiscallInfo, err)
		return nil, err
	}
	loggingInfo(hiscallInfo, start)

	if infoList == nil {
		infoList = []*irs.PublicIPInfo{}
	}
	return infoList, nil
}

func (h *KTVpcPublicIPHandler) GetPublicIP(publicIPIID irs.IID) (irs.PublicIPInfo, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, publicIPIID.NameId, "floatingips.Get()")
	start := call.Start()

	var foundFip *ips.FloatingIP
	var err error

	if publicIPIID.SystemId != "" {
		result := ips.Get(h.NetworkClient, publicIPIID.SystemId)
		if result.Err != nil {
			err = result.Err
		} else {
			fip, extractErr := result.ExtractFloatingIP()
			if extractErr != nil {
				err = extractErr
			} else {
				foundFip = fip
			}
		}
	} else {
		// Search by listing
		listOpts := ips.ListOpts{
			Page: 1,
			Size: 2000, // Max page size, to list all data in a single page
		}
		pager := ips.List(h.NetworkClient, listOpts)
		pager.EachPage(func(page pagination.Page) (bool, error) {
			fipList, listErr := ips.ExtractFloatingIPs(page)
			if listErr != nil {
				return false, listErr
			}
			for i, fip := range fipList {
				if fip.PublicIpID == publicIPIID.NameId || fip.PublicIP == publicIPIID.NameId {
					foundFip = &fipList[i]
					return false, nil
				}
			}
			return true, nil
		})
		if foundFip == nil {
			err = fmt.Errorf("PublicIP not found: %s", publicIPIID.NameId)
		}
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		loggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	loggingInfo(hiscallInfo, start)

	info := extractKTPublicIPInfo(foundFip)
	if publicIPIID.NameId != "" {
		info.IId.NameId = publicIPIID.NameId
	}
	return info, nil
}

func (h *KTVpcPublicIPHandler) DeletePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, publicIPIID.NameId, "floatingips.Delete()")
	start := call.Start()

	systemId := publicIPIID.SystemId
	if systemId == "" {
		info, err := h.GetPublicIP(publicIPIID)
		if err != nil {
			return false, err
		}
		systemId = info.IId.SystemId
	}

	result := ips.Delete(h.NetworkClient, systemId)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if result.Err != nil {
		cblogger.Error(result.Err)
		loggingError(hiscallInfo, result.Err)
		return false, result.Err
	}
	loggingInfo(hiscallInfo, start)

	return true, nil
}

func extractKTPublicIPInfo(fip *ips.FloatingIP) irs.PublicIPInfo {
	status := irs.PublicIPAvailable
	if len(fip.StaticNats) > 0 {
		status = irs.PublicIPAssociated
	}

	info := irs.PublicIPInfo{
		IId:             irs.IID{NameId: fip.PublicIpID, SystemId: fip.PublicIpID},
		PublicIPAddress: fip.PublicIP,
		Status:          status,
		CreatedTime:     time.Time{},
	}

	info.KeyValueList = []irs.KeyValue{
		{Key: "PublicIpID", Value: fip.PublicIpID},
		{Key: "VpcID", Value: fip.VpcID},
		{Key: "ZoneID", Value: fip.ZoneID},
		{Key: "Type", Value: fip.Type},
	}

	return info
}

// resolveKTPublicIPOwner fills OwnedPrivateIP from StaticNAT list for the given PublicIpID.
func (h *KTVpcPublicIPHandler) resolveKTPublicIPOwner(info *irs.PublicIPInfo) {
	if info.Status != irs.PublicIPAssociated {
		return
	}
	allPages, err := staticnat.List(h.NetworkClient, staticnat.ListOpts{}).AllPages()
	if err != nil {
		return
	}
	nats, err := staticnat.ExtractStaticNats(allPages)
	if err != nil {
		return
	}
	for _, nat := range nats {
		if nat.PublicIpID == info.IId.SystemId {
			info.OwnedPrivateIP = nat.MappedIP
			break
		}
	}
}

// AssociatePublicIP associates a Public IP to a VM private IP via KT Cloud StaticNAT.
func (h *KTVpcPublicIPHandler) AssociatePublicIP(publicIPIID irs.IID, vmIID irs.IID, nicIID irs.IID, privateIP string) (irs.PublicIPInfo, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, publicIPIID.NameId, "staticnat.Create()")
	start := call.Start()

	info, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return irs.PublicIPInfo{}, err
	}

	if privateIP == "" {
		// Auto-resolve private IP from VM if vmIID is provided
		if vmIID.SystemId != "" || vmIID.NameId != "" {
			resolved, resolveErr := h.resolveVMPrivateIP(vmIID)
			if resolveErr != nil {
				return irs.PublicIPInfo{}, fmt.Errorf("AssociatePublicIP: privateIP not provided and failed to resolve from VM [%s]: %w", vmIID.NameId, resolveErr)
			}
			privateIP = resolved
		} else {
			return irs.PublicIPInfo{}, fmt.Errorf("AssociatePublicIP: privateIP is required for KT Cloud StaticNAT (or provide vmIID to auto-resolve)")
		}
	}

	createOpts := staticnat.CreateOpts{
		PublicIpID:   info.IId.SystemId,
		PrivateIpAddr: privateIP,
	}
	_, err = staticnat.Create(h.NetworkClient, createOpts).ExtractCreate()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		loggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	loggingInfo(hiscallInfo, start)

	result, getErr := h.GetPublicIP(irs.IID{NameId: publicIPIID.NameId, SystemId: info.IId.SystemId})
	if getErr != nil {
		return irs.PublicIPInfo{}, getErr
	}
	h.resolveKTPublicIPOwner(&result)
	return result, nil
}

// DisassociatePublicIP removes the StaticNAT binding for a Public IP.
func (h *KTVpcPublicIPHandler) DisassociatePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := getCallLogScheme(h.RegionInfo.Zone, call.PUBLICIP, publicIPIID.NameId, "staticnat.Delete()")
	start := call.Start()

	info, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return false, err
	}
	if info.Status != irs.PublicIPAssociated {
		return false, fmt.Errorf("DisassociatePublicIP: PublicIP %s is not associated", publicIPIID.NameId)
	}

	// Find the StaticNAT ID for this PublicIP
	allPages, err := staticnat.List(h.NetworkClient, staticnat.ListOpts{}).AllPages()
	if err != nil {
		return false, fmt.Errorf("DisassociatePublicIP: failed to list StaticNATs: %w", err)
	}
	nats, err := staticnat.ExtractStaticNats(allPages)
	if err != nil {
		return false, fmt.Errorf("DisassociatePublicIP: failed to extract StaticNATs: %w", err)
	}
	staticNatID := ""
	for _, nat := range nats {
		if nat.PublicIpID == info.IId.SystemId {
			staticNatID = nat.StaticNatID
			break
		}
	}
	if staticNatID == "" {
		return false, fmt.Errorf("DisassociatePublicIP: StaticNAT not found for PublicIP %s", publicIPIID.NameId)
	}

	err = staticnat.Delete(h.NetworkClient, staticNatID).ExtractErr()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		loggingError(hiscallInfo, err)
		return false, err
	}
	loggingInfo(hiscallInfo, start)
	return true, nil
}

// resolveVMPrivateIP finds the first private IP of a VM via its ports.
func (h *KTVpcPublicIPHandler) resolveVMPrivateIP(vmIID irs.IID) (string, error) {
	deviceID := vmIID.SystemId
	if deviceID == "" {
		deviceID = vmIID.NameId
	}
	allPages, err := ports.List(h.NetworkClient, ports.ListOpts{DeviceID: deviceID}).AllPages()
	if err != nil {
		return "", fmt.Errorf("failed to list ports for VM [%s]: %w", deviceID, err)
	}
	portList, err := ports.ExtractPorts(allPages)
	if err != nil {
		return "", fmt.Errorf("failed to extract ports for VM [%s]: %w", deviceID, err)
	}
	for _, p := range portList {
		if len(p.FixedIPs) > 0 && p.FixedIPs[0].IPAddress != "" {
			return p.FixedIPs[0].IPAddress, nil
		}
	}
	return "", fmt.Errorf("no private IP found for VM [%s]", deviceID)
}
