// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by powerkim@etri.re.kr, 2019.06.

package connect

import (
	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	trs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/tencent/resources"
	"github.com/sirupsen/logrus"

	//ec2drv "github.com/tencent/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2"
)

//type TencentCloudConnection struct{}
type TencentCloudConnection struct {
	Region        idrv.RegionInfo
	KeyPairClient *ec2.EC2
	VMClient      *ec2.EC2

	VNetworkClient *ec2.EC2
	//VNicClient     *ec2.EC2
	ImageClient *ec2.EC2
	//PublicIPClient *ec2.EC2
	SecurityClient *ec2.EC2
	VmSpecClient   *ec2.EC2
}

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

func (cloudConn *TencentCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("Start CreateKeyPairHandler()")

	keyPairHandler := trs.TencentKeyPairHandler{cloudConn.Region, cloudConn.KeyPairClient}

	return &keyPairHandler, nil
}

func (cloudConn *TencentCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("Start CreateVMHandler()")

	vmHandler := trs.TencentVMHandler{cloudConn.Region, cloudConn.VMClient}
	return &vmHandler, nil
}

func (cloudConn *TencentCloudConnection) IsConnected() (bool, error) {
	return true, nil
}
func (cloudConn *TencentCloudConnection) Close() error {
	return nil
}

func (cloudConn *TencentCloudConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	cblogger.Info("Start")
	handler := trs.TencentVPCHandler{cloudConn.Region, cloudConn.VNetworkClient}

	return &handler, nil
}

//func (cloudConn *TencentCloudConnection) CreateImageHandler() (irs2.ImageHandler, error) {
func (cloudConn *TencentCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("Start")
	handler := trs.TencentImageHandler{cloudConn.Region, cloudConn.ImageClient}

	return &handler, nil
}

func (cloudConn *TencentCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("Start")
	handler := trs.TencentSecurityHandler{cloudConn.Region, cloudConn.SecurityClient}

	return &handler, nil
}

/*
func (cloudConn *TencentCloudConnection) CreateVNicHandler() (irs.VNicHandler, error) {
	cblogger.Info("Start")
	handler := trs.TencentVNicHandler{cloudConn.Region, cloudConn.VNicClient}

	return &handler, nil
}

func (cloudConn *TencentCloudConnection) CreatePublicIPHandler() (irs.PublicIPHandler, error) {
	cblogger.Info("Start")
	handler := trs.TencentPublicIPHandler{cloudConn.Region, cloudConn.PublicIPClient}

	return &handler, nil
}
*/

func (cloudConn *TencentCloudConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	cblogger.Info("Start")
	handler := trs.TencentVmSpecHandler{cloudConn.Region, cloudConn.VmSpecClient}
	return &handler, nil
}
