// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2024.05.

package securitygroupinfomanager

import (
	"fmt"
	"strings"
	"github.com/sirupsen/logrus"

	idrv 		"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	infostore 	"github.com/cloud-barista/cb-spider/info-store"	

	cblog 		"github.com/cloud-barista/cb-log"
)

const KEY_COLUMN_NAME = "vm_id"

type SecurityGroupInfo struct {
	VmID   				string          	`gorm:"primaryKey"`
	ProviderName      	string               // ex) "NCPVPC"
	KeyValueInfoList  	infostore.KVList 	`gorm:"type:text"`
}

var cblogger *logrus.Logger
func init() {
	cblogger = cblog.GetLogger("CB-SPIDER")
	db, err := infostore.Open()
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&SecurityGroupInfo{})
	infostore.Close(db)
}

func RegisterSecurityGroupInfo(sgInfo SecurityGroupInfo) (*SecurityGroupInfo, error) {
	cblogger.Info("NCP VPC Driver: called RegisterSecurityGroupInfo()")

	cblogger.Debug("check params")
	err := checkParams(sgInfo.VmID, sgInfo.ProviderName, sgInfo.KeyValueInfoList)
	if err != nil {
		return nil, err
	}

	// trim user inputs
	sgInfo.VmID = strings.TrimSpace(sgInfo.VmID)
	sgInfo.ProviderName = strings.ToUpper(strings.TrimSpace(sgInfo.ProviderName))

	cblogger.Debug("insert metainfo into store")

	err = infostore.Insert(&sgInfo)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	return &sgInfo, nil
}

// Register GetSecurityGroupInfo to info-store (DB)
func RegisterSecurityGroup(vmID string, providereName string, keyValueInfoList []idrv.KeyValue) (*SecurityGroupInfo, error) {
	cblogger.Info("NCP VPC Driver: called RegisterSecurityGroup()")

	return RegisterSecurityGroupInfo(SecurityGroupInfo{vmID, providereName, keyValueInfoList})
}

func checkParams(vmID string, providerName string, keyValueInfoList []idrv.KeyValue) error {
	if vmID == "" {
		return fmt.Errorf("vmID is empty!")
	}
	if providerName == "" {
		return fmt.Errorf("providerName is empty!")
	}
	if keyValueInfoList == nil {
		return fmt.Errorf("KeyValue List is nil!")
	}

	return nil
}

// Get GetSecurityGroupInfo from info-store (DB)
func GetSecurityGroup(vmID string) (*SecurityGroupInfo, error) {
	cblogger.Info("NCP VPC Driver: called GetSecurityGroup()")

	if vmID == "" {
		return nil, fmt.Errorf("vmID is empty!")
	}

	var sgInfo SecurityGroupInfo
	err := infostore.Get(&sgInfo, KEY_COLUMN_NAME, vmID)
	if err != nil {
		cblogger.Debug(err)
		// return nil, err
	}

	return &sgInfo, err
}

func UnRegisterSecurityGroup(vmID string) (bool, error) {
	cblogger.Info("NCP VPC Driver: called UnRegisterSecurityGroup()")

	if vmID == "" {
		return false, fmt.Errorf("vmID is empty!")
	}

	result, err := infostore.Delete(&SecurityGroupInfo{}, KEY_COLUMN_NAME, vmID)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	return result, nil
}
