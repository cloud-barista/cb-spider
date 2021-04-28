package resources

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"golang.org/x/crypto/ssh"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
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
		if err := os.Mkdir(folderPath, 0700); err != nil {
			return err
		}
	}
	return nil
}

func (keyPairHandler *ClouditKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(keyPairHandler.Client.IdentityEndpoint, call.VMKEYPAIR, keyPairReqInfo.IId.NameId, "CreateKey()")

	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
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
		cblogger.Error(createErr.Error())
		hiscallInfo.ErrorMSG = createErr.Error()
		calllogger.Info(call.String(hiscallInfo))
		return irs.KeyPairInfo{}, createErr
	}

	start := call.Start()

	// 지정된 바이트크기의 RSA 형식 개인키(비공개키)를 만듬
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

	LoggingInfo(hiscallInfo, start)

	keyPairInfo := irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   keyPairReqInfo.IId.NameId,
			SystemId: keyPairReqInfo.IId.NameId,
		},
		PublicKey:  string(publicKeyBytes),
		PrivateKey: string(privateKeyBytes),
	}
	return keyPairInfo, nil
}

func (keyPairHandler *ClouditKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(keyPairHandler.Client.IdentityEndpoint, call.VMKEYPAIR, KeyPair, "ListKey()")

	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	var keyPairInfoList []*irs.KeyPairInfo

	start := call.Start()

	files, err := ioutil.ReadDir(keyPairPath)
	if err != nil {
		LoggingError(hiscallInfo, err)
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

	LoggingInfo(hiscallInfo, start)

	return keyPairInfoList, nil
}

func (keyPairHandler *ClouditKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(keyPairHandler.Client.IdentityEndpoint, call.VMKEYPAIR, keyIID.NameId, "GetKey()")

	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)

	privateKeyPath := keyPairPath + hashString + "--" + keyIID.NameId
	publicKeyPath := keyPairPath + hashString + "--" + keyIID.NameId + ".pub"

	start := call.Start()

	// Private Key, Public Key 파일 정보 가져오기
	privateKeyBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}
	publicKeyBytes, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.KeyPairInfo{}, err
	}

	LoggingInfo(hiscallInfo, start)

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
	// log HisCall
	hiscallInfo := GetCallLogScheme(keyPairHandler.Client.IdentityEndpoint, call.VMKEYPAIR, keyIID.NameId, "DeleteKey()")

	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		return false, err
	}
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	privateKeyPath := keyPairPath + hashString + "--" + keyIID.NameId
	publicKeyPath := keyPairPath + hashString + "--" + keyIID.NameId + ".pub"

	start := call.Start()

	// Private Key, Public Key 삭제
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

	LoggingInfo(hiscallInfo, start)

	return true, nil
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

// 파일에 Key를 쓴다
func writeKeyToFile(keyBytes []byte, saveFileTo string) error {
	err := ioutil.WriteFile(saveFileTo, keyBytes, 0600)
	if err != nil {
		return err
	}

	log.Printf("Key 저장위치: %s", saveFileTo)
	return nil
}
