// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// by ETRI, 2024.01.
// by ETRI, 2024.04.

package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	ktvpcdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/kt"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("KT Cloud Resource Test")
	cblog.SetLevel("info")
}

func handleDisk() {
	cblogger.Debug("Start Disk Resource Test")

	resourceHandler, err := getResourceHandler("Disk")
	if err != nil {
		cblogger.Error(err)
		return
	}
	diskHandler := resourceHandler.(irs.DiskHandler)

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ DiskHandler Test ]")
		fmt.Println("1. ListDisk()")
		fmt.Println("2. GetDisk()")
		fmt.Println("3. CreateDisk()")
		fmt.Println("4. DeleteDisk()")
		fmt.Println("5. ChangeDiskSize()")
		fmt.Println("6. AttachDisk()")
		fmt.Println("7. DetachDisk()")
		fmt.Println("8. ListIID()")
		fmt.Println("0. Exit")
		fmt.Println("\n   Select a number above!! : ")
		fmt.Println("============================================================================================")

		config := readConfigFile()
		cblogger.Info("# config.KT.Zone : ", config.KT.Zone)

		var commandNum int

		diskIId := irs.IID{
			NameId:   "...",
			SystemId: config.KT.DiskID,
		}

		createReqInfo := irs.DiskInfo{
			IId: irs.IID{
				NameId: config.KT.ReqDiskName,
			},
			// DiskType: "default",
			DiskType: "HDD",
			// DiskType: "SSD",
			// DiskSize: "default",
			DiskSize: "100",
		}

		vmIId := irs.IID{ // To attach disk
			NameId:   "kt-vm-01",
			SystemId: "0e0f6583-64f9-4d27-8e1d-36d4174d4a40",
		}

		newDiskSize := "200"

		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return
				
			case 1:
				cblogger.Info("Start ListDisk() ...")
				if listResult, err := diskHandler.ListDisk(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listResult)
					cblogger.Info("# Count : ", len(listResult))
				}
				cblogger.Info("Finish ListDisk()")

			case 2:
				cblogger.Info("Start GetDisk() ...")
				if diskInfo, err := diskHandler.GetDisk(diskIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(diskInfo)
				}
				cblogger.Info("Finish GetDisk()")

			case 3:
				cblogger.Info("Start CreateDisk() ...")
				if diskInfo, err := diskHandler.CreateDisk(createReqInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(diskInfo)
				}
				cblogger.Info("Finish CreateDisk()")

			case 4:
				cblogger.Info("Start DeleteDisk() ...")
				if delResult, err := diskHandler.DeleteDisk(diskIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(delResult)
				}
				cblogger.Info("Finish DeleteDisk()")				

			case 5:
				cblogger.Info("Start ChangeDiskSize() ...")
				if diskInfo, err := diskHandler.ChangeDiskSize(diskIId, newDiskSize); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(diskInfo)
				}
				cblogger.Info("Finish ChangeDiskSize()")

			case 6:
				cblogger.Info("Start AttachDisk() ...")
				if diskInfo, err := diskHandler.AttachDisk(diskIId, vmIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(diskInfo)
				}
				cblogger.Info("Finish AttachDisk()")

			case 7:
				cblogger.Info("Start DetachDisk() ...")
				if result, err := diskHandler.DetachDisk(diskIId, vmIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(result)
				}
				cblogger.Info("Finish DetachDisk()")

			case 8:
				cblogger.Info("Start ListIID() ...")
				result, err := diskHandler.ListIID()
				if err != nil {
					cblogger.Error("Failed to Get Disk IID list : ", err)
				} else {
					cblogger.Info("Succeeded in Getting Disk IID list!!")
					spew.Dump(result)
					cblogger.Debug(result)
					cblogger.Infof("Total IID list count : [%d]", len(result))
				}
				cblogger.Info("\nListIID() Test Finished")
			}
		}
	}
}

func testErr() error {
	return errors.New("")
}

func main() {
	cblogger.Info("KT Cloud Resource Test")
	handleDisk()
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
	} `yaml:"ktcloudvpc"`
}

func readConfigFile() Config {
	// Set Environment Value of Project Root Path
	rootPath := os.Getenv("CBSPIDER_ROOT")
	configPath := rootPath + "/cloud-control-manager/cloud-driver/drivers/ktcloudvpc/main/conf/config.yaml"
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
