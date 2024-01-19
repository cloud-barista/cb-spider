// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// KT Cloud Driver PoC
//
// by ETRI, 2021.05.
// Updated by ETRI, 2023.10.

package ktcloud

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	// "github.com/davecgh/go-spew/spew"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"

	ktsdk "github.com/cloud-barista/ktcloud-sdk-go"
	//ktcloudcon "github.com/cloud-barista/ktcloud/ktcloud/connect"
	ktcloudcon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ktcloud/connect"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("KT Cloud VMHandler")
}

type KtCloudDriver struct {
}

func (KtCloudDriver) GetDriverVersion() string {
	return "TEST KT Cloud DRIVER Version 1.0"
}

func (KtCloudDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
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

const (
	KOR_Seoul_M2_ZoneID string = "d7d0177e-6cda-404a-a46f-a5b356d2874e"
)

func getVMClient(connectionInfo idrv.ConnectionInfo) (*ktsdk.KtCloudClient, error) {
	// cblogger.Info("### connectionInfo.RegionInfo.Zone : " + connectionInfo.RegionInfo.Zone)
	
	// $$$ Caution!!
	var apiurl string
	if strings.EqualFold(connectionInfo.RegionInfo.Zone, KOR_Seoul_M2_ZoneID) { // When Zone is "KOR-Seoul M2"
	apiurl = "https://api.ucloudbiz.olleh.com/server/v2/client/api"
	} else {
	apiurl = "https://api.ucloudbiz.olleh.com/server/v1/client/api"
	}

	if len(apiurl) == 0 {
		cblogger.Error("KT Cloud API URL Not Found!!")
		os.Exit(1)
	}

	apikey := connectionInfo.CredentialInfo.ClientId
	if len(apikey) == 0 {
		cblogger.Error("KT Cloud API Key Not Found!!")
		os.Exit(1)
	}

	secretkey := connectionInfo.CredentialInfo.ClientSecret
	if len(secretkey) == 0 {
		cblogger.Error("KT Cloud Secret Key Not Found!!")
		os.Exit(1)
	}

	// NOTE for just test
	// cblogger.Info(apiurl)
	// cblogger.Info(apikey)
	// cblogger.Info(secretkey)

	// Always validate any SSL certificates in the chain
	insecureskipverify := false
	client := ktsdk.KtCloudClient{}.New(apiurl, apikey, secretkey, insecureskipverify)

	return client, nil
}

func getNLBClient(connectionInfo idrv.ConnectionInfo) (*ktsdk.KtCloudClient, error) {
	// cblogger.Info("### connectionInfo.RegionInfo.Zone : " + connectionInfo.RegionInfo.Zone)
	
	// $$$ Caution!!
	var apiurl string
	if strings.EqualFold(connectionInfo.RegionInfo.Zone, KOR_Seoul_M2_ZoneID) { // When Zone is "KOR-Seoul M2"
	apiurl = "https://api.ucloudbiz.olleh.com/loadbalancer/v2/client/api"
	} else {
	apiurl = "https://api.ucloudbiz.olleh.com/loadbalancer/v1/client/api"
	}

	if len(apiurl) == 0 {
		cblogger.Error("KT Cloud API URL Not Found!!")
		os.Exit(1)
	}

	apikey := connectionInfo.CredentialInfo.ClientId
	if len(apikey) == 0 {
		cblogger.Error("KT Cloud API Key Not Found!!")
		os.Exit(1)
	}

	secretkey := connectionInfo.CredentialInfo.ClientSecret
	if len(secretkey) == 0 {
		cblogger.Error("KT Cloud Secret Key Not Found!!")
		os.Exit(1)
	}

	// NOTE for just test
	// cblogger.Info(apiurl)
	// cblogger.Info(apikey)
	// cblogger.Info(secretkey)

	// Always validate any SSL certificates in the chain
	insecureskipverify := false
	client := ktsdk.KtCloudClient{}.New(apiurl, apikey, secretkey, insecureskipverify)

	return client, nil
}

func (driver *KtCloudDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	// NOTE Just for Test
	// spew.Dump(connectionInfo.CredentialInfo.ClientId) // 전달 받은 idrv.ConnectionInfo check
	// spew.Dump(connectionInfo.RegionInfo)              // 전달 받은 idrv.ConnectionInfo check

	vmClient, err := getVMClient(connectionInfo)
	if err != nil {
		return nil, err
	}
	// spew.Dump(vmClient)       

	nlbClient, err := getNLBClient(connectionInfo)
	if err != nil {
		return nil, err
	}

	iConn := ktcloudcon.KtCloudConnection{
		CredentialInfo: connectionInfo.CredentialInfo,
		RegionInfo:     connectionInfo.RegionInfo,		
		Client:         vmClient,
		NLBClient:      nlbClient,
	}
	return &iConn, nil
}
