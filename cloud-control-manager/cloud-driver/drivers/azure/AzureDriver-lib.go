// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by hyokyung.kim@innogrid.co.kr, 2019.07.

package main

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2018-06-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2017-05-10/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	azcon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/azure/connect"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	"time"
)

type AzureDriver struct{}

func (AzureDriver) GetDriverVersion() string {
	return "AZURE DRIVER Version 1.0"
}

func (AzureDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	drvCapabilityInfo.ImageHandler = true
	drvCapabilityInfo.VNetworkHandler = true
	drvCapabilityInfo.SecurityHandler = true
	drvCapabilityInfo.KeyPairHandler = true
	drvCapabilityInfo.VNicHandler = true
	drvCapabilityInfo.PublicIPHandler = true
	drvCapabilityInfo.VMHandler = true

	return drvCapabilityInfo
}

func (driver *AzureDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	// Credentail에 등록된 ResourceGroup 존재 여부 체크 및 생성
	err := checkResourceGroup(connectionInfo.CredentialInfo, connectionInfo.RegionInfo)
	if err != nil {
		return nil, err
	}

	Ctx, VMClient, err := getVMClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, imageClient, err := getImageClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, publicIPClient, err := getPublicIPClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, sgClient, err := getSecurityGroupClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, vNicClient, err := getVNicClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, SubnetClient, err := getSubnetClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, VNetClient, err := getVNetworkClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, IPConfigClient, err := getIPConfigClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, VMImageClient, err := getVMImageClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, DiskClient, err := getDiskClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}

	iConn := azcon.AzureCloudConnection{
		CredentialInfo:      connectionInfo.CredentialInfo,
		Region:              connectionInfo.RegionInfo,
		Ctx:                 Ctx,
		VMClient:            VMClient,
		ImageClient:         imageClient,
		PublicIPClient:      publicIPClient,
		SecurityGroupClient: sgClient,
		VNetClient:          VNetClient,
		VNicClient:          vNicClient,
		IPConfigClient:      IPConfigClient,
		SubnetClient:        SubnetClient,
		VMImageClient:       VMImageClient,
		DiskClient:          DiskClient,
	}
	return &iConn, nil
}

func checkResourceGroup(credential idrv.CredentialInfo, region idrv.RegionInfo) error {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil
	}

	resourceClient := resources.NewGroupsClient(credential.SubscriptionId)
	resourceClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), 600*time.Second)

	rg, err := resourceClient.Get(ctx, region.ResourceGroup)

	// 해당 리소스 그룹이 없을 경우 생성
	if rg.ID == nil {
		rg, err = resourceClient.CreateOrUpdate(ctx, region.ResourceGroup,
			resources.Group{
				Name:     to.StringPtr(region.ResourceGroup),
				Location: to.StringPtr(region.Region),
			})
		if err != nil {
			return err
		}
	}
	return nil
}

func getVMClient(credential idrv.CredentialInfo) (context.Context, *compute.VirtualMachinesClient, error) {
	/*auth.NewClientCredentialsConfig()
	  authorizer, err := auth.NewAuthorizerFromFile(azure.PublicCloud.ResourceManagerEndpoint)
	  if err != nil {
	      return nil, nil, err
	  }*/
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	vmClient := compute.NewVirtualMachinesClient(credential.SubscriptionId)
	vmClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), 600*time.Second)

	return ctx, &vmClient, nil
}

func getImageClient(credential idrv.CredentialInfo) (context.Context, *compute.ImagesClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	imageClient := compute.NewImagesClient(credential.SubscriptionId)
	imageClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), 600*time.Second)

	return ctx, &imageClient, nil
}

func getPublicIPClient(credential idrv.CredentialInfo) (context.Context, *network.PublicIPAddressesClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	publicIPClient := network.NewPublicIPAddressesClient(credential.SubscriptionId)
	publicIPClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), 600*time.Second)

	return ctx, &publicIPClient, nil
}

func getSecurityGroupClient(credential idrv.CredentialInfo) (context.Context, *network.SecurityGroupsClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	sgClient := network.NewSecurityGroupsClient(credential.SubscriptionId)
	sgClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), 600*time.Second)

	return ctx, &sgClient, nil
}

func getVNetworkClient(credential idrv.CredentialInfo) (context.Context, *network.VirtualNetworksClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	vNetClient := network.NewVirtualNetworksClient(credential.SubscriptionId)
	vNetClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), 600*time.Second)

	return ctx, &vNetClient, nil
}

func getVNicClient(credential idrv.CredentialInfo) (context.Context, *network.InterfacesClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	vNicClient := network.NewInterfacesClient(credential.SubscriptionId)
	vNicClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), 600*time.Second)

	return ctx, &vNicClient, nil
}

func getIPConfigClient(credential idrv.CredentialInfo) (context.Context, *network.InterfaceIPConfigurationsClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	ipConfigClient := network.NewInterfaceIPConfigurationsClient(credential.SubscriptionId)
	ipConfigClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), 600*time.Second)

	return ctx, &ipConfigClient, nil
}

func getSubnetClient(credential idrv.CredentialInfo) (context.Context, *network.SubnetsClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	subnetClient := network.NewSubnetsClient(credential.SubscriptionId)
	subnetClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), 600*time.Second)

	return ctx, &subnetClient, nil
}

func getVMImageClient(credential idrv.CredentialInfo) (context.Context, *compute.VirtualMachineImagesClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	vmImageClient := compute.NewVirtualMachineImagesClient(credential.SubscriptionId)
	vmImageClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), 600*time.Second)

	return ctx, &vmImageClient, nil
}

func getDiskClient(credential idrv.CredentialInfo) (context.Context, *compute.DisksClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	diskClient := compute.NewDisksClient(credential.SubscriptionId)
	diskClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), 600*time.Second)

	return ctx, &diskClient, nil
}

var CloudDriver AzureDriver
