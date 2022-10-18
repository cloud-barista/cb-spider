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
	"strings"

	"github.com/cloud-barista/cb-store/utils"
	"github.com/cloud-barista/cb-store"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	icbs "github.com/cloud-barista/cb-store/interfaces"
)

var store icbs.Store

func init() {
        store = cbstore.GetStore()
}

type IIDGroup string

const (
        IIDSGROUP IIDGroup = "iids"
        SUBNETGROUP IIDGroup = "iids:subnet"
        SGGROUP IIDGroup = "iids:sg"
        NLBGROUP IIDGroup = "iids:nlb"
        CLUSTERGROUP IIDGroup = "iids:cluster"
        NGGROUP IIDGroup = "iids:nodegroup"
)

/* //====================================================================
type IIDInfo struct {
        ConnectionName  string  // ex) "aws-seoul-config"
        ResourceType    string  // ex) "VM"
        IId             resources.IID   // ex) {NameId, SystemId} = {"powerkim_vm_01", "i-0bc7123b7e5cbf79d"}
}
*/ //====================================================================


// format
// /resource-info-spaces/iids/{ConnectionName}/{ResourceType}/<IID.NameId> [IID.SystemId]
        // Key: "/resource-info-spaces/iids/{ConnectionName}/{ResourceType}/<IID.NameId>"
        // Value: "[IID.SystemId]"
// ex) /resource-info-spaces/iids/aws-seoul-config/VM/powerkim_vm_01 [i-0bc7123b7e5cbf79d]
// iidGroup: iids(default) or iids:subnet or iids:sg
func insertInfo(iidGroup IIDGroup, connectionName string, resourceType string, iid resources.IID) error {
        // ex) /resource-info-spaces/iids/aws-seoul-config/VM/powerkim_vm_01 [i-0bc7123b7e5cbf79d]
        //key := "/resource-info-spaces/iids/" + connectionName + "/" + resourceType + "/" + iid.NameId
        key := "/resource-info-spaces/" + string(iidGroup) + "/" + connectionName + "/" + resourceType + "/" + iid.NameId
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
// iidGroup: iids(default) or iids:subnet or iids:sg
func listResourceType(iidGroup IIDGroup, connectionName string) ([]string, error) {
	// format) /resource-info-spaces/{iidGroup}/{connectionName}/{resourceType}/{resourceName} [{resourceID}]
	// ex)     /resource-info-spaces/iids:sg/mock-config01/vpc-01/sg-01 [s9e0ccc0fb04747fbb5cac8aabbd1e:s9e0ccc0fb04747fbb5cac8aabbd1e]

        key := "/resource-info-spaces/" + string(iidGroup) + "/" + connectionName
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return nil, err
        }

	resourceTypeList := []string{}
        for _, kv := range keyValueList {
		resourceTypeList = append(resourceTypeList, utils.GetNodeValue(kv.Key, 4))
        }

        return resourceTypeList, nil
}

// 1. get key-value list
// 2. create IIDInfo List & return
// iidGroup: iids(default) or iids:subnet or iids:sg
func listInfo(iidGroup IIDGroup, connectionName string, resourceType string) ([]*IIDInfo, error) {
        // ex) /resource-info-spaces/iids/aws-seoul-config/VM/powerkim_vm_01 [i-0bc7123b7e5cbf79d]

        //key := "/resource-info-spaces/iids/" + connectionName + "/" + resourceType
        key := "/resource-info-spaces/" + string(iidGroup) + "/" + connectionName + "/" + resourceType
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return nil, err
        }

	iidInfoList := []*IIDInfo{}
        for _, kv := range keyValueList {
		// Don't forget, GetList() based on prefix. ex) If SG, rsType: vpc-1, vpc-10
		if utils.GetNodeValue(kv.Key, 4) == resourceType {
			iidInfo := &IIDInfo{connectionName, resourceType, resources.IID{utils.GetNodeValue(kv.Key, 5), kv.Value} }
			iidInfoList = append(iidInfoList, iidInfo)
		}
        }

        return iidInfoList, nil
}

// 1. get a key-value
// 2. create IIDInfo & return
// iidGroup: iids(default) or iids:subnet or iids:sg
func getInfo(iidGroup IIDGroup, connectionName string, resourceType string, nameId string) (*IIDInfo, error) {
        // ex) /resource-info-spaces/iids/aws-seoul-config/VM/powerkim_vm_01 [i-0bc7123b7e5cbf79d]

        //key := "/resource-info-spaces/iids/" + connectionName + "/" + resourceType + "/" + nameId
        key := "/resource-info-spaces/" + string(iidGroup) + "/" + connectionName + "/" + resourceType + "/" + nameId

        // key is not the key of cb-store, so we have to use GetList()
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return nil, err
        }

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
// iidGroup: iids(default) or iids:subnet or iids:sg
func getIIDInfoByValue(iidGroup IIDGroup, connectionName string, resourceType string, systemId string) (*IIDInfo, error) {
        // ex) /resource-info-spaces/iids/aws-seoul-config/VM/??? [i-0bc7123b7e5cbf79d]

        // key := "/resource-info-spaces/iids/" + connectionName + "/" + resourceType
        key := "/resource-info-spaces/" + string(iidGroup) + "/" + connectionName + "/" + resourceType
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return nil, err
        }

        for _, kv := range keyValueList {
		// Don't forget, GetList() based on prefix. ex) If SG, rsType: vpc-1, vpc-10
		if utils.GetNodeValue(kv.Key, 4) == resourceType {
			// keyValueList should have ~/nameId/... or ~/nameId-01/...,
			// so we have to check the sameness of nameId.
			//if kv.Value == systemId {  changed, because kv.Value is spiderIID
			if strings.Contains(kv.Value, systemId) {
				iidInfo := &IIDInfo{connectionName, resourceType, resources.IID{utils.GetNodeValue(kv.Key, 5), kv.Value} } // kv.Value => sp-uuid:csp-iid
				return iidInfo, nil
			}
		}
        }

        return &IIDInfo{}, nil
}

// 1. get a key-value
// 2. return existence of  or not
// iidGroup: iids(default) or iids:subnet or iids:sg
func isExist(iidGroup IIDGroup, connectionName string, resourceType string, nameId string) (bool, error) {
        // ex) /resource-info-spaces/iids/aws-seoul-config/VM/powerkim_vm_01 [i-0bc7123b7e5cbf79d]


        //key := "/resource-info-spaces/iids/" + connectionName + "/" + resourceType + "/" + nameId
        key := "/resource-info-spaces/" + string(iidGroup) + "/" + connectionName + "/" + resourceType + "/" + nameId


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
// iidGroup: iids(default) or iids:subnet or iids:sg
func deleteInfo(iidGroup IIDGroup, connectionName string, resourceType string, nameId string) (bool, error) {
        // ex) /resource-info-spaces/iids/aws-seoul-config/VM/powerkim_vm_01 [i-0bc7123b7e5cbf79d]


        //key := "/resource-info-spaces/iids/" + connectionName + "/" + resourceType + "/" + nameId
        key := "/resource-info-spaces/" + string(iidGroup) + "/" + connectionName + "/" + resourceType + "/" + nameId

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
