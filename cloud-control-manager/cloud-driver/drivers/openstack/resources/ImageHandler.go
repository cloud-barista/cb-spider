package resources

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/imagedata"
	imgsvc "github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	Image = "IMAGE"
)

type OpenStackImageHandler struct {
	Client      *gophercloud.ServiceClient
	ImageClient *gophercloud.ServiceClient
}

func getMoreImageInfoFromAPI(imageClient *gophercloud.ServiceClient, imageID string) (map[string]interface{}, error) {
	url := imageClient.ServiceURL("images", imageID)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-Auth-Token", imageClient.TokenID)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get image details: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var serverResponse map[string]interface{}
	if err := json.Unmarshal(body, &serverResponse); err != nil {
		return nil, err
	}

	return serverResponse, nil
}

func formatDiskSizeValue(value float64) string {
	value = value / 1024 / 1024 / 1024

	if value == float64(int64(value)) {
		return fmt.Sprintf("%d", int64(value))
	}

	str := fmt.Sprintf("%.3f", value)
	return strings.TrimRight(strings.TrimRight(str, "0"), ".")
}

func setterImage(imageClient *gophercloud.ServiceClient, image images.Image) *irs.ImageInfo {
	resp, err := getMoreImageInfoFromAPI(imageClient, image.ID)
	if err != nil {
		cblogger.Error(err)
	}

	diskType := resp["disk_format"].(string)
	diskSize := resp["size"].(float64)

	imageStatus := irs.ImageUnavailable
	status := strings.ToLower(image.Status)
	if status == "active" {
		imageStatus = irs.ImageAvailable
	}

	imageInfo := &irs.ImageInfo{
		Name:           image.ID,
		OSArchitecture: irs.ArchitectureNA,
		OSPlatform:     irs.PlatformNA,
		OSDistribution: image.Name,
		OSDiskType:     diskType,
		OSDiskSizeInGB: formatDiskSizeValue(diskSize),
		ImageStatus:    imageStatus,
	}

	var keyValueList []irs.KeyValue
	for key, val := range resp {
		if key == "os_hidden" || key == "os_hash_algo" || key == "os_hash_value" {
			property := irs.KeyValue{
				Key:   key,
				Value: fmt.Sprintf("%v", val),
			}
			keyValueList = append(keyValueList, property)
		}
	}
	imageInfo.KeyValueList = keyValueList

	return imageInfo
}

func (imageHandler *OpenStackImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(imageHandler.Client.IdentityEndpoint, call.VMIMAGE, imageReqInfo.IId.NameId, "CreateImage()")

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
	rootPath := os.Getenv("CBSPIDER_ROOT")
	imageFilePath := fmt.Sprintf("%s/image/%s.iso", rootPath, reqInfo.Name)
	if _, err := os.Stat(imageFilePath); os.IsNotExist(err) {
		createErr := errors.New(fmt.Sprintf("Image files in path %s not exist", imageFilePath))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.ImageInfo{}, createErr
	}

	// Create Image
	start := call.Start()
	image, err := imgsvc.Create(imageHandler.ImageClient, createOpts).Extract()
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.ImageInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	// Upload Image file
	imageBytes, err := ioutil.ReadFile(imageFilePath)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.ImageInfo{}, err
	}
	result := imagedata.Upload(imageHandler.ImageClient, image.ID, bytes.NewReader(imageBytes))
	if result.Err != nil {
		cblogger.Error(result.Err.Error())
		LoggingError(hiscallInfo, err)
		return irs.ImageInfo{}, err
	}

	// 생성된 Imgae 정보 리턴
	mappedImageInfo := images.Image{
		ID:       image.ID,
		Created:  image.CreatedAt.String(),
		MinDisk:  image.MinDiskGigabytes,
		MinRAM:   image.MinRAMMegabytes,
		Name:     image.Name,
		Status:   string(image.Status),
		Updated:  image.UpdatedAt.String(),
		Metadata: image.Properties,
	}
	imageInfo := setterImage(imageHandler.ImageClient, mappedImageInfo)
	return *imageInfo, nil
}

func (imageHandler *OpenStackImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(imageHandler.Client.IdentityEndpoint, call.VMIMAGE, Image, "ListImage()")

	start := call.Start()
	imageList, err := getRawImageList(imageHandler.Client)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List Image. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	imageInfoList := make([]*irs.ImageInfo, len(imageList))
	for i, img := range imageList {
		imageInfo := setterImage(imageHandler.ImageClient, img)
		imageInfoList[i] = imageInfo
	}
	LoggingInfo(hiscallInfo, start)
	return imageInfoList, nil
}

func (imageHandler *OpenStackImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(imageHandler.Client.IdentityEndpoint, call.VMIMAGE, imageIID.NameId, "GetImage()")

	start := call.Start()
	image, err := getRawImage(imageIID, imageHandler.Client)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Image. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.ImageInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)

	imageInfo := setterImage(imageHandler.ImageClient, image)
	return *imageInfo, nil
}

func (imageHandler *OpenStackImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(imageHandler.Client.IdentityEndpoint, call.VMIMAGE, imageIID.NameId, "DeleteImage()")
	start := call.Start()
	image, err := getRawImage(imageIID, imageHandler.Client)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return false, err
	}
	err = images.Delete(imageHandler.Client, image.ID).ExtractErr()
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}
func getRawImage(imageIId irs.IID, computeClient *gophercloud.ServiceClient) (images.Image, error) {
	if !CheckIIDValidation(imageIId) {
		return images.Image{}, errors.New("invalid IID")
	}
	if imageIId.SystemId == "" {
		imageIId.SystemId = imageIId.NameId
	}
	image, err := images.Get(computeClient, imageIId.SystemId).Extract()
	if err != nil {
		return images.Image{}, err
	}
	return *image, nil
}

func getRawImageList(computeClient *gophercloud.ServiceClient) ([]images.Image, error) {
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
		if !snapshotFlag {
			imageList = append(imageList, image)
		}
	}
	if imageList == nil {
		emptyList := make([]images.Image, 0)
		return emptyList, nil
	}
	return imageList, err
}

func (imageHandler *OpenStackImageHandler) CheckWindowsImage(imageIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(imageHandler.Client.IdentityEndpoint, call.VMIMAGE, imageIID.NameId, "CheckWindowsImage()")
	start := call.Start()
	image, err := getRawImage(imageIID, imageHandler.Client)
	if err != nil {
		checkWindowsImageErr := errors.New(fmt.Sprintf("Failed to CheckWindowsImage By Image. err = %s", err.Error()))
		cblogger.Error(checkWindowsImageErr.Error())
		LoggingError(hiscallInfo, checkWindowsImageErr)
		return false, checkWindowsImageErr
	}
	LoggingInfo(hiscallInfo, start)
	value, exist := image.Metadata["os_type"]
	if !exist {
		return false, nil
	}
	if value == "windows" {
		return true, nil
	}
	return false, nil
}
