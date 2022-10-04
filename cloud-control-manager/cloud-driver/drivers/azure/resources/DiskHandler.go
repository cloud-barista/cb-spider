package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-03-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"strconv"
	"strings"
)

type AzureDiskHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	Ctx            context.Context
	VMClient       *compute.VirtualMachinesClient
	DiskClient     *compute.DisksClient
}

func (diskHandler *AzureDiskHandler) CreateDisk(DiskReqInfo irs.DiskInfo) (diskInfo irs.DiskInfo, createErr error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, "DISK", DiskReqInfo.IId.NameId, "CreateDisk()")
	start := call.Start()
	err := diskHandler.validationDiskReq(DiskReqInfo)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}
	diskType, err := GetDiskTypeInitType(DiskReqInfo.DiskType)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}
	diskSku := compute.DiskSku{
		Name: diskType,
	}
	creationData := compute.CreationData{
		CreateOption: compute.DiskCreateOptionEmpty,
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
	diskProperties := compute.DiskProperties{
		DiskSizeGB:   to.Int32Ptr(int32(diskSizeInt)),
		CreationData: &creationData,
	}
	diskCreateOpt := compute.Disk{DiskProperties: &diskProperties, Sku: &diskSku, Location: to.StringPtr(diskHandler.Region.Region)}
	result, err := diskHandler.DiskClient.CreateOrUpdate(diskHandler.Ctx, diskHandler.Region.ResourceGroup, DiskReqInfo.IId.NameId, diskCreateOpt)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}
	err = result.WaitForCompletionRef(diskHandler.Ctx, diskHandler.DiskClient.Client)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}
	convertedIId, err := ConvertDiskIID(DiskReqInfo.IId, diskHandler.CredentialInfo, diskHandler.Region)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}
	disk, err := GetRawDisk(convertedIId, diskHandler.Region.ResourceGroup, diskHandler.DiskClient, diskHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Disk. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.DiskInfo{}, getErr
	}
	info, err := setterDiskInfo(disk)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Disk. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.DiskInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return *info, nil
}
func (diskHandler *AzureDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, "DISK", "DISK", "ListDisk()")
	start := call.Start()
	diskList, err := diskHandler.DiskClient.ListByResourceGroup(diskHandler.Ctx, diskHandler.Region.ResourceGroup)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List Disk. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return []*irs.DiskInfo{}, getErr
	}
	var diskStatusList []*irs.DiskInfo
	for _, disk := range diskList.Values() {
		diskStatus, err := setterDiskInfo(disk)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List Disk. err = %s", err))
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
	hiscallInfo := GetCallLogScheme(diskHandler.Region, "DISK", diskIID.NameId, "GetDisk()")
	start := call.Start()
	convertedIId, err := ConvertDiskIID(diskIID, diskHandler.CredentialInfo, diskHandler.Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Disk. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.DiskInfo{}, getErr
	}
	disk, err := GetRawDisk(convertedIId, diskHandler.Region.ResourceGroup, diskHandler.DiskClient, diskHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Disk. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.DiskInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	info, err := setterDiskInfo(disk)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Disk. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.DiskInfo{}, getErr
	}
	return *info, nil
}
func (diskHandler *AzureDiskHandler) ChangeDiskSize(diskIID irs.IID, size string) (bool, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, "DISK", diskIID.NameId, "ChangeDiskSize()")
	start := call.Start()
	// Exist Disk
	convertedDiskIId, err := ConvertDiskIID(diskIID, diskHandler.CredentialInfo, diskHandler.Region)
	if err != nil {
		changeDiskSizeErr := errors.New(fmt.Sprintf("Failed to ChangeDiskSize. err = %s", err))
		cblogger.Error(changeDiskSizeErr.Error())
		LoggingError(hiscallInfo, changeDiskSizeErr)
		return false, changeDiskSizeErr
	}
	sizeChangeDisk, err := GetRawDisk(convertedDiskIId, diskHandler.Region.ResourceGroup, diskHandler.DiskClient, diskHandler.Ctx)
	if err != nil {
		changeDiskSizeErr := errors.New(fmt.Sprintf("Failed to ChangeDiskSize. err = %s", err))
		cblogger.Error(changeDiskSizeErr.Error())
		LoggingError(hiscallInfo, changeDiskSizeErr)
		return false, changeDiskSizeErr
	}
	// size Check
	newSize, err := checkSize(size, *sizeChangeDisk.DiskSizeGB)
	if err != nil {
		changeDiskSizeErr := errors.New(fmt.Sprintf("Failed to ChangeDiskSize. err = %s", err))
		cblogger.Error(changeDiskSizeErr.Error())
		LoggingError(hiscallInfo, changeDiskSizeErr)
		return false, changeDiskSizeErr
	}
	// disk Status Check
	err = checkChangeStatus(sizeChangeDisk)
	if err != nil {
		changeDiskSizeErr := errors.New(fmt.Sprintf("Failed to ChangeDiskSize. err = %s", err))
		cblogger.Error(changeDiskSizeErr.Error())
		LoggingError(hiscallInfo, changeDiskSizeErr)
		return false, changeDiskSizeErr
	}
	// Size Change
	diskUpdateOpt := compute.DiskUpdate{
		DiskUpdateProperties: &compute.DiskUpdateProperties{
			DiskSizeGB: to.Int32Ptr(newSize),
		},
	}
	result, err := diskHandler.DiskClient.Update(diskHandler.Ctx, diskHandler.Region.ResourceGroup, *sizeChangeDisk.Name, diskUpdateOpt)
	if err != nil {
		changeDiskSizeErr := errors.New(fmt.Sprintf("Failed to ChangeDiskSize. err = %s", err))
		cblogger.Error(changeDiskSizeErr.Error())
		LoggingError(hiscallInfo, changeDiskSizeErr)
		return false, changeDiskSizeErr
	}
	err = result.WaitForCompletionRef(diskHandler.Ctx, diskHandler.DiskClient.Client)
	if err != nil {
		changeDiskSizeErr := errors.New(fmt.Sprintf("Failed to ChangeDiskSize. err = %s", err))
		cblogger.Error(changeDiskSizeErr.Error())
		LoggingError(hiscallInfo, changeDiskSizeErr)
		return false, changeDiskSizeErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}
func (diskHandler *AzureDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, "DISK", diskIID.NameId, "DeleteDisk()")
	start := call.Start()
	convertedDiskIId, err := ConvertDiskIID(diskIID, diskHandler.CredentialInfo, diskHandler.Region)
	if err != nil {
		deleteDiskSizeErr := errors.New(fmt.Sprintf("Failed to DeleteDisk. err = %s", err))
		cblogger.Error(deleteDiskSizeErr.Error())
		LoggingError(hiscallInfo, deleteDiskSizeErr)
		return false, deleteDiskSizeErr
	}
	deleteDisk, err := GetRawDisk(convertedDiskIId, diskHandler.Region.ResourceGroup, diskHandler.DiskClient, diskHandler.Ctx)
	if err != nil {
		deleteDiskSizeErr := errors.New(fmt.Sprintf("Failed to DeleteDisk. err = %s", err))
		cblogger.Error(deleteDiskSizeErr.Error())
		LoggingError(hiscallInfo, deleteDiskSizeErr)
		return false, deleteDiskSizeErr
	}
	// Check status
	err = checkDeleteStatus(deleteDisk)
	if err != nil {
		deleteDiskSizeErr := errors.New(fmt.Sprintf("Failed to DeleteDisk. err = %s", err))
		cblogger.Error(deleteDiskSizeErr.Error())
		LoggingError(hiscallInfo, deleteDiskSizeErr)
		return false, deleteDiskSizeErr
	}
	result, err := diskHandler.DiskClient.Delete(diskHandler.Ctx, diskHandler.Region.ResourceGroup, convertedDiskIId.NameId)
	if err != nil {
		deleteDiskSizeErr := errors.New(fmt.Sprintf("Failed to DeleteDisk. err = %s", err.Error()))
		cblogger.Error(deleteDiskSizeErr.Error())
		LoggingError(hiscallInfo, deleteDiskSizeErr)
		return false, deleteDiskSizeErr
	}
	err = result.WaitForCompletionRef(diskHandler.Ctx, diskHandler.DiskClient.Client)
	if err != nil {
		deleteDiskSizeErr := errors.New(fmt.Sprintf("Failed to DeleteDisk. err = %s", err))
		cblogger.Error(deleteDiskSizeErr.Error())
		LoggingError(hiscallInfo, deleteDiskSizeErr)
		return false, deleteDiskSizeErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (diskHandler *AzureDiskHandler) AttachDisk(diskIID irs.IID, ownerVM irs.IID) (irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, "DISK", diskIID.NameId, "AttachDisk()")
	start := call.Start()

	disk, err := Attach(diskIID, ownerVM, diskHandler.CredentialInfo, diskHandler.Region, diskHandler.Ctx, diskHandler.VMClient, diskHandler.DiskClient)
	if err != nil {
		attachErr := errors.New(fmt.Sprintf("Failed to AttachDisk. err = %s", err))
		cblogger.Error(attachErr.Error())
		LoggingError(hiscallInfo, attachErr)
		return irs.DiskInfo{}, attachErr
	}
	info, err := setterDiskInfo(disk)
	if err != nil {
		attachErr := errors.New(fmt.Sprintf("Failed to AttachDisk. err = %s", err))
		cblogger.Error(attachErr.Error())
		LoggingError(hiscallInfo, attachErr)
		return irs.DiskInfo{}, attachErr
	}
	LoggingInfo(hiscallInfo, start)
	return *info, nil
}

func (diskHandler *AzureDiskHandler) DetachDisk(diskIID irs.IID, ownerVM irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, "DISK", diskIID.NameId, "DetachDisk()")
	start := call.Start()
	convertedDiskIId, err := ConvertDiskIID(diskIID, diskHandler.CredentialInfo, diskHandler.Region)
	if err != nil {
		dettachErr := errors.New(fmt.Sprintf("Failed to DetachDisk. err = %s", err))
		cblogger.Error(dettachErr.Error())
		LoggingError(hiscallInfo, dettachErr)
		return false, dettachErr
	}
	detachDisk, err := GetRawDisk(convertedDiskIId, diskHandler.Region.ResourceGroup, diskHandler.DiskClient, diskHandler.Ctx)
	if err != nil {
		dettachErr := errors.New(fmt.Sprintf("Failed to DetachDisk. err = %s", err))
		cblogger.Error(dettachErr.Error())
		LoggingError(hiscallInfo, dettachErr)
		return false, dettachErr
	}
	convertedVMIID, err := ConvertVMIID(ownerVM, diskHandler.CredentialInfo, diskHandler.Region)
	if err != nil {
		dettachErr := errors.New(fmt.Sprintf("Failed to DetachDisk. GetVM err = %s", err))
		cblogger.Error(dettachErr.Error())
		LoggingError(hiscallInfo, dettachErr)
		return false, dettachErr
	}
	vm, err := GetRawVM(convertedVMIID, diskHandler.Region.ResourceGroup, diskHandler.VMClient, diskHandler.Ctx)
	if err != nil {
		dettachErr := errors.New(fmt.Sprintf("Failed to DetachDisk. GetVM err = %s", err))
		cblogger.Error(dettachErr.Error())
		LoggingError(hiscallInfo, dettachErr)
		return false, dettachErr
	}
	vmManagedDiskList := *vm.StorageProfile.DataDisks
	if len(vmManagedDiskList) == 0 || vmManagedDiskList == nil {
		dettachErr := errors.New(fmt.Sprintf("Failed to DetachDisk. Not Eixst Disk : %s", diskIID.NameId))
		cblogger.Error(dettachErr.Error())
		LoggingError(hiscallInfo, dettachErr)
		return false, dettachErr
	}
	var newDiskList []compute.DataDisk
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
	if len(newDiskList) == 0 {
		newDiskList = make([]compute.DataDisk, 0)
	}
	vmOpts := compute.VirtualMachine{
		Location: to.StringPtr(diskHandler.Region.Region),
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			StorageProfile: &compute.StorageProfile{
				DataDisks: &newDiskList,
			},
		},
	}
	feature, err := diskHandler.VMClient.CreateOrUpdate(diskHandler.Ctx, diskHandler.Region.ResourceGroup, *vm.Name, vmOpts)
	if err != nil {
		dettachErr := errors.New(fmt.Sprintf("Failed to DetachDisk. err = %s", err))
		cblogger.Error(dettachErr.Error())
		LoggingError(hiscallInfo, dettachErr)
		return false, dettachErr
	}
	err = feature.WaitForCompletionRef(diskHandler.Ctx, diskHandler.VMClient.Client)
	if err != nil {
		dettachErr := errors.New(fmt.Sprintf("Failed to DetachDisk. err = %s", err))
		cblogger.Error(dettachErr.Error())
		LoggingError(hiscallInfo, dettachErr)
		return false, dettachErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func GetRawDisk(diskIID irs.IID, resourceGroup string, client *compute.DisksClient, ctx context.Context) (compute.Disk, error) {
	if diskIID.NameId == "" {
		diskList, err := client.ListByResourceGroup(ctx, resourceGroup)
		if err != nil {
			return compute.Disk{}, err
		}
		for _, disk := range diskList.Values() {
			if *disk.ID == diskIID.SystemId {
				return disk, nil
			}
		}
		notExistVpcErr := errors.New(fmt.Sprintf("The Disk id %s not found", diskIID.SystemId))
		return compute.Disk{}, notExistVpcErr
	} else {
		return client.Get(ctx, resourceGroup, diskIID.NameId)
	}
}

func setterDiskInfo(disk compute.Disk) (*irs.DiskInfo, error) {
	diskStatus := irs.DiskInfo{
		IId: irs.IID{
			NameId:   *disk.Name,
			SystemId: *disk.ID,
		},
	}
	if disk.Sku != nil {
		diskStatus.DiskType = GetDiskInfoType(disk.Sku.Name)
	}
	if disk.DiskProperties != nil {
		if disk.DiskProperties.DiskSizeGB != nil {
			diskStatus.DiskSize = strconv.Itoa(int(*disk.DiskProperties.DiskSizeGB))
		}
		//https://docs.microsoft.com/en-us/dotnet/api/microsoft.azure.management.compute.models.galleryprovisioningstate?view=azure-dotnet
		if disk.DiskProperties.ProvisioningState != nil {
			switch *disk.DiskProperties.ProvisioningState {
			case "Creating":
				diskStatus.Status = irs.DiskCreating
			case "Deleting":
				diskStatus.Status = irs.DiskDeleting
			case "Failed":
				diskStatus.Status = irs.DiskError
			case "Migrating", "Updating":
				diskStatus.Status = irs.DiskAttached
			case "Succeeded":
				if disk.DiskProperties.DiskState == compute.DiskStateUnattached {
					diskStatus.Status = irs.DiskAvailable
				} else {
					diskStatus.Status = irs.DiskAttached
				}
			}
		}
		diskStatus.CreatedTime = disk.DiskProperties.TimeCreated.Time
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
	// TODO KeyValueList
	return &diskStatus, nil
}

func ConvertDiskIID(diskIID irs.IID, credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo) (irs.IID, error) {
	if diskIID.NameId == "" && diskIID.SystemId == "" {
		return diskIID, errors.New(fmt.Sprintf("invalid IID"))
	}
	if diskIID.SystemId == "" {
		sysID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Compute/disks/%s", credentialInfo.SubscriptionId, regionInfo.ResourceGroup, diskIID.NameId)
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
	exist, err := CheckExistDisk(diskReq.IId, diskHandler.Region.ResourceGroup, diskHandler.DiskClient, diskHandler.Ctx)
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

func CheckExistDisk(diskIID irs.IID, resourceGroup string, client *compute.DisksClient, ctx context.Context) (bool, error) {
	diskList, err := client.ListByResourceGroup(ctx, resourceGroup)
	if err != nil {
		return false, err
	}
	for _, disk := range diskList.Values() {
		if diskIID.SystemId != "" && diskIID.SystemId == *disk.ID {
			return true, nil
		}
		if diskIID.NameId != "" && diskIID.NameId == *disk.Name {
			return true, nil
		}
	}
	return false, nil
}

func getMinDataDiskLun(dataDisks *[]compute.DataDisk) (int32, error) {
	if dataDisks == nil || len(*dataDisks) == 0 {
		return int32(0), nil
	}
	oldlunIntArray := make([]int32, len(*dataDisks))
	for i, disk := range *dataDisks {
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

func checkChangeStatus(disk compute.Disk) error {
	if disk.DiskProperties != nil && disk.DiskProperties.ProvisioningState != nil {
		if *disk.DiskProperties.ProvisioningState == "Succeeded" && (disk.DiskProperties.DiskState == compute.DiskStateUnattached || disk.DiskProperties.DiskState == compute.DiskStateReserved) {
			return nil
		}
	}
	return errors.New(fmt.Sprintf("Resizing is only possible if it is mounted on a VM in the deallocated state or if it is in the Unattached state."))
}

func checkDeleteStatus(disk compute.Disk) error {
	if disk.DiskProperties != nil && disk.DiskProperties.ProvisioningState != nil {
		if *disk.DiskProperties.ProvisioningState == "Succeeded" && (disk.DiskProperties.DiskState == compute.DiskStateUnattached || disk.DiskProperties.DiskState == compute.DiskStateReserved) {
			return nil
		}
	}
	return errors.New(fmt.Sprintf("Deleting is only possible if it is mounted on a VM in the deallocated state or if it is in the Unattached state."))
}

func CheckAttachStatus(disk compute.Disk) error {
	if disk.DiskProperties != nil && disk.DiskProperties.ProvisioningState != nil {
		if *disk.DiskProperties.ProvisioningState == "Succeeded" && disk.DiskProperties.DiskState == compute.DiskStateUnattached {
			return nil
		}
	}
	return errors.New(fmt.Sprintf("Attach is only available when UnAttached"))
}

func Attach(diskIID irs.IID, ownerVM irs.IID, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context, vmClient *compute.VirtualMachinesClient, diskClient *compute.DisksClient) (compute.Disk, error) {

	convertedDiskIId, err := ConvertDiskIID(diskIID, credentialInfo, region)
	if err != nil {
		return compute.Disk{}, err
	}
	disk, err := GetRawDisk(convertedDiskIId, region.ResourceGroup, diskClient, ctx)
	if err != nil {
		return compute.Disk{}, err
	}
	err = CheckAttachStatus(disk)
	if err != nil {
		return compute.Disk{}, err
	}
	convertedVMIId, err := ConvertVMIID(ownerVM, credentialInfo, region)
	if err != nil {
		return compute.Disk{}, errors.New(fmt.Sprintf("GetVM err = %s", err))
	}
	vm, err := GetRawVM(convertedVMIId, region.ResourceGroup, vmClient, ctx)
	if err != nil {
		return compute.Disk{}, errors.New(fmt.Sprintf("GetVM err = %s", err))
	}
	oldDataDisks := *vm.StorageProfile.DataDisks
	minLunNums, err := getMinDataDiskLun(&oldDataDisks)
	if err != nil {
		return compute.Disk{}, err
	}
	oldDataDisks = append(oldDataDisks, compute.DataDisk{
		Lun:          to.Int32Ptr(minLunNums),
		CreateOption: compute.DiskCreateOptionTypesAttach,
		ManagedDisk: &compute.ManagedDiskParameters{
			ID: to.StringPtr(*disk.ID),
		},
	})
	vmOpts := compute.VirtualMachine{
		Location: to.StringPtr(region.Region),
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			StorageProfile: &compute.StorageProfile{
				DataDisks: &oldDataDisks,
			},
		},
	}
	feature, err := vmClient.CreateOrUpdate(ctx, region.ResourceGroup, *vm.Name, vmOpts)
	if err != nil {
		return compute.Disk{}, err
	}
	err = feature.WaitForCompletionRef(ctx, vmClient.Client)
	if err != nil {
		return compute.Disk{}, err
	}
	disk, err = GetRawDisk(convertedDiskIId, region.ResourceGroup, diskClient, ctx)
	if err != nil {
		return compute.Disk{}, err
	}
	return disk, err
}

func AttachList(diskIIDList []irs.IID, ownerVM irs.IID, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context, vmClient *compute.VirtualMachinesClient, diskClient *compute.DisksClient) (compute.VirtualMachine, error) {
	rawDataDiskList := make([]compute.Disk, len(diskIIDList))
	// get RawDisk List
	if len(diskIIDList) > 0 {
		for i, dataDiskIID := range diskIIDList {
			convertedDiskIId, err := ConvertDiskIID(dataDiskIID, credentialInfo, region)
			if err != nil {
				convertErr := errors.New(fmt.Sprintf("Failed to get DataDisk err = %s", err.Error()))
				return compute.VirtualMachine{}, convertErr
			}
			disk, err := GetRawDisk(convertedDiskIId, region.ResourceGroup, diskClient, ctx)
			if err != nil {
				convertErr := errors.New(fmt.Sprintf("Failed to get DataDisk err = %s", err.Error()))
				return compute.VirtualMachine{}, convertErr
			}
			err = CheckAttachStatus(disk)
			if err != nil {
				return compute.VirtualMachine{}, err
			}
			rawDataDiskList[i] = disk
		}
	} else {
		return compute.VirtualMachine{}, nil
	}
	convertedVMIId, err := ConvertVMIID(ownerVM, credentialInfo, region)
	if err != nil {
		return compute.VirtualMachine{}, errors.New(fmt.Sprintf("Failed to get VM err = %s", err))
	}
	vm, err := GetRawVM(convertedVMIId, region.ResourceGroup, vmClient, ctx)
	if err != nil {
		return compute.VirtualMachine{}, errors.New(fmt.Sprintf("Failed to get VMerr = %s", err))
	}
	oldDataDisks := *vm.StorageProfile.DataDisks

	minLunNums, err := getMinDataDiskLun(&oldDataDisks)
	if err != nil {
		return compute.VirtualMachine{}, err
	}
	for i, rawDisk := range rawDataDiskList {
		oldDataDisks = append(oldDataDisks, compute.DataDisk{
			Lun:          to.Int32Ptr(minLunNums + int32(i)),
			CreateOption: compute.DiskCreateOptionTypesAttach,
			ManagedDisk: &compute.ManagedDiskParameters{
				ID: to.StringPtr(*rawDisk.ID),
			},
		})
	}
	vmOpts := compute.VirtualMachine{
		Location: to.StringPtr(region.Region),
		VirtualMachineProperties: &compute.VirtualMachineProperties{
			StorageProfile: &compute.StorageProfile{
				DataDisks: &oldDataDisks,
			},
		},
	}
	feature, err := vmClient.CreateOrUpdate(ctx, region.ResourceGroup, *vm.Name, vmOpts)
	if err != nil {
		return compute.VirtualMachine{}, err
	}
	err = feature.WaitForCompletionRef(ctx, vmClient.Client)
	if err != nil {
		return compute.VirtualMachine{}, err
	}
	vm, err = GetRawVM(convertedVMIId, region.ResourceGroup, vmClient, ctx)
	if err != nil {
		return compute.VirtualMachine{}, errors.New(fmt.Sprintf("Failed to get VMerr = %s", err))
	}
	return vm, nil
}
