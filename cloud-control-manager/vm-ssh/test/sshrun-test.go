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
zv3TCSvod2f09Bx7ebowLVRzyJe4UG+0OuM10Sk9dXRXL+viizyyPp1Ie2+FN32i
KVTG9jVd21kWUYxT7eKuqH78Jt5Ezmsqs4ArND5qM3B2BWQ9GiyOcOl6NfyA4+RH
wv8eYRJkkjv5q7R675U+EWLe7ktpmboOgl/I5hV1Oj/SQ3F90RqUcLrRz9XTsRKl
nKY2KG/2Q3ZYabf9TpZ/DeHNLus5n4STzFmukQIBIwKCAQEAqF+Nx0TGlCq7P/3Y
GnjAYQr0BAslEoco6KQxkhHDmaaQ0hT8KKlMNlEjGw5Og1TS8UhMRhuCkwsleapF
pksxsZRksc2PJGvVNHNsp4EuyKnz+XvFeJ7NAZheKtoD5dKGk4GrJLhwebf04GyD
XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
-----END RSA PRIVATE KEY-----`)


	server := "node12"
	port := ":22"
	serverPort := server + port

	// command for ssh run
        cmd := "/bin/hostname"


	sshInfo := sshrun.SSHInfo{
		UserName: userName,
		PrivateKey: privateKey,
		ServerPort: serverPort,
	}	
	var result string
	var err	error
        if result, err = sshrun.SSHRun(sshInfo, cmd); err != nil {
                fmt.Println("Error while running cmd: " + cmd, err)
        }

	fmt.Println(result)
}
