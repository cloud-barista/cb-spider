// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// by ETRI, 2022.08.

package main

import (
	"os"
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	ktvpcdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/kt"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cblog "github.com/cloud-barista/cb-log"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("KT Cloud VPC Resource Test")
	cblog.SetLevel("info")
}

func handleImage() {
	cblogger.Debug("Start ImageHandler Resource Test")

	ResourceHandler, err := getResourceHandler("Image")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.ImageHandler)

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ Image Management Test ]")
		fmt.Println("1. Image List")
		fmt.Println("2. Image Get")
		fmt.Println("3. Image Create (TBD)")
		fmt.Println("4. Image Delete (TBD)")
		fmt.Println("0. Quit")
		fmt.Println("\n   Select a number above!! : ")
		fmt.Println("============================================================================================")

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		imageReqInfo := irs.ImageReqInfo{
			// IId: irs.IID{NameId: "d3f14f02-15b8-445e-9fb6-4cbd3f3c3387", SystemId: "d3f14f02-15b8-445e-9fb6-4cbd3f3c3387"},
			// KT Cloud VPC : ubuntu-18.04-64bit-221115

			IId: irs.IID{NameId: "c6814d96-9746-42eb-a7d3-80f31d9cd297", SystemId: "c6814d96-9746-42eb-a7d3-80f31d9cd297"},
			// KT Cloud VPC : ubuntu-18.04-64bit
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return

			case 1:
				cblogger.Infof("Image list lookup test")

				result, err := handler.ListImage()
				if err != nil {
					cblogger.Error(err)
					cblogger.Error("Image list lookup failed : ", err)
				} else {
					fmt.Println("\n==================================================================================================================")
					cblogger.Info("Image list lookup result")
					//cblogger.Info(result)
					cblogger.Info("Number of results : ", len(result))

					fmt.Println("\n")
					spew.Dump(result)

					cblogger.Info("Number of results : ", len(result))

					// For lookup and delete tests, automatically update the request ID with the first info in the list.
					if result != nil {
						imageReqInfo.IId = result[0].IId // Change to the created ID for lookup and delete
					}
				}

				cblogger.Info("\nListImage Test Finished")

			case 2:
				cblogger.Infof("[%s] Image lookup test", imageReqInfo.IId)

				result, err := handler.GetImage(imageReqInfo.IId)
				if err != nil {
					cblogger.Error("[%s] Image lookup failed : ", imageReqInfo.IId.SystemId, err)
				} else {
					fmt.Println("\n==================================================================================================================")
					cblogger.Infof("[%s] Image lookup result : \n[%s]", imageReqInfo.IId.SystemId, result)

					fmt.Println("\n")
					spew.Dump(result)
				}

				cblogger.Info("\nGetImage Test Finished")

				// case 3:
				// 	cblogger.Infof("[%s] Image create test", imageReqInfo.IId.NameId)
				// 	result, err := handler.CreateImage(imageReqInfo)
				// 	if err != nil {
				// 		cblogger.Infof(imageReqInfo.IId.NameId, " Image create failed : ", err)
				// 	} else {
				// 		cblogger.Infof("Image create result : ", result)
				// 		imageReqInfo.IId = result.IId // Change to the created ID for lookup and delete
				// 		spew.Dump(result)
				// 	}

				// case 4:
				// 	cblogger.Infof("[%s] Image delete test", imageReqInfo.IId.NameId)
				// 	result, err := handler.DeleteImage(imageReqInfo.IId)
				// 	if err != nil {
				// 		cblogger.Infof("[%s] Image delete failed : ", imageReqInfo.IId.NameId, err)
				// 	} else {
				// 		cblogger.Infof("[%s] Image delete result : [%s]", imageReqInfo.IId.NameId, result)
				// 	}
			}
		}
	}
}

func testErr() error {

	return errors.New("")
}

func main() {
	cblogger.Info("KT Cloud VPC Resource Test")

	handleImage()
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
