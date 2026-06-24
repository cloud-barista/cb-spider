// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// AWS Public IP (Elastic IP) Handler
//
// by CB-Spider Team, 2025.06.

package resources

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsPublicIPHandler struct {
	Region     idrv.RegionInfo
	Client     *ec2.EC2
	TagHandler *AwsTagHandler
}

// ListIID returns all EIP IIDs in the current region (VPC domain only).
func (h *AwsPublicIPHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, "ListIID", "DescribeAddresses()")
	start := call.Start()

	result, err := h.Client.DescribeAddresses(&ec2.DescribeAddressesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("domain"), Values: []*string{aws.String("vpc")}},
		},
	})
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	var iidList []*irs.IID
	for _, addr := range result.Addresses {
		nameId := aws.StringValue(addr.AllocationId)
		for _, tag := range addr.Tags {
			if aws.StringValue(tag.Key) == "Name" {
				nameId = aws.StringValue(tag.Value)
				break
			}
		}
		iidList = append(iidList, &irs.IID{NameId: nameId, SystemId: aws.StringValue(addr.AllocationId)})
	}
	return iidList, nil
}

// CreatePublicIP allocates a new Elastic IP address (VPC domain) and tags it.
func (h *AwsPublicIPHandler) CreatePublicIP(reqInfo irs.PublicIPInfo) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, reqInfo.IId.NameId, "AllocateAddress()")
	start := call.Start()

	allocRes, err := h.Client.AllocateAddress(&ec2.AllocateAddressInput{
		Domain: aws.String("vpc"),
	})
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	// Build tag list: Name tag + user-provided tags
	tags := []*ec2.Tag{{Key: aws.String("Name"), Value: aws.String(reqInfo.IId.NameId)}}
	for _, kv := range reqInfo.TagList {
		tags = append(tags, &ec2.Tag{Key: aws.String(kv.Key), Value: aws.String(kv.Value)})
	}
	_, err = h.Client.CreateTags(&ec2.CreateTagsInput{
		Resources: []*string{allocRes.AllocationId},
		Tags:      tags,
	})
	if err != nil {
		// Best-effort cleanup: release the address
		h.Client.ReleaseAddress(&ec2.ReleaseAddressInput{AllocationId: allocRes.AllocationId})
		cblogger.Error(err)
		return irs.PublicIPInfo{}, fmt.Errorf("failed to tag Public IP %s: %w", aws.StringValue(allocRes.AllocationId), err)
	}

	return h.GetPublicIP(irs.IID{NameId: reqInfo.IId.NameId, SystemId: aws.StringValue(allocRes.AllocationId)})
}

// ListPublicIP returns all EIPs (VPC domain) that have a Name tag.
func (h *AwsPublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, "All", "DescribeAddresses()")
	start := call.Start()

	result, err := h.Client.DescribeAddresses(&ec2.DescribeAddressesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("domain"), Values: []*string{aws.String("vpc")}},
		},
	})
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	var infoList []*irs.PublicIPInfo
	for _, addr := range result.Addresses {
		info := extractAwsPublicIPInfo(addr)
		infoList = append(infoList, &info)
	}
	if infoList == nil {
		infoList = []*irs.PublicIPInfo{}
	}
	return infoList, nil
}

// GetPublicIP retrieves a single EIP by IID (NameId = Name tag, SystemId = AllocationId).
func (h *AwsPublicIPHandler) GetPublicIP(publicIPIID irs.IID) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "DescribeAddresses()")
	start := call.Start()

	input := &ec2.DescribeAddressesInput{}
	if publicIPIID.SystemId != "" {
		input.AllocationIds = []*string{aws.String(publicIPIID.SystemId)}
	} else {
		input.Filters = []*ec2.Filter{
			{Name: aws.String("tag:Name"), Values: []*string{aws.String(publicIPIID.NameId)}},
		}
	}

	result, err := h.Client.DescribeAddresses(input)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	if len(result.Addresses) == 0 {
		return irs.PublicIPInfo{}, fmt.Errorf("PublicIP not found: %s", publicIPIID.NameId)
	}

	info := extractAwsPublicIPInfo(result.Addresses[0])
	if publicIPIID.NameId != "" {
		info.IId.NameId = publicIPIID.NameId
	}
	return info, nil
}

// DeletePublicIP releases an Elastic IP address.
func (h *AwsPublicIPHandler) DeletePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "ReleaseAddress()")
	start := call.Start()

	systemId := publicIPIID.SystemId
	if systemId == "" {
		info, err := h.GetPublicIP(publicIPIID)
		if err != nil {
			return false, err
		}
		systemId = info.IId.SystemId
	}

	_, err := h.Client.ReleaseAddress(&ec2.ReleaseAddressInput{
		AllocationId: aws.String(systemId),
	})
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func extractAwsPublicIPInfo(addr *ec2.Address) irs.PublicIPInfo {
	info := irs.PublicIPInfo{
		IId: irs.IID{
			NameId:   aws.StringValue(addr.AllocationId),
			SystemId: aws.StringValue(addr.AllocationId),
		},
		PublicIPAddress: aws.StringValue(addr.PublicIp),
		Status:          irs.PublicIPAvailable,
		CreatedTime:     time.Time{},
	}

	// Name tag
	var tagList []irs.KeyValue
	for _, t := range addr.Tags {
		if aws.StringValue(t.Key) == "Name" {
			info.IId.NameId = aws.StringValue(t.Value)
		} else {
			tagList = append(tagList, irs.KeyValue{Key: aws.StringValue(t.Key), Value: aws.StringValue(t.Value)})
		}
	}
	info.TagList = tagList

	if addr.InstanceId != nil {
		info.OwnedVM = irs.IID{NameId: aws.StringValue(addr.InstanceId), SystemId: aws.StringValue(addr.InstanceId)}
	}
	// NIC association (EIP may be associated with NIC without an instance)
	if addr.NetworkInterfaceId != nil {
		info.Status = irs.PublicIPAssociated
		info.OwnedNIC = irs.IID{NameId: aws.StringValue(addr.NetworkInterfaceId), SystemId: aws.StringValue(addr.NetworkInterfaceId)}
	}
	if addr.PrivateIpAddress != nil {
		info.OwnedPrivateIP = aws.StringValue(addr.PrivateIpAddress)
	}

	kvList := []irs.KeyValue{
		{Key: "Domain", Value: aws.StringValue(addr.Domain)},
		{Key: "AllocationId", Value: aws.StringValue(addr.AllocationId)},
	}
	if addr.AssociationId != nil {
		kvList = append(kvList, irs.KeyValue{Key: "AssociationId", Value: aws.StringValue(addr.AssociationId)})
	}
	if addr.PublicIpv4Pool != nil {
		kvList = append(kvList, irs.KeyValue{Key: "PublicIpv4Pool", Value: aws.StringValue(addr.PublicIpv4Pool)})
	}
	info.KeyValueList = kvList

	return info
}

// AssociatePublicIP associates an Elastic IP with a VM instance.
func (h *AwsPublicIPHandler) AssociatePublicIP(publicIPIID irs.IID, vmIID irs.IID, nicIID irs.IID, privateIP string) (irs.PublicIPInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "AssociateAddress()")
	start := call.Start()

	// Resolve allocation ID
	info, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return irs.PublicIPInfo{}, err
	}
	allocationId := info.IId.SystemId

	// Resolve instance ID
	instanceId := vmIID.SystemId
	if instanceId == "" {
		instanceId = vmIID.NameId
	}

	assocInput := &ec2.AssociateAddressInput{
		AllocationId: aws.String(allocationId),
	}
	// Use NIC-level association if nicIID provided (more precise)
	if nicIID.SystemId != "" {
		assocInput.NetworkInterfaceId = aws.String(nicIID.SystemId)
		if privateIP != "" {
			assocInput.PrivateIpAddress = aws.String(privateIP)
		}
	} else if nicIID.NameId != "" {
		assocInput.NetworkInterfaceId = aws.String(nicIID.NameId)
		if privateIP != "" {
			assocInput.PrivateIpAddress = aws.String(privateIP)
		}
	} else {
		assocInput.InstanceId = aws.String(instanceId)
	}
	_, err = h.Client.AssociateAddress(assocInput)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.PublicIPInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	return h.GetPublicIP(irs.IID{NameId: publicIPIID.NameId, SystemId: allocationId})
}

// DisassociatePublicIP removes the association between an EIP and a VM instance.
func (h *AwsPublicIPHandler) DisassociatePublicIP(publicIPIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.PUBLICIP, publicIPIID.NameId, "DisassociateAddress()")
	start := call.Start()

	info, err := h.GetPublicIP(publicIPIID)
	if err != nil {
		return false, err
	}

	// Find AssociationId from KeyValueList
	associationId := ""
	for _, kv := range info.KeyValueList {
		if kv.Key == "AssociationId" {
			associationId = kv.Value
			break
		}
	}
	if associationId == "" {
		return false, fmt.Errorf("PublicIP %s is not associated with any VM", publicIPIID.NameId)
	}

	_, err = h.Client.DisassociateAddress(&ec2.DisassociateAddressInput{
		AssociationId: aws.String(associationId),
	})
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}
