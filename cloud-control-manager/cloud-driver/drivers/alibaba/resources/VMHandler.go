// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// EC2 Hander (AWS SDK GO Version 1.16.26, Thanks AWS.)
//
// by zephy@mz.co.kr, 2019.09.

package resources

import (
	"errors"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	/*
		"github.com/sirupsen/logrus"
		"reflect"
		"strings"
		"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
		"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
		cblog "github.com/cloud-barista/cb-log"
		idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
		irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
		"github.com/davecgh/go-spew/spew"
	*/)

type AlibabaVMHandler struct {
	Region idrv.RegionInfo
	Client *ecs.Client
}

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("ALIBABA VMHandler")
}

// 1개의 VM만 생성되도록 수정 (MinCount / MaxCount 이용 안 함)
//키페어 이름(예:mcloud-barista)은 아래 URL에 나오는 목록 중 "키페어 이름"의 값을 적으면 됨.
//https://ap-northeast-2.console.aws.amazon.com/ec2/v2/home?region=ap-northeast-2#KeyPairs:sort=keyName

// @TODO : PublicIp 요금제 방식과 대역폭 설정 방법 논의 필요
func (vmHandler *AlibabaVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	//cblogger.Info(vmReqInfo)
	spew.Dump(vmReqInfo)

	//=============================
	// 보안그룹 처리 - SystemId 기반
	//=============================
	cblogger.Info("SystemId 기반으로 처리하기 위해 IID 기반의 보안그룹 배열을 SystemId 기반 보안그룹 배열로 조회및 변환함.")
	var newSecurityGroupIds []string
	//var firstSecurityGroupId string

	for _, sgId := range vmReqInfo.SecurityGroupIIDs {
		cblogger.Infof("보안그룹 변환 : [%s]", sgId)
		newSecurityGroupIds = append(newSecurityGroupIds, sgId.SystemId)
		//firstSecurityGroupId = sgId.SystemId
		//break
	}

	cblogger.Info("보안그룹 변환 완료")
	cblogger.Info(newSecurityGroupIds)

	//request := ecs.CreateCreateInstanceRequest()	// CreateInstance는 PublicIp가 자동으로 할당되지 않음.
	request := ecs.CreateRunInstancesRequest() // RunInstances는 PublicIp가 자동으로 할당됨.
	request.Scheme = "https"

	request.InstanceChargeType = "PostPaid" //저렴한 실시간 요금으로 설정 //PrePaid: subscription.  / PostPaid: pay-as-you-go. Default value: PostPaid.
	request.ImageId = vmReqInfo.ImageIID.SystemId
	//request.SecurityGroupIds *[]string
	request.SecurityGroupIds = &newSecurityGroupIds
	//request.SecurityGroupId = firstSecurityGroupId // string 타입이라 첫번째 보안 그룹만 적용
	//request.SecurityGroupId =  "[\"" + newSecurityGroupIds + "\"]" // string 타입이라 첫번째 보안 그룹만 적용

	request.InstanceName = vmReqInfo.IId.NameId
	//request.HostName = vmReqInfo.IId.NameId	// OS 호스트 명
	request.InstanceType = vmReqInfo.VMSpecName
	request.KeyPairName = vmReqInfo.KeyPairIID.SystemId
	request.VSwitchId = vmReqInfo.SubnetIID.SystemId

	//==============
	//PublicIp 설정
	//==============
	//Public Ip를 생성하기 위해서는 과금형태와 대역폭(1 Mbit/s이상)을 지정해야 함.
	//PayByTraffic(기본값) : 트래픽 기준 결제(GB 단위) - 트래픽 기준 결제(GB 단위)를 사용하면 대역폭 사용료가 시간별로 청구
	//PayByBandwidth : 대역폭 사용료는 구독 기반이고 ECS 인스턴스 사용료에 포함 됨.
	request.InternetChargeType = "PayByBandwidth"           //Public Ip요금 방식을 1시간 단위(PayByBandwidth) 요금으로 설정 / PayByTraffic(기본값) : 1GB단위 시간당 트래픽 요금 청구
	request.InternetMaxBandwidthOut = requests.Integer("5") // 0보다 크면 Public IP가 할당 됨 - 최대 아웃 바운드 공용 대역폭 단위 : Mbit / s 유효한 값 : 0 ~ 100
	spew.Dump(request)

	//=============================
	// VM생성 처리
	//=============================
	cblogger.Info("Create EC2 Instance")
	cblogger.Info(request)

	//response, err := vmHandler.Client.CreateInstance(request)
	response, err := vmHandler.Client.RunInstances(request)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.VMInfo{}, err
	}
	spew.Dump(response)

	if len(response.InstanceIdSets.InstanceIdSet) < 1 {
		return irs.VMInfo{}, errors.New("No errors have occurred, but no VMs have been created.")
	}

	if 1 == 1 {
		return irs.VMInfo{}, nil
	}

	//vmInfo, errVmInfo := vmHandler.GetVM(irs.IID{SystemId: response.InstanceId})
	vmInfo, errVmInfo := vmHandler.GetVM(irs.IID{SystemId: response.InstanceIdSets.InstanceIdSet[0]})
	if errVmInfo != nil {
		cblogger.Error(errVmInfo.Error())
		return irs.VMInfo{}, errVmInfo
	}
	vmInfo.IId.NameId = vmReqInfo.IId.NameId
	return vmInfo, nil
}

func (vmHandler *AlibabaVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmID : [%s]", vmIID.SystemId)

	request := ecs.CreateStartInstanceRequest()
	request.Scheme = "https"
	request.InstanceId = vmIID.SystemId

	response, err := vmHandler.Client.StartInstance(request)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.VMStatus("Failed"), err
	}
	cblogger.Info(response)
	return irs.VMStatus("Resuming"), nil

}

// @TODO - 이슈 : 인스턴스 일시정지 시에 과금 정책을 결정해야 함 - StopCharging / KeepCharging
func (vmHandler *AlibabaVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmID : [%s]", vmIID.SystemId)

	request := ecs.CreateStopInstanceRequest()
	request.Scheme = "https"
	request.InstanceId = vmIID.SystemId
	request.StoppedMode = "StopCharging"

	response, err := vmHandler.Client.StopInstance(request)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.VMStatus("Failed"), err
	}
	cblogger.Info(response)
	return irs.VMStatus("Suspending"), nil
}

func (vmHandler *AlibabaVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmID : [%s]", vmIID.SystemId)

	request := ecs.CreateRebootInstanceRequest()
	request.Scheme = "https"
	request.InstanceId = vmIID.SystemId

	response, err := vmHandler.Client.RebootInstance(request)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.VMStatus("Failed"), err
	}
	cblogger.Info(response)
	return irs.VMStatus("Rebooting"), nil
}

func (vmHandler *AlibabaVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmID : [%s]", vmIID.SystemId)

	cblogger.Infof("VM을 종료하기 위해 Suspend 모드로 실행합니다.")
	//Terminate하려면 VM이 Running 상태면 안됨.
	sus, errSus := vmHandler.SuspendVM(vmIID)
	if errSus != nil {
		cblogger.Error(errSus.Error())
		return irs.VMStatus("Failed"), errSus
	}

	if sus != "Suspending" {
		cblogger.Errorf("[%s] VM의 Suspend 모드 실행 결과[%s]가 Suspending이 아닙니다.", vmIID.SystemId, sus)
		return irs.VMStatus("Failed"), errors.New(vmIID.SystemId + " VM의 Suspend 모드 실행 결과 가 Suspending이 아닙니다.")
	}

	//===================================
	// Suspending 되도록 3초 정도 대기 함.
	//===================================
	curRetryCnt := 0
	maxRetryCnt := 10
	for {
		curStatus, errStatus := vmHandler.GetVMStatus(vmIID)
		if errStatus != nil {
			cblogger.Error(errStatus.Error())
		}

		cblogger.Info("===>VM Status : ", curStatus)
		if curStatus != irs.VMStatus("Suspended") {
			curRetryCnt++
			cblogger.Error("VM 상태가 Suspended가 아니라서 1초가 대기후 조회합니다.")
			time.Sleep(time.Second * 1)
			if curRetryCnt > maxRetryCnt {
				cblogger.Error("장시간 대기해도 VM의 Status 값이 Suspended로 변경되지 않아서 강제로 중단합니다.")
			}
		} else {
			break
		}
	}

	request := ecs.CreateDeleteInstanceRequest()
	request.Scheme = "https"
	request.InstanceId = vmIID.SystemId

	response, err := vmHandler.Client.DeleteInstance(request)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.VMStatus("Failed"), err
	}
	cblogger.Info(response)
	return irs.VMStatus("Terminating"), nil
}

func (vmHandler *AlibabaVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
	cblogger.Infof("vmID : [%s]", vmIID.SystemId)

	request := ecs.CreateDescribeInstancesRequest()
	request.Scheme = "https"
	request.InstanceIds = "[\"" + vmIID.SystemId + "\"]"

	response, err := vmHandler.Client.DescribeInstances(request)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.VMInfo{}, err
	}
	spew.Dump(response)

	if response.TotalCount < 1 {
		return irs.VMInfo{}, errors.New("Notfound: '" + vmIID.SystemId + "' VM Not found")
	}

	//	vmInfo := vmHandler.ExtractDescribeInstances(response.Instances.Instance[0])
	vmInfo := vmHandler.ExtractDescribeInstances(&response.Instances.Instance[0])
	cblogger.Info("vmInfo", vmInfo)
	return vmInfo, nil
}

//@TODO : 2020-03-26 Ali클라우드 API 구조가 바뀐 것 같아서 임시로 변경해 놓음.
//func (vmHandler *AlibabaVMHandler) ExtractDescribeInstances() irs.VMInfo {
func (vmHandler *AlibabaVMHandler) ExtractDescribeInstances(instancInfo *ecs.Instance) irs.VMInfo {
	cblogger.Info(instancInfo)

	//time.Parse(layout, str)
	vmInfo := irs.VMInfo{
		IId:        irs.IID{NameId: instancInfo.InstanceName, SystemId: instancInfo.InstanceId},
		ImageIId:   irs.IID{SystemId: instancInfo.ImageId},
		VMSpecName: instancInfo.InstanceType,
		KeyPairIId: irs.IID{SystemId: instancInfo.KeyPairName},
		//StartTime:  instancInfo.StartTime,

		Region:    irs.RegionInfo{Region: instancInfo.RegionId, Zone: instancInfo.ZoneId}, //  ex) {us-east1, us-east1-c} or {ap-northeast-2}
		VpcIID:    irs.IID{SystemId: instancInfo.VpcAttributes.VpcId},
		SubnetIID: irs.IID{SystemId: instancInfo.VpcAttributes.VSwitchId},
		//SecurityGroupIIds []IID // AWS, ex) sg-0b7452563e1121bb6
		//NetworkInterface string // ex) eth0
		//PublicDNS
		//PrivateIP
		PrivateIP: instancInfo.VpcAttributes.PrivateIpAddress.IpAddress[0],
		//PrivateDNS

		//VMBootDisk  string // ex) /dev/sda1
		//VMBlockDisk string // ex)

		KeyValueList: []irs.KeyValue{{Key: "", Value: ""}},
	}

	if len(instancInfo.PublicIpAddress.IpAddress) > 0 {
		vmInfo.PublicIP = instancInfo.PublicIpAddress.IpAddress[0]
	}

	for _, security := range instancInfo.SecurityGroupIds.SecurityGroupId {
		//vmInfo.SecurityGroupIds = append(vmInfo.SecurityGroupIds, *security.GroupId)
		vmInfo.SecurityGroupIIds = append(vmInfo.SecurityGroupIIds, irs.IID{SystemId: security})
	}

	timeLen := len(instancInfo.CreationTime)
	cblogger.Infof("서버 구동 시간 포멧 변환 처리")
	cblogger.Infof("======> 생성시간 길이 [%s]", timeLen)
	if timeLen > 7 {
		cblogger.Infof("======> 생성시간 마지막 문자열 [%s]", instancInfo.CreationTime[timeLen-1:])
		var NewStartTime string
		if instancInfo.CreationTime[timeLen-1:] == "Z" && timeLen == 17 {
			//cblogger.Infof("======> 문자열 변환 : [%s]", StartTime[:timeLen-1])
			NewStartTime = instancInfo.CreationTime[:timeLen-1] + ":00Z"
			cblogger.Infof("======> 최종 문자열 변환 : [%s]", NewStartTime)
		} else {
			NewStartTime = instancInfo.CreationTime
		}

		cblogger.Infof("Convert StartTime string [%s] to time.time", NewStartTime)

		//layout := "2020-05-07T01:36Z"
		t, err := time.Parse(time.RFC3339, NewStartTime)
		if err != nil {
			cblogger.Error(err)
		} else {
			cblogger.Infof("======> [%v]", t)
			vmInfo.StartTime = t
		}
	}

	return vmInfo
}

func (vmHandler *AlibabaVMHandler) ListVM() ([]*irs.VMInfo, error) {
	cblogger.Infof("Start")

	request := ecs.CreateDescribeInstancesRequest()
	request.Scheme = "https"

	response, err := vmHandler.Client.DescribeInstances(request)
	if err != nil {
		cblogger.Error(err.Error())
		return nil, err
	}
	spew.Dump(response)

	var vmInfoList []*irs.VMInfo
	for _, curInstance := range response.Instances.Instance {

		cblogger.Info("[%s] ECS 정보 조회", curInstance.InstanceId)
		vmInfo, errVmInfo := vmHandler.GetVM(irs.IID{SystemId: curInstance.InstanceId})
		if errVmInfo != nil {
			cblogger.Error(errVmInfo.Error())
			return nil, errVmInfo
		}
		//cblogger.Info("=======>VM 조회 결과")
		spew.Dump(vmInfo)

		vmInfoList = append(vmInfoList, &vmInfo)
	}

	//cblogger.Info("=======>VM 최종 목록결과")
	spew.Dump(vmInfoList)
	//cblogger.Info("=======>VM 목록 완료")
	return vmInfoList, nil
}

//SHUTTING-DOWN / TERMINATED
func (vmHandler *AlibabaVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	vmID := vmIID.SystemId
	cblogger.Infof("vmID : [%s]", vmID)

	request := ecs.CreateDescribeInstanceStatusRequest()
	request.Scheme = "https"
	request.InstanceId = &[]string{vmIID.SystemId}
	cblogger.Infof("request : [%v]", request)

	response, err := vmHandler.Client.DescribeInstanceStatus(request)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.VMStatus("Failed"), err
	}

	cblogger.Info("Success", response)
	if response.TotalCount < 1 {
		//return irs.VMStatus("Failed"), errors.New("Notfound: '" + vmIID.SystemId + "' VM Not found")
		return irs.VMStatus("NotExist"), nil
	}

	for _, vm := range response.InstanceStatuses.InstanceStatus {
		//vmStatus := strings.ToUpper(vm.Status)
		cblogger.Infof("Req VM:[%s] / Cur VM:[%s] / ECS Status : [%s]", vmID, vm.InstanceId, vm.Status)
		vmStatus, errStatus := vmHandler.ConvertVMStatusString(vm.Status)
		if errStatus != nil {
			cblogger.Error(errStatus.Error())
			return irs.VMStatus("Failed"), errStatus
		}
		return vmStatus, errStatus
	}

	return irs.VMStatus("Failed"), errors.New("No status information found.")
}

//https://www.alibabacloud.com/help/doc-detail/25380.htm
func (vmHandler *AlibabaVMHandler) ConvertVMStatusString(vmStatus string) (irs.VMStatus, error) {
	var resultStatus string
	cblogger.Infof("vmStatus : [%s]", vmStatus)

	if strings.EqualFold(vmStatus, "pending") {
		//resultStatus = "Creating"	// VM 생성 시점의 Pending은 CB에서는 조회가 안되기 때문에 일단 처리하지 않음.
		resultStatus = "Resuming" // Resume 요청을 받아서 재기동되는 단계에도 Pending이 있기 때문에 Pending은 Resuming으로 맵핑함.
	} else if strings.EqualFold(vmStatus, "running") {
		resultStatus = "Running"
	} else if strings.EqualFold(vmStatus, "stopping") {
		resultStatus = "Suspending"
	} else if strings.EqualFold(vmStatus, "stopped") {
		resultStatus = "Suspended"
		//} else if strings.EqualFold(vmStatus, "pending") {
		//	resultStatus = "Resuming"
	} else if strings.EqualFold(vmStatus, "Rebooting") {
		resultStatus = "Rebooting"
	} else if strings.EqualFold(vmStatus, "shutting-down") {
		resultStatus = "Terminating"
	} else if strings.EqualFold(vmStatus, "Terminated") {
		resultStatus = "Terminated"
	} else {
		//resultStatus = "Failed"
		cblogger.Errorf("vmStatus [%s]와 일치하는 맵핑 정보를 찾지 못 함.", vmStatus)
		return irs.VMStatus("Failed"), errors.New(vmStatus + "와 일치하는 CB VM 상태정보를 찾을 수 없습니다.")
	}
	cblogger.Infof("VM 상태 치환 : [%s] ==> [%s]", vmStatus, resultStatus)
	return irs.VMStatus(resultStatus), nil
}

func (vmHandler *AlibabaVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	cblogger.Infof("Start")

	request := ecs.CreateDescribeInstanceStatusRequest()
	request.Scheme = "https"

	response, err := vmHandler.Client.DescribeInstanceStatus(request)
	if err != nil {
		cblogger.Error(err.Error())
		return nil, err
	}

	cblogger.Info("Success", response)
	if response.TotalCount < 1 {
		return nil, nil
	}

	var vmInfoList []*irs.VMStatusInfo
	for _, vm := range response.InstanceStatuses.InstanceStatus {
		cblogger.Infof("Cur VM:[%s] / ECS Status : [%s]", vm.InstanceId, vm.Status)
		vmStatus, errStatus := vmHandler.ConvertVMStatusString(vm.Status)
		if errStatus != nil {
			cblogger.Error(errStatus.Error())
			return nil, errStatus
		}
		curVmStatusInfo := irs.VMStatusInfo{IId: irs.IID{SystemId: vm.InstanceId}, VmStatus: vmStatus}
		vmInfoList = append(vmInfoList, &curVmStatusInfo)
	}

	return vmInfoList, nil
}
