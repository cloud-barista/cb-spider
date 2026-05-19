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


	ktvpcdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/kt"
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
			NameId: 	config.KT.Resources.FileSystem.IID.NameId,
			SystemId: 	config.KT.Resources.FileSystem.IID.SystemId,
		}

		vpcIID := irs.IID{
			NameId: 	config.KT.Resources.FileSystem.VpcIID.NameId,
			SystemId: 	config.KT.Resources.FileSystem.VpcIID.SystemId,
		}

		subnetIIDs := config.KT.Resources.FileSystem.AccessSubnetIIDs

		zoneName := config.KT.Resources.FileSystem.Zone
		
		var accessSubnetList []irs.IID
		for _, subnet := range subnetIIDs {
			accessSubnetList = append(accessSubnetList, irs.IID{
				NameId:   subnet.NameId,
				SystemId: "",
			})
		}

		createreq := irs.FileSystemInfo{
			IId:              fileNameIID,
			VpcIID:           vpcIID,
			NFSVersion:       "3.0",
			AccessSubnetList: accessSubnetList,
			CapacityGB:       500,
			Zone: 			  zoneName,
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
	fmt.Println("KT Resource Test")

	handleFileSystemInfo()
}


// handlerType : resources폴더의 xxxHandler.go에서 Handler이전까지의 문자열
// (예) ImageHandler.go -> "Image"
func getResourceHandler(handlerType string) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(ktvpcdrv.KTCloudVpcDriver)

	config := readConfigFile()
	// spew.Dump(config)

	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			IdentityEndpoint: 	  config.KT.IdentityEndpoint,
			Username:         	  config.KT.Username,
			Password:         	  config.KT.Password,
			DomainName:      	  config.KT.DomainName,
			ProjectID:        	  config.KT.ProjectID,
		},
		RegionInfo: idrv.RegionInfo{
			Region: config.KT.Region,
			Zone: 	config.KT.Zone,
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
	case "Disk":
		resourceHandler, err = cloudConnection.CreateDiskHandler()
	case "FileSystem":
		resourceHandler, err = cloudConnection.CreateFileSystemHandler()		
	}

	if err != nil {
		return nil, err
	}
	return resourceHandler, nil
}

type Config struct {
	KT struct {
		IdentityEndpoint string `yaml:"identity_endpoint"`
		Username     	 string `yaml:"username"`
		Password     	 string `yaml:"password"`
		DomainName       string `yaml:"domain_name"`
		ProjectID        string `yaml:"project_id"`
		Region           string `yaml:"region"`
		Zone             string `yaml:"zone"`

		VMName           string `yaml:"vm_name"`
		ImageId          string `yaml:"image_id"`
		VMSpecId         string `yaml:"vmspec_id"`
		NetworkId        string `yaml:"network_id"`
		SecurityGroups   string `yaml:"security_groups"`
		KeypairName      string `yaml:"keypair_name"`
				
		VMId 			 string `yaml:"vm_id"`
		DiskID 			 string `yaml:"disk_id"`
		ReqDiskName		 string `yaml:"req_disk_name"`

		Image struct {
			Name string `yaml:"name"`
		} `yaml:"image_info"`

		KeyPair struct {
			Name string `yaml:"name"`
		} `yaml:"keypair_info"`

		PublicIP struct {
			Name string `yaml:"name"`
		} `yaml:"public_info"`

		SecurityGroup struct {
			Name string `yaml:"name"`
		} `yaml:"security_group_info"`

		VirtualNetwork struct {
			Name string `yaml:"name"`
		} `yaml:"vnet_info"`

		Subnet struct {
			Id string `yaml:"id"`
		} `yaml:"subnet_info"`

		Router struct {
			Name         string `yaml:"name"`
			GateWayId    string `yaml:"gateway_id"`
			AdminStateUp bool   `yaml:"adminstatup"`
		} `yaml:"router_info"`

		Resources	struct {
			FileSystem struct {
				IID struct {
					NameId 		string `yaml:"nameId"`
					SystemId 	string `yaml:"systemId"`
				} `yaml:"IID"`
				VpcIID struct {
					NameId string `yaml:"nameId"`
					SystemId 	string `yaml:"systemId"`
				} `yaml:"VpcIID"`
				Zone string `yaml:"zone"`
				AccessSubnetIIDs []struct {
					NameId string `yaml:"nameId"`
				} `yaml:"AccessSubnetIIDs"`
			} `yaml:"filesystem"`
		} `yaml:"resources"`	
	} `yaml:"ktcloudvpc"`
}

func readConfigFile() Config {
	// Set Environment Value of Project Root Path
	rootPath := os.Getenv("CBSPIDER_ROOT")
	configPath := rootPath + "/cloud-control-manager/cloud-driver/drivers/kt/main/conf/config.yaml"
	cblogger.Debugf("Test Environment Config : [%s]", configPath)

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
