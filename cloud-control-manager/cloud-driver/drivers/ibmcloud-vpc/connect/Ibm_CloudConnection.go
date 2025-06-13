package connect

import (
	"context"
	"errors"

	"github.com/IBM/platform-services-go-sdk/globalsearchv2"

	"github.com/IBM/platform-services-go-sdk/globaltaggingv1"
	vpcv0230 "github.com/IBM/vpc-go-sdk/0.23.0/vpcv1"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	cblog "github.com/cloud-barista/cb-log"
	ibmrs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibmcloud-vpc/resources"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ibmcloud-vpc/utils/kubernetesserviceapiv1"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type IbmCloudConnection struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	VpcService     *vpcv1.VpcV1
	ClusterService *kubernetesserviceapiv1.KubernetesServiceApiV1
	TaggingService *globaltaggingv1.GlobalTaggingV1
	SearchService  *globalsearchv2.GlobalSearchV2
	VpcService0230 *vpcv0230.VpcV1
	Ctx            context.Context
}

// CreateFileSystemHandler implements connect.CloudConnection.
func (cloudConn *IbmCloudConnection) CreateFileSystemHandler() (irs.FileSystemHandler, error) {
	panic("unimplemented")
}

func (cloudConn *IbmCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateImageHandler()!")
	imageHandler := ibmrs.IbmImageHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
	}
	return &imageHandler, nil
}

func (cloudConn *IbmCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateVMHandler()!")
	vmHandler := ibmrs.IbmVMHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		VpcService0230: cloudConn.VpcService0230,
		Ctx:            cloudConn.Ctx,
		TaggingService: cloudConn.TaggingService,
		SearchService:  cloudConn.SearchService,
	}
	return &vmHandler, nil
}

func (cloudConn *IbmCloudConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateVPCHandler()!")
	vpcHandler := ibmrs.IbmVPCHandler{
		Region:         cloudConn.Region,
		CredentialInfo: cloudConn.CredentialInfo,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
		TaggingService: cloudConn.TaggingService,
		SearchService:  cloudConn.SearchService,
	}
	return &vpcHandler, nil
}
func (cloudConn *IbmCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateSecurityHandler()!")
	securityHandler := ibmrs.IbmSecurityHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
		TaggingService: cloudConn.TaggingService,
		SearchService:  cloudConn.SearchService,
	}
	return &securityHandler, nil
}
func (cloudConn *IbmCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateVPCHandler()!")
	keyPairHandler := ibmrs.IbmKeyPairHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
		TaggingService: cloudConn.TaggingService,
		SearchService:  cloudConn.SearchService,
	}
	return &keyPairHandler, nil
}
func (cloudConn *IbmCloudConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateVMSpecHandler()!")
	vmSpecHandler := ibmrs.IbmVmSpecHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
	}
	return &vmSpecHandler, nil
}

func (cloudConn *IbmCloudConnection) CreateNLBHandler() (irs.NLBHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateNLBHandler()!")
	nlbHandler := ibmrs.IbmNLBHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
		TaggingService: cloudConn.TaggingService,
		SearchService:  cloudConn.SearchService,
	}
	return &nlbHandler, nil
}

func (cloudConn *IbmCloudConnection) IsConnected() (bool, error) {
	cblogger.Info("Ibm Cloud Driver: called IsConnected()!")
	return true, nil
}
func (cloudConn *IbmCloudConnection) Close() error {
	cblogger.Info("Ibm Cloud Driver: called Close()!")
	return nil
}

func (cloudConn *IbmCloudConnection) CreateDiskHandler() (irs.DiskHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateDiskHandler()!")
	diskHandler := ibmrs.IbmDiskHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
		TaggingService: cloudConn.TaggingService,
		SearchService:  cloudConn.SearchService,
	}
	return &diskHandler, nil
}

func (cloudConn *IbmCloudConnection) CreateClusterHandler() (irs.ClusterHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateClusterHandler()!")
	clusterHandler := ibmrs.IbmClusterHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		ClusterService: cloudConn.ClusterService,
		Ctx:            cloudConn.Ctx,
		TaggingService: cloudConn.TaggingService,
		SearchService:  cloudConn.SearchService,
	}
	return &clusterHandler, nil
}

func (cloudConn *IbmCloudConnection) CreateMyImageHandler() (irs.MyImageHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateMyImageHandler()!")
	myIamgeHandler := ibmrs.IbmMyImageHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
		TaggingService: cloudConn.TaggingService,
		SearchService:  cloudConn.SearchService,
	}
	return &myIamgeHandler, nil
}

func (cloudConn *IbmCloudConnection) CreateAnyCallHandler() (irs.AnyCallHandler, error) {
	return nil, errors.New("Ibm Driver: not implemented")
}

func (cloudConn *IbmCloudConnection) CreateRegionZoneHandler() (irs.RegionZoneHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateRegionZoneHandler()!")
	regionZoneHandler := ibmrs.IbmRegionZoneHandler{
		Region:     cloudConn.Region,
		VpcService: cloudConn.VpcService,
		Ctx:        cloudConn.Ctx,
	}
	return &regionZoneHandler, nil
}

func (cloudConn *IbmCloudConnection) CreatePriceInfoHandler() (irs.PriceInfoHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreatePriceInfoHandler()!")
	priceInfoHandler := ibmrs.IbmPriceInfoHandler{
		CredentialInfo: cloudConn.CredentialInfo,
		Region:         cloudConn.Region,
		VpcService:     cloudConn.VpcService,
		Ctx:            cloudConn.Ctx,
	}
	return &priceInfoHandler, nil
}

func (cloudConn *IbmCloudConnection) CreateTagHandler() (irs.TagHandler, error) {
	cblogger.Info("Ibm Cloud Driver: called CreateTagHandler()!")
	TagHandler := ibmrs.IbmTagHandler{
		Region:         cloudConn.Region,
		CredentialInfo: cloudConn.CredentialInfo,
		VpcService:     cloudConn.VpcService,
		ClusterService: cloudConn.ClusterService,
		Ctx:            cloudConn.Ctx,
		TaggingService: cloudConn.TaggingService,
		SearchService:  cloudConn.SearchService,
	}
	return &TagHandler, nil
}
