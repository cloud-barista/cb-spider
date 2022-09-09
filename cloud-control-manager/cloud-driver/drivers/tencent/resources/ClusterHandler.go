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
	"strings"
	"sync"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/tencent/utils/tencent"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"github.com/jeremywohl/flatten"
	"github.com/sirupsen/logrus"
	tke "github.com/tencentcloud/tencentcloud-sdk-go-intl-en/tencentcloud/tke/v20180525"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
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

func (clusterHandler *TencentClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (irs.ClusterInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called CreateCluster()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, "CreateCluster()", "CreateCluster()")

	// 클러스터 생성 요청 변환
	request, err := getCreateClusterRequest(clusterReqInfo)
	if err != nil {
		cblogger.Error(err)
		return irs.ClusterInfo{}, err
	}

	start := call.Start()
	res, err := tencent.CreateCluster(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, request)
	loggingInfo(callLogInfo, start)
	if err != nil {
		cblogger.Error(err)
		loggingError(callLogInfo, err)
		return irs.ClusterInfo{}, err
	}
	println(res.ToJsonString())

	// var response_json_obj map[string]interface{}
	// json.Unmarshal([]byte(response_json_str), &response_json_obj)
	// cluster_id := response_json_obj["cluster_id"].(string)
	cluster_info, err := getClusterInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, *res.Response.ClusterId)
	if err != nil {
		return irs.ClusterInfo{}, err
	}

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

	return *cluster_info, nil
}

func (clusterHandler *TencentClusterHandler) ListCluster() ([]*irs.ClusterInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called ListCluster()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, "ListCluster()", "ListCluster()")

	start := call.Start()
	res, err := tencent.GetClusters(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region)
	loggingInfo(callLogInfo, start)
	if err != nil {
		return nil, err
	}

	cluster_info_list := make([]*irs.ClusterInfo, *res.Response.TotalCount)
	for i, cluster := range res.Response.Clusters {
		cluster_info_list[i], err = getClusterInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, *cluster.ClusterId)
		if err != nil {
			return nil, err
		}
	}

	return cluster_info_list, nil
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

func getClusterInfo(access_key string, access_secret string, region_id string, cluster_id string) (*irs.ClusterInfo, error) {
	var err error = nil
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("getClusterInfo() -> %v", r)
		}
	}()

	res, err := tencent.GetCluster(access_key, access_secret, region_id, cluster_id)
	if err != nil {
		return nil, err
	}

	if *res.Response.TotalCount == 0 {
		return nil, fmt.Errorf("cluster[%s] does not exist", cluster_id)
	}

	printFlattenJSON(res)

	// // k,v 추출
	// // k,v 변환 규칙 작성 [k,v]:[ClusterInfo.k, ClusterInfo.v]
	// // 변환 규칙에 따라 k,v 변환

	// https://intl.cloud.tencent.com/document/api/457/32022#ClusterStatus
	// Cluster status (Running, Creating, Idling or Abnormal)

	health_status := *res.Response.Clusters[0].ClusterStatus
	cluster_status := irs.ClusterActive
	if strings.EqualFold(health_status, "Creating") {
		cluster_status = irs.ClusterCreating
	} else if strings.EqualFold(health_status, "Creating") {
		cluster_status = irs.ClusterUpdating
	} else if strings.EqualFold(health_status, "Abnormal") {
		cluster_status = irs.ClusterInactive
	} else if strings.EqualFold(health_status, "Running") {
		cluster_status = irs.ClusterActive
	}
	// } else if strings.EqualFold(health_status, "") { // tencent has no "delete" state
	// // 	cluster_status = irs.ClusterDeleting
	println(cluster_status)

	// "2022-09-09T13:10:06Z",
	created_at := *res.Response.Clusters[0].CreatedTime // 2022-09-08T09:02:16+08:00,
	datetime, err := time.Parse(time.RFC3339, created_at)
	if err != nil {
		panic(err)
	}

	// "Response.Clusters.0.ClusterName": "cluster-x1",
	// "Response.Clusters.0.ClusterVersion": "1.22.5",
	// "Response.Clusters.0.ClusterNetworkSettings.VpcId": "vpc-q1c6fr9e",
	// "Response.Clusters.0.ClusterStatus": "Creating",
	// "Response.Clusters.0.CreatedTime": "2022-09-09T13:10:06Z",

	cluster_info := &irs.ClusterInfo{
		IId: irs.IID{
			NameId:   *res.Response.Clusters[0].ClusterName,
			SystemId: *res.Response.Clusters[0].ClusterId,
		},
		Version: *res.Response.Clusters[0].ClusterVersion,
		Network: irs.NetworkInfo{
			VpcIID: irs.IID{
				NameId:   "",
				SystemId: *res.Response.Clusters[0].ClusterVersion,
			},
		},
		Status:      cluster_status,
		CreatedTime: datetime,
		// KeyValueList: []irs.KeyValue{}, // flatten data 입력하기
	}
	println(cluster_info)

	// NodeGroups
	res2, err := tencent.ListNodeGroup(access_key, access_secret, region_id, cluster_id)
	if err != nil {
		return nil, err
	}
	print(res.ToJsonString())

	// // k,v 추출
	// // k,v 변환 규칙 작성 [k,v]:[NodeGroup.k, NodeGroup.v]
	// // 변환 규칙에 따라 k,v 변환
	// flat, err = flatten.FlattenString(node_groups_json_str, "", flatten.DotStyle)
	// if err != nil {
	// 	return nil, err
	// }
	// println(flat)

	for _, nodepool := range res2.Response.NodePoolSet {
		node_group_info, err := getNodeGroupInfo(access_key, access_secret, region_id, cluster_id, *nodepool.NodePoolId)
		if err != nil {
			return nil, err
		}
		cluster_info.NodeGroupList = append(cluster_info.NodeGroupList, *node_group_info)
	}

	//return cluster_info, nil

	return cluster_info, nil
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

func getNodeGroupInfo(access_key, access_secret, region_id, cluster_id, node_group_id string) (*irs.NodeGroupInfo, error) {
	var err error = nil
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("getNodeGroupInfo() -> %v", r)
		}
	}()

	res, err := tencent.GetNodeGroup(access_key, access_secret, region_id, cluster_id, node_group_id)
	if err != nil {
		return nil, err
	}
	printFlattenJSON(res)

	launch_config, err := tencent.GetLaunchConfiguration(access_key, access_secret, region_id, *res.Response.NodePool.LaunchConfigurationId)
	if err != nil {
		return nil, err
	}
	printFlattenJSON(launch_config)

	auto_scaling_group, err := tencent.GetAutoScalingGroup(access_key, access_secret, region_id, *res.Response.NodePool.AutoscalingGroupId)
	if err != nil {
		return nil, err
	}
	printFlattenJSON(auto_scaling_group)

	// nodepool LifeState
	// The lifecycle state of the current node pool.
	// Valid values: creating, normal, updating, deleting, and deleted.
	health_status := *res.Response.NodePool.LifeState
	status := irs.NodeGroupActive
	if strings.EqualFold(health_status, "normal") {
		status = irs.NodeGroupActive
	} else if strings.EqualFold(health_status, "creating") {
		status = irs.NodeGroupUpdating
	} else if strings.EqualFold(health_status, "removing") {
		status = irs.NodeGroupUpdating // removing is a kind of updating?
	} else if strings.EqualFold(health_status, "deleting") {
		status = irs.NodeGroupDeleting
	} else if strings.EqualFold(health_status, "updating") {
		status = irs.NodeGroupUpdating
	}

	println(status)

	auto_scale_enalbed := false
	if strings.EqualFold("Response.AutoScalingGroupSet.0.EnabledStatus", "ENABLED") {
		auto_scale_enalbed = true
	}

	node_group_info := irs.NodeGroupInfo{
		IId: irs.IID{
			NameId:   *res.Response.NodePool.Name,
			SystemId: *res.Response.NodePool.NodePoolId,
		},
		ImageIID: irs.IID{
			NameId:   "",
			SystemId: *launch_config.Response.LaunchConfigurationSet[0].ImageId,
		},
		VMSpecName:      *launch_config.Response.LaunchConfigurationSet[0].InstanceType,
		RootDiskType:    *launch_config.Response.LaunchConfigurationSet[0].SystemDisk.DiskType,
		RootDiskSize:    fmt.Sprintf("%d", *launch_config.Response.LaunchConfigurationSet[0].SystemDisk.DiskSize),
		KeyPairIID:      irs.IID{NameId: "", SystemId: ""}, // not available
		Status:          status,
		OnAutoScaling:   auto_scale_enalbed,
		MinNodeSize:     int(*auto_scaling_group.Response.AutoScalingGroupSet[0].MinSize),
		MaxNodeSize:     int(*auto_scaling_group.Response.AutoScalingGroupSet[0].MaxSize),
		DesiredNodeSize: int(*auto_scaling_group.Response.AutoScalingGroupSet[0].DesiredCapacity),
		NodeList:        []irs.IID{},      // to be implemented
		KeyValueList:    []irs.KeyValue{}, // to be implemented
	}

	return &node_group_info, nil
}

func getCreateClusterRequest(clusterInfo irs.ClusterInfo) (*tke.CreateClusterRequest, error) {

	var err error = nil
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered: %v", r)
		}
	}()

	// clusterInfo := irs.ClusterInfo{
	// 	IId: irs.IID{
	// 		NameId:   "cluster-x1",
	// 		SystemId: "",
	// 	},
	// 	Version: "1.22.5",
	// 	Network: irs.NetworkInfo{
	// 		VpcIID: irs.IID{NameId: "", SystemId: "vpc-q1c6fr9e"},
	// 	},
	// 	KeyValueList: []irs.KeyValue{
	// 		{
	// 			Key:   "cluster_cidr", // 조회가능한 값이면, 내부에서 처리하는 코드 추가
	// 			Value: "172.20.0.0/16",
	// 		},
	// 	},
	// }

	cluster_cidr := "" // 172.X.0.0.16: X Range:16, 17, ... , 31
	for _, v := range clusterInfo.KeyValueList {
		switch v.Key {
		case "cluster_cidr":
			cluster_cidr = v.Value
		}
	}

	request := tke.NewCreateClusterRequest()
	request.ClusterCIDRSettings = &tke.ClusterCIDRSettings{
		ClusterCIDR: common.StringPtr(cluster_cidr), // 172.X.0.0.16: X Range:16, 17, ... , 31
	}
	request.ClusterBasicSettings = &tke.ClusterBasicSettings{
		ClusterName:    common.StringPtr(clusterInfo.IId.NameId),
		VpcId:          common.StringPtr(clusterInfo.Network.VpcIID.SystemId),
		ClusterVersion: common.StringPtr(clusterInfo.Version), //option, version: 1.22.5
	}
	request.ClusterType = common.StringPtr("MANAGED_CLUSTER") //default value

	return request, err
}

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

// getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, "ListCluster()", "ListCluster()")
func getCallLogScheme(region string, resourceType call.RES_TYPE, resourceName string, apiName string) call.CLOUDLOGSCHEMA {
	cblogger.Info(fmt.Sprintf("Call %s %s", call.TENCENT, apiName))
	return call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
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
