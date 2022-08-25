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
	"github.com/gophercloud/gophercloud/openstack/identity/v3/services"

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
	drvCapabilityInfo.DiskHandler = true
	drvCapabilityInfo.MyImageHandler = true

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

	iConn, err := clientCreator(connectionInfo)
	if err != nil {
		return nil, err
	}

	return iConn, nil
}

func getIdentityClient(connInfo idrv.ConnectionInfo) (*gophercloud.ServiceClient, error) {
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

	client, err := openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}

	return client, err
}

func clientCreator(connInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	identityClient, err := getIdentityClient(connInfo)
	if err != nil {
		return nil, err
	}
	pager, err := services.List(identityClient, services.ListOpts{}).AllPages()
	if err != nil {
		return nil, err
	}
	list, err := services.ExtractServices(pager)
	if err != nil {
		return nil, err
	}
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

	iConn := oscon.OpenStackCloudConnection{
		CredentialInfo: connInfo.CredentialInfo,
		Region:         connInfo.RegionInfo,
	}
	for _, service := range list {
		err = insertClient(&iConn, provider, connInfo, service.Type)
		if err != nil {
			return nil, err
		}
	}
	return &iConn, nil
}

func insertClient(openstackCon *oscon.OpenStackCloudConnection, provider *gophercloud.ProviderClient, connInfo idrv.ConnectionInfo, serviceType string) error {
	switch serviceType {
	case "image":
		client, err := openstack.NewImageServiceV2(provider, gophercloud.EndpointOpts{
			Region: connInfo.RegionInfo.Region,
		})
		if err == nil {
			openstackCon.ImageClient = client
		}
	case "load-balancer":
		client, err := openstack.NewLoadBalancerV2(provider, gophercloud.EndpointOpts{
			Name:   "octavia",
			Region: connInfo.RegionInfo.Region,
		})
		if err == nil {
			openstackCon.NLBClient = client
		}
	case "volumev2":
		client, err := openstack.NewBlockStorageV2(provider, gophercloud.EndpointOpts{
			Region: connInfo.RegionInfo.Region,
		})
		if err == nil {
			openstackCon.Volume2Client = client
		}
	case "volumev3":
		client, err := openstack.NewBlockStorageV3(provider, gophercloud.EndpointOpts{
			Region: connInfo.RegionInfo.Region,
		})
		if err == nil {
			openstackCon.Volume3Client = client
		}
	case "network":
		client, err := openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{
			Name:   "neutron",
			Region: connInfo.RegionInfo.Region,
		})
		if err == nil {
			openstackCon.NetworkClient = client
		}
	case "compute":
		client, err := openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
			Region: connInfo.RegionInfo.Region,
		})
		if err == nil {
			openstackCon.ComputeClient = client
		}
	}
	return nil
}
