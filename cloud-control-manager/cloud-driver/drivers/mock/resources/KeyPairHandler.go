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
        _ "github.com/sirupsen/logrus"
        cblog "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"fmt"
)

var keyPairInfoMap map[string][]*irs.KeyPairInfo

type MockKeyPairHandler struct {
	MockName      string
}

func init() {
        // cblog is a global variable.
	keyPairInfoMap = make(map[string][]*irs.KeyPairInfo)
}

// (1) create keyPairInfo object
// (2) insert keyPairInfo into global Map
func (keyPairHandler *MockKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mock Driver: called CreateKey()!")

	mockName := keyPairHandler.MockName
	keyPairReqInfo.IId.SystemId = keyPairReqInfo.IId.NameId

	// (1) create keyPairInfo object
	keyPairInfo := irs.KeyPairInfo{keyPairReqInfo.IId,
			"XXXXFingerprint", "XXXXPublicKey", "XXXXPrivateKey", "cb-user", nil}

	// (2) insert KeyPairInfo into global Map
	infoList, _ := keyPairInfoMap[mockName]
	infoList = append(infoList, &keyPairInfo)
	keyPairInfoMap[mockName]=infoList

	return keyPairInfo, nil
}

func (keyPairHandler *MockKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mock Driver: called ListKey()!")
	
	mockName := keyPairHandler.MockName
	infoList, ok := keyPairInfoMap[mockName]
	if !ok {
		return []*irs.KeyPairInfo{}, nil
	}
	// cloning list of KeyPair
	resultList := make([]*irs.KeyPairInfo, len(infoList))
	copy(resultList, infoList)
	return resultList, nil
}

func (keyPairHandler *MockKeyPairHandler) GetKey(iid irs.IID) (irs.KeyPairInfo, error) {
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mock Driver: called GetKey()!")

	infoList, err := keyPairHandler.ListKey()
	if err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	for _, info := range infoList {
		if((*info).IId.NameId == iid.NameId) {
			return *info, nil
		}
	}
	
	return irs.KeyPairInfo{}, fmt.Errorf("%s keypair does not exist!!")
}

func (keyPairHandler *MockKeyPairHandler) DeleteKey(iid irs.IID) (bool, error) {
        cblogger := cblog.GetLogger("CB-SPIDER")
        cblogger.Info("Mock Driver: called DeleteKey()!")

        infoList, err := keyPairHandler.ListKey()
        if err != nil {
                cblogger.Error(err)
                return false, err
        }

	mockName := keyPairHandler.MockName
        for idx, info := range infoList {
                if(info.IId.NameId == iid.NameId) {
			infoList = append(infoList[:idx], infoList[idx+1:]...)
			keyPairInfoMap[mockName]=infoList
			return true, nil
                }
        }
	return false, nil
}
