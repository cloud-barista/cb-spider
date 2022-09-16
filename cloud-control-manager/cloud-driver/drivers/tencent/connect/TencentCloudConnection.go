// Tencent Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//   - Cloud-Barista: https://github.com/cloud-barista
//
// This is Tencent Driver.
//
// by CB-Spider Team, 2022.09.

package connect

import (
	cblog "github.com/cloud-barista/cb-log"
	trs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/tencent/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"

	//"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	//"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/errors"
	//"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"

	cbs "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cbs/v20170312"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"

	"errors"
)

type TencentCloudConnection struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	VNetworkClient *vpc.Client
	NLBClient      *clb.Client
	VMClient       *cvm.Client
	KeyPairClient  *cvm.Client
	ImageClient    *cvm.Client
	SecurityClient *vpc.Client
	VmSpecClient   *cvm.Client
	DiskClient     *cbs.Client
	MyImageClient  *cvm.Client
	//VNicClient     *cvm.Client
	//PublicIPClient *cvm.Client
}

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER TencentCloudConnection")
}

func (cloudConn *TencentCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("Start CreateKeyPairHandler()")

	keyPairHandler := trs.TencentKeyPairHandler{cloudConn.Region, cloudConn.KeyPairClient}

	return &keyPairHandler, nil
}

func (cloudConn *TencentCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("Start CreateVMHandler()")

	vmHandler := trs.TencentVMHandler{cloudConn.Region, cloudConn.VMClient, cloudConn.DiskClient}
	return &vmHandler, nil
}

func (cloudConn *TencentCloudConnection) IsConnected() (bool, error) {
	return true, nil
}
func (cloudConn *TencentCloudConnection) Close() error {
	return nil
}

func (cloudConn *TencentCloudConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	cblogger.Info("Start")
	handler := trs.TencentVPCHandler{cloudConn.Region, cloudConn.VNetworkClient}

	return &handler, nil
}

func (cloudConn *TencentCloudConnection) CreateNLBHandler() (irs.NLBHandler, error) {
	cblogger.Info("Start")
	handler := trs.TencentNLBHandler{cloudConn.Region, cloudConn.NLBClient, cloudConn.VNetworkClient}

	return &handler, nil
}

func (cloudConn *TencentCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("Start")
	handler := trs.TencentImageHandler{cloudConn.Region, cloudConn.ImageClient}

	return &handler, nil
}

func (cloudConn *TencentCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("Start")
	handler := trs.TencentSecurityHandler{cloudConn.Region, cloudConn.SecurityClient}

	return &handler, nil
}

func (cloudConn *TencentCloudConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	cblogger.Info("Start")
	handler := trs.TencentVmSpecHandler{cloudConn.Region, cloudConn.VmSpecClient}
	return &handler, nil
}

/*
func (cloudConn *TencentCloudConnection) CreateVNicHandler() (irs.VNicHandler, error) {
	cblogger.Info("Start")
	handler := trs.TencentVNicHandler{cloudConn.Region, cloudConn.VNicClient}

	return &handler, nil
}

func (cloudConn *TencentCloudConnection) CreatePublicIPHandler() (irs.PublicIPHandler, error) {
	cblogger.Info("Start")
	handler := trs.TencentPublicIPHandler{cloudConn.Region, cloudConn.PublicIPClient}

	return &handler, nil
}
*/

func (cloudConn *TencentCloudConnection) CreateDiskHandler() (irs.DiskHandler, error) {

	cblogger.Info("Start")
	handler := trs.TencentDiskHandler{cloudConn.Region, cloudConn.DiskClient}

	return &handler, nil
}

func (cloudConn *TencentCloudConnection) CreateMyImageHandler() (irs.MyImageHandler, error) {
	cblogger.Info("Start")
	handler := trs.TencentMyImageHandler{cloudConn.Region, cloudConn.MyImageClient}

	return &handler, nil
}

func (cloudConn *TencentCloudConnection) CreateClusterHandler() (irs.ClusterHandler, error) {
	// temp
	// getEnv & Setting
	clusterHandler := trs.TencentClusterHandler{RegionInfo: cloudConn.Region, CredentialInfo: cloudConn.CredentialInfo}

	return &clusterHandler, nil

}


func (cloudConn *TencentCloudConnection) CreateAnyCallHandler() (irs.AnyCallHandler, error) {
	return nil, errors.New("Tencent Driver: not implemented")
}

