// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI Team, 2021.12.
// by ETRI Team, 2022.08.

package connect

import (
	"errors"
	"fmt"

	// "github.com/davecgh/go-spew/spew"
	"github.com/sirupsen/logrus"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"

	// nhnrs "github.com/cloud-barista/nhncloud/nhncloud/resources"
	nhnrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/nhncloud/resources"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type NhnCloudConnection struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	VMClient       *nhnsdk.ServiceClient
	ImageClient    *nhnsdk.ServiceClient
	NetworkClient  *nhnsdk.ServiceClient
	VolumeClient   *nhnsdk.ServiceClient
	ClusterClient  *nhnsdk.ServiceClient
}

// CreateFileSystemHandler implements connect.CloudConnection.
func (cloudConn *NhnCloudConnection) CreateFileSystemHandler() (irs.FileSystemHandler, error) {
	panic("unimplemented")
}

func (cloudConn *NhnCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("NhnCloud Cloud Driver: called CreateVMHandler()!")
	vmHandler := nhnrs.NhnCloudVMHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VMClient, ImageClient: cloudConn.ImageClient, NetworkClient: cloudConn.NetworkClient, VolumeClient: cloudConn.VolumeClient}

	return &vmHandler, nil
}

func (cloudConn *NhnCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("NhnCloud Cloud Driver: called CreateImageHandler()!")
	imageHandler := nhnrs.NhnCloudImageHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VMClient, ImageClient: cloudConn.ImageClient}

	return &imageHandler, nil
}

func (cloudConn *NhnCloudConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	cblogger.Info("NhnCloud Cloud Driver: called CreateVMSpecHandler()!")
	vmSpecHandler := nhnrs.NhnCloudVMSpecHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VMClient}

	return &vmSpecHandler, nil
}

func (cloudConn *NhnCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("NhnCloud Cloud Driver: called CreateKeyPairHandler()!")
	keypairHandler := nhnrs.NhnCloudKeyPairHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VMClient}

	return &keypairHandler, nil
}

func (cloudConn NhnCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("NhnCloud Cloud Driver: called CreateSecurityHandler()!")
	securityHandler := nhnrs.NhnCloudSecurityHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VMClient, NetworkClient: cloudConn.NetworkClient}

	return &securityHandler, nil
}

func (cloudConn *NhnCloudConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	cblogger.Info("NhnCloud Cloud Driver: called CreateVPCHandler()!")
	vpcHandler := nhnrs.NhnCloudVPCHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, NetworkClient: cloudConn.NetworkClient}

	return &vpcHandler, nil
}

func (cloudConn *NhnCloudConnection) CreateNLBHandler() (irs.NLBHandler, error) {
	cblogger.Info("NhnCloud Cloud Driver: called CreateNLBHandler()!")
	nlbHandler := nhnrs.NhnCloudNLBHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VMClient, NetworkClient: cloudConn.NetworkClient} //Caution!! : No NLBClient

	return &nlbHandler, nil
}

func (cloudConn *NhnCloudConnection) CreateDiskHandler() (irs.DiskHandler, error) {
	cblogger.Info("NhnCloud Cloud Driver: called CreateDiskHandler()!")
	diskHandler := nhnrs.NhnCloudDiskHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VMClient, VolumeClient: cloudConn.VolumeClient}

	return &diskHandler, nil
}

func (cloudConn *NhnCloudConnection) CreateClusterHandler() (irs.ClusterHandler, error) {
	cblogger.Info("NhnCloud Cloud Driver: called CreateClusterHandler()!")

	if cloudConn.ClusterClient == nil {
		// Some regions(ex. JPN) do not support a cluster service.
		err := fmt.Errorf("ClusterClient is invalid, indicating that no suitable endpoint was found in the service catalog.")
		return nil, err
	}

	clusterHandler := nhnrs.NhnCloudClusterHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VMClient, ImageClient: cloudConn.ImageClient, NetworkClient: cloudConn.NetworkClient, ClusterClient: cloudConn.ClusterClient}

	return &clusterHandler, nil
}

func (cloudConn *NhnCloudConnection) CreateMyImageHandler() (irs.MyImageHandler, error) {
	cblogger.Info("NhnCloud Cloud Driver: called CreateMyImageHandler()!")
	myimageHandler := nhnrs.NhnCloudMyImageHandler{RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VMClient, ImageClient: cloudConn.ImageClient, NetworkClient: cloudConn.NetworkClient, VolumeClient: cloudConn.VolumeClient}

	return &myimageHandler, nil
}

func (cloudConn *NhnCloudConnection) CreateAnyCallHandler() (irs.AnyCallHandler, error) {
	cblogger.Info("NhnCloud Cloud Driver: called CreateAnyCallHandler()!")

	return nil, fmt.Errorf("NHN Cloud Driver does not support CreateAnyCallHandler yet.")
}

func (cloudConn *NhnCloudConnection) CreateRegionZoneHandler() (irs.RegionZoneHandler, error) {
	cblogger.Info("NhnCloud Cloud Driver: called CreateRegionZoneHandler()!")
	regionZoneHandler := nhnrs.NhnCloudRegionZoneHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, VMClient: cloudConn.VMClient}

	return &regionZoneHandler, nil
}

func (cloudConn *NhnCloudConnection) CreatePriceInfoHandler() (irs.PriceInfoHandler, error) {

	return nil, errors.New("NHN Cloud Driver: not implemented")
}

func (cloudConn *NhnCloudConnection) IsConnected() (bool, error) {
	if cloudConn == nil {
		return false, nil
	}

	return true, nil
}

func (cloudConn *NhnCloudConnection) Close() error {

	return nil
}

func (cloudConn *NhnCloudConnection) CreateTagHandler() (irs.TagHandler, error) {
	return nil, errors.New("NHN Cloud Driver: not implemented")
}
