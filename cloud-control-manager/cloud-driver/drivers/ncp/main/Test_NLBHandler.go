// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// by ETRI, 2023.08.

package main

import (
	"os"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	// ncpdrv "github.com/cloud-barista/ncp/ncp"  // For local test
	ncpdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncp"	
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP Classic Resource Test")
	cblog.SetLevel("info")
}

// Test VMSpec
func handleNLB() {
	cblogger.Debug("Start NLBHandler Resource Test")

	ResourceHandler, err := getResourceHandler("NLB")
	if err != nil {
		panic(err)
	}

	nlbHandler := ResourceHandler.(irs.NLBHandler)

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ NLB Resource Test ]")
		cblogger.Info("1. ListNLB()")
		cblogger.Info("2. GetNLB()")
		cblogger.Info("3. CreateNLB()")
		cblogger.Info("4. DeleteNLB()")
		cblogger.Info("5. ChangeListener()")
		cblogger.Info("6. ChangeVMGroupInfo()")
		cblogger.Info("7. AddVMs()")
		cblogger.Info("8. RemoveVMs()")
		cblogger.Info("9. GetVMGroupHealthInfo()")
		cblogger.Info("10. ChangeHealthCheckerInfo()")
		cblogger.Info("11. Exit")
		fmt.Println("============================================================================================")

		config := readConfigFile()
		cblogger.Infof("\n # NCP Classic Region : [%s]", config.Ncp.Region)
		cblogger.Info("\n # Select Num : ")

		nlbIId := irs.IID{
			NameId: "new-nlb-01",
			SystemId: "19315507",
		}

		// Caution!! : Listener, VM Group and Healthchecker all use the same protocol type in NCP Classic NLB.(The Protocol specified by 'ProtocolTypeCode' when created).
		nlbCreateReqInfo := irs.NLBInfo{
			IId: irs.IID{
				NameId: "new-nlb-05",
			},
			VpcIID: irs.IID{
				NameId: "ncp-vpc-01",
			},
			Listener: irs.ListenerInfo{
				Protocol: "TCP",
				Port:     "80",
				// Port:     "8080",
			},
			VMGroup: irs.VMGroupInfo{
				Protocol: "TCP",
				Port:     "8080",
				// Port:     "8080",
				VMs: &[]irs.IID{
					{NameId: "ncp-test-vm-001"},
					// {NameId: "ncp-test-vm-002"},
				},
			},
			HealthChecker: irs.HealthCheckerInfo{
				Protocol:  "TCP",
				Port:      "8080",
				Interval:  -1,
				Timeout:   -1,
				Threshold: -1,
			},
		}

		updateListener := irs.ListenerInfo{
			Protocol: "TCP",
			Port:     "8087",
		}

		updateVMGroups := irs.VMGroupInfo{
			Protocol: "TCP",
			Port:     "8087",
		}

		addVMs := []irs.IID{
			// {NameId: "ncp-test-vm-002"},
			{NameId: "vm-01-cjo33q9jcupp70i7ad1g"},
			// {NameId: "s18431a1837f"},
		}

		removeVMs := []irs.IID{
			{NameId: "ncp-test-vm-002"},
			// {NameId: "s18431a1837f"},
		}
	
		updateHealthCheckerInfo := irs.HealthCheckerInfo{
			Protocol:  "HTTP",
			Port:      "8080",
			Interval:  7,
			Timeout:   5,
			Threshold: 4,
		}

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				cblogger.Info("Start ListNLB() ...")
				if list, err := nlbHandler.ListNLB(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(list)
					cblogger.Info("출력 결과 수 : ", len(list))
				}
				cblogger.Info("Finish ListNLB()")
			case 2:
				cblogger.Info("Start GetNLB() ...")
				if nlbInfo, err := nlbHandler.GetNLB(nlbIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(nlbInfo)
				}
				cblogger.Info("Finish GetNLB()")
			case 3:
				cblogger.Info("Start CreateNLB() ...")
				if nlbInfo, err := nlbHandler.CreateNLB(nlbCreateReqInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(nlbInfo)
				}
				cblogger.Info("Finish CreateNLB()")
			case 4:
				cblogger.Info("Start DeleteNLB() ...")
				if nlbStatus, err := nlbHandler.DeleteNLB(nlbIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(nlbStatus)
				}
				cblogger.Info("Finish DeleteNLB()")
			case 5:
				cblogger.Info("Start ChangeListener() ...")
				if nlbInfo, err := nlbHandler.ChangeListener(nlbIId, updateListener); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(nlbInfo)
				}
				cblogger.Info("Finish ChangeListener()")
			case 6:
				cblogger.Info("Start ChangeVMGroupInfo() ...")
				if info, err := nlbHandler.ChangeVMGroupInfo(nlbIId, updateVMGroups); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(info)
				}
				cblogger.Info("Finish ChangeVMGroupInfo()")
			case 7:
				cblogger.Info("Start AddVMs() ...")
				if info, err := nlbHandler.AddVMs(nlbIId, &addVMs); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(info)
				}
				cblogger.Info("Finish AddVMs()")
			case 8:
				cblogger.Info("Start RemoveVMs() ...")
				if result, err := nlbHandler.RemoveVMs(nlbIId, &removeVMs); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(result)
				}
				cblogger.Info("Finish RemoveVMs()")
			case 9:
				cblogger.Info("Start GetVMGroupHealthInfo() ...")
				if info, err := nlbHandler.GetVMGroupHealthInfo(nlbIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(info)
				}
				cblogger.Info("Finish GetVMGroupHealthInfo()")
			case 10:
				cblogger.Info("Start ChangeHealthCheckerInfo() ...")
				if info, err := nlbHandler.ChangeHealthCheckerInfo(nlbIId, updateHealthCheckerInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(info)
				}
				cblogger.Info("Finish ChangeHealthCheckerInfo()")
			case 11:
				cblogger.Info("Exit")
			}
		}
	}
}

func main() {
	cblogger.Info("NCP Classic Resource Test")

	handleNLB()
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
	case "VPC":
		resourceHandler, err = cloudConnection.CreateVPCHandler()
	case "NLB":
		resourceHandler, err = cloudConnection.CreateNLBHandler()
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

//환경 설정 파일 읽기
//환경변수 CBSPIDER_PATH 설정 후 해당 폴더 하위에 /config/config.yaml 파일 생성해야 함.
func readConfigFile() Config {
	// Set Environment Value of Project Root Path
	// rootPath := os.Getenv("CBSPIDER_PATH")
	goPath := os.Getenv("GOPATH")
	rootPath := goPath + "/src/github.com/cloud-barista/ncp/ncp/main"
	//rootpath := "D:/Workspace/mcloud-barista-config"
	// /mnt/d/Workspace/mcloud-barista-config/config/config.yaml
	cblogger.Debugf("Test Data 설정파일 : [%]", rootPath+"/config/config.yaml")

	data, err := os.ReadFile(rootPath + "/config/config.yaml")
	//data, err := ioutil.ReadFile("D:/Workspace/mcloud-bar-config/config/config.yaml")
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
	// cblogger.Info(config.Ncp.NcpAccessKeyID)
	// cblogger.Info(config.Ncp.NcpSecretKey)

	// NOTE Just for test
	cblogger.Debug(config.Ncp.NcpAccessKeyID, " ", config.Ncp.Region)

	return config
}
