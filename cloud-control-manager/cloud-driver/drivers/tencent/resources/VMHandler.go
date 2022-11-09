// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//   - Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2019.03.
package resources

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	//"regexp"

	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	cdcom "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	tencentcbs "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cbs/v20170312"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	//lighthouse "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/lighthouse/v20200324"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER Tencent VMHandler")
}

type TencentVMHandler struct {
	Region     idrv.RegionInfo
	Client     *cvm.Client
	DiskClient *tencentcbs.Client
}

//type TencentCbsHandler struct {
//	Region idrv.RegionInfo
//	Client *cbs.Client
//}

// LightHouse 사용을 위한 handler : TODO : region의 disk정보를 가져올 수 있으므로 사용방안 찾기
//type TencentDiskHandler struct {
//	Region idrv.RegionInfo
//	Client *lighthouse.Client
//}

//type DiskInfo struct {
//	Zone           string
//	DiskType       string
//	DiskMinSize    int64
//	DiskMaxSize    int64
//	DiskStepSize   int64
//	DiskSalesState string
//}

// Cloud Block Storage(cbs) Disk Info
//type CbsDiskInfo struct {
//	DiskType        string
//	DiskState       string
//	InstanceIdList  []string
//	DiskName        string
//	DiskId          string
//	SnapshotSize    uint64
//	SnapshotAbility bool
//}

// VM생성 시 Zone이 필수라서 Credential의 Zone에만 생성함.
func (vmHandler *TencentVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	cblogger.Info(vmReqInfo)

	zoneId := vmHandler.Region.Zone
	cblogger.Debugf("Zone : %s", zoneId)
	if zoneId == "" {
		cblogger.Error("Connection 정보에 Zone 정보가 없습니다.")
		return irs.VMInfo{}, errors.New("Connection 정보에 Zone 정보가 없습니다")
	}

	//=================================================
	// 동일 이름 생성 방지 추가(cb-spider 요청 필수 기능)
	//=================================================
	//vmExist, errExist := vmHandler.vmExist(vmReqInfo.IId.NameId)
	//cblogger.Error("vmExist ::: ", vmExist)
	//cblogger.Error("errExist :::", errExist)
	//if errExist != nil {
	//	cblogger.Error(errExist)
	//	return irs.VMInfo{}, errExist
	//}
	//if vmExist {
	//	return irs.VMInfo{}, errors.New("A VM with the name " + vmReqInfo.IId.NameId + " already exists.")
	//}

	cblogger.Error("imageInfo begin")
	// Image의 크기 -> rootdisk size, datadisk attach 확인 및 설정
	imageTypes := []string{"PUBLIC_IMAGE", "SHARED_IMAGE", "PRIVATE_IMAGE"}
	imageInfo, err := DescribeImagesByID(vmHandler.Client, vmReqInfo.ImageIID, imageTypes)
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}

	isWindow := false
	cblogger.Info("OsName,", *imageInfo.OsName)     //"OsName": "Windows Server 2012 R2 DataCenter 64bitEN",
	cblogger.Info("Platform,", *imageInfo.Platform) //"Platform": "Windows", "Ubuntu",

	platform := GetOsType(imageInfo)
	if platform == "Windows" {
		err := cdcom.ValidateWindowsPassword(vmReqInfo.VMUserPasswd)
		if err != nil {
			return irs.VMInfo{}, err
		}

		isWindow = true
		vmReqInfo.KeyPairIID = irs.IID{}
		vmReqInfo.VMUserId = "administrator" // window은 administrator로 set
		cblogger.Error("Window 이므로 keyPair는 사용하지 않고 admin, pass만 사용", vmReqInfo.VMUserId, vmReqInfo.VMUserPasswd, vmReqInfo.KeyPairIID)
	}

	/* 2021-10-26 이슈 #480에 의해 제거
	// 2021-04-28 cbuser 추가에 따른 Local KeyPair만 VM 생성 가능하도록 강제
	//=============================
	// KeyPair의 PublicKey 정보 처리
	//=============================
	cblogger.Infof("[%s] KeyPair 조회 시작", vmReqInfo.KeyPairIID.SystemId)
	keypairHandler := TencentKeyPairHandler{
		//CredentialInfo:
		Region: vmHandler.Region,
		Client: vmHandler.Client,
	}
	cblogger.Info(keypairHandler)
	keyPairInfo, errKeyPair := keypairHandler.GetKey(vmReqInfo.KeyPairIID)
	if errKeyPair != nil {
		cblogger.Error(errKeyPair)
		return irs.VMInfo{}, errKeyPair
	}
	*/

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmReqInfo.IId.NameId,
		CloudOSAPI:   "RunInstances()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := cvm.NewRunInstancesRequest()
	request.InstanceType = common.StringPtr(vmReqInfo.VMSpecName)

	request.ImageId = common.StringPtr(vmReqInfo.ImageIID.SystemId)
	request.VirtualPrivateCloud = &cvm.VirtualPrivateCloud{
		VpcId:    common.StringPtr(vmReqInfo.VpcIID.SystemId),
		SubnetId: common.StringPtr(vmReqInfo.SubnetIID.SystemId),
	}

	request.InstanceChargeType = common.StringPtr("POSTPAID_BY_HOUR")

	request.InternetAccessible = &cvm.InternetAccessible{
		// 	InternetChargeType: common.StringPtr("TRAFFIC_POSTPAID_BY_HOUR"),
		PublicIpAssigned:        common.BoolPtr(true),
		InternetMaxBandwidthOut: common.Int64Ptr(1), //Public Ip를 할당하려면 The maximum outbound bandwidth of the public network가 1Mbps이상이어야 함.
	}

	request.InstanceName = common.StringPtr(vmReqInfo.IId.NameId)

	// windows의 경우 keyPair set 하면 오류. password setting 되어있는지 확인
	if isWindow {
		//user := vmReqInfo.VMUserId // administrator
		request.LoginSettings = &cvm.LoginSettings{
			Password: &vmReqInfo.VMUserPasswd,
		}
	} else {
		request.LoginSettings = &cvm.LoginSettings{
			KeyIds: common.StringPtrs([]string{vmReqInfo.KeyPairIID.SystemId}),
		}
	}

	//=============================
	// 보안그룹 처리 - SystemId 기반
	//=============================
	cblogger.Debug("SystemId 기반으로 처리하기 위해 IID 기반의 보안그룹 배열을 SystemId 기반 보안그룹 배열로 조회및 변환함.")
	var newSecurityGroupIds []string
	for _, curSecurityGroup := range vmReqInfo.SecurityGroupIIDs {
		cblogger.Debugf("보안그룹 변환 : [%s]", curSecurityGroup)
		newSecurityGroupIds = append(newSecurityGroupIds, curSecurityGroup.SystemId)
	}

	cblogger.Debug("보안그룹 변환 완료")
	cblogger.Debug(newSecurityGroupIds)
	request.SecurityGroupIds = common.StringPtrs(newSecurityGroupIds)

	//=============================
	// Placement 처리
	//=============================
	request.Placement = &cvm.Placement{
		Zone: common.StringPtr(vmHandler.Region.Zone),
	}

	/* 이슈 #348에 의해 RootDisk 기능 지원하면서 기존 로직 제거
	request.SystemDisk = &cvm.SystemDisk{
		DiskType: common.StringPtr("CLOUD_PREMIUM"), //일부 인스턴스의 경우 기본 스토리지가 없는 스펙이 있어서 강제로 CLOUD_PREMIUM 지정
	}
	*/
	request.SystemDisk = &cvm.SystemDisk{}

	//=============================
	// SystemDisk 처리 - 이슈 #348에 의해 RootDisk 기능 지원
	//=============================
	//이슈#660 반영
	if strings.EqualFold(vmReqInfo.RootDiskType, "default") {
		vmReqInfo.RootDiskType = ""
	}

	if vmReqInfo.RootDiskType != "" || vmReqInfo.RootDiskSize != "" {
		request.SystemDisk = &cvm.SystemDisk{}
	}
	//	request.SystemDisk.DiskType = common.StringPtr(vmReqInfo.RootDiskType)
	//=============================
	// Root Disk Type 변경
	//=============================

	//cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo("TENCENT") // cloudos_meta 에 DiskType, min, max 값 정의 되어있음.
	//arrDiskSizeOfType := cloudOSMetaInfo.RootDiskSize
	//
	//fmt.Println("arrDiskSizeOfType: ", arrDiskSizeOfType)
	//

	//if vmReqInfo.RootDiskType == "" || strings.EqualFold(vmReqInfo.RootDiskType, "default") {
	//	cblogger.Debug("RootDiskType is default ")
	//	diskSizeArr := strings.Split(arrDiskSizeOfType[0], "|")
	//	diskSizeValue.diskType = diskSizeArr[0]
	//	diskSizeValue.unit = diskSizeArr[3]
	//	diskSizeValue.diskMinSize, err = strconv.ParseInt(diskSizeArr[1], 10, 64)
	//	if err != nil {
	//		cblogger.Error(err)
	//		return irs.VMInfo{}, err
	//	}
	//
	//	diskSizeValue.diskMaxSize, err = strconv.ParseInt(diskSizeArr[2], 10, 64)
	//	if err != nil {
	//		cblogger.Error(err)
	//		return irs.VMInfo{}, err
	//	}
	//
	//	// default에도 DiskType 넣어야 하나?
	//	//request.SystemDisk.DiskType = common.StringPtr(vmReqInfo.RootDiskType)
	//
	//} else {
	if !strings.EqualFold(vmReqInfo.RootDiskType, "") {
		//	isExists := false
		//	for idx, _ := range arrDiskSizeOfType {
		//		diskSizeArr := strings.Split(arrDiskSizeOfType[idx], "|")
		//		fmt.Println("diskSizeArr: ", diskSizeArr)
		//
		//		if strings.EqualFold(vmReqInfo.RootDiskType, diskSizeArr[0]) {
		//			diskSizeValue.diskType = diskSizeArr[0]
		//			diskSizeValue.unit = diskSizeArr[3]
		//			diskSizeValue.diskMinSize, err = strconv.ParseInt(diskSizeArr[1], 10, 64)
		//			if err != nil {
		//				cblogger.Error(err)
		//				return irs.VMInfo{}, err
		//			}
		//
		//			diskSizeValue.diskMaxSize, err = strconv.ParseInt(diskSizeArr[2], 10, 64)
		//			if err != nil {
		//				cblogger.Error(err)
		//				return irs.VMInfo{}, err
		//			}
		//
		//			isExists = true
		//		}
		//		cblogger.Debug("RootDiskSize isExists "+strconv.FormatInt(int64(idx), 10), isExists)
		//	}
		//	cblogger.Debug("RootDiskSize isExists ", isExists)
		//	if !isExists {
		//		return irs.VMInfo{}, errors.New("Invalid Root Disk Type : " + vmReqInfo.RootDiskType)
		//	}
		//
		request.SystemDisk.DiskType = common.StringPtr(vmReqInfo.RootDiskType)
	}

	//}

	//=============================
	// Root Disk Size 변경
	//=============================
	if vmReqInfo.RootDiskSize != "" {
		if strings.EqualFold(vmReqInfo.RootDiskSize, "default") {
		} else {
			rootDiskSize, err := strconv.ParseInt(vmReqInfo.RootDiskSize, 10, 64)
			if err != nil {
				cblogger.Error(err)
				return irs.VMInfo{}, err
			}

			imageSize := *imageInfo.ImageSize
			fmt.Println("image : ", imageSize)

			if rootDiskSize < imageSize {
				fmt.Println("Disk Size Error!!: ", rootDiskSize, imageSize)
				return irs.VMInfo{}, errors.New("Root Disk Size must be larger then the image size (" + strconv.FormatInt(imageSize, 10) + " GB).")
			}

			fmt.Println("rootDiskSize : ", rootDiskSize)
			fmt.Println("rootDiskSize : ", common.Int64Ptr(rootDiskSize))

			request.SystemDisk.DiskSize = common.Int64Ptr(rootDiskSize)
			cblogger.Debug("request.SystemDisk.DiskSize ", request.SystemDisk.DiskSize)
		}

	}

	// image 정보에 포함된 data disk setting
	snapshotSet := imageInfo.SnapshotSet
	dataDiskList := []*cvm.DataDisk{}
	for _, snapshot := range snapshotSet {
		dataDisk := cvm.DataDisk{}
		if *snapshot.DiskUsage == "DATA_DISK" {
			dataDisk.SnapshotId = snapshot.SnapshotId
			dataDisk.DiskSize = snapshot.DiskSize
			dataDiskList = append(dataDiskList, &dataDisk)
			cblogger.Info("Image에 DataDisk 포함 되어 있음. ")
		}
	}
	request.DataDisks = dataDiskList

	//=============================
	// UserData생성 처리(File기반)
	//=============================
	// 향후 공통 파일이나 외부에서 수정 가능하도록 cloud-init 스크립트 파일로 설정
	rootPath := os.Getenv("CBSPIDER_ROOT")
	fileDataCloudInit, err := ioutil.ReadFile(rootPath + CBCloudInitFilePath)
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}
	userData := string(fileDataCloudInit)
	//userData = strings.ReplaceAll(userData, "{{username}}", CBDefaultVmUserName)
	//userData = strings.ReplaceAll(userData, "{{public_key}}", keyPairInfo.PublicKey)
	userDataBase64 := base64.StdEncoding.EncodeToString([]byte(userData))
	cblogger.Debugf("cloud-init data : [%s]", userDataBase64)
	request.UserData = common.StringPtr(userDataBase64)

	cblogger.Debug("===== 요청 객체====")
	spew.Config.Dump(request)
	callLogStart := call.Start()
	response, err := vmHandler.Client.RunInstances(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return irs.VMInfo{}, err
	}
	spew.Dump(response)
	callogger.Info(call.String(callLogInfo))
	cblogger.Debug(response.ToJsonString())

	//=========================================
	// VM 정보를 조회할 수 있을 때까지 대기
	//-----------------------------------------
	// WaitForRun을 호출하지 않아도 상관 없지만 Public Ip 등은 할당되지 않아서 조회되지 않으며
	// cb-tumblebug에서 일부 정보를 사용하기 때문에 Tencent도 Running 상태가 될때까지 대기 함.
	//=========================================
	newVmIID := irs.IID{SystemId: *response.Response.InstanceIdSet[0]}

	curStatus, errStatus := vmHandler.WaitForRun(newVmIID)
	if errStatus != nil {
		cblogger.Error(errStatus.Error())
		return irs.VMInfo{}, nil
	}
	cblogger.Info("==>생성된 VM[%s]의 현재 상태[%s]", newVmIID, curStatus)

	DiskHandler := TencentDiskHandler{
		Region: vmHandler.Region,
		Client: vmHandler.DiskClient,
	}

	for _, disk := range vmReqInfo.DataDiskIIDs {
		_, attachErr := AttachDisk(DiskHandler.Client, irs.IID{SystemId: disk.SystemId}, irs.IID{SystemId: newVmIID.SystemId})
		if attachErr != nil {
			return irs.VMInfo{}, attachErr
		}

		_, statusErr := WaitForDone(DiskHandler.Client, irs.IID{SystemId: disk.SystemId}, "ATTACHED")
		if statusErr != nil {
			return irs.VMInfo{}, statusErr
		}
	}

	vmInfo, errVmInfo := vmHandler.GetVM(newVmIID)
	vmInfo.IId.NameId = vmReqInfo.IId.NameId
	if isWindow == false && vmInfo.KeyPairIId.SystemId == "" {
		vmInfo.KeyPairIId.SystemId = vmReqInfo.KeyPairIID.SystemId // keypairIID가 없으면 채워 넣음, VM 생성 직후에는 안 들어올 수 있음
	}
	cblogger.Debug(vmInfo)
	return vmInfo, errVmInfo
}

func (vmHandler *TencentVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmNameId : [%s]", vmIID.SystemId)

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmIID.SystemId,
		CloudOSAPI:   "StopInstances()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := cvm.NewStopInstancesRequest()
	request.InstanceIds = common.StringPtrs([]string{vmIID.SystemId})
	/*
		Whether to force shut down an instance after a normal shutdown fails. Valid values:
		TRUE: force shut down an instance after a normal shutdown fails
		FALSE: do not force shut down an instance after a normal shutdown fails
		Default value: FALSE.
	*/
	// request.ForceStop = common.BoolPtr(true)

	/*
		Instance shutdown mode. Valid values:
		SOFT_FIRST: perform a soft shutdown first, and force shut down the instance if the soft shutdown fails
		HARD: force shut down the instance directly
		SOFT: soft shutdown only
		Default value: SOFT.
	*/
	// request.StopType = common.StringPtr("SOFT")

	/*
		Billing method of a pay-as-you-go instance after shutdown. Valid values:
		KEEP_CHARGING: billing continues after shutdown
		STOP_CHARGING: billing stops after shutdown
		Default value: KEEP_CHARGING. This parameter is only valid for some pay-as-you-go instances using cloud disks. For more information, see No charges when shut down for pay-as-you-go instances.
		https://intl.cloud.tencent.com/document/product/213/19918
	*/
	// request.StoppedMode = common.StringPtr("STOP_CHARGING")

	callLogStart := call.Start()
	response, err := vmHandler.Client.StopInstances(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return irs.VMStatus("Failed"), err
	}
	//spew.Dump(response)
	callogger.Info(call.String(callLogInfo))
	cblogger.Debug(response.ToJsonString())

	return irs.VMStatus("Suspending"), nil
}

func (vmHandler *TencentVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmNameId : [%s]", vmIID.SystemId)

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmIID.SystemId,
		CloudOSAPI:   "StartInstances()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	curStatus, errStatus := vmHandler.GetVMStatus(vmIID)
	if errStatus != nil {
		cblogger.Error(errStatus.Error())
	}
	cblogger.Debug(curStatus)
	if curStatus != "Suspended" {
		return irs.VMStatus("Failed"), errors.New(string("vm 상태가 Suspended 가 아닙니다." + curStatus))
	}

	request := cvm.NewStartInstancesRequest()
	request.InstanceIds = common.StringPtrs([]string{vmIID.SystemId})

	callLogStart := call.Start()
	response, err := vmHandler.Client.StartInstances(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return irs.VMStatus("Failed"), err
	}
	//spew.Dump(response)
	callogger.Info(call.String(callLogInfo))
	cblogger.Debug(response.ToJsonString())

	return irs.VMStatus("Resuming"), nil
}

func (vmHandler *TencentVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmNameId : [%s]", vmIID.NameId)

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmIID.SystemId,
		CloudOSAPI:   "RebootInstances()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	curStatus, errStatus := vmHandler.GetVMStatus(vmIID)
	if errStatus != nil {
		cblogger.Error(errStatus.Error())
	}
	cblogger.Debug(curStatus)
	if curStatus == "Running" {
		request := cvm.NewRebootInstancesRequest()
		request.InstanceIds = common.StringPtrs([]string{vmIID.SystemId})

		callLogStart := call.Start()
		response, err := vmHandler.Client.RebootInstances(request)
		callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

		if err != nil {
			callLogInfo.ErrorMSG = err.Error()
			callogger.Error(call.String(callLogInfo))

			cblogger.Error(err)
			return irs.VMStatus("Failed"), err
		}
		//spew.Dump(response)
		callogger.Info(call.String(callLogInfo))
		cblogger.Debug(response.ToJsonString())
	} else if curStatus == "Suspended" {
		_, err := vmHandler.ResumeVM(vmIID)
		if err != nil {
			callLogInfo.ErrorMSG = err.Error()
			callogger.Error(call.String(callLogInfo))

			cblogger.Error(err)
			return irs.VMStatus("Failed"), err
		}
	} else {
		return irs.VMStatus("Failed"), errors.New(string(curStatus + "상태인 경우에는 Reboot할 수 없습니다."))
	}

	return irs.VMStatus("Rebooting"), nil
}

func (vmHandler *TencentVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmNameId : [%s]", vmIID.NameId)

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmIID.SystemId,
		CloudOSAPI:   "TerminateInstances()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := cvm.NewTerminateInstancesRequest()
	request.InstanceIds = common.StringPtrs([]string{vmIID.SystemId})

	callLogStart := call.Start()
	response, err := vmHandler.Client.TerminateInstances(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return irs.VMStatus("Failed"), err
	}
	//spew.Dump(response)
	callogger.Info(call.String(callLogInfo))
	cblogger.Debug(response.ToJsonString())

	return irs.VMStatus("Terminating"), nil
}

func (vmHandler *TencentVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
	cblogger.Infof("vmNameId : [%s]", vmIID.SystemId)

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmIID.SystemId,
		CloudOSAPI:   "DescribeInstances()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := cvm.NewDescribeInstancesRequest()
	request.InstanceIds = common.StringPtrs([]string{vmIID.SystemId})

	callLogStart := call.Start()
	response, err := vmHandler.Client.DescribeInstances(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return irs.VMInfo{}, err
	}

	//spew.Dump(response)
	callogger.Info(call.String(callLogInfo))
	cblogger.Debug(response.ToJsonString())

	if *response.Response.TotalCount < 1 {
		return irs.VMInfo{}, errors.New("VM 정보를 찾을 수 없습니다")
	}

	vmInfo, errVmInfo := vmHandler.ExtractDescribeInstances(response.Response.InstanceSet[0])
	cblogger.Info("vmInfo", vmInfo)
	return vmInfo, errVmInfo
}

func (vmHandler *TencentVMHandler) ExtractDescribeInstances(curVm *cvm.Instance) (irs.VMInfo, error) {
	//cblogger.Info("ExtractDescribeInstances", curVm)
	//spew.Dump(curVm)

	//VM상태와 무관하게 항상 값이 존재하는 항목들만 초기화
	vmInfo := irs.VMInfo{
		IId:        irs.IID{SystemId: *curVm.InstanceId},
		VMSpecName: *curVm.InstanceType,
		VMUserId:   "cb-user",
		//KeyPairIId: irs.IID{SystemId: *curVm.},
	}

	if !reflect.ValueOf(curVm.ImageId).IsNil() {
		imageIID := irs.IID{SystemId: *curVm.ImageId}
		vmInfo.ImageIId = imageIID
		imageInfo, err := DescribeImagesByID(vmHandler.Client, imageIID, nil) // imageTypes := []string{"PUBLIC_IMAGE", "SHARED_IMAGE","PRIVATE_IMAGE"}
		if err != nil {
			cblogger.Error(err)
		}

		if *imageInfo.ImageType == "PRIVATE_IMAGE" {
			vmInfo.ImageType = irs.MyImage
		} else {
			vmInfo.ImageType = irs.PublicImage // "PUBLIC_IMAGE", "SHARED_IMAGE"
		}
	}

	// vmInfo.StartTime = *curVm.CreatedTime
	vmStartTime := *curVm.CreatedTime
	timeLen := len(vmStartTime)
	cblogger.Debug("서버 구동 시간 포멧 변환 처리")
	cblogger.Debugf("======> 생성시간 길이 [%s]", timeLen)
	if timeLen > 7 {
		cblogger.Debugf("======> 생성시간 마지막 문자열 [%s]", vmStartTime[timeLen-1:])
		var NewStartTime string
		if vmStartTime[timeLen-1:] == "Z" && timeLen == 17 {
			//cblogger.Infof("======> 문자열 변환 : [%s]", StartTime[:timeLen-1])
			NewStartTime = vmStartTime[:timeLen-1] + ":00Z"
			cblogger.Debugf("======> 최종 문자열 변환 : [%s]", NewStartTime)
		} else {
			NewStartTime = vmStartTime
		}

		cblogger.Debugf("Convert StartTime string [%s] to time.time", NewStartTime)

		//layout := "2020-05-07T01:36Z"
		t, err := time.Parse(time.RFC3339, NewStartTime)
		if err != nil {
			cblogger.Error(err)
		} else {
			cblogger.Debugf("======> [%v]", t)
			vmInfo.StartTime = t
		}
	}

	cblogger.Debug(curVm.LoginSettings)

	if !reflect.ValueOf(curVm.LoginSettings).IsNil() {
		if !reflect.ValueOf(curVm.LoginSettings.KeyIds).IsNil() {
			if len(curVm.LoginSettings.KeyIds) > 0 {
				vmInfo.KeyPairIId = irs.IID{SystemId: *curVm.LoginSettings.KeyIds[0]}
			}
		}
	}

	if !reflect.ValueOf(curVm.PublicIpAddresses).IsNil() {
		vmInfo.PublicIP = *curVm.PublicIpAddresses[0]
	}

	if !reflect.ValueOf(curVm.Placement.Zone).IsNil() {
		vmInfo.Region = irs.RegionInfo{
			Region: vmHandler.Region.Region, //리전 정보 추가
			Zone:   *curVm.Placement.Zone,
		}
	}

	if !reflect.ValueOf(curVm.VirtualPrivateCloud.VpcId).IsNil() {
		vmInfo.VpcIID = irs.IID{SystemId: *curVm.VirtualPrivateCloud.VpcId}
	}

	if !reflect.ValueOf(curVm.VirtualPrivateCloud.SubnetId).IsNil() {
		vmInfo.SubnetIID = irs.IID{SystemId: *curVm.VirtualPrivateCloud.SubnetId}
	}

	if !reflect.ValueOf(curVm.SecurityGroupIds).IsNil() {
		for _, curSecurityGroupId := range curVm.SecurityGroupIds {
			vmInfo.SecurityGroupIIds = append(vmInfo.SecurityGroupIIds, irs.IID{SystemId: *curSecurityGroupId})
		}
	}

	if !reflect.ValueOf(curVm.PrivateIpAddresses).IsNil() {
		vmInfo.PrivateIP = *curVm.PrivateIpAddresses[0]
	}

	keyValueList := []irs.KeyValue{
		{Key: "InstanceState", Value: *curVm.InstanceState},
		{Key: "OsName", Value: *curVm.OsName},
	}

	//요금타입
	if !reflect.ValueOf(curVm.InstanceChargeType).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "InstanceChargeType", Value: *curVm.InstanceChargeType})
	}

	//데이터 디스크 정보
	if !reflect.ValueOf(curVm.DataDisks).IsNil() {
		if len(curVm.DataDisks) > 0 {
			// if !reflect.ValueOf(curVm.DataDisks[0].DiskId).IsNil() {
			// 	vmInfo.VMBlockDisk = *curVm.DataDisks[0].DiskId
			// }
			for _, dataDisk := range curVm.DataDisks {
				dataDiskIID := irs.IID{SystemId: *dataDisk.DiskId}
				vmInfo.DataDiskIIDs = append(vmInfo.DataDiskIIDs, dataDiskIID)
			}
		}
	}

	//시스템 디스크 정보
	if !reflect.ValueOf(curVm.SystemDisk).IsNil() {
		if !reflect.ValueOf(curVm.SystemDisk.DiskType).IsNil() {
			keyValueList = append(keyValueList, irs.KeyValue{Key: "SystemDiskType", Value: *curVm.SystemDisk.DiskType})
			vmInfo.RootDiskType = *curVm.SystemDisk.DiskType
		}
		if !reflect.ValueOf(curVm.SystemDisk.DiskId).IsNil() {
			keyValueList = append(keyValueList, irs.KeyValue{Key: "SystemDiskId", Value: *curVm.SystemDisk.DiskId})
			vmInfo.VMBootDisk = *curVm.SystemDisk.DiskId
			vmInfo.RootDeviceName = *curVm.SystemDisk.DiskId
		}
		if !reflect.ValueOf(curVm.SystemDisk.DiskSize).IsNil() {
			keyValueList = append(keyValueList, irs.KeyValue{Key: "SystemDiskSize", Value: strconv.FormatInt(*curVm.SystemDisk.DiskSize, 10)})
			vmInfo.RootDiskSize = strconv.FormatInt(*curVm.SystemDisk.DiskSize, 10)
		}
	}

	if !reflect.ValueOf(curVm.InternetAccessible).IsNil() {
		if !reflect.ValueOf(curVm.InternetAccessible.InternetChargeType).IsNil() {
			keyValueList = append(keyValueList, irs.KeyValue{Key: "InternetChargeType", Value: *curVm.InternetAccessible.InternetChargeType})
		}
		if !reflect.ValueOf(curVm.InternetAccessible.InternetMaxBandwidthOut).IsNil() {
			keyValueList = append(keyValueList, irs.KeyValue{Key: "InternetMaxBandwidthOut", Value: strconv.FormatInt(*curVm.InternetAccessible.InternetMaxBandwidthOut, 10)})
		}
	}

	vmInfo.KeyValueList = keyValueList

	return vmInfo, nil
}

func (vmHandler *TencentVMHandler) ListVM() ([]*irs.VMInfo, error) {
	cblogger.Infof("Start")

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: "ListVM()",
		CloudOSAPI:   "DescribeInstances()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := cvm.NewDescribeInstancesRequest()
	request.Limit = common.Int64Ptr(100)

	callLogStart := call.Start()
	response, err := vmHandler.Client.DescribeInstances(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return nil, err
	}

	//spew.Dump(response)
	callogger.Info(call.String(callLogInfo))
	cblogger.Debug(response.ToJsonString())

	var vmInfoList []*irs.VMInfo
	for _, curVm := range response.Response.InstanceSet {
		vmInfo, _ := vmHandler.GetVM(irs.IID{SystemId: *curVm.InstanceId})
		vmInfoList = append(vmInfoList, &vmInfo)
	}

	return vmInfoList, nil
}

func (vmHandler *TencentVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmNameId : [%s]", vmIID.SystemId)

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmIID.SystemId,
		CloudOSAPI:   "DescribeInstancesStatus()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := cvm.NewDescribeInstancesStatusRequest()
	request.InstanceIds = common.StringPtrs([]string{vmIID.SystemId})

	callLogStart := call.Start()
	response, err := vmHandler.Client.DescribeInstancesStatus(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return irs.VMStatus("Failed"), err
	}
	//spew.Dump(response)
	callogger.Info(call.String(callLogInfo))
	cblogger.Debug(response.ToJsonString())

	if *response.Response.TotalCount < 1 {
		return irs.VMStatus("Failed"), errors.New("상태 정보를 찾을 수 없습니다")
	}

	vmStatus, errStatus := ConvertVMStatusString(*response.Response.InstanceStatusSet[0].InstanceState)
	cblogger.Infof("vmStatus : [%s]", vmStatus)
	return vmStatus, errStatus
}

func (vmHandler *TencentVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	cblogger.Debug("Start")

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: "ListVMStatus()",
		CloudOSAPI:   "DescribeInstancesStatus()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := cvm.NewDescribeInstancesStatusRequest()
	request.Limit = common.Int64Ptr(100)

	callLogStart := call.Start()
	response, err := vmHandler.Client.DescribeInstancesStatus(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return nil, err
	}
	//spew.Dump(response)
	callogger.Info(call.String(callLogInfo))
	cblogger.Debug(response.ToJsonString())

	var vmStatusList []*irs.VMStatusInfo
	for _, curVm := range response.Response.InstanceStatusSet {
		vmStatus, _ := ConvertVMStatusString(*curVm.InstanceState)

		vmStatusInfo := irs.VMStatusInfo{
			IId:      irs.IID{SystemId: *curVm.InstanceId},
			VmStatus: vmStatus,
		}
		cblogger.Info(vmStatusInfo.IId.SystemId, " Instance Status : ", vmStatusInfo.VmStatus)
		vmStatusList = append(vmStatusList, &vmStatusInfo)
	}

	return vmStatusList, nil
}

// tencent life cycle
// https://intl.cloud.tencent.com/document/product/213/4856?lang=en&pg=
func ConvertVMStatusString(vmStatus string) (irs.VMStatus, error) {
	var resultStatus string
	cblogger.Infof("vmStatus : [%s]", vmStatus)

	if strings.EqualFold(vmStatus, "pending") {
		//resultStatus = "Creating"	// VM 생성 시점의 Pending은 CB에서는 조회가 안되기 때문에 일단 처리하지 않음.
		resultStatus = "Resuming" // Resume 요청을 받아서 재기동되는 단계에도 Pending이 있기 때문에 Pending은 Resuming으로 맵핑함.
	} else if strings.EqualFold(vmStatus, "starting") {
		resultStatus = "Resuming"
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
	} else if strings.EqualFold(vmStatus, "terminating") {
		resultStatus = "Terminating"
	} else if strings.EqualFold(vmStatus, "shut-down") {
		resultStatus = "Terminated"
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

// VM 정보를 조회할 수 있을 때까지 최대 30초간 대기
func (vmHandler *TencentVMHandler) WaitForRun(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("======> VM 생성 직후에는 Public IP등 일부 정보 조회가 안되기 때문에 Running 될 때까지 대기함.")

	//waitStatus := "NotExist"	//VM정보 조회가 안됨.
	waitStatus := "Running"
	//waitStatus := "Creating" //너무 일찍 종료 시 리턴할 VM의 세부 항목의 정보 조회가 안됨.

	//===================================
	// Suspending 되도록 3초 정도 대기 함.
	//===================================
	curRetryCnt := 0
	maxRetryCnt := 120
	for {
		curStatus, errStatus := vmHandler.GetVMStatus(vmIID)
		if errStatus != nil {
			cblogger.Error(errStatus.Error())
		}

		cblogger.Info("===>VM Status : ", curStatus)

		if curStatus == irs.VMStatus(waitStatus) { //|| curStatus == irs.VMStatus("Running") {
			cblogger.Infof("===>VM 상태가 [%s]라서 대기를 중단합니다.", curStatus)
			break
		}

		//if curStatus != irs.VMStatus(waitStatus) {
		curRetryCnt++
		cblogger.Infof("VM 상태가 [%s]이 아니라서 1초 대기후 조회합니다.", waitStatus)
		time.Sleep(time.Second * 1)
		if curRetryCnt > maxRetryCnt {
			cblogger.Errorf("장시간(%d 초) 대기해도 VM의 Status 값이 [%s]으로 변경되지 않아서 강제로 중단합니다.", maxRetryCnt, waitStatus)
			return irs.VMStatus("Failed"), errors.New("장시간 기다렸으나 생성된 VM의 상태가 [" + waitStatus + "]으로 바뀌지 않아서 중단 합니다.")
		}
	}

	return irs.VMStatus(waitStatus), nil
}

// VM 이름으로 중복 생성을 막아야 해서 VM존재 여부를 체크함.
func (vmHandler *TencentVMHandler) vmExist(vmName string) (bool, error) {
	cblogger.Infof("VM조회(Name기반) : %s", vmName)
	request := cvm.NewDescribeInstancesRequest()
	request.Filters = []*cvm.Filter{
		&cvm.Filter{
			Name:   common.StringPtr("instance-name"),
			Values: common.StringPtrs([]string{vmName}),
		},
	}

	response, err := vmHandler.Client.DescribeInstances(request)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	if *response.Response.TotalCount < 1 {
		return false, nil
	}

	cblogger.Infof("VM 정보 찾음 - VmId:[%s] / VmName:[%s]", *response.Response.InstanceSet[0].InstanceId, *response.Response.InstanceSet[0].InstanceName)
	return true, nil
}

//DescribeDiskConfigs : Querying disk configuration
/*
해당 zone에 사용가능한 disk type 및 size 목록
Region 	Yes 	String 	Common parameter. For more information, please see the list of regions supported by the product.
Filters.N 	No 	Array of Filter 	Filter list.
zone
Filter by availability zone.
Type: String
Required: no

Lighthouse API가 있으나 ClientInterface 부분 처리 방법 필요
사용 방법 예시)
	var validDiskMin int64
	var validDiskMax int64
	diskHandler := TencentDiskHandler{Client: (*lighthouse.Client)(vmHandler.Client)}
	if vmReqInfo.RootDiskType != "" {
		request.SystemDisk.DiskType = common.StringPtr(vmReqInfo.RootDiskType)

		diskInfoList, err := diskHandler.DescribeDiskConfigs()
		if err != nil {
			cblogger.Debug("there is no available disk ")
		}
		for _, diskInfo := range diskInfoList {
			if strings.EqualFold(vmReqInfo.RootDiskType, diskInfo.DiskType) && strings.EqualFold(diskInfo.DiskSalesState, "AVAILABLE") {
				validDiskMin = diskInfo.DiskMinSize
				validDiskMax = diskInfo.DiskMaxSize
			} else {
				return irs.VMInfo{}, errors.New("Invalid Root Disk Type : " + vmReqInfo.RootDiskType + ", " + diskInfo.DiskSalesState)
			}
		}
	}
	cblogger.Debug("request.SystemDisk.DiskType ", request.SystemDisk.DiskType)

  다음 error로 주석 했음.  undefined: "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common".CredentialIface
*/
//func (diskHandler *TencentDiskHandler) DescribeDiskConfigs() ([]*DiskInfo, error) {
//	cblogger.Debug("Start")
//
//	callogger := call.GetLogger("HISCALL")
//	callLogInfo := call.CLOUDLOGSCHEMA{
//		CloudOS:      call.TENCENT,
//		RegionZone:   diskHandler.Region.Zone,
//		ResourceType: call.VM,
//		ResourceName: "DescribeDiskConfigs()",
//		CloudOSAPI:   "DescribeInstancesStatus()",
//		ElapsedTime:  "",
//		ErrorMSG:     "",
//	}
//
//	request := lighthouse.NewDescribeDiskConfigsRequest()
//
//	callLogStart := call.Start()
//	response, err := diskHandler.Client.DescribeDiskConfigs(request)
//	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
//
//	if err != nil {
//		callLogInfo.ErrorMSG = err.Error()
//		callogger.Error(call.String(callLogInfo))
//
//		cblogger.Error(err)
//		return nil, err
//	}
//	//spew.Dump(response)
//	callogger.Info(call.String(callLogInfo))
//	cblogger.Debug(response.ToJsonString())
//
//	//var vmStatusList []*irs.VMStatusInfo
//	var diskInfoList []*DiskInfo
//	for _, diskConfig := range response.Response.DiskConfigSet {
//
//		diskInfo := DiskInfo{
//			Zone:           *diskConfig.Zone,
//			DiskType:       *diskConfig.DiskType,
//			DiskMaxSize:    *diskConfig.MaxDiskSize,
//			DiskMinSize:    *diskConfig.MinDiskSize,
//			DiskStepSize:   *diskConfig.MinDiskSize,
//			DiskSalesState: *diskConfig.DiskSalesState,
//		}
//		diskInfoList = append(diskInfoList, &diskInfo)
//	}
//
//	return diskInfoList, nil
//}

// TODO : cloud storage block Interface에 추가되면 사용할 것
//func (cbsHandler *TencentCbsHandler) DescribeDisks() ([]*CbsDiskInfo, error) {
//	cblogger.Debug("Start")
//
//	callogger := call.GetLogger("HISCALL")
//	callLogInfo := call.CLOUDLOGSCHEMA{
//		CloudOS:      call.TENCENT,
//		RegionZone:   cbsHandler.Region.Zone,
//		ResourceType: call.VM,
//		ResourceName: "DescribeDisks()",
//		CloudOSAPI:   "DescribeDisks()",
//		ElapsedTime:  "",
//		ErrorMSG:     "",
//	}
//
//	request := cbs.NewDescribeDisksRequest()
//	request.Limit = common.Uint64Ptr(100)
//
//	callLogStart := call.Start()
//	response, err := cbsHandler.Client.DescribeDisks(request)
//	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
//
//	if err != nil {
//		callLogInfo.ErrorMSG = err.Error()
//		callogger.Error(call.String(callLogInfo))
//
//		cblogger.Error(err)
//		return nil, err
//	}
//	//spew.Dump(response)
//	callogger.Info(call.String(callLogInfo))
//	cblogger.Debug(response.ToJsonString())
//	//
//	//Response *struct {
//	//	TotalCount *uint64 `json:"TotalCount,omitempty" name:"TotalCount"`
//	//	DiskSet    []*Disk `json:"DiskSet,omitempty" name:"DiskSet"`
//	//	RequestId  *string `json:"RequestId,omitempty" name:"RequestId"`
//	//} `json:"Response"`
//	//var vmStatusList []*irs.VMStatusInfo
//
//	//type Disk struct {
//	//	DeleteWithInstance    *bool      `json:"DeleteWithInstance,omitempty" name:"DeleteWithInstance"`
//	//	RenewFlag             *string    `json:"RenewFlag,omitempty" name:"RenewFlag"`
//	//	DiskType              *string    `json:"DiskType,omitempty" name:"DiskType"`
//	//	DiskState             *string    `json:"DiskState,omitempty" name:"DiskState"`
//	//	SnapshotCount         *int64     `json:"SnapshotCount,omitempty" name:"SnapshotCount"`
//	//	AutoRenewFlagError    *bool      `json:"AutoRenewFlagError,omitempty" name:"AutoRenewFlagError"`
//	//	Rollbacking           *bool      `json:"Rollbacking,omitempty" name:"Rollbacking"`
//	//	InstanceIdList        []*string  `json:"InstanceIdList,omitempty" name:"InstanceIdList"`
//	//	Encrypt               *bool      `json:"Encrypt,omitempty" name:"Encrypt"`
//	//	DiskName              *string    `json:"DiskName,omitempty" name:"DiskName"`
//	//	BackupDisk            *bool      `json:"BackupDisk,omitempty" name:"BackupDisk"`
//	//	Tags                  []*Tag     `json:"Tags,omitempty" name:"Tags"`
//	//	InstanceId            *string    `json:"InstanceId,omitempty" name:"InstanceId"`
//	//	AttachMode            *string    `json:"AttachMode,omitempty" name:"AttachMode"`
//	//	AutoSnapshotPolicyIds []*string  `json:"AutoSnapshotPolicyIds,omitempty" name:"AutoSnapshotPolicyIds"`
//	//	ThroughputPerformance *uint64    `json:"ThroughputPerformance,omitempty" name:"ThroughputPerformance"`
//	//	Migrating             *bool      `json:"Migrating,omitempty" name:"Migrating"`
//	//	DiskId                *string    `json:"DiskId,omitempty" name:"DiskId"`
//	//	SnapshotSize          *uint64    `json:"SnapshotSize,omitempty" name:"SnapshotSize"`
//	//	Placement             *Placement `json:"Placement,omitempty" name:"Placement"`
//	//	IsReturnable          *bool      `json:"IsReturnable,omitempty" name:"IsReturnable"`
//	//	DeadlineTime          *string    `json:"DeadlineTime,omitempty" name:"DeadlineTime"`
//	//	Attached              *bool      `json:"Attached,omitempty" name:"Attached"`
//	//	DiskSize              *uint64    `json:"DiskSize,omitempty" name:"DiskSize"`
//	//	MigratePercent        *uint64    `json:"MigratePercent,omitempty" name:"MigratePercent"`
//	//	DiskUsage             *string    `json:"DiskUsage,omitempty" name:"DiskUsage"`
//	//	DiskChargeType        *string    `json:"DiskChargeType,omitempty" name:"DiskChargeType"`
//	//	Portable              *bool      `json:"Portable,omitempty" name:"Portable"`
//	//	SnapshotAbility       *bool      `json:"SnapshotAbility,omitempty" name:"SnapshotAbility"`
//	//	DeadlineError         *bool      `json:"DeadlineError,omitempty" name:"DeadlineError"`
//	//	RollbackPercent       *uint64    `json:"RollbackPercent,omitempty" name:"RollbackPercent"`
//	//	DifferDaysOfDeadline  *int64     `json:"DifferDaysOfDeadline,omitempty" name:"DifferDaysOfDeadline"`
//	//	ReturnFailCode        *int64     `json:"ReturnFailCode,omitempty" name:"ReturnFailCode"`
//	//	Shareable             *bool      `json:"Shareable,omitempty" name:"Shareable"`
//	//	CreateTime            *string    `json:"CreateTime,omitempty" name:"CreateTime"`
//	//}
//	var diskInfoList []*CbsDiskInfo
//	for _, disk := range response.Response.DiskSet {
//		//	DiskType              *string    `json:"DiskType,omitempty" name:"DiskType"`
//		//	DiskState             *string    `json:"DiskState,omitempty" name:"DiskState"`
//		//	InstanceIdList        []*string  `json:"InstanceIdList,omitempty" name:"InstanceIdList"`
//		//	DiskName              *string    `json:"DiskName,omitempty" name:"DiskName"`
//		//	DiskId                *string    `json:"DiskId,omitempty" name:"DiskId"`
//		//	SnapshotSize          *uint64    `json:"SnapshotSize,omitempty" name:"SnapshotSize"`
//		//	SnapshotAbility       *bool      `json:"SnapshotAbility,omitempty" name:"SnapshotAbility"`
//		diskInfo := CbsDiskInfo{
//			DiskType:  *disk.DiskType,
//			DiskState: *disk.DiskState,
//			//InstanceIdList:  *disk.InstanceIdList,
//			DiskName:        *disk.DiskName,
//			DiskId:          *disk.DiskId,
//			SnapshotSize:    *disk.SnapshotSize,
//			SnapshotAbility: *disk.SnapshotAbility,
//		}
//		diskInfoList = append(diskInfoList, &diskInfo)
//	}
//
//	return diskInfoList, nil
//}
