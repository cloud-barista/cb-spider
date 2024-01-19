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
	"os"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	ktdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ktcloud"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("KT Cloud Resource Test")
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
		cblogger.Infof("\n # KT Cloud Region : [%s]", config.KtCloud.Region)
		cblogger.Infof("\n # KT Cloud Zone : [%s]", config.KtCloud.Zone)
		cblogger.Info("\n # Num : ")

		nlbIId := irs.IID{
			NameId: "new_lb-5",
			SystemId: "38044",
		}

		nlbCreateReqInfo := irs.NLBInfo{
			IId: irs.IID{
				NameId: "new-nlb-2",
			},
			VpcIID: irs.IID{
				NameId: "kt-vpc-01",
			},
			Listener: irs.ListenerInfo{
				Protocol: "TCP",
				Port:     "80",
			},
			VMGroup: irs.VMGroupInfo{
				Protocol: "TCP",
				Port:     "80",
				VMs: &[]irs.IID{
					{NameId: "kt-vm-1"},
					{NameId: "kt-vm-2"},
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
			{NameId: "kt-vm-03"},
			{NameId: "kt-vm-04"},
		}

		removeVMs := []irs.IID{
			{NameId: "kt-vm-01"},
			{NameId: "kt-vm-02"},
			// {NameId: "kt-vm-3"},
			// {NameId: "kt-vm-4"},
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
	cblogger.Info("KT Cloud Resource Test")

	handleNLB()
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
	case "NLB":
		resourceHandler, err = cloudConnection.CreateNLBHandler()
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
