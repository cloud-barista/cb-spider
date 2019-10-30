package resources

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type AwsKeyPairHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
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
	cblogger.Info(keyPairHandler)

	input := &ec2.DescribeKeyPairsInput{
		KeyNames: []*string{
			nil,
		},
	}

	//  Returns a list of key pairs
	result, err := keyPairHandler.Client.DescribeKeyPairs(input)
	cblogger.Info(result)
	if err != nil {
		cblogger.Errorf("Unable to get key pairs, %v", err)
		return keyPairList, err
	}

	//cblogger.Debugf("Key Pairs:")
	for _, pair := range result.KeyPairs {
		/*
			cblogger.Debugf("%s: %s\n", *pair.KeyName, *pair.KeyFingerprint)
			keyPairInfo := new(irs.KeyPairInfo)
			keyPairInfo.Name = *pair.KeyName
			keyPairInfo.Fingerprint = *pair.KeyFingerprint
		*/
		keyPairInfo := ExtractKeyPairDescribeInfo(pair)
		keyPairList = append(keyPairList, &keyPairInfo)
	}

	cblogger.Info(keyPairList)
	spew.Dump(keyPairList)
	return keyPairList, nil
}

func (keyPairHandler *AwsKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	cblogger.Info(keyPairReqInfo)

	// Creates a new  key pair with the given name
	result, err := keyPairHandler.Client.CreateKeyPair(&ec2.CreateKeyPairInput{
		KeyName: aws.String(keyPairReqInfo.Name),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "InvalidKeyPair.Duplicate" {
			cblogger.Errorf("Keypair %q already exists.", keyPairReqInfo.Name)
			return irs.KeyPairInfo{}, err
		}
		cblogger.Errorf("Unable to create key pair: %s, %v.", keyPairReqInfo.Name, err)
		return irs.KeyPairInfo{}, err
	}

	cblogger.Infof("Created key pair %q %s\n%s\n", *result.KeyName, *result.KeyFingerprint, *result.KeyMaterial)
	spew.Dump(result)
	keyPairInfo := irs.KeyPairInfo{
		Name:        *result.KeyName,
		Fingerprint: *result.KeyFingerprint,
		PrivateKey:  *result.KeyMaterial, // AWS(PEM파일-RSA PRIVATE KEY)
		//KeyMaterial: *result.KeyMaterial,
		KeyValueList: []irs.KeyValue{
			{Key: "KeyMaterial", Value: *result.KeyMaterial},
		},
	}

	return keyPairInfo, nil
}

//혼선을 피하기 위해 keyPairID 대신 keyName으로 변경 함.
func (keyPairHandler *AwsKeyPairHandler) GetKey(keyName string) (irs.KeyPairInfo, error) {
	//keyPairID := keyName
	cblogger.Infof("keyName : [%s]", keyName)
	input := &ec2.DescribeKeyPairsInput{
		KeyNames: []*string{
			aws.String(keyName),
		},
	}

	result, err := keyPairHandler.Client.DescribeKeyPairs(input)
	cblogger.Info("result : ", result)
	cblogger.Info("err : ", err)

	if err != nil {
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

	if len(result.KeyPairs) > 0 {
		keyPairInfo := ExtractKeyPairDescribeInfo(result.KeyPairs[0])
		return keyPairInfo, nil
	} else {
		return irs.KeyPairInfo{}, errors.New("정보를 찾을 수 없습니다.")
	}
}

//KeyPair 정보를 추출함
func ExtractKeyPairDescribeInfo(keyPair *ec2.KeyPairInfo) irs.KeyPairInfo {
	spew.Dump(keyPair)
	keyPairInfo := irs.KeyPairInfo{
		Name:        *keyPair.KeyName,
		Fingerprint: *keyPair.KeyFingerprint,
	}

	keyValueList := []irs.KeyValue{
		//{Key: "KeyMaterial", Value: *keyPair.KeyMaterial},
	}

	keyPairInfo.KeyValueList = keyValueList

	return keyPairInfo
}

func (keyPairHandler *AwsKeyPairHandler) DeleteKey(keyName string) (bool, error) {
	cblogger.Infof("삭제 요청된 키페어 : [%s]", keyName)

	_, errGet := keyPairHandler.GetKey(keyName)
	if errGet != nil {
		return false, errGet
	}

	// Delete the key pair by name
	result, err := keyPairHandler.Client.DeleteKeyPair(&ec2.DeleteKeyPairInput{
		KeyName: aws.String(keyName),
	})

	spew.Dump(result)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "InvalidKeyPair.Duplicate" {
			cblogger.Error("Key pair %q does not exist.", keyName)
			return false, err
		}
		cblogger.Errorf("Unable to delete key pair: %s, %v.", keyName, err)
		return false, err
	}

	cblogger.Infof("Successfully deleted %q key pair\n", keyName)

	return true, nil
}
