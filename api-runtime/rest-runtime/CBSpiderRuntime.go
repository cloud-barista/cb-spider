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
	"strings"
	"time"

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

	"github.com/natefinch/lumberjack"
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

	// REST and GO SERVER_ADDRESS since v0.4.4
	cr.ServerIPorName = getServerIPorName("SERVER_ADDRESS")
	cr.ServerPort = getServerPort("SERVER_ADDRESS")

	// REST SERVICE_ADDRESS for AdminWeb since v0.4.4
	cr.ServiceIPorName = getServiceIPorName("SERVICE_ADDRESS")
	cr.ServicePort = getServicePort("SERVICE_ADDRESS")
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

//// CB-Spider Servcie Address Configuration
////   cf)  https://github.com/cloud-barista/cb-spider/wiki/CB-Spider-Service-Address-Configuration

// REST and GO SERVER_ADDRESS since v0.4.4

// unset                           # default: like 'curl ifconfig.co':1024
// SERVER_ADDRESS="1.2.3.4:3000"  # => 1.2.3.4:3000
// SERVER_ADDRESS=":3000"         # => like 'curl ifconfig.co':3000
// SERVER_ADDRESS="localhost"      # => localhost:1024
// SERVER_ADDRESS="1.2.3.4:3000"        # => 1.2.3.4::3000
func getServerIPorName(env string) string {

	hostEnv := os.Getenv(env) // SERVER_ADDRESS or SERVICE_ADDRESS

	if hostEnv == "" {
		return getPublicIP()
	}

	// "1.2.3.4" or "localhost"
	if !strings.Contains(hostEnv, ":") {
		return hostEnv
	}

	strs := strings.Split(hostEnv, ":")
	fmt.Println(len(strs))
	if strs[0] == "" { // ":31024"
		return getPublicIP()
	} else { // "1.2.3.4:31024" or "localhost:31024"
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

func getServerPort(env string) string {
	// default REST Service Port
	servicePort := ":1024"

	hostEnv := os.Getenv(env) // SERVER_ADDRESS or SERVICE_ADDRESS
	if hostEnv == "" {
		return servicePort
	}

	// "1.2.3.4" or "localhost"
	if !strings.Contains(hostEnv, ":") {
		return servicePort
	}

	// ":31024" or "1.2.3.4:31024" or "localhost:31024"
	strs := strings.Split(hostEnv, ":")
	servicePort = ":" + strs[1]

	return servicePort
}

// unset  SERVER_ADDRESS => SERVICE_ADDRESS
func getServiceIPorName(env string) string {
	hostEnv := os.Getenv(env)
	if hostEnv == "" {
		return cr.ServerIPorName
	}
	return getServerIPorName(env)
}

// unset  SERVER_ADDRESS => SERVICE_ADDRESS
func getServicePort(env string) string {
	hostEnv := os.Getenv(env)
	if hostEnv == "" {
		return cr.ServerPort
	}
	return getServerPort(env)
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
		{"GET", "/endpointinfo", EndpointInfo},

		//----------EndpointInfo
		{"GET", "/healthcheck", EndpointInfo},

		//----------CloudOS
		{"GET", "/cloudos", ListCloudOS},

		//----------CloudOSMetaInfo
		{"GET", "/cloudos/metainfo/:CloudOSName", GetCloudOSMetaInfo},

		//----------CloudDriverInfo
		{"POST", "/driver", RegisterCloudDriver},
		{"POST", "/driver/upload", UploadCloudDriver},
		{"GET", "/driver", ListCloudDriver},
		{"GET", "/driver/:DriverName", GetCloudDriver},
		{"DELETE", "/driver/:DriverName", UnRegisterCloudDriver},

		//----------CredentialInfo
		{"POST", "/credential", RegisterCredential},
		{"GET", "/credential", ListCredential},
		{"GET", "/credential/:CredentialName", GetCredential},
		{"DELETE", "/credential/:CredentialName", UnRegisterCredential},

		//----------RegionInfo
		{"POST", "/region", RegisterRegion},
		{"GET", "/region", ListRegion},
		{"GET", "/region/:RegionName", GetRegion},
		{"DELETE", "/region/:RegionName", UnRegisterRegion},

		//----------ConnectionConfigInfo
		{"POST", "/connectionconfig", CreateConnectionConfig},
		{"GET", "/connectionconfig", ListConnectionConfig},
		{"GET", "/connectionconfig/:ConfigName", GetConnectionConfig},
		{"DELETE", "/connectionconfig/:ConfigName", DeleteConnectionConfig},

		//-------------------------------------------------------------------//

		//----------Image Handler
		{"POST", "/vmimage", CreateImage},
		{"GET", "/vmimage", ListImage},
		{"GET", "/vmimage/:Name", GetImage},
		{"DELETE", "/vmimage/:Name", DeleteImage},

		//----------VMSpec Handler
		{"GET", "/vmspec", ListVMSpec},
		{"GET", "/vmspec/:Name", GetVMSpec},
		{"GET", "/vmorgspec", ListOrgVMSpec},
		{"GET", "/vmorgspec/:Name", GetOrgVMSpec},

		//----------VPC Handler
		{"POST", "/regvpc", RegisterVPC},
		{"DELETE", "/regvpc/:Name", UnregisterVPC},

		{"POST", "/vpc", CreateVPC},
		{"GET", "/vpc", ListVPC},
		{"GET", "/vpc/:Name", GetVPC},
		{"DELETE", "/vpc/:Name", DeleteVPC},
		//-- for subnet
		{"POST", "/vpc/:VPCName/subnet", AddSubnet},
		{"DELETE", "/vpc/:VPCName/subnet/:SubnetName", RemoveSubnet},
		{"DELETE", "/vpc/:VPCName/cspsubnet/:Id", RemoveCSPSubnet},
		//-- for management
		{"GET", "/allvpc", ListAllVPC},
		{"DELETE", "/cspvpc/:Id", DeleteCSPVPC},

		//----------SecurityGroup Handler
		{"GET", "/getsecuritygroupowner", GetSGOwnerVPC},
		{"POST", "/regsecuritygroup", RegisterSecurity},
		{"DELETE", "/regsecuritygroup/:Name", UnregisterSecurity},

		{"POST", "/securitygroup", CreateSecurity},
		{"GET", "/securitygroup", ListSecurity},
		{"GET", "/securitygroup/:Name", GetSecurity},
		{"DELETE", "/securitygroup/:Name", DeleteSecurity},
		//-- for rule
		{"POST", "/securitygroup/:SGName/rules", AddRules},
		{"DELETE", "/securitygroup/:SGName/rules", RemoveRules}, // no force option
		// no CSP Option, {"DELETE", "/securitygroup/:SGName/csprules", RemoveCSPRules},
		//-- for management
		{"GET", "/allsecuritygroup", ListAllSecurity},
		{"DELETE", "/cspsecuritygroup/:Id", DeleteCSPSecurity},

		//----------KeyPair Handler
		{"POST", "/regkeypair", RegisterKey},
		{"DELETE", "/regkeypair/:Name", UnregisterKey},

		{"POST", "/keypair", CreateKey},
		{"GET", "/keypair", ListKey},
		{"GET", "/keypair/:Name", GetKey},
		{"DELETE", "/keypair/:Name", DeleteKey},
		//-- for management
		{"GET", "/allkeypair", ListAllKey},
		{"DELETE", "/cspkeypair/:Id", DeleteCSPKey},
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
		{"GET", "/getvmusingresources", GetVMUsingRS},
		{"POST", "/regvm", RegisterVM},
		{"DELETE", "/regvm/:Name", UnregisterVM},

		{"POST", "/vm", StartVM},
		{"GET", "/vm", ListVM},
		{"GET", "/vm/:Name", GetVM},
		{"DELETE", "/vm/:Name", TerminateVM},
		//-- for management
		{"GET", "/allvm", ListAllVM},
		{"DELETE", "/cspvm/:Id", TerminateCSPVM},

		{"GET", "/vmstatus", ListVMStatus},
		{"GET", "/vmstatus/:Name", GetVMStatus},

		{"GET", "/controlvm/:Name", ControlVM}, // suspend, resume, reboot
		// only for AdminWeb
		{"PUT", "/controlvm/:Name", ControlVM}, // suspend, resume, reboot

		//----------NLB Handler
		{"GET", "/getnlbowner", GetNLBOwnerVPC},
		{"POST", "/regnlb", RegisterNLB},
		{"DELETE", "/regnlb/:Name", UnregisterNLB},

		{"POST", "/nlb", CreateNLB},
		{"GET", "/nlb", ListNLB},
		{"GET", "/nlb/:Name", GetNLB},
		{"DELETE", "/nlb/:Name", DeleteNLB},
		//-- for vm
		{"POST", "/nlb/:Name/vms", AddNLBVMs},
		{"DELETE", "/nlb/:Name/vms", RemoveNLBVMs}, // no force option
		{"PUT", "/nlb/:Name/listener", ChangeListener},
		{"PUT", "/nlb/:Name/vmgroup", ChangeVMGroup},
		{"PUT", "/nlb/:Name/healthchecker", ChangeHealthChecker},
		{"GET", "/nlb/:Name/health", GetVMGroupHealthInfo},

		//-- for management
		{"GET", "/allnlb", ListAllNLB},
		{"DELETE", "/cspnlb/:Id", DeleteCSPNLB},


		//----------Disk Handler
		{"POST", "/regdisk", RegisterDisk},
		{"DELETE", "/regdisk/:Name", UnregisterDisk},

		{"POST", "/disk", CreateDisk},
		{"GET", "/disk", ListDisk},
		{"GET", "/disk/:Name", GetDisk},
		{"PUT", "/disk/:Name/size", ChangeDiskSize},
		{"DELETE", "/disk/:Name", DeleteDisk},
		//-- for vm
		{"PUT", "/disk/:Name/attach", AttachDisk},
		{"PUT", "/disk/:Name/detach", DetachDisk},

		//-- for management
		{"GET", "/alldisk", ListAllDisk},
		{"DELETE", "/cspdisk/:Id", DeleteCSPDisk},


		//----------MyImage Handler
		{"POST", "/regmyimage", RegisterMyImage},
		{"DELETE", "/regmyimage/:Name", UnregisterMyImage},

		{"POST", "/myimage", SnapshotVM},
		{"GET", "/myimage", ListMyImage},
		{"GET", "/myimage/:Name", GetMyImage},
		{"DELETE", "/myimage/:Name", DeleteMyImage},

		//-- for management
		{"GET", "/allmyimage", ListAllMyImage},
		{"DELETE", "/cspmyimage/:Id", DeleteCSPMyImage},

		//----------Cluster Handler
		{"GET", "/getclusterowner", GetClusterOwnerVPC},
		{"POST", "/regcluster", RegisterCluster},
		{"DELETE", "/regcluster/:Name", UnregisterCluster},

		{"POST", "/cluster", CreateCluster},
		{"GET", "/cluster", ListCluster},
		{"GET", "/cluster/:Name", GetCluster},
		{"DELETE", "/cluster/:Name", DeleteCluster},
		//-- for NodeGroup
		{"POST", "/cluster/:Name/nodegroup", AddNodeGroup},
		{"DELETE", "/cluster/:Name/nodegroup/:NodeGroupName", RemoveNodeGroup},
		{"PUT", "/cluster/:Name/nodegroup/:NodeGroupName/onautoscaling", SetNodeGroupAutoScaling},
		{"PUT", "/cluster/:Name/nodegroup/:NodeGroupName/autoscalesize", ChangeNodeGroupScaling},
		{"PUT", "/cluster/:Name/upgrade", UpgradeCluster},
		{"GET", "/cspvm/:Id", GetCSPVM},

		//-- for management
		{"GET", "/allcluster", ListAllCluster},
		{"DELETE", "/cspcluster/:Id", DeleteCSPCluster},

		//-- only for WebTool
		{"GET", "/nscluster", AllClusterList},

		//-------------------------------------------------------------------//
        //----------Additional Info
		{"GET", "/cspresourcename/:Name", GetCSPResourceName},
		{"GET", "/cspresourceinfo/:Name", GetCSPResourceInfo},
		//----------AnyCall Handler
		{"POST", "/anycall", AnyCall},

		//-------------------------------------------------------------------//
		//----------SPLock Info
		{"GET", "/splockinfo", GetAllSPLockInfo},
		//----------SSH RUN
		{"POST", "/sshrun", SSHRun},

		//----------AdminWeb Handler
		{"GET", "/adminweb", aw.Frame},
		{"GET", "/adminweb/top", aw.Top},
		{"GET", "/adminweb/log", aw.Log},

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
		{"GET", "/adminweb/nlb/:ConnectConfig", aw.NLB},
		{"GET", "/adminweb/nlbmgmt/:ConnectConfig", aw.NLBMgmt},
		{"GET", "/adminweb/disk/:ConnectConfig", aw.Disk},
		{"GET", "/adminweb/diskmgmt/:ConnectConfig", aw.DiskMgmt},

		{"GET", "/adminweb/myimage/:ConnectConfig", aw.MyImage},
		{"GET", "/adminweb/myimagemgmt/:ConnectConfig", aw.MyImageMgmt},
		{"GET", "/adminweb/vmimage/:ConnectConfig", aw.VMImage},
		{"GET", "/adminweb/vmspec/:ConnectConfig", aw.VMSpec},

		{"GET", "/adminweb/cluster/:ConnectConfig", aw.Cluster},
		{"GET", "/adminweb/clustermgmt/:ConnectConfig", aw.ClusterMgmt},
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

        cbspiderRoot := os.Getenv("CBSPIDER_ROOT")

	// for HTTP Access Log
	e.Logger.SetOutput(&lumberjack.Logger{
	    Filename:   cbspiderRoot+"/log/http-access.log",
	    MaxSize:    10,  // megabytes
	    MaxBackups: 10,  // number of backups
	    MaxAge:     31,  // days
	})

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
	e.File("/spider/adminweb/images/logo.png", cbspiderRoot+"/api-runtime/rest-runtime/admin-web/images/cb-spider-circle-logo.png")
	e.File("/spider/adminweb/images/pmks.png", cbspiderRoot+"/api-runtime/rest-runtime/admin-web/images/cb-spider-pmks-logo.png")

	e.HideBanner = true
	e.HidePort = true

	spiderBanner()

	e.Logger.Fatal(e.Start(cr.ServerPort))
}

//================ API Info
func apiInfo(c echo.Context) error {
	cblog.Info("call apiInfo()")

	apiInfo := "api info"
	return c.String(http.StatusOK, apiInfo)
}

//================ Endpoint Info
func EndpointInfo(c echo.Context) error {
	cblog.Info("call endpointInfo()")

	endpointInfo := fmt.Sprintf("\n  <CB-Spider> Multi-Cloud Infrastructure Federation Framework\n")
	adminWebURL := "http://" + cr.ServiceIPorName + cr.ServicePort + "/spider/adminweb"
	endpointInfo += fmt.Sprintf("     - AdminWeb: %s\n", adminWebURL)
	restEndPoint := "http://" + cr.ServiceIPorName + cr.ServicePort + "/spider"
	endpointInfo += fmt.Sprintf("     - REST API: %s\n", restEndPoint)
	// swaggerURL := "http://" + cr.ServiceIPorName + cr.ServicePort + "/spider/swagger/index.html"
	// endpointInfo += fmt.Sprintf("     - Swagger : %s\n", swaggerURL)
	gRPCServer := "grpc://" + cr.ServiceIPorName + cr.GoServicePort
	endpointInfo += fmt.Sprintf("     - Go   API: %s\n", gRPCServer)

	return c.String(http.StatusOK, endpointInfo)
}

func spiderBanner() {
	fmt.Println("\n  <CB-Spider> Multi-Cloud Infrastructure Federation Framework")

	// AdminWeb
	adminWebURL := "http://" + cr.ServiceIPorName + cr.ServicePort + "/spider/adminweb"
	fmt.Printf("     - AdminWeb: %s\n", adminWebURL)

	// REST API EndPoint
	restEndPoint := "http://" + cr.ServiceIPorName + cr.ServicePort + "/spider"
	fmt.Printf("     - REST API: %s\n", restEndPoint)

	// Swagger
	// swaggerURL := "http://" + cr.ServiceIPorName + cr.ServicePort + "/spider/swagger/index.html"
	// fmt.Printf("     - Swagger : %s\n", swaggerURL)
}
