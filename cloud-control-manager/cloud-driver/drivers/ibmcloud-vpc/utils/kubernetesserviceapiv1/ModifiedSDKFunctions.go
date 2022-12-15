/**
 * Copy and modified from IBM Cloud Container Services Go SDK Version 1.0
 * Original source: https://github.com/IBM-Cloud/container-services-go-sdk
 * Original source copied date: 2022. 12. 01
 */

/**
 * (C) Copyright IBM Corp. 2021.
 * Modifications Copyright (C) 2022 - Cloud-Barista Community (https://cloud-barista.github.io)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/*
 * IBM OpenAPI SDK Code Generator Version: 3.28.0-55613c9e-20210220-164656
 */

package kubernetesserviceapiv1

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/vpc-go-sdk/common"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func isJsonObjectOrArray(rawResult []byte) (bool, bool) {
	x := bytes.TrimLeft(rawResult, " \t\r\n")
	isObject := len(x) > 0 && x[0] == '{'
	isArray := len(x) > 0 && x[0] == '['

	return isObject, isArray
}

// VpcGetCluster : Get details of a VPC cluster
// Get details of a VPC cluster.
func (kubernetesServiceApi *KubernetesServiceApiV1) VpcGetCluster(vpcGetClusterOptions *VpcGetClusterOptions) (result *[]GetClusterDetailResponse, response *core.DetailedResponse, err error) {
	return kubernetesServiceApi.VpcGetClusterWithContext(context.Background(), vpcGetClusterOptions)
}

// VpcGetClusterWithContext is an alternate form of the VpcGetCluster method which supports a Context parameter
func (kubernetesServiceApi *KubernetesServiceApiV1) VpcGetClusterWithContext(ctx context.Context, vpcGetClusterOptions *VpcGetClusterOptions) (result *[]GetClusterDetailResponse, response *core.DetailedResponse, err error) {
	err = core.ValidateNotNil(vpcGetClusterOptions, "vpcGetClusterOptions cannot be nil")
	if err != nil {
		return
	}
	err = core.ValidateStruct(vpcGetClusterOptions, "vpcGetClusterOptions")
	if err != nil {
		return
	}

	builder := core.NewRequestBuilder(core.GET)
	builder = builder.WithContext(ctx)
	builder.EnableGzipCompression = kubernetesServiceApi.GetEnableGzipCompression()
	_, err = builder.ResolveRequestURL(kubernetesServiceApi.Service.Options.URL, `/v2/vpc/getCluster`, nil)
	if err != nil {
		return
	}

	for headerName, headerValue := range vpcGetClusterOptions.Headers {
		builder.AddHeader(headerName, headerValue)
	}

	sdkHeaders := common.GetSdkHeaders("kubernetes_service_api", "V1", "VpcGetCluster")
	for headerName, headerValue := range sdkHeaders {
		builder.AddHeader(headerName, headerValue)
	}
	builder.AddHeader("Accept", "application/json")
	if vpcGetClusterOptions.XAuthResourceGroup != nil {
		builder.AddHeader("X-Auth-Resource-Group", fmt.Sprint(*vpcGetClusterOptions.XAuthResourceGroup))
	}

	builder.AddQuery("cluster", fmt.Sprint(*vpcGetClusterOptions.Cluster))
	if vpcGetClusterOptions.ShowResources != nil {
		builder.AddQuery("showResources", fmt.Sprint(*vpcGetClusterOptions.ShowResources))
	}

	request, err := builder.Build()
	if err != nil {
		return
	}

	var rawResponse []json.RawMessage
	result = &[]GetClusterDetailResponse{}
	response, err = kubernetesServiceApi.Service.Request(request, &rawResponse)
	if !(response.StatusCode < 200 || response.StatusCode >= 300) {
		isObject, isArray := isJsonObjectOrArray(response.RawResult)
		if isArray {
			err = json.Unmarshal(response.RawResult, result)
			if err != nil {
				return
			}
			return
		}
		if isObject {
			var detailResponse GetClusterDetailResponse
			err = json.Unmarshal(response.RawResult, &detailResponse)
			if err != nil {
				return
			}
			*result = append(*result, detailResponse)
			return
		}

		err = errors.New("JSON Response is not an Object nor an Array")
		return
	}

	return
}

// VpcGetWorkerPools : List the worker pools in a VPC cluster
func (kubernetesServiceApi *KubernetesServiceApiV1) VpcGetWorkerPools(vpcGetWorkerPoolsOptions *VpcGetWorkerPoolsOptions) (result *[]GetWorkerPoolsDetailResponse, response *core.DetailedResponse, err error) {
	return kubernetesServiceApi.VpcGetWorkerPoolsWithContext(context.Background(), vpcGetWorkerPoolsOptions)
}

// VpcGetWorkerPoolsWithContext is an alternate form of the VpcGetWorkerPools method which supports a Context parameter
func (kubernetesServiceApi *KubernetesServiceApiV1) VpcGetWorkerPoolsWithContext(ctx context.Context, vpcGetWorkerPoolsOptions *VpcGetWorkerPoolsOptions) (result *[]GetWorkerPoolsDetailResponse, response *core.DetailedResponse, err error) {
	err = core.ValidateNotNil(vpcGetWorkerPoolsOptions, "vpcGetWorkerPoolsOptions cannot be nil")
	if err != nil {
		return
	}
	err = core.ValidateStruct(vpcGetWorkerPoolsOptions, "vpcGetWorkerPoolsOptions")
	if err != nil {
		return
	}

	builder := core.NewRequestBuilder(core.GET)
	builder = builder.WithContext(ctx)
	builder.EnableGzipCompression = kubernetesServiceApi.GetEnableGzipCompression()
	_, err = builder.ResolveRequestURL(kubernetesServiceApi.Service.Options.URL, `/v2/vpc/getWorkerPools`, nil)
	if err != nil {
		return
	}

	for headerName, headerValue := range vpcGetWorkerPoolsOptions.Headers {
		builder.AddHeader(headerName, headerValue)
	}

	sdkHeaders := common.GetSdkHeaders("kubernetes_service_api", "V1", "VpcGetWorkerPools")
	for headerName, headerValue := range sdkHeaders {
		builder.AddHeader(headerName, headerValue)
	}
	builder.AddHeader("Accept", "application/json")
	if vpcGetWorkerPoolsOptions.XRegion != nil {
		builder.AddHeader("X-Region", fmt.Sprint(*vpcGetWorkerPoolsOptions.XRegion))
	}
	if vpcGetWorkerPoolsOptions.XAuthResourceGroup != nil {
		builder.AddHeader("X-Auth-Resource-Group", fmt.Sprint(*vpcGetWorkerPoolsOptions.XAuthResourceGroup))
	}

	builder.AddQuery("cluster", fmt.Sprint(*vpcGetWorkerPoolsOptions.Cluster))

	request, err := builder.Build()
	if err != nil {
		return
	}

	var rawResponse map[string]json.RawMessage
	result = &[]GetWorkerPoolsDetailResponse{}
	response, err = kubernetesServiceApi.Service.Request(request, &rawResponse)
	if !(response.StatusCode < 200 || response.StatusCode >= 300) {
		isObject, isArray := isJsonObjectOrArray(response.RawResult)
		if isArray {
			err = json.Unmarshal(response.RawResult, result)
			if err != nil {
				return
			}
			return
		}
		if isObject {
			var detailResponse GetWorkerPoolsDetailResponse
			err = json.Unmarshal(response.RawResult, &detailResponse)
			if err != nil {
				return
			}
			*result = append(*result, detailResponse)
			return
		}

		err = errors.New("JSON Response is not an Object nor an Array")
		return
	}

	return
}

// GetKubeconfig : Get the cluster's kubeconfig file
// Get the cluster's Kubernetes configuration file (`kubeconfig`) to connect to your cluster and run Kubernetes API
// calls. You can also get the networking and admin configuration files for the cluster.
func (kubernetesServiceApi *KubernetesServiceApiV1) GetKubeconfig(getKubeconfigOptions *GetKubeconfigOptions) (response *core.DetailedResponse, err error) {
	return kubernetesServiceApi.GetKubeconfigWithContext(context.Background(), getKubeconfigOptions)
}

// GetKubeconfigWithContext is an alternate form of the GetKubeconfig method which supports a Context parameter
func (kubernetesServiceApi *KubernetesServiceApiV1) GetKubeconfigWithContext(ctx context.Context, getKubeconfigOptions *GetKubeconfigOptions) (response *core.DetailedResponse, err error) {
	err = core.ValidateNotNil(getKubeconfigOptions, "getKubeconfigOptions cannot be nil")
	if err != nil {
		return
	}
	err = core.ValidateStruct(getKubeconfigOptions, "getKubeconfigOptions")
	if err != nil {
		return
	}

	builder := core.NewRequestBuilder(core.GET)
	builder = builder.WithContext(ctx)
	builder.EnableGzipCompression = kubernetesServiceApi.GetEnableGzipCompression()
	_, err = builder.ResolveRequestURL(kubernetesServiceApi.Service.Options.URL, `/v2/getKubeconfig`, nil)
	if err != nil {
		return
	}

	for headerName, headerValue := range getKubeconfigOptions.Headers {
		builder.AddHeader(headerName, headerValue)
	}

	sdkHeaders := common.GetSdkHeaders("kubernetes_service_api", "V1", "GetKubeconfig")
	for headerName, headerValue := range sdkHeaders {
		builder.AddHeader(headerName, headerValue)
	}
	if getKubeconfigOptions.Authorization != nil {
		builder.AddHeader("Authorization", fmt.Sprint(*getKubeconfigOptions.Authorization))
	}
	if getKubeconfigOptions.XAuthResourceGroup != nil {
		builder.AddHeader("X-Auth-Resource-Group", fmt.Sprint(*getKubeconfigOptions.XAuthResourceGroup))
	}

	builder.AddQuery("cluster", fmt.Sprint(*getKubeconfigOptions.Cluster))
	if getKubeconfigOptions.Format != nil {
		builder.AddQuery("format", fmt.Sprint(*getKubeconfigOptions.Format))
	}
	if getKubeconfigOptions.Admin != nil {
		builder.AddQuery("admin", fmt.Sprint(*getKubeconfigOptions.Admin))
	}
	if getKubeconfigOptions.Network != nil {
		builder.AddQuery("network", fmt.Sprint(*getKubeconfigOptions.Network))
	}

	request, err := builder.Build()
	if err != nil {
		return
	}

	var fileContent []byte
	response, err = kubernetesServiceApi.Service.Request(request, &fileContent)

	return
}

// GetKubernetesClient : Newly created, Get client-go Kubernetes Client from given KubeConfig
func (kubernetesServiceApi *KubernetesServiceApiV1) GetKubernetesClient(kubeConfig string) (kubeClient *kubernetes.Clientset, err error) {
	clientConfig, err := clientcmd.NewClientConfigFromBytes([]byte(kubeConfig))
	if err != nil {
		return nil, err
	}
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(restConfig)
}

// NewGetKubeconfigOptions : Instantiate GetKubeconfigOptions
func (*KubernetesServiceApiV1) NewGetKubeconfigOptions(authorization string, cluster string) *GetKubeconfigOptions {
	return &GetKubeconfigOptions{
		Authorization: core.StringPtr(authorization),
		Cluster:       core.StringPtr(cluster),
	}
}

// SetAuthorization : Allow user to set Autorization
func (options *GetKubeconfigOptions) SetAuthorization(authorization string) *GetKubeconfigOptions {
	options.Authorization = core.StringPtr(authorization)
	return options
}

// VpcCreateWorkerPool : Create a worker pool for a VPC cluster
// Create a worker pool for the specified VPC cluster. Creating a worker pool requires Operator access to Kubernetes
// Service in the IBM Cloud account.
func (kubernetesServiceApi *KubernetesServiceApiV1) VpcCreateWorkerPool(vpcCreateWorkerPoolOptions *VpcCreateWorkerPoolOptions) (result *CreateWorkerpoolResponse, response *core.DetailedResponse, err error) {
	return kubernetesServiceApi.VpcCreateWorkerPoolWithContext(context.Background(), vpcCreateWorkerPoolOptions)
}

// VpcCreateWorkerPoolWithContext is an alternate form of the VpcCreateWorkerPool method which supports a Context parameter
func (kubernetesServiceApi *KubernetesServiceApiV1) VpcCreateWorkerPoolWithContext(ctx context.Context, vpcCreateWorkerPoolOptions *VpcCreateWorkerPoolOptions) (result *CreateWorkerpoolResponse, response *core.DetailedResponse, err error) {
	err = core.ValidateNotNil(vpcCreateWorkerPoolOptions, "vpcCreateWorkerPoolOptions cannot be nil")
	if err != nil {
		return
	}
	err = core.ValidateStruct(vpcCreateWorkerPoolOptions, "vpcCreateWorkerPoolOptions")
	if err != nil {
		return
	}

	builder := core.NewRequestBuilder(core.POST)
	builder = builder.WithContext(ctx)
	builder.EnableGzipCompression = kubernetesServiceApi.GetEnableGzipCompression()
	_, err = builder.ResolveRequestURL(kubernetesServiceApi.Service.Options.URL, `/v2/vpc/createWorkerPool`, nil)
	if err != nil {
		return
	}

	for headerName, headerValue := range vpcCreateWorkerPoolOptions.Headers {
		builder.AddHeader(headerName, headerValue)
	}

	sdkHeaders := common.GetSdkHeaders("kubernetes_service_api", "V1", "VpcCreateWorkerPool")
	for headerName, headerValue := range sdkHeaders {
		builder.AddHeader(headerName, headerValue)
	}
	builder.AddHeader("Accept", "application/json")
	builder.AddHeader("Content-Type", "application/json")
	if vpcCreateWorkerPoolOptions.Authorization != nil {
		builder.AddHeader("Authorization", fmt.Sprint(*vpcCreateWorkerPoolOptions.Authorization))
	}
	if vpcCreateWorkerPoolOptions.XAuthResourceGroup != nil {
		builder.AddHeader("X-Auth-Resource-Group", fmt.Sprint(*vpcCreateWorkerPoolOptions.XAuthResourceGroup))
	}

	body := make(map[string]interface{})
	if vpcCreateWorkerPoolOptions.Cluster != nil {
		body["cluster"] = vpcCreateWorkerPoolOptions.Cluster
	}
	if vpcCreateWorkerPoolOptions.DiskEncryption != nil {
		body["diskEncryption"] = vpcCreateWorkerPoolOptions.DiskEncryption
	}
	if vpcCreateWorkerPoolOptions.Entitlement != nil {
		body["entitlement"] = vpcCreateWorkerPoolOptions.Entitlement
	}
	if vpcCreateWorkerPoolOptions.Flavor != nil {
		body["flavor"] = vpcCreateWorkerPoolOptions.Flavor
	}
	if vpcCreateWorkerPoolOptions.Isolation != nil {
		body["isolation"] = vpcCreateWorkerPoolOptions.Isolation
	}
	if vpcCreateWorkerPoolOptions.KmsInstanceID != nil {
		body["kmsInstanceID"] = vpcCreateWorkerPoolOptions.KmsInstanceID
	}
	if vpcCreateWorkerPoolOptions.Labels != nil {
		body["labels"] = vpcCreateWorkerPoolOptions.Labels
	}
	if vpcCreateWorkerPoolOptions.Name != nil {
		body["name"] = vpcCreateWorkerPoolOptions.Name
	}
	if vpcCreateWorkerPoolOptions.VpcID != nil {
		body["vpcID"] = vpcCreateWorkerPoolOptions.VpcID
	}
	if vpcCreateWorkerPoolOptions.WorkerCount != nil {
		body["workerCount"] = vpcCreateWorkerPoolOptions.WorkerCount
	}
	if vpcCreateWorkerPoolOptions.WorkerVolumeCRKID != nil {
		body["workerVolumeCRKID"] = vpcCreateWorkerPoolOptions.WorkerVolumeCRKID
	}
	if vpcCreateWorkerPoolOptions.Zones != nil {
		body["zones"] = vpcCreateWorkerPoolOptions.Zones
	}
	_, err = builder.SetBodyContentJSON(body)
	if err != nil {
		return
	}

	request, err := builder.Build()
	if err != nil {
		return
	}

	var rawResponse map[string]json.RawMessage
	response, err = kubernetesServiceApi.Service.Request(request, &rawResponse)
	if err != nil {
		return
	}
	err = core.UnmarshalModel(rawResponse, "", &result, UnmarshalCreateWorkerpoolResponse)
	if err != nil {
		return
	}
	response.Result = result

	return
}

// SetXAuthRefreshToken : Allow user to set XAuthRefreshToken
func (options *VpcCreateWorkerPoolOptions) SetXAuthRefreshToken(xAuthRefreshToken string) *VpcCreateWorkerPoolOptions {
	options.Authorization = core.StringPtr(xAuthRefreshToken)
	return options
}
