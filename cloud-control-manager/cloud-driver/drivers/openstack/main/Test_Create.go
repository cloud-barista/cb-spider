package main

import (
	cblog "github.com/cloud-barista/cb-log"
	osdrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack"
	osconn "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack/connect"
	osrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
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

	// 리소스 핸들러 로드
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

	//imageHandler, _ := cloudConnection.CreateImageHandler()
	vNetworkHandler, _ := cloudConnection.CreateVNetworkHandler()
	securityHandler, _ := cloudConnection.CreateSecurityHandler()
	//keyPairHandler, _ := cloudConnection.CreateKeyPairHandler()
	vmHandler, _ := cloudConnection.CreateVMHandler()
	publicIPHandler, _ := cloudConnection.CreatePublicIPHandler()

	// TODO: RouterHandler 인터페이스 추가
	osConnection := cloudConnection.(*osconn.OpenStackCloudConnection)
	routerHandler := osrs.OpenStackRouterHandler{Client: osConnection.NetworkClient}

	// 1. Virtual Network, Subnet 생성
	vNetReqInfo := irs.VNetworkReqInfo{Name: config.Openstack.VirtualNetwork.Name}
	vNet, err := vNetworkHandler.CreateVNetwork(vNetReqInfo)
	if err != nil {
		cblogger.Error(err)
	}

	// 2. Router 생성 및 인터페이스 등록
	// Router 생성
	routerReqInfo := osrs.RouterReqInfo{
		Name:         config.Openstack.Router.Name,
		GateWayId:    config.Openstack.Router.GateWayId,
		AdminStateUp: config.Openstack.Router.AdminStateUp,
	}
	router, err := routerHandler.CreateRouter(routerReqInfo)
	if err != nil {
		cblogger.Error(err)
	}
	// 인터페이스 등록(연결)
	irReqInfo := osrs.InterfaceReqInfo{RouterId: router.Id, SubnetId: vNet.SubnetId}
	_, err = routerHandler.AddInterface(irReqInfo)
	if err != nil {
		cblogger.Error(err)
	}

	// 3. Security Group 생성
	sgReqInfo := irs.SecurityReqInfo{Name: config.Openstack.SecurityGroup.Name}
	sg, err := securityHandler.CreateSecurity(sgReqInfo)
	if err != nil {
		cblogger.Error(err)
	}

	// 4. KeyPair 생성
	/*keypairReqInfo := irs.KeyPairReqInfo{Name: config.Openstack.KeyPair.Name}
	keypair, err := keyPairHandler.CreateKey(keypairReqInfo)
	if err != nil {
		cblogger.Error(err)
	}*/

	// 5. VM 생성
	vmReqInfo := irs.VMReqInfo{
		Name: config.Openstack.VMName,
		ImageInfo: irs.ImageInfo{
			Id: config.Openstack.ImageId,
		},
		SpecID: config.Openstack.FlavorId,
		VNetworkInfo: irs.VNetworkInfo{
			Id: vNet.Id,
		},
		SecurityInfo: irs.SecurityInfo{
			Name: sg.Name,
		},
		KeyPairInfo: irs.KeyPairInfo{
			Name: config.Openstack.KeyPair.Name,
		},
	}

	vm, err := vmHandler.StartVM(vmReqInfo)
	if err != nil {
		cblogger.Error(err)
	}
	spew.Dump(vm)

	// 6. PublicIP 생성 및 할당
	// PublicIP 생성
	pubIPReqInfo := irs.PublicIPReqInfo{}
	publicIP, err := publicIPHandler.CreatePublicIP(pubIPReqInfo)
	if err != nil {
		cblogger.Error(err)
	}

	// 네트워크 초기화
	time.Sleep(time.Second * 10)

	// PublicIP 할당
	IP, err := publicIPHandler.GetPublicIP(publicIP.Id)
	openStackPublicIPHandler := publicIPHandler.(*osrs.OpenStackPublicIPHandler)
	_, err = openStackPublicIPHandler.AssociatePublicIP(vm.Id, IP.Id)
	if err != nil {
		cblogger.Error(err)
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

		Router struct {
			Name         string `yaml:"name"`
			GateWayId    string `yaml:"gateway_id"`
			AdminStateUp bool   `yaml:"adminstatup"`
		} `yaml:"router_info"`
	} `yaml:"openstack"`
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
