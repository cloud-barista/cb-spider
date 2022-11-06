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
	"strings"

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

func GetClusterKubeConfig(access_key string, access_secret string, region_id string, cluster_id string) (string, error) {

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
	request.PathPattern = "/k8s/" + cluster_id + "/user_config"
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

// package main

// import (
// 	"fmt"

// 	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
// 	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
// )

func DescribeClusterNodes(access_key string, access_secret string, region_id string, cluster_id string, nodepool_id string) (string, error) {

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
	request.PathPattern = "/clusters/" + cluster_id + "/nodes"
	request.Headers["Content-Type"] = "application/json"
	request.QueryParams["nodepool_id"] = nodepool_id

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
	request.QueryParams["force"] = "true"

	// 노드그룹을 삭제 또는 오토스케일러 변경을 최초 호출하면 에러가 발생한다.
	// 어떻게 하면 좋은가?
	// 첫번째 호출에서 에러가 없으면 리턴! // 최초 호출이 아닌 경우
	// 첫번째 호출에서 에러 발생하면 한번 더 호출! // 최초 호출인 경우
	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "cluster-autoscaler") {
			response, err = client.ProcessCommonRequest(request)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
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
	request.Content = []byte(body)

	// 노드그룹을 삭제 또는 오토스케일러 변경을 최초 호출하면 에러가 발생한다.
	// 어떻게 하면 좋은가?
	// 첫번째 호출에서 에러가 없으면 리턴! // 최초 호출이 아닌 경우
	// 첫번째 호출에서 에러 발생하면 한번 더 호출! // 최초 호출인 경우
	response, err := client.ProcessCommonRequest(request)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "cluster-autoscaler") {
			response, err = client.ProcessCommonRequest(request)
			if err != nil {
				return "", err
			}
		} else {
			return "", err
		}
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
