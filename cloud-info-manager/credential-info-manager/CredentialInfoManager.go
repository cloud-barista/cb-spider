// Cloud Credential Info. Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2019.09.

package credentialinfomanager

import (
	"fmt"
	"strings"
	"github.com/cloud-barista/cb-store/config"
	icbs "github.com/cloud-barista/cb-store/interfaces"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"

	"github.com/sirupsen/logrus"

	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
)

var cblog *logrus.Logger

func init() {
	cblog = config.Cblogger
}

//====================================================================
type CredentialInfo struct {
	CredentialName   string          // ex) "credential01"
	ProviderName     string          // ex) "AWS"
	KeyValueInfoList []icbs.KeyValue // ex) { {ClientId, XXX},
	//	 {ClientSecret, XXX},
	//	 {TenantId, XXX},
	//	 {SubscriptionId, XXX} }
}

//====================================================================

func RegisterCredentialInfo(crdInfo CredentialInfo) (*CredentialInfo, error) {
	return RegisterCredential(crdInfo.CredentialName, crdInfo.ProviderName, crdInfo.KeyValueInfoList)
}

// 1. check params
// 2. insert them into cb-store
func RegisterCredential(credentialName string, providerName string, keyValueInfoList []icbs.KeyValue) (*CredentialInfo, error) {
	cblog.Info("call RegisterCredential()")

	cblog.Debug("check params")
	err := checkParams(credentialName, providerName, keyValueInfoList)
	if err != nil {
		return nil, err

	}

	// trim user inputs
        credentialName = strings.TrimSpace(credentialName)
	providerName = strings.ToUpper(strings.TrimSpace(providerName))

	cblog.Debug("insert metainfo into store")

        err = encryptKeyValueList(keyValueInfoList)
        if err != nil {
                return &CredentialInfo{}, err
	}

	err = insertInfo(credentialName, providerName, keyValueInfoList)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	crdInfo := &CredentialInfo{credentialName, providerName, keyValueInfoList}
	return crdInfo, nil
}

func ListCredential() ([]*CredentialInfo, error) {
	cblog.Info("call ListCredential()")

	credentialInfoList, err := listInfo()
	if err != nil {
		return nil, err
	}

	return credentialInfoList, nil
}

// 1. check params
// 2. get CredentialInfo from cb-store
func GetCredential(credentialName string) (*CredentialInfo, error) {
	cblog.Info("call GetCredential()")

	if credentialName == "" {
		return nil, fmt.Errorf("CredentialName is empty!")
	}

	crdInfo, err := getInfo(credentialName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	return crdInfo, err
}

// 1. check params
// 2. get CredentialInfo from cb-store
// 3. decrypt CrednetialInfo
func GetCredentialDecrypt(credentialName string) (*CredentialInfo, error) {
        cblog.Info("call GetCredential()")

        if credentialName == "" {
                return nil, fmt.Errorf("CredentialName is empty!")
        }

        crdInfo, err := getInfo(credentialName)
        if err != nil {
                cblog.Error(err)
                return nil, err
        }

	err = decryptKeyValueList(crdInfo.KeyValueInfoList)	
	if err != nil {
		return &CredentialInfo{}, err
	}
	return crdInfo, nil
}

// @todo env by powerkim, 2020.06.01.
var spider_key = []byte("cloud-barista-cb-spider-cloud-ba") // 32 bytes

func encryptKeyValueList(keyValueInfoList []icbs.KeyValue) error {

        for i, kv := range keyValueInfoList {
                encb, err := encrypt(spider_key, []byte(kv.Value))
                kv.Value = string(encb)
                if err != nil {
                        return err
                }
                keyValueInfoList[i] = kv
        }
        return  nil
}

func decryptKeyValueList(keyValueInfoList []icbs.KeyValue) error {

	for i, kv := range keyValueInfoList {
		decb, err := decrypt(spider_key, []byte(kv.Value))
		kv.Value = string(decb)
		if err != nil {
			return err
		}
		keyValueInfoList[i] = kv
	}
	return nil	
}

// encription with spider key
func encrypt(spider_key, contents []byte) ([]byte, error) {

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

// decryption with spider key
func decrypt(spider_key, contents []byte) ([]byte, error) {

	if len(contents) < aes.BlockSize {
		err := fmt.Errorf("decryption: " + "contents too short")
		cblog.Error(err)
		return nil, err
	}

	cipherBlock, err := aes.NewCipher(spider_key)
	if err != nil {
		return nil, err
	}

	initVector := contents[:aes.BlockSize]
	contents = contents[aes.BlockSize:]
	cipherTextFB := cipher.NewCFBDecrypter(cipherBlock, initVector)
	cipherTextFB.XORKeyStream(contents, contents)
	decryptData, err := base64.StdEncoding.DecodeString(string(contents))

	if err != nil {
		return nil, err
	}
	return decryptData, nil
}

func UnRegisterCredential(credentialName string) (bool, error) {
	cblog.Info("call UnRegisterCredential()")

	if credentialName == "" {
		return false, fmt.Errorf("CredentialName is empty!")
	}

	result, err := deleteInfo(credentialName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	return result, nil
}

//----------------

func checkParams(credentialName string, providerName string, keyValueInfoList []icbs.KeyValue) error {
	if credentialName == "" {
		return fmt.Errorf("CredentialName is empty!")
	}
	if providerName == "" {
		return fmt.Errorf("ProviderName is empty!")
	}
	if keyValueInfoList == nil {
                return fmt.Errorf("KeyValue List is nil!")
        }

	// get Provider's Meta Info
        cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo(providerName)
        if err != nil {
                cblog.Error(err)
                return err
        }

        // validate the KeyValueList of Credential Input
        err = cim.ValidateKeyValueList(keyValueInfoList, cloudOSMetaInfo.Credential)
        if err != nil {
                cblog.Error(err)
                return err
        }

	return nil
}
