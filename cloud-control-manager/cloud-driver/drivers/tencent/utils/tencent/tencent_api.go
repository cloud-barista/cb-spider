// Tencent Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Tencent Driver.
//
// by CB-Spider Team, 2022.09.

package tencent

import (
	as "github.com/tencentcloud/tencentcloud-sdk-go-intl-en/tencentcloud/as/v20180419"
	"github.com/tencentcloud/tencentcloud-sdk-go-intl-en/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go-intl-en/tencentcloud/common/profile"
	tke "github.com/tencentcloud/tencentcloud-sdk-go-intl-en/tencentcloud/tke/v20180525"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go-intl-en/tencentcloud/vpc/v20170312"
)

func CreateCluster(secret_id string, secret_key string, region_id string, request *tke.CreateClusterRequest) (*tke.CreateClusterResponse, error) {

	// Required steps:
	// Instantiate an authentication object. The Tencent Cloud account key pair `secretId` and `secretKey` need to be passed in as the input parameters
	// This example uses the way to read from the environment variable, so you need to set these two values in the environment variable in advance
	// You can also write the key pair directly into the code, but be careful not to copy, upload, or share the code to others
	// Query the CAM key: https://console.cloud.tencent.com/cam/capi
	credential := common.NewCredential(secret_id, secret_key)
	// Optional steps:

	// Instantiate a client configuration object. You can specify the timeout period and other configuration items
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	// Instantiate an client object
	// The second parameter is the region information. You can directly enter the string "ap-guangzhou" or import the preset constant
	client, _ := tke.NewClient(credential, region_id, cpf)

	// // Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
	// request := tke.NewCreateClusterRequest()

	// // request.FromJsonString()

	// request.ClusterCIDRSettings = &tke.ClusterCIDRSettings{
	// 	ClusterCIDR:               common.StringPtr("172.17.0.0/16"), // CIDR: 10.0.0.0/16
	// 	IgnoreClusterCIDRConflict: common.BoolPtr(false),
	// 	MaxNodePodNum:             common.Uint64Ptr(64),
	// 	MaxClusterServiceNum:      common.Uint64Ptr(1024),
	// 	ServiceCIDR:               common.StringPtr("172.17.252.0/22"), // 172.X.0.0.16: X Range:16, 17, ... , 31
	// }
	// request.ClusterBasicSettings = &tke.ClusterBasicSettings{
	// 	ClusterName: common.StringPtr(cluster_name),
	// 	VpcId:       common.StringPtr("vpc-q1c6fr9e"),
	// }
	// request.ClusterType = common.StringPtr("MANAGED_CLUSTER")

	// The returned "resp" is an instance of the CreateClusterResponse class which corresponds to the request object
	response, err := client.CreateCluster(request)
	if err != nil {
		return nil, err
	}
	// A string return packet in JSON format is output
	return response, nil
}

func GetClusters(secret_id string, secret_key string, region_id string) (*tke.DescribeClustersResponse, error) {
	// Required steps:
	// Instantiate an authentication object. The Tencent Cloud account key pair `secretId` and `secretKey` need to be passed in as the input parameters
	// This example uses the way to read from the environment variable, so you need to set these two values in the environment variable in advance
	// You can also write the key pair directly into the code, but be careful not to copy, upload, or share the code to others
	// Query the CAM key: https://console.cloud.tencent.com/cam/capi
	credential := common.NewCredential(secret_id, secret_key)
	// Optional steps:

	// Instantiate a client configuration object. You can specify the timeout period and other configuration items
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	// Instantiate an client object
	// The second parameter is the region information. You can directly enter the string "ap-guangzhou" or import the preset constant
	client, _ := tke.NewClient(credential, region_id, cpf)

	// Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
	request := tke.NewDescribeClustersRequest()

	// The returned "resp" is an instance of the DescribeClustersResponse class which corresponds to the request object
	response, err := client.DescribeClusters(request)
	if err != nil {
		return nil, err
	}

	// A string return packet in JSON format is output
	return response, nil
}

func GetCluster(secret_id string, secret_key string, region_id string, cluster_id string) (*tke.DescribeClustersResponse, error) {

	// Required steps:
	// Instantiate an authentication object. The Tencent Cloud account key pair `secretId` and `secretKey` need to be passed in as the input parameters
	// This example uses the way to read from the environment variable, so you need to set these two values in the environment variable in advance
	// You can also write the key pair directly into the code, but be careful not to copy, upload, or share the code to others
	// Query the CAM key: https://console.cloud.tencent.com/cam/capi
	credential := common.NewCredential(secret_id, secret_key)
	// Optional steps:

	// Instantiate a client configuration object. You can specify the timeout period and other configuration items
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	// Instantiate an client object
	// The second parameter is the region information. You can directly enter the string "ap-guangzhou" or import the preset constant
	client, _ := tke.NewClient(credential, region_id, cpf)

	// Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
	request := tke.NewDescribeClustersRequest()

	request.ClusterIds = common.StringPtrs([]string{cluster_id})

	// The returned "resp" is an instance of the DescribeClustersResponse class which corresponds to the request object
	response, err := client.DescribeClusters(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func DeleteCluster(secret_id string, secret_key string, region_id string, cluster_id string) (*tke.DeleteClusterResponse, error) {

	// Required steps:
	// Instantiate an authentication object. The Tencent Cloud account key pair `secretId` and `secretKey` need to be passed in as the input parameters
	// This example uses the way to read from the environment variable, so you need to set these two values in the environment variable in advance
	// You can also write the key pair directly into the code, but be careful not to copy, upload, or share the code to others
	// Query the CAM key: https://console.cloud.tencent.com/cam/capi
	credential := common.NewCredential(secret_id, secret_key)
	// Optional steps:

	// Instantiate a client configuration object. You can specify the timeout period and other configuration items
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	// Instantiate an client object
	// The second parameter is the region information. You can directly enter the string "ap-guangzhou" or import the preset constant
	client, _ := tke.NewClient(credential, region_id, cpf)

	// Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
	request := tke.NewDeleteClusterRequest()

	request.ClusterId = common.StringPtr(cluster_id)
	request.InstanceDeleteMode = common.StringPtr("terminate") // or "retain"

	// The returned "resp" is an instance of the DescribeClustersResponse class which corresponds to the request object
	response, err := client.DeleteCluster(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// 오토스케일링 파라이터 => json
// 론치컨피그 => json
// https://intl.cloud.tencent.com/ko/document/product/457/33852?lang=ko&pg=

func CreateNodeGroup(secret_id string, secret_key string, region_id string, request *tke.CreateClusterNodePoolRequest) (*tke.CreateClusterNodePoolResponse, error) {
	// Required steps:
	// Instantiate an authentication object. The Tencent Cloud account key pair `secretId` and `secretKey` need to be passed in as the input parameters
	// This example uses the way to read from the environment variable, so you need to set these two values in the environment variable in advance
	// You can also write the key pair directly into the code, but be careful not to copy, upload, or share the code to others
	// Query the CAM key: https://console.cloud.tencent.com/cam/capi
	credential := common.NewCredential(secret_id, secret_key)
	// Optional steps:

	// Instantiate a client configuration object. You can specify the timeout period and other configuration items
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	// Instantiate an client object
	// The second parameter is the region information. You can directly enter the string "ap-guangzhou" or import the preset constant
	client, _ := tke.NewClient(credential, region_id, cpf)

	// // Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
	// request := tke.NewCreateClusterNodePoolRequest()
	// request.Name = common.StringPtr(nodegroup_name)
	// request.ClusterId = common.StringPtr(cluster_id)
	// request.LaunchConfigurePara = common.StringPtr(launch_config_json_str)
	// request.AutoScalingGroupPara = common.StringPtr(auto_scaling_group_json_str)
	// request.EnableAutoscale = common.BoolPtr(enable_auto_scale)
	// request.InstanceAdvancedSettings = &tke.InstanceAdvancedSettings{
	// 	DataDisks: []*tke.DataDisk{
	// 		{
	// 			DiskType: common.StringPtr(disk_type),
	// 			DiskSize: common.Int64Ptr(disk_size),
	// 		},
	// 	},
	// }

	// The returned "resp" is an instance of the CreateClusterNodePoolResponse class which corresponds to the request object
	response, err := client.CreateClusterNodePool(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func ListNodeGroup(secret_id string, secret_key string, region_id string, cluster_id string) (*tke.DescribeClusterNodePoolsResponse, error) {

	// Required steps:
	// Instantiate an authentication object. The Tencent Cloud account key pair `secretId` and `secretKey` need to be passed in as the input parameters
	// This example uses the way to read from the environment variable, so you need to set these two values in the environment variable in advance
	// You can also write the key pair directly into the code, but be careful not to copy, upload, or share the code to others
	// Query the CAM key: https://console.cloud.tencent.com/cam/capi
	credential := common.NewCredential(secret_id, secret_key)
	// Optional steps:

	// Instantiate a client configuration object. You can specify the timeout period and other configuration items
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	// Instantiate an client object
	// The second parameter is the region information. You can directly enter the string "ap-guangzhou" or import the preset constant
	client, _ := tke.NewClient(credential, region_id, cpf)

	// Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
	request := tke.NewDescribeClusterNodePoolsRequest()

	request.ClusterId = common.StringPtr(cluster_id)

	// The returned "resp" is an instance of the DescribeClusterNodePoolsResponse class which corresponds to the request object
	response, err := client.DescribeClusterNodePools(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func GetNodeGroup(secret_id string, secret_key string, region_id string, cluster_id string, nodegroup_id string) (*tke.DescribeClusterNodePoolDetailResponse, error) {
	// Required steps:
	// Instantiate an authentication object. The Tencent Cloud account key pair `secretId` and `secretKey` need to be passed in as the input parameters
	// This example uses the way to read from the environment variable, so you need to set these two values in the environment variable in advance
	// You can also write the key pair directly into the code, but be careful not to copy, upload, or share the code to others
	// Query the CAM key: https://console.cloud.tencent.com/cam/capi
	credential := common.NewCredential(secret_id, secret_key)
	// Optional steps:

	// Instantiate a client configuration object. You can specify the timeout period and other configuration items
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	// Instantiate an client object
	// The second parameter is the region information. You can directly enter the string "ap-guangzhou" or import the preset constant
	client, _ := tke.NewClient(credential, region_id, cpf)

	// Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
	request := tke.NewDescribeClusterNodePoolDetailRequest()

	request.ClusterId = common.StringPtr(cluster_id)
	request.NodePoolId = common.StringPtr(nodegroup_id)

	// The returned "resp" is an instance of the DescribeClusterNodePoolDetailResponse class which corresponds to the request object
	response, err := client.DescribeClusterNodePoolDetail(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func DeleteNodeGroup(secret_id string, secret_key string, region_id string, cluster_id string, nodepool_id string) (*tke.DeleteClusterNodePoolResponse, error) {
	credential := common.NewCredential(secret_id, secret_key)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	client, _ := tke.NewClient(credential, region_id, cpf)

	// Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
	request := tke.NewDeleteClusterNodePoolRequest()

	request.ClusterId = common.StringPtr(cluster_id)
	request.NodePoolIds = common.StringPtrs([]string{nodepool_id})
	request.KeepInstance = common.BoolPtr(false)

	// The returned "resp" is an instance of the DeleteClusterNodePoolResponse class which corresponds to the request object
	response, err := client.DeleteClusterNodePool(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func SetNodeGroupAutoScaling(secret_id string, secret_key string, region_id string, cluster_id string, nodepool_id string, enable bool) (*tke.ModifyClusterNodePoolResponse, error) {

	credential := common.NewCredential(secret_id, secret_key)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	client, _ := tke.NewClient(credential, region_id, cpf)

	// Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
	request := tke.NewModifyClusterNodePoolRequest()

	request.EnableAutoscale = common.BoolPtr(false)
	request.ClusterId = common.StringPtr(cluster_id)
	request.NodePoolId = common.StringPtr(nodepool_id)
	request.EnableAutoscale = common.BoolPtr(enable)

	// The returned "resp" is an instance of the ModifyClusterNodePoolResponse class which corresponds to the request object
	response, err := client.ModifyClusterNodePool(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func ChangeNodeGroupScaling(secret_id string, secret_key string, region_id string, autoscaling_id string, desired_count uint64, min_count uint64, max_count uint64) (*as.ModifyAutoScalingGroupResponse, error) {

	credential := common.NewCredential(secret_id, secret_key)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "as.tencentcloudapi.com"
	client, _ := as.NewClient(credential, "ap-beijing", cpf)

	request := as.NewModifyAutoScalingGroupRequest()
	request.AutoScalingGroupId = common.StringPtr(autoscaling_id)
	request.DesiredCapacity = common.Uint64Ptr(desired_count)
	request.MinSize = common.Uint64Ptr(min_count)
	request.MaxSize = common.Uint64Ptr(max_count)

	// The returned "resp" is an instance of the ModifyAutoScalingGroupResponse class which corresponds to the request object
	response, err := client.ModifyAutoScalingGroup(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func UpgradeCluster(secret_id string, secret_key string, region_id string, cluster_id string, version string) (*tke.UpdateClusterVersionResponse, error) {

	credential := common.NewCredential(secret_id, secret_key)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	client, _ := tke.NewClient(credential, region_id, cpf)

	// Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
	request := tke.NewUpdateClusterVersionRequest()

	request.ClusterId = common.StringPtr(cluster_id)
	request.DstVersion = common.StringPtr(version)

	// The returned "resp" is an instance of the UpdateClusterVersionResponse class which corresponds to the request object
	response, err := client.UpdateClusterVersion(request)

	if err != nil {
		return nil, err
	}

	return response, nil
}

// func CreateAutoScalingGroup(secret_id string, secret_key string, region_id string, auto_scaling_group_name string, launch_config_id string, vpc_id string, subnet_id string, max_size uint64, min_size uint64, desired_size uint64) (string, error) {

// 	// Required steps:
// 	// Instantiate an authentication object. The Tencent Cloud account key pair `secretId` and `secretKey` need to be passed in as the input parameters
// 	// This example uses the way to read from the environment variable, so you need to set these two values in the environment variable in advance
// 	// You can also write the key pair directly into the code, but be careful not to copy, upload, or share the code to others
// 	// Query the CAM key: https://console.cloud.tencent.com/cam/capi
// 	credential := common.NewCredential(secret_id, secret_key)

// 	// Optional steps:
// 	// Instantiate a client configuration object. You can specify the timeout period and other configuration items
// 	cpf := profile.NewClientProfile()
// 	cpf.HttpProfile.Endpoint = "as.tencentcloudapi.com"
// 	// Instantiate an client object
// 	// The second parameter is the region information. You can directly enter the string "ap-guangzhou" or import the preset constant
// 	client, _ := as.NewClient(credential, region_id, cpf)

// 	// Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
// 	request := as.NewCreateAutoScalingGroupRequest()

// 	request.AutoScalingGroupName = common.StringPtr(auto_scaling_group_name)
// 	request.LaunchConfigurationId = common.StringPtr(launch_config_id)
// 	request.VpcId = common.StringPtr(vpc_id)
// 	request.SubnetIds = common.StringPtrs([]string{subnet_id})
// 	//request.SubnetIds = common.StringPtrs(subnet_ids)
// 	request.MaxSize = common.Uint64Ptr(max_size)
// 	request.MinSize = common.Uint64Ptr(min_size)
// 	// request.DesiredCapacity = common.Uint64Ptr(desired_size)  // 1 이상이면, 오토스케일링 그룹 삭제가 불가능해진다. 그래서 0으로 세팅한다.
// 	request.DesiredCapacity = common.Uint64Ptr(0)

// 	println(request.ToJsonString())

// 	// The returned "resp" is an instance of the CreateAutoScalingGroupResponse class which corresponds to the request object
// 	response, err := client.CreateAutoScalingGroup(request)
// 	if _, ok := err.(*errors.TencentCloudSDKError); ok {
// 		fmt.Printf("An API error has returned: %s", err)
// 		return "", err
// 	}
// 	if err != nil {
// 		panic(err)
// 	}
// 	// A string return packet in JSON format is output
// 	fmt.Printf("%s", response.ToJsonString())
// 	return response.ToJsonString(), nil
// }

func GetAutoScalingGroup(secret_id string, secret_key string, region_id string, auto_scaling_group_id string) (*as.DescribeAutoScalingGroupsResponse, error) {
	// Required steps:
	// Instantiate an authentication object. The Tencent Cloud account key pair `secretId` and `secretKey` need to be passed in as the input parameters
	// This example uses the way to read from the environment variable, so you need to set these two values in the environment variable in advance
	// You can also write the key pair directly into the code, but be careful not to copy, upload, or share the code to others
	// Query the CAM key: https://console.cloud.tencent.com/cam/capi
	credential := common.NewCredential(secret_id, secret_key)

	// Optional steps:
	// Instantiate a client configuration object. You can specify the timeout period and other configuration items
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "as.tencentcloudapi.com"
	// Instantiate an client object
	// The second parameter is the region information. You can directly enter the string "ap-guangzhou" or import the preset constant
	client, _ := as.NewClient(credential, region_id, cpf)

	// Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
	request := as.NewDescribeAutoScalingGroupsRequest()

	request.AutoScalingGroupIds = common.StringPtrs([]string{auto_scaling_group_id})

	// The returned "resp" is an instance of the DescribeAutoScalingGroupsResponse class which corresponds to the request object
	response, err := client.DescribeAutoScalingGroups(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// func DisableAutoScalingGroup(secret_id string, secret_key string, region_id string, auto_scaling_group_id string) (string, error) {
// 	// Required steps:
// 	// Instantiate an authentication object. The Tencent Cloud account key pair `secretId` and `secretKey` need to be passed in as the input parameters
// 	// This example uses the way to read from the environment variable, so you need to set these two values in the environment variable in advance
// 	// You can also write the key pair directly into the code, but be careful not to copy, upload, or share the code to others
// 	// Query the CAM key: https://console.cloud.tencent.com/cam/capi
// 	credential := common.NewCredential(secret_id, secret_key)

// 	// Optional steps:
// 	// Instantiate a client configuration object. You can specify the timeout period and other configuration items
// 	cpf := profile.NewClientProfile()
// 	cpf.HttpProfile.Endpoint = "as.tencentcloudapi.com"
// 	// Instantiate an client object
// 	// The second parameter is the region information. You can directly enter the string "ap-guangzhou" or import the preset constant
// 	client, _ := as.NewClient(credential, region_id, cpf)

// 	// Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
// 	request := as.NewDisableAutoScalingGroupRequest()

// 	request.AutoScalingGroupId = common.StringPtr(auto_scaling_group_id)

// 	// The returned "resp" is an instance of the DisableAutoScalingGroupResponse class which corresponds to the request object
// 	response, err := client.DisableAutoScalingGroup(request)
// 	if _, ok := err.(*errors.TencentCloudSDKError); ok {
// 		fmt.Printf("An API error has returned: %s", err)
// 		return "", err
// 	}
// 	if err != nil {
// 		panic(err)
// 	}
// 	// A string return packet in JSON format is output
// 	fmt.Printf("%s", response.ToJsonString())
// 	return response.ToJsonString(), nil
// }

// func DeleteAutoScalingGroup(secret_id string, secret_key string, region_id string, auto_scaling_group_id string) (string, error) {

// 	// Required steps:
// 	// Instantiate an authentication object. The Tencent Cloud account key pair `secretId` and `secretKey` need to be passed in as the input parameters
// 	// This example uses the way to read from the environment variable, so you need to set these two values in the environment variable in advance
// 	// You can also write the key pair directly into the code, but be careful not to copy, upload, or share the code to others
// 	// Query the CAM key: https://console.cloud.tencent.com/cam/capi
// 	credential := common.NewCredential(secret_id, secret_key)

// 	// Optional steps:
// 	// Instantiate a client configuration object. You can specify the timeout period and other configuration items
// 	cpf := profile.NewClientProfile()
// 	cpf.HttpProfile.Endpoint = "as.tencentcloudapi.com"
// 	// Instantiate an client object
// 	// The second parameter is the region information. You can directly enter the string "ap-guangzhou" or import the preset constant
// 	client, _ := as.NewClient(credential, region_id, cpf)

// 	// Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
// 	request := as.NewDeleteAutoScalingGroupRequest()

// 	request.AutoScalingGroupId = common.StringPtr(auto_scaling_group_id)

// 	// The returned "resp" is an instance of the DeleteAutoScalingGroupResponse class which corresponds to the request object
// 	response, err := client.DeleteAutoScalingGroup(request)
// 	if _, ok := err.(*errors.TencentCloudSDKError); ok {
// 		fmt.Printf("An API error has returned: %s", err)
// 		return "", err
// 	}
// 	if err != nil {
// 		panic(err)
// 	}
// 	// A string return packet in JSON format is output
// 	fmt.Printf("%s", response.ToJsonString())
// 	return response.ToJsonString(), nil
// }

// func CreateLaunchConfiguration(secret_id string, secret_key string, region_id string, name string, instance_type string, image_id string, disk_type string, disk_size uint64) (string, error) {

// 	// Required steps:
// 	// Instantiate an authentication object. The Tencent Cloud account key pair `secretId` and `secretKey` need to be passed in as the input parameters
// 	// This example uses the way to read from the environment variable, so you need to set these two values in the environment variable in advance
// 	// You can also write the key pair directly into the code, but be careful not to copy, upload, or share the code to others
// 	// Query the CAM key: https://console.cloud.tencent.com/cam/capi
// 	credential := common.NewCredential(secret_id, secret_key)

// 	// Optional steps:
// 	// Instantiate a client configuration object. You can specify the timeout period and other configuration items
// 	cpf := profile.NewClientProfile()
// 	cpf.HttpProfile.Endpoint = "as.tencentcloudapi.com"
// 	// Instantiate an client object
// 	// The second parameter is the region information. You can directly enter the string "ap-guangzhou" or import the preset constant
// 	client, _ := as.NewClient(credential, region_id, cpf)

// 	// Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
// 	request := as.NewCreateLaunchConfigurationRequest()

// 	request.LaunchConfigurationName = common.StringPtr(name)
// 	request.InstanceType = common.StringPtr(instance_type)
// 	request.ImageId = common.StringPtr(image_id)
// 	request.SystemDisk = &as.SystemDisk{
// 		DiskType: common.StringPtr(disk_type),
// 		DiskSize: common.Uint64Ptr(disk_size),
// 	}
// 	request.SecurityGroupIds = common.StringPtrs([]string{"sg-46eef229"})

// 	println(request.ToJsonString())

// 	// The returned "resp" is an instance of the CreateLaunchConfigurationResponse class which corresponds to the request object
// 	response, err := client.CreateLaunchConfiguration(request)
// 	if _, ok := err.(*errors.TencentCloudSDKError); ok {
// 		fmt.Printf("An API error has returned: %s", err)
// 		return "", err
// 	}
// 	if err != nil {
// 		panic(err)
// 	}
// 	// A string return packet in JSON format is output
// 	fmt.Printf("%s", response.ToJsonString())

// 	return response.ToJsonString(), nil

// }

func GetLaunchConfiguration(secret_id string, secret_key string, region_id string, launch_config_id string) (*as.DescribeLaunchConfigurationsResponse, error) {
	// Required steps:
	// Instantiate an authentication object. The Tencent Cloud account key pair `secretId` and `secretKey` need to be passed in as the input parameters
	// This example uses the way to read from the environment variable, so you need to set these two values in the environment variable in advance
	// You can also write the key pair directly into the code, but be careful not to copy, upload, or share the code to others
	// Query the CAM key: https://console.cloud.tencent.com/cam/capi
	credential := common.NewCredential(secret_id, secret_key)

	// Optional steps:
	// Instantiate a client configuration object. You can specify the timeout period and other configuration items
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "as.tencentcloudapi.com"
	// Instantiate an client object
	// The second parameter is the region information. You can directly enter the string "ap-guangzhou" or import the preset constant
	client, _ := as.NewClient(credential, region_id, cpf)

	// Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
	request := as.NewDescribeLaunchConfigurationsRequest()

	request.LaunchConfigurationIds = common.StringPtrs([]string{launch_config_id})

	// The returned "resp" is an instance of the DescribeLaunchConfigurationsResponse class which corresponds to the request object
	response, err := client.DescribeLaunchConfigurations(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// func DeleteLaunchConfiguration(secret_id string, secret_key string, region_id string, launch_configuration_id string) string {

// 	// Required steps:
// 	// Instantiate an authentication object. The Tencent Cloud account key pair `secretId` and `secretKey` need to be passed in as the input parameters
// 	// This example uses the way to read from the environment variable, so you need to set these two values in the environment variable in advance
// 	// You can also write the key pair directly into the code, but be careful not to copy, upload, or share the code to others
// 	// Query the CAM key: https://console.cloud.tencent.com/cam/capi
// 	credential := common.NewCredential(secret_id, secret_key)

// 	// Optional steps:
// 	// Instantiate a client configuration object. You can specify the timeout period and other configuration items
// 	cpf := profile.NewClientProfile()
// 	cpf.HttpProfile.Endpoint = "as.tencentcloudapi.com"
// 	// Instantiate an client object
// 	// The second parameter is the region information. You can directly enter the string "ap-guangzhou" or import the preset constant
// 	client, _ := as.NewClient(credential, region_id, cpf)

// 	// Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
// 	request := as.NewDeleteLaunchConfigurationRequest()
// 	request.LaunchConfigurationId = common.StringPtr(launch_configuration_id)

// 	// The returned "resp" is an instance of the DeleteLaunchConfigurationResponse class which corresponds to the request object
// 	response, err := client.DeleteLaunchConfiguration(request)
// 	if _, ok := err.(*errors.TencentCloudSDKError); ok {
// 		fmt.Printf("An API error has returned: %s", err)
// 		return err.Error()
// 	}
// 	if err != nil {
// 		panic(err)
// 	}
// 	// A string return packet in JSON format is output
// 	fmt.Printf("%s", response.ToJsonString())

// 	return response.ToJsonString()
// }

//         // Instantiate a client configuration object. You can specify the timeout period and other configuration items
//         cpf := profile.NewClientProfile()
//         cpf.HttpProfile.Endpoint = "vpc.tencentcloudapi.com"
//         // Instantiate an client object
//         // The second parameter is the region information. You can directly enter the string "ap-guangzhou" or import the preset constant
//         client, _ := vpc.NewClient(credential, "ap-beijing", cpf)

//         // Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
//         request := vpc.NewDescribeSecurityGroupsRequest()

//         // The returned "resp" is an instance of the DescribeSecurityGroupsResponse class which corresponds to the request object
//         response, err := client.DescribeSecurityGroups(request)
//         if _, ok := err.(*errors.TencentCloudSDKError); ok {
//                 fmt.Printf("An API error has returned: %s", err)
//                 return
//         }
//         if err != nil {
//                 panic(err)
//         }
//         // A string return packet in JSON format is output
//         fmt.Printf("%s", response.ToJsonString())
// }

func DescribeSecurityGroups(secret_id string, secret_key string, region_id string) (*vpc.DescribeSecurityGroupsResponse, error) {

	credential := common.NewCredential(secret_id, secret_key)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "vpc.tencentcloudapi.com"

	client, _ := vpc.NewClient(credential, region_id, cpf)

	request := vpc.NewDescribeSecurityGroupsRequest()

	response, err := client.DescribeSecurityGroups(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

// {
// 	"Response": {
// 	  "RequestId": "d222da41-336b-4c36-aebf-0e7dd4493e06",
// 	  "SecurityGroupSet": [
// 		{
// 		  "CreatedTime": "2022-08-30 10:30:03",
// 		  "IsDefault": true,
// 		  "ProjectId": "0",
// 		  "SecurityGroupDesc": "System created security group",
// 		  "SecurityGroupId": "sg-46eef229",
// 		  "SecurityGroupName": "default",
// 		  "TagSet": [],
// 		  "UpdateTime": "2022-08-30 10:30:04"
// 		},
// 		{
// 		  "CreatedTime": "2022-08-30 10:27:56",
// 		  "IsDefault": false,
// 		  "ProjectId": "0",
// 		  "SecurityGroupDesc": "Host login and web service port open for Internet: all ports open for private network.",
// 		  "SecurityGroupId": "sg-esz2nqpj",
// 		  "SecurityGroupName": "TCP port 22, 80, 443, 3389 and ICMP open-2022083011275171687",
// 		  "TagSet": [],
// 		  "UpdateTime": "2022-08-30 10:27:58"
// 		}
// 	  ],
// 	  "TotalCount": 2
// 	}
//   }
