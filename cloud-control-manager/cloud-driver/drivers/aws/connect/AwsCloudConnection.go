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
	"github.com/aws/aws-sdk-go/service/elbv2"

	"errors"
)

//type AwsCloudConnection struct{}
type AwsCloudConnection struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	KeyPairClient  *ec2.EC2
	VMClient       *ec2.EC2

	VNetworkClient *ec2.EC2
	//VNicClient     *ec2.EC2
	ImageClient *ec2.EC2
	//PublicIPClient *ec2.EC2
	SecurityClient *ec2.EC2
	VmSpecClient   *ec2.EC2

	//NLBClient *elb.ELB
	NLBClient *elbv2.ELBV2

	DiskClient    *ec2.EC2
	MyImageClient *ec2.EC2

	AnyCallClient *ec2.EC2
}

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

func (cloudConn *AwsCloudConnection) IsConnected() (bool, error) {
	return true, nil
}

func (cloudConn *AwsCloudConnection) Close() error {
	return nil
}

func (cloudConn *AwsCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	keyPairHandler := ars.AwsKeyPairHandler{cloudConn.CredentialInfo, cloudConn.Region, cloudConn.KeyPairClient}
	//keyPairHandler := ars.AwsKeyPairHandler{cloudConn.Region, cloudConn.KeyPairClient}

	return &keyPairHandler, nil
}

func (cloudConn *AwsCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	vmHandler := ars.AwsVMHandler{cloudConn.Region, cloudConn.VMClient}
	return &vmHandler, nil
}

func (cloudConn *AwsCloudConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	handler := ars.AwsVPCHandler{cloudConn.Region, cloudConn.VNetworkClient}

	return &handler, nil
}

//func (cloudConn *AwsCloudConnection) CreateImageHandler() (irs2.ImageHandler, error) {
func (cloudConn *AwsCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
	handler := ars.AwsImageHandler{cloudConn.Region, cloudConn.ImageClient}

	return &handler, nil
}

func (cloudConn *AwsCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	handler := ars.AwsSecurityHandler{cloudConn.Region, cloudConn.SecurityClient}

	return &handler, nil
}

/*
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
*/

func (cloudConn *AwsCloudConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	handler := ars.AwsVmSpecHandler{cloudConn.Region, cloudConn.VmSpecClient}
	return &handler, nil
}

func (cloudConn *AwsCloudConnection) CreateNLBHandler() (irs.NLBHandler, error) {
	handler := ars.AwsNLBHandler{cloudConn.Region, cloudConn.NLBClient, cloudConn.VMClient}
	return &handler, nil
}

func (cloudConn *AwsCloudConnection) CreateDiskHandler() (irs.DiskHandler, error) {
	handler := ars.AwsDiskHandler{cloudConn.Region, cloudConn.DiskClient}
	return &handler, nil
}


func (cloudConn *AwsCloudConnection) CreateMyImageHandler() (irs.MyImageHandler, error) {
	handler := ars.AwsMyImageHandler{cloudConn.Region, cloudConn.MyImageClient}
	return &handler, nil
}

func (cloudConn *AwsCloudConnection) CreateClusterHandler() (irs.ClusterHandler, error) {
        return nil, errors.New("AWS Driver: not implemented")
}

func (cloudConn *AwsCloudConnection) CreateAnyCallHandler() (irs.AnyCallHandler, error) {
	handler := ars.AwsAnyCallHandler{cloudConn.Region, cloudConn.CredentialInfo, cloudConn.AnyCallClient}
        return &handler, nil
}

