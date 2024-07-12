// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// by ETRI, 2024.07.

package main

import (
	"os"
	"errors"
	"fmt"
	// "bufio"
	// "strings"
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

func handleTag() {
	cblogger.Debug("Start Disk Resource Test")

	resourceHandler, err := getResourceHandler("Tag")
	if err != nil {
		cblogger.Error(err)
		return
	}
	tagHandler := resourceHandler.(irs.TagHandler)
	//config := readConfigFile()

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ TagHandler Test ]")
		cblogger.Info("1. ListTag()")
		cblogger.Info("2. GetTag()")
		cblogger.Info("3. AddTag()")
		cblogger.Info("4. RemoveTag()")
		cblogger.Info("5. FindTag()")
		cblogger.Info("0. Exit")
		fmt.Println("\n   Select a number above!! : ")
		fmt.Println("============================================================================================")

		var commandNum int

		rsType := irs.RSType(irs.VM)

		rsIId := irs.IID{
			NameId: "MyVM-1",
			SystemId: "25453063",
		}

		tagKV := irs.KeyValue{
			Key: 	"VPCNameTest", 
			Value: 	"NCP-myVPC-000001",
		}

		tagKey := "VPCNameTest"

		keyword := "VPC"

		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return
			case 1:
				cblogger.Info("Start ListTag() ...")
				if tagList, err := tagHandler.ListTag(rsType, rsIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(tagList)
				}
				cblogger.Info("Finish ListTag()")
			case 2:
				cblogger.Info("Start GetTag() ...")
				if tagKeyValue, err := tagHandler.GetTag(rsType, rsIId, tagKey); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(tagKeyValue)
				}
				cblogger.Info("Finish GetTag()")
			case 3:
				cblogger.Info("Start AddTag() ...")
				if tagKeyValue, err := tagHandler.AddTag(rsType, rsIId, tagKV); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(tagKeyValue)
				}
				cblogger.Info("Finish AddTag()")			
			case 4:
				cblogger.Info("Start RemoveTag() ...")				
				if tagKeyValue, err := tagHandler.RemoveTag(rsType, rsIId, tagKey); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(tagKeyValue)
				}
				cblogger.Info("Finish RemoveTag()")
			case 5:
				cblogger.Info("Start FindTag() ...")
				if tagKeyValue, err := tagHandler.FindTag(rsType, keyword); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(tagKeyValue)
				}
				cblogger.Info("Finish FindTag()")	
			}
		}
	}
}

func testErr() error {

	return errors.New("")
}

func main() {
	cblogger.Info("NCP Resource Test")
	handleTag()
}

//handlerType : resources폴더의 xxxHandler.go에서 Handler이전까지의 문자열
//(예) ImageHandler.go -> "Image"
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
	case "Tag":
		resourceHandler, err = cloudConnection.CreateTagHandler()
	}

	if err != nil {
		return nil, err
	}
	return resourceHandler, nil
}

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
