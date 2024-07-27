// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// NCPVPC Cloud Driver PoC
//
// by ETRI, 2020.12.
// by ETRI, 2022.03. updated
// by ETRI, 2024.07.

package main

import (
	"C"
	ncpvpc "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ncpvpc"
)

var CloudDriver ncpvpc.NcpVpcDriver
