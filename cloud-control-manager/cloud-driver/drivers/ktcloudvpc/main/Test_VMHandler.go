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

	ktvpcdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ktcloudvpc"

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

		config := readConfigFile()
		cblogger.Info("# config.KTCloudVPC.Zone : ", config.KTCloudVPC.Zone)

		vmID := irs.IID{SystemId: config.KTCloudVPC.VMId}
		
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

					VMUserPasswd: "cbuser357505**", // No Simple PW!!

					IId: irs.IID{NameId: config.KTCloudVPC.ReqVMName},

					// MyImage
					// ImageIID: irs.IID{NameId: "ubuntu-18.04-64bit-221115", SystemId: "22f5e22d-ebaf-4ffe-a56b-7ea12a9be770"}, //

					// Public Image
					ImageIID: irs.IID{NameId: "ubuntu-20.04-64bit", SystemId: "be9a35ed-5988-49d2-b86d-2ca9b375564b"},
					// ImageIID: irs.IID{NameId: "windows-2019-std-64bit", SystemId: "0668b053-2a3c-4751-aef9-b6342d3a19c3"},
					// ImageIID: irs.IID{NameId: "ubuntu-18.04-64bit-221115", SystemId: "d3f14f02-15b8-445e-9fb6-4cbd3f3c3387"},
					// ImageIID: irs.IID{NameId: "ubuntu-18.04-64bit", SystemId: "c6814d96-9746-42eb-a7d3-80f31d9cd297"}, // ubuntu-18.04-64bit

					VMSpecName: config.KTCloudVPC.ReqVMSpec,

					RootDiskType: "HDD",
					// RootDiskType: "SSD",
					// RootDiskType: "default",

					RootDiskSize: "50",
					// RootDiskSize: "default",

					KeyPairIID: irs.IID{NameId: "kt-keypair-02"},
					// KeyPairIID: irs.IID{SystemId: "ktvpc-key-01"},

					SecurityGroupIIDs: []irs.IID{{SystemId: "ktvpc-sg-2"}},

					// $$$ Needs KT Cloud VPC VPC 'SystemId'
					VpcIID: irs.IID{
						NameId: "",
						SystemId: "60e5d9da-55cd-47be-a0d9-6cf67c54f15c",
					},
					
					// Caution!! Not Tier 'ID' but 'OsNetworkID' (among REST API parameters) to Create VM!!
					SubnetIID: irs.IID{
						// NameId: "kt-subnet-ck1f929jcuppgg7kbvig",
						SystemId: "99f7fee3-ebe0-42df-87fe-f1e48c8a52d3",

						// NameId: "kt-dx-subnet-1", // 172.25.6.1/24
						// SystemId: "908bb72a-aa50-46d1-ba7d-32d23c0d3eea", // Not 'ID' of Tier but 'OsNetworkID' of Tier to Create VM!!
					},
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
					cblogger.Infof("[%s] VM info. 조회 결과", vmID.SystemId)
					cblogger.Info(vmInfo)
					spew.Dump(vmInfo)
				}
				cblogger.Info("\nGetVM Test Finished")

			case 3:
				cblogger.Info("Start Suspend the VM ...")
				result, err := vmHandler.SuspendVM(vmID)
				if err != nil {
					cblogger.Error(err)
					cblogger.Errorf("[%s] VM Suspend 실패 : [%v]", vmID.SystemId, result)
				} else {
					cblogger.Infof("[%s] VM Suspend 실행 성공 : [%v]", vmID.SystemId, result)
				}
				cblogger.Info("\nSuspendVM Test Finished")

			case 4:
				cblogger.Info("Start Resume the VM ...")
				result, err := vmHandler.ResumeVM(vmID)
				if err != nil {
					cblogger.Error(err)
					cblogger.Errorf("[%s] VM Resume 실패 : [%s]", vmID.SystemId, result)
				} else {
					cblogger.Infof("[%s] VM Resume 실행 성공 : [%s]", vmID.SystemId, result)
				}
				cblogger.Info("\nResumeVM Test Finished")

			case 5:
				cblogger.Info("Start Reboot the VM ...")
				result, err := vmHandler.RebootVM(vmID)
				if err != nil {
					cblogger.Error(err)
					cblogger.Errorf("[%s] VM Reboot 실패 : [%s]", vmID.SystemId, result)
				} else {
					cblogger.Infof("[%s] VM Reboot 실행 성공 : [%s]", vmID.SystemId, result)
				}
				cblogger.Info("\nRebootVM Test Finished")

			case 6:
				cblogger.Info("Start Terminate  VM ...")
				result, err := vmHandler.TerminateVM(vmID)
				if err != nil {
					cblogger.Error(err)
					cblogger.Errorf("[%s] Terminate VM 실패 : [%s]", vmID.SystemId, result)
				} else {
					cblogger.Infof("[%s] Terminate VM 실행 성공 : [%s]", vmID.SystemId, result)
				}
				cblogger.Info("\nTerminateVM Test Finished")

			case 7:
				cblogger.Info("Start Get VM Status...")
				vmStatus, err := vmHandler.GetVMStatus(vmID)
				if err != nil {
					cblogger.Error(err)
					cblogger.Errorf("[%s] Get VM Status 실패 : ", vmID.SystemId)
				} else {
					cblogger.Infof("[%s] Get VM Status 실행 성공 : [%s]", vmID.SystemId, vmStatus)
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
	cblogger.Info("KT Cloud VPC Resource Test")

	handleVM()
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
			IdentityEndpoint: 	  config.KTCloudVPC.IdentityEndpoint,
			Username:         	  config.KTCloudVPC.Username,
			Password:         	  config.KTCloudVPC.Password,
			DomainName:      	  config.KTCloudVPC.DomainName,
			ProjectID:        	  config.KTCloudVPC.ProjectID,
		},
		RegionInfo: idrv.RegionInfo{
			Region: config.KTCloudVPC.Region,
			Zone: 	config.KTCloudVPC.Zone,
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
	KTCloudVPC struct {
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

		VMId 			 string `yaml:"vm_id"`
		ReqVMName 		 string `yaml:"req_vm_name"`
		ReqVMSpec 		 string `yaml:"req_vm_spec"`
		
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
	configPath := rootPath + "/cloud-control-manager/cloud-driver/drivers/ktcloudvpc/main/conf/config.yaml"
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
