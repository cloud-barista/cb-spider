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

// Create Instance
func createVM(config Config, vmHandler irs.VMHandler) {

	vmName := "CBVm"
	//imageId := "/subscriptions/cb592624-b77b-4a8f-bb13-0e5a48cae40f/resourceGroups/CB-GROUP/providers/Microsoft.Compute/images/CB-IMG"
	imageId := "Canonical:UbuntuServer:18.04-LTS:18.04.201804262"
	vmSpecId := "Standard_B1ls"
	networkInterfaceId := "/subscriptions/cb592624-b77b-4a8f-bb13-0e5a48cae40f/resourceGroups/CB-GROUP/providers/Microsoft.Network/networkInterfaces/CB-VNic"
	keypairName := "CB-Keypair"

	vmReqInfo := irs.VMReqInfo{
		VMName:             vmName,
		ImageId:            imageId,
		VMSpecId:           vmSpecId,
		NetworkInterfaceId: networkInterfaceId,
		KeyPairName:        keypairName,
	}

	vm, err := vmHandler.StartVM(vmReqInfo)
	if err != nil {
		panic(err)
	}
	spew.Dump(vm)
}

func testVMHandler() {
	vmHandler, err := getVMHandler()
	if err != nil {
		panic(err)
	}
	config := readConfigFile()

	cblogger.Info("==========================================================")
	cblogger.Info("Test VMHandler")
	cblogger.Info("1. List VM")
	cblogger.Info("2. Get VM")
	cblogger.Info("3. List VMStatus")
	cblogger.Info("4. Get VMStatus")
	cblogger.Info("5. Create VM")
	cblogger.Info("6. Suspend VM")
	cblogger.Info("7. Resume VM")
	cblogger.Info("8. Reboot VM")
	cblogger.Info("9. Terminate VM")
	cblogger.Info("10. Exit")
	cblogger.Info("==========================================================")

	vmId := "CBVm"

	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			cblogger.Error(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				cblogger.Info("Start List VM ...")
				vmList, err := vmHandler.ListVM()
				if err != nil {
					cblogger.Error(err)
				} else {
					for i, vm := range vmList {
						cblogger.Info("[", i, "] ")
						spew.Dump(vm)
					}
				}
				cblogger.Info("Finish List VM")
			case 2:
				cblogger.Info("Start Get VM ...")
				vmInfo, err := vmHandler.GetVM(vmId)
				if err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vmInfo)
				}
				cblogger.Info("Finish Get VM")
			case 3:
				cblogger.Info("Start List VMStatus ...")
				vmStatusList, err := vmHandler.ListVMStatus()
				if err != nil {
					cblogger.Error(err)
				} else {
					for i, vmStatus := range vmStatusList {
						cblogger.Info("[", i, "] ", *vmStatus)
					}
				}
				cblogger.Info("Finish List VMStatus")
			case 4:
				cblogger.Info("Start Get VMStatus ...")
				vmStatus, err := vmHandler.GetVMStatus(vmId)
				if err != nil {
					cblogger.Error(err)
				} else {
					cblogger.Info(vmStatus)
				}
				cblogger.Info("Finish Get VMStatus")
			case 5:
				cblogger.Info("Start Create VM ...")
				createVM(config, vmHandler)
				cblogger.Info("Finish Create VM")
			case 6:
				cblogger.Info("Start Suspend VM ...")
				err := vmHandler.SuspendVM(vmId)
				if err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish Suspend VM")
			case 7:
				cblogger.Info("Start Resume  VM ...")
				err := vmHandler.ResumeVM(vmId)
				if err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish Resume VM")
			case 8:
				cblogger.Info("Start Reboot  VM ...")
				err := vmHandler.RebootVM(vmId)
				if err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish Reboot VM")
			case 9:
				cblogger.Info("Start Terminate  VM ...")
				err := vmHandler.TerminateVM(vmId)
				if err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish Terminate VM")
			}
		}
	}
}

func getVMHandler() (irs.VMHandler, error) {
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
	vmHandler, err := cloudConnection.CreateVMHandler()
	if err != nil {
		return nil, err
	}
	return vmHandler, nil
}

func main() {
	testVMHandler()
}

type Config struct {
	Azure struct {
		ClientId       string `yaml:"client_id"`
		ClientSecret   string `yaml:"client_secret"`
		TenantId       string `yaml:"tenant_id"`
		SubscriptionID string `yaml:"subscription_id"`

		GroupName string `yaml:"group_name"`
		VMName    string `yaml:"vm_name"`

		AdminUsername string `yaml:"admin_username"`
		AdminPassword string `yaml:"admin_password"`

		Location string `yaml:"location"`
		VMSize   string `yaml:"vm_size"`
		Image    struct {
			Publisher string `yaml:"publisher"`
			Offer     string `yaml:"offer"`
			Sku       string `yaml:"sku"`
			Version   string `yaml:"version"`
		} `yaml:"image"`
		Os struct {
			ComputeName string `yaml:"compute_name"`
		} `yaml:"os"`
		Nic struct {
			ID string `yaml:"id"`
		} `yaml:"nic"`
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
