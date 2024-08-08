// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by hyokyung.kim@innogrid.co.kr, 2019.07.

package connect

import (
	"context"
	"errors"
	"github.com/Azure/azure-sdk-for-go/sdk/monitor/azquery"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"

	cblog "github.com/cloud-barista/cb-log"
	azrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/azure/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type AzureCloudConnection struct {
	CredentialInfo                  idrv.CredentialInfo
	Region                          idrv.RegionInfo
	Ctx                             context.Context
	SubscriptionsClient             *armsubscription.SubscriptionsClient
	VMClient                        *armcompute.VirtualMachinesClient
	ImageClient                     *armcompute.ImagesClient
	VMImageClient                   *armcompute.VirtualMachineImagesClient
	PublicIPClient                  *armnetwork.PublicIPAddressesClient
	SecurityGroupClient             *armnetwork.SecurityGroupsClient
	SecurityGroupRuleClient         *armnetwork.SecurityRulesClient
	VNetClient                      *armnetwork.VirtualNetworksClient
	VNicClient                      *armnetwork.InterfacesClient
	IPConfigClient                  *armnetwork.InterfaceIPConfigurationsClient
	SubnetClient                    *armnetwork.SubnetsClient
	DiskClient                      *armcompute.DisksClient
	VmSpecClient                    *armcompute.VirtualMachineSizesClient
	SshKeyClient                    *armcompute.SSHPublicKeysClient
	NLBClient                       *armnetwork.LoadBalancersClient
	NLBBackendAddressPoolsClient    *armnetwork.LoadBalancerBackendAddressPoolsClient
	NLBLoadBalancingRulesClient     *armnetwork.LoadBalancerLoadBalancingRulesClient
	MetricClient                    *azquery.MetricsClient
	ManagedClustersClient           *armcontainerservice.ManagedClustersClient
	AgentPoolsClient                *armcontainerservice.AgentPoolsClient
	VirtualMachineScaleSetsClient   *armcompute.VirtualMachineScaleSetsClient
	VirtualMachineScaleSetVMsClient *armcompute.VirtualMachineScaleSetVMsClient
	VirtualMachineRunCommandsClient *armcompute.VirtualMachineRunCommandsClient
	ResourceGroupsClient            *armresources.ResourceGroupsClient
	ResourceSKUsClient              *armcompute.ResourceSKUsClient
	TagsClient                      *armresources.TagsClient
}

func (cloudConn *AzureCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("Azure Cloud Driver: called CreateImageHandler()!")
	imageHandler := azrs.AzureImageHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.ImageClient, cloudConn.VMImageClient}
	return &imageHandler, nil
}

/*func (cloudConn *AzureCloudConnection) CreateVNetworkHandler() (irs.VNetworkHandler, error) {
	cblogger.Info("Azure Cloud Driver: called CreateVNetworkHandler()!")
	vNetHandler := azrs.AzureVPCHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.VNetClient, cloudConn.SubnetClient}
	return &vNetHandler, nil
}*/

func (cloudConn *AzureCloudConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	cblogger.Info("Azure Cloud Driver: called CreateVPCHandler()!")
	vpcHandler := azrs.AzureVPCHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.VNetClient, cloudConn.SubnetClient}
	return &vpcHandler, nil
}

func (cloudConn *AzureCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("Azure Cloud Driver: called CreateSecurityHandler()!")
	sgHandler := azrs.AzureSecurityHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.SecurityGroupClient, cloudConn.SecurityGroupRuleClient}
	return &sgHandler, nil
}

func (cloudConn *AzureCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("Azure Cloud Driver: called CreateKeyPairHandler()!")
	keypairHandler := azrs.AzureKeyPairHandler{cloudConn.CredentialInfo, cloudConn.Region, cloudConn.Ctx, cloudConn.SshKeyClient}
	return &keypairHandler, nil
}

/*func (cloudConn *AzureCloudConnection) CreateVNicHandler() (irs.VNicHandler, error) {
	cblogger.Info("Azure Cloud Driver: called CreateVNicHandler()!")
	vNicHandler := azrs.AzureVNicHandler{cloudConn.CredentialInfo, cloudConn.Region, cloudConn.Ctx, cloudConn.VNicClient, cloudConn.SubnetClient}
	return &vNicHandler, nil
}*/

/*func (cloudConn *AzureCloudConnection) CreatePublicIPHandler() (irs.PublicIPHandler, error) {
	cblogger.Info("Azure Cloud Driver: called CreatePublicIPHandler()!")
	publicIPHandler := azrs.AzurePublicIPHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.PublicIPClient, cloudConn.IPConfigClient}
	return &publicIPHandler, nil
}*/

func (cloudConn *AzureCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("Azure Cloud Driver: called CreateVMHandler()!")
	vmHandler := azrs.AzureVMHandler{
		CredentialInfo:                  cloudConn.CredentialInfo,
		Region:                          cloudConn.Region,
		Ctx:                             cloudConn.Ctx,
		Client:                          cloudConn.VMClient,
		SubnetClient:                    cloudConn.SubnetClient,
		NicClient:                       cloudConn.VNicClient,
		PublicIPClient:                  cloudConn.PublicIPClient,
		DiskClient:                      cloudConn.DiskClient,
		SshKeyClient:                    cloudConn.SshKeyClient,
		ImageClient:                     cloudConn.ImageClient,
		VirtualMachineRunCommandsClient: cloudConn.VirtualMachineRunCommandsClient,
	}
	return &vmHandler, nil
}

func (cloudConn *AzureCloudConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	cblogger.Info("Azure Cloud Driver: called CreateVMSpecHandler()!")
	vmSpecHandler := azrs.AzureVmSpecHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.VmSpecClient}
	return &vmSpecHandler, nil
}

func (cloudConn *AzureCloudConnection) CreateNLBHandler() (irs.NLBHandler, error) {
	cblogger.Info("Azure Cloud Driver: called CreateNLBHandler()!")
	nlbHandler := azrs.AzureNLBHandler{
		CredentialInfo:               cloudConn.CredentialInfo,
		Region:                       cloudConn.Region,
		Ctx:                          cloudConn.Ctx,
		NLBClient:                    cloudConn.NLBClient,
		NLBBackendAddressPoolsClient: cloudConn.NLBBackendAddressPoolsClient,
		VNicClient:                   cloudConn.VNicClient,
		PublicIPClient:               cloudConn.PublicIPClient,
		VMClient:                     cloudConn.VMClient,
		SubnetClient:                 cloudConn.SubnetClient,
		IPConfigClient:               cloudConn.IPConfigClient,
		NLBLoadBalancingRulesClient:  cloudConn.NLBLoadBalancingRulesClient,
		MetricClient:                 cloudConn.MetricClient,
	}
	return &nlbHandler, nil
}

func (cloudConn *AzureCloudConnection) CreateDiskHandler() (irs.DiskHandler, error) {
	cblogger.Info("Azure Cloud Driver: called CreateDiskHandler()!")
	diskHandler := azrs.AzureDiskHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		Ctx:            cloudConn.Ctx,
		DiskClient:     cloudConn.DiskClient,
		VMClient:       cloudConn.VMClient,
	}
	return &diskHandler, nil
}

func (cloudConn *AzureCloudConnection) CreateMyImageHandler() (irs.MyImageHandler, error) {
	cblogger.Info("Azure Cloud Driver: called CreateMyImageHandler()!")
	myImageHandler := azrs.AzureMyImageHandler{
		CredentialInfo:                  cloudConn.CredentialInfo,
		Region:                          cloudConn.Region,
		Ctx:                             cloudConn.Ctx,
		ImageClient:                     cloudConn.ImageClient,
		VMClient:                        cloudConn.VMClient,
		VirtualMachineRunCommandsClient: cloudConn.VirtualMachineRunCommandsClient,
	}
	return &myImageHandler, nil
}

func (cloudConn *AzureCloudConnection) CreateRegionZoneHandler() (irs.RegionZoneHandler, error) {
	cblogger.Info("Azure Cloud Driver: called CreateRegionZoneHandler()!")
	regionZoneHandler := azrs.AzureRegionZoneHandler{
		CredentialInfo:      cloudConn.CredentialInfo,
		Region:              cloudConn.Region,
		Ctx:                 cloudConn.Ctx,
		SubscriptionsClient: cloudConn.SubscriptionsClient,
		GroupsClient:        cloudConn.ResourceGroupsClient,
		ResourceSkusClient:  cloudConn.ResourceSKUsClient,
	}
	return &regionZoneHandler, nil
}

func (cloudConn *AzureCloudConnection) CreatePriceInfoHandler() (irs.PriceInfoHandler, error) {
	cblogger.Info("Azure Cloud Driver: called CreatePriceInfoHandler()!")
	priceInfoHandler := azrs.AzurePriceInfoHandler{
		CredentialInfo:     cloudConn.CredentialInfo,
		Region:             cloudConn.Region,
		Ctx:                cloudConn.Ctx,
		ResourceSkusClient: cloudConn.ResourceSKUsClient,
	}
	return &priceInfoHandler, nil
}

func (cloudConn *AzureCloudConnection) IsConnected() (bool, error) {
	return true, nil
}

func (cloudConn *AzureCloudConnection) Close() error {
	return nil
}

func (cloudConn *AzureCloudConnection) CreateClusterHandler() (irs.ClusterHandler, error) {
	cblogger.Info("Azure Cloud Driver: called CreateClusterHandler()!")
	clusterHandler := azrs.AzureClusterHandler{
		CredentialInfo:                  cloudConn.CredentialInfo,
		Region:                          cloudConn.Region,
		Ctx:                             cloudConn.Ctx,
		ManagedClustersClient:           cloudConn.ManagedClustersClient,
		VirtualNetworksClient:           cloudConn.VNetClient,
		AgentPoolsClient:                cloudConn.AgentPoolsClient,
		VirtualMachineScaleSetsClient:   cloudConn.VirtualMachineScaleSetsClient,
		VirtualMachineScaleSetVMsClient: cloudConn.VirtualMachineScaleSetVMsClient,
		SubnetClient:                    cloudConn.SubnetClient,
		SecurityGroupsClient:            cloudConn.SecurityGroupClient,
		SecurityRulesClient:             cloudConn.SecurityGroupRuleClient,
		VirtualMachineSizesClient:       cloudConn.VmSpecClient,
		SSHPublicKeysClient:             cloudConn.SshKeyClient,
	}
	return &clusterHandler, nil
}

func (cloudConn *AzureCloudConnection) CreateAnyCallHandler() (irs.AnyCallHandler, error) {
	return nil, errors.New("Azure Driver: not implemented")
}

func (cloudConn *AzureCloudConnection) CreateTagHandler() (irs.TagHandler, error) {
	cblogger.Info("Azure Cloud Driver: called CreateTagHandler()!")
	tagHandler := azrs.AzureTagHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		Ctx:            cloudConn.Ctx,
		Client:         cloudConn.TagsClient,
	}
	return &tagHandler, nil
	// return nil, errors.New("Azure Driver: not implemented")
}
