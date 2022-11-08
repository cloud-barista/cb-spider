// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Mock Driver.
//
// by CB-Spider Team, 2022.09.

package resources

import (
	"fmt"
	"sync"
	"time"

	cblog "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	_ "github.com/sirupsen/logrus"
)

var myImageInfoMap map[string][]*irs.MyImageInfo

type MockMyImageHandler struct {
	MockName string
}

func init() {
	// cblog is a global variable.
	myImageInfoMap = make(map[string][]*irs.MyImageInfo)
}

var myImageMapLock = new(sync.RWMutex)

// (1) create myImageInfo object
// (2) insert myImageInfo into global Map
func (myImageHandler *MockMyImageHandler) SnapshotVM(myImageReqInfo irs.MyImageInfo) (irs.MyImageInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called SnapshotVM()!")

	mockName := myImageHandler.MockName
	myImageReqInfo.IId.SystemId = myImageReqInfo.IId.NameId
	myImageReqInfo.Status = irs.MyImageAvailable
	myImageReqInfo.CreatedTime = time.Now()

	// (2) insert MyImageInfo into global Map
myImageMapLock.Lock()
defer myImageMapLock.Unlock()
	infoList, _ := myImageInfoMap[mockName]
	infoList = append(infoList, &myImageReqInfo)
	myImageInfoMap[mockName] = infoList

	return CloneMyImageInfo(myImageReqInfo), nil
}

func CloneMyImageInfoList(srcInfoList []*irs.MyImageInfo) []*irs.MyImageInfo {
        clonedInfoList := []*irs.MyImageInfo{}
        for _, srcInfo := range srcInfoList {
                clonedInfo := CloneMyImageInfo(*srcInfo)
                clonedInfoList = append(clonedInfoList, &clonedInfo)
        }
        return clonedInfoList
}

func CloneMyImageInfo(srcInfo irs.MyImageInfo) irs.MyImageInfo {
        /*
		type MyImageInfo struct {
		        IId     IID     // {NameId, SystemId}

		        SourceVM IID

		        Status          MyImageStatus  // Available | Deleting

		        CreatedTime     time.Time
		        KeyValueList    []KeyValue
		}
        */

        // clone MyImageInfo
        clonedInfo := irs.MyImageInfo{
                IId:       	irs.IID{srcInfo.IId.NameId, srcInfo.IId.SystemId},
		SourceVM: 	irs.IID{srcInfo.SourceVM.NameId, srcInfo.SourceVM.SystemId},
		Status: 	srcInfo.Status,

		CreatedTime: 	srcInfo.CreatedTime,
                KeyValueList:  	srcInfo.KeyValueList, // now, do not need cloning
        }

        return clonedInfo
}

func (myImageHandler *MockMyImageHandler) ListMyImage() ([]*irs.MyImageInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListMyImage()!")

	mockName := myImageHandler.MockName
myImageMapLock.RLock()
defer myImageMapLock.RUnlock()
	infoList, ok := myImageInfoMap[mockName]
	if !ok {
		return []*irs.MyImageInfo{}, nil
	}
	// cloning list of MyImage
	return CloneMyImageInfoList(infoList), nil
}

func (myImageHandler *MockMyImageHandler) GetMyImage(iid irs.IID) (irs.MyImageInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called GetMyImage()!")

	mockName := myImageHandler.MockName
myImageMapLock.RLock()
defer myImageMapLock.RUnlock()
        infoList, ok := myImageInfoMap[mockName]
        if !ok {
		return irs.MyImageInfo{}, fmt.Errorf("%s MyImage does not exist!!", iid.NameId)
        }

	for _, info := range infoList {
		if (*info).IId.NameId == iid.NameId {
			return CloneMyImageInfo(*info), nil
		}
	}

	return irs.MyImageInfo{}, fmt.Errorf("%s MyImage does not exist!!", iid.NameId)
}

func (myImageHandler *MockMyImageHandler) DeleteMyImage(iid irs.IID) (bool, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called DeleteMyImage()!")

        mockName := myImageHandler.MockName

myImageMapLock.Lock()
defer myImageMapLock.Unlock()

        infoList, ok := myImageInfoMap[mockName]
        if !ok {
                return false, fmt.Errorf("%s MyImage does not exist!!", iid.NameId)
        }

	for idx, info := range infoList {
		if info.IId.SystemId == iid.SystemId {
			infoList = append(infoList[:idx], infoList[idx+1:]...)
			myImageInfoMap[mockName] = infoList
			return true, nil
		}
	}
	return false, nil
}

func (myImageHandler *MockMyImageHandler) CheckWindowsImage(iid irs.IID) (bool, error) {
	return false, fmt.Errorf("Does not support CheckWindowsImage() yet!!")
}

