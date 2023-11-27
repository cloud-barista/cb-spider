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
	ktsdk "github.com/cloud-barista/ktcloud-sdk-go"

	"errors"
	"github.com/davecgh/go-spew/spew"

	cblog "github.com/cloud-barista/cb-log"
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
		keyPairInfo := MappingKeyPairInfo(keyPair)
		keyPairList = append(keyPairList, &keyPairInfo)
	}

	cblogger.Debug(keyPairList)
	//spew.Dump(keyPairList)
	return keyPairList, nil
}

func (keyPairHandler *KtCloudKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called CreateKey()!!")
	cblogger.Info(keyPairReqInfo)
	
	//***** Make sure that Keypair Name already exists *****
	resultKey, keyGetError := keyPairHandler.GetKey(keyPairReqInfo.IId)
	if keyGetError != nil {
		cblogger.Errorf("The KeyPair with the Name does't exit!!: ", keyGetError)
		// spew.Dump(keyGetError)
	}

	if resultKey.Fingerprint == "" {
		cblogger.Infof("# You can Create the KeyPair with the Name!!")

		// Creates a new key pair with the given name on the NCP platform
		result, err := keyPairHandler.Client.CreateSSHKeyPair(keyPairReqInfo.IId.NameId)
		if err != nil {
			cblogger.Errorf("Failed to Create KeyPair: %s, %v.", keyPairReqInfo.IId.NameId, err)
			return irs.KeyPairInfo{}, err
		}

		//spew.Dump(result)

		//cblogger.Infof("Created Private key \n%s\n", result.Createsshkeypairresponse.KeyPair.PrivateKey)

		//Since KT Cloud does not have a SystemID, the unique nameID value is also input to the SystemID
		keyPairInfo := irs.KeyPairInfo{
			IId:        irs.IID{NameId: keyPairReqInfo.IId.NameId, SystemId: keyPairReqInfo.IId.NameId},
			Fingerprint: result.Createsshkeypairresponse.KeyPair.Fingerprint,
			PrivateKey: result.Createsshkeypairresponse.KeyPair.PrivateKey,
			PublicKey:  "N/A",
			VMUserID: vmUserName,
		}
		return keyPairInfo, nil
	}
	return irs.KeyPairInfo{}, errors.New("The KeyPair name already exists!!")
}

func (keyPairHandler *KtCloudKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {
	cblogger.Info("KT Cloud cloud driver: called GetKey()!!")

	cblogger.Infof("keyName : [%s]", keyIID.NameId)
	var resultKeyPairInfo irs.KeyPairInfo

	result, err := keyPairHandler.Client.ListSSHKeyPairs(keyIID.NameId)
	if err != nil {
		cblogger.Errorf("Failed to Get the KeyPair with the keyName : ", err)
		//spew.Dump(err)
		return irs.KeyPairInfo{}, err
	}

	// spew.Dump(result)

	if result.Listsshkeypairsresponse.Count < 1 {
	// if len(result.Listsshkeypairsresponse.Keypair) < 1 {
		errors.New("Failed to Find KeyPair with the Name!!")
		return irs.KeyPairInfo{}, errors.New("Failed to Find KeyPair with the Name!!")

	} else 
	{
		// spew.Dump(result.Listsshkeypairsresponse.Keypair[0])
		resultKeyPairInfo = MappingKeyPairInfo(result.Listsshkeypairsresponse.KeyPair[0])
	}

	return resultKeyPairInfo, nil
}


func (keyPairHandler *KtCloudKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error) {

	cblogger.Debug("Start DeleteKey()")

	cblogger.Infof("DeleteKey(KeyPairName) : [%s]", keyIID.NameId)
	// Delete the key pair by key 'NameID'

	//It is necessary to check in advance because it succeeds unconditionally without Keypair.
	_, keyError := keyPairHandler.GetKey(keyIID)
	if keyError != nil {
		cblogger.Errorf("Failed to Get the KeyPair : [%s]", keyIID.SystemId)
		cblogger.Error(keyError)
		return false, keyError
	}

	result, err := keyPairHandler.Client.DeleteSSHKeyPair(keyIID.NameId)
	if err != nil {
		cblogger.Errorf("Failed to Delete the KeyPair : %s, %v", keyIID.NameId, err)
		spew.Dump(err)
		return false, err
	}

	spew.Dump(result)

	cblogger.Infof("Succeeded in Deleting the KeyPair : ", keyIID.NameId)

	return true, nil
}


// KeyPair 정보를 추출함
func MappingKeyPairInfo(KtCloudKeyPair ktsdk.KeyPair) irs.KeyPairInfo {

	cblogger.Infof("\n*** Mapping KeyPair Info!! ")

	//Since KT Cloud does not have a SystemID, the unique nameID value is also input to the SystemID
	keyPairInfo := irs.KeyPairInfo{
		IId:         irs.IID{
			NameId: KtCloudKeyPair.Name, 
			SystemId: KtCloudKeyPair.Name,
		},
		Fingerprint: KtCloudKeyPair.Fingerprint,
		PublicKey:  "N/A",
		PrivateKey: "N/A",
		VMUserID: vmUserName,

		//PrivateKey: KtCloudKeyPairList.Privatekey,
		//Caution!! : KT Cloud에서 KtCloud KeyPairList 조회시에는 response에 Private key값은 없음.
	}

	// keyValueList := []irs.KeyValue{
	// 	{Key: "CreateDate", Value: *KtCloudKeyPairList.CreateDate},
	// }

	// keyPairInfo.KeyValueList = keyValueList

	return keyPairInfo
}
