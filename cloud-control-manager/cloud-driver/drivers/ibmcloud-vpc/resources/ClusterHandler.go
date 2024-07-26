package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/globaltaggingv1"
	"github.com/IBM/platform-services-go-sdk/resourcemanagerv2"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibmcloud-vpc/utils/kubernetesserviceapiv1"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/go-openapi/strfmt"
	"github.com/hashicorp/go-version"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strings"
	"sync"
	"time"
)

const (
	// Resource Names
	DefaultResourceGroup              = "Default"
	AutoscalerAddon                   = "cluster-autoscaler"
	ConfigMapNamespace                = "kube-system"
	AutoscalerConfigMap               = "iks-ca-configmap"
	AutoscalerConfigMapOptionProperty = "workerPoolsConfig.json"

	// Error Codes
	RetrieveUnableErr      = "Not visible in IBMCloud-VPC"
	GetKubeConfigErr       = "Get Kube Config Error"
	GetAutoScalerConfigErr = "Get Autoscaler Config Map Error"

	// Retry Counts
	EnableAutoScalerRetry  = 120 // minutes
	DisableAutoScalerRetry = 120 // minutes
	InitSecurityGroupRetry = 120 // minutes
	RestoreDefaultSGRetry  = 120 // minutes
	UpgradeMasterRetry     = 120 // minutes

	// Status tags
	AutoScalerStatus    = "CB-SPIDER-PMKS-AUTOSCALER-STATUS:"
	SecurityGroupStatus = "CB-SPIDER-PMKS-SECURITYGROUP-STATUS:"
	MasterUpgradeStatus = "CB-SPIDER-PMKS-MASTERUPGRADE-STATUS:"

	// State Codes
	WAITING           = "WAITING"
	DEPLOYING         = "DEPLOYING"
	UPGRADE_DEPLOYING = "UPGRADE-DEPLOYING"
	ACTIVE            = "ACTIVE"
	UNINSTALLING      = "UNINSTALLING"
	FAILED            = "FAILED"
	INITIALIZING      = "INITIALIZING"
	INITIALIZED       = "INITIALIZED"
	UPGRADING         = "UPGRADING"
)

var autoSaclerStates []string
var securityGroupStates []string
var masterUpgradeStates []string

func init() {
	autoSaclerStates = []string{WAITING, DEPLOYING, UPGRADE_DEPLOYING, ACTIVE, UNINSTALLING, FAILED}
	for i, state := range autoSaclerStates {
		autoSaclerStates[i] = fmt.Sprintf("%s%s", AutoScalerStatus, state)
	}
	securityGroupStates = []string{INITIALIZING, INITIALIZED, FAILED}
	for i, state := range securityGroupStates {
		securityGroupStates[i] = fmt.Sprintf("%s%s", SecurityGroupStatus, state)
	}
	masterUpgradeStates = []string{WAITING, UPGRADING}
	for i, state := range masterUpgradeStates {
		masterUpgradeStates[i] = fmt.Sprintf("%s%s", MasterUpgradeStatus, state)
	}
}

type IbmClusterHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	Ctx            context.Context
	VpcService     *vpcv1.VpcV1
	ClusterService *kubernetesserviceapiv1.KubernetesServiceApiV1
	TaggingService *globaltaggingv1.GlobalTaggingV1
}

var defaultResourceGroupId string

func (ic *IbmClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (irs.ClusterInfo, error) {
	hiscallInfo := GetCallLogScheme(ic.Region, call.CLUSTER, clusterReqInfo.IId.NameId, "CreateCluster()")
	start := call.Start()

	// validation
	validationErr := ic.validateAtCreateCluster(clusterReqInfo)
	if validationErr != nil {
		cblogger.Error(validationErr)
		LoggingError(hiscallInfo, validationErr)
		return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Create Cluster. err = %s", validationErr))
	}

	// get resource group id
	resourceGroupId, getResourceGroupErr := ic.getDefaultResourceGroupId()
	if getResourceGroupErr != nil {
		cblogger.Error(getResourceGroupErr)
		LoggingError(hiscallInfo, getResourceGroupErr)
		return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Create Cluster. err = %s", getResourceGroupErr))
	}

	// get vpc info
	vpcHandler := IbmVPCHandler{
		CredentialInfo: ic.CredentialInfo,
		Region:         ic.Region,
		VpcService:     ic.VpcService,
		Ctx:            ic.Ctx,
	}
	vpcInfo, getVpcInfoErr := vpcHandler.GetVPC(clusterReqInfo.Network.VpcIID)
	if getVpcInfoErr != nil {
		cblogger.Error(getVpcInfoErr)
		LoggingError(hiscallInfo, getVpcInfoErr)
		return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Create Cluster. err = %s", getVpcInfoErr))
	}

	// get subnet info
	subnetInfo, getSubnetInfoErr := ic.validateAndGetSubnetInfo(clusterReqInfo.Network)
	if getSubnetInfoErr != nil {
		cblogger.Error(getSubnetInfoErr)
		LoggingError(hiscallInfo, getSubnetInfoErr)
		return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Create Cluster. err = %s", getSubnetInfoErr))
	}

	// check exists
	_, _, getClusterErr := ic.ClusterService.VpcGetClusterWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcGetClusterOptions{
		Cluster:            core.StringPtr(clusterReqInfo.IId.NameId),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
		ShowResources:      core.StringPtr("false"),
	})
	if getClusterErr != nil && getClusterErr.Error() == "Not Found" {
		// get first worker pool for cluster creation
		workerPool := ic.getWorkerPoolFromNodeGroupInfo(clusterReqInfo.NodeGroupList[0], vpcInfo.IId.SystemId, subnetInfo.IId.SystemId)

		// create cluster if not exists
		_, _, createClusterErr := ic.ClusterService.VpcCreateClusterWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcCreateClusterOptions{
			DisablePublicServiceEndpoint: core.BoolPtr(false),
			KubeVersion:                  core.StringPtr(clusterReqInfo.Version),
			Name:                         core.StringPtr(clusterReqInfo.IId.NameId),
			Provider:                     core.StringPtr("vpc-gen2"),
			WorkerPool:                   &workerPool,
			XAuthResourceGroup:           core.StringPtr(resourceGroupId),
		})
		if createClusterErr != nil {
			cblogger.Error(createClusterErr)
			LoggingError(hiscallInfo, createClusterErr)
			return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Create Cluster. err = %s", createClusterErr))
		}

		rawVpcInfo, _, getRawVpcErr := ic.VpcService.GetVPCWithContext(ic.Ctx, &vpcv1.GetVPCOptions{
			ID: core.StringPtr(vpcInfo.IId.SystemId),
		})
		if getRawVpcErr != nil {
			cblogger.Error(getRawVpcErr)
			LoggingError(hiscallInfo, getRawVpcErr)
			_, _ = ic.DeleteCluster(clusterReqInfo.IId)
			return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Create Cluster. err = %s", getRawVpcErr))
		}

		sgHander := IbmSecurityHandler{
			CredentialInfo: ic.CredentialInfo,
			Region:         ic.Region,
			Ctx:            ic.Ctx,
			VpcService:     ic.VpcService,
		}

		// restore VPC default security group
		// VPC default security group lost rules for unknown reasons while creating a cluster with API
		// It makes communication failure between worker and master nodes in the cluster and worker status reaches failure
		go func() {
			cnt := 0
			for cnt < RestoreDefaultSGRetry {
				isSkip := false
				brokenVpcDefaultSg, getBrokenVpcDefaultSgErr := sgHander.GetSecurity(irs.IID{SystemId: *rawVpcInfo.DefaultSecurityGroup.ID})
				if getBrokenVpcDefaultSgErr != nil {
					isSkip = true
				} else {
					rawClusters, _, getClustersErr := ic.ClusterService.VpcGetClusterWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcGetClusterOptions{
						Cluster:            core.StringPtr(clusterReqInfo.IId.NameId),
						XAuthResourceGroup: core.StringPtr(resourceGroupId),
						ShowResources:      core.StringPtr("true"),
					})
					if getClustersErr != nil {
						isSkip = true
					}
					rawCluster := (*rawClusters)[0]
					if rawCluster.State != "deploying" {
						isSkip = true
					}
				}
				if isSkip {
					cnt++
					time.Sleep(time.Minute)
				} else {
					mandatoriyRuleList := []irs.SecurityRuleInfo{{
						Direction:  "inbound",
						IPProtocol: "tcp",
						FromPort:   "22",
						ToPort:     "22",
					}, {
						Direction:  "inbound",
						IPProtocol: "icmp",
						FromPort:   "-1",
						ToPort:     "-1",
					}, {
						Direction:  "outbound",
						IPProtocol: "all",
						FromPort:   "-1",
						ToPort:     "-1",
						CIDR:       "0.0.0.0/0",
					}}

					_, addRuleErr := sgHander.AddRules(brokenVpcDefaultSg.IId, &mandatoriyRuleList)
					if addRuleErr != nil {
						cblogger.Error(addRuleErr)
						LoggingError(hiscallInfo, addRuleErr)
						ic.DeleteCluster(clusterReqInfo.IId)
					}
					break
				}
			}
		}()

	} else if getClusterErr != nil {
		cblogger.Error(getClusterErr)
		LoggingError(hiscallInfo, getClusterErr)
		return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Create Cluster. err = %s", getClusterErr))
	}

	// get created cluster info
	rawClusters, _, getClustersErr := ic.ClusterService.VpcGetClusterWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcGetClusterOptions{
		Cluster:            core.StringPtr(clusterReqInfo.IId.NameId),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
		ShowResources:      core.StringPtr("true"),
	})
	if getClustersErr != nil {
		cblogger.Error(getClustersErr)
		LoggingError(hiscallInfo, getClustersErr)
		ic.DeleteCluster(clusterReqInfo.IId)
		return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Get Cluster. err = %s", getClustersErr))
	}
	rawCluster := (*rawClusters)[0]

	// Enable cluster-autoscaler addon and apply autoscaler option
	autoScalerErr := ic.installAutoScalerAddon(clusterReqInfo, rawCluster.Id, rawCluster.Crn, resourceGroupId, false)
	if autoScalerErr != nil {
		cblogger.Error(autoScalerErr)
		LoggingError(hiscallInfo, autoScalerErr)
		ic.DeleteCluster(clusterReqInfo.IId)
		return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Get Cluster. err = %s", autoScalerErr))
	}

	// Add remaining worker pools
	ic.createRemainingWorkerPools(clusterReqInfo, vpcInfo, subnetInfo, rawCluster, resourceGroupId)

	// Set Security Group
	ic.initSecurityGroup(clusterReqInfo, rawCluster.Id, rawCluster.Crn)

	clusterInfo, getClusterErr := ic.GetCluster(irs.IID{SystemId: rawCluster.Id})
	if getClusterErr != nil {
		cblogger.Error(getClusterErr)
		LoggingError(hiscallInfo, getClusterErr)
		return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Create Cluster. err = %s", getClusterErr))
	}

	LoggingInfo(hiscallInfo, start)

	return clusterInfo, nil
}

func (ic *IbmClusterHandler) ListCluster() ([]*irs.ClusterInfo, error) {
	hiscallInfo := GetCallLogScheme(ic.Region, call.CLUSTER, "", "ListCluster()")
	start := call.Start()

	resourceGroupId, getResourceGroupIdErr := ic.getDefaultResourceGroupId()
	if getResourceGroupIdErr != nil {
		cblogger.Error(getResourceGroupIdErr)
		LoggingError(hiscallInfo, getResourceGroupIdErr)
		return []*irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to List Cluster. err = %s", getResourceGroupIdErr))
	}

	clusterList, _, getClusterListErr := ic.ClusterService.VpcGetClustersWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcGetClustersOptions{
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
		Provider:           core.StringPtr("vpc-gen2"),
	})
	if getClusterListErr != nil {
		cblogger.Error(getClusterListErr)
		LoggingError(hiscallInfo, getClusterListErr)
		return []*irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to List Cluster. err = %s", getClusterListErr))
	}

	var wait sync.WaitGroup
	wait.Add(len(clusterList))
	var ret []*irs.ClusterInfo
	for _, cluster := range clusterList {
		go func() {
			defer wait.Done()
			irsCluster, getIrsClusterErr := ic.GetCluster(irs.IID{SystemId: *cluster.ID})
			if getIrsClusterErr == nil {
				ret = append(ret, &irsCluster)
			}
		}()
	}
	wait.Wait()

	LoggingInfo(hiscallInfo, start)

	return ret, nil
}

func (ic *IbmClusterHandler) GetCluster(clusterIID irs.IID) (irs.ClusterInfo, error) {
	hiscallInfo := GetCallLogScheme(ic.Region, call.CLUSTER, clusterIID.NameId, "GetCluster()")
	start := call.Start()

	if clusterIID.NameId == "" && clusterIID.SystemId == "" {
		return irs.ClusterInfo{}, errors.New("Failed to Get Cluster. err = invalid IID")
	}

	resourceGroupId, getResourceGroupIdErr := ic.getDefaultResourceGroupId()
	if getResourceGroupIdErr != nil {
		cblogger.Error(getResourceGroupIdErr)
		LoggingError(hiscallInfo, getResourceGroupIdErr)
		return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Get Cluster. err = %s", getResourceGroupIdErr))
	}

	var cluster string
	if clusterIID.SystemId != "" {
		cluster = clusterIID.SystemId
	} else {
		cluster = clusterIID.NameId
	}
	rawClusters, _, getClustersErr := ic.ClusterService.VpcGetClusterWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcGetClusterOptions{
		Cluster:            core.StringPtr(cluster),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
		ShowResources:      core.StringPtr("true"),
	})
	if getClustersErr != nil {
		cblogger.Error(getClustersErr)
		LoggingError(hiscallInfo, getClustersErr)
		return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Get Cluster. err = %s", getClustersErr))
	}

	for _, rawCluster := range *rawClusters {
		if rawCluster.Id == clusterIID.SystemId || rawCluster.Name == clusterIID.NameId {
			ret, getClusterInfoErr := ic.setClusterInfo(rawCluster)
			if getClusterInfoErr != nil {
				cblogger.Error(getClusterInfoErr)
				LoggingError(hiscallInfo, getClusterInfoErr)
				return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Get Cluster. err = %s", getClusterInfoErr))
			}
			LoggingInfo(hiscallInfo, start)
			return ret, nil
		}
	}

	LoggingInfo(hiscallInfo, start)
	return irs.ClusterInfo{}, nil
}

func (ic *IbmClusterHandler) DeleteCluster(clusterIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(ic.Region, call.CLUSTER, clusterIID.NameId, "DeleteCluster()")
	start := call.Start()

	if clusterIID.NameId == "" && clusterIID.SystemId == "" {
		return false, errors.New("Failed to Delete Cluster. err = invalid IID")
	}

	// get resource group id
	resourceGroupId, getResourceGroupErr := ic.getDefaultResourceGroupId()
	if getResourceGroupErr != nil {
		cblogger.Error(getResourceGroupErr)
		LoggingError(hiscallInfo, getResourceGroupErr)
		return false, errors.New(fmt.Sprintf("Failed to Delete Cluster. err = %s", getResourceGroupErr))
	}

	// check exists
	fullClusterIID, getClusterIIDErr := ic.getClusterIID(clusterIID)
	if getClusterIIDErr != nil {
		cblogger.Error(getClusterIIDErr)
		LoggingError(hiscallInfo, getClusterIIDErr)
		return false, errors.New(fmt.Sprintf("Failed to Delete Cluster. err = %s", getClusterIIDErr))
	}

	rawCluster, _, getClusterErr := ic.ClusterService.VpcGetClusterWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcGetClusterOptions{
		Cluster:            core.StringPtr(fullClusterIID.SystemId),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
		ShowResources:      core.StringPtr("false"),
	})
	if getClusterErr != nil {
		cblogger.Error(getClusterErr)
		LoggingError(hiscallInfo, getClusterErr)
		return false, errors.New(fmt.Sprintf("Failed to Delete Cluster. err = %s", getClusterErr))
	}

	// delete cluster
	_, deleteClusterErr := ic.ClusterService.RemoveClusterWithContext(ic.Ctx, &kubernetesserviceapiv1.RemoveClusterOptions{
		IdOrName:           core.StringPtr((*rawCluster)[0].Id),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
		DeleteResources:    core.StringPtr("true"),
	})
	if deleteClusterErr != nil {
		cblogger.Error(deleteClusterErr)
		LoggingError(hiscallInfo, deleteClusterErr)
		return false, errors.New(fmt.Sprintf("Failed to Delete Cluster. err = %s", deleteClusterErr))
	}

	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func (ic *IbmClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (irs.NodeGroupInfo, error) {
	hiscallInfo := GetCallLogScheme(ic.Region, call.CLUSTER, clusterIID.NameId, "AddNodeGroup()")
	start := call.Start()

	// validation
	validateErr := ic.validateAtAddNodeGroup(clusterIID, nodeGroupReqInfo)
	if validateErr != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("Failed to Add Node Group. err = %s", validateErr))
	}

	// get resource group id
	resourceGroupId, getResourceGroupErr := ic.getDefaultResourceGroupId()
	if getResourceGroupErr != nil {
		cblogger.Error(getResourceGroupErr)
		LoggingError(hiscallInfo, getResourceGroupErr)
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("Failed to Add Node Group. err = %s", getResourceGroupErr))
	}

	irsCluster, getIrsClusterErr := ic.GetCluster(clusterIID)
	if getIrsClusterErr != nil {
		cblogger.Error(getIrsClusterErr)
		LoggingError(hiscallInfo, getIrsClusterErr)
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("Failed to Add Node Group. err = %s", getIrsClusterErr))
	}
	if irsCluster.Status == irs.ClusterCreating || irsCluster.Status == irs.ClusterDeleting {
		clusterStatusErr := errors.New(fmt.Sprintf("Cannot Add Node Group at %s status", irsCluster.Status))
		cblogger.Error(clusterStatusErr)
		LoggingError(hiscallInfo, clusterStatusErr)
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("Failed to Add Node Group. err = %s", clusterStatusErr))
	}

	// Get Network.Subnet Info
	rawVpeList, _, getRawVpeListErr := ic.VpcService.ListEndpointGateways(&vpcv1.ListEndpointGatewaysOptions{
		ResourceGroupID: core.StringPtr(resourceGroupId),
	})
	if getRawVpeListErr != nil {
		cblogger.Error(getRawVpeListErr)
		LoggingError(hiscallInfo, getRawVpeListErr)
		return irs.NodeGroupInfo{}, getRawVpeListErr
	}

	var target vpcv1.EndpointGateway
	for _, rawVpe := range rawVpeList.EndpointGateways {
		if *rawVpe.Name == fmt.Sprintf("iks-%s", irsCluster.IId.SystemId) {
			target = rawVpe
		}
	}

	var subnetIID irs.IID
	for _, ip := range target.Ips {
		if strings.Contains(*ip.Name, irsCluster.IId.SystemId) {
			subnetId := strings.Split(*ip.Href, "/")[5]
			rawSubnet, _, getRawSubnetErr := ic.VpcService.GetSubnet(&vpcv1.GetSubnetOptions{
				ID: core.StringPtr(subnetId),
			})
			if getRawSubnetErr != nil {
				cblogger.Error(getRawSubnetErr)
				LoggingError(hiscallInfo, getRawSubnetErr)
				return irs.NodeGroupInfo{}, getRawSubnetErr
			}
			subnetIID.NameId = *rawSubnet.Name
			subnetIID.SystemId = *rawSubnet.ID
		}
	}

	addNodeGroupResponse, _, addNodeGroupErr := ic.ClusterService.VpcCreateWorkerPoolWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcCreateWorkerPoolOptions{
		Cluster:     core.StringPtr(irsCluster.IId.SystemId),
		Flavor:      core.StringPtr(nodeGroupReqInfo.VMSpecName),
		Isolation:   core.StringPtr("public"),
		Name:        core.StringPtr(nodeGroupReqInfo.IId.NameId),
		VpcID:       core.StringPtr(irsCluster.Network.VpcIID.SystemId),
		WorkerCount: core.Int64Ptr(int64(nodeGroupReqInfo.DesiredNodeSize)),
		Zones: []kubernetesserviceapiv1.Zone{{
			ID:       core.StringPtr(ic.Region.Zone),
			SubnetID: core.StringPtr(subnetIID.SystemId),
		}},
		Authorization:      core.StringPtr(ic.CredentialInfo.AuthToken),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
	})
	if addNodeGroupErr != nil {
		cblogger.Error(addNodeGroupErr)
		LoggingError(hiscallInfo, addNodeGroupErr)
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("Failed to Add Node Group. err = %s", addNodeGroupErr))
	}

	newNodeGroup, _, getNewNodeGroupErr := ic.ClusterService.VpcGetWorkerPoolWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcGetWorkerPoolOptions{
		Cluster:            core.StringPtr(irsCluster.IId.SystemId),
		Workerpool:         addNodeGroupResponse.WorkerPoolID,
		XRegion:            core.StringPtr(ic.Region.Region),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
	})
	if getNewNodeGroupErr != nil {
		cblogger.Error(getNewNodeGroupErr)
		LoggingError(hiscallInfo, getNewNodeGroupErr)
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("Failed to Add Node Group. err = %s", getNewNodeGroupErr))
	}

	// Apply Node Group autosacler options
	irsCluster.NodeGroupList = append(irsCluster.NodeGroupList, nodeGroupReqInfo)
	applyAutoScalerOptionErr := ic.applyAutoScalerOptions(irsCluster, irsCluster.IId.SystemId, resourceGroupId)
	if applyAutoScalerOptionErr != nil {
		cblogger.Error(applyAutoScalerOptionErr)
		LoggingError(hiscallInfo, applyAutoScalerOptionErr)
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("Failed to Add Node Group. err = %s", applyAutoScalerOptionErr))
	}

	// Get Workers in pool
	getWorkersResult, _, getWorkersErr := ic.ClusterService.VpcGetWorkersWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcGetWorkersOptions{
		Cluster:            core.StringPtr(irsCluster.IId.SystemId),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
		ShowDeleted:        core.StringPtr("false"),
		Pool:               newNodeGroup.ID,
	})
	if getWorkersErr != nil {
		cblogger.Error(getWorkersErr)
		LoggingError(hiscallInfo, getWorkersErr)
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("Failed to Add Node Group. err = %s", getWorkersErr))
	}

	var nodesIID []irs.IID
	for _, worker := range getWorkersResult {
		nodesIID = append(nodesIID, irs.IID{
			NameId:   RetrieveUnableErr,
			SystemId: *worker.ID,
		})
	}

	LoggingInfo(hiscallInfo, start)

	return irs.NodeGroupInfo{
		IId: irs.IID{
			NameId:   *newNodeGroup.PoolName,
			SystemId: *newNodeGroup.ID,
		},
		ImageIID: irs.IID{
			NameId:   RetrieveUnableErr,
			SystemId: RetrieveUnableErr,
		},
		VMSpecName:   *newNodeGroup.Flavor,
		RootDiskType: RetrieveUnableErr,
		RootDiskSize: RetrieveUnableErr,
		KeyPairIID: irs.IID{
			NameId:   RetrieveUnableErr,
			SystemId: RetrieveUnableErr,
		},
		OnAutoScaling:   false,
		DesiredNodeSize: -1,
		MinNodeSize:     -1,
		MaxNodeSize:     -1,
		Status:          ic.getNodeGroupStatusFromString(*newNodeGroup.Lifecycle.DesiredState),
		Nodes:           nodesIID,
		KeyValueList:    nil,
	}, nil
}

func (ic *IbmClusterHandler) SetNodeGroupAutoScaling(clusterIID irs.IID, nodeGroupIID irs.IID, on bool) (bool, error) {
	hiscallInfo := GetCallLogScheme(ic.Region, call.CLUSTER, clusterIID.NameId, "SetNodeGroupAutoScaling()")
	start := call.Start()

	// validation
	if clusterIID.SystemId == "" && clusterIID.NameId == "" {
		return false, errors.New("Failed to Set Node Group Auto Scaling. err = Invalid Cluster IID")
	}
	if nodeGroupIID.SystemId == "" && nodeGroupIID.NameId == "" {
		return false, errors.New("Failed to Set Node Group Auto Scaling. err = Invalid Node Group IID")
	}

	// get resource group id
	resourceGroupId, getResourceGroupErr := ic.getDefaultResourceGroupId()
	if getResourceGroupErr != nil {
		cblogger.Error(getResourceGroupErr)
		LoggingError(hiscallInfo, getResourceGroupErr)
		return false, errors.New(fmt.Sprintf("Failed to Set Node Group Auto Scaling. err = %s", getResourceGroupErr))
	}

	irsCluster, getIrsClusterErr := ic.GetCluster(clusterIID)
	if getIrsClusterErr != nil {
		cblogger.Error(getIrsClusterErr)
		LoggingError(hiscallInfo, getIrsClusterErr)
		return false, errors.New(fmt.Sprintf("Failed to Set Node Group Auto Scaling. err = %s", getIrsClusterErr))
	}
	if irsCluster.Status == irs.ClusterCreating || irsCluster.Status == irs.ClusterDeleting || irsCluster.Status == irs.ClusterUpdating {
		clusterStatusErr := errors.New(fmt.Sprintf("Cannot Set Node Group AutoScaling at %s status", irsCluster.Status))
		cblogger.Error(clusterStatusErr)
		LoggingError(hiscallInfo, clusterStatusErr)
		return false, errors.New(fmt.Sprintf("Failed to Set Node Group AutoScaling. err = %s", clusterStatusErr))
	}

	nodeGroups, _, getNodeGroupsErr := ic.ClusterService.VpcGetWorkerPoolsWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcGetWorkerPoolsOptions{
		Cluster:            core.StringPtr(irsCluster.IId.SystemId),
		XRegion:            core.StringPtr(ic.Region.Region),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
	})
	if getNodeGroupsErr != nil {
		cblogger.Error(getNodeGroupsErr)
		LoggingError(hiscallInfo, getNodeGroupsErr)
		return false, errors.New(fmt.Sprintf("Failed to Set Node Group Auto Scaling. err = %s", getNodeGroupsErr))
	}

	var targetNodeGroup *kubernetesserviceapiv1.GetWorkerPoolsDetailResponse
	for _, nodeGroup := range *nodeGroups {
		if nodeGroup.Id == nodeGroupIID.SystemId || nodeGroup.PoolName == nodeGroupIID.NameId {
			targetNodeGroup = &nodeGroup
			break
		}
	}
	if targetNodeGroup == nil {
		nodeGroupNotExistErr := errors.New(fmt.Sprintf("Failed to Set Node Group Auto Scaling. err = cannot find node group: %s", nodeGroupIID))
		cblogger.Error(nodeGroupNotExistErr)
		LoggingError(hiscallInfo, nodeGroupNotExistErr)
		return false, nodeGroupNotExistErr
	}

	nodeGroupName := targetNodeGroup.PoolName

	kubeConfigStr, getKubeConfigErr := ic.getKubeConfig(irsCluster.IId.SystemId, resourceGroupId)
	if getKubeConfigErr != nil {
		cblogger.Error(getKubeConfigErr)
		LoggingError(hiscallInfo, getKubeConfigErr)
		return false, errors.New(fmt.Sprintf("Failed to Set Node Group Auto Scaling. err = %s", getKubeConfigErr))
	}

	var newNodeGroupInfo []irs.NodeGroupInfo
	configMap, getConfigMapErr := ic.getAutoScalerConfigMap(kubeConfigStr)
	if getConfigMapErr != nil {
		cblogger.Error(getConfigMapErr)
		LoggingError(hiscallInfo, getConfigMapErr)
		return false, errors.New(fmt.Sprintf("Failed to Set Node Group Auto Scaling. err = %s", getConfigMapErr))
	}
	if configMap == nil {
		configMapNotExistErr := errors.New("Failed to Set Node Group Auto Scaling. err = Cannot find Auto Scaler Config Map, Please try after autoscaler addon is deployed")
		cblogger.Error(configMapNotExistErr)
		LoggingError(hiscallInfo, configMapNotExistErr)
		return false, configMapNotExistErr
	} else {
		jsonProperty, exists := configMap.Data[AutoscalerConfigMapOptionProperty]
		if !exists {
			propertyNotExistErr := errors.New("Failed to Set Node Group Auto Scaling. err = Cannot find Auto Scaler Config Map, Please try after autoscaler addon is deployed")
			cblogger.Error(propertyNotExistErr)
			LoggingError(hiscallInfo, propertyNotExistErr)
			return false, propertyNotExistErr
		}

		var workerPoolAutoscalerConfigs []kubernetesserviceapiv1.WorkerPoolAutoscalerConfig
		unmarshalErr := json.Unmarshal([]byte(jsonProperty), &workerPoolAutoscalerConfigs)
		if unmarshalErr != nil {
			cblogger.Error(unmarshalErr)
			LoggingError(hiscallInfo, unmarshalErr)
			return false, errors.New(fmt.Sprintf("Failed to Set Node Group Auto Scaling. err = %s", unmarshalErr))
		}

		isIncluded := false
		for i, config := range workerPoolAutoscalerConfigs {
			if config.Name == nodeGroupName {
				isIncluded = true
				workerPoolAutoscalerConfigs[i].Enabled = on
			}
			newNodeGroupInfo = append(newNodeGroupInfo, irs.NodeGroupInfo{
				IId:           irs.IID{NameId: workerPoolAutoscalerConfigs[i].Name},
				OnAutoScaling: workerPoolAutoscalerConfigs[i].Enabled,
				MinNodeSize:   workerPoolAutoscalerConfigs[i].MinSize,
				MaxNodeSize:   workerPoolAutoscalerConfigs[i].MaxSize,
			})
		}

		if !isIncluded {
			autoScalingSettingNotExistsErr := errors.New("Failed to Set Node Group Auto Scaling. err = Cannot find Node Group Auto Scaling Setting in Auto Scaler Config Map, Please try change Node Group scaling")
			cblogger.Error(autoScalingSettingNotExistsErr)
			LoggingError(hiscallInfo, autoScalingSettingNotExistsErr)
			return false, autoScalingSettingNotExistsErr
		}
	}

	updateConfigMapErr := ic.updateAutoScalerConfigMap(kubeConfigStr, newNodeGroupInfo)
	if updateConfigMapErr != nil {
		cblogger.Error(updateConfigMapErr)
		LoggingError(hiscallInfo, updateConfigMapErr)
		return false, errors.New(fmt.Sprintf("Failed to Set Node Group Auto Scaling. err = %s", updateConfigMapErr))
	}

	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func (ic *IbmClusterHandler) ChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID, DesiredNodeSize int, MinNodeSize int, MaxNodeSize int) (irs.NodeGroupInfo, error) {
	hiscallInfo := GetCallLogScheme(ic.Region, call.CLUSTER, clusterIID.NameId, "ChangeNodeGroupScaling()")
	start := call.Start()

	// validation
	validateErr := ic.validateAtChangeNodeGroupScaling(clusterIID, nodeGroupIID, MinNodeSize, MaxNodeSize)
	if validateErr != nil {
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("Failed to Change Node Group Scaling. err = %s", validateErr))
	}

	// get resource group id
	resourceGroupId, getResourceGroupErr := ic.getDefaultResourceGroupId()
	if getResourceGroupErr != nil {
		cblogger.Error(getResourceGroupErr)
		LoggingError(hiscallInfo, getResourceGroupErr)
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("Failed to Change Node Group Scaling. err = %s", getResourceGroupErr))
	}

	irsCluster, getIrsClusterErr := ic.GetCluster(clusterIID)
	if getIrsClusterErr != nil {
		cblogger.Error(getIrsClusterErr)
		LoggingError(hiscallInfo, getIrsClusterErr)
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("Failed to Change Node Group Scaling. err = %s", getIrsClusterErr))
	}
	if irsCluster.Status == irs.ClusterCreating || irsCluster.Status == irs.ClusterDeleting || irsCluster.Status == irs.ClusterUpdating {
		clusterStatusErr := errors.New(fmt.Sprintf("Cannot Change Node Group Scaling at %s status", irsCluster.Status))
		cblogger.Error(clusterStatusErr)
		LoggingError(hiscallInfo, clusterStatusErr)
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("Failed to Change Node Group Scaling. err = %s", clusterStatusErr))
	}

	nodeGroups, _, getNodeGroupsErr := ic.ClusterService.VpcGetWorkerPoolsWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcGetWorkerPoolsOptions{
		Cluster:            core.StringPtr(irsCluster.IId.SystemId),
		XRegion:            core.StringPtr(ic.Region.Region),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
	})
	if getNodeGroupsErr != nil {
		cblogger.Error(getNodeGroupsErr)
		LoggingError(hiscallInfo, getNodeGroupsErr)
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("Failed to Change Node Group Scaling. err = %s", getNodeGroupsErr))
	}

	var targetNodeGroup *kubernetesserviceapiv1.GetWorkerPoolsDetailResponse
	for _, nodeGroup := range *nodeGroups {
		if nodeGroup.Id == nodeGroupIID.SystemId || nodeGroup.PoolName == nodeGroupIID.NameId {
			targetNodeGroup = &nodeGroup
			break
		}
	}
	if targetNodeGroup == nil {
		nodeGroupNotExistErr := errors.New(fmt.Sprintf("Failed to Change Node Group Scaling. err = cannot find node group: %s", nodeGroupIID))
		cblogger.Error(nodeGroupNotExistErr)
		LoggingError(hiscallInfo, nodeGroupNotExistErr)
		return irs.NodeGroupInfo{}, nodeGroupNotExistErr
	}

	nodeGroupName := targetNodeGroup.PoolName

	kubeConfigStr, getKubeConfigErr := ic.getKubeConfig(irsCluster.IId.SystemId, resourceGroupId)
	if getKubeConfigErr != nil {
		cblogger.Error(getKubeConfigErr)
		LoggingError(hiscallInfo, getKubeConfigErr)
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("Failed to Change Node Group Scaling. err = %s", getKubeConfigErr))
	}

	var newNodeGroupInfo []irs.NodeGroupInfo
	var changedNodeGroupIndex int
	configMap, getConfigMapErr := ic.getAutoScalerConfigMap(kubeConfigStr)
	if getConfigMapErr != nil {
		cblogger.Error(getConfigMapErr)
		LoggingError(hiscallInfo, getConfigMapErr)
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("Failed to Change Node Group Scaling. err = %s", getConfigMapErr))
	}
	if configMap == nil {
		configMapNotExistErr := errors.New("Failed to Change Node Group Scaling. err = Cannot find Auto Scaler Config Map, Please try after autoscaler addon is deployed")
		cblogger.Error(configMapNotExistErr)
		LoggingError(hiscallInfo, configMapNotExistErr)
		return irs.NodeGroupInfo{}, configMapNotExistErr
	} else {
		jsonProperty, exists := configMap.Data[AutoscalerConfigMapOptionProperty]
		if !exists {
			propertyNotExistErr := errors.New("Failed to Change Node Group Scaling. err = Cannot find Auto Scaler Config Map, Please try after autoscaler addon is deployed")
			cblogger.Error(propertyNotExistErr)
			LoggingError(hiscallInfo, propertyNotExistErr)
			return irs.NodeGroupInfo{}, propertyNotExistErr
		}

		var workerPoolAutoscalerConfigs []kubernetesserviceapiv1.WorkerPoolAutoscalerConfig
		unmarshalErr := json.Unmarshal([]byte(jsonProperty), &workerPoolAutoscalerConfigs)
		if unmarshalErr != nil {
			cblogger.Error(unmarshalErr)
			LoggingError(hiscallInfo, unmarshalErr)
			return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("Failed to Change Node Group Scaling. err = %s", unmarshalErr))
		}

		isIncluded := false
		for i, config := range workerPoolAutoscalerConfigs {
			if config.Name == nodeGroupName {
				isIncluded = true
				changedNodeGroupIndex = i
				workerPoolAutoscalerConfigs[i].MinSize = MinNodeSize
				workerPoolAutoscalerConfigs[i].MaxSize = MaxNodeSize
			}
			newNodeGroupInfo = append(newNodeGroupInfo, irs.NodeGroupInfo{
				IId:           irs.IID{NameId: workerPoolAutoscalerConfigs[i].Name},
				OnAutoScaling: workerPoolAutoscalerConfigs[i].Enabled,
				MinNodeSize:   workerPoolAutoscalerConfigs[i].MinSize,
				MaxNodeSize:   workerPoolAutoscalerConfigs[i].MaxSize,
			})
		}

		if !isIncluded {
			newNodeGroupInfo = append(newNodeGroupInfo, irs.NodeGroupInfo{
				IId:           irs.IID{NameId: nodeGroupName},
				OnAutoScaling: false,
				MinNodeSize:   MinNodeSize,
				MaxNodeSize:   MaxNodeSize,
			})
			changedNodeGroupIndex = len(newNodeGroupInfo) - 1
		}
	}

	updateConfigMapErr := ic.updateAutoScalerConfigMap(kubeConfigStr, newNodeGroupInfo)
	if updateConfigMapErr != nil {
		cblogger.Error(updateConfigMapErr)
		LoggingError(hiscallInfo, updateConfigMapErr)
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("Failed to Change Node Group Scaling. err = %s", updateConfigMapErr))
	}

	// Get Workers in pool
	getWorkersResult, _, getWorkersErr := ic.ClusterService.VpcGetWorkersWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcGetWorkersOptions{
		Cluster:            core.StringPtr(irsCluster.IId.SystemId),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
		ShowDeleted:        core.StringPtr("false"),
		Pool:               core.StringPtr(targetNodeGroup.Id),
	})
	if getWorkersErr != nil {
		cblogger.Error(getWorkersErr)
		LoggingError(hiscallInfo, getWorkersErr)
		return irs.NodeGroupInfo{}, errors.New(fmt.Sprintf("Failed to Add Node Group. err = %s", getWorkersErr))
	}

	var nodesIID []irs.IID
	for _, worker := range getWorkersResult {
		nodesIID = append(nodesIID, irs.IID{
			NameId:   RetrieveUnableErr,
			SystemId: *worker.ID,
		})
	}

	LoggingInfo(hiscallInfo, start)

	return irs.NodeGroupInfo{
		IId: irs.IID{
			NameId:   targetNodeGroup.PoolName,
			SystemId: targetNodeGroup.Id,
		},
		ImageIID: irs.IID{
			NameId:   RetrieveUnableErr,
			SystemId: RetrieveUnableErr,
		},
		VMSpecName:   targetNodeGroup.Flavor,
		RootDiskType: RetrieveUnableErr,
		RootDiskSize: RetrieveUnableErr,
		KeyPairIID: irs.IID{
			NameId:   RetrieveUnableErr,
			SystemId: RetrieveUnableErr,
		},
		OnAutoScaling:   newNodeGroupInfo[changedNodeGroupIndex].OnAutoScaling,
		DesiredNodeSize: -1,
		MinNodeSize:     newNodeGroupInfo[changedNodeGroupIndex].MinNodeSize,
		MaxNodeSize:     newNodeGroupInfo[changedNodeGroupIndex].MaxNodeSize,
		Status:          ic.getNodeGroupStatusFromString(targetNodeGroup.Lifecycle.DesiredState),
		Nodes:           nodesIID,
		KeyValueList:    nil,
	}, nil
}

func (ic *IbmClusterHandler) RemoveNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(ic.Region, call.CLUSTER, clusterIID.NameId, "RemoveNodeGroup()")
	start := call.Start()

	// validation
	if clusterIID.SystemId == "" && clusterIID.NameId == "" {
		return false, errors.New("Failed to Set Node Group Auto Scaling. err = Invalid Cluster IID")
	}
	if nodeGroupIID.SystemId == "" && nodeGroupIID.NameId == "" {
		return false, errors.New("Failed to Set Node Group Auto Scaling. err = Invalid Node Group IID")
	}

	// get resource group id
	resourceGroupId, getResourceGroupErr := ic.getDefaultResourceGroupId()
	if getResourceGroupErr != nil {
		cblogger.Error(getResourceGroupErr)
		LoggingError(hiscallInfo, getResourceGroupErr)
		return false, errors.New(fmt.Sprintf("Failed to Remove Node Group. err = %s", getResourceGroupErr))
	}

	irsCluster, getIrsClusterErr := ic.GetCluster(clusterIID)
	if getIrsClusterErr != nil {
		cblogger.Error(getIrsClusterErr)
		LoggingError(hiscallInfo, getIrsClusterErr)
		return false, errors.New(fmt.Sprintf("Failed to Remove Node Group. err = %s", getIrsClusterErr))
	}
	if irsCluster.Status == irs.ClusterCreating || irsCluster.Status == irs.ClusterDeleting {
		clusterStatusErr := errors.New(fmt.Sprintf("Cannot Remove Node Group at %s status", irsCluster.Status))
		cblogger.Error(clusterStatusErr)
		LoggingError(hiscallInfo, clusterStatusErr)
		return false, errors.New(fmt.Sprintf("Failed to Remove Node Group. err = %s", clusterStatusErr))
	}

	nodeGroups, _, getNodeGroupsErr := ic.ClusterService.VpcGetWorkerPoolsWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcGetWorkerPoolsOptions{
		Cluster:            core.StringPtr(irsCluster.IId.SystemId),
		XRegion:            core.StringPtr(ic.Region.Region),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
	})
	if getNodeGroupsErr != nil {
		cblogger.Error(getNodeGroupsErr)
		LoggingError(hiscallInfo, getNodeGroupsErr)
		return false, errors.New(fmt.Sprintf("Failed to Remove Node Group. err = %s", getNodeGroupsErr))
	}

	var targetNodeGroup *kubernetesserviceapiv1.GetWorkerPoolsDetailResponse
	for _, nodeGroup := range *nodeGroups {
		if nodeGroup.Id == nodeGroupIID.SystemId || nodeGroup.PoolName == nodeGroupIID.NameId {
			targetNodeGroup = &nodeGroup
			break
		}
	}

	_, removeErr := ic.ClusterService.RemoveWorkerPoolWithContext(ic.Ctx, &kubernetesserviceapiv1.RemoveWorkerPoolOptions{
		IdOrName:           core.StringPtr(irsCluster.IId.SystemId),
		PoolidOrName:       core.StringPtr(targetNodeGroup.Id),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
	})
	if removeErr != nil {
		cblogger.Error(removeErr)
		LoggingError(hiscallInfo, removeErr)
		return false, errors.New(fmt.Sprintf("Failed to Remove Node Group. err = %s", removeErr))
	}

	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func (ic *IbmClusterHandler) UpgradeCluster(clusterIID irs.IID, newVersion string) (irs.ClusterInfo, error) {
	hiscallInfo := GetCallLogScheme(ic.Region, call.CLUSTER, clusterIID.NameId, "UpgradeCluster()")
	start := call.Start()

	// validation
	if clusterIID.SystemId == "" && clusterIID.NameId == "" {
		return irs.ClusterInfo{}, errors.New("Failed to Set Node Group Auto Scaling. err = Invalid Cluster IID")
	}
	if newVersion == "" {
		return irs.ClusterInfo{}, errors.New("Failed to Set Node Group Auto Scaling. err = New Version is required")
	}

	// get resource group id
	resourceGroupId, getResourceGroupErr := ic.getDefaultResourceGroupId()
	if getResourceGroupErr != nil {
		cblogger.Error(getResourceGroupErr)
		LoggingError(hiscallInfo, getResourceGroupErr)
		return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Upgrade Cluster. err = %s", getResourceGroupErr))
	}

	fullClusterIID, getClusterIIDErr := ic.getClusterIID(clusterIID)
	if getClusterIIDErr != nil {
		cblogger.Error(getClusterIIDErr)
		LoggingError(hiscallInfo, getClusterIIDErr)
		return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Upgrade Cluster. err = %s", getClusterIIDErr))
	}

	prevIrsClsuter, getIrsClusterErr := ic.GetCluster(fullClusterIID)
	if getIrsClusterErr != nil {
		cblogger.Error(getIrsClusterErr)
		LoggingError(hiscallInfo, getIrsClusterErr)
		return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Upgrade Cluster. err = %s", getIrsClusterErr))
	}
	if prevIrsClsuter.Status != irs.ClusterActive {
		clusterStatusErr := errors.New(fmt.Sprintf("Failed to Upgrade Cluster. err = Cannot upgrade cluster in %s status", prevIrsClsuter.Status))
		cblogger.Error(clusterStatusErr)
		LoggingError(hiscallInfo, clusterStatusErr)
		return irs.ClusterInfo{}, clusterStatusErr
	}

	rawCluster, _, getClusterErr := ic.ClusterService.VpcGetClusterWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcGetClusterOptions{
		Cluster:            core.StringPtr(fullClusterIID.SystemId),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
		ShowResources:      core.StringPtr("true"),
	})
	if getClusterErr != nil {
		cblogger.Error(getClusterErr)
		LoggingError(hiscallInfo, getClusterErr)
		return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Upgrade Cluster. err = %s", getClusterErr))
	}
	targetCluster := (*rawCluster)[0]

	// uninstall autoscaler addon
	uninstallErr := ic.uninstallAutoScalerAddon(fullClusterIID.SystemId, targetCluster.Crn, resourceGroupId)
	if uninstallErr != nil {
		cblogger.Error(uninstallErr)
		LoggingError(hiscallInfo, uninstallErr)
		return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Upgrade Cluster. err = %s", uninstallErr))
	}

	// upgrade master
	_, updateErr := ic.ClusterService.V2UpdateMasterWithContext(ic.Ctx, &kubernetesserviceapiv1.V2UpdateMasterOptions{
		Cluster:            core.StringPtr(fullClusterIID.SystemId),
		Force:              core.BoolPtr(false),
		Version:            core.StringPtr(newVersion),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
	})
	if updateErr != nil {
		cblogger.Error(updateErr)
		LoggingError(hiscallInfo, updateErr)
		return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Upgrade Cluster. err = %s", updateErr))
	}
	ic.manageStatusTag(targetCluster.Crn, MasterUpgradeStatus, WAITING)

	// upgrade workers
	go func() {
		// wait until update master and uninstall autoscaler addon is done
		cnt := 0
		for cnt < UpgradeMasterRetry {
			// get autoscaler tags
			rawCluster, _, getClusterErr := ic.ClusterService.VpcGetClusterWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcGetClusterOptions{
				Cluster:            core.StringPtr(fullClusterIID.SystemId),
				XAuthResourceGroup: core.StringPtr(resourceGroupId),
				ShowResources:      core.StringPtr("true"),
			})
			if getClusterErr != nil {
				cnt++
				time.Sleep(time.Minute)
				continue
			}
			targetCluster := (*rawCluster)[0]
			tags, _, getTagsErr := ic.TaggingService.ListTagsWithContext(ic.Ctx, &globaltaggingv1.ListTagsOptions{
				TagType:    core.StringPtr("user"),
				AttachedTo: core.StringPtr(targetCluster.Crn),
			})
			if getTagsErr != nil {
				cnt++
				time.Sleep(time.Minute)
				continue
			}

			// check if master upgrade is done
			isMasterUpgrading := true
			isMasterUpgradeIndicated := strings.Contains(targetCluster.MasterKubeVersion, "-->")
			for _, tag := range (*tags).Items {
				if isTagStatusOf(*tag.Name, MasterUpgradeStatus) {
					if compareTag(*tag.Name, MasterUpgradeStatus, WAITING) {
						if isMasterUpgradeIndicated {
							ic.manageStatusTag(targetCluster.Crn, MasterUpgradeStatus, UPGRADING)
							break
						}
					}
					if compareTag(*tag.Name, MasterUpgradeStatus, UPGRADING) {
						if !isMasterUpgradeIndicated {
							ic.manageStatusTag(targetCluster.Crn, MasterUpgradeStatus, "")
							isMasterUpgrading = false
						}
						break
					}
				}
			}
			if isMasterUpgrading {
				cnt++
				time.Sleep(time.Minute)
				continue
			}

			// upgrade workers
			getWorkersResult, _, _ := ic.ClusterService.VpcGetWorkersWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcGetWorkersOptions{
				Cluster:            core.StringPtr(fullClusterIID.SystemId),
				XAuthResourceGroup: core.StringPtr(resourceGroupId),
				ShowDeleted:        core.StringPtr("false"),
			})
			for _, worker := range getWorkersResult {
				ic.ClusterService.VpcReplaceWorker(&kubernetesserviceapiv1.VpcReplaceWorkerOptions{
					Cluster:            core.StringPtr(fullClusterIID.SystemId),
					Update:             core.BoolPtr(true),
					WorkerID:           worker.ID,
					XAuthResourceGroup: core.StringPtr(resourceGroupId),
				})
			}
			break
		}
	}()

	// reinstall new autoscaler
	prevIrsClsuter.Version = newVersion
	autoScalerErr := ic.installAutoScalerAddon(prevIrsClsuter, fullClusterIID.SystemId, targetCluster.Crn, resourceGroupId, true)
	if autoScalerErr != nil {
		cblogger.Error(autoScalerErr)
		LoggingError(hiscallInfo, autoScalerErr)
		return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Upgrade Cluster. err = %s", autoScalerErr))
	}

	// get cluster info
	irsCluster, getIrsClusterErr := ic.GetCluster(clusterIID)
	if getIrsClusterErr != nil {
		cblogger.Error(getIrsClusterErr)
		LoggingError(hiscallInfo, getIrsClusterErr)
		return irs.ClusterInfo{}, errors.New(fmt.Sprintf("Failed to Upgrade Cluster. err = %s", getIrsClusterErr))
	}

	LoggingInfo(hiscallInfo, start)

	return irsCluster, nil
}

func (ic *IbmClusterHandler) applyAutoScalerOptions(clusterReqInfo irs.ClusterInfo, clusterId string, resourceGroupId string) error {
	kubeConfigStr, getKubeConfigErr := ic.getKubeConfig(clusterId, resourceGroupId)
	if getKubeConfigErr != nil {
		return getKubeConfigErr
	}
	patchConfigMapErr := ic.updateAutoScalerConfigMap(kubeConfigStr, clusterReqInfo.NodeGroupList)
	if patchConfigMapErr != nil {
		return patchConfigMapErr
	}

	return nil
}

func (ic *IbmClusterHandler) manageStatusTag(crn string, tag string, status string) {
	var deleteTargets *[]string
	switch tag {
	case AutoScalerStatus:
		deleteTargets = &autoSaclerStates
	case SecurityGroupStatus:
		deleteTargets = &securityGroupStates
	case MasterUpgradeStatus:
		deleteTargets = &masterUpgradeStates
	default:
	}

	if deleteTargets != nil && len(*deleteTargets) != 0 {
		ic.TaggingService.DetachTag(&globaltaggingv1.DetachTagOptions{
			Resources: []globaltaggingv1.Resource{{
				ResourceID: core.StringPtr(crn),
			}},
			TagNames: *deleteTargets,
			TagType:  core.StringPtr("user"),
		})
	}

	if status != "" {
		ic.TaggingService.AttachTag(&globaltaggingv1.AttachTagOptions{
			Resources: []globaltaggingv1.Resource{{
				ResourceID: core.StringPtr(crn),
			}},
			TagName: core.StringPtr(fmt.Sprintf("%s%s", tag, status)),
			TagType: core.StringPtr("user"),
		})
	}
}

func (ic *IbmClusterHandler) checkIfClusterIsSupported(versionRange string, clusterK8sVersion string) bool {
	clusterVersion, err := version.NewVersion(clusterK8sVersion)
	if err != nil {
		return false
	}

	minVersion, err := version.NewVersion(strings.Split(strings.Split(versionRange, " ")[0], ">=")[1])
	if err != nil {
		return false
	}

	maxVersion, err := version.NewVersion(strings.Split(strings.Split(versionRange, " ")[1], "<")[1])
	if err != nil {
		return false
	}

	return clusterVersion.GreaterThanOrEqual(minVersion) && clusterVersion.LessThan(maxVersion)
}

func (ic *IbmClusterHandler) createRemainingWorkerPools(clusterReqInfo irs.ClusterInfo, vpcInfo irs.VPCInfo, subnetInfo irs.SubnetInfo, rawCluster kubernetesserviceapiv1.GetClusterDetailResponse, resourceGroupId string) {
	for i := 1; i < len(clusterReqInfo.NodeGroupList); i++ {
		workerPool := ic.getWorkerPoolFromNodeGroupInfo(clusterReqInfo.NodeGroupList[i], vpcInfo.IId.SystemId, subnetInfo.IId.SystemId)
		_, _, createWorkerPoolErr := ic.ClusterService.VpcCreateWorkerPoolWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcCreateWorkerPoolOptions{
			Cluster:     core.StringPtr(rawCluster.Id),
			Flavor:      workerPool.Flavor,
			Isolation:   workerPool.Isolation,
			Name:        workerPool.Name,
			VpcID:       workerPool.VpcID,
			WorkerCount: workerPool.WorkerCount,
			Zones: []kubernetesserviceapiv1.Zone{{
				ID:       core.StringPtr(ic.Region.Zone),
				SubnetID: core.StringPtr(subnetInfo.IId.SystemId),
			}},
			Authorization:      core.StringPtr(ic.CredentialInfo.AuthToken),
			XAuthResourceGroup: core.StringPtr(resourceGroupId),
			Headers:            nil,
		})
		if createWorkerPoolErr != nil {
			cblogger.Error(createWorkerPoolErr)
			ic.DeleteCluster(clusterReqInfo.IId)
		}
	}
}

func (ic *IbmClusterHandler) getAddonInfo(clusterId string, resourceGroupId string) (irs.AddonsInfo, error) {
	rawClusterAddons, _, getClusterAddonsErr := ic.ClusterService.GetClusterAddonsWithContext(ic.Ctx, &kubernetesserviceapiv1.GetClusterAddonsOptions{
		IdOrName:           core.StringPtr(clusterId),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
	})
	if getClusterAddonsErr != nil {
		return irs.AddonsInfo{}, getClusterAddonsErr
	}
	var keyValues []irs.KeyValue
	for _, clusterAddon := range rawClusterAddons {
		addonJsonValue, marshalErr := json.Marshal(clusterAddon)
		if marshalErr != nil {
			cblogger.Error(marshalErr)
		}
		keyValues = append(keyValues, irs.KeyValue{
			Key:   *clusterAddon.Name,
			Value: string(addonJsonValue),
		})
	}
	clusterAddons := irs.AddonsInfo{KeyValueList: keyValues}
	return clusterAddons, nil
}

func (ic *IbmClusterHandler) getAutoScalerConfigMap(kubeConfigStr string) (*v1.ConfigMap, error) {
	k8sClient, getK8sClientErr := ic.ClusterService.GetKubernetesClient(kubeConfigStr)
	if getK8sClientErr != nil {
		return nil, getK8sClientErr
	}

	configMap, getConfigMapErr := k8sClient.CoreV1().ConfigMaps(ConfigMapNamespace).Get(context.Background(), AutoscalerConfigMap, metav1.GetOptions{})
	if getConfigMapErr != nil {
		return nil, getConfigMapErr
	}

	return configMap, nil
}

func (ic *IbmClusterHandler) getClusterFinalStatus(clusterStatus irs.ClusterStatus, autoSaclerStatus string, securityGroupStatus string) irs.ClusterStatus {
	if compareTag(autoSaclerStatus, AutoScalerStatus, FAILED) || compareTag(securityGroupStatus, SecurityGroupStatus, FAILED) {
		return irs.ClusterInactive
	}
	if compareTag(autoSaclerStatus, AutoScalerStatus, WAITING) ||
		compareTag(autoSaclerStatus, AutoScalerStatus, DEPLOYING) ||
		compareTag(securityGroupStatus, SecurityGroupStatus, INITIALIZING) {
		return irs.ClusterCreating
	}
	if compareTag(autoSaclerStatus, AutoScalerStatus, UPGRADE_DEPLOYING) {
		return irs.ClusterUpdating
	}
	return clusterStatus
}

func (ic *IbmClusterHandler) getClusterIID(clusterIID irs.IID) (irs.IID, error) {
	if clusterIID.NameId == "" && clusterIID.SystemId == "" {
		return irs.IID{}, errors.New("Failed to Get Cluster IID")
	}

	resourceGroupId, getResourceGroupIdErr := ic.getDefaultResourceGroupId()
	if getResourceGroupIdErr != nil {
		return irs.IID{}, errors.New(fmt.Sprintf("Failed to Get Cluster IID. err = %s", getResourceGroupIdErr))
	}

	var cluster string
	if clusterIID.SystemId != "" {
		cluster = clusterIID.SystemId
	} else {
		cluster = clusterIID.NameId
	}
	rawClusters, _, getClustersErr := ic.ClusterService.VpcGetClusterWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcGetClusterOptions{
		Cluster:            core.StringPtr(cluster),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
		ShowResources:      core.StringPtr("false"),
	})
	if getClustersErr != nil {
		return irs.IID{}, errors.New(fmt.Sprintf("Failed to Get Cluster IID. err = %s", getClustersErr))
	}

	return irs.IID{
		NameId:   (*rawClusters)[0].Name,
		SystemId: (*rawClusters)[0].Id,
	}, nil
}

func (ic *IbmClusterHandler) getClusterStatusFromString(clusterStatus string) irs.ClusterStatus {
	switch strings.ToLower(clusterStatus) {
	case "requested", "deploying", "pending":
		return irs.ClusterCreating
	case "normal":
		return irs.ClusterActive
	case "updating":
		return irs.ClusterUpdating
	case "aborted", "deleting":
		return irs.ClusterDeleting
	default:
		return irs.ClusterInactive
	}
}

func (ic *IbmClusterHandler) getDefaultResourceGroupId() (string, error) {
	if defaultResourceGroupId == "" {
		var err error

		// create resource controller
		resourceManagerManagerOptions := &resourcemanagerv2.ResourceManagerV2Options{
			Authenticator: &core.IamAuthenticator{
				ApiKey: ic.CredentialInfo.ApiKey,
			},
		}
		resourceManagerService, err := resourcemanagerv2.NewResourceManagerV2UsingExternalConfig(resourceManagerManagerOptions)
		if err != nil {
			return "", err
		}

		resourceGroups, _, listResourceGroupsErr := resourceManagerService.ListResourceGroups(&resourcemanagerv2.ListResourceGroupsOptions{
			Default: core.BoolPtr(true),
		})
		if listResourceGroupsErr != nil {
			return "", listResourceGroupsErr
		}

		for _, resourceGroup := range resourceGroups.Resources {
			if strings.EqualFold(*resourceGroup.Name, DefaultResourceGroup) {
				defaultResourceGroupId = *resourceGroup.ID
				return defaultResourceGroupId, nil
			}
		}

		return "", errors.New("failed to get default resource group")
	}

	return defaultResourceGroupId, nil
}

func (ic *IbmClusterHandler) getKubeConfig(clusterId string, resourceGroupId string) (string, error) {
	kubeConfig, getKubeConfigErr := ic.ClusterService.GetKubeconfigWithContext(ic.Ctx, &kubernetesserviceapiv1.GetKubeconfigOptions{
		Authorization:      core.StringPtr(ic.CredentialInfo.ApiKey),
		Cluster:            core.StringPtr(clusterId),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
		Format:             core.StringPtr("yaml"),
		Admin:              core.BoolPtr(true),
		Network:            core.BoolPtr(false),
	})
	if getKubeConfigErr != nil {
		return "", getKubeConfigErr
	} else {
		resultStr, ok := kubeConfig.Result.([]byte)
		if !ok {
			return "", errors.New("Failed to Get Kube Config String")
		} else {
			return string(resultStr), nil
		}
	}
}

func (ic *IbmClusterHandler) getNodeGroupStatusFromString(nodeGroupStatus string) irs.NodeGroupStatus {
	// IBM Node Group does not have Creating or Updating status
	// While creating a Node Group, its status is indicated as Active
	// While updating Node Group such as autoscale configuration, its status is indicated as Active
	switch strings.ToLower(nodeGroupStatus) {
	case "active":
		return irs.NodeGroupActive
	case "deleting":
		return irs.NodeGroupDeleting
	default:
		return irs.NodeGroupInactive
	}
}

func (ic *IbmClusterHandler) getWorkerPoolFromNodeGroupInfo(nodeGroupInfo irs.NodeGroupInfo, vpcId string, subnetId string) kubernetesserviceapiv1.VPCCreateClusterWorkerPool {
	return kubernetesserviceapiv1.VPCCreateClusterWorkerPool{
		Name:        core.StringPtr(nodeGroupInfo.IId.NameId),
		Flavor:      core.StringPtr(nodeGroupInfo.VMSpecName),
		Isolation:   core.StringPtr("public"),
		VpcID:       core.StringPtr(vpcId),
		WorkerCount: core.Int64Ptr(int64(nodeGroupInfo.DesiredNodeSize)),
		Zones: []kubernetesserviceapiv1.VPCCreateClusterWorkerPoolZone{{
			ID:       core.StringPtr(ic.Region.Zone),
			SubnetID: core.StringPtr(subnetId),
		}},
	}
}

func (ic *IbmClusterHandler) initSecurityGroup(clusterReqInfo irs.ClusterInfo, clusterId string, clusterCrn string) {
	sgHandler := IbmSecurityHandler{
		CredentialInfo: ic.CredentialInfo,
		Region:         ic.Region,
		VpcService:     ic.VpcService,
		Ctx:            ic.Ctx,
	}

	ic.manageStatusTag(clusterCrn, SecurityGroupStatus, INITIALIZING)
	go func() {
		cnt := 0
		for cnt < InitSecurityGroupRetry {
			defaultSgInfo, getDSIErr := sgHandler.GetSecurity(irs.IID{NameId: fmt.Sprintf("kube-%s", clusterId)})
			if getDSIErr != nil {
				time.Sleep(time.Minute)
				cnt++
				continue
			}

			initSuccess := true
			for _, sgIID := range clusterReqInfo.Network.SecurityGroupIIDs {
				sgInfo, getSgErr := sgHandler.GetSecurity(sgIID)
				if getSgErr != nil {
					initSuccess = false
					ic.manageStatusTag(clusterCrn, SecurityGroupStatus, FAILED)
					ic.DeleteCluster(clusterReqInfo.IId)
					break
				}

				_, sgUpdateErr := sgHandler.AddRules(defaultSgInfo.IId, sgInfo.SecurityRules)
				if sgUpdateErr != nil {
					initSuccess = false
					ic.manageStatusTag(clusterCrn, SecurityGroupStatus, FAILED)
					ic.DeleteCluster(clusterReqInfo.IId)
					break
				}
			}
			if initSuccess {
				ic.manageStatusTag(clusterCrn, SecurityGroupStatus, INITIALIZED)
			}
			break
		}
	}()

	return
}

func (ic *IbmClusterHandler) installAutoScalerAddon(clusterReqInfo irs.ClusterInfo, clusterId string, clusterCrn string, resourceGroupId string, isUpdating bool) error {
	// check exists
	addonsInfo, _ := ic.getAddonInfo(clusterId, resourceGroupId)
	for _, addon := range addonsInfo.KeyValueList {
		if addon.Key == AutoscalerAddon && strings.Contains(addon.Value, "\"healthState\":\"normal\"") {
			ic.manageStatusTag(clusterCrn, AutoScalerStatus, ACTIVE)
			return nil
		}
	}

	// start install addon
	availableAddons, _, getAvailAddonsErr := ic.ClusterService.GetAddonsWithContext(ic.Ctx, &kubernetesserviceapiv1.GetAddonsOptions{})
	if getAvailAddonsErr != nil {
		return errors.New(fmt.Sprintf("Failed to Create Cluster. err = %s", getAvailAddonsErr))
	}
	var availableAutoScalerAddons []kubernetesserviceapiv1.AddonCommon
	for _, availableAddon := range availableAddons {
		if *availableAddon.Name == AutoscalerAddon &&
			ic.checkIfClusterIsSupported(*availableAddon.SupportedKubeRange, clusterReqInfo.Version) {
			availableAutoScalerAddons = append(availableAutoScalerAddons, availableAddon)
		}
	}
	if len(availableAutoScalerAddons) < 1 {
		return errors.New(fmt.Sprintf("Failed to install autoscaler addon. err = No available autoscaler addon for this Kubernetes version: %s", clusterReqInfo.Version))
	}

	addons := []kubernetesserviceapiv1.ClusterAddon{{
		Name:    availableAutoScalerAddons[0].Name,
		Version: availableAutoScalerAddons[0].Version,
	}}

	go func() {
		cnt := 0
		enableSuccess := false
		ic.manageStatusTag(clusterCrn, AutoScalerStatus, WAITING)
		for cnt < EnableAutoScalerRetry {
			_, enableAddonDetail, enableAddonErr := ic.ClusterService.ManageClusterAddonsWithContext(ic.Ctx, &kubernetesserviceapiv1.ManageClusterAddonsOptions{
				IdOrName:           core.StringPtr(clusterId),
				Addons:             addons,
				Enable:             core.BoolPtr(true),
				XAuthResourceGroup: core.StringPtr(resourceGroupId),
			})
			if enableAddonErr != nil {
				resultMap, toMapOk := enableAddonDetail.GetResultAsMap()
				if toMapOk {
					descriptionStr, toStrOk := resultMap["description"].(string)
					if toStrOk {
						if !strings.Contains(descriptionStr, "The cluster is not fully deployed, please wait a couple minutes before enabling the add-on.") {
							break
						}
					}
				}
			} else {
				enableSuccess = true
				break
			}
			time.Sleep(time.Minute)
			cnt++
		}
		if !enableSuccess {
			ic.manageStatusTag(clusterCrn, AutoScalerStatus, FAILED)
		} else {
			if isUpdating {
				ic.manageStatusTag(clusterCrn, AutoScalerStatus, UPGRADE_DEPLOYING)
			} else {
				ic.manageStatusTag(clusterCrn, AutoScalerStatus, DEPLOYING)
			}

			go func() {
				cnt = 0
				for cnt < EnableAutoScalerRetry {
					addonsInfo, _ := ic.getAddonInfo(clusterId, resourceGroupId)
					for _, addon := range addonsInfo.KeyValueList {
						if addon.Key == AutoscalerAddon && strings.Contains(addon.Value, "\"healthState\":\"normal\"") {
							ic.manageStatusTag(clusterCrn, AutoScalerStatus, ACTIVE)
							ic.applyAutoScalerOptions(clusterReqInfo, clusterId, resourceGroupId)
							return
						}
					}

					time.Sleep(time.Minute)
					cnt++
				}
				ic.manageStatusTag(clusterCrn, AutoScalerStatus, FAILED)
				if !isUpdating {
					ic.DeleteCluster(clusterReqInfo.IId)
				}
			}()
		}
	}()

	return nil
}

func (ic *IbmClusterHandler) setClusterInfo(rawCluster kubernetesserviceapiv1.GetClusterDetailResponse) (irs.ClusterInfo, error) {
	resourceGroupId, getResourceGroupErr := ic.getDefaultResourceGroupId()
	if getResourceGroupErr != nil {
		return irs.ClusterInfo{}, getResourceGroupErr
	}

	// Get VPC Infos
	vpcIID := irs.IID{
		NameId:   "",
		SystemId: rawCluster.Vpcs[0],
	}
	vpcInfo, getVPCErr := GetRawVPC(vpcIID, ic.VpcService, ic.Ctx)
	if getVPCErr != nil {
		return irs.ClusterInfo{}, getVPCErr
	}
	vpcIID.NameId = *vpcInfo.Name

	// Get Addon Infos
	clusterAddons, getAddonErr := ic.getAddonInfo(rawCluster.Id, resourceGroupId)
	if getAddonErr != nil {
		return irs.ClusterInfo{}, getAddonErr
	}

	// Get Worker pool Infos
	getWorkerPoolsResult, _, getWorkerPoolErr := ic.ClusterService.VpcGetWorkerPoolsWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcGetWorkerPoolsOptions{
		Cluster:            core.StringPtr(rawCluster.Id),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
	})
	if getWorkerPoolErr != nil {
		return irs.ClusterInfo{}, getWorkerPoolErr
	}

	var nodeGroupList []irs.NodeGroupInfo
	for _, element := range *getWorkerPoolsResult {
		// Get Workers in pool
		getWorkersResult, _, getWorkersErr := ic.ClusterService.VpcGetWorkersWithContext(ic.Ctx, &kubernetesserviceapiv1.VpcGetWorkersOptions{
			Cluster:            core.StringPtr(rawCluster.Id),
			XAuthResourceGroup: core.StringPtr(resourceGroupId),
			ShowDeleted:        core.StringPtr("false"),
			Pool:               core.StringPtr(element.Id),
		})
		if getWorkersErr != nil {
			return irs.ClusterInfo{}, getWorkersErr
		}

		var nodesIID []irs.IID
		for _, worker := range getWorkersResult {
			nodesIID = append(nodesIID, irs.IID{
				NameId:   RetrieveUnableErr,
				SystemId: *worker.ID,
			})
		}

		nodeGroupList = append(nodeGroupList, irs.NodeGroupInfo{
			IId: irs.IID{
				NameId:   element.PoolName,
				SystemId: element.Id,
			},
			ImageIID: irs.IID{
				NameId:   RetrieveUnableErr,
				SystemId: RetrieveUnableErr,
			},
			VMSpecName:   element.Flavor,
			RootDiskType: RetrieveUnableErr,
			RootDiskSize: RetrieveUnableErr,
			KeyPairIID: irs.IID{
				NameId:   RetrieveUnableErr,
				SystemId: RetrieveUnableErr,
			},
			OnAutoScaling:   false,
			DesiredNodeSize: -1,
			MinNodeSize:     0,
			MaxNodeSize:     0,
			Status:          ic.getNodeGroupStatusFromString(element.Lifecycle.DesiredState),
			Nodes:           nodesIID,
			KeyValueList:    nil,
		})
	}

	// Get Network.Subnet Info
	rawVpeList, _, getRawVpeListErr := ic.VpcService.ListEndpointGateways(&vpcv1.ListEndpointGatewaysOptions{
		ResourceGroupID: core.StringPtr(resourceGroupId),
	})
	if getRawVpeListErr != nil {
		return irs.ClusterInfo{}, getRawVpeListErr
	}

	var target vpcv1.EndpointGateway
	for _, rawVpe := range rawVpeList.EndpointGateways {
		if *rawVpe.Name == fmt.Sprintf("iks-%s", rawCluster.Id) {
			target = rawVpe
		}
	}

	var subnetIIDs []irs.IID
	for _, ip := range target.Ips {
		if strings.Contains(*ip.Name, rawCluster.Id) {
			subnetId := strings.Split(*ip.Href, "/")[5]
			rawSubnet, _, getRawSubnetErr := ic.VpcService.GetSubnet(&vpcv1.GetSubnetOptions{
				ID: core.StringPtr(subnetId),
			})
			if getRawSubnetErr == nil {
				subnetIIDs = append(subnetIIDs, irs.IID{
					NameId:   *rawSubnet.Name,
					SystemId: *rawSubnet.ID,
				})
			}
		}
	}

	// Get Security Group
	sgHandler := IbmSecurityHandler{
		CredentialInfo: ic.CredentialInfo,
		Region:         ic.Region,
		VpcService:     ic.VpcService,
		Ctx:            ic.Ctx,
	}
	sgInfo, sgInfoErr := sgHandler.GetSecurity(irs.IID{NameId: fmt.Sprintf("kube-%s", rawCluster.Id)})
	if sgInfoErr != nil {
		return irs.ClusterInfo{}, sgInfoErr
	}

	securityGroups := []irs.IID{sgInfo.IId}

	// Convert Created Time
	createdTime, _ := strfmt.ParseDateTime(rawCluster.CreatedDate)

	// Determine Endpoint
	var serviceEndpoint string
	if rawCluster.ServiceEndpoints.PublicServiceEndpointEnabled {
		serviceEndpoint = rawCluster.ServiceEndpoints.PublicServiceEndpointURL
	} else {
		serviceEndpoint = rawCluster.ServiceEndpoints.PrivateServiceEndpointURL
	}

	// Get KubeConfig
	kubeConfigStr, getKubeConfigStrErr := ic.getKubeConfig(rawCluster.Id, resourceGroupId)
	if getKubeConfigStrErr == nil {
		// Get Autoscaling Info
		configMap, getAutoScalerConfigMapErr := ic.getAutoScalerConfigMap(kubeConfigStr)
		if getAutoScalerConfigMapErr != nil {
			cblogger.Error(getAutoScalerConfigMapErr)
		}

		if configMap == nil {
			for index, _ := range nodeGroupList {
				nodeGroupList[index].OnAutoScaling = false
				nodeGroupList[index].MinNodeSize = -1
				nodeGroupList[index].MaxNodeSize = -1
				nodeGroupList[index].DesiredNodeSize = -1
			}
		} else {
			jsonProperty, exists := configMap.Data[AutoscalerConfigMapOptionProperty]
			if !exists {
				cblogger.Error(errors.New("Failed to get Autoscaler Config from Config Map"))
			}

			var workerPoolAutoscalerConfigs []kubernetesserviceapiv1.WorkerPoolAutoscalerConfig
			unmarshalErr := json.Unmarshal([]byte(jsonProperty), &workerPoolAutoscalerConfigs)
			if unmarshalErr != nil {
				cblogger.Error(unmarshalErr)
			}

			for index, nodeGroup := range nodeGroupList {
				for _, config := range workerPoolAutoscalerConfigs {
					if config.Name == nodeGroup.IId.NameId {
						nodeGroupList[index].OnAutoScaling = config.Enabled
						nodeGroupList[index].MinNodeSize = config.MinSize
						nodeGroupList[index].MaxNodeSize = config.MaxSize
						nodeGroupList[index].DesiredNodeSize = -1
					}
				}
			}
		}
	} else {
		cblogger.Error(getKubeConfigStrErr)
	}

	// get cluster status
	clusterStatus := ic.getClusterStatusFromString(rawCluster.State)
	rawTags, _, getTagsErr := ic.TaggingService.ListTagsWithContext(ic.Ctx, &globaltaggingv1.ListTagsOptions{
		TagType:    core.StringPtr("user"),
		Providers:  []string{"ghost"},
		AttachedTo: core.StringPtr(rawCluster.Crn),
	})
	if getTagsErr != nil {
		clusterStatus = irs.ClusterInactive
	}

	autoScalerStatus := fmt.Sprintf("%s%s", AutoScalerStatus, FAILED)
	for _, tag := range rawTags.Items {
		if isTagStatusOf(*tag.Name, AutoScalerStatus) {
			autoScalerStatus = *tag.Name
			break
		}
	}
	securityGroupStatus := fmt.Sprintf("%s%s", SecurityGroupStatus, FAILED)
	for _, tag := range rawTags.Items {
		if isTagStatusOf(*tag.Name, SecurityGroupStatus) {
			securityGroupStatus = *tag.Name
			break
		}
	}
	clusterStatus = ic.getClusterFinalStatus(clusterStatus, autoScalerStatus, securityGroupStatus)

	// Build result
	return irs.ClusterInfo{
		IId: irs.IID{
			NameId:   rawCluster.Name,
			SystemId: rawCluster.Id,
		},
		Version: rawCluster.MasterKubeVersion,
		Network: irs.NetworkInfo{
			VpcIID:            vpcIID,
			SubnetIIDs:        subnetIIDs,
			SecurityGroupIIDs: securityGroups,
			KeyValueList:      nil,
		},
		NodeGroupList: nodeGroupList,
		AccessInfo: irs.AccessInfo{
			Endpoint:   serviceEndpoint,
			Kubeconfig: kubeConfigStr,
		},
		Addons:       clusterAddons,
		Status:       clusterStatus,
		CreatedTime:  time.Time(createdTime).Local(),
		KeyValueList: nil,
	}, nil
}

func (ic *IbmClusterHandler) uninstallAutoScalerAddon(clusterId string, clusterCrn string, resourceGroupId string) error {
	addons := []kubernetesserviceapiv1.ClusterAddon{{
		Name: core.StringPtr(AutoscalerAddon),
	}}
	_, _, disableAddonErr := ic.ClusterService.ManageClusterAddonsWithContext(ic.Ctx, &kubernetesserviceapiv1.ManageClusterAddonsOptions{
		IdOrName:           core.StringPtr(clusterId),
		Addons:             addons,
		Enable:             core.BoolPtr(false),
		XAuthResourceGroup: core.StringPtr(resourceGroupId),
	})
	if disableAddonErr != nil {
		return disableAddonErr
	}

	go func() {
		ic.manageStatusTag(clusterCrn, AutoScalerStatus, UNINSTALLING)

		cnt := 0
		for cnt < DisableAutoScalerRetry {
			autoScalerAddonExists := false
			addonsInfo, _ := ic.getAddonInfo(clusterId, resourceGroupId)
			for _, addon := range addonsInfo.KeyValueList {
				if addon.Key == AutoscalerAddon {
					cnt++
					autoScalerAddonExists = true
					time.Sleep(time.Minute)
					break
				}
			}
			if !autoScalerAddonExists {
				// remove autoscaler tag
				ic.manageStatusTag(clusterCrn, AutoScalerStatus, "")
				break
			}
		}
	}()

	return nil
}

func (ic *IbmClusterHandler) updateAutoScalerConfigMap(kubeConfigStr string, nodeGroupList []irs.NodeGroupInfo) error {
	k8sClient, getK8sClientErr := ic.ClusterService.GetKubernetesClient(kubeConfigStr)
	if getK8sClientErr != nil {
		return getK8sClientErr
	}

	configMap, getConfigMapErr := k8sClient.CoreV1().ConfigMaps(ConfigMapNamespace).Get(context.Background(), AutoscalerConfigMap, metav1.GetOptions{})
	if getConfigMapErr != nil {
		return getConfigMapErr
	}

	workerPoolAutoscalerConfigs := make([]kubernetesserviceapiv1.WorkerPoolAutoscalerConfig, len(nodeGroupList))
	for i, nodeGroup := range nodeGroupList {
		workerPoolAutoscalerConfigs[i].Name = nodeGroup.IId.NameId
		workerPoolAutoscalerConfigs[i].Enabled = nodeGroup.OnAutoScaling
		workerPoolAutoscalerConfigs[i].MaxSize = nodeGroup.MaxNodeSize
		workerPoolAutoscalerConfigs[i].MinSize = nodeGroup.MinNodeSize
	}

	newConfigJsonStr, marshalErr := json.Marshal(workerPoolAutoscalerConfigs)
	if marshalErr != nil {
		return marshalErr
	}
	configMap.Data[AutoscalerConfigMapOptionProperty] = string(newConfigJsonStr)

	_, updateErr := k8sClient.CoreV1().ConfigMaps(ConfigMapNamespace).Update(context.Background(), configMap, metav1.UpdateOptions{})
	if updateErr != nil {
		return updateErr
	}

	return nil
}

func (ic *IbmClusterHandler) validateAndGetSubnetInfo(networkInfo irs.NetworkInfo) (irs.SubnetInfo, error) {
	if len(networkInfo.SubnetIIDs) > 1 {
		return irs.SubnetInfo{}, errors.New("IBM Kubernetes cluster can be created with only 1 subnet per zone")
	}

	vpcHandler := IbmVPCHandler{
		CredentialInfo: ic.CredentialInfo,
		Region:         ic.Region,
		VpcService:     ic.VpcService,
		Ctx:            ic.Ctx,
	}
	vpcInfo, getVpcErr := vpcHandler.GetVPC(networkInfo.VpcIID)
	if getVpcErr != nil {
		return irs.SubnetInfo{}, getVpcErr
	}

	for _, subnetInfo := range vpcInfo.SubnetInfoList {
		if subnetInfo.IId.NameId == networkInfo.SubnetIIDs[0].NameId ||
			subnetInfo.IId.SystemId == networkInfo.SubnetIIDs[0].SystemId {
			return subnetInfo, nil
		}
	}
	return irs.SubnetInfo{}, errors.New(fmt.Sprintf("Cannot use given subnet: %v from VPC: %v", networkInfo.SubnetIIDs[0], vpcInfo.IId))
}

func (ic *IbmClusterHandler) validateAtCreateCluster(clusterInfo irs.ClusterInfo) error {
	if clusterInfo.IId.NameId == "" {
		return errors.New("Cluster name is required")
	}
	if clusterInfo.Network.VpcIID.SystemId == "" && clusterInfo.Network.VpcIID.NameId == "" {
		return errors.New(fmt.Sprintf("Cannot identify VPC. IID: %s", clusterInfo.Network.VpcIID))
	}
	if len(clusterInfo.Network.SubnetIIDs) < 1 {
		return errors.New("At least one Subnet must be specified")
	}
	if len(clusterInfo.NodeGroupList) < 1 {
		return errors.New("At least one Node Group must be specified")
	}
	if clusterInfo.Version == "" || clusterInfo.Version == "default" {
		clusterInfo.Version = "1.24.8"
	}
	for i, nodeGroup := range clusterInfo.NodeGroupList {
		if i != 0 && nodeGroup.IId.NameId == "" {
			return errors.New(fmt.Sprintf("Node Group name is required for Node Group #%d ", i))
		}
		if nodeGroup.MaxNodeSize < 1 {
			return errors.New(fmt.Sprintf("MaxNodeSize of Node Group: %s cannot be smaller than 1", nodeGroup.IId))
		}
		if nodeGroup.MinNodeSize < 1 {
			return errors.New(fmt.Sprintf("MinNodeSize of Node Group: %s cannot be smaller than 1", nodeGroup.IId))
		}
		if nodeGroup.DesiredNodeSize < 1 {
			return errors.New(fmt.Sprintf("DesiredNodeSize of Node Group: %s cannot be smaller than 1", nodeGroup.IId))
		}
		if nodeGroup.VMSpecName == "" {
			return errors.New("VM Spec Name is required")
		}
	}

	return nil
}

func (ic *IbmClusterHandler) validateAtAddNodeGroup(clusterIID irs.IID, nodeGroupInfo irs.NodeGroupInfo) interface{} {
	if clusterIID.SystemId == "" && clusterIID.NameId == "" {
		return errors.New("Invalid Cluster IID")
	}
	if nodeGroupInfo.IId.NameId == "" {
		return errors.New("Node Group name is required")
	}
	if nodeGroupInfo.MaxNodeSize < 1 {
		return errors.New("MaxNodeSize cannot be smaller than 1")
	}
	if nodeGroupInfo.MinNodeSize < 1 {
		return errors.New("MaxNodeSize cannot be smaller than 1")
	}
	if nodeGroupInfo.DesiredNodeSize < 1 {
		return errors.New("DesiredNodeSize cannot be smaller than 1")
	}
	if nodeGroupInfo.VMSpecName == "" {
		return errors.New("VM Spec Name is required")
	}

	return nil
}

func (ic *IbmClusterHandler) validateAtChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID, minNodeSize int, maxNodeSize int) error {
	if clusterIID.SystemId == "" && clusterIID.NameId == "" {
		return errors.New("Invalid Cluster IID")
	}
	if nodeGroupIID.SystemId == "" && nodeGroupIID.NameId == "" {
		return errors.New("Invalid Node Group IID")
	}
	if minNodeSize < 1 {
		return errors.New("MaxNodeSize cannot be smaller than 1")
	}
	if maxNodeSize < 1 {
		return errors.New("MaxNodeSize cannot be smaller than 1")
	}

	return nil
}

func compareTag(tag string, statusCode string, status string) bool {
	return strings.EqualFold(tag, fmt.Sprintf("%s%s", statusCode, status))
}

func isTagStatusOf(tag string, statusCode string) bool {
	return strings.Contains(tag, strings.ToLower(statusCode))
}
