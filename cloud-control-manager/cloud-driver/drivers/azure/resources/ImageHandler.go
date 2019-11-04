package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"strings"
)

type AzureImageHandler struct {
	Region        idrv.RegionInfo
	Ctx           context.Context
	Client        *compute.ImagesClient
	VMImageClient *compute.VirtualMachineImagesClient
}

func (imageHandler *AzureImageHandler) setterImage(image compute.Image) *irs.ImageInfo {
	imageInfo := &irs.ImageInfo{
		Id:           *image.ID,
		Name:         *image.Name,
		GuestOS:      fmt.Sprint(image.ImageProperties.StorageProfile.OsDisk.OsType),
		Status:       *image.ProvisioningState,
		KeyValueList: []irs.KeyValue{{Key: "ResourceGroup", Value: imageHandler.Region.ResourceGroup}},
	}

	return imageInfo
}

func (imageHandler *AzureImageHandler) setterVMImage(image compute.VirtualMachineImage) *irs.ImageInfo {
	imageInfo := &irs.ImageInfo{
		Id:           *image.ID,
		Name:         *image.Name,
		GuestOS:      fmt.Sprint(image.OsDiskImage.OperatingSystem),
		KeyValueList: []irs.KeyValue{{Key: "ResourceGroup", Value: imageHandler.Region.ResourceGroup}},
	}

	return imageInfo
}

func (imageHandler *AzureImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	// @TODO: PublicIP 생성 요청 파라미터 정의 필요
	type ImageReqInfo struct {
		OSType string
		DiskId string
	}

	reqInfo := ImageReqInfo{
		//BlobUrl: "https://md-ds50xp550wh2.blob.core.windows.net/kt0lhznvgx2h/abcd?sv=2017-04-17&sr=b&si=b9674241-fb8e-4cb2-89c7-614d336dc3a7&sig=uvbqvAZQITSpxas%2BWosG%2FGOf6e%2BIBmWNxlUmvARnxiM%3D",
		OSType: "Linux",
		DiskId: "/subscriptions/cb592624-b77b-4a8f-bb13-0e5a48cae40f/resourceGroups/INNO-PLATFORM1-RSRC-GRUP/providers/Microsoft.Compute/disks/inno-test-vm_OsDisk_1_61bf675b990f4aa381d7ee3d766974aa",
		// edited by powerkim for test, 2019.08.13
		//DiskId: "/subscriptions/f1548292-2be3-4acd-84a4-6df079160846/resourceGroups/CB-RESOURCE-GROUP/providers/Microsoft.Compute/disks/vm_name_OsDisk_1_2d63d9cd754c4094b1b1fb6a98c36b71",
	}

	// Check Image Exists
	image, err := imageHandler.Client.Get(imageHandler.Ctx, imageHandler.Region.ResourceGroup, imageReqInfo.Name, "")
	if image.ID != nil {
		errMsg := fmt.Sprintf("Image with name %s already exist", imageReqInfo.Name)
		createErr := errors.New(errMsg)
		return irs.ImageInfo{}, createErr
	}

	createOpts := compute.Image{
		ImageProperties: &compute.ImageProperties{
			StorageProfile: &compute.ImageStorageProfile{
				OsDisk: &compute.ImageOSDisk{
					//BlobURI: to.StringPtr(reqInfo.BlobUrl),
					ManagedDisk: &compute.SubResource{
						ID: to.StringPtr(reqInfo.DiskId),
					},
					OsType: compute.OperatingSystemTypes(reqInfo.OSType),
				},
			},
		},
		Location: &imageHandler.Region.Region,
	}

	future, err := imageHandler.Client.CreateOrUpdate(imageHandler.Ctx, imageHandler.Region.ResourceGroup, imageReqInfo.Name, createOpts)
	if err != nil {
		return irs.ImageInfo{}, err
	}
	err = future.WaitForCompletionRef(imageHandler.Ctx, imageHandler.Client.Client)
	if err != nil {
		return irs.ImageInfo{}, err
	}

	// 생성된 Image 정보 리턴
	imageInfo, err := imageHandler.GetImage(imageReqInfo.Name)
	if err != nil {
		return irs.ImageInfo{}, err
	}
	return imageInfo, nil
}

func (imageHandler *AzureImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	resultList, err := imageHandler.Client.ListByResourceGroup(imageHandler.Ctx, imageHandler.Region.ResourceGroup)
	if err != nil {
		cblogger.Error(err)
	}

	var imageList []*irs.ImageInfo
	for _, image := range resultList.Values() {
		imageInfo := imageHandler.setterImage(image)
		imageList = append(imageList, imageInfo)
	}
	return imageList, nil
}

func (imageHandler *AzureImageHandler) GetImage(imageID string) (irs.ImageInfo, error) {

	imageArr := strings.Split(imageID, ":")

	// 해당 이미지 publisher, offer, skus 기준 version 목록 조회 (latest 기준 조회 불가)
	vmImageList, err := imageHandler.VMImageClient.List(imageHandler.Ctx, imageHandler.Region.Region, imageArr[0], imageArr[1], imageArr[2], "", to.Int32Ptr(1), "")

	var imageVersion string
	if len(*vmImageList.Value) != 0 {
		vmImage := (*vmImageList.Value)[0]
		imageIdArr := strings.Split(*vmImage.ID, "/")
		imageVersion = imageIdArr[len(imageIdArr)-1]
	}

	// 1개의 버전 정보를 기준으로 이미지 정보 조회
	vmImage, err := imageHandler.VMImageClient.Get(imageHandler.Ctx, imageHandler.Region.Region, imageArr[0], imageArr[1], imageArr[2], imageVersion)

	if err != nil {
		cblogger.Error(err)
		return irs.ImageInfo{}, err
	}

	//imageInfo := setterImage(image)
	//return *imageInfo, nil

	imageInfo := imageHandler.setterVMImage(vmImage)
	return *imageInfo, nil
}

func (imageHandler *AzureImageHandler) DeleteImage(imageID string) (bool, error) {
	future, err := imageHandler.Client.Delete(imageHandler.Ctx, imageHandler.Region.ResourceGroup, imageID)
	if err != nil {
		return false, err
	}
	err = future.WaitForCompletionRef(imageHandler.Ctx, imageHandler.Client.Client)
	if err != nil {
		return false, err
	}
	return true, nil
}
