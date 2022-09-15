// Alibaba Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Alibaba Driver.
//
// by CB-Spider Team, 2022.09.

package alibaba

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	vpc "github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
)

func CreateCluster(access_key string, access_secret string, region_id string, body string) (string, error) {
	config := sdk.NewConfig()
	credential := credentials.NewAccessKeyCredential(access_key, access_secret)
	client, err := sdk.NewClientWithOptions(region_id, config, credential)
	if err != nil {
		panic(err)
	}

	request := requests.NewCommonRequest()

	request.Method = "POST"
	request.Scheme = "https" // https | http
	request.Domain = "cs." + region_id + ".aliyuncs.com"
	request.Version = "2015-12-15"
	request.PathPattern = "/clusters"
	request.Headers["Content-Type"] = "application/json"

	request.Content = []byte(body)

	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		return "", err
	}

	return response.GetHttpContentString(), nil
}

func GetClusters(access_key string, access_secret string, region_id string) (string, error) {

	config := sdk.NewConfig()
	credential := credentials.NewAccessKeyCredential(access_key, access_secret)
	client, err := sdk.NewClientWithOptions(region_id, config, credential)
	if err != nil {
		panic(err)
	}

	request := requests.NewCommonRequest()

	request.Method = "GET"
	request.Scheme = "https" // https | http
	request.Domain = "cs." + region_id + ".aliyuncs.com"
	request.Version = "2015-12-15"
	request.PathPattern = "/api/v1/clusters"
	request.Headers["Content-Type"] = "application/json"

	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		return "", err
	}

	return response.GetHttpContentString(), nil
}

func GetCluster(access_key string, access_secret string, region_id string, cluster_id string) (string, error) {

	config := sdk.NewConfig()
	credential := credentials.NewAccessKeyCredential(access_key, access_secret)
	client, err := sdk.NewClientWithOptions(region_id, config, credential)
	if err != nil {
		return "", err
	}

	request := requests.NewCommonRequest()

	request.Method = "GET"
	request.Scheme = "https" // https | http
	request.Domain = "cs." + region_id + ".aliyuncs.com"
	request.Version = "2015-12-15"
	request.PathPattern = "/clusters/" + cluster_id
	request.Headers["Content-Type"] = "application/json"

	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		return "", err
	}

	return response.GetHttpContentString(), nil
}

func DeleteCluster(access_key string, access_secret string, region_id string, cluster_id string) (string, error) {

	config := sdk.NewConfig()
	credential := credentials.NewAccessKeyCredential(access_key, access_secret)
	client, err := sdk.NewClientWithOptions(region_id, config, credential)
	if err != nil {
		panic(err)
	}

	request := requests.NewCommonRequest()

	request.Method = "DELETE"
	request.Scheme = "https" // https | http
	request.Domain = "cs." + region_id + ".aliyuncs.com"
	request.Version = "2015-12-15"
	request.PathPattern = "/clusters/" + cluster_id
	request.Headers["Content-Type"] = "application/json"

	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		return "", err
	}

	return response.GetHttpContentString(), nil
}

func CreateNodeGroup(access_key string, access_secret string, region_id string, cluster_id string, body string) (string, error) {

	config := sdk.NewConfig()
	credential := credentials.NewAccessKeyCredential(access_key, access_secret)
	client, err := sdk.NewClientWithOptions(region_id, config, credential)
	if err != nil {
		return "", err
	}

	request := requests.NewCommonRequest()

	request.Method = "POST"
	request.Scheme = "https" // https | http
	request.Domain = "cs." + region_id + ".aliyuncs.com"
	request.Version = "2015-12-15"
	request.PathPattern = "/clusters/" + cluster_id + "/nodepools"
	request.Headers["Content-Type"] = "application/json"

	request.Content = []byte(body)

	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		return "", err
	}

	return response.GetHttpContentString(), nil
}

func ListNodeGroup(access_key string, access_secret string, region_id string, cluster_id string) (string, error) {

	config := sdk.NewConfig()
	credential := credentials.NewAccessKeyCredential(access_key, access_secret)
	client, err := sdk.NewClientWithOptions(region_id, config, credential)
	if err != nil {
		return "", err
	}

	request := requests.NewCommonRequest()

	request.Method = "GET"
	request.Scheme = "https" // https | http
	request.Domain = "cs." + region_id + ".aliyuncs.com"
	request.Version = "2015-12-15"
	request.PathPattern = "/clusters/" + cluster_id + "/nodepools"
	request.Headers["Content-Type"] = "application/json"

	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		return "", err
	}

	return response.GetHttpContentString(), nil
}

func GetNodeGroup(access_key string, access_secret string, region_id string, cluster_id string, nodepool_id string) (string, error) {
	config := sdk.NewConfig()
	credential := credentials.NewAccessKeyCredential(access_key, access_secret)
	client, err := sdk.NewClientWithOptions(region_id, config, credential)
	if err != nil {
		return "", err
	}

	request := requests.NewCommonRequest()

	request.Method = "GET"
	request.Scheme = "https" // https | http
	request.Domain = "cs." + region_id + ".aliyuncs.com"
	request.Version = "2015-12-15"
	request.PathPattern = "/clusters/" + cluster_id + "/nodepools/" + nodepool_id
	request.Headers["Content-Type"] = "application/json"

	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		return "", err
	}

	return response.GetHttpContentString(), nil
}

func DeleteNodeGroup(access_key string, access_secret string, region_id string, cluster_id string, nodepool_id string) (string, error) {
	config := sdk.NewConfig()
	credential := credentials.NewAccessKeyCredential(access_key, access_secret)
	client, err := sdk.NewClientWithOptions(region_id, config, credential)
	if err != nil {
		return "", err
	}

	request := requests.NewCommonRequest()

	request.Method = "DELETE"
	request.Scheme = "https" // https | http
	request.Domain = "cs." + region_id + ".aliyuncs.com"
	request.Version = "2015-12-15"
	request.PathPattern = "/clusters/" + cluster_id + "/nodepools/" + nodepool_id
	request.Headers["Content-Type"] = "application/json"

	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		return "", err
	}

	return response.GetHttpContentString(), nil
}

func ModifyNodeGroup(access_key string, access_secret string, region_id string, cluster_id string, nodepool_id string, body string) (string, error) {

	config := sdk.NewConfig()
	credential := credentials.NewAccessKeyCredential(access_key, access_secret)
	client, err := sdk.NewClientWithOptions(region_id, config, credential)
	if err != nil {
		return "", err
	}

	request := requests.NewCommonRequest()

	request.Method = "PUT"
	request.Scheme = "https" // https | http
	request.Domain = "cs." + region_id + ".aliyuncs.com"
	request.Version = "2015-12-15"
	request.PathPattern = "/clusters/" + cluster_id + "/nodepools/" + nodepool_id
	request.Headers["Content-Type"] = "application/json"

	// body := `{"auto_scaling":{"enable":false}}`
	// body := `{"auto_scaling":{"enable":true}}`
	// body := `{"auto_scaling":{"max_instances":5,"min_instances":0},"scaling_group":{"desired_size":2}}`
	request.Content = []byte(body)

	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		return "", err
	}

	return response.GetHttpContentString(), nil
}

func UpgradeCluster(access_key string, access_secret string, region_id string, cluster_id string, body string) (string, error) {

	config := sdk.NewConfig()
	credential := credentials.NewAccessKeyCredential(access_key, access_secret)
	client, err := sdk.NewClientWithOptions(region_id, config, credential)
	if err != nil {
		return "", err
	}

	request := requests.NewCommonRequest()

	request.Method = "POST"
	request.Scheme = "https" // https | http
	request.Domain = "cs." + region_id + ".aliyuncs.com"
	request.Version = "2015-12-15"
	request.PathPattern = "/api/v2/clusters/" + cluster_id + "/upgrade"
	request.Headers["Content-Type"] = "application/json"

	request.Content = []byte(body)

	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		return "", err
	}

	return response.GetHttpContentString(), nil
}

func DescribeVSwitches(access_key string, access_secret string, region_id string, vpc_id string) (*vpc.DescribeVSwitchesResponse, error) {

	config := sdk.NewConfig()
	credential := credentials.NewAccessKeyCredential(access_key, access_secret)
	client, err := vpc.NewClientWithOptions(region_id, config, credential)
	if err != nil {
		return nil, err
	}

	request := vpc.CreateDescribeVSwitchesRequest()
	request.Scheme = "https"
	request.VpcId = vpc_id
	response, err := client.DescribeVSwitches(request)
	if err != nil {
		return nil, err
	}

	return response, nil
}
