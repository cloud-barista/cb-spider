// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is AWS Driver.
//
// by CB-Spider Team, 2022.09.

package resources

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsAnyCallHandler struct {
	Region         idrv.RegionInfo
	CredentialInfo idrv.CredentialInfo
	Client         *ec2.EC2
}

/********************************************************
        // call example
        curl -sX POST http://localhost:1024/spider/anycall -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName" : "aws-ohio-config",
                "ReqInfo" : {
                        "FID" : "createTags",
                        "IKeyValueList" : [{"Key":"key1", "Value":"value1"}, {"Key":"key2", "Value":"value2"}]
                }
        }' | json_pp
********************************************************/
func (anyCallHandler *AwsAnyCallHandler) AnyCall(callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	cblogger.Info("AWS Driver: called AnyCall()!")

	switch callInfo.FID {
	case "createTags":
		return createTags(anyCallHandler, callInfo)
	case "associateIamInstanceProfile":
		return associateIamInstanceProfile(anyCallHandler, callInfo)
	case "getRegionInfo":
		return getRegionInfo(anyCallHandler, callInfo)
	// add more ...

	default:
		return irs.AnyCallInfo{}, errors.New("AWS Driver: " + callInfo.FID + " Function is not implemented!")
	}
}

///////////////////////////////////////////////////////////////////////
// implemented by developer user, like 'createTags(kv []KeyVale) bool'
///////////////////////////////////////////////////////////////////////
const (
	CreateTagsResourceId = "ResourceId"
	CreateTagsTag        = "Tag"
)

func createTags(anyCallHandler *AwsAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	cblogger.Info("AWS Driver: called AnyCall()/createTags()!")

	if anyCallHandler.Client == nil {
		return irs.AnyCallInfo{}, errors.New("AWS Driver: " + callInfo.FID + " has no session")
	}

	// make results
	if callInfo.OKeyValueList == nil {
		callInfo.OKeyValueList = []irs.KeyValue{}
	}

	// Input Arg Validation
	if callInfo.IKeyValueList == nil {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "false"})
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Reason", fmt.Sprintf("Input Argument is empty!")})
		return callInfo, nil
	}

	var resourceId string
	var tags []*ec2.Tag

	// run
	for _, kv := range callInfo.IKeyValueList {
		if kv.Key == CreateTagsResourceId {
			resourceId = kv.Value
		} else if kv.Key == CreateTagsTag {
			var kvTag irs.KeyValue
			if err := json.Unmarshal([]byte(kv.Value), &kvTag); err != nil {
				//return irs.AnyCallInfo{}, errors.New("AWS Driver: "+callInfo.FID+"'s argument(key=%s, value=%s) is invalid", kv.Key, kv.Value)
				callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "false"})
				callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Reason", fmt.Sprintf("Input Argument(key=%s,value=%s) is invalid!(err=%v)", kv.Key, kv.Value, err)})
				return callInfo, nil
			}
			tags = append(tags, &ec2.Tag{Key: &kvTag.Key, Value: &kvTag.Value})
		}
	}

	input := &ec2.CreateTagsInput{
		Resources: []*string{aws.String(resourceId)},
		Tags:      tags,
	}

	cblogger.Info("input:", input)

	output, err := anyCallHandler.Client.CreateTags(input)
	if err != nil {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "false"})
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Reason", fmt.Sprintf("got error: %v", err)})
		return callInfo, nil
	}

	cblogger.Info("output:", output)

	callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "true"})
	return callInfo, nil
}

const (
	AssociateIamInstanceProfileInstanceId = "InstanceId"
	AssociateIamInstanceProfileRole       = "Role"
)

///////////////////////////////////////////////////////////////////
// implemented by developer user, like 'associateIamInstanceProfile(kv []KeyValue) bool'
///////////////////////////////////////////////////////////////////
func associateIamInstanceProfile(anyCallHandler *AwsAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	cblogger.Info("AWS Driver: called AnyCall()/associateIamInstanceProfile()!")

	if anyCallHandler.Client == nil {
		return irs.AnyCallInfo{}, errors.New("AWS Driver: " + callInfo.FID + " has no session")
	}

	// make results
	if callInfo.OKeyValueList == nil {
		callInfo.OKeyValueList = []irs.KeyValue{}
	}

	// Input Arg Validation
	if callInfo.IKeyValueList == nil {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "false"})
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Reason", fmt.Sprintf("Input Argument is empty!")})
		return callInfo, nil
	}

	var instanceId string
	var role string

	// run
	for _, kv := range callInfo.IKeyValueList {
		if kv.Key == AssociateIamInstanceProfileInstanceId {
			instanceId = kv.Value
		} else if kv.Key == AssociateIamInstanceProfileRole {
			role = kv.Value
		}
	}

	input := &ec2.AssociateIamInstanceProfileInput{
		IamInstanceProfile: &ec2.IamInstanceProfileSpecification{
			Name: &role,
		},
		InstanceId: &instanceId,
	}

	cblogger.Info("input:", input)

	output, err := anyCallHandler.Client.AssociateIamInstanceProfile(input)
	if err != nil {
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "false"})
		callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Reason", fmt.Sprintf("got error: %v", err)})
		return callInfo, nil
	}

	cblogger.Info("output:", output)

	callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Result", "true"})
	return callInfo, nil
}

func getRegionInfo(anyCallHandler *AwsAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	cblogger.Info("AWS Driver: called AnyCall()/getRegionInfo()!")

	// encryption and make results
	if callInfo.OKeyValueList == nil {
		callInfo.OKeyValueList = []irs.KeyValue{}
	}

	callInfo.OKeyValueList = append(callInfo.OKeyValueList,
		irs.KeyValue{"Region", tmpEncryptAndEncode(anyCallHandler.Region.Region)})
	callInfo.OKeyValueList = append(callInfo.OKeyValueList,
		irs.KeyValue{"Zone", tmpEncryptAndEncode(anyCallHandler.Region.Zone)})

	return callInfo, nil
}

// exmaples
func tmpEncryptAndEncode(i string) string {
	// Implement to encrypt secure info
	// ref) encryptKeyValueList() and decryptKeyValueList() in cloud-info-manager/credential-info-manager/CredentialInfoManager.go
	// this is example codes
	encb, _ := encrypt([]byte(i))
	sEnc := base64.StdEncoding.EncodeToString(encb)
	return sEnc
}

// examples: encryption with spider key
func encrypt(contents []byte) ([]byte, error) {
	var spider_key = []byte("cloud-barista-cb-spider-cloud-ba") // 32 bytes

	encryptData := make([]byte, aes.BlockSize+len(contents))
	initVector := encryptData[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, initVector); err != nil {
		return nil, err
	}

	cipherBlock, err := aes.NewCipher(spider_key)
	if err != nil {
		return nil, err
	}
	cipherTextFB := cipher.NewCFBEncrypter(cipherBlock, initVector)
	cipherTextFB.XORKeyStream(encryptData[aes.BlockSize:], []byte(contents))

	return encryptData, nil
}
