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

	// ncpvpcdrv "github.com/cloud-barista/ncpvpc/ncpvpc"  // For local test
	ncpvpcdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncpvpc"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP VPC Resource Test")
	cblog.SetLevel("info")
}

func handleImage() {
	cblogger.Debug("Start ImageHandler Resource Test")

	ResourceHandler, err := getResourceHandler("Image")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.ImageHandler)

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ Image Management Test ]")
		fmt.Println("1. ListImage()")
		fmt.Println("2. GetImage()")
		fmt.Println("3. CheckWindowsImage()")		
		fmt.Println("4. CreateImage() (TBD)")
		fmt.Println("5. DeleteImage() (TBD)")
		fmt.Println("0. Quit")
		fmt.Println("\n   Select a number above!! : ")
		fmt.Println("============================================================================================")

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		imageReqInfo := irs.ImageReqInfo{
			IId: irs.IID{NameId: "Test OS Image", SystemId: "23214590"}, //NCP : ubuntu-22.04, Ubuntu Server 64-bit

			// NCP VPC 공공
			// IId: irs.IID{NameId: "Test OS Image", SystemId: "SW.VSVR.OS.LNX64.UBNTU.SVR2004.B050"}, //NCP VPC 공공 : Ubuntu Server 20.04 (64-bit)

			// IId: irs.IID{NameId: "Test OS Image", SystemId: "SW.VSVR.OS.WND64.WND.SVR2019EN.B100"}, //NCP : Windows Server 2019 (64-bit) English Edition			
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return

			case 1:
				cblogger.Infof("Image list 조회 테스트")

				result, err := handler.ListImage()
				if err != nil {
					cblogger.Infof(" Image list 조회 실패 : ", err)
				} else {
					fmt.Println("\n==================================================================================================================")
					cblogger.Info("Image list 조회 결과")
					
					fmt.Println("\n")
					spew.Dump(result)
					fmt.Println("\n")

					cblogger.Infof("전체 Image list 개수 : [%d]", len(result))

					//조회및 삭제 테스트를 위해 리스트의 첫번째 정보의 ID를 요청ID로 자동 갱신함.
					if result != nil {
						imageReqInfo.IId = result[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
					}
				}

				cblogger.Info("\nListImage Test Finished")

			case 2:
				cblogger.Infof("[%s] Image 조회 테스트", imageReqInfo.IId)

				result, err := handler.GetImage(imageReqInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] Image 조회 실패 : ", imageReqInfo.IId.SystemId, err)
				} else {
					fmt.Println("\n==================================================================================================================")
					cblogger.Infof("[%s] Image 조회 결과 : \n[%s]", imageReqInfo.IId.SystemId, result)

					fmt.Println("\n")
					spew.Dump(result)
				}

				cblogger.Info("\nGetImage Test Finished")

			case 3:
				cblogger.Infof("[%s] Image Check 테스트", imageReqInfo.IId.NameId)
				checkResult, err := handler.CheckWindowsImage(imageReqInfo.IId)
				if err != nil {
					cblogger.Infof(imageReqInfo.IId.NameId, " Image Check 실패 : ", err)
				} else {
					cblogger.Infof("Image 생성 결과 : ", checkResult)
					spew.Dump(checkResult)
				}

			case 4:
				cblogger.Infof("[%s] Image Check 테스트", imageReqInfo.IId.NameId)
				result, err := handler.CreateImage(imageReqInfo)
				if err != nil {
					cblogger.Infof(imageReqInfo.IId.NameId, " Image 생성 실패 : ", err)
				} else {
					cblogger.Infof("Image 생성 결과 : ", result)
					imageReqInfo.IId = result.IId // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			// case 5:
			// 	cblogger.Infof("[%s] Image 삭제 테스트", imageReqInfo.IId.NameId)
			// 	result, err := handler.DeleteImage(imageReqInfo.IId)
			// 	if err != nil {
			// 		cblogger.Infof("[%s] Image 삭제 실패 : ", imageReqInfo.IId.NameId, err)
			// 	} else {
			// 		cblogger.Infof("[%s] Image 삭제 결과 : [%s]", imageReqInfo.IId.NameId, result)
			// 	}
			}
		}
	}
}

func testErr() error {
	//return awserr.Error("")
	return errors.New("")
	// return ncloud.New("504", "찾을 수 없음", nil)
}

func main() {
	cblogger.Info("NCP VPC Resource Test")

	handleImage()
}

//handlerType : resources폴더의 xxxHandler.go에서 Handler이전까지의 문자열
//(예) ImageHandler.go -> "Image"
func getResourceHandler(handlerType string) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(ncpvpcdrv.NcpVpcDriver)

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
	} `yaml:"ncpvpc"`
}

func readConfigFile() Config {
	// # Set Environment Value of Project Root Path
	// goPath := os.Getenv("GOPATH")
	// rootPath := goPath + "/src/github.com/cloud-barista/ncp/ncp/main"
	// cblogger.Debugf("Test Config file : [%]", rootPath+"/config/config.yaml")
	rootPath 	:= os.Getenv("CBSPIDER_ROOT")
	configPath 	:= rootPath + "/cloud-control-manager/cloud-driver/drivers/ncpvpc/main/config/config.yaml"
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
	cblogger.Info("ConfigFile Loaded ...")

	// Just for test
	cblogger.Debug(config.Ncp.NcpAccessKeyID, " ", config.Ncp.Region)

	return config
}
