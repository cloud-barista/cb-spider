package resources

import (
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/image"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"
	"github.com/davecgh/go-spew/spew"
	//"strconv"
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
		VolumeId:     "7f7a87f0-3acb-4313-90f7-22eb65a6d33f",
		SnapshotId:   "",
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
		spew.Dump(image)
		return irs.ImageInfo{Id: image.ID, Name: image.Name}, nil
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

func (imageHandler *ClouditImageHandler) GetImage(imageID string) (irs.ImageInfo, error) {
	imageHandler.Client.TokenID = imageHandler.CredentialInfo.AuthToken
	authHeader := imageHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if image, err := image.Get(imageHandler.Client, imageID, &requestOpts); err != nil {
		return irs.ImageInfo{}, err
	} else {
		spew.Dump(image)
		return irs.ImageInfo{Id: image.ID, Name: image.Name}, nil
	}
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
