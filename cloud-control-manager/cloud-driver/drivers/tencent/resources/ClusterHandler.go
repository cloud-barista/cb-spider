// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2022.08.

package resources

import (
	"sync"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	//"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/tencent/utils/Tencent"

	"github.com/sirupsen/logrus"
	// call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
)

// tempCalllogger
// 공통로거 만들기 이전까지 사용
var once sync.Once
var tempCalllogger *logrus.Logger

func init() {
	once.Do(func() {
		tempCalllogger = call.GetLogger("HISCALL")
	})
}

type TencentClusterHandler struct {
	RegionInfo     idrv.RegionInfo
	CredentialInfo idrv.CredentialInfo
}

// connectionInfo.CredentialInfo.AccessKey
// connectionInfo.CredentialInfo.AccessSecret
// connectionInfo.RegionInfo.Region = "region-1"

func (clusterHandler *TencentClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (irs.ClusterInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called CreateCluster()")
	// callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, "CreateCluster()", "CreateCluster()")

	// // 클러스터 생성 요청을 JSON 요청으로 변환
	// payload, err := getClusterInfoJSON(clusterReqInfo, clusterHandler.RegionInfo.Region)
	// if err != nil {
	// 	cblogger.Error(err)
	// 	return irs.ClusterInfo{}, err
	// }

	// start := call.Start()
	// response_json_str, err := tencent.CreateCluster(clusterHandler.CredentialInfo.AccessKey, clusterHandler.CredentialInfo.AccessSecret, clusterHandler.RegionInfo.Region, payload)
	// loggingInfo(callLogInfo, start)
	// if err != nil {
	// 	cblogger.Error(err)
	// 	loggingError(callLogInfo, err)
	// 	return irs.ClusterInfo{}, err
	// }

	// println(response_json_str)
	// // {"cluster_id":"c913aebba53eb40f3978495d92b8da57f","request_id":"2C0836DA-ED3B-5B1E-94C9-5B7E355E2E44","task_id":"T-63185224055a0b07c6000083","instanceId":"c913aebba53eb40f3978495d92b8da57f"}

	// var response_json_obj map[string]interface{}
	// json.Unmarshal([]byte(response_json_str), &response_json_obj)
	// cluster_id := response_json_obj["cluster_id"].(string)
	// cluster_info, err := getClusterInfo(clusterHandler.CredentialInfo.AccessKey, clusterHandler.CredentialInfo.AccessSecret, clusterHandler.RegionInfo.Region, cluster_id)
	// if err != nil {
	// 	return irs.ClusterInfo{}, err
	// }

	// // 리턴할 ClusterInfo 만들기
	// // 일단은 단순하게 만들어서 반환한다.
	// // 추후에 정보 추가 필요

	// // NodeGroup 생성 정보가 있는경우 생성을 시도한다.
	// // 문제는 Cluster 생성이 완료되어야 NodeGroup 생성이 가능하다.
	// // Cluster 생성이 완료되려면 최소 10분 이상 걸린다.
	// // 성공할때까지 반복하면서 생성을 시도해야 하는가?
	// for _, node_group := range clusterReqInfo.NodeGroupList {
	// 	res, err := clusterHandler.AddNodeGroup(clusterReqInfo.IId, node_group)
	// 	if err != nil {
	// 		cblogger.Error(err)
	// 		return irs.ClusterInfo{}, err
	// 	}
	// 	printFlattenJSON(res)
	// }
	// return *cluster_info, nil

	return irs.ClusterInfo{}, nil
}

func (clusterHandler *TencentClusterHandler) ListCluster() ([]*irs.ClusterInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called ListCluster()")
	// callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, "ListCluster()", "ListCluster()")

	// start := call.Start()
	// clusters_json_str, err := tencent.GetClusters(clusterHandler.CredentialInfo.AccessKey, clusterHandler.CredentialInfo.AccessSecret, clusterHandler.RegionInfo.Region)
	// loggingInfo(callLogInfo, start)
	// if err != nil {
	// 	return nil, err
	// }

	// var clusters_json_obj map[string]interface{}
	// json.Unmarshal([]byte(clusters_json_str), &clusters_json_obj)
	// clusters := clusters_json_obj["clusters"].([]interface{})
	// cluster_info_list := make([]*irs.ClusterInfo, len(clusters))
	// for i, cluster := range clusters {
	// 	println(i, cluster)
	// 	cluster_id := cluster.(map[string]interface{})["cluster_id"].(string)
	// 	cluster_info_list[i], err = getClusterInfo(clusterHandler.CredentialInfo.AccessKey, clusterHandler.CredentialInfo.AccessSecret, clusterHandler.RegionInfo.Region, cluster_id)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	// return cluster_info_list, nil

	return nil, nil
}

func (clusterHandler *TencentClusterHandler) GetCluster(clusterIID irs.IID) (irs.ClusterInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called GetCluster()")
	// callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "GetCluster()")

	// start := call.Start()
	// cluster_info, err := getClusterInfo(clusterHandler.CredentialInfo.AccessKey, clusterHandler.CredentialInfo.AccessSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId)
	// loggingInfo(callLogInfo, start)
	// if err != nil {
	// 	return irs.ClusterInfo{}, err
	// }

	// return *cluster_info, nil

	return irs.ClusterInfo{}, nil
}

func (clusterHandler *TencentClusterHandler) DeleteCluster(clusterIID irs.IID) (bool, error) {
	cblogger.Info("Tencent Cloud Driver: called DeleteCluster()")
	// callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "DeleteCluster()")

	// start := call.Start()
	// res, err := tencent.DeleteCluster(clusterHandler.CredentialInfo.AccessKey, clusterHandler.CredentialInfo.AccessSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId)
	// loggingInfo(callLogInfo, start)
	// if err != nil {
	// 	return false, err
	// }
	// println(res)

	// return true, nil

	return true, nil
}

func (clusterHandler *TencentClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (irs.NodeGroupInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called AddNodeGroup()")

	// callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "AddNodeGroup()")

	// // 노드 그룹 생성 요청을 JSON 요청으로 변환
	// payload, err := getNodeGroupJSONString(nodeGroupReqInfo)
	// if err != nil {
	// 	return irs.NodeGroupInfo{}, err
	// }

	// start := call.Start()
	// result_json_str, err := tencent.CreateNodeGroup(clusterHandler.CredentialInfo.AccessKey, clusterHandler.CredentialInfo.AccessSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, payload)
	// loggingInfo(callLogInfo, start)
	// if err != nil {
	// 	return irs.NodeGroupInfo{}, err
	// }

	// var result_json_obj map[string]interface{}
	// json.Unmarshal([]byte(result_json_str), &result_json_obj)
	// printFlattenJSON(result_json_obj)
	// //{"nodepool_id":"np031dc18d09ee4959a2c6444570150c89","request_id":"BF1C50C9-E1C0-5DB1-B290-EC01B6F1BFD1","task_id":"T-63198517f47545090c000376"}
	// nodepool_id := result_json_obj["nodepool_id"].(string)
	// node_group_info, err := getNodeGroupInfo(clusterHandler.CredentialInfo.AccessKey, clusterHandler.CredentialInfo.AccessSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodepool_id)
	// if err != nil {
	// 	return irs.NodeGroupInfo{}, err
	// }

	// return *node_group_info, nil

	return irs.NodeGroupInfo{}, nil
}

func (clusterHandler *TencentClusterHandler) ListNodeGroup(clusterIID irs.IID) ([]*irs.NodeGroupInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called ListNodeGroup()")
	// callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "ListNodeGroup()")

	// node_group_info_list := []*irs.NodeGroupInfo{}

	// start := call.Start()
	// node_groups_json_str, err := tencent.ListNodeGroup(clusterHandler.CredentialInfo.AccessKey, clusterHandler.CredentialInfo.AccessSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId)
	// loggingInfo(callLogInfo, start)
	// if err != nil {
	// 	return node_group_info_list, err
	// }

	// var node_groups_json_obj map[string]interface{}
	// json.Unmarshal([]byte(node_groups_json_str), &node_groups_json_obj)
	// node_groups := node_groups_json_obj["nodepools"].([]interface{})
	// for _, node_group := range node_groups {
	// 	node_group_id := node_group.(map[string]interface{})["nodepool_info"].(map[string]interface{})["nodepool_id"].(string)
	// 	node_group_info, err := getNodeGroupInfo(clusterHandler.CredentialInfo.AccessKey, clusterHandler.CredentialInfo.AccessSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, node_group_id)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	node_group_info_list = append(node_group_info_list, node_group_info)
	// }

	// // var node_groups_json_obj map[string]interface{}
	// // json.Unmarshal([]byte(node_groups_json_str), &node_groups_json_obj)
	// // node_groups := node_groups_json_obj["nodegroups"].([]interface{})
	// // for _, node_group := range node_groups {
	// // 	nodepool_id := node_group.(map[string]interface{})["nodepool_id"].(string)
	// // 	node_group_info, err := getNodeGroupInfo(clusterHandler.CredentialInfo.AccessKey, clusterHandler.CredentialInfo.AccessSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodepool_id)
	// // 	if err != nil {
	// // 		return node_group_info_list, err
	// // 	}
	// // 	node_group_info_list = append(node_group_info_list, node_group_info)
	// // }

	// return node_group_info_list, nil

	return nil, nil
}

func (clusterHandler *TencentClusterHandler) GetNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (irs.NodeGroupInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called GetNodeGroup()")
	// callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "GetNodeGroup()")

	// start := call.Start()
	// temp, err := getNodeGroupInfo(clusterHandler.CredentialInfo.AccessKey, clusterHandler.CredentialInfo.AccessSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodeGroupIID.SystemId)
	// loggingInfo(callLogInfo, start)
	// if err != nil {
	// 	return irs.NodeGroupInfo{}, err
	// }

	// return *temp, nil

	return irs.NodeGroupInfo{}, nil
}

func (clusterHandler *TencentClusterHandler) SetNodeGroupAutoScaling(clusterIID irs.IID, nodeGroupIID irs.IID, on bool) (bool, error) {
	cblogger.Info("Tencent Cloud Driver: called SetNodeGroupAutoScaling()")

	// temp := `{"auto_scaling":{"enable":%t}}`
	// body := fmt.Sprintf(temp, on)

	// res, err := tencent.ModifyNodeGroup(clusterHandler.CredentialInfo.AccessKey, clusterHandler.CredentialInfo.AccessSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodeGroupIID.SystemId, body)
	// if err != nil {
	// 	return false, err
	// }
	// println(res)

	// return true, nil
	return false, nil
}

func (clusterHandler *TencentClusterHandler) ChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID, desiredNodeSize int, minNodeSize int, maxNodeSize int) (irs.NodeGroupInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called ChangeNodeGroupScaling()")

	// // temp := `{"auto_scaling":{"max_instances":%d,"min_instances":%d},"scaling_group":{"desired_size":%d}}`
	// // desired_size is not supported in Tencent with auto scaling mode
	// temp := `{"auto_scaling":{"max_instances":%d,"min_instances":%d}}`
	// body := fmt.Sprintf(temp, maxNodeSize, minNodeSize)
	// res, err := tencent.ModifyNodeGroup(clusterHandler.CredentialInfo.AccessKey, clusterHandler.CredentialInfo.AccessSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodeGroupIID.SystemId, body)
	// if err != nil {
	// 	return irs.NodeGroupInfo{}, err
	// }
	// println(res)

	// node_group_info, err := getNodeGroupInfo(clusterHandler.CredentialInfo.AccessKey, clusterHandler.CredentialInfo.AccessSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodeGroupIID.SystemId)
	// if err != nil {
	// 	return irs.NodeGroupInfo{}, err
	// }

	// return *node_group_info, nil

	return irs.NodeGroupInfo{}, nil
}

func (clusterHandler *TencentClusterHandler) RemoveNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (bool, error) {
	cblogger.Info("Tencent Cloud Driver: called RemoveNodeGroup()")
	// callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "RemoveNodeGroup()")

	// start := call.Start()
	// res, err := tencent.DeleteNodeGroup(clusterHandler.CredentialInfo.AccessKey, clusterHandler.CredentialInfo.AccessSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodeGroupIID.SystemId)
	// loggingInfo(callLogInfo, start)
	// if err != nil {
	// 	return false, err
	// }
	// println(res)

	// return true, nil
	return false, nil
}

func (clusterHandler *TencentClusterHandler) UpgradeCluster(clusterIID irs.IID, newVersion string) (irs.ClusterInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called UpgradeCluster()")
	// //callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "UpgradeCluster()")

	// temp := `{"next_version" : "%s"}`
	// body := fmt.Sprintf(temp, newVersion)

	// res, err := tencent.UpgradeCluster(clusterHandler.CredentialInfo.AccessKey, clusterHandler.CredentialInfo.AccessSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, body)
	// if err != nil {
	// 	return irs.ClusterInfo{}, err
	// }
	// println(res)

	// clusterInfo, err := getClusterInfo(clusterHandler.CredentialInfo.AccessKey, clusterHandler.CredentialInfo.AccessSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId)
	// if err != nil {
	// 	return irs.ClusterInfo{}, err
	// }

	// return *clusterInfo, nil

	return irs.ClusterInfo{}, nil
}

// func getClusterInfo(access_key string, access_secret string, region_id string, cluster_id string) (*irs.ClusterInfo, error) {
// 	defer func() {
// 		if r := recover(); r != nil {
// 			cblogger.Error("getClusterInfo() failed!", r)
// 		}
// 	}()

// 	cluster_json_str, err := tencent.GetCluster(access_key, access_secret, region_id, cluster_id)
// 	if err != nil {
// 		return nil, err
// 	}
// 	println(cluster_json_str)
// 	flat, err := flatten.FlattenString(cluster_json_str, "", flatten.DotStyle)
// 	if err != nil {
// 		return nil, err
// 	}
// 	println(flat)

// 	// k,v 추출
// 	// k,v 변환 규칙 작성 [k,v]:[ClusterInfo.k, ClusterInfo.v]
// 	// 변환 규칙에 따라 k,v 변환
// 	var cluster_json_obj map[string]interface{}
// 	json.Unmarshal([]byte(cluster_json_str), &cluster_json_obj)

// 	// https://www.Tencentcloud.com/help/doc-detail/86987.html
// 	// Initializing	Creating the cloud resources that are used by the cluster.
// 	// Creation Failed	Failed to create the cloud resources that are used by the cluster.
// 	// Running	The cloud resources used by the cluster are created.
// 	// Updating	Updating the metadata of the cluster.
// 	// Scaling	Adding nodes to the cluster.
// 	// Removing	Removing nodes from the cluster.
// 	// Upgrading	Upgrading the cluster.
// 	// Draining	Evicting pods from a node to other nodes. After all pods are evicted from the node, the node becomes unschudulable.
// 	// Deleting	Deleting the cluster.
// 	// Deletion Failed	Failed to delete the cluster.
// 	// Deleted (invisible to users)	The cluster is deleted.

// 	// ClusterCreating ClusterStatus = "Creating"
// 	// ClusterActive   ClusterStatus = "Active"
// 	// ClusterInactive ClusterStatus = "Inactive"
// 	// ClusterUpdating ClusterStatus = "Updating"
// 	// ClusterDeleting ClusterStatus = "Deleting"

// 	health_status := cluster_json_obj["state"].(string)
// 	cluster_status := irs.ClusterActive
// 	if strings.EqualFold(health_status, "Initializing") {
// 		cluster_status = irs.ClusterCreating
// 	} else if strings.EqualFold(health_status, "Updating") {
// 		cluster_status = irs.ClusterUpdating
// 	} else if strings.EqualFold(health_status, "Creation Failed") {
// 		cluster_status = irs.ClusterInactive
// 	} else if strings.EqualFold(health_status, "Deleting") {
// 		cluster_status = irs.ClusterDeleting
// 	} else if strings.EqualFold(health_status, "Running") {
// 		cluster_status = irs.ClusterActive
// 	}

// 	println(cluster_status)

// 	created_at := cluster_json_obj["created"].(string) // 2022-09-08T09:02:16+08:00,
// 	datetime, err := time.Parse(time.RFC3339, created_at)
// 	if err != nil {
// 		panic(err)
// 	}

// 	// name
// 	// cluster_id
// 	// current_version
// 	// security_group_id
// 	// vpc_id
// 	// state
// 	// created
// 	cluster_info := &irs.ClusterInfo{
// 		IId: irs.IID{
// 			NameId:   cluster_json_obj["name"].(string),
// 			SystemId: cluster_json_obj["cluster_id"].(string),
// 		},
// 		Version: cluster_json_obj["current_version"].(string),
// 		Network: irs.NetworkInfo{
// 			VpcIID: irs.IID{
// 				NameId:   "",
// 				SystemId: cluster_json_obj["vpc_id"].(string),
// 			},
// 			SecurityGroupIIDs: []irs.IID{
// 				{
// 					NameId:   "",
// 					SystemId: cluster_json_obj["security_group_id"].(string),
// 				},
// 			},
// 		},
// 		Status:      cluster_status,
// 		CreatedTime: datetime,
// 		// KeyValueList: []irs.KeyValue{}, // flatten data 입력하기
// 	}
// 	println(cluster_info)

// 	// NodeGroups
// 	node_groups_json_str, err := tencent.ListNodeGroup(access_key, access_secret, region_id, cluster_id)
// 	if err != nil {
// 		return nil, err
// 	}
// 	print(node_groups_json_str)
// 	// {"NextToken":"","TotalCount":0,"nodepools":[],"request_id":"4529A823-F344-5EA6-8E60-47FC30117668"}

// 	// k,v 추출
// 	// k,v 변환 규칙 작성 [k,v]:[NodeGroup.k, NodeGroup.v]
// 	// 변환 규칙에 따라 k,v 변환
// 	flat, err = flatten.FlattenString(node_groups_json_str, "", flatten.DotStyle)
// 	if err != nil {
// 		return nil, err
// 	}
// 	println(flat)

// 	var node_groups_json_obj map[string]interface{}
// 	json.Unmarshal([]byte(node_groups_json_str), &node_groups_json_obj)
// 	node_groups := node_groups_json_obj["nodepools"].([]interface{})
// 	for _, node_group := range node_groups {
// 		// printFlattenJSON(node_group)
// 		// "nodepool_info.nodepool_id": "np02b049a03b8141858697497e12a61aa1",
// 		node_group_id := node_group.(map[string]interface{})["nodepool_info"].(map[string]interface{})["nodepool_id"].(string)
// 		node_group_info, err := getNodeGroupInfo(access_key, access_secret, region_id, cluster_id, node_group_id)
// 		if err != nil {
// 			return nil, err
// 		}
// 		cluster_info.NodeGroupList = append(cluster_info.NodeGroupList, *node_group_info)
// 	}

// 	return cluster_info, nil
// }

// func printFlattenJSON(json_obj interface{}) {
// 	temp, err := json.MarshalIndent(json_obj, "", "  ")
// 	if err != nil {
// 		println(err)
// 	} else {
// 		flat, err := flatten.FlattenString(string(temp), "", flatten.DotStyle)
// 		if err != nil {
// 			println(err)
// 		} else {
// 			println(flat)
// 		}
// 	}
// }

// func getNodeGroupInfo(access_key, access_secret, region_id, cluster_id, node_group_id string) (*irs.NodeGroupInfo, error) {
// 	defer func() {
// 		if r := recover(); r != nil {
// 			cblogger.Error("getNodeGroupInfo() failed!", r)
// 		}
// 	}()

// 	node_group_json_str, err := tencent.GetNodeGroup(access_key, access_secret, region_id, cluster_id, node_group_id)
// 	if err != nil {
// 		return nil, err
// 	}

// 	var node_group_json_obj map[string]interface{}
// 	json.Unmarshal([]byte(node_group_json_str), &node_group_json_obj)

// 	// mapping
// 	// NodeGroupList.0.IId.NameId: 			nodepool_info.name
// 	// NodeGroupList.0.IId.SystemId: 		nodepool_info.nodepool_id
// 	// NodeGroupList.0.ImageIID.NameId: 	scaling_group.image_type
// 	// NodeGroupList.0.ImageIID.SystemId: 	scaling_group.image_id
// 	// NodeGroupList.0.VMSpecName: 			scaling_group.instance_types.0
// 	// NodeGroupList.0.RootDiskType: 		scaling_group.system_disk_category
// 	// NodeGroupList.0.RootDiskSize: 		scaling_group.system_disk_size
// 	// NodeGroupList.0.KeyPairIID.NameId: 	scaling_group.key_pair
// 	// NodeGroupList.0.KeyPairIID.SystemId: ""
// 	// NodeGroupList.0.Status: 				status.state
// 	// NodeGroupList.0.OnAutoScaling: 		auto_scaling.enable
// 	// NodeGroupList.0.DesiredNodeSize: 	n/a
// 	// NodeGroupList.0.MaxNodeSize: 		auto_scaling.max_instances
// 	// NodeGroupList.0.MinNodeSize: 		auto_scaling.min_instances
// 	// NodeGroupList.0.NodeList.0.NameId: 	//node정보추가 // not yet
// 	// NodeGroupList.0.NodeList.0.SystemId:

// 	// NodeGroupList.0.KeyValueList.0.Key: key,	//keyvalue 정보 추가
// 	// NodeGroupList.0.KeyValueList.0.Value: value

// 	// nodegroup.state
// 	// https://www.Tencentcloud.com/help/en/container-service-for-kubernetes/latest/query-the-details-of-a-node-pool
// 	// active: The node pool is active.
// 	// scaling: The node pool is being scaled.
// 	// removing: Nodes are being removed from the node pool.
// 	// deleting: The node pool is being deleted.
// 	// updating: The node pool is being updated.
// 	health_status := node_group_json_obj["status"].(map[string]interface{})["state"].(string)
// 	status := irs.NodeGroupActive
// 	if strings.EqualFold(health_status, "active") {
// 		status = irs.NodeGroupActive
// 	} else if strings.EqualFold(health_status, "scaling") {
// 		status = irs.NodeGroupUpdating
// 	} else if strings.EqualFold(health_status, "removing") {
// 		status = irs.NodeGroupUpdating // removing is a kind of updating?
// 	} else if strings.EqualFold(health_status, "deleting") {
// 		status = irs.NodeGroupDeleting
// 	} else if strings.EqualFold(health_status, "updating") {
// 		status = irs.NodeGroupUpdating
// 	}

// 	println(status)

// 	// 변환 자동화 고려
// 	// 변환 규칙 이용 고려 // https://github.com/qntfy/kazaam
// 	// https://github.com/antchfx/jsonquery
// 	node_group_info := irs.NodeGroupInfo{
// 		IId: irs.IID{
// 			NameId:   node_group_json_obj["nodepool_info"].(map[string]interface{})["name"].(string),
// 			SystemId: node_group_json_obj["nodepool_info"].(map[string]interface{})["nodepool_id"].(string),
// 		},
// 		ImageIID: irs.IID{
// 			NameId:   node_group_json_obj["scaling_group"].(map[string]interface{})["image_type"].(string),
// 			SystemId: node_group_json_obj["scaling_group"].(map[string]interface{})["image_id"].(string),
// 		},
// 		VMSpecName:   node_group_json_obj["scaling_group"].(map[string]interface{})["instance_types"].([]interface{})[0].(string),
// 		RootDiskType: node_group_json_obj["scaling_group"].(map[string]interface{})["system_disk_category"].(string),
// 		RootDiskSize: strconv.Itoa(int(node_group_json_obj["scaling_group"].(map[string]interface{})["system_disk_size"].(float64))),
// 		KeyPairIID: irs.IID{
// 			NameId:   node_group_json_obj["scaling_group"].(map[string]interface{})["key_pair"].(string),
// 			SystemId: "",
// 		},
// 		Status:          status,
// 		OnAutoScaling:   node_group_json_obj["auto_scaling"].(map[string]interface{})["enable"].(bool),
// 		MinNodeSize:     int(node_group_json_obj["auto_scaling"].(map[string]interface{})["min_instances"].(float64)),
// 		MaxNodeSize:     int(node_group_json_obj["auto_scaling"].(map[string]interface{})["max_instances"].(float64)),
// 		DesiredNodeSize: 0,                // not supported in Tencent
// 		NodeList:        []irs.IID{},      // to be implemented
// 		KeyValueList:    []irs.KeyValue{}, // to be implemented
// 	}

// 	return &node_group_info, nil
// }

// func getClusterInfoJSON(clusterInfo irs.ClusterInfo, region_id string) (string, error) {

// 	defer func() {
// 		if r := recover(); r != nil {
// 			cblogger.Error("getClusterInfoJSON failed", r)
// 		}
// 	}()

// 	// clusterInfo := irs.ClusterInfo{
// 	// 	IId: irs.IID{
// 	// 		NameId:   "cluster-x",
// 	// 		SystemId: "",
// 	// 	},
// 	// 	Version: "1.22.10-aliyun.1",
// 	// 	Network: irs.NetworkInfo{
// 	// 		VpcIID: irs.IID{NameId: "", SystemId: "vpc-2zek5slojo5bh621ftnrg"},
// 	// 	},
// 	// 	KeyValueList: []irs.KeyValue{
// 	// 		{
// 	// 			Key:   "container_cidr",
// 	// 			Value: "172.31.0.0/16",
// 	// 		},
// 	// 		{
// 	// 			Key:   "service_cidr",
// 	// 			Value: "172.32.0.0/16",
// 	// 		},
// 	// 		{
// 	// 			Key:   "master_vswitch_id",
// 	// 			Value: "vsw-2ze0qpwcio7r5bx3nqbp1",
// 	// 		},
// 	// 	},
// 	// }

// 	//cidr: Valid values: 10.0.0.0/16-24, 172.16-31.0.0/16-24, and 192.168.0.0/16-24.
// 	container_cidr := ""
// 	service_cidr := ""
// 	master_vswitch_id := ""

// 	for _, v := range clusterInfo.KeyValueList {
// 		switch v.Key {
// 		case "container_cidr":
// 			container_cidr = v.Value
// 		case "service_cidr":
// 			service_cidr = v.Value
// 		case "master_vswitch_id":
// 			master_vswitch_id = v.Value
// 		}
// 	}

// 	temp := `{
// 		"name": "%s",
// 		"region_id": "%s",
// 		"cluster_type": "ManagedKubernetes",
// 		"kubernetes_version": "1.22.10-aliyun.1",
// 		"vpcid": "%s",
// 		"container_cidr": "%s",
// 		"service_cidr": "%s",
// 		"num_of_nodes": 0,
// 		"master_vswitch_ids": [
// 			"%s"
// 		]
// 	}`

// 	clusterInfoJSON := fmt.Sprintf(temp, clusterInfo.IId.NameId, region_id, clusterInfo.Network.VpcIID.SystemId, container_cidr, service_cidr, master_vswitch_id)

// 	return clusterInfoJSON, nil
// }

// func getNodeGroupJSONString(nodeGroupReqInfo irs.NodeGroupInfo) (string, error) {

// 	defer func() {
// 		if r := recover(); r != nil {
// 			cblogger.Error("getNodeGroupJSONString failed", r)
// 		}
// 	}()

// 	// new_node_group := &irs.NodeGroupInfo{
// 	// 	IId:             irs.IID{NameId: "nodepoolx100", SystemId: ""},
// 	// 	ImageIID:        irs.IID{NameId: "", SystemId: "image_id"}, // 이미지 id 선택 추가
// 	// 	VMSpecName:      "ecs.c6.xlarge",
// 	// 	RootDiskType:    "cloud_essd",
// 	// 	RootDiskSize:    "70",
// 	// 	KeyPairIID:      irs.IID{NameId: "kp1", SystemId: ""},
// 	// 	OnAutoScaling:   true,
// 	// 	DesiredNodeSize: 1,
// 	// 	MinNodeSize:     0,
// 	// 	MaxNodeSize:     3,
// 	// 	// KeyValueList: []irs.KeyValue{ // 클러스터 조회해서 처리한다. // //vswitch_id":"vsw-2ze0qpwcio7r5bx3nqbp1"
// 	// 	// 	{
// 	// 	// 		Key:   "vswitch_ids",
// 	// 	// 		Value: "vsw-2ze0qpwcio7r5bx3nqbp1",
// 	// 	// 	},
// 	// 	// },
// 	// }

// 	name := nodeGroupReqInfo.IId.NameId
// 	//image_id := nodeGroupReqInfo.ImageIID.SystemId
// 	enable := nodeGroupReqInfo.OnAutoScaling
// 	max_instances := nodeGroupReqInfo.MaxNodeSize
// 	min_instances := nodeGroupReqInfo.MinNodeSize
// 	// desired_instances := nodeGroupReqInfo.DesiredNodeSize // not supported in Tencent

// 	instance_type := nodeGroupReqInfo.VMSpecName
// 	key_pair := nodeGroupReqInfo.KeyPairIID.NameId

// 	system_disk_category := nodeGroupReqInfo.RootDiskType
// 	system_disk_size, _ := strconv.ParseInt(nodeGroupReqInfo.RootDiskSize, 10, 32)

// 	vswitch_id := "vsw-2ze0qpwcio7r5bx3nqbp1" // get vswitch_id, get from cluster info

// 	temp := `{
// 		"nodepool_info": {
// 			"name": "%s"
// 		},
// 		"auto_scaling": {
// 			"enable": %t,
// 			"max_instances": %d,
// 			"min_instances": %d
// 		},
// 		"scaling_group": {
// 			"instance_types": ["%s"],
// 			"key_pair": "%s",
// 			"system_disk_category": "%s",
// 			"system_disk_size": %d,
// 			"vswitch_ids": ["%s"]
// 		},
// 		"management": {
// 			"enable":true
// 		}
// 	}`

// 	payload := fmt.Sprintf(temp, name, enable, max_instances, min_instances, instance_type, key_pair, system_disk_category, system_disk_size, vswitch_id)

// 	return payload, nil
// }

// // getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, "ListCluster()", "ListCluster()")
// func getCallLogScheme(region string, resourceType call.RES_TYPE, resourceName string, apiName string) call.CLOUDLOGSCHEMA {
// 	cblogger.Info(fmt.Sprintf("Call %s %s", call.TENCENT, apiName))
// 	return call.CLOUDLOGSCHEMA{
// 		CloudOS:      call.TENCENT,
// 		RegionZone:   region,
// 		ResourceType: resourceType,
// 		ResourceName: resourceName,
// 		CloudOSAPI:   apiName,
// 	}
// }

// func loggingError(hiscallInfo call.CLOUDLOGSCHEMA, err error) {
// 	hiscallInfo.ErrorMSG = err.Error()
// 	tempCalllogger.Info(call.String(hiscallInfo))
// }

// func loggingInfo(hiscallInfo call.CLOUDLOGSCHEMA, start time.Time) {
// 	hiscallInfo.ElapsedTime = call.Elapsed(start)
// 	tempCalllogger.Info(call.String(hiscallInfo))
// }
