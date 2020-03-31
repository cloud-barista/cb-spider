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

	"github.com/cloud-barista/cb-store/config"
	rsid "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
)
func main() {

var iidRWLock = new(iidm.IIDRWLOCK)

	cName := "aws-seoul-config"	// Connection Name
	rName := "VM"			// Resource Type
	nameID1 := "powerkim_vm_01"	// NameID
	nameID2 := "powerkim_vm_02"	// NameID

	iId := rsid.IID{nameID1, ""}
fmt.Println("\n============== CreateIID(" + iId.NameId + ")\n")
	iidInfo, err := iidRWLock.CreateIID(cName, rName, iId)
	if err != nil {
		config.Cblogger.Error(err)
	}
	fmt.Printf(" === %#v\n", iidInfo)



        iId = rsid.IID{nameID2, ""}
fmt.Println("\n============== CreateIID(" + iId.NameId +")\n")
        iidInfo, err = iidRWLock.CreateIID(cName, rName, iId)
        if err != nil {
                config.Cblogger.Error(err)
        }
        fmt.Printf(" === %#v\n", iidInfo)



        iId = rsid.IID{nameID1, ""}
fmt.Println("\n============== CreateIID(" + iId.NameId +"): check the duplicated ID Creation\n")
        iidInfo, err = iidRWLock.CreateIID(cName, rName, iId)
        if err != nil {
                config.Cblogger.Error(err)
        }

        fmt.Printf(" === %#v\n", iidInfo)



        iId = rsid.IID{nameID1, "i-0bc7123b7e5cbf79d"}
fmt.Println("\n============== UpdateIID(" + iId.NameId +"): update test\n")
        iidInfo, err = iidRWLock.UpdateIID(cName, rName, iId)
        if err != nil {
                config.Cblogger.Error(err)
        }

        fmt.Printf(" === %#v\n", iidInfo)

	
fmt.Println("\n============== ListIID()")
	keyValueList, err2 := iidRWLock.ListIID(cName, rName)
	if err2 != nil {
		config.Cblogger.Error(err2)
	}

	for _, keyValue := range keyValueList {
                fmt.Printf(" === %#v\n", keyValue)
		iidRWLock.GetIID(cName, rName, keyValue.IId)
        }

fmt.Println("\n============== ================ ================ ================\n")

fmt.Println("\n============== DeleteIID()")
	iId = rsid.IID{nameID1, ""}
        result, err3 := iidRWLock.DeleteIID(cName, rName, iId)
        if err3 != nil {
                config.Cblogger.Error(err3)
        }

	fmt.Printf(" === DeleteIID %s : %#v\n", iId.NameId, result)

fmt.Println("\n============== ListIID()")
        keyValueList, err2 = iidRWLock.ListIID(cName, rName)
        if err2 != nil {
                config.Cblogger.Error(err2)
        }

        for _, keyValue := range keyValueList {
                fmt.Printf(" === %#v\n", keyValue)
                iidRWLock.GetIID(cName, rName, keyValue.IId)
        }

fmt.Println("\n============== ================ ================ ================\n")

fmt.Println("\n============== DeleteIID()")
	iId = rsid.IID{nameID2, ""}
        result, err3 = iidRWLock.DeleteIID(cName, rName, iId)
        if err3 != nil {
                config.Cblogger.Error(err3)
        }

        fmt.Printf(" === DeleteIID %s : %#v\n", iId.NameId, result)

fmt.Println("\n============== ListIID()")
        keyValueList, err2 = iidRWLock.ListIID(cName, rName)
        if err2 != nil {
                config.Cblogger.Error(err2)
        }

        for _, keyValue := range keyValueList {
                fmt.Printf(" === %#v\n", keyValue)
                iidRWLock.GetIID(cName, rName, keyValue.IId)
        }

fmt.Println("\n============== ================ ================ ================\n")

}
