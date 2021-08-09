package resources

import (
	"errors"
	"fmt"
	keypair "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	"io/ioutil"
	"os"
	"strings"

	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	KeyPair = "KEYPAIR"
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

	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	savePrivateFileTo := keyPairPath + hashString + "--" + keyPairReqInfo.IId.NameId
	savePublicFileTo := keyPairPath + hashString + "--" + keyPairReqInfo.IId.NameId + ".pub"

	// Check KeyPair Exists
	if _, err := os.Stat(savePrivateFileTo); err == nil {
		errMsg := fmt.Sprintf("KeyPair with name %s already exist", keyPairReqInfo.IId.NameId)
		createErr := errors.New(errMsg)
		cblogger.Error(createErr.Error())
		return irs.KeyPairInfo{}, createErr
	}

	// 지정된 바이트크기의 RSA 형식 개인키(비공개키)를 만듬
	privateKey, publicKey, err := keypair.GenKeyPair()
	if err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	// 파일에 private Key를 쓴다
	err = keypair.SaveKey(privateKey, savePrivateFileTo)
	if err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	// 파일에 public Key를 쓴다
	err = keypair.SaveKey([]byte(publicKey), savePublicFileTo)
	if err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	keyPairInfo := irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   keyPairReqInfo.IId.NameId,
			SystemId: keyPairReqInfo.IId.NameId,
		},
		PublicKey:  string(publicKey),
		PrivateKey: string(privateKey),
	}
	return keyPairInfo, nil
}

func (keyPairHandler *ClouditKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {

	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		cblogger.Error(err)
		return nil, err
	}
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	var keyPairInfoList []*irs.KeyPairInfo

	files, err := ioutil.ReadDir(keyPairPath)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	for _, f := range files {
		if strings.Contains(f.Name(), ".pub") {
			continue
		}
		if strings.Contains(f.Name(), hashString) {
			fileNameArr := strings.Split(f.Name(), "--")
			keypairInfo, err := keyPairHandler.GetKey(irs.IID{NameId: fileNameArr[1]})
			if err != nil {
				return nil, err
			}
			keyPairInfoList = append(keyPairInfoList, &keypairInfo)
		}
	}

	return keyPairInfoList, nil
}

func (keyPairHandler *ClouditKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {

	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)

	privateKeyPath := keyPairPath + hashString + "--" + keyIID.NameId
	publicKeyPath := keyPairPath + hashString + "--" + keyIID.NameId + ".pub"

	// Private Key, Public Key 파일 정보 가져오기
	privateKeyBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}
	publicKeyBytes, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	keypairInfo := irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   keyIID.NameId,
			SystemId: keyIID.NameId,
		},
		PublicKey:  string(publicKeyBytes),
		PrivateKey: string(privateKeyBytes),
	}
	return keypairInfo, nil
}

func (keyPairHandler *ClouditKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error) {

	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		return false, err
	}
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	privateKeyPath := keyPairPath + hashString + "--" + keyIID.NameId
	publicKeyPath := keyPairPath + hashString + "--" + keyIID.NameId + ".pub"

	// Private Key, Public Key 삭제
	err = os.Remove(privateKeyPath)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}
	err = os.Remove(publicKeyPath)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	return true, nil
}
