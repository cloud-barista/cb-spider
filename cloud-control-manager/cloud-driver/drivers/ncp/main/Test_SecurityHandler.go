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
	cblogger = cblog.GetLogger("NCP Resource Test")
	cblog.SetLevel("info")
}

func handleSecurity() {
	cblogger.Debug("Start Security Resource Test")

	ResourceHandler, err := getResourceHandler("Security")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.SecurityHandler)

	//config := readConfigFile()
	//VmID := config.Ncp.VmID

	for {
		cblogger.Info("\n============================================================================================")
		cblogger.Info("[ Security Management Test ]")

		// Related APIs on NCP classic services are not supported.
		cblogger.Info("1. Create Security")
		cblogger.Info("2. List Security")
		cblogger.Info("3. Get Security")
		cblogger.Info("4. Delete Security")
		cblogger.Info("5. List IID")
		cblogger.Info("0. Quit")
		cblogger.Info("\n   Select a number above!! : ")
		cblogger.Info("============================================================================================")

		var commandNum int

		securityName := "default1"
		securityId := "1333707" //NCP default S/G
		// securityId := "214436"
		// vpcId := "vpc-c0479cab"

		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return

			case 1:
				reqInfo := irs.SecurityReqInfo{
					IId: irs.IID{
						NameId: securityName,
					},
				}
				result, err := handler.CreateSecurity(reqInfo)
				if err != nil {
					cblogger.Infof("%s SecurityGroup creation failed: %v", securityName, err)
				} else {
					cblogger.Infof("[%s] SecurityGroup creation result: [%v]", securityName, result)
					spew.Dump(result)
				}
				cblogger.Info("\nCreateSecurity() Test Finished")

			case 2:
				result, err := handler.ListSecurity()
				if err != nil {
					cblogger.Infof("Failed to retrieve SecurityGroup list: ", err)
				} else {
					cblogger.Info("SecurityGroup list retrieval result")
					//cblogger.Info(result)
					spew.Dump(result)
					cblogger.Infof("=========== S/G count : [%d] ================", len(result))
					if result != nil {
						securityId = result[0].IId.SystemId // Change to the created ID for retrieval and deletion
					}
				}
				cblogger.Info("\nListSecurity() Test Finished")

			case 3:
				cblogger.Infof("[%s] SecurityGroup information retrieval test", securityId)
				result, err := handler.GetSecurity(irs.IID{SystemId: securityId})
				if err != nil {
					cblogger.Infof("Failed to retrieve SecurityGroup [%s]: %v", securityId, err)
				} else {
					cblogger.Infof("[%s] SecurityGroup retrieval result: [%v]", securityId, result)
					spew.Dump(result)
				}
				cblogger.Info("\nGetSecurity() Test Finished")

			case 4:
				cblogger.Infof("[%s] SecurityGroup deletion test", securityId)
				result, err := handler.DeleteSecurity(irs.IID{SystemId: securityId})
				if err != nil {
					cblogger.Infof("Failed to delete SecurityGroup [%s]: %v", securityId, err)
				} else {
					cblogger.Infof("[%s] SecurityGroup deletion result: [%s]", securityId, result)
				}
				cblogger.Info("\nDeleteSecurity() Test Finished")

			case 5:
				cblogger.Info("Start ListIID() ...")
				result, err := handler.ListIID()
				if err != nil {
					cblogger.Error("Failed to retrieve S/G IID list: ", err)
				} else {
					cblogger.Info("Successfully retrieved S/G IID list!!")
					spew.Dump(result)
					cblogger.Debug(result)
					cblogger.Infof("Total number of IID list: [%d]", len(result))
				}
				cblogger.Info("\nListIID() Test Finished")		
			}
		}
	}
}

func testErr() error {
	//return awserr.Error("")
	return errors.New("")
	// return ncloud.New("504", "Not found", nil)
}

func main() {
	cblogger.Info("NCP Resource Test")
	/*
		err := testErr()
		spew.Dump(err)
		if err != nil {
			cblogger.Info("Error occurred")
			awsErr, ok := err.(awserr.Error)
			spew.Dump(awsErr)
			spew.Dump(ok)
			if ok {
				if "404" == awsErr.Code() {
					cblogger.Info("404!!!")
				} else {
					cblogger.Info("Not 404")
				}
			}
		}
	*/

	handleSecurity()
}

// handlerType: The string before "Handler" in the xxxHandler.go file in the resources folder
// (e.g., ImageHandler.go -> "Image")
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
	cblogger.Info(config.Ncp.NcpAccessKeyID)
	cblogger.Info(config.Ncp.NcpSecretKey)

	cloudConnection, errCon := cloudDriver.ConnectCloud(connectionInfo)
	if errCon != nil {
		return nil, errCon
	}

	var resourceHandler interface{}
	var err error

	switch handlerType {
	case "Image":
		resourceHandler, err = cloudConnection.CreateImageHandler()
	//case "Publicip":
	//	resourceHandler, err = cloudConnection.CreatePublicIPHandler()
	case "Security":
		resourceHandler, err = cloudConnection.CreateSecurityHandler()
	case "VNetwork":
		resourceHandler, err = cloudConnection.CreateVPCHandler()
		//case "VNic":
		//	resourceHandler, err = cloudConnection.CreateVNicHandler()
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

// Region: The region to use (e.g., ap-northeast-2)
// ImageID: The AMI ID to use for VM creation (e.g., ami-047f7b46bd6dd5d84)
// BaseName: The prefix name to use when creating multiple VMs (VMs will be created in the format "BaseName" + "_" + "number") (e.g., mcloud-barista)
// VmID: The EC2 instance ID to test the lifecycle
// InstanceType: The instance type to use when creating a VM (e.g., t2.micro)
// KeyName: The key pair name to use when creating a VM (e.g., mcloud-barista-keypair)
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
