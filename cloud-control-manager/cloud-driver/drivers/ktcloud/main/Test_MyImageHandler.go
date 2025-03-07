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

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cblog "github.com/cloud-barista/cb-log"

	ktdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ktcloud"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("KT Cloud Resource Test")
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
			// NameId: "kt-myimage-03",
			SystemId: "d5a97247-0170-4e8c-8397-661cb16e07b0",
		}

		snapshotReqInfo := irs.MyImageInfo{
			IId: irs.IID{
				NameId: "kt-my-ubuntu-image-01",
			},
			SourceVM: irs.IID{
				// NameId: "kt-vm-03",
				SystemId: "340f010b-578b-47a5-bd41-4f9615505054",
			}, 
			TagList: []irs.KeyValue{
				{ Key: "Mymy", Value: "aaaAAAAA"},
				{ Key: "iMage", Value: "cccCCCCC"},
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
					cblogger.Info("# 출력 결과 수 : ", len(listResult))
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
	cblogger.Info("KT Cloud Resource Test")
	handleMyImage()
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
	case "MyImage":
		resourceHandler, err = cloudConnection.CreateMyImageHandler()
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
		Region             string `yaml:"region"`
		Zone               string `yaml:"zone"`

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
