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
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	"github.com/jeremywohl/flatten"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NhnCloudClusterHandler struct {
	RegionInfo    idrv.RegionInfo
	VMClient      *nhnsdk.ServiceClient
	ClusterClient *nhnsdk.ServiceClient
}

func (clusterHandler *NhnCloudClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (irs.ClusterInfo, error) {
	cblogger.Info("NHN Cloud Driver: called CreateCluster()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterReqInfo.IId.NameId, "CreateCluster()")

	start := call.Start()
	// 클러스터 생성 요청을 JSON 요청으로 변환
	payload, err := getClusterInfoRequestJSONString(clusterReqInfo, clusterHandler)
	if err != nil {
		err := fmt.Errorf("Failed to Get ClusterInfo JSON String :  %v", err)
		cblogger.Error(err)
		return irs.ClusterInfo{}, err
	}
	response_json_str, err := CreateCluster(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, payload)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Create Cluster :  %v", err)
		cblogger.Error(err)
		cblogger.Error(call.String(callLogInfo))
		return irs.ClusterInfo{}, err
	}
	cblogger.Info(call.String(callLogInfo))

	// 클러스터를 생성하고 나서 바로 GetClusterInfo를 하면
	// 클러스터 정보를 가져올 때도 있지만 못가져 오는 경우도 있다.
	// 생성요청이 진행되는 중에 조회를 시도해서 그런것으로 추정된다.
	// 실패를 방지하기 위해 10초간 대기한 후에 조회를 시도한다.
	time.Sleep(time.Second * 10)

	// 클러스터 생성이 성공하면 getClusterInfo로 조회해서 반환한다.
	// 만약 getClusterInfo가 실패하면, 요청정보에 cluster_id를 포함해서 반환하는 것으로 처리한다.
	var response_json_obj map[string]interface{}
	json.Unmarshal([]byte(response_json_str), &response_json_obj)
	clusterReqInfo.IId.SystemId = response_json_obj["uuid"].(string)

	// // NodeGroup 생성 정보가 있는경우 생성을 시도한다.
	// // 문제는 Cluster 생성이 완료되어야 NodeGroup 생성이 가능하다.
	// // Cluster 생성이 완료되려면 최소 10분 이상 걸린다.
	// // 성공할때까지 대기한 후에 생성을 시도해야 하는가?
	// for {
	// 	time.Sleep(10 * time.Second)
	// 	clusterInfo, err := getClusterInfo(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterReqInfo.IId.SystemId)
	// 	if err != nil {
	// 		err := fmt.Errorf("Failed to Get ClusterInfo :  %v", err)
	// 		cblogger.Error(err)
	// 		return irs.ClusterInfo{}, err
	// 	}
	// 	cblogger.Info(clusterInfo.Status)
	// 	if clusterInfo.Status == irs.ClusterActive {
	// 		break
	// 	}
	// }

	// for _, node_group := range clusterReqInfo.NodeGroupList {
	// 	node_group_info, err := clusterHandler.AddNodeGroup(cluster_info.IId, node_group)
	// 	if err != nil {
	// 		cblogger.Error(err)
	// 		return irs.ClusterInfo{}, err
	// 	}
	// 	cluster_info.NodeGroupList = append(cluster_info.NodeGroupList, node_group_info)
	// }

	cluster_info, err := getClusterInfo(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterReqInfo.IId.SystemId)
	if err != nil {
		err := fmt.Errorf("Failed to Get ClusterInfo :  %v", err)
		cblogger.Error(err)
		return irs.ClusterInfo{}, err
	}
	if cluster_info == nil {
		err = fmt.Errorf("Succeeded in Creating Cluster but Failed to Get ClusterInfo")
		cblogger.Error(err)
		return clusterReqInfo, err
	}

	return *cluster_info, nil
}

func (clusterHandler *NhnCloudClusterHandler) ListCluster() ([]*irs.ClusterInfo, error) {

	cblogger.Info("NHN Cloud Driver: called ListCluster()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, "ListCluster()", "ListCluster()") // HisCall logging

	start := call.Start()
	clusters_json_str, err := GetClusters(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to List Cluster :  %v", err)
		cblogger.Error(err)
		cblogger.Error(call.String(callLogInfo))
		return nil, err
	}
	cblogger.Info(call.String(callLogInfo))

	var clusters_json_obj map[string]interface{}
	json.Unmarshal([]byte(clusters_json_str), &clusters_json_obj)
	clusters := clusters_json_obj["clusters"].([]interface{})
	cluster_info_list := make([]*irs.ClusterInfo, len(clusters))
	for i, cluster := range clusters {
		uuid := cluster.(map[string]interface{})["uuid"].(string)
		cluster_info_list[i], err = getClusterInfo(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, uuid)
		if err != nil {
			err := fmt.Errorf("Failed to Get ClusterInfo :  %v", err)
			cblogger.Error(err)
			return nil, err
		}
	}

	return cluster_info_list, nil
}

func (clusterHandler *NhnCloudClusterHandler) GetCluster(clusterIID irs.IID) (irs.ClusterInfo, error) {
	cblogger.Info("NHN Cloud Driver: called GetCluster()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "GetCluster()")

	start := call.Start()
	cluster_info, err := getClusterInfo(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Get ClusterInfo :  %v", err)
		cblogger.Error(err)
		cblogger.Error(call.String(callLogInfo))
		return irs.ClusterInfo{}, err
	}
	cblogger.Info(call.String(callLogInfo))

	return *cluster_info, nil
}

func (clusterHandler *NhnCloudClusterHandler) DeleteCluster(clusterIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called DeleteCluster()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "DeleteCluster()")

	start := call.Start()
	res, err := DeleteCluster(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Delete Cluster :  %v", err)
		cblogger.Error(err)
		cblogger.Error(call.String(callLogInfo))
		return false, err
	}
	if res != "" {
		// 삭제 처리를 성공하면 ""를 리턴한다.
		// 삭제 처리를 실패하면 에러 메시지를 리턴한다.
		err = fmt.Errorf("Failed to Delete Cluster :  %s", res)
		cblogger.Error(err)
		cblogger.Error(call.String(callLogInfo))
		return false, err
	}
	cblogger.Info(call.String(callLogInfo))

	return true, nil
}

func (clusterHandler *NhnCloudClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (irs.NodeGroupInfo, error) {
	cblogger.Info("NHN Cloud Driver: called AddNodeGroup()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "AddNodeGroup()")

	start := call.Start()
	// 노드 그룹 생성 요청을 JSON 요청으로 변환
	payload, err := getNodeGroupRequestJSONString(nodeGroupReqInfo, clusterHandler)
	if err != nil {
		err := fmt.Errorf("Failed to Get NodeGroup Request JSON String :  %v", err)
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}
	result_json_str, err := CreateNodeGroup(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId, payload)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Create NodeGroup :  %v", err)
		cblogger.Error(err)
		cblogger.Error(call.String(callLogInfo))
		return irs.NodeGroupInfo{}, err
	}
	cblogger.Info(call.String(callLogInfo))

	var result_json_obj map[string]interface{}
	json.Unmarshal([]byte(result_json_str), &result_json_obj)
	if result_json_obj["errors"] != nil {
		err := fmt.Errorf("Failed to Create NodeGroup :  %s", result_json_str)
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}
	uuid := result_json_obj["uuid"].(string)
	node_group_info, err := getNodeGroupInfo(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId, uuid)
	if err != nil {
		err := fmt.Errorf("Failed to Get NodeGroupInfo :  %v", err)
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}

	return *node_group_info, nil
}

func (clusterHandler *NhnCloudClusterHandler) ListNodeGroup(clusterIID irs.IID) ([]*irs.NodeGroupInfo, error) {
	cblogger.Info("NHN Cloud Driver: called ListNodeGroup()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "ListNodeGroup()")

	start := call.Start()
	node_group_info_list := []*irs.NodeGroupInfo{}
	node_groups_json_str, err := GetNodeGroups(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to List NodeGroup :  %v", err)
		cblogger.Error(err)
		cblogger.Error(call.String(callLogInfo))
		return node_group_info_list, err
	}
	cblogger.Info(call.String(callLogInfo))

	var node_groups_json_obj map[string]interface{}
	json.Unmarshal([]byte(node_groups_json_str), &node_groups_json_obj)
	node_groups := node_groups_json_obj["nodegroups"].([]interface{})
	for _, node_group := range node_groups {
		node_group_info, err := getNodeGroupInfo(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId, node_group.(map[string]interface{})["uuid"].(string))
		if err != nil {
			err := fmt.Errorf("Failed to Get NodeGroupInfo :  %v", err)
			cblogger.Error(err)
			return node_group_info_list, err
		}
		node_group_info_list = append(node_group_info_list, node_group_info)
	}

	return node_group_info_list, nil
}

func (clusterHandler *NhnCloudClusterHandler) GetNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (irs.NodeGroupInfo, error) {
	cblogger.Info("NHN Cloud Driver: called GetNodeGroup()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "GetNodeGroup()")

	start := call.Start()
	temp, err := getNodeGroupInfo(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId, nodeGroupIID.SystemId)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Get NodeGroup :  %v", err)
		cblogger.Error(err)
		cblogger.Error(call.String(callLogInfo))
		return irs.NodeGroupInfo{}, err
	}
	cblogger.Info(call.String(callLogInfo))

	return *temp, nil
}

func (clusterHandler *NhnCloudClusterHandler) SetNodeGroupAutoScaling(clusterIID irs.IID, nodeGroupIID irs.IID, on bool) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called SetNodeGroupAutoScaling()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "GetNodeGroup()")

	start := call.Start()
	temp := `{"ca_enable": %t}`
	payload := fmt.Sprintf(temp, on)
	_, err := ChangeNodeGroupAutoScaler(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId, nodeGroupIID.SystemId, payload)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Set NodeGroup AutoScaling :  %v", err)
		cblogger.Error(err)
		cblogger.Error(call.String(callLogInfo))
		return false, err
	}
	cblogger.Info(call.String(callLogInfo))

	return true, nil
}

func (clusterHandler *NhnCloudClusterHandler) ChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID, desiredNodeSize int, minNodeSize int, maxNodeSize int) (irs.NodeGroupInfo, error) {
	cblogger.Info("NHN Cloud Driver: called ChangeNodeGroupScaling()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "GetNodeGroup()")

	start := call.Start()
	temp := `{
		"ca_enable": true,
		"ca_max_node_count": %d,
		"ca_min_node_count": %d
	}`
	payload := fmt.Sprintf(temp, maxNodeSize, minNodeSize)
	//desiredNodeSize is not supported in NHN Cloud
	_, err := ChangeNodeGroupAutoScaler(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId, nodeGroupIID.SystemId, payload)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Change NodeGroup Scaling :  %v", err)
		cblogger.Error(err)
		cblogger.Error(call.String(callLogInfo))
		return irs.NodeGroupInfo{}, err
	}
	cblogger.Info(call.String(callLogInfo))

	node_group_info, err := getNodeGroupInfo(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId, nodeGroupIID.SystemId)
	if err != nil {
		err := fmt.Errorf("Failed to Get NodeGroupInfo :  %v", err)
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}

	return *node_group_info, nil
}

func (clusterHandler *NhnCloudClusterHandler) RemoveNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called RemoveNodeGroup()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "RemoveNodeGroup()")

	start := call.Start()
	res, err := DeleteNodeGroup(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId, nodeGroupIID.SystemId)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Remove NodeGroup :  %v", err)
		cblogger.Error(err)
		cblogger.Error(call.String(callLogInfo))
		return false, err
	}
	if res != "" {
		// 삭제 처리를 성공하면 ""를 리턴한다.
		// 삭제 처리를 실패하면 에러 메시지를 리턴한다.
		err := fmt.Errorf("Failed to Remove NodeGroup :  %s", res)
		cblogger.Error(err)
		cblogger.Error(call.String(callLogInfo))
		return false, err
	}
	cblogger.Info(call.String(callLogInfo))

	return true, nil
}

// 업그레이드 순서
// default-master 노드 그룹을 업그레이드한다.
// default-master 업그레이드가 완료되면, worker 노드들을 업그레이드 한다.
// default-master 업그레이드가 완료되기 전에는 worker 노드를 업그레이드 할 수 없다.
// default-master 업그레이드가 완료된 후에 (10분? 정도 소요됨) worker 노드를 업그레이드해야 한다.
func (clusterHandler *NhnCloudClusterHandler) UpgradeCluster(clusterIID irs.IID, newVersion string) (irs.ClusterInfo, error) {
	cblogger.Info("NHN Cloud Driver: called UpgradeCluster()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "UpgradeCluster()")

	start := call.Start()
	temp := `{"version": "%s"}`
	payload := fmt.Sprintf(temp, newVersion)
	res, err := UpgradeCluster(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId, "default-master", payload)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Upgrade Cluster :  %v", err)
		cblogger.Error(err)
		cblogger.Error(call.String(callLogInfo))
		return irs.ClusterInfo{}, err
	}
	if strings.Contains(res, "errors") {
		// {"errors": [{"request_id": "", "code": "", "status": 406, "title": "", "detail": "", "links": []}]}
		err := fmt.Errorf("Failed to Upgrade Cluster :  %s", res)
		cblogger.Error(err)
		cblogger.Error(call.String(callLogInfo))
		return irs.ClusterInfo{}, err
	}
	cblogger.Info(call.String(callLogInfo))

	// // default-master 업그레이드 완료 확인 코드
	// // 업그레이드가 완료되면 worker 노드 업그레이드 진행
	// for {
	// 	time.Sleep(10 * time.Second)
	// 	clusterInfo, err := getClusterInfo(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId)
	// 	if err != nil {
	// 		err := fmt.Errorf("Failed to Get ClusterInfo :  %v", err)
	// 		cblogger.Error(err)
	// 		return irs.ClusterInfo{}, err
	// 	}
	// 	cblogger.Info(clusterInfo.Status)
	// 	if clusterInfo.Status == irs.ClusterActive {
	// 		break
	// 	}
	// }

	// node_groups_json_str, err := GetNodeGroups(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId)
	// if err != nil {
	// 	err := fmt.Errorf("Failed to Get NodeGroups :  %v", err)
	// 	cblogger.Error(err)
	// 	return irs.ClusterInfo{}, err
	// }
	// var node_groups_json_obj map[string]interface{}
	// json.Unmarshal([]byte(node_groups_json_str), &node_groups_json_obj)
	// node_groups := node_groups_json_obj["nodegroups"].([]interface{})
	// for _, node_group := range node_groups {
	// 	start := call.Start()
	// 	res, err := UpgradeCluster(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId, node_group.(map[string]interface{})["uuid"].(string), payload)
	// 	callLogInfo.ElapsedTime = call.Elapsed(start)
	// 	cblogger.Info(call.String(callLogInfo))
	// 	if err != nil {
	// 		err := fmt.Errorf("Failed to Upgrade Cluster :  %v", err)
	// 		cblogger.Error(err)
	// 		return irs.ClusterInfo{}, err
	// 	}
	// 	if strings.Contains(res, "errors") {
	// 		// {"errors": [{"request_id": "", "code": "", "status": 406, "title": "", "detail": "", "links": []}]}
	// 		err := fmt.Errorf("Failed to Upgrade Cluster :  %s", res)
	// 		cblogger.Error(err)
	// 		return irs.ClusterInfo{}, err
	// 	}
	// }

	clusterInfo, err := getClusterInfo(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId)
	if err != nil {
		err := fmt.Errorf("Failed to Get ClusterInfo :  %v", err)
		cblogger.Error(err)
		return irs.ClusterInfo{}, err
	}

	return *clusterInfo, nil
}

func getClusterInfo(host string, token string, cluster_id string) (clusterInfo *irs.ClusterInfo, err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Failed to Process GetClusterInfo() %v\n\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cluster_json_str, err := GetCluster(host, token, cluster_id)
	if err != nil {
		err := fmt.Errorf("Failed to Get Cluster :  %v", err)
		cblogger.Error(err)
		return nil, err
	}
	var cluster_json_obj map[string]interface{}
	json.Unmarshal([]byte(cluster_json_str), &cluster_json_obj)

	health_status := cluster_json_obj["status"].(string)
	cluster_status := irs.ClusterActive
	if health_status == "CREATE_IN_PROGRESS" {
		cluster_status = irs.ClusterCreating
	} else if health_status == "UPDATE_IN_PROGRESS" {
		cluster_status = irs.ClusterUpdating
	} else if health_status == "UPDATE_FAILED" {
		cluster_status = irs.ClusterInactive
	} else if health_status == "DELETE_IN_PROGRESS" {
		cluster_status = irs.ClusterDeleting
	} else if health_status == "CREATE_COMPLETE" {
		cluster_status = irs.ClusterActive
	}

	created_at := cluster_json_obj["created_at"].(string)
	datetime, err := time.Parse(time.RFC3339, created_at)
	//RFC3339     = "2006-01-02T15:04:05Z07:00"
	//datetime, err := time.Parse("2006-01-02T15:04:05+07:00", created_at)
	if err != nil {
		err := fmt.Errorf("Failed to Parse Created Time :  %v", err)
		cblogger.Error(err)
		panic(err)
	}

	version := ""
	if cluster_json_obj["coe_version"] != nil {
		version = cluster_json_obj["coe_version"].(string)
	}

	clusterInfo = &irs.ClusterInfo{
		IId: irs.IID{
			NameId:   cluster_json_obj["name"].(string),
			SystemId: cluster_json_obj["uuid"].(string),
		},
		Version: version,
		Network: irs.NetworkInfo{
			VpcIID: irs.IID{
				NameId:   "",
				SystemId: cluster_json_obj["fixed_network"].(string),
			},
			SubnetIIDs: []irs.IID{
				{
					NameId:   "",
					SystemId: cluster_json_obj["fixed_subnet"].(string),
				},
			},
			SecurityGroupIIDs: []irs.IID{
				{
					NameId:   "",
					SystemId: "",
				},
			},
		},
		Addons: irs.AddonsInfo{
			KeyValueList: []irs.KeyValue{},
		},
		Status:       cluster_status,
		CreatedTime:  datetime,
		KeyValueList: []irs.KeyValue{},
	}

	// k,v 추출
	// k,v 변환 규칙 작성 [k,v]:[ClusterInfo.k, ClusterInfo.v]
	// 변환 규칙에 따라 k,v 변환
	// flat, err := flatten.FlattenString(cluster_json_str, "", flatten.DotStyle)
	flat, err := flatten.Flatten(cluster_json_obj, "", flatten.DotStyle)
	if err != nil {
		err := fmt.Errorf("Failed to Flatten Cluster Info :  %v", err)
		cblogger.Error(err)
		return nil, err
	}
	for k, v := range flat {
		temp := fmt.Sprintf("%v", v)
		clusterInfo.KeyValueList = append(clusterInfo.KeyValueList, irs.KeyValue{Key: k, Value: temp})
	}

	// NodeGroups
	node_groups_json_str, err := GetNodeGroups(host, token, cluster_id)
	if err != nil {
		err := fmt.Errorf("Failed to Get NodeGroups :  %v", err)
		cblogger.Error(err)
		return nil, err
	}

	if node_groups_json_str == "" {
		err := fmt.Errorf("Failed to Get Node Groups")
		cblogger.Error(err)
		return nil, err
	}

	var node_groups_json_obj map[string]interface{}
	json.Unmarshal([]byte(node_groups_json_str), &node_groups_json_obj)
	node_groups := node_groups_json_obj["nodegroups"].([]interface{})
	for _, node_group := range node_groups {
		node_group_id := node_group.(map[string]interface{})["uuid"].(string)
		node_group_info, err := getNodeGroupInfo(host, token, cluster_id, node_group_id)
		if err != nil {
			err := fmt.Errorf("Failed to Get NodeGroupInfo :  %v", err)
			cblogger.Error(err)
			return nil, err
		}
		clusterInfo.NodeGroupList = append(clusterInfo.NodeGroupList, *node_group_info)
	}

	return clusterInfo, err
}

func getNodeGroupInfo(host string, token string, cluster_id string, node_group_id string) (nodeGroupInfo *irs.NodeGroupInfo, err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Failed to Process NodeGroupInfo : %v\n\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	node_group_json_str, err := GetNodeGroup(host, token, cluster_id, node_group_id)
	if err != nil {
		err := fmt.Errorf("Failed to Get NodeGroup :  %v", err)
		cblogger.Error(err)
		return nil, err
	}

	if node_group_json_str == "" {
		err := fmt.Errorf("Failed to Get NodeGroup")
		cblogger.Error(err)
		return nil, err
	}

	var node_group_json_obj map[string]interface{}
	json.Unmarshal([]byte(node_group_json_str), &node_group_json_obj)

	health_status := node_group_json_obj["status"].(string)
	status := irs.NodeGroupActive
	if strings.EqualFold(health_status, "UPDATE_COMPLETE") {
		status = irs.NodeGroupActive
	} else if strings.EqualFold(health_status, "CREATE_IN_PROGRESS") {
		status = irs.NodeGroupUpdating
	} else if strings.EqualFold(health_status, "UPDATE_IN_PROGRESS") {
		status = irs.NodeGroupUpdating // removing is a kind of updating?
	} else if strings.EqualFold(health_status, "DELETE_IN_PROGRESS") {
		status = irs.NodeGroupDeleting
	} else if strings.EqualFold(health_status, "UPDATE_IN_PROGRESS") {
		status = irs.NodeGroupUpdating
	} else if strings.EqualFold(health_status, "CREATE_COMPLETE") {
		status = irs.NodeGroupActive
	}

	auto_scaling, _ := strconv.ParseBool(node_group_json_obj["labels"].(map[string]interface{})["ca_enable"].(string))
	ca_min_node_count, _ := strconv.ParseInt(node_group_json_obj["labels"].(map[string]interface{})["ca_min_node_count"].(string), 10, 32)
	ca_max_node_count, _ := strconv.ParseInt(node_group_json_obj["labels"].(map[string]interface{})["ca_max_node_count"].(string), 10, 32)
	node_count := int(node_group_json_obj["node_count"].(float64))

	nodeGroupInfo = &irs.NodeGroupInfo{
		IId: irs.IID{
			NameId:   node_group_json_obj["name"].(string),
			SystemId: node_group_json_obj["uuid"].(string),
		},
		ImageIID: irs.IID{
			NameId:   "",
			SystemId: node_group_json_obj["image_id"].(string),
		},
		VMSpecName:      node_group_json_obj["flavor_id"].(string),
		RootDiskType:    node_group_json_obj["labels"].(map[string]interface{})["boot_volume_size"].(string),
		RootDiskSize:    node_group_json_obj["labels"].(map[string]interface{})["boot_volume_type"].(string),
		KeyPairIID:      irs.IID{},
		OnAutoScaling:   auto_scaling,
		MinNodeSize:     int(ca_min_node_count),
		MaxNodeSize:     int(ca_max_node_count),
		DesiredNodeSize: int(node_count),
		Nodes:           []irs.IID{},
		KeyValueList:    []irs.KeyValue{},
		Status:          status,
	}

	// k,v 추출
	// k,v 변환 규칙 작성 [k,v]:[ClusterInfo.k, ClusterInfo.v]
	// 변환 규칙에 따라 k,v 변환
	// flat, err := flatten.FlattenString(cluster_json_str, "", flatten.DotStyle)
	flat, err := flatten.Flatten(node_group_json_obj, "", flatten.DotStyle)
	if err != nil {
		err := fmt.Errorf("Failed to Flatten NodeGroup")
		cblogger.Error(err)
		return nil, err
	}
	for k, v := range flat {
		temp := fmt.Sprintf("%v", v)
		nodeGroupInfo.KeyValueList = append(nodeGroupInfo.KeyValueList, irs.KeyValue{Key: k, Value: temp})
	}

	return nodeGroupInfo, err
}

func getClusterInfoRequestJSONString(clusterInfo irs.ClusterInfo, clusterHandler *NhnCloudClusterHandler) (info string, err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Failed to Process getClusterInfoRequestJSONString() : %v\n\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	fixed_network := clusterInfo.Network.VpcIID.SystemId
	fixed_subnet := clusterInfo.Network.SubnetIIDs[0].SystemId
	flavor_id := clusterInfo.NodeGroupList[0].VMSpecName
	keypair := clusterInfo.NodeGroupList[0].KeyPairIID.NameId

	availability_zone := clusterHandler.RegionInfo.Zone
	boot_volume_size := clusterInfo.NodeGroupList[0].RootDiskSize
	boot_volume_type := clusterInfo.NodeGroupList[0].RootDiskType

	ca_enable := "false"
	if clusterInfo.NodeGroupList[0].OnAutoScaling {
		ca_enable = "true"
	}
	ca_max_node_count := strconv.Itoa(clusterInfo.NodeGroupList[0].MaxNodeSize)
	ca_min_node_count := strconv.Itoa(clusterInfo.NodeGroupList[0].MinNodeSize)
	kube_tag := clusterInfo.Version
	node_image := clusterInfo.NodeGroupList[0].ImageIID.SystemId
	name := clusterInfo.IId.NameId
	node_count := strconv.Itoa(clusterInfo.NodeGroupList[0].DesiredNodeSize)

	for _, v := range clusterInfo.NodeGroupList[0].KeyValueList {
		switch v.Key {
		case "availability_zone":
			availability_zone = v.Value
		}
	}

	temp := `{
		"cluster_template_id": "iaas_console", 
		"create_timeout": 60,
		"fixed_network": "%s",
		"fixed_subnet": "%s",
		"flavor_id": "%s",
		"keypair": "%s",
		"labels": {
			"availability_zone": "%s",
			"boot_volume_size": "%s",
			"boot_volume_type": "%s",
			"ca_enable": "%s",
			"ca_max_node_count": "%s",
			"ca_min_node_count": "%s",
			"cert_manager_api": "True",
			"clusterautoscale": "nodegroupfeature",
			"kube_tag": "%s",
			"master_lb_floating_ip_enabled": "False",
			"node_image": "%s",
			"user_script_v2": ""
		},
		"name": "%s",
		"node_count": %s
	}`

	info = fmt.Sprintf(temp, fixed_network, fixed_subnet, flavor_id, keypair, availability_zone, boot_volume_size, boot_volume_type, ca_enable, ca_max_node_count, ca_min_node_count, kube_tag, node_image, name, node_count)

	return info, err
}

func getNodeGroupRequestJSONString(nodeGroupReqInfo irs.NodeGroupInfo, clusterHandler *NhnCloudClusterHandler) (payload string, err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Failed to Process getNodeGroupRequestJSONString() : %v\n\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	name := nodeGroupReqInfo.IId.NameId
	node_count := nodeGroupReqInfo.DesiredNodeSize
	flavor_id := nodeGroupReqInfo.VMSpecName
	image_id := nodeGroupReqInfo.ImageIID.SystemId

	boot_volume_size := nodeGroupReqInfo.RootDiskSize
	boot_volume_type := nodeGroupReqInfo.RootDiskType

	ca_enable := strconv.FormatBool(nodeGroupReqInfo.OnAutoScaling)
	ca_max_node_count := strconv.Itoa(nodeGroupReqInfo.MaxNodeSize)
	ca_min_node_count := strconv.Itoa(nodeGroupReqInfo.MinNodeSize)

	availability_zone := clusterHandler.RegionInfo.Zone

	temp := `{
	    "name": "%s",
	    "node_count": %d,
	    "flavor_id": "%s",
	    "image_id": "%s",
	    "labels": {
	        "availability_zone": "%s",
	        "boot_volume_size": "%s",
	        "boot_volume_type": "%s",
	        "ca_enable": "%s",
			"ca_max_node_count": "%s",
			"ca_min_node_count": "%s"
	    }
	}`

	payload = fmt.Sprintf(temp, name, node_count, flavor_id, image_id, availability_zone, boot_volume_size, boot_volume_type, ca_enable, ca_max_node_count, ca_min_node_count)

	return payload, err
}
