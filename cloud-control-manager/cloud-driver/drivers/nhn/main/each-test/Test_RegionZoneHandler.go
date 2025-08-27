// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// by ETRI, 2023.09.

package main

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"os"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	
	nhndrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/nhn"
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
		reqRegion := config.NhnCloud.Region // Region Code Ex) KR, HK, SGN, JPN, DEN, USWN
		cblogger.Info("config.NhnCloud.Region : ", reqRegion)

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

// handlerType : resources폴더의 xxxHandler.go에서 Handler이전까지의 문자열
// (예) ImageHandler.go -> "Image"
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

// Region : 사용할 리전명 (ex) ap-northeast-2
// ImageID : VM 생성에 사용할 AMI ID (ex) ami-047f7b46bd6dd5d84
// BaseName : 다중 VM 생성 시 사용할 Prefix이름 ("BaseName" + "_" + "숫자" 형식으로 VM을 생성 함.) (ex) mcloud-barista
// VmID : 라이프 사이트클을 테스트할 EC2 인스턴스ID
// InstanceType : VM 생성시 사용할 인스턴스 타입 (ex) t2.micro
// KeyName : VM 생성시 사용할 키페어 이름 (ex) mcloud-barista-keypair
// MinCount :
// MaxCount :
// SubnetId : VM이 생성될 VPC의 SubnetId (ex) subnet-cf9ccf83
// SecurityGroupID : 생성할 VM에 적용할 보안그룹 ID (ex) sg-0df1c209ea1915e4b
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
	rootPath := os.Getenv("CBSPIDER_ROOT")
	configPath := rootPath + "/cloud-control-manager/cloud-driver/drivers/nhn/main/conf/config.yaml"
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
