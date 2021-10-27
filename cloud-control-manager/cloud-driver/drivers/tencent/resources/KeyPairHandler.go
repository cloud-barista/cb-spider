package resources

import (
	"errors"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	_ "github.com/davecgh/go-spew/spew"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
)

type TencentKeyPairHandler struct {
	Region idrv.RegionInfo
	Client *cvm.Client
}

func (keyPairHandler *TencentKeyPairHandler) ListKey() ([]*irs.KeyPairInfo, error) {
	var keyPairList []*irs.KeyPairInfo
	cblogger.Debug("Start ListKey()")

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   keyPairHandler.Region.Zone,
		ResourceType: call.VMKEYPAIR,
		ResourceName: "ListKey()",
		CloudOSAPI:   "DescribeKeyPairs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := cvm.NewDescribeKeyPairsRequest()

	callLogStart := call.Start()
	response, err := keyPairHandler.Client.DescribeKeyPairs(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return nil, err
	}
	//spew.Dump(response)
	cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	for _, pair := range response.Response.KeyPairSet {
		keyPairInfo, errKeyPair := ExtractKeyPairDescribeInfo(pair)
		if errKeyPair != nil {
			// 2021-10-27 이슈#480에 의해 Local Key 로직 제거
			//cblogger.Infof("[%s] KeyPair는 Local에서 관리하는 대상이 아니기 때문에 Skip합니다.", *pair.KeyName)
			cblogger.Info(errKeyPair.Error())
			//return nil, errKeyPair
		} else {
			keyPairList = append(keyPairList, &keyPairInfo)
		}
	}

	return keyPairList, nil
}

// 2021-10-27 이슈#480에 의해 Local Key 로직 제거
//KeyPair 정보를 추출함
func ExtractKeyPairDescribeInfo(keyPair *cvm.KeyPair) (irs.KeyPairInfo, error) {
	spew.Dump(keyPair)
	keyPairInfo := irs.KeyPairInfo{
		IId: irs.IID{NameId: *keyPair.KeyName, SystemId: *keyPair.KeyId},
		//PublicKey: *keyPair.PublicKey,
	}

	//PrivateKey는 최초 생성시에만 존재하며 조회 시에는 PrivateKey는 Nil임.
	// if !reflect.ValueOf(keyPair.PrivateKey).IsNil() {
	// 	keyPairInfo.PrivateKey = *keyPair.PrivateKey
	// 	keyPairInfo.PublicKey = *keyPair.PublicKey
	// }
	//조회 용도

	/* 2021-10-27 이슈#480에 의해 Local Key 로직 제거
	// Local Keyfile 처리
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	hashString := strings.ReplaceAll(*keyPair.KeyId, ":", "") // 필요한 경우 리전 정보 추가하면 될 듯. 나중에 키 이름과 리전으로 암복호화를 진행하면 될 것같음.
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
	*/
	keyValueList := []irs.KeyValue{
		{Key: "KeyId", Value: *keyPair.KeyId},
		//{Key: "KeyMaterial", Value: *keyPair.KeyMaterial},
	}

	keyPairInfo.KeyValueList = keyValueList

	return keyPairInfo, nil
}

// 2021-10-27 이슈#480에 의해 Local Key 로직 제거
//KeyPair 생성시 이름은 알파벳, 숫자 또는 밑줄 "_"만 지원
func (keyPairHandler *TencentKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	cblogger.Info(keyPairReqInfo)

	//=================================================
	// 동일 이름 생성 방지 추가(cb-spider 요청 필수 기능)
	//=================================================
	isExist, errExist := keyPairHandler.isExist(keyPairReqInfo.IId.NameId)
	if errExist != nil {
		cblogger.Error(errExist)
		return irs.KeyPairInfo{}, errExist
	}
	if isExist {
		return irs.KeyPairInfo{}, errors.New("A keyPair with the name " + keyPairReqInfo.IId.NameId + " already exists.")
	}

	/* 2021-10-27 이슈#480에 의해 Local Key 로직 제거
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	cblogger.Infof("Getenv[CBSPIDER_ROOT] : [%s]", os.Getenv("CBSPIDER_ROOT"))
	cblogger.Infof("CBKeyPairPath : [%s]", CBKeyPairPath)
	cblogger.Infof("Final keyPairPath : [%s]", keyPairPath)

	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}
	*/

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   keyPairHandler.Region.Zone,
		ResourceType: call.VMKEYPAIR,
		ResourceName: keyPairReqInfo.IId.NameId,
		CloudOSAPI:   "CreateKeyPair()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := cvm.NewCreateKeyPairRequest()
	request.KeyName = common.StringPtr(keyPairReqInfo.IId.NameId)
	request.ProjectId = common.Int64Ptr(0)

	callLogStart := call.Start()
	response, err := keyPairHandler.Client.CreateKeyPair(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}
	//spew.Dump(response)
	cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	cblogger.Infof("Created [%s]key pair", *response.Response.KeyPair.KeyName)
	//spew.Dump(result)
	keyPairInfo := irs.KeyPairInfo{
		//Name:        *result.KeyName,
		IId:        irs.IID{NameId: keyPairReqInfo.IId.NameId, SystemId: *response.Response.KeyPair.KeyId},
		PublicKey:  *response.Response.KeyPair.PublicKey,
		PrivateKey: *response.Response.KeyPair.PrivateKey,
		KeyValueList: []irs.KeyValue{
			{Key: "KeyId", Value: *response.Response.KeyPair.KeyId},
		},
	}

	//spew.Dump(keyPairInfo)

	/* 2021-10-27 이슈#480에 의해 Local Key 로직 제거
	//=============================
	// 키 페어를 로컬 파일에 기록 함.
	//=============================
	hashString := strings.ReplaceAll(*response.Response.KeyPair.KeyId, ":", "") // 필요한 경우 리전 정보 추가하면 될 듯. 나중에 키 이름과 리전으로 암복호화를 진행하면 될 것같음.
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
	*/
	return keyPairInfo, nil
}

// cb-spider 정책상 이름 기반으로 중복 생성을 막아야 함.
func (keyPairHandler *TencentKeyPairHandler) isExist(chkName string) (bool, error) {
	cblogger.Debugf("chkName : %s", chkName)

	request := cvm.NewDescribeKeyPairsRequest()
	request.Filters = []*cvm.Filter{
		&cvm.Filter{
			Name:   common.StringPtr("key-name"),
			Values: common.StringPtrs([]string{chkName}),
		},
	}

	response, err := keyPairHandler.Client.DescribeKeyPairs(request)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	if *response.Response.TotalCount < 1 {
		return false, nil
	}

	cblogger.Infof("SSH Key 정보 찾음 - KeyId:[%s] / KeyName:[%s]", *response.Response.KeyPairSet[0].KeyId, *response.Response.KeyPairSet[0].KeyName)
	return true, nil
}

// 2021-10-27 이슈#480에 의해 Local Key 로직 제거
func (keyPairHandler *TencentKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {
	//keyPairID := keyName
	cblogger.Infof("keyName : [%s]", keyIID.SystemId)

	/* 2021-10-27 이슈#480에 의해 Local Key 로직 제거
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	cblogger.Infof("Getenv[CBSPIDER_ROOT] : [%s]", os.Getenv("CBSPIDER_ROOT"))
	cblogger.Infof("CBKeyPairPath : [%s]", CBKeyPairPath)
	cblogger.Infof("Final keyPairPath : [%s]", keyPairPath)

	if err := keyPairHandler.CheckKeyPairFolder(keyPairPath); err != nil {
		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}
	*/

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   keyPairHandler.Region.Zone,
		ResourceType: call.VMKEYPAIR,
		ResourceName: keyIID.NameId,
		CloudOSAPI:   "DescribeKeyPairs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := cvm.NewDescribeKeyPairsRequest()
	request.KeyIds = common.StringPtrs([]string{keyIID.SystemId})

	callLogStart := call.Start()
	response, err := keyPairHandler.Client.DescribeKeyPairs(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}
	//spew.Dump(response)
	cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	if *response.Response.TotalCount > 0 {
		keyPairInfo, errKeyPair := ExtractKeyPairDescribeInfo(response.Response.KeyPairSet[0])
		if errKeyPair != nil {
			cblogger.Error(errKeyPair.Error())
			return irs.KeyPairInfo{}, errKeyPair
		}

		//spew.Dump(keyPairInfo)
		return keyPairInfo, nil
	} else {
		return irs.KeyPairInfo{}, errors.New("정보를 찾을 수 없습니다.")
	}
}

/* 2021-10-27 이슈#480에 의해 Local Key 로직 제거
//Tencent의 경우 FingerPrint같은 고유 값을 조회할 수 없기 때문에 KeyId를 로컬 파일의 고유 키 값으로 이용함.
func (keyPairHandler *TencentKeyPairHandler) GetLocalKeyId(keyIID irs.IID) (string, error) {
	//삭제할 Local Keyfile을 찾기 위해 조회
	request := cvm.NewDescribeKeyPairsRequest()
	request.KeyIds = common.StringPtrs([]string{keyIID.SystemId})
	response, err := keyPairHandler.Client.DescribeKeyPairs(request)
	if err != nil {
		cblogger.Error(err)
		return "", err
	}

	if *response.Response.TotalCount > 0 {
		return *response.Response.KeyPairSet[0].KeyId, nil
	} else {
		return "", errors.New("InvalidKeyPair.NotFound: The KeyPair " + keyIID.SystemId + " does not exist")
	}
}
*/

// 2021-10-27 이슈#480에 의해 Local Key 로직 제거
func (keyPairHandler *TencentKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error) {
	cblogger.Infof("삭제 요청된 키페어 : [%s]", keyIID.SystemId)

	/* 2021-10-27 이슈#480에 의해 Local Key 로직 제거
	keyPairId, errGet := keyPairHandler.GetLocalKeyId(keyIID)
	if errGet != nil {
		return false, errGet
	}
	*/

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   keyPairHandler.Region.Zone,
		ResourceType: call.VMKEYPAIR,
		ResourceName: keyIID.NameId,
		CloudOSAPI:   "DeleteKeyPairs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := cvm.NewDeleteKeyPairsRequest()
	request.KeyIds = common.StringPtrs([]string{keyIID.SystemId})

	callLogStart := call.Start()
	response, err := keyPairHandler.Client.DeleteKeyPairs(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return false, err
	}
	//spew.Dump(response)
	cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	/* 2021-10-27 이슈#480에 의해 Local Key 로직 제거
	//====================
	// Local Keyfile 처리
	//====================
	keyPairPath := os.Getenv("CBSPIDER_ROOT") + CBKeyPairPath
	cblogger.Infof("Getenv[CBSPIDER_ROOT] : [%s]", os.Getenv("CBSPIDER_ROOT"))
	cblogger.Infof("CBKeyPairPath : [%s]", CBKeyPairPath)
	cblogger.Infof("Final keyPairPath : [%s]", keyPairPath)

	hashString := strings.ReplaceAll(keyPairId, ":", "") // 필요한 경우 리전 정보 추가하면 될 듯. 나중에 키 이름과 리전으로 암복호화를 진행하면 될 것같음.
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
	*/

	return true, nil
}

/* 2021-10-27 이슈#480에 의해 Local Key 로직 제거
//=================================
// 공개 키 변환 및 키 정보 로컬 보관 로직 추가
//=================================
func (keyPairHandler *TencentKeyPairHandler) CheckKeyPairFolder(keyPairPath string) error {
	//키페어 생성 시 폴더가 존재하지 않으면 생성 함.
	_, errChkDir := os.Stat(keyPairPath)
	if os.IsNotExist(errChkDir) {
		cblogger.Infof("[%s] Path가 존재하지 않아서 생성합니다.", keyPairPath)

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

// @TODO - PK 이슈 처리해야 함. (A User / B User / User 하위의 IAM 계정간의 호환성에 이슈가 없어야 하는데 현재는 안 됨.)
//       - 따라서 AWS는 대안으로 KeyPair의 FingerPrint를 이용하도록 변경 - 필요시 리전및 키 이름과 혼용해서 만들어야할 듯.
// KeyPair 해시 생성 함수 (PK 이슈로 현재는 사용하지 않음)
func CreateHashString(credentialInfo idrv.CredentialInfo, Region idrv.RegionInfo) (string, error) {
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
*/
