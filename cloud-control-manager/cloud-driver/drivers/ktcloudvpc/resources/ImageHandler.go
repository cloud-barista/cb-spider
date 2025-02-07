// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2022.08.

package resources

import (
	"strings"
	"fmt"
	// "strconv"	
	// "github.com/davecgh/go-spew/spew"

	ktvpcsdk 	"github.com/cloud-barista/ktcloudvpc-sdk-go"

	// "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/compute/v2/images"
	images 		"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/imageservice/v2/images"  // Caution!!
		//Ref) 'Image API' return struct of image : ktcloudvpc-sdk-go/openstack/imageservice/v2/images/results.go
		//Ref) 'Compute API' return struct of image : ktcloudvpc-sdk-go/openstack/compute/v2/images/results.go

	call 		"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv 		"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs 		"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type KTVpcImageHandler struct {
	RegionInfo    idrv.RegionInfo
	VMClient      *ktvpcsdk.ServiceClient
	ImageClient   *ktvpcsdk.ServiceClient
}

func (imageHandler *KTVpcImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called ListImage()")	
	callLogInfo := getCallLogScheme(imageHandler.RegionInfo.Zone, call.VMIMAGE, "ListImage()", "ListImage()") // HisCall logging

	listOpts :=	images.ListOpts{
		Limit: 300,  //default : 20
		Visibility: images.ImageVisibilityPublic, // Note : Public image only
	}
	start := call.Start()
	allPages, err := images.List(imageHandler.ImageClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud VPC Image List. [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	loggingInfo(callLogInfo, start)
	// cblogger.Info("### allPages : ")
	// spew.Dump(allPages)

	imageList, err := images.ExtractImages(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud VPC Image List. [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	// cblogger.Info("### imageList : ")
	// spew.Dump(imageList)

	var imageInfoList []*irs.ImageInfo
    for _, image := range imageList {
		imageInfo, err := imageHandler.mappingImageInfo(&image)
		if err != nil {
			newErr := fmt.Errorf("Failed to Map KT Cloud VPC Image Info. [%v]", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return nil, newErr
		}
		imageInfoList = append(imageInfoList, imageInfo)
    }
	return imageInfoList, nil
}

func (imageHandler *KTVpcImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetImage()")
	callLogInfo := getCallLogScheme(imageHandler.RegionInfo.Zone, call.VMIMAGE, imageIID.SystemId, "GetImage()") // HisCall logging

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.ImageInfo{}, newErr
	}

	ktImage, err := imageHandler.getKTImage(imageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find the Image info with the ID!! [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.ImageInfo{}, newErr
	}

	if strings.EqualFold(ktImage.ID, "") {
		return irs.ImageInfo{}, fmt.Errorf("Failed to Get Any Image Info.")
	}

	//Ref) 'Image API' return struct of image :ktcloudvpc-sdk-go/openstack/imageservice/v2/images/results.go
	//Ref) 'Compute API' return struct of image : ktcloudvpc-sdk-go/openstack/compute/v2/images/results.go
	imageInfo, err := imageHandler.mappingImageInfo(ktImage)
	if err != nil {
		cblogger.Error(err.Error())
		loggingError(callLogInfo, err)
		return irs.ImageInfo{}, err
	}
	return *imageInfo, nil
}

func (imageHandler *KTVpcImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called CreateImage()!")

	return irs.ImageInfo{}, fmt.Errorf("Does not support CreateImage() yet!!")
}

func (imageHandler *KTVpcImageHandler) CheckWindowsImage(imageIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud VP Driver: called CheckWindowsImage()")

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Image SystemId!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}	

	ktImage, err := imageHandler.getKTImage(imageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find the Image with the ID!! [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	if strings.EqualFold(ktImage.ID, "") {
		return false, fmt.Errorf("Failed to Get Any Image Info.")
	}

	isWindowsImage := false
	if strings.Contains(strings.ToLower(ktImage.Name), "windows") {
		isWindowsImage = true
	}
	return isWindowsImage, nil
}

func (imageHandler *KTVpcImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called DeleteImage()!")

	return true, fmt.Errorf("Does not support DeleteImage() yet!!")
}

func (imageHandler *KTVpcImageHandler) mappingImageInfo(image *images.Image) (*irs.ImageInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called mappingImageInfo()!")
	// cblogger.Info("\n\n### image : ")
	// spew.Dump(image)
	// cblogger.Info("\n")

	//Ref) 'Image API' return struct of image :ktcloudvpc-sdk-go/openstack/imageservice/v2/images/results.go
	//Ref) 'Compute API' return struct of image : ktcloudvpc-sdk-go/openstack/compute/v2/images/results.go

	if strings.EqualFold(image.ID, "") {
		return nil, fmt.Errorf("Failed to Get Any Image Info.")
	}

	var imgAvailability string
	if strings.EqualFold(string(image.Status), "active") {
		imgAvailability = "available"
	} else {
		imgAvailability = "unavailable"
	}

	var osPlatform irs.OSPlatform 
		if image.Name != "" {
			if strings.Contains(image.Name, "Windows") || strings.Contains(image.Name, "windows") || strings.Contains(image.Name, "win") {
				osPlatform = irs.Windows				
			} else {
				osPlatform = irs.Linux_UNIX
			}			
		} else {
			osPlatform = irs.PlatformNA
		}

	var imageStatus irs.ImageStatus
	if image.Status != "" {
		if strings.EqualFold(string(image.Status), "active") { 
			imageStatus = irs.ImageAvailable
		} else {
			imageStatus = irs.ImageUnavailable
		}
	} else {
		imageStatus = irs.ImageNA
	}

	// # Note) image.SizeBytes is not Root Disk Size
	// valueInGB := float64(image.SizeBytes) / (1024 * 1024 * 1024)	
	// diskSizeInGB := strconv.FormatFloat(valueInGB, 'f', 0, 64)

	imageInfo := &irs.ImageInfo {
		IId: irs.IID{
			NameId:   image.ID, // Caution!!
			SystemId: image.ID,
		},
		GuestOS:      image.Name, // Caution!!
		Status: 	  imgAvailability,

		Name: 			image.ID,
		OSArchitecture: "NA",
		OSPlatform: 	osPlatform,		
		OSDistribution: image.Name,
		OSDiskType: 	"NA",
		OSDiskSizeInGB: "NA",
		ImageStatus: 	imageStatus,
	}

	keyValueList := []irs.KeyValue{
		{Key: "Zone", 		 	  Value: imageHandler.RegionInfo.Zone},
		{Key: "DiskFormat:", 	  Value: string(image.DiskFormat)},
		{Key: "ContainerFormat:", Value: string(image.ContainerFormat)},
		{Key: "Visibility:", 	  Value: string(image.Visibility)},

	}
	imageInfo.KeyValueList = keyValueList
	return imageInfo, nil
}


// # Get 'MyImage' Info from KT Cloud
func (imageHandler *KTVpcImageHandler) getKTImage(imageIID irs.IID) (*images.Image, error) {
	cblogger.Info("KT Cloud Driver: called getKTImage()")
	callLogInfo := getCallLogScheme(imageHandler.RegionInfo.Zone, call.VMIMAGE, imageIID.SystemId, "isPublicImage()")

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Image SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}

	start := call.Start()
	image, err := images.Get(imageHandler.ImageClient, imageIID.SystemId).Extract()  // Not ~.VMClient
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud VPC Image Info. [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	loggingInfo(callLogInfo, start)
	
	if strings.EqualFold(image.ID, "") {
		return nil, fmt.Errorf("Failed to Get Any Image Info.")
	}
	return image, nil
}

func (imageHandler *KTVpcImageHandler) isPublicImage(imageIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud Driver: called isPublicImage()")

	if strings.EqualFold(imageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Image SystemId!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}	

	ktImage, err := imageHandler.getKTImage(imageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find the Image with the ID!! [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	if strings.EqualFold(ktImage.ID, "") {
		return false, fmt.Errorf("Failed to Get Any Image Info.")
	}

	isPublicImage := false
	if (ktImage.Visibility == images.ImageVisibilityPublic) {
		isPublicImage = true
	}
	return isPublicImage, nil
}
