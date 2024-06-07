package resources

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"
	compute "google.golang.org/api/compute/v1"
)

type GCPDiskHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

const (
	GCPDiskCreating string = "CREATING"
	GCPDiskReady    string = "READY"
	GCPDiskFailed   string = "FAILED"
	GCPDiskDeleting string = "DELETING"

	DefaultDiskType string = "pd-standard"
)

// disk 생성
func (DiskHandler *GCPDiskHandler) CreateDisk(diskReqInfo irs.DiskInfo) (irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(DiskHandler.Region, call.DISK, diskReqInfo.IId.NameId, "CreateDisk()")
	start := call.Start()

	projectID := DiskHandler.Credential.ProjectID
	region := DiskHandler.Region.Region
	zone := DiskHandler.Region.Zone
	diskName := diskReqInfo.IId.NameId

	if diskReqInfo.Zone != "" { // #1067 disk의 zone이 있으면 해당 zone 사용.
		cblogger.Info("SetDisk zone before ", DiskHandler.Region)
		zone = diskReqInfo.Zone
		DiskHandler.Region.Zone = zone // Region은 동일할 것이고 zone을 새로 설정.
		cblogger.Info("SetDisk zone after ", DiskHandler.Region)
	}

	disk := &compute.Disk{
		Name: diskName,
	}

	if diskReqInfo.DiskType != "" && diskReqInfo.DiskType != "default" {
		disk.Type = "projects/" + projectID + "/zones/" + zone + "/diskTypes/" + diskReqInfo.DiskType
	} else {
		diskReqInfo.DiskType = DefaultDiskType
	}

	if diskReqInfo.DiskSize != "" && diskReqInfo.DiskSize != "default" {
		diskSize, err := strconv.ParseInt(diskReqInfo.DiskSize, 10, 64)
		if err != nil {
			cblogger.Error(err)
			return irs.DiskInfo{}, err
		}

		//disk size validation check
		validateDiskSizeErr := validateDiskSize(diskReqInfo)
		if validateDiskSizeErr != nil {
			cblogger.Error(validateDiskSizeErr)
			return irs.DiskInfo{}, validateDiskSizeErr
		}

		disk.SizeGb = diskSize
	}

	op, err := DiskHandler.Client.Disks.Insert(projectID, zone, disk).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.DiskInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	// Disk 생성 대기
	WaitOperationComplete(DiskHandler.Client, projectID, region, zone, op.Name, 3)
	cblogger.Info("GetDisk zone ", DiskHandler.Region)
	diskInfo, errDiskInfo := DiskHandler.GetDisk(irs.IID{NameId: diskName, SystemId: diskName})
	if errDiskInfo != nil {
		cblogger.Error(errDiskInfo)
		return irs.DiskInfo{}, errDiskInfo
	}

	return diskInfo, nil
}

func (DiskHandler *GCPDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(DiskHandler.Region, call.DISK, "Disk", "ListDisk()")
	start := call.Start()

	diskInfoList := []*irs.DiskInfo{}

	projectID := DiskHandler.Credential.ProjectID
	regionID := DiskHandler.Region.Region
	//zone := DiskHandler.Region.Zone

	cblogger.Error("get ZoneInfo by region ")
	//GetRegionZone(regionName string) (irs.RegionZoneInfo, error)
	// #1067에 의해 connection의 zone -> region내 disk 조회로 변경
	regionZoneHandler := GCPRegionZoneHandler{
		Client:     DiskHandler.Client,
		Credential: DiskHandler.Credential,
		Region:     DiskHandler.Region,
		Ctx:        DiskHandler.Ctx,
	}
	regionZoneInfo, err := regionZoneHandler.GetRegionZone(regionID)
	if err != nil {
		cblogger.Error("failed to get ZoneInfo by region ", err)
		// failed to get ZoneInfo by region
		cblogger.Error(err)
		return nil, err
	} else {
		cblogger.Error("get region zone Info ", regionZoneInfo)
		for _, zoneItem := range regionZoneInfo.ZoneList {
			cblogger.Error("zone Info ", zoneItem)
			// get Disks by Zone
			hiscallInfo.ElapsedTime = call.Elapsed(start)
			diskList, err := DiskHandler.Client.Disks.List(projectID, zoneItem.Name).Do()
			if err != nil {
				cblogger.Error(err)
				LoggingError(hiscallInfo, err)
				return nil, err
			}
			calllogger.Info(call.String(hiscallInfo))

			for _, disk := range diskList.Items {
				diskInfo, err := convertDiskInfo(disk)
				if err != nil {
					cblogger.Error(err)
					return nil, err
				}
				diskInfoList = append(diskInfoList, &diskInfo)
			}
		}

	}

	return diskInfoList, nil
}

func (DiskHandler *GCPDiskHandler) GetDisk(diskIID irs.IID) (irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(DiskHandler.Region, call.DISK, diskIID.NameId, "GetDisk()")
	start := call.Start()
	cblogger.Info("GetDisk zone ", DiskHandler.Region)
	diskResp, err := GetDiskInfo(DiskHandler.Client, DiskHandler.Credential, DiskHandler.Region, diskIID.SystemId)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.DiskInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	diskInfo, errDiskInfo := convertDiskInfo(diskResp)
	if errDiskInfo != nil {
		cblogger.Error(errDiskInfo)
		return irs.DiskInfo{}, errDiskInfo
	}

	return diskInfo, nil
}

func (DiskHandler *GCPDiskHandler) ChangeDiskSize(diskIID irs.IID, size string) (bool, error) {
	hiscallInfo := GetCallLogScheme(DiskHandler.Region, call.DISK, diskIID.NameId, "ChangeDiskSize()")
	start := call.Start()

	projectID := DiskHandler.Credential.ProjectID
	region := DiskHandler.Region.Region
	zone := DiskHandler.Region.Zone
	disk := diskIID.SystemId

	diskInfo, err := DiskHandler.GetDisk(diskIID)
	if err != nil {
		return false, err
	}

	err = validateChangeDiskSize(diskInfo, size)
	if err != nil {
		return false, err
	}

	newSize, err := strconv.ParseInt(size, 10, 64)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	diskSize := &compute.DisksResizeRequest{
		SizeGb: newSize,
	}

	op, err := DiskHandler.Client.Disks.Resize(projectID, zone, disk, diskSize).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(op)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	calllogger.Info(call.String(hiscallInfo))

	WaitOperationComplete(DiskHandler.Client, projectID, region, zone, op.Name, 3)

	return true, nil
}

func (DiskHandler *GCPDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(DiskHandler.Region, call.DISK, diskIID.NameId, "DeleteDisk()")
	start := call.Start()

	projectID := DiskHandler.Credential.ProjectID
	region := DiskHandler.Region.Region
	zone := DiskHandler.Region.Zone
	targetZone := DiskHandler.Region.TargetZone
	disk := diskIID.SystemId

	// 대상 zone이 다른경우 targetZone을 사용
	if targetZone != "" {
		zone = targetZone
	}

	op, err := DiskHandler.Client.Disks.Delete(projectID, zone, disk).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(op)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	calllogger.Info(call.String(hiscallInfo))

	WaitOperationComplete(DiskHandler.Client, projectID, region, zone, op.Name, 3)

	return true, nil
}

func (DiskHandler *GCPDiskHandler) AttachDisk(diskIID irs.IID, ownerVM irs.IID) (irs.DiskInfo, error) {
	// disk와 vm의 zone valid check
	attachDiskInfo, err := DiskHandler.GetDisk(diskIID)
	if err != nil {
		cblogger.Error(err)
		return irs.DiskInfo{}, err
	}

	// check disk status : "available" state only
	if attachDiskInfo.Status != irs.DiskStatus("Available") {
		return irs.DiskInfo{}, errors.New(string("The disk must be in the Available state : " + attachDiskInfo.Status))
	}

	vmHandler := GCPVMHandler{
		Client:     DiskHandler.Client,
		Region:     DiskHandler.Region,
		Ctx:        DiskHandler.Ctx,
		Credential: DiskHandler.Credential,
	}
	vmInfo, err := vmHandler.GetVmById(ownerVM)
	if err != nil {
		cblogger.Error(err.Error())
		return irs.DiskInfo{}, err
	}

	if vmInfo.Region.Zone != attachDiskInfo.Zone {
		cblogger.Error("The disk and the VM must be in the same zone."+vmInfo.Region.Zone, attachDiskInfo.Zone)
		return irs.DiskInfo{}, errors.New(string("The disk and the VM must be in the same zone."))
	}

	// vmStatus는 다시 조회해야 하기 때문에 attach할 수 있는 상태가 아니면 오류로 return
	// valid check end

	hiscallInfo := GetCallLogScheme(DiskHandler.Region, call.DISK, diskIID.NameId, "AttachDisk()")
	start := call.Start()

	projectID := DiskHandler.Credential.ProjectID
	region := DiskHandler.Region.Region
	//zone := DiskHandler.Region.Zone
	zone := vmInfo.Region.Zone // vm의 zone으로 설정
	instance := ownerVM.SystemId

	attachedDisk := &compute.AttachedDisk{
		Source: "/projects/" + projectID + "/zones/" + zone + "/disks/" + diskIID.SystemId,
	}

	op, err := DiskHandler.Client.Instances.AttachDisk(projectID, zone, instance, attachedDisk).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(op)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.DiskInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	WaitOperationComplete(DiskHandler.Client, projectID, region, zone, op.Name, 3)

	// attach가 끝나면 disk정보 return
	diskInfo, errDiskInfo := DiskHandler.GetDisk(diskIID)
	if errDiskInfo != nil {
		cblogger.Error(errDiskInfo)
		return irs.DiskInfo{}, errDiskInfo
	}

	return diskInfo, nil
}

func (DiskHandler *GCPDiskHandler) DetachDisk(diskIID irs.IID, ownerVM irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(DiskHandler.Region, call.DISK, diskIID.NameId, "DetachDisk()")
	start := call.Start()

	projectID := DiskHandler.Credential.ProjectID
	region := DiskHandler.Region.Region
	zone := DiskHandler.Region.Zone
	instance := ownerVM.SystemId
	deviceName := ""

	ownerVMInfo, err := DiskHandler.Client.Instances.Get(projectID, zone, instance).Do()
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	isExist := false
	for _, diskInfo := range ownerVMInfo.Disks {
		arrDiskName := strings.Split(diskInfo.Source, "/")
		diskName := arrDiskName[len(arrDiskName)-1]
		if strings.EqualFold(diskName, diskIID.SystemId) {
			deviceName = diskInfo.DeviceName
			isExist = true
			break
		}
	}

	if !isExist {
		return false, errors.New("Disk does not exist!")
	}

	op, err := DiskHandler.Client.Instances.DetachDisk(projectID, zone, instance, deviceName).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(op)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	calllogger.Info(call.String(hiscallInfo))

	WaitOperationComplete(DiskHandler.Client, projectID, region, zone, op.Name, 3)

	return true, nil
}

func validateDiskSize(diskInfo irs.DiskInfo) error {
	cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo("GCP")
	arrDiskSizeOfType := cloudOSMetaInfo.DiskSize

	diskSize, err := strconv.ParseInt(diskInfo.DiskSize, 10, 64)
	if err != nil {
		cblogger.Error(err)
		return err
	}

	type diskSizeModel struct {
		diskType    string
		diskMinSize int64
		diskMaxSize int64
		unit        string
	}

	diskSizeValue := diskSizeModel{}
	isExists := false

	for _, diskSizeInfo := range arrDiskSizeOfType {
		diskSizeArr := strings.Split(diskSizeInfo, "|")
		if strings.EqualFold(diskInfo.DiskType, diskSizeArr[0]) {
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
		return errors.New("Invalid Disk Type : " + diskInfo.DiskType)
	}

	if diskSize < diskSizeValue.diskMinSize {
		cblogger.Error("Disk Size Error!!: ", diskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
		return errors.New("Disk Size must be at least the minimum size (" + strconv.FormatInt(diskSizeValue.diskMinSize, 10) + " GB).")
	}

	if diskSize > diskSizeValue.diskMaxSize {
		cblogger.Error("Disk Size Error!!: ", diskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
		return errors.New("Disk Size must be smaller than or equal to the maximum size (" + strconv.FormatInt(diskSizeValue.diskMaxSize, 10) + " GB).")
	}

	return nil
}

func validateChangeDiskSize(diskInfo irs.DiskInfo, newSize string) error {
	cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo("GCP")
	arrDiskSizeOfType := cloudOSMetaInfo.DiskSize

	diskSize, err := strconv.ParseInt(diskInfo.DiskSize, 10, 64)
	if err != nil {
		cblogger.Error(err)
		return err
	}

	newDiskSize, err := strconv.ParseInt(newSize, 10, 64)
	if err != nil {
		cblogger.Error(err)
		return err
	}

	if diskSize >= newDiskSize {
		return errors.New("Target Disk Size: " + newSize + " must be larger than existing Disk Size " + diskInfo.DiskSize)
	}

	type diskSizeModel struct {
		diskType    string
		diskMinSize int64
		diskMaxSize int64
		unit        string
	}

	diskSizeValue := diskSizeModel{}

	for _, diskSizeInfo := range arrDiskSizeOfType {
		diskSizeArr := strings.Split(diskSizeInfo, "|")
		if strings.EqualFold(diskInfo.DiskType, diskSizeArr[0]) {
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
		}
	}

	if newDiskSize > diskSizeValue.diskMaxSize {
		cblogger.Error("Disk Size Error!!: ", diskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
		return errors.New("Disk Size must be smaller than or equal to the maximum size (" + strconv.FormatInt(diskSizeValue.diskMaxSize, 10) + " GB).")
	}

	return nil
}

func convertGCPStatusToDiskStatus(status string, users []string) (irs.DiskStatus, error) {
	var returnStatus irs.DiskStatus

	if status == GCPDiskCreating {
		returnStatus = irs.DiskCreating
	} else if status == GCPDiskDeleting {
		returnStatus = irs.DiskDeleting
	} else if status == GCPDiskFailed {
		returnStatus = irs.DiskError
	} else if status == GCPDiskReady {
		if users != nil {
			returnStatus = irs.DiskAttached
		} else {
			returnStatus = irs.DiskAvailable
		}
	}

	return returnStatus, nil

}

func convertDiskInfo(diskResp *compute.Disk) (irs.DiskInfo, error) {
	diskInfo := irs.DiskInfo{}

	diskInfo.IId = irs.IID{NameId: diskResp.Name, SystemId: diskResp.Name}
	diskInfo.DiskSize = strconv.FormatInt(diskResp.SizeGb, 10)
	diskInfo.CreatedTime, _ = time.Parse(time.RFC3339, diskResp.CreationTimestamp)
	//diskInfo.Zone = diskResp.Zone // diskResp의 zone은 url 형태이므로 zone 만 추출
	index := strings.Index(diskResp.Zone, "zones/") // "zones/"의 인덱스를 찾음
	if index != -1 {
		diskInfo.Zone = diskResp.Zone[index+len("zones/"):] // "zones/" 다음의 문자열을 추출
	} else {
		diskInfo.Zone = diskResp.Zone
	}

	// Users : the users of the disk (attached instances)
	if diskResp.Users != nil {
		arrUsers := strings.Split(diskResp.Users[0], "/")
		ownerVM := arrUsers[len(arrUsers)-1]
		diskInfo.OwnerVM = irs.IID{NameId: ownerVM, SystemId: ownerVM}
	}

	arrDiskType := strings.Split(diskResp.Type, "/")
	diskInfo.DiskType = arrDiskType[len(arrDiskType)-1]

	diskStatus, errStatus := convertGCPStatusToDiskStatus(diskResp.Status, diskResp.Users)
	if errStatus != nil {
		return irs.DiskInfo{}, errStatus
	}

	diskInfo.Status = diskStatus

	return diskInfo, nil
}
