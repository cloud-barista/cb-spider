// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// by ETRI, 2024.01.

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

	ktvpcdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/kt"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("KT Cloud VPC Resource Test")
	cblog.SetLevel("info")
}

func handleMyImage() {
	cblogger.Debug("Start MyImage Resource Test")

	resourceHandler, err := getResourceHandler("MyImage")
	if err != nil {
		cblogger.Error(err)
		return
	}
	myImageHandler := resourceHandler.(irs.MyImageHandler)
	//config := readConfigFile()

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ MyImageHandler Test ]")
		fmt.Println("1. ListMyImage()")
		fmt.Println("2. GetMyImage()")
		fmt.Println("3. SnapshotVM()")
		fmt.Println("4. CheckWindowsImage()")
		fmt.Println("5. DeleteMyImage()")
		fmt.Println("6. ListIID()")
		fmt.Println("0. Exit")
		fmt.Println("\n   Select a number above!! : ")
		fmt.Println("============================================================================================")

		var commandNum int

		myImageIId := irs.IID{
			NameId: "ktvpc-disk-01",
			SystemId: "de2f41c9-7c56-4fd3-a567-d542f4174f80",
		}

		snapshotReqInfo := irs.MyImageInfo{
			IId: irs.IID{
				NameId: "ktvpc-myimage-03",
			},
			SourceVM: irs.IID{
				NameId: "kt-vm-api-29",
				SystemId: "d5d679e1-8239-43c1-9b8b-0c8d33767370",
			}, 
		}

		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return
			case 1:
				cblogger.Info("Start ListMyImage() ...")
				if listResult, err := myImageHandler.ListMyImage(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listResult)
					cblogger.Info("# My Image Count : ", len(listResult))
				}
				cblogger.Info("Finish ListMyImage()")
			case 2:
				cblogger.Info("Start GetMyImage() ...")
				if diskInfo, err := myImageHandler.GetMyImage(myImageIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(diskInfo)
				}
				cblogger.Info("Finish GetMyImage()")
			case 3:
				cblogger.Info("Start SnapshotVM() ...")
				if diskInfo, err := myImageHandler.SnapshotVM(snapshotReqInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(diskInfo)
				}
				cblogger.Info("Finish SnapshotVM()")

			case 4:
				cblogger.Info("Start CheckWindowsImage() ...")
				if checkresult, err := myImageHandler.CheckWindowsImage(myImageIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(checkresult)
				}
				cblogger.Info("Finish CheckWindowsImage()")

			case 5:
				cblogger.Info("Start DeleteMyImage() ...")
				if delResult, err := myImageHandler.DeleteMyImage(myImageIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(delResult)
				}
				cblogger.Info("Finish DeleteMyImage()")

			case 6:
				cblogger.Info("Start ListIID() ...")
				result, err := myImageHandler.ListIID()
				if err != nil {
					cblogger.Error("Failed to Get MyImage IID list : ", err)
				} else {
					cblogger.Info("Succeeded in Getting MyImage IID list!!")
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
	cblogger.Info("KT Cloud VPC Resource Test")
	handleMyImage()
}

//handlerType : resources폴더의 xxxHandler.go에서 Handler이전까지의 문자열
//(예) ImageHandler.go -> "Image"
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
	case "MyImage":
		resourceHandler, err = cloudConnection.CreateMyImageHandler()
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

		VMId string `yaml:"vm_id"`

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
