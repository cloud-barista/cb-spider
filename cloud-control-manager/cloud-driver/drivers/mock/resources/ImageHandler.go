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
	"fmt"

	cblog "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

var imgInfoMap map[string][]*irs.ImageInfo

type MockImageHandler struct {
	MockName string
}

var PrepareImageInfoList []*irs.ImageInfo

func init() {
	// cblog is a global variable.
	imgInfoMap = make(map[string][]*irs.ImageInfo)
}

// Be called before using the User function.
// Called in MockDriver
func PrepareVMImage(mockName string) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called PrepareVMImage()!")

	if imgInfoMap[mockName] != nil {
		return
	}

	PrepareImageInfoList := []*irs.ImageInfo{
		{
			IId:            irs.IID{"mock-vmimage-01", "mock-vmimage-01"},
			GuestOS:        "TestGuestOS",
			Name:           "mock-vmimage-name-01",
			OSArchitecture: "x86_64",
			OSPlatform:     "Linux/UNIX",
			OSDistribution: "Ubuntu 18.04",
			DiskType:       "gp3",
			DiskSize:       "35",
			Status:         "Available",
			KeyValueList:   nil,
		},
		{
			IId:            irs.IID{"mock-vmimage-02", "mock-vmimage-02"},
			GuestOS:        "TestGuestOS",
			Name:           "mock-vmimage-name-02",
			OSArchitecture: "arm64",
			OSPlatform:     "Linux/UNIX",
			OSDistribution: "CentOS 8",
			DiskType:       "gp3",
			DiskSize:       "40",
			Status:         "Available",
			KeyValueList:   nil,
		},
		{
			IId:            irs.IID{"mock-vmimage-03", "mock-vmimage-03"},
			GuestOS:        "TestGuestOS",
			Name:           "mock-vmimage-name-03",
			OSArchitecture: "x86_64",
			OSPlatform:     "Windows",
			OSDistribution: "Windows Server 2019",
			DiskType:       "gp3",
			DiskSize:       "50",
			Status:         "Available",
			KeyValueList:   nil,
		},
		{
			IId:            irs.IID{"mock-vmimage-04", "mock-vmimage-04"},
			GuestOS:        "TestGuestOS",
			Name:           "mock-vmimage-name-04",
			OSArchitecture: "arm64",
			OSPlatform:     "Linux/UNIX",
			OSDistribution: "Ubuntu 22.04",
			DiskType:       "gp3",
			DiskSize:       "35",
			Status:         "Available",
			KeyValueList:   nil,
		},
		{
			IId:            irs.IID{"mock-vmimage-05", "mock-vmimage-05"},
			GuestOS:        "TestGuestOS",
			Name:           "mock-vmimage-name-05",
			OSArchitecture: "x86_64",
			OSPlatform:     "Linux/UNIX",
			OSDistribution: "Amazon Linux 2",
			DiskType:       "gp3",
			DiskSize:       "30",
			Status:         "Available",
			KeyValueList:   nil,
		},
	}

	imgInfoMap[mockName] = PrepareImageInfoList
}

// (1) create imageInfo object
// (2) insert ImageInfo into global Map
func (imageHandler *MockImageHandler) CreateImage(imageReqInfo irs.ImageReqInfo) (irs.ImageInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called CreateImage()!")

	mockName := imageHandler.MockName
	imageReqInfo.IId.SystemId = imageReqInfo.IId.NameId

	// (1) create imageInfo object
	imageInfo := irs.ImageInfo{
		IId:            irs.IID{imageReqInfo.IId.NameId, imageReqInfo.IId.SystemId},
		GuestOS:        "TestGuestOS",
		Name:           "test-image-name",
		OSArchitecture: "x86_64",
		OSPlatform:     "Linux/UNIX",
		OSDistribution: "Ubuntu 18.04",
		DiskType:       "gp3",
		DiskSize:       "35",
		Status:         "Available",
		KeyValueList:   nil,
	}

	// (2) insert ImageInfo into global Map
	imgInfoList, _ := imgInfoMap[mockName]
	imgInfoList = append(imgInfoList, &imageInfo)
	imgInfoMap[mockName] = imgInfoList

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
		if (*info).IId.NameId == imageIID.NameId {
			return *info, nil
		}
	}

	return irs.ImageInfo{}, fmt.Errorf("%s image does not exist!!", imageIID.NameId)
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
		if info.IId.SystemId == imageIID.SystemId {
			imgInfoList = append(imgInfoList[:idx], imgInfoList[idx+1:]...)
			imgInfoMap[mockName] = imgInfoList
			return true, nil
		}
	}
	return false, nil
}

func (imageHandler *MockImageHandler) CheckWindowsImage(imageIID irs.IID) (bool, error) {
	return false, fmt.Errorf("Does not support CheckWindowsImage() yet!!")
}
