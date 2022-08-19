package resources

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-03-01/compute"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AzureDiskHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	Ctx            context.Context
	VMClient       *compute.VirtualMachinesClient
	DiskClient     *compute.DisksClient
}

//------ Disk Management
func (vmHandler *AzureDiskHandler) CreateDisk(DiskReqInfo irs.DiskInfo) (irs.DiskInfo, error) {
	return irs.DiskInfo{}, nil
}
func (vmHandler *AzureDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	return nil, nil
}
func (vmHandler *AzureDiskHandler) GetDisk(diskIID irs.IID) (irs.DiskInfo, error) {
	return irs.DiskInfo{}, nil
}
func (vmHandler *AzureDiskHandler) ChangeDiskSize(diskIID irs.IID, size string) (bool, error) {
	return false, nil
}
func (vmHandler *AzureDiskHandler) DeleteDisk(diskIID irs.IID) (bool, error) {
	return false, nil
}

//------ Disk Attachment
func (vmHandler *AzureDiskHandler) AttachDisk(diskIID irs.IID, ownerVM irs.IID) (irs.DiskInfo, error) {
	return irs.DiskInfo{}, nil
}
func (vmHandler *AzureDiskHandler) DetachDisk(diskIID irs.IID, ownerVM irs.IID) (bool, error) {
	return false, nil
}
