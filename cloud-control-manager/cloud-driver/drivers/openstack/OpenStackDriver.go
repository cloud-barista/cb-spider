// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by hyokyung.kim@innogrid.co.kr, 2019.07.

package openstack

import (
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"

	oscon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack/connect"
	osrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
)

type OpenStackDriver struct{}

func (OpenStackDriver) GetDriverVersion() string {
	return "OPENSTACK DRIVER Version 1.0"
}

func (OpenStackDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
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

	return drvCapabilityInfo
}

// modifiled by powerkim, 2019.07.29.
func (driver *OpenStackDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	// Initialize Logger
	osrs.InitLog()

	Client, err := getServiceClient(connectionInfo)
	if err != nil {
		return nil, err
	}
	ImageClient, err := getImageClient(connectionInfo)
	if err != nil {
		return nil, err
	}
	NetworkClient, err := getNetworkClient(connectionInfo)
	if err != nil {
		return nil, err
	}
	VolumeClient, _ := getVolumeClient(connectionInfo)

	NLBClient, _ := getNLBClient(connectionInfo)

	iConn := oscon.OpenStackCloudConnection{CredentialInfo: connectionInfo.CredentialInfo, Region: connectionInfo.RegionInfo, Client: Client, ImageClient: ImageClient, NetworkClient: NetworkClient, VolumeClient: VolumeClient, NLBClient: NLBClient}

	return &iConn, nil
}

// moved by powerkim, 2019.07.29.
func getServiceClient(connInfo idrv.ConnectionInfo) (*gophercloud.ServiceClient, error) {

	authOpts := gophercloud.AuthOptions{
		IdentityEndpoint: connInfo.CredentialInfo.IdentityEndpoint,
		Username:         connInfo.CredentialInfo.Username,
		Password:         connInfo.CredentialInfo.Password,
		DomainName:       connInfo.CredentialInfo.DomainName,
		TenantID:         connInfo.CredentialInfo.ProjectID,
	}

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, err
	}

	client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}

	return client, err
}

func getImageClient(connInfo idrv.ConnectionInfo) (*gophercloud.ServiceClient, error) {

	authOpts := gophercloud.AuthOptions{
		IdentityEndpoint: connInfo.CredentialInfo.IdentityEndpoint,
		Username:         connInfo.CredentialInfo.Username,
		Password:         connInfo.CredentialInfo.Password,
		DomainName:       connInfo.CredentialInfo.DomainName,
		TenantID:         connInfo.CredentialInfo.ProjectID,
	}
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, err
	}

	c, err := openstack.NewImageServiceV2(provider, gophercloud.EndpointOpts{
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}

	return c, err
}

func getNetworkClient(connInfo idrv.ConnectionInfo) (*gophercloud.ServiceClient, error) {

	authOpts := gophercloud.AuthOptions{
		IdentityEndpoint: connInfo.CredentialInfo.IdentityEndpoint,
		Username:         connInfo.CredentialInfo.Username,
		Password:         connInfo.CredentialInfo.Password,
		DomainName:       connInfo.CredentialInfo.DomainName,
		TenantID:         connInfo.CredentialInfo.ProjectID,
	}

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, err
	}

	client, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{
		Name:   "neutron",
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}

	return client, err
}
func getVolumeClient(connInfo idrv.ConnectionInfo) (*gophercloud.ServiceClient, error) {
	authOpts := gophercloud.AuthOptions{
		IdentityEndpoint: connInfo.CredentialInfo.IdentityEndpoint,
		Username:         connInfo.CredentialInfo.Username,
		Password:         connInfo.CredentialInfo.Password,
		DomainName:       connInfo.CredentialInfo.DomainName,
		TenantID:         connInfo.CredentialInfo.ProjectID,
	}
	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, err
	}
	client, err := openstack.NewBlockStorageV2(provider, gophercloud.EndpointOpts{
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		client, err = openstack.NewBlockStorageV3(provider, gophercloud.EndpointOpts{
			Region: connInfo.RegionInfo.Region,
		})
		if err != nil {
			return nil, err
		}
	}
	return client, err
}

func getNLBClient(connInfo idrv.ConnectionInfo) (*gophercloud.ServiceClient, error) {

	authOpts := gophercloud.AuthOptions{
		IdentityEndpoint: connInfo.CredentialInfo.IdentityEndpoint,
		Username:         connInfo.CredentialInfo.Username,
		Password:         connInfo.CredentialInfo.Password,
		DomainName:       connInfo.CredentialInfo.DomainName,
		TenantID:         connInfo.CredentialInfo.ProjectID,
	}

	provider, err := openstack.AuthenticatedClient(authOpts)
	if err != nil {
		return nil, err
	}

	client, err := openstack.NewLoadBalancerV2(provider, gophercloud.EndpointOpts{
		Name:   "octavia",
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}

	return client, err
}
