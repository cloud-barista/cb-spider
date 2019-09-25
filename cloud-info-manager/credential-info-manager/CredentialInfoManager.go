// Cloud Credential Info. Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by powerkim@etri.re.kr, 2019.09.

package credentialinfomanager

import (
	"fmt"

	"github.com/sirupsen/logrus"
	icbs "github.com/cloud-barista/cb-store/interfaces"
	"github.com/cloud-barista/cb-store/config"
)

var cblog *logrus.Logger

func init() {
        cblog = config.Cblogger
}

//====================================================================
type CredentialInfo struct {
	CredentialName	string	// ex) "credential01"
	ProviderName	string	// ex) "AWS"
	KeyValueInfoList	[]icbs.KeyValue	// ex) { {ClientId, XXX}, 
						//	 {ClientSecret, XXX},
						//	 {TenantId, XXX},
						//	 {SubscriptionId, XXX} }
}
//====================================================================


func RegisterCredentialInfo(crdInfo CredentialInfo) (*CredentialInfo, error) {
        return RegisterCredential(crdInfo.CredentialName, crdInfo.ProviderName, crdInfo.KeyValueInfoList)
}


// 1. check params
// 2. insert them into cb-store
func RegisterCredential(credentialName string, providerName string, keyValueInfoList []icbs.KeyValue) (*CredentialInfo, error) {
	cblog.Info("call RegisterCredential()")

	cblog.Debug("check params")
	err := checkParams(credentialName, providerName, keyValueInfoList)
	if err != nil {
		return nil, err
	
	}

	cblog.Debug("insert metainfo into store")

	err = insertInfo(credentialName, providerName, keyValueInfoList)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	crdInfo := &CredentialInfo{credentialName, providerName, keyValueInfoList}
	return crdInfo, nil
}

func ListCredential() ([]*CredentialInfo, error) {
	cblog.Info("call ListCredential()")

        credentialInfoList, err := listInfo()
        if err != nil {
                return nil, err
        }

        return credentialInfoList, nil
}

// 1. check params
// 2. get CredentialInfo from cb-store
func GetCredential(credentialName string) (*CredentialInfo, error) {
	cblog.Info("call GetCredential()")

	if credentialName == "" {
                return nil, fmt.Errorf("credentialName is empty!")
        }
	
	crdInfo, err := getInfo(credentialName)
	if err != nil {
                cblog.Error(err)
                return nil, err
        }

	return crdInfo, err
}

func UnRegisterCredential(credentialName string) (bool, error) {
	cblog.Info("call UnRegisterCredential()")

        if credentialName == "" {
                return false, fmt.Errorf("credentialName is empty!")
        }

        result, err := deleteInfo(credentialName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        return result, nil
}

//----------------

func checkParams(credentialName string, providerName string, keyValueInfoList []icbs.KeyValue) error {
        if credentialName == "" {
                return fmt.Errorf("credentialName is empty!")
        }
        if providerName == "" {
                return fmt.Errorf("providerName is empty!")
        }
	for _, kv := range keyValueInfoList {
		if kv.Key == "" { // Value can be empty.
			return fmt.Errorf("Key is empty!")
		}
	}
	return nil
}

