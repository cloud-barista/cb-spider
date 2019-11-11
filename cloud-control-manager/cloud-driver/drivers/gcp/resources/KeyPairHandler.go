// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// program by ysjeon@mz.co.kr, 2019.07.
// modify by devunet@mz.co.kr, 2019.11.

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

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"golang.org/x/crypto/ssh"
)

type GCPKeyPairHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
}

func (keyPairHandler *GCPKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	keyPairName := strings.ToLower(keyPairReqInfo.Name)
	cblogger.Infof("keyPairName [%s] --> [%s]", keyPairReqInfo.Name, keyPairName)

	//projectId := keyPairHandler.CredentialInfo.ProjectID
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		return irs.KeyPairInfo{}, err
	}

	savePrivateFileTo := keyPairPath + hashString + "--" + keyPairName
	savePublicFileTo := keyPairPath + hashString + "--" + keyPairName + ".pub"
	bitSize := 4096

	// Check KeyPair Exists
	if _, err := os.Stat(savePrivateFileTo); err == nil {
		errMsg := fmt.Sprintf("KeyPair with name %s already exist", keyPairName)
		createErr := errors.New(errMsg)
		return irs.KeyPairInfo{}, createErr
	}

	// 지정된 바이트크기의 RSA 형식 개인키(비공개키)를 만듬
	privateKey, err := generatePrivateKey(bitSize)
	if err != nil {
		return irs.KeyPairInfo{}, err
	}

	// 개인키를 RSA에서 PEM 형식으로 인코딩
	privateKeyBytes := encodePrivateKeyToPEM(privateKey)

	// rsa.PublicKey를 가져와서 .pub 파일에 쓰기 적합한 바이트로 변환
	// "ssh-rsa ..."형식으로 변환
	publicKeyBytes, err := generatePublicKey(&privateKey.PublicKey)
	publicKeyString := string(publicKeyBytes)
	// projectId 대신에 cb-user 고정
	publicKeyString = strings.TrimSpace(publicKeyString) + " " + "cb-user"
	fmt.Println("publicKeyString : ", publicKeyString)
	if err != nil {
		return irs.KeyPairInfo{}, err
	}

	// 파일에 private Key를 쓴다
	err = writeKeyToFile(privateKeyBytes, savePrivateFileTo)
	if err != nil {
		return irs.KeyPairInfo{}, err
	}

	// 파일에 public Key를 쓴다
	err = writeKeyToFile([]byte(publicKeyString), savePublicFileTo)
	if err != nil {
		return irs.KeyPairInfo{}, err
	}

	keyPairInfo := irs.KeyPairInfo{
		Name:       keyPairName,
		PublicKey:  publicKeyString,
		PrivateKey: string(privateKeyBytes),
	}
	return keyPairInfo, nil
}

func (keyPairHandler *GCPKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		return nil, err
	}

	var keyPairInfoList []*irs.KeyPairInfo

	files, err := ioutil.ReadDir(keyPairPath)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if strings.Contains(f.Name(), ".pub") {
			continue
		}
		if strings.Contains(f.Name(), hashString) {
			fileNameArr := strings.Split(f.Name(), "--")
			keypairInfo, err := keyPairHandler.GetKey(fileNameArr[1])
			if err != nil {
				return nil, err
			}
			keyPairInfoList = append(keyPairInfoList, &keypairInfo)
		}
	}

	return keyPairInfoList, nil
}

func (keyPairHandler *GCPKeyPairHandler) GetKey(keyPairName string) (irs.KeyPairInfo, error) {
	cblogger.Infof("keyPairName : [%s]", keyPairName)
	keyPairName = strings.ToLower(keyPairName)
	cblogger.Infof("keyPairName 소문자로 치환 : [%s]", keyPairName)

	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)

	privateKeyPath := keyPairPath + hashString + "--" + keyPairName
	publicKeyPath := keyPairPath + hashString + "--" + keyPairName + ".pub"

	// Private Key, Public Key 파일 정보 가져오기
	privateKeyBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return irs.KeyPairInfo{}, err
	}
	publicKeyBytes, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		return irs.KeyPairInfo{}, err
	}

	keypairInfo := irs.KeyPairInfo{
		Name:       keyPairName,
		PublicKey:  string(publicKeyBytes),
		PrivateKey: string(privateKeyBytes),
	}
	return keypairInfo, nil
}

func (keyPairHandler *GCPKeyPairHandler) DeleteKey(keyPairName string) (bool, error) {
	cblogger.Infof("keyPairName : [%s]", keyPairName)
	keyPairName = strings.ToLower(keyPairName)
	cblogger.Infof("keyPairName 소문자로 치환 : [%s]", keyPairName)

	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		return false, err
	}

	privateKeyPath := keyPairPath + hashString + "--" + keyPairName
	publicKeyPath := keyPairPath + hashString + "--" + keyPairName + ".pub"

	// Private Key, Public Key 삭제
	err = os.Remove(privateKeyPath)
	if err != nil {
		return false, err
	}
	err = os.Remove(publicKeyPath)
	if err != nil {
		return false, err
	}

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

// Credential 기반 hash 생성
/*func createHashString(credentialInfo idrv.CredentialInfo) (string, error) {
	keyString := credentialInfo.ClientId + credentialInfo.ClientSecret + credentialInfo.TenantId + credentialInfo.SubscriptionId
	hasher := md5.New()
	_, err := io.WriteString(hasher, keyString)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}*/
