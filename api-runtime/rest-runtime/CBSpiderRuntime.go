// Rest Runtime Server of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2019.10.

package restruntime

import (
	"crypto/subtle"
	"fmt"
	"time"
	"strings"

	"net/http"
	"os"

	"github.com/chyeh/pubip"

	cr "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	aw "github.com/cloud-barista/cb-spider/api-runtime/rest-runtime/admin-web"
	"github.com/cloud-barista/cb-store/config"
	"github.com/sirupsen/logrus"

	// REST API (echo)
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	// echo-swagger middleware
	_ "github.com/cloud-barista/cb-spider/api-runtime/rest-runtime/docs"
	echoSwagger "github.com/swaggo/echo-swagger"
)

var cblog *logrus.Logger

// @title CB-Spider REST API
// @version latest
// @description CB-Spider REST API

// @contact.name API Support
// @contact.url http://cloud-barista.github.io
// @contact.email contact-to-cloud-barista@googlegroups.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:1024
// @BasePath /spider

// @securityDefinitions.basic BasicAuth

func init() {
	cblog = config.Cblogger
	currentTime := time.Now()
	cr.StartTime = currentTime.Format("2006.01.02 15:04:05 Mon")
	cr.MiddleStartTime = currentTime.Format("2006.01.02.15:04:05")
	cr.ShortStartTime = fmt.Sprintf("T%02d:%02d:%02d", currentTime.Hour(), currentTime.Minute(), currentTime.Second())
	cr.HostIPorName = getHostIPorName()
	cr.ServicePort = getServicePort()
}

// REST API Return struct for boolean type
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

// JSON Simple message struct
type SimpleMsg struct {
	Message string `json:"message" example:"Any message"`
}

//====== temporary trick for shared public IP Host or VirtualBox VM, etc.
//====== user can setup spider server's IP manually.

// unset                      # default: like 'curl ifconfig.co':1024
// LOCALHOST="OFF"            # default: like 'curl ifconfig.co':1024
// LOCALHOST="ON"             # => localhost:1024 
// LOCALHOST="1.2.3.4"        # => 1.2.3.4:1024
// LOCALHOST="1.2.3.4:31024"  # => 1.2.3.4:31024
// LOCALHOST=":31024"         # => like 'curl ifconfig.co':31024
func getHostIPorName() string {

        hostEnv := os.Getenv("LOCALHOST")

        if hostEnv == "ON" {
                return "localhost"
        }

        if hostEnv == "" || hostEnv=="OFF" {
                return getPublicIP()
        }

        // "1.2.3.4"
        if !strings.Contains(hostEnv, ":") {
                return hostEnv
        }

        strs := strings.Split(hostEnv, ":")
        fmt.Println(len(strs))
        if strs[0] =="" {  // ":31024"
                return getPublicIP()
        }else {  // "1.2.3.4:31024"
                return strs[0]
        }
}

func getPublicIP() string {
        ip, err := pubip.Get()
        if err != nil {
                cblog.Error(err)
                hostName, err := os.Hostname()
                if err != nil {
                        cblog.Error(err)
                }
                return hostName
        }

        return ip.String()
}

func getServicePort() string {
        servicePort := ":1024"

        hostEnv := os.Getenv("LOCALHOST")
        if hostEnv == "" || hostEnv=="OFF" || hostEnv=="ON" {
                return servicePort
        }

        // "1.2.3.4"
        if !strings.Contains(hostEnv, ":") {
                return  servicePort
        }

        // ":31024" or "1.2.3.4:31024"
        strs := strings.Split(hostEnv, ":")
        servicePort = ":" + strs[1]

        return servicePort
}

func RunServer() {

	//======================================= setup routes
	routes := []route{
		//----------root
		{"GET", "", aw.SpiderInfo},
		{"GET", "/", aw.SpiderInfo},

		//----------Swagger
		{"GET", "/swagger/*", echoSwagger.WrapHandler},

		//----------EndpointInfo
		{"GET", "/endpointinfo", endpointInfo},

		//----------CloudOS
		{"GET", "/cloudos", listCloudOS},

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
		{"GET", "/vmimage/:Name", getImage},
		{"DELETE", "/vmimage/:Name", deleteImage},

		//----------VMSpec Handler
		{"GET", "/vmspec", listVMSpec},
		{"GET", "/vmspec/:Name", getVMSpec},
		{"GET", "/vmorgspec", listOrgVMSpec},
		{"GET", "/vmorgspec/:Name", getOrgVMSpec},

		//----------VPC Handler
		{"POST", "/vpc", createVPC},
		{"GET", "/vpc", listVPC},
		{"GET", "/vpc/:Name", getVPC},
		{"DELETE", "/vpc/:Name", deleteVPC},
		//-- for subnet
		{"POST", "/vpc/:VPCName/subnet", addSubnet},
		{"DELETE", "/vpc/:VPCName/subnet/:SubnetName", removeSubnet},
		{"DELETE", "/vpc/:VPCName/cspsubnet/:Id", removeCSPSubnet},
		//-- for management
		{"GET", "/allvpc", listAllVPC},
		{"DELETE", "/cspvpc/:Id", deleteCSPVPC},

		//----------SecurityGroup Handler
		{"POST", "/securitygroup", createSecurity},
		{"GET", "/securitygroup", listSecurity},
		{"GET", "/securitygroup/:Name", getSecurity},
		{"DELETE", "/securitygroup/:Name", deleteSecurity},
		//-- for management
		{"GET", "/allsecuritygroup", listAllSecurity},
		{"DELETE", "/cspsecuritygroup/:Id", deleteCSPSecurity},

		//----------KeyPair Handler
		{"POST", "/keypair", createKey},
		{"GET", "/keypair", listKey},
		{"GET", "/keypair/:Name", getKey},
		{"DELETE", "/keypair/:Name", deleteKey},
		//-- for management
		{"GET", "/allkeypair", listAllKey},
		{"DELETE", "/cspkeypair/:Id", deleteCSPKey},
		/*
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
		*/
		//----------VM Handler
		{"POST", "/vm", startVM},
		{"GET", "/vm", listVM},
		{"GET", "/vm/:Name", getVM},
		{"DELETE", "/vm/:Name", terminateVM},
		//-- for management
		{"GET", "/allvm", listAllVM},
		{"DELETE", "/cspvm/:Id", terminateCSPVM},

		{"GET", "/vmstatus", listVMStatus},
		{"GET", "/vmstatus/:Name", getVMStatus},

		{"GET", "/controlvm/:Name", controlVM}, // suspend, resume, reboot

		//-------------------------------------------------------------------//
		//----------SSH RUN
		{"POST", "/sshrun", sshRun},

		//----------AdminWeb Handler
		{"GET", "/adminweb", aw.Frame},
		{"GET", "/adminweb/top", aw.Top},
		{"GET", "/adminweb/driver", aw.Driver},
		{"GET", "/adminweb/credential", aw.Credential},
		{"GET", "/adminweb/region", aw.Region},
		{"GET", "/adminweb/connectionconfig", aw.Connectionconfig},
		{"GET", "/adminweb/spiderinfo", aw.SpiderInfo},

		{"GET", "/adminweb/vpc/:ConnectConfig", aw.VPC},
		{"GET", "/adminweb/vpcmgmt/:ConnectConfig", aw.VPCMgmt},
		{"GET", "/adminweb/securitygroup/:ConnectConfig", aw.SecurityGroup},
		{"GET", "/adminweb/securitygroupmgmt/:ConnectConfig", aw.SecurityGroupMgmt},
		{"GET", "/adminweb/keypair/:ConnectConfig", aw.KeyPair},
		{"GET", "/adminweb/keypairmgmt/:ConnectConfig", aw.KeyPairMgmt},
		{"GET", "/adminweb/vm/:ConnectConfig", aw.VM},
		{"GET", "/adminweb/vmmgmt/:ConnectConfig", aw.VMMgmt},

		{"GET", "/adminweb/vmimage/:ConnectConfig", aw.VMImage},
		{"GET", "/adminweb/vmspec/:ConnectConfig", aw.VMSpec},
	}
	//======================================= setup routes

	// Run API Server
	ApiServer(routes)

}

//================ REST API Server: setup & start
func ApiServer(routes []route) {
	e := echo.New()

	// Middleware
	e.Use(middleware.CORS())
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	API_USERNAME := os.Getenv("API_USERNAME")
	API_PASSWORD := os.Getenv("API_PASSWORD")

	if API_USERNAME != "" && API_PASSWORD != "" {
		cblog.Info("**** Rest Auth Enabled ****")
		e.Use(middleware.BasicAuth(func(username, password string, c echo.Context) (bool, error) {
			// Be careful to use constant time comparison to prevent timing attacks
			if subtle.ConstantTimeCompare([]byte(username), []byte(API_USERNAME)) == 1 &&
				subtle.ConstantTimeCompare([]byte(password), []byte(API_PASSWORD)) == 1 {
				return true, nil
			}
			return false, nil
		}))
	} else {
		cblog.Info("**** Rest Auth Disabled ****")
	}

	for _, route := range routes {
		// /driver => /spider/driver
		route.path = "/spider" + route.path
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

	// for spider logo
	cbspiderRoot := os.Getenv("CBSPIDER_ROOT")
	e.File("/spider/adminweb/images/logo.png", cbspiderRoot+"/api-runtime/rest-runtime/admin-web/images/cb-spider-circle-logo.png")

	e.HideBanner = true
	e.HidePort = true

	spiderBanner()

	e.Logger.Fatal(e.Start(cr.ServicePort))
}

//================ API Info
func apiInfo(c echo.Context) error {
	cblog.Info("call apiInfo()")

	apiInfo := "api info"
	return c.String(http.StatusOK, apiInfo)
}

//================ Endpoint Info
func endpointInfo(c echo.Context) error {
	cblog.Info("call endpointInfo()")

	endpointInfo := fmt.Sprintf("\n  <CB-Spider> Multi-Cloud Infrastructure Federation Framework\n")
	adminWebURL := "http://" + cr.HostIPorName + cr.ServicePort + "/spider/adminweb"
	endpointInfo += fmt.Sprintf("     - AdminWeb: %s\n", adminWebURL)
	restEndPoint := "http://" + cr.HostIPorName + cr.ServicePort + "/spider"
	endpointInfo += fmt.Sprintf("     - REST API: %s\n", restEndPoint)
	// swaggerURL := "http://" + cr.HostIPorName + cr.ServicePort + "/spider/swagger/index.html"
	// endpointInfo += fmt.Sprintf("     - Swagger : %s\n", swaggerURL)
	gRPCServer := "grpc://" + cr.HostIPorName + cr.GoServicePort
	endpointInfo += fmt.Sprintf("     - Go   API: %s\n", gRPCServer)

	return c.String(http.StatusOK, endpointInfo)
}

func spiderBanner() {
	fmt.Println("\n  <CB-Spider> Multi-Cloud Infrastructure Federation Framework")

	// AdminWeb
	adminWebURL := "http://" + cr.HostIPorName + cr.ServicePort + "/spider/adminweb"
	fmt.Printf("     - AdminWeb: %s\n", adminWebURL)

	// REST API EndPoint
	restEndPoint := "http://" + cr.HostIPorName + cr.ServicePort + "/spider"
	fmt.Printf("     - REST API: %s\n", restEndPoint)

	// Swagger
	// swaggerURL := "http://" + cr.HostIPorName + cr.ServicePort + "/spider/swagger/index.html"
	// fmt.Printf("     - Swagger : %s\n", swaggerURL)
}
