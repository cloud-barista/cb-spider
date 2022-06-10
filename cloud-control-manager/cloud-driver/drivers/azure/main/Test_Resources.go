package main

import (
	"errors"
	"fmt"
	cblog "github.com/cloud-barista/cb-log"
	azdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/azure"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
)

type Config struct {
	Azure struct {
		ClientId       string `yaml:"client_id"`
		ClientSecret   string `yaml:"client_secret"`
		TenantId       string `yaml:"tenant_id"`
		SubscriptionID string `yaml:"subscription_id"`

		GroupName string `yaml:"group_name"`
		Location  string `yaml:"location"`
		Resources struct {
			Image struct {
				NameId string `yaml:"nameId"`
			} `yaml:"image"`
			Security struct {
				NameId string `yaml:"nameId"`
				VpcIID struct {
					NameId string `yaml:"nameId"`
				} `yaml:"VpcIID"`
				Rules []struct {
					FromPort   string `yaml:"FromPort"`
					ToPort     string `yaml:"ToPort"`
					IPProtocol string `yaml:"IPProtocol"`
					CIDR       string `yaml:"CIDR"`
					Direction  string `yaml:"Direction"`
				} `yaml:"rules"`
				AddRules []struct {
					FromPort   string `yaml:"FromPort"`
					ToPort     string `yaml:"ToPort"`
					IPProtocol string `yaml:"IPProtocol"`
					CIDR       string `yaml:"CIDR"`
					Direction  string `yaml:"Direction"`
				} `yaml:"addRules"`
				RemoveRules []struct {
					FromPort   string `yaml:"FromPort"`
					ToPort     string `yaml:"ToPort"`
					IPProtocol string `yaml:"IPProtocol"`
					CIDR       string `yaml:"CIDR"`
					Direction  string `yaml:"Direction"`
				} `yaml:"removeRules"`
			} `yaml:"security"`
			KeyPair struct {
				NameId string `yaml:"nameId"`
			} `yaml:"keyPair"`
			VmSpec struct {
				NameId string `yaml:"nameId"`
			} `yaml:"vmSpec"`
			VPC struct {
				NameId   string `yaml:"nameId"`
				IPv4CIDR string `yaml:"ipv4CIDR"`
				Subnets  []struct {
					NameId   string `yaml:"nameId"`
					IPv4CIDR string `yaml:"ipv4CIDR"`
				} `yaml:"subnets"`
				AddSubnet struct {
					NameId   string `yaml:"nameId"`
					IPv4CIDR string `yaml:"ipv4CIDR"`
				} `yaml:"addSubnet"`
			} `yaml:"vpc"`
			Vm struct {
				IID struct {
					NameId string `yaml:"nameId"`
				} `yaml:"IID"`
				ImageIID struct {
					NameId string `yaml:"nameId"`
				} `yaml:"ImageIID"`
				VmSpecName string `yaml:"VmSpecName"`
				KeyPairIID struct {
					NameId string `yaml:"nameId"`
				} `yaml:"KeyPairIID"`
				VpcIID struct {
					NameId string `yaml:"nameId"`
				} `yaml:"VpcIID"`
				SubnetIID struct {
					NameId string `yaml:"nameId"`
				} `yaml:"SubnetIID"`
				SecurityGroupIIDs []struct {
					NameId string `yaml:"nameId"`
				} `yaml:"SecurityGroupIIDs"`
				RootDiskSize string `yaml:"RootDiskSize"`
				RootDiskType string `yaml:"RootDiskType"`
			} `yaml:"vm"`
		} `yaml:"resources"`
	} `yaml:"azure"`
}

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

func readConfigFile() Config {
	// Set Environment Value of Project Root Path
	rootPath := os.Getenv("CBSPIDER_ROOT")
	data, err := ioutil.ReadFile(rootPath + "/cloud-control-manager/cloud-driver/drivers/azure/main/conf/config.yaml")

	if err != nil {
		cblogger.Error(err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		cblogger.Error(err)
	}

	return config
}

func showTestHandlerInfo() {
	cblogger.Info("==========================================================")
	cblogger.Info("[Test ResourceHandler]")
	cblogger.Info("1. ImageHandler")
	cblogger.Info("2. SecurityHandler")
	cblogger.Info("3. VPCHandler")
	cblogger.Info("4. KeyPairHandler")
	cblogger.Info("5. VmSpecHandler")
	cblogger.Info("6. VmHandler")
	cblogger.Info("7. Exit")
	cblogger.Info("==========================================================")
}

func getResourceHandler(resourceType string, config Config) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(azdrv.AzureDriver)
	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			ClientId:       config.Azure.ClientId,
			ClientSecret:   config.Azure.ClientSecret,
			TenantId:       config.Azure.TenantId,
			SubscriptionId: config.Azure.SubscriptionID,
		},
		RegionInfo: idrv.RegionInfo{
			Region:        config.Azure.Location,
			ResourceGroup: config.Azure.GroupName,
		},
	}

	cloudConnection, _ := cloudDriver.ConnectCloud(connectionInfo)
	var resourceHandler interface{}
	var err error

	switch resourceType {
	case "image":
		resourceHandler, err = cloudConnection.CreateImageHandler()
	case "security":
		resourceHandler, err = cloudConnection.CreateSecurityHandler()
	case "vpc":
		resourceHandler, err = cloudConnection.CreateVPCHandler()
	case "keypair":
		resourceHandler, err = cloudConnection.CreateKeyPairHandler()
	case "vmspec":
		resourceHandler, err = cloudConnection.CreateVMSpecHandler()
	case "vm":
		resourceHandler, err = cloudConnection.CreateVMHandler()
	}

	if err != nil {
		return nil, err
	}
	return resourceHandler, nil
}
func testImageHandlerListPrint() {
	cblogger.Info("Test ImageHandler")
	cblogger.Info("0. Print Menu")
	cblogger.Info("1. ListImage()")
	cblogger.Info("2. GetImage()")
	cblogger.Info("3. CreateImage()")
	cblogger.Info("4. DeleteImage()")
	cblogger.Info("5. Exit")
}
func testImageHandler(config Config) {
	resourceHandler, err := getResourceHandler("image", config)
	if err != nil {
		cblogger.Error(err)
		return
	}

	imageHandler := resourceHandler.(irs.ImageHandler)

	testImageHandlerListPrint()

	imageIID := irs.IID{NameId: config.Azure.Resources.Image.NameId}

Loop:
	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			cblogger.Error(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				testImageHandlerListPrint()
			case 1:
				cblogger.Info("Start ListImage() ...")
				if list, err := imageHandler.ListImage(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				cblogger.Info("Finish ListImage()")
			case 2:
				cblogger.Info("Start GetImage() ...")
				if imageInfo, err := imageHandler.GetImage(imageIID); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(imageInfo)
				}
				cblogger.Info("Finish GetImage()")
			case 3:
				cblogger.Info("Start CreateImage() ...")
				cblogger.Info("Finish CreateImage()")
			case 4:
				cblogger.Info("Start DeleteImage() ...")
				cblogger.Info("Finish DeleteImage()")
			case 5:
				cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

func testSecurityHandlerListPrint() {
	cblogger.Info("Test securityHandler")
	cblogger.Info("0. Print Menu")
	cblogger.Info("1. ListSecurity()")
	cblogger.Info("2. GetSecurity()")
	cblogger.Info("3. CreateSecurity()")
	cblogger.Info("4. DeleteSecurity()")
	cblogger.Info("5. AddRules()")
	cblogger.Info("6. RemoveRules()")
	cblogger.Info("7. Exit")
}

//SecurityGroup
func testSecurityHandler(config Config) {
	resourceHandler, err := getResourceHandler("security", config)
	if err != nil {
		cblogger.Error(err)
		return
	}

	securityHandler := resourceHandler.(irs.SecurityHandler)

	testSecurityHandlerListPrint()

	securityIId := irs.IID{NameId: config.Azure.Resources.Security.NameId}
	securityRules := config.Azure.Resources.Security.Rules
	var securityRulesInfos []irs.SecurityRuleInfo
	for _, securityRule := range securityRules {
		infos := irs.SecurityRuleInfo{
			FromPort:   securityRule.FromPort,
			ToPort:     securityRule.ToPort,
			IPProtocol: securityRule.IPProtocol,
			Direction:  securityRule.Direction,
			CIDR:       securityRule.CIDR,
		}
		securityRulesInfos = append(securityRulesInfos, infos)
	}
	targetVPCIId := irs.IID{
		NameId: config.Azure.Resources.Security.VpcIID.NameId,
	}
	securityAddRules := config.Azure.Resources.Security.AddRules
	var securityAddRulesInfos []irs.SecurityRuleInfo
	for _, securityRule := range securityAddRules {
		infos := irs.SecurityRuleInfo{
			FromPort:   securityRule.FromPort,
			ToPort:     securityRule.ToPort,
			IPProtocol: securityRule.IPProtocol,
			Direction:  securityRule.Direction,
			CIDR:       securityRule.CIDR,
		}
		securityAddRulesInfos = append(securityAddRulesInfos, infos)
	}
	securityRemoveRules := config.Azure.Resources.Security.RemoveRules
	var securityRemoveRulesInfos []irs.SecurityRuleInfo
	for _, securityRule := range securityRemoveRules {
		infos := irs.SecurityRuleInfo{
			FromPort:   securityRule.FromPort,
			ToPort:     securityRule.ToPort,
			IPProtocol: securityRule.IPProtocol,
			Direction:  securityRule.Direction,
			CIDR:       securityRule.CIDR,
		}
		securityRemoveRulesInfos = append(securityRemoveRulesInfos, infos)
	}
Loop:
	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			fmt.Println(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				testSecurityHandlerListPrint()
			case 1:
				fmt.Println("Start ListSecurity() ...")
				if securityList, err := securityHandler.ListSecurity(); err != nil {
					fmt.Println(err)
				} else {
					spew.Dump(securityList)
				}
				fmt.Println("Finish ListSecurity()")
			case 2:
				fmt.Println("Start GetSecurity() ...")
				if secGroupInfo, err := securityHandler.GetSecurity(securityIId); err != nil {
					fmt.Println(err)
				} else {
					spew.Dump(secGroupInfo)
				}
				fmt.Println("Finish GetSecurity()")
			case 3:
				fmt.Println("Start CreateSecurity() ...")
				reqInfo := irs.SecurityReqInfo{
					IId:           securityIId,
					SecurityRules: &securityRulesInfos,
					VpcIID:        targetVPCIId,
				}
				security, err := securityHandler.CreateSecurity(reqInfo)
				if err != nil {
					fmt.Println(err)
				} else {
					spew.Dump(security)
				}
				//securityGroupId = security.IId
				fmt.Println("Finish CreateSecurity()")
			case 4:
				fmt.Println("Start DeleteSecurity() ...")
				if ok, err := securityHandler.DeleteSecurity(securityIId); !ok {
					fmt.Println(err)
				}
				fmt.Println("Finish DeleteSecurity()")
			case 5:
				fmt.Println("Start AddRules() ...")
				security, err := securityHandler.AddRules(securityIId, &securityAddRulesInfos)
				if err != nil {
					fmt.Println(err)
				} else {
					spew.Dump(security)
				}
				fmt.Println("Finish AddRules()")
			case 6:
				fmt.Println("Start RemoveRules() ...")
				if ok, err := securityHandler.RemoveRules(securityIId, &securityRemoveRulesInfos); !ok {
					fmt.Println(err)
				}
				fmt.Println("Finish RemoveRules()")
			case 7:
				fmt.Println("Exit")
				break Loop
			}
		}
	}
}

func testVPCHandlerListPrint() {
	cblogger.Info("Test VPCHandler")
	cblogger.Info("0. Print Menu")
	cblogger.Info("1. ListVPC()")
	cblogger.Info("2. GetVPC()")
	cblogger.Info("3. CreateVPC()")
	cblogger.Info("4. DeleteVPC()")
	cblogger.Info("5. AddSubnet()")
	cblogger.Info("6. RemoveSubnet()")
	cblogger.Info("7. Exit")
}

func testVPCHandler(config Config) {
	resourceHandler, err := getResourceHandler("vpc", config)
	if err != nil {
		cblogger.Error(err)
		return
	}

	vpcHandler := resourceHandler.(irs.VPCHandler)

	testVPCHandlerListPrint()

	vpcIID := irs.IID{NameId: config.Azure.Resources.VPC.NameId}

	subnetLists := config.Azure.Resources.VPC.Subnets
	var subnetInfoList []irs.SubnetInfo
	for _, sb := range subnetLists {
		info := irs.SubnetInfo{
			IId: irs.IID{
				NameId: sb.NameId,
			},
			IPv4_CIDR: sb.IPv4CIDR,
		}
		subnetInfoList = append(subnetInfoList, info)
	}

	VPCReqInfo := irs.VPCReqInfo{
		IId:            vpcIID,
		IPv4_CIDR:      config.Azure.Resources.VPC.IPv4CIDR,
		SubnetInfoList: subnetInfoList,
	}
	addSubnet := config.Azure.Resources.VPC.AddSubnet
	addSubnetInfo := irs.SubnetInfo{
		IId: irs.IID{
			NameId: addSubnet.NameId,
		},
		IPv4_CIDR: addSubnet.IPv4CIDR,
	}
	//deleteVpcid := irs.IID{
	//	NameId: "bcr02a.tok02.774",
	//}
Loop:

	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			cblogger.Error(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				testVPCHandlerListPrint()
			case 1:
				cblogger.Info("Start ListVPC() ...")
				if list, err := vpcHandler.ListVPC(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				cblogger.Info("Finish ListVPC()")
			case 2:
				cblogger.Info("Start GetVPC() ...")
				if vpcInfo, err := vpcHandler.GetVPC(vpcIID); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vpcInfo)
				}
				cblogger.Info("Finish GetVPC()")
			case 3:
				cblogger.Info("Start CreateVPC() ...")
				if vpcInfo, err := vpcHandler.CreateVPC(VPCReqInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vpcInfo)
				}
				cblogger.Info("Finish CreateVPC()")
			case 4:
				cblogger.Info("Start DeleteVPC() ...")
				if result, err := vpcHandler.DeleteVPC(vpcIID); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(result)
				}
				cblogger.Info("Finish DeleteVPC()")
			case 5:
				cblogger.Info("Start AddSubnet() ...")
				if vpcInfo, err := vpcHandler.AddSubnet(vpcIID, addSubnetInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vpcInfo)
				}
				cblogger.Info("Finish AddSubnet()")
			case 6:
				cblogger.Info("Start RemoveSubnet() ...")
				vpcInfo, err := vpcHandler.GetVPC(vpcIID)
				if err != nil {
					cblogger.Error(err)
				}
				if vpcInfo.SubnetInfoList != nil && len(vpcInfo.SubnetInfoList) > 0 {
					firstSubnet := vpcInfo.SubnetInfoList[0]
					cblogger.Info(fmt.Sprintf("RemoveSubnet : %s %s", firstSubnet.IId.NameId, firstSubnet.IPv4_CIDR))
					result, err := vpcHandler.RemoveSubnet(vpcIID, firstSubnet.IId)
					if err != nil {
						cblogger.Error(err)
					} else {
						spew.Dump(result)
					}
				} else {
					err = errors.New("not exist subnet")
					cblogger.Error(err)
				}
				cblogger.Info("Finish RemoveSubnet()")
			case 7:
				cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

func testKeyPairHandlerListPrint() {
	cblogger.Info("Test KeyPairHandler")
	cblogger.Info("0. Print Menu")
	cblogger.Info("1. ListKey()")
	cblogger.Info("2. GetKey()")
	cblogger.Info("3. CreateKey()")
	cblogger.Info("4. DeleteKey()")
	cblogger.Info("5. Exit")
}
func testKeyPairHandler(config Config) {
	resourceHandler, err := getResourceHandler("keypair", config)
	if err != nil {
		cblogger.Error(err)
		return
	}

	keyPairHandler := resourceHandler.(irs.KeyPairHandler)

	testKeyPairHandlerListPrint()

	keypairIId := irs.IID{
		NameId: config.Azure.Resources.KeyPair.NameId,
	}

Loop:
	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			cblogger.Error(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				testKeyPairHandlerListPrint()
			case 1:
				cblogger.Info("Start ListKey() ...")
				if keyPairList, err := keyPairHandler.ListKey(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(keyPairList)
				}
				cblogger.Info("Finish ListKey()")
			case 2:
				cblogger.Info("Start GetKey() ...")
				if keyPairInfo, err := keyPairHandler.GetKey(keypairIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(keyPairInfo)
				}
				cblogger.Info("Finish GetKey()")
			case 3:
				cblogger.Info("Start CreateKey() ...")
				reqInfo := irs.KeyPairReqInfo{
					IId: keypairIId,
				}
				if keyInfo, err := keyPairHandler.CreateKey(reqInfo); err != nil {
					cblogger.Error(err)
				} else {
					keypairIId = keyInfo.IId
					spew.Dump(keyInfo)
				}
				cblogger.Info("Finish CreateKey()")
			case 4:
				cblogger.Info("Start DeleteKey() ...")
				if ok, err := keyPairHandler.DeleteKey(keypairIId); !ok {
					cblogger.Error(err)
				}
				cblogger.Info("Finish DeleteKey()")
			case 5:
				cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

func testVMSpecHandlerListPrint() {
	cblogger.Info("Test VMSpecHandler")
	cblogger.Info("0. Print Menu")
	cblogger.Info("1. ListVMSpec()")
	cblogger.Info("2. GetVMSpec()")
	cblogger.Info("3. ListOrgVMSpec()")
	cblogger.Info("4. GetOrgVMSpec()")
	cblogger.Info("5. Exit")
}

func testVMSpecHandler(config Config) {
	resourceHandler, err := getResourceHandler("vmspec", config)
	if err != nil {
		cblogger.Error(err)
		return
	}

	vmSpecHandler := resourceHandler.(irs.VMSpecHandler)

	testVMSpecHandlerListPrint()

Loop:
	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			cblogger.Error(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				testVMSpecHandlerListPrint()
			case 1:
				cblogger.Info("Start ListVMSpec() ...")
				if list, err := vmSpecHandler.ListVMSpec(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				cblogger.Info("Finish ListVMSpec()")
			case 2:
				cblogger.Info("Start GetVMSpec() ...")
				if vmSpecInfo, err := vmSpecHandler.GetVMSpec(config.Azure.Resources.VmSpec.NameId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vmSpecInfo)
				}
				cblogger.Info("Finish GetVMSpec()")
			case 3:
				cblogger.Info("Start ListOrgVMSpec() ...")
				if listStr, err := vmSpecHandler.ListOrgVMSpec(); err != nil {
					cblogger.Error(err)
				} else {
					fmt.Println(listStr)
				}
				cblogger.Info("Finish ListOrgVMSpec()")
			case 4:
				cblogger.Info("Start GetOrgVMSpec() ...")
				if vmSpecStr, err := vmSpecHandler.GetOrgVMSpec(config.Azure.Resources.VmSpec.NameId); err != nil {
					cblogger.Error(err)
				} else {
					fmt.Println(vmSpecStr)
				}
				cblogger.Info("Finish GetOrgVMSpec()")
			case 5:
				cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

func testVMHandlerListPrint() {
	cblogger.Info("Test VMSpecHandler")
	cblogger.Info("0. Print Menu")
	cblogger.Info("1. ListVM()")
	cblogger.Info("2. GetVM()")
	cblogger.Info("3. ListVMStatus()")
	cblogger.Info("4. GetVMStatus()")
	cblogger.Info("5. StartVM()")
	cblogger.Info("6. RebootVM()")
	cblogger.Info("7. SuspendVM()")
	cblogger.Info("8. ResumeVM()")
	cblogger.Info("9. TerminateVM()")
	cblogger.Info("10. Exit")
}

func testVMHandler(config Config) {
	resourceHandler, err := getResourceHandler("vm", config)
	if err != nil {
		cblogger.Error(err)
		return
	}

	vmHandler := resourceHandler.(irs.VMHandler)

	testVMHandlerListPrint()

	configsgIIDs := config.Azure.Resources.Vm.SecurityGroupIIDs
	var SecurityGroupIIDs []irs.IID
	for _, sg := range configsgIIDs {
		SecurityGroupIIDs = append(SecurityGroupIIDs, irs.IID{NameId: sg.NameId})
	}
	vmIID := irs.IID{
		NameId: config.Azure.Resources.Vm.IID.NameId,
	}
	vmReqInfo := irs.VMReqInfo{
		IId: irs.IID{
			NameId: config.Azure.Resources.Vm.IID.NameId,
		},
		ImageIID: irs.IID{
			NameId: config.Azure.Resources.Vm.ImageIID.NameId,
		},
		VpcIID: irs.IID{
			NameId: config.Azure.Resources.Vm.VpcIID.NameId,
		},
		SubnetIID: irs.IID{
			NameId: config.Azure.Resources.Vm.SubnetIID.NameId,
		},
		VMSpecName: config.Azure.Resources.Vm.VmSpecName,
		KeyPairIID: irs.IID{
			NameId: config.Azure.Resources.Vm.KeyPairIID.NameId,
		},
		RootDiskSize:      config.Azure.Resources.Vm.RootDiskSize,
		RootDiskType:      config.Azure.Resources.Vm.RootDiskType,
		SecurityGroupIIDs: SecurityGroupIIDs,
	}

Loop:
	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			cblogger.Error(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				testVMHandlerListPrint()
			case 1:
				cblogger.Info("Start ListVM() ...")
				if list, err := vmHandler.ListVM(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				cblogger.Info("Finish ListVM()")
			case 2:
				cblogger.Info("Start GetVM() ...")
				if vm, err := vmHandler.GetVM(vmIID); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vm)
				}
				cblogger.Info("Finish GetVM()")
			case 3:
				cblogger.Info("Start ListVMStatus() ...")
				if statusList, err := vmHandler.ListVMStatus(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(statusList)
				}
				cblogger.Info("Finish ListVMStatus()")
			case 4:
				cblogger.Info("Start GetVMStatus() ...")
				if vmStatus, err := vmHandler.GetVMStatus(vmIID); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vmStatus)
				}
				cblogger.Info("Finish GetVMStatus()")
			case 5:
				cblogger.Info("Start StartVM() ...")
				if vm, err := vmHandler.StartVM(vmReqInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vm)
				}
				cblogger.Info("Finish StartVM()")
			case 6:
				cblogger.Info("Start RebootVM() ...")
				if vmStatus, err := vmHandler.RebootVM(vmIID); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vmStatus)
				}
				cblogger.Info("Finish RebootVM()")
			case 7:
				cblogger.Info("Start SuspendVM() ...")
				if vmStatus, err := vmHandler.SuspendVM(vmIID); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vmStatus)
				}
				cblogger.Info("Finish SuspendVM()")
			case 8:
				cblogger.Info("Start ResumeVM() ...")
				if vmStatus, err := vmHandler.ResumeVM(vmIID); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vmStatus)
				}
				cblogger.Info("Finish ResumeVM()")
			case 9:
				cblogger.Info("Start TerminateVM() ...")
				if vmStatus, err := vmHandler.TerminateVM(vmIID); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vmStatus)
				}
				cblogger.Info("Finish TerminateVM()")
			case 10:
				cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

func main() {
	showTestHandlerInfo()
	config := readConfigFile()
Loop:
	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			cblogger.Error(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				testImageHandler(config)
				showTestHandlerInfo()
			case 2:
				testSecurityHandler(config)
				showTestHandlerInfo()
			case 3:
				testVPCHandler(config)
				showTestHandlerInfo()
			case 4:
				testKeyPairHandler(config)
				showTestHandlerInfo()
			case 5:
				testVMSpecHandler(config)
				showTestHandlerInfo()
			case 6:
				testVMHandler(config)
				showTestHandlerInfo()
			case 7:
				cblogger.Info("Exit Test ResourceHandler Program")
				break Loop
			}
		}
	}

}
