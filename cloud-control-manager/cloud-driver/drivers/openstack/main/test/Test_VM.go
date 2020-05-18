package main

import (
	"fmt"
	cblog "github.com/cloud-barista/cb-log"
	osdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack"
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
func createVM(config Config, vmHandler irs.VMHandler) (*string, error) {

	vmName := "CBVm"
	imageId := "c14a9728-eb03-4813-9e1a-8f57fe62b4fb"
	imageName := "ubuntu16.04"
	vmSpecName := "nano.1"
	vpcId := "deae15ce-9789-4a4a-8f84-885e01ce8c1d"
	//subnetId := "03e48ba2-c0ac-4e2f-8d9f-dd4fb1615d5a"
	securityId := "392b779d-4c41-4f7e-aeb2-a5673108df86"
	keypairName := "CB-Keypair"

	vmReqInfo := irs.VMReqInfo{
		IId:               irs.IID{NameId: vmName},
		ImageIID:          irs.IID{SystemId: imageId, NameId: imageName},
		VpcIID:            irs.IID{SystemId: vpcId},
		SecurityGroupIIDs: []irs.IID{{SystemId: securityId}},
		VMSpecName:        vmSpecName,
		KeyPairIID:        irs.IID{NameId: keypairName},
	}

	/*vmReqInfo := irs.VMReqInfo{
		VMName:           config.Openstack.VMName,
		ImageId:          config.Openstack.ImageId,
		VMSpecId:         config.Openstack.FlavorId,
		VirtualNetworkId: config.Openstack.NetworkId,
		SecurityGroupIds: []string{config.Openstack.SecurityGroups},
		KeyPairName:      config.Openstack.KeypairName,
	}*/

	vm, err := vmHandler.StartVM(vmReqInfo)
	if err != nil {
		return nil, err
	}
	return &vm.IId.SystemId, nil
}

func testVMHandler() {
	vmHandler, err := getVMHandler()
	if err != nil {
		panic(err)
	}
	config := readConfigFile()

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

	//vmIId := irs.IID{SystemId: "352d5aca-78d4-4eee-99d0-10404d9ed197"}
	vmIId := irs.IID{SystemId: "5cdf19b2-ece4-4ec7-99f5-31f8b7db05e6"}
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
				vmInfo, err := vmHandler.GetVM(vmIId)
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
				vmStatus, err := vmHandler.GetVMStatus(vmIId)
				if err != nil {
					cblogger.Error(err)
				} else {
					cblogger.Info(vmStatus)
				}
				cblogger.Info("Finish Get VMStatus")
			case 5:
				cblogger.Info("Start Create VM ...")
				vmId, err := createVM(config, vmHandler)
				if err != nil {
					cblogger.Error(err)
				} else {
					vmIId.SystemId = *vmId
				}
				cblogger.Info("Finish Create VM")
			case 6:
				cblogger.Info("Start Suspend VM ...")
				_, err := vmHandler.SuspendVM(vmIId)
				if err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish Suspend VM")
			case 7:
				cblogger.Info("Start Resume  VM ...")
				_, err := vmHandler.ResumeVM(vmIId)
				if err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish Resume VM")
			case 8:
				cblogger.Info("Start Reboot  VM ...")
				_, err := vmHandler.RebootVM(vmIId)
				if err != nil {
					cblogger.Error(err)
				}
				cblogger.Info("Finish Reboot VM")
			case 9:
				cblogger.Info("Start Terminate  VM ...")
				_, err := vmHandler.TerminateVM(vmIId)
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

		ServerId   string `yaml:"server_id"`
		PublicIPID string `yaml:"public_ip_id"`

		Image struct {
			Name string `yaml:"name"`
		} `yaml:"image_info"`

		KeyPair struct {
			Name string `yaml:"name"`
		} `yaml:"keypair_info"`

		SecurityGroup struct {
			Name string `yaml:"name"`
		} `yaml:"security_group_info"`

		VirtualNetwork struct {
			Name string `yaml:"name"`
		} `yaml:"vnet_info"`
	} `yaml:"openstack"`
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
