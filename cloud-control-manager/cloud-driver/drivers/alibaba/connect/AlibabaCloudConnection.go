// Alibaba Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Alibaba Driver.
//
// by CB-Spider Team, 2022.09.

package connect

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	cblog "github.com/cloud-barista/cb-log"
	alirs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/alibaba/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"

	"errors"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type AlibabaCloudConnection struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo

	VMClient      *ecs.Client
	KeyPairClient *ecs.Client
	ImageClient   *ecs.Client
	//PublicIPClient      *vpc.Client
	SecurityGroupClient *ecs.Client
	//VNetClient          *vpc.Client
	VpcClient *vpc.Client
	//VNicClient          *ecs.Client
	SubnetClient  *vpc.Client
	VmSpecClient  *ecs.Client
	NLBClient     *slb.Client
	DiskClient    *ecs.Client
	MyImageClient *ecs.Client
}

/*
	func (cloudConn *AlibabaCloudConnection) CreateVNetworkHandler() (irs.VNetworkHandler, error) {
		cblogger.Info("Alibaba Cloud Driver: called CreateVNetworkHandler()!")
		vNetHandler := alirs.AlibabaVNetworkHandler{cloudConn.Region, cloudConn.VNetClient}
		return &vNetHandler, nil
	}
*/
func (cloudConn *AlibabaCloudConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	cblogger.Info("Alibaba Cloud Driver: called CreateVPCHandler()!")
	vpcHandler := alirs.AlibabaVPCHandler{cloudConn.Region, cloudConn.VpcClient}
	return &vpcHandler, nil
}

func (cloudConn *AlibabaCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("Alibaba Cloud Driver: called CreateImageHandler()!")
	imageHandler := alirs.AlibabaImageHandler{cloudConn.Region, cloudConn.ImageClient}
	return &imageHandler, nil
}

func (cloudConn *AlibabaCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("Alibaba Cloud Driver: called CreateSecurityHandler()!")
	sgHandler := alirs.AlibabaSecurityHandler{cloudConn.Region, cloudConn.SecurityGroupClient}
	return &sgHandler, nil
}

func (cloudConn *AlibabaCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("Alibaba Cloud Driver: called CreateKeyPairHandler()!")
	keyPairHandler := alirs.AlibabaKeyPairHandler{cloudConn.Region, cloudConn.KeyPairClient}
	return &keyPairHandler, nil
}

/*
func (cloudConn *AlibabaCloudConnection) CreateVNicHandler() (irs.VNicHandler, error) {
	cblogger.Info("Alibaba Cloud Driver: called CreateVNicHandler()!")
	//vNicHandler := alirs.AlibabaVNicHandler{cloudConn.Region, cloudConn.VNicClient, cloudConn.SubnetClient}
	vNicHandler := alirs.AlibabaVNicHandler{cloudConn.Region, cloudConn.VNicClient}
	return &vNicHandler, nil
}
*/

/*
func (cloudConn *AlibabaCloudConnection) CreatePublicIPHandler() (irs.PublicIPHandler, error) {
	cblogger.Info("Alibaba Cloud Driver: called CreatePublicIPHandler()!")
	publicIPHandler := alirs.AlibabaPublicIPHandler{cloudConn.Region, cloudConn.PublicIPClient}
	return &publicIPHandler, nil
}
*/

func (cloudConn *AlibabaCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("Alibaba Cloud Driver: called CreateVMHandler()!")
	vmHandler := alirs.AlibabaVMHandler{cloudConn.Region, cloudConn.VMClient}
	return &vmHandler, nil
}

func (cloudConn *AlibabaCloudConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	cblogger.Info("Start")
	handler := alirs.AlibabaVmSpecHandler{cloudConn.Region, cloudConn.VmSpecClient}
	return &handler, nil
}

func (cloudConn *AlibabaCloudConnection) CreateNLBHandler() (irs.NLBHandler, error) {
	cblogger.Info("Start")
	handler := alirs.AlibabaNLBHandler{cloudConn.Region, cloudConn.NLBClient, cloudConn.VMClient, cloudConn.VpcClient}
	return &handler, nil
}

func (cloudConn *AlibabaCloudConnection) CreateDiskHandler() (irs.DiskHandler, error) {
	cblogger.Info("Start")
	handler := alirs.AlibabaDiskHandler{cloudConn.Region, cloudConn.DiskClient}
	return &handler, nil
}

func (cloudConn *AlibabaCloudConnection) CreateMyImageHandler() (irs.MyImageHandler, error) {
	cblogger.Info("Start")
	handler := alirs.AlibabaMyImageHandler{cloudConn.Region, cloudConn.MyImageClient}
	return &handler, nil

}

func (cloudConn *AlibabaCloudConnection) CreateClusterHandler() (irs.ClusterHandler, error) {
	cblogger.Info("Alibaba Cloud Driver: called CreateClusterHandler()!")

	// temp
	// getEnv & Setting
	clusterHandler := alirs.AlibabaClusterHandler{RegionInfo: cloudConn.Region, CredentialInfo: cloudConn.CredentialInfo}

	return &clusterHandler, nil

}

func (AlibabaCloudConnection) IsConnected() (bool, error) {
	return true, nil
}

func (AlibabaCloudConnection) Close() error {
	return nil
}

func (cloudConn *AlibabaCloudConnection) CreateAnyCallHandler() (irs.AnyCallHandler, error) {
        return nil, errors.New("GCP Driver: not implemented")
}

