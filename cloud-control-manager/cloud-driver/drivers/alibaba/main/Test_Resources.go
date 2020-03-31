<<<<<<< HEAD
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

	testconf "./conf"

	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	cblog "github.com/cloud-barista/cb-log"
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

	ResourceHandler, err := testconf.GetResourceHandler("Publicip")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.PublicIPHandler)

	config := testconf.ReadConfigFile()
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

	ResourceHandler, err := testconf.GetResourceHandler("VMSpec")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.VMSpecHandler)

	config := testconf.ReadConfigFile()
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

		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

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
					spew.Dump(result)
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

func main() {
	cblogger.Info("Alibaba Cloud Resource Test")
	//handleKeyPair()
	//handlePublicIP() // PublicIP 생성 후 conf
	handleVMSpec()

	//handleVNetwork() //VPC
	//handleImage() //AMI
	//handleVNic() //Lancard
	//handleSecurity()
}
=======
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

	//testconf "./conf"
	testconf "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/alibaba/main/conf"

	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	cblog "github.com/cloud-barista/cb-log"
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

	ResourceHandler, err := testconf.GetResourceHandler("Publicip")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.PublicIPHandler)

	config := testconf.ReadConfigFile()
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

	ResourceHandler, err := testconf.GetResourceHandler("VMSpec")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.VMSpecHandler)

	config := testconf.ReadConfigFile()
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

		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

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
					cblogger.Info(result)
					//spew.Dump(result)
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

	ResourceHandler, err := testconf.GetResourceHandler("Image")
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
					//spew.Dump(result)
					cblogger.Info("출력 결과 수 : ", len(result))

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

// Test SecurityHandler
func handleSecurity() {
	cblogger.Debug("Start handler")

	ResourceHandler, err := testconf.GetResourceHandler("Security")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.SecurityHandler)

	config := testconf.ReadConfigFile()
	securityId := config.Ali.SecurityGroupID
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

// Test KeyPair
func handleKeyPair() {
	cblogger.Debug("Start KeyPair Resource Test")

	ResourceHandler, err := testconf.GetResourceHandler("KeyPair")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.KeyPairHandler)

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
				result, err := handler.ListKey()
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
				result, err := handler.CreateKey(keyPairReqInfo)
				if err != nil {
					cblogger.Infof(keyPairName, " 키 페어 생성 실패 : ", err)
				} else {
					cblogger.Infof("[%s] 키 페어 생성 결과 : [%s]", keyPairName, result)
					spew.Dump(result)
				}
			case 3:
				cblogger.Infof("[%s] 키 페어 조회 테스트", keyPairName)
				result, err := handler.GetKey(keyPairName)
				if err != nil {
					cblogger.Infof(keyPairName, " 키 페어 조회 실패 : ", err)
				} else {
					cblogger.Infof("[%s] 키 페어 조회 결과 : [%s]", keyPairName, result)
				}
			case 4:
				cblogger.Infof("[%s] 키 페어 삭제 테스트", keyPairName)
				result, err := handler.DeleteKey(keyPairName)
				if err != nil {
					cblogger.Infof(keyPairName, " 키 페어 삭제 실패 : ", err)
				} else {
					cblogger.Infof("[%s] 키 페어 삭제 결과 : [%s]", keyPairName, result)
				}
			}
		}
	}
}

func TestMain() {
	cblogger.Debug("Start ImageHandler Resource Test")

	ResourceHandler, err := testconf.GetResourceHandler("Image")
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

func main() {
	cblogger.Info("Alibaba Cloud Resource Test")
	//handleKeyPair()
	//handlePublicIP() // PublicIP 생성 후 conf
	//handleVMSpec()

	//handleVNetwork() //VPC
	handleImage() //AMI
	//handleVNic() //Lancard
	//handleSecurity()
}
>>>>>>> 1fc066dc9d23ee89b34ba142dedd9f50e9b16c77
