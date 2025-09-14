// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by zephy@mz.co.kr, 2019.09.

package resources

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	cdcom "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"
	/*
		"github.com/davecgh/go-spew/spew"
	*/)

type AlibabaVMHandler struct {
	Region    idrv.RegionInfo
	Client    *ecs.Client
	VpcClient *vpc.Client
}

// 주어진 이미지 id에 대한 이미지 사이즈 조회
// -1 : 정보 조회 실패
// deprecated
func (vmHandler *AlibabaVMHandler) GetImageSize(ImageSystemId string) (int64, error) {
	cblogger.Debugf("ImageID : [%s]", ImageSystemId)

	imageRequest := ecs.CreateDescribeImagesRequest()
	imageRequest.Scheme = "https"

	imageRequest.ImageId = ImageSystemId
	imageRequest.ShowExpired = requests.NewBoolean(true) //default는 false, false일 때는 최신 이미지 정보만 조회됨, true일 때는 오래된 이미지도 조회

	response, err := vmHandler.Client.DescribeImages(imageRequest)
	if err != nil {
		cblogger.Error(err)
		return -1, err
	}

	if len(response.Images.Image) > 0 {
		cblogger.Info(response.Images.Image[0].Size)
		imageSize := int64(response.Images.Image[0].Size)
		return imageSize, nil

	} else {
		cblogger.Error("The requested Image information [" + ImageSystemId + "] could not be found.")
		return -1, errors.New("The requested Image information [" + ImageSystemId + "] could not be found.")
	}
}

// 참고 : VM 생성 시 인증 방식은 KeyPair 또는 ID&PWD 방식이 가능하지만 계정은 모두 root  - 비번 조회 기능은 없음
//
//	비밀번호는 8-30자로서 대문자, 소문자, 숫자 및/또는 특수 문자가 포함되어야 합니다.
//
// @TODO : root 계정의 비번만 설정 가능한 데 다른 계정이 요청되었을 경우 예외 처리할 것인지.. 아니면 비번을 설정할 것인지 확인 필요.
// @TODO : PublicIp 요금제 방식과 대역폭 설정 방법 논의 필요
func (vmHandler *AlibabaVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	cblogger.Debug(vmReqInfo)
	zoneId := vmHandler.Region.Zone

	vpcHandler := AlibabaVPCHandler{
		Region: vmHandler.Region,
		Client: vmHandler.VpcClient,
	}

	subnetInfo, err := GetSubnet(vpcHandler.Client, vmReqInfo.SubnetIID.SystemId, zoneId)
	if err != nil {
		return irs.VMInfo{}, errors.New("there is no available subnet")
	}
	cblogger.Info("subnetInfo response : ", subnetInfo)
	if subnetInfo.Zone != "" {
		zoneId = subnetInfo.Zone
	}
	cblogger.Debugf("Zone : %s", zoneId)
	if zoneId == "" {
		cblogger.Error("Connection information does not contain Zone information.")
		return irs.VMInfo{}, errors.New("Connection Connection information does not contain Zone information.")
	}
	//cblogger.Debug(vmReqInfo)

	/* 2021-10-26 이슈 #480에 의해 제거
	// 2021-04-28 cbuser 추가에 따른 Local KeyPair만 VM 생성 가능하도록 강제
	//=============================
	// KeyPair의 PublicKey 정보 처리
	//=============================
	cblogger.Infof("[%s] KeyPair 조회 시작", vmReqInfo.KeyPair
	keypairHandler := AlibabaKeyPairHandler{
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

	//=============================
	// UserData생성 처리
	//=============================
	/*
		package_update: true
		packages:
		 - sudo
		users:
		  - default
		  - name: cb-user
			groups: sudo
			shell: /bin/bash
			sudo: ['ALL=(ALL) NOPASSWD:ALL']
			ssh-authorized-keys:
			  - ssh-rsa AAAAB3NzaC1y
	*/
	/*
		//sudo 패키지 설치
		//userData := "#cloud-config\npackage_update: true\npackages:\n  - sudo\nusers:\n  - default\n  - name: " + CBDefaultVmUserName + "\n    groups: sudo\n    shell: /bin/bash\n    sudo: ['ALL=(ALL) NOPASSWD:ALL']\n    ssh-authorized-keys:\n      - "
		//sudo 그룹 사용
		//userData := "#cloud-config\nusers:\n  - default\n  - name: " + CBDefaultVmUserName + "\n    groups: sudo\n    shell: /bin/bash\n    sudo: ['ALL=(ALL) NOPASSWD:ALL']\n    ssh-authorized-keys:\n      - "
		//그룹 제거
		userData := "#cloud-config\nusers:\n  - default\n  - name: " + CBDefaultVmUserName + "\n    shell: /bin/bash\n    sudo: ['ALL=(ALL) NOPASSWD:ALL']\n    ssh-authorized-keys:\n      - "
		userData = userData + keyPairInfo.PublicKey
		userDataBase64 := base64.StdEncoding.EncodeToString([]byte(userData))
		cblogger.Infof("===== userData ===")
		cblogger.Debug(userDataBase64)
	*/

	vmImage, err := DescribeImageByImageId(vmHandler.Client, vmHandler.Region, vmReqInfo.ImageIID, false)
	if err != nil {
		cblogger.Error(err)
		errMsg := "We cannot retrieve information for the requested image." + err.Error()
		return irs.VMInfo{}, errors.New(errMsg)
	}

	isWindows := false
	osType := GetOsType(vmImage) //"OSType": "windows"
	if osType == "windows" {
		isWindows = true

		err := cdcom.ValidateWindowsPassword(vmReqInfo.VMUserPasswd)
		if err != nil {
			return irs.VMInfo{}, err
		}
	}

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

	//=============================
	// 보안그룹 처리 - SystemId 기반
	//=============================
	cblogger.Debug("To process based on SystemId, the security group array based on IID is retrieved and converted into a security group array based on SystemId.")
	var newSecurityGroupIds []string
	//var firstSecurityGroupId string

	for _, sgId := range vmReqInfo.SecurityGroupIIDs {
		cblogger.Debugf("Security group transformation: [%s]", sgId)
		newSecurityGroupIds = append(newSecurityGroupIds, sgId.SystemId)
		//firstSecurityGroupId = sgId.SystemId
		//break
	}

	cblogger.Debug("Security group transformation complete.")
	cblogger.Debug(newSecurityGroupIds)

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

	// request.ZoneId = vmHandler.Region.Zone // Disk의 경우 zone dependency가 있어 Zone 명시해야 함.(disk가 없으면 무시해도 됨.)
	request.ZoneId = zoneId

	// windows 일 떄는 password 만 set, keypairName은 비움.
	// 다른 os일 때 password는 cb-user의 password 로 사용
	if isWindows {
		request.Password = vmReqInfo.VMUserPasswd
	} else {
		request.KeyPairName = vmReqInfo.KeyPairIID.SystemId

		// cb user 추가
		request.Password = vmReqInfo.VMUserPasswd //값에는 8-30자가 포함되고 대문자, 소문자, 숫자 및/또는 특수 문자가 포함되어야 합니다.
		request.UserData = userDataBase64         // cbuser 추가
	}

	request.VSwitchId = vmReqInfo.SubnetIID.SystemId

	//==============
	//PublicIp 설정
	//==============
	//Public Ip를 생성하기 위해서는 과금형태와 대역폭(1 Mbit/s이상)을 지정해야 함.
	//PayByTraffic(기본값) : 트래픽 기준 결제(GB 단위) - 트래픽 기준 결제(GB 단위)를 사용하면 대역폭 사용료가 시간별로 청구
	//PayByBandwidth : 대역폭 사용료는 구독 기반이고 ECS 인스턴스 사용료에 포함 됨.
	request.InternetChargeType = "PayByBandwidth"           //Public Ip요금 방식을 1시간 단위(PayByBandwidth) 요금으로 설정 / PayByTraffic(기본값) : 1GB단위 시간당 트래픽 요금 청구
	request.InternetMaxBandwidthOut = requests.Integer("5") // 0보다 크면 Public IP가 할당 됨 - 최대 아웃 바운드 공용 대역폭 단위 : Mbit / s 유효한 값 : 0 ~ 100

	/// 0717 ///

	if vmReqInfo.TagList != nil && len(vmReqInfo.TagList) > 0 {
		vmTags := []ecs.RunInstancesTag{}
		for _, vmTag := range vmReqInfo.TagList {
			tag0 := ecs.RunInstancesTag{
				Key:   vmTag.Key,
				Value: vmTag.Value,
			}
			vmTags = append(vmTags, tag0)

		}
		request.Tag = &vmTags
	}

	/// 0717 ///
	//=============================
	// instance 사용 가능 검사
	//=============================

	// availableResourceResp, err := DescribeAvailableResource(vmHandler.Client, vmHandler.Region.Region, vmHandler.Region.Zone, "instance", "InstanceType", vmReqInfo.VMSpecName)
	availableResourceResp, err := DescribeAvailableResource(vmHandler.Client, vmHandler.Region.Region, zoneId, "instance", "InstanceType", vmReqInfo.VMSpecName)
	if err != nil {
		cblogger.Error(err)
	}
	if len(availableResourceResp.AvailableZone) == 0 {
		return irs.VMInfo{}, errors.New("No AvailableInstanceType in the request region")
	}

	//=============================
	// Root Disk Type 설정
	//=============================

	// 인스턴스 타입 별로 가능한 목록 불러오기
	// availableSystemDisksResp, err := DescribeAvailableSystemDisksByInstanceType(vmHandler.Client, vmHandler.Region.Region, vmHandler.Region.Zone, "PostPaid", "SystemDisk", vmReqInfo.VMSpecName)
	availableSystemDisksResp, err := DescribeAvailableSystemDisksByInstanceType(vmHandler.Client, vmHandler.Region.Region, zoneId, "PostPaid", "SystemDisk", vmReqInfo.VMSpecName)
	if err != nil {
		cblogger.Error(err)
	}
	if len(availableSystemDisksResp.AvailableZone) == 0 {
		return irs.VMInfo{}, errors.New("No AvailableSystemDisk for that instance type in the request region")
	}

	var supportedDiskTypes []string

	for _, zone := range availableSystemDisksResp.AvailableZone {
		for _, resource := range zone.AvailableResources.AvailableResource {
			for _, supportedResource := range resource.SupportedResources.SupportedResource {
				supportedDiskTypes = append(supportedDiskTypes, supportedResource.Value)
			}
		}
	}

	if vmReqInfo.RootDiskType == "" || strings.EqualFold(vmReqInfo.RootDiskType, "default") {
		// get Alibaba's Meta Info
		cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo("ALIBABA")
		if err != nil {
			cblogger.Error(err)
		}

		found := false
		useDiskType := ""

		// 가져온 목록과 Meta목록의 순서에 맞는 DiskType 찾기
		for _, metaDisk := range cloudOSMetaInfo.RootDiskType {
			if found {
				break
			}
			for _, availableDiskType := range supportedDiskTypes {
				if metaDisk == availableDiskType {
					found = true
					useDiskType = metaDisk
					break
				}
			}
		}

		if found {
			request.SystemDiskCategory = useDiskType
		} else {
			// TODO : 만약, alibaba에서 availableDisk의 우선순위가 있으면 적용한다. 없어서 0번째로 set
			request.SystemDiskCategory = supportedDiskTypes[0]
		}
	} else {
		// default가 아닐 때
		// vmReqInfo.RootDiskType와 비교
		// 들어온 값이 가능 목록에 있는지 체크
		// 있으면 set
		// 없으면 에러 리턴
		// InstanceType 별로 가능한 DiskType목록에 있는지 확인
		// 있으면 Set, 없으면 err return

		found := false
		useDiskType := ""

		for _, availableDiskType := range supportedDiskTypes {
			if vmReqInfo.RootDiskType == availableDiskType {
				found = true
				useDiskType = vmReqInfo.RootDiskType
				break
			}
		}

		if found == false {
			return irs.VMInfo{}, errors.New("The disktype you entered is not available for this instancetype.")
		}

		request.SystemDiskCategory = useDiskType
	}
	//=============================
	// Root Disk Size 변경
	//=============================
	if vmReqInfo.RootDiskSize == "" || strings.EqualFold(vmReqInfo.RootDiskSize, "default") {
		//디스크 정보가 없으면 건드리지 않음.
	} else {

		rootDiskSize, err := strconv.ParseInt(vmReqInfo.RootDiskSize, 10, 64)
		if err != nil {
			cblogger.Error(err)
			return irs.VMInfo{}, err
		}

		// cloudos_meta 에 DiskType, min, max 값 정의
		cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo("ALIBABA")

		arrDiskSizeOfType := cloudOSMetaInfo.RootDiskSize

		cblogger.Info("arrDiskSizeOfType: ", arrDiskSizeOfType)

		diskSizeValue := DiskSize{}
		// DiskType default 도 건드리지 않음
		if vmReqInfo.RootDiskType == "" || strings.EqualFold(vmReqInfo.RootDiskType, "default") {

			//diskSizeArr := strings.Split(arrDiskSizeOfType[0], "|")
			//diskSizeValue.diskType = diskSizeArr[0]
			//diskSizeValue.unit = diskSizeArr[3]
			//diskSizeValue.diskMinSize, err = strconv.ParseInt(diskSizeArr[1], 10, 64)
			//if err != nil {
			//	cblogger.Error(err)
			//	return irs.VMInfo{}, err
			//}
			//
			//diskSizeValue.diskMaxSize, err = strconv.ParseInt(diskSizeArr[2], 10, 64)
			//if err != nil {
			//	cblogger.Error(err)
			//	return irs.VMInfo{}, err
			//}
		} else {
			// diskType이 있으면 type에 맞는 min, max, default 값 사용
			isExists := false
			for idx, _ := range arrDiskSizeOfType {
				diskSizeArr := strings.Split(arrDiskSizeOfType[idx], "|")
				cblogger.Info("diskSizeArr: ", diskSizeArr)

				if strings.EqualFold(vmReqInfo.RootDiskType, diskSizeArr[0]) {
					diskSizeValue.diskType = diskSizeArr[0]
					diskSizeValue.unit = diskSizeArr[3]
					diskSizeValue.diskMinSize, err = strconv.ParseInt(diskSizeArr[1], 10, 64)
					if err != nil {
						cblogger.Error(err)
						return irs.VMInfo{}, err
					}

					diskSizeValue.diskMaxSize, err = strconv.ParseInt(diskSizeArr[2], 10, 64)
					if err != nil {
						cblogger.Error(err)
						return irs.VMInfo{}, err
					}
					isExists = true
				}
			}
			if !isExists {
				return irs.VMInfo{}, errors.New("Invalid Root Disk Type : " + vmReqInfo.RootDiskType)
			}

			if rootDiskSize < diskSizeValue.diskMinSize {
				cblogger.Error("Disk Size Error!!: ", rootDiskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
				//return irs.VMInfo{}, errors.New("Requested disk size cannot be smaller than the minimum disk size, invalid")
				return irs.VMInfo{}, errors.New("Root Disk Size must be at least the default size (" + strconv.FormatInt(diskSizeValue.diskMinSize, 10) + " GB).")
			}

			if rootDiskSize > diskSizeValue.diskMaxSize {
				cblogger.Error("Disk Size Error!!: ", rootDiskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
				//return irs.VMInfo{}, errors.New("Requested disk size cannot be larger than the maximum disk size, invalid")
				return irs.VMInfo{}, errors.New("Root Disk Size must be smaller than the maximum size (" + strconv.FormatInt(diskSizeValue.diskMaxSize, 10) + " GB).")
			}
		}

		//imageSize, err := vmHandler.GetImageSize(vmReqInfo.ImageIID.SystemId)
		imageSize := int64(vmImage.Size)
		if imageSize < 0 {
			return irs.VMInfo{}, errors.New("we cannot retrieve the basic size information for the requested image")
		} else {
			if rootDiskSize < imageSize {
				cblogger.Error("Disk Size Error!!: ", rootDiskSize)
				return irs.VMInfo{}, errors.New("Root Disk Size must be larger then the image size (" + strconv.FormatInt(imageSize, 10) + " GB).")
			}

		}

		request.SystemDiskSize = vmReqInfo.RootDiskSize

	}

	// Windows OS 처리
	//"Platform": "Windows Server 2012",
	//"OSName": "Windows Server  2012 R2 数据中心版 64位英文版",
	//"OSType": "windows",
	if isWindows {
		//The password must be 8 to 30 characters in length
		//and contain at least three of the following character types: uppercase letters, lowercase letters, digits, and special characters.
		//Special characters include: // ( ) ` ~ ! @ # $ % ^ & * - _ + = | { } [ ] : ; ' < > , . ? /

	}

	//=============================
	// VM생성 처리
	//=============================
	cblogger.Debug("Create EC2 Instance")
	cblogger.Debug(request)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   zoneId,
		ResourceType: call.VM,
		ResourceName: vmReqInfo.IId.NameId,
		CloudOSAPI:   "RunInstances()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	response, err := vmHandler.Client.RunInstances(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		cblogger.Error(err.Error())
		return irs.VMInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))
	//cblogger.Debug(response)

	if len(response.InstanceIdSets.InstanceIdSet) < 1 {
		return irs.VMInfo{}, errors.New("No errors have occurred, but no VMs have been created.")
	}

	//=========================================
	// VM 정보를 조회할 수 있을 때까지 대기
	//=========================================
	newVmIID := irs.IID{SystemId: response.InstanceIdSets.InstanceIdSet[0]}

	//VM 생성 요청 후에는 곧바로 VM 정보를 조회할 수 없기 때문에 VM 상태를 조회할 수 있는 NotExist 상태가 아닐 때까지만 대기 함.
	//2021-05-11 WaitForRun을 호출하지 않아도 GetVM() 호출 시 에러가 발생하지 않는 것은 확인했음. (우선은 정책이 최종 확정이 아니라서 WaitForRun을 사용하도록 원복함.)
	//curStatus, errStatus := vmHandler.WaitForExist(newVmIID) // 20210511 - NotExist 상태가 아닐 때 까지만 대기
	curStatus, errStatus := vmHandler.WaitForRun(newVmIID) // 20210511 아직 정책이 최종 확정이 아니라서 WaitForRun을 사용하도록 원복함
	if errStatus != nil {
		cblogger.Error(errStatus.Error())

		_, cleanupErr := vmHandler.TerminateVM(newVmIID)
		if cleanupErr != nil {
			combinedErr := fmt.Errorf("VM creation failed: %v, and cleanup also failed: %v. VM may remain in cloud",
				errStatus, cleanupErr)
			return irs.VMInfo{}, combinedErr
		}
		return irs.VMInfo{}, errStatus
	}

	cblogger.Debug("==> Current status [%s] of the created VM [%s]", newVmIID, curStatus)

	// dataDisk attach
	for _, dataDiskIID := range vmReqInfo.DataDiskIIDs {
		err = AttachDisk(vmHandler.Client, vmHandler.Region, newVmIID, dataDiskIID)
		if err != nil {
			return irs.VMInfo{}, errors.New("Instance created but attach disk failed " + err.Error())
		}
	}

	//vmInfo, errVmInfo := vmHandler.GetVM(irs.IID{SystemId: response.InstanceId})
	vmInfo, errVmInfo := vmHandler.GetVM(newVmIID)
	if errVmInfo != nil {
		cblogger.Error(errVmInfo.Error())
		return irs.VMInfo{}, errVmInfo
	}

	// VM을 삭제해도 DataDisk는 삭제되지 않도록 Attribute 설정
	diskRequest := ecs.CreateModifyDiskAttributeRequest()
	diskRequest.Scheme = "https"
	diskRequest.DeleteWithInstance = requests.NewBoolean(false)

	diskIds := []string{}

	for _, dataDiskId := range vmInfo.DataDiskIIDs {
		diskIds = append(diskIds, dataDiskId.SystemId)
	}

	diskRequest.DiskIds = &diskIds

	_, diskErr := vmHandler.Client.ModifyDiskAttribute(diskRequest)
	if err != nil {
		return irs.VMInfo{}, errors.New("Instance created but modifying disk attributes failed " + diskErr.Error())
	}

	vmInfo.IId.NameId = vmReqInfo.IId.NameId

	//VM 생성 시 요청한 계정 정보가 있을 경우 사용된 계정 정보를 함께 전달 함.
	if vmReqInfo.VMUserPasswd != "" {
		vmInfo.VMUserPasswd = vmReqInfo.VMUserPasswd
		vmInfo.VMUserId = "root"
	}
	return vmInfo, nil
}

// VM 상태가 정보를 조회할 수 있는 상태가 될때까지 기다림(최대 30초간 대기)
func (vmHandler *AlibabaVMHandler) WaitForExist(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Debug("======> After creating a VM, it's not possible to retrieve VM information immediately, so it waits until the status is not \"NotExist\" before proceeding.")

	waitStatus := "NotExist" //VM정보 조회가 안됨.
	//waitStatus := "Running"
	//waitStatus := "Creating" //너무 일찍 종료 시 리턴할 VM의 세부 항목의 정보 조회가 안됨.

	//===================================
	// Suspending 되도록 3초 정도 대기 함.
	//===================================
	curRetryCnt := 0
	maxRetryCnt := 30
	for {
		curStatus, errStatus := vmHandler.GetVMStatus(vmIID)
		if errStatus != nil {
			cblogger.Error(errStatus.Error())
		}

		cblogger.Info("===>VM Status : ", curStatus)

		if curStatus != irs.VMStatus(waitStatus) { //|| curStatus == irs.VMStatus("Running") {
			cblogger.Infof("===> Suspended waiting because the VM status [%s] is not [%s].", curStatus, waitStatus)
			break
		}

		//if curStatus != irs.VMStatus(waitStatus) {
		curRetryCnt++
		cblogger.Errorf("Waiting for 1 second and then querying because the VM status is [%s].", curStatus)
		time.Sleep(time.Second * 1)
		if curRetryCnt > maxRetryCnt {
			cblogger.Errorf("Forcing termination as the VM status remains [%s] even after a long wait (%d seconds).", maxRetryCnt, waitStatus)
			return irs.VMStatus("Failed"), errors.New("After waiting for a long time, the status of the created VM did not change to [" + waitStatus + "], so the process is being terminated.")
		}
		//} else {
		//break
		//}
	}

	return irs.VMStatus(waitStatus), nil
}

// VM 정보를 조회할 수 있을 때까지 최대 30초간 대기
func (vmHandler *AlibabaVMHandler) WaitForRun(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Debug("======> Waiting until information retrieval is not possible immediately after VM creation because it is not running yet.")

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
			cblogger.Infof("===> Stopping the wait because the VM status is [%s].", curStatus)
			break
		}

		//if curStatus != irs.VMStatus(waitStatus) {
		curRetryCnt++
		cblogger.Debugf("The VM status is not [%s], so waiting for 1 second before querying.", waitStatus)
		time.Sleep(time.Second * 1)
		if curRetryCnt > maxRetryCnt {
			cblogger.Errorf("Forcing termination as the VM status remains unchanged as [%s] even after waiting for a long time (%d seconds).", maxRetryCnt, waitStatus)
			return irs.VMStatus("Failed"), errors.New("After waiting for a long time, the status of the created VM did not change to [" + waitStatus + "], so the process is being terminated.")
		}
		//} else {
		//break
		//}
	}

	return irs.VMStatus(waitStatus), nil
}

func (vmHandler *AlibabaVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmID : [%s]", vmIID.SystemId)

	request := ecs.CreateStartInstanceRequest()
	request.Scheme = "https"
	request.InstanceId = vmIID.SystemId

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmIID.SystemId,
		CloudOSAPI:   "StartInstance()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()

	curStatus, errStatus := vmHandler.GetVMStatus(vmIID)
	if errStatus != nil {
		cblogger.Error(errStatus.Error())
	}

	if curStatus != "Suspended" {
		return irs.VMStatus("Failed"), errors.New(string("The VM status is not Suspended." + curStatus))
	}
	response, err := vmHandler.Client.StartInstance(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		cblogger.Error(err.Error())
		return irs.VMStatus("Failed"), err
	}
	callogger.Debug(call.String(callLogInfo))

	cblogger.Debug(response)
	return irs.VMStatus("Resuming"), nil

}

// @TODO - 이슈 : 인스턴스 일시정지 시에 과금 정책을 결정해야 함 - StopCharging / KeepCharging
func (vmHandler *AlibabaVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmID : [%s]", vmIID.SystemId)

	request := ecs.CreateStopInstanceRequest()
	request.Scheme = "https"
	request.InstanceId = vmIID.SystemId
	request.StoppedMode = "StopCharging"

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmIID.SystemId,
		CloudOSAPI:   "StopInstance()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	response, err := vmHandler.Client.StopInstance(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		cblogger.Error(err.Error())
		return irs.VMStatus("Failed"), err
	}
	callogger.Debug(call.String(callLogInfo))
	cblogger.Debug(response)
	return irs.VMStatus("Suspending"), nil
}

func (vmHandler *AlibabaVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmID : [%s]", vmIID.SystemId)

	request := ecs.CreateRebootInstanceRequest()
	request.Scheme = "https"
	request.InstanceId = vmIID.SystemId

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmIID.SystemId,
		CloudOSAPI:   "RebootInstance()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	response, err := vmHandler.Client.RebootInstance(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		cblogger.Error(err.Error())
		return irs.VMStatus("Failed"), err
	}
	callogger.Debug(call.String(callLogInfo))
	cblogger.Debug(response)
	return irs.VMStatus("Rebooting"), nil
}

func (vmHandler *AlibabaVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmID : [%s]", vmIID.SystemId)
	/*
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
		maxRetryCnt := 60
		for {
			curStatus, errStatus := vmHandler.GetVMStatus(vmIID)
			if errStatus != nil {
				cblogger.Error(errStatus.Error())
			}
			cblogger.Info("===>VM Status : ", curStatus)
			if curStatus != irs.VMStatus("Suspended") {
				curRetryCnt++
				cblogger.Error("VM 상태가 Suspended가 아니라서 1초간 대기후 조회합니다.")
				time.Sleep(time.Second * 1)
				if curRetryCnt > maxRetryCnt {
					cblogger.Error("장시간 대기해도 VM의 Status 값이 Suspended로 변경되지 않아서 강제로 중단합니다.")
				}
			} else {
				break
			}
		}
	*/
	request := ecs.CreateDeleteInstanceRequest()
	request.Scheme = "https"
	request.InstanceId = vmIID.SystemId
	request.Force = requests.Boolean("true")

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmIID.SystemId,
		CloudOSAPI:   "DeleteInstance()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	maxRetryCnt := 40 // retry until 120s
	for i := 0; i < maxRetryCnt; i++ {

		callLogStart := call.Start()
		response, err := vmHandler.Client.DeleteInstance(request)
		callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

		if err != nil {
			if strings.Contains(err.Error(), "IncorrectInstanceStatus") {
				// Loop: IncorrectInstanceStatus error
				callLogInfo.ErrorMSG = err.Error()
				callogger.Error(call.String(callLogInfo))
				cblogger.Error(err.Error())
				time.Sleep(time.Second * 3)
			} else { // general error
				callLogInfo.ErrorMSG = err.Error()
				callogger.Error(call.String(callLogInfo))
				cblogger.Error(err.Error())
				return irs.VMStatus("Failed"), err
			}
		} else {
			callogger.Debug(call.String(callLogInfo))
			cblogger.Debug(response)
			break
		}
	}
	return irs.VMStatus("Terminating"), nil
}

func (vmHandler *AlibabaVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
	cblogger.Infof("vmID : [%s]", vmIID.SystemId)

	//request := ecs.CreateDescribeInstancesRequest()
	//request.Scheme = "https"
	//request.InstanceIds = "[\"" + vmIID.SystemId + "\"]"
	//
	//// logger for HisCall
	//callogger := call.GetLogger("HISCALL")
	//callLogInfo := call.CLOUDLOGSCHEMA{
	//	CloudOS:      call.ALIBABA,
	//	RegionZone:   vmHandler.Region.Zone,
	//	ResourceType: call.VM,
	//	ResourceName: vmIID.SystemId,
	//	CloudOSAPI:   "DescribeInstances()",
	//	ElapsedTime:  "",
	//	ErrorMSG:     "",
	//}
	//
	//callLogStart := call.Start()
	//response, err := vmHandler.Client.DescribeInstances(request)
	//callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	//
	//if err != nil {
	//	callLogInfo.ErrorMSG = err.Error()
	//	callogger.Error(call.String(callLogInfo))
	//	cblogger.Error(err.Error())
	//	return irs.VMInfo{}, err
	//}
	//callogger.Info(call.String(callLogInfo))
	//cblogger.Info(response)

	//if response.TotalCount < 1 {
	//	return irs.VMInfo{}, errors.New("Notfound: '" + vmIID.SystemId + "' VM Not found")
	//}

	//	vmInfo := vmHandler.ExtractDescribeInstances(response.Instances.Instance[0])
	//vmInfo, err := vmHandler.ExtractDescribeInstances(&response.Instances.Instance[0])

	instanceInfo, err := DescribeInstanceById(vmHandler.Client, vmHandler.Region, vmIID)
	vmInfo, err := vmHandler.ExtractDescribeInstances(&instanceInfo)
	cblogger.Debug("vmInfo", vmInfo)
	return vmInfo, err
}

// @TODO : 2020-03-26 Ali클라우드 API 구조가 바뀐 것 같아서 임시로 변경해 놓음.
// func (vmHandler *AlibabaVMHandler) ExtractDescribeInstances() irs.VMInfo {
func (vmHandler *AlibabaVMHandler) ExtractDescribeInstances(instanceInfo *ecs.Instance) (irs.VMInfo, error) {
	cblogger.Debug(instanceInfo)
	//diskInfo := vmHandler.getDiskInfo(instanceInfo.InstanceId)
	diskInfoList, err := DescribeDisksByInstanceId(vmHandler.Client, vmHandler.Region, irs.IID{SystemId: instanceInfo.InstanceId})
	if err != nil {
		//return irs.VMInfo{}, err
	}

	//time.Parse(layout, str)
	vmInfo := irs.VMInfo{
		IId:        irs.IID{NameId: instanceInfo.InstanceName, SystemId: instanceInfo.InstanceId},
		ImageIId:   irs.IID{SystemId: instanceInfo.ImageId},
		VMSpecName: instanceInfo.InstanceType,
		KeyPairIId: irs.IID{SystemId: instanceInfo.KeyPairName},
		//StartTime:  instancInfo.StartTime,

		Region:    irs.RegionInfo{Region: instanceInfo.RegionId, Zone: instanceInfo.ZoneId}, //  ex) {us-east1, us-east1-c} or {ap-northeast-2}
		VpcIID:    irs.IID{SystemId: instanceInfo.VpcAttributes.VpcId},
		SubnetIID: irs.IID{SystemId: instanceInfo.VpcAttributes.VSwitchId},
		//SecurityGroupIIds []IID // AWS, ex) sg-0b7452563e1121bb6
		//NetworkInterface string // ex) eth0
		//PublicDNS
		//PrivateIP
		//PrivateIP: instancInfo.VpcAttributes.PrivateIpAddress.IpAddress[0],
		//PrivateDNS

		//VMBootDisk  string // ex) /dev/sda1
		//VMBlockDisk string // ex)

		KeyValueList: []irs.KeyValue{{Key: "", Value: ""}},
	}
	tagList := []irs.KeyValue{}

	for _, aliTag := range instanceInfo.Tags.Tag {
		sTag := irs.KeyValue{}
		sTag.Key = aliTag.TagKey
		sTag.Value = aliTag.TagValue

		tagList = append(tagList, sTag)
	}
	vmInfo.TagList = tagList

	// Platform 정보 추가
	if instanceInfo.OSType == "windows" {
		vmInfo.Platform = irs.WINDOWS
	} else {
		vmInfo.Platform = irs.LINUX_UNIX
	}

	var dataDiskList []irs.IID
	for _, diskInfo := range diskInfoList {
		if diskInfo.Type == "system" {
			vmInfo.RootDiskType = diskInfo.Category
			vmInfo.RootDiskSize = strconv.Itoa(diskInfo.Size)
			vmInfo.RootDeviceName = diskInfo.Device
		} else {
			dataDiskList = append(dataDiskList, irs.IID{NameId: diskInfo.DiskName, SystemId: diskInfo.DiskId})
		}
	}
	if len(dataDiskList) > 0 {
		vmInfo.DataDiskIIDs = dataDiskList
	}

	if len(instanceInfo.NetworkInterfaces.NetworkInterface) > 0 {
		vmInfo.NetworkInterface = instanceInfo.NetworkInterfaces.NetworkInterface[0].NetworkInterfaceId
	}

	//vmInfo.VMUserId = "root"
	vmInfo.VMUserId = CBDefaultVmUserName //2021-05-11 VMUserId 정보를 cb-user로 리턴.

	//2021-05-11 VM생성 후 WaitForRun()을 사용하지 않기 위해 추가
	//VM을 생성하자 마자 조회하면 PrivateIpAddress 정보가 없음.
	if len(instanceInfo.VpcAttributes.PrivateIpAddress.IpAddress) > 0 {
		vmInfo.PrivateIP = instanceInfo.VpcAttributes.PrivateIpAddress.IpAddress[0]
	}

	/*
		if !reflect.ValueOf(reservation.Instances[0].PublicDnsName).IsNil() {
			vmInfo.PublicDNS = *reservation.Instances[0].PublicDnsName
		}
	*/

	//VMUserId
	//VMUserPasswd
	//NetworkInterfaceId

	if len(instanceInfo.PublicIpAddress.IpAddress) > 0 {
		vmInfo.PublicIP = instanceInfo.PublicIpAddress.IpAddress[0]
	}

	for _, security := range instanceInfo.SecurityGroupIds.SecurityGroupId {
		//vmInfo.SecurityGroupIds = append(vmInfo.SecurityGroupIds, *security.GroupId)
		vmInfo.SecurityGroupIIds = append(vmInfo.SecurityGroupIIds, irs.IID{SystemId: security})
	}

	timeLen := len(instanceInfo.CreationTime)
	cblogger.Infof("Server startup time format conversion processing")
	cblogger.Infof("======> Length of creation time [%s]", timeLen)
	if timeLen > 7 {
		cblogger.Infof("======> Last character of creation time [%s]", instanceInfo.CreationTime[timeLen-1:])
		var NewStartTime string
		if instanceInfo.CreationTime[timeLen-1:] == "Z" && timeLen == 17 {
			//cblogger.Infof("======> 문자열 변환 : [%s]", StartTime[:timeLen-1])
			NewStartTime = instanceInfo.CreationTime[:timeLen-1] + ":00Z"
			cblogger.Infof("======> Final string conversion: [%s]", NewStartTime)
		} else {
			NewStartTime = instanceInfo.CreationTime
		}

		cblogger.Infof("Convert StartTime string [%s] to time.time", NewStartTime)

		//layout := "2020-05-07T01:36Z"
		t, err := time.Parse(time.RFC3339, NewStartTime)
		if err != nil {
			cblogger.Error(err)
			return irs.VMInfo{}, err
		} else {
			cblogger.Debug("======> [%v]", t)
			vmInfo.StartTime = t
		}
	}
	vmInfo.KeyValueList = irs.StructToKeyValueList(instanceInfo)
	return vmInfo, nil
}

func (vmHandler *AlibabaVMHandler) ListVM() ([]*irs.VMInfo, error) {
	cblogger.Infof("Start")

	//request := ecs.CreateDescribeInstancesRequest()
	//request.Scheme = "https"
	//
	//// logger for HisCall
	//callogger := call.GetLogger("HISCALL")
	//callLogInfo := call.CLOUDLOGSCHEMA{
	//	CloudOS:      call.ALIBABA,
	//	RegionZone:   vmHandler.Region.Zone,
	//	ResourceType: call.VM,
	//	ResourceName: "ListVM()",
	//	CloudOSAPI:   "DescribeInstances()",
	//	ElapsedTime:  "",
	//	ErrorMSG:     "",
	//}
	//
	//callLogStart := call.Start()
	//response, err := vmHandler.Client.DescribeInstances(request)
	//callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	//
	//if err != nil {
	//	callLogInfo.ErrorMSG = err.Error()
	//	callogger.Error(call.String(callLogInfo))
	//	cblogger.Error(err.Error())
	//	return nil, err
	//}
	//callogger.Info(call.String(callLogInfo))
	//cblogger.Info(response)

	resultInstanceList, err := DescribeInstances(vmHandler.Client, vmHandler.Region, nil)
	if err != nil {
		return nil, err
	}
	var vmInfoList []*irs.VMInfo
	for _, curInstance := range resultInstanceList {
		//for _, curInstance := range response.Instances.Instance {

		cblogger.Info("[%s] ECS information retrieval", curInstance.InstanceId)
		vmInfo, errVmInfo := vmHandler.GetVM(irs.IID{SystemId: curInstance.InstanceId})
		if errVmInfo != nil {
			cblogger.Error(errVmInfo.Error())
			return nil, errVmInfo
		}
		//cblogger.Info("=======>VM 조회 결과")
		cblogger.Debug(vmInfo)

		vmInfoList = append(vmInfoList, &vmInfo)
	}
	cblogger.Debug(vmInfoList)
	return vmInfoList, nil
}

// SHUTTING-DOWN / TERMINATED
func (vmHandler *AlibabaVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	vmID := vmIID.SystemId
	cblogger.Infof("vmID : [%s]", vmID)

	request := ecs.CreateDescribeInstanceStatusRequest()
	request.Scheme = "https"
	request.InstanceId = &[]string{vmIID.SystemId}
	cblogger.Infof("request : [%v]", request)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmIID.SystemId,
		CloudOSAPI:   "DescribeInstanceStatus()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	response, err := vmHandler.Client.DescribeInstanceStatus(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err.Error())
		return irs.VMStatus("Failed"), err
	}
	callogger.Debug(call.String(callLogInfo))

	cblogger.Debug("Success", response)
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

//알리 클라우드 라이프 사이클 : https://www.alibabacloud.com/help/doc-detail/25380.htm
/*
const (
        Creating VMStatus = “Creating" // from launch to running
        Running VMStatus = “Running"
        Suspending VMStatus = “Suspending" // from running to suspended
        Suspended  VMStatus = “Suspended"
        Resuming VMStatus = “Resuming" // from suspended to running
        Rebooting VMStatus = “Rebooting" // from running to running
        Terminating VMStatus = “Terminating" // from running, suspended to terminated
        Terminated  VMStatus = “Terminated“
        NotExist  VMStatus = “NotExist“  // VM does not exist
        Failed  VMStatus = “Failed“
)
<최종 상태>
Running(동작 상태): MCIS가 동작 상태
Suspended(중지 상태): MCIS가 중지된 상태
Failed(실패 상태): MCIS가 오류로 인해 중단된 상태
Terminated(종료 상태): MCIS가 종료된 상태
<전이 상태>
Creating(생성 진행 상태): MCIS가 생성되는 중간 상태
Suspending(중지 진행 상태): MCIS를 일시 중지하기 위한 중간 상태
Resuming(재개 진행 상태): MCIS를 다시 실행하기 위한 중간 상태
Rebooting(재시작 진행 상태): MCIS를 재부팅하는 상태
Terminating(종료 진행 상태): MCIS의 종료를 실행하고 있는 중간 상태
*/
func (vmHandler *AlibabaVMHandler) ConvertVMStatusString(vmStatus string) (irs.VMStatus, error) {
	var resultStatus string
	cblogger.Infof("vmStatus : [%s]", vmStatus)

	if strings.EqualFold(vmStatus, "Pending") {
		resultStatus = "Creating"
	} else if strings.EqualFold(vmStatus, "Starting") {
		resultStatus = "Resuming" // Resume 요청을 받아서 재기동되는 단계는 Resuming으로 맵핑함.
	} else if strings.EqualFold(vmStatus, "Running") {
		resultStatus = "Running"
	} else if strings.EqualFold(vmStatus, "Stopping") {
		resultStatus = "Suspending"
	} else if strings.EqualFold(vmStatus, "Stopped") {
		resultStatus = "Suspended"
	} else {
		//resultStatus = "Failed"
		cblogger.Errorf("Cannot find mapping information matching vmStatus [%s].", vmStatus)
		return irs.VMStatus("Failed"), errors.New("Cannot find CB VM status information matching " + vmStatus)
	}
	cblogger.Infof("Replace VMStatus : [%s] ==> [%s]", vmStatus, resultStatus)
	return irs.VMStatus(resultStatus), nil
}

func (vmHandler *AlibabaVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	cblogger.Infof("Start")

	request := ecs.CreateDescribeInstanceStatusRequest()
	request.Scheme = "https"

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: "ListVMStatus()",
		CloudOSAPI:   "DescribeInstanceStatus()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	response, err := vmHandler.Client.DescribeInstanceStatus(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err.Error())
		return nil, err
	}
	callogger.Info(call.String(callLogInfo))

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

// deprecated
func (vmHandler *AlibabaVMHandler) getDiskInfo(instanceId string) ecs.Disk {

	diskRequest := ecs.CreateDescribeDisksRequest()
	diskRequest.Scheme = "https"

	diskRequest.InstanceId = instanceId

	response, err := vmHandler.Client.DescribeDisks(diskRequest)
	if err != nil {
		cblogger.Error(err.Error())
	}
	cblogger.Info("response: ", response)

	return response.Disks.Disk[0]
}

func (vmHandler *AlibabaVMHandler) ListIID() ([]*irs.IID, error) {
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: "ListIID()",
		CloudOSAPI:   "DescribeInstances()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()

	iidList, err := DescribeInstancesIdOnly(vmHandler.Client, vmHandler.Region)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callogger.Error(call.String(callLogInfo))
		return iidList, err
	}

	return iidList, nil
}
