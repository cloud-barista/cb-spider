// Alibaba Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Alibaba Driver.
//
// by CB-Spider Team, 2022.09.

package pmks

import (
	"os"
	"testing"

	adrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/alibaba"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

func getClusterHandler() (irs.ClusterHandler, error) {

	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			ClientId:     os.Getenv("CLIENT_ID"),
			ClientSecret: os.Getenv("CLIENT_SECRET"),
		},
		RegionInfo: idrv.RegionInfo{
			Region: os.Getenv("REGION_ID"),
		},
	}

	cloudDriver := new(adrv.AlibabaDriver)
	cloudConnection, err := cloudDriver.ConnectCloud(connectionInfo)
	if err != nil {
		return nil, err
	}

	clusterHandler, err := cloudConnection.CreateClusterHandler()
	if err != nil {
		return nil, err
	}

	return clusterHandler, nil
}

// K8S 버전 변경
// 기존에 정상적으로 생되던 K8S 버전이 지금은 지원안됨
// K8S 버전을 지원되는 버전으로 변경
//
//	"message":"The specified KubernetesVersion 1.22.10-aliyun.1 is invalid,
//	allowd values are [1.24.6-aliyun.1 1.22.15-aliyun.1]
func TestCreateClusterOnly(t *testing.T) {

	t.Log("클러스터 생성, 노드그룹은 생성안함")

	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	clusterInfo := irs.ClusterInfo{
		IId: irs.IID{
			NameId:   "cluster-1",
			SystemId: "",
		},
		Version: "1.22.15-aliyun.1",
		Network: irs.NetworkInfo{
			VpcIID:            irs.IID{NameId: "", SystemId: "vpc-6wegylv6bnfsrxfyli7ni"},
			SecurityGroupIIDs: []irs.IID{{NameId: "", SystemId: "sg-6we5h09p1u380n7or9hc"}},
		},
	}

	cluster_, err := clusterHandler.CreateCluster(clusterInfo)
	if err != nil {
		t.Error(err)
	}

	t.Log(cluster_)
}

func TestCreateClusterOnlyAtTokyo(t *testing.T) {

	t.Log("클러스터 생성, 노드그룹은 생성안함")

	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	clusterInfo := irs.ClusterInfo{
		IId: irs.IID{
			NameId:   "cluster-tokyo",
			SystemId: "",
		},
		Version: "1.22.15-aliyun.1",
		Network: irs.NetworkInfo{
			VpcIID:            irs.IID{NameId: "", SystemId: "vpc-6wegylv6bnfsrxfyli7ni"},
			SecurityGroupIIDs: []irs.IID{{NameId: "", SystemId: "sg-6we5h09p1u380n7or9hc"}},
		},
	}

	cluster_, err := clusterHandler.CreateCluster(clusterInfo)
	if err != nil {
		t.Error(err)
	}

	t.Log(cluster_)
}

func TestCreateClusterWith1NodeGroup(t *testing.T) {
	//
	t.Log("클러스터 + 노드그룹 생성")

	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	clusterInfo := irs.ClusterInfo{
		IId: irs.IID{
			NameId:   "cluster-x",
			SystemId: "",
		},
		Version: "1.22.15-aliyun.1",
		Network: irs.NetworkInfo{
			VpcIID:            irs.IID{NameId: "", SystemId: "vpc-2zek5slojo5bh621ftnrg"},
			SubnetIIDs:        []irs.IID{},
			SecurityGroupIIDs: []irs.IID{},
			KeyValueList:      []irs.KeyValue{},
		},
		NodeGroupList: []irs.NodeGroupInfo{
			{
				IId: irs.IID{NameId: "nodepoolx101", SystemId: ""},
				// ImageIID:        irs.IID{NameId: "", SystemId: "ubuntu_20_04_x64_20G_alibase_20220824.vhd"}, //옵션, Not Working
				VMSpecName:      "ecs.c6.xlarge",
				RootDiskType:    "cloud_essd",
				RootDiskSize:    "70",
				KeyPairIID:      irs.IID{NameId: "kp1", SystemId: ""},
				OnAutoScaling:   true,
				DesiredNodeSize: 1,
				MinNodeSize:     0,
				MaxNodeSize:     3,
			},
		},
	}

	cluster_, err := clusterHandler.CreateCluster(clusterInfo)
	if err != nil {
		t.Error(err)
	}

	t.Log(cluster_)
}

func TestListCluster(t *testing.T) {

	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	clusters, err := clusterHandler.ListCluster()
	if err != nil {
		t.Error(err)
	}

	for _, cluster := range clusters {
		t.Log(cluster.IId.SystemId)
		println(cluster.IId.NameId, cluster.Status)
	}
}

func TestGetCluster(t *testing.T) {

	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	clusters, err := clusterHandler.ListCluster()
	if err != nil {
		t.Error(err)
	}

	t.Log(clusters)

	for _, cluster := range clusters {
		cluster_, err := clusterHandler.GetCluster(cluster.IId)
		if err != nil {
			println(err.Error())
		}
		t.Log(cluster_)
	}
}

// 마지막 테스트로 이동
// func TestDeleteCluster(t *testing.T) {
// }

func TestAddNodeGroup(t *testing.T) {

	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	new_node_group := &irs.NodeGroupInfo{
		IId: irs.IID{NameId: "nodepool-x", SystemId: ""},
		// ImageIID:        irs.IID{NameId: "", SystemId: "ubuntu_20_04_x64_20G_alibase_20220824.vhd"}, // 옵션, 설정해도 안됨
		VMSpecName:      "ecs.c6.xlarge",
		RootDiskType:    "cloud_essd",
		RootDiskSize:    "70",
		KeyPairIID:      irs.IID{NameId: "kp1", SystemId: ""},
		OnAutoScaling:   true,
		DesiredNodeSize: 0, // not supported.
		MinNodeSize:     3,
		MaxNodeSize:     3,
	}

	clusters, _ := clusterHandler.ListCluster()
	for _, cluster := range clusters {
		t.Log(cluster)
		node_group, err := clusterHandler.AddNodeGroup(cluster.IId, *new_node_group)
		if err != nil {
			t.Error(err)
		}
		t.Log(node_group)
	}
}

func TestAddNodeGroupTokyo(t *testing.T) {

	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	new_node_group := &irs.NodeGroupInfo{
		IId: irs.IID{NameId: "nodepool-x2", SystemId: ""},
		// ImageIID:        irs.IID{NameId: "", SystemId: "ubuntu_20_04_x64_20G_alibase_20220824.vhd"}, // 옵션, 설정해도 안됨
		VMSpecName:      "ecs.c6.xlarge",
		RootDiskType:    "cloud_essd",
		RootDiskSize:    "70",
		KeyPairIID:      irs.IID{NameId: "kp1", SystemId: ""},
		OnAutoScaling:   true,
		DesiredNodeSize: 2,
		MinNodeSize:     1,
		MaxNodeSize:     3,
	}

	clusters, _ := clusterHandler.ListCluster()
	for _, cluster := range clusters {
		t.Log(cluster)
		node_group, err := clusterHandler.AddNodeGroup(cluster.IId, *new_node_group)
		if err != nil {
			t.Error(err)
		}
		t.Log(node_group)
	}
}

// func TestListNodeGroup(t *testing.T) {
// 	clusterHandler, err := getClusterHandler()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	clusters, _ := clusterHandler.ListCluster()
// 	for _, cluster := range clusters {
// 		node_groups, _ := clusterHandler.ListNodeGroup(cluster.IId)
// 		for _, node_group := range node_groups {
// 			t.Log(node_group.IId.NameId, node_group.IId.SystemId)
// 			t.Log(node_group)
// 		}
// 	}
// }

// func TestGetNodeGroup(t *testing.T) {
// 	clusterHandler, err := getClusterHandler()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	clusters, _ := clusterHandler.ListCluster()
// 	for _, cluster := range clusters {
// 		node_groups, _ := clusterHandler.ListNodeGroup(cluster.IId)
// 		for _, node_group := range node_groups {
// 			node_group_, err := clusterHandler.GetNodeGroup(cluster.IId, node_group.IId)
// 			if err != nil {
// 				t.Error(err)
// 			}
// 			t.Log(node_group_.IId.NameId, node_group_.IId.SystemId)
// 			t.Log(node_group_)
// 		}
// 	}

// 	node_group, err := clusterHandler.GetNodeGroup(irs.IID{NameId: "", SystemId: "cluster_id_not_exist"}, irs.IID{NameId: "", SystemId: "node_group_id_not_exist"})
// 	if err != nil {
// 		println(err.Error())
// 	}
// 	println(node_group.IId.NameId)
// }

func TestSetNodeGroupAutoScalingOn(t *testing.T) {
	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	clusters, _ := clusterHandler.ListCluster()
	for _, cluster := range clusters {
		// node_groups, _ := clusterHandler.ListNodeGroup(cluster.IId)
		// for _, node_group := range node_groups {
		for _, node_group_info := range cluster.NodeGroupList {
			res, err := clusterHandler.SetNodeGroupAutoScaling(cluster.IId, node_group_info.IId, true)
			if err != nil {
				t.Error(err)
			}
			println(res)
		}
	}
}

func TestSetNodeGroupAutoScalingOff(t *testing.T) {
	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	clusters, _ := clusterHandler.ListCluster()
	for _, cluster := range clusters {
		// node_groups, _ := clusterHandler.ListNodeGroup(cluster.IId)
		// for _, node_group := range node_groups {
		for _, node_group_info := range cluster.NodeGroupList {
			res, err := clusterHandler.SetNodeGroupAutoScaling(cluster.IId, node_group_info.IId, false)
			if err != nil {
				t.Error(err)
			}
			println(res)
		}
	}
}

func TestChangeNodeGroupScaling(t *testing.T) {
	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	clusters, _ := clusterHandler.ListCluster()
	for _, cluster := range clusters {
		for _, node_group_info := range cluster.NodeGroupList {
			res, err := clusterHandler.ChangeNodeGroupScaling(cluster.IId, node_group_info.IId, 0, 1, 2)
			if err != nil {
				t.Error(err)
			}
			println(res.IId.NameId, res.IId.SystemId)
		}
	}
}

func TestRemoveNodeGroup(t *testing.T) {
	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	clusters, _ := clusterHandler.ListCluster()
	for _, cluster := range clusters {
		for _, node_group_info := range cluster.NodeGroupList {
			res, _ := clusterHandler.RemoveNodeGroup(cluster.IId, node_group_info.IId)
			if err != nil {
				t.Error(err)
			}
			if res == false {
				t.Error("Failed to remove node group")
			}
		}
	}
}

func TestUpgradeCluster(t *testing.T) {
	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	clusters, _ := clusterHandler.ListCluster()
	for _, cluster := range clusters {
		res, err := clusterHandler.UpgradeCluster(cluster.IId, "1.22.3-aliyun.1")
		if err != nil {
			t.Error(err)
		}
		t.Log(res)
	}
}

func TestDeleteCluster(t *testing.T) {

	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	clusters, _ := clusterHandler.ListCluster()
	for _, cluster := range clusters {
		println(cluster.IId.NameId, cluster.IId.SystemId)
		result, err := clusterHandler.DeleteCluster(cluster.IId)
		if err != nil {
			t.Error(err)
		}
		t.Log(result)
	}
}
