package resources

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	// "google.golang.org/grpc/metadata"
	// "github.com/davecgh/go-spew/spew"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/flavors"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/servers"
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

func (myImageHandler *NhnCloudMyImageHandler) getRawSnapshot(snapshotIID irs.IID) (images.Image, error) {
	if snapshotIID.NameId == "" && snapshotIID.SystemId == "" {
		return images.Image{}, errors.New("invalid IID")
	}
	if snapshotIID.SystemId != "" {
		image, err := images.Get(myImageHandler.ImageClient, snapshotIID.SystemId).Extract()
		if err != nil {
			newErr := fmt.Errorf("Failed to Get NHN Cloud Image. [%v]", err)
			cblogger.Error(newErr.Error())
			return images.Image{}, newErr
		}
		return *image, nil
	}

	listOpts := images.ListOpts{
		Visibility: images.ImageVisibilityPrivate, // Note : Private image only
	}
	allPages, err := images.List(myImageHandler.ImageClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud Image pages. [%v]", err)
		cblogger.Error(newErr.Error())
		return images.Image{}, newErr
	}
	nhnImageList, err := images.ExtractImages(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud Image List. [%v]", err)
		cblogger.Error(newErr.Error())
		return images.Image{}, newErr
	}

	for _, nhnImage := range nhnImageList {
		if nhnImage.Name == snapshotIID.NameId {
			return nhnImage, nil
		}
	}

	return images.Image{}, errors.New("MyImage not found")
}

// To Take a Snapshot with VM ID (To Create My Image)
func (myImageHandler *NhnCloudMyImageHandler) SnapshotVM(snapshotReqInfo irs.MyImageInfo) (irs.MyImageInfo, error) {
	cblogger.Info("NHN Cloud Driver: called SnapshotVM()")
	callLogInfo := getCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, snapshotReqInfo.SourceVM.SystemId, "SnapshotVM()")

	vmHandler := NhnCloudVMHandler{
		VMClient: myImageHandler.VMClient,
	}
	vm, err := vmHandler.getRawVM(snapshotReqInfo.SourceVM)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VM. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}

	var snapshotName string
	if !strings.EqualFold(snapshotReqInfo.IId.NameId, "") {
		snapshotName = snapshotReqInfo.IId.NameId
	}

	nhnVMSpecType, err := myImageHandler.getVMSpecType(vm.ID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the VM Spec Type. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	// cblogger.Info("\n\n### nhnVMSpecType : ")
	// spew.Dump(nhnVMSpecType)
	// cblogger.Info("\n")

	// snapShotMap := make(map[string]string)
	// snapShotMap["vmID"] = snapshotReqInfo.SourceVM.SystemId

	vmSpecType := nhnVMSpecType[:2] // Ex) vmSpecType : 'u2', 'm2' or 'c2' ...
	cblogger.Infof("# vmSpecType : [%s]", vmSpecType)

	var newImageIID irs.IID

	if strings.EqualFold(vmSpecType, "u2") {
		start := call.Start()
		snapshotOpts := servers.CreateImageOpts{
			Name: snapshotName,
			// Metadata: 	snapShotMap,
		}
		snapShotImageId, err := servers.CreateImage(myImageHandler.VMClient, vm.ID, snapshotOpts).ExtractImageID() // Not images.CreateImage()
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
		bootableVolumeId, err := myImageHandler.getBootableVolumeID(vm.ID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get Bootable VolumeID of the VM. [%v]", err.Error())
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return irs.MyImageInfo{}, newErr
		}

		start := call.Start()
		uploadImageOpts := volumeactions.UploadImageOpts{
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

		// cblogger.Info("\n\n### volumeImage : ")
		// spew.Dump(volumeImage)
		// cblogger.Info("\n")
	}

	// To Wait for Creating a Snapshot Image
	curStatus, err := myImageHandler.waitForImageSnapshot(newImageIID)
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
	callLogInfo := getCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, "ListMyImage()", "ListMyImage()")

	start := call.Start()
	listOpts := images.ListOpts{
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

	var imageInfoList []*irs.MyImageInfo
	for _, nhnImage := range nhnImageList {
		imageInfo := myImageHandler.mappingMyImageInfo(nhnImage)
		imageInfoList = append(imageInfoList, imageInfo)
	}
	return imageInfoList, nil
}

func (myImageHandler *NhnCloudMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {
	cblogger.Info("NHN Cloud Driver: called GetMyImage()")
	callLogInfo := getCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "GetMyImage()")

	start := call.Start()
	// nhnImage, err := comimages.Get(myImageHandler.VMClient, myImageIID.SystemId).Extract() // VM Client
	nhnImage, err := myImageHandler.getRawSnapshot(myImageIID) // Image Client
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud My Image Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.MyImageInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	imageInfo := myImageHandler.mappingMyImageInfo(nhnImage)
	return *imageInfo, nil
}

func (myImageHandler *NhnCloudMyImageHandler) CheckWindowsImage(myImageIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called CheckWindowsImage()")
	callLogInfo := getCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "GetMyImage()")

	start := call.Start()
	nhnImage, err := myImageHandler.getRawSnapshot(myImageIID) // Image Client
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud My Image Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, start)

	var osDistro string
	if !strings.EqualFold(nhnImage.Properties["os_distro"].(string), "") {
		osDistro = nhnImage.Properties["os_distro"].(string)
	} else {
		newErr := fmt.Errorf("Failed to Find OS Distro Info from MyImage. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	isWindowsImage := false
	if strings.Contains(osDistro, "windows") {
		isWindowsImage = true
	}
	return isWindowsImage, nil
}

func (myImageHandler *NhnCloudMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called DeleteMyImage()")
	callLogInfo := getCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "DeleteMyImage()")

	nhnImage, err := myImageHandler.getRawSnapshot(myImageIID) // Image Client
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud My Image Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	err = images.Delete(myImageHandler.ImageClient, nhnImage.ID).ExtractErr()
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the Image. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	return true, nil
}

func (myImageHandler *NhnCloudMyImageHandler) getImageStatus(myImageIID irs.IID) (irs.MyImageStatus, error) {
	cblogger.Info("NHN Cloud Driver: called getImageStatus()")
	callLogInfo := getCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "getImageStatus()")

	start := call.Start()
	// nhnImage, err := comimages.Get(myImageHandler.VMClient, myImageIID.SystemId).Extract() // VM Client
	nhnImage, err := myImageHandler.getRawSnapshot(myImageIID) // Image Client
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
func (myImageHandler *NhnCloudMyImageHandler) waitForImageSnapshot(myImageIID irs.IID) (irs.MyImageStatus, error) {
	cblogger.Info("===> Since Snapshot info. cannot be retrieved immediately after taking a snapshot, waits ....")

	curRetryCnt := 0
	maxRetryCnt := 500
	for {
		curStatus, err := myImageHandler.getImageStatus(myImageIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Image Status of [%s] : [%v] ", myImageIID.NameId, err)
			cblogger.Error(newErr.Error())
			return "Failed. ", newErr
		} else {
			cblogger.Infof("Succeeded in Getting the Image Status of [%s] : [%s]", myImageIID.NameId, string(curStatus))
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

func (myImageHandler *NhnCloudMyImageHandler) mappingMyImageInfo(myImage images.Image) *irs.MyImageInfo {
	cblogger.Info("NHN Cloud Driver: called mappingMyImageInfo()!")

	myImageInfo := &irs.MyImageInfo{
		IId: irs.IID{
			NameId:   myImage.Name,
			SystemId: myImage.ID,
		},
		Status:      ConvertImageStatus(myImage.Status),
		CreatedTime: myImage.CreatedAt,
	}

	myImageInfo.KeyValueList = irs.StructToKeyValueList(myImage)

	//keyValueList := []irs.KeyValue{
	//	{Key: "Region", Value: myImageHandler.RegionInfo.Region},
	//	{Key: "Visibility", Value: string(myImage.Visibility)},
	//	{Key: "DiskSize", Value: strconv.Itoa(myImage.MinDiskGigabytes)},
	//}

	// In case the VMSpec type of the SourceVM is 'u2', the map of a snapshot image contains "instance_uuid".
	if val, ok := myImage.Properties["instance_uuid"]; ok {
		myImageInfo.SourceVM = irs.IID{SystemId: fmt.Sprintf("%v", val)}
	}

	for key, val := range myImage.Properties {
		if key == "os_type" || key == "description" || key == "os_architecture" || key == "hypervisor_type" || key == "image_type" || key == "os_distro" || key == "os_version" {
			metadata := irs.KeyValue{
				Key:   strings.ToUpper(key),
				Value: fmt.Sprintf("%v", val),
			}
			myImageInfo.KeyValueList = append(myImageInfo.KeyValueList, metadata)
		}
	}

	//myImageInfo.KeyValueList = keyValueList
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

func (myImageHandler *NhnCloudMyImageHandler) getVMSpecType(vmSystemId string) (string, error) {
	cblogger.Info("NHN Cloud Driver: called getVMSpecType()")

	nhnVM, err := servers.Get(myImageHandler.VMClient, vmSystemId).Extract()
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

func (myImageHandler *NhnCloudMyImageHandler) getBootableVolumeID(vmSystemId string) (string, error) {
	cblogger.Info("NHN Cloud Driver: called getBootableVolumeID()")

	diskHandler := NhnCloudDiskHandler{
		RegionInfo:   myImageHandler.RegionInfo,
		VMClient:     myImageHandler.VMClient,
		VolumeClient: myImageHandler.VolumeClient,
	}

	nhnVolumeList, err := diskHandler.getNhnVolumeList()
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
				if strings.EqualFold(attachment.ServerID, vmSystemId) {
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

func (myImageHandler *NhnCloudMyImageHandler) isPublicImage(myImageIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called isPublicImage()")
	callLogInfo := getCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, myImageIID.SystemId, "isPublicImage()")

	start := call.Start()
	nhnImage, err := myImageHandler.getRawSnapshot(myImageIID) // Image Client
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud My Image Info. [%v]", err.Error())
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

func (myImageHandler *NhnCloudMyImageHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("NHN Cloud Driver: called ListIID()")
	callLogInfo := getCallLogScheme(myImageHandler.RegionInfo.Region, call.MYIMAGE, "ListMyImage()", "ListMyImage()")

	start := call.Start()

	var iidList []*irs.IID

	listOpts := images.ListOpts{
		Visibility: images.ImageVisibilityPrivate, // Note : Private image only
	}
	allPages, err := images.List(myImageHandler.ImageClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud Image pages. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return make([]*irs.IID, 0), newErr
	}
	nhnImageList, err := images.ExtractImages(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud Image List. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return make([]*irs.IID, 0), newErr
	}

	for _, nhnImage := range nhnImageList {
		var iid irs.IID
		iid.SystemId = nhnImage.ID
		iid.NameId = nhnImage.Name

		iidList = append(iidList, &iid)
	}

	LoggingInfo(callLogInfo, start)

	return iidList, nil
}
