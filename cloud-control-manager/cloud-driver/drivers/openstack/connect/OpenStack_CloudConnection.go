// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by CB-Spider Team, 2019.06.

package connect

import (
	"errors"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/gophercloud/gophercloud"
	"github.com/sirupsen/logrus"

	osrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

// OpenStackCloudConnection modified by powerkim, 2019.07.29
type OpenStackCloudConnection struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	ComputeClient  *gophercloud.ServiceClient
	ImageClient    *gophercloud.ServiceClient
	NetworkClient  *gophercloud.ServiceClient
	Volume2Client  *gophercloud.ServiceClient
	Volume3Client  *gophercloud.ServiceClient
	NLBClient      *gophercloud.ServiceClient
	IdentityClient *gophercloud.ServiceClient
}

func (cloudConn *OpenStackCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("OpenStack Cloud Driver: called CreateImageHandler()!")
	imageHandler := osrs.OpenStackImageHandler{Client: cloudConn.ComputeClient, ImageClient: cloudConn.ImageClient}
	return &imageHandler, nil
}

func (cloudConn *OpenStackCloudConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	cblogger.Info("OpenStack Cloud Driver: called CreateVPCHandler()!")
	vpcHandler := osrs.OpenStackVPCHandler{Client: cloudConn.NetworkClient, VMClient: cloudConn.ComputeClient}
	return &vpcHandler, nil
}

func (cloudConn OpenStackCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("OpenStack Cloud Driver: called CreateSecurityHandler()!")
	securityHandler := osrs.OpenStackSecurityHandler{Client: cloudConn.ComputeClient, NetworkClient: cloudConn.NetworkClient}
	return &securityHandler, nil
}

func (cloudConn *OpenStackCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("OpenStack Cloud Driver: called CreateKeyPairHandler()!")
	keypairHandler := osrs.OpenStackKeyPairHandler{Client: cloudConn.ComputeClient}
	return &keypairHandler, nil
}

func (cloudConn *OpenStackCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("OpenStack Cloud Driver: called CreateVMHandler()!")
	vmHandler := osrs.OpenStackVMHandler{Region: cloudConn.Region, ComputeClient: cloudConn.ComputeClient, NetworkClient: cloudConn.NetworkClient, Volume2Client: cloudConn.Volume2Client, Volume3Client: cloudConn.Volume3Client}
	return &vmHandler, nil
}

func (cloudConn *OpenStackCloudConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	cblogger.Info("OpenStack Cloud Driver: called CreateVMSpecHandler()!")
	vmSpecHandler := osrs.OpenStackVMSpecHandler{Region: cloudConn.Region, Client: cloudConn.ComputeClient}
	return &vmSpecHandler, nil
}

func (cloudConn *OpenStackCloudConnection) CreateNLBHandler() (irs.NLBHandler, error) {
	cblogger.Info("OpenStack Cloud Driver: called CreateNLBHandler()!")
	nlbHandler := osrs.OpenStackNLBHandler{CredentialInfo: cloudConn.CredentialInfo, Region: cloudConn.Region, VMClient: cloudConn.ComputeClient, NetworkClient: cloudConn.NetworkClient, NLBClient: cloudConn.NLBClient}
	return &nlbHandler, nil
}

func (cloudConn *OpenStackCloudConnection) CreateDiskHandler() (irs.DiskHandler, error) {
	return nil, errors.New("OpenStack Driver: not implemented")
}

func (cloudConn *OpenStackCloudConnection) IsConnected() (bool, error) {
	return true, nil
}
func (cloudConn *OpenStackCloudConnection) Close() error {
	return nil
}
