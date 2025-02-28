// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP VPC Image Handler
//
// by ETRI, 2020.12.
// Updated by ETRI, 2025.01.

package resources

import (
	// "errors"
	"fmt"
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
	InitLog()
	callLogInfo := GetCallLogScheme(imageHandler.RegionInfo.Zone, call.VMIMAGE, imageIID.SystemId, "GetImage()")

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Image SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.ImageInfo{}, newErr
	}

	ncpImage, err := imageHandler.getNcpVpcImage(imageIID.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Image Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.ImageInfo{}, newErr
	}
	return mappingImageInfo(ncpImage), nil
}

func (imageHandler *NcpVpcImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	cblogger.Info("NCP VPC Cloud driver: called ListImage()!")
	InitLog()
	callLogInfo := GetCallLogScheme(imageHandler.RegionInfo.Zone, call.VMIMAGE, "ListImage()", "ListImage()")

	// // Search NCP VPC All Region
	// if len(imageHandler.RegionInfo.Region) > 0 {
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

	imageReq := vserver.GetServerImageListRequest{
		RegionCode:              ncloud.String(imageHandler.RegionInfo.Region), // CAUTION!! : Searching VM Image Info by RegionCode (Not RegionNo)
		ServerImageTypeCodeList: []*string{ncloud.String("NCP")},               // Options: SELF | NCP
	}

	callLogStart := call.Start()
	result, err := imageHandler.VMClient.V2Api.GetServerImageList(&imageReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Image list from NCP VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var vmImageList []*irs.ImageInfo
	if len(result.ServerImageList) < 1 {
		cblogger.Info("### Image info does Not Exist!!")
		return nil, nil
	} else {
		// cblogger.Info("Succeeded in Getting NCP VPC Image list.")
		for _, image := range result.ServerImageList {
			imageInfo := mappingImageInfo(image)
			vmImageList = append(vmImageList, &imageInfo)
		}
	}
	// cblogger.Infof("# Supported image count : [%d]", len(vmImageList))
	return vmImageList, nil
}

func mappingImageInfo(serverImage *vserver.ServerImage) irs.ImageInfo {
	cblogger.Infof("Mapping Image Info!! ")
	// spew.Dump(serverImage)

	var architectureType irs.OSArchitecture
	if serverImage.CpuArchitectureType != nil {
		if serverImage.CpuArchitectureType.Code != nil {
			if strings.EqualFold(*serverImage.CpuArchitectureType.Code, "arm64") {
				architectureType = irs.ARM64
			} else if strings.EqualFold(*serverImage.CpuArchitectureType.Code, "arm64_mac") {
				architectureType = irs.ARM64_MAC
			} else if strings.EqualFold(*serverImage.CpuArchitectureType.Code, "x86_64") {
				architectureType = irs.X86_64
			} else if strings.EqualFold(*serverImage.CpuArchitectureType.Code, "x86_64_mac") {
				architectureType = irs.X86_64_MAC
			}
		} else {
			architectureType = irs.ArchitectureNA
		}
	} else {
		architectureType = irs.ArchitectureNA
	}

	var osPlatform irs.OSPlatform
	if serverImage.OsCategoryType != nil {
		if serverImage.OsCategoryType.CodeName != nil {
			if strings.EqualFold(*serverImage.OsCategoryType.CodeName, "LINUX") {
				osPlatform = irs.Linux_UNIX
			} else {
				osPlatform = irs.Windows
			}

		} else {
			osPlatform = irs.PlatformNA
		}
	} else {
		osPlatform = irs.PlatformNA
	}

	var guestOS string
	if serverImage.ServerImageName != nil {
		guestOS = *serverImage.ServerImageName
	} else {
		guestOS = "NA"
	}

	// Note) *serverImage.ServerImageDescription => sometimes : "kernel version : 5.14.0-427.37.1.el9_4.x86_64",

	var diskType string
	if len(serverImage.BlockStorageMappingList) > 0 {
		blockStorageMapping := serverImage.BlockStorageMappingList[0]
		if blockStorageMapping.BlockStorageVolumeType != nil && blockStorageMapping.BlockStorageVolumeType.CodeName != nil {
			diskType = *blockStorageMapping.BlockStorageVolumeType.CodeName
		} else {
			diskType = "NA"
		}
	} else {
		diskType = "NA"
	}

	var blockStorageSize string
	if len(serverImage.BlockStorageMappingList) > 0 {
		blockStorageMapping := serverImage.BlockStorageMappingList[0]
		if blockStorageMapping.BlockStorageSize != nil {
			blockStorageSize = irs.ConvertByteToGBInt64(*blockStorageMapping.BlockStorageSize)
		} else {
			blockStorageSize = "-1"
		}
	} else {
		blockStorageSize = "-1"
	}

	var imageStatus irs.ImageStatus
	if serverImage.ServerImageStatusName != nil {
		if strings.EqualFold(*serverImage.ServerImageStatusName, "created") {
			imageStatus = irs.ImageAvailable
		} else {
			imageStatus = irs.ImageUnavailable
		}
	} else {
		imageStatus = irs.ImageNA
	}

	// *serverImage.ServerImageNo : numeric type
	// *serverImage.ServerImageProductCode : ex) "SW.VSVR.OS.LNX64.UBNTU.SVR22.G003"
	imageInfo := irs.ImageInfo{
		IId: irs.IID{
			NameId:   *serverImage.ServerImageNo,
			SystemId: *serverImage.ServerImageNo,
		},

		Name:           *serverImage.ServerImageNo,
		OSArchitecture: architectureType,
		OSPlatform:     osPlatform,
		OSDistribution: guestOS,
		OSDiskType:     diskType,
		OSDiskSizeGB:   blockStorageSize,
		ImageStatus:    imageStatus,
		KeyValueList:   irs.StructToKeyValueList(serverImage),
	}

	return imageInfo
}

func (imageHandler *NcpVpcImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateImage()!")

	return irs.ImageInfo{}, fmt.Errorf("Does not support CreateImage() yet!!")
}

func (imageHandler *NcpVpcImageHandler) CheckWindowsImage(imageIID irs.IID) (bool, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CheckWindowsImage()")
	InitLog()
	callLogInfo := GetCallLogScheme(imageHandler.RegionInfo.Region, call.VMIMAGE, imageIID.SystemId, "CheckWindowsImage()")

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	ncpImage, err := imageHandler.getNcpVpcImage(imageIID.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Image Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	isWindowsImage := false
	if strings.Contains(strings.ToUpper(*ncpImage.ServerImageName), "WIN") { // Ex) "win-2019-64-en", "mssql(2019std)-win-2016-64-en"
		isWindowsImage = true
	}
	return isWindowsImage, nil
}

func (imageHandler *NcpVpcImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	cblogger.Info("NCP VPC Cloud Driver: called DeleteImage()!")

	return true, fmt.Errorf("Does not support DeleteImage() yet!!")
}

func (imageHandler *NcpVpcImageHandler) isPublicImage(imageName string) (bool, error) {
	cblogger.Info("NCP VPC Cloud Driver: called isPublicImage()")
	InitLog()
	callLogInfo := GetCallLogScheme(imageHandler.RegionInfo.Region, call.VMIMAGE, imageName, "isPublicImage()") // HisCall logging

	if strings.EqualFold(imageName, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	imageInfo, err := imageHandler.getNcpVpcImage(imageName) // ServerImageTypeCode : SELF | NCP (All types of image)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Image Info : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	isPublicImage := false
	if strings.EqualFold(*imageInfo.ServerImageType.Code, "NCP") { // *imageInfo.ServerImageType.Code : SELF | NCP
		isPublicImage = true
	}
	return isPublicImage, nil
}

func (imageHandler *NcpVpcImageHandler) getNcpVpcImage(imageName string) (*vserver.ServerImage, error) {
	cblogger.Info("NCP VPC Cloud Driver: called getNcpVpcImage()!!")
	InitLog()
	callLogInfo := GetCallLogScheme(imageHandler.RegionInfo.Zone, call.VMIMAGE, imageName, "getNcpVpcImage()")

	if strings.EqualFold(imageName, "") {
		createErr := fmt.Errorf("Invalid Image Name!!")
		cblogger.Error(createErr.Error())
		LoggingError(callLogInfo, createErr)
		return nil, createErr
	}

	imageReq := vserver.GetServerImageListRequest{
		RegionCode:        ncloud.String(imageHandler.RegionInfo.Region),
		ServerImageNoList: []*string{ncloud.String(imageName)},
		// ServerImageTypeCodeList: 	[]*string{ncloud.String("NCP"),}, // <= Options: SELF | NCP. Need too include all types of image!!
	}
	callLogStart := call.Start()
	result, err := imageHandler.VMClient.V2Api.GetServerImageList(&imageReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Image list from NCP : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.ServerImageList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any Image Info from NCP!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	return result.ServerImageList[0], nil
}
