package main

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	cblog "github.com/cloud-barista/cb-log"
	azdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/azure"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Azure struct {
		ClientId       string `yaml:"client_id"`
		ClientSecret   string `yaml:"client_secret"`
		TenantId       string `yaml:"tenant_id"`
		SubscriptionID string `yaml:"subscription_id"`

		Location  string `yaml:"location"`
		Zone      string `yaml:"zone"`
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
				ImageType  string `yaml:"ImageType"`
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
				VMUserId     string `yaml:"VMUserId"`
				VMUserPasswd string `yaml:"VMUserPasswd"`
			} `yaml:"vm"`
			MyImage struct {
				IID struct {
					NameId string `yaml:"nameId"`
				} `yaml:"IID"`
				SourceVM struct {
					NameId string `yaml:"nameId"`
				} `yaml:"sourceVM"`
			} `yaml:"myImage"`
			Disk struct {
				IID struct {
					NameId string `yaml:"nameId"`
				} `yaml:"IID"`
				DiskType       string `yaml:"diskType"`
				DiskSize       string `yaml:"diskSize"`
				UpdateDiskSize string `yaml:"updateDiskSize"`
				AttachedVM     struct {
					NameId string `yaml:"nameId"`
				} `yaml:"attachedVM"`
			} `yaml:"disk"`
			File struct {
				IID struct {
					NameId string `yaml:"nameId"`
				} `yaml:"IID"`
				VpcIID struct {
					NameId string `yaml:"nameId"`
				} `yaml:"VpcIID"`
				AccessSubnetIIDs []struct {
					NameId string `yaml:"nameId"`
				} `yaml:"AccessSubnetIIDs"`
			} `yaml:"file"`
		} `yaml:"resources"`
	} `yaml:"azure"`
}

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
	cblog.SetLevel("info")
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
	cblogger.Info("7. NLBHandler")
	cblogger.Info("8. DiskHandler")
	cblogger.Info("9. MyImageHandler")
	cblogger.Info("10. RegionZoneHandler")
	cblogger.Info("11. PriceInfoHandler")
	cblogger.Info("12. ClusterHandler")
	cblogger.Info("13. TagHandler")
	cblogger.Info("14. FileSystemHandler")
	cblogger.Info("15. Exit")
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
			Region: config.Azure.Location,
			Zone:   config.Azure.Zone,
		},
	}

	cloudConnection, err := cloudDriver.ConnectCloud(connectionInfo)
	if err != nil {
		panic(err)
	}

	var resourceHandler interface{}

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
	case "nlb":
		resourceHandler, err = cloudConnection.CreateNLBHandler()
	case "disk":
		resourceHandler, err = cloudConnection.CreateDiskHandler()
	case "myimage":
		resourceHandler, err = cloudConnection.CreateMyImageHandler()
	case "regionzone":
		resourceHandler, err = cloudConnection.CreateRegionZoneHandler()
	case "price":
		resourceHandler, err = cloudConnection.CreatePriceInfoHandler()
	case "cluster":
		resourceHandler, err = cloudConnection.CreateClusterHandler()
	case "tag":
		resourceHandler, err = cloudConnection.CreateTagHandler()
	case "fileSystem":
		resourceHandler, err = cloudConnection.CreateFileSystemHandler()
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
	cblogger.Info("7. ListIID()")
	cblogger.Info("8. Exit")
}

// SecurityGroup
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
					TagList:       []irs.KeyValue{{Key: "Environment", Value: "Production"}, {Key: "Environment2", Value: "Production2"}},
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
				cblogger.Info("Start ListIID() ...")
				if listIID, err := securityHandler.ListIID(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listIID)
				}
				cblogger.Info("Finish ListIID()")
			case 8:
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
	cblogger.Info("7. ListIID()")
	cblogger.Info("8. Exit")
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
		TagList:        []irs.KeyValue{{Key: "Environment", Value: "Production"}, {Key: "Environment2", Value: "Production2"}},
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
				cblogger.Info("Start ListIID() ...")
				if listIID, err := vpcHandler.ListIID(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listIID)
				}
				cblogger.Info("Finish ListIID()")
			case 8:
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
	cblogger.Info("5. ListIID()")
	cblogger.Info("6. Exit")
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
					IId:     keypairIId,
					TagList: []irs.KeyValue{{Key: "Environment", Value: "Production"}, {Key: "Environment2", Value: "Production2"}},
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
				cblogger.Info("Start ListIID() ...")
				if listIID, err := keyPairHandler.ListIID(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listIID)
				}
				cblogger.Info("Finish ListIID()")
			case 6:
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
	cblogger.Info("10. ListIID()")
	cblogger.Info("11. Exit")
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
	imageType := irs.PublicImage
	if config.Azure.Resources.Vm.ImageType == "MyImage" {
		imageType = irs.MyImage
	}
	vmIID := irs.IID{
		NameId: config.Azure.Resources.Vm.IID.NameId,
	}
	vmReqInfo := irs.VMReqInfo{
		IId: irs.IID{
			NameId: config.Azure.Resources.Vm.IID.NameId,
		},
		ImageType: imageType,
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
		VMUserId:          config.Azure.Resources.Vm.VMUserId,
		VMUserPasswd:      config.Azure.Resources.Vm.VMUserPasswd,
		TagList:           []irs.KeyValue{{Key: "Environment", Value: "Production"}, {Key: "Environment2", Value: "Production2"}},
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
				cblogger.Info("Start ListIID() ...")
				if listIID, err := vmHandler.ListIID(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listIID)
				}
				cblogger.Info("Finish ListIID()")
			case 11:
				cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

func testNLBHandlerListPrint() {
	cblogger.Info("Test NLBHandler")
	cblogger.Info("0. Print Menu")
	cblogger.Info("1. ListNLB()")
	cblogger.Info("2. GetNLB()")
	cblogger.Info("3. CreateNLB()")
	cblogger.Info("4. DeleteNLB()")
	cblogger.Info("5. ChangeListener()")
	cblogger.Info("6. ChangeVMGroupInfo()")
	cblogger.Info("7. AddVMs()")
	cblogger.Info("8. RemoveVMs()")
	cblogger.Info("9. GetVMGroupHealthInfo()")
	cblogger.Info("10. ChangeHealthCheckerInfo()")
	cblogger.Info("11. ListIID()")
	cblogger.Info("12. Exit")
}

func testNLBHandler(config Config) {
	resourceHandler, err := getResourceHandler("nlb", config)
	if err != nil {
		cblogger.Error(err)
		return
	}

	nlbHandler := resourceHandler.(irs.NLBHandler)

	testNLBHandlerListPrint()

	nlbIId := irs.IID{
		NameId: "nlb-tester",
	}
	nlbCreateReqInfo := irs.NLBInfo{
		IId: irs.IID{
			NameId: "nlb-tester",
		},
		VpcIID: irs.IID{
			NameId: "mcb-test-vpc",
		},
		// Type: "PUBLIC",
		// Scope: "REGION",
		Listener: irs.ListenerInfo{
			Protocol: "TCP",
			Port:     "22",
		},
		VMGroup: irs.VMGroupInfo{
			Port:     "22",
			Protocol: "TCP",
			VMs: &[]irs.IID{
				{NameId: "vm-01"},
				{NameId: "vm-02"},
			},
		},
		HealthChecker: irs.HealthCheckerInfo{
			Protocol:  "TCP",
			Port:      "22",
			Interval:  10,
			Timeout:   -1,
			Threshold: 5,
			// Threshold: 429496728,
		},
		TagList: []irs.KeyValue{{Key: "Environment", Value: "Production"}, {Key: "Environment2", Value: "Production2"}},
	}
	updateListener := irs.ListenerInfo{
		Protocol: "TCP",
		Port:     "8087",
	}
	updateVMGroups := irs.VMGroupInfo{
		Protocol: "TCP",
		Port:     "8087",
		VMs: &[]irs.IID{
			{NameId: "mcb-test-vm"},
			{NameId: "mcb-test-vm2"},
		},
	}
	addVMs := []irs.IID{
		{NameId: "mcb-test-vm"},
		{NameId: "mcb-test-vm2"},
	}
	removeVMs := []irs.IID{
		{NameId: "mcb-test-vm"},
		{NameId: "mcb-test-vm2"},
	}

	updateHealthCheckerInfo := irs.HealthCheckerInfo{
		Protocol:  "TCP",
		Port:      "80",
		Interval:  10,
		Threshold: 1,
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
				testNLBHandlerListPrint()
			case 1:
				cblogger.Info("Start ListNLB() ...")
				if list, err := nlbHandler.ListNLB(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				cblogger.Info("Finish ListNLB()")
			case 2:
				cblogger.Info("Start GetNLB() ...")
				if vm, err := nlbHandler.GetNLB(nlbIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vm)
				}
				cblogger.Info("Finish GetNLB()")
			case 3:
				cblogger.Info("Start CreateNLB() ...")
				if createInfo, err := nlbHandler.CreateNLB(nlbCreateReqInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(createInfo)
				}
				cblogger.Info("Finish CreateNLB()")
			case 4:
				cblogger.Info("Start DeleteNLB() ...")
				if vmStatus, err := nlbHandler.DeleteNLB(nlbIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vmStatus)
				}
				cblogger.Info("Finish DeleteNLB()")
			case 5:
				cblogger.Info("Start ChangeListener() ...")
				if nlbInfo, err := nlbHandler.ChangeListener(nlbIId, updateListener); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(nlbInfo)
				}
				cblogger.Info("Finish ChangeListener()")
			case 6:
				cblogger.Info("Start ChangeVMGroupInfo() ...")
				if info, err := nlbHandler.ChangeVMGroupInfo(nlbIId, updateVMGroups); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(info)
				}
				cblogger.Info("Finish ChangeVMGroupInfo()")
			case 7:
				cblogger.Info("Start AddVMs() ...")
				if info, err := nlbHandler.AddVMs(nlbIId, &addVMs); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(info)
				}
				cblogger.Info("Finish AddVMs()")
			case 8:
				cblogger.Info("Start RemoveVMs() ...")
				if result, err := nlbHandler.RemoveVMs(nlbIId, &removeVMs); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(result)
				}
				cblogger.Info("Finish RemoveVMs()")
			case 9:
				cblogger.Info("Start GetVMGroupHealthInfo() ...")
				if result, err := nlbHandler.GetVMGroupHealthInfo(nlbIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(result)
				}
				cblogger.Info("Finish GetVMGroupHealthInfo()")
			case 10:
				cblogger.Info("Start ChangeHealthCheckerInfo() ...")
				if info, err := nlbHandler.ChangeHealthCheckerInfo(nlbIId, updateHealthCheckerInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(info)
				}
				cblogger.Info("Finish ChangeHealthCheckerInfo()")
			case 11:
				cblogger.Info("Start ListIID() ...")
				if listIID, err := nlbHandler.ListIID(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listIID)
				}
				cblogger.Info("Finish ListIID()")
			case 12:
				cblogger.Info("Exit")
				break Loop
			}
		}
	}
}
func testDiskHandlerListPrint() {
	cblogger.Info("Test DiskHandler")
	cblogger.Info("0. Print Menu")
	cblogger.Info("1. ListDisk()")
	cblogger.Info("2. GetDisk()")
	cblogger.Info("3. CreateDisk()")
	cblogger.Info("4. DeleteDisk()")
	cblogger.Info("5. ChangeDiskSize()")
	cblogger.Info("6. AttachDisk()")
	cblogger.Info("7. DetachDisk()")
	cblogger.Info("8. ListIID()")
	cblogger.Info("9. Exit")
}

func testDiskHandler(config Config) {
	resourceHandler, err := getResourceHandler("disk", config)
	if err != nil {
		cblogger.Error(err)
		return
	}
	diskHandler := resourceHandler.(irs.DiskHandler)

	testDiskHandlerListPrint()
	diskIId := irs.IID{
		NameId: config.Azure.Resources.Disk.IID.NameId,
	}
	createDiskReqInfo := irs.DiskInfo{
		IId: irs.IID{
			NameId: config.Azure.Resources.Disk.IID.NameId,
		},
		Zone:     config.Azure.Zone,
		TagList:  []irs.KeyValue{{Key: "Environment", Value: "Production"}, {Key: "Environment2", Value: "Production2"}},
		DiskSize: config.Azure.Resources.Disk.DiskSize,
		DiskType: config.Azure.Resources.Disk.DiskType,
	}
	delDiskIId := irs.IID{
		NameId: config.Azure.Resources.Disk.IID.NameId,
	}
	attachDiskIId := irs.IID{
		NameId: config.Azure.Resources.Disk.IID.NameId,
	}
	attachVMIId := irs.IID{
		NameId: config.Azure.Resources.Disk.AttachedVM.NameId,
	}
	updateSize := config.Azure.Resources.Disk.UpdateDiskSize
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
				testDiskHandlerListPrint()
			case 1:
				cblogger.Info("Start ListDisk() ...")
				if list, err := diskHandler.ListDisk(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				cblogger.Info("Finish ListDisk()")
			case 2:
				cblogger.Info("Start GetDisk() ...")
				if vm, err := diskHandler.GetDisk(diskIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vm)
				}
				cblogger.Info("Finish GetDisk()")
			case 3:
				cblogger.Info("Start CreateDisk() ...")
				if createInfo, err := diskHandler.CreateDisk(createDiskReqInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(createInfo)
				}
				cblogger.Info("Finish CreateDisk()")
			case 4:
				cblogger.Info("Start DeleteDisk() ...")
				if vmStatus, err := diskHandler.DeleteDisk(delDiskIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vmStatus)
				}
				cblogger.Info("Finish DeleteDisk()")
			case 5:
				cblogger.Info("Start ChangeDiskSize() ...")
				if nlbInfo, err := diskHandler.ChangeDiskSize(diskIId, updateSize); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(nlbInfo)
				}
				cblogger.Info("Finish ChangeDiskSize()")
			case 6:
				cblogger.Info("Start AttachDisk() ...")
				if info, err := diskHandler.AttachDisk(attachDiskIId, attachVMIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(info)
				}
				cblogger.Info("Finish AttachDisk()")
			case 7:
				cblogger.Info("Start DetachDisk() ...")
				if info, err := diskHandler.DetachDisk(attachDiskIId, attachVMIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(info)
				}
				cblogger.Info("Finish DetachDisk()")
			case 8:
				cblogger.Info("Start ListIID() ...")
				if listIID, err := diskHandler.ListIID(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listIID)
				}
				cblogger.Info("Finish ListIID()")
			case 9:
				cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

func testMyImageHandlerListPrint() {
	cblogger.Info("Test MyImageHandler")
	cblogger.Info("0. Print Menu")
	cblogger.Info("1. ListMyImage()")
	cblogger.Info("2. GetMyImage()")
	cblogger.Info("3. SnapshotVM()")
	cblogger.Info("4. DeleteMyImage()")
	cblogger.Info("5. ListIID()")
	cblogger.Info("6. Exit")
}

func testMyImageHandler(config Config) {
	resourceHandler, err := getResourceHandler("myimage", config)
	if err != nil {
		cblogger.Error(err)
		return
	}
	myimageHandler := resourceHandler.(irs.MyImageHandler)

	testMyImageHandlerListPrint()
	getimageIId := irs.IID{NameId: config.Azure.Resources.MyImage.IID.NameId}
	targetvm := irs.MyImageInfo{
		IId:      irs.IID{NameId: config.Azure.Resources.MyImage.IID.NameId},
		SourceVM: irs.IID{NameId: config.Azure.Resources.MyImage.SourceVM.NameId},
		TagList:  []irs.KeyValue{{Key: "Environment", Value: "Production"}, {Key: "Environment2", Value: "Production2"}},
	}
	delimageIId := irs.IID{NameId: config.Azure.Resources.MyImage.IID.NameId}
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
				testMyImageHandlerListPrint()
			case 1:
				cblogger.Info("Start ListMyImage() ...")
				if list, err := myimageHandler.ListMyImage(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				cblogger.Info("Finish ListMyImage()")
			case 2:
				cblogger.Info("Start GetMyImage() ...")
				if myimage, err := myimageHandler.GetMyImage(getimageIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(myimage)
				}
				cblogger.Info("Finish GetMyImage()")
			case 3:
				cblogger.Info("Start SnapshotVM() ...")
				if createInfo, err := myimageHandler.SnapshotVM(targetvm); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(createInfo)
				}
				cblogger.Info("Finish SnapshotVM()")
			case 4:
				cblogger.Info("Start DeleteMyImage() ...")
				if del, err := myimageHandler.DeleteMyImage(delimageIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(del)
				}
				cblogger.Info("Finish DeleteMyImage()")
			case 5:
				cblogger.Info("Start ListIID() ...")
				if listIID, err := myimageHandler.ListIID(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listIID)
				}
				cblogger.Info("Finish ListIID()")
			case 6:
				cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

func testRegionZoneHandlerListPrint() {
	cblogger.Info("Test RegionZoneHandler")
	cblogger.Info("0. Print Menu")
	cblogger.Info("1. ListRegionZone()")
	cblogger.Info("2. GetRegionZone()")
	cblogger.Info("3. ListOrgRegion()")
	cblogger.Info("4. ListOrgZone()")
	cblogger.Info("5. Exit")
}

func testRegionZoneHandler(config Config) {
	resourceHandler, err := getResourceHandler("regionzone", config)
	if err != nil {
		cblogger.Error(err)
		return
	}
	regionzoneHandler := resourceHandler.(irs.RegionZoneHandler)

	testRegionZoneHandlerListPrint()
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
				testRegionZoneHandlerListPrint()
			case 1:
				cblogger.Info("Start ListRegionZone() ...")
				if listRegionZone, err := regionzoneHandler.ListRegionZone(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listRegionZone)
				}
				cblogger.Info("Finish ListRegionZone()")
			case 2:
				cblogger.Info("Start GetRegionZone() ...")
				var region string
				fmt.Print("Enter Region Name: ")
				if _, err := fmt.Scanln(&region); err != nil {
					cblogger.Error(err)
				}
				if listRegionZone, err := regionzoneHandler.GetRegionZone(region); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listRegionZone)
				}
				cblogger.Info("Finish GetRegionZone()")
			case 3:
				cblogger.Info("Start ListOrgRegion() ...")
				if listOrgRegion, err := regionzoneHandler.ListOrgRegion(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listOrgRegion)
				}
				cblogger.Info("Finish ListOrgRegion()")
			case 4:
				cblogger.Info("Start ListOrgZone() ...")
				if listOrgZone, err := regionzoneHandler.ListOrgZone(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listOrgZone)
				}
				cblogger.Info("Finish ListOrgZone()")
			case 5:
				cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

func testPriceInfoHandlerListPrint() {
	cblogger.Info("Test PriceInfoHandler")
	cblogger.Info("0. Print Menu")
	cblogger.Info("1. ListProductFamily()")
	cblogger.Info("2. GetPriceInfo()")
	cblogger.Info("3. Exit")
}

func testPriceInfoHandler(config Config) {
	resourceHandler, err := getResourceHandler("price", config)
	if err != nil {
		cblogger.Error(err)
		return
	}
	priceInfoHandler := resourceHandler.(irs.PriceInfoHandler)

	testPriceInfoHandlerListPrint()
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
				testPriceInfoHandlerListPrint()
			case 1:
				cblogger.Info("Start ListProductFamily() ...")
				var region string
				fmt.Print("Enter Region Name: ")
				if _, err := fmt.Scanln(&region); err != nil {
					cblogger.Error(err)
				}
				if listProductFamily, err := priceInfoHandler.ListProductFamily(region); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listProductFamily)
				}
				cblogger.Info("Finish ListProductFamily()")
			case 2:
				cblogger.Info("Start GetPriceInfo() ...")
				fmt.Println("=== Enter Product Familiy ===")
				in := bufio.NewReader(os.Stdin)
				productFamiliy, err := in.ReadString('\n')
				if err != nil {
					cblogger.Error(err)
				}

				productFamiliy = strings.TrimSpace(productFamiliy)
				var region string
				fmt.Print("Enter Region Name: ")
				if _, err := fmt.Scanln(&region); err != nil {
					cblogger.Error(err)
				}

				var addFilterList string
				var filterList []irs.KeyValue
				for {
					fmt.Print("Add filter list? (y/N): ")
					_, err := fmt.Scanln(&addFilterList)
					if err != nil || strings.ToLower(addFilterList) == "n" {
						break
					}

					fmt.Println("=== Enter key to filter ===")
					in = bufio.NewReader(os.Stdin)
					key, err := in.ReadString('\n')
					if err != nil {
						cblogger.Error(err)
					}
					key = strings.TrimSpace(key)

					fmt.Println("=== Enter value to filter ===")
					in = bufio.NewReader(os.Stdin)
					value, err := in.ReadString('\n')
					if err != nil {
						cblogger.Error(err)
					}
					value = strings.TrimSpace(value)

					filterList = append(filterList, irs.KeyValue{
						Key:   key,
						Value: value,
					})
				}

				if priceInfo, err := priceInfoHandler.GetPriceInfo(productFamiliy, region, filterList); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(priceInfo)
				}
				cblogger.Info("Finish GetPriceInfo()")
			case 3:
				cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

func testClusterHandlerListPrint() {
	cblogger.Info("Test ClusterHandler")
	cblogger.Info("0. Print Menu")
	cblogger.Info("1. ListCluster()")
	cblogger.Info("2. GetCluster()")
	cblogger.Info("3. CreateCluster()")
	cblogger.Info("4. DeleteCluster()") //AddNodeGroup
	cblogger.Info("5. AddNodeGroup()")
	cblogger.Info("6. RemoveNodeGroup()")
	cblogger.Info("7. SetNodeGroupAutoScaling()")
	cblogger.Info("8. ChangeNodeGroupScaling()")
	cblogger.Info("9. UpgradeCluster()")
	cblogger.Info("10. ListIID()")
	cblogger.Info("11. Create->GET->List->AddNodeGroup->RemoveNodeGroup->SetNodeGroupAutoScaling(Change)->SetNodeGroupAutoScaling(restore)->ChangeNodeGroupScaling->Upgrade->Delete")
	cblogger.Info("12. Exit")
}

func testClusterHandler(config Config) {
	resourceHandler, err := getResourceHandler("cluster", config)
	if err != nil {
		cblogger.Error(err)
		return
	}

	clusterHandler := resourceHandler.(irs.ClusterHandler)
	testClusterHandlerListPrint()
	createreq := irs.ClusterInfo{
		IId: irs.IID{
			NameId: "test-cluster-1",
		},
		Network: irs.NetworkInfo{
			VpcIID:            irs.IID{NameId: "mcb-test-vpc"},
			SubnetIIDs:        []irs.IID{{NameId: "mcb-test-vpc-subnet1"}},
			SecurityGroupIIDs: []irs.IID{{NameId: "mcb-test-sg"}},
		},
		Version: "1.30.2",
		// ImageIID
		NodeGroupList: []irs.NodeGroupInfo{
			{
				IId:             irs.IID{NameId: "nodegroup0"},
				VMSpecName:      "Standard_B2s",
				RootDiskSize:    "default",
				KeyPairIID:      irs.IID{NameId: "mcb-test-key"},
				DesiredNodeSize: 1,
				MaxNodeSize:     2,
				MinNodeSize:     1,
				OnAutoScaling:   true,
			},
			//{
			//	IId:             irs.IID{NameId: "nodegroup1"},
			//	VMSpecName:      "Standard_B2s",
			//	RootDiskSize:    "default",
			//	KeyPairIID:      irs.IID{NameId: "azure0916"},
			//	DesiredNodeSize: 1,
			//	MaxNodeSize:     3,
			//	MinNodeSize:     1,
			//	OnAutoScaling:   true,
			//},
		},
		TagList: []irs.KeyValue{{Key: "Environment", Value: "Production"}, {Key: "Environment2", Value: "Production2"}},
	}
	addNodeGroup := irs.NodeGroupInfo{
		IId:             irs.IID{NameId: "nodegroup3"},
		VMSpecName:      "Standard_B2s",
		RootDiskSize:    "default",
		KeyPairIID:      irs.IID{NameId: "mcb-test-key"},
		DesiredNodeSize: 3,
		MaxNodeSize:     5,
		MinNodeSize:     2,
		OnAutoScaling:   true,
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
				testClusterHandlerListPrint()
			case 1:
				cblogger.Info("Start ListCluster() ...")
				if list, err := clusterHandler.ListCluster(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				cblogger.Info("Finish ListCluster()")
			case 2:
				cblogger.Info("Start GetCluster() ...")
				if clusterInfo, err := clusterHandler.GetCluster(createreq.IId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(clusterInfo)
				}
				cblogger.Info("Finish GetCluster()")
			case 3:
				cblogger.Info("Start CreateCluster() ...")
				if createInfo, err := clusterHandler.CreateCluster(createreq); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(createInfo)
				}
				cblogger.Info("Finish CreateCluster()")
			case 4:
				cblogger.Info("Start DeleteCluster() ...")
				if del, err := clusterHandler.DeleteCluster(createreq.IId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(del)
				}
				cblogger.Info("Finish DeleteCluster()")
			case 5:
				cblogger.Info("Start AddNodeGroup() ...")
				if del, err := clusterHandler.AddNodeGroup(createreq.IId, addNodeGroup); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(del)
				}
				cblogger.Info("Finish AddNodeGroup()")
			case 6:
				cblogger.Info("Start RemoveNodeGroup() ...")
				if del, err := clusterHandler.RemoveNodeGroup(createreq.IId, addNodeGroup.IId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(del)
				}
				cblogger.Info("Finish RemoveNodeGroup()")
			case 7:
				cblogger.Info("Start SetNodeGroupAutoScaling() ...")
				if del, err := clusterHandler.SetNodeGroupAutoScaling(createreq.IId, createreq.NodeGroupList[0].IId, true); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(del)
				}
				cblogger.Info("Finish SetNodeGroupAutoScaling()")
			case 8:
				cblogger.Info("Start ChangeNodeGroupScaling() ...")
				if del, err := clusterHandler.ChangeNodeGroupScaling(createreq.IId, createreq.NodeGroupList[0].IId, 3, 3, 5); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(del)
				}
				cblogger.Info("Finish ChangeNodeGroupScaling()")
			case 9:
				cblogger.Info("Start UpgradeCluster() ...")
				if del, err := clusterHandler.UpgradeCluster(createreq.IId, "1.22.12"); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(del)
				}
				cblogger.Info("Finish UpgradeCluster()")
			case 10:
				cblogger.Info("Start ListIID() ...")
				if listIID, err := clusterHandler.ListIID(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listIID)
				}
				cblogger.Info("Finish ListIID()")
			case 11:
				falowStr := "Create->GET->AddNodeGroup->RemoveNodeGroup->SetNodeGroupAutoScaling(Change)->SetNodeGroupAutoScaling(restore)->ChangeNodeGroupScaling->Delete"
				cblogger.Info(fmt.Sprintf("Start %s =====", falowStr))
				cblogger.Info("Start Create =====")
				continueCheck := true
				if createInfo, err := clusterHandler.CreateCluster(createreq); err != nil {
					continueCheck = false
					cblogger.Error("!!!!!!!!!!!!!!!!!!!Failed Create =====")
					cblogger.Error(err)
					cblogger.Info(fmt.Sprintf("Finish Failed Create ====="))
				} else {
					spew.Dump(createInfo)
					cblogger.Info("Finish Create =====")
				}
				if !continueCheck {
					continue
				}
				cblogger.Info("Start Get =====")
				if clusterInfo, err := clusterHandler.GetCluster(createreq.IId); err != nil {
					continueCheck = false
					cblogger.Error("!!!!!!!!!!!!!!!!!!!Failed Get =====")
					cblogger.Error(err)
					cblogger.Info(fmt.Sprintf("Finish Failed Get ====="))
				} else {
					spew.Dump(clusterInfo)
					cblogger.Info("Finish Get =====")
				}
				if !continueCheck {
					continue
				}
				waitCount := 0
				waitMaxCount := 200
				for {
					waitCount++
					clusterInfo, err := clusterHandler.GetCluster(createreq.IId)
					cblogger.Info(fmt.Sprintf("Waiting Check Creating Cluster Status %s", string(clusterInfo.Status)))
					if err != nil {
						cblogger.Info(fmt.Sprintf("Failed Waiting Check Creating Cluster Status"))
						continueCheck = false
						break
					}
					if clusterInfo.Status == irs.ClusterActive {
						cblogger.Info(fmt.Sprintf("Waiting Check Creating Cluster Status %s", string(clusterInfo.Status)))
						cblogger.Info("Pre-Next Current Cluster")
						spew.Dump(clusterInfo)
						cblogger.Info("@@@@@@@@@@@@@@@@@@@@@=============NextStep!=============@@@@@@@@@@@@@@@@@@@@@")
						break
					}
					time.Sleep(10 * time.Second)
					if waitCount > waitMaxCount {
						cblogger.Info(fmt.Sprintf("Waiting Check Creating Cluster Status TimeOut"))
						continueCheck = false
						break
					}
				}
				if !continueCheck {
					continue
				}
				cblogger.Info("Start AddNodeGroup =====")
				if add, err := clusterHandler.AddNodeGroup(createreq.IId, addNodeGroup); err != nil {
					continueCheck = false
					cblogger.Error("!!!!!!!!!!!!!!!!!!!Failed AddNodeGroup =====")
					cblogger.Error(err)
					cblogger.Info(fmt.Sprintf("Finish Failed AddNodeGroup ====="))
				} else {
					spew.Dump(add)
					cblogger.Info("Finish AddNodeGroup =====")
				}
				if !continueCheck {
					continue
				}
				waitCount = 0
				for {
					waitCount++
					clusterInfo, err := clusterHandler.GetCluster(createreq.IId)
					if err != nil {
						cblogger.Info(fmt.Sprintf("Failed Waiting Check AddNodeGroup Status"))
						continueCheck = false
						break
					}
					subChek := false
					for _, nodeGroup := range clusterInfo.NodeGroupList {
						if nodeGroup.IId.NameId == addNodeGroup.IId.NameId {
							if nodeGroup.Status == irs.NodeGroupActive {
								cblogger.Info(fmt.Sprintf("Waiting Check Creating AddNodeGroup Status %s", string(clusterInfo.Status)))
								cblogger.Info("Pre-Next Current Cluster")
								spew.Dump(clusterInfo)
								cblogger.Info("@@@@@@@@@@@@@@@@@@@@@=============NextStep!=============@@@@@@@@@@@@@@@@@@@@")
								subChek = true
								break
							} else {
								cblogger.Info(fmt.Sprintf("Waiting Check Creating AddNodeGroup Status %s", string(clusterInfo.Status)))
							}
						}
					}
					if subChek {
						break
					}
					time.Sleep(10 * time.Second)
					if waitCount > waitMaxCount {
						cblogger.Info(fmt.Sprintf("Waiting Check Creating Cluster AddNodeGroup TimeOut"))
						continueCheck = false
						break
					}
				}
				if !continueCheck {
					continue
				}
				cblogger.Info("Start RemoveNodeGroup =====")
				if del, err := clusterHandler.RemoveNodeGroup(createreq.IId, addNodeGroup.IId); err != nil {
					continueCheck = false
					cblogger.Error("!!!!!!!!!!!!!!!!!!!Failed RemoveNodeGroup =====")
					cblogger.Error(err)
					cblogger.Info(fmt.Sprintf("Finish Failed RemoveNodeGroup ====="))
				} else {
					spew.Dump(del)
					cblogger.Info("Finish RemoveNodeGroup =====")
				}
				if !continueCheck {
					continue
				}
				waitCount = 0
				for {
					waitCount++
					clusterInfo, err := clusterHandler.GetCluster(createreq.IId)
					if err != nil {
						cblogger.Info(fmt.Sprintf("Failed Waiting Check RemoveNodeGroup Status"))
						continueCheck = false
						break
					}
					existChk := false
					for _, nodeGroup := range clusterInfo.NodeGroupList {
						if nodeGroup.IId.NameId == addNodeGroup.IId.NameId {
							existChk = true
						}
					}
					if existChk {
						cblogger.Info(fmt.Sprintf("Waiting Check RemoveNodeGroup Exist"))
					} else {
						cblogger.Info(fmt.Sprintf("Waiting Check RemoveNodeGroup Not Exist", string(clusterInfo.Status)))
						cblogger.Info("Pre-Next Current Cluster")
						spew.Dump(clusterInfo)
						cblogger.Info("@@@@@@@@@@@@@@@@@@@@@=============NextStep!=============@@@@@@@@@@@@@@@@@@@@")
						break
					}
					time.Sleep(10 * time.Second)
					if waitCount > waitMaxCount {
						cblogger.Info(fmt.Sprintf("Waiting Check RemoveNodeGroup TimeOut"))
						continueCheck = false
						break
					}
				}
				if !continueCheck {
					continue
				}
				cblogger.Info("Start SetNodeGroupAutoScaling Change =====")
				if ch, err := clusterHandler.SetNodeGroupAutoScaling(createreq.IId, createreq.NodeGroupList[0].IId, !createreq.NodeGroupList[0].OnAutoScaling); err != nil {
					continueCheck = false
					cblogger.Error("!!!!!!!!!!!!!!!!!!!Failed SetNodeGroupAutoScaling Change=====")
					cblogger.Error(err)
					cblogger.Info(fmt.Sprintf("Finish Failed SetNodeGroupAutoScaling Change====="))
				} else {
					spew.Dump(ch)
					cblogger.Info("Finish SetNodeGroupAutoScaling =====")
				}
				if !continueCheck {
					continue
				}
				waitCount = 0
				for {
					waitCount++
					clusterInfo, err := clusterHandler.GetCluster(createreq.IId)
					if err != nil {
						cblogger.Info(fmt.Sprintf("Failed Waiting Check SetNodeGroupAutoScaling Status"))
						continueCheck = false
						break
					}
					subCheck := false
					for _, nodeGroup := range clusterInfo.NodeGroupList {
						if nodeGroup.IId.NameId == createreq.NodeGroupList[0].IId.NameId {
							if nodeGroup.Status == irs.NodeGroupActive {
								cblogger.Info(fmt.Sprintf("Waiting Check SetNodeGroupAutoScaling Status %s", string(clusterInfo.Status)))
								cblogger.Info("Pre-Next Current Cluster")
								spew.Dump(clusterInfo)
								cblogger.Info("@@@@@@@@@@@@@@@@@@@@@=============NextStep!=============@@@@@@@@@@@@@@@@@@@@")
								subCheck = true
								break
							} else {
								cblogger.Info(fmt.Sprintf("Waiting Check SetNodeGroupAutoScaling Status %s", string(clusterInfo.Status)))
							}
						}
					}
					if subCheck {
						break
					}
					time.Sleep(10 * time.Second)
					if waitCount > waitMaxCount {
						cblogger.Info(fmt.Sprintf("Waiting Check SetNodeGroupAutoScaling TimeOut"))
						continueCheck = false
						break
					}
				}
				if !continueCheck {
					continue
				}
				cblogger.Info("Start SetNodeGroupAutoScaling restore =====")
				if ch, err := clusterHandler.SetNodeGroupAutoScaling(createreq.IId, createreq.NodeGroupList[0].IId, createreq.NodeGroupList[0].OnAutoScaling); err != nil {
					continueCheck = false
					cblogger.Error("!!!!!!!!!!!!!!!!!!!Failed SetNodeGroupAutoScaling restore=====")
					cblogger.Error(err)
					cblogger.Info(fmt.Sprintf("Finish Failed SetNodeGroupAutoScaling restore====="))
				} else {
					spew.Dump(ch)
					cblogger.Info("Finish SetNodeGroupAutoScaling =====")
				}
				if !continueCheck {
					continue
				}
				waitCount = 0
				for {
					waitCount++
					clusterInfo, err := clusterHandler.GetCluster(createreq.IId)
					if err != nil {
						cblogger.Info(fmt.Sprintf("Failed Waiting Check ReStore SetNodeGroupAutoScaling Status"))
						continueCheck = false
						break
					}
					subCheck := false
					for _, nodeGroup := range clusterInfo.NodeGroupList {
						if nodeGroup.IId.NameId == createreq.NodeGroupList[0].IId.NameId {
							if nodeGroup.Status == irs.NodeGroupActive {
								cblogger.Info(fmt.Sprintf("Waiting Check ReStore SetNodeGroupAutoScaling Status %s", string(clusterInfo.Status)))
								cblogger.Info("Pre-Next Current Cluster")
								spew.Dump(clusterInfo)
								cblogger.Info("@@@@@@@@@@@@@@@@@@@@@=============NextStep!=============@@@@@@@@@@@@@@@@@@@@")
								subCheck = true
								break
							} else {
								cblogger.Info(fmt.Sprintf("Waiting Check ReStore SetNodeGroupAutoScaling Status %s", string(clusterInfo.Status)))
							}
						}
					}
					if subCheck {
						break
					}
					time.Sleep(10 * time.Second)
					if waitCount > waitMaxCount {
						cblogger.Info(fmt.Sprintf("Waiting Check ReStore SetNodeGroupAutoScaling TimeOut"))
						continueCheck = false
						break
					}
				}
				if !continueCheck {
					continue
				}
				cblogger.Info("Start ChangeNodeGroupScaling =====")
				if del, err := clusterHandler.ChangeNodeGroupScaling(createreq.IId, createreq.NodeGroupList[0].IId, 3, 1, 3); err != nil {
					continueCheck = false
					cblogger.Error("!!!!!!!!!!!!!!!!!!!Failed ChangeNodeGroupScaling =====")
					cblogger.Error(err)
					cblogger.Info(fmt.Sprintf("Finish Failed ChangeNodeGroupScaling ====="))
				} else {
					spew.Dump(del)
					cblogger.Info("Finish ChangeNodeGroupScaling =====")
				}
				if !continueCheck {
					continue
				}
				waitCount = 0
				for {
					waitCount++
					clusterInfo, err := clusterHandler.GetCluster(createreq.IId)
					if err != nil {
						cblogger.Info(fmt.Sprintf("Failed Waiting Check ChangeNodeGroupScaling Status"))
						continueCheck = false
						break
					}
					subCheck := false
					for _, nodeGroup := range clusterInfo.NodeGroupList {
						if nodeGroup.IId.NameId == createreq.NodeGroupList[0].IId.NameId {
							if nodeGroup.Status == irs.NodeGroupActive {
								cblogger.Info(fmt.Sprintf("Waiting Check ChangeNodeGroupScaling Status %s", string(clusterInfo.Status)))
								cblogger.Info("Pre-Next Current Cluster")
								spew.Dump(clusterInfo)
								cblogger.Info("@@@@@@@@@@@@@@@@@@@@@=============NextStep!=============@@@@@@@@@@@@@@@@@@@@")
								subCheck = true
								break
							} else {
								cblogger.Info(fmt.Sprintf("Waiting Check ChangeNodeGroupScaling Status %s", string(clusterInfo.Status)))
							}
						}
					}
					if subCheck {
						break
					}
					time.Sleep(10 * time.Second)
					if waitCount > waitMaxCount {
						cblogger.Info(fmt.Sprintf("Waiting Check ChangeNodeGroupScaling TimeOut"))
						continueCheck = false
						break
					}
				}
				if !continueCheck {
					continue
				}
				////////////////
				cblogger.Info("Start Upgrade =====")
				if up, err := clusterHandler.UpgradeCluster(createreq.IId, "1.23.8"); err != nil {
					continueCheck = false
					cblogger.Error("!!!!!!!!!!!!!!!!!!!Failed Upgrade =====")
					cblogger.Error(err)
					cblogger.Info(fmt.Sprintf("Finish Failed Upgrade ====="))
				} else {
					spew.Dump(up)
					cblogger.Info("Finish Upgrade =====")
				}
				if !continueCheck {
					continue
				}
				waitCount = 0
				for {
					waitCount++
					clusterInfo, err := clusterHandler.GetCluster(createreq.IId)
					cblogger.Info(fmt.Sprintf("Waiting Check Upgrade Cluster Status %s", string(clusterInfo.Status)))
					if err != nil {
						cblogger.Info(fmt.Sprintf("Failed Waiting Check Upgrade Cluster Status"))
						continueCheck = false
						break
					}
					if clusterInfo.Status == irs.ClusterActive {
						cblogger.Info(fmt.Sprintf("Waiting Check Upgrade Cluster Status %s", string(clusterInfo.Status)))
						cblogger.Info("Pre-Next Current Cluster")
						spew.Dump(clusterInfo)
						cblogger.Info("@@@@@@@@@@@@@@@@@@@@@=============NextStep!=============@@@@@@@@@@@@@@@@@@@@")
						break
					}
					time.Sleep(10 * time.Second)
					if waitCount > waitMaxCount {
						cblogger.Info(fmt.Sprintf("Waiting Check Upgrade Cluster Status TimeOut"))
						continueCheck = false
						break
					}
				}
				if !continueCheck {
					continue
				}

				///////////
				cblogger.Info("Start Delete =====")
				if del, err := clusterHandler.DeleteCluster(createreq.IId); err != nil {
					cblogger.Error("Failed Delete =====")
					cblogger.Error(err)
				} else {
					spew.Dump(del)
					cblogger.Info("Finish Delete =====")
				}
				cblogger.Info(fmt.Sprintf("Finish %s =====", falowStr))
			case 12:
				cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

func testTagHandlerListPrint() {
	cblogger.Info("Test TagHandler")
	cblogger.Info("0. Print Menu")
	cblogger.Info("1. AddTag()")
	cblogger.Info("2. ListTag()")
	cblogger.Info("3. GetTag()")
	cblogger.Info("4. RemoveTag()")
	cblogger.Info("5. FindTag()")
	cblogger.Info("6. Exit")
}

func testTagHandler(config Config) {
	resourceHandler, err := getResourceHandler("tag", config)
	if err != nil {
		cblogger.Error(err)
		return
	}

	tagHandler := resourceHandler.(irs.TagHandler)
	testTagHandlerListPrint()

	tagReq := irs.KeyValue{Key: "Environment", Value: "Production"}
	resType := irs.RSType("cluster")
	resIID := irs.IID{NameId: "test-cluster-2", SystemId: ""}
	// resIID := irs.IID{NameId: "sg01", SystemId: ""}
	// resIID := irs.IID{NameId: "keypair-01", SystemId: ""}
	// resIID := irs.IID{NameId: "vm-01", SystemId: ""}

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
				testTagHandlerListPrint()
			case 1:
				cblogger.Info("Start AddTag() ...")
				if tag, err := tagHandler.AddTag(resType, resIID, tagReq); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(tag)
				}
				cblogger.Info("Finish AddTag()")
			case 2:
				cblogger.Info("Start ListTag() ...")
				if tags, err := tagHandler.ListTag(resType, resIID); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(tags)
				}
				cblogger.Info("Finish ListTag()")
			case 3:
				cblogger.Info("Start GetTag() ...")
				if tag, err := tagHandler.GetTag(resType, resIID, tagReq.Key); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(tag)
				}
				cblogger.Info("Finish GetTag()")
			case 4:
				cblogger.Info("Start RemoveTag() ...")
				if success, err := tagHandler.RemoveTag(resType, resIID, tagReq.Key); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(success)
				}
				cblogger.Info("Finish RemoveTag()")
			case 5:
				cblogger.Info("Start FindTag() ...")
				keyword := "Environment"
				// keyword := "createdBy"
				if tagInfos, err := tagHandler.FindTag(resType, keyword); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(tagInfos)
				}
				cblogger.Info("Finish FindTag()")
			case 6:
				break Loop
			}
		}
	}
}
func testFileSystemHandlerListPrint() {
	cblogger.Info("Test fileSystemHandler")
	cblogger.Info("0. Print Menu")
	cblogger.Info("1. ListFileSystem()")
	cblogger.Info("2. GetFileSystem()")
	cblogger.Info("3. CreateFileSystem()")
	cblogger.Info("4. DeleteFileSystem()")
	cblogger.Info("5. AddAccessSubnet()")
	cblogger.Info("6. RemoveAccessSubnet()")
	cblogger.Info("7. ListAccessSubnet")
	cblogger.Info("8. ListIID()")
	cblogger.Info("9. GetMetaInfo()")
	cblogger.Info("10. Exit")
}

// FileSystemHandler
func testFileSystemHandler(config Config) {
	resourceHandler, err := getResourceHandler("fileSystem", config)
	if err != nil {
		cblogger.Error(err)
		return
	}

	fileSystemHandler := resourceHandler.(irs.FileSystemHandler)

	fileNameId := irs.IID{
		NameId: config.Azure.Resources.File.IID.NameId,
	}

	subnetIID := irs.IID{
		NameId: config.Azure.Resources.File.AccessSubnetIIDs[0].NameId,
	}

	vpcIID := irs.IID{
		NameId: config.Azure.Resources.File.VpcIID.NameId,
	}

	subnetIIDs := config.Azure.Resources.File.AccessSubnetIIDs
	var accessSubnetList []irs.IID
	for _, subnet := range subnetIIDs {
		accessSubnetList = append(accessSubnetList, irs.IID{
			NameId:   subnet.NameId,
			SystemId: "",
		})
	}
	createreq := irs.FileSystemInfo{
		IId: fileNameId,
		//Region:           string,
		//Zone:
		VpcIID:           vpcIID,
		AccessSubnetList: accessSubnetList,
		NFSVersion:       "4.1",
		PerformanceInfo: map[string]string{
			"Tier": "Premium",
		},
	}

	testFileSystemHandlerListPrint()

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
				testFileSystemHandlerListPrint()
			case 1:
				fmt.Println("Start ListFileSystem() ...")
				if fileSystemList, err := fileSystemHandler.ListFileSystem(); err != nil {
					fmt.Println(err)
				} else {
					spew.Dump(fileSystemList)
				}
				fmt.Println("Finish ListFileSystem()")
			case 2:
				cblogger.Info("Start GetFileSystem() ...")
				if fileSystem, err := fileSystemHandler.GetFileSystem(fileNameId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(fileSystem)
				}
				cblogger.Info("Finish GetFileSystem()")
			case 3:
				fmt.Println("Start CreateFileSystem() ...")
				fileSystem, err := fileSystemHandler.CreateFileSystem(createreq)
				if err != nil {
					fmt.Println(err)
				} else {
					spew.Dump(fileSystem)
				}
				fmt.Println("Finish CreateFileSystem()")
			case 4:
				fmt.Println("Start DeleteFileSystem() ...")
				if ok, err := fileSystemHandler.DeleteFileSystem(fileNameId); !ok {
					fmt.Println(err)
				}
				fmt.Println("Finish DeleteFileSystem()")
			case 5:
				fmt.Println("Start AddAccessSubnet() ...")
				fmt.Printf("DEBUG: subnetIID.NameId = '%s'\n", accessSubnetList)
				if fileSystem, err := fileSystemHandler.AddAccessSubnet(fileNameId, subnetIID); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(fileSystem)
				}
				fmt.Println("Finish AddAccessSubnet()")
			case 6:
				fmt.Println("Start RemoveAccessSubnet() ...")
				if ok, err := fileSystemHandler.RemoveAccessSubnet(fileNameId, subnetIID); !ok {
					fmt.Println(err)
				}
				fmt.Println("Finish RemoveAccessSubnet()")
			case 7:
				cblogger.Info("Start ListAccessSubnet() ...")
				if listSubnet, err := fileSystemHandler.ListAccessSubnet(fileNameId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listSubnet)
				}
				cblogger.Info("Finish ListAccessSubnet()")
			case 8:
				cblogger.Info("Start ListIID() ...")
				if listIID, err := fileSystemHandler.ListIID(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listIID)
				}
				cblogger.Info("Finish ListIID()")
			case 9:
				cblogger.Info("Start GetMetaInfo() ...")
				if listIID, err := fileSystemHandler.GetMetaInfo(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(listIID)
				}
				cblogger.Info("Finish ListIID()")
			case 10:
				fmt.Println("Exit")
				break Loop
			}
		}
	}
}

func testMonitoringHandlerListPrint() {
	cblogger.Info("Test MonitoringHandler")
	cblogger.Info("0. Print Menu")
	cblogger.Info("1. GetVMMetricData()")
	cblogger.Info("2. GetClusterNodeMetricData()")
	cblogger.Info("3. Exit")
}

func testMonitoringHandlerMetricTypeListPrint() {
	cblogger.Info("Metric Types")
	cblogger.Info("1. CPUUsage")
	cblogger.Info("2. MemoryUsage")
	cblogger.Info("3. DiskRead")
	cblogger.Info("4. DiskWrite")
	cblogger.Info("5. DiskReadOps")
	cblogger.Info("6. DiskWriteOps")
	cblogger.Info("7. NetworkIn")
	cblogger.Info("8. NetworkOut")
}

func testMonitoringHandler(config Config) {
	resourceHandler, err := getResourceHandler("monitoring", config)
	if err != nil {
		cblogger.Error(err)
		return
	}
	monitoringHandler := resourceHandler.(irs.MonitoringHandler)

	testMonitoringHandlerListPrint()
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
				testMonitoringHandlerListPrint()
			case 1:
				cblogger.Info("Start GetMetricData() ...")

				fmt.Println("=== Enter VM's name ===")
				in := bufio.NewReader(os.Stdin)
				vmName, err := in.ReadString('\n')
				if err != nil {
					cblogger.Error(err)
				}
				vmName = strings.TrimSpace(vmName)

				fmt.Println("=== Enter metric type (Default: cpu_usage) ===")
				testMonitoringHandlerMetricTypeListPrint()
				inputCnt, err := fmt.Scan(&commandNum)
				if err != nil {
					cblogger.Error(err)
				}
				var metricType irs.MetricType
				if inputCnt == 1 {
					switch commandNum {
					case 1:
						metricType = irs.CPUUsage
					case 2:
						metricType = irs.MemoryUsage
					case 3:
						metricType = irs.DiskRead
					case 4:
						metricType = irs.DiskWrite
					case 5:
						metricType = irs.DiskReadOps
					case 6:
						metricType = irs.DiskWriteOps
					case 7:
						metricType = irs.NetworkIn
					case 8:
						metricType = irs.NetworkOut
					default:
						cblogger.Error("Invalid input")
					}
				}

				fmt.Println("=== Enter period (minute) (Default: 1m) ===")
				in = bufio.NewReader(os.Stdin)
				periodMinute, err := in.ReadString('\n')
				if err != nil {
					cblogger.Error(err)
				}
				periodMinute = strings.TrimSpace(periodMinute)

				fmt.Println("=== Enter time before (hour) (Default: 1h) ===")
				in = bufio.NewReader(os.Stdin)
				timeBeforeHour, err := in.ReadString('\n')
				if err != nil {
					cblogger.Error(err)
				}
				timeBeforeHour = strings.TrimSpace(timeBeforeHour)

				if getVMMetricData, err := monitoringHandler.GetVMMetricData(
					irs.VMMonitoringReqInfo{
						VMIID: irs.IID{
							NameId: vmName,
						},
						MetricType:     metricType,
						IntervalMinute: periodMinute,
						TimeBeforeHour: timeBeforeHour,
					}); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(getVMMetricData)
				}
				cblogger.Info("Finish GetVMMetricData()")
			case 2:
				cblogger.Info("Start GetClusterNodeMetricData() ...")

				fmt.Println("=== Enter Cluster's name ===")
				in := bufio.NewReader(os.Stdin)
				clusterName, err := in.ReadString('\n')
				if err != nil {
					cblogger.Error(err)
				}
				clusterName = strings.TrimSpace(clusterName)

				fmt.Println("=== Enter NodeGroup's name ===")
				in = bufio.NewReader(os.Stdin)
				nodeGroupName, err := in.ReadString('\n')
				if err != nil {
					cblogger.Error(err)
				}
				nodeGroupName = strings.TrimSpace(nodeGroupName)

				fmt.Println("=== Enter VM's name ===")
				in = bufio.NewReader(os.Stdin)
				vmName, err := in.ReadString('\n')
				if err != nil {
					cblogger.Error(err)
				}
				vmName = strings.TrimSpace(vmName)

				fmt.Println("=== Enter metric type (Default: cpu_usage) ===")
				testMonitoringHandlerMetricTypeListPrint()
				inputCnt, err := fmt.Scan(&commandNum)
				if err != nil {
					cblogger.Error(err)
				}
				var metricType irs.MetricType
				if inputCnt == 1 {
					switch commandNum {
					case 1:
						metricType = irs.CPUUsage
					case 2:
						metricType = irs.MemoryUsage
					case 3:
						metricType = irs.DiskRead
					case 4:
						metricType = irs.DiskWrite
					case 5:
						metricType = irs.DiskReadOps
					case 6:
						metricType = irs.DiskWriteOps
					case 7:
						metricType = irs.NetworkIn
					case 8:
						metricType = irs.NetworkOut
					default:
						cblogger.Error("Invalid input")
					}
				}

				fmt.Println("=== Enter period (minute) (Default: 1m) ===")
				in = bufio.NewReader(os.Stdin)
				periodMinute, err := in.ReadString('\n')
				if err != nil {
					cblogger.Error(err)
				}
				periodMinute = strings.TrimSpace(periodMinute)

				fmt.Println("=== Enter time before (hour) (Default: 1h) ===")
				in = bufio.NewReader(os.Stdin)
				timeBeforeHour, err := in.ReadString('\n')
				if err != nil {
					cblogger.Error(err)
				}
				timeBeforeHour = strings.TrimSpace(timeBeforeHour)

				if getVMMetricData, err := monitoringHandler.GetClusterNodeMetricData(
					irs.ClusterNodeMonitoringReqInfo{
						ClusterIID: irs.IID{
							NameId: clusterName,
						},
						NodeGroupID: irs.IID{
							NameId: nodeGroupName,
						},
						NodeIID: irs.IID{
							NameId: vmName,
						},
						MetricType:     metricType,
						IntervalMinute: periodMinute,
						TimeBeforeHour: timeBeforeHour,
					}); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(getVMMetricData)
				}
				cblogger.Info("Finish GetVMMetricData()")
			case 3:
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
				testNLBHandler(config)
				showTestHandlerInfo()
			case 8:
				testDiskHandler(config)
				showTestHandlerInfo()
			case 9:
				testMyImageHandler(config)
				showTestHandlerInfo()
			case 10:
				testRegionZoneHandler(config)
				showTestHandlerInfo()
			case 11:
				testPriceInfoHandler(config)
				showTestHandlerInfo()
			case 12:
				testClusterHandler(config)
				showTestHandlerInfo()
			case 13:
				testTagHandler(config)
				showTestHandlerInfo()
			case 14:
				testFileSystemHandler(config)
				showTestHandlerInfo()
			case 15:
				cblogger.Info("Exit Test ResourceHandler Program")
				break Loop
			}
		}
	}
}
