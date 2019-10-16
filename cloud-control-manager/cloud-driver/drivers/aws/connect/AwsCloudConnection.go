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

	//irs2 "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"

	ars "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/aws/resources"

	//ec2drv "github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ec2"
)

//type AwsCloudConnection struct{}
type AwsCloudConnection struct {
	Region        idrv.RegionInfo
	KeyPairClient *ec2.EC2
	VMClient      *ec2.EC2

	VNetworkClient *ec2.EC2
	VNicClient     *ec2.EC2
	ImageClient    *ec2.EC2
	PublicIPClient *ec2.EC2
	SecurityClient *ec2.EC2
}

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("AWS Connect")
}

func (cloudConn *AwsCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("Start CreateKeyPairHandler()")

	keyPairHandler := ars.AwsKeyPairHandler{cloudConn.Region, cloudConn.KeyPairClient}

	return &keyPairHandler, nil
}

func (cloudConn *AwsCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("Start CreateVMHandler()")

	vmHandler := ars.AwsVMHandler{cloudConn.Region, cloudConn.VMClient}
	return &vmHandler, nil
}

func (cloudConn *AwsCloudConnection) IsConnected() (bool, error) {
	return true, nil
}
func (cloudConn *AwsCloudConnection) Close() error {
	return nil
}

func (cloudConn *AwsCloudConnection) CreateVNetworkHandler() (irs.VNetworkHandler, error) {
	cblogger.Info("Start")
	handler := ars.AwsVNetworkHandler{cloudConn.Region, cloudConn.VNetworkClient}

	return &handler, nil
}

//func (cloudConn *AwsCloudConnection) CreateImageHandler() (irs2.ImageHandler, error) {
func (cloudConn *AwsCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("Start")
	handler := ars.AwsImageHandler{cloudConn.Region, cloudConn.ImageClient}

	return &handler, nil
}

func (cloudConn *AwsCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("Start")
	handler := ars.AwsSecurityHandler{cloudConn.Region, cloudConn.SecurityClient}

	return &handler, nil
}

func (cloudConn *AwsCloudConnection) CreateVNicHandler() (irs.VNicHandler, error) {
	cblogger.Info("Start")
	handler := ars.AwsVNicHandler{cloudConn.Region, cloudConn.VNicClient}

	return &handler, nil
}
func (cloudConn *AwsCloudConnection) CreatePublicIPHandler() (irs.PublicIPHandler, error) {
	cblogger.Info("Start")
	handler := ars.AwsPublicIPHandler{cloudConn.Region, cloudConn.PublicIPClient}

	return &handler, nil
}
