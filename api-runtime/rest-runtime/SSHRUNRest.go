// Rest Runtime Server for VM's SSH and SCP of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by powerkim@etri.re.kr, 2019.10.

package main

import (
	"github.com/cloud-barista/cb-spider/cloud-control-manager/vm-ssh"

	"strings"
	// REST API (echo)
	"github.com/labstack/echo"
	"net/http"
)

type SSHRUNReqInfo struct {
	UserName   string   // ex) "root"
	PrivateKey []string // ex)   ["-----BEGIN RSA PRIVATE KEY-----",
	//          "MIIEoQIBAAKCAQEArVNOLwMIp5VmZ4VPZotcoCHdEzimKalAsz+ccLfvAA1Y2ELH",
	//          "..."]
	ServerPort string // ex) "node12:22"
	Command    string // ex) "hostname"
}

type SSHRUNKEYPATHReqInfo struct {
	UserName   string // ex) "root"
	KeyPath    string // ex) "/root/.ssh/keyfile.pem"
	ServerPort string // ex) "node12:22"
	Command    string // ex) "hostname"
}

type SSHCOPYKEYPATHReqInfo struct {
	UserName   string // ex) "root"
	KeyPath    string // ex) "/root/.ssh/keyfile.pem"
	ServerPort string // ex) "node12:22"
	SourceFile string // ex) "/root/go/src/github.com/cloud-barista/aaaaa"
	TargetFile string // ex) "/root/temp/src/aaaaa"
}

//================ SSH RUN
func sshRun(c echo.Context) error {
	cblog.Info("call sshRun()")

	req := &SSHRUNReqInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	strPrivateKey := strings.Join(req.PrivateKey[:], "\n")

	sshInfo := sshrun.SSHInfo{
		UserName:   req.UserName,
		PrivateKey: []byte(strPrivateKey),
		ServerPort: req.ServerPort,
	}
	var result string
	var err error
	if result, err = sshrun.SSHRun(sshInfo, req.Command); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Error while running cmd: "+req.Command+"]"+err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

func sshRunkeyPath(c echo.Context) error {
	cblog.Info("call sshRunkeyPath()")

	req := &SSHRUNKEYPATHReqInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	sshInfo := sshrun.SSHKeyPathInfo{
		UserName:   req.UserName,
		KeyPath:    req.KeyPath,
		ServerPort: req.ServerPort,
	}
	var result string
	var err error
	if result, err = sshrun.SSHRunByKeyPath(sshInfo, req.Command); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Error while running cmd: "+req.Command+"]"+err.Error())
	}

	return c.JSON(http.StatusOK, result)

}

func sshCopykeyPath(c echo.Context) error {
	cblog.Info("call sshCopykeyPath()")

	req := &SSHCOPYKEYPATHReqInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	sshInfo := sshrun.SSHKeyPathInfo{
		UserName:   req.UserName,
		KeyPath:    req.KeyPath,
		ServerPort: req.ServerPort,
	}

	var err error
	if err = sshrun.SSHCopyByKeyPath(sshInfo, req.SourceFile, req.TargetFile); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Error: "+err.Error())
	}

	return c.JSON(http.StatusOK, "Copy succeeded!!")

}
