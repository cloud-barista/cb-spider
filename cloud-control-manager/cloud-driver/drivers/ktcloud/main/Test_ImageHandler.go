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
	"errors"
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
			// Zone : KOR-Central A
			IId: irs.IID{NameId: "WIN 2019 STD [Korean] MSSQL 2019 ENT", SystemId: "297b2872-7ba9-4e6f-a8b7-e7f81fffc5f3-"},
			
			// Zone : KOR-Seoul M2
			// IId: irs.IID{NameId: "Ubuntu 20.04 64bit", SystemId: "23bc4025-8a16-4ebf-aa49-3160ee2ac24b"},

			// IId: irs.IID{NameId: "ubuntu-18.04, Ubuntu Server 64-bit", SystemId: "528fdf79-c57f-4f9a-b8ea-c887b9fed003"},			
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return

			case 1:
				cblogger.Infof("Image list inquiry test")

				result, err := handler.ListImage()
				if err != nil {
					cblogger.Error("Failed to Get Image list : ", err)
				} else {
					fmt.Println("\n==================================================================================================================")
					cblogger.Info("Image list inquiry Result")
					//cblogger.Info(result)
					cblogger.Info("Image Info Count : ", len(result))

					fmt.Println("\n")
					spew.Dump(result)

					cblogger.Info("Image Info Count : ", len(result))

					//조회및 삭제 테스트를 위해 리스트의 첫번째 정보의 ID를 요청ID로 자동 갱신함.
					if result != nil {
						imageReqInfo.IId = result[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
					}
				}

				cblogger.Info("\nListImage() Test Finished")

			case 2:
				cblogger.Infof("[%s] Image inquiry test with an ID", imageReqInfo.IId)

				result, err := handler.GetImage(imageReqInfo.IId)
				if err != nil {
					cblogger.Error("Failed to Get Image Info : ", err)
				} else {
					fmt.Println("\n==================================================================================================================")
					cblogger.Infof("[%s] Image Info inquiry Result : ", imageReqInfo.IId.SystemId)
					spew.Dump(result)
				}

				cblogger.Info("\nGetImage() Test Finished")

				// case 3:
				// 	cblogger.Infof("[%s] Image Creation 테스트", imageReqInfo.IId.NameId)
				// 	result, err := handler.CreateImage(imageReqInfo)
				// 	if err != nil {
				// 		cblogger.Infof(imageReqInfo.IId.NameId, " Image Creation 실패 : ", err)
				// 	} else {
				// 		cblogger.Infof("Image Creation 결과 : ", result)
				// 		imageReqInfo.IId = result.IId // 조회 및 삭제를 위해 생성된 ID로 변경
				// 		spew.Dump(result)
				// 	}

				// case 4:
				// 	cblogger.Infof("[%s] Image 삭제 테스트", imageReqInfo.IId.NameId)
				// 	result, err := handler.DeleteImage(imageReqInfo.IId)
				// 	if err != nil {
				// 		cblogger.Infof("[%s] Image 삭제 실패 : ", imageReqInfo.IId.NameId, err)
				// 	} else {
				// 		cblogger.Infof("[%s] Image 삭제 결과 : [%s]", imageReqInfo.IId.NameId, result)
				// 	}
			}
		}
	}
}

func testErr() error {
	return errors.New("")
}

func main() {
	cblogger.Info("KT Cloud Resource Test")
	handleImage()
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

	// NOTE Just for test
	//cblogger.Info(config.KtCloud.KtCloudAccessKeyID)
	//cblogger.Info(config.KtCloud.KtCloudSecretKey)

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

		ImageID string `yaml:"image_id"`

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
