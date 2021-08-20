package connect

import (
	"context"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	cblog "github.com/cloud-barista/cb-log"
	ibms "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibmcloud-vpc/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type IbmCloudConnection struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	VpcService     *vpcv1.VpcV1
	Ctx            context.Context
}

func (cloudConn *IbmCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateImageHandler()!")
	imageHandler := ibms.IbmImageHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
	}
	return &imageHandler, nil
}

func (cloudConn *IbmCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateVMHandler()!")
	vmHandler := ibms.IbmVMHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
	}
	return &vmHandler, nil
}

func (cloudConn *IbmCloudConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateVPCHandler()!")
	vpcHandler := ibms.IbmVPCHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
	}
	return &vpcHandler, nil
}
func (cloudConn *IbmCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateSecurityHandler()!")
	securityHandler := ibms.IbmSecurityHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
	}
	return &securityHandler, nil
}
func (cloudConn *IbmCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateVPCHandler()!")
	keyPairHandler := ibms.IbmKeyPairHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
	}
	return &keyPairHandler, nil
}
func (cloudConn *IbmCloudConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateVMSpecHandler()!")
	vmSpecHandler := ibms.IbmVmSpecHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
	}
	return &vmSpecHandler, nil
}
func (cloudConn *IbmCloudConnection) IsConnected() (bool, error) {
	cblogger.Info("Ibm Cloud Driver: called IsConnected()!")
	return true, nil
}
func (cloudConn *IbmCloudConnection) Close() error {
	cblogger.Info("Ibm Cloud Driver: called Close()!")
	return nil
}
