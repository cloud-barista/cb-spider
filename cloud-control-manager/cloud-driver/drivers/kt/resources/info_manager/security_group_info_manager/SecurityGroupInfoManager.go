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
	ProviderName      	string               // ex) "KT"
	KeyValueInfoList  	infostore.KVList 	`gorm:"type:text"`
}

type KTCloudSecurityGroupDef struct {
	SgID string `gorm:"primaryKey"`
	Zone string `gorm:"index"`
	Data string `gorm:"type:text"`
}

var cblogger *logrus.Logger
func init() {
	cblogger = cblog.GetLogger("CB-SPIDER")
	db, err := infostore.Open()
	if err != nil {
		panic("failed to connect database")
	}
	db.AutoMigrate(&SecurityGroupInfo{}, &KTCloudSecurityGroupDef{})
	infostore.Close(db)
}

func RegisterSecurityGroupInfo(sgInfo SecurityGroupInfo) (*SecurityGroupInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called RegisterSecurityGroupInfo()")

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
	cblogger.Info("KT Cloud VPC Driver: called RegisterSecurityGroup()")

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
	cblogger.Info("KT Cloud VPC Driver: called GetSecurityGroup()")

	if vmID == "" {
		return nil, fmt.Errorf("vmID is empty!")
	}

	var sgInfo SecurityGroupInfo
	err := infostore.Get(&sgInfo, KEY_COLUMN_NAME, vmID)
	if err != nil {
		cblogger.Debug(err)
	}

	return &sgInfo, err
}

func UnRegisterSecurityGroup(vmID string) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called UnRegisterSecurityGroup()")

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

// GetVMIDsBySecurityGroup returns all VM IDs that are associated with the given security group ID or name.
func GetVMIDsBySecurityGroup(sgID string) ([]string, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetVMIDsBySecurityGroup()")

	if strings.TrimSpace(sgID) == "" {
		return nil, fmt.Errorf("sgID is empty!")
	}

	var sgInfoList []SecurityGroupInfo
	err := infostore.List(&sgInfoList)
	if err != nil {
		cblogger.Errorf("Failed to list SecurityGroupInfo from store: %v", err)
		return nil, err
	}

	var vmIDs []string
	seen := make(map[string]bool)
	for _, info := range sgInfoList {
		for _, kv := range info.KeyValueInfoList {
			if strings.EqualFold(kv.Key, sgID) || strings.EqualFold(kv.Value, sgID) {
				if !seen[info.VmID] {
					seen[info.VmID] = true
					vmIDs = append(vmIDs, info.VmID)
				}
				break
			}
		}
	}

	return vmIDs, nil
}

// SaveKTCloudSGDef persists the full SecurityGroup definition into infostore (DB)
func SaveKTCloudSGDef(sgID string, zone string, sgInfoJSON string) error {
	cblogger.Info("KT Cloud VPC Driver: called SaveKTCloudSGDef()")
	if strings.TrimSpace(sgID) == "" {
		return fmt.Errorf("sgID is empty")
	}
	sgDef := KTCloudSecurityGroupDef{
		SgID: strings.TrimSpace(sgID),
		Zone: strings.TrimSpace(zone),
		Data: sgInfoJSON,
	}
	return infostore.Insert(&sgDef)
}

// GetKTCloudSGDef retrieves the SecurityGroup definition from infostore (DB)
func GetKTCloudSGDef(sgID string) (*KTCloudSecurityGroupDef, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetKTCloudSGDef()")
	if strings.TrimSpace(sgID) == "" {
		return nil, fmt.Errorf("sgID is empty")
	}
	var sgDef KTCloudSecurityGroupDef
	err := infostore.Get(&sgDef, "sg_id", strings.TrimSpace(sgID))
	if err != nil {
		return nil, err
	}
	return &sgDef, nil
}

// DeleteKTCloudSGDef removes the SecurityGroup definition from infostore (DB)
func DeleteKTCloudSGDef(sgID string) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called DeleteKTCloudSGDef()")
	if strings.TrimSpace(sgID) == "" {
		return false, fmt.Errorf("sgID is empty")
	}
	return infostore.Delete(&KTCloudSecurityGroupDef{}, "sg_id", strings.TrimSpace(sgID))
}

// ListKTCloudSGDefs lists all SecurityGroup definitions from infostore (DB)
func ListKTCloudSGDefs(zone string) ([]KTCloudSecurityGroupDef, error) {
	cblogger.Info("KT Cloud VPC Driver: called ListKTCloudSGDefs()")
	var allDefs []KTCloudSecurityGroupDef
	err := infostore.List(&allDefs)
	if err != nil {
		return nil, err
	}
	if zone == "" {
		return allDefs, nil
	}
	var filtered []KTCloudSecurityGroupDef
	for _, d := range allDefs {
		if strings.EqualFold(d.Zone, zone) {
			filtered = append(filtered, d)
		}
	}
	return filtered, nil
}
