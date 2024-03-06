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

package main

import (
	// "github.com/davecgh/go-spew/spew"
	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	ostack "github.com/cloud-barista/nhncloud-sdk-go/openstack"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"

	// nhncon "github.com/cloud-barista/nhncloud/nhncloud/connect"
	nhncon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/nhncloud/connect"

	// nhnrs "github.com/cloud-barista/nhncloud/nhncloud/resources"
	nhnrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/nhncloud/resources"
)

type NhnCloudDriver struct{}

func (NhnCloudDriver) GetDriverVersion() string {
	return "NHNCLOUD DRIVER Version 1.0"
}

func (NhnCloudDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
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
	drvCapabilityInfo.ClusterHandler = true
	drvCapabilityInfo.MyImageHandler = true
	drvCapabilityInfo.DiskHandler = true
	drvCapabilityInfo.RegionZoneHandler = true
	drvCapabilityInfo.PriceInfoHandler = false

	drvCapabilityInfo.SINGLE_VPC = false

	return drvCapabilityInfo
}

func (driver *NhnCloudDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	// Initialize Logger
	nhnrs.InitLog()

	VMClient, err := getVMClient(connectionInfo)
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

	VolumeClient, err := getVolumeClient(connectionInfo)
	if err != nil {
		return nil, err
	}

	ClusterClient, err := getClusterClient(connectionInfo)
	if err != nil {
		return nil, err
	}

	iConn := nhncon.NhnCloudConnection{
		CredentialInfo: connectionInfo.CredentialInfo, // Note) Need in RegionZoneHandler
		RegionInfo: 	connectionInfo.RegionInfo,
		VMClient: 		VMClient,
		ImageClient: 	ImageClient,
		NetworkClient: 	NetworkClient,
		VolumeClient: 	VolumeClient,
		ClusterClient: 	ClusterClient,
	}
	return &iConn, nil
}

func getVMClient(connInfo idrv.ConnectionInfo) (*nhnsdk.ServiceClient, error) {
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

	client, err := ostack.NewComputeV2(providerClient, nhnsdk.EndpointOpts{
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}

	return client, err
}

func getImageClient(connInfo idrv.ConnectionInfo) (*nhnsdk.ServiceClient, error) {
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

	client, err := ostack.NewImageServiceV2(providerClient, nhnsdk.EndpointOpts{
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}

	return client, err
}

func getNetworkClient(connInfo idrv.ConnectionInfo) (*nhnsdk.ServiceClient, error) {
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

	client, err := ostack.NewNetworkV2(providerClient, nhnsdk.EndpointOpts{
		Name:   "neutron",
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}

	return client, err
}

func getVolumeClient(connInfo idrv.ConnectionInfo) (*nhnsdk.ServiceClient, error) {
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

	client, err := ostack.NewBlockStorageV2(providerClient, nhnsdk.EndpointOpts{
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}
	return client, err
}

func getClusterClient(connInfo idrv.ConnectionInfo) (*nhnsdk.ServiceClient, error) {
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

	client, err := ostack.NewContainerInfraV1(providerClient, nhnsdk.EndpointOpts{
		Region: connInfo.RegionInfo.Region,
	})
	if err != nil {
		return nil, err
	}
	return client, err
}

var CloudDriver NhnCloudDriver
