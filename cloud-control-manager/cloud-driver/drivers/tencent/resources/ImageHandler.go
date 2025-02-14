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
	"strconv"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	//irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

type TencentImageHandler struct {
	Region idrv.RegionInfo
	Client *cvm.Client
}

// @TODO - 이미지 생성에 따른 구조체 정의 필요 - 현재는 IID뿐이 없어서 이미지 이름으로만 생성하도록 했음.(인스턴스Id가 없어서 에러 발생함.)
func (imageHandler *TencentImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	cblogger.Debug(imageReqInfo)

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   imageHandler.Region.Zone,
		ResourceType: call.VMIMAGE,
		ResourceName: imageReqInfo.IId.NameId,
		CloudOSAPI:   "CreateImage()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := cvm.NewCreateImageRequest()
	request.ImageName = common.StringPtr(imageReqInfo.IId.NameId)
	request.ImageDescription = common.StringPtr(imageReqInfo.IId.NameId)
	//request.InstanceId = common.StringPtr("InstanceId") //필수 - 이미지로 만들 인스턴스 Id

	//request.ForcePoweroff = common.StringPtr("ForcePoweroff")	//옵션

	// // Whether to enable Sysprep when creating a Windows image. Click here to learn more about Sysprep.
	// // https://intl.cloud.tencent.com/document/product/213/35876
	// request.Sysprep = common.StringPtr("Sysprep")

	callLogStart := call.Start()
	response, err := imageHandler.Client.CreateImage(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return irs.ImageInfo{}, err
	}
	//cblogger.Debug(response)
	cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	// imageInfo := irs.ImageInfo{
	// 	IId: irs.IID{NameId: imageReqInfo.IId.NameId, SystemId: *response.Response.ImageId},
	// }

	//OS등의 정보 확인을 위해 GetImage를 호출 함.
	imageInfo, errGetImage := imageHandler.GetImage(irs.IID{SystemId: *response.Response.ImageId})
	if errGetImage != nil {
		cblogger.Error(errGetImage)
		return irs.ImageInfo{}, errGetImage
	}
	imageInfo.IId.NameId = imageReqInfo.IId.NameId
	return imageInfo, nil
}

func (imageHandler *TencentImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	var imageInfoList []*irs.ImageInfo

	cblogger.Debug("Start")
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   imageHandler.Region.Zone,
		ResourceType: call.VMIMAGE,
		ResourceName: "ListImage()",
		CloudOSAPI:   "DescribeImages()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := cvm.NewDescribeImagesRequest()
	request.Limit = common.Uint64Ptr(100) //default : 20 / max : 100

	callLogStart := call.Start()
	response, err := imageHandler.Client.DescribeImages(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return nil, err
	}
	//cblogger.Debug(response)
	//cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	//cnt := 0
	for _, curImage := range response.Response.ImageSet {
		cblogger.Debugf("[%s] AMI information processing", *curImage.ImageId)
		imageInfo := ExtractImageDescribeInfo(curImage)
		imageInfoList = append(imageInfoList, &imageInfo)
	}

	//cblogger.Debug(imageInfoList)
	return imageInfoList, nil
}

func ExtractImageDescribeInfo(image *cvm.Image) irs.ImageInfo {
	//cblogger.Debug(image)
	imageInfo := irs.ImageInfo{
		//IId: irs.IID{*image.Name, *image.ImageId},
		IId: irs.IID{NameId: *image.ImageId, SystemId: *image.ImageId},
	}

	osPlatform := extractOsPlatform(image)
	osArchitecture := extractOsArchitecture(image)
	distribution := extractOsDistribution(image)

	imageStatus := extractImageAvailability(image)

	imageInfo.OSPlatform = osPlatform
	imageInfo.OSArchitecture = osArchitecture
	imageInfo.OSDistribution = distribution
	imageInfo.ImageStatus = imageStatus
	imageInfo.OSDiskSizeInGB = strconv.FormatInt(*image.ImageSize, 10)
	imageInfo.OSDiskType = "NA"

	//KeyValue 목록 처리
	keyValueList, errKeyValue := ConvertKeyValueList(image)
	if errKeyValue != nil {
		cblogger.Errorf("KeyValue extraction failed for [%]", *image.ImageId)
		cblogger.Error(errKeyValue)
	}

	imageInfo.KeyValueList = keyValueList
	return imageInfo
}

func (imageHandler *TencentImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
	cblogger.Infof("imageID : [%s]", imageIID.SystemId)

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   imageHandler.Region.Zone,
		ResourceType: call.VMIMAGE,
		ResourceName: imageIID.SystemId,
		CloudOSAPI:   "DescribeImages()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := cvm.NewDescribeImagesRequest()
	request.ImageIds = common.StringPtrs([]string{imageIID.SystemId})

	callLogStart := call.Start()
	response, err := imageHandler.Client.DescribeImages(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return irs.ImageInfo{}, err
	}

	//cblogger.Debug(response)
	cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	if *response.Response.TotalCount > 0 {
		imageInfo := ExtractImageDescribeInfo(response.Response.ImageSet[0])
		return imageInfo, nil
	} else {
		return irs.ImageInfo{}, errors.New("I couldn't find the information.")
	}

}

func (imageHandler *TencentImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	cblogger.Infof("imageIID : [%s]", imageIID.SystemId)

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   imageHandler.Region.Zone,
		ResourceType: call.VMIMAGE,
		ResourceName: imageIID.NameId,
		CloudOSAPI:   "DeleteImages()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := cvm.NewDeleteImagesRequest()
	request.ImageIds = common.StringPtrs([]string{imageIID.NameId})

	callLogStart := call.Start()
	response, err := imageHandler.Client.DeleteImages(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return false, err
	}
	//cblogger.Debug(response)
	cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	return true, nil
}

// windows 여부 return
// imate-type : PUBLIC_IMAGE, SHARED_IMAGE, PRIVATE_IMAGE
func (imageHandler *TencentImageHandler) CheckWindowsImage(imageIID irs.IID) (bool, error) {
	isWindow := false
	imageTypes := []string{"PUBLIC_IMAGE", "SHARED_IMAGE"}

	resultImg, err := DescribeImagesByID(imageHandler.Client, imageIID, imageTypes)
	if err != nil {
		return isWindow, err
	}

	platform := GetOsType(resultImg)
	if platform == "Windows" {
		isWindow = true
	}

	return isWindow, nil
}

func extractOsPlatform(image *cvm.Image) irs.OSPlatform {
	platform := image.Platform

	if platform == nil {
		return irs.PlatformNA
	}
	switch *platform {
	case "CentOS", "Ubuntu", "TencentOS", "OpenCloudOS", "Debian", "CentOS Stream", "Red Hat":
		return irs.Linux_UNIX
	case "Windows":
		return irs.Windows
	default:
		return irs.PlatformNA
	}
}

func extractOsDistribution(image *cvm.Image) string {
	return *image.ImageName
}

func extractImageAvailability(image *cvm.Image) irs.ImageStatus {
	state := image.ImageState

	if state == nil {
		return irs.ImageNA
	}
	switch *state {
	case "NORMAL":
		return irs.ImageAvailable
	default:
		return irs.ImageUnavailable
	}
}

func extractOsArchitecture(orgImage *cvm.Image) irs.OSArchitecture {
	arch := orgImage.Architecture
	if arch == nil {
		return irs.ArchitectureNA
	}

	// arm64
	// arm64_mac
	// x86_64
	// x86_64_mac
	// NA
	switch *arch {
	case "arm", "arm64":
		return irs.ARM64

	case "x86_64":
		return irs.X86_64

	default:
		return irs.ArchitectureNA
	}

}
