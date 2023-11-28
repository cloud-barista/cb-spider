// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// by ETRI, 2021.05.

package main

import (
	"os"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cblog "github.com/cloud-barista/cb-log"

	// ktdrv "github.com/cloud-barista/ktcloud/ktcloud"
	ktdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ktcloud"	
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("KT Cloud Resource Test")
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


		reqVMSpec := "df5e0f9d-b19e-456a-ab1f-7c19c3b737f3!_disk20GB" //Zone: KOR-Seoul M2
		// Ubuntu 20.04 이미지와 호환

		// reqVMSpec := "d3530ad2-462b-43ad-97d5-e1087b952b7d!87c0a6f6-c684-4fbe-a393-d8412bcf788d_disk100GB" //KOR-Cheonan(KOR-Central-B)
		//reqVMSpec := "d3530ad2-462b-43ad-97d5-e1087b952b7d!_disk20GB" //KOR-Cheonan(KOR-Central-B)

		config := readConfigFile()

		//reqRegion := config.KtCloud.Region

		cblogger.Info("config.KtCloud.Region : ", config.KtCloud.Region)
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
					cblogger.Error(err)
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
					cblogger.Error(err)
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
	cblogger.Info("KT Cloud Resource Test")
	handleVMSpec()
}

//handlerType : resources폴더의 xxxHandler.go에서 Handler이전까지의 문자열
//(예) ImageHandler.go -> "Image"
func getResourceHandler(handlerType string) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(ktdrv.KtCloudDriver)

	config := readConfigFile()
	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			ClientId:     config.KtCloud.KtCloudAccessKeyID,
			ClientSecret: config.KtCloud.KtCloudSecretKey,
		},
		RegionInfo: idrv.RegionInfo{
			Region: config.KtCloud.Region,
			Zone:   config.KtCloud.Zone,
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
	KtCloud struct {
		KtCloudAccessKeyID string `yaml:"ktcloud_access_key_id"`
		KtCloudSecretKey   string `yaml:"ktcloud_secret_key"`
		Region         string `yaml:"region"`
		Zone           string `yaml:"zone"`
		ImageID 	 string `yaml:"image_id"`
		VmID         string `yaml:"ktcloud_instance_id"`
		BaseName     string `yaml:"base_name"`
		InstanceType string `yaml:"instance_type"`
		KeyName      string `yaml:"key_name"`
		MinCount     int64  `yaml:"min_count"`
		MaxCount     int64  `yaml:"max_count"`

		SubnetID        string `yaml:"subnet_id"`
		SecurityGroupID string `yaml:"security_group_id"`

		PublicIP string `yaml:"public_ip"`
	} `yaml:"ktcloud"`
}

// 환경변수 CBSPIDER_PATH 설정 후 해당 폴더 하위에 /config/config.yaml 파일 생성해야 함.
func readConfigFile() Config {
	// Set Environment Value of Project Root Path
	rootPath := os.Getenv("CBSPIDER_ROOT")
	configPath := rootPath + "/cloud-control-manager/cloud-driver/drivers/ktcloud/main/config/config.yaml"
	cblogger.Debugf("Test Data 설정파일 : [%s]", configPath)

	data, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}

	cblogger.Info("Loaded ConfigFile...")
	//spew.Dump(config)
	//cblogger.Info(config)

	// NOTE Just for test
	//cblogger.Info(config.KtCloud.KtCloudAccessKeyID)
	//cblogger.Info(config.KtCloud.KtCloudSecretKey)

	// NOTE Just for test
	cblogger.Debug(config.KtCloud.KtCloudAccessKeyID, " ", config.KtCloud.Region)

	return config
}
