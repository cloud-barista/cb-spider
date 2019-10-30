// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by powerkim@etri.re.kr, 2019.06.

package resources

//@TODO : Default VPC & Default Subnet 처리해야 함.
import (
	"errors"
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

//https://amzn.to/2L0lfQS
type AwsVNicHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

//@TODO : 퍼블릭IP(EIP)는 이 곳이 아닌 VM생성 시 처리함. 이곳에서 처리해야 하면 구현해야 함.
func (vNicHandler *AwsVNicHandler) CreateVNic(vNicReqInfo irs.VNicReqInfo) (irs.VNicInfo, error) {

	//Nic 생성을 위해 VPC & Subnet 정보를 조회하고 없을 경우 VPC & Subnet을 자동으로 생성함.
	vNetworkHandler := AwsVNetworkHandler{
		//Region: vNicHandler.Region,
		Client: vNicHandler.Client,
	}
	cblogger.Debug(vNetworkHandler)
	//vNetworkHandler.FindOrCreateMcloudBaristaDefaultVPC(irs.VNetworkReqInfo{})
	cblogger.Info("Default Subnet 정보를 찾기 위해 CBNetwork 정보 조회")
	awsCBNetworkInfo, errAutoCBNetInfo := vNetworkHandler.GetAutoCBNetworkInfo()
	if errAutoCBNetInfo != nil || awsCBNetworkInfo.VpcId == "" {
		return irs.VNicInfo{}, nil
	}

	//기존에 생성된 vNic이 있는지 체크
	cblogger.Infof("[%s]VPC안에 [%s]Name으로 생성된 vNic이있는지 체크", awsCBNetworkInfo.VpcId, vNicReqInfo.Name)
	resultvNicExist, errChkvNicExist := vNicHandler.GetVNicByName(vNicReqInfo.Name, awsCBNetworkInfo.VpcId)
	if errChkvNicExist == nil {
		return resultvNicExist, errors.New(vNicReqInfo.Name + " vNic이 이미 존재합니다.")
	}

	cblogger.Info("신규 vNic 생성 시작")
	input := &ec2.CreateNetworkInterfaceInput{
		Description: aws.String(vNicReqInfo.Name),
		//PrivateIpAddress: aws.String("10.0.2.17"),
		//SubnetId: aws.String("subnet-0a25f65671fa64155"),
		SubnetId: aws.String(awsCBNetworkInfo.SubnetId),
		Groups:   aws.StringSlice(vNicReqInfo.SecurityGroupIds),
	}

	/*
		//보안그룹 처리
		securityGroupIds := []*string{}
		for _, id := range vNicReqInfo.SecurityGroupIds {
			securityGroupIds = append(securityGroupIds, aws.String(id))
		}
		input.Groups = securityGroupIds
	*/

	cblogger.Info(input)
	//spew.Dump(input)
	result, err := vNicHandler.Client.CreateNetworkInterface(input)
	//spew.Dump(result)
	cblogger.Info(result)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return irs.VNicInfo{}, err
	}

	//Name 태그 에러 대비 먼저 추출된 정보 저장
	vNicInfo := ExtractVNicDescribeInfo(result.NetworkInterface)

	//VNic Name 태깅
	tagInput := &ec2.CreateTagsInput{
		Resources: []*string{
			aws.String(*result.NetworkInterface.NetworkInterfaceId),
		},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(vNicReqInfo.Name),
			},
		},
	}
	//spew.Dump(tagInput)
	cblogger.Info(tagInput)
	_, errTag := vNicHandler.Client.CreateTags(tagInput)
	if errTag != nil {
		cblogger.Errorf("[%s]VNic 생성 성공 후 Name 태깅 실패", vNicInfo.Id)
		//@TODO : Name 태깅 실패시 생성된 Nic을 삭제할지 Name 태깅을 하라고 전달할지 결정해야 함. - 일단, 바깥에서 처리 가능하도록 생성된 VPC 정보는 전달 함.
		if aerr, ok := errTag.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(errTag.Error())
		}
		return vNicInfo, errTag
	}

	//Name 설정 여부 등 가장 최신 정보로 리턴
	vNicInfo, _ = vNicHandler.GetVNic(vNicInfo.Id)
	return vNicInfo, nil
}

func (vNicHandler *AwsVNicHandler) ListVNic() ([]*irs.VNicInfo, error) {
	cblogger.Info("Start")

	//VPC ID 조회
	vNetworkHandler := AwsVNetworkHandler{Client: vNicHandler.Client}
	vpcId := vNetworkHandler.GetMcloudBaristaDefaultVpcId()
	if vpcId == "" {
		return nil, nil
	}

	input := &ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: []*string{
			nil,
		},
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: aws.StringSlice([]string{vpcId}),
			},
		},
	}

	result, err := vNicHandler.Client.DescribeNetworkInterfaces(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return nil, err
	}

	var vNicInfoList []*irs.VNicInfo
	for _, cur := range result.NetworkInterfaces {
		cblogger.Infof("[%s] vNic 정보 처리", *cur.NetworkInterfaceId)
		vNicInfo := ExtractVNicDescribeInfo(cur)
		vNicInfoList = append(vNicInfoList, &vNicInfo)
	}

	return vNicInfoList, nil
}

//VNic 정보를 추출함
func ExtractVNicDescribeInfo(netIf *ec2.NetworkInterface) irs.VNicInfo {
	spew.Dump(netIf)
	vNicInfo := irs.VNicInfo{
		Id:     *netIf.NetworkInterfaceId,
		Status: *netIf.Status,
	}

	keyValueList := []irs.KeyValue{
		{Key: "VpcId", Value: *netIf.VpcId},
		{Key: "SubnetId", Value: *netIf.SubnetId},
		{Key: "OwnerId", Value: *netIf.OwnerId},
		{Key: "PrivateIpAddress", Value: *netIf.PrivateIpAddress},
		{Key: "InterfaceType", Value: *netIf.InterfaceType},
		{Key: "AvailabilityZone", Value: *netIf.AvailabilityZone},
	}

	if !reflect.ValueOf(netIf.MacAddress).IsNil() {
		vNicInfo.MacAddress = *netIf.MacAddress
	}

	// 할당된 VM 정보 조회
	if !reflect.ValueOf(netIf.Attachment).IsNil() {
		//인스턴스에 할당된 경우
		if !reflect.ValueOf(netIf.Attachment.InstanceId).IsNil() {
			vNicInfo.OwnedVMID = *netIf.Attachment.InstanceId
			keyValueList = append(keyValueList, irs.KeyValue{Key: "InstanceOwnerId", Value: *netIf.Attachment.InstanceOwnerId})

			keyValueList = append(keyValueList, irs.KeyValue{Key: "AttachTime", Value: netIf.Attachment.AttachTime.String()})
		}
	}

	//보안그룹
	if !reflect.ValueOf(netIf.Groups).IsNil() {
		for _, t := range netIf.Groups {
			vNicInfo.SecurityGroupIds = append(vNicInfo.SecurityGroupIds, *t.GroupId)
		}

	}

	//Name은 Tag의 "Name" 속성에만 저장됨
	cblogger.Debug("Name Tag 찾기")
	for _, t := range netIf.TagSet {
		if *t.Key == "Name" {
			vNicInfo.Name = *t.Value
			cblogger.Debug("vNic 명칭 : ", vNicInfo.Name)
			break
		}
	}

	if !reflect.ValueOf(netIf.Association).IsNil() {
		vNicInfo.PublicIP = *netIf.Association.PublicIp

		//keyValueList = append(keyValueList, irs.KeyValue{Key: "AllocationId", Value: *netIf.Association.AllocationId})
		//keyValueList = append(keyValueList, irs.KeyValue{Key: "AssociationId", Value: *netIf.Association.AssociationId})
		keyValueList = append(keyValueList, irs.KeyValue{Key: "IpOwnerId", Value: *netIf.Association.IpOwnerId})
	}

	// 일부 이미지들은 아래 정보가 없어서 예외 처리 함.
	if !reflect.ValueOf(netIf.Description).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "Description", Value: *netIf.Description})
	}

	vNicInfo.KeyValueList = keyValueList

	return vNicInfo
}

func (vNicHandler *AwsVNicHandler) GetVNicByName(vNicName string, vpcId string) (irs.VNicInfo, error) {
	cblogger.Infof("vNicName : [%s] / vpcId : [%s]", vNicName, vpcId)
	input := &ec2.DescribeNetworkInterfacesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: aws.StringSlice([]string{vpcId}),
			},
			{
				Name: aws.String("tag:Name"),
				Values: []*string{
					aws.String(vNicName),
				},
			},
		},
	}

	result, err := vNicHandler.Client.DescribeNetworkInterfaces(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return irs.VNicInfo{}, err
	}

	if len(result.NetworkInterfaces) > 0 {
		vNicInfo := ExtractVNicDescribeInfo(result.NetworkInterfaces[0])
		return vNicInfo, nil
	} else {
		return irs.VNicInfo{}, errors.New("정보를 찾을 수 없습니다.")
	}
}

func (vNicHandler *AwsVNicHandler) GetVNic(vNicID string) (irs.VNicInfo, error) {
	cblogger.Info("vNicID : ", vNicID)
	input := &ec2.DescribeNetworkInterfacesInput{
		NetworkInterfaceIds: []*string{
			aws.String(vNicID),
		},
	}

	result, err := vNicHandler.Client.DescribeNetworkInterfaces(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return irs.VNicInfo{}, err
	}

	if len(result.NetworkInterfaces) > 0 {
		vNicInfo := ExtractVNicDescribeInfo(result.NetworkInterfaces[0])
		return vNicInfo, nil
	} else {
		return irs.VNicInfo{}, errors.New("정보를 찾을 수 없습니다.")
	}
}

func (vNicHandler *AwsVNicHandler) DeleteVNic(vNicID string) (bool, error) {
	cblogger.Info("vNicID : ", vNicID)
	input := &ec2.DeleteNetworkInterfaceInput{
		NetworkInterfaceId: aws.String(vNicID),
	}

	_, err := vNicHandler.Client.DeleteNetworkInterface(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return false, err
	}

	/****
		//@TODO : Nic외에도 보안 그룹 등도 함께 체크해야하므로 명시적으로 Subnet 삭제 외에는 삭제되지 않도록 주석 처리해 놓지만 나중에 처리해야 함.

		//마지막 Nic이 삭제되면 VPC & Subnet도 자동 삭제함.
		vNicList, err := vNicHandler.ListVNic()
		//생성된 Nic이 없는 경우 Auto CB Network 삭제
		if len(vNicList) < 1 && err == nil {
			cblogger.Info("마지막 Nic이 삭제되었으므로 자동생성된 VPC와 Subnet을 제거함.")
			//VPC ID 조회
			vNetworkHandler := AwsVNetworkHandler{Client: vNicHandler.Client}

			subnetId := vNetworkHandler.GetMcloudBaristaDefaultSubnetId()
			if subnetId != "" {
				_, errDel := vNetworkHandler.DeleteVNetwork(subnetId)
				if errDel != nil {
					cblogger.Error("CB Default Virtual Network 자동 제거 실패")
					cblogger.Error(errDel)
				}
			}
		}
	****/
	return true, nil
}
