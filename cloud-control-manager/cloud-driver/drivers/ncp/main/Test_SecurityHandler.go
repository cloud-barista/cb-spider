// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Tester Example.
//
// by ETRI, 2020.12.

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

func handleSecurity() {
	cblogger.Debug("Start Security Resource Test")

	ResourceHandler, err := getResourceHandler("Security")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.SecurityHandler)

	//config := readConfigFile()
	//VmID := config.Ncp.VmID

	for {
		fmt.Println("\n============================================================================================")
		fmt.Println("[ Security Management Test ]")

		// Related APIs on NCP classic services are not supported.
		fmt.Println("1. Create Security")
		fmt.Println("2. List Security")
		fmt.Println("3. Get Security")
		fmt.Println("4. Delete Security")
		fmt.Println("0. Quit")
		fmt.Println("\n   Select a number above!! : ")
		fmt.Println("============================================================================================")

		var commandNum int

		securityName := "ncp-sg01"
		securityId := "661762" //NCP default S/G
		// securityId := "214436"
		// vpcId := "vpc-c0479cab"

		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return

			case 1:
				reqInfo := irs.SecurityReqInfo{
					IId: irs.IID{
						NameId: securityName,
					},
				}
				result, err := handler.CreateSecurity(reqInfo)
				if err != nil {
					cblogger.Infof(" Security 생성 실패 : ", err)
				} else {
					cblogger.Info("Security 생성 결과")
					//cblogger.Info(result)
					spew.Dump(result)
					//					if result != nil {
					//						securityId = result.IId.SystemId // 조회 및 삭제를 위해 생성된 ID로 변경
					//						spew.Dump(securityId)
					//					}
				}

				cblogger.Info("\nCreateSecurity() Test Finished")

			case 2:
				result, err := handler.ListSecurity()
				if err != nil {
					cblogger.Infof(" Security list 조회 실패 : ", err)
				} else {
					cblogger.Info("Security list 조회 결과")
					//cblogger.Info(result)
					spew.Dump(result)
					cblogger.Infof("=========== S/G 목록 수 : [%d] ================", len(result))
					if result != nil {
						securityId = result[0].IId.SystemId // 조회 및 삭제를 위해 생성된 ID로 변경
					}
				}

				cblogger.Info("\nListSecurity Test Finished")

			case 3:
				cblogger.Infof("[%s] Security 조회 테스트", securityId)
				result, err := handler.GetSecurity(irs.IID{SystemId: securityId})
				if err != nil {
					cblogger.Infof(securityId, " Security 조회 실패 : ", err)
				} else {
					cblogger.Infof("[%s] Security 조회 결과 : [%v]", securityId, result)
					spew.Dump(result)
				}

				cblogger.Info("\nGetSecurity Test Finished")

				// case 3:
				// 	cblogger.Infof("[%s] Security 생성 테스트", securityName)

				// 	securityReqInfo := irs.SecurityReqInfo{
				// 		IId:    irs.IID{NameId: securityName},
				// 		VpcIID: irs.IID{SystemId: vpcId},
				// 		SecurityRules: &[]irs.SecurityRuleInfo{ //보안 정책 설정
				// 			{
				// 				FromPort:   "20",
				// 				ToPort:     "22",
				// 				IPProtocol: "tcp",
				// 				Direction:  "inbound",
				// 			},

				// 			{
				// 				FromPort:   "80",
				// 				ToPort:     "80",
				// 				IPProtocol: "tcp",
				// 				Direction:  "inbound",
				// 			},
				// 			{
				// 				FromPort:   "8080",
				// 				ToPort:     "8080",
				// 				IPProtocol: "tcp",
				// 				Direction:  "inbound",
				// 			},
				// 			{
				// 				FromPort:   "-1",
				// 				ToPort:     "-1",
				// 				IPProtocol: "icmp",
				// 				Direction:  "inbound",
				// 			},
				// 			{
				// 				FromPort:   "443",
				// 				ToPort:     "443",
				// 				IPProtocol: "tcp",
				// 				Direction:  "outbound",
				// 			},
				// 			{
				// 				FromPort:   "8443",
				// 				ToPort:     "9999",
				// 				IPProtocol: "tcp",
				// 				Direction:  "outbound",
				// 			},
				// 			/*
				// 				{
				// 					//FromPort:   "8443",
				// 					//ToPort:     "9999",
				// 					IPProtocol: "-1", // 모두 허용 (포트 정보 없음)
				// 					Direction:  "inbound",
				// 				},
				// 			*/
				// 		},
				// 	}

				// 	result, err := handler.CreateSecurity(securityReqInfo)
				// 	if err != nil {
				// 		cblogger.Infof(securityName, " Security 생성 실패 : ", err)
				// 	} else {
				// 		cblogger.Infof("[%s] Security 생성 결과 : [%v]", securityName, result)
				// 		spew.Dump(result)
				// 	}

				case 4:
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
	//return awserr.Error("")
	return errors.New("")
	// return ncloud.New("504", "찾을 수 없음", nil)
}

func main() {
	cblogger.Info("NCP Resource Test")
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
	cblogger.Info(config.Ncp.NcpAccessKeyID)
	cblogger.Info(config.Ncp.NcpSecretKey)

	cloudConnection, errCon := cloudDriver.ConnectCloud(connectionInfo)
	if errCon != nil {
		return nil, errCon
	}

	var resourceHandler interface{}
	var err error

	switch handlerType {
	case "Image":
		resourceHandler, err = cloudConnection.CreateImageHandler()
	//case "Publicip":
	//	resourceHandler, err = cloudConnection.CreatePublicIPHandler()
	case "Security":
		resourceHandler, err = cloudConnection.CreateSecurityHandler()
	case "VNetwork":
		resourceHandler, err = cloudConnection.CreateVPCHandler()
		//case "VNic":
		//	resourceHandler, err = cloudConnection.CreateVNicHandler()
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

//환경 설정 파일 읽기
//환경변수 CBSPIDER_PATH 설정 후 해당 폴더 하위에 /config/config.yaml 파일 생성해야 함.
func readConfigFile() Config {
	// Set Environment Value of Project Root Path
	goPath := os.Getenv("GOPATH")
	rootPath := goPath + "/src/github.com/cloud-barista/ncp/ncp/main"
	//rootpath := "D:/Workspace/mcloud-barista-config"
	// /mnt/d/Workspace/mcloud-barista-config/config/config.yaml
	cblogger.Debugf("Test Data 설정파일 : [%]", rootPath+"/config/config.yaml")

	data, err := os.ReadFile(rootPath + "/config/config.yaml")
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

	// NOTE Just for test
	// cblogger.Info(config.Ncp.NcpAccessKeyID)
	// cblogger.Info(config.Ncp.NcpSecretKey)

	// NOTE Just for test
	cblogger.Debug(config.Ncp.NcpAccessKeyID, " ", config.Ncp.Region)

	return config
}
