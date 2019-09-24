// Cloud Info Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by powerkim@etri.re.kr, 2019.09.

//package cimrestruntime
package main

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/cloud-barista/cb-store/config"

	dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"

        // REST API (echo)
        "net/http"
        "github.com/labstack/echo"
        "github.com/labstack/echo/middleware"
)

var cblog *logrus.Logger

func init() {
        cblog = config.Cblogger
}

//====================================================================
type CloudDriverInfo struct {
	DriverName	string	// ex) "AWS-Test-Driver-V0.5"
	ProviderName	string	// ex) "AWS"
	DriverLibFileName	string	// ex) "aws-test-driver-v0.5.so"  //Already, you need to insert "*.so" in $CB_SPIDER_ROOT/cloud-driver/libs.
}
//====================================================================


func apiServer() {

        e := echo.New()

        // Middleware
        e.Use(middleware.Logger())
        e.Use(middleware.Recover())

        e.GET("/", func(c echo.Context) error {
                return c.String(http.StatusOK, "CB-Spider!")
        })

        // Route
        e.GET("/driver", listCloudDriver)

	e.Logger.Fatal(e.Start(":1323"))

}

func main() {

        fmt.Println("\n[CB-Spider (Multi-Cloud Infra Connection Framework)]")
        fmt.Println("\nInitiating REST API Server ...")

        // Run API Server
        apiServer()

}

func listCloudDriver(c echo.Context) error {
	cblog.Info("call ListCloudDriver()")

        var content struct {
		CdrInfo []*dim.CloudDriverInfo `json:"CloudDriver"`
        }

	var err error
        content.CdrInfo, err = dim.ListCloudDriver()
        if err != nil {
                return err
        }

	cblog.Debugf("content %+v\n", content)
        return c.JSON(http.StatusOK, &content)
}

