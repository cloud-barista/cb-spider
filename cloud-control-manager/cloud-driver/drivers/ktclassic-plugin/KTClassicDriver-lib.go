// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// KT Cloud Driver PoC
//
// by ETRI, 2021.05.
// Updated by ETRI, 2023.10.
// Updated by ETRI, 2024.08.

package main

import (
	"C"
	kt "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/ktclassic"
)

var CloudDriver kt.KtCloudDriver
