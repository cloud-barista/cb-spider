// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by zephy@mz.co.kr, 2019.09.

package resources

import (
	"errors"
	"reflect"

	//"github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type AlibabaPublicIPHandler struct {
	Region idrv.RegionInfo
	Client *vpc.Client
}

// VM 생성 시 PublicIP의 AllocationId를 전달 받는 방식으로 1차 확정되어서 이 곳에서는 관련 로직을 제거 함.
// VMHandler.go에서 할당및 회수 함
func (publicIpHandler *AlibabaPublicIPHandler) CreatePublicIP(publicIPReqInfo irs.PublicIPReqInfo) (irs.PublicIPInfo, error) {
	cblogger.Info("Start : ", publicIPReqInfo)

	request := vpc.CreateAllocateEipAddressRequest()
	request.Scheme = "https"

	// 필수 Req Option
	request.InternetChargeType = CBInternetChargeType
	// 부가 Req Option
	request.Bandwidth = CBBandwidth

	//var publicIPInfo irs.PublicIPInfo

	// Attempt to allocate the Elastic IP address.
	allocRes, err := publicIpHandler.Client.AllocateEipAddress(request)
	if err != nil {
		cblogger.Errorf("Unable to allocate IP address, %v", err)
		return irs.PublicIPInfo{}, err
	}

	spew.Dump(allocRes)
	cblogger.Infof("EIP 생성 성공 - Public IP : [%s], Allocation Id : [%s]", allocRes.EipAddress, allocRes.AllocationId)

	// Tag에 Name 설정
	// cblogger.Info("Name 설정 ", publicIPReqInfo.Name)
	// _, errtag := publicIpHandler.Client.CreateTags(&ec2.CreateTagsInput{
	// 	Resources: []*string{allocRes.AllocationId},
	// 	Tags: []*ecs.Tag{
	// 		{
	// 			Key:   "Name",
	// 			Value: publicIPReqInfo.Name,
	// 		},
	// 	},
	// })
	// if errtag != nil {
	// 	cblogger.Error("Public IP에 Name Tag 설정 실패 : ")
	// }

	publicIPInfo, errExtract := publicIpHandler.GetPublicIP(allocRes.AllocationId)
	if errExtract != nil {
		cblogger.Errorf("Public IP 정보 추출 실패 ", errExtract)
		return irs.PublicIPInfo{}, err
	}

	return publicIPInfo, nil
}

func (publicIpHandler *AlibabaPublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	cblogger.Info("Start~")
	var publicIpList []*irs.PublicIPInfo

	request := vpc.CreateDescribeEipAddressesRequest()
	request.Scheme = "https"

	// Make the API request to EC2 filtering for the addresses in the
	// account's VPC.
	// result, err := publicIpHandler.Client.DescribeAddresses(&ec2.DescribeAddressesInput{
	// 	Filters: []*ecs.Filter{
	// 		{
	// 			Name:   aws.String("domain"),
	// 			Values: aws.StringSlice([]string{"vpc"}),
	// 		},
	// 	},
	// })
	result, err := publicIpHandler.Client.DescribeEipAddresses(request)
	if err != nil {
		cblogger.Errorf("Unable to get elastic IP address, %v", err)
		return nil, err
	}

	// Printout the IP addresses if there are any.
	if reflect.ValueOf(result.EipAddresses).IsNil() {
		//if len(result.EipAddresses) == 0 {
		//cblogger.Infof("No elastic IPs for %s region\n", publicIpHandler.Region)
		cblogger.Info("Not found Elastic IP List Information")
	} else {
		cblogger.Info("Elastic IPs")
		for _, addr := range result.EipAddresses.EipAddress {
			publicIPInfo := extractPublicIpDescribeInfo(&addr)
			publicIpList = append(publicIpList, &publicIPInfo)
		}
	}

	return publicIpList, nil
}

func extractPublicIpDescribeInfo(allocRes *vpc.EipAddress) irs.PublicIPInfo {
	var publicIPInfo irs.PublicIPInfo

	spew.Dump(allocRes)

	publicIPInfo.Name = allocRes.Name
	publicIPInfo.PublicIP = allocRes.IpAddress
	publicIPInfo.Status = allocRes.Status

	if !reflect.ValueOf(allocRes.InstanceId).IsNil() {
		//publicIPInfo.InstanceId = allocRes.InstanceId // Alibaba:연결된 VM
		publicIPInfo.OwnedVMID = allocRes.InstanceId // Alibaba:연결된 VM
	}

	keyValueList := []irs.KeyValue{
		{Key: "AllocationId", Value: allocRes.AllocationId},
		{Key: "AllocationTime", Value: allocRes.AllocationTime},
		{Key: "ChargeType", Value: allocRes.ChargeType},
		{Key: "Bandwidth", Value: allocRes.Bandwidth},
		{Key: "InternetChargeType", Value: allocRes.InternetChargeType},
		{Key: "InstanceType", Value: allocRes.InstanceType},
		{Key: "ResourceGroupId", Value: allocRes.ResourceGroupId},

		{Key: "Descritpion", Value: allocRes.Descritpion},
		{Key: "Mode", Value: allocRes.Mode},
		{Key: "InstanceRegionId", Value: allocRes.InstanceRegionId},
		{Key: "RegionId", Value: allocRes.RegionId},
	}

	publicIPInfo.KeyValueList = keyValueList

	//publicIPInfo.Domain = *allocRes.Domain
	//publicIPInfo.PublicIpv4Pool = *allocRes.PublicIpv4Pool
	//publicIPInfo.AllocationId = *allocRes.AllocationId

	if !reflect.ValueOf(allocRes.InstanceId).IsNil() {
		//publicIPInfo.InstanceId = *allocRes.InstanceId // Alibaba:연결 VMID
		keyValueList = append(keyValueList, irs.KeyValue{Key: "InstanceId", Value: allocRes.InstanceId}) // Alibaba:연결 VMID
	}

	// if !reflect.ValueOf(allocRes.AssociationId).IsNil() {
	// 	//publicIPInfo.AssociationId = *allocRes.AssociationId // AWS:연결ID
	// 	keyValueList = append(keyValueList, irs.KeyValue{Key: "AssociationId", Value: *allocRes.AssociationId}) // AWS:연결ID
	// }

	// if !reflect.ValueOf(allocRes.NetworkInterfaceId).IsNil() {
	// 	//publicIPInfo.NetworkInterfaceId = *allocRes.NetworkInterfaceId // AWS:연결된 Nic
	// 	//publicIPInfo.NetworkInterfaceOwnerId = *allocRes.NetworkInterfaceOwnerId
	// 	//publicIPInfo.PrivateIpAddress = *allocRes.PrivateIpAddress

	// 	keyValueList = append(keyValueList, irs.KeyValue{Key: "NetworkInterfaceId", Value: *allocRes.NetworkInterfaceId}) // AWS:연결된 Nic
	// 	keyValueList = append(keyValueList, irs.KeyValue{Key: "NetworkInterfaceOwnerId", Value: *allocRes.NetworkInterfaceOwnerId})
	// 	keyValueList = append(keyValueList, irs.KeyValue{Key: "PrivateIpAddress", Value: *allocRes.PrivateIpAddress})
	// }

	//Name 태그 설정
	// for _, t := range allocRes.Tags {
	// 	if *t.Key == "Name" {
	// 		publicIPInfo.Name = *t.Value
	// 		cblogger.Debug("명칭 : ", publicIPInfo.Name)
	// 		break
	// 	}
	// }

	return publicIPInfo
}

//@TODO : 2차 정책에 의해 IP에서 할당ID 기반으로 변경함.
func (publicIpHandler *AlibabaPublicIPHandler) GetPublicIP(publicIPID string) (irs.PublicIPInfo, error) {
	cblogger.Infof("get publicIPID : [%s]", publicIPID)

	var publicIPInfo irs.PublicIPInfo

	request := vpc.CreateDescribeEipAddressesRequest()
	request.Scheme = "https"

	request.AllocationId = publicIPID

	// Make the API request to EC2 filtering for the addresses in the account's VPC.
	result, err := publicIpHandler.Client.DescribeEipAddresses(request)
	if err != nil {
		cblogger.Errorf("Unable to get elastic IP address, %v", err)
		return irs.PublicIPInfo{}, err
	}

	// Printout the IP addresses if there are any.
	if reflect.ValueOf(result.EipAddresses).IsNil() {
		//if len(result.EipAddresses) == 0 {
		//cblogger.Infof("No elastic IPs for %s region\n", publicIpHandler.Region)
		cblogger.Errorf("Not found Elastic IP Information - Request allocation-id : [%s]", publicIPID)
		return irs.PublicIPInfo{}, errors.New("PublicIP NotFound")

	} else {
		cblogger.Info("Elastic IPs")
		for _, addr := range result.EipAddresses.EipAddress {
			publicIPInfo = extractPublicIpDescribeInfo(&addr)
		}
	}

	return publicIPInfo, nil
}

// Public IP를 완전히 제거 함.(AWS Pool로 되돌려 보냄)
func (publicIpHandler *AlibabaPublicIPHandler) DeletePublicIP(allocationId string) (bool, error) {
	cblogger.Infof("delete allocationId : [%s]", allocationId)

	request := vpc.CreateReleaseEipAddressRequest()
	request.Scheme = "https"

	request.AllocationId = allocationId

	result, err := publicIpHandler.Client.ReleaseEipAddress(request)
	if err != nil {
		// if aerr, ok := err.(errors.Error); ok {
		// 	switch aerr.Code() {
		// 	default:
		// 		cblogger.Errorf(aerr.Error())
		// 	}
		// } else {
		// 	// Print the error, cast err to awserr.Error to get the Code and
		// 	// Message from an error.
		// 	cblogger.Errorf(err.Error())
		// }
		cblogger.Errorf("Unable to release elastic IP address, %v", err)
		return false, err
	}

	cblogger.Info(result)
	return true, nil
}

//@TODO : 공통 I/F에 함수 추가해야 함. - ECS 인스턴스와의 연결을 생성하는 AssociatePublicIP
// publicIPID는 AssociationId임.
func (publicIPHandler *AlibabaPublicIPHandler) AssociatePublicIP(serverID string, publicIPID string) (bool, error) {
	cblogger.Infof("serverID : [%s], publicIPID : [%s]", serverID, publicIPID)

	request := vpc.CreateAssociateEipAddressRequest()
	request.Scheme = "https"

	request.InstanceId = serverID
	request.AllocationId = publicIPID

	result, err := publicIPHandler.Client.AssociateEipAddress(request)
	if err != nil {
		// if aerr, ok := err.(*errors.Error); ok {
		// 	switch aerr.Code() {
		// 	default:
		// 		cblogger.Errorf(aerr.Error())
		// 	}
		// } else {
		// 	// Print the error, cast err to awserr.Error to get the Code and
		// 	// Message from an error.
		// 	cblogger.Errorf(err.Error())
		// }
		cblogger.Errorf("Unable to Associate elastic IP address, %v", err)
		return false, err
	}

	cblogger.Info(result)
	return true, nil
}

//@TODO : 공통 I/F에 함수 추가해야 함. - ECS 인스턴스와의 연결만 해제하는 UnassociatePublicIP
// publicIPID는 AssociationId임.
func (publicIpHandler *AlibabaPublicIPHandler) UnassociatePublicIP(serverID string, publicIPID string) (bool, error) {
	cblogger.Infof("serverID : [%s], publicIPID : [%s]", serverID, publicIPID)

	request := vpc.CreateUnassociateEipAddressRequest()
	request.Scheme = "https"

	request.InstanceId = serverID
	request.AllocationId = publicIPID

	result, err := publicIpHandler.Client.UnassociateEipAddress(request)
	if err != nil {
		cblogger.Errorf("Unable to Unassociate elastic IP address, %v", err)
		return false, err
	}

	cblogger.Info(result)
	return true, nil
}
