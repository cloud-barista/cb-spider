// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by ETRI, Innogrid, 2021.12.
// by ETRI Team, 2022.08.
// by ETRI Team, 2024.04.

package main

import (
	"C"

	nhn "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/nhn"
)

var CloudDriver nhn.NhnCloudDriver
