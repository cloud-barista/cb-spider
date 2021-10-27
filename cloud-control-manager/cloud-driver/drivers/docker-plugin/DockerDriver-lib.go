// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Docker Driver.
//
// by CB-Spider Team, 2020.05.

package main

import (
        "C"
        docker "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/docker"
)

var CloudDriver docker.DockerDriver
