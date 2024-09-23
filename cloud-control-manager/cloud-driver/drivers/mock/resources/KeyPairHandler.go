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
	"sync"

	cblog "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	_ "github.com/sirupsen/logrus"
)

var keyPairInfoMap map[string][]*irs.KeyPairInfo

type MockKeyPairHandler struct {
	MockName string
}

func init() {
	// cblog is a global variable.
	keyPairInfoMap = make(map[string][]*irs.KeyPairInfo)
}

var keyMapLock = new(sync.RWMutex)

// (1) create keyPairInfo object
// (2) insert keyPairInfo into global Map
func (keyPairHandler *MockKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called CreateKey()!")

	mockName := keyPairHandler.MockName
	keyPairReqInfo.IId.SystemId = keyPairReqInfo.IId.NameId

	// (1) create keyPairInfo object
	keyPairInfo := irs.KeyPairInfo{
		IId:          keyPairReqInfo.IId,
		Fingerprint:  "XXXXFingerprint",
		PublicKey:    "XXXXPublicKey",
		PrivateKey:   "XXXXPrivateKey",
		VMUserID:     "cb-user",
		TagList:      keyPairReqInfo.TagList,
		KeyValueList: nil,
	}

	// (2) insert KeyPairInfo into global Map
	keyMapLock.Lock()
	defer keyMapLock.Unlock()
	infoList, _ := keyPairInfoMap[mockName]
	infoList = append(infoList, &keyPairInfo)
	keyPairInfoMap[mockName] = infoList

	return CloneKeyPairInfo(keyPairInfo), nil
}

func CloneKeyPairInfoList(srcInfoList []*irs.KeyPairInfo) []*irs.KeyPairInfo {
	clonedInfoList := []*irs.KeyPairInfo{}
	for _, srcInfo := range srcInfoList {
		clonedInfo := CloneKeyPairInfo(*srcInfo)
		clonedInfoList = append(clonedInfoList, &clonedInfo)
	}
	return clonedInfoList
}

func CloneKeyPairInfo(srcInfo irs.KeyPairInfo) irs.KeyPairInfo {
	/*
		type KeyPairInfo struct {
			IId   IID       // {NameId, SystemId}
			Fingerprint string
			PublicKey   string
			PrivateKey  string
			VMUserID      string

			TagList      []KeyValue
			KeyValueList []KeyValue
		}
	*/

	// clone KeyPairInfo
	clonedInfo := irs.KeyPairInfo{
		IId:          irs.IID{srcInfo.IId.NameId, srcInfo.IId.SystemId},
		Fingerprint:  srcInfo.Fingerprint,
		PublicKey:    srcInfo.PublicKey,
		PrivateKey:   srcInfo.PrivateKey,
		VMUserID:     srcInfo.VMUserID,
		TagList:      srcInfo.TagList, // clone TagList
		KeyValueList: srcInfo.KeyValueList,
	}

	return clonedInfo
}

func (keyPairHandler *MockKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListKey()!")

	mockName := keyPairHandler.MockName
	keyMapLock.RLock()
	defer keyMapLock.RUnlock()
	infoList, ok := keyPairInfoMap[mockName]
	if !ok {
		return []*irs.KeyPairInfo{}, nil
	}
	// cloning list of KeyPair
	return CloneKeyPairInfoList(infoList), nil
}

func (keyPairHandler *MockKeyPairHandler) GetKey(iid irs.IID) (irs.KeyPairInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called GetKey()!")

	mockName := keyPairHandler.MockName
	keyMapLock.RLock()
	defer keyMapLock.RUnlock()
	infoList, ok := keyPairInfoMap[mockName]
	if !ok {
		return irs.KeyPairInfo{}, fmt.Errorf("%s Keypair does not exist!!", iid.NameId)
	}

	for _, info := range infoList {
		if info.IId.NameId == iid.NameId {
			return CloneKeyPairInfo(*info), nil
		}
	}

	return irs.KeyPairInfo{}, fmt.Errorf("%s Keypair does not exist!!", iid.NameId)
}

func (keyPairHandler *MockKeyPairHandler) DeleteKey(iid irs.IID) (bool, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called DeleteKey()!")

	mockName := keyPairHandler.MockName

	keyMapLock.Lock()
	defer keyMapLock.Unlock()

	infoList, ok := keyPairInfoMap[mockName]
	if !ok {
		return false, fmt.Errorf("%s Keypair does not exist!!", iid.NameId)
	}

	for idx, info := range infoList {
		if info.IId.SystemId == iid.SystemId {
			infoList = append(infoList[:idx], infoList[idx+1:]...)
			keyPairInfoMap[mockName] = infoList
			return true, nil
		}
	}
	return false, nil
}

func (keyPairHandler *MockKeyPairHandler) ListIID() ([]*irs.IID, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListIID()!")

	mockName := keyPairHandler.MockName
	keyMapLock.RLock()
	defer keyMapLock.RUnlock()
	infoList, ok := keyPairInfoMap[mockName]
	if !ok {
		return []*irs.IID{}, nil
	}

	iidList := []*irs.IID{}
	for _, info := range infoList {
		iidList = append(iidList, &info.IId)
	}
	return iidList, nil
}
