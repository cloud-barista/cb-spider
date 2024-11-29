// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// by ETRI, 2020.09.
// Updated by ETRI, 2024.11.

package main

import (
	"os"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cblog "github.com/cloud-barista/cb-log"

	// ncpdrv "github.com/cloud-barista/ncp/ncp"  // For local test
	ncpdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncp"	
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP Resource Test")
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
		cblogger.Info("\n============================================================================================")
		cblogger.Info("[ VPC Resource Test ]")
		cblogger.Info("1. CreateVPC()")
		cblogger.Info("2. ListVPC()")
		cblogger.Info("3. GetVPC()")
		cblogger.Info("4. DeleteVPC()")
		cblogger.Info("5. ListIID()")
		cblogger.Info("0. Exit")
		cblogger.Info("\n   Select a number above!! : ")
		cblogger.Info("============================================================================================")

		subnetReqName := "myTest-subnet-03"
		vpcIId := irs.IID{NameId: "ncp-vpc-0-cjo2smhjcupp70i7acu0"}

		var subnetInfoList []irs.SubnetInfo
		subnetInfo := irs.SubnetInfo{
			IId: irs.IID{
				NameId: subnetReqName,
			},
			IPv4_CIDR: "10.0.0.0/24",
		}
		subnetInfoList = append(subnetInfoList, subnetInfo)

		// vpcReqInfo := irs.VPCReqInfo{
		// 	IId: irs.IID{NameId: reqVPCName, SystemId: vpcId},
		// }
		
		vpcReqInfo := irs.VPCReqInfo{
			IId:            vpcIId,
			IPv4_CIDR:      "10.0.0.0/16",
			SubnetInfoList: subnetInfoList,
		}

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				cblogger.Info("Start CreateVPC() ...")
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
				cblogger.Info("\nCreateVPC() Test Finished")

			case 2:
				cblogger.Info("Start ListVPC() ...")
				result, err := handler.ListVPC()
				if err != nil {
					cblogger.Error("Failed to retrieve VPC list: ", err)
				} else {
					cblogger.Info("Successfully retrieved VPC list!!")
					spew.Dump(result)
					cblogger.Debug(result)
					cblogger.Infof("Total list count : [%d]", len(result))
				}
				cblogger.Info("\nListVPC() Test Finished")
				
			case 3:
				cblogger.Info("Start GetVPC() ...")
				if vpcInfo, err := handler.GetVPC(vpcIId); err != nil {
					cblogger.Error(err)
					cblogger.Error("Failed to retrieve VPC information: ", err)
				} else {
					cblogger.Info("Successfully retrieved VPC information!!")
					spew.Dump(vpcInfo)
				}
				cblogger.Info("\nGetVPC() Test Finished")

			case 4:
				cblogger.Info("Start DeleteVPC() ...")
				if result, err := handler.DeleteVPC(vpcIId); err != nil {
					cblogger.Error(err)
					cblogger.Error("Failed to delete VPC: ", err)
				} else {
					cblogger.Info("Successfully deleted VPC!!")
					spew.Dump(result)
				}
				cblogger.Info("\nDeleteVPC() Test Finished")

			case 5:
				cblogger.Info("Start ListIID() ...")
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
	cblogger.Info("NCP Resource Test")

	handleVPC()
}

// handlerType: The string before "Handler" in the xxxHandler.go file in the resources folder
// (e.g.) ImageHandler.go -> "Image"
func getResourceHandler(handlerType string) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(ncpdrv.NcpDriver)

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
	} `yaml:"ncp"`
}

func readConfigFile() Config {
	// # Set Environment Value of Project Root Path
	// goPath := os.Getenv("GOPATH")
	// rootPath := goPath + "/src/github.com/cloud-barista/ncp/ncp/main"
	// cblogger.Debugf("Test Config file : [%]", rootPath+"/config/config.yaml")
	rootPath 	:= os.Getenv("CBSPIDER_ROOT")
	configPath 	:= rootPath + "/cloud-control-manager/cloud-driver/drivers/ncp/main/config/config.yaml"
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

	cblogger.Info("Loaded ConfigFile...")

	// Just for test
	cblogger.Debug(config.Ncp.NcpAccessKeyID, " ", config.Ncp.Region)

	return config
}
