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
	"strconv"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

type TencentVPCHandler struct {
	Region idrv.RegionInfo
	Client *vpc.Client
}

func (VPCHandler *TencentVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	cblogger.Debug(vpcReqInfo)

	//=================================================
	// 동일 이름 생성 방지 추가(cb-spider 요청 필수 기능)
	//=================================================
	isExist, errExist := VPCHandler.isExist(vpcReqInfo.IId.NameId)
	if errExist != nil {
		cblogger.Error(errExist)
		return irs.VPCInfo{}, errExist
	}
	if isExist {
		return irs.VPCInfo{}, errors.New("A VPC with the name " + vpcReqInfo.IId.NameId + " already exists.")
	}

	zoneId := VPCHandler.Region.Zone // default
	cblogger.Infof("Zone : %s", zoneId)
	// if zoneId == "" { // vpc 자체는 region dependency임.
	// 	cblogger.Error("Connection information does not contain Zone information.")
	// 	return irs.VPCInfo{}, errors.New("Connection information does not contain Zone information.")
	// }

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: vpcReqInfo.IId.NameId,
		CloudOSAPI:   "CreateVpc()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	//=========================
	// VPC 생성
	//=========================
	request := vpc.NewCreateVpcRequest()
	request.VpcName = common.StringPtr(vpcReqInfo.IId.NameId)
	request.CidrBlock = common.StringPtr(vpcReqInfo.IPv4_CIDR)

	var tags []*vpc.Tag
	for _, inputTag := range vpcReqInfo.TagList {
		tag := &vpc.Tag{
			Key:   &inputTag.Key,
			Value: &inputTag.Value,
		}
		tags = append(tags, tag)
	}

	request.Tags = tags

	callLogStart := call.Start()
	response, err := VPCHandler.Client.CreateVpc(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	cblogger.Debug(response.ToJsonString())
	//cblogger.Debug(result)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.VPCInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	newVpcId := *response.Response.Vpc.VpcId // Subnet이 포함된 정보를 전달해야 하기 때문에 생성된 VPC Id를 보관함.

	//=========================
	// Subnet 생성
	//========================
	requestSubnet := vpc.NewCreateSubnetsRequest()

	requestSubnet.VpcId = common.StringPtr(newVpcId)
	requestSubnet.Subnets = []*vpc.SubnetInput{}

	for _, curSubnet := range vpcReqInfo.SubnetInfoList {
		cblogger.Infof("[%s] Subnet processing", curSubnet.IId.NameId)
		subnetZoneId := zoneId
		if curSubnet.Zone != "" {
			subnetZoneId = curSubnet.Zone
		}

		reqSubnet := &vpc.SubnetInput{
			CidrBlock:  common.StringPtr(curSubnet.IPv4_CIDR),
			SubnetName: common.StringPtr(curSubnet.IId.NameId),
			Zone:       common.StringPtr(subnetZoneId),
			//RouteTableId: common.StringPtr("route"),
		}
		requestSubnet.Subnets = append(requestSubnet.Subnets, reqSubnet)
	}

	responseSubnet, errSubnet := VPCHandler.Client.CreateSubnets(requestSubnet)
	cblogger.Debug(responseSubnet.ToJsonString())
	//cblogger.Debug(responseSubnet)
	if errSubnet != nil {
		cblogger.Error(errSubnet)
		return irs.VPCInfo{}, errSubnet
	}

	//신규로 생성된 VPC와 Subnet 정보를 irs.VPCInfo{}로 치환해도 되지만 수정의 편의및 최신 정보 통일을 위해 GetVPC롤 호출함.
	//생성된 Subnet을 포함한 VPC의 최신 정보를 조회함.
	retVpcInfo, errVpc := VPCHandler.GetVPC(irs.IID{SystemId: newVpcId})
	if errVpc != nil {
		cblogger.Error(errVpc)
		return irs.VPCInfo{}, errVpc
	}
	retVpcInfo.IId.NameId = vpcReqInfo.IId.NameId // 생성 시에는 NameId는 cb-spider를 위해 요청 받은 값을 그대로 리턴해야 함.

	return retVpcInfo, nil
}

// VPC 정보를 추출함
func ExtractVpcDescribeInfo(vpcInfo *vpc.Vpc) irs.VPCInfo {
	// cblogger.Debug("전달 받은 내용")
	// cblogger.Debug(vpcInfo)
	resVpcInfo := irs.VPCInfo{
		//NameId는 사용되지 않기 때문에 전달할 필요가 없지만 Tencent는 Name도 필수로 들어가니 전달함.
		IId:       irs.IID{SystemId: *vpcInfo.VpcId, NameId: *vpcInfo.VpcName},
		IPv4_CIDR: *vpcInfo.CidrBlock,
	}

	return resVpcInfo
}

func (VPCHandler *TencentVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	cblogger.Info("Start")

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: "ListVPC",
		CloudOSAPI:   "DescribeVpcs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := vpc.NewDescribeVpcsRequest()
	callLogStart := call.Start()
	response, err := VPCHandler.Client.DescribeVpcs(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	cblogger.Debug(response.ToJsonString())
	//cblogger.Debug(result)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		cblogger.Error(err)
		return nil, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Info("VPC Count : ", *response.Response.TotalCount)

	var vpcInfoList []*irs.VPCInfo
	if *response.Response.TotalCount > 0 {
		for _, curVpc := range response.Response.VpcSet {
			cblogger.Debugf("[%s] VPC Infomation reteive - [%s]", *curVpc.VpcId, *curVpc.VpcName)
			vpcInfo, vpcErr := VPCHandler.GetVPC(irs.IID{SystemId: *curVpc.VpcId})
			// cblogger.Info("==>조회 결과")
			// cblogger.Debug(vpcInfo)
			if vpcErr != nil {
				cblogger.Error(vpcErr)
				return nil, vpcErr
			}
			vpcInfoList = append(vpcInfoList, &vpcInfo)
		}
	}

	cblogger.Debugf("Number of Return Results List : [%d]", len(vpcInfoList))
	// cblogger.Debug(vpcInfoList)
	return vpcInfoList, nil
}

// cb-spider 정책상 이름 기반으로 중복 생성을 막아야 함.
func (VPCHandler *TencentVPCHandler) isExist(chkName string) (bool, error) {
	cblogger.Debugf("chkName : %s", chkName)

	request := vpc.NewDescribeVpcsRequest()
	request.Filters = []*vpc.Filter{
		&vpc.Filter{
			Name:   common.StringPtr("vpc-name"),
			Values: common.StringPtrs([]string{chkName}),
		},
	}

	response, err := VPCHandler.Client.DescribeVpcs(request)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	if *response.Response.TotalCount < 1 {
		return false, nil
	}

	cblogger.Infof("VPC information found - VpcId:[%s] / VpcName:[%s]", *response.Response.VpcSet[0].VpcId, *response.Response.VpcSet[0].VpcName)
	return true, nil
}

func (VPCHandler *TencentVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	cblogger.Info("VPC IID : ", vpcIID.SystemId)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: "GetVPC",
		CloudOSAPI:   "DescribeVpcs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := vpc.NewDescribeVpcsRequest()
	request.VpcIds = common.StringPtrs([]string{vpcIID.SystemId})

	callLogStart := call.Start()
	response, err := VPCHandler.Client.DescribeVpcs(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		cblogger.Errorf("An API error has returned: %s", err.Error())
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		return irs.VPCInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Debug("Number of VPCs : ", *response.Response.TotalCount)
	if *response.Response.TotalCount < 1 {
		return irs.VPCInfo{}, errors.New("Notfound: '" + vpcIID.SystemId + "' VPC Not found")
	}

	vpcInfo := ExtractVpcDescribeInfo(response.Response.VpcSet[0])
	cblogger.Debug(vpcInfo)

	//=======================
	// Subnet 처리
	//=======================
	var errSubnet error
	vpcInfo.SubnetInfoList, errSubnet = VPCHandler.ListSubnet(vpcIID.SystemId)
	if errSubnet != nil {
		callogger.Error(errSubnet)
		return vpcInfo, errSubnet
	}

	return vpcInfo, nil
}

func (VPCHandler *TencentVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	cblogger.Infof("Delete VPC : [%s]", vpcIID.SystemId)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: vpcIID.SystemId,
		CloudOSAPI:   "DeleteVpc()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := vpc.NewDeleteVpcRequest()
	request.VpcId = common.StringPtr(vpcIID.SystemId)

	callLogStart := call.Start()
	_, err := VPCHandler.Client.DeleteVpc(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		cblogger.Errorf("An API error has returned: %s", err.Error())
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		return false, err
	}
	callogger.Info(call.String(callLogInfo))

	return true, nil
}

func (VPCHandler *TencentVPCHandler) ListSubnet(reqVpcId string) ([]irs.SubnetInfo, error) {
	cblogger.Infof("reqVpcId : [%s]", reqVpcId)
	var arrSubnetInfoList []irs.SubnetInfo

	/*
		// logger for HisCall
		callogger := call.GetLogger("HISCALL")
		callLogInfo := call.CLOUDLOGSCHEMA{
			CloudOS:      call.TENCENT,
			RegionZone:   VPCHandler.Region.Zone,
			ResourceType: call.VPCSUBNET,
			ResourceName: "ListSubnet - VpcId:" + reqVpcId,
			CloudOSAPI:   "DescribeSubnets()",
			ElapsedTime:  "",
			ErrorMSG:     "",
		}
	*/

	request := vpc.NewDescribeSubnetsRequest()
	request.Filters = []*vpc.Filter{
		&vpc.Filter{
			Name:   common.StringPtr("vpc-id"),
			Values: common.StringPtrs([]string{reqVpcId}),
		},
	}

	// callLogStart := call.Start()
	response, err := VPCHandler.Client.DescribeSubnets(request)
	// callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	//cblogger.Debug(response.ToJsonString())
	cblogger.Debug(response)
	if err != nil {
		// callLogInfo.ErrorMSG = err.Error()
		// callogger.Error(call.String(callLogInfo))
		cblogger.Error(err)
		return nil, err
	}
	// callogger.Info(call.String(callLogInfo))

	for _, curSubnet := range response.Response.SubnetSet {
		cblogger.Infof("[%s] Check Subnet Information", *curSubnet.SubnetId)
		resSubnetInfo := irs.SubnetInfo{
			IId:       irs.IID{SystemId: *curSubnet.SubnetId, NameId: *curSubnet.SubnetName},
			IPv4_CIDR: *curSubnet.CidrBlock,
			//Status:    *subnetInfo.State,
		}

		keyValueList := []irs.KeyValue{
			{Key: "VpcId", Value: *curSubnet.VpcId},
			{Key: "IsDefault", Value: strconv.FormatBool(*curSubnet.IsDefault)},
			{Key: "AvailabilityZone", Value: *curSubnet.Zone},
		}
		resSubnetInfo.KeyValueList = keyValueList
		arrSubnetInfoList = append(arrSubnetInfoList, resSubnetInfo)
	}

	return arrSubnetInfoList, nil
}

// 동일 이름으로 생성되는 것을 막기 위해 중복 체크함.
// reqSubnetNameId : 서브넷 Name
func (VPCHandler *TencentVPCHandler) isExistSubnet(reqSubnetNameId string) (bool, error) {
	cblogger.Infof("reqSubnetNameId : [%s]", reqSubnetNameId)

	request := vpc.NewDescribeSubnetsRequest()
	request.Filters = []*vpc.Filter{
		&vpc.Filter{
			Name:   common.StringPtr("subnet-name"),
			Values: common.StringPtrs([]string{reqSubnetNameId}),
		},
	}

	//cblogger.Debug(request)
	response, err := VPCHandler.Client.DescribeSubnets(request)
	//cblogger.Debug("서브넷 실행 결과")
	//cblogger.Debug(response)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	if *response.Response.TotalCount < 1 {
		return false, nil
	}

	return true, nil
}

func (VPCHandler *TencentVPCHandler) AddSubnet(vpcIID irs.IID, subnetInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	cblogger.Infof("[%s] Add Subnet - CIDR : %s", subnetInfo.IId.NameId, subnetInfo.IPv4_CIDR)

	zoneId := VPCHandler.Region.Zone
	if subnetInfo.Zone != "" {
		zoneId = subnetInfo.Zone
	}
	cblogger.Infof("Zone : %s", zoneId)
	if zoneId == "" {
		cblogger.Error("Connection information does not contain Zone information.")
		return irs.VPCInfo{}, errors.New("Connection information does not contain Zone information.")
	}

	if subnetInfo.IId.NameId == "" {
		return irs.VPCInfo{}, errors.New("No SubnetId information to create.")
	}

	isExit, errSubnetInfo := VPCHandler.isExistSubnet(subnetInfo.IId.NameId)
	if errSubnetInfo != nil {
		cblogger.Error(errSubnetInfo)
		return irs.VPCInfo{}, errSubnetInfo
	}

	cblogger.Info("Subnet presence or absence : ")
	cblogger.Info(isExit)

	if isExit {
		cblogger.Errorf("[%S] returns an error with existing information without creating it because Subnet already exists.", subnetInfo.IId.NameId)
		return irs.VPCInfo{}, errors.New("InvalidVNetwork.Duplicate: The Subnet '" + subnetInfo.IId.NameId + "' already exists.")
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: vpcIID.SystemId,
		CloudOSAPI:   "CreateSubnet()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := vpc.NewCreateSubnetRequest()

	request.VpcId = common.StringPtr(vpcIID.SystemId)
	request.SubnetName = common.StringPtr(subnetInfo.IId.NameId)
	request.CidrBlock = common.StringPtr(subnetInfo.IPv4_CIDR)
	request.Zone = common.StringPtr(zoneId)

	var tags []*vpc.Tag
	for _, inputTag := range subnetInfo.TagList {
		tag := &vpc.Tag{
			Key:   &inputTag.Key,
			Value: &inputTag.Value,
		}
		tags = append(tags, tag)
	}

	request.Tags = tags

	callLogStart := call.Start()
	response, err := VPCHandler.Client.CreateSubnet(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	cblogger.Debug(response.ToJsonString())
	cblogger.Debug(response)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.VPCInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	retVpcInfo, errVpcInfo := VPCHandler.GetVPC(vpcIID)
	if errVpcInfo != nil {
		cblogger.Error(errVpcInfo)
		return irs.VPCInfo{}, err
	}

	//retVpcInfo.SubnetInfoList[0].IId.NameId = vpcReqInfo.IId.NameId // 생성 시에는 NameId는 요청 받은 값으로 리턴해야 함.

	return retVpcInfo, nil
}

func (VPCHandler *TencentVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger.Infof("[%s] Delete [%s] Subnet on VPC", vpcIID.SystemId, subnetIID.SystemId)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: vpcIID.SystemId,
		CloudOSAPI:   "DeleteSubnet()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := vpc.NewDeleteSubnetRequest()
	request.SubnetId = common.StringPtr(subnetIID.SystemId)

	callLogStart := call.Start()
	response, err := VPCHandler.Client.DeleteSubnet(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	cblogger.Debug(response.ToJsonString())
	//cblogger.Debug(response)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		cblogger.Error(err)
		return false, err
	}
	callogger.Info(call.String(callLogInfo))

	return true, nil
}
