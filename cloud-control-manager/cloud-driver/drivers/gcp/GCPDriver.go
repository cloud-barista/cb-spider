// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by jazmandorf@gmail.com MZC

package gcp

import (
	"context"
	"encoding/json"

	gcpcon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/gcp/connect"
	gcps "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/gcp/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"

	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"

	goo "golang.org/x/oauth2/google"

	cblogger "github.com/sirupsen/logrus"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/container/v1"
	"google.golang.org/api/option"
)

type GCPDriver struct {
}

func (GCPDriver) GetDriverVersion() string {
	return "GCP DRIVER Version 1.0"
}

func (GCPDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	drvCapabilityInfo.ImageHandler = true
	drvCapabilityInfo.VPCHandler = true
	drvCapabilityInfo.SecurityHandler = true
	drvCapabilityInfo.KeyPairHandler = true
	//drvCapabilityInfo.VNicHandler = true
	//drvCapabilityInfo.PublicIPHandler = true
	drvCapabilityInfo.VMHandler = true
	drvCapabilityInfo.VMSpecHandler = true
	drvCapabilityInfo.VPCHandler = true
	drvCapabilityInfo.DiskHandler = false
	drvCapabilityInfo.MyImageHandler = false
	drvCapabilityInfo.ClusterHandler = true

	return drvCapabilityInfo
}

func (driver *GCPDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	// Initialize Logger
	gcps.InitLog()

	Ctx, VMClient, err := getVMClient(connectionInfo.CredentialInfo)
	cblogger.Debug("################## getVMClient ##################")
	cblogger.Debug("getVMClient")
	cblogger.Debug("################## getVMClient ##################")
	if err != nil {
		return nil, err
	}
	//Ctx2, containerClient, err := getContainerClient(connectionInfo.CredentialInfo)
	_, containerClient, err := getContainerClient(connectionInfo.CredentialInfo)
	cblogger.Debug("################## getContainerClient ##################")
	cblogger.Debug("getContainerClient")
	cblogger.Debug("################## getContainerClient ##################")
	if err != nil {
		return nil, err
	}

	iConn := gcpcon.GCPCloudConnection{
		Region:      connectionInfo.RegionInfo,
		Credential:  connectionInfo.CredentialInfo,
		Ctx:         Ctx,
		VMClient:    VMClient,
		ImageClient: VMClient,
		// PublicIPClient:      VMClient,
		SecurityGroupClient: VMClient,
		// VNetClient:          VMClient,
		// VNicClient:          VMClient,
		SubnetClient:    VMClient,
		VMSpecHandler:   VMClient,
		VPCHandler:      VMClient,
		ContainerClient: containerClient,
	}

	//cblogger.Debug("################## resource ConnectionInfo ##################")
	//cblogger.Debug("iConn : ", iConn)
	//cblogger.Debug("################## resource ConnectionInfo ##################")
	return &iConn, nil
}

// authorization scopes : https://developers.google.com/identity/protocols/oauth2/scopes
// cloud-platform, cloud-platform.read-only, compute, compute.readonly

// auth scope : compute
// 아래에서 cloud-platform을 사용하는데 vmClient 대체가 되는지 확인 필요.
func getVMClient(credential idrv.CredentialInfo) (context.Context, *compute.Service, error) {

	// GCP 는  ClientSecret에
	gcpType := "service_account"
	data := make(map[string]string)

	data["type"] = gcpType
	data["private_key"] = credential.PrivateKey
	data["client_email"] = credential.ClientEmail

	cblogger.Debug("################## data ##################")
	//cblogger.Debug("data to json : ", data)
	cblogger.Debug("################## data ##################")

	res, _ := json.Marshal(data)
	// data, err := ioutil.ReadFile(credential.ClientSecret)
	authURL := "https://www.googleapis.com/auth/compute"

	conf, err := goo.JWTConfigFromJSON(res, authURL)

	if err != nil {

		return nil, nil, err
	}

	ctx := context.Background()

	client := conf.Client(ctx)

	vmClient, err := compute.NewService(ctx, option.WithHTTPClient(client))

	return ctx, vmClient, nil
}

// auth scope : cloud-platform
func getContainerClient(credential idrv.CredentialInfo) (context.Context, *container.Service, error) {

	// GCP 는  ClientSecret에
	gcpType := "service_account"
	data := make(map[string]string)

	data["type"] = gcpType
	data["private_key"] = credential.PrivateKey
	data["client_email"] = credential.ClientEmail

	cblogger.Debug("################## data ##################")
	//cblogger.Debug("data to json : ", data)
	cblogger.Debug("################## data ##################")

	res, _ := json.Marshal(data)
	authURL := "https://www.googleapis.com/auth/cloud-platform"

	conf, err := goo.JWTConfigFromJSON(res, authURL)

	if err != nil {

		return nil, nil, err
	}

	ctx := context.Background()

	client := conf.Client(ctx)

	containerClient, err := container.NewService(ctx, option.WithHTTPClient(client))

	return ctx, containerClient, nil
}
