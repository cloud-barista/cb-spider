// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is OpenStack Driver.
//
// by CB-Spider Team, 2022.09.

package resources

import (
	"errors"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	_ "github.com/gophercloud/gophercloud"
	_ "github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"

	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
)

type OpenStackAnyCallHandler struct {
	Region         idrv.RegionInfo
	CredentialInfo idrv.CredentialInfo
}

/********************************************************
	// call example
	curl -sX POST http://localhost:1024/spider/anycall -H 'Content-Type: application/json' -d \
        '{
                "ConnectionName" : "openstack-config01",
                "ReqInfo" : {
                        "FID" : "getConnectionInfo"
                }
        }' | json_pp
********************************************************/
func (anyCallHandler *OpenStackAnyCallHandler) AnyCall(callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	cblogger.Info("OpenStack Driver: called AnyCall()!")

	switch callInfo.FID {
	case "getConnectionInfo":
		return getConnectionInfo(anyCallHandler, callInfo)

	// add more ...

	default:
		return irs.AnyCallInfo{}, errors.New("OpenStack Driver: " + callInfo.FID + " Function is not implemented!")
	}
}

///////////////////////////////////////////////////////////////////
// implemented by developer user, like 'getConnectionInfo() ConnectionInfo'
///////////////////////////////////////////////////////////////////
func getConnectionInfo(anyCallHandler *OpenStackAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
	cblogger.Info("OpenStack Driver: called AnyCall()/getConnectionInfo()!")

	// encryption and make results
	if callInfo.OKeyValueList == nil {
		callInfo.OKeyValueList = []irs.KeyValue{}
	}
	callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"IdentityEndpoint",
		tmpEncryptAndEncode(anyCallHandler.CredentialInfo.IdentityEndpoint)})
	callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"DomainName",
		tmpEncryptAndEncode(anyCallHandler.CredentialInfo.DomainName)})
	callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"ProjectID",
		tmpEncryptAndEncode(anyCallHandler.CredentialInfo.ProjectID)})
	callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Username",
		tmpEncryptAndEncode(anyCallHandler.CredentialInfo.Username)})
	callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Password",
		tmpEncryptAndEncode(anyCallHandler.CredentialInfo.Password)})

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
