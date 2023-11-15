package resources

import (
	"fmt"
	"strings"
	"strconv"
	"time"
	// "google.golang.org/grpc/metadata"
	"github.com/davecgh/go-spew/spew"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/servers"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/flavors"
	images "github.com/cloud-barista/nhncloud-sdk-go/openstack/imageservice/v2/images" // imageservice/v2/images : For Visibility parameter
	// comimages "github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/images" // compute/v2/images
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/blockstorage/extensions/volumeactions"
	// "github.com/cloud-barista/nhncloud-sdk-go/openstack/blockstorage/v2/snapshots"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NhnCloudMyImageHandler struct {
	RegionInfo    idrv.RegionInfo
	VMClient      *nhnsdk.ServiceClient
	ImageClient   *nhnsdk.ServiceClient
	NetworkClient *nhnsdk.ServiceClient
	VolumeClient  *nhnsdk.ServiceClient
}

// To Take a Snapshot with VM ID (To Create My Image) 
func (myImageHandler *NhnCloudMyImageHandler) SnapshotVM(snapshotReqInfo irs.MyImageInfo) (irs.MyImageInfo, error) {
	cblogger.Info("NHN Cloud Driver: called SnapshotVM()")
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, snapshotReqInfo.SourceVM.SystemId, "SnapshotVM()")

	if strings.EqualFold(snapshotReqInfo.SourceVM.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}

	var snapshotName string
	if !strings.EqualFold(snapshotReqInfo.IId.NameId, "") {
		snapshotName = snapshotReqInfo.IId.NameId
	}

	nhnVMSpecType, err := myImageHandler.GetVMSpecType(irs.IID{SystemId: snapshotReqInfo.SourceVM.SystemId})
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VM Spec Type. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}

	cblogger.Info("\n\n### nhnVMSpecType : ")
	spew.Dump(nhnVMSpecType)
	cblogger.Info("\n")

	// snapShotMap := make(map[string]string)
    // snapShotMap["vmID"] = snapshotReqInfo.SourceVM.SystemId

	vmSpecType := nhnVMSpecType[:2] // Ex) vmSpecType : 'u2', 'm2' or 'c2' ...
	cblogger.Infof("# vmSpecType : [%s]", vmSpecType)

	var newImageIID irs.IID

	if strings.EqualFold(vmSpecType, "u2") {
		start := call.Start()
		snapshotOpts := servers.CreateImageOpts{
			Name:		snapshotName,
			// Metadata: 	snapShotMap,
		}
		snapShotImageId, err := servers.CreateImage(myImageHandler.VMClient, snapshotReqInfo.SourceVM.SystemId, snapshotOpts).ExtractImageID() // Not images.CreateImage()
		if err != nil {
			newErr := fmt.Errorf("Failed to Create Snapshot of the VM. [%v]", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.MyImageInfo{}, newErr
		}	
		LoggingInfo(callLogInfo, start)	
		cblogger.Infof("\n\n# snapShotImageId : [%s]\n", snapShotImageId)

		newImageIID = irs.IID{SystemId: snapShotImageId}
	} else {
		bootableVolumeId, err := myImageHandler.GetBootableVolumeID(snapshotReqInfo.SourceVM)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get Bootable VolumeID of the VM. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.MyImageInfo{}, newErr
		}

		start := call.Start()
		uploadImageOpts := volumeactions.UploadImageOpts {
			ImageName: snapshotName,
			Force:     true,
		}
		volumeImage, err := volumeactions.UploadImage(myImageHandler.VolumeClient, bootableVolumeId, uploadImageOpts).Extract()
		if err != nil {
			newErr := fmt.Errorf("Failed to Create Image from the Volume!! : [%v] ", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.MyImageInfo{}, newErr
		}
		LoggingInfo(callLogInfo, start)
		cblogger.Infof("\n\n# snapShotImageId : [%s]\n", volumeImage.ImageID)
	
		newImageIID = irs.IID{SystemId: volumeImage.ImageID}
	
		cblogger.Info("\n\n### volumeImage : ")
		spew.Dump(volumeImage)
		cblogger.Info("\n")
	}

	// To Wait for Creating a Snapshot Image
	curStatus, err := myImageHandler.WaitForImageSnapshot(newImageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait to Get Image Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	cblogger.Infof("==> Image Status of [%s] : [%s]", newImageIID.SystemId, string(curStatus))

	myImageInfo, err := myImageHandler.GetMyImage(newImageIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait to Get Image Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	return myImageInfo, nil
}

// To Manage My Images
func (myImageHandler *NhnCloudMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
	cblogger.Info("NHN Cloud Driver: called ListMyImage()")
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, "ListMyImage()", "ListMyImage()")

	start := call.Start()
	listOpts :=	images.ListOpts{
		Visibility: images.ImageVisibilityPrivate, // Note : Private image only
	}
	allPages, err := images.List(myImageHandler.ImageClient, listOpts).AllPages()
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

	var imageInfoList []*irs.MyImageInfo
    for _, nhnImage := range nhnImageList {
		imageInfo := myImageHandler.MappingMyImageInfo(nhnImage)
		imageInfoList = append(imageInfoList, imageInfo)
    }
	return imageInfoList, nil
}

func (myImageHandler *NhnCloudMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {
	cblogger.Info("NHN Cloud Driver: called GetMyImage()")
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "GetMyImage()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}

	start := call.Start()
	// nhnImage, err := comimages.Get(myImageHandler.VMClient, myImageIID.SystemId).Extract() // VM Client
	nhnImage, err := images.Get(myImageHandler.ImageClient, myImageIID.SystemId).Extract() // Image Client
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud My Image Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	imageInfo := myImageHandler.MappingMyImageInfo(*nhnImage)
	return *imageInfo, nil
}

func (myImageHandler *NhnCloudMyImageHandler) CheckWindowsImage(myImageIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called CheckWindowsImage()")

	return false, fmt.Errorf("NHN Cloud Driver Does not support CheckWindowsImage() yet!!")
}

func (myImageHandler *NhnCloudMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called DeleteMyImage()")
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "DeleteMyImage()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	err := images.Delete(myImageHandler.ImageClient, myImageIID.SystemId).ExtractErr()
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the Image. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	return true, nil
}

func (myImageHandler *NhnCloudMyImageHandler) GetImageStatus(myImageIID irs.IID) (irs.MyImageStatus, error) {
	cblogger.Info("NHN Cloud Driver: called GetImageStatus()")
	callLogInfo := GetCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "GetImageStatus()")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}

	start := call.Start()
	// nhnImage, err := comimages.Get(myImageHandler.VMClient, myImageIID.SystemId).Extract() // VM Client
	nhnImage, err := images.Get(myImageHandler.ImageClient, myImageIID.SystemId).Extract() // Image Client
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud My Image Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return "", newErr
	}
	LoggingInfo(callLogInfo, start)

	myImageStatus := ConvertImageStatus(nhnImage.Status)
	return myImageStatus, nil
}

// Waiting for up to 500 seconds during Taking a Snapshot from a VM
func (myImageHandler *NhnCloudMyImageHandler) WaitForImageSnapshot(myImageIID irs.IID) (irs.MyImageStatus, error) {
	cblogger.Info("===> Since Snapshot info. cannot be retrieved immediately after taking a snapshot, waits ....")

	if strings.EqualFold(myImageIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid SystemId!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	curRetryCnt := 0
	maxRetryCnt := 500
	for {
		curStatus, err := myImageHandler.GetImageStatus(myImageIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Image Status of [%s] : [%v] ", myImageIID.NameId, err)
			cblogger.Error(newErr.Error())
			return "Failed. ", newErr
		} else {
			cblogger.Infof("Succeeded in Getting the Image Status of [%s] : [%s]", myImageIID.SystemId, string(curStatus))
		}

		cblogger.Infof("===> Image Status : [%s]", string(curStatus))

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

func (myImageHandler *NhnCloudMyImageHandler) MappingMyImageInfo(myImage images.Image) *irs.MyImageInfo {
	cblogger.Info("NHN Cloud Driver: called MappingMyImageInfo()!")

	// cblogger.Info("\n\n### myImage in MappingMyImageInfo() : ")
	// spew.Dump(myImage)
	// cblogger.Info("\n")

	myImageInfo := &irs.MyImageInfo {
		IId: irs.IID{
			NameId:   myImage.Name,
			SystemId: myImage.ID,
		},
		Status: 	  ConvertImageStatus(myImage.Status),
		CreatedTime:  myImage.CreatedAt,
	}

	keyValueList := []irs.KeyValue{
		{Key: "Region", Value: myImageHandler.RegionInfo.Region},
		{Key: "Visibility", Value: string(myImage.Visibility)},
		{Key: "DiskSize", Value: strconv.Itoa(myImage.MinDiskGigabytes)},
	}

	// In case the VMSpec type of the SourceVM is 'u2', the map of a snapshot image contains "instance_uuid".
	if val, ok := myImage.Properties["instance_uuid"]; ok {
		myImageInfo.SourceVM = irs.IID{SystemId: fmt.Sprintf("%v", val)}
	}

	for key, val := range myImage.Properties {
		if (key == "os_type" || key == "description" || key == "os_architecture" || key == "hypervisor_type" || key == "image_type" || key == "os_distro" || key == "os_version") {
			metadata := irs.KeyValue{
				Key:   strings.ToUpper(key),
				Value: fmt.Sprintf("%v", val),
			}
		keyValueList = append(keyValueList, metadata)
		}
	}	

	myImageInfo.KeyValueList = keyValueList
	return myImageInfo
}

func ConvertImageStatus(myImageStatus images.ImageStatus) irs.MyImageStatus {
	cblogger.Info("NHN Cloud Driver: called ConvertImageStatus()")
	
	// Ref) https://github.com/cloud-barista/nhncloud-sdk-go/blob/main/openstack/imageservice/v2/images/types.go
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

func (myImageHandler *NhnCloudMyImageHandler) GetVMSpecType(vmIID irs.IID) (string, error) {
	cblogger.Info("NHN Cloud Driver: called GetVMSpecType()")

	if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid VM SystemId!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	nhnVM, err := servers.Get(myImageHandler.VMClient, vmIID.SystemId).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VM info form NHN Cloud!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	var vmSpecType string
	flavorId := nhnVM.Flavor["id"].(string)
	nhnFlavor, err := flavors.Get(myImageHandler.VMClient, flavorId).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Flavor info form NHN Cloud!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	} else if nhnFlavor != nil {		
		vmSpecType = nhnFlavor.Name
	}

	return vmSpecType, nil
}

func (myImageHandler *NhnCloudMyImageHandler) GetBootableVolumeID(vmIID irs.IID) (string, error) {
	cblogger.Info("NHN Cloud Driver: called GetBootableVolumeID()")

	vmHandler := NhnCloudVMHandler{
		RegionInfo:     myImageHandler.RegionInfo,
		VMClient:       myImageHandler.VMClient,
		ImageClient:   	myImageHandler.ImageClient,
		NetworkClient:  myImageHandler.NetworkClient,
		VolumeClient:  	myImageHandler.VolumeClient,
	}

	diskHandler := NhnCloudDiskHandler{
		RegionInfo:     myImageHandler.RegionInfo,
		VMClient:       myImageHandler.VMClient,
		VolumeClient:   myImageHandler.VolumeClient,
	}

	vmInfo, err := vmHandler.GetVM(vmIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VM Info. : [%v] ", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	var bootableVolumeId string
	for _, diskIID := range vmInfo.DataDiskIIDs {
		isBootable, err := diskHandler.IsBootableDisk(diskIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Bootable Disk Info. : [%v] ", err)
			cblogger.Error(newErr.Error())
			return "", newErr
		} else if isBootable {
			bootableVolumeId = diskIID.SystemId
		}
	}
	
	if strings.EqualFold(bootableVolumeId, "") {
		newErr := fmt.Errorf("Failed to Find any Bootable Volume : [%v] ", err)
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	return bootableVolumeId, nil
}
