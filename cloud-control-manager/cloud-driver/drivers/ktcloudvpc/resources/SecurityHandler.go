// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// KT Cloud Security Group Handler
//
// by ETRI, 2022.12.

package resources

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	// "crypto/aes"
	// "crypto/cipher"
	"encoding/base64"
	// "github.com/davecgh/go-spew/spew"
	"encoding/json"
	// "strconv"

	ktvpcsdk "github.com/cloud-barista/ktcloudvpc-sdk-go"

	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type KTVpcSecurityHandler struct {
	RegionInfo    idrv.RegionInfo
	VMClient      *ktvpcsdk.ServiceClient
	NetworkClient *ktvpcsdk.ServiceClient
}

const (
	sgDir string = "/cloud-driver-libs/.securitygroup-kt/"
)

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("KT Cloud SecurityGroup Handler")
}

type SecurityGroup struct {
	IID        IId             `json:"IId"`
	VpcIID     VpcIId          `json:"VpcIID"`
	Direc      string          `json:"Direction"`
	Secu_Rules []Security_Rule `json:"SecurityRules"`
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

func (securityHandler *KTVpcSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	cblogger.Info("KT Cloud VPC driver: called CreateSecurity()!")
	callLogInfo := getCallLogScheme(securityHandler.RegionInfo.Zone, call.SECURITYGROUP, securityReqInfo.IId.NameId, "CreateSecurity()")

	if strings.EqualFold(securityHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Invalid Region Info!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}

	sgPath := os.Getenv("CBSPIDER_ROOT") + sgDir
	sgFilePath := sgPath + securityHandler.RegionInfo.Zone + "/"

	// Check if the KeyPair Folder Exists, and Create it
	if err := checkFolderAndCreate(sgPath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup Path : [%v]", err)
		return irs.SecurityInfo{}, err
	}

	// Check if the KeyPair Folder Exists, and Create it
	if err := checkFolderAndCreate(sgFilePath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup File Path : [%v]", err)
		return irs.SecurityInfo{}, err
	}

	// Check SecurityGroup Exists
	sgList, err := securityHandler.ListSecurity()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get S/G list. [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}
	for _, sg := range sgList {
		if sg.IId.NameId == securityReqInfo.IId.NameId {
			newErr := fmt.Errorf("Security Group with the Name [%s] Already Exists", securityReqInfo.IId.NameId)
			cblogger.Error(newErr.Error())
			loggingError(callLogInfo, newErr)
			return irs.SecurityInfo{}, newErr
		}
	}

	hashFileName := base64.StdEncoding.EncodeToString([]byte(securityReqInfo.IId.NameId))
	cblogger.Infof("# S/G NameId : " + securityReqInfo.IId.NameId)
	cblogger.Infof("# Hashed FileName : " + hashFileName + ".json")

	file, _ := json.MarshalIndent(securityReqInfo, "", " ")

	writeErr := os.WriteFile(sgFilePath+hashFileName+".json", file, 0644)
	if writeErr != nil {
		cblogger.Error("Failed to write the file: "+sgFilePath+hashFileName+".json", writeErr)
		return irs.SecurityInfo{}, writeErr
	}

	cblogger.Infof("Succeeded in writing the S/G file: " + sgFilePath + hashFileName + ".json")

	// Because it's managed as a file, there's no SystemId created.
	securityReqInfo.IId.SystemId = securityReqInfo.IId.NameId

	// Return the created SecurityGroup info.
	securityInfo, err := securityHandler.GetSecurity(irs.IID{SystemId: securityReqInfo.IId.SystemId})
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	return securityInfo, nil
}

func (securityHandler *KTVpcSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	cblogger.Info("KT Cloud VPC driver: called GetSecurity()!!")
	callLogInfo := getCallLogScheme(securityHandler.RegionInfo.Zone, call.SECURITYGROUP, securityIID.SystemId, "GetSecurity()")

	var sg SecurityGroup
	securityIID.NameId = securityIID.SystemId
	hashFileName := base64.StdEncoding.EncodeToString([]byte(securityIID.NameId))

	cblogger.Infof("# securityIID.NameId : " + securityIID.NameId)
	cblogger.Infof("# hashFileName : " + hashFileName + ".json")

	if strings.EqualFold(securityHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Invalid Region Info!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.SecurityInfo{}, newErr
	}

	sgPath := os.Getenv("CBSPIDER_ROOT") + sgDir
	sgFilePath := sgPath + securityHandler.RegionInfo.Zone + "/"

	// Check if the KeyPair Folder Exists, and Create it
	if err := checkFolderAndCreate(sgPath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup Path : [%v]", err)
		return irs.SecurityInfo{}, err
	}

	// Check if the KeyPair Folder Exists, and Create it
	if err := checkFolderAndCreate(sgFilePath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup File Path : [%v]", err)
		return irs.SecurityInfo{}, err
	}

	sgFileName := sgFilePath + hashFileName + ".json"
	jsonFile, err := os.Open(sgFileName)
	if err != nil {
		cblogger.Error("Failed to Find the S/G file : "+sgFileName+" ", err)
		return irs.SecurityInfo{}, err
	}
	cblogger.Infof("Succeeded in Finding and Opening the S/G file: " + sgFileName)

	defer jsonFile.Close()
	byteValue, readErr := io.ReadAll(jsonFile)
	if readErr != nil {
		cblogger.Error("Failed to Read the S/G file : "+sgFileName, readErr)
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

func (securityHandler *KTVpcSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	cblogger.Info("KT Cloud VPC driver: called ListSecurity()!!")
	callLogInfo := getCallLogScheme(securityHandler.RegionInfo.Zone, call.SECURITYGROUP, "ListSecurity()", "ListSecurity()")

	var securityIID irs.IID
	var securityGroupList []*irs.SecurityInfo
	// var sg SecurityGroup

	if strings.EqualFold(securityHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Invalid Region Info!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}

	sgPath := os.Getenv("CBSPIDER_ROOT") + sgDir
	sgFilePath := sgPath + securityHandler.RegionInfo.Zone + "/"

	// Check if the KeyPair Folder Exists, and Create it
	if err := checkFolderAndCreate(sgPath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup Path : [%v]", err)
		return nil, err
	}

	// Check if the KeyPair Folder Exists, and Create it
	if err := checkFolderAndCreate(sgFilePath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup File Path : [%v]", err)
		return nil, err
	}

	// File list on the local directory
	dirFiles, readRrr := os.ReadDir(sgFilePath)
	if readRrr != nil {
		return nil, readRrr
	}

	for _, file := range dirFiles {
		fileName := strings.TrimSuffix(file.Name(), ".json") // 접미사 제거
		decString, baseErr := base64.StdEncoding.DecodeString(fileName)
		if baseErr != nil {
			cblogger.Errorf("Failed to Decode the Filename : %s", fileName)
			return nil, baseErr
		}
		sgFileName := string(decString)

		// sgFileName := filePath + file.Name()

		securityIID.SystemId = sgFileName
		cblogger.Infof("# S/G Group Name : " + securityIID.SystemId)

		sgInfo, err := securityHandler.GetSecurity(irs.IID{SystemId: securityIID.SystemId})
		if err != nil {
			cblogger.Errorf("Failed to Find the SecurityGroup : %s", securityIID.SystemId)
			return nil, err
		}
		securityGroupList = append(securityGroupList, &sgInfo)
	}

	return securityGroupList, nil
}

func (securityHandler *KTVpcSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud VPC driver: called DeleteSecurity()!")
	callLogInfo := getCallLogScheme(securityHandler.RegionInfo.Zone, call.SECURITYGROUP, securityIID.SystemId, "DeleteSecurity()")

	securityIID.NameId = securityIID.SystemId

	if strings.EqualFold(securityHandler.RegionInfo.Zone, "") {
		newErr := fmt.Errorf("Invalid Region Info!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	sgPath := os.Getenv("CBSPIDER_ROOT") + sgDir
	sgFilePath := sgPath + securityHandler.RegionInfo.Zone + "/"

	// Check if the KeyPair Folder Exists, and Create it
	if err := checkFolderAndCreate(sgPath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup Path : [%v]", err)
		return false, err
	}

	// Check if the KeyPair Folder Exists, and Create it
	if err := checkFolderAndCreate(sgFilePath); err != nil {
		cblogger.Errorf("Failed to Create the SecurityGroup File Path : [%v]", err)
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
	cblogger.Infof("Succeeded in Deleting the SecurityGroup : " + securityIID.SystemId)

	return true, nil
}

func (securityHandler *KTVpcSecurityHandler) AddRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	cblogger.Info("KT Cloud VPC driver: called AddRules()!")
	return irs.SecurityInfo{}, fmt.Errorf("Does not support AddRules() yet!!")
}

func (securityHandler *KTVpcSecurityHandler) RemoveRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (bool, error) {
	cblogger.Info("KT Cloud VPC sriver: called RemoveRules()!")
	return false, fmt.Errorf("Does not support RemoveRules() yet!!")
}

func (securityHandler *KTVpcSecurityHandler) mappingSecurityInfo(secuGroup SecurityGroup) (irs.SecurityInfo, error) {
	cblogger.Info("KT Cloud VPC driver: called mappingSecurityInfo()!")

	var securityRuleList []irs.SecurityRuleInfo
	var securityRuleInfo irs.SecurityRuleInfo

	for i := 0; i < len(secuGroup.Secu_Rules); i++ {
		securityRuleInfo.FromPort = secuGroup.Secu_Rules[i].FromPort
		securityRuleInfo.ToPort = secuGroup.Secu_Rules[i].ToPort
		securityRuleInfo.IPProtocol = secuGroup.Secu_Rules[i].Protocol // For KT Cloud VPC S/G, TCP/UDP/ICMP is available
		securityRuleInfo.Direction = secuGroup.Secu_Rules[i].Direc     // For KT Cloud VPC S/G, supports inbound/outbound rule.
		securityRuleInfo.CIDR = secuGroup.Secu_Rules[i].Cidr

		securityRuleList = append(securityRuleList, securityRuleInfo)
	}

	securityInfo := irs.SecurityInfo{
		IId: irs.IID{NameId: secuGroup.IID.NameID, SystemId: secuGroup.IID.NameID},
		// Since it is managed as a file, the systemID is the same as the name ID.
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

func (securityHandler *KTVpcSecurityHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	return nil, errors.New("Does not support ListIID() yet!!")
}
