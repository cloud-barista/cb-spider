// CloudDriverInfo <-> CB-Store Handler for Cloud Driver Info. Manager.
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

	metadb "github.com/cloud-barista/cb-spider/meta-db"
)

func init() {
	db, err := metadb.Open()
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&CloudDriverInfo{})
	metadb.Close(db)
}

/* //====================================================================
type CloudDriverInfo struct {
	DriverName	string	// ex) "AWS-Test-Driver-V0.5"
	ProviderName	string	// ex) "AWS"
	DriverLibFileName	string	// ex) "aws-test-driver-v0.5.so"  //Already, you need to insert "*.so" in $CB_SPIDER_ROOT/cloud-driver/libs.
}
*/ //====================================================================

func insert(driverInfo *CloudDriverInfo) error {
	// ex) ("driver01", "AWS", "aws-driver-v2.0.so")

	db, err := metadb.Open()
	if err != nil {
		return err
	}

	defer metadb.Close(db)
	db.Save(driverInfo)

	return nil
}

func list() ([]*CloudDriverInfo, error) {
	// ex) ("driver01", "AWS", "aws-driver-v2.0.so")

	db, err := metadb.Open()
	if err != nil {
		return nil, err
	}

	defer metadb.Close(db)
	var cloudDriverInfoList []*CloudDriverInfo
	db.Find(&cloudDriverInfoList)

	return cloudDriverInfoList, nil
}

func get(driverName string) (*CloudDriverInfo, error) {
	// ex) ("driver01", "AWS", "aws-driver-v2.0.so")

	db, err := metadb.Open()
	if err != nil {
		return nil, err
	}

	defer metadb.Close(db)
	var cloudDriverInfo CloudDriverInfo
	db.Where("driver_name = ?", driverName).Find(&cloudDriverInfo)

	return &cloudDriverInfo, nil
}

func delete(driverName string) (bool, error) {
	db, err := metadb.Open()
	if err != nil {
		return false, err
	}

	defer metadb.Close(db)
	var cloudDriverInfo CloudDriverInfo
	db.Where("driver_name = ?", driverName).Find(&cloudDriverInfo)
	if cloudDriverInfo.DriverName == "" {
		return false, fmt.Errorf(driverName + ": does not exist!")
	}

	db.Delete(&cloudDriverInfo, "driver_name = ?", driverName)

	return true, nil
}
