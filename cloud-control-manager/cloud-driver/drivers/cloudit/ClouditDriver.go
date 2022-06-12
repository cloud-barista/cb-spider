package cloudit

import (
    "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
    cicon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/connect"
    cirs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/resources"
    idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
    icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
)

type ClouditDriver struct{}

func (ClouditDriver) GetDriverVersion() string {
    return "CLOUDIT DRIVER Version 1.0"
}

func (ClouditDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
    var drvCapabilityInfo idrv.DriverCapabilityInfo

    drvCapabilityInfo.ImageHandler = false
    drvCapabilityInfo.VPCHandler = false
    drvCapabilityInfo.SecurityHandler = false
    drvCapabilityInfo.KeyPairHandler = false
    drvCapabilityInfo.VNicHandler = false
    drvCapabilityInfo.PublicIPHandler = false
    drvCapabilityInfo.VMHandler = true
    drvCapabilityInfo.NLBHandler = true

    drvCapabilityInfo.SINGLE_VPC = true

    return drvCapabilityInfo
}

func (driver *ClouditDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
    // 1. get info of credential and region for Test A Cloud from connectionInfo.
    // 2. create a client object(or service  object) of Test A Cloud with credential info.
    // 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
    // 4. return CloudConnection Interface of TDA_CloudConnection.

    // Initialize Logger
    cirs.InitLog()

    Client, err := getServiceClient(connectionInfo)
    if err != nil {
        return nil, err
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
        ClouditVersion: "v5.0",
        TenantID:       connInfo.CredentialInfo.TenantId,
    }
    return &restClient, nil
}
