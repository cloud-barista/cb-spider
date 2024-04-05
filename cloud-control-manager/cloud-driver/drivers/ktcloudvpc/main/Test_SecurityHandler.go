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

func handleSecurity() {
	cblogger.Debug("Start Security Resource Test")

	ResourceHandler, err := getResourceHandler("Security")
	if err != nil {
		panic(err)
	}
	
	handler := ResourceHandler.(irs.SecurityHandler)

	//config := readConfigFile()

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ Security Management Test ]")
		fmt.Println("1. List Security")
		fmt.Println("2. Get Security")
		fmt.Println("3. Create Security")
		fmt.Println("4. Add Rules")
		fmt.Println("5. Remove Rules")
		fmt.Println("6. Delete Security")
		fmt.Println("0. Quit")
		fmt.Println("\n   Select a number above!! : ")
		fmt.Println("============================================================================================")

		var commandNum int

		securityName := "ktvpc-sg-1"
		securityId := "ktvpc-sg-1"
		vpcId := "60e5d9da-55cd-47be-a0d9-6cf67c54f15c"
		// vpcNameId := "nhn-vpc-01"

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
				// result, err := handler.GetSecurity(irs.IID{NameId: securityName})
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
					// VpcIID: irs.IID{NameId: vpcNameId},
					SecurityRules: &[]irs.SecurityRuleInfo{ //보안 정책 설정
						// {
						// 	Direction:  "inbound",
						// 	IPProtocol: "tcp",
						// 	FromPort:   "20",
						// 	ToPort:     "22",
						// 	CIDR: 		"0.0.0.0/0",
						// },

						// {
						// 	Direction:  "inbound",
						// 	IPProtocol: "tcp",
						// 	FromPort:   "80",
						// 	ToPort:     "80",
						// 	CIDR: 		"0.0.0.0/0",						
						// },
						// {
						// 	Direction:  "inbound",
						// 	IPProtocol: "tcp",
						// 	FromPort:   "-1",
						// 	ToPort:     "-1",
						// 	CIDR: 		"192.168.0.0/16",							
						// },

						// {
						// 	Direction:  "inbound",
						// 	IPProtocol: "udp",
						// 	FromPort:   "8080",
						// 	ToPort:     "8080",
						// 	CIDR: 		"0.0.0.0/0",
						// },
						// {
						// 	Direction:  "inbound",
						// 	IPProtocol: "icmp",
						// 	FromPort:   "-1",
						// 	ToPort:     "-1",
						// 	CIDR: 		"0.0.0.0/0",
						// },
						// {
						// 	Direction:  "outbound",
						// 	IPProtocol: "tcp",
						// 	FromPort:   "443",
						// 	ToPort:     "443",
						// 	CIDR: 		"0.0.0.0/0",
						// },

						// {
						// 	Direction:  "outbound",
						// 	IPProtocol: "tcp",
						// 	FromPort:   "8443",
						// 	ToPort:     "9999",
						// 	CIDR: 		"192.168.0.0/16",	
						// },


						// // All traffic 허용 rule
						{
							Direction:  "inbound",
							IPProtocol: "ALL",
							FromPort:   "-1",
							ToPort:     "-1",
							CIDR: 		"0.0.0.0/0",
						},

						{
							Direction:  "outbound",
							IPProtocol: "ALL",
							FromPort:   "-1",
							ToPort:     "-1",
							CIDR: 		"0.0.0.0/0",
						},

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
				cblogger.Infof("[%s] Security Rule 추가 테스트", securityName)

				securityRuleReqInfo := &[]irs.SecurityRuleInfo{
					// {
					// 	Direction:  "inbound",
					// 	IPProtocol: "tcp",
					// 	FromPort:   "20",
					// 	ToPort:     "22",
					// 	CIDR: 		"0.0.0.0/0",
					// },

					// {
					// 	Direction:  "inbound",
					// 	IPProtocol: "tcp",
					// 	FromPort:   "80",
					// 	ToPort:     "80",
					// 	CIDR: 		"192.168.0.0/16",						
					// },
					// {
					// 	Direction:  "inbound",
					// 	IPProtocol: "tcp",
					// 	FromPort:   "-1",
					// 	ToPort:     "-1",
					// 	CIDR: 		"192.168.0.0/16",							
					// },

					// {
					// 	Direction:  "inbound",
					// 	IPProtocol: "udp",
					// 	FromPort:   "8080",
					// 	ToPort:     "8080",
					// 	CIDR: 		"0.0.0.0/0",
					// },
					// {
					// 	Direction:  "inbound",
					// 	IPProtocol: "icmp",
					// 	FromPort:   "-1",
					// 	ToPort:     "-1",
					// 	CIDR: 		"0.0.0.0/0",
					// },
					// {
					// 	Direction:  "outbound",
					// 	IPProtocol: "tcp",
					// 	FromPort:   "443",
					// 	ToPort:     "443",
					// 	CIDR: 		"0.0.0.0/0",
					// },

					// {
					// 	Direction:  "outbound",
					// 	IPProtocol: "tcp",
					// 	FromPort:   "8443",
					// 	ToPort:     "9999",
					// 	CIDR: 		"192.168.0.0/16",	
					// },


					// // All traffic 허용 rule
					{
						Direction:  "inbound",
						IPProtocol: "ALL",
						FromPort:   "-1",
						ToPort:     "-1",
						CIDR: 		"0.0.0.0/0",
					},
					{
						Direction:  "outbound",
						IPProtocol: "ALL",
						FromPort:   "-1",
						ToPort:     "-1",
						CIDR: 		"0.0.0.0/0",
					},
				}

				result, err := handler.AddRules(irs.IID{SystemId: securityId}, securityRuleReqInfo)
				if err != nil {
					cblogger.Infof("[%s] Security Rule Add failed : [%v]", securityName, err)
				} else {
					cblogger.Infof("[%s] Security Rule 추가 결과 : [%v]", securityName, result)
					spew.Dump(result)
				}
				cblogger.Info("\nAddRules Test Finished")

			case 5:
				cblogger.Infof("[%s] Security Rule 제거 테스트", securityName)

				securityRuleReqInfo := &[]irs.SecurityRuleInfo{
					// {
						
					// 	Direction:  "inbound",
					// 	IPProtocol: "tcp",
					// 	FromPort:   "20",
					// 	ToPort:     "22",
					// 	CIDR: 		"0.0.0.0/0",
					// },
					// {
					// 	Direction:  "inbound",
					// 	IPProtocol: "tcp",
					// 	FromPort:   "80",
					// 	ToPort:     "80",
					// 	CIDR: 		"192.168.0.0/16",							
					// },
					// {
					// 	Direction:  "inbound",
					// 	IPProtocol: "tcp",
					// 	FromPort:   "-1",
					// 	ToPort:     "-1",
					// 	CIDR: 		"192.168.0.0/16",							
					// },
					// {
					// 	Direction:  "inbound",
					// 	IPProtocol: "udp",
					// 	FromPort:   "8080",
					// 	ToPort:     "8080",
					// 	CIDR: 		"0.0.0.0/0",
					// },
					// {
					// 	Direction:  "inbound",
					// 	IPProtocol: "icmp",
					// 	FromPort:   "-1",
					// 	ToPort:     "-1",
					// 	CIDR: 		"0.0.0.0/0",
					// },
					// {
					// 	Direction:  "outbound",
					// 	IPProtocol: "tcp",
					// 	FromPort:   "443",
					// 	ToPort:     "443",
					// 	CIDR: 		"0.0.0.0/0",
					// },
					// {
					// 	Direction:  "outbound",
					// 	IPProtocol: "tcp",
					// 	FromPort:   "8443",
					// 	ToPort:     "9999",
					// 	CIDR: 		"192.168.0.0/16",	
					// },


					// // All traffic 허용 rule
					{
						Direction:  "inbound",
						IPProtocol: "ALL",
						FromPort:   "-1",
						ToPort:     "-1",
						CIDR: 		"0.0.0.0/0",
					},
					{
						Direction:  "outbound",
						IPProtocol: "ALL",
						FromPort:   "-1",
						ToPort:     "-1",
						CIDR: 		"0.0.0.0/0",
					},
				}
				result, err := handler.RemoveRules(irs.IID{SystemId: securityId}, securityRuleReqInfo)
				if err != nil {
					cblogger.Infof("[%s] Security Rule Remove failed : [%v]", securityName, err)
				} else {
					cblogger.Infof("[%s] Security Rule 제거 결과 : [%t]", securityName, result)
					spew.Dump(result)
				}	
				cblogger.Info("\nRemoveRules Test Finished")

			case 6:
				cblogger.Infof("[%s] Security 삭제 테스트", securityId)
				result, err := handler.DeleteSecurity(irs.IID{SystemId: securityId})
				if err != nil {
					cblogger.Infof(securityId, " Security 삭제 실패 : ", err)
				} else {
					cblogger.Infof("[%s] Security 삭제 결과 : [%t]", securityId, result)
				}
			}
		}
	}
}

func testErr() error {

	return errors.New("")
}

func main() {
	cblogger.Info("KT Cloud VPC Resource Test")

	handleSecurity()
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
