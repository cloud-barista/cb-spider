package resources

import (
	"errors"
	"fmt"
	"strings"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/image"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	Image = "IMAGE"
)

type ClouditImageHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func setterImage(image image.ImageInfo) *irs.ImageInfo {
	imageInfo := &irs.ImageInfo{
		IId: irs.IID{
			NameId:   image.Name,
			SystemId: image.Name,
		},
		GuestOS: image.OS,
		Status:  image.State,
	}
	return imageInfo
}

func (imageHandler *ClouditImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(imageHandler.Client.IdentityEndpoint, call.VMIMAGE, imageReqInfo.IId.NameId, "CreateImage()")

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

	// Create Image
	start := call.Start()
	image, err := image.Create(imageHandler.Client, &createOpts)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.ImageInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	imageInfo := setterImage(*image)
	return *imageInfo, nil
}

func (imageHandler *ClouditImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(imageHandler.Client.IdentityEndpoint, call.VMIMAGE, Image, "ListImage()")

	imageHandler.Client.TokenID = imageHandler.CredentialInfo.AuthToken
	authHeader := imageHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	start := call.Start()
	imageList, err := image.List(imageHandler.Client, &requestOpts)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	resultList := make([]*irs.ImageInfo, len(*imageList))
	for i, vmImage := range *imageList {
		imageInfo := setterImage(vmImage)
		resultList[i] = imageInfo
	}
	return resultList, nil
}

func (imageHandler *ClouditImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(imageHandler.Client.IdentityEndpoint, call.VMIMAGE, imageIID.NameId, "GetImage()")

	start := call.Start()
	imageInfo, err := imageHandler.getImageByName(imageIID.NameId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.ImageInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	return *imageInfo, nil
}

func (imageHandler *ClouditImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(imageHandler.Client.IdentityEndpoint, call.VMIMAGE, imageIID.NameId, "DeleteImage()")

	imageHandler.Client.TokenID = imageHandler.CredentialInfo.AuthToken
	authHeader := imageHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	start := call.Start()
	err := image.Delete(imageHandler.Client, imageIID.SystemId, &requestOpts)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
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
