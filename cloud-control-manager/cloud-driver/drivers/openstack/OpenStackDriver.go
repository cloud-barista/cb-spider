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
	cblog "github.com/cloud-barista/cb-log"
	oscon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack/connect"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/sirupsen/logrus"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type OpenStackDriver struct{}

func (OpenStackDriver) GetDriverVersion() string {
	return "OPENSTACK DRIVER Version 1.0"
}

func (OpenStackDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	drvCapabilityInfo.ImageHandler = true
	drvCapabilityInfo.VNetworkHandler = true
	drvCapabilityInfo.SecurityHandler = true
	drvCapabilityInfo.KeyPairHandler = true
	drvCapabilityInfo.VNicHandler = false
	drvCapabilityInfo.PublicIPHandler = true
	drvCapabilityInfo.VMHandler = true

	return drvCapabilityInfo
}

/* org
func (OpenStackDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	// sample code, do not user like this^^
	var iConn icon.CloudConnection
	iConn = oscon.OpenStackCloudConnection{}

	return iConn, nil // return type: (icon.CloudConnection, error)
}
*/

// modifiled by powerkim, 2019.07.29.
func (driver *OpenStackDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	// sample code, do not user like this^^

	Client, err := getServiceClient(connectionInfo)
	if err != nil {
		cblogger.Error(err)
	}
	ImageClient, err := getImageClient(connectionInfo)
	if err != nil {
		cblogger.Error(err)
	}
	NetworkClient, err := getNetworkClient(connectionInfo)
	if err != nil {
		cblogger.Error(err)
	}

	iConn := oscon.OpenStackCloudConnection{Client, ImageClient, NetworkClient}

	return &iConn, nil // return type: (icon.CloudConnection, error)
}

/*func (driver *OpenStackDriver) ConnectNetworkCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {

	NetworkClient, err := getNetworkClient(connectionInfo)
	if err != nil {
		cblogger.Error(err)
	}

	//var iConn icon.CloudConnection
	iConn := oscon.OpenStackCloudConnection{nil, NetworkClient}

	return &iConn, nil // return type: (icon.CloudConnection, error)
}*/

//--------- temporary  by powerkim, 2019.07.29.
/*type Config struct {
	Openstack struct {
		DomainName       string `yaml:"domain_name"`
		IdentityEndpoint string `yaml:"identity_endpoint"`
		Password         string `yaml:"password"`
		ProjectID        string `yaml:"project_id"`
		Username         string `yaml:"username"`
		Region           string `yaml:"region"`
		VMName           string `yaml:"vm_name"`
		ImageId          string `yaml:"image_id"`
		FlavorId         string `yaml:"flavor_id"`
		NetworkId        string `yaml:"network_id"`
		SecurityGroups   string `yaml:"security_groups"`
		KeypairName      string `yaml:"keypair_name"`

		ServerId string `yaml:"server_id"`
	} `yaml:"openstack"`
}*/

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

	client, err := openstack.NewClient(connInfo.CredentialInfo.IdentityEndpoint)

	authOpts := gophercloud.AuthOptions{
		//IdentityEndpoint: connInfo.CredentialInfo.IdentityEndpoint,
		Username:   connInfo.CredentialInfo.Username,
		Password:   connInfo.CredentialInfo.Password,
		DomainName: connInfo.CredentialInfo.DomainName,
		TenantID:   connInfo.CredentialInfo.ProjectID,
	}
	err = openstack.AuthenticateV3(client, authOpts)

	c, err := openstack.NewImageServiceV2(client, gophercloud.EndpointOpts{
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

var TestDriver OpenStackDriver
