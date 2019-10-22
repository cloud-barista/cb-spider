// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by hyokyung.kim@innogrid.co.kr, 2019.07.

package connect

import (
	"context"
	"fmt"

	idrv "../../../interfaces"
	irs "../../../interfaces/resources"
	gcprs "../../gcp/resources"
	compute "google.golang.org/api/compute/v1"
)

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
}

func (cloudConn *GCPCloudConnection) CreateVNetworkHandler() (irs.VNetworkHandler, error) {
	fmt.Println("GCP Cloud Driver: called CreateVNetworkHandler()!")
	vNetHandler := gcprs.GCPVNetworkHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.VNetClient, cloudConn.Credential}
	return &vNetHandler, nil
}

// func (cloudConn *GCPCloudConnection) CreateImageHandler() (irs.ImageHandler, error) {
// 	fmt.Println("GCP Cloud Driver: called CreateImageHandler()!")
// 	imageHandler := gcprs.GCPImageHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.ImageClient}
// 	return &imageHandler, nil
// }

func (cloudConn *GCPCloudConnection) CreateSecurityHandler() (irs.SecurityHandler, error) {
	fmt.Println("GCP Cloud Driver: called CreateSecurityHandler()!")
	sgHandler := gcprs.GCPSecurityHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.SecurityGroupClient, cloudConn.Credential}
	return &sgHandler, nil
}

// func (GCPCloudConnection) CreateKeyPairHandler() (irs.KeyPairHandler, error) {
// 	return nil, nil
// }

// func (cloudConn *GCPCloudConnection) CreateVNicHandler() (irs.VNicHandler, error) {
// 	fmt.Println("GCP Cloud Driver: called CreateVNicHandler()!")
// 	vNicHandler := gcprs.GCPVNicHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.VNicClient, cloudConn.SubnetClient}
// 	return &vNicHandler, nil
// }
func (cloudConn *GCPCloudConnection) CreatePublicIPHandler() (irs.PublicIPHandler, error) {
	fmt.Println("GCP Cloud Driver: called CreatePublicIPHandler()!")
	publicIPHandler := gcprs.GCPPublicIPHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.PublicIPClient, cloudConn.Credential}
	return &publicIPHandler, nil
}

func (cloudConn *GCPCloudConnection) CreateVMHandler() (irs.VMHandler, error) {
	fmt.Println("GCP Cloud Driver: called CreateVMHandler()!")
	vmHandler := gcprs.GCPVMHandler{cloudConn.Region, cloudConn.Ctx, cloudConn.VMClient, cloudConn.Credential}
	return &vmHandler, nil
}

func (GCPCloudConnection) IsConnected() (bool, error) {
	return true, nil
}
func (GCPCloudConnection) Close() error {
	return nil
}
