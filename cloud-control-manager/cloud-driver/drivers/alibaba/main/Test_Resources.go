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
	cblog.SetLevel("info")
}

/*
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
*/

// Test VMSpec
func handleVMSpec() {
	cblogger.Debug("Start VMSpec Resource Test")

	ResourceHandler, err := testconf.GetResourceHandler("VMSpec")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.VMSpecHandler)

	//config := testconf.ReadConfigFile()
	//reqVMSpec := config.Ali.VMSpec
	//reqVMSpec := "ecs.g6.large"	// GPU가 없음
	reqVMSpec := "ecs.vgn5i-m8.4xlarge" // GPU 1개
	//reqVMSpec := "ecs.gn6i-c24g1.24xlarge" // GPU 4개

	//reqRegion := config.Ali.Region
	//reqRegion = "us-east-1"
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
				result, err := handler.ListVMSpec()
				if err != nil {
					cblogger.Error("VMSpec 목록 조회 실패 : ", err)
				} else {
					cblogger.Info("VMSpec 목록 조회 결과")
					spew.Dump(result)
				}

				fmt.Println("Finish ListVMSpec()")

			case 2:
				fmt.Println("Start GetVMSpec() ...")
				result, err := handler.GetVMSpec(reqVMSpec)
				if err != nil {
					cblogger.Error(reqVMSpec, " VMSpec 정보 조회 실패 : ", err)
				} else {
					cblogger.Infof("VMSpec[%s]  정보 조회 결과", reqVMSpec)
					spew.Dump(result)
				}
				fmt.Println("Finish GetVMSpec()")

			case 3:
				fmt.Println("Start ListOrgVMSpec() ...")
				result, err := handler.ListOrgVMSpec()
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
				result, err := handler.GetOrgVMSpec(reqVMSpec)
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

/*
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

*/

func handleSecurity() {
	cblogger.Debug("Start Security Resource Test")

	ResourceHandler, err := testconf.GetResourceHandler("Security")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.SecurityHandler)

	//config := readConfigFile()
	//VmID := config.Aws.VmID

	//securityName := "CB-SecurityTestCidr"
	securityName := "sg10"
	securityId := "sg-6we0jr4qremmfu2wyd8q"
	vpcId := "vpc-6weuepknbuvs90y6k1ss2"

	for {
		fmt.Println("Security Management")
		fmt.Println("0. Quit")
		fmt.Println("1. Security List")
		fmt.Println("2. Security Create")
		fmt.Println("3. Security Get")
		fmt.Println("4. Security Delete")
		fmt.Println("5. Rule Add")
		fmt.Println("6. Rule Remove")

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
				result, err := handler.ListSecurity()
				if err != nil {
					cblogger.Infof(" Security 목록 조회 실패 : ", err)
				} else {
					cblogger.Info("Security 목록 조회 결과")
					//cblogger.Info(result)
					spew.Dump(result)
					if result != nil {
						securityId = result[0].IId.SystemId // 조회 및 삭제를 위해 생성된 ID로 변경
					}
				}

			case 2:
				cblogger.Infof("[%s] Security 생성 테스트", securityName)
				securityReqInfo := irs.SecurityReqInfo{
					IId:    irs.IID{NameId: securityName},
					VpcIID: irs.IID{SystemId: vpcId},
					SecurityRules: &[]irs.SecurityRuleInfo{ //보안 정책 설정
						//CIDR 테스트
						/*{
							FromPort:   "20",
							ToPort:     "22",
							IPProtocol: "tcp",
							Direction:  "inbound",
							CIDR:       "0.0.0.0/0",
						},
						{
							FromPort:   "40",
							ToPort:     "40",
							IPProtocol: "tcp",
							Direction:  "outbound",
							CIDR:       "10.13.1.10/32",
						},*/
						/*
							{
								FromPort:   "20",
								ToPort:     "22",
								IPProtocol: "tcp",
								Direction:  "inbound",
							},

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
								FromPort:   "-1",
								ToPort:     "-1",
								IPProtocol: "icmp",
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
						*/
						/*
							{
								//FromPort:   "8443",
								//ToPort:     "9999",
								IPProtocol: "-1", // 모두 허용 (포트 정보 없음)
								Direction:  "inbound",
							},*/
						{
							FromPort:   "-1",
							ToPort:     "-1",
							IPProtocol: "all",
							Direction:  "outbound",
							CIDR:       "0.0.0.0/0",
						},
					},
				}

				result, err := handler.CreateSecurity(securityReqInfo)
				if err != nil {
					cblogger.Infof(securityName, " Security 생성 실패 : ", err)
				} else {
					cblogger.Infof("[%s] Security 생성 결과 : [%v]", securityName, result)
					securityId = result.IId.SystemId
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] Security 조회 테스트", securityId)
				result, err := handler.GetSecurity(irs.IID{SystemId: securityId})
				if err != nil {
					cblogger.Infof(securityId, " Security 조회 실패 : ", err)
				} else {
					cblogger.Infof("[%s] Security 조회 결과 : [%v]", securityId, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] Security 삭제 테스트", securityId)
				result, err := handler.DeleteSecurity(irs.IID{SystemId: securityId})
				if err != nil {
					cblogger.Infof(securityId, " Security 삭제 실패 : ", err)
				} else {
					cblogger.Infof("[%s] Security 삭제 결과 : [%s]", securityId, result)
				}
			case 5:
				cblogger.Infof("[%s] Rule 추가 테스트", securityId)
				securityRules := &[]irs.SecurityRuleInfo{

					/*{
						//20-22 Prot로 등록
						FromPort:   "20",
						ToPort:     "21",
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						//20-22 Prot로 등록
						FromPort:   "20",
						ToPort:     "21",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						// 8080 Port로 등록
						FromPort:   "8080",
						ToPort:     "8080", //FromPort나 ToPort중 하나에 -1이 입력될 경우 -1이 입력된 경우 -1을 공백으로 처리
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{ // 1323 Prot로 등록
						FromPort:   "1323", //FromPort나 ToPort중 하나에 -1이 입력될 경우 -1이 입력된 경우 -1을 공백으로 처리
						ToPort:     "1323",
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						// All Port로 등록
						FromPort:   "",
						ToPort:     "",
						IPProtocol: "icmp", //icmp는 포트 정보가 없음
						Direction:  "inbound",
					},*/
					/*{
						//20-22 Prot로 등록
						FromPort:   "20",
						ToPort:     "22",
						IPProtocol: "tcp",
						Direction:  "inbound",
					},*/
					/*{
						// 80 Port로 등록
						FromPort:   "80",
						ToPort:     "80",
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{ // 모든 프로토콜 모든 포트로 등록
						//FromPort:   "",
						//ToPort:     "",
						IPProtocol: "all", // 모두 허용 (포트 정보 없음)
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						FromPort:   "443",
						ToPort:     "443",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						FromPort:   "8443",
						ToPort:     "9999",
						IPProtocol: "tcp",
						Direction:  "outbound",
					},*/
					/*{
						//20-22 Prot로 등록
						FromPort:   "22",
						ToPort:     "22",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						//20-22 Prot로 등록
						FromPort:   "1000",
						ToPort:     "1000",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						//20-22 Prot로 등록
						FromPort:   "1",
						ToPort:     "65535",
						IPProtocol: "udp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						//20-22 Prot로 등록
						FromPort:   "-1",
						ToPort:     "-1",
						IPProtocol: "icmp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						//20-22 Prot로 등록
						FromPort:   "-1",
						ToPort:     "-1",
						IPProtocol: "all",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						//20-22 Prot로 등록
						FromPort:   "22",
						ToPort:     "22",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						//20-22 Prot로 등록
						FromPort:   "1000",
						ToPort:     "1000",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						//20-22 Prot로 등록
						FromPort:   "1",
						ToPort:     "65535",
						IPProtocol: "udp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						//20-22 Prot로 등록
						FromPort:   "22",
						ToPort:     "22",
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						//20-22 Prot로 등록
						FromPort:   "1000",
						ToPort:     "1000",
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "4.5.6.7/32",
					},*/
					/*{
						//20-22 Prot로 등록
						FromPort:   "1",
						ToPort:     "65535",
						IPProtocol: "udp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						//20-22 Prot로 등록
						FromPort:   "-1",
						ToPort:     "-1",
						IPProtocol: "icmp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},*/
					{
						//20-22 Prot로 등록
						FromPort:   "22",
						ToPort:     "22",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						//20-22 Prot로 등록
						FromPort:   "1000",
						ToPort:     "1000",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						//20-22 Prot로 등록
						FromPort:   "1",
						ToPort:     "65535",
						IPProtocol: "udp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						//20-22 Prot로 등록
						FromPort:   "-1",
						ToPort:     "-1",
						IPProtocol: "icmp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},
				}

				result, err := handler.AddRules(irs.IID{SystemId: securityId}, securityRules)
				if err != nil {
					cblogger.Infof(securityId, " Rule 추가 실패 : ", err)
				} else {
					cblogger.Infof("[%s] Rule 추가 결과 : [%v]", securityId, result)
					spew.Dump(result)
				}
			case 6:
				cblogger.Infof("[%s] Rule 삭제 테스트", securityId)
				securityRules := &[]irs.SecurityRuleInfo{
					/*{
							//20-22 Prot로 등록
							FromPort:   "20",
							ToPort:     "21",
							IPProtocol: "tcp",
							Direction:  "inbound",
							CIDR:       "0.0.0.0/0",
					},*/
					/*{
							//20-22 Prot로 등록
							FromPort:   "20",
							ToPort:     "21",
							IPProtocol: "tcp",
							Direction:  "outbound",
							CIDR:       "0.0.0.0/0",
					},*/
					/*{
						FromPort:   "40",
						ToPort:     "40",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "10.13.1.10/32",
					},*/
					/*{
						//20-22 Prot로 등록
						FromPort:   "20",
						ToPort:     "22",
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						//20-22 Prot로 등록
						FromPort:   "20",
						ToPort:     "21",
						IPProtocol: "udp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						// 8080 Port로 등록
						FromPort:   "8080",
						ToPort:     "8080", //FromPort나 ToPort중 하나에 -1이 입력될 경우 -1이 입력된 경우 -1을 공백으로 처리
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{ // 1323 Prot로 등록
						FromPort:   "1323", //FromPort나 ToPort중 하나에 -1이 입력될 경우 -1이 입력된 경우 -1을 공백으로 처리
						ToPort:     "1323",
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						// All Port로 등록
						FromPort:   "",
						ToPort:     "",
						IPProtocol: "icmp", //icmp는 포트 정보가 없음
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{ // 모든 프로토콜 모든 포트로 등록
						//FromPort:   "",
						//ToPort:     "",
						IPProtocol: "all", // 모두 허용 (포트 정보 없음)
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},*/
					{
						//20-22 Prot로 등록
						FromPort:   "22",
						ToPort:     "22",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						//20-22 Prot로 등록
						FromPort:   "1000",
						ToPort:     "1000",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						//20-22 Prot로 등록
						FromPort:   "1",
						ToPort:     "65535",
						IPProtocol: "udp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						//20-22 Prot로 등록
						FromPort:   "-1",
						ToPort:     "-1",
						IPProtocol: "icmp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},
					/*{
						//20-22 Prot로 등록
						FromPort:   "-1",
						ToPort:     "-1",
						IPProtocol: "all",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},*/
				}

				result, err := handler.RemoveRules(irs.IID{SystemId: securityId}, securityRules)
				if err != nil {
					cblogger.Infof(securityId, " Rule 삭제 실패 : ", err)
				} else {
					cblogger.Infof("[%s] Rule 삭제 결과 : [%v]", securityId, result)
				}
			}
		}
	}
}

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
					if result != nil {
						keyPairName = result[0].IId.SystemId // 조회 및 삭제를 위해 생성된 ID로 변경
					}
					cblogger.Info("키 페어 수 : ", len(result))
				}

			case 2:
				cblogger.Infof("[%s] 키 페어 생성 테스트", keyPairName)
				keyPairReqInfo := irs.KeyPairReqInfo{
					IId: irs.IID{NameId: keyPairName},
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
				result, err := handler.GetKey(irs.IID{SystemId: keyPairName})
				if err != nil {
					cblogger.Infof(keyPairName, " 키 페어 조회 실패 : ", err)
				} else {
					cblogger.Infof("[%s] 키 페어 조회 결과 : [%s]", keyPairName, result)
					spew.Dump(result)
				}
			case 4:
				cblogger.Infof("[%s] 키 페어 삭제 테스트", keyPairName)
				result, err := handler.DeleteKey(irs.IID{SystemId: keyPairName})
				if err != nil {
					cblogger.Infof(keyPairName, " 키 페어 삭제 실패 : ", err)
				} else {
					cblogger.Infof("[%s] 키 페어 삭제 결과 : [%s]", keyPairName, result)
				}
			}
		}
	}
}

/*
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
*/

func handleVPC() {
	cblogger.Debug("Start VPC Resource Test")
	ResourceHandler, err := testconf.GetResourceHandler("VPC")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.VPCHandler)

	subnetReqInfo := irs.SubnetInfo{
		IId:       irs.IID{NameId: "AddTest-Subnet"},
		IPv4_CIDR: "10.0.3.0/24",
	}

	subnetReqVpcInfo := irs.IID{SystemId: "vpc-6wex2mrx1fovfecsl44mx"}
	reqSubnetId := irs.IID{SystemId: "vsw-6we4h4n4wp9xdtakrno15"}
	cblogger.Debug(subnetReqInfo)
	cblogger.Debug(subnetReqVpcInfo)
	cblogger.Debug(reqSubnetId)

	vpcReqInfo := irs.VPCReqInfo{
		IId:       irs.IID{NameId: "New-CB-VPC"},
		IPv4_CIDR: "10.0.0.0/16",
		SubnetInfoList: []irs.SubnetInfo{
			{
				IId:       irs.IID{NameId: "New-CB-Subnet"},
				IPv4_CIDR: "10.0.1.0/24",
			},

			{
				IId:       irs.IID{NameId: "New-CB-Subnet2"},
				IPv4_CIDR: "10.0.2.0/24",
			},
		},
		//Id:   "subnet-044a2b57145e5afc5",
		//Name: "CB-VNet-Subnet", // 웹 도구 등 외부에서 전달 받지 않고 드라이버 내부적으로 자동 구현때문에 사용하지 않음.
		//CidrBlock: "10.0.0.0/16",
		//CidrBlock: "192.168.0.0/16",
	}

	reqVpcId := irs.IID{SystemId: "vpc-6we11xwqjc9tyma5i68z0"}

	for {
		fmt.Println("Handler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. VPC List")
		fmt.Println("2. VPC Create")
		fmt.Println("3. VPC Get")
		fmt.Println("4. VPC Delete")
		fmt.Println("5. Add Subnet")
		fmt.Println("6. Delete Subnet")

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
					cblogger.Infof(" VPC 목록 조회 실패 : ", err)
				} else {
					cblogger.Info("VPC 목록 조회 결과")
					//cblogger.Info(result)
					spew.Dump(result)

					// 내부적으로 1개만 존재함.
					//조회및 삭제 테스트를 위해 리스트의 첫번째 서브넷 ID를 요청ID로 자동 갱신함.
					if result != nil {
						reqVpcId = result[0].IId    // 조회 및 삭제를 위해 생성된 ID로 변경
						subnetReqVpcInfo = reqVpcId //Subnet 추가/삭제 테스트용
					}
				}

			case 2:
				cblogger.Infof("[%s] VPC 생성 테스트", vpcReqInfo.IId.NameId)
				//vpcReqInfo := irs.VPCReqInfo{}
				result, err := handler.CreateVPC(vpcReqInfo)
				if err != nil {
					cblogger.Infof(reqVpcId.NameId, " VPC 생성 실패 : ", err)
				} else {
					cblogger.Infof("VPC 생성 결과 : ", result)
					reqVpcId = result.IId // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] VPC 조회 테스트", reqVpcId)
				result, err := handler.GetVPC(reqVpcId)
				if err != nil {
					cblogger.Infof("[%s] VPC 조회 실패 : ", reqVpcId, err)
				} else {
					cblogger.Infof("[%s] VPC 조회 결과 : [%s]", reqVpcId, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] VPC 삭제 테스트", reqVpcId)
				result, err := handler.DeleteVPC(reqVpcId)
				if err != nil {
					cblogger.Infof("[%s] VPC 삭제 실패 : ", reqVpcId, err)
				} else {
					cblogger.Infof("[%s] VPC 삭제 결과 : [%s]", reqVpcId, result)
				}

			case 5:
				cblogger.Infof("[%s] Subnet 추가 테스트", vpcReqInfo.IId.NameId)
				result, err := handler.AddSubnet(subnetReqVpcInfo, subnetReqInfo)
				if err != nil {
					cblogger.Infof(reqSubnetId.NameId, " Subnet 추가 실패 : ", err)
				} else {
					cblogger.Infof("Subnet 추가 결과 : ", result)
					reqSubnetId = result.IId // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 6:
				cblogger.Infof("[%s] Subnet 삭제 테스트", reqSubnetId.SystemId)
				result, err := handler.RemoveSubnet(subnetReqVpcInfo, reqSubnetId)
				if err != nil {
					cblogger.Infof("[%s] Subnet 삭제 실패 : ", reqSubnetId.SystemId, err)
				} else {
					cblogger.Infof("[%s] Subnet 삭제 결과 : [%s]", reqSubnetId.SystemId, result)
				}
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
		IId: irs.IID{NameId: "Test OS Image", SystemId: "ami-047f7b46bd6dd5d84"},
		//Id:   "ami-047f7b46bd6dd5d84",
		//Name: "Test OS Image",
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
					cblogger.Debug(result)
					cblogger.Info("출력 결과 수 : ", len(result))

					if cblogger.Level.String() == "debug" {
						spew.Dump(result)
					}

					//조회및 삭제 테스트를 위해 리스트의 첫번째 정보의 ID를 요청ID로 자동 갱신함.
					if result != nil {
						imageReqInfo.IId = result[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
					}
				}

			case 2:
				cblogger.Infof("[%s] Image 생성 테스트", imageReqInfo.IId.NameId)
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
				cblogger.Infof("[%s] Image 삭제 테스트", imageReqInfo.IId.NameId)
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

// Test VM Lifecycle Management (Create/Suspend/Resume/Reboot/Terminate)
func handleVM() {
	cblogger.Debug("Start VMHandler Resource Test")

	ResourceHandler, err := testconf.GetResourceHandler("VM")
	if err != nil {
		panic(err)
	}
	vmHandler := ResourceHandler.(irs.VMHandler)

	//config := readConfigFile()
	//VmID := irs.IID{NameId: config.Aws.BaseName, SystemId: config.Aws.VmID}
	VmID := irs.IID{SystemId: "i-6weayupx7qvidhmyl48d"}

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
				vmReqInfo := irs.VMReqInfo{
					IId: irs.IID{NameId: "mcloud-barista-vm-test"},
					//ImageIID: irs.IID{SystemId: "aliyun_3_x64_20G_alibase_20210425.vhd"},
					//ImageIID: irs.IID{SystemId: "aliyun_2_1903_x64_20G_alibase_20200324.vhd"},
					//ImageIID:  irs.IID{SystemId: "ubuntu_18_04_x64_20G_alibase_20210318.vhd"},
					ImageIID: irs.IID{SystemId: "ubuntu_18_04_x64_20G_alibase_20210420.vhd"},
					//VpcIID:    irs.IID{SystemId: "vpc-0jl4l19l51gn2exrohgci"},
					//SubnetIID: irs.IID{SystemId: "vsw-0jlj155cbwhjumtipnm6d"},
					SubnetIID: irs.IID{SystemId: "vsw-6we8tac8w7dzqbxbyhj9o"}, //Tokyo Zone B
					//SecurityGroupIIDs: []irs.IID{{SystemId: "sg-6we0rxnoai067qbkdkgw"}, {SystemId: "sg-6weeb9xaodr65g7bq10c"}},
					SecurityGroupIIDs: []irs.IID{{SystemId: "sg-6we7156yw8c8xbzi9f7v"}},
					//VMSpecName:        "ecs.t5-lc2m1.nano",
					//VMSpecName: "ecs.g6.large", //cn-wulanchabu 리전
					VMSpecName: "ecs.t5-lc2m1.nano", //도쿄리전
					KeyPairIID: irs.IID{SystemId: "cb-japan"},
					//VMUserId:          "root", //root만 가능
					//VMUserPasswd: "Cbuser!@#", //대문자 소문자 모두 사용되어야 함. 그리고 숫자나 특수 기호 중 하나가 포함되어야 함.

					RootDiskType: "cloud_efficiency", //cloud / cloud_efficiency / cloud_ssd / cloud_essd
					RootDiskSize: "default",
					//RootDiskType: "cloud_ssd", //cloud / cloud_efficiency / cloud_ssd / cloud_essd
					//RootDiskSize: "22",
				}

				vmInfo, err := vmHandler.StartVM(vmReqInfo)
				if err != nil {
					//panic(err)
					cblogger.Error("VM 생성 실패 - 실패 이유")
					cblogger.Error(err)
				} else {
					cblogger.Info("VM 생성 완료!!", vmInfo)
					spew.Dump(vmInfo)
					VmID = vmInfo.IId
				}
				//cblogger.Info(vm)

				cblogger.Info("Finish Create VM")

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
					if len(vmList) > 0 {
						VmID = vmList[0].IId
					}
				}

			}
		}
	}
}

func handleNLB() {
	cblogger.Debug("Start NLB Resource Test")
	ResourceHandler, err := testconf.GetResourceHandler("NLB")
	if err != nil {
		//panic(err)
		cblogger.Error(err)
	}
	handler := ResourceHandler.(irs.NLBHandler)
	cblogger.Info(handler)
	nlbReqInfo := irs.NLBInfo{
		// TCP
		IId:           irs.IID{NameId: "New-CB-TCPNLB4"},
		VpcIID:        irs.IID{SystemId: "vpc-t4naidq3kbofx4y09ignm"},
		Type:          "PUBLIC",
		Listener:      irs.ListenerInfo{Protocol: "TCP", Port: "80"},
		HealthChecker: irs.HealthCheckerInfo{Protocol: "HTTP", Port: "80", Interval: 5, Timeout: 2, Threshold: 3},
		VMGroup: irs.VMGroupInfo{
			Protocol: "TCP",
			Port:     "80",

			VMs: &[]irs.IID{{SystemId: "i-t4ndzu5q27yow3xhk9ns"}, {SystemId: "i-t4ndzu5q27yow3xhk9nt"}},
		},

		// UDP
		//IId:    irs.IID{NameId: "New-CB-UDPNLB2"},
		//VpcIID:   irs.IID{SystemId: "vpc-t4naidq3kbofx4y09ignm"},
		//Type:     "PUBLIC",
		//Listener: irs.ListenerInfo{Protocol: "UDP", Port: "23"},
		//HealthChecker: irs.HealthCheckerInfo{Protocol: "UDP", Port: "23", Interval: 5, Timeout: 2, Threshold: 3},
		//VMGroup: irs.VMGroupInfo{
		//	//Protocol: "UDP",
		//	//Port:     "23",
		//
		//	VMs: &[]irs.IID{{SystemId: "i-t4ndzu5q27yow3xhk9ns"}, {SystemId: "i-t4ndzu5q27yow3xhk9nt"}},
		//},
	}

	reqNLBId := irs.IID{SystemId: "lb-ecyd4pb5"}

	for {
		fmt.Println("Handler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. NLB List")
		fmt.Println("2. NLB Create")
		fmt.Println("3. NLB Get")
		fmt.Println("4. NLB Delete")
		fmt.Println("5. VM Add")
		fmt.Println("6. VM Delete")
		fmt.Println("7. VM Health Get")
		fmt.Println("8. Listener Change")
		fmt.Println("9. VM Group Change")
		fmt.Println("10. Health Checker Change")

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
				result, err := handler.ListNLB()
				if err != nil {
					cblogger.Infof(" NLB 목록 조회 실패 : ", err)
				} else {
					cblogger.Info("NLB 목록 조회 결과")
					//cblogger.Info(result)
					spew.Dump(result)
				}

			case 2:
				cblogger.Infof("[%s] NLB 생성 테스트", nlbReqInfo.IId.NameId)
				//vpcReqInfo := irs.VPCReqInfo{}
				result, err := handler.CreateNLB(nlbReqInfo)
				if err != nil {
					cblogger.Infof(nlbReqInfo.IId.NameId, " NLB 생성 실패 : ", err)
				} else {
					cblogger.Infof("NLB 생성 결과 : ", result)
					//reqNLBId = result.IId // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 3:
				reqNLBId = irs.IID{SystemId: "lb-gs5xf7uv5iiwzpwwpx39e"}
				cblogger.Infof("[%s] NLB 조회 테스트", reqNLBId)
				result, err := handler.GetNLB(reqNLBId)
				if err != nil {
					cblogger.Infof("[%s] NLB 조회 실패 : ", reqNLBId, err)
				} else {
					cblogger.Infof("[%s] NLB 조회 결과 : [%s]", reqNLBId, result)
					spew.Dump(result)
				}

			case 4:
				reqNLBId.SystemId = "lb-gs5x5ab796t61x95m7upp"
				cblogger.Infof("[%s] NLB 삭제 테스트", reqNLBId)
				result, err := handler.DeleteNLB(reqNLBId)
				if err != nil {
					cblogger.Infof("[%s] NLB 삭제 실패 : ", reqNLBId, err)
				} else {
					cblogger.Infof("[%s] NLB 삭제 결과 : [%s]", reqNLBId, result)
				}

			case 5:
				cblogger.Infof("[%s] VM 추가 테스트", reqNLBId)
				reqNLBId.SystemId = "lb-gs5xf7uv5iiwzpwwpx39e"
				vmIID := irs.IID{SystemId: "i-t4n771gareh1wzo7ghpe"}
				result, err := handler.AddVMs(reqNLBId, &[]irs.IID{vmIID})
				if err != nil {
					cblogger.Infof("VM 추가 실패 : ", err)
				} else {
					cblogger.Infof("VM 추가 결과 : ", result)
					//reqSubnetId = result.SubnetInfoList[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 6:
				cblogger.Infof("[%s] VM 삭제 테스트", reqNLBId.SystemId)

				reqNLBId.SystemId = "lb-gs5xf7uv5iiwzpwwpx39e"
				vmIID := irs.IID{SystemId: "i-t4n771gareh1wzo7ghpe"}
				result, err := handler.RemoveVMs(reqNLBId, &[]irs.IID{vmIID})
				if err != nil {
					cblogger.Infof("VM 삭제 실패 : ", err)
				} else {
					cblogger.Infof("VM 삭제 결과 : [%s]", result)
				}
			case 7:
				cblogger.Infof("[%s] NLB VM Health 조회 테스트", reqNLBId)
				cblogger.Infof("[%s] VM 추가 테스트", reqNLBId)
				reqNLBId.SystemId = "lb-gs5xf7uv5iiwzpwwpx39e"
				result, err := handler.GetVMGroupHealthInfo(reqNLBId)
				if err != nil {
					cblogger.Infof("[%s] NLB VM Health 조회 실패 : ", reqNLBId.SystemId, err)
				} else {
					cblogger.Infof("[%s] NLB VM Health 조회 결과 : [%s]", reqNLBId.SystemId, result)
					spew.Dump(result)
				}
			case 8:
				cblogger.Infof("[%s] NLB Listener 변경 테스트", reqNLBId)
				reqNLBId.SystemId = "lb-gs5xf7uv5iiwzpwwpx39e"
				changeListener := irs.ListenerInfo{}
				changeListener.Protocol = "tcp"
				changeListener.Port = "8080" // 포트만 변경
				result, err := handler.ChangeListener(reqNLBId, changeListener)
				if err != nil {
					cblogger.Infof("[%s] NLB Listener 변경 실패 : ", reqNLBId.SystemId, err)
				} else {
					cblogger.Infof("[%s] NLB Listener 변경 결과 : [%s]", reqNLBId.SystemId, result)
					spew.Dump(result)
				}
			case 9:
				cblogger.Infof("[%s] NLB VM Group 변경 테스트", reqNLBId)
				result, err := handler.ChangeVMGroupInfo(reqNLBId, irs.VMGroupInfo{
					Protocol: "TCP",
					Port:     "8080",
				})
				if err != nil {
					cblogger.Infof("[%s] NLB VM Group 변경 실패 : ", reqNLBId.SystemId, err)
				} else {
					cblogger.Infof("[%s] NLB VM Group 변경 결과 : [%s]", reqNLBId.SystemId, result)
					spew.Dump(result)
				}
			case 10:
				reqNLBId.SystemId = "lb-gs5xf7uv5iiwzpwwpx39e"
				reqHealthCheckInfo := irs.HealthCheckerInfo{
					Protocol:  "tcp",
					Port:      "85",
					Interval:  49,
					Timeout:   30,
					Threshold: 9,
				}
				cblogger.Infof("[%s] NLB Health Checker 변경 테스트", reqNLBId)
				result, err := handler.ChangeHealthCheckerInfo(reqNLBId, reqHealthCheckInfo)
				if err != nil {
					cblogger.Infof("[%s] NLB Health Checker 변경 실패 : ", reqNLBId.SystemId, err)
				} else {
					cblogger.Infof("[%s] NLB Health Checker 변경 결과 : [%s]", reqNLBId.SystemId, result)
					spew.Dump(result)
				}

			}
		}
	}
}

func main() {
	cblogger.Info("Alibaba Cloud Resource Test")
	cblogger.Debug("Debug mode")

	//handleVPC() //VPC
	//handleVMSpec()
	//handleImage() //AMI
	//handleSecurity()
	//handleKeyPair()
	//handleVM()
	handleNLB()
	//handlePublicIP() // PublicIP 생성 후 conf

	//handleVNic() //Lancard

	/*
		//StartTime := "2020-05-07T01:35:00Z"
		StartTime := "2020-05-07T01:35Z"
		timeLen := len(StartTime)
		cblogger.Infof("======> 생성시간 길이 [%s]", timeLen)
		if timeLen > 7 {
			cblogger.Infof("======> 생성시간 마지막 문자열 [%s]", StartTime[timeLen-1:])
			if StartTime[timeLen-1:] == "Z" {
				cblogger.Infof("======> 문자열 변환 : [%s]", StartTime[:timeLen-1])
				NewStartTime := StartTime[:timeLen-1] + ":00Z"
				cblogger.Infof("======> 최종 문자열 변환 : [%s]", NewStartTime)
			}
		}

		//:41+00:00
		cblogger.Infof("Convert StartTime string [%s] to time.time", StartTime)

		//layout := "2020-05-07T01:36Z"
		t, err := time.Parse(time.RFC3339, StartTime)
		if err != nil {
			cblogger.Error(err)
		} else {
			cblogger.Infof("======> [%v]", t)
		}
	*/
}
