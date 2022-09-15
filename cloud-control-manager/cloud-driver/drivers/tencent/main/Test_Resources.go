// Tencent Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Tencent Driver.
//
// by CB-Spider Team, 2022.09.

package main

import (
	"fmt"

	testconf "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/tencent/main/conf"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	cblog "github.com/cloud-barista/cb-log"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("TencentCloud Resource Test")
	cblog.SetLevel("debug")
}

// Test VMSpec
func handleVMSpec() {
	cblogger.Debug("Start VMSpec Resource Test")

	ResourceHandler, err := testconf.GetResourceHandler("VMSpec")
	if err != nil {
		//panic(err)
		cblogger.Error(err)
		return
	}

	handler := ResourceHandler.(irs.VMSpecHandler)

	//config := testconf.ReadConfigFile()
	reqVMSpec := "C2.4XLARGE64" // GPU 1개

	//reqZone := config.Tencent.Zone

	//cblogger.Info("reqZone : ", reqZone)
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
					cblogger.Info("출력 결과 수 : ", len(result))
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

func handleSecurity() {
	cblogger.Debug("Start Security Resource Test")

	ResourceHandler, err := testconf.GetResourceHandler("Security")
	if err != nil {
		//panic(err)
		cblogger.Error(err)
	}
	handler := ResourceHandler.(irs.SecurityHandler)

	securityName := "sg20"
	securityId := "sg-5m5pezaj"
	vpcId := "vpc-f3teez1l"

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
						{
							FromPort:   "-1",
							ToPort:     "-1",
							IPProtocol: "all",
							Direction:  "inbound",
							CIDR:       "0.0.0.0/0",
						},
						/*{
							FromPort:   "-1",
							ToPort:     "-1",
							IPProtocol: "all",
							Direction:  "inbound",
							CIDR:       "0.0.0.0/0",
						},*/

						/*{
							FromPort:   "8080",
							ToPort:     "",
							IPProtocol: "tcp",
							Direction:  "inbound",
							CIDR:       "0.0.0.0/0",
						},*/

						// {
						// 	FromPort:   "40",
						// 	ToPort:     "",
						// 	IPProtocol: "tcp",
						// 	Direction:  "outbound",
						// 	CIDR:       "10.13.1.10/32",
						// },
						/*
							{
								FromPort:   "20",
								ToPort:     "22",
								IPProtocol: "tcp",
								Direction:  "inbound",
							},

							{
								FromPort:   "80",
								ToPort:     "",
								IPProtocol: "tcp",
								Direction:  "inbound",
							},
							{
								FromPort:   "8080",
								ToPort:     "",
								IPProtocol: "tcp",
								Direction:  "inbound",
							},
							{
								FromPort:   "ALL",
								ToPort:     "",
								IPProtocol: "icmp",
								Direction:  "inbound",
							},
						*/

						// {
						// 	FromPort:   "443",
						// 	ToPort:     "",
						// 	IPProtocol: "tcp",
						// 	Direction:  "outbound",
						// },
						// {
						// 	FromPort:   "8443",
						// 	ToPort:     "9999",
						// 	IPProtocol: "tcp",
						// 	Direction:  "outbound",
						// },
						/*
							{
								//FromPort:   "8443",
								//ToPort:     "9999",
								IPProtocol: "ALL", // 모두 허용 (포트 정보 없음)
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
				cblogger.Infof("[%s] Rule 추가 테스트", securityId)
				securityRules := &[]irs.SecurityRuleInfo{

					{
						//20-22 Prot로 등록
						FromPort:   "-1",
						ToPort:     "-1",
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},
					// {
					// 	//20-22 Prot로 등록
					// 	FromPort:   "22",
					// 	ToPort:     "22",
					// 	IPProtocol: "tcp",
					// 	Direction:  "inbound",
					// 	CIDR:       "0.0.0.0/0",
					// },
					// {
					// 	FromPort:   "88",
					// 	ToPort:     "90",
					// 	IPProtocol: "tcp",
					// 	Direction:  "inbound",
					// 	CIDR:       "0.0.0.0/0",
					// },
					/*{
						FromPort:   "3000",
						ToPort:     "3000",
						IPProtocol: "udp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						FromPort:   "3000",
						ToPort:     "3000",
						IPProtocol: "udp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},*/
					// {
					// 	FromPort:   "1000",
					// 	ToPort:     "",
					// 	IPProtocol: "udp",
					// 	Direction:  "inbound",
					// 	CIDR:       "0.0.0.0/0",
					// },
					/*{
						// 8080 Port로 등록
						FromPort:   "8080",
						ToPort:     "-1", //FromPort나 ToPort중 하나에 -1이 입력될 경우 -1이 입력된 경우 -1을 공백으로 처리
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{ // 1323 Prot로 등록
						FromPort:   "-1", //FromPort나 ToPort중 하나에 -1이 입력될 경우 -1이 입력된 경우 -1을 공백으로 처리
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
					/*{
						FromPort:   "-1",
						ToPort:     "-1",
						IPProtocol: "all",
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
					// {
					// 	FromPort:   "22",
					// 	ToPort:     "22",
					// 	IPProtocol: "tcp",
					// 	Direction:  "inbound",
					// 	CIDR:       "0.0.0.0/0",
					// },
					// {
					// 	FromPort:   "1000",
					// 	ToPort:     "1000",
					// 	IPProtocol: "tcp",
					// 	Direction:  "inbound",
					// 	CIDR:       "0.0.0.0/0",
					// },
					// {
					// 	FromPort:   "1",
					// 	ToPort:     "65535",
					// 	IPProtocol: "udp",
					// 	Direction:  "inbound",
					// 	CIDR:       "0.0.0.0/0",
					// },
					// {
					// 	FromPort:   "-1",
					// 	ToPort:     "-1",
					// 	IPProtocol: "icmp",
					// 	Direction:  "inbound",
					// 	CIDR:       "0.0.0.0/0",
					// },
					// {
					// 	FromPort:   "22",
					// 	ToPort:     "22",
					// 	IPProtocol: "tcp",
					// 	Direction:  "outbound",
					// 	CIDR:       "0.0.0.0/0",
					// },
					// {
					// 	FromPort:   "1000",
					// 	ToPort:     "1000",
					// 	IPProtocol: "tcp",
					// 	Direction:  "outbound",
					// 	CIDR:       "0.0.0.0/0",
					// },
					// {
					// 	FromPort:   "1",
					// 	ToPort:     "65535",
					// 	IPProtocol: "udp",
					// 	Direction:  "outbound",
					// 	CIDR:       "0.0.0.0/0",
					// },
					// {
					// 	FromPort:   "-1",
					// 	ToPort:     "-1",
					// 	IPProtocol: "icmp",
					// 	Direction:  "outbound",
					// 	CIDR:       "0.0.0.0/0",
					// },

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
						IPProtocol: "udp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						// 8080 Port로 등록
						FromPort:   "8080",
						ToPort:     "-1", //FromPort나 ToPort중 하나에 -1이 입력될 경우 -1이 입력된 경우 -1을 공백으로 처리
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{ // 1323 Prot로 등록
						FromPort:   "-1", //FromPort나 ToPort중 하나에 -1이 입력될 경우 -1이 입력된 경우 -1을 공백으로 처리
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
					/*{
						FromPort:   "-1",
						ToPort:     "-1",
						IPProtocol: "all",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						// 80 Port로 등록
						FromPort:   "88",
						ToPort:     "90",
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						FromPort:   "22",
						ToPort:     "22",
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						FromPort:   "1000",
						ToPort:     "1000",
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						FromPort:   "1",
						ToPort:     "65535",
						IPProtocol: "udp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						FromPort:   "-1",
						ToPort:     "-1",
						IPProtocol: "icmp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},*/
					{
						FromPort:   "-10",
						ToPort:     "-10",
						IPProtocol: "tcp",
						Direction:  "inbound",
						CIDR:       "0.0.0.0/0",
					},
					/*{
						FromPort:   "-1",
						ToPort:     "-1",
						IPProtocol: "all",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},*/
					/*{
						FromPort:   "22",
						ToPort:     "22",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						FromPort:   "1000",
						ToPort:     "1000",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						FromPort:   "1",
						ToPort:     "65535",
						IPProtocol: "udp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						FromPort:   "-1",
						ToPort:     "-1",
						IPProtocol: "icmp",
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
		//panic(err)
		cblogger.Error(err)
	}
	handler := ResourceHandler.(irs.KeyPairHandler)

	//KeyPair 생성은 알파벳, 숫자 또는 밑줄 "_"만 지원
	keyPairName := "CB_KeyPairTest123123"
	keyPairId := ""

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
						keyPairId = result[0].IId.SystemId // 조회 및 삭제를 위해 생성된 ID로 변경
						keyPairName = result[0].IId.NameId
					}
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
					keyPairId = result.IId.SystemId
					spew.Dump(result)
				}
			case 3:
				cblogger.Infof("[%s] 키 페어 조회 테스트", keyPairName)
				result, err := handler.GetKey(irs.IID{SystemId: keyPairId})
				if err != nil {
					cblogger.Infof(keyPairName, " 키 페어 조회 실패 : ", err)
				} else {
					cblogger.Infof("[%s] 키 페어 조회 결과 : [%s]", keyPairName, result)
					keyPairName = result.IId.NameId

					spew.Dump(result)
				}
			case 4:
				cblogger.Infof("[%s] 키 페어 삭제 테스트", keyPairName)
				result, err := handler.DeleteKey(irs.IID{SystemId: keyPairId})
				if err != nil {
					cblogger.Infof(keyPairName, " 키 페어 삭제 실패 : ", err)
				} else {
					cblogger.Infof("[%s] 키 페어 삭제 결과 : [%s]", keyPairName, result)
				}
			}
		}
	}
}

func handleVPC() {
	cblogger.Debug("Start VPC Resource Test")
	ResourceHandler, err := testconf.GetResourceHandler("VPC")
	if err != nil {
		//panic(err)
		cblogger.Error(err)
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

	reqVpcId := irs.IID{SystemId: "vpc-2u04wg6k"}

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
						if len(result[0].SubnetInfoList) > 0 {
							reqSubnetId = result[0].SubnetInfoList[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
						}
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
					reqSubnetId = result.SubnetInfoList[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
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
		//panic(err)
		cblogger.Error(err)
	}
	handler := ResourceHandler.(irs.ImageHandler)

	imageReqInfo := irs.ImageReqInfo{
		IId: irs.IID{NameId: "Test OS Image", SystemId: "ami-047f7b46bd6dd5d84"},
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
					cblogger.Info(result)
					cblogger.Info("출력 결과 수 : ", len(result))
					spew.Dump(result)

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
		//panic(err)
		cblogger.Error(err)
	}
	vmHandler := ResourceHandler.(irs.VMHandler)
	VmID := irs.IID{SystemId: "ins-rqoo65fo"}

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
					//IId:      irs.IID{NameId: "bill-test"},
					//ImageIID: irs.IID{SystemId: "img-22trbn9x"}, //Ubuntu Server 20.04 LTS 64

					ImageIID:          irs.IID{SystemId: "img-9x5o844i"}, //Ubuntu Server 18.04.1 LTS 64
					VpcIID:            irs.IID{SystemId: "vpc-g3imdykc"},
					SubnetIID:         irs.IID{SystemId: "subnet-rlr71m6n"}, //Zone2
					SecurityGroupIIDs: []irs.IID{{SystemId: "sg-j43bvarj"}},
					VMSpecName:        "SA2.MEDIUM2",
					KeyPairIID:        irs.IID{SystemId: "skey-cp2013rp"}, //cb_user_test
					//VMUserId:          "root", //root만 가능
					//VMUserPasswd: "Cbuser!@#", //대문자 소문자 모두 사용되어야 함. 그리고 숫자나 특수 기호 중 하나가 포함되어야 함.
					//RootDiskType: "CLOUD_PREMIUM", //LOCAL_BASIC/LOCAL_SSD/CLOUD_BASIC/CLOUD_SSD/CLOUD_PREMIUM
					RootDiskType: "CLOUD_PREMIUM", //LOCAL_BASIC/LOCAL_SSD/CLOUD_BASIC/CLOUD_SSD/CLOUD_PREMIUM
					RootDiskSize: "60",            //Image Size 보다 작으면 에러 남
					//RootDiskSize: "Default", //Image Size 보다 작으면 에러 남
					//DataDiskIIDs: []irs.IID{{SystemId: "disk-obk07o6e"}},
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

func handleNLB() {
	cblogger.Debug("Start NLB Resource Test")
	ResourceHandler, err := testconf.GetResourceHandler("NLB")
	if err != nil {
		//panic(err)
		cblogger.Error(err)
	}
	handler := ResourceHandler.(irs.NLBHandler)

	nlbReqInfo := irs.NLBInfo{

		IId:           irs.IID{NameId: "New-CB-NLB03"},
		VpcIID:        irs.IID{SystemId: "vpc-i614yona"},
		Type:          "PUBLIC",
		Listener:      irs.ListenerInfo{Protocol: "TCP", Port: "80"},
		HealthChecker: irs.HealthCheckerInfo{Port: "1234"},
		VMGroup: irs.VMGroupInfo{
			Protocol: "TCP",
			Port:     "80",
			VMs:      &[]irs.IID{{SystemId: "ins-5tf50w2x"}, {SystemId: "ins-lqds5b1h"}},
		},
	}

	reqNLBId := irs.IID{SystemId: "lb-qfipv1il"}

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
				cblogger.Infof("[%s] NLB 조회 테스트", reqNLBId)
				result, err := handler.GetNLB(reqNLBId)
				if err != nil {
					cblogger.Infof("[%s] NLB 조회 실패 : ", reqNLBId, err)
				} else {
					cblogger.Infof("[%s] NLB 조회 결과 : [%s]", reqNLBId, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] NLB 삭제 테스트", reqNLBId)
				result, err := handler.DeleteNLB(reqNLBId)
				if err != nil {
					cblogger.Infof("[%s] NLB 삭제 실패 : ", reqNLBId, err)
				} else {
					cblogger.Infof("[%s] NLB 삭제 결과 : [%s]", reqNLBId, result)
				}

			case 5:
				cblogger.Infof("[%s] VM 추가 테스트", reqNLBId)
				result, err := handler.AddVMs(reqNLBId, &[]irs.IID{{SystemId: "ins-lqds5b1h"}})
				if err != nil {
					cblogger.Infof("VM 추가 실패 : ", err)
				} else {
					cblogger.Infof("VM 추가 결과 : ", result)
					//reqSubnetId = result.SubnetInfoList[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 6:
				cblogger.Infof("[%s] VM 삭제 테스트", reqNLBId.SystemId)
				result, err := handler.RemoveVMs(reqNLBId, &[]irs.IID{{SystemId: "ins-lqds5b1h"}})
				if err != nil {
					cblogger.Infof("VM 삭제 실패 : ", err)
				} else {
					cblogger.Infof("VM 삭제 결과 : [%s]", result)
				}
			case 7:
				cblogger.Infof("[%s] NLB VM Health 조회 테스트", reqNLBId)
				result, err := handler.GetVMGroupHealthInfo(reqNLBId)
				if err != nil {
					cblogger.Infof("[%s] NLB VM Health 조회 실패 : ", reqNLBId.SystemId, err)
				} else {
					cblogger.Infof("[%s] NLB VM Health 조회 결과 : [%s]", reqNLBId.SystemId, result)
					spew.Dump(result)
				}
			case 8:
				cblogger.Infof("[%s] NLB Listener 변경 테스트", reqNLBId)
				result, err := handler.ChangeListener(reqNLBId, irs.ListenerInfo{})
				if err != nil {
					cblogger.Infof("[%s] NLB Listener 변경 실패 : ", reqNLBId.SystemId, err)
				} else {
					cblogger.Infof("[%s] NLB Listener 변경 결과 : [%s]", reqNLBId.SystemId, result)
					spew.Dump(result)
				}
			case 9:
				cblogger.Infof("[%s] NLB VM Group 변경 테스트", reqNLBId)
				result, err := handler.ChangeVMGroupInfo(reqNLBId, irs.VMGroupInfo{
					//Protocol: "TCP",
					Port: "8080",
				})
				if err != nil {
					cblogger.Infof("[%s] NLB VM Group 변경 실패 : ", reqNLBId.SystemId, err)
				} else {
					cblogger.Infof("[%s] NLB VM Group 변경 결과 : [%s]", reqNLBId.SystemId, result)
					spew.Dump(result)
				}
			case 10:
				cblogger.Infof("[%s] NLB Health Checker 변경 테스트", reqNLBId)
				result, err := handler.ChangeHealthCheckerInfo(reqNLBId, irs.HealthCheckerInfo{
					Protocol:  "HTTP",
					Port:      "80",
					Interval:  10,
					Timeout:   5,
					Threshold: 5,
				})
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

func handleDisk() {
	cblogger.Debug("Start DiskHandler Resource Test")

	ResourceHandler, err := testconf.GetResourceHandler("Disk")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.DiskHandler)

	diskReqInfo := irs.DiskInfo{
		IId:      irs.IID{NameId: "cb-disk-01"},
		DiskType: "CLOUD_PREMIUM",
		DiskSize: "20",
	}

	for {
		fmt.Println("DiskHandler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. Disk List")
		fmt.Println("2. Disk Create")
		fmt.Println("3. Disk Get")
		fmt.Println("4. Disk Change Size")
		fmt.Println("5. Disk Delete")
		fmt.Println("6. Disk Attach")
		fmt.Println("7. Disk Detach")

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
				result, err := handler.ListDisk()
				if err != nil {
					cblogger.Infof(" Disk 목록 조회 실패 : ", err)
				} else {
					cblogger.Info("Disk 목록 조회 결과")
					cblogger.Info(result)
					cblogger.Info("출력 결과 수 : ", len(result))
					spew.Dump(result)
					//spew.Dump(result)

					//조회및 삭제 테스트를 위해 리스트의 첫번째 정보의 ID를 요청ID로 자동 갱신함.
					// if result != nil {
					// 	diskReqInfo.IId = result[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
					// }
				}

			case 2:
				cblogger.Infof("[%s] Disk 생성 테스트", diskReqInfo.IId.NameId)
				//vNetworkReqInfo := irs.VNetworkReqInfo{}
				result, err := handler.CreateDisk(diskReqInfo)
				if err != nil {
					cblogger.Infof(diskReqInfo.IId.NameId, " Disk 생성 실패 : ", err)
				} else {
					cblogger.Infof("Disk 생성 결과 : ", result)
					diskReqInfo.IId = result.IId // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] Disk 조회 테스트", diskReqInfo.IId.NameId)
				result, err := handler.GetDisk(diskReqInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] Disk 조회 실패 : ", diskReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] Disk 조회 결과 : [%s]", diskReqInfo.IId.NameId, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] Disk Size 변경 테스트", diskReqInfo.IId.NameId)
				result, err := handler.ChangeDiskSize(diskReqInfo.IId, "30")
				if err != nil {
					cblogger.Infof("[%s] Disk Size 변경 실패 : ", diskReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] Disk Size 변경 결과 : [%s]", diskReqInfo.IId.NameId, result)
				}
			case 5:
				cblogger.Infof("[%s] Disk 삭제 테스트", diskReqInfo.IId.NameId)
				result, err := handler.DeleteDisk(diskReqInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] Disk 삭제 실패 : ", diskReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] Disk 삭제 결과 : [%s]", diskReqInfo.IId.NameId, result)
				}
			case 6:
				cblogger.Infof("[%s] Disk Attach 테스트", diskReqInfo.IId.NameId)
				result, err := handler.AttachDisk(diskReqInfo.IId, irs.IID{SystemId: "ins-fptlw6mc"})
				if err != nil {
					cblogger.Infof("[%s] Disk Attach 실패 : ", diskReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] Disk Attach 결과 : [%s]", diskReqInfo.IId.NameId, result)
					spew.Dump(result)
				}
			case 7:
				cblogger.Infof("[%s] Disk Detach 테스트", diskReqInfo.IId.NameId)
				result, err := handler.DetachDisk(diskReqInfo.IId, irs.IID{SystemId: "mcloud-barista-vm-test"})
				if err != nil {
					cblogger.Infof("[%s] Disk Detach 실패 : ", diskReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] Disk Detach 결과 : [%s]", diskReqInfo.IId.NameId, result)
				}
			}
		}
	}
}

func handleMyImage() {
	cblogger.Debug("Start MyImageHandler Resource Test")

	ResourceHandler, err := testconf.GetResourceHandler("MyImage")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.MyImageHandler)

	myImageReqInfo := irs.MyImageInfo{
		IId:      irs.IID{NameId: "cb-myimage-03", SystemId: "img-9x5o844i"},
		SourceVM: irs.IID{SystemId: "ins-fptlw6mc"},
	}

	for {
		fmt.Println("MyImageHandler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. MyImage List")
		fmt.Println("2. MyImage Create")
		fmt.Println("3. MyImage Get")
		fmt.Println("4. MyImage Delete")

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
				result, err := handler.ListMyImage()
				if err != nil {
					cblogger.Infof(" MyImage 목록 조회 실패 : ", err)
				} else {
					cblogger.Info("MyImage 목록 조회 결과")
					cblogger.Info(result)
					cblogger.Info("출력 결과 수 : ", len(result))
					spew.Dump(result)
					//spew.Dump(result)

					//조회및 삭제 테스트를 위해 리스트의 첫번째 정보의 ID를 요청ID로 자동 갱신함.
					// if result != nil {
					// 	diskReqInfo.IId = result[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
					// }
				}

			case 2:
				cblogger.Infof("[%s] MyImage 생성 테스트", myImageReqInfo.IId.NameId)
				//vNetworkReqInfo := irs.VNetworkReqInfo{}
				result, err := handler.SnapshotVM(myImageReqInfo)
				if err != nil {
					cblogger.Infof(myImageReqInfo.IId.NameId, " MyImage 생성 실패 : ", err)
				} else {
					cblogger.Infof("MyImage 생성 결과 : ", result)
					myImageReqInfo.IId = result.IId // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] MyImage 조회 테스트", myImageReqInfo.IId.NameId)
				result, err := handler.GetMyImage(myImageReqInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] MyImage 조회 실패 : ", myImageReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] MyImage 조회 결과 : [%s]", myImageReqInfo.IId.NameId, result)
					spew.Dump(result)
				}
			case 4:
				cblogger.Infof("[%s] MyImage 삭제 테스트", myImageReqInfo.IId.NameId)
				result, err := handler.DeleteMyImage(myImageReqInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] MyImage 삭제 실패 : ", myImageReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] MyImage 삭제 결과 : [%s]", myImageReqInfo.IId.NameId, result)
				}
			}
		}
	}
}

func main() {
	cblogger.Info("Tencent Cloud Resource Test")
	//handleVPC() //VPC
	//handleNLB()
	//handleVMSpec()
	//handleSecurity()
	//handleImage() //AMI
	//handleKeyPair()
	//handleVM()
	//handleDisk()
	handleMyImage()
	//handlePublicIP() // PublicIP 생성 후 conf
	//handleVNic() //Lancard
}
