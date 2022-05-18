// +build dyna

// Cloud Driver Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.12.

package clouddriverhandler

import (
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	// icbs "github.com/cloud-barista/cb-store/interfaces"

	"github.com/cloud-barista/cb-store/config"
	"github.com/sirupsen/logrus"

	dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"

	//"encoding/json"
	"fmt"
	//"net/http"
	"os"
	"plugin"
	"sync"
)

var cblog *logrus.Logger

func init() {
	cblog = config.Cblogger
}

// definition of RWLock to avoid 'plugin already loaded' panic
var pluginRWLock = new(sync.RWMutex)

func getCloudDriver(cldDrvInfo dim.CloudDriverInfo) (idrv.CloudDriver, error) {
	// $CBSPIDER_ROOT/cloud-driver-libs/*
	cbspiderRoot := os.Getenv("CBSPIDER_ROOT")
	if cbspiderRoot == "" {
		cblog.Error("$CBSPIDER_ROOT is not set!!")
		os.Exit(1)
	}
	driverLibPath := cbspiderRoot + "/cloud-driver-libs/"

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
// RWLock to avoid 'plugin already loaded' panic
pluginRWLock.Lock()
	//var plug *plugin.Plugin
	plug, err := plugin.Open(driverPath)
	if err != nil {
   pluginRWLock.Unlock()
		cblog.Errorf("plugin.Open: %v\n", err)
		return nil, err
	}
pluginRWLock.Unlock()

	//      fmt.Printf("plug: %#v\n\n", plug)

	//driver, err := plug.Lookup(cccInfo.DriverName)
	driver, err := plug.Lookup("CloudDriver")
	if err != nil {
		cblog.Errorf("plug.Lookup: %v\n", err)
		return nil, err
	}

	cloudDriver, ok := driver.(idrv.CloudDriver)
	if !ok {
		cblog.Error(ok)
		cblog.Error("Not CloudDriver interface!!")
		return nil, err
	}

	return cloudDriver, nil
}
