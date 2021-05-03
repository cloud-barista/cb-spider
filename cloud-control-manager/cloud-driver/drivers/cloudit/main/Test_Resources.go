package main

import (
	"fmt"
	cblog "github.com/cloud-barista/cb-log"
	cidrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit"
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

func testImageHandler(config Config) {

	var imageHandler irs.ImageHandler
	if resourceHandler, err := getResourceHandler("image"); err != nil {
		panic(err)
	} else {
		imageHandler = resourceHandler.(irs.ImageHandler)
	}

	fmt.Println("Test ImageHandler")
	fmt.Println("1. ListImage()")
	fmt.Println("2. GetImage()")
	fmt.Println("3. CreateImage()")
	fmt.Println("4. DeleteImage()")
	fmt.Println("5. Exit")

	//var imageId string
	imageId := irs.IID{
		NameId:   "CentOS-7",
		SystemId: "a846af3b-5d80-4182-b38e-5501ad9f78f4",
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
			case 1:
				fmt.Println("Start ListImage() ...")
				if imageList, err := imageHandler.ListImage(); err != nil {
					fmt.Println(err)
				} else {
					spew.Dump(imageList)
				}
				fmt.Println("Finish ListImage()")
			case 2:
				fmt.Println("Start GetImage() ...")
				if imageInfo, err := imageHandler.GetImage(imageId); err != nil {
					fmt.Println(err)
				} else {
					spew.Dump(imageInfo)
				}
				fmt.Println("Finish GetImage()")
			case 3:
				fmt.Println("Start CreateImage() ...")
				reqInfo := irs.ImageReqInfo{
					IId: irs.IID{
						NameId: config.Cloudit.Resource.Image.Name,
					},
				}
				if image, err := imageHandler.CreateImage(reqInfo); err != nil {
					fmt.Println(err)
				} else {
					imageId = image.IId
				}
				fmt.Println("Finish CreateImage()")
			case 4:
				fmt.Println("Start DeleteImage() ...")
				if ok, err := imageHandler.DeleteImage(imageId); !ok {
					fmt.Println(err)
				}
				fmt.Println("Finish DeleteImage()")
			case 5:
				fmt.Println("Exit")
				break Loop
			}
		}
	}
}

/*
//AdaptiveIP
func testPublicIPHanlder(config Config) {
	resourceHandler, err := getResourceHandler("publicip")
	if err != nil {
		fmt.Println(err)
	}

	publicIPHandler := resourceHandler.(irs.PublicIPHandler)

	fmt.Println("Test PublicIPHandler")
	fmt.Println("1. ListPublicIP()")
	fmt.Println("2. GetPublicIP()")
	fmt.Println("3. CreatePublicIP()")
	fmt.Println("4. DeletePublicIP()")
	fmt.Println("5. Exit")

	var publicIPId string

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
				fmt.Println("Start ListPublicIP() ...")
				if publicList, err := publicIPHandler.ListPublicIP(); err != nil {
					fmt.Println(err)
				} else {
					spew.Dump(publicList)
				}
				fmt.Println("Finish ListPublicIP()")
			case 2:
				fmt.Println("Start GetPublicIP() ...")
				if _, err := publicIPHandler.GetPublicIP(publicIPId); err != nil {
					fmt.Println(err)
				}
				fmt.Println("Finish GetPublicIP()")
			case 3:
				fmt.Println("Start CreatePublicIP() ...")
				reqInfo := irs.PublicIPReqInfo{Name: config.Cloudit.Resource.PublicIP.Name}
				if publicIP, err := publicIPHandler.CreatePublicIP(reqInfo); err != nil {
					fmt.Println(err)
				} else {
					publicIPId = publicIP.Name
				}
				fmt.Println("Finish CreatePublicIP()")
			case 4:
				fmt.Println("Start DeletePublicIP() ...")
				if ok, err := publicIPHandler.DeletePublicIP(publicIPId); !ok {
					fmt.Println(err)
				}
				fmt.Println("Finish DeletePublicIP()")
			case 5:
				fmt.Println("Exit")
				break Loop
			}
		}
	}
}
*/
//SecurityGroup
func testSecurityHandler(config Config) {
	resourceHandler, err := getResourceHandler("security")
	if err != nil {
		fmt.Println(err)
	}

	securityHandler := resourceHandler.(irs.SecurityHandler)

	fmt.Println("Test securityHandler")
	fmt.Println("1. ListSecurity()")
	fmt.Println("2. GetSecurity()")
	fmt.Println("3. CreateSecurity()")
	fmt.Println("4. DeleteSecurity()")
	fmt.Println("5. Exit")

	securityGroupId := irs.IID{
		NameId: config.Cloudit.Resource.Security.Name,
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
				if secGroupInfo, err := securityHandler.GetSecurity(securityGroupId); err != nil {
					fmt.Println(err)
				} else {
					spew.Dump(secGroupInfo)
				}
				fmt.Println("Finish GetSecurity()")
			case 3:
				fmt.Println("Start CreateSecurity() ...")
				reqInfo := irs.SecurityReqInfo{
					IId: irs.IID{
						NameId: config.Cloudit.Resource.Security.Name,
					},
					SecurityRules: &[]irs.SecurityRuleInfo{
						{
							FromPort:   "22",
							ToPort:     "22",
							IPProtocol: "TCP",
							Direction:  "inbound",
						},
						{
							FromPort:   "0",
							ToPort:     "0",
							IPProtocol: "TCP",
							Direction:  "outbound",
						},
					},
				}
				security, err := securityHandler.CreateSecurity(reqInfo)
				if err != nil {
					fmt.Println(err)
				}
				securityGroupId = security.IId
				fmt.Println("Finish CreateSecurity()")
			case 4:
				fmt.Println("Start DeleteSecurity() ...")
				if ok, err := securityHandler.DeleteSecurity(securityGroupId); !ok {
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

//Subnet
func testVPCHandler(config Config) {
	resourceHandler, err := getResourceHandler("vpc")
	if err != nil {
		fmt.Println(err)
	}

	vpcHandler := resourceHandler.(irs.VPCHandler)

	fmt.Println("Test testVPCHandler")
	fmt.Println("1. ListVPC()")
	fmt.Println("2. GetVPC()")
	fmt.Println("3. CreateVPC()")
	fmt.Println("4. DeleteVPC()")
	fmt.Println("5. Exit")

	vpcId := irs.IID{NameId: "Default-VPC", SystemId: "Default-VPC"}

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
				fmt.Println("Start ListVPC() ...")
				if subnetList, err := vpcHandler.ListVPC(); err != nil {
					fmt.Println(err)
				} else {
					spew.Dump(subnetList)
				}
				fmt.Println("Finish ListVPC()")
			case 2:
				fmt.Println("Start GetVPC() ...")
				if subnetList, err := vpcHandler.GetVPC(vpcId); err != nil {
					fmt.Println(err)
				} else {
					spew.Dump(subnetList)
				}
				fmt.Println("Finish GetVPC()")
			case 3:
				fmt.Println("Start CreateVPC() ...")
				reqInfo := irs.VPCReqInfo{
					IId: vpcId,
					SubnetInfoList: []irs.SubnetInfo{
						{
							IId: irs.IID{
								NameId: vpcId.NameId + "-subnet-1",
							},
						},
						{
							IId: irs.IID{
								NameId: vpcId.NameId + "-subnet-2",
							},
						},
					},
				}

				vpcInfo, err := vpcHandler.CreateVPC(reqInfo)
				if err != nil {
					fmt.Println(err)
				}

				vpcId = vpcInfo.IId
				spew.Dump(vpcInfo)
				fmt.Println("Finish CreateVPC()")

			case 4:
				fmt.Println("Start DeleteVPC() ...")
				if ok, err := vpcHandler.DeleteVPC(vpcId); !ok {
					fmt.Println(err)
				}
				fmt.Println("Finish DeleteVPC()")
			case 5:
				fmt.Println("Exit")
				break Loop
			}
		}
	}
}

/*
func testVNicHandler(config Config) {
	resourceHandler, err := getResourceHandler("vnic")
	if err != nil {
		fmt.Println(err)
	}

	vNicHandler := resourceHandler.(irs.VNicHandler)

	fmt.Println("Test vNetworkHandler")
	fmt.Println("1. ListVNic()")
	fmt.Println("2. GetVNic()")
	fmt.Println("3. CreateVNic()")
	fmt.Println("4. DeleteVNic()")
	fmt.Println("5. Exit")

	nicId := config.Cloudit.Resource.VNic.Mac

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
				fmt.Println("Start ListVNic() ...")
				if vNicList, err := vNicHandler.ListVNic(); err != nil {
					fmt.Println(err)
				} else {
					spew.Dump(vNicList)
				}
				fmt.Println("Finish ListVNic()")
			case 2:
				fmt.Println("Start GetVNic() ...")
				if _, err := vNicHandler.GetVNic(nicId); err != nil {
					fmt.Println(err)
				}
				fmt.Println("Finish GetVNic()")
			case 3:
				fmt.Println("Start CreateVNic() ...")
				//reqInfo := nic.VNicReqInfo{
				//	SubnetAddr: "10.0.8.0",
				//	VmId:       "025e5edc-54ad-4b98-9292-6eeca4c36a6d",
				//	Type:       "INTERNAL",
				//	Secgroups: []securitygroup.SecurityGroupRules{
				//		{
				//			ID: "b2be62e7-fd29-43ff-b008-08ae736e092a",
				//		},
				//	},
				//	IP: "",
				//}
				reqInfo := irs.VNicReqInfo{}
				if _, err := vNicHandler.CreateVNic(reqInfo); err != nil {
					fmt.Println(err)
				}
				fmt.Println("Finish CreateVNic()")
			case 4:
				fmt.Println("Start DeleteVNic() ...")
				if ok, err := vNicHandler.DeleteVNic(nicId); !ok {
					fmt.Println(err)
				}
				fmt.Println("Finish DeleteVNic()")
			case 5:
				fmt.Println("Exit")
				break Loop
			}
		}
	}
}
*/
func testVmSpecHandler(config Config) {
	resourceHandler, err := getResourceHandler("vmspec")
	if err != nil {
		fmt.Println(err)
	}
	vmSpecHandler := resourceHandler.(irs.VMSpecHandler)

	fmt.Println("Test VmSpecHandler")
	fmt.Println("1. ListVmSpec()")
	fmt.Println("2. GetVmSpec()")
	fmt.Println("3. ListOrgVmSpec()")
	fmt.Println("4. GetOrgVmSpec()")
	fmt.Println("9. Exit")

	region := config.Cloudit.TenantID
	vmSpecName := "large-4"

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
				fmt.Println("Start ListVmSpec() ...")
				if list, err := vmSpecHandler.ListVMSpec(region); err != nil {
					fmt.Println(err)
				} else {
					spew.Dump(list)
				}
				fmt.Println("Finish ListVmSpec()")
			case 2:
				fmt.Println("Start GetVmSpec() ...")
				if vmSpec, err := vmSpecHandler.GetVMSpec(region, vmSpecName); err != nil {
					fmt.Println(err)
				} else {
					spew.Dump(vmSpec)
				}
				fmt.Println("Finish GetVmSpec()")
			case 3:
				fmt.Println("Start ListOrgVmSpec() ...")
				if listStr, err := vmSpecHandler.ListOrgVMSpec(region); err != nil {
					fmt.Println(err)
				} else {
					fmt.Println(listStr)
				}
				fmt.Println("Finish ListOrgVmSpec()")
			case 4:
				fmt.Println("Start GetOrgVmSpec() ...")
				if vmSpecStr, err := vmSpecHandler.GetOrgVMSpec(region, vmSpecName); err != nil {
					fmt.Println(err)
				} else {
					fmt.Println(vmSpecStr)
				}
				fmt.Println("Finish GetOrgVmSpec()")
			case 9:
				fmt.Println("Exit")
				break Loop
			}
		}
	}
}

func testKeypairHandler(config Config) {
	resourceHandler, err := getResourceHandler("keypair")
	if err != nil {
		cblogger.Error(err)
	}

	keypairHandler := resourceHandler.(irs.KeyPairHandler)

	cblogger.Info("Test KeypairHandler")
	cblogger.Info("1. ListKeyPair()")
	cblogger.Info("2. GetKeyPair()")
	cblogger.Info("3. CreateKeyPair()")
	cblogger.Info("4. DeleteKeyPair()")
	cblogger.Info("5. Exit Program")

	iid := irs.IID{
		NameId:   "CB-Keypair",
		SystemId: "CB-Keypair",
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
				cblogger.Info("Start ListKeyPair() ...")
				if list, err := keypairHandler.ListKey(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				cblogger.Info("Finish ListKeyPair()")
			case 2:
				cblogger.Info("Start GetKeyPair() ...")
				if vNicInfo, err := keypairHandler.GetKey(iid); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vNicInfo)
				}
				cblogger.Info("Finish GetKeyPair()")
			case 3:
				cblogger.Info("Start CreateKeyPair() ...")
				reqInfo := irs.KeyPairReqInfo{
					IId: iid,
				}
				if vNicInfo, err := keypairHandler.CreateKey(reqInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vNicInfo)
				}
				cblogger.Info("Finish CreateKeyPair()")
			case 4:
				cblogger.Info("Start DeleteKeyPair() ...")
				if ok, err := keypairHandler.DeleteKey(iid); !ok {
					cblogger.Error(err)
				}
				cblogger.Info("Finish DeleteKeyPair()")
			case 5:
				cblogger.Info("Exit Program")
				break Loop
			}
		}
	}
}

func getResourceHandler(resourceType string) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(cidrv.ClouditDriver)

	config := readConfigFile()
	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			IdentityEndpoint: config.Cloudit.IdentityEndpoint,
			Username:         config.Cloudit.Username,
			Password:         config.Cloudit.Password,
			TenantId:         config.Cloudit.TenantID,
			AuthToken:        config.Cloudit.AuthToken,
		},
	}

	cloudConnection, _ := cloudDriver.ConnectCloud(connectionInfo)

	var resourceHandler interface{}
	var err error

	switch resourceType {
	case "image":
		resourceHandler, err = cloudConnection.CreateImageHandler()
	case "keypair":
		resourceHandler, err = cloudConnection.CreateKeyPairHandler()
	//case "publicip":
	//	resourceHandler, err = cloudConnection.CreatePublicIPHandler()
	case "security":
		resourceHandler, err = cloudConnection.CreateSecurityHandler()
	case "vpc":
		resourceHandler, err = cloudConnection.CreateVPCHandler()
	//case "vnic":
	//	resourceHandler, err = cloudConnection.CreateVNicHandler()
	case "vmspec":
		resourceHandler, err = cloudConnection.CreateVMSpecHandler()
	}

	if err != nil {
		return nil, err
	}
	return resourceHandler, nil
}

func showTestHandlerInfo() {
	fmt.Println("==========================================================")
	fmt.Println("[Test ResourceHandler]")
	fmt.Println("1. ImageHandler")
	fmt.Println("2. PublicIPHandler x")
	fmt.Println("3. SecurityHandler")
	fmt.Println("4. VPCHandler")
	fmt.Println("5. VNicHandler x")
	fmt.Println("6. VMSpecHandler")
	fmt.Println("7. KeyPairHandler")
	fmt.Println("8. Exit")
	fmt.Println("==========================================================")
}

func main() {

	showTestHandlerInfo()      // ResourceHandler 테스트 정보 출력
	config := readConfigFile() // config.yaml 파일 로드

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
				testImageHandler(config)
				showTestHandlerInfo()
			case 2:
				//testPublicIPHanlder(config)
				//showTestHandlerInfo()
			case 3:
				testSecurityHandler(config)
				showTestHandlerInfo()
			case 4:
				testVPCHandler(config)
				showTestHandlerInfo()
			case 5:
				//testVNicHandler(config)
				//showTestHandlerInfo()
			case 6:
				testVmSpecHandler(config)
				showTestHandlerInfo()
			case 7:
				testKeypairHandler(config)
				showTestHandlerInfo()
			case 8:
				fmt.Println("Exit Test ResourceHandler Program")
				break Loop
			}
		}
	}
}

type Config struct {
	Cloudit struct {
		IdentityEndpoint string `yaml:"identity_endpoint"`
		Username         string `yaml:"user_id"`
		Password         string `yaml:"password"`
		TenantID         string `yaml:"tenant_id"`
		ServerId         string `yaml:"server_id"`
		AuthToken        string `yaml:"auth_token"`

		Resource struct {
			Image struct {
				Name string `yaml:"name"`
			} `yaml:"image"`

			PublicIP struct {
				Name string `yaml:"name"`
			} `yaml:"public_ip"`

			Security struct {
				Name string `yaml:"name"`
			} `yaml:"security_group"`

			VirtualNetwork struct {
				Name string `yaml:"name"`
			} `yaml:"vnet_info"`

			VNic struct {
				VMID string `yaml:"vm_id"`
				Mac  string `yaml:"mac"`
			} `yaml:"vnic_info"`
		} `yaml:"resource"`
	} `yaml:"cloudit"`
}

func readConfigFile() Config {
	// Set Environment Value of Project Root Path4
	rootPath := os.Getenv("CBSPIDER_ROOT")
	data, err := ioutil.ReadFile(rootPath + "/conf/config.yaml")
	if err != nil {
		fmt.Println(err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		fmt.Println(err)
	}
	//spew.Dump(config)
	return config
}
