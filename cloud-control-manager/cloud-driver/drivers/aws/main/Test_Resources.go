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
	fmt.Println("init start")
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

	securityName := "CB-SecurityTagTest"
	securityId := "sg-0d6a2bb960481ce68"
	vpcId := "vpc-0a115f43d4fcbab36" //New-CB-VPC

	for {
		fmt.Println("Security Management")
		fmt.Println("0. Quit")
		fmt.Println("1. Security List")
		fmt.Println("2. Security Create")
		fmt.Println("3. Security Get")
		fmt.Println("4. Security Delete")
		fmt.Println("5. Security Add Rules")
		fmt.Println("6. Security Delete Rules")
		fmt.Println("9. ListIID")

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return

			case 9:
				result, err := handler.ListIID()
				if err != nil {
					cblogger.Info(" ListIID Lookup Failed : ", err)
				} else {
					cblogger.Info(" ListIID Lookup Result")
					//cblogger.Info(result)
					spew.Dump(result)
				}

			case 1:
				result, err := handler.ListSecurity()
				if err != nil {
					cblogger.Info(" Security List Lookup Failed : ", err)
				} else {
					cblogger.Info("Security List Lookup Result")
					cblogger.Info(result)

					if result != nil {
						securityId = result[0].IId.SystemId // 조회 및 삭제를 위해 생성된 ID로 변경
					}
				}

			case 2:
				cblogger.Infof("[%s] Security Create Test", securityName)

				securityReqInfo := irs.SecurityReqInfo{
					IId:    irs.IID{NameId: securityName},
					VpcIID: irs.IID{SystemId: vpcId},
					//TagList: []irs.KeyValue{{Key: "Name1", Value: "Tag Name Value1"}, {Key: "Name2", Value: "Tag Name Value2"}, {Key: "Name", Value: securityName+"123"}},
					TagList: []irs.KeyValue{{Key: "Name1", Value: "Tag Name Value1"}, {Key: "Name2", Value: "Tag Name Value2"}},
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
					cblogger.Infof(securityName, " Security Create Failed : ", err)
				} else {
					cblogger.Infof("[%s] Security Create Result : [%v]", securityName, result)
					securityId = result.IId.SystemId
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] Security Lookup Test", securityId)
				result, err := handler.GetSecurity(irs.IID{SystemId: securityId})
				if err != nil {
					cblogger.Infof(securityId, " Security Lookup Failed : ", err)
				} else {
					cblogger.Infof("[%s] Security Lookup Result : [%v]", securityId, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] Security Delete Test", securityId)
				result, err := handler.DeleteSecurity(irs.IID{SystemId: securityId})
				if err != nil {
					cblogger.Infof(securityId, " Security Delete Failed : ", err)
				} else {
					cblogger.Infof("[%s] Security Delete Result : [%t]", securityId, result)
				}

			case 5:
				cblogger.Infof("[%s] Security Group Rule - Add Test", securityId)
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
					cblogger.Infof(securityId, " Security Group Rule - Add Failed : ", err)
				} else {
					cblogger.Infof("[%s] Security Group Rule - Add Result : [%s]", securityId, result)
				}

			case 6:
				cblogger.Infof("[%s] Security Group Rule - Delete Test", securityId)
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
					cblogger.Infof(securityId, " Security Group Rule - Delete Failed : ", err)
				} else {
					cblogger.Infof("[%s] Security Group Rule - Delete Result : [%s]", securityId, result)
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
		cblogger.Infof("Security Group Lookup Failed : ", err)
	} else {
		cblogger.Info("Security Group Lookup Result")
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
					cblogger.Error("PublicIP List Lookup Failed : ", err)
				} else {
					cblogger.Info("PublicIP List Lookup Result")
					spew.Dump(result)
				}

				fmt.Println("Finish ListPublicIP()")

			case 2:
				fmt.Println("Start GetPublicIP() ...")
				result, err := handler.GetPublicIP(reqPublicIP)
				if err != nil {
					cblogger.Error(reqPublicIP, " PublicIP 정보 Lookup Failed : ", err)
				} else {
					cblogger.Infof("PublicIP[%s]  정보 Lookup Result", reqPublicIP)
					spew.Dump(result)
				}
				fmt.Println("Finish GetPublicIP()")

			case 3:
				fmt.Println("Start CreatePublicIP() ...")
				reqInfo := irs.PublicIPReqInfo{Name: "mcloud-barista-eip-test"}
				result, err := handler.CreatePublicIP(reqInfo)
				if err != nil {
					cblogger.Error("PublicIP Create Failed : ", err)
				} else {
					cblogger.Info("PublicIP 생성 Success ", result)
					spew.Dump(result)
				}
				fmt.Println("Finish CreatePublicIP()")

			case 4:
				fmt.Println("Start DeletePublicIP() ...")
				result, err := handler.DeletePublicIP(reqPublicIP)
				if err != nil {
					cblogger.Error(reqDelIP, " PublicIP Delete Failed : ", err)
				} else {
					if result {
						cblogger.Infof("PublicIP[%s] 삭제 완료", reqDelIP)
					} else {
						cblogger.Errorf("PublicIP[%s] Delete Failed", reqDelIP)
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

	keyPairName := "CB-KeyPairTagTest"
	//keyPairName := config.Aws.KeyName

	for {
		fmt.Println("KeyPair Management")
		fmt.Println("0. Quit")
		fmt.Println("1. KeyPair List")
		fmt.Println("2. KeyPair Create")
		fmt.Println("3. KeyPair Get")
		fmt.Println("4. KeyPair Delete")
		fmt.Println("9. ListIID")

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return

			case 9:
				result, err := KeyPairHandler.ListIID()
				if err != nil {
					cblogger.Info(" ListIID Lookup Failed : ", err)
				} else {
					cblogger.Info(" ListIID Lookup Result")
					//cblogger.Info(result)
					spew.Dump(result)
				}

			case 1:
				result, err := KeyPairHandler.ListKey()
				if err != nil {
					cblogger.Infof(" KeyPair List Lookup Failed : ", err)
				} else {
					cblogger.Info("KeyPair List Lookup Result")
					//cblogger.Info(result)
					spew.Dump(result)
				}

			case 2:
				cblogger.Infof("[%s] KeyPair Create Test", keyPairName)
				keyPairReqInfo := irs.KeyPairReqInfo{
					IId: irs.IID{NameId: keyPairName},
					//Name: keyPairName,
					TagList: []irs.KeyValue{{Key: "Name1", Value: "Tag Name Value1"}, {Key: "Name2", Value: "Tag Name Value2"}, {Key: "Name", Value: keyPairName + "123"}},
					//TagList: []irs.KeyValue{{Key: "Name1", Value: "Tag Name Value1"}, {Key: "Name2", Value: "Tag Name Value2"}},
				}
				result, err := KeyPairHandler.CreateKey(keyPairReqInfo)
				if err != nil {
					cblogger.Infof(keyPairName, " KeyPair Create Failed : ", err)
				} else {
					cblogger.Infof("[%s] KeyPair Create Result : [%s]", keyPairName, result)
					spew.Dump(result)
				}
			case 3:
				cblogger.Infof("[%s] KeyPair Lookup Test", keyPairName)
				result, err := KeyPairHandler.GetKey(irs.IID{SystemId: keyPairName})
				if err != nil {
					cblogger.Infof(keyPairName, " KeyPair Lookup Failed : ", err)
				} else {
					cblogger.Infof("[%s] KeyPair Lookup Result : [%s]", keyPairName, result)
					spew.Dump(result)
				}
			case 4:
				cblogger.Infof("[%s] KeyPair Delete Test", keyPairName)
				result, err := KeyPairHandler.DeleteKey(irs.IID{SystemId: keyPairName})
				if err != nil {
					cblogger.Infof(keyPairName, " KeyPair Delete Failed : ", err)
				} else {
					cblogger.Infof("[%s] KeyPair Delete Result : [%s]", keyPairName, result)
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
					cblogger.Infof(" VNetwork List Lookup Failed : ", err)
				} else {
					cblogger.Info("VNetwork List Lookup Result")
					//cblogger.Info(result)
					spew.Dump(result)

					// 내부적으로 1개만 존재함.
					//조회및 Delete Test를 위해 리스트의 첫번째 서브넷 ID를 요청ID로 자동 갱신함.
					if result != nil {
						reqSubnetId = result[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
					}
				}

			case 2:
				cblogger.Infof("[%s] VNetwork Create Test", vNetworkReqInfo.IId.NameId)
				//vNetworkReqInfo := irs.VNetworkReqInfo{}
				result, err := VPCHandler.CreateVNetwork(vNetworkReqInfo)
				if err != nil {
					cblogger.Infof(reqSubnetId.NameId, " VNetwork Create Failed : ", err)
				} else {
					cblogger.Infof("VNetwork Create Result : ", result)
					reqSubnetId = result.IId // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] VNetwork Lookup Test", reqSubnetId)
				result, err := VPCHandler.GetVNetwork(reqSubnetId)
				if err != nil {
					cblogger.Infof("[%s] VNetwork Lookup Failed : ", reqSubnetId, err)
				} else {
					cblogger.Infof("[%s] VNetwork Lookup Result : [%s]", reqSubnetId, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] VNetwork Delete Test", reqSubnetId)
				result, err := VPCHandler.DeleteVNetwork(reqSubnetId)
				if err != nil {
					cblogger.Infof("[%s] VNetwork Delete Failed : ", reqSubnetId, err)
				} else {
					cblogger.Infof("[%s] VNetwork Delete Result : [%s]", reqSubnetId, result)
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
		//TagList: []irs.KeyValue{{Key: "Name1", Value: "Subnet Name Value1"}, {Key: "Name2", Value: "Subnet Name Value2"}, {Key: "Name", Value: "AddTest-Subnet123"}},
		TagList: []irs.KeyValue{{Key: "Name1", Value: "Subnet Name Value1"}, {Key: "Name2", Value: "Subnet Name Value2"}},
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
				TagList:   []irs.KeyValue{{Key: "Name1", Value: "Subnet Name Value1"}, {Key: "Name2", Value: "Subnet Name Value2"}, {Key: "Name", Value: "AddTest-Subnet123"}},
				//TagList: []irs.KeyValue{{Key: "Name1", Value: "Subnet Name Value1"}, {Key: "Name2", Value: "Subnet Name Value2"}},
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
		TagList: []irs.KeyValue{{Key: "Name1", Value: "Subnet Name Value1"}, {Key: "Name2", Value: "Subnet Name Value2"}, {Key: "Name", Value: "New-CB-VPC123"}},
		//TagList: []irs.KeyValue{{Key: "Name1", Value: "VPC Name Value1"}, {Key: "Name2", Value: "VPC Name Value2"}},
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
		fmt.Println("9. ListIID")

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return

			case 9:
				result, err := VPCHandler.ListIID()
				if err != nil {
					cblogger.Infof(" VNetwork ListIID Lookup Failed : ", err)
				} else {
					cblogger.Info("VNetwork ListIID Lookup Result")
					//cblogger.Info(result)
					spew.Dump(result)
				}

			case 1:
				result, err := VPCHandler.ListVPC()
				if err != nil {
					cblogger.Infof(" VNetwork List Lookup Failed : ", err)
				} else {
					cblogger.Info("VNetwork List Lookup Result")
					//cblogger.Info(result)
					spew.Dump(result)

					// 내부적으로 1개만 존재함.
					//조회및 Delete Test를 위해 리스트의 첫번째 서브넷 ID를 요청ID로 자동 갱신함.
					if result != nil {
						reqSubnetId = result[0].IId    // 조회 및 삭제를 위해 생성된 ID로 변경
						subnetReqVpcInfo = reqSubnetId //Subnet 추가/Delete Test용
					}
				}

			case 2:
				cblogger.Infof("[%s] VNetwork Create Test", vpcReqInfo.IId.NameId)
				//vpcReqInfo := irs.VPCReqInfo{}
				result, err := VPCHandler.CreateVPC(vpcReqInfo)
				if err != nil {
					cblogger.Infof(reqSubnetId.NameId, " VNetwork Create Failed : ", err)
				} else {
					cblogger.Infof("VNetwork Create Result : ", result)
					reqSubnetId = result.IId // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] VNetwork Lookup Test", reqSubnetId)
				result, err := VPCHandler.GetVPC(reqSubnetId)
				if err != nil {
					cblogger.Infof("[%s] VNetwork Lookup Failed : ", reqSubnetId, err)
				} else {
					cblogger.Infof("[%s] VNetwork Lookup Result : [%s]", reqSubnetId, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] VNetwork Delete Test", reqSubnetId)
				result, err := VPCHandler.DeleteVPC(reqSubnetId)
				if err != nil {
					cblogger.Infof("[%s] VNetwork Delete Failed : ", reqSubnetId, err)
				} else {
					cblogger.Infof("[%s] VNetwork Delete Result : [%s]", reqSubnetId, result)
				}

			case 5:
				cblogger.Infof("[%s] Subnet Add Test", vpcReqInfo.IId.NameId)
				result, err := VPCHandler.AddSubnet(subnetReqVpcInfo, subnetReqInfo)
				if err != nil {
					cblogger.Infof(reqSubnetId.NameId, " VNetwork Create Failed : ", err)
				} else {
					cblogger.Infof("VNetwork Create Result : ", result)
					//reqSubnetId = result.IId // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 6:
				cblogger.Infof("[%s] Subnet Delete Test", reqSubnetId.SystemId)
				result, err := VPCHandler.RemoveSubnet(subnetReqVpcInfo, reqSubnetId)
				if err != nil {
					cblogger.Infof("[%s] Subnet Delete Failed : ", reqSubnetId.SystemId, err)
				} else {
					cblogger.Infof("[%s] Subnet Delete Result : [%s]", reqSubnetId.SystemId, result)
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
					cblogger.Infof(" Image List Lookup Failed : ", err)
				} else {
					cblogger.Info("Image List Lookup Result")
					cblogger.Debug(result)
					cblogger.Infof("Log Level : [%s]", cblog.GetLevel())
					//spew.Dump(result)
					cblogger.Info("Number of output results : ", len(result))

					//조회및 Delete Test를 위해 리스트의 첫번째 정보의 ID를 요청ID로 자동 갱신함.
					if result != nil {
						imageReqInfo.IId = result[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
					}
				}

			case 2:
				cblogger.Infof("[%s] Image Create Test", imageReqInfo.IId.NameId)
				result, err := handler.CreateImage(imageReqInfo)
				if err != nil {
					cblogger.Infof(imageReqInfo.IId.NameId, " Image Create Failed : ", err)
				} else {
					cblogger.Infof("Image Create Result : ", result)
					imageReqInfo.IId = result.IId // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] Image Lookup Test", imageReqInfo.IId)
				result, err := handler.GetImage(imageReqInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] Image Lookup Failed : ", imageReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] Image Lookup Result : [%s]", imageReqInfo.IId.NameId, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] Image Delete Test", imageReqInfo.IId.NameId)
				result, err := handler.DeleteImage(imageReqInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] Image Delete Failed : ", imageReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] Image Delete Result : [%s]", imageReqInfo.IId.NameId, result)
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
					cblogger.Infof(" VNic List Lookup Failed : ", err)
				} else {
					cblogger.Info("VNic List Lookup Result")
					spew.Dump(result)
					if len(result) > 0 {
						reqVnicID = result[0].Id // 조회 및 삭제 편의를 위해 목록의 첫번째 ID로 변경
					}
				}

			case 2:
				cblogger.Infof("[%s] VNic Create Test", vNicReqInfo.Name)
				result, err := handler.CreateVNic(vNicReqInfo)
				if err != nil {
					cblogger.Infof(reqVnicID, " VNic Create Failed : ", err)
				} else {
					cblogger.Infof("VNic Create Result : ", result)
					reqVnicID = result.Id // 조회 및 삭제를 위해 생성된 ID로 변경
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] VNic Lookup Test", reqVnicID)
				result, err := handler.GetVNic(reqVnicID)
				if err != nil {
					cblogger.Infof("[%s] VNic Lookup Failed : ", reqVnicID, err)
				} else {
					cblogger.Infof("[%s] VNic Lookup Result : [%s]", reqVnicID, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] VNic Delete Test", reqVnicID)
				result, err := handler.DeleteVNic(reqVnicID)
				if err != nil {
					cblogger.Infof("[%s] VNic Delete Failed : ", reqVnicID, err)
				} else {
					cblogger.Infof("[%s] VNic Delete Result : [%s]", reqVnicID, result)
				}
			}
		}
	}
}
*/

func testErr() error {
	//return awserr.Error("")
	//return errors.New("")
	return awserr.New("504", "not found", nil)
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
	// VmID := irs.IID{SystemId: "i-0cea86282a9e2a569"}
	VmID := irs.IID{SystemId: "i-02ac1c4ff1d40815c"}

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
		fmt.Println("10. ListIID")

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return

			case 10:
				result, err := vmHandler.ListIID()
				if err != nil {
					cblogger.Info(" ListIID Lookup Failed : ", err)
				} else {
					cblogger.Info(" ListIID Lookup Result")
					//cblogger.Info(result)
					spew.Dump(result)
				}

			case 1:
				vmReqInfo := irs.VMReqInfo{
					IId:     irs.IID{NameId: "mcloud-barista-tag-test"},
					TagList: []irs.KeyValue{{Key: "Name1", Value: "Tag Name Value1"}, {Key: "Name2", Value: "Tag Name Value2"}, {Key: "Name", Value: "mcloud-barista-tag-test123"}},
					//TagList: []irs.KeyValue{{Key: "Name1", Value: "Tag Name Value1"}, {Key: "Name2", Value: "Tag Name Value2"}},
					//ImageIID:          irs.IID{SystemId: "ami-001b6f8703b50e077"}, //centos-stable-7.2003.13-ebs-202005201235
					ImageIID: irs.IID{SystemId: "ami-056a29f2eddc40520"}, //Ubuntu Server 22.04 LTS (HVM), SSD Volume Type
					//ImageIID:          irs.IID{SystemId: "ami-09e67e426f25ce0d7"}, //Ubuntu Server 20.04 LTS (HVM) - 버지니아 북부 리전
					//ImageIID:          irs.IID{SystemId: "ami-059b6d3840b03d6dd"}, //Ubuntu Server 20.04 LTS (HVM)
					//ImageIID: irs.IID{SystemId: "ami-0fe22bffdec36361c"}, //Ubuntu Server 18.04 LTS (HVM) - Japan 리전
					//ImageIID:          irs.IID{SystemId: "ami-093f427eb324bb754"}, //Microsoft Windows Server 2012 R2 RTM 64-bit Locale English AMI provided by Amazon - Japan 리전
					SubnetIID:         irs.IID{SystemId: "subnet-02127b9d8c84f7440"},
					SecurityGroupIIDs: []irs.IID{{SystemId: "sg-0209d9dc23ebd4cdd"}}, //3389 RDP 포트 Open
					VMSpecName:        "t2.micro",
					KeyPairIID:        irs.IID{SystemId: "CB-KeyPairTagTest"},
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
					cblogger.Info("VM Created!!", vmInfo)
					spew.Dump(vmInfo)
					VmID = vmInfo.IId
				}
				//cblogger.Info(vm)

				cblogger.Info("Finish Create VM")

			case 2:
				vmInfo, err := vmHandler.GetVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] VM Info Lookup Failed", VmID)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] VM Info Lookup Result", VmID)
					cblogger.Info(vmInfo)
					spew.Dump(vmInfo)
				}

			case 3:
				cblogger.Info("Start Suspend VM ...")
				result, err := vmHandler.SuspendVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] VM Suspend Fail - [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] VM Suspend Success - [%s]", VmID, result)
				}

			case 4:
				cblogger.Info("Start Resume  VM ...")
				result, err := vmHandler.ResumeVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] VM Resume Fail - [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] VM Resume Success - [%s]", VmID, result)
				}

			case 5:
				cblogger.Info("Start Reboot  VM ...")
				result, err := vmHandler.RebootVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] VM Reboot Fail - [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] VM Reboot Success - [%s]", VmID, result)
				}

			case 6:
				cblogger.Info("Start Terminate  VM ...")
				result, err := vmHandler.TerminateVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] VM Terminate Fail - [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] VM Terminate Success - [%s]", VmID, result)
				}

			case 7:
				cblogger.Info("Start Get VM Status...")
				vmStatus, err := vmHandler.GetVMStatus(VmID)
				if err != nil {
					cblogger.Errorf("[%s] VM Get Status Fail", VmID)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] VM Get Status Success : [%s]", VmID, vmStatus)
				}

			case 8:
				cblogger.Info("Start ListVMStatus ...")
				vmStatusInfos, err := vmHandler.ListVMStatus()
				if err != nil {
					cblogger.Error("ListVMStatus Fail")
					cblogger.Error(err)
				} else {
					cblogger.Info("ListVMStatus Success")
					cblogger.Info(vmStatusInfos)
					spew.Dump(vmStatusInfos)
				}

			case 9:
				cblogger.Info("Start ListVM ...")
				vmList, err := vmHandler.ListVM()
				if err != nil {
					cblogger.Error("ListVM Fail")
					cblogger.Error(err)
				} else {
					cblogger.Info("ListVM Success")
					cblogger.Info("=========== VM List ================")
					cblogger.Info(vmList)
					spew.Dump(vmList)
					cblogger.Infof("=========== VM List count : [%d] ================", len(vmList))
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
					cblogger.Error("VMSpec List Lookup Failed : ", err)
				} else {
					cblogger.Debug("VMSpec List Lookup Result")
					//spew.Dump(result)
					cblogger.Debug(result)
					cblogger.Infof("Total number of lists : [%d]", len(result))
				}

				fmt.Println("Finish ListVMSpec()")

			case 2:
				fmt.Println("Start GetVMSpec() ...")
				result, err := handler.GetVMSpec(reqVMSpec)
				if err != nil {
					cblogger.Error(reqVMSpec, " VMSpec Info Lookup Failed : ", err)
				} else {
					cblogger.Debugf("VMSpec[%s] Info Lookup Result", reqVMSpec)
					cblogger.Debug(result)
				}
				fmt.Println("Finish GetVMSpec()")

			case 3:
				fmt.Println("Start ListOrgVMSpec() ...")
				result, err := handler.ListOrgVMSpec()
				if err != nil {
					cblogger.Error("VMSpec Org List Lookup Failed : ", err)
				} else {
					cblogger.Debug("VMSpec Org List Lookup Result")
					//spew.Dump(result)
					cblogger.Debug(result)
					//spew.Dump(result)
					//fmt.Println(result)
					//fmt.Println("=========================")
					//fmt.Println(result)
					cblogger.Infof("Total number of lists : [%d]", len(result))
				}

				fmt.Println("Finish ListOrgVMSpec()")

			case 4:
				fmt.Println("Start GetOrgVMSpec() ...")
				result, err := handler.GetOrgVMSpec(reqVMSpec)
				if err != nil {
					cblogger.Error(reqVMSpec, " VMSpec Org Info Lookup Failed : ", err)
				} else {
					cblogger.Debugf("VMSpec[%s] Org Info Lookup Result", reqVMSpec)
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
		VpcIID: irs.IID{SystemId: "vpc-0a115f43d4fcbab36"},
		Type:   "PUBLIC",
		Scope:  "REGION",

		TagList: []irs.KeyValue{{Key: "Name1", Value: "Tag Name Value1"}, {Key: "Name2", Value: "Tag Name Value2"}, {Key: "Name", Value: "cb-nlb-test01123"}},
		//TagList: []irs.KeyValue{{Key: "Name1", Value: "Tag Name Value1"}, {Key: "Name2", Value: "Tag Name Value2"}},

		Listener: irs.ListenerInfo{
			Protocol: "TCP", // AWS NLB : TCP, TLS, UDP, or TCP_UDP
			//IP: "",
			Port: "22",
		},

		VMGroup: irs.VMGroupInfo{
			Protocol: "TCP", //TCP|UDP|HTTP|HTTPS
			Port:     "22",  //1-65535
			VMs:      &[]irs.IID{irs.IID{SystemId: "i-0c65033e158e0fd99"}},
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
		fmt.Println("11. ListIID")

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return

			case 11:
				result, err := handler.ListIID()
				if err != nil {
					cblogger.Info(" ListIID Lookup Failed : ", err)
				} else {
					cblogger.Info(" ListIID Lookup Result")
					//cblogger.Info(result)
					spew.Dump(result)
				}

			case 1:
				result, err := handler.ListNLB()
				if err != nil {
					cblogger.Infof(" NLB List Lookup Failed : ", err)
				} else {
					cblogger.Info("NLB List Lookup Result")
					cblogger.Debug(result)
					cblogger.Infof("Log Level : [%s]", cblog.GetLevel())
					//spew.Dump(result)
					cblogger.Info("Number of output results : ", len(result))

					//조회및 Delete Test를 위해 리스트의 첫번째 정보의 ID를 요청ID로 자동 갱신함.
					if result != nil {
						nlbReqInfo.IId = result[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
					}
				}

			case 2:
				cblogger.Infof("[%s] NLB Create Test", nlbReqInfo.IId.NameId)
				result, err := handler.CreateNLB(nlbReqInfo)
				if err != nil {
					cblogger.Infof(nlbReqInfo.IId.NameId, " NLB Create Failed : ", err)
				} else {
					cblogger.Infof("NLB Create Success : ", result)
					nlbReqInfo.IId = result.IId // 조회 및 삭제를 위해 생성된 ID로 변경
					if cblogger.Level.String() == "debug" {
						spew.Dump(result)
					}
				}

			case 3:
				cblogger.Infof("[%s] NLB Lookup Test", nlbReqInfo.IId)
				result, err := handler.GetNLB(nlbReqInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] NLB Lookup Failed : ", nlbReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] NLB Lookup Success : [%s]", nlbReqInfo.IId.NameId, result)
					if cblogger.Level.String() == "debug" {
						spew.Dump(result)
					}
				}

			case 4:
				cblogger.Infof("[%s] NLB Delete Test", nlbReqInfo.IId.NameId)
				result, err := handler.DeleteNLB(nlbReqInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] NLB Delete Failed : ", nlbReqInfo.IId.NameId, err)
				} else {
					cblogger.Info("Success")
					cblogger.Infof("[%s] NLB Delete Success : [%s]", nlbReqInfo.IId.NameId, result)
				}

			case 5:
				cblogger.Infof("[%s] Change listener Test", nlbReqInfo.IId)
				reqListenerInfo := irs.ListenerInfo{
					Protocol: "TCP", // AWS NLB : TCP, TLS, UDP, or TCP_UDP
					//IP: "",
					Port: "80",
				}
				result, err := handler.ChangeListener(nlbReqInfo.IId, reqListenerInfo)
				if err != nil {
					cblogger.Infof("[%s] Change listener Fail : ", nlbReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] Change listener Success : [%s]", nlbReqInfo.IId.NameId, result)
					if cblogger.Level.String() == "debug" {
						spew.Dump(result)
					}
				}

			case 7:
				cblogger.Infof("[%s] AddVMs Test", nlbReqInfo.IId.NameId)
				cblogger.Info(reqAddVMs)
				result, err := handler.AddVMs(nlbReqInfo.IId, reqAddVMs)
				if err != nil {
					cblogger.Infof("[%s] AddVMs Fail : ", nlbReqInfo.IId.NameId, err)
				} else {
					cblogger.Info("Success")
					cblogger.Infof("[%s] AddVMs Success : [%s]", nlbReqInfo.IId.NameId, result)
				}

			case 8:
				cblogger.Infof("[%s] RemoveVMs Test", nlbReqInfo.IId.NameId)
				cblogger.Info(reqRemoveVMs)
				result, err := handler.RemoveVMs(nlbReqInfo.IId, reqRemoveVMs)
				if err != nil {
					cblogger.Infof("[%s] RemoveVMs Fail : ", nlbReqInfo.IId.NameId, err)
				} else {
					cblogger.Info("Success")
					cblogger.Infof("[%s] RemoveVMs Success : [%s]", nlbReqInfo.IId.NameId, result)
				}

			case 9:
				cblogger.Infof("[%s] GetVMGroupHealthInfo Test", nlbReqInfo.IId)
				result, err := handler.GetVMGroupHealthInfo(nlbReqInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] GetVMGroupHealthInfo Fail : ", nlbReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] GetVMGroupHealthInfo Success : [%s]", nlbReqInfo.IId.NameId, result)
					if cblogger.Level.String() == "debug" {
						spew.Dump(result)
					}
				}

			case 6:
				cblogger.Infof("[%s] NLB VM Group Change Test", nlbReqInfo.IId.NameId)
				result, err := handler.ChangeVMGroupInfo(nlbReqInfo.IId, irs.VMGroupInfo{
					Protocol: "TCP",
					Port:     "8080",
				})
				if err != nil {
					cblogger.Infof("[%s] NLB VM Group Change Fail : ", nlbReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] NLB VM Group Change Success : [%s]", nlbReqInfo.IId.NameId, result)
					if cblogger.Level.String() == "debug" {
						spew.Dump(result)
					}
				}
			case 10:
				cblogger.Infof("[%s] NLB Health Checker Change Test", nlbReqInfo.IId.NameId)
				result, err := handler.ChangeHealthCheckerInfo(nlbReqInfo.IId, irs.HealthCheckerInfo{
					Protocol: "TCP",
					Port:     "22",
					//Interval: 10, //미지원
					//Timeout:   3,	//미지원
					Threshold: 5,
				})
				if err != nil {
					cblogger.Infof("[%s] NLB Health Checker Change Fail : ", nlbReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] NLB Health Checker Change Success : [%s]", nlbReqInfo.IId.NameId, result)
					if cblogger.Level.String() == "debug" {
						spew.Dump(result)
					}
				}
			}
		}
	}
}

func handleCluster() {
	cblogger.Debug("Start PMKS Handler Resource Test")

	ResourceHandler, err := getResourceHandler("PMKS")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.ClusterHandler)

	if handler == nil {
		fmt.Println("handler nil")
		panic(err)
	}

	subnets := []irs.IID{}
	// subnets = append(subnets, irs.IID{SystemId: "subnet-02127b9d8c84f7440"}) //2a
	// subnets = append(subnets, irs.IID{SystemId: "subnet-0c2b7e03a5f397e25"}) //2c

	subnets = append(subnets, irs.IID{SystemId: "subnet-0d5357fe8ea45219c"}) //2a
	subnets = append(subnets, irs.IID{SystemId: "subnet-05fb1d006711bc103"}) //2c

	//vpc-0c4d36a3ac3924419
	clusterReqInfo := irs.ClusterInfo{
		//TagList: []irs.KeyValue{{Key: "Name1", Value: "Tag Name Value1"}, {Key: "Name2", Value: "Tag Name Value2"}, {Key: "Name", Value: securityName+"123"}},
		TagList: []irs.KeyValue{{Key: "Name1", Value: "Tag Name Value1"}, {Key: "Name2", Value: "Tag Name Value2"}},
		IId:     irs.IID{NameId: "cb-eks-cluster-tag-test", SystemId: "cb-eks-cluster-tag-test"},
		Version: "1.23.3", //K8s version
		Network: irs.NetworkInfo{
			VpcIID: irs.IID{SystemId: "vpc-002cb8cb8b00b0769"},
			//SubnetIID: [irs.IID{SystemId: "subnet-262d6d7a"},irs.IID{SystemId: "vpc-c0479cab"}],
			SubnetIIDs: subnets,
		},
	} // nlbReqInfo

	reqNodeGroupInfo := irs.NodeGroupInfo{
		IId:         irs.IID{NameId: "cb-eks-node-tag-test"},
		MinNodeSize: 1,
		MaxNodeSize: 2,

		// VM config.
		ImageIID:     irs.IID{SystemId: "ami-056a29f2eddc40520"}, // Amazon Linux 2 (AL2_x86_64) - https://docs.aws.amazon.com/eks/latest/userguide/eks-optimized-ami.html
		VMSpecName:   "t3.medium",
		RootDiskType: "SSD(gp2)", // "SSD(gp2)", "Premium SSD", ...
		RootDiskSize: "20",       // "", "default", "50", "1000" (GB)
		// KeyPairIID:   irs.IID{SystemId: "CB-KeyPairTagTest"},
		KeyPairIID: irs.IID{SystemId: "aws-ap-northeast-2-ap-northeast-2a-keypair-be3-test-cvejdsqhucphg4rlj3gg"},
		//Status NodeGroupStatus

		// Scaling config.
		OnAutoScaling:   true, // default: true
		DesiredNodeSize: 1,
	}

	for {
		fmt.Println("ClusterHandler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. Cluster List")
		fmt.Println("2. Cluster Create")
		fmt.Println("3. Cluster Get")
		fmt.Println("4. Cluster Delete")

		fmt.Println("5. ListNodeGroup")
		fmt.Println("6. AddNodeGroup")
		fmt.Println("7. RemoveNodeGroup")

		fmt.Println("8. UpgradeCluster")
		fmt.Println("9. ChangeNodeGroupScaling")
		fmt.Println("10. ListIID")

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return

			case 10:
				result, err := handler.ListIID()
				if err != nil {
					cblogger.Info(" ListIID Lookup Failed : ", err)
				} else {
					cblogger.Info(" ListIID Lookup Result")
					//cblogger.Info(result)
					spew.Dump(result)
				}

			case 1:
				result, err := handler.ListCluster()
				if err != nil {
					cblogger.Info(" Cluster List Lookup Failed : ", err)
				} else {
					cblogger.Info("Cluster List Lookup Result")
					cblogger.Debug(result)
					cblogger.Infof("Log Level : [%s]", cblog.GetLevel())
					cblogger.Info("Number of output results : ", len(result))

					if cblogger.Level.String() == "debug" {
						cblogger.Info("========= DEBUG START ==========")
						spew.Dump(result)
						cblogger.Info("========= DEBUG END ==========")
					}
					//조회및 삭제 테스트를 위해 리스트의 첫번째 정보의 ID를 요청ID로 자동 갱신함.
					if len(result) > 0 {
						clusterReqInfo.IId = result[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
						cblogger.Info("---> Req IID Change : ", clusterReqInfo.IId)
					}
				}
			case 2:
				cblogger.Infof("[%s] Cluster Create Test", clusterReqInfo.IId.NameId)
				result, err := handler.CreateCluster(clusterReqInfo)
				if err != nil {
					cblogger.Infof(clusterReqInfo.IId.NameId, " Cluster Create Fail : ", err)
				} else {
					cblogger.Info("Cluster Create Success : ", result)
					clusterReqInfo.IId = result.IId // 조회 및 삭제를 위해 생성된 ID로 변경
					if cblogger.Level.String() == "debug" {
						cblogger.Info("========= DEBUG START ==========")
						spew.Dump(result)
						cblogger.Info("========= DEBUG END ==========")
					}
				}

				//eks-cb-eks-node-test02a-aws-9cc2876a-d3cb-2c25-55a8-9a19c431e716

			case 3:
				cblogger.Infof("[%s] Cluster Get Test", clusterReqInfo.IId)
				result, err := handler.GetCluster(clusterReqInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] Cluster Get Fail : ", clusterReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] Cluster Get Success : [%s]", clusterReqInfo.IId.NameId, result)
					if cblogger.Level.String() == "debug" {
						cblogger.Info("========= DEBUG START ==========")
						spew.Dump(result)
						cblogger.Info("========= DEBUG END ==========")
					}
				}

			case 4:
				cblogger.Infof("[%s] Cluster Delete Test", clusterReqInfo.IId.NameId)
				result, err := handler.DeleteCluster(clusterReqInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] Cluster Delete Fail : ", clusterReqInfo.IId.NameId, err)
				} else {
					cblogger.Info("Success")
					cblogger.Infof("[%s] Cluster Delete Success : [%s]", clusterReqInfo.IId.NameId, result)
				}

			/*
				case 5:
					cblogger.Infof("[%s] ListNodeGroup Test", clusterReqInfo.IId)
					result, err := handler.ListNodeGroup(clusterReqInfo.IId)
					if err != nil {
						cblogger.Infof("[%s] ListNodeGroup Fail : ", clusterReqInfo.IId.NameId, err)
					} else {
						cblogger.Infof("[%s] ListNodeGroup Success : [%s]", clusterReqInfo.IId.NameId, result)
						if cblogger.Level.String() == "debug" {
							spew.Dump(result)
						}

						cblogger.Info("Number of output results : ", len(result))

						//조회및 삭제 Test를 위해 리스트의 첫번째 정보의 ID를 요청ID로 자동 갱신함.
						if len(result) > 0 {
							reqNodeGroupInfo.IId = result[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
							cblogger.Info("---> Req IID 변경 : ", reqNodeGroupInfo.IId)
						}
					}
			*/

			case 6:
				cblogger.Infof("[%s] AddNodeGroup Test", clusterReqInfo.IId)
				result, err := handler.AddNodeGroup(clusterReqInfo.IId, reqNodeGroupInfo)
				if err != nil {
					cblogger.Infof("[%s] AddNodeGroup Fail : ", clusterReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] AddNodeGroup Success : [%s]", clusterReqInfo.IId.NameId, result)
					if cblogger.Level.String() == "debug" {
						cblogger.Info("========= DEBUG START ==========")
						spew.Dump(result)
						cblogger.Info("========= DEBUG END ==========")
					}
				}

			case 7:
				cblogger.Infof("[%s] RemoveNodeGroup Test", clusterReqInfo.IId)
				result, err := handler.RemoveNodeGroup(clusterReqInfo.IId, reqNodeGroupInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] RemoveNodeGroup Fail : ", reqNodeGroupInfo.IId.SystemId, err)
				} else {
					cblogger.Infof("[%s] RemoveNodeGroup Success", reqNodeGroupInfo.IId.SystemId)
					if cblogger.Level.String() == "debug" {
						spew.Dump(result)
					}
				}

			case 8:
				cblogger.Infof("[%s] UpgradeCluster Test", clusterReqInfo.IId)
				result, err := handler.UpgradeCluster(clusterReqInfo.IId, "1.24")
				if err != nil {
					cblogger.Infof("[%s] UpgradeCluster Fail : ", clusterReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] UpgradeCluster Success : [%s]", clusterReqInfo.IId.NameId, result)
					if cblogger.Level.String() == "debug" {
						cblogger.Info("========= DEBUG START ==========")
						spew.Dump(result)
						cblogger.Info("========= DEBUG END ==========")
					}
				}

			case 9:
				cblogger.Infof("[%s] ChangeNodeGroupScaling Test", clusterReqInfo.IId)
				//원하는 크기 / 최소 크기 / 최대 크기
				result, err := handler.ChangeNodeGroupScaling(clusterReqInfo.IId, reqNodeGroupInfo.IId, 2, 2, 4)
				if err != nil {
					cblogger.Infof("[%s] ChangeNodeGroupScaling Fail : ", clusterReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] ChangeNodeGroupScaling Success : [%s]", clusterReqInfo.IId.NameId, result)
					if cblogger.Level.String() == "debug" {
						cblogger.Info("========= DEBUG START ==========")
						spew.Dump(result)
						cblogger.Info("========= DEBUG END ==========")
					}
				}
			}
		}
	}
}

func handleRegionZone() {
	cblogger.Debug("Start Region Zone Handler Test")

	ResourceHandler, err := getResourceHandler("RegionZone")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.RegionZoneHandler)
	if handler == nil {
		fmt.Println("handler nil")
		panic(err)
	}

	for {
		fmt.Println("RegionZoneHandler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. List RegionZone")
		fmt.Println("2. ('us-west-1') RegionZone")
		fmt.Println("3. List OrgRegion")
		fmt.Println("4. List OrgZone")
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
				result, err := handler.ListRegionZone()
				if err != nil {
					cblogger.Infof("ListRegionZone List Lookup Failed : %s", err)
				} else {
					cblogger.Info("ListRegionZone List Lookup Result")
					// cblogger.Debugf("결과 %s", result[0])
					spew.Dump(result)
					cblogger.Infof("Log Level : [%s]", cblog.GetLevel())
					//spew.Dump(result)
					cblogger.Info("Number of output results : ", len(result))
				}

			case 2:
				result, err := handler.GetRegionZone("us-west-1")
				if err != nil {
					cblogger.Infof("GetRegionZone Lookup Failed : ", err)
				} else {
					cblogger.Info("GetRegionZone Lookup Result")
					cblogger.Debug(result)
					cblogger.Infof("Log Level : [%s]", cblog.GetLevel())
					// spew.Dump(result)
				}

			case 3:
				result, err := handler.ListOrgRegion()
				if err != nil {
					cblogger.Infof("ListOrgRegion List Lookup Failed : ", err)
				} else {
					cblogger.Info("ListOrgRegion List Lookup Result")
					cblogger.Debug(result)
					cblogger.Infof("Log Level : [%s]", cblog.GetLevel())
					//spew.Dump(result)
					cblogger.Info("Number of output results : ", len(result))
				}
			case 4:
				result, err := handler.ListOrgZone()
				if err != nil {
					cblogger.Infof("ListOrgZone List Lookup Failed : %s", err)
				} else {
					cblogger.Info("ListOrgZone List Lookup Result")
					cblogger.Debug(result)
					cblogger.Infof("Log Level : [%s]", cblog.GetLevel())
					//spew.Dump(result)
					cblogger.Info("Number of output results : ", len(result))
				}
			}
		}
	}
}

func handlePriceInfo() {
	cblogger.Debug("Start Price Info Test")
	ResourceHandler, err := getResourceHandler("PriceInfo")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.PriceInfoHandler)
	if handler == nil {
		fmt.Println("handler nil")
		panic(err)
	}

	for {
		fmt.Println("PriceInfoHandler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. ListProductFamily")
		fmt.Println("2. GetPriceInfo")
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
				result, err := handler.ListProductFamily("us-west-1")
				if err != nil {
					cblogger.Infof("ListProductFamily List Lookup Failed : %s", err)
				} else {
					cblogger.Info("ListProductFamily List Lookup Result")
					// cblogger.Debugf("결과 %s", result[0])
					spew.Dump(result)
					cblogger.Infof("Log Level : [%s]", cblog.GetLevel())
					//spew.Dump(result)
					cblogger.Info("Number of output results : ", len(result))
				}

			case 2:
				var filterList []irs.KeyValue
				// Type : [TERM_CONTAIN, ANY_OF, TERM_MATCH, NONE_OF, CONTAINS, EQUALS]
				// filterList = append(filterList, irs.KeyValue{Key: "filter", Value: "{\"Field\":\"instanceType\",\"Type\":\"TERM_MATCH\",\"Value\":\"t2.nano\"}"})
				// filterList = append(filterList, irs.KeyValue{Key: "filter", Value: "{\"Field\":\"operatingSystem\",\"Type\":\"TERM_MATCH\",\"Value\":\"Linux\"}"})
				//filterList = append(filterList, irs.KeyValue{Key: "productId", Value: "22JRCBWS3QQEPYKE.MZU6U2429S.2TG2D8R56U"})

				// TEST seraach data
				// instanceType = t2.nano
				// operatingSystem = Linux

				// cli 에서 아래 명령어를 통해 attribute를 검색할 수 있음.
				// aws pricing get-attribute-values --service-code AmazonEC2 --attribute-name instanceType

				//result, err := handler.GetPriceInfo("AmazonEC2", "ap-northeast-2", filterList)

				// AmazonEC2는 ServiceCode고정 -> ProductFamily : Compute Instance로 두고 테스트
				result, err := handler.GetPriceInfo("Compute Instance", "us-west-1", filterList)

				if err != nil {
					cblogger.Infof("GetPriceInfo Lookup Failed : ", err)
				} else {
					cblogger.Info("GetPriceInfo Lookup Result")
					cblogger.Info(result)
					cblogger.Infof("Log Level : [%s]", cblog.GetLevel())
				}
			}
		}
	}
}

// Test Tag
func handleTag() {
	cblogger.Debug("Start TagHandler Resource Test")

	ResourceHandler, err := getResourceHandler("Tag")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.TagHandler)

	var reqType irs.RSType = irs.FILESYSTEM
	//reqIID := irs.IID{SystemId: "i-02ac1c4ff1d40815c"}
	reqIID := irs.IID{SystemId: "CB-KeyPairTest123123"}
	reqIID = irs.IID{SystemId: "fs-0580954bd3ccd0119"}

	reqTag := irs.KeyValue{Key: "tag3", Value: "tag3 test"}
	reqKey := "tag3"
	reqKey = "CB-KeyPairTagTest"
	reqKey = "cb-nlb-test01123"
	reqKey = "Name2"
	reqKey = "cb-eks-cluster-tag-test"
	reqKey = "tag3"
	reqType = irs.ALL
	reqType = irs.FILESYSTEM

	for {
		fmt.Println("TagHandler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. Tag List")
		fmt.Println("2. Tag Add")
		fmt.Println("3. Tag Get")
		fmt.Println("4. Tag Delete")
		fmt.Println("5. Tag Find")

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
				cblogger.Infof("Lookup request tag type : [%s]", reqType)
				if reqType == irs.VM {
					cblogger.Debug("VM Requested")
				}

				result, err := handler.ListTag(reqType, reqIID)
				if err != nil {
					cblogger.Info(" Tag List Lookup Failed : ", err)
				} else {
					cblogger.Info("Tag List Lookup Result")
					cblogger.Debug(result)
					cblogger.Infof("Log Level : [%s]", cblog.GetLevel())
					//spew.Dump(result)
					cblogger.Info("Number of output results : ", len(result))

					//조회및 삭제 테스트를 위해 리스트의 첫번째 정보의 ID를 요청ID로 자동 갱신함.
					if result != nil {
						//tagReqInfo.IId = result[0].IId // 조회 및 삭제를 위해 생성된 ID로 변경
					}
				}

			case 2:
				cblogger.Infof("[%s] Tag Add Test", reqIID.SystemId)
				result, err := handler.AddTag(reqType, reqIID, reqTag)
				if err != nil {
					cblogger.Infof(reqIID.SystemId, " Tag Create Failed : ", err)
				} else {
					cblogger.Info("Tag Create Result : ", result)
					reqKey = result.Key
					cblogger.Infof("Request Target Tag Key changed to [%s]", reqKey)
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] Tag Lookup Test - Key[%s]", reqIID.SystemId, reqKey)
				result, err := handler.GetTag(reqType, reqIID, reqKey)
				if err != nil {
					cblogger.Infof("[%s] Tag Lookup Failed : [%v]", reqKey, err)
				} else {
					cblogger.Infof("[%s] Tag Lookup Result : [%s]", reqKey, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] Tag Delete Test - Key[%s]", reqIID.SystemId, reqKey)
				result, err := handler.RemoveTag(reqType, reqIID, reqKey)
				if err != nil {
					cblogger.Infof("[%s] Tag Delete Failed : [%v]", reqKey, err)
				} else {
					cblogger.Infof("[%s] Tag Delete Result : [%v]", reqKey, result)
				}

			case 5:
				cblogger.Infof("[%s] Tag Find Test - Key[%s]", reqType, reqKey)
				result, err := handler.FindTag(reqType, reqKey)
				if err != nil {
					cblogger.Infof("[%s] Tag Find Failed : [%s]", reqKey, err)
				} else {
					spew.Dump(result)
					cblogger.Infof("[%s] Tag Find Result : [%d] count", reqKey, len(result))
				}
			}
		}
	}
}

// Test Disk
func handleDisk() {
	cblogger.Debug("Start Disk Resource Test")

	ResourceHandler, err := getResourceHandler("Disk")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.DiskHandler)

	reqIID := irs.IID{NameId: "tag-test", SystemId: "vol-0d0bdcd08f027c2a1"}
	reqKey := "tag3"
	reqInfo := irs.DiskInfo{
		IId:      reqIID,
		DiskType: "gp2",
		DiskSize: "50",

		//TagList: []irs.KeyValue{{Key: "Name1", Value: "Tag Name Value1"}, {Key: "Name2", Value: "Tag Name Value2"}, {Key: "Name", Value: securityName+"123"}},
		TagList: []irs.KeyValue{{Key: "Name1", Value: "Tag Name Value1"}, {Key: "Name2", Value: "Tag Name Value2"}},
	}

	for {
		fmt.Println("TagHandler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. Disk List")
		fmt.Println("2. Disk Create")
		fmt.Println("3. Disk Get")
		fmt.Println("4. Disk Delete")
		fmt.Println("9. ListIID")

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return
			case 9:
				result, err := handler.ListIID()
				if err != nil {
					cblogger.Info(" ListIID Lookup Failed : ", err)
				} else {
					cblogger.Info(" ListIID Lookup Result")
					//cblogger.Info(result)
					spew.Dump(result)
				}

			case 1:
				cblogger.Infof("Lookup disk list")
				result, err := handler.ListDisk()
				if err != nil {
					cblogger.Info(" Disk List Lookup Failed : ", err)
				} else {
					cblogger.Info("Disk List Lookup Result")
					cblogger.Debug(result)
					cblogger.Infof("Log Level : [%s]", cblog.GetLevel())
					//spew.Dump(result)
					cblogger.Info("Number of output results : ", len(result))
				}

			case 2:
				cblogger.Infof("[%s] Disk Create Test", reqIID.SystemId)
				result, err := handler.CreateDisk(reqInfo)
				if err != nil {
					cblogger.Infof(reqIID.SystemId, " Disk Create Failed : ", err)
				} else {
					cblogger.Info("Disk Create Result : ", result)
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] Disk Lookup Test - Key[%s]", reqIID.SystemId, reqKey)
				result, err := handler.GetDisk(reqIID)
				if err != nil {
					cblogger.Infof("[%s] Disk Lookup Failed : [%v]", reqKey, err)
				} else {
					cblogger.Infof("[%s] Disk Lookup Result : [%s]", reqKey, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] Disk Delete Test - Key[%s]", reqIID.SystemId, reqKey)
				result, err := handler.DeleteDisk(reqIID)
				if err != nil {
					cblogger.Infof("[%s] Disk Delete Failed : [%v]", reqKey, err)
				} else {
					cblogger.Infof("[%s] Disk Delete Result : [%v]", reqKey, result)
				}
			}
		}
	}
}

// Test MyImage
func handleMyImage() {
	cblogger.Debug("Start MyImage Resource Test")

	ResourceHandler, err := getResourceHandler("MyImage")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.MyImageHandler)

	//config := readConfigFile()
	//VmID := config.Aws.VmID

	myImageName := "CB-MyImage-Test"
	myImageId := "ami-0d6a2bb960481ce68"

	for {
		fmt.Println("MyImage Management")
		fmt.Println("0. Quit")
		fmt.Println("1. MyImage List")
		fmt.Println("2. MyImage Create")
		fmt.Println("3. MyImage Get")
		fmt.Println("4. MyImage Delete")
		fmt.Println("9. ListIID")

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return

			case 9:
				result, err := handler.ListIID()
				if err != nil {
					cblogger.Info(" ListIID Lookup Failed : ", err)
				} else {
					cblogger.Info(" ListIID Lookup Result")
					//cblogger.Info(result)
					spew.Dump(result)
				}

			case 1:
				result, err := handler.ListMyImage()
				if err != nil {
					cblogger.Info(" MyImage List Lookup Failed : ", err)
				} else {
					cblogger.Info("MyImage List Lookup Result")
					cblogger.Info(result)

					if result != nil {
						myImageId = result[0].IId.SystemId // 조회 및 삭제를 위해 생성된 ID로 변경
					}
				}

			case 2:
				cblogger.Infof("[%s] MyImage Create Test", myImageName)

				myImageReqInfo := irs.MyImageInfo{
					IId: irs.IID{NameId: myImageName},
					SourceVM: irs.IID{
						SystemId: "i-0d6a2bb960481ce68", // VM ID
					},
					TagList: []irs.KeyValue{{Key: "Name", Value: myImageName}},
				}

				result, err := handler.SnapshotVM(myImageReqInfo)
				if err != nil {
					cblogger.Infof(myImageName, " MyImage Create Failed : ", err)
				} else {
					cblogger.Infof("[%s] MyImage Create Result : [%v]", myImageName, result)
					myImageId = result.IId.SystemId
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] MyImage Lookup Test", myImageId)
				result, err := handler.GetMyImage(irs.IID{SystemId: myImageId})
				if err != nil {
					cblogger.Infof(myImageId, " MyImage Lookup Failed : ", err)
				} else {
					cblogger.Infof("[%s] MyImage Lookup Result : [%v]", myImageId, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] MyImage Delete Test", myImageId)
				result, err := handler.DeleteMyImage(irs.IID{SystemId: myImageId})
				if err != nil {
					cblogger.Infof(myImageId, " MyImage Delete Failed : ", err)
				} else {
					cblogger.Infof("[%s] MyImage Delete Result : [%v]", myImageId, result)
				}
			}
		}
	}
}

func handleFileSystem() {
	cblogger.Debug("Start FileSystem Resource Test")

	ResourceHandler, err := getResourceHandler("FileSystem")
	if err != nil {
		panic(err)
	}
	handler := ResourceHandler.(irs.FileSystemHandler)

	fileSystemName := "CB-FileSystem-Test"
	fileSystemId := "fs-03e7efece09ee8a48"
	subnetId := "subnet-0bfe6a60232e5bb3b" // vpc-c0479cab // ap-northeast-2a

	for {
		fmt.Println("FileSystem Management")
		fmt.Println("0. Quit")
		fmt.Println("1. Get Meta Info")
		fmt.Println("2. FileSystem List")
		fmt.Println("3. FileSystem Create")
		fmt.Println("4. FileSystem Get")
		fmt.Println("5. FileSystem Delete")
		fmt.Println("6. Add Access Subnet")
		fmt.Println("7. Remove Access Subnet")
		fmt.Println("8. List Access Subnet")
		fmt.Println("9. ListIID")

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
				cblogger.Info("Get Meta Info Test")
				result, err := handler.GetMetaInfo()
				if err != nil {
					cblogger.Info("Get Meta Info Failed : ", err)
				} else {
					cblogger.Info("Get Meta Info Result")
					spew.Dump(result)
				}

			case 2:
				result, err := handler.ListFileSystem()
				if err != nil {
					cblogger.Info("FileSystem List Lookup Failed : ", err)
				} else {
					cblogger.Info("FileSystem List Lookup Result")
					// cblogger.Info(result)
					spew.Dump(result)
					if result != nil && len(result) > 0 {
						fileSystemId = result[0].IId.SystemId // 조회 및 삭제를 위해 생성된 ID로 변경
						cblogger.Infof("Total number of file systems: %d", len(result))
					}
				}

			case 3:
				cblogger.Infof("[%s] FileSystem Create Test", fileSystemName)

				// 기본 설정 모드 테스트
				fileSystemReqInfo := irs.FileSystemInfo{
					IId: irs.IID{NameId: fileSystemName},
					VpcIID: irs.IID{
						SystemId: "vpc-0a48d45f6bc3a71da", // VPC ID
					},
				}

				// One Zone + MaxIO 에러 테스트 (주석 해제하여 테스트)
				/*
					fileSystemReqInfo := irs.FileSystemInfo{
						IId: irs.IID{NameId: fileSystemName},
						VpcIID: irs.IID{
							SystemId: "vpc-0a48d45f6bc3a71da",
						},
						FileSystemType: irs.ZoneType,
						Zone:           "ap-northeast-2a",
						PerformanceInfo: map[string]string{
							"ThroughputMode":  "Bursting",
							"PerformanceMode": "MaxIO", // One Zone에서는 에러가 발생해야 함
						},
						Encryption: true,
					}
				*/

				// Regional + MaxIO 테스트 (정상 동작 확인용)
				/*
					fileSystemReqInfo := irs.FileSystemInfo{
						IId: irs.IID{NameId: fileSystemName},
						VpcIID: irs.IID{
							SystemId: "vpc-0a48d45f6bc3a71da",
						},
						FileSystemType: irs.RegionType,
						PerformanceInfo: map[string]string{
							"ThroughputMode":        "Provisioned",
							"PerformanceMode":       "MaxIO",
							"ProvisionedThroughput": "128",
						},
						Encryption: true,
					}
				*/

				result, err := handler.CreateFileSystem(fileSystemReqInfo)
				if err != nil {
					cblogger.Infof(fileSystemName, " FileSystem Create Failed : ", err)
				} else {
					cblogger.Infof("[%s] FileSystem Create Result : [%v]", fileSystemName, result)
					fileSystemId = result.IId.SystemId
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] FileSystem Lookup Test", fileSystemId)
				result, err := handler.GetFileSystem(irs.IID{SystemId: fileSystemId})
				if err != nil {
					cblogger.Infof(fileSystemId, " FileSystem Lookup Failed : ", err)
				} else {
					cblogger.Infof("[%s] FileSystem Lookup Result : [%v]", fileSystemId, result)
					spew.Dump(result)
				}

			case 5:
				cblogger.Infof("[%s] FileSystem Delete Test", fileSystemId)
				result, err := handler.DeleteFileSystem(irs.IID{SystemId: fileSystemId})
				if err != nil {
					cblogger.Infof(fileSystemId, " FileSystem Delete Failed : ", err)
				} else {
					cblogger.Infof("[%s] FileSystem Delete Result : [%v]", fileSystemId, result)
				}

			case 6:
				cblogger.Infof("[%s] Add Access Subnet Test", subnetId)
				result, err := handler.AddAccessSubnet(irs.IID{SystemId: fileSystemId}, irs.IID{SystemId: subnetId})
				if err != nil {
					cblogger.Infof(subnetId, " Add Access Subnet Failed : ", err)
				} else {
					cblogger.Infof("[%s] Add Access Subnet Result : [%v]", subnetId, result)
					spew.Dump(result)
				}

			case 7:
				cblogger.Infof("[%s] Remove Access Subnet Test", subnetId)
				result, err := handler.RemoveAccessSubnet(irs.IID{SystemId: fileSystemId}, irs.IID{SystemId: subnetId})
				if err != nil {
					cblogger.Infof(subnetId, " Remove Access Subnet Failed : ", err)
				} else {
					cblogger.Infof("[%s] Remove Access Subnet Result : [%v]", subnetId, result)
				}

			case 8:
				cblogger.Infof("[%s] List Access Subnet Test", fileSystemId)
				result, err := handler.ListAccessSubnet(irs.IID{SystemId: fileSystemId})
				if err != nil {
					cblogger.Infof(fileSystemId, " List Access Subnet Failed : ", err)
				} else {
					cblogger.Infof("[%s] List Access Subnet Result : [%v]", fileSystemId, result)
					spew.Dump(result)
				}

			case 9:
				result, err := handler.ListIID()
				if err != nil {
					cblogger.Info(" ListIID Lookup Failed : ", err)
				} else {
					cblogger.Info(" ListIID Lookup Result")
					spew.Dump(result)
				}
			}
		}
	}
}

// handlerType : resources폴더의 xxxHandler.go에서 Handler이전까지의 문자열
// (예) ImageHandler.go -> "Image"
func getResourceHandler(handlerType string) (interface{}, error) {
	//var cloudDriver idrv.CloudDriver
	//cloudDriver = new(awsdrv.AwsDriver)
	cloudDriver := new(awsdrv.AwsDriver)

	config := readConfigFile()
	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			ClientId:     config.Aws.AwsAccessKeyID,
			ClientSecret: config.Aws.AwsSecretAccessKey,
			StsToken:     config.Aws.AwsStsToken,
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
	case "PMKS":
		resourceHandler, err = cloudConnection.CreateClusterHandler()
	case "RegionZone":
		resourceHandler, err = cloudConnection.CreateRegionZoneHandler()
	case "PriceInfo":
		resourceHandler, err = cloudConnection.CreatePriceInfoHandler()
	case "Tag":
		resourceHandler, err = cloudConnection.CreateTagHandler()
	case "Disk":
		resourceHandler, err = cloudConnection.CreateDiskHandler()
	case "MyImage":
		resourceHandler, err = cloudConnection.CreateMyImageHandler()
	case "FileSystem":
		resourceHandler, err = cloudConnection.CreateFileSystemHandler()
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
			ClientId:     config.Aws.AwsAccessKeyID,
			ClientSecret: config.Aws.AwsSecretAccessKey,
			StsToken:     config.Aws.AwsStsToken,
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
			ClientId:     config.Aws.AwsAccessKeyID,
			ClientSecret: config.Aws.AwsSecretAccessKey,
			StsToken:     config.Aws.AwsStsToken,
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
		AwsAccessKeyID     string `yaml:"aws_access_key_id"`
		AwsSecretAccessKey string `yaml:"aws_secret_access_key"`
		AwsStsToken        string `yaml:"aws_sts_token"`
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

// 환경 설정 파일 읽기
// 환경변수 CBSPIDER_TEST_CONF_PATH에 테스트에 사용할 파일 경로를 설정해야 함. (예: config/config.yaml)
func readConfigFile() Config {
	// Set Environment Value of Project Root Path
	confPath := os.Getenv("CBSPIDER_TEST_CONF_PATH")
	cblogger.Info("Set the full path to the config files you want to use for testing, including your AWS credentials, in the OS environment variable [CBSPIDER_TEST_CONF_PATH].")
	cblogger.Infof("OS environment variable [CBSPIDER_TEST_CONF_PATH] : [%s]", confPath)
	//cblogger.Infof("최종 환경 설정파일 경로 : [%s]", rootPath+"/config/config.yaml")
	//data, err := ioutil.ReadFile(rootPath + "/config/config.yaml")
	//data, err = ioutil.ReadFile("/Users/mzc01-swy/projects/feature_aws_filter_swy_240130/cloud-control-manager/cloud-driver/drivers/aws/main/Sample/config/config.yaml")

	data, err := ioutil.ReadFile(confPath)
	if err != nil {
		panic(err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}

	cblogger.Info("Loaded ConfigFile...")
	cblogger.Debug(config.Aws.AwsAccessKeyID, " ", config.Aws.Region)
	return config
}

func main() {
	cblogger.Info("=======================================")
	cblogger.Info("CB-Spider DEBUG Level : ", cblogger.Level.String())
	cblogger.Info("=======================================")

	// myimage
	//handleTag()
	// handlePublicIP() // PublicIP 생성 후 conf

	//handleVPC()
	//handleSecurity()
	//handleKeyPair()
	//handleVM()
	//handleDisk()
	//handleMyImage()
	//handleNLB()
	//handleCluster()
	// handleImage() //AMI
	// handleVNic() //Lancard
	//handleVMSpec()
	//handleRegionZone()
	//handlePriceInfo()
	//handleTag()
	handleFileSystem()
	//ts.TestScenarioFileSystem()
}
