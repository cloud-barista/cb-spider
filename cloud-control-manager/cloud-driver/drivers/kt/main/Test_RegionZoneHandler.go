// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// by ETRI, 2023.10.

package main

import (
	"os"
	"fmt"

	ktvpcdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/kt"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	cblog "github.com/cloud-barista/cb-log"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NHN Cloud Resource Test")
	cblog.SetLevel("info")
}

// Test RegionZone
func handleRegionZone() {
	cblogger.Debug("Start RegionZoneHandler Resource Test")

	ResourceHandler, err := getResourceHandler("RegionZone")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.RegionZoneHandler)

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ RegionZone Resource Test ]")
		fmt.Println("1. ListRegionZone()")
		fmt.Println("2. GetRegionZone()")
		fmt.Println("3. ListOrgRegion()")
		fmt.Println("4. ListOrgZone()")
		fmt.Println("0. Exit")
		fmt.Println("\n   Select a number above!! : ")
		fmt.Println("============================================================================================")

		config := readConfigFile()
		reqRegion := config.KT.Region // Region Code Ex) KR, HK, SGN, JPN, DEN, USWN
		cblogger.Info("config.KT.Region : ", reqRegion)

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				fmt.Println("Start ListRegionZone() ...")
				result, err := handler.ListRegionZone()
				if err != nil {
					cblogger.Error("RegionZone list 조회 실패 : ", err)
				} else {
					fmt.Println("\n==================================================================================================")
					cblogger.Debug("RegionZone list 조회 성공!!")
					spew.Dump(result)
					cblogger.Debug(result)
					cblogger.Infof("RegionZone list 개수 : [%d]", len(result))
				}
				fmt.Println("\n# ListRegionZone() Test Finished")

			case 2:
				fmt.Println("Start ListOrgRegion() ...")
				result, err := handler.GetRegionZone(reqRegion)
				if err != nil {
					cblogger.Error("Region(Org) list 정보 조회 실패 : ", err)
				} else {
					fmt.Println("\n==================================================================================================")	
					cblogger.Debug("Region Info 조회 성공!!")
					spew.Dump(result)
					cblogger.Debug(result)
				}
				fmt.Println("\n# GetRegionZone() Test Finished")

			case 3:
				fmt.Println("Start ListOrgRegion() ...")
				result, err := handler.ListOrgRegion()
				if err != nil {
					cblogger.Error("Region(Org) list 정보 조회 실패 : ", err)
				} else {
					fmt.Println("\n==================================================================================================")	
					cblogger.Debug("Region(Org) list 조회 성공!!")
					spew.Dump(result)
					cblogger.Debug(result)
				}
				fmt.Println("\n# ListOrgRegion() Test Finished")

			case 4:
				fmt.Println("Start ListOrgZone() ...")
				result, err := handler.ListOrgZone()
				if err != nil {
					cblogger.Error("Zone(Org) list 조회 실패 : ", err)
				} else {
					fmt.Println("\n==================================================================================================")	
					cblogger.Debug("Zone(Org) list 조회 성공")
					spew.Dump(result)
					cblogger.Debug(result)
				}
				fmt.Println("\n# ListOrgZone() Test Finished")

			case 0:
				fmt.Println("Exit")
				return
			}
		}
	}
}

func main() {
	cblogger.Info("NHN Cloud Resource Test")

	handleRegionZone()
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
	case "NLB":
		resourceHandler, err = cloudConnection.CreateNLBHandler()
	case "RegionZone":
		resourceHandler, err = cloudConnection.CreateRegionZoneHandler()
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
