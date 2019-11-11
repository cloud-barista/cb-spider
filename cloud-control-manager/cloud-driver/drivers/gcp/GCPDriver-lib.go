// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by jazmandorf@gmail.com MZC

package main

//package gcp

import (
	"C"

	"context"
	"encoding/json"
	"fmt"

	gcpcon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/gcp/connect"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"

	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"

	o2 "golang.org/x/oauth2"
	goo "golang.org/x/oauth2/google"

	compute "google.golang.org/api/compute/v1"
)

type GCPDriver struct {
}

func (GCPDriver) GetDriverVersion() string {
	return "GCP DRIVER Version 1.0"
}

func (GCPDriver) GetDriverCapability() idrv.DriverCapabilityInfo {
	var drvCapabilityInfo idrv.DriverCapabilityInfo

	drvCapabilityInfo.ImageHandler = true
	drvCapabilityInfo.VNetworkHandler = true
	drvCapabilityInfo.SecurityHandler = true
	drvCapabilityInfo.KeyPairHandler = true
	drvCapabilityInfo.VNicHandler = true
	drvCapabilityInfo.PublicIPHandler = true
	drvCapabilityInfo.VMHandler = true

	return drvCapabilityInfo
}

func (driver *GCPDriver) ConnectCloud(connectionInfo idrv.ConnectionInfo) (icon.CloudConnection, error) {
	// 1. get info of credential and region for Test A Cloud from connectionInfo.
	// 2. create a client object(or service  object) of Test A Cloud with credential info.
	// 3. create CloudConnection Instance of "connect/TDA_CloudConnection".
	// 4. return CloudConnection Interface of TDA_CloudConnection.

	Ctx, VMClient, err := getVMClient(connectionInfo.CredentialInfo)
	fmt.Println("################## getVMClient ##################")
	fmt.Println("getVMClient")
	fmt.Println("################## getVMClient ##################")
	if err != nil {
		return nil, err
	}

	iConn := gcpcon.GCPCloudConnection{
		Region:              connectionInfo.RegionInfo,
		Credential:          connectionInfo.CredentialInfo,
		Ctx:                 Ctx,
		VMClient:            VMClient,
		ImageClient:         VMClient,
		PublicIPClient:      VMClient,
		SecurityGroupClient: VMClient,
		VNetClient:          VMClient,
		VNicClient:          VMClient,
		SubnetClient:        VMClient,
	}

	fmt.Println("################## resource ConnectionInfo ##################")
	fmt.Println("iConn : ", iConn)
	fmt.Println("################## resource ConnectionInfo ##################")
	return &iConn, nil
}

func getVMClient(credential idrv.CredentialInfo) (context.Context, *compute.Service, error) {

	// GCP 는  ClientSecret에
	gcpType := "service_account"
	data := make(map[string]string)

	data["type"] = gcpType
	data["private_key"] = credential.PrivateKey
	data["client_email"] = credential.ClientEmail

	fmt.Println("################## data ##################")
	fmt.Println("data to json : ", data)
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

var CloudDriver GCPDriver
