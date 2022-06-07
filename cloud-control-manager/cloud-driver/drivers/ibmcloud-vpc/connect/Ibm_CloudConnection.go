package connect

import (
	"context"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	cblog "github.com/cloud-barista/cb-log"
	ibmrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibmcloud-vpc/resources"
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
	imageHandler := ibmrs.IbmImageHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
	}
	return &imageHandler, nil
}

func (cloudConn *IbmCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateVMHandler()!")
	vmHandler := ibmrs.IbmVMHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
	}
	return &vmHandler, nil
}

func (cloudConn *IbmCloudConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateVPCHandler()!")
	vpcHandler := ibmrs.IbmVPCHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
	}
	return &vpcHandler, nil
}
func (cloudConn *IbmCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateSecurityHandler()!")
	securityHandler := ibmrs.IbmSecurityHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
	}
	return &securityHandler, nil
}
func (cloudConn *IbmCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateVPCHandler()!")
	keyPairHandler := ibmrs.IbmKeyPairHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
	}
	return &keyPairHandler, nil
}
func (cloudConn *IbmCloudConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateVMSpecHandler()!")
	vmSpecHandler := ibmrs.IbmVmSpecHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
	}
	return &vmSpecHandler, nil
}

func (cloudConn *IbmCloudConnection) CreateNLBHandler() (irs.NLBHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateNLBHandler()!")
	nlbHandler := ibmrs.IbmNLBHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
	}
	return &nlbHandler, nil
}

func (cloudConn *IbmCloudConnection) IsConnected() (bool, error) {
	cblogger.Info("Ibm Cloud Driver: called IsConnected()!")
	return true, nil
}
func (cloudConn *IbmCloudConnection) Close() error {
	cblogger.Info("Ibm Cloud Driver: called Close()!")
	return nil
}
