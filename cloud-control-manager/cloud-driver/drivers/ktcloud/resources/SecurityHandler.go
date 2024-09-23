// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// KT Cloud Security Group Handler
//
// by ETRI, 2021.05.

package resources

import (
	"fmt"
	"os"
	"io"
	"strings"
	// "crypto/aes"
	// "crypto/cipher"
	"encoding/base64"

	// "github.com/davecgh/go-spew/spew"
	ktsdk "github.com/cloud-barista/ktcloud-sdk-go"

	"encoding/json"
	"errors"
	// "strconv"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type KtCloudSecurityHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	Client         *ktsdk.KtCloudClient
}

const (
	sgDir string = "/cloud-driver-libs/.securitygroup-kt/"
	//filePath string = "./log/"  // ~/ktcloud/main/log
)

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("KT Cloud SecurityGroup Handler")
}

type SecurityGroup struct {
    IID   		IId 		`json:"IId"`
    VpcIID   	VpcIId 		`json:"VpcIID"`
    Direc   	string 		`json:"Direction"`
    Secu_Rules  []Security_Rule `json:"SecurityRules"`
}

type IId struct {
    NameID   	string 		`json:"NameId"`
    SystemID   	string 		`json:"SystemId"`
}

type VpcIId struct {
    NameID   	string 		`json:"NameId"`
    SystemID   	string 		`json:"SystemId"`
}

type Security_Rule struct {
    FromPort 	string 		`json:"FromPort"`
    ToPort  	string 		`json:"ToPort"`
    Protocol  	string 		`json:"IPProtocol"`
    Direc  		string 		`json:"Direction"`
    Cidr  		string 		`json:"CIDR"`
}

func (securityHandler *KtCloudSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called CreateSecurity()!")
	zoneId := securityHandler.RegionInfo.Zone
	if zoneId == "" {
		cblogger.Error("Failed to Get Zone info. from the connection info.")
		return irs.SecurityInfo{}, errors.New("Failed to Get Zone info. from the connection info.")
	} else {
		cblogger.Infof("ZoneId : %s", zoneId)
	}

	sgPath := os.Getenv("CBSPIDER_ROOT") + sgDir	
	sgFilePath := sgPath + zoneId + "/"
	
	// Check if the KeyPair Folder Exists, and Create it
	if err := CheckFolderAndCreate(sgPath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup Path : ", err)
		return irs.SecurityInfo{}, err
	}	

	// Check if the KeyPair Folder Exists, and Create it
	if err := CheckFolderAndCreate(sgFilePath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup File Path : ", err)
		return irs.SecurityInfo{}, err
	}

	// Check SecurityGroup Exists
	sgList, err := securityHandler.ListSecurity()
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	for _, sg := range sgList {
		if sg.IId.NameId == securityReqInfo.IId.NameId {
			createErr := errors.New("Security Group with name " + securityReqInfo.IId.NameId + " already exists", )
			return irs.SecurityInfo{}, createErr
		}
	}

	hashFileName := base64.StdEncoding.EncodeToString([]byte(securityReqInfo.IId.NameId))	
	cblogger.Infof("# S/G NameId : "+ securityReqInfo.IId.NameId)
	cblogger.Infof("# Hashed FileName : "+ hashFileName + ".json")

	file, _ := json.MarshalIndent(securityReqInfo, "", " ")
	writeErr := os.WriteFile(sgFilePath + hashFileName + ".json", file, 0644)
	if writeErr != nil {
		cblogger.Error("Failed to write the file: "+ sgFilePath + hashFileName + ".json", writeErr)
		return irs.SecurityInfo{}, writeErr
	}
	cblogger.Infof("Succeeded in writing the S/G file: "+ sgFilePath + hashFileName + ".json")

	// Because it's managed as a file, there's no SystemId created.
	securityReqInfo.IId.SystemId = securityReqInfo.IId.NameId
	// Return the created SecurityGroup info.
	securityInfo, err := securityHandler.GetSecurity(irs.IID{SystemId: securityReqInfo.IId.SystemId})
	if err != nil {
		return irs.SecurityInfo{}, err
	}
	return securityInfo, nil
}

func (securityHandler *KtCloudSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called GetSecurity()!!")

	securityIID.NameId = securityIID.SystemId
	hashFileName := base64.StdEncoding.EncodeToString([]byte(securityIID.NameId))	

	cblogger.Infof("# securityIID.NameId : "+ securityIID.NameId)
	cblogger.Infof("# hashFileName : "+ hashFileName + ".json")

	zoneId := securityHandler.RegionInfo.Zone
	if zoneId == "" {
		cblogger.Error("Failed to Get Zone info. from the connection info.")
		return irs.SecurityInfo{}, errors.New("Failed to Get Zone info. from the connection info.")
	} else {
		cblogger.Infof("ZoneId : %s", zoneId)
	}

	sgPath := os.Getenv("CBSPIDER_ROOT") + sgDir	
	sgFilePath := sgPath + zoneId + "/"
	
	// Check if the KeyPair Folder Exists, and Create it
	if err := CheckFolderAndCreate(sgPath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup Path : ", err)
		return irs.SecurityInfo{}, err
	}	

	// Check if the KeyPair Folder Exists, and Create it
	if err := CheckFolderAndCreate(sgFilePath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup File Path : ", err)
		return irs.SecurityInfo{}, err
	}

	sgFileName := sgFilePath + hashFileName + ".json"
    jsonFile, err := os.Open(sgFileName)
    if err != nil {
		cblogger.Error("Failed to Find the S/G file : "+ sgFileName +" ", err)
		return irs.SecurityInfo{}, err
    }
	cblogger.Infof("Succeeded in Finding and Opening the S/G file: "+ sgFileName)
    defer jsonFile.Close()

	var sg SecurityGroup
	byteValue, readErr := io.ReadAll(jsonFile)
	if readErr != nil {
		cblogger.Error("Failed to Read the S/G file : "+ sgFileName, readErr)
    }
    json.Unmarshal(byteValue, &sg)
	// spew.Dump(sg)

	// Caution : ~~~ := mappingSecurityInfo( ) =>  ~~~ := securityHandler.mappingSecurityInfo( )
	securityGroupInfo, securityInfoError := securityHandler.mappingSecurityInfo(sg)
	if securityInfoError != nil {
		cblogger.Error(securityInfoError)
		return irs.SecurityInfo{}, securityInfoError
	}
	return securityGroupInfo, nil
}

func (securityHandler *KtCloudSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called ListSecurity()!!")

	var securityIID irs.IID
	var securityGroupList []*irs.SecurityInfo
	// var sg SecurityGroup

	zoneId := securityHandler.RegionInfo.Zone
	if zoneId == "" {
		cblogger.Error("Failed to Get Zone info. from the connection info.")
		return []*irs.SecurityInfo{}, errors.New("Failed to Get Zone info. from the connection info.")
	} else {
		cblogger.Infof("ZoneId : %s", zoneId)
	}

	sgPath := os.Getenv("CBSPIDER_ROOT") + sgDir	
	sgFilePath := sgPath + zoneId + "/"
	
	// Check if the KeyPair Folder Exists, and Create it
	if err := CheckFolderAndCreate(sgPath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup Path : ", err)
		return []*irs.SecurityInfo{}, err
	}	

	// Check if the KeyPair Folder Exists, and Create it
	if err := CheckFolderAndCreate(sgFilePath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup File Path : ", err)
		return []*irs.SecurityInfo{}, err
	}

	// File list on the local directory 
	dirFiles, readRrr := os.ReadDir(sgFilePath)
	if readRrr != nil {
		return []*irs.SecurityInfo{}, readRrr
	}

	for _, file := range dirFiles {
		fileName := strings.TrimSuffix(file.Name(), ".json")  // 접미사 제거
		decString, baseErr := base64.StdEncoding.DecodeString(fileName)
		if baseErr != nil {
			cblogger.Errorf("Failed to Decode the Filename : %s", fileName)
			return []*irs.SecurityInfo{}, baseErr
		}
		sgFileName := string(decString)
		// sgFileName := filePath + file.Name()
		securityIID.SystemId = sgFileName
		cblogger.Infof("# S/G Group Name : " + securityIID.SystemId)

		sgInfo, err := securityHandler.GetSecurity(irs.IID{SystemId: securityIID.SystemId})
		if err != nil {
			cblogger.Errorf("Failed to Find the SecurityGroup : %s", securityIID.SystemId)
			return []*irs.SecurityInfo{}, err
		}
		securityGroupList = append(securityGroupList, &sgInfo)
	}
	return securityGroupList, nil
}

func (securityHandler *KtCloudSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud cloud driver: called DeleteSecurity()!")

	securityIID.NameId = securityIID.SystemId
	zoneId := securityHandler.RegionInfo.Zone
	if zoneId == "" {
		cblogger.Error("Failed to Get Zone info. from the connection info.")

		return false, errors.New("Failed to Get Zone info. from the connection info.")
	} else {
		cblogger.Infof("ZoneId : %s", zoneId)
	}

	sgPath := os.Getenv("CBSPIDER_ROOT") + sgDir	
	sgFilePath := sgPath + zoneId + "/"
	// Check if the KeyPair Folder Exists, and Create it
	if err := CheckFolderAndCreate(sgPath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup Path : ", err)
		return false, err
	}	

	// Check if the KeyPair Folder Exists, and Create it
	if err := CheckFolderAndCreate(sgFilePath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup File Path : ", err)
		return false, err
	}

	hashFileName := base64.StdEncoding.EncodeToString([]byte(securityIID.NameId))	
	sgFileName := sgFilePath + hashFileName + ".json"
	cblogger.Infof("S/G file to Delete : [%s]", sgFileName)

	//To check whether the security group exists.
	_, getErr := securityHandler.GetSecurity(irs.IID{SystemId: securityIID.SystemId})
	if getErr != nil {
		cblogger.Errorf("Failed to Find the SecurityGroup : %s", securityIID.SystemId)
		return false, getErr
	}

	// To Remove the S/G file on the Local machine.
	delErr := os.Remove(sgFileName) 
	if delErr != nil {
		newErr := fmt.Errorf("Failed to Delete the file : %s, [%v]", sgFileName, delErr)
		cblogger.Error(newErr.Error())
		return false, newErr
	}
	cblogger.Infof("Succeeded in Deleting the SecurityGroup : " + securityIID.NameId)

	return true, nil
}

func (securityHandler *KtCloudSecurityHandler) mappingSecurityInfo(secuGroup SecurityGroup) (irs.SecurityInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called mappingSecurityInfo()!")
	var securityRuleList []irs.SecurityRuleInfo
	var securityRuleInfo irs.SecurityRuleInfo

	for i := 0; i < len(secuGroup.Secu_Rules); i++ {
		securityRuleInfo.FromPort = secuGroup.Secu_Rules[i].FromPort
		securityRuleInfo.ToPort = secuGroup.Secu_Rules[i].ToPort
		securityRuleInfo.IPProtocol = secuGroup.Secu_Rules[i].Protocol //KT Cloud S/G의 경우, TCP, UDP, ICMP 가능 
		securityRuleInfo.Direction = secuGroup.Secu_Rules[i].Direc //KT Cloud S/G의 경우 inbound rule만 지원
		securityRuleInfo.CIDR = secuGroup.Secu_Rules[i].Cidr
	
		securityRuleList = append(securityRuleList, securityRuleInfo)
    }

	securityInfo := irs.SecurityInfo{
		IId:           irs.IID{NameId: secuGroup.IID.NameID, SystemId: secuGroup.IID.NameID},
		//KT Cloud의 CB에서 파일로 관리되므로 SystemId는 NameId와 동일하게
		VpcIID:        irs.IID{NameId: secuGroup.VpcIID.NameID, SystemId: secuGroup.VpcIID.SystemID},
		SecurityRules: &securityRuleList,

		// KeyValueList: []irs.KeyValue{
		// 	{Key: "IpAddress", Value: KtCloudFirewallRule.IpAddress},
		// 	{Key: "IpAddressID", Value: KtCloudFirewallRule.IpAddressId},
		// 	{Key: "State", Value: KtCloudFirewallRule.State},
		// },
	}

	return securityInfo, nil
}

func (securityHandler *KtCloudSecurityHandler) AddRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	cblogger.Info("KT Cloud cloud Driver: called AddRules()!")
    return irs.SecurityInfo{}, fmt.Errorf("Does not support AddRules() yet!!")
}

func (securityHandler *KtCloudSecurityHandler) RemoveRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (bool, error) {
	cblogger.Info("KT Cloud cloud Driver: called RemoveRules()!")
    return false, fmt.Errorf("Does not support RemoveRules() yet!!")
}

func (securityHandler *KtCloudSecurityHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	return nil, errors.New("Does not support ListIID() yet!!")
}

