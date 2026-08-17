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
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/tencent/utils/tencent"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tke "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/tke/v20180525"
)

// calllogger
// Used before creating a common logger
// var once sync.Once
// var calllogger *logrus.Logger

// func init() {
// 	once.Do(func() {
// 		calllogger = call.GetLogger("HISCALL")
// 	})
// }

const (
	defaultContainerRuntime = "containerd"

	// Tencent TKE has no first-class field to attach a Security Group to a cluster,
	// and ClusterNetworkSettings.SubnetId is only populated for CiliumOverlay/CDC
	// clusters. Both values are instead persisted as structured cluster Tags so they
	// can be reliably read back via getClusterInfo(); these keys are excluded from the
	// user-facing TagList (see getClusterInfo()).
	tagKeySecurityGroupID = "CB-SPIDER-PMKS-SECURITYGROUP-ID"
	tagKeySubnetID        = "CB-SPIDER-PMKS-SUBNET-ID"
)

type TencentClusterHandler struct {
	RegionInfo     idrv.RegionInfo
	CredentialInfo idrv.CredentialInfo
}

func (clusterHandler *TencentClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (irs.ClusterInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called CreateCluster()")
	callLogInfo := GetCallLogScheme(clusterHandler.RegionInfo, call.CLUSTER, "CreateCluster()", "CreateCluster()")

	start := call.Start()

	//
	// Validation
	//
	err := validateAtCreateCluster(clusterReqInfo)
	if err != nil {
		err = fmt.Errorf("Failed to Create Cluster :  %v", err)
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		return irs.ClusterInfo{}, err
	}

	err = validateClusterVersion(clusterHandler, clusterReqInfo.Version)
	if err != nil {
		err = fmt.Errorf("Failed to Create Cluster :  %v", err)
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		return irs.ClusterInfo{}, err
	}

	// Convert cluster creation request
	request, err := getCreateClusterRequest(clusterHandler, clusterReqInfo)
	if err != nil {
		err := fmt.Errorf("Failed to Get Create Cluster Request :  %v", err)
		cblogger.Error(err)
		return irs.ClusterInfo{}, err
	}
	res, err := tencent.CreateCluster(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, request)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Create Cluster :  %v", err)
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		return irs.ClusterInfo{}, err
	}
	calllogger.Info(call.String(callLogInfo))

	// If there is NodeGroup creation information, attempt creation.
	// Currently, creation is not attempted. If decided to create, uncomment below.
	// Reason:
	// - NodeGroup creation is possible only after Cluster creation is complete.
	// - Cluster creation takes at least 10 minutes.
	// - Must wait until success before attempting creation.
	// for _, node_group := range clusterReqInfo.NodeGroupList {
	// 	res, err := clusterHandler.AddNodeGroup(clusterReqInfo.IId, node_group)
	// 	if err != nil {
	// 		cblogger.Error(err)
	// 		return irs.ClusterInfo{}, err
	// 	}
	// }

	cluster_info, err := getClusterInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, *res.Response.ClusterId)
	if err != nil {
		err := fmt.Errorf("Failed to Get ClusterInfo :  %v", err)
		cblogger.Error(err)
		return irs.ClusterInfo{}, err
	}

	return *cluster_info, nil
}

func (clusterHandler *TencentClusterHandler) ListCluster() ([]*irs.ClusterInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called ListCluster()")
	callLogInfo := GetCallLogScheme(clusterHandler.RegionInfo, call.CLUSTER, "ListCluster()", "ListCluster()")

	start := call.Start()
	res, err := tencent.GetClusters(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Get Clusters :  %v", err)
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		return nil, err
	}
	calllogger.Info(call.String(callLogInfo))

	cluster_info_list := make([]*irs.ClusterInfo, 0, len(res.Response.Clusters))
	for _, cluster := range res.Response.Clusters {
		clusterInfo, err := getClusterInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, *cluster.ClusterId)
		if err != nil {
			err := fmt.Errorf("Failed to Get ClusterInfo :  %v", err)
			cblogger.Error(err)
			return nil, err
		}
		cluster_info_list = append(cluster_info_list, clusterInfo)
	}

	return cluster_info_list, nil
}

func (clusterHandler *TencentClusterHandler) GetCluster(clusterIID irs.IID) (irs.ClusterInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called GetCluster()")
	callLogInfo := GetCallLogScheme(clusterHandler.RegionInfo, call.CLUSTER, clusterIID.NameId, "GetCluster()")

	start := call.Start()
	cluster_info, err := getClusterInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Get ClusterInfo :  %v", err)
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		return irs.ClusterInfo{}, err
	}
	calllogger.Info(call.String(callLogInfo))

	return *cluster_info, nil
}

// GenerateClusterToken generates a token for cluster authentication
// Tencent Cloud does not support dynamic token generation yet
func (clusterHandler *TencentClusterHandler) GenerateClusterToken(clusterIID irs.IID) (string, error) {
	return "", fmt.Errorf("GenerateClusterToken is not supported for Tencent Cloud clusters yet")
}

func (clusterHandler *TencentClusterHandler) DeleteCluster(clusterIID irs.IID) (bool, error) {
	cblogger.Info("Tencent Cloud Driver: called DeleteCluster()")
	callLogInfo := GetCallLogScheme(clusterHandler.RegionInfo, call.CLUSTER, clusterIID.NameId, "DeleteCluster()")

	start := call.Start()
	res, err := tencent.DeleteCluster(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Delete Cluster :  %v", err)
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		return false, err
	}
	cblogger.Info("DeleteCluster(): ", res)
	calllogger.Info(call.String(callLogInfo))

	return true, nil
}

func (clusterHandler *TencentClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (irs.NodeGroupInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called AddNodeGroup()")
	callLogInfo := GetCallLogScheme(clusterHandler.RegionInfo, call.CLUSTER, clusterIID.NameId, "AddNodeGroup()")

	start := call.Start()

	//
	// Validation
	//
	err := validateAtAddNodeGroup(clusterIID, nodeGroupReqInfo)
	if err != nil {
		err := fmt.Errorf("Failed to Add Node Group :  %v", err)
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		return irs.NodeGroupInfo{}, err
	}

	// Convert node group creation request
	// get cluster info. to get security_group_id
	request, err := getNodeGroupRequest(clusterHandler, clusterIID.SystemId, nodeGroupReqInfo)
	if err != nil {
		err := fmt.Errorf("Failed to Get Node Group Request :  %v", err)
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}
	response, err := tencent.CreateNodeGroup(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, request)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Add Node Group :  %v", err)
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		return irs.NodeGroupInfo{}, err
	}
	calllogger.Info(call.String(callLogInfo))

	node_group_info, err := getNodeGroupInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, *response.Response.NodePoolId)
	if err != nil {
		err := fmt.Errorf("Failed to Get Node Group Info :  %v", err)
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}

	return *node_group_info, nil
}

// func (clusterHandler *TencentClusterHandler) ListNodeGroup(clusterIID irs.IID) ([]*irs.NodeGroupInfo, error) {
// 	cblogger.Info("Tencent Cloud Driver: called ListNodeGroup()")
// 	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "ListNodeGroup()")

// 	start := call.Start()
// 	node_group_info_list := []*irs.NodeGroupInfo{}
// 	res, err := tencent.ListNodeGroup(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId)
// 	callLogInfo.ElapsedTime = call.Elapsed(start)
// 	if err != nil {
// 		err := fmt.Errorf("Failed to List Node Group :  %v", err)
// 		cblogger.Error(err)
// 		callLogInfo.ErrorMSG = err.Error()
// 		calllogger.Error(call.String(callLogInfo))
// 		return node_group_info_list, err
// 	}
// 	calllogger.Info(call.String(callLogInfo))

// 	for _, node_group := range res.Response.NodePoolSet {
// 		node_group_info, err := getNodeGroupInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, *node_group.NodePoolId)
// 		if err != nil {
// 			err := fmt.Errorf("Failed to Get Node Group Info:  %v", err)
// 			cblogger.Error(err)
// 			return nil, err
// 		}
// 		node_group_info_list = append(node_group_info_list, node_group_info)
// 	}

// 	return node_group_info_list, nil
// }

// func (clusterHandler *TencentClusterHandler) GetNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (irs.NodeGroupInfo, error) {
// 	cblogger.Info("Tencent Cloud Driver: called GetNodeGroup()")
// 	callLogInfo := getCallLogScheme(clusterHandler.RegionInfo.Region, call.CLUSTER, clusterIID.NameId, "GetNodeGroup()")

// 	start := call.Start()
// 	temp, err := getNodeGroupInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodeGroupIID.SystemId)
// 	callLogInfo.ElapsedTime = call.Elapsed(start)
// 	if err != nil {
// 		err := fmt.Errorf("Failed to Get Node Group Info:  %v", err)
// 		cblogger.Error(err)
// 		callLogInfo.ErrorMSG = err.Error()
// 		calllogger.Error(call.String(callLogInfo))
// 		return irs.NodeGroupInfo{}, err
// 	}
// 	calllogger.Info(call.String(callLogInfo))

// 	return *temp, nil
// }

func (clusterHandler *TencentClusterHandler) SetNodeGroupAutoScaling(clusterIID irs.IID, nodeGroupIID irs.IID, on bool) (bool, error) {
	cblogger.Info("Tencent Cloud Driver: called SetNodeGroupAutoScaling()")
	callLogInfo := GetCallLogScheme(clusterHandler.RegionInfo, call.CLUSTER, clusterIID.NameId, "SetNodeGroupAutoScaling()")

	start := call.Start()
	temp, err := tencent.SetNodeGroupAutoScaling(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodeGroupIID.SystemId, on)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Set Node Group AutoScaling:  %v", err)
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		return false, err
	}
	cblogger.Debug(temp.ToJsonString())
	calllogger.Info(call.String(callLogInfo))

	return true, nil
}

func (clusterHandler *TencentClusterHandler) ChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID, desiredNodeSize int, minNodeSize int, maxNodeSize int) (irs.NodeGroupInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called ChangeNodeGroupScaling()")
	callLogInfo := GetCallLogScheme(clusterHandler.RegionInfo, call.CLUSTER, clusterIID.NameId, "ChangeNodeGroupScaling()")

	start := call.Start()

	//
	// Validation
	//
	err := validateAtChangeNodeGroupScaling(clusterIID, nodeGroupIID, minNodeSize, maxNodeSize)
	if err != nil {
		err := fmt.Errorf("Failed to Change Node Group Scaling:  %v", err)
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		return irs.NodeGroupInfo{}, err
	}

	nodegroup, err := tencent.GetNodeGroup(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodeGroupIID.SystemId)
	if err != nil {
		err := fmt.Errorf("Failed to Get Node Group:  %v", err)
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}
	temp, err := tencent.ChangeNodeGroupScaling(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, *nodegroup.Response.NodePool.AutoscalingGroupId, uint64(desiredNodeSize), uint64(minNodeSize), uint64(maxNodeSize))
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Change Node Group Scaling:  %v", err)
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		return irs.NodeGroupInfo{}, err
	}
	cblogger.Debug(temp.ToJsonString())
	calllogger.Info(call.String(callLogInfo))

	node_group_info, err := getNodeGroupInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodeGroupIID.SystemId)
	if err != nil {
		err := fmt.Errorf("Failed to Get NodeGroupInfo:  %v", err)
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}

	return *node_group_info, nil
}

func (clusterHandler *TencentClusterHandler) RemoveNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (bool, error) {
	cblogger.Info("Tencent Cloud Driver: called RemoveNodeGroup()")
	callLogInfo := GetCallLogScheme(clusterHandler.RegionInfo, call.CLUSTER, clusterIID.NameId, "RemoveNodeGroup()")

	start := call.Start()
	res, err := tencent.DeleteNodeGroup(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, nodeGroupIID.SystemId)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Delete NodeGroup:  %v", err)
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		return false, err
	}
	cblogger.Debug(res.ToJsonString())
	calllogger.Info(call.String(callLogInfo))

	return true, nil
}

func (clusterHandler *TencentClusterHandler) UpgradeCluster(clusterIID irs.IID, newVersion string) (irs.ClusterInfo, error) {
	cblogger.Info("Tencent Cloud Driver: called UpgradeCluster()")
	callLogInfo := GetCallLogScheme(clusterHandler.RegionInfo, call.CLUSTER, clusterIID.NameId, "UpgradeCluster()")

	start := call.Start()
	res, err := tencent.UpgradeCluster(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId, newVersion)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Upgrade Cluster:  %v", err)
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		return irs.ClusterInfo{}, err
	}
	cblogger.Debug(res.ToJsonString())
	calllogger.Info(call.String(callLogInfo))

	clusterInfo, err := getClusterInfo(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region, clusterIID.SystemId)
	if err != nil {
		err := fmt.Errorf("Failed to Get ClusterInfo:  %v", err)
		cblogger.Error(err)
		return irs.ClusterInfo{}, err
	}

	return *clusterInfo, nil
}

// extractNetworkIDsFromTags reads back the SecurityGroup/Subnet IDs that
// getCreateClusterRequest() stored as internal cluster Tags.
func extractNetworkIDsFromTags(tagSpecs []*tke.TagSpecification) (securityGroupID string, subnetID string) {
	for _, tagSpec := range tagSpecs {
		if tagSpec == nil {
			continue
		}
		for _, tag := range tagSpec.Tags {
			if tag == nil || tag.Key == nil || tag.Value == nil {
				continue
			}
			switch *tag.Key {
			case tagKeySecurityGroupID:
				securityGroupID = *tag.Value
			case tagKeySubnetID:
				subnetID = *tag.Value
			}
		}
	}
	return securityGroupID, subnetID
}

func getClusterInfo(access_key string, access_secret string, region_id string, cluster_id string) (clusterInfo *irs.ClusterInfo, err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Failed to Process getClusterInfo() : %v\n\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	res, err := tencent.GetCluster(access_key, access_secret, region_id, cluster_id)
	if err != nil {
		err := fmt.Errorf("Failed to Get Cluster:  %v", err)
		cblogger.Error(err)
		return nil, err
	}

	if *res.Response.TotalCount == 0 {
		// NOTE: This message must contain a token recognized by common-runtime
		// checkNotFoundError() ("not found" / "not exist"). DeleteCluster() polls
		// GetCluster() until the cluster disappears from the CSP and ends the wait
		// on that check; a message it cannot recognize turns a successful deletion
		// into an error and leaves a stale ClusterIIDInfo in the meta DB.
		err := fmt.Errorf("Cluster not found: cluster_id: %s", cluster_id)
		cblogger.Error(err)
		return nil, err
	}

	// https://intl.cloud.tencent.com/document/api/457/32022#ClusterStatus
	// Cluster status (Running, Creating, Idling or Abnormal)
	health_status := *res.Response.Clusters[0].ClusterStatus
	cluster_status := irs.ClusterActive
	if strings.EqualFold(health_status, "Creating") {
		cluster_status = irs.ClusterCreating
	} else if strings.EqualFold(health_status, "Upgrading") {
		cluster_status = irs.ClusterUpdating
	} else if strings.EqualFold(health_status, "Deleting") {
		cluster_status = irs.ClusterDeleting
	} else if strings.EqualFold(health_status, "Running") {
		cluster_status = irs.ClusterActive
	} else {
		cluster_status = irs.ClusterInactive
	}
	// else if strings.EqualFold(health_status, "") { // tencent has no "delete" state
	// cluster_status = irs.ClusterDeleting
	//}

	created_at := *res.Response.Clusters[0].CreatedTime // "2022-09-09T13:10:06Z",
	datetime, err := time.Parse(time.RFC3339, created_at)
	if err != nil {
		err := fmt.Errorf("Failed to Parse Create Time :  %v", err)
		cblogger.Error(err)
		panic(err)
	}
	jsonRes, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		cblogger.Error(fmt.Sprintf("Failed to marshal res: %v", err))
	} else {
		cblogger.Info(fmt.Sprintf("res in JSON format: %s", string(jsonRes)))
	}
	// SecurityGroup/Subnet are round-tripped as internal cluster Tags (see getCreateClusterRequest()).
	security_group_id, subnet_id := extractNetworkIDsFromTags(res.Response.Clusters[0].TagSpecification)

	accessInfo, err := getClusterAccessInfo(access_key, access_secret, region_id, cluster_id, security_group_id)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	clusterInfo = &irs.ClusterInfo{
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
			SecurityGroupIIDs: []irs.IID{{NameId: "", SystemId: security_group_id}},
			SubnetIIDs:        []irs.IID{{NameId: "", SystemId: subnet_id}},
		},
		Status:      cluster_status,
		CreatedTime: datetime,
		AccessInfo:  accessInfo,
		// KeyValueList: []irs.KeyValue{}, // Input flatten data
	}

	// Check and add if tags exist
	if res.Response.Clusters[0].TagSpecification != nil {
		var tagList []irs.KeyValue
		for _, tagSpec := range res.Response.Clusters[0].TagSpecification {
			if tagSpec == nil || tagSpec.Tags == nil {
				continue
			}
			for _, tag := range tagSpec.Tags {
				if tag == nil {
					continue
				}
				key := ""
				if tag.Key != nil {
					key = *tag.Key
				}
				// Internal CB-Spider bookkeeping tags (SecurityGroup/Subnet, see
				// getCreateClusterRequest()) are not part of the user-supplied TagList.
				if key == tagKeySecurityGroupID || key == tagKeySubnetID {
					continue
				}
				value := ""
				if tag.Value != nil {
					value = *tag.Value
				}
				tagList = append(tagList, irs.KeyValue{
					Key:   key,
					Value: value,
				})
			}
		}
		clusterInfo.TagList = tagList
	}

	// 2025-03-13 Changed to use StructToKeyValueList
	clusterInfo.KeyValueList = irs.StructToKeyValueList(res.Response.Clusters[0])
	// Extract & add k, v

	// KeyValueList: []irs.KeyValue{}, // Input flatten data
	// temp, err := json.Marshal(*res.Response.Clusters[0])
	// if err != nil {
	// 	err := fmt.Errorf("Failed to Marshal Cluster Info :  %v", err)
	// 	cblogger.Error(err)
	// 	panic(err)
	// }
	// var json_obj map[string]interface{}
	// json.Unmarshal([]byte(temp), &json_obj)

	// flat, err := flatten.Flatten(json_obj, "", flatten.DotStyle)
	// if err != nil {
	// 	err := fmt.Errorf("Failed to Flatten Cluster Info :  %v", err)
	// 	cblogger.Error(err)
	// 	return nil, err
	// }
	// for k, v := range flat {
	// 	temp := fmt.Sprintf("%v", v)
	// 	clusterInfo.KeyValueList = append(clusterInfo.KeyValueList, irs.KeyValue{Key: k, Value: temp})
	// }

	clusterInfo.KeyValueList = irs.StructToKeyValueList(*res.Response.Clusters[0])

	// NodeGroups
	res2, err := tencent.ListNodeGroup(access_key, access_secret, region_id, cluster_id)
	if err != nil {
		err := fmt.Errorf("Failed to List Node Group :  %v", err)
		cblogger.Error(err)
		return nil, err
	}

	for _, nodepool := range res2.Response.NodePoolSet {
		node_group_info, err := getNodeGroupInfo(access_key, access_secret, region_id, cluster_id, *nodepool.NodePoolId)
		if err != nil {
			err := fmt.Errorf("Failed to Get Node Group Info :  %v", err)
			cblogger.Error(err)
			return nil, err
		}
		clusterInfo.NodeGroupList = append(clusterInfo.NodeGroupList, *node_group_info)
	}

	return clusterInfo, err
}

// isClusterNotReadyError reports whether err is one of the expected, non-fatal Tencent
// error conditions seen while a cluster's control plane isn't reachable yet — most
// commonly because it has no worker nodes yet (CB-Spider's Tencent driver always
// creates the cluster before any NodeGroup, see validateAtCreateCluster()).
// KUBE_CLIENT_CONNECTION_ERROR ("host must be a URL or a host:port pair: \"http://\"")
// is Tencent's error when it tries to reach the cluster's kube-apiserver to finish
// provisioning the external endpoint but there is no node/kubelet to route to yet.
func isClusterNotReadyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "CLUSTER_IN_ABNORMAL_STAT") ||
		strings.Contains(msg, "CLUSTER_STATE_ERROR") ||
		strings.Contains(msg, "KubeClientConnection") ||
		strings.Contains(msg, "KUBE_CLIENT_CONNECTION_ERROR")
}

func getClusterAccessInfo(access_key string, access_secret string, region_id string, cluster_id string, security_group_id string) (accessInfo irs.AccessInfo, err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Failed to Process getClusterAccessInfo() : %v\n\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	accessInfo = irs.AccessInfo{
		Endpoint:   "Endpoint is not ready yet!",
		Kubeconfig: "Kubeconfig is not ready yet!",
	}

	// (0) Endpoint creation is a prerequisite for both the Endpoint and Kubeconfig below:
	// Tencent only starts serving ClusterExternalEndpoint/Kubeconfig once a public access
	// endpoint has actually finished being created. Drive that off the explicit endpoint
	// status (Created/Creating/NotFound/CreateFailed) rather than guessing readiness from
	// an empty ClusterExternalEndpoint string.
	statusRes, err := tencent.GetClusterEndpointStatus(access_key, access_secret, region_id, cluster_id)
	if err != nil {
		if isClusterNotReadyError(err) {
			cblogger.Info(cluster_id + err.Error())
			accessInfo.Endpoint = "Cluster is not ready yet!"
			return accessInfo, nil
		}
		// AccessInfo is a best-effort side-channel: any other (including opaque/transient
		// Tencent-side) failure here must not fail GetCluster()/ListCluster() as a whole —
		// surface it as an informational message instead of propagating a hard error.
		cblogger.Error(fmt.Errorf("Failed to Get Cluster Endpoint Status:  %v", err))
		accessInfo.Endpoint = fmt.Sprintf("Endpoint status unavailable: %v", err)
		return accessInfo, nil
	}
	if statusRes == nil || statusRes.Response == nil || statusRes.Response.Status == nil {
		return accessInfo, nil
	}

	switch *statusRes.Response.Status {
	case "NotFound":
		// No external (public) access endpoint has been requested for this cluster yet.
		// Request one now, by default, so the cluster becomes externally reachable
		// without requiring a separate explicit call.
		_, err := tencent.CreateClusterEndpoint(access_key, access_secret, region_id, cluster_id, security_group_id)
		if err != nil {
			if isClusterNotReadyError(err) {
				cblogger.Info(cluster_id + err.Error())
				accessInfo.Endpoint = "First, add a nodegroup."
			} else if strings.Contains(err.Error(), "same type task in execution") {
				cblogger.Error(cluster_id + err.Error())
				accessInfo.Endpoint = "Preparing...."
			} else {
				// Best-effort: don't fail GetCluster()/ListCluster() over a failed attempt
				// to (re)request the endpoint. It will be retried on the next call.
				cblogger.Error(fmt.Errorf("Failed to Create Cluster Endpoint:  %v", err))
				accessInfo.Endpoint = fmt.Sprintf("Endpoint creation error: %v", err)
			}
		} else {
			accessInfo.Endpoint = "Preparing...."
		}
		return accessInfo, nil

	case "Creating":
		accessInfo.Endpoint = "Preparing...."
		return accessInfo, nil

	case "CreateFailed":
		errMsg := "unknown reason"
		if statusRes.Response.ErrorMsg != nil && *statusRes.Response.ErrorMsg != "" {
			errMsg = *statusRes.Response.ErrorMsg
		}
		accessInfo.Endpoint = fmt.Sprintf("Endpoint creation failed: %s", errMsg)
		return accessInfo, nil

	case "Created":
		// fall through: only now is the endpoint address (and, after that, the Kubeconfig) obtainable.

	default:
		return accessInfo, nil
	}

	// (1) Endpoint address
	res, err := tencent.GetClusterEndpoint(access_key, access_secret, region_id, cluster_id)
	if err != nil {
		if isClusterNotReadyError(err) {
			cblogger.Info(cluster_id + err.Error())
			accessInfo.Endpoint = "Cluster is not ready yet!"
			return accessInfo, nil
		}
		cblogger.Error(fmt.Errorf("Failed to Get Cluster Endpoint:  %v", err))
		accessInfo.Endpoint = fmt.Sprintf("Endpoint unavailable: %v", err)
		return accessInfo, nil
	}
	if res == nil || res.Response == nil || res.Response.ClusterExternalEndpoint == nil || *res.Response.ClusterExternalEndpoint == "" {
		accessInfo.Endpoint = "Preparing...."
		return accessInfo, nil
	}
	accessInfo.Endpoint = *res.Response.ClusterExternalEndpoint

	// If the endpoint is not a valid value (e.g. still a temporary message),
	// do not expose the kubeconfig — it won't be externally accessible yet.
	if !strings.Contains(accessInfo.Endpoint, ":") {
		return accessInfo, nil
	}

	// (2) Kubeconfig — only obtainable once the endpoint address above is real,
	// since the API server address embedded in it must already be live.
	resKubeconfig, err := tencent.GetClusterKubeconfig(access_key, access_secret, region_id, cluster_id)
	if err != nil {
		if isClusterNotReadyError(err) {
			cblogger.Info(cluster_id + err.Error())
			accessInfo.Kubeconfig = "Cluster is not ready yet!"
			return accessInfo, nil
		}
		cblogger.Error(fmt.Errorf("Failed to Get Cluster Kubeconfig:  %v", err))
		accessInfo.Kubeconfig = fmt.Sprintf("Kubeconfig unavailable: %v", err)
		return accessInfo, nil
	}

	if resKubeconfig == "" {
		accessInfo.Kubeconfig = "Preparing...."
	} else {
		accessInfo.Kubeconfig = changeDomainNameToIP(resKubeconfig, accessInfo.Endpoint)
	}

	return accessInfo, nil
}

func changeDomainNameToIP(kubeConfig string, endpoint string) string {

	TargetStr := "    server: https://"

	if kubeConfig == "" || !strings.Contains(kubeConfig, TargetStr) {
		return kubeConfig
	}
	if endpoint == "" || !strings.Contains(endpoint, ":") {
		return kubeConfig
	}

	// get IP from 1.2.3.4:443
	splits := strings.Split(endpoint, ":")
	ip := splits[0]

	// replace 'domain name' with 'ip'
	// ex) server: https://cls-amu0j0tf.ccs.tencent-cloud.com
	//     => server: https://1.2.3.4
	lines := strings.Split(kubeConfig, "\n")
	for i, line := range lines {
		if strings.Contains(line, TargetStr) {
			lines[i] = TargetStr + ip
		}
	}

	return strings.Join(lines, "\n")
}

func getNodeGroupInfo(access_key, access_secret, region_id, cluster_id, node_group_id string) (nodeGroupInfo *irs.NodeGroupInfo, err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Failed to Process getNodeGroupInfo() : %v\n\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	res, err := tencent.GetNodeGroup(access_key, access_secret, region_id, cluster_id, node_group_id)
	if err != nil {
		err := fmt.Errorf("Failed to Get Node Group :  %v", err)
		cblogger.Error(err)
		return nil, err
	}

	launch_config, err := tencent.GetLaunchConfiguration(access_key, access_secret, region_id, *res.Response.NodePool.LaunchConfigurationId)
	if err != nil {
		err := fmt.Errorf("Failed to Get Launch Configuration :  %v", err)
		cblogger.Error(err)
		return nil, err
	}

	auto_scaling_group, err := tencent.GetAutoScalingGroup(access_key, access_secret, region_id, *res.Response.NodePool.AutoscalingGroupId)
	if err != nil {
		err := fmt.Errorf("Failed to Get Auto Scaling Group :  %v", err)
		cblogger.Error(err)
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
	if strings.EqualFold(*res.Response.NodePool.AutoscalingGroupStatus, "ENABLED") {
		auto_scale_enalbed = true
	}

	// Always initialize with basic info; details filled in when launch config and ASG are available.
	// If the node group is being deleted, its launch config and ASG may already be gone,
	// causing an empty set — which would leave nodeGroupInfo nil and panic on subsequent access.
	nodeGroupInfo = &irs.NodeGroupInfo{
		IId: irs.IID{
			NameId:   *res.Response.NodePool.Name,
			SystemId: *res.Response.NodePool.NodePoolId,
		},
		Status:        status,
		OnAutoScaling: auto_scale_enalbed,
		Nodes:         []irs.IID{},
		KeyValueList:  []irs.KeyValue{},
	}

	if len(launch_config.Response.LaunchConfigurationSet) > 0 && len(auto_scaling_group.Response.AutoScalingGroupSet) > 0 {
		nodeGroupInfo.ImageIID = irs.IID{
			NameId:   "",
			SystemId: *launch_config.Response.LaunchConfigurationSet[0].ImageId,
		}
		nodeGroupInfo.VMSpecName = *launch_config.Response.LaunchConfigurationSet[0].InstanceType
		nodeGroupInfo.RootDiskType = *launch_config.Response.LaunchConfigurationSet[0].SystemDisk.DiskType
		nodeGroupInfo.RootDiskSize = fmt.Sprintf("%d", *launch_config.Response.LaunchConfigurationSet[0].SystemDisk.DiskSize)
		nodeGroupInfo.KeyPairIID = irs.IID{NameId: "", SystemId: *launch_config.Response.LaunchConfigurationSet[0].LoginSettings.KeyIds[0]}
		nodeGroupInfo.MinNodeSize = int(*auto_scaling_group.Response.AutoScalingGroupSet[0].MinSize)
		nodeGroupInfo.MaxNodeSize = int(*auto_scaling_group.Response.AutoScalingGroupSet[0].MaxSize)
		nodeGroupInfo.DesiredNodeSize = int(*auto_scaling_group.Response.AutoScalingGroupSet[0].DesiredCapacity)
	}

	nodes, err := tencent.DescribeClusterInstances(access_key, access_secret, region_id, cluster_id)
	if err != nil {
		err := fmt.Errorf("Failed to Get Nodes :  %v", err)
		cblogger.Error(err)
		return nil, err
	}
	for _, node := range nodes.Response.InstanceSet {
		if node_group_id == *node.NodePoolId {
			if *node.InstanceId != "" {
				nodeGroupInfo.Nodes = append(nodeGroupInfo.Nodes, irs.IID{NameId: "", SystemId: *node.InstanceId})
			}
		}
	}

	// 2025-03-13 Changed to use StructToKeyValueList
	nodeGroupInfo.KeyValueList = irs.StructToKeyValueList(res.Response.NodePool)
	// add key value list

	// temp, err := json.Marshal(*res.Response.NodePool)
	// if err != nil {
	// 	err := fmt.Errorf("Failed to Marshal NodeGroup Info :  %v", err)
	// 	cblogger.Error(err)
	// 	panic(err)
	// }
	// var json_obj map[string]interface{}
	// json.Unmarshal([]byte(temp), &json_obj)
	// flat, err := flatten.Flatten(json_obj, "", flatten.DotStyle)
	// if err != nil {
	// 	err := fmt.Errorf("Failed to Flatten NodeGroup Info :  %v", err)
	// 	cblogger.Error(err)
	// 	return nil, err
	// }
	// for k, v := range flat {
	// 	temp := fmt.Sprintf("%v", v)
	// 	nodeGroupInfo.KeyValueList = append(nodeGroupInfo.KeyValueList, irs.KeyValue{Key: k, Value: temp})
	// }

	nodeGroupInfo.KeyValueList = irs.StructToKeyValueList(*res.Response.NodePool)

	return nodeGroupInfo, err
}

func getCreateClusterRequest(clusterHandler *TencentClusterHandler, clusterInfo irs.ClusterInfo) (request *tke.CreateClusterRequest, err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Failed to Process getCreateClusterRequest() : %v\n\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	// Read existing clusters' CIDRs directly from Tencent's typed response rather than
	// through clusterHandler.ListCluster()'s irs.ClusterInfo.KeyValueList: that list is
	// built by irs.StructToKeyValueList(), which only flattens one level, so a nested
	// field like ClusterNetworkSettings.ClusterCIDR never actually appears under a
	// "ClusterNetworkSettings.ClusterCIDR" key (nested structs are JSON-dumped under
	// their own top-level field name instead) — matching against that key would always
	// silently fail to find a used CIDR, and picking would always land on the first
	// pool candidate regardless of what's already in use.
	existingClusters, err := tencent.GetClusters(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region)
	if err != nil {
		err := fmt.Errorf("Failed to List Cluster :  %v", err)
		cblogger.Error(err)
		return nil, err
	}

	usedClusterCIDRs := make(map[string]bool)
	usedServiceCIDRs := make(map[string]bool)
	if existingClusters.Response != nil {
		for _, cluster := range existingClusters.Response.Clusters {
			if cluster == nil || cluster.ClusterNetworkSettings == nil {
				continue
			}
			if cluster.ClusterNetworkSettings.ClusterCIDR != nil {
				usedClusterCIDRs[*cluster.ClusterNetworkSettings.ClusterCIDR] = true
			}
			if cluster.ClusterNetworkSettings.ServiceCIDR != nil {
				usedServiceCIDRs[*cluster.ClusterNetworkSettings.ServiceCIDR] = true
			}
		}
	}

	// 172.X.0.0/16: X Range:16, 17, ... , 31
	clusterCIDRCandidates := make([]string, 0, 16)
	for i := 16; i <= 31; i++ {
		clusterCIDRCandidates = append(clusterCIDRCandidates, fmt.Sprintf("172.%d.0.0/16", i))
	}
	clusterCIDR, err := pickUnusedCIDR(usedClusterCIDRs, clusterCIDRCandidates)
	if err != nil {
		err := fmt.Errorf("Failed to Find an Unused ClusterCIDR :  %v", err)
		cblogger.Error(err)
		return nil, err
	}

	// 10.200.X.0/20: X Range:0, 16, 32, ... , 240 (16 blocks).
	// Tencent requires the ServiceCIDR mask to be between /17 and /27, so /16 (used for
	// ClusterCIDR above) is not valid here; /20 is used instead. A distinct first-two-octet
	// pool (10.200.x vs 172.x) is used so ServiceCIDR can never collide with a ClusterCIDR.
	serviceCIDRCandidates := make([]string, 0, 16)
	for i := 0; i <= 240; i += 16 {
		serviceCIDRCandidates = append(serviceCIDRCandidates, fmt.Sprintf("10.200.%d.0/20", i))
	}
	serviceCIDR, err := pickUnusedCIDR(usedServiceCIDRs, serviceCIDRCandidates)
	if err != nil {
		err := fmt.Errorf("Failed to Find an Unused ServiceCIDR :  %v", err)
		cblogger.Error(err)
		return nil, err
	}

	request = tke.NewCreateClusterRequest()
	request.ClusterCIDRSettings = &tke.ClusterCIDRSettings{
		ClusterCIDR: common.StringPtr(clusterCIDR),
		ServiceCIDR: common.StringPtr(serviceCIDR),
	}

	// Tencent TKE's CreateCluster API has no field to attach a Security Group to the
	// cluster itself, and ClusterBasicSettings/ClusterNetworkSettings.SubnetId is only
	// honored for CiliumOverlay / CDC+VPC-CNI clusters (not the GR-mode clusters created
	// here), so it isn't a reliable way to recall an arbitrary Subnet either.
	// Both values are therefore round-tripped as structured cluster Tags (rather than
	// packed into ClusterDescription) and read back in getClusterInfo(). This avoids the
	// old free-text/regex approach entirely, including the historical bug where TKE
	// propagates ClusterDescription onto CVM tags during Auto Scaling scale-out and
	// certain characters (e.g. '#') broke RunInstances.
	var tags []*tke.Tag
	for _, inputTag := range clusterInfo.TagList {
		tags = append(tags, &tke.Tag{
			Key:   common.StringPtr(inputTag.Key),
			Value: common.StringPtr(inputTag.Value),
		})
	}
	tags = append(tags,
		&tke.Tag{
			Key:   common.StringPtr(tagKeySecurityGroupID),
			Value: common.StringPtr(clusterInfo.Network.SecurityGroupIIDs[0].SystemId),
		},
		&tke.Tag{
			Key:   common.StringPtr(tagKeySubnetID),
			Value: common.StringPtr(clusterInfo.Network.SubnetIIDs[0].SystemId),
		},
	)

	request.ClusterBasicSettings = &tke.ClusterBasicSettings{
		ClusterName:    common.StringPtr(clusterInfo.IId.NameId),
		VpcId:          common.StringPtr(clusterInfo.Network.VpcIID.SystemId),
		ClusterVersion: common.StringPtr(clusterInfo.Version), // option, version: 1.22.5
		TagSpecification: []*tke.TagSpecification{{
			ResourceType: common.StringPtr("cluster"),
			Tags:         tags,
		}},
	}
	request.ClusterType = common.StringPtr("MANAGED_CLUSTER") //default value
	request.ClusterAdvancedSettings = &tke.ClusterAdvancedSettings{
		ContainerRuntime: common.StringPtr(defaultContainerRuntime),
	}

	return request, err
}

// pickUnusedCIDR returns the first entry of candidates that is not already recorded
// under keyValueKey in any of the given clusters' KeyValueList.
func pickUnusedCIDR(used map[string]bool, candidates []string) (string, error) {
	for _, cidr := range candidates {
		if !used[cidr] {
			return cidr, nil
		}
	}
	return "", fmt.Errorf("no unused CIDR block available among candidates %v", candidates)
}

func getNodeGroupRequest(clusterHandler *TencentClusterHandler, cluster_id string, nodeGroupReqInfo irs.NodeGroupInfo) (request *tke.CreateClusterNodePoolRequest, err error) {

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("Failed to Process getNodeGroupRequest() : %v\n\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cluster, res := clusterHandler.GetCluster(irs.IID{SystemId: cluster_id})
	if res != nil {
		err := fmt.Errorf("Failed to Get Cluster :  %v", err)
		cblogger.Error(err)
		return nil, res
	}
	vpc_id := cluster.Network.VpcIID.SystemId
	subnet_id := cluster.Network.SubnetIIDs[0].SystemId
	security_group_id := cluster.Network.SecurityGroupIIDs[0].SystemId
	disk_size, _ := strconv.ParseInt(nodeGroupReqInfo.RootDiskSize, 10, 64)

	strSystemDisk := ""
	switch {
	case nodeGroupReqInfo.RootDiskType == "" && disk_size == 0:
		strSystemDisk = ""
	case nodeGroupReqInfo.RootDiskType == "" && disk_size != 0:
		strSystemDisk = `"SystemDisk": { "DiskSize": %d },`
		strSystemDisk = fmt.Sprintf(strSystemDisk, disk_size)
	case nodeGroupReqInfo.RootDiskType != "" && disk_size == 0:
		strSystemDisk = `"SystemDisk": { "DiskType" : "%s" },`
		strSystemDisk = fmt.Sprintf(strSystemDisk, nodeGroupReqInfo.RootDiskType)
	default:
		strSystemDisk = `"SystemDisk": { "DiskType" : "%s", "DiskSize": %d },`
		strSystemDisk = fmt.Sprintf(strSystemDisk, nodeGroupReqInfo.RootDiskType, disk_size)
	}

	// '{"LaunchConfigurationName":"name","InstanceType":"S3.MEDIUM2","ImageId":"img-pi0ii46r"}'
	// "SystemDisk": { "DiskType" : "CLOUD_BSSD", "DiskSize": 50 },
	// Error occurs if ImageId is set, not set.
	launch_config_json_str := `{
		"InstanceType": "%s",
		"SecurityGroupIds": ["%s"],
		"LoginSettings": { "KeyIds" : ["%s"] },
		"InstanceChargeType": "POSTPAID_BY_HOUR",
		%s
		"InternetAccessible": {
			"InternetChargeType":"TRAFFIC_POSTPAID_BY_HOUR",
			"InternetMaxBandwidthOut": 1,
			"PublicIpAssigned": true
		}
	}`
	launch_config_json_str = fmt.Sprintf(launch_config_json_str, nodeGroupReqInfo.VMSpecName, security_group_id, nodeGroupReqInfo.KeyPairIID.SystemId, strSystemDisk)

	auto_scaling_group_json_str := `{
		"MinSize": %d,
		"MaxSize": %d,			
		"DesiredCapacity": %d,
		"VpcId": "%s",
		"SubnetIds": ["%s"]
	}`

	auto_scaling_group_json_str = fmt.Sprintf(auto_scaling_group_json_str, nodeGroupReqInfo.MinNodeSize, nodeGroupReqInfo.MaxNodeSize, nodeGroupReqInfo.DesiredNodeSize, vpc_id, subnet_id)

	request = tke.NewCreateClusterNodePoolRequest()
	request.Name = common.StringPtr(nodeGroupReqInfo.IId.NameId)
	request.ClusterId = common.StringPtr(cluster_id)
	request.LaunchConfigurePara = common.StringPtr(launch_config_json_str)
	request.AutoScalingGroupPara = common.StringPtr(auto_scaling_group_json_str)
	request.EnableAutoscale = common.BoolPtr(nodeGroupReqInfo.OnAutoScaling)
	request.InstanceAdvancedSettings = &tke.InstanceAdvancedSettings{
		// DataDisks: []*tke.DataDisk{
		// 	{
		// 		DiskType: common.StringPtr(nodeGroupReqInfo.RootDiskType), //ex. "CLOUD_PREMIUM"
		// 		DiskSize: common.Int64Ptr(disk_size),                      //ex. 50
		// 	},
		// },
	}
	if nodeGroupReqInfo.ImageIID.SystemId != "" {
		// List of registrable image names: https://www.tencentcloud.com/document/product/457/46750
		request.NodePoolOs = common.StringPtr(nodeGroupReqInfo.ImageIID.SystemId) // ex: "tlinux3.1x86_64"
	}
	// request.ContainerRuntime = common.StringPtr("docker")
	// request.RuntimeVersion = common.StringPtr("19.3")
	// print(request.ToJsonString())

	return request, err
}

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
	// CAUTION: Currently CB-Spider's Tencent PMKS Drivers does not support to create a cluster with nodegroups
	if len(clusterInfo.NodeGroupList) > 0 {
		return fmt.Errorf("Node Group cannot be specified")
	}

	return nil
}

// validateClusterVersion checks the requested Kubernetes version against the versions
// Tencent TKE actually supports in the target region, and reports the supported list
// in the error message when the requested version is invalid.
func validateClusterVersion(clusterHandler *TencentClusterHandler, version string) error {
	if version == "" {
		return fmt.Errorf("Cluster Version is required")
	}

	res, err := tencent.GetClusterVersions(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region)
	if err != nil {
		return fmt.Errorf("Failed to Get Supported Cluster Versions :  %v", err)
	}
	if res == nil || res.Response == nil {
		return fmt.Errorf("Failed to Get Supported Cluster Versions : empty response")
	}

	requested := strings.TrimPrefix(version, "v")
	supportedVersions := make([]string, 0, len(res.Response.VersionInstanceSet))
	for _, item := range res.Response.VersionInstanceSet {
		if item == nil || item.Version == nil || *item.Version == "" {
			continue
		}
		supported := strings.TrimPrefix(*item.Version, "v")
		supportedVersions = append(supportedVersions, supported)
		if supported == requested {
			return nil
		}
	}

	return fmt.Errorf("Unsupported Cluster Version %q. Supported versions in region %q: %s", version, clusterHandler.RegionInfo.Region, strings.Join(supportedVersions, ", "))
}

func validateAtAddNodeGroup(clusterIID irs.IID, nodeGroupInfo irs.NodeGroupInfo) error {
	if clusterIID.SystemId == "" && clusterIID.NameId == "" {
		return fmt.Errorf("Invalid Cluster IID")
	}
	if nodeGroupInfo.IId.NameId == "" {
		return fmt.Errorf("Node Group name is required")
	}
	if nodeGroupInfo.MaxNodeSize < 1 {
		return fmt.Errorf("MaxNodeSize cannot be smaller than 1")
	}
	if nodeGroupInfo.MinNodeSize < 1 {
		return fmt.Errorf("MaxNodeSize cannot be smaller than 1")
	}
	if nodeGroupInfo.DesiredNodeSize < 1 {
		return fmt.Errorf("DesiredNodeSize cannot be smaller than 1")
	}
	if nodeGroupInfo.VMSpecName == "" {
		return fmt.Errorf("VM Spec Name is required")
	}

	return nil
}

func validateAtChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID, minNodeSize int, maxNodeSize int) error {
	if clusterIID.SystemId == "" && clusterIID.NameId == "" {
		return fmt.Errorf("Invalid Cluster IID")
	}
	if nodeGroupIID.SystemId == "" && nodeGroupIID.NameId == "" {
		return fmt.Errorf("Invalid Node Group IID")
	}
	if minNodeSize < 1 {
		return fmt.Errorf("MaxNodeSize cannot be smaller than 1")
	}
	if maxNodeSize < 1 {
		return fmt.Errorf("MaxNodeSize cannot be smaller than 1")
	}

	return nil
}

func (clusterHandler *TencentClusterHandler) ListIID() ([]*irs.IID, error) {
	var iidList []*irs.IID

	callLogInfo := GetCallLogScheme(clusterHandler.RegionInfo, call.CLUSTER, "ListIID", "GetClusters()")

	start := call.Start()
	res, err := tencent.GetClusters(clusterHandler.CredentialInfo.ClientId, clusterHandler.CredentialInfo.ClientSecret, clusterHandler.RegionInfo.Region)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Get Clusters :  %v", err)
		cblogger.Error(err)
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		return iidList, err
	}
	calllogger.Debug(call.String(callLogInfo))

	for _, cluster := range res.Response.Clusters {
		iid := irs.IID{SystemId: *cluster.ClusterId}
		iidList = append(iidList, &iid)
	}
	return iidList, nil
}
