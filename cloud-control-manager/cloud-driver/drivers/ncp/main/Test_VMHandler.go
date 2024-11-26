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

func testErr() error {
	return errors.New("")
	// return ncloud.New("504", "Not found", nil)
}

// Test VM Lifecycle Management (Create/Suspend/Resume/Reboot/Terminate)
func handleVM() {
	cblogger.Debug("Start VMHandler Resource Test")

	config := readConfigFile()

	ResourceHandler, err := getResourceHandler("VM")
	if err != nil {
		panic(err)
	}

	vmHandler := ResourceHandler.(irs.VMHandler)

	for {
		cblogger.Info("\n============================================================================================")
		cblogger.Info("[ VM Management Test ]")
		cblogger.Info("1. Start(Create) VM")
		cblogger.Info("2. Get VM Info")
		cblogger.Info("3. Suspend VM")
		cblogger.Info("4. Resume VM")
		cblogger.Info("5. Reboot VM")
		cblogger.Info("6. Terminate VM")
		cblogger.Info("7. Get VMStatus")
		cblogger.Info("8. List VMStatus")
		cblogger.Info("9. List VM")
		cblogger.Info("10. List IID")
		cblogger.Info("0. Exit")
		cblogger.Info("\n   Select a number above!! : ")
		cblogger.Info("============================================================================================")

		//config := readConfigFile()
		VmID := irs.IID{SystemId: config.Ncp.VmID}

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)

		if err != nil {
			panic(err)
		}

		vmReqInfo := irs.VMReqInfo{
			// ImageType:	irs.MyImage,
			ImageType:	irs.PublicImage,

			// # NCP does not allow uppercase letters in VM instance names, so they are converted to lowercase in the VMHandler.
			// Caution!! : Underscore characters are not allowed.
			IId: irs.IID{NameId: "ncp-test-vm-10"},

			// Caution!!) You must set the region in /home/sean/go/src/github.com/cloud-barista/ncp/ncp/main/config/config.yaml to create a VM in that region.

			//(Reference) When Region is 'DEN'. 
			//VMSpec := "SPSVRSTAND000063"   vCPU 8EA, Memory 64GB, [SSD]Disk 50GB", 
			//Image ID : SPSW0LINUX000031

			// KR
			// ImageIID:   irs.IID{NameId: "Ubuntu Server 18.04 (64-bit)", SystemId: "SPSW0LINUX000130"}, // $$$ PublicImage $$$
			// VMSpecName: "SPSVRSTAND000006",

			// KR
			ImageIID:   irs.IID{NameId: "CentOS 7.8 (64-bit)", SystemId: "SPSW0LINUX000139"}, // $$$ PublicImage $$$
			VMSpecName: "SPSVRSTAND000005",

			// KR
			// ImageIID:   irs.IID{NameId: "Windows Server 2016 (64-bit) English Edition", SystemId: "SPSW0WINNTEN0016A"}, // $$$ PublicImage $$$
			// SPSW0WINNTEN0016A - 'Windows' Server 2016 (64-bit) English Edition
			// # VMSpecName: "SPSVRSSD00000005A" // Compatible with the above win server

			// KR
			// ImageIID:   irs.IID{NameId: "Windows Server 2012 (64bit) R2 English Edition", SystemId: "SPSW0WINNTEN0015A"},
			// VMSpecName: "SPSVRSTAND000005A",

			// KR
			// ImageIID:   irs.IID{NameId: "Windows Server (64bit)", SystemId: "96215"}, // $$$ MyImage $$$
			// VMSpecName: "SPSVRSTAND000005A",

			// USWN
			//ImageIID: irs.IID{NameId: "Ubuntu Server 18.04 (64-bit)", SystemId: "SPSW0LINUX000130"},
			//VMSpecName: "SPSVRSTAND000025",

			// USWN
			//ImageIID: irs.IID{NameId: "WordPress-Ubuntu-16.04-64", SystemId: "SPSW0LINUX000088"},
			//VMSpecName: "SPSVRSTAND000050",

			// DEN :
			//ImageIID:   irs.IID{NameId: "Ubuntu Server 18.04 (64-bit)", SystemId: "SPSW0LINUX000130"},
			//VMSpecName: "SPSVRSTAND000025",

			// DEN :
			//ImageIID:   irs.IID{NameId: "centOS-6.3-64", SystemId: "SPSW0LINUX000031"},
			//VMSpecName: "SPSVRSSD00000006",

			// JPN
			//ImageIID: irs.IID{NameId: "Ubuntu Server 18.04 (64-bit)", SystemId: "SPSW0LINUX000130"},
			//VMSpecName: "SPSVRSTAND000025",

			// SGN
			//ImageIID: irs.IID{NameId: "Ubuntu Server 18.04 (64-bit)", SystemId: "SPSW0LINUX000130"},
			//VMSpecName: "SPSVRSTAND000025",

			// HK
			//ImageIID:   irs.IID{NameId: "ubuntu-16.04", SystemId: "SPSW0LINUX000095"},
			//VMSpecName: "SPSVRSTAND000052",

			KeyPairIID: irs.IID{SystemId: "oh-keypai-cqccsj4vtts7hk9ghtmg"},
			// KeyPairIID: irs.IID{SystemId: "ncp-key-0-cjheqe9jcupqtmoaa6bg"},

			// # NCP Classic 2nd generation service does not support specifying subnet, VPC
			VpcIID:    irs.IID{SystemId: "oh-vpc-01-cqab15kvtts35l1k5c6g"},
			SubnetIID: irs.IID{SystemId: "oh-subnet-cqab15kvtts35l1k5c70"},

			// If SecurityGroupIIDs is not specified, the NCP default value "ncloud-default-acg" with "293807" is applied.
			// SecurityGroupIIDs: []irs.IID{{SystemId: "293807"},{SystemId: "332703"}},
			SecurityGroupIIDs: []irs.IID{{SystemId: "1333707"}},

			VMUserPasswd: "abcd000abcd",

			TagList: []irs.KeyValue{
				{ Key: "aaa", Value: "aaaAAAAA"},
				{ Key: "ccc", Value: "cccCCCCC"},
			},
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return

			case 1:
				vmInfo, err := vmHandler.StartVM(vmReqInfo)
				if err != nil {
					//panic(err)
					cblogger.Error(err)
				} else {
					cblogger.Info("Succeeded in VM Creation!!", vmInfo)
					spew.Dump(vmInfo)
				}
				cblogger.Info("\nCreateVM Test Finished")

			case 2:
				vmInfo, err := vmHandler.GetVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] Failed to Get VM info!!", VmID)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] Result : ", VmID)
					spew.Dump(vmInfo)
				}
				cblogger.Info("\nGetVM Test Finished")

			case 3:
				cblogger.Info("Start Suspend VM ...")
				result, err := vmHandler.SuspendVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] Failed to Suspend VM : [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] Succeeded in VM Suspend : [%s]", VmID, result)
				}
				cblogger.Info("\nSuspendVM Test Finished")

			case 4:
				cblogger.Info("Start Resume  VM ...")
				result, err := vmHandler.ResumeVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] Failed to Resume VM : [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] Succeeded in VM Resumme : [%s]", VmID, result)
				}
				cblogger.Info("\nResumeVM Test Finished")

			case 5:
				cblogger.Info("Start Reboot  VM ...")
				result, err := vmHandler.RebootVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] Failed to Reboot VM : [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] Succeeded in VM Reboot : [%s]", VmID, result)
				}
				cblogger.Info("\nRebootVM Test Finished")

			case 6:
				cblogger.Info("Start Terminate  VM ...")
				result, err := vmHandler.TerminateVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] Failed to Terminate VM : [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] Succeeded in VM Terminate : [%s]", VmID, result)
				}
				cblogger.Info("\nTerminateVM Test Finished")

			case 7:
				cblogger.Info("Start Get VM Status...")
				vmStatus, err := vmHandler.GetVMStatus(VmID)
				if err != nil {
					cblogger.Errorf("[%s] Failed to Get VM Status : ", VmID)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] Succeeded in Getting VM Status : [%s]", VmID, vmStatus)
				}
				cblogger.Info("\nGet VMStatus Test Finished")

			case 8:
				cblogger.Info("Start ListVMStatus ...")
				vmStatusInfos, err := vmHandler.ListVMStatus()
				if err != nil {
					cblogger.Error("Failed to List VMStatus")
					cblogger.Error(err)
				} else {
					cblogger.Info("Succeeded in Listing VMStatus")
					spew.Dump(vmStatusInfos)
				}
				cblogger.Info("\nListVM Status Test Finished")

			case 9:
				cblogger.Info("Start ListVM ...")
				vmList, err := vmHandler.ListVM()
				if err != nil {
					cblogger.Error("Failed to List VM")
					cblogger.Error(err)
				} else {
					cblogger.Info("Succeeded in Listing VM")
					spew.Dump(vmList)
					cblogger.Infof("=========== Count VM : [%d] ================", len(vmList))
					if len(vmList) > 0 {
						VmID = vmList[0].IId
					}
				}
				cblogger.Info("\nListVM Test Finished")

			case 10:
				cblogger.Info("Start ListIID() ...")
				result, err := vmHandler.ListIID()
				if err != nil {
					cblogger.Error("Failed to Get VM IID list : ", err)
				} else {
					cblogger.Info("Succeeded in Getting VM IID list!!")
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
	cblogger.Info("NCP Resource Test")

	handleVM()
}

// handlerType: The string before "Handler" in the xxxHandler.go file in the resources folder
// (e.g., ImageHandler.go -> "Image")
func getResourceHandler(handlerType string) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(ncpdrv.NcpDriver)

	config := readConfigFile()
	// cblogger.Info("### config :")
	// spew.Dump(config)

	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			ClientId:     config.Ncp.NcpAccessKeyID,
			ClientSecret: config.Ncp.NcpSecretKey,
		},
		RegionInfo: idrv.RegionInfo{
			Region: 	config.Ncp.Region,
			Zone:   	config.Ncp.Zone,
			TargetZone: config.Ncp.TargetZone,
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
		TargetZone     string `yaml:"target_zone"` // For Zone-based control!!

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
