// Tencent Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//   - Cloud-Barista: https://github.com/cloud-barista
//
// This is Tencent Driver.
//
// by CB-Spider Team, 2022.09.
package main

import (
	"fmt"
	"os"
	"testing"

	tencent "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/tencent/utils/tencent"
	tke "github.com/tencentcloud/tencentcloud-sdk-go-intl-en/tencentcloud/tke/v20180525"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

var secret_id = os.Getenv("CLIENT_ID")
var secret_key = os.Getenv("CLIENT_SECRET")
var region_id = os.Getenv("REGION")

func TestCreateCluster1(t *testing.T) {

	// Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
	request := tke.NewCreateClusterRequest()

	// request.FromJsonString()
	request.ClusterCIDRSettings = &tke.ClusterCIDRSettings{
		ClusterCIDR: common.StringPtr("172.21.0.0/16"), // 172.X.0.0.16: X Range:16, 17, ... , 31
		//IgnoreClusterCIDRConflict: common.BoolPtr(false),
		//MaxNodePodNum:             common.Uint64Ptr(64),
		//MaxClusterServiceNum:      common.Uint64Ptr(1024),
		//ServiceCIDR: common.StringPtr("172.17.252.0/22"),
	}
	request.ClusterBasicSettings = &tke.ClusterBasicSettings{
		ClusterName:    common.StringPtr("cluster-x2"),
		VpcId:          common.StringPtr("vpc-q1c6fr9e"),
		ClusterVersion: common.StringPtr("1.22.5"), //version: 1.22.5
	}
	request.ClusterType = common.StringPtr("MANAGED_CLUSTER")

	res, err := tencent.CreateCluster(secret_id, secret_key, region_id, request)
	if err != nil {
		t.Errorf("CreateCluster failed: %v", err)
		return
	}

	println(res.ToJsonString())
	println(res)
}

func TestGetClusters(t *testing.T) {
	res, err := tencent.GetClusters(secret_id, secret_key, region_id)
	if err != nil {
		t.Errorf("GetClusters failed: %v", err)
		return
	}

	println(res.ToJsonString())
	println(res)
}

func TestGetCluster(t *testing.T) {
	res, err := tencent.GetClusters(secret_id, secret_key, region_id)
	if err != nil {
		t.Errorf("GetClusters failed: %v", err)
		return
	}

	for _, cluster := range res.Response.Clusters {
		res, err := tencent.GetCluster(secret_id, secret_key, region_id, *cluster.ClusterId)
		if err != nil {
			t.Errorf("GetCluster failed: %v", err)
		}

		if *res.Response.TotalCount == 0 {
			t.Errorf("GetCluster failed: %v", err)
		} else {
			println(*res.Response.Clusters[0].ClusterId)
		}

		println(res.ToJsonString())
	}
}

func TestDeleteCluster(t *testing.T) {

	res, err := tencent.GetClusters(secret_id, secret_key, region_id)
	if err != nil {
		t.Errorf("GetClusters failed: %v", err)
		return
	}

	for _, cluster := range res.Response.Clusters {
		temp, err := tencent.DeleteCluster(secret_id, secret_key, region_id, *cluster.ClusterId)
		if err != nil {
			t.Errorf("DeleteCluster failed: %v", err)
		}
		println(temp.ToJsonString())
	}
}

/*
{"LaunchConfigurationName":"cls-ke0ztn01_nodepool-x","ImageId":"img-pi0ii46r","InstanceType":"S3.MEDIUM2","SystemDisk":{"DiskType":"CLOUD_PREMIUM","DiskSize":50}}

{"AutoScalingGroupName":"cls-ke0ztn01_nodepool-x","LaunchConfigurationId":"asc-idvw29yj","MaxSize":3,"MinSize":0,"VpcId":"vpc-q1c6fr9e","DesiredCapacity":0,"SubnetIds":["subnet-rl79gxhv"]}
*/

// // lc_json_str := `{
// // 	"LaunchConfigurationName": "cls-ke0ztn01_nodepool-x",
// // 	"ImageId": "img-pi0ii46r",
// // 	"InstanceType": "S3.MEDIUM2",
// // 	"SystemDisk": {
// // 		"DiskType": "CLOUD_PREMIUM",
// // 		"DiskSize": 50
// // 	}
// // }`
// /*
// 	https://intl.cloud.tencent.com/ko/document/product/457/33852?lang=ko&pg=
// 	The pass-through parameters for launch configuration creation, in the format of a JSON string.
// 	For more information, see the CreateLaunchConfiguration API.
// 	ImageId is not required as it is already included in the cluster dimension.
// 	UserData is not required as it's set through the UserScript.
// */

// image_ids
// img-pi0ii46r
// Ubuntu Server 18.04.1 LTS 64bit
// https://console.intl.cloud.tencent.com/cvm/image/index?rid=1&imageType=PUBLIC_IMAGE
// https://console.intl.cloud.tencent.com/cvm/image?rid=1&imageType=PUBLIC_IMAGE

// /*
// System disk type. For more information on limits of system disk types, see Cloud Disk Types. Valid values:
// LOCAL_BASIC: local disk
// LOCAL_SSD: local SSD disk
// CLOUD_BASIC: HDD cloud disk
// CLOUD_PREMIUM: premium cloud storage
// CLOUD_SSD: SSD cloud disk

// Default value: CLOUD_PREMIUM.

func TestCreateNodeGroup(t *testing.T) {

	// cluster
	res, err := tencent.GetClusters(secret_id, secret_key, region_id)
	if err != nil {
		t.Errorf("GetClusters failed: %v", err)
		return
	}

	for _, cluster := range res.Response.Clusters {
		cluster_id := *cluster.ClusterId
		println(*cluster.ClusterId)

		// https://intl.cloud.tencent.com/ko/document/product/377/31001?has_map=1
		// launch_config_json_str := `{
		// 	"InstanceType": "%s",
		// 	"ImageId": "img-pi0ii46r",
		// 	"SecurityGroupIds": ["%s"]
		// }`
		launch_config_json_str := `{
			"InstanceType": "%s",
			"SecurityGroupIds": ["%s"]
		}`
		launch_config_json_str = fmt.Sprintf(launch_config_json_str, "S3.MEDIUM2", "sg-46eef229")

		auto_scaling_group_json_str := `{
			"MinSize": %d,
			"MaxSize": %d,			
			"DesiredCapacity": %d,
			"VpcId": "%s",
			"SubnetIds": ["%s"]
		}`
		auto_scaling_group_json_str = fmt.Sprintf(auto_scaling_group_json_str, 0, 3, 1, "vpc-q1c6fr9e", "subnet-rl79gxhv")

		enable_auto_scale := true

		// cluster_id, "cls-ke0ztn01_nodepool-x", lc_json_str, asc_json_str, true, "CLOUD_PREMIUM", 50
		// Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
		request := tke.NewCreateClusterNodePoolRequest()
		request.Name = common.StringPtr("nodegroup_x1")
		request.ClusterId = common.StringPtr(cluster_id)
		request.LaunchConfigurePara = common.StringPtr(launch_config_json_str)
		request.AutoScalingGroupPara = common.StringPtr(auto_scaling_group_json_str)
		request.EnableAutoscale = common.BoolPtr(enable_auto_scale)
		request.InstanceAdvancedSettings = &tke.InstanceAdvancedSettings{
			DataDisks: []*tke.DataDisk{
				{
					DiskType: common.StringPtr("CLOUD_PREMIUM"),
					DiskSize: common.Int64Ptr(50),
				},
			},
		}

		res, err := tencent.CreateNodeGroup(secret_id, secret_key, region_id, request)
		if err != nil {
			println(err)
		}
		println(res.ToJsonString())
	}
}

func TestListNodeGroup(t *testing.T) {
	res, err := tencent.GetClusters(secret_id, secret_key, region_id)
	if err != nil {
		println(err)
	}

	for _, cluster := range res.Response.Clusters {
		println(cluster.ClusterId)
		nodegroups, err := tencent.ListNodeGroup(secret_id, secret_key, region_id, *cluster.ClusterId)
		if err != nil {
			println(err)
		}
		for _, nodepool := range nodegroups.Response.NodePoolSet {
			println(*nodepool.Name, *nodepool.NodePoolId)
		}
	}
}

func TestGetNodeGroup(t *testing.T) {

	res, err := tencent.GetClusters(secret_id, secret_key, region_id)
	if err != nil {
		println(err)
	}

	for _, cluster := range res.Response.Clusters {
		println(cluster.ClusterId)
		nodegroups, err := tencent.ListNodeGroup(secret_id, secret_key, region_id, *cluster.ClusterId)
		if err != nil {
			println(err)
		}
		for _, nodepool := range nodegroups.Response.NodePoolSet {
			println(*nodepool.Name, *nodepool.NodePoolId)
			nodegroup, err := tencent.GetNodeGroup(secret_id, secret_key, region_id, *cluster.ClusterId, *nodepool.NodePoolId)
			if err != nil {
				println(err)
			}
			println(nodegroup.ToJsonString())
		}
	}
}

func TestSetNodeGroupAutoScaling(t *testing.T) {

	res, err := tencent.GetClusters(secret_id, secret_key, region_id)
	if err != nil {
		println(err)
	}

	for _, cluster := range res.Response.Clusters {
		println(cluster.ClusterId)
		nodegroups, err := tencent.ListNodeGroup(secret_id, secret_key, region_id, *cluster.ClusterId)
		if err != nil {
			println(err)
			continue
		}
		for _, nodepool := range nodegroups.Response.NodePoolSet {
			println(*nodepool.Name, *nodepool.NodePoolId)
			// nodepool.AutoscalingGroupId
			temp, err := tencent.SetNodeGroupAutoScaling(secret_id, secret_key, region_id, *cluster.ClusterId, *nodepool.NodePoolId, false)
			if err != nil {
				println(err)
			}
			println(temp.ToJsonString())

			temp, err = tencent.SetNodeGroupAutoScaling(secret_id, secret_key, region_id, *cluster.ClusterId, *nodepool.NodePoolId, true)
			if err != nil {
				println(err)
			}
			println(temp.ToJsonString())
		}
	}
}

func TestChangeNodeGroupAutoScaling(t *testing.T) {

	res, err := tencent.GetClusters(secret_id, secret_key, region_id)
	if err != nil {
		println(err)
	}

	for _, cluster := range res.Response.Clusters {
		println(cluster.ClusterId)
		nodegroups, err := tencent.ListNodeGroup(secret_id, secret_key, region_id, *cluster.ClusterId)
		if err != nil {
			println(err)
			continue
		}
		for _, nodepool := range nodegroups.Response.NodePoolSet {
			println(*nodepool.Name, *nodepool.NodePoolId)
			// nodepool.AutoscalingGroupId
			temp, err := tencent.ChangeNodeGroupScaling(secret_id, secret_key, region_id, *nodepool.AutoscalingGroupId, 2, 1, 5)
			if err != nil {
				println(err)
			}
			println(temp.ToJsonString())

			temp, err = tencent.ChangeNodeGroupScaling(secret_id, secret_key, region_id, *nodepool.AutoscalingGroupId, 1, 0, 3)
			if err != nil {
				println(err)
			}
			println(temp.ToJsonString())
		}
	}
}

func TestDeleteNodeGroup(t *testing.T) {

	res, err := tencent.GetClusters(secret_id, secret_key, region_id)
	if err != nil {
		println(err)
	}

	for _, cluster := range res.Response.Clusters {
		println(cluster.ClusterId)
		nodegroups, err := tencent.ListNodeGroup(secret_id, secret_key, region_id, *cluster.ClusterId)
		if err != nil {
			println(err)
		}
		for _, nodepool := range nodegroups.Response.NodePoolSet {
			println(*nodepool.Name, *nodepool.NodePoolId)
			temp, err := tencent.DeleteNodeGroup(secret_id, secret_key, region_id, *cluster.ClusterId, *nodepool.NodePoolId)
			if err != nil {
				println(err)
			}
			println(temp.ToJsonString())

		}
	}
}

func TestUpgradeCluster(t *testing.T) {

	res, err := tencent.GetClusters(secret_id, secret_key, region_id)
	if err != nil {
		println(err)
	}

	for _, cluster := range res.Response.Clusters {
		cluster_id := *cluster.ClusterId
		version := "1.22.5"
		res, err := tencent.UpgradeCluster(secret_id, secret_key, region_id, cluster_id, version)
		if err != nil {
			println(err.Error())
			//[TencentCloudSDKError] Code=InvalidParameter.Param,
			//Message=PARAM_ERROR(unsupported convert 1.20.6 to 1.22.5),
			//RequestId=859d4b16-91c8-40e6-97dd-c7b8006ba7aa
		}
		println(res.ToJsonString())

		version = "1.20.6"
		res, err = tencent.UpgradeCluster(secret_id, secret_key, region_id, cluster_id, version)
		if err != nil {
			println(err.Error())
		}
		println(res.ToJsonString())

		version = "1.20.7"
		res, err = tencent.UpgradeCluster(secret_id, secret_key, region_id, cluster_id, version)
		if err != nil {
			println(err.Error())
			//[TencentCloudSDKError] Code=ResourceUnavailable.ClusterState,
			//Message=CLUSTER_STATE_ERROR(cluster is in upgrading),
			//RequestId=304b274a-3500-4d33-9e79-60d849dd192d
		}
		println(res.ToJsonString())

	}
}

func TestDescribeSecurityGroups(t *testing.T) {

	res, err := tencent.DescribeSecurityGroups(secret_id, secret_key, region_id)
	if err != nil {
		println(err)
	}

	for _, group := range res.Response.SecurityGroupSet {
		if *group.IsDefault {
			println(*group.SecurityGroupId, *group.SecurityGroupName)
		}
	}

}
