// Tencent Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Tencent Driver.
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
	request, err := getCreateClusterRequest(clusterHandler, clusterReqInfo)
	if err != nil {
		cblogger.Error(err)
		return irs.ClusterInfo{}, err
	}

	start := call.Start()
	res, err := tencent.CreateCluster(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, request)
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
	// 	res, err := clusterHandler.AddNodeGroup(clusterReqInfo.IId, node_group)
	// 	if err != nil {
	// 		cblogger.Error(err)
	// 		return irs.ClusterInfo{}, err
	// 	}
	// }

	cluster_info, err := getClusterInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, *res.Response.ClusterId)
	if err != nil {
		cblogger.Error(err)
		return irs.ClusterInfo{}, err
	}

	return *cluster_info, nil
}

func (clusterHandler *TencentClusterHandler) ListCluster() ([]*irs.ClusterInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called ListCluster()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, "ListCluster()", "ListCluster()")

	start := call.Start()
	res, err := tencent.GetClusters(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		tempCalllogger.Info(call.String(callLogInfo))
		return nil, err
	}

	cluster_info_list := make([]*irs.ClusterInfo, *res.Response.TotalCount)
	for i, cluster := range res.Response.Clusters {
		cluster_info_list[i], err = getClusterInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, *cluster.ClusterId)
		if err != nil {
			cblogger.Error(err)
			return nil, err
		}
	}

	return cluster_info_list, nil
}

func (clusterHandler *TencentClusterHandler) GetCluster(clusterIID irs.IID) (irs.ClusterInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called GetCluster()")
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

func (clusterHandler *TencentClusterHandler) DeleteCluster(clusterIID irs.IID) (bool, error) {
	cblogger.Info("Tencent Cloud Driver: called DeleteCluster()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "DeleteCluster()")

	start := call.Start()
	res, err := tencent.DeleteCluster(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		tempCalllogger.Info(call.String(callLogInfo))
		return false, err
	}
	cblogger.Info("DeleteCluster(): ", res)

	return true, nil
}

func (clusterHandler *TencentClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (irs.NodeGroupInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called AddNodeGroup()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "AddNodeGroup()")

	// 노드 그룹 생성 요청 변환
	// get cluster info. to get security_group_id
	request, err := getNodeGroupRequest(clusterHandler, clusterIID.SystemId, nodeGroupReqInfo)
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}

	start := call.Start()
	response, err := tencent.CreateNodeGroup(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, request)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		tempCalllogger.Info(call.String(callLogInfo))
		return irs.NodeGroupInfo{}, err
	}

	node_group_info, err := getNodeGroupInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, *response.Response.NodePoolId)
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}

	return *node_group_info, nil
}

func (clusterHandler *TencentClusterHandler) ListNodeGroup(clusterIID irs.IID) ([]*irs.NodeGroupInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called ListNodeGroup()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "ListNodeGroup()")

	node_group_info_list := []*irs.NodeGroupInfo{}

	start := call.Start()
	res, err := tencent.ListNodeGroup(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		tempCalllogger.Info(call.String(callLogInfo))
		return node_group_info_list, err
	}

	for _, node_group := range res.Response.NodePoolSet {
		node_group_info, err := getNodeGroupInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, *node_group.NodePoolId)
		if err != nil {
			cblogger.Error(err)
			return nil, err
		}
		node_group_info_list = append(node_group_info_list, node_group_info)
	}

	return node_group_info_list, nil
}

func (clusterHandler *TencentClusterHandler) GetNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (irs.NodeGroupInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called GetNodeGroup()")
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

func (clusterHandler *TencentClusterHandler) SetNodeGroupAutoScaling(clusterIID irs.IID, nodeGroupIID irs.IID, on bool) (bool, error) {
	cblogger.Info("Tencent Cloud Driver: called SetNodeGroupAutoScaling()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "SetNodeGroupAutoScaling()")

	start := call.Start()
	temp, err := tencent.SetNodeGroupAutoScaling(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodeGroupIID.SystemId, on)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		tempCalllogger.Info(call.String(callLogInfo))
		return false, err
	}
	cblogger.Info(temp.ToJsonString())

	return true, nil
}

func (clusterHandler *TencentClusterHandler) ChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID, desiredNodeSize int, minNodeSize int, maxNodeSize int) (irs.NodeGroupInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called ChangeNodeGroupScaling()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "ChangeNodeGroupScaling()")

	nodegroup, err := tencent.GetNodeGroup(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodeGroupIID.SystemId)
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}

	start := call.Start()
	temp, err := tencent.ChangeNodeGroupScaling(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, *nodegroup.Response.NodePool.AutoscalingGroupId, uint64(desiredNodeSize), uint64(minNodeSize), uint64(maxNodeSize))
	callLogInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(callLogInfo))
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		tempCalllogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
	}
	cblogger.Info(temp.ToJsonString())

	node_group_info, err := getNodeGroupInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodeGroupIID.SystemId)
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}

	return *node_group_info, nil
}

func (clusterHandler *TencentClusterHandler) RemoveNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (bool, error) {
	cblogger.Info("Tencent Cloud Driver: called RemoveNodeGroup()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "RemoveNodeGroup()")

	start := call.Start()
	res, err := tencent.DeleteNodeGroup(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodeGroupIID.SystemId)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		tempCalllogger.Info(call.String(callLogInfo))
		return false, err
	}
	cblogger.Info(res.ToJsonString())

	return true, nil
}

func (clusterHandler *TencentClusterHandler) UpgradeCluster(clusterIID irs.IID, newVersion string) (irs.ClusterInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called UpgradeCluster()")
	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "UpgradeCluster()")

	start := call.Start()
	res, err := tencent.UpgradeCluster(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, newVersion)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	tempCalllogger.Info(call.String(callLogInfo))
	if err != nil {
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		tempCalllogger.Info(call.String(callLogInfo))
		return irs.ClusterInfo{}, err
	}
	cblogger.Info(res.ToJsonString())

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
			err = fmt.Errorf("getNodeGroupInfo: %v", r)
		}
	}()

	res, err := tencent.GetCluster(access_key, access_secret, region_id, cluster_id)
	if err != nil {
		return nil, err
	}

	if *res.Response.TotalCount == 0 {
		return nil, fmt.Errorf("cluster[%s] does not exist", cluster_id)
	}

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
	// else if strings.EqualFold(health_status, "") { // tencent has no "delete" state
	// cluster_status = irs.ClusterDeleting
	//}

	created_at := *res.Response.Clusters[0].CreatedTime // "2022-09-09T13:10:06Z",
	datetime, err := time.Parse(time.RFC3339, created_at)
	if err != nil {
		panic(err)
	}

	cluster_info := &irs.ClusterInfo{
		IId: irs.IID{
			NameId:   *res.Response.Clusters[0].ClusterName,
			SystemId: *res.Response.Clusters[0].ClusterId,
		},
		Version: *res.Response.Clusters[0].ClusterVersion,
		Network: irs.NetworkInfo{
			VpcIID: irs.IID{
				NameId:   "",
				SystemId: *res.Response.Clusters[0].ClusterNetworkSettings.VpcId,
			},
			SubnetIID: []irs.IID{
				{
					NameId:   "",
					SystemId: *res.Response.Clusters[0].ClusterNetworkSettings.Subnets[0],
				},
			},
		},
		Status:      cluster_status,
		CreatedTime: datetime,
		// KeyValueList: []irs.KeyValue{}, // flatten data 입력하기
	}

	// k,v 추출 & 추가
	// KeyValueList: []irs.KeyValue{}, // flatten data 입력하기
	temp, err := json.Marshal(*res.Response.Clusters[0])
	if err != nil {
		panic(err)
	}
	var json_obj map[string]interface{}
	json.Unmarshal([]byte(temp), &json_obj)

	flat, err := flatten.Flatten(json_obj, "", flatten.DotStyle)
	if err != nil {
		return nil, err
	}
	for k, v := range flat {
		temp := fmt.Sprintf("%v", v)
		cluster_info.KeyValueList = append(cluster_info.KeyValueList, irs.KeyValue{Key: k, Value: temp})
	}

	// NodeGroups
	res2, err := tencent.ListNodeGroup(access_key, access_secret, region_id, cluster_id)
	if err != nil {
		return nil, err
	}

	for _, nodepool := range res2.Response.NodePoolSet {
		node_group_info, err := getNodeGroupInfo(access_key, access_secret, region_id, cluster_id, *nodepool.NodePoolId)
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
			err = fmt.Errorf("getNodeGroupInfo: %v", r)
		}
	}()

	res, err := tencent.GetNodeGroup(access_key, access_secret, region_id, cluster_id, node_group_id)
	if err != nil {
		return nil, err
	}

	launch_config, err := tencent.GetLaunchConfiguration(access_key, access_secret, region_id, *res.Response.NodePool.LaunchConfigurationId)
	if err != nil {
		return nil, err
	}

	auto_scaling_group, err := tencent.GetAutoScalingGroup(access_key, access_secret, region_id, *res.Response.NodePool.AutoscalingGroupId)
	if err != nil {
		return nil, err
	}

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

	// add key value list
	temp, err := json.Marshal(*res.Response.NodePool)
	if err != nil {
		panic(err)
	}
	var json_obj map[string]interface{}
	json.Unmarshal([]byte(temp), &json_obj)
	flat, err := flatten.Flatten(json_obj, "", flatten.DotStyle)
	if err != nil {
		return nil, err
	}
	for k, v := range flat {
		temp := fmt.Sprintf("%v", v)
		node_group_info.KeyValueList = append(node_group_info.KeyValueList, irs.KeyValue{Key: k, Value: temp})
	}

	return &node_group_info, err
}

func getCreateClusterRequest(clusterHandler *TencentClusterHandler, clusterInfo irs.ClusterInfo) (*tke.CreateClusterRequest, error) {

	var err error = nil
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("getCreateClusterRequest: %v", r)
		}
	}()

	// 172.X.0.0.16: X Range:16, 17, ... , 31
	m_cidr := make(map[string]bool)
	for i := 16; i < 32; i++ {
		m_cidr[fmt.Sprintf("172.%v.0.0/16", i)] = true
	}

	clusters, err := clusterHandler.ListCluster()
	if err != nil {
		return nil, err
	}
	for _, cluster := range clusters {
		for _, v := range cluster.KeyValueList {
			if v.Key == "ClusterNetworkSettings.ClusterCIDR" {
				delete(m_cidr, v.Value)
			}
		}
	}

	cidr_list := []string{}
	for k := range m_cidr {
		cidr_list = append(cidr_list, k)
	}

	request := tke.NewCreateClusterRequest()
	request.ClusterCIDRSettings = &tke.ClusterCIDRSettings{
		ClusterCIDR:  common.StringPtr(cidr_list[0]), // 172.X.0.0.16: X Range:16, 17, ... , 31
		EniSubnetIds: common.StringPtrs([]string{clusterInfo.Network.SubnetIID[0].SystemId}),
	}
	request.ClusterBasicSettings = &tke.ClusterBasicSettings{
		ClusterName:    common.StringPtr(clusterInfo.IId.NameId),
		VpcId:          common.StringPtr(clusterInfo.Network.VpcIID.SystemId),
		ClusterVersion: common.StringPtr(clusterInfo.Version), //option, version: 1.22.5
	}
	request.ClusterType = common.StringPtr("MANAGED_CLUSTER") //default value

	return request, err
}

func getNodeGroupRequest(clusterHandler *TencentClusterHandler, cluster_id string, nodeGroupReqInfo irs.NodeGroupInfo) (*tke.CreateClusterNodePoolRequest, error) {

	var err error = nil
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("getNodeGroupRequest: %v", r)
		}
	}()

	cluster, res := clusterHandler.GetCluster(irs.IID{SystemId: cluster_id})
	if res != nil {
		return nil, res
	}
	vpc_id := cluster.Network.VpcIID.SystemId
	subnet_id := cluster.Network.SubnetIID[0].SystemId

	response, err := tencent.DescribeSecurityGroups(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region)
	if err != nil {
		println(err)
	}

	security_group_id := ""
	for _, group := range response.Response.SecurityGroupSet {
		if *group.IsDefault {
			security_group_id = *group.SecurityGroupId
			break
		}
	}

	// '{"LaunchConfigurationName":"name","InstanceType":"S3.MEDIUM2","ImageId":"img-pi0ii46r"}'
	// ImageId를 설정하면 에러 발생, 설정안됨.
	launch_config_json_str := `{
		"InstanceType": "%s",
		"SecurityGroupIds": ["%s"]
	}`
	launch_config_json_str = fmt.Sprintf(launch_config_json_str, nodeGroupReqInfo.VMSpecName, security_group_id)

	auto_scaling_group_json_str := `{
		"MinSize": %d,
		"MaxSize": %d,			
		"DesiredCapacity": %d,
		"VpcId": "%s",
		"SubnetIds": ["%s"]
	}`
	auto_scaling_group_json_str = fmt.Sprintf(auto_scaling_group_json_str, 0, 3, 1, vpc_id, subnet_id)

	disk_size, _ := strconv.ParseInt(nodeGroupReqInfo.RootDiskSize, 10, 64)
	request := tke.NewCreateClusterNodePoolRequest()
	request.Name = common.StringPtr(nodeGroupReqInfo.IId.NameId)
	request.ClusterId = common.StringPtr(cluster_id)
	request.LaunchConfigurePara = common.StringPtr(launch_config_json_str)
	request.AutoScalingGroupPara = common.StringPtr(auto_scaling_group_json_str)
	request.EnableAutoscale = common.BoolPtr(nodeGroupReqInfo.OnAutoScaling)
	request.InstanceAdvancedSettings = &tke.InstanceAdvancedSettings{
		DataDisks: []*tke.DataDisk{
			{
				DiskType: common.StringPtr(nodeGroupReqInfo.RootDiskType), //ex. "CLOUD_PREMIUM"
				DiskSize: common.Int64Ptr(disk_size),                      //ex. 50
			},
		},
	}

	return request, err
}

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
