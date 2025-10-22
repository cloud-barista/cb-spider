// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2024.05.

package nlbinfomanager

import (
	"fmt"
	"strings"
	"github.com/sirupsen/logrus"

	idrv 		"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	infostore 	"github.com/cloud-barista/cb-spider/info-store"	

	cblog 		"github.com/cloud-barista/cb-log"
)

const KEY_COLUMN_NAME = "nlb_id"

type NlbInfo struct {
	NlbID   			string          	`gorm:"primaryKey"`
	ProviderName      	string               // ex) "KT"
	KeyValueInfoList  	infostore.KVList 	`gorm:"type:text"`
}

var cblogger *logrus.Logger
func init() {
	cblogger = cblog.GetLogger("CB-SPIDER")
	db, err := infostore.Open()
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&NlbInfo{})
	infostore.Close(db)
}

func RegisterNlbInfo(nlbInfo NlbInfo) (*NlbInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called RegisterNlbInfo()")

	cblogger.Debug("check params")
	err := checkParams(nlbInfo.NlbID, nlbInfo.ProviderName, nlbInfo.KeyValueInfoList)
	if err != nil {
		return nil, err
	}

	// trim user inputs
	nlbInfo.NlbID = strings.TrimSpace(nlbInfo.NlbID)
	nlbInfo.ProviderName = strings.ToUpper(strings.TrimSpace(nlbInfo.ProviderName))

	cblogger.Debug("insert metainfo into store")

	err = infostore.Insert(&nlbInfo)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	return &nlbInfo, nil
}

// Register GetNlbInfo to info-store (DB)
func RegisterNlb(nlbID string, providereName string, keyValueInfoList []idrv.KeyValue) (*NlbInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called RegisterNlb()")

	return RegisterNlbInfo(NlbInfo{nlbID, providereName, keyValueInfoList})
}

func checkParams(nlbID string, providerName string, keyValueInfoList []idrv.KeyValue) error {
	if nlbID == "" {
		return fmt.Errorf("NlbID is empty!")
	}
	if providerName == "" {
		return fmt.Errorf("providerName is empty!")
	}
	if keyValueInfoList == nil {
		return fmt.Errorf("KeyValue List is nil!")
	}

	return nil
}

// Get GetNlbInfo from info-store (DB)
func GetNlb(nlbID string) (*NlbInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetNlb()")

	if nlbID == "" {
		return nil, fmt.Errorf("NlbID is empty!")
	}

	var nlbInfo NlbInfo
	err := infostore.Get(&nlbInfo, KEY_COLUMN_NAME, nlbID)
	if err != nil {
		cblogger.Debug(err)
		// return nil, err
	}

	return &nlbInfo, err
}

func UnRegisterNlb(nlbID string) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called UnRegisterNlb()")

	if nlbID == "" {
		return false, fmt.Errorf("NlbID is empty!")
	}

	result, err := infostore.Delete(&NlbInfo{}, KEY_COLUMN_NAME, nlbID)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	return result, nil
}
