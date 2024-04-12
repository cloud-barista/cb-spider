package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/go-autorest/autorest/to"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-03-01/compute"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	keypair "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	KeyPair = "KEYPAIR"
)

type AzureKeyPairHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	Ctx            context.Context
	Client         *compute.SSHPublicKeysClient
}

func (keyPairHandler *AzureKeyPairHandler) setterKey(key compute.SSHPublicKeyResource, privateKey string) (*irs.KeyPairInfo, error) {
	if key.Name == nil || key.ID == nil || key.PublicKey == nil {
		return nil, errors.New(fmt.Sprintf("Invalid Key Resource"))
	}
	keypairInfo := irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   *key.Name,
			SystemId: *key.ID,
		},
		PublicKey:  *key.PublicKey,
		PrivateKey: privateKey,
	}
	return &keypairInfo, nil
}

func (keyPairHandler *AzureKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	hiscallInfo := GetCallLogScheme(keyPairHandler.Region, call.VMKEYPAIR, keyPairReqInfo.IId.NameId, "CreateKey()")
	// 0. Check keyPairReqInfo
	err := checkKeyPairReqInfo(keyPairReqInfo)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Key. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.KeyPairInfo{}, createErr
	}

	// 1. Check Exist
	exist, err := CheckExistKey(keyPairReqInfo.IId, keyPairHandler.Region.Region, keyPairHandler.Client, keyPairHandler.Ctx)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Key. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.KeyPairInfo{}, createErr
	}

	if exist {
		createErr := errors.New(fmt.Sprintf("Failed to Create Key. err = The Key already exist"))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.KeyPairInfo{}, createErr
	}
	// 2. Create KeyPairData
	privateKey, publicKey, err := keypair.GenKeyPair()

	// 3. Set KeyPairData & keyPairReqInfo
	createOpt := compute.SSHPublicKeyResource{
		Location: to.StringPtr(keyPairHandler.Region.Region),
		SSHPublicKeyResourceProperties: &compute.SSHPublicKeyResourceProperties{
			PublicKey: to.StringPtr(string(publicKey)),
		},
	}

	start := call.Start()
	// 4. Create KeyPair(Azure SSH Resource)
	keyResult, err := keyPairHandler.Client.Create(keyPairHandler.Ctx, keyPairHandler.Region.Region, keyPairReqInfo.IId.NameId, createOpt)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Key. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.KeyPairInfo{}, createErr
	}
	// 5. Set keyPairInfo
	keyPairInfo, err := keyPairHandler.setterKey(keyResult, string(privateKey))
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Key. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.KeyPairInfo{}, createErr
	}
	LoggingInfo(hiscallInfo, start)
	return *keyPairInfo, nil
}

func (keyPairHandler *AzureKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	hiscallInfo := GetCallLogScheme(keyPairHandler.Region, call.VMKEYPAIR, KeyPair, "ListKey()")
	start := call.Start()

	// 0. Get List Resource
	listResult, err := keyPairHandler.Client.ListByResourceGroup(keyPairHandler.Ctx, keyPairHandler.Region.Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List Key. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	// 0. Set List Resource
	var keyInfoList []*irs.KeyPairInfo
	for _, key := range listResult.Values() {
		keyInfo, err := keyPairHandler.setterKey(key, "")
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List Key. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return nil, getErr
		}
		keyInfoList = append(keyInfoList, keyInfo)
	}

	LoggingInfo(hiscallInfo, start)

	return keyInfoList, nil
}

func (keyPairHandler *AzureKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {
	hiscallInfo := GetCallLogScheme(keyPairHandler.Region, call.VMKEYPAIR, keyIID.NameId, "GetKey()")
	start := call.Start()
	// 0. Check keyPairInfo
	if iidCheck := CheckIIDValidation(keyIID); !iidCheck {
		getErr := errors.New(fmt.Sprintf("Failed to Get Key. err = InValid IID"))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.KeyPairInfo{}, getErr
	}
	// 1. Get Resource
	key, err := GetRawKey(keyIID, keyPairHandler.Region.Region, keyPairHandler.Client, keyPairHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Key. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.KeyPairInfo{}, getErr
	}
	// 2. Set Resource
	keyPairInfo, err := keyPairHandler.setterKey(key, "")
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Key. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.KeyPairInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return *keyPairInfo, nil
}

func (keyPairHandler *AzureKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(keyPairHandler.Region, call.VMKEYPAIR, keyIID.NameId, "DeleteKey()")
	// 0. Check keyPairInfo
	if iidCheck := CheckIIDValidation(keyIID); !iidCheck {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Key err = InValid IID"))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	// 1. Check Exist
	exist, err := CheckExistKey(keyIID, keyPairHandler.Region.Region, keyPairHandler.Client, keyPairHandler.Ctx)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Key. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}

	if !exist {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Key. err = The Key not exist"))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}

	start := call.Start()
	// 2. Delete Resource
	_, err = keyPairHandler.Client.Delete(keyPairHandler.Ctx, keyPairHandler.Region.Region, keyIID.NameId)

	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Key. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}
func CheckExistKey(keypairIId irs.IID, resourceGroup string, client *compute.SSHPublicKeysClient, ctx context.Context) (bool, error) {
	keyList, err := client.ListByResourceGroup(ctx, resourceGroup)
	if err != nil {
		return false, err
	}
	for _, keyValue := range keyList.Values() {
		if keypairIId.SystemId != "" && keypairIId.SystemId == *keyValue.ID {
			return true, nil
		}
		if keypairIId.NameId != "" && keypairIId.NameId == *keyValue.Name {
			return true, nil
		}
	}
	return false, nil
}

func GetRawKey(keypairIId irs.IID, resourceGroup string, client *compute.SSHPublicKeysClient, ctx context.Context) (compute.SSHPublicKeyResource, error) {
	if keypairIId.NameId == "" {
		convertedNameId, err := GetSshKeyNameById(keypairIId.SystemId)
		if err != nil {
			return compute.SSHPublicKeyResource{}, err
		}
		return client.Get(ctx, resourceGroup, convertedNameId)
	} else {
		return client.Get(ctx, resourceGroup, keypairIId.NameId)
	}
}

func checkKeyPairReqInfo(keyPairReqInfo irs.KeyPairReqInfo) error {
	if keyPairReqInfo.IId.NameId == "" {
		return errors.New("invalid Key IID")
	}
	return nil
}
