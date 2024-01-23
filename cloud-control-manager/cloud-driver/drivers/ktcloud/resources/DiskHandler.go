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

	ktsdk "github.com/cloud-barista/ktcloud-sdk-go"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	STG_Type 				string = "STG" // General HDD Type
	SSD_Type               	string = "SSD"
	DefaultDiskSize        	string = "100"
	DefaultUsagePlanType	string = "hourly"
	DefaultDiskIOPS        	string = "10000"
	DefaultWindowsDiskSize 	string = "50"
	KOR_Seoul_M_ZoneID  	string = "95e2f517-d64a-4866-8585-5177c256f7c7"
)

type KtCloudDiskHandler struct {
	RegionInfo idrv.RegionInfo
	Client     *ktsdk.KtCloudClient
}

func (diskHandler *KtCloudDiskHandler) CreateDisk(diskReqInfo irs.DiskInfo) (irs.DiskInfo, error) {
	cblogger.Info("KT Cloud Driver: called CreateDisk()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Zone, call.DISK, diskReqInfo.IId.NameId, "CreateDisk()")

	if strings.EqualFold(diskReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid Disk NameId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}

	if strings.EqualFold(diskHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Invalid Zone Info!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}

	// # New Volume Creation Parameters : https://cloud.kt.com/docs/open-api-guide/g/computing/disk-volume
	reqDiskType := diskReqInfo.DiskType // 'default', 'HDD' or 'SSD'
	reqDiskSize := diskReqInfo.DiskSize
	
	if strings.EqualFold(diskHandler.RegionInfo.Zone, "dfd6f03d-dae5-458e-a2ea-cb6a55d0d994") && strings.EqualFold(reqDiskType, "SSD") {
		newErr := fmt.Errorf("Invalid Disk Type!! 'KOR-HA' zone does Not support 'SSD' disk type!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}

	var reqIOPS string
	if strings.EqualFold(reqDiskType, "") || strings.EqualFold(reqDiskType, "default") {
		reqDiskType = STG_Type // In case, Volume Type is not specified.
	} else if strings.EqualFold(reqDiskType, "HDD") {
		reqDiskType = STG_Type
	} else if strings.EqualFold(reqDiskType, "SSD") {
		reqDiskType = SSD_Type
		reqIOPS 	= DefaultDiskIOPS
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

	// # HDD type : 10~300(GB)
	if strings.EqualFold(reqDiskType, "STG") && (reqDiskSizeInt < 10 || reqDiskSizeInt > 300) {
		newErr := fmt.Errorf("Invalid Disk Size. 'HDD' Disk Size Must be between 10 and 300.")
		cblogger.Error(newErr.Error())
		return irs.DiskInfo{}, newErr
	}
	// # SSD-provisioned type : 100~800(GB). It can be designated in 100GB units.
	if strings.EqualFold(reqDiskType, "SSD") && (reqDiskSizeInt < 100 || reqDiskSizeInt > 800) {
		newErr := fmt.Errorf("Invalid Disk Size. 'HDD' Disk Size Must be between 100 and 800.")
		cblogger.Error(newErr.Error())
		return irs.DiskInfo{}, newErr
	}

	// ### Create volume using product abbreviations (ex. STG 100G, SSD 300G, etc.)
	volumeProductCode := reqDiskType + " " + reqDiskSize + "G"
	cblogger.Infof("# ProductCode : %s", volumeProductCode)

	// ### ProductCode : Create volume using product abbreviations (ex. STG 100G, SSD 300G, etc.)
	// ### If the 'ProductCode' field is used, the 'DiskOfferingId' field value is ignored.
	volumeReq := ktsdk.CreateVolumeReqInfo{
		Name:           diskReqInfo.IId.NameId,  		// Required
		DiskOfferingId: "",								// Required
		ZoneId:         diskHandler.RegionInfo.Zone, 	// Required
		UsagePlanType:  DefaultUsagePlanType,
		ProductCode:    volumeProductCode,
		IOPS: 			reqIOPS, // When entering IOPS value, it is created with 'SSD-Provisioned' type of volume. (Not general SSD type)
	}
	start := call.Start()
	createVolumeResponse, err := diskHandler.Client.CreateVolume(volumeReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create New Disk Volume. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	cblogger.Info("### Waiting for the Disk to be Created(300sec)!!\n")
	waitErr := diskHandler.Client.WaitForAsyncJob(createVolumeResponse.Createvolumeresponse.JobId, 300000000000)
	if waitErr != nil {
		cblogger.Errorf("Failed to Wait the Job : [%v]", waitErr)
		return irs.DiskInfo{}, waitErr
	}

	newVolumeIID := irs.IID{SystemId: createVolumeResponse.Createvolumeresponse.ID}
	newDiskInfo, err := diskHandler.GetDisk(newVolumeIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud Volume Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.DiskInfo{}, newErr
	}
	return newDiskInfo, nil
}

// ### Caution!!) Except for the DISK(ROOT and DATADISK installed an OS) provided when creating VMs.
func (diskHandler *KtCloudDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	cblogger.Info("KT Cloud Driver: called ListDisk()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Zone, call.DISK, "ListDisk()", "ListDisk()")

	volumeReq := ktsdk.ListVolumeReqInfo{
		ZoneId: diskHandler.RegionInfo.Zone,
	}
	start := call.Start()
	result, err := diskHandler.Client.ListVolumes(volumeReq)
	if err != nil {
		cblogger.Error("Failed to Get KT Cloud Volume list : [%v]", err)
		return []*irs.DiskInfo{}, err
	}
	LoggingInfo(callLogInfo, start)
	// spew.Dump(result)

	if len(result.Listvolumesresponse.Volume) < 1 {
		cblogger.Info("# KT Cloud Volume does Not Exist!!")
		return []*irs.DiskInfo{}, nil // Not Return Error
	}
	// spew.Dump(result.Listvolumesresponse.Volume)

	var volumeInfoList []*irs.DiskInfo
	for _, volume := range result.Listvolumesresponse.Volume {
		if !strings.Contains(volume.Name, "ROOT-") && !strings.Contains(volume.Name, "DATA-"){
			volumeInfo, err := diskHandler.MappingDiskInfo(&volume)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get Disk Info list!! : [%v] ", err)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return nil, newErr
			}
			volumeInfoList = append(volumeInfoList, &volumeInfo)
		}
	}
	return volumeInfoList, nil
}

func (diskHandler *KtCloudDiskHandler) GetDisk(diskIID irs.IID) (irs.DiskInfo, error) {
	cblogger.Info("KT Cloud Driver: called GetDisk()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Zone, call.DISK, diskIID.SystemId, "GetDisk()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}

	volume, err := diskHandler.GetKtVolumeInfo(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud Volume Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.DiskInfo{}, newErr
	}

	volumeInfo, err := diskHandler.MappingDiskInfo(&volume)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Disk Info!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	return volumeInfo, nil
}

// $$$ Caution!!) This is only for Bootalbe Disk of a VM.
func (diskHandler *KtCloudDiskHandler) ChangeDiskSize(diskIID irs.IID, newDiskSize string) (bool, error) {
	cblogger.Info("KT Cloud Driver: called ChangeDiskSize()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Zone, call.DISK, diskIID.SystemId, "ChangeDiskSize()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
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

	newDiskSizeInt, err := strconv.Atoi(newDiskSize)
	if err != nil {
		newErr := fmt.Errorf("Failed to Convert New Disk Size to Int. type. [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	// TBD
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

	addDiskSizeInt 	:= newDiskSizeInt - curDiskSizeInt // Disk Size 'to Add'
	addDiskSize 	:= strconv.Itoa(addDiskSizeInt)

	// $$$ Caution!!) At least one VM is required to resize the disk volume in case of KT Cloud G1/G2 plaform.
	// $$$ It can only be used with the VM which is 'Susppended' status.
	volumeReq := ktsdk.ResizeVolumeReqInfo{
		ID:         diskIID.SystemId,  			// Required. Volume ID
		VMId:		diskInfo.OwnerVM.SystemId,	// Required
		Size:     	addDiskSize, 				// Required. Disk Size 'to Add'. Only 50(Linux series only), 80, and 100 are available.
		IsLinux:  	"Y", 						// Required. 'Y' for Linux series, 'N' for Windows series
	}
	start := call.Start()
	ResizeVolumeResponse, err := diskHandler.Client.ResizeVolume(volumeReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create New Disk Volume. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, start)

	cblogger.Info("### Waiting for the Disk to be Resized(300sec)!!\n")
	waitErr := diskHandler.Client.WaitForAsyncJob(ResizeVolumeResponse.Resizevolumeresponse.JobId, 300000000000)
	if waitErr != nil {
		cblogger.Errorf("Failed to Wait the Job : [%v]", waitErr)
		return false, waitErr
	}

	return true, nil
}

func (diskHandler *KtCloudDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud Driver: called DeleteDisk()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Zone, call.DISK, "DeleteDisk()", "DeleteDisk()")

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
	} else if strings.EqualFold(string(curStatus), string(irs.DiskAttached)) {
		newErr := fmt.Errorf("Failed to Delete the Disk Volume. The Disk Status is 'Attached'.")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	start := call.Start()
	delResult, err := diskHandler.Client.DeleteVolume(diskIID.SystemId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the Disk Volume!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, start)
	// cblogger.Info("\n\n### delResult : ")
	// spew.Dump(delResult)

	if !strings.EqualFold(delResult.Deletevolumeresponse.Success, "true") { // String type of value
		newErr := fmt.Errorf("Failed to Delete the Disk Volume!!")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	return true, nil
}

func (diskHandler *KtCloudDiskHandler) AttachDisk(diskIID irs.IID, vmIID irs.IID) (irs.DiskInfo, error) {
	cblogger.Info("KT Cloud Driver: called AttachDisk()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Zone, call.DISK, diskIID.SystemId, "AttachDisk()")

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

	volumeReq := ktsdk.AttachVolumeReqInfo{
		ID:         diskIID.SystemId,  		// Required. Volume ID
		VMId:		vmIID.SystemId,			// Required
		// DeviceId:     	"",
	}
	start := call.Start()
	AttachVolumeResponse, err := diskHandler.Client.AttachVolume(volumeReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Attach the Disk Volume. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	cblogger.Info("\n\n### Waiting for the Disk to be Attached(600sec)!!\n")
	waitErr := diskHandler.Client.WaitForAsyncJob(AttachVolumeResponse.Attachvolumeresponse.JobId, 600000000000)
	if waitErr != nil {
		cblogger.Errorf("Failed to Wait the Job : [%v]", waitErr)
		return irs.DiskInfo{}, waitErr
	}

	diskInfo, err := diskHandler.GetDisk(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Disk Info!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.DiskInfo{}, newErr
	}
	return diskInfo, nil
}

func (diskHandler *KtCloudDiskHandler) DetachDisk(diskIID irs.IID, vmIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud Driver: called DetachDisk()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Zone, call.DISK, diskIID.SystemId, "DetachDisk()")

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

	isBootable, err := diskHandler.IsBootableDisk(diskIID)
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

	volumeReq := ktsdk.DetachVolumeReqInfo{
		ID:         diskIID.SystemId,  	 // Required. Volume ID
		// VMId:		vmIID.SystemId,  // When input VMId -> Error : 'Please provide either a volume id, or a tuple(device id, instance id)'
		// DeviceId:     	"",
	}
	start := call.Start()
	DetachVolumeResponse, err := diskHandler.Client.DetachVolume(volumeReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Detach the Disk Volume. [%v]", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, start)

	cblogger.Info("\n\n### Waiting for the Disk to be Detached(600sec)!!\n")
	waitErr := diskHandler.Client.WaitForAsyncJob(DetachVolumeResponse.Detachvolumeresponse.JobId, 600000000000)
	if waitErr != nil {
		cblogger.Errorf("Failed to Wait the Job : [%v]", waitErr)
		return false, waitErr
	}

	return true, nil
}

func (diskHandler *KtCloudDiskHandler) GetDiskStatus(diskIID irs.IID) (irs.DiskStatus, error) {
	cblogger.Info("KT Cloud Driver: called GetDiskStatus()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		return irs.DiskError, newErr
	}

	volume, err := diskHandler.GetKtVolumeInfo(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud Volume Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.DiskError, newErr
	}

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		return irs.DiskError, newErr
	}

	// Note) Because the 'state' value of volume info (from KT Cloud) does not indicate whether it is Attachment.
	if !strings.EqualFold(volume.VMId, "") { 
		return ConvertDiskStatus("attached"), nil
	}
	return ConvertDiskStatus("allocated"), nil	
}

// $$$ Caution!!) KT Volume State : chanaged to 'Ready' after Attachment. Stil 'Ready' after Detachment.
// Caution!!) And, The DataDisk with the OS installed is "state : ready" even when it is in the detached state.
func ConvertDiskStatus(diskStatus string) irs.DiskStatus {
	cblogger.Info("KT Cloud Driver: called ConvertDiskStatus()")

	var resultStatus irs.DiskStatus
	switch strings.ToLower(diskStatus) { // Caution!! : ToLower()
	case "creating":	// Caution!! : // KT Volume State : "Creating" => Attachment is in Progress
		resultStatus = irs.DiskCreating
	case "allocated":	// KT Volume State : "Allocated" (Confirmed)
		resultStatus = irs.DiskAvailable
	case "ready": 		
		resultStatus = irs.DiskAttached
	case "attached": 	// Note) Added this status!!
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

func (diskHandler *KtCloudDiskHandler) IsBootableDisk(diskIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud Driver: called IsBootableDisk()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	volume, err := diskHandler.GetKtVolumeInfo(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud Volume Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	isBootable := false
	if strings.EqualFold(volume.Type, "ROOT") || strings.Contains(volume.DiskOfferingName, "linux") || strings.Contains(volume.DiskOfferingName, "win"){
		isBootable = true
	}
	return isBootable, nil
}

func (diskHandler *KtCloudDiskHandler) MappingDiskInfo(volume *ktsdk.Volume) (irs.DiskInfo, error) {
	cblogger.Info("KT Cloud Driver: called MappingDiskInfo()")
	// cblogger.Info("\n\n### volume : ")
	// spew.Dump(volume)
	// cblogger.Info("\n")
	cblogger.Infof("# Given Volume State on KT Cloud : %s", volume.State) // Not Correct!!

	volumeIID := irs.IID{SystemId: volume.ID}
	curStatus, err := diskHandler.GetDiskStatus(volumeIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the Disk Status : [%v] ", err)
		cblogger.Error(newErr.Error())
		return irs.DiskInfo{}, newErr
	}

	var convertedTime time.Time
	var convertErr error
	if !strings.EqualFold(volume.Created, "") {
		convertedTime, convertErr = convertTimeFormat(volume.Created)
		if convertErr != nil {
			newErr := fmt.Errorf("Failed to Convert the Time Format!! : [%v]", convertErr)
			cblogger.Error(newErr.Error())
			return irs.DiskInfo{}, newErr
		}
	}

	diskInfo := irs.DiskInfo{
		IId: irs.IID{
			SystemId: volume.ID,
		},
		DiskSize:    strconv.FormatInt(volume.Size/(1024*1024*1024), 10),
		Status:      curStatus,
		// Status:      ConvertDiskStatus(volume.State),
		CreatedTime: convertedTime,
	}

	if !strings.EqualFold(volume.Name, "") {
		diskInfo.IId.NameId = volume.Name
	}

	// Caution!!) In 'KOR Seoul M' zone, in case the created disk is 'SSD', it appears as "volumetype": "general". 
	// 			  (Shoud be "volumetype": "ssd")
	if strings.EqualFold(diskHandler.RegionInfo.Zone, KOR_Seoul_M_ZoneID){
		if strings.Contains(volume.DiskOfferingName, "SSD") {
			diskInfo.DiskType = "SSD"
		} else {
			diskInfo.DiskType = "HDD"
		}
	} else if !strings.EqualFold(volume.VolumeType, "") {
		if strings.EqualFold(volume.VolumeType, "general") {
			diskInfo.DiskType = "HDD"
		} else if strings.EqualFold(volume.VolumeType, "ssd") {
			diskInfo.DiskType = "SSD"
		}
	}

	if !strings.EqualFold(volume.VMName, "") || !strings.EqualFold(volume.VMId, "") {
		diskInfo.OwnerVM = irs.IID{
			NameId:   volume.VMName,
			SystemId: volume.VMId,
		}
	}

	var iops string
	if !strings.EqualFold(strconv.FormatInt(volume.MaxIOPS, 10), "0")  {
		iops = strconv.FormatInt(volume.MaxIOPS, 10)
	}
	
	keyValueList := []irs.KeyValue{
		{Key: "Type", 			Value: volume.Type},
		{Key: "MaxIOPS", 		Value: iops},
		{Key: "UsagePlanType", 	Value: volume.UsagePlanType},
		{Key: "AttachedTime", 	Value: volume.AttachedTime},
		{Key: "VMState", 		Value: volume.VMState},
	}
	diskInfo.KeyValueList = keyValueList

	return diskInfo, nil
}

func (diskHandler *KtCloudDiskHandler) GetKtVolumeInfo(diskIID irs.IID) (ktsdk.Volume, error) {
	cblogger.Info("KT Cloud Driver: called GetKtVolumeInfo()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Zone, call.DISK, diskIID.SystemId, "GetKtVolumeInfo()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		return ktsdk.Volume{}, newErr
	}

	volumeReq := ktsdk.ListVolumeReqInfo{
		ZoneId: diskHandler.RegionInfo.Zone,
		ID: 	diskIID.SystemId,
	}
	start := call.Start()
	result, err := diskHandler.Client.ListVolumes(volumeReq)
	if err != nil {
		cblogger.Error("Failed to Get KT Cloud Volume list : [%v]", err)
		return ktsdk.Volume{}, err
	}
	LoggingInfo(callLogInfo, start)
	// spew.Dump(result)

	if len(result.Listvolumesresponse.Volume) < 1 {
		newErr := fmt.Errorf("Failed to Find the Volume Info with the ID on the Zone!!")
		cblogger.Error(newErr.Error())
		return ktsdk.Volume{}, newErr
	}
	return result.Listvolumesresponse.Volume[0], nil
}

func (diskHandler *KtCloudDiskHandler) GetVolumeIdWithVMid(vmId string) (string, error) {
	cblogger.Info("KT Cloud Driver: called GetVolumeIdWithVMid()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Zone, call.DISK, vmId, "GetVolumeIdWithVMid()")

	if strings.EqualFold(vmId, "") {
		newErr := fmt.Errorf("Invalid VM ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	volumeReq := ktsdk.ListVolumeReqInfo{
		ZoneId: diskHandler.RegionInfo.Zone,
	}
	start := call.Start()
	result, err := diskHandler.Client.ListVolumes(volumeReq)
	if err != nil {
		cblogger.Error("Failed to Get KT Cloud Volume list : [%v]", err)
		return "", err
	}
	LoggingInfo(callLogInfo, start)
	// spew.Dump(result)

	if len(result.Listvolumesresponse.Volume) < 1 {
		newErr := fmt.Errorf("Failed to Get Volume List on the Zone!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}

	var volumeId string
	for _, volume := range result.Listvolumesresponse.Volume {
		if strings.EqualFold(volume.VMId, vmId){
			volumeId = volume.ID
			break
		}
	}
	if strings.EqualFold(volumeId, ""){
		newErr := fmt.Errorf("Failed to Get Volume ID with the VM ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	return volumeId, nil	
}
