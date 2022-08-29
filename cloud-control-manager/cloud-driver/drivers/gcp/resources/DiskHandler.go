package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

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
	projectID := DiskHandler.Credential.ProjectID
	region := DiskHandler.Region.Region
	zone := DiskHandler.Region.Zone
	diskName := diskReqInfo.IId.NameId

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
	if err != nil {
		cblogger.Error(err)
		return irs.DiskInfo{}, err
	}

	// Disk 생성 대기
	WaitOperationComplete(DiskHandler.Client, projectID, region, zone, op.Name, 3)

	diskInfo, errDiskInfo := DiskHandler.GetDisk(irs.IID{NameId: diskName, SystemId: diskName})
	if errDiskInfo != nil {
		cblogger.Error(errDiskInfo)
		return irs.DiskInfo{}, errDiskInfo
	}

	return diskInfo, nil
}

func (DiskHandler *GCPDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	diskInfoList := []*irs.DiskInfo{}

	projectID := DiskHandler.Credential.ProjectID
	zone := DiskHandler.Region.Zone

	diskList, err := DiskHandler.Client.Disks.List(projectID, zone).Do()
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	for _, disk := range diskList.Items {
		diskInfo, err := convertDiskInfo(disk)
		if err != nil {
			cblogger.Error(err)
			return nil, err
		}
		diskInfoList = append(diskInfoList, &diskInfo)
	}

	return diskInfoList, nil
}

func (DiskHandler *GCPDiskHandler) GetDisk(diskIID irs.IID) (irs.DiskInfo, error) {

	diskResp, err := GetDiskInfo(DiskHandler.Client, DiskHandler.Credential, DiskHandler.Region, diskIID.SystemId)
	if err != nil {
		cblogger.Error(err)
		return irs.DiskInfo{}, err
	}

	diskInfo, errDiskInfo := convertDiskInfo(diskResp)
	if errDiskInfo != nil {
		cblogger.Error(errDiskInfo)
		return irs.DiskInfo{}, errDiskInfo
	}

	return diskInfo, nil
}

func (DiskHandler *GCPDiskHandler) ChangeDiskSize(diskIID irs.IID, size string) (bool, error) {
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
	if err != nil {
		cblogger.Info(op)
		cblogger.Error(err)
		return false, err
	}

	WaitOperationComplete(DiskHandler.Client, projectID, region, zone, op.Name, 3)

	return true, nil
}

func (DiskHandler *GCPDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {
	projectID := DiskHandler.Credential.ProjectID
	region := DiskHandler.Region.Region
	zone := DiskHandler.Region.Zone
	disk := diskIID.SystemId

	op, err := DiskHandler.Client.Disks.Delete(projectID, zone, disk).Do()
	if err != nil {
		cblogger.Info(op)
		cblogger.Error(err)
		return false, err
	}

	WaitOperationComplete(DiskHandler.Client, projectID, region, zone, op.Name, 3)

	return true, nil
}

func (DiskHandler *GCPDiskHandler) AttachDisk(diskIID irs.IID, ownerVM irs.IID) (irs.DiskInfo, error) {
	projectID := DiskHandler.Credential.ProjectID
	region := DiskHandler.Region.Region
	zone := DiskHandler.Region.Zone
	instance := ownerVM.SystemId

	attachedDisk := &compute.AttachedDisk{
		Source: "/projects/" + projectID + "/zones/" + zone + "/disks/" + diskIID.SystemId,
	}

	op, err := DiskHandler.Client.Instances.AttachDisk(projectID, zone, instance, attachedDisk).Do()
	if err != nil {
		cblogger.Info(op)
		cblogger.Error(err)
		return irs.DiskInfo{}, err
	}

	WaitOperationComplete(DiskHandler.Client, projectID, region, zone, op.Name, 3)

	diskInfo, errDiskInfo := DiskHandler.GetDisk(diskIID)
	if errDiskInfo != nil {
		cblogger.Error(errDiskInfo)
		return irs.DiskInfo{}, errDiskInfo
	}

	return diskInfo, nil
}

func (DiskHandler *GCPDiskHandler) DetachDisk(diskIID irs.IID, ownerVM irs.IID) (bool, error) {
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
	if err != nil {
		cblogger.Info(op)
		cblogger.Error(err)
		return false, err
	}

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
		fmt.Println("Disk Size Error!!: ", diskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
		return errors.New("Disk Size must be at least the minimum size (" + strconv.FormatInt(diskSizeValue.diskMinSize, 10) + " GB).")
	}

	if diskSize > diskSizeValue.diskMaxSize {
		fmt.Println("Disk Size Error!!: ", diskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
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
		fmt.Println("Disk Size Error!!: ", diskSize, diskSizeValue.diskMinSize, diskSizeValue.diskMaxSize)
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
