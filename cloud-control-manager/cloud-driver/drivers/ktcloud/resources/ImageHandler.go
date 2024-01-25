// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// KT Cloud Image Handler
//
// by ETRI, 2021.05.

package resources

import (
	"fmt"
	"errors"
	"strings"
	//"github.com/davecgh/go-spew/spew"

	ktsdk "github.com/cloud-barista/ktcloud-sdk-go"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type KtCloudImageHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	Client         *ktsdk.KtCloudClient
}

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("KT Cloud Image Handler")
}

// <Note> 'Image' in KT Cloud API manual means an image created for Volume or Snapshot of VM stopped state.

func (imageHandler *KtCloudImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called GetImage()!!")

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Image SystemId!!")
		cblogger.Error(newErr.Error())
		return irs.ImageInfo{}, newErr
	}

	var resultImageInfo irs.ImageInfo
	zoneId := imageHandler.RegionInfo.Zone

	cblogger.Infof("KT Cloud image ID(Templateid) : [%s]", imageIID.SystemId)
	cblogger.Info("imageHandler.RegionInfo.Zone : ", zoneId)

	// Caution!! : KT Cloud searches by 'zoneId' when searching Image info/VMSpc info.
	result, err := imageHandler.Client.ListAvailableProductTypes(zoneId)
	if err != nil {
		cblogger.Error("Failed to Get List of Available Product Types: %s", err)
		return irs.ImageInfo{}, err
	}

	if len(result.Listavailableproducttypesresponse.ProductTypes) < 1 {
		return irs.ImageInfo{}, errors.New("Failed to Find Product Types!!")
	}

	for _, productType := range result.Listavailableproducttypesresponse.ProductTypes {
		cblogger.Info("# Search criteria of Image Template ID : ", imageIID.SystemId)
		if productType.TemplateId == imageIID.SystemId {
			resultImageInfo = MappingImageInfo(productType)
			break
		}
	}
	return resultImageInfo, nil
}

func (imageHandler *KtCloudImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called ListImage()!")
	
	var vmImageList []*irs.ImageInfo
	zoneId := imageHandler.RegionInfo.Zone
	cblogger.Info("RegionInfo.Zone : ", zoneId)

	result, err := imageHandler.Client.ListAvailableProductTypes(zoneId)
	if err != nil {
		cblogger.Error("Failed to Get List of Available Product Types: %s", err)
		return []*irs.ImageInfo{}, err
	}
	// spew.Dump(result)

	if len(result.Listavailableproducttypesresponse.ProductTypes) < 1 {
		return []*irs.ImageInfo{}, errors.New("Failed to Find Product Types!!")
	}

	// ### 아래는 사용자가 생성한 image만 해당함. ###
	// //templatefilter := "selfexecutable"
	// templatefilter := "self"
	// listall := true

	// result, err := imageHandler.Client.ListTemplates(templatefilter, "", listall)
	// if err != nil {
	// 	cblogger.Error("Error listing images: %s", err)

	// 	return []*irs.ImageInfo{}, err
	// }

	// spew.Dump(result)

	// if len(result.Listtemplatesresponse.Template) < 1 {
	// 	return []*irs.ImageInfo{}, errors.New("Image list Not found!!")
	// }


	// ### Without Deduplication of Templates
	// for _, productType := range result.Listavailableproducttypesresponse.Producttypes {
	// 	var serverProductType ktsdk.Producttypes
	// 	serverProductType = productType

	// 	imageInfo := MappingImageInfo(serverProductType)
	// 	vmImageList = append(vmImageList, &imageInfo)
	// }	

	// ### In order to remove the list of identical duplicates over and over again
	tempID := ""
	for _, productType := range result.Listavailableproducttypesresponse.ProductTypes {
	//	if (tempID == "") || (productType.Templateid != tempID) { 
		if productType.TemplateId != tempID { 
			imageInfo := MappingImageInfo(productType)
			vmImageList = append(vmImageList, &imageInfo)

			tempID = productType.TemplateId
			cblogger.Infof("\nImage Template Id : " + tempID)
		}
	}
	cblogger.Info("# Supported Image Product Count : ", len(vmImageList))
	return vmImageList, nil
}

func MappingImageInfo(ktServerProductType ktsdk.ProductTypes) irs.ImageInfo {
	cblogger.Info("KT Cloud Cloud Driver: called MappingImageInfo()!")
	imageInfo := irs.ImageInfo{
		// NOTE!! : TemplateId -> Image Name (TemplateId as Image Name)
		IId: 		irs.IID{ktServerProductType.TemplateId, ktServerProductType.TemplateId},
		GuestOS: 	ktServerProductType.TemplateDesc,
		Status: 	ktServerProductType.ProductState,
	}

	// Since KT Cloud has different supported images for each zone, zone information is also presented.
	keyValueList := []irs.KeyValue{
		{Key: "Zone", Value: ktServerProductType.ZoneDesc},	
	}
	imageInfo.KeyValueList = keyValueList
	return imageInfo
}

func (imageHandler *KtCloudImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	cblogger.Info("KT Cloud Cloud Driver: called CreateImage()!")

	return irs.ImageInfo{}, errors.New("Does not support CreateImage() yet!!")
}

func (imageHandler *KtCloudImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud Cloud Driver: called DeleteImage()!")

	return true, errors.New("Does not support DeleteImage() yet!!")
}

func (imageHandler *KtCloudImageHandler) CheckWindowsImage(imageIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud Driver: called CheckWindowsImage()")

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Image SystemId!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	imageInfo, err := imageHandler.GetImage(imageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Image Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}
	
	isWindowsImage := false
	if strings.Contains(imageInfo.GuestOS, "WIN") {
		isWindowsImage = true
	}
	return isWindowsImage, nil	
}
