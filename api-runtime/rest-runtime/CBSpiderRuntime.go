// Rest Runtime Server of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by powerkim@etri.re.kr, 2019.10.

package main

import (
	"fmt"

	"github.com/cloud-barista/cb-store/config"
	"github.com/sirupsen/logrus"

	// REST API (echo)
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

var cblog *logrus.Logger

func init() {
	cblog = config.Cblogger
}

// REST API Return struct for boolena type
type BooleanInfo struct {
	Result string // true or false
}

type StatusInfo struct {
	Status string // PENDING | RUNNING | SUSPENDING | SUSPENDED | REBOOTING | TERMINATING | TERMINATED
}

//ex) {"POST", "/driver", registerCloudDriver}
type route struct {
	method, path string
	function     echo.HandlerFunc
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

		//-------------------------------------------------------------------//

		//----------Image Handler
		{"POST", "/vmimage", createImage},
		{"GET", "/vmimage", listImage},
		{"GET", "/vmimage/:ImageName", getImage},
		{"DELETE", "/vmimage/:ImageName", deleteImage},

		//----------VNet Handler
		{"POST", "/vnetwork", createVNetwork},
		{"GET", "/vnetwork", listVNetwork},
		{"GET", "/vnetwork/:VNetId", getVNetwork},
		{"DELETE", "/vnetwork/:VNetId", deleteVNetwork},

		//----------SecurityGroup Handler
		{"POST", "/securitygroup", createSecurity},
		{"GET", "/securitygroup", listSecurity},
		{"GET", "/securitygroup/:SecurityGroupId", getSecurity},
		{"DELETE", "/securitygroup/:SecurityGroupId", deleteSecurity},

		//----------KeyPair Handler
		{"POST", "/keypair", createKey},
		{"GET", "/keypair", listKey},
		{"GET", "/keypair/:KeyPairId", getKey},
		{"DELETE", "/keypair/:KeyPairId", deleteKey},

		//----------VNic Handler
		{"POST", "/vnic", createVNic},
		{"GET", "/vnic", listVNic},
		{"GET", "/vnic/:VNicId", getVNic},
		{"DELETE", "/vnic/:VNicId", deleteVNic},

		//----------PublicIP Handler
		{"POST", "/publicip", createPublicIP},
		{"GET", "/publicip", listPublicIP},
		{"GET", "/publicip/:PublicIPId", getPublicIP},
		{"DELETE", "/publicip/:PublicIPId", deletePublicIP},

		//----------VM Handler
		{"POST", "/vm", startVM},
		{"GET", "/vm", listVM},
		{"GET", "/vm/:VmId", getVM},
		{"DELETE", "/vm/:VmId", terminateVM},

		{"GET", "/vmstatus", listVMStatus},
		{"GET", "/vmstatus/:VmId", getVMStatus},

		{"GET", "/controlvm/:VmId", controlVM}, // suspend, resume, reboot

		//-------------------------------------------------------------------//
		//----------SSH RUN
		{"POST", "/sshrun", sshRun},
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
	e.Use(middleware.CORS())
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
