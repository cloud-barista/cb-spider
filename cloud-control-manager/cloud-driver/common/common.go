// common package of CB-Spider's Cloud Drivers
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2021.12.
// by CB-Spider Team, 2021.08.

package common

import (
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"

	"errors"
	"regexp"

	cblogger "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	enc "github.com/cloud-barista/cb-spider/cloud-info-manager/credential-info-manager"
	infostore "github.com/cloud-barista/cb-spider/info-store"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

// generate a KeyPair with 4KB length
// returns: privateKeyBytes, publicKeyBytes, error
func GenKeyPair() ([]byte, []byte, error) {

	// (1) Generate a private Key
	keyLength := 4096
	privateKey, err := rsa.GenerateKey(rand.Reader, keyLength)
	if err != nil {
		return nil, nil, err
	}

	err = privateKey.Validate()
	if err != nil {
		return nil, nil, err
	}

	// for ASN.1 DER format
	DERKey := x509.MarshalPKCS1PrivateKey(privateKey)
	keyBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   DERKey,
	}

	// for PEM format
	privateKeyBytes := pem.EncodeToMemory(&keyBlock)

	// (2) Generate a public key
	publicKey, err := ssh.NewPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, nil, err
	}

	publicKeyBytes := ssh.MarshalAuthorizedKey(publicKey)

	return privateKeyBytes, publicKeyBytes, nil
}

// ex)
//	privateKey, publicKey, err := GenKeyPair()
//
//	srcList[0] = credentialInfo.IdentityEndpoint
//	srcList[1] = credentialInfo.AuthToken
//	srcList[2] = credentialInfo.TenantId
// 	strHash, err := GenHash(srcList)
//
//      AddKey("CLOUDIT", strHash, keyPairReqInfo.IId.NameId, privateKey)

type LocalKeyInfo struct {
	ProviderName string `gorm:"primaryKey"`
	HashString   string `gorm:"primaryKey"`
	NameId       string `gorm:"primaryKey"`
	PrivateKey   string
}

func (LocalKeyInfo) TableName() string {
	return "local_key_infos"
}

//====================================================================

var cblog *logrus.Logger

func init() {
	cblog = cblogger.GetLogger("CLOUD-BARISTA")
	db, err := infostore.Open()
	if err != nil {
		cblog.Error(err)
		return
	}
	db.AutoMigrate(&LocalKeyInfo{})
	infostore.Close(db)
}

func AddKey(providerName string, hashString string, keyPairNameId string, privateKey string) error {

	encPrivateKey, err := enc.Encrypt(enc.SPIDER_KEY, []byte(privateKey))
	if err != nil {
		return err
	}

	err = infostore.Insert(&LocalKeyInfo{ProviderName: providerName, HashString: hashString, NameId: keyPairNameId, PrivateKey: encPrivateKey})
	if err != nil {
		cblog.Error(err)
		return err
	}
	return nil
}

// return: []KeyValue{Key:KeyPairNameId, Value:PrivateKey}
func ListKey(providerName string, hashString string) ([]*irs.KeyValue, error) {

	var iidInfoList []*LocalKeyInfo
	err := infostore.ListByConditions(&iidInfoList, "provider_name", providerName, "hash_string", hashString)
	if err != nil {
		cblog.Error(err)
		return nil, err
	}

	var keyValueList []*irs.KeyValue
	for _, iidInfo := range iidInfoList {

		decPrivateKey, err := enc.Decrypt(enc.SPIDER_KEY, []byte(iidInfo.PrivateKey))
		if err != nil {
			return nil, err
		}

		keyValue := &irs.KeyValue{
			Key:   iidInfo.NameId,
			Value: decPrivateKey,
		}
		keyValueList = append(keyValueList, keyValue)
	}

	return keyValueList, nil
}

// return: KeyValue{Key:KeyPairNameId, Value:PrivateKey}
func GetKey(providerName string, hashString string, keyPairNameId string) (*irs.KeyValue, error) {

	var localKeyInfo LocalKeyInfo
	err := infostore.GetBy3Conditions(&localKeyInfo, "provider_name", providerName, "hash_string", hashString, "name_id", keyPairNameId)
	if err != nil {
		// cblog.Error(err) // Call GetKey during creation to check if the Key already exists. This situation is not an error.
		return nil, err
	}

	decPrivateKey, err := enc.Decrypt(enc.SPIDER_KEY, []byte(localKeyInfo.PrivateKey))
	if err != nil {
		return nil, err
	}

	keyValue := &irs.KeyValue{
		Key:   localKeyInfo.NameId,
		Value: decPrivateKey,
	}

	return keyValue, nil
}

func DelKey(providerName string, hashString string, keyPairNameId string) error {

	_, err := infostore.DeleteBy3Conditions(&LocalKeyInfo{}, "provider_name", providerName, "hash_string", hashString, "name_id", keyPairNameId)
	if err != nil {
		cblog.Error(err)
		return err
	}
	return nil
}

func GenHash(sourceList []string) (string, error) {
	var keyString string
	for _, str := range sourceList {
		keyString += str
	}
	hasher := md5.New()
	_, err := io.WriteString(hasher, keyString)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// ParseKey reads the given RSA private key and create a public one for it.
func MakePublicKeyFromPrivateKey(pem string) (string, error) {
	key, err := ssh.ParseRawPrivateKey([]byte(pem))
	if err != nil {
		return "", err
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return "", fmt.Errorf("%q is not a RSA key", pem)
	}
	pub, err := ssh.NewPublicKey(&rsaKey.PublicKey)
	if err != nil {
		return "", err
	}

	return string(bytes.TrimRight(ssh.MarshalAuthorizedKey(pub), "\n")), nil
}

//-------------------

func ValidateWindowsPassword(pw string) error {

	invalidMSG := `Password must be between 12 and 123 characters long and must have 3 of the following: 
			1 lower case character, 
			1 upper case character, 
			1 number,
			1 special character`

	if len(pw) < 12 || len(pw) > 123 {
		return errors.New(invalidMSG)
	}

	checkNum := 0
	matchCase, err := regexp.MatchString(".*[a-z]+", pw)
	if matchCase && err == nil {
		checkNum++
	}
	matchCase, _ = regexp.MatchString(".*[A-Z]+", pw)
	if matchCase && err == nil {
		checkNum++
	}
	matchCase, _ = regexp.MatchString(".*[0-9]+", pw)
	if matchCase && err == nil {
		checkNum++
	}
	matchCase, _ = regexp.MatchString(`[\{\}\[\]\/?.,;:|\)*~!^\-_+<>@\#$%&\\\=\(\'\"\n\r]+`, pw)
	if matchCase && err == nil {
		checkNum++
	}
	if checkNum >= 3 {
		return nil
	} else {
		return errors.New(invalidMSG)

	}
}
