// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2020.12.
// by ETRI, 2022.10. updated

package connect

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	vas "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vautoscaling"
	vlb "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vloadbalancer"
	vnks "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vnks"
	vpc "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vpc"
	vserver "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"

	// ncprs "github.com/cloud-barista/ncp/ncp/resources" // For local testing
	ncprs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncp/resources"
)

type NcpVpcCloudConnection struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VmClient       *vserver.APIClient
	VpcClient      *vpc.APIClient
	VlbClient      *vlb.APIClient
	VnksClient     *vnks.APIClient
	VasClient      *vas.APIClient
}

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP VPC Connect")
}

func (cloudConn *NcpVpcCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateVMHandler()!")

	//NOTE Just for Test!!
	// cblogger.Info("cloudConn.CredentialInfo.ClientId : ")
	// spew.Dump(cloudConn.CredentialInfo.ClientId)
	// cblogger.Info("cloudConn.RegionInfo : ")
	// spew.Dump(cloudConn.RegionInfo)

	vmHandler := ncprs.NcpVpcVMHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		RegionInfo:     cloudConn.RegionInfo,
		VMClient:       cloudConn.VmClient,
	}

	return &vmHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateVMSpecHandler()!")
	vmspecHandler := ncprs.NcpVpcVMSpecHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}

	return &vmspecHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateImagehandler()!")
	imageHandler := ncprs.NcpVpcImageHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}

	return &imageHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateKeyPairHandler()!")
	keypairHandler := ncprs.NcpVpcKeyPairHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}

	return &keypairHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateSecurityHandler()!")
	sgHandler := ncprs.NcpVpcSecurityHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}

	return &sgHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateVPCHandler()!")
	vpcHandler := ncprs.NcpVpcVPCHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, VPCClient: cloudConn.VpcClient}

	return &vpcHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreateNLBHandler() (irs.NLBHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateNLBHandler()!")
	nlbHandler := ncprs.NcpVpcNLBHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient, VPCClient: cloudConn.VpcClient, VLBClient: cloudConn.VlbClient}

	return &nlbHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreateDiskHandler() (irs.DiskHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateDiskHandler()!")
	// cblogger.Info("\n### cloudConn.RegionInfo : ")
	// spew.Dump(cloudConn.RegionInfo)
	// cblogger.Info("\n")

	diskHandler := ncprs.NcpVpcDiskHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}

	return &diskHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreateMyImageHandler() (irs.MyImageHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateMyImageHandler()!")
	myimageHandler := ncprs.NcpVpcMyImageHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}

	return &myimageHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreateClusterHandler() (irs.ClusterHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateClusterHandler()!")

	ctx := context.Background()
	clusterHandler := ncprs.NcpVpcClusterHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		RegionInfo:     cloudConn.RegionInfo,
		Ctx:            ctx,
		VMClient:       cloudConn.VmClient,
		VPCClient:      cloudConn.VpcClient,
		ClusterClient:  cloudConn.VnksClient,
		ASClient:       cloudConn.VasClient,
	}
	return &clusterHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreateAnyCallHandler() (irs.AnyCallHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateAnyCallHandler()!")

	return nil, fmt.Errorf("NCP VPC Cloud Driver does not support CreateAnyCallHandler yet.")
}

func (cloudConn *NcpVpcCloudConnection) CreateRegionZoneHandler() (irs.RegionZoneHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateRegionZoneHandler()!")

	regionZoneHandler := ncprs.NcpRegionZoneHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}
	return &regionZoneHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreatePriceInfoHandler() (irs.PriceInfoHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreatePriceInfoHandler()!")

	priceInfoHandler := ncprs.NcpVpcPriceInfoHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}
	return &priceInfoHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) IsConnected() (bool, error) {
	cblogger.Info("NCP VPC Cloud Driver: called IsConnected()!")
	if cloudConn == nil {
		return false, nil
	}

	if cloudConn.VmClient.V2Api == nil {
		return false, nil
	}

	return true, nil
}

func (cloudConn *NcpVpcCloudConnection) Close() error {
	cblogger.Info("NCP VPC Cloud Driver: called Close()!")

	return nil
}

func (cloudConn *NcpVpcCloudConnection) CreateTagHandler() (irs.TagHandler, error) {
	return nil, fmt.Errorf("NCP VPC Cloud Driver: not implemented")
}

func (cloudConn *NcpVpcCloudConnection) CreateFileSystemHandler() (irs.FileSystemHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateFileSystemHandler()!")
	return nil, fmt.Errorf("NCP VPC Cloud Driver: CreateFileSystemHandler is not implemented")
}
