// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2021.05.
// Updated by ETRI, 2023.10.

package connect

import (
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

	cblog "github.com/cloud-barista/cb-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	ktsdk "github.com/cloud-barista/ktcloud-sdk-go"
	//ktrs "github.com/cloud-barista/ktcloud/ktcloud/resources"
	ktrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ktcloud/resources"
)

type KtCloudConnection struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	Client         *ktsdk.KtCloudClient
	NLBClient      *ktsdk.KtCloudClient
}

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("KT Cloud Connect")
}

func (cloudConn *KtCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("KT Cloud Driver: called CreateVMHandler()!")

	//NOTE Just for Test!!
	// cblogger.Info("cloudConn.CredentialInfo.ClientId : ")
	// spew.Dump(cloudConn.CredentialInfo.ClientId)
	// cblogger.Info("cloudConn.RegionInfo : ")
	// spew.Dump(cloudConn.RegionInfo)

	vmHandler := ktrs.KtCloudVMHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		RegionInfo:     cloudConn.RegionInfo,
		Client:         cloudConn.Client,
	}
	return &vmHandler, nil
}

func (cloudConn *KtCloudConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	cblogger.Info("KT Cloud Driver: called CreateVMSpecHandler()!")
	vmspecHandler := ktrs.KtCloudVMSpecHandler{cloudConn.CredentialInfo, cloudConn.RegionInfo, cloudConn.Client}
	return &vmspecHandler, nil
}

func (cloudConn *KtCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("KT Cloud Driver: called CreateImagehandler()!")
	imageHandler := ktrs.KtCloudImageHandler{cloudConn.CredentialInfo, cloudConn.RegionInfo, cloudConn.Client}
	return &imageHandler, nil
}

func (cloudConn *KtCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("KT Cloud Driver: called CreateKeyPairHandler()!")
	keypairHandler := ktrs.KtCloudKeyPairHandler{cloudConn.CredentialInfo, cloudConn.RegionInfo, cloudConn.Client}
	return &keypairHandler, nil
}

func (cloudConn *KtCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("KT Cloud Driver: called CreateSecurityHandler()!")
	sgHandler := ktrs.KtCloudSecurityHandler{cloudConn.CredentialInfo, cloudConn.RegionInfo, cloudConn.Client}
	return &sgHandler, nil
}

func (cloudConn *KtCloudConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	cblogger.Info("KT Cloud Driver: called CreateVPCHandler()!")
	vpcHandler := ktrs.KtCloudVPCHandler{cloudConn.CredentialInfo, cloudConn.RegionInfo, cloudConn.Client}
	return &vpcHandler, nil
}

func (cloudConn *KtCloudConnection) CreateNLBHandler() (irs.NLBHandler, error) {
	cblogger.Info("KT Cloud Driver: called CreateNLBHandler()!")
	nlbHandler := ktrs.KtCloudNLBHandler{RegionInfo: cloudConn.RegionInfo, Client: cloudConn.Client, NLBClient: cloudConn.NLBClient}
	return &nlbHandler, nil
}

func (cloudConn *KtCloudConnection) CreateDiskHandler() (irs.DiskHandler, error) {
	cblogger.Info("KT Cloud Driver: called CreateDiskHandler()!")
	diskHandler := ktrs.KtCloudDiskHandler{RegionInfo: cloudConn.RegionInfo, Client: cloudConn.Client}
	return &diskHandler, nil
}

func (cloudConn *KtCloudConnection) CreateClusterHandler() (irs.ClusterHandler, error) {
	cblogger.Info("KT Cloud Driver: called CreateClusterHandler()!")
	return nil, fmt.Errorf("KT Cloud Driver does not support CreateClusterHandler yet.")
}

func (cloudConn *KtCloudConnection) CreateMyImageHandler() (irs.MyImageHandler, error) {
	cblogger.Info("KT Cloud Driver: called CreateMyImageHandler()!")
	myimageHandler := ktrs.KtCloudMyImageHandler{RegionInfo: cloudConn.RegionInfo, Client: cloudConn.Client}
	return &myimageHandler, nil
}

func (cloudConn *KtCloudConnection) CreateAnyCallHandler() (irs.AnyCallHandler, error) {
	cblogger.Info("KT Cloud Driver: called CreateAnyCallHandler()!")
	return nil, fmt.Errorf("KT Cloud Driver does not support CreateAnyCallHandler yet.")
}

func (cloudConn *KtCloudConnection) CreateRegionZoneHandler() (irs.RegionZoneHandler, error) {
	cblogger.Info("KT Cloud Driver: called CreateRegionZoneHandler()!")
	regionZoneHandler := ktrs.KtCloudRegionZoneHandler{CredentialInfo: cloudConn.CredentialInfo, RegionInfo: cloudConn.RegionInfo, Client: cloudConn.Client}
	return &regionZoneHandler, nil
}

func (cloudConn *KtCloudConnection) IsConnected() (bool, error) {
	cblogger.Info("KT Cloud Driver: called IsConnected()!")
	if cloudConn == nil {
		return false, nil
	}
	return true, nil
}

func (cloudConn *KtCloudConnection) Close() error {
	cblogger.Info("KT Cloud Driver: called Close()!")
	return nil
}

func (*KtCloudConnection) CreatePriceInfoHandler() (irs.PriceInfoHandler, error) {
	return nil, errors.New("KT Cloud Driver: not implemented")
}
