// Mini Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Mini Driver.
//
// by CB-Spider Team, 2021.11.

package mini

import (

	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"

	minicon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mini/connect"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
)

type MiniDriver struct{}

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

func (MiniDriver) GetDriverVersion() string {
	return "MINI DRIVER Version 1.0"
}

func (MiniDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	drvCapabilityInfo.ImageHandler = true
	drvCapabilityInfo.VMSpecHandler = true

	drvCapabilityInfo.VPCHandler = false
	drvCapabilityInfo.SecurityHandler = false
	drvCapabilityInfo.KeyPairHandler = false
	drvCapabilityInfo.VNicHandler = false
	drvCapabilityInfo.PublicIPHandler = false
	drvCapabilityInfo.VMHandler = false

	return drvCapabilityInfo
}

func (driver *MiniDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// <standard flow>
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of XXX Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/XXX_CloudConnection".
	// 4. return CloudConnection Interface of XXX_CloudConnection.

	// ex)
	iConn := minicon.MiniConnection{
		IdentityEndpoint: connectionInfo.CredentialInfo.IdentityEndpoint,
		AuthToken: connectionInfo.CredentialInfo.AuthToken,
	}

	return &iConn, nil
}

