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
	"strings"
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
// (2) get repo digests id from pulling return
// (3) get all image summary from local repos
// (4) get image ID and OS Info from inspection
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
//	cblogger.Info(msg)

	// (2) get repo digests id from pulling return
	repoDigests, err := getRepoDigests(msg)
	if err != nil {
		cblogger.Error(err)
		return irs.ImageInfo{}, err
	}

	// (3) get all image summary from local repos
        images, err := imageHandler.Client.ImageList(imageHandler.Context, types.ImageListOptions{})
        if err != nil {
                cblogger.Error(err)
                return irs.ImageInfo{}, err
        }

	// (4) get image ID and OS Info from inspection
        for _, image := range images {
		if strings.Contains(image.RepoDigests[0], repoDigests) {
			imageReqInfo.IId.SystemId = image.ID
			// (3) inspect image info for OS info
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

func getRepoDigests(msg string) (string, error) {

	/*---------- msg example
	{"status":"Pulling from panubo/sshd","id":"latest"}
	{"status":"Digest: sha256:b260ab0136c734d80ef643387af0eeb807deb7e1f0a85cb432c7f310eca3bb83"}
	{"status":"Status: Image is up to date for panubo/sshd:latest"}
	------------*/
	strList := strings.Split(msg, "\n")	

	for _, str := range strList {
		if strings.Contains(str, "sha256") {
			tmpList := strings.Split(str, ":")
			str1 := strings.Trim(tmpList[2], " ") // sha256
			runes := []rune(tmpList[3])  // b260ab0136c734d80ef643387af0eeb807deb7e1f0a85cb432c7f310eca3bb83"}
			str2 := string(runes[0:len(tmpList[3])-3]) // b260ab0136c734d80ef643387af0eeb807deb7e1f0a85cb432c7f310eca3bb83
			return str1+":"+str2, nil

		}
	}

	return "", fmt.Errorf("failed image pulling-" + msg)
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
		osName, err := getOSInfo(imageHandler, image.ID)
		if err != nil {
			cblogger.Error(err)
			return []*irs.ImageInfo{}, err
		}

                //listImages[i] = &irs.ImageInfo{irs.IID{"", image.ID}, osName, "", nil } 
		// To avoid empty validator, Using CSPID for NameID by powerkim, 2022.01.25.
                listImages[i] = &irs.ImageInfo{irs.IID{image.ID, image.ID}, osName, "", nil } 
        }

	return listImages, nil
}

func (imageHandler *DockerImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
        cblogger.Info("Docker Cloud Driver: called GetImage()!")

	// inspect image info for OS info
	osName, err := getOSInfo(imageHandler, imageIID.SystemId)
	if err != nil {
		cblogger.Error(err)
		return irs.ImageInfo{}, err
	}
	return irs.ImageInfo{imageIID, osName, "", nil}, nil
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

func (imageHandler *DockerImageHandler) CheckWindowsImage(imageIID irs.IID) (bool, error) {
	return false, fmt.Errorf("Does not support CheckWindowsImage() yet!!")
}

