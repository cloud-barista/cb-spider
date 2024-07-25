// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by zephy@mz.co.kr, 2019.09.
// by devunet@mz.co.kr, 2020.04.

// VPC & Subnet 처리 (AlibabaCloud's Subnet --> VSwitch 임)
package resources

import (
	"errors"
	"reflect"
	"strconv"
	"strings"
	"time"

	//vpc "github.com/alibabacloud-go/vpc-20160428/v6/client"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	/*
		"github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
		"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
		idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
		irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
		"github.com/davecgh/go-spew/spew"
	*/)

type AlibabaVPCHandler struct {
	Region idrv.RegionInfo
	Client *vpc.Client
}

func (VPCHandler *AlibabaVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	cblogger.Debug(vpcReqInfo)

	request := vpc.CreateCreateVpcRequest()
	request.Scheme = "https"
	request.VpcName = vpcReqInfo.IId.NameId
	request.CidrBlock = vpcReqInfo.IPv4_CIDR

	// tag 받아서 배열로 추가

	if vpcReqInfo.TagList != nil && len(vpcReqInfo.TagList) > 0 {

		vpcTags := []vpc.CreateVpcTag{}
		for _, vpcTag := range vpcReqInfo.TagList {
			tag0 := vpc.CreateVpcTag{
				Key:   vpcTag.Key,
				Value: vpcTag.Value,
			}
			vpcTags = append(vpcTags, tag0)

		}
		request.Tag = &vpcTags
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: vpcReqInfo.IId.NameId,
		CloudOSAPI:   "CreateVpc()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	response, err := VPCHandler.Client.CreateVpc(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Info(response)
	//cblogger.Debug(response)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.VPCInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	//VPC를 생성하면 Pending 상태라서 Subnet을 추가할 수 없기 때문에 Available로 바뀔 때까지 대기함.
	err = VPCHandler.WaitForRun(response.VpcId)
	if err != nil {
		cblogger.Error(err)
		return irs.VPCInfo{}, err
	}

	//==========================
	// Subnet 생성
	//==========================
	cblogger.Info("Subnet creation started")
	//var resSubnetList []irs.SubnetInfo
	for _, curSubnet := range vpcReqInfo.SubnetInfoList {
		cblogger.Infof("[%s] Subnet 생성", curSubnet.IId.NameId)
		resSubnet, errSubnet := VPCHandler.CreateSubnet(response.VpcId, curSubnet)

		cblogger.Info(resSubnet)
		if errSubnet != nil {
			return irs.VPCInfo{}, errSubnet
		}
	}

	//생성된 Subnet을 포함한 VPC의 최신 정보를 조회함.
	retVpcInfo, errVpc := VPCHandler.GetVPC(irs.IID{SystemId: response.VpcId})
	if errVpc != nil {
		cblogger.Error(errVpc)
		return irs.VPCInfo{}, errVpc
	}
	retVpcInfo.IId.NameId = vpcReqInfo.IId.NameId // NameId는 요청 받은 값으로 리턴해야 함.

	// if vpcReqInfo.TagList != nil && len(vpcReqInfo.TagList) > 0 {
	// 	//tagHandler.Client, tagHandler.Region, resType, resIID, key
	// 	for _, vpcTag := range vpcReqInfo.TagList {
	// 		response, err := AddVpcTags(VPCHandler.Client, VPCHandler.Region, irs.RSType("VPC"), retVpcInfo.IId, vpcTag)
	// 		cblogger.Info("AddVpcTage", response)
	// 		if err != nil {
	// 			cblogger.Error(errVpc)
	// 		}
	// 		cblogger.Debug("vpc add tag response ", response)
	// 	}
	// }
	return retVpcInfo, nil
}

func (VPCHandler *AlibabaVPCHandler) CreateSubnet(vpcId string, reqSubnetInfo irs.SubnetInfo) (irs.SubnetInfo, error) {
	cblogger.Debug(reqSubnetInfo)

	/*
		vpcInfo, errVpcInfo := VPCHandler.GetSubnet(reqSubnetInfo.IId.SystemId)
		if errVpcInfo == nil {
			cblogger.Errorf("이미 [%S] Subnet이 존재하기 때문에 생성하지 않고 기존 정보와 함께 에러를 리턴함.", reqSubnetInfo.IId.SystemId)
			cblogger.Info(vpcInfo)
			return vpcInfo, errors.New("InvalidVNetwork.Duplicate: The Subnet '" + reqSubnetInfo.IId.SystemId + "' already exists.")
		}
	*/

	zoneId := VPCHandler.Region.Zone
	if reqSubnetInfo.Zone != "" {
		zoneId = reqSubnetInfo.Zone
	}
	//서브넷 생성
	request := vpc.CreateCreateVSwitchRequest()
	request.Scheme = "https"
	request.VpcId = vpcId
	request.CidrBlock = reqSubnetInfo.IPv4_CIDR
	request.VSwitchName = reqSubnetInfo.IId.NameId
	request.ZoneId = zoneId
	// 0717 tag 추가

	if reqSubnetInfo.TagList != nil && len(reqSubnetInfo.TagList) > 0 {

		subnetTags := []vpc.CreateVSwitchTag{}
		for _, vpcTag := range reqSubnetInfo.TagList {
			tag0 := vpc.CreateVSwitchTag{
				Key:   vpcTag.Key,
				Value: vpcTag.Value,
			}
			subnetTags = append(subnetTags, tag0)

		}
		request.Tag = &subnetTags
	}

	/////

	cblogger.Info(request)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   zoneId,
		ResourceType: call.VPCSUBNET,
		ResourceName: reqSubnetInfo.IId.NameId,
		CloudOSAPI:   "CreateVSwitch()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	response, err := VPCHandler.Client.CreateVSwitch(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Info(response)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		cblogger.Error(err.Error())
		return irs.SubnetInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))
	//cblogger.Debug(response)

	subnetInfo, errSunetInfo := VPCHandler.GetSubnet(response.VSwitchId)
	if errSunetInfo != nil {
		cblogger.Error(subnetInfo)
		return irs.SubnetInfo{}, errSunetInfo
	}

	return subnetInfo, nil
}

func (VPCHandler *AlibabaVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	cblogger.Info("Start")

	request := vpc.CreateDescribeVpcsRequest()
	request.Scheme = "https"

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: "List()",
		CloudOSAPI:   "DescribeVpcs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := VPCHandler.Client.DescribeVpcs(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Debug(result)
	//cblogger.Debug(result)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		return nil, err
	}
	callogger.Info(call.String(callLogInfo))

	var vpcInfoList []*irs.VPCInfo
	for _, curVpc := range result.Vpcs.Vpc {
		cblogger.Infof("[%s] VPC information retrieval", curVpc.VpcId)
		//vpcInfo := ExtractVpcDescribeInfo(&curVpc)
		vpcInfo, vpcErr := VPCHandler.GetVPC(irs.IID{SystemId: curVpc.VpcId})
		if vpcErr != nil {
			return nil, vpcErr
		}
		vpcInfoList = append(vpcInfoList, &vpcInfo)
	}

	cblogger.Debug(result)
	//cblogger.Debug(vpcInfoList)
	return vpcInfoList, nil
}

// VPC 정보를 추출함
func ExtractVpcDescribeInfo(vpcInfo *vpc.Vpc) irs.VPCInfo {
	aliVpcInfo := irs.VPCInfo{
		IId:       irs.IID{NameId: vpcInfo.VpcName, SystemId: vpcInfo.VpcId},
		IPv4_CIDR: vpcInfo.CidrBlock,
	}

	tagList := []irs.KeyValue{}
	for _, aliTag := range vpcInfo.Tags.Tag {
		sTag := irs.KeyValue{}
		sTag.Key = aliTag.Key
		sTag.Value = aliTag.Value

		tagList = append(tagList, sTag)
	}
	aliVpcInfo.TagList = tagList

	keyValueList := []irs.KeyValue{
		{Key: "IsDefault", Value: strconv.FormatBool(vpcInfo.IsDefault)},
		{Key: "Status", Value: vpcInfo.Status},
		{Key: "VRouterId", Value: vpcInfo.VRouterId},
		{Key: "RegionId", Value: vpcInfo.RegionId},
	}
	aliVpcInfo.KeyValueList = keyValueList

	return aliVpcInfo
}

// Pending , Available
func (VPCHandler *AlibabaVPCHandler) WaitForRun(vpcId string) error {
	cblogger.Debug("======> Waiting until VPC is running.")

	maxRetryCnt := 20
	curRetryCnt := 0
	status := ""
	request := vpc.CreateDescribeVpcsRequest()
	request.Scheme = "https"
	request.VpcId = vpcId

	for {
		result, err := VPCHandler.Client.DescribeVpcs(request)
		if err != nil {
			return err
		}

		if len(result.Vpcs.Vpc) < 1 {
			return errors.New("Not found")
		}

		status = result.Vpcs.Vpc[0].Status
		cblogger.Info("===>VPC Status : ", status)
		if strings.EqualFold(status, "Pending") {
			curRetryCnt++
			cblogger.Debug("Waiting for 1 second and then querying because the VPC status is not Available.")
			time.Sleep(time.Second * 1)
			if curRetryCnt > maxRetryCnt {
				cblogger.Error("Forcing termination as the VPC status remains unchanged as Available for a long time.")
			}
		} else {
			if strings.EqualFold(status, "Available") {
				break
			} else {
				return errors.New("Unknown VPC Status value.")
			}
		}
	}

	return nil
}

func (VPCHandler *AlibabaVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	cblogger.Info("VPC IID : ", vpcIID.SystemId)

	request := vpc.CreateDescribeVpcsRequest()
	request.Scheme = "https"
	request.VpcId = vpcIID.SystemId

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: vpcIID.SystemId,
		CloudOSAPI:   "DescribeVpcs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := VPCHandler.Client.DescribeVpcs(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Debug(result)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		return irs.VPCInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Info("VPC Count : ", len(result.Vpcs.Vpc))
	//if result.TotalCount < 1 {
	if len(result.Vpcs.Vpc) < 1 {
		return irs.VPCInfo{}, errors.New("Notfound: '" + vpcIID.SystemId + "' VPC Not found")
	}

	vpcInfo := ExtractVpcDescribeInfo(&result.Vpcs.Vpc[0])
	//cblogger.Debug(vpcInfo)

	//==========================
	// VPC의 서브넷들 처리
	//==========================
	var subnetInfoList []irs.SubnetInfo
	for _, curSubnet := range result.Vpcs.Vpc[0].VSwitchIds.VSwitchId {
		//cblogger.Infof("\n\n\n\n")
		//cblogger.Infof("---------------------------------------------------------------------")
		cblogger.Infof("[%s] VSwitch Information retrieval", curSubnet)
		subnetInfo, errSubnet := VPCHandler.GetSubnet(curSubnet)
		if errSubnet != nil {
			cblogger.Errorf("[%s] VSwitch Information retrieval failed", curSubnet)
			cblogger.Error(errSubnet)
			return irs.VPCInfo{}, errSubnet
		}
		//cblogger.Infof("    =====> [%s] 조회 결과", curSubnet)
		//cblogger.Debug(subnetInfo)
		subnetInfoList = append(subnetInfoList, subnetInfo)
	}
	//cblogger.Info("===========> 서브넷 목록")
	//cblogger.Debug(subnetInfoList)

	vpcInfo.SubnetInfoList = subnetInfoList
	return vpcInfo, nil
}

//@TODO : 라우트 삭제 로직이 없어서 VPC가 삭제 안되는 현상이 있어서 라우트 정보를 조회해서 삭제하려다 서브넷 삭제 후 특정 시간 이후에 Route가 자동으로 삭제되기 때문에 임시로 4초 대기 후 VPC를 삭제하도록 로직을 변경함.
//@TODO : VPCHandler로 생성하지 않은 VPC의 경우 다른 서비스가 있을 수 있기 때문에 관련 서비스들을 조회후 삭제하는 로직이 필요할 수 있음.
/*
  - 삭제 오류
	자동 할당된 Route가 남아있어서 삭제가 안되는 듯.
	ErrorCode: Forbbiden
	Recommend:
	RequestId: 8871BF19-330B-4F00-93ED-D886F2CE066F
	Message: Active custom route in vpc.)
*/
func (VPCHandler *AlibabaVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	cblogger.Infof("Delete VPC : [%s]", vpcIID.SystemId)

	//Subnet 등으 연계된 인프라 제거를 위해 VPC 정보를 조회함.
	vpcInfo, errVpcInfo := VPCHandler.GetVPC(vpcIID)
	if errVpcInfo != nil {
		return false, errVpcInfo
	}

	//=================
	// Subnet삭제
	//=================
	for _, curSubnet := range vpcInfo.SubnetInfoList {
		cblogger.Infof("[%s] VSwitch deletion processing", curSubnet.IId.SystemId)
		_, errSubnet := VPCHandler.DeleteSubnet(curSubnet.IId)
		if errSubnet != nil {
			return false, errSubnet
		}
	}

	//=====================
	// 라우트를 제거해야 삭제 가능 함.
	//=================
	//특정 시간 이후 자동 삭제되니 라우트 삭제 대신 3초 대기후 시도해 봄.
	time.Sleep(time.Second * 3)

	cblogger.Infof("[%s] Deleting VPC.", vpcInfo.IId.SystemId)
	//cblogger.Info("VPC 제거를 위해 생성된 IGW / Route들 제거 시작")

	request := vpc.CreateDeleteVpcRequest()
	request.Scheme = "https"
	request.VpcId = vpcIID.SystemId

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: vpcIID.SystemId,
		CloudOSAPI:   "DeleteVpc()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	response, err := VPCHandler.Client.DeleteVpc(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Info(response)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Infof("[%s] VPC Delete fail", vpcIID.SystemId)
		cblogger.Error(err.Error())
		return false, err
	}
	callogger.Info(call.String(callLogInfo))
	return true, nil
}

func (VPCHandler *AlibabaVPCHandler) DeleteSubnet(subnetIID irs.IID) (bool, error) {
	cblogger.Infof("Delete VSwitch : [%s]", subnetIID.SystemId)

	request := vpc.CreateDeleteVSwitchRequest()
	request.Scheme = "https"
	request.VSwitchId = subnetIID.SystemId

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: subnetIID.SystemId,
		CloudOSAPI:   "DeleteVSwitch()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	response, err := VPCHandler.Client.DeleteVSwitch(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Info(response)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Infof("[%s] VSwitch Delete fail", subnetIID.SystemId)
		cblogger.Error(err.Error())
		return false, err
	}
	callogger.Info(call.String(callLogInfo))
	return true, nil
}

func (VPCHandler *AlibabaVPCHandler) GetSubnet(reqSubnetId string) (irs.SubnetInfo, error) {
	cblogger.Infof("SubnetId : [%s]", reqSubnetId)

	request := vpc.CreateDescribeVSwitchesRequest()
	request.Scheme = "https"
	request.VSwitchId = reqSubnetId

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: reqSubnetId,
		CloudOSAPI:   "DescribeVSwitches()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := VPCHandler.Client.DescribeVSwitches(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	//cblogger.Debug(result)
	//cblogger.Info(result)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.SubnetInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	if result.TotalCount < 1 {
		return irs.SubnetInfo{}, errors.New("Notfound: '" + reqSubnetId + "' Subnet Not found")
	}

	if !reflect.ValueOf(result.VSwitches.VSwitch).IsNil() {
		retSubnetInfo := ExtractSubnetDescribeInfo(result.VSwitches.VSwitch[0])
		return retSubnetInfo, nil
	} else {
		return irs.SubnetInfo{}, errors.New("InvalidVSwitch.NotFound: The '" + reqSubnetId + "' does not exist")
	}
}

// Subnet(VSwitch) 정보를 추출함
func ExtractSubnetDescribeInfo(subnetInfo vpc.VSwitch) irs.SubnetInfo {
	vNetworkInfo := irs.SubnetInfo{
		IId:       irs.IID{NameId: subnetInfo.VSwitchName, SystemId: subnetInfo.VSwitchId},
		IPv4_CIDR: subnetInfo.CidrBlock,
		Zone:      subnetInfo.ZoneId,
	}
	/////
	// 0717 tag 추출

	tagList := []irs.KeyValue{}
	for _, aliTag := range subnetInfo.Tags.Tag {
		sTag := irs.KeyValue{}
		sTag.Key = aliTag.Key
		sTag.Value = aliTag.Value

		tagList = append(tagList, sTag)
	}
	vNetworkInfo.TagList = tagList

	/////

	keyValueList := []irs.KeyValue{
		{Key: "Status", Value: subnetInfo.Status},
		{Key: "IsDefault", Value: strconv.FormatBool(subnetInfo.IsDefault)},
		{Key: "ZoneId", Value: subnetInfo.ZoneId},
	}
	vNetworkInfo.KeyValueList = keyValueList

	return vNetworkInfo
}

func (VPCHandler *AlibabaVPCHandler) AddSubnet(vpcIID irs.IID, subnetInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	cblogger.Infof("[%s] Subnet 추가 - CIDR : %s", subnetInfo.IId.NameId, subnetInfo.IPv4_CIDR)
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   VPCHandler.Region.Zone,
		ResourceType: call.VPCSUBNET,
		ResourceName: vpcIID.SystemId,
		CloudOSAPI:   "CreateSubnet()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	resSubnet, errSubnet := VPCHandler.CreateSubnet(vpcIID.SystemId, subnetInfo)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if errSubnet != nil {
		callLogInfo.ErrorMSG = errSubnet.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(errSubnet)
		return irs.VPCInfo{}, errSubnet
	}
	callogger.Info(call.String(callLogInfo))
	cblogger.Info(resSubnet)

	//#330이슈 처리
	vpcInfo, errVpcInfo := VPCHandler.GetVPC(vpcIID)
	if errVpcInfo != nil {
		cblogger.Error(errVpcInfo)
		return irs.VPCInfo{}, errVpcInfo
	}

	findSubnet := false
	cblogger.Debug("============== Checking Values =========")
	for posSubnet, curSubnetInfo := range vpcInfo.SubnetInfoList {
		cblogger.Debugf("%d - [%s] Subnet processing started", posSubnet, curSubnetInfo.IId.SystemId)
		if resSubnet.IId.SystemId == curSubnetInfo.IId.SystemId {
			cblogger.Infof("Discovered additional requested [%s] Subnet. - SystemID:[%s]", subnetInfo.IId.NameId, curSubnetInfo.IId.SystemId)
			//for ~ range는 포인터가 아니라서 값 수정이 안됨. for loop으로 직접 서브넷을 체크하거나 vpcInfo의 배열의 값을 수정해야 함.
			cblogger.Infof("Index position: %d", posSubnet)
			//vpcInfo.SubnetInfoList[posSubnet].IId.NameId = "테스트~"
			vpcInfo.SubnetInfoList[posSubnet].IId.NameId = subnetInfo.IId.NameId
			findSubnet = true
			break
		}
	}

	if !findSubnet {
		cblogger.Errorf("Subnet creation was successful, but the information [%s] requested for the [%s] subnet could not be found in the VPC's subnet list.", subnetInfo.IId.NameId, resSubnet.IId.SystemId)
		return irs.VPCInfo{}, errors.New("MismatchSubnet.NotFound: No SysmteId[" + resSubnet.IId.SystemId + "] found for newly created Subnet[" + subnetInfo.IId.NameId + "].")
	}

	return vpcInfo, nil

	//return irs.VPCInfo{}, nil
}

func (VPCHandler *AlibabaVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger.Infof("[%s] VPC의 [%s] Subnet Delete", vpcIID.SystemId, subnetIID.SystemId)

	return VPCHandler.DeleteSubnet(subnetIID)
}
