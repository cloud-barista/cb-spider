package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2021-03-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/containerservice/mgmt/2022-03-01/containerservice"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"math"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	maxPodCount = 110
)

type AzureClusterHandler struct {
	CredentialInfo                  idrv.CredentialInfo
	Region                          idrv.RegionInfo
	Ctx                             context.Context
	ManagedClustersClient           *containerservice.ManagedClustersClient
	VirtualNetworksClient           *network.VirtualNetworksClient
	AgentPoolsClient                *containerservice.AgentPoolsClient
	VirtualMachineScaleSetsClient   *compute.VirtualMachineScaleSetsClient
	VirtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient
	SubnetClient                    *network.SubnetsClient
	SecurityGroupsClient            *network.SecurityGroupsClient
	SecurityRulesClient             *network.SecurityRulesClient
	VirtualMachineSizesClient       *compute.VirtualMachineSizesClient
	SSHPublicKeysClient             *compute.SSHPublicKeysClient
}

func (ac *AzureClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (info irs.ClusterInfo, createErr error) {
	hiscallInfo := GetCallLogScheme(ac.Region, call.CLUSTER, clusterReqInfo.IId.NameId, "CreateCluster()")
	start := call.Start()
	cluster, err := createCluster(clusterReqInfo, ac.VirtualNetworksClient, ac.ManagedClustersClient, ac.VirtualMachineSizesClient, ac.SSHPublicKeysClient, ac.CredentialInfo, ac.Region, ac.Ctx)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create Cluster. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.ClusterInfo{}, createErr
	}
	for _, sg := range clusterReqInfo.Network.SecurityGroupIIDs {
		err = applySecurityGroup(cluster, sg, ac.AgentPoolsClient, ac.VirtualMachineScaleSetsClient, ac.SecurityGroupsClient, ac.SecurityRulesClient, ac.Ctx)
		if err != nil {
			createErr = errors.New(fmt.Sprintf("Failed to Create Cluster. err = %s", err))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.ClusterInfo{}, createErr
		}
	}
	cluster, err = getRawCluster(clusterReqInfo.IId, ac.ManagedClustersClient, ac.Ctx, ac.CredentialInfo, ac.Region)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create Cluster. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.ClusterInfo{}, createErr
	}
	info, err = setterClusterInfo(cluster, ac.VirtualNetworksClient, ac.AgentPoolsClient, ac.VirtualMachineScaleSetsClient, ac.VirtualMachineScaleSetVMsClient, ac.CredentialInfo, ac.Region, ac.Ctx)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create Cluster. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.ClusterInfo{}, createErr
	}
	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func (ac *AzureClusterHandler) ListCluster() (listInfo []*irs.ClusterInfo, getErr error) {
	hiscallInfo := GetCallLogScheme(ac.Region, call.CLUSTER, "CLUSTER", "ListCluster()")
	start := call.Start()
	clusterList, err := ac.ManagedClustersClient.List(ac.Ctx)
	if err != nil {
		getErr = errors.New(fmt.Sprintf("Failed to List Cluster. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return make([]*irs.ClusterInfo, 0), getErr
	}
	listInfo = make([]*irs.ClusterInfo, len(clusterList.Values()))
	for i, cluster := range clusterList.Values() {
		info, err := setterClusterInfo(cluster, ac.VirtualNetworksClient, ac.AgentPoolsClient, ac.VirtualMachineScaleSetsClient, ac.VirtualMachineScaleSetVMsClient, ac.CredentialInfo, ac.Region, ac.Ctx)
		if err != nil {
			getErr = errors.New(fmt.Sprintf("Failed to List Cluster. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return make([]*irs.ClusterInfo, 0), getErr
		}
		listInfo[i] = &info
	}
	LoggingInfo(hiscallInfo, start)
	return listInfo, nil
}

func (ac *AzureClusterHandler) GetCluster(clusterIID irs.IID) (info irs.ClusterInfo, getErr error) {
	hiscallInfo := GetCallLogScheme(ac.Region, call.CLUSTER, clusterIID.NameId, "GetCluster()")
	start := call.Start()

	cluster, err := getRawCluster(clusterIID, ac.ManagedClustersClient, ac.Ctx, ac.CredentialInfo, ac.Region)
	if err != nil {
		getErr = errors.New(fmt.Sprintf("Failed to Get Cluster. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.ClusterInfo{}, getErr
	}

	info, err = setterClusterInfo(cluster, ac.VirtualNetworksClient, ac.AgentPoolsClient, ac.VirtualMachineScaleSetsClient, ac.VirtualMachineScaleSetVMsClient, ac.CredentialInfo, ac.Region, ac.Ctx)
	if err != nil {
		getErr = errors.New(fmt.Sprintf("Failed to Get Cluster. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.ClusterInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return info, nil
}
func (ac *AzureClusterHandler) DeleteCluster(clusterIID irs.IID) (deleteResult bool, delErr error) {
	hiscallInfo := GetCallLogScheme(ac.Region, call.CLUSTER, clusterIID.NameId, "DeleteCluster()")
	start := call.Start()

	err := cleanCluster(clusterIID.NameId, ac.ManagedClustersClient, ac.Region.ResourceGroup, ac.Ctx)
	if err != nil {
		delErr = errors.New(fmt.Sprintf("Failed to Delete Cluster. err = %s", err))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (ac *AzureClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (nodeInfo irs.NodeGroupInfo, addNodeErr error) {
	hiscallInfo := GetCallLogScheme(ac.Region, call.CLUSTER, clusterIID.NameId, "AddNodeGroup()")
	start := call.Start()
	cluster, err := getRawCluster(clusterIID, ac.ManagedClustersClient, ac.Ctx, ac.CredentialInfo, ac.Region)
	if err != nil {
		addNodeErr = errors.New(fmt.Sprintf("Failed to Add NodeGroup. err = %s", err))
		cblogger.Error(addNodeErr.Error())
		LoggingError(hiscallInfo, addNodeErr)
		return irs.NodeGroupInfo{}, addNodeErr
	}
	nodeGroupInfo, err := addNodeGroupPool(cluster, nodeGroupReqInfo, ac.ManagedClustersClient, ac.AgentPoolsClient, ac.VirtualMachineScaleSetsClient, ac.VirtualMachineScaleSetVMsClient, ac.SubnetClient, ac.CredentialInfo, ac.Region, ac.Ctx)
	if err != nil {
		addNodeErr = errors.New(fmt.Sprintf("Failed to Add NodeGroup. err = %s", err))
		cblogger.Error(addNodeErr.Error())
		LoggingError(hiscallInfo, addNodeErr)
		return irs.NodeGroupInfo{}, addNodeErr
	}
	LoggingInfo(hiscallInfo, start)
	return nodeGroupInfo, nil
}

func (ac *AzureClusterHandler) SetNodeGroupAutoScaling(clusterIID irs.IID, nodeGroupIID irs.IID, on bool) (result bool, setErr error) {
	hiscallInfo := GetCallLogScheme(ac.Region, call.CLUSTER, clusterIID.NameId, "SetNodeGroupAutoScaling()")
	start := call.Start()

	cluster, err := getRawCluster(clusterIID, ac.ManagedClustersClient, ac.Ctx, ac.CredentialInfo, ac.Region)
	if err != nil {
		setErr = errors.New(fmt.Sprintf("Failed to Set NodeGroupAutoScaling. err = %s", err))
		cblogger.Error(setErr.Error())
		LoggingError(hiscallInfo, setErr)
		return false, setErr
	}

	err = autoScalingChange(cluster, nodeGroupIID, on, ac.AgentPoolsClient, ac.Region, ac.Ctx)
	if err != nil {
		setErr = errors.New(fmt.Sprintf("Failed to Set NodeGroupAutoScaling. err = %s", err))
		cblogger.Error(setErr.Error())
		LoggingError(hiscallInfo, setErr)
		return false, setErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}
func (ac *AzureClusterHandler) ChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID, DesiredNodeSize int, MinNodeSize int, MaxNodeSize int) (nodeGroupInfo irs.NodeGroupInfo, changeErr error) {
	hiscallInfo := GetCallLogScheme(ac.Region, call.CLUSTER, clusterIID.NameId, "ChangeNodeGroupScaling()")
	start := call.Start()

	cluster, err := getRawCluster(clusterIID, ac.ManagedClustersClient, ac.Ctx, ac.CredentialInfo, ac.Region)
	if err != nil {
		changeErr = errors.New(fmt.Sprintf("Failed to Change NodeGroupScaling. err = %s", err))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.NodeGroupInfo{}, changeErr
	}
	info, err := changeNodeGroupScaling(cluster, nodeGroupIID, DesiredNodeSize, MinNodeSize, MaxNodeSize, ac.ManagedClustersClient, ac.AgentPoolsClient, ac.VirtualMachineScaleSetsClient, ac.VirtualMachineScaleSetVMsClient, ac.CredentialInfo, ac.Region, ac.Ctx)
	if err != nil {
		changeErr = errors.New(fmt.Sprintf("Failed to Change NodeGroupScaling. err = %s", err))
		cblogger.Error(changeErr.Error())
		LoggingError(hiscallInfo, changeErr)
		return irs.NodeGroupInfo{}, changeErr
	}

	LoggingInfo(hiscallInfo, start)
	return info, nil
}
func (ac *AzureClusterHandler) RemoveNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (result bool, delErr error) {
	hiscallInfo := GetCallLogScheme(ac.Region, call.CLUSTER, clusterIID.NameId, "RemoveNodeGroup()")
	start := call.Start()
	cluster, err := getRawCluster(clusterIID, ac.ManagedClustersClient, ac.Ctx, ac.CredentialInfo, ac.Region)
	if err != nil {
		delErr = errors.New(fmt.Sprintf("Failed to Remove NodeGroup. err = %s", err))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	err = deleteNodeGroup(cluster, nodeGroupIID, ac.AgentPoolsClient, ac.Region, ac.Ctx)
	if err != nil {
		delErr = errors.New(fmt.Sprintf("Failed to Remove NodeGroup. err = %s", err))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

// ------ Upgrade K8S
func (ac *AzureClusterHandler) UpgradeCluster(clusterIID irs.IID, newVersion string) (info irs.ClusterInfo, upgradeErr error) {
	hiscallInfo := GetCallLogScheme(ac.Region, call.CLUSTER, clusterIID.NameId, "UpgradeCluster()")
	start := call.Start()
	cluster, err := getRawCluster(clusterIID, ac.ManagedClustersClient, ac.Ctx, ac.CredentialInfo, ac.Region)
	if err != nil {
		upgradeErr = errors.New(fmt.Sprintf("Failed to Upgrade Cluster. err = %s", err))
		cblogger.Error(upgradeErr.Error())
		LoggingError(hiscallInfo, upgradeErr)
		return irs.ClusterInfo{}, upgradeErr
	}
	err = upgradeCluter(cluster, newVersion, ac.ManagedClustersClient, ac.Ctx, ac.Region)
	if err != nil {
		upgradeErr = errors.New(fmt.Sprintf("Failed to Upgrade Cluster. err = %s", err))
		cblogger.Error(upgradeErr.Error())
		LoggingError(hiscallInfo, upgradeErr)
		return irs.ClusterInfo{}, upgradeErr
	}
	LoggingInfo(hiscallInfo, start)
	return irs.ClusterInfo{}, nil
}

func upgradeCluter(cluster containerservice.ManagedCluster, newVersion string, managedClustersClient *containerservice.ManagedClustersClient, ctx context.Context, region idrv.RegionInfo) error {
	updateCluster := cluster
	updateCluster.KubernetesVersion = to.StringPtr(newVersion)
	upgradeResult, err := managedClustersClient.CreateOrUpdate(ctx, region.ResourceGroup, *cluster.Name, updateCluster)
	if err != nil {
		return err
	}
	err = upgradeResult.WaitForCompletionRef(ctx, managedClustersClient.Client)
	if err != nil {
		return err
	}
	return nil
}

func convertedClusterIID(clusterIID irs.IID, credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo) (irs.IID, error) {
	if clusterIID.NameId == "" && clusterIID.SystemId == "" {
		return clusterIID, errors.New(fmt.Sprintf("invalid IID"))
	}
	if clusterIID.SystemId == "" {
		clusterId := GetClusterIdByName(credentialInfo, regionInfo, clusterIID.NameId)
		return irs.IID{NameId: clusterIID.NameId, SystemId: clusterId}, nil
	} else {
		clusterName, err := GetClusterNameById(clusterIID.SystemId)
		if err != nil {
			return irs.IID{}, err
		}
		return irs.IID{NameId: clusterName, SystemId: clusterIID.SystemId}, nil
	}
}

func getRawCluster(clusterIID irs.IID, managedClustersClient *containerservice.ManagedClustersClient, ctx context.Context, credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo) (containerservice.ManagedCluster, error) {
	clusterName := clusterIID.NameId
	if clusterName == "" {
		convertedIID, err := convertedClusterIID(clusterIID, credentialInfo, regionInfo)
		if err != nil {
			return containerservice.ManagedCluster{}, err
		}
		clusterName = convertedIID.NameId
	}
	cluster, err := managedClustersClient.Get(ctx, regionInfo.ResourceGroup, clusterName)
	if err != nil {
		return containerservice.ManagedCluster{}, err
	}
	return cluster, nil
}

func setterClusterInfo(cluster containerservice.ManagedCluster, virtualNetworksClient *network.VirtualNetworksClient, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context) (clusterInfo irs.ClusterInfo, err error) {
	clusterInfo.IId = irs.IID{*cluster.Name, *cluster.ID}
	if cluster.ManagedClusterProperties != nil {
		// Version
		if cluster.ManagedClusterProperties.KubernetesVersion != nil {
			clusterInfo.Version = *cluster.ManagedClusterProperties.KubernetesVersion
		}
		// NetworkInfo - Network Configuration AzureCNI
		networkInfo, err := getNetworkInfo(cluster, agentPoolsClient, virtualNetworksClient, virtualMachineScaleSetsClient, credentialInfo, region, ctx)
		if err == nil {
			clusterInfo.Network = networkInfo
		}
		NodeGroupInfoList, err := getNodeGroupInfoList(cluster, agentPoolsClient, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, ctx)
		if err == nil {
			clusterInfo.NodeGroupList = NodeGroupInfoList
		}
		addOnInfo, err := getAddonInfo(cluster)
		if err == nil {
			clusterInfo.Addons = addOnInfo
		}
		status := getClusterStatus(cluster)
		if err == nil {
			clusterInfo.Status = status
		}
		keyValues := []irs.KeyValue{}
		if cluster.Tags != nil {
			tags := cluster.Tags
			createdTime, createdTimeExist := tags["createdAt"]
			if createdTimeExist && createdTime != nil {
				timeInt64, err := strconv.ParseInt(*createdTime, 10, 64)
				if err == nil {
					clusterInfo.CreatedTime = time.Unix(timeInt64, 0)
				}
			}
			sshkey, sshKeyExist := tags["sshkey"]
			if sshKeyExist && sshkey != nil {
				keyValues = append(keyValues, irs.KeyValue{Key: "sshkey", Value: *sshkey})
			}
		}
		clusterInfo.KeyValueList = keyValues
	}

	return clusterInfo, nil
}

func getClusterStatus(cluster containerservice.ManagedCluster) irs.ClusterStatus {
	resultStatus := irs.ClusterInactive
	if cluster.ProvisioningState == nil || cluster.PowerState == nil {
		return resultStatus
	}
	provisioningState := *cluster.ProvisioningState
	powerState := cluster.PowerState.Code
	if provisioningState == "Starting" {
		resultStatus = irs.ClusterCreating
	}
	if provisioningState == "Succeeded" && powerState == containerservice.CodeRunning {
		resultStatus = irs.ClusterActive
	}
	if provisioningState == "Deleting" {
		resultStatus = irs.ClusterDeleting
	}
	if provisioningState == "InProgress" {
		resultStatus = irs.ClusterUpdating
	}
	return resultStatus
}

func getAddonInfo(cluster containerservice.ManagedCluster) (info irs.AddonsInfo, err error) {
	defer func() {
		if r := recover(); r != nil {
			info = irs.AddonsInfo{}
			err = errors.New("faild get AddonProfiles")
		}
	}()
	keyvalues := make([]irs.KeyValue, 0)
	for AddonProName, AddonProfile := range cluster.ManagedClusterProperties.AddonProfiles {
		val := "Disabled"
		if *AddonProfile.Enabled {
			val = "Enabled"
		}
		keyvalues = append(keyvalues, irs.KeyValue{Key: AddonProName, Value: val})
	}
	info = irs.AddonsInfo{
		keyvalues,
	}
	return info, nil
}

func getSubnetIdByAgentPoolProfiles(agentPoolProfiles []containerservice.ManagedClusterAgentPoolProfile) (subnetId string, err error) {
	var targetSubnetId *string
	for _, agentPool := range agentPoolProfiles {
		if targetSubnetId == nil && agentPool.VnetSubnetID != nil {
			targetSubnetId = agentPool.VnetSubnetID
			break
		}
	}
	if targetSubnetId == nil {
		return "", errors.New("invalid cluster Network")
	}
	return *targetSubnetId, nil
}

func getNetworkInfo(cluster containerservice.ManagedCluster, agentPoolsClient *containerservice.AgentPoolsClient, virtualNetworksClient *network.VirtualNetworksClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, CredentialInfo idrv.CredentialInfo, Region idrv.RegionInfo, ctx context.Context) (info irs.NetworkInfo, err error) {
	defer func() {
		if r := recover(); r != nil {
			info = irs.NetworkInfo{}
			err = errors.New("invalid cluster Network")
		}
	}()
	if cluster.ManagedClusterProperties.NetworkProfile.NetworkPlugin == containerservice.NetworkPluginAzure {
		if cluster.ManagedClusterProperties.AgentPoolProfiles != nil && len(*cluster.ManagedClusterProperties.AgentPoolProfiles) > 0 {
			subnetId, err := getSubnetIdByAgentPoolProfiles(*cluster.ManagedClusterProperties.AgentPoolProfiles)
			if subnetId == "" {
				return irs.NetworkInfo{}, errors.New("invalid cluster Network")
			}
			vpcName, err := getNameById(subnetId, AzureVirtualNetworks)
			if err != nil {
				return irs.NetworkInfo{}, errors.New("failed get cluster vpcName")
			}
			vpcId := GetNetworksResourceIdByName(CredentialInfo, Region, AzureVirtualNetworks, vpcName)
			subnetName, err := getNameById(subnetId, AzureSubnet)
			if err != nil {
				return irs.NetworkInfo{}, errors.New("failed get cluster subnetName")
			}
			targetSecurityGroupIdList, err := getClusterAgentPoolSecurityGroupId(cluster, agentPoolsClient, virtualMachineScaleSetsClient, ctx)
			if err != nil {
				return irs.NetworkInfo{}, errors.New("failed get cluster SecurityGroups")
			}
			sgIIDs := make([]irs.IID, len(targetSecurityGroupIdList))
			for i, sgid := range targetSecurityGroupIdList {
				sgName, err := getNameById(sgid, AzureSecurityGroups)
				if err != nil {
					return irs.NetworkInfo{}, errors.New("failed get cluster SecurityGroups")
				}
				sgIIDs[i] = irs.IID{
					NameId:   sgName,
					SystemId: sgid,
				}
			}
			info = irs.NetworkInfo{
				VpcIID: irs.IID{
					NameId:   vpcName,
					SystemId: vpcId,
				},
				SubnetIIDs:        []irs.IID{{subnetName, subnetId}},
				SecurityGroupIIDs: sgIIDs,
			}

			return info, nil
		}
	} else if cluster.ManagedClusterProperties.NetworkProfile.NetworkPlugin == containerservice.NetworkPluginKubenet {
		// NetworkInfo - Network Configuration Kubenet
		if cluster.ManagedClusterProperties.NodeResourceGroup != nil {
			NetworkInfo := irs.NetworkInfo{}
			networkList, err := virtualNetworksClient.List(ctx, *cluster.ManagedClusterProperties.NodeResourceGroup)
			if err == nil && len(networkList.Values()) > 0 {
				vpcNetwork := networkList.Values()[0]
				NetworkInfo.VpcIID = irs.IID{*vpcNetwork.Name, *vpcNetwork.ID}
				subnetIIDArray := make([]irs.IID, len(*vpcNetwork.Subnets))
				segIIDArray := make([]irs.IID, 0)
				for i, subnet := range *vpcNetwork.Subnets {
					subnetIIDArray[i] = irs.IID{*subnet.Name, *subnet.ID}
				}
				NetworkInfo.SubnetIIDs = segIIDArray
				NetworkInfo.SecurityGroupIIDs = segIIDArray
			}
		}
	}
	return irs.NetworkInfo{}, errors.New("empty cluster AgentPoolProfiles")
}

type NodePoolPair struct {
	AgentPool              containerservice.AgentPool
	virtualMachineScaleSet compute.VirtualMachineScaleSet
}

func getRawNodePoolPairList(cluster containerservice.ManagedCluster, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, ctx context.Context) ([]NodePoolPair, error) {
	clusterResourceGroup, err := getResourceGroupById(*cluster.ID)
	if err != nil {
		return make([]NodePoolPair, 0), errors.New(fmt.Sprintf("failed get clusterResourceGroup err = %s", err.Error()))
	}
	if cluster.ManagedClusterProperties.NodeResourceGroup == nil {
		return make([]NodePoolPair, 0), errors.New("invalid cluster Resource")
	}
	resourceGroupManagedK8s := *cluster.ManagedClusterProperties.NodeResourceGroup
	agentPoolList, err := agentPoolsClient.List(ctx, clusterResourceGroup, *cluster.Name)
	if err != nil {
		return nil, err
	}
	if len(agentPoolList.Values()) == 0 {
		return make([]NodePoolPair, 0), err
	}

	sacleSetList, err := virtualMachineScaleSetsClient.List(ctx, resourceGroupManagedK8s)
	if err != nil {
		return nil, err
	}

	var filteredNodePoolPairList []NodePoolPair

	for _, scaleSet := range sacleSetList.Values() {
		value, exist := scaleSet.Tags["aks-managed-poolName"]
		if exist {
			for _, agentPool := range agentPoolList.Values() {
				if *value == *agentPool.Name {
					filteredNodePoolPairList = append(filteredNodePoolPairList, NodePoolPair{
						AgentPool:              agentPool,
						virtualMachineScaleSet: scaleSet,
					})
					break
				}
			}
		}
	}
	return filteredNodePoolPairList, nil
}

func convertNodePoolPairSpecifiedNodePool(cluster containerservice.ManagedCluster, nodePoolName string, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, ctx context.Context) (irs.NodeGroupInfo, error) {
	if cluster.ManagedClusterProperties.NodeResourceGroup == nil {
		return irs.NodeGroupInfo{}, errors.New("invalid cluster resource")
	}
	resourceGroupManagedK8s := *cluster.ManagedClusterProperties.NodeResourceGroup
	nodePoolPairList, err := getRawNodePoolPairList(cluster, agentPoolsClient, virtualMachineScaleSetsClient, ctx)
	if err != nil {
		return irs.NodeGroupInfo{}, err
	}
	if nodePoolName == "" {
		return irs.NodeGroupInfo{}, errors.New("empty nodePool Name")
	}
	var nodeInfo *irs.NodeGroupInfo
	sshkeyName, sshKeyExist := cluster.Tags["sshkey"]
	keyPairIID := irs.IID{}
	if sshKeyExist && sshkeyName != nil {
		keyPairIID.NameId = *sshkeyName
		clusterSubscriptionsById, subscriptionsErr := getSubscriptionsById(*cluster.ID)
		clusterResourceGroup, err := getResourceGroupById(*cluster.ID)
		if err == nil && subscriptionsErr == nil {
			keyPairIID.SystemId = GetSshKeyIdByName(idrv.CredentialInfo{SubscriptionId: clusterSubscriptionsById}, idrv.RegionInfo{
				ResourceGroup: clusterResourceGroup,
			}, *sshkeyName)
		}
	}
	for _, nodePoolPair := range nodePoolPairList {
		scaleSet := nodePoolPair.virtualMachineScaleSet
		agentPool := nodePoolPair.AgentPool
		if *agentPool.Name == nodePoolName {
			nodeInfo = &irs.NodeGroupInfo{
				IId: irs.IID{
					NameId:   *agentPool.Name,
					SystemId: *agentPool.ID,
				},
				ImageIID: irs.IID{
					SystemId: *scaleSet.VirtualMachineProfile.StorageProfile.ImageReference.ID,
					NameId:   *scaleSet.VirtualMachineProfile.StorageProfile.ImageReference.ID,
				},
				VMSpecName:      *scaleSet.Sku.Name,
				RootDiskType:    GetVMDiskInfoType(scaleSet.VirtualMachineProfile.StorageProfile.OsDisk.ManagedDisk.StorageAccountType),
				RootDiskSize:    strconv.Itoa(int(*scaleSet.VirtualMachineProfile.StorageProfile.OsDisk.DiskSizeGB)),
				KeyPairIID:      keyPairIID,
				Status:          getNodeInfoStatus(scaleSet),
				OnAutoScaling:   *agentPool.EnableAutoScaling,
				DesiredNodeSize: int(*agentPool.Count),
				MinNodeSize:     int(*agentPool.MinCount),
				MaxNodeSize:     int(*agentPool.MaxCount),
			}
			if agentPool.MinCount != nil {
				nodeInfo.MinNodeSize = int(*agentPool.MinCount)
			}
			if agentPool.MaxCount != nil {
				nodeInfo.MaxNodeSize = int(*agentPool.MaxCount)
			}
			vmIIds := make([]irs.IID, 0)
			vms, err := virtualMachineScaleSetVMsClient.List(ctx, resourceGroupManagedK8s, *scaleSet.Name, "", "", "")
			if err == nil {
				for _, vm := range vms.Values() {
					vmIIds = append(vmIIds, irs.IID{*vm.Name, *vm.ID})
				}
			}
			nodeInfo.Nodes = vmIIds
			break
		}
		continue
	}
	if nodeInfo == nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("not found nodePool %s", nodePoolName))
	}
	return *nodeInfo, nil
}

func convertNodePoolPair(cluster containerservice.ManagedCluster, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, ctx context.Context) ([]irs.NodeGroupInfo, error) {
	if cluster.ManagedClusterProperties.NodeResourceGroup == nil {
		return make([]irs.NodeGroupInfo, 0), errors.New("invalid cluster resource")
	}
	resourceGroupManagedK8s := *cluster.ManagedClusterProperties.NodeResourceGroup
	nodePoolPairList, err := getRawNodePoolPairList(cluster, agentPoolsClient, virtualMachineScaleSetsClient, ctx)
	if err != nil {
		return make([]irs.NodeGroupInfo, 0), err
	}
	nodeInfoGroupList := make([]irs.NodeGroupInfo, len(nodePoolPairList))

	sshkeyName, sshKeyExist := cluster.Tags["sshkey"]
	keyPairIID := irs.IID{}

	if sshKeyExist && sshkeyName != nil {
		keyPairIID.NameId = *sshkeyName
		clusterSubscriptionsById, subscriptionsErr := getSubscriptionsById(*cluster.ID)
		clusterResourceGroup, err := getResourceGroupById(*cluster.ID)
		if err == nil && subscriptionsErr == nil {
			keyPairIID.SystemId = GetSshKeyIdByName(idrv.CredentialInfo{SubscriptionId: clusterSubscriptionsById}, idrv.RegionInfo{
				ResourceGroup: clusterResourceGroup,
			}, *sshkeyName)
		}
	}
	for i, nodePoolPair := range nodePoolPairList {
		scaleSet := nodePoolPair.virtualMachineScaleSet
		agentPool := nodePoolPair.AgentPool
		nodeInfoGroup := irs.NodeGroupInfo{
			IId: irs.IID{
				NameId:   *agentPool.Name,
				SystemId: *agentPool.ID,
			},
			ImageIID: irs.IID{
				SystemId: *scaleSet.VirtualMachineProfile.StorageProfile.ImageReference.ID,
				NameId:   *scaleSet.VirtualMachineProfile.StorageProfile.ImageReference.ID,
			},
			VMSpecName:      *scaleSet.Sku.Name,
			RootDiskType:    GetVMDiskInfoType(scaleSet.VirtualMachineProfile.StorageProfile.OsDisk.ManagedDisk.StorageAccountType),
			RootDiskSize:    strconv.Itoa(int(*scaleSet.VirtualMachineProfile.StorageProfile.OsDisk.DiskSizeGB)),
			KeyPairIID:      keyPairIID,
			Status:          getNodeInfoStatus(scaleSet),
			OnAutoScaling:   *agentPool.EnableAutoScaling,
			DesiredNodeSize: int(*agentPool.Count),
		}
		if agentPool.MinCount != nil {
			nodeInfoGroup.MinNodeSize = int(*agentPool.MinCount)
		}
		if agentPool.MaxCount != nil {
			nodeInfoGroup.MaxNodeSize = int(*agentPool.MaxCount)
		}
		vmIIds := make([]irs.IID, 0)
		vms, err := virtualMachineScaleSetVMsClient.List(ctx, resourceGroupManagedK8s, *scaleSet.Name, "", "", "")
		if err == nil {
			for _, vm := range vms.Values() {
				vmIIds = append(vmIIds, irs.IID{*vm.Name, *vm.ID})
			}
		}
		nodeInfoGroup.Nodes = vmIIds
		nodeInfoGroupList[i] = nodeInfoGroup
	}
	return nodeInfoGroupList, nil
}

func getNodeGroupInfoSpecifiedNodePool(cluster containerservice.ManagedCluster, nodePoolName string, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, ctx context.Context) (irs.NodeGroupInfo, error) {
	nodeInfoGroup, err := convertNodePoolPairSpecifiedNodePool(cluster, nodePoolName, agentPoolsClient, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, ctx)
	if err != nil {
		return irs.NodeGroupInfo{}, err
	}
	return nodeInfoGroup, nil
}

func getNodeGroupInfoList(cluster containerservice.ManagedCluster, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, ctx context.Context) ([]irs.NodeGroupInfo, error) {
	nodeInfoGroupList, err := convertNodePoolPair(cluster, agentPoolsClient, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, ctx)
	if err != nil {
		return make([]irs.NodeGroupInfo, 0), err
	}
	return nodeInfoGroupList, nil
}

func getNodeInfoStatus(virtualMachineScaleSet compute.VirtualMachineScaleSet) irs.NodeGroupStatus {
	if virtualMachineScaleSet.ProvisioningState == nil {
		return irs.NodeGroupInactive
	}
	switch *virtualMachineScaleSet.ProvisioningState {
	case "Canceled":
		return irs.NodeGroupInactive
	case "Deleting":
		return irs.NodeGroupDeleting
	case "Failed":
		return irs.NodeGroupInactive
	case "InProgress":
		return irs.NodeGroupUpdating
	case "Succeeded":
		return irs.NodeGroupActive
	default:
		return irs.NodeGroupInactive
	}
}

func getclusterDNSPrefix(clusterName string) string {
	return fmt.Sprintf("%s-dns", clusterName)
}

func getclusterNodeResourceGroup(clusterName string, resourceGroup string, regionString string) string {
	return fmt.Sprintf("CB_%s_%s_%s", resourceGroup, clusterName, regionString)
}

func checkValidationNodeGroups(nodeGroups []irs.NodeGroupInfo, virtualMachineSizesClient *compute.VirtualMachineSizesClient, regionInfo idrv.RegionInfo, ctx context.Context) error {
	// https://learn.microsoft.com/en-us/azure/aks/quotas-skus-regions
	if len(nodeGroups) == 0 {
		return errors.New("nodeGroup Empty")
	}
	vmspecListResult, err := virtualMachineSizesClient.List(ctx, regionInfo.Region)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed get VMSPEC List"))
	}
	// Azure의 SSH키는 클러스터 의존, NodeGroup에 의존하지 않음
	var sshKeyIID *irs.IID
	for i, nodeGroup := range nodeGroups {
		// require Name
		if nodeGroup.IId.NameId == "" {
			return errors.New("nodeGroup Name Empty")
		}
		// vmSpec CPU 2이상
		if nodeGroup.VMSpecName == "" {
			return errors.New("nodeGroup VMSpecName Empty")
		}
		if i == 0 {
			if nodeGroup.KeyPairIID.NameId == "" && nodeGroup.KeyPairIID.SystemId == "" {
				return errors.New("nodeGroup KeyPairIID Empty")
			}
			keyIID := nodeGroup.KeyPairIID
			sshKeyIID = &keyIID
		}
		if i != 0 {
			// NameId, SystemId 둘다 값이 있음
			if nodeGroup.KeyPairIID.NameId != "" && nodeGroup.KeyPairIID.SystemId != "" {
				if nodeGroup.KeyPairIID.NameId != sshKeyIID.NameId || nodeGroup.KeyPairIID.SystemId != sshKeyIID.SystemId {
					return errors.New("The SSHkey in the Azure Cluster NodeGroup must all be the same")
				}
			} else if nodeGroup.KeyPairIID.NameId != "" {
				if nodeGroup.KeyPairIID.NameId != sshKeyIID.NameId {
					return errors.New("The SSHkey in the Azure Cluster NodeGroup must all be the same")
				}
			} else if nodeGroup.KeyPairIID.SystemId != "" {
				if nodeGroup.KeyPairIID.SystemId != sshKeyIID.SystemId {
					return errors.New("The SSHkey in the Azure Cluster NodeGroup must all be the same")
				}
			} else {
				// nodeGroup.KeyPairIID.NameId == "" && nodeGroup.KeyPairIID.SystemId == ""
				return errors.New("The SSHkey in the Azure Cluster NodeGroup must all be the same")
			}
		}
		checkVMSpecCore := false
		checkVMSpecExist := false
		for _, rawspec := range *vmspecListResult.Value {
			if *rawspec.Name == nodeGroup.VMSpecName {
				checkVMSpecExist = true
				if *rawspec.NumberOfCores >= 2 {
					checkVMSpecCore = true
				}
				break
			}
		}
		if !checkVMSpecExist {
			return errors.New(fmt.Sprintf("notfound nodeGroup VMSpecName"))
		}
		if !checkVMSpecCore {
			return errors.New(fmt.Sprintf("VMSpec for nodeGroup must have at least 2 cores."))
		}
		// RootDiskType 제공하지 않음 PremiumSSD
		if !(nodeGroup.RootDiskType == "" || strings.ToLower(nodeGroup.RootDiskType) == "default" || strings.ToLower(nodeGroup.RootDiskType) == "PremiumSSD") {
			return errors.New(fmt.Sprintf("The RootDiskType of the Azure Cluster NodeGroup provides Premium SSDs only. Set the RootDiskType of NodeGroup to default or Premium SSD."))
		}
		// RootDiskSize
		if !(nodeGroup.RootDiskSize == "" || nodeGroup.RootDiskSize == "default") {
			_, err := strconv.Atoi(nodeGroup.RootDiskSize)
			if err != nil {
				return errors.New("invalid NodeGroup RootDiskSize")
			}
		}
		// ??? KeyPairIID

		// OnAutoScaling + MinNodeSize
		// MaxNodeSize
		// DesiredNodeSize
		if nodeGroup.OnAutoScaling && nodeGroup.MinNodeSize < 1 {
			return errors.New(fmt.Sprintf("MinNodeSize must be greater than 0 when OnAutoScaling is enabled."))
		}
		if nodeGroup.MinNodeSize > 0 && !nodeGroup.OnAutoScaling {
			return errors.New(fmt.Sprintf("If MinNodeSize is specified, OnAutoScaling must be enabled."))
		}
		if nodeGroup.MinNodeSize > 0 && (nodeGroup.MinNodeSize > nodeGroup.MaxNodeSize) {
			return errors.New(fmt.Sprintf("MaxNodeSize must be greater than MinNodeSize."))
		}
		if nodeGroup.MinNodeSize > 0 && (nodeGroup.DesiredNodeSize < nodeGroup.MinNodeSize) {
			return errors.New(fmt.Sprintf("DesiredNodeSize must be greater than or equal to MinNodeSize."))
		}
		if nodeGroup.MaxNodeSize > 1000 {
			return errors.New(fmt.Sprintf("MaxNodeSize must be 1000 or less."))
		}
	}
	return nil
}

func checkValidationCreateCluster(clusterReqInfo irs.ClusterInfo, virtualMachineSizesClient *compute.VirtualMachineSizesClient, regionInfo idrv.RegionInfo, ctx context.Context) error {
	// nodegroup 확인
	err := checkValidationNodeGroups(clusterReqInfo.NodeGroupList, virtualMachineSizesClient, regionInfo, ctx)
	if err != nil {
		return errors.New(fmt.Sprintf("Failed Validation Check NodeGroup. err = %s", err.Error()))
	}
	return nil
}

func createCluster(clusterReqInfo irs.ClusterInfo, virtualNetworksClient *network.VirtualNetworksClient, managedClustersClient *containerservice.ManagedClustersClient, virtualMachineSizesClient *compute.VirtualMachineSizesClient, sshPublicKeysClient *compute.SSHPublicKeysClient, credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo, ctx context.Context) (containerservice.ManagedCluster, error) {
	// 사전 확인
	err := checkValidationCreateCluster(clusterReqInfo, virtualMachineSizesClient, regionInfo, ctx)
	if err != nil {
		return containerservice.ManagedCluster{}, err
	}
	targetSubnet, err := getRawClusterTargetSubnet(clusterReqInfo.Network, virtualNetworksClient, ctx, regionInfo.ResourceGroup)
	if err != nil {
		return containerservice.ManagedCluster{}, err
	}
	// agentPoolProfiles
	agentPoolProfiles, err := generateAgentPoolProfileList(clusterReqInfo, targetSubnet)
	if err != nil {
		return containerservice.ManagedCluster{}, err
	}
	// networkProfile
	networkProfile, err := generatorNetworkProfile(clusterReqInfo, targetSubnet)
	if err != nil {
		return containerservice.ManagedCluster{}, err
	}
	// mapping ssh
	linuxProfileSSH, sshKey, err := generateManagedClusterLinuxProfileSSH(clusterReqInfo, sshPublicKeysClient, regionInfo.ResourceGroup, ctx)
	if err != nil {
		return containerservice.ManagedCluster{}, err
	}
	tags, err := generatorClusterTags(*sshKey.Name)
	if err != nil {
		return containerservice.ManagedCluster{}, err
	}
	addonProfiles := generatePreparedAddonProfiles()

	clusterCreateOpts := containerservice.ManagedCluster{
		Location: to.StringPtr(regionInfo.Region),
		Sku: &containerservice.ManagedClusterSKU{
			Name: containerservice.ManagedClusterSKUNameBasic,
			Tier: containerservice.ManagedClusterSKUTierPaid,
		},
		Identity: &containerservice.ManagedClusterIdentity{
			Type: containerservice.ResourceIdentityTypeSystemAssigned,
		},
		Tags: tags,
		ManagedClusterProperties: &containerservice.ManagedClusterProperties{
			KubernetesVersion: to.StringPtr(clusterReqInfo.Version),
			EnableRBAC:        to.BoolPtr(true),
			DNSPrefix:         to.StringPtr(getclusterDNSPrefix(clusterReqInfo.IId.NameId)),
			NodeResourceGroup: to.StringPtr(getclusterNodeResourceGroup(clusterReqInfo.IId.NameId, regionInfo.ResourceGroup, regionInfo.Region)),
			AgentPoolProfiles: &agentPoolProfiles,
			NetworkProfile:    &networkProfile,
			LinuxProfile:      &linuxProfileSSH,
			AddonProfiles:     addonProfiles,
		},
	}
	result, err := managedClustersClient.CreateOrUpdate(ctx, regionInfo.ResourceGroup, clusterReqInfo.IId.NameId, clusterCreateOpts)
	if err != nil {
		return containerservice.ManagedCluster{}, err
	}
	err = result.WaitForCompletionRef(ctx, managedClustersClient.Client)
	if err != nil {
		return containerservice.ManagedCluster{}, err
	}
	newCluster, err := getRawCluster(irs.IID{NameId: clusterReqInfo.IId.NameId}, managedClustersClient, ctx, credentialInfo, regionInfo)
	if err != nil {
		return containerservice.ManagedCluster{}, err
	}
	return newCluster, nil
}

func cleanCluster(clusterName string, managedClustersClient *containerservice.ManagedClustersClient, resourceGroup string, ctx context.Context) error {
	// cluster subresource Clean 현재 없음
	// delete Cluster
	clsuterDeleteResult, err := managedClustersClient.Delete(ctx, resourceGroup, clusterName)
	if err != nil {
		return err
	}
	err = clsuterDeleteResult.WaitForCompletionRef(ctx, managedClustersClient.Client)
	if err != nil {
		return err
	}
	return nil
}

func checkSubnetRequireIPRange(subnet network.Subnet, NodeGroupInfos []irs.NodeGroupInfo) error {
	_, ipnet, err := net.ParseCIDR(*subnet.SubnetPropertiesFormat.AddressPrefix)
	if err != nil {
		return errors.New("invalid Cidr")
	}
	ones, octaBits := ipnet.Mask.Size()
	realRangeBit := float64(octaBits - ones)
	inUsedIPCount := float64(5) // default Azure reserved
	if subnet.SubnetPropertiesFormat.IPConfigurations != nil {
		inUsedIPCount += float64(len(*subnet.SubnetPropertiesFormat.IPConfigurations))
	}
	subnetAvailableCount := math.Pow(2, realRangeBit) - (inUsedIPCount)
	requireIPCount := float64(0)
	for _, NodeGroupInfo := range NodeGroupInfos {
		//PodCount * maxNode + 1
		//float64((NodeGroupInfo.MaxNodeSize + 1) + (maxPodCount * NodeGroupInfo.MaxNodeSize) + 1)
		requireIPCount += float64(maxPodCount * (NodeGroupInfo.MaxNodeSize + 1))
	}
	if subnetAvailableCount < requireIPCount {
		return errors.New(fmt.Sprintf("The subnet id not large enough to support all node pools. Current available IP address space: %d addressed. Required %d addresses.", subnetAvailableCount, requireIPCount))
	}
	return nil
}

func getRawClusterTargetSubnet(networkInfo irs.NetworkInfo, virtualNetworksClient *network.VirtualNetworksClient, ctx context.Context, resourceGroup string) (network.Subnet, error) {
	if len(networkInfo.SubnetIIDs) != 1 {
		return network.Subnet{}, errors.New("The Azure Cluster uses only one subnet in the VPC.")
	}
	if networkInfo.SubnetIIDs[0].NameId == "" && networkInfo.SubnetIIDs[0].SystemId == "" {
		return network.Subnet{}, errors.New("subnet IID within networkInfo is empty")
	}
	targetSubnetName := ""
	if networkInfo.SubnetIIDs[0].NameId != "" {
		targetSubnetName = networkInfo.SubnetIIDs[0].NameId
	} else {
		name, err := getNameById(networkInfo.SubnetIIDs[0].SystemId, AzureSubnet)
		if err != nil {
			return network.Subnet{}, errors.New("subnet IID within networkInfo is invalid ID")
		}
		targetSubnetName = name
	}
	rawVPC, err := getRawVirtualNetwork(networkInfo.VpcIID, virtualNetworksClient, ctx, resourceGroup)
	if err != nil {
		return network.Subnet{}, errors.New("failed get Cluster Vpc And Subnet")
	}
	// first Subnet
	var targetSubnet *network.Subnet
	for _, subnet := range *rawVPC.Subnets {
		if subnet.Name != nil && *subnet.Name == targetSubnetName {
			targetSubnet = &subnet
			break
		}
	}
	if targetSubnet == nil {
		return network.Subnet{}, errors.New(fmt.Sprintf("subnet within that vpc does not exist."))
	}
	return *targetSubnet, nil
}

func createServiceCidrList(networkRange int) ([]string, error) {
	if networkRange == 16 || networkRange == 8 {
		cidrList := make([]string, 256)
		index := 0
		for {
			cidrList[index] = fmt.Sprintf("10.%d.0.0/%d", index, networkRange)
			index++
			if index > 255 {
				break
			}
		}
		index172 := 16
		for {
			cidrList = append(cidrList, fmt.Sprintf("172.%d.0.0/%d", index, networkRange))
			index172++
			if index172 > 29 {
				break
			}
		}
		return cidrList, nil
	}
	return nil, errors.New("invalid networkRange only 8,16")
}

func generatorServiceCidrDNSServiceIP(subnetCidr string) (ServiceCidr string, DNSServiceIP string, DockerBridgeCidr string, err error) {
	//Kubernetes service address range: This parameter is the set of virtual IPs that Kubernetes assigns to internal services in your cluster. You can use any private address range that satisfies the following requirements:
	//
	//Must not be within the virtual network IP address range of your cluster
	//Must not overlap with any other virtual networks with which the cluster virtual network peers
	//Must not overlap with any on-premises IPs
	//Must not be within the ranges 169.254.0.0/16, 172.30.0.0/16, 172.31.0.0/16, or 192.0.2.0/24
	defer func() {
		if r := recover(); r != nil {
			err = errors.New("invalid subnet Cidr")
			ServiceCidr = ""
			DNSServiceIP = ""
		}
	}()
	subnetIP, _, err := net.ParseCIDR(subnetCidr)
	if err != nil {
		return "", "", "", errors.New("invalid subnet Cidr")
	}

	ipSplits := strings.Split(subnetIP.String(), ".")
	if len(ipSplits) != 4 {
		return "", "", "", errors.New("invalid subnet Cidr")
	}
	serviceCidrList, _ := createServiceCidrList(16)
	for _, tempCidr := range serviceCidrList {
		check, _ := overlapCheckCidr(tempCidr, subnetCidr)
		if check {
			ServiceCidr = tempCidr
			break
		}
	}
	// 허용 범위중 서브넷 범위중 가장 큰 ?.?.?.?/9 대역이여도,
	//ServiceCidr = fmt.Sprintf("10.128.0.0/%d", networkRange)
	if ServiceCidr == "" {
		return "", "", "", errors.New("The cidr on the current subnet and the serviceCidr on the cb-spider are overlapping. The areas of ServiceCidr checking for superposition in cb-spider are 10.0.0.0/16 to 10.255.0.0/16, and 172.16.0.0 to 172.29.0")
	}
	newip, _, err := net.ParseCIDR(ServiceCidr)
	netipSplits := strings.Split(newip.String(), ".")
	if len(ipSplits) != 4 {
		return "", "", "", errors.New("invalid subnet Cidr")
	}
	netipSplits[3] = "10"
	DNSServiceIP = strings.Join(netipSplits, ".")
	if ServiceCidr == "172.17.0.1/16" {
		DockerBridgeCidr = "172.18.0.1/16"
	} else {
		DockerBridgeCidr = "172.17.0.1/16"
	}
	return ServiceCidr, DNSServiceIP, DockerBridgeCidr, nil
}

func generatorNetworkProfile(ClusterInfo irs.ClusterInfo, targetSubnet network.Subnet) (containerservice.NetworkProfile, error) {
	err := checkSubnetRequireIPRange(targetSubnet, ClusterInfo.NodeGroupList)
	if err != nil {
		return containerservice.NetworkProfile{}, errors.New(fmt.Sprintf("failed get vpc err = %s", err.Error()))
	}

	ServiceCidr, DNSServiceIP, DockerBridgeCidr, err := generatorServiceCidrDNSServiceIP(*targetSubnet.AddressPrefix)
	if err != nil {
		return containerservice.NetworkProfile{}, errors.New(fmt.Sprintf("failed calculate ServiceCidr, DNSServiceIP err = %s", err.Error()))
	}

	return containerservice.NetworkProfile{
		LoadBalancerSku:  containerservice.LoadBalancerSkuStandard,
		NetworkPlugin:    containerservice.NetworkPluginAzure,
		NetworkPolicy:    containerservice.NetworkPolicyAzure,
		ServiceCidr:      to.StringPtr(ServiceCidr),
		DNSServiceIP:     to.StringPtr(DNSServiceIP),
		DockerBridgeCidr: to.StringPtr(DockerBridgeCidr),
	}, nil

}

func generatorClusterTags(sshKeyName string) (map[string]*string, error) {
	tags := make(map[string]*string)
	nowTime := strconv.FormatInt(time.Now().Unix(), 10)
	tags["sshkey"] = to.StringPtr(sshKeyName)
	tags["createdAt"] = to.StringPtr(nowTime)
	return tags, nil
}

func getSSHKeyIIDByNodeGroups(NodeGroupInfos []irs.NodeGroupInfo) (irs.IID, error) {
	var key *irs.IID
	for _, nodeGroup := range NodeGroupInfos {
		if key == nil && !(nodeGroup.KeyPairIID.NameId == "" && nodeGroup.KeyPairIID.SystemId == "") {
			key = &nodeGroup.KeyPairIID
			break
		}
	}
	if key == nil {
		return irs.IID{}, errors.New("failed find SSHKey IID By nodeGroups")
	}
	return *key, nil
}

func generatePreparedAddonProfiles() map[string]*containerservice.ManagedClusterAddonProfile {
	return map[string]*containerservice.ManagedClusterAddonProfile{
		"httpApplicationRouting": &containerservice.ManagedClusterAddonProfile{
			Enabled: to.BoolPtr(true),
		},
	}
}

func generateManagedClusterLinuxProfileSSH(clusterReqInfo irs.ClusterInfo, sshPublicKeysClient *compute.SSHPublicKeysClient, resourceGroup string, ctx context.Context) (containerservice.LinuxProfile, compute.SSHPublicKeyResource, error) {
	sshkeyId, err := getSSHKeyIIDByNodeGroups(clusterReqInfo.NodeGroupList)
	if err != nil {
		return containerservice.LinuxProfile{}, compute.SSHPublicKeyResource{}, err
	}
	key, err := GetRawKey(sshkeyId, resourceGroup, sshPublicKeysClient, ctx)
	if err != nil {
		return containerservice.LinuxProfile{}, compute.SSHPublicKeyResource{}, errors.New(fmt.Sprintf("failed get ssh Key, err = %s", err.Error()))
	}
	linuxProfile := containerservice.LinuxProfile{
		AdminUsername: to.StringPtr(CBVMUser),
		SSH: &containerservice.SSHConfiguration{
			PublicKeys: &[]containerservice.SSHPublicKey{
				{
					KeyData: key.PublicKey,
				},
			},
		},
	}
	return linuxProfile, key, nil
}

func generateAgentPoolProfileList(info irs.ClusterInfo, targetSubnet network.Subnet) ([]containerservice.ManagedClusterAgentPoolProfile, error) {
	agentPoolProfiles := make([]containerservice.ManagedClusterAgentPoolProfile, len(info.NodeGroupList))
	for i, nodeGroupInfo := range info.NodeGroupList {
		agentPoolProfile, err := generateAgentPoolProfile(nodeGroupInfo, targetSubnet)
		if err != nil {
			return make([]containerservice.ManagedClusterAgentPoolProfile, 0), err
		}
		agentPoolProfiles[i] = agentPoolProfile
	}
	return agentPoolProfiles, nil
}

func generateAgentPoolProfileProperties(nodeGroupInfo irs.NodeGroupInfo, subnet network.Subnet) (containerservice.ManagedClusterAgentPoolProfileProperties, error) {
	var nodeOSDiskSize *int32
	if nodeGroupInfo.RootDiskSize == "" || nodeGroupInfo.RootDiskSize == "default" {
		nodeOSDiskSize = nil
	} else {
		osDiskSize, err := strconv.Atoi(nodeGroupInfo.RootDiskSize)
		if err != nil {
			return containerservice.ManagedClusterAgentPoolProfileProperties{}, errors.New("invalid NodeGroup RootDiskSize")
		}
		nodeOSDiskSize = to.Int32Ptr(int32(osDiskSize))
	}
	agentPoolProfileProperties := containerservice.ManagedClusterAgentPoolProfileProperties{
		// Name:         to.StringPtr(nodeGroupInfo.IId.NameId),
		Count:        to.Int32Ptr(int32(nodeGroupInfo.DesiredNodeSize)),
		MinCount:     to.Int32Ptr(int32(nodeGroupInfo.MinNodeSize)),
		MaxCount:     to.Int32Ptr(int32(nodeGroupInfo.MaxNodeSize)),
		VMSize:       to.StringPtr(nodeGroupInfo.VMSpecName),
		OsDiskSizeGB: nodeOSDiskSize,
		OsType:       containerservice.OSTypeLinux,
		Type:         containerservice.AgentPoolTypeVirtualMachineScaleSets,
		MaxPods:      to.Int32Ptr(maxPodCount),
		Mode:         containerservice.AgentPoolModeSystem, // User? System?
		// https://learn.microsoft.com/en-us/azure/availability-zones/az-overview#availability-zones
		// Azure availability zones : To ensure resiliency, a minimum of three separate availability zones are present in all availability zone-enabled regions.
		//AvailabilityZones:  &[]string{"1", "2", "3"},
		AvailabilityZones:  &[]string{"1"},
		EnableNodePublicIP: to.BoolPtr(true),
		EnableAutoScaling:  to.BoolPtr(nodeGroupInfo.OnAutoScaling),
		// MinCount가 있으려면 true 여야함
		VnetSubnetID: subnet.ID,
	}
	return agentPoolProfileProperties, nil
}

func generateAgentPoolProfile(nodeGroupInfo irs.NodeGroupInfo, subnet network.Subnet) (containerservice.ManagedClusterAgentPoolProfile, error) {
	var nodeOSDiskSize *int32
	if nodeGroupInfo.RootDiskSize == "" || nodeGroupInfo.RootDiskSize == "default" {
		nodeOSDiskSize = nil
	} else {
		osDiskSize, err := strconv.Atoi(nodeGroupInfo.RootDiskSize)
		if err != nil {
			return containerservice.ManagedClusterAgentPoolProfile{}, errors.New("invalid NodeGroup RootDiskSize")
		}
		nodeOSDiskSize = to.Int32Ptr(int32(osDiskSize))
	}
	agentPoolProfile := containerservice.ManagedClusterAgentPoolProfile{
		Name:         to.StringPtr(nodeGroupInfo.IId.NameId),
		Count:        to.Int32Ptr(int32(nodeGroupInfo.DesiredNodeSize)),
		MinCount:     to.Int32Ptr(int32(nodeGroupInfo.MinNodeSize)),
		MaxCount:     to.Int32Ptr(int32(nodeGroupInfo.MaxNodeSize)),
		VMSize:       to.StringPtr(nodeGroupInfo.VMSpecName),
		OsDiskSizeGB: nodeOSDiskSize,
		OsType:       containerservice.OSTypeLinux,
		Type:         containerservice.AgentPoolTypeVirtualMachineScaleSets,
		MaxPods:      to.Int32Ptr(maxPodCount),
		Mode:         containerservice.AgentPoolModeSystem, // User? System?
		// https://learn.microsoft.com/en-us/azure/availability-zones/az-overview#availability-zones
		// Azure availability zones : To ensure resiliency, a minimum of three separate availability zones are present in all availability zone-enabled regions.
		//AvailabilityZones:  &[]string{"1", "2", "3"},
		AvailabilityZones:  &[]string{"1"},
		EnableNodePublicIP: to.BoolPtr(true),
		EnableAutoScaling:  to.BoolPtr(nodeGroupInfo.OnAutoScaling),
		// MinCount가 있으려면 true 여야함
		VnetSubnetID: subnet.ID,
	}
	if !nodeGroupInfo.OnAutoScaling {
		agentPoolProfile.MinCount = nil
		agentPoolProfile.MaxCount = nil
	}
	return agentPoolProfile, nil
}

// attachNodeGroupSSHSecurityGroup All Nodegroup in Cluster SSH Allow SecurityGroup
//func attachNodeGroupSSHSecurityGroup(cluster containerservice.ManagedCluster, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, subnetClient *network.SubnetsClient, securityGroupsClient *network.SecurityGroupsClient, securityRulesClient *network.SecurityRulesClient, ctx context.Context, clusterResourceGroup string) error {
//	if cluster.ManagedClusterProperties.NodeResourceGroup == nil {
//		return errors.New("invalid cluster Resource")
//	}
//
//	nodePoolPairList, err := getRawNodePoolPairList(cluster, agentPoolsClient, virtualMachineScaleSetsClient, ctx, clusterResourceGroup)
//	if err != nil {
//		return err
//	}
//	if nodePoolPairList == nil || len(nodePoolPairList) == 0 {
//		return errors.New("not exist NodeGroup")
//	}
//
//	// clusterManagedResourceGroup 클러스터 생성시 만들어지는 리소스 그룹
//	clusterManagedResourceGroup := *cluster.ManagedClusterProperties.NodeResourceGroup
//
//	primarySubnetList, err := getPrimarySubnetListByNodePoolList(nodePoolPairList, subnetClient, ctx, clusterManagedResourceGroup)
//	if err != nil {
//		return err
//	}
//	rawSGList, err := getRawSecurityGroupListBySubnetList(primarySubnetList, securityGroupsClient, ctx, clusterManagedResourceGroup)
//	if err != nil {
//		return err
//	}
//	for _, sg := range rawSGList {
//		err = addSSHRule(sg, securityRulesClient, ctx, clusterManagedResourceGroup)
//		if err != nil {
//			return err
//		}
//	}
//	return nil
//}

// getPrimarySubnetIDByVMScaleSet To find PrimarySubnet in VirtualMachineScaleSet
//func getPrimarySubnetIDByVMScaleSet(set compute.VirtualMachineScaleSet) (subnetId string, err error) {
//	defer func() {
//		if r := recover(); r != nil {
//			subnetId = ""
//			err = errors.New("not Exist Subnet")
//		}
//	}()
//	if set.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations != nil && len(*set.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations) > 0 {
//		scaleSetNetworkInterfaceConfigurations := *set.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations
//		var primaryNetworkInterface *compute.VirtualMachineScaleSetNetworkConfiguration
//		for _, scaleSetNetworkInterfaceConfiguration := range scaleSetNetworkInterfaceConfigurations {
//			if *scaleSetNetworkInterfaceConfiguration.Primary {
//				primaryNetworkInterface = &scaleSetNetworkInterfaceConfiguration
//				break
//			}
//		}
//		if primaryNetworkInterface == nil {
//			return "", errors.New("not Exist Subnet")
//		}
//		subnetID := ""
//		for _, IPConfigurations := range *primaryNetworkInterface.IPConfigurations {
//			if *IPConfigurations.Primary {
//				subnetID = *IPConfigurations.Subnet.ID
//			}
//		}
//		if subnetID == "" {
//			return "", errors.New("not Exist Subnet")
//		}
//		return subnetID, nil
//	}
//	return "", errors.New("not Exist Subnet")
//}

// getPrimarySubnetIDListByNodePoolList To find PrimarySubnetList in NodePoolPair, NodePoolPair is a pair of AgentPool, virtualMachineScaleSet
//func getPrimarySubnetIDListByNodePoolList(nodePoolPairList []NodePoolPair) ([]string, error) {
//	var subnetIds []string
//	for _, nodePoolPair := range nodePoolPairList {
//		subnetId, err := getPrimarySubnetIDByVMScaleSet(nodePoolPair.virtualMachineScaleSet)
//		if err == nil {
//			subnetIds = append(subnetIds, subnetId)
//		}
//	}
//	keys := make(map[string]bool)
//	filterdSubnetIds := []string{}
//	for _, subnetId := range subnetIds {
//		if _, saveValue := keys[subnetId]; !saveValue {
//			keys[subnetId] = true
//			filterdSubnetIds = append(filterdSubnetIds, subnetId)
//		}
//	}
//	return filterdSubnetIds, nil
//}

// getRawSubnetList To get Azure RawSubnet By Id
//func getRawSubnetList(subnetIdList []string, subnetClient *network.SubnetsClient, ctx context.Context, resourceGroup string) ([]network.Subnet, error) {
//	vpcName := ""
//	for _, subnetId := range subnetIdList {
//		vpcname, VPCerr := GetVPCNameById(subnetId)
//		if VPCerr == nil {
//			vpcName = vpcname
//			break
//		}
//
//	}
//	if resourceGroup == "" || vpcName == "" {
//		return nil, errors.New("not found Subnet")
//	}
//	result, err := subnetClient.List(ctx, resourceGroup, vpcName)
//	if err != nil {
//		return nil, errors.New("not found Subnet")
//	}
//	keys := make(map[string]bool)
//	for _, subnetId := range subnetIdList {
//		keys[subnetId] = true
//	}
//	subnets := []network.Subnet{}
//	for _, subnet := range result.Values() {
//		_, exist := keys[*subnet.ID]
//		if exist {
//			subnets = append(subnets, subnet)
//		}
//	}
//	return subnets, nil
//}

// getPrimarySubnetListByNodePoolList Composite function of getPrimarySubnetIDListByNodePoolList and getRawSubnetList
//func getPrimarySubnetListByNodePoolList(nodePoolPairList []NodePoolPair, subnetClient *network.SubnetsClient, ctx context.Context, resourceGroup string) ([]network.Subnet, error) {
//	sunetIDs, err := getPrimarySubnetIDListByNodePoolList(nodePoolPairList)
//	if err != nil {
//		return nil, err
//	}
//	return getRawSubnetList(sunetIDs, subnetClient, ctx, resourceGroup)
//}

// getSecurityGroupIdBySubnet to find SecurityGroupId by Subnet
//func getSecurityGroupIdBySubnet(subnet network.Subnet) (string, error) {
//	if subnet.SubnetPropertiesFormat.NetworkSecurityGroup != nil {
//		return *subnet.SubnetPropertiesFormat.NetworkSecurityGroup.ID, nil
//	}
//	return "", errors.New("not found NetworkSecurityGroup")
//}

// getRawSecurityGroupListBySubnet to get Azure SecurityGroup By SubnetList
//func getRawSecurityGroupListBySubnetList(subnets []network.Subnet, SecurityGroupsClient *network.SecurityGroupsClient, ctx context.Context, resourceGroup string) ([]network.SecurityGroup, error) {
//	listResult, err := SecurityGroupsClient.List(ctx, resourceGroup)
//	if err != nil {
//		return nil, err
//	}
//	sgMap := make(map[string]network.SecurityGroup)
//	for _, sg := range listResult.Values() {
//		sgName, err := getNameById(*sg.ID, AzureSecurityGroups)
//		if err == nil {
//			sgMap[sgName] = sg
//		}
//
//	}
//	sgList := []network.SecurityGroup{}
//	for _, sb := range subnets {
//		sgId, err := getSecurityGroupIdBySubnet(sb)
//		if err == nil {
//			sgName, _ := getNameById(sgId, AzureSecurityGroups)
//			sg, exist := sgMap[sgName]
//			if exist {
//				sgList = append(sgList, sg)
//			}
//		}
//	}
//	return sgList, nil
//}

// addSSHRule attach SSH Rule
//
//	func addSSHRule(sg network.SecurityGroup, securityRulesClient *network.SecurityRulesClient, ctx context.Context, resourceGroup string) error {
//		existSSHRuele := false
//		inboundPriority := initPriority
//		for _, sgrule := range *sg.SecurityRules {
//			if sgrule.Protocol == network.SecurityRuleProtocolTCP {
//				if sgrule.Direction == network.SecurityRuleDirectionInbound && *sgrule.DestinationPortRange == "22" {
//					existSSHRuele = true
//				}
//				inboundPriority = int(*sgrule.Priority)
//			}
//		}
//		if existSSHRuele {
//			return nil
//		}
//		inboundPriority = inboundPriority + 1
//		sshsgrule := network.SecurityRule{
//			SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
//				SourceAddressPrefix:      to.StringPtr("*"),
//				SourcePortRange:          to.StringPtr("*"),
//				DestinationAddressPrefix: to.StringPtr("*"),
//				DestinationPortRange:     to.StringPtr("22"),
//				Protocol:                 network.SecurityRuleProtocolTCP,
//				Access:                   network.SecurityRuleAccessAllow,
//				Priority:                 to.Int32Ptr(int32(inboundPriority)),
//				Direction:                network.SecurityRuleDirectionInbound,
//			},
//		}
//		result, err := securityRulesClient.CreateOrUpdate(ctx, resourceGroup, *sg.Name, "cluster-node-ssh-by-cbspider", sshsgrule)
//		if err != nil {
//			return err
//		}
//		err = result.WaitForCompletionRef(ctx, securityRulesClient.Client)
//		return err
//	}

func getSecurityGroupIdForVirtualMachineScaleSet(virtualMachineScaleSet compute.VirtualMachineScaleSet) (securityGroupId string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("failed get securityGroup in VirtualMachineScaleSet err = %s", err.Error()))
			securityGroupId = ""
		}
	}()
	if virtualMachineScaleSet.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations != nil {
		networkInterfaceConfigurations := *virtualMachineScaleSet.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations
		var targetSecurityGroupId *string
		for _, niconfig := range networkInterfaceConfigurations {
			if targetSecurityGroupId == nil && *niconfig.Primary {
				targetSecurityGroupId = niconfig.NetworkSecurityGroup.ID
				break
			}
		}
		if targetSecurityGroupId == nil {
			return "", errors.New(fmt.Sprintf("failed get securityGroup in VirtualMachineScaleSet err = %s", err.Error()))
		}
		return *targetSecurityGroupId, nil
	}
	return "", errors.New(fmt.Sprintf("failed get securityGroup in VirtualMachineScaleSet err = %s", err.Error()))
}

func getNextSecurityGroupRulePriority(sourceSecurity network.SecurityGroup) (inboundPriority int, outboundPriority int, err error) {
	inboundPriority = initPriority
	outboundPriority = initPriority
	for _, sgrule := range *sourceSecurity.SecurityRules {
		if sgrule.Direction == network.SecurityRuleDirectionInbound {
			inboundPriority = int(*sgrule.Priority)
		}
		if sgrule.Direction == network.SecurityRuleDirectionOutbound {
			outboundPriority = int(*sgrule.Priority)
		}
	}
	inboundPriority = inboundPriority + 1
	outboundPriority = outboundPriority + 1
	return inboundPriority, outboundPriority, nil
}

func sliceSecurityGroupRuleINAndOUT(copySourceSGRules []network.SecurityRule) (inboundRules []network.SecurityRule, outboundRules []network.SecurityRule, err error) {
	inboundRules = make([]network.SecurityRule, 0)
	outboundRules = make([]network.SecurityRule, 0)
	for _, sourcesgrule := range copySourceSGRules {
		if sourcesgrule.Direction == network.SecurityRuleDirectionInbound {
			inboundRules = append(inboundRules, sourcesgrule)
		} else if sourcesgrule.Direction == network.SecurityRuleDirectionOutbound {
			outboundRules = append(outboundRules, sourcesgrule)
		} else {
			return make([]network.SecurityRule, 0), make([]network.SecurityRule, 0), errors.New("invalid SecurityRules")
		}

	}
	return inboundRules, outboundRules, nil
}

func applySecurityGroupList(sourceSecurity network.SecurityGroup, targetSecurityGroupList []network.SecurityGroup, SecurityRulesClient *network.SecurityRulesClient, ctx context.Context) error {
	sourceSGRules := *sourceSecurity.SecurityRules
	for _, targetsg := range targetSecurityGroupList {
		copySourceSGRules := sourceSGRules
		sgresourceGroup, _ := getResourceGroupById(*targetsg.ID)
		inboundPriority, outboundPriority, err := getNextSecurityGroupRulePriority(targetsg)
		if err != nil {
			return err
		}
		inboundsgRules, outboundsgRules, err := sliceSecurityGroupRuleINAndOUT(copySourceSGRules)
		if err != nil {
			return err
		}
		for i, inboundsgRule := range inboundsgRules {
			update := inboundsgRule
			update.Priority = to.Int32Ptr(int32(inboundPriority + i))
			updateResult, err := SecurityRulesClient.CreateOrUpdate(ctx, sgresourceGroup, *targetsg.Name, *inboundsgRule.Name, update)
			if err != nil {
				return err
			}
			err = updateResult.WaitForCompletionRef(ctx, SecurityRulesClient.Client)
			if err != nil {
				return err
			}
		}
		for i, outboundsgRule := range outboundsgRules {
			update := outboundsgRule
			update.Priority = to.Int32Ptr(int32(outboundPriority + i))
			updateResult, err := SecurityRulesClient.CreateOrUpdate(ctx, sgresourceGroup, *targetsg.Name, *outboundsgRule.Name, update)
			if err != nil {
				return err
			}
			err = updateResult.WaitForCompletionRef(ctx, SecurityRulesClient.Client)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getClusterAgentPoolSecurityGroupId(cluster containerservice.ManagedCluster, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, ctx context.Context) ([]string, error) {
	filteredNodePoolPairList, err := getRawNodePoolPairList(cluster, agentPoolsClient, virtualMachineScaleSetsClient, ctx)
	if err != nil {
		return make([]string, 0), errors.New(fmt.Sprintf("failed apply securityGroup err = %s", err.Error()))
	}
	sgmap := make(map[string]string)
	for _, poolPair := range filteredNodePoolPairList {
		sgid, err := getSecurityGroupIdForVirtualMachineScaleSet(poolPair.virtualMachineScaleSet)
		if err != nil {
			return make([]string, 0), errors.New(fmt.Sprintf("failed apply securityGroup err = %s", err.Error()))
		}
		sgName, _ := getNameById(sgid, AzureSecurityGroups)
		_, exist := sgmap[sgName]
		if !exist {
			sgmap[sgName] = sgid
		}
	}
	targetSecurityGroupIdsList := make([]string, 0, len(sgmap))
	for _, sg := range sgmap {
		targetSecurityGroupIdsList = append(targetSecurityGroupIdsList, sg)
	}
	return targetSecurityGroupIdsList, nil
}

func getClusterAgentPoolSecurityGroup(cluster containerservice.ManagedCluster, agentPoolsClient *containerservice.AgentPoolsClient, securityGroupsClient *network.SecurityGroupsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, ctx context.Context) ([]network.SecurityGroup, error) {
	filteredNodePoolPairList, err := getRawNodePoolPairList(cluster, agentPoolsClient, virtualMachineScaleSetsClient, ctx)
	if err != nil {
		return make([]network.SecurityGroup, 0), errors.New(fmt.Sprintf("failed apply securityGroup err = %s", err.Error()))
	}

	sgmap := make(map[string]network.SecurityGroup)
	for _, poolPair := range filteredNodePoolPairList {
		sgid, err := getSecurityGroupIdForVirtualMachineScaleSet(poolPair.virtualMachineScaleSet)
		if err != nil {
			return make([]network.SecurityGroup, 0), errors.New(fmt.Sprintf("failed apply securityGroup err = %s", err.Error()))
		}
		sgName, _ := getNameById(sgid, AzureSecurityGroups)
		sgresourceGroup, _ := getResourceGroupById(sgid)
		_, exist := sgmap[sgName]
		if !exist {
			sg, err := getRawSecurityGroup(irs.IID{NameId: sgName}, securityGroupsClient, ctx, sgresourceGroup)
			if err != nil {
				return make([]network.SecurityGroup, 0), errors.New(fmt.Sprintf("failed apply securityGroup err = %s", err.Error()))
			}
			sgmap[sgName] = *sg
		}
	}
	targetSecurityGroupList := make([]network.SecurityGroup, 0, len(sgmap))
	for _, sg := range sgmap {
		targetSecurityGroupList = append(targetSecurityGroupList, sg)
	}
	return targetSecurityGroupList, nil
}

func applySecurityGroup(cluster containerservice.ManagedCluster, securityGroupIID irs.IID, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, securityGroupsClient *network.SecurityGroupsClient, securityRulesClient *network.SecurityRulesClient, ctx context.Context) error {
	clusterResourceGroup, err := getResourceGroupById(*cluster.ID)
	if err != nil {
		return errors.New(fmt.Sprintf("failed get clusterResourceGroup err = %s", err.Error()))
	}
	sourceSecurityGroup, err := getRawSecurityGroup(securityGroupIID, securityGroupsClient, ctx, clusterResourceGroup)
	if err != nil {
		return errors.New(fmt.Sprintf("failed apply securityGroup err = %s", err.Error()))
	}
	targetSecurityGroupList, err := getClusterAgentPoolSecurityGroup(cluster, agentPoolsClient, securityGroupsClient, virtualMachineScaleSetsClient, ctx)
	if err != nil {
		return errors.New(fmt.Sprintf("failed get securityGroup by AgentPool err = %s", err.Error()))
	}
	err = applySecurityGroupList(*sourceSecurityGroup, targetSecurityGroupList, securityRulesClient, ctx)
	return err
	// virtualMachineScaleSet에서, VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations에서 Primary 확인 후, NetworkSecurityGroup를 가져와, 수정
}

func checkNodeGroupScaleValid(desiredNodeSize int, minNodeSize int, maxNodeSize int) error {
	if minNodeSize < 1 {
		return errors.New("The MinNodeSize of the node group must be at least 1 autoScaling.")
	}
	if minNodeSize > desiredNodeSize {
		return errors.New("The MinNodeSize for a node group cannot be greater than the DesiredNodeSize.")
	}
	if minNodeSize > maxNodeSize {
		return errors.New("The MinNodeSize for a node group cannot be greater than the MaxNodeSize.")
	}
	if desiredNodeSize > maxNodeSize {
		return errors.New("The DesiredNodeSize for a node group cannot be greater than the DesiredNodeSize.")
	}
	return nil
}

func changeNodeGroupScaling(cluster containerservice.ManagedCluster, nodeGroupIID irs.IID, desiredNodeSize int, minNodeSize int, maxNodeSize int, managedClustersClient *containerservice.ManagedClustersClient, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context) (irs.NodeGroupInfo, error) {
	err := checkNodeGroupScaleValid(desiredNodeSize, minNodeSize, maxNodeSize)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed scalingChange agentPool err = %s", err.Error()))
	}
	if nodeGroupIID.NameId == "" && nodeGroupIID.SystemId == "" {
		return irs.NodeGroupInfo{}, errors.New("failed scalingChange agentPool err = invalid NodeGroup NameId")
	}
	agentPools, err := agentPoolsClient.List(ctx, region.ResourceGroup, *cluster.Name)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed scalingChange agentPool err = %s", err.Error()))
	}
	var targetAgentPool *containerservice.AgentPool
	for _, agentPool := range agentPools.Values() {
		if targetAgentPool == nil {
			if nodeGroupIID.NameId == "" {
				if *agentPool.ID == nodeGroupIID.SystemId {
					targetAgentPool = &agentPool
					break
				}
			} else {
				if *agentPool.Name == nodeGroupIID.NameId {
					targetAgentPool = &agentPool
					break
				}
			}
		}
	}
	if targetAgentPool == nil {
		return irs.NodeGroupInfo{}, errors.New("failed scalingChange agentPool err = not Exist NodeGroup")
	}
	if !*targetAgentPool.EnableAutoScaling {
		return irs.NodeGroupInfo{}, errors.New("failed scalingChange agentPool err = AutoScaling is disabled for the node group. DesiredNodeSize, MinNodeSize, and MaxNodeSize can be set when autoScaling is enabled")
	}
	updateAgentPool := *targetAgentPool
	updateAgentPool.MinCount = to.Int32Ptr(int32(minNodeSize))
	updateAgentPool.MaxCount = to.Int32Ptr(int32(maxNodeSize))
	updateAgentPool.Count = to.Int32Ptr(int32(desiredNodeSize))
	updateResult, err := agentPoolsClient.CreateOrUpdate(ctx, region.ResourceGroup, *cluster.Name, *targetAgentPool.Name, updateAgentPool)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed scalingChange agentPool err = %s", err.Error()))
	}
	err = updateResult.WaitForCompletionRef(ctx, agentPoolsClient.Client)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed scalingChange agentPool err = %s", err.Error()))
	}
	newCluster, err := getRawCluster(irs.IID{NameId: *cluster.Name}, managedClustersClient, ctx, credentialInfo, region)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed get agentPool err = %s", err.Error()))
	}
	nodeGroupInfo, err := getNodeGroupInfoSpecifiedNodePool(newCluster, nodeGroupIID.NameId, agentPoolsClient, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, ctx)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed get agentPool err = %s", err.Error()))
	}
	return nodeGroupInfo, nil
}

func autoScalingChange(cluster containerservice.ManagedCluster, nodeGroupIID irs.IID, autoScalingSet bool, agentPoolsClient *containerservice.AgentPoolsClient, region idrv.RegionInfo, ctx context.Context) error {
	// exist Check
	if nodeGroupIID.NameId == "" && nodeGroupIID.SystemId == "" {
		return errors.New("failed autoScalingChange agentPool err = invalid NodeGroup NameId")
	}
	agentPools, err := agentPoolsClient.List(ctx, region.ResourceGroup, *cluster.Name)
	if err != nil {
		return errors.New(fmt.Sprintf("failed autoScalingChange agentPool err = %s", err.Error()))
	}
	var targetAgentPool *containerservice.AgentPool
	for _, agentPool := range agentPools.Values() {
		if targetAgentPool == nil {
			if nodeGroupIID.NameId == "" {
				if *agentPool.ID == nodeGroupIID.SystemId {
					targetAgentPool = &agentPool
					break
				}
			} else {
				if *agentPool.Name == nodeGroupIID.NameId {
					targetAgentPool = &agentPool
					break
				}
			}
		}
	}
	if targetAgentPool == nil {
		return errors.New("failed autoScalingChange agentPool err = not Exist NodeGroup")
	}
	if *targetAgentPool.EnableAutoScaling == autoScalingSet {
		return errors.New("failed autoScalingChange agentPool err = already autoScaling status Equal")
	}
	updateAgentPool := *targetAgentPool
	if *updateAgentPool.ProvisioningState != "Succeeded" {
		return errors.New(fmt.Sprintf("failed autoScalingChange agentPool err = The status of the Agent Pool is currently %s. You cannot change the Agent Pool at this time.", *updateAgentPool.ProvisioningState))
	}
	if !autoScalingSet {
		// False
		updateAgentPool.MinCount = nil
		updateAgentPool.MaxCount = nil
	} else {
		// TODO autoScale 시, Min, Max 값 필요
		updateAgentPool.MinCount = to.Int32Ptr(1)
		updateAgentPool.MaxCount = updateAgentPool.Count
	}
	updateAgentPool.EnableAutoScaling = to.BoolPtr(autoScalingSet)
	updateResult, err := agentPoolsClient.CreateOrUpdate(ctx, region.ResourceGroup, *cluster.Name, *targetAgentPool.Name, updateAgentPool)
	if err != nil {
		return errors.New(fmt.Sprintf("failed autoScalingChange agentPool err = %s", err.Error()))
	}
	err = updateResult.WaitForCompletionRef(ctx, agentPoolsClient.Client)
	if err != nil {
		return errors.New(fmt.Sprintf("failed autoScalingChange agentPool err = %s", err.Error()))
	}
	return nil
}

func deleteNodeGroup(cluster containerservice.ManagedCluster, nodeGroupIID irs.IID, agentPoolsClient *containerservice.AgentPoolsClient, region idrv.RegionInfo, ctx context.Context) error {
	// exist Check
	if nodeGroupIID.NameId == "" && nodeGroupIID.SystemId == "" {
		return errors.New("failed remove agentPool err = invalid NodeGroup NameId")
	}
	agentPools, err := agentPoolsClient.List(ctx, region.ResourceGroup, *cluster.Name)
	if err != nil {
		return errors.New(fmt.Sprintf("failed remove agentPool err = %s", err.Error()))
	}
	existNodeGroup := false
	for _, agentPool := range agentPools.Values() {
		if nodeGroupIID.NameId == "" {
			if *agentPool.ID == nodeGroupIID.SystemId {
				existNodeGroup = true
				break
			}
		} else {
			if *agentPool.Name == nodeGroupIID.NameId {
				existNodeGroup = true
				break
			}
		}
	}
	if !existNodeGroup {
		return errors.New("failed remove agentPool err = not Exist NodeGroup")
	}
	deleteResult, err := agentPoolsClient.Delete(ctx, region.ResourceGroup, *cluster.Name, nodeGroupIID.NameId)
	if err != nil {
		return errors.New(fmt.Sprintf("failed remove agentPool err = %s", err.Error()))
	}
	err = deleteResult.WaitForCompletionRef(ctx, agentPoolsClient.Client)
	if err != nil {
		return errors.New(fmt.Sprintf("failed remove agentPool err = %s", err.Error()))
	}
	return nil
}
func addNodeGroupPool(cluster containerservice.ManagedCluster, nodeGroup irs.NodeGroupInfo, managedClustersClient *containerservice.ManagedClustersClient, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, subnetClient *network.SubnetsClient, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context) (irs.NodeGroupInfo, error) {
	// exist Check
	if nodeGroup.IId.NameId == "" && nodeGroup.IId.SystemId == "" {
		return irs.NodeGroupInfo{}, errors.New("failed add agentPool err = invalid NodeGroup NameId")
	}
	// SSh same Check
	if nodeGroup.KeyPairIID.NameId == "" && nodeGroup.KeyPairIID.SystemId == "" {
		return irs.NodeGroupInfo{}, errors.New("failed add agentPool err = sshkey in the Azure Cluster NodeGroup is empty.")
	}
	sshkeyName, sshKeyExist := cluster.Tags["sshkey"]
	if !sshKeyExist || sshkeyName == nil {
		return irs.NodeGroupInfo{}, errors.New("failed add agentPool err = sshkey in the Azure Cluster is empty.")
	}
	if nodeGroup.KeyPairIID.NameId != "" {
		if nodeGroup.KeyPairIID.NameId != *sshkeyName {
			return irs.NodeGroupInfo{}, errors.New("The SSHkey in the Azure Cluster NodeGroup must all be the same")
		}
	} else {
		clusterSShKeyId := GetSshKeyIdByName(credentialInfo, region, *sshkeyName)
		if nodeGroup.KeyPairIID.SystemId != clusterSShKeyId {
			return irs.NodeGroupInfo{}, errors.New("The SSHkey in the Azure Cluster NodeGroup must all be the same")
		}
	}
	agentPools, err := agentPoolsClient.List(ctx, region.ResourceGroup, *cluster.Name)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed add agentPool err = %s", err.Error()))
	}
	existNodeGroup := false
	for _, agentPool := range agentPools.Values() {
		if nodeGroup.IId.NameId == "" {
			if *agentPool.ID == nodeGroup.IId.SystemId {
				existNodeGroup = true
				break
			}
		} else {
			if *agentPool.Name == nodeGroup.IId.NameId {
				existNodeGroup = true
				break
			}
		}
	}
	if existNodeGroup {
		return irs.NodeGroupInfo{}, errors.New("failed add agentPool err = already Exist NodeGroup")
	}
	subnetId, err := getSubnetIdByAgentPoolProfiles(*cluster.AgentPoolProfiles)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed add agentPool err = %s", err.Error()))
	}
	subnetName, err := getNameById(subnetId, AzureSubnet)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed add agentPool err = %s", err.Error()))
	}
	vpcName, err := getNameById(subnetId, AzureVirtualNetworks)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed add agentPool err = %s", err.Error()))
	}
	subnet, err := subnetClient.Get(ctx, region.ResourceGroup, vpcName, subnetName, "")
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed addNodeGroupPool err = %s", err.Error()))
	}
	// Add AgentPoolProfiles
	agentPoolProfileProperties, err := generateAgentPoolProfileProperties(nodeGroup, subnet)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed add agentPool err = %s", err.Error()))
	}
	result, err := agentPoolsClient.CreateOrUpdate(ctx, region.ResourceGroup, *cluster.Name, nodeGroup.IId.NameId, containerservice.AgentPool{ManagedClusterAgentPoolProfileProperties: &agentPoolProfileProperties})
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed add agentPool err = %s", err.Error()))
	}
	err = result.WaitForCompletionRef(ctx, agentPoolsClient.Client)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed add agentPool err = %s", err.Error()))
	}
	newCluster, err := getRawCluster(irs.IID{NameId: *cluster.Name}, managedClustersClient, ctx, credentialInfo, region)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed get agentPool err = %s", err.Error()))
	}
	nodeGroupInfo, err := getNodeGroupInfoSpecifiedNodePool(newCluster, nodeGroup.IId.NameId, agentPoolsClient, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, ctx)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed get agentPool err = %s", err.Error()))
	}
	return nodeGroupInfo, nil
}
