// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by hyokyung.kim@innogrid.co.kr, 2019.07.

package main

import (
	"fmt"
	cblog "github.com/cloud-barista/cb-log"
	azdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/azure"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
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

// Test VM Handler Functions (Get VM Info, VM Status)
func getVMInfo() {
	vmHandler, err := setVMHandler()
	if err != nil {
		cblogger.Error(err)
	}
	config := readConfigFile()

	// Get VM List
	vmList := vmHandler.ListVM()
	for i, vm := range vmList {
		cblogger.Info("[", i, "] ")
		cblogger.Info(vm)
	}

	vmId := config.Azure.GroupName + ":" + config.Azure.VMName

	// Get VM Info
	vmInfo := vmHandler.GetVM(vmId)
	cblogger.Info(vmInfo)

	// Get VM Status List
	vmStatusList := vmHandler.ListVMStatus()
	for i, vmStatus := range vmStatusList {
		cblogger.Info("[", i, "] ", *vmStatus)
	}

	// Get VM Status
	vmStatus := vmHandler.GetVMStatus(vmId)
	cblogger.Info(vmStatus)
}

// Test VM Lifecycle Management (Suspend/Resume/Reboot/Terminate)
func handleVM() {
	vmHandler, err := setVMHandler()
	if err != nil {
		cblogger.Error(err)
	}
	config := readConfigFile()

	cblogger.Info("VM LifeCycle Management")
	cblogger.Info("1. Suspend VM")
	cblogger.Info("2. Resume VM")
	cblogger.Info("3. Reboot VM")
	cblogger.Info("4. Terminate VM")

	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			cblogger.Error(err)
		}

		vmId := config.Azure.GroupName + ":" + config.Azure.VMName

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				cblogger.Info("Start Suspend VM ...")
				vmHandler.SuspendVM(vmId)
				cblogger.Info("Finish Suspend VM")
			case 2:
				cblogger.Info("Start Resume  VM ...")
				vmHandler.ResumeVM(vmId)
				cblogger.Info("Finish Resume VM")
			case 3:
				cblogger.Info("Start Reboot  VM ...")
				vmHandler.RebootVM(vmId)
				cblogger.Info("Finish Reboot VM")
			case 4:
				cblogger.Info("Start Terminate  VM ...")
				vmHandler.TerminateVM(vmId)
				cblogger.Info("Finish Terminate VM")
			}
		}
	}
}

// Test VM Deployment
func createVM() {
	cblogger.Info("Start Create VM ...")
	vmHandler, err := setVMHandler()
	if err != nil {
		cblogger.Error(err)
	}
	config := readConfigFile()

	vmName := config.Azure.GroupName + ":" + config.Azure.VMName
	imageId := config.Azure.Image.Publisher + ":" + config.Azure.Image.Offer + ":" + config.Azure.Image.Sku + ":" + config.Azure.Image.Version
	vmReqInfo := irs.VMReqInfo{
		Name: vmName,
		ImageInfo: irs.ImageInfo{
			Id: imageId,
		},
		SpecID: config.Azure.VMSize,
		VNetworkInfo: irs.VNetworkInfo{
			Id: config.Azure.Network.ID,
		},
		LoginInfo: irs.LoginInfo{
			AdminUsername: config.Azure.Os.AdminUsername,
			AdminPassword: config.Azure.Os.AdminPassword,
		},
	}

	vm, err := vmHandler.StartVM(vmReqInfo)
	if err != nil {
		cblogger.Error(err)
	}
	cblogger.Info(vm)
	cblogger.Info("Finish Create VM")
}

func testImageHandler() {
	imageHandler, err := setImageHandler()
	if err != nil {
		cblogger.Error(err)
	}
	config := readConfigFile()

	cblogger.Info("Test ImageHandler")
	cblogger.Info("1. ListImage()")
	cblogger.Info("2. GetImage()")
	cblogger.Info("3. CreateImage()")
	cblogger.Info("4. DeleteImage()")
	cblogger.Info("5. Exit Program")

Loop:
	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			cblogger.Error(err)
		}

		imageId := config.Azure.ImageInfo.GroupName + ":" + config.Azure.ImageInfo.Name

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				cblogger.Info("Start ListImage() ...")
				imageHandler.ListImage()
				cblogger.Info("Finish ListImage()")
			case 2:
				cblogger.Info("Start GetImage() ...")
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
				cblogger.Info("Exit Program")
				break Loop
			}
		}
	}
}

func testPublicIPHandler() {
	publicIPHandler, err := setPublicIPHandler()
	if err != nil {
		cblogger.Error(err)
	}
	config := readConfigFile()

	cblogger.Info("Test PublicIPHandler")
	cblogger.Info("1. ListPublicIP()")
	cblogger.Info("2. GetPublicIP()")
	cblogger.Info("3. CreatePublicIP()")
	cblogger.Info("4. DeletePublicIP()")
	cblogger.Info("5. Exit Program")

Loop:
	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			cblogger.Error(err)
		}

		publicIPId := config.Azure.PublicIP.GroupName + ":" + config.Azure.PublicIP.Name

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
				reqInfo := irs.PublicIPReqInfo{Id: publicIPId}
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
				cblogger.Info("Exit Program")
				break Loop
			}
		}
	}
}

func testSecurityHandler() {
	securityHandler, err := setSecurityHandler()
	if err != nil {
		cblogger.Error(err)
	}
	config := readConfigFile()

	cblogger.Info("Test SecurityHandler")
	cblogger.Info("1. ListSecurity()")
	cblogger.Info("2. GetSecurity()")
	cblogger.Info("3. CreateSecurity()")
	cblogger.Info("4. DeleteSecurity()")
	cblogger.Info("5. Exit Program")

Loop:
	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			cblogger.Error(err)
		}

		securityId := config.Azure.Security.GroupName + ":" + config.Azure.Security.Name

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				cblogger.Info("Start ListSecurity() ...")
				securityHandler.ListSecurity()
				cblogger.Info("Finish ListSecurity()")
			case 2:
				cblogger.Info("Start GetSecurity() ...")
				securityHandler.GetSecurity(securityId)
				cblogger.Info("Finish GetSecurity()")
			case 3:
				cblogger.Info("Start CreateSecurity() ...")
				reqInfo := irs.SecurityReqInfo{Id: securityId}
				_, err := securityHandler.CreateSecurity(reqInfo)
				if err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish CreateSecurity()")
			case 4:
				cblogger.Info("Start DeleteSecurity() ...")
				securityHandler.DeleteSecurity(securityId)
				cblogger.Info("Finish DeleteSecurity()")
			case 5:
				cblogger.Info("Exit Program")
				break Loop
			}
		}
	}
}

func testVNetworkHandler() {
	vNetHandler, err := setVNetHandler()
	if err != nil {
		cblogger.Error(err)
	}
	config := readConfigFile()

	cblogger.Info("Test VNetworkHandler")
	cblogger.Info("1. ListVNetwork()")
	cblogger.Info("2. GetVNetwork()")
	cblogger.Info("3. CreateVNetwork()")
	cblogger.Info("4. DeleteVNetwork()")
	cblogger.Info("5. Exit Program")

Loop:
	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			cblogger.Error(err)
		}

		networkId := config.Azure.VNetwork.GroupName + ":" + config.Azure.VNetwork.Name

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				cblogger.Info("Start ListVNetwork() ...")
				vNetHandler.ListVNetwork()
				cblogger.Info("Finish ListVNetwork()")
			case 2:
				cblogger.Info("Start GetVNetwork() ...")
				vNetHandler.GetVNetwork(networkId)
				cblogger.Info("Finish GetVNetwork()")
			case 3:
				cblogger.Info("Start CreateVNetwork() ...")
				reqInfo := irs.VNetworkReqInfo{Id: networkId}
				_, err := vNetHandler.CreateVNetwork(reqInfo)
				if err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish CreateVNetwork()")
			case 4:
				cblogger.Info("Start DeleteVNetwork() ...")
				vNetHandler.DeleteVNetwork(networkId)
				cblogger.Info("Finish DeleteVNetwork()")
			case 5:
				cblogger.Info("Exit Program")
				break Loop
			}
		}
	}
}

func testVNicHandler() {
	vNicHandler, err := setVNicHandler()
	if err != nil {
		cblogger.Error(err)
	}
	config := readConfigFile()

	cblogger.Info("Test VNicHandler")
	cblogger.Info("1. ListVNic()")
	cblogger.Info("2. GetVNic()")
	cblogger.Info("3. CreateVNic()")
	cblogger.Info("4. DeleteVNic()")
	cblogger.Info("5. Exit Program")

Loop:
	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			cblogger.Error(err)
		}

		vNicId := config.Azure.VNic.GroupName + ":" + config.Azure.VNic.Name

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
				reqInfo := irs.VNicReqInfo{Id: vNicId}
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

func setVMHandler() (irs.VMHandler, error) {
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

	cloudConnection, err := cloudDriver.ConnectCloud(connectionInfo)
	if err != nil {
		return nil, err
	}
	vmHandler, err := cloudConnection.CreateVMHandler()
	if err != nil {
		return nil, err
	}
	return vmHandler, nil
}

func setImageHandler() (irs.ImageHandler, error) {
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

	cloudConnection, err := cloudDriver.ConnectCloud(connectionInfo)
	if err != nil {
		return nil, err
	}
	imageHandler, err := cloudConnection.CreateImageHandler()
	if err != nil {
		return nil, err
	}
	return imageHandler, nil
}

func setPublicIPHandler() (irs.PublicIPHandler, error) {
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

	cloudConnection, err := cloudDriver.ConnectCloud(connectionInfo)
	if err != nil {
		return nil, err
	}
	publicIPHandler, err := cloudConnection.CreatePublicIPHandler()
	if err != nil {
		return nil, err
	}
	return publicIPHandler, nil
}

func setSecurityHandler() (irs.SecurityHandler, error) {
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

	cloudConnection, err := cloudDriver.ConnectCloud(connectionInfo)
	if err != nil {
		return nil, err
	}
	securityHandler, err := cloudConnection.CreateSecurityHandler()
	if err != nil {
		return nil, err
	}
	return securityHandler, nil
}

func setVNetHandler() (irs.VNetworkHandler, error) {
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

	cloudConnection, err := cloudDriver.ConnectCloud(connectionInfo)
	if err != nil {
		return nil, err
	}
	vNetHandler, err := cloudConnection.CreateVNetworkHandler()
	if err != nil {
		return nil, err
	}
	return vNetHandler, nil
}

func setVNicHandler() (irs.VNicHandler, error) {
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

	cloudConnection, err := cloudDriver.ConnectCloud(connectionInfo)
	if err != nil {
		return nil, err
	}
	vNicHandler, err := cloudConnection.CreateVNicHandler()
	if err != nil {
		return nil, err
	}
	return vNicHandler, nil
}

func main() {
	// Test VM Handler
	//getVMInfo()
	//handleVM()
	//createVM()

	// Teset Resource Handler
	//testImageHandler()
	//testPublicIPHandler()
	//testSecurityHandler()
	//testVNetworkHandler()
	testVNicHandler()
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
	data, err := ioutil.ReadFile(rootPath + "/config/config.yaml")
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
