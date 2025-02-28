// Cloud Driver Interface of CB-Spider.  // The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2019.06.

package resources

import (
	"errors"
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	//irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
)

type AwsImageHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

// @TODO : 작업해야 함.
func (imageHandler *AwsImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   imageHandler.Region.Zone,
		ResourceType: call.VMIMAGE,
		ResourceName: imageReqInfo.IId.NameId,
		CloudOSAPI:   "-",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	imageReqInfo.IId.SystemId = imageReqInfo.IId.NameId

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))

	return irs.ImageInfo{
		IId:            imageReqInfo.IId,
		Name:           "default-image-name",
		OSArchitecture: "x86_64",
		OSPlatform:     "Linux/UNIX",
		OSDistribution: "Ubuntu 18.04",
		OSDiskType:     "gp3",
		OSDiskSizeGB:   "35",
		ImageStatus:    "Available",
		KeyValueList:   nil,
	}, nil

}

// @TODO : 목록이 너무 많기 때문에 amazon 계정으로 공유된 퍼블릭 이미지중 AMI만 조회 함.
// 20210607 - Tumblebug에서 필터할 수 있도록 state는 모든 이미지를 대상으로 하며, 이미지가 너무 많기 때문에 AWS 소유의 이미지만 제공 함.
func (imageHandler *AwsImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	cblogger.Debug("Start")
	var imageInfoList []*irs.ImageInfo
	input := &ec2.DescribeImagesInput{
		//ImageIds: []*string{aws.String("ami-0d097db2fb6e0f05e")},
		Owners: []*string{
			aws.String("amazon"), //사용자 계정 번호를 넣으면 사용자의 이미지를 대상으로 조회 함.
		},
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("image-type"),
				Values: aws.StringSlice([]string{"machine"}),
			},
			{
				Name:   aws.String("is-public"),
				Values: aws.StringSlice([]string{"true"}),
			},
			/*
				{
					Name:   aws.String("state"),
					Values: aws.StringSlice([]string{"available"}),
				},
			*/
		},
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   imageHandler.Region.Zone,
		ResourceType: call.VMIMAGE,
		ResourceName: "ListImage()",
		CloudOSAPI:   "DescribeImages",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := imageHandler.Client.DescribeImages(input)

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

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
		return nil, err
	}
	callogger.Info(call.String(callLogInfo))

	cnt := 0
	for _, cur := range result.Images {
		if reflect.ValueOf(cur.State).IsNil() {
			cblogger.Errorf("===>[%s] AMI is skipped because it lacks State information.", *cur.ImageId)
			continue
		}

		if reflect.ValueOf(cur.Name).IsNil() {
			cblogger.Infof("===>[%s] AMI is skipped because it lacks Name information.", *cur.ImageId)
			continue
		}

		cblogger.Debugf("[%s] - [%s] - [%s] AMI State name", *cur.ImageId, *cur.State, *cur.Name)
		//cblogger.Infof("[%s] - [%s] - [%s] - [%s] AMI 정보 처리", *cur.ImageId, *cur.State, *cur.Name, *cur.UsageOperation)

		imageInfo := ExtractImageDescribeInfo(cur)
		imageInfoList = append(imageInfoList, &imageInfo)
		cnt++
		/*
			if cnt > 20 {
				break
			}
		*/
	}

	cblogger.Info("%d images retrieved.", cnt)

	return imageInfoList, nil
}

// Image 정보를 추출함
// @TODO : GuestOS 쳌크할 것
func ExtractImageDescribeInfo(image *ec2.Image) irs.ImageInfo {

	imageInfo := irs.ImageInfo{
		//IId: irs.IID{*image.Name, *image.ImageId},
		IId: irs.IID{*image.ImageId, *image.ImageId},
		//Id:     *image.ImageId,
		//Name:   *image.Name,
		//Status: *image.State,
	}

	osPlatform := extractOsPlatform(image)
	osArchitecture := extractOsArchitecture(image)
	distribution := extractOsDistribution(image)
	imageStatus := extractImageAvailability(image)

	imageDiskType := "NA"

	// 생성 될 vm에 요구되는 root disk의 type
	if image.RootDeviceType != nil {
		imageDiskType = *image.RootDeviceType
	}

	// 생성 될 vm에 요구되는 root disk에 참고할 size 는 필요하다면 ebs volume size를 참고하여 계산할 것.
	// for _, blockDevice := range image.BlockDeviceMappings {
	// 	// EBS 또는 인스턴스 스토어 볼륨
	// 	if blockDevice.Ebs != nil {
	// 		imageSize = strconv.FormatInt(*blockDevice.Ebs.VolumeSize, 10)
	// 		imageDiskType = "EBS"
	// 		break
	// 	} else {
	// 		// cblogger.Error("blockDevice: ", blockDevice)
	// 		cblogger.Error("image: ", image)
	// 		continue
	// 	}
	// }

	imageInfo.OSPlatform = osPlatform
	imageInfo.OSArchitecture = osArchitecture
	imageInfo.OSDistribution = distribution
	imageInfo.ImageStatus = imageStatus
	imageInfo.OSDiskSizeGB = "-1"
	imageInfo.OSDiskType = imageDiskType
	imageInfo.KeyValueList = irs.StructToKeyValueList(image)
	return imageInfo
}

func (imageHandler *AwsImageHandler) GetAmiImage(imageIID irs.IID) (*ec2.Image, error) {

	cblogger.Infof("imageID : [%s]", imageIID.SystemId)

	input := &ec2.DescribeImagesInput{
		ImageIds: []*string{
			aws.String(imageIID.SystemId),
		},
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   imageHandler.Region.Zone,
		ResourceType: call.VMIMAGE,
		ResourceName: imageIID.SystemId,
		CloudOSAPI:   "DescribeImages",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := imageHandler.Client.DescribeImages(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	cblogger.Debug(result)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			cblogger.Error(err.Error())
		}
		return nil, err
	}
	callogger.Info(call.String(callLogInfo))

	if len(result.Images) > 0 {
		return result.Images[0], nil
	} else {
		return nil, errors.New("Image Not Found.")
	}

}

// func (imageHandler *AwsImageHandler) GetImage(imageID string) (irs.ImageInfo, error) {
func (imageHandler *AwsImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {

	cblogger.Infof("imageID : [%s]", imageIID.SystemId)
	resultImage, err := DescribeImageById(imageHandler.Client, &imageIID, nil)

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
		return irs.ImageInfo{}, err
	}

	if resultImage != nil {
		imageInfo := ExtractImageDescribeInfo(resultImage)
		return imageInfo, nil
	} else {
		return irs.ImageInfo{}, errors.New("Image Not Found.")
	}

}

// @TODO : 삭제 API 찾아야 함.
func (imageHandler *AwsImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   imageHandler.Region.Zone,
		ResourceType: call.VMIMAGE,
		ResourceName: imageIID.SystemId,
		CloudOSAPI:   "-",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))

	return false, nil
}

// windows os 여부 return
func (imageHandler *AwsImageHandler) CheckWindowsImage(imageIID irs.IID) (bool, error) {
	isWindowsImage := false

	// image 조회 : myImage []*string{aws.String("self")} / public image []*string{aws.String("amazon")}
	resultImage, err := DescribeImageById(imageHandler.Client, &imageIID, nil)
	//amiImage, err := imageHandler.GetAmiImage(imageIID)

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

func extractOsPlatform(image *ec2.Image) irs.OSPlatform {
	var platform string

	if !reflect.ValueOf(image.Platform).IsNil() {
		platform = *image.Platform //Linux/UNIX
	} else {
		if !reflect.ValueOf(image.PlatformDetails).IsNil() {
			platform = *image.PlatformDetails //Linux/UNIX
		}
	}

	if platform == "" {
		return irs.PlatformNA
	}

	switch {
	case strings.Contains(platform, "Linux"), strings.Contains(platform, "Ubuntu"), strings.Contains(platform, "Red Hat"):
		return irs.Linux_UNIX
	case strings.Contains(platform, "Windows"), strings.Contains(platform, "windows"):
		return irs.Windows
	default:
		return irs.PlatformNA
	}
}

func extractOsArchitecture(image *ec2.Image) irs.OSArchitecture {
	arch := image.Architecture
	if arch == nil {
		return irs.ArchitectureNA
	}

	switch *arch {
	case "arm64":
		return irs.ARM64
	case "arm64_mac":
		return irs.ARM64_MAC
	case "x86_64":
		return irs.X86_64
	case "x86_64_mac":
		return irs.X86_64_MAC
	default:
		return irs.ArchitectureNA
	}
}

func extractImageAvailability(image *ec2.Image) irs.ImageStatus {
	state := image.State

	if state == nil {
		return irs.ImageNA
	}
	switch *state {
	case "available":
		return irs.ImageAvailable
	default:
		return irs.ImageUnavailable
	}
}

func extractOsDistribution(image *ec2.Image) string {
	return *image.Name
}
