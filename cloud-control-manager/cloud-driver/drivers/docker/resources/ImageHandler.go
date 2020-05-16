// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Docker Driver.
//
// by CB-Spider Team, 2020.05.

package resources

import (
        "github.com/sirupsen/logrus"
        cblog "github.com/cloud-barista/cb-log"
	"context"
	"github.com/docker/docker/client"
	"github.com/docker/docker/api/types"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"bytes"
	"fmt"
)

type DockerImageHandler struct {
	Region        idrv.RegionInfo
	Context       context.Context
	Client        *client.Client
}

var cblogger *logrus.Logger

func init() {
        // cblog is a global variable.
        cblogger = cblog.GetLogger("CB-SPIDER")
}


// (1) pull from dockerhub
// (2) get image summary from local
// (3) inspect image info from local for OS info
func (imageHandler *DockerImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
        cblogger.Info("Docker Cloud Driver: called CreateImage()!")

	// (1) pull from dockerhub
        //ref) images, err := cli.ImagePull(context.Background(), "alpine:latest", types.ImagePullOptions{})
        out, err := imageHandler.Client.ImagePull(imageHandler.Context, imageReqInfo.IId.NameId, types.ImagePullOptions{})
        if err != nil {
                cblogger.Error(err)
                return irs.ImageInfo{}, err
        }
	defer out.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(out)
	msg := buf.String()
	cblogger.Info(msg)

	// (2) get image summary from local
        images, err := imageHandler.Client.ImageList(imageHandler.Context, types.ImageListOptions{})
        if err != nil {
                cblogger.Error(err)
                return irs.ImageInfo{}, err
        }

        for _, image := range images {
		if image.RepoTags[0] == imageReqInfo.IId.NameId {
			imageReqInfo.IId.SystemId = image.ID
			// (3) inspect image info from local for OS info
			osName, err := getOSInfo(imageHandler, image.ID)
			if err != nil {
				cblogger.Error(err)
				return irs.ImageInfo{}, err
			}
			return irs.ImageInfo{imageReqInfo.IId, osName, "", nil}, nil
		}
        }
	
	return irs.ImageInfo{}, fmt.Errorf("[Local Repos:" + imageReqInfo.IId.NameId + "] does not exist!")
}

func getOSInfo(imageHandler *DockerImageHandler, imageID string) (string, error) {
        inspec, _, err := imageHandler.Client.ImageInspectWithRaw(imageHandler.Context, imageID)
        if err != nil {
                cblogger.Error(err)
                return "", err
        }
	return inspec.Os + ":" + inspec.OsVersion + ":" + inspec.Architecture, nil
}

func (imageHandler *DockerImageHandler) ListImage() ([]*irs.ImageInfo, error) {
        cblogger.Info("Docker Cloud Driver: called ListImage()!")
	
	//ref) images, err := cli.ImageList(context.Background(), types.ImageListOptions{})
	images, err := imageHandler.Client.ImageList(imageHandler.Context, types.ImageListOptions{})
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	listImages := make([]*irs.ImageInfo, len(images))
        for i, image := range images {
		//fmt.Printf("\n\n======================\n%#v", image)
		//cblogger.Info(image.ID)
		//cblogger.Info(image.RepoTags[0])
                listImages[i] = &irs.ImageInfo{irs.IID{"", image.ID}, "", "", nil } 
        }

	return listImages, nil
}

// (1) get image summary from local
// (2) inspect image info from local for OS info
func (imageHandler *DockerImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
        cblogger.Info("Docker Cloud Driver: called GetImage()!")


        // (1) get image summary from local
        images, err := imageHandler.Client.ImageList(imageHandler.Context, types.ImageListOptions{})
        if err != nil {
                cblogger.Error(err)
                return irs.ImageInfo{}, err
        }

        for _, image := range images {
                if image.RepoTags[0] == imageIID.NameId {
                        imageIID.SystemId = image.ID
                        // (2) inspect image info from local for OS info
                        osName, err := getOSInfo(imageHandler, image.ID)
                        if err != nil {
                                cblogger.Error(err)
                                return irs.ImageInfo{}, err
                        }
                        return irs.ImageInfo{imageIID, osName, "", nil}, nil
                }
        }

        return irs.ImageInfo{}, fmt.Errorf("[Local Repos:" + imageIID.NameId + "] does not exist!")
}

func (imageHandler *DockerImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
        cblogger.Info("Docker Cloud Driver: called DeleteImage()!")


        response, err := imageHandler.Client.ImageRemove(imageHandler.Context, imageIID.SystemId, types.ImageRemoveOptions{})
        if err != nil {
                cblogger.Error(err)
                return false, err
        }

	fmt.Printf("\n\n=================\n %#v", response)

	return true, nil
}
