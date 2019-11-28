// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by zephy@mz.co.kr, 2019.09.

//VNetworkHandler는 서브넷을 처리하는 핸들러임.
//VPC & Subnet 처리 (AlibabaCloud's Subnet --> VSwitch 임)
//Ver2 - <CB-Virtual Network> 개발 방안에 맞게 VPC기능은 외부에 숨기고 Subnet을 Main으로 함.

package resources

import (
	"reflect"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	/*
		"github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
		"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
		idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
		irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
		"github.com/davecgh/go-spew/spew"
	*/)

type AlibabaVNetworkHandler struct {
	Region idrv.RegionInfo
	Client *vpc.Client
}

type AlibabaVpcReqInfo struct {
	Name      string
	Id        string // Alibaba
	CidrBlock string // AWS
	IsDefault bool   // Alibaba
}

type AlibabaVpcInfo struct {
	Name      string
	Id        string
	CidrBlock string // AWS, Alibaba
	IsDefault bool   // AWS, Alibaba
	// State     string // AWS

	Status    string // Alibaba
	CenStatus string // Alibaba

	ResourceGroupId string // Alibaba
	VRouterId       string // Alibaba
	RegionId        string // Alibaba
	RouterTableIds  []string
	VSwitchId       []string

	CreationTime string // Alibaba
	Description  string // Alibaba
}

type AlibabaZoneInfo struct {
	LocalName string
	ZoneId    string // Alibaba
}

func (vNetworkHandler *AlibabaVNetworkHandler) GetZone(regionId string) (AlibabaZoneInfo, error) {
	cblogger.Info("Region Id : ", regionId)

	request := vpc.CreateDescribeZonesRequest()
	request.Scheme = "https"

	result, err := vNetworkHandler.Client.DescribeZones(request)
	if err != nil {
		cblogger.Error(err.Error())
		return AlibabaZoneInfo{}, err
	}

	cblogger.Info(result)
	//spew.Dump(result)

	if !reflect.ValueOf(result.Zones).IsNil() {
		alibabaZoneInfo := ExtractZoneDescribeInfo(&result.Zones.Zone[0])
		return alibabaZoneInfo, nil
	} else {
		return AlibabaZoneInfo{}, nil
	}
}

func (vNetworkHandler *AlibabaVNetworkHandler) ListVpc() ([]*AlibabaVpcInfo, error) {
	cblogger.Debug("Start")

	request := vpc.CreateDescribeVpcsRequest()
	request.Scheme = "https"

	result, err := vNetworkHandler.Client.DescribeVpcs(request)
	if err != nil {
		return nil, err
	}

	var vNetworkInfoList []*AlibabaVpcInfo
	for _, curVpc := range result.Vpcs.Vpc {
		cblogger.Infof("[%s] VPC 정보 조회", curVpc.VpcId)
		vNetworkInfo := ExtractVpcDescribeInfo(&curVpc)
		vNetworkInfoList = append(vNetworkInfoList, &vNetworkInfo)
	}

	spew.Dump(vNetworkInfoList)
	return vNetworkInfoList, nil
}

//@TODO : 여러 VPC에 속한 Subnet 목록을 조회하게되는데... CB-Vnet의 서브넷만 조회해야할지 결정이 필요함. 현재는 1차 버전 문맥상 CB-Vnet으로 내부적으로 제한해서 구현했음.
func (vNetworkHandler *AlibabaVNetworkHandler) ListVNetwork() ([]*irs.VNetworkInfo, error) {
	cblogger.Debug("Start")
	var vNetworkInfoList []*irs.VNetworkInfo

	//cblogger.Infof("조회 범위를 CBDefaultVPC[%s]로 제한합니다.", GetCBDefaultVNetName())
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
	request := vpc.CreateDescribeVSwitchesRequest()
	request.Scheme = "https"

	request.VpcId = VpcId

	// input := &ec2.DescribeSubnetsInput{
	// 	Filters: []*ec2.Filter{
	// 		{
	// 			Name: aws.String("vpc-id"),
	// 			Values: []*string{
	// 				aws.String(VpcId),
	// 			},
	// 		},
	// 	},
	// }

	result, err := vNetworkHandler.Client.DescribeVSwitches(request)
	cblogger.Info(result)
	if err != nil {
		return nil, err
	}

	//for _, curSubnet := range result.VSwitch {
	//cblogger.Infof("[%s] Subnet 정보 조회", curSubnet.VSwitchId)
	//vNetworkInfo := ExtractSubnetDescribeInfo(&curSubnet)
	//vNetworkInfoList = append(vNetworkInfoList, &vNetworkInfo)
	//}

	spew.Dump(vNetworkInfoList)
	return vNetworkInfoList, nil
}

func (vNetworkHandler *AlibabaVNetworkHandler) GetVpc(vpcName string) (AlibabaVpcInfo, error) {
	cblogger.Info("VPC Name : ", vpcName)

	request := vpc.CreateDescribeVpcsRequest()
	request.Scheme = "https"

	request.VpcName = vpcName

	result, err := vNetworkHandler.Client.DescribeVpcs(request)
	if err != nil {
		return AlibabaVpcInfo{}, err
	}

	cblogger.Info(result)
	//spew.Dump(result)

	if !reflect.ValueOf(result.Vpcs.Vpc).IsNil() {
		alibabaVpcInfo := ExtractVpcDescribeInfo(&result.Vpcs.Vpc[0])
		return alibabaVpcInfo, nil
	} else {
		return AlibabaVpcInfo{}, nil
	}
}

// FindOrCreateMcloudBaristaDefaultVPC()에서 호출됨. - 이 곳은 나중을 위해 전달 받은 정보는 이용함
// 기본 VPC 생성이 필요하면 FindOrCreateMcloudBaristaDefaultVPC()를 호출할 것
func (vNetworkHandler *AlibabaVNetworkHandler) CreateVpc(alibabaVpcReqInfo AlibabaVpcReqInfo) (AlibabaVpcInfo, error) {
	cblogger.Info(alibabaVpcReqInfo)
	return AlibabaVpcInfo{}, nil
	/*

		request := vpc.CreateCreateVpcRequest()
		request.Scheme = "https"

		// input := &ec2.CreateVpcInput{
		// 	CidrBlock: aws.String(alibabaVpcReqInfo.CidrBlock),
		// }

		request.CidrBlock = alibabaVpcReqInfo.CidrBlock
		request.VpcName = alibabaVpcReqInfo.Name     // "cb-vpc-zep02"
		request.Description = alibabaVpcReqInfo.Name // "cb vpc zep02"

		spew.Dump(request)
		result, err := vNetworkHandler.Client.CreateVpc(request)
		if err != nil {
			return AlibabaVpcInfo{}, err
		}

		cblogger.Info(result)
		spew.Dump(result)
		// alibabaVpcInfo := ExtractVpcDescribeInfo(result.Vpc)

		alibabaVpcInfo := AlibabaVpcInfo{}
		//VPC Name 태깅
		// tagInput := &ec2.CreateTagsInput{
		// 	Resources: []*string{
		// 		aws.String(*result.Vpc.VpcId),
		// 	},
		// 	Tags: []*ec2.Tag{
		// 		{
		// 			Key:   aws.String("Name"),
		// 			Value: aws.String(alibabaVpcReqInfo.Name),
		// 		},
		// 	},
		// }
		// spew.Dump(tagInput)

		// _, errTag := vNetworkHandler.Client.CreateTags(tagInput)
		// if errTag != nil {
		// 	//@TODO : Name 태깅 실패시 생성된 VPC를 삭제할지 Name 태깅을 하라고 전달할지 결정해야 함. - 일단, 바깥에서 처리 가능하도록 생성된 VPC 정보는 전달 함.
		// 	if aerr, ok := errTag.(awserr.Error); ok {
		// 		switch aerr.Code() {
		// 		default:
		// 			cblogger.Error(aerr.Error())
		// 		}
		// 	} else {
		// 		// Print the error, cast err to awserr.Error to get the Code and
		// 		// Message from an error.
		// 		cblogger.Error(errTag.Error())
		// 	}
		// 	return alibabaVpcInfo, errTag
		// }
		var routerTableIdsList []string
		routerTableIdsList = append(routerTableIdsList, result.RouteTableId)
		alibabaVpcInfo.RouteTableId = routerTableIdsList

		var vSwitchIdList []string
		vSwitchIdList = append(vSwitchIdList, result.VRouterId)
		alibabaVpcInfo.VRouterId = vSwitchIdList

		alibabaVpcInfo.Name = alibabaVpcReqInfo.Name
		alibabaVpcInfo.Id = result.VpcId
		alibabaVpcInfo.ResourceGroupId = result.ResourceGroupId
		alibabaVpcInfo.RequestId = result.RequestId

		return alibabaVpcInfo, nil
	*/
}

func (vNetworkHandler *AlibabaVNetworkHandler) CreateVNetwork(vNetworkReqInfo irs.VNetworkReqInfo) (irs.VNetworkInfo, error) {
	// 기본 가상 네트워크가 생성되지 않았을 경우 디폴트 네트워크 생성 (CB-VNet)
	cblogger.Info(vNetworkReqInfo)

	return irs.VNetworkInfo{}, nil

	/*

		//최대 5개의 VPC 생성 제한이 있기 때문에 기본VPC 조회시 에러 처리를 해줌.
		VpcId, errVpc := vNetworkHandler.FindOrCreateMcloudBaristaDefaultVPC(vNetworkReqInfo)
		cblogger.Info("CBDefaultVPC 조회 결과 : ", VpcId)
		if errVpc != nil {
			return irs.VNetworkInfo{}, errVpc
		}

		//서브넷 생성
		//@TODO : Subnet과 VPC모두 CSP별 고정된 값으로 드라이버가 내부적으로 자동으로 생성하도록 CB규약이 바뀌어서 서브넷 정보 기반의 로직은 모두 잠시 죽여 놓음 - 리스트 요청시에도 내부적으로 자동 생성하도록 변경 중
		request := vpc.CreateCreateVSwitchRequest()
		request.Scheme = "https"

		request.CidrBlock = vNetworkReqInfo.CidrBlock // "192.168.4.0/24"
		request.VpcId = VpcId                         // "vpc-t4nokxb60pv7ejm9ebsjr"

		// Zone ID 획득이 필요함. call getZoneID(regionID)
		alibabaZoneInfo, err := vNetworkHandler.GetZone(GetCBDefaultVNetName())
		if err != nil {
			cblogger.Error(err.Error())
			return irs.VNetworkInfo{}, err
		}

		//정보가 존재하면...
		if alibabaZoneInfo.ZoneId != "" {
			request.ZoneId = alibabaZoneInfo.ZoneId
		} else {
			return irs.VNetworkInfo{}, err
		}

		request.VSwitchName = GetCBDefaultSubnetName() // "CB-VNet-Sub"
		if vNetworkReqInfo.Name != nil {
			request.VSwitchName = vNetworkReqInfo.Name // "vsw-zep04"
		}
		request.Description = request.VSwitchName

		// input := &ec2.CreateSubnetInput{
		// 	//CidrBlock: aws.String(vNetworkReqInfo.CidrBlock),
		// 	CidrBlock: aws.String(GetCBDefaultCidrBlock()), // VPC와 동일한 대역의 CB-Default Subnet을 생성 함.
		// 	VpcId:     aws.String(VpcId),
		// }

		cblogger.Info(request)
		result, err := vNetworkHandler.Client.CreateVSwitch(request)
		if err != nil {
			if aerr, ok := err.(errors.Error); ok {
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

		// getVSWitch() GetVNetwork(vNetworkID string) (VNetworkInfo, error)
		alibabaVNetworkInfo, err := vNetworkHandler.GetVNetwork(result.VSwitchId)
		if err != nil {
			if aerr, ok := err.(errors.Error); ok {
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

		//기존 정보가 존재하면...
		if alibabaVNetworkInfo.Id != "" {
			return irs.VNetworkInfo{}, nil
		} else {
			//vNetworkInfo := irs.VNetworkInfo{}
			vNetworkInfo := ExtractSubnetDescribeInfo(result.VSwitches[0])
		}

		//@TODO : Subnet과 VPC모두 CSP별 고정된 값으로 드라이버가 내부적으로 자동으로 생성하도록 CB규약이 바뀌어서 서브넷 정보 기반의 로직은 모두 잠시 죽여 놓음 - 리스트 요청시에도 내부적으로 자동 생성하도록 변경 중
		//Subnet Name 태깅
		// cblogger.Info("**필수 정보 없이 CB Subnet 자동 구현을 위해 사용자의 정보는 무시하고 기본 서브넷을 구성함.**")
		// tagInput := &ec2.CreateTagsInput{
		// 	Resources: []*string{
		// 		aws.String(*result.Subnet.SubnetId),
		// 	},
		// 	Tags: []*ec2.Tag{
		// 		{
		// 			Key: aws.String("Name"),
		// 			//Value: aws.String(vNetworkReqInfo.Name),
		// 			Value: aws.String(GetCBDefaultSubnetName()), //서브넷도 히든 컨셉이라 CB-Default SUbnet 이름으로 생성 함.
		// 		},
		// 	},
		// }
		// spew.Dump(tagInput)

		// _, errTag := vNetworkHandler.Client.CreateTags(tagInput)
		// if errTag != nil {
		// 	//@TODO : Name 태깅 실패시 생성된 VPC를 삭제할지 Name 태깅을 하라고 전달할지 결정해야 함. - 일단, 바깥에서 처리 가능하도록 생성된 VPC 정보는 전달 함.
		// 	if aerr, ok := errTag.(awserr.Error); ok {
		// 		switch aerr.Code() {
		// 		default:
		// 			cblogger.Error(aerr.Error())
		// 		}
		// 	} else {
		// 		// Print the error, cast err to awserr.Error to get the Code and
		// 		// Message from an error.
		// 		cblogger.Error(errTag.Error())
		// 	}
		// 	return vNetworkInfo, errTag
		// }

		vNetworkInfo.Name = vNetworkReqInfo.Name

		return vNetworkInfo, nil
	*/
}

//vNetworkID를 전달 받으면 해당 Subnet을 조회하고 / vNetworkID의 값이 없으면 CB Default Subnet을 조회함.
func (vNetworkHandler *AlibabaVNetworkHandler) GetVNetwork(vNetworkID string) (irs.VNetworkInfo, error) {
	cblogger.Infof("vNetworkID : [%s]", vNetworkID)
	return irs.VNetworkInfo{}, nil
	/*

		request := vpc.CreateDescribeVSwitchesRequest()
		request.Scheme = "https"

		// request.VpcId = "vpc-t4nokxb60pv7ejm9ebsjr"
		request.VSwitchId = vNetworkID // "vsw-t4nqm69s5d8284ywbvbpx"
		// request.ZoneId = "ap-southeast-1b"
		// request.VSwitchName = "cb-vsw-zep02"
		// request.RouteTableId = "vtb-t4ndqzv7pc7svkqe0eqtl"

		// input := &ec2.DescribeSubnetsInput{
		// 	SubnetIds: []*string{
		// 		aws.String(vNetworkID),
		// 	},
		// }

		result, err := vNetworkHandler.Client.DescribeVSwitches(request)
		if err != nil {
			if aerr, ok := err.(errors.Error); ok {
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
		if !reflect.ValueOf(result.VSwitches).IsNil() {
			vNetworkInfo := ExtractSubnetDescribeInfo(result.VSwitches[0])
			return vNetworkInfo, nil
		} else {
			return irs.VNetworkInfo{}, nil
		}
	*/
}

func (vNetworkHandler *AlibabaVNetworkHandler) GetVNetworkByName(vNetworkName string) (irs.VNetworkInfo, error) {
	cblogger.Infof("vNetworkID : [%s]", vNetworkName)
	return irs.VNetworkInfo{}, nil

	/*

		request := vpc.CreateDescribeVSwitchesRequest()
		request.Scheme = "https"

		// request.VpcId = "vpc-t4nokxb60pv7ejm9ebsjr"
		// request.VSwitchId = vNetworkID // "vsw-t4nqm69s5d8284ywbvbpx"
		// request.ZoneId = "ap-southeast-1b"
		request.VSwitchName = vNetworkName // "cb-vsw-zep02"
		// request.RouteTableId = "vtb-t4ndqzv7pc7svkqe0eqtl"

		// input := &ec2.DescribeSubnetsInput{
		// 	SubnetIds: []*string{
		// 		aws.String(vNetworkID),
		// 	},
		// }

		result, err := vNetworkHandler.Client.DescribeVSwitches(request)
		if err != nil {
			if aerr, ok := err.(errors.Error); ok {
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
		if !reflect.ValueOf(result.VSwitches).IsNil() {
			vNetworkInfo := ExtractSubnetDescribeInfo(result.VSwitches[0])
			return vNetworkInfo, nil
		} else {
			return irs.VNetworkInfo{}, nil
		}
	*/
}

//Zone 정보를 추출함
func ExtractZoneDescribeInfo(zoneInfo *vpc.Zone) AlibabaZoneInfo {
	return AlibabaZoneInfo{}
	/*
			alibabaZoneInfo := AlibabaZoneInfo{
				Name: *zoneInfo.LocalName,
				Id:   *zoneInfo.ZoneId,
			}
		return alibabaZoneInfo
	*/
}

//VPC 정보를 추출함
func ExtractVpcDescribeInfo(vpcInfo *vpc.Vpc) AlibabaVpcInfo {
	return AlibabaVpcInfo{}
	/*
		alibabaVpcInfo := AlibabaVpcInfo{
			Name:      *vpcInfo.VpcName,
			Id:        *vpcInfo.VpcId,
			CidrBlock: *vpcInfo.CidrBlock,
			IsDefault: *vpcInfo.IsDefault,
			Status:    *vpcInfo.Status,

			CenStatus:       *vpcInfo.CenStatus,
			ResourceGroupId: *vpcInfo.ResourceGroupId,
			VRouterId:       *vpcInfo.VRouterId,

			CreationTime: *vpcInfo.CreationTime,
			RegionId:     *vpcInfo.RegionId,

			RouterTableIds: *vpcInfo.RouterTableIds,
			VSwitchId:      *vpcInfo.VSwitchIds,

			Description: *vpcInfo.Description,
		}

		cblogger.Debug("RouterTableId 찾기")
		for _, rt := range vpcInfo.RouterTableIds {
			alibabaVpcInfo.RouterTableIds.append(*rt.RouterTableIds)
		}

		cblogger.Debug("VSwitchId 찾기")
		for _, vs := range vpcInfo.VSwitchIds {
			alibabaVpcInfo.VSwitchId.append(*vs.VSwitchIds)
		}

		//Name은 Tag의 "Name" 속성에만 저장됨
		// cblogger.Debug("Name Tag 찾기")
		// for _, t := range vpcInfo.Tags {
		// 	if *t.Key == "Name" {
		// 		alibabaVpcInfo.Name = *t.Value
		// 		cblogger.Debug("VPC Name : ", alibabaVpcInfo.Name)
		// 		break
		// 	}
		// }

		return alibabaVpcInfo
	*/
}

//Subnet 정보를 추출함
func ExtractSubnetDescribeInfo(subnetInfo *vpc.VSwitch) irs.VNetworkInfo {
	return irs.VNetworkInfo{}
	/*
		vNetworkInfo := irs.VNetworkInfo{
			Id:            *subnetInfo.VSwitchId,
			Name:          *subnetInfo.VSwitchName,
			AddressPrefix: *subnetInfo.CidrBlock,
			Status:        *subnetInfo.Status,
		}

		//Name은 Tag의 "Name" 속성에만 저장됨
		// cblogger.Debug("Name Tag 찾기")
		// for _, t := range subnetInfo.Tags {
		// 	if *t.Key == "Name" {
		// 		vNetworkInfo.Name = *t.Value
		// 		cblogger.Debug("Subnet Name : ", vNetworkInfo.Name)
		// 		break
		// 	}
		// }

		keyValueList := []irs.KeyValue{
			{Key: "VpcId", Value: *subnetInfo.VpcId},
			// {Key: "MapPublicIpOnLaunch", Value: strconv.FormatBool(*subnetInfo.MapPublicIpOnLaunch)},
			// {Key: "AvailableIpAddressCount", Value: strconv.FormatInt(*subnetInfo.AvailableIpAddressCount, 10)},
			// {Key: "AvailabilityZone", Value: *subnetInfo.AvailabilityZone},

			{Key: "AvailableIpAddressCount", Value: strconv.FormatInt(*subnetInfo.AvailableIpAddressCount)},
			{Key: "CreationTime", Value: *subnetInfo.CreationTime},
			{Key: "Description", Value: *subnetInfo.Description},
			{Key: "IsDefault", Value: strconv.FormatBool(*subnetInfo.IsDefault)},
			{Key: "ResourceGroupId", Value: *subnetInfo.ResourceGroupId},
			{Key: "RouteTable", Value: *subnetInfo.RouteTable},
			{Key: "ZoneId", Value: *subnetInfo.ZoneId},
			{Key: "Tags", Value: *subnetInfo.Tags},
			{Key: "NetworkAclId", Value: *subnetInfo.NetworkAclId},
		}
		vNetworkInfo.KeyValueList = keyValueList

		return vNetworkInfo
	*/
}

func (vNetworkHandler *AlibabaVNetworkHandler) DeleteVpc(vpcId string) (bool, error) {
	cblogger.Info("vpcId : [%s]", vpcId)
	return false, nil
	/*

		request := vpc.CreateDeleteVpcRequest()
		request.Scheme = "https"

		request.VpcId = vpcId // "vpc-t4n4e2wr3x13ewzq2fmoq"

		// input := &ec2.DeleteVpcInput{
		// 	VpcId: aws.String(vpcId),
		// }

		result, err := vNetworkHandler.Client.DeleteVpc(request)
		if err != nil {
			return false, err
		}

		cblogger.Info(result)
		spew.Dump(result)
		return true, nil
	*/
}

//서브넷 삭제
//마지막 서브넷인 경우 CB-Default Virtual Network도 함께 제거
func (vNetworkHandler *AlibabaVNetworkHandler) DeleteVNetwork(vNetworkID string) (bool, error) {
	cblogger.Info("vNetworkID : [%s]", vNetworkID)
	return false, nil

	/*
		request := vpc.CreateDeleteVSwitchRequest()
		request.Scheme = "https"

		request.VSwitchId = vNetworkID // "vsw-t4nf9v444i50py65ghtes"

		// input := &ec2.DeleteSubnetInput{
		// 	SubnetId: aws.String(vNetworkID),
		// }

		_, err := vNetworkHandler.Client.DeleteVSwitch(request)
		if err != nil {
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
	*/

}
