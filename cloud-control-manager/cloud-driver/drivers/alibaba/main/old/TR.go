// Proof of Concepts of CB-Spider.
// The CB-Spider is sub-Framework of the Cloud-Barista Multi-Cloud Project.
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

	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	cblog "github.com/cloud-barista/cb-log"

	"io/ioutil"
	"os"

	alidrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/alibaba"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	"gopkg.in/yaml.v3"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("AlibabaCloud Resource Test")
	cblog.SetLevel("debug")
}

// Test PublicIp
func handlePublicIP() {
	cblogger.Debug("Start Publicip Resource Test")

	ResourceHandler, err := GetResourceHandler("Publicip")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.PublicIPHandler)

	config := ReadConfigFile()
	//reqGetPublicIP := "13.124.140.207"
	reqPublicIP := config.Ali.PublicIP
	//reqPublicIP = "eipalloc-0231a3e16ec42e869"
	cblogger.Info("reqPublicIP : ", reqPublicIP)
	//handler.CreatePublicIP(publicIPReqInfo)
	//handler.ListPublicIP()
	//handler.GetPublicIP("13.124.140.207")

	for {
		fmt.Println("")
		fmt.Println("Publicip Resource Test")
		fmt.Println("1. ListPublicIP()")
		fmt.Println("2. GetPublicIP()")
		fmt.Println("3. CreatePublicIP()")
		fmt.Println("4. DeletePublicIP()")
		fmt.Println("5. Exit")

		var commandNum int
		var reqDelIP string

		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				fmt.Println("Start ListPublicIP() ...")
				result, err := handler.ListPublicIP()
				if err != nil {
					cblogger.Error("PublicIP 목록 조회 실패 : ", err)
				} else {
					cblogger.Info("PublicIP 목록 조회 결과")
					spew.Dump(result)
				}

				fmt.Println("Finish ListPublicIP()")

			case 2:
				fmt.Println("Start GetPublicIP() ...")
				result, err := handler.GetPublicIP(reqPublicIP)
				if err != nil {
					cblogger.Error(reqPublicIP, " PublicIP 정보 조회 실패 : ", err)
				} else {
					cblogger.Infof("PublicIP[%s]  정보 조회 결과", reqPublicIP)
					spew.Dump(result)
				}
				fmt.Println("Finish GetPublicIP()")

			case 3:
				fmt.Println("Start CreatePublicIP() ...")
				reqInfo := irs.PublicIPReqInfo{Name: "mcloud-barista-eip-test"}
				result, err := handler.CreatePublicIP(reqInfo)
				if err != nil {
					cblogger.Error("PublicIP 생성 실패 : ", err)
				} else {
					cblogger.Info("PublicIP 생성 성공 ", result)
					spew.Dump(result)
				}
				fmt.Println("Finish CreatePublicIP()")

			case 4:
				fmt.Println("Start DeletePublicIP() ...")
				fmt.Print("삭제할 PublicIP를 입력하세요 : ")
				inputCnt, err := fmt.Scan(&reqDelIP)
				if err != nil {
					panic(err)
				}

				if inputCnt == 1 {
					cblogger.Info("삭제할 PublicIP : ", reqDelIP)
				} else {
					fmt.Println("삭제할 Public IP만 입력하세요.")
				}

				result, err := handler.DeletePublicIP(reqDelIP)
				if err != nil {
					cblogger.Error(reqDelIP, " PublicIP 삭제 실패 : ", err)
				} else {
					if result {
						cblogger.Infof("PublicIP[%s] 삭제 완료", reqDelIP)
					} else {
						cblogger.Errorf("PublicIP[%s] 삭제 실패", reqDelIP)
					}
				}
				fmt.Println("Finish DeletePublicIP()")

			case 5:
				fmt.Println("Exit")
				return
			}
		}
	}
}

// Test VMSpec
func handleVMSpec() {
	cblogger.Debug("Start VMSpec Resource Test")

	ResourceHandler, err := GetResourceHandler("VMSpec")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.VMSpecHandler)

	config := ReadConfigFile()
	//reqVMSpec := config.Ali.VMSpec
	//reqVMSpec := "ecs.g6.large"	// GPU가 없음
	reqVMSpec := "ecs.vgn5i-m8.4xlarge" // GPU 1개
	//reqVMSpec := "ecs.gn6i-c24g1.24xlarge" // GPU 4개

	reqRegion := config.Ali.Region
	reqRegion = "us-east-1"
	cblogger.Info("reqVMSpec : ", reqVMSpec)

	for {
		fmt.Println("")
		fmt.Println("VMSpec Resource Test")
		fmt.Println("1. ListVMSpec()")
		fmt.Println("2. GetVMSpec()")
		fmt.Println("3. ListOrgVMSpec()")
		fmt.Println("4. GetOrgVMSpec()")
		fmt.Println("0. Exit")

		var commandNum int
		//var reqDelIP string

		/*
			inputCnt, err := fmt.Scan(&commandNum)
			if err != nil {
				panic(err)
			}
		*/

		inputCnt := 1
		commandNum = 1

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				fmt.Println("Start ListVMSpec() ...")
				result, err := handler.ListVMSpec(reqRegion)
				if err != nil {
					cblogger.Error("VMSpec 목록 조회 실패 : ", err)
				} else {
					cblogger.Info("VMSpec 목록 조회 결과")
					spew.Dump(result)
				}

				fmt.Println("Finish ListVMSpec()")

			case 2:
				fmt.Println("Start GetVMSpec() ...")
				result, err := handler.GetVMSpec(reqRegion, reqVMSpec)
				if err != nil {
					cblogger.Error(reqVMSpec, " VMSpec 정보 조회 실패 : ", err)
				} else {
					cblogger.Infof("VMSpec[%s]  정보 조회 결과", reqVMSpec)
					spew.Dump(result)
				}
				fmt.Println("Finish GetVMSpec()")

			case 3:
				fmt.Println("Start ListOrgVMSpec() ...")
				result, err := handler.ListOrgVMSpec(reqRegion)
				if err != nil {
					cblogger.Error("VMSpec 목록 조회 실패 : ", err)
				} else {
					cblogger.Info("VMSpec 목록 조회 결과")
					cblogger.Info(result)
					//spew.Dump(result)
				}

				fmt.Println("Finish ListOrgVMSpec()")

			case 4:
				fmt.Println("Start GetOrgVMSpec() ...")
				result, err := handler.GetOrgVMSpec(reqRegion, reqVMSpec)
				if err != nil {
					cblogger.Error(reqVMSpec, " VMSpec 정보 조회 실패 : ", err)
				} else {
					cblogger.Infof("VMSpec[%s]  정보 조회 결과", reqVMSpec)
					spew.Dump(result)
				}
				fmt.Println("Finish GetOrgVMSpec()")

			case 0:
				fmt.Println("Exit")
				return
			}
		}
	}
}

// Test AMI
func handleImage() {
	cblogger.Debug("Start ImageHandler Resource Test")

	ResourceHandler, err := GetResourceHandler("Image")
	if err != nil {
		panic(err)
	}
	//handler := ResourceHandler.(irs2.ImageHandler)
	handler := ResourceHandler.(irs.ImageHandler)

	//imageReqInfo := irs2.ImageReqInfo{
	imageReqInfo := irs.ImageReqInfo{
		Id:   "ami-047f7b46bd6dd5d84",
		Name: "Test OS Image",
	}

	for {
		fmt.Println("ImageHandler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. Image List")
		fmt.Println("2. Image Create")
		fmt.Println("3. Image Get")
		fmt.Println("4. Image Delete")

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
				result, err := handler.ListImage()
				if err != nil {
					cblogger.Infof(" Image 목록 조회 실패 : ", err)
				} else {
					cblogger.Info("Image 목록 조회 결과")
					cblogger.Info(result)
					cblogger.Info("출력 결과 수 : ", len(result))
					//spew.Dump(result)

					//조회및 삭제 테스트를 위해 리스트의 첫번째 정보의 ID를 요청ID로 자동 갱신함.
					if result != nil {
						imageReqInfo.Id = result[0].Id // 조회 및 삭제를 위해 생성된 ID로 변경
					}
				}

			case 2:
				cblogger.Infof("[%s] Image 생성 테스트", imageReqInfo.Name)
				//vNetworkReqInfo := irs.VNetworkReqInfo{}
				result, err := handler.CreateImage(imageReqInfo)
				if err != nil {
					cblogger.Infof(imageReqInfo.Id, " Image 생성 실패 : ", err)
				} else {
					cblogger.Infof("Image 생성 결과 : ", result)
					imageReqInfo.Id = result.Id // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] Image 조회 테스트", imageReqInfo.Id)
				result, err := handler.GetImage(imageReqInfo.Id)
				if err != nil {
					cblogger.Infof("[%s] Image 조회 실패 : ", imageReqInfo.Id, err)
				} else {
					cblogger.Infof("[%s] Image 조회 결과 : [%s]", imageReqInfo.Id, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] Image 삭제 테스트", imageReqInfo.Id)
				result, err := handler.DeleteImage(imageReqInfo.Id)
				if err != nil {
					cblogger.Infof("[%s] Image 삭제 실패 : ", imageReqInfo.Id, err)
				} else {
					cblogger.Infof("[%s] Image 삭제 결과 : [%s]", imageReqInfo.Id, result)
				}
			}
		}
	}
}

func main() {
	cblogger.Info("Alibaba Cloud Resource Test")
	//handleKeyPair()
	//handlePublicIP() // PublicIP 생성 후 conf
	//handleVMSpec()

	//handleVNetwork() //VPC
	//handleImage() //AMI
	//handleVNic() //Lancard
	//handleSecurity()

	TestMain()
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
	Ali struct {
		AliAccessKeyID     string `yaml:"ali_access_key_id"`
		AliSecretAccessKey string `yaml:"ali_secret_access_key"`
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

		PublicIP string `yaml:"public_ip"`
	} `yaml:"ali"`
}

//환경 설정 파일 읽기
//환경변수 CBSPIDER_PATH 설정 후 해당 폴더 하위에 /config/config.yaml 파일 생성해야 함.
func ReadConfigFile() Config {
	// Set Environment Value of Project Root Path
	rootPath := os.Getenv("CBSPIDER_PATH")
	//rootpath := "D:/Workspace/mcloud-barista-config"
	// /mnt/d/Workspace/mcloud-barista-config/config/config.yaml
	cblogger.Debugf("Test Data 설정파일 : [%]", rootPath+"/config/configAli.yaml")

	data, err := ioutil.ReadFile(rootPath + "/config/configAli.yaml")
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
	//cblogger.Info(config)
	return config
}

//handlerType : resources폴더의 xxxHandler.go에서 Handler이전까지의 문자열
//(예) ImageHandler.go -> "Image"
func GetResourceHandler(handlerType string) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(alidrv.AlibabaDriver)

	config := ReadConfigFile()
	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			ClientId:     config.Ali.AliAccessKeyID,
			ClientSecret: config.Ali.AliSecretAccessKey,
		},
		RegionInfo: idrv.RegionInfo{
			Region: config.Ali.Region,
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
	case "Publicip":
		resourceHandler, err = cloudConnection.CreatePublicIPHandler()
	case "Security":
		resourceHandler, err = cloudConnection.CreateSecurityHandler()
	case "VNetwork":
		resourceHandler, err = cloudConnection.CreateVNetworkHandler()
	case "VNic":
		resourceHandler, err = cloudConnection.CreateVNicHandler()
	case "VMSpec":
		resourceHandler, err = cloudConnection.CreateVMSpecHandler()
	}

	if err != nil {
		return nil, err
	}
	return resourceHandler, nil
}

func TestMain() {
	cblogger.Debug("Start ImageHandler Resource Test")

	ResourceHandler, err := GetResourceHandler("Image")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.ImageHandler)

	result, err := handler.ListImage()
	if err != nil {
		cblogger.Infof(" Image 목록 조회 실패 : ", err)
	} else {
		cblogger.Info("Image 목록 조회 결과")
		cblogger.Info(result)
		cblogger.Info("출력 결과 수 : ", len(result))
		spew.Dump(result)
	}

}
