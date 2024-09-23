package resources

import (
	"errors"
	"fmt"
	"os"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	keypair "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	KeyPair         = "KEYPAIR"
	KeyPairProvider = "cloudit"
)

type ClouditKeyPairHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func (keyPairHandler *ClouditKeyPairHandler) CheckKeyPairFolder(folderPath string) error {
	// Check KeyPair Folder Exists
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		if err := os.MkdirAll(folderPath, 0700); err != nil {
			return err
		}
	}
	return nil
}

func (keyPairHandler *ClouditKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VMKEYPAIR, keyPairReqInfo.IId.NameId, "CreateKey()")
	start := call.Start()
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Key. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.KeyPairInfo{}, createErr
	}
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Key. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.KeyPairInfo{}, createErr
	}
	existKey, _ := keypair.GetKey(KeyPairProvider, hashString, keyPairReqInfo.IId.NameId)
	if existKey != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Key. err = The Key already exist"))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.KeyPairInfo{}, createErr
	}

	privateKeyBytes, _, err := keypair.GenKeyPair()

	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Key. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.KeyPairInfo{}, createErr
	}

	err = keypair.AddKey(KeyPairProvider, hashString, keyPairReqInfo.IId.NameId, string(privateKeyBytes))

	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Key. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.KeyPairInfo{}, createErr
	}

	keyPairInfo := irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   keyPairReqInfo.IId.NameId,
			SystemId: keyPairReqInfo.IId.NameId,
		},
		VMUserID: SSHDefaultUser,
		//PublicKey:  string(publicKeyBytes),
		PrivateKey: string(privateKeyBytes),
	}
	LoggingInfo(hiscallInfo, start)
	return keyPairInfo, nil
}

func (keyPairHandler *ClouditKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VMKEYPAIR, KeyPair, "ListKey()")
	start := call.Start()
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List Key. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List Key. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	keyValueList, err := keypair.ListKey(KeyPairProvider, hashString)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List Key. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	var keyPairInfoList []*irs.KeyPairInfo

	for _, keyValue := range keyValueList {
		//keypairInfo, err := keyPairHandler.GetKey(irs.IID{SystemId: keyValue.Key})
		keypairInfo := irs.KeyPairInfo{
			IId: irs.IID{
				NameId:   keyValue.Key,
				SystemId: keyValue.Key,
			},
			VMUserID: SSHDefaultUser,
			//PrivateKey: keyValue.Value,
		}
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List Key. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return nil, getErr
		}
		keyPairInfoList = append(keyPairInfoList, &keypairInfo)
	}
	LoggingInfo(hiscallInfo, start)
	return keyPairInfoList, nil
}

func (keyPairHandler *ClouditKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VMKEYPAIR, keyIID.NameId, "GetKey()")
	start := call.Start()

	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Key. err = InValid IID"))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.KeyPairInfo{}, getErr
	}
	_, err = keypair.GetKey(KeyPairProvider, hashString, keyIID.NameId)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Key. err = InValid IID"))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.KeyPairInfo{}, getErr
	}
	keypairInfo := irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   keyIID.NameId,
			SystemId: keyIID.NameId,
		},
		VMUserID: SSHDefaultUser,
		// PrivateKey: keyValue.Value,
	}
	LoggingInfo(hiscallInfo, start)
	return keypairInfo, nil
}

func (keyPairHandler *ClouditKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VMKEYPAIR, keyIID.NameId, "DeleteKey()")
	start := call.Start()
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		return false, err
	}
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Key err = InValid IID"))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}

	keyValue, err := keypair.GetKey(KeyPairProvider, hashString, keyIID.NameId)
	if keyValue == nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Key err = InValid IID"))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	delerr := keypair.DelKey(KeyPairProvider, hashString, keyIID.NameId)
	if delerr != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Key err = InValid IID"))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (keyPairHandler *ClouditKeyPairHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	return nil, errors.New("Does not support ListIID() yet!!")
}
