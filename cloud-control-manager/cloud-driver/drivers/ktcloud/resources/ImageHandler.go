// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// KT Cloud Image Handler
//
// by ETRI, 2021.05.
// Updated by ETRI, 2025.02.

package resources

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	// "github.com/davecgh/go-spew/spew"

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
	// cblogger.Infof("KT Cloud image ID(Templateid) : [%s]", imageIID.SystemId)
	// cblogger.Info("imageHandler.RegionInfo.Zone : ", zoneId)

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Image SystemId!!")
		cblogger.Error(newErr.Error())
		return irs.ImageInfo{}, newErr
	}

	// Note!!) Use ListImage() to search within the organized image list information
	imageListResult, err := imageHandler.ListImage()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VMSpec info list!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.ImageInfo{}, newErr
	}

	for _, image := range imageListResult {
		if strings.EqualFold(image.Name, imageIID.SystemId) {
			return *image, nil
		}
	}
	return irs.ImageInfo{}, errors.New("Failed to find the VM Image info : '" + imageIID.SystemId)
}

func (imageHandler *KtCloudImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called ListImage()!")

	result, err := imageHandler.Client.ListAvailableProductTypes(imageHandler.RegionInfo.Zone)
	if err != nil {
		cblogger.Error("Failed to Get List of Available Product Types: %s", err)
		return []*irs.ImageInfo{}, err
	}
	// spew.Dump(result)

	if len(result.Listavailableproducttypesresponse.ProductTypes) < 1 {
		return []*irs.ImageInfo{}, errors.New("Failed to Find Product Types!!")
	}

	// ### In order to remove the list of identical duplicates over and over again
	var vmImageInfoMap = make(map[string]*irs.ImageInfo) // Map to track unique VMSpec Info.
	for _, productType := range result.Listavailableproducttypesresponse.ProductTypes {
		// ### Caution!!) If the diskofferingid value exists, additional data disks are created.(=> So Not include to image list for 'Correct RootDiskSize')
		if strings.EqualFold(productType.DiskOfferingId, "") {
			imageInfo := mappingImageInfo(&productType)
			if _, exists := vmImageInfoMap[imageInfo.Name]; exists {
				// break
			} else {
				vmImageInfoMap[imageInfo.Name] = &imageInfo
			}
		}
	}

	// Convert the map to a list
	var vmImageInfoList []*irs.ImageInfo
	for _, imageInfo := range vmImageInfoMap {
		vmImageInfoList = append(vmImageInfoList, imageInfo)
	}
	// cblogger.Info("# Supported Image Product Count : ", len(vmImageInfoList))
	return vmImageInfoList, nil
}

func mappingImageInfo(productType *ktsdk.ProductTypes) irs.ImageInfo {
	// cblogger.Info("KT Cloud Cloud Driver: called mappingImageInfo()!")
	// cblogger.Info("\n\n### productType : ")
	// spew.Dump(productType)
	// cblogger.Info("\n")

	var osPlatform irs.OSPlatform
	if productType.TemplateDesc != "" {
		if strings.Contains(productType.TemplateDesc, "WIN ") {
			osPlatform = irs.Windows
		} else {
			osPlatform = irs.Linux_UNIX
		}
	} else {
		osPlatform = irs.PlatformNA
	}

	var imageStatus irs.ImageStatus
	if productType.ProductState != "" {
		if strings.EqualFold(productType.ProductState, "available") {
			imageStatus = irs.ImageAvailable
		} else {
			imageStatus = irs.ImageUnavailable
		}
	} else {
		imageStatus = irs.ImageNA
	}

	diskSize := getImageDiskSize(productType.DiskOfferingDesc)

	imageInfo := irs.ImageInfo{
		// NOTE!! : TemplateId -> Image Name (TemplateId as Image Name)
		IId: irs.IID{NameId: productType.TemplateId, SystemId: productType.TemplateId},

		Name:           productType.TemplateId,
		OSArchitecture: irs.ArchitectureNA,
		OSPlatform:     osPlatform,
		OSDistribution: productType.TemplateDesc,
		OSDiskType:     "NA",
		OSDiskSizeInGB: diskSize,
		ImageStatus:    imageStatus,
	}

	// Since KT Cloud has different supported images for each zone, zone information is also presented.
	keyValueList := []irs.KeyValue{
		{Key: "ProductType", Value: productType.Product},
		{Key: "Zone", Value: productType.ZoneDesc},
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

	isWindowsImage := false
	productType, err := imageHandler.getKTProductType(imageIID) // In case of 'Public Image'
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the ProductType Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	} else {
		if strings.Contains(productType.TemplateDesc, "WIN") {
			isWindowsImage = true
		}
		return isWindowsImage, nil
	}
}

// # Get KT Cloud ProductType : 'Public' Image and VMSpec Info
func (imageHandler *KtCloudImageHandler) getKTProductType(imageIID irs.IID) (*ktsdk.ProductTypes, error) {
	cblogger.Info("KT Cloud cloud driver: called getKTProductType()!!")

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Image SystemId!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	// Caution!! : KT Cloud searches by 'zoneId' when searching Image info/VMSpc info.
	result, err := imageHandler.Client.ListAvailableProductTypes(imageHandler.RegionInfo.Zone)
	if err != nil {
		cblogger.Error("Failed to Get List of Available Product Types: %s", err)
		return nil, err
	}

	if len(result.Listavailableproducttypesresponse.ProductTypes) < 1 {
		return nil, errors.New("Failed to Find Any Product Type on the zone!!")
	}

	var productType ktsdk.ProductTypes
	for _, productType := range result.Listavailableproducttypesresponse.ProductTypes {
		if productType.TemplateId == imageIID.SystemId { // Not productType.ProductId
			productType = productType
			return &productType, nil
		}
	}

	if productType.ProductId == "" {
		newErr := fmt.Errorf("Failed to Find any ProductType with the Image ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	return nil, nil
}

func getImageDiskSize(sizeGB string) string {
	// sizeGB Ex : "100GB"
	re := regexp.MustCompile(`(\d+)GB`)
	matches := re.FindStringSubmatch(sizeGB) // Find the match

	var diskSize string
	if len(matches) > 1 {
		diskSize = matches[1] // Extract only the numeric part
	}
	if strings.EqualFold(diskSize, "") {
		diskSize = "-1"
	}
	return diskSize // diskSize Ex : "100"
}
