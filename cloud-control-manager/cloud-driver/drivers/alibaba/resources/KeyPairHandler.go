// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by devunet@mz.co.kr, 2019.09.

package resources

import (
	"errors"
	"io/ioutil"
	"os"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	keypair "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type AlibabaKeyPairHandler struct {
	Region idrv.RegionInfo
	Client *ecs.Client
}

/*
// @TODO: KeyPairInfo 리소스 프로퍼티 정의 필요
type KeyPairInfo struct {
	Name        string
	Fingerprint string
	KeyMaterial string //RSA PRIVATE KEY
}
*/

func (keyPairHandler *AlibabaKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	cblogger.Debug("Start ListKey()")
	var keyPairList []*irs.KeyPairInfo
	//spew.Dump(keyPairHandler)
	cblogger.Info(keyPairHandler)

	request := ecs.CreateDescribeKeyPairsRequest()
	request.Scheme = "https"

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   keyPairHandler.Region.Zone,
		ResourceType: call.VMKEYPAIR,
		ResourceName: "ListKey()",
		CloudOSAPI:   "DescribeKeyPairs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	//  Returns a list of key pairs
	result, err := keyPairHandler.Client.DescribeKeyPairs(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Errorf("Unable to get key pairs, %v", err)
		return keyPairList, err
	}
	callogger.Info(call.String(callLogInfo))
	cblogger.Info(result)

	//cblogger.Debugf("Key Pairs:")
	for _, pair := range result.KeyPairs.KeyPair {
		keyPairInfo, errKeyPair := ExtractKeyPairDescribeInfo(&pair)

		if errKeyPair != nil {
			cblogger.Infof("[%s] KeyPair는 Local에서 관리하는 대상이 아니기 때문에 Skip합니다.", *&pair.KeyPairName)
			cblogger.Info(errKeyPair.Error())
		} else {
			keyPairList = append(keyPairList, &keyPairInfo)
		}
	}

	cblogger.Info(keyPairList)
	//spew.Dump(keyPairList)
	return keyPairList, nil
}

func (keyPairHandler *AlibabaKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	cblogger.Info("Start CreateKey() : ", keyPairReqInfo)

	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	cblogger.Infof("Getenv[CBSPIDER_ROOT] : [%s]", os.Getenv("CBSPIDER_ROOT"))
	cblogger.Infof("CBKeyPairPath : [%s]", CBKeyPairPath)
	cblogger.Infof("Final keyPairPath : [%s]", keyPairPath)

	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	request := ecs.CreateCreateKeyPairRequest()
	request.Scheme = "https"

	request.KeyPairName = keyPairReqInfo.IId.NameId

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   keyPairHandler.Region.Zone,
		ResourceType: call.VMKEYPAIR,
		ResourceName: keyPairReqInfo.IId.NameId,
		CloudOSAPI:   "CreateKeyPair()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	// Creates a new  key pair with the given name
	result, err := keyPairHandler.Client.CreateKeyPair(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Errorf("Unable to create key pair: %s, %v.", keyPairReqInfo.IId.NameId, err)
		return irs.KeyPairInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Infof("Created key pair %q %s\n%s\n", result.KeyPairName, result.KeyPairFingerPrint, result.PrivateKeyBody)
	spew.Dump(result)

	cblogger.Info("공개키 생성")
	publicKey, errPub := keypair.MakePublicKeyFromPrivateKey(result.PrivateKeyBody)
	if errPub != nil {
		cblogger.Error(errPub)
		return irs.KeyPairInfo{}, err
	}

	cblogger.Infof("Public Key")
	spew.Dump(publicKey)

	keyPairInfo := irs.KeyPairInfo{
		IId:         irs.IID{NameId: result.KeyPairName, SystemId: result.KeyPairName},
		Fingerprint: result.KeyPairFingerPrint,
		PrivateKey:  result.PrivateKeyBody,
		PublicKey:   publicKey,
		KeyValueList: []irs.KeyValue{
			{Key: "KeyMaterial", Value: result.PrivateKeyBody},
		},
	}

	hashString := strings.ReplaceAll(keyPairInfo.Fingerprint, ":", "") // 필요한 경우 리전 정보 추가하면 될 듯. 나중에 키 이름과 리전으로 암복호화를 진행하면 될 것같음.
	savePrivateFileTo := keyPairPath + hashString + ".pem"
	savePublicFileTo := keyPairPath + hashString + ".pub"
	//cblogger.Infof("hashString : [%s]", hashString)
	cblogger.Infof("savePrivateFileTo : [%s]", savePrivateFileTo)
	cblogger.Infof("savePublicFileTo : [%s]", savePublicFileTo)

	// 파일에 private Key를 쓴다
	err = keypair.SaveKey([]byte(keyPairInfo.PrivateKey), savePrivateFileTo)
	if err != nil {
		return irs.KeyPairInfo{}, err
	}

	// 파일에 public Key를 쓴다
	err = keypair.SaveKey([]byte(keyPairInfo.PublicKey), savePublicFileTo)
	if err != nil {
		return irs.KeyPairInfo{}, err
	}

	return keyPairInfo, nil
}

// 혼선을 피하기 위해 keyPairID 대신 keyPairName으로 변경 함.
func (keyPairHandler *AlibabaKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {
	//keyPairID := keyPairName
	cblogger.Infof("GetKey(keyPairName) : [%s]", keyIID.SystemId)

	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	cblogger.Infof("Getenv[CBSPIDER_ROOT] : [%s]", os.Getenv("CBSPIDER_ROOT"))
	cblogger.Infof("CBKeyPairPath : [%s]", CBKeyPairPath)
	cblogger.Infof("Final keyPairPath : [%s]", keyPairPath)

	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}

	request := ecs.CreateDescribeKeyPairsRequest()
	request.Scheme = "https"
	request.KeyPairName = keyIID.SystemId

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   keyPairHandler.Region.Zone,
		ResourceType: call.VMKEYPAIR,
		ResourceName: keyIID.NameId,
		CloudOSAPI:   "DescribeKeyPairs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	result, err := keyPairHandler.Client.DescribeKeyPairs(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		// if aerr, ok := err.(errors.Error); ok {
		cblogger.Errorf("Unable to get key pair: %s, %v.", keyIID.SystemId, err)
		return irs.KeyPairInfo{}, nil
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Info("result : ", result)
	if result.TotalCount < 1 {
		return irs.KeyPairInfo{}, errors.New("Notfound: '" + keyIID.SystemId + "' KeyPair Not found")
	}

	keyPairInfo, errKeyPair := ExtractKeyPairDescribeInfo(&result.KeyPairs.KeyPair[0])
	if errKeyPair != nil {
		cblogger.Error(errKeyPair.Error())
		return irs.KeyPairInfo{}, errKeyPair
	}

	return keyPairInfo, nil
}

// KeyPair 정보를 추출함
func ExtractKeyPairDescribeInfo(keyPair *ecs.KeyPair) (irs.KeyPairInfo, error) {
	spew.Dump(keyPair)

	keyPairInfo := irs.KeyPairInfo{
		IId:         irs.IID{NameId: keyPair.KeyPairName, SystemId: keyPair.KeyPairName},
		Fingerprint: keyPair.KeyPairFingerPrint,
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
		//{Key: "ResourceGroupId", Value: keyPair.ResourceGroupId},
		{Key: "CreationTime", Value: keyPair.CreationTime},
	}

	keyPairInfo.KeyValueList = keyValueList

	return keyPairInfo, nil
}

func (keyPairHandler *AlibabaKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error) {
	cblogger.Infof("DeleteKey(KeyPairName) : [%s]", keyIID.SystemId)
	// Delete the key pair by name

	//없는 키도 무조건 성공하기 때문에 미리 조회함.
	keyPairInfo, errKey := keyPairHandler.GetKey(keyIID)
	if errKey != nil {
		cblogger.Errorf("[%s] KeyPair Delete fail", keyIID.SystemId)
		cblogger.Error(errKey)
		return false, errKey
	}

	request := ecs.CreateDeleteKeyPairsRequest()
	request.Scheme = "https"
	request.KeyPairNames = "[" + "\"" + keyIID.SystemId + "\"]"

	spew.Dump(request)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   keyPairHandler.Region.Zone,
		ResourceType: call.VMKEYPAIR,
		ResourceName: keyIID.NameId,
		CloudOSAPI:   "DeleteKeyPairs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	result, err := keyPairHandler.Client.DeleteKeyPairs(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Errorf("Unable to delete key pair: %s, %v.", keyIID.SystemId, err)
		return false, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Info(result)
	cblogger.Infof("Successfully deleted %q Alibaba Cloud key pair\n", keyIID.SystemId)

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
func (keyPairHandler *AlibabaKeyPairHandler) CheckKeyPairFolder(keyPairPath string) error {
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
