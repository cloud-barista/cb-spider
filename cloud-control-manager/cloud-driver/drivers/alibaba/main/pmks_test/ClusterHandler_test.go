package pmks

import (
	"os"
	"testing"

	_ "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/alibaba/main/pmks_test/env" // 위치 변경 하면 안됨. 환경설정 정보 읽기 전에 테스트 수행됨

	adrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/alibaba"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

func getClusterHandler() (irs.ClusterHandler, error) {

	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			AccessKey:    os.Getenv("ACCESS_KEY"),
			AccessSecret: os.Getenv("ACCESS_SECRET"),
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

func TestCreateCluster(t *testing.T) {

	t.Log("TestCreateCluster()")

	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	clusterInfo := irs.ClusterInfo{
		IId: irs.IID{
			NameId:   "cluster-x",
			SystemId: "",
		},
		Version: "v1.23.3",
		Network: irs.NetworkInfo{
			VpcIID: irs.IID{
				NameId:   "",
				SystemId: "8a07c257-72ba-4183-bb77-fa46f6d12c39",
			},
			SubnetIID: []irs.IID{
				{
					NameId:   "",
					SystemId: "e4ba747b-d9f5-45a2-b749-45567054bbe5",
				},
			},
			// SecurityGroupIIDs: []irs.IID{
			// 	{
			// 		NameId:   "",
			// 		SystemId: "",
			// 	},
			// },
		},
		KeyValueList: []irs.KeyValue{
			{
				Key:   "external_network_id",
				Value: "a858742a-245b-41d3-9a05-617e1b069eb9",
			},
			{
				Key:   "external_subnet_id_list",
				Value: "0641f8ac-c7e9-43a8-9eb5-ba63d08b83e0",
			},
		},
		NodeGroupList: []irs.NodeGroupInfo{
			{
				IId: irs.IID{
					NameId:   "default-nodegroup",
					SystemId: "",
				},
				ImageIID: irs.IID{
					NameId:   "ubuntu 18.04",
					SystemId: "2717ec03-3a4d-4728-b372-183065facdba",
				},
				VMSpecName:   "13646526-0bb9-400b-929f-797fdb7547eb", // flavor_id,
				RootDiskType: "General HDD",
				RootDiskSize: "20", // root_volume_size
				KeyPairIID: irs.IID{
					NameId:   "kp1",
					SystemId: "",
				},
				OnAutoScaling:   true,
				MinNodeSize:     1,
				MaxNodeSize:     3,
				DesiredNodeSize: 1, // node_count

				KeyValueList: []irs.KeyValue{
					{
						Key:   "availability_zone",
						Value: "kr-pub-a",
					},
				},
			},
		},
	}

	cluster, err := clusterHandler.CreateCluster(clusterInfo)
	if err != nil {
		t.Error(err)
	}

	t.Log(cluster)
}

// func TestListCluster(t *testing.T) {

// 	clusterHandler, err := getClusterHandler()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	clusters, err := clusterHandler.ListCluster()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	if len(clusters) == 0 {
// 		t.Error("No cluster found")
// 	}

// 	for _, cluster := range clusters {
// 		t.Log(cluster)
// 	}
// }

// func TestGetCluster(t *testing.T) {

// 	clusterHandler, err := getClusterHandler()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	clusters, err := clusterHandler.ListCluster()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	if len(clusters) == 0 {
// 		t.Error("No cluster found")
// 	}

// 	t.Log(clusters)

// 	for _, cluster := range clusters {
// 		cluster_, err := clusterHandler.GetCluster(cluster.IId)
// 		if err != nil {
// 			println(err.Error())
// 		}
// 		t.Log(cluster_)
// 	}
// }

// // func TestDeleteCluster(t *testing.T) {
// // }

// func TestAddNodeGroup(t *testing.T) {
// 	clusterHandler, err := getClusterHandler()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	clusters, _ := clusterHandler.ListCluster()
// 	for _, cluster := range clusters {
// 		new_node_group := irs.NodeGroupInfo{
// 			IId:             irs.IID{NameId: "added-nodegroup", SystemId: ""},
// 			ImageIID:        irs.IID{NameId: "", SystemId: "2717ec03-3a4d-4728-b372-183065facdba"},
// 			VMSpecName:      "13646526-0bb9-400b-929f-797fdb7547eb",
// 			RootDiskType:    "General HDD",
// 			RootDiskSize:    "20",
// 			KeyPairIID:      irs.IID{NameId: "kp1", SystemId: ""},
// 			OnAutoScaling:   true,
// 			MinNodeSize:     1,
// 			MaxNodeSize:     3,
// 			DesiredNodeSize: 1,
// 			// NodeList: []irs.IID{},
// 			KeyValueList: []irs.KeyValue{
// 				{Key: "cluster_id", Value: "96c017e5-94d0-4001-bbb1-b1e768c75720"},
// 				{Key: "availability_zone", Value: "kr-pub-a"},
// 			},
// 		}

// 		// println(cluster.IId.NameId, cluster.IId.SystemId)
// 		t.Log(cluster)
// 		node_group, err := clusterHandler.AddNodeGroup(cluster.IId, new_node_group)
// 		if err != nil {
// 			t.Error(err)
// 		}
// 		// println(node_group.IId.NameId, node_group.IId.SystemId)
// 		t.Log(node_group)
// 	}
// }

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

// 	// node_group, err := clusterHandler.GetNodeGroup(irs.IID{NameId: "", SystemId: "cluster_id_not_exist"}, irs.IID{NameId: "", SystemId: "node_group_id_not_exist"})
// 	// if err != nil {
// 	// 	println(err.Error())
// 	// }
// 	// println(node_grop)
// }

// func TestSetNodeGroupAutoScaling(t *testing.T) {
// 	clusterHandler, err := getClusterHandler()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	_, err = clusterHandler.SetNodeGroupAutoScaling(irs.IID{NameId: "", SystemId: ""}, irs.IID{NameId: "", SystemId: ""}, true)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestChangeNodeGroupScaling(t *testing.T) {
// 	clusterHandler, err := getClusterHandler()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	_, err = clusterHandler.ChangeNodeGroupScaling(irs.IID{NameId: "", SystemId: ""}, irs.IID{NameId: "", SystemId: ""}, 1, 0, 5)
// 	if err != nil {
// 		t.Error(err)
// 	}
// }

// func TestRemoveNodeGroup(t *testing.T) {
// 	clusterHandler, err := getClusterHandler()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	clusters, _ := clusterHandler.ListCluster()
// 	for _, cluster := range clusters {
// 		node_groups, _ := clusterHandler.ListNodeGroup(cluster.IId)
// 		for _, node_group := range node_groups {
// 			res, _ := clusterHandler.RemoveNodeGroup(cluster.IId, node_group.IId)
// 			if err != nil {
// 				t.Error(err)
// 			}
// 			if res == false {
// 				t.Error("Failed to remove node group")
// 			}
// 		}
// 	}
// }

// func TestUpgradeCluster(t *testing.T) {
// 	clusterHandler, err := getClusterHandler()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	clusters, _ := clusterHandler.ListCluster()
// 	for _, cluster := range clusters {
// 		// payload := `{
// 		// 	"version": "v1.23.3"
// 		// }`
// 		res, err := clusterHandler.UpgradeCluster(cluster.IId, "v1.23.3")
// 		if err != nil {
// 			t.Error(err)
// 		}
// 		t.Log(res)
// 	}
// }

// func TestDeleteCluster(t *testing.T) {

// 	clusterHandler, err := getClusterHandler()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	clusters, _ := clusterHandler.ListCluster()
// 	for _, cluster := range clusters {
// 		println(cluster.IId.NameId, cluster.IId.SystemId)
// 		result, err := clusterHandler.DeleteCluster(cluster.IId)
// 		if err != nil {
// 			t.Error(err)
// 		}
// 		t.Log(result)
// 	}

// 	// result, err := clusterHandler.DeleteCluster(irs.IID{NameId: "cluster_not_exist", SystemId: "cluster_id_not_exist"})
// 	// if err != nil {
// 	// 	println(err.Error())
// 	// }
// 	// println(result)
// }
