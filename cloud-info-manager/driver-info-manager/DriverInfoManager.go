// Cloud Driver Info. Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by powerkim@etri.re.kr, 2019.09.

package driverinfomanager

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/cloud-barista/cb-store/config"
)

var cblog *logrus.Logger

func init() {
        cblog = config.Cblogger
}

//====================================================================
type CloudDriverInfo struct {
	DriverName	string	// ex) "AWS-Test-Driver-V0.5"
	ProviderName	string	// ex) "AWS"
	DriverLibFileName	string	// ex) "aws-test-driver-v0.5.so"  //Already, you need to insert "*.so" in $CB_SPIDER_ROOT/cloud-driver/libs.
}
//====================================================================

func RegisterCloudDriverInfo(cldInfo CloudDriverInfo) (*CloudDriverInfo, error) {
	return RegisterCloudDriver(cldInfo.DriverName, cldInfo.ProviderName, cldInfo.DriverLibFileName)
}


// 1. check params
// 2. check driver files
// 3. insert them into cb-store
// You should copy the driver library into ~/libs before.
func RegisterCloudDriver(driverName string, providerName string, driverLibFileName string) (*CloudDriverInfo, error) {
	cblog.Info("call RegisterCloudDriver()")

	cblog.Debug("check params")
	err := checkParams(driverName, providerName, driverLibFileName)
	if err != nil {
		return nil, err
	
	}

	cblog.Debug("check the driver library file")
	err = checkDriverLibFile(driverLibFileName)
	if err != nil {
		return nil, err
	}
	

	cblog.Debug("insert metainfo into store")

	err = insertInfo(driverName, providerName, driverLibFileName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	drvInfo := &CloudDriverInfo{driverName, providerName, driverLibFileName}
	return drvInfo, nil
}

func ListCloudDriver() ([]*CloudDriverInfo, error) {
	cblog.Info("call ListCloudDriver()")

        cloudDriverInfoList, err := listInfo()
        if err != nil {
                return nil, err
        }

        return cloudDriverInfoList, nil
}

// 1. check params
// 2. get DriverInfo from cb-store
func GetCloudDriver(driverName string) (*CloudDriverInfo, error) {
	cblog.Info("call GetCloudDriver()")

	if driverName == "" {
                return nil, fmt.Errorf("driverName is empty!")
        }
	
	drvInfo, err := getInfo(driverName)
	if err != nil {
                cblog.Error(err)
                return nil, err
        }

	return drvInfo, err
}

func UnRegisterCloudDriver(driverName string) (bool, error) {
	cblog.Info("call UnRegisterCloudDriver()")

        if driverName == "" {
                return false, fmt.Errorf("driverName is empty!")
        }

        result, err := deleteInfo(driverName)
        if err != nil {
                cblog.Error(err)
                return false, err
        }

        return result, nil
}

//----------------

func checkParams(driverName string, providerName string, driverLibFileName string) error {
        if driverName == "" {
                return fmt.Errorf("driverName is empty!")
        }
        if providerName == "" {
                return fmt.Errorf("providerName is empty!")
        }
        if driverLibFileName == "" {
                return fmt.Errorf("driverLibFileName is empty!")
        }
	return nil
}

// 1. check to exist file
// 2. check to be a shared library file
func checkDriverLibFile(driverLibFileName string) error {
	// @todo
	return nil
}

