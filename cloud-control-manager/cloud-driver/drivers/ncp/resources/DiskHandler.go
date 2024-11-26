// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2023.07.

package resources

import (
	// "errors"
	"errors"
	"fmt"
	// "io/ioutil"
	// "os"
	"strconv"
	"strings"
	"time"
	// "github.com/davecgh/go-spew/spew"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/server"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	DefaultDiskSize string = "10"
)

type NcpDiskHandler struct {
	RegionInfo idrv.RegionInfo
	VMClient   *server.APIClient
}

// Caution : Incase of NCP, there must be a created VM to create a new disk volume.
func (diskHandler *NcpDiskHandler) CreateDisk(diskReqInfo irs.DiskInfo) (irs.DiskInfo, error) {
	cblogger.Info("NCP Driver: called CreateDisk()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Zone, call.DISK, diskReqInfo.IId.NameId, "CreateDisk()")

	if strings.EqualFold(diskReqInfo.IId.NameId, "") {
		rtnErr := logAndReturnError(callLogInfo, "Invalid Disk NameId!!", "")
		return irs.DiskInfo{}, rtnErr
	}

	var reqZoneId string
	if strings.EqualFold(diskReqInfo.Zone, "") {
		reqZoneId = diskHandler.RegionInfo.Zone
	} else {
		reqZoneId = diskReqInfo.Zone
	}
	// $$$ At least one VM is required to create new disk volume in case of NCP.
	instanceList, err := diskHandler.GetNcpVMListWithZone(reqZoneId)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NCP Instacne List :", err)
		return irs.DiskInfo{}, rtnErr
	}
	var instanceNo string
	if len(instanceList) < 1 {		
		rtnErr := logAndReturnError(callLogInfo, "At least one VM is required on a zone to create new disk volume in case of NCP.", "")
		cblogger.Error("### There is no VM in the zone.!!")
		return irs.DiskInfo{}, rtnErr
	} else {
		instanceNo = *instanceList[0].ServerInstanceNo // InstanceNo of any VM on the Zone
		cblogger.Infof("# VM instanceNo : [%v]", instanceNo)
	}

	reqDiskType := diskReqInfo.DiskType // Option : 'default', 'SSD' or 'HDD'
	reqDiskSize := diskReqInfo.DiskSize // Range : 10~2000(GB)

	if strings.EqualFold(reqDiskType, "") || strings.EqualFold(reqDiskType, "default") {
		reqDiskType = "SSD" // In case, Volume Type is not specified.
	}
	if strings.EqualFold(reqDiskSize, "") || strings.EqualFold(reqDiskSize, "default") {
		reqDiskSize = DefaultDiskSize // In case, Volume Size is not specified.
	}

	// Covert String to Int64
	reqDiskSizeInt, err := strconv.ParseInt(reqDiskSize, 10, 64) // Caution : Need 64bit int
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Parse to Get 64bit int. : ", err)
		return irs.DiskInfo{}, rtnErr
	}
	if reqDiskSizeInt < 10 || reqDiskSizeInt > 2000 { // Range : 10~2000(GB)
		rtnErr := logAndReturnError(callLogInfo, "Invalid Disk Size. Disk Size Must be between 10 and 2000(GB).", "")
		return irs.DiskInfo{}, rtnErr
	}

	storageReq := server.CreateBlockStorageInstanceRequest{
		BlockStorageName:   ncloud.String(diskReqInfo.IId.NameId),
		BlockStorageSize:   &reqDiskSizeInt,           // *** Required (Not Optional
		ServerInstanceNo:   ncloud.String(instanceNo), // *** Required (Not Optional)
		DiskDetailTypeCode: ncloud.String(reqDiskType),
	}
	callLogStart := call.Start()
	result, err := diskHandler.VMClient.V2Api.CreateBlockStorageInstance(&storageReq)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Create New Disk Volume. : ", err)
		return irs.DiskInfo{}, rtnErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.BlockStorageInstanceList) < 1 {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Find the Created New Disk Volume Info!!", "")
		return irs.DiskInfo{}, rtnErr
	} else {
		cblogger.Info("Succeeded in Creating New Block Storage Volume.")
	}

	newDiskIID := irs.IID{NameId: *result.BlockStorageInstanceList[0].BlockStorageName, SystemId: *result.BlockStorageInstanceList[0].BlockStorageInstanceNo}
	// Wait for Disk Creation Process finished
	curStatus, err := diskHandler.WaitForDiskCreation(newDiskIID)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Wait for the Disk Creation. : ", err)
		return irs.DiskInfo{}, rtnErr
	}
	cblogger.Infof("==> New Disk Volume Status : [%s]", curStatus)

	// Wait for Disk Attachment finished
	curStatus, waitErr := diskHandler.WaitForDiskAttachment(newDiskIID)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Wait for the Disk Attachement. : ", waitErr)
		return irs.DiskInfo{}, rtnErr
	}
	cblogger.Infof("==> Disk Status : [%s]", string(curStatus))

	// Caution!!
	// Incase of NCP, there must be a created VM to create a new disk volume.
	// Therefore, the status of the new disk volume is 'Attached' after creation.
	// ### Need to be 'Available' status after disk creation process like other CSP (with detachment).
	isDetached, err := diskHandler.DetachDisk(newDiskIID, irs.IID{SystemId: instanceNo})
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Detach the Disk Volume : ", err)
		return irs.DiskInfo{}, rtnErr
	} else if !isDetached {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Detach the Disk Volume!!", "")
		return irs.DiskInfo{}, rtnErr
	}

	newDiskInfo, err := diskHandler.GetDisk(newDiskIID)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to the Get Disk Info!! : ", err)
		return irs.DiskInfo{}, rtnErr
	}
	return newDiskInfo, nil
}

func (diskHandler *NcpDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	cblogger.Info("NCP Driver: called ListDisk()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Zone, call.DISK, "ListDisk()", "ListDisk()")

	vmHandler := NcpVMHandler{
		RegionInfo: diskHandler.RegionInfo,
		VMClient:   diskHandler.VMClient,
	}
	zoneNo, err := vmHandler.getZoneNo(diskHandler.RegionInfo.Region, diskHandler.RegionInfo.Zone) // Region/Zone info of diskHandler
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NCP Zone No of the Zone Code : ", err)
		return nil, rtnErr
	}
	storageReq := server.GetBlockStorageInstanceListRequest{
		ZoneNo: zoneNo, // $$$ Caution!! : Not ZoneCode
	}
	callLogStart := call.Start()
	result, err := diskHandler.VMClient.V2Api.GetBlockStorageInstanceList(&storageReq)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get Block Storage List from NCP : ", err)
		return nil, rtnErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var diskInfoList []*irs.DiskInfo
	if len(result.BlockStorageInstanceList) < 1 {
		cblogger.Info("### Block Storage does Not Exist!!")
	} else {
		cblogger.Info("Succeeded in Getting Block Storage list from NCP.")
		for _, storage := range result.BlockStorageInstanceList {
			storageInfo, err := diskHandler.MappingDiskInfo(*storage)
			if err != nil {
				rtnErr := logAndReturnError(callLogInfo, "Failed to Map Block Storage Info : ", err)
				return nil, rtnErr
			}
			diskInfoList = append(diskInfoList, &storageInfo)
		}
	}
	// cblogger.Infof("# DiskInfo List count : [%d]", len(diskInfoList))
	return diskInfoList, nil
}

func (diskHandler *NcpDiskHandler) GetDisk(diskIID irs.IID) (irs.DiskInfo, error) {
	cblogger.Info("NCP Driver: called GetDisk()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskIID.SystemId, "GetDisk()")

	if strings.EqualFold(diskIID.SystemId, "") {
		rtnErr := logAndReturnError(callLogInfo, "Invalid Disk SystemId!!", "")
		return irs.DiskInfo{}, rtnErr
	}

	diskInfo, err := diskHandler.GetNcpDiskInfo(diskIID)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get the Disk Info : ", err)
		return irs.DiskInfo{}, rtnErr
	}

	storageInfo, err := diskHandler.MappingDiskInfo(*diskInfo)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Map the Block Storage Info : ", err)
		return irs.DiskInfo{}, rtnErr
	}
	return storageInfo, nil
}

func (diskHandler *NcpDiskHandler) ChangeDiskSize(diskIID irs.IID, size string) (bool, error) {
	cblogger.Info("NCP Driver: called ChangeDiskSize()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskIID.SystemId, "ChangeDiskSize()")

	if strings.EqualFold(diskIID.SystemId, "") {
		rtnErr := logAndReturnError(callLogInfo, "Invalid Disk SystemId!!", "")
		return false, rtnErr
	}

	curStatus, err := diskHandler.GetDiskStatus(diskIID)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get the Disk Status : ", err)
		return false, rtnErr
	} else if strings.EqualFold(string(curStatus), string(irs.DiskAttached)) {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Change the Disk Size. The Disk Status is 'Attached'. Change Size after Detaching.", "")
		return false, rtnErr
	}

	// Covert String to Int64
	reqDiskSizeInt, err := strconv.ParseInt(size, 10, 64) // Caution : Need 64bit int
	if err != nil {
		panic(err)
	}
	if reqDiskSizeInt < 10 || reqDiskSizeInt > 2000 { // Range : 10~2000(GB)
		rtnErr := logAndReturnError(callLogInfo, "Invalid Disk Size. Disk Size Must be between 10 and 2000(GB).", "")
		return false, rtnErr
	}
	changeReq := server.ChangeBlockStorageVolumeSizeRequest{
		BlockStorageInstanceNo: ncloud.String(diskIID.SystemId),
		BlockStorageSize:       &reqDiskSizeInt,
	}
	callLogStart := call.Start()
	result, err := diskHandler.VMClient.V2Api.ChangeBlockStorageVolumeSize(&changeReq)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Change the Block Storage Volume Size : ", err)
		return false, rtnErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if !strings.EqualFold(*result.ReturnMessage, "success") {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Change the Block Storage Volume Size!!", "")
		return false, rtnErr
	} else {
		cblogger.Info("Succeeded in Changing the Block Storage Volume Size.")
	}
	cblogger.Infof("# Chaneged Size : %s(GB)", strconv.FormatInt(*result.BlockStorageInstanceList[0].BlockStorageSize/(1024*1024*1024), 10))

	return true, nil
}

func (diskHandler *NcpDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {
	cblogger.Info("NCP Driver: called DeleteDisk()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskIID.SystemId, "DeleteDisk()")

	if strings.EqualFold(diskIID.SystemId, "") {
		rtnErr := logAndReturnError(callLogInfo, "Invalid Disk SystemId!!", "")
		return false, rtnErr
	}

	isBasicBlockStorage, err := diskHandler.IsBasicBlockStorage(diskIID)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get the Disk Info. : ", err)
		return false, rtnErr
	} else if isBasicBlockStorage {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Delete the Disk Volume. The Disk is Basic(Bootable) Disk Volume.", "")
		return false, rtnErr
	}

	curStatus, err := diskHandler.GetDiskStatus(diskIID)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get the Disk Status : ", err)
		return false, rtnErr
	}
	if strings.EqualFold(string(curStatus), string(irs.DiskAttached)) {
		rtnErr := logAndReturnError(callLogInfo, "The Block Storage is Attached to a VM. First Detach it before Deleting!!", "")
		return false, rtnErr
	}

	delReq := server.DeleteBlockStorageInstancesRequest{
		BlockStorageInstanceNoList: []*string{ncloud.String(diskIID.SystemId)},
	}
	callLogStart := call.Start()
	result, err := diskHandler.VMClient.V2Api.DeleteBlockStorageInstances(&delReq)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Delete the Block Storage : ", err)
		return false, rtnErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if !strings.EqualFold(*result.ReturnMessage, "success") {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Delete the Block Storage!!", "")
		return false, rtnErr
	} else {
		cblogger.Info("Succeeded in Deleting the Block Storage.")
	}

	return true, nil
}

func (diskHandler *NcpDiskHandler) AttachDisk(diskIID irs.IID, vmIID irs.IID) (irs.DiskInfo, error) {
	cblogger.Info("NCP Driver: called AttachDisk()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskIID.SystemId, "AttachDisk()")

	if strings.EqualFold(diskIID.SystemId, "") {
		rtnErr := logAndReturnError(callLogInfo, "Invalid Disk SystemId!!", "")
		return irs.DiskInfo{}, rtnErr
	} else if strings.EqualFold(vmIID.SystemId, "") {
		rtnErr := logAndReturnError(callLogInfo, "Invalid VM SystemId!!", "")
		return irs.DiskInfo{}, rtnErr
	}

	curStatus, err := diskHandler.GetDiskStatus(diskIID)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get the Disk Status : ", err)
		return irs.DiskInfo{}, rtnErr
	} else if strings.EqualFold(string(curStatus), string(irs.DiskAttached)) {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Attach the Disk Volume. The Disk is already 'Attached'.", "")
		return irs.DiskInfo{}, rtnErr
	}

	attachReq := server.AttachBlockStorageInstanceRequest{
		ServerInstanceNo:       ncloud.String(vmIID.SystemId),
		BlockStorageInstanceNo: ncloud.String(diskIID.SystemId),
	}
	callLogStart := call.Start()
	result, err := diskHandler.VMClient.V2Api.AttachBlockStorageInstance(&attachReq)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Attach the Block Storage : ", err)
		return irs.DiskInfo{}, rtnErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if !strings.EqualFold(*result.ReturnMessage, "success") {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Attach the Block Storage!!", "")
		return irs.DiskInfo{}, rtnErr
	} else {
		cblogger.Info("Succeeded in Attaching the Block Storage.")
	}

	// Wait for Disk Attachment finished
	curStatus, waitErr := diskHandler.WaitForDiskAttachment(diskIID)
	if waitErr != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Wait for the Disk Attachment. : ", waitErr)
		return irs.DiskInfo{}, rtnErr
	}
	cblogger.Infof("==> Disk Status : [%s]", string(curStatus))

	diskInfo, err := diskHandler.GetDisk(diskIID)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get Disk Info!! : ", err)
		return irs.DiskInfo{}, rtnErr
	}
	return diskInfo, nil
}

func (diskHandler *NcpDiskHandler) DetachDisk(diskIID irs.IID, ownerVM irs.IID) (bool, error) {
	cblogger.Info("NCP Driver: called DetachDisk()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskIID.SystemId, "DetachDisk()")

	if strings.EqualFold(diskIID.SystemId, "") {
		rtnErr := logAndReturnError(callLogInfo, "Invalid Disk SystemId!!", "")
		return false, rtnErr
	}

	curStatus, err := diskHandler.GetDiskStatus(diskIID)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get the Disk Status : ", err)
		return false, rtnErr
	} else if !strings.EqualFold(string(curStatus), string(irs.DiskAttached)) {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Detach the Disk Volume. The Disk Status is Not 'Attached'", "")
		return false, rtnErr
	}

	isBasicBlockStorage, err := diskHandler.IsBasicBlockStorage(diskIID)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get the Disk Info. : ", err)
		return false, rtnErr
	} else if isBasicBlockStorage {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Detach the Disk Volume. The Disk is Basic(Bootable) Disk Volume.", "")
		return false, rtnErr
	}

	detachReq := server.DetachBlockStorageInstancesRequest{
		BlockStorageInstanceNoList: []*string{ncloud.String(diskIID.SystemId)},
	}
	callLogStart := call.Start()
	result, err := diskHandler.VMClient.V2Api.DetachBlockStorageInstances(&detachReq)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Detach the Block Storage : ", err)
		return false, rtnErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if !strings.EqualFold(*result.ReturnMessage, "success") {
		newErr := fmt.Errorf("Failed to Detach the Block Storage!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		// return false, newErr

		// Try again!!
		callLogStart := call.Start()
		result, err := diskHandler.VMClient.V2Api.DetachBlockStorageInstances(&detachReq)
		if err != nil {
			rtnErr := logAndReturnError(callLogInfo, "Failed to Detach the Block Storage : ", err)
			return false, rtnErr
		}
		LoggingInfo(callLogInfo, callLogStart)

		if !strings.EqualFold(*result.ReturnMessage, "success") {
			rtnErr := logAndReturnError(callLogInfo, "Failed to Detach the Block Storage!!", "")
			return false, rtnErr
		} else {
			cblogger.Info("Succeeded in Detaching the Block Storage.")
		}
	} else {
		cblogger.Info("Succeeded in Detaching the Block Storage.")
	}

	// Wait for Disk Detachment finished
	curStatus, waitErr := diskHandler.WaitForDiskDetachment(diskIID)
	if waitErr != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Wait to Get Disk Info. : ", waitErr)
		return false, rtnErr
	}
	cblogger.Infof("==> Disk Status : [%s]", string(curStatus))

	return true, nil
}

func (diskHandler *NcpDiskHandler) GetNcpDiskInfo(diskIID irs.IID) (*server.BlockStorageInstance, error) {
	cblogger.Info("NCP Cloud Driver: called GetNCPDiskInfo()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, diskIID.SystemId, "DetachDisk()")

	if strings.EqualFold(diskIID.SystemId, "") {
		rtnErr := logAndReturnError(callLogInfo, "Invalid Disk SystemId!!", "")
		return nil, rtnErr
	}

	vmHandler := NcpVMHandler{
		RegionInfo: diskHandler.RegionInfo,
		VMClient:   diskHandler.VMClient,
	}
	zoneNo, err := vmHandler.getZoneNo(diskHandler.RegionInfo.Region, diskHandler.RegionInfo.Zone) // Region/Zone info of diskHandler
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NCP Zone No of the Zone Code : ", err)
		return nil, rtnErr
	}
	storageNoList := []*string{ncloud.String(diskIID.SystemId)}
	storageReq := server.GetBlockStorageInstanceListRequest{
		ZoneNo:                     zoneNo, // Caution : Not ZoneCode
		BlockStorageInstanceNoList: storageNoList,
	}
	result, err := diskHandler.VMClient.V2Api.GetBlockStorageInstanceList(&storageReq)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get the Block Storage Info : ", err)
		return nil, rtnErr
	}
	if len(result.BlockStorageInstanceList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any Block Storage Info with the ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	} else {
		cblogger.Info("Succeeded in Getting NCP Block Storage Info.")
	}
	return result.BlockStorageInstanceList[0], nil
}

// Waiting for up to 500 seconds during Disk creation until Disk info. can be get
func (diskHandler *NcpDiskHandler) WaitForDiskCreation(diskIID irs.IID) (irs.DiskStatus, error) {
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
func (diskHandler *NcpDiskHandler) WaitForDiskAttachment(diskIID irs.IID) (irs.DiskStatus, error) {
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
		case string(irs.DiskCreating), string(irs.DiskAvailable), string(irs.DiskDeleting), string(irs.DiskError), "Attaching", "Detaching", "Unknown":
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
func (diskHandler *NcpDiskHandler) WaitForDiskDetachment(diskIID irs.IID) (irs.DiskStatus, error) {
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
		case string(irs.DiskCreating), string(irs.DiskAttached), string(irs.DiskDeleting), string(irs.DiskError), "Attaching", "Detaching", "Unknown":
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

func (diskHandler *NcpDiskHandler) GetDiskStatus(diskIID irs.IID) (irs.DiskStatus, error) {
	cblogger.Info("NCP Cloud Driver: called GetDiskStatus()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		return irs.DiskError, newErr
	}

	diskInfo, err := diskHandler.GetNcpDiskInfo(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Disk Info : [%v] ", err)
		cblogger.Error(newErr.Error())
		return irs.DiskError, newErr
	}
	cblogger.Infof("# Disk Status of NCP : [%s]", *diskInfo.BlockStorageInstanceStatusName)
	return ConvertDiskStatus(*diskInfo.BlockStorageInstanceStatusName), nil
}

func (diskHandler *NcpDiskHandler) MappingDiskInfo(storage server.BlockStorageInstance) (irs.DiskInfo, error) {
	cblogger.Info("NCP Cloud Driver: called MappingDiskInfo()")

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
			NameId:   ncloud.StringValue(storage.BlockStorageName),
			SystemId: ncloud.StringValue(storage.BlockStorageInstanceNo),
		},
		Zone: 		 ncloud.StringValue(storage.Zone.ZoneCode),
		DiskSize:    strconv.FormatInt((*storage.BlockStorageSize)/(1024*1024*1024), 10),
		Status:      ConvertDiskStatus(ncloud.StringValue(storage.BlockStorageInstanceStatusName)), // Not BlockStorageInstanceStatus.Code
		CreatedTime: convertedTime,
		DiskType:    ncloud.StringValue(storage.DiskDetailType.Code),
	}

	if strings.EqualFold(ncloud.StringValue(storage.BlockStorageInstanceStatusName), "attached") {
		vmHandler := NcpVMHandler{
			RegionInfo: diskHandler.RegionInfo,
			VMClient:   diskHandler.VMClient,
		}
		subnetZone, err := vmHandler.getVMSubnetZone(storage.ServerInstanceNo)
		if err != nil {
			newErr := fmt.Errorf("Failed to Get the Subnet Zone info of the VM!! : [%v]", err)
			cblogger.Debug(newErr.Error())
			// return irs.VMInfo{}, newErr  // Caution!!
		}
	
		var ncpVMInfo *server.ServerInstance
		vmErr := errors.New("")	
		if strings.EqualFold(vmHandler.RegionInfo.Zone, subnetZone){ // Not diskHandler.RegionInfo.Zone
			ncpVMInfo, vmErr = vmHandler.getNcpVMInfo(ncloud.StringValue(storage.ServerInstanceNo))
			if vmErr != nil {
				newErr := fmt.Errorf("Failed to Get the VM Info of the Zone : [%s], [%v]", subnetZone, vmErr)
				cblogger.Error(newErr.Error())
				return irs.DiskInfo{}, newErr
			}
		} else {
			ncpVMInfo, vmErr = vmHandler.getNcpTargetZoneVMInfo(storage.ServerInstanceNo)
			if vmErr != nil {
				newErr := fmt.Errorf("Failed to Get the VM Info of the Zone : [%s], [%v]", subnetZone, vmErr)
				cblogger.Error(newErr.Error())
				return irs.DiskInfo{}, newErr
			}
		}

		diskInfo.OwnerVM = irs.IID{
			NameId:   ncloud.StringValue(ncpVMInfo.ServerName),
			SystemId: ncloud.StringValue(storage.ServerInstanceNo),
		}
	}

	keyValueList := []irs.KeyValue{
		{Key: "DeviceName", Value: ncloud.StringValue(storage.DeviceName)},
		{Key: "RegionCode", Value: ncloud.StringValue(storage.Region.RegionCode)},
		{Key: "ZoneCode", Value: ncloud.StringValue(storage.Zone.ZoneCode)},
		{Key: "BlockStorageType", Value: ncloud.StringValue(storage.BlockStorageType.CodeName)},
		{Key: "BlockStorageDiskType", Value: ncloud.StringValue(storage.DiskType.CodeName)},
	}
	diskInfo.KeyValueList = keyValueList

	return diskInfo, nil
}

func ConvertDiskStatus(diskStatus string) irs.DiskStatus {
	cblogger.Info("NCP Cloud Driver: called ConvertDiskStatus()")

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
	case "attaching":
		resultStatus = "Attaching"
	case "detaching":
		resultStatus = "Detaching"
	default:
		resultStatus = "Unknown"
	}
	return resultStatus
}

func (diskHandler *NcpDiskHandler) GetNcpVMList() ([]*server.ServerInstance, error) {
	cblogger.Info("NCP Cloud Driver: called GetNcpVMList()")
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, "GetNcpVMList()", "GetNcpVMList()")

	vmHandler := NcpVMHandler{
		RegionInfo: diskHandler.RegionInfo,
		VMClient:   diskHandler.VMClient,
	}
	regionNo, err := vmHandler.getRegionNo(diskHandler.RegionInfo.Region)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NCP Region No of the Region Code : ", err)
		return nil, rtnErr
	}
	zoneNo, err := vmHandler.getZoneNo(diskHandler.RegionInfo.Region, diskHandler.RegionInfo.Zone) // Region/Zone info of diskHandler
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NCP Zone No of the Zone Code : ", err)
		return nil, rtnErr
	}
	instanceReq := server.GetServerInstanceListRequest{
		ServerInstanceNoList: []*string{},
		RegionNo:             regionNo, // Caution : Not RegionCode
		ZoneNo:               zoneNo,   // Caution : Not ZoneCode
	}
	callLogStart := call.Start()
	result, err := diskHandler.VMClient.V2Api.GetServerInstanceList(&instanceReq)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Find the VM Instance list from NCP : ", err)
		return nil, rtnErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.ServerInstanceList) < 1 {
		cblogger.Info("### VM Instance does Not Exist!!")
	} else {
		cblogger.Info("Succeeded in Getting VM Instance list from NCP")
	}
	return result.ServerInstanceList, nil
}

func (diskHandler *NcpDiskHandler) GetNcpVMListWithZone(reqZoneId string) ([]*server.ServerInstance, error) {
	cblogger.Info("NCP Cloud Driver: called GetNcpVMListWithZone()")
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Region, call.DISK, "GetNcpVMList()", "GetNcpVMListWithSubnetZone()")

	vmHandler := NcpVMHandler{
		RegionInfo: diskHandler.RegionInfo,
		VMClient:   diskHandler.VMClient,
	}
	regionNo, err := vmHandler.getRegionNo(diskHandler.RegionInfo.Region)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NCP Region No of the Region Code : ", err)
		return nil, rtnErr
	}
	reqZoneNo, err := vmHandler.getZoneNo(diskHandler.RegionInfo.Region, reqZoneId) // Not diskHandler.RegionInfo.Zone
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NCP Zone No of the Zone Code : ", err)
		return nil, rtnErr
	}
	instanceReq := server.GetServerInstanceListRequest{
		ServerInstanceNoList: []*string{},
		RegionNo:             regionNo, // Caution : Not RegionCode
		ZoneNo:               reqZoneNo,   // Caution : Not ZoneCode
	}
	callLogStart := call.Start()
	result, err := diskHandler.VMClient.V2Api.GetServerInstanceList(&instanceReq)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Find the VM Instance list from NCP : ", err)
		return nil, rtnErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.ServerInstanceList) < 1 {
		cblogger.Info("### VM Instance does Not Exist!!")
	} else {
		cblogger.Info("Succeeded in Getting VM Instance list from NCP")
	}
	return result.ServerInstanceList, nil
}

func (diskHandler *NcpDiskHandler) IsBasicBlockStorage(diskIID irs.IID) (bool, error) {
	cblogger.Info("NCP Cloud Driver: called IsBasicBlockStorage()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	diskInfo, err := diskHandler.GetNcpDiskInfo(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Disk Info : [%v] ", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}
	if strings.EqualFold(*diskInfo.BlockStorageType.Code, "BASIC") { // Ex) "BASIC", "SVRBS", ...
		return true, nil
	} else {
		cblogger.Infof("# BlockStorageType : [%s]", *diskInfo.BlockStorageType.CodeName) // Ex) "Basic BS", "Server BS", ...
		return false, nil
	}
}

func (diskHandler *NcpDiskHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("NCP Driver: called ListIID()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Zone, call.DISK, "ListIID()", "ListIID()")

	vmHandler := NcpVMHandler{
		RegionInfo: diskHandler.RegionInfo,
		VMClient:   diskHandler.VMClient,
	}
	zoneNo, err := vmHandler.getZoneNo(diskHandler.RegionInfo.Region, diskHandler.RegionInfo.Zone) // Region/Zone info of diskHandler
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get NCP Zone No of the Zone Code : ", err)
		return nil, rtnErr
	}
	storageReq := server.GetBlockStorageInstanceListRequest{
		ZoneNo: zoneNo, // $$$ Caution!! : Not ZoneCode
	}
	callLogStart := call.Start()
	result, err := diskHandler.VMClient.V2Api.GetBlockStorageInstanceList(&storageReq)
	if err != nil {
		rtnErr := logAndReturnError(callLogInfo, "Failed to Get Block Storage List from NCP : ", err)
		return nil, rtnErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var iidList []*irs.IID
	if len(result.BlockStorageInstanceList) < 1 {
		cblogger.Info("### Block Storage does Not Exist!!")
		return nil, nil
	} else {
		cblogger.Info("Succeeded in Getting Block Storage list from NCP.")
		for _, storage := range result.BlockStorageInstanceList {
			iid := &irs.IID{
				NameId:   ncloud.StringValue(storage.BlockStorageName),
				SystemId: ncloud.StringValue(storage.BlockStorageInstanceNo),
			}
			iidList = append(iidList, iid)
		}
	}
	return iidList, nil
}
