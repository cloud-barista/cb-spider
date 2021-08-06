// common package of CB-Spider's Cloud Drivers
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2021.08.

package common

import (
        "crypto/rand"
        "crypto/rsa"
        "crypto/x509"
        "encoding/pem"
        "io/ioutil"

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

// save a key to a file
func SaveKey(keyBytes []byte, targetFile string) error {
        err := ioutil.WriteFile(targetFile, keyBytes, 0600)
        if err != nil {
                return err
        }

        return nil
}
