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
		IId: irs.IID{
			NameId:   image.Name,
			SystemId: image.ID,
		},
		GuestOS: image.OS,
		Status:  image.State,
	}
	return imageInfo
}

func (imageHandler *ClouditImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	imageHandler.Client.TokenID = imageHandler.CredentialInfo.AuthToken
	authHeader := imageHandler.Client.AuthenticatedHeaders()

	reqInfo := image.ImageReqInfo{
		Name:         imageReqInfo.IId.NameId,
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

	image, err := image.Create(imageHandler.Client, &createOpts)
	if err != nil {
		return irs.ImageInfo{}, err
	}
	imageInfo := setterImage(*image)
	return *imageInfo, nil
}

func (imageHandler *ClouditImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	imageHandler.Client.TokenID = imageHandler.CredentialInfo.AuthToken
	authHeader := imageHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	imageList, err := image.List(imageHandler.Client, &requestOpts)
	if err != nil {
		return nil, err
	}

	resultList := make([]*irs.ImageInfo, len(*imageList))
	for i, image := range *imageList {
		imageInfo := setterImage(image)
		resultList[i] = imageInfo
	}
	return resultList, nil
}

func (imageHandler *ClouditImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
	imageInfo, err := imageHandler.getImageByName(imageIID.NameId)
	if err != nil {
		return irs.ImageInfo{}, err
	}
	return *imageInfo, nil
}

func (imageHandler *ClouditImageHandler) DeleteImage(mageIID irs.IID) (bool, error) {
	imageHandler.Client.TokenID = imageHandler.CredentialInfo.AuthToken
	authHeader := imageHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if err := image.Delete(imageHandler.Client, mageIID.SystemId, &requestOpts); err != nil {
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
		if strings.EqualFold(image.ID, imageName) {
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
