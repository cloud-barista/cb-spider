package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/asn1"
	"encoding/gob"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"os/exec"
)

func main() {
	reader := rand.Reader
	bitSize := 2048
	fmt.Println("reader : ", reader)
	key, err := rsa.GenerateKey(reader, bitSize)
	checkError(err)
	fmt.Println("key : ", key)
	// publicKey := key.PublicKey

	// saveGobKey("private.key", key)
	// savePEMKey("private.pem", key)

	// saveGobKey("public.key", publicKey)
	// savePublicPEMKey("public.pem", publicKey)
	username := "cscservice"
	cmdStr := `ssh-keygen -t rsa -f ./gce-vm-key -q -N "" -C ` + username
	fmt.Println(cmdStr)
	cmd := exec.Command("./", cmdStr)
	err = cmd.Run()
	log.Fatal(err)

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

func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}
