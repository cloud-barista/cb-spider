package resources

import (
	"context"
	"errors"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	compute "google.golang.org/api/compute/v1"
	container "google.golang.org/api/container/v1"
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

type GCPClusterHandler struct {
	Region          idrv.RegionInfo
	Ctx             context.Context
	Client          *compute.Service
	ContainerClient *container.Service
	Credential      idrv.CredentialInfo
}

/*
NodePool 이름이 default-pool로 생성 됨.
Machine Type 이 e2-medium으로 생성 됨.
BootDisk 도 100으로 생성 됨
sg(firewall rule) 추가 안됨.

fail 기다리는것 처리 확인할 것.
*/
func (ClusterHandler *GCPClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (irs.ClusterInfo, error) {
	projectID := ClusterHandler.Credential.ProjectID
	region := ClusterHandler.Region.Region
	zone := ClusterHandler.Region.Zone

	cblogger.Info("GCP Cloud Driver: called CreateCluster()")
	hiscallInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterReqInfo.IId.NameId, "CreateCluster()")

	parent := getParentAtContainer(projectID, zone)
	// parent := "projects/" + projectID + "/locations/" + zone
	//projects/csta-349809/locations/asia-northeast3-a

	// Meta정보에 securityGroup 정보를 Key,Val 형태로 넣고 실제 값(val)은 nodeConfig 에 set하여 사용
	securityGroupMap := make(map[string]string)
	var sgTags []string
	if clusterReqInfo.Network.SecurityGroupIIDs != nil && len(clusterReqInfo.Network.SecurityGroupIIDs) > 0 {
		for idx, securityGroupIID := range clusterReqInfo.Network.SecurityGroupIIDs {
			securityGroupMap[GCP_PMKS_SECURITYGROUP_TAG+strconv.Itoa(idx)] = securityGroupIID.NameId
			sgTags = append(sgTags, securityGroupIID.NameId)
		}
	}

	reqCluster := container.Cluster{}
	reqCluster.Name = clusterReqInfo.IId.NameId

	// version 없으면 default
	if clusterReqInfo.Version != "" {
		reqCluster.InitialClusterVersion = clusterReqInfo.Version
	}

	reqCluster.Network = clusterReqInfo.Network.VpcIID.SystemId
	if len(clusterReqInfo.Network.SubnetIIDs) > 0 {
		reqCluster.Subnetwork = clusterReqInfo.Network.SubnetIIDs[0].NameId
	}

	rb := &container.CreateClusterRequest{}
	rb.Cluster = &reqCluster
	rb.Cluster.ResourceLabels = securityGroupMap

	// nodeGroup List set
	nodePools := []*container.NodePool{}
	cblogger.Info("clusterReqInfo.NodeGroupList ", len(clusterReqInfo.NodeGroupList))
	if clusterReqInfo.NodeGroupList != nil && len(clusterReqInfo.NodeGroupList) > 0 {
		// 최초 생성 시 nodeGroup을 1개 지정함. 2개 이상일 때는 생성 후에 add NodeGroup으로 추가
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
			diskSize, err := strconv.ParseInt(reqNodeGroup.RootDiskSize, 10, 64)
			if err != nil {
				return irs.ClusterInfo{}, err
			}
			if diskSize > 0 {
				nodeConfig.DiskSizeGb = diskSize
			}
			nodeConfig.DiskType = reqNodeGroup.RootDiskType
			nodeConfig.MachineType = reqNodeGroup.VMSpecName
			nodeConfig.Tags = sgTags

			keyPair := map[string]string{}
			if reqNodeGroup.KeyPairIID.SystemId != "" {
				//keyPair[GCP_PMKS_KEYPAIR_KEY] = reqNodeGroup.KeyPairIID.NameId
				keyPair[GCP_PMKS_KEYPAIR_KEY] = reqNodeGroup.KeyPairIID.SystemId
				nodeConfig.Labels = keyPair
			}

			nodePool.Config = &nodeConfig

			nodePools = append(nodePools, &nodePool)
			//break //1개만 add?
		}
		rb.Cluster.NodePools = nodePools
	} else {
		// NodeGroup 이 1개는 넘어오므로 cluster의 InitialNodeCount는 동시에 Set 못함.
		// NodeGroup이 없는경우 Set.
		reqCluster.InitialNodeCount = 3 // Cluster.initial_node_count must be greater than zero
	}

	spew.Dump(rb)
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

	return clusterInfo, nil
}

// location은 region 또는 zone
// path param으로 location이 사용되고
// 기존 request 객체 내 projectId, zone 은 deprecated
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
	//spew.Dump(resp)

	clusterInfoList := []*irs.ClusterInfo{}

	respClusters := resp.Clusters
	cblogger.Info(respClusters)
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
		// 				// nodeGroup의 Instance ID
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
			cblogger.Error(err) // 에러가 났어도 다음 항목 조회
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
	// 				// nodeGroup의 Instance ID
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

// 성공 실패여부만 return하는 경우는 Done까지 기다린 후 결과를 return
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
	spew.Dump(op)

	operationErr := WaitContainerOperationFail(ClusterHandler.ContainerClient, projectID, region, zone, op.Name, GCP_CONTAINER_OPERATION_DELETE_CLUSTER)
	if operationErr != nil {
		cblogger.Error(err)
		return false, err
	}

	return true, nil
}

// 객체 조회를 하는 것은 status 가 ing로 나타날 것이므로 operation 수행후 얼마간 실패로 떨어지는지 대기
// 실패하지 않으면 대기를 종료하고 조회시킴
func (ClusterHandler *GCPClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (irs.NodeGroupInfo, error) {
	nodeGroupInfo := irs.NodeGroupInfo{}

	projectID := ClusterHandler.Credential.ProjectID
	region := ClusterHandler.Region.Region
	zone := ClusterHandler.Region.Zone

	// cluster 조회
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
	//spew.Dump(nodeGroupReqInfo)
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
	diskSize, err := strconv.ParseInt(nodeGroupReqInfo.RootDiskSize, 10, 64)
	if err != nil {
		return nodeGroupInfo, err
	}
	nodeConfig.DiskSizeGb = diskSize
	nodeConfig.DiskType = nodeGroupReqInfo.RootDiskType
	nodeConfig.MachineType = nodeGroupReqInfo.VMSpecName
	nodeConfig.Tags = sgTags

	// if clusterInfo.Network.SecurityGroupIIDs != nil && len(clusterInfo.Network.SecurityGroupIIDs) > 0 {
	// 	var sgTags []string
	// 	for _, securityGroupIID := range clusterInfo.Network.SecurityGroupIIDs {
	// 		sgTags = append(sgTags, securityGroupIID.NameId)
	// 	}
	// 	nodeConfig.Tags = sgTags
	// }

	reqNodePool.Config = &nodeConfig

	cblogger.Info("GCP Cloud Driver: called AddNodeGroup()")
	hiscallInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterIID.NameId, "AddNodeGroup()")

	parent := getParentClusterAtContainer(projectID, zone, clusterIID.NameId)
	cblogger.Info("parent : ", parent)
	//spew.Dump(reqNodePool)

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
	//spew.Dump(op)

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
				// nodeGroup의 Instance ID
				nodeIID := irs.IID{NameId: instanceInfo.Name, SystemId: instanceInfo.Name}
				nodeList = append(nodeList, nodeIID)
			}
		}
	}
	cblogger.Info("nodeList: ", nodeList)
	nodeGroupInfo.Nodes = nodeList

	return nodeGroupInfo, nil

}

// autoScaling 에 대한 true/false 만 바꾼다.
func (ClusterHandler *GCPClusterHandler) SetNodeGroupAutoScaling(clusterIID irs.IID, nodeGroupIID irs.IID, on bool) (bool, error) {
	projectID := ClusterHandler.Credential.ProjectID
	region := ClusterHandler.Region.Region
	zone := ClusterHandler.Region.Zone

	parent := getParentNodePoolsAtContainer(projectID, zone, clusterIID.NameId, nodeGroupIID.NameId)

	rb := &container.SetNodePoolAutoscalingRequest{
		Autoscaling: &container.NodePoolAutoscaling{Enabled: on},
	}
	spew.Dump(rb)

	hiscallInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterIID.NameId, "SetNodeGroupAutoScaling()")
	start := call.Start()
	op, err := ClusterHandler.ContainerClient.Projects.Locations.Clusters.NodePools.SetAutoscaling(parent, rb).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to SetNodeGroupAutoScaling :  %v", err)
		cblogger.Error(err)
		return false, err
	}
	spew.Dump(op)

	operationErr := WaitContainerOperationDone(ClusterHandler.ContainerClient, projectID, region, zone, op.Name, GCP_CONTAINER_OPERATION_SET_NODE_POOL_MANAGEMENT, 30)
	if operationErr != nil {
		cblogger.Error(operationErr)
		return false, operationErr
	}

	return true, nil
}

// autoScaling에 대한 설정 값을 바꾼다.
// TODO : 현재 autoScaling 설정값을 조회해서 다르면 Set 해야하나
func (ClusterHandler *GCPClusterHandler) ChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID, desiredNodeSize int, minNodeSize int, maxNodeSize int) (irs.NodeGroupInfo, error) {
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

	// autoScaling의 min/max 변경
	orgAutoScaling := orgNodePool.Autoscaling
	hiscallInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterIID.NameId, "ChangeNodeGroupScaling()")

	// min, max의 변경일 때
	if intMaxNodeSize > 0 || intMinNodeSize > 0 {
		//기존 autoscaling이 false였으면 둘 다 값이 있어야 함.
		if orgAutoScaling == nil || orgAutoScaling.Enabled == false {
			if intMaxNodeSize == 0 {
				return irs.NodeGroupInfo{}, errors.New("Maximum Node size must be greater than zero")
			}
			if intMinNodeSize == 0 {
				return irs.NodeGroupInfo{}, errors.New("Minimum Node size must be greater than zero")
			}
			cblogger.Info("autoScaling : ", orgAutoScaling)
			orgAutoScaling = &container.NodePoolAutoscaling{}
			orgAutoScaling.Enabled = true
			orgAutoScaling.MaxNodeCount = intMaxNodeSize
			orgAutoScaling.MinNodeCount = intMinNodeSize
		} else {
			// autoscaling == true 일 때, min과 max는 기존값과 달라야 함. 다른것만 set
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
		spew.Dump(op)

		operationErr := WaitContainerOperationDone(ClusterHandler.ContainerClient, projectID, region, zone, op.Name, GCP_CONTAINER_OPERATION_SET_NODE_POOL_MANAGEMENT, 30)
		if operationErr != nil {
			cblogger.Error(operationErr)
			return irs.NodeGroupInfo{}, err
		}

	}
	// case1 : orgAutoScaling == false 면 true로 변경
	//			min, max 값 둘 다 필요

	// case2 : orgAutoScaling == true 면
	//			min, max 둘 중 하나만 변경해도 됨.

	// case3 : desire 변경이면
	//			기존값과 다르면 set. --> 다른 API임.

	// 1. autoscaling off -> on
	//    on, min, max 도 지정필요
	// 2. autoscaling on -> on. min, max change
	//	  기존 autoScaling 이 on 이어야 하고
	//	  min, max 둘 중 하나는 값이 달라야 함.
	// 3. initNodeCount change
	//	  기존 initNodeCount 와 달라야 함.

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

	// autoScaling의 desired node Count 변경
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
		spew.Dump(op2)

		operationErr2 := WaitContainerOperationDone(ClusterHandler.ContainerClient, projectID, region, zone, op2.Name, GCP_CONTAINER_OPERATION_SET_NODE_POOL_SIZE, 30)
		if operationErr2 != nil {
			cblogger.Error(operationErr2)
			return irs.NodeGroupInfo{}, err
		}
	}

	// 처리가 끝났으면 NodePool 조회
	nodePool, err := getNodePools(ClusterHandler.ContainerClient, projectID, ClusterHandler.Region, clusterIID, nodeGroupIID)
	if err != nil {
		return irs.NodeGroupInfo{}, err
	}
	return mappingNodeGroupInfo(nodePool)

}

// 성공 실패여부만 return하는 경우는 Done까지 기다린 후 결과를 return
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
	spew.Dump(op)

	operationErr := WaitContainerOperationDone(ClusterHandler.ContainerClient, projectID, region, zone, op.Name, GCP_CONTAINER_OPERATION_DELETE_NODE_POOL, 600)
	if operationErr != nil {
		cblogger.Error(err)
		return false, err
	}

	return true, nil
}

// cluster version upgrade
// 객체 조회를 하는 것은 status 가 ing로 나타날 것이므로 operation 수행후 얼마간 실패로 떨어지는지 대기
// 실패하지 않으면 대기를 종료하고 조회시킴
func (ClusterHandler *GCPClusterHandler) UpgradeCluster(clusterIID irs.IID, newVersion string) (irs.ClusterInfo, error) {
	clusterInfo := irs.ClusterInfo{}

	projectID := ClusterHandler.Credential.ProjectID
	region := ClusterHandler.Region.Region
	zone := ClusterHandler.Region.Zone

	cblogger.Info("GCP Cloud Driver: called UpgradeCluster()")
	hiscallInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, clusterIID.NameId, "UpgradeCluster()")

	parent := getParentClusterAtContainer(projectID, zone, clusterIID.NameId)
	rb := &container.UpdateMasterRequest{
		MasterVersion: newVersion,
	}

	start := call.Start()
	op, err := ClusterHandler.ContainerClient.Projects.Locations.Clusters.UpdateMaster(parent, rb).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to UpgradeCluster :  %v", err)
		cblogger.Error(err)
		return clusterInfo, err
	}
	spew.Dump(op)

	operationErr := WaitContainerOperationFail(ClusterHandler.ContainerClient, projectID, region, zone, op.Name, GCP_CONTAINER_OPERATION_UPDATE_CLUSTER)
	if operationErr != nil {
		cblogger.Error(err)
		return clusterInfo, err
	}

	return ClusterHandler.GetCluster(clusterIID)
}

// location은 region 또는 zone.
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

// container.Cluster에서 spider의 clusterInfo로 변경
// cluster의 nodePool에는 instanceGroupUrl 만 있어서 nodeGroup의 상세정보가 없음
// 추가로 nodepool을 조회해야 함.
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
	securityGroups := []irs.IID{} // SecurityGroup으로 정의된 Label추출
	var metaSecurityGroupTags []string
	for resourceKey, resourceVal := range cluster.ResourceLabels {
		if strings.HasPrefix(resourceKey, GCP_PMKS_SECURITYGROUP_TAG) {
			//securityGroups = append(securityGroups, irs.IID{NameId: resourceVal, SystemId: resourceVal})
			metaSecurityGroupTags = append(metaSecurityGroupTags, resourceVal)
		}
	}
	cblogger.Info("metaSecurityGroupTags : ", metaSecurityGroupTags)
	// NodeConfig의 Tag가 SecurityGroup으로 사용하는 Tag인지 알려면
	// Metadata에 Label이 정의되어있는지 여부로 확인
	// Create에서 Metadata 와 nodeConfig의 Tag가 같은값이 들어가는데 굳이 한번 더 체크할 필요가 있을까?
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

			//nodeGroupInfo.Nodes : 별도의 API 호출필요
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

	// spew.Dump(clusterInfo)

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

	keyValueList := []irs.KeyValue{}
	if nodePool.InstanceGroupUrls != nil {
		for idx, instanceGroupUrl := range nodePool.InstanceGroupUrls {
			urlArr := strings.Split(instanceGroupUrl, "/")
			instanceGroupName := urlArr[len(urlArr)-1]
			keyValueList = append(keyValueList, irs.KeyValue{Key: "InstanceGroup_" + strconv.Itoa(idx), Value: instanceGroupName})

			// InstanceGroup.ListInstances
			//"instance": "https://www.googleapis.com/compute/v1/projects/csta-349809/zones/asia-northeast1-a/instances/gke-spider-cluster-03-ce-default-pool-3082fa6f-kvks",
		}

	}
	nodeGroupInfo.KeyValueList = keyValueList

	return nodeGroupInfo, nil
}

// Cluster의 상태
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

// NodeGroup의 상태
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
// TODO : nodePool정보 조회하여 변경할 것만 전송할까? getNodePools(containerClient *container.Service, projectID string, region string, zone string, clusterIID irs.IID, nodeGroupIID irs.IID) (*container.NodePool, error)
// 필요한 parameter만 set 해서 org와 다른것들을 update
// func updateNodeGroupAutoScaling(containerClient *container.Service, projectID string, region string, zone string, clusterIID irs.IID, orgNodeGroupReqInfo irs.NodeGroupInfo, nodeGroupReqInfo irs.NodeGroupInfo, autoscalingType string) (bool, error) {
// 	reqNodePool := container.NodePool{}
// 	autoScaling := container.NodePoolAutoscaling{}

// 	parent := getParentNodePoolsAtContainer(projectID, zone, clusterIID.NameId, nodeGroupReqInfo.IId.NameId)
// 	spew.Dump(nodeGroupReqInfo)
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
// 	spew.Dump(op)

// 	operationErr := WaitContainerOperationDone(containerClient, projectID, region, zone, op.Name, 3, 1200)
// 	if operationErr != nil {
// 		cblogger.Error(operationErr)
// 		return false, operationErr
// 	}

// 	return true, nil
// }

// NodePool 조회
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

// clusterInfo 로 Set
// cluster에는 NodeGroup의 link정보만 있어서 NodeGroup정보를 추가로 조회해야 함.
func convertCluster(client *compute.Service, credential idrv.CredentialInfo, region idrv.RegionInfo, cluster *container.Cluster) (irs.ClusterInfo, error) {
	clusterInfo, err := mappingClusterInfo(cluster)
	if err != nil {
		// cluster err
		// return irs.ClusterInfo{}, err
	}

	// mappingClusterInfo에서 우선 nodeGroup 정보가 set 됨. nodeGroup.KeyValueList에 instanceGroupID가 들어있음.
	cblogger.Info("nodeGroupList ", clusterInfo.NodeGroupList)
	//nodePools = resp.NodePools
	nodeGroupList, err := convertNodeGroup(client, credential, region, clusterInfo.NodeGroupList)
	if err != err { // 오류가 나도 clusterInfo를 넘김
		// failed to get nodeGroupInfo
		cblogger.Error(err)
	}

	clusterInfo.NodeGroupList = nodeGroupList
	return clusterInfo, nil
}

// nodeGroupInfo로 set
// nodeGroup 정보에서 Node
func convertNodeGroup(client *compute.Service, credential idrv.CredentialInfo, region idrv.RegionInfo, orgNodeGroupList []irs.NodeGroupInfo) ([]irs.NodeGroupInfo, error) {
	nodeGroupList := []irs.NodeGroupInfo{}
	for _, nodeGroupInfo := range orgNodeGroupList {
		cblogger.Info("convertNodeGroup ", nodeGroupInfo)
		//spew.Dump(nodeGroupInfo)
		keyValueList := nodeGroupInfo.KeyValueList
		for _, keyValue := range keyValueList {
			cblogger.Info("keyValue ", keyValue)
			if strings.HasPrefix(keyValue.Key, GCP_PMKS_INSTANCEGROUP_KEY) {
				cblogger.Info("HasPrefix ")
				nodeList := []irs.IID{}
				instanceGroupValue := keyValue.Value
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
					//spew.Dump(instanceInfo)
					// nodeGroup의 Instance ID
					nodeIID := irs.IID{NameId: instanceInfo.Name, SystemId: instanceInfo.Name}
					nodeList = append(nodeList, nodeIID)

					cblogger.Info("instanceInfo.Labels ", instanceInfo.Labels)
					nodeGroupInfo.KeyPairIID = irs.IID{NameId: "NameId", SystemId: "SystemId"} // empty면 오류나므로 기본값으로 설정후 update하도록
					if instanceInfo.Labels != nil {
						keyPairVal, exists := instanceInfo.Labels[GCP_PMKS_KEYPAIR_KEY]
						if exists {
							cblogger.Info("nodeGroup set keypair ", keyPairVal)
							nodeGroupInfo.KeyPairIID = irs.IID{NameId: keyPairVal, SystemId: keyPairVal}
						}
					}
				}
				cblogger.Info("nodeList ", nodeList)
				nodeGroupInfo.Nodes = nodeList
				//cblogger.Info("nodeGroupInfo ", nodeGroupInfo)
				spew.Dump(nodeGroupInfo)
			}
		}
		nodeGroupList = append(nodeGroupList, nodeGroupInfo)
	}
	return nodeGroupList, nil
}
