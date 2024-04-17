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
	"fmt"
	"strconv"
	"strings"
	"time"
	// "github.com/davecgh/go-spew/spew"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/blockstorage/extensions/volumeactions" // For Attachment
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/blockstorage/v2/volumes"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/extensions/volumeattach"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/servers"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	HDD string = "General HDD"
	SSD string = "General SSD"
)

type NhnCloudDiskHandler struct {
	RegionInfo   idrv.RegionInfo
	VMClient     *nhnsdk.ServiceClient
	VolumeClient *nhnsdk.ServiceClient
}

func (diskHandler *NhnCloudDiskHandler) CreateDisk(diskReqInfo irs.DiskInfo) (irs.DiskInfo, error) {
	cblogger.Info("NHN Cloud Driver: called CreateDisk()")
	callLogInfo := getCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskReqInfo.IId.NameId, "CreateDisk()")

	if strings.EqualFold(diskReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid Disk NameId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}

	if strings.EqualFold(diskReqInfo.Zone, "") {
		newErr := fmt.Errorf("Invalid Zone Info!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}

	reqDiskType := diskReqInfo.DiskType // 'default', 'General_HDD' or 'General_SSD'
	reqDiskSize := diskReqInfo.DiskSize // 10~2000(GB)

	if strings.EqualFold(reqDiskType, "") || strings.EqualFold(reqDiskType, "default") {
		reqDiskType = HDD // In case, Volume Type is not specified.
	} else if strings.EqualFold(reqDiskType, "General_HDD") {
		reqDiskType = HDD // "General HDD"
	} else if strings.EqualFold(reqDiskType, "General_SSD") {
		reqDiskType = SSD // "General SSD"
	} else {
		newErr := fmt.Errorf("Invalid Disk Type!!")
		cblogger.Error(newErr.Error())
	}

	if strings.EqualFold(reqDiskSize, "") || strings.EqualFold(reqDiskSize, "default") {
		reqDiskSize = DefaultDiskSize // In case, Volume Size is not specified.
	}

	reqDiskSizeInt, err := strconv.Atoi(reqDiskSize)
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert Disk Size to Int. type. [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	if reqDiskSizeInt < 10 || reqDiskSizeInt > 2000 { // 10~2000(GB)
		newErr := fmt.Errorf("Invalid Disk Size. Disk Size Must be between 10 and 2000.")
		cblogger.Error(newErr.Error())
		return irs.DiskInfo{}, newErr
	}

	start := call.Start()
	create0pts := volumes.CreateOpts{
		Size:             reqDiskSizeInt,
		AvailabilityZone: diskReqInfo.Zone,
		Name:             diskReqInfo.IId.NameId,
		VolumeType:       reqDiskType,
	}
	diskResult, err := volumes.Create(diskHandler.VolumeClient, create0pts).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Create New Disk Volume. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	// Because there are functions that use 'NameId', Input NameId too
	newDiskIID := irs.IID{NameId: diskResult.Name, SystemId: diskResult.ID}

	// Wait for created Disk info to be inquired
	curStatus, err := diskHandler.waitForDiskCreation(newDiskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait to Get Disk Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	cblogger.Infof("==> Disk Status of [%s] : [%s]", newDiskIID.NameId, curStatus)

	// Check VM Deploy Status
	diskResult, err = volumes.Get(diskHandler.VolumeClient, diskResult.ID).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the NHN Disk Info!! : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}

	newDiskInfo, err := diskHandler.mappingDiskInfo(*diskResult)
	if err != nil {
		newErr := fmt.Errorf("Failed to Map Disk Info. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	return newDiskInfo, nil
}

func (diskHandler *NhnCloudDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	cblogger.Info("NHN Cloud Driver: called ListDisk()")
	callLogInfo := getCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, "ListDisk()", "ListDisk()")

	start := call.Start()
	listOpts := volumes.ListOpts{}
	allPages, err := volumes.List(diskHandler.VolumeClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud Volume list!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	nhnVolumeList, err := volumes.ExtractVolumes(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Extract NHN Cloud Volume list!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, start)
	// spew.Dump(nhnVolumeList)

	var volumeInfoList []*irs.DiskInfo
	for _, volume := range nhnVolumeList {
		volumeInfo, err := diskHandler.mappingDiskInfo(volume)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get Disk Info list!! : [%v] ", err)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return nil, newErr
		}
		volumeInfoList = append(volumeInfoList, &volumeInfo)
	}
	return volumeInfoList, nil
}

func (diskHandler *NhnCloudDiskHandler) GetDisk(diskIID irs.IID) (irs.DiskInfo, error) {
	cblogger.Info("NHN Cloud Driver: called GetDisk()")
	callLogInfo := getCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskIID.SystemId, "GetDisk()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}

	nhnVolumeInfo, err := volumes.Get(diskHandler.VolumeClient, diskIID.SystemId).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the NHN Disk Info!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}

	volumeInfo, err := diskHandler.mappingDiskInfo(*nhnVolumeInfo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Disk Info!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	return volumeInfo, nil
}

func (diskHandler *NhnCloudDiskHandler) ChangeDiskSize(diskIID irs.IID, newDiskSize string) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called ChangeDiskSize()")
	callLogInfo := getCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskIID.SystemId, "ChangeDiskSize()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	newDiskSizeInt, err := strconv.Atoi(newDiskSize)
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert New Disk Size to Int. type. [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	diskInfo, err := diskHandler.GetDisk(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Disk Info!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	curDiskSizeInt, err := strconv.Atoi(diskInfo.DiskSize)
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert Current Disk Size to Int. type. [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	if newDiskSizeInt < 10 || newDiskSizeInt > 2000 { // 10~2000(GB)
		newErr := fmt.Errorf("Invalid Disk Size. Disk Size Must be between 10 and 2000.")
		cblogger.Error(newErr.Error())
		return false, newErr
	} else if newDiskSizeInt <= curDiskSizeInt {
		newErr := fmt.Errorf("Invalid Disk Size. New Disk Size must be Greater than Current Disk Size.")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	if diskInfo.Status != "Available" {
		newErr := fmt.Errorf("Disk Resizing is possible only when the Disk status is in 'Available'.")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	start := call.Start()
	extendOpts := volumeactions.ExtendSizeOpts{
		NewSize: newDiskSizeInt,
	}
	extendErr := volumeactions.ExtendSize(diskHandler.VolumeClient, diskIID.SystemId, extendOpts).ExtractErr()
	if err != nil {
		newErr := fmt.Errorf("Failed to Change the Disk Size!! : [%v] ", extendErr)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, start)

	return true, nil
}

func (diskHandler *NhnCloudDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called DeleteDisk()")
	callLogInfo := getCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, "DeleteDisk()", "DeleteDisk()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	curStatus, err := diskHandler.getDiskStatus(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Disk Status : [%v] ", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	} else if strings.EqualFold(string(curStatus), string(irs.DiskAttached)) {
		newErr := fmt.Errorf("Failed to Delete the Disk Volume. The Disk Status is 'Attached'.")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	start := call.Start()
	delOpts := volumes.DeleteOpts{
		Cascade: true, // Delete all snapshots of this volume as well.
	}
	delErr := volumes.Delete(diskHandler.VolumeClient, diskIID.SystemId, delOpts).ExtractErr()
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the Disk Volume!! : [%v] ", delErr)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, start)

	return true, nil
}

func (diskHandler *NhnCloudDiskHandler) AttachDisk(diskIID irs.IID, vmIID irs.IID) (irs.DiskInfo, error) {
	cblogger.Info("NHN Cloud Driver: called AttachDisk()")
	callLogInfo := getCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskIID.SystemId, "AttachDisk()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	} else if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}

	curStatus, err := diskHandler.getDiskStatus(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Disk Status : [%v] ", err)
		cblogger.Error(newErr.Error())
		return irs.DiskInfo{}, newErr
	} else if strings.EqualFold(string(curStatus), string(irs.DiskAttached)) {
		newErr := fmt.Errorf("Failed to Attach the Disk Volume. The Disk is already 'Attached'.")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}

	start := call.Start()
	createOpts := volumeattach.CreateOpts{
		VolumeID: diskIID.SystemId,
	}
	_, createErr := volumeattach.Create(diskHandler.VMClient, vmIID.SystemId, createOpts).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Attach the Disk Volume!! : [%v] ", createErr)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	// Wait for Disk Attachment finished
	curStatus, waitErr := diskHandler.waitForDiskAttachment(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait to Get Disk Info. [%v]", waitErr.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	cblogger.Infof("==> Disk Status : [%s]", string(curStatus))

	diskInfo, err := diskHandler.GetDisk(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Disk Info!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	return diskInfo, nil
}

func (diskHandler *NhnCloudDiskHandler) DetachDisk(diskIID irs.IID, vmIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called DetachDisk()")
	callLogInfo := getCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskIID.SystemId, "DetachDisk()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	} else if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	curStatus, err := diskHandler.getDiskStatus(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Disk Status : [%v] ", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	} else if !strings.EqualFold(string(curStatus), string(irs.DiskAttached)) {
		newErr := fmt.Errorf("Failed to Detach the Disk Volume. The Disk Status is Not 'Attached'.")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	isBootable, err := diskHandler.isBootableDisk(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Bootable Disk Info. : [%v] ", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	} else if isBootable {
		newErr := fmt.Errorf("Failed to Detach the Disk Volume. The Disk is 'Bootable Disk and Attached'.")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	start := call.Start()
	delErr := volumeattach.Delete(diskHandler.VMClient, vmIID.SystemId, diskIID.SystemId).ExtractErr()
	if err != nil {
		newErr := fmt.Errorf("Failed to Detach the Disk Volume!! : [%v] ", delErr)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, start)

	return true, nil
}

// Waiting for up to 500 seconds during Disk creation until Disk info. can be get
func (diskHandler *NhnCloudDiskHandler) waitForDiskCreation(diskIID irs.IID) (irs.DiskStatus, error) {
	cblogger.Info("===> Since Disk info. cannot be retrieved immediately after Disk creation, it waits until running.")

	curRetryCnt := 0
	maxRetryCnt := 500
	for {
		curStatus, err := diskHandler.getDiskStatus(diskIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Disk Status of [%s] : [%v] ", diskIID.NameId, err)
			cblogger.Error(newErr.Error())
			return "Failed. ", newErr
		} else {
			cblogger.Infof("Succeeded in Getting the Disk Status of [%s] : [%s]", diskIID.NameId, string(curStatus))
		}

		cblogger.Infof("===> Disk Status : [%s]", string(curStatus))

		switch string(curStatus) {
		case "Creating":
			curRetryCnt++
			cblogger.Infof("The Disk is still 'Creating', so wait for a second more before inquiring the Disk info.")
			time.Sleep(time.Second * 2)
			if curRetryCnt > maxRetryCnt {
				newErr := fmt.Errorf("Despite waiting for a long time(%d sec), the Disk status is %s, so it is forcibly finished.", maxRetryCnt, string(curStatus))
				cblogger.Error(newErr.Error())
				return "Failed. ", newErr
			}
		default:
			cblogger.Infof("===> ### The Disk 'Creation' is finished, stopping the waiting.")
			return curStatus, nil
			//break
		}
	}
}

// Waiting for up to 500 seconds during Disk Attachment
func (diskHandler *NhnCloudDiskHandler) waitForDiskAttachment(diskIID irs.IID) (irs.DiskStatus, error) {
	curRetryCnt := 0
	maxRetryCnt := 500
	for {
		curStatus, err := diskHandler.getDiskStatus(diskIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Disk Status of [%s] : [%v] ", diskIID.NameId, err)
			cblogger.Error(newErr.Error())
			return "Failed. ", newErr
		} else {
			cblogger.Infof("Succeeded in Getting the Disk Status of [%s] : [%s]", diskIID.NameId, curStatus)
		}

		cblogger.Infof("===> Disk Status : [%s]", string(curStatus))

		switch string(curStatus) {
		case string(irs.DiskCreating), string(irs.DiskAvailable), string(irs.DiskDeleting), string(irs.DiskError), "Unknown":
			curRetryCnt++
			cblogger.Infof("The Disk is still [%s], so wait for a second more during the Disk 'Attachment'.", string(curStatus))
			time.Sleep(time.Second * 2)
			if curRetryCnt > maxRetryCnt {
				newErr := fmt.Errorf("Despite waiting for a long time(%d sec), the Disk status is '%s', so it is forcibly finished.", maxRetryCnt, string(curStatus))
				cblogger.Error(newErr.Error())
				return "Failed. ", newErr
			}
		default:
			cblogger.Infof("===> ### The Disk 'Attachment' is finished, stopping the waiting.")
			return curStatus, nil
			//break
		}
	}
}

func (diskHandler *NhnCloudDiskHandler) getDiskStatus(diskIID irs.IID) (irs.DiskStatus, error) {
	cblogger.Info("NHN Cloud Driver: called getDiskStatus()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		return irs.DiskError, newErr
	}

	diskResult, err := volumes.Get(diskHandler.VolumeClient, diskIID.SystemId).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the NHN Disk Info!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.DiskError, newErr
	}

	cblogger.Infof("# diskResult.Status of NHN Cloud : [%s]", diskResult.Status)
	diskStatus := convertDiskStatus(diskResult.Status)

	return diskStatus, nil
}

func convertDiskStatus(diskStatus string) irs.DiskStatus {
	cblogger.Info("NHN Cloud Driver: called convertDiskStatus()")

	var resultStatus irs.DiskStatus
	switch strings.ToLower(diskStatus) {
	case "creating":
		resultStatus = irs.DiskCreating
	case "available":
		resultStatus = irs.DiskAvailable
	case "in-use":
		resultStatus = irs.DiskAttached
	case "deleting":
		resultStatus = irs.DiskDeleting
	case "error":
		resultStatus = irs.DiskError
	case "error_deleting":
		resultStatus = irs.DiskError
	case "error_backing-up":
		resultStatus = irs.DiskError
	case "error_restoring":
		resultStatus = irs.DiskError
	case "error_extending":
		resultStatus = irs.DiskError
	default:
		resultStatus = "Unknown"
	}

	return resultStatus
}

func (diskHandler *NhnCloudDiskHandler) isBootableDisk(diskIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called isBootableDisk()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	diskResult, err := volumes.Get(diskHandler.VolumeClient, diskIID.SystemId).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the NHN Disk Info!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	isBootable, err := strconv.ParseBool(diskResult.Bootable)
	if err != nil {
		newErr := fmt.Errorf("Failed to Parse the String value!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	return isBootable, nil
}

func (diskHandler *NhnCloudDiskHandler) mappingDiskInfo(volume volumes.Volume) (irs.DiskInfo, error) {
	cblogger.Info("NHN Cloud Driver: called mappingDiskInfo()")
	// cblogger.Info("\n\n### volume : ")
	// spew.Dump(volume)
	// cblogger.Info("\n")

	diskInfo := irs.DiskInfo{
		IId: irs.IID{
			SystemId: volume.ID,
		},
		Zone: 		 volume.AvailabilityZone,
		DiskSize:    strconv.Itoa(volume.Size),
		Status:      convertDiskStatus(volume.Status),
		CreatedTime: volume.CreatedAt,
	}

	if strings.EqualFold(volume.Name, "") { // Bootable disk of Not 'u2' VMSpec
		diskInfo.IId.NameId = "Auto_Created_Booting_Disk"
	} else {
		diskInfo.IId.NameId = volume.Name
	}

	if strings.EqualFold(volume.VolumeType, HDD) { // "General HDD"
		diskInfo.DiskType = "General_HDD"
	} else if strings.EqualFold(volume.VolumeType, SSD) { // "General SSD"
		diskInfo.DiskType = "General_SSD"
	}

	if volume.Attachments != nil && len(volume.Attachments) > 0 {
		for _, attachment := range volume.Attachments {
			nhnVm, err := servers.Get(diskHandler.VMClient, attachment.ServerID).Extract()
			if err != nil {
				newErr := fmt.Errorf("Failed to Get Volume Info list!! : [%v] ", err)
				cblogger.Error(newErr.Error())
				return irs.DiskInfo{}, newErr
			} else {
				diskInfo.OwnerVM = irs.IID{
					NameId:   nhnVm.Name,
					SystemId: nhnVm.ID,
				}
			}
		}
	}

	keyValueList := []irs.KeyValue{
		// {Key: "AvailabilityZone", Value: volume.AvailabilityZone},
		{Key: "IsBootable", Value: volume.Bootable},
		{Key: "IsMultiattached", Value: strconv.FormatBool(volume.Multiattach)},
		{Key: "IsEncrypted", Value: strconv.FormatBool(volume.Encrypted)},
	}
	diskInfo.KeyValueList = keyValueList

	return diskInfo, nil
}

func (diskHandler *NhnCloudDiskHandler) getNhnVolumeList() ([]volumes.Volume, error) {
	cblogger.Info("NHN Cloud Driver: called getNhnVolumeList()")
	callLogInfo := getCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, "getNhnVolumeList()", "getNhnVolumeList()")

	start := call.Start()
	listOpts := volumes.ListOpts{}
	allPages, err := volumes.List(diskHandler.VolumeClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NHN Cloud Volume Pages!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	nhnVolumeList, err := volumes.ExtractVolumes(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Extract NHN Cloud Volume list!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, start)
	// spew.Dump(nhnVolumeList)

	if len(nhnVolumeList) < 1 {
		newErr := fmt.Errorf("NHN Cloud Volume does Not Exist!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	return nhnVolumeList, nil
}
