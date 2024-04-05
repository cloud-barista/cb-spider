// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, Innogrid, 2021.12.
// by ETRI 2022.08.

package resources

import (
	// "errors"
	"fmt"
	"strings"
	// "github.com/davecgh/go-spew/spew"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	images "github.com/cloud-barista/nhncloud-sdk-go/openstack/imageservice/v2/images" // imageservice/v2/images : For Visibility parameter
	// comimages "github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/images" // compute/v2/images

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NhnCloudImageHandler struct {
	RegionInfo  idrv.RegionInfo
	VMClient    *nhnsdk.ServiceClient
	ImageClient *nhnsdk.ServiceClient
}

func (imageHandler *NhnCloudImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	cblogger.Info("NHN Cloud Driver: called ListImage()")
	callLogInfo := getCallLogScheme(imageHandler.RegionInfo.Region, call.VMIMAGE, "ListImage()", "ListImage()")

	start := call.Start()
	listOpts := images.ListOpts{
		Visibility: images.ImageVisibilityPublic, // Note : Public image only
	}
	allPages, err := images.List(imageHandler.ImageClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud Image pages. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	nhnImageList, err := images.ExtractImages(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud Image List. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, start)

	// cblogger.Info("\n\n### nhnImageList : ")
	// spew.Dump(nhnImageList)
	// cblogger.Info("# 출력 결과 수 : ", len(nhnImageList))

	var imageInfoList []*irs.ImageInfo
	for _, nhnImage := range nhnImageList {
		imageInfo := imageHandler.mappingImageInfo(nhnImage)
		imageInfoList = append(imageInfoList, imageInfo)
	}
	return imageInfoList, nil
}

func (imageHandler *NhnCloudImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
	cblogger.Info("NHN Cloud Driver: called GetImage()")
	callLogInfo := getCallLogScheme(imageHandler.RegionInfo.Region, call.VMIMAGE, imageIID.SystemId, "GetImage()")

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.ImageInfo{}, newErr
	}

	start := call.Start()
	// nhnImage, err := comimages.Get(imageHandler.VMClient, imageIID.SystemId).Extract() // VM Client
	nhnImage, err := images.Get(imageHandler.ImageClient, imageIID.SystemId).Extract() // Image Client
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud Image Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.ImageInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	imageInfo := imageHandler.mappingImageInfo(*nhnImage)
	return *imageInfo, nil
}

func (imageHandler *NhnCloudImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	cblogger.Info("NHN Cloud Driver: called CreateImage()!")

	return irs.ImageInfo{}, fmt.Errorf("Does not support CreateImage() yet!!")
}

func (imageHandler *NhnCloudImageHandler) CheckWindowsImage(imageIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called CheckWindowsImage()")
	callLogInfo := getCallLogScheme(imageHandler.RegionInfo.Region, call.VMIMAGE, imageIID.SystemId, "CheckWindowsImage()")

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	start := call.Start()
	// nhnImage, err := comimages.Get(imageHandler.VMClient, imageIID.SystemId).Extract() // VM Client
	nhnImage, err := images.Get(imageHandler.ImageClient, imageIID.SystemId).Extract() // Image Client
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud Image Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, start)

	isWindowsImage := false
	if strings.Contains(nhnImage.Name, "Windows") {
		isWindowsImage = true
	}
	return isWindowsImage, nil
}

func (imageHandler *NhnCloudImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called DeleteImage()!")

	return true, fmt.Errorf("Does not support DeleteImage() yet!!")
}

func (imageHandler *NhnCloudImageHandler) mappingImageInfo(image images.Image) *irs.ImageInfo {
	cblogger.Info("NHN Cloud Driver: called mappingImagInfo()!")

	var imgAvailability string
	if strings.EqualFold(string(image.Status), "active") {
		imgAvailability = "available"
	} else {
		imgAvailability = "unavailable"
	}

	imageInfo := &irs.ImageInfo{
		IId: irs.IID{
			NameId:   image.ID, // Caution!!
			SystemId: image.ID,
		},
		GuestOS: image.Name, // Caution!!
		Status:  imgAvailability,
	}

	keyValueList := []irs.KeyValue{
		{Key: "Region", Value: imageHandler.RegionInfo.Region},
		{Key: "Visibility:", Value: string(image.Visibility)},
	}

	for key, val := range image.Properties {
		if key == "os_architecture" || key == "hypervisor_type" || key == "release_date" || key == "description" || key == "os_distro" || key == "os_version" || key == "nhncloud_product" {
			metadata := irs.KeyValue{
				Key:   strings.ToUpper(key),
				Value: fmt.Sprintf("%v", val),
			}
			keyValueList = append(keyValueList, metadata)
		}
	}

	imageInfo.KeyValueList = keyValueList
	return imageInfo
}

func (imageHandler *NhnCloudImageHandler) isPublicImage(imageIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called isPublicImage()")
	callLogInfo := getCallLogScheme(imageHandler.RegionInfo.Region, call.VMIMAGE, imageIID.SystemId, "isPublicImage()")

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	start := call.Start()
	nhnImage, err := images.Get(imageHandler.ImageClient, imageIID.SystemId).Extract() // Image Client
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud Image Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, start)

	isPublicImage := false
	if strings.EqualFold(string(nhnImage.Visibility), "public") {
		isPublicImage = true
	}
	return isPublicImage, nil
}
