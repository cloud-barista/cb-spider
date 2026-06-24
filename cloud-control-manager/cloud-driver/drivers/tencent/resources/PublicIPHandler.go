// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// Tencent Cloud EIP (Elastic IP) Handler
//
// by CB-Spider Team, 2025.06.

package resources

import (
	"fmt"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tencentvpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

type TencentPublicIPHandler struct {
	Region    idrv.RegionInfo
	VPCClient *tencentvpc.Client
}

// Fix 9: ListIID - pagination loop using Offset/Limit.
func (h *TencentPublicIPHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, "ListIID", "DescribeAddresses()")
	start := call.Start()

	var iidList []*irs.IID
	var offset int64 = 0
	const limit int64 = 100

	for {
		req := tencentvpc.NewDescribeAddressesRequest()
		req.Offset = common.Int64Ptr(offset)
		req.Limit = common.Int64Ptr(limit)

		resp, err := h.VPCClient.DescribeAddresses(req)
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		if err != nil {
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return nil, err
		}

		for _, eip := range resp.Response.AddressSet {
			nameId := tencentEIPNameId(eip)
			iidList = append(iidList, &irs.IID{NameId: nameId, SystemId: *eip.AddressId})
		}

		fetched := offset + int64(len(resp.Response.AddressSet))
		if resp.Response.TotalCount == nil || fetched >= *resp.Response.TotalCount || len(resp.Response.AddressSet) == 0 {
			break
		}
		offset += limit
	}

	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

func (h *TencentPublicIPHandler) CreatePublicIP(reqInfo irs.PublicIPInfo) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, reqInfo.IId.NameId, "AllocateAddresses()")
	start := call.Start()

	req := tencentvpc.NewAllocateAddressesRequest()
	req.AddressCount = common.Int64Ptr(1)
	req.InternetServiceProvider = common.StringPtr("BGP")
	req.AddressType = common.StringPtr("EIP")

	// Set tags including Name
	tags := []*tencentvpc.Tag{{Key: common.StringPtr("Name"), Value: common.StringPtr(reqInfo.IId.NameId)}}
	for _, kv := range reqInfo.TagList {
		tags = append(tags, &tencentvpc.Tag{Key: common.StringPtr(kv.Key), Value: common.StringPtr(kv.Value)})
	}
	req.Tags = tags

	resp, err := h.VPCClient.AllocateAddresses(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	if len(resp.Response.AddressSet) == 0 {
		return irs.PublicIPInfo{}, fmt.Errorf("AllocateAddresses returned empty set")
	}

	allocID := *resp.Response.AddressSet[0]

	// Tencent EIP starts in CREATING status — wait until UNBIND (available) and IP is assigned.
	for i := 0; i < 30; i++ {
		info, getErr := h.GetPublicIP(irs.IID{NameId: reqInfo.IId.NameId, SystemId: allocID})
		if getErr == nil && info.PublicIPAddress != "" && info.PublicIPAddress != "NA" {
			return info, nil
		}
		time.Sleep(2 * time.Second)
	}
	return h.GetPublicIP(irs.IID{NameId: reqInfo.IId.NameId, SystemId: allocID})
}

// Fix 9: ListPublicIP - pagination loop using Offset/Limit.
func (h *TencentPublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, "All", "DescribeAddresses()")
	start := call.Start()

	var infoList []*irs.PublicIPInfo
	var offset int64 = 0
	const limit int64 = 100

	for {
		req := tencentvpc.NewDescribeAddressesRequest()
		req.Offset = common.Int64Ptr(offset)
		req.Limit = common.Int64Ptr(limit)

		resp, err := h.VPCClient.DescribeAddresses(req)
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		if err != nil {
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return nil, err
		}

		for _, eip := range resp.Response.AddressSet {
			info := extractTencentPublicIPInfo(eip)
			infoList = append(infoList, &info)
		}

		fetched := offset + int64(len(resp.Response.AddressSet))
		if resp.Response.TotalCount == nil || fetched >= *resp.Response.TotalCount || len(resp.Response.AddressSet) == 0 {
			break
		}
		offset += limit
	}

	LoggingInfo(hiscallInfo, start)
	if infoList == nil {
		infoList = []*irs.PublicIPInfo{}
	}
	return infoList, nil
}

func (h *TencentPublicIPHandler) GetPublicIP(publicIPIID irs.IID) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "DescribeAddresses()")
	start := call.Start()

	req := tencentvpc.NewDescribeAddressesRequest()
	if publicIPIID.SystemId != "" {
		req.AddressIds = []*string{common.StringPtr(publicIPIID.SystemId)}
	} else {
		req.Filters = []*tencentvpc.Filter{
			{Name: common.StringPtr("address-name"), Values: []*string{common.StringPtr(publicIPIID.NameId)}},
		}
	}

	resp, err := h.VPCClient.DescribeAddresses(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	if len(resp.Response.AddressSet) == 0 {
		return irs.PublicIPInfo{}, fmt.Errorf("PublicIP not found: %s", publicIPIID.NameId)
	}

	info := extractTencentPublicIPInfo(resp.Response.AddressSet[0])
	if publicIPIID.NameId != "" {
		info.IId.NameId = publicIPIID.NameId
	}
	return info, nil
}

func (h *TencentPublicIPHandler) DeletePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "ReleaseAddresses()")
	start := call.Start()

	systemId := publicIPIID.SystemId
	if systemId == "" {
		info, err := h.GetPublicIP(publicIPIID)
		if err != nil {
			return false, err
		}
		systemId = info.IId.SystemId
	}

	req := tencentvpc.NewReleaseAddressesRequest()
	req.AddressIds = []*string{common.StringPtr(systemId)}

	_, err := h.VPCClient.ReleaseAddresses(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

// extractTencentPublicIPInfo converts a Tencent Address to irs.PublicIPInfo.
// Fix 6: set OwnedNIC when NetworkInterfaceId is set (ENI-bound EIP), otherwise OwnedVM.
//        Also set OwnedPrivateIP from PrivateAddressIp.
// Fix 7: proper status mapping for all Tencent EIP statuses.
func extractTencentPublicIPInfo(eip *tencentvpc.Address) irs.PublicIPInfo {
	nameId := tencentEIPNameId(eip)

	// Fix 7: map all known Tencent EIP statuses
	var status irs.PublicIPStatus
	if eip.AddressStatus != nil {
		switch *eip.AddressStatus {
		case "BIND", "BINDING", "BIND_ENI":
			status = irs.PublicIPAssociated
		default:
			// UNBIND, UNBINDING, OFFLINING, CREATING
			status = irs.PublicIPAvailable
		}
	} else {
		status = irs.PublicIPAvailable
	}

	info := irs.PublicIPInfo{
		IId:             irs.IID{NameId: nameId, SystemId: *eip.AddressId},
		PublicIPAddress: derefTencentStr(eip.AddressIp),
		Status:          status,
		CreatedTime:     time.Time{},
	}

	// Fix 6: distinguish NIC-bound vs VM-bound EIP using NetworkInterfaceId field.
	// When NetworkInterfaceId is set the EIP is bound to an ENI (BIND_ENI or secondary IP on ENI).
	if eip.NetworkInterfaceId != nil && *eip.NetworkInterfaceId != "" {
		info.OwnedNIC = irs.IID{NameId: *eip.NetworkInterfaceId, SystemId: *eip.NetworkInterfaceId}
	} else if eip.InstanceId != nil && *eip.InstanceId != "" {
		info.OwnedVM = irs.IID{NameId: *eip.InstanceId, SystemId: *eip.InstanceId}
	}

	// Fix 6: set OwnedPrivateIP from PrivateAddressIp field
	if eip.PrivateAddressIp != nil && *eip.PrivateAddressIp != "" {
		info.OwnedPrivateIP = *eip.PrivateAddressIp
	}

	var tagList []irs.KeyValue
	for _, t := range eip.TagSet {
		if t.Key != nil && *t.Key != "Name" {
			tagList = append(tagList, irs.KeyValue{Key: derefTencentStr(t.Key), Value: derefTencentStr(t.Value)})
		}
	}
	info.TagList = tagList

	info.KeyValueList = []irs.KeyValue{
		{Key: "AddressId", Value: *eip.AddressId},
		{Key: "AddressStatus", Value: derefTencentStr(eip.AddressStatus)},
		{Key: "AddressType", Value: derefTencentStr(eip.AddressType)},
	}

	return info
}

func tencentEIPNameId(eip *tencentvpc.Address) string {
	for _, t := range eip.TagSet {
		if t.Key != nil && *t.Key == "Name" && t.Value != nil {
			return *t.Value
		}
	}
	if eip.AddressName != nil && *eip.AddressName != "" {
		return *eip.AddressName
	}
	return *eip.AddressId
}

func derefTencentStr(s *string) string {
	if s == nil {
		return "NA"
	}
	return *s
}

// AssociatePublicIP binds a Tencent EIP to a CVM instance or ENI.
// Fix 8: support NIC-level association when nicIID is provided.
func (h *TencentPublicIPHandler) AssociatePublicIP(publicIPIID irs.IID, vmIID irs.IID, nicIID irs.IID, privateIP string) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "AssociateAddress()")
	start := call.Start()

	info, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return irs.PublicIPInfo{}, err
	}

	req := tencentvpc.NewAssociateAddressRequest()
	req.AddressId = common.StringPtr(info.IId.SystemId)

	// Fix 8: use ENI-level association when nicIID is provided.
	// Tencent requires both NetworkInterfaceId AND PrivateIpAddress for ENI binding.
	// If privateIP is empty (auto), fetch the NIC's primary private IP.
	if nicIID.SystemId != "" || nicIID.NameId != "" {
		nicSystemId := nicIID.SystemId
		if nicSystemId == "" {
			nicSystemId = nicIID.NameId
		}
		req.NetworkInterfaceId = common.StringPtr(nicSystemId)

		ip := privateIP
		if ip == "" {
			// Tencent requires PrivateIpAddress; fetch primary IP from the ENI.
			nicReq := tencentvpc.NewDescribeNetworkInterfacesRequest()
			nicReq.NetworkInterfaceIds = []*string{common.StringPtr(nicSystemId)}
			if nicResp, e := h.VPCClient.DescribeNetworkInterfaces(nicReq); e == nil &&
				len(nicResp.Response.NetworkInterfaceSet) > 0 {
				for _, pip := range nicResp.Response.NetworkInterfaceSet[0].PrivateIpAddressSet {
					if pip.Primary != nil && *pip.Primary && pip.PrivateIpAddress != nil {
						ip = *pip.PrivateIpAddress
						break
					}
				}
			}
		}
		if ip != "" {
			req.PrivateIpAddress = common.StringPtr(ip)
		}
	} else {
		instanceId := vmIID.SystemId
		if instanceId == "" {
			instanceId = vmIID.NameId
		}
		req.InstanceId = common.StringPtr(instanceId)
	}

	_, err = h.VPCClient.AssociateAddress(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	return h.GetPublicIP(publicIPIID)
}

// DisassociatePublicIP unbinds a Tencent EIP from a CVM instance.
func (h *TencentPublicIPHandler) DisassociatePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "DisassociateAddress()")
	start := call.Start()

	info, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return false, err
	}
	if info.Status != irs.PublicIPAssociated {
		return false, fmt.Errorf("PublicIP %s is not associated", publicIPIID.NameId)
	}

	req := tencentvpc.NewDisassociateAddressRequest()
	req.AddressId = common.StringPtr(info.IId.SystemId)

	_, err = h.VPCClient.DisassociateAddress(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	// Tencent DisassociateAddress is async — poll until the EIP reaches UNBIND (Available).
	for i := 0; i < 30; i++ {
		pollInfo, pollErr := h.GetPublicIP(publicIPIID)
		if pollErr == nil && pollInfo.Status == irs.PublicIPAvailable {
			return true, nil
		}
		time.Sleep(2 * time.Second)
	}
	return true, nil
}
