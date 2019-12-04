// Proof of Concepts for the Cloud-Barista Multi-Cloud Project.
//      * Cloud-Barista: https://github.com/cloud-barista
//
// scp & ssh test for serverhandler
//
// by CB-Spider Team, 2019.03.
package main

import (
	"fmt"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/vm-ssh"
)

func main() {

	// server connection info
	userName := "root"

        privateKey  := []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEoQIBAAKCAQEArVNOLwMIp5VmZ4VPZotcoCHdEzimKalAsz+ccLfvAA1Y2ELH
VwihRvkrqukUlkC7B3ASSCtgxIt5ZqfAKy9JvlT+Po/XHfaIpu9KM/XsZSdsF2jS
xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
-----END RSA PRIVATE KEY-----`)

	server := "node12"
	port := ":22"
	serverPort := server + port

        sshInfo := sshrun.SSHInfo{
                UserName: userName,
                PrivateKey: privateKey,
                ServerPort: serverPort,
        }

        // file info to copy
        sourceFile := "/root/go/src/farmoni/farmoni_agent/farmoni_agent"
        targetFile := "/tmp/farmoni_agent"

        // copy agent into the server.
        if err := sshrun.SSHCopy(sshInfo, sourceFile, targetFile); err !=nil {
                fmt.Println("Error while copying file ", err)
        }
}
