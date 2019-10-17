// Test for Cloud Driver Handler of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by powerkim@etri.re.kr, 2019.10.

package main

import (
	"github.com/sirupsen/logrus"
	"github.com/cloud-barista/cb-store/config"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"

	"fmt"
)


var cblog *logrus.Logger

func init() {
        cblog = config.Cblogger
}


func main() {


	fmt.Println("\n============== GetCloudDriver()")

	cloudConnectConfigName := "azure-config01"

	cldDrv, err := ccm.GetCloudDriver(cloudConnectConfigName)
	if err != nil {
		cblog.Error(err)
	}

	fmt.Printf(" === %#v\n", cldDrv)


        fmt.Println("\n============== GetCloudConnection()")

        cldConn, err := ccm.GetCloudConnection(cloudConnectConfigName)
        if err != nil {
                cblog.Error(err)
        }

        fmt.Printf(" === %#v\n", cldConn)

}

