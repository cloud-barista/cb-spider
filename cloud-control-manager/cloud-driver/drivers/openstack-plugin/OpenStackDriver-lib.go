// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by hyokyung.kim@innogrid.co.kr, 2019.07.

package main

import (
        "C"
        openstack "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/openstack"
)

var CloudDriver openstack.OpenStackDriver
