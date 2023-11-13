// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCPVPC Cloud Driver PoC
//
// by ETRI, 2020.12.
// by ETRI, 2022.10. updated

package ncpvpc

import (
	// "github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	cblog "github.com/cloud-barista/cb-log"	
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"

	ncloud "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	vserver "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"
	vpc "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vpc"
	vlb "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vloadbalancer"

	// ncpvpccon "github.com/cloud-barista/ncpvpc/ncpvpc/connect"	// For local testing
	ncpvpccon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncpvpc/connect"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCPVPC Handler")
}

type NcpVpcDriver struct {
}

func (NcpVpcDriver) GetDriverVersion() string {
	return "TEST NCP VPC DRIVER Version 1.0"
}

func (NcpVpcDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
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
	drvCapabilityInfo.NLBHandler = true

	return drvCapabilityInfo
}

func getVmClient(connectionInfo idrv.ConnectionInfo) (*vserver.APIClient, error) {

	// NOTE 주의!!
	apiKeys := ncloud.APIKey{
		AccessKey: connectionInfo.CredentialInfo.ClientId,
		SecretKey: connectionInfo.CredentialInfo.ClientSecret,
	}

	// NOTE for just test
	// cblogger.Info(apiKeys.AccessKey)
	// cblogger.Info(apiKeys.SecretKey)

	// Create NCPVPC service client
	client := vserver.NewAPIClient(vserver.NewConfiguration(&apiKeys))

	return client, nil
}

func getVpcClient(connectionInfo idrv.ConnectionInfo) (*vpc.APIClient, error) {
	apiKeys := ncloud.APIKey{
		AccessKey: connectionInfo.CredentialInfo.ClientId,
		SecretKey: connectionInfo.CredentialInfo.ClientSecret,
	}

	// Create NCP VPC service client
	client := vpc.NewAPIClient(vpc.NewConfiguration(&apiKeys))

	return client, nil
}

func getVlbClient(connectionInfo idrv.ConnectionInfo) (*vlb.APIClient, error) {
	apiKeys := ncloud.APIKey{
		AccessKey: connectionInfo.CredentialInfo.ClientId,
		SecretKey: connectionInfo.CredentialInfo.ClientSecret,
	}

	// Create NCP VPC Load Balancer service client
	client := vlb.NewAPIClient(vlb.NewConfiguration(&apiKeys))

	return client, nil
}

func (driver *NcpVpcDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	vmClient, err := getVmClient(connectionInfo)
	if err != nil {
		return nil, err
	}

	vpcClient, err := getVpcClient(connectionInfo)
	if err != nil {
		return nil, err
	}

	vlbClient, err := getVlbClient(connectionInfo)
	if err != nil {
		return nil, err
	}

	iConn := ncpvpccon.NcpVpcCloudConnection{
		CredentialInfo: connectionInfo.CredentialInfo,
		RegionInfo:     connectionInfo.RegionInfo,
		VmClient:       vmClient,
		VpcClient:      vpcClient,
		VlbClient:      vlbClient,
	}

	return &iConn, nil
}
