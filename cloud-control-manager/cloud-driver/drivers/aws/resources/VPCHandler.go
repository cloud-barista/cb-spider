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
	"fmt"
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsVPCHandler struct {
	Region     idrv.RegionInfo
	Client     *ec2.EC2
	TagHandler *AwsTagHandler // 2024-07-18 TagHandler add
}

func (VPCHandler *AwsVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	cblogger.Info(vpcReqInfo)

	zoneId := VPCHandler.Region.Zone
	cblogger.Infof("Zone : %s", zoneId)
	if zoneId == "" {
		cblogger.Error("Connection information does not contain Zone information.")
		return irs.VPCInfo{}, errors.New("Connection information does not contain Zone information.")
	}

	// Convert TagList to TagSpecifications
	tagSpecifications, err := ConvertTagListToTagSpecifications("vpc", vpcReqInfo.TagList, vpcReqInfo.IId.NameId)
	if err != nil {
		return irs.VPCInfo{}, fmt.Errorf("failed to convert tag list: %w", err)
	}

	// Create VPC input with tag specifications
	input := &ec2.CreateVpcInput{
		CidrBlock:         aws.String(vpcReqInfo.IPv4_CIDR),
		TagSpecifications: tagSpecifications,
	}

	//cblogger.Debug(input)
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: vpcReqInfo.IId.NameId,
		CloudOSAPI:   "CreateVpc()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := VPCHandler.Client.CreateVpc(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
				callLogInfo.ErrorMSG = aerr.Error()
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
			callLogInfo.ErrorMSG = err.Error()
		}
		callogger.Info(call.String(callLogInfo))
		return irs.VPCInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Info(result)
	//cblogger.Debug(result)
	retVpcInfo := ExtractVpcDescribeInfo(result.Vpc)
	retVpcInfo.IId.NameId = vpcReqInfo.IId.NameId // NameId는 요청 받은 값으로 리턴해야 함.

	/*
		// 2024.07.16 Delete with Tag support
		if len(tagSpecifications) == 0 {
			//IGW Name Tag 설정
			if SetNameTag(VPCHandler.Client, *result.Vpc.VpcId, vpcReqInfo.IId.NameId) {
				cblogger.Infof("set name %s to VPC", vpcReqInfo.IId.NameId)
			} else {
				cblogger.Errorf("set name  %s to VPC failed", vpcReqInfo.IId.NameId)
			}
		}
	*/

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
		cblogger.Infof("set Name %s to IGW", vpcReqInfo.IId.NameId)
	} else {
		cblogger.Errorf("set Name %s to IGW failed", vpcReqInfo.IId.NameId)
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
	//VPCHandler.CreateSubnet(retVpcInfo.IId.SystemId, vpcReqInfo.SubnetInfoList[0])
	var resSubnetList []irs.SubnetInfo
	for _, curSubnet := range vpcReqInfo.SubnetInfoList {
		cblogger.Infof("[%s] Subnet Create", curSubnet.IId.NameId)
		cblogger.Infof("Req Subnet Info [%v]", curSubnet)
		resSubnet, errSubnet := VPCHandler.CreateSubnet(retVpcInfo.IId.SystemId, curSubnet)

		if errSubnet != nil {
			return retVpcInfo, errSubnet
		}
		resSubnetList = append(resSubnetList, resSubnet)
	}
	retVpcInfo.SubnetInfoList = resSubnetList

	retVpcInfo.TagList, _ = VPCHandler.TagHandler.ListTag(irs.VPC, retVpcInfo.IId)
	return retVpcInfo, nil
}

// 생성된 VPC의 라우팅 테이블에 IGW(Internet Gateway) 라우팅 정보를 생성함 (AWS 콘솔의 라우팅 테이블의 [라우팅] Tab 처리)
func (VPCHandler *AwsVPCHandler) CreateRouteIGW(vpcId string, igwId string) error {
	cblogger.Infof("VPC ID : [%s] / IGW ID : [%s]", vpcId, igwId)
	routeTableId, errRoute := VPCHandler.GetDefaultRouteTable(vpcId)
	if errRoute != nil {
		return errRoute
	}

	cblogger.Infof("Adding routing information for IGW [%s] to RouteTable [%s] for destination (0.0.0.0/0).", routeTableId, igwId)
	input := &ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            aws.String(igwId),
		RouteTableId:         aws.String(routeTableId),
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")

	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: igwId,
		CloudOSAPI:   "CreateRoute()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := VPCHandler.Client.CreateRoute(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		cblogger.Errorf("Failed to add routing information for IGW [%s] to RouteTable [%s] for destination (0.0.0.0/0).", routeTableId, igwId)
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
				callLogInfo.ErrorMSG = aerr.Error()
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
			callLogInfo.ErrorMSG = err.Error()
		}
		callogger.Info(call.String(callLogInfo))
		return err
	}
	cblogger.Infof("Added routing information for IGW [%s] to RouteTable [%s] for destination (0.0.0.0/0) successfully.", routeTableId, igwId)
	callogger.Info(call.String(callLogInfo))

	cblogger.Info(result)
	cblogger.Debug(result)
	return nil
}

// https://docs.aws.amazon.com/ko_kr/vpc/latest/userguide/VPC_Route_Tables.html
// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeRouteTables.html
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
	//cblogger.Debug(result)

	if len(result.RouteTables) > 0 {
		routeTableId := *result.RouteTables[0].RouteTableId
		cblogger.Infof("Found routing table ID: [%s]", routeTableId)
		return routeTableId, nil
	} else {
		return "", errors.New("The routing table ID assigned to the VPC could not be found.")
	}
}

func (VPCHandler *AwsVPCHandler) CreateSubnet(vpcId string, reqSubnetInfo irs.SubnetInfo) (irs.SubnetInfo, error) {
	cblogger.Info(reqSubnetInfo)

	zoneId := VPCHandler.Region.Zone

	// subnet에 zone 정보가 있으면 해당 zone 사용. default는 connection의 zone
	if reqSubnetInfo.Zone != "" {
		zoneId = reqSubnetInfo.Zone
	}
	cblogger.Infof("Zone : %s", zoneId)
	if zoneId == "" {
		cblogger.Error("Connection information does not contain Zone information.")
		return irs.SubnetInfo{}, errors.New("Connection information does not contain Zone information.")
	}

	if reqSubnetInfo.IId.SystemId != "" {
		vpcInfo, errVpcInfo := VPCHandler.GetSubnet(reqSubnetInfo.IId.SystemId)
		if errVpcInfo == nil {
			cblogger.Errorf("[%s] subnet already exists. returns an error without creating it", reqSubnetInfo.IId.SystemId)
			cblogger.Info(vpcInfo)
			return vpcInfo, errors.New("InvalidVNetwork.Duplicate: The Subnet '" + reqSubnetInfo.IId.SystemId + "' already exists.")
		}
	}

	// Convert TagList to TagSpecifications
	tagSpecifications, err := ConvertTagListToTagSpecifications("subnet", reqSubnetInfo.TagList, reqSubnetInfo.IId.NameId)
	if err != nil {
		return irs.SubnetInfo{}, fmt.Errorf("failed to convert tag list: %w", err)
	}

	// Create subnet input with tag specifications
	input := &ec2.CreateSubnetInput{
		CidrBlock: aws.String(reqSubnetInfo.IPv4_CIDR),
		VpcId:     aws.String(vpcId),
		//AvailabilityZoneId: aws.String(zoneId),	//use1-az1, use1-az2, use1-az3, use1-az4, use1-az5, use1-az6
		AvailabilityZone:  aws.String(zoneId),
		TagSpecifications: tagSpecifications,
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")

	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   zoneId,
		ResourceType: call.VPCSUBNET,
		ResourceName: reqSubnetInfo.IId.NameId,
		CloudOSAPI:   "CreateSubnet()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	start := call.Start()

	cblogger.Info(input)
	result, err := VPCHandler.Client.CreateSubnet(input)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
				callLogInfo.ErrorMSG = aerr.Error()
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
			callLogInfo.ErrorMSG = err.Error()
		}
		callogger.Info(call.String(callLogInfo))
		return irs.SubnetInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))
	cblogger.Info(result)
	//cblogger.Debug(result)

	//vNetworkInfo := irs.VNetworkInfo{}
	vNetworkInfo := ExtractSubnetDescribeInfo(result.Subnet)
	vNetworkInfo.TagList, _ = VPCHandler.TagHandler.ListTag(irs.SUBNET, vNetworkInfo.IId)

	/*
		//Subnet Name 태깅
		if SetNameTag(VPCHandler.Client, *result.Subnet.SubnetId, reqSubnetInfo.IId.NameId) {
			cblogger.Infof("set %s Name to subnet", reqSubnetInfo.IId.NameId)
		} else {
			cblogger.Errorf("set %s Name to subnet failed", reqSubnetInfo.IId.NameId)
		}
	*/

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

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")

	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: subnetId,
		CloudOSAPI:   "AssociateRouteTable()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := VPCHandler.Client.AssociateRouteTable(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
				callLogInfo.ErrorMSG = aerr.Error()
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
			callLogInfo.ErrorMSG = err.Error()
		}
		callogger.Info(call.String(callLogInfo))
		return err
	}

	callogger.Info(call.String(callLogInfo))
	cblogger.Info(result)
	//cblogger.Debug(result)
	return nil
}

func (VPCHandler *AwsVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	cblogger.Debug("Start")
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: "ListVPC",
		CloudOSAPI:   "DescribeVpcs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := VPCHandler.Client.DescribeVpcs(&ec2.DescribeVpcsInput{})
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
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
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		return nil, err
	}
	callogger.Info(call.String(callLogInfo))

	var vNetworkInfoList []*irs.VPCInfo
	for _, curVpc := range result.Vpcs {
		cblogger.Infof("Retrieve VPC Info [%s] ", *curVpc.VpcId)
		vNetworkInfo, vpcErr := VPCHandler.GetVPC(irs.IID{SystemId: *curVpc.VpcId})
		if vpcErr != nil {
			return nil, vpcErr
		}
		vNetworkInfoList = append(vNetworkInfoList, &vNetworkInfo)
	}

	cblogger.Debug(vNetworkInfoList)
	return vNetworkInfoList, nil
}

func (VPCHandler *AwsVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	cblogger.Info("VPC IID : ", vpcIID.SystemId)

	input := &ec2.DescribeVpcsInput{
		VpcIds: []*string{
			aws.String(vpcIID.SystemId),
		},
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: vpcIID.SystemId,
		CloudOSAPI:   "DescribeVpcs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := VPCHandler.Client.DescribeVpcs(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
				callLogInfo.ErrorMSG = aerr.Error()
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
			callLogInfo.ErrorMSG = err.Error()
		}
		callogger.Info(call.String(callLogInfo))
		return irs.VPCInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Info(result)
	//cblogger.Debug(result)

	if reflect.ValueOf(result.Vpcs).IsNil() {
		return irs.VPCInfo{}, nil
	}

	var errSubnet error
	awsVpcInfo := ExtractVpcDescribeInfo(result.Vpcs[0])
	awsVpcInfo.SubnetInfoList, errSubnet = VPCHandler.ListSubnet(vpcIID.SystemId)
	if errSubnet != nil {
		return awsVpcInfo, errSubnet
	}

	awsVpcInfo.TagList, _ = VPCHandler.TagHandler.ListTag(irs.VPC, awsVpcInfo.IId)
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

	cblogger.Debug("retrieve Name Tag")
	for _, t := range vpcInfo.Tags {
		if *t.Key == "Name" {
			awsVpcInfo.IId.NameId = *t.Value
			cblogger.Debug("VPC Name : ", awsVpcInfo.IId.NameId)
			break
		}
	}

	awsVpcInfo.KeyValueList = irs.StructToKeyValueList(vpcInfo)

	return awsVpcInfo
}

func (VPCHandler *AwsVPCHandler) DeleteSubnet(subnetIID irs.IID) (bool, error) {
	input := &ec2.DeleteSubnetInput{
		SubnetId: aws.String(subnetIID.SystemId),
	}
	cblogger.Info(input)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")

	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: subnetIID.SystemId,
		CloudOSAPI:   "DeleteSubnet()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	start := call.Start()

	_, err := VPCHandler.Client.DeleteSubnet(input)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err) //#577
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
				callLogInfo.ErrorMSG = aerr.Error()
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			cblogger.Error(err.Error())
			callLogInfo.ErrorMSG = err.Error()
		}
		callogger.Info(call.String(callLogInfo))
		return false, err
	}

	callogger.Info(call.String(callLogInfo))
	return true, nil
}

func (VPCHandler *AwsVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	cblogger.Infof("Delete VPC : [%s]", vpcIID.SystemId)

	vpcInfo, errVpcInfo := VPCHandler.GetVPC(vpcIID)
	if errVpcInfo != nil {
		return false, errVpcInfo
	}

	//=================
	// Subnet삭제
	//=================
	for _, curSubnet := range vpcInfo.SubnetInfoList {
		cblogger.Infof("delete [%s] Subnet", curSubnet.IId.SystemId)
		delSubnet, errSubnet := VPCHandler.DeleteSubnet(curSubnet.IId)
		if errSubnet != nil {
			return false, errSubnet
		}

		if delSubnet {
			cblogger.Infof("  ==> [%s] Subnet deleted", curSubnet.IId.SystemId)
		} else {
			cblogger.Errorf("  ==> failed to delete [%s] Subnet", curSubnet.IId.SystemId)
			return false, errors.New("Failed to delete VPC due to Subnet deletion failure.") //삭제 실패 이유를 모르는 경우
		}
	}

	cblogger.Infof("[%s] VPC deleted.", vpcInfo.IId.SystemId)
	cblogger.Info("Starting the removal of IGWs / Routes created for VPC deletion")

	// 라우팅 테이블에 추가한 IGW 라우터를 먼저 삭제함.
	errRoute := VPCHandler.DeleteRouteIGW(vpcInfo.IId.SystemId)
	if errRoute != nil {
		cblogger.Error("Failed to delete the 0.0.0.0/0 IGW router added to the routing table")
		cblogger.Error(errRoute)
		if "InvalidRoute.NotFound" == errRoute.Error() {
			cblogger.Infof("[%s] is considered normal due to Exception #255 and proceeds to the next step.", errRoute)
		} else {
			return false, errRoute
		}
		//} else {
		//	cblogger.Info("라우팅 테이블에 추가한 0.0.0.0/0 IGW 라우터 삭제 완료")
	}

	//VPC에 연결된 모든 IGW를 삭제함. (VPC에 할당된 모든 IGW조회후 삭제)
	errIgw := VPCHandler.DeleteAllIGW(vpcInfo.IId.SystemId)
	if errIgw != nil {
		cblogger.Error("Failed to delete all IGWs : ", errIgw)
	} else {
		cblogger.Info("All IGWs deleted")
	}

	input := &ec2.DeleteVpcInput{
		VpcId: aws.String(vpcInfo.IId.SystemId),
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:    call.AWS,
		RegionZone: VPCHandler.Region.Zone,
		//RegionZone:   curSubnet.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: vpcInfo.IId.SystemId,
		CloudOSAPI:   "DeleteVpc()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := VPCHandler.Client.DeleteVpc(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
				callLogInfo.ErrorMSG = aerr.Error()
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			cblogger.Error(err.Error())
			callLogInfo.ErrorMSG = err.Error()
		}
		callogger.Info(call.String(callLogInfo))
		return false, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Info(result)
	//cblogger.Debug(result)
	return true, nil
}

/*
// VPC에 설정된 0.0.0.0/0 라우터를 제거 함.
func (VPCHandler *AwsVPCHandler) DeleteRouteIGWOld(vpcId string) error {
	cblogger.Infof("VPC ID : [%s]", vpcId)
	routeTableId, errRoute := VPCHandler.GetDefaultRouteTable(vpcId)
	if errRoute != nil {
		return errRoute
	}

	cblogger.Infof("RouteTable[%s]에 할당된 라우팅(0.0.0.0/0) 정보를 삭제합니다.", routeTableId)
	input := &ec2.DeleteRouteInput{
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		RouteTableId:         aws.String(routeTableId),
	}
	cblogger.Info(input)

	//https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DeleteRoute.html
	result, err := VPCHandler.Client.DeleteRoute(input)
	if err != nil {
		cblogger.Errorf("RouteTable[%s]에 대한 라우팅(0.0.0.0/0) 정보 삭제 실패", routeTableId)
		if aerr, ok := err.(awserr.Error); ok {
			//InvalidRoute.NotFound
			cblogger.Errorf("Error Code : [%s] - Error:[%s] - Message:[%s]", aerr.Code(), aerr.Error(), aerr.Message())
			switch aerr.Code() {
			case "InvalidRoute.NotFound": //NotFound에러는 무시하라고 해서 (예외#255)
				return errors.New(aerr.Code())
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
	cblogger.Infof("RouteTable[%s]에 대한 라우팅(0.0.0.0/0) 정보 삭제 완료", routeTableId)

	cblogger.Info(result)
	cblogger.Debug(result)
	cblogger.Info("라우팅 테이블에 추가한 0.0.0.0/0 IGW 라우터 삭제 완료")
	return nil
}
*/

// VPC에 설정된 0.0.0.0/0 라우터를 제거 함.
// #255예외 처리 보완에 따른 라우팅 정보 삭제전 0.0.0.0 조회후 삭제하도록 로직 변경
func (VPCHandler *AwsVPCHandler) DeleteRouteIGW(vpcId string) error {
	cblogger.Infof("VPC ID : [%s]", vpcId)
	routeTableId := ""

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
		return err
	}

	cblogger.Info(result)
	//cblogger.Debug(result)

	if len(result.RouteTables) < 1 {
		return errors.New("The routing table information assigned to the VPC could not be found.")
	}

	routeTableId = *result.RouteTables[0].RouteTableId
	cblogger.Infof("Found the routing table ID : [%s]", routeTableId)

	cblogger.Infof("Querying routing information (0.0.0.0/0) assigned to RouteTable[%s", routeTableId)

	//ec2.Route
	findIgw := false
	for _, curRoute := range result.RouteTables[0].Routes {
		cblogger.Infof("DestinationCidrBlock[%s] Check", *curRoute.DestinationCidrBlock)

		if "0.0.0.0/0" == *curRoute.DestinationCidrBlock {
			cblogger.Infof("===>Found the routing information (0.0.0.0/0) assigned to RouteTable[%s", routeTableId)
			findIgw = true
			break
		}
	}

	if !findIgw {
		cblogger.Infof("there is no routing information (0.0.0.0/0) for the IGW assigned to RouteTable[%s], the route deletion process is stopped", routeTableId)
		return nil
	}

	cblogger.Infof("Deleting the routing information (0.0.0.0/0) assigned to RouteTable[%s]", routeTableId)
	inputDel := &ec2.DeleteRouteInput{
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		RouteTableId:         aws.String(routeTableId),
	}
	cblogger.Info(inputDel)

	//https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DeleteRoute.html
	resultDel, err := VPCHandler.Client.DeleteRoute(inputDel)
	if err != nil {
		cblogger.Errorf("Failed to delete routing information (0.0.0.0/0) for RouteTable[%s", routeTableId)
		if aerr, ok := err.(awserr.Error); ok {
			//InvalidRoute.NotFound
			cblogger.Errorf("Error Code : [%s] - Error:[%s] - Message:[%s]", aerr.Code(), aerr.Error(), aerr.Message())
			switch aerr.Code() {
			case "InvalidRoute.NotFound": //NotFound에러는 무시하라고 해서 (예외#255)
				return errors.New(aerr.Code())
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
	cblogger.Infof("Completed deleting routing information (0.0.0.0/0) for RouteTable[%s]", routeTableId)

	cblogger.Info(resultDel)
	//cblogger.Debug(resultDel)
	cblogger.Info("Completed deleting the 0.0.0.0/0 IGW router added to the routing table")
	return nil
}

// VPC에 연결된 모든 IGW를 삭제함.
func (VPCHandler *AwsVPCHandler) DeleteAllIGW(vpcId string) error {
	input := &ec2.DescribeInternetGatewaysInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("attachment.vpc-id"),
				Values: []*string{
					aws.String(vpcId),
				},
			},
		},
	}

	result, err := VPCHandler.Client.DescribeInternetGateways(input)
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
	cblogger.Debug(result)

	// VPC 삭제를 위해 연결된 모든 IGW 제거
	// 일단, 에러는 무시함.
	for _, curIgw := range result.InternetGateways {
		//IGW 삭제전 연결된 IGW의 연결을 끊어야함.
		VPCHandler.DetachInternetGateway(vpcId, *curIgw.InternetGatewayId)
		//IGW 삭제
		VPCHandler.DeleteIGW(*curIgw.InternetGatewayId)
	}

	return nil
}

// VPC에 연결된 IGW의 연결을 해제함.
func (VPCHandler *AwsVPCHandler) DetachInternetGateway(vpcId string, igwId string) error {
	cblogger.Infof("Disconnected the IGW[%s] connected to VPC[%s]", vpcId, igwId)

	input := &ec2.DetachInternetGatewayInput{
		InternetGatewayId: aws.String(igwId),
		VpcId:             aws.String(vpcId),
	}

	result, err := VPCHandler.Client.DetachInternetGateway(input)
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
	//cblogger.Debug(result)
	return nil
}

// IGW를 삭제 함.
func (VPCHandler *AwsVPCHandler) DeleteIGW(igwId string) error {
	input := &ec2.DeleteInternetGatewayInput{
		InternetGatewayId: aws.String(igwId),
	}

	result, err := VPCHandler.Client.DeleteInternetGateway(input)
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
	//cblogger.Debug(result)
	return nil
}

// VPC의 하위 서브넷 목록을 조회함.
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

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")

	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: "ListSubnet",
		CloudOSAPI:   "DescribeSubnets()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	//cblogger.Debug(input)
	result, err := VPCHandler.Client.DescribeSubnets(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
				callLogInfo.ErrorMSG = aerr.Error()
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
			callLogInfo.ErrorMSG = err.Error()
		}
		callogger.Info(call.String(callLogInfo))
		return nil, err
	}
	callogger.Info(call.String(callLogInfo))

	//cblogger.Debug(result)
	for _, curSubnet := range result.Subnets {
		cblogger.Infof("Retrieve [%s] Subnet info", *curSubnet.SubnetId)
		arrSubnetInfo := ExtractSubnetDescribeInfo(curSubnet)
		arrSubnetInfo.TagList, _ = VPCHandler.TagHandler.ListTag(irs.SUBNET, arrSubnetInfo.IId)

		//arrSubnetInfo, errSubnet := VPCHandler.GetSubnet(*curSubnet.SubnetId)
		/*
			if errSubnet != nil {
				return nil, errSubnet
			}
		*/
		//arrSubnetInfoList = append(arrSubnetInfoList, arrSubnetInfo)
		arrSubnetInfoList = append(arrSubnetInfoList, arrSubnetInfo)
	}

	//cblogger.Debug(arrSubnetInfoList)
	return arrSubnetInfoList, nil
}

func (VPCHandler *AwsVPCHandler) GetSubnet(reqSubnetId string) (irs.SubnetInfo, error) {
	cblogger.Infof("SubnetId : [%s]", reqSubnetId)

	input := &ec2.DescribeSubnetsInput{
		SubnetIds: []*string{
			aws.String(reqSubnetId),
		},
	}

	//cblogger.Debug(input)
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")

	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: reqSubnetId,
		CloudOSAPI:   "DescribeSubnets()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	result, err := VPCHandler.Client.DescribeSubnets(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Info(result)
	//cblogger.Debug(result)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
				callLogInfo.ErrorMSG = aerr.Error()
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
			callLogInfo.ErrorMSG = err.Error()
		}
		callogger.Info(call.String(callLogInfo))
		return irs.SubnetInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	if !reflect.ValueOf(result.Subnets).IsNil() {
		retSubnetInfo := ExtractSubnetDescribeInfo(result.Subnets[0])
		retSubnetInfo.TagList, _ = VPCHandler.TagHandler.ListTag(irs.SUBNET, retSubnetInfo.IId)

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

// Subnet 정보를 추출함
func ExtractSubnetDescribeInfo(subnetInfo *ec2.Subnet) irs.SubnetInfo {
	vNetworkInfo := irs.SubnetInfo{
		IId:       irs.IID{SystemId: *subnetInfo.SubnetId},
		IPv4_CIDR: *subnetInfo.CidrBlock,
		//Status:    *subnetInfo.State,
		Zone: *subnetInfo.AvailabilityZone,
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

	// keyValueList := []irs.KeyValue{
	// 	{Key: "VpcId", Value: *subnetInfo.VpcId},
	// 	{Key: "MapPublicIpOnLaunch", Value: strconv.FormatBool(*subnetInfo.MapPublicIpOnLaunch)},
	// 	{Key: "AvailableIpAddressCount", Value: strconv.FormatInt(*subnetInfo.AvailableIpAddressCount, 10)},
	// 	{Key: "AvailabilityZone", Value: *subnetInfo.AvailabilityZone},
	// 	{Key: "Status", Value: *subnetInfo.State},
	// }
	// vNetworkInfo.KeyValueList = keyValueList
	vNetworkInfo.KeyValueList = irs.StructToKeyValueList(subnetInfo)

	return vNetworkInfo
}

func (VPCHandler *AwsVPCHandler) AddSubnet(vpcIID irs.IID, subnetInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	cblogger.Infof("add [%s] Subnet - CIDR : %s", subnetInfo.IId.NameId, subnetInfo.IPv4_CIDR)
	resSubnet, errSubnet := VPCHandler.CreateSubnet(vpcIID.SystemId, subnetInfo)
	if errSubnet != nil {
		cblogger.Error(errSubnet)
		return irs.VPCInfo{}, errSubnet
	}
	cblogger.Info(resSubnet)

	vpcInfo, errVpcInfo := VPCHandler.GetVPC(vpcIID)
	if errVpcInfo != nil {
		cblogger.Error(errVpcInfo)
		return irs.VPCInfo{}, errVpcInfo
	}

	findSubnet := false
	cblogger.Debug("============== check values =========")
	for posSubnet, curSubnetInfo := range vpcInfo.SubnetInfoList {
		cblogger.Debugf("%d - [%s] Subnet", posSubnet, curSubnetInfo.IId.SystemId)
		if resSubnet.IId.SystemId == curSubnetInfo.IId.SystemId {
			cblogger.Infof("found the requested [%s] Subnet. - SystemID:[%s]", subnetInfo.IId.NameId, curSubnetInfo.IId.SystemId)
			//for ~ range는 포인터가 아니라서 값 수정이 안됨. for loop으로 직접 서브넷을 체크하거나 vpcInfo의 배열의 값을 수정해야 함.
			cblogger.Infof("index : %d", posSubnet)
			//vpcInfo.SubnetInfoList[posSubnet].IId.NameId = "테스트~"
			vpcInfo.SubnetInfoList[posSubnet].IId.NameId = subnetInfo.IId.NameId
			findSubnet = true
			break
		}
	}

	if !findSubnet {
		cblogger.Errorf("The subnet creation was successful, but the information [%s] for the requested [%s] subnet was not found in the VPC's subnet list.", subnetInfo.IId.NameId, resSubnet.IId.SystemId)
		return irs.VPCInfo{}, errors.New("MismatchSubnet.NotFound: No SysmteId[" + resSubnet.IId.SystemId + "] found for newly created Subnet[" + subnetInfo.IId.NameId + "].")
	}
	//cblogger.Debug(vpcInfo)

	return vpcInfo, nil
}

func (VPCHandler *AwsVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger.Infof("remove [%s] Subnet from [%s] VPC", vpcIID.SystemId, subnetIID.SystemId)

	return VPCHandler.DeleteSubnet(subnetIID)
	//return false, nil
}

func (vpcHandler *AwsVPCHandler) ListIID() ([]*irs.IID, error) {
	var iidList []*irs.IID
	callLogInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, "ListIID", "DescribeVpcs()")
	start := call.Start()

	result, err := vpcHandler.Client.DescribeVpcs(&ec2.DescribeVpcsInput{})
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		cblogger.Error(err)
		return nil, err
	}
	calllogger.Info(call.String(callLogInfo))

	for _, curVpc := range result.Vpcs {
		cblogger.Debugf("Retrieve VPC Info [%s] ", *curVpc.VpcId)
		iid := irs.IID{SystemId: *curVpc.VpcId}
		iidList = append(iidList, &iid)
	}

	return iidList, nil
}
