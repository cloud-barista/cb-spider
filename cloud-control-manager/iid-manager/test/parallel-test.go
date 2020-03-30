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
	"sync"
	"time"
	"strconv"
	"runtime"

	"github.com/cloud-barista/cb-store/config"
	rsid "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"
)

var count = 100
var iidRWLock = new(iidm.IIDRWLOCK)
var waitGroup sync.WaitGroup

var cName = "aws-seoul-config"		// Connection Name
var rName = "VM"			// Resource Type


func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Printf("\n\n\n================================== CPU : %v \n", runtime.GOMAXPROCS(0))

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		for i:=0;i<10;i++ {
			list()
			time.Sleep(1 * time.Second)
		}
	}()

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		for i:=0; i<count; i++ {
			create(strconv.Itoa(i))
		}
	}()


	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		for i:=0; i<count; i++ {
			update(strconv.Itoa(i))
		}
	}()

	waitGroup.Add(1)
	go func() {
		defer waitGroup.Done()
		for i:=0; i<count; i++ {
			delete(strconv.Itoa(i))
		}
	}()


	waitGroup.Wait()
	fmt.Printf("\n\n\n================================== Press any key to clean!!!\n")
	fmt.Scanln()

	waitGroup.Add(1)
        go func() {
		defer waitGroup.Done()
                for i:=0; i<count; i++ {
                        delete(strconv.Itoa(i))
                }
        }()

	waitGroup.Wait()
	fmt.Printf("\n\n\n================================== Press any key to exit!!!\n")
	fmt.Scanln()

}

func create(id string) {
	nameID := "powerkim_vm_" + id	// NameID

	iId := rsid.IID{nameID, ""}
	iidInfo, err := iidRWLock.CreateIID(cName, rName, iId)
	if err != nil {
		config.Cblogger.Error(err)
	}
	config.Cblogger.Infof(" === %#v\n", iidInfo)
}

func update(id string) {
	nameID := "powerkim_vm_" + id	// NameID

        iId := rsid.IID{nameID, "i-0bc7123b7e5cbf79d"}
        iidInfo, err := iidRWLock.UpdateIID(cName, rName, iId)
        if err != nil {
                config.Cblogger.Error(err)
        }

        config.Cblogger.Infof(" === %#v\n", iidInfo)
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
			fmt.Printf("{%s, %s}\n", iidInfo.IId.NameId, iidInfo.IId.SystemId)
		}
        }
}

func delete(id string) {
	nameID := "powerkim_vm_" + id	// NameID
	fmt.Printf(" === DeleteIID %s\n", nameID)

	iId := rsid.IID{nameID, ""}
        result, err3 := iidRWLock.DeleteIID(cName, rName, iId)
        if err3 != nil {
                config.Cblogger.Error(err3)
        }

	//config.Cblogger.Infof(" === DeleteIID %s : %#v\n", nameID, result)
	fmt.Printf(" === DeleteIID %s : %#v\n", nameID, result)
}

