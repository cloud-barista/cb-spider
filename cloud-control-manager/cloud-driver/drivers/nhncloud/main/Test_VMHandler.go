// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// by ETRI, 2021.12.
// by ETRI, 2022.03. updated

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

	// nhndrv "github.com/cloud-barista/nhncloud/nhncloud"
	nhndrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/nhncloud"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NHN Cloud Resource Test")
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
		vmID := irs.IID{SystemId: "b16c6fc6-e8f7-499e-b2c5-04c3414f5004"}

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
					IId: irs.IID{NameId: "nhn-vm-03-WinSvr2022"},

					// $$$ Needs NHN Cloud VPC 'SystemId'
					VpcIID: irs.IID{
						NameId: "Default Network",
						SystemId: "7e3af4cc-407b-47f6-beda-7c161ebe56f0", 
					},
					
					SubnetIID: irs.IID{
						NameId: "Default Network",
						SystemId: "d73487de-8986-4d3d-9887-986057f7cd7b", 
					},
					
					SecurityGroupIIDs: []irs.IID{{SystemId: "9f0a43a9-8c73-45ce-9aac-e9c226e99533"}},
					// SecurityGroupIIDs: []irs.IID{{SystemId: "79965cd0-b9e9-42ef-9c66-201f824273cb"},{SystemId: "67167e2e-2390-48d6-8f27-78c9293b26f3"}},

					KeyPairIID: irs.IID{NameId: "nhn-pub-a-keypair-01-cmbrbe9jcups9eecbdfg"},			
					// KeyPairIID: irs.IID{NameId: "nhn-key-01-c9584r9jcupvtimg81l0"},					
					// KeyPairIID: irs.IID{SystemId: "nhn-key-01"},

					//KR1					
					// ImageIID: irs.IID{NameId: "Ubuntu Server 22.04.3 LTS (2023.11.21)", SystemId: "68fb7e1a-191c-45bf-8328-b7864f9fbaf4"},

					// ImageIID: irs.IID{NameId: "Debian 11.8 Bullseye (2023.11.21)", SystemId: "7753643d-f0ea-4260-a2c0-5ead4e041725"},

					ImageIID: irs.IID{NameId: "Windows 2022 STD (2023.11.21) KO", SystemId: "4b2b5dfb-d325-4153-8a88-eb39cdb181c9"},
					
					// ImageIID: irs.IID{NameId: "Ubuntu Server 18.04.6 LTS (2021.12.21)", SystemId: "5396655e-166a-4875-80d2-ed8613aa054f"},

					// ImageIID:  irs.IID{NameId: "CentOS 6.10 (2018.10.23)", SystemId: "1c868787-6207-4ff2-a1e7-ae1331d6829b"},

					// VMSpecName: "u2.c2m4", //vCPU: 2, Mem: 4GB
					VMSpecName: "m2.c4m8", //vCPU: 4, Mem: 8GB, LocalDiskSize(GB): 0 

					RootDiskType: "General_SSD",
					// RootDiskType: "General_HDD",
					// RootDiskType: "default",

					RootDiskSize: "100", // Except for u2.~ type of VMSpec
					// RootDiskSize: "default", // When u2.~ type of VMSpec

					// DataDiskIIDs: []irs.IID{ // Disk volume list to Attach
					// 	{  
					// 	SystemId: "eface614-e6c0-40ee-8237-e1d28edb1bb4",
					// 	},
					// },
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
				vmInfo, err := vmHandler.GetVM(vmID)
				if err != nil {
					cblogger.Error(err)
					cblogger.Errorf("[%s] VM info. 조회 실패 : ", err)
				} else {
					cblogger.Infof("[%s] VM info. 조회 결과", vmID)
					cblogger.Info(vmInfo)
					spew.Dump(vmInfo)
				}

				cblogger.Info("\nGetVM Test Finished")

			case 3:
				cblogger.Info("Start Suspend the VM ...")
				result, err := vmHandler.SuspendVM(vmID)
				if err != nil {
					cblogger.Error(err)
					cblogger.Errorf("[%s] VM Suspend 실패 : [%s]", vmID, result)
				} else {
					cblogger.Infof("[%s] VM Suspend 실행 성공 : [%s]", vmID, result)
				}

				cblogger.Info("\nSuspendVM Test Finished")

			case 4:
				cblogger.Info("Start Resume the VM ...")
				result, err := vmHandler.ResumeVM(vmID)
				if err != nil {
					cblogger.Error(err)
					cblogger.Errorf("[%s] VM Resume 실패 : [%s]", vmID, result)
				} else {
					cblogger.Infof("[%s] VM Resume 실행 성공 : [%s]", vmID, result)
				}

				cblogger.Info("\nResumeVM Test Finished")

			case 5:
				cblogger.Info("Start Reboot the VM ...")
				result, err := vmHandler.RebootVM(vmID)
				if err != nil {
					cblogger.Error(err)
					cblogger.Errorf("[%s] VM Reboot 실패 : [%s]", vmID, result)
				} else {
					cblogger.Infof("[%s] VM Reboot 실행 성공 : [%s]", vmID, result)
				}

				cblogger.Info("\nRebootVM Test Finished")

			case 6:
				cblogger.Info("Start Terminate  VM ...")
				result, err := vmHandler.TerminateVM(vmID)
				if err != nil {
					cblogger.Error(err)
					cblogger.Errorf("[%s] Terminate VM 실패 : [%s]", vmID, result)
				} else {
					cblogger.Infof("[%s] Terminate VM 실행 성공 : [%s]", vmID, result)
				}

				cblogger.Info("\nTerminateVM Test Finished")

			case 7:
				cblogger.Info("Start Get VM Status...")
				vmStatus, err := vmHandler.GetVMStatus(vmID)
				if err != nil {
					cblogger.Error(err)
					cblogger.Errorf("[%s] Get VM Status 실패 : ", vmID)
				} else {
					cblogger.Infof("[%s] Get VM Status 실행 성공 : [%s]", vmID, vmStatus)
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
						vmID = vmList[0].IId
					}
				}

				cblogger.Info("\nListVM Test Finished")

			}
		}
	}
}

func main() {
	cblogger.Info("NHN Cloud Resource Test")

	handleVM()
}

//handlerType : resources폴더의 xxxHandler.go에서 Handler이전까지의 문자열
//(예) ImageHandler.go -> "Image"
func getResourceHandler(handlerType string) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(nhndrv.NhnCloudDriver)

	config := readConfigFile()
	// spew.Dump(config)

	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			IdentityEndpoint: config.NhnCloud.IdentityEndpoint,
			Username:         	  config.NhnCloud.Nhn_Username,
			Password:         	  config.NhnCloud.Api_Password,
			DomainName:      	  config.NhnCloud.DomainName,
			TenantId:        	  config.NhnCloud.TenantId,
		},
		RegionInfo: idrv.RegionInfo{
			Region: config.NhnCloud.Region,
			Zone: 	config.NhnCloud.Zone,
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
// VMID : 라이프 사이트클을 테스트할 EC2 인스턴스ID
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
		Zone           	 string `yaml:"zone"`

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
