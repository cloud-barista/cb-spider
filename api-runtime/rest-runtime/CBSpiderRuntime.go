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

        // REST API (echo)
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

