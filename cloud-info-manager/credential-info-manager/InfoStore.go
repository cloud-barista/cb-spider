// CredentialInfo <-> CB-Store Handler for Cloud Credential Info. Manager.
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
	"github.com/cloud-barista/cb-store/utils"
	"github.com/cloud-barista/cb-store"
	icbs "github.com/cloud-barista/cb-store/interfaces"
)

var store icbs.Store

func init() {
        store = cbstore.GetStore()
}


/* //====================================================================
type CredentialInfo struct {
        CredentialName  string  // ex) "credential01"
        ProviderName    string  // ex) "AWS"
        KeyValueInfoList        []icbs.KeyValue      // ex) { {ClientId, XXX},
                                                         {ClientSecret, XXX},
                                                         {TenantId, XXX},
                                                         {SubscriptionId, XXX} }
}
*/ //====================================================================


// format
// /cloud-info-spaces/credentials/<CredentialName>/{ProviderName}/{key1} [value1]
// /cloud-info-spaces/credentials/<CredentialName>/{ProviderName}/{key2} [value2]
// ex)
// /cloud-info-spaces/credentials/aws_credential01/AWS/ClientId [value1]
// /cloud-info-spaces/credentials/aws_credential01/AWS/ClientSecret [value2]
// /cloud-info-spaces/credentials/aws_credential01/AWS/TenantId [value3]
// /cloud-info-spaces/credentials/aws_credential01/AWS/SubscriptionId [value4]



func insertInfo(credentialName string, providerName string, keyValueList []icbs.KeyValue) error {
	// ex)
	// /cloud-info-spaces/credentials/aws_credential01/AWS/ClientId [value1]
	// /cloud-info-spaces/credentials/aws_credential01/AWS/ClientSecret [value2]
	// /cloud-info-spaces/credentials/aws_credential01/AWS/TenantId [value3]
	// /cloud-info-spaces/credentials/aws_credential01/AWS/SubscriptionId [value4]

	format := "/cloud-info-spaces/credentials/" + credentialName + "/" + providerName
// @todo lock
	for _, kv := range keyValueList {
		key := format + "/" + kv.Key
		value := kv.Value

		err := store.Put(key, value)
		if err != nil {
			//cblog.Error(err)
			return err
		}
	}
// @todo lock
	return nil
}

// 1. get key-value list
// 2. create CredentialInfo List & return
func listInfo() ([]*CredentialInfo, error) {
        // ex)
        // /cloud-info-spaces/credentials/aws_credential01/AWS/ClientId [value1]
        // /cloud-info-spaces/credentials/aws_credential01/AWS/ClientSecret [value2]
        // /cloud-info-spaces/credentials/aws_credential01/AWS/TenantId [value3]
        // /cloud-info-spaces/credentials/aws_credential01/AWS/SubscriptionId [value4]

        key := "/cloud-info-spaces/credentials"
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return nil, err
        }

        var credentialInfoList []*CredentialInfo
	var inKeyValueList []icbs.KeyValue
	prevCredName := ""
	prevProviderName := ""
        for _, kv := range keyValueList {

		credName := utils.GetNodeValue(kv.Key, 3)
		providerName := utils.GetNodeValue(kv.Key, 4)

		if prevCredName=="" || credName == prevCredName {
			prevCredName = credName
			prevProviderName = providerName
			keyValue := icbs.KeyValue{utils.GetNodeValue(kv.Key, 5), kv.Value}
			inKeyValueList = append(inKeyValueList, keyValue)
		} else {
			// insert prev CredentialInfo
			crdInfo := &CredentialInfo{prevCredName, prevProviderName, inKeyValueList}
			credentialInfoList = append(credentialInfoList, crdInfo)

			prevCredName = credName
			prevProviderName = providerName
			inKeyValueList = nil
			keyValue := icbs.KeyValue{utils.GetNodeValue(kv.Key, 5), kv.Value}
			inKeyValueList = append(inKeyValueList, keyValue)
		}

        }

	if len(keyValueList) > 0 {
		// insert last CredentialInfo
		crdInfo := &CredentialInfo{prevCredName, prevProviderName, inKeyValueList}
		credentialInfoList = append(credentialInfoList, crdInfo)
	}

        return credentialInfoList, nil
}

// 1. get a key-value
// 2. create CredentialInfo & return
func getInfo(credentialName string) (*CredentialInfo, error) {
        // ex)
        // /cloud-info-spaces/credentials/aws_credential01/AWS/ClientId [value1]
        // /cloud-info-spaces/credentials/aws_credential01/AWS/ClientSecret [value2]
        // /cloud-info-spaces/credentials/aws_credential01/AWS/TenantId [value3]
        // /cloud-info-spaces/credentials/aws_credential01/AWS/SubscriptionId [value4]
	
	key := "/cloud-info-spaces/credentials/" + credentialName

        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return nil, err
        }

	if len(keyValueList) > 0 {
		var inKeyValueList []icbs.KeyValue

		providerName := utils.GetNodeValue(keyValueList[0].Key, 4)
		// get KeyValueList
		for _, kv := range keyValueList {
			keyValue := icbs.KeyValue{utils.GetNodeValue(kv.Key, 5), kv.Value}
			inKeyValueList = append(inKeyValueList, keyValue)
		}
		return &CredentialInfo{credentialName, providerName, inKeyValueList}, nil 
	}

        return nil, fmt.Errorf("no Results!")
}

// 1. get the original Key.
// 2. delete the key.
func deleteInfo(credentialName string) (bool, error) {
        // ex)
        // /cloud-info-spaces/credentials/aws_credential01/AWS/ClientId [value1]
        // /cloud-info-spaces/credentials/aws_credential01/AWS/ClientSecret [value2]
        // /cloud-info-spaces/credentials/aws_credential01/AWS/TenantId [value3]
        // /cloud-info-spaces/credentials/aws_credential01/AWS/SubscriptionId [value4]

	key := "/cloud-info-spaces/credentials/" + credentialName

// @todo lock-start
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return false, err
        }

	for _, kv := range keyValueList {
		err = store.Delete(kv.Key)
		if err != nil {
			return false, err
		}
	}
// @todo lock-end

        return true, nil
}

