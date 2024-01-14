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
	"fmt"

	"github.com/sirupsen/logrus"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	vlb "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vloadbalancer"
	vpc "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vpc"
	vserver "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"

	// ncpvpcrs "github.com/cloud-barista/ncpvpc/ncpvpc/resources" // For local testing
	ncpvpcrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncpvpc/resources"
)

type NcpVpcCloudConnection struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VmClient       *vserver.APIClient
	VpcClient      *vpc.APIClient
	VlbClient      *vlb.APIClient
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

	vmHandler := ncpvpcrs.NcpVpcVMHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		RegionInfo:     cloudConn.RegionInfo,
		VMClient:       cloudConn.VmClient,
	}

	return &vmHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateVMSpecHandler()!")
	vmspecHandler := ncpvpcrs.NcpVpcVMSpecHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}

	return &vmspecHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateImagehandler()!")
	imageHandler := ncpvpcrs.NcpVpcImageHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}

	return &imageHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateKeyPairHandler()!")
	keypairHandler := ncpvpcrs.NcpVpcKeyPairHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}

	return &keypairHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateSecurityHandler()!")
	sgHandler := ncpvpcrs.NcpVpcSecurityHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}

	return &sgHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateVPCHandler()!")
	vpcHandler := ncpvpcrs.NcpVpcVPCHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, VPCClient: cloudConn.VpcClient}

	return &vpcHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreateNLBHandler() (irs.NLBHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateNLBHandler()!")
	nlbHandler := ncpvpcrs.NcpVpcNLBHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient, VPCClient: cloudConn.VpcClient, VLBClient: cloudConn.VlbClient}

	return &nlbHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreateDiskHandler() (irs.DiskHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateDiskHandler()!")
	// cblogger.Info("\n### cloudConn.RegionInfo : ")
	// spew.Dump(cloudConn.RegionInfo)
	// cblogger.Info("\n")

	diskHandler := ncpvpcrs.NcpVpcDiskHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}

	return &diskHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreateMyImageHandler() (irs.MyImageHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateMyImageHandler()!")
	myimageHandler := ncpvpcrs.NcpVpcMyImageHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}

	return &myimageHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreateClusterHandler() (irs.ClusterHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateClusterHandler()!")

	return nil, fmt.Errorf("NCP VPC Cloud Driver does not support CreateClusterHandler yet.")
}

func (cloudConn *NcpVpcCloudConnection) CreateAnyCallHandler() (irs.AnyCallHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateAnyCallHandler()!")

	return nil, fmt.Errorf("NCP VPC Cloud Driver does not support CreateAnyCallHandler yet.")
}

func (cloudConn *NcpVpcCloudConnection) CreateRegionZoneHandler() (irs.RegionZoneHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreateRegionZoneHandler()!")

	regionZoneHandler := ncpvpcrs.NcpRegionZoneHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}
	return &regionZoneHandler, nil
}

func (cloudConn *NcpVpcCloudConnection) CreatePriceInfoHandler() (irs.PriceInfoHandler, error) {
	cblogger.Info("NCP VPC Cloud Driver: called CreatePriceInfoHandler()!")

	priceInfoHandler := ncpvpcrs.NcpVpcPriceInfoHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}
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
