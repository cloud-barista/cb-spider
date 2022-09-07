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
	"errors"
	"fmt"
	"sync"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
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

type AlibabaClusterHandler struct {
	RegionInfo   idrv.RegionInfo
	AccessKey    string
	AccessSecret string
	RegionID     string
}

func (clusterHandler *AlibabaClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (irs.ClusterInfo, error) {
	cblogger.Info("Alibaba Cloud Driver: called CreateCluster()")
	//callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, "CreateCluster()", "CreateCluster()")
	// // 클러스터 생성 요청을 JSON 요청으로 변환
	// payload, err := getClusterInfoJSON(clusterReqInfo)
	// if err != nil {
	// 	cblogger.Error(err)
	// 	return irs.ClusterInfo{}, err
	// }

	// start := call.Start()
	// // access_key string, access_secret string, region_id string, body string
	// response_json_str, err := alibaba.CreateCluster(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, payload)
	// loggingInfo(callLogInfo, start)
	// if err != nil {
	// 	cblogger.Error(err)
	// 	loggingError(callLogInfo, err)
	// 	return irs.ClusterInfo{}, err
	// }

	// var response_json_obj map[string]interface{}
	// json.Unmarshal([]byte(response_json_str), &response_json_obj)
	// if v, found := response_json_obj["uuid"]; found {
	// 	clusterReqInfo.IId.SystemId = v.(string)
	// } else {
	// 	err = errors.New(response_json_str)
	// 	cblogger.Error(err)
	// 	return irs.ClusterInfo{}, err
	// }

	// return clusterReqInfo, nil

	return irs.ClusterInfo{}, nil
}

func (clusterHandler *AlibabaClusterHandler) ListCluster() ([]*irs.ClusterInfo, error) {
	cblogger.Info("Alibaba Cloud Driver: called ListCluster()")
	// callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, "ListCluster()", "ListCluster()")

	// start := call.Start()
	// clusters_json_str, err := alibaba.GetClusters(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID)
	// LoggingInfo(callLogInfo, start)
	// if err != nil {
	// 	return nil, err
	// }

	// var clusters_json_obj map[string]interface{}
	// json.Unmarshal([]byte(clusters_json_str), &clusters_json_obj)
	// clusters := clusters_json_obj["clusters"].([]interface{})
	// cluster_info_list := make([]*irs.ClusterInfo, len(clusters))
	// for i, cluster := range clusters {
	// 	uuid := cluster.(map[string]interface{})["uuid"].(string)
	// 	cluster_info_list[i], err = getClusterInfo(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, uuid)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	// return cluster_info_list, nil

	return nil, nil
}

func (clusterHandler *AlibabaClusterHandler) GetCluster(clusterIID irs.IID) (irs.ClusterInfo, error) {
	cblogger.Info("Alibaba Cloud Driver: called GetCluster()")
	// callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "GetCluster()")

	// start := call.Start()
	// cluster_info, err := getClusterInfo(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId)
	// LoggingInfo(callLogInfo, start)
	// if err != nil {
	// 	return irs.ClusterInfo{}, err
	// }

	// return *cluster_info, nil

	return irs.ClusterInfo{}, nil
}

func (clusterHandler *AlibabaClusterHandler) DeleteCluster(clusterIID irs.IID) (bool, error) {
	cblogger.Info("Alibaba Cloud Driver: called DeleteCluster()")
	// callLogInfo := GetCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "DeleteCluster()")

	// start := call.Start()
	// res, err := alibaba.DeleteCluster(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId)
	// LoggingInfo(callLogInfo, start)
	// if err != nil {
	// 	return false, err
	// }
	// if res != "" {
	// 	// 삭제 처리를 성공하면 ""를 리턴한다.
	// 	// 삭제 처리를 실패하면 에러 메시지를 리턴한다.
	// 	return false, errors.New(res)
	// }

	// return true, nil

	return false, nil
}

func (clusterHandler *AlibabaClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (irs.NodeGroupInfo, error) {
	cblogger.Info("Alibaba Cloud Driver: called AddNodeGroup()")
	// callLogInfo := GetCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "AddNodeGroup()")

	// // 노드 그룹 생성 요청을 JSON 요청으로 변환
	// payload, err := getNodeGroupJSONString(nodeGroupReqInfo)
	// if err != nil {
	// 	return irs.NodeGroupInfo{}, err
	// }

	// start := call.Start()
	// result_json_str, err := alibaba.CreateNodeGroup(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId, payload)
	// LoggingInfo(callLogInfo, start)
	// if err != nil {
	// 	return irs.NodeGroupInfo{}, err
	// }

	// var result_json_obj map[string]interface{}
	// json.Unmarshal([]byte(result_json_str), &result_json_obj)
	// if result_json_obj["errors"] != nil {
	// 	return irs.NodeGroupInfo{}, errors.New(result_json_str)
	// }
	// uuid := result_json_obj["uuid"].(string)
	// node_group_info, err := getNodeGroupInfo(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId, uuid)
	// if err != nil {
	// 	return irs.NodeGroupInfo{}, err
	// }

	// return *node_group_info, nil

	return irs.NodeGroupInfo{}, nil
}

func (clusterHandler *AlibabaClusterHandler) ListNodeGroup(clusterIID irs.IID) ([]*irs.NodeGroupInfo, error) {
	cblogger.Info("Alibaba Cloud Driver: called ListNodeGroup()")
	// callLogInfo := GetCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "ListNodeGroup()")

	// node_group_info_list := []*irs.NodeGroupInfo{}

	// start := call.Start()
	// node_groups_json_str, err := alibaba.GetNodeGroups(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId)
	// LoggingInfo(callLogInfo, start)
	// if err != nil {
	// 	return node_group_info_list, err
	// }
	// var node_groups_json_obj map[string]interface{}
	// json.Unmarshal([]byte(node_groups_json_str), &node_groups_json_obj)
	// node_groups := node_groups_json_obj["nodegroups"].([]interface{})
	// for _, node_group := range node_groups {
	// 	node_group_info, err := getNodeGroupInfo(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId, node_group.(map[string]interface{})["uuid"].(string))
	// 	if err != nil {
	// 		return node_group_info_list, err
	// 	}
	// 	node_group_info_list = append(node_group_info_list, node_group_info)
	// }

	// return node_group_info_list, nil

	return nil, nil
}

func (clusterHandler *AlibabaClusterHandler) GetNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (irs.NodeGroupInfo, error) {
	cblogger.Info("Alibaba Cloud Driver: called GetNodeGroup()")
	// callLogInfo := GetCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "GetNodeGroup()")

	// start := call.Start()
	// temp, err := getNodeGroupInfo(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId, nodeGroupIID.SystemId)
	// LoggingInfo(callLogInfo, start)
	// if err != nil {
	// 	return irs.NodeGroupInfo{}, err
	// }

	// return *temp, nil

	return irs.NodeGroupInfo{}, nil
}

func (clusterHandler *AlibabaClusterHandler) SetNodeGroupAutoScaling(clusterIID irs.IID, nodeGroupIID irs.IID, on bool) (bool, error) {
	cblogger.Info("Alibaba Cloud Driver: called SetNodeGroupAutoScaling()")
	return false, errors.New("SetNodeGroupAutoScaling is not supported")
}

func (clusterHandler *AlibabaClusterHandler) ChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID, desiredNodeSize int, minNodeSize int, maxNodeSize int) (irs.NodeGroupInfo, error) {
	cblogger.Info("Alibaba Cloud Driver: called ChangeNodeGroupScaling()")
	return irs.NodeGroupInfo{}, errors.New("ChangeNodeGroupScaling is not supported")
}

func (clusterHandler *AlibabaClusterHandler) RemoveNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (bool, error) {
	cblogger.Info("Alibaba Cloud Driver: called RemoveNodeGroup()")
	// callLogInfo := GetCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "RemoveNodeGroup()")

	// start := call.Start()
	// res, err := alibaba.DeleteNodeGroup(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId, nodeGroupIID.SystemId)
	// LoggingInfo(callLogInfo, start)
	// if err != nil {
	// 	return false, err
	// }
	// if res != "" {
	// 	// 삭제 처리를 성공하면 ""를 리턴한다.
	// 	// 삭제 처리를 실패하면 에러 메시지를 리턴한다.
	// 	return false, errors.New(res)
	// }

	// return true, nil

	return false, nil
}

func (clusterHandler *AlibabaClusterHandler) UpgradeCluster(clusterIID irs.IID, newVersion string) (irs.ClusterInfo, error) {
	cblogger.Info("Alibaba Cloud Driver: called UpgradeCluster()")
	// callLogInfo := GetCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "UpgradeCluster()")

	// node_groups_json_str, err := alibaba.GetNodeGroups(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId)
	// if err != nil {
	// 	return irs.ClusterInfo{}, err
	// }
	// var node_groups_json_obj map[string]interface{}
	// json.Unmarshal([]byte(node_groups_json_str), &node_groups_json_obj)
	// node_groups := node_groups_json_obj["nodegroups"].([]interface{})
	// for _, node_group := range node_groups {
	// 	start := call.Start()
	// 	res, err := alibaba.UpgradeCluster(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId, node_group.(map[string]interface{})["uuid"].(string), newVersion)
	// 	LoggingInfo(callLogInfo, start)
	// 	if res != "" {
	// 		return irs.ClusterInfo{}, err
	// 	}
	// }

	// clusterInfo, err := getClusterInfo(clusterHandler.ClusterClient.Endpoint, clusterHandler.ClusterClient.TokenID, clusterIID.SystemId)
	// if err != nil {
	// 	return irs.ClusterInfo{}, err
	// }

	// return *clusterInfo, nil

	return irs.ClusterInfo{}, nil
}

func getClusterInfo(host string, token string, cluster_id string) (*irs.ClusterInfo, error) {

	// defer func() {
	// 	if r := recover(); r != nil {
	// 		cblogger.Error("getClusterInfo() failed!", r)
	// 	}
	// }()

	// cluster_json_str, err := alibaba.GetCluster(host, token, cluster_id)
	// if err != nil {
	// 	return nil, err
	// }
	// var cluster_json_obj map[string]interface{}
	// json.Unmarshal([]byte(cluster_json_str), &cluster_json_obj)

	// health_status := cluster_json_obj["status"].(string)
	// cluster_status := irs.ClusterActive
	// if health_status == "CREATE_IN_PROGRESS" {
	// 	cluster_status = irs.ClusterCreating
	// } else if health_status == "UPDATE_IN_PROGRESS" {
	// 	cluster_status = irs.ClusterUpdating
	// } else if health_status == "UPDATE_FAILED" {
	// 	cluster_status = irs.ClusterInactive
	// } else if health_status == "DELETE_IN_PROGRESS" {
	// 	cluster_status = irs.ClusterDeleting
	// } else if health_status == "CREATE_COMPLETE" {
	// 	cluster_status = irs.ClusterActive
	// }

	// created_at := cluster_json_obj["created_at"].(string)
	// datetime, err := time.Parse(time.RFC3339, created_at)
	// //RFC3339     = "2006-01-02T15:04:05Z07:00"
	// //datetime, err := time.Parse("2006-01-02T15:04:05+07:00", created_at)
	// if err != nil {
	// 	panic(err)
	// }

	// version := ""
	// if cluster_json_obj["coe_version"] != nil {
	// 	version = cluster_json_obj["coe_version"].(string)
	// }

	// cluster_info := &irs.ClusterInfo{
	// 	IId: irs.IID{
	// 		NameId:   cluster_json_obj["name"].(string),
	// 		SystemId: cluster_json_obj["uuid"].(string),
	// 	},
	// 	Version: version,
	// 	Network: irs.NetworkInfo{
	// 		VpcIID: irs.IID{
	// 			NameId:   "",
	// 			SystemId: cluster_json_obj["fixed_network"].(string),
	// 		},
	// 		SubnetIID: []irs.IID{
	// 			{
	// 				NameId:   "",
	// 				SystemId: cluster_json_obj["fixed_subnet"].(string),
	// 			},
	// 		},
	// 		SecurityGroupIIDs: []irs.IID{
	// 			{
	// 				NameId:   "",
	// 				SystemId: "",
	// 			},
	// 		},
	// 	},
	// 	// NodeGroupList: []irs.NodeGroupInfo{
	// 	// 	{
	// 	// 		IId: irs.IID{
	// 	// 			NameId:   "",
	// 	// 			SystemId: "",
	// 	// 		},
	// 	// 		ImageIID: irs.IID{
	// 	// 			NameId:   "",
	// 	// 			SystemId: "",
	// 	// 		},
	// 	// 		VMSpecName:   "",
	// 	// 		RootDiskType: "",
	// 	// 		RootDiskSize: "",
	// 	// 		KeyPairIID: irs.IID{
	// 	// 			NameId:   "",
	// 	// 			SystemId: "",
	// 	// 		},
	// 	// 		AutoScaling:        false,
	// 	// 		MinNumberNodes:     0,
	// 	// 		MaxNumberNodes:     0,
	// 	// 		DesiredNumberNodes: 0,
	// 	// 		NodeList:           []irs.IID{},
	// 	// 		KeyValueList:       []irs.KeyValue{},
	// 	// 	},
	// 	// },
	// 	Addons: irs.AddonsInfo{
	// 		KeyValueList: []irs.KeyValue{},
	// 	},
	// 	Status:       cluster_status,
	// 	CreatedTime:  datetime,
	// 	KeyValueList: []irs.KeyValue{
	// 		// {
	// 		// 	Key:   "external_network_id",
	// 		// 	Value: cluster.(map[string]interface{})["labels"].(map[string]interface{})["external_network_id"].(string),
	// 		// },
	// 		// {
	// 		// 	Key:   "external_subnet_id_list",
	// 		// 	Value: cluster.(map[string]interface{})["labels"].(map[string]interface{})["external_subnet_id_list"].(string),
	// 		// },
	// 		// {
	// 		// 	Key:   "availability_zone",
	// 		// 	Value: cluster.(map[string]interface{})["labels"].(map[string]interface{})["availability_zone"].(string),
	// 		// },
	// 		// {
	// 		// 	Key:   "node_count",
	// 		// 	Value: strconv.Itoa(int(cluster.(map[string]interface{})["node_count"].(float64))),
	// 		// },
	// 	},
	// }

	// // NodeGroups
	// node_groups_json_str, err := alibaba.GetNodeGroups(host, token, cluster_id)
	// if err != nil {
	// 	return nil, err
	// }

	// var node_groups_json_obj map[string]interface{}
	// json.Unmarshal([]byte(node_groups_json_str), &node_groups_json_obj)
	// node_groups := node_groups_json_obj["nodegroups"].([]interface{})
	// for _, node_group := range node_groups {
	// 	node_group_id := node_group.(map[string]interface{})["uuid"].(string)
	// 	node_group_info, err := getNodeGroupInfo(host, token, cluster_id, node_group_id)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// 	cluster_info.NodeGroupList = append(cluster_info.NodeGroupList, *node_group_info)
	// }

	// return cluster_info, nil

	return nil, nil
}

func getNodeGroupInfo(host string, token string, cluster_id string, node_group_id string) (*irs.NodeGroupInfo, error) {
	// 	defer func() {
	// 		if r := recover(); r != nil {
	// 			cblogger.Error("getNodeGroupInfo() failed!", r)
	// 		}
	// 	}()

	// 	node_group_json_str, err := alibaba.GetNodeGroup(host, token, cluster_id, node_group_id)
	// 	if err != nil {
	// 		return nil, err
	// 	}

	// 	if node_group_json_str == "" {
	// 		return nil, errors.New("not found")
	// 	}

	// 	var node_group_json_obj map[string]interface{}
	// 	json.Unmarshal([]byte(node_group_json_str), &node_group_json_obj)

	// 	auto_scaling, _ := strconv.ParseBool(node_group_json_obj["labels"].(map[string]interface{})["ca_enable"].(string))
	// 	ca_min_node_count, _ := strconv.ParseInt(node_group_json_obj["labels"].(map[string]interface{})["ca_min_node_count"].(string), 10, 32)
	// 	ca_max_node_count, _ := strconv.ParseInt(node_group_json_obj["labels"].(map[string]interface{})["ca_max_node_count"].(string), 10, 32)
	// 	node_count := int(node_group_json_obj["node_count"].(float64))

	// 	node_group_info := irs.NodeGroupInfo{
	// 		IId: irs.IID{
	// 			NameId:   node_group_json_obj["name"].(string),
	// 			SystemId: node_group_json_obj["uuid"].(string),
	// 		},
	// 		ImageIID: irs.IID{
	// 			NameId:   "",
	// 			SystemId: node_group_json_obj["image_id"].(string),
	// 		},
	// 		VMSpecName:      node_group_json_obj["flavor_id"].(string),
	// 		RootDiskType:    node_group_json_obj["labels"].(map[string]interface{})["boot_volume_size"].(string),
	// 		RootDiskSize:    node_group_json_obj["labels"].(map[string]interface{})["boot_volume_type"].(string),
	// 		KeyPairIID:      irs.IID{},
	// 		OnAutoScaling:   auto_scaling,
	// 		MinNodeSize:     int(ca_min_node_count),
	// 		MaxNodeSize:     int(ca_max_node_count),
	// 		DesiredNodeSize: int(node_count),
	// 		NodeList:        []irs.IID{},
	// 		KeyValueList:    []irs.KeyValue{},
	// 	}

	// 	return &node_group_info, nil
	// }

	// func getClusterInfoJSON(clusterInfo irs.ClusterInfo) (string, error) {
	// 	defer func() {
	// 		if r := recover(); r != nil {
	// 			cblogger.Error("getClusterInfoJSON failed", r)
	// 		}
	// 	}()

	// 	fixed_network := clusterInfo.Network.VpcIID.SystemId
	// 	fixed_subnet := clusterInfo.Network.SubnetIID[0].SystemId
	// 	flavor_id := clusterInfo.NodeGroupList[0].VMSpecName
	// 	keypair := clusterInfo.NodeGroupList[0].KeyPairIID.NameId

	// 	availability_zone := ""
	// 	boot_volume_size := clusterInfo.NodeGroupList[0].RootDiskSize
	// 	boot_volume_type := clusterInfo.NodeGroupList[0].RootDiskType

	// 	ca_enable := "false"
	// 	if clusterInfo.NodeGroupList[0].OnAutoScaling {
	// 		ca_enable = "true"
	// 	}
	// 	ca_max_node_count := strconv.Itoa(clusterInfo.NodeGroupList[0].MaxNodeSize)
	// 	ca_min_node_count := strconv.Itoa(clusterInfo.NodeGroupList[0].MinNodeSize)
	// 	external_network_id := ""
	// 	external_subnet_id_list := ""
	// 	kube_tag := clusterInfo.Version
	// 	node_image := clusterInfo.NodeGroupList[0].ImageIID.SystemId
	// 	name := clusterInfo.IId.NameId
	// 	node_count := strconv.Itoa(clusterInfo.NodeGroupList[0].DesiredNodeSize)

	// 	for _, v := range clusterInfo.KeyValueList {
	// 		switch v.Key {
	// 		case "external_network_id":
	// 			external_network_id = v.Value
	// 		case "external_subnet_id_list":
	// 			external_subnet_id_list = v.Value
	// 		}
	// 	}

	// 	for _, v := range clusterInfo.NodeGroupList[0].KeyValueList {
	// 		switch v.Key {
	// 		case "availability_zone":
	// 			availability_zone = v.Value
	// 		}
	// 	}

	// 	temp := `{
	// 		"cluster_template_id": "iaas_console",
	// 		"create_timeout": 60,
	// 		"fixed_network": "%s",
	// 		"fixed_subnet": "%s",
	// 		"flavor_id": "%s",
	// 		"keypair": "%s",
	// 		"labels": {
	// 			"availability_zone": "%s",
	// 			"boot_volume_size": "%s",
	// 			"boot_volume_type": "%s",
	// 			"ca_enable": "%s",
	// 			"ca_max_node_count": "%s",
	// 			"ca_min_node_count": "%s",
	// 			"cert_manager_api": "True",
	// 			"clusterautoscale": "nodegroupfeature",
	// 			"external_network_id": "%s",
	// 			"external_subnet_id_list": "%s",
	// 			"kube_tag": "%s",
	// 			"master_lb_floating_ip_enabled": "true",
	// 			"node_image": "%s",
	// 			"user_script_v2": ""
	// 		},
	// 		"name": "%s",
	// 		"node_count": %s
	// 	}`

	// 	info := fmt.Sprintf(temp, fixed_network, fixed_subnet, flavor_id, keypair, availability_zone, boot_volume_size, boot_volume_type, ca_enable, ca_max_node_count, ca_min_node_count, external_network_id, external_subnet_id_list, kube_tag, node_image, name, node_count)

	// 	return info, nil

	return nil, nil
}

func getNodeGroupJSONString(nodeGroupReqInfo irs.NodeGroupInfo) (string, error) {
	// defer func() {
	// 	if r := recover(); r != nil {
	// 		cblogger.Error("getNodeGroupJSONString failed", r)
	// 	}
	// }()

	// name := nodeGroupReqInfo.IId.NameId
	// node_count := nodeGroupReqInfo.DesiredNodeSize
	// flavor_id := nodeGroupReqInfo.VMSpecName
	// image_id := nodeGroupReqInfo.ImageIID.SystemId

	// boot_volume_size := nodeGroupReqInfo.RootDiskSize
	// boot_volume_type := nodeGroupReqInfo.RootDiskType

	// ca_enable := strconv.FormatBool(nodeGroupReqInfo.OnAutoScaling)
	// ca_max_node_count := strconv.Itoa(nodeGroupReqInfo.MaxNodeSize)
	// ca_min_node_count := strconv.Itoa(nodeGroupReqInfo.MinNodeSize)

	// availability_zone := ""
	// for _, v := range nodeGroupReqInfo.KeyValueList {
	// 	switch v.Key {
	// 	case "availability_zone":
	// 		availability_zone = v.Value
	// 	}
	// }

	// temp := `{
	//     "name": "%s",
	//     "node_count": %d,
	//     "flavor_id": "%s",
	//     "image_id": "%s",
	//     "labels": {
	//         "availability_zone": "%s",
	//         "boot_volume_size": "%s",
	//         "boot_volume_type": "%s",
	//         "ca_enable": "%s",
	// 		"ca_max_node_count": "%s",
	// 		"ca_min_node_count": "%s"
	//     }
	// }`

	// payload := fmt.Sprintf(temp, name, node_count, flavor_id, image_id, availability_zone, boot_volume_size, boot_volume_type, ca_enable, ca_max_node_count, ca_min_node_count)

	// return payload, nil

	return "", nil
}

// getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, "ListCluster()", "ListCluster()")
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

func loggingError(hiscallInfo call.CLOUDLOGSCHEMA, err error) {
	hiscallInfo.ErrorMSG = err.Error()
	tempCalllogger.Info(call.String(hiscallInfo))
}

func loggingInfo(hiscallInfo call.CLOUDLOGSCHEMA, start time.Time) {
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(hiscallInfo))
}
