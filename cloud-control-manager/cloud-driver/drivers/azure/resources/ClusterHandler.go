package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/containerservice/armcontainerservice/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"gopkg.in/yaml.v2"
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
	ManagedClustersClient           *armcontainerservice.ManagedClustersClient
	VirtualNetworksClient           *armnetwork.VirtualNetworksClient
	AgentPoolsClient                *armcontainerservice.AgentPoolsClient
	VirtualMachineScaleSetsClient   *armcompute.VirtualMachineScaleSetsClient
	VirtualMachineScaleSetVMsClient *armcompute.VirtualMachineScaleSetVMsClient
	SubnetClient                    *armnetwork.SubnetsClient
	SecurityGroupsClient            *armnetwork.SecurityGroupsClient
	SecurityRulesClient             *armnetwork.SecurityRulesClient
	VirtualMachineSizesClient       *armcompute.VirtualMachineSizesClient
	SSHPublicKeysClient             *armcompute.SSHPublicKeysClient
}

type auth struct {
	AccessToken string `json:"access_token"`
}

func getToken(tenantID string, clientID string, clientSecret string) (string, error) {
	URL := "https://login.microsoftonline.com/" + tenantID + "/oauth2/token"

	params := url.Values{}
	params.Add("client_id", clientID)
	params.Add("grant_type", "client_credentials")
	params.Add("resource", "https://management.azure.com/")
	params.Add("client_secret", clientSecret)
	body := strings.NewReader(params.Encode())

	ctx := context.Background()
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, URL, body)
	if err != nil {
		return "", err
	}

	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var azureAuth auth
	err = json.Unmarshal(responseBody, &azureAuth)
	if err != nil {
		return "", err
	}

	return azureAuth.AccessToken, nil
}

type patchVersion struct {
	Upgrades []string `json:"upgrades"`
}

type version struct {
	Version       string                  `json:"version"`
	IsDefault     bool                    `json:"isDefault,omitempty"`
	Capabilities  map[string][]string     `json:"capabilities"`
	PatchVersions map[string]patchVersion `json:"patchVersions"`
}

type k8sVersions struct {
	Values []version `json:"values"`
}

func getK8SVersions(credentialInfo idrv.CredentialInfo, location string) ([]string, error) {
	URL := "https://management.azure.com/subscriptions/" + credentialInfo.SubscriptionId +
		"/providers/Microsoft.ContainerService/locations/" + location + "/kubernetesVersions?api-version=2024-02-01"

	token, err := getToken(credentialInfo.TenantId, credentialInfo.ClientId, credentialInfo.ClientSecret)
	if err != nil {
		return nil, err
	}
	var bearer = "Bearer " + token

	ctx := context.Background()
	client := &http.Client{}
	req, err := http.NewRequest(http.MethodGet, URL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Authorization", bearer)
	req = req.WithContext(ctx)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var vs k8sVersions
	err = json.Unmarshal(responseBody, &vs)
	if err != nil {
		return nil, err
	}

	var versions []string
	for _, v := range vs.Values {
		keys := reflect.ValueOf(v.PatchVersions).MapKeys()
		for _, key := range keys {
			versions = append(versions, key.Interface().(string))
		}
	}

	sort.Slice(versions, func(i, j int) bool { return versions[i] > versions[j] })

	return versions, nil
}

func (ac *AzureClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (info irs.ClusterInfo, createErr error) {
	hiscallInfo := GetCallLogScheme(ac.Region, call.CLUSTER, clusterReqInfo.IId.NameId, "CreateCluster()")
	start := call.Start()

	versions, err := getK8SVersions(ac.CredentialInfo, ac.Region.Region)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to get K8S versions while Creating Cluster. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.ClusterInfo{}, createErr
	}

	var k8sVersionUnsupported = true
	for _, version := range versions {
		if clusterReqInfo.Version == version {
			k8sVersionUnsupported = false
			break
		}
	}
	if k8sVersionUnsupported {
		createErr = errors.New(fmt.Sprintf("Failed to Create Cluster. " +
			"err = Unsupported K8S version. (Available versions: " + strings.Join(versions[:], ", ") + ")"))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.ClusterInfo{}, createErr
	}

	err = createCluster(clusterReqInfo, ac)
	if err != nil {
		createErr = errors.New(fmt.Sprintf("Failed to Create Cluster. err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.ClusterInfo{}, createErr
	}
	defer func() {
		if createErr != nil {
			_ = cleanCluster(clusterReqInfo.IId.NameId, ac.ManagedClustersClient, ac.Region.Region, ac.Ctx)
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
	info, err = setterClusterInfo(&cluster, ac.ManagedClustersClient, ac.SecurityGroupsClient, ac.VirtualNetworksClient, ac.AgentPoolsClient, ac.VirtualMachineScaleSetsClient, ac.VirtualMachineScaleSetVMsClient, ac.CredentialInfo, ac.Region, ac.Ctx)
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

	var clusterList []*armcontainerservice.ManagedCluster

	pager := ac.ManagedClustersClient.NewListPager(nil)

	for pager.More() {
		page, err := pager.NextPage(ac.Ctx)
		if err != nil {
			getErr = errors.New(fmt.Sprintf("Failed to List Cluster. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return make([]*irs.ClusterInfo, 0), getErr
		}

		for _, cluster := range page.Value {
			clusterList = append(clusterList, cluster)
		}
	}

	listInfo, err := setterClusterInfoList(clusterList, ac.ManagedClustersClient, ac.SecurityGroupsClient, ac.VirtualNetworksClient, ac.AgentPoolsClient, ac.VirtualMachineScaleSetsClient, ac.VirtualMachineScaleSetVMsClient, ac.CredentialInfo, ac.Region, ac.Ctx)
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
	info, err = setterClusterInfo(&cluster, ac.ManagedClustersClient, ac.SecurityGroupsClient, ac.VirtualNetworksClient, ac.AgentPoolsClient, ac.VirtualMachineScaleSetsClient, ac.VirtualMachineScaleSetVMsClient, ac.CredentialInfo, ac.Region, ac.Ctx)
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

	err := cleanCluster(clusterIID.NameId, ac.ManagedClustersClient, ac.Region.Region, ac.Ctx)
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
	nodeGroupInfo, err := addNodeGroupPool(&cluster, nodeGroupReqInfo, ac)
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
	err = upgradeCluter(&cluster, newVersion, ac.ManagedClustersClient, ac.AgentPoolsClient, ac.VirtualMachineScaleSetsClient, ac.Ctx, ac.Region)
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
	info, err = setterClusterInfo(&cluster, ac.ManagedClustersClient, ac.SecurityGroupsClient, ac.VirtualNetworksClient, ac.AgentPoolsClient, ac.VirtualMachineScaleSetsClient, ac.VirtualMachineScaleSetVMsClient, ac.CredentialInfo, ac.Region, ac.Ctx)
	if err != nil {
		upgradeErr = errors.New(fmt.Sprintf("Failed to Upgrade Cluster. err = %s", err))
		cblogger.Error(upgradeErr.Error())
		LoggingError(hiscallInfo, upgradeErr)
		return irs.ClusterInfo{}, upgradeErr
	}
	LoggingInfo(hiscallInfo, start)
	return info, nil
}

func checkUpgradeCluster(cluster *armcontainerservice.ManagedCluster, agentPoolsClient *armcontainerservice.AgentPoolsClient, virtualMachineScaleSetsClient *armcompute.VirtualMachineScaleSetsClient, ctx context.Context) error {
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

func upgradeCluter(cluster *armcontainerservice.ManagedCluster, newVersion string, managedClustersClient *armcontainerservice.ManagedClustersClient, agentPoolsClient *armcontainerservice.AgentPoolsClient, virtualMachineScaleSetsClient *armcompute.VirtualMachineScaleSetsClient, ctx context.Context, region idrv.RegionInfo) error {
	err := checkUpgradeCluster(cluster, agentPoolsClient, virtualMachineScaleSetsClient, ctx)
	if err != nil {
		return err
	}
	updateCluster := cluster
	updateCluster.Properties.KubernetesVersion = &newVersion
	_, err = managedClustersClient.BeginCreateOrUpdate(ctx, region.Region, *cluster.Name, *updateCluster, nil)
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

func getRawCluster(clusterIID irs.IID, managedClustersClient *armcontainerservice.ManagedClustersClient, ctx context.Context, credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo) (armcontainerservice.ManagedCluster, error) {
	clusterName := clusterIID.NameId
	if clusterName == "" {
		convertedIID, err := convertedClusterIID(clusterIID, credentialInfo, regionInfo)
		if err != nil {
			return armcontainerservice.ManagedCluster{}, err
		}
		clusterName = convertedIID.NameId
	}
	result, err := managedClustersClient.Get(ctx, regionInfo.Region, clusterName, nil)
	if err != nil {
		return armcontainerservice.ManagedCluster{}, err
	}
	return result.ManagedCluster, nil
}

type ClusterInfoWithError struct {
	ClusterInfo irs.ClusterInfo
	err         error
}

func setterClusterInfoWithCancel(cluster *armcontainerservice.ManagedCluster, managedClustersClient *armcontainerservice.ManagedClustersClient, securityGroupsClient *armnetwork.SecurityGroupsClient, virtualNetworksClient *armnetwork.VirtualNetworksClient, agentPoolsClient *armcontainerservice.AgentPoolsClient, virtualMachineScaleSetsClient *armcompute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *armcompute.VirtualMachineScaleSetVMsClient, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context, cancelCtx context.Context) (irs.ClusterInfo, error) {
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

func setterClusterInfoList(clusterList []*armcontainerservice.ManagedCluster, managedClustersClient *armcontainerservice.ManagedClustersClient, securityGroupsClient *armnetwork.SecurityGroupsClient, virtualNetworksClient *armnetwork.VirtualNetworksClient, agentPoolsClient *armcontainerservice.AgentPoolsClient, virtualMachineScaleSetsClient *armcompute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *armcompute.VirtualMachineScaleSetVMsClient, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context) (clusterInfoList []*irs.ClusterInfo, err error) {
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

func setterClusterInfo(cluster *armcontainerservice.ManagedCluster,
	managedClustersClient *armcontainerservice.ManagedClustersClient,
	securityGroupsClient *armnetwork.SecurityGroupsClient,
	virtualNetworksClient *armnetwork.VirtualNetworksClient,
	agentPoolsClient *armcontainerservice.AgentPoolsClient,
	virtualMachineScaleSetsClient *armcompute.VirtualMachineScaleSetsClient,
	virtualMachineScaleSetVMsClient *armcompute.VirtualMachineScaleSetVMsClient,
	credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context) (clusterInfo irs.ClusterInfo, err error) {
	clusterInfo.IId = irs.IID{*cluster.Name, *cluster.ID}
	if cluster.Properties != nil {
		// Version
		if cluster.Properties.KubernetesVersion != nil {
			clusterInfo.Version = *cluster.Properties.KubernetesVersion
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
	if cluster.Tags != nil {
		clusterInfo.TagList = setTagList(cluster.Tags)
	}
	return clusterInfo, nil
}

func getClusterStatus(cluster *armcontainerservice.ManagedCluster) (resultStatus irs.ClusterStatus) {
	defer func() {
		if r := recover(); r != nil {
			resultStatus = irs.ClusterInactive
		}
	}()
	resultStatus = irs.ClusterInactive
	if cluster.Properties.ProvisioningState == nil || cluster.Properties.PowerState == nil {
		return resultStatus
	}
	provisioningState := *cluster.Properties.ProvisioningState
	powerState := *cluster.Properties.PowerState.Code
	if powerState != armcontainerservice.CodeRunning {
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

func getAddonInfo(cluster *armcontainerservice.ManagedCluster) (info irs.AddonsInfo, err error) {
	defer func() {
		if r := recover(); r != nil {
			info = irs.AddonsInfo{}
			err = errors.New("faild get AddonProfiles")
		}
	}()
	keyvalues := make([]irs.KeyValue, 0)
	for AddonProName, AddonProfile := range cluster.Properties.AddonProfiles {
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

func getSubnetIdByAgentPoolProfiles(agentPoolProfiles []*armcontainerservice.ManagedClusterAgentPoolProfile) (subnetId string, err error) {
	var targetSubnetId *string
	for _, agentPool := range agentPoolProfiles {
		if agentPool.VnetSubnetID != nil {
			targetSubnetId = agentPool.VnetSubnetID
			break
		}
	}
	if targetSubnetId == nil {
		return "", errors.New("invalid cluster Network")
	}
	return *targetSubnetId, nil
}

func getNetworkInfo(cluster *armcontainerservice.ManagedCluster, securityGroupsClient *armnetwork.SecurityGroupsClient, virtualNetworksClient *armnetwork.VirtualNetworksClient, CredentialInfo idrv.CredentialInfo, Region idrv.RegionInfo, ctx context.Context) (info irs.NetworkInfo, err error) {
	defer func() {
		if r := recover(); r != nil {
			info = irs.NetworkInfo{}
			err = errors.New("invalid cluster Network")
		}
	}()
	if *cluster.Properties.NetworkProfile.NetworkPlugin == armcontainerservice.NetworkPluginAzure {
		if cluster.Properties.AgentPoolProfiles != nil && len(cluster.Properties.AgentPoolProfiles) > 0 {
			subnetId, err := getSubnetIdByAgentPoolProfiles(cluster.Properties.AgentPoolProfiles)
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
	} else if *cluster.Properties.NetworkProfile.NetworkPlugin == armcontainerservice.NetworkPluginKubenet {
		// NetworkInfo - Network Configuration Kubenet
		if cluster.Properties.NodeResourceGroup != nil {
			var networkList []*armnetwork.VirtualNetwork

			pager := virtualNetworksClient.NewListPager(*cluster.Properties.NodeResourceGroup, nil)

			for pager.More() {
				page, err := pager.NextPage(ctx)
				if err != nil {
					return irs.NetworkInfo{}, err
				}

				for _, vpcNetwork := range page.Value {
					networkList = append(networkList, vpcNetwork)
				}
			}

			vpcNetwork := networkList[0]
			info.VpcIID = irs.IID{*vpcNetwork.Name, *vpcNetwork.ID}
			subnetIIDArray := make([]irs.IID, len(vpcNetwork.Properties.Subnets))
			segIIDArray := make([]irs.IID, 0)
			for i, subnet := range vpcNetwork.Properties.Subnets {
				subnetIIDArray[i] = irs.IID{*subnet.Name, *subnet.ID}
			}
			info.SubnetIIDs = segIIDArray
			info.SecurityGroupIIDs = segIIDArray

			return info, nil
		}
	}
	return irs.NetworkInfo{}, errors.New("empty cluster AgentPoolProfiles")
}

type NodePoolPair struct {
	AgentPool              armcontainerservice.AgentPool
	virtualMachineScaleSet armcompute.VirtualMachineScaleSet
}

func getRawNodePoolPairList(cluster *armcontainerservice.ManagedCluster, agentPoolsClient *armcontainerservice.AgentPoolsClient, virtualMachineScaleSetsClient *armcompute.VirtualMachineScaleSetsClient, ctx context.Context) ([]NodePoolPair, error) {
	clusterResourceGroup, err := getResourceGroupById(*cluster.ID)
	if err != nil {
		return make([]NodePoolPair, 0), errors.New(fmt.Sprintf("failed get clusterResourceGroup err = %s", err.Error()))
	}
	if cluster.Properties.NodeResourceGroup == nil {
		return make([]NodePoolPair, 0), errors.New("invalid cluster Resource")
	}
	resourceGroupManagedK8s := *cluster.Properties.NodeResourceGroup

	var agentPoolList []*armcontainerservice.AgentPool

	pager := agentPoolsClient.NewListPager(clusterResourceGroup, *cluster.Name, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, agentPool := range page.Value {
			agentPoolList = append(agentPoolList, agentPool)
		}
	}

	var scaleSetList []*armcompute.VirtualMachineScaleSet

	pager2 := virtualMachineScaleSetsClient.NewListPager(resourceGroupManagedK8s, nil)

	for pager2.More() {
		page, err := pager2.NextPage(ctx)
		if err != nil {
			return nil, err
		}

		for _, scaleSet := range page.Value {
			scaleSetList = append(scaleSetList, scaleSet)
		}
	}

	var filteredNodePoolPairList []NodePoolPair

	for _, scaleSet := range scaleSetList {
		value, exist := scaleSet.Tags["aks-managed-poolName"]
		if exist {
			for _, agentPool := range agentPoolList {
				if *value == *agentPool.Name {
					filteredNodePoolPairList = append(filteredNodePoolPairList, NodePoolPair{
						AgentPool:              *agentPool,
						virtualMachineScaleSet: *scaleSet,
					})
					break
				}
			}
		}
	}
	return filteredNodePoolPairList, nil
}

func convertNodePoolPairSpecifiedNodePool(cluster *armcontainerservice.ManagedCluster, nodePoolName string, agentPoolsClient *armcontainerservice.AgentPoolsClient, virtualMachineScaleSetsClient *armcompute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *armcompute.VirtualMachineScaleSetVMsClient, ctx context.Context) (irs.NodeGroupInfo, error) {
	if cluster.Properties.NodeResourceGroup == nil {
		return irs.NodeGroupInfo{}, errors.New("invalid cluster resource")
	}
	resourceGroupManagedK8s := *cluster.Properties.NodeResourceGroup
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

func convertNodePoolPair(cluster *armcontainerservice.ManagedCluster,
	agentPoolsClient *armcontainerservice.AgentPoolsClient,
	virtualMachineScaleSetsClient *armcompute.VirtualMachineScaleSetsClient,
	virtualMachineScaleSetVMsClient *armcompute.VirtualMachineScaleSetVMsClient, ctx context.Context) ([]irs.NodeGroupInfo, error) {
	if cluster.Properties.NodeResourceGroup == nil {
		return make([]irs.NodeGroupInfo, 0), errors.New("invalid cluster resource")
	}
	resourceGroupManagedK8s := *cluster.Properties.NodeResourceGroup
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

func getNodeGroupInfoSpecifiedNodePool(cluster *armcontainerservice.ManagedCluster, nodePoolName string, agentPoolsClient *armcontainerservice.AgentPoolsClient, virtualMachineScaleSetsClient *armcompute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *armcompute.VirtualMachineScaleSetVMsClient, ctx context.Context) (irs.NodeGroupInfo, error) {
	nodeInfoGroup, err := convertNodePoolPairSpecifiedNodePool(cluster, nodePoolName, agentPoolsClient, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, ctx)
	if err != nil {
		return irs.NodeGroupInfo{}, err
	}
	return nodeInfoGroup, nil
}

func getNodeGroupInfoList(cluster *armcontainerservice.ManagedCluster, agentPoolsClient *armcontainerservice.AgentPoolsClient, virtualMachineScaleSetsClient *armcompute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *armcompute.VirtualMachineScaleSetVMsClient, ctx context.Context) ([]irs.NodeGroupInfo, error) {
	nodeInfoGroupList, err := convertNodePoolPair(cluster, agentPoolsClient, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, ctx)
	if err != nil {
		return make([]irs.NodeGroupInfo, 0), err
	}
	return nodeInfoGroupList, nil
}

func getNodeInfoStatus(agentPool armcontainerservice.AgentPool, virtualMachineScaleSet armcompute.VirtualMachineScaleSet) (status irs.NodeGroupStatus) {
	defer func() {
		if r := recover(); r != nil {
			status = irs.NodeGroupInactive
		}
	}()
	if reflect.ValueOf(virtualMachineScaleSet.Properties.ProvisioningState).IsNil() || reflect.ValueOf(agentPool.Properties.ProvisioningState).IsNil() {
		return irs.NodeGroupInactive
	}
	if *virtualMachineScaleSet.Properties.ProvisioningState == "Succeeded" && *agentPool.Properties.ProvisioningState == "Succeeded" {
		return irs.NodeGroupActive
	}
	if *virtualMachineScaleSet.Properties.ProvisioningState == "Creating" || *agentPool.Properties.ProvisioningState == "Creating" {
		return irs.NodeGroupCreating
	}
	if *virtualMachineScaleSet.Properties.ProvisioningState == "Updating" || *agentPool.Properties.ProvisioningState == "Updating" ||
		*virtualMachineScaleSet.Properties.ProvisioningState == "Upgrading" || *agentPool.Properties.ProvisioningState == "Upgrading" ||
		*virtualMachineScaleSet.Properties.ProvisioningState == "Scaling" || *agentPool.Properties.ProvisioningState == "Scaling" {
		return irs.NodeGroupUpdating
	}
	if *virtualMachineScaleSet.Properties.ProvisioningState == "Deleting" || *agentPool.Properties.ProvisioningState == "Deleting" {
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

func checkValidationNodeGroups(nodeGroups []irs.NodeGroupInfo, virtualMachineSizesClient *armcompute.VirtualMachineSizesClient, regionInfo idrv.RegionInfo, ctx context.Context) error {
	// https://learn.microsoft.com/en-us/azure/aks/quotas-skus-regions
	if len(nodeGroups) == 0 {
		return errors.New("nodeGroup Empty")
	}

	var vmSpecList []*armcompute.VirtualMachineSize

	pager := virtualMachineSizesClient.NewListPager(regionInfo.Region, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return errors.New(fmt.Sprintf("Failed get VMSPEC List"))
		}

		for _, vmSpec := range page.Value {
			vmSpecList = append(vmSpecList, vmSpec)
		}
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
		if i != 0 && sshKeyIID != nil {
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
		for _, rawspec := range vmSpecList {
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

func checkValidationCreateCluster(clusterReqInfo irs.ClusterInfo, virtualMachineSizesClient *armcompute.VirtualMachineSizesClient, regionInfo idrv.RegionInfo, ctx context.Context) error {
	// nodegroup 확인
	err := checkValidationNodeGroups(clusterReqInfo.NodeGroupList, virtualMachineSizesClient, regionInfo, ctx)
	// network
	if err != nil {
		return errors.New(fmt.Sprintf("Failed Validation Check NodeGroup. err = %s", err.Error()))
	}
	return nil
}

func createCluster(clusterReqInfo irs.ClusterInfo, ac *AzureClusterHandler) error {
	// 사전 확인
	err := checkValidationCreateCluster(clusterReqInfo, ac.VirtualMachineSizesClient, ac.Region, ac.Ctx)
	if err != nil {
		return err
	}
	targetSubnet, err := getRawClusterTargetSubnet(clusterReqInfo.Network, ac.VirtualNetworksClient, ac.Ctx, ac.Region.Region)
	if err != nil {
		return err
	}
	// agentPoolProfiles
	agentPoolProfiles, err := generateAgentPoolProfileList(clusterReqInfo, targetSubnet, ac)
	if err != nil {
		return err
	}
	// networkProfile
	networkProfile, err := generatorNetworkProfile(clusterReqInfo, targetSubnet)
	if err != nil {
		return err
	}
	// mapping ssh
	linuxProfileSSH, sshKey, err := generateManagedClusterLinuxProfileSSH(clusterReqInfo, ac.SSHPublicKeysClient, ac.Region.Region, ac.Ctx)
	if err != nil {
		return err
	}
	tags, err := generatorClusterTags(*sshKey.Name, clusterReqInfo.IId.NameId)
	if err != nil {
		return err
	}
	addonProfiles := generatePreparedAddonProfiles()

	clusterCreateOpts := armcontainerservice.ManagedCluster{
		Location: toStrPtr(ac.Region.Region),
		SKU: &armcontainerservice.ManagedClusterSKU{
			Name: (*armcontainerservice.ManagedClusterSKUName)(toStrPtr(string(armcontainerservice.ManagedClusterSKUNameBase))),
			Tier: (*armcontainerservice.ManagedClusterSKUTier)(toStrPtr(string(armcontainerservice.ManagedClusterSKUTierStandard))),
		},
		Identity: &armcontainerservice.ManagedClusterIdentity{
			Type: (*armcontainerservice.ResourceIdentityType)(toStrPtr(string(armcontainerservice.ResourceIdentityTypeSystemAssigned))),
		},
		Tags: tags,
		Properties: &armcontainerservice.ManagedClusterProperties{
			KubernetesVersion: toStrPtr(clusterReqInfo.Version),
			EnableRBAC:        toBoolPtr(true),
			DNSPrefix:         toStrPtr(getclusterDNSPrefix(clusterReqInfo.IId.NameId)),
			NodeResourceGroup: toStrPtr(getclusterNodeResourceGroup(clusterReqInfo.IId.NameId, ac.Region.Region, ac.Region.Region)),
			AgentPoolProfiles: agentPoolProfiles,
			NetworkProfile:    &networkProfile,
			LinuxProfile:      &linuxProfileSSH,
			AddonProfiles:     addonProfiles,
		},
	}
	if clusterReqInfo.TagList != nil {
		for _, tag := range clusterReqInfo.TagList {
			clusterCreateOpts.Tags[tag.Key] = &tag.Value
		}
	}

	_, err = ac.ManagedClustersClient.BeginCreateOrUpdate(ac.Ctx, ac.Region.Region, clusterReqInfo.IId.NameId, clusterCreateOpts, nil)
	if err != nil {
		return err
	}
	//newCluster, err := getRawCluster(irs.IID{NameId: clusterReqInfo.IId.NameId}, managedClustersClient, ctx, credentialInfo, regionInfo)
	//if err != nil {
	//	return armcontainerservice.ManagedCluster{}, err
	//}
	//return newCluster, nil
	return nil
}

func cleanCluster(clusterName string, managedClustersClient *armcontainerservice.ManagedClustersClient, region string, ctx context.Context) error {
	// cluster subresource Clean 현재 없음
	// delete Cluster
	_, err := managedClustersClient.BeginDelete(ctx, region, clusterName, nil)
	if err != nil {
		return err
	}

	return nil
}

func checkSubnetRequireIPRange(subnet armnetwork.Subnet, NodeGroupInfos []irs.NodeGroupInfo) error {
	_, ipnet, err := net.ParseCIDR(*subnet.Properties.AddressPrefix)
	if err != nil {
		return errors.New("invalid Cidr")
	}
	ones, octaBits := ipnet.Mask.Size()
	realRangeBit := float64(octaBits - ones)
	inUsedIPCount := float64(5) // default Azure reserved
	if subnet.Properties.IPConfigurations != nil {
		inUsedIPCount += float64(len(subnet.Properties.IPConfigurations))
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

func getRawClusterTargetSubnet(networkInfo irs.NetworkInfo, virtualNetworksClient *armnetwork.VirtualNetworksClient, ctx context.Context, resourceGroup string) (armnetwork.Subnet, error) {
	if len(networkInfo.SubnetIIDs) != 1 {
		return armnetwork.Subnet{}, errors.New("The Azure Cluster uses only one subnet in the VPC.")
	}
	if networkInfo.SubnetIIDs[0].NameId == "" && networkInfo.SubnetIIDs[0].SystemId == "" {
		return armnetwork.Subnet{}, errors.New("subnet IID within networkInfo is empty")
	}
	targetSubnetName := ""
	if networkInfo.SubnetIIDs[0].NameId != "" {
		targetSubnetName = networkInfo.SubnetIIDs[0].NameId
	} else {
		name, err := getNameById(networkInfo.SubnetIIDs[0].SystemId, AzureSubnet)
		if err != nil {
			return armnetwork.Subnet{}, errors.New("subnet IID within networkInfo is invalid ID")
		}
		targetSubnetName = name
	}
	rawVPC, err := getRawVirtualNetwork(networkInfo.VpcIID, virtualNetworksClient, ctx, resourceGroup)
	if err != nil {
		return armnetwork.Subnet{}, errors.New("failed get Cluster Vpc And Subnet")
	}
	// first Subnet
	var targetSubnet *armnetwork.Subnet
	for _, subnet := range rawVPC.Properties.Subnets {
		if subnet.Name != nil && *subnet.Name == targetSubnetName {
			targetSubnet = subnet
			break
		}
	}
	if targetSubnet == nil {
		return armnetwork.Subnet{}, errors.New(fmt.Sprintf("subnet within that vpc does not exist."))
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

func generatorServiceCidrDNSServiceIP(subnetCidr string) (ServiceCidr string, DNSServiceIP string, err error) {
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
		return "", "", errors.New("invalid subnet Cidr")
	}

	ipSplits := strings.Split(subnetIP.String(), ".")
	if len(ipSplits) != 4 {
		return "", "", errors.New("invalid subnet Cidr")
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
		return "", "", errors.New("the cidr on the current subnet and the serviceCidr on the cb-spider are overlapping. The areas of ServiceCidr checking for superposition in cb-spider are 10.0.0.0/16 to 10.255.0.0/16, and 172.16.0.0 to 172.29.0")
	}
	newip, _, err := net.ParseCIDR(ServiceCidr)
	netipSplits := strings.Split(newip.String(), ".")
	if len(ipSplits) != 4 {
		return "", "", errors.New("invalid subnet Cidr")
	}
	netipSplits[3] = "10"
	DNSServiceIP = strings.Join(netipSplits, ".")
	return ServiceCidr, DNSServiceIP, nil
}

func generatorNetworkProfile(ClusterInfo irs.ClusterInfo, targetSubnet armnetwork.Subnet) (armcontainerservice.NetworkProfile, error) {
	err := checkSubnetRequireIPRange(targetSubnet, ClusterInfo.NodeGroupList)
	if err != nil {
		return armcontainerservice.NetworkProfile{}, errors.New(fmt.Sprintf("failed get vpc err = %s", err.Error()))
	}

	serviceCidr, DNSServiceIP, err := generatorServiceCidrDNSServiceIP(*targetSubnet.Properties.AddressPrefix)
	if err != nil {
		return armcontainerservice.NetworkProfile{}, errors.New(fmt.Sprintf("failed calculate ServiceCidr, DNSServiceIP err = %s", err.Error()))
	}

	return armcontainerservice.NetworkProfile{
		LoadBalancerSKU: (*armcontainerservice.LoadBalancerSKU)(toStrPtr(string(armcontainerservice.LoadBalancerSKUStandard))),
		NetworkPlugin:   (*armcontainerservice.NetworkPlugin)(toStrPtr(string(armcontainerservice.NetworkPluginAzure))),
		NetworkPolicy:   (*armcontainerservice.NetworkPolicy)(toStrPtr(string(armcontainerservice.NetworkPolicyAzure))),
		ServiceCidr:     &serviceCidr,
		DNSServiceIP:    &DNSServiceIP,
	}, nil

}

func generatorClusterTags(sshKeyName string, clusterName string) (map[string]*string, error) {
	tags := make(map[string]*string)
	nowTime := strconv.FormatInt(time.Now().Unix(), 10)
	tags[ClusterNodeSSHKeyKey] = &sshKeyName
	tags["createdAt"] = &nowTime
	tags[OwnerClusterKey] = &clusterName
	return tags, nil
}

func getSSHKeyIIDByNodeGroups(NodeGroupInfos []irs.NodeGroupInfo) (irs.IID, error) {
	var key *irs.IID
	for _, nodeGroup := range NodeGroupInfos {
		if nodeGroup.KeyPairIID.NameId != "" && nodeGroup.KeyPairIID.SystemId != "" {
			key = &nodeGroup.KeyPairIID
			break
		}
	}
	if key == nil {
		return irs.IID{}, errors.New("failed find SSHKey IID By nodeGroups")
	}
	return *key, nil
}

func generatePreparedAddonProfiles() map[string]*armcontainerservice.ManagedClusterAddonProfile {
	return map[string]*armcontainerservice.ManagedClusterAddonProfile{
		"httpApplicationRouting": {
			Enabled: toBoolPtr(true),
		},
	}
}

func generateManagedClusterLinuxProfileSSH(clusterReqInfo irs.ClusterInfo, sshPublicKeysClient *armcompute.SSHPublicKeysClient, resourceGroup string, ctx context.Context) (armcontainerservice.LinuxProfile, armcompute.SSHPublicKeyResource, error) {
	sshkeyId, err := getSSHKeyIIDByNodeGroups(clusterReqInfo.NodeGroupList)
	if err != nil {
		return armcontainerservice.LinuxProfile{}, armcompute.SSHPublicKeyResource{}, err
	}
	key, err := GetRawKey(sshkeyId, resourceGroup, sshPublicKeysClient, ctx)
	if err != nil {
		return armcontainerservice.LinuxProfile{}, armcompute.SSHPublicKeyResource{}, errors.New(fmt.Sprintf("failed get ssh Key, err = %s", err.Error()))
	}
	linuxProfile := armcontainerservice.LinuxProfile{
		AdminUsername: toStrPtr(CBVMUser),
		SSH: &armcontainerservice.SSHConfiguration{
			PublicKeys: []*armcontainerservice.SSHPublicKey{
				{
					KeyData: key.Properties.PublicKey,
				},
			},
		},
	}
	return linuxProfile, key, nil
}

func generateAgentPoolProfileList(info irs.ClusterInfo, targetSubnet armnetwork.Subnet, ac *AzureClusterHandler) ([]*armcontainerservice.ManagedClusterAgentPoolProfile, error) {
	agentPoolProfiles := make([]*armcontainerservice.ManagedClusterAgentPoolProfile, len(info.NodeGroupList))
	for i, nodeGroupInfo := range info.NodeGroupList {
		agentPoolProfile, err := generateAgentPoolProfile(nodeGroupInfo, targetSubnet, ac)
		if err != nil {
			return make([]*armcontainerservice.ManagedClusterAgentPoolProfile, 0), err
		}
		agentPoolProfiles[i] = &agentPoolProfile
	}
	return agentPoolProfiles, nil
}

func generateAgentPoolProfileProperties(nodeGroupInfo irs.NodeGroupInfo, subnet armnetwork.Subnet, ac *AzureClusterHandler) (armcontainerservice.ManagedClusterAgentPoolProfileProperties, error) {
	var nodeOSDiskSize *int32

	if nodeGroupInfo.RootDiskSize == "" || nodeGroupInfo.RootDiskSize == "default" {
		nodeOSDiskSize = nil
	} else {
		osDiskSize, err := strconv.Atoi(nodeGroupInfo.RootDiskSize)
		if err != nil {
			return armcontainerservice.ManagedClusterAgentPoolProfileProperties{}, errors.New("invalid NodeGroup RootDiskSize")
		}
		nodeOSDiskSize = toInt32Ptr(osDiskSize)
	}

	targetZones := []*string{&ac.Region.Zone}

	agentPoolProfileProperties := armcontainerservice.ManagedClusterAgentPoolProfileProperties{
		// Name:         to.StringPtr(nodeGroupInfo.IId.NameId),
		Count:              toInt32Ptr(nodeGroupInfo.DesiredNodeSize),
		MinCount:           toInt32Ptr(nodeGroupInfo.MinNodeSize),
		MaxCount:           toInt32Ptr(nodeGroupInfo.MaxNodeSize),
		VMSize:             &nodeGroupInfo.VMSpecName,
		OSDiskSizeGB:       nodeOSDiskSize,
		OSType:             (*armcontainerservice.OSType)(toStrPtr(string(armcontainerservice.OSTypeLinux))),
		Type:               (*armcontainerservice.AgentPoolType)(toStrPtr(string(armcontainerservice.AgentPoolTypeVirtualMachineScaleSets))),
		MaxPods:            toInt32Ptr(maxPodCount),
		Mode:               (*armcontainerservice.AgentPoolMode)(toStrPtr(string(armcontainerservice.AgentPoolModeSystem))), // User? System?
		AvailabilityZones:  targetZones,
		EnableNodePublicIP: toBoolPtr(true),
		EnableAutoScaling:  toBoolPtr(nodeGroupInfo.OnAutoScaling),
		// MinCount가 있으려면 true 여야함
		VnetSubnetID: subnet.ID,
	}
	return agentPoolProfileProperties, nil
}

func generateAgentPoolProfile(nodeGroupInfo irs.NodeGroupInfo, subnet armnetwork.Subnet, ac *AzureClusterHandler) (armcontainerservice.ManagedClusterAgentPoolProfile, error) {
	var nodeOSDiskSize *int32

	if nodeGroupInfo.RootDiskSize == "" || nodeGroupInfo.RootDiskSize == "default" {
		nodeOSDiskSize = nil
	} else {
		osDiskSize, err := strconv.Atoi(nodeGroupInfo.RootDiskSize)
		if err != nil {
			return armcontainerservice.ManagedClusterAgentPoolProfile{}, errors.New("invalid NodeGroup RootDiskSize")
		}
		osDiskSizeInt32 := int32(osDiskSize)
		nodeOSDiskSize = &osDiskSizeInt32
	}

	targetZones := []*string{&ac.Region.Zone}

	agentPoolProfile := armcontainerservice.ManagedClusterAgentPoolProfile{
		Name:               &nodeGroupInfo.IId.NameId,
		Count:              toInt32Ptr(nodeGroupInfo.DesiredNodeSize),
		MinCount:           toInt32Ptr(nodeGroupInfo.MinNodeSize),
		MaxCount:           toInt32Ptr(nodeGroupInfo.MaxNodeSize),
		VMSize:             &nodeGroupInfo.VMSpecName,
		OSDiskSizeGB:       nodeOSDiskSize,
		OSType:             (*armcontainerservice.OSType)(toStrPtr(string(armcontainerservice.OSTypeLinux))),
		Type:               (*armcontainerservice.AgentPoolType)(toStrPtr(string(armcontainerservice.AgentPoolTypeVirtualMachineScaleSets))),
		MaxPods:            toInt32Ptr(maxPodCount),
		Mode:               (*armcontainerservice.AgentPoolMode)(toStrPtr(string(armcontainerservice.AgentPoolModeSystem))), // User? System?
		AvailabilityZones:  targetZones,
		EnableNodePublicIP: toBoolPtr(true),
		EnableAutoScaling:  toBoolPtr(nodeGroupInfo.OnAutoScaling),
		// MinCount가 있으려면 true 여야함
		VnetSubnetID: subnet.ID,
	}
	if !nodeGroupInfo.OnAutoScaling {
		agentPoolProfile.MinCount = nil
		agentPoolProfile.MaxCount = nil
	}
	return agentPoolProfile, nil
}

func getSecurityGroupIdForVirtualMachineScaleSet(virtualMachineScaleSet armcompute.VirtualMachineScaleSet) (securityGroupId string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprintf("failed get securityGroup in VirtualMachineScaleSet err = %s", err.Error()))
			securityGroupId = ""
		}
	}()
	if virtualMachineScaleSet.Properties.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations != nil {
		networkInterfaceConfigurations := virtualMachineScaleSet.Properties.VirtualMachineProfile.NetworkProfile.NetworkInterfaceConfigurations
		var targetSecurityGroupId *string
		for _, niconfig := range networkInterfaceConfigurations {
			if *niconfig.Properties.Primary {
				targetSecurityGroupId = niconfig.Properties.NetworkSecurityGroup.ID
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

func getNextSecurityGroupRulePriority(sourceSecurity armnetwork.SecurityGroup) (inboundPriority int, outboundPriority int, err error) {
	inboundPriority = initPriority
	outboundPriority = initPriority
	for _, sgrule := range sourceSecurity.Properties.SecurityRules {
		if *sgrule.Properties.Direction == armnetwork.SecurityRuleDirectionInbound {
			inboundPriority = int(*sgrule.Properties.Priority)
		}
		if *sgrule.Properties.Direction == armnetwork.SecurityRuleDirectionOutbound {
			outboundPriority = int(*sgrule.Properties.Priority)
		}
	}
	inboundPriority = inboundPriority + 1
	outboundPriority = outboundPriority + 1
	return inboundPriority, outboundPriority, nil
}

func sliceSecurityGroupRuleINAndOUT(copySourceSGRules []*armnetwork.SecurityRule) (inboundRules []*armnetwork.SecurityRule, outboundRules []*armnetwork.SecurityRule, err error) {
	for _, sourcesgrule := range copySourceSGRules {
		if *sourcesgrule.Properties.Direction == armnetwork.SecurityRuleDirectionInbound {
			inboundRules = append(inboundRules, sourcesgrule)
		} else if *sourcesgrule.Properties.Direction == armnetwork.SecurityRuleDirectionOutbound {
			outboundRules = append(outboundRules, sourcesgrule)
		} else {
			return []*armnetwork.SecurityRule{}, []*armnetwork.SecurityRule{}, errors.New("invalid SecurityRules")
		}

	}
	return inboundRules, outboundRules, nil
}

func applySecurityGroupList(sourceSecurity armnetwork.SecurityGroup, targetSecurityGroupList []armnetwork.SecurityGroup, SecurityRulesClient *armnetwork.SecurityRulesClient, ctx context.Context) error {
	sourceSGRules := sourceSecurity.Properties.SecurityRules
	for _, targetsg := range targetSecurityGroupList {
		copySourceSGRules := sourceSGRules
		sgresourceGroup, _ := getResourceGroupById(*targetsg.ID)
		inboundsgRules, outboundsgRules, err := sliceSecurityGroupRuleINAndOUT(copySourceSGRules)
		if err != nil {
			return err
		}
		for _, inboundsgRule := range inboundsgRules {
			update := inboundsgRule
			update.Properties.Priority = inboundsgRule.Properties.Priority
			if *update.Properties.Priority >= 500 {
				// AKS baseRule 회피
				priority := *inboundsgRule.Properties.Priority + 100
				update.Properties.Priority = &priority
			}
			poller, err := SecurityRulesClient.BeginCreateOrUpdate(ctx, sgresourceGroup, *targetsg.Name, *inboundsgRule.Name, *update, nil)
			if err != nil {
				return err
			}
			_, err = poller.PollUntilDone(ctx, nil)
			if err != nil {
				return err
			}
		}
		for _, outboundsgRule := range outboundsgRules {
			update := outboundsgRule
			update.Properties.Priority = outboundsgRule.Properties.Priority
			poller, err := SecurityRulesClient.BeginCreateOrUpdate(ctx, sgresourceGroup, *targetsg.Name, *outboundsgRule.Name, *update, nil)
			if err != nil {
				return err
			}
			_, err = poller.PollUntilDone(ctx, nil)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getClusterAgentPoolSecurityGroupId(cluster *armcontainerservice.ManagedCluster, agentPoolsClient *armcontainerservice.AgentPoolsClient, virtualMachineScaleSetsClient *armcompute.VirtualMachineScaleSetsClient, ctx context.Context) ([]string, error) {
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

func getClusterAgentPoolSecurityGroup(cluster *armcontainerservice.ManagedCluster, agentPoolsClient *armcontainerservice.AgentPoolsClient, securityGroupsClient *armnetwork.SecurityGroupsClient, virtualMachineScaleSetsClient *armcompute.VirtualMachineScaleSetsClient, ctx context.Context) ([]armnetwork.SecurityGroup, error) {
	filteredNodePoolPairList, err := getRawNodePoolPairList(cluster, agentPoolsClient, virtualMachineScaleSetsClient, ctx)
	if err != nil {
		return make([]armnetwork.SecurityGroup, 0), errors.New(fmt.Sprintf("failed apply securityGroup err = %s", err.Error()))
	}

	sgmap := make(map[string]armnetwork.SecurityGroup)
	for _, poolPair := range filteredNodePoolPairList {
		sgid, err := getSecurityGroupIdForVirtualMachineScaleSet(poolPair.virtualMachineScaleSet)
		if err != nil {
			return make([]armnetwork.SecurityGroup, 0), errors.New(fmt.Sprintf("failed apply securityGroup err = %s", err.Error()))
		}
		sgName, _ := getNameById(sgid, AzureSecurityGroups)
		sgresourceGroup, _ := getResourceGroupById(sgid)
		_, exist := sgmap[sgName]
		if !exist {
			sg, err := getRawSecurityGroup(irs.IID{NameId: sgName}, securityGroupsClient, ctx, sgresourceGroup)
			if err != nil {
				return make([]armnetwork.SecurityGroup, 0), errors.New(fmt.Sprintf("failed apply securityGroup err = %s", err.Error()))
			}
			sgmap[sgName] = *sg
		}
	}
	targetSecurityGroupList := make([]armnetwork.SecurityGroup, 0, len(sgmap))
	for _, sg := range sgmap {
		targetSecurityGroupList = append(targetSecurityGroupList, sg)
	}
	return targetSecurityGroupList, nil
}

func applySecurityGroup(clusterIID irs.IID, sourceSecurityGroupIID irs.IID, clusterBaseSecurityGroup armnetwork.SecurityGroup, managedClustersClient *armcontainerservice.ManagedClustersClient, securityGroupsClient *armnetwork.SecurityGroupsClient, securityRulesClient *armnetwork.SecurityRulesClient, ctx context.Context, credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo) error {
	//clusterResourceGroup, err := getResourceGroupById(*cluster.ID)
	//if err != nil {
	//	return errors.New(fmt.Sprintf("failed get clusterResourceGroup err = %s", err.Error()))
	//}
	sourceSecurityGroup, err := getRawSecurityGroup(sourceSecurityGroupIID, securityGroupsClient, ctx, regionInfo.Region)
	if err != nil {
		return errors.New(fmt.Sprintf("failed apply securityGroup err = %s", err.Error()))
	}
	//baseSecurityGroup, err := waitingClusterBaseSecurityGroup(clusterIID, managedClustersClient, securityGroupsClient, ctx, credentialInfo, regionInfo)
	//targetSecurityGroupList, err := getClusterAgentPoolSecurityGroup(cluster, agentPoolsClient, securityGroupsClient, virtualMachineScaleSetsClient, ctx)
	//if err != nil {
	//	return errors.New(fmt.Sprintf("failed get BaseSecurityGroup by Cluster err = %s", err.Error()))
	//}
	err = applySecurityGroupList(*sourceSecurityGroup, []armnetwork.SecurityGroup{clusterBaseSecurityGroup}, securityRulesClient, ctx)
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

func menualScaleModechangeNodeGroupScaling(cluster armcontainerservice.ManagedCluster, agentPool armcontainerservice.AgentPool, desiredNodeSize int, minNodeSize int, maxNodeSize int, managedClustersClient *armcontainerservice.ManagedClustersClient, agentPoolsClient *armcontainerservice.AgentPoolsClient, virtualMachineScaleSetsClient *armcompute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *armcompute.VirtualMachineScaleSetVMsClient, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context) (irs.NodeGroupInfo, error) {
	err := checkmanualScaleModeNodeGroupScaleValid(minNodeSize, maxNodeSize)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed scalingChange agentPool err = %s", err.Error()))
	}
	updateAgentPool := agentPool
	desiredNodeSizeInt32 := int32(desiredNodeSize)
	updateAgentPool.Properties.Count = &desiredNodeSizeInt32
	_, err = agentPoolsClient.BeginCreateOrUpdate(ctx, region.Region, *cluster.Name, *agentPool.Name, updateAgentPool, nil)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed scalingChange agentPool err = %s", err.Error()))
	}
	newCluster, err := getRawCluster(irs.IID{NameId: *cluster.Name}, managedClustersClient, ctx, credentialInfo, region)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed get agentPool err = %s", err.Error()))
	}
	nodeGroupInfo, err := getNodeGroupInfoSpecifiedNodePool(&newCluster, *agentPool.Name, agentPoolsClient, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, ctx)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed get agentPool err = %s", err.Error()))
	}
	nodeGroupInfo.Status = irs.NodeGroupUpdating
	return nodeGroupInfo, nil
}

func autoScaleModechangeNodeGroupScaling(cluster armcontainerservice.ManagedCluster, agentPool armcontainerservice.AgentPool, desiredNodeSize int, minNodeSize int, maxNodeSize int, managedClustersClient *armcontainerservice.ManagedClustersClient, agentPoolsClient *armcontainerservice.AgentPoolsClient, virtualMachineScaleSetsClient *armcompute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *armcompute.VirtualMachineScaleSetVMsClient, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context) (irs.NodeGroupInfo, error) {
	err := checkAutoScaleModeNodeGroupScaleValid(desiredNodeSize, minNodeSize, maxNodeSize)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed scalingChange agentPool err = %s", err.Error()))
	}
	updateAgentPool := agentPool
	minNodeSizeInt32 := int32(minNodeSize)
	maxNodeSizeInt32 := int32(maxNodeSize)
	desiredNodeSizeInt32 := int32(desiredNodeSize)
	updateAgentPool.Properties.MinCount = &minNodeSizeInt32
	updateAgentPool.Properties.MaxCount = &maxNodeSizeInt32
	updateAgentPool.Properties.Count = &desiredNodeSizeInt32
	_, err = agentPoolsClient.BeginCreateOrUpdate(ctx, region.Region, *cluster.Name, *agentPool.Name, updateAgentPool, nil)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed scalingChange agentPool err = %s", err.Error()))
	}

	newCluster, err := getRawCluster(irs.IID{NameId: *cluster.Name}, managedClustersClient, ctx, credentialInfo, region)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed get agentPool err = %s", err.Error()))
	}
	nodeGroupInfo, err := getNodeGroupInfoSpecifiedNodePool(&newCluster, *agentPool.Name, agentPoolsClient, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, ctx)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed get agentPool err = %s", err.Error()))
	}
	nodeGroupInfo.Status = irs.NodeGroupUpdating
	return nodeGroupInfo, nil
}

func changeNodeGroupScaling(cluster armcontainerservice.ManagedCluster, nodeGroupIID irs.IID, desiredNodeSize int, minNodeSize int, maxNodeSize int, managedClustersClient *armcontainerservice.ManagedClustersClient, agentPoolsClient *armcontainerservice.AgentPoolsClient, virtualMachineScaleSetsClient *armcompute.VirtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient *armcompute.VirtualMachineScaleSetVMsClient, credentialInfo idrv.CredentialInfo, region idrv.RegionInfo, ctx context.Context) (irs.NodeGroupInfo, error) {
	if nodeGroupIID.NameId == "" && nodeGroupIID.SystemId == "" {
		return irs.NodeGroupInfo{}, errors.New("failed scalingChange agentPool err = invalid NodeGroup NameId")
	}

	var agentPoolList []*armcontainerservice.AgentPool

	pager := agentPoolsClient.NewListPager(region.Region, *cluster.Name, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed scalingChange agentPool err = %s", err.Error()))
		}

		for _, agentPool := range page.Value {
			agentPoolList = append(agentPoolList, agentPool)
		}
	}

	var targetAgentPool *armcontainerservice.AgentPool
	for _, agentPool := range agentPoolList {
		if nodeGroupIID.NameId == "" {
			if *agentPool.ID == nodeGroupIID.SystemId {
				targetAgentPool = agentPool
				break
			}
		} else {
			if *agentPool.Name == nodeGroupIID.NameId {
				targetAgentPool = agentPool
				break
			}
		}
	}
	if targetAgentPool == nil {
		return irs.NodeGroupInfo{}, errors.New("failed scalingChange agentPool err = not Exist NodeGroup")
	}
	if *targetAgentPool.Properties.EnableAutoScaling {
		// AutoScale
		return autoScaleModechangeNodeGroupScaling(cluster, *targetAgentPool, desiredNodeSize, minNodeSize, maxNodeSize, managedClustersClient, agentPoolsClient, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, credentialInfo, region, ctx)
	} else {
		// MenualScale
		return menualScaleModechangeNodeGroupScaling(cluster, *targetAgentPool, desiredNodeSize, minNodeSize, maxNodeSize, managedClustersClient, agentPoolsClient, virtualMachineScaleSetsClient, virtualMachineScaleSetVMsClient, credentialInfo, region, ctx)
	}
}

func autoScalingChange(cluster armcontainerservice.ManagedCluster, nodeGroupIID irs.IID, autoScalingSet bool, agentPoolsClient *armcontainerservice.AgentPoolsClient, region idrv.RegionInfo, ctx context.Context) error {
	// exist Check
	if nodeGroupIID.NameId == "" && nodeGroupIID.SystemId == "" {
		return errors.New("failed autoScalingChange agentPool err = invalid NodeGroup NameId")
	}

	var agentPoolList []*armcontainerservice.AgentPool

	pager := agentPoolsClient.NewListPager(region.Region, *cluster.Name, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return errors.New(fmt.Sprintf("failed autoScalingChange agentPool err = %s", err.Error()))
		}

		for _, agentPool := range page.Value {
			agentPoolList = append(agentPoolList, agentPool)
		}
	}

	var targetAgentPool *armcontainerservice.AgentPool
	for _, agentPool := range agentPoolList {
		if nodeGroupIID.NameId == "" {
			if *agentPool.ID == nodeGroupIID.SystemId {
				targetAgentPool = agentPool
				break
			}
		} else {
			if *agentPool.Name == nodeGroupIID.NameId {
				targetAgentPool = agentPool
				break
			}
		}
	}
	if targetAgentPool == nil {
		return errors.New("failed autoScalingChange agentPool err = not Exist NodeGroup")
	}
	if *targetAgentPool.Properties.EnableAutoScaling == autoScalingSet {
		return errors.New("failed autoScalingChange agentPool err = already autoScaling status Equal")
	}
	updateAgentPool := *targetAgentPool
	if *updateAgentPool.Properties.ProvisioningState != "Succeeded" {
		return errors.New(fmt.Sprintf("failed autoScalingChange agentPool err = The status of the Agent Pool is currently %s. You cannot change the Agent Pool at this time.", *updateAgentPool.Properties.ProvisioningState))
	}
	if !autoScalingSet {
		// False
		updateAgentPool.Properties.MinCount = nil
		updateAgentPool.Properties.MaxCount = nil
	} else {
		// TODO autoScale 시, Min, Max 값 필요
		minCount := int32(1)
		updateAgentPool.Properties.MinCount = &minCount
		updateAgentPool.Properties.MaxCount = updateAgentPool.Properties.Count
	}
	updateAgentPool.Properties.EnableAutoScaling = &autoScalingSet
	_, err := agentPoolsClient.BeginCreateOrUpdate(ctx, region.Region, *cluster.Name, *targetAgentPool.Name, updateAgentPool, nil)
	if err != nil {
		return errors.New(fmt.Sprintf("failed autoScalingChange agentPool err = %s", err.Error()))
	}
	return nil
}

func deleteNodeGroup(cluster armcontainerservice.ManagedCluster, nodeGroupIID irs.IID, agentPoolsClient *armcontainerservice.AgentPoolsClient, region idrv.RegionInfo, ctx context.Context) error {
	// exist Check
	if nodeGroupIID.NameId == "" && nodeGroupIID.SystemId == "" {
		return errors.New("failed remove agentPool err = invalid NodeGroup NameId")
	}

	var agentPoolList []*armcontainerservice.AgentPool

	pager := agentPoolsClient.NewListPager(region.Region, *cluster.Name, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return errors.New(fmt.Sprintf("failed autoScalingChange agentPool err = %s", err.Error()))
		}

		for _, agentPool := range page.Value {
			agentPoolList = append(agentPoolList, agentPool)
		}
	}

	existNodeGroup := false
	for _, agentPool := range agentPoolList {
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
	_, err := agentPoolsClient.BeginDelete(ctx, region.Region, *cluster.Name, nodeGroupIID.NameId, nil)
	if err != nil {
		return errors.New(fmt.Sprintf("failed remove agentPool err = %s", err.Error()))
	}
	return nil
}
func addNodeGroupPool(cluster *armcontainerservice.ManagedCluster, nodeGroup irs.NodeGroupInfo, ac *AzureClusterHandler) (irs.NodeGroupInfo, error) {
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
		clusterSShKeyId := GetSshKeyIdByName(ac.CredentialInfo, ac.Region, clusterNodeSSHkey.NameId)
		if nodeGroup.KeyPairIID.SystemId != clusterSShKeyId {
			return irs.NodeGroupInfo{}, errors.New("The SSHkey in the Azure Cluster NodeGroup must all be the same")
		}
	}

	var agentPoolList []*armcontainerservice.AgentPool

	pager := ac.AgentPoolsClient.NewListPager(ac.Region.Region, *cluster.Name, nil)

	for pager.More() {
		page, err := pager.NextPage(ac.Ctx)
		if err != nil {
			return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed add agentPool err = %s", err.Error()))
		}

		for _, agentPool := range page.Value {
			agentPoolList = append(agentPoolList, agentPool)
		}
	}

	existNodeGroup := false
	for _, agentPool := range agentPoolList {
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
	subnetId, err := getSubnetIdByAgentPoolProfiles(cluster.Properties.AgentPoolProfiles)
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
	resp, err := ac.SubnetClient.Get(ac.Ctx, ac.Region.Region, vpcName, subnetName, nil)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed addNodeGroupPool err = %s", err.Error()))
	}
	// Add AgentPoolProfiles
	agentPoolProfileProperties, err := generateAgentPoolProfileProperties(nodeGroup, resp.Subnet, ac)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed add agentPool err = %s", err.Error()))
	}
	_, err = ac.AgentPoolsClient.BeginCreateOrUpdate(ac.Ctx, ac.Region.Region, *cluster.Name, nodeGroup.IId.NameId, armcontainerservice.AgentPool{Properties: &agentPoolProfileProperties}, nil)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed add agentPool err = %s", err.Error()))
	}
	nodePoolPair, err := waitingSpecifiedNodePoolPair(cluster, nodeGroup.IId.NameId, ac.AgentPoolsClient, ac.VirtualMachineScaleSetsClient, ac.Ctx)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed add agentPool err = %s", err.Error()))
	}
	clusterSSHKey, err := getClusterSSHKey(cluster)
	if err != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("failed add agentPool err = %s", err.Error()))
	}
	info := convertNodePairToNodeInfo(nodePoolPair, clusterSSHKey, ac.VirtualMachineScaleSetVMsClient, *cluster.Properties.NodeResourceGroup, ac.Ctx)
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

func getClusterSecurityGroup(cluster *armcontainerservice.ManagedCluster, securityGroupsClient *armnetwork.SecurityGroupsClient, ctx context.Context) ([]*armnetwork.SecurityGroup, error) {
	var securityGroupList []*armnetwork.SecurityGroup

	if cluster.Properties.NodeResourceGroup == nil {
		return securityGroupList, errors.New("failed get Cluster Managed ResourceGroup")
	}

	pager := securityGroupsClient.NewListPager(*cluster.Properties.NodeResourceGroup, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return securityGroupList, err
		}

		for _, sg := range page.Value {
			securityGroupList = append(securityGroupList, sg)
		}
	}

	return securityGroupList, nil
}

func waitingClusterBaseSecurityGroup(createdClusterIID irs.IID, managedClustersClient *armcontainerservice.ManagedClustersClient, securityGroupsClient *armnetwork.SecurityGroupsClient, ctx context.Context, credentialInfo idrv.CredentialInfo, regionInfo idrv.RegionInfo) (armnetwork.SecurityGroup, error) {
	// exist Cluster
	apiCallCount := 0
	maxAPICallCount := 240
	var waitingErr error
	var rawCluster armcontainerservice.ManagedCluster
	var err error

	for {
		rawCluster, err = getRawCluster(createdClusterIID, managedClustersClient, ctx, credentialInfo, regionInfo)
		if err == nil {
			break
		}
		apiCallCount++
		if apiCallCount >= maxAPICallCount {
			waitingErr = errors.New("failed get Cluster: The maximum number of verification requests has been exceeded while waiting for the creation of that resource, " +
				"err = " + err.Error())
			return armnetwork.SecurityGroup{}, waitingErr
		}
		time.Sleep(1 * time.Second)
	}

	// exist basicSecurity in clusterResourceGroup
	apiCallCount = 0
	if rawCluster.Properties.NodeResourceGroup == nil || *rawCluster.Properties.NodeResourceGroup == "" {
		return armnetwork.SecurityGroup{}, errors.New("failed get Cluster Managed ResourceGroup err = Invalid value of NodeResourceGroup for cluster")
	}
	var clusterManagedResourceGroup = *rawCluster.Properties.NodeResourceGroup
	var baseSecurityGroup armnetwork.SecurityGroup
	for {
		var securityGroupList []*armnetwork.SecurityGroup

		pager := securityGroupsClient.NewListPager(regionInfo.Region, nil)

		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return armnetwork.SecurityGroup{}, errors.New("failed get Cluster Managed ResourceGroup err = " + err.Error())
			}

			for _, securityGroup := range page.Value {
				securityGroupList = append(securityGroupList, securityGroup)
			}
		}

		var isSGExist bool
		for _, sg := range securityGroupList {
			if sg.Tags != nil {
				val, exist := sg.Tags[OwnerClusterKey]
				if exist && val != nil && *val == *rawCluster.Name {
					baseSecurityGroup = *sg
					isSGExist = true
					break
				}
			}
		}
		if isSGExist {
			break
		}

		apiCallCount++
		if apiCallCount >= maxAPICallCount {
			waitingErr = errors.New("failed get Cluster BaseSecurityGroup err = The maximum number of verification requests has been exceeded while waiting for the creation of that resource")
			break
		}
		time.Sleep(1 * time.Second)
	}
	if waitingErr != nil {
		return armnetwork.SecurityGroup{}, waitingErr
	}
	// check ClusterBaseRule..
	apiCallCount = 0
	for {
		var isTcp80Exist bool
		var isTcp443Exist bool
		var defaultRulesExist bool
		resp, err := securityGroupsClient.Get(ctx, clusterManagedResourceGroup, *baseSecurityGroup.Name, nil)
		if err == nil && resp.SecurityGroup.Properties != nil {
			for _, rule := range resp.SecurityGroup.Properties.SecurityRules {
				if rule.Properties.Direction == nil || rule.Properties.Protocol == nil {
					cblogger.Warn("Failed to get security group info of the cluster!")
					continue
				}
				if *rule.Properties.Direction == armnetwork.SecurityRuleDirectionInbound && *rule.Properties.Protocol == armnetwork.SecurityRuleProtocolTCP {
					for _, portRange := range rule.Properties.DestinationPortRanges {
						if portRange == nil {
							cblogger.Warn("Failed to get security group rule's port range of the cluster!")
							continue
						}

						if *portRange == "80" {
							isTcp80Exist = true
						} else if *portRange == "443" {
							isTcp443Exist = true
						}
					}

					if isTcp80Exist && isTcp443Exist {
						defaultRulesExist = true
						break
					}

					if rule.Properties.DestinationPortRange != nil {
						if *rule.Properties.DestinationPortRange == "80" {
							isTcp80Exist = true
						} else if *rule.Properties.DestinationPortRange == "443" {
							isTcp443Exist = true
						}

						if isTcp80Exist && isTcp443Exist {
							defaultRulesExist = true
							break
						}
					}
				}
			}
		}

		if defaultRulesExist {
			break
		}

		apiCallCount++
		if apiCallCount >= maxAPICallCount {
			waitingErr = errors.New("failed wait creating BaseRule in Cluster BaseSecurityGroup err = The maximum number of verification requests has been exceeded while waiting for the creation of that resource")
			break
		}
		time.Sleep(1 * time.Second)
	}
	if waitingErr != nil {
		return armnetwork.SecurityGroup{}, waitingErr
	}
	return baseSecurityGroup, nil
	// exist basicSecurity
}

func convertNodePairToNodeInfo(nodePoolPair NodePoolPair, clusterCommonSSHKey irs.IID, virtualMachineScaleSetVMsClient *armcompute.VirtualMachineScaleSetVMsClient, resourceGroupManagedK8s string, ctx context.Context) irs.NodeGroupInfo {
	scaleSet := nodePoolPair.virtualMachineScaleSet
	agentPool := nodePoolPair.AgentPool
	rootDiskType := ""
	if reflect.ValueOf(scaleSet.Properties.VirtualMachineProfile.StorageProfile.OSDisk.ManagedDisk.StorageAccountType).String() != "" {
		rootDiskType = GetVMDiskInfoType(scaleSet.Properties.VirtualMachineProfile.StorageProfile.OSDisk.ManagedDisk.StorageAccountType)
	}
	VMSpecName := ""
	if !reflect.ValueOf(scaleSet.SKU.Name).IsNil() {
		VMSpecName = *scaleSet.SKU.Name
	}
	RootDiskSize := ""
	if !reflect.ValueOf(scaleSet.Properties.VirtualMachineProfile.StorageProfile.OSDisk.DiskSizeGB).IsNil() {
		RootDiskSize = strconv.Itoa(int(*scaleSet.Properties.VirtualMachineProfile.StorageProfile.OSDisk.DiskSizeGB))
	}
	OnAutoScaling := false
	if !reflect.ValueOf(agentPool.Properties.EnableAutoScaling).IsNil() {
		OnAutoScaling = *agentPool.Properties.EnableAutoScaling
	}
	DesiredNodeSize := 0
	if !reflect.ValueOf(agentPool.Properties.Count).IsNil() {
		DesiredNodeSize = int(*agentPool.Properties.Count)
	}
	MinNodeSize := 0
	if !reflect.ValueOf(agentPool.Properties.MinCount).IsNil() {
		MinNodeSize = int(*agentPool.Properties.MinCount)
	}
	MaxNodeSize := 0
	if !reflect.ValueOf(agentPool.Properties.MaxCount).IsNil() {
		MaxNodeSize = int(*agentPool.Properties.MaxCount)
	}
	imageName := ""
	if !reflect.ValueOf(scaleSet.Properties.VirtualMachineProfile.StorageProfile.ImageReference.ID).IsNil() {
		imageName = *scaleSet.Properties.VirtualMachineProfile.StorageProfile.ImageReference.ID
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

	var scaleSetList []*armcompute.VirtualMachineScaleSetVM

	pager := virtualMachineScaleSetVMsClient.NewListPager(resourceGroupManagedK8s, *scaleSet.Name, nil)

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			break
		}

		for _, scSet := range page.Value {
			scaleSetList = append(scaleSetList, scSet)
		}
	}

	for _, vm := range scaleSetList {
		vmIIds = append(vmIIds, irs.IID{*vm.Name, *vm.ID})
	}

	nodeInfo.Nodes = vmIIds

	return nodeInfo
}

func getClusterSSHKey(cluster *armcontainerservice.ManagedCluster) (irs.IID, error) {
	sshkeyName, sshKeyExist := cluster.Tags[ClusterNodeSSHKeyKey]
	keyPairIID := irs.IID{}

	if sshKeyExist && sshkeyName != nil {
		keyPairIID.NameId = *sshkeyName
		clusterSubscriptionsById, subscriptionsErr := getSubscriptionsById(*cluster.ID)
		clusterResourceGroup, err := getResourceGroupById(*cluster.ID)
		if err != nil {
			return irs.IID{}, errors.New("failed get Cluster Node SSHKey err = invalid Cluster ID")
		}
		if subscriptionsErr == nil {
			keyPairIID.SystemId = GetSshKeyIdByName(idrv.CredentialInfo{SubscriptionId: clusterSubscriptionsById}, idrv.RegionInfo{
				Region: clusterResourceGroup, // Azure uses region as ResourceGroup
			}, *sshkeyName)
		}
	}
	return keyPairIID, nil
}

func waitingSpecifiedNodePoolPair(cluster *armcontainerservice.ManagedCluster, agentPoolName string, agentPoolsClient *armcontainerservice.AgentPoolsClient, virtualMachineScaleSetsClient *armcompute.VirtualMachineScaleSetsClient, ctx context.Context) (NodePoolPair, error) {
	clusterResourceGroup, err := getResourceGroupById(*cluster.ID)
	if err != nil {
		return NodePoolPair{}, errors.New(fmt.Sprintf("failed get clusterResourceGroup err = %s", err.Error()))
	}
	if cluster.Properties.NodeResourceGroup == nil {
		return NodePoolPair{}, errors.New(fmt.Sprintf("failed get Cluster Managed ResourceGroup err = Invalid value of NodeResourceGroup for cluster"))
	}
	clusterManagedResourceGroup := ""
	clusterManagedResourceGroup = *cluster.Properties.NodeResourceGroup
	apiCallCount := 0
	maxAPICallCount := 100
	returnNodePoolPair := NodePoolPair{}
	// var targetAgentPool *armcontainerservice.AgentPool
	var waitingErr error
	for {
		resp, err := agentPoolsClient.Get(ctx, clusterResourceGroup, *cluster.Name, agentPoolName, nil)
		if err == nil {
			returnNodePoolPair.AgentPool = resp.AgentPool
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
		var scaleSetList []*armcompute.VirtualMachineScaleSet

		var page armcompute.VirtualMachineScaleSetsClientListResponse
		var err error

		pager := virtualMachineScaleSetsClient.NewListPager(clusterManagedResourceGroup, nil)

		for pager.More() {
			page, err = pager.NextPage(ctx)
			if err != nil {
				break
			}

			for _, scaleSet := range page.Value {
				scaleSetList = append(scaleSetList, scaleSet)
			}
		}

		if err == nil {
			scaleSetCheck := false
			for _, scaleSet := range scaleSetList {
				val, exist := scaleSet.Tags[ScaleSetOwnerKey]
				if exist && *val == agentPoolName {
					returnNodePoolPair.virtualMachineScaleSet = *scaleSet
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

func getClusterAccessInfo(cluster *armcontainerservice.ManagedCluster, managedClustersClient *armcontainerservice.ManagedClustersClient, ctx context.Context) (accessInfo irs.AccessInfo, err error) {
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
	profile, err := managedClustersClient.GetAccessProfile(ctx, clusterResourceGroup, *cluster.Name, ClusterAdminKey, nil)
	if err != nil {
		return irs.AccessInfo{}, errors.New(fmt.Sprintf("faild get AccessInfo err = %s", err.Error()))
	}
	accessInfo.Kubeconfig = string(profile.Properties.KubeConfig)

	kubeConfig := ServerKubeConfig{}
	err = yaml.Unmarshal(profile.Properties.KubeConfig, &kubeConfig)
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
