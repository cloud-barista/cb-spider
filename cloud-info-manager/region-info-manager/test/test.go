// Test for Cloud Region Info. Manager of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2019.09.

package main

import (
	"fmt"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/region-info-manager"
	"github.com/sirupsen/logrus"
)

func main() {

	// ex-1)
	// /cloud-info-spaces/regions/<aws_region01>/{aws}/{region} [ap-northeast-2]
	// ex-2)
	// /cloud-info-spaces/regions/<gcp_region02>/{gcp}/{region} [us-east1]
	// /cloud-info-spaces/regions/<gcp_region02>/{gcp}/{zone}} [us-east1-c]

	var Cblogger *logrus.Logger

	fmt.Println("\n============== RegisterRegion()")
	cName := "aws_region01"
	pName := "AWS"
	keyValueList := []idrv.KeyValue{{"region", "ap-northeast-2"}}

	crdInfo, err := cim.RegisterRegion(cName, pName, keyValueList)
	if err != nil {
		Cblogger.Error(err)
	}

	fmt.Printf(" === %#v\n", crdInfo)

	fmt.Println("\n============== RegisterRegion()")
	cName = "gcp_region02"
	pName = "GCP"
	keyValueList = []idrv.KeyValue{{"region", "us-east1"},
		{"zone", "us-east1-c"},
	}

	crdInfo, err = cim.RegisterRegion(cName, pName, keyValueList)
	if err != nil {
		Cblogger.Error(err)
	}

	fmt.Printf(" === %#v\n", crdInfo)

	fmt.Println("\n============== ListRegion()")
	regionInfoList, err2 := cim.ListRegion()
	if err2 != nil {
		Cblogger.Error(err2)
	}

	for _, keyValue := range regionInfoList {
		fmt.Printf(" === %#v\n", keyValue)
		cim.GetRegion(keyValue.RegionName)
	}

	fmt.Println("\n============== UnRegisterRegion()")
	cName = "aws_region01"
	result, err3 := cim.UnRegisterRegion(cName)
	if err3 != nil {
		Cblogger.Error(err3)
	}

	fmt.Printf(" === cim.UnRegisterRegion %s : %#v\n", cName, result)

	fmt.Println("\n============== ListRegion()")
	regionInfoList, err2 = cim.ListRegion()
	if err2 != nil {
		Cblogger.Error(err2)
	}

	for _, keyValue := range regionInfoList {
		fmt.Printf(" === %#v\n", keyValue)
	}

	fmt.Println("\n============== UnRegisterRegion()")
	cName = "gcp_region02"
	result, err3 = cim.UnRegisterRegion(cName)
	if err3 != nil {
		Cblogger.Error(err3)
	}

	fmt.Printf(" === cim.UnRegisterRegion %s : %#v\n", cName, result)

	fmt.Println("\n============== ListRegion()")
	regionInfoList, err2 = cim.ListRegion()
	if err2 != nil {
		Cblogger.Error(err2)
	}

	for _, keyValue := range regionInfoList {
		fmt.Printf(" === %#v\n", keyValue)
	}

}
