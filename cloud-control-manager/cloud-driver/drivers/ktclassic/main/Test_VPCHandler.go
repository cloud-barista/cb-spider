// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// by ETRI, 2020.09.

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
	ktdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ktclassic"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("KT Cloud Resource Test")
	cblog.SetLevel("info")
}

// Test VMSpec
func handleVPC() {
	cblogger.Debug("Start VPCHandler Resource Test")

	ResourceHandler, err := getResourceHandler("VPC")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.VPCHandler)

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ VPC Resource Test ]")
		fmt.Println("1. CreateVPC()")
		fmt.Println("2. ListVPC()")
		fmt.Println("3. GetVPC()")
		fmt.Println("4. DeleteVPC()")
		fmt.Println("5. ListIID()")
		fmt.Println("0. Exit")
		fmt.Println("\n   Select a number above!! : ")
		fmt.Println("============================================================================================")

		vpcReqName := "ohvpc-02"
		cblogger.Info("reqVPCName : ", vpcReqName)
		
		subnetReqName := "myTest-subnet-01"

		vpcIId := irs.IID{NameId: vpcReqName}

		var subnetInfoList []irs.SubnetInfo
			info := irs.SubnetInfo{
				IId: irs.IID{
					NameId: subnetReqName,
				},
				IPv4_CIDR: "172.27.0.0/24",
			}
			subnetInfoList = append(subnetInfoList, info)

		// vpcReqInfo := irs.VPCReqInfo{
		// 	IId: irs.IID{NameId: reqVPCName, SystemId: vpcId},
		// }
		
		vpcReqInfo := irs.VPCReqInfo{
		IId:            vpcIId,
		IPv4_CIDR:      "172.27.0.0/16",
		SubnetInfoList: subnetInfoList,
		}

		var commandNum int

		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				fmt.Println("Start CreateVPC() ...")

				vpcInfo, err := handler.CreateVPC(vpcReqInfo)
				if err != nil {
					//panic(err)
					cblogger.Error(err)
					cblogger.Error("VPC 생성 실패 : ", err)
				} else {
					cblogger.Info("VPC 생성 완료!!", vpcInfo)
					spew.Dump(vpcInfo)
					cblogger.Debug(vpcInfo)
				}

				fmt.Println("\nCreateVPC() Test Finished")

			case 2:
				fmt.Println("Start ListVPC() ...")
				result, err := handler.ListVPC()
				if err != nil {
					cblogger.Error(err)
					cblogger.Error("VPC list 조회 실패 : ", err)
				} else {
					cblogger.Info("VPC list 조회 성공!!")
					spew.Dump(result)
					cblogger.Debug(result)
					cblogger.Infof("전체 list 개수 : [%d]", len(result))
				}

				fmt.Println("\nListVMSpec() Test Finished")
				
			case 3:
				fmt.Println("Start GetVPC() ...")
				if vpcInfo, err := handler.GetVPC(vpcIId); err != nil {
					cblogger.Error(err)
					cblogger.Error("VPC 정보 조회 실패 : ", err)
				} else {
					cblogger.Info("VPC 정보 조회 성공!!")
					spew.Dump(vpcInfo)
				}
				fmt.Println("\nGetVPC() Test Finished")

			case 4:
				fmt.Println("Start DeleteVPC() ...")
				if result, err := handler.DeleteVPC(vpcIId); err != nil {
					cblogger.Error(err)
					cblogger.Error("VPC 삭제 실패 : ", err)
				} else {
					cblogger.Info("VPC 삭제 성공!!")
					spew.Dump(result)
				}
				fmt.Println("\nGetVPC() Test Finished")

				
			case 5:
				cblogger.Info("Start ListIID() ...")
				result, err := handler.ListIID()
				if err != nil {
					cblogger.Error("Failed to retrieve VPC IID list: ", err)
				} else {
					cblogger.Info("Successfully retrieved VPC IID list!!")
					spew.Dump(result)
					cblogger.Debug(result)
					cblogger.Infof("Total IID list count : [%d]", len(result))
				}
				cblogger.Info("\nListIID() Test Finished")

			case 0:
				fmt.Println("Exit")
				return
			}
		}
	}
}

func main() {
	cblogger.Info("KT Cloud Resource Test")
	handleVPC()
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
