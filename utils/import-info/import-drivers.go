// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.05.

package main

import (
        "github.com/sirupsen/logrus"
        "github.com/cloud-barista/cb-store/config"

	dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"
	"strings"
	"os"
	"path/filepath"
)


var cblog *logrus.Logger

func init() {
        cblog = config.Cblogger
}



func main() {
	InsertDriverInfos()
}

// (1) get driver-lib file list
// (2) loop: 
// 		load DriverInfo List from all driver-lib file list
// (3) insert
func InsertDriverInfos() {

	var files []string

        cbspiderRoot := os.Getenv("CBSPIDER_ROOT")
        if cbspiderRoot == "" {
                Cblogger.Error("$CBSPIDER_ROOT is not set!!")
                os.Exit(1)
        }
	drvLibPath := cbspiderRoot + "/cloud-driver-libs/"
	err := filepath.Walk(drvLibPath, func(path string, info os.FileInfo, err error) error {
		files = append(files, info.Name())
		return nil
	})
	if err != nil {
		panic(err)
	}

	for _, file := range files {
		if strings.Contains(file, ".so") {
			// docker-driver-v1.0.so
			driverName := strings.ReplaceAll(file, ".so", "")
			strs := strings.Split(file, "-")
			cloudos := strings.ToUpper(strs[0])
			cloudDriverInfo := dim.CloudDriverInfo{driverName, cloudos, file}

			_, err := dim.RegisterCloudDriverInfo(cloudDriverInfo)
	                if err != nil {
				cblog.Error(err)
			}
		}
	}

}

