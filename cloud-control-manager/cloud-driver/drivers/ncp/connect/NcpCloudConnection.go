// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2020.08.
// by ETRI, 2022.08.

package connect

import (
	"fmt"

	"github.com/sirupsen/logrus"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	lb "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/loadbalancer"
	server "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/server"

	// ncprs 	"github.com/cloud-barista/ncp/ncp/resources"
	ncprs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncp/resources" //To be built in the container
)

type NcpCloudConnection struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VmClient       *server.APIClient
	LbClient       *lb.APIClient
}

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("NCP Connect")
}

func (cloudConn *NcpCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("NCP Cloud Driver: called CreateVMHandler()!")

	vmHandler := ncprs.NcpVMHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		RegionInfo:     cloudConn.RegionInfo,
		VMClient:       cloudConn.VmClient,
	}
	return &vmHandler, nil
}

func (cloudConn *NcpCloudConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	cblogger.Info("NCP Cloud Driver: called CreateVMSpecHandler()!")

	vmspecHandler := ncprs.NcpVMSpecHandler{cloudConn.CredentialInfo, cloudConn.RegionInfo, cloudConn.VmClient}
	return &vmspecHandler, nil
}

func (cloudConn *NcpCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("NCP Cloud Driver: called CreateImagehandler()!")

	imageHandler := ncprs.NcpImageHandler{cloudConn.CredentialInfo, cloudConn.RegionInfo, cloudConn.VmClient}
	return &imageHandler, nil
}

func (cloudConn *NcpCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("NCP Cloud Driver: called CreateKeyPairHandler()!")

	keypairHandler := ncprs.NcpKeyPairHandler{cloudConn.CredentialInfo, cloudConn.RegionInfo, cloudConn.VmClient}
	return &keypairHandler, nil
}

func (cloudConn *NcpCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("NCP Cloud Driver: called CreateSecurityHandler()!")

	sgHandler := ncprs.NcpSecurityHandler{cloudConn.CredentialInfo, cloudConn.RegionInfo, cloudConn.VmClient}
	return &sgHandler, nil
}

func (cloudConn *NcpCloudConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	cblogger.Info("NCP Cloud Driver: called CreateVPCHandler()!")

	vpcHandler := ncprs.NcpVPCHandler{cloudConn.CredentialInfo, cloudConn.RegionInfo, cloudConn.VmClient}
	return &vpcHandler, nil
}

func (cloudConn *NcpCloudConnection) CreateNLBHandler() (irs.NLBHandler, error) {
	cblogger.Info("NCP Cloud Driver: called CreateNLBHandler()!")

	nlbHandler := ncprs.NcpNLBHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient, LBClient: cloudConn.LbClient}
	return &nlbHandler, nil
}

func (cloudConn *NcpCloudConnection) CreateDiskHandler() (irs.DiskHandler, error) {
	cblogger.Info("NCP Cloud Driver: called CreateDiskHandler()!")

	diskHandler := ncprs.NcpDiskHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}
	return &diskHandler, nil
}

func (cloudConn *NcpCloudConnection) CreateMyImageHandler() (irs.MyImageHandler, error) {
	cblogger.Info("NCP Cloud Driver: called CreateMyImageHandler()!")

	myimageHandler := ncprs.NcpMyImageHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}
	return &myimageHandler, nil
}

func (cloudConn *NcpCloudConnection) CreateClusterHandler() (irs.ClusterHandler, error) {
	cblogger.Info("NCP Cloud Driver: called CreateClusterHandler()!")

	return nil, fmt.Errorf("NCP Cloud Driver does not support CreateClusterHandler yet.")
}

func (cloudConn *NcpCloudConnection) CreateAnyCallHandler() (irs.AnyCallHandler, error) {
	cblogger.Info("NCP Cloud Driver: called CreateAnyCallHandler()!")

	return nil, fmt.Errorf("NCP Cloud Driver does not support CreateAnyCallHandler yet.")
}

func (cloudConn *NcpCloudConnection) CreateRegionZoneHandler() (irs.RegionZoneHandler, error) {
	cblogger.Info("NCP Cloud Driver: called CreateRegionZoneHandler()!")

	regionZoneHandler := ncprs.NcpRegionZoneHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}
	return &regionZoneHandler, nil
}

func (cloudConn *NcpCloudConnection) CreatePriceInfoHandler() (irs.PriceInfoHandler, error) {
	cblogger.Info("NCP Cloud Driver: called CreatePriceInfoHandler()!")

	priceInfoHandler := ncprs.NcpPriceInfoHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VmClient}
	return &priceInfoHandler, nil
}

func (cloudConn *NcpCloudConnection) IsConnected() (bool, error) {
	cblogger.Info("NCP Cloud Driver: called IsConnected()!")
	if cloudConn == nil {
		return false, nil
	}

	if cloudConn.VmClient.V2Api == nil {
		return false, nil
	}

	return true, nil
}

func (cloudConn *NcpCloudConnection) Close() error {
	cblogger.Info("NCP Cloud Driver: called Close()!")

	return nil
}
