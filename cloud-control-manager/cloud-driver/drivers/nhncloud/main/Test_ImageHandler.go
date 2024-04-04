// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// by ETRI, 2021.12.

package main

import (
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"os"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	// nhndrv "github.com/cloud-barista/nhncloud/nhncloud"
	nhndrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/nhncloud"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NHN Cloud Resource Test")
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
		fmt.Println("1. ListImage()")
		fmt.Println("2. GetImage()")
		fmt.Println("3. CheckWindowsImage()")
		fmt.Println("4. CreateImage (TBD)")
		fmt.Println("5. DeleteImage (TBD)")
		fmt.Println("0. Quit")
		fmt.Println("\n   Select a number above!! : ")
		fmt.Println("============================================================================================")

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		imageReqInfo := irs.ImageReqInfo{
			// IId: irs.IID{NameId: "Ubuntu Server 18.04.6 LTS (2021.12.21)", SystemId: "5396655e-166a-4875-80d2-ed8613aa054f"},
			IId: irs.IID{NameId: "Windows 2022 STD (2023.11.21) EN", SystemId: "2ffacb1d-8442-4ba1-b08e-d820a9d2ec9f"},

			//IId: irs.IID{NameId: "CentOS 6.10 (2018.10.23)", SystemId: "1c868787-6207-4ff2-a1e7-ae1331d6829b"},
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return

			case 1:
				cblogger.Infof("ListImage() Test")

				result, err := handler.ListImage()
				if err != nil {
					cblogger.Error(err)
					cblogger.Error("Failed to List Image : ", err)
				} else {
					fmt.Println("\n==================================================================================================================")
					cblogger.Info("Result of ListImage()")
					//cblogger.Info(result)
					cblogger.Info("ListImage() count : ", len(result))
					fmt.Println("\n")
					spew.Dump(result)

					if result != nil {
						imageReqInfo.IId = result[0].IId
					}
				}

				cblogger.Info("\nListImage Test Finished")

			case 2:
				cblogger.Infof("[%s]GetImage() Test", imageReqInfo.IId)

				result, err := handler.GetImage(imageReqInfo.IId)
				if err != nil {
					cblogger.Error(err)
					cblogger.Error("Failed to Get the Image Info of [%s] : ", imageReqInfo.IId.SystemId, err)
				} else {
					fmt.Println("\n==================================================================================================================")
					cblogger.Infof("Result of GetImage() of [%s] : \n[%s]", imageReqInfo.IId.SystemId, result)

					fmt.Println("\n")
					spew.Dump(result)
				}

				cblogger.Info("\nGetImage Test Finished")

			case 3:
				cblogger.Infof("[%s] CheckWindowsImage() Test", imageReqInfo.IId)

				result, err := handler.CheckWindowsImage(imageReqInfo.IId)
				if err != nil {
					cblogger.Error(err)
					cblogger.Error("Failed to CheckWindowsImage() of [%s] : ", imageReqInfo.IId.SystemId, err)
				} else {
					fmt.Println("\n==================================================================================================================")
					cblogger.Infof("Result of CheckWindowsImage() of [%s] : [%v]", imageReqInfo.IId.SystemId, result)

					fmt.Println("\n")
					spew.Dump(result)
				}

				cblogger.Info("\nCheckWindowsImage() Test Finished")
			}
		}
	}
}

func testErr() error {

	return errors.New("")
}

func main() {
	cblogger.Info("NHN Cloud Resource Test")

	handleImage()
}

// (Ex) ImageHandler.go -> "Image"
func getResourceHandler(handlerType string) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(nhndrv.NhnCloudDriver)

	config := readConfigFile()
	// spew.Dump(config)

	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			IdentityEndpoint: config.NhnCloud.IdentityEndpoint,
			Username:         config.NhnCloud.Nhn_Username,
			Password:         config.NhnCloud.Api_Password,
			DomainName:       config.NhnCloud.DomainName,
			TenantId:         config.NhnCloud.TenantId,
		},
		RegionInfo: idrv.RegionInfo{
			Region: config.NhnCloud.Region,
			Zone:   config.NhnCloud.Zone,
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
	NhnCloud struct {
		IdentityEndpoint string `yaml:"identity_endpoint"`
		Nhn_Username     string `yaml:"nhn_username"`
		Api_Password     string `yaml:"api_password"`
		DomainName       string `yaml:"domain_name"`
		TenantId         string `yaml:"tenant_id"`
		Region           string `yaml:"region"`
		Zone             string `yaml:"zone"`

		VMName         string `yaml:"vm_name"`
		ImageId        string `yaml:"image_id"`
		VMSpecId       string `yaml:"vmspec_id"`
		NetworkId      string `yaml:"network_id"`
		SecurityGroups string `yaml:"security_groups"`
		KeypairName    string `yaml:"keypair_name"`

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
	} `yaml:"nhncloud"`
}

func readConfigFile() Config {
	// Set Environment Value of Project Root Path
	// rootPath := "/home/sean/go/src/github.com/cloud-barista/nhncloud/nhncloud/main"
	rootPath := os.Getenv("CBSPIDER_ROOT")
	configPath := rootPath + "/cloud-control-manager/cloud-driver/drivers/nhncloud/main/conf/config.yaml"
	cblogger.Info("Config file : " + configPath)

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
