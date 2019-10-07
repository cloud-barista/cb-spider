package cloudit

import (
	cblog "github.com/cloud-barista/cb-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	cicon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/connect"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	"github.com/sirupsen/logrus"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type ClouditDriver struct{}

func (ClouditDriver) GetDriverVersion() string {
	return "CLOUDIT DRIVER Version 1.0"
}

func (ClouditDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	drvCapabilityInfo.ImageHandler = false
	drvCapabilityInfo.VNetworkHandler = false
	drvCapabilityInfo.SecurityHandler = false
	drvCapabilityInfo.KeyPairHandler = false
	drvCapabilityInfo.VNicHandler = false
	drvCapabilityInfo.PublicIPHandler = false
	drvCapabilityInfo.VMHandler = true

	return drvCapabilityInfo
}

func (driver *ClouditDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	Client, err := getServiceClient(connectionInfo)
	if err != nil {
		cblogger.Error(err)
	}

	iConn := cicon.ClouditCloudConnection{
		CredentialInfo: connectionInfo.CredentialInfo,
		Client:         *Client,
	}

	return &iConn, nil
}

func getServiceClient(connInfo idrv.ConnectionInfo) (*client.RestClient, error) {
	restClient := client.RestClient{
		IdentityBase:   connInfo.CredentialInfo.IdentityEndpoint,
		ClouditVersion: "v4.0",
		TenantID:       connInfo.CredentialInfo.TenantId,
	}
	return &restClient, nil
}

var TestDriver ClouditDriver
