// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// by ETRI, 2025.12.

package main

import (
	"errors"
	"fmt"
	"os"

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
	cblogger = cblog.GetLogger("NCP Resource Test")
	cblog.SetLevel("info")
}

func handleFileSystemInfo() {
	cblogger.Debug("Start FileSystemHandler Resource Test")

	config := readConfigFile()
	// spew.Dump(config)

	ResourceHandler, err := getResourceHandler("FileSystem")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.FileSystemHandler)

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("Test fileSystemHandler")
		fmt.Println("1. ListFileSystem()")
		fmt.Println("2. GetFileSystem()")
		fmt.Println("3. CreateFileSystem()")
		fmt.Println("4. DeleteFileSystem()")
		fmt.Println("5. AddAccessSubnet()")
		fmt.Println("6. RemoveAccessSubnet()")
		fmt.Println("7. ListAccessSubnet")
		fmt.Println("8. ListIID()")
		fmt.Println("9. GetMetaInfo()")
		fmt.Println("10. Exit")
		fmt.Println("============================================================================================")

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		fileNameIID := irs.IID{
			NameId: config.Ncp.Resources.FileSystem.IID.NameId,
			SystemId: config.Ncp.Resources.FileSystem.IID.SystemId,
		}

		vpcIID := irs.IID{
			NameId: config.Ncp.Resources.FileSystem.VpcIID.NameId,
		}

		subnetIIDs := config.Ncp.Resources.FileSystem.AccessSubnetIIDs
		
		var accessSubnetList []irs.IID
		for _, subnet := range subnetIIDs {
			accessSubnetList = append(accessSubnetList, irs.IID{
				NameId:   subnet.NameId,
				SystemId: "",
			})
		}

		createreq := irs.FileSystemInfo{
			IId:              fileNameIID,
			VpcIID:           vpcIID, // NCP does not support
			NFSVersion:       "3.0",
			AccessSubnetList: accessSubnetList, // NCP does not support
			CapacityGB:       500,
			Encryption: 	  true,     
		}

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				fmt.Println("Start ListFileSystem() ...")
				if fileSystemList, err := handler.ListFileSystem(); err != nil {
					fmt.Println(err)
				} else {
					spew.Dump(fileSystemList)
					cblogger.Infof("Total FileSystem list num : [%d]", len(fileSystemList))
				}
				fmt.Println("Finish ListFileSystem()")
			case 2:
				fmt.Println("Start GetFileSystem() ...")
				if fileSystem, err := handler.GetFileSystem(fileNameIID); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(fileSystem)
				}
				cblogger.Info("Finish GetFileSystem()")
			case 3:
				fmt.Println("Start CreateFileSystem() ...")
				fileSystem, err := handler.CreateFileSystem(createreq)
				if err != nil {
					fmt.Println(err)
				} else {
					spew.Dump(fileSystem)
				}
				fmt.Println("Finish CreateFileSystem()")
			case 4:
				fmt.Println("Start DeleteFileSystem() ...")
				if ok, err := handler.DeleteFileSystem(fileNameIID); !ok {
					fmt.Println(err)
				}
				fmt.Println("Finish DeleteFileSystem()")
			case 5:
				fmt.Println("Start AddAccessSubnet() ...")
				fsInfo, err := handler.GetFileSystem(fileNameIID)
				if err != nil {
					cblogger.Errorf("Failed to get NAS info: %v", err)
					return
				}
				fileNameIID.SystemId = fsInfo.IId.SystemId
				for _, subnetIID := range accessSubnetList {
					fileSystem, err := handler.AddAccessSubnet(fileNameIID, subnetIID)
					if err != nil {
						cblogger.Errorf("Failed to add access subnet (%s): %v", subnetIID.NameId, err)
						continue
					}
					cblogger.Infof("Successfully added subnet %s", subnetIID.NameId)
					spew.Dump(fileSystem)
				}
				fmt.Println("Finish AddAccessSubnet()")
			case 6:
				fmt.Println("Start RemoveAccessSubnet() ...")
				fsInfo, err := handler.GetFileSystem(fileNameIID)
				if err != nil {
					cblogger.Errorf("Failed to get NAS info: %v", err)
					return
				}
				fileNameIID.SystemId = fsInfo.IId.SystemId

				for _, attachedSubnet := range fsInfo.AccessSubnetList {
					ok, err := handler.RemoveAccessSubnet(fileNameIID, attachedSubnet)
					if !ok || err != nil {
						cblogger.Errorf("Failed to remove access subnet (%s): %v", attachedSubnet.NameId, err)
						continue
					}
					cblogger.Infof("Successfully removed subnet %s", attachedSubnet.NameId)
				}
				fmt.Println("Finish RemoveAccessSubnet()")
			case 7:
				cblogger.Info("Start ListAccessSubnet() ...")
				if listSubnet, err := handler.ListAccessSubnet(fileNameIID); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listSubnet)
				}
				cblogger.Info("Finish ListAccessSubnet()")
			case 8:
				cblogger.Info("Start ListIID() ...")
				if listIID, err := handler.ListIID(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listIID)
				}
				cblogger.Info("Finish ListIID()")
			case 9:
				cblogger.Info("Start GetMetaInfo() ...")
				if listIID, err := handler.GetMetaInfo(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listIID)
				}
				cblogger.Info("Finish ListIID()")
			case 10:
				fmt.Println("Exit")
				return				
			}
		}
	}
}

func testErr() error {
	return errors.New("")
}

func main() {
	fmt.Println("NCP Resource Test")

	handleFileSystemInfo()
}

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
	//fmt.Println(config.Ncp.NcpAccessKeyID)
	//fmt.Println(config.Ncp.NcpSecretKey)

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
	case "Price":
		resourceHandler, err = cloudConnection.CreatePriceInfoHandler()
	case "FileSystem":
		resourceHandler, err = cloudConnection.CreateFileSystemHandler()		
	}

	if err != nil {
		return nil, err
	}
	return resourceHandler, nil
}

// Region: Name of the region to use
// ImageID: Image ID to use when creating the VM
// BaseName: Prefix to use when creating multiple VMs (VMs are created in the format “BaseName” + ‘_’ + “number”). (e.g., mcloud-barista)
// VmID: EC2 instance ID to use for testing the lifecycle
// InstanceType : Instance type to use when creating the VM (e.g., t2.micro)
// KeyName : Key pair name to use when creating the VM (e.g., mcloud-barista-keypair)
// MinCount :
// MaxCount :
// SubnetId : Subnet ID of the VPC where the VM will be created (e.g., subnet-cf9ccf83)
// SecurityGroupID: The security group ID to apply to the VM being created (e.g., sg-0df1c209ea1915e4b)
Translated with DeepL.com (free version)
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

		Resources	struct {
			FileSystem struct {
				IID struct {
					NameId 		string `yaml:"nameId"`
					SystemId 	string `yaml:"systemId"`
				} `yaml:"IID"`
				VpcIID struct {
					NameId string `yaml:"nameId"`
				} `yaml:"VpcIID"`
				AccessSubnetIIDs []struct {
					NameId string `yaml:"nameId"`
				} `yaml:"AccessSubnetIIDs"`
			} `yaml:"filesystem"`
		} `yaml:"resources"`
	} `yaml:"ncp"`
}

func readConfigFile() Config {
	// # Set Environment Value of Project Root Path
	// goPath := os.Getenv("GOPATH")
	// rootPath := goPath + "/src/github.com/cloud-barista/ncp/ncp/main"
	// cblogger.Debugf("Test Config file : [%]", rootPath+"/config/config.yaml")
	rootPath := os.Getenv("CBSPIDER_ROOT")
	configPath := rootPath + "/cloud-control-manager/cloud-driver/drivers/ncp/main/config/config.yaml"
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

	fmt.Println("Loaded ConfigFile...")

	// Just for test
	cblogger.Debug(config.Ncp.NcpAccessKeyID, " ", config.Ncp.Region)

	return config
}
