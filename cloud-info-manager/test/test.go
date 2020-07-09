// Test for Cloud Driver Handler of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2019.10.

package main

import (
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"

	"fmt"
)

func getCloudOSList() {
        cloudOSList := cim.ListCloudOS()
        fmt.Printf(" === %#v\n", cloudOSList)
}

func main() {
	getCloudOSList()
}

