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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"sync"
	"io"
	"io/ioutil"
	"crypto/md5"

	"golang.org/x/crypto/ssh"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"errors"
	"regexp"
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


// Lock to store and read private key
var rwMutex sync.RWMutex

// ex) 
//	privateKey, publicKey, err := GenKeyPair()
//
//	srcList[0] = credentialInfo.IdentityEndpoint
//	srcList[1] = credentialInfo.AuthToken
//	srcList[2] = credentialInfo.TenantId
// 	strHash, err := GenHash(srcList)
//
//      AddKey("CLOUDIT", strHash, keyPairReqInfo.IId.NameId, privateKey)
func AddKey(providerName string, hashString string, keyPairNameId string, privateKey string) error {

	rwMutex.Lock()
	defer rwMutex.Unlock()

	err := insertInfo(providerName, hashString, keyPairNameId, privateKey)
        if err != nil {
                 return err
        }
	return nil	
}

// return: []KeyValue{Key:KeyPairNameId, Value:PrivateKey}
func ListKey(providerName string, hashString string) ([]*irs.KeyValue, error) {

	rwMutex.Lock()
	defer rwMutex.Unlock()

	keyValueList, err := listInfo(providerName, hashString)
        if err != nil {
                return nil, err
        }

        return keyValueList, nil
}

// return: KeyValue{Key:KeyPairNameId, Value:PrivateKey}
func GetKey(providerName string, hashString string, keyPairNameId string) (*irs.KeyValue, error) {

	rwMutex.Lock()
	defer rwMutex.Unlock()

	keyValue, err := getInfo(providerName, hashString, keyPairNameId)
        if err != nil {
                return nil, err
        }
        return keyValue, nil
}

func DelKey(providerName string, hashString string, keyPairNameId string) error {

	rwMutex.Lock()
	defer rwMutex.Unlock()

	_, err := deleteInfo(providerName, hashString, keyPairNameId)
        if err != nil {
                return err
        }
        return nil
}

func GenHash(sourceList []string) (string, error) {
	var keyString  string
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

// save a key to a file
func SaveKey(keyBytes []byte, targetFile string) error {
	err := ioutil.WriteFile(targetFile, keyBytes, 0600)
	if err != nil {
		return err
	}

	return nil
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
