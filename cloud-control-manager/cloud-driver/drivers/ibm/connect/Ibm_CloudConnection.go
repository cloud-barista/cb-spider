package connect

import (
	cblog "github.com/cloud-barista/cb-log"
	ibms "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibm/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
	// "github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/services"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type IbmCloudConnection struct {
	CredentialInfo      idrv.CredentialInfo
	Region              idrv.RegionInfo
	AccountClient 		*services.Account
	VirtualGuestClient		*services.Virtual_Guest
	SecuritySshKeyClient *services.Security_Ssh_Key
	ProductPackageClient * services.Product_Package
	SecurityGroupClient *services.Network_SecurityGroup
	NetworkVlanClient *services.Network_Vlan
	ProductOrderClient *services.Product_Order
	NetworkSubnetClient *services.Network_Subnet
	BillingItemClient *services.Billing_Item
	LocationDatacenterClient *services.Location_Datacenter

}
func(cloudConn *IbmCloudConnection) CreateImageHandler() (irs.ImageHandler, error){
	cblogger.Info("Ibm Cloud Driver: called CreateImageHandler()!")
	vmSpecHandler := ibms.IbmImageHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region: cloudConn.Region,
		AccountClient: cloudConn.AccountClient,
		ProductPackageClient: cloudConn.ProductPackageClient,
	}
	return &vmSpecHandler,nil
}

func(cloudConn *IbmCloudConnection) CreateVMHandler() (irs.VMHandler, error){
	cblogger.Info("Ibm Cloud Driver: called CreateVMHandler()!")
	vmHandler := ibms.IbmVMHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region: cloudConn.Region,
		AccountClient: cloudConn.AccountClient,
		VirtualGuestClient: cloudConn.VirtualGuestClient,
		ProductPackageClient: cloudConn.ProductPackageClient,
		LocationDatacenterClient: cloudConn.LocationDatacenterClient,
		ProductOrderClient: cloudConn.ProductOrderClient,
		SecuritySshKeyClient: cloudConn.SecuritySshKeyClient,
	}
	return &vmHandler,nil
}

func(cloudConn *IbmCloudConnection) CreateVPCHandler() (irs.VPCHandler, error){
	cblogger.Info("Ibm Cloud Driver: called CreateVPCHandler()!")
	vpcHandler := ibms.IbmVPCHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region: cloudConn.Region,
		AccountClient: cloudConn.AccountClient,
		NetworkVlanClient: cloudConn.NetworkVlanClient,
		ProductPackageClient: cloudConn.ProductPackageClient,
		ProductOrderClient: cloudConn.ProductOrderClient,
		NetworkSubnetClient: cloudConn.NetworkSubnetClient,
		BillingItemClient: cloudConn.BillingItemClient,
		LocationDatacenterClient: cloudConn.LocationDatacenterClient,
	}
	return &vpcHandler,nil
}
func(cloudConn *IbmCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error){
	cblogger.Info("Ibm Cloud Driver: called CreateSecurityHandler()!")
	securityHandler := ibms.IbmSecurityHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region: cloudConn.Region,
		SecurityGroupClient : cloudConn.SecurityGroupClient,
		AccountClient: cloudConn.AccountClient,
	}
	return &securityHandler,nil
}
func(cloudConn *IbmCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error){
	cblogger.Info("Ibm Cloud Driver: called CreateKeyPairHandler()!")
	keyPairHandler := ibms.IbmKeyPairHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region: cloudConn.Region,
		AccountClient: cloudConn.AccountClient,
		SecuritySshKeyClient : cloudConn.SecuritySshKeyClient,
	}
	return &keyPairHandler,nil
}
func(cloudConn *IbmCloudConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error){
	cblogger.Info("Ibm Cloud Driver: called CreateVMSpecHandler()!")
	vmSpecHandler := ibms.IbmVmSpecHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region: cloudConn.Region,
		ProductPackageClient: cloudConn.ProductPackageClient,
	}
	return &vmSpecHandler,nil
}
func(cloudConn *IbmCloudConnection) IsConnected() (bool, error){
	cblogger.Info("Ibm Cloud Driver: called IsConnected()!")
	return true, nil
}
func(cloudConn *IbmCloudConnection) Close() error{
	cblogger.Info("Ibm Cloud Driver: called Close()!")
	return nil
}
