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
        cblog "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"fmt"
)

var imgInfoMap map[string][]*irs.ImageInfo

type MockImageHandler struct {
	MockName      string
}

func init() {
        // cblog is a global variable.
	imgInfoMap = make(map[string][]*irs.ImageInfo)
}

// (1) create imageInfo object
// (2) insert ImageInfo into global Map
func (imageHandler *MockImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mock Driver: called CreateImage()!")

	mockName := imageHandler.MockName
	imageReqInfo.IId.SystemId = imageReqInfo.IId.NameId

	// (1) create imageInfo object
	imageInfo := irs.ImageInfo{irs.IID{imageReqInfo.IId.NameId, imageReqInfo.IId.SystemId}, "TestGuestOS", "TestStatus", nil}

	// (2) insert ImageInfo into global Map
	imgInfoList, _ := imgInfoMap[mockName]
	imgInfoList = append(imgInfoList, &imageInfo)
	imgInfoMap[mockName]=imgInfoList

	return imageInfo, nil
}

func (imageHandler *MockImageHandler) ListImage() ([]*irs.ImageInfo, error) {
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mock Driver: called ListImage()!")
	
	mockName := imageHandler.MockName
	imgInfoList, ok := imgInfoMap[mockName]
	if !ok {
		return []*irs.ImageInfo{}, nil
	}
	// cloning list of Image
	resultList := make([]*irs.ImageInfo, len(imgInfoList))
	copy(resultList, imgInfoList)
	return resultList, nil
}

func (imageHandler *MockImageHandler) GetImage(imageIID irs.IID) (irs.ImageInfo, error) {
        cblogger := cblog.GetLogger("CB-SPIDER")
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
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mock Driver: called DeleteImage()!")

        imgInfoList, err := imageHandler.ListImage()
        if err != nil {
                cblogger.Error(err)
                return false, err
        }

	mockName := imageHandler.MockName
        for idx, info := range imgInfoList {
                if(info.IId.NameId == imageIID.NameId) {
			imgInfoList = append(imgInfoList[:idx], imgInfoList[idx+1:]...)
			imgInfoMap[mockName]=imgInfoList
			return true, nil
                }
        }
	return false, nil
}
