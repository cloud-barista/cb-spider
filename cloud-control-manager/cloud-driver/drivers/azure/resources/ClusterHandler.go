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
	"gopkg.in/yaml.v2"
	"math"
	"net"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	maxPodCount          = 110
	OwnerClusterKey      = "ownerCluster"
	ScaleSetOwnerKey     = "aks-managed-poolName"
	ClusterNodeSSHKeyKey = "sshkey"
	ClusterAdminKey      = "clusterAdmin"
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
	err := createCluster(clusterReqInfo, ac.VirtualNetworksClient, ac.ManagedClustersClient, ac.VirtualMachineSizesClient, ac.SSHPublicKeysClient, ac.CredentialInfo, ac.Region, ac.Ctx)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create Cluster. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.ClusterInfo{}, createErr
	}
	defer func() {
		if createErr != nil {
			cleanCluster(clusterReqInfo.IId.NameId, ac.ManagedClustersClient, ac.Region.ResourceGroup, ac.Ctx)
		}
	}()
	baseSecurityGroup, err := waitingClusterBaseSecurityGroup(irs.IID{NameId: clusterReqInfo.IId.NameId}, ac.ManagedClustersClient, ac.SecurityGroupsClient, ac.Ctx, ac.CredentialInfo, ac.Region)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create Cluster. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.ClusterInfo{}, createErr
	}
	for _, sg := range clusterReqInfo.Network.SecurityGroupIIDs {
		err = applySecurityGroup(irs.IID{NameId: clusterReqInfo.IId.NameId}, irs.IID{NameId: sg.NameId}, baseSecurityGroup, ac.ManagedClustersClient, ac.SecurityGroupsClient, ac.SecurityRulesClient, ac.Ctx, ac.CredentialInfo, ac.Region)
		if err != nil {
			createErr = errors.New(fmt.Sprintf("Failed to Create Cluster. err = %s", err))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.ClusterInfo{}, createErr
		}
	}
	cluster, err := getRawCluster(clusterReqInfo.IId, ac.ManagedClustersClient, ac.Ctx, ac.CredentialInfo, ac.Region)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create Cluster. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.ClusterInfo{}, createErr
	}
	info, err = setterClusterInfo(cluster, ac.ManagedClustersClient, ac.SecurityGroupsClient, ac.VirtualNetworksClient, ac.AgentPoolsClient, ac.VirtualMachineScaleSetsClient, ac.VirtualMachineScaleSetVMsClient, ac.CredentialInfo, ac.Region, ac.Ctx)
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
	listInfo, err = setterClusterInfoList(clusterList.Values(), ac.ManagedClustersClient, ac.SecurityGroupsClient, ac.VirtualNetworksClient, ac.AgentPoolsClient, ac.VirtualMachineScaleSetsClient, ac.VirtualMachineScaleSetVMsClient, ac.CredentialInfo, ac.Region, ac.Ctx)
	if err != nil {
		getErr = errors.New(fmt.Sprintf("Failed to List Cluster. err = %s", err))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return make([]*irs.ClusterInfo, 0), getErr
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
	info, err = setterClusterInfo(cluster, ac.ManagedClustersClient, ac.SecurityGroupsClient, ac.VirtualNetworksClient, ac.AgentPoolsClient, ac.VirtualMachineScaleSetsClient, ac.VirtualMachineScaleSetVMsClient, ac.CredentialInfo, ac.Region, ac.Ctx)
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
	nodeGroupInfo, err := addNodeGroupPool(cluster, nodeGroupReqInfo, ac.AgentPoolsClient, ac.VirtualMachineScaleSetsClient, ac.VirtualMachineScaleSetVMsClient, ac.SubnetClient, ac.CredentialInfo, ac.Region, ac.Ctx)
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
	err = upgradeCluter(cluster, newVersion, ac.ManagedClustersClient, ac.AgentPoolsClient, ac.VirtualMachineScaleSetsClient, ac.Ctx, ac.Region)
	if err != nil {
		upgradeErr = errors.New(fmt.Sprintf("Failed to Upgrade Cluster. err = %s", err))
		cblogger.Error(upgradeErr.Error())
		LoggingError(hiscallInfo, upgradeErr)
		return irs.ClusterInfo{}, upgradeErr
	}
	cluster, err = getRawCluster(clusterIID, ac.ManagedClustersClient, ac.Ctx, ac.CredentialInfo, ac.Region)
	if err != nil {
		upgradeErr = errors.New(fmt.Sprintf("Failed to Upgrade Cluster. err = %s", err))
		cblogger.Error(upgradeErr.Error())
		LoggingError(hiscallInfo, upgradeErr)
		return irs.ClusterInfo{}, upgradeErr
	}
	info, err = setterClusterInfo(cluster, ac.ManagedClustersClient, ac.SecurityGroupsClient, ac.VirtualNetworksClient, ac.AgentPoolsClient, ac.VirtualMachineScaleSetsClient, ac.VirtualMachineScaleSetVMsClient, ac.CredentialInfo, ac.Region, ac.Ctx)
	if err != nil {
		upgradeErr = errors.New(fmt.Sprintf("Failed to Upgrade Cluster. err = %s", err))
		cblogger.Error(upgradeErr.Error())
		LoggingError(hiscallInfo, upgradeErr)
		return irs.ClusterInfo{}, upgradeErr
	}
	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func checkUpgradeCluster(cluster containerservice.ManagedCluster, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, ctx context.Context) error {
	if getClusterStatus(cluster) != irs.ClusterActive {
		return errors.New("failed Upgrade Cluster err = Cluster's status must be Active")

	}
	nodePoolPairList, err := getRawNodePoolPairList(cluster, agentPoolsClient, virtualMachineScaleSetsClient, ctx)
	if err != nil {
		return errors.New(fmt.Sprintf("failed Upgrade Cluster err = failed to get information for agentPool and virtualMachineScaleSetts while checking for upgradeability err = %s", err))
	}
	check := true
	for _, nodePoolPair := range nodePoolPairList {
		if getNodeInfoStatus(nodePoolPair.AgentPool, nodePoolPair.virtualMachineScaleSet) != irs.NodeGroupActive {
			check = false
		}
	}
	if !check {
		return errors.New("failed Upgrade Cluster err = NodeGroup's status must be Active")

	}
	return nil
}

func upgradeCluter(cluster containerservice.ManagedCluster, newVersion string, managedClustersClient *containerservice.ManagedClustersClient, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, ctx context.Context, region idrv.RegionInfo) error {
	err := checkUpgradeCluster(cluster, agentPoolsClient, virtualMachineScaleSetsClient, ctx)
	if err != nil {
		return err
	}
	updateCluster := cluster
	updateCluster.KubernetesVersion = to.StringPtr(newVersion)
	_, err = managedClustersClient.CreateOrUpdate(ctx, region.ResourceGroup, *cluster.Name, updateCluster)
	if err != nil {
		return err
	}
	//err = upgradeResult.WaitForCompletionRef(ctx, managedClustersClient.Client)
	//if err != nil {
	//	return err
	//}
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

type ClusterInfoWithError struct {
	ClusterInfo irs.ClusterInfo
	err         error
}

func setterClusterInfoWithCancel(cluster containerservice.ManagedCluster, managedClustersClient *containerservice.ManagedClustersClient, securityGroupsClient *network.SecurityGroupsClient, virtualNetworksClient *network.VirtualNetworksClient, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context, cancelCtx context.Context) (irs.ClusterInfo, error) {
	done := make(chan ClusterInfoWithError)

	go func() {
		clusterInfo, err := setterClusterInfo(cluster, managedClustersClient, securityGroupsClient, virtualNetworksClient, agentPoolsClient, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, credentialInfo, region, ctx)
		done <- ClusterInfoWithError{
			ClusterInfo: clusterInfo,
			err:         err,
		}
	}()
	select {
	case vmInfoWithErrorDone := <-done:
		return vmInfoWithErrorDone.ClusterInfo, vmInfoWithErrorDone.err
	case <-cancelCtx.Done():
		return irs.ClusterInfo{}, nil
	}
}

func setterClusterInfoList(clusterList []containerservice.ManagedCluster, managedClustersClient *containerservice.ManagedClustersClient, securityGroupsClient *network.SecurityGroupsClient, virtualNetworksClient *network.VirtualNetworksClient, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context) (clusterInfoList []*irs.ClusterInfo, err error) {
	clusterListCount := len(clusterList)

	clusterInfos := make([]*irs.ClusterInfo, clusterListCount)
	if clusterListCount == 0 {
		return clusterInfos, nil
	}
	var wg sync.WaitGroup
	cancelCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var globalErr error

	for i, cluster := range clusterList {
		wg.Add(1)
		index := i
		copyCluster := cluster
		go func() {
			defer wg.Done()
			info, err := setterClusterInfoWithCancel(copyCluster, managedClustersClient, securityGroupsClient, virtualNetworksClient, agentPoolsClient, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, credentialInfo, region, ctx, cancelCtx)
			if err != nil {
				cancel()
				if globalErr == nil {
					globalErr = err
				}
			}
			clusterInfos[index] = &info
		}()
	}
	wg.Wait()
	if globalErr != nil {
		return nil, globalErr
	}

	return clusterInfos, nil
}

func setterClusterInfo(cluster containerservice.ManagedCluster, managedClustersClient *containerservice.ManagedClustersClient, securityGroupsClient *network.SecurityGroupsClient, virtualNetworksClient *network.VirtualNetworksClient, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context) (clusterInfo irs.ClusterInfo, err error) {
	clusterInfo.IId = irs.IID{*cluster.Name, *cluster.ID}
	if cluster.ManagedClusterProperties != nil {
		// Version
		if cluster.ManagedClusterProperties.KubernetesVersion != nil {
			clusterInfo.Version = *cluster.ManagedClusterProperties.KubernetesVersion
		}
		// NetworkInfo - Network Configuration AzureCNI
		networkInfo, err := getNetworkInfo(cluster, securityGroupsClient, virtualNetworksClient, credentialInfo, region, ctx)
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
		accessInfo, err := getClusterAccessInfo(cluster, managedClustersClient, ctx)
		if err == nil {
			clusterInfo.AccessInfo = accessInfo
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
			sshkey, sshKeyExist := tags[ClusterNodeSSHKeyKey]
			if sshKeyExist && sshkey != nil {
				keyValues = append(keyValues, irs.KeyValue{Key: ClusterNodeSSHKeyKey, Value: *sshkey})
			}
		}
		clusterInfo.KeyValueList = keyValues
	}

	return clusterInfo, nil
}

func getClusterStatus(cluster containerservice.ManagedCluster) (resultStatus irs.ClusterStatus) {
	defer func() {
		if r := recover(); r != nil {
			resultStatus = irs.ClusterInactive
		}
	}()
	resultStatus = irs.ClusterInactive
	if cluster.ProvisioningState == nil || cluster.PowerState == nil {
		return resultStatus
	}
	provisioningState := *cluster.ProvisioningState
	powerState := cluster.PowerState.Code
	if powerState != containerservice.CodeRunning {
		resultStatus = irs.ClusterInactive
	}
	if provisioningState == "Creating" {
		resultStatus = irs.ClusterCreating
	}
	if provisioningState == "Succeeded" {
		resultStatus = irs.ClusterActive
	}
	if provisioningState == "Deleting" {
		resultStatus = irs.ClusterDeleting
	}
	if provisioningState == "Updating" || provisioningState == "Upgrading" {
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

func getNetworkInfo(cluster containerservice.ManagedCluster, securityGroupsClient *network.SecurityGroupsClient, virtualNetworksClient *network.VirtualNetworksClient, CredentialInfo idrv.CredentialInfo, Region idrv.RegionInfo, ctx context.Context) (info irs.NetworkInfo, err error) {
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
			securityGroupList, err := getClusterSecurityGroup(cluster, securityGroupsClient, ctx)
			if err != nil {
				return irs.NetworkInfo{}, errors.New("failed get cluster SecurityGroups")
			}
			sgIIDs := make([]irs.IID, len(securityGroupList))
			for i, sg := range securityGroupList {
				sgIIDs[i] = irs.IID{
					NameId:   *sg.Name,
					SystemId: *sg.ID,
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
	clusterNodeSShkey, err := getClusterSSHKey(cluster)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed get convertNodeGroupInfo err = %s", err))
	}

	for _, nodePoolPair := range nodePoolPairList {
		agentPool := nodePoolPair.AgentPool
		if *agentPool.Name == nodePoolName {
			info := convertNodePairToNodeInfo(nodePoolPair, clusterNodeSShkey, virtualMachineScaleSetVMsClient, resourceGroupManagedK8s, ctx)
			nodeInfo = &info
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

	clusterNodeSShkey, err := getClusterSSHKey(cluster)
	if err != nil {
		return make([]irs.NodeGroupInfo, 0), errors.New(fmt.Sprintf("failed get convertNodeGroupInfo err = %s", err))
	}
	for i, nodePoolPair := range nodePoolPairList {
		nodeInfoGroupList[i] = convertNodePairToNodeInfo(nodePoolPair, clusterNodeSShkey, virtualMachineScaleSetVMsClient, resourceGroupManagedK8s, ctx)
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

func getNodeInfoStatus(agentPool containerservice.AgentPool, virtualMachineScaleSet compute.VirtualMachineScaleSet) (status irs.NodeGroupStatus) {
	defer func() {
		if r := recover(); r != nil {
			status = irs.NodeGroupInactive
		}
	}()
	if reflect.ValueOf(virtualMachineScaleSet.ProvisioningState).IsNil() || reflect.ValueOf(agentPool.ProvisioningState).IsNil() {
		return irs.NodeGroupInactive
	}
	if *virtualMachineScaleSet.ProvisioningState == "Succeeded" && *agentPool.ProvisioningState == "Succeeded" {
		return irs.NodeGroupActive
	}
	if *virtualMachineScaleSet.ProvisioningState == "Creating" || *agentPool.ProvisioningState == "Creating" {
		return irs.NodeGroupCreating
	}
	if *virtualMachineScaleSet.ProvisioningState == "Updating" || *agentPool.ProvisioningState == "Updating" ||
		*virtualMachineScaleSet.ProvisioningState == "Upgrading" || *agentPool.ProvisioningState == "Upgrading" ||
		*virtualMachineScaleSet.ProvisioningState == "Scaling" || *agentPool.ProvisioningState == "Scaling" {
		return irs.NodeGroupUpdating
	}
	if *virtualMachineScaleSet.ProvisioningState == "Deleting" || *agentPool.ProvisioningState == "Deleting" {
		return irs.NodeGroupDeleting
	}
	return irs.NodeGroupInactive
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
		// image지정 제공 안함(Azure에서 기본적으로 매핑.)
		if nodeGroup.ImageIID.NameId != "" || nodeGroup.ImageIID.SystemId != "" {
			return errors.New("The Cluster in Azure does not provide Image Designation. Please remove the name of the image and try")
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
					return errors.New("the SSHkey in the Azure Cluster NodeGroup must all be the same")
				}
			} else if nodeGroup.KeyPairIID.NameId != "" {
				if nodeGroup.KeyPairIID.NameId != sshKeyIID.NameId {
					return errors.New("the SSHkey in the Azure Cluster NodeGroup must all be the same")
				}
			} else if nodeGroup.KeyPairIID.SystemId != "" {
				if nodeGroup.KeyPairIID.SystemId != sshKeyIID.SystemId {
					return errors.New("the SSHkey in the Azure Cluster NodeGroup must all be the same")
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
	// network
	if err != nil {
		return errors.New(fmt.Sprintf("Failed Validation Check NodeGroup. err = %s", err.Error()))
	}
	return nil
}

func createCluster(clusterReqInfo irs.ClusterInfo, virtualNetworksClient *network.VirtualNetworksClient, managedClustersClient *containerservice.ManagedClustersClient, virtualMachineSizesClient *compute.VirtualMachineSizesClient, sshPublicKeysClient *compute.SSHPublicKeysClient, credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo, ctx context.Context) error {
	// 사전 확인
	err := checkValidationCreateCluster(clusterReqInfo, virtualMachineSizesClient, regionInfo, ctx)
	if err != nil {
		return err
	}
	targetSubnet, err := getRawClusterTargetSubnet(clusterReqInfo.Network, virtualNetworksClient, ctx, regionInfo.ResourceGroup)
	if err != nil {
		return err
	}
	// agentPoolProfiles
	agentPoolProfiles, err := generateAgentPoolProfileList(clusterReqInfo, targetSubnet)
	if err != nil {
		return err
	}
	// networkProfile
	networkProfile, err := generatorNetworkProfile(clusterReqInfo, targetSubnet)
	if err != nil {
		return err
	}
	// mapping ssh
	linuxProfileSSH, sshKey, err := generateManagedClusterLinuxProfileSSH(clusterReqInfo, sshPublicKeysClient, regionInfo.ResourceGroup, ctx)
	if err != nil {
		return err
	}
	tags, err := generatorClusterTags(*sshKey.Name, clusterReqInfo.IId.NameId)
	if err != nil {
		return err
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
	_, err = managedClustersClient.CreateOrUpdate(ctx, regionInfo.ResourceGroup, clusterReqInfo.IId.NameId, clusterCreateOpts)
	if err != nil {
		return err
	}
	//err = result.WaitForCompletionRef(ctx, managedClustersClient.Client)
	//if err != nil {
	//	return containerservice.ManagedCluster{}, err
	//}
	//newCluster, err := getRawCluster(irs.IID{NameId: clusterReqInfo.IId.NameId}, managedClustersClient, ctx, credentialInfo, regionInfo)
	//if err != nil {
	//	return containerservice.ManagedCluster{}, err
	//}
	//return newCluster, nil
	return nil
}

func cleanCluster(clusterName string, managedClustersClient *containerservice.ManagedClustersClient, resourceGroup string, ctx context.Context) error {
	// cluster subresource Clean 현재 없음
	// delete Cluster
	_, err := managedClustersClient.Delete(ctx, resourceGroup, clusterName)
	if err != nil {
		return err
	}
	//err = clsuterDeleteResult.WaitForCompletionRef(ctx, managedClustersClient.Client)
	//if err != nil {
	//	return err
	//}
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
		requireIPCount += float64((maxPodCount + 1) * (NodeGroupInfo.MaxNodeSize))
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
		return "", "", "", errors.New("the cidr on the current subnet and the serviceCidr on the cb-spider are overlapping. The areas of ServiceCidr checking for superposition in cb-spider are 10.0.0.0/16 to 10.255.0.0/16, and 172.16.0.0 to 172.29.0")
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

func generatorClusterTags(sshKeyName string, clusterName string) (map[string]*string, error) {
	tags := make(map[string]*string)
	nowTime := strconv.FormatInt(time.Now().Unix(), 10)
	tags[ClusterNodeSSHKeyKey] = to.StringPtr(sshKeyName)
	tags["createdAt"] = to.StringPtr(nowTime)
	tags[OwnerClusterKey] = to.StringPtr(clusterName)
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
		inboundsgRules, outboundsgRules, err := sliceSecurityGroupRuleINAndOUT(copySourceSGRules)
		if err != nil {
			return err
		}
		for _, inboundsgRule := range inboundsgRules {
			update := inboundsgRule
			update.Priority = inboundsgRule.Priority
			if *update.Priority >= 500 {
				// AKS baseRule 회피
				update.Priority = to.Int32Ptr(*inboundsgRule.Priority + 100)
			}
			updateResult, err := SecurityRulesClient.CreateOrUpdate(ctx, sgresourceGroup, *targetsg.Name, *inboundsgRule.Name, update)
			if err != nil {
				return err
			}
			err = updateResult.WaitForCompletionRef(ctx, SecurityRulesClient.Client)
			if err != nil {
				return err
			}
		}
		for _, outboundsgRule := range outboundsgRules {
			update := outboundsgRule
			update.Priority = outboundsgRule.Priority
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

func applySecurityGroup(clusterIID irs.IID, sourceSecurityGroupIID irs.IID, clusterBaseSecurityGroup network.SecurityGroup, managedClustersClient *containerservice.ManagedClustersClient, securityGroupsClient *network.SecurityGroupsClient, securityRulesClient *network.SecurityRulesClient, ctx context.Context, credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo) error {
	//clusterResourceGroup, err := getResourceGroupById(*cluster.ID)
	//if err != nil {
	//	return errors.New(fmt.Sprintf("failed get clusterResourceGroup err = %s", err.Error()))
	//}
	sourceSecurityGroup, err := getRawSecurityGroup(sourceSecurityGroupIID, securityGroupsClient, ctx, regionInfo.ResourceGroup)
	if err != nil {
		return errors.New(fmt.Sprintf("failed apply securityGroup err = %s", err.Error()))
	}
	//baseSecurityGroup, err := waitingClusterBaseSecurityGroup(clusterIID, managedClustersClient, securityGroupsClient, ctx, credentialInfo, regionInfo)
	//targetSecurityGroupList, err := getClusterAgentPoolSecurityGroup(cluster, agentPoolsClient, securityGroupsClient, virtualMachineScaleSetsClient, ctx)
	//if err != nil {
	//	return errors.New(fmt.Sprintf("failed get BaseSecurityGroup by Cluster err = %s", err.Error()))
	//}
	err = applySecurityGroupList(*sourceSecurityGroup, []network.SecurityGroup{clusterBaseSecurityGroup}, securityRulesClient, ctx)
	return err
	// virtualMachineScaleSet에서, VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations에서 Primary 확인 후, NetworkSecurityGroup를 가져와, 수정
}

func checkAutoScaleModeNodeGroupScaleValid(desiredNodeSize int, minNodeSize int, maxNodeSize int) error {
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
func checkmanualScaleModeNodeGroupScaleValid(minNodeSize int, maxNodeSize int) error {
	if minNodeSize != 0 || maxNodeSize != 0 {
		return errors.New("If it is not autoScaleMode, you cannot specify minNodeSize and maxNodeSize.")
	}
	return nil
}

func menualScaleModechangeNodeGroupScaling(cluster containerservice.ManagedCluster, agentPool containerservice.AgentPool, desiredNodeSize int, minNodeSize int, maxNodeSize int, managedClustersClient *containerservice.ManagedClustersClient, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context) (irs.NodeGroupInfo, error) {
	err := checkmanualScaleModeNodeGroupScaleValid(minNodeSize, maxNodeSize)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed scalingChange agentPool err = %s", err.Error()))
	}
	updateAgentPool := agentPool
	updateAgentPool.Count = to.Int32Ptr(int32(desiredNodeSize))
	_, err = agentPoolsClient.CreateOrUpdate(ctx, region.ResourceGroup, *cluster.Name, *agentPool.Name, updateAgentPool)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed scalingChange agentPool err = %s", err.Error()))
	}
	//err = updateResult.WaitForCompletionRef(ctx, agentPoolsClient.Client)
	//if err != nil {
	//	return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed scalingChange agentPool err = %s", err.Error()))
	//}
	newCluster, err := getRawCluster(irs.IID{NameId: *cluster.Name}, managedClustersClient, ctx, credentialInfo, region)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed get agentPool err = %s", err.Error()))
	}
	nodeGroupInfo, err := getNodeGroupInfoSpecifiedNodePool(newCluster, *agentPool.Name, agentPoolsClient, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, ctx)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed get agentPool err = %s", err.Error()))
	}
	nodeGroupInfo.Status = irs.NodeGroupUpdating
	return nodeGroupInfo, nil
}

func autoScaleModechangeNodeGroupScaling(cluster containerservice.ManagedCluster, agentPool containerservice.AgentPool, desiredNodeSize int, minNodeSize int, maxNodeSize int, managedClustersClient *containerservice.ManagedClustersClient, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context) (irs.NodeGroupInfo, error) {
	err := checkAutoScaleModeNodeGroupScaleValid(desiredNodeSize, minNodeSize, maxNodeSize)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed scalingChange agentPool err = %s", err.Error()))
	}
	updateAgentPool := agentPool
	updateAgentPool.MinCount = to.Int32Ptr(int32(minNodeSize))
	updateAgentPool.MaxCount = to.Int32Ptr(int32(maxNodeSize))
	updateAgentPool.Count = to.Int32Ptr(int32(desiredNodeSize))
	_, err = agentPoolsClient.CreateOrUpdate(ctx, region.ResourceGroup, *cluster.Name, *agentPool.Name, updateAgentPool)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed scalingChange agentPool err = %s", err.Error()))
	}

	//err = updateResult.WaitForCompletionRef(ctx, agentPoolsClient.Client)
	//if err != nil {
	//	return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed scalingChange agentPool err = %s", err.Error()))
	//}
	newCluster, err := getRawCluster(irs.IID{NameId: *cluster.Name}, managedClustersClient, ctx, credentialInfo, region)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed get agentPool err = %s", err.Error()))
	}
	nodeGroupInfo, err := getNodeGroupInfoSpecifiedNodePool(newCluster, *agentPool.Name, agentPoolsClient, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, ctx)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed get agentPool err = %s", err.Error()))
	}
	nodeGroupInfo.Status = irs.NodeGroupUpdating
	return nodeGroupInfo, nil
}

func changeNodeGroupScaling(cluster containerservice.ManagedCluster, nodeGroupIID irs.IID, desiredNodeSize int, minNodeSize int, maxNodeSize int, managedClustersClient *containerservice.ManagedClustersClient, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context) (irs.NodeGroupInfo, error) {
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
	if *targetAgentPool.EnableAutoScaling {
		// AutoScale
		return autoScaleModechangeNodeGroupScaling(cluster, *targetAgentPool, desiredNodeSize, minNodeSize, maxNodeSize, managedClustersClient, agentPoolsClient, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, credentialInfo, region, ctx)
	} else {
		// MenualScale
		return menualScaleModechangeNodeGroupScaling(cluster, *targetAgentPool, desiredNodeSize, minNodeSize, maxNodeSize, managedClustersClient, agentPoolsClient, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, credentialInfo, region, ctx)
	}
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
	_, err = agentPoolsClient.CreateOrUpdate(ctx, region.ResourceGroup, *cluster.Name, *targetAgentPool.Name, updateAgentPool)
	if err != nil {
		return errors.New(fmt.Sprintf("failed autoScalingChange agentPool err = %s", err.Error()))
	}
	//err = updateResult.WaitForCompletionRef(ctx, agentPoolsClient.Client)
	//if err != nil {
	//	return errors.New(fmt.Sprintf("failed autoScalingChange agentPool err = %s", err.Error()))
	//}
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
	_, err = agentPoolsClient.Delete(ctx, region.ResourceGroup, *cluster.Name, nodeGroupIID.NameId)
	if err != nil {
		return errors.New(fmt.Sprintf("failed remove agentPool err = %s", err.Error()))
	}
	//err = deleteResult.WaitForCompletionRef(ctx, agentPoolsClient.Client)
	//if err != nil {
	//	return errors.New(fmt.Sprintf("failed remove agentPool err = %s", err.Error()))
	//}
	return nil
}
func addNodeGroupPool(cluster containerservice.ManagedCluster, nodeGroup irs.NodeGroupInfo, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, subnetClient *network.SubnetsClient, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context) (irs.NodeGroupInfo, error) {
	// exist Check
	if nodeGroup.IId.NameId == "" && nodeGroup.IId.SystemId == "" {
		return irs.NodeGroupInfo{}, errors.New("failed add agentPool err = invalid NodeGroup NameId")
	}
	// SSh same Check
	if nodeGroup.KeyPairIID.NameId == "" && nodeGroup.KeyPairIID.SystemId == "" {
		return irs.NodeGroupInfo{}, errors.New("failed add agentPool err = sshkey in the Azure Cluster NodeGroup is empty.")
	}
	clusterNodeSSHkey, err := getClusterSSHKey(cluster)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New("failed add agentPool err = sshkey in the Azure Cluster is empty.")
	}

	if nodeGroup.KeyPairIID.NameId != "" {
		if nodeGroup.KeyPairIID.NameId != clusterNodeSSHkey.NameId {
			return irs.NodeGroupInfo{}, errors.New("The SSHkey in the Azure Cluster NodeGroup must all be the same")
		}
	} else {
		clusterSShKeyId := GetSshKeyIdByName(credentialInfo, region, clusterNodeSSHkey.NameId)
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
	_, err = agentPoolsClient.CreateOrUpdate(ctx, region.ResourceGroup, *cluster.Name, nodeGroup.IId.NameId, containerservice.AgentPool{ManagedClusterAgentPoolProfileProperties: &agentPoolProfileProperties})
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed add agentPool err = %s", err.Error()))
	}
	nodePoolPair, err := waitingSpecifiedNodePoolPair(cluster, nodeGroup.IId.NameId, agentPoolsClient, virtualMachineScaleSetsClient, ctx)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed add agentPool err = %s", err.Error()))
	}
	clusterSSHKey, err := getClusterSSHKey(cluster)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed add agentPool err = %s", err.Error()))
	}
	info := convertNodePairToNodeInfo(nodePoolPair, clusterSSHKey, virtualMachineScaleSetVMsClient, *cluster.NodeResourceGroup, ctx)
	//err = result.WaitForCompletionRef(ctx, agentPoolsClient.Client)
	//if err != nil {
	//	return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed add agentPool err = %s", err.Error()))
	//}
	//newCluster, err := getRawCluster(irs.IID{NameId: *cluster.Name}, managedClustersClient, ctx, credentialInfo, region)
	//if err != nil {
	//	return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed get agentPool err = %s", err.Error()))
	//}
	//nodeGroupInfo, err := getNodeGroupInfoSpecifiedNodePool(newCluster, nodeGroup.IId.NameId, agentPoolsClient, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, ctx)
	//if err != nil {
	//	return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed get agentPool err = %s", err.Error()))
	//}
	return info, nil
}

func getClusterSecurityGroup(cluster containerservice.ManagedCluster, securityGroupsClient *network.SecurityGroupsClient, ctx context.Context) ([]network.SecurityGroup, error) {
	if cluster.ManagedClusterProperties.NodeResourceGroup == nil {
		return make([]network.SecurityGroup, 0), errors.New("failed get Cluster Managed ResourceGroup")
	}
	securityGroupList, err := securityGroupsClient.List(ctx, *cluster.ManagedClusterProperties.NodeResourceGroup)
	if err != nil {
		return make([]network.SecurityGroup, 0), err
	}
	return securityGroupList.Values(), nil
}

func waitingClusterBaseSecurityGroup(createdClusterIID irs.IID, managedClustersClient *containerservice.ManagedClustersClient, securityGroupsClient *network.SecurityGroupsClient, ctx context.Context, credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo) (network.SecurityGroup, error) {
	// exist Cluster
	apiCallCount := 0
	maxAPICallCount := 240
	var waitingErr error
	var targetRawCluster *containerservice.ManagedCluster
	for {
		rawCluster, err := getRawCluster(createdClusterIID, managedClustersClient, ctx, credentialInfo, regionInfo)
		if err == nil && rawCluster.NodeResourceGroup != nil {
			targetRawCluster = &rawCluster
			break
		}
		apiCallCount++
		if apiCallCount >= maxAPICallCount {
			waitingErr = errors.New("failed get Cluster err = The maximum number of verification requests has been exceeded while waiting for the creation of that resource")
			break
		}
		time.Sleep(1 * time.Second)
	}

	if waitingErr != nil {
		return network.SecurityGroup{}, waitingErr
	}
	// exist basicSecurity in clusterResourceGroup
	apiCallCount = 0
	clusterManagedResourceGroup := *targetRawCluster.NodeResourceGroup
	if clusterManagedResourceGroup == "" {
		return network.SecurityGroup{}, errors.New("failed get Cluster Managed ResourceGroup err = Invalid value of NodeResourceGroup for cluster")
	}
	var baseSecurityGroup network.SecurityGroup
	for {
		securityGroupList, err := securityGroupsClient.ListAll(ctx)
		if err == nil && len(securityGroupList.Values()) > 0 {
			// securityGroupList get Success
			sgCheck := false
			for _, sg := range securityGroupList.Values() {
				if sg.Tags != nil {
					val, exist := sg.Tags[OwnerClusterKey]
					if exist && val != nil && *val == *targetRawCluster.Name {
						baseSecurityGroup = sg
						sgCheck = true
						break
					}
				}
			}
			if sgCheck {
				// base securityGroup Success
				break
			}

		}
		apiCallCount++
		if apiCallCount >= maxAPICallCount {
			waitingErr = errors.New("failed get Cluster BaseSecurityGroup err = The maximum number of verification requests has been exceeded while waiting for the creation of that resource")
			break
		}
		time.Sleep(4 * time.Second)
	}
	if waitingErr != nil {
		return network.SecurityGroup{}, waitingErr
	}
	// check ClusterBaseRule..
	apiCallCount = 0
	for {
		baseRuleCheck := 0
		sg, err := securityGroupsClient.Get(ctx, clusterManagedResourceGroup, *baseSecurityGroup.Name, "")
		if err == nil {
			for _, rule := range *sg.SecurityRules {
				if *rule.Priority == 500 && *rule.DestinationPortRange == "80" {
					baseRuleCheck++
				}
				if *rule.Priority == 501 && *rule.DestinationPortRange == "443" {
					baseRuleCheck++
				}
			}
		}
		if baseRuleCheck == 2 {
			break
		}
		apiCallCount++
		if apiCallCount >= maxAPICallCount {
			waitingErr = errors.New("failed wait creating BaseRule in Cluster BaseSecurityGroup err = The maximum number of verification requests has been exceeded while waiting for the creation of that resource")
			break
		}
		time.Sleep(4 * time.Second)
	}
	if waitingErr != nil {
		return network.SecurityGroup{}, waitingErr
	}
	return baseSecurityGroup, nil
	// exist basicSecurity
}

func convertNodePairToNodeInfo(nodePoolPair NodePoolPair, clusterCommonSSHKey irs.IID, virtualMachineScaleSetVMsClient *compute.VirtualMachineScaleSetVMsClient, resourceGroupManagedK8s string, ctx context.Context) irs.NodeGroupInfo {
	scaleSet := nodePoolPair.virtualMachineScaleSet
	agentPool := nodePoolPair.AgentPool
	rootDiskType := ""
	if reflect.ValueOf(scaleSet.VirtualMachineProfile.StorageProfile.OsDisk.ManagedDisk.StorageAccountType).String() != "" {
		rootDiskType = GetVMDiskInfoType(scaleSet.VirtualMachineProfile.StorageProfile.OsDisk.ManagedDisk.StorageAccountType)
	}
	VMSpecName := ""
	if !reflect.ValueOf(scaleSet.Sku.Name).IsNil() {
		VMSpecName = *scaleSet.Sku.Name
	}
	RootDiskSize := ""
	if !reflect.ValueOf(scaleSet.VirtualMachineProfile.StorageProfile.OsDisk.DiskSizeGB).IsNil() {
		RootDiskSize = strconv.Itoa(int(*scaleSet.VirtualMachineProfile.StorageProfile.OsDisk.DiskSizeGB))
	}
	OnAutoScaling := false
	if !reflect.ValueOf(agentPool.EnableAutoScaling).IsNil() {
		OnAutoScaling = *agentPool.EnableAutoScaling
	}
	DesiredNodeSize := 0
	if !reflect.ValueOf(agentPool.Count).IsNil() {
		DesiredNodeSize = int(*agentPool.Count)
	}
	MinNodeSize := 0
	if !reflect.ValueOf(agentPool.MinCount).IsNil() {
		MinNodeSize = int(*agentPool.MinCount)
	}
	MaxNodeSize := 0
	if !reflect.ValueOf(agentPool.MaxCount).IsNil() {
		MaxNodeSize = int(*agentPool.MaxCount)
	}
	imageName := ""
	if !reflect.ValueOf(scaleSet.VirtualMachineProfile.StorageProfile.ImageReference.ID).IsNil() {
		imageName = *scaleSet.VirtualMachineProfile.StorageProfile.ImageReference.ID
	}
	nodeInfo := irs.NodeGroupInfo{
		IId: irs.IID{
			NameId:   *agentPool.Name,
			SystemId: *agentPool.ID,
		},
		ImageIID: irs.IID{
			SystemId: imageName,
			NameId:   imageName,
		},
		VMSpecName:      VMSpecName,
		RootDiskType:    rootDiskType,
		RootDiskSize:    RootDiskSize,
		KeyPairIID:      clusterCommonSSHKey,
		Status:          getNodeInfoStatus(agentPool, scaleSet),
		OnAutoScaling:   OnAutoScaling,
		DesiredNodeSize: DesiredNodeSize,
		MinNodeSize:     MinNodeSize,
		MaxNodeSize:     MaxNodeSize,
	}
	vmIIds := make([]irs.IID, 0)
	vms, err := virtualMachineScaleSetVMsClient.List(ctx, resourceGroupManagedK8s, *scaleSet.Name, "", "", "")
	if err == nil {
		for _, vm := range vms.Values() {
			vmIIds = append(vmIIds, irs.IID{*vm.Name, *vm.ID})
		}
	}
	nodeInfo.Nodes = vmIIds
	return nodeInfo
}

func getClusterSSHKey(cluster containerservice.ManagedCluster) (irs.IID, error) {
	sshkeyName, sshKeyExist := cluster.Tags[ClusterNodeSSHKeyKey]
	keyPairIID := irs.IID{}

	if sshKeyExist && sshkeyName != nil {
		keyPairIID.NameId = *sshkeyName
		clusterSubscriptionsById, subscriptionsErr := getSubscriptionsById(*cluster.ID)
		clusterResourceGroup, err := getResourceGroupById(*cluster.ID)
		if err != nil {
			return irs.IID{}, errors.New("failed get Cluster Node SSHKey err = invalid Cluster ID")
		}
		if err == nil && subscriptionsErr == nil {
			keyPairIID.SystemId = GetSshKeyIdByName(idrv.CredentialInfo{SubscriptionId: clusterSubscriptionsById}, idrv.RegionInfo{
				ResourceGroup: clusterResourceGroup,
			}, *sshkeyName)
		}
	}
	return keyPairIID, nil
}

func waitingSpecifiedNodePoolPair(cluster containerservice.ManagedCluster, agentPoolName string, agentPoolsClient *containerservice.AgentPoolsClient, virtualMachineScaleSetsClient *compute.VirtualMachineScaleSetsClient, ctx context.Context) (NodePoolPair, error) {
	clusterResourceGroup, err := getResourceGroupById(*cluster.ID)
	if err != nil {
		return NodePoolPair{}, errors.New(fmt.Sprintf("failed get clusterResourceGroup err = %s", err.Error()))
	}
	if cluster.NodeResourceGroup == nil {
		return NodePoolPair{}, errors.New(fmt.Sprintf("failed get Cluster Managed ResourceGroup err = Invalid value of NodeResourceGroup for cluster"))
	}
	clusterManagedResourceGroup := ""
	clusterManagedResourceGroup = *cluster.NodeResourceGroup
	apiCallCount := 0
	maxAPICallCount := 100
	returnNodePoolPair := NodePoolPair{}
	// var targetAgentPool *containerservice.AgentPool
	var waitingErr error
	for {
		agentPool, err := agentPoolsClient.Get(ctx, clusterResourceGroup, *cluster.Name, agentPoolName)
		if err == nil {
			returnNodePoolPair.AgentPool = agentPool
			break
		}
		apiCallCount++
		if apiCallCount >= maxAPICallCount {
			waitingErr = errors.New("failed get Cluster AgentPool err = The maximum number of verification requests has been exceeded while waiting for the creation of that resource")
			break
		}
		time.Sleep(4 * time.Second)
	}
	if waitingErr != nil {
		return NodePoolPair{}, waitingErr
	}
	apiCallCount = 0
	for {
		scaleSetList, err := virtualMachineScaleSetsClient.List(ctx, clusterManagedResourceGroup)
		if err == nil {
			scaleSetCheck := false
			for _, scaleSet := range scaleSetList.Values() {
				val, exist := scaleSet.Tags[ScaleSetOwnerKey]
				if exist && *val == agentPoolName {
					returnNodePoolPair.virtualMachineScaleSet = scaleSet
					scaleSetCheck = true
					break
				}
			}
			if scaleSetCheck {
				break
			}
		}
		apiCallCount++
		if apiCallCount >= maxAPICallCount {
			waitingErr = errors.New("failed get Cluster VirtualMachineScaleSet err = The maximum number of verification requests has been exceeded while waiting for the creation of that resource")
			break
		}
		time.Sleep(4 * time.Second)
	}
	if waitingErr != nil {
		return NodePoolPair{}, waitingErr
	}
	return returnNodePoolPair, nil
}

type ServerKubeConfig struct {
	Clusters []struct {
		Cluster struct {
			Server                   string `yaml:"server"`
			CertificateAuthorityData string `yaml:"certificate-authority-data"`
		} `yaml:"cluster"`
		Name string `yaml:"name"`
	} `yaml:"clusters"`
	Context []struct {
		Context struct {
			Cluster string `yaml:"cluster"`
			User    string `yaml:"user"`
		} `yaml:"context"`
		Name string `yaml:"name"`
	} `yaml:"contexts"`
	CurrentContext string `yaml:"current-context"`
	Kind           string `yaml:"kind"`
	Users          []struct {
		User struct {
			ClientCertificateData string `yaml:"client-certificate-data"`
			ClientKeyData         string `yaml:"client-key-data"`
			Token                 string `yaml:"token"`
		}
		Name string `yaml:"name"`
	} `yaml:"users"`
}

func getClusterAccessInfo(cluster containerservice.ManagedCluster, managedClustersClient *containerservice.ManagedClustersClient, ctx context.Context) (accessInfo irs.AccessInfo, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("faild get AccessInfo"))
			accessInfo = irs.AccessInfo{}
		}
	}()
	clusterResourceGroup, err := getResourceGroupById(*cluster.ID)
	if err != nil {
		return irs.AccessInfo{}, errors.New(fmt.Sprintf("faild get AccessInfo err = %s", err.Error()))
	}
	config, err := managedClustersClient.GetAccessProfile(ctx, clusterResourceGroup, *cluster.Name, ClusterAdminKey)
	if err != nil {
		return irs.AccessInfo{}, errors.New(fmt.Sprintf("faild get AccessInfo err = %s", err.Error()))
	}
	accessInfo.Kubeconfig = string(*config.KubeConfig)

	kubeConfig := ServerKubeConfig{}
	err = yaml.Unmarshal(*config.KubeConfig, &kubeConfig)
	if err != nil {
		return irs.AccessInfo{}, errors.New(fmt.Sprintf("faild get AccessInfo err = %s", err.Error()))
	}
	for _, clusterConfig := range kubeConfig.Clusters {
		if clusterConfig.Name == *cluster.Name {
			accessInfo.Endpoint = clusterConfig.Cluster.Server
			break
		}
	}
	return accessInfo, nil
}
