package resources

import (
	"context"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/gophercloud/gophercloud"
)

type OpenStackMyImageHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	Ctx            context.Context
	ComputeClient  *gophercloud.ServiceClient
	Volume2Client  *gophercloud.ServiceClient
	Volume3Client  *gophercloud.ServiceClient
}

//------ Snapshot to create a MyImage
func (myimageHandler *OpenStackMyImageHandler) SnapshotVM(snapshotReqInfo irs.MyImageInfo) (irs.MyImageInfo, error) {
	return irs.MyImageInfo{}, nil
}

//------ MyImage Management
func (myimageHandler *OpenStackMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
	return nil, nil
}
func (myimageHandler *OpenStackMyImageHandler) GetMyImage(myImageIID irs.IID) (irs.MyImageInfo, error) {
	return irs.MyImageInfo{}, nil
}
func (myimageHandler *OpenStackMyImageHandler) DeleteMyImage(myImageIID irs.IID) (bool, error) {
	return false, nil
}
