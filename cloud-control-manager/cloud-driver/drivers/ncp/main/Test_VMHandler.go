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
	"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cblog "github.com/cloud-barista/cb-log"

	// ncpdrv "github.com/cloud-barista/ncp/ncp"  // For local test
	ncpdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncp"	
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP Resource Test")
	cblog.SetLevel("info")
}

func testErr() error {
	return errors.New("")
	// return ncloud.New("504", "찾을 수 없음", nil)
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
		VmID := irs.IID{SystemId: "25504672"}

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)

		if err != nil {
			panic(err)
		}

		vmReqInfo := irs.VMReqInfo{
			// ImageType:	irs.MyImage,
			ImageType:	irs.PublicImage,

			// # NCP에서는 VM instance 이름에 대문자 허용 안되므로 VMHandler 내부에서 소문자로 변환되어 반영됨.	
			// Caution!! : Under bar 문자 허용 안됨.
			IId: irs.IID{NameId: "ncp-test-vm-10"},

			// Caution!!) /home/sean/go/src/github.com/cloud-barista/ncp/ncp/main/config/config.yaml 에서 해당 region을 설정해야 그 region에 VM이 생성됨.

			//(참고) When Region is 'DEN'. 
			//VMSpec := "SPSVRSTAND000063"   vCPU 8EA, Memory 64GB, [SSD]Disk 50GB", 
			//Image ID : SPSW0LINUX000031

			// KR
			// ImageIID:   irs.IID{NameId: "Ubuntu Server 18.04 (64-bit)", SystemId: "SPSW0LINUX000130"}, // $$$ PublicImage $$$
			// VMSpecName: "SPSVRSTAND000006",

			// KR
			ImageIID:   irs.IID{NameId: "CentOS 7.8 (64-bit)", SystemId: "SPSW0LINUX000139"}, // $$$ PublicImage $$$
			VMSpecName: "SPSVRSTAND000005",

			// KR
			// ImageIID:   irs.IID{NameId: "Windows Server 2016 (64-bit) English Edition", SystemId: "SPSW0WINNTEN0016A"}, // $$$ PublicImage $$$
			// SPSW0WINNTEN0016A - 'Windows' Server 2016 (64-bit) English Edition
			// # VMSpecName: "SPSVRSSD00000005A" // 상기 win server와 호환

			// KR
			// ImageIID:   irs.IID{NameId: "Windows Server 2012 (64bit) R2 English Edition", SystemId: "SPSW0WINNTEN0015A"},
			// VMSpecName: "SPSVRSTAND000005A",

			// KR
			// ImageIID:   irs.IID{NameId: "Windows Server (64bit)", SystemId: "96215"}, // $$$ MyImage $$$
			// VMSpecName: "SPSVRSTAND000005A",

			// USWN
			//ImageIID: irs.IID{NameId: "Ubuntu Server 18.04 (64-bit)", SystemId: "SPSW0LINUX000130"},
			//VMSpecName: "SPSVRSTAND000025",

			// USWN
			//ImageIID: irs.IID{NameId: "WordPress-Ubuntu-16.04-64", SystemId: "SPSW0LINUX000088"},
			//VMSpecName: "SPSVRSTAND000050",

			// DEN :
			//ImageIID:   irs.IID{NameId: "Ubuntu Server 18.04 (64-bit)", SystemId: "SPSW0LINUX000130"},
			//VMSpecName: "SPSVRSTAND000025",

			// DEN :
			//ImageIID:   irs.IID{NameId: "centOS-6.3-64", SystemId: "SPSW0LINUX000031"},
			//VMSpecName: "SPSVRSSD00000006",

			// JPN
			//ImageIID: irs.IID{NameId: "Ubuntu Server 18.04 (64-bit)", SystemId: "SPSW0LINUX000130"},
			//VMSpecName: "SPSVRSTAND000025",

			// SGN
			//ImageIID: irs.IID{NameId: "Ubuntu Server 18.04 (64-bit)", SystemId: "SPSW0LINUX000130"},
			//VMSpecName: "SPSVRSTAND000025",

			// HK
			//ImageIID:   irs.IID{NameId: "ubuntu-16.04", SystemId: "SPSW0LINUX000095"},
			//VMSpecName: "SPSVRSTAND000052",

			KeyPairIID: irs.IID{SystemId: "oh-keypai-cqccsj4vtts7hk9ghtmg"},
			// KeyPairIID: irs.IID{SystemId: "ncp-key-0-cjheqe9jcupqtmoaa6bg"},

			// # NCP Classic 2세대 service에서 subnet, VPC 지정은 미지원
			VpcIID:    irs.IID{SystemId: "oh-vpc-01-cqab15kvtts35l1k5c6g"},
			SubnetIID: irs.IID{SystemId: "oh-subnet-cqab15kvtts35l1k5c70"},

			// SecurityGroupIIDs 미지정시, NCP default 값으로서 "ncloud-default-acg"인 "293807이 적용됨.
			// SecurityGroupIIDs: []irs.IID{{SystemId: "293807"},{SystemId: "332703"}},
			SecurityGroupIIDs: []irs.IID{{SystemId: "1333707"}},

			VMUserPasswd: "abcd000abcd",

			TagList: []irs.KeyValue{
				{ Key: "aaa", Value: "aaaAAAAA"},
				{ Key: "ccc", Value: "cccCCCCC"},
			},
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return

			case 1:
				vmInfo, err := vmHandler.StartVM(vmReqInfo)
				if err != nil {
					//panic(err)
					cblogger.Error(err)
				} else {
					cblogger.Info("Succeeded in VM Creation!!", vmInfo)
					spew.Dump(vmInfo)
				}
				cblogger.Info("\nCreateVM Test Finished")

			case 2:
				vmInfo, err := vmHandler.GetVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] Failed to Get VM info!!", VmID)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] Result : ", VmID)
					spew.Dump(vmInfo)
				}
				cblogger.Info("\nGetVM Test Finished")

			case 3:
				cblogger.Info("Start Suspend VM ...")
				result, err := vmHandler.SuspendVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] Failed to Suspend VM : [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] Succeeded in VM Suspend : [%s]", VmID, result)
				}
				cblogger.Info("\nSuspendVM Test Finished")

			case 4:
				cblogger.Info("Start Resume  VM ...")
				result, err := vmHandler.ResumeVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] Failed to Resume VM : [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] Succeeded in VM Resumme : [%s]", VmID, result)
				}
				cblogger.Info("\nResumeVM Test Finished")

			case 5:
				cblogger.Info("Start Reboot  VM ...")
				result, err := vmHandler.RebootVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] Failed to Reboot VM : [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] Succeeded in VM Reboot : [%s]", VmID, result)
				}
				cblogger.Info("\nRebootVM Test Finished")

			case 6:
				cblogger.Info("Start Terminate  VM ...")
				result, err := vmHandler.TerminateVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] Failed to Terminate VM : [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] Succeeded in VM Terminate : [%s]", VmID, result)
				}
				cblogger.Info("\nTerminateVM Test Finished")

			case 7:
				cblogger.Info("Start Get VM Status...")
				vmStatus, err := vmHandler.GetVMStatus(VmID)
				if err != nil {
					cblogger.Errorf("[%s] Failed to Get VM Status : ", VmID)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] Succeeded in Getting VM Status : [%s]", VmID, vmStatus)
				}
				cblogger.Info("\nGet VMStatus Test Finished")

			case 8:
				cblogger.Info("Start ListVMStatus ...")
				vmStatusInfos, err := vmHandler.ListVMStatus()
				if err != nil {
					cblogger.Error("Failed to List VMStatus")
					cblogger.Error(err)
				} else {
					cblogger.Info("Succeeded in Listing VMStatus")
					spew.Dump(vmStatusInfos)
				}
				cblogger.Info("\nListVM Status Test Finished")

			case 9:
				cblogger.Info("Start ListVM ...")
				vmList, err := vmHandler.ListVM()
				if err != nil {
					cblogger.Error("Failed to List VM")
					cblogger.Error(err)
				} else {
					cblogger.Info("Succeeded in Listing VM")
					spew.Dump(vmList)
					cblogger.Infof("=========== Count VM : [%d] ================", len(vmList))
					if len(vmList) > 0 {
						VmID = vmList[0].IId
					}
				}
		}
	}
}
}

func main() {
	cblogger.Info("NCP Resource Test")

	handleVM()
}

//handlerType : resources폴더의 xxxHandler.go에서 Handler이전까지의 문자열
//(예) ImageHandler.go -> "Image"
func getResourceHandler(handlerType string) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(ncpdrv.NcpDriver)

	config := readConfigFile()
	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			ClientId:     config.Ncp.NcpAccessKeyID,
			ClientSecret: config.Ncp.NcpSecretKey,
		},
		RegionInfo: idrv.RegionInfo{
			Region: config.Ncp.Region,
			Zone:   config.Ncp.Zone,
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
	Ncp struct {
		NcpAccessKeyID string `yaml:"ncp_access_key_id"`
		NcpSecretKey   string `yaml:"ncp_secret_key"`
		Region         string `yaml:"region"`
		Zone           string `yaml:"zone"`

		ImageID string `yaml:"image_id"`

		VmID         string `yaml:"ncp_instance_id"`
		BaseName     string `yaml:"base_name"`
		InstanceType string `yaml:"instance_type"`
		KeyName      string `yaml:"key_name"`
		MinCount     int64  `yaml:"min_count"`
		MaxCount     int64  `yaml:"max_count"`

		SubnetID        string `yaml:"subnet_id"`
		SecurityGroupID string `yaml:"security_group_id"`

		PublicIP string `yaml:"public_ip"`
	} `yaml:"ncp"`
}

func readConfigFile() Config {
	// # Set Environment Value of Project Root Path
	// goPath := os.Getenv("GOPATH")
	// rootPath := goPath + "/src/github.com/cloud-barista/ncp/ncp/main"
	// cblogger.Debugf("Test Config file : [%]", rootPath+"/config/config.yaml")
	rootPath 	:= os.Getenv("CBSPIDER_ROOT")
	configPath 	:= rootPath + "/cloud-control-manager/cloud-driver/drivers/ncp/main/config/config.yaml"
	cblogger.Debugf("Test Config file : [%s]", configPath)

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

	// Just for test
	cblogger.Debug(config.Ncp.NcpAccessKeyID, " ", config.Ncp.Region)

	return config
}
