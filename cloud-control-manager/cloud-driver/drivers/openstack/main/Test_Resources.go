package main

import (
	"fmt"
	"io/ioutil"
	"os"

	cblog "github.com/cloud-barista/cb-log"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	osdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack"
	_ "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack/connect"
	_ "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

func testImageHandler(config Config) {
	resourceHandler, err := getResourceHandler("image")
	if err != nil {
		panic(err)
	}

	imageHandler := resourceHandler.(irs.ImageHandler)

	cblogger.Info("Test ImageHandler")
	cblogger.Info("1. ListImage()")
	cblogger.Info("2. GetImage()")
	cblogger.Info("3. CreateImage()")
	cblogger.Info("4. DeleteImage()")
	cblogger.Info("5. Exit")

	imageId := irs.IID{
		NameId: "openshift-4hn7m-ignition", // Ubuntu 16.04
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
				cblogger.Info("Start ListImage() ...")
				if list, err := imageHandler.ListImage(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				cblogger.Info("Finish ListImage()")
			case 2:
				cblogger.Info("Start GetImage() ...")
				if imageInfo, err := imageHandler.GetImage(imageId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(imageInfo)
				}
				cblogger.Info("Finish GetImage()")
			case 3:
				cblogger.Info("Start CreateImage() ...")
				reqInfo := irs.ImageReqInfo{
					IId: irs.IID{
						NameId: config.Openstack.Image.Name,
					},
				}
				image, err := imageHandler.CreateImage(reqInfo)
				if err != nil {
					cblogger.Error(err)
				}
				imageId = image.IId
				cblogger.Info("Finish CreateImage()")
			case 4:
				cblogger.Info("Start DeleteImage() ...")
				if ok, err := imageHandler.DeleteImage(imageId); !ok {
					cblogger.Error(err)
				}
				cblogger.Info("Finish DeleteImage()")
			case 5:
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

	vpcId := irs.IID{NameId: "CB-VNet"}

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
				reqInfo := irs.VPCReqInfo{
					IId: vpcId,
					SubnetInfoList: []irs.SubnetInfo{
						{
							IId: irs.IID{
								NameId: vpcId.NameId + "-subnet-1",
							},
							IPv4_CIDR: "180.0.10.0/24",
						},
						{
							IId: irs.IID{
								NameId: vpcId.NameId + "-subnet-2",
							},
							IPv4_CIDR: "180.0.20.0/24",
						},
					},
				}

				vpcInfo, err := vpcHandler.CreateVPC(reqInfo)
				if err != nil {
					cblogger.Error(err)
				}

				vpcId = vpcInfo.IId
				spew.Dump(vpcInfo)
				cblogger.Info("Finish CreateVPC()")
			case 4:
				cblogger.Info("Start DeleteVPC() ...")
				if ok, err := vpcHandler.DeleteVPC(vpcId); !ok {
					cblogger.Error(err)
				}
				cblogger.Info("Finish DeleteVPC()")
			case 5:
				cblogger.Info("Exit")
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
		NameId: "CB-Keypair",
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

/*func testPublicIPHanlder(config Config) {
	resourceHandler, err := getResourceHandler("publicip")
	if err != nil {
		cblogger.Error(err)
	}

	publicIPHandler := resourceHandler.(irs.PublicIPHandler)

	cblogger.Info("Test PublicIPHandler")
	cblogger.Info("1. ListPublicIP()")
	cblogger.Info("2. GetPublicIP()")
	cblogger.Info("3. CreatePublicIP()")
	cblogger.Info("4. DeletePublicIP()")
	cblogger.Info("5. Exit")

	var publicIPId string

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
				cblogger.Info("Start ListPublicIP() ...")
				if publicList, err := publicIPHandler.ListPublicIP(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(publicList)
				}
				cblogger.Info("Finish ListPublicIP()")
			case 2:
				cblogger.Info("Start GetPublicIP() ...")
				if publicInfo, err := publicIPHandler.GetPublicIP(publicIPId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(publicInfo)
				}
				cblogger.Info("Finish GetPublicIP()")
			case 3:
				cblogger.Info("Start CreatePublicIP() ...")

				reqInfo := irs.PublicIPReqInfo{}
				if publicIP, err := publicIPHandler.CreatePublicIP(reqInfo); err != nil {
					cblogger.Error(err)
				} else {
					publicIPId = publicIP.Name
					spew.Dump(publicIP)
				}
				cblogger.Info("Finish CreatePublicIP()")
			case 4:
				cblogger.Info("Start DeletePublicIP() ...")
				if ok, err := publicIPHandler.DeletePublicIP(publicIPId); !ok {
					cblogger.Error(err)
				}
				cblogger.Info("Finish DeletePublicIP()")
			case 5:
				cblogger.Info("Exit")
				break Loop
			}
		}
	}
}*/

func testSecurityHandler(config Config) {
	resourceHandler, err := getResourceHandler("security")
	if err != nil {
		cblogger.Error(err)
	}

	securityHandler := resourceHandler.(irs.SecurityHandler)

	cblogger.Info("Test SecurityHandler")
	cblogger.Info("1. ListSecurity()")
	cblogger.Info("2. GetSecurity()")
	cblogger.Info("3. CreateSecurity()")
	cblogger.Info("4. DeleteSecurity()")
	cblogger.Info("5. Exit")

	securityGroupIId := irs.IID{
		NameId:   "CB-SecGroup",
		SystemId: "45a9a7be-917b-4e9f-8cbf-4aca231ff607",
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
				cblogger.Info("Start ListSecurity() ...")
				if securityList, err := securityHandler.ListSecurity(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(securityList)
				}
				cblogger.Info("Finish ListSecurity()")
			case 2:
				cblogger.Info("Start GetSecurity() ...")
				if secInfo, err := securityHandler.GetSecurity(securityGroupIId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(secInfo)
				}
				cblogger.Info("Finish GetSecurity()")
			case 3:
				cblogger.Info("Start CreateSecurity() ...")

				reqInfo := irs.SecurityReqInfo{
					IId: irs.IID{
						NameId: securityGroupIId.NameId,
					},
					SecurityRules: &[]irs.SecurityRuleInfo{
						{
							FromPort:   "22",
							ToPort:     "22",
							IPProtocol: "TCP",
							Direction:  "inbound",
						},
						{
							FromPort:   "3306",
							ToPort:     "3306",
							IPProtocol: "TCP",
							Direction:  "outbound",
						},
						{
							IPProtocol: "ICMP",
							Direction:  "outbound",
						},
					},
				}
				if securityInfo, err := securityHandler.CreateSecurity(reqInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(securityInfo)
					securityGroupIId = securityInfo.IId
				}
				cblogger.Info("Finish CreateSecurity()")
			case 4:
				cblogger.Info("Start DeleteSecurity() ...")
				if ok, err := securityHandler.DeleteSecurity(securityGroupIId); !ok {
					cblogger.Error(err)
				}
				cblogger.Info("Finish DeleteSecurity()")
			case 5:
				cblogger.Info("Exit")
				break Loop
			}
		}
	}
}

/*func testVNetworkHandler(config Config) {
	resourceHandler, err := getResourceHandler("vnetwork")
	if err != nil {
		cblogger.Error(err)
	}

	vNetworkHandler := resourceHandler.(irs.VNetworkHandler)

	cblogger.Info("Test VNetworkHandler")
	cblogger.Info("1. ListVNetwork()")
	cblogger.Info("2. GetVNetwork()")
	cblogger.Info("3. CreateVNetwork()")
	cblogger.Info("4. DeleteVNetwork()")
	cblogger.Info("5. Exit")

	vNetWorkName := "CB-VNet"
	var vNetworkId string

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
				cblogger.Info("Start ListVNetwork() ...")
				if list, err := vNetworkHandler.ListVNetwork(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				cblogger.Info("Finish ListVNetwork()")
			case 2:
				cblogger.Info("Start GetVNetwork() ...")
				if vNetInfo, err := vNetworkHandler.GetVNetwork(vNetworkId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vNetInfo)
				}
				cblogger.Info("Finish GetVNetwork()")
			case 3:
				cblogger.Info("Start CreateVNetwork() ...")

				reqInfo := irs.VNetworkReqInfo{
					Name: vNetWorkName,
				}

				if vNetworkInfo, err := vNetworkHandler.CreateVNetwork(reqInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vNetworkInfo)
					vNetworkId = vNetworkInfo.Id
				}
				cblogger.Info("Finish CreateVNetwork()")
			case 4:
				cblogger.Info("Start DeleteVNetwork() ...")
				if ok, err := vNetworkHandler.DeleteVNetwork(vNetworkId); !ok {
					cblogger.Error(err)
				}
				cblogger.Info("Finish DeleteVNetwork()")
			case 5:
				cblogger.Info("Exit")
				break Loop
			}
		}
	}
}*/

/*func testVNicHandler(config Config) {
	resourceHandler, err := getResourceHandler("vnic")
	if err != nil {
		cblogger.Error(err)
	}

	vNicHandler := resourceHandler.(irs.VNicHandler)

	cblogger.Info("Test VNicHandler")
	cblogger.Info("1. ListVNic()")
	cblogger.Info("2. GetVNic()")
	cblogger.Info("3. CreateVNic()")
	cblogger.Info("4. DeleteVNic()")
	cblogger.Info("5. Exit")

	vNicName := "CB-VNic"
	var vNicId string

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
				cblogger.Info("Start ListVNic() ...")
				if List, err := vNicHandler.ListVNic(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(List)
				}
				cblogger.Info("Finish ListVNic()")
			case 2:
				cblogger.Info("Start GetVNic() ...")
				if vNicInfo, err := vNicHandler.GetVNic(vNicId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vNicInfo)
				}
				cblogger.Info("Finish GetVNic()")
			case 3:
				cblogger.Info("Start CreateVNic() ...")

				//todo : port로 맵핑
				reqInfo := irs.VNicReqInfo{
					Name:             vNicName,
					VNetId:           "fe284dbf-e9f4-4add-a03f-9249cc30a2ac",
					SecurityGroupIds: []string{"34585b5e-5ea8-49b5-b38b-0d395689c994", "6d4085c1-e915-487d-9e83-7a5b64f27237"},
					//SubnetId:         "fe284dbf-e9f4-4add-a03f-9249cc30a2ac",
				}

				if vNicInfo, err := vNicHandler.CreateVNic(reqInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vNicInfo)
					vNicId = vNicInfo.Id
				}
				cblogger.Info("Finish CreateVNic()")
			case 4:
				cblogger.Info("Start DeleteVNic() ...")
				if ok, err := vNicHandler.DeleteVNic(vNicId); !ok {
					cblogger.Error(err)
				}
				cblogger.Info("Finish DeleteVNic()")
			case 5:
				cblogger.Info("Exit")
				break Loop
			}
		}
	}
}*/

/*func testRouterHandler(config Config) {
	resourceHandler, err := getResourceHandler("router")
	if err != nil {
		cblogger.Error(err)
	}

	routerHandler := resourceHandler.(osrs.OpenStackRouterHandler)

	cblogger.Info("Test RouterHandler")
	cblogger.Info("1. ListRouter()")
	cblogger.Info("2. GetRouter()")
	cblogger.Info("3. CreateRouter()")
	cblogger.Info("4. DeleteRouter()")
	cblogger.Info("5. AddInterface()")
	cblogger.Info("6. DeleteInterface()")
	cblogger.Info("7. Exit")

	var routerId string

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
				cblogger.Info("Start ListRouter() ...")
				routerHandler.ListRouter()
				cblogger.Info("Finish ListRouter()")
			case 2:
				cblogger.Info("Start GetRouter() ...")
				routerHandler.GetRouter(routerId)
				cblogger.Info("Finish GetRouter()")
			case 3:
				cblogger.Info("Start CreateRouter() ...")
				reqInfo := osrs.RouterReqInfo{
					Name:         config.Openstack.Router.Name,
					GateWayId:    config.Openstack.Router.GateWayId,
					AdminStateUp: config.Openstack.Router.AdminStateUp,
				}
				router, err := routerHandler.CreateRouter(reqInfo)
				if err != nil {
					cblogger.Error(err)
				}
				routerId = router.Id
				cblogger.Info("Finish CreateRouter()")
			case 4:
				cblogger.Info("Start DeleteRouter() ...")
				routerHandler.DeleteRouter(routerId)
				cblogger.Info("Finish DeleteRouter()")
			case 5:
				cblogger.Info("Start AddInterface() ...")
				reqInfo := osrs.InterfaceReqInfo{
					SubnetId: config.Openstack.Subnet.Id,
					RouterId: routerId,
				}
				_, err := routerHandler.AddInterface(reqInfo)
				if err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish AddInterface()")
			case 6:
				cblogger.Info("Start DeleteInterface() ...")
				_, err := routerHandler.DeleteInterface(routerId, config.Openstack.Subnet.Id)
				if err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish DeleteInterface()")
			case 7:
				cblogger.Info("Exit")
				break Loop
			}
		}
	}
}*/

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

	var vmSpecId string
	vmSpecId = "k8s-farm-master"

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
				cblogger.Info("Start ListVMSpec() ...")
				if list, err := vmSpecHandler.ListVMSpec(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				cblogger.Info("Finish ListVMSpec()")
			case 2:
				cblogger.Info("Start GetVMSpec() ...")
				if vmSpecInfo, err := vmSpecHandler.GetVMSpec(vmSpecId); err != nil {
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
				if vmSpecStr, err := vmSpecHandler.GetOrgVMSpec(vmSpecId); err != nil {
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

func getResourceHandler(resourceType string) (interface{}, error) {
	var cloudDriver idrv.CloudDriver
	cloudDriver = new(osdrv.OpenStackDriver)

	config := readConfigFile()
	connectionInfo := idrv.ConnectionInfo{
		CredentialInfo: idrv.CredentialInfo{
			IdentityEndpoint: config.Openstack.IdentityEndpoint,
			Username:         config.Openstack.Username,
			Password:         config.Openstack.Password,
			DomainName:       config.Openstack.DomainName,
			ProjectID:        config.Openstack.ProjectID,
		},
		RegionInfo: idrv.RegionInfo{
			Region: config.Openstack.Region,
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
	//case "vnetwork":
	//	resourceHandler, err = cloudConnection.CreateVNetworkHandler()
	case "vpc":
		resourceHandler, err = cloudConnection.CreateVPCHandler()
	//case "vnic":
	//	resourceHandler, err = cloudConnection.CreateVNicHandler()
	case "router":
		//osDriver := osdrv.OpenStackDriver{}
		//cloudConn, err := osDriver.ConnectCloud(connectionInfo)
		//if err != nil {
		//	cblogger.Error(err)
		//}
		//osCloudConn := cloudConn.(*connect.OpenStackCloudConnection)
		//resourceHandler = osrs.OpenStackRouterHandler{Client: osCloudConn.NetworkClient}
	case "vmspec":
		resourceHandler, err = cloudConnection.CreateVMSpecHandler()
	}

	if err != nil {
		return nil, err
	}
	return resourceHandler, nil
}

func showTestHandlerInfo() {
	cblogger.Info("==========================================================")
	cblogger.Info("[Test ResourceHandler]")
	cblogger.Info("1. ImageHandler")
	cblogger.Info("2. KeyPairHandler")
	//cblogger.Info("3. PublicIPHandler")
	cblogger.Info("4. SecurityHandler")
	cblogger.Info("5. VPCHandler")
	//cblogger.Info("6. VNicHandler")
	cblogger.Info("7. RouterHandler")
	cblogger.Info("8. VMSpecHandler")
	cblogger.Info("9. Exit")
	cblogger.Info("==========================================================")
}

func main() {

	showTestHandlerInfo()      // ResourceHandler 테스트 정보 출력
	config := readConfigFile() // config.yaml 파일 로드

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
				testKeyPairHandler(config)
				showTestHandlerInfo()
			case 3:
				//testPublicIPHanlder(config)
				//showTestHandlerInfo()
			case 4:
				testSecurityHandler(config)
				showTestHandlerInfo()
			case 5:
				//testVNetworkHandler(config)
				testVPCHandler(config)
				showTestHandlerInfo()
			case 6:
				//testVNicHandler(config)
				//showTestHandlerInfo()
			case 7:
				//testRouterHandler(config)
				showTestHandlerInfo()
			case 8:
				testVMSpecHandler(config)
				showTestHandlerInfo()
			case 9:
				cblogger.Info("Exit Test ResourceHandler Program")
				break Loop
			}
		}
	}
}

type Config struct {
	Openstack struct {
		DomainName       string `yaml:"domain_name"`
		IdentityEndpoint string `yaml:"identity_endpoint"`
		Password         string `yaml:"password"`
		ProjectID        string `yaml:"project_id"`
		Username         string `yaml:"username"`
		Region           string `yaml:"region"`
		VMName           string `yaml:"vm_name"`
		ImageId          string `yaml:"image_id"`
		FlavorId         string `yaml:"flavor_id"`
		NetworkId        string `yaml:"network_id"`
		SecurityGroups   string `yaml:"security_groups"`
		KeypairName      string `yaml:"keypair_name"`

		ServerId string `yaml:"server_id"`

		Image struct {
			Name string `yaml:"name"`
		} `yaml:"image_info"`

		KeyPair struct {
			Name string `yaml:"name"`
		} `yaml:"keypair_info"`

		PublicIP struct {
			Name string `yaml:"name"`
		} `yaml:"public_info"`

		SecurityGroup struct {
			Name string `yaml:"name"`
		} `yaml:"security_group_info"`

		VirtualNetwork struct {
			Name string `yaml:"name"`
		} `yaml:"vnet_info"`

		Subnet struct {
			Id string `yaml:"id"`
		} `yaml:"subnet_info"`

		Router struct {
			Name         string `yaml:"name"`
			GateWayId    string `yaml:"gateway_id"`
			AdminStateUp bool   `yaml:"adminstatup"`
		} `yaml:"router_info"`
	} `yaml:"openstack"`
}

func readConfigFile() Config {
	// Set Environment Value of Project Root Path
	rootPath := os.Getenv("CBSPIDER_ROOT")
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
