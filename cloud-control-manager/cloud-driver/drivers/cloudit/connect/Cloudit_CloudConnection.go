// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by hyokyung.kim@innogrid.com, 2019.08.

package connect

import (
	"errors"
	cblog "github.com/cloud-barista/cb-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	cirs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type ClouditCloudConnection struct {
	CredentialInfo idrv.CredentialInfo
	Client         client.RestClient
}

func (cloudConn *ClouditCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("Cloudit Cloud Driver: called CreateImageHandler()!")
	imageHandler := cirs.ClouditImageHandler{CredentialInfo: cloudConn.CredentialInfo, Client: &cloudConn.Client}
	return &imageHandler, nil
}

func (cloudConn *ClouditCloudConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	cblogger.Info("Cloudit Cloud Driver: called CreateVNetworkHandler()!")
	vNetHandler := cirs.ClouditVPCHandler{CredentialInfo: cloudConn.CredentialInfo, Client: &cloudConn.Client}
	return &vNetHandler, nil
}

func (cloudConn ClouditCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("Cloudit Cloud Driver: called CreateSecurityHandler()!")
	securityHandler := cirs.ClouditSecurityHandler{CredentialInfo: cloudConn.CredentialInfo, Client: &cloudConn.Client}
	return &securityHandler, nil
}

func (cloudConn *ClouditCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("Cloudit Cloud Driver: called CreateKeyPairHandler()!")
	keypairHandler := cirs.ClouditKeyPairHandler{CredentialInfo: cloudConn.CredentialInfo, Client: &cloudConn.Client}
	return &keypairHandler, nil
}

/*func (cloudConn ClouditCloudConnection) CreateVNicHandler() (irs.VNicHandler, error) {
	cblogger.Info("Cloudit Cloud Driver: called CreateVNicHandler()!")
	vNicHandler := cirs.ClouditNicHandler{cloudConn.CredentialInfo, &cloudConn.Client}
	return &vNicHandler, nil
}*/

/*func (cloudConn ClouditCloudConnection) CreatePublicIPHandler() (irs.PublicIPHandler, error) {
	cblogger.Info("Cloudit Cloud Driver: called CreatePublicIPHandler()!")
	publicIPHandler := cirs.ClouditPublicIPHandler{cloudConn.CredentialInfo, &cloudConn.Client}
	return &publicIPHandler, nil
}*/

func (cloudConn *ClouditCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("Cloudit Cloud Driver: called CreateVMHandler()!")
	vmHandler := cirs.ClouditVMHandler{CredentialInfo: cloudConn.CredentialInfo, Client: &cloudConn.Client}
	return &vmHandler, nil
}

func (cloudConn *ClouditCloudConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	cblogger.Info("Cloudit Cloud Driver: called CreateVMSpecHandler()!")
	vmSpecHandler := cirs.ClouditVMSpecHandler{CredentialInfo: cloudConn.CredentialInfo, Client: &cloudConn.Client}
	return &vmSpecHandler, nil
}

func (cloudConn *ClouditCloudConnection) CreateNLBHandler() (irs.NLBHandler, error) {
	cblogger.Info("Cloudit Cloud Driver: called CreateNLBHandler()!")
	nlbHandler := cirs.ClouditNLBHandler{CredentialInfo: cloudConn.CredentialInfo, Client: &cloudConn.Client}
	return &nlbHandler, nil
}

func (ClouditCloudConnection) IsConnected() (bool, error) {
	return true, nil
}
func (ClouditCloudConnection) Close() error {
	return nil
}

func (cloudConn *ClouditCloudConnection) CreateDiskHandler() (irs.DiskHandler, error) {
	cblogger.Info("Cloudit Cloud Driver: called CreateDiskHandler()!")
	diskHandler := cirs.ClouditDiskHandler{CredentialInfo: cloudConn.CredentialInfo, Client: &cloudConn.Client}
	return &diskHandler, nil
}

func (cloudConn *ClouditCloudConnection) CreateClusterHandler() (irs.ClusterHandler, error) {
	return nil, errors.New("Cloudit Driver: not implemented")
}

func (cloudConn *ClouditCloudConnection) CreateMyImageHandler() (irs.MyImageHandler, error) {
	cblogger.Info("Cloudit Cloud Driver: called CreateMyImageHandler()!")
	myImageHandler := cirs.ClouditMyImageHandler{CredentialInfo: cloudConn.CredentialInfo, Client: &cloudConn.Client}
	return &myImageHandler, nil
}


func (cloudConn *ClouditCloudConnection) CreateAnyCallHandler() (irs.AnyCallHandler, error) {
	return nil, errors.New("Cloudit Driver: not implemented")
}

