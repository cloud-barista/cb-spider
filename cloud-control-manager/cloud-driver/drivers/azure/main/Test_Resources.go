package main

import (
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

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
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

	imageId := "Canonical:UbuntuServer:18.04-LTS:18.04.201804262"

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
				reqInfo := irs.ImageReqInfo{Name: imageId}
				if imageInfo, err := imageHandler.CreateImage(reqInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(imageInfo)
				}
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

func testPublicIPHanlder(config Config) {
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

	publicIPId := "CB-PublicIP"

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
				if list, err := publicIPHandler.ListPublicIP(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				cblogger.Info("Finish ListPublicIP()")
			case 2:
				cblogger.Info("Start GetPublicIP() ...")
				if publicIpInfo, err := publicIPHandler.GetPublicIP(publicIPId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(publicIpInfo)
				}
				cblogger.Info("Finish GetPublicIP()")
			case 3:
				cblogger.Info("Start CreatePublicIP() ...")
				reqInfo := irs.PublicIPReqInfo{Name: publicIPId}
				if publicIpInfo, err := publicIPHandler.CreatePublicIP(reqInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(publicIpInfo)
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
}

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

	securityGroupId := "CB-SecGroup"

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
				if list, err := securityHandler.ListSecurity(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(list)
				}
				cblogger.Info("Finish ListSecurity()")
			case 2:
				cblogger.Info("Start GetSecurity() ...")
				if securityInfo, err := securityHandler.GetSecurity(securityGroupId); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(securityInfo)
				}
				cblogger.Info("Finish GetSecurity()")
			case 3:
				cblogger.Info("Start CreateSecurity() ...")
				reqInfo := irs.SecurityReqInfo{
					Name: securityGroupId,
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
							Direction:  "inbound",
						},
					},
				}
				if securityInfo, err := securityHandler.CreateSecurity(reqInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(securityInfo)
				}
				cblogger.Info("Finish CreateSecurity()")
			case 4:
				cblogger.Info("Start DeleteSecurity() ...")
				if ok, err := securityHandler.DeleteSecurity(securityGroupId); !ok {
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

func testVNetworkHandler(config Config) {
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

	vNetworkId := "CB-Subnet"

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
				reqInfo := irs.VNetworkReqInfo{Name: vNetworkId}
				if vNetInfo, err := vNetworkHandler.CreateVNetwork(reqInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vNetInfo)
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
}

func testVNicHandler(config Config) {
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
	cblogger.Info("5. Exit Program")

	vNicId := "CB-VNic"
	subetName := "CB-Subnet"
	segGroupId := "/subscriptions/cb592624-b77b-4a8f-bb13-0e5a48cae40f/resourceGroups/CB-GROUP/providers/Microsoft.Network/networkSecurityGroups/CB-SecGroup"
	publicIpId := "/subscriptions/cb592624-b77b-4a8f-bb13-0e5a48cae40f/resourceGroups/CB-GROUP/providers/Microsoft.Network/publicIPAddresses/CB-PublicIP"

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
				if list, err := vNicHandler.ListVNic(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(list)
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
				reqInfo := irs.VNicReqInfo{
					Name:             vNicId,
					VNetName:         subetName,
					SecurityGroupIds: []string{segGroupId},
					PublicIPid:       publicIpId,
				}
				if vNicInfo, err := vNicHandler.CreateVNic(reqInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vNicInfo)
				}
				cblogger.Info("Finish CreateVNic()")
			case 4:
				cblogger.Info("Start DeleteVNic() ...")
				if ok, err := vNicHandler.DeleteVNic(vNicId); !ok {
					cblogger.Error(err)
				}
				cblogger.Info("Finish DeleteVNic()")
			case 5:
				cblogger.Info("Exit Program")
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

	keypairName := "CB-Keypair"

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
				if vNicInfo, err := keypairHandler.GetKey(keypairName); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vNicInfo)
				}
				cblogger.Info("Finish GetKeyPair()")
			case 3:
				cblogger.Info("Start CreateKeyPair() ...")
				reqInfo := irs.KeyPairReqInfo{
					Name: keypairName,
				}
				if vNicInfo, err := keypairHandler.CreateKey(reqInfo); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vNicInfo)
				}
				cblogger.Info("Finish CreateKeyPair()")
			case 4:
				cblogger.Info("Start DeleteKeyPair() ...")
				if ok, err := keypairHandler.DeleteKey(keypairName); !ok {
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
	cloudDriver = new(azdrv.AzureDriver)

	config := readConfigFile()
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
	case "publicip":
		resourceHandler, err = cloudConnection.CreatePublicIPHandler()
	case "security":
		resourceHandler, err = cloudConnection.CreateSecurityHandler()
	case "vnetwork":
		resourceHandler, err = cloudConnection.CreateVNetworkHandler()
	case "vnic":
		resourceHandler, err = cloudConnection.CreateVNicHandler()
	case "keypair":
		resourceHandler, err = cloudConnection.CreateKeyPairHandler()
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
	cblogger.Info("2. PublicIPHandler")
	cblogger.Info("3. SecurityHandler")
	cblogger.Info("4. VNetworkHandler")
	cblogger.Info("5. VNicHandler")
	cblogger.Info("6. KeyPairHandler")
	cblogger.Info("7. Exit")
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
				testPublicIPHanlder(config)
				showTestHandlerInfo()
			case 3:
				testSecurityHandler(config)
				showTestHandlerInfo()
			case 4:
				testVNetworkHandler(config)
				showTestHandlerInfo()
			case 5:
				testVNicHandler(config)
				showTestHandlerInfo()
			case 6:
				testKeypairHandler(config)
				showTestHandlerInfo()
			case 7:
				cblogger.Info("Exit Test ResourceHandler Program")
				break Loop
			}
		}
	}
}

type Config struct {
	Azure struct {
		ClientId       string `yaml:"client_id"`
		ClientSecret   string `yaml:"client_secret"`
		TenantId       string `yaml:"tenant_id"`
		SubscriptionID string `yaml:"subscription_id"`

		GroupName string `yaml:"group_name"`
		VMName    string `yaml:"vm_name"`

		Location string `yaml:"location"`
		VMSize   string `yaml:"vm_size"`
		Image    struct {
			Publisher string `yaml:"publisher"`
			Offer     string `yaml:"offer"`
			Sku       string `yaml:"sku"`
			Version   string `yaml:"version"`
		} `yaml:"image"`
		Os struct {
			ComputeName   string `yaml:"compute_name"`
			AdminUsername string `yaml:"admin_username"`
			AdminPassword string `yaml:"admin_password"`
		} `yaml:"os"`
		Network struct {
			ID      string `yaml:"id"`
			Primary bool   `yaml:"primary"`
		} `yaml:"network"`
		ServerId string `yaml:"server_id"`

		ImageInfo struct {
			GroupName string `yaml:"group_name"`
			Name      string `yaml:"name"`
		} `yaml:"image_info"`

		PublicIP struct {
			GroupName string `yaml:"group_name"`
			Name      string `yaml:"name"`
		} `yaml:"public_ip"`

		Security struct {
			GroupName string `yaml:"group_name"`
			Name      string `yaml:"name"`
		} `yaml:"security_group"`

		VNetwork struct {
			GroupName string `yaml:"group_name"`
			Name      string `yaml:"name"`
		} `yaml:"virtual_network"`

		VNic struct {
			GroupName string `yaml:"group_name"`
			Name      string `yaml:"name"`
		} `yaml:"network_interface"`
	} `yaml:"azure"`
}

func readConfigFile() Config {
	// Set Environment Value of Project Root Path
	rootPath := os.Getenv("CBSPIDER_PATH")
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
