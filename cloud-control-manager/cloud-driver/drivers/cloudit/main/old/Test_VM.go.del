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

const (
	DefaultVPCName = "Default-VPC"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

func createVM(config Config, vmHandler irs.VMHandler) (irs.VMInfo, error) {

	vmReqInfo := irs.VMReqInfo{
		IId: irs.IID{
			NameId: config.Cloudit.VMInfo.Name,
		},
		ImageIID: irs.IID{
			SystemId: config.Cloudit.VMInfo.TemplateId,
			NameId:   config.Cloudit.VMInfo.TemplateName,
		},
		VMSpecName: config.Cloudit.VMInfo.SpecName,
		SubnetIID: irs.IID{
			SystemId: config.Cloudit.VMInfo.SubnetId,
			NameId:   config.Cloudit.VMInfo.SubnetName,
		},
		VMUserPasswd: config.Cloudit.VMInfo.RootPassword,
		SecurityGroupIIDs: []irs.IID{
			{
				SystemId: config.Cloudit.VMInfo.SecGroupsID,
				NameId:   config.Cloudit.VMInfo.SecGroupsName,
			},
		},
		VpcIID: irs.IID{
			NameId:   DefaultVPCName,
			SystemId: DefaultVPCName,
		},
		KeyPairIID: irs.IID{
			NameId:   config.Cloudit.VMInfo.KeypairName,
			SystemId: config.Cloudit.VMInfo.KeypairName,
		},
		// original
		/*
			VMName:           config.Cloudit.VMInfo.Name,
			ImageId:          config.Cloudit.VMInfo.TemplateId,
			VMSpecId:         config.Cloudit.VMInfo.SpecId,
			VirtualNetworkId: config.Cloudit.VMInfo.SubnetAddr,
			//SecurityGroupIds: config.Cloudit.VMInfo.SecGroups,
			VMUserPasswd: config.Cloudit.VMInfo.RootPassword,
		*/
	}

	return vmHandler.StartVM(vmReqInfo)
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

	serverId := irs.IID{
		NameId: config.Cloudit.VMInfo.Name,
	}

	for {
		var commandNum int
		inputCnt, err := fmt.Scan(&commandNum)
		if err != nil {
			fmt.Println(err)
		}

		if inputCnt == 1 {
			switch commandNum {
			case 1:
				fmt.Println("Start List VM ...")
				vmList, _ := vmHandler.ListVM()
				for i, vm := range vmList {
					fmt.Println("[", i, "] ")
					spew.Dump(vm)
				}
				fmt.Println("Finish List VM")
			case 2:
				fmt.Println("Start Get VM ...")
				vmInfo, _ := vmHandler.GetVM(serverId)
				spew.Dump(vmInfo)
				fmt.Println("Finish Get VM")
			case 3:
				fmt.Println("Start List VMStatus ...")
				vmStatusList, _ := vmHandler.ListVMStatus()
				for i, vmStatus := range vmStatusList {
					fmt.Println("[", i, "] ", *vmStatus)
				}
				fmt.Println("Finish List VMStatus")
			case 4:
				fmt.Println("Start Get VMStatus ...")
				vmStatus, _ := vmHandler.GetVMStatus(serverId)
				fmt.Println(vmStatus)
				fmt.Println("Finish Get VMStatus")
			case 5:
				fmt.Println("Start Create VM ...")
				if vm, err := createVM(config, vmHandler); err != nil {
					fmt.Println(err)
				} else {
					spew.Dump(vm)
					serverId = vm.IId
				}
				fmt.Println("Finish Create VM")
			case 6:
				fmt.Println("Start Suspend VM ...")
				vmHandler.SuspendVM(serverId)
				fmt.Println("Finish Suspend VM")
			case 7:
				fmt.Println("Start Resume  VM ...")
				vmHandler.ResumeVM(serverId)
				fmt.Println("Finish Resume VM")
			case 8:
				fmt.Println("Start Reboot  VM ...")
				vmHandler.RebootVM(serverId)
				fmt.Println("Finish Reboot VM")
			case 9:
				fmt.Println("Start Terminate  VM ...")
				vmHandler.TerminateVM(serverId)
				fmt.Println("Finish Terminate VM")
			}
		}
	}
}

func getVMHandler() (irs.VMHandler, error) {
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
	Cloudit struct {
		Username         string `yaml:"user_id"`
		Password         string `yaml:"password"`
		IdentityEndpoint string `yaml:"identity_endpoint"`
		AuthToken        string `yaml:"auth_token"`
		TenantID         string `yaml:"tenant_id"`
		ServerId         string `yaml:"server_id"`
		VMInfo           struct {
			Name          string `yaml:"name"`
			TemplateId    string `yaml:"template_id"`
			TemplateName  string `yaml:"template_name"`
			SpecId        string `yaml:"spec_id"`
			SpecName      string `yaml:"spec_name"`
			SubnetId      string `yaml:"subnet_id"`
			SubnetName    string `yaml:"subnet_name"`
			RootPassword  string `yaml:"root_password"`
			SubnetAddr    string `yaml:"subnet_addr"`
			SecGroupsID   string `yaml:"sec_groups_id"`
			SecGroupsName string `yaml:"sec_groups_name"`
			KeypairName   string `yaml:"keypair_name"`
			Description   string `yaml:"description"`
			Protection    int    `yaml:"protection"`
		} `yaml:"vm_info"`
	} `yaml:"cloudit"`
}

func readConfigFile() Config {
	// Set Environment Value of Project Root Path
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
	return config
}
