// CloudDriverInfo <-> CB-Store Handler for Cloud Driver Info. Manager.
// Cloud Driver Info. Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by powerkim@etri.re.kr, 2019.09.

package driverinfomanager

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
type CloudDriverInfo struct {
	DriverName	string	// ex) "AWS-Test-Driver-V0.5"
	ProviderName	string	// ex) "AWS"
	DriverLibFileName	string	// ex) "aws-test-driver-v0.5.so"  //Already, you need to insert "*.so" in $CB_SPIDER_ROOT/cloud-driver/libs.
}
*/ //====================================================================


// format
// /cloud-info-spaces/drivers/<ID>/{Param1} [Value]
// /cloud-info-spaces/drivers/<DriverName>/{ProviderName} [DriverLibFileName]
// ex) /cloud-info-spaces/drivers/AWS_driver01-V0.5/AWS [aws-test-driver-v0.5.so]

func insertInfo(driverName string, providerName string, driverLibFileName string) error {
	// ex) /cloud-info-spaces/drivers/AWS_driver01-V0.5/AWS [aws-test-driver-v0.5.so]

	key := "/cloud-info-spaces/drivers/" + driverName + "/" + providerName
	value := driverLibFileName

	err := store.Put(key, value)
        if err != nil {
                //cblog.Error(err)
		return err
        }
	return nil
}

// 1. get key-value list
// 2. create CloudDriverInfo List & return
func listInfo() ([]*CloudDriverInfo, error) {
        // ex) /cloud-info-spaces/drivers/AWS_driver01-V0.5/AWS [aws-test-driver-v0.5.so]

        key := "/cloud-info-spaces/drivers"
        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return nil, err
        }

        cloudDriverInfoList := make([]*CloudDriverInfo, len(keyValueList))
        for count, kv := range keyValueList {
                drvInfo := &CloudDriverInfo{utils.GetNodeValue(kv.Key, 3), utils.GetNodeValue(kv.Key, 4), kv.Value}
                cloudDriverInfoList[count] = drvInfo
        }

        return cloudDriverInfoList, nil
}

// 1. get a key-value
// 2. create CloudDriverInfo & return
func getInfo(driverName string) (*CloudDriverInfo, error) {
        // ex) /cloud-info-spaces/drivers/AWS_driver01-V0.5/AWS [aws-test-driver-v0.5.so]

	
	key := "/cloud-info-spaces/drivers/" + driverName

        keyValueList, err := store.GetList(key, true)
        if err != nil {
                return nil, err
        }

	if len(keyValueList) < 1 {
		return nil, nil
	}

	providerName := utils.GetNodeValue(keyValueList[0].Key, 4)
	driverLibFileName := keyValueList[0].Value

	drvInfo := &CloudDriverInfo{driverName, providerName, driverLibFileName}

        return drvInfo, nil
}

// 1. get the original Key.
// 2. delete the key.
func deleteInfo(driverName string) (bool, error) {
        // ex) /cloud-info-spaces/drivers/AWS_driver01-V0.5/AWS [aws-test-driver-v0.5.so]


        key := "/cloud-info-spaces/drivers/" + driverName

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

