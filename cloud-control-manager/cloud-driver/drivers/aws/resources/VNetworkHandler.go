// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by devunet@mz.co.kr

//VNetworkHandler는 서브넷을 처리하는 핸들러임.
//VPC & Subnet 처리
//Ver2 - <CB-Virtual Network> 개발 방안에 맞게 AWS의 VPC기능은 외부에 숨기고 Subnet을 Main으로 함.

package resources

import (
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type AwsVNetworkHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

type AwsVpcReqInfo struct {
	Name      string
	CidrBlock string // AWS
}

type AwsVpcInfo struct {
	Name      string
	Id        string
	CidrBlock string // AWS
	IsDefault bool   // AWS
	State     string // AWS
}

func (vNetworkHandler *AwsVNetworkHandler) ListVpc() ([]*AwsVpcInfo, error) {
	cblogger.Debug("Start")
	result, err := vNetworkHandler.Client.DescribeVpcs(&ec2.DescribeVpcsInput{})
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

	var vNetworkInfoList []*AwsVpcInfo
	for _, curVpc := range result.Vpcs {
		cblogger.Infof("[%s] VPC 정보 조회", *curVpc.VpcId)
		vNetworkInfo := ExtractVpcDescribeInfo(curVpc)
		vNetworkInfoList = append(vNetworkInfoList, &vNetworkInfo)
	}

	spew.Dump(vNetworkInfoList)
	return vNetworkInfoList, nil
}

//@TODO : 여러 VPC에 속한 Subnet 목록을 조회하게되는데... CB-Vnet의 서브넷만 조회해야할지 결정이 필요함. 현재는 1차 버전 문맥상 CB-Vnet으로 내부적으로 제한해서 구현했음.
func (vNetworkHandler *AwsVNetworkHandler) ListVNetwork() ([]*irs.VNetworkInfo, error) {
	cblogger.Debug("Start")
	var vNetworkInfoList []*irs.VNetworkInfo

	cblogger.Infof("조회 범위를 CBDefaultVPC[%s]로 제한합니다.", irs.GetCBDefaultVNetName())
	//defaultVpcInfo := irs.VNetworkReqInfo{}
	VpcId, errVpc := vNetworkHandler.FindOrCreateMcloudBaristaDefaultVPC(irs.VNetworkReqInfo{})
	cblogger.Info("CBDefaultVPC 조회 결과 : ", VpcId)
	if errVpc != nil {
		return nil, errVpc
	}

	//생성된 CB Default Virtual Network가 없는 경우 nil 리턴
	if VpcId == "" {
		return vNetworkInfoList, nil
	}

	//기본 CBVPC에 속한 서브넷만 조회
	input := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []*string{
					aws.String(VpcId),
				},
			},
		},
	}

	result, err := vNetworkHandler.Client.DescribeSubnets(input)
	//result, err := vNetworkHandler.Client.DescribeSubnets(&ec2.DescribeSubnetsInput{})	//전체 서브넷 조회
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

	for _, curSubnet := range result.Subnets {
		cblogger.Infof("[%s] Subnet 정보 조회", *curSubnet.SubnetId)
		vNetworkInfo := ExtractSubnetDescribeInfo(curSubnet)
		vNetworkInfoList = append(vNetworkInfoList, &vNetworkInfo)
	}

	spew.Dump(vNetworkInfoList)
	return vNetworkInfoList, nil
}

//@TODO : ListVNetwork()에서 호출되는 경우도 있기 때문에 필요하면 VPC조회와 생성을 별도의 Func으로 분리해야함.(일단은 큰 문제는 없어서 놔둠)
//CB Default Virtual Network가 존재하지 않으면 생성하며, 존재하는 경우 Vpc ID를 리턴 함.
func (vNetworkHandler *AwsVNetworkHandler) FindOrCreateMcloudBaristaDefaultVPC(vNetworkReqInfo irs.VNetworkReqInfo) (string, error) {
	cblogger.Info(vNetworkReqInfo)

	awsVpcInfo, err := vNetworkHandler.GetVpc(irs.GetCBDefaultVNetName())
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
		return "", err
	}

	if awsVpcInfo.Id != "" {
		return awsVpcInfo.Id, nil
	} else {
		cblogger.Infof("기본 VPC[%s]가 없어서 Subnet 요청 정보를 기반으로 /16 범위의 VPC를 생성합니다.", irs.GetCBDefaultVNetName())
		cblogger.Info("Subnet CIDR 요청 정보 : ", vNetworkReqInfo.CidrBlock)
		if vNetworkReqInfo.CidrBlock == "" {
			//VPC가 없는 최초 상태에서 List()에서 호출되었을 수 있기 때문에 에러 처리는 하지 않고 nil을 전달함.
			cblogger.Infof("요청 정보에 CIDR 정보가 없어서 Default VPC[%s]를 생성하지 않음", irs.GetCBDefaultVNetName())
			return "", nil
		}

		reqCidr := strings.Split(vNetworkReqInfo.CidrBlock, ".")
		//cblogger.Info("CIDR 추출 정보 : ", reqCidr[0])
		VpcCidrBlock := reqCidr[0] + "." + reqCidr[1] + ".0.0/16"
		cblogger.Info("신규 VPC에 사용할 CIDR 정보 : ", VpcCidrBlock)

		awsVpcReqInfo := AwsVpcReqInfo{
			Name:      irs.GetCBDefaultVNetName(),
			CidrBlock: VpcCidrBlock,
		}

		result, errVpc := vNetworkHandler.CreateVpc(awsVpcReqInfo)
		if errVpc != nil {
			cblogger.Error(errVpc)
			return "", errVpc
		}
		cblogger.Infof("CB Default VPC[%s] 생성 완료 - CIDR : [%s]", irs.GetCBDefaultVNetName(), result.CidrBlock)
		cblogger.Info(result)
		spew.Dump(result)

		return result.Id, nil
	}
}

func (vNetworkHandler *AwsVNetworkHandler) GetVpc(vpcName string) (AwsVpcInfo, error) {
	cblogger.Info("VPC Name : ", vpcName)

	input := &ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("tag:Name"), // vpc-id , dhcp-options-id
				Values: []*string{
					aws.String(vpcName),
				},
			},
		},
	}

	result, err := vNetworkHandler.Client.DescribeVpcs(input)
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
		return AwsVpcInfo{}, err
	}

	cblogger.Info(result)
	//spew.Dump(result)

	if !reflect.ValueOf(result.Vpcs).IsNil() {
		awsVpcInfo := ExtractVpcDescribeInfo(result.Vpcs[0])
		return awsVpcInfo, nil
	} else {
		return AwsVpcInfo{}, nil
	}

}

func (vNetworkHandler *AwsVNetworkHandler) CreateVpc(awsVpcReqInfo AwsVpcReqInfo) (AwsVpcInfo, error) {
	cblogger.Info(awsVpcReqInfo)

	input := &ec2.CreateVpcInput{
		CidrBlock: aws.String(awsVpcReqInfo.CidrBlock),
	}

	spew.Dump(input)
	result, err := vNetworkHandler.Client.CreateVpc(input)
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
		return AwsVpcInfo{}, err
	}

	cblogger.Info(result)
	spew.Dump(result)
	awsVpcInfo := ExtractVpcDescribeInfo(result.Vpc)

	//VPC Name 태깅
	tagInput := &ec2.CreateTagsInput{
		Resources: []*string{
			aws.String(*result.Vpc.VpcId),
		},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(awsVpcReqInfo.Name),
			},
		},
	}
	spew.Dump(tagInput)

	_, errTag := vNetworkHandler.Client.CreateTags(tagInput)
	if errTag != nil {
		//@TODO : Name 태깅 실패시 생성된 VPC를 삭제할지 Name 태깅을 하라고 전달할지 결정해야 함. - 일단, 바깥에서 처리 가능하도록 생성된 VPC 정보는 전달 함.
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
		return awsVpcInfo, errTag
	}

	awsVpcInfo.Name = awsVpcReqInfo.Name

	return awsVpcInfo, nil
}

func (vNetworkHandler *AwsVNetworkHandler) CreateVNetwork(vNetworkReqInfo irs.VNetworkReqInfo) (irs.VNetworkInfo, error) {
	cblogger.Info(vNetworkReqInfo)

	//최대 5개의 VPC 생성 제한이 있기 때문에 기본VPC 조회시 에러 처리를 해줌.
	VpcId, errVpc := vNetworkHandler.FindOrCreateMcloudBaristaDefaultVPC(vNetworkReqInfo)
	cblogger.Info("CBDefaultVPC 조회 결과 : ", VpcId)
	if errVpc != nil {
		return irs.VNetworkInfo{}, errVpc
	}

	//서브넷 생성
	input := &ec2.CreateSubnetInput{
		CidrBlock: aws.String(vNetworkReqInfo.CidrBlock),
		VpcId:     aws.String(VpcId),
	}

	cblogger.Info(input)
	result, err := vNetworkHandler.Client.CreateSubnet(input)
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
		return irs.VNetworkInfo{}, err
	}
	cblogger.Info(result)
	spew.Dump(result)

	//vNetworkInfo := irs.VNetworkInfo{}
	vNetworkInfo := ExtractSubnetDescribeInfo(result.Subnet)

	//Subnet Name 태깅
	tagInput := &ec2.CreateTagsInput{
		Resources: []*string{
			aws.String(*result.Subnet.SubnetId),
		},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(vNetworkReqInfo.Name),
			},
		},
	}
	spew.Dump(tagInput)

	_, errTag := vNetworkHandler.Client.CreateTags(tagInput)
	if errTag != nil {
		//@TODO : Name 태깅 실패시 생성된 VPC를 삭제할지 Name 태깅을 하라고 전달할지 결정해야 함. - 일단, 바깥에서 처리 가능하도록 생성된 VPC 정보는 전달 함.
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
		return vNetworkInfo, errTag
	}

	vNetworkInfo.Name = vNetworkReqInfo.Name

	return vNetworkInfo, nil
}

func (vNetworkHandler *AwsVNetworkHandler) GetVNetwork(vNetworkID string) (irs.VNetworkInfo, error) {
	cblogger.Info("vNetworkID : [%s]", vNetworkID)

	input := &ec2.DescribeSubnetsInput{
		SubnetIds: []*string{
			aws.String(vNetworkID),
		},
	}

	result, err := vNetworkHandler.Client.DescribeSubnets(input)
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
		return irs.VNetworkInfo{}, err
	}

	cblogger.Info(result)
	//spew.Dump(result)
	if !reflect.ValueOf(result.Subnets).IsNil() {
		vNetworkInfo := ExtractSubnetDescribeInfo(result.Subnets[0])
		return vNetworkInfo, nil
	} else {
		return irs.VNetworkInfo{}, nil
	}
}

//VPC 정보를 추출함
func ExtractVpcDescribeInfo(vpcInfo *ec2.Vpc) AwsVpcInfo {
	awsVpcInfo := AwsVpcInfo{
		Id:        *vpcInfo.VpcId,
		CidrBlock: *vpcInfo.CidrBlock,
		IsDefault: *vpcInfo.IsDefault,
		State:     *vpcInfo.State,
	}

	//Name은 Tag의 "Name" 속성에만 저장됨
	cblogger.Debug("Name Tag 찾기")
	for _, t := range vpcInfo.Tags {
		if *t.Key == "Name" {
			awsVpcInfo.Name = *t.Value
			cblogger.Debug("VPC Name : ", awsVpcInfo.Name)
			break
		}
	}

	return awsVpcInfo
}

//Subnet 정보를 추출함
func ExtractSubnetDescribeInfo(subnetInfo *ec2.Subnet) irs.VNetworkInfo {
	vNetworkInfo := irs.VNetworkInfo{
		SubnetId:  *subnetInfo.SubnetId,
		CidrBlock: *subnetInfo.CidrBlock,
		State:     *subnetInfo.State,

		Id:                      *subnetInfo.VpcId,
		MapPublicIpOnLaunch:     *subnetInfo.MapPublicIpOnLaunch,
		AvailableIpAddressCount: *subnetInfo.AvailableIpAddressCount,
		AvailabilityZone:        *subnetInfo.AvailabilityZone,
	}

	//Name은 Tag의 "Name" 속성에만 저장됨
	cblogger.Debug("Name Tag 찾기")
	for _, t := range subnetInfo.Tags {
		if *t.Key == "Name" {
			vNetworkInfo.Name = *t.Value
			cblogger.Debug("Subnet Name : ", vNetworkInfo.Name)
			break
		}
	}

	return vNetworkInfo
}

func (vNetworkHandler *AwsVNetworkHandler) DeleteVpc(vpcId string) (bool, error) {
	cblogger.Info("vpcId : [%s]", vpcId)
	input := &ec2.DeleteVpcInput{
		VpcId: aws.String(vpcId),
	}

	result, err := vNetworkHandler.Client.DeleteVpc(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			cblogger.Error(err.Error())
		}
		return false, err
	}

	cblogger.Info(result)
	spew.Dump(result)
	return true, nil
}

//서브넷 삭제
//마지막 서브넷인 경우 CB-Default Virtual Network도 함께 제거
func (vNetworkHandler *AwsVNetworkHandler) DeleteVNetwork(vNetworkID string) (bool, error) {
	cblogger.Info("vNetworkID : [%s]", vNetworkID)

	input := &ec2.DeleteSubnetInput{
		SubnetId: aws.String(vNetworkID),
	}

	_, err := vNetworkHandler.Client.DeleteSubnet(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			cblogger.Error(err.Error())
		}
		return false, err
	}

	subnetList, _ := vNetworkHandler.ListVNetwork()

	//서브넷이 존재하는경우 서브넷 삭제 결과 리턴
	if subnetList != nil {
		return true, nil
	} else {
		//서브넷이 없는 경우 기본 CBVPC도 삭제
		VpcId, errVpc := vNetworkHandler.FindOrCreateMcloudBaristaDefaultVPC(irs.VNetworkReqInfo{})
		cblogger.Info("삭제할 CBDefaultVPC 조회 결과 : ", VpcId)
		if errVpc != nil {
			cblogger.Error("삭제할 CBDefaultVPC 조회 실패 : ", errVpc)
			return false, errVpc
		}

		//발생할 경우가 없어 보이지만 삭제할 CB Default VPC가 없으면 종료
		if VpcId == "" {
			cblogger.Error("삭제할 CBDefaultVPC가 존재하지 않음")
			return true, nil
		}

		cblogger.Info("CBDefaultVPC를 삭제 함.")
		delVpc, errDelVpc := vNetworkHandler.DeleteVpc(VpcId)
		if errDelVpc != nil {
			cblogger.Error("CBDefaultVPC 삭제 실패 : ", errDelVpc)
			return false, errDelVpc
		}

		if delVpc {
			cblogger.Info("CBDefaultVPC를 삭제 완료.")
			return true, nil
		} else {
			cblogger.Info("CBDefaultVPC를 삭제 실패.")
			return false, nil //삭제 실패 이유를 모르는 경우
		}

	}

}
