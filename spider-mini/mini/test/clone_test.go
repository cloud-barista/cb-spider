// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2021.11.

package clonetest

import (
        "testing"
	"fmt"
        _ "log"

	mini "github.com/cloud-barista/cb-spider/spider-mini/mini"
)

func TestClone(t *testing.T) {

	cloneName := "test-devstack-openstack-imageinfo"
	connectName := "imageinfo:aws:seoul"
	//connectName := "imageinfo:openstack:devstack"

        mini.Add(cloneName, connectName, mini.IMAGEINFO)

	mini.Cloner()

        //mini.Del(cloneName)

	fmt.Println("============> : ", mini.Count())
}


