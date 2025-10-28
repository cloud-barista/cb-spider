package resources

import (
	"errors"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
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
	//cblogger.Debug(response)
	cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	for _, pair := range response.Response.KeyPairSet {
		keyPairInfo, errKeyPair := ExtractKeyPairDescribeInfo(pair)
		if errKeyPair != nil {
			// 2021-10-27 이슈#480에 의해 Local Key 로직 제거
			//cblogger.Infof("[%s] KeyPair는 Local에서 관리하는 대상이 아니기 때문에 Skip합니다.", *pair.KeyName)
			cblogger.Error(errKeyPair.Error())
			//return nil, errKeyPair
		} else {
			keyPairList = append(keyPairList, &keyPairInfo)
		}
	}

	return keyPairList, nil
}

// 2021-10-27 이슈#480에 의해 Local Key 로직 제거
// KeyPair 정보를 추출함
func ExtractKeyPairDescribeInfo(keyPair *cvm.KeyPair) (irs.KeyPairInfo, error) {
	cblogger.Debug(keyPair)
	keyPairInfo := irs.KeyPairInfo{
		IId: irs.IID{NameId: *keyPair.KeyName, SystemId: *keyPair.KeyId},
		//PublicKey: *keyPair.PublicKey,
	}
	if keyPair.PrivateKey != nil {
		keyPairInfo.PrivateKey = *keyPair.PrivateKey
	}

	cblogger.Info(" keyPair.Tags", keyPair.Tags)
	if keyPair.Tags != nil {
		var tagList []irs.KeyValue
		for _, tag := range keyPair.Tags {
			tagList = append(tagList, irs.KeyValue{
				Key:   *tag.Key,
				Value: *tag.Value,
			})
		}
		keyPairInfo.TagList = tagList
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
	// 2025-03-13 StructToKeyValueList 사용으로 변경
	keyPairInfo.KeyValueList = irs.StructToKeyValueList(keyPair)
	// keyValueList := []irs.KeyValue{
	// 	{Key: "KeyId", Value: *keyPair.KeyId},
	// 	//{Key: "KeyMaterial", Value: *keyPair.KeyMaterial},
	// }

	// keyPairInfo.KeyValueList = keyValueList

	return keyPairInfo, nil
}

// 2021-10-27 이슈#480에 의해 Local Key 로직 제거
// KeyPair 생성시 이름은 알파벳, 숫자 또는 밑줄 "_"만 지원
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

	var tags []*cvm.Tag
	for _, inputTag := range keyPairReqInfo.TagList {
		tags = append(tags, &cvm.Tag{
			Key:   common.StringPtr(inputTag.Key),
			Value: common.StringPtr(inputTag.Value),
		})
	}

	if len(tags) > 0 {
		request.TagSpecification = []*cvm.TagSpecification{
			{
				ResourceType: common.StringPtr(string(irs.KEY)),
				Tags:         tags,
			},
		}
	}

	callLogStart := call.Start()
	response, err := keyPairHandler.Client.CreateKeyPair(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return irs.KeyPairInfo{}, err
	}
	//cblogger.Debug(response)
	cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	cblogger.Infof("Created key pair", *response.Response.KeyPair)
	//cblogger.Debug(result)
	keyPairInfo, errKeyPair := ExtractKeyPairDescribeInfo(response.Response.KeyPair)
	if errKeyPair != nil {
		cblogger.Error(errKeyPair.Error())
		return irs.KeyPairInfo{}, errKeyPair
	}

	// keyPairInfo := irs.KeyPairInfo{
	// 	//Name:        *result.KeyName,
	// 	IId:        irs.IID{NameId: keyPairReqInfo.IId.NameId, SystemId: *response.Response.KeyPair.KeyId},
	// 	PublicKey:  *response.Response.KeyPair.PublicKey,
	// 	PrivateKey: *response.Response.KeyPair.PrivateKey,
	// 	KeyValueList: []irs.KeyValue{
	// 		{Key: "KeyId", Value: *response.Response.KeyPair.KeyId},
	// 	},
	// }

	// //

	//cblogger.Debug(keyPairInfo)

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

	cblogger.Infof("SSH Key - KeyId:[%s] / KeyName:[%s]", *response.Response.KeyPairSet[0].KeyId, *response.Response.KeyPairSet[0].KeyName)
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
	//cblogger.Debug(response)
	cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	if *response.Response.TotalCount > 0 {
		keyPairInfo, errKeyPair := ExtractKeyPairDescribeInfo(response.Response.KeyPairSet[0])
		if errKeyPair != nil {
			cblogger.Error(errKeyPair.Error())
			return irs.KeyPairInfo{}, errKeyPair
		}

		//cblogger.Debug(keyPairInfo)
		return keyPairInfo, nil
	} else {
		return irs.KeyPairInfo{}, errors.New("I couldn't find the information.")
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
	cblogger.Infof("Key pair requested for deletion: [%s]", keyIID.SystemId)

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
	//cblogger.Debug(response)
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

func (keyPairHandler *TencentKeyPairHandler) ListIID() ([]*irs.IID, error) {
	var iidList []*irs.IID

	callLogInfo := GetCallLogScheme(keyPairHandler.Region, call.VMKEYPAIR, "ListIID", "DescribeKeyPairs()")

	request := cvm.NewDescribeKeyPairsRequest()

	start := call.Start()
	response, err := keyPairHandler.Client.DescribeKeyPairs(request)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		cblogger.Error(err)
		return nil, err
	}
	calllogger.Debug(call.String(callLogInfo))
	cblogger.Debug("keyPair Count : ", *response.Response.TotalCount)
	for _, pair := range response.Response.KeyPairSet {
		iid := irs.IID{SystemId: *pair.KeyId}
		iidList = append(iidList, &iid)
	}

	return iidList, nil
}
