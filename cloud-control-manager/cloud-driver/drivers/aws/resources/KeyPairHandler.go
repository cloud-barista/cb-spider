package resources

import (
	"bytes"
	"crypto/md5"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	_ "github.com/davecgh/go-spew/spew"
	"golang.org/x/crypto/ssh"
)

type AwsKeyPairHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	Client         *ec2.EC2
}

/*
// @TODO: KeyPairInfo 리소스 프로퍼티 정의 필요
type KeyPairInfo struct {
	Name        string
	Fingerprint string
	KeyMaterial string //RSA PRIVATE KEY
}
*/

func (keyPairHandler *AwsKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	cblogger.Debug("Start ListKey()")
	var keyPairList []*irs.KeyPairInfo
	//spew.Dump(keyPairHandler)
	cblogger.Debug(keyPairHandler)

	input := &ec2.DescribeKeyPairsInput{
		KeyNames: []*string{
			nil,
		},
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   keyPairHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: "ListKey()",
		CloudOSAPI:   "DescribeKeyPairs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	//  Returns a list of key pairs
	result, err := keyPairHandler.Client.DescribeKeyPairs(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Debug(result)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Errorf("Unable to get key pairs, %v", err)
		return keyPairList, err
	}
	callogger.Info(call.String(callLogInfo))

	//cblogger.Debugf("Key Pairs:")
	for _, pair := range result.KeyPairs {
		/*
			cblogger.Debugf("%s: %s\n", *pair.KeyName, *pair.KeyFingerprint)
			keyPairInfo := new(irs.KeyPairInfo)
			keyPairInfo.Name = *pair.KeyName
			keyPairInfo.Fingerprint = *pair.KeyFingerprint
		*/
		keyPairInfo, errKeyPair := ExtractKeyPairDescribeInfo(pair)
		if errKeyPair != nil {
			cblogger.Infof("[%s] KeyPair는 Local에서 관리하는 대상이 아니기 때문에 Skip합니다.", *pair.KeyName)
			cblogger.Info(errKeyPair.Error())
			//return nil, errKeyPair
		} else {
			keyPairList = append(keyPairList, &keyPairInfo)
		}
	}

	cblogger.Debug(keyPairList)
	//spew.Dump(keyPairList)
	return keyPairList, nil
}

func (keyPairHandler *AwsKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	cblogger.Info(keyPairReqInfo)
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	cblogger.Infof("Getenv[CBSPIDER_ROOT] : [%s]", os.Getenv("CBSPIDER_ROOT"))
	cblogger.Infof("CBKeyPairPath : [%s]", CBKeyPairPath)
	cblogger.Infof("Final keyPairPath : [%s]", keyPairPath)

	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   keyPairHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: keyPairReqInfo.IId.NameId,
		CloudOSAPI:   "CreateKeyPair()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	// Creates a new  key pair with the given name
	result, err := keyPairHandler.Client.CreateKeyPair(&ec2.CreateKeyPairInput{
		//KeyName: aws.String(keyPairReqInfo.Name),
		KeyName: aws.String(keyPairReqInfo.IId.NameId),
	})
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "InvalidKeyPair.Duplicate" {
			cblogger.Errorf("Keypair %q already exists.", keyPairReqInfo.IId.NameId)
			return irs.KeyPairInfo{}, err
		}
		cblogger.Errorf("Unable to create key pair: %s, %v.", keyPairReqInfo.IId.NameId, err)
		return irs.KeyPairInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Infof("Created key pair %q %s\n%s\n", *result.KeyName, *result.KeyFingerprint, *result.KeyMaterial)

	cblogger.Info("공개키 생성")
	publicKey, errPub := makePublicKeyFromPrivateKey(*result.KeyMaterial)
	if errPub != nil {
		cblogger.Error(errPub)
		return irs.KeyPairInfo{}, err
	}

	cblogger.Infof("Public Key")
	//spew.Dump(publicKey)
	keyPairInfo := irs.KeyPairInfo{
		//Name:        *result.KeyName,
		IId:         irs.IID{keyPairReqInfo.IId.NameId, *result.KeyName},
		Fingerprint: *result.KeyFingerprint,
		PrivateKey:  *result.KeyMaterial, // AWS(PEM파일-RSA PRIVATE KEY)
		PublicKey:   publicKey,
		//KeyMaterial: *result.KeyMaterial,
		KeyValueList: []irs.KeyValue{
			{Key: "KeyMaterial", Value: *result.KeyMaterial},
		},
	}

	//spew.Dump(keyPairInfo)

	//	resultStr = strings.ReplaceAll(resultStr, "//", "/")
	//@TODO : File에 저장할 키 파일 이름의 PK 특징 때문에 제약이 걸릴 수 있음. (인증정보로 해쉬를 하면 부모 하위의 IAM 계정에서 해당 키를 못 찾을 수 있으며 사용자 정보를 사용하지 않으면 Uniqueue하지 않아서 충돌 날 수 있음.) 현재는 핑거프린트와 리전을 키로 사용함.
	/*
		hashString, err := CreateHashString(keyPairHandler.CredentialInfo, keyPairHandler.Region)
		if err != nil {
			cblogger.Error(err)
			return irs.KeyPairInfo{}, err
		}
		savePrivateFileTo := keyPairPath + hashString + "--" + keyPairReqInfo.IId.NameId + ".pem"
		savePublicFileTo := keyPairPath + hashString + "--" + keyPairReqInfo.IId.NameId + ".pub"
	*/
	hashString := strings.ReplaceAll(keyPairInfo.Fingerprint, ":", "") // 필요한 경우 리전 정보 추가하면 될 듯. 나중에 키 이름과 리전으로 암복호화를 진행하면 될 것같음.
	savePrivateFileTo := keyPairPath + hashString + ".pem"
	savePublicFileTo := keyPairPath + hashString + ".pub"
	//cblogger.Infof("hashString : [%s]", hashString)
	cblogger.Infof("savePrivateFileTo : [%s]", savePrivateFileTo)
	cblogger.Infof("savePublicFileTo : [%s]", savePublicFileTo)

	// 파일에 private Key를 쓴다
	err = writeKeyToFile([]byte(keyPairInfo.PrivateKey), savePrivateFileTo)
	if err != nil {
		return irs.KeyPairInfo{}, err
	}

	// 파일에 public Key를 쓴다
	err = writeKeyToFile([]byte(keyPairInfo.PublicKey), savePublicFileTo)
	if err != nil {
		return irs.KeyPairInfo{}, err
	}

	return keyPairInfo, nil
}

//혼선을 피하기 위해 keyPairID 대신 keyName으로 변경 함.
func (keyPairHandler *AwsKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {

	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	cblogger.Infof("Getenv[CBSPIDER_ROOT] : [%s]", os.Getenv("CBSPIDER_ROOT"))
	cblogger.Infof("CBKeyPairPath : [%s]", CBKeyPairPath)
	cblogger.Infof("Final keyPairPath : [%s]", keyPairPath)

	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	//keyPairID := keyName
	cblogger.Infof("keyName : [%s]", keyIID.SystemId)
	input := &ec2.DescribeKeyPairsInput{
		KeyNames: []*string{
			aws.String(keyIID.SystemId),
		},
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   keyPairHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: keyIID.SystemId,
		CloudOSAPI:   "DescribeKeyPairs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := keyPairHandler.Client.DescribeKeyPairs(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Info("result : ", result)
	cblogger.Info("err : ", err)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		if aerr, ok := err.(awserr.Error); ok {
			cblogger.Info("aerr : ", aerr)
			cblogger.Info("aerr.Code()  : ", aerr.Code())
			cblogger.Info("ok : ", ok)
			switch aerr.Code() {
			default:
				//fmt.Println(aerr.Error())
				cblogger.Error(aerr.Error())
				return irs.KeyPairInfo{}, aerr
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			cblogger.Error(err.Error())
			return irs.KeyPairInfo{}, err
		}
		return irs.KeyPairInfo{}, nil
	}
	callogger.Info(call.String(callLogInfo))

	if len(result.KeyPairs) > 0 {
		keyPairInfo, errKeyPair := ExtractKeyPairDescribeInfo(result.KeyPairs[0])
		if errKeyPair != nil {
			cblogger.Error(errKeyPair.Error())
			return irs.KeyPairInfo{}, errKeyPair
		}

		spew.Dump(keyPairInfo)
		return keyPairInfo, nil
	} else {
		return irs.KeyPairInfo{}, errors.New("정보를 찾을 수 없습니다.")
	}
}

//KeyPair 정보를 추출함
func ExtractKeyPairDescribeInfo(keyPair *ec2.KeyPairInfo) (irs.KeyPairInfo, error) {
	//spew.Dump(keyPair)
	keyPairInfo := irs.KeyPairInfo{
		IId: irs.IID{*keyPair.KeyName, *keyPair.KeyName},
		//Name:        *keyPair.KeyName,
		Fingerprint: *keyPair.KeyFingerprint,
	}

	// Local Keyfile 처리
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	hashString := strings.ReplaceAll(keyPairInfo.Fingerprint, ":", "") // 필요한 경우 리전 정보 추가하면 될 듯. 나중에 키 이름과 리전으로 암복호화를 진행하면 될 것같음.
	privateKeyPath := keyPairPath + hashString + ".pem"
	publicKeyPath := keyPairPath + hashString + ".pub"
	//cblogger.Infof("hashString : [%s]", hashString)
	cblogger.Debugf("[%s] ==> [%s]", keyPairInfo.IId.NameId, privateKeyPath)
	cblogger.Debugf("[%s] ==> [%s]", keyPairInfo.IId.NameId, publicKeyPath)

	// Private Key, Public Key 파일 정보 가져오기
	privateKeyBytes, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		cblogger.Errorf("[%s] KeyPair의 Local Private 파일 조회 실패", keyPairInfo.IId.NameId)
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}
	publicKeyBytes, err := ioutil.ReadFile(publicKeyPath)
	if err != nil {
		cblogger.Errorf("[%s] KeyPair의 Local Public 파일 조회 실패", keyPairInfo.IId.NameId)
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	keyPairInfo.PublicKey = string(publicKeyBytes)
	keyPairInfo.PrivateKey = string(privateKeyBytes)

	keyValueList := []irs.KeyValue{
		//{Key: "KeyMaterial", Value: *keyPair.KeyMaterial},
	}

	keyPairInfo.KeyValueList = keyValueList

	return keyPairInfo, nil
}

func (keyPairHandler *AwsKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error) {
	cblogger.Infof("삭제 요청된 키페어 : [%s]", keyIID.SystemId)

	keyPairInfo, errGet := keyPairHandler.GetKey(keyIID)
	if errGet != nil {
		return false, errGet
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   keyPairHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: keyIID.SystemId,
		CloudOSAPI:   "DeleteKeyPair()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	// Delete the key pair by name
	//by powerkim, result, err := keyPairHandler.Client.DeleteKeyPair(&ec2.DeleteKeyPairInput{
	_, err := keyPairHandler.Client.DeleteKeyPair(&ec2.DeleteKeyPairInput{
		KeyName: aws.String(keyIID.SystemId),
	})
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	//spew.Dump(result)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "InvalidKeyPair.Duplicate" {
			cblogger.Error("Key pair %q does not exist.", keyIID.SystemId)
			return false, err
		}
		cblogger.Errorf("Unable to delete key pair: %s, %v.", keyIID.SystemId, err)
		return false, err
	}
	callogger.Info(call.String(callLogInfo))
	cblogger.Infof("Successfully deleted %q AWS key pair\n", keyIID.SystemId)

	//====================
	// Local Keyfile 처리
	//====================
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	cblogger.Infof("Getenv[CBSPIDER_ROOT] : [%s]", os.Getenv("CBSPIDER_ROOT"))
	cblogger.Infof("CBKeyPairPath : [%s]", CBKeyPairPath)
	cblogger.Infof("Final keyPairPath : [%s]", keyPairPath)

	hashString := strings.ReplaceAll(keyPairInfo.Fingerprint, ":", "") // 필요한 경우 리전 정보 추가하면 될 듯. 나중에 키 이름과 리전으로 암복호화를 진행하면 될 것같음.
	privateKeyPath := keyPairPath + hashString + ".pem"
	publicKeyPath := keyPairPath + hashString + ".pub"

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

//=================================
// 공개 키 변환 및 키 정보 로컬 보관 로직 추가
//=================================
func (keyPairHandler *AwsKeyPairHandler) CheckKeyPairFolder(keyPairPath string) error {
	//키페어 생성 시 폴더가 존재하지 않으면 생성 함.
	_, errChkDir := os.Stat(keyPairPath)
	if os.IsNotExist(errChkDir) {
		cblogger.Errorf("[%s] Path가 존재하지 않아서 생성합니다.", keyPairPath)

		//errDir := os.MkdirAll(keyPairPath, 0755)
		errDir := os.MkdirAll(keyPairPath, 0700)
		//errDir := os.MkdirAll(keyPairPath, os.ModePerm) // os.ModePerm : 0777	//os.ModeDir
		if errDir != nil {
			//log.Fatal(err)
			cblogger.Errorf("[%s] Path가 생성 실패", keyPairPath)
			cblogger.Error(errDir)
			return errDir
		}
	}
	return nil
}

// KeyPair 해시 생성 함수
func CreateHashString(credentialInfo idrv.CredentialInfo, Region idrv.RegionInfo) (string, error) {
	//func CreateHashString(credentialInfo string) (string, error) {
	log.Println("credentialInfo.ClientId : " + credentialInfo.ClientId)
	log.Println("Region.Region : " + Region.Region)
	keyString := credentialInfo.ClientId + credentialInfo.ClientSecret + Region.Region
	//keyString := credentialInfo
	hasher := md5.New()
	_, err := io.WriteString(hasher, keyString)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
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

// ParseKey reads the given RSA private key and create a public one for it.
func makePublicKeyFromPrivateKey(pem string) (string, error) {
	key, err := ssh.ParseRawPrivateKey([]byte(pem))
	if err != nil {
		return "", err
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("%q is not a RSA key", pem)
	}
	pub, err := ssh.NewPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return "", err
	}

	return string(bytes.TrimRight(ssh.MarshalAuthorizedKey(pub), "\n")), nil
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
