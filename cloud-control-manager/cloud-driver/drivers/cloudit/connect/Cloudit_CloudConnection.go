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

func (cloudConn *ClouditCloudConnection) CreateVNetworkHandler() (irs.VNetworkHandler, error) {
	cblogger.Info("Cloudit Cloud Driver: called CreateVNetworkHandler()!")
	vNetHandler := cirs.ClouditVNetworkHandler{cloudConn.CredentialInfo, &cloudConn.Client}
	return &vNetHandler, nil
}

func (cloudConn *ClouditCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("Cloudit Cloud Driver: called CreateImageHandler()!")
	imageHandler := cirs.ClouditImageHandler{cloudConn.CredentialInfo, &cloudConn.Client}
	return &imageHandler, nil
}

func (cloudConn ClouditCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("Cloudit Cloud Driver: called CreateSecurityHandler()!")
	securityHandler := cirs.ClouditSecurityHandler{cloudConn.CredentialInfo, &cloudConn.Client}
	return &securityHandler, nil
}

func (cloudConn *ClouditCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("Cloudit Cloud Driver: called CreateKeyPairHandler()!")
	return nil, nil
}

func (cloudConn ClouditCloudConnection) CreateVNicHandler() (irs.VNicHandler, error) {
	cblogger.Info("Cloudit Cloud Driver: called CreateVNicHandler()!")
	vNicHandler := cirs.ClouditNicHandler{cloudConn.CredentialInfo, &cloudConn.Client}
	return &vNicHandler, nil
}

func (cloudConn ClouditCloudConnection) CreatePublicIPHandler() (irs.PublicIPHandler, error) {
	cblogger.Info("Cloudit Cloud Driver: called CreatePublicIPHandler()!")
	publicIPHandler := cirs.ClouditPublicIPHandler{cloudConn.CredentialInfo, &cloudConn.Client}
	return &publicIPHandler, nil
}
func (cloudConn *ClouditCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("Cloudit Cloud Driver: called CreateVMHandler()!")
	vmHandler := cirs.ClouditVMHandler{cloudConn.CredentialInfo, &cloudConn.Client}
	return &vmHandler, nil
}

func (ClouditCloudConnection) IsConnected() (bool, error) {
	return true, nil
}
func (ClouditCloudConnection) Close() error {
	return nil
}
