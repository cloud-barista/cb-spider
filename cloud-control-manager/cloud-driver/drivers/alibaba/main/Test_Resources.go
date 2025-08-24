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
	//reqVMSpec := "ecs.g6.large"	// No GPU
	// reqVMSpec := "ecs.vgn5i-m8.4xlarge" // 1 GPU
	reqVMSpec := "" // 1 GPU
	//reqVMSpec := "ecs.gn6i-c24g1.24xlarge" // 4 GPUs

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
					cblogger.Error("Failed to retrieve VMSpec list : ", err)
				} else {
					cblogger.Info("Result of VMSpec list retrieval")
					spew.Dump(result)
				}

				fmt.Println("Finish ListVMSpec()")

			case 2:
				fmt.Println("Start GetVMSpec() ...")
				result, err := handler.GetVMSpec(reqVMSpec)
				if err != nil {
					cblogger.Error(reqVMSpec, "Failed to retrieve VMSpec list : ", err)
				} else {
					cblogger.Infof("Result of VMSpec list retrieval", reqVMSpec)
					spew.Dump(result)
				}
				fmt.Println("Finish GetVMSpec()")

			case 3:
				fmt.Println("Start ListOrgVMSpec() ...")
				result, err := handler.ListOrgVMSpec()
				if err != nil {
					cblogger.Error("Failed to retrieve VMSpec list : ", err)
				} else {
					cblogger.Info("Result of VMSpec list retrieval")
					cblogger.Info(result)
					//spew.Dump(result)
				}

				fmt.Println("Finish ListOrgVMSpec()")

			case 4:
				fmt.Println("Start GetOrgVMSpec() ...")
				result, err := handler.GetOrgVMSpec(reqVMSpec)
				if err != nil {
					cblogger.Error(reqVMSpec, "Failed to retrieve VMSpec list : ", err)
				} else {
					cblogger.Infof("Result of VMSpec list retrieval", reqVMSpec)
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
		panic(err)
	}
	handler := ResourceHandler.(irs.SecurityHandler)

	//config := readConfigFile()
	//VmID := config.Aws.VmID

	//securityName := "CB-SecurityTestCidr"
	securityName := ""
	securityId := ""
	vpcId := ""

	for {
		fmt.Println("Security Management")
		fmt.Println("0. Quit")
		fmt.Println("1. Security List")
		fmt.Println("2. Security Create")
		fmt.Println("3. Security Get")
		fmt.Println("4. Security Delete")
		fmt.Println("5. Rule Add")
		fmt.Println("6. Rule Remove")
		fmt.Println("7. List IID")

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
					cblogger.Infof("Result of Security list retrieval : ", err)
				} else {
					cblogger.Info("Result of Security list retrieval")

					spew.Dump(result)
					if result != nil {
						securityId = result[0].IId.SystemId // Changed to the ID created for retrieval and deletion
					}
				}

			case 2:
				cblogger.Infof("[%s] Security creation test", securityName)
				tag1 := irs.KeyValue{Key: "", Value: ""}
				securityReqInfo := irs.SecurityReqInfo{
					IId:     irs.IID{NameId: securityName},
					VpcIID:  irs.IID{SystemId: vpcId},
					TagList: []irs.KeyValue{tag1},
					SecurityRules: &[]irs.SecurityRuleInfo{ // Security policy configuration
						// CIDR Test
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
						      IPProtocol: "-1", // Allow all
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
					cblogger.Infof(securityName, "Security creation Failed : ", err)
				} else {
					cblogger.Infof("[%s] Result of Security creation : [%v]", securityName, result)
					securityId = result.IId.SystemId
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] Security retrieval test", securityId)
				result, err := handler.GetSecurity(irs.IID{SystemId: securityId})
				if err != nil {
					cblogger.Infof(securityId, "Failed to retrieve Security list : ", err)
				} else {
					cblogger.Infof("[%s] Result of Security retrieve : [%v]", securityId, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] Security delete test", securityId)
				result, err := handler.DeleteSecurity(irs.IID{SystemId: securityId})
				if err != nil {
					cblogger.Infof(securityId, "Failed to deletion Security : ", err)
				} else {
					cblogger.Infof("[%s] Result of Security delete : [%s]", securityId, result)
				}
			case 5:
				cblogger.Infof("[%s] Test of add Rule", securityId)
				securityRules := &[]irs.SecurityRuleInfo{

					/*{
					   FromPort:   "20",
					   ToPort:     "21",
					   IPProtocol: "tcp",
					   Direction:  "inbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					/*{
					   FromPort:   "20",
					   ToPort:     "21",
					   IPProtocol: "tcp",
					   Direction:  "outbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					/*{
					   FromPort:   "8080",
					   ToPort:     "8080", // If either FromPort or ToPort is set to -1, replace the -1 with a blank
					   IPProtocol: "tcp",
					   Direction:  "inbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					/*{ //
					   FromPort:   "1323", // If either FromPort or ToPort is set to -1, replace the -1 with a blank
					   ToPort:     "1323",
					   IPProtocol: "tcp",
					   Direction:  "inbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					/*{
					   // All Port
					   FromPort:   "",
					   ToPort:     "",
					   IPProtocol: "icmp", // ICMP has no port information.
					   Direction:  "inbound",
					},*/
					/*{
					   //20-22 Port
					   FromPort:   "20",
					   ToPort:     "22",
					   IPProtocol: "tcp",
					   Direction:  "inbound",
					},*/
					/*{
					   // 80 Port
					   FromPort:   "80",
					   ToPort:     "80",
					   IPProtocol: "tcp",
					   Direction:  "inbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					/*{ // All port
					   //FromPort:   "",
					   //ToPort:     "",
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
					/*{
					   //20-22 Prot
					   FromPort:   "22",
					   ToPort:     "22",
					   IPProtocol: "tcp",
					   Direction:  "outbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					/*{
					   //20-22 Prot
					   FromPort:   "1000",
					   ToPort:     "1000",
					   IPProtocol: "tcp",
					   Direction:  "outbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					/*{
					   //20-22 Prot
					   FromPort:   "1",
					   ToPort:     "65535",
					   IPProtocol: "udp",
					   Direction:  "outbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					/*{
					   //20-22 Prot
					   FromPort:   "-1",
					   ToPort:     "-1",
					   IPProtocol: "icmp",
					   Direction:  "outbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					/*{
					   //20-22 Prot
					   FromPort:   "-1",
					   ToPort:     "-1",
					   IPProtocol: "all",
					   Direction:  "outbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					/*{
					   //20-22 Prot
					   FromPort:   "22",
					   ToPort:     "22",
					   IPProtocol: "tcp",
					   Direction:  "outbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					/*{
					   //20-22 Prot
					   FromPort:   "1000",
					   ToPort:     "1000",
					   IPProtocol: "tcp",
					   Direction:  "outbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					/*{
					   //20-22 Prot
					   FromPort:   "1",
					   ToPort:     "65535",
					   IPProtocol: "udp",
					   Direction:  "outbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					/*{
					   //20-22 Prot
					   FromPort:   "22",
					   ToPort:     "22",
					   IPProtocol: "tcp",
					   Direction:  "inbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					/*{
					   //20-22 Prot
					   FromPort:   "1000",
					   ToPort:     "1000",
					   IPProtocol: "tcp",
					   Direction:  "inbound",
					   CIDR:       "4.5.6.7/32",
					},*/
					/*{
					     //20-22 Port
					     FromPort:   "1",
					     ToPort:     "65535",
					     IPProtocol: "udp",
					     Direction:  "inbound",
					     CIDR:       "0.0.0.0/0",
					  },
					  {
					     //20-22 Prot
					     FromPort:   "-1",
					     ToPort:     "-1",
					     IPProtocol: "icmp",
					     Direction:  "inbound",
					     CIDR:       "0.0.0.0/0",
					  },*/
					{
						//20-22 Port
						FromPort:   "22",
						ToPort:     "22",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						//20-22 Port
						FromPort:   "1000",
						ToPort:     "1000",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						//20-22 Port
						FromPort:   "1",
						ToPort:     "65535",
						IPProtocol: "udp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						//20-22 Port
						FromPort:   "-1",
						ToPort:     "-1",
						IPProtocol: "icmp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},
				}

				result, err := handler.AddRules(irs.IID{SystemId: securityId}, securityRules)
				if err != nil {
					cblogger.Infof(securityId, "Failed to add Rule : ", err)
				} else {
					cblogger.Infof("[%s] Result of add Rule : [%v]", securityId, result)
					spew.Dump(result)
				}
			case 6:
				cblogger.Infof("[%s] Test of delete Rule", securityId)
				securityRules := &[]irs.SecurityRuleInfo{
					/*{
					      //20-22 Port
					      FromPort:   "20",
					      ToPort:     "21",
					      IPProtocol: "tcp",
					      Direction:  "inbound",
					      CIDR:       "0.0.0.0/0",
					},*/
					/*{
					      //20-22 Port
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
					   //20-22 Port
					   FromPort:   "20",
					   ToPort:     "22",
					   IPProtocol: "tcp",
					   Direction:  "inbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					/*{
					   //20-22 Port
					   FromPort:   "20",
					   ToPort:     "21",
					   IPProtocol: "udp",
					   Direction:  "inbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					/*{
					   // 8080 Port
					   FromPort:   "8080",
					   ToPort:     "8080", // If either FromPort or ToPort is set to -1, replace the -1 with a blank
					   IPProtocol: "tcp",
					   Direction:  "inbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					/*{ // 1323 Port
					   FromPort:   "1323", // If either FromPort or ToPort is set to -1, replace the -1 with a blank
					   ToPort:     "1323",
					   IPProtocol: "tcp",
					   Direction:  "inbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					/*{
					   // All Port
					   FromPort:   "",
					   ToPort:     "",
					   IPProtocol: "icmp", // ICMP has no port information
					   Direction:  "inbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					/*{
					   //FromPort:   "",
					   //ToPort:     "",
					   IPProtocol: "all",
					   Direction:  "inbound",
					   CIDR:       "0.0.0.0/0",
					},*/
					{
						//20-22 Port
						FromPort:   "22",
						ToPort:     "22",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						//20-22 Port
						FromPort:   "1000",
						ToPort:     "1000",
						IPProtocol: "tcp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						//20-22 Port
						FromPort:   "1",
						ToPort:     "65535",
						IPProtocol: "udp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},
					{
						//20-22 Port
						FromPort:   "-1",
						ToPort:     "-1",
						IPProtocol: "icmp",
						Direction:  "outbound",
						CIDR:       "0.0.0.0/0",
					},
					/*{
					   //20-22 Port
					   FromPort:   "-1",
					   ToPort:     "-1",
					   IPProtocol: "all",
					   Direction:  "outbound",
					   CIDR:       "0.0.0.0/0",
					},*/
				}

				result, err := handler.RemoveRules(irs.IID{SystemId: securityId}, securityRules)
				if err != nil {
					cblogger.Infof(securityId, "Fail to delete Rule : ", err)
				} else {
					cblogger.Infof("[%s] Result of delete Rule : [%v]", securityId, result)
				}
			case 7:
				cblogger.Infof("[%s] List IID test", securityId)

				result, err := handler.ListIID()
				if err != nil {
					cblogger.Infof(securityId, "Fail to delete Rule : ", err)
				} else {
					cblogger.Infof("[%s] Result of IID List ", securityId)
					for _, v := range result {
						cblogger.Infof("IID List: %+v", v)
					}
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

	keyPairName := ""
	//keyPairName := config.Aws.KeyName
	tag1 := irs.KeyValue{Key: "aaa", Value: "bbb"}

	for {
		fmt.Println("KeyPair Management")
		fmt.Println("0. Quit")
		fmt.Println("1. KeyPair List")
		fmt.Println("2. KeyPair Create")
		fmt.Println("3. KeyPair Get")
		fmt.Println("4. KeyPair Delete")
		fmt.Println("5. List IID")

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
					cblogger.Infof("Fail to retrieve Keypair list : ", err)
				} else {
					cblogger.Info("Result of retrieve Keypair list")
					//cblogger.Info(result)
					spew.Dump(result)
					if result != nil {
						keyPairName = result[0].IId.SystemId // Changed to the ID created for retrieval and deletion
					}
					cblogger.Info("Number of Keypair : ", len(result))
				}

			case 2:
				cblogger.Infof("[%s] Test of creation Keypair", keyPairName)
				keyPairReqInfo := irs.KeyPairReqInfo{
					IId:     irs.IID{NameId: keyPairName},
					TagList: []irs.KeyValue{tag1},
				}
				result, err := handler.CreateKey(keyPairReqInfo)
				if err != nil {
					cblogger.Infof(keyPairName, "Fail to create Keypair : ", err)
				} else {
					cblogger.Infof("[%s] Result of create Keypair : [%s]", keyPairName, result)
					spew.Dump(result)
				}
			case 3:
				cblogger.Infof("[%s] Test of retrieve Keypair", keyPairName)
				result, err := handler.GetKey(irs.IID{SystemId: keyPairName})
				if err != nil {
					cblogger.Infof(keyPairName, " Fail to retrieve Keypair : ", err)
				} else {
					cblogger.Infof("[%s] Result of Keypair : [%s]", keyPairName, result)
					spew.Dump(result)
				}
			case 4:
				cblogger.Infof("[%s] Test of delete Keypair", keyPairName)
				result, err := handler.DeleteKey(irs.IID{SystemId: keyPairName})
				if err != nil {
					cblogger.Infof(keyPairName, "Fail to delete Keypair : ", err)
				} else {
					cblogger.Infof("[%s] Result of delete Keypair : [%s]", keyPairName, result)
				}
			case 5:
				cblogger.Infof("List IID test")

				result, err := handler.ListIID()
				if err != nil {
					cblogger.Infof("Fail to List IID : ", err)
				} else {
					cblogger.Infof("[%s] Result of IID List")
					for _, v := range result {
						cblogger.Infof("IID List: %+v", v)
					}
				}

			}
		}
	}
}

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

	subnetReqVpcInfo := irs.IID{SystemId: ""}
	reqSubnetId := irs.IID{SystemId: ""}
	cblogger.Debug(subnetReqInfo)
	cblogger.Debug(subnetReqVpcInfo)
	cblogger.Debug(reqSubnetId)
	// tag1 := irs.KeyValue{Key: "aaa", Value: "bbb"}
	// stag1 := irs.KeyValue{Key: "saa", Value: "sbb"}
	// stag2 := irs.KeyValue{Key: "scc", Value: "sdd"}
	vpcReqInfo := irs.VPCReqInfo{
		IId:       irs.IID{NameId: "New-CB-VPC"},
		IPv4_CIDR: "10.0.0.0/16",
		SubnetInfoList: []irs.SubnetInfo{
			{
				IId:       irs.IID{NameId: "New-CB-Subnet"},
				IPv4_CIDR: "10.0.1.0/24",
				// TagList:   []irs.KeyValue{stag1},
			},

			{
				IId:       irs.IID{NameId: "New-CB-Subnet2"},
				IPv4_CIDR: "10.0.2.0/24",
				// TagList:   []irs.KeyValue{stag2},
			},
		},
		// TagList: []irs.KeyValue{tag1},
		//Id:   "",
		//Name: "CB-VNet-Subnet", // Not used due to internal automatic implementation within the driver, without relying on external web tools
		//CidrBlock: "10.0.0.0/16",
		//CidrBlock: "192.168.0.0/16",
	}

	reqVpcId := irs.IID{SystemId: ""}

	for {
		fmt.Println("Handler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. VPC List")
		fmt.Println("2. VPC Create")
		fmt.Println("3. VPC Get")
		fmt.Println("4. VPC Delete")
		fmt.Println("5. Add Subnet")
		fmt.Println("6. Delete Subnet")
		fmt.Println("7. List IID")

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
					cblogger.Infof("Fail to retrieve VPC list : ", err)
				} else {
					cblogger.Info("Result of retrieve VPC list")

					spew.Dump(result)
					// Only one exists internally
					// Automatically updated the request ID to the first subnet ID in the list for retrieval and deletion tests
					if result != nil {
						reqVpcId = result[0].IId    // Changed to the ID created for retrieval and deletion.
						subnetReqVpcInfo = reqVpcId // For testing subnet addition/deletion.
					}
				}

			case 2:
				cblogger.Infof("[%s] Test of create VPC", vpcReqInfo.IId.NameId)
				//vpcReqInfo := irs.VPCReqInfo{}
				result, err := handler.CreateVPC(vpcReqInfo)
				if err != nil {
					cblogger.Infof(reqVpcId.NameId, "Fail to create VPC : ", err)
				} else {
					cblogger.Infof("Result of create VPC : ", result)
					reqVpcId = result.IId
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] Test of retrieve VPC", reqVpcId)
				result, err := handler.GetVPC(reqVpcId)
				if err != nil {
					cblogger.Infof("[%s] Fail to retrieve VPC : ", reqVpcId, err)
				} else {
					cblogger.Infof("[%s] Result of retrieve VPC : [%s]", reqVpcId, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] Test of delete VPC", reqVpcId)
				result, err := handler.DeleteVPC(reqVpcId)
				if err != nil {
					cblogger.Infof("[%s] Fail to delete VPC : ", reqVpcId, err)
				} else {
					cblogger.Infof("[%s] Result of delete VPC : [%s]", reqVpcId, result)
				}

			case 5:
				cblogger.Infof("[%s] Test of add Subnet", vpcReqInfo.IId.NameId)
				result, err := handler.AddSubnet(subnetReqVpcInfo, subnetReqInfo)
				if err != nil {
					cblogger.Infof(reqSubnetId.NameId, "Fail to add Subnet : ", err)
				} else {
					cblogger.Infof("Result of ad Subnet : ", result)
					reqSubnetId = result.IId
					spew.Dump(result)
				}

			case 6:
				cblogger.Infof("[%s] Test of delete Subnet", reqSubnetId.SystemId)
				result, err := handler.RemoveSubnet(subnetReqVpcInfo, reqSubnetId)
				if err != nil {
					cblogger.Infof("[%s] Fail to delete Subnet : ", reqSubnetId.SystemId, err)
				} else {
					cblogger.Infof("[%s] Result of delete Subnet : [%s]", reqSubnetId.SystemId, result)
				}
			case 7:
				cblogger.Infof("List IID test")

				result, err := handler.ListIID()
				if err != nil {
					cblogger.Infof("Fail to List IID : ", err)
				} else {
					cblogger.Infof("[%s] Result of IID List")
					for _, v := range result {
						cblogger.Infof("IID List: %+v", v)
					}
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
		IId: irs.IID{NameId: "Test OS Image", SystemId: ""},
		//Id:   "",
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
					cblogger.Infof("Fail to retrieve Image list : ", err)
				} else {
					cblogger.Info("Result of retrieve Image list")
					cblogger.Debug(result)
					cblogger.Info("Number of result : ", len(result))

					if cblogger.Level.String() == "debug" {
						spew.Dump(result)
					}

					if result != nil {
						imageReqInfo.IId = result[0].IId
					}
				}

			case 2:
				cblogger.Infof("[%s] Test of create Image", imageReqInfo.IId.NameId)
				result, err := handler.CreateImage(imageReqInfo)
				if err != nil {
					cblogger.Infof(imageReqInfo.IId.NameId, "Fail to create Image : ", err)
				} else {
					cblogger.Infof("Result of create Image : ", result)
					imageReqInfo.IId = result.IId // Changed to the ID created for retrieval and deletion
					spew.Dump(result)
				}

			case 3:
				cblogger.Infof("[%s] Test of retrieve Image", imageReqInfo.IId)
				result, err := handler.GetImage(imageReqInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] Fail to retrieve Image : ", imageReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] Result of retrieve Image : [%s]", imageReqInfo.IId.NameId, result)
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] Test of delete Image", imageReqInfo.IId.NameId)
				result, err := handler.DeleteImage(imageReqInfo.IId)
				if err != nil {
					cblogger.Infof("[%s] Fail to delete Image : ", imageReqInfo.IId.NameId, err)
				} else {
					cblogger.Infof("[%s] Result of delete Image : [%s]", imageReqInfo.IId.NameId, result)
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
	VmID := irs.IID{SystemId: ""}

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
		fmt.Println("10. List IID")

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
				tag1 := irs.KeyValue{Key: "vaa", Value: "vbb"}

				vmReqInfo := irs.VMReqInfo{
					// IId: irs.IID{NameId: ""},

					ImageIID: irs.IID{SystemId: ""},
					VpcIID:   irs.IID{SystemId: ""},
					//SubnetIID: irs.IID{SystemId: ""},
					// SubnetIID: irs.IID{SystemId: ""}, //Tokyo Zone B
					SubnetIID: irs.IID{SystemId: ""}, //hongkong-c
					//SecurityGroupIIDs: []irs.IID{{SystemId: ""}, {SystemId: ""}},
					SecurityGroupIIDs: []irs.IID{{SystemId: ""}}, // Hong Kong region
					//VMSpecName:        "ecs.t5-lc2m1.nano",
					//VMSpecName: "ecs.g6.large", //cn-wulanchabu region
					// VMSpecName: "ecs.t5-lc2m1.nano", //Tokyo region
					VMSpecName: "ecs.t5-lc2m1.nano", //Hong Kong region
					// KeyPairIID: irs.IID{SystemId: ""},
					KeyPairIID: irs.IID{SystemId: ""}, //Hong Kong region
					//VMUserId:          "root", //root only
					//VMUserPasswd: "Cbuser!@#", //Must include both uppercase and lowercase letters, and one of numbers or special characters

					// RootDiskType: "cloud_efficiency", //cloud / cloud_efficiency / cloud_ssd / cloud_essd
					// RootDiskSize: "default",
					//RootDiskType: "cloud_ssd", //cloud / cloud_efficiency / cloud_ssd / cloud_essd
					//RootDiskSize: "22",
					TagList: []irs.KeyValue{tag1},
				}

				vmInfo, err := vmHandler.StartVM(vmReqInfo)
				if err != nil {
					//panic(err)
					cblogger.Error("Fail to create VM")
					cblogger.Error(err)
				} else {
					cblogger.Info("Result of create VM", vmInfo)
					spew.Dump(vmInfo)
					VmID = vmInfo.IId
				}
				//cblogger.Info(vm)

				cblogger.Info("Finish Create VM")

			case 2:
				vmInfo, err := vmHandler.GetVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] Fail to retrieve VM", VmID)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] Result of retrieve VM", VmID)
					cblogger.Info(vmInfo)
					spew.Dump(vmInfo)
				}

			case 3:
				cblogger.Info("Start Suspend VM ...")
				result, err := vmHandler.SuspendVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] VM Suspend fail - [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] VM Suspend success - [%s]", VmID, result)
				}

			case 4:
				cblogger.Info("Start Resume  VM ...")
				result, err := vmHandler.ResumeVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] VM Resume fail - [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] VM Resume success - [%s]", VmID, result)
				}

			case 5:
				cblogger.Info("Start Reboot  VM ...")
				result, err := vmHandler.RebootVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] VM Reboot fail - [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] VM Reboot success - [%s]", VmID, result)
				}

			case 6:
				cblogger.Info("Start Terminate  VM ...")
				result, err := vmHandler.TerminateVM(VmID)
				if err != nil {
					cblogger.Errorf("[%s] VM Terminate fail - [%s]", VmID, result)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] VM Terminate success - [%s]", VmID, result)
				}

			case 7:
				cblogger.Info("Start Get VM Status...")
				vmStatus, err := vmHandler.GetVMStatus(VmID)
				if err != nil {
					cblogger.Errorf("[%s] VM Get Status fail", VmID)
					cblogger.Error(err)
				} else {
					cblogger.Infof("[%s] VM Get Status success : [%s]", VmID, vmStatus)
				}

			case 8:
				cblogger.Info("Start ListVMStatus ...")
				vmStatusInfos, err := vmHandler.ListVMStatus()
				if err != nil {
					cblogger.Error("ListVMStatus fail")
					cblogger.Error(err)
				} else {
					cblogger.Info("ListVMStatus success")
					cblogger.Info(vmStatusInfos)
					spew.Dump(vmStatusInfos)
				}

			case 9:
				cblogger.Info("Start ListVM ...")
				vmList, err := vmHandler.ListVM()
				if err != nil {
					cblogger.Error("ListVM fail")
					cblogger.Error(err)
				} else {
					cblogger.Info("ListVM success")
					cblogger.Info("=========== VM List ================")
					cblogger.Info(vmList)
					spew.Dump(vmList)
					if len(vmList) > 0 {
						VmID = vmList[0].IId
					}
				}
			case 10:
				cblogger.Infof("List IID test")

				result, err := vmHandler.ListIID()
				if err != nil {
					cblogger.Infof("Fail to List IID : ", err)
				} else {
					cblogger.Infof("[%s] Result of IID List")
					for _, v := range result {
						cblogger.Infof("IID List: %+v", v)
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
	tag1 := irs.KeyValue{Key: "naa", Value: "vbb"}

	nlbReqInfo := irs.NLBInfo{
		// TCP
		IId:           irs.IID{NameId: ""},
		VpcIID:        irs.IID{SystemId: ""},
		Type:          "PUBLIC",
		Listener:      irs.ListenerInfo{Protocol: "TCP", Port: "80"},
		HealthChecker: irs.HealthCheckerInfo{Protocol: "TCP", Port: "80", Interval: 5, Timeout: 2, Threshold: 3},
		VMGroup: irs.VMGroupInfo{
			Protocol: "TCP",
			Port:     "80",

			VMs: &[]irs.IID{{SystemId: ""}, {SystemId: ""}},
		},
		TagList: []irs.KeyValue{tag1},
		// UDP
		//IId:    irs.IID{NameId: ""},
		//VpcIID:   irs.IID{SystemId: ""},
		//Type:     "PUBLIC",
		//Listener: irs.ListenerInfo{Protocol: "UDP", Port: "23"},
		//HealthChecker: irs.HealthCheckerInfo{Protocol: "UDP", Port: "23", Interval: 5, Timeout: 2, Threshold: 3},
		//VMGroup: irs.VMGroupInfo{
		//	//Protocol: "UDP",
		//	//Port:     "23",
		//
		//	VMs: &[]irs.IID{{SystemId: ""}, {SystemId: ""}},
		//},
	}

	reqNLBId := irs.IID{SystemId: ""}

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
					cblogger.Infof("Fail to retrieve NLB list : ", err)
				} else {
					cblogger.Info("Result of retrieve NLB list")

					spew.Dump(result)
				}

			case 2:
				cblogger.Infof("[%s] Test of create NLB", nlbReqInfo.IId.NameId)
				//vpcReqInfo := irs.VPCReqInfo{}
				result, err := handler.CreateNLB(nlbReqInfo)
				if err != nil {
					cblogger.Infof(nlbReqInfo.IId.NameId, "Fail to create NLB : ", err)
				} else {
					cblogger.Infof("Result of create NLB : ", result)
					//reqNLBId = result.IId // Changed to the ID created for retrieval and deletion
					spew.Dump(result)
				}

			case 3:
				reqNLBId = irs.IID{SystemId: ""}
				cblogger.Infof("[%s] Test of retrieve NLB", reqNLBId)
				result, err := handler.GetNLB(reqNLBId)
				if err != nil {
					cblogger.Infof("[%s] Fail to retrieve NLB : ", reqNLBId, err)
				} else {
					cblogger.Infof("[%s] Result of retrieve NLB : [%s]", reqNLBId, result)
					spew.Dump(result)
				}

			case 4:
				reqNLBId.SystemId = ""
				cblogger.Infof("[%s] Test of delete NLB", reqNLBId)
				result, err := handler.DeleteNLB(reqNLBId)
				if err != nil {
					cblogger.Infof("[%s] Fail to delete NLB : ", reqNLBId, err)
				} else {
					cblogger.Infof("[%s] Result of delete NLB : [%s]", reqNLBId, result)
				}

			case 5:
				cblogger.Infof("[%s] Test of add VM", reqNLBId)
				reqNLBId.SystemId = ""
				vmIID := irs.IID{SystemId: ""}
				result, err := handler.AddVMs(reqNLBId, &[]irs.IID{vmIID})
				if err != nil {
					cblogger.Infof("Fail to add VM : ", err)
				} else {
					cblogger.Infof("Result of add VM : ", result)
					//reqSubnetId = result.SubnetInfoList[0].IId // Changed to the ID created for retrieval and deletion
					spew.Dump(result)
				}

			case 6:
				cblogger.Infof("[%s] Test of delete VM", reqNLBId.SystemId)

				reqNLBId.SystemId = ""
				vmIID := irs.IID{SystemId: ""}
				result, err := handler.RemoveVMs(reqNLBId, &[]irs.IID{vmIID})
				if err != nil {
					cblogger.Infof("Fail to delete VM : ", err)
				} else {
					cblogger.Infof("Result of delete VM : [%s]", result)
				}
			case 7:
				cblogger.Infof("[%s] Test of retrieve NLB VM", reqNLBId)
				cblogger.Infof("[%s] Test of add VM", reqNLBId)
				reqNLBId.SystemId = ""
				result, err := handler.GetVMGroupHealthInfo(reqNLBId)
				if err != nil {
					cblogger.Infof("[%s] Fail to retrieve NLB VM : ", reqNLBId.SystemId, err)
				} else {
					cblogger.Infof("[%s] Result of retrieve NLB VM Health : [%s]", reqNLBId.SystemId, result)
					spew.Dump(result)
				}
			case 8:
				cblogger.Infof("[%s] Test of change NLB Listener", reqNLBId)
				reqNLBId.SystemId = ""
				changeListener := irs.ListenerInfo{}
				changeListener.Protocol = "tcp"
				changeListener.Port = "8080"
				result, err := handler.ChangeListener(reqNLBId, changeListener)
				if err != nil {
					cblogger.Infof("[%s] Fail to change NLB Listener : ", reqNLBId.SystemId, err)
				} else {
					cblogger.Infof("[%s] Result of change NLB Listener : [%s]", reqNLBId.SystemId, result)
					spew.Dump(result)
				}
			case 9:
				cblogger.Infof("[%s] Test of change NLB VM Group", reqNLBId)
				result, err := handler.ChangeVMGroupInfo(reqNLBId, irs.VMGroupInfo{
					Protocol: "TCP",
					Port:     "8080",
				})
				if err != nil {
					cblogger.Infof("[%s] Fail to change NLB VM Group : ", reqNLBId.SystemId, err)
				} else {
					cblogger.Infof("[%s] Result of change NLB VM Group : [%s]", reqNLBId.SystemId, result)
					spew.Dump(result)
				}
			case 10:
				reqNLBId.SystemId = ""
				reqHealthCheckInfo := irs.HealthCheckerInfo{
					Protocol:  "tcp",
					Port:      "85",
					Interval:  49,
					Timeout:   30,
					Threshold: 9,
				}
				cblogger.Infof("[%s] Test of change NLB Health Checker", reqNLBId)
				result, err := handler.ChangeHealthCheckerInfo(reqNLBId, reqHealthCheckInfo)
				if err != nil {
					cblogger.Infof("[%s] Fail to change NLB Health Checker : ", reqNLBId.SystemId, err)
				} else {
					cblogger.Infof("[%s] Resutl of change NLB Health Checker : [%s]", reqNLBId.SystemId, result)
					spew.Dump(result)
				}

			}
		}
	}
}

func handleRegionZone() {
	cblogger.Debug("Start RegionZone Test")
	ResourceHandler, err := testconf.GetResourceHandler("RegionZone")
	if err != nil {
		//panic(err)
		cblogger.Error(err)
	}
	handler := ResourceHandler.(irs.RegionZoneHandler)
	cblogger.Info(handler)

	for {
		fmt.Println("Handler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. ListRegionZone List")
		fmt.Println("2. ListOrgRegion ")
		fmt.Println("3. ListOrgZone ")
		fmt.Println("4. GetRegionZone ")

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
					cblogger.Infof("Fail to retrieve RegionZone list : ", err)
				} else {
					cblogger.Info("Result of retrieve RegionZone list")
					spew.Dump(result)
				}

			case 2:
				result, err := handler.ListOrgRegion()
				if err != nil {
					cblogger.Infof("Fail to retrieve ListOrgRegion list : ", err)
				} else {
					cblogger.Info("Result of retrieve ListOrgRegion list")
					spew.Dump(result)
				}

			case 3:
				result, err := handler.ListOrgZone()
				if err != nil {
					cblogger.Infof("Fail to retrieve ListOrgZone list : ", err)
				} else {
					cblogger.Info("Result of retrieve ListOrgZone list")
					spew.Dump(result)
				}
			case 4:
				regionId := "ap-northeast-2"
				result, err := handler.GetRegionZone(regionId)
				if err != nil {
					cblogger.Infof("Fail to retrieve GetRegionZone : ", regionId, err)
				} else {
					cblogger.Info("Result of retrieve GetRegionZone", regionId)
					spew.Dump(result)
				}

			}
		}
	}
}

func handlePriceInfo() {
	cblogger.Debug("Start handlePriceInfo Test")
	ResourceHandler, err := testconf.GetResourceHandler("PriceInfo")
	if err != nil {
		//panic(err)
		cblogger.Error(err)
	}
	handler := ResourceHandler.(irs.PriceInfoHandler)
	cblogger.Info(handler)

	for {
		fmt.Println("Handler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. ListProductFamily List")
		fmt.Println("2. GetPriceInfo ")

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
				regionName := "cn-hongkong"
				result, err := handler.ListProductFamily(regionName)
				if err != nil {
					cblogger.Infof("Fail to retrieve ProductFamily list : ", err)
				} else {
					cblogger.Info("Result of retrieve ProductFamily list")
					spew.Dump(result)
				}

			case 2:
				productFamily := "ecs"
				regionName := ""

				result, err := handler.GetPriceInfo(productFamily, regionName, []irs.KeyValue{})
				if err != nil {
					cblogger.Info("Fail to retrieve PriceInfo : ", err)
				} else {
					cblogger.Info("Result of retrieve PriceInfo")

					cblogger.Info(result)
				}
			}
		}
	}
}

func handleTagInfo() {
	cblogger.Debug("Start handleTagInfo Test")
	ResourceHandler, err := testconf.GetResourceHandler("Tag")
	if err != nil {
		//panic(err)
		cblogger.Error(err)
	}
	handler := ResourceHandler.(irs.TagHandler)
	cblogger.Info(handler)

	for {
		fmt.Println("Handler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. ListTag List")
		fmt.Println("2. GetTag ")
		fmt.Println("3. FindTag ")
		fmt.Println("4. AddTag ")
		fmt.Println("5. RemoveTag ")

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}

		// resourceType := irs.RSType("VM")
		resourceType := irs.ALL

		resourceIID := irs.IID{NameId: "", SystemId: ""}
		// resourceType := irs.RSType("CLUSTER")
		// resourceIID := irs.IID{NameId: "cs-issue-test", SystemId: ""}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return
			case 1:

				result, err := handler.ListTag(resourceType, resourceIID)
				if err != nil {
					cblogger.Infof("Fail to retrieve Tag list : ", err)
				} else {
					cblogger.Info("Result of Tag list")
					spew.Dump(result)
				}

			case 2:
				tagName := "tagntest3"
				tagName = ""
				result, err := handler.GetTag(resourceType, resourceIID, tagName)
				if err != nil {
					cblogger.Info("Fail to retrieve Tag : ", err)
				} else {
					cblogger.Info("Result of retrieve GetTag")
					//spew.Dump(result)
					cblogger.Info(result)
				}
			case 3:
				tagName := "aaa"

				result, err := handler.FindTag(resourceType, tagName)
				if err != nil {
					cblogger.Info("Fail to retrieve Tag : ", err)
				} else {
					cblogger.Info("Result of retrieve FindTag")
					spew.Dump(result)
					// cblogger.Info(result)
				}
			case 4:
				newTag := irs.KeyValue{}
				newTag.Key = "addKeyT1"
				newTag.Value = "addValueT1"
				result, err := handler.AddTag(resourceType, resourceIID, newTag)
				if err != nil {
					cblogger.Info("Fail to add Tag : ", err)
				} else {
					cblogger.Info("Result of add Tag")
					//spew.Dump(result)
					cblogger.Info(result)
				}
			case 5:
				tagName := "addKeyT1"
				result, err := handler.RemoveTag(resourceType, resourceIID, tagName)
				if err != nil {
					cblogger.Info("Fail to remove Tag : ", err)
				} else {
					cblogger.Info("Result of remove Tag")
					//spew.Dump(result)
					cblogger.Info(result)
				}
			}
		}
	}
}

func handleDisk() {
	cblogger.Debug("Start handleDisk Test")
	ResourceHandler, err := testconf.GetResourceHandler("Disk")
	if err != nil {
		//panic(err)
		cblogger.Error(err)
	}
	handler := ResourceHandler.(irs.DiskHandler)
	cblogger.Info(handler)

	tag1 := irs.KeyValue{Key: "daa", Value: "dbb"}
	diskReqInfo := irs.DiskInfo{
		// IID: irs.IID{},
		TagList: []irs.KeyValue{tag1},
	}

	for {
		fmt.Println("Handler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. Disk List")
		fmt.Println("2. GetDisk ")
		fmt.Println("3. CreateDisk ")
		fmt.Println("4. AddDisk ")
		fmt.Println("5. RemoveDisk ")
		fmt.Println("6. List IID ")

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}
		// resourceType := irs.RSType("disk")
		resourceIID := irs.IID{NameId: "", SystemId: ""}
		// resourceType := irs.RSType("CLUSTER")
		// resourceIID := irs.IID{NameId: "cs-issue-test", SystemId: ""}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return
			case 1:

				result, err := handler.ListDisk()
				if err != nil {
					cblogger.Infof("Fail to retrieve Disk list : ", err)
				} else {
					cblogger.Info("Result of Disk list")
					spew.Dump(result)
				}

			case 2:
				// tagName := "tagntest3"
				// tagName = ""
				result, err := handler.GetDisk(resourceIID)
				if err != nil {
					cblogger.Info("Fail to get Disk : ", err)
				} else {
					cblogger.Info("Result of get Disk")
					//spew.Dump(result)
					cblogger.Info(result)
				}
			case 3:
				// tagName := "tagntest3"
				// tagName = ""
				result, err := handler.CreateDisk(diskReqInfo)
				if err != nil {
					cblogger.Info("Fail to create Disk : ", err)
				} else {
					cblogger.Info("Result of create Disk")
					//spew.Dump(result)
					cblogger.Info(result)
				}
			case 6:
				cblogger.Infof("List IID test")

				result, err := handler.ListIID()
				if err != nil {
					cblogger.Infof("Fail to List IID : ", err)
				} else {
					cblogger.Infof("[%s] Result of IID List")
					for _, v := range result {
						cblogger.Infof("IID List: %+v", v)
					}
				}
			}
		}
	}
}

func handleMyImage() {
	cblogger.Debug("Start handleMyImage Test")
	ResourceHandler, err := testconf.GetResourceHandler("MyImage")
	if err != nil {
		//panic(err)
		cblogger.Error(err)
	}
	handler := ResourceHandler.(irs.MyImageHandler)
	cblogger.Info(handler)

	tag1 := irs.KeyValue{Key: "daa", Value: "dbb"}
	snapshotReqInfo := irs.MyImageInfo{
		SourceVM: irs.IID{SystemId: "i-j6camjyolcjjhbs3ack7"},
		IId:      irs.IID{NameId: "tagtestimage2"},
		TagList:  []irs.KeyValue{tag1},
	}

	for {
		fmt.Println("Handler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. MyImage List")
		fmt.Println("2. GetMyImage ")
		fmt.Println("3. CreateMyImage ")
		fmt.Println("4. List IID ")

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}
		// resourceType := irs.RSType("disk")
		resourceIID := irs.IID{NameId: "", SystemId: "i-j6cd7rv72uwjyx8qy304"}
		// resourceType := irs.RSType("CLUSTER")
		// resourceIID := irs.IID{NameId: "cs-issue-test", SystemId: ""}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return
			case 1:

				result, err := handler.ListMyImage()
				if err != nil {
					cblogger.Infof("Failed to retrieve MyImage list : ", err)
				} else {
					cblogger.Info("Result of MyImage list retrieval")
					spew.Dump(result)
				}

			case 2:
				// tagName := "tagntest3"
				// tagName = ""
				result, err := handler.GetMyImage(resourceIID)
				if err != nil {
					cblogger.Info("Failed to retrieve MyImage : ", err)
				} else {
					cblogger.Info("Result of GetMyImage retrieval")
					//spew.Dump(result)
					cblogger.Info(result)
				}
			case 3:
				// tagName := "tagntest3"
				// tagName = ""
				result, err := handler.SnapshotVM(snapshotReqInfo)
				if err != nil {
					cblogger.Info("Failed to create MyImage : ", err)
				} else {
					cblogger.Info("Result of CreateMyImage")
					//spew.Dump(result)
					cblogger.Info(result)
				}
				// case 4:
				// 	newTag := irs.KeyValue{}
				// 	newTag.Key = "addKeyT1"
				// 	newTag.Value = "addValueT1"
				// 	result, err := handler.AddTag(resourceType, resourceIID, newTag)
				// 	if err != nil {
				// 		cblogger.Info("Failed to retrieve Tag : ", err)
				// 	} else {
				// 		cblogger.Info("Result of AddTag")
				// 		//spew.Dump(result)
				// 		cblogger.Info(result)
				// 	}
				// case 5:
				// 	tagName := "addKeyT1"
				// 	result, err := handler.RemoveTag(resourceType, resourceIID, tagName)
				// 	if err != nil {
				// 		cblogger.Info("Failed to retrieve Tag : ", err)
				// 	} else {
				// 		cblogger.Info("Result of RemoveTag")
				// 		//spew.Dump(result)
				// 		cblogger.Info(result)
				// 	}
			case 4:
				cblogger.Infof("List IID test")

				result, err := handler.ListIID()
				if err != nil {
					cblogger.Infof("Fail to List IID : ", err)
				} else {
					cblogger.Infof("[%s] Result of IID List")
					for _, v := range result {
						cblogger.Infof("IID List: %+v", v)
					}
				}
			}
		}
	}
}

func handleCluster() {
	cblogger.Debug("Start Cluster Test")
	ResourceHandler, err := testconf.GetResourceHandler("Cluster")
	if err != nil {
		//panic(err)
		cblogger.Error(err)
	}
	handler := ResourceHandler.(irs.ClusterHandler)
	cblogger.Info(handler)

	vpcID := "vpc-j6ctiyol6osnuy9sfj2xm"
	vswitchID := "vsw-j6cgu2z3paapkojcj918q"
	securityGroupIIDs := "sg-j6cauy6gbpy261k9k74c"

	tag1 := irs.KeyValue{Key: "caa", Value: "cbb"}

	clusterReqInfo := irs.ClusterInfo{
		IId: irs.IID{NameId: "tagtestcluster22"},
		Network: irs.NetworkInfo{
			VpcIID: irs.IID{SystemId: vpcID},
			SubnetIIDs: []irs.IID{
				{SystemId: vswitchID},
			},
			SecurityGroupIIDs: []irs.IID{
				{SystemId: securityGroupIIDs},
			},
			// Optionally, add other network-related fields if necessary
		},
		TagList: []irs.KeyValue{tag1},
	}

	for {
		fmt.Println("Handler Management")
		fmt.Println("0. Quit")
		fmt.Println("1. Cluster List")
		fmt.Println("2. GetCluster ")
		fmt.Println("3. CreateCluster")

		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			panic(err)
		}
		// resourceType := irs.RSType("disk")
		resourceIID := irs.IID{NameId: "", SystemId: "i-j6cd7rv72uwjyx8qy304"}
		// resourceType := irs.RSType("CLUSTER")
		// resourceIID := irs.IID{NameId: "cs-issue-test", SystemId: ""}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				return
			case 1:

				result, err := handler.ListCluster()
				if err != nil {
					cblogger.Infof("Failed to retrieve Cluster list : ", err)
				} else {
					cblogger.Info("Result of Cluster list retrieval")
					spew.Dump(result)
				}
			case 2:
				// tagName := "tagntest3"
				// tagName = ""
				result, err := handler.GetCluster(resourceIID)
				if err != nil {
					cblogger.Info("Failed to retrieve Cluster : ", err)
				} else {
					cblogger.Info("Result of GetCluster retrieval")
					//spew.Dump(result)
					cblogger.Info(result)
				}
			case 3:
				// tagName := "tagntest3"
				// tagName = ""
				result, err := handler.CreateCluster(clusterReqInfo)
				if err != nil {
					cblogger.Info("Failed to create Cluster : ", err)
				} else {
					cblogger.Info("Result of CreateCluster")
					//spew.Dump(result)
					cblogger.Info(result)
				}
				// case 4:
				// 	newTag := irs.KeyValue{}
				// 	newTag.Key = "addKeyT1"
				// 	newTag.Value = "addValueT1"
				// 	result, err := handler.AddTag(resourceType, resourceIID, newTag)
				// 	if err != nil {
				// 		cblogger.Info("Failed to retrieve Tag : ", err)
				// 	} else {
				// 		cblogger.Info("Result of AddTag")
				// 		//spew.Dump(result)
				// 		cblogger.Info(result)
				// 	}
				// case 5:
				// 	tagName := "addKeyT1"
				// 	result, err := handler.RemoveTag(resourceType, resourceIID, tagName)
				// 	if err != nil {
				// 		cblogger.Info("Failed to retrieve Tag : ", err)
				// 	} else {
				// 		cblogger.Info("Result of RemoveTag")
				// 		//spew.Dump(result)
				// 		cblogger.Info(result)
				// 	}
			}
		}
	}
}

func handleFileSystem() {
	cblogger.Debug("Start FileSystem Resource Test")
	ResourceHandler, err := testconf.GetResourceHandler("FileSystem")
	if err != nil {
		cblogger.Error(err)
		return
	}
	handler := ResourceHandler.(irs.FileSystemHandler)
	cblogger.Info(handler)

	tag1 := irs.KeyValue{Key: "fsaa", Value: "fsbb"}
	subnetIID := irs.IID{SystemId: "vsw-6wemkeefn461cmrqi5fbl"}

	// Basic setup - minimum required information
	fileSystemReqInfoBasic := irs.FileSystemInfo{
		IId:    irs.IID{NameId: "cb-test-nas-basic"},
		VpcIID: irs.IID{SystemId: "vpc-6weua85b8bmwduzec8bvj"}, // VPC ID required
		Zone:   "ap-northeast-1a",                              // Zone ID required (e.g., "cn-hongkong-b")
		AccessSubnetList: []irs.IID{
			{SystemId: "vsw-6wemkeefn461cmrqi5fbl"}, // Subnet ID required
		},
		TagList: []irs.KeyValue{tag1},
		PerformanceInfo: map[string]string{
			"StorageType": "Capacity", // Required: StorageType must be specified
		},
	}

	// Advanced setup - with custom performance options (Capacity)
	fileSystemReqInfoAdvanced := irs.FileSystemInfo{
		IId:    irs.IID{NameId: "cb-test-nas-advanced"},
		VpcIID: irs.IID{SystemId: "vpc-6weua85b8bmwduzec8bvj"}, // VPC ID required
		Zone:   "ap-northeast-1a",                              // Zone ID required
		AccessSubnetList: []irs.IID{
			{SystemId: "vsw-6wemkeefn461cmrqi5fbl"}, // Subnet ID required
		},
		TagList: []irs.KeyValue{tag1},
		PerformanceInfo: map[string]string{
			"StorageType":  "Capacity",
			"ProtocolType": "NFS",
			"Capacity":     "1024",
		},
	}

	// Premium storage type example
	fileSystemReqInfoPremium := irs.FileSystemInfo{
		IId:    irs.IID{NameId: "cb-test-nas-premium"},
		VpcIID: irs.IID{SystemId: "vpc-6weua85b8bmwduzec8bvj"}, // VPC ID required
		Zone:   "ap-northeast-1a",                              // Zone ID required
		AccessSubnetList: []irs.IID{
			{SystemId: "vsw-6wemkeefn461cmrqi5fbl"}, // Subnet ID required
		},
		TagList: []irs.KeyValue{tag1},
		PerformanceInfo: map[string]string{
			"StorageType":  "Premium",
			"ProtocolType": "NFS",
			"Capacity":     "2048",
		},
	}

	// Performance storage type example
	fileSystemReqInfoPerformance := irs.FileSystemInfo{
		IId:    irs.IID{NameId: "cb-test-nas-performance"},
		VpcIID: irs.IID{SystemId: "vpc-6weua85b8bmwduzec8bvj"}, // VPC ID required
		Zone:   "ap-northeast-1a",                              // Zone ID required
		AccessSubnetList: []irs.IID{
			{SystemId: "vsw-6wemkeefn461cmrqi5fbl"}, // Subnet ID required
		},
		TagList: []irs.KeyValue{tag1},
		PerformanceInfo: map[string]string{
			"StorageType":  "Performance",
			"ProtocolType": "NFS",
			"Capacity":     "2048",
		},
	}

	reqFileSystemId := irs.IID{SystemId: ""}

	for {
		fmt.Println("FileSystem Management")
		fmt.Println("0. Quit")
		fmt.Println("1. GetMetaInfo")
		fmt.Println("2. FileSystem List")
		fmt.Println("3. FileSystem Create (Basic Setup)")
		fmt.Println("4. FileSystem Create (Advanced Setup - Capacity)")
		fmt.Println("5. FileSystem Create (Premium Storage)")
		fmt.Println("6. FileSystem Create (Performance Storage)")
		fmt.Println("7. FileSystem Get")
		fmt.Println("8. FileSystem Delete")
		fmt.Println("9. Add Access Subnet")
		fmt.Println("10. Remove Access Subnet")
		fmt.Println("11. List Access Subnet")
		fmt.Println("12. List IID")

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
				cblogger.Info("Start GetMetaInfo() ...")
				result, err := handler.GetMetaInfo()
				if err != nil {
					cblogger.Error("Failed to get meta info: ", err)
				} else {
					cblogger.Info("Result of GetMetaInfo()")
					spew.Dump(result)
				}

			case 2:
				cblogger.Info("Start ListFileSystem() ...")
				result, err := handler.ListFileSystem()
				if err != nil {
					cblogger.Error("Failed to retrieve FileSystem list: ", err)
				} else {
					cblogger.Info("Result of FileSystem list")
					spew.Dump(result)
					if result != nil && len(result) > 0 {
						reqFileSystemId = result[0].IId
					}
				}

			case 3:
				cblogger.Infof("[%s] Test of create FileSystem (Basic Setup)", fileSystemReqInfoBasic.IId.NameId)
				result, err := handler.CreateFileSystem(fileSystemReqInfoBasic)
				if err != nil {
					cblogger.Error("Failed to create FileSystem: ", err)
				} else {
					cblogger.Info("Result of create FileSystem (Basic Setup)")
					reqFileSystemId = result.IId
					spew.Dump(result)
				}

			case 4:
				cblogger.Infof("[%s] Test of create FileSystem (Advanced Setup - Capacity)", fileSystemReqInfoAdvanced.IId.NameId)
				result, err := handler.CreateFileSystem(fileSystemReqInfoAdvanced)
				if err != nil {
					cblogger.Error("Failed to create FileSystem: ", err)
				} else {
					cblogger.Info("Result of create FileSystem (Advanced Setup - Capacity)")
					reqFileSystemId = result.IId
					spew.Dump(result)
				}

			case 5:
				cblogger.Infof("[%s] Test of create FileSystem (Premium Storage)", fileSystemReqInfoPremium.IId.NameId)
				result, err := handler.CreateFileSystem(fileSystemReqInfoPremium)
				if err != nil {
					cblogger.Error("Failed to create FileSystem: ", err)
				} else {
					cblogger.Info("Result of create FileSystem (Premium Storage)")
					reqFileSystemId = result.IId
					spew.Dump(result)
				}

			case 6:
				cblogger.Infof("[%s] Test of create FileSystem (Performance Storage)", fileSystemReqInfoPerformance.IId.NameId)
				result, err := handler.CreateFileSystem(fileSystemReqInfoPerformance)
				if err != nil {
					cblogger.Error("Failed to create FileSystem: ", err)
				} else {
					cblogger.Info("Result of create FileSystem (Performance Storage)")
					reqFileSystemId = result.IId
					spew.Dump(result)
				}

			case 7:
				cblogger.Infof("[%s] Test of retrieve FileSystem", reqFileSystemId)
				result, err := handler.GetFileSystem(reqFileSystemId)
				if err != nil {
					cblogger.Error("Failed to retrieve FileSystem: ", err)
				} else {
					cblogger.Info("Result of retrieve FileSystem")
					spew.Dump(result)
				}

			case 8:
				cblogger.Infof("[%s] Test of delete FileSystem", reqFileSystemId)
				result, err := handler.DeleteFileSystem(reqFileSystemId)
				if err != nil {
					cblogger.Error("Failed to delete FileSystem: ", err)
				} else {
					cblogger.Info("Result of delete FileSystem: ", result)
				}

			case 9:
				cblogger.Infof("[%s] Test of add access subnet", reqFileSystemId)
				result, err := handler.AddAccessSubnet(reqFileSystemId, subnetIID)
				if err != nil {
					cblogger.Error("Failed to add access subnet: ", err)
				} else {
					cblogger.Info("Result of add access subnet")
					spew.Dump(result)
				}

			case 10:
				cblogger.Infof("[%s] Test of remove access subnet", reqFileSystemId)
				result, err := handler.RemoveAccessSubnet(reqFileSystemId, subnetIID)
				if err != nil {
					cblogger.Error("Failed to remove access subnet: ", err)
				} else {
					cblogger.Info("Result of remove access subnet: ", result)
				}

			case 11:
				cblogger.Infof("[%s] Test of list access subnet", reqFileSystemId)
				result, err := handler.ListAccessSubnet(reqFileSystemId)
				if err != nil {
					cblogger.Error("Failed to list access subnet: ", err)
				} else {
					cblogger.Info("Result of list access subnet")
					spew.Dump(result)
				}

			case 12:
				cblogger.Info("Start ListIID() ...")
				result, err := handler.ListIID()
				if err != nil {
					cblogger.Error("Failed to retrieve FileSystem IID list: ", err)
				} else {
					cblogger.Info("Result of FileSystem IID list")
					spew.Dump(result)
				}

			default:
				fmt.Println("Unknown command")
			}
		}
	}
}

func main() {
	cblogger.Info("Alibaba Cloud Resource Test")
	cblogger.Debug("Debug mode")

	// handleVPC() //VPC
	//handleVMSpec()
	// handleImage() //AMI
	// handleSecurity()
	// handleKeyPair()
	// handleVM()
	// handleNLB()
	// handleDisk()
	// handleMyImage()
	// handleCluster()
	handleFileSystem() // FileSystem (NAS)
	//handlePublicIP()

	//handleVNic() //Lancard
	//handleRegionZone()
	//handlePriceInfo()
	// handleTagInfo()
	/*
	   //StartTime := "2020-05-07T01:35:00Z"
	   StartTime := "2020-05-07T01:35Z"
	   timeLen := len(StartTime)
	   cblogger.Infof("======> create time length [%s]", timeLen)
	   if timeLen > 7 {
	      cblogger.Infof("======> create time last string [%s]", StartTime[timeLen-1:])
	      if StartTime[timeLen-1:] == "Z" {
	         cblogger.Infof("======> change string : [%s]", StartTime[:timeLen-1])
	         NewStartTime := StartTime[:timeLen-1] + ":00Z"
	         cblogger.Infof("======> return result string : [%s]", NewStartTime)
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
