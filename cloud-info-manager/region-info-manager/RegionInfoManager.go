// Cloud regioninfomanager Info. Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by powerkim@etri.re.kr, 2019.09.

package regioninfomanager

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
type RegionInfo struct {
	RegionName	string	// ex) "region01"
	ProviderName	string	// ex) "GCP"
	KeyValueInfoList	[]icbs.KeyValue	// ex) { {region, us-east1}, 
						//	 {zone, us-east1-c},
}
//====================================================================

func RegisterRegionInfo(rgnInfo RegionInfo) (*RegionInfo, error) {
        return RegisterRegion(rgnInfo.RegionName, rgnInfo.ProviderName, rgnInfo.KeyValueInfoList)
}

// 1. check params
// 2. insert them into cb-store
func RegisterRegion(regionName string, providerName string, keyValueInfoList []icbs.KeyValue) (*RegionInfo, error) {
	cblog.Info("call RegisterRegion()")

	cblog.Debug("check params")
	err := checkParams(regionName, providerName, keyValueInfoList)
	if err != nil {
		return nil, err
	
	}

	cblog.Debug("insert metainfo into store")

	err = insertInfo(regionName, providerName, keyValueInfoList)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	rgnInfo := &RegionInfo{regionName, providerName, keyValueInfoList}
	return rgnInfo, nil
}

func ListRegion() ([]*RegionInfo, error) {
	cblog.Info("call ListRegion()")

        regionInfoList, err := listInfo()
        if err != nil {
                return nil, err
        }

        return regionInfoList, nil
}

// 1. check params
// 2. get CredentialInfo from cb-store
func GetRegion(regionName string) (*RegionInfo, error) {
	cblog.Info("call GetRegion()")

	if regionName == "" {
                return nil, fmt.Errorf("regionName is empty!")
        }
	
	rgnInfo, err := getInfo(regionName)
	if err != nil {
                cblog.Error(err)
                return nil, err
        }

	return rgnInfo, err
}

func UnRegisterRegion(regionName string) (bool, error) {
	cblog.Info("call UnRegisterRegion()")

        if regionName == "" {
                return false, fmt.Errorf("regionName is empty!")
        }

        result, err := deleteInfo(regionName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        return result, nil
}

//----------------

func checkParams(regionName string, providerName string, keyValueInfoList []icbs.KeyValue) error {
        if regionName == "" {
                return fmt.Errorf("regionName is empty!")
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

