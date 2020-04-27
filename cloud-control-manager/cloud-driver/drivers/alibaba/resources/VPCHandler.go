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
	"strings"
	"time"

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

	//VPC를 생성하면 Pending 상태라서 Subnet을 추가할 수 없기 때문에 Available로 바뀔 때까지 대기함.
	VPCHandler.WaitForRun(response.VpcId)

	//==========================
	// Subnet 생성
	//==========================
	cblogger.Info("Subnet 생성 시작")
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
	//retVpcInfo.IId.NameId = vpcReqInfo.IId.NameId // NameId는 요청 받은 값으로 리턴해야 함.

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
	cblogger.Info("Start")

	request := vpc.CreateDescribeVpcsRequest()
	request.Scheme = "https"

	result, err := VPCHandler.Client.DescribeVpcs(request)
	cblogger.Debug(result)
	//spew.Dump(result)
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

	cblogger.Debug(result)
	//spew.Dump(vpcInfoList)
	return vpcInfoList, nil
}

//VPC 정보를 추출함
func ExtractVpcDescribeInfo(vpcInfo *vpc.Vpc) irs.VPCInfo {
	aliVpcInfo := irs.VPCInfo{
		IId:       irs.IID{NameId: vpcInfo.VpcName, SystemId: vpcInfo.VpcId},
		IPv4_CIDR: vpcInfo.CidrBlock,
	}

	keyValueList := []irs.KeyValue{
		{Key: "IsDefault", Value: strconv.FormatBool(vpcInfo.IsDefault)},
		{Key: "Status", Value: vpcInfo.Status},
		{Key: "VRouterId", Value: vpcInfo.VRouterId},
		{Key: "RegionId", Value: vpcInfo.RegionId},
	}
	aliVpcInfo.KeyValueList = keyValueList

	return aliVpcInfo
}

//Pending , Available
func (VPCHandler *AlibabaVPCHandler) WaitForRun(vpcId string) error {
	cblogger.Info("======> VPC가 Running 될 때까지 대기함.")

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
			cblogger.Error("VPC 상태가 Available이 아니라서 1초가 대기후 조회합니다.")
			time.Sleep(time.Second * 1)
			if curRetryCnt > maxRetryCnt {
				cblogger.Error("장시간 VPC의 Status 값이 Available로 변경되지 않아서 강제로 중단합니다.")
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

	result, err := VPCHandler.Client.DescribeVpcs(request)
	spew.Dump(result)
	if err != nil {
		return irs.VPCInfo{}, err
	}

	cblogger.Info("VPC 개수 : ", len(result.Vpcs.Vpc))
	if len(result.Vpcs.Vpc) < 1 {
		return irs.VPCInfo{}, errors.New("Not found")
	}

	vpcInfo := ExtractVpcDescribeInfo(&result.Vpcs.Vpc[0])
	spew.Dump(vpcInfo)

	//==========================
	// VPC의 서브넷들 처리
	//==========================
	var subnetInfoList []irs.SubnetInfo
	for _, curSubnet := range result.Vpcs.Vpc[0].VSwitchIds.VSwitchId {
		//cblogger.Infof("\n\n\n\n")
		//cblogger.Infof("---------------------------------------------------------------------")
		cblogger.Infof("[%s] VSwitch 정보 조회", curSubnet)
		subnetInfo, errSubnet := VPCHandler.GetSubnet(curSubnet)
		if errSubnet != nil {
			cblogger.Errorf("[%s] VSwitch 정보 조회 실패", curSubnet)
			cblogger.Error(errSubnet)
			return irs.VPCInfo{}, errSubnet
		}
		//cblogger.Infof("    =====> [%s] 조회 결과", curSubnet)
		//spew.Dump(subnetInfo)
		subnetInfoList = append(subnetInfoList, subnetInfo)
	}
	//cblogger.Info("===========> 서브넷 목록")
	//spew.Dump(subnetInfoList)

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
		cblogger.Infof("[%s] VSwitch 삭제 처리", curSubnet.IId.SystemId)
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
	request.VSwitchId = reqSubnetId

	result, err := VPCHandler.Client.DescribeVSwitches(request)
	spew.Dump(result)
	//cblogger.Info(result)
	if err != nil {
		cblogger.Error(err)
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
		{Key: "ZoneId", Value: subnetInfo.ZoneId},
	}
	vNetworkInfo.KeyValueList = keyValueList

	return vNetworkInfo
}
