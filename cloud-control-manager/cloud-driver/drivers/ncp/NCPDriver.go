// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
//
// by ETRI, 2020.12.
// by ETRI, 2022.10. updated

package ncp

import (
	// "github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	ires "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	ncloud "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	vas "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vautoscaling"
	vlb "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vloadbalancer"
	vnks "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vnks"
	vpc "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vpc"
	vserver "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"

	// ncpcon "github.com/cloud-barista/ncp/ncp/connect"	// For local testing
	ncpcon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncp/connect"
	ncprs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncp/resources"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP Handler")
}

type NcpVpcDriver struct {
}

func (NcpVpcDriver) GetDriverVersion() string {
	return "TEST NCP VPC DRIVER Version 1.0"
}

func (NcpVpcDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	drvCapabilityInfo.ZoneBasedControl = true

	drvCapabilityInfo.RegionZoneHandler = true
	drvCapabilityInfo.PriceInfoHandler = true
	drvCapabilityInfo.ImageHandler = true
	drvCapabilityInfo.VMSpecHandler = true

	drvCapabilityInfo.VPCHandler = true
	drvCapabilityInfo.SecurityHandler = true
	drvCapabilityInfo.KeyPairHandler = true
	drvCapabilityInfo.VMHandler = true
	drvCapabilityInfo.DiskHandler = true
	drvCapabilityInfo.MyImageHandler = true
	drvCapabilityInfo.NLBHandler = true
	drvCapabilityInfo.ClusterHandler = true

	drvCapabilityInfo.TagHandler = false
	drvCapabilityInfo.TagSupportResourceType = []ires.RSType{}

	drvCapabilityInfo.VPC_CIDR = true

	return drvCapabilityInfo
}

func getVmClient(connectionInfo idrv.ConnectionInfo) (*vserver.APIClient, error) {

	// NOTE 주의!!
	apiKeys := ncloud.APIKey{
		AccessKey: connectionInfo.CredentialInfo.ClientId,
		SecretKey: connectionInfo.CredentialInfo.ClientSecret,
	}

	// Create NCP service client
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

func getVnksClient(connectionInfo idrv.ConnectionInfo) (*vnks.APIClient, error) {
	apiKeys := ncloud.APIKey{
		AccessKey: connectionInfo.CredentialInfo.ClientId,
		SecretKey: connectionInfo.CredentialInfo.ClientSecret,
	}

	// Create NCP VNKS service client
	client := vnks.NewAPIClient(vnks.NewConfiguration(connectionInfo.RegionInfo.Region, &apiKeys))

	return client, nil
}

func getVasClient(connectionInfo idrv.ConnectionInfo) (*vas.APIClient, error) {
	apiKeys := ncloud.APIKey{
		AccessKey: connectionInfo.CredentialInfo.ClientId,
		SecretKey: connectionInfo.CredentialInfo.ClientSecret,
	}

	// Create NCP VPC Load Balancer service client
	client := vas.NewAPIClient(vas.NewConfiguration(&apiKeys))

	return client, nil
}

func (driver *NcpVpcDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	// Initialize Logger
	ncprs.InitLog()

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

	vnksClient, err := getVnksClient(connectionInfo)
	if err != nil {
		return nil, err
	}

	vasClient, err := getVasClient(connectionInfo)
	if err != nil {
		return nil, err
	}

	iConn := ncpcon.NcpVpcCloudConnection{
		CredentialInfo: connectionInfo.CredentialInfo,
		RegionInfo:     connectionInfo.RegionInfo,
		VmClient:       vmClient,
		VpcClient:      vpcClient,
		VlbClient:      vlbClient,
		VnksClient:     vnksClient,
		VasClient:      vasClient,
	}

	return &iConn, nil
}
