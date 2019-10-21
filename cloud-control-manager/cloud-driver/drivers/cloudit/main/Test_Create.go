package main

import (
	"fmt"
	"github.com/Azure/go-autorest/autorest/to"
	cblog "github.com/cloud-barista/cb-log"
	cidrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

func main() {
	testCreateVM()
}

func testCreateVM() {

	//리소스 핸들러 로드
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

	//imageHandler, _ := cloudConnection.CreateImageHandler()
	vNetworkHandler, _ := cloudConnection.CreateVNetworkHandler()
	securityHandler, _ := cloudConnection.CreateSecurityHandler()
	vmHandler, _ := cloudConnection.CreateVMHandler()
	publicIPHandler, _ := cloudConnection.CreatePublicIPHandler()
	//vNicHandler, _ := cloudConnection.CreateVNicHandler()

	// 1. Virtual Network 생성
	cblogger.Info("Start CreateVNetwork() ...")
	vNetReqInfo := irs.VNetworkReqInfo{Name: config.Cloudit.Resource.VirtualNetwork.Name}
	vNetwork, err := vNetworkHandler.CreateVNetwork(vNetReqInfo)
	if err != nil {
		cblogger.Error(err)
	}
	cblogger.Info("Finish CreateVNetwork()")

	// 2. Security Group 생성
	cblogger.Info("Start CreateSecurity() ...")
	secReqInfo := irs.SecurityReqInfo{Name: config.Cloudit.Resource.Security.Name}
	securityGroup, err := securityHandler.CreateSecurity(secReqInfo)
	if err != nil {
		cblogger.Error(err)
	}
	cblogger.Info("Finish CreateSecurity()")

	//todo: //SecurityGroupIds: []string(securityGroup.Id) 받는 형식??
	// 3. VM 생성
	cblogger.Info("Start Create VM ...")
	vmReqInfo := irs.VMReqInfo{
		VMName:           config.Cloudit.VMInfo.Name,
		ImageId:          config.Cloudit.VMInfo.TemplateId,
		VMSpecId:         config.Cloudit.VMInfo.SpecId,
		VirtualNetworkId: vNetwork.Id,
		SecurityGroupIds: []string{
			securityGroup.Id,
		},
		VMUserPasswd: config.Cloudit.VMInfo.RootPassword,
	}

	spew.Dump(vmReqInfo)

	vm, err := vmHandler.StartVM(vmReqInfo)
	if err != nil {
		cblogger.Error(err)
	}
	cblogger.Info("Finish Create VM")

	// VM 생성이 완료까지 대기
	var vmInfo irs.VMInfo
	vmCreated := false
	for !vmCreated {
		if status, _ := vmHandler.GetVMStatus(vm.Id); strings.ToUpper(fmt.Sprint(status)) != "RUNNING" {
			cblogger.Info("Wait for VM Create finished...")
			time.Sleep(3 * time.Second)
		} else {
			vmCreated = true
			vmInfo, _ = vmHandler.GetVM(vm.Id)
		}
	}
	spew.Dump(vmInfo)

	// 4. Public IP 생성
	cblogger.Info("Start CreatePublicIP() ...")
	publicIPReqInfo := irs.PublicIPReqInfo{
		Name: config.Cloudit.Resource.PublicIP.Name,
		//Id:   vmInfo.PrivateIP,
	}
	publicIP, err := publicIPHandler.CreatePublicIP(publicIPReqInfo)
	if err != nil {
		cblogger.Error(err)
	}
	spew.Dump(publicIP)
	cblogger.Info("Finish CreatePublicIP()")
}

func cleanResource() {

}

type Config struct {
	Cloudit struct {
		IdentityEndpoint string `yaml:"identity_endpoint"`
		Username         string `yaml:"user_id"`
		Password         string `yaml:"password"`
		TenantID         string `yaml:"tenant_id"`
		ServerId         string `yaml:"server_id"`
		AuthToken        string `yaml:"auth_token"`

		Image struct {
			Name string `yaml:"name"`
			ID   string `yaml:"id"`
		} `yaml:"image_info"`

		securityGroup struct {
			Name           string `yaml:"name"`
			ID             string `yaml:"id"`
			SecuiryGroupID string `yaml:"securitygroupid"`
		} `yaml:"sg_info"`

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
				Mac string `yaml:"mac"`
			} `yaml:"vnic_info"`
			VM struct {
				Name string `yaml:"name"`
			} `yaml:"vm"`
		} `yaml:"resource"`
		VMInfo struct {
			TemplateId   string `yaml:"template_id"`
			SpecId       string `yaml:"spec_id"`
			Name         string `yaml:"name"`
			RootPassword string `yaml:"root_password"`
			SubnetAddr   string `yaml:"subnet_addr"`
			SecGroups    string `yaml:"sec_groups"`
			Description  string `yaml:"description"`
			Protection   int    `yaml:"protection"`
		} `yaml:"vm_info"`
	} `yaml:"cloudit"`
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
