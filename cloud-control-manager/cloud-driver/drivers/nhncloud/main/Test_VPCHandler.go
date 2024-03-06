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

		vpcIId := irs.IID{SystemId: "328d03c7-4656-4e14-88cb-fc4c222a97d4"}
		subnetIId := irs.IID{SystemId: "b7bf20e6-eca7-444a-a6b9-3b57aa8b8da7"}

		vpcReqName := "nhn-vpc-1"
		subnetReqName := "nhn-subnet-1"

		var subnetInfoList []irs.SubnetInfo
			info := irs.SubnetInfo{
				IId: irs.IID{
					NameId: subnetReqName,
				},
				// IPv4_CIDR: "10.0.0.0/28",
				IPv4_CIDR: "172.16.0.0/24",
			}
			subnetInfoList = append(subnetInfoList, info)
		
		vpcReqInfo := irs.VPCReqInfo{
			IId: irs.IID{
					NameId: vpcReqName,
				},
			// IPv4_CIDR:      "10.0.0.0/24",
			IPv4_CIDR:      "172.16.0.0/12",
			// IPv4_CIDR:      "172.16.0.0/16",
			SubnetInfoList: subnetInfoList,
		}
		
		addSubnetReqInfo := irs.SubnetInfo {
			IId: irs.IID{
				NameId: "nhn-subnet-3",
			},
			IPv4_CIDR: "172.16.1.0/24",
		}

		//NHN Cloud VPC CIDR은 아래의 사설 주소 범위로 입력되어야 함.
			// 10.0.0.0/8
			// 172.16.0.0/12
			// 192.168.0.0/16
			
			// CIDR은 링크 로컬 주소 범위(169.254.0.0/16)로 입력할 수 없음.
			//	/24보다 큰 CIDR 블록은 입력할 수 없음.
			// VPC은 최대 3개까지 생성 가능

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
				if result, err := handler.AddSubnet(vpcIId, addSubnetReqInfo); err != nil {
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
	cblogger.Info("NHN Cloud Resource Test")

	handleVPC()
}

//handlerType : resources폴더의 xxxHandler.go에서 Handler이전까지의 문자열
//(예) ImageHandler.go -> "Image"
func getResourceHandler(handlerType string) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(nhndrv.NhnCloudDriver)

	config := readConfigFile()
	// spew.Dump(config)

	cblogger.Infof("\n # NHN Cloud Region : [%s]", config.NhnCloud.Region)

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
