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

	"github.com/aws/aws-sdk-go/aws/awserr"

	awsdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/aws"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"

	//irs2 "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	cblog "github.com/cloud-barista/cb-log"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("AWS Resource Test")
	cblog.SetLevel("debug")
}

// Test SecurityHandler
func handleSecurity() {
	cblogger.Debug("Start handler")

	ResourceHandler, err := getResourceHandler("Security")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.SecurityHandler)

	config := readConfigFile()
	securityId := config.Aws.SecurityGroupID
	cblogger.Infof(securityId)
	//securityId = "sg-06c4523b969eaafc7"
	securityId = "cb-sgtest-mcloud-barista"

	//result, err := handler.GetSecurity(securityId)
	//result, err := handler.GetSecurity("sg-0fd2d90b269ebc082") // sgtest-mcloub-barista
	//result, err := handler.DeleteSecurity(securityId)
	//result, err := handler.ListSecurity()

	securityReqInfo := irs.SecurityReqInfo{
		Name: securityId,
		SecurityRules: &[]irs.SecurityRuleInfo{ //보안 정책 설정
			{
				FromPort:   "20",
				ToPort:     "22",
				IPProtocol: "tcp",
				Direction:  "inbound",
			},
			/*
				{
					FromPort:   "80",
					ToPort:     "80",
					IPProtocol: "tcp",
					Direction:  "inbound",
				},
				{
					FromPort:   "8080",
					ToPort:     "8080",
					IPProtocol: "tcp",
					Direction:  "inbound",
				},
				{
					FromPort:   "443",
					ToPort:     "443",
					IPProtocol: "tcp",
					Direction:  "outbound",
				},
				{
					FromPort:   "8443",
					ToPort:     "9999",
					IPProtocol: "tcp",
					Direction:  "outbound",
				},
				{
					//FromPort:   "8443",
					//ToPort:     "9999",
					IPProtocol: "-1", // 모두 허용 (포트 정보 없음)
					Direction:  "inbound",
				},
			*/
		},
	}

	cblogger.Info(securityReqInfo)
	result, err := handler.CreateSecurity(securityReqInfo)

	if err != nil {
		cblogger.Infof("보안 그룹 조회 실패 : ", err)
	} else {
		cblogger.Info("보안 그룹 조회 결과")
		//cblogger.Info(result)
		spew.Dump(result)
	}
}

// Test PublicIp
func handlePublicIP() {
	cblogger.Debug("Start Publicip Resource Test")

	ResourceHandler, err := getResourceHandler("Publicip")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.PublicIPHandler)

	config := readConfigFile()
	//reqGetPublicIP := "13.124.140.207"
	reqPublicIP := config.Aws.PublicIP
	reqPublicIP = "mcloud-barista-eip-test"
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
				/*
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
				*/

				result, err := handler.DeletePublicIP(reqPublicIP)
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

// Test KeyPair
func handleKeyPair() {
	cblogger.Debug("Start KeyPair Resource Test")

	KeyPairHandler, err := setKeyPairHandler()
	if err != nil {
		panic(err)
	}
	//config := readConfigFile()
	//VmID := config.Aws.VmID

	keyPairName := "CB-KeyPairTest123123"
	//keyPairName := config.Aws.KeyName

	for {
		fmt.Println("KeyPair Management")
		fmt.Println("0. Quit")
		fmt.Println("1. KeyPair List")
		fmt.Println("2. KeyPair Create")
		fmt.Println("3. KeyPair Get")
		fmt.Println("4. KeyPair Delete")

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
				result, err := KeyPairHandler.ListKey()
				if err != nil {
					cblogger.Infof(" 키 페어 목록 조회 실패 : ", err)
				} else {
					cblogger.Info("키 페어 목록 조회 결과")
					//cblogger.Info(result)
					spew.Dump(result)
				}

			case 2:
				cblogger.Infof("[%s] 키 페어 생성 테스트", keyPairName)
				keyPairReqInfo := irs.KeyPairReqInfo{
					Name: keyPairName,
				}
				result, err := KeyPairHandler.CreateKey(keyPairReqInfo)
				if err != nil {
					cblogger.Infof(keyPairName, " 키 페어 생성 실패 : ", err)
				} else {
					cblogger.Infof("[%s] 키 페어 생성 결과 : [%s]", keyPairName, result)
					spew.Dump(result)
				}
			case 3:
				cblogger.Infof("[%s] 키 페어 조회 테스트", keyPairName)
				result, err := KeyPairHandler.GetKey(keyPairName)
				if err != nil {
					cblogger.Infof(keyPairName, " 키 페어 조회 실패 : ", err)
				} else {
					cblogger.Infof("[%s] 키 페어 조회 결과 : [%s]", keyPairName, result)
				}
			case 4:
				cblogger.Infof("[%s] 키 페어 삭제 테스트", keyPairName)
				result, err := KeyPairHandler.DeleteKey(keyPairName)
				if err != nil {
					cblogger.Infof(keyPairName, " 키 페어 삭제 실패 : ", err)
				} else {
					cblogger.Infof("[%s] 키 페어 삭제 결과 : [%s]", keyPairName, result)
				}
			}
		}
	}
}

// Test handleVNetwork (VPC)
func handleVNetwork() {
	cblogger.Debug("Start VPC Resource Test")

	vNetworkHandler, err := setVNetworkHandler()
	if err != nil {
		panic(err)
	}

	vNetworkReqInfo := irs.VNetworkReqInfo{
		//Id:   "subnet-044a2b57145e5afc5",
		Name: "CB-VNet-Subnet", // 웹 도구 등 외부에서 전달 받지 않고 드라이버 내부적으로 자동 구현때문에 사용하지 않음.
		//CidrBlock: "10.0.0.0/16",
		//CidrBlock: "192.168.0.0/16",
	}
	reqSubnetId := "subnet-0b9ea37601d46d8fa"
	//reqSubnetId = ""

	for {
		fmt.Println("VNetworkHandler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. VNetwork List")
		fmt.Println("2. VNetwork Create")
		fmt.Println("3. VNetwork Get")
		fmt.Println("4. VNetwork Delete")

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
				result, err := vNetworkHandler.ListVNetwork()
				if err != nil {
					cblogger.Infof(" VNetwork 목록 조회 실패 : ", err)
				} else {
					cblogger.Info("VNetwork 목록 조회 결과")
					//cblogger.Info(result)
					spew.Dump(result)

					// 내부적으로 1개만 존재함.
					//조회및 삭제 테스트를 위해 리스트의 첫번째 서브넷 ID를 요청ID로 자동 갱신함.
					if result != nil {
						reqSubnetId = result[0].Id // 조회 및 삭제를 위해 생성된 ID로 변경
					}
				}

			case 2:
				cblogger.Infof("[%s] VNetwork 생성 테스트", vNetworkReqInfo.Name)
				//vNetworkReqInfo := irs.VNetworkReqInfo{}
				result, err := vNetworkHandler.CreateVNetwork(vNetworkReqInfo)
				if err != nil {
					cblogger.Infof(reqSubnetId, " VNetwork 생성 실패 : ", err)
				} else {
					cblogger.Infof("VNetwork 생성 결과 : ", result)
					reqSubnetId = result.Id // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] VNetwork 조회 테스트", reqSubnetId)
				result, err := vNetworkHandler.GetVNetwork(reqSubnetId)
				if err != nil {
					cblogger.Infof("[%s] VNetwork 조회 실패 : ", reqSubnetId, err)
				} else {
					cblogger.Infof("[%s] VNetwork 조회 결과 : [%s]", reqSubnetId, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] VNetwork 삭제 테스트", reqSubnetId)
				result, err := vNetworkHandler.DeleteVNetwork(reqSubnetId)
				if err != nil {
					cblogger.Infof("[%s] VNetwork 삭제 실패 : ", reqSubnetId, err)
				} else {
					cblogger.Infof("[%s] VNetwork 삭제 결과 : [%s]", reqSubnetId, result)
				}
			}
		}
	}
}

// Test AMI
func handleImage() {
	cblogger.Debug("Start ImageHandler Resource Test")

	ResourceHandler, err := getResourceHandler("Image")
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

// Test VNic
func handleVNic() {
	cblogger.Debug("Start VNicHandler Resource Test")

	ResourceHandler, err := getResourceHandler("VNic")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.VNicHandler)
	reqVnicID := "eni-093deb03ca6eb70eb"
	vNicReqInfo := irs.VNicReqInfo{
		Name: "TestCB-VNic2",
		SecurityGroupIds: []string{
			//"sg-0d4d11c090c4814e8", "sg-0dc15d050f8272e24",
			"sg-06c4523b969eaafc7",
		},
	}

	for {
		fmt.Println("VNicHandler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. VNic List")
		fmt.Println("2. VNic Create")
		fmt.Println("3. VNic Get")
		fmt.Println("4. VNic Delete")

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
				result, err := handler.ListVNic()
				if err != nil {
					cblogger.Infof(" VNic 목록 조회 실패 : ", err)
				} else {
					cblogger.Info("VNic 목록 조회 결과")
					spew.Dump(result)
					if len(result) > 0 {
						reqVnicID = result[0].Id // 조회 및 삭제 편의를 위해 목록의 첫번째 ID로 변경
					}
				}

			case 2:
				cblogger.Infof("[%s] VNic 생성 테스트", vNicReqInfo.Name)
				result, err := handler.CreateVNic(vNicReqInfo)
				if err != nil {
					cblogger.Infof(reqVnicID, " VNic 생성 실패 : ", err)
				} else {
					cblogger.Infof("VNic 생성 결과 : ", result)
					reqVnicID = result.Id // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] VNic 조회 테스트", reqVnicID)
				result, err := handler.GetVNic(reqVnicID)
				if err != nil {
					cblogger.Infof("[%s] VNic 조회 실패 : ", reqVnicID, err)
				} else {
					cblogger.Infof("[%s] VNic 조회 결과 : [%s]", reqVnicID, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] VNic 삭제 테스트", reqVnicID)
				result, err := handler.DeleteVNic(reqVnicID)
				if err != nil {
					cblogger.Infof("[%s] VNic 삭제 실패 : ", reqVnicID, err)
				} else {
					cblogger.Infof("[%s] VNic 삭제 결과 : [%s]", reqVnicID, result)
				}
			}
		}
	}
}

func testErr() error {
	//return awserr.Error("")
	//return errors.New("")
	return awserr.New("504", "찾을 수 없음", nil)
}

func main() {
	cblogger.Info("AWS Resource Test")
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

	//handleVNetwork() //VPC
	//handleKeyPair()
	handlePublicIP() // PublicIP 생성 후 conf
	//handleSecurity()

	//handleImage() //AMI
	//handleVNic() //Lancard

	/*
		KeyPairHandler, err := setKeyPairHandler()
		if err != nil {
			panic(err)
		}

		keyPairName := "test123"
		cblogger.Infof("[%s] 키 페어 조회 테스트", keyPairName)
		result, err := KeyPairHandler.GetKey(keyPairName)
		if err != nil {
			cblogger.Infof(keyPairName, " 키 페어 조회 실패 : ", err)
		} else {
			cblogger.Infof("[%s] 키 페어 조회 결과")
			spew.Dump(result)
		}
	*/
}

//handlerType : resources폴더의 xxxHandler.go에서 Handler이전까지의 문자열
//(예) ImageHandler.go -> "Image"
func getResourceHandler(handlerType string) (interface{}, error) {
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
	}

	if err != nil {
		return nil, err
	}
	return resourceHandler, nil
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

func setVNetworkHandler() (irs.VNetworkHandler, error) {
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

	handler, err := cloudConnection.CreateVNetworkHandler()
	if err != nil {
		return nil, err
	}
	return handler, nil
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

		PublicIP string `yaml:"public_ip"`
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
