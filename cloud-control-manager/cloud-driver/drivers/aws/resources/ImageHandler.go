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
	"strconv"
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

	return irs.ImageInfo{imageReqInfo.IId, "", "", nil}, nil
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
	//spew.Dump(result)	//출력 정보가 너무 많아서 생략

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
		//spew.Dump(cur)
		if reflect.ValueOf(cur.State).IsNil() {
			cblogger.Errorf("===>[%s] AMI는 State 정보가 없어서 Skip함.", *cur.ImageId)
			continue
		}

		if reflect.ValueOf(cur.Name).IsNil() {
			cblogger.Infof("===>[%s] AMI는 Name 정보가 없어서 Skip함.", *cur.ImageId)
			continue
		}

		cblogger.Debugf("[%s] - [%s] - [%s] AMI 정보 처리", *cur.ImageId, *cur.State, *cur.Name)
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

	cblogger.Info("%d개의 이미지가 조회됨.", cnt)
	//spew.Dump(imageInfoList)

	return imageInfoList, nil
}

// Image 정보를 추출함
// @TODO : GuestOS 쳌크할 것
func ExtractImageDescribeInfo(image *ec2.Image) irs.ImageInfo {
	//spew.Dump(image)
	imageInfo := irs.ImageInfo{
		//IId: irs.IID{*image.Name, *image.ImageId},
		IId: irs.IID{*image.ImageId, *image.ImageId},
		//Id:     *image.ImageId,
		//Name:   *image.Name,
		Status: *image.State,
	}

	keyValueList := []irs.KeyValue{
		//{Key: "Name", Value: *image.Name}, //20200723-Name이 없는 이미지 존재 - 예)ami-0008a301
		{Key: "CreationDate", Value: *image.CreationDate},
		{Key: "Architecture", Value: *image.Architecture}, //x86_64
		{Key: "OwnerId", Value: *image.OwnerId},
		{Key: "ImageType", Value: *image.ImageType},
		{Key: "ImageLocation", Value: *image.ImageLocation},
		{Key: "VirtualizationType", Value: *image.VirtualizationType},
		{Key: "Public", Value: strconv.FormatBool(*image.Public)},
	}

	//주로 윈도우즈는 Platform 정보가 존재하며 리눅스 계열은 PlatformDetails만 존재하는 듯. - "Linux/UNIX"
	//윈도우즈 계열은 PlatformDetails에는 "Windows with SQL Server Standard"처럼 SQL정보도 포함되어있음.
	if !reflect.ValueOf(image.Platform).IsNil() {
		imageInfo.GuestOS = *image.Platform //Linux/UNIX
		keyValueList = append(keyValueList, irs.KeyValue{Key: "Platform", Value: *image.Platform})
	} else {
		// Platform 정보가 없는 경우 PlatformDetails 정보가 존재하면 PlatformDetails 값을 이용함.
		if !reflect.ValueOf(image.PlatformDetails).IsNil() {
			imageInfo.GuestOS = *image.PlatformDetails //Linux/UNIX
		}
	}

	// 일부 이미지들은 아래 정보가 없어서 예외 처리 함.
	if !reflect.ValueOf(image.PlatformDetails).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "PlatformDetails", Value: *image.PlatformDetails})
	}

	// 일부 이미지들은 아래 정보가 없어서 예외 처리 함.
	if !reflect.ValueOf(image.Name).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "Name", Value: *image.Name})
	}
	if !reflect.ValueOf(image.Description).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "Description", Value: *image.Description})
	}
	if !reflect.ValueOf(image.ImageOwnerAlias).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "ImageOwnerAlias", Value: *image.ImageOwnerAlias})
	}
	if !reflect.ValueOf(image.RootDeviceName).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "RootDeviceName", Value: *image.RootDeviceName})
		keyValueList = append(keyValueList, irs.KeyValue{Key: "RootDeviceType", Value: *image.RootDeviceType})
	}
	if !reflect.ValueOf(image.EnaSupport).IsNil() {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "EnaSupport", Value: strconv.FormatBool(*image.EnaSupport)})
	}

	imageInfo.KeyValueList = keyValueList

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
	//spew.Dump(result)
	cblogger.Info(result)

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
