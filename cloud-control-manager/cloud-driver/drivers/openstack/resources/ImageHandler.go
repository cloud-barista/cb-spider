package resources

import (
	"bytes"
	"errors"
	"fmt"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/images"
	imgsvc "github.com/rackspace/gophercloud/openstack/imageservice/v2/images"
	"github.com/rackspace/gophercloud/pagination"
	"io/ioutil"
	"os"
)

type OpenStackImageHandler struct {
	Client      *gophercloud.ServiceClient
	ImageClient *gophercloud.ServiceClient
}

func setterImage(image images.Image) *irs.ImageInfo {
	imageInfo := &irs.ImageInfo{
		Id:     image.ID,
		Name:   image.Name,
		Status: image.Status,
	}

	// 메타 정보 등록
	//metadataList := make([]irs.KeyValue, len(image.Metadata))
	var metadataList []irs.KeyValue
	for key, val := range image.Metadata {
		metadata := irs.KeyValue{
			Key:   key,
			Value: val,
		}
		metadataList = append(metadataList, metadata)
	}
	imageInfo.KeyValueList = metadataList

	return imageInfo
}

func (imageHandler *OpenStackImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {

	// @TODO: Image 생성 요청 파라미터 정의 필요
	type ImageReqInfo struct {
		Name            string
		ContainerFormat string
		DiskFormat      string
	}
	reqInfo := ImageReqInfo{
		Name:            imageReqInfo.Name,
		ContainerFormat: "bare",
		DiskFormat:      "iso",
	}

	createOpts := imgsvc.CreateOpts{
		Name:            reqInfo.Name,
		ContainerFormat: reqInfo.ContainerFormat,
		DiskFormat:      reqInfo.DiskFormat,
	}

	rootPath := os.Getenv("CBSPIDER_PATH")

	// Check Image file exists
	imageFilePath := rootPath + "/image/mcb_custom_image.iso"
	if _, err := os.Stat(imageFilePath); os.IsNotExist(err) {
		errMsg := fmt.Sprintf("Image files in path %s not exist", imageFilePath)
		createErr := errors.New(errMsg)
		return irs.ImageInfo{}, createErr
	}

	// Create Image
	image, err := imgsvc.Create(imageHandler.ImageClient, createOpts).Extract()
	if err != nil {
		return irs.ImageInfo{}, err
	}

	// Upload Image file
	imageBytes, err := ioutil.ReadFile(rootPath + "/image/mcb_custom_image.iso")
	if err != nil {
		return irs.ImageInfo{}, err
	}
	result := imgsvc.Upload(imageHandler.ImageClient, image.ID, bytes.NewReader(imageBytes))
	if result.Err != nil {
		return irs.ImageInfo{}, err
	}

	// 생성된 Imgae 정보 리턴
	mappedImageInfo := images.Image{
		ID:       image.ID,
		Created:  image.CreatedDate,
		MinDisk:  image.MinDiskGigabytes,
		MinRAM:   image.MinRAMMegabytes,
		Name:     image.Name,
		Status:   string(image.Status),
		Updated:  image.LastUpdate,
		Metadata: image.Metadata,
	}
	imageInfo := setterImage(mappedImageInfo)
	return *imageInfo, nil
}

func (imageHandler *OpenStackImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	var imageList []*irs.ImageInfo

	pager := images.ListDetail(imageHandler.Client, images.ListOpts{})
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		// Get Image
		list, err := images.ExtractImages(page)
		if err != nil {
			return false, err
		}
		// Add to List
		for _, img := range list {
			imageInfo := setterImage(img)
			imageList = append(imageList, imageInfo)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	return imageList, nil
}

func (imageHandler *OpenStackImageHandler) GetImage(imageNameId string) (irs.ImageInfo, error) {
	imageId, err := images.IDFromName(imageHandler.Client, imageNameId)
	if err != nil {
		return irs.ImageInfo{}, err
	}
	image, err := images.Get(imageHandler.Client, imageId).Extract()
	if err != nil {
		return irs.ImageInfo{}, err
	}

	imageInfo := setterImage(*image)
	return *imageInfo, nil
}

func (imageHandler *OpenStackImageHandler) DeleteImage(imageID string) (bool, error) {
	/*imageId, err := images.IDFromName(imageHandler.Client, imageID)
	if err != nil {
		return false, err
	}*/
	err := images.Delete(imageHandler.Client, imageID).ExtractErr()
	if err != nil {
		return false, err
	}
	return true, nil
}
