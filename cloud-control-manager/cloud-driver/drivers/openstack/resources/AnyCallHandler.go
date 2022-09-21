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
	"fmt"
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
        case "getConnectionInfo" :
                return getConnectionInfo(anyCallHandler, callInfo)

        // add more ...

        default :
                return irs.AnyCallInfo{}, errors.New("OpenStack Driver: " + callInfo.FID + " Function is not implemented!")
        }
}

///////////////////////////////////////////////////////////////////
// implemented by developer user, like 'getConnectionInfo() ConnectionInfo'
///////////////////////////////////////////////////////////////////
func getConnectionInfo(anyCallHandler *OpenStackAnyCallHandler, callInfo irs.AnyCallInfo) (irs.AnyCallInfo, error) {
        cblogger.Info("OpenStack Driver: called AnyCall()/addTag()!")

        // you must delete this line
        fmt.Printf("\n\n\n * Region:%s, *IdentityEndpoint:%s, *DomainName:%s, *TenantId:%s, *Username:%s, *Password:%s\n", 
                anyCallHandler.Region.Region,
                anyCallHandler.CredentialInfo.IdentityEndpoint, 
                anyCallHandler.CredentialInfo.DomainName, 
                anyCallHandler.CredentialInfo.TenantId, 
                anyCallHandler.CredentialInfo.Username, 
		anyCallHandler.CredentialInfo.Password)

        // encryption and make results
        if callInfo.OKeyValueList == nil {
                callInfo.OKeyValueList = []irs.KeyValue{}
        }
	callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"IdentityEndpoint", 
		tmpEncrypt(anyCallHandler.CredentialInfo.IdentityEndpoint)} )
	callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"DomainName", 
		tmpEncrypt(anyCallHandler.CredentialInfo.DomainName)} )
	callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"TenantId", 
		tmpEncrypt(anyCallHandler.CredentialInfo.TenantId)} )
	callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Username", 
		tmpEncrypt(anyCallHandler.CredentialInfo.Username)} )
	callInfo.OKeyValueList = append(callInfo.OKeyValueList, irs.KeyValue{"Password", 
		tmpEncrypt(anyCallHandler.CredentialInfo.Password)} )


        return callInfo, nil
}

// exmaples
func tmpEncrypt(i string) string {
	// Implement to encrypt secure info
	// ref) encryptKeyValueList() and decryptKeyValueList() in cloud-info-manager/credential-info-manager/CredentialInfoManager.go
	// this is example codes
	encb, _ := encrypt([]byte(i))
	return string(encb)
}

// examples: encription with spider key
func encrypt(contents []byte) ([]byte, error) {
	var spider_key = []byte("cloud-barista-cb-spider-cloud-ba") // 32 bytes

        base64Encoding := base64.StdEncoding.EncodeToString(contents)
        encryptData := make([]byte, aes.BlockSize+len(base64Encoding))
        initVector := encryptData[:aes.BlockSize]
        if _, err := io.ReadFull(rand.Reader, initVector); err != nil {
                return nil, err
        }

        cipherBlock, err := aes.NewCipher(spider_key)
        if err != nil {
                return nil, err
        }
        cipherTextFB := cipher.NewCFBEncrypter(cipherBlock, initVector)
        cipherTextFB.XORKeyStream(encryptData[aes.BlockSize:], []byte(base64Encoding))

        return encryptData, nil
}

