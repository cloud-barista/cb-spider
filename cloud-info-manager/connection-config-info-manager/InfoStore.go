// ConnectionConfigInfo <-> CB-Store Handler for Cloud ConnectionConfig Info. Manager.
// Cloud ConnectionConfig Info. Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by powerkim@etri.re.kr, 2019.09.

package connectionconfiginfomanager

import (
	"github.com/cloud-barista/cb-store/utils"
	"github.com/cloud-barista/cb-store"
	icbs "github.com/cloud-barista/cb-store/interfaces"
)

var store icbs.Store

func init() {
        store = cbstore.GetStore()
}


/* //====================================================================
type ConnectionConfigInfo struct {
        ConfigName      string  // ex) "config01"
        ProviderName    string  // ex) "AWS"
        DriverName      string  // ex) "AWS-Test-Driver-V0.5"
        CredentialName  string  // ex) "credential01"
        RegionName      string  // ex) "region01"
}
*/ //====================================================================


// format
// /cloud-info-spaces/connection-configs/<ID>/{Param1}/{Param2}/{Param3}/{Param4} []
// /cloud-info-spaces/connection-configs/<ConfigName>/{ProviderName}/{DriverName}/{CredentialName}/{RegionName} []
// ex) /cloud-info-spaces/connection-configs/config01/AWS/AWS-Test-Driver-V0.5/credential01/region01

func insertInfo(configName string, providerName string, driverName string, credentialName string, regionName string) error {
	// ex) /cloud-info-spaces/connection-configs/config01/AWS/AWS-Test-Driver-V0.5/credential01/region01

	key := "/cloud-info-spaces/connection-configs/" + configName + "/" + providerName + "/" +
		driverName + "/" + credentialName + "/" + regionName

	var value string

	err := store.Put(key, value)
        if err != nil {
                //cblog.Error(err)
		return err
        }
	return nil
}

// 1. get key-value list
// 2. create ConnectionConfigInfo List & return
func listInfo() ([]*ConnectionConfigInfo, error) {
        // ex) /cloud-info-spaces/connection-configs/config01/AWS/AWS-Test-Driver-V0.5/credential01/region01

        key := "/cloud-info-spaces/connection-configs"
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return nil, err
        }

        configInfoList := make([]*ConnectionConfigInfo, len(keyValueList))
        for count, kv := range keyValueList {
                cncInfo := &ConnectionConfigInfo{utils.GetNodeValue(kv.Key, 3), utils.GetNodeValue(kv.Key, 4),
						utils.GetNodeValue(kv.Key, 5), utils.GetNodeValue(kv.Key, 6), 
						utils.GetNodeValue(kv.Key, 7),
						}
                configInfoList[count] = cncInfo
        }

        return configInfoList, nil
}

// 1. get a key-value
// 2. create ConnectionConfigInfo & return
func getInfo(configName string) (*ConnectionConfigInfo, error) {
        // ex) /cloud-info-spaces/connection-configs/config01/AWS/AWS-Test-Driver-V0.5/credential01/region01
	
	key := "/cloud-info-spaces/connection-configs/" + configName

        kv, err := store.Get(key)
        if err != nil {
                return nil, err
        }

	cncInfo := &ConnectionConfigInfo{utils.GetNodeValue(kv.Key, 3), utils.GetNodeValue(kv.Key, 4),
					utils.GetNodeValue(kv.Key, 5), utils.GetNodeValue(kv.Key, 6),
					utils.GetNodeValue(kv.Key, 7),
					}
        return cncInfo, nil
}

// 1. get the original Key.
// 2. delete the key.
func deleteInfo(configName string) (bool, error) {
        // ex) /cloud-info-spaces/connection-configs/config01/AWS/AWS-Test-Driver-V0.5/credential01/region01

        key := "/cloud-info-spaces/connection-configs/" + configName

// @todo lock-start
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return false, err
        }

	err = store.Delete(keyValueList[0].Key)
	if err != nil {
		return false, err
	}
// @todo lock-end

        return true, nil
}

