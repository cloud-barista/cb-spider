// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2022.08.
// by ETRI, 2024.04.

package ktcloudvpc

import (
	"github.com/sirupsen/logrus"
	// "github.com/davecgh/go-spew/spew"

	ktvpcsdk 	"github.com/cloud-barista/ktcloudvpc-sdk-go"
	ostack 		"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack"

	cblog 		"github.com/cloud-barista/cb-log"
	idrv 		"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon 		"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"

	// ktvpccon "github.com/cloud-barista/ktcloudvpc/ktcloudvpc/connect"
	ktvpccon 	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ktcloudvpc/connect" //To be built in the container1

	// ktvpcrs "github.com/cloud-barista/ktcloudvpc/ktcloudvpc/resources"
	ktvpcrs 	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ktcloudvpc/resources" //To be built in the container
)

type KTCloudVpcDriver struct{}

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

func (KTCloudVpcDriver) GetDriverVersion() string {
	return "KTCLOUD VPC DRIVER Version 1.0"
}

func (KTCloudVpcDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	drvCapabilityInfo.ImageHandler = true
	drvCapabilityInfo.VPCHandler = true
	drvCapabilityInfo.SecurityHandler = true
	drvCapabilityInfo.KeyPairHandler = true
	drvCapabilityInfo.VNicHandler = false
	drvCapabilityInfo.PublicIPHandler = false
	drvCapabilityInfo.VMHandler = true
	drvCapabilityInfo.VMSpecHandler = true
	drvCapabilityInfo.NLBHandler = true
	drvCapabilityInfo.ClusterHandler = false
	drvCapabilityInfo.MyImageHandler = true
	drvCapabilityInfo.DiskHandler = true
	drvCapabilityInfo.RegionZoneHandler = true
	drvCapabilityInfo.PriceInfoHandler = false

	drvCapabilityInfo.SINGLE_VPC = true

	return drvCapabilityInfo
}

func (driver *KTCloudVpcDriver) ConnectCloud(connInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// Initialize Logger
	ktvpcrs.InitLog()

	authOpts := ktvpcsdk.AuthOptions{
		IdentityEndpoint: connInfo.CredentialInfo.IdentityEndpoint,
		Username:         connInfo.CredentialInfo.Username,
		Password:         connInfo.CredentialInfo.Password,
		DomainName:       connInfo.CredentialInfo.DomainName,
		TenantID:         connInfo.CredentialInfo.ProjectID, // Caution : ProjectID to TenantID on SDK
	}
	providerClient, err := ostack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, err
	}

	VMClient, err := getVMClient(providerClient, connInfo)
	if err != nil {
		return nil, err
	}

	ImageClient, err := getImageClient(providerClient, connInfo)
	if err != nil {
		return nil, err
	}

	NetworkClient, err := getNetworkClient(providerClient, connInfo)
	if err != nil {
		return nil, err
	}

	VolumeClient, err := getVolumeClient(providerClient, connInfo)
	if err != nil {
		return nil, err
	}

	NLBClient, err := getNLBClient(providerClient, connInfo)
	if err != nil {
		return nil, err
	}	

	iConn := ktvpccon.KTCloudVpcConnection{
		RegionInfo: 	connInfo.RegionInfo, 
		VMClient: 		VMClient, 
		ImageClient: 	ImageClient, 
		NetworkClient: 	NetworkClient, 
		VolumeClient: 	VolumeClient, 
		NLBClient: 		NLBClient,
	}
	return &iConn, nil
}

// Caution!! : Region info in real client info from KT Cloud VPC(D1) : Not 'DX-M1' but 'regionOne'.
func getVMClient(providerClient *ktvpcsdk.ProviderClient, connInfo idrv.ConnectionInfo) (*ktvpcsdk.ServiceClient, error) {
	client, err := ostack.NewComputeV2(providerClient, ktvpcsdk.EndpointOpts{
		Type:   "compute",
        Region: connInfo.RegionInfo.Zone,
	})
	if err != nil {
		return nil, err
	}
	// cblogger.Info("\n# Result serviceClient : ")
	// spew.Dump(client)
	// cblogger.Info("\n\n")

	return client, err
}

func getImageClient(providerClient *ktvpcsdk.ProviderClient, connInfo idrv.ConnectionInfo) (*ktvpcsdk.ServiceClient, error) {
	client, err := ostack.NewImageServiceV2(providerClient, ktvpcsdk.EndpointOpts{
		Region: connInfo.RegionInfo.Zone,
	})
	if err != nil {
		return nil, err
	}
	// cblogger.Info("\n# Result serviceClient : ")
	// spew.Dump(client)
	// cblogger.Info("\n\n")

	return client, err
}

func getNetworkClient(providerClient *ktvpcsdk.ProviderClient, connInfo idrv.ConnectionInfo) (*ktvpcsdk.ServiceClient, error) {
	client, err := ostack.NewNetworkV2(providerClient, ktvpcsdk.EndpointOpts{
		Name:   "neutron",
		Region: connInfo.RegionInfo.Zone,
	})
	if err != nil {
		return nil, err
	}
	// cblogger.Info("\n# Result serviceClient : ")
	// spew.Dump(client)
	// cblogger.Info("\n\n")

	return client, err
}

func getVolumeClient(providerClient *ktvpcsdk.ProviderClient, connInfo idrv.ConnectionInfo) (*ktvpcsdk.ServiceClient, error) {
	client, err := ostack.NewBlockStorageV2(providerClient, ktvpcsdk.EndpointOpts{
		Region: connInfo.RegionInfo.Zone,
	})
	if err != nil {
		return nil, err
	}

	return client, err
}

func getNLBClient(providerClient *ktvpcsdk.ProviderClient, connInfo idrv.ConnectionInfo) (*ktvpcsdk.ServiceClient, error) {
	client, err := ostack.NewLoadBalancerV1(providerClient, ktvpcsdk.EndpointOpts{
		Region: connInfo.RegionInfo.Zone,
	})
	if err != nil {
		return nil, err
	}

	return client, err
}
