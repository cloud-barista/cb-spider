/**
 * Copy and modified code from IBM Cloud Container Services Go SDK Version 1.0
 * Original source: https://github.com/IBM-Cloud/container-services-go-sdk
 * Original source copied date: 2022. 12. 01
 * /

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

// GetClusterDetailResponse Newly created for manual unmarshal JSON response
type GetClusterDetailResponse struct {
	Id                          string `json:"id"`
	Name                        string `json:"name"`
	Region                      string `json:"region"`
	ResourceGroup               string `json:"resourceGroup"`
	ResourceGroupName           string `json:"resourceGroupName"`
	PodSubnet                   string `json:"podSubnet"`
	ServiceSubnet               string `json:"serviceSubnet"`
	CreatedDate                 string `json:"createdDate"`
	MasterKubeVersion           string `json:"masterKubeVersion"`
	TargetVersion               string `json:"targetVersion"`
	WorkerCount                 int    `json:"workerCount"`
	Location                    string `json:"location"`
	Datacenter                  string `json:"datacenter"`
	MultiAzCapable              bool   `json:"multiAzCapable"`
	Provider                    string `json:"provider"`
	CalicoIPAutodetectionConfig string `json:"calicoIPAutodetectionConfig"`
	State                       string `json:"state"`
	Status                      string `json:"status"`
	VersionEOS                  string `json:"versionEOS"`
	IsPaid                      bool   `json:"isPaid"`
	Entitlement                 string `json:"entitlement"`
	Type                        string `json:"type"`
	Addons                      []struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"addons"`
	EtcdPort  string `json:"etcdPort"`
	MasterURL string `json:"masterURL"`
	Ingress   struct {
		Hostname   string `json:"hostname"`
		SecretName string `json:"secretName"`
		Status     string `json:"status"`
		Message    string `json:"message"`
	} `json:"ingress"`
	CaCertRotationStatus struct {
		Status              string `json:"status"`
		ActionTriggerDate   string `json:"actionTriggerDate"`
		ActionCompletedDate string `json:"actionCompletedDate"`
	}
	ImageSecurityEnabled bool     `json:"imageSecurityEnabled"`
	DisableAutoUpdate    bool     `json:"disableAutoUpdate"`
	Crn                  string   `json:"crn"`
	WorkerZones          []string `json:"workerZones"`
	// SupportedOperatingSystems => unknown type
	Lifecycle struct {
		MasterStatus             string `json:"masterStatus"`
		MasterStatusModifiedDate string `json:"masterStatusModifiedDate"`
		MasterHealth             string `json:"masterHealth"`
		MasterState              string `json:"masterState"`
		ModifiedDate             string `json:"modifiedDate"`
	} `json:"lifecycle"`
	ServiceEndpoints struct {
		PrivateServiceEndpointEnabled bool   `json:"privateServiceEndpointEnabled"`
		PrivateServiceEndpointURL     string `json:"privateServiceEndpointURL"`
		PublicServiceEndpointEnabled  bool   `json:"publicServiceEndpointEnabled"`
		PublicServiceEndpointURL      string `json:"publicServiceEndpointURL"`
	} `json:"serviceEndpoints"`
	Features struct {
		KeyProtectEnabled bool `json:"keyProtectEnabled"`
		PullSecretApplied bool `json:"pullSecretApplied"`
	} `json:"features"`
	Vpcs []string `json:"vpcs"`
}

// GetWorkerPoolsDetailResponse Newly created for manual unmarshal JSON response
type GetWorkerPoolsDetailResponse struct {
	Id       string `json:"id"`
	PoolName string `json:"poolName"`
	Flavor   string `json:"flavor"`
	// Taints => unknown type
	WorkerCount      int    `json:"workerCount"`
	Isolation        string `json:"isolation"`
	Provider         string `json:"provider"`
	IsBalanced       bool   `json:"isBalanced"`
	AutoscaleEnabled bool   `json:"autoscaleEnabled"`
	OpenshiftLicense string `json:"openshiftLicense"`
	Lifecycle        struct {
		DesiredState string `json:"desiredState"`
		ActualState  string `json:"actualState"`
	} `json:"lifecycle"`
	OperatingSystem string `json:"operatingSystem"`
	Zones           []struct {
		Id                 string `json:"id"`
		WorkerCount        int    `json:"workerCount"`
		AutobalanceEnabled bool   `json:"autobalanceEnabled"`
		// Messages => unknown type
		Subnets []struct {
			Id      string `json:"id"`
			Primary bool   `json:"primary"`
		} `json:"subnets"`
	}
	VpcID string `json:"vpcID"`
}

// GetKubeconfigOptions : The GetKubeconfig options.
// Modified XAuthRefreshToken to Authorization
type GetKubeconfigOptions struct {
	// Your IBM Cloud Identity and Access Management (IAM) token. To retrieve your IAM token, run ibmcloud iam oauth-tokens.
	Authorization *string `validate:"required"`

	// The name or ID of the cluster that you want to get the worker node details from. To list the clusters that you have
	// access to, use the `GET /v2/getClusters` API or run `ibmcloud ks cluster ls`.
	Cluster *string `validate:"required"`

	// The ID of the resource group that the cluster is in. To check the resource group ID of the cluster, use the `GET
	// /v2/getCluster` API.
	XAuthResourceGroup *string

	// Default format is json. Other options include yaml, and zip.
	Format *string

	// Retrieve the admin kubeconfig file.
	Admin *bool

	// Retrieve the Calico network config. Requires admin=true and format=zip.
	Network *bool

	// Allows users to set headers on API requests
	Headers map[string]string
}

type WorkerPoolAutoscalerConfig struct {
	Name    string `json:"name"`
	MinSize int    `json:"minSize"`
	MaxSize int    `json:"maxSize"`
	Enabled bool   `json:"enabled"`
}

// VPCCreateClusterWorkerPool : VPCCreateClusterWorkerPool is the vpc version of the worker pool part of a create cluster request.
type VPCCreateClusterWorkerPool struct {
	DiskEncryption *bool `json:"diskEncryption,omitempty"`

	Flavor *string `json:"flavor,omitempty"`

	Isolation *string `json:"isolation,omitempty"`

	KmsInstanceID *string `json:"kmsInstanceID,omitempty"`

	Labels map[string]string `json:"labels,omitempty"`

	Name *string `json:"name,omitempty"`

	VpcID *string `json:"vpcID,omitempty"`

	WorkerCount *int64 `json:"workerCount,omitempty"`

	WorkerVolumeCRKID *string `json:"workerVolumeCRKID,omitempty"`

	Zones []VPCCreateClusterWorkerPoolZone `json:"zones,omitempty"`

	OperatingSystem *string `json:"operatingSystem"`
}

// VpcCreateWorkerPoolOptions : The VpcCreateWorkerPool options.
type VpcCreateWorkerPoolOptions struct {
	Cluster *string

	DiskEncryption *bool

	Entitlement *string

	Flavor *string

	Isolation *string

	KmsInstanceID *string

	Labels map[string]string

	Name *string

	VpcID *string

	WorkerCount *int64

	WorkerVolumeCRKID *string

	Zones []Zone

	// Your IBM Cloud Identity and Access Management (IAM) token. To retrieve your IAM token, run ibmcloud iam oauth-tokens.
	Authorization *string `validate:"required"`

	// The ID of the resource group that the cluster is in. To check the resource group ID of the cluster, use the `GET
	// /v1/clusters/idOrName` API.
	XAuthResourceGroup *string

	// Allows users to set headers on API requests
	Headers map[string]string
}

// Zone : Zone describes a worker pool zone.
type Zone struct {
	AutobalanceEnabled *bool `json:"autobalanceEnabled,omitempty"`

	ID       *string `json:"id,omitempty"`
	SubnetID *string `json:"subnetID,omitempty"`
}
