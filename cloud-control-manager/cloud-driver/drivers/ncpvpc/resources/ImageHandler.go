// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP VPC Image Handler
//
// by ETRI, 2020.12.

package resources

import (
	// "errors"
	"fmt"
	"strconv"
	"strings"
	// "github.com/davecgh/go-spew/spew"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"
	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpVpcImageHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VMClient       *vserver.APIClient
}

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP VPC ImageHandler")
}

func (imageHandler *NcpVpcImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
	cblogger.Info("NCP VPC Cloud driver: called GetImage()!!")
	cblogger.Infof("NCP VPC image ID(ImageProductCode) : [%s]", imageIID.SystemId)

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(imageHandler.RegionInfo.Zone, call.VMIMAGE, imageIID.SystemId, "GetImage()")

	if imageIID.SystemId == "" {
		createErr := fmt.Errorf("Invalid Image SystemId!!")
		cblogger.Error(createErr.Error())
		LoggingError(callLogInfo, createErr)
		return irs.ImageInfo{}, createErr
	}

	imageReq := vserver.GetServerImageProductListRequest {
		RegionCode:  ncloud.String(imageHandler.RegionInfo.Region),
		ProductCode: ncloud.String(imageIID.SystemId),
	}
	callLogStart := call.Start()
	// Image ID와 NCP VPC의 Image ProductCode를 비교해서
	result, err := imageHandler.VMClient.V2Api.GetServerImageProductList(&imageReq)
	if err != nil {
		cblogger.Error(*result.ReturnMessage)
		newErr := fmt.Errorf("Failed to Find Image list from NCP VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.ImageInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var imageInfo irs.ImageInfo
	if *result.TotalRows < 1 {
		cblogger.Info("### Image info does Not Exist!!")
	} else {
		cblogger.Info("Succeeded in Getting NCP VPC Image info.")
		imageInfo = MappingImageInfo(*result.ProductList[0])	
	}

	return imageInfo, nil
}

func (imageHandler *NcpVpcImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	cblogger.Info("NCP VPC Cloud driver: called ListImage()!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(imageHandler.RegionInfo.Zone, call.VMIMAGE, "ListImage()", "ListImage()")

	ncpVpcRegion := imageHandler.RegionInfo.Region
	cblogger.Infof("imageHandler.RegionInfo.Region : [%s}", ncpVpcRegion)

	// // Search NCP VPC All Region
	// if len(ncpVpcRegion) > 0 {
	// 	regionListReq := vserver.GetRegionListRequest{}

	// 	regionListResult, err := imageHandler.VMClient.V2Api.GetRegionList(&regionListReq)
	// 	if err != nil {
	// 		cblogger.Error(*regionListResult.ReturnMessage)
	// 		newErr := fmt.Errorf("Failed to Get Region list!! : [%v]", err)
	// 		cblogger.Error(newErr.Error())
	// 		LoggingError(callLogInfo, newErr)
	// 	}

	// 	if *regionListResult.TotalRows < 1 {
	// 		cblogger.Info("### Region info does Not Exist!!")
	// 	}  else {
	// 		cblogger.Infof("Succeeded in Getting NCP VPC Region list!! : ")
	// 	}

	// 	cblogger.Infof("# Supporting All Region (by NCP VPC) Count : [%d]", len(regionListResult.RegionList))
	// 	cblogger.Info("\n# Supporting All Region (by NCP VPC) List : ")
	// 	spew.Dump(regionListResult.RegionList)
	// }

	imageReq := vserver.GetServerImageProductListRequest{
		ProductCode:  	nil,
		RegionCode: 	&ncpVpcRegion, // CAUTION!! : Searching VM Image Info by RegionCode (Not RegionNo)
	}

	callLogStart := call.Start()
	result, err := imageHandler.VMClient.V2Api.GetServerImageProductList(&imageReq)
	if err != nil {
		cblogger.Error(*result.ReturnMessage)
		newErr := fmt.Errorf("Failed to Find Image list from NCP VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)
	
	var vmImageList []*irs.ImageInfo
	if *result.TotalRows < 1 {
		cblogger.Info("### Image info does Not Exist!!")
	} else {
		cblogger.Info("Succeeded in Getting NCP VPC Image list.")
		for _, image := range result.ProductList {
			imageInfo := MappingImageInfo(*image)
			vmImageList = append(vmImageList, &imageInfo)
		}
	}

	cblogger.Infof("# Supported image count : [%d]", len(vmImageList))
	return vmImageList, nil
}

func MappingImageInfo(serverImage vserver.Product) irs.ImageInfo {
	cblogger.Infof("Mapping Image Info!! ")

	imageInfo := irs.ImageInfo {
		// IId: irs.IID{*serverImage.ProductName, *serverImage.ProductCode},
		// NOTE 주의 : serverImage.ProductCode -> ProductName 으로
		IId: irs.IID{
			NameId: 	*serverImage.ProductCode, 
			SystemId: 	*serverImage.ProductCode,
		},
		GuestOS: *serverImage.ProductDescription,
	}

	keyValueList := []irs.KeyValue{
		{Key: "OSName(En)", Value: *serverImage.PlatformType.CodeName},
		{Key: "InfraResourceType", Value: *serverImage.InfraResourceType.CodeName},
		{Key: "BaseBlockStorageSize(GB)", Value: strconv.FormatFloat(float64(*serverImage.BaseBlockStorageSize)/(1024*1024*1024), 'f', 0, 64)},
	}

	keyValueList = append(keyValueList, irs.KeyValue{Key: "OSType", Value: *serverImage.PlatformType.Code})
	imageInfo.KeyValueList = keyValueList

	return imageInfo
}

func (imageHandler *NcpVpcImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateImage()!")

	return irs.ImageInfo{}, fmt.Errorf("Does not support CreateImage() yet!!")
}

func (imageHandler *NcpVpcImageHandler) CheckWindowsImage(imageIID irs.IID) (bool, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CheckWindowsImage()")

	InitLog()
	callLogInfo := GetCallLogScheme(imageHandler.RegionInfo.Region, call.VMIMAGE, imageIID.SystemId, "CheckWindowsImage()") // HisCall logging

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}	
	
	isWindowsImage := false
	if strings.Contains(strings.ToUpper(imageIID.SystemId), "WND") {
		isWindowsImage = true
	}
	return isWindowsImage, nil
}

func (imageHandler *NcpVpcImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	cblogger.Info("NCP VPC Cloud Driver: called DeleteImage()!")

	return true, fmt.Errorf("Does not support DeleteImage() yet!!")
}

func (imageHandler *NcpVpcImageHandler) isPublicImage(imageIID irs.IID) (bool, error) {
	cblogger.Info("NCP VPC Cloud Driver: called isPublicImage()")

	InitLog()
	callLogInfo := GetCallLogScheme(imageHandler.RegionInfo.Region, call.VMIMAGE, imageIID.SystemId, "isPublicImage()") // HisCall logging

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}	
	
	imageInfo, err := imageHandler.GetImage(imageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Image Info : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	isPublicImage := false
	if strings.EqualFold(imageInfo.IId.SystemId, imageIID.SystemId) {
		isPublicImage = true
	}
	return isPublicImage, nil
}
