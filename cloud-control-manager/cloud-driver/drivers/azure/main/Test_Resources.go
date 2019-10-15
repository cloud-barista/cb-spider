package main

import (
	"fmt"
	cblog "github.com/cloud-barista/cb-log"
	azdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/azure"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"
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

	//imageId := config.Azure.ImageInfo.GroupName + ":" + config.Azure.ImageInfo.Name
	imageId := config.Azure.ImageInfo.GroupName + ":" + "Test-mcb-test-image"

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
				imageHandler.ListImage()
				cblogger.Info("Finish ListImage()")
			case 2:
				cblogger.Info("Start GetImage() ...")
				//imageHandler.GetImage(imageId)
				imageHandler.GetImage(imageId)
				cblogger.Info("Finish GetImage()")
			case 3:
				cblogger.Info("Start CreateImage() ...")
				reqInfo := irs.ImageReqInfo{Id: imageId}
				_, err := imageHandler.CreateImage(reqInfo)
				if err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish CreateImage()")
			case 4:
				cblogger.Info("Start DeleteImage() ...")
				imageHandler.DeleteImage(imageId)
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

	//publicIPId := config.Azure.PublicIP.GroupName + ":" + config.Azure.PublicIP.Name
	publicIPId := config.Azure.PublicIP.GroupName + ":" + "Test-mcb-test-publicIp"

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
				publicIPHandler.ListPublicIP()
				cblogger.Info("Finish ListPublicIP()")
			case 2:
				cblogger.Info("Start GetPublicIP() ...")
				publicIPHandler.GetPublicIP(publicIPId)
				cblogger.Info("Finish GetPublicIP()")
			case 3:
				cblogger.Info("Start CreatePublicIP() ...")
				reqInfo := irs.PublicIPReqInfo{Name: publicIPId}
				_, err := publicIPHandler.CreatePublicIP(reqInfo)
				if err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish CreatePublicIP()")
			case 4:
				cblogger.Info("Start DeletePublicIP() ...")
				publicIPHandler.DeletePublicIP(publicIPId)
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

	//securityGroupId := config.Azure.Security.GroupName + ":" + config.Azure.Security.Name
	securityGroupId := config.Azure.Security.GroupName + ":" + "Test-mcb-test-sg"
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
				securityHandler.ListSecurity()
				cblogger.Info("Finish ListSecurity()")
			case 2:
				cblogger.Info("Start GetSecurity() ...")
				securityHandler.GetSecurity(securityGroupId)
				cblogger.Info("Finish GetSecurity()")
			case 3:
				cblogger.Info("Start CreateSecurity() ...")
				reqInfo := irs.SecurityReqInfo{Name: securityGroupId}
				_, err := securityHandler.CreateSecurity(reqInfo)
				if err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish CreateSecurity()")
			case 4:
				cblogger.Info("Start DeleteSecurity() ...")
				securityHandler.DeleteSecurity(securityGroupId)
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

	//vNetworkId := config.Azure.VNetwork.GroupName + ":" + config.Azure.VNetwork.Name
	vNetworkId := config.Azure.VNetwork.GroupName + ":" + "Test-mcb-test-vnet"

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
				vNetworkHandler.ListVNetwork()
				cblogger.Info("Finish ListVNetwork()")
			case 2:
				cblogger.Info("Start GetVNetwork() ...")
				vNetworkHandler.GetVNetwork(vNetworkId)
				cblogger.Info("Finish GetVNetwork()")
			case 3:
				cblogger.Info("Start CreateVNetwork() ...")
				reqInfo := irs.VNetworkReqInfo{Name: vNetworkId}
				_, err := vNetworkHandler.CreateVNetwork(reqInfo)
				if err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish CreateVNetwork()")
			case 4:
				cblogger.Info("Start DeleteVNetwork() ...")
				vNetworkHandler.DeleteVNetwork(vNetworkId)
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

	//vNicId := config.Azure.VNic.GroupName + ":" + config.Azure.VNic.Name
	vNicId := config.Azure.VNic.GroupName + ":" + "Test-mcb-test-vnic"

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
				vNicHandler.ListVNic()
				cblogger.Info("Finish ListVNic()")
			case 2:
				cblogger.Info("Start GetVNic() ...")
				vNicHandler.GetVNic(vNicId)
				cblogger.Info("Finish GetVNic()")
			case 3:
				cblogger.Info("Start CreateVNic() ...")
				// TODO: vNicReqInfo 파라미터 정의
				reqInfo := irs.VNicReqInfo{Name: vNicId}
				_, err := vNicHandler.CreateVNic(reqInfo)
				if err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish CreateVNic()")
			case 4:
				cblogger.Info("Start DeleteVNic() ...")
				vNicHandler.DeleteVNic(vNicId)
				cblogger.Info("Finish DeleteVNic()")
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
