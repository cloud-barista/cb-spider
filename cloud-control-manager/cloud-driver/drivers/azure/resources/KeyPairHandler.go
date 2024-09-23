package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"

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
	Client         *armcompute.SSHPublicKeysClient
}

func (keyPairHandler *AzureKeyPairHandler) setterKey(key *armcompute.SSHPublicKeyResource, privateKey string) (*irs.KeyPairInfo, error) {
	if key.Name == nil || key.ID == nil || key.Properties.PublicKey == nil {
		return nil, errors.New(fmt.Sprintf("Invalid Key Resource"))
	}
	keypairInfo := irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   *key.Name,
			SystemId: *key.ID,
		},
		PublicKey:  *key.Properties.PublicKey,
		PrivateKey: privateKey,
	}

	if key.Tags != nil {
		keypairInfo.TagList = setTagList(key.Tags)
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
	// Create Tag
	tags := setTags(keyPairReqInfo.TagList)

	// 2. Create KeyPairData
	privateKey, publicKey, err := keypair.GenKeyPair()

	// 3. Set KeyPairData & keyPairReqInfo
	createOpt := armcompute.SSHPublicKeyResource{
		Location: &keyPairHandler.Region.Region,
		Properties: &armcompute.SSHPublicKeyResourceProperties{
			PublicKey: toStrPtr(string(publicKey)),
		},
		Tags: tags,
	}

	start := call.Start()
	// 4. Create KeyPair(Azure SSH Resource)
	keyResult, err := keyPairHandler.Client.Create(keyPairHandler.Ctx, keyPairHandler.Region.Region, keyPairReqInfo.IId.NameId, createOpt, nil)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Key. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.KeyPairInfo{}, createErr
	}
	// 5. Set keyPairInfo
	keyPairInfo, err := keyPairHandler.setterKey(&keyResult.SSHPublicKeyResource, string(privateKey))
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
	var keyList []*armcompute.SSHPublicKeyResource

	pager := keyPairHandler.Client.NewListByResourceGroupPager(keyPairHandler.Region.Region, nil)

	for pager.More() {
		page, err := pager.NextPage(keyPairHandler.Ctx)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List Key. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return nil, getErr
		}

		for _, key := range page.Value {
			keyList = append(keyList, key)
		}
	}

	// 0. Set List Resource
	var keyInfoList []*irs.KeyPairInfo
	for _, key := range keyList {
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
	keyPairInfo, err := keyPairHandler.setterKey(&key, "")
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
	_, err = keyPairHandler.Client.Delete(keyPairHandler.Ctx, keyPairHandler.Region.Region, keyIID.NameId, nil)

	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Key. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}
func CheckExistKey(keypairIId irs.IID, resourceGroup string, client *armcompute.SSHPublicKeysClient, ctx context.Context) (bool, error) {
	var keyList []*armcompute.SSHPublicKeyResource

	pager := client.NewListByResourceGroupPager(resourceGroup, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return false, err
		}

		for _, key := range page.Value {
			keyList = append(keyList, key)
		}
	}

	for _, key := range keyList {
		if keypairIId.SystemId != "" && keypairIId.SystemId == *key.ID {
			return true, nil
		}
		if keypairIId.NameId != "" && keypairIId.NameId == *key.Name {
			return true, nil
		}
	}
	return false, nil
}

func GetRawKey(keypairIId irs.IID, resourceGroup string, client *armcompute.SSHPublicKeysClient, ctx context.Context) (armcompute.SSHPublicKeyResource, error) {
	var publicKeyName = keypairIId.NameId

	if publicKeyName == "" {
		convertedNameId, err := GetSshKeyNameById(keypairIId.SystemId)
		if err != nil {
			return armcompute.SSHPublicKeyResource{}, err
		}
		publicKeyName = convertedNameId
	}

	resp, err := client.Get(ctx, resourceGroup, publicKeyName, nil)
	if err != nil {
		return armcompute.SSHPublicKeyResource{}, err
	}

	return resp.SSHPublicKeyResource, nil
}

func checkKeyPairReqInfo(keyPairReqInfo irs.KeyPairReqInfo) error {
	if keyPairReqInfo.IId.NameId == "" {
		return errors.New("invalid Key IID")
	}
	return nil
}

func (keyPairHandler *AzureKeyPairHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	return nil, errors.New("Does not support ListIID() yet!!")
}
