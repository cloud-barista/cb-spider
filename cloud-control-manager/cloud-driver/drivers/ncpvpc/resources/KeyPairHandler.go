// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCP VPC KeyPair Handler
//
// by ETRI, 2020.10.

package resources

import (
	"errors"
	"fmt"
	"strings"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"

	// "github.com/davecgh/go-spew/spew"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	keycommon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpVpcKeyPairHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VMClient       *vserver.APIClient
}

func (keyPairHandler *NcpVpcKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	cblogger.Info("NCP VPC cloud driver: called ListKey()!!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(keyPairHandler.RegionInfo.Zone, call.VMKEYPAIR, "ListKey()", "ListKey()")

	keypairReq := vserver.GetLoginKeyListRequest{
		KeyName: nil,
	}

	callLogStart := call.Start()
	result, err := keyPairHandler.VMClient.V2Api.GetLoginKeyList(&keypairReq)
	if err != nil {
		cblogger.Errorf("Failed to Get KeyPairList : %v", err)
		LoggingError(callLogInfo, err)
		return nil, err
	}
	LoggingInfo(callLogInfo, callLogStart)

	var keyPairList []*irs.KeyPairInfo
	if len(result.LoginKeyList) < 1 {
		cblogger.Info("### Keypair info does Not Exist!!")
	} else {
		for _, keyPair := range result.LoginKeyList {
			keyPairInfo := MappingKeyPairInfo(keyPair)
			keyPairList = append(keyPairList, &keyPairInfo)
		}
	}

	return keyPairList, nil
}

func (keyPairHandler *NcpVpcKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	cblogger.Info("NCP VPC cloud driver: called CreateKey()!!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(keyPairHandler.RegionInfo.Zone, call.VMKEYPAIR, keyPairReqInfo.IId.NameId, "CreateKey()")

	cblogger.Infof("KeyPairName to Create : [%s]", keyPairReqInfo.IId.NameId)

	// To Generate Hash
	strList := []string{
		keyPairHandler.CredentialInfo.ClientId,
		keyPairHandler.CredentialInfo.ClientSecret,
	}

	hashString, err := keycommon.GenHash(strList)
	if err != nil {
		cblogger.Errorf("Failed to Generate Hash String : %v", err)
		LoggingError(callLogInfo, err)
		return irs.KeyPairInfo{}, err
	}

	keypairReq := vserver.CreateLoginKeyRequest{
		KeyName: ncloud.String(keyPairReqInfo.IId.NameId),
	}

	// Creates a new  keypair with the given name
	callLogStart := call.Start()
	result, err := keyPairHandler.VMClient.V2Api.CreateLoginKey(&keypairReq)
	if err != nil {
		cblogger.Errorf("Failed to Create KeyPair: %s, %v.", keyPairReqInfo.IId.NameId, err)
		LoggingError(callLogInfo, err)
		return irs.KeyPairInfo{}, err
	}
	LoggingInfo(callLogInfo, callLogStart)
	cblogger.Infof("(# result.ReturnMessage : %s ", ncloud.StringValue(result.ReturnMessage))

	// Create PublicKey from PrivateKey
	publicKey, makePublicKeyErr := keycommon.MakePublicKeyFromPrivateKey(*result.PrivateKey)
	if makePublicKeyErr != nil {
		cblogger.Errorf("Failed to Create PublicKey from PrivateKey : %v", makePublicKeyErr)
		LoggingError(callLogInfo, makePublicKeyErr)
		return irs.KeyPairInfo{}, makePublicKeyErr
	}

	publicKey = strings.TrimSpace(publicKey) + " " + lnxUserName

	// Save the publicKey to DB in other to use on VMHandler(Cloud-init)
	addKeyErr := keycommon.AddKey("NCPVPC", hashString, keyPairReqInfo.IId.NameId, publicKey)
	if addKeyErr != nil {
		cblogger.Errorf("Failed to Save the PublicKey to DB : %v", addKeyErr)
		LoggingError(callLogInfo, addKeyErr)
		return irs.KeyPairInfo{}, addKeyErr
	}

	resultKey, keyError := keyPairHandler.GetKey(keyPairReqInfo.IId)
	if keyError != nil {
		cblogger.Errorf("Failed to Get the KeyPair Info : %v", keyError)
		LoggingError(callLogInfo, keyError)
		return irs.KeyPairInfo{}, keyError
	}

	// NCP Key does not have SystemId, so the unique NameId value is also applied to the SystemId
	keyPairInfo := irs.KeyPairInfo{
		IId:         irs.IID{NameId: keyPairReqInfo.IId.NameId, SystemId: keyPairReqInfo.IId.NameId},
		Fingerprint: resultKey.Fingerprint,
		PublicKey:   publicKey, // Made above
		PrivateKey:  *result.PrivateKey,
		VMUserID:    lnxUserName,
	}

	return keyPairInfo, nil
}

func (keyPairHandler *NcpVpcKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {
	cblogger.Info("NCP VPC cloud driver: called GetKey()!!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(keyPairHandler.RegionInfo.Zone, call.VMKEYPAIR, keyIID.NameId, "GetKey()")

	var keyNameId string
	if keyIID.SystemId == "" {
		keyNameId = keyIID.NameId
	}

	// NCP VPC Key does not have SystemId, so the unique NameId value is also applied to the SystemId when create it.
	keypairReq := vserver.GetLoginKeyListRequest{
		KeyName: ncloud.String(keyNameId),
	}

	callLogStart := call.Start()
	result, err := keyPairHandler.VMClient.V2Api.GetLoginKeyList(&keypairReq)
	if err != nil {
		newErr := fmt.Errorf("Failed to Find the KeyPair list from NCP VPC : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}
	LoggingInfo(callLogInfo, callLogStart)

	if len(result.LoginKeyList) < 1 {
		newErr := fmt.Errorf("Failed to Find Any KeyPair Info with the Name.")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	} else {
		keyPairInfo := MappingKeyPairInfo(result.LoginKeyList[0])
		return keyPairInfo, nil
	}

	return irs.KeyPairInfo{}, errors.New("Failed to Find KeyPair Info with the Name!!")
}

func (keyPairHandler *NcpVpcKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error) {
	cblogger.Info("NCP VPC cloud driver: called DeleteKey()!!")

	InitLog() // Caution!!
	callLogInfo := GetCallLogScheme(keyPairHandler.RegionInfo.Zone, call.VMKEYPAIR, keyIID.NameId, "DeleteKey()")

	cblogger.Infof("KeyPairName to Delete : [%s]", keyIID.NameId)
	// Delete the key pair by key 'Name'

	// To Generate Hash
	strList := []string{
		keyPairHandler.CredentialInfo.ClientId,
		keyPairHandler.CredentialInfo.ClientSecret,
	}
	hashString, err := keycommon.GenHash(strList)
	if err != nil {
		cblogger.Errorf("Failed to Gen Hash String : %v", err)
		LoggingError(callLogInfo, err)
		return false, err
	}

	// Because it succeeds unconditionally even if the corresponding keypair does not exist, the existence is checked in advance.
	_, keyError := keyPairHandler.GetKey(keyIID)
	if keyError != nil {
		cblogger.Errorf("There isn't any KeyPair with the Name : [%s]", keyIID.SystemId)
		cblogger.Error(keyError)
		return false, keyError
	}

	keypairDelReq := vserver.DeleteLoginKeysRequest{
		KeyNameList: []*string{
			ncloud.String(keyIID.NameId),
		},
	}

	// Caution!! : In case, server.~~
	// keypairDelReq := server.DeleteLoginKeyRequest{
	// 	KeyName: ncloud.String(keyIID.NameId),
	// }

	callLogStart := call.Start()
	result, err := keyPairHandler.VMClient.V2Api.DeleteLoginKeys(&keypairDelReq)
	if err != nil {
		cblogger.Errorf("Failed to Delete the KeyPair : %s, %v", keyIID.NameId, err)
		LoggingError(callLogInfo, err)
		return false, err
	}
	LoggingInfo(callLogInfo, callLogStart)

	// Delete the saved publicKey from DB
	delKeyErr := keycommon.DelKey("NCPVPC", hashString, keyIID.NameId)
	if delKeyErr != nil {
		cblogger.Errorf("Failed to Delete the Key info form DB : %v", delKeyErr)
		LoggingError(callLogInfo, delKeyErr)
		return false, delKeyErr
	}

	cblogger.Infof("(# result.ReturnMessage : %s ", ncloud.StringValue(result.ReturnMessage))
	cblogger.Infof("Succeeded in Deleting KeyPair Info : '%s' \n", keyIID.NameId)

	return true, nil
}

// KeyPair 정보를 추출함
func MappingKeyPairInfo(NcpKeyPairList *vserver.LoginKey) irs.KeyPairInfo {
	cblogger.Infof("*** Mapping KeyPair Info of : %s", *NcpKeyPairList.KeyName)

	// NCP Key does not have SystemId, so the unique NameId value is also applied to the SystemId
	keyPairInfo := irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   *NcpKeyPairList.KeyName,
			SystemId: *NcpKeyPairList.KeyName,
		},
		Fingerprint: *NcpKeyPairList.Fingerprint,
		PublicKey:   "N/A",
		// PublicKey:  	*NcpKeyPairList.PublicKey, // Creates Error
		PrivateKey: "N/A",
		VMUserID:   lnxUserName,
	}

	keyValueList := []irs.KeyValue{
		{Key: "CreateDate", Value: *NcpKeyPairList.CreateDate},
	}

	keyPairInfo.KeyValueList = keyValueList

	return keyPairInfo
}

func (keyPairHandler *NcpVpcKeyPairHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	return nil, errors.New("Does not support ListIID() yet!!")
}
