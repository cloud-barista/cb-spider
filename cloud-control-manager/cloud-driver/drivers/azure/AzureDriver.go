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
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/arm"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	cblogger "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"

	azcon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/azure/connect"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/azure/profile"
	azrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/azure/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	ires "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

var cblog *logrus.Logger

func init() {
	cblog = cblogger.GetLogger("CLOUD-BARISTA")
}

type AzureDriver struct{}

func (AzureDriver) GetDriverVersion() string {
	return "AZURE DRIVER Version 1.0"
}

const (
	cspTimeout time.Duration = 6000
)

func (AzureDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
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

	drvCapabilityInfo.TagHandler = true
	// ires.SUBNET: not supported (Azure: tagging to VPC)
	drvCapabilityInfo.TagSupportResourceType = []ires.RSType{ires.VPC, ires.SG, ires.KEY, ires.VM, ires.NLB, ires.DISK, ires.MYIMAGE, ires.CLUSTER}

	drvCapabilityInfo.VPC_CIDR = true

	return drvCapabilityInfo
}

func getResourceGroupsClient(credential idrv.CredentialInfo) (context.Context, *armresources.ResourceGroupsClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}
	resourceGroupsClient, err := armresources.NewResourceGroupsClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, resourceGroupsClient, nil
}

func hasResourceGroup(connectionInfo idrv.ConnectionInfo) (bool, error) {
	ctx, resourceGroupsClient, err := getResourceGroupsClient(connectionInfo.CredentialInfo)
	if err != nil {
		return false, err
	}

	_, err = (*resourceGroupsClient).Get(ctx, connectionInfo.RegionInfo.Region, nil)
	if err != nil {
		reErr, ok := err.(*azcore.ResponseError)
		if ok {
			if reErr.ErrorCode == "ResourceGroupNotFound" {
				return false, nil
			}
		}
	}

	return true, nil
}

func createResourceGroup(connectionInfo idrv.ConnectionInfo) error {
	ctx, resourceGroupsClient, err := getResourceGroupsClient(connectionInfo.CredentialInfo)
	if err != nil {
		return err
	}

	_, err = resourceGroupsClient.CreateOrUpdate(ctx, connectionInfo.RegionInfo.Region,
		armresources.ResourceGroup{
			Name:     &connectionInfo.RegionInfo.Region,
			Location: &connectionInfo.RegionInfo.Region,
		}, nil)
	if err != nil {
		return err
	}

	return nil
}

func (driver *AzureDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	// Initialize Logger
	azrs.InitLog()

	// Credentail에 등록된 ResourceGroup 존재 여부 체크 및 생성
	exist, err := hasResourceGroup(connectionInfo)
	if err != nil {
		return nil, err
	}
	if !exist {
		err = createResourceGroup(connectionInfo)
		if err != nil {
			return nil, err
		}
	}

	Ctx, subscriptionsClient, err := getSubscriptionClient(connectionInfo.CredentialInfo)
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
	Ctx, resourceGroupsClient, err := getResourceGroupsClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, resourceSKUsClient, err := getResourceSKUsClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	Ctx, tagsClient, err := getTagsClient(connectionInfo.CredentialInfo)
	if err != nil {
		return nil, err
	}
	iConn := azcon.AzureCloudConnection{
		CredentialInfo:                  connectionInfo.CredentialInfo,
		Region:                          connectionInfo.RegionInfo,
		Ctx:                             Ctx,
		SubscriptionsClient:             subscriptionsClient,
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
		ResourceGroupsClient:            resourceGroupsClient,
		TagsClient:                      tagsClient,
		ResourceSKUsClient:              resourceSKUsClient,
	}

	regionZoneHandler, err := iConn.CreateRegionZoneHandler()
	if err != nil {
		return nil, err
	}
	regionZoneInfo, err := regionZoneHandler.GetRegionZone(connectionInfo.RegionInfo.Region)
	if err != nil {
		return nil, err
	}

	if len(regionZoneInfo.ZoneList) == 0 {
		cblog.Warn("Zone is not available for this region. (" + connectionInfo.RegionInfo.Region + ")")
		iConn.Region.Zone = ""
	} else {
		var zoneFound bool
		for _, zone := range regionZoneInfo.ZoneList {
			if zone.Name == connectionInfo.RegionInfo.Zone {
				zoneFound = true
				break
			}
		}

		if !zoneFound {
			cblog.Warn("Configured zone is not found in the selected region." +
				" (Region: " + connectionInfo.RegionInfo.Region + ", Zone: " + connectionInfo.RegionInfo.Zone + ")")
			cblog.Warn("1 will be used as the default zone.")
			iConn.Region.Zone = "1"
		}
	}

	return &iConn, nil
}

func getCred(credential idrv.CredentialInfo) (*azidentity.ClientSecretCredential, error) {
	cred, err := azidentity.NewClientSecretCredential(credential.TenantId, credential.ClientId, credential.ClientSecret, nil)
	if err != nil {
		return nil, err
	}

	return cred, nil
}

func getSubscriptionClient(credential idrv.CredentialInfo) (context.Context, *armsubscription.SubscriptionsClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}

	subscriptionsClient, err := armsubscription.NewSubscriptionsClient(cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, subscriptionsClient, nil
}

func getVMClient(credential idrv.CredentialInfo) (context.Context, *armcompute.VirtualMachinesClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}

	var clientOptions *arm.ClientOptions

	if os.Getenv("CALL_COUNT") != "" {
		clientOptions = &arm.ClientOptions{
			ClientOptions: azcore.ClientOptions{
				PerCallPolicies: []policy.Policy{profile.NewCountingPolicy()},
			},
		}
	} else {
		clientOptions = &arm.ClientOptions{}
	}

	vmClient, err := armcompute.NewVirtualMachinesClient(credential.SubscriptionId, cred, clientOptions)
	if err != nil {
		return nil, nil, err
	}

	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, vmClient, nil
}

func getImageClient(credential idrv.CredentialInfo) (context.Context, *armcompute.ImagesClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}

	imageClient, err := armcompute.NewImagesClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, imageClient, nil
}

func getPublicIPClient(credential idrv.CredentialInfo) (context.Context, *armnetwork.PublicIPAddressesClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}

	publicIPClient, err := armnetwork.NewPublicIPAddressesClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, publicIPClient, nil
}

func getSecurityGroupClient(credential idrv.CredentialInfo) (context.Context, *armnetwork.SecurityGroupsClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}

	sgClient, err := armnetwork.NewSecurityGroupsClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, sgClient, nil
}

func getSecurityGroupRuleClient(credential idrv.CredentialInfo) (context.Context, *armnetwork.SecurityRulesClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}

	sgClient, err := armnetwork.NewSecurityRulesClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, sgClient, nil
}

func getVNetworkClient(credential idrv.CredentialInfo) (context.Context, *armnetwork.VirtualNetworksClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}

	vNetClient, err := armnetwork.NewVirtualNetworksClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, vNetClient, nil
}

func getVNicClient(credential idrv.CredentialInfo) (context.Context, *armnetwork.InterfacesClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}

	vNicClient, err := armnetwork.NewInterfacesClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, vNicClient, nil
}

func getIPConfigClient(credential idrv.CredentialInfo) (context.Context, *armnetwork.InterfaceIPConfigurationsClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}

	ipConfigClient, err := armnetwork.NewInterfaceIPConfigurationsClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, ipConfigClient, nil
}

func getSubnetClient(credential idrv.CredentialInfo) (context.Context, *armnetwork.SubnetsClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}

	subnetClient, err := armnetwork.NewSubnetsClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, subnetClient, nil
}

func getSshKeyClient(credential idrv.CredentialInfo) (context.Context, *armcompute.SSHPublicKeysClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}

	sshClientClient, err := armcompute.NewSSHPublicKeysClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, sshClientClient, nil
}

func getVMImageClient(credential idrv.CredentialInfo) (context.Context, *armcompute.VirtualMachineImagesClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}

	vmImageClient, err := armcompute.NewVirtualMachineImagesClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, vmImageClient, nil
}

func getDiskClient(credential idrv.CredentialInfo) (context.Context, *armcompute.DisksClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}

	diskClient, err := armcompute.NewDisksClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, diskClient, nil
}

func getVmSpecClient(credential idrv.CredentialInfo) (context.Context, *armcompute.VirtualMachineSizesClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}

	vmSpecClient, err := armcompute.NewVirtualMachineSizesClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, vmSpecClient, nil
}

func getNLBClient(credential idrv.CredentialInfo) (context.Context, *armnetwork.LoadBalancersClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}

	nlbClient, err := armnetwork.NewLoadBalancersClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, nlbClient, nil
}

func getNLBBackendAddressPoolsClient(credential idrv.CredentialInfo) (context.Context, *armnetwork.LoadBalancerBackendAddressPoolsClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}

	nlbBackendAddressPoolsClient, err := armnetwork.NewLoadBalancerBackendAddressPoolsClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, nlbBackendAddressPoolsClient, nil
}

func getLoadBalancingRulesClient(credential idrv.CredentialInfo) (context.Context, *armnetwork.LoadBalancerLoadBalancingRulesClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}

	nlbBackendAddressPoolsClient, err := armnetwork.NewLoadBalancerLoadBalancingRulesClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, nlbBackendAddressPoolsClient, nil
}

func getMetricClient(credential idrv.CredentialInfo) (context.Context, *azquery.MetricsClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}

	metricClient, err := azquery.NewMetricsClient(cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, metricClient, nil
}

func getManagedClustersClient(credential idrv.CredentialInfo) (context.Context, *armcontainerservice.ManagedClustersClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}
	managedClustersClient, err := armcontainerservice.NewManagedClustersClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, managedClustersClient, nil
}

func getAgentPoolsClient(credential idrv.CredentialInfo) (context.Context, *armcontainerservice.AgentPoolsClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}
	agentPoolsClient, err := armcontainerservice.NewAgentPoolsClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, agentPoolsClient, nil
}

func getVirtualMachineScaleSetsClient(credential idrv.CredentialInfo) (context.Context, *armcompute.VirtualMachineScaleSetsClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}
	virtualMachineScaleSetsClient, err := armcompute.NewVirtualMachineScaleSetsClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, virtualMachineScaleSetsClient, nil
}

func getVirtualMachineScaleSetVMsClient(credential idrv.CredentialInfo) (context.Context, *armcompute.VirtualMachineScaleSetVMsClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}
	virtualMachineScaleSetVMsClient, err := armcompute.NewVirtualMachineScaleSetVMsClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, virtualMachineScaleSetVMsClient, nil
}

func getVirtualMachineRunCommandClient(credential idrv.CredentialInfo) (context.Context, *armcompute.VirtualMachineRunCommandsClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}
	virtualMachineRunCommandsClient, err := armcompute.NewVirtualMachineRunCommandsClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, virtualMachineRunCommandsClient, nil
}

func getResourceSKUsClient(credential idrv.CredentialInfo) (context.Context, *armcompute.ResourceSKUsClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}
	resourceSKUsClient, err := armcompute.NewResourceSKUsClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, resourceSKUsClient, nil
}
func getTagsClient(credential idrv.CredentialInfo) (context.Context, *armresources.TagsClient, error) {
	cred, err := getCred(credential)
	if err != nil {
		return nil, nil, err
	}

	tagsClient, err := armresources.NewTagsClient(credential.SubscriptionId, cred, nil)
	if err != nil {
		return nil, nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	return ctx, tagsClient, nil
}
