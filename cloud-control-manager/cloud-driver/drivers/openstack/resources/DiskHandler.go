package resources

import (
	"encoding/json"
	"errors"
	"fmt"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/gophercloud/gophercloud"
	volumes2 "github.com/gophercloud/gophercloud/openstack/blockstorage/v2/volumes"
	volumes3 "github.com/gophercloud/gophercloud/openstack/blockstorage/v3/volumes"
)

type OpenstackDiskHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	ComputeClient  *gophercloud.ServiceClient
	Volume2Client  *gophercloud.ServiceClient
	Volume3Client  *gophercloud.ServiceClient
}

//------ Disk Management
func (diskHandler *OpenstackDiskHandler) CreateDisk(DiskReqInfo irs.DiskInfo) (irs.DiskInfo, error) {
	return irs.DiskInfo{}, nil
}
func (diskHandler *OpenstackDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	list2, err := diskHandler.getRawDiskList()
	list3, err := GetRawDiskListV3(diskHandler.Volume3Client)

	fmt.Println(list2, list3, err)
	fmt.Println("dd")
	return nil, nil
}
func (diskHandler *OpenstackDiskHandler) GetDisk(diskIID irs.IID) (irs.DiskInfo, error) {
	return irs.DiskInfo{}, nil
}
func (diskHandler *OpenstackDiskHandler) ChangeDiskSize(diskIID irs.IID, size string) (bool, error) {
	return false, nil
}
func (diskHandler *OpenstackDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {
	return false, nil
}

//------ Disk Attachment
func (diskHandler *OpenstackDiskHandler) AttachDisk(diskIID irs.IID, ownerVM irs.IID) (irs.DiskInfo, error) {
	return irs.DiskInfo{}, nil
}
func (diskHandler *OpenstackDiskHandler) DetachDisk(diskIID irs.IID, ownerVM irs.IID) (bool, error) {
	return false, nil
}

func volumes2Tovolumes3(vol2 volumes2.Volume) (volumes3.Volume, error) {
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

func GetRawDiskListV2(volume2Client *gophercloud.ServiceClient) ([]volumes3.Volume, error) {
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
		vol3, err := volumes2Tovolumes3(vol)
		if err != nil {
			return nil, err
		}
		newList[i] = vol3
	}
	return newList, nil
}

func GetRawDiskListV3(volume3Client *gophercloud.ServiceClient) ([]volumes3.Volume, error) {
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

func (diskHandler *OpenstackDiskHandler) getRawDiskList() ([]volumes3.Volume, error) {
	if diskHandler.Volume3Client != nil {
		return GetRawDiskListV3(diskHandler.Volume3Client)
	}
	if diskHandler.Volume2Client != nil {
		return GetRawDiskListV2(diskHandler.Volume2Client)
	}
	return nil, errors.New("VolumeClient not found")
}
