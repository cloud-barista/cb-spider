// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2022.09.

package restruntime

import (

        cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
        cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

        // REST API (echo)
        "net/http"

        "github.com/labstack/echo/v4"

        "strconv"
        "strings"
)


//================ NLB Handler

func GetNLBOwnerVPC(c echo.Context) error {
        cblog.Info("call GetNLBOwnerVPC()")

        var req struct {
                ConnectionName string
                ReqInfo        struct {
                        CSPId          string
                }
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.GetNLBOwnerVPC(req.ConnectionName, req.ReqInfo.CSPId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

type NLBRegisterReq struct {
        ConnectionName string
        ReqInfo        struct {
                VPCName           string
                Name           string
                CSPId          string
        }
}

func RegisterNLB(c echo.Context) error {
        cblog.Info("call RegisterNLB()")

        req := NLBRegisterReq{}

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // create UserIID
        userIId := cres.IID{req.ReqInfo.Name, req.ReqInfo.CSPId}

        // Call common-runtime API
        result, err := cmrt.RegisterNLB(req.ConnectionName, req.ReqInfo.VPCName, userIId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func UnregisterNLB(c echo.Context) error {
        cblog.Info("call UnregisterNLB()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.UnregisterResource(req.ConnectionName, rsNLB, c.Param("Name"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}

type NLBReq struct {
        ConnectionName string
        ReqInfo        struct {
                Name		string
                VPCName		string
                Type		string 	// PUBLIC(V) | INTERNAL
                Scope		string 	// REGION(V) | GLOBAL

		//------ Frontend
		Listener        cres.ListenerInfo

		//------ Backend
		VMGroup         VMGroupReq
		HealthChecker   HealthCheckerReq  // for int mapping with string
        }
}
// for int mapping with string
type HealthCheckerReq struct {
	Protocol        string  // TCP|HTTP|HTTPS
	Port            string  // Listener Port or 1-65535
	Interval        string     // secs, Interval time between health checks.
	Timeout         string     // secs, Waiting time to decide an unhealthy VM when no response.
	Threshold       string     // num, The number of continuous health checks to change the VM status.
}

// for VM IID mapping
type VMGroupReq struct {
        Protocol        string  // TCP|HTTP|HTTPS
        Port            string  // Listener Port or 1-65535
        VMs        	[]string
}

func CreateNLB(c echo.Context) error {
        cblog.Info("call CreateNLB()")

        req := NLBReq{}

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Rest RegInfo => Driver ReqInfo
        reqInfo := cres.NLBInfo{
                IId:           cres.IID{req.ReqInfo.Name, req.ReqInfo.Name}, 
                VpcIID:        cres.IID{req.ReqInfo.VPCName, ""},
                Type:        	req.ReqInfo.Type,
                Scope:        	req.ReqInfo.Scope,
                Listener: 	req.ReqInfo.Listener,
                VMGroup: 	convertVMGroupInfo(req.ReqInfo.VMGroup),
                //HealthChecker: below
        }
	healthChecker, err := convertHealthCheckerInfo(req.ReqInfo.HealthChecker)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	reqInfo.HealthChecker = healthChecker

        // Call common-runtime API
        result, err := cmrt.CreateNLB(req.ConnectionName, rsNLB, reqInfo)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

func convertVMGroupInfo(vgInfo VMGroupReq) cres.VMGroupInfo {
	vmIIDList := []cres.IID{}
	for _, vm := range vgInfo.VMs {
		vmIIDList = append(vmIIDList, cres.IID{vm, ""})	
	}
	return cres.VMGroupInfo{vgInfo.Protocol, vgInfo.Port, &vmIIDList, "", nil}
}

func convertHealthCheckerInfo(hcInfo HealthCheckerReq) (cres.HealthCheckerInfo, error) {
	// default: "default" or "" or "-1" => -1

	var err error
	// (1) Interval
	interval := -1
	strInterval := strings.ToLower(hcInfo.Interval)
	switch  strInterval {
	case "default", "", "-1":
	default: 
		interval, err = strconv.Atoi(hcInfo.Interval)
		if err != nil {
			cblog.Error(err)
			return cres.HealthCheckerInfo{}, err
		}
	}

	// (2) Timeout
        timeout := -1
	strTimeout := strings.ToLower(hcInfo.Timeout)
        switch  strTimeout {
	case "default", "", "-1":
        default:
		timeout, err = strconv.Atoi(hcInfo.Timeout)
                if err != nil {
                        cblog.Error(err)
                        return cres.HealthCheckerInfo{}, err
                }
        }

	// (3) Threshold
        threshold := -1
	strThreshold := strings.ToLower(hcInfo.Threshold)
        switch  strThreshold {
	case "default", "", "-1":
        default:
		threshold, err = strconv.Atoi(hcInfo.Threshold)
                if err != nil {
                        cblog.Error(err)
                        return cres.HealthCheckerInfo{}, err
                }
        }

        return cres.HealthCheckerInfo{hcInfo.Protocol, hcInfo.Port, interval, timeout, threshold, "", nil}, nil
}

func ListNLB(c echo.Context) error {
        cblog.Info("call ListNLB()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // To support for Get-Query Param Type API
        if req.ConnectionName == "" {
                req.ConnectionName = c.QueryParam("ConnectionName")
        }

        // Call common-runtime API
        result, err := cmrt.ListNLB(req.ConnectionName, rsNLB)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        var jsonResult struct {
                Result []*cres.NLBInfo `json:"nlb"`
        }
        jsonResult.Result = result
        return c.JSON(http.StatusOK, &jsonResult)
}

// list all NLBs for management
// (1) get args from REST Call
// (2) get all NLB List by common-runtime API
// (3) return REST Json Format
func ListAllNLB(c echo.Context) error {
        cblog.Info("call ListAllNLB()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // To support for Get-Query Param Type API
        if req.ConnectionName == "" {
                req.ConnectionName = c.QueryParam("ConnectionName")
        }

        // Call common-runtime API
        allResourceList, err := cmrt.ListAllResource(req.ConnectionName, rsNLB)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, &allResourceList)
}

func GetNLB(c echo.Context) error {
        cblog.Info("call GetNLB()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // To support for Get-Query Param Type API
        if req.ConnectionName == "" {
                req.ConnectionName = c.QueryParam("ConnectionName")
        }

        // Call common-runtime API
        result, err := cmrt.GetNLB(req.ConnectionName, rsNLB, c.Param("Name"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

func AddNLBVMs(c echo.Context) error {
        cblog.Info("call AddNLBVMs()")

        var req struct {
                ConnectionName string
                ReqInfo        struct {
                        VMs      []string
                }
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.AddNLBVMs(req.ConnectionName, c.Param("Name"), req.ReqInfo.VMs)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

func RemoveNLBVMs(c echo.Context) error {
        cblog.Info("call RemoveNLBVMs()")

        var req struct {
                ConnectionName string
                ReqInfo        struct {
                        VMs      []string
                }
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.RemoveNLBVMs(req.ConnectionName, c.Param("Name"), req.ReqInfo.VMs)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

//---------------------------------------------------//
// @todo  To support or not will be decided later.   //
//---------------------------------------------------//
func ChangeListener(c echo.Context) error {
        cblog.Info("call ChangeListener()")

        var req struct {
                ConnectionName string
                ReqInfo        struct {
                        Protocol      	string
                        Port		string
                }
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        reqInfo := cres.ListenerInfo{
                Protocol:       req.ReqInfo.Protocol,
                Port:       	req.ReqInfo.Port,
        }

        // Call common-runtime API
        result, err := cmrt.ChangeListener(req.ConnectionName, c.Param("Name"), reqInfo)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

//---------------------------------------------------//
// @todo  To support or not will be decided later.   //
//---------------------------------------------------//
func ChangeVMGroup(c echo.Context) error {
        cblog.Info("call ChangeVMGroup()")

        var req struct {
                ConnectionName string
                ReqInfo        struct {
                        Protocol      string
                        Port          string
                }
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        reqInfo := cres.VMGroupInfo{
                Protocol:       req.ReqInfo.Protocol,
                Port:       	req.ReqInfo.Port,
	}

        // Call common-runtime API
        result, err := cmrt.ChangeVMGroup(req.ConnectionName, c.Param("Name"), reqInfo)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

//---------------------------------------------------//
// @todo  To support or not will be decided later.   //
//---------------------------------------------------//
func ChangeHealthChecker(c echo.Context) error {
        cblog.Info("call ChangeHealthChecker()")

        var req struct {
                ConnectionName string
                ReqInfo        struct {
                        Protocol      string
                        Port          string
                        Interval      string
                        Timeout       string
                        Threshold     string
                }
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

	interval, err := strconv.Atoi(req.ReqInfo.Interval)
	timeout, err := strconv.Atoi(req.ReqInfo.Timeout)
	threshold, err := strconv.Atoi(req.ReqInfo.Threshold)

        reqInfo := cres.HealthCheckerInfo{
                Protocol:       req.ReqInfo.Protocol,
                Port:           req.ReqInfo.Port,
                Interval:       interval,
                Timeout:       	timeout,
                Threshold:      threshold,
        }

        // Call common-runtime API
        result, err := cmrt.ChangeHealthChecker(req.ConnectionName, c.Param("Name"), reqInfo)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

func GetVMGroupHealthInfo(c echo.Context) error {
        cblog.Info("call GetVMGroupHealthInfo()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // To support for Get-Query Param Type API
        if req.ConnectionName == "" {
                req.ConnectionName = c.QueryParam("ConnectionName")
        }

        // Call common-runtime API
        result, err := cmrt.GetVMGroupHealthInfo(req.ConnectionName, c.Param("Name"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        var jsonResult struct {
                Result cres.HealthInfo `json:"healthinfo"`
        }
        jsonResult.Result = *result

        return c.JSON(http.StatusOK, &jsonResult)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func DeleteNLB(c echo.Context) error {
        cblog.Info("call DeleteNLB()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, _, err := cmrt.DeleteResource(req.ConnectionName, rsNLB, c.Param("Name"), c.QueryParam("force"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func DeleteCSPNLB(c echo.Context) error {
        cblog.Info("call DeleteCSPNLB()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, rsNLB, c.Param("Id"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}
