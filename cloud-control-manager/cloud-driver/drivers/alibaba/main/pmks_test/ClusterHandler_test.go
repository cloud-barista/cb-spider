package pmks

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/alibaba/main/pmks_test/env" // 위치 변경 하면 안됨. 환경설정 정보 읽기 전에 테스트 수행됨
	"github.com/jeremywohl/flatten"

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

func TestNewClusterInfo(t *testing.T) {

	temp := irs.ClusterInfo{
		IId: irs.IID{
			NameId:   "cluster-name",
			SystemId: "cluser-id",
		},
		Version: "1.21.2",
		Network: irs.NetworkInfo{
			VpcIID: irs.IID{
				NameId:   "",
				SystemId: "vpc-id",
			},
			SubnetIID: []irs.IID{
				{
					NameId:   "subnet-name",
					SystemId: "subnet-id",
				},
			},
			SecurityGroupIIDs: []irs.IID{
				{
					NameId:   "security-group-name",
					SystemId: "sg-id",
				},
			},
			KeyValueList: []irs.KeyValue{
				{
					Key:   "key",
					Value: "value",
				},
			},
		},
		NodeGroupList: []irs.NodeGroupInfo{
			{
				IId: irs.IID{
					NameId:   "test-node-group-name",
					SystemId: "test-node-group-id",
				},
				ImageIID: irs.IID{
					NameId:   "image-name",
					SystemId: "image-id",
				},
				VMSpecName:   "ecs.g6.large",
				RootDiskType: "disk_type",
				RootDiskSize: "20",
				KeyPairIID: irs.IID{
					NameId:   "keypair",
					SystemId: "keypair-id",
				},
				Status:          irs.NodeGroupCreating,
				OnAutoScaling:   false,
				DesiredNodeSize: 1,
				MinNodeSize:     1,
				MaxNodeSize:     1,
				NodeList: []irs.IID{
					{
						NameId:   "node-name",
						SystemId: "node-id",
					},
				},
				KeyValueList: []irs.KeyValue{
					{
						Key:   "key",
						Value: "value",
					},
				},
			},
		},
		Addons: irs.AddonsInfo{
			KeyValueList: []irs.KeyValue{
				{
					Key:   "ingress",
					Value: "nginx",
				},
			},
		},
		Status:      irs.ClusterCreating,
		CreatedTime: time.Now(),
		KeyValueList: []irs.KeyValue{
			{
				Key:   "test-key",
				Value: "test-value",
			},
		},
	}

	j, err := json.MarshalIndent(temp, "", "  ")
	if err != nil {
		fmt.Println(err)
		return
	}
	println(string(j))

	flat, _ := flatten.FlattenString(string(j), "", flatten.DotStyle)
	println(flat)

}

func TestCreateClusterOnly(t *testing.T) {

	t.Log("클러스터 생성, 노드그룹은 생성안함")

	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	// body := `{
	// 	"name": "cluster_2",
	// 	"region_id": "cn-beijing",
	// 	"cluster_type": "ManagedKubernetes",
	// 	"kubernetes_version": "1.22.10-aliyun.1",
	// 	"vpcid": "vpc-2zek5slojo5bh621ftnrg",
	// 	"container_cidr": "172.24.0.0/16",
	// 	"service_cidr": "172.23.0.0/16",
	// 	"num_of_nodes": 0,
	// 	"master_vswitch_ids": [
	// 		"vsw-2ze0qpwcio7r5bx3nqbp1"
	// 	]
	// }`

	clusterInfo := irs.ClusterInfo{
		IId: irs.IID{
			NameId:   "cluster-x",
			SystemId: "",
		},
		Version: "1.22.10-aliyun.1",
		Network: irs.NetworkInfo{
			VpcIID: irs.IID{NameId: "", SystemId: "vpc-2zek5slojo5bh621ftnrg"},
		},
		KeyValueList: []irs.KeyValue{
			{
				Key:   "container_cidr",
				Value: "172.22.0.0/16",
			},
			{
				Key:   "service_cidr",
				Value: "172.23.0.0/16",
			},
			{
				Key:   "master_vswitch_id",
				Value: "vsw-2ze0qpwcio7r5bx3nqbp1",
			},
		},
	}

	// container_cidr + ?
	// service_cidr + ?
	// login_password + ?
	// master_vswitch_ids
	//cidr: Valid values: 10.0.0.0/16-24, 172.16-31.0.0/16-24, and 192.168.0.0/16-24.
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
		Version: "1.22.10-aliyun.1",
		Network: irs.NetworkInfo{
			VpcIID:            irs.IID{NameId: "", SystemId: "vpc-2zek5slojo5bh621ftnrg"},
			SubnetIID:         []irs.IID{},
			SecurityGroupIIDs: []irs.IID{},
			KeyValueList:      []irs.KeyValue{},
		},
		NodeGroupList: []irs.NodeGroupInfo{
			{
				IId: irs.IID{
					NameId:   "test-node-group-name",
					SystemId: "test-node-group-id",
				},
				ImageIID: irs.IID{
					NameId:   "image-name",
					SystemId: "image-id",
				},
				VMSpecName:   "ecs.g6.large",
				RootDiskType: "disk_type",
				RootDiskSize: "20",
				KeyPairIID: irs.IID{
					NameId:   "keypair",
					SystemId: "keypair-id",
				},
				Status:          irs.NodeGroupCreating,
				OnAutoScaling:   false,
				DesiredNodeSize: 1,
				MinNodeSize:     1,
				MaxNodeSize:     1,
				NodeList: []irs.IID{
					{
						NameId:   "node-name",
						SystemId: "node-id",
					},
				},
				KeyValueList: []irs.KeyValue{
					{
						Key:   "key",
						Value: "value",
					},
				},
			},
		},
		Addons: irs.AddonsInfo{
			KeyValueList: []irs.KeyValue{
				{
					Key:   "ingress",
					Value: "nginx",
				},
			},
		},
		Status:      irs.ClusterCreating,
		CreatedTime: time.Now(),
		KeyValueList: []irs.KeyValue{
			{
				Key:   "test-key",
				Value: "test-value",
			},
		},
	}

	// container_cidr + ?
	// service_cidr + ?
	// login_password + ?
	// master_vswitch_ids

	cluster_, err := clusterHandler.CreateCluster(clusterInfo)
	if err != nil {
		t.Error(err)
	}

	t.Log(cluster_)
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

func TestAddNodeGroup(t *testing.T) {
	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	// body := `{
	// 	"nodepool_info": {
	// 		"name": "nodepoolx"
	// 	},
	// 	"auto_scaling": {
	// 		"enable": true,
	// 		"max_instances": 5,
	// 		"min_instances": 0
	// 	},
	// 	"scaling_group": {
	// 		"instance_charge_type": "PostPaid",
	// 		"instance_types": ["ecs.c6.xlarge"],
	// 		"key_pair": "kp1",
	// 		"system_disk_category": "cloud_essd",
	// 		"system_disk_size": 70,
	// 		"vswitch_ids": ["vsw-2ze0qpwcio7r5bx3nqbp1"]
	// 	},
	// 	"management": {
	// 		"enable":true
	// 	}
	// }`

	new_node_group := &irs.NodeGroupInfo{
		IId:             irs.IID{NameId: "nodepoolx101", SystemId: ""},
		ImageIID:        irs.IID{NameId: "", SystemId: "image_id"}, // 이미지 id 선택 추가
		VMSpecName:      "ecs.c6.xlarge",
		RootDiskType:    "cloud_essd",
		RootDiskSize:    "70",
		KeyPairIID:      irs.IID{NameId: "kp1", SystemId: ""},
		OnAutoScaling:   true,
		DesiredNodeSize: 1,
		MinNodeSize:     0,
		MaxNodeSize:     3,
		// KeyValueList: []irs.KeyValue{ // 클러스터 조회해서 처리한다. // //vswitch_id":"vsw-2ze0qpwcio7r5bx3nqbp1"
		// 	{
		// 		Key:   "vswitch_ids",
		// 		Value: "vsw-2ze0qpwcio7r5bx3nqbp1",
		// 	},
		// },
	}

	clusters, _ := clusterHandler.ListCluster()
	for _, cluster := range clusters {
		// println(cluster.IId.NameId, cluster.IId.SystemId)
		t.Log(cluster)
		node_group, err := clusterHandler.AddNodeGroup(cluster.IId, *new_node_group)
		if err != nil {
			t.Error(err)
		}
		// println(node_group.IId.NameId, node_group.IId.SystemId)
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
