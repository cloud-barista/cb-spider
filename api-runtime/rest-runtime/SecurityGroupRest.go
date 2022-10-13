// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.

package restruntime

import (

        cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
        cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

        // REST API (echo)
        "net/http"

        "github.com/labstack/echo/v4"

        "strconv"
)


//================ SecurityGroup Handler

type securityGroupRegisterReq struct {
        ConnectionName string
        ReqInfo        struct {
                VPCName           string
                Name           string
                CSPId          string
        }
}

func RegisterSecurity(c echo.Context) error {
        cblog.Info("call RegisterSecurity()")

        req := securityGroupRegisterReq{}

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // create UserIID
        userIId := cres.IID{req.ReqInfo.Name, req.ReqInfo.CSPId}

        // Call common-runtime API
        result, err := cmrt.RegisterSecurity(req.ConnectionName, req.ReqInfo.VPCName, userIId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func UnregisterSecurity(c echo.Context) error {
        cblog.Info("call UnregisterSecurity()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.UnregisterResource(req.ConnectionName, rsSG, c.Param("Name"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}

type securityGroupCreateReq struct {
	ConnectionName string
	ReqInfo        struct {
		Name          string
		VPCName       string
		Direction     string
		SecurityRules *[]cres.SecurityRuleInfo
	}
}

/* // createSecurity godoc
// @Summary Create Security Group
// @Description Create Security Group
// @Tags [CCM] Security Group management
// @Accept  json
// @Produce  json
// @Param securityGroupCreateReq body securityGroupCreateReq true "Request body to create Security Group"
// @Success 200 {object} resources.SecurityInfo
// @Failure 404 {object} SimpleMsg
// @Failure 500 {object} SimpleMsg
// @Router /securitygroup [post] */
func CreateSecurity(c echo.Context) error {
	cblog.Info("call CreateSecurity()")

	req := securityGroupCreateReq{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Rest RegInfo => Driver ReqInfo
	reqInfo := cres.SecurityReqInfo{
		//IId:           cres.IID{req.ReqInfo.VPCName + cm.SG_DELIMITER + req.ReqInfo.Name, ""},
		IId:           cres.IID{req.ReqInfo.Name, req.ReqInfo.Name}, // for NCP: fixed NameID => SystemID, Driver: (1)search systemID with fixed NameID (2)replace fixed NameID into SysemID
		VpcIID:        cres.IID{req.ReqInfo.VPCName, ""},
		// deprecated; Direction:     req.ReqInfo.Direction,
		SecurityRules: req.ReqInfo.SecurityRules,
	}

	// Call common-runtime API
	result, err := cmrt.CreateSecurity(req.ConnectionName, rsSG, reqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

func ListSecurity(c echo.Context) error {
	cblog.Info("call ListSecurity()")

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
	result, err := cmrt.ListSecurity(req.ConnectionName, rsSG)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result []*cres.SecurityInfo `json:"securitygroup"`
	}
	jsonResult.Result = result
	return c.JSON(http.StatusOK, &jsonResult)
}

// list all SGs for management
// (1) get args from REST Call
// (2) get all SG List by common-runtime API
// (3) return REST Json Format
func ListAllSecurity(c echo.Context) error {
	cblog.Info("call ListAllSecurity()")

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
	allResourceList, err := cmrt.ListAllResource(req.ConnectionName, rsSG)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &allResourceList)
}

func GetSecurity(c echo.Context) error {
	cblog.Info("call GetSecurity()")

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
	result, err := cmrt.GetSecurity(req.ConnectionName, rsSG, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func DeleteSecurity(c echo.Context) error {
	cblog.Info("call DeleteSecurity()")

	var req struct {
		ConnectionName string
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, _, err := cmrt.DeleteResource(req.ConnectionName, rsSG, c.Param("Name"), c.QueryParam("force"))
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
func DeleteCSPSecurity(c echo.Context) error {
	cblog.Info("call DeleteCSPSecurity()")

	var req struct {
		ConnectionName string
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, rsSG, c.Param("Id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

type ruleControlReq struct {
	ConnectionName string
	ReqInfo        struct {
		RuleInfoList []struct {
			Direction       string
			IPProtocol      string
			FromPort        string
			ToPort          string
			CIDR            string
		}
	}
}
// (1) get rules info from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func AddRules(c echo.Context) error {
        cblog.Info("call AddRules()")

        req := ruleControlReq{}

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Rest RegInfo => Driver ReqInfo
        // create RuleInfo List
        reqRuleInfoList := []cres.SecurityRuleInfo{}
        for _, info := range req.ReqInfo.RuleInfoList {
                ruleInfo := cres.SecurityRuleInfo{Direction: info.Direction,
			IPProtocol: info.IPProtocol, FromPort: info.FromPort, ToPort: info.ToPort, CIDR: info.CIDR}
                reqRuleInfoList = append(reqRuleInfoList, ruleInfo)
        }

        // Call common-runtime API
        result, err := cmrt.AddRules(req.ConnectionName, c.Param("SGName"), reqRuleInfoList)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

// (1) get rules info from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func RemoveRules(c echo.Context) error {
        cblog.Info("call RemoveRules()")

        req := ruleControlReq{}

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Rest RegInfo => Driver ReqInfo
        // create RuleInfo List
        reqRuleInfoList := []cres.SecurityRuleInfo{}
        for _, info := range req.ReqInfo.RuleInfoList {
                ruleInfo := cres.SecurityRuleInfo{Direction: info.Direction,
                        IPProtocol: info.IPProtocol, FromPort: info.FromPort, ToPort: info.ToPort, CIDR: info.CIDR}
                reqRuleInfoList = append(reqRuleInfoList, ruleInfo)
        }

        // Call common-runtime API
	// no force option
        result, err := cmrt.RemoveRules(req.ConnectionName, c.Param("SGName"), reqRuleInfoList)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}

