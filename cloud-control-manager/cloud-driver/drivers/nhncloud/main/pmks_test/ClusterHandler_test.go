package pmks

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/jeremywohl/flatten"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	// nhndrv "github.com/cloud-barista/nhncloud/nhncloud"
	nhndrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/nhncloud"
)

func printObj2Flat(object interface{}) {
	temp, err := json.MarshalIndent(object, "", "  ")
	if err != nil {
		println(err)
	} else {
		flat, err := flatten.FlattenString(string(temp), "", flatten.DotStyle)
		if err != nil {
			println(err)
		} else {
			println(flat)
		}
	}
}

func getClusterHandler() (irs.ClusterHandler, error) {

	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			IdentityEndpoint: os.Getenv("ID_ENDPOINT"),
			Username:         os.Getenv("USERNAME"),
			DomainName:       os.Getenv("DOMAIN_NAME"),
			Password:         os.Getenv("PASSWORD"),
			TenantId:         os.Getenv("TENANT_ID"),
		},
		RegionInfo: idrv.RegionInfo{
			Region: os.Getenv("REGION_NAME"),
			Zone:   os.Getenv("REGION_ZONE"),
		},
	}

	cloudDriver := new(nhndrv.NhnCloudDriver)
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
			NameId:   "cluster-test1",
			SystemId: "",
		},
		Version: "v1.23.3",
		Network: irs.NetworkInfo{
			VpcIID: irs.IID{
				NameId:   "",
				SystemId: "8a07c257-72ba-4183-bb77-fa46f6d12c39",
			},
			SubnetIIDs: []irs.IID{
				{
					NameId:   "",
					SystemId: "e4ba747b-d9f5-45a2-b749-45567054bbe5",
				},
			},
		},
		NodeGroupList: []irs.NodeGroupInfo{
			{
				IId: irs.IID{
					NameId:   "default-worker",
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
			},
		},
	}

	cluster, err := clusterHandler.CreateCluster(clusterInfo)
	if err != nil {
		t.Error(err)
	}
	printObj2Flat(cluster)

	cluster_, err := clusterHandler.GetCluster(cluster.IId)
	if err != nil {
		println(err.Error())
	}
	t.Log(cluster_)
	printObj2Flat(cluster_)

	t.Log(cluster)
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
		t.Log(cluster)
		printObj2Flat(cluster)
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
		printObj2Flat(cluster_)
	}
}

// func TestDeleteCluster(t *testing.T) {
// }

func TestAddNodeGroup(t *testing.T) {
	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	clusters, _ := clusterHandler.ListCluster()
	for _, cluster := range clusters {
		new_node_group := irs.NodeGroupInfo{
			IId:             irs.IID{NameId: "added-nodegroup", SystemId: ""},
			ImageIID:        irs.IID{NameId: "", SystemId: "2717ec03-3a4d-4728-b372-183065facdba"},
			VMSpecName:      "13646526-0bb9-400b-929f-797fdb7547eb",
			RootDiskType:    "General HDD",
			RootDiskSize:    "20",
			KeyPairIID:      irs.IID{NameId: "kp1", SystemId: ""},
			OnAutoScaling:   true,
			MinNodeSize:     1,
			MaxNodeSize:     3,
			DesiredNodeSize: 1,
		}

		t.Log(cluster)
		node_group, err := clusterHandler.AddNodeGroup(cluster.IId, new_node_group)
		if err != nil {
			t.Error(err)
		}

		t.Log(node_group)

		printObj2Flat(node_group)
	}
}

func TestListNodeGroup(t *testing.T) {
	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	clusters, _ := clusterHandler.ListCluster()
	for _, cluster := range clusters {
		node_groups, _ := clusterHandler.ListNodeGroup(cluster.IId)
		for _, node_group := range node_groups {
			t.Log(node_group.IId.NameId, node_group.IId.SystemId)
			t.Log(node_group)
			printObj2Flat(node_group)
		}

	}
}

func TestGetNodeGroup(t *testing.T) {
	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	clusters, _ := clusterHandler.ListCluster()
	for _, cluster := range clusters {
		node_groups, _ := clusterHandler.ListNodeGroup(cluster.IId)
		for _, node_group := range node_groups {
			node_group_, err := clusterHandler.GetNodeGroup(cluster.IId, node_group.IId)
			if err != nil {
				t.Error(err)
			}
			t.Log(node_group_.IId.NameId, node_group_.IId.SystemId)
			t.Log(node_group_)
			printObj2Flat(node_group_)
		}
	}
}

func TestSetNodeGroupAutoScaling(t *testing.T) {
	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	clusters, _ := clusterHandler.ListCluster()
	for _, cluster := range clusters {
		node_groups, _ := clusterHandler.ListNodeGroup(cluster.IId)
		for _, node_group := range node_groups {

			res, err := clusterHandler.SetNodeGroupAutoScaling(cluster.IId, node_group.IId, false)
			if err != nil {
				t.Error(err)
			}
			t.Log(res)

			res, err = clusterHandler.SetNodeGroupAutoScaling(cluster.IId, node_group.IId, true)
			if err != nil {
				t.Error(err)
			}
			t.Log(res)
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
		node_groups, _ := clusterHandler.ListNodeGroup(cluster.IId)
		for _, node_group := range node_groups {

			res, err := clusterHandler.ChangeNodeGroupScaling(cluster.IId, node_group.IId, 0, 2, 5)
			if err != nil {
				t.Error(err)
			}
			t.Log(res)
			printObj2Flat(res)

			res, err = clusterHandler.ChangeNodeGroupScaling(cluster.IId, node_group.IId, 0, 1, 3)
			if err != nil {
				t.Error(err)
			}
			t.Log(res)
			printObj2Flat(res)
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
		node_groups, _ := clusterHandler.ListNodeGroup(cluster.IId)
		for _, node_group := range node_groups {
			res, err := clusterHandler.RemoveNodeGroup(cluster.IId, node_group.IId)
			if err != nil {
				t.Error(err)
			}
			if res == false {
				t.Error("Failed to remove node group.")
				//"deleting the last nodegroup is not supported
			}
			printObj2Flat(res)
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
		//res, err := clusterHandler.UpgradeCluster(cluster.IId, "v1.23.3")
		res, err := clusterHandler.UpgradeCluster(cluster.IId, "v1.24.3")
		if err != nil {
			t.Error(err)
		}
		t.Log(res)
		printObj2Flat(res)
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
		printObj2Flat(result)
	}
}

func TestAll(t *testing.T) {
	// CSP 리전별 시나리오 기반 테스트
}
