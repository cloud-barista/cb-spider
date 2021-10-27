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

import (
        "C"
        gcp "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/gcp"
)

var CloudDriver gcp.GCPDriver
