// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by zephy@mz.co.kr, 2019.09.

package main

import (
	"C"

	alibaba "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/alibaba"
)

var CloudDriver alibaba.AlibabaDriver
