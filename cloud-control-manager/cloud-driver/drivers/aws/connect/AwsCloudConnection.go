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

	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/costexplorer"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/efs"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/pricing"
	"github.com/aws/aws-sdk-go/service/sts"

	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/iam"
)

// type AwsCloudConnection struct{}
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

	//RegionZoneClient
	RegionZoneClient *ec2.EC2

	//PriceInfoClient
	PriceInfoClient *pricing.Pricing

	DiskClient    *ec2.EC2
	MyImageClient *ec2.EC2

	EKSClient         *eks.EKS
	IamClient         *iam.IAM
	StsClient         *sts.STS
	AutoScalingClient *autoscaling.AutoScaling

	AnyCallClient *ec2.EC2
	TagClient     *ec2.EC2

	CostExplorerClient *costexplorer.CostExplorer

	CloudWatchClient *cloudwatch.CloudWatch
	FileSystemClient *efs.EFS
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
	tagHandler := cloudConn.CreateAwsTagHandler()
	keyPairHandler := ars.AwsKeyPairHandler{CredentialInfo: cloudConn.CredentialInfo, Region: cloudConn.Region, Client: cloudConn.KeyPairClient, TagHandler: &tagHandler}
	//keyPairHandler := ars.AwsKeyPairHandler{cloudConn.Region, cloudConn.KeyPairClient}

	return &keyPairHandler, nil
}

func (cloudConn *AwsCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	tagHandler := cloudConn.CreateAwsTagHandler()
	vmHandler := ars.AwsVMHandler{Region: cloudConn.Region, Client: cloudConn.VMClient, TagHandler: &tagHandler}
	return &vmHandler, nil
}

func (cloudConn *AwsCloudConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	tagHandler := cloudConn.CreateAwsTagHandler()
	handler := ars.AwsVPCHandler{Region: cloudConn.Region, Client: cloudConn.VNetworkClient, TagHandler: &tagHandler}

	return &handler, nil
}

// func (cloudConn *AwsCloudConnection) CreateImageHandler() (irs2.ImageHandler, error) {
func (cloudConn *AwsCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
	handler := ars.AwsImageHandler{Region: cloudConn.Region, Client: cloudConn.ImageClient}

	return &handler, nil
}

func (cloudConn *AwsCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	tagHandler := cloudConn.CreateAwsTagHandler()
	handler := ars.AwsSecurityHandler{Region: cloudConn.Region, Client: cloudConn.SecurityClient, TagHandler: &tagHandler}

	return &handler, nil
}

func (cloudConn *AwsCloudConnection) CreateTagHandler() (irs.TagHandler, error) {
	handler := ars.AwsTagHandler{Region: cloudConn.Region, Client: cloudConn.VMClient, NLBClient: cloudConn.NLBClient, EKSClient: cloudConn.EKSClient, EFSClient: cloudConn.FileSystemClient}
	return &handler, nil
}

func (cloudConn *AwsCloudConnection) CreateAwsTagHandler() ars.AwsTagHandler {
	handler := ars.AwsTagHandler{Region: cloudConn.Region, Client: cloudConn.VMClient, NLBClient: cloudConn.NLBClient, EKSClient: cloudConn.EKSClient, EFSClient: cloudConn.FileSystemClient}
	return handler
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
	handler := ars.AwsVmSpecHandler{Region: cloudConn.Region, Client: cloudConn.VmSpecClient}
	return &handler, nil
}

func (cloudConn *AwsCloudConnection) CreateNLBHandler() (irs.NLBHandler, error) {
	tagHandler := cloudConn.CreateAwsTagHandler()
	handler := ars.AwsNLBHandler{Region: cloudConn.Region, Client: cloudConn.NLBClient, VMClient: cloudConn.VMClient, TagHandler: &tagHandler}
	return &handler, nil
}

func (cloudConn *AwsCloudConnection) CreateDiskHandler() (irs.DiskHandler, error) {
	tagHandler := cloudConn.CreateAwsTagHandler()
	handler := ars.AwsDiskHandler{Region: cloudConn.Region, Client: cloudConn.DiskClient, TagHandler: &tagHandler}
	return &handler, nil
}

// CreateFileSystemHandler implements connect.CloudConnection.
func (cloudConn *AwsCloudConnection) CreateFileSystemHandler() (irs.FileSystemHandler, error) {
	tagHandler := cloudConn.CreateAwsTagHandler()
	handler := ars.AwsFileSystemHandler{Region: cloudConn.Region, Client: cloudConn.FileSystemClient, EC2Client: cloudConn.VNetworkClient, TagHandler: &tagHandler}
	return &handler, nil
}

func (cloudConn *AwsCloudConnection) CreateMyImageHandler() (irs.MyImageHandler, error) {
	tagHandler := cloudConn.CreateAwsTagHandler()
	handler := ars.AwsMyImageHandler{Region: cloudConn.Region, Client: cloudConn.MyImageClient, TagHandler: &tagHandler}
	return &handler, nil
}

func (cloudConn *AwsCloudConnection) CreateClusterHandler() (irs.ClusterHandler, error) {
	tagHandler := cloudConn.CreateAwsTagHandler()
	cblogger.Info("CreateClusterHandler through")
	if cloudConn.MyImageClient == nil {
		cblogger.Info("cloudConn.MyImageClient is nil")
	}
	if cloudConn.EKSClient == nil {
		cblogger.Info("cloudConn.EKSClient is nil")
	}
	if cloudConn.VNetworkClient == nil {
		cblogger.Info("cloudConn.VNetworkClient is nil")
	}
	if cloudConn.IamClient == nil {
		cblogger.Info("cloudConn.IamClient is nil")
	}
	if cloudConn.AutoScalingClient == nil {
		cblogger.Info("cloudConn.AutoScalingClient is nil")
	}
	handler := ars.AwsClusterHandler{CredentialInfo: cloudConn.CredentialInfo, Region: cloudConn.Region, Client: cloudConn.EKSClient, EC2Client: cloudConn.VNetworkClient, Iam: cloudConn.IamClient, StsClient: cloudConn.StsClient, AutoScaling: cloudConn.AutoScalingClient, TagHandler: &tagHandler}
	return &handler, nil
}

func (cloudConn *AwsCloudConnection) CreateAnyCallHandler() (irs.AnyCallHandler, error) {
	handler := ars.AwsAnyCallHandler{Region: cloudConn.Region, CredentialInfo: cloudConn.CredentialInfo, Client: cloudConn.AnyCallClient, CeClient: cloudConn.CostExplorerClient, CwClient: cloudConn.CloudWatchClient}
	return &handler, nil
}

func (cloudConn *AwsCloudConnection) CreateRegionZoneHandler() (irs.RegionZoneHandler, error) {
	handler := ars.AwsRegionZoneHandler{Region: cloudConn.Region, Client: cloudConn.RegionZoneClient}
	return &handler, nil
}

func (cloudConn *AwsCloudConnection) CreatePriceInfoHandler() (irs.PriceInfoHandler, error) {
	handler := ars.AwsPriceInfoHandler{Region: cloudConn.Region, Client: cloudConn.PriceInfoClient}
	return &handler, nil
}
