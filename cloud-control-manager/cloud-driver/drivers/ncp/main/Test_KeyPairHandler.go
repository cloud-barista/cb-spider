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
	//return awserr.Error("")
	return errors.New("")
	// return ncloud.New("504", "찾을 수 없음", nil)
}

// Test KeyPair
func handleKeyPair() {
	cblogger.Debug("Start KeyPair Resource Test")

	ResourceHandler, err := getResourceHandler("KeyPair")
	if err != nil {
		panic(err)
	}
	//config := readConfigFile()
	//VmID := config.Ncp.VmID

	keyPairHandler := ResourceHandler.(irs.KeyPairHandler)

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ KeyPair Management Test ]")
		fmt.Println("1. List KeyPair")
		fmt.Println("2. Create KeyPair")
		fmt.Println("3. Get KeyPair")
		fmt.Println("4. Delete KeyPair")
		fmt.Println("0. Quit")
		fmt.Println("\n   Select a number above!! : ")
		fmt.Println("============================================================================================")

		//keyPairName := config.Ncp.KeyName
		keyPairName := "NCP-keypair-06"
		var commandNum int

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
					cblogger.Infof("KeyPair list 조회 실패 : ", err)
				} else {
					cblogger.Info("KeyPair list 조회 결과")
					//cblogger.Info(result)
					spew.Dump(result)

					cblogger.Infof("=========== KeyPair list 수 : [%d] ================", len(result))
				}

				cblogger.Info("\nListKey Test Finished")

			case 2:
				cblogger.Infof("[%s] KeyPair 생성 테스트", keyPairName)
				keyPairReqInfo := irs.KeyPairReqInfo{
					IId: irs.IID{NameId: keyPairName},
					//Name: keyPairName,
				}
				result, err := keyPairHandler.CreateKey(keyPairReqInfo)
				if err != nil {
					cblogger.Infof(keyPairName, " KeyPair 생성 실패 : ", err)
				} else {
					cblogger.Infof("[%s] KeyPair 생성 결과 : \n[%s]", keyPairName, result)
					//spew.Dump(result)
				}

				cblogger.Info("\nCreateKey Test Finished")

			case 3:
				cblogger.Infof("[%s] KeyPair 조회 테스트", keyPairName)
				result, err := keyPairHandler.GetKey(irs.IID{NameId: keyPairName})
				if err != nil {
					cblogger.Infof(keyPairName, " KeyPair 조회 실패 : ", err)
				} else {
					cblogger.Infof("[%s] KeyPair 조회 결과 : \n[%s]", keyPairName, result)
					//spew.Dump(result)
				}

				cblogger.Info("\nGetKey Test Finished")

			case 4:
				cblogger.Infof("[%s] KeyPair 삭제 테스트", keyPairName)
				result, err := keyPairHandler.DeleteKey(irs.IID{NameId: keyPairName})
				if err != nil {
					cblogger.Infof(keyPairName, " KeyPair 삭제 실패 : ", err)
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
	cblogger.Info("NCP Resource Test")

	handleKeyPair()
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
	case "KeyPair":
		resourceHandler, err = cloudConnection.CreateKeyPairHandler()
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
