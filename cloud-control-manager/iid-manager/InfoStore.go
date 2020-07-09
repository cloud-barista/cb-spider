// CloudDriverInfo <-> CB-Store Handler for Cloud Driver Info. Manager.
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

	"github.com/cloud-barista/cb-store/utils"
	"github.com/cloud-barista/cb-store"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	icbs "github.com/cloud-barista/cb-store/interfaces"
)

var store icbs.Store

func init() {
        store = cbstore.GetStore()
}


/* //====================================================================
type IIDInfo struct {
	ConnectionName	string	// ex) "aws-seoul-config"
	ResourceType	string	// ex) "VM"
	IId		resources.IID	// ex) {NameId, SystemId} = {"powerkim_vm_01", "i-0bc7123b7e5cbf79d"}
}
*/ //====================================================================


// format
// /resource-info-spaces/iids/{ConnectionName}/{ResourceType}/<IID.NameId> [IID.SystemId]
	// Key: "/resource-info-spaces/iids/{ConnectionName}/{ResourceType}/<IID.NameId>"
	// Value: "[IID.SystemId]"
// ex) /resource-info-spaces/iids/aws-seoul-config/VM/powerkim_vm_01 [i-0bc7123b7e5cbf79d]

func insertInfo(connectionName string, resourceType string, iid resources.IID) error {
	// ex) /resource-info-spaces/iids/aws-seoul-config/VM/powerkim_vm_01 [i-0bc7123b7e5cbf79d]

	key := "/resource-info-spaces/iids/" + connectionName + "/" + resourceType + "/" + iid.NameId
	value := iid.SystemId

	err := store.Put(key, value)
        if err != nil {
                //cblog.Error(err)
		return err
        }
	return nil
}

// 1. get key-value list
// 2. create IIDInfo List & return
func listInfo(connectionName string, resourceType string) ([]*IIDInfo, error) {
        // ex) /resource-info-spaces/iids/aws-seoul-config/VM/powerkim_vm_01 [i-0bc7123b7e5cbf79d]

        key := "/resource-info-spaces/iids/" + connectionName + "/" + resourceType
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return nil, err
        }

        iidInfoList := make([]*IIDInfo, len(keyValueList))
        for count, kv := range keyValueList {
                iidInfo := &IIDInfo{connectionName, resourceType, resources.IID{utils.GetNodeValue(kv.Key, 5), kv.Value} }
                iidInfoList[count] = iidInfo
        }

        return iidInfoList, nil
}

// 1. get a key-value
// 2. create IIDInfo & return
func getInfo(connectionName string, resourceType string, nameId string) (*IIDInfo, error) {
        // ex) /resource-info-spaces/iids/aws-seoul-config/VM/powerkim_vm_01 [i-0bc7123b7e5cbf79d]

	key := "/resource-info-spaces/iids/" + connectionName + "/" + resourceType + "/" + nameId

	// key is not the key of cb-store, so we have to use GetList()
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return nil, err
        }

//	if len(keyValueList) < 1 {
//		return nil, fmt.Errorf("[" + connectionName + ":" + resourceType +  ":" + nameId + "] does not exist!")
//	}

        for _, kv := range keyValueList {
		// keyValueList should have ~/nameId/... or ~/nameId-01/..., 
		// so we have to check the sameness of nameId.
                if utils.GetNodeValue(kv.Key, 5) == nameId {
			iidInfo := &IIDInfo{connectionName, resourceType, resources.IID{nameId, kv.Value} }
			return iidInfo, nil
                }
        }

        return nil, fmt.Errorf("[" + connectionName + ":" + resourceType +  ":" + nameId + "] does not exist!")
}

// 1. get list
// 2. find keyvalue by value
// 2. create IIDInfo & return
func getInfoByValue(connectionName string, resourceType string, systemId string) (*IIDInfo, error) {
        // ex) /resource-info-spaces/iids/aws-seoul-config/VM/??? [i-0bc7123b7e5cbf79d]

        key := "/resource-info-spaces/iids/" + connectionName + "/" + resourceType
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return nil, err
        }

        for _, kv := range keyValueList {
                // keyValueList should have ~/nameId/... or ~/nameId-01/...,
                // so we have to check the sameness of nameId.
                if kv.Value == systemId {
                        iidInfo := &IIDInfo{connectionName, resourceType, resources.IID{utils.GetNodeValue(kv.Key, 5), systemId} }
                        return iidInfo, nil
                }
        }

        return &IIDInfo{}, nil
}

// 1. get a key-value
// 2. return existence of  or not
func isExist(connectionName string, resourceType string, nameId string) (bool, error) {
        // ex) /resource-info-spaces/iids/aws-seoul-config/VM/powerkim_vm_01 [i-0bc7123b7e5cbf79d]


        key := "/resource-info-spaces/iids/" + connectionName + "/" + resourceType + "/" + nameId


        // key is not the key of cb-store, so we have to use GetList()
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return false, err
        }

        for _, kv := range keyValueList {
                // keyValueList should have ~/nameId/... or ~/nameId-01/...,
                // so we have to check the sameness of nameId.
                if utils.GetNodeValue(kv.Key, 5) == nameId {
                        return true, nil
                }
        }

        return false, nil
}

// 1. get the original Key.
// 2. delete the key.
func deleteInfo(connectionName string, resourceType string, nameId string) (bool, error) {
        // ex) /resource-info-spaces/iids/aws-seoul-config/VM/powerkim_vm_01 [i-0bc7123b7e5cbf79d]


        key := "/resource-info-spaces/iids/" + connectionName + "/" + resourceType + "/" + nameId

	// key is not the key of cb-store, so we have to use GetList()
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return false, err
        }
        for _, kv := range keyValueList {
		// keyValueList should have ~/nameId/... or ~/nameId-01/..., 
		// so we have to check the sameness of nameId.
		if utils.GetNodeValue(kv.Key, 5) == nameId {
			err = store.Delete(kv.Key)
			if err != nil {
				return false, err
			}
			return true, nil
		}
	}

        return false, fmt.Errorf("[" + connectionName + ":" + resourceType +  ":" + nameId + "] does not exist!")
}

