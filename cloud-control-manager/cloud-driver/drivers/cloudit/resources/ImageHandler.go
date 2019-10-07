package resources

import (
	//"fmt"
	//cblog "github.com/cloud-barista/cb-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/ace/image"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	//"github.com/sirupsen/logrus"
	"strconv"
)

type ClouditImageHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func (imageHandler *ClouditImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	imageHandler.Client.TokenID = imageHandler.CredentialInfo.AuthToken
	authHeader := imageHandler.Client.AuthenticatedHeaders()

	// @TODO: Image 생성 요청 파라미터 정의 필요
	type ImageReqInfo struct {
		Name         string `json:"name" required:"true"`
		VolumeId     string `json:"volumeId" required:"true"`   // 정지된 서버 볼륨을 기준으로 이미지 템플릿 생성
		SnapshotId   string `json:"snapshotId" required:"true"` // 서버 스냅샷을 기준으로 이미지 템플릿 생성
		Ownership    string `json:"ownership" required:"true"`  // TENANT, PRIVATE
		Format       string `json:"format" required:"true"`     // raw, vdi, vmdk, vpc, qcow2
		SourceType   string `json:"sourceType" required:"true"` // server, snapshot
		TemplateType string `json:"templateType" required:"true"`
		Size         int    `json:"size" required:"false"`
		PoolId       string `json:"poolId" required:"false"`
		Protection   int    `json:"protection" required:"false"`
	}

	reqInfo := ImageReqInfo{
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
		for i, image := range *imageList {
			cblogger.Info("[" + strconv.Itoa(i) + "]")
			spew.Dump(image)
		}
		return nil, nil
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
