package resources

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/images"
	imgsvc "github.com/rackspace/gophercloud/openstack/imageservice/v2/images"

	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type OpenStackImageHandler struct {
	Client      *gophercloud.ServiceClient
	ImageClient *gophercloud.ServiceClient
}

func setterImage(image images.Image) *irs.ImageInfo {
	imageInfo := &irs.ImageInfo{
		IId: irs.IID{
			NameId:   image.Name,
			SystemId: image.Name,
		},
		Status: image.Status,
	}

	// 메타 정보 등록
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
		Name:            imageReqInfo.IId.NameId,
		ContainerFormat: "bare",
		DiskFormat:      "iso",
	}

	createOpts := imgsvc.CreateOpts{
		Name:            reqInfo.Name,
		ContainerFormat: reqInfo.ContainerFormat,
		DiskFormat:      reqInfo.DiskFormat,
	}

	// Check Image file exists
	rootPath := os.Getenv("CBSPIDER_PATH")
	imageFilePath := fmt.Sprintf("%s/image/%s.iso", rootPath, reqInfo.Name)
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
	imageBytes, err := ioutil.ReadFile(imageFilePath)
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
	pager, err := images.ListDetail(imageHandler.Client, images.ListOpts{}).AllPages()
	if err != nil {
		return nil, err
	}
	imageList, err := images.ExtractImages(pager)
	if err != nil {
		return nil, err
	}

	imageInfoList := make([]*irs.ImageInfo, len(imageList))
	for i, img := range imageList {
		imageInfo := setterImage(img)
		imageInfoList[i] = imageInfo
	}
	return imageInfoList, nil
}

func (imageHandler *OpenStackImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
	imageId, err := images.IDFromName(imageHandler.Client, imageIID.NameId)
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

func (imageHandler *OpenStackImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	/*imageId, err := images.IDFromName(imageHandler.Client, imageIID.NameId)
	if err != nil {
		return false, err
	}*/
	err := images.Delete(imageHandler.Client, imageIID.SystemId).ExtractErr()
	if err != nil {
		return false, err
	}
	return true, nil
}
