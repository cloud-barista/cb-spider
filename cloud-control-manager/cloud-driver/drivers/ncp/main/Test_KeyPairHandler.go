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
	"errors"
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
	cblogger = cblog.GetLogger("NCP VPC Resource Test")
	cblog.SetLevel("info")
}

func testErr() error {
	//return awserr.Error("")
	return errors.New("")
	// return ncloud.New("504", "Not Found", nil)
}

// Test KeyPair
func handleKeyPair() {
	cblogger.Debug("Start KeyPair Resource Test")

	ResourceHandler, err := getResourceHandler("KeyPair")
	if err != nil {
		panic(err)
	}
	//config := readConfigFile()
	//VmID := config.Ncp.VmID

	keyPairHandler := ResourceHandler.(irs.KeyPairHandler)

	for {
		cblogger.Info("\n============================================================================================")
		cblogger.Info("[ KeyPair Management Test ]")
		cblogger.Info("1. List KeyPair")
		cblogger.Info("2. Create KeyPair")
		cblogger.Info("3. Get KeyPair")
		cblogger.Info("4. Delete KeyPair")
		cblogger.Info("5. List IID")
		cblogger.Info("0. Quit")
		cblogger.Info("\n   Select a number above!! : ")
		cblogger.Info("============================================================================================")

		//keyPairName := config.Ncp.KeyName
		keyPairName := "NCP-keypair-05"
		var commandNum int

		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				cblogger.Infof("Exit")
				return

			case 1:
				result, err := keyPairHandler.ListKey()
				if err != nil {
					cblogger.Infof("Failed to retrieve KeyPair list: ", err)
				} else {
					cblogger.Info("KeyPair list retrieval result: ")
					spew.Dump(result)

					cblogger.Infof("=========== KeyPair list count : [%d] ================", len(result))
				}
				cblogger.Info("\nListKey Test Finished")

			case 2:
				cblogger.Infof("[%s] KeyPair Creation Test", keyPairName)
				keyPairReqInfo := irs.KeyPairReqInfo{
					IId: irs.IID{NameId: keyPairName},
				}
				result, err := keyPairHandler.CreateKey(keyPairReqInfo)
				if err != nil {
					cblogger.Infof(keyPairName, "Failed to Create KeyPair : ", err)
				} else {
					cblogger.Infof("[%s] KeyPair creation result : \n", keyPairName)
					spew.Dump(result)
				}
				cblogger.Info("\nCreateKey Test Finished")

			case 3:
				cblogger.Infof("[%s] KeyPair retrieval test", keyPairName)
				result, err := keyPairHandler.GetKey(irs.IID{NameId: keyPairName})
				if err != nil {
					cblogger.Infof("Failed to retrieve KeyPair [%s]: %v", keyPairName, err)
				} else {
					cblogger.Infof("[%s] KeyPair retrieval result : \n", keyPairName)
					spew.Dump(result)
				}
				cblogger.Info("\nGetKey Test Finished")

			case 4:
				cblogger.Infof("[%s] KeyPair deletion test", keyPairName)
				result, err := keyPairHandler.DeleteKey(irs.IID{NameId: keyPairName})
				if err != nil {
					cblogger.Infof(keyPairName, "Failed to Delete the KeyPair : ", err)
				} else {
					cblogger.Infof("[%s] KeyPair deletion result : ", keyPairName)
					spew.Dump(result)
				}
				cblogger.Info("\nDeleteKey Test Finished")

			case 5:
				cblogger.Info("Start ListIID() ...")
				result, err := keyPairHandler.ListIID()
				if err != nil {
					cblogger.Error("Failed to Get KeyPair IID list : ", err)
				} else {
					cblogger.Info("Succeeded in Getting KeyPair IID list!!")
					spew.Dump(result)
					cblogger.Debug(result)
					cblogger.Infof("Total IID list count : [%d]", len(result))
				}
				cblogger.Info("\nListIID() Test Finished")
			}
		}
	}
}

func main() {
	cblogger.Info("NCP VPC Resource Test")

	handleKeyPair()
}

// handlerType: The string before "Handler" in the xxxHandler.go file in the resources folder
// (e.g.) ImageHandler.go -> "Image"
func getResourceHandler(handlerType string) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(ncpdrv.NcpVpcDriver)

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
	case "KeyPair":
		resourceHandler, err = cloudConnection.CreateKeyPairHandler()
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

// Read the configuration file
// You need to set the CBSPIDER_PATH environment variable and create the /config/config.yaml file under that folder.
func readConfigFile() Config {
	// Set Environment Value of Project Root Path
	// rootPath := os.Getenv("CBSPIDER_PATH")
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
	//spew.Dump(config)
	//cblogger.Info(config)

	// NOTE Just for test
	//cblogger.Info(config.Ncp.NcpAccessKeyID)
	//cblogger.Info(config.Ncp.NcpSecretKey)

	// NOTE Just for test
	cblogger.Debug(config.Ncp.NcpAccessKeyID, " ", config.Ncp.Region)

	return config
}
