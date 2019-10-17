package resources

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"
	"golang.org/x/crypto/ssh"
	"io/ioutil"
	"log"
	"os"
)

type AzureKeyPairHandler struct {
	Region idrv.RegionInfo
}

func setterKeypair(keypairName string) *irs.KeyPairInfo {
	keypairInfo := &irs.KeyPairInfo{}
	return keypairInfo
}

func (keyPairHandler *AzureKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	// 생성된 KeyPair 정보 리턴
	return irs.KeyPairInfo{}, nil
}

func (keyPairHandler *AzureKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	return nil, nil
}

func (keyPairHandler *AzureKeyPairHandler) GetKey(keyPairID string) (irs.KeyPairInfo, error) {
	return irs.KeyPairInfo{}, nil
}

func (keyPairHandler *AzureKeyPairHandler) DeleteKey(keyPairID string) (bool, error) {
	return true, nil
}

func generateSSHKey(keyName string) (string, error) {

	// TODO: ENV 환경변수 PATH에 키 저장
	rootPath := os.Getenv("CBSPIDER_PATH")
	savePrivateFileTo := rootPath + "/conf/PrivateKey"
	savePublicFileTo := rootPath + "/conf/PublicKey"
	bitSize := 4096

	// 지정된 바이트크기의 RSA 형식 개인키(비공개키)를 만듬
	privateKey, err := generatePrivateKey(bitSize)
	if err != nil {
		//log.Fatal(err.Error())
	}

	// 개인키를 RSA에서 PEM 형식으로 인코딩
	privateKeyBytes := encodePrivateKeyToPEM(privateKey)

	// rsa.PublicKey를 가져와서 .pub 파일에 쓰기 적합한 바이트로 변환
	// "ssh-rsa ..."형식으로 변환
	publicKeyBytes, err := generatePublicKey(&privateKey.PublicKey)
	if err != nil {
		//log.Fatal(err.Error())
	}

	// 파일에 private Key를 쓴다
	err = writeKeyToFile(privateKeyBytes, savePrivateFileTo)
	if err != nil {
		//log.Fatal(err.Error())
	}

	// 파일에 public Key를 쓴다
	err = writeKeyToFile([]byte(publicKeyBytes), savePublicFileTo)
	if err != nil {
		//log.Fatal(err.Error())
	}

	// TODO: 파일 bytes로 읽어들여서 string으로 변환
	var pubKeyStr string

	data, err := ioutil.ReadFile(savePublicFileTo)
	if err != nil {

	}
	pubKeyStr = string(data)

	return pubKeyStr, nil
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
	fmt.Println(privateKey)
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
	fmt.Println(privatePEM)
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
	fmt.Println(pubKeyBytes)
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
