// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// by ETRI, 2022.07.

package main

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"os"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	nhndrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/nhn"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NHN Cloud Resource Test")
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
		cblogger.Infof("\n # NHN Cloud Region : [%s]", config.NhnCloud.Region)
		cblogger.Info("\n # Num : ")

		nlbIId := irs.IID{
			NameId:   "new_lb-1",
			SystemId: "e2cfd23e-0af5-4c58-802d-e8c0c23b190d",
		}

		nlbCreateReqInfo := irs.NLBInfo{
			IId: irs.IID{
				NameId: "new_nlb-1",
			},
			VpcIID: irs.IID{
				NameId: "ohvpc-0415-coecqt4vtts22sf78g3g",
			},
			Listener: irs.ListenerInfo{
				Protocol: "TCP",
				Port:     "80",
			},
			VMGroup: irs.VMGroupInfo{
				Protocol: "TCP",
				Port:     "80",
				VMs: &[]irs.IID{
					// {NameId: "nhn-vm-1"},
					{NameId: "ohvm-01-coeddocvtts22sf78g70"},
				},
			},
			HealthChecker: irs.HealthCheckerInfo{
				Protocol:  "TCP",
				Port:      "80",
				Interval:  -1,
				Timeout:   -1,
				Threshold: -1,
			},
			// HealthChecker: irs.HealthCheckerInfo{
			// 	Protocol:  "TCP",
			// 	Port:      "8080",
			// 	Interval:  30,
			// 	Timeout:   5,
			// 	Threshold: 3,
			// },
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
			{NameId: "nhn-vm-3"},
			{NameId: "nhn-vm-02"},
		}

		removeVMs := []irs.IID{
			{NameId: "ohvm-01-coeddocvtts22sf78g70"},
			// {NameId: "nhn-vm-3"},
			// {NameId: "nhn-vm-4"},
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
	cblogger.Info("NHN Cloud Resource Test")

	handleNLB()
}

// handlerType : resources폴더의 xxxHandler.go에서 Handler이전까지의 문자열
// (예) ImageHandler.go -> "Image"
func getResourceHandler(handlerType string) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(nhndrv.NhnCloudDriver)

	config := readConfigFile()
	// spew.Dump(config)

	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			IdentityEndpoint: config.NhnCloud.IdentityEndpoint,
			Username:         config.NhnCloud.Nhn_Username,
			Password:         config.NhnCloud.Api_Password,
			DomainName:       config.NhnCloud.DomainName,
			TenantId:         config.NhnCloud.TenantId,
		},
		RegionInfo: idrv.RegionInfo{
			Region: config.NhnCloud.Region,
			Zone:   config.NhnCloud.Zone,
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
	NhnCloud struct {
		IdentityEndpoint string `yaml:"identity_endpoint"`
		Nhn_Username     string `yaml:"nhn_username"`
		Api_Password     string `yaml:"api_password"`
		DomainName       string `yaml:"domain_name"`
		TenantId         string `yaml:"tenant_id"`
		Region           string `yaml:"region"`
		Zone             string `yaml:"zone"`

		VMName         string `yaml:"vm_name"`
		ImageId        string `yaml:"image_id"`
		VMSpecId       string `yaml:"vmspec_id"`
		NetworkId      string `yaml:"network_id"`
		SecurityGroups string `yaml:"security_groups"`
		KeypairName    string `yaml:"keypair_name"`

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
	rootPath := os.Getenv("CBSPIDER_ROOT")
	configPath := rootPath + "/cloud-control-manager/cloud-driver/drivers/nhn/main/conf/config.yaml"
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
