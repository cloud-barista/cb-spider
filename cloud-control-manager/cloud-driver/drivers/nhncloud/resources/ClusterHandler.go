// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2022/08, 2024/04

package resources

import (
	"errors"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/flavors"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/servers"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/containerinfra/v1/clusters"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/containerinfra/v1/nodegroups"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/imageservice/v2/images"
	netsecgroups "github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/extensions/security/groups"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/networks"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/subnets"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/vpcs"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/vpcsubnets"
	"golang.org/x/mod/semver"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	clusterTemplateId               = "iaas_console"
	createTimeout                   = 60
	secgroupPostfix                 = "secgroup_kube_minion"
	masterNodeGroup                 = "default-master"
	publicNetworkSubnet             = "Public Network Subnet"
	defaultContainerImageNamePrefix = "Ubuntu Server 22"

	clusterLabelsAvailabilityZone          = "availability_zone"
	clusterLabelsCertManagerApi            = "cert_manager_api"
	clusterLabelsClusterautoscale          = "clusterautoscale"
	clusterLabelsExternalNetworkId         = "external_network_id"
	clusterLabelsExternalSubnetIdList      = "external_subnet_id_list"
	clusterLabelsKubeTag                   = "kube_tag"
	clusterLabelsMasterLbFloatingIpEnabled = "master_lb_floating_ip_enabled"
	clusterLabelsNodeImage                 = "node_image"
	clusterLabelsCniDriver                 = "cni_driver"
	clusterLabelsServiceClusterIpRange     = "service_cluster_ip_range"
	clusterLabelsPodsNetworkCidr           = "pods_network_cidr"
	clusterLabelsPodsNetworkSubnet         = "pods_network_subnet"
	clusterLabelsAdditionalNetworkIdList   = "additional_network_id_list"
	clusterLabelsAdditionalSubnetIdList    = "additional_subnet_id_list"
	clusterLabelsBootVolumeType            = "boot_volume_type"
	clusterLabelsBootVolumeSize            = "boot_volume_size"
	clusterLabelsCaEnable                  = "ca_enable"
	clusterLabelsCaMaxNodeCount            = "ca_max_node_count"
	clusterLabelsCaMinNodeCount            = "ca_min_node_count"
	clusterLabelsCaScaleDownEnable         = "ca_scale_down_enable"
	clusterLabelsCaScaleDownUnneededTime   = "ca_scale_down_unneeded_time"
	clusterLabelsCaScaleDownUtilThresh     = "ca_scale_down_util_thresh"
	clusterLabelsCaScaleDownDelayAfterAdd  = "ca_scale_down_delay_after_add"
	clusterLabelsUserScriptV2              = "user_script_v2"

	clusterStatusCreateInProgress   = "CREATE_IN_PROGRESS"
	clusterStatusCreateFailed       = "CREATE_FAILED"
	clusterStatusCreateComplete     = "CREATE_COMPLETE"
	clusterStatusUpdateInProgress   = "UPDATE_IN_PROGRESS"
	clusterStatusUpdateFailed       = "UPDATE_FAILED"
	clusterStatusUpdateComplete     = "UPDATE_COMPLETE"
	clusterStatusDeleteInProgress   = "DELETE_IN_PROGRESS"
	clusterStatusDeleteFailed       = "DELETE_FAILED"
	clusterStatusDeleteComplete     = "DELETE_COMPLETE"
	clusterStatusUpgradeInProgress  = "UPGRADE_IN_PROGRESS"
	clusterStatusUpgradeFailed      = "UPGRADE_FAILED"
	clusterStatusUpgradeComplete    = "UPGRADE_COMPLETE"
	clusterStatusResumeComplete     = "RESUME_COMPLETE"
	clusterStatusResumeFailed       = "RESUME_FAILED"
	clusterStatusRestoreComplete    = "RESTORE_COMPLETE"
	clusterStatusRollbackInProgress = "ROLLBACK_IN_PROGRESS"
	clusterStatusRollbackFailed     = "ROLLBACK_FAILED"
	clusterStatusRollbackComplete   = "ROLLBACK_COMPLETE"
	clusterStatusSnapshotComplete   = "SNAPSHOT_COMPLETE"
	clusterStatusCheckComplete      = "CHECK_COMPLETE"
	clusterStatusAdoptComplete      = "ADOPT_COMPLETE"
)

type NhnCloudClusterHandler struct {
	RegionInfo    idrv.RegionInfo
	VMClient      *nhnsdk.ServiceClient
	ImageClient   *nhnsdk.ServiceClient
	NetworkClient *nhnsdk.ServiceClient
	ClusterClient *nhnsdk.ServiceClient
}

func (nch *NhnCloudClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (irs.ClusterInfo, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NHN Cloud Driver: called CreateCluster()")
	emptyClusterInfo := irs.ClusterInfo{}
	hiscallInfo := getCallLogScheme(nch.RegionInfo.Region, call.CLUSTER, clusterReqInfo.IId.NameId, "CreateCluster()")
	start := call.Start()

	cblogger.Info("Create Cluster")

	var clusterId string
	var createErr error
	defer func() {
		if createErr != nil {
			cblogger.Error(createErr)
			LoggingError(hiscallInfo, createErr)

			if clusterId != "" {
				_ = nch.deleteCluster(clusterId)
				cblogger.Infof("Cluster(Name=%s) will be Deleted.", clusterReqInfo.IId.NameId)
			}
		}
	}()

	//
	// Validation
	//
	supportedK8sVersions, err := nch.getSupportedK8sVersions()
	if err != nil {
		createErr = fmt.Errorf("Failed to Create Cluster: %v", err)
		return emptyClusterInfo, createErr
	}

	err = validateAtCreateCluster(clusterReqInfo, supportedK8sVersions)
	if err != nil {
		createErr = fmt.Errorf("Failed to Create Cluster: %v", err)
		return emptyClusterInfo, createErr
	}

	//
	// Create Cluster
	//
	clusterId, err = nch.createCluster(&clusterReqInfo)
	if err != nil {
		createErr = fmt.Errorf("Failed to Create Cluster: %v", err)
		return emptyClusterInfo, createErr
	}
	cblogger.Debug("To Create a Cluster is In Progress.")

	err = nch.waitUntilClusterSecGroupIsCreated(clusterId)
	if err != nil {
		createErr = fmt.Errorf("Failed to Create Cluster: %v", err)
		return emptyClusterInfo, createErr
	}

	//
	// Get ClusterInfo
	//
	clusterInfo, err := nch.getClusterInfo(clusterId)
	if err != nil {
		createErr = fmt.Errorf("Failed to Create Cluster: %v", err)
		return emptyClusterInfo, createErr
	}

	LoggingInfo(hiscallInfo, start)

	cblogger.Infof("Creating Cluster(name=%s, id=%s)", clusterInfo.IId.NameId, clusterInfo.IId.SystemId)

	return *clusterInfo, nil
}

func (nch *NhnCloudClusterHandler) ListCluster() ([]*irs.ClusterInfo, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NHN Cloud Driver: called ListCluster()")
	hiscallInfo := getCallLogScheme(nch.RegionInfo.Region, call.CLUSTER, "ListCluster()", "ListCluster()") // HisCall logging
	start := call.Start()

	cblogger.Info("Get Cluster List")

	var listErr error
	defer func() {
		if listErr != nil {
			cblogger.Error(listErr)
			LoggingError(hiscallInfo, listErr)
		}
	}()

	//
	// Get Cluster List
	//
	clusterList, err := nhnGetClusterList(nch.ClusterClient)
	if err != nil {
		listErr = fmt.Errorf("Failed to List Cluster: %v", err)
		return nil, listErr
	}

	//
	// Get ClusterInfo List
	//
	var clusterInfoList []*irs.ClusterInfo
	for _, cluster := range clusterList {
		clusterInfo, err := nch.getClusterInfo(cluster.UUID)
		if err != nil {
			listErr = fmt.Errorf("Failed to List Cluster: %v", err)
			return nil, listErr
		}

		clusterInfoList = append(clusterInfoList, clusterInfo)
	}

	LoggingInfo(hiscallInfo, start)

	return clusterInfoList, nil
}

func (nch *NhnCloudClusterHandler) GetCluster(clusterIID irs.IID) (irs.ClusterInfo, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NHN Cloud Driver: called GetCluster()")
	emptyClusterInfo := irs.ClusterInfo{}
	hiscallInfo := getCallLogScheme(nch.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "GetCluster()")
	start := call.Start()

	cblogger.Info("Get Cluster")

	var getErr error
	defer func() {
		if getErr != nil {
			cblogger.Error(getErr)
			LoggingError(hiscallInfo, getErr)
		}
	}()

	//
	// Get ClusterInfo
	//
	cluster, err := nhnGetRawCluster(nch.ClusterClient, clusterIID)
	if err != nil {
		getErr = fmt.Errorf("Failed to Get Cluster: %v", err)
		return emptyClusterInfo, getErr
	}

	clusterInfo, err := nch.getClusterInfo(cluster.UUID)
	if err != nil {
		getErr = fmt.Errorf("Failed to Get Cluster: %v", err)
		return emptyClusterInfo, getErr
	}

	LoggingInfo(hiscallInfo, start)

	cblogger.Infof("Get Cluster(Name=%s, ID=%s)", clusterInfo.IId.NameId, clusterInfo.IId.SystemId)

	return *clusterInfo, nil
}

func (nch *NhnCloudClusterHandler) DeleteCluster(clusterIID irs.IID) (bool, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NHN Cloud Driver: called DeleteCluster()")
	hiscallInfo := getCallLogScheme(nch.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "DeleteCluster()")
	start := call.Start()

	cblogger.Info("Delete Cluster")

	var delErr error
	defer func() {
		if delErr != nil {
			cblogger.Error(delErr)
			LoggingError(hiscallInfo, delErr)
		}
	}()

	//
	// Delete Cluster
	//
	cluster, err := nhnGetRawCluster(nch.ClusterClient, clusterIID)
	if err != nil {
		delErr = fmt.Errorf("Failed to Get Cluster: %v", err)
		return false, delErr
	}

	err = nch.deleteCluster(cluster.UUID)
	if err != nil {
		delErr = fmt.Errorf("Failed to Delete Cluster: %v", err)
		return false, delErr
	}

	LoggingInfo(hiscallInfo, start)

	cblogger.Infof("Deleting Cluster(name=%s, id=%s)", cluster.Name, cluster.UUID)

	return true, nil
}

// 업그레이드 순서
// default-master 노드 그룹을 업그레이드한다.
// default-master 업그레이드가 완료되면, worker 노드들을 업그레이드 한다.
// default-master 업그레이드가 완료되기 전에는 worker 노드를 업그레이드 할 수 없다.
// default-master 업그레이드가 완료된 후에 (10분? 정도 소요됨) worker 노드를 업그레이드해야 한다.
func (nch *NhnCloudClusterHandler) UpgradeCluster(clusterIID irs.IID, newVersion string) (irs.ClusterInfo, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NHN Cloud Driver: called UpgradeCluster()")
	emptyClusterInfo := irs.ClusterInfo{}
	hiscallInfo := getCallLogScheme(nch.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "UpgradeCluster()")
	start := call.Start()

	cblogger.Info("Upgrade Cluster")

	var upgradeErr error
	defer func() {
		if upgradeErr != nil {
			cblogger.Error(upgradeErr)
			LoggingError(hiscallInfo, upgradeErr)
		}
	}()

	//
	// Upgrade Cluster
	// https://docs.nhncloud.com/ko/Container/NKS/ko/public-api/#_57
	//
	cluster, err := nhnGetRawCluster(nch.ClusterClient, clusterIID)
	if err != nil {
		upgradeErr = fmt.Errorf("Failed to Upgrade Cluster: %v", err)
		return emptyClusterInfo, upgradeErr
	}

	clusterVersion, err := getClusterVersion(cluster)
	if err != nil {
		upgradeErr = fmt.Errorf("Failed to Upgrade Cluster: %v", err)
		return emptyClusterInfo, upgradeErr
	}

	if semver.Compare(clusterVersion, newVersion) < 0 {
		// At first, upgrades a node group for master: it takes 8~10 minutes
		err := nhnUpgradeCluster(nch.ClusterClient, cluster.UUID, masterNodeGroup, newVersion)
		if err != nil {
			upgradeErr = fmt.Errorf("Failed to Upgrade Cluster: %v", err)
			return emptyClusterInfo, upgradeErr
		}
		cblogger.Debug("To Upgrade a Cluster is In Progress")

		// When upgrading a cluster(masterNodeGroup), the status of cluster is UpdateInProgress
		err = nch.waitUntilClusterIsStatus(cluster.UUID, clusterStatusUpdateComplete)
		if err != nil {
			upgradeErr = fmt.Errorf("Failed to Upgrade Cluster: %v", err)
			return emptyClusterInfo, upgradeErr
		}
		cblogger.Info("To Upgrade a Cluster/Master(id=%s) is Completed", cluster.UUID)
	}

	// And then, upgrades node groups for worker
	nodeGroupList, err := nhnGetNodeGroupList(nch.ClusterClient, cluster.UUID)
	if err != nil {
		upgradeErr = fmt.Errorf("Failed to Upgrade Cluster: %v", err)
		return emptyClusterInfo, upgradeErr
	}

	for _, nodeGroup := range nodeGroupList {
		nodeGroupId := nodeGroup.UUID

		nodeGroupDetail, err := nhnGetRawNodeGroup(nch.ClusterClient, cluster.UUID, irs.IID{SystemId: nodeGroupId})
		if err != nil {
			upgradeErr = fmt.Errorf("Failed to Upgrade Cluster: %v", err)
			return emptyClusterInfo, upgradeErr
		}
		cblogger.Debug("To Upgrade a NodeGroup is In Progress")

		nodeGroupVersion, err := getNodeGroupVersion(nodeGroupDetail)
		if err != nil {
			upgradeErr = fmt.Errorf("Failed to Upgrade Cluster: %v", err)
			return emptyClusterInfo, upgradeErr
		}

		if semver.Compare(nodeGroupVersion, newVersion) < 0 {
			err := nhnUpgradeCluster(nch.ClusterClient, cluster.UUID, nodeGroupId, newVersion)
			if err != nil {
				upgradeErr = fmt.Errorf("Failed to Upgrade Cluster: %v", err)
				return emptyClusterInfo, upgradeErr
			}

			// When upgrading a nodegroup, the status of cluster is UpgradeInProress and the status of nodegroup is UpdateInProgress
			err = nch.waitUntilNodeGroupIsStatus(cluster.UUID, nodeGroupId, clusterStatusUpdateComplete)
			if err != nil {
				upgradeErr = fmt.Errorf("Failed to Upgrade Cluster: %v", err)
				return emptyClusterInfo, upgradeErr
			}

			cblogger.Infof("To Upgrade a NodeGroup(id=%s) is Completed", nodeGroupId)
		}
	}

	//
	// Get ClusterInfo
	//
	clusterInfo, err := nch.getClusterInfo(cluster.UUID)
	if err != nil {
		err = fmt.Errorf("Failed to Upgrade Cluster: %v", err)
		return emptyClusterInfo, err
	}

	LoggingInfo(hiscallInfo, start)

	cblogger.Infof("Upgrading Cluster(name=%s, id=%s)", clusterInfo.IId.NameId, clusterInfo.IId.SystemId)

	return *clusterInfo, nil

}

func (nch *NhnCloudClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (irs.NodeGroupInfo, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NHN Cloud Driver: called AddNodeGroup()")
	emptyNodeGroupInfo := irs.NodeGroupInfo{}
	hiscallInfo := getCallLogScheme(nch.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "AddNodeGroup()")
	start := call.Start()

	cblogger.Info("Add NodeGroup")

	var addErr error
	var clusterId, nodeGroupId string
	defer func() {
		if addErr != nil {
			cblogger.Error(addErr)
			LoggingError(hiscallInfo, addErr)

			if clusterId != "" && nodeGroupId != "" {
				_ = nch.deleteNodeGroup(clusterId, nodeGroupId)
				cblogger.Infof("NodeGroup(id=%s) of Cluster(id=%s) will be Deleted", nodeGroupId, clusterId)
			}
		}
	}()

	//
	// Validation
	//
	err := validateAtAddNodeGroup(clusterIID, nodeGroupReqInfo)
	if err != nil {
		err = fmt.Errorf("Failed to Add Node Group: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyNodeGroupInfo, err
	}

	//
	// Create Node Group
	//
	cluster, err := nhnGetRawCluster(nch.ClusterClient, clusterIID)
	if err != nil {
		err = fmt.Errorf("Failed to Upgrade Cluster: %v", err)
		return emptyNodeGroupInfo, err
	}

	nodeGroupId, err = nch.createNodeGroup(cluster.UUID, &nodeGroupReqInfo)
	if err != nil {
		err = fmt.Errorf("Failed to Add NodeGroup: %v", err)
		return emptyNodeGroupInfo, err
	}
	cblogger.Debug("To Create a NodeGroup is In Progress")

	//
	// Get NodeGroupInfo
	//
	nodeGroupInfo, err := nch.getNodeGroupInfo(cluster.UUID, nodeGroupId, cluster.KeyPair)
	if err != nil {
		addErr = fmt.Errorf("Failed to Add NodeGroup: %v", err)
		return emptyNodeGroupInfo, addErr
	}

	LoggingInfo(hiscallInfo, start)

	cblogger.Infof("Adding NodeGroup(name=%s, id=%s) to Cluster(%s)", nodeGroupInfo.IId.NameId, nodeGroupInfo.IId.SystemId, clusterId)

	return *nodeGroupInfo, nil
}

func (nch *NhnCloudClusterHandler) SetNodeGroupAutoScaling(clusterIID irs.IID, nodeGroupIID irs.IID, on bool) (bool, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NHN Cloud Driver: called SetNodeGroupAutoScaling()")
	hiscallInfo := getCallLogScheme(nch.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "GetNodeGroup()")
	start := call.Start()

	cblogger.Info("Set NodeGroup AutoScaling")

	//
	// Set NodeGroup AutoScaling
	//
	cluster, err := nhnGetRawCluster(nch.ClusterClient, clusterIID)
	if err != nil {
		err = fmt.Errorf("Failed to Upgrade Cluster: %v", err)
		return false, err
	}

	nodeGroup, err := nhnGetRawNodeGroup(nch.ClusterClient, cluster.UUID, nodeGroupIID)
	if err != nil {
		err = fmt.Errorf("Failed to Upgrade Cluster: %v", err)
		return false, err
	}

	clusterId := cluster.UUID
	nodeGroupId := nodeGroup.UUID
	enable := on

	_, err = nhnSetNodeGroupAutoscaleEnable(nch.ClusterClient, clusterId, nodeGroupId, enable)
	if err != nil {
		err := fmt.Errorf("Failed to Set NodeGroup AutoScaling: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	LoggingInfo(hiscallInfo, start)

	cblogger.Infof("Modifying AutoScaling of NodeGroup(name=%s, id=%s) in Cluster(%s)", nodeGroup.Name, nodeGroup.UUID, cluster.Name)

	return true, nil
}

func (nch *NhnCloudClusterHandler) ChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID, desiredNodeSize int, minNodeSize int, maxNodeSize int) (irs.NodeGroupInfo, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NHN Cloud Driver: called ChangeNodeGroupScaling()")
	emptyNodeGroupInfo := irs.NodeGroupInfo{}
	hiscallInfo := getCallLogScheme(nch.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "GetNodeGroup()")
	start := call.Start()

	cblogger.Info("Change NodeGroup Scaling")

	//
	// Validation
	//
	err := validateAtChangeNodeGroupScaling(minNodeSize, maxNodeSize)
	if err != nil {
		err = fmt.Errorf("Failed to Change Node Group Scaling: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyNodeGroupInfo, err
	}

	//
	// Change NodeGroup's Scaling Size
	//
	cluster, err := nhnGetRawCluster(nch.ClusterClient, clusterIID)
	if err != nil {
		err = fmt.Errorf("Failed to Change NodeGroup Scaling: %v", err)
		return emptyNodeGroupInfo, err
	}

	nodeGroup, err := nhnGetRawNodeGroup(nch.ClusterClient, cluster.UUID, nodeGroupIID)
	if err != nil {
		err = fmt.Errorf("Failed to Change NodeGroup Scaling: %v", err)
		return emptyNodeGroupInfo, err
	}

	nodeGroupId := nodeGroup.UUID

	enable := true
	// CAUTION: desiredNodeSize cannot be applied in NHN Cloud
	minNodeCount := minNodeSize
	maxNodeCount := maxNodeSize

	// Check if CurrentNodeCount >= minNodeCount or minNodeCount >= 1
	// And CurrentNodeCount <= maxNodeCount or maxNodeCount <= 10
	nodeCount := nodeGroup.NodeCount
	if minNodeCount < 1 {
		minNodeCount = 1
		cblogger.Info("MinNodeSize must be 1 or greater. It will be set to 1.")
	}
	if nodeCount < minNodeCount {
		minNodeCount = nodeCount
		cblogger.Infof("MinNodeSize must be less than or equal to current node count. It will be set to current node count(%d).", nodeCount)
	}

	if maxNodeCount > 10 {
		maxNodeCount = 10
		cblogger.Info("MaxNodeSize must be 10 or less. It will be set to 10.")
	}
	if nodeCount > maxNodeCount {
		maxNodeCount = nodeCount
		cblogger.Infof("MaxNodeSize must be greater than or equal to current node count. It will be set to current node count(%d).", nodeCount)
	}

	// Set NodeGroup's Autoscale
	_, err = nhnSetNodeGroupAutoscale(nch.ClusterClient, cluster.UUID, nodeGroupId, enable, minNodeCount, maxNodeCount)
	if err != nil {
		err = fmt.Errorf("Failed to Change NodeGroup Scaling: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyNodeGroupInfo, err
	}

	//
	// Get NodeGroupInfo
	//
	nodeGroupInfo, err := nch.getNodeGroupInfo(cluster.UUID, nodeGroupId, cluster.KeyPair)
	if err != nil {
		err = fmt.Errorf("Failed to Change NodeGroup Scaling: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyNodeGroupInfo, err
	}

	LoggingInfo(hiscallInfo, start)

	cblogger.Infof("Modifying Scaling of NodeGroup(id=%s) in Cluster(id=%s).", nodeGroupId, cluster.UUID)

	return *nodeGroupInfo, nil
}

func (nch *NhnCloudClusterHandler) RemoveNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (bool, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NHN Cloud Driver: called RemoveNodeGroup()")
	hiscallInfo := getCallLogScheme(nch.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "RemoveNodeGroup()")
	start := call.Start()

	cblogger.Info("Remove NodeGroup")

	var removeErr error
	defer func() {
		if removeErr != nil {
			cblogger.Error(removeErr)
			LoggingError(hiscallInfo, removeErr)
		}
	}()

	//
	// Remove NodeGroup
	//
	cluster, err := nhnGetRawCluster(nch.ClusterClient, clusterIID)
	if err != nil {
		err = fmt.Errorf("Failed to Upgrade Cluster: %v", err)
		return false, err
	}

	nodeGroup, err := nhnGetRawNodeGroup(nch.ClusterClient, cluster.UUID, nodeGroupIID)
	if err != nil {
		err = fmt.Errorf("Failed to Change NodeGroup Scaling: %v", err)
		return false, err
	}

	err = nch.deleteNodeGroup(cluster.UUID, nodeGroup.UUID)
	if err != nil {
		err := fmt.Errorf("Failed to Remove NodeGroup: %v", err)
		return false, err
	}

	LoggingInfo(hiscallInfo, start)

	cblogger.Infof("Removing NodeGroup(name=%s, id=%s) to Cluster(%s)", nodeGroup.Name, nodeGroup.UUID, cluster.Name)

	return true, nil
}

func (nch *NhnCloudClusterHandler) getClusterInfoWithoutNodeGroupList(clusterId string) (*irs.ClusterInfo, string, error) {
	//
	// Fill clusterInfo
	//
	cluster, err := nhnGetRawCluster(nch.ClusterClient, irs.IID{SystemId: clusterId})
	if err != nil {
		err = fmt.Errorf("failed to get cluster(id=%s): %v", clusterId, err)
		return nil, "", err
	}

	clusterName := cluster.Name

	version, err := getClusterVersion(cluster)
	if err != nil {
		err = fmt.Errorf("failed to get cluster version: %v", err)
		return nil, "", err
	}

	networkInfo, err := nch.getClusterNetworkInfo(cluster)
	if err != nil {
		err = fmt.Errorf("failed to get network info: %v", err)
		return nil, "", err
	}

	ciStatus := convertClusterStatusToClusterInfoStatus(cluster)
	createdTime := cluster.CreatedAt
	accessInfo, err := nch.getClusterAccessInfo(cluster)
	if err != nil {
		err = fmt.Errorf("failed to get access info: %v", err)
		return nil, "", err
	}

	clusterInfo := &irs.ClusterInfo{
		IId: irs.IID{
			NameId:   clusterName,
			SystemId: clusterId,
		},
		Version:     version,
		Network:     *networkInfo,
		Status:      ciStatus,
		CreatedTime: createdTime,
		AccessInfo:  *accessInfo,
	}

	//
	// Fill clusterInfo.KeyValueList
	//
	clusterInfo.KeyValueList = irs.StructToKeyValueList(cluster)

	return clusterInfo, cluster.KeyPair, nil
}

func (nch *NhnCloudClusterHandler) getClusterInfoListWithoutNodeGroupList() ([]*irs.ClusterInfo, error) {
	clusterList, err := nhnGetClusterList(nch.ClusterClient)
	if err != nil {
		err = fmt.Errorf("failed to get cluster list: %v", err)
		return nil, err
	}

	var clusterInfoList []*irs.ClusterInfo
	for _, cluster := range clusterList {
		clusterInfo, _, err := nch.getClusterInfoWithoutNodeGroupList(cluster.UUID)
		if err != nil {
			err = fmt.Errorf("failed to get ClusterInfo: %v", err)
			return nil, err
		}

		clusterInfoList = append(clusterInfoList, clusterInfo)
	}

	return clusterInfoList, nil
}

func (nch *NhnCloudClusterHandler) getClusterInfo(clusterId string) (clusterInfo *irs.ClusterInfo, err error) {
	//
	// Fill clusterInfo
	//
	clusterInfo, clusterKeyPair, err := nch.getClusterInfoWithoutNodeGroupList(clusterId)
	if err != nil {
		err = fmt.Errorf("failed to get ClusterInfo: %v", err)
		return nil, err
	}

	for _, kv := range clusterInfo.KeyValueList {
		// If creation of a cluster is failed, DO NOT need to get nodegroups' information
		if strings.EqualFold(kv.Key, "status") &&
			strings.EqualFold(kv.Value, clusterStatusCreateFailed) {
			return clusterInfo, nil
		}
	}

	//
	// Fill clusterInfo.NodeGroupList
	//
	nodeGroupList, err := nhnGetNodeGroupList(nch.ClusterClient, clusterId)
	if err != nil {
		err = fmt.Errorf("failed to get node group list: %v", err)
		return nil, err
	}

	for _, nodeGroup := range nodeGroupList {
		nodeGroupId := nodeGroup.UUID
		nodeGroupInfo, err := nch.getNodeGroupInfo(clusterId, nodeGroupId, clusterKeyPair)
		if err != nil {
			err = fmt.Errorf("failed to get node group info: %v", err)
			return nil, err
		}

		clusterInfo.NodeGroupList = append(clusterInfo.NodeGroupList, *nodeGroupInfo)
	}

	return clusterInfo, nil
}

func (nch *NhnCloudClusterHandler) getClusterNetworkInfo(cluster *clusters.Cluster) (*irs.NetworkInfo, error) {
	fixedNetworkId := cluster.FixedNetwork
	fixedNetworkName, err := nch.getVpcName(fixedNetworkId)
	if err != nil {
		err = fmt.Errorf("failed to get the cluster's vpc: %v", err)
		return nil, err
	}

	fixedSubnetId := cluster.FixedSubnet
	fixedSubnetName, err := nch.getVpcsubnetName(fixedSubnetId)
	if err != nil {
		err = fmt.Errorf("failed to get the cluster's subnet: %v", err)
		return nil, err
	}

	secGroupId, err := nch.getClusterSecGroupId(cluster)
	if err != nil {
		err = fmt.Errorf("failed to get the cluster's securitygroup: %v", err)
		return nil, err
	}

	networkInfo := &irs.NetworkInfo{
		VpcIID: irs.IID{
			NameId:   fixedNetworkName,
			SystemId: fixedNetworkId,
		},
		SubnetIIDs:        []irs.IID{{NameId: fixedSubnetName, SystemId: fixedSubnetId}},
		SecurityGroupIIDs: []irs.IID{*secGroupId},
	}

	return networkInfo, nil
}

func (nch *NhnCloudClusterHandler) getClusterAddonInfo(cluster *clusters.Cluster) (info irs.AddonsInfo, err error) {
	keyvalues := make([]irs.KeyValue, 0)

	info = irs.AddonsInfo{
		keyvalues,
	}
	return info, nil
}

func (nch *NhnCloudClusterHandler) getNodeGroupInfo(clusterId, nodeGroupId, keyPair string) (*irs.NodeGroupInfo, error) {
	//
	// Fill nodeGroupInfo
	//
	nodeGroup, err := nhnGetRawNodeGroup(nch.ClusterClient, clusterId, irs.IID{SystemId: nodeGroupId})
	if err != nil {
		err = fmt.Errorf("failed to get nodegroup info: %v", err)
		return nil, err
	}

	nodeGroupName := nodeGroup.Name

	imageId := nodeGroup.ImageID
	imageName, err := nch.getImageNameById(imageId)
	if err != nil {
		err = fmt.Errorf("failed to get image name by id(%s): %v", imageId, err)
		return nil, err
	}

	flavorId := nodeGroup.FlavorID
	flavorName, err := nch.getFlavorNameById(flavorId)
	if err != nil {
		err = fmt.Errorf("failed to get flavor name by id(%s): %v", flavorId, err)
		return nil, err
	}

	var rootDiskType string
	if bootVolumeType, ok := nodeGroup.Labels[clusterLabelsBootVolumeType]; !ok {
		err = fmt.Errorf("failed to get nodegroup info: no %s field", clusterLabelsBootVolumeType)
		return nil, err
	} else {
		rootDiskType = bootVolumeType
	}

	var rootDiskSize string
	if bootVolumeSize, ok := nodeGroup.Labels[clusterLabelsBootVolumeSize]; !ok {
		err = fmt.Errorf("failed to get nodegroup info: no %s field", clusterLabelsBootVolumeSize)
		return nil, err
	} else {
		rootDiskSize = bootVolumeSize
	}

	ngiStatus := convertNodeGroupStatusToNodeGroupInfoStatus(nodeGroup)

	var onAutoScaling bool
	if caEnable, ok := nodeGroup.Labels[clusterLabelsCaEnable]; !ok {
		err = fmt.Errorf("failed to get nodegroup info: no %s field", clusterLabelsCaEnable)
		return nil, err
	} else {
		onAutoScaling, _ = strconv.ParseBool(caEnable)
	}

	var minNodeSize int
	if caMinNodeCount, ok := nodeGroup.Labels[clusterLabelsCaMinNodeCount]; !ok {
		err = fmt.Errorf("failed to get nodegroup info: no %s field", clusterLabelsCaMinNodeCount)
		return nil, err
	} else {
		nodeSize, _ := strconv.ParseInt(caMinNodeCount, 10, 64)
		minNodeSize = int(nodeSize)
	}

	var maxNodeSize int
	if caMaxNodeCount, ok := nodeGroup.Labels[clusterLabelsCaMaxNodeCount]; !ok {
		err = fmt.Errorf("failed to get nodegroup info: no %s field", clusterLabelsCaMaxNodeCount)
		return nil, err
	} else {
		nodeSize, _ := strconv.ParseInt(caMaxNodeCount, 10, 64)
		maxNodeSize = int(nodeSize)
	}

	desiredNodeSize := nodeGroup.NodeCount

	nodeGroupInfo := &irs.NodeGroupInfo{
		IId: irs.IID{
			NameId:   nodeGroupName,
			SystemId: nodeGroupId,
		},
		ImageIID: irs.IID{
			NameId:   imageName,
			SystemId: imageId,
		},
		VMSpecName:   flavorName,
		RootDiskType: rootDiskType,
		RootDiskSize: rootDiskSize,
		KeyPairIID: irs.IID{
			NameId:   keyPair,
			SystemId: keyPair,
		},
		Status:          ngiStatus,
		OnAutoScaling:   onAutoScaling,
		MinNodeSize:     minNodeSize,
		MaxNodeSize:     maxNodeSize,
		DesiredNodeSize: desiredNodeSize,
	}

	//
	// Fill nodeGroupInfo.Nodes
	//
	if ngiStatus == irs.NodeGroupActive {
		serverList, err := nch.getServerListByAddresses(imageId, flavorId, nodeGroup.NodeAddresses)
		if err != nil {
			err = fmt.Errorf("failed to get server lists by addresses(%v): %v", nodeGroup.NodeAddresses, err)
			return nil, err
		}
		for _, server := range serverList {
			nodeIId := irs.IID{NameId: server.Name, SystemId: server.ID}
			nodeGroupInfo.Nodes = append(nodeGroupInfo.Nodes, nodeIId)
		}
	}

	//
	// Fill nodeGroupInfo.KeyValueList
	//
	nodeGroupInfo.KeyValueList = irs.StructToKeyValueList(nodeGroup)

	return nodeGroupInfo, err
}

func (nch *NhnCloudClusterHandler) getLabelsForCluster(clusterInfo *irs.ClusterInfo, nodeGroupInfo *irs.NodeGroupInfo) (map[string]string, error) {
	emptyLabels := make(map[string]string, 0)

	// https://docs.nhncloud.com/ko/Container/NKS/ko/public-api/#_15

	nodeImage := nodeGroupInfo.ImageIID.NameId
	certManagerApi := "True"
	clusterautoscale := "nodegroupfeature"

	kubeTag := clusterInfo.Version
	masterLbFloatingIpEnabled := "True" // Kubernetes API Endpoint set to Public

	var extNetworkId string
	var extSubnetIdList string

	extNetworkList, err := nhnGetNetworkList(nch.NetworkClient, true)
	if err != nil {
		err = fmt.Errorf("failed to get networks' list with external: %v", err)
		return emptyLabels, err
	}

	if len(extNetworkList) > 0 {
		extNetworkId = extNetworkList[0].ID
		if len(extNetworkList[0].Subnets) > 0 {
			extSubnetList := extNetworkList[0].Subnets
			for i, subnet := range extSubnetList {
				if i > 0 {
					extSubnetIdList = extSubnetIdList + ":"
				}
				extSubnetIdList = extSubnetIdList + subnet
			}

		} else {
			err = fmt.Errorf("no subnet with external network(%s)", extNetworkId)
			return emptyLabels, err
		}
	} else {
		err = fmt.Errorf("no external network")
		return emptyLabels, err
	}

	labels := map[string]string{
		clusterLabelsNodeImage:                 nodeImage,
		clusterLabelsExternalNetworkId:         extNetworkId,
		clusterLabelsExternalSubnetIdList:      extSubnetIdList,
		clusterLabelsCertManagerApi:            certManagerApi,
		clusterLabelsClusterautoscale:          clusterautoscale,
		clusterLabelsKubeTag:                   kubeTag,
		clusterLabelsMasterLbFloatingIpEnabled: masterLbFloatingIpEnabled,
	}

	var addVpcIdList string
	var addSubnetIdList string
	if len(clusterInfo.Network.SubnetIIDs) > 1 {
		addVpcId := clusterInfo.Network.VpcIID.SystemId
		for i, addSubnetId := range clusterInfo.Network.SubnetIIDs {
			if i == 0 {
				continue
			} else {
				addVpcIdList = addVpcIdList + ":"
				addSubnetIdList = addSubnetIdList + ":"
			}
			addVpcIdList = addVpcIdList + addVpcId
			addSubnetIdList = addSubnetIdList + addSubnetId.SystemId
		}
	}

	labelsNodeGroup, err := nch.getLabelsForNodeGroup(nodeGroupInfo, addVpcIdList, addSubnetIdList)
	if err != nil {
		err = fmt.Errorf("failed to get labels for node group(%s) of cluster(%s): %v", nodeGroupInfo.IId.NameId, clusterInfo.IId.NameId, err)
		return emptyLabels, err
	}

	for key, value := range labelsNodeGroup {
		labels[key] = value
	}

	return labels, nil
}

func (nch *NhnCloudClusterHandler) getLabelsForNodeGroup(nodeGroupInfo *irs.NodeGroupInfo, addVpcIdList, addSubnetIdList string) (map[string]string, error) {

	// https://docs.nhncloud.com/ko/Container/NKS/ko/public-api/#_39

	availabilityZone := nch.RegionInfo.Zone
	for _, v := range nodeGroupInfo.KeyValueList {
		switch v.Key {
		case "availability_zone":
			availabilityZone = v.Value
			break
		}
	}

	if strings.EqualFold(nodeGroupInfo.RootDiskSize, "") || strings.EqualFold(nodeGroupInfo.RootDiskSize, "default") {
		nodeGroupInfo.RootDiskSize = DefaultDiskSize
	}
	bootVolumeSize := nodeGroupInfo.RootDiskSize

	if strings.EqualFold(nodeGroupInfo.RootDiskType, "") || strings.EqualFold(nodeGroupInfo.RootDiskType, "default") {
		nodeGroupInfo.RootDiskType = HDD
	}
	bootVolumeType := nodeGroupInfo.RootDiskType

	caEnable := "false"
	if nodeGroupInfo.OnAutoScaling {
		caEnable = "true"
	}
	caMaxNodeCount := strconv.Itoa(nodeGroupInfo.MaxNodeSize)
	caMinNodeCount := strconv.Itoa(nodeGroupInfo.MinNodeSize)
	caScaleDownEnable := "true"

	labels := map[string]string{
		clusterLabelsAvailabilityZone:        availabilityZone,
		clusterLabelsBootVolumeSize:          bootVolumeSize,
		clusterLabelsBootVolumeType:          bootVolumeType,
		clusterLabelsCaEnable:                caEnable,
		clusterLabelsCaMaxNodeCount:          caMaxNodeCount,
		clusterLabelsCaMinNodeCount:          caMinNodeCount,
		clusterLabelsCaScaleDownEnable:       caScaleDownEnable,
		clusterLabelsAdditionalNetworkIdList: addVpcIdList,
		clusterLabelsAdditionalSubnetIdList:  addSubnetIdList,
	}

	return labels, nil
}

func (nch *NhnCloudClusterHandler) waitUntilClusterIsStatus(clusterId, status string) error {
	apiCallCount := 0
	maxAPICallCount := 240

	var waitErr error
	for {
		cluster, err := nhnGetRawCluster(nch.ClusterClient, irs.IID{SystemId: clusterId})
		if err != nil {
			maxAPICallCount = maxAPICallCount / 2
			cblogger.Infof("failed to get cluster(id=%s): %v", clusterId, err)
		} else {
			if strings.EqualFold(cluster.Status, status) {
				return nil
			}
		}

		apiCallCount++
		if apiCallCount >= maxAPICallCount {
			waitErr = fmt.Errorf("failed to get cluster: " +
				"The maximum number of verification requests has been exceeded " +
				"while waiting for availability of that resource")
			break
		}
		cblogger.Infof("Wait until cluster(id=%s)'s status is %s", clusterId, status)
		time.Sleep(10 * time.Second)
	}

	return waitErr
}

func (nch *NhnCloudClusterHandler) waitUntilClusterSecGroupIsCreated(clusterId string) error {
	var waitErr error

	// Wait Cluster
	apiCallCount := 0
	maxAPICallCount := 240
	var targetCluster *clusters.Cluster
	for {
		cluster, err := nhnGetRawCluster(nch.ClusterClient, irs.IID{SystemId: clusterId})
		if err != nil {
			maxAPICallCount = maxAPICallCount / 2
			cblogger.Infof("failed to get cluster(id=%s): %v", clusterId, err)
		} else {
			targetCluster = cluster
			break
		}
		apiCallCount++
		if apiCallCount >= maxAPICallCount {
			waitErr = fmt.Errorf("failed to get cluster: " +
				"The maximum number of verification requests has been exceeded " +
				"while waiting for the creation of that resource")
			return waitErr
		}
		cblogger.Infof("Wait until creation of cluster(id=%s) is started.", clusterId)
		time.Sleep(10 * time.Second)
	}

	// Wait Security Group
	apiCallCount = 0
	maxAPICallCount = 240
	var targetSecGroupId *irs.IID
	for {
		secGroupId, err := nch.getClusterSecGroupId(targetCluster)
		if err != nil {
			maxAPICallCount = maxAPICallCount / 2
			cblogger.Infof("failed to get cluster(id=%s)'s security group': %v", clusterId, err)
		} else {
			if secGroupId != nil {
				targetSecGroupId = secGroupId
				break
			}
		}
		apiCallCount++
		if apiCallCount >= maxAPICallCount {
			waitErr = fmt.Errorf("failed to get security group: " +
				"The maximum number of verification requests has been exceeded " +
				"while waiting for the creation of that resource")
			return waitErr
		}
		cblogger.Infof("Wait until cluster(id=%s)'s security group is created.", clusterId)
		time.Sleep(10 * time.Second)
	}

	if targetCluster == nil || targetSecGroupId == nil {
		waitErr = fmt.Errorf("failed to get cluster security group: unknown reason")
		return waitErr
	}

	return nil
}

func (nch *NhnCloudClusterHandler) waitUntilNodeGroupIsStatus(clusterId, nodeGroupId, status string) error {
	apiCallCount := 0
	maxAPICallCount := 240

	var waitErr error
	for {
		nodeGroup, err := nhnGetRawNodeGroup(nch.ClusterClient, clusterId, irs.IID{SystemId: nodeGroupId})
		if err != nil {
			maxAPICallCount = maxAPICallCount / 2
			cblogger.Infof("failed to get node group(id=%s): %v", nodeGroupId, err)
		} else {
			if strings.EqualFold(nodeGroup.Status, status) {
				return nil
			}
		}

		apiCallCount++
		if apiCallCount >= maxAPICallCount {
			waitErr = fmt.Errorf("failed to get nodegroup: " +
				"The maximum number of verification requests has been exceeded " +
				"while waiting for availability of that resource")
			break
		}
		cblogger.Infof("Wait until nodegroup(id=%s)'s status is %s", nodeGroupId, status)
		time.Sleep(10 * time.Second)
	}

	return waitErr

}

func (nch *NhnCloudClusterHandler) getClusterSecGroupId(cluster *clusters.Cluster) (*irs.IID, error) {
	secGroupList, err := nhnGetSecGroupListByProjectId(nch.NetworkClient, cluster.ProjectID)
	if err != nil {
		err = fmt.Errorf("failed to get security groups' list by project id(%s): %v", cluster.ProjectID, err)
		return nil, err
	}

	for _, secGroup := range secGroupList {
		if strings.Contains(secGroup.Name, secgroupPostfix) {
			clusterNameWithDash := cluster.Name + "-"
			if strings.Contains(secGroup.Name, clusterNameWithDash) {
				return &irs.IID{secGroup.Name, secGroup.ID}, nil
			}
		}
	}

	return &irs.IID{}, nil
}

func (nch *NhnCloudClusterHandler) createCluster(clusterReqInfo *irs.ClusterInfo) (string, error) {
	//
	// Check if VPC is connected to an internet gateway
	//
	vpcHanlder := NhnCloudVPCHandler{
		NetworkClient: nch.NetworkClient,
	}
	vpc, err := vpcHanlder.getRawVPC(clusterReqInfo.Network.VpcIID)
	if err != nil {
		return "", fmt.Errorf("Failed to get VPC: %v", err)
	}
	hasGateway, err := nch.isVpcConnectedToGateway(vpc.ID)
	if err != nil {
		return "", fmt.Errorf("Failed to Create Cluster: %v", err)
	}
	if hasGateway == false {
		return "", fmt.Errorf("Failed to Create Cluster: VPC Should Be Connected to Internet Gateway for Providing Public Endpoint")
	}

	clusterName := clusterReqInfo.IId.NameId
	firstNodeGroupInfo := &clusterReqInfo.NodeGroupList[0]

	imageName := firstNodeGroupInfo.ImageIID.NameId

	if strings.EqualFold(imageName, "") || strings.EqualFold(imageName, "default") {
		image, err := nch.getContainerImageByNamePrefix(defaultContainerImageNamePrefix)
		if err != nil {
			err = fmt.Errorf("failed to create a cluster(%s): %v", clusterName, err)
			return "", err
		}
		firstNodeGroupInfo.ImageIID.NameId = image.ID
	} else {
		isValid, err := nch.isValidContainerImageId(imageName)
		if err != nil {
			err = fmt.Errorf("failed to create a cluster(%s): %v", clusterName, err)
			return "", err
		}
		if isValid == false {
			imageList, err := nch.getAvailableContainerImageList()
			if err != nil {
				err = fmt.Errorf("failed to create a cluster(%s): %v", clusterName, err)
				return "", err
			}

			err = fmt.Errorf("available container images: (" + strings.Join(imageList, ", ") + ")")
			return "", fmt.Errorf("failed to create a cluster(%s): %v", clusterName, err)
		}
	}

	flavorName := firstNodeGroupInfo.VMSpecName
	flavorId, err := nch.getFlavorIdByName(flavorName)
	if err != nil {
		return "", err
	}

	fixedNetwork := vpc.ID
	var fixedSubnet string
	if clusterReqInfo.Network.SubnetIIDs[0].SystemId == "" {
		if len(vpc.Subnets) > 0 {
			for _, subnet := range vpc.Subnets {
				if subnet.Name == clusterReqInfo.Network.SubnetIIDs[0].NameId {
					fixedSubnet = subnet.ID
					break
				}
			}
		}
	} else {
		fixedSubnet = clusterReqInfo.Network.SubnetIIDs[0].SystemId
	}

	keyPair := firstNodeGroupInfo.KeyPairIID.NameId
	if firstNodeGroupInfo.KeyPairIID.NameId == "" {
		keyPair = firstNodeGroupInfo.KeyPairIID.SystemId
	}

	labels, err := nch.getLabelsForCluster(clusterReqInfo, firstNodeGroupInfo)
	if err != nil {
		return "", err
	}

	nodeCount := firstNodeGroupInfo.DesiredNodeSize
	timeout := createTimeout

	uuid, err := nhnCreateCluster(nch.ClusterClient, clusterName, timeout, fixedNetwork, fixedSubnet, flavorId, keyPair, labels, nodeCount)
	if err != nil {
		err = fmt.Errorf("failed to create a cluster(%s): %v", clusterName, err)
		return "", err
	}

	return uuid, nil
}

func (nch *NhnCloudClusterHandler) deleteCluster(clusterId string) error {
	// cluster subresource Clean 현재 없음

	err := nhnDeleteCluster(nch.ClusterClient, clusterId)
	if err != nil {
		err = fmt.Errorf("failed to delete a cluster(id=%s): %v", clusterId, err)
		return err
	}

	return nil
}

func nhnCreateCluster(scCluster *nhnsdk.ServiceClient, clusterName string, timeout int, fixedNetwork, fixedSubnet, flavorId, keyPair string, labels map[string]string, nodeCount int) (string, error) {
	createOpts := clusters.CreateOpts{
		ClusterTemplateID: clusterTemplateId,
		CreateTimeout:     &timeout,
		FixedNetwork:      fixedNetwork,
		FixedSubnet:       fixedSubnet,
		FlavorID:          flavorId,
		Keypair:           keyPair,
		Labels:            labels,
		Name:              clusterName,
		NodeCount:         &nodeCount,
	}
	uuid, err := clusters.Create(scCluster, createOpts).Extract()
	if err != nil {
		return "", err
	}

	return uuid, nil
}

func nhnGetClusterList(scCluster *nhnsdk.ServiceClient) ([]clusters.Cluster, error) {
	emptyClusterList := make([]clusters.Cluster, 0)

	allPages, err := clusters.List(scCluster, nil).AllPages()
	if err != nil {
		err = fmt.Errorf("failed to get clusters' list: %v", err)
		return emptyClusterList, err
	}

	clusterList, err := clusters.ExtractClusters(allPages)
	if err != nil {
		err = fmt.Errorf("failed to extract clusters' list: %v", err)
		return emptyClusterList, err
	}

	return clusterList, nil
}

func nhnGetRawCluster(scCluster *nhnsdk.ServiceClient, clusterIID irs.IID) (*clusters.Cluster, error) {
	if clusterIID.SystemId != "" {
		cluster, err := clusters.Get(scCluster, clusterIID.SystemId).Extract()
		if err != nil {
			return nil, err
		}

		return cluster, nil
	}

	clusterList, err := nhnGetClusterList(scCluster)
	if err != nil {
		return nil, err
	}
	for _, cluster := range clusterList {
		if cluster.Name == clusterIID.NameId {
			return &cluster, nil
		}
	}

	return nil, errors.New("cluster not found")
}

func nhnDeleteCluster(scCluster *nhnsdk.ServiceClient, clusterId string) error {
	err := clusters.Delete(scCluster, clusterId).ExtractErr()
	if err != nil {
		return err
	}

	return nil
}

func nhnUpgradeCluster(scCluster *nhnsdk.ServiceClient, clusterId, nodeGroupId, newVersion string) error {
	upgradeOpts := nodegroups.UpgradeOpts{
		Version: newVersion,
	}

	_, err := nodegroups.Upgrade(scCluster, clusterId, nodeGroupId, upgradeOpts).Extract()
	if err != nil {
		return err
	}

	return nil
}

func nhnGetSecGroupListByProjectId(scNetwork *nhnsdk.ServiceClient, projectId string) ([]netsecgroups.SecGroup, error) {
	emptySecGroupList := []netsecgroups.SecGroup{}

	listOpts := netsecgroups.ListOpts{
		ProjectID: projectId,
	}

	allPages, err := netsecgroups.List(scNetwork, listOpts).AllPages()
	if err != nil {
		err = fmt.Errorf("failed to get secrutiry groups' list: %v", err)
		return emptySecGroupList, err
	}

	secGroupList, err := netsecgroups.ExtractGroups(allPages)
	if err != nil {
		err = fmt.Errorf("failed to extract security groups' list: %v", err)
		return emptySecGroupList, err
	}

	return secGroupList, nil
}

func convertNodeGroupStatusToNodeGroupInfoStatus(nodeGroup *nodegroups.NodeGroup) irs.NodeGroupStatus {
	status := irs.NodeGroupInactive
	if strings.EqualFold(nodeGroup.Status, clusterStatusCreateInProgress) {
		status = irs.NodeGroupCreating
	} else if strings.EqualFold(nodeGroup.Status, clusterStatusUpdateInProgress) {
		status = irs.NodeGroupUpdating // removing is a kind of updating?
	} else if strings.EqualFold(nodeGroup.Status, clusterStatusUpdateFailed) {
		status = irs.NodeGroupInactive
	} else if strings.EqualFold(nodeGroup.Status, clusterStatusDeleteInProgress) {
		status = irs.NodeGroupDeleting
	} else if strings.EqualFold(nodeGroup.Status, clusterStatusCreateComplete) {
		status = irs.NodeGroupActive
	} else if strings.EqualFold(nodeGroup.Status, clusterStatusUpdateComplete) {
		status = irs.NodeGroupActive
	}

	return status
}

func convertClusterStatusToClusterInfoStatus(cluster *clusters.Cluster) irs.ClusterStatus {
	status := irs.ClusterInactive
	if strings.EqualFold(cluster.Status, clusterStatusCreateInProgress) {
		status = irs.ClusterCreating
	} else if strings.EqualFold(cluster.Status, clusterStatusUpdateInProgress) {
		status = irs.ClusterUpdating
	} else if strings.EqualFold(cluster.Status, clusterStatusUpdateFailed) {
		status = irs.ClusterInactive
	} else if strings.EqualFold(cluster.Status, clusterStatusDeleteInProgress) {
		status = irs.ClusterDeleting
	} else if strings.EqualFold(cluster.Status, clusterStatusCreateComplete) {
		status = irs.ClusterActive
	} else if strings.EqualFold(cluster.Status, clusterStatusUpdateComplete) {
		status = irs.ClusterActive
	}

	return status
}

func (nch *NhnCloudClusterHandler) getClusterAccessInfo(cluster *clusters.Cluster) (*irs.AccessInfo, error) {
	endpoint := "Endpoint is not ready yet!"
	if !strings.EqualFold(cluster.APIAddress, "") {
		endpoint = cluster.APIAddress
	}

	kubeconfig := "Kubeconfig is not ready yet!"
	clusterId := cluster.UUID
	config, err := nhnGetConfig(nch.ClusterClient, clusterId)
	if err != nil {
		cblogger.Info("Kubeconfig is not ready yet!")
	} else {
		kubeconfig = strings.TrimSpace(config)
	}

	accessInfo := &irs.AccessInfo{
		Endpoint:   endpoint,
		Kubeconfig: kubeconfig,
	}

	return accessInfo, nil
}

func (nch *NhnCloudClusterHandler) createNodeGroup(clusterId string, nodeGroupReqInfo *irs.NodeGroupInfo) (string, error) {
	nodeGroupName := nodeGroupReqInfo.IId.NameId

	cluster, err := nhnGetRawCluster(nch.ClusterClient, irs.IID{SystemId: clusterId})
	if err != nil {
		err = fmt.Errorf("failed to a cluster(id=%s): %v",
			nodeGroupName, clusterId, err)
		return "", err
	}

	addVpcIdList, _ := cluster.Labels[clusterLabelsAdditionalNetworkIdList]
	addSubnetIdList, _ := cluster.Labels[clusterLabelsAdditionalSubnetIdList]

	labels, err := nch.getLabelsForNodeGroup(nodeGroupReqInfo, addVpcIdList, addSubnetIdList)
	if err != nil {
		err = fmt.Errorf("failed to create a node group(%s) of cluster(id=%s): %v",
			nodeGroupName, clusterId, err)
		return "", err
	}

	nodeCount := nodeGroupReqInfo.DesiredNodeSize
	minNodeCount := nodeGroupReqInfo.MinNodeSize
	maxNodeCount := nodeGroupReqInfo.MaxNodeSize

	imageName := nodeGroupReqInfo.ImageIID.NameId
	imageId := ""

	if strings.EqualFold(imageName, "") || strings.EqualFold(imageName, "default") {
		image, err := nch.getContainerImageByNamePrefix(defaultContainerImageNamePrefix)
		if err != nil {
			err = fmt.Errorf("failed to create a node group(%s) of cluster(id=%s): %v",
				nodeGroupName, clusterId, err)
			return "", err
		}
		imageId = image.ID
	} else {
		isValid, err := nch.isValidContainerImageId(imageName)
		if err != nil {
			err = fmt.Errorf("failed to create a node group(%s) of cluster(id=%s): %v",
				nodeGroupName, clusterId, err)
			return "", err
		}
		if isValid == false {
			imageList, err := nch.getAvailableContainerImageList()
			if err != nil {
				err = fmt.Errorf("failed to create a node group(%s) of cluster(id=%s): %v",
					nodeGroupName, clusterId, err)
				return "", err
			}

			err = fmt.Errorf("available container images: (" + strings.Join(imageList, ", ") + ")")
			return "", fmt.Errorf("failed to create a node group(%s) of cluster(id=%s): %v",
				nodeGroupName, clusterId, err)
		}
	}

	flavorId, err := nch.getFlavorIdByName(nodeGroupReqInfo.VMSpecName)
	if err != nil {
		err = fmt.Errorf("failed to create a node group(%s) of cluster(id=%s): %v",
			nodeGroupName, clusterId, err)
		return "", err
	}

	nodeGroup, err := nhnCreateNodeGroup(nch.ClusterClient, clusterId, nodeGroupName,
		labels, nodeCount, minNodeCount, maxNodeCount, imageId, flavorId)
	if err != nil {
		err = fmt.Errorf("failed to create a node group(%s) of cluster(id=%s): %v",
			nodeGroupName, clusterId, err)
		return "", err
	}

	return nodeGroup.UUID, nil
}

func (nch *NhnCloudClusterHandler) deleteNodeGroup(clusterId, nodeGroupId string) error {
	// nodeGroup subresource Clean 현재 없음

	err := nhnDeleteNodeGroup(nch.ClusterClient, clusterId, nodeGroupId)
	if err != nil {
		err = fmt.Errorf("failed to delete a node group(id=%s) of cluster(id=%s): %v",
			nodeGroupId, clusterId, err)
		return err
	}

	return nil
}

func (nch *NhnCloudClusterHandler) getFlavorIdByName(flavorName string) (string, error) {
	flavorList, err := nhnGetFlavorList(nch.VMClient)
	if err != nil {
		return "", err
	}

	for _, flavor := range flavorList {
		if strings.EqualFold(flavor.Name, flavorName) {
			return flavor.ID, nil
		}
	}

	return "", fmt.Errorf("failed to find a flavor by name(%s)", flavorName)
}

func (nch *NhnCloudClusterHandler) getFlavorNameById(flavorId string) (string, error) {
	flavorList, err := nhnGetFlavorList(nch.VMClient)
	if err != nil {
		return "", err
	}

	for _, flavor := range flavorList {
		if strings.EqualFold(flavor.ID, flavorId) {
			return flavor.Name, nil
		}
	}

	return "", fmt.Errorf("failed to find a flavor by id(%s)", flavorId)
}

func (nch *NhnCloudClusterHandler) isValidContainerImageId(imageId string) (bool, error) {
	imageList, err := nhnGetContainerImageList(nch.ImageClient)
	if err != nil {
		return false, err
	}

	for _, image := range imageList {
		if strings.EqualFold(image.ID, imageId) {
			return true, nil
		}
	}

	return false, nil
}

func (nch *NhnCloudClusterHandler) getContainerImageByNamePrefix(imageNamePrefix string) (images.Image, error) {
	imageList, err := nhnGetContainerImageList(nch.ImageClient)
	if err != nil {
		return images.Image{}, err
	}

	for _, image := range imageList {
		if strings.Contains(image.Name, imageNamePrefix) {
			return image, nil
		}
	}

	return images.Image{}, fmt.Errorf("no container image with name prefix(%s)", imageNamePrefix)
}

func (nch *NhnCloudClusterHandler) getAvailableContainerImageList() ([]string, error) {
	var containerImageList []string
	imageList, err := nhnGetContainerImageList(nch.ImageClient)
	if err != nil {
		return []string{}, err
	}

	for _, image := range imageList {
		nameAndId := fmt.Sprintf("%s[ID=%s]", image.Name, image.ID)
		containerImageList = append(containerImageList, nameAndId)
	}

	return containerImageList, nil
}

func (nch *NhnCloudClusterHandler) getServerListByAddresses(imageId, flavorId string, addresses []string) ([]servers.Server, error) {
	emptyServerList := make([]servers.Server, 0)

	if len(addresses) == 0 {
		return emptyServerList, nil
	}

	targetServerList, err := nhnGetServerListByIds(nch.VMClient, imageId, flavorId)
	if err != nil {
		err = fmt.Errorf("failed to get server list with image(id=%s) and flavor(id=%s)", imageId, flavorId)
		return emptyServerList, err
	}

	var found bool
	var serverList []servers.Server
	for _, address := range addresses {
		for _, server := range targetServerList {
			found = false
			for _, valueAddr := range server.Addresses {
				for _, mapAddr := range valueAddr.([]interface{}) {
					if addr, ok := mapAddr.(map[string]interface{})["addr"]; ok {
						if strings.EqualFold(address, addr.(string)) {
							serverList = append(serverList, server)
							found = true
							break
						}
					}
				}
			}
			if found {
				break
			}
		}
	}

	if len(serverList) == 0 {
		err = fmt.Errorf("failed to find a server by addresses(%v)", addresses)
		return emptyServerList, err
	}

	return serverList, nil
}

func (nch *NhnCloudClusterHandler) getVpcName(vpcId string) (string, error) {
	vpc, err := nhnGetVpc(nch.NetworkClient, vpcId)
	if err != nil {
		err = fmt.Errorf("failed to get vpc name with id(%s): %v", vpcId, err)
		return "", err
	}

	return vpc.Name, nil
}

func (nch *NhnCloudClusterHandler) getVpcsubnetName(vpcsubnetId string) (string, error) {
	vpcsubnet, err := nhnGetVpcsubnet(nch.NetworkClient, vpcsubnetId)
	if err != nil {
		return "", err
	}

	return vpcsubnet.Name, nil
}

// # Check whether the Routing Table (of the VPC) is connected to an Internet Gateway
func (nch *NhnCloudClusterHandler) isVpcConnectedToGateway(vpcId string) (bool, error) {
	vpc, err := nhnGetVpc(nch.NetworkClient, vpcId)
	if err != nil {
		return false, err
	}

	hasInternetGateway := false
	if len(vpc.RoutingTables) > 0 {
		if !strings.EqualFold(vpc.RoutingTables[0].GatewayID, "") {
			hasInternetGateway = true
		}
	}

	return hasInternetGateway, nil
}

func (nch *NhnCloudClusterHandler) getSupportedK8sVersions() ([]string, error) {
	supports, err := nhnGetSupports(nch.ClusterClient)
	if err != nil {
		return make([]string, 0), err
	}

	supportedK8sVersions := make([]string, 0)
	for version, supported := range supports.SupportedK8s {
		if supported {
			if strings.EqualFold(version, "") == false {
				supportedK8sVersions = append(supportedK8sVersions, version)
			}
		}
	}

	return supportedK8sVersions, nil
}

func nhnCreateNodeGroup(scCluster *nhnsdk.ServiceClient, clusterId, nodeGroupName string, labels map[string]string, nodeCount, minNodeCount, maxNodeCount int, imageId, flavorId string) (*nodegroups.NodeGroup, error) {
	createOpts := nodegroups.CreateOpts{
		Name:         nodeGroupName,
		Labels:       labels,
		NodeCount:    &nodeCount,
		MinNodeCount: minNodeCount,
		MaxNodeCount: &maxNodeCount,
		ImageID:      imageId,
		FlavorID:     flavorId,
	}
	nodeGroup, err := nodegroups.Create(scCluster, clusterId, createOpts).Extract()
	if err != nil {
		err = fmt.Errorf("failed to create nodgroup(%s) of cluster(id=%s): %v", nodeGroupName, clusterId, err)
		return nil, err
	}

	return nodeGroup, nil
}

func nhnGetNodeGroupList(scCluster *nhnsdk.ServiceClient, clusterId string) ([]nodegroups.NodeGroup, error) {
	emptyNodeGroupList := make([]nodegroups.NodeGroup, 0)
	nodeGroupListOpts := nodegroups.ListOpts{}
	allPages, err := nodegroups.List(scCluster, clusterId, nodeGroupListOpts).AllPages()
	if err != nil {
		err = fmt.Errorf("failed to get the cluster(id=%s)'s nodegroups: %v", clusterId, err)
		return emptyNodeGroupList, err
	}

	nodeGroupList, err := nodegroups.ExtractNodeGroups(allPages)
	if err != nil {
		err = fmt.Errorf("failed to extract the cluster(id=%s)'s nodegroups: %v", clusterId, err)
		return emptyNodeGroupList, err
	}

	return nodeGroupList, nil
}

func nhnGetRawNodeGroup(scCluster *nhnsdk.ServiceClient, clusterId string, nodeGroupIID irs.IID) (*nodegroups.NodeGroup, error) {
	if nodeGroupIID.SystemId != "" {
		nodeGroup, err := nodegroups.Get(scCluster, clusterId, nodeGroupIID.SystemId).Extract()
		if err != nil {
			err = fmt.Errorf("failed to get the cluster(id=%s)'s nodegroup(id=%s): %v", clusterId, nodeGroupIID.SystemId, err)
			return nil, err
		}

		return nodeGroup, nil
	}

	nodeGroupList, err := nhnGetNodeGroupList(scCluster, clusterId)
	if err != nil {
		err = fmt.Errorf("failed to get the cluster(id=%s)'s nodegroups: %v", clusterId, err)
	}

	for _, nodeGroup := range nodeGroupList {
		if nodeGroup.Name == nodeGroupIID.NameId {
			return &nodeGroup, nil
		}
	}

	return nil, errors.New("node group not found")
}

func nhnDeleteNodeGroup(scCluster *nhnsdk.ServiceClient, clusterId, nodeGroupId string) error {
	err := nodegroups.Delete(scCluster, clusterId, nodeGroupId).ExtractErr()
	if err != nil {
		return err
	}

	return nil
}

func nhnGetNodeGroupAutoscale(scCluster *nhnsdk.ServiceClient, clusterId, nodeGroupId string) (*nodegroups.Autoscale, error) {
	autoscale, err := nodegroups.GetAutoscale(scCluster, clusterId, nodeGroupId).Extract()
	if err != nil {
		err = fmt.Errorf("failed to get nodegroup(id=%s)'s autoscale of cluster(id=%s): %v", nodeGroupId, clusterId, err)
		return nil, err
	}

	return autoscale, nil
}

func nhnSetNodeGroupAutoscale(scCluster *nhnsdk.ServiceClient, clusterId, nodeGroupId string, enable bool, minNodeCount, maxNodeCount int) (string, error) {
	setAutoscaleOpts := nodegroups.SetAutoscaleOpts{
		CaEnable:       &enable,
		CaMaxNodeCount: maxNodeCount,
		CaMinNodeCount: minNodeCount,
	}
	uuid, err := nodegroups.SetAutoscale(scCluster, clusterId, nodeGroupId, setAutoscaleOpts).Extract()
	if err != nil {
		err = fmt.Errorf("failed to set nodegroup(id=%s)'s autoscale of cluster(id=%s): %v", nodeGroupId, clusterId, err)
		return "", err
	}
	if nodeGroupId != uuid {
		err = fmt.Errorf("failed to set nodegroup(id=%s)'s autoscale of cluster(id=%s): %v", nodeGroupId, clusterId, err)
		return "", err
	}

	return uuid, nil
}

func nhnSetNodeGroupAutoscaleEnable(scCluster *nhnsdk.ServiceClient, clusterId, nodeGroupId string, enable bool) (string, error) {
	setAutoscaleOpts := nodegroups.SetAutoscaleOpts{
		CaEnable: &enable,
	}
	uuid, err := nodegroups.SetAutoscale(scCluster, clusterId, nodeGroupId, setAutoscaleOpts).Extract()
	if err != nil {
		err = fmt.Errorf("failed to set nodegroup(id=%s)'s autoscale of cluster(id=%s): %v", nodeGroupId, clusterId, err)
		return "", err
	}
	if nodeGroupId != uuid {
		err = fmt.Errorf("failed to set nodegroup(id=%s)'s autoscale of cluster(id=%s): %v", nodeGroupId, clusterId, err)
		return "", err
	}

	return uuid, nil
}

func nhnGetFlavorList(scVM *nhnsdk.ServiceClient) ([]flavors.Flavor, error) {
	allPages, err := flavors.ListDetail(scVM, nil).AllPages()
	if err != nil {
		return make([]flavors.Flavor, 0), fmt.Errorf("failed to get flavors' list: %v", err)
	}

	flavorList, err := flavors.ExtractFlavors(allPages)
	if err != nil {
		return make([]flavors.Flavor, 0), fmt.Errorf("failed to extract flavors: %v", err)
	}

	return flavorList, nil
}

func nhnGetImageList(scImage *nhnsdk.ServiceClient) ([]images.Image, error) {
	emptyImageList := make([]images.Image, 0)

	listOpts := images.ListOpts{}
	allPages, err := images.List(scImage, listOpts).AllPages()
	if err != nil {
		return emptyImageList, fmt.Errorf("failed to get images' list: %v", err)
	}

	imageList, err := images.ExtractImages(allPages)
	if err != nil {
		return emptyImageList, fmt.Errorf("failed to extract images: %v", err)
	}

	return imageList, nil
}

func nhnGetContainerImageList(scImage *nhnsdk.ServiceClient) ([]images.Image, error) {
	emptyImageList := make([]images.Image, 0)

	listOpts := images.ListOpts{
		Visibility:      images.ImageVisibilityPublic,
		NhncloudProduct: images.ImageNhncloudProductContainer,
	}
	allPages, err := images.List(scImage, listOpts).AllPages()
	if err != nil {
		return emptyImageList, fmt.Errorf("failed to get container images' list: %v", err)
	}

	imageList, err := images.ExtractImages(allPages)
	if err != nil {
		return emptyImageList, fmt.Errorf("failed to extract images: %v", err)
	}

	return imageList, nil
}

func nhnGetImageById(scImage *nhnsdk.ServiceClient, imageId string) (*images.Image, error) {
	image, err := images.Get(scImage, imageId).Extract()
	if err != nil {
		return nil, fmt.Errorf("failed to get a image by id(%s)", imageId)
	}

	return image, nil
}

func (nch *NhnCloudClusterHandler) getImageNameById(imageId string) (string, error) {
	image, err := nhnGetImageById(nch.ImageClient, imageId)
	if err != nil {
		return "", err
	}

	return image.Name, nil
}

func nhnGetServerListByIds(scVM *nhnsdk.ServiceClient, imageId, flavorId string) ([]servers.Server, error) {
	listOpts := servers.ListOpts{Image: imageId, Flavor: flavorId}
	allPages, err := servers.List(scVM, listOpts).AllPages()
	if err != nil {
		return make([]servers.Server, 0), fmt.Errorf("failed to get servers' list: %v", err)
	}

	serverList, err := servers.ExtractServers(allPages)
	if err != nil {
		return make([]servers.Server, 0), fmt.Errorf("failed to extract servers: %v", err)
	}

	return serverList, nil
}

func nhnGetConfig(scCluster *nhnsdk.ServiceClient, clusterId string) (string, error) {
	config, err := clusters.GetConfig(scCluster, clusterId).Extract()
	if err != nil {
		err = fmt.Errorf("failed to get config by cluster(id=%s)", clusterId)
		return "", err
	}

	return config, nil
}

func nhnGetSupports(scCluster *nhnsdk.ServiceClient) (*clusters.Supports, error) {
	supports, err := clusters.GetSupports(scCluster).Extract()
	if err != nil {
		err = fmt.Errorf("failed to get supported kubernetes version and event type")
		return nil, err
	}

	return supports, nil
}

func nhnGetVpcList(scNetwork *nhnsdk.ServiceClient, external bool) ([]vpcs.VPC, error) {
	listOpts := vpcs.ListOpts{
		RouterExternal: external,
	}
	allPages, err := vpcs.List(scNetwork, listOpts).AllPages()
	if err != nil {
		err = fmt.Errorf("failed to get vpcs' list: %v", err)
		return nil, err
	}

	vpcList, err := vpcs.ExtractVPCs(allPages)
	if err != nil {
		err = fmt.Errorf("failed to extract vpcs' list: %v", err)
		return nil, err
	}

	return vpcList, nil
}

func nhnGetNetworkList(scNetwork *nhnsdk.ServiceClient, external bool) ([]networks.Network, error) {
	listOpts := networks.ListOpts{
		RouterExternal: external,
		//		Shared:         true,
	}
	allPages, err := networks.List(scNetwork, listOpts).AllPages()
	if err != nil {
		err = fmt.Errorf("failed to get networks' list: %v", err)
		return nil, err
	}

	networkList, err := networks.ExtractNetworks(allPages)
	if err != nil {
		err = fmt.Errorf("failed to extract networks' list: %v", err)
		return nil, err
	}

	return networkList, nil
}

func nhnGetSubnetListInNetwork(scNetwork *nhnsdk.ServiceClient, networkId string) ([]subnets.Subnet, error) {
	listOpts := subnets.ListOpts{
		NetworkID: networkId,
	}
	allPages, err := subnets.List(scNetwork, listOpts).AllPages()
	if err != nil {
		err = fmt.Errorf("failed to get subnets' list: %v", err)
		return nil, err
	}

	subnetList, err := subnets.ExtractSubnets(allPages)
	if err != nil {
		err = fmt.Errorf("failed to extract subnets' list: %v", err)
		return nil, err
	}

	return subnetList, nil
}

func nhnGetSubnetById(scNetwork *nhnsdk.ServiceClient, subnetId string) (*subnets.Subnet, error) {
	listOpts := subnets.ListOpts{
		ID: subnetId,
		//RouterExternal: true,
	}
	allPages, err := subnets.List(scNetwork, listOpts).AllPages()
	if err != nil {
		err = fmt.Errorf("failed to get subnets' list: %v", err)
		return nil, err
	}

	subnetList, err := subnets.ExtractSubnets(allPages)
	if err != nil {
		err = fmt.Errorf("failed to extract subnets' list: %v", err)
		return nil, err
	}

	if len(subnetList) == 0 {
		err = fmt.Errorf("failed to find a subnet with id(%s)", subnetId)
		return nil, err
	} else if len(subnetList) > 1 {
		err = fmt.Errorf("failed to get only one subnet with id(%s)", subnetId)
		return nil, err
	}

	return &subnetList[0], nil
}

func nhnGetVpc(scNetwork *nhnsdk.ServiceClient, vpcId string) (*vpcs.VPC, error) {
	vpc, err := vpcs.Get(scNetwork, vpcId).Extract()
	if err != nil {
		err = fmt.Errorf("failed to get vpc: %v", err)
		return nil, err
	}

	return vpc, nil
}

func nhnGetVpcById(scNetwork *nhnsdk.ServiceClient, vpcId string) (*vpcs.VPC, error) {
	listOpts := vpcs.ListOpts{
		ID: vpcId,
		//RouterExternal: true,
	}
	allPages, err := vpcs.List(scNetwork, listOpts).AllPages()
	if err != nil {
		err = fmt.Errorf("failed to get vpcs' list: %v", err)
		return nil, err
	}

	vpcList, err := vpcs.ExtractVPCs(allPages)
	if err != nil {
		err = fmt.Errorf("failed to extract vpcs' list: %v", err)
		return nil, err
	}

	if len(vpcList) == 0 {
		err = fmt.Errorf("failed to find a vpc with id(%s)", vpcId)
		return nil, err
	} else if len(vpcList) > 1 {
		err = fmt.Errorf("failed to get only one vpc with id(%s)", vpcId)
		return nil, err
	}

	return &vpcList[0], nil
}

func nhnGetVpcsubnetListInVpc(scNetwork *nhnsdk.ServiceClient, vpcId string) ([]vpcsubnets.Vpcsubnet, error) {
	listOpts := vpcsubnets.ListOpts{
		VPCID: vpcId,
	}
	allPages, err := vpcsubnets.List(scNetwork, listOpts).AllPages()
	if err != nil {
		err = fmt.Errorf("failed to get VPCs' list: %v", err)
		return nil, err
	}

	vpcsubnetList, err := vpcsubnets.ExtractVpcsubnets(allPages)
	if err != nil {
		err = fmt.Errorf("failed to extract VPCs' list: %v", err)
		return nil, err
	}

	return vpcsubnetList, nil
}

func nhnGetVpcsubnet(scNetwork *nhnsdk.ServiceClient, vpcsubnetId string) (*vpcsubnets.Vpcsubnet, error) {
	vpcsubnet, err := vpcsubnets.Get(scNetwork, vpcsubnetId).Extract()
	if err != nil {
		err = fmt.Errorf("failed to get vpcsubnet with id(%s): %v", vpcsubnetId, err)
		return nil, err
	}

	return vpcsubnet, nil
}

func getClusterVersion(cluster *clusters.Cluster) (string, error) {
	if version, ok := cluster.Labels[clusterLabelsKubeTag]; !ok {
		err := fmt.Errorf("failed to get version: labels.kube_tag")
		return "", err
	} else {
		return version, nil
	}
}

func getNodeGroupVersion(nodeGroup *nodegroups.NodeGroup) (string, error) {
	if version, ok := nodeGroup.Labels[clusterLabelsKubeTag]; !ok {
		err := fmt.Errorf("failed to get version: labels.kube_tag")
		return "", err
	} else {
		return version, nil
	}
}

func validateNodeGroupInfoList(nodeGroupInfoList []irs.NodeGroupInfo) error {
	if len(nodeGroupInfoList) == 0 {
		return fmt.Errorf("Node Group must be specified")
	}

	// NHN Cloud의 KeyPair는 클러스터 의존, NodeGroup에 의존하지 않음
	var firstKeypairId *irs.IID
	for i, nodeGroupInfo := range nodeGroupInfoList {
		if nodeGroupInfo.IId.NameId == "" {
			return fmt.Errorf("Node Group's name is required")
		}
		if nodeGroupInfo.VMSpecName == "" {
			return fmt.Errorf("Node Group's vm spec name is required")
		}
		if i == 0 {
			if nodeGroupInfo.KeyPairIID.NameId == "" && nodeGroupInfo.KeyPairIID.SystemId == "" {
				return fmt.Errorf("Node Group's keypair is required")
			}
			firstKeypairId = &nodeGroupInfo.KeyPairIID
		} else {
			// NameId, SystemId 둘다 값이 있음
			if nodeGroupInfo.KeyPairIID.NameId != "" && nodeGroupInfo.KeyPairIID.SystemId != "" {
				if nodeGroupInfo.KeyPairIID.NameId != firstKeypairId.NameId || nodeGroupInfo.KeyPairIID.SystemId != firstKeypairId.SystemId {
					return fmt.Errorf("Node Group's keypair must all be the same")
				}
			} else if nodeGroupInfo.KeyPairIID.NameId != "" {
				if nodeGroupInfo.KeyPairIID.NameId != firstKeypairId.NameId {
					return fmt.Errorf("Node Group's keypair must all be the same")
				}
			} else if nodeGroupInfo.KeyPairIID.SystemId != "" {
				if nodeGroupInfo.KeyPairIID.SystemId != firstKeypairId.SystemId {
					return fmt.Errorf("Node Group's keypair must all be the same")
				}
			} else {
				return fmt.Errorf("Node Group's keypair must all be the same")
			}
		}

		// OnAutoScaling + MinNodeSize
		// MaxNodeSize
		// DesiredNodeSize
		if nodeGroupInfo.OnAutoScaling && nodeGroupInfo.MinNodeSize < 1 {
			return fmt.Errorf("MinNodeSize must be greater than 0 when OnAutoScaling is enabled.")
		}
		if nodeGroupInfo.MinNodeSize > 0 && !nodeGroupInfo.OnAutoScaling {
			return fmt.Errorf("If MinNodeSize is specified, OnAutoScaling must be enabled.")
		}
		if nodeGroupInfo.MinNodeSize > 0 && (nodeGroupInfo.MinNodeSize > nodeGroupInfo.MaxNodeSize) {
			return fmt.Errorf("MaxNodeSize must be greater than MinNodeSize.")
		}
		if nodeGroupInfo.MinNodeSize > 0 && (nodeGroupInfo.DesiredNodeSize < nodeGroupInfo.MinNodeSize) {
			return fmt.Errorf("DesiredNodeSize must be greater than or equal to MinNodeSize.")
		}
	}

	return nil
}

func validateAtCreateCluster(clusterInfo irs.ClusterInfo, supportedK8sVersions []string) error {
	// Check clusterInfo.IId.NameId
	if clusterInfo.IId.NameId == "" {
		return fmt.Errorf("Cluster name is required")
	}

	// Check clusterInfo.Network
	if len(clusterInfo.Network.SubnetIIDs) < 1 {
		return fmt.Errorf("At least one Subnet must be specified")
	}
	if len(clusterInfo.Network.SecurityGroupIIDs) < 1 {
		return fmt.Errorf("At least one Subnet must be specified")
	}

	// Check clusterInfo.Version
	var supported = false
	for _, version := range supportedK8sVersions {
		if strings.EqualFold(clusterInfo.Version, version) {
			supported = true
			break
		}
	}
	if supported == false {
		return fmt.Errorf("Unsupported K8s version. (Available version: " + strings.Join(supportedK8sVersions[:], ", ") + ")")
	}

	// Check clusterInfo.NodeGroupList
	err := validateNodeGroupInfoList(clusterInfo.NodeGroupList)
	if err != nil {
		return err
	}

	return nil
}

func validateAtAddNodeGroup(clusterIID irs.IID, nodeGroupInfo irs.NodeGroupInfo) error {
	//
	// Check nodeGroupInfo
	//
	err := validateNodeGroupInfoList([]irs.NodeGroupInfo{nodeGroupInfo})
	if err != nil {
		return err
	}

	return nil
}

func validateAtChangeNodeGroupScaling(minNodeSize int, maxNodeSize int) error {
	if minNodeSize < 1 {
		return fmt.Errorf("MaxNodeSize cannot be smaller than 1")
	}
	if maxNodeSize < 1 {
		return fmt.Errorf("MaxNodeSize cannot be smaller than 1")
	}

	return nil
}

func (nch *NhnCloudClusterHandler) ListIID() ([]*irs.IID, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NHN Cloud Driver: called ListCluster()")
	hiscallInfo := getCallLogScheme(nch.RegionInfo.Region, call.CLUSTER, "ListCluster()", "ListIID()") // HisCall logging

	start := call.Start()

	var iidList []*irs.IID

	var listErr error
	defer func() {
		if listErr != nil {
			cblogger.Error(listErr)
			LoggingError(hiscallInfo, listErr)
		}
	}()

	clusterList, err := nhnGetClusterList(nch.ClusterClient)
	if err != nil {
		listErr = fmt.Errorf("Failed to List Cluster: %v", err)
		return nil, listErr
	}

	for _, cluster := range clusterList {
		var iid irs.IID
		iid.SystemId = cluster.UUID
		iid.NameId = cluster.Name

		iidList = append(iidList, &iid)
	}

	LoggingInfo(hiscallInfo, start)

	return iidList, nil
}
