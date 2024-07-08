// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2024.01.

package resources

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	// "github.com/davecgh/go-spew/spew"

	ktvpcsdk 	"github.com/cloud-barista/ktcloudvpc-sdk-go"
	volumes2 	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/blockstorage/v2/volumes"
	vattach "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/compute/v2/extensions/volumeattach"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/compute/v2/servers"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	DefaultDataDiskSize     	string = "100"
	DefaultDiskUsagePlanType	string = "hourly"
)

type KTVpcDiskHandler struct {
	RegionInfo    idrv.RegionInfo
	VMClient      *ktvpcsdk.ServiceClient
	VolumeClient  *ktvpcsdk.ServiceClient
}

func (diskHandler *KTVpcDiskHandler) CreateDisk(diskReqInfo irs.DiskInfo) (irs.DiskInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called CreateDisk()")	
	callLogInfo := getCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskReqInfo.IId.NameId, "CreateDisk()")

	if strings.EqualFold(diskReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid Disk NameId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}

	if strings.EqualFold(diskHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Invalid Zone Info!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}

	reqDiskType := diskReqInfo.DiskType  // 'default', 'HDD' or 'SSD'
	reqDiskSize := diskReqInfo.DiskSize  // 10~2000(GB)
	
	if strings.EqualFold(reqDiskType, "") || strings.EqualFold(reqDiskType, "default") {
		reqDiskType = "HDD"  // In case, Volume Type is not specified.
	} else if strings.EqualFold(reqDiskType, "HDD") {
		reqDiskType = "HDD"
	} else if strings.EqualFold(reqDiskType, "SSD") {
		reqDiskType = "SSD"
	} else {
		newErr := fmt.Errorf("Invalid Disk Type!!")
		cblogger.Error(newErr.Error())
	}

	if strings.EqualFold(reqDiskSize, "") || strings.EqualFold(reqDiskSize, "default") {
		reqDiskSize = DefaultDataDiskSize  // In case, Volume Size is not specified.
	} 
	
	reqDiskSizeInt, err := strconv.Atoi(reqDiskSize)
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert Disk Size to Int. type. [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	if reqDiskSizeInt < 10 || reqDiskSizeInt > 2000 {  // 10~2000(GB)
		newErr := fmt.Errorf("Invalid Disk Size. Disk Size Must be between 10 and 2000.")
		cblogger.Error(newErr.Error())
		return irs.DiskInfo{}, newErr
	}

	start := call.Start()
	create0pts := volumes2.CreateOpts{
		AvailabilityZone: 	diskHandler.RegionInfo.Zone,
		Size: 				reqDiskSizeInt,
		Name:				diskReqInfo.IId.NameId,
		UsagePlanType:		DefaultDiskUsagePlanType,
	}
	// cblogger.Info("\n### Disk create 0pts : ")
	// spew.Dump(create0pts)
	// cblogger.Info("\n")

	diskResult, err := volumes2.Create(diskHandler.VolumeClient, create0pts).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Create New Disk Volume. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	loggingInfo(callLogInfo, start)

	// Because there are functions that use 'NameId', Input NameId too
	newDiskIID := irs.IID{NameId: diskResult.Name, SystemId: diskResult.ID}

	// Wait for created VM info to be inquired
	curStatus, err := diskHandler.waitForDiskCreation(newDiskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait to Get Disk Info. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	cblogger.Infof("==> Disk Status of [%s] : [%s]", newDiskIID.NameId, curStatus)

	// Check VM Deploy Status
	diskResult, err = volumes2.Get(diskHandler.VolumeClient, diskResult.ID).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the KT Disk Info!! : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}

	newDiskInfo, err := diskHandler.mappingDiskInfo(*diskResult)
	if err != nil {
		newErr := fmt.Errorf("Failed to Map Disk Info. : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	return newDiskInfo, nil
}

func (diskHandler *KTVpcDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called ListDisk()")
	callLogInfo := getCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, "ListDisk()", "ListDisk()")

	start := call.Start()
	listOpts :=	volumes2.ListOpts{}
	allPages, err := volumes2.List(diskHandler.VolumeClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud Volume list!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	ktVolumeList, err := volumes2.ExtractVolumes(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Extract KT Cloud Volume list!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	loggingInfo(callLogInfo, start)
	// spew.Dump(ktVolumeList)

	var volumeInfoList []*irs.DiskInfo
	for _, volume := range ktVolumeList {
		volumeInfo, err := diskHandler.mappingDiskInfo(volume)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get Disk Info list!! : [%v] ", err)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return nil, newErr
		}
		volumeInfoList = append(volumeInfoList, &volumeInfo)
	}
	return volumeInfoList, nil
}

func (diskHandler *KTVpcDiskHandler) GetDisk(diskIID irs.IID) (irs.DiskInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetDisk()")
	callLogInfo := getCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskIID.SystemId, "GetDisk()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}

	ktVolumeInfo, err := volumes2.Get(diskHandler.VolumeClient, diskIID.SystemId).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the KT Disk Info!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	// cblogger.Info(ktVolumeInfo)
	// spew.Dump(ktVolumeInfo)

	volumeInfo, err := diskHandler.mappingDiskInfo(*ktVolumeInfo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Disk Info!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	return volumeInfo, nil
}

func (diskHandler *KTVpcDiskHandler) ChangeDiskSize(diskIID irs.IID, newDiskSize string) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called ChangeDiskSize()")

	return false, fmt.Errorf("Does not support ChangeDiskSize() yet!!")
}

func (diskHandler *KTVpcDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called DeleteDisk()")
	callLogInfo := getCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, "DeleteDisk()", "DeleteDisk()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	// curStatus, err := diskHandler.getDiskStatus(diskIID)
	// if err != nil {
	// 	newErr := fmt.Errorf("Failed to Get the Disk Status : [%v] ", err)
	// 	cblogger.Error(newErr.Error())
	// 	return false, newErr
	// } else if strings.EqualFold(string(curStatus), string(irs.DiskAttached)) {
	// 	newErr := fmt.Errorf("Failed to Delete the Disk Volume. The Disk Status is 'Attached'.")
	// 	cblogger.Error(newErr.Error())
	// 	loggingError(callLogInfo, newErr)
	// 	return false, newErr
	// }

	start := call.Start()
	delOpts := volumes2.DeleteOpts {		
		Cascade : true, // Delete all snapshots of this volume as well.
	}
	delErr := volumes2.Delete(diskHandler.VolumeClient, diskIID.SystemId, delOpts).ExtractErr()
	if delErr != nil {
		newErr := fmt.Errorf("Failed to Delete the Disk Volume!! : [%v] ", delErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}
	loggingInfo(callLogInfo, start)

	return true, nil
}

func (diskHandler *KTVpcDiskHandler) AttachDisk(diskIID irs.IID, vmIID irs.IID) (irs.DiskInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called AttachDisk()")
	callLogInfo := getCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskIID.SystemId, "AttachDisk()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	} else if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
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
		loggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}

	start := call.Start()
	createOpts := vattach.CreateOpts{
		VolumeID: diskIID.SystemId,
	}	
	_, createErr := vattach.Create(diskHandler.VMClient, vmIID.SystemId, createOpts).Extract()
	if createErr != nil {
		newErr := fmt.Errorf("Failed to Attach the Disk Volume!! : [%v] ", createErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	loggingInfo(callLogInfo, start)

	// Wait for Disk Attachment finished
	curStatus, waitErr := diskHandler.waitForDiskAttachment(diskIID)
	if waitErr != nil {
		newErr := fmt.Errorf("Failed to Wait to Get Disk Info. [%v]", waitErr.Error())
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	cblogger.Infof("==> Disk Status : [%s]", string(curStatus))

	diskInfo, err := diskHandler.GetDisk(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Disk Info!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	return diskInfo, nil
}

func (diskHandler *KTVpcDiskHandler) DetachDisk(diskIID irs.IID, vmIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called DetachDisk()")
	callLogInfo := getCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskIID.SystemId, "DetachDisk()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	} else if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
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
		loggingError(callLogInfo, newErr)
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
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	start := call.Start()
	delErr := vattach.Delete(diskHandler.VMClient, vmIID.SystemId, diskIID.SystemId).ExtractErr()
	if delErr != nil {
		newErr := fmt.Errorf("Failed to Detach the Disk Volume!! : [%v] ", delErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}
	loggingInfo(callLogInfo, start)

	return true, nil
}

// Waiting for up to 500 seconds during Disk creation until Disk info. can be get
func (diskHandler *KTVpcDiskHandler) waitForDiskCreation(diskIID irs.IID) (irs.DiskStatus, error) {
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
func (diskHandler *KTVpcDiskHandler) waitForDiskAttachment(diskIID irs.IID) (irs.DiskStatus, error) {
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
		case string(irs.DiskCreating), string(irs.DiskAvailable), string(irs.DiskDeleting), string(irs.DiskError), "Unknown" :
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

func (diskHandler *KTVpcDiskHandler) getDiskStatus(diskIID irs.IID) (irs.DiskStatus, error) {
	cblogger.Info("KT Cloud VPC Driver: called getDiskStatus()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		return irs.DiskError, newErr
	}

	diskResult, err := volumes2.Get(diskHandler.VolumeClient, diskIID.SystemId).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the KT Disk Info!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.DiskError, newErr
	}

	cblogger.Infof("# diskResult.Status of KT Cloud : [%s]", diskResult.Status)
	diskStatus := convertDiskStatus(diskResult.Status)

	return diskStatus, nil
}

func convertDiskStatus(diskStatus string) irs.DiskStatus {
	cblogger.Info("KT Cloud VPC Driver: called convertDiskStatus()")
	
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

func (diskHandler *KTVpcDiskHandler) isBootableDisk(diskIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called isBootableDisk()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	diskResult, err := volumes2.Get(diskHandler.VolumeClient, diskIID.SystemId).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the KT Disk Info!! : [%v]", err)
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

func (diskHandler *KTVpcDiskHandler) mappingDiskInfo(volume volumes2.Volume) (irs.DiskInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called mappingDiskInfo()")		
	// cblogger.Info("\n\n### volume : ")
	// spew.Dump(volume)
	// cblogger.Info("\n")

	// Convert to KTC
    convertedTime, err := convertTimeToKTC(volume.CreatedAt)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Converted Time. [%v]", err)
		return irs.DiskInfo{}, newErr
	}

	diskInfo := irs.DiskInfo{
		IId: irs.IID{
			SystemId: volume.ID,
		},
		Zone: 		 volume.AvailabilityZone,
		DiskSize:    strconv.Itoa(volume.Size),
		Status:		 convertDiskStatus(volume.Status),
		CreatedTime: convertedTime,
	}

	if strings.EqualFold(volume.Name, "") {  // Bootable disk of Not 'u2' VMSpec
		diskInfo.IId.NameId = "Auto_Created_Booting_Disk"
	} else {
		diskInfo.IId.NameId = volume.Name
	}

	if strings.Contains(volume.VolumeType, "HDD") {  // Ex) volume.VolumeType : "HDD-2000iops"
		diskInfo.DiskType = "HDD"
	} else if strings.Contains(volume.VolumeType, "SSD") {
		diskInfo.DiskType = "SSD"
	}

	if volume.Attachments != nil && len(volume.Attachments) > 0 {
		for _, attachment := range volume.Attachments {			
			// ### Because of the abnormal cases of KT Cloud Volume aftger VM Terminateon(Except Not exist VM)
			ktVm, err := servers.Get(diskHandler.VMClient, attachment.ServerID).Extract()
			if err != nil {
				newErr := fmt.Errorf("Failed to Get Volume Info list!! : [%v] ", err)
				cblogger.Error(newErr.Error())
				// return irs.DiskInfo{}, newErr
			} else if !strings.EqualFold(ktVm.Name, "")  {
				diskInfo.OwnerVM = irs.IID{
					NameId:   ktVm.Name,
					SystemId: attachment.ServerID,
				}
			}
		}
	}

	keyValueList := []irs.KeyValue{
		// {Key: "AvailabilityZone",   Value: volume.AvailabilityZone},		 
		{Key: "IsBootable",   		Value: volume.Bootable},
		{Key: "IsMultiattached", 	Value: strconv.FormatBool(volume.Multiattach)},
		{Key: "IsEncrypted", 		Value: strconv.FormatBool(volume.Encrypted)},
	}

	// Check if 'Image Name' value exists and add it to the key/value list
	keyValue := irs.KeyValue{}		
	if imageName, exists := volume.VolumeImageMetadata["image_name"]; exists {
        // fmt.Printf("Image Name: %s\n", imageName)
		keyValue = irs.KeyValue{Key: "ImageName", Value: imageName}		
    } else {
		cblogger.Info("Image Name not found in volume info.")
    }
	keyValueList = append(keyValueList, keyValue)
	diskInfo.KeyValueList = keyValueList

	return diskInfo, nil
}

func (diskHandler *KTVpcDiskHandler) getImageNameandIDWithDiskID(diskId string) (irs.IID, error) {
	cblogger.Info("KT Cloud VPC Driver: called getImageNameandIDWithDiskID()")

	if strings.EqualFold(diskId, "") {
		newErr := fmt.Errorf("Invalid Disk ID!!")
		cblogger.Error(newErr.Error())
		return irs.IID{}, newErr
	}

	diskResult, err := volumes2.Get(diskHandler.VolumeClient, diskId).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the KT Disk Info!! : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.IID{}, newErr
	}

	var imageIID irs.IID
	bootable, err := strconv.ParseBool(diskResult.Bootable)
	if err != nil {
		newErr := fmt.Errorf("Failed to Parse the String to Bool!! : [%v]", err)
		return irs.IID{}, newErr
	}
	if bootable {
		// Extract the image name
		imageName, ok := diskResult.VolumeImageMetadata["image_name"]
		if !ok {
			cblogger.Info("Image Name not found")
		}

		// Extract the image id
		imageId, ok := diskResult.VolumeImageMetadata["image_id"]
		if !ok {
			cblogger.Info("Image ID not found")
		}
		
		if !strings.EqualFold(imageName, "") && !strings.EqualFold(imageId, "") {
			imageIID.NameId   = imageName
			imageIID.SystemId = imageId
		} else {
			newErr := fmt.Errorf("Failed to Get the KT Disk Info!! : [%v]", err)
			cblogger.Error(newErr.Error())
			return irs.IID{}, newErr
		}
	} else {
		newErr := fmt.Errorf("The Disk Volume is Not Bootable!!")
		return irs.IID{}, newErr
	}

	return imageIID, nil
}

func (diskHandler *KTVpcDiskHandler) getKtVolumeList() ([]volumes2.Volume, error) {
	cblogger.Info("KT Cloud VPC Driver: called getKtVolumeList()")
	callLogInfo := getCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, "getKtVolumeList()", "getKtVolumeList()")

	start := call.Start()
	listOpts :=	volumes2.ListOpts{}
	allPages, err := volumes2.List(diskHandler.VolumeClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud Volume Pages!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	ktVolumeList, err := volumes2.ExtractVolumes(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Extract KT Cloud Volume list!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	loggingInfo(callLogInfo, start)
	// spew.Dump(ktVolumeList)

	if len(ktVolumeList) < 1 {
		newErr := fmt.Errorf("KT Cloud Volume does Not Exist!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	return ktVolumeList, nil
}
