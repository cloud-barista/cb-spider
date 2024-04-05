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
	// "errors"
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

// Test VMSpec
func handleVPC() {
	cblogger.Debug("Start VPCHandler Resource Test")

	ResourceHandler, err := getResourceHandler("VPC")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.VPCHandler)

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ VPC Resource Test ]")
		fmt.Println("1. CreateVPC()")
		fmt.Println("2. ListVPC()")
		fmt.Println("3. GetVPC()")
		fmt.Println("4. AddSubnet()")
		fmt.Println("5. RemoveSubnet()")
		fmt.Println("6. DeleteVPC()")
		fmt.Println("0. Exit")
		fmt.Println("\n   Select a number above!! : ")
		fmt.Println("============================================================================================")

		config := readConfigFile()
		cblogger.Info("# config.KTCloudVPC.Zone : ", config.KTCloudVPC.Zone)

		vpcReqName := "kt-vpc-1"
		vpcIId := irs.IID{NameId: vpcReqName, SystemId: "60e5d9da-55cd-47be-a0d9-6cf67c54f15c"}

		// subnetIId := irs.IID{SystemId: "e1a55d19-9412-4cff-bee0-0c446752ce91"}
		subnetIId := irs.IID{SystemId: "afaa6ca3-67e8-4a53-95ec-c1a98c79ab55"} // Caution!!) Tier 'OSNetworkID' among REST API parameters

		cblogger.Info("reqVPCName : ", vpcReqName)

		// KT Cloud VPC (D1) 서비스의 VPC CIDR은 다음의 사설 주소 범위로 입력되어야 함. : 172.25.0.0/12
		// 네트워크를 기본설정 선택시, 각각 DMZ : 172.25.0.0/24, Private : 172.25.1.0/24로 제공됨.

		vpcReqInfo := irs.VPCReqInfo{
			IId:            vpcIId,
			IPv4_CIDR:      "172.25.0.0/12",
			SubnetInfoList: []irs.SubnetInfo {
				// {
				// 	IId: irs.IID{
				// 		NameId: "ktsubnet1-1",
				// 	},
				// 	IPv4_CIDR: "172.25.2.0/24",
				// },
				{
					IId: irs.IID{
						NameId: "ktsubnet1-2",
					},
					IPv4_CIDR: "172.25.3.0/24",
				},
			},
		}
			
		subnetReqInfo := irs.SubnetInfo {
			IId: irs.IID{
				NameId: "NLB-SUBNET",
			},
			IPv4_CIDR: "172.25.100.0/24",
		}
		
		var commandNum int

		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				fmt.Println("Start CreateVPC() ...")

				vpcInfo, err := handler.CreateVPC(vpcReqInfo)
				if err != nil {
					//panic(err)
					cblogger.Error(err)
					cblogger.Error("VPC 생성 실패 : ", err)
				} else {
					cblogger.Info("VPC 생성 완료!!", vpcInfo)
					spew.Dump(vpcInfo)
					cblogger.Debug(vpcInfo)
				}

				fmt.Println("\nCreateVPC() Test Finished")

			case 2:
				fmt.Println("Start ListVPC() ...")
				result, err := handler.ListVPC()
				if err != nil {
					cblogger.Error(err)
					cblogger.Error("VPC list 조회 실패 : ", err)
				} else {
					cblogger.Info("VPC list 조회 성공!!")
					spew.Dump(result)
					cblogger.Debug(result)
					cblogger.Infof("전체 list 개수 : [%d]", len(result))
				}

				fmt.Println("\nListVMSpec() Test Finished")
				
			case 3:
				fmt.Println("Start GetVPC() ...")
				if vpcInfo, err := handler.GetVPC(vpcIId); err != nil {
					cblogger.Error(err)
					cblogger.Error("VPC 정보 조회 실패 : ", err)
				} else {
					cblogger.Info("VPC 정보 조회 성공!!")
					spew.Dump(vpcInfo)
				}
				fmt.Println("\nGetVPC() Test Finished")

			case 4:
				fmt.Println("Start AddSubnet() ...")
				if result, err := handler.AddSubnet(vpcIId, subnetReqInfo); err != nil {
					cblogger.Error(err)
					cblogger.Error("Subnet 추가 실패 : ", err)
				} else {
					cblogger.Info("Subnet 추가 성공!!")
					spew.Dump(result)
				}
				fmt.Println("\nAddSubnet() Test Finished")

			case 5:
				fmt.Println("Start RemoveSubnet() ...")
				if result, err := handler.RemoveSubnet(vpcIId, subnetIId); err != nil {
					cblogger.Error(err)
					cblogger.Error("Subnet 제거 실패 : ", err)
				} else {
					cblogger.Info("Subnet 제거 성공!!")
					spew.Dump(result)
				}
				fmt.Println("\nRemoveSubnet() Test Finished")

			case 6:
				fmt.Println("Start DeleteVPC() ...")
				if result, err := handler.DeleteVPC(vpcIId); err != nil {
					cblogger.Error(err)
					cblogger.Error("VPC 삭제 실패 : ", err)
				} else {
					cblogger.Info("VPC 삭제 성공!!")
					spew.Dump(result)
				}
				fmt.Println("\nGetVPC() Test Finished")

			case 0:
				fmt.Println("Exit")
				return
			}
		}
	}
}

func main() {
	cblogger.Info("KT Cloud VPC Resource Test")

	handleVPC()
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
// VmID : 라이프 사이트클을 테스트할 EC2 인스턴스ID
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
