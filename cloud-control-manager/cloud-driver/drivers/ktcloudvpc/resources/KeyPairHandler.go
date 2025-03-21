// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2022.08.
// Updated by ETRI, 2025.02.

package resources

import (
	"fmt"
	"strings"
	// "github.com/davecgh/go-spew/spew"

	ktvpcsdk "github.com/cloud-barista/ktcloudvpc-sdk-go"
	keys "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/compute/v2/extensions/keypairs"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	keycommon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type KTVpcKeyPairHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VMClient       *ktvpcsdk.ServiceClient
}

func (keyPairHandler *KTVpcKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called CreateKey()")
	callLogInfo := getCallLogScheme(keyPairHandler.RegionInfo.Zone, call.VMKEYPAIR, keyPairReqInfo.IId.NameId, "CreateKey()")

	if strings.EqualFold(keyPairReqInfo.IId.NameId, "") {
		newErr := fmt.Errorf("Invalid KeyPair Name!!")
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}

	exist, err := keyPairHandler.keyPairExists(keyPairReqInfo.IId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create Key. : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}
	if exist {
		keyName := keyPairReqInfo.IId.SystemId
		if strings.EqualFold(keyPairReqInfo.IId.SystemId, "") {
			keyName = keyPairReqInfo.IId.NameId
		}
		newErr := fmt.Errorf("Failed to Create Key. The Key name [%s] already exists", keyName)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}

	create0pts := keys.CreateOpts{
		Name: keyPairReqInfo.IId.NameId,
	}

	start := call.Start()
	keyPair, err := keys.Create(keyPairHandler.VMClient, create0pts).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Create Key. : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}
	loggingInfo(callLogInfo, start)

	// # Save the publicKey to DB in other to use on VMHandler(Cloud-init)
	publicKey := strings.TrimSpace(keyPair.PublicKey) + " " + LnxUserName // Append VM User Name

	strList := []string{
		keyPairHandler.CredentialInfo.Username,
		keyPairHandler.CredentialInfo.Password,
	}
	hashString, err := keycommon.GenHash(strList)
	if err != nil {
		newErr := fmt.Errorf("Failed to Generate Hash String : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}

	addKeyErr := keycommon.AddKey("KTCLOUDVPC", hashString, keyPairReqInfo.IId.NameId, publicKey)
	if addKeyErr != nil {
		newErr := fmt.Errorf("Failed to Save the Private Key to DB : [%v]", addKeyErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}

	// Returns the value of key only when created.
	keyPairInfo := mappingKeyPairInfo(*keyPair)
	keyPairInfo.PublicKey = keyPair.PublicKey
	keyPairInfo.PrivateKey = keyPair.PrivateKey

	return *keyPairInfo, nil
}

func (keyPairHandler *KTVpcKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called ListKey()")
	callLogInfo := getCallLogScheme(keyPairHandler.RegionInfo.Zone, call.VMKEYPAIR, "ListKey()", "ListKey()")

	var listOptsBuilder keys.ListOptsBuilder
	start := call.Start()
	allPages, err := keys.List(keyPairHandler.VMClient, listOptsBuilder).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KeyPair Pages from KT Cloud. : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	loggingInfo(callLogInfo, start)

	keys, err := keys.ExtractKeyPairs(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KeyPair list. : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return nil, newErr
	}
	keyPairList := make([]*irs.KeyPairInfo, len(keys))
	for i, key := range keys {
		keyPairList[i] = mappingKeyPairInfo(key)
	}
	return keyPairList, nil
}

func (keyPairHandler *KTVpcKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {
	cblogger.Info("KT Cloud VPC Driver: called GetKey()")
	callLogInfo := getCallLogScheme(keyPairHandler.RegionInfo.Zone, call.VMKEYPAIR, keyIID.NameId, "GetKey()")

	var keyNameId string
	if strings.EqualFold(keyIID.SystemId, "") {
		keyNameId = keyIID.NameId
	} else {
		keyNameId = keyIID.SystemId
	}

	var keyPair *keys.KeyPair
	var getOptsBuilder keys.GetOptsBuilder

	start := call.Start()
	keyPair, err := keys.Get(keyPairHandler.VMClient, keyNameId, getOptsBuilder).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get the KeyPair info from KT Cloud : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}
	loggingInfo(callLogInfo, start)

	keyPairInfo := mappingKeyPairInfo(*keyPair)
	return *keyPairInfo, nil
}

func (keyPairHandler *KTVpcKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called DeleteKey()")
	callLogInfo := getCallLogScheme(keyPairHandler.RegionInfo.Zone, call.VMKEYPAIR, keyIID.NameId, "DeleteKey()")

	exist, err := keyPairHandler.keyPairExists(keyIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Delete the KeyPair. : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}
	if !exist {
		var keyNameId string
		if strings.EqualFold(keyIID.SystemId, "") {
			keyNameId = keyIID.NameId
		} else {
			keyNameId = keyIID.SystemId
		}

		newErr := fmt.Errorf("Failed to Delete the KeyPair. The Key name [%s] Not found", keyNameId)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	var delOptsBuilder keys.DeleteOptsBuilder
	start := call.Start()
	delErr := keys.Delete(keyPairHandler.VMClient, keyIID.NameId, delOptsBuilder).ExtractErr()
	if delErr != nil {
		newErr := fmt.Errorf("Failed to Delete the KeyPair : [%v]", delErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}
	loggingInfo(callLogInfo, start)

	// # Delete the Key on DB
	strList := []string{
		keyPairHandler.CredentialInfo.Username,
		keyPairHandler.CredentialInfo.Password,
	}
	hashString, err := keycommon.GenHash(strList)
	if err != nil {
		newErr := fmt.Errorf("Failed to Generate Hash String : [%v]", err)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}

	// Delete the saved publicKey from DB
	delKeyErr := keycommon.DelKey("KTCLOUDVPC", hashString, keyIID.NameId)
	if delKeyErr != nil {
		newErr := fmt.Errorf("Failed to Delete the KeyPair info form DB : [%v]", delKeyErr)
		cblogger.Error(newErr.Error())
		loggingError(callLogInfo, newErr)
		return false, newErr
	}
	cblogger.Infof("Succeeded in Deleting Process of the KeyPair : [%s]\n", keyIID.NameId)

	return true, nil
}

func mappingKeyPairInfo(keypair keys.KeyPair) *irs.KeyPairInfo {
	cblogger.Info("KT Cloud VPC Driver: called mappingKeyPairInfo()")

	keyPairInfo := &irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   	keypair.Name,
			SystemId: 	keypair.Name,
		},
		Fingerprint: 	keypair.Fingerprint,
		PublicKey:   	"NA",
		PrivateKey:  	"NA",
		VMUserID:    	LnxUserName,
		KeyValueList:   irs.StructToKeyValueList(keypair),
	}

	// For security
	for i, kv := range keyPairInfo.KeyValueList {
		if kv.Key == "PublicKey" {
			keyPairInfo.KeyValueList[i].Value = "NA"
			break
		}
	}

	return keyPairInfo
}

func (keyPairHandler *KTVpcKeyPairHandler) keyPairExists(keyIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud VPC Driver: called keyPairExists()")

	var keyNameId string
	if strings.EqualFold(keyIID.SystemId, "") {
		keyNameId = keyIID.NameId
	} else {
		keyNameId = keyIID.SystemId
	}

	var listOptsBuilder keys.ListOptsBuilder
	allPages, err := keys.List(keyPairHandler.VMClient, listOptsBuilder).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KeyPair Pages from KT Cloud. : [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}
	keypairList, err := keys.ExtractKeyPairs(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KeyPair list. : [%v]", err)
		cblogger.Error(newErr.Error())
		return false, newErr
	}
	for _, keypair := range keypairList {
		if strings.EqualFold(keypair.Name, keyNameId) {
			return true, nil
		}
	}

	return false, nil
}

func (keyPairHandler *KTVpcKeyPairHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("KT Cloud VPC driver: called ListIID()!!")

    var listOptsBuilder keys.ListOptsBuilder
    allPages, err := keys.List(keyPairHandler.VMClient, listOptsBuilder).AllPages()
    if err != nil {
        newErr := fmt.Errorf("Failed to Get KeyPair Pages from KT Cloud. : [%v]", err)
        cblogger.Error(newErr.Error())
        return nil, newErr
    }

    keyPairs, err := keys.ExtractKeyPairs(allPages)
    if err != nil {
        newErr := fmt.Errorf("Failed to Get KeyPair list. : [%v]", err)
        cblogger.Error(newErr.Error())
        return nil, newErr
    }

    var iidList []*irs.IID
    for _, keyPair := range keyPairs {
        iid := &irs.IID{
            NameId:   keyPair.Name,
            SystemId: keyPair.Name,
        }
        iidList = append(iidList, iid)
    }
    return iidList, nil
}
