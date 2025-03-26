// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, Innogrid, 2021.12.
// by ETRI, 2022.04.

package resources

import (
	"fmt"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/extensions/keypairs"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NhnCloudKeyPairHandler struct {
	RegionInfo idrv.RegionInfo
	VMClient   *nhnsdk.ServiceClient
}

func (keyPairHandler *NhnCloudKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	cblogger.Info("NHN Cloud Driver: called CreateKey()")
	callLogInfo := getCallLogScheme(keyPairHandler.RegionInfo.Region, call.VMKEYPAIR, keyPairReqInfo.IId.NameId, "CreateKey()")

	fmt.Println(keyPairReqInfo)

	if keyPairReqInfo.IId.NameId == "" {
		newErr := fmt.Errorf("Invalid KeyPair NameId.")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}

	exist, err := checkExistKey(keyPairHandler.VMClient, keyPairReqInfo.IId)
	if err != nil {
		newErr := fmt.Errorf("Failed to Create Key. err = %s", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}
	if exist {
		keyName := keyPairReqInfo.IId.SystemId
		if keyPairReqInfo.IId.SystemId == "" {
			keyName = keyPairReqInfo.IId.NameId
		}
		newErr := fmt.Errorf("Failed to Create Key. err = The Key name %s already exists", keyName)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}

	start := call.Start()
	create0pts := keypairs.CreateOpts{
		Name:      keyPairReqInfo.IId.NameId,
		PublicKey: "",
	}
	keyPair, err := keypairs.Create(keyPairHandler.VMClient, create0pts).Extract()
	if err != nil {
		newErr := fmt.Errorf("Failed to Create Key. err = %s", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	keyPairInfo := mappingKeypairInfo(*keyPair)
	return *keyPairInfo, nil
}

func (keyPairHandler *NhnCloudKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	cblogger.Info("NHN Cloud Driver: called ListKey()")
	callLogInfo := getCallLogScheme(keyPairHandler.RegionInfo.Region, call.VMKEYPAIR, "ListKey()", "ListKey()")

	start := call.Start()
	var listOptsBuilder keypairs.ListOptsBuilder
	allPages, err := keypairs.List(keyPairHandler.VMClient, listOptsBuilder).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KeyList err = %s", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	LoggingInfo(callLogInfo, start)

	keypair, err := keypairs.ExtractKeyPairs(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get KeyList err = %s", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	keyPairList := make([]*irs.KeyPairInfo, len(keypair))
	for i, k := range keypair {
		keyPairList[i] = mappingKeypairInfo(k)
	}
	return keyPairList, nil
}

func (keyPairHandler *NhnCloudKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {
	cblogger.Info("NHN Cloud Driver: called GetKey()")
	callLogInfo := getCallLogScheme(keyPairHandler.RegionInfo.Region, call.VMKEYPAIR, keyIID.NameId, "GetKey()")

	if iidCheck := checkIIDValidation(keyIID); !iidCheck {
		newErr := fmt.Errorf("Failed to Get Key. InValid IID")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}

	start := call.Start()
	keyPair, err := getRawKey(keyPairHandler.VMClient, keyIID)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get Key. %s", err.Error())
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.KeyPairInfo{}, newErr
	}
	LoggingInfo(callLogInfo, start)

	keyPairInfo := mappingKeypairInfo(keyPair)
	return *keyPairInfo, nil
}

func (keyPairHandler *NhnCloudKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called DeleteKey()")
	callLogInfo := getCallLogScheme(keyPairHandler.RegionInfo.Region, call.VMKEYPAIR, keyIID.NameId, "DeleteKey()")

	exist, err := checkExistKey(keyPairHandler.VMClient, keyIID)
	if err != nil {
		delErr := fmt.Errorf("Failed to Delete Key. %s", err)
		cblogger.Error(delErr.Error())
		LoggingError(callLogInfo, delErr)
		return false, delErr
	}

	if !exist {
		keyName := keyIID.SystemId

		if keyIID.SystemId == "" {
			keyName = keyIID.NameId
		}
		delErr := fmt.Errorf("Failed to Delete Key. The Key name %s not found", keyName)
		cblogger.Error(delErr.Error())
		LoggingError(callLogInfo, delErr)
		return false, delErr
	}

	start := call.Start()
	var delOptsBuilder keypairs.DeleteOptsBuilder
	err = keypairs.Delete(keyPairHandler.VMClient, keyIID.NameId, delOptsBuilder).ExtractErr()
	if err != nil {
		delErr := fmt.Errorf("Failed to Delete Key : %v", err.Error())
		cblogger.Error(delErr.Error())
		LoggingError(callLogInfo, delErr)
		return false, delErr
	}
	LoggingInfo(callLogInfo, start)

	return true, nil
}

func checkExistKey(client *nhnsdk.ServiceClient, keyIID irs.IID) (bool, error) {
	if ok := checkIIDValidation(keyIID); !ok {
		return false, fmt.Errorf("Invalid KeyPair IID!!")
	}

	keyName := keyIID.SystemId
	if keyIID.SystemId == "" {
		keyName = keyIID.NameId
	}

	var listOptsBuilder keypairs.ListOptsBuilder

	allPages, err := keypairs.List(client, listOptsBuilder).AllPages()
	if err != nil {
		return false, err
	}

	keypairList, err := keypairs.ExtractKeyPairs(allPages)
	if err != nil {
		return false, err
	}

	for _, keypair := range keypairList {
		if keypair.Name == keyName {
			return true, nil
		}
	}

	return false, nil
}

func getRawKey(client *nhnsdk.ServiceClient, keyIID irs.IID) (keypairs.KeyPair, error) {
	keyName := keyIID.SystemId

	if keyIID.SystemId == "" {
		keyName = keyIID.NameId
	}

	var getOptsBuilder keypairs.GetOptsBuilder
	keyPair, err := keypairs.Get(client, keyName, getOptsBuilder).Extract()
	if err != nil {
		return keypairs.KeyPair{}, err
	}

	return *keyPair, nil
}

func mappingKeypairInfo(keypair keypairs.KeyPair) *irs.KeyPairInfo {
	cblogger.Info("NHN Cloud Driver: called mappingKeypairInfo()")

	keypairInfo := &irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   keypair.Name,
			SystemId: keypair.Name,
		},
		Fingerprint: keypair.Fingerprint,
		PublicKey:   keypair.PublicKey,
		PrivateKey:  keypair.PrivateKey,
		VMUserID:    DefaultVMUserName,
	}

	keypairInfo.KeyValueList = irs.StructToKeyValueList(keypair)

	return keypairInfo
}

func (keyPairHandler *NhnCloudKeyPairHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	callLogInfo := getCallLogScheme(keyPairHandler.RegionInfo.Zone, call.VMKEYPAIR, "keyId", "ListIID()")

	start := call.Start()

	var iidList []*irs.IID

	listOpts := keypairs.ListOpts{}

	allPages, err := keypairs.List(keyPairHandler.VMClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get kypairs information from NhnCloud!! : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return make([]*irs.IID, 0), newErr
	}

	allKeypairs, err := keypairs.ExtractKeyPairs(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get kypairs List from NhnCloud!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return make([]*irs.IID, 0), newErr
	}

	for _, keypair := range allKeypairs {
		var iid irs.IID
		iid.NameId = keypair.Name
		iid.SystemId = keypair.Name

		iidList = append(iidList, &iid)
	}

	LoggingInfo(callLogInfo, start)

	return iidList, nil
}
