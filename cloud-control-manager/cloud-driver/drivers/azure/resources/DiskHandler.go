package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"strconv"
	"strings"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AzureDiskHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	Ctx            context.Context
	VMClient       *armcompute.VirtualMachinesClient
	DiskClient     *armcompute.DisksClient
}

func (diskHandler *AzureDiskHandler) CreateDisk(DiskReqInfo irs.DiskInfo) (diskInfo irs.DiskInfo, createErr error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, call.DISK, DiskReqInfo.IId.NameId, "CreateDisk()")
	start := call.Start()
	err := diskHandler.validationDiskReq(DiskReqInfo)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}
	diskType, err := GetDiskTypeInitType(DiskReqInfo.DiskType)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}
	diskSKU := armcompute.DiskSKU{
		Name: &diskType,
	}
	// Create Tag
	tags := setTags(DiskReqInfo.TagList)

	creationData := armcompute.CreationData{
		CreateOption: (*armcompute.DiskCreateOption)(toStrPtr(string(armcompute.DiskCreateOptionEmpty))),
	}

	diskSizeInt, err := strconv.Atoi(DiskReqInfo.DiskSize)
	if DiskReqInfo.DiskSize == "" || strings.ToLower(DiskReqInfo.DiskSize) == "default" {
		diskSizeInt = 1024 // Azure console Init Value
		err = nil
	}
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create Disk. err = invalid Disk Size"))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}
	diskProperties := armcompute.DiskProperties{
		DiskSizeGB:   toInt32Ptr(diskSizeInt),
		CreationData: &creationData,
	}
	diskCreateOpt := armcompute.Disk{
		Properties: &diskProperties,
		SKU:        &diskSKU,
		Location:   &diskHandler.Region.Region,
		Tags:       tags,
	}
	// Setting zone if available
	if diskHandler.Region.Zone != "" {
		diskCreateOpt.Zones = []*string{
			&DiskReqInfo.Zone,
		}
	}
	poller, err := diskHandler.DiskClient.BeginCreateOrUpdate(diskHandler.Ctx, diskHandler.Region.Region, DiskReqInfo.IId.NameId, diskCreateOpt, nil)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}
	_, err = poller.PollUntilDone(diskHandler.Ctx, nil)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}
	convertedIId, err := ConvertDiskIID(DiskReqInfo.IId, diskHandler.CredentialInfo, diskHandler.Region)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}
	disk, err := GetRawDisk(convertedIId, diskHandler.Region.Region, diskHandler.DiskClient, diskHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Disk. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.DiskInfo{}, getErr
	}
	info, err := setterDiskInfo(&disk)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Disk. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.DiskInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return *info, nil
}
func (diskHandler *AzureDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, call.DISK, "DISK", "ListDisk()")
	start := call.Start()

	var diskList []*armcompute.Disk

	pager := diskHandler.DiskClient.NewListPager(nil)

	for pager.More() {
		page, err := pager.NextPage(diskHandler.Ctx)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List Disk. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return []*irs.DiskInfo{}, getErr
		}

		for _, disk := range page.Value {
			diskList = append(diskList, disk)
		}
	}

	var diskStatusList []*irs.DiskInfo
	for _, disk := range diskList {
		diskStatus, err := setterDiskInfo(disk)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List Disk. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return []*irs.DiskInfo{}, getErr
		}
		diskStatusList = append(diskStatusList, diskStatus)
	}
	LoggingInfo(hiscallInfo, start)
	return diskStatusList, nil
}
func (diskHandler *AzureDiskHandler) GetDisk(diskIID irs.IID) (irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, call.DISK, diskIID.NameId, "GetDisk()")
	start := call.Start()
	convertedIId, err := ConvertDiskIID(diskIID, diskHandler.CredentialInfo, diskHandler.Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Disk. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.DiskInfo{}, getErr
	}
	disk, err := GetRawDisk(convertedIId, diskHandler.Region.Region, diskHandler.DiskClient, diskHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Disk. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.DiskInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	info, err := setterDiskInfo(&disk)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Disk. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.DiskInfo{}, getErr
	}
	return *info, nil
}
func (diskHandler *AzureDiskHandler) ChangeDiskSize(diskIID irs.IID, size string) (bool, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, call.DISK, diskIID.NameId, "ChangeDiskSize()")
	start := call.Start()
	// Exist Disk
	convertedDiskIId, err := ConvertDiskIID(diskIID, diskHandler.CredentialInfo, diskHandler.Region)
	if err != nil {
		changeDiskSizeErr := errors.New(fmt.Sprintf("Failed to ChangeDiskSize. err = %s", err.Error()))
		cblogger.Error(changeDiskSizeErr.Error())
		LoggingError(hiscallInfo, changeDiskSizeErr)
		return false, changeDiskSizeErr
	}
	sizeChangeDisk, err := GetRawDisk(convertedDiskIId, diskHandler.Region.Region, diskHandler.DiskClient, diskHandler.Ctx)
	if err != nil {
		changeDiskSizeErr := errors.New(fmt.Sprintf("Failed to ChangeDiskSize. err = %s", err.Error()))
		cblogger.Error(changeDiskSizeErr.Error())
		LoggingError(hiscallInfo, changeDiskSizeErr)
		return false, changeDiskSizeErr
	}
	// size Check
	newSize, err := checkSize(size, *sizeChangeDisk.Properties.DiskSizeGB)
	if err != nil {
		changeDiskSizeErr := errors.New(fmt.Sprintf("Failed to ChangeDiskSize. err = %s", err.Error()))
		cblogger.Error(changeDiskSizeErr.Error())
		LoggingError(hiscallInfo, changeDiskSizeErr)
		return false, changeDiskSizeErr
	}
	// disk Status Check
	err = checkChangeStatus(sizeChangeDisk)
	if err != nil {
		changeDiskSizeErr := errors.New(fmt.Sprintf("Failed to ChangeDiskSize. err = %s", err.Error()))
		cblogger.Error(changeDiskSizeErr.Error())
		LoggingError(hiscallInfo, changeDiskSizeErr)
		return false, changeDiskSizeErr
	}
	// Size Change
	diskUpdateOpt := armcompute.DiskUpdate{
		Properties: &armcompute.DiskUpdateProperties{
			DiskSizeGB: &newSize,
		},
	}
	poller, err := diskHandler.DiskClient.BeginUpdate(diskHandler.Ctx, diskHandler.Region.Region, *sizeChangeDisk.Name, diskUpdateOpt, nil)
	if err != nil {
		changeDiskSizeErr := errors.New(fmt.Sprintf("Failed to ChangeDiskSize. err = %s", err.Error()))
		cblogger.Error(changeDiskSizeErr.Error())
		LoggingError(hiscallInfo, changeDiskSizeErr)
		return false, changeDiskSizeErr
	}
	_, err = poller.PollUntilDone(diskHandler.Ctx, nil)
	if err != nil {
		changeDiskSizeErr := errors.New(fmt.Sprintf("Failed to ChangeDiskSize. err = %s", err.Error()))
		cblogger.Error(changeDiskSizeErr.Error())
		LoggingError(hiscallInfo, changeDiskSizeErr)
		return false, changeDiskSizeErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}
func (diskHandler *AzureDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, call.DISK, diskIID.NameId, "DeleteDisk()")
	start := call.Start()
	convertedDiskIId, err := ConvertDiskIID(diskIID, diskHandler.CredentialInfo, diskHandler.Region)
	if err != nil {
		deleteDiskSizeErr := errors.New(fmt.Sprintf("Failed to DeleteDisk. err = %s", err.Error()))
		cblogger.Error(deleteDiskSizeErr.Error())
		LoggingError(hiscallInfo, deleteDiskSizeErr)
		return false, deleteDiskSizeErr
	}
	deleteDisk, err := GetRawDisk(convertedDiskIId, diskHandler.Region.Region, diskHandler.DiskClient, diskHandler.Ctx)
	if err != nil {
		deleteDiskSizeErr := errors.New(fmt.Sprintf("Failed to DeleteDisk. err = %s", err.Error()))
		cblogger.Error(deleteDiskSizeErr.Error())
		LoggingError(hiscallInfo, deleteDiskSizeErr)
		return false, deleteDiskSizeErr
	}
	// Check status
	err = checkDeleteStatus(deleteDisk)
	if err != nil {
		deleteDiskSizeErr := errors.New(fmt.Sprintf("Failed to DeleteDisk. err = %s", err.Error()))
		cblogger.Error(deleteDiskSizeErr.Error())
		LoggingError(hiscallInfo, deleteDiskSizeErr)
		return false, deleteDiskSizeErr
	}
	poller, err := diskHandler.DiskClient.BeginDelete(diskHandler.Ctx, diskHandler.Region.Region, convertedDiskIId.NameId, nil)
	if err != nil {
		deleteDiskSizeErr := errors.New(fmt.Sprintf("Failed to DeleteDisk. err = %s", err.Error()))
		cblogger.Error(deleteDiskSizeErr.Error())
		LoggingError(hiscallInfo, deleteDiskSizeErr)
		return false, deleteDiskSizeErr
	}
	_, err = poller.PollUntilDone(diskHandler.Ctx, nil)
	if err != nil {
		deleteDiskSizeErr := errors.New(fmt.Sprintf("Failed to DeleteDisk. err = %s", err.Error()))
		cblogger.Error(deleteDiskSizeErr.Error())
		LoggingError(hiscallInfo, deleteDiskSizeErr)
		return false, deleteDiskSizeErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (diskHandler *AzureDiskHandler) AttachDisk(diskIID irs.IID, ownerVM irs.IID) (irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, call.DISK, diskIID.NameId, "AttachDisk()")
	start := call.Start()

	disk, err := Attach(diskIID, ownerVM, diskHandler.CredentialInfo, diskHandler.Region, diskHandler.Ctx, diskHandler.VMClient, diskHandler.DiskClient)
	if err != nil {
		attachErr := errors.New(fmt.Sprintf("Failed to AttachDisk. err = %s", err.Error()))
		cblogger.Error(attachErr.Error())
		LoggingError(hiscallInfo, attachErr)
		return irs.DiskInfo{}, attachErr
	}
	info, err := setterDiskInfo(&disk)
	if err != nil {
		attachErr := errors.New(fmt.Sprintf("Failed to AttachDisk. err = %s", err.Error()))
		cblogger.Error(attachErr.Error())
		LoggingError(hiscallInfo, attachErr)
		return irs.DiskInfo{}, attachErr
	}
	LoggingInfo(hiscallInfo, start)
	return *info, nil
}

func (diskHandler *AzureDiskHandler) DetachDisk(diskIID irs.IID, ownerVM irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, call.DISK, diskIID.NameId, "DetachDisk()")
	start := call.Start()
	convertedDiskIId, err := ConvertDiskIID(diskIID, diskHandler.CredentialInfo, diskHandler.Region)
	if err != nil {
		dettachErr := errors.New(fmt.Sprintf("Failed to DetachDisk. err = %s", err.Error()))
		cblogger.Error(dettachErr.Error())
		LoggingError(hiscallInfo, dettachErr)
		return false, dettachErr
	}
	detachDisk, err := GetRawDisk(convertedDiskIId, diskHandler.Region.Region, diskHandler.DiskClient, diskHandler.Ctx)
	if err != nil {
		dettachErr := errors.New(fmt.Sprintf("Failed to DetachDisk. err = %s", err.Error()))
		cblogger.Error(dettachErr.Error())
		LoggingError(hiscallInfo, dettachErr)
		return false, dettachErr
	}
	convertedVMIID, err := ConvertVMIID(ownerVM, diskHandler.CredentialInfo, diskHandler.Region)
	if err != nil {
		dettachErr := errors.New(fmt.Sprintf("Failed to DetachDisk. GetVM err = %s", err.Error()))
		cblogger.Error(dettachErr.Error())
		LoggingError(hiscallInfo, dettachErr)
		return false, dettachErr
	}
	vm, err := GetRawVM(convertedVMIID, diskHandler.Region.Region, diskHandler.VMClient, diskHandler.Ctx)
	if err != nil {
		dettachErr := errors.New(fmt.Sprintf("Failed to DetachDisk. GetVM err = %s", err.Error()))
		cblogger.Error(dettachErr.Error())
		LoggingError(hiscallInfo, dettachErr)
		return false, dettachErr
	}
	vmManagedDiskList := vm.Properties.StorageProfile.DataDisks
	if len(vmManagedDiskList) == 0 || vmManagedDiskList == nil {
		dettachErr := errors.New(fmt.Sprintf("Failed to DetachDisk. Not Eixst Disk : %s", diskIID.NameId))
		cblogger.Error(dettachErr.Error())
		LoggingError(hiscallInfo, dettachErr)
		return false, dettachErr
	}
	var newDiskList []*armcompute.DataDisk
	diskExistCheck := false
	for _, vmManagedDisk := range vmManagedDiskList {
		if *vmManagedDisk.ManagedDisk.ID == *detachDisk.ID {
			diskExistCheck = true
		} else {
			newDiskList = append(newDiskList, vmManagedDisk)
		}
	}
	if !diskExistCheck {
		dettachErr := errors.New(fmt.Sprintf("Failed to DetachDisk. Not Eixst Disk : %s", diskIID.NameId))
		cblogger.Error(dettachErr.Error())
		LoggingError(hiscallInfo, dettachErr)
		return false, dettachErr
	}
	vmOpts := armcompute.VirtualMachine{
		Location: &diskHandler.Region.Region,
		Properties: &armcompute.VirtualMachineProperties{
			StorageProfile: &armcompute.StorageProfile{
				DataDisks: newDiskList,
			},
		},
	}
	poller, err := diskHandler.VMClient.BeginCreateOrUpdate(diskHandler.Ctx, diskHandler.Region.Region, *vm.Name, vmOpts, nil)
	if err != nil {
		dettachErr := errors.New(fmt.Sprintf("Failed to DetachDisk. err = %s", err.Error()))
		cblogger.Error(dettachErr.Error())
		LoggingError(hiscallInfo, dettachErr)
		return false, dettachErr
	}
	_, err = poller.PollUntilDone(diskHandler.Ctx, nil)
	if err != nil {
		dettachErr := errors.New(fmt.Sprintf("Failed to DetachDisk. err = %s", err.Error()))
		cblogger.Error(dettachErr.Error())
		LoggingError(hiscallInfo, dettachErr)
		return false, dettachErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func GetRawDisk(diskIID irs.IID, resourceGroup string, client *armcompute.DisksClient, ctx context.Context) (armcompute.Disk, error) {
	if diskIID.NameId == "" {
		var diskList []*armcompute.Disk

		pager := client.NewListPager(nil)

		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return armcompute.Disk{}, err
			}

			for _, disk := range page.Value {
				diskList = append(diskList, disk)
			}
		}

		for _, disk := range diskList {
			if *disk.ID == diskIID.SystemId {
				return *disk, nil
			}
		}
		notExistVpcErr := errors.New(fmt.Sprintf("The Disk id %s not found", diskIID.SystemId))
		return armcompute.Disk{}, notExistVpcErr
	} else {
		resp, err := client.Get(ctx, resourceGroup, diskIID.NameId, nil)
		if err != nil {
			return armcompute.Disk{}, err
		}

		return resp.Disk, nil
	}
}

func setterDiskInfo(disk *armcompute.Disk) (*irs.DiskInfo, error) {
	diskStatus := irs.DiskInfo{
		IId: irs.IID{
			NameId:   *disk.Name,
			SystemId: *disk.ID,
		},
	}
	if disk.SKU != nil {
		diskStatus.DiskType = GetDiskInfoType(*disk.SKU.Name)
	}
	if disk.Properties != nil {
		if disk.Properties.DiskSizeGB != nil {
			diskStatus.DiskSize = strconv.Itoa(int(*disk.Properties.DiskSizeGB))
		}
		//https://docs.microsoft.com/en-us/dotnet/api/microsoft.azure.management.compute.models.galleryprovisioningstate?view=azure-dotnet
		if disk.Properties.ProvisioningState != nil {
			switch *disk.Properties.ProvisioningState {
			case "Creating":
				diskStatus.Status = irs.DiskCreating
			case "Deleting":
				diskStatus.Status = irs.DiskDeleting
			case "Failed":
				diskStatus.Status = irs.DiskError
			case "Migrating", "Updating":
				diskStatus.Status = irs.DiskAttached
			case "Succeeded":
				if *disk.Properties.DiskState == armcompute.DiskStateUnattached {
					diskStatus.Status = irs.DiskAvailable
				} else {
					diskStatus.Status = irs.DiskAttached
				}
			}
		}
		diskStatus.CreatedTime = *disk.Properties.TimeCreated
	}
	if disk.ManagedBy != nil && *disk.ManagedBy != "" {
		vmName, err := GetVMNameById(*disk.ManagedBy)
		if err == nil {
			diskStatus.OwnerVM = irs.IID{
				NameId:   vmName,
				SystemId: *disk.ManagedBy,
			}
		}
	}
	if len(disk.Zones) > 0 {
		diskStatus.Zone = *disk.Zones[0]
	}
	if disk.Tags != nil {
		diskStatus.TagList = setTagList(disk.Tags)
	}
	// TODO KeyValueList
	return &diskStatus, nil
}

func ConvertDiskIID(diskIID irs.IID, credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo) (irs.IID, error) {
	if diskIID.NameId == "" && diskIID.SystemId == "" {
		return diskIID, errors.New(fmt.Sprintf("invalid IID"))
	}
	if diskIID.SystemId == "" {
		sysID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/disks/%s", credentialInfo.SubscriptionId, regionInfo.Region, diskIID.NameId)
		return irs.IID{NameId: diskIID.NameId, SystemId: sysID}, nil
	} else {
		slist := strings.Split(diskIID.SystemId, "/")
		if len(slist) == 0 {
			return diskIID, errors.New(fmt.Sprintf("Invalid IID"))
		}
		s := slist[len(slist)-1]
		return irs.IID{NameId: s, SystemId: diskIID.SystemId}, nil
	}
}

func (diskHandler *AzureDiskHandler) validationDiskReq(diskReq irs.DiskInfo) error {
	if diskReq.IId.NameId == "" {
		return errors.New("invalid DiskReqInfo NameId")
	}
	exist, err := CheckExistDisk(diskReq.IId, diskHandler.Region.Region, diskHandler.DiskClient, diskHandler.Ctx)
	if err != nil {
		return errors.New("failed Check disk Name Exist")
	}
	if exist {
		return errors.New("invalid DiskReqInfo NameId, Already exist")
	}
	//if diskReq.DiskType == "" {
	//	return errors.New("invalid DiskReqInfo DiskType")
	//}
	//if diskReq.DiskSize == "" {
	//	return errors.New("invalid DiskReqInfo DiskSize")
	//}
	// default?
	//_, err = strconv.Atoi(diskReq.DiskSize)
	//if err != nil {
	//	return errors.New(fmt.Sprintf("invalid DiskReqInfo DiskSize, %s", err.Error()))
	//}
	//if diskReq.DiskType == "" {
	//	return errors.New("invalid DiskReqInfo DiskSize")
	//}
	//disktypeErr := errors.New("invalid DiskReqInfo DiskType")
	//if diskReq.DiskType == PremiumSSD || diskReq.DiskType == StandardSSD || diskReq.DiskType == StandardHHD || strings.ToLower(diskReq.DiskType) == "default" {
	//	disktypeErr = nil
	//}
	//if disktypeErr != nil {
	//	return disktypeErr
	//}
	return nil
}

func CheckExistDisk(diskIID irs.IID, resourceGroup string, client *armcompute.DisksClient, ctx context.Context) (bool, error) {
	var diskList []*armcompute.Disk

	pager := client.NewListPager(nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return false, err
		}

		for _, disk := range page.Value {
			diskList = append(diskList, disk)
		}
	}

	for _, disk := range diskList {
		if diskIID.SystemId != "" && diskIID.SystemId == *disk.ID {
			return true, nil
		}
		if diskIID.NameId != "" && diskIID.NameId == *disk.Name {
			return true, nil
		}
	}
	return false, nil
}

func getMinDataDiskLun(dataDisks []*armcompute.DataDisk) (int32, error) {
	if len(dataDisks) == 0 {
		return int32(0), nil
	}
	oldlunIntArray := make([]int32, len(dataDisks))
	for i, disk := range dataDisks {
		if disk == nil {
			continue
		}
		oldlunIntArray[i] = *disk.Lun
	}
	for lunNum := 0; lunNum < 64; lunNum++ {
		check := false
		for _, oldLun := range oldlunIntArray {
			if int32(lunNum) == oldLun {
				check = true
				break
			}
		}
		// find min
		if !check {
			return int32(lunNum), nil
		}
	}
	return int32(-1), errors.New("not found dataDisk Lun Number")
}

func checkSize(newSize string, oldSize int32) (int32, error) {
	newSizeNum, err := strconv.Atoi(newSize)
	if err != nil {
		return -1, errors.New("invalid Disk Size")
	}
	if int32(newSizeNum) < oldSize {
		return -1, errors.New(fmt.Sprintf("invalid Disk Size, Reducing disk size is not supported in Azure to prevent data loss."))
	}
	return int32(newSizeNum), nil
}

func checkChangeStatus(disk armcompute.Disk) error {
	if disk.Properties != nil && disk.Properties.ProvisioningState != nil && disk.Properties.DiskState != nil {
		if *disk.Properties.ProvisioningState == "Succeeded" && (*disk.Properties.DiskState == armcompute.DiskStateUnattached || *disk.Properties.DiskState == armcompute.DiskStateReserved) {
			return nil
		}
	}
	return errors.New(fmt.Sprintf("Resizing is only possible if it is mounted on a VM in the deallocated state or if it is in the Unattached state."))
}

func checkDeleteStatus(disk armcompute.Disk) error {
	if disk.Properties != nil && disk.Properties.ProvisioningState != nil && disk.Properties.DiskState != nil {
		if *disk.Properties.ProvisioningState == "Succeeded" && (*disk.Properties.DiskState == armcompute.DiskStateUnattached || *disk.Properties.DiskState == armcompute.DiskStateReserved) {
			return nil
		}
	}
	return errors.New(fmt.Sprintf("Deleting is only possible if it is mounted on a VM in the deallocated state or if it is in the Unattached state."))
}

func CheckAttachStatus(disk *armcompute.Disk) error {
	if disk.Properties != nil && disk.Properties.ProvisioningState != nil && disk.Properties.DiskState != nil {
		if *disk.Properties.ProvisioningState == "Succeeded" && *disk.Properties.DiskState == armcompute.DiskStateUnattached {
			return nil
		}
	}
	return errors.New(fmt.Sprintf("Attach is only available when UnAttached"))
}

func Attach(diskIID irs.IID, ownerVM irs.IID, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context, vmClient *armcompute.VirtualMachinesClient, diskClient *armcompute.DisksClient) (armcompute.Disk, error) {
	convertedDiskIId, err := ConvertDiskIID(diskIID, credentialInfo, region)
	if err != nil {
		return armcompute.Disk{}, err
	}
	disk, err := GetRawDisk(convertedDiskIId, region.Region, diskClient, ctx)
	if err != nil {
		return armcompute.Disk{}, err
	}
	err = CheckAttachStatus(&disk)
	if err != nil {
		return armcompute.Disk{}, err
	}
	convertedVMIId, err := ConvertVMIID(ownerVM, credentialInfo, region)
	if err != nil {
		return armcompute.Disk{}, errors.New(fmt.Sprintf("GetVM err = %s", err))
	}
	vm, err := GetRawVM(convertedVMIId, region.Region, vmClient, ctx)
	if err != nil {
		return armcompute.Disk{}, errors.New(fmt.Sprintf("GetVM err = %s", err))
	}
	oldDataDisks := vm.Properties.StorageProfile.DataDisks
	minLunNums, err := getMinDataDiskLun(oldDataDisks)
	if err != nil {
		return armcompute.Disk{}, err
	}
	oldDataDisks = append(oldDataDisks, &armcompute.DataDisk{
		Lun:          &minLunNums,
		CreateOption: (*armcompute.DiskCreateOptionTypes)(toStrPtr(string(armcompute.DiskCreateOptionTypesAttach))),
		ManagedDisk: &armcompute.ManagedDiskParameters{
			ID: disk.ID,
		},
	})
	vmOpts := armcompute.VirtualMachine{
		Location: toStrPtr(region.Region),
		Properties: &armcompute.VirtualMachineProperties{
			StorageProfile: &armcompute.StorageProfile{
				DataDisks: oldDataDisks,
			},
		},
	}
	poller, err := vmClient.BeginCreateOrUpdate(ctx, region.Region, *vm.Name, vmOpts, nil)
	if err != nil {
		return armcompute.Disk{}, err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return armcompute.Disk{}, err
	}
	disk, err = GetRawDisk(convertedDiskIId, region.Region, diskClient, ctx)
	if err != nil {
		return armcompute.Disk{}, err
	}
	return disk, err
}

func AttachList(diskIIDList []irs.IID, ownerVM irs.IID, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context, vmClient *armcompute.VirtualMachinesClient, diskClient *armcompute.DisksClient) (armcompute.VirtualMachine, error) {
	rawDataDiskList := make([]armcompute.Disk, len(diskIIDList))
	// get RawDisk List
	if len(diskIIDList) > 0 {
		for i, dataDiskIID := range diskIIDList {
			convertedDiskIId, err := ConvertDiskIID(dataDiskIID, credentialInfo, region)
			if err != nil {
				convertErr := errors.New(fmt.Sprintf("Failed to get DataDisk err = %s", err.Error()))
				return armcompute.VirtualMachine{}, convertErr
			}
			disk, err := GetRawDisk(convertedDiskIId, region.Region, diskClient, ctx)
			if err != nil {
				convertErr := errors.New(fmt.Sprintf("Failed to get DataDisk err = %s", err.Error()))
				return armcompute.VirtualMachine{}, convertErr
			}
			err = CheckAttachStatus(&disk)
			if err != nil {
				return armcompute.VirtualMachine{}, err
			}
			rawDataDiskList[i] = disk
		}
	} else {
		return armcompute.VirtualMachine{}, nil
	}
	convertedVMIId, err := ConvertVMIID(ownerVM, credentialInfo, region)
	if err != nil {
		return armcompute.VirtualMachine{}, errors.New(fmt.Sprintf("Failed to get VM err = %s", err))
	}
	vm, err := GetRawVM(convertedVMIId, region.Region, vmClient, ctx)
	if err != nil {
		return armcompute.VirtualMachine{}, errors.New(fmt.Sprintf("Failed to get VMerr = %s", err))
	}
	oldDataDisks := vm.Properties.StorageProfile.DataDisks
	minLunNums, err := getMinDataDiskLun(oldDataDisks)
	if err != nil {
		return armcompute.VirtualMachine{}, err
	}
	for i, rawDisk := range rawDataDiskList {
		createOption := armcompute.DiskCreateOptionTypesAttach
		oldDataDisks = append(oldDataDisks, &armcompute.DataDisk{
			Lun:          toInt32Ptr(int(minLunNums) + i),
			CreateOption: &createOption,
			ManagedDisk: &armcompute.ManagedDiskParameters{
				ID: rawDisk.ID,
			},
		})
	}
	vmOpts := armcompute.VirtualMachine{
		Location: toStrPtr(region.Region),
		Properties: &armcompute.VirtualMachineProperties{
			StorageProfile: &armcompute.StorageProfile{
				DataDisks: oldDataDisks,
			},
		},
	}
	poller, err := vmClient.BeginCreateOrUpdate(ctx, region.Region, *vm.Name, vmOpts, nil)
	if err != nil {
		return armcompute.VirtualMachine{}, err
	}
	_, err = poller.PollUntilDone(ctx, nil)
	if err != nil {
		return armcompute.VirtualMachine{}, err
	}
	vm, err = GetRawVM(convertedVMIId, region.Region, vmClient, ctx)
	if err != nil {
		return armcompute.VirtualMachine{}, errors.New(fmt.Sprintf("Failed to get VMerr = %s", err))
	}
	return vm, nil
}
