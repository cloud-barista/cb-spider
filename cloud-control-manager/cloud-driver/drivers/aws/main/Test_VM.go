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
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

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

	//minCount := aws.Int64(int64(config.Aws.MinCount))
	//maxCount := aws.Int64(config.Aws.MaxCount)
	/*
		vmReqInfo := irs.VMReqInfo{
			VMName:             config.Aws.BaseName,
			ImageId:            config.Aws.ImageID,
			VirtualNetworkId:   config.Aws.SubnetID,
			NetworkInterfaceId: "eni-00befb6d8c3a87b24",
			PublicIPId:         "eipalloc-0e95789a23e6d0c6f",
			//SecurityGroupIds: []string{"sg-0df1c209ea1915e4b"},
			SecurityGroupIds: []string{config.Aws.SecurityGroupID},
			VMSpecId:         config.Aws.InstanceType,
			KeyPairName:      config.Aws.KeyName,
		}
	*/
	vmReqInfo := irs.VMReqInfo{
		//VMName:           config.Aws.BaseName,
		VMName:           "mcloud-barista-vnictest3",
		ImageId:          config.Aws.ImageID,
		VirtualNetworkId: "CB-VNet-Subnet",
		//NetworkInterfaceId: "eni-00befb6d8c3a87b24",
		PublicIPId:       "mcloud-barista-eip-test",
		SecurityGroupIds: []string{"cb-sgtest-mcloud-barista"},
		//SecurityGroupIds: []string{"cb-sgtest-mcloud-barista", "cb-sgtest-mcloud-barista2"},
		//SecurityGroupIds: []string{config.Aws.SecurityGroupID},
		VMSpecId:    config.Aws.InstanceType,
		KeyPairName: config.Aws.KeyName,
	}

	vmInfo, err := vmHandler.StartVM(vmReqInfo)
	if err != nil {
		//panic(err)
		cblogger.Error(err)
	} else {
		cblogger.Info("VM 생성 완료!!", vmInfo)
		spew.Dump(vmInfo)
	}
	//cblogger.Info(vm)

	cblogger.Info("Finish Create VM")
}

/*
func suspendVM(vmID string) {
	fmt.Println("Start Suspend VM Test.. [" + vmID + "]")
	vmHandler, err := setVMHandler()
	if err != nil {
		panic(err)
	}

	vmHandler.SuspendVM(vmID)
	fmt.Println("Finish Suspend VM")
}

func resumeVM(vmID string) {
	fmt.Println("Start ResumeVM VM Test.. [" + vmID + "]")
	vmHandler, err := setVMHandler()
	if err != nil {
		panic(err)
	}

	vmHandler.ResumeVM(vmID)
	fmt.Println("Finish ResumeVM VM")
}
*/

// Test VM Lifecycle Management (Create/Suspend/Resume/Reboot/Terminate)
func handleVM() {
	cblogger.Debug("Start handleVM()")

	vmHandler, err := setVMHandler()
	if err != nil {
		panic(err)
	}
	config := readConfigFile()
	VmID := config.Aws.VmID
	VmID = "mcloud-barista-vnictest3"

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
				vmInfo, err := vmHandler.GetVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] VM 정보 조회 실패", VmID)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] VM 정보 조회 결과", VmID)
					cblogger.Info(vmInfo)
					spew.Dump(vmInfo)
				}

			case 3:
				cblogger.Info("Start Suspend VM ...")
				result, err := vmHandler.SuspendVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] VM Suspend 실패 - [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] VM Suspend 성공 - [%s]", VmID, result)
				}

			case 4:
				cblogger.Info("Start Resume  VM ...")
				result, err := vmHandler.ResumeVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] VM Resume 실패 - [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] VM Resume 성공 - [%s]", VmID, result)
				}

			case 5:
				cblogger.Info("Start Reboot  VM ...")
				result, err := vmHandler.RebootVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] VM Reboot 실패 - [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] VM Reboot 성공 - [%s]", VmID, result)
				}

			case 6:
				cblogger.Info("Start Terminate  VM ...")
				result, err := vmHandler.TerminateVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] VM Terminate 실패 - [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] VM Terminate 성공 - [%s]", VmID, result)
				}

			case 7:
				cblogger.Info("Start Get VM Status...")
				vmStatus, err := vmHandler.GetVMStatus(VmID)
				if err != nil {
					cblogger.Errorf("[%s] VM Get Status 실패", VmID)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] VM Get Status 성공 : [%s]", VmID, vmStatus)
				}

			case 8:
				cblogger.Info("Start ListVMStatus ...")
				vmStatusInfos, err := vmHandler.ListVMStatus()
				if err != nil {
					cblogger.Error("ListVMStatus 실패")
					cblogger.Error(err)
				} else {
					cblogger.Info("ListVMStatus 성공")
					cblogger.Info(vmStatusInfos)
					spew.Dump(vmStatusInfos)
				}

			case 9:
				cblogger.Info("Start ListVM ...")
				vmList, err := vmHandler.ListVM()
				if err != nil {
					cblogger.Error("ListVM 실패")
					cblogger.Error(err)
				} else {
					cblogger.Info("ListVM 성공")
					cblogger.Info("=========== VM 목록 ================")
					cblogger.Info(vmList)
					spew.Dump(vmList)
				}

			}
		}
	}
}

type VMStatus string

const (
	Pending VMStatus = "PENDING" // from launch, suspended to running
	Running VMStatus = "RUNNING"

	Suspending VMStatus = "SUSPENDING" // from running to suspended
	Suspended  VMStatus = "SUSPENDED"

	Rebooting VMStatus = "REBOOTING" // from running to running

	Termiating VMStatus = "TERMINATING" // from running, suspended to terminated
	Termiated  VMStatus = "TERMINATED"
)

/*
	Creating VMStatus = "Creating" // from launch to running
	Running  VMStatus = "Running"
	Suspending VMStatus = "Suspending" // from running to suspended
	Suspended  VMStatus = "Suspended"
	Resuming   VMStatus = "Resuming" // from suspended to running
	Rebooting VMStatus = "Rebooting" // from running to running
	Terminating VMStatus = "Terminating" // from running, suspended to terminated
	Terminated  VMStatus = "Terminated"
	Failed VMStatus = "Failed"
*/
//Cloud-Barista 기반의 VM Status로 변환 함.
func ConvertVMStatusString(vmStatus string) (irs.VMStatus, error) {
	var resultStatus string
	cblogger.Info("vmStatus : [%s]", vmStatus)

	if strings.EqualFold(vmStatus, "pending") {
		resultStatus = "Creating"
	} else if strings.EqualFold(vmStatus, "running") {
		resultStatus = "Running"
	} else if strings.EqualFold(vmStatus, "stopping") {
		resultStatus = "Suspending"
	} else if strings.EqualFold(vmStatus, "stopped") {
		resultStatus = "Suspended"
	} else if strings.EqualFold(vmStatus, "pending") {
		resultStatus = "Resuming"
	} else if strings.EqualFold(vmStatus, "Rebooting") {
		resultStatus = "Rebooting"
	} else if strings.EqualFold(vmStatus, "shutting-down") {
		resultStatus = "Terminating"
	} else if strings.EqualFold(vmStatus, "Terminated") {
		resultStatus = "Terminated"
	} else {
		//resultStatus = "Failed"
		return irs.VMStatus("Failed"), errors.New(vmStatus + "와 일치하는 CB VM 상태정보를 찾을 수 없습니다.")
	}
	return irs.VMStatus(resultStatus), nil
}

func main() {
	cblogger.Info("AWS Driver Test")
	/*
		cblogger.Info(ConvertVMStatusString("Creating"))
		cblogger.Info(ConvertVMStatusString("Running"))
		cblogger.Info(ConvertVMStatusString("Suspending"))
		cblogger.Info(ConvertVMStatusString("Suspended"))
		cblogger.Info(ConvertVMStatusString("Resuming"))
		cblogger.Info(ConvertVMStatusString("Rebooting"))
		cblogger.Info(ConvertVMStatusString("Terminating"))
		cblogger.Info(ConvertVMStatusString("Terminated"))
	*/
	//irs.VMStatus("")
	//status := irs.VMStatus("PenDing")
	/*
		status := VMStatus("PENDING")
		spew.Dump(status)
		spew.Dump(Pending)

		if status == Pending {
			cblogger.Info("같음")
		} else {
			cblogger.Info("다름")
		}
	*/

	//createVM()
	//suspendVM(vmID)
	//RebootVM
	//resumeVM(vmID)
	handleVM()
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
