// Cloud Driver Info. Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.03.

package iidmanager

import (
	"fmt"
	"sync"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/cloud-barista/cb-store/config"
)

var cblog *logrus.Logger

func init() {
        cblog = config.Cblogger
}

//====================================================================
type IIDRWLOCK struct {
        rwMutex	sync.RWMutex // for global readwrite Locking
}
//====================================================================


//====================================================================
type IIDInfo struct {
        ConnectionName  string  // ex) "aws-seoul-config"
        ResourceType    string  // ex) "VM"
        IId             resources.IID  // ex) {NameId, SystemId} = {"powerkim_vm_01", "i-0bc7123b7e5cbf79d"}
}
//====================================================================

func (iidRWLock *IIDRWLOCK)IsExistIID(connectionName string, resourceType string, iId resources.IID) (bool, error) {
        cblog.Debug("check the IID.NameId:" + iId.NameId + " existence")

iidRWLock.rwMutex.RLock()
defer iidRWLock.rwMutex.RUnlock()

	// escape: "/" => "%2F"
	iId.NameId = strings.ReplaceAll(iId.NameId, "/", "%2F")

        return isExist(connectionName, resourceType, iId.NameId)
}

func (iidRWLock *IIDRWLOCK)CreateIID(connectionName string, resourceType string, iId resources.IID) (*IIDInfo, error) {
	cblog.Debug("check the IID.NameId:" + iId.NameId + " existence")

iidRWLock.rwMutex.Lock()
defer iidRWLock.rwMutex.Unlock()

	// escape: "/" => "%2F"
	iId.NameId = strings.ReplaceAll(iId.NameId, "/", "%2F")

	ret, err := isExist(connectionName, resourceType, iId.NameId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if ret == true {
		return nil, fmt.Errorf(iId.NameId + " already exists!")
	}

	iidInfo, err2 := forceCreateIID(connectionName, resourceType, iId)

	// escape: "%2F" => "/"
	iidInfo.IId.NameId = strings.ReplaceAll(iidInfo.IId.NameId, "%2F", "/")

	return iidInfo, err2
}

func (iidRWLock *IIDRWLOCK)UpdateIID(connectionName string, resourceType string, iId resources.IID) (*IIDInfo, error) {
        cblog.Debug("check the IID.NameId:" + iId.NameId + " existence")

iidRWLock.rwMutex.Lock()
defer iidRWLock.rwMutex.Unlock()

	// escape: "/" => "%2F"
	iId.NameId = strings.ReplaceAll(iId.NameId, "/", "%2F")

        ret, err := isExist(connectionName, resourceType, iId.NameId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        if ret == false {
                return nil, fmt.Errorf(iId.NameId + " does not exists!")
        }

        iidInfo, err2 := forceCreateIID(connectionName, resourceType, iId)

        // escape: "%2F" => "/"
        iidInfo.IId.NameId = strings.ReplaceAll(iidInfo.IId.NameId, "%2F", "/")

        return iidInfo, err2
}

// 1. check params
// 2. check pre-existing id
// 3. insert new IIDInfo into cb-store
func forceCreateIID(connectionName string, resourceType string, iId resources.IID) (*IIDInfo, error) {
	cblog.Info("call CreateIID()")

	cblog.Debug("check params")
	err := checkParams(connectionName, resourceType, &iId)
	if err != nil {
		return nil, err
	
	}

	cblog.Debug("insert metainfo into store")
        err = insertInfo(connectionName, resourceType, iId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

	iidInfo := &IIDInfo{connectionName, resourceType, iId}
	return iidInfo, nil
}

func (iidRWLock *IIDRWLOCK)ListIID(connectionName string, resourceType string) ([]*IIDInfo, error) {
	cblog.Info("call ListIID()")

iidRWLock.rwMutex.RLock()
defer iidRWLock.rwMutex.RUnlock()
        iIDInfoList, err := listInfo(connectionName, resourceType)
        if err != nil {
                return nil, err
        }

        // escape: "%2F" => "/"
	for i, iidInfo := range iIDInfoList {
		iIDInfoList[i].IId.NameId = strings.ReplaceAll(iidInfo.IId.NameId, "%2F", "/")
	}

        return iIDInfoList, nil
}

// 1. check params
// 2. get IIDInfo from cb-store
func (iidRWLock *IIDRWLOCK)GetIID(connectionName string, resourceType string, iId resources.IID) (*IIDInfo, error) {
	cblog.Info("call GetIID()")

        cblog.Debug("check params")
        err := checkParams(connectionName, resourceType, &iId)
        if err != nil {
                return nil, err

        }

iidRWLock.rwMutex.RLock()
defer iidRWLock.rwMutex.RUnlock()

	// escape: "/" => "%2F"
	iId.NameId = strings.ReplaceAll(iId.NameId, "/", "%2F")

	iidInfo, err := getInfo(connectionName, resourceType, iId.NameId)
	if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // escape: "%2F" => "/"
        iidInfo.IId.NameId = strings.ReplaceAll(iidInfo.IId.NameId, "%2F", "/")

	return iidInfo, err
}

// 1. check params
// 2. find IIDInfo from cb-store
func (iidRWLock *IIDRWLOCK)FindIID(connectionName string, resourceType string, keyword string) (*IIDInfo, error) {
        cblog.Info("call FindIID()")

        cblog.Debug("check params")
        err := checkParamsKeyword(connectionName, resourceType, &keyword)
        if err != nil {
                return nil, err

        }

iidRWLock.rwMutex.RLock()
defer iidRWLock.rwMutex.RUnlock()

	// escape: "/" => "%2F"
	keyword = strings.ReplaceAll(keyword, "/", "%2F")

        iIDInfoList, err := listInfo(connectionName, resourceType)
        if err != nil {
                return nil, err
        }
	for _, iidInfo := range iIDInfoList {
		if strings.Contains(iidInfo.IId.NameId, keyword) {
			// escape: "%2F" => "/"
			iidInfo.IId.NameId = strings.ReplaceAll(iidInfo.IId.NameId, "%2F", "/")
			return iidInfo, nil
		}
	}
        return &IIDInfo{}, fmt.Errorf("[" + connectionName + ":" + resourceType +  ":" + keyword + "] does not exist!")
}

// 1. check params
// 2. get IIDInfo from cb-store
func (iidRWLock *IIDRWLOCK)GetIIDbySystemID(connectionName string, resourceType string, iId resources.IID) (*IIDInfo, error) {
        cblog.Info("call GetIIDbySystemID()")

        cblog.Debug("check params")
        err := checkParamsSystemId(connectionName, resourceType, &iId)
        if err != nil {
                return nil, err
        }

iidRWLock.rwMutex.RLock()
defer iidRWLock.rwMutex.RUnlock()

	// escape: "/" => "%2F"
	iId.NameId = strings.ReplaceAll(iId.NameId, "/", "%2F")

        iidInfo, err := getInfoByValue(connectionName, resourceType, iId.SystemId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // escape: "%2F" => "/"
        iidInfo.IId.NameId = strings.ReplaceAll(iidInfo.IId.NameId, "%2F", "/")

        return iidInfo, err
}


func (iidRWLock *IIDRWLOCK)DeleteIID(connectionName string, resourceType string, iId resources.IID) (bool, error) {
	cblog.Info("call DeleteIID()")


        cblog.Debug("check params")
        err := checkParams(connectionName, resourceType, &iId)
        if err != nil {
                return false, err

        }

iidRWLock.rwMutex.Lock()
defer iidRWLock.rwMutex.Unlock()

	// escape: "/" => "%2F"
	iId.NameId = strings.ReplaceAll(iId.NameId, "/", "%2F")

        result, err := deleteInfo(connectionName, resourceType, iId.NameId)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        return result, nil
}

//----------------

func checkParams(connectionName string, resourceType string, iId *resources.IID) error {
        if connectionName == "" {
                return fmt.Errorf("ConnectionName is empty!")
        }
        if resourceType == "" {
                return fmt.Errorf("ResourceType is empty!")
        }
        if iId == nil {
                return fmt.Errorf("IID is empty!")
        }
        if iId.NameId == "" {
                return fmt.Errorf("IID.NameId is empty!")
        }
	return nil
}

func checkParamsSystemId(connectionName string, resourceType string, iId *resources.IID) error {
        if connectionName == "" {
                return fmt.Errorf("ConnectionName is empty!")
        }
        if resourceType == "" {
                return fmt.Errorf("ResourceType is empty!")
        }
        if iId == nil {
                return fmt.Errorf("IID is empty!")
        }
        if iId.SystemId == "" {
                return fmt.Errorf("IID.SystemId is empty!")
        }
        return nil
}

func checkParamsKeyword(connectionName string, resourceType string, keyword *string) error {
        if connectionName == "" {
                return fmt.Errorf("ConnectionName is empty!")
        }
        if resourceType == "" {
                return fmt.Errorf("ResourceType is empty!")
        }
        if keyword == nil {
                return fmt.Errorf("Keyword is empty!")
        }
        return nil
}

