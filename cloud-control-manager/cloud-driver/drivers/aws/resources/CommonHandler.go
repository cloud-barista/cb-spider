package resources

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/davecgh/go-spew/spew"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

//// AWS API 1:1로 대응

// ---------------- Instance Area begin ---------------//
/*
	Instance 정보조회.
	기본은 목록 조회이며 filter조건이 있으면 해당 filter 조건으로 검색하도록
*/
func DescribeInstances(svc *ec2.EC2, vmIIDs []irs.IID) (*ec2.DescribeInstancesOutput, error) {
	input := &ec2.DescribeInstancesInput{}
	var instanceIds []*string

	if vmIIDs != nil {
		for _, vmIID := range vmIIDs {
			instanceIds = append(instanceIds, aws.String(vmIID.SystemId))
		}
		input.InstanceIds = instanceIds
	}

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		ResourceType: call.VM,
		CloudOSAPI:   "DescribeInstances",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()

	result, err := svc.DescribeInstances(input)

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))

	return result, err
}

/*
1개 인스턴스의 정보 조회
*/
func DescribeInstanceById(svc *ec2.EC2, vmIID irs.IID) (*ec2.Instance, error) {
	var vmIIDs []irs.IID
	var iid irs.IID

	if vmIID == iid {
		return nil, errors.New("instanceID is empty.")
	}

	vmIIDs = append(vmIIDs, vmIID)

	result, err := DescribeInstances(svc, vmIIDs)
	if err != nil {
		return nil, err
	}

	if len(result.Reservations) < 1 || len(result.Reservations[0].Instances) < 1 {

		return nil, errors.New(vmIID.SystemId + " instance not found.")

	}
	instance := result.Reservations[0].Instances[0]
	return instance, err
}

/*
1개 인스턴스의 상태조회
*/
func DescribeInstanceStatus(svc *ec2.EC2, vmIID irs.IID) (string, error) {

	instance, err := DescribeInstanceById(svc, vmIID)
	if err != nil {
		return "", err
	}
	//type InstanceState struct {
	//	_    struct{} `type:"structure"`
	//	Code *int64   `locationName:"code" type:"integer"`
	//	Name *string  `locationName:"name" type:"string" enum:"InstanceStateName"`
	//}
	status := instance.State.Name

	return *status, err
}

/*
1개 인스턴스에서 사용중인 disk 와 device 목록
*/
func DescribeInstanceDiskDeviceList(svc *ec2.EC2, vmIID irs.IID) ([]*ec2.InstanceBlockDeviceMapping, error) {

	instance, err := DescribeInstanceById(svc, vmIID)
	if err != nil {
		return nil, err
	}

	//device := instance.BlockDeviceMappings[0].DeviceName
	//blockDevice := instance.BlockDeviceMappings[0].Ebs
	return instance.BlockDeviceMappings, err
}

/*
1개 인스턴스에서 사용가능한 device 이름 목록
존재하는 device 이름 제거 후 가능한 목록만 return
*/
func DescribeAvailableDiskDeviceList(svc *ec2.EC2, vmIID irs.IID) ([]string, error) {
	defaultVirtualizationType := "/dev/sd" // default :  linux
	availableVolumeNames := []string{"f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p"}

	diskDeviceList, err := DescribeInstanceDiskDeviceList(svc, vmIID)
	if err != nil {
		return nil, err
	}

	availableDevices := []string{}
	// 이미 사용중인이름은 제외
	isAvailable := true
	for _, avn := range availableVolumeNames {
		device := defaultVirtualizationType + avn
		for _, diskDevice := range diskDeviceList {

			cblogger.Debug(device + " : " + *diskDevice.DeviceName)
			if device == *diskDevice.DeviceName {
				isAvailable = false
				break
			}
		}
		if isAvailable {
			availableDevices = append(availableDevices, device)
		}
	}

	return availableDevices, nil
}

// ---------------- Instance Area end ---------------//

// ---------------- VOLUME Area begin -----------------//
// WaitUntilVolumeAvailable
func WaitUntilVolumeAvailable(svc *ec2.EC2, volumeID string) error {
	input := &ec2.DescribeVolumesInput{
		VolumeIds: []*string{
			aws.String(volumeID),
		},
	}
	err := svc.WaitUntilVolumeAvailable(input)
	if err != nil {
		cblogger.Errorf("failed to wait until volume available: %v", err)
		return err
	}
	cblogger.Info("=========WaitUntilVolumeAvailable() 종료")
	return nil
}

// WaitUntilVolumeDeleted
func WaitUntilVolumeDeleted(svc *ec2.EC2, volumeID string) error {
	input := &ec2.DescribeVolumesInput{
		VolumeIds: []*string{
			aws.String(volumeID),
		},
	}
	err := svc.WaitUntilVolumeDeleted(input)
	if err != nil {
		cblogger.Errorf("failed to wait until volume deleted: %v", err)
		return err
	}
	cblogger.Info("=========WaitUntilVolumeDeleted() 종료")
	return nil
}

// WaitUntilVolumeInUse : attached
func WaitUntilVolumeInUse(svc *ec2.EC2, volumeID string) error {
	input := &ec2.DescribeVolumesInput{
		VolumeIds: []*string{
			aws.String(volumeID),
		},
	}
	err := svc.WaitUntilVolumeInUse(input)
	if err != nil {
		cblogger.Errorf("failed to wait until volume in use: %v", err)
		return err
	}
	cblogger.Info("=========WaitUntilVolumeInUse() 종료")
	return nil
}

/*
List 와 Get 이 같은 API 호출
filter 조건으로 VolumeId 를 넣도록하고
return은 그대로 DescribeVolumesOutput.
Get에서는 1개만 들어있으므로 [0]번째 사용

각 항목을 irs.DiskInfo로 변환하는 convertVolumeInfoToDiskInfo 로 필요Data 생성
*/
func DescribeVolumnes(svc *ec2.EC2, volumeIdList []*string) (*ec2.DescribeVolumesOutput, error) {

	input := &ec2.DescribeVolumesInput{}

	if volumeIdList != nil {
		input.VolumeIds = volumeIdList
	}

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		ResourceType: call.DISK,
		CloudOSAPI:   "DescribeVolumes",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()

	result, err := svc.DescribeVolumes(input)

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return nil, err
	}
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	return result, nil
}

func DescribeVolumneById(svc *ec2.EC2, volumeId string) (*ec2.Volume, error) {
	volumeIdList := []*string{}
	//input := &ec2.DescribeVolumesInput{}
	//
	//if volumeId != "" {
	//	volumeIdList = append(volumeIdList, aws.String(volumeId))
	//	input.VolumeIds = volumeIdList
	//}
	//result, err := svc.DescribeVolumes(input)

	volumeIdList = append(volumeIdList, aws.String(volumeId))
	result, err := DescribeVolumnes(svc, volumeIdList)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return nil, err
	}
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	for _, volume := range result.Volumes {
		if strings.EqualFold(volumeId, *volume.VolumeId) {
			//break
			return volume, nil
		}
	}

	return nil, awserr.New("404", "["+volumeId+"] Volume Not Found", nil)
}

func DescribeVolumnesBySnapshot(svc *ec2.EC2, snapShotIIDs []string) (*ec2.DescribeVolumesOutput, error) {
	var ids []*string
	for _, snapShotIID := range snapShotIIDs {
		ids = append(ids, aws.String(snapShotIID))
	}
	input := &ec2.DescribeVolumesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("snapshot-id"),
				//Values: ids,
				Values: []*string{
					aws.String(snapShotIIDs[0]),
				},
			},
		},
	}

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		ResourceType: call.DISK,
		CloudOSAPI:   "DescribeVolumnesBySnapshot",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()

	result, err := svc.DescribeVolumes(input)
	callogger.Info("DescribeVolumnesBySnapshot   IN PU T")
	spew.Dump(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return nil, err
	}
	//if cblogger.Level.String() == "debug" {
	spew.Dump(result.Volumes)
	//}

	return result, nil
}
func AttachVolume(svc *ec2.EC2, deviceName string, instanceId string, volumeId string) error {
	input := &ec2.AttachVolumeInput{
		Device:     aws.String(deviceName),
		InstanceId: aws.String(instanceId),
		VolumeId:   aws.String(volumeId),
	}

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		ResourceType: call.DISK,
		CloudOSAPI:   "AttachVolume",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()

	result, err := svc.AttachVolume(input)

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return err
	}

	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	err = WaitUntilVolumeInUse(svc, volumeId)
	if err != nil {
		return err
	}
	return nil
}

func DeleteDisk(svc *ec2.EC2, disks []irs.IID) (bool, error) {
	returnResult := false
	if disks != nil && len(disks) > 0 {
		for _, diskIID := range disks {
			input := &ec2.DeleteVolumeInput{
				VolumeId: aws.String(diskIID.SystemId),
			}

			result, err := svc.DeleteVolume(input)
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok {
					switch aerr.Code() {
					default:
						fmt.Println(aerr.Error())
					}
				} else {
					// Print the error, cast err to awserr.Error to get the Code and
					// Message from an error.
					fmt.Println(err.Error())
				}
				return false, err
			}

			if cblogger.Level.String() == "debug" {
				spew.Dump(result)
			}

			err = WaitUntilVolumeDeleted(svc, diskIID.SystemId)
			if err != nil {
				return false, err
			}

			returnResult = true
		}
	}

	return returnResult, nil
}

// ---------------- VOLUME Area end -----------------//

// ---------------- MyImage Area begin ---------------//

//func CreateImage(svc *ec2, EC2, )

func DescribeImages(svc *ec2.EC2, imageIIDs []*irs.IID, owners []*string) (*ec2.DescribeImagesOutput, error) {
	input := &ec2.DescribeImagesInput{}

	var imageIds []*string

	if imageIIDs != nil {
		for _, imageIID := range imageIIDs {
			imageIds = append(imageIds, aws.String(imageIID.SystemId))
		}
		input.ImageIds = imageIds
	}

	// ExecutableUser : 공유 받은 이미지인가? self, userid 지정시 조회 결과 '0'
	//request.ExecutableUsers.Add("all");
	//request.Owners.Add("amazon");

	//if executableBy != nil {
	//	input.ExecutableUsers = executableBy
	//}
	//input.ExecutableUsers = []*string{aws.String("0508-6470-2683")}	// wrong
	//input.ExecutableUsers = []*string{aws.String("050864702683")}// 0
	//input.ExecutableUsers = []*string{aws.String("all")}// 전체 : 56551개
	//input.ExecutableUsers = []*string{aws.String("self")} // 0

	// ExecutableUsers = all, owner = amazon => 10061
	//input.ExecutableUsers = []*string{aws.String("all")}// all image
	//input.Owners = []*string{aws.String("amazon")}

	// ExecutableUsers = all, 특정 유저 id
	//input.Owners = []*string{aws.String("013907871322")} // 270   suse linux 소유한 public image
	//input.Owners = []*string{aws.String("801119661308")} //1186  microsoft 가 소유한 public image

	// 소유한 갯수 : self로 하거나 12자리 숫자인 userid를 넣거나.
	//input.Owners = []*string{aws.String("self")}
	//input.Owners = []*string{aws.String("050864702683")}	// self 와 소유자 계정ID가 같은 결과. 그러므로 self 사용

	input.Owners = owners

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		ResourceType: call.VMIMAGE,
		CloudOSAPI:   "DescribeImages",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()

	result, err := svc.DescribeImages(input)

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
	}

	callogger.Info(call.String(callLogInfo))

	return result, err
}

func DescribeImageById(svc *ec2.EC2, imageIID *irs.IID, owners []*string) (*ec2.Image, error) {
	var imageIIDs []*irs.IID
	var iid irs.IID

	if *imageIID == iid {
		return nil, errors.New("imageID is empty.")
	}

	imageIIDs = append(imageIIDs, imageIID)

	result, err := DescribeImages(svc, imageIIDs, owners)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
	}

	if result.Images == nil || len(result.Images) == 0 {
		return nil, awserr.New("404", "["+imageIID.SystemId+"] Image Not Found", nil)
	}
	resultImage := result.Images[0]
	return resultImage, err
}

// Image 정보에서 image size(GB) return
func GetImageSizeFromEc2Image(ec2Image *ec2.Image) (int64, error) {
	if !reflect.ValueOf(ec2Image.BlockDeviceMappings).IsNil() {
		if !reflect.ValueOf(ec2Image.BlockDeviceMappings[0].Ebs).IsNil() {
			isize := aws.Int64(*ec2Image.BlockDeviceMappings[0].Ebs.VolumeSize)
			return *isize, nil
		} else {
			cblogger.Error("Ebs information not found in BlockDeviceMappings.")
			return -1, errors.New("Ebs information not found in BlockDeviceMappings.")
		}
	} else {
		cblogger.Error("BlockDeviceMappings information not found.")
		return -1, errors.New("BlockDeviceMappings information not found.")
	}
}

// Image 정보에서 Snapshot Id return
//
//	func GetSnapshotIdFromEc2Image(ec2Image *ec2.Image) (string, error) {
//		if !reflect.ValueOf(ec2Image.BlockDeviceMappings).IsNil() {
//			if !reflect.ValueOf(ec2Image.BlockDeviceMappings[0].Ebs).IsNil() {
//				snapshotId := *ec2Image.BlockDeviceMappings[0].Ebs.SnapshotId
//				return snapshotId, nil
//			} else {
//				cblogger.Error("Ebs information not found in BlockDeviceMappings.")
//				return "", errors.New("Ebs information not found in BlockDeviceMappings.")
//			}
//		} else {
//			cblogger.Error("BlockDeviceMappings information not found.")
//			return "", errors.New("BlockDeviceMappings information not found.")
//		}
//	}
func GetSnapshotIdFromEc2Image(ec2Image *ec2.Image) ([]string, error) {
	var snapshotIds []string
	if !reflect.ValueOf(ec2Image.BlockDeviceMappings).IsNil() {
		// rootdisk 찾기
		// if !reflect.ValueOf(ec2Image.BlockDeviceMappings[0].Ebs).IsNil() {
		// 	snapshotId := *ec2Image.BlockDeviceMappings[0].Ebs.SnapshotId
		// 	return snapshotId, nil
		// } else {
		// 	cblogger.Error("Ebs information not found in BlockDeviceMappings.")
		// 	return "", errors.New("Ebs information not found in BlockDeviceMappings.")
		// }
		for _, blockDevice := range ec2Image.BlockDeviceMappings {
			if !reflect.ValueOf(blockDevice.Ebs).IsNil() {
				snapshotId := *blockDevice.Ebs.SnapshotId
				snapshotIds = append(snapshotIds, snapshotId)
			}
		}
	} else {
		cblogger.Error("BlockDeviceMappings information not found.")
		return snapshotIds, errors.New("BlockDeviceMappings information not found.")
	}

	return snapshotIds, nil
}

// 이미지에서 루트 볼륨 외 disk ID들을 return
func GetDisksFromEc2Image(ec2Image *ec2.Image) ([]irs.IID, error) {
	diskIIDs := []irs.IID{}
	if !reflect.ValueOf(ec2Image.BlockDeviceMappings).IsNil() {
		if !reflect.ValueOf(ec2Image.BlockDeviceMappings[0].Ebs).IsNil() {
			//snapshotId := *ec2Image.BlockDeviceMappings[0].Ebs.SnapshotId
			return diskIIDs, nil
		} else {
			cblogger.Error("Ebs information not found in BlockDeviceMappings.")
			return diskIIDs, errors.New("Ebs information not found in BlockDeviceMappings.")
		}
	} else {
		cblogger.Error("BlockDeviceMappings information not found.")
		return diskIIDs, errors.New("BlockDeviceMappings information not found.")
	}
}

// Image 정보에서 osType return
func GetOsTypeFromEc2Image(ec2Image *ec2.Image) string {
	var guestOS string
	//주로 윈도우즈는 Platform 정보가 존재하며 리눅스 계열은 PlatformDetails만 존재하는 듯. - "Linux/UNIX"
	//윈도우즈 계열은 PlatformDetails에는 "Windows with SQL Server Standard"처럼 SQL정보도 포함되어있음.
	if !reflect.ValueOf(ec2Image.Platform).IsNil() {
		cblogger.Info("guestOS =", *ec2Image.Platform)
		guestOS = *ec2Image.Platform //Linux/UNIX

	} else {
		// Platform 정보가 없는 경우 PlatformDetails 정보가 존재하면 PlatformDetails 값을 이용함.
		if !reflect.ValueOf(ec2Image.PlatformDetails).IsNil() {
			guestOS = *ec2Image.PlatformDetails //Linux/UNIX
		}
	}
	return guestOS
}

// ---------------- MyImage Area end ---------------//

// ---------------- EBS Snapshot area begin --------//
//func DescribeSnapshots(svc *ec2.EC2, snapshotIIDs []*irs.IID) (*ec2.DescribeSnapshotsOutput, error) {
//	input := &ec2.DescribeSnapshotsInput{
//		//Filters: []*ec2.Filter{
//		//	{
//		//		Name:   aws.String("instance-state-name"),
//		//		Values: []*string{aws.String("running"), aws.String("pending")},
//		//	},
//		//},
//	}
//
//	var snapshotIds []*string
//
//	if snapshotIIDs != nil {
//		for _, snapshotId := range snapshotIIDs {
//			snapshotIds = append(snapshotIds, aws.String(snapshotId.SystemId))
//		}
//		input.SnapshotIds = snapshotIds
//	}
//	//fmt.Println("sign name " + svc.Client.SigningName)// ec2
//
//	//input.OwnerIds = []*string{aws.String("050864702683")}
//	input.OwnerIds = []*string{aws.String("self")}
//
//	result, err := svc.DescribeSnapshots(input)
//	if err != nil {
//		if aerr, ok := err.(awserr.Error); ok {
//			switch aerr.Code() {
//			default:
//				fmt.Println(aerr.Error())
//			}
//		} else {
//			// Print the error, cast err to awserr.Error to get the Code and
//			// Message from an error.
//			fmt.Println(err.Error())
//		}
//	}
//	spew.Dump(result)
//	return result, err
//}
//func DescribeSnapshotById(svc *ec2.EC2, snapshotIID *irs.IID) (*ec2.Snapshot, error) {
//	var snapshotIIDs []*irs.IID
//	var iid irs.IID
//
//	if *snapshotIID == iid {
//		return nil, errors.New("snapshot ID is empty.)")
//	}
//
//	snapshotIIDs = append(snapshotIIDs, snapshotIID)
//
//	result, err := DescribeSnapshots(svc, snapshotIIDs)
//	if err != nil {
//		if aerr, ok := err.(awserr.Error); ok {
//			switch aerr.Code() {
//			default:
//				fmt.Println(aerr.Error())
//			}
//		} else {
//			// Print the error, cast err to awserr.Error to get the Code and
//			// Message from an error.
//			fmt.Println(err.Error())
//		}
//		return nil, err
//	}
//
//	resultSnapshot := result.Snapshots[0]
//	return resultSnapshot, err
//}

// ---------------- EBS Snapshot area end ----------//
