package resources

import (
	"context"
	"strconv"
	"strings"
	"time"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
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
)

func (DiskHandler *GCPDiskHandler) CreateDisk(diskReqInfo irs.DiskInfo) (irs.DiskInfo, error) {
	projectID := DiskHandler.Credential.ProjectID
	region := DiskHandler.Region.Region
	zone := DiskHandler.Region.Zone
	diskName := diskReqInfo.IId.NameId

	disk := &compute.Disk{
		Name: diskName,
		Type: diskReqInfo.DiskType,
	}

	if diskReqInfo.DiskSize != "" && diskReqInfo.DiskSize != "default" {
		diskSize, err := strconv.ParseInt(diskReqInfo.DiskSize, 10, 64)
		if err != nil {
			cblogger.Error(err)
			return irs.DiskInfo{}, err
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
		return []*irs.DiskInfo{}, err
	}

	for _, disk := range diskList.Items {
		nlbInfo, err := DiskHandler.GetDisk(irs.IID{SystemId: disk.Name})
		if err != nil {
			cblogger.Error(err)
			return []*irs.DiskInfo{}, err
		}
		diskInfoList = append(diskInfoList, &nlbInfo)
	}

	return diskInfoList, nil
}

func (DiskHandler *GCPDiskHandler) GetDisk(diskIID irs.IID) (irs.DiskInfo, error) {
	diskInfo := irs.DiskInfo{}

	projectID := DiskHandler.Credential.ProjectID
	zone := DiskHandler.Region.Zone

	diskResp, err := DiskHandler.Client.Disks.Get(projectID, zone, diskIID.SystemId).Do()
	if err != nil {
		cblogger.Error(err)
		return irs.DiskInfo{}, err
	}

	diskInfo.IId = diskIID
	diskInfo.DiskSize = strconv.FormatInt(diskResp.SizeGb, 10)
	diskInfo.CreatedTime, _ = time.Parse(time.RFC3339, diskResp.CreationTimestamp)

	if diskResp.Users != nil {
		arrUsers := strings.Split(diskResp.Users[0], "/")
		ownerVM := arrUsers[len(arrUsers)-1]
		diskInfo.OwnerVM = irs.IID{SystemId: ownerVM}
	}

	arrType := strings.Split(diskResp.Type, "/")
	diskInfo.DiskType = arrType[len(arrType)-1]

	if diskResp.Status == GCPDiskCreating {
		diskInfo.Status = irs.DiskCreating
	} else if diskResp.Status == GCPDiskDeleting {
		diskInfo.Status = irs.DiskDeleting
	} else if diskResp.Status == GCPDiskFailed {

	} else if diskResp.Status == GCPDiskReady {
		if diskResp.Users != nil {
			diskInfo.Status = irs.DiskAttached
		} else {
			diskInfo.Status = irs.DiskAvailable
		}
	}

	return diskInfo, nil
}

func (DiskHandler *GCPDiskHandler) ChangeDiskSize(diskIID irs.IID, size string) (bool, error) {
	projectID := DiskHandler.Credential.ProjectID
	zone := DiskHandler.Region.Zone
	disk := diskIID.SystemId
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

	return true, nil
}

func (DiskHandler *GCPDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {
	projectID := DiskHandler.Credential.ProjectID
	zone := DiskHandler.Region.Zone
	disk := diskIID.SystemId

	op, err := DiskHandler.Client.Disks.Delete(projectID, zone, disk).Do()
	if err != nil {
		cblogger.Info(op)
		cblogger.Error(err)
		return false, err
	}

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
	zone := DiskHandler.Region.Zone
	instance := ownerVM.SystemId
	deviceName := ""

	ownerVMInfo, err := DiskHandler.Client.Instances.Get(projectID, zone, instance).Do()
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	for _, diskInfo := range ownerVMInfo.Disks {
		arrDiskName := strings.Split(diskInfo.Source, "/")
		diskName := arrDiskName[len(arrDiskName)-1]
		if strings.EqualFold(diskName, diskIID.SystemId) {
			deviceName = diskInfo.DeviceName
		}
	}

	op, err := DiskHandler.Client.Instances.DetachDisk(projectID, zone, instance, deviceName).Do()
	if err != nil {
		cblogger.Info(op)
		cblogger.Error(err)
		return false, err
	}

	return true, nil
}
