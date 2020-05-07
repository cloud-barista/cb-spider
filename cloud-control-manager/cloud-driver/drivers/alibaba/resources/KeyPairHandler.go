// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by zephy@mz.co.kr, 2019.09.

package resources

import (
	"errors"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

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

	//  Returns a list of key pairs
	result, err := keyPairHandler.Client.DescribeKeyPairs(request)
	cblogger.Info(result)
	if err != nil {
		cblogger.Errorf("Unable to get key pairs, %v", err)
		return keyPairList, err
	}

	//cblogger.Debugf("Key Pairs:")
	for _, pair := range result.KeyPairs.KeyPair {
		keyPairInfo := ExtractKeyPairDescribeInfo(&pair)
		keyPairList = append(keyPairList, &keyPairInfo)
	}

	cblogger.Info(keyPairList)
	//spew.Dump(keyPairList)
	return keyPairList, nil
}

func (keyPairHandler *AlibabaKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	cblogger.Info("Start CreateKey() : ", keyPairReqInfo)

	request := ecs.CreateCreateKeyPairRequest()
	request.Scheme = "https"

	request.KeyPairName = keyPairReqInfo.IId.NameId

	// Creates a new  key pair with the given name
	result, err := keyPairHandler.Client.CreateKeyPair(request)
	if err != nil {
		cblogger.Errorf("Unable to create key pair: %s, %v.", keyPairReqInfo.IId.NameId, err)
		return irs.KeyPairInfo{}, err
	}

	cblogger.Infof("Created key pair %q %s\n%s\n", result.KeyPairName, result.KeyPairFingerPrint, result.PrivateKeyBody)
	spew.Dump(result)
	keyPairInfo := irs.KeyPairInfo{
		IId:         irs.IID{NameId: result.KeyPairName, SystemId: result.KeyPairName},
		Fingerprint: result.KeyPairFingerPrint,
		PrivateKey:  result.PrivateKeyBody,
		KeyValueList: []irs.KeyValue{
			{Key: "KeyMaterial", Value: result.PrivateKeyBody},
		},
	}

	return keyPairInfo, nil
}

// 혼선을 피하기 위해 keyPairID 대신 keyPairName으로 변경 함.
func (keyPairHandler *AlibabaKeyPairHandler) GetKey(keyIID irs.IID) (irs.KeyPairInfo, error) {
	//keyPairID := keyPairName
	cblogger.Infof("GetKey(keyPairName) : [%s]", keyIID.SystemId)

	request := ecs.CreateDescribeKeyPairsRequest()
	request.Scheme = "https"
	request.KeyPairName = keyIID.SystemId

	result, err := keyPairHandler.Client.DescribeKeyPairs(request)
	if err != nil {
		// if aerr, ok := err.(errors.Error); ok {
		cblogger.Errorf("Unable to get key pair: %s, %v.", keyIID.SystemId, err)
		return irs.KeyPairInfo{}, nil
	}
	cblogger.Info("result : ", result)
	if result.TotalCount < 1 {
		return irs.KeyPairInfo{}, errors.New("Notfound: '" + keyIID.SystemId + "' KeyPair Not found")
	}

	keyPairInfo := ExtractKeyPairDescribeInfo(&result.KeyPairs.KeyPair[0])
	return keyPairInfo, nil
}

// KeyPair 정보를 추출함
func ExtractKeyPairDescribeInfo(keyPair *ecs.KeyPair) irs.KeyPairInfo {
	spew.Dump(keyPair)

	keyPairInfo := irs.KeyPairInfo{
		IId:         irs.IID{NameId: keyPair.KeyPairName, SystemId: keyPair.KeyPairName},
		Fingerprint: keyPair.KeyPairFingerPrint,
	}

	keyValueList := []irs.KeyValue{
		//{Key: "ResourceGroupId", Value: keyPair.ResourceGroupId},
		{Key: "CreationTime", Value: keyPair.CreationTime},
	}

	keyPairInfo.KeyValueList = keyValueList

	return keyPairInfo
}

func (keyPairHandler *AlibabaKeyPairHandler) DeleteKey(keyIID irs.IID) (bool, error) {
	cblogger.Infof("DeleteKey(KeyPairName) : [%s]", keyIID.SystemId)
	// Delete the key pair by name

	//없는 키도 무조건 성공하기 때문에 미리 조회함.
	_, errKey := keyPairHandler.GetKey(keyIID)
	if errKey != nil {
		cblogger.Errorf("[%s] KeyPair Delete fail", keyIID.SystemId)
		cblogger.Error(errKey)
		return false, errKey
	}

	request := ecs.CreateDeleteKeyPairsRequest()
	request.Scheme = "https"
	request.KeyPairNames = "[" + "\"" + keyIID.SystemId + "\"]"

	spew.Dump(request)
	result, err := keyPairHandler.Client.DeleteKeyPairs(request)
	cblogger.Info(result)
	if err != nil {
		cblogger.Errorf("Unable to delete key pair: %s, %v.", keyIID.SystemId, err)
		return false, err
	}

	cblogger.Infof("Successfully deleted %q key pair\n", keyIID.SystemId)

	return true, nil
}
