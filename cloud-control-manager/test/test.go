// Test for Cloud Driver Handler of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2019.10.

package main

import (
	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"

	"fmt"
)

func getDriver() {

	fmt.Println("\n============== GetCloudDriver()")

	cloudConnectConfigName := "azure-config01"

	cldDrv, err := ccm.GetCloudDriver(cloudConnectConfigName)
	if err != nil {
                panic(err)
	}

	fmt.Printf(" === %#v\n", cldDrv)


        fmt.Println("\n============== GetCloudConnection()")

        cldConn, err := ccm.GetCloudConnection(cloudConnectConfigName)
        if err != nil {
                panic(err)
        }

        fmt.Printf(" === %#v\n", cldConn)
}


func main() {
	getDriver()
}

