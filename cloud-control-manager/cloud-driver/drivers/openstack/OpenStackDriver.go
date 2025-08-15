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
	"crypto/tls"
	"fmt"
	"net/http"

	oscon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack/connect"
	osrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	ires "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
)

type OpenStackDriver struct{}

func (OpenStackDriver) GetDriverVersion() string {
	return "OPENSTACK DRIVER Version 1.0"
}

func (OpenStackDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
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
	drvCapabilityInfo.ClusterHandler = false
	drvCapabilityInfo.FileSystemHandler = true

	drvCapabilityInfo.TagHandler = true
	// ires.KEY, ires.DISK, ires.MYIMAGE, ires.CLUSTER: not supported
	drvCapabilityInfo.TagSupportResourceType = []ires.RSType{ires.VPC, ires.SUBNET, ires.SG, ires.VM, ires.NLB}

	drvCapabilityInfo.VPC_CIDR = false

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

func getIdentityClient(provider *gophercloud.ProviderClient, connInfo idrv.ConnectionInfo) (*gophercloud.ServiceClient, error) {
	client, err := openstack.NewIdentityV3(provider, gophercloud.EndpointOpts{
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}

	return client, err
}

func clientCreator(connInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	authOpts := gophercloud.AuthOptions{
		IdentityEndpoint: connInfo.CredentialInfo.IdentityEndpoint,
		Username:         connInfo.CredentialInfo.Username,
		Password:         connInfo.CredentialInfo.Password,
		DomainName:       connInfo.CredentialInfo.DomainName,
		TenantID:         connInfo.CredentialInfo.ProjectID,
	}

	config := &tls.Config{InsecureSkipVerify: true}
	httpClient := &http.Client{
		Transport: &http.Transport{TLSClientConfig: config},
	}

	provider, err := openstack.NewClient(authOpts.IdentityEndpoint)
	if err != nil {
		return nil, err
	}

	provider.HTTPClient = *httpClient

	err = openstack.Authenticate(provider, authOpts)
	if err != nil {
		return nil, err
	}

	identityClient, err := getIdentityClient(provider, connInfo)
	if err != nil {
		return nil, err
	}

	iConn := oscon.OpenStackCloudConnection{
		CredentialInfo: connInfo.CredentialInfo,
		Region:         connInfo.RegionInfo,
		IdentityClient: identityClient,
	}

	iConn.ImageClient, err = openstack.NewImageServiceV2(provider, gophercloud.EndpointOpts{
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}

	iConn.NLBClient, err = openstack.NewLoadBalancerV2(provider, gophercloud.EndpointOpts{
		Name:   "octavia",
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}

	iConn.Volume3Client, err = openstack.NewBlockStorageV3(provider, gophercloud.EndpointOpts{
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		iConn.Volume2Client, err = openstack.NewBlockStorageV2(provider, gophercloud.EndpointOpts{
			Region: connInfo.RegionInfo.Region,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize both volume v3 and v2 clients: %v", err)
		}
	}

	iConn.NetworkClient, err = openstack.NewNetworkV2(provider, gophercloud.EndpointOpts{
		Name:   "neutron",
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}

	iConn.ComputeClient, err = openstack.NewComputeV2(provider, gophercloud.EndpointOpts{
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}

	iConn.SharedFileSystemClient, err = openstack.NewSharedFileSystemV2(provider, gophercloud.EndpointOpts{
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}

	return &iConn, nil
}
