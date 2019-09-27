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
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/credential-info-manager"
	rim "github.com/cloud-barista/cb-spider/cloud-info-manager/region-info-manager"
	ccim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"

        // REST API (echo)
        "net/http"
        "github.com/labstack/echo"
        "github.com/labstack/echo/middleware"
)

var cblog *logrus.Logger

func init() {
        cblog = config.Cblogger
}


//ex) {"POST", "/driver", registerCloudDriver}
type route struct {
	method, path string
	function echo.HandlerFunc
}

func main() {

//======================================= setup routes
        routes := []route{
			//----------CloudDriverInfo
			{"POST", "/driver", registerCloudDriver},
                        {"GET", "/driver", listCloudDriver},
                        {"GET", "/driver/:DriverName", getCloudDriver},
                        {"DELETE", "/driver/:DriverName", unRegisterCloudDriver},

			//----------CredentialInfo
			{"POST", "/credential", registerCredential},
                        {"GET", "/credential", listCredential},
                        {"GET", "/credential/:CredentialName", getCredential},
                        {"DELETE", "/credential/:CredentialName", unRegisterCredential},

			//----------RegionInfo
			{"POST", "/region", registerRegion},
                        {"GET", "/region", listRegion},
                        {"GET", "/region/:RegionName", getRegion},
                        {"DELETE", "/region/:RegionName", unRegisterRegion},

			//----------ConnectionConfigInfo
			{"POST", "/connectionconfig", createConnectionConfig},
                        {"GET", "/connectionconfig", listConnectionConfig},
                        {"GET", "/connectionconfig/:ConfigName", getConnectionConfig},
                        {"DELETE", "/connectionconfig/:ConfigName", deleteConnectionConfig},

                        }
//======================================= setup routes

        fmt.Println("\n[CB-Spider:Cloud Info Management Framework]")
        fmt.Println("\n   Initiating REST API Server....__^..^__....\n\n")

        // Run API Server
        ApiServer(routes, ":1024")
}


//================ REST API Server: setup & start
func ApiServer(routes []route, strPort string) {
        e := echo.New()

        // Middleware
        e.Use(middleware.Logger())
        e.Use(middleware.Recover())

        for _, route := range routes {
                switch route.method {
                case "POST":
                        e.POST(route.path, route.function)
                case "GET":
                        e.GET(route.path, route.function)
                case "PUT":
                        e.PUT(route.path, route.function)
                case "DELETE":
                        e.DELETE(route.path, route.function)

                }
        }

	e.HideBanner = true
	if strPort == "" {
		strPort = ":1323"
	}
        e.Logger.Fatal(e.Start(strPort))
}

//================ CloudDriver Handler
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

func unRegisterCloudDriver(c echo.Context) error {
        cblog.Info("call unRegisterCloudDriver()")

        result, err:= dim.UnRegisterCloudDriver(c.Param("DriverName"))
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        return c.JSON(http.StatusOK, &result)
}

//================ Credential Handler
func registerCredential(c echo.Context) error {
        cblog.Info("call registerCredential()")

        req := &cim.CredentialInfo{}
        if err := c.Bind(req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        crdinfoList, err:= cim.RegisterCredentialInfo(*req)
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        return c.JSON(http.StatusOK, &crdinfoList)
}

func listCredential(c echo.Context) error {
        cblog.Info("call listCredential()")

        crdinfoList, err:= cim.ListCredential()
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        return c.JSON(http.StatusOK, &crdinfoList)
}

func getCredential(c echo.Context) error {
        cblog.Info("call getCredential()")

        crdinfo, err:= cim.GetCredential(c.Param("CredentialName"))
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        return c.JSON(http.StatusOK, &crdinfo)
}

func unRegisterCredential(c echo.Context) error {
        cblog.Info("call unRegisterCredential()")

        result, err:= cim.UnRegisterCredential(c.Param("CredentialName"))
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        return c.JSON(http.StatusOK, &result)
}

//================ Region Handler
func registerRegion(c echo.Context) error {
        cblog.Info("call registerRegion()")

        req := &rim.RegionInfo{}
        if err := c.Bind(req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        crdinfoList, err:= rim.RegisterRegionInfo(*req)
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        return c.JSON(http.StatusOK, &crdinfoList)
}

func listRegion(c echo.Context) error {
        cblog.Info("call listRegion()")

        crdinfoList, err:= rim.ListRegion()
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        return c.JSON(http.StatusOK, &crdinfoList)
}

func getRegion(c echo.Context) error {
        cblog.Info("call getRegion()")

        crdinfo, err:= rim.GetRegion(c.Param("RegionName"))
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        return c.JSON(http.StatusOK, &crdinfo)
}

func unRegisterRegion(c echo.Context) error {
        cblog.Info("call unRegisterRegion()")

        result, err:= rim.UnRegisterRegion(c.Param("RegionName"))
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        return c.JSON(http.StatusOK, &result)
}

//================ ConnectionConfig Handler
func createConnectionConfig(c echo.Context) error {
        cblog.Info("call registerConnectionConfig()")

        req := &ccim.ConnectionConfigInfo{}
        if err := c.Bind(req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        crdinfoList, err:= ccim.CreateConnectionConfigInfo(*req)
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        return c.JSON(http.StatusOK, &crdinfoList)
}

func listConnectionConfig(c echo.Context) error {
        cblog.Info("call listConnectionConfig()")

        crdinfoList, err:= ccim.ListConnectionConfig()
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        return c.JSON(http.StatusOK, &crdinfoList)
}

func getConnectionConfig(c echo.Context) error {
        cblog.Info("call getConnectionConfig()")

        crdinfo, err:= ccim.GetConnectionConfig(c.Param("ConfigName"))
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        return c.JSON(http.StatusOK, &crdinfo)
}

func deleteConnectionConfig(c echo.Context) error {
        cblog.Info("call unRegisterConnectionConfig()")

        result, err:= ccim.DeleteConnectionConfig(c.Param("ConfigName"))
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        return c.JSON(http.StatusOK, &result)
}

