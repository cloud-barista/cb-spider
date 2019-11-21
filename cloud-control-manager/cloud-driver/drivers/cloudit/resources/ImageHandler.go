package resources

import (
	"errors"
	"fmt"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/image"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"strings"
)

type ClouditImageHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func setterImage(image image.ImageInfo) *irs.ImageInfo {
	imageInfo := &irs.ImageInfo{
		Id:      image.ID,
		Name:    image.Name,
		GuestOS: image.OS,
		Status:  image.State,
	}
	return imageInfo
}

func (imageHandler *ClouditImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	imageHandler.Client.TokenID = imageHandler.CredentialInfo.AuthToken
	authHeader := imageHandler.Client.AuthenticatedHeaders()

	reqInfo := image.ImageReqInfo{
		Name:         imageReqInfo.Name,
		VolumeId:     "fa4bb8d7-bf09-4fd7-b123-d08677ac0691",
		SnapshotId:   "dbc61213-b37e-4cc2-94ca-47991337e36f",
		Ownership:    "TENANT",
		Format:       "qcow2",
		SourceType:   "server",
		TemplateType: "DEFAULT",
	}

	createOpts := client.RequestOpts{
		JSONBody:    reqInfo,
		MoreHeaders: authHeader,
	}

	if image, err := image.Create(imageHandler.Client, &createOpts); err != nil {
		return irs.ImageInfo{}, err
	} else {
		imageInfo := setterImage(*image)
		return *imageInfo, nil
	}
}

func (imageHandler *ClouditImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	imageHandler.Client.TokenID = imageHandler.CredentialInfo.AuthToken
	authHeader := imageHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if imageList, err := image.List(imageHandler.Client, &requestOpts); err != nil {
		return nil, err
	} else {
		var resultList []*irs.ImageInfo

		for _, image := range *imageList {
			imageInfo := setterImage(image)
			resultList = append(resultList, imageInfo)
		}
		return resultList, nil
	}
}

func (imageHandler *ClouditImageHandler) GetImage(imageNameId string) (irs.ImageInfo, error) {

	imageInfo, err := imageHandler.getImageByName(imageNameId)
	if err != nil {
		return irs.ImageInfo{}, err
	}
	return *imageInfo, nil
}

func (imageHandler *ClouditImageHandler) DeleteImage(imageID string) (bool, error) {
	imageHandler.Client.TokenID = imageHandler.CredentialInfo.AuthToken
	authHeader := imageHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if err := image.Delete(imageHandler.Client, imageID, &requestOpts); err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func (imageHandler *ClouditImageHandler) getImageByName(imageName string) (*irs.ImageInfo, error) {
	imageHandler.Client.TokenID = imageHandler.CredentialInfo.AuthToken
	authHeader := imageHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	imageList, err := image.List(imageHandler.Client, &requestOpts)
	if err != nil {
		return nil, err
	}

	var imageInfo *irs.ImageInfo
	for _, image := range *imageList {
		if strings.EqualFold(image.Name, imageName) {
			imageInfo = setterImage(image)
			break
		}
	}

	if imageInfo == nil {
		err := errors.New(fmt.Sprintf("failed to find image with name %s", imageName))
		return nil, err
	}
	return imageInfo, nil
}
