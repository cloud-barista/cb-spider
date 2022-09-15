// Tencent Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Tencent Driver.
//
// by CB-Spider Team, 2022.09.

package tencent

import (
	"errors"

	tcon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/tencent/connect"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"

	cbs "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cbs/v20170312"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	cvm "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cvm/v20170312"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"

	cblog "github.com/cloud-barista/cb-log"
	"github.com/sirupsen/logrus"
)

type TencentDriver struct {
}

func (TencentDriver) GetDriverVersion() string {
	return "Test Tencent Driver Version 0.1"
}

func (TencentDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	drvCapabilityInfo.ImageHandler = true
	drvCapabilityInfo.VPCHandler = true
	drvCapabilityInfo.SecurityHandler = true
	drvCapabilityInfo.KeyPairHandler = true
	drvCapabilityInfo.VNicHandler = false
	drvCapabilityInfo.PublicIPHandler = false
	drvCapabilityInfo.VMHandler = true
	drvCapabilityInfo.VMSpecHandler = true
	drvCapabilityInfo.ClusterHandler = true

	return drvCapabilityInfo
}

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER TencentDriver")
}

func getVmClient(connectionInfo idrv.ConnectionInfo) (*cvm.Client, error) {
	// setup Region
	cblogger.Debug("TencentDriver : getVpcClient() - Region : [" + connectionInfo.RegionInfo.Region + "]")
	cblogger.Debug("TencentDriver : getVpcClient() - Zone : [" + connectionInfo.RegionInfo.Zone + "]")
	cblogger.Debug("TencentDriver : getVpcClient() - ClientId : [" + connectionInfo.CredentialInfo.ClientId + "]")

	zoneId := connectionInfo.RegionInfo.Zone
	if len(zoneId) < 1 {
		cblogger.Error("Connection 정보에 Zone 정보가 없습니다.")
		return nil, errors.New("Connection 정보에 Zone 정보가 없습니다")
	}

	credential := common.NewCredential(
		connectionInfo.CredentialInfo.ClientId,
		connectionInfo.CredentialInfo.ClientSecret,
	)

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "cvm.tencentcloudapi.com"
	cpf.Language = "en-US" //메시지를 영어로 설정
	client, err := cvm.NewClient(credential, connectionInfo.RegionInfo.Region, cpf)

	if err != nil {
		cblogger.Error("Could not create New Session")
		cblogger.Error(err)
		return nil, err
	}

	return client, nil
}

func getVpcClient(connectionInfo idrv.ConnectionInfo) (*vpc.Client, error) {
	// setup Region
	cblogger.Debug("TencentDriver : getVpcClient() - Region : [" + connectionInfo.RegionInfo.Region + "]")
	cblogger.Debug("TencentDriver : getVpcClient() - Zone : [" + connectionInfo.RegionInfo.Zone + "]")
	cblogger.Debug("TencentDriver : getVpcClient() - ClientId : [" + connectionInfo.CredentialInfo.ClientId + "]")

	zoneId := connectionInfo.RegionInfo.Zone
	if len(zoneId) < 1 {
		cblogger.Error("Connection 정보에 Zone 정보가 없습니다.")
		return nil, errors.New("Connection 정보에 Zone 정보가 없습니다")
	}

	credential := common.NewCredential(
		connectionInfo.CredentialInfo.ClientId,
		connectionInfo.CredentialInfo.ClientSecret,
	)

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "vpc.tencentcloudapi.com"
	cpf.Language = "en-US" //메시지를 영어로 설정
	client, err := vpc.NewClient(credential, connectionInfo.RegionInfo.Region, cpf)

	if err != nil {
		cblogger.Error("Could not create New Session")
		cblogger.Error(err)
		return nil, err
	}

	return client, nil
}

func getClbClient(connectionInfo idrv.ConnectionInfo) (*clb.Client, error) {
	// setup Region
	cblogger.Debug("TencentDriver : getVpcClient() - Region : [" + connectionInfo.RegionInfo.Region + "]")
	cblogger.Debug("TencentDriver : getVpcClient() - Zone : [" + connectionInfo.RegionInfo.Zone + "]")
	cblogger.Debug("TencentDriver : getVpcClient() - ClientId : [" + connectionInfo.CredentialInfo.ClientId + "]")

	zoneId := connectionInfo.RegionInfo.Zone
	if len(zoneId) < 1 {
		cblogger.Error("Connection 정보에 Zone 정보가 없습니다.")
		return nil, errors.New("Connection 정보에 Zone 정보가 없습니다")
	}

	credential := common.NewCredential(
		connectionInfo.CredentialInfo.ClientId,
		connectionInfo.CredentialInfo.ClientSecret,
	)

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "clb.tencentcloudapi.com"
	cpf.Language = "en-US" //메시지를 영어로 설정
	client, err := clb.NewClient(credential, connectionInfo.RegionInfo.Region, cpf)

	if err != nil {
		cblogger.Error("Could not create New Session")
		cblogger.Error(err)
		return nil, err
	}

	return client, nil
}

func getCbsClient(connectionInfo idrv.ConnectionInfo) (*cbs.Client, error) {
	// setup Region
	cblogger.Debug("TencentDriver : getVpcClient() - Region : [" + connectionInfo.RegionInfo.Region + "]")
	cblogger.Debug("TencentDriver : getVpcClient() - Zone : [" + connectionInfo.RegionInfo.Zone + "]")
	cblogger.Debug("TencentDriver : getVpcClient() - ClientId : [" + connectionInfo.CredentialInfo.ClientId + "]")

	zoneId := connectionInfo.RegionInfo.Zone
	if len(zoneId) < 1 {
		cblogger.Error("Connection 정보에 Zone 정보가 없습니다.")
		return nil, errors.New("Connection 정보에 Zone 정보가 없습니다")
	}

	credential := common.NewCredential(
		connectionInfo.CredentialInfo.ClientId,
		connectionInfo.CredentialInfo.ClientSecret,
	)

	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "cbs.tencentcloudapi.com"
	cpf.Language = "en-US" //메시지를 영어로 설정
	client, err := cbs.NewClient(credential, connectionInfo.RegionInfo.Region, cpf)

	if err != nil {
		cblogger.Error("Could not create New Session")
		cblogger.Error(err)
		return nil, err
	}

	return client, nil
}

func (driver *TencentDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	//fmt.Println("ConnectCloud의 전달 받은 idrv.ConnectionInfo 정보")
	//spew.Dump(connectionInfo)

	// sample code, do not user like this^^
	//var iConn icon.CloudConnection
	vmClient, err := getVmClient(connectionInfo)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	vpcClient, err := getVpcClient(connectionInfo)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	clbClient, err := getClbClient(connectionInfo)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	cbsClient, err := getCbsClient(connectionInfo)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	iConn := tcon.TencentCloudConnection{
		CredentialInfo: connectionInfo.CredentialInfo,
		Region:         connectionInfo.RegionInfo,
		VNetworkClient: vpcClient,
		NLBClient:      clbClient,
		VMClient:       vmClient,
		KeyPairClient:  vmClient,
		ImageClient:    vmClient,
		SecurityClient: vpcClient,
		VmSpecClient:   vmClient,
		DiskClient:     cbsClient,
		MyImageClient:  vmClient,

		//VNicClient:     vmClient,
		//PublicIPClient: vmClient,
	}

	return &iConn, nil // return type: (icon.CloudConnection, error)
}

/*
func (TencentDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.
	// sample code, do not user like this^^
	var iConn icon.CloudConnection
	iConn = tcon.TencentCloudConnection{}
	return iConn, nil // return type: (icon.CloudConnection, error)
}
*/
