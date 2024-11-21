// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// Updated by ETRI, 2024.11.

package main

import (
	"os"
	"fmt"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"github.com/davecgh/go-spew/spew"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cblog "github.com/cloud-barista/cb-log"

	// ncpvpcdrv "github.com/cloud-barista/ncpvpc/ncpvpc"  // For local test
	ncpvpcdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncpvpc"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP VPC Resource Test")
	cblog.SetLevel("info")
}

// Test VMSpec
func handleVPC() {
	cblogger.Debug("Start VMSpecHandler Resource Test")

	ResourceHandler, err := getResourceHandler("VPC")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.VPCHandler)

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ VPC Resource Test ]")
		fmt.Println("1. CreateVPC()")
		fmt.Println("2. ListVPC()")
		fmt.Println("3. GetVPC()")
		fmt.Println("4. AddSubnet()")
		fmt.Println("5. RemoveSubnet()")
		fmt.Println("6. DeleteVPC()")
		fmt.Println("7. ListIID()")
		fmt.Println("0. Exit")
		fmt.Println("\n   Select a number above!! : ")
		fmt.Println("============================================================================================")
		
		reqVPCName := "ncp-vpc-01"
		vpcId := "40859"
		subnetId := "3176"

		vpcIId := irs.IID{NameId: reqVPCName, SystemId: vpcId}
		subnetIId := irs.IID{SystemId: subnetId}

		cblogger.Info("reqVPCName : ", reqVPCName)

		vpcReqInfo := irs.VPCReqInfo {
			IId: irs.IID {NameId: reqVPCName, SystemId: vpcId},
			IPv4_CIDR: "10.0.0.0/16",
			// IPv4_CIDR: "172.16.0.0/24",
			SubnetInfoList: []irs.SubnetInfo {
				{
					IId: irs.IID{
						NameId: "ncp-subnet-for-vm",
					},
					IPv4_CIDR: "10.0.0.0/28",
					// IPv4_CIDR: "172.16.0.0/28",
				},
				// {
				// 	IId: irs.IID{
				// 		NameId: "ncp-subnet-04",
				// 	},
				// 	IPv4_CIDR: "172.16.1.0/28",
				// },
			},
		}

		subnetInfo := irs.SubnetInfo {
				IId: irs.IID{
					NameId: "ncp-subnet-05",
				},
				IPv4_CIDR: "172.16.2.0/24",
			}

		var commandNum int

		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				fmt.Println("Start CreateVPC() ...")
				vpcInfo, err := handler.CreateVPC(vpcReqInfo)
				if err != nil {
					//panic(err)
					cblogger.Error(err)
					cblogger.Error("Failed to create VPC : ", err)
				} else {
					cblogger.Info("VPC creation completed!!", vpcInfo)
					spew.Dump(vpcInfo)
					cblogger.Debug(vpcInfo)
				}
				fmt.Println("\nCreateVPC() Test Finished")

			case 2:
				fmt.Println("Start ListVPC() ...")
				result, err := handler.ListVPC()
				if err != nil {
					cblogger.Error("Failed to retrieve VPC list: ", err)
				} else {
					cblogger.Info("Successfully retrieved VPC list!!")
					spew.Dump(result)
					cblogger.Debug(result)
					cblogger.Infof("Total list count : [%d]", len(result))
				}
				fmt.Println("\nListVPC() Test Finished")
				
			case 3:
				fmt.Println("Start GetVPC() ...")
				if vpcInfo, err := handler.GetVPC(vpcIId); err != nil {
					cblogger.Error(err)
					cblogger.Error("Failed to retrieve VPC information: ", err)
				} else {
					cblogger.Info("Successfully retrieved VPC information!!")
					spew.Dump(vpcInfo)
				}
				fmt.Println("\nGetVPC() Test Finished")


			case 4:
				fmt.Println("Start AddSubnet() ...")
				if result, err := handler.AddSubnet(vpcIId, subnetInfo); err != nil {
					cblogger.Error(err)
					cblogger.Error("Failed to add Subnet: ", err)
				} else {
					cblogger.Info("Successfully added Subnet!!")
					spew.Dump(result)
				}
				fmt.Println("\nAddSubnet() Test Finished")

			case 5:
				fmt.Println("Start RemoveSubnet() ...")
				if result, err := handler.RemoveSubnet(vpcIId, subnetIId); err != nil {
					cblogger.Error(err)
					cblogger.Error("Failed to remove Subnet: ", err)
				} else {
					cblogger.Info("Successfully removed Subnet!!")
					spew.Dump(result)
				}
				fmt.Println("\nRemoveSubnet() Test Finished")

			case 6:
				fmt.Println("Start DeleteVPC() ...")
				if result, err := handler.DeleteVPC(vpcIId); err != nil {
					cblogger.Error(err)
					cblogger.Error("Failed to delete VPC: ", err)
				} else {
					cblogger.Info("Successfully deleted VPC!!")
					spew.Dump(result)
				}
				fmt.Println("\nDeleteVPC() Test Finished")

			case 7:
				fmt.Println("Start ListIID() ...")
				result, err := handler.ListIID()
				if err != nil {
					cblogger.Error("Failed to retrieve VPC IID list: ", err)
				} else {
					cblogger.Info("Successfully retrieved VPC IID list!!")
					spew.Dump(result)
					cblogger.Debug(result)
					cblogger.Infof("Total IID list count : [%d]", len(result))
				}
				cblogger.Info("\nListIID() Test Finished")	

			case 0:
				cblogger.Infof("Exit")
				return
			}
		}
	}
}

func main() {
	cblogger.Info("NCP VPC Resource Test")

	handleVPC()
}

// handlerType: The string before "Handler" in the xxxHandler.go file in the resources folder
// (e.g.) ImageHandler.go -> "Image"
func getResourceHandler(handlerType string) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(ncpvpcdrv.NcpVpcDriver)

	config := readConfigFile()
	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			ClientId:     config.Ncp.NcpAccessKeyID,
			ClientSecret: config.Ncp.NcpSecretKey,
		},
		RegionInfo: idrv.RegionInfo{
			Region: config.Ncp.Region,
			Zone:   config.Ncp.Zone,
		},
	}

	// NOTE Just for test
	//cblogger.Info(config.Ncp.NcpAccessKeyID)
	//cblogger.Info(config.Ncp.NcpSecretKey)

	cloudConnection, errCon := cloudDriver.ConnectCloud(connectionInfo)
	if errCon != nil {
		return nil, errCon
	}

	var resourceHandler interface{}
	var err error

	switch handlerType {
	case "Image":
		resourceHandler, err = cloudConnection.CreateImageHandler()
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
	}

	if err != nil {
		return nil, err
	}
	return resourceHandler, nil
}

// Region: The name of the region to use (e.g., ap-northeast-2)
// ImageID: The AMI ID to use for creating the VM (e.g., ami-047f7b46bd6dd5d84)
// BaseName: The prefix name to use when creating multiple VMs (VMs will be created in the format "BaseName" + "_" + "number") (e.g., mcloud-barista)
// VmID: The EC2 instance ID to test the lifecycle
// InstanceType: The instance type to use when creating the VM (e.g., t2.micro)
// KeyName: The key pair name to use when creating the VM (e.g., mcloud-barista-keypair)
// MinCount: The minimum number of instances to create
// MaxCount: The maximum number of instances to create
// SubnetId: The SubnetId of the VPC where the VM will be created (e.g., subnet-cf9ccf83)
// SecurityGroupID: The security group ID to apply to the created VM (e.g., sg-0df1c209ea1915e4b)
type Config struct {
	Ncp struct {
		NcpAccessKeyID string `yaml:"ncp_access_key_id"`
		NcpSecretKey   string `yaml:"ncp_secret_key"`
		Region         string `yaml:"region"`
		Zone           string `yaml:"zone"`

		ImageID string `yaml:"image_id"`

		VmID         string `yaml:"ncp_instance_id"`
		BaseName     string `yaml:"base_name"`
		InstanceType string `yaml:"instance_type"`
		KeyName      string `yaml:"key_name"`
		MinCount     int64  `yaml:"min_count"`
		MaxCount     int64  `yaml:"max_count"`

		SubnetID        string `yaml:"subnet_id"`
		SecurityGroupID string `yaml:"security_group_id"`

		PublicIP string `yaml:"public_ip"`
	} `yaml:"ncpvpc"`
}

func readConfigFile() Config {
	// # Set Environment Value of Project Root Path
	// goPath := os.Getenv("GOPATH")
	// rootPath := goPath + "/src/github.com/cloud-barista/ncp/ncp/main"
	// cblogger.Debugf("Test Config file : [%]", rootPath+"/config/config.yaml")
	rootPath 	:= os.Getenv("CBSPIDER_ROOT")
	configPath 	:= rootPath + "/cloud-control-manager/cloud-driver/drivers/ncpvpc/main/config/config.yaml"
	cblogger.Debugf("Test Config file : [%s]", configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}
	cblogger.Info("ConfigFile Loaded ...")

	// Just for test
	cblogger.Debug(config.Ncp.NcpAccessKeyID, " ", config.Ncp.Region)

	return config
}
