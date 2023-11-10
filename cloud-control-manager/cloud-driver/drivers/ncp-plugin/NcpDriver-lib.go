// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2020.08.

package main

import (
	//cblog "github.com/cloud-barista/cb-log"

	idrv 	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon 	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"

	// "github.com/davecgh/go-spew/spew"
	// unused import "github.com/sirupsen/logrus"

	ncloud 	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	server 	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/server"
	lb 		"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/loadbalancer"

	// ncpcon "github.com/cloud-barista/ncp/ncp/connect"
	ncpcon 	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncp/connect" //To be built in the container
)

/*
var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP VMHandler")
}
*/

type NcpDriver struct {
}

func (NcpDriver) GetDriverVersion() string {
	return "TEST NCP DRIVER Version 1.0"
}

func (NcpDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	// NOTE Temporary Setting
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

// func getVMClient(credential idrv.CredentialInfo) (*server.APIClient, error) {
func getVMClient(connectionInfo idrv.ConnectionInfo) (*server.APIClient, error) {
	// NOTE 주의!!
	apiKeys := ncloud.APIKey{
		AccessKey: connectionInfo.CredentialInfo.ClientId,
		SecretKey: connectionInfo.CredentialInfo.ClientSecret,
	}
	// Create NCP service client
	client := server.NewAPIClient(server.NewConfiguration(&apiKeys))
	return client, nil
}

func getLbClient(connectionInfo idrv.ConnectionInfo) (*lb.APIClient, error) {
	apiKeys := ncloud.APIKey{
		AccessKey: connectionInfo.CredentialInfo.ClientId,
		SecretKey: connectionInfo.CredentialInfo.ClientSecret,
	}
	// Create NCP Classic Load Balancer service client
	client := lb.NewAPIClient(lb.NewConfiguration(&apiKeys))
	return client, nil
}

func (driver *NcpDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	vmClient, err := getVMClient(connectionInfo)
	if err != nil {
		return nil, err
	}

	lbClient, err := getLbClient(connectionInfo)
	if err != nil {
		return nil, err
	}
	
	iConn := ncpcon.NcpCloudConnection{
		CredentialInfo: connectionInfo.CredentialInfo,
		RegionInfo:     connectionInfo.RegionInfo,
		VmClient:       vmClient,
		LbClient:       lbClient,
	}

	return &iConn, nil
}

var CloudDriver NcpDriver
