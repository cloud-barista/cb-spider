// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2021.11.

package main

import (
        "github.com/cloud-barista/cb-store/config"
        "github.com/sirupsen/logrus"

        "github.com/cloud-barista/cb-spider/interface/api"
        rim "github.com/cloud-barista/cb-spider/cloud-info-manager/region-info-manager"

	"encoding/json"
	"os"
	"strings"
	"time"
	"bufio"
	"fmt"
)


var cblog *logrus.Logger

func init() {
        cblog = config.Cblogger
}

func main() {

	spiderServer := "localhost:2048"

	rootPath := os.Getenv("CBSPIDER_ROOT")
        if rootPath == "" {
                cblog.Error("$CBSPIDER_ROOT is not set!!")
                os.Exit(1)
        }

	fo, err := os.Open(rootPath + "/utils/import-drv-region-info/export-region-list/exported-regions-list.json")
	if err != nil {
		cblog.Error(err)
                os.Exit(1)
	}
	defer fo.Close()



	// 1. Create CloudInfoManager
        cim := api.NewCloudInfoManager()
        err = cim.SetServerAddr(spiderServer)
        if err != nil {
                cblog.Error(err)
        }

        // 2. Setup env.
        err = cim.SetTimeout(90 * time.Second)
        if err != nil {
                cblog.Error(err)
        }

        // 3. Open New Session
        err = cim.Open()
        if err != nil {
                cblog.Fatal(err)
        }
        // 4. Close (with defer)
        defer cim.Close()

	reader := bufio.NewReader(fo)
	for {
		regionName, isPrefix, err := reader.ReadLine()
		if isPrefix || err != nil {
			break
		}
		if insertConnection(cim, string(regionName)) != nil {
			cblog.Error(err)
			os.Exit(1)
		}
	}
}
 
// regionName format: 'aws:ap-east-1:ap-east-1a'
func insertConnection(cim *api.CIMApi, regionName string) error {
	cblog.Info("========== : ", regionName)
	strRegionInfo, err := cim.GetRegionByParam(regionName)
        if err != nil {
                cblog.Error(err)
		return err
        }

	var regInfo rim.RegionInfo
        json.Unmarshal([]byte(strRegionInfo), &regInfo)

	reqConnectionConfig := &api.ConnectionConfigReq{
		ConfigName:     "mini:imageinfo:" + regInfo.RegionName, // 'mini:imageinfo:aws:ap-east-1:ap-east-1a'
		ProviderName:   regInfo.ProviderName,
		DriverName:     strings.ToLower(regInfo.ProviderName) + "-driver01", // aws-driver01
		CredentialName: strings.ToLower(regInfo.ProviderName) + "-credential01", // aws-credential01
		RegionName:     regInfo.RegionName,
	}
	result, err := cim.CreateConnectionConfigByParam(reqConnectionConfig)
	if err != nil {
		cblog.Fatal(err)
	}

	fmt.Printf("\nresult :\n%s\n", result)

	return nil
}


