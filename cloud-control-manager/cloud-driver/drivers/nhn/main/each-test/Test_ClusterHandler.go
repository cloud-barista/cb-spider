// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// by ETRI, 2021.0512.

package main

import (
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"os"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	nhndrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/nhn"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NHN Cloud Resource Test")
	cblog.SetLevel("info")
}

func testErr() error {

	return errors.New("")
}

// Test Cluster
func handleCluster() {
	cblogger.Debug("Start Cluster Resource Test")

	ResourceHandler, err := getResourceHandler("Cluster")
	if err != nil {
		panic(err)
	}
	//config := readConfigFile()

	clusterHandler := ResourceHandler.(irs.ClusterHandler)

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ Cluster Management Test ]")
		fmt.Println("1. Create Cluster")
		fmt.Println("2. List Cluster")
		fmt.Println("3. Get Cluster")
		fmt.Println("4. Delete Cluster")
		fmt.Println("5. Add NodeGroup")
		fmt.Println("6. List NodeGroup")
		fmt.Println("7. Get NodeGroup")
		fmt.Println("8. Set NodeGroup AutoScaling")
		fmt.Println("9. Change NodeGroup Scale")
		fmt.Println("10. Remove NodeGroup")
		fmt.Println("11. Upgrade Cluster")
		fmt.Println("0. Quit")
		fmt.Println("\n   Select a number above!! : ")
		fmt.Println("============================================================================================")

		var commandNum int

		clusterIId := irs.IID{ // For Get()/Delete() Test
			NameId:   "nhn-cluster-01",
			SystemId: "b80c6c3d-7db7-4aaa-9aa7-0891c7e95f89",
		}

		nodeGroupIId := irs.IID{ // For Get()/ChangeNodeGroupScaling()/Remove() Test
			NameId:   "nhn-nodeGroup-01",
			SystemId: "b80c6c3d-7db7-4aaa-9aa7-0891c7e95f89",
		}

		networkInfo := irs.NetworkInfo{
			VpcIID: irs.IID{
				NameId:   "nhn-vpc-01",
				SystemId: "b80c6c3d-7db7-4aaa-9aa7-0891c7e95f89",
			},
			SubnetIIDs: []irs.IID{
				{
					NameId:   "nhn-subnet-01",
					SystemId: "b80c6c3d-7db7-4aaa-9aa7-0891c7e95f89",
				},
			},
			SecurityGroupIIDs: []irs.IID{
				{
					NameId:   "nhn-sg-01",
					SystemId: "b80c6c3d-7db7-4aaa-9aa7-0891c7e95f89",
				},
			},
		}

		nodeReqInfolist := []irs.NodeGroupInfo{
			{
				IId: irs.IID{
					NameId: "nhn-nodeGroup-01",
				},

				// ImageIID     IID
				// VMSpecName   string
				// RootDiskType string // "SSD(gp2)", "Premium SSD", ...
				// RootDiskSize string // "", "default", "50", "1000" (GB)
				// KeyPairIID   IID
			},
		}

		addNodeReqInfo := irs.NodeGroupInfo{ // For AddNodeGroup() Test
			IId: irs.IID{
				NameId: "nhn-nodeGroup-02",
			},

			// ImageIID     IID
			// VMSpecName   string
			// RootDiskType string // "SSD(gp2)", "Premium SSD", ...
			// RootDiskSize string // "", "default", "50", "1000" (GB)
			// KeyPairIID   IID
		}

		clusterReqInfo := irs.ClusterInfo{
			IId: irs.IID{
				NameId: "nhn-cluster-01",
			},
			Version:       "1.22.12", // Kubernetes Version, ex) 1.23.3
			Network:       networkInfo,
			NodeGroupList: nodeReqInfolist,
			// Addons        AddonsInfo
		}

		desiredNodeSize := 10
		minNodeSize := 5
		maxNodeSize := 20
		setNodeGroupAutoScaling := true
		newClusterVersion := "1.23.9"

		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return

			case 1:
				cblogger.Info("Start CreateCluster() ...")
				result, err := clusterHandler.CreateCluster(clusterReqInfo)
				if err != nil {
					cblogger.Error(err)
				} else {
					cblogger.Info("# Cluster 생성 결과 : \n")
					spew.Dump(result)
				}

				cblogger.Info("CreateCluster Test Finished!!")

			case 2:
				cblogger.Info("Start ListCluster() ...")
				result, err := clusterHandler.ListCluster()
				if err != nil {
					cblogger.Error(err)
				} else {
					cblogger.Info("# Cluster list 조회 결과 : \n")
					spew.Dump(result)

					cblogger.Infof("=========== Cluster list 수 : [%d] ================", len(result))
				}

				cblogger.Info("ListCluster Test Finished!!")

			case 3:
				cblogger.Info("Start GetCluster() ...")
				result, err := clusterHandler.GetCluster(clusterIId)
				if err != nil {
					cblogger.Error(err)
				} else {
					cblogger.Info("# Cluster 조회 결과 : \n")
					spew.Dump(result)
				}

				cblogger.Info("GetCluster Test Finished!!")

			case 4:
				cblogger.Info("Start DeleteCluster() ...")
				result, err := clusterHandler.DeleteCluster(clusterIId)
				if err != nil {
					cblogger.Error(err)
				} else {
					cblogger.Info("# Cluster 삭제 결과 : ")
					spew.Dump(result)
				}

				cblogger.Info("DeleteCluster Test Finished!!")

			case 5:
				cblogger.Info("Start AddNodeGroup() ...")
				result, err := clusterHandler.AddNodeGroup(clusterIId, addNodeReqInfo)
				if err != nil {
					cblogger.Error(err)
				} else {
					cblogger.Info("# AddNodeGroup 결과 : ")
					spew.Dump(result)
				}

				cblogger.Info("AddNodeGroup Test Finished!!")

			case 6:
				cblogger.Info("Start ListNodeGroup() ...")
				result, err := clusterHandler.ListNodeGroup(clusterIId)
				if err != nil {
					cblogger.Error(err)
				} else {
					cblogger.Info("# Cluster list 조회 결과 : \n")
					spew.Dump(result)

					cblogger.Infof("=========== NodeGroup list 수 : [%d] ================", len(result))
				}

				cblogger.Info("ListNodeGroup Test Finished!!")

			case 7:
				cblogger.Info("Start GetNodeGroup() ...")
				result, err := clusterHandler.GetNodeGroup(clusterIId, nodeGroupIId)
				if err != nil {
					cblogger.Error(err)
				} else {
					cblogger.Info("# GetNodeGroup 결과 : \n")
					spew.Dump(result)
				}

				cblogger.Info("GetNodeGroup Test Finished!!")

			case 8:
				cblogger.Info("Start SetNodeGroupAutoScaling() ...")
				result, err := clusterHandler.SetNodeGroupAutoScaling(clusterIId, nodeGroupIId, setNodeGroupAutoScaling)
				if err != nil {
					cblogger.Error(err)
				} else {
					cblogger.Info("# SetNodeGroupAutoScaling 결과 : \n")
					spew.Dump(result)
				}

				cblogger.Info("SetNodeGroupAutoScaling Test Finished!!")

			case 9:
				cblogger.Info("Start ChangeNodeGroupScaling() ...")
				result, err := clusterHandler.ChangeNodeGroupScaling(clusterIId, nodeGroupIId, desiredNodeSize, minNodeSize, maxNodeSize)
				if err != nil {
					cblogger.Error(err)
				} else {
					cblogger.Info("# ChangeNodeGroupScaling 결과 : \n")
					spew.Dump(result)
				}

				cblogger.Info("ChangeNodeGroupScaling Test Finished!!")

			case 10:
				cblogger.Info("Start RemoveNodeGroup() ...")
				result, err := clusterHandler.RemoveNodeGroup(clusterIId, nodeGroupIId)
				if err != nil {
					cblogger.Error(err)
				} else {
					cblogger.Info("# RemoveNodeGroup 결과 : \n")
					spew.Dump(result)
				}

				cblogger.Info("RemoveNodeGroup Test Finished!!")

			case 11:
				cblogger.Info("Start RemoveNodeGroup() ...")
				result, err := clusterHandler.UpgradeCluster(clusterIId, newClusterVersion)
				if err != nil {
					cblogger.Error(err)
				} else {
					cblogger.Info("# RemoveNodeGroup 결과 : \n")
					spew.Dump(result)
				}

				cblogger.Info("RemoveNodeGroup Test Finished!!")
			}
		}
	}
}

func main() {
	cblogger.Info("NHN Cloud Resource Test")

	handleCluster()
}

// handlerType : resources폴더의 xxxHandler.go에서 Handler이전까지의 문자열
// (예) ImageHandler.go -> "Image"
func getResourceHandler(handlerType string) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(nhndrv.NhnCloudDriver)

	config := readConfigFile()
	// spew.Dump(config)

	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			IdentityEndpoint: config.NhnCloud.IdentityEndpoint,
			Username:         config.NhnCloud.Nhn_Username,
			Password:         config.NhnCloud.Api_Password,
			DomainName:       config.NhnCloud.DomainName,
			TenantId:         config.NhnCloud.TenantId,
		},
		RegionInfo: idrv.RegionInfo{
			Region: config.NhnCloud.Region,
			Zone:   config.NhnCloud.Zone,
		},
	}

	cloudConnection, errCon := cloudDriver.ConnectCloud(connectionInfo)
	if errCon != nil {
		return nil, errCon
	}

	var resourceHandler interface{}
	var err error

	switch handlerType {
	case "Image":
		resourceHandler, err = cloudConnection.CreateImageHandler()
	case "KeyPair":
		resourceHandler, err = cloudConnection.CreateKeyPairHandler()
	case "Security":
		resourceHandler, err = cloudConnection.CreateSecurityHandler()
	case "VNetwork":
		resourceHandler, err = cloudConnection.CreateVPCHandler()
	case "VM":
		resourceHandler, err = cloudConnection.CreateVMHandler()
	case "VMSpec":
		resourceHandler, err = cloudConnection.CreateVMSpecHandler()
	case "VPC":
		resourceHandler, err = cloudConnection.CreateVPCHandler()
	case "Cluster":
		resourceHandler, err = cloudConnection.CreateClusterHandler()
	}
	if err != nil {
		return nil, err
	}
	return resourceHandler, nil
}

type Config struct {
	NhnCloud struct {
		IdentityEndpoint string `yaml:"identity_endpoint"`
		Nhn_Username     string `yaml:"nhn_username"`
		Api_Password     string `yaml:"api_password"`
		DomainName       string `yaml:"domain_name"`
		TenantId         string `yaml:"tenant_id"`
		Region           string `yaml:"region"`
		Zone             string `yaml:"zone"`
	} `yaml:"nhncloud"`
}

func readConfigFile() Config {
	// Set Environment Value of Project Root Path
	rootPath := os.Getenv("CBSPIDER_ROOT")
	configPath := rootPath + "/cloud-control-manager/cloud-driver/drivers/nhn/main/conf/config.yaml"
	cblogger.Info("Config file : " + configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		cblogger.Error(err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		cblogger.Error(err)
	}
	return config
}
