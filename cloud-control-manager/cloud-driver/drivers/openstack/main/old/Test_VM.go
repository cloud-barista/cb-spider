package main

import (
	"fmt"
	"io/ioutil"
	"os"

	cblog "github.com/cloud-barista/cb-log"
	osdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

// Create Instance
func createVM(config Config, vmHandler irs.VMHandler) (*string, error) {

	vmName := "CB-Vm"
	imageId := "062ec06f-ead4-453b-91e4-e8c859b93afc"
	imageName := "ubuntu-18.04"
	vmSpecName := "m1.medium"
	vpcId := "af8010c9-4769-4545-9770-a31e9bb8b645"
	securityId := "45a9a7be-917b-4e9f-8cbf-4aca231ff607"
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

	fmt.Println("Test VMHandler")
	fmt.Println("1. List VM")
	fmt.Println("2. Get VM")
	fmt.Println("3. List VMStatus")
	fmt.Println("4. Get VMStatus")
	fmt.Println("5. Create VM")
	fmt.Println("6. Suspend VM")
	fmt.Println("7. Resume VM")
	fmt.Println("8. Reboot VM")
	fmt.Println("9. Terminate VM")
	fmt.Println("10. Exit")

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
				fmt.Println("Start List VM ...")
				vmList, err := vmHandler.ListVM()
				if err != nil {
					cblogger.Error(err)
				} else {
					for i, vm := range vmList {
						fmt.Println("[", i, "] ")
						spew.Dump(vm)
					}
				}
				fmt.Println("Finish List VM")
			case 2:
				fmt.Println("Start Get VM ...")
				vmInfo, err := vmHandler.GetVM(vmIId)
				if err != nil {
					cblogger.Error(err)
				} else {
					spew.Dump(vmInfo)
				}
				fmt.Println("Finish Get VM")
			case 3:
				fmt.Println("Start List VMStatus ...")
				vmStatusList, err := vmHandler.ListVMStatus()
				if err != nil {
					cblogger.Error(err)
				} else {
					for i, vmStatus := range vmStatusList {
						fmt.Println("[", i, "] ", *vmStatus)
					}
				}
				fmt.Println("Finish List VMStatus")
			case 4:
				fmt.Println("Start Get VMStatus ...")
				vmStatus, err := vmHandler.GetVMStatus(vmIId)
				if err != nil {
					cblogger.Error(err)
				} else {
					fmt.Println(vmStatus)
				}
				fmt.Println("Finish Get VMStatus")
			case 5:
				fmt.Println("Start Create VM ...")
				vmId, err := createVM(config, vmHandler)
				if err != nil {
					cblogger.Error(err)
				} else {
					vmIId.SystemId = *vmId
				}
				fmt.Println("Finish Create VM")
			case 6:
				fmt.Println("Start Suspend VM ...")
				_, err := vmHandler.SuspendVM(vmIId)
				if err != nil {
					cblogger.Error(err)
				}
				fmt.Println("Finish Suspend VM")
			case 7:
				fmt.Println("Start Resume  VM ...")
				_, err := vmHandler.ResumeVM(vmIId)
				if err != nil {
					cblogger.Error(err)
				}
				fmt.Println("Finish Resume VM")
			case 8:
				fmt.Println("Start Reboot  VM ...")
				_, err := vmHandler.RebootVM(vmIId)
				if err != nil {
					cblogger.Error(err)
				}
				fmt.Println("Finish Reboot VM")
			case 9:
				fmt.Println("Start Terminate  VM ...")
				_, err := vmHandler.TerminateVM(vmIId)
				if err != nil {
					cblogger.Error(err)
				}
				fmt.Println("Finish Terminate VM")
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
