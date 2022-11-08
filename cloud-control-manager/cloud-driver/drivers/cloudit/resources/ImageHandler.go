package resources

import (
	"errors"
	"fmt"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/snapshot"
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
			NameId:   image.ID,
			SystemId: image.ID,
		},
		GuestOS: image.OS,
		Status:  image.State,
	}
	return imageInfo
}

func (imageHandler *ClouditImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VMIMAGE, imageReqInfo.IId.NameId, "CreateImage()")

	createErr := errors.New(fmt.Sprintf("Failed to Create Image. err = CreateImage Function Not Offer"))
	cblogger.Error(createErr.Error())
	LoggingError(hiscallInfo, createErr)

	return irs.ImageInfo{}, createErr

	//imageHandler.Client.TokenID = imageHandler.CredentialInfo.AuthToken
	//authHeader := imageHandler.Client.AuthenticatedHeaders()

	//reqInfo := image.ImageReqInfo{
	//	Name:         imageReqInfo.IId.NameId,
	//	VolumeId:     "fa4bb8d7-bf09-4fd7-b123-d08677ac0691",
	//	SnapshotId:   "dbc61213-b37e-4cc2-94ca-47991337e36f",
	//	Ownership:    "TENANT",
	//	Format:       "qcow2",
	//	SourceType:   "server",
	//	TemplateType: "DEFAULT",
	//}
	//
	//createOpts := client.RequestOpts{
	//	JSONBody:    reqInfo,
	//	MoreHeaders: authHeader,
	//}

	// Create Image
	//start := call.Start()
	//image, err := image.Create(imageHandler.Client, &createOpts)
	//if err != nil {
	//	cblogger.Error(err.Error())
	//	LoggingError(hiscallInfo, err)
	//	return irs.ImageInfo{}, err
	//}
	// LoggingInfo(hiscallInfo, start)

	//imageInfo := setterImage(*image)
	//return *imageInfo, nil
}

func (imageHandler *ClouditImageHandler) ListImage() ([]*irs.ImageInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VMIMAGE, Image, "ListImage()")

	imageHandler.Client.TokenID = imageHandler.CredentialInfo.AuthToken
	authHeader := imageHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	start := call.Start()
	imageList, err := image.List(imageHandler.Client, &requestOpts)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get ImageList. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	resultList := make([]*irs.ImageInfo, len(*imageList))
	for i, vmImage := range *imageList {
		imageInfo := setterImage(vmImage)
		resultList[i] = imageInfo
	}
	LoggingInfo(hiscallInfo, start)
	return resultList, nil
}

func (imageHandler *ClouditImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VMIMAGE, imageIID.NameId, "GetImage()")

	start := call.Start()
	rawImage, err := imageHandler.getRawImage(imageIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Image. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.ImageInfo{}, getErr
	}
	imageInfo := setterImage(*rawImage)
	LoggingInfo(hiscallInfo, start)

	return *imageInfo, nil
}

func (imageHandler *ClouditImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VMIMAGE, imageIID.NameId, "DeleteImage()")

	// Create not offer => Delete not offer
	//imageHandler.Client.TokenID = imageHandler.CredentialInfo.AuthToken
	//authHeader := imageHandler.Client.AuthenticatedHeaders()
	//
	//requestOpts := client.RequestOpts{
	//	MoreHeaders: authHeader,
	//}
	//
	//start := call.Start()
	//err := image.Delete(imageHandler.Client, imageIID.SystemId, &requestOpts)
	//if err != nil {
	//	getErr := errors.New(fmt.Sprintf("Failed to Delete Image. err = %s", err.Error()))
	//	cblogger.Error(getErr.Error())
	//	LoggingError(hiscallInfo, getErr)
	//	return false, getErr
	//}
	//LoggingInfo(hiscallInfo, start)
	//
	//return true, nil

	createErr := errors.New(fmt.Sprintf("Failed to Delete Image. err = DeleteImage Function Not Offer"))
	cblogger.Error(createErr.Error())
	LoggingError(hiscallInfo, createErr)

	return false, createErr
}

func (imageHandler *ClouditImageHandler) getRawImage(imageIId irs.IID) (*image.ImageInfo, error) {
	if imageIId.SystemId == "" && imageIId.NameId == "" {
		return nil, errors.New("invalid IID")
	}
	imageHandler.Client.TokenID = imageHandler.CredentialInfo.AuthToken
	authHeader := imageHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	imageList, err := image.List(imageHandler.Client, &requestOpts)
	if err != nil {
		return nil, err
	}

	for _, rawImage := range *imageList {
		if strings.EqualFold(imageIId.SystemId, rawImage.ID) {
			return &rawImage, nil
		}
	}
	return nil, errors.New("not found image")
}

func (imageHandler *ClouditImageHandler) GetRawRootImage(imageIId irs.IID, isMyImage bool) (*image.ImageInfo, error) {
	if imageIId.SystemId == "" && imageIId.NameId == "" {
		return nil, errors.New("invalid IID")
	}
	imageHandler.Client.TokenID = imageHandler.CredentialInfo.AuthToken
	authHeader := imageHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	if isMyImage {
		snapshotList, err := snapshot.List(imageHandler.Client, &requestOpts)
		if err != nil {
			return nil, err
		}

		for _, rawSnapshot := range *snapshotList {
			if strings.Contains(rawSnapshot.Name, imageIId.NameId) && rawSnapshot.Bootable == "yes" {
				rootImage, getRootImageErr := imageHandler.getRawImage(irs.IID{SystemId: rawSnapshot.TemplateId})
				if getRootImageErr != nil {
					return nil, getRootImageErr
				}
				return rootImage, nil
			}
		}
	} else {
		imageList, err := image.List(imageHandler.Client, &requestOpts)
		if err != nil {
			return nil, err
		}

		for _, rawImage := range *imageList {
			if strings.EqualFold(imageIId.SystemId, rawImage.ID) {
				return &rawImage, nil
			}
		}
	}
	return nil, errors.New("not found image")
}

func (imageHandler *ClouditImageHandler) CheckWindowsImage(imageIID irs.IID) (bool, error) {
	rawRootImage, getRawRootImageErr := imageHandler.GetRawRootImage(imageIID, false)
	if getRawRootImageErr != nil {
		return false, errors.New(fmt.Sprintf("Failed to Check Windows Image. err = %s", getRawRootImageErr.Error()))
	}

	isWindows := strings.Contains(strings.ToLower(rawRootImage.OS), "windows")
	return isWindows, nil
}
