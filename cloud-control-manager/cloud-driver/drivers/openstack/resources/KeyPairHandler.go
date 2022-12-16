package resources

import (
	"errors"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/keypairs"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	KeyPair = "KEYPAIR"
)

type OpenStackKeyPairHandler struct {
	Client *gophercloud.ServiceClient
}

func setterKeypair(keypair keypairs.KeyPair) *irs.KeyPairInfo {
	keypairInfo := &irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   keypair.Name,
			SystemId: keypair.Name,
		},
		Fingerprint: keypair.Fingerprint,
		PublicKey:   keypair.PublicKey,
		PrivateKey:  keypair.PrivateKey,
	}
	return keypairInfo
}

func (keyPairHandler *OpenStackKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(keyPairHandler.Client.IdentityEndpoint, call.VMKEYPAIR, keyPairReqInfo.IId.NameId, "CreateKey()")
	start := call.Start()

	// 0. Check keyPairReqInfo
	err := CheckKeyPairReqInfo(keyPairReqInfo)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Key. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.KeyPairInfo{}, createErr
	}

	// 1. Check Exist
	exist, err := CheckExistKey(keyPairHandler.Client, keyPairReqInfo.IId)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Key. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.KeyPairInfo{}, createErr
	}
	if exist {
		keyName := keyPairReqInfo.IId.SystemId
		if keyPairReqInfo.IId.SystemId == "" {
			keyName = keyPairReqInfo.IId.NameId
		}
		createErr := errors.New(fmt.Sprintf("Failed to Create Key. err = The Key name %s already exists", keyName))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.KeyPairInfo{}, createErr
	}
	// 2. Set keyPairReqInfo
	create0pts := keypairs.CreateOpts{
		Name:      keyPairReqInfo.IId.NameId,
		PublicKey: "",
	}
	// 3. Create KeyPair
	keyPair, err := keypairs.Create(keyPairHandler.Client, create0pts).Extract()
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Key. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.KeyPairInfo{}, createErr
	}
	LoggingInfo(hiscallInfo, start)
	// 4. Set keyPairInfo
	keyPairInfo := setterKeypair(*keyPair)
	return *keyPairInfo, nil
}

func (keyPairHandler *OpenStackKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	hiscallInfo := GetCallLogScheme(keyPairHandler.Client.IdentityEndpoint, call.VMKEYPAIR, KeyPair, "ListKey()")
	start := call.Start()
	// 0. Get List Resource
	pager, err := keypairs.List(keyPairHandler.Client).AllPages()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List Key err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	LoggingInfo(hiscallInfo, start)

	keypair, err := keypairs.ExtractKeyPairs(pager)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List Key err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	// 1. Set List Resource
	keyPairList := make([]*irs.KeyPairInfo, len(keypair))
	for i, k := range keypair {
		keyPairList[i] = setterKeypair(k)
	}
	return keyPairList, nil
}

func (keyPairHandler *OpenStackKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(keyPairHandler.Client.IdentityEndpoint, call.VMKEYPAIR, keyIID.NameId, "GetKey()")
	// 0. Check keyPairInfo
	if iidCheck := CheckIIDValidation(keyIID); !iidCheck {
		getErr := errors.New(fmt.Sprintf("Failed to Get Key err = InValid IID"))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.KeyPairInfo{}, getErr
	}

	start := call.Start()

	// 1. Get Resource
	keyPair, err := GetRawKey(keyPairHandler.Client, keyIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Key. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.KeyPairInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)

	// 2. Set Resource
	keyPairInfo := setterKeypair(keyPair)
	return *keyPairInfo, nil
}

func (keyPairHandler *OpenStackKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(keyPairHandler.Client.IdentityEndpoint, call.VMKEYPAIR, keyIID.NameId, "DeleteKey()")
	// 0. Check keyPairInfo
	exist, err := CheckExistKey(keyPairHandler.Client, keyIID)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Key. err = %s", err))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	// 1. Check Exist
	if !exist {
		keyName := keyIID.SystemId
		if keyIID.SystemId == "" {
			keyName = keyIID.NameId
		}
		delErr := errors.New(fmt.Sprintf("Failed to Delete Key. err = The Key name %s not found", keyName))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	start := call.Start()
	// 2. Delete Resource
	err = keypairs.Delete(keyPairHandler.Client, keyIID.NameId).ExtractErr()
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Key. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func CheckExistKey(client *gophercloud.ServiceClient, keyIID irs.IID) (bool, error) {
	if ok := CheckIIDValidation(keyIID); !ok {
		return false, errors.New(fmt.Sprintf("invalid IID"))
	}
	keyName := keyIID.SystemId
	if keyIID.SystemId == "" {
		keyName = keyIID.NameId
	}
	pager, err := keypairs.List(client).AllPages()
	if err != nil {
		return false, err
	}
	keypairList, err := keypairs.ExtractKeyPairs(pager)
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

func CheckKeyPairReqInfo(keyPairReqInfo irs.KeyPairReqInfo) error {
	if keyPairReqInfo.IId.NameId == "" {
		return errors.New("invalid KeyPairReqInfo IID")
	}
	return nil
}

func GetRawKey(client *gophercloud.ServiceClient, keyIID irs.IID) (keypairs.KeyPair, error) {
	keyName := keyIID.SystemId
	if keyIID.SystemId == "" {
		keyName = keyIID.NameId
	}
	keyPair, err := keypairs.Get(client, keyName).Extract()
	if err != nil {
		return keypairs.KeyPair{}, err
	}
	return *keyPair, nil
}
