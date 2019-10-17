package resources

import (
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/image"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"
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
		//spew.Dump(image)
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

func (imageHandler *ClouditImageHandler) GetImage(imageID string) (irs.ImageInfo, error) {
	imageHandler.Client.TokenID = imageHandler.CredentialInfo.AuthToken
	authHeader := imageHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if image, err := image.Get(imageHandler.Client, imageID, &requestOpts); err != nil {
		return irs.ImageInfo{}, err
	} else {
		//spew.Dump(image)
		imageInfo := setterImage(*image)
		return *imageInfo, nil
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
