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

func handleSecurity() {
	cblogger.Debug("Start Security Resource Test")

	ResourceHandler, err := getResourceHandler("Security")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.SecurityHandler)

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ Security Management Test ]")
		fmt.Println("1. List Security")
		fmt.Println("2. Get Security")
		fmt.Println("3. Create Security")
		fmt.Println("4. Delete Security")
		fmt.Println("5. List IID")
		fmt.Println("0. Quit")
		fmt.Println("\n   Select a number above!! : ")
		fmt.Println("============================================================================================")

		var commandNum int

		securityName := "KT-SG-1"
		securityId := "oh-sg-01" // KT Cloud SG은 파일로 저장되어 관리되므로 security id는 Name과 동일하게
		vpcId := "myTest-vpc-01"

		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return

			case 1:
				result, err := handler.ListSecurity()
				if err != nil {
					cblogger.Error(err)
					cblogger.Error("SecurityGroup list 조회 실패 : ", err)
				} else {
					cblogger.Info("SecurityGroup list 조회 결과")
					//cblogger.Info(result)
					spew.Dump(result)

					cblogger.Infof("=========== S/G list 수 : [%d] ================", len(result))
					if result != nil {
						securityId = result[0].IId.SystemId // 조회 및 삭제를 위해 생성된 ID로 변경
					}
				}

				cblogger.Info("\nListSecurity Test Finished")

			case 2:
				cblogger.Infof("[%s] SecurityGroup 정보 조회 테스트", securityId)
				result, err := handler.GetSecurity(irs.IID{SystemId: securityId})
				if err != nil {
					cblogger.Error(err)
					cblogger.Error(securityId, " SecurityGroup 조회 실패 : ", err)
				} else {
					cblogger.Infof("[%s] SecurityGroup 조회 결과 : [%v]", securityId, result)
					spew.Dump(result)
				}

				cblogger.Info("\nGetSecurity Test Finished")

			case 3:
				cblogger.Infof("[%s] Security 생성 테스트", securityName)

				securityReqInfo := irs.SecurityReqInfo{
					IId:    irs.IID{NameId: securityName},
					VpcIID: irs.IID{SystemId: vpcId},
					SecurityRules: &[]irs.SecurityRuleInfo{ //보안 정책 설정
						// {
						// 	FromPort:   "22",
						// 	ToPort:     "22",
						// 	IPProtocol: "tcp",
						// 	Direction:  "inbound",
						// },
						
						{
							FromPort:   "1",
							ToPort:     "65535",
							IPProtocol: "tcp",
							Direction:  "inbound",
						},

						{
							FromPort:   "-1",
							ToPort:     "-1",
							IPProtocol: "icmp",
							Direction:  "inbound",
						},

						// {
						// 	FromPort:   "9999",
						// 	ToPort:     "9999",
						// 	IPProtocol: "tcp",
						// 	Direction:  "outbound",
						// },

						{
							FromPort:   "-1",
							ToPort:     "-1",
							IPProtocol: "udp",
							Direction:  "inbound",
						},						

						// {
						// 	FromPort:   "80",
						// 	ToPort:     "80",
						// 	IPProtocol: "tcp",
						// 	Direction:  "inbound",
						// },
						// {
						// 	FromPort:   "8080",
						// 	ToPort:     "8080",
						// 	IPProtocol: "tcp",
						// 	Direction:  "inbound",
						// },

						// {
						// 	FromPort:   "443",
						// 	ToPort:     "443",
						// 	IPProtocol: "tcp",
						// 	Direction:  "outbound",
						// },
						// {
						// 	FromPort:   "8443",
						// 	ToPort:     "9999",
						// 	IPProtocol: "tcp",
						// 	Direction:  "outbound",
						// },

						// {
						// 	FromPort:   "1024",
						// 	ToPort:     "1024",
						// 	IPProtocol: "tcp",
						// 	Direction:  "inbound",
						// },

					},
				}

				result, err := handler.CreateSecurity(securityReqInfo)
				if err != nil {
					cblogger.Infof(securityName, " Security 생성 실패 : ", err)
				} else {
					cblogger.Infof("[%s] Security 생성 결과 : [%v]", securityName, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] Security 삭제 테스트", securityId)
				result, err := handler.DeleteSecurity(irs.IID{SystemId: securityId})
				if err != nil {
					cblogger.Infof(securityId, " Security 삭제 실패 : ", err)
				} else {
					cblogger.Infof("[%s] Security 삭제 결과 : [%s]", securityId, result)
				}

			case 5:
				cblogger.Info("Start ListIID() ...")
				result, err := handler.ListIID()
				if err != nil {
					cblogger.Error("Failed to retrieve S/G IID list: ", err)
				} else {
					cblogger.Info("Successfully retrieved S/G IID list!!")
					spew.Dump(result)
					cblogger.Debug(result)
					cblogger.Infof("Total number of IID list: [%d]", len(result))
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
	handleSecurity()
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
