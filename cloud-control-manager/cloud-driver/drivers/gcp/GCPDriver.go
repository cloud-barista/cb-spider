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
	"fmt"

	gcpcon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/gcp/connect"
	gcps "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/gcp/resources"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"

	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"

	o2 "golang.org/x/oauth2"
	goo "golang.org/x/oauth2/google"

	"google.golang.org/api/cloudbilling/v1"
	cbb "google.golang.org/api/cloudbilling/v1beta"
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
	drvCapabilityInfo.RegionZoneHandler = true
	drvCapabilityInfo.ClusterHandler = true
	drvCapabilityInfo.PriceInfoHandler = true

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
	fmt.Println("################## getVMClient ##################")
	fmt.Println("getVMClient")
	fmt.Println("################## getVMClient ##################")
	if err != nil {
		return nil, err
	}
	//Ctx2, containerClient, err := getContainerClient(connectionInfo.CredentialInfo)
	_, containerClient, err := getContainerClient(connectionInfo.CredentialInfo)
	fmt.Println("################## getContainerClient ##################")
	fmt.Println("getContainerClient")
	fmt.Println("################## getContainerClient ##################")
	if err != nil {
		return nil, err
	}

	_, billingCatalogClient, err := getBillingCatalogClient(connectionInfo.CredentialInfo)
	fmt.Println("################## getBillingCatalogClient ##################")
	fmt.Println("getBillingCatalogClient")
	fmt.Println("################## getBillingCatalogClient ##################")
	if err != nil {
		return nil, err
	}

	_, costEstimationClient, err := getCostEstimationClient(connectionInfo.CredentialInfo)
	fmt.Println("################## getCostEstimationClient ##################")
	fmt.Println("getCostEstimationClient")
	fmt.Println("################## getCostEstimationClient ##################")
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
		SubnetClient:         VMClient,
		VMSpecClient:         VMClient,
		VPCClient:            VMClient,
		RegionZoneClient:     VMClient,
		ContainerClient:      containerClient,
		BillingCatalogClient: billingCatalogClient,
		CostEstimationClient: costEstimationClient,
	}

	//fmt.Println("################## resource ConnectionInfo ##################")
	//fmt.Println("iConn : ", iConn)
	//fmt.Println("################## resource ConnectionInfo ##################")
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

	fmt.Println("################## data ##################")
	//fmt.Println("data to json : ", data)
	fmt.Println("################## data ##################")

	res, _ := json.Marshal(data)
	// data, err := ioutil.ReadFile(credential.ClientSecret)
	authURL := "https://www.googleapis.com/auth/compute"

	conf, err := goo.JWTConfigFromJSON(res, authURL)

	if err != nil {

		return nil, nil, err
	}

	client := conf.Client(o2.NoContext)

	vmClient, err := compute.New(client)

	ctx := context.Background()

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

	fmt.Println("################## data ##################")
	//fmt.Println("data to json : ", data)
	fmt.Println("################## data ##################")

	res, _ := json.Marshal(data)
	authURL := "https://www.googleapis.com/auth/cloud-platform"

	conf, err := goo.JWTConfigFromJSON(res, authURL)

	if err != nil {

		return nil, nil, err
	}

	client := conf.Client(o2.NoContext)

	containerClient, err := container.New(client)

	ctx := context.Background()

	return ctx, containerClient, nil
}

func getBillingCatalogClient(credential idrv.CredentialInfo) (context.Context, *cloudbilling.APIService, error) {

	// GCP 는  ClientSecret에
	gcpType := "service_account"
	data := make(map[string]string)

	data["type"] = gcpType
	data["private_key"] = credential.PrivateKey
	data["client_email"] = credential.ClientEmail

	fmt.Println("################## data ##################")
	//fmt.Println("data to json : ", data)
	fmt.Println("################## data ##################")
	// https://www.googleapis.com/auth/cloud-platform

	// https://www.googleapis.com/auth/cloud-billing
	res, _ := json.Marshal(data)
	//authURL := "https://www.googleapis.com/auth/cloud-platform"
	authURL := "https://www.googleapis.com/auth/cloud-billing"

	conf, err := goo.JWTConfigFromJSON(res, authURL)

	if err != nil {
		fmt.Println("JWTConfig ", conf)
		return nil, nil, err
	}

	client := conf.Client(o2.NoContext)

	billingCatalogClient, err := cloudbilling.New(client)
	if err != nil {
		fmt.Println("billingCatalogClient err ", err)
		return nil, nil, err
	}

	ctx := context.Background()

	return ctx, billingCatalogClient, nil
}

func getCostEstimationClient(credential idrv.CredentialInfo) (context.Context, *cbb.Service, error) {

	// GCP 는  ClientSecret에
	gcpType := "service_account"
	data := make(map[string]string)

	data["type"] = gcpType
	data["private_key"] = credential.PrivateKey
	data["client_email"] = credential.ClientEmail

	fmt.Println("################## data ##################")
	//fmt.Println("data to json : ", data)
	fmt.Println("################## data ##################")
	// https://www.googleapis.com/auth/cloud-platform

	// https://www.googleapis.com/auth/cloud-billing
	res, _ := json.Marshal(data)
	//authURL := "https://www.googleapis.com/auth/cloud-platform"
	authURL := "https://www.googleapis.com/auth/cloud-billing"

	conf, err := goo.JWTConfigFromJSON(res, authURL)

	if err != nil {
		fmt.Println("JWTConfig ", conf)
		return nil, nil, err
	}

	client := conf.Client(o2.NoContext)

	ctx := context.Background()

	costEstimationClient, err := cbb.NewService(ctx, option.WithHTTPClient(client))

	if err != nil {
		fmt.Println("costEstimation Service create err ", err)
		return nil, nil, err
	}

	return ctx, costEstimationClient, nil
}
