// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// program by ysjeon@mz.co.kr, 2019.07.
// modify by devunet@mz.co.kr, 2019.11.

package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	compute "google.golang.org/api/compute/v1"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type GCPImageHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

/*
이미지를 생성할 때 GCP 같은 경우는 내가 생성한 이미지에서만 리스트를 가져 올 수 있다.
퍼블릭 이미지를 가져 올 수 없다.
가져올라면 다르게 해야 함.
Insert할때 필수 값
name, sourceDisk(sourceImage),storageLocations(배열 ex : ["asia"])
이미지를 어떻게 생성하는냐에 따라서 키 값이 변경됨
디스크, 스냅샷,이미지, 가상디스크, Cloud storage
1) Disk일 경우 :
	{"sourceDisk": "projects/mcloud-barista-251102/zones/asia-northeast1-b/disks/my-root-pd",}
2) Image일 경우 :
	{"sourceImage": "projects/mcloud-barista-251102/global/images/image-1",}



*/

func (imageHandler *GCPImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	return irs.ImageInfo{}, errors.New("Feature not implemented.")
}

func (imageHandler *GCPImageHandler) ListImage() ([]*irs.ImageInfo, error) {

	//projectId := imageHandler.Credential.ProjectID
	projectId := "gce-uefi-images"

	// list, err := imageHandler.Client.Images.List(projectId).Do()
	list, err := imageHandler.Client.Images.List(projectId).Do()
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}
	var imageList []*irs.ImageInfo
	for _, item := range list.Items {
		info := mappingImageInfo(item)
		imageList = append(imageList, &info)
	}

	spew.Dump(imageList)
	return imageList, nil
}

func (imageHandler *GCPImageHandler) GetImage(imageID string) (irs.ImageInfo, error) {
	//projectId := imageHandler.Credential.ProjectID
	projectId := "gce-uefi-images"

	image, err := imageHandler.Client.Images.Get(projectId, imageID).Do()
	if err != nil {
		cblogger.Error(err)
		return irs.ImageInfo{}, err
	}
	imageInfo := mappingImageInfo(image)
	return imageInfo, nil
}

func (imageHandler *GCPImageHandler) DeleteImage(imageID string) (bool, error) {
	// public Image 는 지울 수 없는데 어떻게 해야 하는가?
	projectId := imageHandler.Credential.ProjectID

	res, err := imageHandler.Client.Images.Delete(projectId, imageID).Do()
	if err != nil {
		cblogger.Error(err)
		return false, err
	}
	fmt.Println(res)
	return true, err
}

func mappingImageInfo(imageInfo *compute.Image) irs.ImageInfo {
	//lArr := strings.Split(imageInfo.Licenses[0], "/")
	//os := lArr[len(lArr)-1]
	imageList := irs.ImageInfo{
		//Id:      strconv.FormatUint(imageInfo.Id, 10),
		Id:      imageInfo.SelfLink,
		Name:    imageInfo.Name,
		GuestOS: imageInfo.Family,
		Status:  imageInfo.Status,
		KeyValueList: []irs.KeyValue{
			{"SourceType", imageInfo.SourceType},
			{"SelfLink", imageInfo.SelfLink},
			{"GuestOsFeature", imageInfo.GuestOsFeatures[0].Type},
			{"Family", imageInfo.Family},
			{"DiskSizeGb", strconv.FormatInt(imageInfo.DiskSizeGb, 10)},
		},
	}

	return imageList

}
