// Test for Cloud Driver Info. Manager of CB-Spider.
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
	dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"
)
func main() {

fmt.Println("\n============== RegisterCredential()")
	dName := "aws_driver01-V0.5"
	pName := "AWS"
	dLibFileName := "aws-test-driver-v0.5.so"
	drvInfo, err := dim.RegisterCloudDriver(dName, pName, dLibFileName)
	if err != nil {
		config.Cblogger.Error(err)
	}

	fmt.Printf(" === %#v\n", drvInfo)

fmt.Println("\n============== RegisterCredential()")
        dName = "aws_driver02-V1.0"
        pName = "AWS"
        dLibFileName = "aws-test-driver-v1.0.so"
        drvInfo, err = dim.RegisterCloudDriver(dName, pName, dLibFileName)
        if err != nil {
                config.Cblogger.Error(err)
        }

	fmt.Printf(" === %#v\n", drvInfo)
	
fmt.Println("\n============== ListCloudDriver()")
	keyValueList, err2 := dim.ListCloudDriver()
	if err2 != nil {
		config.Cblogger.Error(err2)
	}

	for _, keyValue := range keyValueList {
                fmt.Printf(" === %#v\n", keyValue)
		dim.GetCloudDriver(keyValue.DriverName)
        }

fmt.Println("\n============== UnRegisterCloudDriver()")
        result, err3 := dim.UnRegisterCloudDriver(dName)
        if err3 != nil {
                config.Cblogger.Error(err3)
        }

	fmt.Printf(" === UnRegisterCloudDriver %s : %#v\n", dName, result)

fmt.Println("\n============== ListCloudDriver()")
        keyValueList, err2 = dim.ListCloudDriver()
        if err2 != nil {
                config.Cblogger.Error(err2)
        }

        for _, keyValue := range keyValueList {
                fmt.Printf(" === %#v\n", keyValue)
        }

fmt.Println("\n============== UnRegisterCloudDriver()")
	dName = "aws_driver01-V0.5"
        result, err3 = dim.UnRegisterCloudDriver(dName)
        if err3 != nil {
                config.Cblogger.Error(err3)
        }

        fmt.Printf(" === UnRegisterCloudDriver %s : %#v\n", dName, result)

fmt.Println("\n============== ListCloudDriver()")
        keyValueList, err2 = dim.ListCloudDriver()
        if err2 != nil {
                config.Cblogger.Error(err2)
        }

        for _, keyValue := range keyValueList {
                fmt.Printf(" === %#v\n", keyValue)
        }

}
