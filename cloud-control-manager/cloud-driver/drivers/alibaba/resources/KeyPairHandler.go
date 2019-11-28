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
		/*
			cblogger.Debugf("%s: %s\n", *pair.KeyName, *pair.KeyFingerprint)
			keyPairInfo := new(irs.KeyPairInfo)
			keyPairInfo.Name = *pair.KeyName
			keyPairInfo.Fingerprint = *pair.KeyFingerprint
		*/
		keyPairInfo := ExtractKeyPairDescribeInfo(&pair)
		keyPairList = append(keyPairList, &keyPairInfo)
	}

	cblogger.Info(keyPairList)
	spew.Dump(keyPairList)
	return keyPairList, nil
}

func (keyPairHandler *AlibabaKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	cblogger.Info("Start CreateKey() : ", keyPairReqInfo)

	request := ecs.CreateCreateKeyPairRequest()
	request.Scheme = "https"

	request.KeyPairName = keyPairReqInfo.Name

	// Creates a new  key pair with the given name
	result, err := keyPairHandler.Client.CreateKeyPair(request)
	if err != nil {
		// if aerr, ok := err.(errors.Error); ok && aerr.Code() == "InvalidKeyPair.Duplicate" {
		// 	cblogger.Errorf("Keypair %q already exists.", keyPairReqInfo.Name)
		// 	return irs.KeyPairInfo{}, err
		// }
		cblogger.Errorf("Unable to create key pair: %s, %v.", keyPairReqInfo.Name, err)
		return irs.KeyPairInfo{}, err
	}

	cblogger.Infof("Created key pair %q %s\n%s\n", result.KeyPairName, result.KeyPairFingerPrint, result.PrivateKeyBody)
	spew.Dump(result)
	keyPairInfo := irs.KeyPairInfo{
		Name:        result.KeyPairName,
		Fingerprint: result.KeyPairFingerPrint,
		//KeyMaterial: *result.PrivateKeyBody,
		KeyValueList: []irs.KeyValue{
			{Key: "KeyMaterial", Value: result.PrivateKeyBody},
		},
	}

	return keyPairInfo, nil
}

// 혼선을 피하기 위해 keyPairID 대신 keyPairName으로 변경 함.
func (keyPairHandler *AlibabaKeyPairHandler) GetKey(keyPairName string) (irs.KeyPairInfo, error) {
	//keyPairID := keyPairName
	cblogger.Infof("GetKey(keyPairName) : [%s]", keyPairName)

	request := ecs.CreateDescribeKeyPairsRequest()
	request.Scheme = "https"

	request.KeyPairName = keyPairName

	result, err := keyPairHandler.Client.DescribeKeyPairs(request)
	cblogger.Info("result : ", result)
	cblogger.Info("err : ", err)

	if err != nil {
		// if aerr, ok := err.(errors.Error); ok {
		// 	cblogger.Info("aerr : ", aerr)
		// 	cblogger.Info("aerr.Code()  : ", aerr.Code())
		// 	cblogger.Info("ok : ", ok)
		// 	switch aerr.Code() {
		// 	default:
		// 		//fmt.Println(aerr.Error())
		// 		cblogger.Error(aerr.Error())
		// 		return irs.KeyPairInfo{}, aerr
		// 	}
		// } else {
		// 	// Print the error, cast err to awserr.Error to get the Code and Message from an error.
		// 	cblogger.Error(err.Error())
		// 	return irs.KeyPairInfo{}, err
		// }
		cblogger.Errorf("Unable to get key pair: %s, %v.", keyPairName, err)
		return irs.KeyPairInfo{}, nil
	}

	/*
		cblogger.Info("KeyName : ", *result.KeyPairs[0].KeyName)
		cblogger.Info("Fingerprint : ", *result.KeyPairs[0].KeyFingerprint)
		keyPairInfo := irs.KeyPairInfo{
			Name:        *result.KeyPairs[0].KeyName,
			Fingerprint: *result.KeyPairs[0].KeyFingerprint,
		}
	*/
	keyPairInfo := ExtractKeyPairDescribeInfo(&result.KeyPairs.KeyPair[0])

	return keyPairInfo, nil
}

// KeyPair 정보를 추출함
func ExtractKeyPairDescribeInfo(keyPair *ecs.KeyPair) irs.KeyPairInfo {
	spew.Dump(keyPair)

	keyPairInfo := irs.KeyPairInfo{
		Name:        keyPair.KeyPairName,
		Fingerprint: keyPair.KeyPairFingerPrint,
		// *keyPair.ResourceGroupId
		// *keyPair.Tags
	}

	keyValueList := []irs.KeyValue{
		{Key: "ResourceGroupId", Value: keyPair.ResourceGroupId},
		//{Key: "KeyMaterial", Value: *keyPair.KeyMaterial},
	}
	// Tags := []ecs.Tag{}

	keyPairInfo.KeyValueList = keyValueList

	return keyPairInfo
}

func (keyPairHandler *AlibabaKeyPairHandler) DeleteKey(keyPairName string) (bool, error) {
	cblogger.Infof("DeleteKey(KeyPairName) : [%s]", keyPairName)
	// Delete the key pair by name

	request := ecs.CreateDeleteKeyPairsRequest()
	request.Scheme = "https"

	request.KeyPairNames = keyPairName

	result, err := keyPairHandler.Client.DeleteKeyPairs(request)
	cblogger.Info(result)
	if err != nil {
		// if aerr, ok := err.(errors.Error); ok && aerr.Code() == "InvalidKeyPair.Duplicate" {
		// 	cblogger.Error("Key pair %q does not exist.", keyPairName)
		// 	return false, err
		// }
		cblogger.Errorf("Unable to delete key pair: %s, %v.", keyPairName, err)
		return false, err
	}

	cblogger.Infof("Successfully deleted %q key pair\n", keyPairName)

	return true, nil
}
