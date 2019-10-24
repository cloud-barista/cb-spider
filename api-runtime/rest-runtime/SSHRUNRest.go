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
        UserName        string  // ex) "root"
        PrivateKey      []string  // ex)   ["-----BEGIN RSA PRIVATE KEY-----",
                                //          "MIIEoQIBAAKCAQEArVNOLwMIp5VmZ4VPZotcoCHdEzimKalAsz+ccLfvAA1Y2ELH",
                                //          "..."]
        ServerPort      string  // ex) "node12:22"
        Command         string  // ex) "hostname"
}

//================ SSH RUN
func sshRun(c echo.Context) error {
	cblog.Info("call sshRun()")

	req := &SSHRUNReqInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	strPrivateKey := strings.Join(req.PrivateKey[:], "\n")
	

	sshInfo := sshrun.SSHInfo {
		UserName : req.UserName,
		PrivateKey : []byte(strPrivateKey),
		ServerPort : req.ServerPort,
	}
	sshCli, err := sshrun.Connect(sshInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var result string
        if result, err = sshrun.RunCommand(sshCli, req.Command); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Error while running cmd: " + req.Command + "]" + err.Error())
        }

        sshrun.Close(sshCli)

	return c.JSON(http.StatusOK, result)
}
