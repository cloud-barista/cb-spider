// Test for Cloud ConnectionConfig Info. Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by powerkim@etri.re.kr, 2019.09.

package main

import (
	"fmt"

	"github.com/cloud-barista/cb-store/config"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"
)
func main() {

fmt.Println("\n============== CreateConnectionConfig()")
	cName := "config01"
	pName := "AWS"
	dName := "AWS-Test-Driver-V0.5"
	cdName := "credential01"
	rName := "region01"
	cncInfo, err := cim.CreateConnectionConfig(cName, pName, dName, cdName, rName)
	if err != nil {
		config.Cblogger.Error(err)
	}

	fmt.Printf(" === %#v\n", cncInfo)

fmt.Println("\n============== CreateConnectionConfig()")
	cName = "config02"
	pName = "AWS"
	dName = "AWS-Test-Driver-V1.0"
	cdName = "credential01"
	rName = "region01"
	cncInfo, err = cim.CreateConnectionConfig(cName, pName, dName, cdName, rName)
        if err != nil {
                config.Cblogger.Error(err)
        }

	fmt.Printf(" === %#v\n", cncInfo)
	
fmt.Println("\n============== ListConnectionConfig()")
	keyValueList, err2 := cim.ListConnectionConfig()
	if err2 != nil {
		config.Cblogger.Error(err2)
	}

	for _, keyValue := range keyValueList {
                fmt.Printf(" === %#v\n", keyValue)
		cim.GetConnectionConfig(keyValue.ConfigName)
        }

fmt.Println("\n============== DeleteConnectionConfig()")
        result, err3 := cim.DeleteConnectionConfig(cName)
        if err3 != nil {
                config.Cblogger.Error(err3)
        }

	fmt.Printf(" === DeleteConnectionConfig %s : %#v\n", cName, result)

fmt.Println("\n============== ListConnectionConfig()")
        keyValueList, err2 = cim.ListConnectionConfig()
        if err2 != nil {
                config.Cblogger.Error(err2)
        }

        for _, keyValue := range keyValueList {
                fmt.Printf(" === %#v\n", keyValue)
        }

fmt.Println("\n============== DeleteConnectionConfig()")
	cName = "config01"
        result, err3 = cim.DeleteConnectionConfig(cName)
        if err3 != nil {
                config.Cblogger.Error(err3)
        }

        fmt.Printf(" === DeleteConnectionConfig %s : %#v\n", cName, result)

fmt.Println("\n============== ListConnectionConfig()")
        keyValueList, err2 = cim.ListConnectionConfig()
        if err2 != nil {
                config.Cblogger.Error(err2)
        }

        for _, keyValue := range keyValueList {
                fmt.Printf(" === %#v\n", keyValue)
        }

}
