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
	STG_Type 					string = "STG" // General HDD Type
	SSD_Type               		string = "SSD"
	DefaultDiskSize        		string = "100"
	DefaultDiskUsagePlanType	string = "hourly"
	DefaultDiskIOPS        		string = "10000"
	DefaultWindowsDiskSize 		string = "50"
	KOR_Seoul_M_ZoneID  		string = "95e2f517-d64a-4866-8585-5177c256f7c7"
)

type KtCloudDiskHandler struct {
	RegionInfo idrv.RegionInfo
	Client     *ktsdk.KtCloudClient
}

type DiskOffering struct {
	Type           string
	Size           string
	DiskOfferingId string
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

	// ### ProductCode : Create volume using product abbreviations (ex. STG 100G, SSD 300G, etc.)
	volumeProductCode := reqDiskType + " " + reqDiskSize + "G"
	cblogger.Infof("# ProductCode : %s", volumeProductCode)
	// ### If the 'ProductCode' field is used, the 'DiskOfferingId' field value is ignored.
	volumeReq := ktsdk.CreateVolumeReqInfo{
		Name:           diskReqInfo.IId.NameId,  		// Required
		DiskOfferingId: "",								// Required
		ZoneId:         diskHandler.RegionInfo.Zone, 	// Required
		UsagePlanType:  DefaultDiskUsagePlanType,
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
		// if !strings.Contains(volume.Name, "ROOT-") && !strings.Contains(volume.Name, "DATA-"){ // When need filtering
			volumeInfo, err := diskHandler.mappingDiskInfo(&volume)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get Disk Info list!! : [%v] ", err)
				cblogger.Error(newErr.Error())
				LoggingError(callLogInfo, newErr)
				return nil, newErr
			}
			volumeInfoList = append(volumeInfoList, &volumeInfo)
		// }
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

	volume, err := diskHandler.getKtVolumeInfo(diskIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KT Cloud Volume Info : [%v]", err)
		cblogger.Error(newErr.Error())
		return irs.DiskInfo{}, newErr
	}

	volumeInfo, err := diskHandler.mappingDiskInfo(&volume)
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

func (diskHandler *KtCloudDiskHandler) getDiskStatus(diskIID irs.IID) (irs.DiskStatus, error) {
	cblogger.Info("KT Cloud Driver: called getDiskStatus()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		return irs.DiskError, newErr
	}

	volume, err := diskHandler.getKtVolumeInfo(diskIID)
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
		return convertDiskStatus("attached"), nil
	}
	return convertDiskStatus("allocated"), nil	
}

// $$$ Caution!!) KT Volume State : chanaged to 'Ready' after Attachment. Stil 'Ready' after Detachment.
// Caution!!) And, The DataDisk with the OS installed is "state : ready" even when it is in the detached state.
func convertDiskStatus(diskStatus string) irs.DiskStatus {
	cblogger.Info("KT Cloud Driver: called convertDiskStatus()")

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

func (diskHandler *KtCloudDiskHandler) isBootableDisk(diskIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud Driver: called isBootableDisk()")

	if strings.EqualFold(diskIID.SystemId, "") {
		newErr := fmt.Errorf("Invalid Disk SystemId!!")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	volume, err := diskHandler.getKtVolumeInfo(diskIID)
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

func (diskHandler *KtCloudDiskHandler) mappingDiskInfo(volume *ktsdk.Volume) (irs.DiskInfo, error) {
	cblogger.Info("KT Cloud Driver: called mappingDiskInfo()")
	// cblogger.Info("\n\n### volume : ")
	// spew.Dump(volume)
	// cblogger.Info("\n")
	cblogger.Infof("# Given Volume State on KT Cloud : %s", volume.State) // Not Correct!!

	volumeIID := irs.IID{SystemId: volume.ID}
	curStatus, err := diskHandler.getDiskStatus(volumeIID)
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
		// Status:      convertDiskStatus(volume.State),
		CreatedTime: convertedTime,
	}

	if !strings.EqualFold(volume.Name, "") {
		diskInfo.IId.NameId = volume.Name
	}

	// Caution!!) In 'KOR Seoul M' zone, in case the created disk is 'SSD', it appears as "volumetype": "general". (Shoud be "volumetype": "ssd")
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

func (diskHandler *KtCloudDiskHandler) getKtVolumeInfo(diskIID irs.IID) (ktsdk.Volume, error) {
	cblogger.Info("KT Cloud Driver: called getKtVolumeInfo()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Zone, call.DISK, diskIID.SystemId, "getKtVolumeInfo()")

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

func (diskHandler *KtCloudDiskHandler) getVolumeIdsWithVMId(vmId string) ([]string, error) {
	cblogger.Info("KT Cloud Driver: called getVolumeIdsWithVMId()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Zone, call.DISK, vmId, "getVolumeIdsWithVMId()")

	if strings.EqualFold(vmId, "") {
		newErr := fmt.Errorf("Invalid VM ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	volumeReq := ktsdk.ListVolumeReqInfo{
		ZoneId: diskHandler.RegionInfo.Zone,
	}
	start := call.Start()
	result, err := diskHandler.Client.ListVolumes(volumeReq)
	if err != nil {
		cblogger.Error("Failed to Get KT Cloud Volume list : [%v]", err)
		return nil, err
	}
	LoggingInfo(callLogInfo, start)
	// spew.Dump(result)

	if len(result.Listvolumesresponse.Volume) < 1 {
		newErr := fmt.Errorf("Failed to Get Volume List on the Zone!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}

	var volumeIds []string
	for _, volume := range result.Listvolumesresponse.Volume {
		if strings.EqualFold(volume.VMId, vmId){
			volumeIds = append(volumeIds, volume.ID)
		}
	}
	if len(volumeIds) < 1 {
		newErr := fmt.Errorf("Failed to Get Volume ID with the VM ID!!")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	return volumeIds, nil	
}

func (diskHandler *KtCloudDiskHandler) getRootVolumeIdWithVMId(vmId string) (string, error) {
	cblogger.Info("KT Cloud Driver: called getRootVolumeIdWithVMId()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Zone, call.DISK, vmId, "getRootVolumeIdWithVMId()")

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
			volumeIID := irs.IID{SystemId: volume.ID}
			isBootable, err := diskHandler.isBootableDisk(volumeIID)
			if err != nil {
				newErr := fmt.Errorf("Failed to Get the Bootable Disk Info. : [%v] ", err)
				cblogger.Error(newErr.Error())
				return "", newErr
			}
			if isBootable {
				volumeId = volume.ID
			}
		}
	}

	if strings.EqualFold(volumeId, "") {
		newErr := fmt.Errorf("Failed to Get Volume ID with the VM ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	return volumeId, nil	
}

/*
// # Caution!!) Root Disk response value does not support the "volumetype" parameter.
func (diskHandler *KtCloudDiskHandler) GetRootVolumeTypeWithVMId(vmId string) (string, error) {
	cblogger.Info("KT Cloud Driver: called GetRootVolumeTypeWithVMId()")
	InitLog()
	callLogInfo := GetCallLogScheme(diskHandler.RegionInfo.Zone, call.DISK, vmId, "GetRootVolumeTypeWithVMId()")

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

	var volumeType string
	for _, volume := range result.Listvolumesresponse.Volume {
		if strings.EqualFold(volume.VMId, vmId){
			volumeType = volume.VolumeType
			break
		}
	}
	if strings.EqualFold(volumeType, ""){
		newErr := fmt.Errorf("Failed to Get Root Volume Type with the VM ID!!")
		cblogger.Error(newErr.Error())
		return "", newErr
	}
	return volumeType, nil	
}
*/

// Note) 'diskofferingid' info : https://cloud.kt.com/docs/open-api-guide/g/computing/disk-volume
func findAllDiskOfferingIds() []DiskOffering {
	return []DiskOffering{
		{"HDD", "10G", "1539f7a2-93bd-45fb-af6d-13d4d428286d"},
		{"HDD", "20G", "64ba2191-b22a-42a2-aaab-075b0235ac18"},
		{"HDD", "30G", "d4fc0ff3-91ee-40af-a2f3-9ec0eb4773f8"},
		{"HDD", "40G", "a7bbe834-c195-45ed-a668-2e4aadf0adb7"},
		{"HDD", "50G", "277d1cea-d51a-4570-af16-0b271795d2a0"},
		{"HDD", "60G", "2791cad6-b68a-4412-8867-ca07e5d64ae4"},
		{"HDD", "70G", "698adf24-7ae2-4100-af56-a35ecd7bd67c"},
		{"HDD", "80G", "78fe5777-8903-4193-bada-ee8686bc543a"},
		{"HDD", "90G", "4df2364f-1548-4d42-9d66-25fc3a195c5c"},
		{"HDD", "100G", "ef334e6f-f197-4988-9781-86c985e82591"},
		{"HDD", "110G", "e91d8b54-28c3-43a7-b3a4-439ced6fe282"},
		{"HDD", "120G", "b67fea21-0360-4072-8877-815e9254ab73"},
		{"HDD", "130G", "b825b79c-7f31-47f8-b886-ed355253c9e9"},
		{"HDD", "140G", "40f5cd46-ec3e-4ca3-9df5-f55d590149a1"},
		{"HDD", "150G", "dc4ec8a0-c0f6-46af-a475-9fcec21bc2fa"},
		{"HDD", "160G", "227ad57a-336f-48dd-8338-68094e6bdef9"},
		{"HDD", "170G", "2b034310-4309-4f43-8e78-c94445c70783"},
		{"HDD", "180G", "d02fc827-2c52-423d-9a6c-d324e0fbb021"},
		{"HDD", "190G", "ec59a93f-36bd-43d0-abe7-af70d97d8b1b"},
		{"HDD", "200G", "cbe4ccad-be3a-43f6-9abd-d2b6d7097e40"},
		{"HDD", "210G", "dc467090-6649-43b2-abd0-4d8ea52d5f49"},
		{"HDD", "220G", "2527cce9-1b7a-4b4e-b6df-568c4a67678c"},
		{"HDD", "230G", "95da4d30-f215-47ad-b7b8-11ae3a055e03"},
		{"HDD", "240G", "a70e745c-ef31-4fbb-a62b-ff5af031f8a1"},
		{"HDD", "250G", "e3782b90-3780-4c2c-85a1-48ccf304590e"},
		{"HDD", "260G", "600a22dc-8955-4fc6-a9d6-516c8db1ac5a"},
		{"HDD", "270G", "bb0227cf-9ef6-4005-bd27-666f49481003"},
		{"HDD", "280G", "c96b005c-81a3-46ca-ad63-96cd271faf6b"},
		{"HDD", "290G", "1c7521b1-8753-427b-874b-a740e6e0184d"},
		{"HDD", "300G", "03ee7edf-a91f-4910-9e1c-551222bc6e94"},

        {"HDD(Seoul-M2 Zone)", "10G", "55780e45-c235-498d-bf83-ab9a6943164a"},
        {"HDD(Seoul-M2 Zone)", "20G", "11c559b5-f8cf-4c02-abc8-8de92af14fe8"},
        {"HDD(Seoul-M2 Zone)", "30G", "b4aebcc5-ba2e-4ad1-97d7-79d85ca5da69"},
        {"HDD(Seoul-M2 Zone)", "40G", "913eba8b-ede3-4a8c-b83e-974ab445a68b"},
        {"HDD(Seoul-M2 Zone)", "50G", "2706dad8-6161-4e86-a30f-7cd35778096e"},
        {"HDD(Seoul-M2 Zone)", "60G", "f7371b79-4a88-4a01-9ae9-6f3a552dcded"},
        {"HDD(Seoul-M2 Zone)", "70G", "2a1c3e31-186a-47b3-9a01-0c00d61cc44e"},
        {"HDD(Seoul-M2 Zone)", "80G", "94a9f2db-f020-423f-b71e-1baf4eda8666"},
        {"HDD(Seoul-M2 Zone)", "90G", "9e4d4ffb-f8d3-4f6d-bed7-9330bc762cfe"},
        {"HDD(Seoul-M2 Zone)", "100G", "ac9cec92-5e8d-4d31-8250-23b62dd70cc5"},
        {"HDD(Seoul-M2 Zone)", "110G", "657f9465-b704-4466-9874-c3d78818974e"},
        {"HDD(Seoul-M2 Zone)", "120G", "4964f14d-848c-4752-b3da-8f40d53cdf99"},
        {"HDD(Seoul-M2 Zone)", "130G", "e054e0df-f165-4c4d-8ce4-75f1eccc5f48"},
        {"HDD(Seoul-M2 Zone)", "140G", "1bded494-5a78-4ce1-9e9d-7b9519544400"},
        {"HDD(Seoul-M2 Zone)", "150G", "55fa59d9-dc4d-4b32-9f9e-e5c66a0668dc"},
        {"HDD(Seoul-M2 Zone)", "160G", "8bca4eb7-b3d3-4b3c-9336-99257b0ef92e"},
        {"HDD(Seoul-M2 Zone)", "170G", "74d7060c-5200-47f4-985e-7d4fc6e1575b"},
        {"HDD(Seoul-M2 Zone)", "180G", "3a1e1743-b2d6-4ddb-80cf-a96081ea171f"},
        {"HDD(Seoul-M2 Zone)", "190G", "fcac36e8-d7be-4285-9dc8-b27a55c38c50"},
        {"HDD(Seoul-M2 Zone)", "200G", "363d943c-98d3-41be-a85c-c39e2e006c61"},
        {"HDD(Seoul-M2 Zone)", "210G", "c6f3a82d-0db7-48fe-8fe4-c8e44996cd3a"},
        {"HDD(Seoul-M2 Zone)", "220G", "44405ea8-27a7-4dec-bf1a-f3823b71f5d1"},
        {"HDD(Seoul-M2 Zone)", "230G", "5a0c80f8-978a-48f6-9add-2ee3627a0b39"},
        {"HDD(Seoul-M2 Zone)", "240G", "1d11f9e5-ec7a-455b-a7a2-5eb1ddea6aa3"},
        {"HDD(Seoul-M2 Zone)", "250G", "b20157e8-06f6-4bc1-9bc2-d6973a16a7bd"},
        {"HDD(Seoul-M2 Zone)", "260G", "8212fed4-82b7-4d1f-a9cc-519f540662fd"},
        {"HDD(Seoul-M2 Zone)", "270G", "45cb3e99-557f-4869-b62c-ffbdefb221c5"},
        {"HDD(Seoul-M2 Zone)", "280G", "74a48fa0-1cfe-41b5-9244-f6d95a2e367d"},
        {"HDD(Seoul-M2 Zone)", "290G", "ca123268-0c71-44d0-b6f2-9a205f1c3221"},
        {"HDD(Seoul-M2 Zone)", "300G", "c866a351-1946-44c8-92d5-188aacfed821"},
        {"HDD(Seoul-M2 Zone)", "400G", "6e92f540-72ab-4580-a696-9bd1e8a31289"},
        {"HDD(Seoul-M2 Zone)", "500G", "4ff4ff48-851d-4131-816d-8dabd2f0a82a"},

		{"SSD", "100G", "0f587eed-cb8f-4b06-8658-6c7e317056fe"},
        {"SSD", "200G", "9413a1c8-3d1f-4ed7-9b9f-8ad889fee0a4"},
        {"SSD", "300G", "ddd14a91-9fcd-4df1-91f7-c8ab72735e16"},
        {"SSD", "400G", "f5466c9f-2d61-4c1b-a611-c8f064bad325"},
        {"SSD", "500G", "27c98e80-c75e-4db3-abf2-ff3097d5b9d9"},
        {"SSD", "600G", "d2850362-36f8-43cd-ab54-6b5bb70082b7"},
        {"SSD", "700G", "b8776967-f962-4c94-a534-2d67133be2b2"},
        {"SSD", "800G", "1f8ee43e-c1bf-49a7-b2fc-91e94155b0e8"},

		{"SSD(Seoul-M2 Zone)", "10G", "a71ba83c-9631-471c-bcfe-07686d99d10a"},
		{"SSD(Seoul-M2 Zone)", "20G", "6edfa457-fdd3-448a-be2b-a16aed9da892"},
		{"SSD(Seoul-M2 Zone)", "30G", "0998d838-e31b-4bc2-9c15-7de463b7c484"},
		{"SSD(Seoul-M2 Zone)", "40G", "2a2a80d3-6ef8-4ffe-bee1-d560fb401ab4"},
		{"SSD(Seoul-M2 Zone)", "50G", "a1e0a2c4-4ab0-46e1-b3fe-bbc039153e00"},
		{"SSD(Seoul-M2 Zone)", "60G", "f0d091e7-7aaf-4785-9692-cd00c4026d83"},
		{"SSD(Seoul-M2 Zone)", "70G", "dffb7c6c-fa95-4335-99af-a1ed2e4b2f68"},
		{"SSD(Seoul-M2 Zone)", "80G", "adbdfd26-40d6-4aa4-9a61-76124ba56871"},
		{"SSD(Seoul-M2 Zone)", "90G", "1f302d21-0a4c-4af9-a6ef-f124d52f7f4d"},
		{"SSD(Seoul-M2 Zone)", "100G", "d5fce583-9741-498b-a96b-e041840ec22e"},
		{"SSD(Seoul-M2 Zone)", "110G", "6ea5f9bc-0999-414f-b1e3-3e3f014671ea"},
		{"SSD(Seoul-M2 Zone)", "120G", "7f61a093-5b54-42c9-9cc5-c309845e6030"},
		{"SSD(Seoul-M2 Zone)", "130G", "fe50bc0d-fde8-4b90-bef7-96316375d788"},
		{"SSD(Seoul-M2 Zone)", "140G", "2cdf8d5e-5718-4a6e-a4ac-4d4c43bf6e2f"},
		{"SSD(Seoul-M2 Zone)", "150G", "0df94ff3-37ee-45e2-ac6f-6ba97bf25288"},
		{"SSD(Seoul-M2 Zone)", "160G", "bc3b908f-7398-4f41-966b-0e71adbba34e"},
		{"SSD(Seoul-M2 Zone)", "170G", "18ee1236-664e-45f1-a19a-8d3aec2c618c"},
		{"SSD(Seoul-M2 Zone)", "180G", "ee18cd67-6df5-44ca-b8f5-e737d036a379"},
		{"SSD(Seoul-M2 Zone)", "190G", "8cc9b842-de49-4ec3-a011-f0b01b893cb5"},
		{"SSD(Seoul-M2 Zone)", "200G", "f40313a9-a2b2-4876-9cd4-eed77b0b27e6"},
		{"SSD(Seoul-M2 Zone)", "210G", "60422f3a-c32d-4604-a64c-0779b66a8898"},
		{"SSD(Seoul-M2 Zone)", "220G", "280bb389-3455-4b0d-98f0-99913ddee438"},
		{"SSD(Seoul-M2 Zone)", "230G", "9778e8ab-c3e0-4277-942b-c635a3bfdc07"},
		{"SSD(Seoul-M2 Zone)", "240G", "38eaa9cb-f91a-4ad9-8917-3e7fcdcb9961"},
		{"SSD(Seoul-M2 Zone)", "250G", "6247e1ad-60b2-416f-b721-36fbf1cc3145"},
		{"SSD(Seoul-M2 Zone)", "260G", "8a8dd03f-4673-4f98-b9af-a6ceb99c6c08"},
		{"SSD(Seoul-M2 Zone)", "270G", "576d93e7-aaa4-4764-a01f-9972543af4f2"},
		{"SSD(Seoul-M2 Zone)", "280G", "b4076e86-d5fc-4d18-aaa9-899bb180167d"},
		{"SSD(Seoul-M2 Zone)", "290G", "9c93de21-d7a4-4a6f-be15-3988ae3aea7e"},
		{"SSD(Seoul-M2 Zone)", "300G", "d4c0398d-0f56-4c81-a8da-44e225097a17"},
		{"SSD(Seoul-M2 Zone)", "400G", "65cee03d-c4f1-4f6a-a9d3-c90b3b5586a3"},
		{"SSD(Seoul-M2 Zone)", "500G", "9cb3594d-1bef-4bd2-8df4-eea3b05d36f3"},

		{"SSD-Provisioned", "100G", "41b6318e-08b1-46df-9e01-b930dccedbdf"},
		{"SSD-Provisioned", "200G", "88f9987f-4b8f-447f-adf3-937adafff33f"},
		{"SSD-Provisioned", "300G", "62238ffd-3db2-48d7-b8a1-63b0afa97a7c"},
		{"SSD-Provisioned", "400G", "fc66c4c6-d8b4-4b41-accb-15aca0c82371"},
		{"SSD-Provisioned", "500G", "b59d5b5c-0ee9-4db0-813e-52fb0138bd79"},
		{"SSD-Provisioned", "600G", "f9a1d04a-5478-4966-a2c9-4511b7526915"},
		{"SSD-Provisioned", "700G", "5aaed1b9-f69e-4e26-914f-976c3c05598d"},
		{"SSD-Provisioned", "800G", "3e6fff31-3f65-448f-9344-3b571a23ed7c"},
	}
}

// # Searches for the 'diskofferingid' based on 'type' and 'size'.
func findDiskOfferingId(diskType, size string, offerings []DiskOffering) (string, error) {
	for _, offering := range offerings {
		if strings.EqualFold(offering.Type, diskType) && strings.EqualFold(offering.Size, size) {
			return offering.DiskOfferingId, nil
		}
	}
	newErr := fmt.Errorf("Failed to Find the 'diskofferingid' for %s of size %s.", diskType, size)
	return "", newErr
}
