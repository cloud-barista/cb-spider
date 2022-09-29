package resources

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"
	"github.com/davecgh/go-spew/spew"
)

type AlibabaDiskHandler struct {
	Region idrv.RegionInfo
	Client *ecs.Client
}

type DiskSize struct {
	diskType    string
	diskMinSize int64
	diskMaxSize int64
	unit        string
}

const (
	ALIBABA_DISK_STATUS_AVAILABLE = "Available"
	ALIBABA_DISK_STATUS_INUSE     = "In_use"
	ALIBABA_DISK_STATUS_ATTACHING = "Attaching"
	ALIBABA_DISK_STATUS_DETACHING = "Detaching"
	ALIBABA_DISK_STATUS_CREATING  = "Creating"
	ALIBABA_DISK_STATUS_REINITING = "ReIniting"
)

/*
*
create 시 특정 instance에 바로 attach 가능하나 CB-SPIDER에서는 사용하지 않음
*/
func (diskHandler *AlibabaDiskHandler) CreateDisk(diskReqInfo irs.DiskInfo) (irs.DiskInfo, error) {
	cblogger.Info("Start CreateDisk : ", diskReqInfo)

	err := validateCreateDisk(&diskReqInfo)
	if err != nil {
		return irs.DiskInfo{}, err
	}

	destinationResource := "DataDisk"
	resourceType := "disk" // instance, disk, reservedinstance, ddh
	//client *ecs.Client, regionId string, zoneId string, resourceType string, destinationResource string, categoryValue string
	_, err = DescribeAvailableResource(diskHandler.Client, diskHandler.Region.Region, diskHandler.Region.Zone, resourceType, destinationResource, diskReqInfo.DiskType)
	if err != nil {
		return irs.DiskInfo{}, err
	}

	request := ecs.CreateCreateDiskRequest()
	request.Scheme = "https"
	// 필수 Req Name
	request.ZoneId = diskHandler.Region.Zone
	request.DiskName = diskReqInfo.IId.NameId
	request.DiskCategory = diskReqInfo.DiskType
	request.Size = requests.Integer(diskReqInfo.DiskSize)

	request.Tag = &[]ecs.CreateDiskTag{ // Default Hidden Tags Info
		{
			Key:   CBMetaDefaultTagName,  // "cbCat",
			Value: CBMetaDefaultTagValue, // "cbAlibaba",
		},
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   diskHandler.Region.Zone,
		ResourceType: call.DISK,
		ResourceName: diskReqInfo.IId.SystemId,
		CloudOSAPI:   "CreateDisk()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	spew.Dump(request)
	callLogStart := call.Start()
	// Creates a new custom Image with the given name
	result, err := diskHandler.Client.CreateDisk(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Errorf("Unable to create Disk: %s, %v.", diskReqInfo.IId.NameId, err)
		return irs.DiskInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Infof("Created Disk %q %s\n %s\n", result.DiskId, diskReqInfo.IId.NameId, result.RequestId)
	spew.Dump(result)

	// 생성된 Disk 정보 획득 후, Image 정보 리턴
	diskInfo, err := diskHandler.GetDisk(irs.IID{SystemId: result.DiskId})
	if err != nil {
		return irs.DiskInfo{}, err
	}

	return diskInfo, nil
}

/*
*
Root-Disk, Data-Disk 구분 없이 모든 Disk 목록을 제공한다.
*/
func (diskHandler *AlibabaDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	cblogger.Debug("Start")
	var diskInfoList []*irs.DiskInfo

	aliDiskInfoList, err := DescribeDisks(diskHandler.Client, diskHandler.Region, irs.IID{}, nil)
	if err != nil {
		return nil, err
	}

	//regionID := diskHandler.Region.Region
	//
	//request := ecs.CreateDescribeDisksRequest()
	//request.Scheme = "https"
	//request.RegionId = regionID
	//
	//if CBPageOn {
	//	request.PageNumber = requests.NewInteger(CBPageNumber)
	//	request.PageSize = requests.NewInteger(CBPageSize)
	//}
	//
	//// logger for HisCall
	//callogger := call.GetLogger("HISCALL")
	//callLogInfo := call.CLOUDLOGSCHEMA{
	//	CloudOS:      call.ALIBABA,
	//	RegionZone:   diskHandler.Region.Zone,
	//	ResourceType: call.DISK,
	//	ResourceName: "ListDisk()",
	//	CloudOSAPI:   "DescribeDisks()",
	//	ElapsedTime:  "",
	//	ErrorMSG:     "",
	//}
	//
	//callLogStart := call.Start()
	//
	//var totalCount = 0
	//curPage := CBPageNumber
	//for {
	//	spew.Dump(request)
	//	result, err := diskHandler.Client.DescribeDisks(request)
	//	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	//	//spew.Dump(result) //출력 정보가 너무 많아서 생략
	//	if err != nil {
	//		callLogInfo.ErrorMSG = err.Error()
	//		callogger.Error(call.String(callLogInfo))
	//
	//		cblogger.Errorf("Unable to get Images, %v", err)
	//		return nil, err
	//	}
	//	callogger.Info(call.String(callLogInfo))
	//
	//	//cnt := 0
	//	for _, cur := range result.Disks.Disk {
	//		cblogger.Debugf("[%s] Image 정보 처리", cur.ImageId)
	//		diskInfo, err := ExtractDiskDescribeInfo(&cur)
	//		if err != nil {
	//			cblogger.Error(err)
	//		}
	//		diskInfoList = append(diskInfoList, &diskInfo)
	//	}
	//
	//	if CBPageOn {
	//		totalCount = len(diskInfoList)
	//		cblogger.Infof("CSP 전체 이미지 갯수 : [%d] - 현재 페이지:[%d] - 누적 결과 개수:[%d]", result.TotalCount, curPage, totalCount)
	//		if totalCount >= result.TotalCount {
	//			break
	//		}
	//		curPage++
	//		request.PageNumber = requests.NewInteger(curPage)
	//	} else {
	//		break
	//	}
	//}
	//spew.Dump(imageInfoList)

	for _, aliDisk := range aliDiskInfoList {
		diskInfo, err := ExtractDiskDescribeInfo(&aliDisk)
		if err != nil {
			continue
		}
		diskInfoList = append(diskInfoList, &diskInfo)
	}

	return diskInfoList, nil
}

func (diskHandler *AlibabaDiskHandler) GetDisk(diskIID irs.IID) (irs.DiskInfo, error) {
	cblogger.Infof("diskID : ", diskIID.SystemId)

	resultDisk, err := DescribeDiskByDiskId(diskHandler.Client, diskHandler.Region, diskIID)
	//request := ecs.CreateDescribeDisksRequest()
	//request.Scheme = "https"
	//
	//// request.Status = "Available"
	//// request.ActionType = "*"
	//
	//diskIIDList := []string{diskIID.SystemId}
	//diskJson, err := json.Marshal(diskIIDList)
	//if err != nil {
	//
	//}
	////diskIIDList := []string{"\"" + diskIID.SystemId + "\""}
	//request.DiskIds = string(diskJson)
	//
	//// logger for HisCall
	//callogger := call.GetLogger("HISCALL")
	//callLogInfo := call.CLOUDLOGSCHEMA{
	//	CloudOS:      call.ALIBABA,
	//	RegionZone:   diskHandler.Region.Zone,
	//	ResourceType: call.DISK,
	//	ResourceName: diskIID.SystemId,
	//	CloudOSAPI:   "DescribeDisks()",
	//	ElapsedTime:  "",
	//	ErrorMSG:     "",
	//}
	//
	//callLogStart := call.Start()
	//result, err := diskHandler.Client.DescribeDisks(request)
	//callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	//
	//cblogger.Info(result)
	//if err != nil {
	//	callLogInfo.ErrorMSG = err.Error()
	//	callogger.Error(call.String(callLogInfo))
	//
	//	cblogger.Errorf("Unable to get Images, %v", err)
	//	return irs.DiskInfo{}, err
	//}
	//callogger.Info(call.String(callLogInfo))
	//
	//if result.TotalCount < 1 {
	//	return irs.DiskInfo{}, errors.New("Notfound: '" + diskIID.SystemId + "' Disks Not found")
	//}
	//
	//diskInfo, err := ExtractDiskDescribeInfo(&result.Disks.Disk[0])

	diskInfo, err := ExtractDiskDescribeInfo(&resultDisk)
	return diskInfo, err
}

func (diskHandler *AlibabaDiskHandler) ChangeDiskSize(diskIID irs.IID, size string) (bool, error) {

	diskInfo, err := diskHandler.GetDisk(diskIID)
	if err != nil {
		return false, err
	}

	err = validateModifyDisk(diskInfo, size)
	if err != nil {
		return false, err
	}

	request := ecs.CreateResizeDiskRequest()
	request.DiskId = diskIID.SystemId
	request.NewSize = requests.Integer(size)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   diskHandler.Region.Zone,
		ResourceType: call.DISK,
		ResourceName: diskIID.SystemId,
		CloudOSAPI:   "ChangeDiskSize()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()

	result, err := diskHandler.Client.ResizeDisk(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Info(result)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Errorf("Unable to resize Disk: %s, %v.", diskIID.SystemId, err)
		return false, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Infof("Successfully resized %q Disk\n", diskIID.SystemId)

	return true, nil
}

func (diskHandler *AlibabaDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {
	cblogger.Infof("DeleteDisk : [%s]", diskIID.SystemId)
	// Delete the Image by Id

	request := ecs.CreateDeleteDiskRequest()
	request.Scheme = "https"

	request.DiskId = diskIID.SystemId

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   diskHandler.Region.Zone,
		ResourceType: call.DISK,
		ResourceName: diskIID.SystemId,
		CloudOSAPI:   "DeleteDisk()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	result, err := diskHandler.Client.DeleteDisk(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Info(result)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Errorf("Unable to delete Disk: %s, %v.", diskIID.SystemId, err)
		return false, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Infof("Successfully deleted %q Disk\n", diskIID.SystemId)

	return true, nil
}

func (diskHandler *AlibabaDiskHandler) AttachDisk(diskIID irs.IID, ownerVM irs.IID) (irs.DiskInfo, error) {
	diskInfo, err := diskHandler.GetDisk(diskIID)
	if err != nil {
		return irs.DiskInfo{}, err
	}
	// check disk status : "available" state only
	if diskInfo.Status != irs.DiskStatus("Available") {
		return irs.DiskInfo{}, errors.New(string("The disk must be in the Available state : " + diskInfo.Status))
	}

	vmHandler := AlibabaVMHandler{Client: diskHandler.Client}
	vmStatus, err := vmHandler.GetVMStatus(ownerVM)
	// check instance status : "running", "stopped" status only
	if err != nil {
		cblogger.Error(err.Error())
		return irs.DiskInfo{}, err
	}

	if vmStatus != irs.VMStatus("Running") && vmStatus != irs.VMStatus("Suspended") {
		return irs.DiskInfo{}, errors.New(string("The instance state must be in the running or stopped. [" + vmStatus + "]"))
	}

	cblogger.Infof("AttachDisk : [%s]", diskIID.SystemId)
	// Delete the Image by Id

	request := ecs.CreateAttachDiskRequest()
	request.Scheme = "https"

	request.DiskId = diskIID.SystemId
	request.InstanceId = ownerVM.SystemId

	// logger for HisCall
	//callogger := call.GetLogger("HISCALL")
	//callLogInfo := call.CLOUDLOGSCHEMA{
	//	CloudOS:      call.ALIBABA,
	//	RegionZone:   diskHandler.Region.Zone,
	//	ResourceType: call.DISK,
	//	ResourceName: diskIID.SystemId,
	//	CloudOSAPI:   "AttachDisk()",
	//	ElapsedTime:  "",
	//	ErrorMSG:     "",
	//}
	//
	//callLogStart := call.Start()
	//result, err := diskHandler.Client.AttachDisk(request)
	err = AttachDisk(diskHandler.Client, diskHandler.Region, ownerVM, diskIID)
	//callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		cblogger.Errorf("Unable to attach Disk: %s, %v.", diskIID.SystemId, err)
		return irs.DiskInfo{}, err
	}

	cblogger.Infof("Successfully attached  %q Disk\n", diskIID.SystemId)
	newDiskInfo, err := diskHandler.GetDisk(irs.IID{SystemId: diskIID.SystemId})

	return newDiskInfo, err
}

/*
*
The disk must be attached to an ECS instance and in the In Use (In_Use) state.
The instance from which you want to detach the data disk must be in the Running (Running) or Stopped (Stopped) state.
The instance from which you want to detach the system disk must be in the Stopped (Stopped) state.
If OperationLocks in the response contains "LockReason" : "security" when you query information of an instance, the instance is locked for security reasons and all operations cannot take effect on the instance.
DetachDisk is an asynchronous operation. It takes about 1 minute for a disk to be detached from an instance after the operation is called.
*/
func (diskHandler *AlibabaDiskHandler) DetachDisk(diskIID irs.IID, ownerVM irs.IID) (bool, error) {
	diskInfo, err := diskHandler.GetDisk(diskIID)
	if err != nil {
		return false, err
	}
	// check disk status : "available" state only
	if diskInfo.Status != irs.DiskStatus("Attached") {
		return false, errors.New(string("The disk must be attached to an instance " + diskInfo.Status))
	}

	vmHandler := AlibabaVMHandler{Client: diskHandler.Client}
	vmStatus, err := vmHandler.GetVMStatus(ownerVM)
	// check instance status : "running", "stopped" status only
	if err != nil {
		cblogger.Error(err.Error())
		return false, err
	}

	cblogger.Info("===>VM Status : ", vmStatus)

	if vmStatus != irs.VMStatus("Running") && vmStatus != irs.VMStatus("Suspended") {
		return false, errors.New(string("The instance state must be in the running or stopped. [" + vmStatus + "]"))
	}

	cblogger.Infof("AttachDisk : [%s]", diskIID.SystemId)
	// Delete the Image by Id

	request := ecs.CreateDetachDiskRequest()
	request.Scheme = "https"

	request.DiskId = diskIID.SystemId
	request.InstanceId = ownerVM.SystemId

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   diskHandler.Region.Zone,
		ResourceType: call.DISK,
		ResourceName: diskIID.SystemId,
		CloudOSAPI:   "AttachDisk()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	result, err := diskHandler.Client.DetachDisk(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Info(result)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Errorf("Unable to detach Disk: %s, %v.", diskIID.SystemId, err)
		return false, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Infof("Successfully detached  %q Disk\n", diskIID.SystemId)
	return true, nil
}

/*
*
Disk 생성시 validation check
  - DiskType
  - DiskType 별 min/max capacity check
*/
func validateCreateDisk(diskReqInfo *irs.DiskInfo) error {
	// Check Disk Exists

	cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo("ALIBABA")
	//disktype: cloud / cloud_efficiency / cloud_ssd / cloud_essd
	//disksize: cloud|5|2000|GB / cloud_efficiency|20|32768|GB / cloud_ssd|20|32768|GB / cloud_essd_PL0|40|32768|GB / cloud_essd_PL1|20|32768|GB / cloud_essd_PL2|461|32768|GB / cloud_essd_PL3|1261|32768|GB

	arrDiskType := cloudOSMetaInfo.DiskType
	arrDiskSizeOfType := cloudOSMetaInfo.DiskSize
	arrRootDiskSizeOfType := cloudOSMetaInfo.RootDiskSize
	// Check Disk available
	// Size :
	// DiskCategory : cloud / cloud_efficiency / cloud_ssd / cloud_essd
	// valid size : cloud 5 ~ 2000, cloud_efficiency 20 ~ 32768, cloud_ssd 20 ~ 32768, cloud_essd

	reqDiskCategory := diskReqInfo.DiskType
	diskSize := diskReqInfo.DiskSize

	if reqDiskCategory == "" || reqDiskCategory == "default" {
		diskSizeArr := strings.Split(arrRootDiskSizeOfType[0], "|")
		reqDiskCategory = diskSizeArr[0]      // ESSD
		diskReqInfo.DiskType = diskSizeArr[0] // set default value
	}
	// 정의된 type인지
	if !ContainString(arrDiskType, reqDiskCategory) {
		return errors.New("Disktype : " + reqDiskCategory + "' is not valid")
	}

	if diskSize == "" || diskSize == "default" {
		diskSizeArr := strings.Split(arrRootDiskSizeOfType[0], "|")
		diskSize = diskSizeArr[1]
		diskReqInfo.DiskSize = diskSizeArr[1] // set default value
	}

	reqDiskSize, err := strconv.ParseInt(diskSize, 10, 64)
	if err != nil {
		return err
	}

	diskSizeValue := DiskSize{}
	isExists := false
	for idx, _ := range arrDiskSizeOfType {
		diskSizeArr := strings.Split(arrDiskSizeOfType[idx], "|")
		reqDiskType := diskReqInfo.DiskType
		switch reqDiskCategory {
		case "cloud_essd":
			// cloud_essd 는 performanceLevel(PL0, PL1, PL2, PL3) 에 따라 또 다시 min/max가 생김.
			// cb-spider는 performanceLevel을 관리하지 않으므로 기본값인 PL2 를 사용한다.
			// console 상 attach disk의 default는 PL1
			reqDiskType += "_PL1"
		}

		if strings.EqualFold(reqDiskType, diskSizeArr[0]) {
			diskSizeValue.diskType = diskSizeArr[0]
			diskSizeValue.unit = diskSizeArr[3]
			diskSizeValue.diskMinSize, err = strconv.ParseInt(diskSizeArr[1], 10, 64)
			if err != nil {
				cblogger.Error(err)
				return err
			}

			diskSizeValue.diskMaxSize, err = strconv.ParseInt(diskSizeArr[2], 10, 64)
			if err != nil {
				cblogger.Error(err)
				return err
			}
			isExists = true
		}
	}

	if !isExists {
		return errors.New("Invalid Disk Type : " + diskReqInfo.DiskType)
	}

	if reqDiskSize < diskSizeValue.diskMinSize {
		fmt.Println("Disk Size Error!!: ", reqDiskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
		return errors.New("Disk Size must be at least the default size (" + strconv.FormatInt(diskSizeValue.diskMinSize, 10) + " GB).")
	}

	if reqDiskSize > diskSizeValue.diskMaxSize {
		fmt.Println("Disk Size Error!!: ", reqDiskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
		return errors.New("Disk Size must be smaller than the maximum size (" + strconv.FormatInt(diskSizeValue.diskMaxSize, 10) + " GB).")
	}

	// 실제로 diskType이 유효한지 check

	return nil
}

func validateModifyDisk(diskReqInfo irs.DiskInfo, diskSize string) error {
	// volume Size
	orgDiskSize, err := strconv.ParseInt(diskReqInfo.DiskSize, 10, 64)
	if err != nil {
		return err
	}

	targetDiskSize, err := strconv.ParseInt(diskSize, 10, 64)

	if err != nil {
		return err
	}

	if orgDiskSize < targetDiskSize {
	} else {
		return errors.New("Target DiskSize : " + diskSize + " must be greater than Original DiskSize " + diskReqInfo.DiskSize)
	}

	cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo("ALIBABA")
	//arrDiskType := cloudOSMetaInfo.DiskType
	arrDiskSizeOfType := cloudOSMetaInfo.DiskSize

	//if !ContainString(arrDiskType, diskReqInfo.DiskType) {
	//	return errors.New("Disktype : " + diskReqInfo.DiskType + "' is not valid")
	//}

	diskSizeValue := DiskSize{}
	isExists := false
	for idx, _ := range arrDiskSizeOfType {
		diskSizeArr := strings.Split(arrDiskSizeOfType[idx], "|")
		reqDiskType := diskReqInfo.DiskType
		switch reqDiskType {
		case "cloud_essd":
			// cloud_essd 는 performanceLevel(PL0, PL1, PL2, PL3) 에 따라 또 다시 min/max가 생김.
			// cb-spider는 performanceLevel을 관리하지 않으므로 기본값인 PL2 를 사용한다.
			reqDiskType += "_PL2"
		}

		if strings.EqualFold(reqDiskType, diskSizeArr[0]) {
			diskSizeValue.diskType = diskSizeArr[0]
			diskSizeValue.unit = diskSizeArr[3]
			diskSizeValue.diskMinSize, err = strconv.ParseInt(diskSizeArr[1], 10, 64)
			if err != nil {
				cblogger.Error(err)
				return err
			}

			diskSizeValue.diskMaxSize, err = strconv.ParseInt(diskSizeArr[2], 10, 64)
			if err != nil {
				cblogger.Error(err)
				return err
			}
			isExists = true
		}
	}

	if !isExists {
		return errors.New("Invalid Disk Type : " + diskReqInfo.DiskType)
	}

	if targetDiskSize < diskSizeValue.diskMinSize {
		fmt.Println("Disk Size Error!!: ", targetDiskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
		return errors.New("Disk Size must be at least the default size (" + strconv.FormatInt(diskSizeValue.diskMinSize, 10) + " GB).")
	}

	if targetDiskSize > diskSizeValue.diskMaxSize {
		fmt.Println("Disk Size Error!!: ", targetDiskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
		return errors.New("Disk Size must be smaller than the maximum size (" + strconv.FormatInt(diskSizeValue.diskMaxSize, 10) + " GB).")
	}

	return nil
}

// https://pkg.go.dev/github.com/aliyun/alibaba-cloud-sdk-go/services/ecs?tab=doc#Image
// package ecs v1.61.170 Latest Published: Apr 30, 2020
// Image 정보를 추출함
func ExtractDiskDescribeInfo(aliDisk *ecs.Disk) (irs.DiskInfo, error) {

	diskInfo := irs.DiskInfo{
		IId: irs.IID{NameId: aliDisk.DiskName, SystemId: aliDisk.DiskId},
	}

	diskInfo.DiskSize = strconv.Itoa(aliDisk.Size)
	diskInfo.DiskType = aliDisk.Category
	diskInfo.CreatedTime, _ = time.Parse(
		time.RFC3339,
		aliDisk.CreationTime)
	diskInfo.OwnerVM = irs.IID{SystemId: aliDisk.InstanceId}
	diskStatus, errStatus := convertAlibabaDiskStatusToDiskStatus(aliDisk.Status)
	if errStatus != nil {
		return irs.DiskInfo{}, errStatus
	}
	diskInfo.Status = diskStatus

	keyValueList := []irs.KeyValue{
		{Key: "CreationTime", Value: aliDisk.CreationTime},
		{Key: "AttachedTime", Value: aliDisk.AttachedTime},
		{Key: "DetachedTime", Value: aliDisk.DetachedTime},
		{Key: "ExpiredTime", Value: aliDisk.ExpiredTime},

		{Key: "SerialNumber", Value: aliDisk.SerialNumber},
		{Key: "Status", Value: aliDisk.Status}, // In_use, Available, Attaching, Detaching, Creating, ReIniting
		{Key: "Size", Value: strconv.Itoa(aliDisk.Size)},
		{Key: "Category", Value: aliDisk.Category},
		{Key: "Type", Value: aliDisk.Type},                         // system, data
		{Key: "PerformanceLevel", Value: aliDisk.PerformanceLevel}, // PL0, PL1, PL2, PL3
		{Key: "BdfId", Value: aliDisk.BdfId},
		{Key: "EnableAutoSnapshot", Value: strconv.FormatBool(aliDisk.EnableAutoSnapshot)},
		{Key: "StorageSetId", Value: aliDisk.StorageSetId},
		{Key: "StorageSetPartitionNumber", Value: strconv.Itoa(aliDisk.StorageSetPartitionNumber)},

		{Key: "aliDiskId", Value: aliDisk.DiskId},
		{Key: "DeleteAutoSnapshot", Value: strconv.FormatBool(aliDisk.DeleteAutoSnapshot)},
		{Key: "Encrypted", Value: strconv.FormatBool(aliDisk.Encrypted)},
		{Key: "IOPS", Value: strconv.Itoa(aliDisk.IOPS)},
		{Key: "IOPSRead", Value: strconv.Itoa(aliDisk.IOPSRead)},
		{Key: "IOPSWrite", Value: strconv.Itoa(aliDisk.IOPSWrite)},
		{Key: "Throughput", Value: strconv.Itoa(aliDisk.Throughput)},
		{Key: "MountInstanceNum", Value: strconv.Itoa(aliDisk.MountInstanceNum)},

		{Key: "Description", Value: aliDisk.Description},
		{Key: "Device", Value: aliDisk.Device},
		{Key: "aliDiskName", Value: aliDisk.DiskName},
		{Key: "Portable", Value: strconv.FormatBool(aliDisk.Portable)},
		{Key: "ImageId", Value: aliDisk.ImageId},
		{Key: "KMSKeyId", Value: aliDisk.KMSKeyId},

		{Key: "DeleteWithInstance", Value: strconv.FormatBool(aliDisk.DeleteWithInstance)},

		{Key: "SourceSnapshotId", Value: aliDisk.SourceSnapshotId},
		{Key: "AutoSnapshotPolicyId", Value: aliDisk.AutoSnapshotPolicyId},
		{Key: "EnableAutomatedSnapshotPolicy", Value: strconv.FormatBool(aliDisk.EnableAutomatedSnapshotPolicy)},
		{Key: "InstanceId", Value: aliDisk.InstanceId},
		{Key: "RegionId", Value: aliDisk.RegionId},
		{Key: "ZoneId", Value: aliDisk.ZoneId},

		{Key: "aliDiskChargeType", Value: aliDisk.DiskChargeType},
		{Key: "ResourceGroupId", Value: aliDisk.ResourceGroupId},
		{Key: "ProductCode", Value: aliDisk.ProductCode},
		{Key: "MultiAttach", Value: aliDisk.MultiAttach},

		{Key: "ResourceGroupId", Value: aliDisk.ResourceGroupId},
	}

	//keyValueList = append(keyValueList, irs.KeyValue{Key: "Description", Value: disk.Description})
	diskInfo.KeyValueList = keyValueList

	return diskInfo, nil
}

/*
*
Alibaba 의 DiskStatus 를 CB-SPIDER의 DiskStatus 로 변환
*/
func convertAlibabaDiskStatusToDiskStatus(aliDiskStaus string) (irs.DiskStatus, error) {
	var returnStatus irs.DiskStatus

	switch aliDiskStaus {
	case ALIBABA_DISK_STATUS_INUSE:
		returnStatus = irs.DiskAttached
	case ALIBABA_DISK_STATUS_AVAILABLE:
		returnStatus = irs.DiskAvailable //
	case ALIBABA_DISK_STATUS_ATTACHING:
		returnStatus = irs.DiskAvailable
	case ALIBABA_DISK_STATUS_DETACHING:
		returnStatus = irs.DiskDeleting
	case ALIBABA_DISK_STATUS_CREATING:
		returnStatus = irs.DiskCreating
	case ALIBABA_DISK_STATUS_REINITING:
		returnStatus = irs.DiskCreating
	default:
		returnStatus = irs.DiskError
		return returnStatus, errors.New("Invalid DiskStatus: " + aliDiskStaus)
	}
	return returnStatus, nil
}
