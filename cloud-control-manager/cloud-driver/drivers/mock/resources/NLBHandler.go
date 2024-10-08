// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Mock Driver.
//
// by CB-Spider Team, 2020.10.

package resources

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/xid"

	cblog "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

var nlbInfoMap map[string][]*irs.NLBInfo

type MockNLBHandler struct {
	MockName string
}

func init() {
	// cblog is a global variable.
	nlbInfoMap = make(map[string][]*irs.NLBInfo)
}

var nlbMapLock = new(sync.RWMutex)

func (nlbHandler *MockNLBHandler) CreateNLB(nlbInfo irs.NLBInfo) (irs.NLBInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called CreateNLB()!")

	mockName := nlbHandler.MockName
	nlbInfo.IId.SystemId = nlbInfo.IId.NameId
	nlbInfo.VpcIID.SystemId = nlbInfo.VpcIID.NameId

	// insert NLBInfo into global Map
	nlbMapLock.Lock()
	defer nlbMapLock.Unlock()
	infoList, _ := nlbInfoMap[mockName]
	nlbInfo.CreatedTime = time.Now()
	nlbInfo.Listener.IP = "1.2.3.4"
	nlbInfo.Listener.DNSName = ""
	nlbInfo.Listener.CspID = nlbInfo.IId.NameId + "-Listener-" + xid.New().String()
	nlbInfo.VMGroup.CspID = nlbInfo.IId.NameId + "-VMGroup-" + xid.New().String()
	nlbInfo.HealthChecker.CspID = nlbInfo.IId.NameId + "-HealthChecker-" + xid.New().String()
	clonedInfo := CloneNLBInfo(nlbInfo)
	infoList = append(infoList, &clonedInfo)
	nlbInfoMap[mockName] = infoList

	return CloneNLBInfo(nlbInfo), nil
}

func CloneNLBInfoList(srcInfoList []*irs.NLBInfo) []*irs.NLBInfo {
	clonedInfoList := []*irs.NLBInfo{}
	for _, srcInfo := range srcInfoList {
		clonedInfo := CloneNLBInfo(*srcInfo)
		clonedInfoList = append(clonedInfoList, &clonedInfo)
	}
	return clonedInfoList
}

func CloneNLBInfo(srcInfo irs.NLBInfo) irs.NLBInfo {
	/*
		type NLBInfo struct {
			IId             IID     // {NameId, SystemId}
			VpcIID          IID     // {NameId, SystemId}

			Type            string  // PUBLIC(V) | INTERNAL
			Scope           string  // REGION(V) | GLOBAL

			//------ Frontend
			Listener        ListenerInfo

			//------ Backend
			VMGroup         VMGroupInfo
			HealthChecker   HealthCheckerInfo

			CreatedTime     time.Time
			KeyValueList []KeyValue
		}
	*/

	// clone NLBInfo
	clonedInfo := irs.NLBInfo{
		IId:           irs.IID{srcInfo.IId.NameId, srcInfo.IId.SystemId},
		VpcIID:        irs.IID{srcInfo.VpcIID.NameId, srcInfo.VpcIID.SystemId},
		Type:          srcInfo.Type,
		Scope:         srcInfo.Scope,
		Listener:      srcInfo.Listener,
		VMGroup:       srcInfo.VMGroup,
		HealthChecker: srcInfo.HealthChecker,
		CreatedTime:   srcInfo.CreatedTime,
		TagList:       srcInfo.TagList, // clone TagList
		KeyValueList:  srcInfo.KeyValueList,
	}

	return clonedInfo
}

func (nlbHandler *MockNLBHandler) ListNLB() ([]*irs.NLBInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListNLB()!")

	mockName := nlbHandler.MockName
	nlbMapLock.RLock()
	defer nlbMapLock.RUnlock()
	infoList, ok := nlbInfoMap[mockName]
	if !ok {
		return []*irs.NLBInfo{}, nil
	}

	return CloneNLBInfoList(infoList), nil
}

func (nlbHandler *MockNLBHandler) GetNLB(iid irs.IID) (irs.NLBInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called GetNLB()!")

	nlbMapLock.RLock()
	defer nlbMapLock.RUnlock()

	mockName := nlbHandler.MockName
	infoList, ok := nlbInfoMap[mockName]
	if !ok {
		return irs.NLBInfo{}, fmt.Errorf("%s NLB does not exist!!", iid.NameId)
	}

	for _, info := range infoList {
		if info.IId.NameId == iid.NameId {
			return CloneNLBInfo(*info), nil
		}
	}

	return irs.NLBInfo{}, fmt.Errorf("%s NLB does not exist!!", iid.NameId)
}

func (nlbHandler *MockNLBHandler) DeleteNLB(iid irs.IID) (bool, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called DeleteNLB()!")

	nlbMapLock.Lock()
	defer nlbMapLock.Unlock()

	mockName := nlbHandler.MockName
	infoList, ok := nlbInfoMap[mockName]
	if !ok {
		return false, fmt.Errorf("%s NLB does not exist!!", iid.NameId)
	}

	for idx, info := range infoList {
		if info.IId.SystemId == iid.SystemId {
			infoList = append(infoList[:idx], infoList[idx+1:]...)
			nlbInfoMap[mockName] = infoList
			return true, nil
		}
	}
	return false, nil
}

func (nlbHandler *MockNLBHandler) AddVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (irs.VMGroupInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called AddVMs()!")

	nlbMapLock.Lock()
	defer nlbMapLock.Unlock()

	mockName := nlbHandler.MockName
	infoList, ok := nlbInfoMap[mockName]
	if !ok {
		return irs.VMGroupInfo{}, fmt.Errorf("%s NLB does not exist!!", nlbIID.NameId)
	}

	// check if all input VMs exist
	for _, info := range infoList {
		if info.IId.NameId == nlbIID.NameId {
			for _, vmIID := range *vmIIDs {
				for _, vm := range *info.VMGroup.VMs {
					if vm.NameId == vmIID.NameId {
						errMSG := fmt.Sprintf("%s NLB already has this VM: %v!!", nlbIID.NameId, vmIID)
						errMSG += fmt.Sprintf(" #### %s NLB has %v!!", nlbIID.NameId, *info.VMGroup.VMs)
						return irs.VMGroupInfo{}, fmt.Errorf(errMSG)
					}
				}
			}
		}
	}

	// Add all VMs
	for _, info := range infoList {
		if info.IId.NameId == nlbIID.NameId {
			*info.VMGroup.VMs = append(*info.VMGroup.VMs, *vmIIDs...)
			return CloneNLBInfo(*info).VMGroup, nil
		}
	}

	return irs.VMGroupInfo{}, fmt.Errorf("%s NLB does not exist!!", nlbIID.NameId)
}

func (nlbHandler *MockNLBHandler) RemoveVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called RemoveVMs()!")

	nlbMapLock.Lock()
	defer nlbMapLock.Unlock()

	mockName := nlbHandler.MockName
	infoList, ok := nlbInfoMap[mockName]
	if !ok {
		return false, fmt.Errorf("%s NLB does not exist!!", nlbIID.NameId)
	}

	// check if all input VMs do not exist
	for _, info := range infoList {
		if info.IId.NameId == nlbIID.NameId {
			for _, vmIID := range *vmIIDs {
				existFlag := false
				for _, vm := range *info.VMGroup.VMs {
					if vm.NameId == vmIID.NameId {
						existFlag = true
					}
				}
				if !existFlag {
					errMSG := fmt.Sprintf("%s NLB does not have this VM: %v!!", nlbIID.NameId, vmIID)
					errMSG += fmt.Sprintf(" #### %s NLB has %v!!", nlbIID.NameId, *info.VMGroup.VMs)
					return false, fmt.Errorf(errMSG)
				}
			}
		}
	}

	for _, info := range infoList {
		if (*info).IId.NameId == nlbIID.NameId {
			for _, vmIID := range *vmIIDs {
				for idx, vm := range *info.VMGroup.VMs {
					if vm.NameId == vmIID.NameId {
						*info.VMGroup.VMs = removeVM(info.VMGroup.VMs, idx)
						break
					}
				}
			}
			break
		}
	}

	return true, nil
}

func removeVM(list *[]irs.IID, idx int) []irs.IID {
	return append((*list)[:idx], (*list)[idx+1:]...)
}

// ------ Frontend Control
func (nlbHandler *MockNLBHandler) ChangeListener(nlbIID irs.IID, listener irs.ListenerInfo) (irs.ListenerInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ChangeListener()!")

	nlbMapLock.RLock()
	defer nlbMapLock.RUnlock()

	mockName := nlbHandler.MockName
	infoList, ok := nlbInfoMap[mockName]
	if !ok {
		return irs.ListenerInfo{}, fmt.Errorf("%s NLB does not exist!!", nlbIID.NameId)
	}

	for _, info := range infoList {
		if info.IId.NameId == nlbIID.NameId {
			info.Listener.Protocol = listener.Protocol
			info.Listener.Port = listener.Port
			return CloneListenerInfo(info.Listener), nil
		}
	}

	return irs.ListenerInfo{}, fmt.Errorf("%s NLB does not exist!!", nlbIID.NameId)
}

func CloneListenerInfo(srcInfo irs.ListenerInfo) irs.ListenerInfo {
	/*
		type ListenerInfo struct {
			Protocol        string  // TCP|UDP
			IP              string  // Auto Generated and attached
			Port            string  // 1-65535
			DNSName         string  // Optional, Auto Generated and attached

			CspID           string  // Optional, May be Used by Driver.
			KeyValueList []KeyValue
		}
	*/

	clonedInfo := irs.ListenerInfo{
		Protocol:     srcInfo.Protocol,
		IP:           srcInfo.IP,
		Port:         srcInfo.Port,
		DNSName:      srcInfo.DNSName,
		CspID:        srcInfo.CspID,
		KeyValueList: srcInfo.KeyValueList,
	}

	return clonedInfo
}

// ------ Backend Control
func (nlbHandler *MockNLBHandler) ChangeVMGroupInfo(nlbIID irs.IID, vmGroup irs.VMGroupInfo) (irs.VMGroupInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ChangeVMGroupInfo()!")

	nlbMapLock.RLock()
	defer nlbMapLock.RUnlock()

	mockName := nlbHandler.MockName
	infoList, ok := nlbInfoMap[mockName]
	if !ok {
		return irs.VMGroupInfo{}, fmt.Errorf("%s NLB does not exist!!", nlbIID.NameId)
	}

	for _, info := range infoList {
		if info.IId.NameId == nlbIID.NameId {
			info.VMGroup.Protocol = vmGroup.Protocol
			info.VMGroup.Port = vmGroup.Port
			return CloneVMGroupInfo(info.VMGroup), nil
		}
	}

	return irs.VMGroupInfo{}, fmt.Errorf("%s NLB does not exist!!", nlbIID.NameId)
}

func CloneVMGroupInfo(srcInfo irs.VMGroupInfo) irs.VMGroupInfo {
	/*
		type VMGroupInfo struct {
			Protocol        string  // TCP|UDP
			Port            string  // 1-65535
			VMs             *[]IID

			CspID           string  // Optional, May be Used by Driver.
			KeyValueList []KeyValue
		}
	*/

	clonedInfo := irs.VMGroupInfo{
		Protocol:     srcInfo.Protocol,
		Port:         srcInfo.Port,
		VMs:          CloneVMs(srcInfo.VMs),
		CspID:        srcInfo.CspID,
		KeyValueList: srcInfo.KeyValueList,
	}

	return clonedInfo
}

func CloneVMs(srcInfo *[]irs.IID) *[]irs.IID {
	clonedList := []irs.IID{}
	for _, one := range *srcInfo {
		clonedInfo := irs.IID{
			NameId:   one.NameId,
			SystemId: one.SystemId,
		}
		clonedList = append(clonedList, clonedInfo)
	}
	return &clonedList
}

func (nlbHandler *MockNLBHandler) ChangeHealthCheckerInfo(nlbIID irs.IID, healthChecker irs.HealthCheckerInfo) (irs.HealthCheckerInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ChangeHealthCheckerInfo()!")

	nlbMapLock.RLock()
	defer nlbMapLock.RUnlock()

	mockName := nlbHandler.MockName
	infoList, ok := nlbInfoMap[mockName]
	if !ok {
		return irs.HealthCheckerInfo{}, fmt.Errorf("%s NLB does not exist!!", nlbIID.NameId)
	}

	for _, info := range infoList {
		if info.IId.NameId == nlbIID.NameId {
			info.HealthChecker.Protocol = healthChecker.Protocol
			info.HealthChecker.Port = healthChecker.Port
			info.HealthChecker.Interval = healthChecker.Interval
			info.HealthChecker.Timeout = healthChecker.Timeout
			info.HealthChecker.Threshold = healthChecker.Threshold
			return CloneHealthCheckerInfo(info.HealthChecker), nil
		}
	}

	return irs.HealthCheckerInfo{}, fmt.Errorf("%s NLB does not exist!!", nlbIID.NameId)
}

func CloneHealthCheckerInfo(srcInfo irs.HealthCheckerInfo) irs.HealthCheckerInfo {
	/*
		type HealthCheckerInfo struct {
			Protocol        string  // TCP|HTTP
			Port            string  // Listener Port or 1-65535
			Interval        int     // secs, Interval time between health checks.
			Timeout         int     // secs, Waiting time to decide an unhealthy VM when no response.
			Threshold       int     // num, The number of continuous health checks to change the VM status.

			CspID           string  // Optional, May be Used by Driver.
			KeyValueList []KeyValue
		}
	*/

	clonedInfo := irs.HealthCheckerInfo{
		Protocol:     srcInfo.Protocol,
		Port:         srcInfo.Port,
		Interval:     srcInfo.Interval,
		Timeout:      srcInfo.Timeout,
		Threshold:    srcInfo.Threshold,
		CspID:        srcInfo.CspID,
		KeyValueList: srcInfo.KeyValueList,
	}

	return clonedInfo
}

func (nlbHandler *MockNLBHandler) GetVMGroupHealthInfo(nlbIID irs.IID) (irs.HealthInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called GetVMGroupHealthInfo()!")

	nlbMapLock.RLock()
	defer nlbMapLock.RUnlock()

	mockName := nlbHandler.MockName
	infoList, ok := nlbInfoMap[mockName]
	if !ok {
		return irs.HealthInfo{}, fmt.Errorf("%s NLB does not exist!!", nlbIID.NameId)
	}

	healthInfo := irs.HealthInfo{&[]irs.IID{}, &[]irs.IID{}, &[]irs.IID{}}
	for _, info := range infoList {
		if info.IId.NameId == nlbIID.NameId {
			for idx, vm := range *info.VMGroup.VMs {
				*healthInfo.AllVMs = append(*healthInfo.AllVMs, vm)
				if (idx + 1) == len(*info.VMGroup.VMs) {
					*healthInfo.UnHealthyVMs = append(*healthInfo.UnHealthyVMs, vm)
				} else {
					*healthInfo.HealthyVMs = append(*healthInfo.HealthyVMs, vm)
				}
			}
			return healthInfo, nil
		}
	}

	return irs.HealthInfo{}, fmt.Errorf("%s NLB VMGroup does not have VMs!!", nlbIID.NameId)
}

func (NLBHandler *MockNLBHandler) ListIID() ([]*irs.IID, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListIID()!")

	mockName := NLBHandler.MockName
	nlbMapLock.RLock()
	defer nlbMapLock.RUnlock()
	infoList, ok := nlbInfoMap[mockName]
	if !ok {
		return []*irs.IID{}, nil
	}

	iidList := []*irs.IID{}
	for _, info := range infoList {
		iidList = append(iidList, &irs.IID{info.IId.NameId, info.IId.SystemId})
	}

	return iidList, nil
}
