// RegionInfo <-> CB-Store Handler for Cloud Region Info. Manager.
// Cloud Region Info. Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by powerkim@etri.re.kr, 2019.09.

package regioninfomanager

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
type RegionInfo struct {
        RegionName      string  // ex) "region01"
        ProviderName    string  // ex) "GCP"
        KeyValueInfoList        []icbs.KeyValue // ex) { {region, us-east1},
                                                //       {zone, us-east1-c},
}
*/ //====================================================================


// format
// /cloud-info-spaces/regions/<RegionName>/{ProviderName}/{key1} [value1]
// /cloud-info-spaces/regions/<RegionName>/{ProviderName}/{key2} [value2]
// ex-1)
// /cloud-info-spaces/regions/aws_region01/AWS/region [ap-northeast-2]
// ex-2)
// /cloud-info-spaces/regions/gcp_region02/GCP/region [us-east1]
// /cloud-info-spaces/regions/gcp_region02/GCP/zone [us-east1-c]



func insertInfo(regionName string, providerName string, keyValueList []icbs.KeyValue) error {
	// ex-1)
	// /cloud-info-spaces/regions/aws_region01/AWS/region [ap-northeast-2]
	// ex-2)
	// /cloud-info-spaces/regions/gcp_region02/GCP/region [us-east1]
	// /cloud-info-spaces/regions/gcp_region02/GCP/zone [us-east1-c]

	format := "/cloud-info-spaces/regions/" + regionName + "/" + providerName
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
// 2. create RegionInfo List & return
func listInfo() ([]*RegionInfo, error) {
        // ex-1)
        // /cloud-info-spaces/regions/aws_region01/AWS/region [ap-northeast-2]
        // ex-2)
        // /cloud-info-spaces/regions/gcp_region02/GCP/region [us-east1]
        // /cloud-info-spaces/regions/gcp_region02/GCP/zone [us-east1-c]


        key := "/cloud-info-spaces/regions"
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return nil, err
        }

        var regionInfoList []*RegionInfo
	var inKeyValueList []icbs.KeyValue
	prevRegionName := ""
	prevProviderName := ""
        for _, kv := range keyValueList {

		regionName := utils.GetNodeValue(kv.Key, 3)
		providerName := utils.GetNodeValue(kv.Key, 4)

		if prevRegionName=="" || regionName == prevRegionName {
			prevRegionName = regionName
			prevProviderName = providerName
			keyValue := icbs.KeyValue{utils.GetNodeValue(kv.Key, 5), kv.Value}
			inKeyValueList = append(inKeyValueList, keyValue)
		} else {
			// insert prev RegionInfo
			rgnInfo := &RegionInfo{prevRegionName, prevProviderName, inKeyValueList}
			regionInfoList = append(regionInfoList, rgnInfo)

			prevRegionName = regionName
			prevProviderName = providerName
			inKeyValueList = nil
			keyValue := icbs.KeyValue{utils.GetNodeValue(kv.Key, 5), kv.Value}
			inKeyValueList = append(inKeyValueList, keyValue)
		}

        }

	if len(keyValueList) > 0 {
		// insert last RegionInfo
		rgnInfo := &RegionInfo{prevRegionName, prevProviderName, inKeyValueList}
		regionInfoList = append(regionInfoList, rgnInfo)
	}

        return regionInfoList, nil
}

// 1. get a key-value
// 2. create RegionInfo & return
func getInfo(regionName string) (*RegionInfo, error) {
        // ex-1)
        // /cloud-info-spaces/regions/aws_region01/AWS/region [ap-northeast-2]
        // ex-2)
        // /cloud-info-spaces/regions/gcp_region02/GCP/region [us-east1]
        // /cloud-info-spaces/regions/gcp_region02/GCP/zone [us-east1-c]

	
	key := "/cloud-info-spaces/regions/" + regionName

	// key is not the key of cb-store, so we have to use GetList()
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return nil, err
        }

        if len(keyValueList) < 1 {
                return nil, fmt.Errorf(regionName + ": is not exist!")
        }

        // keyValueList should have ~/driverName/... or ~/driverName-01/...,
        // so we have to check the sameness of driverName.
        // and make a KeyValueList for only Target key
        var oneKeyValueList []icbs.KeyValue
        for _, kv := range keyValueList {
                if utils.GetNodeValue(kv.Key, 3) == regionName {
                        oneKeyValueList = append(oneKeyValueList, *kv)
                }
        }

        if len(oneKeyValueList) < 1 {
                return nil, fmt.Errorf(regionName + ": is not exist!")
        }

        var inKeyValueList []icbs.KeyValue
        // get ProviderName
        providerName := utils.GetNodeValue(oneKeyValueList[0].Key, 4)
        // get KeyValueList
        for _, kv := range oneKeyValueList {
                keyValue := icbs.KeyValue{utils.GetNodeValue(kv.Key, 5), kv.Value}
                inKeyValueList = append(inKeyValueList, keyValue)
        }
	return &RegionInfo{regionName, providerName, inKeyValueList}, nil 
}

// 1. get the original Key.
// 2. delete the key.
func deleteInfo(regionName string) (bool, error) {
        // ex-1)
        // /cloud-info-spaces/regions/aws_region01/AWS/region [ap-northeast-2]
        // ex-2)
        // /cloud-info-spaces/regions/gcp_region02/GCP/region [us-east1]
        // /cloud-info-spaces/regions/gcp_region02/GCP/zone [us-east1-c]


	key := "/cloud-info-spaces/regions/" + regionName

// @todo lock-start
	// key is not the key of cb-store, so we have to use GetList()
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return false, err
        }

        var isDelete bool
        for _, kv := range keyValueList {
		// keyValueList should have ~/driverName/... or ~/driverName-01/..., 
		// so we have to check the sameness of driverName.
                if utils.GetNodeValue(kv.Key, 3) == regionName {
                        err = store.Delete(kv.Key)
                        if err != nil {
                                return false, err
                        }
                        isDelete = true
                }
        }
// @todo lock-end

        if isDelete {
                return true, nil
        }
        return false, fmt.Errorf(regionName + ": is not exist!")
}

