// KeyPair <-> CB-Store Handler for Cloud Driver 
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2021.11.

package common

import (
	"fmt"

	"github.com/cloud-barista/cb-store/utils"
	"github.com/cloud-barista/cb-store"
	icbs "github.com/cloud-barista/cb-store/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

var store icbs.Store

const STORE_KEYPAIR_PREFIX string ="/driver-info-spaces/keypair"

func init() {
        store = cbstore.GetStore()
}

// format
// /driver-info-spaces/keypair/{Param1}/{Param2}/{Param3} [Param4]
// /driver-info-spaces/keypair/{ProviderName}/{HashString}/{KeyPairNameId} [privateKey]
// ex) /driver-info-spaces/keypair/CLOUDIT/c4240bec42480e764a4381c10c92e2ce/keypair-0-c6ncl9aba5o081np93og [private key]

func insertInfo(providerName string, hashString string, keyPairNameId string, privateKey string) error {
	key := STORE_KEYPAIR_PREFIX + "/" + providerName + "/" + hashString + "/" + keyPairNameId

	err := store.Put(key, privateKey)
        if err != nil {
                //cblog.Error(err)
		return err
        }
	return nil
}

// create KeyValue{KeyPairNameId, PrivateKey} List & return
func listInfo(providerName string, hashString string) ([]*irs.KeyValue, error) {
        key := STORE_KEYPAIR_PREFIX + "/" + providerName + "/" + hashString
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return nil, err
        }
        keyList := make([]*irs.KeyValue, len(keyValueList))
        for count, kv := range keyValueList {
                keyValue := &irs.KeyValue{
			Key : utils.GetNodeValue(kv.Key, 5), 	// KeyPairNameId
			Value : kv.Value, 			// private key
		}
                keyList[count] = keyValue
        }

        return keyList, nil
}

// create KeyValue{KeyPairNameId, PrivateKey} & return
func getInfo(providerName string, hashString string, keyPairNameId string) (*irs.KeyValue, error) {
	key := STORE_KEYPAIR_PREFIX + "/" + providerName + "/" + hashString + "/" + keyPairNameId

	// key is not the key of cb-store, so we have to use GetList()
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return nil, err
        }

        if len(keyValueList) < 1 {
                return nil, fmt.Errorf(keyPairNameId + ": does not exist!")
        }

        for _, kv := range keyValueList {
                // keyValueList should have ~/keypair or ~/keypair-01
                // so we have to check the sameness of keyPairNameId.
                if utils.GetNodeValue(kv.Key, 5) == keyPairNameId {
			keyValue := &irs.KeyValue{
				Key : utils.GetNodeValue(kv.Key, 5),    // KeyPairNameId
				Value : kv.Value,                       // private key
			}
			return keyValue, nil
                } // end of if
	} // end of for

        return nil, fmt.Errorf(keyPairNameId + ": does not exist!")
}

// 1. get the original Key.
// 2. delete the key.
func deleteInfo(providerName string, hashString string, keyPairNameId string) (bool, error) {
	key := STORE_KEYPAIR_PREFIX + "/" + providerName + "/" + hashString + "/" + keyPairNameId

	// key is not the key of cb-store, so we have to use GetList(
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return false, err
        }
	for _, kv := range keyValueList {
		// keyValueList should have ~/keypair or ~/keypair-01
                // so we have to check the sameness of keyPairNameId.
                if utils.GetNodeValue(kv.Key, 5) == keyPairNameId {
			err = store.Delete(kv.Key)
			if err != nil {
				return false, err
			}
			return true, nil
		}
        }

        return false, fmt.Errorf(keyPairNameId + ": does not exist!")
}

