package ibm

import (
	ibmcon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibm/connect"
	ibms "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibm/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/session"
	"time"
)

const (
	cspTimeout time.Duration = 6000
)

type IbmCloudDriver struct{}

func (driver *IbmCloudDriver) GetDriverVersion() string {
	return "Ibm DRIVER Version 1.0"
}

func (IbmCloudDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	drvCapabilityInfo.ImageHandler = true
	drvCapabilityInfo.VPCHandler = true
	drvCapabilityInfo.SecurityHandler = true
	drvCapabilityInfo.KeyPairHandler = true
	drvCapabilityInfo.VNicHandler = false
	drvCapabilityInfo.PublicIPHandler = false
	drvCapabilityInfo.VMHandler = true
	drvCapabilityInfo.VMSpecHandler = true

	return drvCapabilityInfo
}

func (driver *IbmCloudDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {

	ibms.InitLog()

	AccountClient, err := getAccountClient(connectionInfo)
	if err != nil {
		return nil, err
	}
	VirtualGuestClient, err := getVirtualGuestClient(connectionInfo)
	if err != nil {
		return nil, err
	}
	ProductPackageClient, err := getProductPackageClient(connectionInfo)
	if err != nil {
		return nil, err
	}
	SecuritySshKeyClient, err := getSecuritySshKeyClient(connectionInfo)
	if err != nil {
		return nil, err
	}
	SecurityGroupClient, err := getSecurityGroupClient(connectionInfo)
	if err != nil {
		return nil, err
	}
	NetworkVlanClient, err := getNetworkVlanClient(connectionInfo)
	if err != nil {
		return nil, err
	}
	NetworkSubnetClient, err := getNetworkSubnetClient(connectionInfo)
	if err != nil {
		return nil, err
	}
	LocationDatacenterClient, err := getLocationDatacenterClient(connectionInfo)
	if err != nil {
		return nil, err
	}
	ProductOrderClient, err := getProductOrderClient(connectionInfo)
	if err != nil {
		return nil, err
	}
	BillingItemClient, err := getBillingItemClient(connectionInfo)
	if err != nil {
		return nil, err
	}

	iConn := ibmcon.IbmCloudConnection{
		CredentialInfo:           connectionInfo.CredentialInfo,
		Region:                   connectionInfo.RegionInfo,
		AccountClient:            AccountClient,
		VirtualGuestClient:       VirtualGuestClient,
		SecuritySshKeyClient:     SecuritySshKeyClient,
		ProductPackageClient:     ProductPackageClient,
		SecurityGroupClient:      SecurityGroupClient,
		NetworkVlanClient:        NetworkVlanClient,
		NetworkSubnetClient:      NetworkSubnetClient,
		LocationDatacenterClient: LocationDatacenterClient,
		BillingItemClient:        BillingItemClient,
		ProductOrderClient:       ProductOrderClient,
	}
	return &iConn, nil
}

func getAccountClient(connectionInfo idrv.ConnectionInfo) (*services.Account, error) {
	sess := session.New(connectionInfo.CredentialInfo.Username, connectionInfo.CredentialInfo.ApiKey)
	service := services.GetAccountService(sess)
	return &service, nil
}

func getVirtualGuestClient(connectionInfo idrv.ConnectionInfo) (*services.Virtual_Guest, error) {
	sess := session.New(connectionInfo.CredentialInfo.Username, connectionInfo.CredentialInfo.ApiKey)
	service := services.GetVirtualGuestService(sess)
	return &service, nil
}

func getSecuritySshKeyClient(connectionInfo idrv.ConnectionInfo) (*services.Security_Ssh_Key, error) {
	sess := session.New(connectionInfo.CredentialInfo.Username, connectionInfo.CredentialInfo.ApiKey)
	service := services.GetSecuritySshKeyService(sess)
	return &service, nil
}

func getSecurityGroupClient(connectionInfo idrv.ConnectionInfo) (*services.Network_SecurityGroup, error) {
	sess := session.New(connectionInfo.CredentialInfo.Username, connectionInfo.CredentialInfo.ApiKey)
	service := services.GetNetworkSecurityGroupService(sess)
	return &service, nil
}

func getProductPackageClient(connectionInfo idrv.ConnectionInfo) (*services.Product_Package, error) {
	sess := session.New(connectionInfo.CredentialInfo.Username, connectionInfo.CredentialInfo.ApiKey)
	service := services.GetProductPackageService(sess)
	return &service, nil
}

func getNetworkSubnetClient(connectionInfo idrv.ConnectionInfo) (*services.Network_Subnet, error) {
	sess := session.New(connectionInfo.CredentialInfo.Username, connectionInfo.CredentialInfo.ApiKey)
	service := services.GetNetworkSubnetService(sess)
	return &service, nil
}

func getNetworkVlanClient(connectionInfo idrv.ConnectionInfo) (*services.Network_Vlan, error) {
	sess := session.New(connectionInfo.CredentialInfo.Username, connectionInfo.CredentialInfo.ApiKey)
	service := services.GetNetworkVlanService(sess)
	return &service, nil
}

func getLocationDatacenterClient(connectionInfo idrv.ConnectionInfo) (*services.Location_Datacenter, error) {
	sess := session.New(connectionInfo.CredentialInfo.Username, connectionInfo.CredentialInfo.ApiKey)
	service := services.GetLocationDatacenterService(sess)
	return &service, nil
}

func getProductOrderClient(connectionInfo idrv.ConnectionInfo) (*services.Product_Order, error) {
	sess := session.New(connectionInfo.CredentialInfo.Username, connectionInfo.CredentialInfo.ApiKey)
	service := services.GetProductOrderService(sess)
	return &service, nil
}

func getBillingItemClient(connectionInfo idrv.ConnectionInfo) (*services.Billing_Item, error) {
	sess := session.New(connectionInfo.CredentialInfo.Username, connectionInfo.CredentialInfo.ApiKey)
	service := services.GetBillingItemService(sess)
	return &service, nil
}

var CloudDriver IbmCloudDriver
