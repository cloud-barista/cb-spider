package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/blockstorage/extensions/volumeactions"
	volumes2 "github.com/gophercloud/gophercloud/openstack/blockstorage/v2/volumes"
	volumes3 "github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/volumeattach"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"strconv"
	"strings"
	"sync"
	"time"
)

type OpenstackDiskHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	ComputeClient  *gophercloud.ServiceClient
	VolumeClient   *gophercloud.ServiceClient
}

const (
	VolumeV2 string = "volumev2"
	VolumeV3 string = "volumev3"
)

func (diskHandler *OpenstackDiskHandler) CreateDisk(DiskReqInfo irs.DiskInfo) (irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.CredentialInfo.IdentityEndpoint, "DISK", "DISK", "CreateDisk()")
	start := call.Start()
	err := diskHandler.CheckDiskHandler()
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}
	err = validationDiskReq(DiskReqInfo, diskHandler.VolumeClient)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}
	vol, err := createDisk(DiskReqInfo, diskHandler.VolumeClient)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}
	info, err := diskHandler.setterDisk(vol)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}
	LoggingInfo(hiscallInfo, start)
	return info, nil
}
func (diskHandler *OpenstackDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.CredentialInfo.IdentityEndpoint, "DISK", "DISK", "ListDisk()")
	start := call.Start()
	err := diskHandler.CheckDiskHandler()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List Disk. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return []*irs.DiskInfo{}, getErr
	}
	list, err := getRawDiskList(diskHandler.VolumeClient)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List Disk. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return []*irs.DiskInfo{}, getErr
	}
	infoList := make([]*irs.DiskInfo, len(list))
	for i, vol := range list {
		info, err := diskHandler.setterDisk(vol)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List Disk. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return []*irs.DiskInfo{}, getErr
		}
		infoList[i] = &info
	}
	LoggingInfo(hiscallInfo, start)

	return infoList, nil
}
func (diskHandler *OpenstackDiskHandler) GetDisk(diskIID irs.IID) (irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.CredentialInfo.IdentityEndpoint, "DISK", diskIID.NameId, "GetDisk()")
	start := call.Start()
	err := diskHandler.CheckDiskHandler()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Disk. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.DiskInfo{}, getErr
	}
	disk, err := getRawDisk(diskIID, diskHandler.VolumeClient)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Disk. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.DiskInfo{}, getErr
	}
	info, err := diskHandler.setterDisk(disk)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Disk. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.DiskInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (diskHandler *OpenstackDiskHandler) ChangeDiskSize(diskIID irs.IID, size string) (bool, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.CredentialInfo.IdentityEndpoint, "DISK", diskIID.NameId, "DeleteDisk()")
	start := call.Start()
	err := diskHandler.CheckDiskHandler()
	if err != nil {
		changeDiskSizeErr := errors.New(fmt.Sprintf("Failed to ChangeDiskSize. err = %s", err))
		cblogger.Error(changeDiskSizeErr.Error())
		LoggingError(hiscallInfo, changeDiskSizeErr)
		return false, changeDiskSizeErr
	}
	err = changeDiskSize(diskIID, size, diskHandler.VolumeClient)
	if err != nil {
		changeDiskSizeErr := errors.New(fmt.Sprintf("Failed to ChangeDiskSize. err = %s", err))
		cblogger.Error(changeDiskSizeErr.Error())
		LoggingError(hiscallInfo, changeDiskSizeErr)
		return false, changeDiskSizeErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (diskHandler *OpenstackDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.CredentialInfo.IdentityEndpoint, "DISK", diskIID.NameId, "DeleteDisk()")
	start := call.Start()
	err := diskHandler.CheckDiskHandler()
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Disk. err = %s", err))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	err = deleteDisk(diskIID, diskHandler.VolumeClient)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Disk. err = %s", err))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (diskHandler *OpenstackDiskHandler) AttachDisk(diskIID irs.IID, ownerVM irs.IID) (irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.CredentialInfo.IdentityEndpoint, "DISK", diskIID.NameId, "AttachDisk()")
	start := call.Start()
	err := diskHandler.CheckDiskHandler()
	if err != nil {
		attachErr := errors.New(fmt.Sprintf("Failed to AttachDisk. err = %s", err))
		cblogger.Error(attachErr.Error())
		LoggingError(hiscallInfo, attachErr)
		return irs.DiskInfo{}, attachErr
	}
	disk, err := attachDisk(diskIID, ownerVM, diskHandler.ComputeClient, diskHandler.VolumeClient)
	if err != nil {
		attachErr := errors.New(fmt.Sprintf("Failed to AttachDisk. err = %s", err))
		cblogger.Error(attachErr.Error())
		LoggingError(hiscallInfo, attachErr)
		return irs.DiskInfo{}, attachErr
	}
	info, err := diskHandler.setterDisk(disk)
	if err != nil {
		attachErr := errors.New(fmt.Sprintf("Failed to AttachDisk. err = %s", err))
		cblogger.Error(attachErr.Error())
		LoggingError(hiscallInfo, attachErr)
		return irs.DiskInfo{}, attachErr
	}
	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (diskHandler *OpenstackDiskHandler) DetachDisk(diskIID irs.IID, ownerVM irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.CredentialInfo.IdentityEndpoint, "DISK", diskIID.NameId, "DetachDisk()")
	start := call.Start()
	err := diskHandler.CheckDiskHandler()
	if err != nil {
		detachErr := errors.New(fmt.Sprintf("Failed to DetachDisk. err = %s", err))
		cblogger.Error(detachErr.Error())
		LoggingError(hiscallInfo, detachErr)
		return false, detachErr
	}
	err = detachDisk(diskIID, ownerVM, diskHandler.ComputeClient, diskHandler.VolumeClient)
	if err != nil {
		detachErr := errors.New(fmt.Sprintf("Failed to DetachDisk. err = %s", err))
		cblogger.Error(detachErr.Error())
		LoggingError(hiscallInfo, detachErr)
		return false, detachErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (diskHandler *OpenstackDiskHandler) CheckDiskHandler() error {
	if diskHandler.VolumeClient == nil {
		return errors.New("DiskHandler is not available on this Openstack. Please check the cinder module installation")
	}
	return nil
}

func volumes2ToVolumes3(vol2 volumes2.Volume) (volumes3.Volume, error) {
	bytes, err := json.Marshal(vol2)
	if err != nil {
		return volumes3.Volume{}, err
	}
	var vol3 volumes3.Volume
	err = json.Unmarshal(bytes, &vol3)
	if err != nil {
		return volumes3.Volume{}, err
	}
	return vol3, err
}

func getRawDiskListV2(volume2Client *gophercloud.ServiceClient) ([]volumes3.Volume, error) {
	pager2, err := volumes2.List(volume2Client, nil).AllPages()
	if err != nil {
		return nil, err
	}
	list2, err := volumes2.ExtractVolumes(pager2)
	if err != nil {
		return nil, err
	}
	newList := make([]volumes3.Volume, len(list2))
	for i, vol := range list2 {
		vol3, err := volumes2ToVolumes3(vol)
		if err != nil {
			return nil, err
		}
		newList[i] = vol3
	}
	return newList, nil
}

func getRawDiskListV3(volume3Client *gophercloud.ServiceClient) ([]volumes3.Volume, error) {
	pager3, err := volumes3.List(volume3Client, nil).AllPages()
	if err != nil {
		return nil, err
	}
	list3, err := volumes3.ExtractVolumes(pager3)
	if err != nil {
		return nil, err
	}
	return list3, nil
}

func getRawDiskV2(diskIID irs.IID, volume3Client *gophercloud.ServiceClient) (volumes3.Volume, error) {
	if diskIID.NameId == "" && diskIID.SystemId == "" {
		return volumes3.Volume{}, errors.New("invalid disk IID")
	}
	if diskIID.SystemId != "" {
		disk, err := volumes2.Get(volume3Client, diskIID.SystemId).Extract()
		if err != nil {
			return volumes3.Volume{}, err
		}
		diskVol3, err := volumes2ToVolumes3(*disk)
		if err != nil {
			return volumes3.Volume{}, err
		}
		return diskVol3, err
	}
	//volumes3.Get By NameId
	nameOpts := volumes2.ListOpts{
		Name: diskIID.NameId,
	}
	pager3, err := volumes2.List(volume3Client, nameOpts).AllPages()
	if err != nil {
		return volumes3.Volume{}, err
	}
	list3, err := volumes2.ExtractVolumes(pager3)
	if err != nil {
		return volumes3.Volume{}, err
	}
	if len(list3) < 1 {
		return volumes3.Volume{}, errors.New("not found Disk")
	}
	diskVol3, err := volumes2ToVolumes3(list3[0])
	if err != nil {
		return volumes3.Volume{}, err
	}
	return diskVol3, nil
}

func getRawDiskV3(diskIID irs.IID, volume3Client *gophercloud.ServiceClient) (volumes3.Volume, error) {
	if diskIID.NameId == "" && diskIID.SystemId == "" {
		return volumes3.Volume{}, errors.New("invalid disk IID")
	}
	if diskIID.SystemId != "" {
		disk, err := volumes3.Get(volume3Client, diskIID.SystemId).Extract()
		if err != nil {
			return volumes3.Volume{}, err
		}
		return *disk, err
	}
	//volumes3.Get By NameId
	nameOpts := volumes3.ListOpts{
		Name: diskIID.NameId,
	}
	pager3, err := volumes3.List(volume3Client, nameOpts).AllPages()
	if err != nil {
		return volumes3.Volume{}, err
	}
	list3, err := volumes3.ExtractVolumes(pager3)
	if err != nil {
		return volumes3.Volume{}, err
	}
	if len(list3) < 1 {
		return volumes3.Volume{}, errors.New("not found Disk")
	}
	return list3[0], nil
}

func getRawDisk(diskIId irs.IID, volumeClient *gophercloud.ServiceClient) (volumes3.Volume, error) {
	if volumeClient.Type == VolumeV3 {
		return getRawDiskV3(diskIId, volumeClient)
	}
	if volumeClient.Type == VolumeV2 {
		return getRawDiskV2(diskIId, volumeClient)
	}
	return volumes3.Volume{}, errors.New("VolumeClient not found")
}

func getRawDiskList(volumeClient *gophercloud.ServiceClient) ([]volumes3.Volume, error) {
	if volumeClient.Type == VolumeV3 {
		return getRawDiskListV3(volumeClient)
	}
	if volumeClient.Type == VolumeV2 {
		return getRawDiskListV2(volumeClient)
	}
	return nil, errors.New("VolumeClient not found")
}

func deleteDiskV2(diskIID irs.IID, volume2Client *gophercloud.ServiceClient) error {
	disk, err := getRawDiskV2(diskIID, volume2Client)
	if err != nil {
		return err
	}
	return volumes2.Delete(volume2Client, disk.ID, nil).ExtractErr()
}

func deleteDiskV3(diskIID irs.IID, volume3Client *gophercloud.ServiceClient) error {
	disk, err := getRawDiskV3(diskIID, volume3Client)
	if err != nil {
		return err
	}
	return volumes3.Delete(volume3Client, disk.ID, nil).ExtractErr()
}

func deleteDisk(diskIID irs.IID, volumeClient *gophercloud.ServiceClient) error {
	if volumeClient.Type == VolumeV3 {
		return deleteDiskV3(diskIID, volumeClient)
	}
	if volumeClient.Type == VolumeV2 {
		return deleteDiskV2(diskIID, volumeClient)
	}
	return errors.New("VolumeClient not found")
}

func (diskHandler *OpenstackDiskHandler) setterDisk(rawVolume volumes3.Volume) (irs.DiskInfo, error) {
	info := irs.DiskInfo{
		IId: irs.IID{
			NameId:   rawVolume.Name,
			SystemId: rawVolume.ID,
		},
		DiskType:    "default",
		DiskSize:    strconv.Itoa(rawVolume.Size),
		CreatedTime: rawVolume.CreatedAt,
	}

	if rawVolume.Attachments != nil && len(rawVolume.Attachments) > 0 {
		vmId := rawVolume.Attachments[0].ServerID
		vm, err := servers.Get(diskHandler.ComputeClient, vmId).Extract()
		if err == nil {
			info.OwnerVM = irs.IID{
				NameId:   vm.Name,
				SystemId: vm.ID,
			}
		}
	}
	// status (“available”, “error”, “creating”, “reserved”, “deleting”, “in-use”, “attaching”, “detaching”, “error_deleting” or “maintenance”)
	switch strings.ToLower(rawVolume.Status) {
	case "creating":
		info.Status = irs.DiskCreating
	case "deleting":
		info.Status = irs.DiskDeleting
	case "error", "error_deleting":
		info.Status = irs.DiskError
	case "available":
		info.Status = irs.DiskAvailable
	default:
		info.Status = irs.DiskAttached
	}
	return info, nil
}

func checkExistDiskV2(diskIID irs.IID, volume2Client *gophercloud.ServiceClient) (bool, error) {
	pager, err := volumes2.List(volume2Client, volumes2.ListOpts{}).AllPages()
	if err != nil {
		return false, err
	}
	list2, err := volumes2.ExtractVolumes(pager)
	if err != nil {
		return false, err
	}
	for _, disk := range list2 {
		if diskIID.SystemId != "" && diskIID.SystemId == disk.ID {
			return true, nil
		}
		if diskIID.NameId != "" && diskIID.NameId == disk.Name {
			return true, nil
		}
	}
	return false, nil
}

func checkExistDiskV3(diskIID irs.IID, volume3Client *gophercloud.ServiceClient) (bool, error) {
	pager, err := volumes3.List(volume3Client, volumes3.ListOpts{}).AllPages()
	if err != nil {
		return false, err
	}
	list3, err := volumes3.ExtractVolumes(pager)
	if err != nil {
		return false, err
	}
	for _, disk := range list3 {
		if diskIID.SystemId != "" && diskIID.SystemId == disk.ID {
			return true, nil
		}
		if diskIID.NameId != "" && diskIID.NameId == disk.Name {
			return true, nil
		}
	}
	return false, nil
}

func checkExistDisk(diskIID irs.IID, volumeClient *gophercloud.ServiceClient) (bool, error) {
	if diskIID.NameId == "" && diskIID.SystemId == "" {
		return false, errors.New("invalid Disk IID")
	}
	if volumeClient.Type == VolumeV3 {
		return checkExistDiskV3(diskIID, volumeClient)
	}
	if volumeClient.Type == VolumeV2 {
		return checkExistDiskV2(diskIID, volumeClient)
	}
	return false, errors.New("VolumeClient not found")
}

func validationDiskReq(diskReq irs.DiskInfo, volumeClient *gophercloud.ServiceClient) error {
	if diskReq.IId.NameId == "" {
		return errors.New("invalid DiskReqInfo NameId")
	}
	exist, err := checkExistDisk(diskReq.IId, volumeClient)
	if err != nil {
		return errors.New("failed Check disk Name Exist")
	}
	if exist {
		return errors.New("invalid DiskReqInfo NameId, Already exist")
	}
	return nil
}
func createDiskV2(diskReq irs.DiskInfo, volume2Client *gophercloud.ServiceClient) (volumes3.Volume, error) {
	size, err := strconv.Atoi(diskReq.DiskSize)
	if diskReq.DiskSize == "" || strings.ToLower(diskReq.DiskSize) == "default" {
		size = 1
		err = nil
	}
	if err != nil {
		return volumes3.Volume{}, errors.New("invalid Disk Size")
	}
	createOpt := volumes2.CreateOpts{
		Name: diskReq.IId.NameId,
		Size: size,
	}
	createVol, err := volumes2.Create(volume2Client, createOpt).Extract()
	if err != nil {
		return volumes3.Volume{}, errors.New("invalid Disk Size")
	}
	createVolV3, err := volumes2ToVolumes3(*createVol)
	if err != nil {
		return volumes3.Volume{}, errors.New("invalid Disk Size")
	}
	return createVolV3, nil
}

func createDiskV3(diskReq irs.DiskInfo, volume3Client *gophercloud.ServiceClient) (volumes3.Volume, error) {
	size, err := strconv.Atoi(diskReq.DiskSize)
	if diskReq.DiskSize == "" || strings.ToLower(diskReq.DiskSize) == "default" {
		size = 1
		err = nil
	}
	if err != nil {
		return volumes3.Volume{}, errors.New("invalid Disk Size")
	}
	createOpt := volumes3.CreateOpts{
		Name: diskReq.IId.NameId,
		Size: size,
	}
	createVol, err := volumes3.Create(volume3Client, createOpt).Extract()
	if err != nil {
		return volumes3.Volume{}, errors.New("invalid Disk Size")
	}
	return *createVol, nil
}

func createDisk(diskReq irs.DiskInfo, volumeClient *gophercloud.ServiceClient) (volumes3.Volume, error) {
	if volumeClient.Type == VolumeV3 {
		return createDiskV3(diskReq, volumeClient)
	}
	if volumeClient.Type == VolumeV2 {
		return createDiskV2(diskReq, volumeClient)
	}
	return volumes3.Volume{}, errors.New("VolumeClient not found")
}

func attachDisk(diskIID irs.IID, ownerVMIID irs.IID, computeClient *gophercloud.ServiceClient, volumeClient *gophercloud.ServiceClient) (volumes3.Volume, error) {
	if diskIID.NameId == "" && diskIID.SystemId == "" {
		return volumes3.Volume{}, errors.New("invalid Disk IID")
	}
	if ownerVMIID.NameId == "" && ownerVMIID.SystemId == "" {
		return volumes3.Volume{}, errors.New("invalid ownerVM IID")
	}
	disk, err := getRawDisk(diskIID, volumeClient)
	if err != nil {
		return volumes3.Volume{}, err
	}
	var ownerRawVM servers.Server
	if ownerVMIID.SystemId == "" {
		pager, err := servers.List(computeClient, nil).AllPages()
		if err != nil {
			return volumes3.Volume{}, err
		}
		rawServers, err := servers.ExtractServers(pager)
		if err != nil {
			return volumes3.Volume{}, err
		}
		vmCheck := false
		for _, vm := range rawServers {
			if vm.Name == ownerVMIID.NameId {
				ownerRawVM = vm
				vmCheck = true
				break
			}
		}
		if !vmCheck {
			return volumes3.Volume{}, errors.New("not found vm")
		}
	} else {
		server, err := servers.Get(computeClient, ownerVMIID.SystemId).Extract()
		if err != nil {
			return volumes3.Volume{}, err
		}
		ownerRawVM = *server
	}
	volumeAttachOpt := volumeattach.CreateOpts{
		VolumeID: disk.ID,
	}
	_, err = volumeattach.Create(computeClient, ownerRawVM.ID, volumeAttachOpt).Extract()
	if err != nil {
		return volumes3.Volume{}, err
	}
	newDisk, err := getRawDisk(irs.IID{NameId: disk.Name, SystemId: disk.ID}, volumeClient)
	if err != nil {
		return volumes3.Volume{}, err
	}
	curRetryCnt := 0
	maxRetryCnt := 20
	for {
		newDisk, err = getRawDisk(diskIID, volumeClient)
		if err != nil {
			return volumes3.Volume{}, err
		}
		switch strings.ToLower(newDisk.Status) {
		case "in-use":
			return newDisk, nil
		default:
			curRetryCnt++
			time.Sleep(1 * time.Second)
			if curRetryCnt > maxRetryCnt {
				return volumes3.Volume{}, errors.New(fmt.Sprintf("attaching failed. exceeded maximum retry count %d", maxRetryCnt))
			}
		}
	}
}

func AttachList(diskIIDList []irs.IID, ownerVMIID irs.IID, computeClient *gophercloud.ServiceClient, volumeClient *gophercloud.ServiceClient) (*servers.Server, error) {
	rawDataDiskList := make([]volumes3.Volume, len(diskIIDList))
	if len(diskIIDList) > 0 {
		for i, dataDiskIID := range diskIIDList {
			disk, err := getRawDisk(dataDiskIID, volumeClient)
			if err != nil {
				convertErr := errors.New(fmt.Sprintf("Failed to get DataDisk err = %s", err.Error()))
				return nil, convertErr
			}
			if disk.Status != "available" {
				return nil, errors.New(fmt.Sprintf("Attach is only available when available Status"))
			}
			rawDataDiskList[i] = disk
		}
	} else {
		return nil, nil
	}

	var ownerRawVM servers.Server
	if ownerVMIID.SystemId == "" {
		pager, err := servers.List(computeClient, nil).AllPages()
		if err != nil {
			return nil, err
		}
		rawServers, err := servers.ExtractServers(pager)
		if err != nil {
			return nil, err
		}
		vmCheck := false
		for _, vm := range rawServers {
			if vm.Name == ownerVMIID.NameId {
				ownerRawVM = vm
				vmCheck = true
				break
			}
		}
		if !vmCheck {
			return nil, errors.New("not found vm")
		}
	} else {
		server, err := servers.Get(computeClient, ownerVMIID.SystemId).Extract()
		if err != nil {
			return nil, err
		}
		ownerRawVM = *server
	}
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var globalErr error

	for _, rawDataDisk := range rawDataDiskList {
		wg.Add(1)
		dumpVolume := rawDataDisk
		go func() {
			defer wg.Done()
			_, err := attachWithCtx(ctx, dumpVolume, ownerRawVM.ID, computeClient, volumeClient)
			if err != nil {
				cancel()
				if globalErr == nil {
					globalErr = err
				}
			}
		}()
	}
	wg.Wait()
	if globalErr != nil {
		return nil, globalErr
	}
	server, err := servers.Get(computeClient, ownerRawVM.ID).Extract()
	if err != nil {
		return nil, err
	}
	return server, nil
}

// volumes3.Volume, error
type volumeWithError struct {
	Volume volumes3.Volume
	Err    error
}

func attachWithCtx(ctx context.Context, volume volumes3.Volume, ownerRawVMID string, computeClient *gophercloud.ServiceClient, volumeClient *gophercloud.ServiceClient) (volumes3.Volume, error) {
	done := make(chan volumeWithError)
	go func() {
		volumeAttachOpt := volumeattach.CreateOpts{
			VolumeID: volume.ID,
		}
		_, err := volumeattach.Create(computeClient, ownerRawVMID, volumeAttachOpt).Extract()
		if err != nil {
			done <- volumeWithError{
				volumes3.Volume{}, err,
			}
		}
		curRetryCnt := 0
		maxRetryCnt := 20
		for {
			newDisk, err := getRawDisk(irs.IID{SystemId: volume.ID}, volumeClient)
			if err != nil {
				done <- volumeWithError{
					volumes3.Volume{}, err,
				}
				break
			}
			switch strings.ToLower(newDisk.Status) {
			case "in-use":
				done <- volumeWithError{
					volumes3.Volume{}, err,
				}
				break
			default:
				curRetryCnt++
				time.Sleep(1 * time.Second)
				if curRetryCnt > maxRetryCnt {
					done <- volumeWithError{
						volumes3.Volume{}, errors.New(fmt.Sprintf("attaching failed. exceeded maximum retry count %d", maxRetryCnt)),
					}
					break
				}
			}
		}
	}()
	select {
	case volumeWithErrorDone := <-done:
		return volumeWithErrorDone.Volume, volumeWithErrorDone.Err
	case <-ctx.Done():
		return volumes3.Volume{}, nil
	}
}

func detachDisk(diskIID irs.IID, ownerVMIID irs.IID, computeClient *gophercloud.ServiceClient, volumeClient *gophercloud.ServiceClient) error {
	if diskIID.NameId == "" && diskIID.SystemId == "" {
		return errors.New("invalid Disk IID")
	}
	if ownerVMIID.NameId == "" && ownerVMIID.SystemId == "" {
		return errors.New("invalid ownerVM IID")
	}
	disk, err := getRawDisk(diskIID, volumeClient)
	if err != nil {
		return err
	}
	if len(disk.Attachments) == 0 {
		return errors.New("not exist Disk Attachment")
	}
	var ownerRawVM servers.Server
	if ownerVMIID.SystemId == "" {
		pager, err := servers.List(computeClient, nil).AllPages()
		if err != nil {
			return err
		}
		rawServers, err := servers.ExtractServers(pager)
		if err != nil {
			return err
		}
		vmCheck := false
		for _, vm := range rawServers {
			if vm.Name == ownerVMIID.NameId {
				ownerRawVM = vm
				vmCheck = true
				break
			}
		}
		if !vmCheck {
			return errors.New("not found vm")
		}
	} else {
		server, err := servers.Get(computeClient, ownerVMIID.SystemId).Extract()
		if err != nil {
			return err
		}
		ownerRawVM = *server
	}
	detachmentVolumeId := ""
	for _, attachmentedVolume := range ownerRawVM.AttachedVolumes {
		if attachmentedVolume.ID == disk.ID {
			detachmentVolumeId = attachmentedVolume.ID
		}
	}
	if detachmentVolumeId == "" {
		return errors.New("not exist Disk Attached VM")
	}
	//볼륨 아이디..
	err = volumeattach.Delete(computeClient, ownerRawVM.ID, detachmentVolumeId).ExtractErr()
	if err != nil {
		return err
	}
	curRetryCnt := 0
	maxRetryCnt := 20
	for {
		newDisk, err := getRawDisk(diskIID, volumeClient)
		if err != nil {
			return err
		}
		// status (“available”, “error”, “creating”, “reserved”, “deleting”, “in-use”, “attaching”, “detaching”, “error_deleting” or “maintenance”)
		switch strings.ToLower(newDisk.Status) {
		case "available":
			return nil
		case "detaching":
			{
				curRetryCnt++
				time.Sleep(1 * time.Second)
				if curRetryCnt > maxRetryCnt {
					return errors.New(fmt.Sprintf("detaching failed. exceeded maximum retry count %d", maxRetryCnt))
				}
			}
		default:
			return errors.New("detaching failed")
		}
	}
}

func changeDiskSize(diskIID irs.IID, diskSize string, volumeClient *gophercloud.ServiceClient) error {
	if diskIID.NameId == "" && diskIID.SystemId == "" {
		return errors.New("invalid Disk IID")
	}
	newSizeNum, err := strconv.Atoi(diskSize)
	if err != nil {
		return errors.New("invalid Disk IID")
	}
	disk, err := getRawDisk(diskIID, volumeClient)
	if err != nil {
		return err
	}
	if disk.Status != "available" {
		return errors.New(fmt.Sprintf("Resizing is only possible if it is mounted on a VM in the Available state"))
	}
	if disk.Size >= newSizeNum {
		return errors.New(fmt.Sprintf("New size must be greater than current size"))
	}
	if volumeClient == nil {
		return errors.New("VolumeClient not found")
	}
	changeSizeOpts := volumeactions.ExtendSizeOpts{
		NewSize: newSizeNum,
	}
	return volumeactions.ExtendSize(volumeClient, disk.ID, changeSizeOpts).ExtractErr()
}
