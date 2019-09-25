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


func apiServer() {

        e := echo.New()

        // Middleware
        e.Use(middleware.Logger())
        e.Use(middleware.Recover())

        e.GET("/", func(c echo.Context) error {
                return c.String(http.StatusOK, "CB-Spider!")
        })

        // Route
        e.POST("/driver", registerCloudDriver)
        e.GET("/driver", listCloudDriver)
        e.GET("/driver/:DriverName", getCloudDriver)

	e.Logger.Fatal(e.Start(":1323"))

}

func main() {

        fmt.Println("\n[CB-Spider (Multi-Cloud Infra Connection Framework)]")
        fmt.Println("\nInitiating REST API Server ...")

        // Run API Server
        apiServer()

}

//================ Handler
func registerCloudDriver(c echo.Context) error {
        cblog.Info("call registerCloudDriver()")

	req := &dim.CloudDriverInfo{}
        if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldinfoList, err:= dim.RegisterCloudDriverInfo(*req)
        if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        return c.JSON(http.StatusOK, &cldinfoList)
}

func listCloudDriver(c echo.Context) error {
        cblog.Info("call listCloudDriver()")

        cldinfoList, err:= dim.ListCloudDriver()
        if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        return c.JSON(http.StatusOK, &cldinfoList)
}

func getCloudDriver(c echo.Context) error {
        cblog.Info("call getCloudDriver()")

        cldinfo, err:= dim.GetCloudDriver(c.Param("DriverName"))
        if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        return c.JSON(http.StatusOK, &cldinfo)
}

