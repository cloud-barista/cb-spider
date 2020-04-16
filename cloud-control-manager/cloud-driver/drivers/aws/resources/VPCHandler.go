// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by devunet@mz.co.kr

package resources

import (
	"errors"
	"reflect"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type AwsVPCHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

func (VPCHandler *AwsVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	cblogger.Info(vpcReqInfo)

	input := &ec2.CreateVpcInput{
		CidrBlock: aws.String(vpcReqInfo.IPv4_CIDR),
	}

	spew.Dump(input)
	result, err := VPCHandler.Client.CreateVpc(input)
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
		return irs.VPCInfo{}, err
	}

	cblogger.Info(result)
	spew.Dump(result)
	retVpcInfo := ExtractVpcDescribeInfo(result.Vpc)
	retVpcInfo.IId.NameId = vpcReqInfo.IId.NameId // NameId는 요청 받은 값으로 리턴해야 함.

	//IGW Name Tag 설정
	if SetNameTag(VPCHandler.Client, *result.Vpc.VpcId, vpcReqInfo.IId.NameId) {
		cblogger.Infof("VPC에 %s Name 설정 성공", vpcReqInfo.IId.NameId)
	} else {
		cblogger.Errorf("VPC에 %s Name 설정 실패", vpcReqInfo.IId.NameId)
	}

	//====================================
	// PublicIP 할당을 위해 IGW 생성및 연결
	//====================================
	//IGW 생성
	resultIGW, errIGW := VPCHandler.Client.CreateInternetGateway(&ec2.CreateInternetGatewayInput{})
	if errIGW != nil {
		if aerr, ok := errIGW.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(errIGW.Error())
		}
		return retVpcInfo, errIGW
	}

	cblogger.Info(resultIGW)

	//IGW Name Tag 설정
	if SetNameTag(VPCHandler.Client, *resultIGW.InternetGateway.InternetGatewayId, vpcReqInfo.IId.NameId) {
		cblogger.Infof("IGW에 %s Name 설정 성공", vpcReqInfo.IId.NameId)
	} else {
		cblogger.Errorf("IGW에 %s Name 설정 실패", vpcReqInfo.IId.NameId)
	}

	// VPC에 IGW연결
	inputIGW := &ec2.AttachInternetGatewayInput{
		InternetGatewayId: aws.String(*resultIGW.InternetGateway.InternetGatewayId),
		VpcId:             aws.String(retVpcInfo.IId.SystemId),
	}

	resultIGWAttach, errIGWAttach := VPCHandler.Client.AttachInternetGateway(inputIGW)
	if err != nil {
		if aerr, ok := errIGWAttach.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(errIGWAttach.Error())
		}
		return retVpcInfo, errIGWAttach
	}

	cblogger.Info(resultIGWAttach)

	// 생성된 VPC의 기본 라우팅 테이블에 IGW 라우팅 정보 추가
	errRoute := VPCHandler.CreateRouteIGW(retVpcInfo.IId.SystemId, *resultIGW.InternetGateway.InternetGatewayId)
	if errRoute != nil {
		return retVpcInfo, errRoute
	}

	//==========================
	// Subnet 생성
	//==========================
	VPCHandler.CreateSubnet(retVpcInfo.IId.SystemId, vpcReqInfo.SubnetInfoList[0])
	//GetSubnet
	return retVpcInfo, nil
}

// 생성된 VPC의 라우팅 테이블에 IGW(Internet Gateway) 라우팅 정보를 생성함 (AWS 콘솔의 라우팅 테이블의 [라우팅] Tab 처리)
func (VPCHandler *AwsVPCHandler) CreateRouteIGW(vpcId string, igwId string) error {
	cblogger.Infof("VPC ID : [%s] / IGW ID : [%s]", vpcId, igwId)
	routeTableId, errRoute := VPCHandler.GetDefaultRouteTable(vpcId)
	if errRoute != nil {
		return errRoute
	}

	cblogger.Infof("RouteTable[%s]에 IGW[%s]에 대한 라우팅(0.0.0.0/0) 정보를 추가 합니다.", routeTableId, igwId)
	input := &ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            aws.String(igwId),
		RouteTableId:         aws.String(routeTableId),
	}

	result, err := VPCHandler.Client.CreateRoute(input)
	if err != nil {
		cblogger.Errorf("RouteTable[%s]에 IGW[%s]에 대한 라우팅(0.0.0.0/0) 정보 추가 실패", routeTableId, igwId)
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
		return err
	}
	cblogger.Infof("RouteTable[%s]에 IGW[%s]에 대한 라우팅(0.0.0.0/0) 정보를 추가 완료", routeTableId, igwId)

	cblogger.Info(result)
	spew.Dump(result)
	return nil
}

// 자동 생성된 VPC의 기본 라우팅 테이블 정보를 찾음
func (VPCHandler *AwsVPCHandler) GetDefaultRouteTable(vpcId string) (string, error) {
	input := &ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []*string{
					aws.String(vpcId),
				},
			},
		},
	}

	result, err := VPCHandler.Client.DescribeRouteTables(input)
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

	cblogger.Info(result)
	spew.Dump(result)

	if len(result.RouteTables) > 0 {
		routeTableId := *result.RouteTables[0].RouteTableId
		cblogger.Infof("라우팅 테이블 ID 찾음 : [%s]", routeTableId)
		return routeTableId, nil
	} else {
		return "", errors.New("VPC에 할당된 라우팅 테이블 ID를 찾을 수 없습니다.")
	}
}

func (VPCHandler *AwsVPCHandler) CreateSubnet(vpcId string, reqSubnetInfo irs.SubnetInfo) (irs.SubnetInfo, error) {
	cblogger.Info(reqSubnetInfo)

	vpcInfo, errVpcInfo := VPCHandler.GetSubnet(reqSubnetInfo.IId.SystemId)
	if errVpcInfo == nil {
		cblogger.Error("이미 Default Subnet이 존재하기 때문에 생성하지 않고 기존 정보와 함께 에러를 리턴함.")
		cblogger.Info(vpcInfo)
		return vpcInfo, errors.New("InvalidVNetwork.Duplicate: The CBVnetwork '" + GetCBDefaultSubnetName() + "' already exists.")
	}

	//서브넷 생성
	input := &ec2.CreateSubnetInput{
		CidrBlock: aws.String(reqSubnetInfo.IPv4_CIDR),
		VpcId:     aws.String(vpcId),
	}

	cblogger.Info(input)
	result, err := VPCHandler.Client.CreateSubnet(input)
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
		return irs.SubnetInfo{}, err
	}
	cblogger.Info(result)
	spew.Dump(result)

	//vNetworkInfo := irs.VNetworkInfo{}
	vNetworkInfo := ExtractSubnetDescribeInfo(result.Subnet)

	//Subnet Name 태깅
	if SetNameTag(VPCHandler.Client, *result.Subnet.SubnetId, reqSubnetInfo.IId.NameId) {
		cblogger.Infof("Subnet에 %s Name 설정 성공", reqSubnetInfo.IId.NameId)
	} else {
		cblogger.Errorf("Subnet에 %s Name 설정 실패", reqSubnetInfo.IId.NameId)
	}

	vNetworkInfo.IId.NameId = reqSubnetInfo.IId.NameId

	// VPC의 라우팅 테이블에 생성된 Subnet 정보를 추가 함.
	errSubnetRoute := VPCHandler.AssociateRouteTable(vpcId, vNetworkInfo.IId.SystemId)
	if errSubnetRoute != nil {
	} else {
		return vNetworkInfo, errSubnetRoute
	}

	return vNetworkInfo, nil
}

// VPC의 라우팅 테이블에 생성된 Subnet을 연결 함.
func (VPCHandler *AwsVPCHandler) AssociateRouteTable(vpcId string, subnetId string) error {
	routeTableId, errRoute := VPCHandler.GetDefaultRouteTable(vpcId)
	if errRoute != nil {
		return errRoute
	}

	input := &ec2.AssociateRouteTableInput{
		RouteTableId: aws.String(routeTableId),
		SubnetId:     aws.String(subnetId),
	}

	result, err := VPCHandler.Client.AssociateRouteTable(input)
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
		return err
	}

	cblogger.Info(result)
	spew.Dump(result)
	return nil
}

func (VPCHandler *AwsVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	cblogger.Debug("Start")
	result, err := VPCHandler.Client.DescribeVpcs(&ec2.DescribeVpcsInput{})
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

	var vNetworkInfoList []*irs.VPCInfo
	for _, curVpc := range result.Vpcs {
		cblogger.Infof("[%s] VPC 정보 조회", *curVpc.VpcId)
		vNetworkInfo, vpcErr := VPCHandler.GetVPC(irs.IID{SystemId: *curVpc.VpcId})
		if vpcErr != nil {
			return nil, vpcErr
		}
		vNetworkInfoList = append(vNetworkInfoList, &vNetworkInfo)
	}

	spew.Dump(vNetworkInfoList)
	return vNetworkInfoList, nil
}

func (VPCHandler *AwsVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	cblogger.Info("VPC IID : ", vpcIID.SystemId)

	input := &ec2.DescribeVpcsInput{
		VpcIds: []*string{
			aws.String(vpcIID.SystemId),
		},
	}

	result, err := VPCHandler.Client.DescribeVpcs(input)
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
		return irs.VPCInfo{}, err
	}

	cblogger.Info(result)
	//spew.Dump(result)

	if reflect.ValueOf(result.Vpcs).IsNil() {
		return irs.VPCInfo{}, nil
	}

	var errSubnet error
	awsVpcInfo := ExtractVpcDescribeInfo(result.Vpcs[0])
	awsVpcInfo.SubnetInfoList, errSubnet = VPCHandler.ListSubnet(vpcIID.SystemId)
	if errSubnet != nil {
		return awsVpcInfo, errSubnet
	}

	return awsVpcInfo, nil
}

/*
type VPCInfo struct {
	IId   IID       // {NameId, SystemId}
	IPv4_CIDR string
	SubnetInfoList []SubnetInfo

	KeyValueList []KeyValue
}
*/
//VPC 정보를 추출함
func ExtractVpcDescribeInfo(vpcInfo *ec2.Vpc) irs.VPCInfo {
	awsVpcInfo := irs.VPCInfo{
		IId:       irs.IID{SystemId: *vpcInfo.VpcId},
		IPv4_CIDR: *vpcInfo.CidrBlock,
		//IsDefault: *vpcInfo.IsDefault,
		//State:     *vpcInfo.State,
	}

	//Name은 Tag의 "Name" 속성에만 저장됨
	//NameId는 전달할 필요가 없음.
	/*
		cblogger.Debug("Name Tag 찾기")
		for _, t := range vpcInfo.Tags {
			if *t.Key == "Name" {
				awsVpcInfo.IId.NameId = *t.Value
				cblogger.Debug("VPC Name : ", awsVpcInfo.IId.NameId)
				break
			}
		}
	*/
	return awsVpcInfo
}

func (VPCHandler *AwsVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	return false, nil
}

//VPC의 하위 서브넷 목록을 조회함.
func (VPCHandler *AwsVPCHandler) ListSubnet(vpcId string) ([]irs.SubnetInfo, error) {
	cblogger.Debug("Start")
	var arrSubnetInfoList []irs.SubnetInfo

	input := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []*string{
					aws.String(vpcId),
				},
			},
		},
	}

	//spew.Dump(input)
	result, err := VPCHandler.Client.DescribeSubnets(input)
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

	spew.Dump(result)
	for _, curSubnet := range result.Subnets {
		cblogger.Infof("[%s] Subnet 정보 조회", *curSubnet.SubnetId)
		arrSubnetInfo := ExtractSubnetDescribeInfo(curSubnet)
		//arrSubnetInfo, errSubnet := VPCHandler.GetSubnet(*curSubnet.SubnetId)
		/*
			if errSubnet != nil {
				return nil, errSubnet
			}
		*/
		//arrSubnetInfoList = append(arrSubnetInfoList, arrSubnetInfo)
		arrSubnetInfoList = append(arrSubnetInfoList, arrSubnetInfo)
	}

	spew.Dump(arrSubnetInfoList)
	return arrSubnetInfoList, nil
}

func (VPCHandler *AwsVPCHandler) GetSubnet(reqSubnetId string) (irs.SubnetInfo, error) {
	cblogger.Infof("SubnetId : [%s]", reqSubnetId)

	input := &ec2.DescribeSubnetsInput{
		SubnetIds: []*string{
			aws.String(reqSubnetId),
		},
	}

	spew.Dump(input)
	result, err := VPCHandler.Client.DescribeSubnets(input)
	cblogger.Info(result)
	//spew.Dump(result)
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
		return irs.SubnetInfo{}, err
	}

	if !reflect.ValueOf(result.Subnets).IsNil() {
		retSubnetInfo := ExtractSubnetDescribeInfo(result.Subnets[0])
		return retSubnetInfo, nil
	} else {
		return irs.SubnetInfo{}, errors.New("InvalidSubnet.NotFound: The CBVnetwork '" + reqSubnetId + "' does not exist")
	}
}

/*
    IId        IID
    IPv4_CIDR    string
	KeyValueList    []KeyValue
*/

//Subnet 정보를 추출함
func ExtractSubnetDescribeInfo(subnetInfo *ec2.Subnet) irs.SubnetInfo {
	vNetworkInfo := irs.SubnetInfo{
		IId:       irs.IID{SystemId: *subnetInfo.SubnetId},
		IPv4_CIDR: *subnetInfo.CidrBlock,
		//Status:    *subnetInfo.State,
	}

	/*
		cblogger.Debug("Name Tag 찾기")
		for _, t := range subnetInfo.Tags {
			if *t.Key == "Name" {
				vNetworkInfo.IId.NameId = *t.Value
				cblogger.Debug("Subnet Name : ", vNetworkInfo.IId.NameId)
				break
			}
		}
	*/

	keyValueList := []irs.KeyValue{
		{Key: "VpcId", Value: *subnetInfo.VpcId},
		{Key: "MapPublicIpOnLaunch", Value: strconv.FormatBool(*subnetInfo.MapPublicIpOnLaunch)},
		{Key: "AvailableIpAddressCount", Value: strconv.FormatInt(*subnetInfo.AvailableIpAddressCount, 10)},
		{Key: "AvailabilityZone", Value: *subnetInfo.AvailabilityZone},
		{Key: "Status", Value: *subnetInfo.State},
	}
	vNetworkInfo.KeyValueList = keyValueList

	return vNetworkInfo
}
