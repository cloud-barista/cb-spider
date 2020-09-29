// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Mock Driver.
//
// by CB-Spider Team, 2020.09.

package resources

import (
        "github.com/sirupsen/logrus"
        cblog "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"fmt"
)

var rsInfoMap map[string][]*irs.ImageInfo

type MockImageHandler struct {
	MockName      string
}

var cblogger *logrus.Logger

func init() {
        // cblog is a global variable.
        cblogger = cblog.GetLogger("CB-SPIDER")
	rsInfoMap = make(map[string][]*irs.ImageInfo)
}

// (1) create imageInfo object
// (2) insert ImageInfo into global Map
func (imageHandler *MockImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
        cblogger.Info("Mock Driver: called CreateImage()!")

	mockName := imageHandler.MockName
	imageReqInfo.IId.SystemId = imageReqInfo.IId.NameId

	// (1) create imageInfo object
	imageInfo := irs.ImageInfo{irs.IID{imageReqInfo.IId.NameId, imageReqInfo.IId.SystemId}, "TestGuestOS", "TestStatus", nil}

	// (2) insert ImageInfo into global Map
	imgInfoList, ok := rsInfoMap[mockName]
	if !ok {
		imgInfoList = make([]*irs.ImageInfo, 1)
		imgInfoList[0] = &imageInfo
		rsInfoMap[mockName]=imgInfoList
	}else {
		imgInfoList = append(imgInfoList, &imageInfo)
		rsInfoMap[mockName]=imgInfoList
	}

	return imageInfo, nil
}

func (imageHandler *MockImageHandler) ListImage() ([]*irs.ImageInfo, error) {
        cblogger.Info("Mock Driver: called ListImage()!")
	
	mockName := imageHandler.MockName
	imgInfoList, ok := rsInfoMap[mockName]
	if !ok {
		return []*irs.ImageInfo{}, nil
	}
	return imgInfoList, nil
}

func (imageHandler *MockImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
        cblogger.Info("Mock Driver: called GetImage()!")


	imgInfoList, err := imageHandler.ListImage()
	if err != nil {
		cblogger.Error(err)
		return irs.ImageInfo{}, err
	}

	for _, info := range imgInfoList {
		if((*info).IId.NameId == imageIID.NameId) {
			return *info, nil
		}
	}
	
	return irs.ImageInfo{}, fmt.Errorf("%s image does not exist!!")
}

func (imageHandler *MockImageHandler) DeleteImage(imageIID irs.IID) (bool, error) {
        cblogger.Info("Mock Driver: called DeleteImage()!")

        imgInfoList, err := imageHandler.ListImage()
        if err != nil {
                cblogger.Error(err)
                return false, err
        }

fmt.Printf("========= target: %s\n", imageIID.NameId)
fmt.Printf("========= before: %v\n", len(imgInfoList))

	mockName := imageHandler.MockName
        for idx, info := range imgInfoList {
fmt.Printf("A: %s, B: %s\n", info.IId.NameId, imageIID.NameId)
                if(info.IId.NameId == imageIID.NameId) {
			imgInfoList = append(imgInfoList[:idx], imgInfoList[idx+1:]...)
fmt.Printf("========= after: %v\n", len(imgInfoList))
			rsInfoMap[mockName]=imgInfoList
			return true, nil
                }
        }
	return false, nil
}
