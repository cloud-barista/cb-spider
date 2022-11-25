package resources

// https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.CreateImage

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

// https://docs.aws.amazon.com/sdk-for-go/api/service/ec2/#EC2.CreateImage
// Snapshot은 현재 운영 중인 자원의 상태를 저장한 후 필요 시에 동일한 상태로 복제하여 재 생산할 수 있는 기능을 말한다.
// Snapshot은 VM의 상태를 저장해주는 VM Snapshot과 Disk(Volume)의 상태를 저장해주는 Disk Snapshot이 존재한다.
// CB-Spider MyImage 관리 기능은 VM Snapshot 실행과 결과로 생성된 VM Image(MyImage)를 관리하는 기능을 제공한다
// CB-Spider VM Snapshot은 운영 중인 VM의 상태와 VM에 Attach된 Data-Disk의 상태도 저장된다.
type AwsMyImageHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

const (
	AWS_IMAGE_STATE_PENDING      = "pending"
	AWS_IMAGE_STATE_AVAILABLE    = "available"
	AWS_IMAGE_STATE_INVAILABLE   = "invalid"
	AWS_IMAGE_STATE_DEREGISTERED = "deregistered"
	AWS_IMAGE_STATE_TRANSIENT    = "transient"
	AWS_IMAGE_STATE_FAILED       = "failed"
	AWS_IMAGE_STATE_ERROR        = "error"

	RESOURCE_TYPE_MYIMAGE = "image"
	IMAGE_TAG_DEFAULT     = "Name"
	IMAGE_TAG_SOURCE_VM   = "CB-VMSNAPSHOT-SOURCEVM-ID"
)

//------ Snapshot to create a MyImage

//// https://www.msp360.com/resources/blog/backup-aws-ec2-instance/
//func (ImageHandler *AwsMyImageHandler) SnapshotVM(snapshotReqInfo irs.MyImageInfo) (irs.MyImageInfo, error) {
//	//Instance 정보 조회
//	instance, err := DescribeInstanceById(ImageHandler.Client, snapshotReqInfo.SourceVM)
//	if err != nil {
//		return irs.MyImageInfo{}, err
//	}
//
//	// instance 가 중지된 상태여야 함.
//	if *instance.State.Name == "running" {
//
//	}
//
//	var tags []*ec2.Tag
//	nameTag := &ec2.Tag{
//		Key:   aws.String(IMAGE_TAG_DEFAULT),
//		Value: aws.String(snapshotReqInfo.IId.NameId),
//	}
//	tags = append(tags, nameTag)
//	sourceVMTag := &ec2.Tag{
//		Key:   aws.String(IMAGE_TAG_SOURCE_VM),
//		Value: aws.String(snapshotReqInfo.IId.SystemId),
//	}
//	tags = append(tags, sourceVMTag)
//	tagSpec := &ec2.TagSpecification{
//		ResourceType: aws.String(RESOURCE_TYPE_SNAPSHOT),
//		Tags:         tags,
//	}
//	var tagSpecs []*ec2.TagSpecification
//	tagSpecs = append(tagSpecs, tagSpec)
//
//	input := &ec2.CreateSnapshotsInput{}
//	input.InstanceSpecification = &ec2.InstanceSpecification{InstanceId: instance.InstanceId}
//	input.TagSpecifications = tagSpecs
//
//	result, err := ImageHandler.Client.CreateSnapshots(input)
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
//		return irs.MyImageInfo{}, err
//	}
//
//	createdImageId := result.Snapshots[0].SnapshotId
//
//	myImage, err := ImageHandler.GetMyImage(irs.IID{SystemId: *createdImageId})
//
//	return myImage, nil
//}
//
////------ MyImage Management
//func (ImageHandler *AwsMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
//	var returnMyImageInfoList []*irs.MyImageInfo
//	//input := &ec2.DescribeImagesInput{}
//	//
//	//result, err := ImageHandler.Client.DescribeImages(input)
//	//result, err := DescribeImages(ImageHandler.Client, nil)
//
//	result, err := DescribeSnapshots(ImageHandler.Client, nil)
//
//	if err != nil {
//		return nil, err
//	}
//	spew.Dump(result)
//	for _, snapShot := range result.Snapshots {
//		myImage, err := convertAWSSnapShopToMyImageInfo(snapShot)
//		if err != nil {
//			// conver Error but continue;
//			//return nil, err
//		}
//		returnMyImageInfoList = append(returnMyImageInfoList, &myImage)
//	}
//
//	return returnMyImageInfoList, nil
//}
//
///*
//
//
//   public: The owner of the snapshot granted create volume permissions for the snapshot to the all group. All Amazon Web Services accounts have create volume permissions for these snapshots.
//   explicit: The owner of the snapshot granted create volume permissions to a specific Amazon Web Services account.
//   implicit: An Amazon Web Services account has implicit create volume permissions for all snapshots it owns.
//*/
//func (ImageHandler *AwsMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {
//	resultImage, err := DescribeSnapshotById(ImageHandler.Client, &myImageIID)
//	if err != nil {
//		return irs.MyImageInfo{}, err
//	}
//
//	returnMyImage, err := convertAWSSnapShopToMyImageInfo(resultImage)
//	return returnMyImage, nil
//}
//func (ImageHandler *AwsMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
//	input := &ec2.DeleteSnapshotInput{
//		SnapshotId: aws.String(myImageIID.SystemId),
//	}
//
//	result, err := ImageHandler.Client.DeleteSnapshot(input)
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
//		return false, err
//	}
//
//	//input := &ec2.DeregisterImageInput{}
//	//input.ImageId = aws.String(myImageIID.SystemId)
//	//
//	//result, err := ImageHandler.Client.DeregisterImage(input)
//	//if err != nil {
//	//	return false, err
//	//}
//	//
//	spew.Dump(result)
//	return true, nil
//}
//
////
//// AWS snapshot state 를 CB-SPIDER MyImage 의 statuf 로 변환
//func convertSnapshotStateToMyImageStatus(snapshotState *string) irs.MyImageStatus {
//	var returnStatus irs.MyImageStatus
//
//	//AWS_SNAPSHOT_STATE_COMPLETE = "completed"
//	//AWS_SNAPSHOT_STATE_PENDING = "pending"
//	//AWS_SNAPSHOT_STATE_RECOVERABLE = "recoverable"
//	//AWS_SNAPSHOT_STATE_RECOVERING = "recovering"
//	//AWS_SNAPSHOT_STATE_UNKNOWN_TO_SDK_VERSION = "unknownToSdkVersion"
//	//AWS_SNAPSHOT_STATE_ERROR = "error"
//
//	switch *snapshotState {
//	case AWS_SNAPSHOT_STATE_PENDING:
//		returnStatus = irs.MyImageUnavailable
//	case AWS_SNAPSHOT_STATE_COMPLETE:
//		returnStatus = irs.MyImageAvailable // 이것만 available 나머지는 unavailable
//	case AWS_SNAPSHOT_STATE_RECOVERABLE:
//		returnStatus = irs.MyImageUnavailable
//	case AWS_SNAPSHOT_STATE_RECOVERING:
//		returnStatus = irs.MyImageUnavailable
//	case AWS_SNAPSHOT_STATE_ERROR:
//		returnStatus = irs.MyImageUnavailable
//	}
//	return returnStatus
//}
//
//func convertAWSSnapShopToMyImageInfo(snapshot *ec2.Snapshot) (irs.MyImageInfo, error) {
//	returnMyImage := irs.MyImageInfo{}
//
//	returnMyImage.IId = irs.IID{SystemId: *snapshot.SnapshotId}
//
//	tags := snapshot.Tags
//
//	//nameTagValue := ""
//	sourceVMTag := ""
//	for _, tag := range tags {
//		//if strings.EqualFold(*tag.Key, IMAGE_TAG_DEFAULT) {
//		//	nameTagValue = *tag.Value
//		//}
//		if strings.EqualFold(*tag.Key, IMAGE_TAG_SOURCE_VM) {
//			sourceVMTag = *tag.Value
//		}
//	}
//	returnMyImage.SourceVM = irs.IID{SystemId: sourceVMTag}
//	returnMyImage.Status = convertSnapshotStateToMyImageStatus(snapshot.State)
//
//	returnMyImage.CreatedTime = *snapshot.StartTime
//
//	//keyValueList := []irs.KeyValue{}
//	//keyValueList = append(keyValueList, irs.KeyValue{Key: "architecture", Value: *awsImage.Architecture})
//	//keyValueList = append(keyValueList, irs.KeyValue{Key: "imageLocation", Value: *awsImage.ImageLocation})
//
//	return returnMyImage, nil
//}

////------ Snapshot to create a MyImage

func (ImageHandler *AwsMyImageHandler) SnapshotVM(snapshotReqInfo irs.MyImageInfo) (irs.MyImageInfo, error) {
	//Instance 정보 조회
	instance, err := DescribeInstanceById(ImageHandler.Client, snapshotReqInfo.SourceVM)
	if err != nil {
		return irs.MyImageInfo{}, err
	}

	//대상 block Device 목록 추출
	blockDeviceMappingList := []*ec2.BlockDeviceMapping{}
	for _, instanceBlockDevice := range instance.BlockDeviceMappings {
		blockDeviceMapping := ec2.BlockDeviceMapping{}
		ebs := ec2.EbsBlockDevice{}

		volume, err := DescribeVolumneById(ImageHandler.Client, *instanceBlockDevice.Ebs.VolumeId)
		if err != nil {
			return irs.MyImageInfo{}, err
		}

		ebs.VolumeSize = volume.Size
		blockDeviceMapping.DeviceName = instanceBlockDevice.DeviceName
		blockDeviceMapping.Ebs = &ebs

		blockDeviceMappingList = append(blockDeviceMappingList, &blockDeviceMapping)
	}

	var tags []*ec2.Tag
	nameTag := &ec2.Tag{
		Key:   aws.String(IMAGE_TAG_DEFAULT),
		Value: aws.String(snapshotReqInfo.IId.NameId),
	}
	tags = append(tags, nameTag)
	sourceVMTag := &ec2.Tag{
		Key:   aws.String(IMAGE_TAG_SOURCE_VM),
		Value: aws.String(snapshotReqInfo.SourceVM.SystemId),
	}
	tags = append(tags, sourceVMTag)
	tagSpec := &ec2.TagSpecification{
		ResourceType: aws.String(RESOURCE_TYPE_MYIMAGE),
		Tags:         tags,
	}
	var tagSpecs []*ec2.TagSpecification
	tagSpecs = append(tagSpecs, tagSpec)

	// Image parameter set

	input := &ec2.CreateImageInput{
		//BlockDeviceMappings: []*ec2.BlockDeviceMapping{
		//	{
		//		DeviceName: aws.String("/dev/sdh"),
		//		Ebs: &ec2.EbsBlockDevice{
		//			VolumeSize: aws.Int64(100),
		//		},
		//	},
		//	{
		//		DeviceName:  aws.String("/dev/sdc"),
		//		VirtualName: aws.String("ephemeral1"),
		//	},
		//},
		//Description: aws.String("An AMI for my server"),
		//InstanceId:  aws.String("i-1234567890abcdef0"),
		//Name:        aws.String("My server"),
		//NoReboot:    aws.Bool(true),
		Name:                aws.String(snapshotReqInfo.IId.NameId),
		InstanceId:          aws.String(snapshotReqInfo.SourceVM.SystemId),
		Description:         aws.String(snapshotReqInfo.IId.NameId),
		BlockDeviceMappings: blockDeviceMappingList,
		TagSpecifications:   tagSpecs,
	}

	if cblogger.Level.String() == "debug" {
		spew.Dump(input)
	}

	result, err := ImageHandler.Client.CreateImage(input)
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
		return irs.MyImageInfo{}, err
	}

	createdImageId := result.ImageId

	myImage, err := ImageHandler.GetMyImage(irs.IID{SystemId: *createdImageId})

	return myImage, nil
}

//------ MyImage Management

func (ImageHandler *AwsMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
	var returnMyImageInfoList []*irs.MyImageInfo
	//input := &ec2.DescribeImagesInput{}
	//
	//result, err := ImageHandler.Client.DescribeImages(input)
	//result, err := DescribeImages(ImageHandler.Client, nil)
	result, err := DescribeImages(ImageHandler.Client, nil, []*string{aws.String("self")})

	if err != nil {
		return nil, err
	}

	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}
	for _, awsImage := range result.Images {
		myImage, err := convertAWSImageToMyImageInfo(awsImage)
		if err != nil {
			// conver Error but continue;
			//return nil, err
		}
		returnMyImageInfoList = append(returnMyImageInfoList, &myImage)
	}

	return returnMyImageInfoList, nil
}

func (ImageHandler *AwsMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {
	resultImage, err := DescribeImageById(ImageHandler.Client, &myImageIID, []*string{aws.String("self")})
	if err != nil {
		return irs.MyImageInfo{}, err
	}

	returnMyImage, err := convertAWSImageToMyImageInfo(resultImage)
	return returnMyImage, nil
}

func (ImageHandler *AwsMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
	resultImage, err := DescribeImageById(ImageHandler.Client, &myImageIID, []*string{aws.String("self")})
	if err != nil {
		return false, err
	}

	snapshotIds, err := GetSnapshotIdFromEc2Image(resultImage)
	if err != nil {
		return false, err
	}

	diskIIDs, err := GetDisksFromEc2Image(resultImage)
	if err != nil {
		return false, err
	}
	if cblogger.Level.String() == "debug" {
		spew.Dump(diskIIDs)
	}

	input := &ec2.DeregisterImageInput{}
	input.ImageId = aws.String(myImageIID.SystemId)

	result, err := ImageHandler.Client.DeregisterImage(input)
	if err != nil {
		return false, err
	}

	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	for _, snapshotId := range snapshotIds {
		snapShotDeleteResult, err := ImageHandler.DeleteSnapshotById(snapshotId)
		if err != nil {
			return snapShotDeleteResult, errors.New("Fail to delete snapshot" + snapshotId + " " + err.Error())
		}
	}
	return true, nil
}

// AWS Image state 를 CB-SPIDER MyImage 의 statuf 로 변환
func convertImageStateToMyImageStatus(awsImageState *string) irs.MyImageStatus {
	var returnStatus irs.MyImageStatus

	switch *awsImageState {
	case AWS_IMAGE_STATE_PENDING:
		returnStatus = irs.MyImageUnavailable
	case AWS_IMAGE_STATE_AVAILABLE:
		returnStatus = irs.MyImageAvailable // 이것만 available 나머지는 unavailable
	case AWS_IMAGE_STATE_INVAILABLE:
		returnStatus = irs.MyImageUnavailable
	case AWS_IMAGE_STATE_DEREGISTERED:
		returnStatus = irs.MyImageUnavailable
	case AWS_IMAGE_STATE_TRANSIENT:
		returnStatus = irs.MyImageUnavailable
	case AWS_IMAGE_STATE_FAILED:
		returnStatus = irs.MyImageUnavailable
	case AWS_IMAGE_STATE_ERROR:
		returnStatus = irs.MyImageUnavailable
	}
	return returnStatus
}

func convertAWSImageToMyImageInfo(awsImage *ec2.Image) (irs.MyImageInfo, error) {
	returnMyImage := irs.MyImageInfo{}

	returnMyImage.IId = irs.IID{SystemId: *awsImage.ImageId}

	tags := awsImage.Tags

	//nameTagValue := ""
	sourceVMTag := ""
	for _, tag := range tags {
		//if strings.EqualFold(*tag.Key, IMAGE_TAG_DEFAULT) {
		//	nameTagValue = *tag.Value
		//}
		if strings.EqualFold(*tag.Key, IMAGE_TAG_SOURCE_VM) {
			sourceVMTag = *tag.Value
		}
	}
	returnMyImage.SourceVM = irs.IID{SystemId: sourceVMTag}
	returnMyImage.Status = convertImageStateToMyImageStatus(awsImage.State)

	createdTime, _ := time.Parse(
		time.RFC3339,
		*awsImage.CreationDate) // RFC3339형태이므로 해당 시간으로 다시 생성
	returnMyImage.CreatedTime = createdTime

	//keyValueList := []irs.KeyValue{}
	//keyValueList = append(keyValueList, irs.KeyValue{Key: "architecture", Value: *awsImage.Architecture})
	//keyValueList = append(keyValueList, irs.KeyValue{Key: "imageLocation", Value: *awsImage.ImageLocation})
	//keyValueList = append(keyValueList, irs.KeyValue{Key: "imageOwnerAlias", Value: *awsImage.ImageOwnerAlias})
	//keyValueList = append(keyValueList, irs.KeyValue{Key: "imageOwnerId", Value: *awsImage.OwnerId})
	//keyValueList = append(keyValueList, irs.KeyValue{Key: "imageState", Value: *awsImage.State})
	//keyValueList = append(keyValueList, irs.KeyValue{Key: "imageType", Value: *awsImage.ImageType})
	//keyValueList = append(keyValueList, irs.KeyValue{Key: "isPublic", Value: strconv.FormatBool(*awsImage.Public)})
	//keyValueList = append(keyValueList, irs.KeyValue{Key: "rootDeviceName", Value: *awsImage.RootDeviceName})
	//keyValueList = append(keyValueList, irs.KeyValue{Key: "rootDeviceType", Value: *awsImage.RootDeviceType})
	//keyValueList = append(keyValueList, irs.KeyValue{Key: "virtualizationType", Value: *awsImage.VirtualizationType})
	//
	//returnMyImage.KeyValueList = keyValueList

	//architecture			The architecture of the image. Type: String Valid Values: i386 | x86_64 | arm64 | x86_64_mac
	//blockDeviceMapping	Any block device mapping entries.
	//bootMode 				The boot mode of the image. For more information, see Boot modes in the Amazon Elastic Compute Cloud User Guide.Type: String	Valid Values: legacy-bios | uefi
	//deprecationTime
	//description
	//enaSupport			Specifies whether enhanced networking with ENA is enabled.		Type: Boolean
	//hypervisor			The hypervisor type of the image.		Type: String	Valid Values: ovm | xen
	//imageId				The ID of the AMI.
	//imageLocation			The location of the AMI.
	//imageOwnerAlias		The AWS account alias (for example, amazon, self) or the AWS account ID of the AMI owner.
	//imageOwnerId			The ID of the AWS account that owns the image.
	//imageState			Valid Values: pending | available | invalid | deregistered | transient | failed | error
	//imageType				The type of image.	Type: String	Valid Values: machine | kernel | ramdisk
	//isPublic				Indicates whether the image has public launch permissions. The value is true if this image has public launch permissions or false if it has only implicit and explicit launch permissions.
	//kernelId				The kernel associated with the image, if any. Only applicable for machine images.
	//platform				This value is set to windows for Windows AMIs; otherwise, it is blank.		Type: String	Valid Values: Windows
	//platformDetails		The platform details associated with the billing code of the AMI. For more information, see Understanding AMI billing in the Amazon Elastic Compute Cloud User Guide.
	//productCodes			Any product codes associated with the AMI.
	//ramdiskId				The RAM disk associated with the image, if any. Only applicable for machine images.
	//rootDeviceName		The device name of the root device volume (for example, /dev/sda1).
	//rootDeviceType		The type of root device used by the AMI. The AMI can use an Amazon EBS volume or an instance store volume.	Valid Values: ebs | instance-store
	//sriovNetSupport		Specifies whether enhanced networking with the Intel 82599 Virtual Function interface is enabled.
	//stateReason			The reason for the state change.	Type: StateReason object
	//tagSet
	//tpmSupport			If the image is configured for NitroTPM support, the value is v2.0. For more information, see NitroTPM in the Amazon Elastic Compute Cloud User Guide.	Type: StringValid Values: v2.0
	//usageOperation		The operation of the Amazon EC2 instance and the billing code that is associated with the AMI. usageOperation corresponds to the lineitem/Operation column on your AWS Cost and Usage Report and in the AWS Price List API. You can view these fields on the Instances or AMIs pages in the Amazon EC2 console, or in the responses that are returned by the DescribeImages command in the Amazon EC2 API, or the describe-images command in the AWS CLI.
	//virtualizationType	The type of virtualization of the AMI.	Type: String	Valid Values: hvm | paravirtual

	return returnMyImage, nil
}

// Image에 대한 snap 삭제
func (MyImageHandler *AwsMyImageHandler) DeleteSnapshotById(snapshotId string) (bool, error) {
	cblogger.Info("DeleteSnapshotBySnapshots -------------")
	// result, err := DescribeVolumnesBySnapshot(MyImageHandler.Client, snapshotIIDs)
	// if err != nil {
	// 	return false, err
	// }
	// spew.Dump(result)

	input := &ec2.DeleteSnapshotInput{
		SnapshotId: aws.String(snapshotId),
	}

	result, err := MyImageHandler.Client.DeleteSnapshot(input)
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

	return true, nil
}

func (ImageHandler *AwsMyImageHandler) CheckWindowsImage(myImageIID irs.IID) (bool, error) {
	isWindowsImage := false
	myImage := []*string{aws.String("self")}

	// image 조회 : myImage []*string{aws.String("self")} / public image []*string{aws.String("amazon")}
	resultImage, err := DescribeImageById(ImageHandler.Client, &myImageIID, myImage)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return false, err
	}

	// image에서 OsType 추출
	guestOS := GetOsTypeFromEc2Image(resultImage)
	cblogger.Debugf("imgInfo.GuestOS : [%s]", guestOS)
	if strings.Contains(strings.ToUpper(guestOS), "WINDOWS") {
		isWindowsImage = true
	}

	return isWindowsImage, nil

}
