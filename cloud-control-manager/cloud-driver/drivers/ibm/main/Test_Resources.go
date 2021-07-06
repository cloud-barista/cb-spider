package main

import (
	"fmt"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibm"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

func getResourceHandler(resourceType string) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(ibm.IbmCloudDriver)

	config := readConfigFile()
	credentialInfo := idrv.CredentialInfo{Username: config.Ibm.Username,ApiKey:  config.Ibm.ApiKey}
	regionInfo := idrv.RegionInfo{
		Region: config.Ibm.Location,
	}
	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: credentialInfo,
		RegionInfo: regionInfo,
	}

	con, err := cloudDriver.ConnectCloud(connectionInfo)
	if err != nil {
		return nil, err
	}
	var resourceHandler interface{}

	switch resourceType {
	case "image":
		resourceHandler, err = con.CreateImageHandler()
	case "security":
		resourceHandler, err = con.CreateSecurityHandler()
	case "vpc":
		resourceHandler, err = con.CreateVPCHandler()
	case "keypair":
		resourceHandler, err = con.CreateKeyPairHandler()
	case "vmspec":
		resourceHandler, err = con.CreateVMSpecHandler()
	case "vm":
		resourceHandler, err = con.CreateVMHandler()
	}
	return resourceHandler, nil
}

func testImageHandler(config Config) {
	resourceHandler, err := getResourceHandler("image")
	if err != nil {
		cblogger.Error(err)
	}

	imageHandler := resourceHandler.(irs.ImageHandler)

	cblogger.Info("Test ImageHandler")
	cblogger.Info("1. ListImage()")
	cblogger.Info("2. GetImage()")
	cblogger.Info("3. CreateImage()")
	cblogger.Info("4. DeleteImage()")
	cblogger.Info("5. Exit")

	imageIId:=irs.IID{NameId: config.Ibm.Resources.Image.NameId, SystemId: config.Ibm.Resources.Image.SystemId}

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
				cblogger.Info("Start ListImage() ...")
				if list, err := imageHandler.ListImage(); err != nil {
					cblogger.Error(err)
				} else {
					fmt.Sprintf("len : %s",len(list))
					spew.Dump(list)
				}
				cblogger.Info("Finish ListImage()")
			case 2:
				cblogger.Info("Start GetImage() ...")
				if imageInfo, err := imageHandler.GetImage(imageIId); err != nil {
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

//SecurityGroup
func testSecurityHandler(config Config) {
	resourceHandler, err := getResourceHandler("security")
	if err != nil {
		fmt.Println(err)
	}

	securityHandler := resourceHandler.(irs.SecurityHandler)

	cblogger.Info("Test securityHandler")
	cblogger.Info("1. ListSecurity()")
	cblogger.Info("2. GetSecurity()")
	cblogger.Info("3. CreateSecurity()")
	cblogger.Info("4. DeleteSecurity()")
	cblogger.Info("5. Exit")

	securityIId := irs.IID{NameId:config.Ibm.Resources.Security.NameId,SystemId: config.Ibm.Resources.Security.SystemId}

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
					IId: irs.IID{
						NameId: config.Ibm.Resources.Security.CreateName,
					},
					SecurityRules: &[]irs.SecurityRuleInfo{
						{
							FromPort:   "22",
							ToPort:     "22",
							IPProtocol: "TCP",
							Direction:  "inbound",
						},
						{
							FromPort:   "1",
							ToPort:     "1",
							IPProtocol: "TCP",
							Direction:  "outbound",
						},
					},
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
				if ok, err := securityHandler.DeleteSecurity(irs.IID{
					NameId: config.Ibm.Resources.Security.CreateName,
				}); !ok {
					fmt.Println(err)
				}
				fmt.Println("Finish DeleteSecurity()")
			case 5:
				fmt.Println("Exit")
				break Loop
			}
		}
	}
}

func testKeyPairHandler(config Config) {
	resourceHandler, err := getResourceHandler("keypair")
	if err != nil {
		cblogger.Error(err)
	}

	keyPairHandler := resourceHandler.(irs.KeyPairHandler)

	cblogger.Info("Test KeyPairHandler")
	cblogger.Info("1. ListKey()")
	cblogger.Info("2. GetKey()")
	cblogger.Info("3. CreateKey()")
	cblogger.Info("4. DeleteKey()")
	cblogger.Info("5. Exit")

	keypairIId := irs.IID{
		NameId:  config.Ibm.Resources.KeyPair.NameId,
		SystemId: config.Ibm.Resources.KeyPair.SystemId,
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

func testVMSpecHandler(config Config) {
	resourceHandler, err := getResourceHandler("vmspec")
	if err != nil {
		panic(err)
	}

	vmSpecHandler := resourceHandler.(irs.VMSpecHandler)

	cblogger.Info("Test VMSpecHandler")
	cblogger.Info("1. ListVMSpec()")
	cblogger.Info("2. GetVMSpec()")
	cblogger.Info("3. ListOrgVMSpec()")
	cblogger.Info("4. GetOrgVMSpec()")
	cblogger.Info("5. Exit")



Loop:
	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			cblogger.Error(err)
		}

		region := config.Ibm.Location

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				cblogger.Info("Start ListVMSpec() ...")
				if list, err := vmSpecHandler.ListVMSpec(region); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				cblogger.Info("Finish ListVMSpec()")
			case 2:
				cblogger.Info("Start GetVMSpec() ...")
				if vmSpecInfo, err := vmSpecHandler.GetVMSpec(region, config.Ibm.Resources.VmSpec.NameId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vmSpecInfo)
				}
				cblogger.Info("Finish GetVMSpec()")
			case 3:
				cblogger.Info("Start ListOrgVMSpec() ...")
				if listStr, err := vmSpecHandler.ListOrgVMSpec(region); err != nil {
					cblogger.Error(err)
				} else {
					fmt.Println(listStr)
				}
				cblogger.Info("Finish ListOrgVMSpec()")
			case 4:
				cblogger.Info("Start GetOrgVMSpec() ...")
				if vmSpecStr, err := vmSpecHandler.GetOrgVMSpec(region, config.Ibm.Resources.VmSpec.NameId); err != nil {
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


func testVMHandler(config Config) {
	resourceHandler, err := getResourceHandler("vm")
	if err != nil {
		panic(err)
	}

	vmHandler := resourceHandler.(irs.VMHandler)

	cblogger.Info("Test VMSpecHandler")
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

	vmReqInfo := irs.VMReqInfo{
		IId: irs.IID{
			NameId: config.Ibm.Resources.Vm.NameId,
		},
		ImageIID: irs.IID{
			NameId: config.Ibm.Resources.Vm.ImageName,
		},
		// VPC(VLAN) 정책 결정후 사용 현재 nil 핸들링
		//VpcIID: irs.IID{
		//	NameId: config.Ibm.Resources.Vlan.NameId,
		//},
		//SubnetIID: irs.IID{
		//	NameId: config.Ibm.Resources.Vm.Subnet.NameId,
		//},
		VMSpecName: config.Ibm.Resources.Vm.VmSpecName,
		KeyPairIID: irs.IID{
			NameId: config.Ibm.Resources.KeyPair.NameId,
			SystemId: config.Ibm.Resources.KeyPair.SystemId,
		},
		SecurityGroupIIDs: []irs.IID{{
			NameId: config.Ibm.Resources.Security.NameId,
			SystemId: config.Ibm.Resources.Security.SystemId,
		}},
	}
	var getVm irs.VMInfo

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
				cblogger.Info("Start ListVM() ...")
				if list, err := vmHandler.ListVM(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				cblogger.Info("Finish ListVM()")
			case 2:
				cblogger.Info("Start GetVM() ...")
				if vm, err := vmHandler.GetVM(getVm.IId); err != nil {
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
				if vmStatus, err := vmHandler.GetVMStatus(getVm.IId); err != nil {
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
					getVm = vm
					spew.Dump(vm)
				}
				cblogger.Info("Finish StartVM()")
			case 6:
				cblogger.Info("Start RebootVM() ...")
				if vmStatus, err := vmHandler.RebootVM(getVm.IId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vmStatus)
				}
				cblogger.Info("Finish RebootVM()")
			case 7:
				cblogger.Info("Start SuspendVM() ...")
				if vmStatus, err := vmHandler.SuspendVM(getVm.IId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vmStatus)
				}
				cblogger.Info("Finish SuspendVM()")
			case 8:
				cblogger.Info("Start ResumeVM() ...")
				if vmStatus, err := vmHandler.ResumeVM(getVm.IId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vmStatus)
				}
				cblogger.Info("Finish ResumeVM()")
			case 9:
				cblogger.Info("Start TerminateVM() ...")
				if vmStatus, err := vmHandler.TerminateVM(getVm.IId); err != nil {
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
func testVPCHandler(config Config) {
	resourceHandler, err := getResourceHandler("vpc")
	if err != nil {
		cblogger.Error(err)
	}

	vpcHandler := resourceHandler.(irs.VPCHandler)

	cblogger.Info("Test VPCHandler")
	cblogger.Info("1. ListVPC()")
	cblogger.Info("2. GetVPC()")
	cblogger.Info("3. CreateVPC()")
	cblogger.Info("4. DeleteVPC()")
	cblogger.Info("5. Exit")

	vpcId := irs.IID{NameId:config.Ibm.Resources.Vlan.NameId,SystemId: config.Ibm.Resources.Vlan.SystemId}
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
				if vNetInfo, err := vpcHandler.GetVPC(vpcId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vNetInfo)
				}
				cblogger.Info("Finish GetVPC()")
			case 3:
				cblogger.Info("Start CreateVPC() ...")
				_,err :=vpcHandler.CreateVPC(irs.VPCReqInfo{})
				spew.Dump(err)
				cblogger.Info("Finish CreateVPC()")
			case 4:
				cblogger.Info("Start DeleteVPC() ...")
				//if result, err := vpcHandler.DeleteVPC(deleteVpcid); err != nil {
				//	cblogger.Error(err)
				//} else {
				//	spew.Dump(result)
				//}
				cblogger.Info("Finish DeleteVPC()")
			case 5:
				cblogger.Info("Exit")
				break Loop
			}
		}
	}
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

func readConfigFile() Config {
	// Set Environment Value of Project Root Path
	rootPath := os.Getenv("CBSPIDER_PATH")
	fmt.Println(rootPath)
	data, err := ioutil.ReadFile(rootPath + "/conf/config.yaml")
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

type Config struct{
	Ibm struct{
		Username		string `yaml:"userName"`
		ApiKey			string `yaml:"apiKey"`
		Location		string `yaml:"location"`
		Resources struct{
			Image struct{
				NameId		string 	`yaml:"nameId"`
				SystemId	string  `yaml:"systemId"`
			} `yaml:"image"`
			Security struct{
				NameId		string 	`yaml:"nameId"`
				SystemId	string  `yaml:"systemId"`
				CreateName 	string `yaml:"createName"`
			} `yaml:"security"`
			KeyPair struct{
				NameId		string 	`yaml:"nameId"`
				SystemId	string  `yaml:"systemId"`
				CreateName 	string `yaml:"createName"`
			} `yaml:"keyPair"`
			VmSpec struct{
				NameId		string 	`yaml:"nameId"`
			} `yaml:"vmSpec"`
			Vlan struct{
				NameId		string 	`yaml:"nameId"`
				SystemId	string  `yaml:"systemId"`
			}`yaml:"vlan"`
			Vm struct{
				NameId		string 	`yaml:"nameId"`
				ImageName	string 	`yaml:"imageName"`
				VmSpecName  string 	`yaml:"vmSpecName"`
				Subnet struct{
					NameId		string 	`yaml:"nameId"`
					SystemId	string  `yaml:"systemId"`
				}`yaml:"subnet"`
			} `yaml:"vm"`
		}`yaml:"resources"`
	}`yaml:"ibm"`
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
