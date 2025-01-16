// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2022.09.

package resources

import (
	"fmt"
	// "io/ioutil"
	// "os"
	"strconv"
	"strings"
	"time"
	// "github.com/davecgh/go-spew/spew"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	DefaultDiskSize	string = "10"
)

type NcpVpcDiskHandler struct {
	RegionInfo    idrv.RegionInfo
	VMClient      *vserver.APIClient
}

// Caution : Incase of NCP VPC, there must be a created VM to create a new disk volume.
func (diskHandler *NcpVpcDiskHandler) CreateDisk(diskReqInfo irs.DiskInfo) (irs.DiskInfo, error) {
	cblogger.Info("NCP VPC Driver: called CreateDisk()")	
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskReqInfo.IId.NameId, "CreateDisk()") // HisCall logging

	if strings.EqualFold(diskReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid Disk NameId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	
	instanceList, err := diskHandler.getNcpVMList()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NCP VPC Instacne List. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}

	instanceNo := *instanceList[0].ServerInstanceNo  // InstanceNo of any VM on the Zone 
	// cblogger.Infof("# instanceNo : [%v]", instanceNo)

	reqDiskType := diskReqInfo.DiskType  // Option : 'default', 'SSD' or 'HDD'
	reqDiskSize := diskReqInfo.DiskSize  // Range : 10~2000(GB)

	if strings.EqualFold(reqDiskType, "") || strings.EqualFold(reqDiskType, "default") {
		reqDiskType = "SSD"  // In case, Volume Type is not specified.
	}		
	if strings.EqualFold(reqDiskSize, "") || strings.EqualFold(reqDiskSize, "default") {
		reqDiskSize = DefaultDiskSize  // In case, Volume Size is not specified.
	} 
	
	// Covert String to Int32
	i, err := strconv.ParseInt(reqDiskSize, 10, 32)
	if err != nil {
		panic(err)
	}
	reqDiskSizeInt := int32(i)

	if reqDiskSizeInt < 10 || reqDiskSizeInt > 2000 {   // Range : 10~2000(GB)
		newErr := fmt.Errorf("Invalid Disk Size. Disk Size Must be between 10 and 2000(GB).")
		cblogger.Error(newErr.Error())
		return irs.DiskInfo{}, newErr		
	}

	// For Zone-based control!!
	var reqZoneId string
	if strings.EqualFold(diskReqInfo.Zone, "") {
		reqZoneId = diskHandler.RegionInfo.Zone
	} else {
		reqZoneId = diskReqInfo.Zone
	}

	storageReq := vserver.CreateBlockStorageInstanceRequest{
		RegionCode: 					ncloud.String(diskHandler.RegionInfo.Region),
		BlockStorageName: 				ncloud.String(diskReqInfo.IId.NameId),
		BlockStorageSize: 				&reqDiskSizeInt,						// *** Required (Not Optional)
		BlockStorageDiskDetailTypeCode: ncloud.String(reqDiskType),
		ServerInstanceNo: 				ncloud.String(instanceNo),				// *** Required (Not Optional)
		ZoneCode: 						ncloud.String(reqZoneId), // Apply Zone-based control!!
	}
	callLogStart := call.Start()
	result, err := diskHandler.VMClient.V2Api.CreateBlockStorageInstance(&storageReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create New Disk Volume. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if *result.TotalRows < 1 {
		newErr := fmt.Errorf("Failed to Find the Created New Disk Volume Info!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	} else {
		cblogger.Info("Succeeded in Creating the Block Storage Volume.")
	}

	newDiskIID := irs.IID{NameId: *result.BlockStorageInstanceList[0].BlockStorageName, SystemId: *result.BlockStorageInstanceList[0].BlockStorageInstanceNo}

	// Wait for Disk Creation Process finished
	curStatus, err := diskHandler.waitForDiskCreation(newDiskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait for the Disk Creation. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	cblogger.Infof("==> New Disk Volume Status : [%s]", curStatus)

	// Wait for Disk Attachment finished
	curStatus, waitErr := diskHandler.waitForDiskAttachment(newDiskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait for the Disk Attachement. [%v]", waitErr.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	cblogger.Infof("==> Disk Status : [%s]", string(curStatus))

	// Caution!!
	// Incase of NCP VPC, there must be a created VM to create a new disk volume.
	// Therefore, the status of the new disk volume is 'Attached' after creation.
    // ### Need to be 'Available' status after disk creation process like other CSP (with detachment). 
	isDetached, err := diskHandler.DetachDisk(newDiskIID, irs.IID{SystemId: instanceNo})
	if err != nil {
		newErr := fmt.Errorf("Failed to Detach the Disk Volume : [%v] ", err)
		cblogger.Error(newErr.Error())
		return irs.DiskInfo{}, newErr
	} else if !isDetached {
		newErr := fmt.Errorf("Failed to Detach the Disk Volume!!")
		cblogger.Error(newErr.Error())
		return irs.DiskInfo{}, newErr
	}

	newDiskInfo, err := diskHandler.GetDisk(newDiskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to the Get Disk Info!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	return newDiskInfo, nil
}

func (diskHandler *NcpVpcDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	cblogger.Info("NCP VPC Driver: called ListDisk()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, "ListDisk()", "ListDisk()") // HisCall logging

	storageReq := vserver.GetBlockStorageInstanceListRequest{
		RegionCode: ncloud.String(diskHandler.RegionInfo.Region),   // $$$ Caution!!
	}

	callLogStart := call.Start()
	result, err := diskHandler.VMClient.V2Api.GetBlockStorageInstanceList(&storageReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Block Storage List : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var diskInfoList []*irs.DiskInfo
	if len(result.BlockStorageInstanceList) < 1 {
		cblogger.Info("### Block Storage does Not Exist!!")
	} else {
		cblogger.Info("Succeeded in Getting Block Storage list from NCP VPC.")
		for _, storage := range result.BlockStorageInstanceList {
			storageInfo, err := diskHandler.mappingDiskInfo(*storage)
			if err != nil {
				newErr := fmt.Errorf("Failed to Map Block Storage Info : [%v]", err)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return nil, newErr
			}
			diskInfoList = append(diskInfoList, &storageInfo)
		}
	}
	// cblogger.Infof("# DiskInfo List count : [%d]", len(diskInfoList))
	return diskInfoList, nil
}

func (diskHandler *NcpVpcDiskHandler) GetDisk(diskIID irs.IID) (irs.DiskInfo, error) {
	cblogger.Info("NCP VPC Driver: called GetDisk()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskIID.SystemId, "GetDisk()") // HisCall logging

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}

	ncpDiskInfo, err := diskHandler.getNcpDiskInfo(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Disk Info : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}

	storageInfo, err := diskHandler.mappingDiskInfo(*ncpDiskInfo)
	if err != nil {
		newErr := fmt.Errorf("Failed to Map the Block Storage Info : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	return storageInfo, nil
}

func (diskHandler *NcpVpcDiskHandler) ChangeDiskSize(diskIID irs.IID, size string) (bool, error) {
	cblogger.Info("NCP VPC Driver: called ChangeDiskSize()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskIID.SystemId, "ChangeDiskSize()") // HisCall logging
	
	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	intSize, _ := strconv.Atoi(size)
	int32Size := int32(intSize)
	changeReq := vserver.ChangeBlockStorageVolumeSizeRequest	{
		RegionCode: 			ncloud.String(diskHandler.RegionInfo.Region),
		BlockStorageInstanceNo: ncloud.String(diskIID.SystemId),
		BlockStorageSize: 		&int32Size,
	}

	callLogStart := call.Start()
	result, err := diskHandler.VMClient.V2Api.ChangeBlockStorageVolumeSize(&changeReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Change the Block Storage Volume Size : %v", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if !strings.EqualFold(*result.ReturnMessage, "success") {
		newErr := fmt.Errorf("Failed to Change the Block Storage Volume Size!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	} else {
		cblogger.Info("Succeeded in Changing the Block Storage Volume Size.")
	}
	cblogger.Infof("\n# Chaneged Size : [%s](GB)", strconv.FormatInt(*result.BlockStorageInstanceList[0].BlockStorageSize/(1024*1024*1024), 10))	

	return true, nil
}

func (diskHandler *NcpVpcDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {
	cblogger.Info("NCP VPC Driver: called DeleteDisk()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskIID.SystemId, "DeleteDisk()") // HisCall logging

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	curStatus, err := diskHandler.GetDiskStatus(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Disk Status : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	if strings.EqualFold(string(curStatus), string(irs.DiskAttached)) {
		newErr := fmt.Errorf("The Block Storage is Attached to a VM. First Detach it before Deleting!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	delReq := vserver.DeleteBlockStorageInstancesRequest	{
		RegionCode: 				ncloud.String(diskHandler.RegionInfo.Region),
		BlockStorageInstanceNoList: []*string{ncloud.String(diskIID.SystemId),},
	}
	callLogStart := call.Start()
	result, err := diskHandler.VMClient.V2Api.DeleteBlockStorageInstances(&delReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the Block Storage : %v", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if !strings.EqualFold(*result.ReturnMessage, "success") {
		newErr := fmt.Errorf("Failed to Delete the Block Storage!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	} else {
		cblogger.Info("Succeeded in Deleting the Block Storage.")
	}

	return true, nil
}

func (diskHandler *NcpVpcDiskHandler) AttachDisk(diskIID irs.IID, vmIID irs.IID) (irs.DiskInfo, error) {
	cblogger.Info("NCP VPC Driver: called AttachDisk()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskIID.SystemId, "AttachDisk()") // HisCall logging

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	} else if strings.EqualFold(vmIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr	}
	
	curStatus, err := diskHandler.GetDiskStatus(diskIID)
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

	attachReq := vserver.AttachBlockStorageInstanceRequest{
		RegionCode: 				ncloud.String(diskHandler.RegionInfo.Region),
		ServerInstanceNo: 			ncloud.String(vmIID.SystemId),
		BlockStorageInstanceNo: 	ncloud.String(diskIID.SystemId),
	}
	callLogStart := call.Start()
	result, err := diskHandler.VMClient.V2Api.AttachBlockStorageInstance(&attachReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Attach the Block Storage : %v", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if !strings.EqualFold(*result.ReturnMessage, "success") {
		newErr := fmt.Errorf("Failed to Attach the Block Storage!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	} else {
		cblogger.Info("Succeeded in Attaching the Block Storage.")
	}

	// Wait for Disk Attachment finished
	curStatus, waitErr := diskHandler.waitForDiskAttachment(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait for the Disk Attachment. [%v]", waitErr.Error())
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

func (diskHandler *NcpVpcDiskHandler) DetachDisk(diskIID irs.IID, ownerVM irs.IID) (bool, error) {
	cblogger.Info("NCP VPC Driver: called DetachDisk()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskIID.SystemId, "DetachDisk()") // HisCall logging

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	curStatus, err := diskHandler.GetDiskStatus(diskIID)
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

	isBasicBlockStorage, err := diskHandler.isBasicBlockStorage(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Disk Info. : [%v] ", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	} else if isBasicBlockStorage {
		newErr := fmt.Errorf("Failed to Detach the Disk Volume. The Disk is Basic(Bootable) Disk Volume and Attached'.")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	detachReq := vserver.DetachBlockStorageInstancesRequest{
		RegionCode: 				ncloud.String(diskHandler.RegionInfo.Region),
		BlockStorageInstanceNoList: []*string{ncloud.String(diskIID.SystemId),},
	}
	callLogStart := call.Start()
	result, err := diskHandler.VMClient.V2Api.DetachBlockStorageInstances(&detachReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Detach the Block Storage : %v", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if !strings.EqualFold(*result.ReturnMessage, "success") {
		newErr := fmt.Errorf("Failed to Detach the Block Storage!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	} else {
		cblogger.Info("Succeeded in Detaching the Block Storage.")
	}

	// Wait for Disk Detachment finished
	curStatus, waitErr := diskHandler.waitForDiskDetachment(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Wait to Get Disk Info. [%v]", waitErr.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	cblogger.Infof("==> Disk Status : [%s]", string(curStatus))

	return true, nil
}

func (diskHandler *NcpVpcDiskHandler) getNcpDiskInfo(diskIID irs.IID) (*vserver.BlockStorageInstance, error) {
	cblogger.Info("NCP VPC Cloud Driver: called GetNCPDiskInfo()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	storageReq := vserver.GetBlockStorageInstanceDetailRequest{
		RegionCode: 			ncloud.String(diskHandler.RegionInfo.Region),
		BlockStorageInstanceNo: ncloud.String(diskIID.SystemId),
	}
	result, err := diskHandler.VMClient.V2Api.GetBlockStorageInstanceDetail(&storageReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Block Storage Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	if len(result.BlockStorageInstanceList) < 1 {
		cblogger.Debug("### BlockStorageInstance does Not Exist!!")
		return nil, nil
	}
	return result.BlockStorageInstanceList[0], nil
}

// Waiting for up to 500 seconds during Disk creation until Disk info. can be get
func (diskHandler *NcpVpcDiskHandler) waitForDiskCreation(diskIID irs.IID) (irs.DiskStatus, error) {
	cblogger.Info("===> Since Disk info. cannot be retrieved immediately after Disk creation, it waits until running.")

	curRetryCnt := 0
	maxRetryCnt := 500
	for {
		curStatus, err := diskHandler.GetDiskStatus(diskIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Disk Status : [%v] ", err)
			cblogger.Error(newErr.Error())
			return "Failed. ", newErr
		} else {
			cblogger.Infof("Succeeded in Getting the Disk Status : [%s]", string(curStatus))
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
func (diskHandler *NcpVpcDiskHandler) waitForDiskAttachment(diskIID irs.IID) (irs.DiskStatus, error) {
	curRetryCnt := 0
	maxRetryCnt := 500
	for {
		curStatus, err := diskHandler.GetDiskStatus(diskIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Disk Status : [%v] ", err)
			cblogger.Error(newErr.Error())
			return "Failed. ", newErr
		} else {
			cblogger.Infof("Succeeded in Getting the Disk Status : [%s]", curStatus)
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

// Waiting for up to 500 seconds during Disk Attachment
func (diskHandler *NcpVpcDiskHandler) waitForDiskDetachment(diskIID irs.IID) (irs.DiskStatus, error) {
	curRetryCnt := 0
	maxRetryCnt := 500
	for {
		curStatus, err := diskHandler.GetDiskStatus(diskIID)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Disk Status : [%v] ", err)
			cblogger.Error(newErr.Error())
			return "Failed. ", newErr
		} else {
			cblogger.Infof("Succeeded in Getting the Disk Status : [%s]", curStatus)
		}

		cblogger.Infof("===> Disk Status : [%s]", string(curStatus))

		switch string(curStatus) {
		case string(irs.DiskCreating), string(irs.DiskAttached), string(irs.DiskDeleting), string(irs.DiskError), "Detaching", "Unknown" :
			curRetryCnt++
			cblogger.Infof("The Disk is still [%s], so wait for a second more during the Disk 'Detachment'.", string(curStatus))
			time.Sleep(time.Second * 2)
			if curRetryCnt > maxRetryCnt {
				newErr := fmt.Errorf("Despite waiting for a long time(%d sec), the Disk status is '%s', so it is forcibly finished.", maxRetryCnt, string(curStatus))
				cblogger.Error(newErr.Error())
				return "Failed. ", newErr
			}
		default:
			cblogger.Infof("===> ### The Disk 'Detachment' is finished, stopping the waiting.")
			return curStatus, nil
			//break
		}
	}
}

func (diskHandler *NcpVpcDiskHandler) GetDiskStatus(diskIID irs.IID) (irs.DiskStatus, error) {
	cblogger.Info("NHN Cloud Driver: called GetDiskStatus()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		return irs.DiskError, newErr
	}

	ncpDiskInfo, err := diskHandler.getNcpDiskInfo(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Disk Info : [%v] ", err)
		cblogger.Error(newErr.Error())
		return irs.DiskError, newErr
	}
	cblogger.Infof("# Disk Status of NCP VPC : [%s]", *ncpDiskInfo.BlockStorageInstanceStatusName)
	return convertDiskStatus(*ncpDiskInfo.BlockStorageInstanceStatusName), nil
}

func (diskHandler *NcpVpcDiskHandler) mappingDiskInfo(storage vserver.BlockStorageInstance) (irs.DiskInfo, error) {
	cblogger.Info("NCP VPC Driver: called mappingDiskInfo()")

	if strings.EqualFold(ncloud.StringValue(storage.BlockStorageInstanceNo), "") {
		newErr := fmt.Errorf("Invalid Block Storage Info!!")
		cblogger.Error(newErr.Error())
		return irs.DiskInfo{}, newErr
	}

	// cblogger.Info("\n\n### storage : ")
	// spew.Dump(storage)
	// cblogger.Info("\n")

	convertedTime, err := convertTimeFormat(*storage.CreateDate)
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert the Time Format!!")
		cblogger.Error(newErr.Error())
		return irs.DiskInfo{}, newErr
	}

	diskInfo := irs.DiskInfo{
		IId: irs.IID{
			NameId: 	ncloud.StringValue(storage.BlockStorageName),
			SystemId: 	ncloud.StringValue(storage.BlockStorageInstanceNo),
		},
		Zone:		 ncloud.StringValue(storage.ZoneCode),
		DiskSize:    strconv.FormatInt((*storage.BlockStorageSize)/(1024*1024*1024), 10),
		Status:		 convertDiskStatus(ncloud.StringValue(storage.BlockStorageInstanceStatusName)), // Not BlockStorageInstanceStatus.Code
		CreatedTime: convertedTime,
		DiskType: 	 ncloud.StringValue(storage.BlockStorageDiskDetailType.Code),
	}

	if strings.EqualFold(ncloud.StringValue(storage.BlockStorageInstanceStatusName), "attached") {
		vmHandler := NcpVpcVMHandler{
			RegionInfo:  	diskHandler.RegionInfo,
			VMClient:    	diskHandler.VMClient,
		}

		vmInfo, err := vmHandler.getNcpVMInfo(ncloud.StringValue(storage.ServerInstanceNo))
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the VM Info. : [%v] ", err)
			cblogger.Error(newErr.Error())
			return irs.DiskInfo{}, newErr
		}

		diskInfo.OwnerVM = irs.IID{
			NameId: 	ncloud.StringValue(vmInfo.ServerName),
			SystemId: 	ncloud.StringValue(storage.ServerInstanceNo),
		}
	}

	keyValueList := []irs.KeyValue{
		{Key: "DeviceName",   			Value: ncloud.StringValue(storage.DeviceName)},				
		// {Key: "ZoneCode",   			Value: ncloud.StringValue(storage.ZoneCode)},		 
		{Key: "BlockStorageType",   	Value: ncloud.StringValue(storage.BlockStorageType.CodeName)},
		{Key: "BlockStorageDiskType",  	Value: ncloud.StringValue(storage.BlockStorageDiskType.CodeName)},		
		{Key: "MaxIOPS",  				Value: strconv.FormatInt(int64(*storage.MaxIopsThroughput), 10)},
		{Key: "IsReturnProtection", 	Value: strconv.FormatBool(*storage.IsReturnProtection)},
		{Key: "IsEncryptedVolume", 		Value: strconv.FormatBool(*storage.IsEncryptedVolume)},		
	}
	diskInfo.KeyValueList = keyValueList

	return diskInfo, nil
}

func convertDiskStatus(diskStatus string) irs.DiskStatus {
	cblogger.Info("NCP VPC Cloud Driver: called convertDiskStatus()")
	
	var resultStatus irs.DiskStatus
	switch strings.ToLower(diskStatus) {
	case "creating":
		resultStatus = irs.DiskCreating
	case "detached":
		resultStatus = irs.DiskAvailable
	case "attached":
		resultStatus = irs.DiskAttached
	case "deleting":
		resultStatus = irs.DiskDeleting
	case "error":
		resultStatus = irs.DiskError
	case "detaching":
		resultStatus = "Detaching"		
	default:
		resultStatus = "Unknown"
	}

	return resultStatus
}

func (diskHandler *NcpVpcDiskHandler) getNcpVMList() ([]*vserver.ServerInstance, error) {
	cblogger.Info("Ncp VPC Cloud Driver: called getNcpVMList()")

	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, "getNcpVMList()", "getNcpVMList()")

	instanceReq := vserver.GetServerInstanceListRequest{
		RegionCode: 		&diskHandler.RegionInfo.Region,
	}
	callLogStart := call.Start()
	instanceResult, err := diskHandler.VMClient.V2Api.GetServerInstanceList(&instanceReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find the VM Instance list from NCP VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(instanceResult.ServerInstanceList) < 1 {
		cblogger.Info("### VM Instance does Not Exist!!")
		return nil, nil
	} else {
		cblogger.Info("Succeeded in Getting VM Instance list from NCP VPC")
	}
	return instanceResult.ServerInstanceList, nil
}

func (diskHandler *NcpVpcDiskHandler) isBasicBlockStorage(diskIID irs.IID) (bool, error) {
	cblogger.Info("NCP VPC Cloud Driver: called isBasicBlockStorage()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	ncpDiskInfo, err := diskHandler.getNcpDiskInfo(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Disk Info : [%v] ", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	if strings.EqualFold(*ncpDiskInfo.BlockStorageType.Code, "BASIC") {  // Ex) Basic, SVRBS, ...
		return true, nil
	} else {
		cblogger.Infof("# BlockStorageType : [%s]", *ncpDiskInfo.BlockStorageType.CodeName) // Ex) Basic BS, Server BS, ...
		return false, nil
	}
}

func (diskHandler *NcpVpcDiskHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("NCP VPC Driver: called diskHandler ListIID()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, "ListIID()", "ListIID()")

	storageReq := vserver.GetBlockStorageInstanceListRequest{
		RegionCode: ncloud.String(diskHandler.RegionInfo.Region),   // $$$ Caution!!
	}

	callLogStart := call.Start()
	result, err := diskHandler.VMClient.V2Api.GetBlockStorageInstanceList(&storageReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Block Storage List : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var iidList []*irs.IID
	if len(result.BlockStorageInstanceList) < 1 {
		cblogger.Debug("### BlockStorageInstance does Not Exist!!")
		return nil, nil
	} else {
		for _, storage := range result.BlockStorageInstanceList {
			var iid irs.IID
			iid.NameId = ncloud.StringValue(storage.BlockStorageName)
			iid.SystemId = ncloud.StringValue(storage.BlockStorageInstanceNo)
	
			iidList = append(iidList, &iid)
		}
	}
	return iidList, nil
}
