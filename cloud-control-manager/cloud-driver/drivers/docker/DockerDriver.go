// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Docker Driver.
//
// by CB-Spider Team, 2020.05.

package docker

import (
	"C"
	"github.com/sirupsen/logrus"
        cblog "github.com/cloud-barista/cb-log"
	"context"
        "github.com/docker/docker/client"

	dkcon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/docker/connect"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
)

type DockerDriver struct{}
var cblogger *logrus.Logger

func init() {
        // cblog is a global variable.
        cblogger = cblog.GetLogger("CB-SPIDER")
}


func (DockerDriver) GetDriverVersion() string {
	return "DOCKER DRIVER Version 1.0"
}

func (DockerDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	drvCapabilityInfo.ImageHandler = true
	drvCapabilityInfo.VPCHandler = false
	drvCapabilityInfo.SecurityHandler = false
	drvCapabilityInfo.KeyPairHandler = false
	drvCapabilityInfo.VNicHandler = false
	drvCapabilityInfo.PublicIPHandler = false
	drvCapabilityInfo.VMHandler = true
	drvCapabilityInfo.VMSpecHandler = false

	return drvCapabilityInfo
}

func (driver *DockerDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
        // 1. get info of credential and region for Test A Cloud from connectionInfo.
        // 2. create a client object(or service  object) of XXX Cloud with credential info.
        // 3. create CloudConnection Instance of "connect/XXX_CloudConnection".
        // 4. return CloudConnection Interface of XXX_CloudConnection.

	//thisContext, _ := context.WithTimeout(context.Background(), 600*time.Second)
	thisContext := context.Background()

	// ex)
        // IdentityEndpoint = "http://18.191.129.154:1004"
	// APIVersion = "v1.36"
	Host:= connectionInfo.CredentialInfo.Host
	APIVersion:= connectionInfo.CredentialInfo.APIVersion
        client, err := client.NewClient(Host, APIVersion, nil, nil)
        if err != nil {
		cblogger.Error(err)
                return nil, err
        }

	iConn := dkcon.DockerCloudConnection{
		ConnectionInfo:      connectionInfo,
		Context:             thisContext,
		Client:		     client,
	}
	return &iConn, nil
}

var CloudDriver DockerDriver
