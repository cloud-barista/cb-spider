// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// by ETRI, 2021.12.

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

		securityName := "nhn-sg-04"
		//securityId := "67167e2e-2390-48d6-8f27-78c9293b26f3"
		securityId := "2f2e7af6-6ecd-4db2-a96b-20cf4987062a"
		vpcId := "1364a68d-4a7c-4a91-ac71-d29c7f4d47fc"
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

						{
							Direction:  "inbound",
							IPProtocol: "tcp",
							FromPort:   "80",
							ToPort:     "80",
							CIDR: 		"192.168.0.0/16",						
						},
						// {
						// 	Direction:  "inbound",
						// 	IPProtocol: "tcp",
						// 	FromPort:   "-1",
						// 	ToPort:     "-1",
						// 	CIDR: 		"192.168.0.0/16",							
						// },

						{
							Direction:  "inbound",
							IPProtocol: "udp",
							FromPort:   "8080",
							ToPort:     "8080",
							CIDR: 		"0.0.0.0/0",
						},
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
					cblogger.Infof(securityName, " Security Rule Add failed : ", err)
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
					cblogger.Infof(securityName, " Security Rule Remove failed : ", err)
				} else {
					cblogger.Infof("[%s] Security Rule 제거 결과 : [%v]", securityName, result)
					spew.Dump(result)
				}	
				cblogger.Info("\nRemoveRules Test Finished")

			case 6:
				cblogger.Infof("[%s] Security 삭제 테스트", securityId)
				result, err := handler.DeleteSecurity(irs.IID{SystemId: securityId})
				if err != nil {
					cblogger.Infof(securityId, " Security 삭제 실패 : ", err)
				} else {
					cblogger.Infof("[%s] Security 삭제 결과 : [%s]", securityId, result)
				}
			}
		}
	}
}

func testErr() error {

	return errors.New("")
}

func main() {
	cblogger.Info("NHN Cloud Resource Test")
	/*
		err := testErr()
		spew.Dump(err)
		if err != nil {
			cblogger.Info("에러 발생")
			awsErr, ok := err.(awserr.Error)
			spew.Dump(awsErr)
			spew.Dump(ok)
			if ok {
				if "404" == awsErr.Code() {
					cblogger.Info("404!!!")
				} else {
					cblogger.Info("404 아님")
				}
			}
		}
	*/

	handleSecurity()
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
