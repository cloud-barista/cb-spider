package resources

//https://docs.aws.amazon.com/sdk-for-go/api/service/elb

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"
	"github.com/davecgh/go-spew/spew"
)

type AwsDiskHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

var VOLUME_TYPE = []string{"standard", "io1", "io2", "gp2", "gp3", "sc1", "st1"} // array 는 const 불가하여 변수로 처리.
const (
	AWS_VOLUME_STATE_CREATING  = "creating"
	AWS_VOLUME_STATE_AVAILABLE = "available"
	AWS_VOLUME_STATE_INUSE     = "in-use"
	AWS_VOLUME_STATE_DELETING  = "deleting"
	AWS_VOLUME_STATE_ERROR     = "error"

	AWS_VOLUME_ATTACH_STATE_ATTACHING = "attaching"
	AWS_VOLUME_ATTACH_STATE_ATTACHED  = "attached"
	AWS_VOLUME_ATTACH_STATE_DETACHING = "detaching"
	AWS_VOLUME_ATTACH_STATE_DETACHED  = "detached"
	AWS_VOLUME_ATTACH_STATE_BUSY      = "busy"

	RESOURCE_TYPE_VOLUME = "volume"
	VOLUME_TAG_DEFAULT   = "Name"
)

//WaitUntilVolumeAvailable
//WaitUntilVolumeDeleted
//WaitUntilVolumeInUse

//------ Disk Management

/*
Spider 의 Disk = AWS 의 Volume

disk type에 따라 달라짐
- 빈 disk
- snapshot에서 가져오는 경우
- 암호화 된 disk
*/
func (DiskHandler *AwsDiskHandler) CreateDisk(diskReqInfo irs.DiskInfo) (irs.DiskInfo, error) {

	zone := DiskHandler.Region.Zone
	spew.Dump(DiskHandler.Region)
	err := validateCreateDisk(&diskReqInfo)
	if err != nil {
		return irs.DiskInfo{}, err
	}

	volumeSize, _ := strconv.ParseInt(diskReqInfo.DiskSize, 10, 64)
	volumeType := diskReqInfo.DiskType

	// volume 이름을 위해 Tag 지정.
	tag := &ec2.Tag{
		Key:   aws.String(VOLUME_TAG_DEFAULT),
		Value: &diskReqInfo.IId.NameId,
	}

	var tags []*ec2.Tag
	tags = append(tags, tag)
	tagSpec := &ec2.TagSpecification{
		ResourceType: aws.String(RESOURCE_TYPE_VOLUME),
		Tags:         tags,
	}
	var tagSpecs []*ec2.TagSpecification
	tagSpecs = append(tagSpecs, tagSpec)

	input := &ec2.CreateVolumeInput{
		AvailabilityZone:  aws.String(zone),
		Size:              aws.Int64(volumeSize),
		TagSpecifications: tagSpecs,
	}

	switch diskReqInfo.DiskType {
	case "":
	}
	// case1 : 빈 disk 생성
	input.VolumeType = aws.String(volumeType)

	// case2 : snapshot에서 disk 생성
	//Iops:             aws.Int64(1000),
	//SnapshotId:       aws.String("snap-066877671789bd71b"),

	// case3 : 암호화된 disk 생성
	//input.Encrypted = true
	//input.KmsKeyId = ""
	spew.Dump(input)
	result, err := DiskHandler.Client.CreateVolume(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return irs.DiskInfo{}, err
	}

	newVolume := irs.IID{}
	newVolume.NameId = diskReqInfo.IId.NameId
	newVolume.SystemId = *result.VolumeId
	err = WaitUntilVolumeAvailable(DiskHandler.Client, newVolume.SystemId)
	if err != nil {
		return irs.DiskInfo{}, err
	}

	returnDiskInfo, err := DiskHandler.GetDisk(newVolume)
	if err != nil {
		return irs.DiskInfo{}, err
	}
	return returnDiskInfo, nil
}

/*
GetDisk와 ListDisk 처리로직 동일하므로 DescribeVolumes 호출.
단, IDList를 nil로 set.
*/
func (DiskHandler *AwsDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	//Filters []*Filter `locationName:"Filter" locationNameList:"Filter" type:"list"`
	//MaxResults *int64 `locationName:"maxResults" type:"integer"`
	//VolumeIds []*string `locationName:"VolumeId" locationNameList:"VolumeId" type:"list"`

	result, err := DescribeVolumnes(DiskHandler.Client, nil)
	if err != nil {
		return nil, err
	}

	var returnDiskInfoList []*irs.DiskInfo
	for _, volumeInfo := range result.Volumes {
		diskInfo, err := DiskHandler.convertVolumeInfoToDiskInfo(volumeInfo)
		if err != nil {
			fmt.Println(err.Error())
		}
		returnDiskInfoList = append(returnDiskInfoList, &diskInfo)
	}
	return returnDiskInfoList, nil
}

/*
ListDisk와 처리로직 동일 but, volumeID 로 호출하므로 1개만 return.
*/
func (DiskHandler *AwsDiskHandler) GetDisk(diskIID irs.IID) (irs.DiskInfo, error) {
	var diskIds []*string
	diskIds = append(diskIds, &diskIID.SystemId)

	result, err := DescribeVolumnes(DiskHandler.Client, diskIds)
	if err != nil {
		return irs.DiskInfo{}, err
	}

	diskInfo, err := DiskHandler.convertVolumeInfoToDiskInfo(result.Volumes[0])
	return diskInfo, nil
}

/*
		IOPS : Default: The existing value is retained if you keep the same volume type.
			If you change the volume type to io1, io2, or gp3, the default is 3,000.
	    //    * gp2 and gp3: 1-16,384
	    //
	    //    * io1 and io2: 4-16,384
	    //
	    //    * st1 and sc1: 125-16,384
	    //
	    //    * standard: 1-1,024
		Size *int64 `type:"integer"`
*/
func (DiskHandler *AwsDiskHandler) ChangeDiskSize(diskIID irs.IID, size string) (bool, error) {

	diskInfo, err := DiskHandler.GetDisk(diskIID)
	if err != nil {
		return false, err
	}

	err = validateModifyDisk(diskInfo, size)
	if err != nil {
		return false, err
	}
	// requestParameters
	// DryRun
	// Iops

	diskSize, _ := strconv.ParseInt(size, 10, 64)
	input := &ec2.ModifyVolumeInput{
		VolumeId: aws.String(diskIID.SystemId),
		Size:     aws.Int64(diskSize),
	}

	result, err := DiskHandler.Client.ModifyVolume(input)
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
	cblogger.Debug("originalSize : " + strconv.Itoa(int(*result.VolumeModification.OriginalSize)))
	cblogger.Debug("targetSize : " + strconv.Itoa(int(*result.VolumeModification.TargetSize)))

	// err = WaitUntilVolumeInUse(DiskHandler.Client, diskIID.SystemId)
	// if err != nil {
	// 	return false, err
	// }
	return true, nil
}
func (DiskHandler *AwsDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {

	input := &ec2.DeleteVolumeInput{
		VolumeId: aws.String(diskIID.SystemId),
	}

	result, err := DiskHandler.Client.DeleteVolume(input)
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

	err = WaitUntilVolumeDeleted(DiskHandler.Client, diskIID.SystemId)
	if err != nil {
		return false, err
	}

	return true, nil
}

//------ Disk Attachment

/*
EBS multi-attach does not support XFS, EXT2, EXT4, and NTFS file systems. It supports only cluster-aware file systems.

Linux용 권장 디바이스 이름: 루트 볼륨의 경우 /dev/sda1, 데이터 볼륨의 경우 /dev/sd[f-p].
Windows용 권장 디바이스 이름: 루트 볼륨의 경우 /dev/sda1, 데이터 볼륨의 경우 xvd[f-p].
max은 아직 권장 디바이스 이름 없고 Linux용 이름 사용

	//device := vmInfo.VMBlockDisk // "/dev/sda1"  이미 있는 이름. rootdisk에서 사용
	//device := "/dev/sdf" [f ~ p] 사이의 값
	//device := "/dev/sdf/aa/" // invalid
*/
func (DiskHandler *AwsDiskHandler) AttachDisk(diskIID irs.IID, ownerVM irs.IID) (irs.DiskInfo, error) {

	// getVM에서 정보 추출
	//VmHandler := AwsVMHandler{Client: DiskHandler.Client}
	//vmInfo, errGetVM := VmHandler.GetVM(ownerVM)
	//if errGetVM != nil {
	//	return irs.DiskInfo{}, errGetVM
	//}
	//for _, kv := range vmInfo.KeyValueList {
	//	if kv.Key == "VirtualizationType" {
	//		virtualizationType = kv.Value
	//		break
	//	}
	//}
	device := ""

	defaultVirtualizationType := "/dev/sd" // default :  linux
	availableVolumeNames := []string{"f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p"}

	diskDeviceList, err := DescribeInstanceDiskDeviceList(DiskHandler.Client, ownerVM)
	if err != nil {
		return irs.DiskInfo{}, err
	}

	spew.Dump(diskDeviceList)
	if diskDeviceList != nil {
		isAvailable := true
		for _, avn := range availableVolumeNames {
			device = defaultVirtualizationType + avn

			for _, diskDevice := range diskDeviceList {
				if *diskDevice.DeviceName == "/dev/sda1" { // root disk 는 skip.
					continue
				} // rootdisk

				cblogger.Debug(device + " : " + *diskDevice.DeviceName)
				if device == *diskDevice.DeviceName {
					isAvailable = false
					continue
				} else {
					isAvailable = true
					break
				}
			}
			if isAvailable {
				break
			}
		}

		if !isAvailable {
			// 다 돌았는데도 사용할 이름이 없으면 error
			return irs.DiskInfo{}, errors.New("There are no names available")
		}
	} else {
		device = defaultVirtualizationType + availableVolumeNames[0]
	}

	// 만약 hvm, pv 별로 device 이름을 달리줘야하면 instance에서 virtualizationType 같은 값으로 추가 처리 필요
	//virtualizationType := output.Reservations[0].Instances[0].VirtualizationType

	//switch virtualizationType {
	//case "hvm":
	//case "pv":
	//}

	//rootDeviceName := vmInfo.RootDeviceName
	//blockDeviceName := vmInfo.VMBlockDisk

	// os가 window일 때
	// os가 linux일 때
	// ...

	input := &ec2.AttachVolumeInput{
		//Device:     aws.String("/dev/sdf"),
		//InstanceId: aws.String("i-01474ef662b89480"),
		//VolumeId:   aws.String("vol-1234567890abcdef0"),
		Device:     aws.String(device),
		InstanceId: aws.String(ownerVM.SystemId),
		VolumeId:   aws.String(diskIID.SystemId),
	}

	result, err := DiskHandler.Client.AttachVolume(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return irs.DiskInfo{}, err
	}

	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	err = WaitUntilVolumeInUse(DiskHandler.Client, diskIID.SystemId)
	if err != nil {
		return irs.DiskInfo{}, err
	}

	returnDiskInfo, err := DiskHandler.GetDisk(diskIID)
	if err != nil {
		return irs.DiskInfo{}, err
	}
	return returnDiskInfo, nil
}
func (DiskHandler *AwsDiskHandler) DetachDisk(diskIID irs.IID, ownerVM irs.IID) (bool, error) {
	//// getVM에서 정보 추출
	//VmHandler := AwsVMHandler{Client: DiskHandler.Client}
	//vmInfo, errGetVM := VmHandler.GetVM(ownerVM)
	//if errGetVM != nil {
	//	return false, errGetVM
	//}
	//
	//device := vmInfo.VMBlockDisk

	input := &ec2.DetachVolumeInput{
		//Device:     aws.String("/dev/sdf"),
		//InstanceId: aws.String("i-01474ef662b89480"),
		//VolumeId:   aws.String("vol-1234567890abcdef0"),

		//Device:     aws.String(device),	// Required : No
		InstanceId: aws.String(ownerVM.SystemId), // Required : No
		VolumeId:   aws.String(diskIID.SystemId), // Required : Yes
	}

	result, err := DiskHandler.Client.DetachVolume(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			fmt.Println(err.Error())
		}
		return false, err
	}

	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	err = WaitUntilVolumeInUse(DiskHandler.Client, diskIID.SystemId)
	if err != nil {
		return false, err
	}

	return true, nil
}

/*
VolumeType : https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-volume-types.html

	The volume type. This parameter can be one of the following values:
	General Purpose SSD: gp2 | gp3
	Provisioned IOPS SSD: io1 | io2
	Throughput Optimized HDD: st1
	Cold HDD: sc1
	Magnetic: standard

Size :

	The size of the volume, in GiBs.
	You must specify either a snapshot ID or a volume size.
	If you specify a snapshot, the default is the snapshot size.
	You can specify a volume size that is equal to or larger than the snapshot size.

IOPS :

	gp3: 3,000-16,000 IOPS
	io1: 100-64,000 IOPS
	io2: 100-64,000 IOPS

MultiAttachEnabled : io1, io2 only

	If you enable Multi-Attach, you can attach the volume to up to 16 Instances built on the Nitro System in the same Availability Zone

Throughput

	The throughput to provision for a volume, with a maximum of 1,000 MiB/s.
	This parameter is valid only for gp3 volumes.
	Valid Range: Minimum value of 125. Maximum value of 1000.
*/
func validateCreateDisk(diskReqInfo *irs.DiskInfo) error {
	// VolumeType
	cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo("AWS")
	arrDiskType := cloudOSMetaInfo.DiskType
	arrRootDiskType := cloudOSMetaInfo.RootDiskType
	arrDiskSizeOfType := cloudOSMetaInfo.DiskSize

	cblogger.Info(arrDiskType)
	reqDiskType := diskReqInfo.DiskType
	reqDiskSize := diskReqInfo.DiskSize
	if reqDiskType == "" || reqDiskType == "default" {
		reqDiskType = arrRootDiskType[0]
		diskReqInfo.DiskType = arrRootDiskType[0]
	}

	// 정의된 type인지
	if !ContainString(arrDiskType, reqDiskType) {
		return errors.New("Disktype : " + reqDiskType + " is not valid")
	}

	if reqDiskSize == "" || reqDiskSize == "default" {
		for _, diskSizeInfo := range arrDiskSizeOfType {
			diskSizeArr := strings.Split(diskSizeInfo, "|")
			if strings.EqualFold(reqDiskType, diskSizeArr[0]) {
				reqDiskSize = diskSizeArr[1]
				diskReqInfo.DiskSize = diskSizeArr[1] // set default value
				break
			}
		}

	}

	// volume Size
	volumeSize, err := strconv.ParseInt(reqDiskSize, 10, 64)
	if err != nil {
		return err
	}

	type diskSizeModel struct {
		diskType    string
		diskMinSize int64
		diskMaxSize int64
		unit        string
	}

	diskSizeValue := diskSizeModel{}
	isExists := false
	for idx, _ := range arrDiskSizeOfType {
		diskSizeArr := strings.Split(arrDiskSizeOfType[idx], "|")
		if strings.EqualFold(diskReqInfo.DiskType, diskSizeArr[0]) {
			diskSizeValue.diskType = diskSizeArr[0]
			diskSizeValue.unit = diskSizeArr[3]
			diskSizeValue.diskMinSize, err = strconv.ParseInt(diskSizeArr[1], 10, 64)
			if err != nil {
				cblogger.Error(err)
				return err
			}

			diskSizeValue.diskMaxSize, err = strconv.ParseInt(diskSizeArr[2], 10, 64)
			if err != nil {
				cblogger.Error(err)
				return err
			}
			isExists = true
		}
	}
	if !isExists {
		return errors.New("Invalid Root Disk Type : " + diskReqInfo.DiskType)
	}

	if volumeSize < diskSizeValue.diskMinSize {
		fmt.Println("Disk Size Error!!: ", volumeSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
		return errors.New("Disk Size must be at least the default size (" + strconv.FormatInt(diskSizeValue.diskMinSize, 10) + " GB).")
	}

	if volumeSize > diskSizeValue.diskMaxSize {
		fmt.Println("Disk Size Error!!: ", volumeSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
		return errors.New("Disk Size must be smaller than the maximum size (" + strconv.FormatInt(diskSizeValue.diskMaxSize, 10) + " GB).")
	}

	//switch diskReqInfo.DiskType {
	//case "standard":
	//	if volumeSize < 1 || volumeSize > 1024 {
	//		return errors.New("Disktype : " + diskReqInfo.DiskType + "' supports 1 to 1024")
	//	}
	//case "gp2":
	//	if volumeSize < 1 || volumeSize > 16384 {
	//		return errors.New("Disktype : " + diskReqInfo.DiskType + "' supports 1 to 16384")
	//	}
	//case "gp3":
	//	if volumeSize < 1 || volumeSize > 16384 {
	//		return errors.New("Disktype : " + diskReqInfo.DiskType + "' supports 1 to 16384")
	//	}
	//
	//	//throughput, err := strconv.ParseInt(diskReqInfo., 10, 64)
	//	// min :125, max :  1000
	//case "io1":
	//	if volumeSize < 4 || volumeSize > 16384 {
	//		return errors.New("Disktype : " + diskReqInfo.DiskType + "' supports 4 to 16384")
	//	}
	//case "io2":
	//	if volumeSize < 4 || volumeSize > 16384 {
	//		return errors.New("Disktype : " + diskReqInfo.DiskType + "' supports 4 to 16384")
	//	}
	//case "st1":
	//	if volumeSize < 125 || volumeSize > 16384 {
	//		return errors.New("Disktype : " + diskReqInfo.DiskType + "' supports 125 to 16384")
	//	}
	//case "sc1":
	//	if volumeSize < 125 || volumeSize > 16384 {
	//		return errors.New("Disktype : " + diskReqInfo.DiskType + "' supports 125 to 16384")
	//	}
	//default:
	//	return errors.New("Invalid DiskType : " + diskReqInfo.DiskType)
	//}

	// IOPS : gp3, io1, io2
	//iopsSize, err := strconv.ParseInt(diskReqInfo.IOPS, 10, 64)
	//if err != nil { return err }
	//switch diskReqInfo.DiskType {
	//gp3: 3,000-16,000 IOPS
	//
	//io1: 100-64,000 IOPS
	//
	//io2: 100-64,000 IOPS
	//case "gp2":
	//case "gp3":
	//	if iopsSize < 3000 || iopsSize > 16000 {
	//		return errors.New("IOPS : " + iopsSize + "' supports 3,000 to 16,000")
	//	}
	//
	//case "io1":
	//	if iopsSize < 100 || iopsSize > 64000 {
	//		return errors.New("IOPS : " + iopsSize + "' supports 100 to 64000")
	//	}
	//case "io2":
	//	if iopsSize < 100 || iopsSize > 64000 {
	//		return errors.New("IOPS : " + iopsSize + "' supports 100 to 64000")
	//	}
	//}

	// encryption 이 true 면 KmsKeyId set 해야 함

	// MultiAttachEnabled
	return nil
}

func validateModifyDisk(diskReqInfo irs.DiskInfo, diskSize string) error {

	// volume Size
	orgVolumeSize, err := strconv.ParseInt(diskReqInfo.DiskSize, 10, 64)
	if err != nil {
		return err
	}

	targetVolumeSize, err := strconv.ParseInt(diskSize, 10, 64)
	if err != nil {
		return err
	}

	if orgVolumeSize < targetVolumeSize {
	} else {
		return errors.New("Target DiskSize : " + diskSize + " must be greater than Original DiskSize " + diskReqInfo.DiskSize)
	}

	cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo("AWS")
	arrDiskType := cloudOSMetaInfo.DiskType
	arrDiskSizeOfType := cloudOSMetaInfo.DiskSize

	// 정의된 type인지
	if !ContainString(arrDiskType, diskReqInfo.DiskType) {
		return errors.New("Disktype : " + diskReqInfo.DiskType + " is not valid")
	}

	type diskSizeModel struct {
		diskType    string
		diskMinSize int64
		diskMaxSize int64
		unit        string
	}

	diskSizeValue := diskSizeModel{}
	isExists := false
	for idx, _ := range arrDiskSizeOfType {
		diskSizeArr := strings.Split(arrDiskSizeOfType[idx], "|")
		if strings.EqualFold(diskReqInfo.DiskType, diskSizeArr[0]) {
			diskSizeValue.diskType = diskSizeArr[0]
			diskSizeValue.unit = diskSizeArr[3]
			diskSizeValue.diskMinSize, err = strconv.ParseInt(diskSizeArr[1], 10, 64)
			if err != nil {
				cblogger.Error(err)
				return err
			}

			diskSizeValue.diskMaxSize, err = strconv.ParseInt(diskSizeArr[2], 10, 64)
			if err != nil {
				cblogger.Error(err)
				return err
			}
			isExists = true
		}
	}
	if !isExists {
		return errors.New("Invalid Root Disk Type : " + diskReqInfo.DiskType)
	}

	if targetVolumeSize < diskSizeValue.diskMinSize {
		fmt.Println("Disk Size Error!!: ", targetVolumeSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
		return errors.New("Root Disk Size must be at least the default size (" + strconv.FormatInt(diskSizeValue.diskMinSize, 10) + " GB).")
	}

	if targetVolumeSize > diskSizeValue.diskMaxSize {
		fmt.Println("Disk Size Error!!: ", targetVolumeSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
		return errors.New("Root Disk Size must be smaller than the maximum size (" + strconv.FormatInt(diskSizeValue.diskMaxSize, 10) + " GB).")
	}

	// VolumeType
	// valid type : standard | io1 | io2 | gp2 | sc1 | st1 | gp3
	//if !ContainString(arrDiskSizeOfType, diskReqInfo.DiskType) {
	//	return errors.New("Disktype : " + diskReqInfo.DiskType + "' is not valid")
	//}
	//
	//switch diskReqInfo.DiskType {
	//case "gp3":
	//	if targetVolumeSize < 1 || targetVolumeSize > 16384 {
	//		return errors.New("Disktype : " + diskReqInfo.DiskType + "' supports 1 to 16384")
	//	}
	//case "io1":
	//	if targetVolumeSize < 4 || targetVolumeSize > 16384 {
	//		return errors.New("Disktype : " + diskReqInfo.DiskType + "' supports 4 to 16384")
	//	}
	//case "io2":
	//	if targetVolumeSize < 4 || targetVolumeSize > 16384 {
	//		return errors.New("Disktype : " + diskReqInfo.DiskType + "' supports 4 to 16384")
	//	}
	//default:
	//	return errors.New("Invalid DiskType : " + diskReqInfo.DiskType)
	//}

	// MultiAttachEnabled

	// Throughput : gp3 only, min 125, max 1000, default 125
	//switch diskReqInfo.DiskType {
	//case "gp3":
	//
	//}
	return nil
}

func convertVolumeStatusToDiskStatus(volumeState string, attachmentList []*ec2.VolumeAttachment, instanceId string) (irs.DiskStatus, error) {
	var returnStatus irs.DiskStatus

	switch volumeState {
	case AWS_VOLUME_STATE_CREATING:
		returnStatus = irs.DiskCreating
	case AWS_VOLUME_STATE_AVAILABLE:
		returnStatus = irs.DiskAvailable
	case AWS_VOLUME_STATE_INUSE:
		returnStatus = irs.DiskAttached
	case AWS_VOLUME_STATE_DELETING:
		returnStatus = irs.DiskDeleting
	default:
		returnStatus = irs.DiskError
		for _, attachment := range attachmentList {
			if strings.EqualFold(instanceId, *attachment.InstanceId) {
				switch *attachment.State {
				case AWS_VOLUME_ATTACH_STATE_ATTACHING:
					returnStatus = irs.DiskAttached
				case AWS_VOLUME_ATTACH_STATE_ATTACHED:
					returnStatus = irs.DiskAttached
				case AWS_VOLUME_ATTACH_STATE_DETACHING:
					returnStatus = irs.DiskAttached
				case AWS_VOLUME_ATTACH_STATE_DETACHED:
					returnStatus = irs.DiskAttached
				case AWS_VOLUME_ATTACH_STATE_BUSY:
					returnStatus = irs.DiskAttached
				}
				break // disk는 1개의 vm만 존재
			}
		}
	}

	//VolumeStateCreating = "creating"
	//
	//// VolumeStateAvailable is a VolumeState enum value
	//VolumeStateAvailable = "available"
	//
	//// VolumeStateInUse is a VolumeState enum value
	//VolumeStateInUse = "in-use"
	//
	//// VolumeStateDeleting is a VolumeState enum value
	//VolumeStateDeleting = "deleting"
	//
	//// VolumeStateDeleted is a VolumeState enum value
	//VolumeStateDeleted = "deleted"
	//
	//// VolumeStateError is a VolumeState enum value
	//VolumeStateError = "error"

	//// VolumeAttachmentStateAttaching is a VolumeAttachmentState enum value
	//VolumeAttachmentStateAttaching = "attaching"
	//
	//// VolumeAttachmentStateAttached is a VolumeAttachmentState enum value
	//VolumeAttachmentStateAttached = "attached"
	//
	//// VolumeAttachmentStateDetaching is a VolumeAttachmentState enum value
	//VolumeAttachmentStateDetaching = "detaching"
	//
	//// VolumeAttachmentStateDetached is a VolumeAttachmentState enum value
	//VolumeAttachmentStateDetached = "detached"
	//
	//// VolumeAttachmentStateBusy is a VolumeAttachmentState enum value
	//VolumeAttachmentStateBusy = "busy"

	//if strings.EqualFold(, volumeState) {
	//
	//}
	//if !ContainString(DiskStatus, diskReqInfo.DiskType) {
	//	return errors.New("Disktype : " + diskReqInfo.DiskType + "' is not valid")
	//}
	//Status 		DiskStatus	// DiskCreating | DiskAvailable | DiskAttached | DiskDeleting

	return returnStatus, nil
}
func (DiskHandler *AwsDiskHandler) convertVolumeInfoToDiskInfo(volumeInfo *ec2.Volume) (irs.DiskInfo, error) {
	diskInfo := irs.DiskInfo{}
	var diskName string

	for _, t := range volumeInfo.Tags {
		if *t.Key == "Name" {
			diskName = *t.Value
			break
		}
	}

	diskInfo.IId = irs.IID{NameId: diskName, SystemId: *volumeInfo.VolumeId}
	// tag에서 빼야하나?
	diskInfo.DiskSize = strconv.Itoa(int(*volumeInfo.Size))
	diskInfo.DiskType = *volumeInfo.VolumeType
	//diskInfo.Status = irs.DiskStatus(*volumeInfo.State) //State: "attached",

	attachments := volumeInfo.Attachments
	vmId := ""
	if attachments != nil {
		for _, attachment := range attachments {
			if attachment.InstanceId != nil {
				// disk는 1개의 vm에만 할당되므로 instanceId를 찾으면 더 찾을 필요 없음.
				vmId = *attachment.InstanceId
				break
			}
		}
	}
	cblogger.Info(vmId)
	diskStatus, errStatus := convertVolumeStatusToDiskStatus(*volumeInfo.State, attachments, vmId)
	if errStatus != nil {

		return irs.DiskInfo{}, errStatus
	}
	cblogger.Info(diskStatus)
	diskInfo.Status = diskStatus

	if !strings.EqualFold(vmId, "") {
		diskInfo.OwnerVM = irs.IID{SystemId: vmId}
		//VmHandler := AwsVMHandler{Client: DiskHandler.Client}
		//vmInfo, errVm := VmHandler.GetVM(irs.IID{SystemId: vmId})
		//if errVm != nil {
		//	return irs.DiskInfo{}, errStatus
		//}
		//diskInfo.OwnerVM = vmInfo.IId
		//spew.Dump(vmInfo)
	}

	diskInfo.CreatedTime = *volumeInfo.CreateTime
	spew.Dump(volumeInfo)
	//KeyValueList []KeyValue
	var inKeyValueList []irs.KeyValue
	//if !reflect.ValueOf(volumeInfo.Encrypted).IsNil() {
	inKeyValueList = append(inKeyValueList, irs.KeyValue{Key: "Encrypted", Value: strconv.FormatBool(*volumeInfo.Encrypted)})
	//}
	if !reflect.ValueOf(volumeInfo.Iops).IsNil() {
		inKeyValueList = append(inKeyValueList, irs.KeyValue{Key: "Iops", Value: strconv.Itoa(int(*volumeInfo.Iops))})
	}
	//inKeyValueList = append(inKeyValueList, icbs.KeyValue{Key: "KmsKeyId", Value: *volumeInfo.KmsKeyId})
	inKeyValueList = append(inKeyValueList, irs.KeyValue{Key: "MultiAttachEnabled", Value: strconv.FormatBool(*volumeInfo.MultiAttachEnabled)})

	//inKeyValueList = append(inKeyValueList, icbs.KeyValue{Key: "Tags", Value: strings.Join(volumeInfo.Tags, ",")})
	//inKeyValueList = append(inKeyValueList, icbs.KeyValue{Key: "OutpostArn", Value: *volumeInfo.OutpostArn})
	diskInfo.KeyValueList = inKeyValueList
	cblogger.Info("keyvalue2")
	if cblogger.Level.String() == "debug" {
		spew.Dump(diskInfo)
	}
	return diskInfo, nil
}
