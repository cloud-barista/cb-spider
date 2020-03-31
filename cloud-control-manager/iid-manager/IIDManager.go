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

func (iidRWLock *IIDRWLOCK)CreateIID(connectionName string, resourceType string, iId resources.IID) (*IIDInfo, error) {
	cblog.Debug("check the IID.NameId:" + iId.NameId + " existence")

iidRWLock.rwMutex.Lock()
defer iidRWLock.rwMutex.Unlock()

	ret, err := isExist(connectionName, resourceType, iId.NameId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if ret == true {
		return nil, fmt.Errorf(iId.NameId + " already exists!")
	}

	return forceCreateIID(connectionName, resourceType, iId)
	//iidInfo, err2 := forceCreateIID(connectionName, resourceType, iId)
	//return iidInfo, err2
}

func (iidRWLock *IIDRWLOCK)UpdateIID(connectionName string, resourceType string, iId resources.IID) (*IIDInfo, error) {
        cblog.Debug("check the IID.NameId:" + iId.NameId + " existence")

iidRWLock.rwMutex.Lock()
defer iidRWLock.rwMutex.Unlock()

        ret, err := isExist(connectionName, resourceType, iId.NameId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        if ret == false {
                return nil, fmt.Errorf(iId.NameId + " does not exists!")
        }

	return forceCreateIID(connectionName, resourceType, iId)
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
	iidInfo, err := getInfo(connectionName, resourceType, iId.NameId)
	if err != nil {
                cblog.Error(err)
                return nil, err
        }

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

