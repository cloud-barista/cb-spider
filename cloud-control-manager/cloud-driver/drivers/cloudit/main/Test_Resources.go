package main

import (
	"fmt"
	cblog "github.com/cloud-barista/cb-log"
	cidrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"strconv"

	"gopkg.in/yaml.v3"
)

// duplicateName
var res_cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	res_cblogger = cblog.GetLogger("CB-SPIDER")
}

func getResourceHandler(resourceType string, config ResourceConfig) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(cidrv.ClouditDriver)

	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			IdentityEndpoint: config.Cloudit.IdentityEndpoint,
			Username:         config.Cloudit.Username,
			Password:         config.Cloudit.Password,
			TenantId:         config.Cloudit.TenantID,
			AuthToken:        config.Cloudit.AuthToken,
			ClusterId:        config.Cloudit.ClusterID,
		},
	}

	cloudConnection, _ := cloudDriver.ConnectCloud(connectionInfo)

	var resourceHandler interface{}

	switch resourceType {
	case "image":
		resourceHandler, _ = cloudConnection.CreateImageHandler()
	case "security":
		resourceHandler, _ = cloudConnection.CreateSecurityHandler()
	case "vpc":
		resourceHandler, _ = cloudConnection.CreateVPCHandler()
	case "keypair":
		resourceHandler, _ = cloudConnection.CreateKeyPairHandler()
	case "vmspec":
		resourceHandler, _ = cloudConnection.CreateVMSpecHandler()
	case "vm":
		resourceHandler, _ = cloudConnection.CreateVMHandler()
	case "nlb":
		resourceHandler, _ = cloudConnection.CreateNLBHandler()
	case "disk":
		resourceHandler, _ = cloudConnection.CreateDiskHandler()
	case "myimage":
		resourceHandler, _ = cloudConnection.CreateMyImageHandler()
	}
	return resourceHandler, nil
}

func testVPCHandlerListPrint() {
	res_cblogger.Info("Test VPCHandler")
	res_cblogger.Info("0. Print Menu")
	res_cblogger.Info("1. ListVPC()")
	res_cblogger.Info("2. GetVPC()")
	res_cblogger.Info("3. CreateVPC()")
	res_cblogger.Info("4. DeleteVPC()")
	res_cblogger.Info("5. AddSubnet()")
	res_cblogger.Info("6. RemoveSubnet()")
	res_cblogger.Info("7. Exit")
}

func testVPCHandler(config ResourceConfig) {
	resourceHandler, err := getResourceHandler("vpc", config)
	if err != nil {
		res_cblogger.Error(err)
		return
	}

	vpcHandler := resourceHandler.(irs.VPCHandler)

	testVPCHandlerListPrint()

	vpcIID := irs.IID{NameId: config.Cloudit.Resources.VpcIID.NameId}

	subnetLists := config.Cloudit.Resources.VpcIID.Subnets
	var subnetInfoList []irs.SubnetInfo
	for _, sb := range subnetLists {
		info := irs.SubnetInfo{
			IId: irs.IID{
				NameId: sb.SubnetIID.NameId,
			},
			IPv4_CIDR: sb.IPv4_CIDR,
		}
		subnetInfoList = append(subnetInfoList, info)
	}

	VPCReqInfo := irs.VPCReqInfo{
		IId:            vpcIID,
		SubnetInfoList: subnetInfoList,
	}
	addSubnet := config.Cloudit.Resources.VpcIID.AddSubnet
	addSubnetInfo := irs.SubnetInfo{
		IId: irs.IID{
			NameId: addSubnet.SubnetIID.NameId,
		},
		IPv4_CIDR: addSubnet.IPv4_CIDR,
	}
	removeSubnet := config.Cloudit.Resources.VpcIID.RemoveSubnet
	removeSubnetInfo := irs.SubnetInfo{
		IId: irs.IID{
			NameId: removeSubnet.SubnetIID.NameId,
		},
	}
	//deleteVpcid := irs.IID{
	//	NameId: "bcr02a.tok02.774",
	//}
Loop:

	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			res_cblogger.Error(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				testVPCHandlerListPrint()
			case 1:
				res_cblogger.Info("Start ListVPC() ...")
				if list, err := vpcHandler.ListVPC(); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				res_cblogger.Info("Finish ListVPC()")
			case 2:
				res_cblogger.Info("Start GetVPC() ...")
				if vpcInfo, err := vpcHandler.GetVPC(vpcIID); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(vpcInfo)
				}
				res_cblogger.Info("Finish GetVPC()")
			case 3:
				res_cblogger.Info("Start CreateVPC() ...")
				if vpcInfo, err := vpcHandler.CreateVPC(VPCReqInfo); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(vpcInfo)
				}
				res_cblogger.Info("Finish CreateVPC()")
			case 4:
				res_cblogger.Info("Start DeleteVPC() ...")
				if result, err := vpcHandler.DeleteVPC(vpcIID); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(result)
				}
				res_cblogger.Info("Finish DeleteVPC()")
			case 5:
				res_cblogger.Info("Start AddSubnet() ...")
				if vpcInfo, err := vpcHandler.AddSubnet(vpcIID, addSubnetInfo); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(vpcInfo)
				}
				res_cblogger.Info("Finish AddSubnet()")
			case 6:
				res_cblogger.Info("Start RemoveSubnet() ...")
				if vpcInfo, err := vpcHandler.RemoveSubnet(vpcIID, removeSubnetInfo.IId); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(vpcInfo)
				}
				res_cblogger.Info("Finish RemoveSubnet()")
			case 7:
				res_cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

func testImageHandlerListPrint() {
	res_cblogger.Info("Test ImageHandler")
	res_cblogger.Info("0. Print Menu")
	res_cblogger.Info("1. ListImage()")
	res_cblogger.Info("2. GetImage()")
	res_cblogger.Info("3. CreateImage()")
	res_cblogger.Info("4. DeleteImage()")
	res_cblogger.Info("5. Exit")
}

func testimageHandler(config ResourceConfig) {
	resourceHandler, err := getResourceHandler("image", config)
	if err != nil {
		res_cblogger.Error(err)
		return
	}

	imageHandler := resourceHandler.(irs.ImageHandler)

	testImageHandlerListPrint()

	imageIID := irs.IID{NameId: config.Cloudit.Resources.ImageIID.NameId, SystemId: config.Cloudit.Resources.ImageIID.SystemId}

Loop:
	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			res_cblogger.Error(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				testImageHandlerListPrint()
			case 1:
				res_cblogger.Info("Start ListImage() ...")
				if list, err := imageHandler.ListImage(); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				res_cblogger.Info("Finish ListImage()")
			case 2:
				res_cblogger.Info("Start GetImage() ...")
				if imageInfo, err := imageHandler.GetImage(imageIID); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(imageInfo)
				}
				res_cblogger.Info("Finish GetImage()")
			case 3:
				res_cblogger.Info("Start CreateImage() ...")
				res_cblogger.Info("Finish CreateImage()")
			case 4:
				res_cblogger.Info("Start DeleteImage() ...")
				res_cblogger.Info("Finish DeleteImage()")
			case 5:
				res_cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

func testVMSpecHandlerListPrint() {
	res_cblogger.Info("Test VMSpecHandler")
	res_cblogger.Info("0. Print Menu")
	res_cblogger.Info("1. ListVMSpec()")
	res_cblogger.Info("2. GetVMSpec()")
	res_cblogger.Info("3. ListOrgVMSpec()")
	res_cblogger.Info("4. GetOrgVMSpec()")
	res_cblogger.Info("5. Exit")
}

func testvmspecHandler(config ResourceConfig) {
	resourceHandler, err := getResourceHandler("vmspec", config)
	if err != nil {
		res_cblogger.Error(err)
		return
	}

	vmSpecHandler := resourceHandler.(irs.VMSpecHandler)

	testVMSpecHandlerListPrint()

Loop:
	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			res_cblogger.Error(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				testVMSpecHandlerListPrint()
			case 1:
				res_cblogger.Info("Start ListVMSpec() ...")
				if list, err := vmSpecHandler.ListVMSpec(); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				res_cblogger.Info("Finish ListVMSpec()")
			case 2:
				res_cblogger.Info("Start GetVMSpec() ...")
				if vmSpecInfo, err := vmSpecHandler.GetVMSpec(config.Cloudit.Resources.VmSpecName); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(vmSpecInfo)
				}
				res_cblogger.Info("Finish GetVMSpec()")
			case 3:
				res_cblogger.Info("Start ListOrgVMSpec() ...")
				if listStr, err := vmSpecHandler.ListOrgVMSpec(); err != nil {
					res_cblogger.Error(err)
				} else {
					fmt.Println(listStr)
				}
				res_cblogger.Info("Finish ListOrgVMSpec()")
			case 4:
				res_cblogger.Info("Start GetOrgVMSpec() ...")
				if vmSpecStr, err := vmSpecHandler.GetOrgVMSpec(config.Cloudit.Resources.VmSpecName); err != nil {
					res_cblogger.Error(err)
				} else {
					fmt.Println(vmSpecStr)
				}
				res_cblogger.Info("Finish GetOrgVMSpec()")
			case 5:
				res_cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

func testSecurityHandlerListPrint() {
	res_cblogger.Info("Test securityHandler")
	res_cblogger.Info("0. Print Menu")
	res_cblogger.Info("1. ListSecurity()")
	res_cblogger.Info("2. GetSecurity()")
	res_cblogger.Info("3. CreateSecurity()")
	res_cblogger.Info("4. DeleteSecurity()")
	res_cblogger.Info("5. AddRules()")
	res_cblogger.Info("6. RemoveRules()")
	res_cblogger.Info("7. Exit")
}

func testsecurityGroupHandler(config ResourceConfig) {
	handler, _ := getResourceHandler("security", config)
	securityHandler := handler.(irs.SecurityHandler)
	testSecurityHandlerListPrint()
	securityIId := irs.IID{NameId: config.Cloudit.Resources.SecurityGroup.NameId}
	securityRules := config.Cloudit.Resources.SecurityGroup.Rules
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
	securityAddRules := config.Cloudit.Resources.SecurityGroup.AddRules
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
	securityRemoveRules := config.Cloudit.Resources.SecurityGroup.RemoveRules
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

func testKeyPairHandlerListPrint() {
	res_cblogger.Info("Test KeyPairHandler")
	res_cblogger.Info("0. Print Menu")
	res_cblogger.Info("1. ListKey()")
	res_cblogger.Info("2. GetKey()")
	res_cblogger.Info("3. CreateKey()")
	res_cblogger.Info("4. DeleteKey()")
	res_cblogger.Info("5. Exit")
}

func testKeypairHandler(config ResourceConfig) {
	handler, _ := getResourceHandler("keypair", config)
	keyPairHandler := handler.(irs.KeyPairHandler)
	testKeyPairHandlerListPrint()

	keypairIId := irs.IID{
		NameId:   config.Cloudit.Resources.KeyPairIID.NameId,
		SystemId: config.Cloudit.Resources.KeyPairIID.SystemId,
	}

Loop:
	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			res_cblogger.Error(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				testKeyPairHandlerListPrint()
			case 1:
				res_cblogger.Info("Start ListKey() ...")
				if keyPairList, err := keyPairHandler.ListKey(); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(keyPairList)
				}
				res_cblogger.Info("Finish ListKey()")
			case 2:
				res_cblogger.Info("Start GetKey() ...")
				if keyPairInfo, err := keyPairHandler.GetKey(keypairIId); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(keyPairInfo)
				}
				res_cblogger.Info("Finish GetKey()")
			case 3:
				res_cblogger.Info("Start CreateKey() ...")
				reqInfo := irs.KeyPairReqInfo{
					IId: keypairIId,
				}
				if keyInfo, err := keyPairHandler.CreateKey(reqInfo); err != nil {
					res_cblogger.Error(err)
				} else {
					keypairIId = keyInfo.IId
					spew.Dump(keyInfo)
				}
				res_cblogger.Info("Finish CreateKey()")
			case 4:
				res_cblogger.Info("Start DeleteKey() ...")
				if ok, err := keyPairHandler.DeleteKey(keypairIId); !ok {
					res_cblogger.Error(err)
				}
				res_cblogger.Info("Finish DeleteKey()")
			case 5:
				res_cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

func testVMHandlerListPrint() {
	res_cblogger.Info("Test VMSpecHandler")
	res_cblogger.Info("0. Print Menu")
	res_cblogger.Info("1. ListVM()")
	res_cblogger.Info("2. GetVM()")
	res_cblogger.Info("3. ListVMStatus()")
	res_cblogger.Info("4. GetVMStatus()")
	res_cblogger.Info("5. StartVM()")
	res_cblogger.Info("6. RebootVM()")
	res_cblogger.Info("7. SuspendVM()")
	res_cblogger.Info("8. ResumeVM()")
	res_cblogger.Info("9. TerminateVM()")
	res_cblogger.Info("10. StartVM() - from MyImage")
	res_cblogger.Info("11. Exit")
}

func testVmHandler(config ResourceConfig) {
	handler, _ := getResourceHandler("vm", config)
	testVMHandlerListPrint()

	vmHandler := handler.(irs.VMHandler)

	configsgIIDs := config.Cloudit.VM.SecurityGroupIIDs
	var SecurityGroupIIDs []irs.IID
	for _, sg := range configsgIIDs {
		SecurityGroupIIDs = append(SecurityGroupIIDs, irs.IID{NameId: sg.NameId, SystemId: sg.SystemId})
	}
	vmIID := irs.IID{
		NameId:   config.Cloudit.VM.IID.NameId,
		SystemId: config.Cloudit.VM.IID.SystemId,
	}
	var vmDataDiskIIDs []irs.IID
	for _, dataDisk := range config.Cloudit.VM.DataDiskIIDs {
		vmDataDiskIIDs = append(vmDataDiskIIDs, irs.IID{NameId: dataDisk.NameId})
	}
	vmReqInfo := irs.VMReqInfo{
		IId: irs.IID{
			NameId: config.Cloudit.VM.IID.NameId,
		},
		ImageType: irs.PublicImage,
		ImageIID: irs.IID{
			NameId:   config.Cloudit.VM.ImageIID.NameId,
			SystemId: config.Cloudit.VM.ImageIID.SystemId,
		},
		VpcIID: irs.IID{
			NameId: config.Cloudit.VM.VpcIID.NameId,
		},
		SubnetIID: irs.IID{
			NameId: config.Cloudit.VM.SubnetIID.NameId,
		},
		VMSpecName: config.Cloudit.VM.VmSpecName,
		KeyPairIID: irs.IID{
			NameId: config.Cloudit.VM.KeyPairIID.NameId,
		},
		SecurityGroupIIDs: SecurityGroupIIDs,
		RootDiskSize:      "",
		RootDiskType:      "",
		DataDiskIIDs:      vmDataDiskIIDs,
		VMUserPasswd:      config.Cloudit.VM.VMUserPasswd,
	}
	vmFromSnapshotReqInfo := irs.VMReqInfo{
		IId: irs.IID{
			NameId: config.Cloudit.VMFromMyImage.IID.NameId,
		},
		ImageType: irs.MyImage,
		ImageIID: irs.IID{
			NameId:   config.Cloudit.VMFromMyImage.ImageIID.NameId,
			SystemId: config.Cloudit.VMFromMyImage.ImageIID.SystemId,
		},
		VpcIID: irs.IID{
			NameId: config.Cloudit.VMFromMyImage.VpcIID.NameId,
		},
		SubnetIID: irs.IID{
			NameId: config.Cloudit.VMFromMyImage.SubnetIID.NameId,
		},
		VMSpecName: config.Cloudit.VMFromMyImage.VmSpecName,
		KeyPairIID: irs.IID{
			NameId: config.Cloudit.VMFromMyImage.KeyPairIID.NameId,
		},
		SecurityGroupIIDs: SecurityGroupIIDs,
		RootDiskSize:      "",
		RootDiskType:      "",
		VMUserId:          config.Cloudit.VMFromMyImage.VMUserId,
		VMUserPasswd:      config.Cloudit.VMFromMyImage.VMUserPasswd,
	}

Loop:
	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			res_cblogger.Error(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				testVMHandlerListPrint()
			case 1:
				res_cblogger.Info("Start ListVM() ...")
				if list, err := vmHandler.ListVM(); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				res_cblogger.Info("Finish ListVM()")
			case 2:
				res_cblogger.Info("Start GetVM() ...")
				if vm, err := vmHandler.GetVM(vmIID); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(vm)
				}
				res_cblogger.Info("Finish GetVM()")
			case 3:
				res_cblogger.Info("Start ListVMStatus() ...")
				if statusList, err := vmHandler.ListVMStatus(); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(statusList)
				}
				res_cblogger.Info("Finish ListVMStatus()")
			case 4:
				res_cblogger.Info("Start GetVMStatus() ...")
				if vmStatus, err := vmHandler.GetVMStatus(vmIID); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(vmStatus)
				}
				res_cblogger.Info("Finish GetVMStatus()")
			case 5:
				res_cblogger.Info("Start StartVM() ...")
				if vm, err := vmHandler.StartVM(vmReqInfo); err != nil {
					res_cblogger.Error(err)
				} else {
					vmIID.SystemId = vm.IId.SystemId
					spew.Dump(vm)
				}
				res_cblogger.Info("Finish StartVM()")
			case 6:
				res_cblogger.Info("Start RebootVM() ...")
				if vmStatus, err := vmHandler.RebootVM(vmIID); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(vmStatus)
				}
				res_cblogger.Info("Finish RebootVM()")
			case 7:
				res_cblogger.Info("Start SuspendVM() ...")
				if vmStatus, err := vmHandler.SuspendVM(vmIID); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(vmStatus)
				}
				res_cblogger.Info("Finish SuspendVM()")
			case 8:
				res_cblogger.Info("Start ResumeVM() ...")
				if vmStatus, err := vmHandler.ResumeVM(vmIID); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(vmStatus)
				}
				res_cblogger.Info("Finish ResumeVM()")
			case 9:
				res_cblogger.Info("Start TerminateVM() ...")
				if vmStatus, err := vmHandler.TerminateVM(vmIID); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(vmStatus)
				}
				res_cblogger.Info("Finish TerminateVM()")
			case 10:
				res_cblogger.Info("Start StartVM() - from MyImage ...")
				if vmStatus, err := vmHandler.StartVM(vmFromSnapshotReqInfo); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(vmStatus)
				}
				res_cblogger.Info("Finish StartVM() - from MyImage ...")
			case 11:
				res_cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

func testNLBHandlerListPrint() {
	res_cblogger.Info("Test NLBHandler")
	res_cblogger.Info("0. Print Menu")
	res_cblogger.Info("1. ListNLB()")
	res_cblogger.Info("2. GetNLB()")
	res_cblogger.Info("3. CreateNLB()")
	res_cblogger.Info("4. DeleteNLB()")
	res_cblogger.Info("5. ChangeListener()")
	res_cblogger.Info("6. ChangeVMGroupInfo()")
	res_cblogger.Info("7. AddVMs()")
	res_cblogger.Info("8. RemoveVMs()")
	res_cblogger.Info("9. GetVMGroupHealthInfo()")
	res_cblogger.Info("10. ChangeHealthCheckerInfo()")
	res_cblogger.Info("11. Exit")
}

func testNLBHandler(config ResourceConfig) {
	resourceHandler, err := getResourceHandler("nlb", config)
	if err != nil {
		res_cblogger.Error(err)
		return
	}

	nlbHandler := resourceHandler.(irs.NLBHandler)

	testNLBHandlerListPrint()
	confignlb := config.Cloudit.NLB

	vms := make([]irs.IID, len(confignlb.VMGroup.VMs))
	for i, vm := range confignlb.VMGroup.VMs {
		vms[i] = irs.IID{NameId: vm.NameId}
	}
	addvms := make([]irs.IID, len(confignlb.AddVMs))
	for i, vm := range confignlb.AddVMs {
		addvms[i] = irs.IID{NameId: vm.NameId}
	}
	removevms := make([]irs.IID, len(confignlb.RemoveVMs))
	for i, vm := range confignlb.RemoveVMs {
		removevms[i] = irs.IID{NameId: vm.NameId}
	}
	nlbIId := irs.IID{
		NameId: confignlb.IID.NameId,
	}
	nlbCreateReqInfo := irs.NLBInfo{
		IId: irs.IID{
			NameId: confignlb.IID.NameId,
		},
		//VpcIID: irs.IID{
		//	NameId: "nlb-tester-vpc",
		//},
		Type:  "PUBLIC",
		Scope: "REGION",
		Listener: irs.ListenerInfo{
			Protocol: confignlb.Listener.Protocol,
			Port:     confignlb.Listener.Port,
		},
		VMGroup: irs.VMGroupInfo{
			Port:     confignlb.VMGroup.Port,
			Protocol: confignlb.VMGroup.Protocol,
			VMs:      &vms,
		},
		HealthChecker: irs.HealthCheckerInfo{
			Protocol:  confignlb.HealthChecker.Protocol,
			Port:      confignlb.HealthChecker.Port,
			Interval:  confignlb.HealthChecker.Interval,
			Timeout:   confignlb.HealthChecker.Timeout,
			Threshold: confignlb.HealthChecker.Threshold,
		},
	}
	updateListener := irs.ListenerInfo{
		Protocol: confignlb.UpdateListener.Protocol,
		Port:     confignlb.UpdateListener.Port,
	}
	updateVMGroups := irs.VMGroupInfo{
		Protocol: confignlb.UpdateVMGroup.Protocol,
		Port:     confignlb.UpdateVMGroup.Port,
	}
	addVMs := addvms
	removeVMs := removevms

	updateHealthCheckerInfo := irs.HealthCheckerInfo{
		Protocol:  confignlb.UpdateHealthChecker.Protocol,
		Port:      confignlb.UpdateHealthChecker.Port,
		Interval:  confignlb.UpdateHealthChecker.Interval,
		Threshold: confignlb.UpdateHealthChecker.Threshold,
		Timeout:   confignlb.UpdateHealthChecker.Timeout,
	}
Loop:
	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			res_cblogger.Error(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				testNLBHandlerListPrint()
			case 1:
				res_cblogger.Info("Start ListNLB() ...")
				if list, err := nlbHandler.ListNLB(); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				res_cblogger.Info("Finish ListNLB()")
			case 2:
				res_cblogger.Info("Start GetNLB() ...")
				if vm, err := nlbHandler.GetNLB(nlbIId); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(vm)
				}
				res_cblogger.Info("Finish GetNLB()")
			case 3:
				res_cblogger.Info("Start CreateNLB() ...")
				if createInfo, err := nlbHandler.CreateNLB(nlbCreateReqInfo); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(createInfo)
				}
				res_cblogger.Info("Finish CreateNLB()")
			case 4:
				res_cblogger.Info("Start DeleteNLB() ...")
				if vmStatus, err := nlbHandler.DeleteNLB(nlbIId); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(vmStatus)
				}
				res_cblogger.Info("Finish DeleteNLB()")
			case 5:
				res_cblogger.Info("Start ChangeListener() ...")
				if nlbInfo, err := nlbHandler.ChangeListener(nlbIId, updateListener); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(nlbInfo)
				}
				res_cblogger.Info("Finish ChangeListener()")
			case 6:
				res_cblogger.Info("Start ChangeVMGroupInfo() ...")
				if info, err := nlbHandler.ChangeVMGroupInfo(nlbIId, updateVMGroups); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(info)
				}
				res_cblogger.Info("Finish ChangeVMGroupInfo()")
			case 7:
				res_cblogger.Info("Start AddVMs() ...")
				if info, err := nlbHandler.AddVMs(nlbIId, &addVMs); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(info)
				}
				res_cblogger.Info("Finish AddVMs()")
			case 8:
				res_cblogger.Info("Start RemoveVMs() ...")
				if result, err := nlbHandler.RemoveVMs(nlbIId, &removeVMs); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(result)
				}
				res_cblogger.Info("Finish RemoveVMs()")
			case 9:
				res_cblogger.Info("Start GetVMGroupHealthInfo() ...")
				if result, err := nlbHandler.GetVMGroupHealthInfo(nlbIId); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(result)
				}
				res_cblogger.Info("Finish GetVMGroupHealthInfo()")
			case 10:
				res_cblogger.Info("Start ChangeHealthCheckerInfo() ...")
				if info, err := nlbHandler.ChangeHealthCheckerInfo(nlbIId, updateHealthCheckerInfo); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(info)
				}
				res_cblogger.Info("Finish ChangeHealthCheckerInfo()")
			case 11:
				res_cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

func testDiskHandlerListPrint() {
	res_cblogger.Info("Test DiskHandler")
	res_cblogger.Info("0. Print Menu")
	res_cblogger.Info("1. ListDisk()")
	res_cblogger.Info("2. GetDisk()")
	res_cblogger.Info("3. CreateDisk()")
	res_cblogger.Info("4. DeleteDisk()")
	res_cblogger.Info("5. ChangeDiskSize()")
	res_cblogger.Info("6. AttachDisk()")
	res_cblogger.Info("7. DetachDisk()")
	res_cblogger.Info("8. Exit")
}

func testDiskHandler(config ResourceConfig) {
	resourceHandler, err := getResourceHandler("disk", config)
	if err != nil {
		res_cblogger.Error(err)
		return
	}

	diskHandler := resourceHandler.(irs.DiskHandler)

	testDiskHandlerListPrint()
	configdisk := config.Cloudit.DISK

	diskCreateReqInfo := irs.DiskInfo{
		IId: irs.IID{
			NameId: configdisk.IID.NameId,
		},
		DiskSize: configdisk.DiskSize,
	}
	diskIId := irs.IID{
		NameId: configdisk.IID.NameId,
	}
	vmIID := irs.IID{
		NameId: config.Cloudit.VM.IID.NameId,
	}
Loop:
	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			res_cblogger.Error(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				testDiskHandlerListPrint()
			case 1:
				res_cblogger.Info("Start ListDisk() ...")
				if list, err := diskHandler.ListDisk(); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				res_cblogger.Info("Finish ListDisk()")
			case 2:
				res_cblogger.Info("Start GetDisk() ...")
				if vm, err := diskHandler.GetDisk(diskIId); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(vm)
				}
				res_cblogger.Info("Finish GetDisk()")
			case 3:
				res_cblogger.Info("Start CreateDisk() ...")
				if createInfo, err := diskHandler.CreateDisk(diskCreateReqInfo); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(createInfo)
				}
				res_cblogger.Info("Finish CreateDisk()")
			case 4:
				res_cblogger.Info("Start DeleteDisk() ...")
				if vmStatus, err := diskHandler.DeleteDisk(diskIId); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(vmStatus)
				}
				res_cblogger.Info("Finish DeleteDisk()")
			case 5:
				res_cblogger.Info("Start ChangeDiskSize() [+ 10G] ...")

				// set new size
				intSize, _ := strconv.Atoi(configdisk.DiskSize)
				intSize += 10

				if nlbInfo, err := diskHandler.ChangeDiskSize(diskIId, strconv.Itoa(intSize)); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(nlbInfo)
				}
				res_cblogger.Info("Finish ChangeDiskSize() [+ 10G]")
			case 6:
				res_cblogger.Info("Start AttachDisk() ...")
				if info, err := diskHandler.AttachDisk(diskIId, vmIID); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(info)
				}
				res_cblogger.Info("Finish AttachDisk()")
			case 7:
				res_cblogger.Info("Start DetachDisk() ...")
				if info, err := diskHandler.DetachDisk(diskIId, vmIID); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(info)
				}
				res_cblogger.Info("Finish DetachDisk()")
			case 8:
				res_cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

func testMyImageHandlerListPrint() {
	res_cblogger.Info("Test MyImageHandler")
	res_cblogger.Info("0. Print Menu")
	res_cblogger.Info("1. ListMyImage()")
	res_cblogger.Info("2. GetMyImage()")
	res_cblogger.Info("3. CreateMyImage()")
	res_cblogger.Info("4. DeleteMyImage()")
	res_cblogger.Info("5. Exit")
}

func testMyImageHandler(config ResourceConfig) {
	resourceHandler, err := getResourceHandler("myimage", config)
	if err != nil {
		res_cblogger.Error(err)
		return
	}

	myImageHandler := resourceHandler.(irs.MyImageHandler)

	testMyImageHandlerListPrint()
	configmyimage := config.Cloudit.MYIMAGE

	snapshotCreateReqInfo := irs.MyImageInfo{
		IId: irs.IID{
			NameId: configmyimage.IID.NameId,
		},
		SourceVM: irs.IID{
			NameId: configmyimage.SourceVMIID.NameId,
		},
	}
	myImageIId := irs.IID{
		NameId:   configmyimage.IID.NameId,
		SystemId: configmyimage.IID.SystemId,
	}
Loop:
	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			res_cblogger.Error(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 0:
				testMyImageHandlerListPrint()
			case 1:
				res_cblogger.Info("Start ListMyImage() ...")
				if list, err := myImageHandler.ListMyImage(); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				res_cblogger.Info("Finish ListMyImage()")
			case 2:
				res_cblogger.Info("Start GetMyImage() ...")
				if vm, err := myImageHandler.GetMyImage(myImageIId); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(vm)
				}
				res_cblogger.Info("Finish GetMyImage()")
			case 3:
				res_cblogger.Info("Start CreateMyImage() ...")
				if createInfo, err := myImageHandler.SnapshotVM(snapshotCreateReqInfo); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(createInfo)
				}
				res_cblogger.Info("Finish CreateMyImage()")
			case 4:
				res_cblogger.Info("Start DeleteMyImage() ...")
				if vmStatus, err := myImageHandler.DeleteMyImage(myImageIId); err != nil {
					res_cblogger.Error(err)
				} else {
					spew.Dump(vmStatus)
				}
				res_cblogger.Info("Finish DeleteMyImage()")
			case 5:
				res_cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

func showTestHandlerInfo() {
	res_cblogger.Info("==========================================================")
	res_cblogger.Info("[Test ResourceHandler]")
	res_cblogger.Info("1. ImageHandler")
	res_cblogger.Info("2. SecurityHandler")
	res_cblogger.Info("3. VPCHandler")
	res_cblogger.Info("4. KeyPairHandler")
	res_cblogger.Info("5. VmSpecHandler")
	res_cblogger.Info("6. VmHandler")
	res_cblogger.Info("7. NLBHandler")
	res_cblogger.Info("8. DiskHandler")
	res_cblogger.Info("9. MyImageHandler")
	res_cblogger.Info("10. Exit")
	res_cblogger.Info("==========================================================")
}

func main() {

	// showTestHandlerInfo()              // ResourceHandler 테스트 정보 출력
	config := readResourceConfigFile() // config.yaml 파일 로드
	showTestHandlerInfo()
Loop:

	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			fmt.Println(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				testimageHandler(config)
				showTestHandlerInfo()
			case 2:
				testsecurityGroupHandler(config)
				showTestHandlerInfo()
			case 3:
				fmt.Println("vpc")
				testVPCHandler(config)
				showTestHandlerInfo()
			case 4:
				testKeypairHandler(config)
				showTestHandlerInfo()
			case 5:
				testvmspecHandler(config)
				showTestHandlerInfo()
			case 6:
				testVmHandler(config)
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
				fmt.Println("Exit Test ResourceHandler Program")
				break Loop
			}
		}
	}
}

// duplicateName
type ResourceConfig struct {
	Cloudit struct {
		Username         string `yaml:"user_id"`
		Password         string `yaml:"password"`
		IdentityEndpoint string `yaml:"identity_endpoint"`
		AuthToken        string `yaml:"auth_token"`
		TenantID         string `yaml:"tenant_id"`
		ClusterID        string `yaml:"cluster_id"`
		ServerId         string `yaml:"server_id"`
		VM               struct {
			IID struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"IID"`
			ImageIID struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"ImageIID"`
			VmSpecName string `yaml:"VmSpecName"`
			KeyPairIID struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"KeyPairIID"`
			VpcIID struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"VpcIID"`
			SubnetIID struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"SubnetIID"`
			SecurityGroupIIDs []struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"SecurityGroupIIDs"`
			VMUserPasswd string `yaml:"VMUserPasswd"`
			DataDiskIIDs []struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"DataDiskIIDs"`
		} `yaml:"vm"`
		VMFromMyImage struct {
			IID struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"IID"`
			ImageIID struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"ImageIID"`
			VmSpecName string `yaml:"VmSpecName"`
			KeyPairIID struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"KeyPairIID"`
			VpcIID struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"VpcIID"`
			SubnetIID struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"SubnetIID"`
			SecurityGroupIIDs []struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"SecurityGroupIIDs"`
			VMUserId     string `yaml:"VMUserId"`
			VMUserPasswd string `yaml:"VMUserPasswd"`
		} `yaml:"vmFromMyImage"`
		Resources struct {
			ImageIID struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"ImageIID"`
			VmSpecName string `yaml:"VmSpecName"`
			KeyPairIID struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"KeyPairIID"`
			VpcIID struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
				Subnets  []struct {
					SubnetIID struct {
						NameId   string `yaml:"nameId"`
						SystemId string `yaml:"systemId"`
					} `yaml:"SubnetIID"`
					IPv4_CIDR string `yaml:"IPv4_CIDR"`
				} `yaml:"Subnets"`
				AddSubnet struct {
					SubnetIID struct {
						NameId   string `yaml:"nameId"`
						SystemId string `yaml:"systemId"`
					} `yaml:"SubnetIID"`
					IPv4_CIDR string `yaml:"IPv4_CIDR"`
				} `yaml:"AddSubnet"`
				RemoveSubnet struct {
					SubnetIID struct {
						NameId   string `yaml:"nameId"`
						SystemId string `yaml:"systemId"`
					} `yaml:"SubnetIID"`
				} `yaml:"RemoveSubnet"`
			} `yaml:"VpcIID"`
			SecurityGroup struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
				Rules    []struct {
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
			} `yaml:"SecurityGroup"`
		} `yaml:"resources"`
		NLB struct {
			IID struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"IID"`
			VpcIID struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"VpcIID"`
			Listener struct {
				Protocol string `yaml:"Protocol"`
				Port     string `yaml:"Port"`
			} `yaml:"Listener"`
			VMGroup struct {
				Protocol string `yaml:"Protocol"`
				Port     string `yaml:"Port"`
				VMs      []struct {
					NameId   string `yaml:"nameId"`
					SystemId string `yaml:"systemId"`
				} `yaml:"VMs"`
			} `yaml:"VMGroup"`
			HealthChecker struct {
				Protocol  string `yaml:"Protocol"`
				Port      string `yaml:"Port"`
				Interval  int    `yaml:"Interval"`
				Timeout   int    `yaml:"Timeout"`
				Threshold int    `yaml:"Threshold"`
			} `yaml:"HealthChecker"`
			UpdateListener struct {
				Protocol string `yaml:"Protocol"`
				Port     string `yaml:"Port"`
			} `yaml:"UpdateListener"`
			UpdateVMGroup struct {
				Protocol string `yaml:"Protocol"`
				Port     string `yaml:"Port"`
			} `yaml:"UpdateVMGroup"`
			UpdateHealthChecker struct {
				Protocol  string `yaml:"Protocol"`
				Port      string `yaml:"Port"`
				Interval  int    `yaml:"Interval"`
				Timeout   int    `yaml:"Timeout"`
				Threshold int    `yaml:"Threshold"`
			} `yaml:"UpdateHealthChecker"`
			AddVMs []struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"AddVMs"`
			RemoveVMs []struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"RemoveVMs"`
		} `yaml:"nlb"`
		DISK struct {
			IID struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"IID"`
			DiskSize string `yaml:"DiskSize"`
		} `yaml:"disk"`
		MYIMAGE struct {
			IID struct {
				NameId   string `yaml:"nameId"`
				SystemId string `yaml:"systemId"`
			} `yaml:"IID"`
			SourceVMIID struct {
				NameId string `yaml:"nameId"`
			} `yaml:"SourceVMIID"`
		} `yaml:"myimage"`
	} `yaml:"cloudit"`
}

// duplicateName
func readResourceConfigFile() ResourceConfig {
	// Set Environment Value of Project Root Path4
	rootPath := os.Getenv("CBSPIDER_ROOT")
	data, err := ioutil.ReadFile(rootPath + "/cloud-control-manager/cloud-driver/drivers/cloudit/main/conf/config.yaml")
	if err != nil {
		fmt.Println(err)
	}

	var config ResourceConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		fmt.Println(err)
	}
	//spew.Dump(config)
	return config
}
