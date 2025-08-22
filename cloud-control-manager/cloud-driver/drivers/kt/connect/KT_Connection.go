// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2022.08.

package connect

import (
	"fmt"

	"github.com/sirupsen/logrus"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	ktvpcsdk "github.com/cloud-barista/ktcloudvpc-sdk-go"

	// ktvpcrs "github.com/cloud-barista/ktcloudvpc/ktcloudvpc/resources"
	ktvpcrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/kt/resources" //To be built in the container
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type KTCloudVpcConnection struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VMClient       *ktvpcsdk.ServiceClient
	ImageClient    *ktvpcsdk.ServiceClient
	NetworkClient  *ktvpcsdk.ServiceClient
	VolumeClient   *ktvpcsdk.ServiceClient
	NLBClient      *ktvpcsdk.ServiceClient
}

// CreateFileSystemHandler implements connect.CloudConnection.
func (cloudConn *KTCloudVpcConnection) CreateFileSystemHandler() (irs.FileSystemHandler, error) {
	panic("unimplemented")
}

func (cloudConn *KTCloudVpcConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("KT Cloud VPC Driver: called CreateVMHandler()!")
	vmHandler := ktvpcrs.KTVpcVMHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VMClient, ImageClient: cloudConn.ImageClient, NetworkClient: cloudConn.NetworkClient, VolumeClient: cloudConn.VolumeClient}
	return &vmHandler, nil
}

func (cloudConn *KTCloudVpcConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("KT Cloud VPC Driver: called CreateImageHandler()!")
	imageHandler := ktvpcrs.KTVpcImageHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VMClient, ImageClient: cloudConn.ImageClient}
	return &imageHandler, nil
}

func (cloudConn *KTCloudVpcConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	cblogger.Info("KT Cloud VPC Driver: called CreateVMSpecHandler()!")
	vmSpecHandler := ktvpcrs.KTVpcVMSpecHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VMClient}
	return &vmSpecHandler, nil
}

func (cloudConn *KTCloudVpcConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("KT Cloud VPC Driver: called CreateKeyPairHandler()!")
	keypairHandler := ktvpcrs.KTVpcKeyPairHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VMClient}
	return &keypairHandler, nil
}

func (cloudConn KTCloudVpcConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("KT Cloud VPC Driver: called CreateSecurityHandler()!")
	securityHandler := ktvpcrs.KTVpcSecurityHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VMClient, NetworkClient: cloudConn.NetworkClient}
	return &securityHandler, nil
}

func (cloudConn *KTCloudVpcConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	cblogger.Info("KT Cloud VPC Driver: called CreateVPCHandler()!")
	vpcHandler := ktvpcrs.KTVpcVPCHandler{RegionInfo: cloudConn.RegionInfo, NetworkClient: cloudConn.NetworkClient}
	return &vpcHandler, nil
}

func (cloudConn *KTCloudVpcConnection) CreateNLBHandler() (irs.NLBHandler, error) {
	cblogger.Info("KT Cloud Driver: called CreateNLBHandler()!")
	nlbHandler := ktvpcrs.KTVpcNLBHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VMClient, NetworkClient: cloudConn.NetworkClient, NLBClient: cloudConn.NLBClient}
	return &nlbHandler, nil
}

func (cloudConn *KTCloudVpcConnection) CreateDiskHandler() (irs.DiskHandler, error) {
	cblogger.Info("KT Cloud VPC Driver: called CreateDiskHandler()!")
	diskHandler := ktvpcrs.KTVpcDiskHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VMClient, VolumeClient: cloudConn.VolumeClient}
	return &diskHandler, nil
}

func (cloudConn *KTCloudVpcConnection) CreateClusterHandler() (irs.ClusterHandler, error) {
	cblogger.Info("KT Cloud VPC Driver: called CreateClusterHandler()!")

	return nil, fmt.Errorf("KT Cloud VPC Driver does not support ClusterHandler yet.")
}

func (cloudConn *KTCloudVpcConnection) CreateMyImageHandler() (irs.MyImageHandler, error) {
	cblogger.Info("KT Cloud VPC Driver: called CreateMyImageHandler()!")
	myimageHandler := ktvpcrs.KTVpcMyImageHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VMClient, ImageClient: cloudConn.ImageClient, NetworkClient: cloudConn.NetworkClient, VolumeClient: cloudConn.VolumeClient}
	return &myimageHandler, nil
}

func (cloudConn *KTCloudVpcConnection) CreateAnyCallHandler() (irs.AnyCallHandler, error) {
	cblogger.Info("KT Cloud VPC Driver: called CreateAnyCallHandler()!")

	return nil, fmt.Errorf("KT Cloud VPC Driver does not support AnyCallHandler yet.")
}

func (cloudConn *KTCloudVpcConnection) CreateRegionZoneHandler() (irs.RegionZoneHandler, error) {
	cblogger.Info("KT Cloud VPC Driver: called CreateRegionZoneHandler()!")
	regionZoneHandler := ktvpcrs.KTVpcRegionZoneHandler{RegionInfo: cloudConn.RegionInfo}
	return &regionZoneHandler, nil
}

func (cloudConn *KTCloudVpcConnection) CreatePriceInfoHandler() (irs.PriceInfoHandler, error) {
	cblogger.Info("KT Cloud VPC Driver: called CreatePriceInfoHandler()!")

	return nil, fmt.Errorf("KT Cloud VPC Driver does not support PriceInfoHandler yet.")
}

func (cloudConn *KTCloudVpcConnection) IsConnected() (bool, error) {
	return true, nil
}

func (cloudConn *KTCloudVpcConnection) Close() error {
	return nil
}

func (cloudConn *KTCloudVpcConnection) CreateTagHandler() (irs.TagHandler, error) {
	return nil, fmt.Errorf("KT Cloud VPC Driver: not implemented")
}
