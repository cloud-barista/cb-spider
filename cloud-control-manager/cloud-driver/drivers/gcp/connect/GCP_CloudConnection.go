// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by jaz, 2019.07.

package connect

import (
	"context"
	"errors"

	cblog "github.com/cloud-barista/cb-log"
	gcprs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/gcp/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/sirupsen/logrus"
	compute "google.golang.org/api/compute/v1"
)

var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}

type GCPCloudConnection struct {
	Region              idrv.RegionInfo
	Credential          idrv.CredentialInfo
	Ctx                 context.Context
	VMClient            *compute.Service
	ImageClient         *compute.Service
	PublicIPClient      *compute.Service
	SecurityGroupClient *compute.Service
	VNetClient          *compute.Service
	VNicClient          *compute.Service
	SubnetClient        *compute.Service
	VMSpecHandler       *compute.Service
	VPCHandler          *compute.Service
}

// func (cloudConn *GCPCloudConnection) CreateVNetworkHandler() (irs.VNetworkHandler, error) {
// 	cblogger.Info("GCP Cloud Driver: called CreateVNetworkHandler()!")

// 	vNetHandler := gcprs.GCPVNetworkHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.VNetClient, cloudConn.Credential}
// 	return &vNetHandler, nil
// }

func (cloudConn *GCPCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
	cblogger.Info("GCP Cloud Driver: called CreateImageHandler()!")
	imageHandler := gcprs.GCPImageHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.ImageClient, cloudConn.Credential}
	return &imageHandler, nil
}

func (cloudConn *GCPCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	cblogger.Info("GCP Cloud Driver: called CreateSecurityHandler()!")
	sgHandler := gcprs.GCPSecurityHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.SecurityGroupClient, cloudConn.Credential}
	return &sgHandler, nil
}

func (cloudConn *GCPCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
	cblogger.Info("GCP Cloud Driver: called CreateKeyPairHandler()!")
	keypairHandler := gcprs.GCPKeyPairHandler{cloudConn.Credential, cloudConn.Region}
	return &keypairHandler, nil
}

// func (cloudConn *GCPCloudConnection) CreateVNicHandler() (irs.VNicHandler, error) {
// 	cblogger.Info("GCP Cloud Driver: called CreateVNicHandler()!")
// 	vNicHandler := gcprs.GCPVNicHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.VNicClient, cloudConn.Credential}
// 	return &vNicHandler, nil
// }

// func (cloudConn *GCPCloudConnection) CreatePublicIPHandler() (irs.PublicIPHandler, error) {
// 	cblogger.Info("GCP Cloud Driver: called CreatePublicIPHandler()!")
// 	publicIPHandler := gcprs.GCPPublicIPHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.PublicIPClient, cloudConn.Credential}
// 	return &publicIPHandler, nil
// }

func (cloudConn *GCPCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	cblogger.Info("GCP Cloud Driver: called CreateVMHandler()!")
	vmHandler := gcprs.GCPVMHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.VMClient, cloudConn.Credential}
	return &vmHandler, nil
}

func (cloudConn *GCPCloudConnection) CreateVPCHandler() (irs.VPCHandler, error) {
	cblogger.Info("GCP Cloud Driver: called CreateVPCHandler()!")
	vpcHandler := gcprs.GCPVPCHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.VMClient, cloudConn.Credential}
	return &vpcHandler, nil
}

func (cloudConn *GCPCloudConnection) CreateVMSpecHandler() (irs.VMSpecHandler, error) {
	cblogger.Info("GCP Cloud Driver: called CreateVMSpecHandler()!")
	vmSpecHandler := gcprs.GCPVMSpecHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.VMClient, cloudConn.Credential}
	return &vmSpecHandler, nil
}

func (cloudConn *GCPCloudConnection) CreateNLBHandler() (irs.NLBHandler, error) {
	cblogger.Info("GCP Cloud Driver: called CreateLoadBalancerHandler()!")
	nlbHandler := gcprs.GCPNLBHandler{Region: cloudConn.Region, Ctx: cloudConn.Ctx, Client: cloudConn.VMClient, Credential: cloudConn.Credential}
	return &nlbHandler, nil
}

func (GCPCloudConnection) IsConnected() (bool, error) {
	return true, nil
}
func (GCPCloudConnection) Close() error {
	return nil
}

func (cloudConn *GCPCloudConnection) CreateDiskHandler() (irs.DiskHandler, error) {
	cblogger.Info("GCP Cloud Driver: called CreateDiskHandler()!")
	diskHandler := gcprs.GCPDiskHandler{Region: cloudConn.Region, Ctx: cloudConn.Ctx, Client: cloudConn.VMClient, Credential: cloudConn.Credential}
	return &diskHandler, nil
}

func (cloudConn *GCPCloudConnection) CreateMyImageHandler() (irs.MyImageHandler, error) {
	cblogger.Info("GCP Cloud Driver: called CreateMyImageHandler()!")
	myImageHandler := gcprs.GCPMyImageHandler{Region: cloudConn.Region, Ctx: cloudConn.Ctx, Client: cloudConn.VMClient, Credential: cloudConn.Credential}
	return &myImageHandler, nil
}

func (cloudConn *GCPCloudConnection) CreateClusterHandler() (irs.ClusterHandler, error) {
	return nil, errors.New("GCP Driver: not implemented")
}



func (cloudConn *GCPCloudConnection) CreateAnyCallHandler() (irs.AnyCallHandler, error) {
	return nil, errors.New("GCP Driver: not implemented")
}

