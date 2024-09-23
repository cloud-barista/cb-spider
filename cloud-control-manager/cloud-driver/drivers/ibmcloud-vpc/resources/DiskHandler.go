package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/globalsearchv2"
	"github.com/IBM/platform-services-go-sdk/globaltaggingv1"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"sort"
	"strconv"
	"time"
)

type IbmDiskHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	VpcService     *vpcv1.VpcV1
	Ctx            context.Context
	TaggingService *globaltaggingv1.GlobalTaggingV1
	SearchService  *globalsearchv2.GlobalSearchV2
}

func (diskHandler *IbmDiskHandler) CreateDisk(diskReqInfo irs.DiskInfo) (irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, call.DISK, diskReqInfo.IId.NameId, "CreateDisk()")
	start := call.Start()
	intCapacity, capacityAtoiErr := strconv.Atoi(diskReqInfo.DiskSize)
	var ptrCapacity *int64
	if capacityAtoiErr == nil {
		ptrCapacity = core.Int64Ptr(int64(intCapacity))
	}
	if ptrCapacity == nil || *ptrCapacity < 10 {
		// set default capacity as minimum
		ptrCapacity = core.Int64Ptr(50)
	}
	if &diskReqInfo.DiskType == nil || diskReqInfo.DiskType == "" || diskReqInfo.DiskType == "default" {
		// set default type as least performance
		diskReqInfo.DiskType = "general-purpose"
	}

	if diskReqInfo.Zone == "" {
		diskReqInfo.Zone = diskHandler.Region.Zone
	}
	if diskHandler.Region.TargetZone != "" {
		diskReqInfo.Zone = diskHandler.Region.TargetZone
	}

	createVolumeOptions := &vpcv1.CreateVolumeOptions{}
	createVolumeOptions.SetVolumePrototype(&vpcv1.VolumePrototype{
		Profile: &vpcv1.VolumeProfileIdentity{
			Name: &diskReqInfo.DiskType,
		},
		Zone: &vpcv1.ZoneIdentity{
			Name: &diskReqInfo.Zone,
		},
		Name:     &diskReqInfo.IId.NameId,
		Capacity: ptrCapacity,
	})

	createdDisk, _, err := diskHandler.VpcService.CreateVolume(createVolumeOptions)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Disk. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.DiskInfo{}, createErr
	}

	// Attach Tag
	if diskReqInfo.TagList != nil && len(diskReqInfo.TagList) > 0 {
		if createdDisk.CRN == nil {
			createErr := errors.New(fmt.Sprintf("Failed to get created Disk's CRN"))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.DiskInfo{}, createErr
		}

		for _, tag := range diskReqInfo.TagList {
			err = addTag(diskHandler.TaggingService, tag, *createdDisk.CRN)
			if err != nil {
				createErr := errors.New(fmt.Sprintf("Failed to Attach Tag to Disk err = %s", err.Error()))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
			}
		}
	}

	LoggingInfo(hiscallInfo, start)

	return *diskHandler.ToIRSDisk(createdDisk), nil
}

func (diskHandler *IbmDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, call.DISK, "DISK", "ListDisk()")

	start := call.Start()
	rawDiskList, listDiskErr := getRawDiskList(diskHandler.VpcService, diskHandler.Ctx)
	if listDiskErr != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List Disk. err = %s", listDiskErr.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	var irsDiskInfoList []*irs.DiskInfo
	for _, rawDisk := range *rawDiskList {
		irsDiskInfoList = append(irsDiskInfoList, diskHandler.ToIRSDisk(&rawDisk))
	}
	LoggingInfo(hiscallInfo, start)
	return irsDiskInfoList, nil
}

func (diskHandler *IbmDiskHandler) GetDisk(diskIID irs.IID) (irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, call.DISK, diskIID.SystemId, "GetDisk()")

	start := call.Start()
	rawDisk, getDiskErr := getRawDisk(diskHandler.VpcService, diskHandler.Ctx, diskIID)
	if getDiskErr != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Disk. err = %s", getDiskErr.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.DiskInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)

	return *diskHandler.ToIRSDisk(rawDisk), nil
}

func (diskHandler *IbmDiskHandler) ChangeDiskSize(diskIID irs.IID, size string) (bool, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, call.DISK, diskIID.SystemId, "ChangeDisk()")
	start := call.Start()
	targetSystemId, getDiskSystemIdErr := getDiskSystemId(diskHandler.VpcService, diskHandler.Ctx, diskIID)
	if getDiskSystemIdErr != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeDiskSize. err = %s", getDiskSystemIdErr.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return false, changeErr
	}

	updateMaps := make(map[string]interface{})
	intSize, err := strconv.Atoi(size)
	if err != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeDiskSize. err = %s", err.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return false, changeErr
	}
	updateMaps["capacity"] = core.Int64Ptr(int64(intSize))

	updateVolumeOptions := &vpcv1.UpdateVolumeOptions{
		ID:          core.StringPtr(targetSystemId),
		VolumePatch: updateMaps,
	}

	_, _, updateDiskErr := diskHandler.VpcService.UpdateVolumeWithContext(diskHandler.Ctx, updateVolumeOptions)
	if updateDiskErr != nil {
		changeErr := errors.New(fmt.Sprintf("Failed to ChangeDiskSize. err = %s", updateDiskErr.Error()))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return false, changeErr
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func (diskHandler *IbmDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, call.DISK, diskIID.SystemId, "DeleteDisk()")
	start := call.Start()
	targetSystemId, err := getDiskSystemId(diskHandler.VpcService, diskHandler.Ctx, diskIID)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Disk. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}

	deleteVolumeOptions := &vpcv1.DeleteVolumeOptions{
		ID: core.StringPtr(targetSystemId),
	}

	_, err = diskHandler.VpcService.DeleteVolumeWithContext(diskHandler.Ctx, deleteVolumeOptions)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Disk. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)

	deleteUnusedTags(diskHandler.TaggingService)

	return true, nil
}

func (diskHandler *IbmDiskHandler) AttachDisk(diskIID irs.IID, ownerVM irs.IID) (irs.DiskInfo, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, call.DISK, diskIID.SystemId, "AttachDisk()")
	start := call.Start()
	instance, getInstanceError := getRawInstance(ownerVM, diskHandler.VpcService, diskHandler.Ctx)
	if getInstanceError != nil {
		attachErr := errors.New(fmt.Sprintf("Failed to Attach Disk. err = %s", getInstanceError.Error()))
		cblogger.Error(attachErr.Error())
		LoggingError(hiscallInfo, attachErr)
		return irs.DiskInfo{}, attachErr
	}

	targetVolumeSystemId, getDiskSystemIdErr := getDiskSystemId(diskHandler.VpcService, diskHandler.Ctx, diskIID)
	if getDiskSystemIdErr != nil {
		attachErr := errors.New(fmt.Sprintf("Failed to Attach Disk. err = %s", getInstanceError.Error()))
		cblogger.Error(attachErr.Error())
		LoggingError(hiscallInfo, attachErr)
		return irs.DiskInfo{}, attachErr
	}

	createInstanceVolumeAttachmentOptions := &vpcv1.CreateInstanceVolumeAttachmentOptions{}
	createInstanceVolumeAttachmentOptions.SetInstanceID(*instance.ID)
	createInstanceVolumeAttachmentOptions.SetVolume(&vpcv1.VolumeAttachmentPrototypeVolumeVolumeIdentityVolumeIdentityByID{ID: &targetVolumeSystemId})
	createInstanceVolumeAttachmentOptions.SetDeleteVolumeOnInstanceDelete(false)

	_, _, attachDiskErr := diskHandler.VpcService.CreateInstanceVolumeAttachmentWithContext(diskHandler.Ctx, createInstanceVolumeAttachmentOptions)
	if attachDiskErr != nil {
		attachErr := errors.New(fmt.Sprintf("Failed to Attach Disk. err = %s", attachDiskErr.Error()))
		cblogger.Error(attachErr.Error())
		LoggingError(hiscallInfo, attachErr)
		return irs.DiskInfo{}, attachErr
	}

	attachedDisk, getDiskErr := diskHandler.GetDisk(diskIID)
	if getDiskErr != nil {
		attachErr := errors.New(fmt.Sprintf("Failed to Attach Disk. err = %s", getDiskErr.Error()))
		cblogger.Error(attachErr.Error())
		LoggingError(hiscallInfo, attachErr)
		return irs.DiskInfo{}, attachErr
	}
	LoggingInfo(hiscallInfo, start)
	return attachedDisk, nil
}

func (diskHandler *IbmDiskHandler) DetachDisk(diskIID irs.IID, ownerVM irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(diskHandler.Region, call.DISK, diskIID.SystemId, "DetachDisk()")
	start := call.Start()

	instance, getInstanceError := getRawInstance(ownerVM, diskHandler.VpcService, diskHandler.Ctx)
	if getInstanceError != nil {
		detachErr := errors.New(fmt.Sprintf("Failed to Detach Disk. err = %s", getInstanceError.Error()))
		cblogger.Error(detachErr.Error())
		LoggingError(hiscallInfo, detachErr)
		return false, detachErr
	}

	targetVolumeSystemId, getDiskSystemIdErr := getDiskSystemId(diskHandler.VpcService, diskHandler.Ctx, diskIID)
	if getDiskSystemIdErr != nil {
		detachErr := errors.New(fmt.Sprintf("Failed to Detach Disk. err = %s", getDiskSystemIdErr.Error()))
		cblogger.Error(detachErr.Error())
		LoggingError(hiscallInfo, detachErr)
		return false, detachErr
	}

	listInstanceVolumeAttachmentsOptions := &vpcv1.ListInstanceVolumeAttachmentsOptions{}
	listInstanceVolumeAttachmentsOptions.SetInstanceID(*instance.ID)
	volumeAttachments, _, listVolumeAttachmentsErr := diskHandler.VpcService.ListInstanceVolumeAttachmentsWithContext(diskHandler.Ctx, listInstanceVolumeAttachmentsOptions)
	if listVolumeAttachmentsErr != nil {
		detachErr := errors.New(fmt.Sprintf("Failed to Detach Disk. err = %s", listVolumeAttachmentsErr.Error()))
		cblogger.Error(detachErr.Error())
		LoggingError(hiscallInfo, detachErr)
		return false, detachErr
	}

	targetVolumeAttachmentId := ""
	for _, volumeAttachment := range (*volumeAttachments).VolumeAttachments {
		if *volumeAttachment.Volume.ID == targetVolumeSystemId {
			targetVolumeAttachmentId = *volumeAttachment.ID
			break
		}
	}
	if targetVolumeAttachmentId == "" {
		detachErr := errors.New(fmt.Sprintf("Failed to Detach Disk. err = Cannot find Volume Attachment"))
		cblogger.Error(detachErr.Error())
		LoggingError(hiscallInfo, detachErr)
		return false, detachErr
	}

	deleteInstanceVolumeAttachmentOptions := &vpcv1.DeleteInstanceVolumeAttachmentOptions{}
	deleteInstanceVolumeAttachmentOptions.SetID(targetVolumeAttachmentId)
	deleteInstanceVolumeAttachmentOptions.SetInstanceID(*instance.ID)

	_, detachDiskErr := diskHandler.VpcService.DeleteInstanceVolumeAttachmentWithContext(diskHandler.Ctx, deleteInstanceVolumeAttachmentOptions)
	if detachDiskErr != nil {
		detachErr := errors.New(fmt.Sprintf("Failed to Detach Disk. err = %s", detachDiskErr.Error()))
		cblogger.Error(detachErr.Error())
		LoggingError(hiscallInfo, detachErr)
		return false, detachErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (diskHandler *IbmDiskHandler) ToIRSDisk(disk *vpcv1.Volume) *irs.DiskInfo {
	var diskKeyValueList []irs.KeyValue
	if disk != nil {
		if disk.Iops != nil {
			strIops := strconv.Itoa(int(*disk.Iops))
			diskKeyValueList = append(diskKeyValueList, irs.KeyValue{Key: "Iops", Value: strIops})
		}
		if disk.ResourceGroup != nil {
			diskKeyValueList = append(diskKeyValueList, irs.KeyValue{Key: "ResourceGroup", Value: *disk.ResourceGroup.Name})
		}

		var diskStatus irs.DiskStatus
		var ownerVmIID irs.IID
		if len(disk.VolumeAttachments) > 0 {
			diskStatus = irs.DiskAttached
			ownerVmIID = irs.IID{
				NameId:   *disk.VolumeAttachments[0].Instance.Name,
				SystemId: *disk.VolumeAttachments[0].Instance.ID,
			}
		} else {
			diskStatus = getDiskStatus(*disk.Status)
			ownerVmIID = irs.IID{}
		}

		tagHandler := IbmTagHandler{
			Region:         diskHandler.Region,
			CredentialInfo: diskHandler.CredentialInfo,
			VpcService:     diskHandler.VpcService,
			Ctx:            diskHandler.Ctx,
			SearchService:  diskHandler.SearchService,
		}

		tags, err := tagHandler.ListTag(irs.DISK, irs.IID{SystemId: *disk.ID})
		if err != nil {
			cblogger.Warn("Failed to get tags of the Disk (" + *disk.Name + "). err = " + err.Error())
		}

		strCapacity := strconv.Itoa(int(*disk.Capacity))
		return &irs.DiskInfo{
			IId: irs.IID{
				SystemId: *disk.ID,
				NameId:   *disk.Name,
			},
			Zone:         *disk.Zone.Name,
			DiskType:     *disk.Profile.Name,
			DiskSize:     strCapacity,
			Status:       diskStatus,
			OwnerVM:      ownerVmIID,
			CreatedTime:  time.Time(*disk.CreatedAt).Local(),
			TagList:      tags,
			KeyValueList: diskKeyValueList,
		}
	}

	return &irs.DiskInfo{}
}

func getRawDiskList(vpcService *vpcv1.VpcV1, ctx context.Context) (*[]vpcv1.Volume, error) {
	listVolumeOptions := &vpcv1.ListVolumesOptions{}

	var entireRawDiskList []vpcv1.Volume
	for {
		curIter, _, listDiskErr := vpcService.ListVolumesWithContext(ctx, listVolumeOptions)
		if listDiskErr != nil {
			return nil, listDiskErr
		}

		if len(curIter.Volumes) > 0 {
			entireRawDiskList = append(entireRawDiskList, curIter.Volumes...)
		} else {
			break
		}

		nextIter := ""
		if curIter.Next != nil && curIter.Next.Href != nil {
			nextIter, _ = getNextHref(*curIter.Next.Href)
		}
		if nextIter != "" {
			listVolumeOptions = &vpcv1.ListVolumesOptions{
				Start: core.StringPtr(nextIter),
			}
		} else {
			break
		}
	}

	return &entireRawDiskList, nil
}

func getRawDisk(vpcService *vpcv1.VpcV1, ctx context.Context, diskIID irs.IID) (*vpcv1.Volume, error) {
	targetSystemId, getDiskSystemIdErr := getDiskSystemId(vpcService, ctx, diskIID)
	if getDiskSystemIdErr != nil {
		return nil, getDiskSystemIdErr
	}

	getVolumeOptions := &vpcv1.GetVolumeOptions{
		ID: core.StringPtr(targetSystemId),
	}

	rawDisk, _, getDiskErr := vpcService.GetVolumeWithContext(ctx, getVolumeOptions)
	if getDiskErr != nil {
		return nil, getDiskErr
	}

	return rawDisk, nil
}

func listRawAttachedDiskByVmIID(vpcService *vpcv1.VpcV1, ctx context.Context, ownerVMIID irs.IID) (*vpcv1.VolumeAttachmentCollection, error) {
	instance, getInstanceErr := getRawInstance(ownerVMIID, vpcService, ctx)
	if getInstanceErr != nil {
		return nil, getInstanceErr
	}

	listInstanceVolumeAttachmentsOptions := &vpcv1.ListInstanceVolumeAttachmentsOptions{}
	listInstanceVolumeAttachmentsOptions.SetInstanceID(*instance.ID)
	volumeAttachments, _, listVolumeAttachmentsErr := vpcService.ListInstanceVolumeAttachmentsWithContext(ctx, listInstanceVolumeAttachmentsOptions)
	if listVolumeAttachmentsErr != nil {
		return nil, errors.New(fmt.Sprintf("Failed to List Volume Attachments. err = %s", listVolumeAttachmentsErr))
	}

	temp := volumeAttachments.VolumeAttachments

	sort.Slice(temp, func(i, j int) bool {
		return temp[i].CreatedAt.String() < temp[j].CreatedAt.String()
	})
	volumeAttachments.VolumeAttachments = temp

	return volumeAttachments, nil
}

func getDiskStatus(status string) irs.DiskStatus {
	switch status {
	case "available":
		return irs.DiskAvailable
	case "failed", "unusable", "updating":
		return irs.DiskError
	case "pending":
		return irs.DiskCreating
	case "pending_deletion":
		return irs.DiskDeleting
	default:
		return irs.DiskError
	}
}

func getDiskSystemId(vpcService *vpcv1.VpcV1, ctx context.Context, iid irs.IID) (string, error) {
	if iid.NameId == "" && iid.SystemId == "" {
		return "", errors.New("Disk Name ID or System ID required.")
	}

	if iid.SystemId != "" {
		return iid.SystemId, nil
	}

	var targetSystemId string
	if iid.SystemId == "" {
		rawDiskList, getRawDiskListErr := getRawDiskList(vpcService, ctx)
		if getRawDiskListErr != nil {
			return "", errors.New(fmt.Sprintf("Failed to List Disk. err = %s", getRawDiskListErr))
		}
		for _, rawDisk := range *rawDiskList {
			if *rawDisk.Name == iid.NameId {
				targetSystemId = *rawDisk.ID
			}
		}
	} else {
		targetSystemId = iid.SystemId
	}

	return targetSystemId, nil
}

func (DiskHandler *IbmDiskHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	return nil, errors.New("Does not support ListIID() yet!!")
}
