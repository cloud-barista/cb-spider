package ibmcloudvpc

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/IBM/cloud-databases-go-sdk/clouddatabasesv5"
	"github.com/IBM/platform-services-go-sdk/globalsearchv2"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/globaltaggingv1"
	"github.com/IBM/platform-services-go-sdk/resourcecontrollerv2"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibm/connect"
	ibms "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibm/resources"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibm/utils/kubernetesserviceapiv1"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	ires "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type IbmCloudDriver struct{}

const (
	cspTimeout time.Duration = 6000
)

// iamAuthCache shares one IamAuthenticator (and its cached IAM token) per ApiKey.
var (
	iamAuthCache   = map[string]*core.IamAuthenticator{}
	iamAuthCacheMu sync.Mutex
)

func getIamAuthenticator(apiKey string) *core.IamAuthenticator {
	iamAuthCacheMu.Lock()
	defer iamAuthCacheMu.Unlock()
	if auth, ok := iamAuthCache[apiKey]; ok {
		return auth
	}
	auth := &core.IamAuthenticator{ApiKey: apiKey}
	iamAuthCache[apiKey] = auth
	return auth
}

// regionEndpointCache caches the VPC endpoint per validated (region, zone) pair.
var (
	regionEndpointCache   = map[string]string{}
	regionEndpointCacheMu sync.Mutex
)

func (IbmCloudDriver) GetDriverVersion() string {
	return "IBM DRIVER Version 1.0"
}
func (IbmCloudDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	drvCapabilityInfo.ZoneBasedControl = true

	drvCapabilityInfo.RegionZoneHandler = true
	drvCapabilityInfo.PriceInfoHandler = true
	drvCapabilityInfo.ImageHandler = true
	drvCapabilityInfo.VMSpecHandler = true

	drvCapabilityInfo.VPCHandler = true
	drvCapabilityInfo.SecurityHandler = true
	drvCapabilityInfo.KeyPairHandler = true
	drvCapabilityInfo.VMHandler = true
	drvCapabilityInfo.DiskHandler = true
	drvCapabilityInfo.MyImageHandler = true
	drvCapabilityInfo.NLBHandler = true
	drvCapabilityInfo.ClusterHandler = true
	drvCapabilityInfo.FileSystemHandler = true
	drvCapabilityInfo.QuotaInfoHandler = false

	drvCapabilityInfo.RDBMSHandler = true
	drvCapabilityInfo.PublicIPHandler = true

	drvCapabilityInfo.TagHandler = true
	drvCapabilityInfo.TagSupportResourceType = []ires.RSType{ires.VPC, ires.SUBNET, ires.SG, ires.KEY, ires.VM, ires.NLB, ires.DISK, ires.MYIMAGE, ires.CLUSTER, ires.RDBMS}

	drvCapabilityInfo.VPC_CIDR = false

	return drvCapabilityInfo
}

func (driver *IbmCloudDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	ibms.InitLog()
	err := checkConnectionInfo(connectionInfo)
	if err != nil {
		return nil, err
	}
	ctx, _ := context.WithTimeout(context.Background(), cspTimeout*time.Second)

	auth := getIamAuthenticator(connectionInfo.CredentialInfo.ApiKey)

	// Region & Zone Check (cached per region/zone pair)
	endpointCacheKey := connectionInfo.RegionInfo.Region + "/" + connectionInfo.RegionInfo.Zone
	regionEndpointCacheMu.Lock()
	endPoint, cached := regionEndpointCache[endpointCacheKey]
	regionEndpointCacheMu.Unlock()
	if !cached {
		initVpcService, err := vpcv1.NewVpcV1(&vpcv1.VpcV1Options{
			Authenticator: auth,
		})
		if err != nil {
			return nil, err
		}
		getRegionOptions := &vpcv1.GetRegionOptions{}
		getRegionOptions.SetName(connectionInfo.RegionInfo.Region)
		region, _, err := initVpcService.GetRegionWithContext(ctx, getRegionOptions)
		if err != nil {
			return nil, err
		}
		getZoneOptions := &vpcv1.GetRegionZoneOptions{}
		getZoneOptions.SetRegionName(*region.Name)
		getZoneOptions.SetName(connectionInfo.RegionInfo.Zone)
		_, _, err = initVpcService.GetRegionZoneWithContext(ctx, getZoneOptions)
		if err != nil {
			return nil, err
		}
		endPoint = *region.Endpoint + "/v1"
		regionEndpointCacheMu.Lock()
		regionEndpointCache[endpointCacheKey] = endPoint
		regionEndpointCacheMu.Unlock()
	}
	vpcService, err := vpcv1.NewVpcV1(&vpcv1.VpcV1Options{
		Authenticator: auth,
		URL:           endPoint,
	})
	if err != nil {
		return nil, err
	}
	clusterService, err := kubernetesserviceapiv1.NewKubernetesServiceApiV1(&kubernetesserviceapiv1.KubernetesServiceApiV1Options{
		Authenticator: auth,
	})
	if err != nil {
		return nil, err
	}
	taggingService, err := globaltaggingv1.NewGlobalTaggingV1(&globaltaggingv1.GlobalTaggingV1Options{
		Authenticator: auth,
	})
	if err != nil {
		return nil, err
	}
	searchService, err := globalsearchv2.NewGlobalSearchV2(&globalsearchv2.GlobalSearchV2Options{
		Authenticator: auth,
	})
	if err != nil {
		return nil, err
	}

	resourceControllerService, err := resourcecontrollerv2.NewResourceControllerV2(&resourcecontrollerv2.ResourceControllerV2Options{
		Authenticator: auth,
	})
	if err != nil {
		return nil, err
	}

	cloudDBService, err := clouddatabasesv5.NewCloudDatabasesV5(&clouddatabasesv5.CloudDatabasesV5Options{
		Authenticator: auth,
	})
	if err != nil {
		return nil, err
	}

	iConn := connect.IbmCloudConnection{
		CredentialInfo:     connectionInfo.CredentialInfo,
		Region:             connectionInfo.RegionInfo,
		VpcService:         vpcService,
		ClusterService:     clusterService,
		TaggingService:     taggingService,
		SearchService:      searchService,
		ResourceController: resourceControllerService,
		CloudDBService:     cloudDBService,
		Ctx:                ctx,
	}
	return &iConn, nil
}

func checkConnectionInfo(connectionInfo idrv.ConnectionInfo) error {
	if connectionInfo.CredentialInfo.ApiKey == "" {
		return errors.New("not exist ApiKey")
	}
	if connectionInfo.RegionInfo.Region == "" {
		return errors.New("not exist Region")
	}
	if connectionInfo.RegionInfo.Zone == "" {
		return errors.New("not exist Zone")
	}
	return nil
}
