// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2022.08.

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
type ClusterInfo struct {
	IId IID // {NameId, SystemId}

	Version string // Kubernetes Version, ex) 1.23.3
	Network       NetworkInfo

	// ---

	NodeGroupList []NodeGroupInfo
	AccessInfo    AccessInfo
	Addons        AddonsInfo

	Status        ClusterStatus

	CreatedTime  time.Time
	KeyValueList []KeyValue
}

type NetworkInfo struct {
	VpcIID            IID // {NameId, SystemId}
	SubnetIIDs        []IID
	SecurityGroupIIDs []IID

	// ---

	KeyValueList []KeyValue
}

type NodeGroupInfo struct {
	IId IID // {NameId, SystemId}

	// VM config.
	ImageIID     IID
	VMSpecName   string
	RootDiskType string // "SSD(gp2)", "Premium SSD", ...
	RootDiskSize string // "", "default", "50", "1000" (GB)
	KeyPairIID   IID

	// Scaling config.
	OnAutoScaling   bool // default: true
	DesiredNodeSize int
	MinNodeSize     int
	MaxNodeSize     int

	// ---

	Status       NodeGroupStatus
	Nodes        []IID

	KeyValueList []KeyValue
}

type AccessInfo struct {
	Endpoint 	string // ex) https://1.2.3.4:6443
	Kubeconfig	string
}

// CNI, DNS, .... @todo
type AddonsInfo struct {
	KeyValueList []KeyValue
}

// -------- Cluster API
type ClusterHandler interface {

	//------ Cluster Management
	CreateCluster(clusterReqInfo ClusterInfo) (ClusterInfo, error)
	ListCluster() ([]*ClusterInfo, error)
	GetCluster(clusterIID IID) (ClusterInfo, error)
	DeleteCluster(clusterIID IID) (bool, error)

	//------ NodeGroup Management
	AddNodeGroup(clusterIID IID, nodeGroupReqInfo NodeGroupInfo) (NodeGroupInfo, error)
	SetNodeGroupAutoScaling(clusterIID IID, nodeGroupIID IID, on bool) (bool, error)
	ChangeNodeGroupScaling(clusterIID IID, nodeGroupIID IID,
		DesiredNodeSize int, MinNodeSize int, MaxNodeSize int) (NodeGroupInfo, error)
	RemoveNodeGroup(clusterIID IID, nodeGroupIID IID) (bool, error)

	//------ Upgrade K8S
	UpgradeCluster(clusterIID IID, newVersion string) (ClusterInfo, error)
}
