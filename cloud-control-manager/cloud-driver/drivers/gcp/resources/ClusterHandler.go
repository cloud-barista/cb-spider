package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	o2 "golang.org/x/oauth2"
	goo "golang.org/x/oauth2/google"
	compute "google.golang.org/api/compute/v1"
	container "google.golang.org/api/container/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	GCP_PMKS_SECURITYGROUP_TAG = "cb-spider-pmks-securitygroup-"
	GCP_PMKS_INSTANCEGROUP_KEY = "InstanceGroup_"
	GCP_PMKS_KEYPAIR_KEY       = "keypair"

	GCP_CONTAINER_OPERATION_TYPE_UNSPECIFIED         = -1 //"Not set."
	GCP_CONTAINER_OPERATION_CREATE_CLUSTER           = 1  //"Cluster create."
	GCP_CONTAINER_OPERATION_DELETE_CLUSTER           = 2  //"Cluster delete."
	GCP_CONTAINER_OPERATION_UPGRADE_MASTER           = 3  //"A master upgrade."
	GCP_CONTAINER_OPERATION_REPAIR_CLUSTER           = 4  //"Cluster repair."
	GCP_CONTAINER_OPERATION_UPDATE_CLUSTER           = 5  //"Cluster update."
	GCP_CONTAINER_OPERATION_CREATE_NODE_POOL         = 11 //"Node pool create."
	GCP_CONTAINER_OPERATION_DELETE_NODE_POOL         = 12 //"Node pool delete."
	GCP_CONTAINER_OPERATION_SET_NODE_POOL_MANAGEMENT = 13 //"Set node pool management."
	GCP_CONTAINER_OPERATION_SET_NODE_POOL_SIZE       = 14 //"Set node pool size."
	GCP_CONTAINER_OPERATION_UPGRADE_NODES            = 21 //"A node upgrade."
	GCP_CONTAINER_OPERATION_AUTO_REPAIR_NODES        = 22 //"Automatic node pool repair."
	GCP_CONTAINER_OPERATION_AUTO_UPGRADE_NODES       = 23 //"Automatic node upgrade."
	GCP_CONTAINER_OPERATION_SET_LABELS               = 31 //"Set labels."
	GCP_CONTAINER_OPERATION_SET_MASTER_AUTH          = 32 //"Set/generate master auth materials"
	GCP_CONTAINER_OPERATION_SET_NETWORK_POLICY       = 33 //"Updates network policy for a cluster."
	GCP_CONTAINER_OPERATION_SET_MAINTENANCE_POLICY   = 34 //"Set the maintenance policy."

	GCP_SET_AUTOSCALING_ENABLE   = "SET_AUTOSCALING_ENABLE"
	GCP_SET_AUTOSCALING_NODESIZE = "SET_AUTOSCALING_NODESIZE"
)

// getServerAddress returns the Spider server address from SERVER_ADDRESS env variable
func getServerAddress() string {
	hostEnv := os.Getenv("SERVER_ADDRESS")
	if hostEnv == "" {
		return "localhost:1024"
	}

	// "1.2.3.4" or "localhost"
	if !strings.Contains(hostEnv, ":") {
		return hostEnv + ":1024"
	}

	// ":31024" => "localhost:31024"
	if strings.HasPrefix(hostEnv, ":") {
		return "localhost" + hostEnv
	}

	// "1.2.3.4:31024" or "localhost:31024"
	return hostEnv
}

type GCPClusterHandler struct {
	Region          idrv.RegionInfo
	Ctx             context.Context
	Client          *compute.Service
	ContainerClient *container.Service
	Credential      idrv.CredentialInfo
}

/*
The NodePool is created with the name default-pool.
The machine type is created as e2-medium.
The boot disk is also created as 100.
The SG (firewall rule) is not added.

Check how waiting for a failure is handled.
*/

var (
	ErrNodeGroupEmpty           = errors.New("node group is empty. it must be greater than 0")
	ErrNodeGroupNameEmpty       = errors.New("node group name is empty")
	ErrNodeGroupVMSpecEmpty     = errors.New("node group vm spec is empty")
	ErrInvalidNodeGroupDiskSize = errors.New("invalid NodeGroup RootDiskSize")
	ErrNodeGroupKeypairEmpty    = errors.New("node group keypair is empty")
	ErrInvalidMinNodeSize       = errors.New("MinNodeSize must be greater than zero")
	ErrInvalidMaxNodeSize       = errors.New("MaxNodeSize must be greater than or equal to MinNodeSize")
	ErrInvalidDesiredNodeSize   = errors.New("DesiredNodeSize must be greater than or equal to MinNodeSize, and less than or equal to MaxNodeSize")
)

// fetchValidGKEImageTypes retrieves the list of valid node image types for the current
// project/zone via the GKE GetServerConfig API.
func (ClusterHandler *GCPClusterHandler) fetchValidGKEImageTypes(ctx context.Context) ([]string, error) {
	parent := getParentAtContainer(ClusterHandler.Credential.ProjectID, ClusterHandler.Region.Zone)
	serverConfig, err := ClusterHandler.ContainerClient.Projects.Locations.GetServerConfig(parent).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch GKE server config for valid image types (location: %s): %w", parent, err)
	}
	if len(serverConfig.ValidImageTypes) == 0 {
		return nil, fmt.Errorf("GKE server config returned empty valid image types for %s", parent)
	}
	return serverConfig.ValidImageTypes, nil
}

// getAvailableGKEImageTypesMessage returns a human-readable list of valid GKE image types.
func getAvailableGKEImageTypesMessage(validTypes []string) string {
	msg := "Available GKE Image Types:\n- " + strings.Join(validTypes, "\n- ")
	msg += "\n\nNotes:"
	msg += "\n- COS_CONTAINERD is the GKE recommended default."
	msg += "\n- GCP GKE does not support custom image IDs. Only the above predefined image types are accepted."
	msg += "\n- Refer to GKE documentation for OS support policy."
	return msg
}

// resolveGKEImageType resolves a non-empty, non-default image identifier to a GKE image type string.
// Precondition: imageNameId is non-empty and not "default" (caller handles those cases).
//   - valid GKE image type → normalised upper-case string
//   - anything else        → error with the list of valid types
func resolveGKEImageType(imageNameId string, validTypes []string) (string, error) {
	upper := strings.ToUpper(imageNameId)
	for _, t := range validTypes {
		if upper == strings.ToUpper(t) { // GKE API returns uppercase, but normalise defensively
			cblogger.Infof("GKE Image Type specified: %s", upper)
			return upper, nil
		}
	}

	// Not a valid GKE image type (e.g. an image ID or URL) — return error with guidance.
	return "", fmt.Errorf(
		"invalid GKE image identifier '%s': GCP GKE does not support image IDs; "+
			"please use a predefined image type\n\n%s",
		imageNameId, getAvailableGKEImageTypesMessage(validTypes))
}

func validateNodeGroup(ngl []irs.NodeGroupInfo) error {
	if len(ngl) <= 0 {
		return ErrNodeGroupEmpty
	}

	for _, nodeGroup := range ngl {
		if nodeGroup.IId.NameId == "" {
			return ErrNodeGroupNameEmpty
		}

		if nodeGroup.VMSpecName == "" {
			return ErrNodeGroupVMSpecEmpty
		}

		if !(nodeGroup.RootDiskSize == "" || strings.ToLower(nodeGroup.RootDiskSize) == "default") {
			_, err := strconv.Atoi(nodeGroup.RootDiskSize)
			if err != nil {
				return ErrInvalidNodeGroupDiskSize
			}
		}

		if nodeGroup.KeyPairIID.SystemId == "" {
			return ErrNodeGroupKeypairEmpty
		}

		if nodeGroup.OnAutoScaling == true {
			intMaxNodeSize := int64(nodeGroup.MaxNodeSize)
			intMinNodeSize := int64(nodeGroup.MinNodeSize)
			intDesiredNodeSize := int64(nodeGroup.DesiredNodeSize)

			if !(intMinNodeSize >= 1) {
				return ErrInvalidMinNodeSize
			}
			if !(intMinNodeSize <= intMaxNodeSize) {
				return ErrInvalidMaxNodeSize
			}
			if !(intDesiredNodeSize >= intMinNodeSize && intDesiredNodeSize <= intMaxNodeSize) {
				return ErrInvalidDesiredNodeSize
			}
		}

	}

	return nil
}

func (ClusterHandler *GCPClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (irs.ClusterInfo, error) {
	cblogger.Info("GCP Cloud Driver: called CreateCluster()")

	// as per https://github.com/cloud-barista/cb-spider/issues/1252#issuecomment-2303556504
	// gcp cluster only support create cluster with at least one custom node pool
	if err := validateAtCreateCluster(clusterReqInfo); err != nil {
		return irs.ClusterInfo{}, err
	}

	projectID := ClusterHandler.Credential.ProjectID
	region := ClusterHandler.Region.Region
	zone := ClusterHandler.Region.Zone

	hiscallInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterReqInfo.IId.NameId, "CreateCluster()")

	parent := getParentAtContainer(projectID, zone)
	// parent := "projects/" + projectID + "/locations/" + zone
	//projects/csta-349809/locations/asia-northeast3-a

	// Store the securityGroup info in Meta as Key/Val, and set the actual value (val) in nodeConfig
	labels := make(map[string]string)
	var sgTags []string
	if len(clusterReqInfo.Network.SecurityGroupIIDs) > 0 {
		for idx, securityGroupIID := range clusterReqInfo.Network.SecurityGroupIIDs {
			labels[GCP_PMKS_SECURITYGROUP_TAG+strconv.Itoa(idx)] = securityGroupIID.NameId
			sgTags = append(sgTags, securityGroupIID.NameId)
		}
	}

	for _, t := range clusterReqInfo.TagList {
		labels[t.Key] = t.Value
	}

	reqCluster := container.Cluster{}
	reqCluster.Name = clusterReqInfo.IId.NameId

	// Use the default when no version is given
	if clusterReqInfo.Version != "" {
		reqCluster.InitialClusterVersion = clusterReqInfo.Version
	}

	reqCluster.Network = clusterReqInfo.Network.VpcIID.SystemId
	if len(clusterReqInfo.Network.SubnetIIDs) > 0 {
		reqCluster.Subnetwork = clusterReqInfo.Network.SubnetIIDs[0].NameId
	}

	rb := &container.CreateClusterRequest{}
	rb.Cluster = &reqCluster
	rb.Cluster.ResourceLabels = labels

	// nodeGroup List set
	nodePools := []*container.NodePool{}
	cblogger.Info("clusterReqInfo.NodeGroupList ", len(clusterReqInfo.NodeGroupList))

	// Fetch valid image types from GKE API once per CreateCluster call.
	validImageTypes, err := ClusterHandler.fetchValidGKEImageTypes(context.TODO())
	if err != nil {
		return irs.ClusterInfo{}, err
	}

	if len(clusterReqInfo.NodeGroupList) > 0 {
		// Specify one nodeGroup at initial creation. Add any additional ones with add NodeGroup after creation
		for _, reqNodeGroup := range clusterReqInfo.NodeGroupList {
			nodePool := container.NodePool{}
			nodePool.Name = reqNodeGroup.IId.NameId
			nodePool.InitialNodeCount = int64(reqNodeGroup.DesiredNodeSize)
			if reqNodeGroup.OnAutoScaling {
				nodePoolAutoScaling := container.NodePoolAutoscaling{}
				nodePoolAutoScaling.Enabled = true
				nodePoolAutoScaling.MaxNodeCount = int64(reqNodeGroup.MaxNodeSize)
				nodePoolAutoScaling.MinNodeCount = int64(reqNodeGroup.MinNodeSize)

				nodePool.Autoscaling = &nodePoolAutoScaling
			}

			nodeConfig := container.NodeConfig{}

			if reqNodeGroup.RootDiskSize != "" && strings.ToLower(reqNodeGroup.RootDiskType) != "default" {
				diskSize, err := strconv.ParseInt(reqNodeGroup.RootDiskSize, 10, 64)
				if err != nil {
					return irs.ClusterInfo{}, err
				}

				if diskSize > 0 {
					nodeConfig.DiskSizeGb = diskSize
				}
			}

			if reqNodeGroup.RootDiskType != "" && strings.ToLower(reqNodeGroup.RootDiskType) != "default" {
				nodeConfig.DiskType = reqNodeGroup.RootDiskType
			}

			nodeConfig.MachineType = reqNodeGroup.VMSpecName
			nodeConfig.Tags = sgTags

			// "" and "default" → explicit default; anything else → validate against API list.
			imageNameId := reqNodeGroup.ImageIID.NameId
			if imageNameId == "" || strings.EqualFold(imageNameId, "default") {
				cblogger.Info("No ImageType provided (empty or 'default'), using GKE default (COS_CONTAINERD)")
				nodeConfig.ImageType = "COS_CONTAINERD"
			} else {
				imageType, err := resolveGKEImageType(imageNameId, validImageTypes)
				if err != nil {
					return irs.ClusterInfo{}, err
				}
				nodeConfig.ImageType = imageType
			}

			keyPair := map[string]string{}
			if reqNodeGroup.KeyPairIID.SystemId != "" {
				keyPair[GCP_PMKS_KEYPAIR_KEY] = reqNodeGroup.KeyPairIID.SystemId
				nodeConfig.Labels = keyPair
			}

			nodePool.Config = &nodeConfig

			nodePools = append(nodePools, &nodePool)
			//break // add only one?
		}
		rb.Cluster.NodePools = nodePools
	}
	// else {
	// At least one NodeGroup is passed in, so the cluster's InitialNodeCount cannot be set at the same time.
	// Set it when there is no NodeGroup.
	// reqCluster.InitialNodeCount = 3 // Cluster.initial_node_count must be greater than zero
	// }

	cblogger.Debug(rb)
	// if 1 == 1 {
	// 	return irs.ClusterInfo{}, nil
	// }

	start := call.Start()
	op, err := ClusterHandler.ContainerClient.Projects.Locations.Clusters.Create(parent, rb).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		return irs.ClusterInfo{}, err
	}

	operationErr := WaitContainerOperationFail(ClusterHandler.ContainerClient, projectID, region, zone, op.Name, GCP_CONTAINER_OPERATION_CREATE_CLUSTER)
	if operationErr != nil {
		cblogger.Error(err)
	}
	//

	//createdClusterName := "projects/" + projectID + "/locations/" + zone + "/clusters/" + clusterReqInfo.IId.NameId

	clusterInfo, err := ClusterHandler.GetCluster(irs.IID{NameId: clusterReqInfo.IId.NameId, SystemId: clusterReqInfo.IId.NameId})
	if err != nil {
		err := fmt.Errorf("Failed to Get Cluster Info :  %v", err)
		cblogger.Error(err)
		return irs.ClusterInfo{}, err
	}

	err = ClusterHandler.grantClusterAdminPermission(clusterInfo)
	if err != nil {
		cblogger.Warn(fmt.Sprintf("Failed to grant cluster-admin permission (non-critical): %v", err))
	}

	return clusterInfo, nil
}

// location is a region or a zone
// location is used as a path param, and
// projectId and zone in the existing request object are deprecated
// Location "-" matches all zones and all regions.
func (ClusterHandler *GCPClusterHandler) ListCluster() ([]*irs.ClusterInfo, error) {
	projectID := ClusterHandler.Credential.ProjectID
	//region := ClusterHandler.Region.Region
	zone := ClusterHandler.Region.Zone

	cblogger.Info("GCP Cloud Driver: called ListCluster()")
	hiscallInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, "ListCluster()", "ListCluster()")

	parent := getParentAtContainer(projectID, zone)
	cblogger.Info("parent : ", parent)
	start := call.Start()
	resp, err := ClusterHandler.ContainerClient.Projects.Locations.Clusters.List(parent).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Get ClusterInfo :  %v", err)
		cblogger.Error(err)
		return nil, err
	}
	//cblogger.Debug(resp)

	clusterInfoList := []*irs.ClusterInfo{}

	respClusters := resp.Clusters
	cblogger.Debug(respClusters)
	for _, cluster := range respClusters {
		// clusterInfo, err := mappingClusterInfo(cluster)
		// if err != nil {
		// 	// cluster err
		// 	return nil, err
		// }

		// nodeGroupList := clusterInfo.NodeGroupList
		// for _, nodeGroupInfo := range nodeGroupList {
		// 	keyValueList := nodeGroupInfo.KeyValueList
		// 	for _, keyValue := range keyValueList {
		// 		if strings.HasPrefix(keyValue.Key, GCP_PMKS_INSTANCEGROUP_KEY) {
		// 			nodeList := []irs.IID{}
		// 			instanceGroupValue := keyValue.Value
		// 			instanceList, err := GetInstancesOfInstanceGroup(ClusterHandler.Client, ClusterHandler.Credential, ClusterHandler.Region, instanceGroupValue)
		// 			if err != nil {
		// 				return clusterInfoList, err
		// 			}
		// 			for _, instance := range instanceList {
		// 				instanceInfo, err := GetInstance(ClusterHandler.Client, ClusterHandler.Credential, ClusterHandler.Region, instance)
		// 				if err != nil {
		// 					return clusterInfoList, err
		// 				}
		// 				// Instance ID of the nodeGroup
		// 				nodeIID := irs.IID{NameId: instanceInfo.Name, SystemId: instanceInfo.Name}
		// 				nodeList = append(nodeList, nodeIID)
		// 			}
		// 			nodeGroupInfo.Nodes = nodeList
		// 		}
		// 	}
		// }
		// clusterInfo.NodeGroupList = nodeGroupList
		clusterInfo, err := convertCluster(ClusterHandler.Client, ClusterHandler.Credential, ClusterHandler.Region, cluster)
		if err != nil {
			cblogger.Error(err) // keep listing the next items even if an error occurred
			continue
		}

		clusterInfoList = append(clusterInfoList, &clusterInfo)
	}
	return clusterInfoList, nil
}

func (ClusterHandler *GCPClusterHandler) GetCluster(clusterIID irs.IID) (irs.ClusterInfo, error) {
	projectID := ClusterHandler.Credential.ProjectID
	//region := ClusterHandler.Region.Region
	zone := ClusterHandler.Region.Zone

	cblogger.Info("GCP Cloud Driver: called GetCluster()")
	hiscallInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterIID.NameId, "GetCluster()")

	parent := getParentClusterAtContainer(projectID, zone, clusterIID.NameId)

	start := call.Start()
	resp, err := ClusterHandler.ContainerClient.Projects.Locations.Clusters.Get(parent).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Get ClusterInfo :  %v", err)
		cblogger.Error(err)
		return irs.ClusterInfo{}, err
	}

	// clusterInfo, err := mappingClusterInfo(resp)
	// if err != nil {
	// 	// cluster err
	// 	return irs.ClusterInfo{}, err
	// }

	// //nodePools = resp.NodePools
	// nodeGroupList := clusterInfo.NodeGroupList
	// for _, nodeGroupInfo := range nodeGroupList {
	// 	keyValueList := nodeGroupInfo.KeyValueList
	// 	for _, keyValue := range keyValueList {
	// 		if strings.HasPrefix(keyValue.Key, GCP_PMKS_INSTANCEGROUP_KEY) {
	// 			nodeList := []irs.IID{}
	// 			instanceGroupValue := keyValue.Value
	// 			instanceList, err := GetInstancesOfInstanceGroup(ClusterHandler.Client, ClusterHandler.Credential, ClusterHandler.Region, instanceGroupValue)
	// 			if err != nil {
	// 				return clusterInfo, err
	// 			}
	// 			for _, instance := range instanceList {
	// 				instanceInfo, err := GetInstance(ClusterHandler.Client, ClusterHandler.Credential, ClusterHandler.Region, instance)
	// 				if err != nil {
	// 					return clusterInfo, err
	// 				}
	// 				// Instance ID of the nodeGroup
	// 				nodeIID := irs.IID{NameId: instanceInfo.Name, SystemId: instanceInfo.Name}
	// 				nodeList = append(nodeList, nodeIID)
	// 			}
	// 			nodeGroupInfo.Nodes = nodeList
	// 		}
	// 	}
	// 	nodeGroupList = append(nodeGroupList, nodeGroupInfo)
	// }
	// clusterInfo.NodeGroupList = nodeGroupList

	clusterInfo, err := convertCluster(ClusterHandler.Client, ClusterHandler.Credential, ClusterHandler.Region, resp)
	if err != nil {
		cblogger.Error(err)
		return irs.ClusterInfo{}, err
	}
	return clusterInfo, nil
}

// When only success/failure is returned, wait until Done and then return the result
func (ClusterHandler *GCPClusterHandler) DeleteCluster(clusterIID irs.IID) (bool, error) {
	projectID := ClusterHandler.Credential.ProjectID
	region := ClusterHandler.Region.Region
	zone := ClusterHandler.Region.Zone

	cblogger.Info("GCP Cloud Driver: called DeleteCluster()")
	hiscallInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterIID.NameId, "DeleteCluster()")

	parent := getParentClusterAtContainer(projectID, zone, clusterIID.NameId)
	cblogger.Info("parent : ", parent)
	start := call.Start()
	op, err := ClusterHandler.ContainerClient.Projects.Locations.Clusters.Delete(parent).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to DeleteCluster :  %v", err)
		cblogger.Error(err)
		return false, err
	}
	cblogger.Debug(op)

	operationErr := WaitContainerOperationFail(ClusterHandler.ContainerClient, projectID, region, zone, op.Name, GCP_CONTAINER_OPERATION_DELETE_CLUSTER)
	if operationErr != nil {
		cblogger.Error(err)
		return false, err
	}

	return true, nil
}

// A lookup shows the status as in progress, so wait a while after the operation to see whether it fails
// If it does not fail, stop waiting and look it up
func (ClusterHandler *GCPClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (irs.NodeGroupInfo, error) {
	cblogger.Info("GCP Cloud Driver: called AddNodeGroup()")
	if err := validateAtAddNodeGroup(clusterIID, nodeGroupReqInfo); err != nil {
		return irs.NodeGroupInfo{}, err
	}

	nodeGroupInfo := irs.NodeGroupInfo{}

	projectID := ClusterHandler.Credential.ProjectID
	region := ClusterHandler.Region.Region
	zone := ClusterHandler.Region.Zone

	// Get the cluster
	clusterInfo, err := ClusterHandler.GetCluster(clusterIID)
	if err != nil {
		cblogger.Info("clusterInfo : ", clusterInfo)
		return nodeGroupInfo, err
	}
	var sgTags []string
	if clusterInfo.Network.SecurityGroupIIDs != nil && len(clusterInfo.Network.SecurityGroupIIDs) > 0 {
		for _, securityGroupIID := range clusterInfo.Network.SecurityGroupIIDs {
			sgTags = append(sgTags, securityGroupIID.NameId)
		}
	}
	//cblogger.Debug(nodeGroupReqInfo)
	// param set
	reqNodePool := container.NodePool{}
	reqNodePool.Name = nodeGroupReqInfo.IId.NameId
	reqNodePool.InitialNodeCount = int64(nodeGroupReqInfo.DesiredNodeSize)
	if nodeGroupReqInfo.OnAutoScaling {
		nodePoolAutoScaling := container.NodePoolAutoscaling{}
		nodePoolAutoScaling.Enabled = true
		nodePoolAutoScaling.MaxNodeCount = int64(nodeGroupReqInfo.MaxNodeSize)
		nodePoolAutoScaling.MinNodeCount = int64(nodeGroupReqInfo.MinNodeSize)

		reqNodePool.Autoscaling = &nodePoolAutoScaling
	}

	nodeConfig := container.NodeConfig{}

	if nodeGroupReqInfo.RootDiskSize != "" && strings.ToLower(nodeGroupReqInfo.RootDiskType) != "default" {
		diskSize, err := strconv.ParseInt(nodeGroupReqInfo.RootDiskSize, 10, 64)
		if err != nil {
			return irs.NodeGroupInfo{}, err
		}

		if diskSize > 0 {
			nodeConfig.DiskSizeGb = diskSize
		}
	}

	if nodeGroupReqInfo.RootDiskType != "" && strings.ToLower(nodeGroupReqInfo.RootDiskType) != "default" {
		nodeConfig.DiskType = nodeGroupReqInfo.RootDiskType
	}

	nodeConfig.MachineType = nodeGroupReqInfo.VMSpecName
	nodeConfig.Tags = sgTags

	// Fetch valid image types from GKE API.
	validImageTypes, err := ClusterHandler.fetchValidGKEImageTypes(context.TODO())
	if err != nil {
		return irs.NodeGroupInfo{}, err
	}

	// "" and "default" → explicit default; anything else → validate against API list.
	imageNameId := nodeGroupReqInfo.ImageIID.NameId
	if imageNameId == "" || strings.EqualFold(imageNameId, "default") {
		cblogger.Info("No ImageType provided (empty or 'default'), using GKE default (COS_CONTAINERD)")
		nodeConfig.ImageType = "COS_CONTAINERD"
	} else {
		imageType, err := resolveGKEImageType(imageNameId, validImageTypes)
		if err != nil {
			return irs.NodeGroupInfo{}, err
		}
		nodeConfig.ImageType = imageType
	}

	keyPair := map[string]string{}
	if nodeGroupReqInfo.KeyPairIID.SystemId != "" {
		keyPair[GCP_PMKS_KEYPAIR_KEY] = nodeGroupReqInfo.KeyPairIID.SystemId
		nodeConfig.Labels = keyPair
	}

	// if clusterInfo.Network.SecurityGroupIIDs != nil && len(clusterInfo.Network.SecurityGroupIIDs) > 0 {
	// 	var sgTags []string
	// 	for _, securityGroupIID := range clusterInfo.Network.SecurityGroupIIDs {
	// 		sgTags = append(sgTags, securityGroupIID.NameId)
	// 	}
	// 	nodeConfig.Tags = sgTags
	// }

	reqNodePool.Config = &nodeConfig

	hiscallInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterIID.NameId, "AddNodeGroup()")

	parent := getParentClusterAtContainer(projectID, zone, clusterIID.NameId)
	cblogger.Info("parent : ", parent)
	//cblogger.Debug(reqNodePool)

	rb := &container.CreateNodePoolRequest{
		NodePool: &reqNodePool,
	}

	start := call.Start()
	op, err := ClusterHandler.ContainerClient.Projects.Locations.Clusters.NodePools.Create(parent, rb).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to AddNodeGroup :  %v", err)
		cblogger.Error(err)
		return nodeGroupInfo, err
	}
	//cblogger.Debug(op)

	operationErr := WaitContainerOperationFail(ClusterHandler.ContainerClient, projectID, region, zone, op.Name, GCP_CONTAINER_OPERATION_CREATE_NODE_POOL)
	if operationErr != nil {
		cblogger.Error(err)
		return nodeGroupInfo, err
	}

	nodePool, err := getNodePools(ClusterHandler.ContainerClient, projectID, ClusterHandler.Region, clusterIID, nodeGroupReqInfo.IId)
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}

	nodeGroupInfo, err = mappingNodeGroupInfo(nodePool)
	if err != nil {
		cblogger.Error(err)
		return nodeGroupInfo, err
	}

	nodeList := []irs.IID{}
	keyValueList := nodeGroupInfo.KeyValueList
	for _, keyValue := range keyValueList {
		cblogger.Info("keyValue : ", keyValue)
		if strings.HasPrefix(keyValue.Key, GCP_PMKS_INSTANCEGROUP_KEY) {
			cblogger.Info("keyValue HasPrefix: ")

			instanceGroupValue := keyValue.Value
			instanceList, err := GetInstancesOfInstanceGroup(ClusterHandler.Client, ClusterHandler.Credential, ClusterHandler.Region, instanceGroupValue)
			if err != nil {
				cblogger.Error(err)
				return nodeGroupInfo, err
			}
			cblogger.Info("instanceList: ", instanceList)
			for _, instance := range instanceList {
				instanceInfo, err := GetInstance(ClusterHandler.Client, ClusterHandler.Credential, ClusterHandler.Region, instance)
				if err != nil {
					cblogger.Error(err)
					return nodeGroupInfo, err
				}
				cblogger.Info("instanceInfo: ", instanceInfo)
				// Instance ID of the nodeGroup
				nodeIID := irs.IID{NameId: instanceInfo.Name, SystemId: instanceInfo.Name}
				nodeList = append(nodeList, nodeIID)
			}
		}
	}
	cblogger.Info("nodeList: ", nodeList)
	nodeGroupInfo.Nodes = nodeList

	return nodeGroupInfo, nil

}

// Changes only the true/false of autoScaling.
func (ClusterHandler *GCPClusterHandler) SetNodeGroupAutoScaling(clusterIID irs.IID, nodeGroupIID irs.IID, on bool) (bool, error) {
	if on == true {
		// https://github.com/cloud-barista/cb-spider/issues/1329
		return false, fmt.Errorf("Enabling autoscaling through the SetNodeGroupAutoScaling API is not supported in GCP. " +
			"Please use the ChangeNodeGroupScaling API instead")
	}

	projectID := ClusterHandler.Credential.ProjectID
	region := ClusterHandler.Region.Region
	zone := ClusterHandler.Region.Zone

	parent := getParentNodePoolsAtContainer(projectID, zone, clusterIID.NameId, nodeGroupIID.NameId)

	rb := &container.SetNodePoolAutoscalingRequest{
		Autoscaling: &container.NodePoolAutoscaling{Enabled: on},
	}
	cblogger.Debug(rb)

	hiscallInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterIID.NameId, "SetNodeGroupAutoScaling()")
	start := call.Start()
	op, err := ClusterHandler.ContainerClient.Projects.Locations.Clusters.NodePools.SetAutoscaling(parent, rb).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to SetNodeGroupAutoScaling :  %v", err)
		cblogger.Error(err)
		return false, err
	}
	cblogger.Debug(op)

	operationErr := WaitContainerOperationDone(ClusterHandler.ContainerClient, projectID, region, zone, op.Name, GCP_CONTAINER_OPERATION_SET_NODE_POOL_MANAGEMENT, 30)
	if operationErr != nil {
		cblogger.Error(operationErr)
		return false, operationErr
	}

	return true, nil
}

// Changes the autoScaling settings.
// TODO : should the current autoScaling settings be looked up and set only when they differ?
func (ClusterHandler *GCPClusterHandler) ChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID, desiredNodeSize int, minNodeSize int, maxNodeSize int) (irs.NodeGroupInfo, error) {
	cblogger.Info("GCP Cloud Driver: called ChangeNodeGroupScaling()")
	if err := validateAtChangeNodeGroupScaling(clusterIID, nodeGroupIID, desiredNodeSize, minNodeSize, maxNodeSize); err != nil {
		return irs.NodeGroupInfo{}, err
	}

	projectID := ClusterHandler.Credential.ProjectID
	region := ClusterHandler.Region.Region
	zone := ClusterHandler.Region.Zone

	parent := getParentNodePoolsAtContainer(projectID, zone, clusterIID.NameId, nodeGroupIID.NameId)

	// orgnodePool
	orgNodePool, err := getNodePools(ClusterHandler.ContainerClient, projectID, ClusterHandler.Region, clusterIID, nodeGroupIID)
	if err != nil {
		return irs.NodeGroupInfo{}, err
	}

	intMaxNodeSize := int64(maxNodeSize)
	intMinNodeSize := int64(minNodeSize)
	intDesiredNodeSize := int64(desiredNodeSize)

	// Change the autoScaling min/max
	orgAutoScaling := orgNodePool.Autoscaling
	hiscallInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterIID.NameId, "ChangeNodeGroupScaling()")

	// When min or max changes
	if intMaxNodeSize > 0 || intMinNodeSize > 0 {
		// If the existing autoscaling was false, both values are required.
		if orgAutoScaling == nil || orgAutoScaling.Enabled == false {
			cblogger.Info("autoScaling : ", orgAutoScaling)
			orgAutoScaling = &container.NodePoolAutoscaling{}
			orgAutoScaling.Enabled = true
			orgAutoScaling.MaxNodeCount = intMaxNodeSize
			orgAutoScaling.MinNodeCount = intMinNodeSize
		} else {
			// When autoscaling == true, min and max must differ from the existing values. Set only the ones that differ
			newCount := 0
			if intMaxNodeSize > 0 && orgAutoScaling.MaxNodeCount != intMaxNodeSize {
				cblogger.Info("intMaxNodeSize : ", intMaxNodeSize)
				orgAutoScaling.MaxNodeCount = intMaxNodeSize
				newCount++
			}
			if intMinNodeSize > 0 && orgAutoScaling.MinNodeCount != intMinNodeSize {
				cblogger.Info("intMinNodeSize : ", intMinNodeSize)
				orgAutoScaling.MinNodeCount = intMinNodeSize
				newCount++
			}

			if intDesiredNodeSize > 0 && orgNodePool.InitialNodeCount != intDesiredNodeSize {
				newCount++
			}

			if newCount == 0 {
				return irs.NodeGroupInfo{}, errors.New("Mininum, Maximum, Desired Nodesize are all the same as before")
			}
		}

		rb := &container.SetNodePoolAutoscalingRequest{
			Autoscaling: orgAutoScaling,
		}

		start := call.Start()
		op, err := ClusterHandler.ContainerClient.Projects.Locations.Clusters.NodePools.SetAutoscaling(parent, rb).Do()
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		if err != nil {
			err := fmt.Errorf("Failed to SetNodeGroupAutoScaling :  %v", err)
			cblogger.Error(err)
			return irs.NodeGroupInfo{}, err
		}
		cblogger.Debug(op)

		operationErr := WaitContainerOperationDone(ClusterHandler.ContainerClient, projectID, region, zone, op.Name, GCP_CONTAINER_OPERATION_SET_NODE_POOL_MANAGEMENT, 30)
		if operationErr != nil {
			cblogger.Error(operationErr)
			return irs.NodeGroupInfo{}, err
		}

	}
	// case1 : if orgAutoScaling == false, change it to true
	//			both min and max are required

	// case2 : if orgAutoScaling == true
	//			changing only one of min and max is enough.

	// case3 : for a desired-size change
	//			set it if it differs from the existing value. --> a different API.

	// 1. autoscaling off -> on
	//    on, min, and max must also be specified
	// 2. autoscaling on -> on. min, max change
	//	  the existing autoScaling must be on, and
	//	  one of min and max must differ.
	// 3. initNodeCount change
	//	  it must differ from the existing initNodeCount.

	// if orgAutoScaling == nil || orgAutoScaling.Enabled == false {
	// 	cblogger.Info("autoScaling : ", orgAutoScaling)
	// 	orgAutoScaling = &container.NodePoolAutoscaling{}
	// 	orgAutoScaling.Enabled = true
	// }

	// if intMaxNodeSize > 0 && orgAutoScaling.MaxNodeCount != intMaxNodeSize {
	// 	cblogger.Info("intMaxNodeSize : ", intMaxNodeSize)
	// 	orgAutoScaling.MaxNodeCount = intMaxNodeSize
	// }
	// if intMinNodeSize > 0 && orgAutoScaling.MinNodeCount != intMinNodeSize {
	// 	cblogger.Info("intMinNodeSize : ", intMinNodeSize)
	// 	orgAutoScaling.MinNodeCount = intMinNodeSize
	// }

	// Change the autoScaling desired node count
	if intDesiredNodeSize > 0 && orgNodePool.InitialNodeCount != intDesiredNodeSize {
		cblogger.Info("InitialNodeCount : ", orgNodePool.InitialNodeCount)
		cblogger.Info("desiredNodeSize : ", intDesiredNodeSize)
		rb2 := &container.SetNodePoolSizeRequest{
			NodeCount: intDesiredNodeSize,
		}

		hiscallInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterIID.NameId, "SetNodeGroupAutoScaling()")
		start := call.Start()
		op2, err2 := ClusterHandler.ContainerClient.Projects.Locations.Clusters.NodePools.SetSize(parent, rb2).Do()
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		if err2 != nil {
			err2 := fmt.Errorf("Failed to SetNodeGroupAutoScaling :  %v", err2)
			cblogger.Error(err2)
			return irs.NodeGroupInfo{}, err
		}
		cblogger.Debug(op2)

		operationErr2 := WaitContainerOperationDone(ClusterHandler.ContainerClient, projectID, region, zone, op2.Name, GCP_CONTAINER_OPERATION_SET_NODE_POOL_SIZE, 30)
		if operationErr2 != nil {
			cblogger.Error(operationErr2)
			return irs.NodeGroupInfo{}, err
		}
	}

	// When processing is done, get the NodePool
	nodePool, err := getNodePools(ClusterHandler.ContainerClient, projectID, ClusterHandler.Region, clusterIID, nodeGroupIID)
	if err != nil {
		return irs.NodeGroupInfo{}, err
	}
	return mappingNodeGroupInfo(nodePool)

}

// When only success/failure is returned, wait until Done and then return the result
func (ClusterHandler *GCPClusterHandler) RemoveNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (bool, error) {
	projectID := ClusterHandler.Credential.ProjectID
	region := ClusterHandler.Region.Region
	zone := ClusterHandler.Region.Zone

	cblogger.Info("GCP Cloud Driver: called RemoveNodeGroup()")
	hiscallInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterIID.NameId, "RemoveNodeGroup()")

	parent := getParentNodePoolsAtContainer(projectID, zone, clusterIID.NameId, nodeGroupIID.NameId)
	cblogger.Info("parent : ", parent)

	start := call.Start()
	op, err := ClusterHandler.ContainerClient.Projects.Locations.Clusters.NodePools.Delete(parent).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to RemoveNodeGroup :  %v", err)
		cblogger.Error(err)
		return false, err
	}
	cblogger.Debug(op)

	operationErr := WaitContainerOperationDone(ClusterHandler.ContainerClient, projectID, region, zone, op.Name, GCP_CONTAINER_OPERATION_DELETE_NODE_POOL, 600)
	if operationErr != nil {
		cblogger.Error(err)
		return false, err
	}

	return true, nil
}

// cluster version upgrade
// A lookup shows the status as in progress, so wait a while after the operation to see whether it fails
// If it does not fail, stop waiting and look it up
func (ClusterHandler *GCPClusterHandler) UpgradeCluster(clusterIID irs.IID, newVersion string) (irs.ClusterInfo, error) {
	clusterInfo := irs.ClusterInfo{}

	projectID := ClusterHandler.Credential.ProjectID
	region := ClusterHandler.Region.Region
	zone := ClusterHandler.Region.Zone

	cblogger.Info("GCP Cloud Driver: called UpgradeCluster()")
	hiscallInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterIID.NameId, "UpgradeCluster()")

	parent := getParentClusterAtContainer(projectID, zone, clusterIID.NameId)

	// Get the current cluster version
	currentCluster, err := ClusterHandler.ContainerClient.Projects.Locations.Clusters.Get(parent).Do()
	if err != nil {
		return clusterInfo, fmt.Errorf("Failed to get current cluster version: %v", err)
	}

	// Version the node pools are upgraded to: the control plane's actual version, since
	// newVersion may be an alias (e.g. "1.35") that GKE resolves to a full gke.N patch.
	masterVersion := currentCluster.CurrentMasterVersion

	// If the control plane is already at the requested version, upgrade only the node pools
	if gkeVersionMatches(currentCluster.CurrentMasterVersion, newVersion) {
		cblogger.Info(fmt.Sprintf("Control plane is already at version %s (requested %s), skipping control plane upgrade",
			currentCluster.CurrentMasterVersion, newVersion))
	} else {
		// Control plane upgrade
		rb := &container.UpdateMasterRequest{
			MasterVersion: newVersion,
		}

		start := call.Start()
		op, err := ClusterHandler.ContainerClient.Projects.Locations.Clusters.UpdateMaster(parent, rb).Do()
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		if err != nil {
			err := fmt.Errorf("Failed to UpgradeCluster: %v", err)
			cblogger.Error(err)
			return clusterInfo, err
		}
		cblogger.Debug(op)

		// Use WaitContainerOperationDone (20-minute timeout)
		operationErr := WaitContainerOperationDone(ClusterHandler.ContainerClient, projectID, region, zone, op.Name, GCP_CONTAINER_OPERATION_UPDATE_CLUSTER, 1200)
		if operationErr != nil {
			cblogger.Error(operationErr)
			return clusterInfo, operationErr
		}

		// Check the cluster state after the control plane upgrade
		updatedCluster, err := ClusterHandler.ContainerClient.Projects.Locations.Clusters.Get(parent).Do()
		if err != nil {
			return clusterInfo, fmt.Errorf("Failed to get cluster after master upgrade: %v", err)
		}

		// Check that the control plane version was actually upgraded
		if !gkeVersionMatches(updatedCluster.CurrentMasterVersion, newVersion) {
			return clusterInfo, fmt.Errorf("Control plane upgrade not complete. Current version: %s, Expected: %s",
				updatedCluster.CurrentMasterVersion, newVersion)
		}
		masterVersion = updatedCluster.CurrentMasterVersion
		cblogger.Info(fmt.Sprintf("Control plane upgraded successfully to version %s (requested %s)", masterVersion, newVersion))
	}

	// Retry logic before the node pool upgrade
	maxRetries := 10
	retryInterval := 120
	backoffFactor := 1.5

	currentInterval := retryInterval
	for i := 0; i < maxRetries; i++ {
		hasActive, err := ClusterHandler.hasActiveOperations(projectID, zone, clusterIID.NameId)
		if err != nil {
			return clusterInfo, err
		}

		if !hasActive {
			break // continue when no operation is in progress
		}

		if i == maxRetries-1 {
			return clusterInfo, fmt.Errorf("Cluster has active operations after %d retries", maxRetries)
		}

		cblogger.Info(fmt.Sprintf("Cluster has active operations, waiting %d seconds before retry (%d/%d)",
			currentInterval, i+1, maxRetries))
		time.Sleep(time.Duration(currentInterval) * time.Second)

		// Apply exponential backoff
		currentInterval = int(float64(currentInterval) * backoffFactor)
	}

	// List the node pools
	cblogger.Info(fmt.Sprintf("Fetching node pools for cluster: %s", clusterIID.NameId))

	nodePools, err := ClusterHandler.ContainerClient.Projects.Locations.Clusters.NodePools.List(parent).Do()
	if err != nil {
		err := fmt.Errorf("Failed to list Node Pools: %v", err)
		cblogger.Error(err)
		return clusterInfo, err // return clusterInfo when listing the node pools fails
	}

	// Worker node (node pool) upgrade
	for _, nodePool := range nodePools.NodePools { // upgrade each node pool in turn
		// cblogger.Info(fmt.Sprintf("Upgrading Node Pool: %s", nodePool.Name))
		cblogger.Info(fmt.Sprintf("Upgrading Node Pool: %s to version %s", nodePool.Name, masterVersion))

		nodePoolParent := fmt.Sprintf("projects/%s/locations/%s/clusters/%s/nodePools/%s", projectID, zone, clusterIID.NameId, nodePool.Name)
		nodePoolRequest := &container.UpdateNodePoolRequest{
			NodeVersion: masterVersion, // upgrade each node pool to the control plane's version
		}

		// Log before sending the upgrade request
		cblogger.Info(fmt.Sprintf("Sending upgrade request for Node Pool: %s with path: %s", nodePool.Name, nodePoolParent))

		nodeOp, err := ClusterHandler.ContainerClient.Projects.Locations.Clusters.NodePools.Update(nodePoolParent, nodePoolRequest).Do()
		if err != nil {
			err := fmt.Errorf("Failed to Upgrade Node Pool: %v", err)
			cblogger.Error(err)
			return clusterInfo, err // return clusterInfo when a node pool upgrade fails
		}

		// Log that the GCP upgrade request was sent
		cblogger.Info(fmt.Sprintf("Upgrade request sent for Node Pool: %s, Operation Name: %s", nodePool.Name, nodeOp.Name))

		// Use WaitContainerOperationDone (20-minute timeout)
		operationErr := WaitContainerOperationDone(ClusterHandler.ContainerClient, projectID, region, zone, nodeOp.Name, GCP_CONTAINER_OPERATION_UPGRADE_NODES, 1200)
		if operationErr != nil {
			return clusterInfo, operationErr
		}

		cblogger.Info(fmt.Sprintf("Node Pool %s upgrade completed", nodePool.Name))
	}

	return ClusterHandler.GetCluster(clusterIID)
}

// gkeVersionMatches reports whether actual, a full GKE version such as "1.35.6-gke.1250001",
// satisfies requested. requested is either that exact version or a GKE version alias:
// "1.X" picks a patch of minor 1.X, and "1.X.Y" picks a gke.N build of 1.X.Y.
func gkeVersionMatches(actual, requested string) bool {
	if strings.EqualFold(actual, requested) {
		return true
	}
	if requested == "" || strings.Contains(requested, "-") {
		return false // empty, or an explicit gke.N version that did not match exactly
	}
	return strings.HasPrefix(actual, requested+".") || strings.HasPrefix(actual, requested+"-")
}

// location is a region or a zone.
func getParentAtContainer(projectID string, location string) string {
	parent := "projects/" + projectID + "/locations/" + location
	return parent
}

func getParentClusterAtContainer(projectID string, location string, clusters string) string {
	parent := "projects/" + projectID + "/locations/" + location + "/clusters/" + clusters
	return parent
}

func getParentNodePoolsAtContainer(projectID string, location string, clusters string, nodePools string) string {
	parent := "projects/" + projectID + "/locations/" + location + "/clusters/" + clusters + "/nodePools/" + nodePools
	return parent
}

// Converts a container.Cluster into Spider's clusterInfo
// The cluster's nodePool has only instanceGroupUrl, without the nodeGroup details
// so the nodepool must be looked up separately.
func mappingClusterInfo(cluster *container.Cluster) (ClusterInfo irs.ClusterInfo, err error) {
	clusterInfo := irs.ClusterInfo{}

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Failed to Process mappingClusterInfo() : %v\n\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	//networkIID := irs.IID{NameId: cluster.Network, SystemId: cluster.Network}
	//SubnetIID := irs.IID{NameId:   cluster.Subnetwork, SystemId: cluster.Subnetwork}
	//securityGroupIID := irs.IID{NameId: cluster.AuthenticatorGroupsConfig.SecurityGroup, SystemId: cluster.AuthenticatorGroupsConfig.SecurityGroup}

	//// ---- ClusterInfo Area ----////
	// 1. IId IID // {NameId, SystemId}
	clusterIID := irs.IID{NameId: cluster.Name, SystemId: cluster.Name}

	// 2. Version string // Kubernetes Version, ex) 1.23.3
	clusterVersion := cluster.InitialClusterVersion //initialClusterVersion, currentMasterVersion, currentNodeVersion

	// 3. Network       NetworkInfo
	securityGroups := []irs.IID{} // Labels extracted as SecurityGroup
	var metaSecurityGroupTags []string

	tags := make([]irs.KeyValue, 0)
	if cluster.ResourceLabels != nil {
		for resourceKey, resourceVal := range cluster.ResourceLabels {
			if strings.HasPrefix(resourceKey, GCP_PMKS_SECURITYGROUP_TAG) {
				//securityGroups = append(securityGroups, irs.IID{NameId: resourceVal, SystemId: resourceVal})
				metaSecurityGroupTags = append(metaSecurityGroupTags, resourceVal)
			}
			tags = append(tags, irs.KeyValue{Key: resourceKey, Value: resourceVal})
		}
	}
	clusterInfo.TagList = tags

	cblogger.Info("metaSecurityGroupTags : ", metaSecurityGroupTags)
	// To tell whether a NodeConfig Tag is the one used as a SecurityGroup,
	// check whether the Label is defined in Metadata
	// Create puts the same value in Metadata and the nodeConfig Tag; is a second check really needed?
	for _, securityGroupTag := range metaSecurityGroupTags {
		securityGroups = append(securityGroups, irs.IID{NameId: securityGroupTag, SystemId: securityGroupTag})
	}
	// nodeConfigTags := cluster.NodeConfig.Tags
	// for _, nodeConfigTag := range nodeConfigTags {
	// 	cblogger.Info("nodeConfigTag len : ", len(nodeConfigTags))
	// 	cblogger.Info("nodeConfigTag : ", nodeConfigTag)
	// 	for _, securityGroupTag := range metaSecurityGroupTags {
	// 		if nodeConfigTag == securityGroupTag {
	// 			securityGroups = append(securityGroups, irs.IID{NameId: securityGroupTag, SystemId: securityGroupTag})
	// 		}
	// 	}
	// 	// 	if strings.HasPrefix(nodeConfigTag, GCP_PMKS_SECURITYGROUP_TAG) {
	// 	// 		securityGroupName := nodeConfigTag[len(GCP_PMKS_SECURITYGROUP_TAG):]
	// 	// 		securityGroups = append(securityGroups, irs.IID{NameId: securityGroupName, SystemId: securityGroupName})
	// 	// 	} else {
	// 	// 		// securityGroup is required
	// 	// 	}
	// }

	networkInfo := irs.NetworkInfo{
		VpcIID:     irs.IID{NameId: cluster.Network, SystemId: cluster.Network},
		SubnetIIDs: []irs.IID{{NameId: cluster.Subnetwork, SystemId: cluster.Subnetwork}},
		// VpcIID:            irs.IID{NameId: cluster.NetworkConfig.Network, SystemId: cluster.NetworkConfig.Network},
		// SubnetIIDs:        []irs.IID{{NameId: cluster.NetworkConfig.Subnetwork, SystemId: cluster.NetworkConfig.Subnetwork}},
		SecurityGroupIIDs: securityGroups,
		//SecurityGroupIIDs: []irs.IID{{NameId: cluster.AuthenticatorGroupsConfig.SecurityGroup, SystemId: cluster.AuthenticatorGroupsConfig.SecurityGroup}},
		// KeyValueList: ,
	}

	// 4. NodeGroupList []NodeGroupInfo
	var nodeGroupList []irs.NodeGroupInfo
	if cluster.NodePools != nil && len(cluster.NodePools) > 0 {
		for _, nodePool := range cluster.NodePools {
			// nodePoolName := nodePool.Name
			// imageType := nodePool.Config.ImageType     // COS_CONTAINERD
			// diskSize := nodePool.Config.DiskSizeGb     // 100Gb
			// diskType := nodePool.Config.DiskType       // pd-standard
			// machineType := nodePool.Config.MachineType // e2-medium

			// // diskSize := cluster.NodeConfig.DiskSizeGb// 100Gb
			// // diskType := cluster.NodeConfig.DiskType// pd-standard
			// // machineType := cluster.NodeConfig.MachineType// e2-medium

			// var maxNodeSize int
			// var minNodeSize int
			// var desiredNodeSize int
			// var autoScaling bool
			// if nodePool.Autoscaling != nil && nodePool.Autoscaling.Enabled {
			// 	autoScaling = nodePool.Autoscaling.Enabled
			// 	maxNodeSize = int(nodePool.Autoscaling.MaxNodeCount)
			// 	minNodeSize = int(nodePool.Autoscaling.MinNodeCount)
			// 	desiredNodeSize = int(nodePool.InitialNodeCount)
			// }

			// IId IID // {NameId, SystemId}

			// // VM config.
			// ImageIID     IID
			// VMSpecName   string
			// // ---

			// Status       NodeGroupStatus
			// Nodes        []IID

			// KeyValueList []KeyValue
			// nodeGroupInfo := irs.NodeGroupInfo{}
			// nodeGroupInfo.IId = irs.IID{NameId: nodePoolName, SystemId: nodePoolName}
			// nodeGroupInfo.RootDiskSize = strconv.FormatInt(diskSize, 10)
			// nodeGroupInfo.RootDiskType = diskType
			// nodeGroupInfo.VMSpecName = machineType
			// nodeGroupInfo.ImageIID = irs.IID{NameId: imageType, SystemId: imageType}
			// nodeGroupInfo.DesiredNodeSize = desiredNodeSize
			// if autoScaling {
			// 	nodeGroupInfo.MaxNodeSize = maxNodeSize
			// 	nodeGroupInfo.MinNodeSize = minNodeSize
			// 	nodeGroupInfo.OnAutoScaling = autoScaling
			// }

			//nodeGroupInfo.Nodes : requires a separate API call
			nodeGroupInfo, err := mappingNodeGroupInfo(nodePool)
			if err != nil {
				return clusterInfo, err
			}
			nodeGroupInfo.KeyPairIID = irs.IID{NameId: "NameId", SystemId: "SystemId"} // for the test

			nodeGroupList = append(nodeGroupList, nodeGroupInfo)
		}
	}

	// 5. AccessInfo    AccessInfo
	kubeConfig := "Kubeconfig is not ready yet!"
	if !reflect.ValueOf(cluster.MasterAuth).IsNil() {
		kubeConfig = getKubeConfig(cluster)
	}

	accessInfo := irs.AccessInfo{
		Endpoint:   cluster.Endpoint,
		Kubeconfig: kubeConfig,
	}

	// 6. Addons        AddonsInfo
	addOns := []irs.KeyValue{}
	//addOns = append(addOns, irs.KeyValue{Key: "CloudRunConfig.LoadBalancerType", Value: cluster.AddonsConfig.CloudRunConfig.LoadBalancerType})
	addOnsInfo := irs.AddonsInfo{}
	if addOns != nil && len(addOns) > 0 {
		addOnsInfo.KeyValueList = addOns
	}

	// 7. Status        ClusterStatus
	clusterStatus := getClusterStatus(cluster.Status)
	cblogger.Info("Cluster status : ", cluster.Status, clusterStatus)

	// 8. CreatedTime  time.Time
	createDatetime, err := time.Parse(time.RFC3339, cluster.CreateTime)
	if err != nil {
		err := fmt.Errorf("Failed to Parse Created Time :  %v", err)
		cblogger.Error(err)
		return clusterInfo, err
	}

	// 9. KeyValueList []KeyValue

	// set all properties
	clusterInfo.IId = clusterIID
	clusterInfo.Version = clusterVersion
	clusterInfo.Network = networkInfo
	clusterInfo.AccessInfo = accessInfo
	clusterInfo.NodeGroupList = nodeGroupList
	clusterInfo.Status = clusterStatus
	clusterInfo.CreatedTime = createDatetime
	clusterInfo.Addons = addOnsInfo

	// 2025-03-13 Changed to use StructToKeyValueList
	clusterInfo.KeyValueList = irs.StructToKeyValueList(cluster)
	// cblogger.Debug(clusterInfo)

	return clusterInfo, nil
}

func mappingNodeGroupInfo(nodePool *container.NodePool) (NodeGroupInfo irs.NodeGroupInfo, err error) {
	nodeGroupInfo := irs.NodeGroupInfo{}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Failed to Process mappingNodeGroupInfo() : %v\n\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	nodeGroupInfo.IId.NameId = nodePool.Name
	nodeGroupInfo.IId.SystemId = nodePool.Name

	imageType := nodePool.Config.ImageType // COS_CONTAINERD

	// scaling
	if nodePool.Autoscaling != nil && nodePool.Autoscaling.Enabled {
		nodeGroupInfo.OnAutoScaling = nodePool.Autoscaling.Enabled
		nodeGroupInfo.MaxNodeSize = int(nodePool.Autoscaling.MaxNodeCount)
		nodeGroupInfo.MinNodeSize = int(nodePool.Autoscaling.MinNodeCount)
	}
	nodeGroupInfo.DesiredNodeSize = int(nodePool.InitialNodeCount)

	nodeGroupStatus := getNodeGroupStatus(nodePool.Status)
	nodeGroupInfo.Status = nodeGroupStatus
	nodeGroupInfo.ImageIID = irs.IID{NameId: imageType, SystemId: imageType}
	nodeGroupInfo.VMSpecName = nodePool.Config.MachineType

	nodeGroupInfo.RootDiskSize = strconv.FormatInt(nodePool.Config.DiskSizeGb, 10)
	nodeGroupInfo.RootDiskType = nodePool.Config.DiskType

	// 2025-03-13 Changed to use StructToKeyValueList. InstanceGroup_idx and GCP_PMKS_KEYPAIR_KEY are kept
	keyValueList := irs.StructToKeyValueList(nodePool)

	if nodePool.InstanceGroupUrls != nil {
		for idx, instanceGroupUrl := range nodePool.InstanceGroupUrls {
			urlArr := strings.Split(instanceGroupUrl, "/")
			instanceGroupName := urlArr[len(urlArr)-1]
			keyValueList = append(keyValueList, irs.KeyValue{Key: "InstanceGroup_" + strconv.Itoa(idx), Value: instanceGroupName})

			// InstanceGroup.ListInstances
			//"instance": "https://www.googleapis.com/compute/v1/projects/csta-349809/zones/asia-northeast1-a/instances/gke-spider-cluster-03-ce-default-pool-3082fa6f-kvks",
		}
	}

	// add keypair label
	if nodePool.Config != nil && nodePool.Config.Labels != nil {
		for k, v := range nodePool.Config.Labels {
			if strings.HasPrefix(k, GCP_PMKS_KEYPAIR_KEY) {
				keyValueList = append(keyValueList, irs.KeyValue{Key: GCP_PMKS_KEYPAIR_KEY, Value: v})
			}
		}
	}
	nodeGroupInfo.KeyValueList = keyValueList
	// nodeGroupInfo.KeyValueList = keyValueList

	return nodeGroupInfo, nil
}

// Cluster status
func getClusterStatus(clusterStatus string) irs.ClusterStatus {
	status := irs.ClusterInactive
	if strings.EqualFold(clusterStatus, "PROVISIONING") {
		status = irs.ClusterCreating
	} else if strings.EqualFold(clusterStatus, "RECONCILING") {
		status = irs.ClusterUpdating
	} else if strings.EqualFold(clusterStatus, "STOPPING") {
		status = irs.ClusterDeleting
	} else if strings.EqualFold(clusterStatus, "RUNNING") {
		status = irs.ClusterActive
	} else if strings.EqualFold(clusterStatus, "ERROR") {

	}
	return status
}

// NodeGroup status
func getNodeGroupStatus(nodePoolStatus string) irs.NodeGroupStatus {
	status := irs.NodeGroupInactive
	if strings.EqualFold(nodePoolStatus, "PROVISIONING") {
		status = irs.NodeGroupCreating
	} else if strings.EqualFold(nodePoolStatus, "RECONCILING") {
		status = irs.NodeGroupUpdating
	} else if strings.EqualFold(nodePoolStatus, "STOPPING") {
		status = irs.NodeGroupDeleting
	} else if strings.EqualFold(nodePoolStatus, "RUNNING") {
		status = irs.NodeGroupActive
	} else if strings.EqualFold(nodePoolStatus, "ERROR") {

	}
	return status
}

// update autoScaling
// TODO : look up the nodePool and send only what changes? getNodePools(containerClient *container.Service, projectID string, region string, zone string, clusterIID irs.IID, nodeGroupIID irs.IID) (*container.NodePool, error)
// Set only the needed parameters and update what differs from org
// func updateNodeGroupAutoScaling(containerClient *container.Service, projectID string, region string, zone string, clusterIID irs.IID, orgNodeGroupReqInfo irs.NodeGroupInfo, nodeGroupReqInfo irs.NodeGroupInfo, autoscalingType string) (bool, error) {
// 	reqNodePool := container.NodePool{}
// 	autoScaling := container.NodePoolAutoscaling{}

// 	parent := getParentNodePoolsAtContainer(projectID, zone, clusterIID.NameId, nodeGroupReqInfo.IId.NameId)
// 	cblogger.Debug(nodeGroupReqInfo)
// 	if strings.EqualFold(autoscalingType, GCP_SET_AUTOSCALING_ENABLE) {
// 		if orgNodeGroupReqInfo.OnAutoScaling != nodeGroupReqInfo.OnAutoScaling {
// 			autoScaling.Enabled = nodeGroupReqInfo.OnAutoScaling
// 		}
// 	} else if strings.EqualFold(autoscalingType, GCP_SET_AUTOSCALING_ENABLE) {
// 		if orgNodeGroupReqInfo.DesiredNodeSize != nodeGroupReqInfo.DesiredNodeSize {
// 			reqNodePool.InitialNodeCount = int64(nodeGroupReqInfo.DesiredNodeSize)
// 		}

// 		if orgNodeGroupReqInfo.MaxNodeSize != nodeGroupReqInfo.MaxNodeSize {
// 			reqNodePool.Autoscaling.MaxNodeCount = int64(nodeGroupReqInfo.MaxNodeSize)
// 		}

// 		if orgNodeGroupReqInfo.MinNodeSize != nodeGroupReqInfo.MinNodeSize {
// 			reqNodePool.Autoscaling.MinNodeCount = int64(nodeGroupReqInfo.MinNodeSize)
// 		}
// 	} else {
// 		if orgNodeGroupReqInfo.OnAutoScaling != nodeGroupReqInfo.OnAutoScaling {
// 			reqNodePool.Autoscaling.Enabled = nodeGroupReqInfo.OnAutoScaling
// 		}
// 		if orgNodeGroupReqInfo.DesiredNodeSize != nodeGroupReqInfo.DesiredNodeSize {
// 			reqNodePool.InitialNodeCount = int64(nodeGroupReqInfo.DesiredNodeSize)
// 		}

// 		if orgNodeGroupReqInfo.MaxNodeSize != nodeGroupReqInfo.MaxNodeSize {
// 			reqNodePool.Autoscaling.MaxNodeCount = int64(nodeGroupReqInfo.MaxNodeSize)
// 		}

// 		if orgNodeGroupReqInfo.MinNodeSize != nodeGroupReqInfo.MinNodeSize {
// 			reqNodePool.Autoscaling.MinNodeCount = int64(nodeGroupReqInfo.MinNodeSize)
// 		}
// 	}
// 	reqNodePool.Autoscaling = &autoScaling

// 	cblogger.Info("GCP Cloud Driver: called updateNodeGroupAutoScaling()")
// 	hiscallInfo := GetCallLogScheme(zone, call.CLUSTER, "updateNodeGroupAutoScaling()", "updateNodeGroupAutoScaling()")

// 	rb := &container.SetNodePoolAutoscalingRequest{
// 		Autoscaling: reqNodePool.Autoscaling,
// 	}

// 	start := call.Start()
// 	op, err := containerClient.Projects.Locations.Clusters.NodePools.SetAutoscaling(parent, rb).Do()
// 	hiscallInfo.ElapsedTime = call.Elapsed(start)
// 	if err != nil {
// 		err := fmt.Errorf("Failed to AddNodeGroup :  %v", err)
// 		cblogger.Error(err)
// 		return false, err
// 	}
// 	cblogger.Debug(op)

// 	operationErr := WaitContainerOperationDone(containerClient, projectID, region, zone, op.Name, 3, 1200)
// 	if operationErr != nil {
// 		cblogger.Error(operationErr)
// 		return false, operationErr
// 	}

// 	return true, nil
// }

// Get the NodePool
func getNodePools(containerClient *container.Service, projectID string, region idrv.RegionInfo, clusterIID irs.IID, nodeGroupIID irs.IID) (*container.NodePool, error) {

	parent := getParentNodePoolsAtContainer(projectID, region.Zone, clusterIID.NameId, nodeGroupIID.NameId)

	cblogger.Info("GCP Cloud Driver: called getNodePools() ", parent)
	hiscallInfo := GetCallLogScheme(region, call.CLUSTER, clusterIID.NameId, "getNodePools()")

	start := call.Start()
	nodePool, err := containerClient.Projects.Locations.Clusters.NodePools.Get(parent).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to getNodePools :  %v", err)
		cblogger.Error(err)
		return nil, err
	}

	return nodePool, nil
}

// Set into clusterInfo
// The cluster has only NodeGroup links, so the NodeGroup info must be looked up separately.
func convertCluster(client *compute.Service, credential idrv.CredentialInfo, region idrv.RegionInfo, cluster *container.Cluster) (irs.ClusterInfo, error) {
	clusterInfo, err := mappingClusterInfo(cluster)
	if err != nil {
		// cluster err
		// return irs.ClusterInfo{}, err
	}

	// mappingClusterInfo sets the nodeGroup info first. nodeGroup.KeyValueList contains the instanceGroupID.
	cblogger.Info("nodeGroupList ", clusterInfo.NodeGroupList)
	//nodePools = resp.NodePools
	nodeGroupList, err := convertNodeGroup(client, credential, region, clusterInfo.NodeGroupList)
	if err != err { // pass clusterInfo along even if an error occurred
		// failed to get nodeGroupInfo
		cblogger.Error(err)
	}

	clusterInfo.NodeGroupList = nodeGroupList
	return clusterInfo, nil
}

// Set into nodeGroupInfo
// Node from the nodeGroup info
func convertNodeGroup(client *compute.Service, credential idrv.CredentialInfo, region idrv.RegionInfo, orgNodeGroupList []irs.NodeGroupInfo) ([]irs.NodeGroupInfo, error) {
	nodeGroupList := []irs.NodeGroupInfo{}
	for _, nodeGroupInfo := range orgNodeGroupList {
		cblogger.Info("convertNodeGroup ", nodeGroupInfo)
		//cblogger.Debug(nodeGroupInfo)
		keyValueList := nodeGroupInfo.KeyValueList
		for _, keyValue := range keyValueList {
			cblogger.Info("keyValue ", keyValue)
			if strings.HasPrefix(keyValue.Key, GCP_PMKS_INSTANCEGROUP_KEY) {
				cblogger.Info("HasPrefix ")
				nodeList := []irs.IID{}
				instanceGroupValue := keyValue.Value
				// check instanceGroup exists, if not exists, set empty list
				// The InstanceGroup may not exist while the Cluster is being created.
				exists, err := hasInstanceGroup(client, credential, region, instanceGroupValue)
				if err != nil {
					return nil, err
				}
				if !exists {
					nodeGroupInfo.Nodes = []irs.IID{}
					continue
				}
				instanceList, err := GetInstancesOfInstanceGroup(client, credential, region, instanceGroupValue)
				if err != nil {
					return nodeGroupList, err
				}
				cblogger.Info("instanceList ", instanceList)
				for _, instance := range instanceList {
					instanceInfo, err := GetInstance(client, credential, region, instance)
					if err != nil {
						return nodeGroupList, err
					}
					//cblogger.Debug(instanceInfo)
					// Instance ID of the nodeGroup
					nodeIID := irs.IID{NameId: instanceInfo.Name, SystemId: instanceInfo.Name}
					nodeList = append(nodeList, nodeIID)
				}
				nodeGroupInfo.KeyPairIID = irs.IID{NameId: "NameId", SystemId: "SystemId"} // set a default and update it later, since an empty value causes an error
				cblogger.Info("nodeList ", nodeList)
				nodeGroupInfo.Nodes = nodeList
				//cblogger.Info("nodeGroupInfo ", nodeGroupInfo)
				cblogger.Debug(nodeGroupInfo)
			} else if strings.HasPrefix(keyValue.Key, GCP_PMKS_KEYPAIR_KEY) {
				nodeGroupInfo.KeyPairIID = irs.IID{NameId: keyValue.Value, SystemId: keyValue.Value}
			}
		}
		nodeGroupList = append(nodeGroupList, nodeGroupInfo)
	}
	return nodeGroupList, nil
}

func getKubeConfig(cluster *container.Cluster) string {
	// Return dynamic token kubeconfig instead of gke-gcloud-auth-plugin
	return getDynamicKubeConfig(cluster)
}

// getStaticKubeConfig generates kubeconfig with gke-gcloud-auth-plugin (original format)
func getStaticKubeConfig(cluster *container.Cluster) string {
	configName := fmt.Sprintf("gke_%s_%s", cluster.Location, cluster.Name)

	// Reference format is from `gcloud container clusters get-credentials <cluster name> --location=<location>`
	kubeconfigContent := fmt.Sprintf(`apiVersion: v1
clusters:
- cluster:
    server: https://%s
    certificate-authority-data: %s
  name: %s
contexts:
- context:
    cluster: %s
    user: %s
  name: %s
current-context: %s
kind: Config
preferences: {}
users:
- name: %s
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: gke-gcloud-auth-plugin
      installHint: Install gke-gcloud-auth-plugin for use with kubectl by following
        https://cloud.google.com/kubernetes-engine/docs/how-to/cluster-access-for-kubectl#install_plugin
      provideClusterInfo: true
`, cluster.Endpoint, cluster.MasterAuth.ClusterCaCertificate,
		configName, configName, configName, configName, configName, configName)

	return kubeconfigContent
}

// getDynamicKubeConfig generates kubeconfig with exec-based dynamic token
// Credentials are read from ~/.cb-spider/.spider-credential file at kubectl execution time
// The credentials file should contain: SPIDER_USERNAME=<username> and SPIDER_PASSWORD=<password>
func getDynamicKubeConfig(cluster *container.Cluster) string {
	configName := fmt.Sprintf("gke_%s_%s", cluster.Location, cluster.Name)

	// Get Spider server address from environment variable
	serverAddr := getServerAddress()

	// Generate kubeconfig content with exec-based dynamic token
	// Credentials are sourced from ~/.cb-spider/.spider-credential at runtime (not embedded in kubeconfig)
	kubeconfigContent := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://%s
    certificate-authority-data: %s
  name: %s
contexts:
- context:
    cluster: %s
    user: gcp-dynamic-token
  name: %s
current-context: %s
users:
- name: gcp-dynamic-token
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      interactiveMode: Never
      command: sh
      args:
      - -c
      - ". ~/.cb-spider/.spider-credential && curl -s -u \"$SPIDER_USERNAME:$SPIDER_PASSWORD\" \"http://%s/spider/cluster/CLUSTER_NAME_PLACEHOLDER/token?ConnectionName=CONNECTION_NAME_PLACEHOLDER\""
`, cluster.Endpoint, cluster.MasterAuth.ClusterCaCertificate, configName, configName, configName, configName, serverAddr)

	return kubeconfigContent
}

func validateAtCreateCluster(clusterInfo irs.ClusterInfo) error {
	if clusterInfo.IId.NameId == "" {
		return fmt.Errorf("Cluster name is required")
	}
	if clusterInfo.Network.VpcIID.SystemId == "" && clusterInfo.Network.VpcIID.NameId == "" {
		return fmt.Errorf("Cannot identify VPC(IID=%s)", clusterInfo.Network.VpcIID)
	}
	if len(clusterInfo.Network.SubnetIIDs) < 1 {
		return fmt.Errorf("At least one Subnet must be specified")
	}
	if len(clusterInfo.Network.SecurityGroupIIDs) < 1 {
		return fmt.Errorf("At least one Security Group must be specified")
	}
	if err := validateNodeGroup(clusterInfo.NodeGroupList); err != nil {
		return err
	}

	return nil
}

func validateAtAddNodeGroup(clusterIID irs.IID, nodeGroupInfo irs.NodeGroupInfo) error {
	if clusterIID.SystemId == "" && clusterIID.NameId == "" {
		return fmt.Errorf("Invalid Cluster IID")
	}
	if nodeGroupInfo.IId.NameId == "" {
		return fmt.Errorf("Node Group's name is required")
	}
	if nodeGroupInfo.VMSpecName == "" {
		return fmt.Errorf("VM Spec Name is required")
	}
	if err := validateNodeGroup([]irs.NodeGroupInfo{nodeGroupInfo}); err != nil {
		return err
	}
	return nil
}

func validateAtChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID, desiredNodeSize, minNodeSize, maxNodeSize int) error {
	if clusterIID.SystemId == "" && clusterIID.NameId == "" {
		return fmt.Errorf("Invalid Cluster IID")
	}
	if nodeGroupIID.SystemId == "" && nodeGroupIID.NameId == "" {
		return fmt.Errorf("Invalid Node Group IID")
	}
	if !(minNodeSize >= 1) {
		return ErrInvalidMinNodeSize
	}
	if !(minNodeSize <= maxNodeSize) {
		return ErrInvalidMaxNodeSize
	}
	if !(desiredNodeSize >= minNodeSize && desiredNodeSize <= maxNodeSize) {
		return ErrInvalidDesiredNodeSize
	}
	return nil
}

func (ClusterHandler *GCPClusterHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, string(call.CLUSTER), "ListIID()")
	start := call.Start()

	projectID := ClusterHandler.Credential.ProjectID
	region := ClusterHandler.Region.Region
	parent := getParentAtContainer(projectID, region)
	cblogger.Info("parent for cluster list call : ", parent)

	resp, err := ClusterHandler.ContainerClient.Projects.Locations.Clusters.List(parent).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)

	if err != nil {
		LoggingError(hiscallInfo, err)
		cblogger.Error(err)
		return nil, err
	}
	calllogger.Info(call.String(hiscallInfo))

	var iidList []*irs.IID

	for _, cluster := range resp.Clusters {

		if cluster == nil {
			continue
		}
		iid := irs.IID{NameId: cluster.Name, SystemId: cluster.Name}
		iidList = append(iidList, &iid)
	}
	return iidList, nil
}

// Checks whether the cluster has an operation in progress
func (ClusterHandler *GCPClusterHandler) hasActiveOperations(projectID, zone, clusterName string) (bool, error) {
	listOperationsParent := fmt.Sprintf("projects/%s/locations/%s", projectID, zone)

	// List the operations in progress
	operations, err := ClusterHandler.ContainerClient.Projects.Locations.Operations.List(listOperationsParent).Do()
	if err != nil {
		return false, err
	}

	// Find the in-progress operations related to the cluster
	clusterPattern := fmt.Sprintf("/clusters/%s/", clusterName)
	for _, op := range operations.Operations {
		if strings.Contains(op.TargetLink, clusterPattern) && op.Status != "DONE" {
			cblogger.Info(fmt.Sprintf("Found active operation: %s, status: %s", op.Name, op.Status))
			return true, nil
		}
	}

	return false, nil
}

// GenerateClusterToken generates a token for cluster authentication
// GCP uses OAuth2 access token for GKE cluster access
func (ClusterHandler *GCPClusterHandler) GenerateClusterToken(clusterIID irs.IID) (irs.ClusterToken, error) {
	cblogger.Info("call GenerateClusterToken()")

	// Create JWT config from credential info
	gcpType := "service_account"
	data := make(map[string]string)
	data["type"] = gcpType
	data["private_key"] = ClusterHandler.Credential.PrivateKey
	data["client_email"] = ClusterHandler.Credential.ClientEmail

	res, err := json.Marshal(data)
	if err != nil {
		cblogger.Errorf("Failed to marshal credential data: %v", err)
		return irs.ClusterToken{}, fmt.Errorf("failed to marshal credential data: %w", err)
	}

	// Use cloud-platform scope for GKE access
	authURL := "https://www.googleapis.com/auth/cloud-platform"
	conf, err := goo.JWTConfigFromJSON(res, authURL)
	if err != nil {
		cblogger.Errorf("Failed to create JWT config: %v", err)
		return irs.ClusterToken{}, fmt.Errorf("failed to create JWT config: %w", err)
	}

	// Get token using the JWT config
	tokenSource := conf.TokenSource(o2.NoContext)
	token, err := tokenSource.Token()
	if err != nil {
		cblogger.Errorf("Failed to get access token: %v", err)
		return irs.ClusterToken{}, fmt.Errorf("failed to get access token: %w", err)
	}

	// oauth2.Token already carries the server-provided expiry; do not discard it.
	return irs.ClusterToken{Token: token.AccessToken, ExpiresAt: token.Expiry}, nil
}

func (ClusterHandler *GCPClusterHandler) createK8sClientFromKubeconfig(kubeconfigContent string) (*kubernetes.Clientset, error) {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfigContent))
	if err != nil {
		return nil, fmt.Errorf("failed to create REST config from kubeconfig: %w", err)
	}

	clusterToken, err := ClusterHandler.GenerateClusterToken(irs.IID{})
	if err != nil {
		return nil, fmt.Errorf("failed to generate cluster token: %w", err)
	}
	config.BearerToken = clusterToken.Token

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	return clientset, nil
}

func (ClusterHandler *GCPClusterHandler) getServiceAccountUserID() (string, error) {
	gcpType := "service_account"
	data := make(map[string]string)
	data["type"] = gcpType
	data["private_key"] = ClusterHandler.Credential.PrivateKey
	data["client_email"] = ClusterHandler.Credential.ClientEmail

	res, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal credential data: %w", err)
	}

	authURL := "https://www.googleapis.com/auth/cloud-platform"
	conf, err := goo.JWTConfigFromJSON(res, authURL)
	if err != nil {
		return "", fmt.Errorf("failed to create JWT config: %w", err)
	}

	tokenSource := conf.TokenSource(o2.NoContext)
	token, err := tokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get access token: %w", err)
	}

	if token.Extra("id_token") != nil {
		if idToken, ok := token.Extra("id_token").(string); ok {
			parts := strings.Split(idToken, ".")
			if len(parts) > 1 {
				return parts[1], nil
			}
		}
	}

	return ClusterHandler.Credential.ClientEmail, nil
}

func (ClusterHandler *GCPClusterHandler) grantClusterAdminPermission(clusterInfo irs.ClusterInfo) error {
	kubeconfigContent := clusterInfo.AccessInfo.Kubeconfig
	if kubeconfigContent == "" {
		return fmt.Errorf("kubeconfig is empty")
	}

	clientset, err := ClusterHandler.createK8sClientFromKubeconfig(kubeconfigContent)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	userName, err := ClusterHandler.getServiceAccountUserID()
	if err != nil {
		return fmt.Errorf("failed to get service account user ID: %w", err)
	}

	crbName := "cb-spider-admin"

	existingCRB, err := clientset.RbacV1().ClusterRoleBindings().Get(
		context.TODO(),
		crbName,
		metav1.GetOptions{},
	)
	if err == nil && existingCRB != nil {
		cblogger.Info(fmt.Sprintf("ClusterRoleBinding '%s' already exists, skipping creation", crbName))
		return nil
	}

	crb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: crbName,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind: "User",
				Name: userName,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	_, err = clientset.RbacV1().ClusterRoleBindings().Create(
		context.TODO(),
		crb,
		metav1.CreateOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to create ClusterRoleBinding: %w", err)
	}

	cblogger.Info(fmt.Sprintf("Successfully created ClusterRoleBinding '%s' for user '%s'", crbName, userName))
	return nil
}
