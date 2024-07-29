// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI, 2020.08.
// by ETRI, 2024.07.

package main

import (
	"C"
	ncp "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncp"
)

var CloudDriver ncp.NcpDriver
