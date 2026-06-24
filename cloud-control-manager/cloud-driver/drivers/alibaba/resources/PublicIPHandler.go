// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// Alibaba Cloud EIP (Elastic IP) Handler
//
// by CB-Spider Team, 2025.06.

package resources

import (
	"fmt"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AlibabaPublicIPHandler struct {
	Region    idrv.RegionInfo
	VpcClient *vpc.Client
}

func (h *AlibabaPublicIPHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, "ListIID", "DescribeEipAddresses()")
	start := call.Start()

	req := vpc.CreateDescribeEipAddressesRequest()
	req.RegionId = h.Region.Region
	req.PageSize = requests.NewInteger(50)
	req.PageNumber = requests.NewInteger(1)

	var iidList []*irs.IID
	for {
		resp, err := h.VpcClient.DescribeEipAddresses(req)
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		if err != nil {
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return nil, err
		}

		for _, eip := range resp.EipAddresses.EipAddress {
			nameId := eip.Name
			if nameId == "" {
				nameId = eip.AllocationId
			}
			iidList = append(iidList, &irs.IID{NameId: nameId, SystemId: eip.AllocationId})
		}

		if len(resp.EipAddresses.EipAddress) < resp.PageSize {
			break
		}
		req.PageNumber = requests.NewInteger(resp.PageNumber + 1)
	}

	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

func (h *AlibabaPublicIPHandler) CreatePublicIP(reqInfo irs.PublicIPInfo) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, reqInfo.IId.NameId, "AllocateEipAddress()")
	start := call.Start()

	req := vpc.CreateAllocateEipAddressRequest()
	req.RegionId = h.Region.Region
	req.Name = reqInfo.IId.NameId
	req.InternetChargeType = "PayByTraffic"
	req.Bandwidth = "100"

	resp, err := h.VpcClient.AllocateEipAddress(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	// Add tags if provided
	if len(reqInfo.TagList) > 0 {
		tagReq := vpc.CreateTagResourcesRequest()
		tagReq.RegionId = h.Region.Region
		tagReq.ResourceType = "eip"
		tagReq.ResourceId = &[]string{resp.AllocationId}
		var tags []vpc.TagResourcesTag
		for _, kv := range reqInfo.TagList {
			tags = append(tags, vpc.TagResourcesTag{Key: kv.Key, Value: kv.Value})
		}
		tagReq.Tag = &tags
		if _, tagErr := h.VpcClient.TagResources(tagReq); tagErr != nil {
			cblogger.Warn("Failed to tag EIP:", tagErr)
		}
	}

	return h.GetPublicIP(irs.IID{NameId: reqInfo.IId.NameId, SystemId: resp.AllocationId})
}

// Fix 5: ListPublicIP - full pagination loop
func (h *AlibabaPublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, "All", "DescribeEipAddresses()")
	start := call.Start()

	req := vpc.CreateDescribeEipAddressesRequest()
	req.RegionId = h.Region.Region
	req.PageSize = requests.NewInteger(50)
	req.PageNumber = requests.NewInteger(1)

	var infoList []*irs.PublicIPInfo
	for {
		resp, err := h.VpcClient.DescribeEipAddresses(req)
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		if err != nil {
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return nil, err
		}

		for _, eip := range resp.EipAddresses.EipAddress {
			info := extractAlibabaPublicIPInfo(eip)
			infoList = append(infoList, &info)
		}

		if len(resp.EipAddresses.EipAddress) < resp.PageSize {
			break
		}
		req.PageNumber = requests.NewInteger(resp.PageNumber + 1)
	}

	LoggingInfo(hiscallInfo, start)
	if infoList == nil {
		infoList = []*irs.PublicIPInfo{}
	}
	return infoList, nil
}

func (h *AlibabaPublicIPHandler) GetPublicIP(publicIPIID irs.IID) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "DescribeEipAddresses()")
	start := call.Start()

	req := vpc.CreateDescribeEipAddressesRequest()
	req.RegionId = h.Region.Region

	if publicIPIID.SystemId != "" {
		req.AllocationId = publicIPIID.SystemId
	} else {
		req.EipName = publicIPIID.NameId
	}

	resp, err := h.VpcClient.DescribeEipAddresses(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	if len(resp.EipAddresses.EipAddress) == 0 {
		return irs.PublicIPInfo{}, fmt.Errorf("PublicIP not found: %s", publicIPIID.NameId)
	}

	info := extractAlibabaPublicIPInfo(resp.EipAddresses.EipAddress[0])
	if publicIPIID.NameId != "" {
		info.IId.NameId = publicIPIID.NameId
	}
	return info, nil
}

func (h *AlibabaPublicIPHandler) DeletePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "ReleaseEipAddress()")
	start := call.Start()

	systemId := publicIPIID.SystemId
	if systemId == "" {
		info, err := h.GetPublicIP(publicIPIID)
		if err != nil {
			return false, err
		}
		systemId = info.IId.SystemId
	}

	req := vpc.CreateReleaseEipAddressRequest()
	req.AllocationId = systemId

	_, err := h.VpcClient.ReleaseEipAddress(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

// Fix 1: extractAlibabaPublicIPInfo - OwnedNIC, OwnedVM, OwnedPrivateIP
// Fix 2: Status mapping - intermediate states
func extractAlibabaPublicIPInfo(eip vpc.EipAddress) irs.PublicIPInfo {
	nameId := eip.Name
	if nameId == "" {
		nameId = eip.AllocationId
	}

	// Fix 2: Map all known Alibaba EIP statuses
	var status irs.PublicIPStatus
	switch eip.Status {
	case "InUse", "Associating", "Binding":
		status = irs.PublicIPAssociated
	case "Unassociating", "Unbinding", "Deleting", "Available":
		status = irs.PublicIPAvailable
	default:
		status = irs.PublicIPAvailable
	}

	info := irs.PublicIPInfo{
		IId:             irs.IID{NameId: nameId, SystemId: eip.AllocationId},
		PublicIPAddress: eip.IpAddress,
		Status:          status,
		CreatedTime:     time.Time{},
	}

	// Fix 1: Distinguish NIC vs VM association by InstanceType
	if eip.InstanceId != "" {
		if eip.InstanceType == "NetworkInterface" {
			info.OwnedNIC = irs.IID{NameId: eip.InstanceId, SystemId: eip.InstanceId}
		} else {
			// EcsInstance or other
			info.OwnedVM = irs.IID{NameId: eip.InstanceId, SystemId: eip.InstanceId}
		}
	}

	// Fix 1: Set OwnedPrivateIP if available
	if eip.PrivateIpAddress != "" {
		info.OwnedPrivateIP = eip.PrivateIpAddress
	}

	info.KeyValueList = []irs.KeyValue{
		{Key: "AllocationId", Value: eip.AllocationId},
		{Key: "RegionId", Value: eip.RegionId},
		{Key: "InternetChargeType", Value: eip.InternetChargeType},
		{Key: "Bandwidth", Value: eip.Bandwidth},
		{Key: "Status", Value: eip.Status},
		{Key: "InstanceType", Value: eip.InstanceType},
	}

	return info
}

// Fix 3: AssociatePublicIP - support nicIID + privateIP
func (h *AlibabaPublicIPHandler) AssociatePublicIP(publicIPIID irs.IID, vmIID irs.IID, nicIID irs.IID, privateIP string) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "AssociateEipAddress()")
	start := call.Start()

	info, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return irs.PublicIPInfo{}, err
	}

	req := vpc.CreateAssociateEipAddressRequest()
	req.AllocationId = info.IId.SystemId

	// Fix 3: Use NIC-level association if nicIID is provided
	if nicIID.SystemId != "" || nicIID.NameId != "" {
		req.InstanceType = "NetworkInterface"
		if nicIID.SystemId != "" {
			req.InstanceId = nicIID.SystemId
		} else {
			req.InstanceId = nicIID.NameId
		}
		if privateIP != "" {
			req.PrivateIpAddress = privateIP
		}
	} else {
		req.InstanceType = "EcsInstance"
		instanceId := vmIID.SystemId
		if instanceId == "" {
			instanceId = vmIID.NameId
		}
		req.InstanceId = instanceId
	}

	_, err = h.VpcClient.AssociateEipAddress(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	return h.GetPublicIP(publicIPIID)
}

// Fix 4: DisassociatePublicIP - set InstanceId and InstanceType before calling unassociate
func (h *AlibabaPublicIPHandler) DisassociatePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "UnassociateEipAddress()")
	start := call.Start()

	info, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return false, err
	}
	if info.Status != irs.PublicIPAssociated {
		return false, fmt.Errorf("PublicIP %s is not associated", publicIPIID.NameId)
	}

	req := vpc.CreateUnassociateEipAddressRequest()
	req.AllocationId = info.IId.SystemId

	// Fix 4: Set InstanceId and InstanceType based on current association
	if info.OwnedNIC.SystemId != "" {
		req.InstanceId = info.OwnedNIC.SystemId
		req.InstanceType = "NetworkInterface"
	} else if info.OwnedVM.SystemId != "" {
		req.InstanceId = info.OwnedVM.SystemId
		req.InstanceType = "EcsInstance"
	}

	_, err = h.VpcClient.UnassociateEipAddress(req)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}
