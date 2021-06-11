package resources

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/filter"
	"github.com/softlayer/softlayer-go/services"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"os"
	"strconv"
)

type IbmKeyPairHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	AccountClient  *services.Account
	SecuritySshKeyClient *services.Security_Ssh_Key
}

func(keyPairHandler *IbmKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error){
	hiscallInfo := GetCallLogScheme(keyPairHandler.Region, call.VMKEYPAIR, keyPairReqInfo.IId.NameId, "CreateKey()")

	if keyPairReqInfo.IId.NameId == ""{
		err := errors.New("invalid keyPairReqInfo")
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	exist, err := keyPairHandler.existCheckKeyPairName(keyPairReqInfo.IId.NameId)
	if exist{
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	// 폴더 체크
	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	//해쉬스트링
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}

	savePrivateFileTo := keyPairPath + hashString + "--" + keyPairReqInfo.IId.NameId
	savePublicFileTo := keyPairPath + hashString + "--" + keyPairReqInfo.IId.NameId + ".pub"
	bitSize := 4096

	// Check KeyPair Exists
	if _, err := os.Stat(savePrivateFileTo); err == nil {
		errMsg := fmt.Sprintf("KeyPair with name %s already exist", keyPairReqInfo.IId.NameId)
		createErr := errors.New(errMsg)
		// cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.KeyPairInfo{}, createErr
	}
	start := call.Start()

	privateKey, err := generatePrivateKey(bitSize)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}

	// 개인키를 RSA에서 PEM 형식으로 인코딩
	privateKeyBytes := encodePrivateKeyToPEM(privateKey)

	// rsa.PublicKey를 가져와서 .pub 파일에 쓰기 적합한 바이트로 변환
	// "ssh-rsa ..."형식으로 변환
	publicKeyBytes, err := generatePublicKey(&privateKey.PublicKey)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	// 파일에 private Key를 쓴다
	err = writeKeyToFile(privateKeyBytes, savePrivateFileTo)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	// 파일에 public Key를 쓴다
	err = writeKeyToFile([]byte(publicKeyBytes), savePublicFileTo)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	pubKey := fmt.Sprintf("%s", publicKeyBytes)
	newKey := datatypes.Security_Ssh_Key{
		Label: &keyPairReqInfo.IId.NameId,
		Key: &pubKey,
	}
	result, err := keyPairHandler.SecuritySshKeyClient.CreateObject(&newKey)
	if err!=nil{
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{},err
	}

	createKeypairInfo := irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   *result.Label,
			SystemId: strconv.Itoa(*result.Id),
		},
		Fingerprint: *result.Fingerprint,
		PublicKey:  *result.Key,
		PrivateKey: string(privateKeyBytes),
	}
	LoggingInfo(hiscallInfo,start)
	return createKeypairInfo,nil
}

func(keyPairHandler *IbmKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error){
	hiscallInfo := GetCallLogScheme(keyPairHandler.Region, call.VMKEYPAIR, "VMKEYPAIR", "ListKey()")
	start := call.Start()
	sshKeys, err := keyPairHandler.AccountClient.GetSshKeys()
	if err!=nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	var keyPairInfos []*irs.KeyPairInfo
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	for _, key := range sshKeys {
		privateKeyPath := keyPairPath + hashString + "--" + *key.Label
		privateKeyBytes, err := ioutil.ReadFile(privateKeyPath)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return nil, err
		}
		keypairInfo := irs.KeyPairInfo{
			IId: irs.IID{
				NameId:   *key.Label,
				SystemId: strconv.Itoa(*key.Id),
			},
			Fingerprint: *key.Fingerprint,
			PublicKey:  *key.Key,
			PrivateKey: string(privateKeyBytes),
		}
		keyPairInfos = append(keyPairInfos, &keypairInfo)
	}
	LoggingInfo(hiscallInfo, start)
	return keyPairInfos, nil
}

func(keyPairHandler *IbmKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error){
	hiscallInfo := GetCallLogScheme(keyPairHandler.Region, call.VMKEYPAIR, keyIID.NameId, "GetKey()")
	var sshKey datatypes.Security_Ssh_Key
	numSystemId, err := strconv.Atoi(keyIID.SystemId)
	start := call.Start()
	if err != nil {
		if keyIID.NameId != ""{
			sshKey, err = keyPairHandler.getterKeyPairByName(keyIID.NameId)
			if err !=nil {
				LoggingError(hiscallInfo, err)
				return irs.KeyPairInfo{}, err
			}
		}else{
			err = errors.New("invalid keyIID")
			LoggingError(hiscallInfo, err)
			return irs.KeyPairInfo{}, err
		}
	} else {
		sshKey, err = keyPairHandler.SecuritySshKeyClient.Id(numSystemId).GetObject()
		if err != nil {
			LoggingError(hiscallInfo, err)
			return irs.KeyPairInfo{}, err
		}
	}

	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}

	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	privateKeyPath := keyPairPath + hashString + "--" + *sshKey.Label
	privateKeyBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	keypairInfo := irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   *sshKey.Label,
			SystemId: strconv.Itoa(*sshKey.Id),
		},
		Fingerprint: *sshKey.Fingerprint,
		PublicKey:  *sshKey.Key,
		PrivateKey: string(privateKeyBytes),
	}
	LoggingInfo(hiscallInfo, start)
	return keypairInfo,nil
}

func(keyPairHandler *IbmKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error){
	hiscallInfo := GetCallLogScheme(keyPairHandler.Region, call.VMKEYPAIR, keyIID.NameId, "DeleteKey()")

	deleteKey, err := keyPairHandler.existCheckKeyPair(keyIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false,err
	}
	start := call.Start()
	result, err := keyPairHandler.SecuritySshKeyClient.Id(*deleteKey.Id).DeleteObject()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}
	if result {
		keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
		err = keyPairHandler.CheckKeyPairFolder(keyPairPath)
		// 폴더 없음 err != nil => local delete 필요 없음
		if err != nil {
			LoggingInfo(hiscallInfo,start)
			return true, nil
		}
		hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return false, err
		}

		privateKeyPath := keyPairPath + hashString + "--" + *deleteKey.Label
		publicKeyPath := keyPairPath + hashString + "--" + *deleteKey.Label + ".pub"

		err = os.Remove(privateKeyPath)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return false, err
		}
		err = os.Remove(publicKeyPath)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return false, err
		}
		LoggingInfo(hiscallInfo,start)
		return true, nil
	}
	err = errors.New(fmt.Sprintf("Failed Delete KeyPair"))
	LoggingError(hiscallInfo, err)
	return false ,err
}

func (keyPairHandler *IbmKeyPairHandler) getterKeyPairByName(KeyName string) (datatypes.Security_Ssh_Key, error){
	existFilter := filter.Path("sshKeys.label").Eq(KeyName).Build()
	sshKeys, err := keyPairHandler.AccountClient.Filter(existFilter).GetSshKeys()
	if err != nil{
		return datatypes.Security_Ssh_Key{}, err
	}
	if len(sshKeys) == 0 {
		return datatypes.Security_Ssh_Key{}, errors.New(fmt.Sprintf("sshKey with name %s not exist", KeyName))
	}else{
		return sshKeys[0], nil
	}
}

func (keyPairHandler *IbmKeyPairHandler) existCheckKeyPairName(KeyName string) (bool,error){
	existFilter := filter.Path("sshKeys.label").Eq(KeyName).Build()
	sshKeys, err := keyPairHandler.AccountClient.Filter(existFilter).GetSshKeys()
	if err != nil{
		return true, err
	}
	if len(sshKeys) == 0 {
		return false, nil
	}else{
		return true, errors.New(fmt.Sprintf("sshKey with name %s already exist", KeyName))
	}
}

func (keyPairHandler *IbmKeyPairHandler) existCheckKeyPair(keyIId irs.IID) (datatypes.Security_Ssh_Key,error){
	if keyIId.SystemId == ""{
		if keyIId.NameId != ""{
			existFilter := filter.Path("sshKeys.label").Eq(keyIId.NameId).Build()
			sshKeys, err := keyPairHandler.AccountClient.Filter(existFilter).GetSshKeys()
			if err != nil{
				return datatypes.Security_Ssh_Key{}, err
			}
			if len(sshKeys) == 0 {
				return datatypes.Security_Ssh_Key{}, errors.New(fmt.Sprintf("sshKey with name not exist"))
			}else{
				return sshKeys[0], nil
			}
		} else {
			return datatypes.Security_Ssh_Key{}, errors.New("invalid KeyIId")
		}
	}else{
		numSystemId, err := strconv.Atoi(keyIId.SystemId)
		if err != nil{
			return datatypes.Security_Ssh_Key{}, err
		}
		sshKey, err := keyPairHandler.SecuritySshKeyClient.Id(numSystemId).GetObject()
		if err != nil{
			return datatypes.Security_Ssh_Key{}, err
		}
		return sshKey, nil
	}
}

func (keyPairHandler *IbmKeyPairHandler) CheckKeyPairFolder(folderPath string) error {
	// Check KeyPair Folder Exists
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		if err := os.MkdirAll(folderPath, 0700); err != nil {
			return err
		}
	}
	return nil
}

// rsa.PublicKey를 가져와서 .pub 파일에 쓰기 적합한 바이트로 변환
// "ssh-rsa ..."형식으로 변환
func generatePublicKey(privatekey *rsa.PublicKey) ([]byte, error) {
	publicRsaKey, err := ssh.NewPublicKey(privatekey)
	if err != nil {
		return nil, err
	}

	pubKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)

	log.Println("Public key 생성")
	//fmt.Println(pubKeyBytes)
	return pubKeyBytes, nil
}

// 지정된 바이트크기의 RSA 형식 개인키(비공개키)를 만듬
func generatePrivateKey(bitSize int) (*rsa.PrivateKey, error) {
	// Private Key 생성
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		return nil, err
	}

	// Private Key 확인
	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	log.Println("Private Key generated(생성)")
	//fmt.Println(privateKey)
	return privateKey, nil
}
// 개인키를 RSA에서 PEM 형식으로 인코딩
func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	// Get ASN.1 DER format
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	// pem.Block
	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	// Private key in PEM format
	privatePEM := pem.EncodeToMemory(&privBlock)
	fmt.Println("privateKey Rsa -> Pem 형식으로 변환")
	//fmt.Println(privatePEM)
	return privatePEM
}

// 파일에 Key를 쓴다
func writeKeyToFile(keyBytes []byte, saveFileTo string) error {
	err := ioutil.WriteFile(saveFileTo, keyBytes, 0600)
	if err != nil {
		return err
	}

	log.Printf("Key 저장위치: %s", saveFileTo)
	return nil
}

