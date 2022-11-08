package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-03-01/compute"
	"github.com/Azure/go-autorest/autorest/to"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	Image = "IMAGE"
)

type AzureImageHandler struct {
	Region        idrv.RegionInfo
	Ctx           context.Context
	Client        *compute.ImagesClient
	VMImageClient *compute.VirtualMachineImagesClient
}

func (imageHandler *AzureImageHandler) setterImage(image compute.Image) *irs.ImageInfo {
	imageInfo := &irs.ImageInfo{
		IId: irs.IID{
			NameId:   *image.Name,
			SystemId: *image.Name,
		},
		GuestOS:      fmt.Sprint(image.ImageProperties.StorageProfile.OsDisk.OsType),
		Status:       *image.ProvisioningState,
		KeyValueList: []irs.KeyValue{{Key: "ResourceGroup", Value: imageHandler.Region.ResourceGroup}},
	}

	return imageInfo
}

func (imageHandler *AzureImageHandler) setterVMImage(image compute.VirtualMachineImage) *irs.ImageInfo {
	imageIdArr := strings.Split(*image.ID, "/")
	imageName := fmt.Sprintf("%s:%s:%s:%s", imageIdArr[8], imageIdArr[12], imageIdArr[14], imageIdArr[16])
	imageInfo := &irs.ImageInfo{
		IId: irs.IID{
			NameId:   imageName,
			SystemId: imageName,
		},
		GuestOS:      fmt.Sprint(image.OsDiskImage.OperatingSystem),
		KeyValueList: []irs.KeyValue{{Key: "ResourceGroup", Value: imageHandler.Region.ResourceGroup}},
	}

	return imageInfo
}

func (imageHandler *AzureImageHandler) setterVMImageforList(image compute.VirtualMachineImageResource) *irs.ImageInfo {
	imageIdArr := strings.Split(*image.ID, "/")
	imageName := fmt.Sprintf("%s:%s:%s:%s", imageIdArr[8], imageIdArr[12], imageIdArr[14], imageIdArr[16])
	imageInfo := &irs.ImageInfo{
		IId: irs.IID{
			NameId:   imageName,
			SystemId: imageName,
		},
		//GuestOS:      fmt.Sprint(image.OsDiskImage.OperatingSystem),
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
	image, err := imageHandler.Client.Get(imageHandler.Ctx, imageHandler.Region.ResourceGroup, imageReqInfo.IId.NameId, "")
	if image.ID != nil {
		errMsg := fmt.Sprintf("Image with name %s already exist", imageReqInfo.IId.NameId)
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

	future, err := imageHandler.Client.CreateOrUpdate(imageHandler.Ctx, imageHandler.Region.ResourceGroup, imageReqInfo.IId.NameId, createOpts)
	if err != nil {
		return irs.ImageInfo{}, err
	}
	err = future.WaitForCompletionRef(imageHandler.Ctx, imageHandler.Client.Client)
	if err != nil {
		return irs.ImageInfo{}, err
	}

	// 생성된 Image 정보 리턴
	imageInfo, err := imageHandler.GetImage(imageReqInfo.IId)
	if err != nil {
		return irs.ImageInfo{}, err
	}
	return imageInfo, nil
}

func (imageHandler *AzureImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, Image, "ListImage()")

	var imageList []*irs.ImageInfo

	publishers, err := imageHandler.VMImageClient.ListPublishers(context.TODO(), imageHandler.Region.Region)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	start := call.Start()

	var publisherWg sync.WaitGroup
	publisherWg.Add(len(*publishers.Value))

	for _, p := range *publishers.Value {
		go func(publisher compute.VirtualMachineImageResource) {
			defer publisherWg.Done()
			offers, err := imageHandler.VMImageClient.ListOffers(context.TODO(), imageHandler.Region.Region, *publisher.Name)
			if err != nil {
				return
			}

			for _, offer := range *offers.Value {
				skus, err := imageHandler.VMImageClient.ListSkus(context.TODO(), imageHandler.Region.Region, *publisher.Name, *offer.Name)
				if err != nil {
					continue
				}
				for _, sku := range *skus.Value {
					imageVersionList, err := imageHandler.VMImageClient.List(context.TODO(), imageHandler.Region.Region, *publisher.Name, *offer.Name, *sku.Name, "", nil, "")
					if err != nil {
						continue
					}
					if len(*imageVersionList.Value) == 0 {
						continue
					}
					for _, version := range *imageVersionList.Value {
						imageIdArr := strings.Split(*version.ID, "/")
						imageVersion := imageIdArr[len(imageIdArr)-1]
						vmImage, err := imageHandler.VMImageClient.Get(context.TODO(), imageHandler.Region.Region, *publisher.Name, *offer.Name, *sku.Name, imageVersion)
						if err != nil {
							continue
						}
						vmImageInfo := imageHandler.setterVMImage(vmImage)
						imageList = append(imageList, vmImageInfo)
					}
				}
			}
			return
		}(p)
	}

	publisherWg.Wait()
	LoggingInfo(hiscallInfo, start)
	return imageList, nil
}

func (imageHandler *AzureImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, imageIID.NameId, "GetImage()")

	imageArr := strings.Split(imageIID.NameId, ":")

	// 이미지 URN 형식 검사
	if len(imageArr) != 4 {
		formatErr := errors.New("invalid format for image ID, imageId=" + imageIID.NameId)
		cblogger.Error(formatErr.Error())
		LoggingError(hiscallInfo, formatErr)
		return irs.ImageInfo{}, formatErr
	}

	// 해당 이미지 publisher, offer, skus 기준 version 목록 조회 (latest 기준 조회 기능 미활용)
	/*
		imageVersion := imageArr[3]
		if strings.EqualFold(imageVersion, "latest") {
			vmImageList, err := imageHandler.VMImageClient.List(imageHandler.Ctx, imageHandler.Region.Region, imageArr[0], imageArr[1], imageArr[2], "", to.Int32Ptr(1), "name desc")
			if err != nil {
				LoggingError(hiscallInfo, err)
				return irs.ImageInfo{}, err
			}
			if &vmImageList == nil {
				getErr := errors.New(fmt.Sprintf("could not found image with imageId %s", imageIID.NameId))
				LoggingError(hiscallInfo, getErr)
				return irs.ImageInfo{}, getErr
			}
			if vmImageList.Value == nil {
				getErr := errors.New(fmt.Sprintf("could not found image with imageId %s", imageIID.NameId))
				LoggingError(hiscallInfo, getErr)
				return irs.ImageInfo{}, getErr
			}
			if len(*vmImageList.Value) == 0 {
				getErr := errors.New(fmt.Sprintf("could not found image with imageId %s", imageIID.NameId))
				LoggingError(hiscallInfo, getErr)
				return irs.ImageInfo{}, getErr
			} else {
				latestVmImage := (*vmImageList.Value)[0]
				imageIdArr := strings.Split(*latestVmImage.ID, "/")
				imageVersion = imageIdArr[len(imageIdArr)-1]
			}
		}
	*/

	// 1개의 버전 정보를 기준으로 이미지 정보 조회
	start := call.Start()
	vmImage, err := imageHandler.VMImageClient.Get(imageHandler.Ctx, imageHandler.Region.Region, imageArr[0], imageArr[1], imageArr[2], imageArr[3])
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.ImageInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	imageInfo := imageHandler.setterVMImage(vmImage)
	return *imageInfo, nil
}

func (imageHandler *AzureImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(imageHandler.Region, call.VMIMAGE, imageIID.NameId, "DeleteImage()")

	start := call.Start()
	future, err := imageHandler.Client.Delete(imageHandler.Ctx, imageHandler.Region.ResourceGroup, imageIID.NameId)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	err = future.WaitForCompletionRef(imageHandler.Ctx, imageHandler.Client.Client)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return false, err
	}

	return true, nil
}

func (imageHandler *AzureImageHandler) CheckWindowsImage(imageIID irs.IID) (bool, error) {
	if imageIID.NameId == "" && imageIID.SystemId == "" {
		return false, errors.New("failed get OSType By ImageIID err = empty ImageIID")
	}
	imageName := imageIID.NameId
	if imageIID.NameId == "" {
		imageName = imageIID.SystemId
	}
	imageNameSplits := strings.Split(imageName, ":")
	if len(imageNameSplits) != 4 {
		return false, errors.New("failed get OSType By ImageIID err = invalid ImageIID, Image Name must be in the form of 'Publisher:Offer:Sku:Version'. ")
	}
	offer := imageNameSplits[1]
	if strings.Contains(strings.ToLower(offer), "window") {
		return true, nil
	}
	return false, nil
}
