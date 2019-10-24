// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// scp & ssh test for serverhandler
//
// by powerkim@powerkim.co.kr, 2019.03.
package main

import (
	"fmt"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/vm-ssh"
)

func main() {

	// server connection info
	userName := "root"
        keyPath  := "/root/.ssh/id_rsa" 
	server := "node12"
	port := ":22"
	serverPort := server + port

	// command for ssh run
        cmd := "/bin/hostname"

	sshKeypathInfo := sshrun.SSHKeyPathInfo{
		UserName: userName,
		KeyPath: keyPath,
		ServerPort: serverPort,
	}	
	var result string
	var err	error
        if result, err = sshrun.SSHRunByKeyPath(sshKeypathInfo, cmd); err != nil {
                fmt.Println("Error while running cmd: " + cmd, err)
        }

	fmt.Println(result)
}
