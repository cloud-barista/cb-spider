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
	//cblog.SetLevel("info")
	cblog.SetLevel("debug")
}

func handleSecurity() {
	cblogger.Debug("Start Security Resource Test")

	ResourceHandler, err := getResourceHandler("Security")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.SecurityHandler)

	//config := readConfigFile()
	//VmID := config.Aws.VmID

	securityName := "CB-SecurityAddTest1"
	securityId := "sg-0d6a2bb960481ce68"
	vpcId := "vpc-c0479cab"

	for {
		fmt.Println("Security Management")
		fmt.Println("0. Quit")
		fmt.Println("1. Security List")
		fmt.Println("2. Security Create")
		fmt.Println("3. Security Get")
		fmt.Println("4. Security Delete")
		fmt.Println("5. Security Add Rules")
		fmt.Println("6. Security Delete Rules")

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
						{
							FromPort:   "30",
							ToPort:     "30",
							IPProtocol: "tcp",
							Direction:  "inbound",
							CIDR:       "10.13.1.10/32",
						},
						{
							FromPort:   "40",
							ToPort:     "40",
							IPProtocol: "tcp",
							Direction:  "outbound",
							CIDR:       "10.13.1.10/32",
						},
						// {
						// 	FromPort:   "30",
						// 	ToPort:     "30",
						// 	IPProtocol: "tcp",
						// 	Direction:  "outbound",
						// 	CIDR:       "1.2.3.4/0",
						// },
						// {
						// 	FromPort:   "20",
						// 	ToPort:     "22",
						// 	IPProtocol: "tcp",
						// 	Direction:  "inbound",
						// 	//CIDR:       "1.2.3.4/0",
						// },
						/*
							{
								FromPort:   "80",
								ToPort:     "80",
								IPProtocol: "tcp",
								Direction:  "inbound",
								CIDR:       "1.2.3.4/0",
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
							},
						*/
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
				cblogger.Infof("[%s] Security 그룹 룰 추가 테스트", securityId)
				result, err := handler.AddRules(irs.IID{SystemId: securityId}, &[]irs.SecurityRuleInfo{
					{
						FromPort:   "80",
						ToPort:     "80",
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "10.13.1.10/32",
					},
					{
						FromPort:   "8080",
						ToPort:     "8080",
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "10.13.1.10/32",
					},
					{
						FromPort:   "81",
						ToPort:     "81",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "10.13.1.10/32",
					},
					{
						FromPort:   "82",
						ToPort:     "82",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "10.13.1.10/32",
					},
				})
				if err != nil {
					cblogger.Infof(securityId, " Security 그룹 룰 추가 실패 : ", err)
				} else {
					cblogger.Infof("[%s] Security 그룹 룰 추가 결과 : [%s]", securityId, result)
				}

			case 6:
				cblogger.Infof("[%s] Security 그룹 룰 제거 테스트", securityId)
				result, err := handler.RemoveRules(irs.IID{SystemId: securityId}, &[]irs.SecurityRuleInfo{
					{
						FromPort:   "80",
						ToPort:     "80",
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "10.13.1.10/32",
					},
					{
						FromPort:   "8080",
						ToPort:     "8080",
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "10.13.1.10/32",
					},
					{
						FromPort:   "81",
						ToPort:     "81",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "10.13.1.10/32",
					},
					{
						FromPort:   "82",
						ToPort:     "82",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "10.13.1.10/32",
					},
				})
				if err != nil {
					cblogger.Infof(securityId, " Security 그룹 룰 제거 실패 : ", err)
				} else {
					cblogger.Infof("[%s] Security 그룹 룰 제거 결과 : [%s]", securityId, result)
				}
			}
		}
	}
}

// Test SecurityHandler
func handleSecurityOld() {
	cblogger.Debug("Start handler")

	ResourceHandler, err := getResourceHandler("Security")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.SecurityHandler)

	config := readConfigFile()
	securityId := config.Aws.SecurityGroupID
	cblogger.Infof(securityId)
	securityId = "sg-0101df0e8d4f27fec"
	//securityId = "cb-sgtest-mcloud-barista"

	//result, err := handler.GetSecurity(irs.IID{SystemId: securityId})
	//result, err := handler.GetSecurity("sg-0fd2d90b269ebc082") // sgtest-mcloub-barista
	//result, err := handler.DeleteSecurity(irs.IID{SystemId: securityId})
	//result, err := handler.DeleteSecurity(irs.IID{SystemId: "sg-0101df0e8d4f27fec"})
	result, err := handler.ListSecurity()

	securityReqInfo := irs.SecurityReqInfo{
		IId:    irs.IID{NameId: "cb-sgtest2-mcloud-barista"},
		VpcIID: irs.IID{NameId: "CB-VNet", SystemId: "vpc-0c23cb9c0e68c735a"},
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
	//result, err := handler.CreateSecurity(securityReqInfo)

	if err != nil {
		cblogger.Infof("보안 그룹 조회 실패 : ", err)
	} else {
		cblogger.Info("보안 그룹 조회 결과")
		//cblogger.Info(result)
		spew.Dump(result)
	}
}

/*
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
*/

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
					IId: irs.IID{NameId: keyPairName},
					//Name: keyPairName,
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
				result, err := KeyPairHandler.GetKey(irs.IID{SystemId: keyPairName})
				if err != nil {
					cblogger.Infof(keyPairName, " 키 페어 조회 실패 : ", err)
				} else {
					cblogger.Infof("[%s] 키 페어 조회 결과 : [%s]", keyPairName, result)
				}
			case 4:
				cblogger.Infof("[%s] 키 페어 삭제 테스트", keyPairName)
				result, err := KeyPairHandler.DeleteKey(irs.IID{SystemId: keyPairName})
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
/*
func handleVNetwork() {
	cblogger.Debug("Start VPC Resource Test")

	VPCHandler, err := setVPCHandler()
	if err != nil {
		panic(err)
	}

	vNetworkReqInfo := irs.VNetworkReqInfo{
		//Id:   "subnet-044a2b57145e5afc5",
		//Name: "CB-VNet-Subnet", // 웹 도구 등 외부에서 전달 받지 않고 드라이버 내부적으로 자동 구현때문에 사용하지 않음.
		IId: irs.IID{NameId: "CB-VNet-Subnet"},
		//CidrBlock: "10.0.0.0/16",
		//CidrBlock: "192.168.0.0/16",
	}
	//reqSubnetId := "subnet-0b9ea37601d46d8fa"
	reqSubnetId := irs.IID{NameId: "subnet-0b9ea37601d46d8fa"}
	//reqSubnetId = ""

	for {
		fmt.Println("VPCHandler Management")
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
				result, err := VPCHandler.ListVNetwork()
				if err != nil {
					cblogger.Infof(" VNetwork 목록 조회 실패 : ", err)
				} else {
					cblogger.Info("VNetwork 목록 조회 결과")
					//cblogger.Info(result)
					spew.Dump(result)

					// 내부적으로 1개만 존재함.
					//조회및 삭제 테스트를 위해 리스트의 첫번째 서브넷 ID를 요청ID로 자동 갱신함.
					if result != nil {
						reqSubnetId = result[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
					}
				}

			case 2:
				cblogger.Infof("[%s] VNetwork 생성 테스트", vNetworkReqInfo.IId.NameId)
				//vNetworkReqInfo := irs.VNetworkReqInfo{}
				result, err := VPCHandler.CreateVNetwork(vNetworkReqInfo)
				if err != nil {
					cblogger.Infof(reqSubnetId.NameId, " VNetwork 생성 실패 : ", err)
				} else {
					cblogger.Infof("VNetwork 생성 결과 : ", result)
					reqSubnetId = result.IId // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] VNetwork 조회 테스트", reqSubnetId)
				result, err := VPCHandler.GetVNetwork(reqSubnetId)
				if err != nil {
					cblogger.Infof("[%s] VNetwork 조회 실패 : ", reqSubnetId, err)
				} else {
					cblogger.Infof("[%s] VNetwork 조회 결과 : [%s]", reqSubnetId, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] VNetwork 삭제 테스트", reqSubnetId)
				result, err := VPCHandler.DeleteVNetwork(reqSubnetId)
				if err != nil {
					cblogger.Infof("[%s] VNetwork 삭제 실패 : ", reqSubnetId, err)
				} else {
					cblogger.Infof("[%s] VNetwork 삭제 결과 : [%s]", reqSubnetId, result)
				}
			}
		}
	}
}
*/

func handleVPC() {
	cblogger.Debug("Start VPC Resource Test")

	VPCHandler, err := setVPCHandler()
	if err != nil {
		panic(err)
	}

	subnetReqInfo := irs.SubnetInfo{
		IId:       irs.IID{NameId: "AddTest-Subnet"},
		IPv4_CIDR: "10.0.2.0/24",
	}

	subnetReqVpcInfo := irs.IID{SystemId: "vpc-00e513fd64a7d9972"}

	cblogger.Debug(subnetReqInfo)
	cblogger.Debug(subnetReqVpcInfo)

	vpcReqInfo := irs.VPCReqInfo{
		IId:       irs.IID{NameId: "New-CB-VPC"},
		IPv4_CIDR: "10.0.0.0/16",
		SubnetInfoList: []irs.SubnetInfo{
			{
				IId:       irs.IID{NameId: "New-CB-Subnet"},
				IPv4_CIDR: "10.0.1.0/24",
			},
			/*
				{
					IId:       irs.IID{NameId: "New-CB-Subnet2"},
					IPv4_CIDR: "10.0.2.0/24",
				},
			*/
		},
		//Id:   "subnet-044a2b57145e5afc5",
		//Name: "CB-VNet-Subnet", // 웹 도구 등 외부에서 전달 받지 않고 드라이버 내부적으로 자동 구현때문에 사용하지 않음.
		//CidrBlock: "10.0.0.0/16",
		//CidrBlock: "192.168.0.0/16",
	}

	reqSubnetId := irs.IID{SystemId: "vpc-04f6de5c2af880978"}
	reqSubnetId = irs.IID{SystemId: "subnet-0ebd316ff47f07628"}

	for {
		fmt.Println("VPCHandler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. VNetwork List")
		fmt.Println("2. VNetwork Create")
		fmt.Println("3. VNetwork Get")
		fmt.Println("4. VNetwork Delete")
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
				result, err := VPCHandler.ListVPC()
				if err != nil {
					cblogger.Infof(" VNetwork 목록 조회 실패 : ", err)
				} else {
					cblogger.Info("VNetwork 목록 조회 결과")
					//cblogger.Info(result)
					spew.Dump(result)

					// 내부적으로 1개만 존재함.
					//조회및 삭제 테스트를 위해 리스트의 첫번째 서브넷 ID를 요청ID로 자동 갱신함.
					if result != nil {
						reqSubnetId = result[0].IId    // 조회 및 삭제를 위해 생성된 ID로 변경
						subnetReqVpcInfo = reqSubnetId //Subnet 추가/삭제 테스트용
					}
				}

			case 2:
				cblogger.Infof("[%s] VNetwork 생성 테스트", vpcReqInfo.IId.NameId)
				//vpcReqInfo := irs.VPCReqInfo{}
				result, err := VPCHandler.CreateVPC(vpcReqInfo)
				if err != nil {
					cblogger.Infof(reqSubnetId.NameId, " VNetwork 생성 실패 : ", err)
				} else {
					cblogger.Infof("VNetwork 생성 결과 : ", result)
					reqSubnetId = result.IId // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] VNetwork 조회 테스트", reqSubnetId)
				result, err := VPCHandler.GetVPC(reqSubnetId)
				if err != nil {
					cblogger.Infof("[%s] VNetwork 조회 실패 : ", reqSubnetId, err)
				} else {
					cblogger.Infof("[%s] VNetwork 조회 결과 : [%s]", reqSubnetId, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] VNetwork 삭제 테스트", reqSubnetId)
				result, err := VPCHandler.DeleteVPC(reqSubnetId)
				if err != nil {
					cblogger.Infof("[%s] VNetwork 삭제 실패 : ", reqSubnetId, err)
				} else {
					cblogger.Infof("[%s] VNetwork 삭제 결과 : [%s]", reqSubnetId, result)
				}

			case 5:
				cblogger.Infof("[%s] Subnet 추가 테스트", vpcReqInfo.IId.NameId)
				result, err := VPCHandler.AddSubnet(subnetReqVpcInfo, subnetReqInfo)
				if err != nil {
					cblogger.Infof(reqSubnetId.NameId, " VNetwork 생성 실패 : ", err)
				} else {
					cblogger.Infof("VNetwork 생성 결과 : ", result)
					//reqSubnetId = result.IId // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 6:
				cblogger.Infof("[%s] Subnet 삭제 테스트", reqSubnetId.SystemId)
				result, err := VPCHandler.RemoveSubnet(subnetReqVpcInfo, reqSubnetId)
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

	ResourceHandler, err := getResourceHandler("Image")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.ImageHandler)

	imageReqInfo := irs.ImageReqInfo{
		//IId: irs.IID{NameId: "Test OS Image", SystemId: "ami-0c068f008ea2bdaa1"}, //Microsoft Windows Server 2019
		IId: irs.IID{NameId: "Test OS Image", SystemId: "ami-088da9557aae42f39"}, //Ubuntu Server 20.04 LTS (HVM), SSD Volume Type 64비트 x86
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
					cblogger.Infof("로그 레벨 : [%s]", cblog.GetLevel())
					//spew.Dump(result)
					cblogger.Info("출력 결과 수 : ", len(result))

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

/*
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
*/

func testErr() error {
	//return awserr.Error("")
	//return errors.New("")
	return awserr.New("504", "찾을 수 없음", nil)
}

// Test VM Lifecycle Management (Create/Suspend/Resume/Reboot/Terminate)
func handleVM() {
	cblogger.Debug("Start VMHandler Resource Test")

	ResourceHandler, err := getResourceHandler("VM")
	if err != nil {
		panic(err)
	}
	//handler := ResourceHandler.(irs2.ImageHandler)
	vmHandler := ResourceHandler.(irs.VMHandler)

	//config := readConfigFile()
	//VmID := irs.IID{NameId: config.Aws.BaseName, SystemId: config.Aws.VmID}
	VmID := irs.IID{SystemId: "i-0cea86282a9e2a569"}

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
					IId: irs.IID{NameId: "mcloud-barista-windows-test"},
					//ImageIID:          irs.IID{SystemId: "ami-001b6f8703b50e077"}, //centos-stable-7.2003.13-ebs-202005201235
					//ImageIID:          irs.IID{SystemId: "ami-059b6d3840b03d6dd"}, //Ubuntu Server 20.04 LTS (HVM)
					//ImageIID:          irs.IID{SystemId: "ami-09e67e426f25ce0d7"}, //Ubuntu Server 20.04 LTS (HVM) - 버지니아 북부 리전
					//ImageIID:          irs.IID{SystemId: "ami-059b6d3840b03d6dd"}, //Ubuntu Server 20.04 LTS (HVM)
					//ImageIID: irs.IID{SystemId: "ami-0fe22bffdec36361c"}, //Ubuntu Server 18.04 LTS (HVM) - Japan 리전
					ImageIID:          irs.IID{SystemId: "ami-093f427eb324bb754"}, //Microsoft Windows Server 2012 R2 RTM 64-bit Locale English AMI provided by Amazon - Japan 리전
					SubnetIID:         irs.IID{SystemId: "subnet-0a6ca346752be1ca4"},
					SecurityGroupIIDs: []irs.IID{{SystemId: "sg-0f4532a525ad09de1"}}, //3389 RDP 포트 Open
					VMSpecName:        "t2.micro",
					KeyPairIID:        irs.IID{SystemId: "japan-test"},
					VMUserPasswd:      "1234qwer!@#$", //윈도우즈용 비밀번호

					RootDiskType: "standard", //gp2/standard/io1/io2/sc1/st1/gp3
					//RootDiskType: "gp2", //gp2/standard/io1/io2/sc1/st1/gp3
					//RootDiskType: "gp3", //gp2/standard/io1/io2/sc1/st1/gp3
					//RootDiskSize: "60", //최소 8GB 이상이어야 함.
					//RootDiskSize: "1", //최소 8GB 이상이어야 함.
					//RootDiskSize: "Default", //8GB
				}

				vmInfo, err := vmHandler.StartVM(vmReqInfo)
				if err != nil {
					//panic(err)
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
					cblogger.Infof("=========== VM 목록 수 : [%d] ================", len(vmList))
					if len(vmList) > 0 {
						VmID = vmList[0].IId
					}
				}

			}
		}
	}
}

// Test VMSpec
func handleVMSpec() {
	cblogger.Debug("Start VMSpec Resource Test")

	ResourceHandler, err := getResourceHandler("VMSpec")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.VMSpecHandler)

	//config := readConfigFile()
	//reqVMSpec := config.Aws.VMSpec
	//reqVMSpec := "t2.small"	// GPU가 없음
	//reqVMSpec := "p3.2xlarge" // GPU 1개
	reqVMSpec := "p3.8xlarge" // GPU 4개

	//reqRegion := config.Aws.Region
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
					cblogger.Debug("VMSpec 목록 조회 결과")
					//spew.Dump(result)
					cblogger.Debug(result)
					cblogger.Infof("전체 목록 개수 : [%d]", len(result))
				}

				fmt.Println("Finish ListVMSpec()")

			case 2:
				fmt.Println("Start GetVMSpec() ...")
				result, err := handler.GetVMSpec(reqVMSpec)
				if err != nil {
					cblogger.Error(reqVMSpec, " VMSpec 정보 조회 실패 : ", err)
				} else {
					cblogger.Debugf("VMSpec[%s]  정보 조회 결과", reqVMSpec)
					//spew.Dump(result)
					cblogger.Debug(result)
				}
				fmt.Println("Finish GetVMSpec()")

			case 3:
				fmt.Println("Start ListOrgVMSpec() ...")
				result, err := handler.ListOrgVMSpec()
				if err != nil {
					cblogger.Error("VMSpec Org 목록 조회 실패 : ", err)
				} else {
					cblogger.Debug("VMSpec Org 목록 조회 결과")
					//spew.Dump(result)
					cblogger.Debug(result)
					//spew.Dump(result)
					//fmt.Println(result)
					//fmt.Println("=========================")
					//fmt.Println(result)
					cblogger.Infof("전체 목록 개수 : [%d]", len(result))
				}

				fmt.Println("Finish ListOrgVMSpec()")

			case 4:
				fmt.Println("Start GetOrgVMSpec() ...")
				result, err := handler.GetOrgVMSpec(reqVMSpec)
				if err != nil {
					cblogger.Error(reqVMSpec, " VMSpec Org 정보 조회 실패 : ", err)
				} else {
					cblogger.Debugf("VMSpec[%s] Org 정보 조회 결과", reqVMSpec)
					//spew.Dump(result)
					cblogger.Debug(result)
					//fmt.Println(result)
				}
				fmt.Println("Finish GetOrgVMSpec()")

			case 0:
				fmt.Println("Exit")
				return
			}
		}
	}
}

// Test NLB
func handleNLB() {
	cblogger.Debug("Start NLBHandler Resource Test")

	ResourceHandler, err := getResourceHandler("NLB")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.NLBHandler)

	nlbReqInfo := irs.NLBInfo{
		IId:    irs.IID{NameId: "cb-nlb-test01"},
		VpcIID: irs.IID{SystemId: "vpc-0c4d36a3ac3924419"},
		Type:   "PUBLIC",
		Scope:  "REGION",

		Listener: irs.ListenerInfo{
			Protocol: "TCP", // AWS NLB : TCP, TLS, UDP, or TCP_UDP
			//IP: "",
			Port: "22",
		},

		VMGroup: irs.VMGroupInfo{
			Protocol: "TCP", //TCP|UDP|HTTP|HTTPS
			Port:     "22",  //1-65535
			VMs:      &[]irs.IID{irs.IID{SystemId: "i-0dcbcbeadbb14212f"}, irs.IID{SystemId: "i-0cba8efe123ab0b42"}, irs.IID{SystemId: "i-010c858cbe5b6fe93"}},
		},

		HealthChecker: irs.HealthCheckerInfo{
			Protocol:  "TCP", // TCP|HTTP|HTTPS
			Port:      "22",  // Listener Port or 1-65535
			Interval:  30,    // TCP는 10이나 30만 가능 - secs, Interval time between health checks.
			Timeout:   0,     // TCP는 타임 아웃 설정 불가 - secs, Waiting time to decide an unhealthy VM when no response.
			Threshold: 10,    // num, The number of continuous health checks to change the VM status
		},
	} // nlbReqInfo

	reqAddVMs := &[]irs.IID{irs.IID{SystemId: "i-0dcbcbeadbb14212f"}}
	reqRemoveVMs := &[]irs.IID{irs.IID{SystemId: "i-0dcbcbeadbb14212f"}}

	for {
		fmt.Println("NLBHandler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. NLB List")
		fmt.Println("2. NLB Create")
		fmt.Println("3. NLB Get")
		fmt.Println("4. NLB Delete")

		fmt.Println("5. ChangeListener")
		fmt.Println("6. ChangeVMGroupInfo")
		fmt.Println("7. AddVMs")
		fmt.Println("8. RemoveVMs")
		fmt.Println("9. GetVMGroupHealthInfo")
		fmt.Println("10. ChangeHealthCheckerInfo")

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
					cblogger.Debug(result)
					cblogger.Infof("로그 레벨 : [%s]", cblog.GetLevel())
					//spew.Dump(result)
					cblogger.Info("출력 결과 수 : ", len(result))

					//조회및 삭제 테스트를 위해 리스트의 첫번째 정보의 ID를 요청ID로 자동 갱신함.
					if result != nil {
						nlbReqInfo.IId = result[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
					}
				}

			case 2:
				cblogger.Infof("[%s] NLB 생성 테스트", nlbReqInfo.IId.NameId)
				result, err := handler.CreateNLB(nlbReqInfo)
				if err != nil {
					cblogger.Infof(nlbReqInfo.IId.NameId, " NLB 생성 실패 : ", err)
				} else {
					cblogger.Infof("NLB 생성 성공 : ", result)
					nlbReqInfo.IId = result.IId // 조회 및 삭제를 위해 생성된 ID로 변경
					if cblogger.Level.String() == "debug" {
						spew.Dump(result)
					}
				}

			case 3:
				cblogger.Infof("[%s] NLB 조회 테스트", nlbReqInfo.IId)
				result, err := handler.GetNLB(nlbReqInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] NLB 조회 실패 : ", nlbReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] NLB 조회 성공 : [%s]", nlbReqInfo.IId.NameId, result)
					if cblogger.Level.String() == "debug" {
						spew.Dump(result)
					}
				}

			case 4:
				cblogger.Infof("[%s] NLB 삭제 테스트", nlbReqInfo.IId.NameId)
				result, err := handler.DeleteNLB(nlbReqInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] NLB 삭제 실패 : ", nlbReqInfo.IId.NameId, err)
				} else {
					cblogger.Info("성공")
					cblogger.Infof("[%s] NLB 삭제 성공 : [%s]", nlbReqInfo.IId.NameId, result)
				}

			case 5:
				cblogger.Infof("[%s] 리스너 변경 테스트", nlbReqInfo.IId)
				reqListenerInfo := irs.ListenerInfo{
					Protocol: "TCP", // AWS NLB : TCP, TLS, UDP, or TCP_UDP
					//IP: "",
					Port: "80",
				}
				result, err := handler.ChangeListener(nlbReqInfo.IId, reqListenerInfo)
				if err != nil {
					cblogger.Infof("[%s] 리스너 변경 실패 : ", nlbReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] 리스너 변경 성공 : [%s]", nlbReqInfo.IId.NameId, result)
					if cblogger.Level.String() == "debug" {
						spew.Dump(result)
					}
				}

			case 7:
				cblogger.Infof("[%s] AddVMs 테스트", nlbReqInfo.IId.NameId)
				cblogger.Info(reqAddVMs)
				result, err := handler.AddVMs(nlbReqInfo.IId, reqAddVMs)
				if err != nil {
					cblogger.Infof("[%s] AddVMs 실패 : ", nlbReqInfo.IId.NameId, err)
				} else {
					cblogger.Info("성공")
					cblogger.Infof("[%s] AddVMs 성공 : [%s]", nlbReqInfo.IId.NameId, result)
				}

			case 8:
				cblogger.Infof("[%s] RemoveVMs 테스트", nlbReqInfo.IId.NameId)
				cblogger.Info(reqRemoveVMs)
				result, err := handler.RemoveVMs(nlbReqInfo.IId, reqRemoveVMs)
				if err != nil {
					cblogger.Infof("[%s] RemoveVMs 실패 : ", nlbReqInfo.IId.NameId, err)
				} else {
					cblogger.Info("성공")
					cblogger.Infof("[%s] RemoveVMs 성공 : [%s]", nlbReqInfo.IId.NameId, result)
				}

			case 9:
				cblogger.Infof("[%s] GetVMGroupHealthInfo 테스트", nlbReqInfo.IId)
				result, err := handler.GetVMGroupHealthInfo(nlbReqInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] GetVMGroupHealthInfo 실패 : ", nlbReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] GetVMGroupHealthInfo 성공 : [%s]", nlbReqInfo.IId.NameId, result)
					if cblogger.Level.String() == "debug" {
						spew.Dump(result)
					}
				}

			case 6:
				cblogger.Infof("[%s] NLB VM Group 변경 테스트", nlbReqInfo.IId.NameId)
				result, err := handler.ChangeVMGroupInfo(nlbReqInfo.IId, irs.VMGroupInfo{
					Protocol: "TCP",
					Port:     "8080",
				})
				if err != nil {
					cblogger.Infof("[%s] NLB VM Group 변경 실패 : ", nlbReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] NLB VM Group 변경 성공 : [%s]", nlbReqInfo.IId.NameId, result)
					if cblogger.Level.String() == "debug" {
						spew.Dump(result)
					}
				}
			case 10:
				cblogger.Infof("[%s] NLB Health Checker 변경 테스트", nlbReqInfo.IId.NameId)
				result, err := handler.ChangeHealthCheckerInfo(nlbReqInfo.IId, irs.HealthCheckerInfo{
					Protocol: "TCP",
					Port:     "22",
					//Interval: 10, //미지원
					//Timeout:   3,	//미지원
					Threshold: 5,
				})
				if err != nil {
					cblogger.Infof("[%s] NLB Health Checker 변경 실패 : ", nlbReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] NLB Health Checker 변경 성공 : [%s]", nlbReqInfo.IId.NameId, result)
					if cblogger.Level.String() == "debug" {
						spew.Dump(result)
					}
				}
			}
		}
	}
}

func main() {
	cblogger.Info("AWS Resource Test")
	//handleVPC()
	//handleKeyPair()
	//handlePublicIP() // PublicIP 생성 후 conf
	//handleSecurity()
	handleVM()

	//handleImage() //AMI
	//handleVNic() //Lancard
	//handleVMSpec()
	//handleNLB()
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
			Zone:   config.Aws.Zone,
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
	case "NLB":
		resourceHandler, err = cloudConnection.CreateNLBHandler()
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
			Zone:   config.Aws.Zone,
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

func setVPCHandler() (irs.VPCHandler, error) {
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
			Zone:   config.Aws.Zone,
		},
	}

	cloudConnection, err := cloudDriver.ConnectCloud(connectionInfo)
	if err != nil {
		return nil, err
	}

	handler, err := cloudConnection.CreateVPCHandler()
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
		Zone               string `yaml:"zone"`

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
	cblogger.Infof("Test Data 설정파일 : [%]", rootPath+"/config/config.yaml")

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
	//cblogger.Info(config)
	cblogger.Debug(config.Aws.AawsAccessKeyID, " ", config.Aws.Region)
	//cblogger.Debug(config.Aws.Region)
	return config
}
