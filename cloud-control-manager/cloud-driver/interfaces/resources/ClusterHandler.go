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


//-------- Const
type ClusterStatus string

const (
        ClusterCreating ClusterStatus = "Creating"
        ClusterActive   ClusterStatus = "Active"
        ClusterInactive   ClusterStatus = "Inactive"
        ClusterUpdating   ClusterStatus = "Updating"
        ClusterDeleting   ClusterStatus = "Deleting"
)

//-------- Info Structure
type ClusterInfo struct {
	IId		IID 	// {NameId, SystemId}

	Version		string	// Kubernetes Version, ex) 1.23.3

	Network		NetworkInfo
	NodeGroupList	[]NodeGroupInfo	
	Addons		AddonsInfo

	
        Status 		ClusterStatus

	CreatedTime	time.Time
	KeyValueList []KeyValue
}

type NetworkInfo struct {
	VpcIID		IID	// {NameId, SystemId}
	SubnetIID	[]IID	
        SecurityGroupIIDs []IID

	KeyValueList []KeyValue
}

type NodeGroupInfo struct {
	IId		IID 	// {NameId, SystemId}

	// VM config.
	ImageIID	IID
        VMSpecName 	string
        RootDiskType    string  // "SSD(gp2)", "Premium SSD", ...
	RootDiskSize 	string  // "", "default", "50", "1000" (GB)
        KeyPairIID 	IID

	// Auto Scaling config.
	AutoScaling		bool
	MinNumberNodes		int
	MaxNumberNodes		int

	DesiredNumberNodes	int

	NodeList	[]IID
	KeyValueList []KeyValue
}

// CNI, DNS, .... @todo
type AddonsInfo struct {
        KeyValueList []KeyValue
}


//-------- Cluster API
type ClusterHandler interface {

	//------ Cluster Management
	CreateCluster(clusterReqInfo ClusterInfo) (ClusterInfo, error)
	ListCluster() ([]*ClusterInfo, error)
	GetCluster(clusterIID IID) (ClusterInfo, error)
	DeleteCluster(clusterIID IID) (bool, error)

	AddNodeGroup(clusterIID IID, nodeGroup IID) (ClusterInfo, error)
	RemoveNodeGroup(clusterIID IID, nodeGroup IID) (bool, error)

	//------ Upgrade K8S
	UpgradeCluster(clusterIID IID, newVersion string) (ClusterInfo, error)

}

//-------- NodeGroup API
type NodeGroupHandler interface {

        //------ NodeGroup Management
        CreateNodeGroup(nodeGroupReqInfo NodeGroupInfo) (NodeGroupInfo, error)
        ListNodeGroup() ([]*NodeGroupInfo, error)
        GetNodeGroup(nodeGroupIID IID) (NodeGroupInfo, error)
        DeleteNodeGroup(nodeGroupIID IID) (bool, error)

        AddNodes(nodeGroupIID IID, number int) (NodeGroupInfo, error)
        RemoveNodes(nodeGroupIID IID, vmIIDs *[]IID) (bool, error)
}
