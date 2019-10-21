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

	cblogger.Info("Test ImageHandler")
	cblogger.Info("1. ListImage()")
	cblogger.Info("2. GetImage()")
	cblogger.Info("3. CreateImage()")
	cblogger.Info("4. DeleteImage()")
	cblogger.Info("5. Exit")

	var imageId string

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
				if imageList, err := imageHandler.ListImage(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(imageList)
				}
				cblogger.Info("Finish ListImage()")
			case 2:
				cblogger.Info("Start GetImage() ...")
				if _, err := imageHandler.GetImage(imageId); err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish GetImage()")
			case 3:
				cblogger.Info("Start CreateImage() ...")
				reqInfo := irs.ImageReqInfo{Name: config.Cloudit.Resource.Image.Name}
				if image, err := imageHandler.CreateImage(reqInfo); err != nil {
					cblogger.Error(err)
				} else {
					imageId = image.Id
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

//AdaptiveIP
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
				if _, err := publicIPHandler.GetPublicIP(publicIPId); err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish GetPublicIP()")
			case 3:
				cblogger.Info("Start CreatePublicIP() ...")
				reqInfo := irs.PublicIPReqInfo{Name: config.Cloudit.Resource.PublicIP.Name}
				if publicIP, err := publicIPHandler.CreatePublicIP(reqInfo); err != nil {
					cblogger.Error(err)
				} else {
					publicIPId = publicIP.Name
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

//SecurityGroup
func testSecurityHandler(config Config) {
	resourceHandler, err := getResourceHandler("security")
	if err != nil {
		cblogger.Error(err)
	}

	securityHandler := resourceHandler.(irs.SecurityHandler)

	cblogger.Info("Test securityHandler")
	cblogger.Info("1. ListSecurity()")
	cblogger.Info("2. GetSecurity()")
	cblogger.Info("3. CreateSecurity()")
	cblogger.Info("4. DeleteSecurity()")
	cblogger.Info("5. Exit")

	var securityGroupId string

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
				if _, err := securityHandler.GetSecurity(securityGroupId); err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish GetSecurity()")
			case 3:
				cblogger.Info("Start CreateSecurity() ...")
				reqInfo := irs.SecurityReqInfo{
					Name: config.Cloudit.Resource.Security.Name,
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
				if security, err := securityHandler.CreateSecurity(reqInfo); err != nil {
					cblogger.Error(err)
				} else {
					securityGroupId = security.Id
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

//Subnet
func testVNetworkHandler(config Config) {
	resourceHandler, err := getResourceHandler("vnetwork")
	if err != nil {
		cblogger.Error(err)
	}

	vNetworkHandler := resourceHandler.(irs.VNetworkHandler)

	cblogger.Info("Test vNetworkHandler")
	cblogger.Info("1. ListVNetwork()")
	cblogger.Info("2. GetVNetwork()")
	cblogger.Info("3. CreateVNetwork()")
	cblogger.Info("4. DeleteVNetwork()")
	cblogger.Info("5. Exit")

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
				if subnetList, err := vNetworkHandler.ListVNetwork(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(subnetList)
				}
				cblogger.Info("Finish ListVNetwork()")
			case 2:
				cblogger.Info("Start GetVNetwork() ...")
				if _, err := vNetworkHandler.GetVNetwork(vNetworkId); err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish GetVNetwork()")
			case 3:
				cblogger.Info("Start CreateVNetwork() ...")
				reqInfo := irs.VNetworkReqInfo{Name: config.Cloudit.Resource.VirtualNetwork.Name}
				if vNetwork, err := vNetworkHandler.CreateVNetwork(reqInfo); err != nil {
					cblogger.Error(err)
				} else {
					vNetworkId = vNetwork.Id
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

	cblogger.Info("Test vNetworkHandler")
	cblogger.Info("1. ListVNic()")
	cblogger.Info("2. GetVNic()")
	cblogger.Info("3. CreateVNic()")
	cblogger.Info("4. DeleteVNic()")
	cblogger.Info("5. Exit")

	nicId := config.Cloudit.Resource.VNic.Mac

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
				if vNicList, err := vNicHandler.ListVNic(); err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vNicList)
				}
				cblogger.Info("Finish ListVNic()")
			case 2:
				cblogger.Info("Start GetVNic() ...")
				if _, err := vNicHandler.GetVNic(nicId); err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish GetVNic()")
			case 3:
				cblogger.Info("Start CreateVNic() ...")
				/*reqInfo := nic.VNicReqInfo{
					SubnetAddr: "10.0.8.0",
					VmId:       "025e5edc-54ad-4b98-9292-6eeca4c36a6d",
					Type:       "INTERNAL",
					Secgroups: []securitygroup.SecurityGroupRules{
						{
							ID: "b2be62e7-fd29-43ff-b008-08ae736e092a",
						},
					},
					IP: "",
				}*/
				reqInfo := irs.VNicReqInfo{}
				if _, err := vNicHandler.CreateVNic(reqInfo); err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish CreateVNic()")
			case 4:
				cblogger.Info("Start DeleteVNic() ...")
				if ok, err := vNicHandler.DeleteVNic(nicId); !ok {
					cblogger.Error(err)
				}
				cblogger.Info("Finish DeleteVNic()")
			case 5:
				cblogger.Info("Exit")
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
	case "publicip":
		resourceHandler, err = cloudConnection.CreatePublicIPHandler()
	case "security":
		resourceHandler, err = cloudConnection.CreateSecurityHandler()
	case "vnetwork":
		resourceHandler, err = cloudConnection.CreateVNetworkHandler()
	case "vnic":
		resourceHandler, err = cloudConnection.CreateVNicHandler()
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
	cblogger.Info("6. Exit")
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
				cblogger.Info("Exit Test ResourceHandler Program")
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
	//spew.Dump(config)
	return config
}
