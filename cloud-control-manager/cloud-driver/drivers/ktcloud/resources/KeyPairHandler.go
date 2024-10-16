// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// KT Cloud KeyPair Handler
//
// by ETRI, 2021.05.

package resources

import (
	"errors"
	"fmt"
	"strings"

	// "github.com/davecgh/go-spew/spew"

	ktsdk "github.com/cloud-barista/ktcloud-sdk-go"

	cblog "github.com/cloud-barista/cb-log"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	keycommon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type KtCloudKeyPairHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	Client         *ktsdk.KtCloudClient
}

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("KT Cloud KeyPair Handler")
}

func (keyPairHandler *KtCloudKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called ListKey()!!")

	keyPairName := ""
	result, err := keyPairHandler.Client.ListSSHKeyPairs(keyPairName)
	if err != nil {
		cblogger.Errorf("Failed to Get the KeyPairList : ", err)
		//spew.Dump(err)
		return []*irs.KeyPairInfo{}, err
	}
	// spew.Dump(result)

	if result.Listsshkeypairsresponse.Count < 1 {
		// if len(result.Listsshkeypairsresponse.Keypair) < 1 {
		cblogger.Info("KeyPair does not exit on the zone!!")
		return nil, nil // Caution!!
	}

	//cblogger.Debugf("Key Pairs:")
	var keyPairList []*irs.KeyPairInfo
	for _, keyPair := range result.Listsshkeypairsresponse.KeyPair {
		keyPairInfo := mappingKeyPairInfo(keyPair)
		keyPairList = append(keyPairList, &keyPairInfo)
	}

	cblogger.Debug(keyPairList)
	//spew.Dump(keyPairList)
	return keyPairList, nil
}

func (keyPairHandler *KtCloudKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called CreateKey()!!")
	InitLog()
	callLogInfo := GetCallLogScheme(keyPairHandler.RegionInfo.Zone, call.VMKEYPAIR, keyPairReqInfo.IId.NameId, "CreateKey()")
	cblogger.Info(keyPairReqInfo)

	//***** Make sure that Keypair Name already exists *****
	resultKey, keyGetError := keyPairHandler.GetKey(keyPairReqInfo.IId)
	if keyGetError != nil {
		cblogger.Debug("The KeyPair with the Name does't exit!!: [%v]", keyGetError)
		// spew.Dump(keyGetError)
	}

	if resultKey.Fingerprint == "" {
		cblogger.Infof("# You can Create the KeyPair with the Name!!")
	} else {
		return irs.KeyPairInfo{}, errors.New("The KeyPair name already exists!!")
	}

	// Creates a new key pair with the given name
	callLogStart := call.Start()
	createResult, err := keyPairHandler.Client.CreateSSHKeyPair(keyPairReqInfo.IId.NameId)
	if err != nil {
		cblogger.Errorf("Failed to Create KeyPair: %s, %v.", keyPairReqInfo.IId.NameId, err)
		return irs.KeyPairInfo{}, err
	}
	LoggingInfo(callLogInfo, callLogStart)
	// spew.Dump(result)
	// cblogger.Infof("Created Private key \n%s\n", result.Createsshkeypairresponse.KeyPair.PrivateKey)

	// Create PublicKey from PrivateKey
	publicKey, makePublicKeyErr := keycommon.MakePublicKeyFromPrivateKey(createResult.Createsshkeypairresponse.KeyPair.PrivateKey)
	if makePublicKeyErr != nil {
		newErr := fmt.Errorf("Failed to Generated the Public Key : [%v]", makePublicKeyErr)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}
	publicKey = strings.TrimSpace(publicKey) + " " + LinuxUserName // Append VM User Name

	strList := []string{
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

	// Save the publicKey to DB in other to use on VMHandler(Cloud-init)
	addKeyErr := keycommon.AddKey("KTCLOUD", hashString, keyPairReqInfo.IId.NameId, publicKey)
	if addKeyErr != nil {
		newErr := fmt.Errorf("Failed to Save the Private Key to DB : [%v]", addKeyErr)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}

	//Since KT Cloud does not have a SystemID, the unique nameID value is also input to the SystemID
	keyPairInfo := irs.KeyPairInfo{
		IId:         irs.IID{NameId: keyPairReqInfo.IId.NameId, SystemId: keyPairReqInfo.IId.NameId},
		Fingerprint: createResult.Createsshkeypairresponse.KeyPair.Fingerprint,
		PublicKey:   publicKey,                                                // Generated above. Show only when KeyPair Creation.
		PrivateKey:  createResult.Createsshkeypairresponse.KeyPair.PrivateKey, // Show only when KeyPair Creation.
		VMUserID:    LinuxUserName,
	}
	return keyPairInfo, nil
}

func (keyPairHandler *KtCloudKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called GetKey()!!")

	cblogger.Infof("keyName : [%s]", keyIID.NameId)
	result, err := keyPairHandler.Client.ListSSHKeyPairs(keyIID.NameId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the KeyPair with the keyName : ", err)
		cblogger.Error(newErr.Error())
		return irs.KeyPairInfo{}, newErr
	}
	// spew.Dump(result)

	var resultKeyPairInfo irs.KeyPairInfo
	if result.Listsshkeypairsresponse.Count < 1 {
		errors.New("Failed to Find KeyPair with the Name!!")
		return irs.KeyPairInfo{}, errors.New("Failed to Find KeyPair with the Name!!")

	} else {
		resultKeyPairInfo = mappingKeyPairInfo(result.Listsshkeypairsresponse.KeyPair[0])
		// spew.Dump(result.Listsshkeypairsresponse.Keypair[0])
	}
	return resultKeyPairInfo, nil
}

func (keyPairHandler *KtCloudKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error) {
	cblogger.Debug("Start DeleteKey()")
	InitLog()
	callLogInfo := GetCallLogScheme(keyPairHandler.RegionInfo.Zone, call.VMKEYPAIR, keyIID.NameId, "DeleteKey()")

	cblogger.Infof("DeleteKey(KeyPairName) : [%s]", keyIID.NameId)
	// Delete the key pair by key 'NameID'

	var keyNameId string
	if keyIID.SystemId == "" {
		keyNameId = keyIID.NameId
	} else {
		keyNameId = keyIID.SystemId
	}

	//It is necessary to check in advance because it succeeds unconditionally without Keypair.
	_, keyError := keyPairHandler.GetKey(keyIID)
	if keyError != nil {
		newErr := fmt.Errorf("Failed to Get the KeyPair : [%s], [%v]", keyIID.SystemId, keyError)
		cblogger.Error(newErr.Error())
		return false, newErr
	}

	result, err := keyPairHandler.Client.DeleteSSHKeyPair(keyIID.NameId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the KeyPair : %s, %v", keyIID.NameId, err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}
	cblogger.Infof("Deletion result on KT Cloud : %s", result.Deletesshkeypairresponse.Success)

	strList := []string{
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

	// Delete the saved publicKey from DB
	delKeyErr := keycommon.DelKey("KTCLOUD", hashString, keyNameId)
	if delKeyErr != nil {
		newErr := fmt.Errorf("Failed to Delete the KeyPair info form DB : [%v]", delKeyErr)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	cblogger.Infof("Succeeded in Deleting Process of the KeyPair : [%s]\n", keyNameId)

	return true, nil
}

func mappingKeyPairInfo(KtCloudKeyPair ktsdk.KeyPair) irs.KeyPairInfo {
	cblogger.Info("KT Cloud Driver: called mappingKeyPairInfo()")
	// Since KT Cloud does not have a SystemID, the unique nameID value is also input to the SystemID
	keyPairInfo := irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   KtCloudKeyPair.Name,
			SystemId: KtCloudKeyPair.Name,
		},
		Fingerprint: KtCloudKeyPair.Fingerprint,
		PublicKey:   "N/A",
		PrivateKey:  "N/A",
		VMUserID:    LinuxUserName,

		// PrivateKey: KtCloudKeyPairList.Privatekey,
		// Note) KT Cloud에서 KtCloud KeyPairList 조회시에는 response에 Private key값은 없음.
		// CreateSSHKeyPair() 할때만 response로 받음.
	}
	return keyPairInfo
}

func (keyPairHandler *KtCloudKeyPairHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	return nil, errors.New("Does not support ListIID() yet!!")
}
