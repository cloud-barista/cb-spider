// Alibaba Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Alibaba Driver.
//
// by CB-Spider Team, 2022.09.

package resources

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/alibaba/utils/alibaba"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/jeremywohl/flatten"
	"github.com/sirupsen/logrus"
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

type AlibabaClusterHandler struct {
	RegionInfo     idrv.RegionInfo
	CredentialInfo idrv.CredentialInfo
}

func (clusterHandler *AlibabaClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (irs.ClusterInfo, error) {
	cblogger.Info("Alibaba Cloud Driver: called CreateCluster()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, "CreateCluster()", "CreateCluster()")

	// 클러스터 생성 요청을 JSON 요청으로 변환
	payload, err := getClusterInfoJSON(clusterHandler, clusterReqInfo)
	if err != nil {
		cblogger.Error(err)
		return irs.ClusterInfo{}, err
	}

	start := call.Start()
	response_json_str, err := alibaba.CreateCluster(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, payload)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		tempCalllogger.Info(call.String(callLogInfo))
		return irs.ClusterInfo{}, err
	}

	// NodeGroup 생성 정보가 있는경우 생성을 시도한다.
	// 현재는 생성 시도를 안한다. 생성하기로 결정되면 아래 주석을 풀어서 사용한다.
	// 이유:
	// - Cluster 생성이 완료되어야 NodeGroup 생성이 가능하다.
	// - Cluster 생성이 완료되려면 최소 10분 이상 걸린다.
	// - 성공할때까지 반복하면서 생성을 시도해야 한다.
	// for _, node_group := range clusterReqInfo.NodeGroupList {
	// 	node_group_info, err := clusterHandler.AddNodeGroup(clusterReqInfo.IId, node_group)
	// 	if err != nil {
	// 		cblogger.Error(err)
	// 		return irs.ClusterInfo{}, err
	// 	}
	// }
	var response_json_obj map[string]interface{}
	json.Unmarshal([]byte(response_json_str), &response_json_obj)
	cluster_id := response_json_obj["cluster_id"].(string)
	cluster_info, err := getClusterInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, cluster_id)
	if err != nil {
		cblogger.Error(err)
		return irs.ClusterInfo{}, err
	}

	return *cluster_info, nil
}

func (clusterHandler *AlibabaClusterHandler) ListCluster() ([]*irs.ClusterInfo, error) {
	cblogger.Info("Alibaba Cloud Driver: called ListCluster()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, "ListCluster()", "ListCluster()")

	start := call.Start()
	clusters_json_str, err := alibaba.GetClusters(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		tempCalllogger.Info(call.String(callLogInfo))
		return nil, err
	}

	var clusters_json_obj map[string]interface{}
	json.Unmarshal([]byte(clusters_json_str), &clusters_json_obj)
	clusters := clusters_json_obj["clusters"].([]interface{})
	cluster_info_list := make([]*irs.ClusterInfo, len(clusters))
	for i, cluster := range clusters {
		println(i, cluster)
		cluster_id := cluster.(map[string]interface{})["cluster_id"].(string)
		cluster_info_list[i], err = getClusterInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, cluster_id)
		if err != nil {
			cblogger.Error(err)
			return nil, err
		}
	}

	return cluster_info_list, nil
}

func (clusterHandler *AlibabaClusterHandler) GetCluster(clusterIID irs.IID) (irs.ClusterInfo, error) {
	cblogger.Info("Alibaba Cloud Driver: called GetCluster()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "GetCluster()")

	start := call.Start()
	cluster_info, err := getClusterInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		tempCalllogger.Info(call.String(callLogInfo))
		return irs.ClusterInfo{}, err
	}

	return *cluster_info, nil
}

func (clusterHandler *AlibabaClusterHandler) DeleteCluster(clusterIID irs.IID) (bool, error) {
	cblogger.Info("Alibaba Cloud Driver: called DeleteCluster()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "DeleteCluster()")

	start := call.Start()
	res, err := alibaba.DeleteCluster(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		tempCalllogger.Info(call.String(callLogInfo))
		return false, err
	}
	cblogger.Info(res)

	return true, nil
}

func (clusterHandler *AlibabaClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (irs.NodeGroupInfo, error) {
	cblogger.Info("Alibaba Cloud Driver: called AddNodeGroup()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "AddNodeGroup()")

	// 노드 그룹 생성 요청을 JSON 요청으로 변환
	payload, err := getNodeGroupJSONString(clusterHandler, clusterIID, nodeGroupReqInfo)
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}

	start := call.Start()
	result_json_str, err := alibaba.CreateNodeGroup(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, payload)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		tempCalllogger.Info(call.String(callLogInfo))
		return irs.NodeGroupInfo{}, err
	}

	var result_json_obj map[string]interface{}
	json.Unmarshal([]byte(result_json_str), &result_json_obj)
	printFlattenJSON(result_json_obj)
	nodepool_id := result_json_obj["nodepool_id"].(string)
	node_group_info, err := getNodeGroupInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodepool_id)
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}

	return *node_group_info, nil
}

func (clusterHandler *AlibabaClusterHandler) ListNodeGroup(clusterIID irs.IID) ([]*irs.NodeGroupInfo, error) {
	cblogger.Info("Alibaba Cloud Driver: called ListNodeGroup()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "ListNodeGroup()")

	node_group_info_list := []*irs.NodeGroupInfo{}

	start := call.Start()
	node_groups_json_str, err := alibaba.ListNodeGroup(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		tempCalllogger.Info(call.String(callLogInfo))
		return node_group_info_list, err
	}

	var node_groups_json_obj map[string]interface{}
	json.Unmarshal([]byte(node_groups_json_str), &node_groups_json_obj)
	node_groups := node_groups_json_obj["nodepools"].([]interface{})
	for _, node_group := range node_groups {
		node_group_id := node_group.(map[string]interface{})["nodepool_info"].(map[string]interface{})["nodepool_id"].(string)
		node_group_info, err := getNodeGroupInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, node_group_id)
		if err != nil {
			cblogger.Error(err)
			return nil, err
		}
		node_group_info_list = append(node_group_info_list, node_group_info)
	}

	return node_group_info_list, nil
}

func (clusterHandler *AlibabaClusterHandler) GetNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (irs.NodeGroupInfo, error) {
	cblogger.Info("Alibaba Cloud Driver: called GetNodeGroup()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "GetNodeGroup()")

	start := call.Start()
	temp, err := getNodeGroupInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodeGroupIID.SystemId)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		tempCalllogger.Info(call.String(callLogInfo))
		return irs.NodeGroupInfo{}, err
	}

	return *temp, nil
}

func (clusterHandler *AlibabaClusterHandler) SetNodeGroupAutoScaling(clusterIID irs.IID, nodeGroupIID irs.IID, on bool) (bool, error) {
	cblogger.Info("Alibaba Cloud Driver: called SetNodeGroupAutoScaling()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "SetNodeGroupAutoScaling()")

	temp := `{"auto_scaling":{"enable":%t}}`
	body := fmt.Sprintf(temp, on)

	start := call.Start()
	res, err := alibaba.ModifyNodeGroup(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodeGroupIID.SystemId, body)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		tempCalllogger.Info(call.String(callLogInfo))
		return false, err
	}
	cblogger.Info(res)

	return true, nil
}

func (clusterHandler *AlibabaClusterHandler) ChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID, desiredNodeSize int, minNodeSize int, maxNodeSize int) (irs.NodeGroupInfo, error) {
	cblogger.Info("Alibaba Cloud Driver: called ChangeNodeGroupScaling()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "ChangeNodeGroupScaling()")

	// desired_size is not supported in alibaba with auto scaling mode
	temp := `{"auto_scaling":{"max_instances":%d,"min_instances":%d}}`
	body := fmt.Sprintf(temp, maxNodeSize, minNodeSize)
	start := call.Start()
	res, err := alibaba.ModifyNodeGroup(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodeGroupIID.SystemId, body)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		tempCalllogger.Info(call.String(callLogInfo))
		return irs.NodeGroupInfo{}, err
	}
	cblogger.Info(res)

	node_group_info, err := getNodeGroupInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodeGroupIID.SystemId)
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}

	return *node_group_info, nil
}

func (clusterHandler *AlibabaClusterHandler) RemoveNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (bool, error) {
	cblogger.Info("Alibaba Cloud Driver: called RemoveNodeGroup()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "RemoveNodeGroup()")

	start := call.Start()
	res, err := alibaba.DeleteNodeGroup(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodeGroupIID.SystemId)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		tempCalllogger.Info(call.String(callLogInfo))
		return false, err
	}
	cblogger.Info(res)

	return true, nil
}

func (clusterHandler *AlibabaClusterHandler) UpgradeCluster(clusterIID irs.IID, newVersion string) (irs.ClusterInfo, error) {
	cblogger.Info("Alibaba Cloud Driver: called UpgradeCluster()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "UpgradeCluster()")

	temp := `{"next_version" : "%s"}`
	body := fmt.Sprintf(temp, newVersion)

	start := call.Start()
	res, err := alibaba.UpgradeCluster(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, body)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		tempCalllogger.Info(call.String(callLogInfo))
		return irs.ClusterInfo{}, err
	}
	cblogger.Info(res)

	clusterInfo, err := getClusterInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId)
	if err != nil {
		cblogger.Error(err)
		return irs.ClusterInfo{}, err
	}

	return *clusterInfo, nil
}

func getClusterInfo(access_key string, access_secret string, region_id string, cluster_id string) (*irs.ClusterInfo, error) {

	var err error = nil
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("getClusterInfo() failed! %v", r)
		}
	}()

	cluster_json_str, err := alibaba.GetCluster(access_key, access_secret, region_id, cluster_id)
	if err != nil {
		return nil, err
	}

	var cluster_json_obj map[string]interface{}
	json.Unmarshal([]byte(cluster_json_str), &cluster_json_obj)

	// https://www.alibabacloud.com/help/doc-detail/86987.html
	// Initializing	Creating the cloud resources that are used by the cluster.
	// Creation Failed	Failed to create the cloud resources that are used by the cluster.
	// Running	The cloud resources used by the cluster are created.
	// Updating	Updating the metadata of the cluster.
	// Scaling	Adding nodes to the cluster.
	// Removing	Removing nodes from the cluster.
	// Upgrading	Upgrading the cluster.
	// Draining	Evicting pods from a node to other nodes. After all pods are evicted from the node, the node becomes unschudulable.
	// Deleting	Deleting the cluster.
	// Deletion Failed	Failed to delete the cluster.
	// Deleted (invisible to users)	The cluster is deleted."
	health_status := cluster_json_obj["state"].(string)
	cluster_status := irs.ClusterActive
	if strings.EqualFold(health_status, "Initializing") {
		cluster_status = irs.ClusterCreating
	} else if strings.EqualFold(health_status, "Updating") {
		cluster_status = irs.ClusterUpdating
	} else if strings.EqualFold(health_status, "Creation Failed") {
		cluster_status = irs.ClusterInactive
	} else if strings.EqualFold(health_status, "Deleting") {
		cluster_status = irs.ClusterDeleting
	} else if strings.EqualFold(health_status, "Running") {
		cluster_status = irs.ClusterActive
	}

	created_at := cluster_json_obj["created"].(string) // 2022-09-08T09:02:16+08:00,
	datetime, err := time.Parse(time.RFC3339, created_at)
	if err != nil {
		panic(err)
	}

	cluster_info := &irs.ClusterInfo{
		IId: irs.IID{
			NameId:   cluster_json_obj["name"].(string),
			SystemId: cluster_json_obj["cluster_id"].(string),
		},
		Version: cluster_json_obj["current_version"].(string),
		Network: irs.NetworkInfo{
			VpcIID: irs.IID{
				NameId:   "",
				SystemId: cluster_json_obj["vpc_id"].(string),
			},
			SecurityGroupIIDs: []irs.IID{
				{
					NameId:   "",
					SystemId: cluster_json_obj["security_group_id"].(string),
				},
			},
		},
		Status:      cluster_status,
		CreatedTime: datetime,
		// KeyValueList: []irs.KeyValue{}, // flatten data 입력하기
	}

	// k,v 추출 & 추가
	flat, err := flatten.Flatten(cluster_json_obj, "", flatten.DotStyle)
	if err != nil {
		return nil, err
	}
	for k, v := range flat {
		temp := fmt.Sprintf("%v", v)
		cluster_info.KeyValueList = append(cluster_info.KeyValueList, irs.KeyValue{Key: k, Value: temp})
	}

	// NodeGroups
	node_groups_json_str, err := alibaba.ListNodeGroup(access_key, access_secret, region_id, cluster_id)
	if err != nil {
		return nil, err
	}

	var node_groups_json_obj map[string]interface{}
	json.Unmarshal([]byte(node_groups_json_str), &node_groups_json_obj)
	node_groups := node_groups_json_obj["nodepools"].([]interface{})
	for _, node_group := range node_groups {
		node_group_id := node_group.(map[string]interface{})["nodepool_info"].(map[string]interface{})["nodepool_id"].(string)
		node_group_info, err := getNodeGroupInfo(access_key, access_secret, region_id, cluster_id, node_group_id)
		if err != nil {
			return nil, err
		}
		cluster_info.NodeGroupList = append(cluster_info.NodeGroupList, *node_group_info)
	}

	return cluster_info, err
}

func getNodeGroupInfo(access_key, access_secret, region_id, cluster_id, node_group_id string) (*irs.NodeGroupInfo, error) {

	var err error = nil
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("getNodeGroupInfo() failed! %v", r)
		}
	}()

	node_group_json_str, err := alibaba.GetNodeGroup(access_key, access_secret, region_id, cluster_id, node_group_id)
	if err != nil {
		return nil, err
	}

	var node_group_json_obj map[string]interface{}
	json.Unmarshal([]byte(node_group_json_str), &node_group_json_obj)

	// nodegroup.state
	// https://www.alibabacloud.com/help/en/container-service-for-kubernetes/latest/query-the-details-of-a-node-pool
	// active: The node pool is active.
	// scaling: The node pool is being scaled.
	// removing: Nodes are being removed from the node pool.
	// deleting: The node pool is being deleted.
	// updating: The node pool is being updated.
	health_status := node_group_json_obj["status"].(map[string]interface{})["state"].(string)
	status := irs.NodeGroupActive
	if strings.EqualFold(health_status, "active") {
		status = irs.NodeGroupActive
	} else if strings.EqualFold(health_status, "scaling") {
		status = irs.NodeGroupUpdating
	} else if strings.EqualFold(health_status, "removing") {
		status = irs.NodeGroupUpdating // removing is a kind of updating?
	} else if strings.EqualFold(health_status, "deleting") {
		status = irs.NodeGroupDeleting
	} else if strings.EqualFold(health_status, "updating") {
		status = irs.NodeGroupUpdating
	}

	node_group_info := irs.NodeGroupInfo{
		IId: irs.IID{
			NameId:   node_group_json_obj["nodepool_info"].(map[string]interface{})["name"].(string),
			SystemId: node_group_json_obj["nodepool_info"].(map[string]interface{})["nodepool_id"].(string),
		},
		ImageIID: irs.IID{
			NameId:   node_group_json_obj["scaling_group"].(map[string]interface{})["image_type"].(string),
			SystemId: node_group_json_obj["scaling_group"].(map[string]interface{})["image_id"].(string),
		},
		VMSpecName:   node_group_json_obj["scaling_group"].(map[string]interface{})["instance_types"].([]interface{})[0].(string),
		RootDiskType: node_group_json_obj["scaling_group"].(map[string]interface{})["system_disk_category"].(string),
		RootDiskSize: strconv.Itoa(int(node_group_json_obj["scaling_group"].(map[string]interface{})["system_disk_size"].(float64))),
		KeyPairIID: irs.IID{
			NameId:   node_group_json_obj["scaling_group"].(map[string]interface{})["key_pair"].(string),
			SystemId: "",
		},
		Status:          status,
		OnAutoScaling:   node_group_json_obj["auto_scaling"].(map[string]interface{})["enable"].(bool),
		MinNodeSize:     int(node_group_json_obj["auto_scaling"].(map[string]interface{})["min_instances"].(float64)),
		MaxNodeSize:     int(node_group_json_obj["auto_scaling"].(map[string]interface{})["max_instances"].(float64)),
		DesiredNodeSize: 0, // not supported in alibaba
		NodeList:        []irs.IID{},
		KeyValueList:    []irs.KeyValue{},
	}

	// k,v 추출 & 추가
	flat, err := flatten.Flatten(node_group_json_obj, "", flatten.DotStyle)
	if err != nil {
		return nil, err
	}
	for k, v := range flat {
		temp := fmt.Sprintf("%v", v)
		node_group_info.KeyValueList = append(node_group_info.KeyValueList, irs.KeyValue{Key: k, Value: temp})
	}

	return &node_group_info, err
}

func getClusterInfoJSON(clusterHandler *AlibabaClusterHandler, clusterInfo irs.ClusterInfo) (string, error) {

	var err error = nil
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("getClusterInfoJSON() failed! %v", r)
		}
	}()

	// get vswitch_id
	master_vswitch_id := ""
	res, err := alibaba.DescribeVSwitches(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterInfo.Network.VpcIID.SystemId)
	if err != nil {
		return "", err
	}
	for _, v := range res.VSwitches.VSwitch {
		master_vswitch_id = v.VSwitchId
		break
	}

	// get cidr list
	//cidr: Valid values: 10.0.0.0/16-24, 172.16-31.0.0/16-24, and 192.168.0.0/16-24.
	m_cidr := make(map[string]bool)
	for i := 16; i < 32; i++ {
		m_cidr[fmt.Sprintf("172.%v.0.0/16", i)] = true
	}
	clusters, err := clusterHandler.ListCluster()
	if err != nil {
		return "", err
	}
	for _, cluster := range clusters {
		for _, v := range cluster.KeyValueList {
			if v.Key == "parameters.ServiceCIDR" || v.Key == "subnet_cidr" {
				delete(m_cidr, v.Value)
			}
		}
	}
	cidr_list := []string{}
	for k := range m_cidr {
		cidr_list = append(cidr_list, k)
	}

	// create request json
	temp := `{
		"name": "%s",
		"region_id": "%s",
		"cluster_type": "ManagedKubernetes",
		"kubernetes_version": "1.22.10-aliyun.1",
		"vpcid": "%s",
		"container_cidr": "%s",
		"service_cidr": "%s",
		"num_of_nodes": 0,
		"master_vswitch_ids": ["%s"]
	}`

	clusterInfoJSON := fmt.Sprintf(temp, clusterInfo.IId.NameId, clusterHandler.RegionInfo.Region, clusterInfo.Network.VpcIID.SystemId, cidr_list[0], cidr_list[1], master_vswitch_id)

	return clusterInfoJSON, err
}

func getNodeGroupJSONString(clusterHandler *AlibabaClusterHandler, clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (string, error) {

	var err error = nil
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("getNodeGroupJSONString() failed! %v", r)
		}
	}()

	name := nodeGroupReqInfo.IId.NameId
	// image_id := nodeGroupReqInfo.ImageIID.SystemId // 옵션은 있으나, 설정해도 반영 안됨
	enable := nodeGroupReqInfo.OnAutoScaling
	max_instances := nodeGroupReqInfo.MaxNodeSize
	min_instances := nodeGroupReqInfo.MinNodeSize
	// desired_instances := nodeGroupReqInfo.DesiredNodeSize // not supported in alibaba
	instance_type := nodeGroupReqInfo.VMSpecName
	key_pair := nodeGroupReqInfo.KeyPairIID.NameId
	system_disk_category := nodeGroupReqInfo.RootDiskType
	system_disk_size, _ := strconv.ParseInt(nodeGroupReqInfo.RootDiskSize, 10, 32)

	// get vswitch_id
	clusterInfo, err := clusterHandler.GetCluster(clusterIID)
	if err != nil {
		return "", err
	}
	vswitch_id := "" // get vswitch_id, get from cluster info
	res, err := alibaba.DescribeVSwitches(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterInfo.Network.VpcIID.SystemId)
	if err != nil {
		return "", err
	}
	for _, v := range res.VSwitches.VSwitch {
		vswitch_id = v.VSwitchId
		break
	}

	temp := `{
		"nodepool_info": {
			"name": "%s"
		},
		"auto_scaling": {
			"enable": %t,
			"max_instances": %d,
			"min_instances": %d
		},
		"scaling_group": {
			"instance_types": ["%s"],
			"key_pair": "%s",
			"system_disk_category": "%s",
			"system_disk_size": %d,
			"vswitch_ids": ["%s"]
		},
		"management": {
			"enable":true
		}
	}`

	payload := fmt.Sprintf(temp, name, enable, max_instances, min_instances, instance_type, key_pair, system_disk_category, system_disk_size, vswitch_id)

	return payload, err
}

func getCallLogScheme(region string, resourceType call.RES_TYPE, resourceName string, apiName string) call.CLOUDLOGSCHEMA {
	cblogger.Info(fmt.Sprintf("Call %s %s", call.ALIBABA, apiName))
	return call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   region,
		ResourceType: resourceType,
		ResourceName: resourceName,
		CloudOSAPI:   apiName,
	}
}

func printFlattenJSON(json_obj interface{}) {
	temp, err := json.MarshalIndent(json_obj, "", "  ")
	if err != nil {
		println(err)
	} else {
		flat, err := flatten.FlattenString(string(temp), "", flatten.DotStyle)
		if err != nil {
			println(err)
		} else {
			println(flat)
		}
	}
}
