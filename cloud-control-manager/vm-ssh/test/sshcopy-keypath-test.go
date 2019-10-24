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

        sshKeypathInfo := sshrun.SSHKeyPathInfo{
                UserName: userName,
                KeyPath: keyPath,
                ServerPort: serverPort,
        }

        // file info to copy
        sourceFile := "/root/go/src/farmoni/farmoni_agent/farmoni_agent"
        targetFile := "/tmp/farmoni_agent"

        // copy agent into the server.
        if err := sshrun.SSHCopyByKeyPath(sshKeypathInfo, sourceFile, targetFile); err !=nil {
                fmt.Println("Error while copying file ", err)
        }
}
