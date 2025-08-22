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
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	// ncpdrv "github.com/cloud-barista/ncp/ncp"  // For local test
	ncpdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncp"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP VPC Resource Test")
	cblog.SetLevel("info")
}

// Test VMSpec
func handleNLB() {
	cblogger.Debug("Start NLBHandler Resource Test")

	ResourceHandler, err := getResourceHandler("NLB")
	if err != nil {
		panic(err)
	}

	nlbHandler := ResourceHandler.(irs.NLBHandler)

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ NLB Resource Test ]")
		cblogger.Info("1. ListNLB()")
		cblogger.Info("2. GetNLB()")
		cblogger.Info("3. CreateNLB()")
		cblogger.Info("4. DeleteNLB()")
		cblogger.Info("5. ChangeListener()")
		cblogger.Info("6. ChangeVMGroupInfo()")
		cblogger.Info("7. AddVMs()")
		cblogger.Info("8. RemoveVMs()")
		cblogger.Info("9. GetVMGroupHealthInfo()")
		cblogger.Info("10. ChangeHealthCheckerInfo()")
		cblogger.Info("12. ChangeHealthCheckerInfo()")
		cblogger.Info("11. ListIID()")
		cblogger.Info("12. Exit")
		fmt.Println("============================================================================================")

		config := readConfigFile()
		cblogger.Infof("\n # NCP VPC Region : [%s]", config.Ncp.Region)
		cblogger.Info("\n # Select Num : ")

		nlbIId := irs.IID{
			NameId: "new-lb-1",
			SystemId: "13230864",
		}

		nlbCreateReqInfo := irs.NLBInfo{
			IId: irs.IID{
				NameId: "new-nlb-1",
			},
			VpcIID: irs.IID{
				NameId: "ncp-vpc-01",
				// NameId: "ncp-vpc-01",
			},
			Listener: irs.ListenerInfo{
				Protocol: "TCP",
				Port:     "8080",
			},
			VMGroup: irs.VMGroupInfo{
				Protocol: "TCP",
				Port:     "8080",
				VMs: &[]irs.IID{
					{NameId: "ncp-vm-1"},
					// {NameId: "ncp-vm-2"},
					// {NameId: "s18431a1837f"},
				},
			},
			HealthChecker: irs.HealthCheckerInfo{
				Protocol:  "TCP",
				Port:      "8080",
				Interval:  -1,
				Timeout:   -1,
				Threshold: -1,
			},
			// HealthChecker: irs.HealthCheckerInfo{
			// 	Protocol:  "TCP",
			// 	Port:      "8080",
			// 	Interval:  30,
			// 	Timeout:   5,
			// 	Threshold: 3,
			// },
		}

		updateListener := irs.ListenerInfo{
			Protocol: "TCP",
			Port:     "8087",
		}

		updateVMGroups := irs.VMGroupInfo{
			Protocol: "TCP",
			Port:     "8087",
		}

		addVMs := []irs.IID{
			{NameId: "ncp-vm-1"},
			// {NameId: "ncp-vm-01-ccuki71jcupot8j6d8t0"},
			// {NameId: "s18431a1837f"},
		}

		removeVMs := []irs.IID{
			{NameId: "ncp-vm-1"},
			// {NameId: "s18431a1837f"},
		}
	
		updateHealthCheckerInfo := irs.HealthCheckerInfo{
			Protocol:  "HTTP",
			Port:      "8080",
			Interval:  7,
			Timeout:   5,
			Threshold: 4,
		}

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				cblogger.Info("Start ListNLB() ...")
				if list, err := nlbHandler.ListNLB(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(list)
					cblogger.Info("# Total count: ", len(list))
				}
				cblogger.Info("Finish ListNLB()")
			case 2:
				cblogger.Info("Start GetNLB() ...")
				if nlbInfo, err := nlbHandler.GetNLB(nlbIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(nlbInfo)
				}
				cblogger.Info("Finish GetNLB()")
			case 3:
				cblogger.Info("Start CreateNLB() ...")
				if nlbInfo, err := nlbHandler.CreateNLB(nlbCreateReqInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(nlbInfo)
				}
				cblogger.Info("Finish CreateNLB()")
			case 4:
				cblogger.Info("Start DeleteNLB() ...")
				if nlbStatus, err := nlbHandler.DeleteNLB(nlbIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(nlbStatus)
				}
				cblogger.Info("Finish DeleteNLB()")
			case 5:
				cblogger.Info("Start ChangeListener() ...")
				if nlbInfo, err := nlbHandler.ChangeListener(nlbIId, updateListener); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(nlbInfo)
				}
				cblogger.Info("Finish ChangeListener()")
			case 6:
				cblogger.Info("Start ChangeVMGroupInfo() ...")
				if info, err := nlbHandler.ChangeVMGroupInfo(nlbIId, updateVMGroups); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(info)
				}
				cblogger.Info("Finish ChangeVMGroupInfo()")
			case 7:
				cblogger.Info("Start AddVMs() ...")
				if info, err := nlbHandler.AddVMs(nlbIId, &addVMs); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(info)
				}
				cblogger.Info("Finish AddVMs()")
			case 8:
				cblogger.Info("Start RemoveVMs() ...")
				if result, err := nlbHandler.RemoveVMs(nlbIId, &removeVMs); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(result)
				}
				cblogger.Info("Finish RemoveVMs()")
			case 9:
				cblogger.Info("Start GetVMGroupHealthInfo() ...")
				if info, err := nlbHandler.GetVMGroupHealthInfo(nlbIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(info)
				}
				cblogger.Info("Finish GetVMGroupHealthInfo()")
			case 10:
				cblogger.Info("Start ChangeHealthCheckerInfo() ...")
				if info, err := nlbHandler.ChangeHealthCheckerInfo(nlbIId, updateHealthCheckerInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(info)
				}
				cblogger.Info("Finish ChangeHealthCheckerInfo()")

			case 11:
				cblogger.Info("Start ListIID() ...")
				result, err := nlbHandler.ListIID()
				if err != nil {
					cblogger.Error("Failed to retrieve NLB IID list: [%v]", err)
				} else {
					cblogger.Info("Successfully retrieved NLB IID list!!")
					spew.Dump(result)
					cblogger.Debug(result)
					cblogger.Infof("Total IID list count : [%d]", len(result))
				}
				cblogger.Info("\nListIID() Test Finished")

			case 12:
				cblogger.Infof("Exit")
				return
			}
		}
	}
}

func main() {
	cblogger.Info("NCP VPC Resource Test")

	handleNLB()
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
	case "NLB":
		resourceHandler, err = cloudConnection.CreateNLBHandler()
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
	cblogger.Info("ConfigFile Loaded ...")

	// Just for test
	cblogger.Debug(config.Ncp.NcpAccessKeyID, " ", config.Ncp.Region)

	return config
}
