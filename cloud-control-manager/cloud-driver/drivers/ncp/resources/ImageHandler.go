// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP Image Handler
//
// by ETRI, 2020.09.
// Updated by ETRI, 2025.02.

package resources

import (
	// "errors"
	"fmt"
	"strings"

	// "github.com/davecgh/go-spew/spew"

	ncloud "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	server "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/server"

	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
)

type NcpImageHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VMClient       *server.APIClient
}

var cblogger2 *logrus.Logger

func init() {
	cblogger2 = cblog.GetLogger("NCP ImageHandler") // cblog is a global variable.
}

func (imageHandler *NcpImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
	cblogger.Info("NCP Classic Cloud Driver: called GetImage()!!")
	InitLog()
	callLogInfo := GetCallLogScheme(imageHandler.RegionInfo.Zone, call.VMIMAGE, imageIID.SystemId, "GetImage()")

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Image SystemId")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.ImageInfo{}, newErr
	}

	ncpImageInfo, err := imageHandler.getNcpImageInfo(imageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Image Info from NCP : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.ImageInfo{}, newErr
	}
	imageInfo := mappingImageInfo(*ncpImageInfo)
	return imageInfo, nil
}

func (imageHandler *NcpImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	cblogger.Info("NCP Classic Cloud Driver: called ListImage()!")
	InitLog()
	callLogInfo := GetCallLogScheme(imageHandler.RegionInfo.Zone, call.VMIMAGE, "ListImage()", "ListImage()")

	vmHandler := NcpVMHandler{
		RegionInfo: imageHandler.RegionInfo,
		VMClient:   imageHandler.VMClient,
	}
	regionNo, err := vmHandler.getRegionNo(imageHandler.RegionInfo.Region)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP Region No of the Region Code: [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	imageReq := server.GetServerImageProductListRequest{
		ProductCode: nil,
		RegionNo:    regionNo, // CAUTION!! : When searching image Info by RegionNo
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

	if len(result.ProductList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any Image Info.")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	} else {
		cblogger.Info("Succeeded in Getting NCP Image list.")
	}

	var vmImageList []*irs.ImageInfo
	for _, image := range result.ProductList {
		imageInfo := mappingImageInfo(*image)
		vmImageList = append(vmImageList, &imageInfo)
	}
	cblogger.Info("# Supported Image Product count : ", len(vmImageList))
	return vmImageList, nil
}

func mappingImageInfo(serverImage server.Product) irs.ImageInfo {
	cblogger.Info("NCP Classic Cloud Driver: called mappingImageInfo()!")

	var osPlatform irs.OSPlatform
	if serverImage.ProductType != nil {
		if serverImage.ProductType.Code != nil {
			if strings.EqualFold(*serverImage.ProductType.Code, "WINNT") {
				osPlatform = irs.Windows
			} else {
				osPlatform = irs.PlatformNA
			}
		} else {
			osPlatform = irs.PlatformNA
		}
	} else {
		osPlatform = irs.PlatformNA
	}

	var guestOS string
	if serverImage.OsInformation != nil {
		guestOS = *serverImage.OsInformation
	} else {
		guestOS = "NA"
	}

	var blockStorageSize string
	if *serverImage.BaseBlockStorageSize != 0 {
		blockStorageSize = irs.ConvertByteToGBInt64(*serverImage.BaseBlockStorageSize)
	} else {
		blockStorageSize = "-1"
	}

	var imageStatus irs.ImageStatus
	if serverImage.ProductCode != nil {
		imageStatus = irs.ImageAvailable
	} else {
		imageStatus = irs.ImageNA
	}

	imageInfo := irs.ImageInfo{
		IId: irs.IID{ // NOTE 주의 : serverImage.ProductCode -> ProductName 으로
			NameId:   *serverImage.ProductCode,
			SystemId: *serverImage.ProductCode,
		},

		Name:           *serverImage.ProductCode,
		OSArchitecture: irs.ArchitectureNA,
		OSPlatform:     osPlatform,
		OSDistribution: guestOS,
		OSDiskType:     "NA",
		OSDiskSizeGB:   blockStorageSize,
		ImageStatus:    imageStatus,
		KeyValueList:   irs.StructToKeyValueList(serverImage),
	}

	return imageInfo
}

func (imageHandler *NcpImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	cblogger.Info("NCP Classic Cloud Driver: called CreateImage()!")

	return irs.ImageInfo{}, fmt.Errorf("Does not support CreateImage() yet!!")
}

func (imageHandler *NcpImageHandler) CheckWindowsImage(imageIID irs.IID) (bool, error) {
	cblogger.Info("NCP Classic Cloud Driver: called CheckWindowsImage()")
	InitLog()
	callLogInfo := GetCallLogScheme(imageHandler.RegionInfo.Region, call.MYIMAGE, imageIID.SystemId, "CheckWindowsImage()")

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid myImage SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	ncpImageInfo, err := imageHandler.getNcpImageInfo(imageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Image Info from NCP : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	isWindowsImage := false
	if ncpImageInfo.ProductType != nil {
		if ncpImageInfo.ProductType.Code != nil {
			if strings.EqualFold(*ncpImageInfo.ProductType.Code, "WINNT") {
				isWindowsImage = true
			}
		}
	}
	return isWindowsImage, nil
}

func (imageHandler *NcpImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	cblogger.Info("NCP Classic Cloud Driver: called DeleteImage()!")

	return false, fmt.Errorf("Does not support DeleteImage() yet!!")
}

func (imageHandler *NcpImageHandler) getNcpImageInfo(imageIID irs.IID) (*server.Product, error) {
	cblogger.Info("NCP Classic Cloud Driver: called getNcpImageInfo()!!")
	InitLog()
	callLogInfo := GetCallLogScheme(imageHandler.RegionInfo.Zone, call.VMIMAGE, imageIID.SystemId, "getNcpImageInfo()")

	if strings.EqualFold(imageIID.SystemId, "") {
		createErr := fmt.Errorf("Invalid Image SystemId")
		cblogger.Error(createErr.Error())
		LoggingError(callLogInfo, createErr)
		return nil, createErr
	}

	// cblogger.Info("\n\n### imageIID : ")
	// spew.Dump(imageIID)
	// cblogger.Info("\n")

	vmHandler := NcpVMHandler{
		RegionInfo: imageHandler.RegionInfo,
		VMClient:   imageHandler.VMClient,
	}
	regionNo, err := vmHandler.getRegionNo(imageHandler.RegionInfo.Region)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP Region No of the Region Code : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	imageReq := server.GetServerImageProductListRequest{
		ProductCode: ncloud.String(imageIID.SystemId),
		RegionNo:    regionNo,
	}
	callLogStart := call.Start()
	result, err := imageHandler.VMClient.V2Api.GetServerImageProductList(&imageReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Image list from NCP : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.ProductList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any Image info with the SystemId : [%s]", imageIID.SystemId)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	cblogger.Info("Succeeded in Getting NCP Image info.")
	return result.ProductList[0], nil
}

func (imageHandler *NcpImageHandler) isPublicImage(imageIID irs.IID) (bool, error) {
	cblogger.Info("NCP Classic Cloud Driver: called isPublicImage()")
	InitLog()
	callLogInfo := GetCallLogScheme(imageHandler.RegionInfo.Region, call.MYIMAGE, imageIID.SystemId, "isPublicImage()") // HisCall logging

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Image SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	ncpImageInfo, err := imageHandler.getNcpImageInfo(imageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Image Info from NCP : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, nil // Caution!!
	} else {
		isPublicImage := false
		if strings.EqualFold(*ncpImageInfo.ProductCode, imageIID.SystemId) {
			isPublicImage = true
		}
		return isPublicImage, nil
	}
}
