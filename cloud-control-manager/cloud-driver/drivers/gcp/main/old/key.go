package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/gob"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"golang.org/x/crypto/ssh"
)

func main() {
	reader := rand.Reader
	bitSize := 2048
	fmt.Println("reader : ", reader)
	key, err := rsa.GenerateKey(reader, bitSize)
	checkError(err)

	publicKey, err := generatePublicKey(&key.PublicKey)
	if err != nil {
		log.Fatal(err)
	}
	pubstr := string(publicKey)
	byteUser := []byte("cscservice")
	fmt.Println("byteUser :", byteUser)
	pubstr = pubstr + "cscservice"
	publicKey = append(publicKey, byteUser...)
	fmt.Println("public key :", string(publicKey))

	fmt.Println("public key :", []byte(pubstr))
	err = writeKeyToFile([]byte(pubstr))
	// //("private.key", key)
	// savePEMKey("private.pem", key)

	//saveGobKey("public.key", publicKey)
	//savePublicPEMKey("public.pem", key.PublicKey)
	// username := "cscservice"
	// cmdStr := `ssh-keygen -t rsa -f ./gce-vm-key -q -N "" -C ` + username
	// fmt.Println(cmdStr)
	// path, _ := os.Getwd()
	// cmd := exec.Command(path, cmdStr)
	// err = cmd.Run()
	// log.Fatal(err)

}
func generatePublicKey(privatekey *rsa.PublicKey) ([]byte, error) {
	publicRsaKey, err := ssh.NewPublicKey(privatekey)
	if err != nil {
		return nil, err
	}

	pubKeyBytes := ssh.MarshalAuthorizedKey(publicRsaKey)

	log.Println("Public key 생성")
	//fmt.Println(pubKeyBytes)
	return pubKeyBytes, nil
}
func saveGobKey(fileName string, key interface{}) {
	outFile, err := os.Create(fileName)
	checkError(err)
	defer outFile.Close()

	encoder := gob.NewEncoder(outFile)
	err = encoder.Encode(key)
	checkError(err)
}

func savePEMKey(fileName string, key *rsa.PrivateKey) {
	outFile, err := os.Create(fileName)
	checkError(err)
	defer outFile.Close()

	var privateKey = &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	err = pem.Encode(outFile, privateKey)
	checkError(err)
}

func savePublicPEMKey(fileName string, pubkey rsa.PublicKey) {
	asn1Bytes, err := asn1.Marshal(pubkey)
	checkError(err)

	var pemkey = &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: asn1Bytes,
	}

	pemfile, err := os.Create(fileName)
	checkError(err)
	defer pemfile.Close()

	err = pem.Encode(pemfile, pemkey)
	checkError(err)
}
func writeKeyToFile(keyBytes []byte) error {
	saveFileTo, _ := os.Getwd()

	err := ioutil.WriteFile(saveFileTo+"/gcp-key", keyBytes, 0600)
	if err != nil {
		return err
	}

	log.Printf("Key 저장위치: %s", saveFileTo)
	return nil
}

func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}
