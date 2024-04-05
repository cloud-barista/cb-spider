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

// Test KeyPair
func handleKeyPair() {
	cblogger.Debug("Start KeyPair Resource Test")

	ResourceHandler, err := getResourceHandler("KeyPair")
	if err != nil {
		panic(err)
	}
	keyPairHandler := ResourceHandler.(irs.KeyPairHandler)

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ KeyPair Management Test ]")
		fmt.Println("1. List KeyPair")
		fmt.Println("2. Get KeyPair")
		fmt.Println("3. Create KeyPair")
		fmt.Println("4. Delete KeyPair")
		fmt.Println("0. Quit")
		fmt.Println("\n   Select a number above!! : ")
		fmt.Println("============================================================================================")

		var commandNum int
		config := readConfigFile()

		keyPairName := config.KTCloudVPC.KeypairName
		reqKeypairName := config.KTCloudVPC.ReqKeypairName

		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return

			case 1:
				result, err := keyPairHandler.ListKey()
				if err != nil {
					cblogger.Error(err)
					cblogger.Error("KeyPair list 조회 실패 : ", err)
				} else {
					cblogger.Info("KeyPair list 조회 결과")
					//cblogger.Info(result)
					spew.Dump(result)

					cblogger.Infof("=========== KeyPair list 수 : [%d] ================", len(result))
				}
				cblogger.Info("\nListKey Test Finished")

			case 2:
				cblogger.Infof("[%s] KeyPair 조회 테스트", keyPairName)
				result, err := keyPairHandler.GetKey(irs.IID{NameId: keyPairName})
				if err != nil {
					cblogger.Error(err)
					cblogger.Error(keyPairName, " KeyPair 조회 실패 : ", err)
				} else {
					cblogger.Infof("[%s] KeyPair 조회 결과 : \n[%s]", keyPairName, result)
					spew.Dump(result)
				}
				cblogger.Info("\nGetKey Test Finished")

			case 3:
				cblogger.Infof("[%s] KeyPair 생성 테스트", reqKeypairName)
				keyPairReqInfo := irs.KeyPairReqInfo{
					IId: irs.IID{NameId: reqKeypairName},
					//Name: keyPairName,
				}
				result, err := keyPairHandler.CreateKey(keyPairReqInfo)
				if err != nil {
					cblogger.Error(err)
					cblogger.Error(reqKeypairName, " KeyPair 생성 실패 : ", err)
				} else {
					cblogger.Infof("[%s] KeyPair 생성 결과 : \n[%s]", reqKeypairName, result)
					spew.Dump(result)
				}
				cblogger.Info("\nCreateKey Test Finished")

			case 4:
				cblogger.Infof("[%s] KeyPair 삭제 테스트", keyPairName)
				result, err := keyPairHandler.DeleteKey(irs.IID{NameId: keyPairName})
				if err != nil {
					cblogger.Error(err)
					cblogger.Error(keyPairName, " KeyPair 삭제 실패 : ", err)
				} else {
					cblogger.Infof("[%s] KeyPair 삭제 결과 : [%s]", keyPairName, result)
					spew.Dump(result)
				}
				cblogger.Info("\nDeleteKey Test Finished")
			}
		}
	}
}

func main() {
	cblogger.Info("KT Cloud VPC Resource Test")

	handleKeyPair()
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

	// NOTE Just for test
	//cblogger.Info(config.Ncp.NcpAccessKeyID)
	//cblogger.Info(config.Ncp.NcpSecretKey)

	cloudConnection, errCon := cloudDriver.ConnectCloud(connectionInfo)
	if errCon != nil {
		return nil, errCon
	}

	var resourceHandler interface{}
	var err error

	switch handlerType {
	case "Image":
		resourceHandler, err = cloudConnection.CreateImageHandler()
	case "KeyPair":
		resourceHandler, err = cloudConnection.CreateKeyPairHandler()
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
		Zone             string `yaml:"zone"`

		VMName           string `yaml:"vm_name"`
		ImageId          string `yaml:"image_id"`
		VMSpecId         string `yaml:"vmspec_id"`
		NetworkId        string `yaml:"network_id"`
		SecurityGroups   string `yaml:"security_groups"`
		KeypairName      string `yaml:"keypair_name"`
		ReqKeypairName 	 string `yaml:"req_keypair_name"`

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
