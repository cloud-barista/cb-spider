// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP Classic Cloud KeyPair Handler
//
// Created by ETRI, 2020.09.
// Updated by ETRI, 2022.09.
// Updated by ETRI, 2023.09.

package resources

import (
	"fmt"
	"errors"
	"strings"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/server"
	// "github.com/davecgh/go-spew/spew"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	keycommon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
)

type NcpKeyPairHandler struct {
	CredentialInfo 		idrv.CredentialInfo
	RegionInfo     		idrv.RegionInfo
	VMClient         	*server.APIClient
}

func (keyPairHandler *NcpKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	cblogger.Info("NCP Classic Cloud driver: called ListKey()!!")

	InitLog()
	callLogInfo := GetCallLogScheme(keyPairHandler.RegionInfo.Zone, call.VMKEYPAIR, "ListKey()", "ListKey()")

	keypairReq := server.GetLoginKeyListRequest{
		KeyName: nil,
	}
	callLogStart := call.Start()
	result, err := keyPairHandler.VMClient.V2Api.GetLoginKeyList(&keypairReq) // *server.APIClient
	if err != nil {
		newErr := fmt.Errorf("Failed to Find KeyPairList from NCP Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	var keyPairList []*irs.KeyPairInfo
	if len(result.LoginKeyList) > 0 {
		for _, keyPair := range result.LoginKeyList {
			keyPairInfo := MappingKeyPairInfo(keyPair)
			keyPairList = append(keyPairList, &keyPairInfo)
		}	
		return keyPairList, nil
	} else {
		return nil, errors.New("Failed to Find KeyPairList!!")
	}
}

func (keyPairHandler *NcpKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	cblogger.Info("NCP Classic Cloud driver: called CreateKey()!!")

	InitLog()
	callLogInfo := GetCallLogScheme(keyPairHandler.RegionInfo.Zone, call.VMKEYPAIR, keyPairReqInfo.IId.NameId, "CreateKey()")

	strList:= []string{
		keyPairHandler.CredentialInfo.ClientId,
		keyPairHandler.CredentialInfo.ClientSecret,
	}
	hashString, err := keycommon.GenHash(strList)
	if err != nil {
		newErr := fmt.Errorf("Failed to Generate Hash String : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}

	keypairReq := server.CreateLoginKeyRequest{
		KeyName: ncloud.String(keyPairReqInfo.IId.NameId),
	}
	callLogStart := call.Start()
	// Creates a new key pair with the given name
	result, err := keyPairHandler.VMClient.V2Api.CreateLoginKey(&keypairReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create KeyPair: [%s], [%v]", keyPairReqInfo.IId.NameId, err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)
	cblogger.Infof("(# result.ReturnMessage : %s ", ncloud.StringValue(result.ReturnMessage))		

	// Create PublicKey from PrivateKey
	publicKey, makePublicKeyErr := keycommon.MakePublicKeyFromPrivateKey(*result.PrivateKey)
	if makePublicKeyErr != nil {
		newErr := fmt.Errorf("Failed to Generated the Public Key : [%v]", makePublicKeyErr)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}
	publicKey = strings.TrimSpace(publicKey) + " " + lnxUserName // Append VM User Name

	// Save the publicKey to DB in other to use on VMHandler(Cloud-init)
	addKeyErr := keycommon.AddKey("NCP", hashString, keyPairReqInfo.IId.NameId, publicKey)
	if addKeyErr != nil {
		newErr := fmt.Errorf("Failed to Save the Private Key to DB : [%v]", addKeyErr)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}

	resultKey, keyError := keyPairHandler.GetKey(keyPairReqInfo.IId)
	if keyError != nil {
		newErr := fmt.Errorf("Failed to Get the KeyPair Info : [%v]", keyError)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}

	// NCP Key does not have SystemId, so the unique NameId value is also applied to the SystemId
	keyPairInfo := irs.KeyPairInfo{
		IId:         irs.IID{NameId: keyPairReqInfo.IId.NameId, SystemId: keyPairReqInfo.IId.NameId},
		Fingerprint: resultKey.Fingerprint,
		PublicKey:   publicKey, 		 // Generated above. Show only when KeyPair Creation.
		PrivateKey:  *result.PrivateKey, // Show only when KeyPair Creation.
		VMUserID: 	 lnxUserName,
	}
	return keyPairInfo, nil
}

func (keyPairHandler *NcpKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {
	cblogger.Info("NCP Classic Cloud driver: called GetKey()!!")

	InitLog()
	callLogInfo := GetCallLogScheme(keyPairHandler.RegionInfo.Zone, call.VMKEYPAIR, keyIID.SystemId, "GetKey()")

	var keyNameId string
	if keyIID.SystemId == "" {
		keyNameId = keyIID.NameId
	} else {
		keyNameId = keyIID.SystemId
	}

	keypairReq := server.GetLoginKeyListRequest{ // server.GetLoginKey~~~
		KeyName: ncloud.String(keyNameId), 	// Caution!!
	}
	callLogStart := call.Start()
	result, err := keyPairHandler.VMClient.V2Api.GetLoginKeyList(&keypairReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find the KeyPair Info from NCP Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.LoginKeyList) > 0 {
		keyPairInfo := MappingKeyPairInfo(result.LoginKeyList[0])
		return keyPairInfo, nil
	} else {
		return irs.KeyPairInfo{}, errors.New("Failed to Find the KeyPair info with the Name!!")
	}
}

func (keyPairHandler *NcpKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error) {
	cblogger.Info("NCP Classic Cloud driver: called DeleteKey()!!")

	InitLog()
	callLogInfo := GetCallLogScheme(keyPairHandler.RegionInfo.Zone, call.VMKEYPAIR, keyIID.NameId, "DeleteKey()")

	var keyNameId string
	if keyIID.SystemId == "" {
		keyNameId = keyIID.NameId
	} else {
		keyNameId = keyIID.SystemId
	}
	cblogger.Infof("KeyPairName to Delete : [%s]", keyNameId)
	// Delete the key pair by key 'Name'

	strList:= []string{
		keyPairHandler.CredentialInfo.ClientId,
		keyPairHandler.CredentialInfo.ClientSecret,
	}
	hashString, err := keycommon.GenHash(strList)
	if err != nil {
		newErr := fmt.Errorf("Failed to Generate Hash String : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	// Because it succeeds unconditionally even if the corresponding keypair does not exist, the existence is checked in advance.
	_, keyError := keyPairHandler.GetKey(keyIID)
	if keyError != nil {
		newErr := fmt.Errorf("Failed to Get the KeyPair with the Name : [%s], [%v]", keyNameId, keyError)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	keypairDelReq := server.DeleteLoginKeyRequest{
		ncloud.String(keyNameId), // Caution!!
	}
	callLogStart := call.Start()
	result, err := keyPairHandler.VMClient.V2Api.DeleteLoginKey(&keypairDelReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the KeyPair with the keyNameId : [%s], [%v]", keyNameId, err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)
	cblogger.Infof("(# result.ReturnMessage : [%s] ", ncloud.StringValue(result.ReturnMessage))

	// Delete the saved publicKey from DB
	delKeyErr := keycommon.DelKey("NCP", hashString, keyNameId)
	if delKeyErr != nil {
		newErr := fmt.Errorf("Failed to Delete the KeyPair info form DB : [%v]", delKeyErr)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	cblogger.Infof("Succeeded in Deleting Process of the KeyPair : [%s]\n", keyNameId)
	return true, nil
}

// KeyPair 정보를 추출함
func MappingKeyPairInfo(NcpKeyPairList *server.LoginKey) irs.KeyPairInfo {
	cblogger.Info("NCP Classic Cloud driver: called MappingKeyPairInfo()!!")

	// Note) NCP Key does not have SystemId, so the unique NameId value is also applied to the SystemId
	keyPairInfo := irs.KeyPairInfo{
		IId:         irs.IID{
			NameId:   *NcpKeyPairList.KeyName, 
			SystemId: *NcpKeyPairList.KeyName,
		},
		Fingerprint:  *NcpKeyPairList.Fingerprint,
		PublicKey:    "N/A",
		PrivateKey:   "N/A",
		VMUserID: 	  lnxUserName,
	}

	keyValueList := []irs.KeyValue{
		{Key: "CreateDate", Value: *NcpKeyPairList.CreateDate},
	}
	keyPairInfo.KeyValueList = keyValueList
	return keyPairInfo
}
