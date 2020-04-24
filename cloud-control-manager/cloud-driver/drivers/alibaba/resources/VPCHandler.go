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

//VPC & Subnet 처리 (AlibabaCloud's Subnet --> VSwitch 임)
package resources

import (
	"errors"
	"reflect"
	"strconv"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/davecgh/go-spew/spew"

	//"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
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
	cblogger.Info(vpcReqInfo)

	request := vpc.CreateCreateVpcRequest()
	request.Scheme = "https"
	request.VpcName = vpcReqInfo.IId.NameId
	request.CidrBlock = vpcReqInfo.IPv4_CIDR

	response, err := VPCHandler.Client.CreateVpc(request)
	cblogger.Info(response)
	spew.Dump(response)
	if err != nil {
		cblogger.Error(err)
		return irs.VPCInfo{}, err
	}

	//==========================
	// Subnet 생성
	//==========================
	//VPCHandler.CreateSubnet(retVpcInfo.IId.SystemId, vpcReqInfo.SubnetInfoList[0])
	//var resSubnetList []irs.SubnetInfo
	for _, curSubnet := range vpcReqInfo.SubnetInfoList {
		cblogger.Infof("[%s] Subnet 생성", curSubnet.IId.NameId)
		resSubnet, errSubnet := VPCHandler.CreateSubnet(response.VpcId, curSubnet)

		cblogger.Info(resSubnet)
		if errSubnet != nil {
			return irs.VPCInfo{}, errSubnet
		}
		//resSubnetList = append(resSubnetList, resSubnet)
	}
	//retVpcInfo.SubnetInfoList = resSubnetList

	//생성된 VPC의 세부 정보를 조회함.
	retVpcInfo, errVpc := VPCHandler.GetVPC(irs.IID{SystemId: response.VpcId})
	if errVpc != nil {
		cblogger.Error(errVpc)
		return irs.VPCInfo{}, errVpc
	}
	retVpcInfo.IId.NameId = vpcReqInfo.IId.NameId // NameId는 요청 받은 값으로 리턴해야 함.

	return retVpcInfo, nil
}

func (VPCHandler *AlibabaVPCHandler) CreateSubnet(vpcId string, reqSubnetInfo irs.SubnetInfo) (irs.SubnetInfo, error) {
	cblogger.Info(reqSubnetInfo)

	/*
		vpcInfo, errVpcInfo := VPCHandler.GetSubnet(reqSubnetInfo.IId.SystemId)
		if errVpcInfo == nil {
			cblogger.Errorf("이미 [%S] Subnet이 존재하기 때문에 생성하지 않고 기존 정보와 함께 에러를 리턴함.", reqSubnetInfo.IId.SystemId)
			cblogger.Info(vpcInfo)
			return vpcInfo, errors.New("InvalidVNetwork.Duplicate: The Subnet '" + reqSubnetInfo.IId.SystemId + "' already exists.")
		}
	*/

	//서브넷 생성
	request := vpc.CreateCreateVSwitchRequest()
	request.Scheme = "https"
	request.VpcId = vpcId
	request.CidrBlock = reqSubnetInfo.IPv4_CIDR
	request.VSwitchName = reqSubnetInfo.IId.NameId
	request.ZoneId = VPCHandler.Region.Zone //"ap-northeast-1a" // @TOTO : ZoneId 전달 받아야 함.
	cblogger.Info(request)

	response, err := VPCHandler.Client.CreateVSwitch(request)
	cblogger.Info(response)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.SubnetInfo{}, err
	}
	spew.Dump(response)

	subnetInfo, errSunetInfo := VPCHandler.GetSubnet(response.VSwitchId)
	if errSunetInfo != nil {
		cblogger.Error(subnetInfo)
		return irs.SubnetInfo{}, errSunetInfo
	}

	return subnetInfo, nil
}

func (VPCHandler *AlibabaVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	cblogger.Debug("Start")

	request := vpc.CreateDescribeVpcsRequest()
	request.Scheme = "https"

	result, err := VPCHandler.Client.DescribeVpcs(request)
	spew.Dump(result)
	if err != nil {
		return nil, err
	}

	var vpcInfoList []*irs.VPCInfo
	for _, curVpc := range result.Vpcs.Vpc {
		cblogger.Infof("[%s] VPC 정보 조회", curVpc.VpcId)
		//vpcInfo := ExtractVpcDescribeInfo(&curVpc)
		vpcInfo, vpcErr := VPCHandler.GetVPC(irs.IID{SystemId: curVpc.VpcId})
		if vpcErr != nil {
			return nil, vpcErr
		}
		vpcInfoList = append(vpcInfoList, &vpcInfo)
	}

	spew.Dump(vpcInfoList)
	return vpcInfoList, nil
}

//VPC 정보를 추출함
func ExtractVpcDescribeInfo(vpcInfo *vpc.Vpc) irs.VPCInfo {
	aliVpcInfo := irs.VPCInfo{
		IId:       irs.IID{NameId: vpcInfo.VpcName, SystemId: vpcInfo.VpcId},
		IPv4_CIDR: vpcInfo.CidrBlock,
		//IsDefault: *vpcInfo.IsDefault,
		//State:     *vpcInfo.State,
	}

	keyValueList := []irs.KeyValue{
		{Key: "IsDefault", Value: strconv.FormatBool(vpcInfo.IsDefault)},
		{Key: "Status", Value: vpcInfo.Status},
		{Key: "VRouterId", Value: vpcInfo.VRouterId},
		{Key: "RegionId", Value: vpcInfo.RegionId},
	}
	aliVpcInfo.KeyValueList = keyValueList

	return aliVpcInfo
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

func (VPCHandler *AlibabaVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	spew.Dump(VPCHandler)
	cblogger.Info("VPC IID : ", vpcIID.SystemId)

	request := vpc.CreateDescribeVpcsRequest()
	request.Scheme = "https"
	request.VpcId = vpcIID.SystemId

	result, err := VPCHandler.Client.DescribeVpcs(request)
	spew.Dump(result)
	if err != nil {
		return irs.VPCInfo{}, err
	}

	cblogger.Info("VPC 개수 : ", len(result.Vpcs.Vpc))
	if len(result.Vpcs.Vpc) < 1 {
		return irs.VPCInfo{}, nil
	}

	vpcInfo := ExtractVpcDescribeInfo(&result.Vpcs.Vpc[0])
	spew.Dump(vpcInfo)

	//==========================
	// VPC의 서브넷들 처리
	//==========================
	var subnetInfoList []irs.SubnetInfo
	for _, curSubnet := range result.Vpcs.Vpc[0].VSwitchIds.VSwitchId {
		cblogger.Infof("[%s] VSwitch 정보 조회", curSubnet)
		subnetInfo, errSubnet := VPCHandler.GetSubnet(curSubnet)
		if errSubnet != nil {
			return irs.VPCInfo{}, errSubnet
		}
		subnetInfoList = append(subnetInfoList, subnetInfo)
	}
	spew.Dump(subnetInfoList)

	vpcInfo.SubnetInfoList = subnetInfoList
	return vpcInfo, nil
}

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
		cblogger.Infof("[%s] VSwitch 삭제 처리", curSubnet.IId.SystemId)
		_, errSubnet := VPCHandler.DeleteSubnet(curSubnet.IId)
		if errSubnet != nil {
			return false, errSubnet
		}
	}

	cblogger.Infof("[%s] VPC를 삭제 함.", vpcInfo.IId.SystemId)
	//cblogger.Info("VPC 제거를 위해 생성된 IGW / Route들 제거 시작")

	request := vpc.CreateDeleteVpcRequest()
	request.Scheme = "https"
	request.VpcId = vpcIID.SystemId

	response, err := VPCHandler.Client.DeleteVpc(request)
	cblogger.Info(response)
	if err != nil {
		cblogger.Infof("[%s] VPC Delete fail", vpcIID.SystemId)
		cblogger.Error(err.Error())
		return false, err
	}
	return true, nil
}

func (VPCHandler *AlibabaVPCHandler) DeleteSubnet(subnetIID irs.IID) (bool, error) {
	cblogger.Infof("Delete VSwitch : [%s]", subnetIID.SystemId)

	request := vpc.CreateDeleteVSwitchRequest()
	request.Scheme = "https"
	request.VSwitchId = subnetIID.SystemId

	response, err := VPCHandler.Client.DeleteVSwitch(request)
	cblogger.Info(response)
	if err != nil {
		cblogger.Infof("[%s] VSwitch Delete fail", subnetIID.SystemId)
		cblogger.Error(err.Error())
		return false, err
	}
	return true, nil
}

func (VPCHandler *AlibabaVPCHandler) GetSubnet(reqSubnetId string) (irs.SubnetInfo, error) {
	cblogger.Infof("SubnetId : [%s]", reqSubnetId)

	request := vpc.CreateDescribeVSwitchesRequest()
	request.Scheme = "https"

	result, err := VPCHandler.Client.DescribeVSwitches(request)
	spew.Dump(result)
	cblogger.Info(result)
	if err != nil {
		return irs.SubnetInfo{}, err
	}

	if !reflect.ValueOf(result.VSwitches.VSwitch).IsNil() {
		retSubnetInfo := ExtractSubnetDescribeInfo(result.VSwitches.VSwitch[0])
		return retSubnetInfo, nil
	} else {
		return irs.SubnetInfo{}, errors.New("InvalidVSwitch.NotFound: The '" + reqSubnetId + "' does not exist")
	}
}

//Subnet(VSwitch) 정보를 추출함
func ExtractSubnetDescribeInfo(subnetInfo vpc.VSwitch) irs.SubnetInfo {
	vNetworkInfo := irs.SubnetInfo{
		IId:       irs.IID{NameId: subnetInfo.VSwitchName, SystemId: subnetInfo.VSwitchId},
		IPv4_CIDR: subnetInfo.CidrBlock,
	}

	keyValueList := []irs.KeyValue{
		{Key: "Status", Value: subnetInfo.Status},
		{Key: "IsDefault", Value: strconv.FormatBool(subnetInfo.IsDefault)},
	}
	vNetworkInfo.KeyValueList = keyValueList

	return vNetworkInfo
}
