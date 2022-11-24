// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//   - Cloud-Barista: https://github.com/cloud-barista
//
// EC2 Hander (AWS SDK GO Version 1.16.26, Thanks AWS.)
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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	cdcom "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsVMHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

func Connect(region string) *ec2.EC2 {
	// setup Region
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region)},
	)

	if err != nil {
		fmt.Println("Could not create instance", err)
		return nil
	}

	// Create EC2 service client
	svc := ec2.New(sess)

	return svc
}

// VM생성 시 사용할 루트 디스크의 최소 볼륨 사이즈 정보를 조회함
// -1 : 정보 조회 실패
func (vmHandler *AwsVMHandler) GetAmiDiskInfo(ImageSystemId string) (int64, error) {
	cblogger.Debugf("ImageID : [%s]", ImageSystemId)

	input := &ec2.DescribeImagesInput{
		ImageIds: []*string{
			aws.String(ImageSystemId),
		},
	}

	result, err := vmHandler.Client.DescribeImages(input)
	cblogger.Debug(result)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			cblogger.Error(err.Error())
		}
		return -1, err
	}

	if len(result.Images) > 0 {
		if !reflect.ValueOf(result.Images[0].BlockDeviceMappings).IsNil() {
			if !reflect.ValueOf(result.Images[0].BlockDeviceMappings[0].Ebs).IsNil() {
				isize := aws.Int64(*result.Images[0].BlockDeviceMappings[0].Ebs.VolumeSize)
				return *isize, nil
			} else {
				cblogger.Error("Ebs information not found in BlockDeviceMappings.")
				return -1, errors.New("Ebs information not found in BlockDeviceMappings.")
			}
		} else {
			cblogger.Error("BlockDeviceMappings information not found.")
			return -1, errors.New("BlockDeviceMappings information not found.")
		}
	} else {
		cblogger.Error("The requested Image[" + ImageSystemId + "]could not be found.")
		return -1, errors.New("The requested Image[" + ImageSystemId + "]could not be found.")
	}
}

// 1개의 VM만 생성되도록 수정 (MinCount / MaxCount 이용 안 함)
// 키페어 이름(예:mcloud-barista)은 아래 URL에 나오는 목록 중 "키페어 이름"의 값을 적으면 됨.
// https://ap-northeast-2.console.aws.amazon.com/ec2/v2/home?region=ap-northeast-2#KeyPairs:sort=keyName
func (vmHandler *AwsVMHandler) StartVM(vmReqInfo irs.VMReqInfo) (irs.VMInfo, error) {
	cblogger.Debug(vmReqInfo)
	if cblogger.Level.String() == "debug" {
		spew.Dump(vmReqInfo)
	}

	// amiImage, errImgInfo := DescribeImageById(imageHandler.Client, &vmReqInfo.ImageIID, nil)
	amiImage, errImgInfo := DescribeImageById(vmHandler.Client, &vmReqInfo.ImageIID, nil)
	//amiImage, errImgInfo := imageHandler.GetAmiImage(vmReqInfo.ImageIID)
	//imgInfo, errImgInfo := imageHandler.GetImage(vmReqInfo.ImageIID)
	if errImgInfo != nil {
		cblogger.Error(errImgInfo)
		return irs.VMInfo{}, errImgInfo
	}

	// public image일 때
	// 	 ImageOwnerAlias: "amazon"
	// 	 OwnerId: "801119661308"
	// MyImage일 때
	//	 ImageOwnerAlias property가 없음
	// 	 OwnerId는 자신의 Id(12자리)

	isMyImage := false
	abc := reflect.ValueOf(amiImage)
	imageOwnerAliasField := abc.Elem().FieldByName("ImageOwnerAlias")
	cblogger.Debugf("field: ", imageOwnerAliasField.IsValid())
	if !imageOwnerAliasField.IsValid() {
		cblogger.Debugf("ownerAlias: myimage ")
		isMyImage = true
	} else {
		cblogger.Debugf("ownerAlias: ", imageOwnerAliasField)
		cblogger.Debug("ownerAliasIsNil: ", imageOwnerAliasField.IsNil())
		if imageOwnerAliasField.IsNil() {
			isMyImage = true
		}
	}
	cblogger.Debugf("isMyImage: ", isMyImage)
	// cblogger.Debugf("abc: ", abc)
	// if abc != nil {
	// 	ownerAlias := amiImage.ImageOwnerAlias
	// 	if *ownerAlias == "amazon" {
	// 		cblogger.Debugf("ownerAlias: amazon ", *ownerAlias)
	// 	} else {
	// 		cblogger.Debugf("ownerAlias: myimage ", *ownerAlias)
	// 		isMyImage = true
	// 	}
	// }
	// if !reflect.ValueOf(&amiImage.ImageOwnerAlias).IsNil() {

	// }

	// cblogger.Debugf("OwnerId = ", *owner)
	//===============================
	// Root Disk Size 사전 검증 - 이슈#536
	//===============================
	if vmReqInfo.RootDiskSize != "" {
		//default로 전달 받은 경우 아무것도 하지 않음 (default는 스파이더 상위에서 다른 값으로 바뀌어서 전달 받기 때문에 로직은 필요 없음)
		if strings.EqualFold(vmReqInfo.RootDiskSize, "default") {
			//default로 전달 받은 경우 이미지의 볼륨 사이즈로 설정 함. - 2022-03-03 혼동을 피하기 위해 로직 제거 함.
			//vmReqInfo.RootDiskSize = strconv.FormatInt(imageVolumeSize, 10)
		} else {
			//이미지의 볼륨 사이즈를 조회함.
			//imageVolumeSize, err := vmHandler.GetAmiDiskInfo(vmReqInfo.ImageIID.SystemId)
			//if err != nil {
			//	cblogger.Error(err)
			//	return irs.VMInfo{}, err
			//}

			imageVolumeSize, err := GetImageSizeFromEc2Image(amiImage)
			if err != nil {
				return irs.VMInfo{}, err
			}

			// if len(result.Images) > 0 {
			// 	if !reflect.ValueOf(result.Images[0].BlockDeviceMappings).IsNil() {
			// 		if !reflect.ValueOf(result.Images[0].BlockDeviceMappings[0].Ebs).IsNil() {
			// 			isize := aws.Int64(*result.Images[0].BlockDeviceMappings[0].Ebs.VolumeSize)
			// 			return *isize, nil
			// 		} else {
			// 			cblogger.Error("BlockDeviceMappings에서 Ebs 정보를 찾을 수 없습니다.")
			// 			return -1, errors.New("BlockDeviceMappings에서 Ebs 정보를 찾을 수 없습니다.")
			// 		}
			// 	} else {
			// 		cblogger.Error("BlockDeviceMappings 정보를 찾을 수 없습니다.")
			// 		return -1, errors.New("BlockDeviceMappings 정보를 찾을 수 없습니다.")
			// 	}
			// } else {
			// 	cblogger.Error("요청된 Image 정보[" + ImageSystemId + "]를 찾을 수 없습니다.")
			// 	return -1, errors.New("요청된 Image 정보[" + ImageSystemId + "]를 찾을 수 없습니다.")
			// }

			if imageVolumeSize < 0 {
				return irs.VMInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "Unable to query the default volume size for the requested image.", nil)
			}

			//요청된 사이즈 체크
			iChkDiskSize, err := strconv.ParseInt(vmReqInfo.RootDiskSize, 10, 64)
			if err != nil {
				cblogger.Error(err)
				return irs.VMInfo{}, err
			}

			// 요청된 사이즈는 볼륨 사이즈 보다는 크거나 같아야 함.
			if iChkDiskSize < imageVolumeSize {
				cblogger.Errorf("루트볼륨은 %dGB보다 커야 합니다.", imageVolumeSize)
				return irs.VMInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "Root Disk Size must be at least the default size ("+strconv.FormatInt(imageVolumeSize, 10)+" GB).", nil)
			}
		}
	}

	imageID := vmReqInfo.ImageIID.SystemId
	instanceType := vmReqInfo.VMSpecName
	minCount := aws.Int64(1)
	maxCount := aws.Int64(1)
	keyName := vmReqInfo.KeyPairIID.SystemId
	baseName := vmReqInfo.IId.NameId
	subnetID := vmReqInfo.SubnetIID.SystemId

	/* 2021-10-26 이슈 #480에 의해 제거
	// 2021-04-28 cbuser 추가에 따른 Local KeyPair만 VM 생성 가능하도록 강제
	//=============================
	// KeyPair의 PublicKey 정보 처리
	//=============================
	cblogger.Infof("[%s] KeyPair 조회 시작", keyName)
	keypairHandler := AwsKeyPairHandler{
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
	// 보안그룹 처리 - SystemId 기반
	//=============================
	cblogger.Debug("SystemId 기반으로 처리하기 위해 IID 기반의 보안그룹 배열을 SystemId 기반 보안그룹 배열로 조회및 변환함.")
	var newSecurityGroupIds []string

	for _, sgName := range vmReqInfo.SecurityGroupIIDs {
		cblogger.Debugf("보안그룹 변환 : [%s]", sgName)
		newSecurityGroupIds = append(newSecurityGroupIds, sgName.SystemId)
	}

	cblogger.Debug("보안그룹 변환 완료")
	cblogger.Debug(newSecurityGroupIds)

	/* 2020-04-08 EIP 로직 제거
	//=============================
	// PublicIp 처리 - NameId 기반
	//=============================
	cblogger.Info("NameId 기반으로 처리하기 위해 PublicIp 정보를 조회함.")
	publicIPHandler := AwsPublicIPHandler{
		//Region: vmHandler.Region,
		Client: vmHandler.Client,
	}
	cblogger.Info(publicIPHandler)

	publicIPInfo, errPublicIPInfo := publicIPHandler.GetPublicIP(vmReqInfo.PublicIPId)
	cblogger.Info(publicIPInfo)
	if errPublicIPInfo != nil {
		cblogger.Error(errPublicIPInfo)
		return irs.VMInfo{}, errPublicIPInfo
	}
	publicIpId := publicIPInfo.Id
	cblogger.Infof("PublicIP ID를 [%s]대신 [%s]로 사용합니다.", publicIPInfo.Id, publicIpId)
	*/

	/*
		//=============================
		// UserData생성 처리
		//=============================
		userData := "#cloud-config\nusers:\n  - default\n  - name: " + CBDefaultVmUserName + "\n    groups: sudo\n    shell: /bin/bash\n    sudo: ['ALL=(ALL) NOPASSWD:ALL']\n    ssh-authorized-keys:\n      - "
		//userData = userData + "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC0wqohybvHvljVsUW7vmyicVNVDcPdzh6ZRkm1H9SyMuUEK0zOB3Kj+1MxMQPnRXgL9fI518ymUxavrkrHr0LwZtG8pfMOwZkZ7WD4WnT6Ho14N14U1JIM/+005cBBYyF+OWYyxD/q5p/y8R19NXLpEbnpTNL0mKjQ1q8a6/LVCsaKxy9OJ9o/ChN2FDXhCdVLPHL/jrUPqzjSLkm/GIt+v9RWJ0BFAk+rZY7abMNfGSorTqWZEYYd8gqofeTPh2mhYr21NVLBiAyzQqs6fgL+FgsnJFBnuIZ2peuCGxcOxZ7h8iEzJG2r+tGn+ivfMpla12oHxwihJhiodN1KxeZ7"
		userData = userData + keyPairInfo.PublicKey
		userDataBase64 := aws.String(base64.StdEncoding.EncodeToString([]byte(userData)))
		cblogger.Infof("===== userData ===")
		spew.Dump(userDataBase64)
	*/

	//=============================
	// UserData생성 처리(File기반)
	//=============================
	// 향후 공통 파일이나 외부에서 수정 가능하도록 cloud-init 스크립트 파일로 설정
	rootPath := os.Getenv("CBSPIDER_ROOT")
	initFilePath := rootPath + CBCloudInitFilePath // Linux용 Cloud Init Data 탬플릿
	userData := ""
	isWindowsImage := false

	guestOS := GetOsTypeFromEc2Image(amiImage)
	cblogger.Debugf("imgInfo.GuestOS : [%s]", guestOS)
	if strings.Contains(strings.ToUpper(guestOS), "WINDOWS") {

		err := cdcom.ValidateWindowsPassword(vmReqInfo.VMUserPasswd)
		if err != nil {
			return irs.VMInfo{}, err
		}
		isWindowsImage = true
		initFilePath = rootPath + CBCloudInitWindowsFilePath //windows용 Cloud-Init 탬플릿으로 변경
	} else {
		isWindowsImage = false
	}

	fileDataCloudInit, err := ioutil.ReadFile(initFilePath)
	if err != nil {
		cblogger.Error(err)
		return irs.VMInfo{}, err
	}

	//OS 종류에 따른 Cloud Init Data 처리
	if isWindowsImage {
		userData = strings.Replace(string(fileDataCloudInit), "*PASSWORD*", vmReqInfo.VMUserPasswd, 1)
		cblogger.Debugf("Windows용 Cloud-Init : [%s]", userData)
	} else {
		userData = string(fileDataCloudInit)
	}

	//userData = strings.ReplaceAll(userData, "{{username}}", CBDefaultVmUserName)
	//userData = strings.ReplaceAll(userData, "{{public_key}}", keyPairInfo.PublicKey)
	userDataBase64 := aws.String(base64.StdEncoding.EncodeToString([]byte(userData)))
	cblogger.Debugf("cloud-init data : [%s]", userDataBase64)

	/*
		if 1 == 1 {
			cblogger.Error("====윈도우즈 지원 테스트로 강제 종료함. ====")
			return irs.VMInfo{}, nil
		}
	*/

	//=============================
	// VM생성 처리
	//=============================
	cblogger.Debug("Create EC2 Instance")
	input := &ec2.RunInstancesInput{
		ImageId:      aws.String(imageID),
		InstanceType: aws.String(instanceType),
		MinCount:     minCount,
		MaxCount:     maxCount,
		KeyName:      aws.String(keyName),

		/*SecurityGroupIds: []*string{
			aws.String(securityGroupID), // "sg-0df1c209ea1915e4b" - 미지정시 보안 그룹명이 "default"인 보안 그룹이 사용 됨.
		},*/

		/* PrivateSubnet에도 PublicIp를 할당하려면 AssociatePublicIpAddress 옵션을 사용하거나 Subnet의 PublicIp 할당 옵션을 True로 생성해야 함.
		// 현재는 PublicIp 자동할딩 옵션이 False인 서브넷을 위해 NetworkInterfaces 필드에서 보안그룹과 서브넷을 정의 함. - 2020-04-19
		SecurityGroupIds: aws.StringSlice(newSecurityGroupIds),
		SubnetId:         aws.String(subnetID), // "subnet-cf9ccf83" - 미지정시 기본 VPC의 기본 서브넷이 임의로 이용되며 PublicIP가 할당 됨.
		*/

		//AdditionalInfo: aws.String("--associate-public-ip-address"),
		//AdditionalInfo: aws.String("AssociatePublicIpAddress=true"),
		//NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{{AssociatePublicIpAddress: aws.Bool(true)}},

		NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{ // PublicIp 할당을 위해 SubnetId와 보안 그룹을 이 곳에서 정의해야 함.
			{AssociatePublicIpAddress: aws.Bool(true),
				DeviceIndex: aws.Int64(0),
				Groups:      aws.StringSlice(newSecurityGroupIds),
				SubnetId:    aws.String(subnetID),
			},
		},

		//ec2.InstanceNetworkInterfaceSpecification
		UserData: userDataBase64,
	}

	//=============================
	// SystemDisk 처리 - 이슈 #348에 의해 RootDisk 기능 지원
	//=============================

	// deleteOnTermination := false
	//이슈#660 반영
	if strings.EqualFold(vmReqInfo.RootDiskType, "default") {
		vmReqInfo.RootDiskType = ""
		// if isMyImage {
		// 	// deleteOnTermination = true
		// 	blockDeviceMappings := []*ec2.BlockDeviceMapping{
		// 		{
		// 			Ebs: &ec2.EbsBlockDevice{
		// 				DeleteOnTermination: aws.Bool(deleteOnTermination),
		// 			},
		// 		},
		// 	}
		// 	input.SetBlockDeviceMappings(blockDeviceMappings)
		// 	cblogger.Debugf("MyImage set DeleteOnTermination = ", isMyImage, deleteOnTermination)
		// }
	}
	if vmReqInfo.RootDiskType != "" || vmReqInfo.RootDiskSize != "" {
		blockDeviceMappings := []*ec2.BlockDeviceMapping{
			{
				DeviceName: aws.String("/dev/sda1"),
				//DeviceName: aws.String("/dev/sdh"),
				Ebs: &ec2.EbsBlockDevice{
					//RootDeviceName
					//VolumeType: aws.String(diskType),
					//VolumeSize: diskSize,
					//DeleteOnTermination: aws.Bool(deleteOnTermination),
				},
			},
		}
		input.SetBlockDeviceMappings(blockDeviceMappings)
		//cblogger.Debugf("MyImage set DeleteOnTermination = ", isMyImage, deleteOnTermination)

		//=============================
		// Root Disk Type 변경
		//=============================
		if vmReqInfo.RootDiskType != "" {
			//diskType = vmReqInfo.RootDiskType
			input.BlockDeviceMappings[0].Ebs.VolumeType = aws.String(vmReqInfo.RootDiskType)
		}

		//=============================
		// Root Disk Size 변경
		//=============================
		if vmReqInfo.RootDiskSize != "" {
			if strings.EqualFold(vmReqInfo.RootDiskSize, "default") {
				input.BlockDeviceMappings[0].Ebs.VolumeSize = aws.Int64(8)
			} else {
				iDiskSize, err := strconv.ParseInt(vmReqInfo.RootDiskSize, 10, 64)
				if err != nil {
					cblogger.Error(err)
					return irs.VMInfo{}, err
				}
				//diskSize = aws.Int64(iDiskSize)
				//input.BlockDeviceMappings[0].Ebs.VolumeSize = diskSize
				input.BlockDeviceMappings[0].Ebs.VolumeSize = aws.Int64(iDiskSize)
			}
		}
	}

	cblogger.Debug(input)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmReqInfo.IId.NameId,
		CloudOSAPI:   "RunInstances()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	// Specify the details of the instance that you want to create.
	runResult, err := vmHandler.Client.RunInstances(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Debug(runResult)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Errorf("EC2 인스턴스 생성 실패 : ", err)
		return irs.VMInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	if len(runResult.Instances) < 1 {
		return irs.VMInfo{}, errors.New("No VM information received from AWS.")
	}

	//=============================
	// Name Tag 처리 - NameId 기반
	//=============================
	newVmId := *runResult.Instances[0].InstanceId
	cblogger.Infof("[%s] VM이 생성되었습니다.", newVmId)

	if baseName != "" {
		// Tag에 VM Name 설정
		_, errtag := vmHandler.Client.CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{runResult.Instances[0].InstanceId},
			Tags: []*ec2.Tag{
				{
					Key:   aws.String("Name"),
					Value: aws.String(baseName),
				},
			},
		})
		if errtag != nil {
			cblogger.Errorf("[%s] VM에 Name Tag 설정 실패", newVmId)
			cblogger.Error(errtag)
			//return irs.VMInfo{}, errtag
		}
	} else {
		cblogger.Error("vmReqInfo.IId.NameId가 전달되지 않아서 Name Tag를 설정하지않습니다.")
	}
	//Public IP및 최신 정보 전달을 위해 부팅이 완료될 때까지 대기했다가 전달하는 것으로 변경 함.
	//cblogger.Info("Public IP 할당 및 VM의 최신 정보 획득을 위해 EC2가 Running 상태가 될때까지 대기")

	//2021-05-11 EIP 할당 로직이 제거되었으며 빠른 생성을 위해 Running 상태가 될때까지 대기하지 않음.
	//2021-05-11 WaitForRun을 호출하지 않아도 GetVM() 호출 시 에러가 발생하지 않는 것은 확인했음. (우선은 정책이 최종 확정이 아니라서 WaitForRun을 사용하도록 원복함.)
	cblogger.Debug("VM의 최신 정보 획득을 위해 EC2가 Running 상태가 될때까지 대기")
	WaitForRun(vmHandler.Client, newVmId)
	cblogger.Debug("EC2 Running 상태 완료 : ", runResult.Instances[0].State.Name)

	/* 2020-04-08 EIP 로직 제거
	//EC2에 EIP 할당 (펜딩 상태에서는 EIP 할당 불가)
	cblogger.Infof("[%s] EC2에 [%s] IP 할당 시작", newVmId, publicIpId)
	assocRes, errIp := vmHandler.AssociatePublicIP(publicIpId, newVmId)
	if errIp != nil {
		cblogger.Errorf("EC2[%s]에 Public IP Id[%s]를 할당 할 수 없습니다 - %v", newVmId, publicIpId, err)
		return irs.VMInfo{}, errIp
	}

	cblogger.Infof("[%s] EC2에 Public IP 할당 결과 : ", newVmId, assocRes)
	*/

	/* 2020-04-08 vNic 로직 제거
	//
	//vNic 추가 요청이 있는 경우 전달 받은 vNic을 VM에 추가 함.
	//
	if vmReqInfo.NetworkInterfaceId != "" {
		_, errvNic := vmHandler.AttachNetworkInterface(vmReqInfo.NetworkInterfaceId, newVmId)
		if errvNic != nil {
			cblogger.Errorf("vNic [%s] 추가 실패!", vmReqInfo.NetworkInterfaceId)
			cblogger.Error(errvNic)
			return irs.VMInfo{}, errvNic
		} else {
			cblogger.Infof("vNic [%s] 추가 완료", vmReqInfo.NetworkInterfaceId)
		}
	}
	*/

	// attach disks : 직접추가하지 않고 이미 있는 volume 사용. 생성시점에 추가하지 못하고 생성 후 추가.
	availableVolumeNames := []string{"f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p"}
	if len(vmReqInfo.DataDiskIIDs) > len(availableVolumeNames) {
		return irs.VMInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "Too many Disks.", nil)
	}
	availableDeviceList, err := DescribeAvailableDiskDeviceList(vmHandler.Client, irs.IID{SystemId: newVmId})
	if err != nil {
		return irs.VMInfo{}, err
	}
	for diskIndex, dataDiskIID := range vmReqInfo.DataDiskIIDs {
		deviceName := availableDeviceList[diskIndex]

		//blockDeviceMapping := ec2.BlockDeviceMapping{
		//	DeviceName: aws.String(defaultVirtualizationType + availableVolumeNames[diskIndex]),
		//	//DeviceName: aws.String("/dev/sdh"),
		//	Ebs: &ec2.EbsBlockDevice{
		//		VolumeType: dataDiskInfo.VolumeType,
		//		VolumeSize: dataDiskInfo.Size,
		//	},
		//}
		//blockDeviceMappingList = append(blockDeviceMappingList, &blockDeviceMapping)

		err := AttachVolume(vmHandler.Client, deviceName, newVmId, dataDiskIID.SystemId)
		if err != nil {
			return irs.VMInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "The instance was created but disk attaching failed", err)
		}
	}

	//최신 정보 조회
	//newVmInfo, _ := vmHandler.GetVM(newVmId)
	newVmInfo, _ := vmHandler.GetVM(irs.IID{SystemId: newVmId})
	newVmInfo.IId.NameId = vmReqInfo.IId.NameId // Tag 정보가 없을 수 있기 때문에 요청 받은 NameId를 전달 함.

	/*
		//빠른 생성을 위해 Running 상태를 대기하지 않고 최소한의 정보만 리턴 함.
		//Running 상태를 대기 후 Public Ip 등의 정보를 추출하려면 GetVM()을 호출해서 최신 정보를 다시 받아와야 함.
		//vmInfo :=GetVM(runResult.Instances[0].InstanceId)

		//cblogger.Info("EC2 Running 상태 대기")
		//WaitForRun(vmHandler.Client, *runResult.Instances[0].InstanceId)
		//cblogger.Info("EC2 Running 상태 완료 : ", runResult.Instances[0].State.Name)

		vmInfo := ExtractDescribeInstances(runResult)
		//속도상 VM 정보를 다시 조회하지 않았기 때문에 Tag 정보가 누락되어서 Name 정보가 설정되어 있지 않음.
		if vmInfo.Name == "" {
			vmInfo.Name = baseName
		}
	*/

	return newVmInfo, nil
}

// VM이 Running 상태일때까지 대기 함.
func WaitForRun(svc *ec2.EC2, instanceID string) {
	cblogger.Infof("EC2 ID : [%s]", instanceID)

	input := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			aws.String(instanceID),
		},
	}
	err := svc.WaitUntilInstanceRunning(input)
	if err != nil {
		cblogger.Errorf("failed to wait until instances exist: %v", err)
	}
	cblogger.Info("=========WaitForRun() 종료")
}

// func (vmHandler *AwsVMHandler) ResumeVM(vmNameId string) (irs.VMStatus, error) {
func (vmHandler *AwsVMHandler) ResumeVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmNameId : [%s]", vmIID.SystemId)

	/*
		vmInfo, errVmInfo := vmHandler.GetVM(vmIID)
		if errVmInfo != nil {
			return irs.VMStatus("Failed"), errVmInfo
		}
		cblogger.Info(vmInfo)
		vmID := vmInfo.IId.SystemId
	*/
	vmID := vmIID.SystemId
	cblogger.Infof("vmID : [%s]", vmID)

	input := &ec2.StartInstancesInput{
		InstanceIds: []*string{
			aws.String(vmID),
		},
		DryRun: aws.Bool(true),
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmIID.SystemId,
		CloudOSAPI:   "StartInstances()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := vmHandler.Client.StartInstances(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	spew.Dump(result)
	awsErr, ok := err.(awserr.Error)

	if ok && awsErr.Code() == "DryRunOperation" {
		// Let's now set dry run to be false. This will allow us to start the instances
		input.DryRun = aws.Bool(false)
		result, err = vmHandler.Client.StartInstances(input)
		spew.Dump(result)
		if err != nil {
			//fmt.Println("Error", err)
			cblogger.Error(err)
			callLogInfo.ErrorMSG = err.Error()
			callogger.Info(call.String(callLogInfo))
			return irs.VMStatus("Failed"), err
		} else {
			//fmt.Println("Success", result.StartingInstances)
			cblogger.Info("Success", result.StartingInstances)
		}
	} else { // This could be due to a lack of permissions
		//fmt.Println("Error", err)
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		return irs.VMStatus("Failed"), err
	}
	callogger.Info(call.String(callLogInfo))

	return irs.VMStatus("Resuming"), nil
}

// func (vmHandler *AwsVMHandler) SuspendVM(vmNameId string) (irs.VMStatus, error) {
func (vmHandler *AwsVMHandler) SuspendVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmNameId : [%s]", vmIID.SystemId)

	/*
		vmInfo, errVmInfo := vmHandler.GetVM(vmIID)
		if errVmInfo != nil {
			return irs.VMStatus("Failed"), errVmInfo
		}
		cblogger.Info(vmInfo)
	*/
	vmID := vmIID.SystemId
	cblogger.Infof("vmID : [%s]", vmID)

	input := &ec2.StopInstancesInput{
		InstanceIds: []*string{
			aws.String(vmID),
		},
		DryRun: aws.Bool(true),
	}
	cblogger.Info(input)
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmIID.SystemId,
		CloudOSAPI:   "StopInstances()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	result, err := vmHandler.Client.StopInstances(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	spew.Dump(result)
	awsErr, ok := err.(awserr.Error)
	if ok && awsErr.Code() == "DryRunOperation" {
		input.DryRun = aws.Bool(false)
		result, err = vmHandler.Client.StopInstances(input)
		spew.Dump(result)
		if err != nil {
			callLogInfo.ErrorMSG = err.Error()
			callogger.Info(call.String(callLogInfo))
			cblogger.Error(err)
			return irs.VMStatus("Failed"), err
		} else {
			cblogger.Info("Success", result.StoppingInstances)
		}
	} else {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error("Error", err)
		return irs.VMStatus("Failed"), err
	}
	callogger.Info(call.String(callLogInfo))

	return irs.VMStatus("Suspending"), nil
}

// func (vmHandler *AwsVMHandler) RebootVM(vmNameId string) (irs.VMStatus, error) {
func (vmHandler *AwsVMHandler) RebootVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmNameId : [%s]", vmIID.NameId)

	/*
		vmInfo, errVmInfo := vmHandler.GetVM(vmIID)
		if errVmInfo != nil {
			return irs.VMStatus("Failed"), errVmInfo
		}
		cblogger.Info(vmInfo)
		vmID := vmInfo.IId.SystemId
	*/
	vmID := vmIID.SystemId
	cblogger.Infof("vmID : [%s]", vmID)

	input := &ec2.RebootInstancesInput{
		InstanceIds: []*string{
			aws.String(vmID),
		},
		DryRun: aws.Bool(true),
	}
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmIID.SystemId,
		CloudOSAPI:   "RebootInstances()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	result, err := vmHandler.Client.RebootInstances(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Info("result 값 : ", result)
	cblogger.Info("err 값 : ", err)

	awsErr, ok := err.(awserr.Error)
	cblogger.Info("ok 값 : ", ok)
	cblogger.Info("awsErr 값 : ", awsErr)
	if ok && awsErr.Code() == "DryRunOperation" {
		cblogger.Info("Reboot 권한 있음 - awsErr.Code() : ", awsErr.Code())

		//DryRun 권한 해제 후 리부팅을 요청 함.
		cblogger.Info("DryRun 권한 해제 후 리부팅을 요청 함.")
		input.DryRun = aws.Bool(false)
		result, err = vmHandler.Client.RebootInstances(input)
		spew.Dump(result)
		cblogger.Info("result 값 : ", result)
		cblogger.Info("err 값 : ", err)
		if err != nil {
			callLogInfo.ErrorMSG = err.Error()
			callogger.Info(call.String(callLogInfo))
			cblogger.Error("Error", err)
			return irs.VMStatus("Failed"), err
		} else {
			cblogger.Info("Success", result)
		}
	} else { // This could be due to a lack of permissions
		cblogger.Info("리부팅 권한이 없는 것같음.")
		cblogger.Error("Error", err)
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		return irs.VMStatus("Failed"), err
	}
	callogger.Info(call.String(callLogInfo))
	return irs.VMStatus("Rebooting"), nil
}

// func (vmHandler *AwsVMHandler) TerminateVM(vmNameId string) (irs.VMStatus, error) {
func (vmHandler *AwsVMHandler) TerminateVM(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmNameId : [%s]", vmIID.NameId)

	vmID := vmIID.SystemId
	cblogger.Infof("vmID : [%s]", vmID)

	input := &ec2.TerminateInstancesInput{
		//InstanceIds: instanceIds,
		InstanceIds: []*string{
			aws.String(vmID),
		},
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmIID.SystemId,
		CloudOSAPI:   "TerminateInstances()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := vmHandler.Client.TerminateInstances(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	spew.Dump(result)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error("Could not termiate instances", err)
		return irs.VMStatus("Failed"), err
	} else {
		cblogger.Info("Success")
	}
	callogger.Info(call.String(callLogInfo))
	return irs.VMStatus("Terminating"), nil
}

// https://docs.aws.amazon.com/ko_kr/AWSEC2/latest/APIReference/API_GetPasswordData.html
// https://awscli.amazonaws.com/v2/documentation/api/latest/reference/ec2/get-password-data.html
// @TODO : ssh key를 이용해서 암호가 해독된 Password를 조회해야 함.
func (vmHandler *AwsVMHandler) GetPasswordData(vmIID irs.IID) (string, error) {
	vmID := vmIID.SystemId
	cblogger.Infof("VM ID : [%s]", vmID)

	input := &ec2.GetPasswordDataInput{
		InstanceId: aws.String(vmID),
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmIID.SystemId,
		CloudOSAPI:   "GetPasswordData()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := vmHandler.Client.GetPasswordData(input)

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return "", err
	}
	callogger.Info(call.String(callLogInfo))

	return *result.PasswordData, nil
}

// 2019-11-16부로 CB-Driver 전체 로직이 NameId 기반으로 변경됨.
// func (vmHandler *AwsVMHandler) GetVM(vmNameId string) (irs.VMInfo, error) {
func (vmHandler *AwsVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {

	/* //Windows OS Password 조회 테스트 중
	passwordData, errPasswd := vmHandler.GetPasswordData(vmIID)
	if errPasswd != nil {
		cblogger.Error(errPasswd)
	}
	cblogger.Debugf("Password : [%s]", passwordData)

	if 1 == 1 {
		return irs.VMInfo{}, nil
	}
	*/

	resultInstance, err := DescribeInstanceById(vmHandler.Client, vmIID)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			cblogger.Error(err.Error())
		}
		return irs.VMInfo{}, err
	}

	vmInfo := irs.VMInfo{}
	vmInfo = vmHandler.ExtractDescribeInstanceToVmInfo(resultInstance)

	//if len(vmInfo.Region.Zone) > 0 {
	//vmInfo.Region.Region = vmHandler.Region.Region
	//}

	cblogger.Info("vmInfo", vmInfo)
	return vmInfo, nil
}

//func (vmHandler *AwsVMHandler) GetVM(vmIID irs.IID) (irs.VMInfo, error) {
//	cblogger.Infof("vmNameId : [%s]", vmIID.SystemId)
//	input := &ec2.DescribeInstancesInput{
//		InstanceIds: []*string{
//			aws.String(vmIID.SystemId),
//		},
//	}
//
//	cblogger.Info(input)
//
//	// logger for HisCall
//	callogger := call.GetLogger("HISCALL")
//	callLogInfo := call.CLOUDLOGSCHEMA{
//		CloudOS:      call.AWS,
//		RegionZone:   vmHandler.Region.Zone,
//		ResourceType: call.VM,
//		ResourceName: vmIID.SystemId,
//		CloudOSAPI:   "DescribeInstances()",
//		ElapsedTime:  "",
//		ErrorMSG:     "",
//	}
//	callLogStart := call.Start()
//
//	result, err := vmHandler.Client.DescribeInstances(input)
//	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
//	cblogger.Info(result)
//	if err != nil {
//		if aerr, ok := err.(awserr.Error); ok {
//			switch aerr.Code() {
//			default:
//				cblogger.Error(aerr.Error())
//			}
//		} else {
//			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
//			cblogger.Error(err.Error())
//		}
//		callLogInfo.ErrorMSG = err.Error()
//		callogger.Info(call.String(callLogInfo))
//		return irs.VMInfo{}, err
//	}
//	callogger.Info(call.String(callLogInfo))
//
//	//cblogger.Info(result)
//	cblogger.Infof("조회된 VM 정보 수 : [%d]", len(result.Reservations))
//	if len(result.Reservations) > 1 {
//		return irs.VMInfo{}, awserr.New("600", "1개 이상의 VM ["+vmIID.NameId+"] 정보가 존재합니다.", nil)
//	} else if len(result.Reservations) == 0 {
//		cblogger.Errorf("VM [%s] 정보가 존재하지 않습니다.", vmIID.NameId)
//		return irs.VMInfo{}, awserr.New("404", "VM ["+vmIID.NameId+"] 정보가 존재하지 않습니다.", nil)
//	}
//
//	vmInfo := irs.VMInfo{}
//	for _, i := range result.Reservations {
//		//vmInfo := ExtractDescribeInstances(result.Reservations[0])
//		vmInfo = vmHandler.ExtractDescribeInstances(i)
//	}
//
//	//if len(vmInfo.Region.Zone) > 0 {
//	//vmInfo.Region.Region = vmHandler.Region.Region
//	//}
//
//	cblogger.Info("vmInfo", vmInfo)
//	return vmInfo, nil
//}

func (vmHandler *AwsVMHandler) ExtractDescribeInstanceToVmInfo(instance *ec2.Instance) irs.VMInfo {
	//cblogger.Info("ExtractDescribeInstances", reservation)
	cblogger.Info("Instance", instance)
	//spew.Dump(reservation.Instances[0])

	//"stopped" / "terminated" / "running" ...
	var state string
	state = *instance.State.Name
	cblogger.Infof("EC2 상태 : [%s]", state)

	//VM상태와 무관하게 항상 값이 존재하는 항목들만 초기화
	vmInfo := irs.VMInfo{
		IId:        irs.IID{"", *instance.InstanceId},
		ImageIId:   irs.IID{*instance.ImageId, *instance.ImageId},
		VMSpecName: *instance.InstanceType,
		//KeyPairIId: irs.IID{*reservation.Instances[0].KeyName, *reservation.Instances[0].KeyName},	//AWS에 키페어 없이 VM 생성하는 기능이 추가됨.
		//GuestUserID:    "",
		//AdditionalInfo: "State:" + *reservation.Instances[0].State.Name,
	}

	keyValueList := []irs.KeyValue{
		{Key: "State", Value: *instance.State.Name},
		{Key: "Architecture", Value: *instance.Architecture},
	}

	//if *reservation.Instances[0].LaunchTime != "" {
	vmInfo.StartTime = *instance.LaunchTime
	//}

	//cblogger.Info("=======>타입 : ", reflect.TypeOf(*reservation.Instances[0]))
	//cblogger.Info("===> PublicIpAddress TypeOf : ", reflect.TypeOf(reservation.Instances[0].PublicIpAddress))
	//cblogger.Info("===> PublicIpAddress ValueOf : ", reflect.ValueOf(reservation.Instances[0].PublicIpAddress))

	//vmInfo.PublicIP = *reservation.Instances[0].NetworkInterfaces[0].Association.PublicIp
	//vmInfo.PublicDNS = *reservation.Instances[0].NetworkInterfaces[0].Association.PublicDnsName

	//AWS에 키페어 없이 VM 생성하는 기능이 추가됨. - 이슈#573
	if !reflect.ValueOf(instance.KeyName).IsNil() {
		vmInfo.KeyPairIId = irs.IID{*instance.KeyName, *instance.KeyName}
	}

	// 특정 항목(예:EIP)은 VM 상태와 무관하게 동작하므로 VM 상태와 무관하게 Nil처리로 모든 필드를 처리 함.
	if !reflect.ValueOf(instance.PublicIpAddress).IsNil() {
		vmInfo.PublicIP = *instance.PublicIpAddress
	}

	if !reflect.ValueOf(instance.PublicDnsName).IsNil() {
		vmInfo.PublicDNS = *instance.PublicDnsName
	}

	cblogger.Info("===> BlockDeviceMappings ValueOf : ", reflect.ValueOf(instance.BlockDeviceMappings))
	if !reflect.ValueOf(instance.BlockDeviceMappings).IsNil() {
		//if !reflect.ValueOf(instance.BlockDeviceMappings[0].DeviceName).IsNil() {
		//	vmInfo.VMBlockDisk = *instance.BlockDeviceMappings[0].DeviceName
		//}
		//
		//if !reflect.ValueOf(instance.BlockDeviceMappings[0].Ebs).IsNil() {
		//	volumeInfo, err := DescribeVolumneById(vmHandler.Client, *instance.BlockDeviceMappings[0].Ebs.VolumeId)
		//	//volumeInfo, err := vmHandler.GetVolumInfo(*instance.BlockDeviceMappings[0].Ebs.VolumeId)
		//	if err != nil {
		//	} else {
		//		vmInfo.RootDiskSize = strconv.FormatInt(*volumeInfo.Size, 10)
		//		vmInfo.RootDiskType = *volumeInfo.VolumeType
		//	}
		//}

		// attached 된 disk. instance의 0번째는 rootDisk
		diskDeviceList := instance.BlockDeviceMappings
		if diskDeviceList != nil {
			dataDiskIIDList := []irs.IID{}
			for diskIndex, diskDevice := range diskDeviceList {
				diskName := diskDevice.DeviceName
				volumeId := diskDevice.Ebs.VolumeId
				if diskIndex == 0 {
					vmInfo.VMBlockDisk = *diskName
					volumeInfo, err := DescribeVolumneById(vmHandler.Client, *volumeId)
					if err != nil {
					} else {
						vmInfo.RootDiskSize = strconv.FormatInt(*volumeInfo.Size, 10)
						vmInfo.RootDiskType = *volumeInfo.VolumeType
					}
				} else {
					dataDiskIIDList = append(dataDiskIIDList, irs.IID{SystemId: *volumeId})
				}
			}
			vmInfo.DataDiskIIDs = dataDiskIIDList
		}
	}

	// TODO : Image 분류 처리 추가할 것
	awsImageInfo, err := DescribeImageById(vmHandler.Client, &irs.IID{SystemId: *instance.ImageId}, nil)
	if err != nil {
		// fail to get ImageInfo
		//awsImageInfo.Public
		//awsImageInfo.OwnerId //
		//awsImageInfo.ImageOwnerAlias
	}
	spew.Dump(awsImageInfo) //ImageId: "ami-00f1068284b9eca92",

	// instance.ImageId
	// describeImage -> is-public

	if !reflect.ValueOf(instance.Placement.AvailabilityZone).IsNil() {
		vmInfo.Region = irs.RegionInfo{
			Region: vmHandler.Region.Region, //리전 정보 추가
			Zone:   *instance.Placement.AvailabilityZone,
		}
	}

	//NetworkInterfaces 배열 값들
	if !reflect.ValueOf(instance.NetworkInterfaces).IsNil() {
		if !reflect.ValueOf(instance.NetworkInterfaces[0].VpcId).IsNil() {
			//vmInfo.VirtualNetworkId = *reservation.Instances[0].NetworkInterfaces[0].VpcId
			vmInfo.VpcIID = irs.IID{"", *instance.NetworkInterfaces[0].VpcId}
			keyValueList = append(keyValueList, irs.KeyValue{Key: "VpcId", Value: *instance.NetworkInterfaces[0].VpcId})
		}

		if !reflect.ValueOf(instance.NetworkInterfaces[0].SubnetId).IsNil() {
			keyValueList = append(keyValueList, irs.KeyValue{Key: "SubnetId", Value: *instance.NetworkInterfaces[0].SubnetId})
			vmInfo.SubnetIID = irs.IID{SystemId: *instance.NetworkInterfaces[0].SubnetId}
		}

		if !reflect.ValueOf(instance.NetworkInterfaces[0].Attachment).IsNil() {
			vmInfo.NetworkInterface = *instance.NetworkInterfaces[0].Attachment.AttachmentId
		}

		for _, security := range instance.NetworkInterfaces[0].Groups {
			//vmInfo.SecurityGroupIds = append(vmInfo.SecurityGroupIds, *security.GroupId)
			vmInfo.SecurityGroupIIds = append(vmInfo.SecurityGroupIIds, irs.IID{*security.GroupName, *security.GroupId})
		}

	}

	//SecurityName: *reservation.Instances[0].NetworkInterfaces[0].Groups[0].GroupName,
	//vmInfo.VNIC = "eth0 - 값 위치 확인 필요"

	//vmInfo.PrivateIP = *reservation.Instances[0].NetworkInterfaces[0].PrivateIpAddress	//없는 경우 존재해서 Instances[0].PrivateIpAddress로 대체 - i-0b75cac73c4575386
	if !reflect.ValueOf(instance.PrivateIpAddress).IsNil() {
		vmInfo.PrivateIP = *instance.PrivateIpAddress
	}

	//vmInfo.PrivateDNS = *reservation.Instances[0].NetworkInterfaces[0].PrivateDnsName		//없는 경우 존재해서 Instances[0].PrivateDnsName로 대체 - i-0b75cac73c4575386
	if !reflect.ValueOf(instance.PrivateDnsName).IsNil() {
		vmInfo.PrivateDNS = *instance.PrivateDnsName
	}

	if !reflect.ValueOf(instance.RootDeviceName).IsNil() {
		//vmInfo.VMBootDisk = *reservation.Instances[0].RootDeviceName
		vmInfo.RootDeviceName = *instance.RootDeviceName
	}

	/*
		if !reflect.ValueOf(reservation.Instances[0].RootDeviceType).IsNil() {
			//vmInfo.VMBootDisk = *reservation.Instances[0].RootDeviceName
			vmInfo.RootDiskType = *reservation.Instances[0].RootDeviceType
		}
	*/

	if !reflect.ValueOf(instance.KeyName).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "KeyName", Value: *instance.KeyName})
	}

	if !reflect.ValueOf(instance.Platform).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "Platform", Value: *instance.Platform})
	}
	if !reflect.ValueOf(instance.VirtualizationType).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "VirtualizationType", Value: *instance.VirtualizationType})
	}

	//Name은 Tag의 "Name" 속성에만 저장됨
	cblogger.Debug("Name Tag 찾기")
	for _, t := range instance.Tags {
		if *t.Key == "Name" {
			vmInfo.IId.NameId = *t.Value
			cblogger.Debug("EC2 명칭 : ", vmInfo.IId.NameId)
			break
		}
	}

	vmInfo.KeyValueList = keyValueList
	return vmInfo
}

// DescribeInstances결과에서 EC2 세부 정보 추출
// VM 생성 시에는 Running 이전 상태의 정보가 넘어오기 때문에
// 최종 정보 기반으로 리턴 받고 싶으면 GetVM에 통합해야 할 듯.
func (vmHandler *AwsVMHandler) ExtractDescribeInstances(reservation *ec2.Reservation) irs.VMInfo {
	//cblogger.Info("ExtractDescribeInstances", reservation)
	cblogger.Info("Instances[0]", reservation.Instances[0])
	//spew.Dump(reservation.Instances[0])

	//"stopped" / "terminated" / "running" ...
	var state string
	state = *reservation.Instances[0].State.Name
	cblogger.Infof("EC2 상태 : [%s]", state)

	//VM상태와 무관하게 항상 값이 존재하는 항목들만 초기화
	vmInfo := irs.VMInfo{
		IId:        irs.IID{"", *reservation.Instances[0].InstanceId},
		ImageIId:   irs.IID{*reservation.Instances[0].ImageId, *reservation.Instances[0].ImageId},
		VMSpecName: *reservation.Instances[0].InstanceType,
		//KeyPairIId: irs.IID{*reservation.Instances[0].KeyName, *reservation.Instances[0].KeyName},	//AWS에 키페어 없이 VM 생성하는 기능이 추가됨.
		//GuestUserID:    "",
		//AdditionalInfo: "State:" + *reservation.Instances[0].State.Name,
	}

	keyValueList := []irs.KeyValue{
		{Key: "State", Value: *reservation.Instances[0].State.Name},
		{Key: "Architecture", Value: *reservation.Instances[0].Architecture},
	}

	//if *reservation.Instances[0].LaunchTime != "" {
	vmInfo.StartTime = *reservation.Instances[0].LaunchTime
	//}

	//cblogger.Info("=======>타입 : ", reflect.TypeOf(*reservation.Instances[0]))
	//cblogger.Info("===> PublicIpAddress TypeOf : ", reflect.TypeOf(reservation.Instances[0].PublicIpAddress))
	//cblogger.Info("===> PublicIpAddress ValueOf : ", reflect.ValueOf(reservation.Instances[0].PublicIpAddress))

	//vmInfo.PublicIP = *reservation.Instances[0].NetworkInterfaces[0].Association.PublicIp
	//vmInfo.PublicDNS = *reservation.Instances[0].NetworkInterfaces[0].Association.PublicDnsName

	//AWS에 키페어 없이 VM 생성하는 기능이 추가됨. - 이슈#573
	if !reflect.ValueOf(reservation.Instances[0].KeyName).IsNil() {
		vmInfo.KeyPairIId = irs.IID{*reservation.Instances[0].KeyName, *reservation.Instances[0].KeyName}
	}

	// 특정 항목(예:EIP)은 VM 상태와 무관하게 동작하므로 VM 상태와 무관하게 Nil처리로 모든 필드를 처리 함.
	if !reflect.ValueOf(reservation.Instances[0].PublicIpAddress).IsNil() {
		vmInfo.PublicIP = *reservation.Instances[0].PublicIpAddress
	}

	if !reflect.ValueOf(reservation.Instances[0].PublicDnsName).IsNil() {
		vmInfo.PublicDNS = *reservation.Instances[0].PublicDnsName
	}

	cblogger.Info("===> BlockDeviceMappings ValueOf : ", reflect.ValueOf(reservation.Instances[0].BlockDeviceMappings))
	if !reflect.ValueOf(reservation.Instances[0].BlockDeviceMappings).IsNil() {
		if !reflect.ValueOf(reservation.Instances[0].BlockDeviceMappings[0].DeviceName).IsNil() {
			vmInfo.VMBlockDisk = *reservation.Instances[0].BlockDeviceMappings[0].DeviceName
		}

		if !reflect.ValueOf(reservation.Instances[0].BlockDeviceMappings[0].Ebs).IsNil() {

			volumeInfo, err := vmHandler.GetVolumInfo(*reservation.Instances[0].BlockDeviceMappings[0].Ebs.VolumeId)
			if err != nil {
			} else {
				vmInfo.RootDiskSize = strconv.FormatInt(*volumeInfo.Size, 10)
				vmInfo.RootDiskType = *volumeInfo.VolumeType
			}
		}

	}

	if !reflect.ValueOf(reservation.Instances[0].Placement.AvailabilityZone).IsNil() {
		vmInfo.Region = irs.RegionInfo{
			Region: vmHandler.Region.Region, //리전 정보 추가
			Zone:   *reservation.Instances[0].Placement.AvailabilityZone,
		}
	}

	//NetworkInterfaces 배열 값들
	if !reflect.ValueOf(reservation.Instances[0].NetworkInterfaces).IsNil() {
		if !reflect.ValueOf(reservation.Instances[0].NetworkInterfaces[0].VpcId).IsNil() {
			//vmInfo.VirtualNetworkId = *reservation.Instances[0].NetworkInterfaces[0].VpcId
			vmInfo.VpcIID = irs.IID{"", *reservation.Instances[0].NetworkInterfaces[0].VpcId}
			keyValueList = append(keyValueList, irs.KeyValue{Key: "VpcId", Value: *reservation.Instances[0].NetworkInterfaces[0].VpcId})
		}

		if !reflect.ValueOf(reservation.Instances[0].NetworkInterfaces[0].SubnetId).IsNil() {
			keyValueList = append(keyValueList, irs.KeyValue{Key: "SubnetId", Value: *reservation.Instances[0].NetworkInterfaces[0].SubnetId})
			vmInfo.SubnetIID = irs.IID{SystemId: *reservation.Instances[0].NetworkInterfaces[0].SubnetId}
		}

		if !reflect.ValueOf(reservation.Instances[0].NetworkInterfaces[0].Attachment).IsNil() {
			vmInfo.NetworkInterface = *reservation.Instances[0].NetworkInterfaces[0].Attachment.AttachmentId
		}

		for _, security := range reservation.Instances[0].NetworkInterfaces[0].Groups {
			//vmInfo.SecurityGroupIds = append(vmInfo.SecurityGroupIds, *security.GroupId)
			vmInfo.SecurityGroupIIds = append(vmInfo.SecurityGroupIIds, irs.IID{*security.GroupName, *security.GroupId})
		}

		/*
			if !reflect.ValueOf(reservation.Instances[0].NetworkInterfaces[0].Groups).IsNil() {
				vmInfo.SecurityGroupIds = *reservation.Instances[0].NetworkInterfaces[0].Groups[0]
				if !reflect.ValueOf(reservation.Instances[0].NetworkInterfaces[0].Groups[0].GroupId).IsNil() {
					vmInfo.SecurityID = *reservation.Instances[0].NetworkInterfaces[0].Groups[0].GroupId
				}
			}
		*/
	}

	//SecurityName: *reservation.Instances[0].NetworkInterfaces[0].Groups[0].GroupName,
	//vmInfo.VNIC = "eth0 - 값 위치 확인 필요"

	//vmInfo.PrivateIP = *reservation.Instances[0].NetworkInterfaces[0].PrivateIpAddress	//없는 경우 존재해서 Instances[0].PrivateIpAddress로 대체 - i-0b75cac73c4575386
	if !reflect.ValueOf(reservation.Instances[0].PrivateIpAddress).IsNil() {
		vmInfo.PrivateIP = *reservation.Instances[0].PrivateIpAddress
	}

	//vmInfo.PrivateDNS = *reservation.Instances[0].NetworkInterfaces[0].PrivateDnsName		//없는 경우 존재해서 Instances[0].PrivateDnsName로 대체 - i-0b75cac73c4575386
	if !reflect.ValueOf(reservation.Instances[0].PrivateDnsName).IsNil() {
		vmInfo.PrivateDNS = *reservation.Instances[0].PrivateDnsName
	}

	if !reflect.ValueOf(reservation.Instances[0].RootDeviceName).IsNil() {
		//vmInfo.VMBootDisk = *reservation.Instances[0].RootDeviceName
		vmInfo.RootDeviceName = *reservation.Instances[0].RootDeviceName
	}

	/*
		if !reflect.ValueOf(reservation.Instances[0].RootDeviceType).IsNil() {
			//vmInfo.VMBootDisk = *reservation.Instances[0].RootDeviceName
			vmInfo.RootDiskType = *reservation.Instances[0].RootDeviceType
		}
	*/

	if !reflect.ValueOf(reservation.Instances[0].KeyName).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "KeyName", Value: *reservation.Instances[0].KeyName})
	}

	if !reflect.ValueOf(reservation.Instances[0].Platform).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "Platform", Value: *reservation.Instances[0].Platform})
	}
	if !reflect.ValueOf(reservation.Instances[0].VirtualizationType).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "VirtualizationType", Value: *reservation.Instances[0].VirtualizationType})
	}

	//Name은 Tag의 "Name" 속성에만 저장됨
	cblogger.Debug("Name Tag 찾기")
	for _, t := range reservation.Instances[0].Tags {
		if *t.Key == "Name" {
			vmInfo.IId.NameId = *t.Value
			cblogger.Debug("EC2 명칭 : ", vmInfo.IId.NameId)
			break
		}
	}

	vmInfo.KeyValueList = keyValueList
	return vmInfo
}

// 볼륨 정보 조회
func (vmHandler *AwsVMHandler) GetVolumInfo(volumeId string) (*ec2.Volume, error) {
	cblogger.Infof("volumeId : [%s]", volumeId)

	input := &ec2.DescribeVolumesInput{
		VolumeIds: []*string{
			aws.String(volumeId),
		},
	}
	/*
		input.Filters = ([]*ec2.Filter{
			&ec2.Filter{
				Name: aws.String("tag:volume-id"),
				Values: []*string{
					aws.String(volumeId),
				},
			},
		})
	*/
	cblogger.Info(input)

	result, err := vmHandler.Client.DescribeVolumes(input)
	cblogger.Info(result)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			cblogger.Error(err.Error())
		}
		return nil, err
	}

	//cblogger.Info(result)
	cblogger.Infof("조회된 볼륨 수 : [%d]", len(result.Volumes))
	if len(result.Volumes) > 1 {
		return nil, awserr.New("700", "One or more volumes exist.", nil)
	} else if len(result.Volumes) == 0 {
		cblogger.Errorf("[%s]와 일치하는 볼륨 정보가 존재하지 않습니다.", volumeId)
		return nil, awserr.New("404", "["+volumeId+"] Volume Not Found", nil)
	}

	cblogger.Info("VolumeInfo", result.Volumes)
	return result.Volumes[0], nil
}

func ExtractVmName(Tags []*ec2.Tag) string {
	for _, t := range Tags {
		if *t.Key == "Name" {
			cblogger.Info("  --> EC2 명칭 : ", *t.Key)
			return *t.Value
		}
	}
	return ""
}

func (vmHandler *AwsVMHandler) ListVM() ([]*irs.VMInfo, error) {
	cblogger.Infof("Start")
	var vmInfoList []*irs.VMInfo

	input := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			nil,
		},
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: "",
		CloudOSAPI:   "ListVM()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := vmHandler.Client.DescribeInstances(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			cblogger.Error(err.Error())
		}
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		return nil, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Info("Success")

	//tmpVmName := ""
	for _, i := range result.Reservations {
		for _, vm := range i.Instances {
			cblogger.Info("[%s] EC2 정보 조회", *vm.InstanceId)
			/*
				tmpVmName = ExtractVmName(vm.Tags)
				if tmpVmName == "" {
					cblogger.Errorf("VM Id[%s]에 해당하는 VM 이름을 찾을 수 없습니다!!!", *vm.InstanceId)
					continue
				}
			*/
			//vmInfo, _ := vmHandler.GetVM(irs.IID{NameId: tmpVmName})
			vmInfo, _ := vmHandler.GetVM(irs.IID{SystemId: *vm.InstanceId})
			vmInfoList = append(vmInfoList, &vmInfo)
		}
	}

	return vmInfoList, nil
}

func ConvertVMStatusString(vmStatus string) (irs.VMStatus, error) {
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
		return irs.VMStatus("Failed"), errors.New("Cannot find status information that matches " + vmStatus)
	}
	cblogger.Infof("VM 상태 치환 : [%s] ==> [%s]", vmStatus, resultStatus)
	return irs.VMStatus(resultStatus), nil
}

// SHUTTING-DOWN / TERMINATED
// func (vmHandler *AwsVMHandler) GetVMStatus(vmNameId string) (irs.VMStatus, error) {
func (vmHandler *AwsVMHandler) GetVMStatus(vmIID irs.IID) (irs.VMStatus, error) {
	cblogger.Infof("vmNameId : [%s]", vmIID.SystemId)

	/*
		vmInfo, errVmInfo := vmHandler.GetVM(vmIID)
		if errVmInfo != nil {
			return irs.VMStatus("Failed"), errVmInfo
		}
		cblogger.Info(vmInfo)
		vmID := vmInfo.IId.SystemId
	*/

	vmID := vmIID.SystemId
	cblogger.Infof("vmID : [%s]", vmID)

	//vmStatus := "pending"
	//return irs.VMStatus(vmStatus)

	input := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			aws.String(vmID),
		},
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: vmIID.SystemId,
		CloudOSAPI:   "DescribeInstances()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := vmHandler.Client.DescribeInstances(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			cblogger.Error(err.Error())
		}
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		return irs.VMStatus("Failed"), err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Info("Success", result)
	for _, i := range result.Reservations {
		for _, vm := range i.Instances {
			//vmStatus := strings.ToUpper(*vm.State.Name)
			cblogger.Info(vmID, " EC2 Status : ", *vm.State.Name)
			vmStatus, errStatus := ConvertVMStatusString(*vm.State.Name)
			return vmStatus, errStatus
			//return irs.VMStatus(vmStatus), nil
		}
	}

	return irs.VMStatus("Failed"), errors.New("Status information not found.")
}

func (vmHandler *AwsVMHandler) ListVMStatus() ([]*irs.VMStatusInfo, error) {
	cblogger.Infof("Start")
	var vmStatusList []*irs.VMStatusInfo

	input := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			nil,
		},
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   vmHandler.Region.Zone,
		ResourceType: call.VM,
		ResourceName: "ListVMStatus()",
		CloudOSAPI:   "DescribeInstances()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	result, err := vmHandler.Client.DescribeInstances(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			cblogger.Error(err.Error())
		}
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		return nil, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Info("Success")

	tmpVmName := ""
	for _, i := range result.Reservations {
		for _, vm := range i.Instances {
			//*vm.State.Name
			//*vm.InstanceId

			vmStatus, _ := ConvertVMStatusString(*vm.State.Name)
			tmpVmName = ExtractVmName(vm.Tags)
			/*
				if tmpVmName == "" {
					cblogger.Errorf("VM Id[%s]에 해당하는 VM 이름을 찾을 수 없습니다!!!", *vm.InstanceId)
					//continue //2020-04-10 Name이 필수는 아니기 때문에 예외에서 제외 함.
				}
			*/

			vmStatusInfo := irs.VMStatusInfo{
				//VmId:   *vm.InstanceId,
				//VmName: tmpVmName,
				IId: irs.IID{tmpVmName, *vm.InstanceId},
				//VmStatus: vmHandler.GetVMStatus(*vm.InstanceId),
				//VmStatus: irs.VMStatus(strings.ToUpper(*vm.State.Name)),
				VmStatus: vmStatus,
			}
			cblogger.Info(vmStatusInfo.IId.SystemId, " EC2 Status : ", vmStatusInfo.VmStatus)
			vmStatusList = append(vmStatusList, &vmStatusInfo)
		}
	}

	return vmStatusList, nil
}

// AssociationId 대신 PublicIP로도 가능 함.
func (vmHandler *AwsVMHandler) AssociatePublicIP(allocationId string, instanceId string) (bool, error) {
	cblogger.Infof("EC2에 퍼블릭 IP할당 - AllocationId : [%s], InstanceId : [%s]", allocationId, instanceId)

	// EC2에 할당.
	// Associate the new Elastic IP address with an existing EC2 instance.
	assocRes, err := vmHandler.Client.AssociateAddress(&ec2.AssociateAddressInput{
		AllocationId: aws.String(allocationId),
		InstanceId:   aws.String(instanceId),
	})

	spew.Dump(assocRes)
	//cblogger.Infof("[%s] EC2에 EIP(AllocationId : [%s]) 할당 완료 - AssociationId Id : [%s]", instanceId, allocationId, *assocRes.AssociationId)

	if err != nil {
		cblogger.Errorf("Unable to associate IP address with %s, %v", instanceId, err)
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Errorf(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Errorf(err.Error())
		}
		return false, err
	}

	cblogger.Info(assocRes)
	return true, nil
}

// 전달 받은 vNic을 VM에 추가함.
func (vmHandler *AwsVMHandler) AttachNetworkInterface(vNicId string, instanceId string) (bool, error) {
	cblogger.Infof("EC2[%s] VM에 vNic[%s] 추가 시작", vNicId, instanceId)

	input := &ec2.AttachNetworkInterfaceInput{
		DeviceIndex:        aws.Int64(1),
		InstanceId:         aws.String(instanceId),
		NetworkInterfaceId: aws.String(vNicId),
	}

	result, err := vmHandler.Client.AttachNetworkInterface(input)
	cblogger.Info(result)

	if err != nil {
		cblogger.Errorf("EC2[%s] VM에 vNic[%s] 추가 실패", vNicId, instanceId)
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Errorf(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			cblogger.Errorf(err.Error())
		}
		return false, err
	}

	return true, nil
}
