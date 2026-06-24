// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// AWS NIC (Elastic Network Interface) Handler
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

type AwsNICHandler struct {
	Region     idrv.RegionInfo
	Client     *ec2.EC2
	TagHandler *AwsTagHandler
}

func (h *AwsNICHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, "ListIID", "DescribeNetworkInterfaces()")
	start := call.Start()
	result, err := h.Client.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{})
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil { cblogger.Error(err); LoggingError(hiscallInfo, err); return nil, err }
	LoggingInfo(hiscallInfo, start)
	var list []*irs.IID
	for _, ni := range result.NetworkInterfaces {
		nameId := awsNICNameId(ni)
		list = append(list, &irs.IID{NameId: nameId, SystemId: aws.StringValue(ni.NetworkInterfaceId)})
	}
	return list, nil
}

func (h *AwsNICHandler) CreateNIC(req irs.NICReqInfo) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, req.IId.NameId, "CreateNetworkInterface()")
	start := call.Start()

	sgIds := make([]*string, len(req.SecurityGroupIIDs))
	for i, sg := range req.SecurityGroupIIDs {
		id := sg.SystemId; if id == "" { id = sg.NameId }
		sgIds[i] = aws.String(id)
	}
	subnetId := req.SubnetIID.SystemId; if subnetId == "" { subnetId = req.SubnetIID.NameId }

	input := &ec2.CreateNetworkInterfaceInput{
		SubnetId:    aws.String(subnetId),
		Description: aws.String(req.IId.NameId),
		Groups:      sgIds,
	}
	result, err := h.Client.CreateNetworkInterface(input)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil { cblogger.Error(err); LoggingError(hiscallInfo, err); return irs.NICInfo{}, err }

	// Tag with Name
	tags := []*ec2.Tag{{Key: aws.String("Name"), Value: aws.String(req.IId.NameId)}}
	for _, kv := range req.TagList { tags = append(tags, &ec2.Tag{Key: aws.String(kv.Key), Value: aws.String(kv.Value)}) }
	h.Client.CreateTags(&ec2.CreateTagsInput{Resources: []*string{result.NetworkInterface.NetworkInterfaceId}, Tags: tags})
	LoggingInfo(hiscallInfo, start)

	info := extractAwsNICInfo(result.NetworkInterface)
	info.IId.NameId = req.IId.NameId
	return info, nil
}

func (h *AwsNICHandler) ListNIC() ([]*irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, "All", "DescribeNetworkInterfaces()")
	start := call.Start()
	result, err := h.Client.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{})
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil { cblogger.Error(err); LoggingError(hiscallInfo, err); return nil, err }
	LoggingInfo(hiscallInfo, start)
	var list []*irs.NICInfo
	for _, ni := range result.NetworkInterfaces {
		info := extractAwsNICInfo(ni)
		list = append(list, &info)
	}
	if list == nil { list = []*irs.NICInfo{} }
	return list, nil
}

func (h *AwsNICHandler) GetNIC(iid irs.IID) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, iid.NameId, "DescribeNetworkInterfaces()")
	start := call.Start()
	input := &ec2.DescribeNetworkInterfacesInput{}
	if iid.SystemId != "" {
		input.NetworkInterfaceIds = []*string{aws.String(iid.SystemId)}
	} else {
		input.Filters = []*ec2.Filter{{Name: aws.String("tag:Name"), Values: []*string{aws.String(iid.NameId)}}}
	}
	result, err := h.Client.DescribeNetworkInterfaces(input)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil { cblogger.Error(err); LoggingError(hiscallInfo, err); return irs.NICInfo{}, err }
	if len(result.NetworkInterfaces) == 0 { return irs.NICInfo{}, fmt.Errorf("NIC not found: %s", iid.NameId) }
	LoggingInfo(hiscallInfo, start)
	info := extractAwsNICInfo(result.NetworkInterfaces[0])
	if iid.NameId != "" { info.IId.NameId = iid.NameId }
	return info, nil
}

func (h *AwsNICHandler) DeleteNIC(iid irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, iid.NameId, "DeleteNetworkInterface()")
	start := call.Start()
	info, err := h.GetNIC(iid)
	if err != nil { return false, err }
	_, err = h.Client.DeleteNetworkInterface(&ec2.DeleteNetworkInterfaceInput{NetworkInterfaceId: aws.String(info.IId.SystemId)})
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil { cblogger.Error(err); LoggingError(hiscallInfo, err); return false, err }
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (h *AwsNICHandler) AttachNIC(nicIID irs.IID, vmIID irs.IID) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, nicIID.NameId, "AttachNetworkInterface()")
	start := call.Start()
	nicInfo, err := h.GetNIC(nicIID)
	if err != nil { return irs.NICInfo{}, err }
	instanceId := vmIID.SystemId; if instanceId == "" { instanceId = vmIID.NameId }
	// Find next device index by listing current NICs on VM
	devIdx := int64(1)
	vms, _ := h.Client.DescribeInstances(&ec2.DescribeInstancesInput{InstanceIds: []*string{aws.String(instanceId)}})
	if vms != nil && len(vms.Reservations) > 0 && len(vms.Reservations[0].Instances) > 0 {
		devIdx = int64(len(vms.Reservations[0].Instances[0].NetworkInterfaces))
	}
	_, err = h.Client.AttachNetworkInterface(&ec2.AttachNetworkInterfaceInput{
		NetworkInterfaceId: aws.String(nicInfo.IId.SystemId),
		InstanceId:         aws.String(instanceId),
		DeviceIndex:        aws.Int64(devIdx),
	})
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil { cblogger.Error(err); LoggingError(hiscallInfo, err); return irs.NICInfo{}, err }
	LoggingInfo(hiscallInfo, start)
	return h.GetNIC(nicIID)
}

func (h *AwsNICHandler) DetachNIC(nicIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, nicIID.NameId, "DetachNetworkInterface()")
	start := call.Start()
	info, err := h.GetNIC(nicIID)
	if err != nil { return false, err }
	if info.Status != irs.NICAttached { return false, fmt.Errorf("NIC %s is not attached", nicIID.NameId) }
	// Find attachment ID
	result, _ := h.Client.DescribeNetworkInterfaces(&ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: []*string{aws.String(info.IId.SystemId)},
	})
	if len(result.NetworkInterfaces) == 0 { return false, fmt.Errorf("NIC not found") }
	ni := result.NetworkInterfaces[0]
	if ni.Attachment == nil { return false, fmt.Errorf("NIC has no attachment info") }
	_, err = h.Client.DetachNetworkInterface(&ec2.DetachNetworkInterfaceInput{AttachmentId: ni.Attachment.AttachmentId})
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil { cblogger.Error(err); LoggingError(hiscallInfo, err); return false, err }
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

// GetNICOSConfigScript returns an empty string for AWS.
// AWS uses DHCP-based multi-NIC routing (ec2-net-utils / cloud-init) and requires no manual OS configuration.
func (h *AwsNICHandler) GetNICOSConfigScript(nicIID irs.IID) (string, error) {
	return "", nil
}

func extractAwsNICInfo(ni *ec2.NetworkInterface) irs.NICInfo {
	status := irs.NICAvailable
	if aws.StringValue(ni.Status) == "in-use" { status = irs.NICAttached }
	info := irs.NICInfo{
		IId:        irs.IID{NameId: awsNICNameId(ni), SystemId: aws.StringValue(ni.NetworkInterfaceId)},
		SubnetIID:  irs.IID{SystemId: aws.StringValue(ni.SubnetId)},
		VpcIID:     irs.IID{SystemId: aws.StringValue(ni.VpcId)},
		PrivateIP:  aws.StringValue(ni.PrivateIpAddress),
		MACAddress: aws.StringValue(ni.MacAddress),
		Status:     status,
		CreatedTime: time.Time{},
	}
	// Build parallel PrivateIPs / PublicIPs arrays (index-aligned for 1:1 mapping)
	var privIPs, pubIPs []string
	for _, pip := range ni.PrivateIpAddresses {
		privIPs = append(privIPs, aws.StringValue(pip.PrivateIpAddress))
		pubIP := ""
		if pip.Association != nil && pip.Association.PublicIp != nil {
			pubIP = aws.StringValue(pip.Association.PublicIp)
			if info.PublicIP == "" {
				info.PublicIP = pubIP
			}
		}
		pubIPs = append(pubIPs, pubIP)
	}
	info.PrivateIPs = privIPs
	info.PublicIPs  = pubIPs
	if info.PublicIP == "" && ni.Association != nil && ni.Association.PublicIp != nil {
		info.PublicIP = aws.StringValue(ni.Association.PublicIp)
	}
	if ni.Attachment != nil {
		if ni.Attachment.InstanceId != nil {
			info.OwnerVM = irs.IID{NameId: aws.StringValue(ni.Attachment.InstanceId), SystemId: aws.StringValue(ni.Attachment.InstanceId)}
		}
		if ni.Attachment.DeviceIndex != nil { info.DeviceIndex = int(aws.Int64Value(ni.Attachment.DeviceIndex)) }
	}
	var sgs []irs.IID
	for _, sg := range ni.Groups { sgs = append(sgs, irs.IID{NameId: aws.StringValue(sg.GroupName), SystemId: aws.StringValue(sg.GroupId)}) }
	info.SecurityGroupIIDs = sgs
	var tagList []irs.KeyValue
	for _, t := range ni.TagSet {
		if aws.StringValue(t.Key) != "Name" { tagList = append(tagList, irs.KeyValue{Key: aws.StringValue(t.Key), Value: aws.StringValue(t.Value)}) }
	}
	info.TagList = tagList
	info.KeyValueList = []irs.KeyValue{
		{Key: "NetworkInterfaceId", Value: aws.StringValue(ni.NetworkInterfaceId)},
		{Key: "InterfaceType", Value: aws.StringValue(ni.InterfaceType)},
	}
	return info
}

func awsNICNameId(ni *ec2.NetworkInterface) string {
	for _, t := range ni.TagSet { if aws.StringValue(t.Key) == "Name" { return aws.StringValue(t.Value) } }
	return aws.StringValue(ni.NetworkInterfaceId)
}

// AddPrivateIP assigns an additional secondary private IP to an ENI.
// If privateIP is empty, AWS auto-assigns one.
func (h *AwsNICHandler) AddPrivateIP(nicIID irs.IID, privateIP string) (irs.NICInfo, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, nicIID.NameId, "AssignPrivateIpAddresses()")
	start := call.Start()

	info, err := h.GetNIC(nicIID)
	if err != nil {
		return irs.NICInfo{}, err
	}

	input := &ec2.AssignPrivateIpAddressesInput{
		NetworkInterfaceId: aws.String(info.IId.SystemId),
	}
	if privateIP != "" {
		input.PrivateIpAddresses = []*string{aws.String(privateIP)}
	} else {
		input.SecondaryPrivateIpAddressCount = aws.Int64(1)
	}

	_, err = h.Client.AssignPrivateIpAddresses(input)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.NICInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	return h.GetNIC(irs.IID{NameId: nicIID.NameId, SystemId: info.IId.SystemId})
}

// RemovePrivateIP unassigns a secondary private IP from an ENI.
func (h *AwsNICHandler) RemovePrivateIP(nicIID irs.IID, privateIP string) (bool, error) {
	hiscallInfo := GetCallLogScheme(h.Region, call.NIC, nicIID.NameId, "UnassignPrivateIpAddresses()")
	start := call.Start()

	info, err := h.GetNIC(nicIID)
	if err != nil {
		return false, err
	}

	_, err = h.Client.UnassignPrivateIpAddresses(&ec2.UnassignPrivateIpAddressesInput{
		NetworkInterfaceId: aws.String(info.IId.SystemId),
		PrivateIpAddresses: []*string{aws.String(privateIP)},
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
