// Test for Cloud Driver Info. Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.03.

package main

import (
	"fmt"
	"time"

	"github.com/cloud-barista/cb-store/config"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
)

var count = 10
var iidRWLock = new(iidm.IIDRWLOCK)
var cName = "aws-seoul-config"		// Connection Name
var rName = "VM"			// Resource Type

func main() {

	for ;; {
		list()
		fmt.Printf("\n==================================\n")
		//fmt.Scanln()
		time.Sleep(2 * time.Second)
	}


}

func list() {
	keyValueList, err2 := iidRWLock.ListIID(cName, rName)
	if err2 != nil {
		config.Cblogger.Error(err2)
	}

	for _, keyValue := range keyValueList {
		iidInfo, err := iidRWLock.GetIID(cName, rName, keyValue.IId)
		if err != nil {
			config.Cblogger.Error(err)
		}else {
			//config.Cblogger.Infof(" === %#v\n", iidInfo)
			fmt.Printf("{%s, %s}\n", iidInfo.IId.NameId, iidInfo.IId.SystemId)
		}
        }
}
