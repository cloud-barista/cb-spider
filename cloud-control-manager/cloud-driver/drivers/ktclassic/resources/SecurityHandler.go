// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// KT Cloud Security Group Handler
//
// by ETRI, 2021.05.
// Updated by ETRI, 2024.09.
// Updated by ETRI, 2025.02.

package resources

import (
	"fmt"
	"io"
	"os"
	"strings"

	// "crypto/aes"
	// "crypto/cipher"
	"encoding/base64"
	// "github.com/davecgh/go-spew/spew"
	"encoding/json"
	"errors"

	// "strconv"

	ktsdk "github.com/cloud-barista/ktcloud-sdk-go"

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
	IID           IId             `json:"IId"`
	VpcIID        VpcIId          `json:"VpcIID"`
	Direc         string          `json:"Direction"`
	Secu_Rules    []Security_Rule `json:"SecurityRules"`
	KeyValue_List []KeyValue      `json:"KeyValueList"`
}

type IId struct {
	NameID   string `json:"NameId"`
	SystemID string `json:"SystemId"`
}

type VpcIId struct {
	NameID   string `json:"NameId"`
	SystemID string `json:"SystemId"`
}

type Security_Rule struct {
	FromPort string `json:"FromPort"`
	ToPort   string `json:"ToPort"`
	Protocol string `json:"IPProtocol"`
	Direc    string `json:"Direction"`
	Cidr     string `json:"CIDR"`
}

func (securityHandler *KtCloudSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	cblogger.Info("KT Classic driver: called CreateSecurity()!")

	if strings.EqualFold(securityHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Failed to Get Zone info. from the connection info.")
		cblogger.Error(newErr.Error())
		return irs.SecurityInfo{}, newErr
	}
	// cblogger.Infof("ZoneId : %s", securityHandler.RegionInfo.Zone)

	sgPath := os.Getenv("CBSPIDER_ROOT") + sgDir
	sgFilePath := sgPath + securityHandler.RegionInfo.Zone + "/"

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
			createErr := errors.New("Security Group with name " + securityReqInfo.IId.NameId + " already exists")
			cblogger.Error(createErr.Error())
			return irs.SecurityInfo{}, createErr
		}
	}

	currentTime := getSeoulCurrentTime()

	newSGInfo := irs.SecurityInfo{
		IId: irs.IID{
			NameId: securityReqInfo.IId.NameId,
			// Caution!! : securityReqInfo.IId.NameId -> SystemId
			SystemId: securityReqInfo.IId.NameId,
		},
		VpcIID:        securityReqInfo.VpcIID,
		SecurityRules: securityReqInfo.SecurityRules,
		KeyValueList: []irs.KeyValue{
			{Key: "KTCloud-SecuriyGroup-info.", Value: "This SecuriyGroup info. is temporary."},
			{Key: "CreateTime", Value: currentTime},
		},
	}
	// spew.Dump(newSGInfo)

	hashFileName := base64.StdEncoding.EncodeToString([]byte(securityReqInfo.IId.NameId))
	// cblogger.Infof("# S/G NameId : "+ securityReqInfo.IId.NameId)
	// cblogger.Infof("# Hashed FileName : "+ hashFileName + ".json")

	file, _ := json.MarshalIndent(newSGInfo, "", " ")
	writeErr := os.WriteFile(sgFilePath+hashFileName+".json", file, 0644)
	if writeErr != nil {
		cblogger.Error("Failed to write the file: "+sgFilePath+hashFileName+".json", writeErr)
		return irs.SecurityInfo{}, writeErr
	}
	// cblogger.Infof("Succeeded in writing the S/G file: "+ sgFilePath + hashFileName + ".json")

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
	cblogger.Info("KT Classic driver: called GetSecurity()!!")

	securityIID.NameId = securityIID.SystemId
	hashFileName := base64.StdEncoding.EncodeToString([]byte(securityIID.NameId))

	// cblogger.Infof("# securityIID.NameId : "+ securityIID.NameId)
	// cblogger.Infof("# hashFileName : "+ hashFileName + ".json")

	if strings.EqualFold(securityHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Failed to Get Zone info. from the connection info.")
		cblogger.Error(newErr.Error())
		return irs.SecurityInfo{}, newErr
	}
	// cblogger.Infof("ZoneId : %s", securityHandler.RegionInfo.Zone)

	sgPath := os.Getenv("CBSPIDER_ROOT") + sgDir
	sgFilePath := sgPath + securityHandler.RegionInfo.Zone + "/"

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
		cblogger.Error("Failed to Find the S/G file : "+sgFileName+" ", err)
		return irs.SecurityInfo{}, err
	}
	defer jsonFile.Close()

	var sg SecurityGroup
	byteValue, readErr := io.ReadAll(jsonFile)
	if readErr != nil {
		cblogger.Error("Failed to Read the S/G file : "+sgFileName, readErr)
		return irs.SecurityInfo{}, readErr
	}
	json.Unmarshal(byteValue, &sg)
	// spew.Dump(sg)

	securityGroupInfo, mapError := securityHandler.mappingSecurityInfo(sg)
	if mapError != nil {
		cblogger.Error(mapError)
		return irs.SecurityInfo{}, mapError
	}
	return securityGroupInfo, nil
}

func (securityHandler *KtCloudSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	cblogger.Info("KT Classic driver: called ListSecurity()!!")

	var securityIID irs.IID
	var securityGroupList []*irs.SecurityInfo
	// var sg SecurityGroup

	if strings.EqualFold(securityHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Failed to Get Zone info. from the connection info.")
		cblogger.Error(newErr.Error())
		return []*irs.SecurityInfo{}, newErr
	}
	// cblogger.Infof("ZoneId : %s", securityHandler.RegionInfo.Zone)

	sgPath := os.Getenv("CBSPIDER_ROOT") + sgDir
	sgFilePath := sgPath + securityHandler.RegionInfo.Zone + "/"

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
		fileName := strings.TrimSuffix(file.Name(), ".json") // 접미사 제거
		decString, baseErr := base64.StdEncoding.DecodeString(fileName)
		if baseErr != nil {
			cblogger.Errorf("Failed to Decode the Filename : %s", fileName)
			return []*irs.SecurityInfo{}, baseErr
		}
		sgFileName := string(decString)
		// sgFileName := filePath + file.Name()
		securityIID.SystemId = sgFileName
		// cblogger.Infof("# S/G Group Name : [%s]", sgFileName)

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
	cblogger.Info("KT Classic driver: called DeleteSecurity()!")

	securityIID.NameId = securityIID.SystemId

	if strings.EqualFold(securityHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Failed to Get Zone info. from the connection info.")
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	sgPath := os.Getenv("CBSPIDER_ROOT") + sgDir
	sgFilePath := sgPath + securityHandler.RegionInfo.Zone + "/"
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
	// cblogger.Infof("S/G file to Delete : [%s]", sgFileName)

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
	// cblogger.Infof("Succeeded in Deleting the SecurityGroup : " + securityIID.NameId)

	return true, nil
}

func (securityHandler *KtCloudSecurityHandler) mappingSecurityInfo(sg SecurityGroup) (irs.SecurityInfo, error) {
	cblogger.Info("KT Classic driver: called mappingSecurityInfo()!")
	var sgRuleList []irs.SecurityRuleInfo
	var sgRuleInfo irs.SecurityRuleInfo
	var sgKeyValue irs.KeyValue
	var sgKeyValueList []irs.KeyValue

	for i := 0; i < len(sg.Secu_Rules); i++ {
		sgRuleInfo.FromPort = sg.Secu_Rules[i].FromPort
		sgRuleInfo.ToPort = sg.Secu_Rules[i].ToPort
		sgRuleInfo.IPProtocol = sg.Secu_Rules[i].Protocol // For KT Cloud Classic S/G, TCP/UDP/ICMP is available
		sgRuleInfo.Direction = sg.Secu_Rules[i].Direc     // For KT Cloud Classic S/G, supports only inbound rule.
		sgRuleInfo.CIDR = sg.Secu_Rules[i].Cidr

		sgRuleList = append(sgRuleList, sgRuleInfo)
	}

	for k := 0; k < len(sg.KeyValue_List); k++ {
		sgKeyValue.Key = sg.KeyValue_List[k].Key
		sgKeyValue.Value = sg.KeyValue_List[k].Value
		sgKeyValueList = append(sgKeyValueList, sgKeyValue)
	}

	securityInfo := irs.SecurityInfo{
		IId: irs.IID{NameId: sg.IID.NameID, SystemId: sg.IID.NameID},
		//KT Cloud의 CB에서 파일로 관리되므로 SystemId는 NameId와 동일하게
		VpcIID:        irs.IID{NameId: sg.VpcIID.NameID, SystemId: sg.VpcIID.SystemID},
		SecurityRules: &sgRuleList,
		KeyValueList:  sgKeyValueList,
	}
	return securityInfo, nil
}

func (securityHandler *KtCloudSecurityHandler) AddRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	cblogger.Info("KT Classic Driver: called AddRules()!")
	return irs.SecurityInfo{}, fmt.Errorf("Does not support AddRules() yet!!")
}

func (securityHandler *KtCloudSecurityHandler) RemoveRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (bool, error) {
	cblogger.Info("KT Classic Driver: called RemoveRules()!")
	return false, fmt.Errorf("Does not support RemoveRules() yet!!")
}

func (securityHandler *KtCloudSecurityHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")

	if strings.EqualFold(securityHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Failed to Get Zone info. from the connection info.")
		cblogger.Error(newErr.Error())
		return nil, newErr
	}
	// cblogger.Infof("ZoneId : %s", securityHandler.RegionInfo.Zone)

	sgPath := os.Getenv("CBSPIDER_ROOT") + sgDir
	sgFilePath := sgPath + securityHandler.RegionInfo.Zone + "/"

	// Check if the KeyPair Folder Exists, and Create it
	if err := CheckFolderAndCreate(sgPath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup Path : ", err)
		return nil, err
	}

	// Check if the KeyPair Folder Exists, and Create it
	if err := CheckFolderAndCreate(sgFilePath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup File Path : ", err)
		return nil, err
	}

	// File list on the local directory
	dirFiles, readErr := os.ReadDir(sgFilePath)
	if readErr != nil {
		return nil, readErr
	}

	var iidList []*irs.IID
	for _, file := range dirFiles {
		fileName := strings.TrimSuffix(file.Name(), ".json") // Remove suffix
		decString, baseErr := base64.StdEncoding.DecodeString(fileName)
		if baseErr != nil {
			cblogger.Errorf("Failed to Decode the Filename : %s", fileName)
			return nil, baseErr
		}
		sgFileName := string(decString)
		// cblogger.Infof("# S/G Group Name : [%s]", sgFileName)

		iid := &irs.IID{
			NameId:   sgFileName,
			SystemId: sgFileName,
		}
		iidList = append(iidList, iid)
	}
	return iidList, nil
}
