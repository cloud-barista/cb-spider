// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2022.08.
// Updated by CB-Spider Team, 2026.10.

package resources

import "time"

// -------- Const
type ClusterStatus string

const (
	ClusterCreating ClusterStatus = "Creating"
	ClusterActive   ClusterStatus = "Active"
	ClusterInactive ClusterStatus = "Inactive"
	ClusterUpdating ClusterStatus = "Updating"
	ClusterDeleting ClusterStatus = "Deleting"
)

type NodeGroupStatus string

const (
	NodeGroupCreating NodeGroupStatus = "Creating"
	NodeGroupActive   NodeGroupStatus = "Active"
	NodeGroupInactive NodeGroupStatus = "Inactive"
	NodeGroupUpdating NodeGroupStatus = "Updating"
	NodeGroupDeleting NodeGroupStatus = "Deleting"
)

// -------- Info Structure
// ClusterInfo represents the details of a Kubernetes Cluster.
// @description Kubernetes Cluster Information
type ClusterInfo struct {
	IId IID `json:"IId" validate:"required"`

	Version string      `json:"Version" validate:"required" example:"1.30"` // Kubernetes Version, ex) 1.30
	Network NetworkInfo `json:"Network" validate:"required"`

	// ---
	NodeGroupList []NodeGroupInfo `json:"NodeGroupList" validate:"omitempty"`
	AccessInfo    AccessInfo      `json:"AccessInfo,omitempty"`
	Addons        AddonsInfo      `json:"Addons,omitempty"`

	Status ClusterStatus `json:"Status" validate:"required" example:"Active"`

	CreatedTime  time.Time  `json:"CreatedTime,omitempty" example:"2024-08-27T10:00:00Z"`
	TagList      []KeyValue `json:"TagList,omitempty" validate:"omitempty"`
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"`
}

// NetworkInfo represents the network configuration of a Cluster.
// @description Network Information for a Kubernetes Cluster
type NetworkInfo struct {
	VpcIID            IID   `json:"VpcIID" validate:"required"`
	SubnetIIDs        []IID `json:"SubnetIIDs" validate:"required"`
	SecurityGroupIIDs []IID `json:"SecurityGroupIIDs" validate:"required"`

	// ---
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"`
}

// NodeGroupInfo represents the configuration of a Node Group in a Cluster.
// @description Node Group Information for a Kubernetes Cluster
type NodeGroupInfo struct {
	IId IID `json:"IId" validate:"required"`

	// VM config.
	ImageIID     IID    `json:"ImageIID" validate:"required"`
	VMSpecName   string `json:"VMSpecName" validate:"required" example:"t3.medium"`
	RootDiskType string `json:"RootDiskType,omitempty" validate:"omitempty"`
	RootDiskSize string `json:"RootDiskSize,omitempty" validate:"omitempty" example:"50"` // in GB
	KeyPairIID   IID    `json:"KeyPairIID" validate:"required"`

	// Scaling config.
	OnAutoScaling   bool `json:"OnAutoScaling" validate:"required" example:"true"`
	DesiredNodeSize int  `json:"DesiredNodeSize" validate:"required" example:"2"`
	MinNodeSize     int  `json:"MinNodeSize" validate:"required" example:"1"`
	MaxNodeSize     int  `json:"MaxNodeSize" validate:"required" example:"3"`

	// ---
	Status NodeGroupStatus `json:"Status" validate:"required" example:"Active"`
	Nodes  []IID           `json:"Nodes,omitempty" validate:"omitempty"`

	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"`
}

// AccessInfo represents the access information of a Cluster.
// @description Access Information for a Kubernetes Cluster. <br> Take some time to provide.
type AccessInfo struct {
	Endpoint   string `json:"Endpoint,omitempty" example:"https://1.2.3.4"`
	Kubeconfig string `json:"Kubeconfig,omitempty" example:"apiVersion: v1\nclusters:\n- cluster:\n ...."`
}

// AddonsInfo represents the additional configuration information of a Cluster.
// @description Addons Information for a Kubernetes Cluster
type AddonsInfo struct {
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"`
}

// ClusterToken is a short-lived credential issued by the CSP for cluster access.
// It is a CSP domain value: the REST layer maps it into a Kubernetes ExecCredential
// (client.authentication.k8s.io/v1), so this type intentionally carries no
// Kubernetes-specific wire format.
// @description Short-lived Cluster Access Token
type ClusterToken struct {
	Token string `json:"Token" example:"k8s-aws-v1.aHR0cHM6Ly9zdHMuYXA..."`

	// ExpiresAt is when the token stops being accepted.
	// Zero means the CSP provides no expiry information; callers must treat it as unknown
	// rather than as "already expired" or "never expires".
	ExpiresAt time.Time `json:"ExpiresAt,omitempty" example:"2026-09-02T05:15:00Z"`
}

// -------- Storage Driver
// StorageDriverKind is the kind of storage a CSI driver provisions.
// Only Block is supported now.
type StorageDriverKind string

const (
	StorageDriverBlock StorageDriverKind = "Block" // block storage, ReadWriteOnce (e.g., EBS, PD, Azure Disk)
)

// StorageDriverManagementMode tells how the CSP manages a storage driver.
type StorageDriverManagementMode string

const (
	StorageDriverAddon        StorageDriverManagementMode = "Addon"        // can be enabled and disabled through the CSP API
	StorageDriverBuiltIn      StorageDriverManagementMode = "BuiltIn"      // always enabled; the CSP API cannot disable it
	StorageDriverNotSupported StorageDriverManagementMode = "NotSupported" // the CSP offers no driver of this kind
)

// StorageDriverStatus is the current state of a storage driver on a Cluster.
type StorageDriverStatus string

const (
	StorageDriverEnabled  StorageDriverStatus = "Enabled"
	StorageDriverDisabled StorageDriverStatus = "Disabled"
	StorageDriverUpdating StorageDriverStatus = "Updating"
	StorageDriverFailed   StorageDriverStatus = "Failed"
	StorageDriverUnknown  StorageDriverStatus = "Unknown"
)

// StorageDriverInfo represents a CSI storage driver that the CSP manages for a Cluster.
// Enabling installs the driver components and wires the permissions the driver needs;
// disabling removes the driver components only. Volumes and their data are never deleted,
// but existing PVs of the driver can no longer be attached or mounted.
// @description CSP-managed CSI Storage Driver Information for a Kubernetes Cluster
type StorageDriverInfo struct {
	Kind           StorageDriverKind           `json:"Kind" validate:"required" example:"Block"`
	CSPDriverName  string                      `json:"CSPDriverName,omitempty" example:"aws-ebs-csi-driver"` // CSP add-on name as-is
	CSIDriverName  string                      `json:"CSIDriverName,omitempty" example:"ebs.csi.aws.com"`    // name of the Kubernetes CSIDriver object
	ManagementMode StorageDriverManagementMode `json:"ManagementMode" validate:"required" example:"Addon"`
	Status         StorageDriverStatus         `json:"Status" validate:"required" example:"Enabled"`
	Version        string                      `json:"Version,omitempty" example:"v1.37.0-eksbuild.1"`
	KeyValueList   []KeyValue                  `json:"KeyValueList,omitempty" validate:"omitempty"`
}

// -------- Cluster API
type ClusterHandler interface {

	//------ Cluster Management
	ListIID() ([]*IID, error)
	CreateCluster(clusterReqInfo ClusterInfo) (ClusterInfo, error)
	ListCluster() ([]*ClusterInfo, error)
	GetCluster(clusterIID IID) (ClusterInfo, error)
	DeleteCluster(clusterIID IID) (bool, error)

	//------ Token Management
	GenerateClusterToken(clusterIID IID) (ClusterToken, error)

	//------ NodeGroup Management
	AddNodeGroup(clusterIID IID, nodeGroupReqInfo NodeGroupInfo) (NodeGroupInfo, error)
	SetNodeGroupAutoScaling(clusterIID IID, nodeGroupIID IID, on bool) (bool, error)
	ChangeNodeGroupScaling(clusterIID IID, nodeGroupIID IID,
		DesiredNodeSize int, MinNodeSize int, MaxNodeSize int) (NodeGroupInfo, error)
	RemoveNodeGroup(clusterIID IID, nodeGroupIID IID) (bool, error)

	//------ Upgrade K8S
	UpgradeCluster(clusterIID IID, newVersion string) (ClusterInfo, error)

	//------ Storage Driver Management
	// ListStorageDriver returns one entry per supported StorageDriverKind (Block).
	ListStorageDriver(clusterIID IID) ([]StorageDriverInfo, error)
	// EnableStorageDriver is idempotent. An empty version means the CSP default version.
	// Drivers whose CSP applies the change asynchronously return Status Updating.
	EnableStorageDriver(clusterIID IID, kind StorageDriverKind, version string) (StorageDriverInfo, error)
	// DisableStorageDriver removes the driver components and keeps volumes and cloud resources.
	DisableStorageDriver(clusterIID IID, kind StorageDriverKind) (bool, error)
}
