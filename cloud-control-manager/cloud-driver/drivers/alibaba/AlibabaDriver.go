// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by zephy@mz.co.kr, 2019.09.

package alibaba

import (
	"fmt"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	alicon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/alibaba/connect"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
)

type AlibabaDriver struct{}

func (AlibabaDriver) GetDriverVersion() string {
	return "ALIBABA-CLOUD DRIVER Version 1.0"
}

func (AlibabaDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	drvCapabilityInfo.ImageHandler = false
	drvCapabilityInfo.VNetworkHandler = false
	drvCapabilityInfo.SecurityHandler = false
	drvCapabilityInfo.KeyPairHandler = false
	drvCapabilityInfo.VNicHandler = false
	drvCapabilityInfo.PublicIPHandler = false
	drvCapabilityInfo.VMHandler = false

	return drvCapabilityInfo
}

func (driver *AlibabaDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	ECSClient, err := getECSClient(connectionInfo)
	if err != nil {
		return nil, err
	}

	VPCClient, err := getVPCClient(connectionInfo)
	if err != nil {
		return nil, err
	}

	iConn := alicon.AlibabaCloudConnection{
		Region:              connectionInfo.RegionInfo,
		VMClient:            ECSClient,
		KeyPairClient:       ECSClient,
		ImageClient:         ECSClient,
		PublicIPClient:      VPCClient,
		SecurityGroupClient: ECSClient,
		VNetClient:          VPCClient,
		VNicClient:          ECSClient,
		SubnetClient:        VPCClient,
	}
	return &iConn, nil
}

func getECSClient(connectionInfo idrv.ConnectionInfo) (*ecs.Client, error) {

	// Region Info
	fmt.Println("AlibabaDriver : getECSClient() - Region : [" + connectionInfo.RegionInfo.Region + "]")

	// Customize config
	config := sdk.NewConfig().
		WithEnableAsync(true).
		WithGoRoutinePoolSize(5).
		WithMaxTaskQueueSize(1000)
		// 600*time.Second

	fmt.Println(config)

	// Create a credential object
	credential := &credentials.BaseCredential{
		AccessKeyId:     connectionInfo.CredentialInfo.ClientId,
		AccessKeySecret: connectionInfo.CredentialInfo.ClientSecret,
	}

	escClient, err := ecs.NewClientWithAccessKey(connectionInfo.RegionInfo.Region, credential.AccessKeyId, credential.AccessKeySecret)
	//escClient, err := ecs.NewClientWithOptions(connectionInfo.RegionInfo.Region, config, credential)
	if err != nil {
		fmt.Println("Could not create alibaba's ecs service client", err)
		return nil, err
	}

	/*
		escClient, err := sdk.NewClientWithAccessKey("REGION_ID", "ACCESS_KEY_ID", "ACCESS_KEY_SECRET")
		if err != nil {
			// Handle exceptions
			panic(err)
		}
	*/

	return escClient, nil
}

func getVPCClient(connectionInfo idrv.ConnectionInfo) (*vpc.Client, error) {

	// Region Info
	fmt.Println("AlibabaDriver : getVPCClient() - Region : [" + connectionInfo.RegionInfo.Region + "]")

	// Customize config
	config := sdk.NewConfig().
		WithEnableAsync(true).
		WithGoRoutinePoolSize(5).
		WithMaxTaskQueueSize(1000)
	// 600*time.Second

	fmt.Println(config)

	// Create a credential object
	credential := &credentials.BaseCredential{
		AccessKeyId:     connectionInfo.CredentialInfo.ClientId,
		AccessKeySecret: connectionInfo.CredentialInfo.ClientSecret,
	}

	vpcClient, err := vpc.NewClientWithAccessKey(connectionInfo.RegionInfo.Region, credential.AccessKeyId, credential.AccessKeySecret)
	//vpcClient, err := vpc.NewClientWithOptions(connectionInfo.RegionInfo.Region, config, credential)
	if err != nil {
		fmt.Println("Could not create alibaba's vpc service client", err)
		return nil, err
	}

	return vpcClient, nil
}

var CloudDriver AlibabaDriver
