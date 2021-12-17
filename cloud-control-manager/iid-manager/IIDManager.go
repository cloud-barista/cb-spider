// IID(Integrated ID) Manager of CB-Spider.
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
	"github.com/rs/xid"
	"regexp"

	"github.com/sirupsen/logrus"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/cloud-barista/cb-store/config"
	ccim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"
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

func (iidRWLock *IIDRWLOCK)IsExistIID(iidGroup IIDGroup, connectionName string, resourceType string, iId resources.IID) (bool, error) {
        cblog.Debug("check the IID.NameId:" + iId.NameId + " existence")

iidRWLock.rwMutex.RLock()
defer iidRWLock.rwMutex.RUnlock()

	// escape: "/" => "%2F"
	iId.NameId = strings.ReplaceAll(iId.NameId, "/", "%2F")

        return isExist(iidGroup, connectionName, resourceType, iId.NameId)
}

func (iidRWLock *IIDRWLOCK)CreateIID(iidGroup IIDGroup, connectionName string, resourceType string, iId resources.IID) (*IIDInfo, error) {
	cblog.Debug("check the IID.NameId:" + iId.NameId + " existence")

iidRWLock.rwMutex.Lock()
defer iidRWLock.rwMutex.Unlock()

	// escape: "/" => "%2F"
	iId.NameId = strings.ReplaceAll(iId.NameId, "/", "%2F")

	ret, err := isExist(iidGroup, connectionName, resourceType, iId.NameId)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}
	if ret == true {
		return nil, fmt.Errorf(iId.NameId + " already exists!")
	}

	iidInfo, err2 := forceCreateIID(iidGroup, connectionName, resourceType, iId)

	// escape: "%2F" => "/"
	iidInfo.IId.NameId = strings.ReplaceAll(iidInfo.IId.NameId, "%2F", "/")

	return iidInfo, err2
}

func (iidRWLock *IIDRWLOCK)UpdateIID(iidGroup IIDGroup, connectionName string, resourceType string, iId resources.IID) (*IIDInfo, error) {
        cblog.Debug("check the IID.NameId:" + iId.NameId + " existence")

iidRWLock.rwMutex.Lock()
defer iidRWLock.rwMutex.Unlock()

	// escape: "/" => "%2F"
	iId.NameId = strings.ReplaceAll(iId.NameId, "/", "%2F")

        ret, err := isExist(iidGroup, connectionName, resourceType, iId.NameId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }
        if ret == false {
                return nil, fmt.Errorf(iId.NameId + " does not exists!")
        }

        iidInfo, err2 := forceCreateIID(iidGroup, connectionName, resourceType, iId)

        // escape: "%2F" => "/"
        iidInfo.IId.NameId = strings.ReplaceAll(iidInfo.IId.NameId, "%2F", "/")

        return iidInfo, err2
}

// 1. check params
// 2. check pre-existing id
// 3. insert new IIDInfo into cb-store
func forceCreateIID(iidGroup IIDGroup, connectionName string, resourceType string, iId resources.IID) (*IIDInfo, error) {
	cblog.Info("call CreateIID()")

	cblog.Debug("check params")
	err := checkParams(connectionName, resourceType, &iId)
	if err != nil {
		return nil, err
	
	}

	cblog.Debug("insert metainfo into store")
        err = insertInfo(iidGroup, connectionName, resourceType, iId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

	iidInfo := &IIDInfo{connectionName, resourceType, iId}
	return iidInfo, nil
}

func (iidRWLock *IIDRWLOCK)ListIID(iidGroup IIDGroup, connectionName string, resourceType string) ([]*IIDInfo, error) {
	cblog.Info("call ListIID()")

iidRWLock.rwMutex.RLock()
defer iidRWLock.rwMutex.RUnlock()
        iIDInfoList, err := listInfo(iidGroup, connectionName, resourceType)
        if err != nil {
                return nil, err
        }

        // escape: "%2F" => "/"
	for i, iidInfo := range iIDInfoList {
		iIDInfoList[i].IId.NameId = strings.ReplaceAll(iidInfo.IId.NameId, "%2F", "/")
	}

        return iIDInfoList, nil
}

func (iidRWLock *IIDRWLOCK)ListResourceType(iidGroup IIDGroup, connectionName string) ([]string, error) {
        cblog.Info("call ListResourceType()")

iidRWLock.rwMutex.RLock()
defer iidRWLock.rwMutex.RUnlock()
        resourceTypeList, err := listResourceType(iidGroup, connectionName)
        if err != nil {
                return nil, err
        }

        // escape: "%2F" => "/"
        for i, rsType := range resourceTypeList {
                resourceTypeList[i] = strings.ReplaceAll(rsType, "%2F", "/")
        }

        return resourceTypeList, nil
}


// 1. check params
// 2. get IIDInfo from cb-store
func (iidRWLock *IIDRWLOCK)GetIID(iidGroup IIDGroup, connectionName string, resourceType string, iId resources.IID) (*IIDInfo, error) {
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

	iidInfo, err := getInfo(iidGroup, connectionName, resourceType, iId.NameId)
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
func (iidRWLock *IIDRWLOCK)FindIID(iidGroup IIDGroup, connectionName string, resourceType string, keyword string) (*IIDInfo, error) {
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

        iIDInfoList, err := listInfo(iidGroup, connectionName, resourceType)
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
func (iidRWLock *IIDRWLOCK)GetIIDbySystemID(iidGroup IIDGroup, connectionName string, resourceType string, iId resources.IID) (*IIDInfo, error) {
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

        iidInfo, err := getIIDInfoByValue(iidGroup, connectionName, resourceType, iId.SystemId)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

        // escape: "%2F" => "/"
        iidInfo.IId.NameId = strings.ReplaceAll(iidInfo.IId.NameId, "%2F", "/")

        return iidInfo, err
}


func (iidRWLock *IIDRWLOCK)DeleteIID(iidGroup IIDGroup, connectionName string, resourceType string, iId resources.IID) (bool, error) {
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

        result, err := deleteInfo(iidGroup, connectionName, resourceType, iId.NameId)
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

//----------------

// generate Spider UUID(SP-XID)
func New(cloudConnectName string, uid string) (string, error) {
	guid := xid.New()

	cookedUID := cookUID(uid)
	// cblog.Info("UID: " + uid + " => cookedUID: " + cookedUID)

	spXID := cookedUID + "-" + guid.String()
	// cblog.Info("SP-XID: " + spXID)

	return convertDashOrUnderScore(cloudConnectName, spXID)
}

func convertDashOrUnderScore(cloudConnectName string, spXID string) (string, error) {
	cccInfo, err := ccim.GetConnectionConfig(cloudConnectName)
        if err != nil {
                return "", err
        }

	var convertedSpXID string
	// Tencent use '_'
	if cccInfo.ProviderName == "TENCENT" {
		convertedSpXID = strings.ReplaceAll(spXID, "-", "_")
	} else { // other CSP use '-'
		convertedSpXID = strings.ReplaceAll(spXID, "_", "-")
	}

	// AWS SecurityGroup: User can not use 'sg-*' format
	convertedSpXID = strings.ReplaceAll(spXID, "sg-", "sg")

	return convertedSpXID, nil
}

func cookUID(orgUID string) string {
        runes := []rune(orgUID)
        filteredUID := []byte{}
        for _, char := range runes {
                // (1) Max length is '9'
                if len(filteredUID)==9 { // max length: 9
                        break
                }
                var matched bool = false
                var err error
                // (2) Check the first character is a lowercase string
                if len(filteredUID) == 0 {
                        matched, err = regexp.MatchString("[a-zA-Z]", string(char))
                // (3) Extract filteredUID([a-zA-Z0-9-_])
                } else {
                        matched, err = regexp.MatchString("[a-zA-Z0-9-_]", string(char))
                }
                if err != nil {
                        cblog.Error(err)
                }
                if matched {
                        //fmt.Printf("%s matches\n", string(char))
                        filteredUID = append(filteredUID, byte(char))
                }
        }

        // (4) Coverting UID into lowercase
        lowercaseUID := strings.ToLower(string(filteredUID))

        return lowercaseUID
}

