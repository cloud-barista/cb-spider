// Test for loading Yaml
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2021.11.

package yamlloadtest

import (
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager"

	"testing"
	"fmt"
	"time"
)

func TestCloudOSList(t *testing.T) {
        cloudOSList := cim.ListCloudOS()
        fmt.Printf(" === %#v\n", cloudOSList)
}

func TestCloudOSMetaInfo(t *testing.T) {
	for ;; {
		cloudOSMetaInfo, err := cim.GetCloudOSMetaInfo("GCP")
		if err != nil {
			fmt.Println(err)
		}
		fmt.Printf(" === %#v\n", cloudOSMetaInfo)
		time.Sleep(1*time.Second)
	}
}

