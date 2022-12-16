// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by hyokyung.kim@innogrid.co.kr, 2019.07.

package azure

import (
	"context"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2022-03-01/containerservice"
	"time"

	"github.com/Azure/azure-sdk-for-go/profiles/2020-09-01/monitor/mgmt/insights"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-03-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-10-01/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"

	azcon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/azure/connect"
	azrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/azure/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
)

type AzureDriver struct{}

func (AzureDriver) GetDriverVersion() string {
	return "AZURE DRIVER Version 1.0"
}

const (
	cspTimeout time.Duration = 6000
)

func (AzureDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
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
	drvCapabilityInfo.ClusterHandler = true

	return drvCapabilityInfo
}

func (driver *AzureDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	// Initialize Logger
	azrs.InitLog()

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
	Ctx, sgRuleClient, err := getSecurityGroupRuleClient(connectionInfo.CredentialInfo)
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
	Ctx, VmSpecClient, err := getVmSpecClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, sshKeyClient, err := getSshKeyClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, nlbClient, err := getNLBClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, nlbBackendAddressPoolsClient, err := getNLBBackendAddressPoolsClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, nlbLoadBalancingRulesClient, err := getLoadBalancingRulesClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, metricClient, err := getMetricClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, managedClustersClient, err := getManagedClustersClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, agentPoolsClient, err := getAgentPoolsClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, virtualMachineScaleSetsClient, err := getVirtualMachineScaleSetsClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, virtualMachineScaleSetVMsClient, err := getVirtualMachineScaleSetVMsClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, virtualMachineRunCommandClient, err := getVirtualMachineRunCommandClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	iConn := azcon.AzureCloudConnection{
		CredentialInfo:                  connectionInfo.CredentialInfo,
		Region:                          connectionInfo.RegionInfo,
		Ctx:                             Ctx,
		VMClient:                        VMClient,
		ImageClient:                     imageClient,
		PublicIPClient:                  publicIPClient,
		SecurityGroupClient:             sgClient,
		SecurityGroupRuleClient:         sgRuleClient,
		VNetClient:                      VNetClient,
		VNicClient:                      vNicClient,
		IPConfigClient:                  IPConfigClient,
		SubnetClient:                    SubnetClient,
		VMImageClient:                   VMImageClient,
		DiskClient:                      DiskClient,
		VmSpecClient:                    VmSpecClient,
		SshKeyClient:                    sshKeyClient,
		NLBClient:                       nlbClient,
		NLBBackendAddressPoolsClient:    nlbBackendAddressPoolsClient,
		NLBLoadBalancingRulesClient:     nlbLoadBalancingRulesClient,
		MetricClient:                    metricClient,
		ManagedClustersClient:           managedClustersClient,
		AgentPoolsClient:                agentPoolsClient,
		VirtualMachineScaleSetsClient:   virtualMachineScaleSetsClient,
		VirtualMachineScaleSetVMsClient: virtualMachineScaleSetVMsClient,
		VirtualMachineRunCommandsClient: virtualMachineRunCommandClient,
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
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	rg, err := resourceClient.Get(ctx, region.ResourceGroup)
	if err != nil {
		return nil
	}

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
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	vmClient := compute.NewVirtualMachinesClient(credential.SubscriptionId)
	vmClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

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
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

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
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

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
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, &sgClient, nil
}

func getSecurityGroupRuleClient(credential idrv.CredentialInfo) (context.Context, *network.SecurityRulesClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	sgClient := network.NewSecurityRulesClient(credential.SubscriptionId)
	sgClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

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
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

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
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

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
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

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
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, &subnetClient, nil
}

func getSshKeyClient(credential idrv.CredentialInfo) (context.Context, *compute.SSHPublicKeysClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	sshClientClient := compute.NewSSHPublicKeysClient(credential.SubscriptionId)
	sshClientClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, &sshClientClient, nil
}

func getVMImageClient(credential idrv.CredentialInfo) (context.Context, *compute.VirtualMachineImagesClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	vmImageClient := compute.NewVirtualMachineImagesClient(credential.SubscriptionId)
	vmImageClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

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
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, &diskClient, nil
}

func getVmSpecClient(credential idrv.CredentialInfo) (context.Context, *compute.VirtualMachineSizesClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	vmSpecClient := compute.NewVirtualMachineSizesClient(credential.SubscriptionId)
	vmSpecClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, &vmSpecClient, nil
}

func getNLBClient(credential idrv.CredentialInfo) (context.Context, *network.LoadBalancersClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	nlbClient := network.NewLoadBalancersClient(credential.SubscriptionId)
	nlbClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, &nlbClient, nil
}

func getNLBBackendAddressPoolsClient(credential idrv.CredentialInfo) (context.Context, *network.LoadBalancerBackendAddressPoolsClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	nlbBackendAddressPoolsClient := network.NewLoadBalancerBackendAddressPoolsClient(credential.SubscriptionId)
	nlbBackendAddressPoolsClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, &nlbBackendAddressPoolsClient, nil
}

func getLoadBalancingRulesClient(credential idrv.CredentialInfo) (context.Context, *network.LoadBalancerLoadBalancingRulesClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	nlbBackendAddressPoolsClient := network.NewLoadBalancerLoadBalancingRulesClient(credential.SubscriptionId)
	nlbBackendAddressPoolsClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, &nlbBackendAddressPoolsClient, nil
}

func getMetricClient(credential idrv.CredentialInfo) (context.Context, *insights.MetricsClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}

	metricClient := insights.NewMetricsClient(credential.SubscriptionId)
	metricClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, &metricClient, nil
}

func getManagedClustersClient(credential idrv.CredentialInfo) (context.Context, *containerservice.ManagedClustersClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}
	managedClustersClient := containerservice.NewManagedClustersClient(credential.SubscriptionId)
	managedClustersClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, &managedClustersClient, nil
}

func getAgentPoolsClient(credential idrv.CredentialInfo) (context.Context, *containerservice.AgentPoolsClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}
	agentPoolsClient := containerservice.NewAgentPoolsClient(credential.SubscriptionId)
	agentPoolsClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, &agentPoolsClient, nil
}

func getVirtualMachineScaleSetsClient(credential idrv.CredentialInfo) (context.Context, *compute.VirtualMachineScaleSetsClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}
	virtualMachineScaleSetsClient := compute.NewVirtualMachineScaleSetsClient(credential.SubscriptionId)
	virtualMachineScaleSetsClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, &virtualMachineScaleSetsClient, nil
}

func getVirtualMachineScaleSetVMsClient(credential idrv.CredentialInfo) (context.Context, *compute.VirtualMachineScaleSetVMsClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}
	virtualMachineScaleSetVMsClient := compute.NewVirtualMachineScaleSetVMsClient(credential.SubscriptionId)
	virtualMachineScaleSetVMsClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, &virtualMachineScaleSetVMsClient, nil
}

func getVirtualMachineRunCommandClient(credential idrv.CredentialInfo) (context.Context, *compute.VirtualMachineRunCommandsClient, error) {
	config := auth.NewClientCredentialsConfig(credential.ClientId, credential.ClientSecret, credential.TenantId)
	authorizer, err := config.Authorizer()
	if err != nil {
		return nil, nil, err
	}
	virtualMachineRunCommandsClient := compute.NewVirtualMachineRunCommandsClient(credential.SubscriptionId)
	virtualMachineRunCommandsClient.Authorizer = authorizer
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, &virtualMachineRunCommandsClient, nil
}
