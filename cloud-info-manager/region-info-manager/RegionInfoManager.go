// Cloud regioninfomanager Info. Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2023.07.
// by CB-Spider Team, 2019.09.

package regioninfomanager

import (
	"fmt"
	"strings"

	icdrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"
	"github.com/cloud-barista/cb-store/config"

	"github.com/sirupsen/logrus"

	infostore "github.com/cloud-barista/cb-spider/info-store"
)

// ====================================================================
const KEY_COLUMN_NAME = "region_name"

type RegionInfo struct {
	RegionName       string           `gorm:"primaryKey"` // ex) "region01"
	ProviderName     string           // ex) "GCP"
	KeyValueInfoList infostore.KVList `gorm:"type:text"` // stored with json format, ex) { {region, us-east1}, {zone, us-east1-c}, ...}
}

//====================================================================

var cblog *logrus.Logger

func init() {
	cblog = config.Cblogger

	db, err := infostore.Open()
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&RegionInfo{})
	infostore.Close(db)
}

// 1. check params
// 2. insert them into info-store
func RegisterRegionInfo(rgnInfo RegionInfo) (*RegionInfo, error) {
	cblog.Info("call RegisterRegionInfo()")

	cblog.Debug("check params")
	err := checkParams(rgnInfo.RegionName, rgnInfo.ProviderName, rgnInfo.KeyValueInfoList)
	if err != nil {
		return nil, err

	}

	// trim user inputs
	rgnInfo.RegionName = strings.TrimSpace(rgnInfo.RegionName)
	rgnInfo.ProviderName = strings.ToUpper(strings.TrimSpace(rgnInfo.ProviderName))

	cblog.Debug("insert metainfo into store")

	err = infostore.Insert(&rgnInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return &rgnInfo, nil
}

func RegisterRegion(regionName string, providerName string, keyValueInfoList []icdrs.KeyValue) (*RegionInfo, error) {
	cblog.Info("call RegisterRegion()")

	return RegisterRegionInfo(RegionInfo{regionName, providerName, keyValueInfoList})
}

func ListRegion() ([]*RegionInfo, error) {
	cblog.Info("call ListRegion()")

	var regionInfoList []*RegionInfo
	err := infostore.List(&regionInfoList)
	if err != nil {
		return nil, err
	}

	return regionInfoList, nil
}

// 1. check params
// 2. get RegionIfno from info-store
func GetRegion(regionName string) (*RegionInfo, error) {
	cblog.Info("call GetRegion()")

	if regionName == "" {
		return nil, fmt.Errorf("RegionName is empty!")
	}

	var regionInfo RegionInfo
	err := infostore.Get(&regionInfo, KEY_COLUMN_NAME, regionName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return &regionInfo, err
}

func UnRegisterRegion(regionName string) (bool, error) {
	cblog.Info("call UnRegisterRegion()")

	if regionName == "" {
		return false, fmt.Errorf("RegionName is empty!")
	}

	result, err := infostore.Delete(&RegionInfo{}, KEY_COLUMN_NAME, regionName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	return result, nil
}

//----------------

func checkParams(regionName string, providerName string, keyValueInfoList []icdrs.KeyValue) error {
	if regionName == "" {
		return fmt.Errorf("RegionName is empty!")
	}
	if providerName == "" {
		return fmt.Errorf("ProviderName is empty!")
	}
	if keyValueInfoList == nil {
		return fmt.Errorf("KeyValue List is nil!")
	}

	// get Provider's Meta Info
	cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo(providerName)
	if err != nil {
		cblog.Error(err)
		return err
	}

	// validate the KeyValueList of Region Input
	err = cim.ValidateKeyValueList(keyValueInfoList, cloudOSMetaInfo.Region)
	if err != nil {
		cblog.Error(err)
		return err
	}

	return nil
}
