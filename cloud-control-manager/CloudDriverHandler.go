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

        "github.com/sirupsen/logrus"
        "github.com/cloud-barista/cb-store/config"

        ccim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"
        dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"
        cim "github.com/cloud-barista/cb-spider/cloud-info-manager/credential-info-manager"
        rim "github.com/cloud-barista/cb-spider/cloud-info-manager/region-info-manager"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"

	"fmt"
	"os"
	"plugin"
	"net/http"
        "encoding/json"
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

// 1. get the driver info
// 2. load driver library
// 3. get CloudDriver
func GetCloudDriver(cloudConnectName string) (idrv.CloudDriver, error) {
	cccInfo, err:= getConnectionConfigInfo(cloudConnectName)
        if err != nil {
                return nil, err
        }

	cldDrvInfo, err:= getCloudDriverInfo(cccInfo.DriverName)
        if err != nil {
                return nil, err
        }

	// $CBSPIDER_ROOT/cloud-driver/libs/*
	driverLibPath := os.Getenv("CBSPIDER_ROOT") + "/cloud-driver/libs/"

	driverFile := cldDrvInfo.DriverLibFileName // ex) "aws-test-driver-v0.5.so"
        if driverFile == "" {
                return nil, fmt.Errorf("%q: driver library file can't nil or empty!!", cccInfo.DriverName )
        }
	driverPath := driverLibPath + driverFile

	cblog.Info(cccInfo.DriverName + ": driver path - " + driverPath)


        var plug *plugin.Plugin
        plug, err = plugin.Open(driverPath)

        // fmt.Printf("plug: %#v\n\n", plug)
        if err != nil {
                cblog.Errorf("plugin.Open: %v\n", err)
                return nil, err
        }


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

// 1. get credential info
// 2. get region info
// 3. get CloudConneciton
func GetCloudConnection(cloudConnectName string, cloudDriver *idrv.CloudDriver) (icon.CloudConnection, error) {
        cccInfo, err:= ccim.GetConnectionConfig(cloudConnectName)
        if err != nil {
                return nil, err
        }

	crdInfo, err:= cim.GetCredential(cccInfo.CredentialName)
        if err != nil {
                return nil, err
        }

	rgnInfo, err:= rim.GetRegion(cccInfo.RegionName)
        if err != nil {
                return nil, err
        }

	// @todo from now
	cblog.Info(crdInfo)
	cblog.Info(rgnInfo)

	return nil, nil
}

func getConnectionConfigInfo(configName string) (ccim.ConnectionConfigInfo, error) {
        // Build the request
        req, err := http.NewRequest("GET", CIM_RESTSERVER + "/connectionconfig/" + configName, nil)
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
        req, err := http.NewRequest("GET", CIM_RESTSERVER + "/driver/" + driverName, nil)
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
        req, err := http.NewRequest("GET", CIM_RESTSERVER + "/credential/" + credentialName, nil)
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
        req, err := http.NewRequest("GET", CIM_RESTSERVER + "/region/" + regionName, nil)
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
