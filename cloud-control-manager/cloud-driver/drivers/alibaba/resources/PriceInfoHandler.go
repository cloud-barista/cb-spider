// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by zephy@mz.co.kr, 2019.09.

package resources

import (
	"log"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	/*
		"github.com/davecgh/go-spew/spew"
	*/)

type AlibabaPriceInfoHandler struct {
	Region idrv.RegionInfo
	Client *ecs.Client
}

// // 주어진 이미지 id에 대한 이미지 사이즈 조회
// // -1 : 정보 조회 실패
// // deprecated
// func (vmHandler *AlibabaVMHandler) GetImageSize(ImageSystemId string) (int64, error) {
// 	cblogger.Debugf("ImageID : [%s]", ImageSystemId)

// 	imageRequest := ecs.CreateDescribeImagesRequest()
// 	imageRequest.Scheme = "https"

// 	imageRequest.ImageId = ImageSystemId
// 	imageRequest.ShowExpired = requests.NewBoolean(true) //default는 false, false일 때는 최신 이미지 정보만 조회됨, true일 때는 오래된 이미지도 조회

// 	response, err := vmHandler.Client.DescribeImages(imageRequest)
// 	if err != nil {
// 		cblogger.Error(err)
// 		return -1, err
// 	}

// 	if len(response.Images.Image) > 0 {
// 		fmt.Println(response.Images.Image[0].Size)
// 		imageSize := int64(response.Images.Image[0].Size)
// 		return imageSize, nil

// 	} else {
// 		cblogger.Error("요청된 Image 정보[" + ImageSystemId + "]를 찾을 수 없습니다.")
// 		return -1, errors.New("요청된 Image 정보[" + ImageSystemId + "]를 찾을 수 없습니다.")
// 	}
// }

// // 참고 : VM 생성 시 인증 방식은 KeyPair 또는 ID&PWD 방식이 가능하지만 계정은 모두 root  - 비번 조회 기능은 없음
// //
// //	비밀번호는 8-30자로서 대문자, 소문자, 숫자 및/또는 특수 문자가 포함되어야 합니다.
// //
// // @TODO : root 계정의 비번만 설정 가능한 데 다른 계정이 요청되었을 경우 예외 처리할 것인지.. 아니면 비번을 설정할 것인지 확인 필요.
// // @TODO : PublicIp 요금제 방식과 대역폭 설정 방법 논의 필요
// func (vmHandler *AlibabaVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
// 	cblogger.Debug(vmReqInfo)
// 	//spew.Dump(vmReqInfo)

// 	/* 2021-10-26 이슈 #480에 의해 제거
// 	// 2021-04-28 cbuser 추가에 따른 Local KeyPair만 VM 생성 가능하도록 강제
// 	//=============================
// 	// KeyPair의 PublicKey 정보 처리
// 	//=============================
// 	cblogger.Infof("[%s] KeyPair 조회 시작", vmReqInfo.KeyPairIID.SystemId)
// 	keypairHandler := AlibabaKeyPairHandler{
// 		//CredentialInfo:
// 		Region: vmHandler.Region,
// 		Client: vmHandler.Client,
// 	}
// 	cblogger.Info(keypairHandler)
// 	keyPairInfo, errKeyPair := keypairHandler.GetKey(vmReqInfo.KeyPairIID)
// 	if errKeyPair != nil {
// 		cblogger.Error(errKeyPair)
// 		return irs.VMInfo{}, errKeyPair
// 	}
// 	*/

// 	//=============================
// 	// UserData생성 처리
// 	//=============================
// 	/*
// 		package_update: true
// 		packages:
// 		 - sudo
// 		users:
// 		  - default
// 		  - name: cb-user
// 			groups: sudo
// 			shell: /bin/bash
// 			sudo: ['ALL=(ALL) NOPASSWD:ALL']
// 			ssh-authorized-keys:
// 			  - ssh-rsa AAAAB3NzaC1y
// 	*/
// 	/*
// 		//sudo 패키지 설치
// 		//userData := "#cloud-config\npackage_update: true\npackages:\n  - sudo\nusers:\n  - default\n  - name: " + CBDefaultVmUserName + "\n    groups: sudo\n    shell: /bin/bash\n    sudo: ['ALL=(ALL) NOPASSWD:ALL']\n    ssh-authorized-keys:\n      - "
// 		//sudo 그룹 사용
// 		//userData := "#cloud-config\nusers:\n  - default\n  - name: " + CBDefaultVmUserName + "\n    groups: sudo\n    shell: /bin/bash\n    sudo: ['ALL=(ALL) NOPASSWD:ALL']\n    ssh-authorized-keys:\n      - "
// 		//그룹 제거
// 		userData := "#cloud-config\nusers:\n  - default\n  - name: " + CBDefaultVmUserName + "\n    shell: /bin/bash\n    sudo: ['ALL=(ALL) NOPASSWD:ALL']\n    ssh-authorized-keys:\n      - "
// 		userData = userData + keyPairInfo.PublicKey
// 		userDataBase64 := base64.StdEncoding.EncodeToString([]byte(userData))
// 		cblogger.Infof("===== userData ===")
// 		spew.Dump(userDataBase64)
// 	*/

// 	vmImage, err := DescribeImageByImageId(vmHandler.Client, vmHandler.Region, vmReqInfo.ImageIID, false)
// 	if err != nil {
// 		cblogger.Error(err)
// 		errMsg := "요청된 이미지의 정보를 조회할 수 없습니다." + err.Error()
// 		return irs.VMInfo{}, errors.New(errMsg)
// 	}

// 	isWindows := false
// 	osType := GetOsType(vmImage) //"OSType": "windows"
// 	if osType == "windows" {
// 		isWindows = true

// 		err := cdcom.ValidateWindowsPassword(vmReqInfo.VMUserPasswd)
// 		if err != nil {
// 			return irs.VMInfo{}, err
// 		}
// 	}

// 	//=============================
// 	// UserData생성 처리(File기반)
// 	//=============================
// 	// 향후 공통 파일이나 외부에서 수정 가능하도록 cloud-init 스크립트 파일로 설정
// 	rootPath := os.Getenv("CBSPIDER_ROOT")
// 	fileDataCloudInit, err := ioutil.ReadFile(rootPath + CBCloudInitFilePath)
// 	if err != nil {
// 		cblogger.Error(err)
// 		return irs.VMInfo{}, err
// 	}
// 	userData := string(fileDataCloudInit)
// 	//userData = strings.ReplaceAll(userData, "{{username}}", CBDefaultVmUserName)
// 	//userData = strings.ReplaceAll(userData, "{{public_key}}", keyPairInfo.PublicKey)
// 	userDataBase64 := base64.StdEncoding.EncodeToString([]byte(userData))
// 	cblogger.Debugf("cloud-init data : [%s]", userDataBase64)

// 	//=============================
// 	// 보안그룹 처리 - SystemId 기반
// 	//=============================
// 	cblogger.Debug("SystemId 기반으로 처리하기 위해 IID 기반의 보안그룹 배열을 SystemId 기반 보안그룹 배열로 조회및 변환함.")
// 	var newSecurityGroupIds []string
// 	//var firstSecurityGroupId string

// 	for _, sgId := range vmReqInfo.SecurityGroupIIDs {
// 		cblogger.Debugf("보안그룹 변환 : [%s]", sgId)
// 		newSecurityGroupIds = append(newSecurityGroupIds, sgId.SystemId)
// 		//firstSecurityGroupId = sgId.SystemId
// 		//break
// 	}

// 	cblogger.Debug("보안그룹 변환 완료")
// 	cblogger.Debug(newSecurityGroupIds)

// 	//request := ecs.CreateCreateInstanceRequest()	// CreateInstance는 PublicIp가 자동으로 할당되지 않음.
// 	request := ecs.CreateRunInstancesRequest() // RunInstances는 PublicIp가 자동으로 할당됨.
// 	request.Scheme = "https"

// 	request.InstanceChargeType = "PostPaid" //저렴한 실시간 요금으로 설정 //PrePaid: subscription.  / PostPaid: pay-as-you-go. Default value: PostPaid.
// 	request.ImageId = vmReqInfo.ImageIID.SystemId
// 	//request.SecurityGroupIds *[]string
// 	request.SecurityGroupIds = &newSecurityGroupIds
// 	//request.SecurityGroupId = firstSecurityGroupId // string 타입이라 첫번째 보안 그룹만 적용
// 	//request.SecurityGroupId =  "[\"" + newSecurityGroupIds + "\"]" // string 타입이라 첫번째 보안 그룹만 적용

// 	request.InstanceName = vmReqInfo.IId.NameId
// 	//request.HostName = vmReqInfo.IId.NameId	// OS 호스트 명
// 	request.InstanceType = vmReqInfo.VMSpecName

// 	request.ZoneId = vmHandler.Region.Zone // Disk의 경우 zone dependency가 있어 Zone 명시해야 함.(disk가 없으면 무시해도 됨.)

// 	// windows 일 떄는 password 만 set, keypairName은 비움.
// 	// 다른 os일 때 password는 cb-user의 password 로 사용
// 	if isWindows {
// 		request.Password = vmReqInfo.VMUserPasswd
// 	} else {
// 		request.KeyPairName = vmReqInfo.KeyPairIID.SystemId

// 		// cb user 추가
// 		request.Password = vmReqInfo.VMUserPasswd //값에는 8-30자가 포함되고 대문자, 소문자, 숫자 및/또는 특수 문자가 포함되어야 합니다.
// 		request.UserData = userDataBase64         // cbuser 추가
// 	}

// 	request.VSwitchId = vmReqInfo.SubnetIID.SystemId

// 	//==============
// 	//PublicIp 설정
// 	//==============
// 	//Public Ip를 생성하기 위해서는 과금형태와 대역폭(1 Mbit/s이상)을 지정해야 함.
// 	//PayByTraffic(기본값) : 트래픽 기준 결제(GB 단위) - 트래픽 기준 결제(GB 단위)를 사용하면 대역폭 사용료가 시간별로 청구
// 	//PayByBandwidth : 대역폭 사용료는 구독 기반이고 ECS 인스턴스 사용료에 포함 됨.
// 	request.InternetChargeType = "PayByBandwidth"           //Public Ip요금 방식을 1시간 단위(PayByBandwidth) 요금으로 설정 / PayByTraffic(기본값) : 1GB단위 시간당 트래픽 요금 청구
// 	request.InternetMaxBandwidthOut = requests.Integer("5") // 0보다 크면 Public IP가 할당 됨 - 최대 아웃 바운드 공용 대역폭 단위 : Mbit / s 유효한 값 : 0 ~ 100

// 	//=============================
// 	// Root Disk Type 변경
// 	//=============================
// 	if vmReqInfo.RootDiskType == "" || strings.EqualFold(vmReqInfo.RootDiskType, "default") {
// 		//디스크 정보가 없으면 건드리지 않음.
// 	} else {
// 		request.SystemDiskCategory = vmReqInfo.RootDiskType
// 	}

// 	//=============================
// 	// Root Disk Size 변경
// 	//=============================
// 	if vmReqInfo.RootDiskSize == "" || strings.EqualFold(vmReqInfo.RootDiskSize, "default") {
// 		//디스크 정보가 없으면 건드리지 않음.
// 	} else {

// 		rootDiskSize, err := strconv.ParseInt(vmReqInfo.RootDiskSize, 10, 64)
// 		if err != nil {
// 			cblogger.Error(err)
// 			return irs.VMInfo{}, err
// 		}

// 		// cloudos_meta 에 DiskType, min, max 값 정의
// 		cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo("ALIBABA")
// 		arrDiskSizeOfType := cloudOSMetaInfo.RootDiskSize

// 		fmt.Println("arrDiskSizeOfType: ", arrDiskSizeOfType)

// 		diskSizeValue := DiskSize{}
// 		// DiskType default 도 건드리지 않음
// 		if vmReqInfo.RootDiskType == "" || strings.EqualFold(vmReqInfo.RootDiskType, "default") {

// 			//diskSizeArr := strings.Split(arrDiskSizeOfType[0], "|")
// 			//diskSizeValue.diskType = diskSizeArr[0]
// 			//diskSizeValue.unit = diskSizeArr[3]
// 			//diskSizeValue.diskMinSize, err = strconv.ParseInt(diskSizeArr[1], 10, 64)
// 			//if err != nil {
// 			//	cblogger.Error(err)
// 			//	return irs.VMInfo{}, err
// 			//}
// 			//
// 			//diskSizeValue.diskMaxSize, err = strconv.ParseInt(diskSizeArr[2], 10, 64)
// 			//if err != nil {
// 			//	cblogger.Error(err)
// 			//	return irs.VMInfo{}, err
// 			//}
// 		} else {
// 			// diskType이 있으면 type에 맞는 min, max, default 값 사용
// 			isExists := false
// 			for idx, _ := range arrDiskSizeOfType {
// 				diskSizeArr := strings.Split(arrDiskSizeOfType[idx], "|")
// 				fmt.Println("diskSizeArr: ", diskSizeArr)

// 				if strings.EqualFold(vmReqInfo.RootDiskType, diskSizeArr[0]) {
// 					diskSizeValue.diskType = diskSizeArr[0]
// 					diskSizeValue.unit = diskSizeArr[3]
// 					diskSizeValue.diskMinSize, err = strconv.ParseInt(diskSizeArr[1], 10, 64)
// 					if err != nil {
// 						cblogger.Error(err)
// 						return irs.VMInfo{}, err
// 					}

// 					diskSizeValue.diskMaxSize, err = strconv.ParseInt(diskSizeArr[2], 10, 64)
// 					if err != nil {
// 						cblogger.Error(err)
// 						return irs.VMInfo{}, err
// 					}
// 					isExists = true
// 				}
// 			}
// 			if !isExists {
// 				return irs.VMInfo{}, errors.New("Invalid Root Disk Type : " + vmReqInfo.RootDiskType)
// 			}

// 			if rootDiskSize < diskSizeValue.diskMinSize {
// 				fmt.Println("Disk Size Error!!: ", rootDiskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
// 				//return irs.VMInfo{}, errors.New("Requested disk size cannot be smaller than the minimum disk size, invalid")
// 				return irs.VMInfo{}, errors.New("Root Disk Size must be at least the default size (" + strconv.FormatInt(diskSizeValue.diskMinSize, 10) + " GB).")
// 			}

// 			if rootDiskSize > diskSizeValue.diskMaxSize {
// 				fmt.Println("Disk Size Error!!: ", rootDiskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
// 				//return irs.VMInfo{}, errors.New("Requested disk size cannot be larger than the maximum disk size, invalid")
// 				return irs.VMInfo{}, errors.New("Root Disk Size must be smaller than the maximum size (" + strconv.FormatInt(diskSizeValue.diskMaxSize, 10) + " GB).")
// 			}
// 		}

// 		//imageSize, err := vmHandler.GetImageSize(vmReqInfo.ImageIID.SystemId)
// 		imageSize := int64(vmImage.Size)
// 		if imageSize < 0 {
// 			return irs.VMInfo{}, errors.New("요청된 이미지의 기본 사이즈 정보를 조회할 수 없습니다.")
// 		} else {
// 			if rootDiskSize < imageSize {
// 				fmt.Println("Disk Size Error!!: ", rootDiskSize)
// 				return irs.VMInfo{}, errors.New("Root Disk Size must be larger then the image size (" + strconv.FormatInt(imageSize, 10) + " GB).")
// 			}

// 		}

// 		request.SystemDiskSize = vmReqInfo.RootDiskSize

// 	}

// 	// Windows OS 처리
// 	//"Platform": "Windows Server 2012",
// 	//"OSName": "Windows Server  2012 R2 数据中心版 64位英文版",
// 	//"OSType": "windows",
// 	if isWindows {
// 		//The password must be 8 to 30 characters in length
// 		//and contain at least three of the following character types: uppercase letters, lowercase letters, digits, and special characters.
// 		//Special characters include: // ( ) ` ~ ! @ # $ % ^ & * - _ + = | { } [ ] : ; ' < > , . ? /

// 	}

// 	//=============================
// 	// VM생성 처리
// 	//=============================
// 	cblogger.Debug("Create EC2 Instance")
// 	cblogger.Debug(request)

// 	// logger for HisCall
// 	callogger := call.GetLogger("HISCALL")
// 	callLogInfo := call.CLOUDLOGSCHEMA{
// 		CloudOS:      call.ALIBABA,
// 		RegionZone:   vmHandler.Region.Zone,
// 		ResourceType: call.VM,
// 		ResourceName: vmReqInfo.IId.NameId,
// 		CloudOSAPI:   "RunInstances()",
// 		ElapsedTime:  "",
// 		ErrorMSG:     "",
// 	}
// 	callLogStart := call.Start()

// 	response, err := vmHandler.Client.RunInstances(request)
// 	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

// 	if err != nil {
// 		callLogInfo.ErrorMSG = err.Error()
// 		callogger.Error(call.String(callLogInfo))
// 		cblogger.Error(err.Error())
// 		return irs.VMInfo{}, err
// 	}
// 	callogger.Info(call.String(callLogInfo))
// 	//spew.Dump(response)

// 	if len(response.InstanceIdSets.InstanceIdSet) < 1 {
// 		return irs.VMInfo{}, errors.New("No errors have occurred, but no VMs have been created.")
// 	}

// 	//=========================================
// 	// VM 정보를 조회할 수 있을 때까지 대기
// 	//=========================================
// 	newVmIID := irs.IID{SystemId: response.InstanceIdSets.InstanceIdSet[0]}

// 	//VM 생성 요청 후에는 곧바로 VM 정보를 조회할 수 없기 때문에 VM 상태를 조회할 수 있는 NotExist 상태가 아닐 때까지만 대기 함.
// 	//2021-05-11 WaitForRun을 호출하지 않아도 GetVM() 호출 시 에러가 발생하지 않는 것은 확인했음. (우선은 정책이 최종 확정이 아니라서 WaitForRun을 사용하도록 원복함.)
// 	//curStatus, errStatus := vmHandler.WaitForExist(newVmIID) // 20210511 - NotExist 상태가 아닐 때 까지만 대기
// 	curStatus, errStatus := vmHandler.WaitForRun(newVmIID) // 20210511 아직 정책이 최종 확정이 아니라서 WaitForRun을 사용하도록 원복함
// 	if errStatus != nil {
// 		cblogger.Error(errStatus.Error())
// 		return irs.VMInfo{}, nil
// 	}
// 	cblogger.Info("==>생성된 VM[%s]의 현재 상태[%s]", newVmIID, curStatus)

// 	// dataDisk attach
// 	for _, dataDiskIID := range vmReqInfo.DataDiskIIDs {
// 		err = AttachDisk(vmHandler.Client, vmHandler.Region, newVmIID, dataDiskIID)
// 		if err != nil {
// 			return irs.VMInfo{}, errors.New("Instance created but attach disk failed " + err.Error())
// 		}
// 	}

// 	//vmInfo, errVmInfo := vmHandler.GetVM(irs.IID{SystemId: response.InstanceId})
// 	vmInfo, errVmInfo := vmHandler.GetVM(newVmIID)
// 	if errVmInfo != nil {
// 		cblogger.Error(errVmInfo.Error())
// 		return irs.VMInfo{}, errVmInfo
// 	}

// 	// VM을 삭제해도 DataDisk는 삭제되지 않도록 Attribute 설정
// 	diskRequest := ecs.CreateModifyDiskAttributeRequest()
// 	diskRequest.Scheme = "https"
// 	diskRequest.DeleteWithInstance = requests.NewBoolean(false)

// 	diskIds := []string{}

// 	for _, dataDiskId := range vmInfo.DataDiskIIDs {
// 		diskIds = append(diskIds, dataDiskId.SystemId)
// 	}

// 	diskRequest.DiskIds = &diskIds

// 	_, diskErr := vmHandler.Client.ModifyDiskAttribute(diskRequest)
// 	if err != nil {
// 		return irs.VMInfo{}, errors.New("Instance created but modifying disk attributes failed " + diskErr.Error())
// 	}

// 	vmInfo.IId.NameId = vmReqInfo.IId.NameId

// 	//VM 생성 시 요청한 계정 정보가 있을 경우 사용된 계정 정보를 함께 전달 함.
// 	if vmReqInfo.VMUserPasswd != "" {
// 		vmInfo.VMUserPasswd = vmReqInfo.VMUserPasswd
// 		vmInfo.VMUserId = "root"
// 	}
// 	return vmInfo, nil
// }

func (priceInfoHandler *AlibabaPriceInfoHandler) ListProductFamily(regionName string) ([]string, error) {
	var familyList []string

	instanceFamilyRequest := ecs.CreateDescribeInstanceTypeFamiliesRequest()
	instanceFamilyRequest.Scheme = "https"

	// //RegionId: tea.String("cn-hongkong"),
	instanceFamilyRequest.RegionId = regionName

	response, err := priceInfoHandler.Client.DescribeInstanceTypeFamilies(instanceFamilyRequest)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}
	log.Println("rr")
	log.Println(response)

	for _, instanceFamily := range response.InstanceTypeFamilies.InstanceTypeFamily {
		// type InstanceTypeFamily struct {
		// 	Generation           string `json:"Generation" xml:"Generation"`
		// 	InstanceTypeFamilyId string `json:"InstanceTypeFamilyId" xml:"InstanceTypeFamilyId"`
		// }
		instanceTypeFamilyId := instanceFamily.InstanceTypeFamilyId
		//generation := instanceFamily.Generation
		log.Println(instanceFamily)
		familyList = append(familyList, instanceTypeFamilyId)
	}

	return familyList, nil
}

func (priceInfoHandler *AlibabaPriceInfoHandler) GetPriceInfo(productFamily string, regionName string, filter irs.KeyValue) (string, error) {
	log.Println(productFamily)
	log.Println(regionName)

	// ecs-4 ecs.smt-bw42f1d9-d
	// cn-hongkong
	var priceInfo string

	priceRequest := ecs.CreateDescribePriceRequest()
	priceRequest.Scheme = "https"

	// //RegionId: tea.String("cn-hongkong"),
	priceRequest.RegionId = priceInfoHandler.Region.Region
	priceRequest.InstanceType = productFamily

	priceRequest.RegionId = "cn-hongkong"
	priceRequest.InstanceType = "ecs-4 ecs.smt-bw42f1d9-d"

	response, err := priceInfoHandler.Client.DescribePrice(priceRequest)
	if err != nil {
		cblogger.Error(err)
		return priceInfo, err
	}
	log.Println("rr")
	log.Println(response)

	// "PriceInfo": {
	// 	"Price": {
	// 	  "OriginalPrice": 0.086,
	// 	  "ReservedInstanceHourPrice": 0,
	// 	  "DiscountPrice": 0,
	// 	  "Currency": "USD",
	// 	  "TradePrice": 0.086
	// 	},
	// 	"Rules": {
	// 	  "Rule": []
	// 	}
	//   }
	price := response.PriceInfo.Price
	log.Print(price)
	return priceInfo, nil
}
