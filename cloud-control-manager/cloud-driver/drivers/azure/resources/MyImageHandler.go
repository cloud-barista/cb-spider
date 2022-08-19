package resources

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-03-01/compute"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AzureMyImageHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	Ctx            context.Context
	VMClient       *compute.VirtualMachinesClient
	ImageClient    *compute.ImagesClient
}

//------ Snapshot to create a MyImage
func (myimageHandler *AzureMyImageHandler) SnapshotVM(snapshotReqInfo irs.MyImageInfo) (irs.MyImageInfo, error) {
	return irs.MyImageInfo{}, nil
}

//------ MyImage Management
func (myimageHandler *AzureMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
	return nil, nil
}
func (myimageHandler *AzureMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {
	return irs.MyImageInfo{}, nil
}
func (myimageHandler *AzureMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
	return false, nil
}
