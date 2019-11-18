// Cloud Driver Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by powerkim@etri.re.kr, 2019.09.

package clouddriverhandler

import (
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icbs "github.com/cloud-barista/cb-store/interfaces"

	"github.com/cloud-barista/cb-store/config"
	"github.com/sirupsen/logrus"

	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	ccim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/credential-info-manager"
	dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"
	rim "github.com/cloud-barista/cb-spider/cloud-info-manager/region-info-manager"

	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"plugin"
	"strings"
)

var CIM_RESTSERVER = "http://localhost:1024"
var cblog *logrus.Logger

func init() {
	//CIM_RESTSERVER = "http://localhost:1024"
	cblog = config.Cblogger
}

/*
func ListCloudDriver() []string {
	var cloudDriverList []string
	// @todo get list from storage
	return cloudDriverList
}
*/

// 1. get the ConnectionConfig Info
// 2. get the driver info
// 3. load driver library
// 4. get CloudDriver
func GetCloudDriver(cloudConnectName string) (idrv.CloudDriver, error) {
	cccInfo, err := getConnectionConfigInfo(cloudConnectName)
	if err != nil {
		return nil, err
	}

	cldDrvInfo, err := getCloudDriverInfo(cccInfo.DriverName)
	if err != nil {
		return nil, err
	}

	return getCloudDriver(cldDrvInfo)
}

// 1. get credential info
// 2. get region info
// 3. get CloudConneciton
func GetCloudConnection(cloudConnectName string) (icon.CloudConnection, error) {
	cccInfo, err := getConnectionConfigInfo(cloudConnectName)
	if err != nil {
		return nil, err
	}

	cldDrvInfo, err := getCloudDriverInfo(cccInfo.DriverName)
	if err != nil {
		return nil, err
	}

	cldDriver, err := getCloudDriver(cldDrvInfo)
	if err != nil {
		return nil, err
	}

	crdInfo, err := getCredentialInfo(cccInfo.CredentialName)
	if err != nil {
		return nil, err
	}

	rgnInfo, err := getRegionInfo(cccInfo.RegionName)
	if err != nil {
		return nil, err
	}

	//cblog.Info(cldDriver)
	//cblog.Info(crdInfo)
	//cblog.Info(rgnInfo)

	// @todo should move KeyValueList into XXXDriver.go, powerkim
	var regionName string
	var zoneName string
	switch strings.ToUpper(rgnInfo.ProviderName) {
	case "AZURE":
		regionName = getValue(rgnInfo.KeyValueInfoList, "location")
	case "AWS":
		regionName = getValue(rgnInfo.KeyValueInfoList, "Region")
	case "GCP":
		regionName = getValue(rgnInfo.KeyValueInfoList, "Region")
		zoneName = getValue(rgnInfo.KeyValueInfoList, "Zone")
	case "OPENSTACK":
	case "CLOUDTWIN":
		regionName = getValue(rgnInfo.KeyValueInfoList, "Region")
	case "CLOUDIT":
		// Cloudit do not use Region, But set default @todo 2019.10.28 by powerkim.
		regionName = getValue(rgnInfo.KeyValueInfoList, "Region")
		//regionName = getValue(rgnInfo.KeyValueInfoList, "Region")
	default:
		errmsg := rgnInfo.ProviderName + " is not a valid ProviderName!!"
		return nil, fmt.Errorf(errmsg)
	}

	connectionInfo := idrv.ConnectionInfo{ // @todo powerkim
		CredentialInfo: idrv.CredentialInfo{
			ClientId:         getValue(crdInfo.KeyValueInfoList, "ClientId"),
			ClientSecret:     getValue(crdInfo.KeyValueInfoList, "ClientSecret"),
			TenantId:         getValue(crdInfo.KeyValueInfoList, "TenantId"),
			SubscriptionId:   getValue(crdInfo.KeyValueInfoList, "SubscriptionId"),
			IdentityEndpoint: getValue(crdInfo.KeyValueInfoList, "IdentityEndpoint"),
			Username:         getValue(crdInfo.KeyValueInfoList, "Username"),
			Password:         getValue(crdInfo.KeyValueInfoList, "Password"),
			DomainName:       getValue(crdInfo.KeyValueInfoList, "DomainName"),
			ProjectID:        getValue(crdInfo.KeyValueInfoList, "ProjectID"),
			AuthToken:        getValue(crdInfo.KeyValueInfoList, "AuthToken"),
			ClientEmail:      getValue(crdInfo.KeyValueInfoList, "ClientEmail"),
			PrivateKey:       getValue(crdInfo.KeyValueInfoList, "PrivateKey"),
		},
		RegionInfo: idrv.RegionInfo{ // @todo powerkim
			Region:        regionName,
			Zone:          zoneName,
			ResourceGroup: getValue(rgnInfo.KeyValueInfoList, "ResourceGroup"),
		},
	}

	cldConnection, err := cldDriver.ConnectCloud(connectionInfo)
	if err != nil {
		return nil, err
	}

	return cldConnection, nil
}

func getValue(keyValueInfoList []icbs.KeyValue, key string) string {
	for _, kv := range keyValueInfoList {
		if kv.Key == key {
			return kv.Value
		}
	}
	return "Not set"
}

func getCloudDriver(cldDrvInfo dim.CloudDriverInfo) (idrv.CloudDriver, error) {
	// $CBSPIDER_ROOT/cloud-driver-libs/*
	driverLibPath := os.Getenv("CBSPIDER_ROOT") + "/cloud-driver-libs/"

	driverFile := cldDrvInfo.DriverLibFileName // ex) "aws-test-driver-v0.5.so"
	if driverFile == "" {
		return nil, fmt.Errorf("%q: driver library file can't nil or empty!!", cldDrvInfo.DriverName)
	}
	driverPath := driverLibPath + driverFile

	cblog.Info(cldDrvInfo.DriverName + ": driver path - " + driverPath)

	/*---------------
	        A plugin is only initialized once, and cannot be closed.
	        ref) https://golang.org/pkg/plugin/
	-----------------*/

	//var plug *plugin.Plugin
	plug, err := plugin.Open(driverPath)
	if err != nil {
		cblog.Errorf("plugin.Open: %v\n", err)
		return nil, err
	}
	//      fmt.Printf("plug: %#v\n\n", plug)

	//driver, err := plug.Lookup(cccInfo.DriverName)
	driver, err := plug.Lookup("CloudDriver")
	if err != nil {
		cblog.Errorf("plug.Lookup: %v\n", err)
		return nil, err
	}

	cloudDriver, ok := driver.(idrv.CloudDriver)
	if !ok {
		cblog.Error("Not CloudDriver interface!!")
		return nil, err
	}

	return cloudDriver, nil
}

func getConnectionConfigInfo(configName string) (ccim.ConnectionConfigInfo, error) {
	// Build the request
	req, err := http.NewRequest("GET", CIM_RESTSERVER+"/connectionconfig/"+configName, nil)
	if err != nil {
		cblog.Errorf("Error is req: ", err)
	}

	// create a Client
	client := &http.Client{}

	// Do sends an HTTP request and
	resp, err := client.Do(req)
	if err != nil {
		cblog.Errorf("error in send req: ", err)
	}

	// Defer the closing of the body
	defer resp.Body.Close()

	// Fill the data with the data from the JSON
	var data ccim.ConnectionConfigInfo

	// Use json.Decode for reading streams of JSON data
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		cblog.Error(err)
	}

	return data, nil
}

func getCloudDriverInfo(driverName string) (dim.CloudDriverInfo, error) {

	// Build the request
	req, err := http.NewRequest("GET", CIM_RESTSERVER+"/driver/"+driverName, nil)
	if err != nil {
		cblog.Errorf("Error is req: ", err)
	}

	// create a Client
	client := &http.Client{}

	// Do sends an HTTP request and
	resp, err := client.Do(req)
	if err != nil {
		cblog.Errorf("error in send req: ", err)
	}

	// Defer the closing of the body
	defer resp.Body.Close()

	// Fill the data with the data from the JSON
	var data dim.CloudDriverInfo

	// Use json.Decode for reading streams of JSON data
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		cblog.Error(err)
	}

	return data, nil
}

func getCredentialInfo(credentialName string) (cim.CredentialInfo, error) {

	// Build the request
	req, err := http.NewRequest("GET", CIM_RESTSERVER+"/credential/"+credentialName, nil)
	if err != nil {
		cblog.Errorf("Error is req: ", err)
	}

	// create a Client
	client := &http.Client{}

	// Do sends an HTTP request and
	resp, err := client.Do(req)
	if err != nil {
		cblog.Errorf("error in send req: ", err)
	}

	// Defer the closing of the body
	defer resp.Body.Close()

	// Fill the data with the data from the JSON
	var data cim.CredentialInfo

	// Use json.Decode for reading streams of JSON data
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		cblog.Error(err)
	}

	return data, nil
}

func getRegionInfo(regionName string) (rim.RegionInfo, error) {

	// Build the request
	req, err := http.NewRequest("GET", CIM_RESTSERVER+"/region/"+regionName, nil)
	if err != nil {
		cblog.Errorf("Error is req: ", err)
	}

	// create a Client
	client := &http.Client{}

	// Do sends an HTTP request and
	resp, err := client.Do(req)
	if err != nil {
		cblog.Errorf("error in send req: ", err)
	}

	// Defer the closing of the body
	defer resp.Body.Close()

	// Fill the data with the data from the JSON
	var data rim.RegionInfo

	// Use json.Decode for reading streams of JSON data
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		cblog.Error(err)
	}

	return data, nil
}
