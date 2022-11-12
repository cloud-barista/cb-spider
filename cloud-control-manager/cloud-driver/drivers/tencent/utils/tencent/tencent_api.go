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

	credential := common.NewCredential(secret_id, secret_key)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	client, _ := tke.NewClient(credential, region_id, cpf)

	response, err := client.CreateCluster(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func GetClusters(secret_id string, secret_key string, region_id string) (*tke.DescribeClustersResponse, error) {
	credential := common.NewCredential(secret_id, secret_key)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	client, _ := tke.NewClient(credential, region_id, cpf)

	request := tke.NewDescribeClustersRequest()
	response, err := client.DescribeClusters(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func GetCluster(secret_id string, secret_key string, region_id string, cluster_id string) (*tke.DescribeClustersResponse, error) {

	credential := common.NewCredential(secret_id, secret_key)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	client, _ := tke.NewClient(credential, region_id, cpf)

	request := tke.NewDescribeClustersRequest()
	request.ClusterIds = common.StringPtrs([]string{cluster_id})
	response, err := client.DescribeClusters(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func DeleteCluster(secret_id string, secret_key string, region_id string, cluster_id string) (*tke.DeleteClusterResponse, error) {

	credential := common.NewCredential(secret_id, secret_key)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	client, _ := tke.NewClient(credential, region_id, cpf)
	request := tke.NewDeleteClusterRequest()

	request.ClusterId = common.StringPtr(cluster_id)
	request.InstanceDeleteMode = common.StringPtr("terminate") // or "retain"
	response, err := client.DeleteCluster(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func CreateNodeGroup(secret_id string, secret_key string, region_id string, request *tke.CreateClusterNodePoolRequest) (*tke.CreateClusterNodePoolResponse, error) {
	credential := common.NewCredential(secret_id, secret_key)

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	client, _ := tke.NewClient(credential, region_id, cpf)
	response, err := client.CreateClusterNodePool(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func ListNodeGroup(secret_id string, secret_key string, region_id string, cluster_id string) (*tke.DescribeClusterNodePoolsResponse, error) {

	credential := common.NewCredential(secret_id, secret_key)

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	client, _ := tke.NewClient(credential, region_id, cpf)

	request := tke.NewDescribeClusterNodePoolsRequest()
	request.ClusterId = common.StringPtr(cluster_id)
	response, err := client.DescribeClusterNodePools(request)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func GetNodeGroup(secret_id string, secret_key string, region_id string, cluster_id string, nodegroup_id string) (*tke.DescribeClusterNodePoolDetailResponse, error) {
	credential := common.NewCredential(secret_id, secret_key)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	client, _ := tke.NewClient(credential, region_id, cpf)

	request := tke.NewDescribeClusterNodePoolDetailRequest()
	request.ClusterId = common.StringPtr(cluster_id)
	request.NodePoolId = common.StringPtr(nodegroup_id)

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

	request := tke.NewDeleteClusterNodePoolRequest()

	request.ClusterId = common.StringPtr(cluster_id)
	request.NodePoolIds = common.StringPtrs([]string{nodepool_id})
	request.KeepInstance = common.BoolPtr(false)

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

	request := tke.NewModifyClusterNodePoolRequest()

	request.EnableAutoscale = common.BoolPtr(false)
	request.ClusterId = common.StringPtr(cluster_id)
	request.NodePoolId = common.StringPtr(nodepool_id)
	request.EnableAutoscale = common.BoolPtr(enable)

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
	client, _ := as.NewClient(credential, region_id, cpf)

	request := as.NewModifyAutoScalingGroupRequest()
	request.AutoScalingGroupId = common.StringPtr(autoscaling_id)
	request.DesiredCapacity = common.Uint64Ptr(desired_count)
	request.MinSize = common.Uint64Ptr(min_count)
	request.MaxSize = common.Uint64Ptr(max_count)

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

	request := tke.NewUpdateClusterVersionRequest()
	request.ClusterId = common.StringPtr(cluster_id)
	request.DstVersion = common.StringPtr(version)

	response, err := client.UpdateClusterVersion(request)

	if err != nil {
		return nil, err
	}

	return response, nil
}

func GetAutoScalingGroup(secret_id string, secret_key string, region_id string, auto_scaling_group_id string) (*as.DescribeAutoScalingGroupsResponse, error) {
	credential := common.NewCredential(secret_id, secret_key)

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "as.tencentcloudapi.com"
	client, _ := as.NewClient(credential, region_id, cpf)

	request := as.NewDescribeAutoScalingGroupsRequest()
	request.AutoScalingGroupIds = common.StringPtrs([]string{auto_scaling_group_id})

	response, err := client.DescribeAutoScalingGroups(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

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

func DescribeClusterInstances(secret_id string, secret_key string, region_id string, cluster_id string) (*tke.DescribeClusterInstancesResponse, error) {

	credential := common.NewCredential(secret_id, secret_key)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
	client, _ := tke.NewClient(credential, region_id, cpf)

	request := tke.NewDescribeClusterInstancesRequest()
	request.ClusterId = common.StringPtr(cluster_id)

	response, err := client.DescribeClusterInstances(request)
	if err != nil {
		panic(err)
	}

	if err != nil {
		return nil, err
	}

	return response, nil
}

func CreateClusterEndpoint(secret_id string, secret_key string, region_id string, 
		cluster_id string, security_group_id string) (*tke.CreateClusterEndpointResponse, error) {
        credential := common.NewCredential(secret_id, secret_key)
        cpf := profile.NewClientProfile()
        cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
        client, _ := tke.NewClient(credential, region_id, cpf)

        request := tke.NewCreateClusterEndpointRequest()
        request.ClusterId = common.StringPtr(cluster_id)
        request.SecurityGroup = common.StringPtr(security_group_id)
        request.IsExtranet = common.BoolPtr(true)
        response, err := client.CreateClusterEndpoint(request)
        if err != nil {
                return nil, err
        }

        return response, nil
}

func GetClusterEndpoint(secret_id string, secret_key string, region_id string, cluster_id string) (*tke.DescribeClusterEndpointsResponse, error) {
        credential := common.NewCredential(secret_id, secret_key)
        cpf := profile.NewClientProfile()
        cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
        client, _ := tke.NewClient(credential, region_id, cpf)

        request := tke.NewDescribeClusterEndpointsRequest()
	request.ClusterId = common.StringPtr(cluster_id)
	response, err := client.DescribeClusterEndpoints(request)
        if err != nil {
                return nil, err
        }

        return response, nil
}

func GetClusterKubeconfig(secret_id string, secret_key string, region_id string, cluster_id string) (*tke.DescribeClusterKubeconfigResponse, error) {
        credential := common.NewCredential(secret_id, secret_key)
        cpf := profile.NewClientProfile()
        cpf.HttpProfile.Endpoint = "tke.tencentcloudapi.com"
        client, _ := tke.NewClient(credential, region_id, cpf)

        request := tke.NewDescribeClusterKubeconfigRequest()
        request.ClusterId = common.StringPtr(cluster_id)
        response, err := client.DescribeClusterKubeconfig(request)
        if err != nil {
                return nil, err
        }

        return response, nil
}
