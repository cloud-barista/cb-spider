// Cloud Driver Info. Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2019.09.

package driverinfomanager

import (
	"fmt"
	"strings"

	"github.com/cloud-barista/cb-store/config"
	"github.com/sirupsen/logrus"

	infostore "github.com/cloud-barista/cb-spider/info-store"
)

var cblog *logrus.Logger

// ====================================================================
const KEY_COLUMN_NAME = "driver_name"

type CloudDriverInfo struct {
	DriverName        string `gorm:"primaryKey"` // ex) "AWS-Test-Driver-V0.5"
	ProviderName      string // ex) "AWS"
	DriverLibFileName string // ex) "aws-test-driver-v0.5.so"  //Already, you need to insert "*.so" in $CB_SPIDER_ROOT/cloud-driver/libs.
}

func init() {
	cblog = config.Cblogger

	fmt.Println("\n============================[Init] Cloud Driver Info Manager")

	db, err := infostore.Open()
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&CloudDriverInfo{})
	infostore.Close(db)
}

//====================================================================

func RegisterCloudDriverInfo(cldInfo CloudDriverInfo) (*CloudDriverInfo, error) {
	cblog.Info("call RegisterCloudDriver()")

	cblog.Debug("check params")
	err := checkParams(cldInfo.DriverName, cldInfo.ProviderName, cldInfo.DriverLibFileName)
	if err != nil {
		return nil, err

	}

	cblog.Debug("check the driver library file")
	err = checkDriverLibFile(cldInfo.DriverLibFileName) // @Todo
	if err != nil {
		return nil, err
	}

	// trim user inputs
	cldInfo.DriverName = strings.TrimSpace(cldInfo.DriverName)
	cldInfo.ProviderName = strings.ToUpper(strings.TrimSpace(cldInfo.ProviderName))
	cldInfo.DriverLibFileName = strings.TrimSpace(cldInfo.DriverLibFileName)

	cblog.Debug("insert metainfo into store")

	err = infostore.Insert(&cldInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return &cldInfo, nil
}

// 1. check params
// 2. check driver files
// 3. insert them into cb-store
// You should copy the driver library into ~/libs before.
func RegisterCloudDriver(driverName string, providerName string, driverLibFileName string) (*CloudDriverInfo, error) {
	return RegisterCloudDriverInfo(CloudDriverInfo{driverName, providerName, driverLibFileName})
}

func ListCloudDriver() ([]*CloudDriverInfo, error) {
	cblog.Info("call ListCloudDriver()")
	fmt.Println("before call ListCloudDriver()")
	var cloudDriverInfoList []*CloudDriverInfo
	err := infostore.List(&cloudDriverInfoList)
	if err != nil {
		return nil, err
	}
	fmt.Println("after call ListCloudDriver()")
	return cloudDriverInfoList, nil
}

// 1. check params
// 2. get DriverInfo from cb-store
func GetCloudDriver(driverName string) (*CloudDriverInfo, error) {
	cblog.Info("call GetCloudDriver()")

	if driverName == "" {
		return nil, fmt.Errorf("DriverName is empty!")
	}

	var driverInfo CloudDriverInfo
	err := infostore.Get(&driverInfo, KEY_COLUMN_NAME, driverName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return &driverInfo, err
}

func UnRegisterCloudDriver(driverName string) (bool, error) {
	cblog.Info("call UnRegisterCloudDriver()")

	if driverName == "" {
		return false, fmt.Errorf("DriverName is empty!")
	}

	var driverInfo CloudDriverInfo
	result, err := infostore.Delete(&driverInfo, KEY_COLUMN_NAME, driverName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	return result, nil
}

//----------------

func checkParams(driverName string, providerName string, driverLibFileName string) error {
	if driverName == "" {
		return fmt.Errorf("DriverName is empty!")
	}
	if providerName == "" {
		return fmt.Errorf("ProviderName is empty!")
	}
	if driverLibFileName == "" {
		return fmt.Errorf("DriverLibFileName is empty!")
	}
	return nil
}

// 1. check to exist file
// 2. check to be a shared library file
func checkDriverLibFile(driverLibFileName string) error {
	// @todo
	return nil
}
