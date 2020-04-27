// Proof of Concepts of CB-Spider.
// The CB-Spider is sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by devunet@mz.co.kr, 2019.11.

package main

import (
	"fmt"

	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	testconf "./conf"
	cblog "github.com/cloud-barista/cb-log"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("GCP Resource Test")
	cblog.SetLevel("debug")
}

// Test PublicIp
// func handlePublicIP() {
// 	cblogger.Info("Start Publicip Resource Test")

// 	ResourceHandler, err := testconf.GetResourceHandler("Publicip")
// 	if err != nil {
// 		panic(err)
// 	}

// 	handler := ResourceHandler.(irs.PublicIPHandler)

// 	reqPublicIP := "publicip-vm01"
// 	cblogger.Info("reqPublicIP : ", reqPublicIP)

// 	for {
// 		fmt.Println("")
// 		fmt.Println("Publicip Resource Test")
// 		fmt.Println("1. ListPublicIP()")
// 		fmt.Println("2. GetPublicIP()")
// 		fmt.Println("3. CreatePublicIP()")
// 		fmt.Println("4. DeletePublicIP()")
// 		fmt.Println("5. Exit")

// 		var commandNum int
// 		var reqDelIP string

// 		inputCnt, err := fmt.Scan(&commandNum)
// 		if err != nil {
// 			panic(err)
// 		}

// 		if inputCnt == 1 {
// 			switch commandNum {
// 			case 1:
// 				fmt.Println("Start ListPublicIP() ...")
// 				result, err := handler.ListPublicIP()
// 				if err != nil {
// 					cblogger.Error("PublicIP 목록 조회 실패 : ", err)
// 				} else {
// 					cblogger.Info("PublicIP 목록 조회 결과")
// 					spew.Dump(result)
// 				}

// 				fmt.Println("Finish ListPublicIP()")

// 			case 2:
// 				fmt.Println("Start GetPublicIP() ...")
// 				result, err := handler.GetPublicIP(reqPublicIP)
// 				if err != nil {
// 					cblogger.Error(reqPublicIP, " PublicIP 정보 조회 실패 : ", err)
// 				} else {
// 					cblogger.Infof("PublicIP[%s]  정보 조회 결과", reqPublicIP)
// 					spew.Dump(result)
// 				}
// 				fmt.Println("Finish GetPublicIP()")

// 			case 3:
// 				fmt.Println("Start CreatePublicIP() ...")
// 				reqInfo := irs.PublicIPReqInfo{Name: "mcloud-barista-eip-test"}
// 				result, err := handler.CreatePublicIP(reqInfo)
// 				if err != nil {
// 					cblogger.Error("PublicIP 생성 실패 : ", err)
// 				} else {
// 					cblogger.Info("PublicIP 생성 성공 ", result)
// 					spew.Dump(result)
// 				}
// 				fmt.Println("Finish CreatePublicIP()")

// 			case 4:
// 				fmt.Println("Start DeletePublicIP() ...")
// 				fmt.Print("삭제할 PublicIP를 입력하세요 : ")
// 				inputCnt, err := fmt.Scan(&reqDelIP)
// 				if err != nil {
// 					panic(err)
// 				}

// 				if inputCnt == 1 {
// 					cblogger.Info("삭제할 PublicIP : ", reqDelIP)
// 				} else {
// 					fmt.Println("삭제할 Public IP만 입력하세요.")
// 				}

// 				result, err := handler.DeletePublicIP(reqDelIP)
// 				if err != nil {
// 					cblogger.Error(reqDelIP, " PublicIP 삭제 실패 : ", err)
// 				} else {
// 					if result {
// 						cblogger.Infof("PublicIP[%s] 삭제 완료", reqDelIP)
// 					} else {
// 						cblogger.Errorf("PublicIP[%s] 삭제 실패", reqDelIP)
// 					}
// 				}
// 				fmt.Println("Finish DeletePublicIP()")

// 			case 5:
// 				fmt.Println("Exit")
// 				return
// 			}
// 		}
// 	}
// }

// Test SecurityHandler
func handleSecurity() {
	cblogger.Debug("Start handler")

	ResourceHandler, err := testconf.GetResourceHandler("Security")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.SecurityHandler)

	securityId := "europe-west1"
	cblogger.Infof(securityId)

	//result, err := handler.GetSecurity(securityId)
	//result, err := handler.GetSecurity("sg-0d4d11c090c4814e8")
	//result, err := handler.GetSecurity("sg-0fd2d90b269ebc082") // sgtest-mcloub-barista
	//result, err := handler.DeleteSecurity(securityId)
	//result, err := handler.ListSecurity()

	securityReqInfo := irs.SecurityReqInfo{
		IId: irs.IID{
			NameId:   securityId,
			SystemId: securityId,
		},

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

// Test AMI
func handleImage() {
	cblogger.Debug("Start ImageHandler Resource Test")

	ResourceHandler, err := testconf.GetResourceHandler("Image")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.ImageHandler)

	//imageReqInfo := irs2.ImageReqInfo{
	imageReqInfo := irs.ImageReqInfo{
		IId: irs.IID{
			NameId:   "Test OS Image",
			SystemId: "vmsg02-asia-northeast1-b",
		},
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
						imageReqInfo.IId = result[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
					}
				}

			case 2:
				cblogger.Infof("[%s] Image 생성 테스트", imageReqInfo.IId.NameId)
				//vNetworkReqInfo := irs.VNetworkReqInfo{}
				result, err := handler.CreateImage(imageReqInfo)
				if err != nil {
					cblogger.Infof(imageReqInfo.IId.NameId, " Image 생성 실패 : ", err)
				} else {
					cblogger.Infof("Image 생성 결과 : ", result)
					imageReqInfo.IId = result.IId // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] Image 조회 테스트", imageReqInfo.IId)
				result, err := handler.GetImage(imageReqInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] Image 조회 실패 : ", imageReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] Image 조회 결과 : [%s]", imageReqInfo.IId.NameId, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] Image 삭제 테스트", imageReqInfo.IId)
				result, err := handler.DeleteImage(imageReqInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] Image 삭제 실패 : ", imageReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] Image 삭제 결과 : [%s]", imageReqInfo.IId.NameId, result)
				}
			}
		}
	}
}

// Test handleVNetwork (VPC)
// func handleVNetwork() {
// 	cblogger.Debug("Start VPC Resource Test")

// 	ResourceHandler, err := testconf.GetResourceHandler("VNetwork")
// 	if err != nil {
// 		panic(err)
// 	}
// 	handler := ResourceHandler.(irs.VNetworkHandler)

// 	vNetworkReqInfo := irs.VNetworkReqInfo{
// 		Name: "cb-subnet3", // 웹 도구 등 외부에서 전달 받지 않고 드라이버 내부적으로 자동 구현때문에 사용하지 않음.
// 	}
// 	reqSubnetId := "subnet-12345"
// 	//reqSubnetId = ""

// 	for {
// 		fmt.Println("VNetworkHandler Management")
// 		fmt.Println("0. Quit")
// 		fmt.Println("1. VNetwork List")
// 		fmt.Println("2. VNetwork Create")
// 		fmt.Println("3. VNetwork Get")
// 		fmt.Println("4. VNetwork Delete")

// 		var commandNum int
// 		inputCnt, err := fmt.Scan(&commandNum)
// 		if err != nil {
// 			panic(err)
// 		}

// 		if inputCnt == 1 {
// 			switch commandNum {
// 			case 0:
// 				return

// 			case 1:
// 				result, err := handler.ListVNetwork()
// 				if err != nil {
// 					cblogger.Infof(" VNetwork 목록 조회 실패 : ", err)
// 				} else {
// 					cblogger.Info("VNetwork 목록 조회 결과")
// 					//cblogger.Info(result)
// 					spew.Dump(result)

// 					// 내부적으로 1개만 존재함.
// 					//조회및 삭제 테스트를 위해 리스트의 첫번째 서브넷 ID를 요청ID로 자동 갱신함.
// 					if result != nil {
// 						reqSubnetId = result[0].Id // 조회 및 삭제를 위해 생성된 ID로 변경
// 					}
// 				}

// 			case 2:
// 				cblogger.Infof("[%s] VNetwork 생성 테스트", vNetworkReqInfo.Name)
// 				//vNetworkReqInfo := irs.VNetworkReqInfo{}
// 				result, err := handler.CreateVNetwork(vNetworkReqInfo)
// 				if err != nil {
// 					cblogger.Infof(reqSubnetId, " VNetwork 생성 실패 : ", err)
// 				} else {
// 					cblogger.Infof("VNetwork 생성 결과 : ", result)
// 					reqSubnetId = result.Id // 조회 및 삭제를 위해 생성된 ID로 변경
// 					spew.Dump(result)
// 				}

// 			case 3:
// 				cblogger.Infof("[%s] VNetwork 조회 테스트", reqSubnetId)
// 				result, err := handler.GetVNetwork(reqSubnetId)
// 				if err != nil {
// 					cblogger.Infof("[%s] VNetwork 조회 실패 : ", reqSubnetId, err)
// 				} else {
// 					cblogger.Infof("[%s] VNetwork 조회 결과 : [%s]", reqSubnetId, result)
// 					spew.Dump(result)
// 				}

// 			case 4:
// 				cblogger.Infof("[%s] VNetwork 삭제 테스트", reqSubnetId)
// 				result, err := handler.DeleteVNetwork(reqSubnetId)
// 				if err != nil {
// 					cblogger.Infof("[%s] VNetwork 삭제 실패 : ", reqSubnetId, err)
// 				} else {
// 					cblogger.Infof("[%s] VNetwork 삭제 결과 : [%s]", reqSubnetId, result)
// 				}
// 			}
// 		}
// 	}
// }

// Test handleVPC (VPC)
func handleVPC() {
	cblogger.Debug("Start VPC Resource Test")

	ResourceHandler, err := testconf.GetResourceHandler("VPCHandler")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.VPCHandler)
	vpcName := "cb-vpc"
	subnetList := []irs.SubnetInfo{
		{
			IId: irs.IID{
				NameId:   "cb-sub1",
				SystemId: "cb-sub1",
			},
			IPv4_CIDR: "10.0.3.0/24",
		},
		{
			IId: irs.IID{
				NameId:   "cb-sub2",
				SystemId: "cb-sub2",
			},
			IPv4_CIDR: "10.0.4.0/24",
		},
	}
	vNetworkReqInfo := irs.VPCReqInfo{
		IId: irs.IID{
			NameId:   vpcName,
			SystemId: vpcName,
		},
		SubnetInfoList: subnetList,
	}
	//reqSubnetId := "subnet-12345"
	reqSubnetId := irs.IID{
		NameId:   "cb-vpc",
		SystemId: "cb-vpc",
	}

	for {
		fmt.Println("VPCHandler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. VPC List")
		fmt.Println("2. VPC Create")
		fmt.Println("3. VPC Get")
		fmt.Println("4. VPC Delete")

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
				result, err := handler.ListVPC()
				if err != nil {
					cblogger.Infof(" VNetwork 목록 조회 실패 : ", err)
				} else {
					cblogger.Info("VPC 목록 조회 결과")
					//cblogger.Info(result)
					spew.Dump(result)

					// 내부적으로 1개만 존재함.
					//조회및 삭제 테스트를 위해 리스트의 첫번째 서브넷 ID를 요청ID로 자동 갱신함.
					if result != nil {
						//reqSubnetId = result[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
					}
				}

			case 2:
				cblogger.Infof("[%s] VNetwork 생성 테스트", vNetworkReqInfo.IId.NameId)
				//vNetworkReqInfo := irs.VNetworkReqInfo{}
				result, err := handler.CreateVPC(vNetworkReqInfo)
				if err != nil {
					cblogger.Infof(vNetworkReqInfo.IId.NameId, " VNetwork 생성 실패 : ", err)
				} else {
					cblogger.Infof("VNetwork 생성 결과 : ", result)
					reqSubnetId = result.IId // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] VNetwork 조회 테스트", reqSubnetId.NameId)
				result, err := handler.GetVPC(reqSubnetId)
				if err != nil {
					cblogger.Infof("[%s] VNetwork 조회 실패 : ", reqSubnetId.NameId, err)
				} else {
					cblogger.Infof("[%s] VNetwork 조회 결과 : [%s]", reqSubnetId.NameId, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] VNetwork 삭제 테스트", reqSubnetId.NameId)
				result, err := handler.DeleteVPC(reqSubnetId)
				if err != nil {
					cblogger.Infof("[%s] VNetwork 삭제 실패 : ", reqSubnetId.NameId, err)
				} else {
					cblogger.Infof("[%s] VNetwork 삭제 결과 : [%s]", reqSubnetId.NameId, result)
				}
			}
		}
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

	keyPairName := "cb-keyPairTest"
	keyReq := irs.IID{
		NameId:   keyPairName,
		SystemId: keyPairName,
	}

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
					IId: irs.IID{
						NameId:   keyPairName,
						SystemId: keyPairName,
					},
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
				result, err := handler.GetKey(keyReq)
				if err != nil {
					cblogger.Infof(keyPairName, " 키 페어 조회 실패 : ", err)
				} else {
					cblogger.Infof("[%s] 키 페어 조회 결과 : [%s]", keyPairName, result)
					spew.Dump(result)
				}
			case 4:
				cblogger.Infof("[%s] 키 페어 삭제 테스트", keyPairName)
				result, err := handler.DeleteKey(keyReq)
				if err != nil {
					cblogger.Infof(keyPairName, " 키 페어 삭제 실패 : ", err)
				} else {
					cblogger.Infof("[%s] 키 페어 삭제 결과 : [%s]", keyPairName, result)
				}
			}
		}
	}
}

// Test VMSpec
func handleVMSpec() {
	cblogger.Info("Start VMSpec Resource Test")

	ResourceHandler, err := testconf.GetResourceHandler("VMSpec")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.VMSpecHandler)
	//region := "asia-northeast1"

	zone := "asia-northeast1-b"
	machinename := ""

	cblogger.Info("zone : ", zone)

	for {
		fmt.Println("")
		fmt.Println("VMSpec Resource Test")
		fmt.Println("1. ListVMSpec()")
		fmt.Println("2. GetVMSpec()")
		fmt.Println("3. ListOrgVMSpec()")
		fmt.Println("4. GetOrgVMSpec()")
		fmt.Println("5. Exit")

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
				result, err := handler.ListVMSpec(zone)
				if err != nil {
					cblogger.Error("ListVMSpec 목록 조회 실패 : ", err)
				} else {
					cblogger.Info("ListVMSpec 목록 조회 결과")
					spew.Dump(result)
				}

				fmt.Println("Finish ListVMSpec()")

			case 2:
				fmt.Println("Start GetVMSpec() ...")
				result, err := handler.GetVMSpec(zone, machinename)
				if err != nil {
					cblogger.Error(machinename, " GetVMSpec 정보 조회 실패 : ", err)
				} else {
					cblogger.Infof("GetVMSpec[%s]  정보 조회 결과", machinename)
					spew.Dump(result)
				}
				fmt.Println("Finish GetVMSpec()")

			case 3:
				fmt.Println("Start ListOrgVMSpec() ...")
				result, err := handler.ListOrgVMSpec(zone)
				if err != nil {
					cblogger.Error("ListOrgVMSpec 목록 조회 실패 : ", err)
				} else {
					cblogger.Info("ListOrgVMSpec 목록 조회 결과")
					spew.Dump(result)
				}

				fmt.Println("Finish ListOrgVMSpec()")

			case 4:
				fmt.Println("Start GetOrgVMSpec() ...")
				result, err := handler.GetOrgVMSpec(zone, machinename)
				if err != nil {
					cblogger.Error(machinename, " GetOrgVMSpec 정보 조회 실패 : ", err)
				} else {
					cblogger.Infof("GetOrgVMSpec[%s]  정보 조회 결과", machinename)
					spew.Dump(result)
				}
				fmt.Println("Finish GetOrgVMSpec()")

			case 5:
				fmt.Println("Exit")
				return
			}
		}
	}
}

func main() {
	cblogger.Info("GCP Resource Test")
	//handlePublicIP()

	// handleKeyPair()

	handleImage() //AMI
	//handleVNic() //Lancard
	//handleSecurity()
	//handleVPC()
}
