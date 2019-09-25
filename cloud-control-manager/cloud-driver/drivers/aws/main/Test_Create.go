// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by devunet@mz.co.kr, 2019.08.

package main

import (
	"fmt"
	"io/ioutil"
	"os"

	awsdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/aws"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	cblog "github.com/cloud-barista/cb-log"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("AWS Test")
	cblog.SetLevel("debug")
}

// Test VM Deployment
func createVM() {
	cblogger.Debug("Start createVM()")

	vmHandler, err := setVMHandler()
	if err != nil {
		panic(err)
		cblogger.Error(err)
	}
	config := readConfigFile()

	KeyPairHandler, err := setKeyPairHandler()
	if err != nil {
		panic(err)
	}

	keyPairName := config.Aws.KeyName
	cblogger.Infof("[%s] 키 페어 조회 테스트", keyPairName)
	keyPairInfo, err := KeyPairHandler.GetKey(keyPairName)
	if err != nil {
		cblogger.Infof(keyPairName, " 키 페어 조회 실패 : ", err)

		cblogger.Infof("[%s] 키 페어 생성 테스트", keyPairName)
		keyPairReqInfo := irs.KeyPairReqInfo{
			Name: keyPairName,
		}
		keyPairInfo, err = KeyPairHandler.CreateKey(keyPairReqInfo)
		if err != nil {
			cblogger.Infof(keyPairName, " 키 페어 생성 실패 : ", err)
			return
		} else {
			cblogger.Infof("[%s] 키 페어 생성 결과 : [%s]", keyPairName, keyPairInfo)
		}
	} else {
		cblogger.Infof("[%s] 키 페어 조회 결과 : [%s]", keyPairName, keyPairInfo)
	}

	vmReqInfo := irs.VMReqInfo{
		Name: config.Aws.BaseName,
		ImageInfo: irs.ImageInfo{
			Id: config.Aws.ImageID,
		},
		SpecID: config.Aws.InstanceType,
		SecurityInfo: irs.SecurityInfo{
			Id: config.Aws.SecurityGroupID,
		},
		//KeyPairInfo: irs.KeyPairInfo{
		//	Name: config.Aws.KeyName,
		//},
		KeyPairInfo: keyPairInfo,
		VNetworkInfo: irs.VNetworkInfo{
			Id: config.Aws.SubnetID,
		},
	}

	vmInfo, err := vmHandler.StartVM(vmReqInfo)
	if err != nil {
		panic(err)
		cblogger.Error(err)
	}
	cblogger.Info("VM 생성 완료!!", vmInfo)
	//cblogger.Info(vm)
	spew.Dump(vmInfo)

	cblogger.Info("Finish Create VM")
}

// Test VM Lifecycle Management (Create/Suspend/Resume/Reboot/Terminate)
func handleVM() {
	cblogger.Debug("Start handleVM()")

	vmHandler, err := setVMHandler()
	if err != nil {
		panic(err)
	}
	config := readConfigFile()
	VmID := config.Aws.VmID

	for {
		fmt.Println("VM Management")
		fmt.Println("0. Quit")
		fmt.Println("1. VM Start")
		fmt.Println("2. VM Info")
		fmt.Println("3. Suspend VM")
		fmt.Println("4. Resume VM")
		fmt.Println("5. Reboot VM")
		fmt.Println("6. Terminate VM")

		fmt.Println("7. GetVMStatus VM")
		fmt.Println("8. ListVMStatus VM")
		fmt.Println("9. ListVM")

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
				createVM()

			case 2:
				vmInfo := vmHandler.GetVM(VmID)
				cblogger.Info("EC2[%s] 인스턴스 정보", VmID)
				cblogger.Debug(vmInfo)
				spew.Dump(vmInfo)

			case 3:
				cblogger.Debug("Start Suspend VM ...")
				vmHandler.SuspendVM(VmID)
				cblogger.Debug("Finish Suspend VM")

			case 4:
				cblogger.Debug("Start Resume  VM ...")
				vmHandler.ResumeVM(VmID)
				cblogger.Debug("Finish Resume VM")

			case 5:
				cblogger.Debug("Start Reboot  VM ...")
				vmHandler.RebootVM(VmID)
				cblogger.Debug("Finish Reboot VM")

			case 6:
				cblogger.Debug("Start Terminate  VM ...")
				vmHandler.TerminateVM(VmID)
				cblogger.Debug("Finish Terminate VM")

			case 7:
				cblogger.Debug("Start Get VM Status...")
				vmStatus := vmHandler.GetVMStatus(VmID)
				cblogger.Debug("Finish Get VM Status")

				cblogger.Info(vmStatus)

			case 8:
				cblogger.Debug("Start ListVMStatus ...")
				vmStatusInfos := vmHandler.ListVMStatus()
				cblogger.Info("리턴 값")
				cblogger.Info(vmStatusInfos)
				spew.Dump(vmStatusInfos)
				cblogger.Debug("Finish ListVMStatus")

			case 9:
				cblogger.Debug("Start ListVM ...")
				vmInfos := vmHandler.ListVM()
				cblogger.Info("=========== VM 목록 ================")
				spew.Dump(vmInfos)
				cblogger.Debug("Finish ListVM")
			}
		}
	}
}

func main() {
	cblogger.Info("AWS Driver Test")

	//createVM()
	//suspendVM(vmID)
	//RebootVM
	//resumeVM(vmID)
	handleVM()
}

func setKeyPairHandler() (irs.KeyPairHandler, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(awsdrv.AwsDriver)

	config := readConfigFile()
	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			ClientId:     config.Aws.AawsAccessKeyID,
			ClientSecret: config.Aws.AwsSecretAccessKey,
		},
		RegionInfo: idrv.RegionInfo{
			Region: config.Aws.Region,
		},
	}

	cloudConnection, err := cloudDriver.ConnectCloud(connectionInfo)
	if err != nil {
		return nil, err
	}

	keyPairHandler, err := cloudConnection.CreateKeyPairHandler()
	if err != nil {
		return nil, err
	}
	return keyPairHandler, nil
}

func setVMHandler() (irs.VMHandler, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(awsdrv.AwsDriver)

	config := readConfigFile()
	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			ClientId:     config.Aws.AawsAccessKeyID,
			ClientSecret: config.Aws.AwsSecretAccessKey,
		},
		RegionInfo: idrv.RegionInfo{
			Region: config.Aws.Region,
		},
	}

	cloudConnection, err := cloudDriver.ConnectCloud(connectionInfo)
	if err != nil {
		return nil, err
	}

	vmHandler, err := cloudConnection.CreateVMHandler()
	if err != nil {
		return nil, err
	}
	return vmHandler, nil
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
	Aws struct {
		AawsAccessKeyID    string `yaml:"aws_access_key_id"`
		AwsSecretAccessKey string `yaml:"aws_secret_access_key"`
		Region             string `yaml:"region"`

		ImageID string `yaml:"image_id"`

		VmID         string `yaml:"ec2_instance_id"`
		BaseName     string `yaml:"base_name"`
		InstanceType string `yaml:"instance_type"`
		KeyName      string `yaml:"key_name"`
		MinCount     int64  `yaml:"min_count"`
		MaxCount     int64  `yaml:"max_count"`

		SubnetID        string `yaml:"subnet_id"`
		SecurityGroupID string `yaml:"security_group_id"`
	} `yaml:"aws"`
}

//환경 설정 파일 읽기
//환경변수 CBSPIDER_PATH 설정 후 해당 폴더 하위에 /config/config.yaml 파일 생성해야 함.
func readConfigFile() Config {
	// Set Environment Value of Project Root Path
	rootPath := os.Getenv("CBSPIDER_PATH")
	//rootpath := "D:/Workspace/mcloud-barista-config"
	// /mnt/d/Workspace/mcloud-barista-config/config/config.yaml
	cblogger.Debugf("Test Data 설정파일 : [%]", rootPath+"/config/config.yaml")

	data, err := ioutil.ReadFile(rootPath + "/config/config.yaml")
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
	cblogger.Info(config)
	return config
}
