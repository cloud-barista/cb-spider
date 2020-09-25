// Mock Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Mock Driver.
//
// by CB-Spider Team, 2020.09.

package mock

import (
	"C"
	"github.com/sirupsen/logrus"
        cblog "github.com/cloud-barista/cb-log"

	mkcon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/mock/connect"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
)

type MockDriver struct{}
var cblogger *logrus.Logger

func init() {
        // cblog is a global variable.
        cblogger = cblog.GetLogger("CB-SPIDER")
}

func (MockDriver) GetDriverVersion() string {
	return "MOCK DRIVER Version 1.0"
}

func (MockDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	drvCapabilityInfo.ImageHandler = true
	drvCapabilityInfo.VPCHandler = true
	drvCapabilityInfo.SecurityHandler = true
	drvCapabilityInfo.KeyPairHandler = true
	drvCapabilityInfo.VNicHandler = false
	drvCapabilityInfo.PublicIPHandler = false
	drvCapabilityInfo.VMHandler = true
	drvCapabilityInfo.VMSpecHandler = true

	return drvCapabilityInfo
}

func (driver *MockDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// <standard flow>
        // 1. get info of credential and region for Test A Cloud from connectionInfo.
        // 2. create a client object(or service  object) of XXX Cloud with credential info.
        // 3. create CloudConnection Instance of "connect/XXX_CloudConnection".
        // 4. return CloudConnection Interface of XXX_CloudConnection.

	// ex)
        // MockName = "mock01"
	iConn := mkcon.MockConnection{
		MockName:      connectionInfo.CredentialInfo.MockName,
	}
	return &iConn, nil
}

var CloudDriver MockDriver
