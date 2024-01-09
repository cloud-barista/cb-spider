// Alibaba Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Alibaba Driver.
//
// by CB-Spider Team, 2022.09.

package alibaba

import (
	"fmt"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/bssopenapi"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"

	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	alicon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/alibaba/connect"
	alirs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/alibaba/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	"github.com/davecgh/go-spew/spew"
)

type AlibabaDriver struct{}

func (AlibabaDriver) GetDriverVersion() string {
	return "ALIBABA-CLOUD DRIVER Version 1.0"
}

func (AlibabaDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	drvCapabilityInfo.ImageHandler = true
	drvCapabilityInfo.VPCHandler = true
	drvCapabilityInfo.SecurityHandler = true
	drvCapabilityInfo.KeyPairHandler = true
	drvCapabilityInfo.VNicHandler = false
	drvCapabilityInfo.PublicIPHandler = false
	drvCapabilityInfo.VMHandler = true
	drvCapabilityInfo.VMSpecHandler = true
	drvCapabilityInfo.DiskHandler = true
	drvCapabilityInfo.ClusterHandler = true
	drvCapabilityInfo.RegionZoneHandler = true
	drvCapabilityInfo.PriceInfoHandler = true

	return drvCapabilityInfo
}

func (driver *AlibabaDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	// Initialize Logger
	alirs.InitLog()

	ECSClient, err := getECSClient(connectionInfo)
	if err != nil {
		return nil, err
	}

	VPCClient, err := getVPCClient(connectionInfo)
	if err != nil {
		return nil, err
	}

	NLBClient, err := getNLBClient(connectionInfo)
	if err != nil {
		return nil, err
	}

	BssClient, err := getBssClient(connectionInfo)
	if err != nil {
		return nil, err
	}

	iConn := alicon.AlibabaCloudConnection{
		CredentialInfo: connectionInfo.CredentialInfo,
		Region:         connectionInfo.RegionInfo,
		VMClient:       ECSClient,
		KeyPairClient:  ECSClient,
		ImageClient:    ECSClient,
		//PublicIPClient:      VPCClient,
		SecurityGroupClient: ECSClient,
		VpcClient:           VPCClient,
		//VNetClient:          VPCClient,
		//VNicClient:          ECSClient,
		//SubnetClient: VPCClient,
		VmSpecClient:     ECSClient,
		NLBClient:        NLBClient,
		DiskClient:       ECSClient,
		MyImageClient:    ECSClient,
		RegionZoneClient: ECSClient,
		BssClient:        BssClient,
	}
	return &iConn, nil
}

func getECSClient(connectionInfo idrv.ConnectionInfo) (*ecs.Client, error) {

	// Region Info
	fmt.Println("AlibabaDriver : getECSClient() - Region : [" + connectionInfo.RegionInfo.Region + "]")

	/*
		// Customize config
		config := sdk.NewConfig().
			WithEnableAsync(true).
			WithGoRoutinePoolSize(5).
			WithMaxTaskQueueSize(1000)
			// 600*time.Second

		//fmt.Println(config)
		spew.Dump(config)
	*/

	// Create a credential object
	/* BaseCredential는 deprecated 되었음.
	credential := &credentials.BaseCredential{
		AccessKeyId:     connectionInfo.CredentialInfo.ClientId,
		AccessKeySecret: connectionInfo.CredentialInfo.ClientSecret,
	}
	*/

	credential := &credentials.AccessKeyCredential{
		AccessKeyId:     connectionInfo.CredentialInfo.ClientId,
		AccessKeySecret: connectionInfo.CredentialInfo.ClientSecret,
	}

	config := sdk.NewConfig()
	config.Timeout = time.Duration(15) * time.Second //time.Millisecond
	config.AutoRetry = true
	config.MaxRetryTime = 2
	//sdk.Timeout(1000)

	//escClient, err := ecs.NewClientWithAccessKey(connectionInfo.RegionInfo.Region, credential.AccessKeyId, credential.AccessKeySecret)

	escClient, err := ecs.NewClientWithOptions(connectionInfo.RegionInfo.Region, config, credential)
	if err != nil {
		fmt.Println("Could not create alibaba's ecs service client", err)
		spew.Dump(err)
		return nil, err
	}

	//spew.Dump(escClient)

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

	/*
		// Customize config
		config := sdk.NewConfig().
			WithEnableAsync(true).
			WithGoRoutinePoolSize(5).
			WithMaxTaskQueueSize(1000)
		// 600*time.Second
		//fmt.Println(config)
	*/

	// Create a credential object
	/* BaseCredential는 deprecated 되었음.
	credential := &credentials.BaseCredential{
		AccessKeyId:     connectionInfo.CredentialInfo.ClientId,
		AccessKeySecret: connectionInfo.CredentialInfo.ClientSecret,
	}
	*/

	credential := &credentials.AccessKeyCredential{
		AccessKeyId:     connectionInfo.CredentialInfo.ClientId,
		AccessKeySecret: connectionInfo.CredentialInfo.ClientSecret,
	}

	config := sdk.NewConfig()
	config.Timeout = time.Duration(15) * time.Second //time.Millisecond
	config.AutoRetry = true
	config.MaxRetryTime = 2
	//sdk.Timeout(1000)

	//vpcClient, err := vpc.NewClientWithAccessKey(connectionInfo.RegionInfo.Region, credential.AccessKeyId, credential.AccessKeySecret)
	vpcClient, err := vpc.NewClientWithOptions(connectionInfo.RegionInfo.Region, config, credential)
	if err != nil {
		fmt.Println("Could not create alibaba's vpc service client", err)
		return nil, err
	}

	return vpcClient, nil
}

func getNLBClient(connectionInfo idrv.ConnectionInfo) (*slb.Client, error) {

	// Region Info
	fmt.Println("AlibabaDriver : getNLBClient() - Region : [" + connectionInfo.RegionInfo.Region + "]")

	credential := &credentials.AccessKeyCredential{
		AccessKeyId:     connectionInfo.CredentialInfo.ClientId,
		AccessKeySecret: connectionInfo.CredentialInfo.ClientSecret,
	}

	config := sdk.NewConfig()
	config.Timeout = time.Duration(15) * time.Second //time.Millisecond
	config.AutoRetry = true
	config.MaxRetryTime = 2
	//sdk.Timeout(1000)

	nlbClient, err := slb.NewClientWithOptions(connectionInfo.RegionInfo.Region, config, credential)
	if err != nil {
		fmt.Println("Could not create alibaba's server loadbalancer service client", err)
		return nil, err
	}

	return nlbClient, nil
}

func getBssClient(connectionInfo idrv.ConnectionInfo) (*bssopenapi.Client, error) {
	// Region Info
	fmt.Println("AlibabaDriver : getNLBClient() - Region : [" + connectionInfo.RegionInfo.Region + "]")

	credential := &credentials.AccessKeyCredential{
		AccessKeyId:     connectionInfo.CredentialInfo.ClientId,
		AccessKeySecret: connectionInfo.CredentialInfo.ClientSecret,
	}

	config := sdk.NewConfig()
	config.Timeout = time.Duration(15) * time.Second //time.Millisecond
	config.AutoRetry = true
	config.MaxRetryTime = 2

	//sdk.Timeout(1000)

	// https://api.alibabacloud.com/document/BssOpenApi/2017-12-14/DescribeResourcePackageProduct?spm=api-workbench-intl.api_explorer.0.0.777f813524Q25K
	// API docs 상 가능 Region 명시되지 않음, 티켓에서도 별도 안내 없음.
	// 모든 Region 테스트 결과 아래 6개 리전에서 bss API 권한을 가진 상태로 정상 결과 응답
	// ++ QueryProductList 는 클라이언트에서 Region 정보를 가져오는 것이 아닌, 별도 Input 으로 리전 받음, 따라서 별도 Client 에 리전 셋팅 필요 없음.
	// ++ 제공되는 Product 는 23.12.18 현재 123개로 모든 리전에서 동일한 응답을 확인
	// Tested request Region
	// us-east-1, us-west-1, eu-west-1, eu-central-1, ap-south-1, me-east-1,

	pricingRegion := []string{"us-east-1", "us-west-1", "eu-west-1", "eu-central-1", "ap-south-1", "me-east-1"} // updated : 23.12.18
	match := false
	for _, str := range pricingRegion {
		if str == connectionInfo.RegionInfo.Region {
			match = true
			break
		}
	}

	var targetRegion string
	if match {
		targetRegion = connectionInfo.RegionInfo.Region
	} else {
		targetRegion = "us-east-1"
	}

	bssClient, err := bssopenapi.NewClientWithOptions(targetRegion, config, credential)
	if err != nil {
		fmt.Println("Could not create alibaba's server bss open api client", err)
		return nil, err
	}

	return bssClient, nil
}
