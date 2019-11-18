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

//2019-10-17 충돌 방지및 CB-VNet을 감추기 위해 명시적으로 각 핸들러들의 Create나 Delete에서만 자동으로 처리하며,
//           정확하지는 않아도 네트워크 범위를 id 기반에서 name기반으로 변경 함. (예) vpc-name / subnet-name
//           *** Subnet은 1개만 생성되도록 제한 함..***
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

	//cbtool "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/aws/resources/tool"
	//cbtool "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/aws/resources/tool"
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

	cblogger.Infof("조회 범위를 CBDefaultVPC[%s]와 CBDefaultSubnet[%s]으로 제한합니다.", GetCBDefaultVNetName(), GetCBDefaultSubnetName())

	VpcId := vNetworkHandler.GetMcloudBaristaDefaultVpcId()
	if VpcId == "" {
		return nil, nil
	}
	/*
		VpcId, errVpc := vNetworkHandler.FindOrCreateMcloudBaristaDefaultVPC(irs.VNetworkReqInfo{})
		cblogger.Info("CBDefaultVPC 조회 결과 : ", VpcId)
		if errVpc != nil {
			return nil, errVpc
		}
		if VpcId == "" {
			return vNetworkInfoList, nil
		}
	*/

	/*
		awsCBNetworkInfo, errCBInfo := vNetworkHandler.GetAutoCBNetworkInfo()
		if errCBInfo != nil {
			return nil, errCBInfo
		}

		//생성된 CB Default Virtual Network가 없는 경우 nil 리턴
		if awsCBNetworkInfo.VpcName == "" {
			return vNetworkInfoList, nil
		}

		VpcId := awsCBNetworkInfo.VpcId
	*/

	//기본 CBVPC에 속한 서브넷만 조회
	input := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("vpc-id"),
				Values: []*string{
					aws.String(VpcId),
				},
			},
			{
				Name: aws.String("tag:Name"),
				Values: []*string{
					aws.String(GetCBDefaultSubnetName()),
				},
			},
		},
	}

	//spew.Dump(input)
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

	spew.Dump(result)
	for _, curSubnet := range result.Subnets {
		cblogger.Infof("[%s] Subnet 정보 조회", *curSubnet.SubnetId)
		vNetworkInfo := ExtractSubnetDescribeInfo(curSubnet)
		vNetworkInfoList = append(vNetworkInfoList, &vNetworkInfo)
	}

	spew.Dump(vNetworkInfoList)
	return vNetworkInfoList, nil
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

// FindOrCreateMcloudBaristaDefaultVPC()에서 호출됨. - 이 곳은 나중을 위해 전달 받은 정보는 이용함
// 기본 VPC 생성이 필요하면 FindOrCreateMcloudBaristaDefaultVPC()를 호출할 것
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
		cblogger.Errorf("VPC에 Name[%s] 설정 실패", awsVpcReqInfo.Name)
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
		//return awsVpcInfo, errTag
	} else {
		awsVpcInfo.Name = awsVpcReqInfo.Name
	}

	//====================================
	// PublicIP 할당을 위해 IGW 생성및 연결
	//====================================
	//IGW 생성
	resultIGW, errIGW := vNetworkHandler.Client.CreateInternetGateway(&ec2.CreateInternetGatewayInput{})
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
		return awsVpcInfo, errIGW
	}

	cblogger.Info(resultIGW)

	//IGW Name Tag 설정
	if SetNameTag(vNetworkHandler.Client, *resultIGW.InternetGateway.InternetGatewayId, awsVpcReqInfo.Name) {
		cblogger.Infof("IGW에 %s Name 설정 성공", awsVpcReqInfo.Name)
	} else {
		cblogger.Errorf("IGW에 %s Name 설정 실패", awsVpcReqInfo.Name)
	}

	// VPC에 IGW연결
	inputIGW := &ec2.AttachInternetGatewayInput{
		InternetGatewayId: aws.String(*resultIGW.InternetGateway.InternetGatewayId),
		VpcId:             aws.String(awsVpcInfo.Id),
	}

	resultIGWAttach, errIGWAttach := vNetworkHandler.Client.AttachInternetGateway(inputIGW)
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
		return awsVpcInfo, errIGWAttach
	}

	cblogger.Info(resultIGWAttach)

	// 생성된 VPC의 기본 라우팅 테이블에 IGW 라우팅 정보 추가
	errRoute := vNetworkHandler.CreateRouteIGW(awsVpcInfo.Id, *resultIGW.InternetGateway.InternetGatewayId)
	if errRoute != nil {
		return awsVpcInfo, errRoute
	}

	return awsVpcInfo, nil
}

//2019-11-16부로 CB-Driver 전체 로직이 NameId 기반으로 변경됨.
func (vNetworkHandler *AwsVNetworkHandler) CreateVNetwork(vNetworkReqInfo irs.VNetworkReqInfo) (irs.VNetworkInfo, error) {
	cblogger.Info(vNetworkReqInfo)

	//사용자에게 Open된 Api의 List는 드라이버가 바라보는 List와 달라서 등록된 이름으로 검색해야 해서 로직 변경.
	/*
		vpcList, _ := vNetworkHandler.ListVNetwork()
		if len(vpcList) > 0 {
			cblogger.Error("이미 Default Subnet이 존재하기 때문에 생성하지 않고 기존 정보와 함께 에러를 리턴함.")
			cblogger.Info(vpcList)
			//return *vpcList[0], errors.New("이미 Default Subnet이 존재합니다.")
			return *vpcList[0], errors.New("InvalidSubnet.Duplicate: The subnet '" + GetCBDefaultSubnetName() + "' already exists.")
		}
	*/

	vpcInfo, errVpcInfo := vNetworkHandler.GetVNetwork(GetCBDefaultSubnetName())
	if errVpcInfo == nil {
		cblogger.Error("이미 Default Subnet이 존재하기 때문에 생성하지 않고 기존 정보와 함께 에러를 리턴함.")
		cblogger.Info(vpcInfo)
		return vpcInfo, errors.New("InvalidVNetwork.Duplicate: The CBVnetwork '" + GetCBDefaultSubnetName() + "' already exists.")
	}

	//최대 5개의 VPC 생성 제한이 있기 때문에 기본VPC 조회시 에러 처리를 해줌.
	vpcId, errVpc := vNetworkHandler.FindOrCreateMcloudBaristaDefaultVPC(vNetworkReqInfo)
	cblogger.Info("CBDefaultVPC 조회 결과 : ", vpcId)
	if errVpc != nil {
		return irs.VNetworkInfo{}, errVpc
	}

	//서브넷 생성
	//@TODO : Subnet과 VPC모두 CSP별 고정된 값으로 드라이버가 내부적으로 자동으로 생성하도록 CB규약이 바뀌어서 서브넷 정보 기반의 로직은 모두 잠시 죽여 놓음 - 리스트 요청시에도 내부적으로 자동 생성하도록 변경 중
	input := &ec2.CreateSubnetInput{
		//CidrBlock: aws.String(vNetworkReqInfo.CidrBlock),
		CidrBlock: aws.String(GetCBDefaultCidrBlock()), // VPC와 동일한 대역의 CB-Default Subnet을 생성 함.
		VpcId:     aws.String(vpcId),
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

	//@TODO : Subnet과 VPC모두 CSP별 고정된 값으로 드라이버가 내부적으로 자동으로 생성하도록 CB규약이 바뀌어서 서브넷 정보 기반의 로직은 모두 잠시 죽여 놓음 - 리스트 요청시에도 내부적으로 자동 생성하도록 변경 중
	//Subnet Name 태깅
	cblogger.Info("**필수 정보 없이 CB Subnet 자동 구현을 위해 사용자의 정보는 무시하고 기본 서브넷을 구성함.**")
	tagInput := &ec2.CreateTagsInput{
		Resources: []*string{
			aws.String(*result.Subnet.SubnetId),
		},
		Tags: []*ec2.Tag{
			{
				Key: aws.String("Name"),
				//Value: aws.String(vNetworkReqInfo.Name),
				Value: aws.String(GetCBDefaultSubnetName()), //서브넷도 히든 컨셉이라 CB-Default SUbnet 이름으로 생성 함.
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
		//return vNetworkInfo, errTag
	} else {
		vNetworkInfo.Name = vNetworkReqInfo.Name
	}

	// VPC의 라우팅 테이블에 생성된 Subnet 정보를 추가 함.
	errSubnetRoute := vNetworkHandler.AssociateRouteTable(vpcId, vNetworkInfo.Id)
	if errSubnetRoute != nil {
	} else {
		return vNetworkInfo, errSubnetRoute
	}

	return vNetworkInfo, nil
}

// 생성된 VPC의 라우팅 테이블에 IGW(Internet Gateway) 라우팅 정보를 생성함 (AWS 콘솔의 라우팅 테이블의 [라우팅] Tab 처리)
func (vNetworkHandler *AwsVNetworkHandler) CreateRouteIGW(vpcId string, igwId string) error {
	cblogger.Infof("VPC ID : [%s] / IGW ID : [%s]", vpcId, igwId)
	routeTableId, errRoute := vNetworkHandler.GetCBDefaultRouteTable(vpcId)
	if errRoute != nil {
		return errRoute
	}

	cblogger.Infof("RouteTable[%s]에 IGW[%s]에 대한 라우팅(0.0.0.0/0) 정보를 추가 합니다.", routeTableId, igwId)
	input := &ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		GatewayId:            aws.String(igwId),
		RouteTableId:         aws.String(routeTableId),
	}

	result, err := vNetworkHandler.Client.CreateRoute(input)
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

// VPC에 설정된 0.0.0.0/0 라우터를 제거 함.
func (vNetworkHandler *AwsVNetworkHandler) DeleteRouteIGW(vpcId string) error {
	cblogger.Infof("VPC ID : [%s]", vpcId)
	routeTableId, errRoute := vNetworkHandler.GetCBDefaultRouteTable(vpcId)
	if errRoute != nil {
		return errRoute
	}

	cblogger.Infof("RouteTable[%s]에 할당된 라우팅(0.0.0.0/0) 정보를 삭제합니다.", routeTableId)
	input := &ec2.DeleteRouteInput{
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		RouteTableId:         aws.String(routeTableId),
	}
	cblogger.Info(input)

	result, err := vNetworkHandler.Client.DeleteRoute(input)
	if err != nil {
		cblogger.Errorf("RouteTable[%s]에 대한 라우팅(0.0.0.0/0) 정보 삭제 실패", routeTableId)
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
	cblogger.Infof("RouteTable[%s]에 대한 라우팅(0.0.0.0/0) 정보 삭제 완료", routeTableId)

	cblogger.Info(result)
	spew.Dump(result)
	return nil
}

// VPC의 라우팅 테이블에 생성된 Subnet을 연결 함.
func (vNetworkHandler *AwsVNetworkHandler) AssociateRouteTable(vpcId string, subnetId string) error {
	routeTableId, errRoute := vNetworkHandler.GetCBDefaultRouteTable(vpcId)
	if errRoute != nil {
		return errRoute
	}

	input := &ec2.AssociateRouteTableInput{
		RouteTableId: aws.String(routeTableId),
		SubnetId:     aws.String(subnetId),
	}

	result, err := vNetworkHandler.Client.AssociateRouteTable(input)
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

// 자동 생성된 VPC의 기본 라우팅 테이블 정보를 찾음
func (vNetworkHandler *AwsVNetworkHandler) GetCBDefaultRouteTable(vpcId string) (string, error) {
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

	result, err := vNetworkHandler.Client.DescribeRouteTables(input)
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

//2019-11-16부로 CB-Driver 전체 로직이 NameId 기반으로 변경됨.
//vNetworkID를 전달 받으면 해당 Subnet을 조회하고 / vNetworkID의 값이 없으면 CB Default Subnet을 조회함.
func (vNetworkHandler *AwsVNetworkHandler) GetVNetwork(vNetworkNameId string) (irs.VNetworkInfo, error) {
	cblogger.Infof("vNetworkNameId : [%s]", vNetworkNameId)
	cblogger.Infof("AWS 드라이버는 고정된 vNetwork 1개만 허용하므로 vNetworkNameId를 [%s] ==> [%s]로 변경합니다.", vNetworkNameId, GetCBDefaultSubnetName())

	vNetworkNameId = GetCBDefaultSubnetName()
	input := &ec2.DescribeSubnetsInput{}
	/*
		//Subnet ID를 전달 받으면 해당 서브넷을 조회
		if vNetworkID != "" {
			input.SubnetIds = ([]*string{
				aws.String(vNetworkID),
			})
		} else {
			//그렇지 않으면 Default CB-Subnet 조회
			input.Filters = ([]*ec2.Filter{
				&ec2.Filter{
					Name: aws.String("tag:Name"), // subnet-id
					Values: []*string{
						aws.String(GetCBDefaultSubnetName()),
					},
				},
			})
		}
	*/
	// end of if
	/*
		input := &ec2.DescribeSubnetsInput{
			SubnetIds: []*string{
				aws.String(vNetworkID),
			},
		}
	*/
	//NameId기반 로직 구현
	input.Filters = ([]*ec2.Filter{
		&ec2.Filter{
			Name: aws.String("tag:Name"), // subnet-id
			Values: []*string{
				aws.String(vNetworkNameId),
			},
		},
	})

	spew.Dump(input)
	result, err := vNetworkHandler.Client.DescribeSubnets(input)
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
		return irs.VNetworkInfo{}, err
	}

	if !reflect.ValueOf(result.Subnets).IsNil() {
		vNetworkInfo := ExtractSubnetDescribeInfo(result.Subnets[0])
		return vNetworkInfo, nil
	} else {
		return irs.VNetworkInfo{}, errors.New("InvalidVNetwork.NotFound: The CBVnetwork '" + vNetworkNameId + "' does not exist")
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
		Id:            *subnetInfo.SubnetId,
		AddressPrefix: *subnetInfo.CidrBlock,
		Status:        *subnetInfo.State,
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

	keyValueList := []irs.KeyValue{
		{Key: "VpcId", Value: *subnetInfo.VpcId},
		{Key: "MapPublicIpOnLaunch", Value: strconv.FormatBool(*subnetInfo.MapPublicIpOnLaunch)},
		{Key: "AvailableIpAddressCount", Value: strconv.FormatInt(*subnetInfo.AvailableIpAddressCount, 10)},
		{Key: "AvailabilityZone", Value: *subnetInfo.AvailabilityZone},
	}
	vNetworkInfo.KeyValueList = keyValueList

	return vNetworkInfo
}

//VPC에 연결된 모든 IGW를 삭제함.
func (vNetworkHandler *AwsVNetworkHandler) DeleteAllIGW(vpcId string) error {
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

	result, err := vNetworkHandler.Client.DescribeInternetGateways(input)
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

	// VPC 삭제를 위해 연결된 모든 IGW 제거
	// 일단, 에러는 무시함.
	for _, curIgw := range result.InternetGateways {
		//IGW 삭제전 연결된 IGW의 연결을 끊어야함.
		vNetworkHandler.DetachInternetGateway(vpcId, *curIgw.InternetGatewayId)
		//IGW 삭제
		vNetworkHandler.DeleteIGW(*curIgw.InternetGatewayId)
	}

	return nil
}

// VPC에 연결된 IGW의 연결을 해제함.
func (vNetworkHandler *AwsVNetworkHandler) DetachInternetGateway(vpcId string, igwId string) error {
	cblogger.Infof("VPC[%s]에 연결된 IGW[%s]의 연결을 해제함.", vpcId, igwId)

	input := &ec2.DetachInternetGatewayInput{
		InternetGatewayId: aws.String(igwId),
		VpcId:             aws.String(vpcId),
	}

	result, err := vNetworkHandler.Client.DetachInternetGateway(input)
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

// IGW를 삭제 함.
func (vNetworkHandler *AwsVNetworkHandler) DeleteIGW(igwId string) error {
	input := &ec2.DeleteInternetGatewayInput{
		InternetGatewayId: aws.String(igwId),
	}

	result, err := vNetworkHandler.Client.DeleteInternetGateway(input)
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

//@TODO : 의존관계를 체크해서 전체 삭제 로직을 추가해야 함.
func (vNetworkHandler *AwsVNetworkHandler) DeleteVpc(vpcId string) (bool, error) {
	cblogger.Infof("vpcId : [%s]", vpcId)

	cblogger.Info("VPC 제거를 위해 생성된 IGW / Route들 제거 시작")

	//VPC 제거에 필요한 리소스들을 모두 제거함.
	// VPC에 연결된 IGW를 삭제함.
	/*
		errIgw := vNetworkHandler.DeleteIGW(vpcId)
		if errIgw != nil {
			return false, errIgw
		}
	*/

	// 라우팅 테이블에 추가한 IGW 라우터를 먼저 삭제함.
	errRoute := vNetworkHandler.DeleteRouteIGW(vpcId)
	if errRoute != nil {
		cblogger.Error("라우팅 테이블에 추가한 0.0.0.0/0 IGW 라우터 삭제 실패")
		cblogger.Error(errRoute)
		//return false, errRoute
	} else {
		cblogger.Info("라우팅 테이블에 추가한 0.0.0.0/0 IGW 라우터 삭제 완료")
	}

	//VPC에 연결된 모든 IGW를 삭제함.
	errIgw := vNetworkHandler.DeleteAllIGW(vpcId)
	if errIgw != nil {
		cblogger.Error("모든 IGW 삭제 실패 : ", errIgw)
	} else {
		cblogger.Info("모든 IGW 삭제 완료")
	}

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
//2019-11-16부로 CB-Driver 전체 로직이 NameId 기반으로 변경됨.
func (vNetworkHandler *AwsVNetworkHandler) DeleteVNetwork(vNetworkNameId string) (bool, error) {
	//cblogger.Info("vNetworkID : [%s]", vNetworkID)
	cblogger.Infof("vNetworkNameId : [%s]", vNetworkNameId)
	cblogger.Infof("AWS 드라이버는 고정된 vNetwork 1개만 허용하므로 사용자 요청과 무관하게 [%s]을 삭제합니다.", GetCBDefaultSubnetName())

	cblogger.Infof("삭제할 vNetworkId를 검색합니다.")
	vNetworkNameId = GetCBDefaultSubnetName()
	vpcInfo, errVpcInfo := vNetworkHandler.GetVNetwork(vNetworkNameId)
	if errVpcInfo != nil {
		return false, errVpcInfo
	}

	vNetworkID := vpcInfo.Id
	input := &ec2.DeleteSubnetInput{
		SubnetId: aws.String(vNetworkID),
	}
	cblogger.Info(input)

	_, err := vNetworkHandler.Client.DeleteSubnet(input)
	cblogger.Info(err)
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

	//서브넷이 존재하는경우 요청한 서브넷 삭제 결과 리턴
	if len(subnetList) > 0 {
		return true, nil
	} else {
		//서브넷이 없는 경우 기본 CBVPC도 삭제
		cblogger.Info("모든 서브넷이 삭제되어서 VPC를 자동으로 삭제합니다.")
		VpcId, errVpc := vNetworkHandler.FindOrCreateMcloudBaristaDefaultVPC(irs.VNetworkReqInfo{})
		cblogger.Info("삭제할 CBDefaultVPC 조회 결과 : ", VpcId)
		if errVpc != nil {
			cblogger.Error("삭제할 CBDefaultVPC 조회 실패 : ", errVpc)
			return false, errVpc
		}

		//발생할 경우가 없어 보이지만 삭제할 CB Default VPC가 없으면 종료
		if VpcId == "" {
			cblogger.Error("삭제할 CBDefaultVPC가 존재하지 않음")
			return true, errors.New("삭제할 CBDefaultVPC가 존재하지 않음")
		}

		cblogger.Info("CBDefaultVPC를 삭제 함.")
		delVpc, errDelVpc := vNetworkHandler.DeleteVpc(VpcId)
		if errDelVpc != nil {
			cblogger.Error("CBDefaultVPC 삭제 실패 : ", errDelVpc)
			return false, errDelVpc
		}

		if delVpc {
			cblogger.Info("CBDefaultVPC 삭제 완료.")
			return true, nil
		} else {
			cblogger.Info("CBDefaultVPC 삭제 실패.")
			return false, errors.New("CBDefaultVPC를 삭제하지 못 했습니다.") //삭제 실패 이유를 모르는 경우
		}
	}

}
