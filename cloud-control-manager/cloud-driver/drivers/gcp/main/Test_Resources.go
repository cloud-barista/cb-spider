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

	cblog "github.com/cloud-barista/cb-log"
	//testconf "./conf"
	testconf "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/gcp/main/conf"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("GCP Resource Test")
	cblog.SetLevel("debug")
}

// Test SecurityHandler
func handleSecurityOld() {
	cblogger.Debug("Start handler")

	ResourceHandler, err := testconf.GetResourceHandler("Security")
	if err != nil {
		panic(err)
	}

	handler := ResourceHandler.(irs.SecurityHandler)

	securityId := "sg1234"
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
		VpcIID: irs.IID{
			NameId:   "vpc-11",
			SystemId: "vpc-11",
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

func handleSecurity() {
	cblogger.Debug("Start Security Resource Test")

	ResourceHandler, err := testconf.GetResourceHandler("Security")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.SecurityHandler)

	//config := readConfigFile()
	//VmID := config.Aws.VmID

	securityName := "cb-securitytest-all"
	securityId := "cb-securitytest-all"
	//securityId := "cb-secu-all"
	vpcId := "cb-vpc"

	for {
		fmt.Println("Security Management")
		fmt.Println("0. Quit")
		fmt.Println("1. Security List")
		fmt.Println("2. Security Create")
		fmt.Println("3. Security Get")
		fmt.Println("4. Security Delete")

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
					cblogger.Infof("전체 목록 수 : [%d]", len(result))
				}

			case 2:
				cblogger.Infof("[%s] Security 생성 테스트", securityName)
				securityReqInfo := irs.SecurityReqInfo{
					IId:    irs.IID{NameId: securityName},
					VpcIID: irs.IID{SystemId: vpcId},
					SecurityRules: &[]irs.SecurityRuleInfo{ //보안 정책 설정
						{
							// All Port로 등록
							FromPort:   "",
							ToPort:     "",
							IPProtocol: "icmp", //icmp는 포트 정보가 없음
							Direction:  "inbound",
						},
						{
							//20-22 Prot로 등록
							FromPort:   "20",
							ToPort:     "22",
							IPProtocol: "tcp",
							Direction:  "inbound",
						},
						{
							// 80 Port로 등록
							FromPort:   "80",
							ToPort:     "80",
							IPProtocol: "tcp",
							Direction:  "inbound",
						},
						{
							// 8080 Port로 등록
							FromPort:   "8080",
							ToPort:     "-1", //FromPort나 ToPort중 하나에 -1이 입력될 경우 -1이 입력된 경우 -1을 공백으로 처리
							IPProtocol: "tcp",
							Direction:  "inbound",
						},
						{ // 1323 Prot로 등록
							FromPort:   "-1", //FromPort나 ToPort중 하나에 -1이 입력될 경우 -1이 입력된 경우 -1을 공백으로 처리
							ToPort:     "1323",
							IPProtocol: "tcp",
							Direction:  "inbound",
						},
						{ // 1024 Prot로 등록
							FromPort:   "",
							ToPort:     "1024",
							IPProtocol: "tcp",
							Direction:  "inbound",
						},
						{ // 1234 Prot로 등록
							FromPort:   "1234",
							ToPort:     "",
							IPProtocol: "tcp",
							Direction:  "inbound",
						},
						/*
							{ // 모든 프로토콜 모든 포트로 등록
								//FromPort:   "",
								//ToPort:     "",
								IPProtocol: "all", // 모두 허용 (포트 정보 없음)
								Direction:  "inbound",
							},
						*/
						/*
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
					},
				}

				result, err := handler.CreateSecurity(securityReqInfo)
				if err != nil {
					cblogger.Infof(securityName, " Security 생성 실패 : ", err)
				} else {
					cblogger.Infof("[%s] Security 생성 결과 : [%v]", securityName, result)
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
	handler := ResourceHandler.(irs.ImageHandler)

	//imageReqInfo := irs2.ImageReqInfo{
	imageReqInfo := irs.ImageReqInfo{
		IId: irs.IID{
			//NameId: "Test OS Image",
			//SystemId: "vmsg02-asia-northeast1-b",
			//SystemId: "https://www.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/images/ubuntu-minimal-1804-bionic-v20200415",
			SystemId: "https://www.googleapis.com/compute/v1/projects/gce-uefi-images/global/images/centos-7-v20190905",
			//SystemId: "2076268724445164462", //centos-7-v20190204
			//SystemId: "ubuntu-minimal-1804-bionic-v20200415",
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

// Test handleVPC (VPC)
func handleVPC() {
	cblogger.Debug("Start VPC Resource Test")

	ResourceHandler, err := testconf.GetResourceHandler("VPCHandler")
	if err != nil {
		panic(err)
	}
	cblogger.Debug("=============================")
	handler := ResourceHandler.(irs.VPCHandler)

	/*
		vpcReqInfo := irs.VPCReqInfo{
			IId: irs.IID{NameId: "cb-vpc"},
			//IPv4_CIDR: "10.0.0.0/16",
			SubnetInfoList: []irs.SubnetInfo{
				{
					IId:       irs.IID{NameId: "cb-sub1"},
					IPv4_CIDR: "10.0.3.0/24",
				},

				{
					IId:       irs.IID{NameId: "cb-sub2"},
					IPv4_CIDR: "10.0.4.0/24",
				},
			},
			//Id:   "subnet-044a2b57145e5afc5",
			//Name: "CB-VNet-Subnet", // 웹 도구 등 외부에서 전달 받지 않고 드라이버 내부적으로 자동 구현때문에 사용하지 않음.
			//CidrBlock: "10.0.0.0/16",
			//CidrBlock: "192.168.0.0/16",
		}
	*/

	subnetReqInfo := irs.SubnetInfo{
		IId:       irs.IID{NameId: "addtest-subnet"},
		IPv4_CIDR: "192.168.5.0/24",
	}

	subnetReqVpcInfo := irs.IID{SystemId: "vpc-gcp-eu-ns-common"}
	reqSubnetId := irs.IID{SystemId: "addtest-subnet"}
	cblogger.Debug(subnetReqInfo)
	cblogger.Debug(subnetReqVpcInfo)
	cblogger.Debug(reqSubnetId)

	vpcReqInfo := irs.VPCReqInfo{
		IId: irs.IID{NameId: "cb-vpc-load-test"},
		SubnetInfoList: []irs.SubnetInfo{
			{
				IId:       irs.IID{NameId: "vpc-loadtest-sub1"},
				IPv4_CIDR: "10.0.3.0/24",
			},
			{
				IId:       irs.IID{NameId: "vpc-loadtest-sub2"},
				IPv4_CIDR: "10.0.4.0/24",
			},
			{
				IId:       irs.IID{NameId: "vpc-loadtest-sub3"},
				IPv4_CIDR: "10.0.5.0/24",
			},
			{
				IId:       irs.IID{NameId: "vpc-loadtest-sub4"},
				IPv4_CIDR: "10.0.6.0/24",
			},
			{
				IId:       irs.IID{NameId: "vpc-loadtest-sub5"},
				IPv4_CIDR: "10.0.7.0/24",
			},
		},
		//CidrBlock: "10.0.0.0/16",
		//CidrBlock: "192.168.0.0/16",
	}

	//reqVpcId := irs.IID{SystemId: "vpc-6we11xwqjc9tyma5i68z0"}
	reqVpcId := irs.IID{SystemId: vpcReqInfo.IId.NameId} //GCP는 SystemId에 NameId사용

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
						reqVpcId = result[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
					}
					cblogger.Infof("전체 VPC 목록 수 : [%d]", len(result))
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
					cblogger.Infof("Subnet 목록 수 : [%d]", len(result.SubnetInfoList))
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
					cblogger.Infof(reqSubnetId.NameId, " VNetwork 생성 실패 : ", err)
				} else {
					cblogger.Infof("VNetwork 생성 결과 : ", result)
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

// Test KeyPair
func handleKeyPairOld() {
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
					cblogger.Infof("키 페어 목록 수 : [%d]", len(result))
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
					//n1-standard-64
					if len(result) > 0 {
						machinename = result[0].Name
					}
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
	VmID := irs.IID{SystemId: "mcloud-barista-vm-test"}

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
					ImageIID: irs.IID{
						NameId: "Test",
						//SystemId: "ubuntu-minimal-1804-bionic-v20200415",
						//SystemId: "https://www.googleapis.com/compute/v1/projects/gce-uefi-images/global/images/centos-7-v20190204",
						SystemId: "https://www.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/images/ubuntu-1804-bionic-v20200521",

						//NameId:   "projects/ubuntu-os-cloud/global/images/ubuntu-minimal-1804-bionic-v20200415",
						//SystemId: "projects/ubuntu-os-cloud/global/images/ubuntu-minimal-1804-bionic-v20191024",
						//SystemId: "2076268724445164462",	//에러
						//SystemId: "ubuntu-minimal-1804-bionic-v20191024",
						//https://www.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/images/ubuntu-1804-bionic-v20200521
						//NameId:   "https://www.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/images/ubuntu-minimal-1804-bionic-v20200415",
						//SystemId: "https://www.googleapis.com/compute/v1/projects/ubuntu-os-cloud/global/images/ubuntu-minimal-1804-bionic-v20200415",
					},
					VpcIID:            irs.IID{SystemId: "cb-vpc"},
					SubnetIID:         irs.IID{SystemId: "cb-sub1"},
					SecurityGroupIIDs: []irs.IID{{SystemId: "cb-securitytest1"}},
					VMSpecName:        "f1-micro",
					KeyPairIID:        irs.IID{SystemId: "cb-keypairtest123123"},
					VMUserId:          "cb-user",
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
					if len(vmList) > 0 {
						VmID = vmList[0].IId
					}
				}

			}
		}
	}
}

//import "path/filepath"

func main() {
	cblogger.Info("GCP Resource Test")
	handleVPC()
	//handleVMSpec()
	//handleImage() //AMI
	//handleKeyPair()
	//handleSecurity()
	//handleVM()
	//cblogger.Info(filepath.Join("a/b", "\\cloud-driver-libs\\.ssh-gcp\\"))
	//cblogger.Info(filepath.Join("\\cloud-driver-libs\\.ssh-gcp\\", "/b/c/d"))
}
