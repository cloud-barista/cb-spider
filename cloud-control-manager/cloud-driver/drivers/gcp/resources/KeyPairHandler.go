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

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"golang.org/x/crypto/ssh"
)

type GCPKeyPairHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
}

func (keyPairHandler *GCPKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	keyPairName := strings.ToLower(keyPairReqInfo.IId.NameId)
	cblogger.Infof("keyPairName [%s] --> [%s]", keyPairReqInfo.IId.NameId, keyPairName)

	//projectId := keyPairHandler.CredentialInfo.ProjectID
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	cblogger.Infof("Getenv[CBSPIDER_ROOT] : [%s]", os.Getenv("CBSPIDER_ROOT"))
	cblogger.Infof("CBKeyPairPath : [%s]", CBKeyPairPath)
	cblogger.Infof("Final keyPairPath : [%s]", keyPairPath)

	//키페어 생성 시 폴더가 존재하지 않으면 생성 함.
	_, errChkDir := os.Stat(keyPairPath)
	if os.IsNotExist(errChkDir) {
		cblogger.Errorf("[%s] Path가 존재하지 않아서 생성합니다.", keyPairPath)

		errDir := os.MkdirAll(keyPairPath, 0755)
		//errDir := os.MkdirAll(keyPairPath, os.ModePerm) // os.ModePerm : 0777	//os.ModeDir
		if errDir != nil {
			//log.Fatal(err)
			cblogger.Errorf("[%s] Path가 생성 실패", keyPairPath)
			cblogger.Error(errDir)
			return irs.KeyPairInfo{}, errDir
		}
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   keyPairHandler.Region.Zone,
		ResourceType: call.VMKEYPAIR,
		ResourceName: keyPairReqInfo.IId.NameId,
		CloudOSAPI:   "CreateHashString()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	savePrivateFileTo := keyPairPath + hashString + "--" + keyPairName
	savePublicFileTo := keyPairPath + hashString + "--" + keyPairName + ".pub"
	bitSize := 4096

	// Check KeyPair Exists
	if _, err := os.Stat(savePrivateFileTo); err == nil {
		errMsg := fmt.Sprintf("KeyPair with name %s already exist", keyPairName)
		createErr := errors.New(errMsg)
		cblogger.Error(err)
		return irs.KeyPairInfo{}, createErr
	}

	// 지정된 바이트크기의 RSA 형식 개인키(비공개키)를 만듬
	privateKey, err := generatePrivateKey(bitSize)
	if err != nil {
		cblogger.Error(err)
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
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	// 파일에 private Key를 쓴다
	err = writeKeyToFile(privateKeyBytes, savePrivateFileTo)
	if err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	// 파일에 public Key를 쓴다
	err = writeKeyToFile([]byte(publicKeyString), savePublicFileTo)
	if err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	keyPairInfo := irs.KeyPairInfo{
		IId: irs.IID{
			NameId:   keyPairName,
			SystemId: keyPairName,
		},
		PublicKey:  publicKeyString,
		PrivateKey: string(privateKeyBytes),
	}
	return keyPairInfo, nil
}

func (keyPairHandler *GCPKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	cblogger.Infof("Getenv[CBSPIDER_ROOT] : [%s]", os.Getenv("CBSPIDER_ROOT"))
	cblogger.Infof("CBKeyPairPath : [%s]", CBKeyPairPath)
	cblogger.Infof("Final keyPairPath : [%s]", keyPairPath)

	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		cblogger.Error("Fail CreateHashString")
		cblogger.Error(err)
		return nil, err
	}

	var keyPairInfoList []*irs.KeyPairInfo

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   keyPairHandler.Region.Zone,
		ResourceType: call.VMKEYPAIR,
		ResourceName: "List",
		CloudOSAPI:   "ioutil.ReadDir()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	files, err := ioutil.ReadDir(keyPairPath)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		//cblogger.Error("Fail ReadDir(keyPairPath)")
		//cblogger.Error(err)
		//return nil, err

		//키페어 폴더가 없는 경우 생성된 키가 없는 것으로 변경
		return nil, nil
	}
	callogger.Info(call.String(callLogInfo))

	for _, f := range files {
		if strings.Contains(f.Name(), ".pub") {
			continue
		}
		if strings.Contains(f.Name(), hashString) {
			fileNameArr := strings.Split(f.Name(), "--")
			keypairInfo, err := keyPairHandler.GetKey(irs.IID{SystemId: fileNameArr[1]})
			if err != nil {
				cblogger.Error("Fail GetKey")
				cblogger.Error(err)
				return nil, err
			}
			keyPairInfoList = append(keyPairInfoList, &keypairInfo)
		}
	}

	return keyPairInfoList, nil
}

func (keyPairHandler *GCPKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {
	cblogger.Infof("keyPairName : [%s]", keyIID.SystemId)
	keyPairName := strings.ToLower(keyIID.SystemId)
	cblogger.Infof("keyPairName 소문자로 치환 : [%s]", keyPairName)

	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	cblogger.Infof("Getenv[CBSPIDER_ROOT] : [%s]", os.Getenv("CBSPIDER_ROOT"))
	cblogger.Infof("CBKeyPairPath : [%s]", CBKeyPairPath)
	cblogger.Infof("Final keyPairPath : [%s]", keyPairPath)

	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		cblogger.Error("Fail CreateHashString")
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	privateKeyPath := keyPairPath + hashString + "--" + keyPairName
	publicKeyPath := keyPairPath + hashString + "--" + keyPairName + ".pub"

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   keyPairHandler.Region.Zone,
		ResourceType: call.VMKEYPAIR,
		ResourceName: keyIID.SystemId,
		CloudOSAPI:   "os.Stat()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	//키 페어 존재 여부 체크
	if _, err := os.Stat(privateKeyPath); err != nil {
		callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.KeyPairInfo{}, errors.New("Not Found : [" + keyIID.SystemId + "] KeyPair Not Found.")
	}
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	callogger.Info(call.String(callLogInfo))

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
			NameId:   keyPairName,
			SystemId: keyPairName,
		},
		PublicKey:  string(publicKeyBytes),
		PrivateKey: string(privateKeyBytes),
	}
	return keypairInfo, nil
}

func (keyPairHandler *GCPKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error) {
	cblogger.Infof("keyPairName : [%s]", keyIID.SystemId)
	keyPairName := strings.ToLower(keyIID.SystemId)
	cblogger.Infof("keyPairName 소문자로 치환 : [%s]", keyPairName)

	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	cblogger.Infof("Getenv[CBSPIDER_ROOT] : [%s]", os.Getenv("CBSPIDER_ROOT"))
	cblogger.Infof("CBKeyPairPath : [%s]", CBKeyPairPath)
	cblogger.Infof("Final keyPairPath : [%s]", keyPairPath)

	hashString, err := CreateHashString(keyPairHandler.CredentialInfo)
	if err != nil {
		cblogger.Error("Fail CreateHashString")
		cblogger.Error(err)
		return false, err
	}

	privateKeyPath := keyPairPath + hashString + "--" + keyPairName
	publicKeyPath := keyPairPath + hashString + "--" + keyPairName + ".pub"

	//키 페어 존재 여부 체크
	if _, err := os.Stat(privateKeyPath); err != nil {
		cblogger.Error(err)
		return false, errors.New("Not Found : [" + keyIID.SystemId + "] KeyPair Not Found.")
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   keyPairHandler.Region.Zone,
		ResourceType: call.VMKEYPAIR,
		ResourceName: keyPairName,
		CloudOSAPI:   "Remove()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	// Private Key, Public Key 삭제
	err = os.Remove(privateKeyPath)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return false, err
	}
	callogger.Info(call.String(callLogInfo))
	err = os.Remove(publicKeyPath)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	return true, nil
}

// 지정된 바이트크기의 RSA 형식 개인키(비공개키)를 만듬
func generatePrivateKey(bitSize int) (*rsa.PrivateKey, error) {
	// Private Key 생성
	privateKey, err := rsa.GenerateKey(rand.Reader, bitSize)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	// Private Key 확인
	err = privateKey.Validate()
	if err != nil {
		cblogger.Error(err)
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
		cblogger.Error(err)
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
		cblogger.Error(err)
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
