// Cloud Credential Info. Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2023.07.
// by CB-Spider Team, 2019.09.

package credentialinfomanager

import (
	"fmt"
	"strings"

	cblogger "github.com/cloud-barista/cb-log"
	icdrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"
	infostore "github.com/cloud-barista/cb-spider/info-store"

	"github.com/sirupsen/logrus"

	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"io"
)

// ====================================================================
const KEY_COLUMN_NAME = "credential_name"

type CredentialInfo struct {
	CredentialName   string           `gorm:"primaryKey"` // ex) "credential01"
	ProviderName     string           // ex) "AWS"
	KeyValueInfoList infostore.KVList `gorm:"type:blob"` // stored with json format, ex) { {ClientId, XXX}, {ClientSecret, XXX}, ...}
}

//====================================================================

var cblog *logrus.Logger

func init() {
	cblog = cblogger.GetLogger("CLOUD-BARISTA")

	// db, err := infostore.Open()
	// if err != nil {
	// 	panic("failed to connect database")
	// }
	// db.AutoMigrate(&CredentialInfo{})
	// infostore.Close(db)
}

// 1. check params
// 2. insert them into info-store
func RegisterCredentialInfo(crdInfo CredentialInfo) (*CredentialInfo, error) {
	cblog.Info("call RegisterCredentialInfo()")

	cblog.Debug("check params")
	err := checkParams(crdInfo.CredentialName, crdInfo.ProviderName, crdInfo.KeyValueInfoList)
	if err != nil {
		return nil, err

	}

	// trim user inputs
	crdInfo.CredentialName = strings.TrimSpace(crdInfo.CredentialName)
	crdInfo.ProviderName = strings.ToUpper(strings.TrimSpace(crdInfo.ProviderName))

	cblog.Debug("insert metainfo into store")

	err = encryptKeyValueList(crdInfo.KeyValueInfoList)
	if err != nil {
		return &CredentialInfo{}, err
	}

	err = infostore.Insert(&crdInfo)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// Hide credential data for security
	kvList := []icdrs.KeyValue{}
	for _, kv := range crdInfo.KeyValueInfoList {
		kv.Value = "Hidden for security."
		kvList = append(kvList, kv)
	}

	return &crdInfo, nil
}

func RegisterCredential(credentialName string, providerName string, keyValueInfoList []icdrs.KeyValue) (*CredentialInfo, error) {
	cblog.Info("call RegisterCredential()")

	return RegisterCredentialInfo(CredentialInfo{credentialName, providerName, keyValueInfoList})
}

func ListCredential() ([]*CredentialInfo, error) {
	cblog.Info("call ListCredential()")

	var credentialInfoList []*CredentialInfo
	err := infostore.List(&credentialInfoList)
	if err != nil {
		return nil, err
	}

	// Hide credential data for security
	returnInfoList := []*CredentialInfo{}
	for _, info := range credentialInfoList {

		kvList := []icdrs.KeyValue{}
		for _, kv := range info.KeyValueInfoList {
			kv.Value = "Hidden for security."
			kvList = append(kvList, kv)
		}
		info.KeyValueInfoList = kvList

		returnInfoList = append(returnInfoList, info)
	}

	return returnInfoList, nil
}

// 1. check params
// 2. get CredentialInfo from info-store
func GetCredential(credentialName string) (*CredentialInfo, error) {
	cblog.Info("call GetCredential()")

	if credentialName == "" {
		return nil, fmt.Errorf("CredentialName is empty!")
	}

	var credentialInfo CredentialInfo
	err := infostore.Get(&credentialInfo, KEY_COLUMN_NAME, credentialName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	// Hide credential data for security
	kvList := []icdrs.KeyValue{}
	for _, kv := range credentialInfo.KeyValueInfoList {
		kv.Value = "Hidden for security."
		kvList = append(kvList, kv)
	}
	credentialInfo.KeyValueInfoList = kvList

	return &credentialInfo, err
}

// 1. check params
// 2. get CredentialInfo from info-store
// 3. decrypt CrednetialInfo
func GetCredentialDecrypt(credentialName string) (*CredentialInfo, error) {
	cblog.Info("call GetCredential()")

	if credentialName == "" {
		return nil, fmt.Errorf("CredentialName is empty!")
	}

	var credentialInfo CredentialInfo
	err := infostore.Get(&credentialInfo, KEY_COLUMN_NAME, credentialName)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	err = decryptKeyValueList(credentialInfo.KeyValueInfoList)
	if err != nil {
		return &CredentialInfo{}, err
	}
	return &credentialInfo, nil
}

func UnRegisterCredential(credentialName string) (bool, error) {
	cblog.Info("call UnRegisterCredential()")

	if credentialName == "" {
		return false, fmt.Errorf("CredentialName is empty!")
	}

	result, err := infostore.Delete(&CredentialInfo{}, KEY_COLUMN_NAME, credentialName)
	if err != nil {
		cblog.Error(err)
		return false, err
	}

	return result, nil
}

//----------------

func checkParams(credentialName string, providerName string, keyValueInfoList []icdrs.KeyValue) error {
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

// #######################################################################
// @todo get from Env file. by powerkim, 2020.06.01.
var SPIDER_KEY = []byte("cloud-barista-cb-spider-cloud-ba") // 32 bytes
//#######################################################################

func encryptKeyValueList(keyValueInfoList []icdrs.KeyValue) error {

	for i, kv := range keyValueInfoList {
		encString, err := Encrypt(SPIDER_KEY, []byte(kv.Value))
		if err != nil {
			return err
		}
		kv.Value = encString
		keyValueInfoList[i] = kv
	}
	return nil
}

func decryptKeyValueList(keyValueInfoList []icdrs.KeyValue) error {

	for i, kv := range keyValueInfoList {
		decString, err := Decrypt(SPIDER_KEY, []byte(kv.Value))
		if err != nil {
			return err
		}
		kv.Value = decString
		keyValueInfoList[i] = kv
	}
	return nil
}

// encription with spider key
func Encrypt(spider_key, contents []byte) (string, error) {

	block, err := aes.NewCipher(spider_key)
	if err != nil {
		return "", err
	}

	ciphertext := make([]byte, aes.BlockSize+len(contents))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", err
	}

	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(ciphertext[aes.BlockSize:], contents)

	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decryption with spider key
func Decrypt(spider_key, contents []byte) (string, error) {

	ciphertext, err := base64.StdEncoding.DecodeString(string(contents))
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(spider_key)
	if err != nil {
		return "", err
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	stream := cipher.NewCTR(block, iv)
	plaintext := make([]byte, len(ciphertext))
	stream.XORKeyStream(plaintext, ciphertext)

	return string(plaintext), nil
}
