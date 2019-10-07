package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"strings"
)

type AzureImageHandler struct {
	Region idrv.RegionInfo
	Ctx    context.Context
	Client *compute.ImagesClient
}

// @TODO: ImageInfo 리소스 프로퍼티 정의 필요
type ImageInfo struct {
	ID            string
	Name          string
	Location      string
	OsType        string
	OsDiskSize    int32
	OsState       string
	ManagedDiskId string
}

func (imageInfo *ImageInfo) setter(image compute.Image) *ImageInfo {
	imageInfo.ID = *image.ID
	imageInfo.Name = *image.Name
	imageInfo.Location = *image.Location
	imageInfo.OsType = fmt.Sprint(image.ImageProperties.StorageProfile.OsDisk.OsType)
	imageInfo.OsDiskSize = *image.StorageProfile.OsDisk.DiskSizeGB
	imageInfo.OsState = fmt.Sprint(image.StorageProfile.OsDisk.OsState)

	if image.StorageProfile.OsDisk.ManagedDisk != nil {
		imageInfo.ManagedDiskId = *image.StorageProfile.OsDisk.ManagedDisk.ID
	}

	return imageInfo
}

func (imageHandler *AzureImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	imageIdArr := strings.Split(imageReqInfo.Id, ":")

	// @TODO: PublicIP 생성 요청 파라미터 정의 필요
	type ImageReqInfo struct {
		OSType string
		DiskId string
	}
	reqInfo := ImageReqInfo{
		//BlobUrl: "https://md-ds50xp550wh2.blob.core.windows.net/kt0lhznvgx2h/abcd?sv=2017-04-17&sr=b&si=b9674241-fb8e-4cb2-89c7-614d336dc3a7&sig=uvbqvAZQITSpxas%2BWosG%2FGOf6e%2BIBmWNxlUmvARnxiM%3D",
		OSType: "Linux",
		//DiskId: "/subscriptions/cb592624-b77b-4a8f-bb13-0e5a48cae40f/resourceGroups/INNO-PLATFORM1-RSRC-GRUP/providers/Microsoft.Compute/disks/inno-test-vm_OsDisk_1_61bf675b990f4aa381d7ee3d766974aa",
		// edited by powerkim for test, 2019.08.13
		DiskId: "/subscriptions/f1548292-2be3-4acd-84a4-6df079160846/resourceGroups/CB-RESOURCE-GROUP/providers/Microsoft.Compute/disks/vm_name_OsDisk_1_2d63d9cd754c4094b1b1fb6a98c36b71",
	}

	// Check Image Exists
	image, err := imageHandler.Client.Get(imageHandler.Ctx, imageIdArr[0], imageIdArr[1], "")
	if image.ID != nil {
		errMsg := fmt.Sprintf("Image with name %s already exist", imageIdArr[1])
		createErr := errors.New(errMsg)
		return irs.ImageInfo{}, createErr
	}

	createOpts := compute.Image{
		ImageProperties: &compute.ImageProperties{
			StorageProfile: &compute.ImageStorageProfile{
				OsDisk: &compute.ImageOSDisk{
					ManagedDisk: &compute.SubResource{
						ID: to.StringPtr(reqInfo.DiskId),
					},
					OsType: compute.OperatingSystemTypes(reqInfo.OSType),
					//BlobURI: to.StringPtr(reqInfo.BlobUrl),
				},
			},
		},
		Location: &imageHandler.Region.Region,
	}

	future, err := imageHandler.Client.CreateOrUpdate(imageHandler.Ctx, imageIdArr[0], imageIdArr[1], createOpts)
	if err != nil {
		return irs.ImageInfo{}, err
	}
	err = future.WaitForCompletionRef(imageHandler.Ctx, imageHandler.Client.Client)
	if err != nil {
		return irs.ImageInfo{}, err
	}

	return irs.ImageInfo{}, nil
}

func (imageHandler *AzureImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	//resultList, err := imageHandler.Client.List(imageHandler.Ctx)
	resultList, err := imageHandler.Client.ListByResourceGroup(imageHandler.Ctx, imageHandler.Region.ResourceGroup)
	if err != nil {
		cblogger.Error(err)
	}

	var imageList []*ImageInfo
	for _, image := range resultList.Values() {

		imageInfo := new(ImageInfo).setter(image)
		imageList = append(imageList, imageInfo)
	}

	spew.Dump(imageList)
	return nil, nil
}

func (imageHandler *AzureImageHandler) GetImage(imageID string) (irs.ImageInfo, error) {
	imageIdArr := strings.Split(imageID, ":")

	image, err := imageHandler.Client.Get(imageHandler.Ctx, imageIdArr[0], imageIdArr[1], "")
	if err != nil {
		cblogger.Error(err)
	}

	imageInfo := new(ImageInfo).setter(image)

	spew.Dump(imageInfo)
	return irs.ImageInfo{}, nil
}

func (imageHandler *AzureImageHandler) DeleteImage(imageID string) (bool, error) {
	imageIdArr := strings.Split(imageID, ":")

	future, err := imageHandler.Client.Delete(imageHandler.Ctx, imageIdArr[0], imageIdArr[1])
	if err != nil {
		return false, err
	}
	err = future.WaitForCompletionRef(imageHandler.Ctx, imageHandler.Client.Client)
	if err != nil {
		return false, err
	}
	return true, nil
}
