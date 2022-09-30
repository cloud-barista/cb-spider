package resources

import (
	"errors"
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/disk"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/server"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"strconv"
	"strings"
)

type ClouditDiskHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func (diskHandler *ClouditDiskHandler) CreateDisk(DiskReqInfo irs.DiskInfo) (irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "DISK", DiskReqInfo.IId.NameId, "CreateDisk()")

	//가상디스크 이름 중복 체크
	exist, err := diskHandler.getExistsDiskName(DiskReqInfo.IId.NameId)
	if exist {
		createErr := errors.New(fmt.Sprintf("Failed to Create Disk. err = %s already exist", DiskReqInfo.IId.NameId))
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.DiskInfo{}, createErr
		}
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}

	// API Call prepare
	diskHandler.Client.TokenID = diskHandler.CredentialInfo.AuthToken
	authHeader := diskHandler.Client.AuthenticatedHeaders()
	var intSize = 50
	if convResult, atoiErr := strconv.Atoi(DiskReqInfo.DiskSize); atoiErr == nil {
		intSize = convResult
	}
	clusterNameId := diskHandler.CredentialInfo.ClusterId
	clusterSystemId := ""
	if clusterNameId == "" {
		return irs.DiskInfo{}, errors.New("Failed to Create Disk. err = ClusterId is required.")
	} else if clusterNameId == "default" {
		return irs.DiskInfo{}, errors.New("Failed to Create Disk. err = Cloudit does not supports \"default\" cluster.")
	}

	requestURL := diskHandler.Client.CreateRequestBaseURL(client.ACE, "clusters")
	cblogger.Info(requestURL)

	var result client.Result
	if _, result.Err = diskHandler.Client.Get(requestURL, &result.Body, &client.RequestOpts{
		MoreHeaders: authHeader,
	}); result.Err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}

	var clusterList []struct {
		Id   string
		Name string
	}
	if err := result.ExtractInto(&clusterList); err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}
	for _, cluster := range clusterList {
		if cluster.Name == clusterNameId {
			clusterSystemId = cluster.Id
		}
	}

	reqInfo := disk.DiskReqInfo{
		Name: DiskReqInfo.IId.NameId,
		// ToDo: to static value?
		ClusterId: clusterSystemId,
		Size:      intSize,
	}
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
		JSONBody:    reqInfo,
	}

	// 디스크 생성
	start := call.Start()
	createdDisk, err := disk.Create(diskHandler.Client, &requestOpts)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}
	LoggingInfo(hiscallInfo, start)

	return createdDisk.ToIRSDisk(diskHandler.Client), nil
}

func (diskHandler *ClouditDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "DISK", "DISK", "ListDisk()")

	// API call prepare
	diskHandler.Client.TokenID = diskHandler.CredentialInfo.AuthToken
	authHeader := diskHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	start := call.Start()
	diskList, err := disk.List(diskHandler.Client, &requestOpts)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get DiskList. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	LoggingInfo(hiscallInfo, start)

	diskInfoList := make([]*irs.DiskInfo, len(*diskList))
	for i, disk := range *diskList {
		irsDisk := disk.ToIRSDisk(diskHandler.Client)
		diskInfoList[i] = &irsDisk
	}

	return diskInfoList, nil
}

func (diskHandler *ClouditDiskHandler) GetDisk(diskIID irs.IID) (irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "DISK", diskIID.NameId, "GetDisk()")

	start := call.Start()
	disk, err := diskHandler.getRawDisk(diskIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Disk. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.DiskInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)

	return disk.ToIRSDisk(diskHandler.Client), nil
}

func (diskHandler *ClouditDiskHandler) ChangeDiskSize(diskIID irs.IID, size string) (bool, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "DISK", diskIID.NameId, "ChangeDiskSize()")

	// get target Disk
	targetDiskSystemId := ""
	if diskIID.SystemId == "" {
		diskInfo, err := diskHandler.getRawDisk(diskIID)
		if err != nil {
			return false, errors.New(fmt.Sprintf("Failed to Update Disk. err = %s", err))
		}
		diskStatus := diskInfo.ToIRSDisk(diskHandler.Client).Status
		if diskStatus != irs.DiskAvailable {
			return false, errors.New(fmt.Sprintf("Failed to Update Disk. err = cannot change disk size in %s state", diskStatus))
		}
		targetDiskSystemId = diskInfo.ID
	} else {
		targetDiskSystemId = diskIID.SystemId
	}

	// set update info
	intSize, _ := strconv.Atoi(size)
	diskUpdateInfo := disk.DiskInfo{
		ID:   targetDiskSystemId,
		Size: intSize,
	}

	start := call.Start()
	result, err := diskHandler.updateDisk(diskUpdateInfo)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Update Disk. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}
	LoggingInfo(hiscallInfo, start)

	return result, nil
}

func (diskHandler *ClouditDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "DISK", diskIID.NameId, "DeleteDisk()")

	// get target Disk
	targetDiskSystemId := ""
	if diskIID.SystemId == "" {
		diskInfo, _ := diskHandler.getRawDisk(diskIID)
		diskStatus := diskInfo.ToIRSDisk(diskHandler.Client).Status
		if !(diskStatus == irs.DiskAvailable || diskStatus == irs.DiskError) {
			return false, errors.New(fmt.Sprintf("Failed to Delete Disk. err = cannot delete disk in %s state", diskStatus))
		}
		targetDiskSystemId = diskInfo.ID
	} else {
		targetDiskSystemId = diskIID.SystemId
	}

	start := call.Start()
	result, err := diskHandler.deleteDisk(irs.IID{SystemId: targetDiskSystemId})
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Delete Disk. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}
	LoggingInfo(hiscallInfo, start)

	return result, nil
}

func (diskHandler *ClouditDiskHandler) AttachDisk(diskIID irs.IID, ownerVM irs.IID) (irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "DISK", diskIID.NameId, "AttachDisk()")

	// get target Disk
	targetDiskSystemId := ""
	if diskIID.SystemId == "" {
		diskInfo, err := diskHandler.getRawDisk(diskIID)
		if err != nil {
			return irs.DiskInfo{}, errors.New(fmt.Sprintf("Cannot Attach Disk. err = %s", err))
		}
		diskStatus := diskInfo.ToIRSDisk(diskHandler.Client).Status
		if diskStatus != irs.DiskAvailable {
			return irs.DiskInfo{}, errors.New(fmt.Sprintf("Cannot Attach Disk. err = cannot attach disk in %s state", diskStatus))
		}
		targetDiskSystemId = diskInfo.ID
	} else {
		targetDiskSystemId = diskIID.SystemId
	}

	// get target VM
	targetVMSystemId := ""
	if ownerVM.SystemId == "" {
		diskHandler.Client.TokenID = diskHandler.CredentialInfo.AuthToken
		authHeader := diskHandler.Client.AuthenticatedHeaders()
		requestOpts := client.RequestOpts{
			MoreHeaders: authHeader,
		}
		vmList, err := server.List(diskHandler.Client, &requestOpts)
		if err != nil {
			return irs.DiskInfo{}, errors.New(fmt.Sprintf("Failed to Attach Disk. err = %s", err))
		}
		for _, vm := range *vmList {
			if strings.EqualFold(vm.Name, ownerVM.NameId) {
				targetVMSystemId = vm.ID
			}
		}
	} else {
		targetVMSystemId = ownerVM.SystemId
	}

	start := call.Start()
	err := diskHandler.attachDisk(irs.IID{SystemId: targetDiskSystemId}, irs.IID{SystemId: targetVMSystemId})
	if err != nil {
		return irs.DiskInfo{}, errors.New(fmt.Sprintf("Failed to attach disk. err = %s", err))
	}
	LoggingInfo(hiscallInfo, start)

	diskInfo, err := diskHandler.GetDisk(irs.IID{SystemId: targetDiskSystemId})
	if err != nil {
		return irs.DiskInfo{}, errors.New(fmt.Sprintf("Failed to get attached disk information. err = %s", err))
	}

	return diskInfo, nil
}

func (diskHandler *ClouditDiskHandler) DetachDisk(diskIID irs.IID, ownerVM irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, "DISK", diskIID.NameId, "DetachDisk()")

	// get target Disk
	targetDiskSystemId := ""
	if diskIID.SystemId == "" {
		diskInfo, err := diskHandler.getRawDisk(diskIID)
		if err != nil {
			return false, errors.New(fmt.Sprintf("Failed to Detach Disk. err = %s", err))
		}
		diskStatus := diskInfo.ToIRSDisk(diskHandler.Client).Status
		if diskStatus != irs.DiskAttached {
			return false, errors.New(fmt.Sprintf("Cannot Detach Disk. err = cannot detach disk in %s state", diskStatus))
		}
		targetDiskSystemId = diskInfo.ID
	} else {
		targetDiskSystemId = diskIID.SystemId
	}

	// get target VM
	targetVMSystemId := ""
	if ownerVM.SystemId == "" {
		diskHandler.Client.TokenID = diskHandler.CredentialInfo.AuthToken
		authHeader := diskHandler.Client.AuthenticatedHeaders()
		requestOpts := client.RequestOpts{
			MoreHeaders: authHeader,
		}
		vmList, err := server.List(diskHandler.Client, &requestOpts)
		if err != nil {
			return false, errors.New(fmt.Sprintf("Failed to Detach Disk. err = %s", err))
		}
		for _, vm := range *vmList {
			if strings.EqualFold(vm.Name, ownerVM.NameId) {
				targetVMSystemId = vm.ID
			}
		}
	} else {
		targetVMSystemId = ownerVM.SystemId
	}

	start := call.Start()
	err := diskHandler.detachDisk(irs.IID{SystemId: targetDiskSystemId}, irs.IID{SystemId: targetVMSystemId})
	if err != nil {
		return false, errors.New(fmt.Sprintf("Failed to Detach Disk. err = %s", err))
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func (diskHandler *ClouditDiskHandler) getExistsDiskName(name string) (bool, error) {
	if name == "" {
		return true, errors.New("invalid disk name")
	}
	diskHandler.Client.TokenID = diskHandler.CredentialInfo.AuthToken
	authHeader := diskHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	diskList, err := disk.List(diskHandler.Client, &requestOpts)
	if err != nil {
		return true, err
	}
	for _, rawDisk := range *diskList {
		if strings.EqualFold(name, rawDisk.Name) {
			return true, nil
		}
	}

	return false, nil
}

func (diskHandler *ClouditDiskHandler) getRawDisk(diskIId irs.IID) (*disk.DiskInfo, error) {
	if diskIId.SystemId == "" && diskIId.NameId == "" {
		return nil, errors.New("invalid IID")
	}
	diskHandler.Client.TokenID = diskHandler.CredentialInfo.AuthToken
	authHeader := diskHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	if diskIId.SystemId == "" {
		diskList, err := disk.List(diskHandler.Client, &requestOpts)
		if err != nil {
			return nil, err
		}
		for _, rawDisk := range *diskList {
			if strings.EqualFold(diskIId.NameId, rawDisk.Name) {
				return &rawDisk, nil
			}
		}
	} else {
		return disk.Get(diskHandler.Client, diskIId.SystemId, &requestOpts)
	}

	return nil, errors.New("cannot find disk")
}

func (diskHandler *ClouditDiskHandler) updateDisk(diskInfo disk.DiskInfo) (bool, error) {
	if diskInfo.ID == "" {
		return false, errors.New("target disk is not specified")
	}

	diskReqInfo := diskInfo.ToUpdateDiskReqInfo()
	if diskReqInfo == nil {
		return false, errors.New("nothings to update")
	}

	// API call prepare
	diskHandler.Client.TokenID = diskHandler.CredentialInfo.AuthToken
	authHeader := diskHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
		JSONBody:    diskReqInfo,
	}

	_, err := disk.Update(diskHandler.Client, diskInfo.ID, &requestOpts)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (diskHandler *ClouditDiskHandler) deleteDisk(diskIId irs.IID) (bool, error) {
	if diskIId.SystemId == "" {
		return false, errors.New("target disk is not specified")
	}

	// API call prepare
	diskHandler.Client.TokenID = diskHandler.CredentialInfo.AuthToken
	authHeader := diskHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	err := disk.Delete(diskHandler.Client, diskIId.SystemId, &requestOpts)
	if err != nil {
		return false, err
	}

	return true, nil
}

func (diskHandler *ClouditDiskHandler) attachDisk(diskIID irs.IID, ownerVmIID irs.IID) error {
	if diskIID.SystemId == "" {
		return errors.New("cannot specify disk")
	}

	if ownerVmIID.SystemId == "" {
		return errors.New("cannot specify owner VM")
	}

	// API call prepare
	diskHandler.Client.TokenID = diskHandler.CredentialInfo.AuthToken
	authHeader := diskHandler.Client.AuthenticatedHeaders()
	attachDiskReqInfo := disk.DiskReqInfo{
		ID:   diskIID.SystemId,
		Mode: "w",
	}
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
		JSONBody:    attachDiskReqInfo,
	}

	err := server.AttachVolume(diskHandler.Client, ownerVmIID.SystemId, &requestOpts)
	if err != nil {
		return err
	}

	return nil
}

func (diskHandler *ClouditDiskHandler) detachDisk(diskIID irs.IID, ownerVmIID irs.IID) error {
	if diskIID.SystemId == "" {
		return errors.New("cannot specify disk")
	}

	if ownerVmIID.SystemId == "" {
		return errors.New("cannot specify owner VM")
	}

	// API call prepare
	diskHandler.Client.TokenID = diskHandler.CredentialInfo.AuthToken
	authHeader := diskHandler.Client.AuthenticatedHeaders()
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	err := server.DetachVolume(diskHandler.Client, ownerVmIID.SystemId, diskIID.SystemId, &requestOpts)
	if err != nil {
		return err
	}

	return nil
}
