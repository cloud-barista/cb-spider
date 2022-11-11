// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// program by ysjeon@mz.co.kr, 2019.07.
// modify by devunet@mz.co.kr, 2019.11.

package resources

import (
	"context"
	"errors"
	_ "errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	//keypair "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	cdcom "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	//cim "github.com/cloud-barista/cb-spider/cloud-info-manager"
	compute "google.golang.org/api/compute/v1"
	// "golang.org/x/oauth2/google"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type GCPVMHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

// https://cloud.google.com/compute/docs/reference/rest/v1/instances
// https://cloud.google.com/compute/docs/disks#disk-types
func (vmHandler *GCPVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	// Set VM Create Information
	// GCP 는 reqinfo에 ProjectID를 받아야 함.
	cblogger.Info(vmReqInfo)

	//ctx := vmHandler.Ctx
	vmName := vmReqInfo.IId.NameId
	projectID := vmHandler.Credential.ProjectID
	prefix := "https://www.googleapis.com/compute/v1/projects/" + projectID

	region := vmHandler.Region.Region
	zone := vmHandler.Region.Zone
	// email을 어디다가 넣지? 이것또한 문제넹
	clientEmail := vmHandler.Credential.ClientEmail

	//imageURL := "projects/ubuntu-os-cloud/global/images/ubuntu-minimal-1804-bionic-v20191024"
	imageURL := vmReqInfo.ImageIID.SystemId
	isMyImage := false
	isWindows := false

	// public Image vs myImage
	if vmReqInfo.ImageType == irs.MyImage {
		isMyImage = true
		imageURL = "global/machineImages/" + imageURL // MyImage는 ImageURL 형태가 아니라 ID를 사용하므로 앞에 URL 형태를 붙여줌
	}
	// 이미지 사이즈 추출
	//var projectIdForImage string
	var imageSize int64
	//imageUrlArr := strings.Split(imageURL, "/")
	//imageName := imageUrlArr[len(imageUrlArr)-1]

	var pubKey string
	if isMyImage {

		//spider-myimage-1-cdlkbi2t39h9lqh14i90
		//projects/csta-349809/global/machineImages",

		machineImage, err := GetMachineImageInfo(vmHandler.Client, projectID, vmReqInfo.ImageIID.SystemId)
		if err != nil {
			return irs.VMInfo{}, err
		}

		// osFeatures := machineImage.GuestOsFeatures

		// for _, feature := range osFeatures {
		// 	if feature.Type == "WINDOWS" {
		// 		isWindows = true
		// 		break
		// 	}
		// }

		// disks := machineImage.SavedDisks
		// for _, disk := range disks {
		// 	if disk
		// 		isWindows = true
		// 		break
		// 	}
		// }
		ip := machineImage.InstanceProperties
		disks := ip.Disks
		for _, disk := range disks {
			if disk.Boot { // Boot Device
				//diskSize := disk.DiskSizeGb
				imageSize = disk.DiskSizeGb // image size가 맞나??
				cblogger.Info(imageSize)
				osFeatures := disk.GuestOsFeatures
				for _, feature := range osFeatures {
					if feature.Type == "WINDOWS" {
						isWindows = true
						break
					}
				}
				cblogger.Info(isWindows)
			}
		}

		//imageSize = machineImage.DiskSizeGb

	} else {

		computeImage, err := GetPublicImageInfo(vmHandler.Client, vmReqInfo.ImageIID)
		if err != nil {
			cblogger.Info("GetPublicImageInfo err : ", err)
			return irs.VMInfo{}, err
		}

		// projectIdForImage = imageUrlArr[6]
		// imageResp, err := vmHandler.Client.Images.Get(projectIdForImage, imageName).Do()
		// if err != nil {
		// 	log.Fatal(err)
		// }
		// spew.Dump(imageResp)
		//osFeatures := imageResp.GuestOsFeatures
		osFeatures := computeImage.GuestOsFeatures

		for _, feature := range osFeatures {
			if feature.Type == "WINDOWS" {
				isWindows = true
			}
		}

		imageSize = computeImage.DiskSizeGb

	}
	cblogger.Info("isMyImage = ", isMyImage)
	cblogger.Info("isWindows = ", isWindows)

	/* // 2020-05-15 Name 기반 로직을 임의로 막아 놓음 - 다음 버전에 적용 예정. 현재는 URL 방식
	//이미지 URL처리
	cblogger.Infof("[%s] Image Name에 해당하는 Image Url 정보를 조회합니다.", vmReqInfo.ImageIID.SystemId)
	imageHandler := GCPImageHandler{Credential: vmHandler.Credential, Region: vmHandler.Region, Client: vmHandler.Client}
	imageInfo, errImage := imageHandler.FindImageInfo(vmReqInfo.ImageIID.SystemId)
	if errImage != nil {
		return irs.VMInfo{}, nil
	}
	cblogger.Infof("ImageName: [%s] ---> ImageUrl : [%s]", vmReqInfo.ImageIID.SystemId, imageInfo.ImageUrl)
	imageURL = imageInfo.ImageUrl
	*/

	//PublicIP처리
	// var publicIPAddress string
	// cblogger.Info("PublicIp 처리 시작")
	// publicIpHandler := GCPPublicIPHandler{
	// 	vmHandler.Region, vmHandler.Ctx, vmHandler.Client, vmHandler.Credential}

	//PublicIp를 전달 받았으면 전달 받은 Ip를 할당
	// if vmReqInfo.PublicIPId != "" {
	// 	cblogger.Info("PublicIp 정보 조회 시작")
	// 	publicIPInfo, err := publicIpHandler.GetPublicIP(vmReqInfo.PublicIPId)
	// 	if err != nil {
	// 		cblogger.Error(err)
	// 		return irs.VMInfo{}, err
	// 	}
	// 	cblogger.Info("PublicIp 조회됨")
	// 	cblogger.Info(publicIPInfo)
	// 	publicIPAddress = publicIPInfo.PublicIP
	// } else { //PublicIp가 없으면 직접 생성
	// 	cblogger.Info("PublicIp 생성 시작")
	// 	// PublicIPHandler  불러서 처리 해야 함.
	// 	publicIpName := vmReqInfo.VMName
	// 	publicIpReqInfo := irs.PublicIPReqInfo{Name: publicIpName}
	// 	publicIPInfo, err := publicIpHandler.CreatePublicIP(publicIpReqInfo)

	// 	if err != nil {
	// 		cblogger.Error(err)
	// 		return irs.VMInfo{}, err
	// 	}
	// 	cblogger.Info("PublicIp 생성됨")
	// 	cblogger.Info(publicIPInfo)
	// 	publicIPAddress = publicIPInfo.PublicIP
	// }

	/*
		type GCPImageHandler struct {
			Region     idrv.RegionInfo
			Ctx        context.Context
			Client     *compute.Service
			Credential idrv.CredentialInfo
		}
	*/

	// keyPair 정보는 window가 아닐 때만 Set
	if !isWindows {
		//KEYPAIR HANDLER
		keypairHandler := GCPKeyPairHandler{
			vmHandler.Credential, vmHandler.Region}
		keypairInfo, errKeypair := keypairHandler.GetKey(vmReqInfo.KeyPairIID)
		if errKeypair != nil {
			cblogger.Error(errKeypair)
			return irs.VMInfo{}, errKeypair
		}

		cblogger.Debug("공개키 생성")
		publicKey, errPub := cdcom.MakePublicKeyFromPrivateKey(keypairInfo.PrivateKey)
		if errPub != nil {
			cblogger.Error(errPub)
			return irs.VMInfo{}, errPub
		}

		//pubKey := "cb-user:" + keypairInfo.PublicKey
		pubKey = "cb-user:" + strings.TrimSpace(publicKey) + " " + "cb-user"
		cblogger.Debug("keypairInfo 정보")
		spew.Dump(keypairInfo)
	}

	// Security Group Tags
	var securityTags []string
	for _, item := range vmReqInfo.SecurityGroupIIDs {
		//securityTags = append(securityTags, item.NameId)
		securityTags = append(securityTags, item.SystemId)
	}
	cblogger.Info("Security Tags 정보 : ", securityTags)
	//networkURL := prefix + "/global/networks/" + vmReqInfo.VpcIID.NameId
	networkURL := prefix + "/global/networks/" + vmReqInfo.VpcIID.SystemId
	//subnetWorkURL := prefix + "/regions/" + region + "/subnetworks/" + vmReqInfo.SubnetIID.NameId
	subnetWorkURL := prefix + "/regions/" + region + "/subnetworks/" + vmReqInfo.SubnetIID.SystemId

	cblogger.Info("networkURL 정보 : ", networkURL)
	cblogger.Info("subnetWorkURL 정보 : ", subnetWorkURL)

	instance := &compute.Instance{
		Name: vmName,
		Metadata: &compute.Metadata{
			Items: []*compute.MetadataItems{
				{Key: "ssh-keys",
					Value: &pubKey},
			},
		},
		Labels: map[string]string{
			//"keypair": strings.ToLower(vmReqInfo.KeyPairIID.NameId),
			"keypair": strings.ToLower(vmReqInfo.KeyPairIID.SystemId),
		},
		Description: "compute sample instance",
		MachineType: prefix + "/zones/" + zone + "/machineTypes/" + vmReqInfo.VMSpecName,
		Disks: []*compute.AttachedDisk{
			{
				AutoDelete: true,
				Boot:       true,
				Type:       "PERSISTENT",
				// InitializeParams: &compute.AttachedDiskInitializeParams{
				// 	//DiskName:    vmName, //disk name 도 매번 바뀌어야 하는 값, 루트 디스크 이름은 특별히 지정하지 않는 경우 vm이름으로 생성됨
				// 	SourceImage: imageURL,
				// },
			},
		},
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				AccessConfigs: []*compute.AccessConfig{
					{
						Type: "ONE_TO_ONE_NAT",
						Name: "External NAT", // default

					},
				},
				Network:    networkURL,
				Subnetwork: subnetWorkURL,
			},
		},
		ServiceAccounts: []*compute.ServiceAccount{
			{
				Email: clientEmail,
				Scopes: []string{
					compute.DevstorageFullControlScope,
					compute.ComputeScope,
				},
			},
		},
		Tags: &compute.Tags{
			Items: securityTags,
		},
	}

	//Windows OS인 경우 administrator 계정 비번 설정 및 계정 활성화
	if isWindows {
		err := cdcom.ValidateWindowsPassword(vmReqInfo.VMUserPasswd)
		if err != nil {
			return irs.VMInfo{}, err
		}

		winOsMeta := "net user \"administrator\" \"" + vmReqInfo.VMUserPasswd + "\"\nnet user administrator /active:yes"
		winOsPwd := compute.MetadataItems{Key: "windows-startup-script-cmd", Value: &winOsMeta}
		instance.Metadata.Items = append(instance.Metadata.Items, &winOsPwd)
	}

	// imageType이 MyImage인 경우 SourceMachineImage Setting
	if isMyImage {
		instance.SourceMachineImage = imageURL
	} else {
		instance.Disks[0].InitializeParams = &compute.AttachedDiskInitializeParams{
			SourceImage: imageURL,
		}

		//이슈 #348에 의해 RootDisk 및 사이즈 변경 기능 지원
		//=============================
		// Root Disk Type 변경
		//=============================

		//var validDiskSize = ""
		if vmReqInfo.RootDiskType == "" || strings.EqualFold(vmReqInfo.RootDiskType, "default") {
			//디스크 정보가 없으면 건드리지 않음.
		} else {
			//https://cloud.google.com/compute/docs/disks#disk-types
			instance.Disks[0].InitializeParams.DiskType = prefix + "/zones/" + zone + "/diskTypes/" + vmReqInfo.RootDiskType
		}

		//=============================
		// Root Disk Size 변경
		//=============================
		// if vmReqInfo.RootDiskSize == "" {
		// 	//디스크 정보가 없으면 건드리지 않음.
		// }

		//=============================
		// Root Disk Size 변경
		//=============================
		if vmReqInfo.RootDiskSize == "" || strings.EqualFold(vmReqInfo.RootDiskSize, "default") {
			//instance.Disks[0].InitializeParams.DiskSizeGb = diskSize.minSize
		} else {

			iDiskSize, err := strconv.ParseInt(vmReqInfo.RootDiskSize, 10, 64)
			if err != nil {
				cblogger.Error(err)
				return irs.VMInfo{}, err
			}

			var diskType = ""

			if vmReqInfo.RootDiskType == "" || strings.EqualFold(vmReqInfo.RootDiskType, "default") {
				// cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo("GCP") // cloudos_meta 에 DiskType, min, max 값 정의 되어있음.
				// if err != nil {
				// 	cblogger.Error(err)
				// 	return irs.VMInfo{}, err
				// }
				// diskType = cloudOSMetaInfo.RootDiskType[0]
			} else {
				diskType = vmReqInfo.RootDiskType

				// RootDiskType을 조회하여 diskSize의 min, max, default값 추출 한 뒤 입력된 diskSize가 있으면 비교시 사용
				diskSizeResp, err := vmHandler.Client.DiskTypes.Get(projectID, zone, diskType).Do()
				if err != nil {
					fmt.Println("Invalid Disk Type Error!!")
					return irs.VMInfo{}, err
				}

				fmt.Printf("valid disk size: %#v\n", diskSizeResp.ValidDiskSize)

				//valid disk size 정의
				re := regexp.MustCompile("GB-?") //ex) 10GB-65536GB
				diskSizeArr := re.Split(diskSizeResp.ValidDiskSize, -1)
				diskMinSize, err := strconv.ParseInt(diskSizeArr[0], 10, 64)
				if err != nil {
					cblogger.Error(err)
					return irs.VMInfo{}, err
				}

				diskMaxSize, err := strconv.ParseInt(diskSizeArr[1], 10, 64)
				if err != nil {
					cblogger.Error(err)
					return irs.VMInfo{}, err
				}

				// diskUnit := "GB" // 기본 단위는 GB

				if iDiskSize < diskMinSize {
					fmt.Println("Disk Size Error!!: ", iDiskSize)
					//return irs.VMInfo{}, errors.New("Requested disk size cannot be smaller than the minimum disk size, invalid")
					return irs.VMInfo{}, errors.New("Root Disk Size must be at least the default size (" + strconv.FormatInt(diskMinSize, 10) + " GB).")
				}

				if iDiskSize > diskMaxSize {
					fmt.Println("Disk Size Error!!: ", iDiskSize)
					//return irs.VMInfo{}, errors.New("Requested disk size cannot be larger than the maximum disk size, invalid")
					return irs.VMInfo{}, errors.New("Root Disk Size must be smaller than the maximum size (" + strconv.FormatInt(diskMaxSize, 10) + " GB).")
				}
			}

			//imageSize = imageResp.DiskSizeGb

			if iDiskSize < imageSize {
				fmt.Println("Disk Size Error!!: ", iDiskSize)
				return irs.VMInfo{}, errors.New("Root Disk Size must be larger then the image size (" + strconv.FormatInt(imageSize, 10) + " GB).")
			}

			instance.Disks[0].InitializeParams.DiskSizeGb = iDiskSize

		}
	}

	for _, dataDisk := range vmReqInfo.DataDiskIIDs {
		disk := compute.AttachedDisk{
			Source: prefix + "/zones/" + zone + "/disks/" + dataDisk.SystemId,
		}
		instance.Disks = append(instance.Disks, &disk)
	}

	cblogger.Info("VM 생성 시작")
	cblogger.Debug(instance)
	spew.Dump(instance)
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmName,
		CloudOSAPI:   "Insert()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	op, err1 := vmHandler.Client.Instances.Insert(projectID, zone, instance).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Info("VM 생성 요청 호출 완료")
	cblogger.Info(op)
	spew.Dump(op)
	if err1 != nil {
		callLogInfo.ErrorMSG = err1.Error()
		callogger.Error(call.String(callLogInfo))
		cblogger.Error("VM 생성 실패")
		cblogger.Error(err1)
		return irs.VMInfo{}, err1
	}
	callogger.Info(call.String(callLogInfo))

	/*
		js, err := op.MarshalJSON()
		if err != nil {
			cblogger.Info("VM 생성 실패")
			cblogger.Error(err)
			return irs.VMInfo{}, err
		}
		cblogger.Info("Insert vm to marshal Json : ", string(js))
		cblogger.Infof("Got compute.Operation, err: %#v, %v", op, err)
	*/

	// 이게 시작하는  api Start 내부 매개변수로 projectID, zone, InstanceID
	//vm, err := vmHandler.Client.Instances.Start(project string, zone string, instance string)

	//time.Sleep(time.Second * 10)

	//2021-05-11 WaitForRun을 호출하지 않아도 GetVM() 호출 시 에러가 발생하지 않는 것은 확인했음. (우선은 정책이 최종 확정이 아니라서 WaitForRun을 사용하도록 원복함.)
	vmStatus, _ := vmHandler.WaitForRun(irs.IID{NameId: vmName, SystemId: vmName})
	cblogger.Info("VM 상태 : ", vmStatus)

	cblogger.Info("VM 정보 조회 호출 - GetVM()")
	//만약 30초 이내에 VM이 Running 상태가 되지 않더라도 GetVM으로 VM의 정보 조회를 요청해 봄.
	vmInfo, errVmInfo := vmHandler.GetVM(irs.IID{NameId: vmName, SystemId: vmName})
	if errVmInfo != nil {
		cblogger.Errorf("[%s] VM을 생성했지만 정보 조회는 실패 함.", vmName)
		cblogger.Error(errVmInfo)

		return irs.VMInfo{}, errVmInfo
	}
	//ImageIId의 NameId는 사용자가 요청한 값으로 리턴
	vmInfo.ImageIId.NameId = vmReqInfo.ImageIID.NameId
	return vmInfo, nil

	/* 2020-05-13 Start & Get 요청 시의 리턴 정보 통일을 위해 기존 로직 임시 제거
	vm, err2 := vmHandler.Client.Instances.Get(projectID, zone, vmName).Context(ctx).Do()
	if err2 != nil {
		cblogger.Error(err2)
		return irs.VMInfo{}, err2
	}
	//vmInfo := vmHandler.mappingServerInfo(vm)
	var securityTag []irs.IID
	for _, item := range vm.Tags.Items {
		iId := irs.IID{
			NameId:   item,
			SystemId: item,
		}
		securityTag = append(securityTag, iId)
	}
	//var vpcHandler *GCPVPCHandler
	vmInfo := irs.VMInfo{
		IId: irs.IID{
			NameId: vm.Name,
			//SystemId: strconv.FormatUint(vm.Id, 10),
			SystemId: vm.Name,
		},
		Region: irs.RegionInfo{
			Region: vmHandler.Region.Region,
			Zone:   vmHandler.Region.Zone,
		},
		VMUserId:          "cb-user",
		NetworkInterface:  vm.NetworkInterfaces[0].Name,
		SecurityGroupIIds: securityTag,
		VMSpecName:        vm.MachineType,
		KeyPairIId: irs.IID{
			NameId:   vm.Labels["keypair"],
			SystemId: vm.Labels["keypair"],
		},
		ImageIId:  vmHandler.getImageInfo(vm.Disks[0].Source),
		PublicIP:  vm.NetworkInterfaces[0].AccessConfigs[0].NatIP,
		PrivateIP: vm.NetworkInterfaces[0].NetworkIP,
		VpcIID:    vmReqInfo.VpcIID,
		SubnetIID: vmReqInfo.SubnetIID,
		KeyValueList: []irs.KeyValue{
			{"SubNetwork", vm.NetworkInterfaces[0].Subnetwork},
			{"AccessConfigName", vm.NetworkInterfaces[0].AccessConfigs[0].Name},
			{"NetworkTier", vm.NetworkInterfaces[0].AccessConfigs[0].NetworkTier},
			{"DiskDeviceName", vm.Disks[0].DeviceName},
			{"DiskName", vm.Disks[0].Source},
		},
	}
	return vmInfo, nil
	*/
}

// stop이라고 보면 될듯
func (vmHandler *GCPVMHandler) SuspendVM(vmID irs.IID) (irs.VMStatus, error) {
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone
	//ctx := vmHandler.Ctx

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmID.SystemId,
		CloudOSAPI:   "Stop()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	//inst, err := vmHandler.Client.Instances.Stop(projectID, zone, vmID.SystemId).Context(ctx).Do()
	inst, err := vmHandler.GCPInstanceStop(projectID, zone, vmID.SystemId)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	spew.Dump(inst)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.VMStatus("Failed"), err
	}
	callogger.Info(call.String(callLogInfo))

	fmt.Println("instance stop status :", inst.Status)
	return irs.VMStatus("Suspending"), nil
}

// GCP Instance Stop
// Spider 의 suspendVM와 reboot에서 공통으로 사용하기 위해 별도로 뺌
// suspend/resume/reboot는 async 인데 다른 function에서 사용하려면 해당 operation이 종료됐는지 체크 필요
// 호출하는 function에 operaion을 전달하여 종료여부 판단이 필요하면 사용
func (vmHandler *GCPVMHandler) GCPInstanceStop(projectID string, zoneID string, gpcInstanceID string) (*compute.Operation, error) {
	ctx := vmHandler.Ctx
	inst, err := vmHandler.Client.Instances.Stop(projectID, zoneID, gpcInstanceID).Context(ctx).Do()
	return inst, err
}

func (vmHandler *GCPVMHandler) ResumeVM(vmID irs.IID) (irs.VMStatus, error) {

	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone
	ctx := vmHandler.Ctx

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmID.SystemId,
		CloudOSAPI:   "Start()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	inst, err := vmHandler.Client.Instances.Start(projectID, zone, vmID.SystemId).Context(ctx).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	spew.Dump(inst)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.VMStatus("Failed"), err
	}
	callogger.Info(call.String(callLogInfo))

	fmt.Println("instance resume status :", inst.Status)
	return irs.VMStatus("Resuming"), nil
}

// reboot vm : using reset function
// Suspend/Resume/Reboot 는 async 이므로 바로 return
func (vmHandler *GCPVMHandler) RebootVM(vmID irs.IID) (irs.VMStatus, error) {
	projectID := vmHandler.Credential.ProjectID
	//region := vmHandler.Region.Region
	zone := vmHandler.Region.Zone
	ctx := vmHandler.Ctx

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmID.SystemId,
		CloudOSAPI:   "SuspendVM()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	status, err := vmHandler.GetVMStatus(vmID)
	if err != nil {
		callogger.Info(err)
		return irs.VMStatus("Failed"), err
	}
	// running 상태일 때는 reset
	if status == "Running" {
		callogger.Info("vm의 상태가 running이므로 reset 호춯")
		operation, err := vmHandler.Client.Instances.Reset(projectID, zone, vmID.SystemId).Context(ctx).Do()

		if err != nil {
			callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
			callLogInfo.ErrorMSG = err.Error()
			callogger.Info(call.String(callLogInfo))
			callogger.Info(operation)
			return irs.VMStatus("Failed"), err
		}
	} else if status == "Suspended" {
		callogger.Info("vm의 상태가 Suspended이므로 ResumeVM 호춯")
		_, err := vmHandler.ResumeVM(vmID)
		if err != nil {
			return irs.VMStatus("Failed"), err
		}
	} else {
		// running/suspended 이외에는 비정상
		return irs.VMStatus("Failed"), errors.New(string("VM의 상태가 [" + status + "] 입니다."))
	}
	//callogger.Info(vmID)
	//callogger.Info(status)

	//operationType := 3 // operationZone := 3
	//err = WaitOperationComplete(vmHandler.Client, projectID, region, zone, operation.Name, operationType)

	//callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	//if err != nil {
	//	callLogInfo.ErrorMSG = err.Error()
	//	callogger.Info(call.String(callLogInfo))
	//	return irs.VMStatus("Failed"), err // stop 자체는 에러가 없으므로 wait 오류는 기록만.
	//}
	//callogger.Info(call.String(callLogInfo))

	return irs.VMStatus("Rebooting"), nil
}

// reboot : suspend -> resome
//func (vmHandler *GCPVMHandler) RebootVM(vmID irs.IID) (irs.VMStatus, error) {
//	projectID := vmHandler.Credential.ProjectID
//	region := vmHandler.Region.Region
//	zone := vmHandler.Region.Zone
//
//	// logger for HisCall
//	callogger := call.GetLogger("HISCALL")
//	callLogInfo := call.CLOUDLOGSCHEMA{
//		CloudOS:      call.GCP,
//		RegionZone:   vmHandler.Region.Zone,
//		ResourceType: call.VM,
//		ResourceName: vmID.SystemId,
//		CloudOSAPI:   "SuspendVM()",
//		ElapsedTime:  "",
//		ErrorMSG:     "",
//	}
//	callLogStart := call.Start()
//	//_, err := vmHandler.SuspendVM(vmID)
//	operation, err := vmHandler.GCPInstanceStop(projectID, zone, vmID.SystemId)
//
//	if err != nil {
//		callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
//		callLogInfo.ErrorMSG = err.Error()
//		callogger.Info(call.String(callLogInfo))
//		return irs.VMStatus("Failed"), err
//	}
//
//	operationZone := 3
//	err = WaitOperationComplete(vmHandler.Client, projectID, region, zone, operation.Name, operationZone)
//
//	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
//	if err != nil {
//		callLogInfo.ErrorMSG = err.Error()
//		callogger.Info(call.String(callLogInfo))
//		//return irs.VMStatus("Failed"), err	// stop 자체는 에러가 없으므로 wait 오류는 기록만.
//	}
//	callogger.Info(call.String(callLogInfo))
//
//	_, err2 := vmHandler.ResumeVM(vmID)
//	if err2 != nil {
//		return irs.VMStatus("Failed"), err2
//	}
//
//	return irs.VMStatus("Rebooting"), nil
//}

func (vmHandler *GCPVMHandler) TerminateVM(vmID irs.IID) (irs.VMStatus, error) {
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone
	ctx := vmHandler.Ctx

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmID.SystemId,
		CloudOSAPI:   "Delete()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	inst, err := vmHandler.Client.Instances.Delete(projectID, zone, vmID.SystemId).Context(ctx).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	spew.Dump(inst)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.VMStatus("Failed"), err
	}
	callogger.Info(call.String(callLogInfo))

	fmt.Println("instance status :", inst.Status)

	return irs.VMStatus("Terminating"), nil
}

func (vmHandler *GCPVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	//serverList, err := vmHandler.Client.ListAll(vmHandler.Ctx)
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: "",
		CloudOSAPI:   "List()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	serverList, err := vmHandler.Client.Instances.List(projectID, zone).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return nil, err
	}
	callogger.Info(call.String(callLogInfo))

	var vmStatusList []*irs.VMStatusInfo
	for _, s := range serverList.Items {
		if s.Name != "" {
			vmId := s.Name
			status, _ := vmHandler.GetVMStatus(irs.IID{NameId: vmId, SystemId: vmId})
			vmStatusInfo := irs.VMStatusInfo{
				IId: irs.IID{
					NameId: vmId,
					//SystemId: strconv.FormatUint(s.Id, 10),
					SystemId: vmId,
				},

				VmStatus: status,
			}
			vmStatusList = append(vmStatusList, &vmStatusInfo)
		}
	}

	return vmStatusList, nil
}

func ConvertVMStatusString(vmStatus string) (irs.VMStatus, error) {
	var resultStatus string
	cblogger.Infof("vmStatus : [%s]", vmStatus)

	if strings.EqualFold(vmStatus, "PROVISIONING") {
		resultStatus = "Creating"
		//resultStatus = "Resuming" // Resume 요청을 받아서 재기동되는 단계에도 Pending이 있기 때문에 Pending은 Resuming으로 맵핑함.
	} else if strings.EqualFold(vmStatus, "RUNNING") {
		resultStatus = "Running"
	} else if strings.EqualFold(vmStatus, "STOPPING") {
		resultStatus = "Suspending"
	} else if strings.EqualFold(vmStatus, "Terminated") {
		resultStatus = "Suspended"
	} else if strings.EqualFold(vmStatus, "STAGING") {
		resultStatus = "Resuming"
	} else {
		//resultStatus = "Failed"
		cblogger.Errorf("vmStatus [%s]와 일치하는 맵핑 정보를 찾지 못 함.", vmStatus)
		return irs.VMStatus("Failed"), errors.New(vmStatus + "와 일치하는 CB VM 상태정보를 찾을 수 없습니다.")
	}
	cblogger.Infof("VM 상태 치환 : [%s] ==> [%s]", vmStatus, resultStatus)
	return irs.VMStatus(resultStatus), nil
}

func (vmHandler *GCPVMHandler) GetVMStatus(vmID irs.IID) (irs.VMStatus, error) { // GCP의 ID는 uint64 이므로 GCP에서는 Name을 ID값으로 사용한다.
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmID.SystemId,
		CloudOSAPI:   "GetVMStatus()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	instanceView, err := vmHandler.Client.Instances.Get(projectID, zone, vmID.SystemId).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.VMStatus("Failed"), err
	}
	callogger.Info(call.String(callLogInfo))

	// Get powerState, provisioningState
	//vmStatus := instanceView.Status
	vmStatus, errStatus := ConvertVMStatusString(instanceView.Status)
	//return irs.VMStatus(vmStatus), err
	return vmStatus, errStatus
}

func (vmHandler *GCPVMHandler) ListVM() ([]*irs.VMInfo, error) {
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone
	cblogger.Info("VMLIST zone info :", zone)
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: "",
		CloudOSAPI:   "List()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	serverList, err := vmHandler.Client.Instances.List(projectID, zone).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		cblogger.Infof("해당존에 만들어진 Vm List 가 없음")
		return nil, err
	}
	callogger.Info(call.String(callLogInfo))

	var vmList []*irs.VMInfo
	for _, server := range serverList.Items {
		vmInfo := vmHandler.mappingServerInfo(server)
		vmList = append(vmList, &vmInfo)
	}

	return vmList, nil
}

func (vmHandler *GCPVMHandler) GetVM(vmID irs.IID) (irs.VMInfo, error) {
	projectID := vmHandler.Credential.ProjectID
	zone := vmHandler.Region.Zone

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmID.SystemId,
		CloudOSAPI:   "GetVM()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	vm, err := vmHandler.Client.Instances.Get(projectID, zone, vmID.SystemId).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))
	spew.Dump(vm)

	vmInfo := vmHandler.mappingServerInfo(vm)
	return vmInfo, nil
}

/*
GCP에서 instance 조회는 Project, ZONE 이 필수임.
경우에 따라서 Zone 없이 VM ID만으로 조회하느 기능이 필요하여
전체 목록에서 id를 filter해서 가져옴.
vmID는 project에서 unique
*/
func (vmHandler *GCPVMHandler) GetVmById(vmID irs.IID) (irs.VMInfo, error) {
	projectID := vmHandler.Credential.ProjectID

	vmInfo := irs.VMInfo{}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmID.SystemId,
		CloudOSAPI:   "GetVM()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	instanceListByzone, err := vmHandler.Client.Instances.AggregatedList(projectID).Filter("name=" + vmID.SystemId).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))
	foundVm := false
	for _, item := range instanceListByzone.Items {
		if foundVm {
			break // 찾았으면 더 돌 필요 없음.
		}
		if item.Instances != nil {
			for _, instance := range item.Instances {
				if strings.EqualFold(vmID.SystemId, instance.Name) {
					spew.Dump(instance)
					vmInfo = vmHandler.mappingServerInfo(instance)
					foundVm = true
					break
				}
			}
		}
	}

	return vmInfo, nil
}

// func getVmStatus(vl *compute.Service) string {
// 	var powerState, provisioningState string

// 	for _, stat := range vl {
// 		statArr := strings.Split(*stat.Code, "/")

// 		if statArr[0] == "PowerState" {
// 			powerState = statArr[1]
// 		} else if statArr[0] == "ProvisioningState" {
// 			provisioningState = statArr[1]
// 		}
// 	}

// 	// Set VM Status Info
// 	var vmState string
// 	if powerState != "" && provisioningState != "" {
// 		vmState = powerState + "(" + provisioningState + ")"
// 	} else if powerState != "" && provisioningState == "" {
// 		vmState = powerState
// 	} else if powerState == "" && provisioningState != "" {
// 		vmState = provisioningState
// 	} else {
// 		vmState = "-"
// 	}
// 	return vmState
// }

func (vmHandler *GCPVMHandler) mappingServerInfo(server *compute.Instance) irs.VMInfo {
	cblogger.Info("================맵핑=====================================")
	spew.Dump(server)
	fmt.Println("server: ", server)

	//var gcpHanler *GCPVMHandler
	vpcArr := strings.Split(server.NetworkInterfaces[0].Network, "/")
	subnetArr := strings.Split(server.NetworkInterfaces[0].Subnetwork, "/")
	vpcName := vpcArr[len(vpcArr)-1]
	subnetName := subnetArr[len(subnetArr)-1]
	// root disk의 type이 instance의 get으로 조회되지 않아서 getDiskInfo 호출
	diskInfo, err := vmHandler.getDiskInfo(server.Disks[0].Source)
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}
	}
	diskTypeArr := strings.Split(diskInfo.Type, "/")
	diskType := diskTypeArr[len(diskTypeArr)-1]

	type IIDBox struct {
		Items []irs.IID
	}

	var iIdBox IIDBox
	for _, item := range server.Tags.Items {
		iId := irs.IID{
			NameId:   item,
			SystemId: item,
		}
		iIdBox.Items = append(iIdBox.Items, iId)
	}

	var attachedDisk IIDBox
	for idx, disk := range server.Disks {
		// index 0은 root disk
		if idx > 0 {
			diskArr := strings.Split(disk.Source, "/")
			diskName := diskArr[len(diskArr)-1]
			diskIID := irs.IID{
				NameId:   diskName,
				SystemId: diskName,
			}
			attachedDisk.Items = append(attachedDisk.Items, diskIID)
		}
	}

	vmInfo := irs.VMInfo{
		IId: irs.IID{
			NameId: server.Name,
			//SystemId: strconv.FormatUint(server.Id, 10),
			SystemId: server.Name,
		},
		//VMSpecName: server.MachineType,

		Region: irs.RegionInfo{
			Region: vmHandler.Region.Region,
			Zone:   vmHandler.Region.Zone,
		},
		VMUserId:          "cb-user",
		NetworkInterface:  server.NetworkInterfaces[0].Name,
		SecurityGroupIIds: iIdBox.Items,
		KeyPairIId: irs.IID{
			NameId:   server.Labels["keypair"],
			SystemId: server.Labels["keypair"],
		},
		ImageIId:  vmHandler.getImageIID(server),
		PublicIP:  server.NetworkInterfaces[0].AccessConfigs[0].NatIP,
		PrivateIP: server.NetworkInterfaces[0].NetworkIP,
		VpcIID: irs.IID{
			NameId:   vpcName,
			SystemId: vpcName,
		},
		SubnetIID: irs.IID{
			NameId:   subnetName,
			SystemId: subnetName,
		},
		RootDiskType:   diskType,
		RootDiskSize:   strconv.FormatInt(diskInfo.SizeGb, 10),
		RootDeviceName: server.Disks[0].DeviceName,
		DataDiskIIDs:   attachedDisk.Items,
		KeyValueList: []irs.KeyValue{
			{"SubNetwork", server.NetworkInterfaces[0].Subnetwork},
			{"AccessConfigName", server.NetworkInterfaces[0].AccessConfigs[0].Name},
			{"NetworkTier", server.NetworkInterfaces[0].AccessConfigs[0].NetworkTier},
			{"DiskDeviceName", server.Disks[0].DeviceName},
			{"DiskName", server.Disks[0].Source},

			{"Kind", server.Kind},
			{"ZoneUrl", server.Zone},
		},
	}

	vmInfo.ImageType = vmHandler.getImageType(server.SourceMachineImage)

	arrVmSpec := strings.Split(server.MachineType, "/")
	cblogger.Info(arrVmSpec)
	if len(arrVmSpec) > 1 {
		cblogger.Info(arrVmSpec[len(arrVmSpec)-1])
		vmInfo.VMSpecName = arrVmSpec[len(arrVmSpec)-1]
	}

	//2020-05-13T00:15:37.183-07:00
	if len(server.CreationTimestamp) > 5 {
		cblogger.Infof("서버 구동 시간 처리 : [%s]", server.CreationTimestamp)
		t, err := time.Parse(time.RFC3339, server.CreationTimestamp)
		if err != nil {
			cblogger.Error(err)
		} else {
			cblogger.Infof("======> [%v]", t)
			vmInfo.StartTime = t
		}
	}

	return vmInfo
}

func (vmHandler *GCPVMHandler) getImageType(sourceMachineImage string) irs.ImageType {
	var imageType irs.ImageType

	if sourceMachineImage != "" {
		imageType = irs.MyImage
	} else {
		imageType = irs.PublicImage
	}

	return imageType
}

// 이미지 URL 방식 대신 이름을 사용하도록 변경 중
// @TODO : 2020-05-15 카푸치노 버전에서는 이름 대신 URL을 사용하기로 했음.
func (vmHandler *GCPVMHandler) getImageIID(server *compute.Instance) irs.IID {
	// projectID := vmHandler.Credential.ProjectID
	// zone := vmHandler.Region.Zone
	// dArr := strings.Split(diskname, "/")
	// var result string
	// if dArr != nil {
	// 	result = dArr[len(dArr)-1]
	// }
	// cblogger.Infof("result : [%s]", result)
	iId := irs.IID{}
	if server.SourceMachineImage != "" {
		iId.NameId = server.SourceMachineImage
		iId.SystemId = server.SourceMachineImage
	} else {
		info, err := vmHandler.getDiskInfo(server.Disks[0].Source)

		cblogger.Infof("********************************** Disk 정보 ****************")
		spew.Dump(info)
		if err != nil {
			cblogger.Error(err)
			return irs.IID{}
		}

		/* 2020-05-14 카푸치노 다음 버전에서 사용 예정
		arrImageUrl := strings.Split(info.SourceImage, "/")
		imageName := ""
		if len(arrImageUrl) > 0 {
			imageName = arrImageUrl[len(arrImageUrl)-1]
		}
		iId := irs.IID{
			NameId:   imageName,
			SystemId: imageName,
		}
		*/

		iId.NameId = info.SourceImage //2020-05-14 NameId는 사용자가 사용한 이름도 있기 때문에 리턴하지 않도록 수정
		iId.SystemId = info.SourceImage

	}
	return iId
}

// getVM에서 DiskSize, DiskType이 넘어오지 않아 Disk정보를 조회
func (vmHandler *GCPVMHandler) getDiskInfo(diskname string) (*compute.Disk, error) {
	dArr := strings.Split(diskname, "/")
	var result string
	if dArr != nil {
		result = dArr[len(dArr)-1]
	}
	cblogger.Infof("result : [%s]", result)

	info, err := GetDiskInfo(vmHandler.Client, vmHandler.Credential, vmHandler.Region, result)

	cblogger.Infof("********************************** Disk 정보 ****************")
	spew.Dump(info)
	if err != nil {
		cblogger.Error(err)
		return &compute.Disk{}, err
	}

	return info, nil
}

// func (vmHandler *GCPVMHandler) getKeyPairInfo(diskname string) irs.IID {
// 	projectID := vmHandler.Credential.ProjectID
// 	zone := vmHandler.Region.Zone
// 	var gcpKeyPairHandler *GCPKeyPairHandler
// 	iId := irs.IID{
// 		NameId:   "cb-user",
// 		SystemId: "cb-user",
// 	}
// 	result, err := gcpKeyPairHandler.GetKey(iId)

// 	spew.Dump(result)
// 	if err != nil {
// 		cblogger.Error(err)
// 		return result
// 	}

// 	return result
// }

// VM 정보를 조회할 수 있을 때까지 최대 30초간 대기
func (vmHandler *GCPVMHandler) WaitForRun(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Info("======> 생성된 VM의 최종 정보 확인을 위해 Running 될 때까지 대기함.")

	waitStatus := "Running"

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
		cblogger.Errorf("VM 상태가 [%s]이 아니라서 1초 대기후 조회합니다.", waitStatus)
		time.Sleep(time.Second * 1)
		if curRetryCnt > maxRetryCnt {
			cblogger.Errorf("장시간(%d 초) 대기해도 VM의 Status 값이 [%s]으로 변경되지 않아서 강제로 중단합니다.", maxRetryCnt, waitStatus)
			return irs.VMStatus("Failed"), errors.New("장시간 기다렸으나 생성된 VM의 상태가 [" + waitStatus + "]으로 바뀌지 않아서 중단 합니다.")
		}
	}

	return irs.VMStatus(waitStatus), nil
}
