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

func testErr() error {
	return errors.New("")
}

// Test VM Lifecycle Management (Create/Suspend/Resume/Reboot/Terminate)
func handleVM() {
	cblogger.Debug("Start VMHandler Resource Test")

	ResourceHandler, err := getResourceHandler("VM")
	if err != nil {
		panic(err)
	}

	vmHandler := ResourceHandler.(irs.VMHandler)

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ VM Management Test ]")
		fmt.Println("1. Start(Create) VM")
		fmt.Println("2. Get VM Info")
		fmt.Println("3. Suspend VM")
		fmt.Println("4. Resume VM")
		fmt.Println("5. Reboot VM")

		fmt.Println("6. Terminate VM")
		fmt.Println("7. Get VMStatus")
		fmt.Println("8. List VMStatus")
		fmt.Println("9. List VM")
		fmt.Println("0. Exit")
		fmt.Println("\n   Select a number above!! : ")
		fmt.Println("============================================================================================")

		//config := readConfigFile()
		VmID := irs.IID{SystemId: "db02bc57-d481-42e8-bf43-738becb14d03"}

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)

		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return

			case 1:
				vmReqInfo := irs.VMReqInfo{
					// ImageType:	irs.MyImage,
					ImageType:	irs.PublicImage,

					VMUserPasswd: "cb-user-cb-user",

					IId: irs.IID{NameId: "kt-win-vm-10"},
					// IId: irs.IID{NameId: "kt-win-vm-02"},

					// # Zone: KOR-Central A
					ImageIID: irs.IID{NameId: "WIN 2019 STD 64bit [Korean]", SystemId: "f22c7425-81b5-4cd8-b8e8-6e525070cf19"},
					VMSpecName: "d3530ad2-462b-43ad-97d5-e1087b952b7d!097b63d7-e725-4db7-b4dd-a893b0c76cb0_disk100GB",				
					// WIN 2019 STD 64bit [Korean] image와 호환

					// # Zone: KOR-Central A
					// ImageIID: irs.IID{NameId: "Ubuntu 20.04 64bit", SystemId: "87838094-af4f-449f-a2f4-f5b4b581eb29"},
					// VMSpecName: "d3530ad2-462b-43ad-97d5-e1087b952b7d!_disk20GB",
					// Ubuntu 20.04 64bit image와 호환

					// VMSpecName: "543b1f26-eddf-4521-9cbd-f3744aa2cc52!cc85e4dd-bfd9-4cec-aa22-cf226c1da92f_disk100GB",
							
					// # Zone: KOR-Seoul M2
					// ImageIID: irs.IID{NameId: "Ubuntu 20.04 64bit", SystemId: "23bc4025-8a16-4ebf-aa49-3160ee2ac24b"},

					// # Zone: KOR-Seoul M2
					// VMSpecName: "df5e0f9d-b19e-456a-ab1f-7c19c3b737f3!_disk20GB",
					// Ubuntu 20.04 이미지와 호환

					// # Zone: KOR-HA
					// ImageIID: irs.IID{NameId: "Centos 7.6 64bit", SystemId: "cfb1834b-14d9-42fc-84e6-3018dbcece71"},

					// # Zone: KOR-HA
					// VMSpecName: "91884f5a-8d72-4bdc-a76d-8526d95b6f40!_disk20GB",
					// Centos 7.6 64bit 이미지와 호환

					// # Zone: M2 (Seoul-2)
					//ImageIID:   irs.IID{NameId: "Ubuntu-18.04-64bit", SystemId: "63de6d04-7f1b-4924-8e95-1acd6581ca0c"},
					
					// # Zone: M2 (Seoul-2)
					//VMSpecName: "c308f760-068a-4cdd-abc9-edb581d18e58", //4 vCore, 8 GB
					//######################################################################

					KeyPairIID: irs.IID{SystemId: "kt-key-15"},

					VpcIID: irs.IID{
						NameId: "myTest-vpc-01",
					},					
					SubnetIID: irs.IID{
						NameId: "myTest-subnet-01",
					},

					SecurityGroupIIDs: []irs.IID{{SystemId: "KT-SG-1"},},
					// SecurityGroupIIDs: []irs.IID{{SystemId: "CB-Security4"},{SystemId: "CB-Security5"}},
					// SecurityGroupIIDs: []irs.IID{{SystemId: "CB-Security5"},{SystemId: "CB-Security6"}},

					// KT Cloud Disk(diskofferingid 지정하지 않을때) : Default : 20 GB
					RootDiskType: "SSD-Provisioned",
					// RootDiskType: "default",

					RootDiskSize: "200",
					// RootDiskSize: "default",
				}

				vmInfo, err := vmHandler.StartVM(vmReqInfo)
				if err != nil {
					//panic(err)
					cblogger.Error(err)
					cblogger.Info("VM 생성 실패 : ", err)
				} else {
					cblogger.Info("VM 생성 완료!!", vmInfo)
					spew.Dump(vmInfo)
				}
				//cblogger.Info(vm)

				cblogger.Info("\nCreateVM Test Finished")

			case 2:
				vmInfo, err := vmHandler.GetVM(VmID)
				if err != nil {
					cblogger.Error(err)
					cblogger.Errorf("[%s] VM info. 조회 실패 : ", err)
				} else {
					cblogger.Infof("[%s] VM info. 조회 결과", VmID)
					cblogger.Info(vmInfo)
					spew.Dump(vmInfo)
				}

				cblogger.Info("\nGetVM Test Finished")

			case 3:
				cblogger.Info("Start Suspend the VM ...")
				result, err := vmHandler.SuspendVM(VmID)
				if err != nil {
					cblogger.Error(err)
					cblogger.Errorf("[%s] VM Suspend 실패 : [%s]", VmID, result)
				} else {
					cblogger.Infof("[%s] VM Suspend 실행 성공 : [%s]", VmID, result)
				}

				cblogger.Info("\nSuspendVM Test Finished")

			case 4:
				cblogger.Info("Start Resume the VM ...")
				result, err := vmHandler.ResumeVM(VmID)
				if err != nil {
					cblogger.Error(err)
					cblogger.Errorf("[%s] VM Resume 실패 : [%s]", VmID, result)
				} else {
					cblogger.Infof("[%s] VM Resume 실행 성공 : [%s]", VmID, result)
				}

				cblogger.Info("\nResumeVM Test Finished")

			case 5:
				cblogger.Info("Start Reboot the VM ...")
				result, err := vmHandler.RebootVM(VmID)
				if err != nil {
					cblogger.Error(err)
					cblogger.Errorf("[%s] VM Reboot 실패 : [%s]", VmID, result)
				} else {
					cblogger.Infof("[%s] VM Reboot 실행 성공 : [%s]", VmID, result)
				}

				cblogger.Info("\nRebootVM Test Finished")

			case 6:
				cblogger.Info("Start Terminate  VM ...")
				result, err := vmHandler.TerminateVM(VmID)
				if err != nil {
					cblogger.Error(err)
					cblogger.Errorf("[%s] Terminate VM 실패 : [%s]", VmID, result)
				} else {
					cblogger.Infof("[%s] Terminate VM 실행 성공 : [%s]", VmID, result)
				}

				cblogger.Info("\nTerminateVM Test Finished")

			case 7:
				cblogger.Info("Start Get VM Status...")
				vmStatus, err := vmHandler.GetVMStatus(VmID)
				if err != nil {
					cblogger.Error(err)
					cblogger.Errorf("[%s] Get VM Status 실패 : ", VmID)
				} else {
					cblogger.Infof("[%s] Get VM Status 실행 성공 : [%s]", VmID, vmStatus)
				}

				cblogger.Info("\nGet VMStatus Test Finished")

			case 8:
				cblogger.Info("Start ListVMStatus ...")
				vmStatusInfos, err := vmHandler.ListVMStatus()
				if err != nil {
					cblogger.Error(err)
					cblogger.Error("ListVMStatus 실패 : ")
				} else {
					cblogger.Info("ListVMStatus 실행 성공")
					//cblogger.Info(vmStatusInfos)
					spew.Dump(vmStatusInfos)
				}

				cblogger.Info("\nListVM Status Test Finished")

			case 9:
				cblogger.Info("Start ListVM ...")
				vmList, err := vmHandler.ListVM()
				if err != nil {
					cblogger.Error(err)
					cblogger.Error("ListVM 실패 : ", err)
				} else {
					cblogger.Info("ListVM 실행 성공")
					cblogger.Info("=========== VM 목록 ================")
					// cblogger.Info(vmList)
					spew.Dump(vmList)
					cblogger.Infof("=========== VM 목록 수 : [%d] ================", len(vmList))
					if len(vmList) > 0 {
						VmID = vmList[0].IId
					}
				}

				cblogger.Info("\nListVM Test Finished")

			}
		}
	}
}

func main() {
	cblogger.Info("KT Cloud Resource Test")
	handleVM()
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

//환경 설정 파일 읽기
//환경변수 CBSPIDER_PATH 설정 후 해당 폴더 하위에 /config/config.yaml 파일 생성해야 함.
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
