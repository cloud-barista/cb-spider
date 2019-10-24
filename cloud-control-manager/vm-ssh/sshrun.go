// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// to connect a remote server and execute a command on that remote server.
//
// by powerkim@powerkim.co.kr, 2019.03.
package sshrun

import (
	"io"
	"fmt"
	"github.com/bramvdbogaerde/go-scp"
	"golang.org/x/crypto/ssh"
	"os"
	"strings"
)

//====================================================================
type SSHInfo struct {
        UserName	string  // ex) "root"
        PrivateKey      []byte  // ex)   []byte(`-----BEGIN RSA PRIVATE KEY-----
				//		MIIEoQIBAAKCAQEArVNOLwMIp5VmZ4VPZotcoCHdEzimKalAsz+ccLfvAA1Y2ELH
				// 		...`)
        ServerPort	string  // ex) "node12:22"
}
//====================================================================

func Connect(sshInfo SSHInfo) (scp.Client, error) {
        clientConfig, _ := privateKey(sshInfo.UserName, sshInfo.PrivateKey, ssh.InsecureIgnoreHostKey())
        client := scp.NewClient(sshInfo.ServerPort, &clientConfig)
        err := client.Connect()
        return client, err
}

func privateKey(username string, privateKey []byte, keyCallBack ssh.HostKeyCallback) (ssh.ClientConfig, error) {

	signer, err := ssh.ParsePrivateKey(privateKey)

	if err != nil {
		return ssh.ClientConfig{}, err
	}

	return ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: keyCallBack,
	}, nil
}

func Close(client scp.Client){
	client.Close()	
}

func RunCommand(client scp.Client, cmd string) (string, error) {
	sess := client.Session
	// setup standard out and error
	// uses writer interface
	//sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr

	sshOut, err := sess.StdoutPipe()

	// run single command
	err = sess.Run(cmd)
	//err = sess.Start(cmd)

	return readBuffForString(sshOut), err
}

func readBuffForString(sshOut io.Reader) string {
	buf := make([]byte, 1000)
	n, err := sshOut.Read(buf) //this reads the ssh terminal
	waitingString := ""
	if err == nil {
/*
		for _, v := range buf[:n] {
			fmt.Printf("%c", v)
		}
*/
		waitingString = string(buf[:n])
	}
	for err == nil {
		// this loop will not end!!
		n, err = sshOut.Read(buf)
		waitingString += string(buf[:n])
/*		for _, v := range buf[:n] {
			fmt.Printf("%c", v)
		}
*/
		if err != nil {
			if err.Error() != "EOF" {
				fmt.Println(err)
			}
		}

	}
	return strings.Trim(waitingString, "\n")
}

func Copy(client scp.Client, sourcePath string, remotePath string) error {
        // Open a file
        file, _ := os.Open(sourcePath)

        // Close the file after it has been copied
        defer file.Close()

        err := client.CopyFile(file, remotePath, "0755")
        return err
}
