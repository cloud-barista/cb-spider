// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, Innogrid, 2021.12.
// by ETRI Team, 2022.08.

package nhn

import (
	// "github.com/davecgh/go-spew/spew"

	"errors"
	"fmt"
	"strings"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	ostack "github.com/cloud-barista/nhncloud-sdk-go/openstack"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	ires "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	// nhncon "github.com/cloud-barista/nhncloud/nhncloud/connect"
	nhncon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/nhn/connect"

	// nhnrs "github.com/cloud-barista/nhncloud/nhncloud/resources"
	nhnrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/nhn/resources"
)

type NhnCloudDriver struct{}

func (NhnCloudDriver) GetDriverVersion() string {
	return "NHN DRIVER Version 1.0"
}

func (NhnCloudDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	drvCapabilityInfo.ZoneBasedControl = true

	drvCapabilityInfo.RegionZoneHandler = true
	drvCapabilityInfo.PriceInfoHandler = false
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

func (driver *NhnCloudDriver) ConnectCloud(connInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	// Initialize Logger
	nhnrs.InitLog()

	authOpts := nhnsdk.AuthOptions{
		IdentityEndpoint: connInfo.CredentialInfo.IdentityEndpoint,
		Username:         connInfo.CredentialInfo.Username,
		Password:         connInfo.CredentialInfo.Password,
		DomainName:       connInfo.CredentialInfo.DomainName,
		TenantID:         connInfo.CredentialInfo.TenantId, // Caution : TenantID spelling for SDK
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

	ClusterClient, err := getClusterClient(providerClient, connInfo)
	if err != nil {
		var endpointErr *nhnsdk.ErrEndpointNotFound
		if errors.As(err, &endpointErr) {
			// If a certain region(ex. JPN) do not provide a cluster service, there is no endpoint for that.
			// In this case it is not an error.
		} else {
			return nil, err
		}
	}

	FSClient, err := getFSClient(providerClient, connInfo)
	if err != nil {
		return nil, err
	}

	iConn := nhncon.NhnCloudConnection{
		CredentialInfo: connInfo.CredentialInfo, // Note) Need in RegionZoneHandler
		RegionInfo:     connInfo.RegionInfo,
		VMClient:       VMClient,
		ImageClient:    ImageClient,
		NetworkClient:  NetworkClient,
		VolumeClient:   VolumeClient,
		ClusterClient:  ClusterClient,
		FSClient:       FSClient,
	}
	return &iConn, nil
}

func getVMClient(providerClient *nhnsdk.ProviderClient, connInfo idrv.ConnectionInfo) (*nhnsdk.ServiceClient, error) {
	client, err := ostack.NewComputeV2(providerClient, nhnsdk.EndpointOpts{
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}

	return client, err
}

func getImageClient(providerClient *nhnsdk.ProviderClient, connInfo idrv.ConnectionInfo) (*nhnsdk.ServiceClient, error) {
	client, err := ostack.NewImageServiceV2(providerClient, nhnsdk.EndpointOpts{
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}

	return client, err
}

func getNetworkClient(providerClient *nhnsdk.ProviderClient, connInfo idrv.ConnectionInfo) (*nhnsdk.ServiceClient, error) {
	client, err := ostack.NewNetworkV2(providerClient, nhnsdk.EndpointOpts{
		Name:   "neutron",
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}

	return client, err
}

func getVolumeClient(providerClient *nhnsdk.ProviderClient, connInfo idrv.ConnectionInfo) (*nhnsdk.ServiceClient, error) {
	client, err := ostack.NewBlockStorageV2(providerClient, nhnsdk.EndpointOpts{
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}

	return client, err
}

func getClusterClient(providerClient *nhnsdk.ProviderClient, connInfo idrv.ConnectionInfo) (*nhnsdk.ServiceClient, error) {
	client, err := ostack.NewContainerInfraV1(providerClient, nhnsdk.EndpointOpts{
		Region: connInfo.RegionInfo.Region,
	})

	// If .Microversion is not set, NHN Cloud rejects the request that is
	// 'GET https://kr1-api-kubernetes-infrastructure/v1/clusters/{UUID}/nodegroups'
	client.Microversion = "latest"
	if err != nil {
		return nil, err
	}

	return client, err
}

func getFSClient(providerClient *nhnsdk.ProviderClient, connInfo idrv.ConnectionInfo) (*nhnsdk.ServiceClient, error) {
	region := connInfo.RegionInfo.Region

	endpoint := fmt.Sprintf("https://%s-api-nas-infrastructure.nhncloudservice.com/v1/", strings.ToLower(region))

	fsClient := &nhnsdk.ServiceClient{
		ProviderClient: providerClient,
		Endpoint:       endpoint,
		Type:           "sharev2",
	}

	return fsClient, nil
}
