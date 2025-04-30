package resources

import (
	// "errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata" // To prevent 'unknown time zone Asia/Seoul' error
	// "github.com/davecgh/go-spew/spew"

	ktvpcsdk "github.com/cloud-barista/ktcloudvpc-sdk-go"
	// volumes2 	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/blockstorage/v2/volumes"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/blockstorage/extensions/volumeactions"

	images "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/imageservice/v2/images" // imageservice/v2/images : For Visibility parameter
	// Not '~/openstack/compute/v2/images'

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type KTVpcMyImageHandler struct {
	RegionInfo    idrv.RegionInfo
	VMClient      *ktvpcsdk.ServiceClient
	ImageClient   *ktvpcsdk.ServiceClient
	NetworkClient *ktvpcsdk.ServiceClient
	VolumeClient  *ktvpcsdk.ServiceClient
}

// To Take a Snapshot Root Volume with VM ID (To Create My Image)
func (myImageHandler *KTVpcMyImageHandler) SnapshotVM(snapshotReqInfo irs.MyImageInfo) (irs.MyImageInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called SnapshotVM()")
	callLogInfo := getCallLogScheme(myImageHandler.RegionInfo.Zone, call.MYIMAGE, snapshotReqInfo.SourceVM.SystemId, "SnapshotVM()")

	if strings.EqualFold(snapshotReqInfo.SourceVM.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}

	var snapshotName string
	if !strings.EqualFold(snapshotReqInfo.IId.NameId, "") {
		snapshotName = snapshotReqInfo.IId.NameId
	}

	bootableVolumeId, err := myImageHandler.getBootableVolumeID(snapshotReqInfo.SourceVM)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Bootable VolumeID of the VM. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}

	uploadImageOpts := volumeactions.UploadImageOpts{
		ImageName: snapshotName,
		Force:     true, // Even if the volume is connected to the server, whether to create an image.
	}
	start := call.Start()
	volumeImage, err := volumeactions.UploadImage(myImageHandler.VolumeClient, bootableVolumeId, uploadImageOpts).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Create Image from the Volume!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	loggingInfo(callLogInfo, start)
	cblogger.Infof("\n\n# snapShotImageId : [%s]\n", volumeImage.ImageID)

	// cblogger.Info("\n\n### volumeImage : ")
	// spew.Dump(volumeImage)
	// cblogger.Info("\n")

	// To Wait for Creating a Snapshot Image
	newImageIID := irs.IID{SystemId: volumeImage.ImageID}
	curStatus, err := myImageHandler.waitForImageSnapshot(newImageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait to Get Image Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	cblogger.Infof("==> Image Status of [%s] : [%s]", newImageIID.SystemId, string(curStatus))

	myImageInfo, err := myImageHandler.GetMyImage(newImageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait for Getting New Image Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	return myImageInfo, nil
}

// To Manage My Images
func (myImageHandler *KTVpcMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called ListMyImage()")
	callLogInfo := getCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, "ListMyImage()", "ListMyImage()")

	/*
		// ImageVisibilityPublic all users
		ImageVisibilityPublic ImageVisibility = "public"

		// ImageVisibilityPrivate users with tenantId == tenantId(owner)
		ImageVisibilityPrivate ImageVisibility = "private"

		// ImageVisibilityShared images are visible to:
		// - users with tenantId == tenantId(owner)
		// - users with tenantId in the member-list of the image
		// - users with tenantId in the member-list with member_status == 'accepted'
		ImageVisibilityShared ImageVisibility = "shared"

		// ImageVisibilityCommunity images:
		// - all users can see and boot it
		// - users with tenantId in the member-list of the image with
		//	 member_status == 'accepted' have this image in their default image-list.
		ImageVisibilityCommunity ImageVisibility = "community"
	*/

	listOpts := images.ListOpts{
		Visibility: images.ImageVisibilityShared, // Not 'ImageVisibilityPrivate'
	}
	start := call.Start()
	allPages, err := images.List(myImageHandler.ImageClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud VPC Image pages. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	loggingInfo(callLogInfo, start)

	ktImageList, err := images.ExtractImages(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud VPC Image List. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}

	// cblogger.Info("\n\n### ktImageList : ")
	// spew.Dump(ktImageList)
	// cblogger.Info("# ktImage count : ", len(ktImageList))

	// Note) Public image : ktImage.Visibility == "public", MyImage : ktImage.Visibility == "shared"
	var imageInfoList []*irs.MyImageInfo
	for _, ktImage := range ktImageList {
		imageInfo, err := myImageHandler.mappingMyImageInfo(ktImage)
		if err != nil {
			newErr := fmt.Errorf("Failed to Map the MyImage Info. [%v]", err)
			return nil, newErr
		}
		imageInfoList = append(imageInfoList, imageInfo)
	}
	return imageInfoList, nil
}

func (myImageHandler *KTVpcMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetMyImage()")
	callLogInfo := getCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "GetMyImage()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}

	start := call.Start()
	// ktImage, err := comimages.Get(myImageHandler.VMClient, myImageIID.SystemId).Extract() // VM Client
	ktImage, err := images.Get(myImageHandler.ImageClient, myImageIID.SystemId).Extract() // Image Client
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud VPC My Image Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	loggingInfo(callLogInfo, start)

	imageInfo, err := myImageHandler.mappingMyImageInfo(*ktImage)
	if err != nil {
		newErr := fmt.Errorf("Failed to Map the MyImage Info. [%v]", err)
		return irs.MyImageInfo{}, newErr
	}
	return *imageInfo, nil
}

func (myImageHandler *KTVpcMyImageHandler) CheckWindowsImage(myImageIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called CheckWindowsImage()")

	return false, fmt.Errorf("KT Cloud VPC Driver Does not support CheckWindowsImage() yet!!")
}

func (myImageHandler *KTVpcMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called DeleteMyImage()")
	callLogInfo := getCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "DeleteMyImage()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	start := call.Start()
	err := images.Delete(myImageHandler.ImageClient, myImageIID.SystemId).ExtractErr()
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the Image. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}
	loggingInfo(callLogInfo, start)

	return true, nil
}

func (myImageHandler *KTVpcMyImageHandler) getImageStatus(myImageIID irs.IID) (irs.MyImageStatus, error) {
	cblogger.Info("KT Cloud VPC Driver: called getImageStatus()")
	callLogInfo := getCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "getImageStatus()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}

	start := call.Start()
	// ktImage, err := comimages.Get(myImageHandler.VMClient, myImageIID.SystemId).Extract() // VM Client
	ktImage, err := images.Get(myImageHandler.ImageClient, myImageIID.SystemId).Extract() // Image Client
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud VPC My Image Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return "", newErr
	}
	loggingInfo(callLogInfo, start)
	cblogger.Infof("===> KT Image Status : [%s]", string(ktImage.Status))

	myImageStatus := convertImageStatus(ktImage.Status)
	return myImageStatus, nil
}

// Waiting for up to 500 seconds during Taking a Snapshot from a VM
func (myImageHandler *KTVpcMyImageHandler) waitForImageSnapshot(myImageIID irs.IID) (irs.MyImageStatus, error) {
	cblogger.Info("KT Cloud VPC Driver: called waitForImageSnapshot()")
	cblogger.Info("===> Since Snapshot info. cannot be retrieved immediately after taking a snapshot, waits ....")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	curRetryCnt := 0
	maxRetryCnt := 500
	for {
		curStatus, err := myImageHandler.getImageStatus(myImageIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Image Status of [%s] : [%v] ", myImageIID.NameId, err)
			cblogger.Error(newErr.Error())
			return "Failed. ", newErr
		} else {
			cblogger.Infof("Succeeded in Getting the Image Status of [%s] : [%s]", myImageIID.SystemId, string(curStatus))
		}
		// cblogger.Infof("===> Image Status(Converted) : [%s]", string(curStatus))

		if strings.EqualFold(string(curStatus), "Unavailable") {
			curRetryCnt++
			cblogger.Infof("The Image is still 'Unavailable', so wait for a second more before inquiring the Image info.")
			time.Sleep(time.Second * 2)
			if curRetryCnt > maxRetryCnt {
				newErr := fmt.Errorf("Despite waiting for a long time(%d sec), the Image status is %s, so it is forcibly finished.", maxRetryCnt, string(curStatus))
				cblogger.Error(newErr.Error())
				return "Failed. ", newErr
			}
		} else {
			cblogger.Infof("===> ### The Image Snapshot is finished, stopping the waiting.")
			return curStatus, nil
			//break
		}
	}
}

func (myImageHandler *KTVpcMyImageHandler) mappingMyImageInfo(myImage images.Image) (*irs.MyImageInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called mappingMyImageInfo()!")
	// cblogger.Info("\n\n### myImage in mappingMyImageInfo() : ")
	// spew.Dump(myImage)
	// cblogger.Info("\n")

	// Convert to KTC
	convertedTime, err := convertTimeToKTC(myImage.CreatedAt)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Converted Time. [%v]", err)
		return nil, newErr
	}

	myImageInfo := &irs.MyImageInfo{
		IId: irs.IID{
			NameId:   myImage.Name,
			SystemId: myImage.ID,
		},
		Status:      convertImageStatus(myImage.Status),
		CreatedTime: convertedTime,

		KeyValueList:   irs.StructToKeyValueList(myImage),
	}

	keyValueList := []irs.KeyValue{
		{Key: "Zone", Value: myImageHandler.RegionInfo.Zone},
		{Key: "ImageSize(GB)", Value: strconv.FormatInt(myImage.SizeBytes/(1024*1024*1024), 10)},
	}
	myImageInfo.KeyValueList = append(myImageInfo.KeyValueList, keyValueList...)
	return myImageInfo, nil
}

func convertImageStatus(myImageStatus images.ImageStatus) irs.MyImageStatus {
	cblogger.Info("KT Cloud VPC Driver: called convertImageStatus()")

	// Ref) https://github.com/cloud-barista/ktcloudvpc-sdk-go/blob/main/openstack/imageservice/v2/images/types.go
	var resultStatus irs.MyImageStatus
	switch myImageStatus {
	case images.ImageStatusQueued:
		resultStatus = irs.MyImageUnavailable
	case images.ImageStatusSaving:
		resultStatus = irs.MyImageUnavailable
	case images.ImageStatusActive:
		resultStatus = irs.MyImageAvailable
	case images.ImageStatusKilled:
		resultStatus = irs.MyImageUnavailable
	case images.ImageStatusDeleted:
		resultStatus = irs.MyImageUnavailable
	case images.ImageStatusPendingDelete:
		resultStatus = irs.MyImageUnavailable
	default:
		resultStatus = "Unknown"
	}

	return resultStatus
}

func (myImageHandler *KTVpcMyImageHandler) getBootableVolumeID(vmIID irs.IID) (string, error) {
	cblogger.Info("KT Cloud VPC Driver: called getBootableVolumeID()")

	diskHandler := KTVpcDiskHandler{
		RegionInfo:   myImageHandler.RegionInfo,
		VMClient:     myImageHandler.VMClient,
		VolumeClient: myImageHandler.VolumeClient,
	}

	nhnVolumeList, err := diskHandler.getKtVolumeList()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud Volume Pages!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	var bootableVolumeId string
	for _, nhnVolume := range nhnVolumeList {
		isBootable, err := strconv.ParseBool(nhnVolume.Bootable)
		if err != nil {
			newErr := fmt.Errorf("Failed to Parse the String value!! : [%v]", err)
			cblogger.Error(newErr.Error())
			return "", newErr
		}

		if isBootable && nhnVolume.Attachments != nil && len(nhnVolume.Attachments) > 0 {
			for _, attachment := range nhnVolume.Attachments {
				if strings.EqualFold(attachment.ServerID, vmIID.SystemId) {
					bootableVolumeId = attachment.VolumeID
					break
				}
			}
		}
	}

	if strings.EqualFold(bootableVolumeId, "") {
		newErr := fmt.Errorf("Failed to Find any Bootable Volume : [%v] ", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	return bootableVolumeId, nil
}

func (myImageHandler *KTVpcMyImageHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("KT Cloud VPC driver: called ListIID()!!")
	callLogInfo := getCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, "ListIID()", "ListMyImage()")

    listOpts := images.ListOpts{
        Visibility: images.ImageVisibilityShared, // Not 'ImageVisibilityPrivate'
    }
	start := call.Start()
    allPages, err := images.List(myImageHandler.ImageClient, listOpts).AllPages()
    if err != nil {
        newErr := fmt.Errorf("Failed to Get KT Cloud VPC Image pages. [%v]", err.Error())
        cblogger.Error(newErr.Error())
        return nil, newErr
    }
	loggingInfo(callLogInfo, start)

    ktImageList, err := images.ExtractImages(allPages)
    if err != nil {
        newErr := fmt.Errorf("Failed to Get KT Cloud VPC Image List. [%v]", err.Error())
        cblogger.Error(newErr.Error())
        return nil, newErr
    }

    var iidList []*irs.IID
    for _, ktImage := range ktImageList {
        iid := &irs.IID{
            NameId:   ktImage.Name,
            SystemId: ktImage.ID,
        }
        iidList = append(iidList, iid)
    }
    return iidList, nil
}
