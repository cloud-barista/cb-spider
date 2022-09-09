package pmks

import (
	"os"
	"testing"

	_ "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/tencent/main/pmks_test/env" // 위치 변경 하면 안됨. 환경설정 정보 읽기 전에 테스트 수행됨

	tdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/tencent"
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
			Region: os.Getenv("REGION"),
			Zone:   os.Getenv("ZONE"),
		},
	}

	cloudDriver := new(tdrv.TencentDriver)
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

func TestGetClusterHander(t *testing.T) {
	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	println(clusterHandler)
}

// func TestNewClusterInfo(t *testing.T) {

// 	temp := irs.ClusterInfo{
// 		IId: irs.IID{
// 			NameId:   "cluster-name",
// 			SystemId: "cluser-id",
// 		},
// 		Version: "1.21.2",
// 		Network: irs.NetworkInfo{
// 			VpcIID: irs.IID{
// 				NameId:   "",
// 				SystemId: "vpc-id",
// 			},
// 			SubnetIID: []irs.IID{
// 				{
// 					NameId:   "subnet-name",
// 					SystemId: "subnet-id",
// 				},
// 			},
// 			SecurityGroupIIDs: []irs.IID{
// 				{
// 					NameId:   "security-group-name",
// 					SystemId: "sg-id",
// 				},
// 			},
// 			KeyValueList: []irs.KeyValue{
// 				{
// 					Key:   "key",
// 					Value: "value",
// 				},
// 			},
// 		},
// 		NodeGroupList: []irs.NodeGroupInfo{
// 			{
// 				IId: irs.IID{
// 					NameId:   "test-node-group-name",
// 					SystemId: "test-node-group-id",
// 				},
// 				ImageIID: irs.IID{
// 					NameId:   "image-name",
// 					SystemId: "image-id",
// 				},
// 				VMSpecName:   "ecs.g6.large",
// 				RootDiskType: "disk_type",
// 				RootDiskSize: "20",
// 				KeyPairIID: irs.IID{
// 					NameId:   "keypair",
// 					SystemId: "keypair-id",
// 				},
// 				Status:          irs.NodeGroupCreating,
// 				OnAutoScaling:   false,
// 				DesiredNodeSize: 1,
// 				MinNodeSize:     1,
// 				MaxNodeSize:     1,
// 				NodeList: []irs.IID{
// 					{
// 						NameId:   "node-name",
// 						SystemId: "node-id",
// 					},
// 				},
// 				KeyValueList: []irs.KeyValue{
// 					{
// 						Key:   "key",
// 						Value: "value",
// 					},
// 				},
// 			},
// 		},
// 		Addons: irs.AddonsInfo{
// 			KeyValueList: []irs.KeyValue{
// 				{
// 					Key:   "ingress",
// 					Value: "nginx",
// 				},
// 			},
// 		},
// 		Status:      irs.ClusterCreating,
// 		CreatedTime: time.Now(),
// 		KeyValueList: []irs.KeyValue{
// 			{
// 				Key:   "test-key",
// 				Value: "test-value",
// 			},
// 		},
// 	}

// 	j, err := json.MarshalIndent(temp, "", "  ")
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}
// 	println(string(j))

// 	flat, _ := flatten.FlattenString(string(j), "", flatten.DotStyle)
// 	println(flat)

// 	// {
// 	// 	"Addons.KeyValueList.0.Key": "ingress",
// 	// 	"Addons.KeyValueList.0.Value": "nginx",
// 	// 	"CreatedTime": "2022-09-08T15:27:45.67002+09:00",
// 	// 	"IId.NameId": "cluster-name",
// 	// 	"IId.SystemId": "cluser-id",
// 	// 	"KeyValueList.0.Key": "test-key",
// 	// 	"KeyValueList.0.Value": "test-value",
// 	// 	"Network.KeyValueList.0.Key": "key",
// 	// 	"Network.KeyValueList.0.Value": "value",
// 	// 	"Network.SecurityGroupIIDs.0.NameId": "security-group-name",
// 	// 	"Network.SecurityGroupIIDs.0.SystemId": "sg-id",
// 	// 	"Network.SubnetIID.0.NameId": "subnet-name",
// 	// 	"Network.SubnetIID.0.SystemId": "subnet-id",
// 	// 	"Network.VpcIID.NameId": "",
// 	// 	"Network.VpcIID.SystemId": "vpc-id",
// 	// 	"NodeGroupList.0.DesiredNodeSize": 1,
// 	// 	"NodeGroupList.0.IId.NameId": "test-node-group-name",
// 	// 	"NodeGroupList.0.IId.SystemId": "test-node-group-id",
// 	// 	"NodeGroupList.0.ImageIID.NameId": "image-name",
// 	// 	"NodeGroupList.0.ImageIID.SystemId": "image-id",
// 	// 	"NodeGroupList.0.KeyPairIID.NameId": "keypair",
// 	// 	"NodeGroupList.0.KeyPairIID.SystemId": "keypair-id",
// 	// 	"NodeGroupList.0.KeyValueList.0.Key": "key",
// 	// 	"NodeGroupList.0.KeyValueList.0.Value": "value",
// 	// 	"NodeGroupList.0.MaxNodeSize": 1,
// 	// 	"NodeGroupList.0.MinNodeSize": 1,
// 	// 	"NodeGroupList.0.NodeList.0.NameId": "node-name",
// 	// 	"NodeGroupList.0.NodeList.0.SystemId": "node-id",
// 	// 	"NodeGroupList.0.OnAutoScaling": false,
// 	// 	"NodeGroupList.0.RootDiskSize": "20",
// 	// 	"NodeGroupList.0.RootDiskType": "disk_type",
// 	// 	"NodeGroupList.0.Status": "Creating",
// 	// 	"NodeGroupList.0.VMSpecName": "ecs.g6.large",
// 	// 	"Status": "Creating",
// 	// 	"Version": "1.21.2"
// 	// }

// }

func TestCreateClusterOnly(t *testing.T) {

	t.Log("클러스터 생성, 노드그룹은 생성안함")

	clusterHandler, err := getClusterHandler()
	if err != nil {
		t.Error(err)
	}

	// // Instantiate a request object. You can further set the request parameters according to the API called and actual conditions
	// request := tke.NewCreateClusterRequest()

	// // request.FromJsonString()
	// request.ClusterCIDRSettings = &tke.ClusterCIDRSettings{
	// 	ClusterCIDR: common.StringPtr("172.20.0.0/16"), // 172.X.0.0.16: X Range:16, 17, ... , 31
	// }
	// request.ClusterBasicSettings = &tke.ClusterBasicSettings{
	// 	ClusterName:    common.StringPtr("cluster-x2"),
	// 	VpcId:          common.StringPtr("vpc-q1c6fr9e"),
	// 	ClusterVersion: common.StringPtr("1.22.5"), //version: 1.22.5
	// }
	// request.ClusterType = common.StringPtr("MANAGED_CLUSTER")

	// res, err := tencent.CreateCluster(secret_id, secret_key, region_id, request)
	// if err != nil {
	// 	t.Errorf("CreateCluster failed: %v", err)
	// 	return
	// }

	clusterInfo := irs.ClusterInfo{
		IId: irs.IID{
			NameId:   "cluster-x1",
			SystemId: "",
		},
		Version: "1.22.5",
		Network: irs.NetworkInfo{
			VpcIID: irs.IID{NameId: "", SystemId: "vpc-q1c6fr9e"},
		},
		KeyValueList: []irs.KeyValue{
			{
				Key:   "cluster_cidr", // 조회가능한 값이면, 내부에서 처리하는 코드 추가
				Value: "172.20.0.0/16",
			},
		},
	}

	cluster_, err := clusterHandler.CreateCluster(clusterInfo)
	if err != nil {
		t.Error(err)
	}

	t.Log(cluster_)
}

// func TestCreateClusterWith1NodeGroup(t *testing.T) {
// 	//
// 	t.Log("클러스터 + 노드그룹 생성")

// 	clusterHandler, err := getClusterHandler()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	clusterInfo := irs.ClusterInfo{
// 		IId: irs.IID{
// 			NameId:   "cluster-x",
// 			SystemId: "",
// 		},
// 		Version: "1.22.10-aliyun.1",
// 		Network: irs.NetworkInfo{
// 			VpcIID:            irs.IID{NameId: "", SystemId: "vpc-2zek5slojo5bh621ftnrg"},
// 			SubnetIID:         []irs.IID{},
// 			SecurityGroupIIDs: []irs.IID{},
// 			KeyValueList:      []irs.KeyValue{},
// 		},
// 		NodeGroupList: []irs.NodeGroupInfo{
// 			{
// 				IId: irs.IID{
// 					NameId:   "test-node-group-name",
// 					SystemId: "test-node-group-id",
// 				},
// 				ImageIID: irs.IID{
// 					NameId:   "image-name",
// 					SystemId: "image-id",
// 				},
// 				VMSpecName:   "ecs.g6.large",
// 				RootDiskType: "disk_type",
// 				RootDiskSize: "20",
// 				KeyPairIID: irs.IID{
// 					NameId:   "keypair",
// 					SystemId: "keypair-id",
// 				},
// 				Status:          irs.NodeGroupCreating,
// 				OnAutoScaling:   false,
// 				DesiredNodeSize: 1,
// 				MinNodeSize:     1,
// 				MaxNodeSize:     1,
// 				NodeList: []irs.IID{
// 					{
// 						NameId:   "node-name",
// 						SystemId: "node-id",
// 					},
// 				},
// 				KeyValueList: []irs.KeyValue{
// 					{
// 						Key:   "key",
// 						Value: "value",
// 					},
// 				},
// 			},
// 		},
// 		Addons: irs.AddonsInfo{
// 			KeyValueList: []irs.KeyValue{
// 				{
// 					Key:   "ingress",
// 					Value: "nginx",
// 				},
// 			},
// 		},
// 		Status:      irs.ClusterCreating,
// 		CreatedTime: time.Now(),
// 		KeyValueList: []irs.KeyValue{
// 			{
// 				Key:   "test-key",
// 				Value: "test-value",
// 			},
// 		},
// 	}

// 	// container_cidr + ?
// 	// service_cidr + ?
// 	// login_password + ?
// 	// master_vswitch_ids

// 	cluster_, err := clusterHandler.CreateCluster(clusterInfo)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	t.Log(cluster_)
// }

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
// 		t.Log(cluster.IId.SystemId)
// 		println(cluster.IId.NameId, cluster.Status)
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

// // // func TestDeleteCluster(t *testing.T) {
// // // }

// func TestAddNodeGroup(t *testing.T) {
// 	clusterHandler, err := getClusterHandler()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	// body := `{
// 	// 	"nodepool_info": {
// 	// 		"name": "nodepoolx"
// 	// 	},
// 	// 	"auto_scaling": {
// 	// 		"enable": true,
// 	// 		"max_instances": 5,
// 	// 		"min_instances": 0
// 	// 	},
// 	// 	"scaling_group": {
// 	// 		"instance_charge_type": "PostPaid",
// 	// 		"instance_types": ["ecs.c6.xlarge"],
// 	// 		"key_pair": "kp1",
// 	// 		"system_disk_category": "cloud_essd",
// 	// 		"system_disk_size": 70,
// 	// 		"vswitch_ids": ["vsw-2ze0qpwcio7r5bx3nqbp1"]
// 	// 	},
// 	// 	"management": {
// 	// 		"enable":true
// 	// 	}
// 	// }`

// 	new_node_group := &irs.NodeGroupInfo{
// 		IId:             irs.IID{NameId: "nodepoolx101", SystemId: ""},
// 		ImageIID:        irs.IID{NameId: "", SystemId: "image_id"}, // 이미지 id 선택 추가
// 		VMSpecName:      "ecs.c6.xlarge",
// 		RootDiskType:    "cloud_essd",
// 		RootDiskSize:    "70",
// 		KeyPairIID:      irs.IID{NameId: "kp1", SystemId: ""},
// 		OnAutoScaling:   true,
// 		DesiredNodeSize: 1,
// 		MinNodeSize:     0,
// 		MaxNodeSize:     3,
// 		// KeyValueList: []irs.KeyValue{ // 클러스터 조회해서 처리한다. // //vswitch_id":"vsw-2ze0qpwcio7r5bx3nqbp1"
// 		// 	{
// 		// 		Key:   "vswitch_ids",
// 		// 		Value: "vsw-2ze0qpwcio7r5bx3nqbp1",
// 		// 	},
// 		// },
// 	}

// 	clusters, _ := clusterHandler.ListCluster()
// 	for _, cluster := range clusters {
// 		// println(cluster.IId.NameId, cluster.IId.SystemId)
// 		t.Log(cluster)
// 		node_group, err := clusterHandler.AddNodeGroup(cluster.IId, *new_node_group)
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

// 	node_group, err := clusterHandler.GetNodeGroup(irs.IID{NameId: "", SystemId: "cluster_id_not_exist"}, irs.IID{NameId: "", SystemId: "node_group_id_not_exist"})
// 	if err != nil {
// 		println(err.Error())
// 	}
// 	println(node_group.IId.NameId)
// }

// func TestSetNodeGroupAutoScaling(t *testing.T) {
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

// 			res, err := clusterHandler.SetNodeGroupAutoScaling(cluster.IId, node_group_.IId, true)
// 			if err != nil {
// 				t.Error(err)
// 			}
// 			println(res)
// 		}
// 	}
// }

// func TestChangeNodeGroupScaling(t *testing.T) {
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

// 			res, err := clusterHandler.ChangeNodeGroupScaling(cluster.IId, node_group_.IId, 1, 0, 5)
// 			if err != nil {
// 				t.Error(err)
// 			}
// 			println(res.IId.NameId, res.IId.SystemId)
// 		}
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
// 		res, err := clusterHandler.UpgradeCluster(cluster.IId, "1.22.3-aliyun.1")
// 		// res, err := clusterHandler.UpgradeCluster(cluster.IId, "1.22.3-aliyun.x")
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
