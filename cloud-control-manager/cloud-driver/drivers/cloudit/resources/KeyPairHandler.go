package resources

import (
	"errors"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	keypair "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"os"
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
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	existKey, _ := keypair.GetKey(KeyPairProvider, hashString, keyPairReqInfo.IId.NameId)
	if existKey != nil {
		cblogger.Error(errors.New("exist Key"))
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}

	privateKeyBytes, _, err := keypair.GenKeyPair()

	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}

	err = keypair.AddKey(KeyPairProvider, hashString, keyPairReqInfo.IId.NameId, string(privateKeyBytes))

	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
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
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	keyValueList, err := keypair.ListKey(KeyPairProvider, hashString)

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
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return nil, err
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
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	_, err = keypair.GetKey(KeyPairProvider, hashString, keyIID.NameId)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
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
		cblogger.Error(err)
		return false, err
	}

	keyValue, err := keypair.GetKey(KeyPairProvider, hashString, keyIID.NameId)
	if keyValue == nil {
		cblogger.Error("KeyPair Not Found")
		return false, err
	}
	delerr := keypair.DelKey(KeyPairProvider, hashString, keyIID.NameId)
	if delerr != nil {
		cblogger.Error(err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}
