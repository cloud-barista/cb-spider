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
	volumes2snapshots "github.com/gophercloud/gophercloud/openstack/blockstorage/v2/snapshots"
	volumes3snapshots "github.com/gophercloud/gophercloud/openstack/blockstorage/v3/snapshots"
	volumes3 "github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"strings"
	"time"
)

type OpenStackMyImageHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	Ctx            context.Context
	ComputeClient  *gophercloud.ServiceClient
	VolumeClient   *gophercloud.ServiceClient
}

func (myImageHandler *OpenStackMyImageHandler) SnapshotVM(snapshotReqInfo irs.MyImageInfo) (irs.MyImageInfo, error) {
	hiscallInfo := GetCallLogScheme(myImageHandler.CredentialInfo.IdentityEndpoint, "MyImage", "MyImage", "SnapshotVM()")
	start := call.Start()

	image, err := myImageHandler.snapshot(snapshotReqInfo)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to SnapshotVM. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.MyImageInfo{}, getErr
	}
	info, err := setterMyImageInfo(image, myImageHandler.ComputeClient)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to SnapshotVM. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.MyImageInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (myImageHandler *OpenStackMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
	hiscallInfo := GetCallLogScheme(myImageHandler.CredentialInfo.IdentityEndpoint, "MyImage", "MyImage", "ListMyImage()")
	start := call.Start()
	list, err := getRawSnapshotList(myImageHandler.ComputeClient)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List MyImage. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	infolist := make([]*irs.MyImageInfo, len(list))
	for i, image := range list {
		info, err := setterMyImageInfo(image, myImageHandler.ComputeClient)
		if err != nil {
			if err != nil {
				getErr := errors.New(fmt.Sprintf("Failed to List MyImage. err = %s", err))
				cblogger.Error(getErr.Error())
				LoggingError(hiscallInfo, getErr)
				return nil, getErr
			}
		}
		infolist[i] = &info
	}
	LoggingInfo(hiscallInfo, start)
	return infolist, nil
}
func (myImageHandler *OpenStackMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {
	hiscallInfo := GetCallLogScheme(myImageHandler.CredentialInfo.IdentityEndpoint, "MyImage", myImageIID.NameId, "GetMyImage()")
	start := call.Start()
	image, err := getRawSnapshot(myImageIID, myImageHandler.ComputeClient)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get MyImage. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.MyImageInfo{}, getErr
	}
	info, err := setterMyImageInfo(image, myImageHandler.ComputeClient)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get MyImage. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.MyImageInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (myImageHandler *OpenStackMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(myImageHandler.CredentialInfo.IdentityEndpoint, "MyImage", myImageIID.NameId, "GetMyImage()")
	start := call.Start()
	_, err := deleteSnapshot(myImageIID, myImageHandler.ComputeClient, myImageHandler.VolumeClient)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get MyImage. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

type BlockDeviceMapping struct {
	SnapshotId string `json:"snapshot_id"`
	VolumeSize int    `json:"volume_size"`
	SourceType string `json:"source_type"`
	DeviceName string `json:"device_name"`
}

type ImageMetaData struct {
	BaseImageRef        string               `json:"base_image_ref"`
	RootDeviceName      string               `json:"root_device_name"`
	ImageType           string               `json:"image_type"`
	ImageLocation       string               `json:"image_location"`
	InstanceUuid        string               `json:"instance_uuid"`
	BlockDeviceMapping  []BlockDeviceMapping `json:"block_device_mapping"`
	DeviceType          string               `json:"device_type"`
	DestinationType     string               `json:"destination_type"`
	DeleteOnTermination bool                 `json:"delete_on_termination"`
	SourceVMID          string               `json:"source_vm_id"`   // spider Tag
	SourceVMName        string               `json:"source_vm_name"` // spider Tag
	DataVolumesString   string               `json:"data_volumes_raw"`
}

func convertImageMetaData(image images.Image) (ImageMetaData, error) {
	if image.Metadata != nil {
		var meta ImageMetaData
		bytes, err := json.Marshal(image.Metadata)
		if err != nil {
			return ImageMetaData{}, errors.New(fmt.Sprintf("Failed convert Metadata err = %s", err.Error()))
		}
		err = json.Unmarshal(bytes, &meta)

		if err != nil {
			return ImageMetaData{}, errors.New(fmt.Sprintf("Failed convert Metadata err = %s", err.Error()))
		}
		//if meta.DataVolumesString != "" {
		//	datas := []byte(meta.DataVolumesString)
		//	err = json.Unmarshal(datas, &meta.DataVolumes)
		//}
		return meta, nil
	}
	return ImageMetaData{}, nil
}

func CheckSnapshot(image images.Image) (bool, error) {
	metadata, err := convertImageMetaData(image)
	if err != nil {
		return false, err
	}
	if metadata.ImageType == "snapshot" {
		return true, nil
	}
	if metadata.BlockDeviceMapping != nil {
		for _, blockDeviceMapping := range metadata.BlockDeviceMapping {
			if blockDeviceMapping.SourceType == "snapshot" {
				return true, nil
			}
		}
	}
	return false, nil
}

func getRawSnapshotList(computeClient *gophercloud.ServiceClient) ([]images.Image, error) {
	pager, err := images.ListDetail(computeClient, images.ListOpts{}).AllPages()
	if err != nil {
		return nil, err
	}
	list, err := images.ExtractImages(pager)
	if err != nil {
		return nil, err
	}
	var imageList []images.Image

	for _, image := range list {
		snapshotFlag, err := CheckSnapshot(image)
		if err != nil {
			return nil, err
		}
		if snapshotFlag {
			imageList = append(imageList, image)
		}
	}
	if imageList == nil {
		emptyList := make([]images.Image, 0)
		return emptyList, nil
	}
	return imageList, err
}

func getRawSnapshot(snapshotIID irs.IID, computeClient *gophercloud.ServiceClient) (images.Image, error) {
	if snapshotIID.NameId == "" && snapshotIID.SystemId == "" {
		return images.Image{}, errors.New("invalid IID")
	}
	if snapshotIID.SystemId != "" {
		image, err := images.Get(computeClient, snapshotIID.SystemId).Extract()
		if err != nil {
			return images.Image{}, err
		}
		snapshotFlag, err := CheckSnapshot(*image)
		if !snapshotFlag {
			if err != nil {
				return images.Image{}, errors.New("not found MyImage")
			}
		}
		return *image, nil
	}
	imageList, err := getRawSnapshotList(computeClient)
	if err != nil {
		return images.Image{}, err
	}
	for _, image := range imageList {
		if image.Name == snapshotIID.NameId {
			return image, nil
		}
	}
	return images.Image{}, errors.New("not found MyImage")
}

func setterMyImageInfo(image images.Image, computeClient *gophercloud.ServiceClient) (irs.MyImageInfo, error) {
	snapshotFlag, err := CheckSnapshot(image)
	if err != nil {
		return irs.MyImageInfo{}, err
	}
	if !snapshotFlag {
		return irs.MyImageInfo{}, errors.New("not snapshot MyImage")
	}
	info := irs.MyImageInfo{
		IId: irs.IID{
			NameId:   image.Name,
			SystemId: image.ID,
		},
		Status: irs.MyImageUnavailable,
	}
	if strings.ToLower(image.Status) == "active" {
		info.Status = irs.MyImageAvailable
	}
	meta, err := convertImageMetaData(image)
	if err == nil {
		if meta.SourceVMID == "" || meta.SourceVMName == "" {
			vmId := meta.InstanceUuid
			if meta.InstanceUuid != "" {
				vm, err := servers.Get(computeClient, vmId).Extract()
				if err == nil {
					info.SourceVM = irs.IID{NameId: vm.Name, SystemId: vm.ID}
				}
			}
		} else {
			info.SourceVM = irs.IID{NameId: meta.SourceVMName, SystemId: meta.SourceVMID}
		}
	}
	createdTime, err := time.Parse("2006-01-02T15:04:05Z", image.Created)
	if err == nil {
		info.CreatedTime = createdTime
	}
	return info, nil
}

type DataDiskInfo struct {
	Volume       volumes3.Volume
	AttachDevice string
}

func deleteSnapshot(snapshotIID irs.IID, computeClient *gophercloud.ServiceClient, volumeClient *gophercloud.ServiceClient) (bool, error) {
	snapshot, err := getRawSnapshot(snapshotIID, computeClient)
	if err != nil {
		return false, err
	}
	snapshotmeta, err := convertImageMetaData(snapshot)
	if err != nil {
		return false, err
	}

	if snapshotmeta.BlockDeviceMapping != nil && len(snapshotmeta.BlockDeviceMapping) > 0 {
		if volumeClient == nil {
			return false, errors.New("the image has Metadata about the volume. However, this Openstack does not have a Cinder module installed. Please check the cinder module installation")
		}
		for _, volumeSnapshot := range snapshotmeta.BlockDeviceMapping {
			if volumeClient.Type == VolumeV3 {
				err = volumes3snapshots.Delete(volumeClient, volumeSnapshot.SnapshotId).ExtractErr()
			}
			if volumeClient.Type == VolumeV2 {
				err = volumes2snapshots.Delete(volumeClient, volumeSnapshot.SnapshotId).ExtractErr()
			}
		}
	}
	if err != nil {
		return false, err
	}
	err = images.Delete(computeClient, snapshot.ID).ExtractErr()
	if err != nil {
		return false, err
	}
	return true, nil
}

func (myImageHandler *OpenStackMyImageHandler) snapshot(snapshotReqInfo irs.MyImageInfo) (images.Image, error) {
	if snapshotReqInfo.SourceVM.NameId == "" && snapshotReqInfo.SourceVM.SystemId == "" {
		return images.Image{}, errors.New("invalid SourceVM IID")
	}
	var ownerRawVM servers.Server
	if snapshotReqInfo.SourceVM.SystemId == "" {
		pager, err := servers.List(myImageHandler.ComputeClient, nil).AllPages()
		if err != nil {
			return images.Image{}, err
		}
		rawServers, err := servers.ExtractServers(pager)
		if err != nil {
			return images.Image{}, err
		}
		vmCheck := false
		for _, vm := range rawServers {
			if vm.Name == snapshotReqInfo.SourceVM.NameId {
				ownerRawVM = vm
				vmCheck = true
				break
			}
		}
		if !vmCheck {
			return images.Image{}, errors.New("not found vm")
		}
	} else {
		server, err := servers.Get(myImageHandler.ComputeClient, snapshotReqInfo.SourceVM.SystemId).Extract()
		if err != nil {
			return images.Image{}, err
		}
		ownerRawVM = *server
	}
	if ownerRawVM.ID == "" {
		return images.Image{}, errors.New("not found vm")
	}
	imagepOpt := servers.CreateImageOpts{
		Name: snapshotReqInfo.IId.NameId,
		Metadata: map[string]string{
			"source_vm_id":   ownerRawVM.ID,
			"source_vm_name": ownerRawVM.Name,
		},
	}
	imageId, err := servers.CreateImage(myImageHandler.ComputeClient, ownerRawVM.ID, imagepOpt).ExtractImageID()
	if err != nil {
		return images.Image{}, err
	}
	image, err := images.Get(myImageHandler.ComputeClient, imageId).Extract()
	if err != nil {
		return images.Image{}, err
	}
	return *image, nil
}

func (myImageHandler *OpenStackMyImageHandler) CheckWindowsImage(myImageIID irs.IID) (bool, error) {
	image, err := getRawSnapshot(myImageIID, myImageHandler.ComputeClient)
	if err != nil {
		return false, err
	}
	value, exist := image.Metadata["os_type"]
	if !exist {
		return false, nil
	}
	if value == "windows" {
		return true, nil
	}
	return false, nil
}
