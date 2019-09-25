package resources

import (
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

	cblogger.Debugf("Key Pairs:")
	for _, pair := range result.KeyPairs {
		cblogger.Debugf("%s: %s\n", *pair.KeyName, *pair.KeyFingerprint)
		keyPairInfo := new(irs.KeyPairInfo)
		keyPairInfo.Name = *pair.KeyName
		keyPairInfo.Fingerprint = *pair.KeyFingerprint

		keyPairList = append(keyPairList, keyPairInfo)
	}

	cblogger.Info(keyPairList)
	spew.Dump(keyPairList)
	return keyPairList, nil
}

func (keyPairHandler *AwsKeyPairHandler) CreateKey(keyPairReqInfo irs.KeyPairReqInfo) (irs.KeyPairInfo, error) {
	cblogger.Infof("Start CreateKey(%s)", keyPairReqInfo)

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
		KeyMaterial: *result.KeyMaterial,
	}

	return keyPairInfo, nil
}

//혼선을 피하기 위해 keyPairID 대신 keyPairName으로 변경 함.
func (keyPairHandler *AwsKeyPairHandler) GetKey(keyPairName string) (irs.KeyPairInfo, error) {
	//keyPairID := keyPairName
	cblogger.Infof("GetKey : [%s]", keyPairName)
	input := &ec2.DescribeKeyPairsInput{
		KeyNames: []*string{
			aws.String(keyPairName),
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

	cblogger.Info("KeyName : ", *result.KeyPairs[0].KeyName)
	cblogger.Info("Fingerprint : ", *result.KeyPairs[0].KeyFingerprint)

	keyPairInfo := irs.KeyPairInfo{
		Name:        *result.KeyPairs[0].KeyName,
		Fingerprint: *result.KeyPairs[0].KeyFingerprint,
	}

	return keyPairInfo, nil
}

func (keyPairHandler *AwsKeyPairHandler) DeleteKey(keyPairName string) (bool, error) {
	cblogger.Infof("DeleteKeyPaid : [%s]", keyPairName)
	// Delete the key pair by name
	_, err := keyPairHandler.Client.DeleteKeyPair(&ec2.DeleteKeyPairInput{
		KeyName: aws.String(keyPairName),
	})

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "InvalidKeyPair.Duplicate" {
			cblogger.Error("Key pair %q does not exist.", keyPairName)
			return false, err
		}
		cblogger.Errorf("Unable to delete key pair: %s, %v.", keyPairName, err)
		return false, err
	}

	cblogger.Infof("Successfully deleted %q key pair\n", keyPairName)

	return true, nil
}
