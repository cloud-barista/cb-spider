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
	// "errors"
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

// Test VMSpec
func handleVMSpec() {
	cblogger.Debug("Start VMSpecHandler Resource Test")

	ResourceHandler, err := getResourceHandler("VMSpec")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.VMSpecHandler)

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ VMSpec Resource Test ]")
		fmt.Println("1. ListVMSpec()")
		fmt.Println("2. GetVMSpec()")
		fmt.Println("3. ListOrgVMSpec()")
		fmt.Println("4. GetOrgVMSpec()")
		fmt.Println("0. Exit")
		fmt.Println("\n   Select a number above!! : ")
		fmt.Println("============================================================================================")

		reqVMSpec := "2x8.itl" //KT Cloud VPC VMSpec
		// reqVMSpec := "32x128.itl.v100-2" //KT Cloud VPC VMSpec

		config := readConfigFile()

		cblogger.Info("config.KT.Zone : ", config.KT.Zone)
		cblogger.Info("reqVMSpec : ", reqVMSpec)

		var commandNum int

		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				fmt.Println("Start ListVMSpec() ...")

				result, err := handler.ListVMSpec()
				if err != nil {
					cblogger.Error("\nVMSpec list 조회 실패 : ", err)
				} else {
					fmt.Println("\n==================================================================================================================")
					cblogger.Debug("VMSpec list 조회 성공!!")
					spew.Dump(result)
					//cblogger.Debug(result)
					cblogger.Infof("전체 VMSpec list 개수 : [%d]", len(result))
				}

				fmt.Println("\nListVMSpec() Test Finished")

			case 2:
				fmt.Println("Start GetVMSpec() ...")

				result, err := handler.GetVMSpec(reqVMSpec)
				if err != nil {
					cblogger.Error(reqVMSpec, " VMSpec 정보 조회 실패 : ", err)
				} else {
					fmt.Println("\n==================================================================================================================")
					cblogger.Debugf("VMSpec[%s] 정보 조회 성공!!", reqVMSpec)
					spew.Dump(result)
					cblogger.Debug(result)
					//cblogger.Infof(result)
				}

				fmt.Println("\nGetVMSpec() Test Finished")

			case 3:
				fmt.Println("Start ListOrgVMSpec() ...")
				result, err := handler.ListOrgVMSpec()
				if err != nil {
					cblogger.Error(err)
					cblogger.Error("VMSpec Org list 조회 실패 : ", err)
				} else {
					cblogger.Debug("VMSpec Org list 조회 성공")
					spew.Dump(result)
					cblogger.Debug(result)
					//spew.Dump(result)
					//fmt.Println(result)
					//fmt.Println("=========================")
					//fmt.Println(result)
					cblogger.Infof("전체 목록 개수 : [%d]", len(result))
				}

				fmt.Println("\nListOrgVMSpec() Test Finished")

			case 4:
				fmt.Println("Start GetOrgVMSpec() ...")
				result, err := handler.GetOrgVMSpec(reqVMSpec)
				if err != nil {
					cblogger.Error(err)
					cblogger.Error(reqVMSpec, " VMSpec Org 정보 조회 실패 : ", err)
				} else {
					cblogger.Debugf("VMSpec[%s] Org 정보 조회 성공", reqVMSpec)
					spew.Dump(result)
					cblogger.Debug(result)
					//fmt.Println(result)
				}

				fmt.Println("\nGetOrgVMSpec() Test Finished")

			case 0:
				fmt.Println("Exit")
				return
			}
		}
	}
}

func main() {
	cblogger.Info("KT Cloud VPC Resource Test")

	handleVMSpec()
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
