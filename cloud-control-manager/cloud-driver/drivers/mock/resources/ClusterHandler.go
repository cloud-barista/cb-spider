// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Mock Driver.
//
// by CB-Spider Team, 2024.07.

package resources

import (
	"fmt"
	"sync"
	"time"

	cblog "github.com/cloud-barista/cb-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

var clusterInfoMap map[string][]*irs.ClusterInfo

type MockClusterHandler struct {
	MockName string
}

func init() {
	// cblog is a global variable.
	clusterInfoMap = make(map[string][]*irs.ClusterInfo)
}

var clusterMapLock = new(sync.RWMutex)

// (1) create clusterInfo object
// (2) insert clusterInfo into global Map
func (clusterHandler *MockClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (irs.ClusterInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called CreateCluster()!")

	mockName := clusterHandler.MockName
	clusterReqInfo.IId.SystemId = clusterReqInfo.IId.NameId

	// Set SystemID for VPC, Subnets, and SecurityGroups
	clusterReqInfo.Network.VpcIID.SystemId = clusterReqInfo.Network.VpcIID.NameId
	for i, subnet := range clusterReqInfo.Network.SubnetIIDs {
		subnet.SystemId = subnet.NameId
		clusterReqInfo.Network.SubnetIIDs[i] = subnet
	}
	for i, sg := range clusterReqInfo.Network.SecurityGroupIIDs {
		sg.SystemId = sg.NameId
		clusterReqInfo.Network.SecurityGroupIIDs[i] = sg
	}

	// Set SystemID for NodeGroups
	for i, nodeGroup := range clusterReqInfo.NodeGroupList {
		nodeGroup.IId.SystemId = nodeGroup.IId.NameId
		nodeGroup.ImageIID.SystemId = nodeGroup.ImageIID.NameId
		nodeGroup.KeyPairIID.SystemId = nodeGroup.KeyPairIID.NameId
		nodeGroup.Status = irs.NodeGroupActive
		for j, node := range nodeGroup.Nodes {
			node.SystemId = node.NameId
			nodeGroup.Nodes[j] = node
		}
		clusterReqInfo.NodeGroupList[i] = nodeGroup
	}

	// Set initial status and created time
	clusterReqInfo.Status = irs.ClusterCreating
	clusterReqInfo.CreatedTime = time.Now()

	// (2) insert ClusterInfo into global Map
	clusterMapLock.Lock()
	defer clusterMapLock.Unlock()
	infoList, _ := clusterInfoMap[mockName]
	infoList = append(infoList, &clusterReqInfo)
	clusterInfoMap[mockName] = infoList

	clusterReqInfo.Status = irs.ClusterActive
	return CloneClusterInfo(clusterReqInfo), nil
}

func CloneClusterInfoList(srcInfoList []*irs.ClusterInfo) []*irs.ClusterInfo {
	clonedInfoList := []*irs.ClusterInfo{}
	for _, srcInfo := range srcInfoList {
		clonedInfo := CloneClusterInfo(*srcInfo)
		clonedInfoList = append(clonedInfoList, &clonedInfo)
	}
	return clonedInfoList
}

func CloneClusterInfo(srcInfo irs.ClusterInfo) irs.ClusterInfo {
	/*
		type ClusterInfo struct {
			IId IID // {NameId, SystemId}
			Version string // Kubernetes Version, ex) 1.23.3
			Network NetworkInfo
			NodeGroupList []NodeGroupInfo
			AccessInfo    AccessInfo
			Addons        AddonsInfo
			Status        ClusterStatus
			CreatedTime   time.Time
			KeyValueList  []KeyValue
		}
	*/

	// clone ClusterInfo
	clonedInfo := irs.ClusterInfo{
		IId:           irs.IID{srcInfo.IId.NameId, srcInfo.IId.SystemId},
		Version:       srcInfo.Version,
		Network:       srcInfo.Network,
		NodeGroupList: CloneNodeGroupInfoList(srcInfo.NodeGroupList),
		AccessInfo:    srcInfo.AccessInfo,
		Addons:        srcInfo.Addons,
		Status:        srcInfo.Status,
		CreatedTime:   srcInfo.CreatedTime,
		TagList:       srcInfo.TagList,
		KeyValueList:  srcInfo.KeyValueList,
	}

	return clonedInfo
}

func CloneNodeGroupInfoList(srcInfoList []irs.NodeGroupInfo) []irs.NodeGroupInfo {
	clonedInfoList := []irs.NodeGroupInfo{}
	for _, srcInfo := range srcInfoList {
		clonedInfo := CloneNodeGroupInfo(srcInfo)
		clonedInfoList = append(clonedInfoList, clonedInfo)
	}
	return clonedInfoList
}

func CloneNodeGroupInfo(srcInfo irs.NodeGroupInfo) irs.NodeGroupInfo {
	/*
		type NodeGroupInfo struct {
			IId IID // {NameId, SystemId}
			ImageIID     IID
			VMSpecName   string
			RootDiskType string // "SSD(gp2)", "Premium SSD", ...
			RootDiskSize string // "", "default", "50", "1000" (GB)
			KeyPairIID   IID
			OnAutoScaling   bool
			DesiredNodeSize int
			MinNodeSize     int
			MaxNodeSize     int
			Status NodeGroupStatus
			Nodes  []IID
			KeyValueList []KeyValue
		}
	*/

	// clone NodeGroupInfo
	clonedInfo := irs.NodeGroupInfo{
		IId:             irs.IID{srcInfo.IId.NameId, srcInfo.IId.SystemId},
		ImageIID:        irs.IID{srcInfo.ImageIID.NameId, srcInfo.ImageIID.SystemId},
		VMSpecName:      srcInfo.VMSpecName,
		RootDiskType:    srcInfo.RootDiskType,
		RootDiskSize:    srcInfo.RootDiskSize,
		KeyPairIID:      irs.IID{srcInfo.KeyPairIID.NameId, srcInfo.KeyPairIID.SystemId},
		OnAutoScaling:   srcInfo.OnAutoScaling,
		DesiredNodeSize: srcInfo.DesiredNodeSize,
		MinNodeSize:     srcInfo.MinNodeSize,
		MaxNodeSize:     srcInfo.MaxNodeSize,
		Status:          srcInfo.Status,
		Nodes:           cloneIIDArray(srcInfo.Nodes),
		KeyValueList:    srcInfo.KeyValueList,
	}

	return clonedInfo
}

func (clusterHandler *MockClusterHandler) ListCluster() ([]*irs.ClusterInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListCluster()!")

	mockName := clusterHandler.MockName
	clusterMapLock.RLock()
	defer clusterMapLock.RUnlock()
	infoList, ok := clusterInfoMap[mockName]
	if !ok {
		return []*irs.ClusterInfo{}, nil
	}

	// cloning list of Cluster
	return CloneClusterInfoList(infoList), nil
}

func (clusterHandler *MockClusterHandler) GetCluster(iid irs.IID) (irs.ClusterInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called GetCluster()!")

	clusterMapLock.RLock()
	defer clusterMapLock.RUnlock()

	mockName := clusterHandler.MockName
	infoList, ok := clusterInfoMap[mockName]
	if !ok {
		return irs.ClusterInfo{}, fmt.Errorf("%s Cluster does not exist!!", iid.NameId)
	}

	for _, info := range infoList {
		if info.IId.NameId == iid.NameId {
			return CloneClusterInfo(*info), nil
		}
	}

	return irs.ClusterInfo{}, fmt.Errorf("%s Cluster does not exist!!", iid.NameId)
}

func (clusterHandler *MockClusterHandler) DeleteCluster(iid irs.IID) (bool, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called DeleteCluster()!")

	clusterMapLock.Lock()
	defer clusterMapLock.Unlock()

	mockName := clusterHandler.MockName
	infoList, ok := clusterInfoMap[mockName]
	if !ok {
		return false, fmt.Errorf("%s Cluster does not exist!!", iid.NameId)
	}

	for idx, info := range infoList {
		if info.IId.SystemId == iid.SystemId {
			infoList = append(infoList[:idx], infoList[idx+1:]...)
			clusterInfoMap[mockName] = infoList
			return true, nil
		}
	}
	return false, nil
}

func (clusterHandler *MockClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (irs.NodeGroupInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called AddNodeGroup()!")

	clusterMapLock.Lock()
	defer clusterMapLock.Unlock()

	mockName := clusterHandler.MockName
	infoList, ok := clusterInfoMap[mockName]
	if !ok {
		return irs.NodeGroupInfo{}, fmt.Errorf("%s Cluster does not exist!!", clusterIID.NameId)
	}

	nodeGroupReqInfo.IId.SystemId = nodeGroupReqInfo.IId.NameId
	nodeGroupReqInfo.Status = irs.NodeGroupActive

	for _, info := range infoList {
		if info.IId.NameId == clusterIID.NameId {
			info.NodeGroupList = append(info.NodeGroupList, nodeGroupReqInfo)
			return CloneNodeGroupInfo(nodeGroupReqInfo), nil
		}
	}

	return irs.NodeGroupInfo{}, fmt.Errorf("%s Cluster does not exist!!", clusterIID.NameId)
}

func (clusterHandler *MockClusterHandler) SetNodeGroupAutoScaling(clusterIID irs.IID, nodeGroupIID irs.IID, on bool) (bool, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called SetNodeGroupAutoScaling()!")

	clusterMapLock.Lock()
	defer clusterMapLock.Unlock()

	mockName := clusterHandler.MockName
	infoList, ok := clusterInfoMap[mockName]
	if !ok {
		return false, fmt.Errorf("%s Cluster does not exist!!", clusterIID.NameId)
	}

	for _, info := range infoList {
		if info.IId.NameId == clusterIID.NameId {
			for idx, ng := range info.NodeGroupList {
				if ng.IId.NameId == nodeGroupIID.NameId {
					info.NodeGroupList[idx].OnAutoScaling = on
					return true, nil
				}
			}
		}
	}

	return false, fmt.Errorf("%s NodeGroup does not exist!!", nodeGroupIID.NameId)
}

func (clusterHandler *MockClusterHandler) ChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID, DesiredNodeSize int, MinNodeSize int, MaxNodeSize int) (irs.NodeGroupInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ChangeNodeGroupScaling()!")

	clusterMapLock.Lock()
	defer clusterMapLock.Unlock()

	mockName := clusterHandler.MockName
	infoList, ok := clusterInfoMap[mockName]
	if !ok {
		return irs.NodeGroupInfo{}, fmt.Errorf("%s Cluster does not exist!!", clusterIID.NameId)
	}

	for _, info := range infoList {
		if info.IId.NameId == clusterIID.NameId {
			for idx, ng := range info.NodeGroupList {
				if ng.IId.NameId == nodeGroupIID.NameId {
					info.NodeGroupList[idx].DesiredNodeSize = DesiredNodeSize
					info.NodeGroupList[idx].MinNodeSize = MinNodeSize
					info.NodeGroupList[idx].MaxNodeSize = MaxNodeSize
					return CloneNodeGroupInfo(info.NodeGroupList[idx]), nil
				}
			}
		}
	}

	return irs.NodeGroupInfo{}, fmt.Errorf("%s NodeGroup does not exist!!", nodeGroupIID.NameId)
}

func (clusterHandler *MockClusterHandler) RemoveNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (bool, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called RemoveNodeGroup()!")

	clusterMapLock.Lock()
	defer clusterMapLock.Unlock()

	mockName := clusterHandler.MockName
	infoList, ok := clusterInfoMap[mockName]
	if !ok {
		return false, fmt.Errorf("%s Cluster does not exist!!", clusterIID.NameId)
	}

	for _, info := range infoList {
		if info.IId.NameId == clusterIID.NameId {
			for idx, ng := range info.NodeGroupList {
				if ng.IId.NameId == nodeGroupIID.NameId {
					info.NodeGroupList = append(info.NodeGroupList[:idx], info.NodeGroupList[idx+1:]...)
					return true, nil
				}
			}
		}
	}

	return false, fmt.Errorf("%s NodeGroup does not exist!!", nodeGroupIID.NameId)
}

func (clusterHandler *MockClusterHandler) UpgradeCluster(clusterIID irs.IID, newVersion string) (irs.ClusterInfo, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called UpgradeCluster()!")

	clusterMapLock.Lock()
	defer clusterMapLock.Unlock()

	mockName := clusterHandler.MockName
	infoList, ok := clusterInfoMap[mockName]
	if !ok {
		return irs.ClusterInfo{}, fmt.Errorf("%s Cluster does not exist!!", clusterIID.NameId)
	}

	for _, info := range infoList {
		if info.IId.NameId == clusterIID.NameId {
			info.Status = irs.ClusterUpdating
			time.Sleep(2 * time.Second) // Simulate upgrade time
			info.Version = newVersion
			info.Status = irs.ClusterActive
			return CloneClusterInfo(*info), nil
		}
	}

	return irs.ClusterInfo{}, fmt.Errorf("%s Cluster does not exist!!", clusterIID.NameId)
}

func (ClusterHandler *MockClusterHandler) ListIID() ([]*irs.IID, error) {
	cblogger := cblog.GetLogger("CB-SPIDER")
	cblogger.Info("Mock Driver: called ListIID()!")

	clusterMapLock.RLock()
	defer clusterMapLock.RUnlock()

	mockName := ClusterHandler.MockName
	infoList, ok := clusterInfoMap[mockName]
	if !ok {
		return []*irs.IID{}, nil
	}

	iidList := make([]*irs.IID, len(infoList))
	for i, info := range infoList {
		iidList[i] = &info.IId
	}
	return iidList, nil
}
