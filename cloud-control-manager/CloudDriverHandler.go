// Cloud Driver Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by powerkim@etri.re.kr, 2019.09.


package driverhandler

import (
	idrv "github.com/cloud-barista/poc-cb-spider/cloud-driver/interfaces"

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
)

var cblog *logrus.Logger

func init() {
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
	cccInfo, err:= ccim.GetConnectionConfig(cloudConnectName)
        if err != nil {
                return nil, err
        }

	cldDrvInfo, err:= dim.GetCloudDriver(cccInfo.DriverName)
        if err != nil {
                return nil, err
        }

	// $CB_SPIDER_ROOT/cloud-driver/libs/*
	driverLibPath := os.Getenv("CB_SPIDER_ROOT") + "/cloud-driver/libs/"

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


        driver, err := plug.Lookup(cccInfo.DriverName)
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

