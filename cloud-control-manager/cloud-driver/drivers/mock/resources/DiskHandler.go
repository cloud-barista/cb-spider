// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Mock Driver.
//
// by CB-Spider Team, 2022.08.

package resources

import (
	"fmt"
	"sync"
	"time"

	cblog "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	_ "github.com/sirupsen/logrus"
)

var diskInfoMap map[string][]*irs.DiskInfo

type MockDiskHandler struct {
	MockName string
}

func init() {
	// cblog is a global variable.
	diskInfoMap = make(map[string][]*irs.DiskInfo)
}

var diskMapLock = new(sync.RWMutex)

// (1) create diskInfo object
// (2) insert diskInfo into global Map
func (diskHandler *MockDiskHandler) CreateDisk(diskReqInfo irs.DiskInfo) (irs.DiskInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called CreateDisk()!")

	mockName := diskHandler.MockName
	diskReqInfo.IId.SystemId = diskReqInfo.IId.NameId
	diskReqInfo.Status = irs.DiskAvailable
	diskReqInfo.CreatedTime = time.Now()

	if diskReqInfo.DiskType == "default" || diskReqInfo.DiskType == "" {
		diskReqInfo.DiskType = "SSD"
	}
	if diskReqInfo.DiskSize == "default" || diskReqInfo.DiskSize == "" {
		diskReqInfo.DiskSize = "512"
	}

	// (2) insert DiskInfo into global Map
diskMapLock.Lock()
defer diskMapLock.Unlock()
	infoList, _ := diskInfoMap[mockName]
	infoList = append(infoList, &diskReqInfo)
	diskInfoMap[mockName] = infoList

	return CloneDiskInfo(diskReqInfo), nil
}

func CloneDiskInfoList(srcInfoList []*irs.DiskInfo) []*irs.DiskInfo {
        clonedInfoList := []*irs.DiskInfo{}
        for _, srcInfo := range srcInfoList {
                clonedInfo := CloneDiskInfo(*srcInfo)
                clonedInfoList = append(clonedInfoList, &clonedInfo)
        }
        return clonedInfoList
}

func CloneDiskInfo(srcInfo irs.DiskInfo) irs.DiskInfo {
        /*
		type DiskInfo struct {
			IId     IID     // {NameId, SystemId}

			DiskType string  // "", "SSD(gp2)", "Premium SSD", ...
			DiskSize string  // "", "default", "50", "1000"  # (GB)

			Status          DiskStatus      // DiskCreating | DiskAvailable | DiskAttached | DiskDeleting | DiskError
			OwnerVM         IID             // When the Status is DiskAttached

			CreatedTime     time.Time
			KeyValueList []KeyValue
		}
        */

        // clone DiskInfo
        clonedInfo := irs.DiskInfo{
                IId:       	irs.IID{srcInfo.IId.NameId, srcInfo.IId.SystemId},
		DiskType: 	srcInfo.DiskType,
		DiskSize: 	srcInfo.DiskSize, 
		Status: 	srcInfo.Status,
		OwnerVM: 	irs.IID{srcInfo.OwnerVM.NameId, srcInfo.OwnerVM.SystemId},
		CreatedTime: 	srcInfo.CreatedTime,
                KeyValueList:  	srcInfo.KeyValueList, // now, do not need cloning
        }

        return clonedInfo
}

func (diskHandler *MockDiskHandler) ListDisk() ([]*irs.DiskInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListDisk()!")

	mockName := diskHandler.MockName
diskMapLock.RLock()
defer diskMapLock.RUnlock()
	infoList, ok := diskInfoMap[mockName]
	if !ok {
		return []*irs.DiskInfo{}, nil
	}
	// cloning list of Disk
	return CloneDiskInfoList(infoList), nil
}

func (diskHandler *MockDiskHandler) GetDisk(iid irs.IID) (irs.DiskInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called GetDisk()!")

	mockName := diskHandler.MockName
diskMapLock.RLock()
defer diskMapLock.RUnlock()
        infoList, ok := diskInfoMap[mockName]
        if !ok {
		return irs.DiskInfo{}, fmt.Errorf("%s Disk does not exist!!", iid.NameId)
        }

	for _, info := range infoList {
		if (*info).IId.NameId == iid.NameId {
			return CloneDiskInfo(*info), nil
		}
	}

	return irs.DiskInfo{}, fmt.Errorf("%s Disk does not exist!!", iid.NameId)
}

func (diskHandler *MockDiskHandler) ChangeDiskSize(iid irs.IID, size string) (bool, error) {
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mock Driver: called ChangeDiskSize()!")

        mockName := diskHandler.MockName

diskMapLock.RLock()
defer diskMapLock.RUnlock()
        infoList, ok := diskInfoMap[mockName]
        if !ok {
                return false, fmt.Errorf("%s Disk does not exist!!", iid.NameId)
        }

        for _, info := range infoList {
                if (*info).IId.NameId == iid.NameId {
                        info.DiskSize = size
                        return true, nil
                }
        }

        return false, fmt.Errorf("%s Disk does not exist!!", iid.NameId)
}

func (diskHandler *MockDiskHandler) DeleteDisk(iid irs.IID) (bool, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called DeleteDisk()!")

        mockName := diskHandler.MockName

diskMapLock.Lock()
defer diskMapLock.Unlock()

        infoList, ok := diskInfoMap[mockName]
        if !ok {
                return false, fmt.Errorf("%s Disk does not exist!!", iid.NameId)
        }

	for idx, info := range infoList {
		if info.IId.SystemId == iid.SystemId {
			diskDetach(mockName, info.OwnerVM, iid)
			infoList = append(infoList[:idx], infoList[idx+1:]...)
			diskInfoMap[mockName] = infoList
			return true, nil
		}
	}
	return false, nil
}


func (diskHandler *MockDiskHandler) AttachDisk(diskIID irs.IID, ownerVM irs.IID) (irs.DiskInfo, error) {
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mock Driver: called AttachDisk()!")

        mockName := diskHandler.MockName

diskMapLock.RLock()
defer diskMapLock.RUnlock()
        infoList, ok := diskInfoMap[mockName]
        if !ok {
                return irs.DiskInfo{}, fmt.Errorf("%s Disk does not exist!!", diskIID.NameId)
        }

        for _, info := range infoList {
                if (*info).IId.NameId == diskIID.NameId {
			if info.Status == irs.DiskAttached {
				return irs.DiskInfo{}, fmt.Errorf("%s Disk is already Attached status!!", diskIID.NameId)
			}
			if info.Status != irs.DiskAvailable {
				return irs.DiskInfo{}, fmt.Errorf("%s Disk is not Available status!! It is %s status", diskIID.NameId, info.Status)
			}
			info.OwnerVM = ownerVM
			info.Status = irs.DiskAttached
			diskAttach(mockName, ownerVM, diskIID)
                        return CloneDiskInfo(*info), nil
                }
        }

        return irs.DiskInfo{}, fmt.Errorf("%s Disk does not exist!!", diskIID.NameId)
}

func (diskHandler *MockDiskHandler) DetachDisk(diskIID irs.IID, ownerVM irs.IID) (bool, error) {
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mock Driver: called DetachDisk()!")

        mockName := diskHandler.MockName

diskMapLock.RLock()
defer diskMapLock.RUnlock()
        infoList, ok := diskInfoMap[mockName]
        if !ok {
                return false, fmt.Errorf("%s Disk does not exist!!", diskIID.NameId)
        }

        for _, info := range infoList {
                if (*info).IId.NameId == diskIID.NameId {
			if info.Status != irs.DiskAttached {
				return false, fmt.Errorf("%s Disk is not Attached status!!. It is %s status", diskIID.NameId, info.Status)
			}
			diskDetach(mockName, ownerVM, diskIID)
                        info.Status = irs.DiskAvailable
                        info.OwnerVM = irs.IID{}
                        return true, nil
                }
        }

        return false, fmt.Errorf("%s Disk does not exist!!", diskIID.NameId)
}

func justAttachDisk(mockName string, diskIID irs.IID, ownerVM irs.IID) (bool, error) {
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mock Driver: called justAttachDisk()!")

diskMapLock.RLock()
defer diskMapLock.RUnlock()
        infoList, ok := diskInfoMap[mockName]
        if !ok {
                return false, fmt.Errorf("%s Disk does not exist!!", diskIID.NameId)
        }

        for _, info := range infoList {
                if (*info).IId.NameId == diskIID.NameId {
                        if info.Status != irs.DiskAvailable {
                                return false, fmt.Errorf("%s Disk is not Available status!!. It is %s status", diskIID.NameId, info.Status)                        }
                        info.Status = irs.DiskAttached
                        info.OwnerVM = ownerVM
                        return true, nil
                }
        }

        return false, fmt.Errorf("%s Disk does not exist!!", diskIID.NameId)
}

func justDetachDisk(mockName string, diskIID irs.IID, ownerVM irs.IID) (bool, error) {
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mock Driver: called justDetachDisk()!")

diskMapLock.RLock()
defer diskMapLock.RUnlock()
        infoList, ok := diskInfoMap[mockName]
        if !ok {
                return false, fmt.Errorf("%s Disk does not exist!!", diskIID.NameId)
        }

        for _, info := range infoList {
                if (*info).IId.NameId == diskIID.NameId {
                        if info.Status != irs.DiskAttached {
                                return false, fmt.Errorf("%s Disk is not Attached status!!. It is %s status", diskIID.NameId, info.Status)                        }
                        info.Status = irs.DiskAvailable
                        info.OwnerVM = irs.IID{}
                        return true, nil
                }
        }

        return false, fmt.Errorf("%s Disk does not exist!!", diskIID.NameId)
}

