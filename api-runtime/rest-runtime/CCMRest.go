// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.04.
// by CB-Spider Team, 2019.10.

package restruntime

import (

	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	// REST API (echo)
	"net/http"
	"net/url"

	"github.com/labstack/echo/v4"

	"strconv"
)

// define string of resource types
const (
	rsImage string = "image"
	rsVPC   string = "vpc"
	rsSubnet string = "subnet"	
	rsSG  string = "sg"
	rsKey string = "keypair"
	rsVM  string = "vm"
	rsNLB  string = "nlb"
	rsDisk  string = "disk"
)

//================ Image Handler
func CreateImage(c echo.Context) error {
	cblog.Info("call CreateImage()")

	var req struct {
		ConnectionName string
		ReqInfo        struct {
			Name string
		}
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	reqInfo := cres.ImageReqInfo{
		IId: cres.IID{req.ReqInfo.Name, ""},
	}

	// Call common-runtime API
	result, err := cmrt.CreateImage(req.ConnectionName, rsImage, reqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

func ListImage(c echo.Context) error {
	cblog.Info("call ListImage()")

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
	result, err := cmrt.ListImage(req.ConnectionName, rsImage)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result []*cres.ImageInfo `json:"image"`
	}

	jsonResult.Result = result
	return c.JSON(http.StatusOK, &jsonResult)
}

func GetImage(c echo.Context) error {
	cblog.Info("call GetImage()")

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
	encodededImageName := c.Param("Name")
	decodedImageName, err := url.QueryUnescape(encodededImageName)
	if err != nil {
		cblog.Fatal(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := cmrt.GetImage(req.ConnectionName, rsImage, decodedImageName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

func DeleteImage(c echo.Context) error {
	cblog.Info("call DeleteImage()")

	var req struct {
		ConnectionName string
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.DeleteImage(req.ConnectionName, rsImage, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

//================ VMSpec Handler
func ListVMSpec(c echo.Context) error {
	cblog.Info("call ListVMSpec()")

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
	result, err := cmrt.ListVMSpec(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result []*cres.VMSpecInfo `json:"vmspec"`
	}
	jsonResult.Result = result
	return c.JSON(http.StatusOK, &jsonResult)
}

func GetVMSpec(c echo.Context) error {
	cblog.Info("call GetVMSpec()")

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
	result, err := cmrt.GetVMSpec(req.ConnectionName, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

func ListOrgVMSpec(c echo.Context) error {
	cblog.Info("call ListOrgVMSpec()")

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
	result, err := cmrt.ListOrgVMSpec(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.String(http.StatusOK, result)
}

func GetOrgVMSpec(c echo.Context) error {
	cblog.Info("call GetOrgVMSpec()")

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
	result, err := cmrt.GetOrgVMSpec(req.ConnectionName, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.String(http.StatusOK, result)
}

type vpcRegisterReq struct {
	ConnectionName string
	ReqInfo        struct {
		Name           string
		CSPId          string
	}
}

func RegisterVPC(c echo.Context) error {
        cblog.Info("call RegisterVPC()")

        req := vpcRegisterReq{}

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // create UserIID
	userIId := cres.IID{req.ReqInfo.Name, req.ReqInfo.CSPId}

        // Call common-runtime API
        result, err := cmrt.RegisterVPC(req.ConnectionName, userIId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func UnregisterVPC(c echo.Context) error {
        cblog.Info("call UnregisterVPC()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.UnregisterResource(req.ConnectionName, rsVPC, c.Param("Name"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}

type vpcCreateReq struct {
        ConnectionName string
        ReqInfo        struct {
                Name           string
                IPv4_CIDR      string
                SubnetInfoList []struct {
                        Name      string
                        IPv4_CIDR string
                }
        }
}

// createVPC godoc
// @Summary Create VPC
// @Description Create VPC
// @Tags [CCM] VPC management
// @Accept  json
// @Produce  json
// @Param vpcCreateReq body vpcCreateReq true "Request body to create VPC"
// @Success 200 {object} resources.VPCInfo
// @Failure 404 {object} SimpleMsg
// @Failure 500 {object} SimpleMsg
// @Router /vpc [post]
func CreateVPC(c echo.Context) error {
	cblog.Info("call CreateVPC()")

	req := vpcCreateReq{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Rest RegInfo => Driver ReqInfo
	// (1) create SubnetInfo List
	subnetInfoList := []cres.SubnetInfo{}
	for _, info := range req.ReqInfo.SubnetInfoList {
		subnetInfo := cres.SubnetInfo{IId: cres.IID{info.Name, ""}, IPv4_CIDR: info.IPv4_CIDR}
		subnetInfoList = append(subnetInfoList, subnetInfo)
	}
	// (2) create VPCReqInfo with SubnetInfo List
	reqInfo := cres.VPCReqInfo{
		IId:            cres.IID{req.ReqInfo.Name, ""},
		IPv4_CIDR:      req.ReqInfo.IPv4_CIDR,
		SubnetInfoList: subnetInfoList,
	}

	// Call common-runtime API
	result, err := cmrt.CreateVPC(req.ConnectionName, rsVPC, reqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

func ListVPC(c echo.Context) error {
	cblog.Info("call ListVPC()")

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
	result, err := cmrt.ListVPC(req.ConnectionName, rsVPC)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result []*cres.VPCInfo `json:"vpc"`
	}
	jsonResult.Result = result

	return c.JSON(http.StatusOK, &jsonResult)
}

// list all VPCs for management
// (1) get args from REST Call
// (2) get all VPC List by common-runtime API
// (3) return REST Json Format
func ListAllVPC(c echo.Context) error {
	cblog.Info("call ListAllVPC()")

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
	allResourceList, err := cmrt.ListAllResource(req.ConnectionName, rsVPC)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &allResourceList)
}

func GetVPC(c echo.Context) error {
	cblog.Info("call GetVPC()")

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
	result, err := cmrt.GetVPC(req.ConnectionName, rsVPC, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func DeleteVPC(c echo.Context) error {
	cblog.Info("call DeleteVPC()")

	var req struct {
		ConnectionName string
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, _, err := cmrt.DeleteResource(req.ConnectionName, rsVPC, c.Param("Name"), c.QueryParam("force"))
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
func DeleteCSPVPC(c echo.Context) error {
	cblog.Info("call DeleteCSPVPC()")

	var req struct {
		ConnectionName string
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, rsVPC, c.Param("Id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// (1) get subnet info from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func AddSubnet(c echo.Context) error {
	cblog.Info("call AddSubnet()")

	var req struct {
		ConnectionName string
		ReqInfo        struct {
			Name      string
			IPv4_CIDR string
		}
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Rest RegInfo => Driver ReqInfo
	reqSubnetInfo := cres.SubnetInfo{IId: cres.IID{req.ReqInfo.Name, ""}, IPv4_CIDR: req.ReqInfo.IPv4_CIDR}

	// Call common-runtime API
	result, err := cmrt.AddSubnet(req.ConnectionName, rsSubnet, c.Param("VPCName"), reqSubnetInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func RemoveSubnet(c echo.Context) error {
	cblog.Info("call RemoveSubnet()")

	var req struct {
		ConnectionName string
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.RemoveSubnet(req.ConnectionName, c.Param("VPCName"), c.Param("SubnetName"), c.QueryParam("force"))
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
func RemoveCSPSubnet(c echo.Context) error {
	cblog.Info("call RemoveCSPSubnet()")

	var req struct {
		ConnectionName string
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.RemoveCSPSubnet(req.ConnectionName, c.Param("VPCName"), c.Param("Id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

func GetSGOwnerVPC(c echo.Context) error {
        cblog.Info("call GetSGOwnerVPC()")

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
        result, err := cmrt.GetSGOwnerVPC(req.ConnectionName, req.ReqInfo.CSPId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

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


type keyRegisterReq struct {
        ConnectionName string
        ReqInfo        struct {
                Name           string
                CSPId          string
        }
}

func RegisterKey(c echo.Context) error {
        cblog.Info("call RegisterKey()")

        req := keyRegisterReq{}

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // create UserIID
        userIId := cres.IID{req.ReqInfo.Name, req.ReqInfo.CSPId}

        // Call common-runtime API
        result, err := cmrt.RegisterKey(req.ConnectionName, userIId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func UnregisterKey(c echo.Context) error {
        cblog.Info("call UnregisterKey()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.UnregisterResource(req.ConnectionName, rsKey, c.Param("Name"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}


// type keyPairCreateReq struct {
// 	ConnectionName string
// 	ReqInfo        struct {
// 		Name string
// 	}
// }

// JSONResult's data field will be overridden by the specific type
type JSONResult struct {
	//Code    int          `json:"code" `
	//Message string       `json:"message"`
	//Data    interface{}  `json:"data"`
}

// createKey godoc
// @Summary Create SSH Key
// @Description Create SSH Key
// @Tags [CCM] Access key management
// @Accept  json
// @Produce  json
// @Param keyPairCreateReq body JSONResult{ConnectionName=string,ReqInfo=JSONResult{Name=string}} true "Request body to create key"
// @Success 200 {object} resources.KeyPairInfo
// @Failure 404 {object} SimpleMsg
// @Failure 500 {object} SimpleMsg
// @Router /keypair [post]
func CreateKey(c echo.Context) error {
	cblog.Info("call CreateKey()")

	var req struct {
		ConnectionName string
		ReqInfo        struct {
			Name string
		}
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Rest RegInfo => Driver ReqInfo
	reqInfo := cres.KeyPairReqInfo{
		IId: cres.IID{req.ReqInfo.Name, ""},
	}

	// Call common-runtime API
	result, err := cmrt.CreateKey(req.ConnectionName, rsKey, reqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

func ListKey(c echo.Context) error {
	cblog.Info("call ListKey()")

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
	result, err := cmrt.ListKey(req.ConnectionName, rsKey)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result []*cres.KeyPairInfo `json:"keypair"`
	}
	jsonResult.Result = result
	return c.JSON(http.StatusOK, &jsonResult)
}

// list all KeyPairs for management
// (1) get args from REST Call
// (2) get all KeyPair List by common-runtime API
// (3) return REST Json Format
func ListAllKey(c echo.Context) error {
	cblog.Info("call ListAllKey()")

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
	allResourceList, err := cmrt.ListAllResource(req.ConnectionName, rsKey)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &allResourceList)
}

func GetKey(c echo.Context) error {
	cblog.Info("call GetKey()")

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
	result, err := cmrt.GetKey(req.ConnectionName, rsKey, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func DeleteKey(c echo.Context) error {
	cblog.Info("call DeleteKey()")

	var req struct {
		ConnectionName string
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, _, err := cmrt.DeleteResource(req.ConnectionName, rsKey, c.Param("Name"), c.QueryParam("force"))
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
func DeleteCSPKey(c echo.Context) error {
	cblog.Info("call DeleteCSPKey()")

	var req struct {
		ConnectionName string
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, rsKey, c.Param("Id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

/****************************
//================ VNic Handler
func createVNic(c echo.Context) error {
	cblog.Info("call createVNic()")

        var req struct {
                ConnectionName string
                ReqInfo cres.VNicReqInfo
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateVNicHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	info, err := handler.CreateVNic(req.ReqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func listVNic(c echo.Context) error {
	cblog.Info("call listVNic()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateVNicHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	infoList, err := handler.ListVNic()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        var jsonResult struct {
                Result []*cres.VNicInfo `json:"vnic"`
        }
        if infoList == nil {
                infoList = []*cres.VNicInfo{}
        }
        jsonResult.Result = infoList
        return c.JSON(http.StatusOK, &jsonResult)
}

func getVNic(c echo.Context) error {
	cblog.Info("call getVNic()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateVNicHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	info, err := handler.GetVNic(c.Param("VNicId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func deleteVNic(c echo.Context) error {
	cblog.Info("call deleteVNic()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateVNicHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := handler.DeleteVNic(c.Param("VNicId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}

//================ PublicIP Handler
func createPublicIP(c echo.Context) error {
	cblog.Info("call createPublicIP()")

        var req struct {
                ConnectionName string
                ReqInfo cres.PublicIPReqInfo
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	info, err := handler.CreatePublicIP(req.ReqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func listPublicIP(c echo.Context) error {
	cblog.Info("call listPublicIP()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	infoList, err := handler.ListPublicIP()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        var jsonResult struct {
                Result []*cres.PublicIPInfo `json:"publicip"`
        }
        if infoList == nil {
                infoList = []*cres.PublicIPInfo{}
        }
        jsonResult.Result = infoList
        return c.JSON(http.StatusOK, &jsonResult)
}

func getPublicIP(c echo.Context) error {
	cblog.Info("call getPublicIP()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	info, err := handler.GetPublicIP(c.Param("PublicIPId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func deletePublicIP(c echo.Context) error {
	cblog.Info("call deletePublicIP()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := handler.DeletePublicIP(c.Param("PublicIPId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}
****************************/

//================ VM Handler


func GetVMUsingRS(c echo.Context) error {
        cblog.Info("call GetVMUsingRS()")

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
        result, err := cmrt.GetVMUsingRS(req.ConnectionName, req.ReqInfo.CSPId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

type vmRegisterReq struct {
        ConnectionName string
        ReqInfo        struct {
                Name           string
                CSPId          string
        }
}

func RegisterVM(c echo.Context) error {
        cblog.Info("call RegisterVM()")

        req := vmRegisterReq{}

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // create UserIID
        userIId := cres.IID{req.ReqInfo.Name, req.ReqInfo.CSPId}

        // Call common-runtime API
        result, err := cmrt.RegisterVM(req.ConnectionName, userIId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func UnregisterVM(c echo.Context) error {
        cblog.Info("call UnregisterVM()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.UnregisterResource(req.ConnectionName, rsVM, c.Param("Name"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}


func StartVM(c echo.Context) error {
	cblog.Info("call StartVM()")

	var req struct {
		ConnectionName string
		ReqInfo        struct {
			Name               string
			ImageName          string
			VPCName            string
			SubnetName         string
			SecurityGroupNames []string
			VMSpecName         string
			KeyPairName        string

			RootDiskType       string
			RootDiskSize       string

			DataDiskNames      []string

			VMUserId     string
			VMUserPasswd string
		}
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Rest RegInfo => Driver ReqInfo
	// (1) create SecurityGroup IID List
	sgIIDList := []cres.IID{}
	for _, sgName := range req.ReqInfo.SecurityGroupNames {
		// SG NameID format => {VPC NameID} + cm.SG_DELIMITER + {SG NameID}
		// transform: SG NameID => {VPC NameID}-{SG NameID}
		//sgIID := cres.IID{req.ReqInfo.VPCName + cm.SG_DELIMITER + sgName, ""}
		sgIID := cres.IID{sgName, ""}
		sgIIDList = append(sgIIDList, sgIID)
	}

	// (2) create DataDisk IID List
        diskIIDList := []cres.IID{}
        for _, diskName := range req.ReqInfo.DataDiskNames {
                diskIID := cres.IID{diskName, ""}
                diskIIDList = append(diskIIDList, diskIID)
        }	

	// (3) create VMReqInfo with SecurityGroup & diskIID IID List
	reqInfo := cres.VMReqInfo{
		IId:               cres.IID{req.ReqInfo.Name, ""},
		ImageIID:          cres.IID{req.ReqInfo.ImageName, ""},
		VpcIID:            cres.IID{req.ReqInfo.VPCName, ""},
		SubnetIID:         cres.IID{req.ReqInfo.SubnetName, ""},
		SecurityGroupIIDs: sgIIDList,

		VMSpecName: req.ReqInfo.VMSpecName,
		KeyPairIID: cres.IID{req.ReqInfo.KeyPairName, ""},

		RootDiskType: req.ReqInfo.RootDiskType,
		RootDiskSize: req.ReqInfo.RootDiskSize,

		DataDiskIIDs: diskIIDList,

		VMUserId:     req.ReqInfo.VMUserId,
		VMUserPasswd: req.ReqInfo.VMUserPasswd,
	}

	// Call common-runtime API
	result, err := cmrt.StartVM(req.ConnectionName, rsVM, reqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

func ListVM(c echo.Context) error {
	cblog.Info("call ListVM()")

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
	result, err := cmrt.ListVM(req.ConnectionName, rsVM)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result []*cres.VMInfo `json:"vm"`
	}
	jsonResult.Result = result

	return c.JSON(http.StatusOK, &jsonResult)
}

// list all VMs for management
// (1) get args from REST Call
// (2) get all VM List by common-runtime API
// (3) return REST Json Format
func ListAllVM(c echo.Context) error {
	cblog.Info("call ListAllVM()")

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
	allResourceList, err := cmrt.ListAllResource(req.ConnectionName, rsVM)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &allResourceList)
}

func GetVM(c echo.Context) error {
	cblog.Info("call GetVM()")

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
	result, err := cmrt.GetVM(req.ConnectionName, rsVM, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func TerminateVM(c echo.Context) error {
	cblog.Info("call TerminateVM()")

	var req struct {
		ConnectionName string
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	_, result, err := cmrt.DeleteResource(req.ConnectionName, rsVM, c.Param("Name"), c.QueryParam("force"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := StatusInfo{
		Status: string(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func TerminateCSPVM(c echo.Context) error {
	cblog.Info("call TerminateCSPVM()")

	var req struct {
		ConnectionName string
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	_, result, err := cmrt.DeleteCSPResource(req.ConnectionName, rsVM, c.Param("Id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := StatusInfo{
		Status: string(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

func ListVMStatus(c echo.Context) error {
	cblog.Info("call ListVMStatus()")

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
	result, err := cmrt.ListVMStatus(req.ConnectionName, rsVM)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result []*cres.VMStatusInfo `json:"vmstatus"`
	}
	jsonResult.Result = result

	return c.JSON(http.StatusOK, &jsonResult)
}

func GetVMStatus(c echo.Context) error {
	cblog.Info("call GetVMStatus()")

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
	result, err := cmrt.GetVMStatus(req.ConnectionName, rsVM, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := StatusInfo{
		Status: string(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

func ControlVM(c echo.Context) error {
	cblog.Info("call ControlVM()")

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
	result, err := cmrt.ControlVM(req.ConnectionName, rsVM, c.Param("Name"), c.QueryParam("action"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := StatusInfo{
		Status: string(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

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
                HealthChecker: 	convertHealthCheckerInfo(req.ReqInfo.HealthChecker),
        }

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

func convertHealthCheckerInfo(hcInfo HealthCheckerReq) cres.HealthCheckerInfo {
	interval, _ := strconv.Atoi(hcInfo.Interval)
	timeout, _ := strconv.Atoi(hcInfo.Timeout)
	threshold, _ := strconv.Atoi(hcInfo.Threshold)
        return cres.HealthCheckerInfo{hcInfo.Protocol, hcInfo.Port, interval, timeout, threshold, "", nil}
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

//================ Get All SPLock Infos

func GetAllSPLockInfo(c echo.Context) error {
        cblog.Info("call GetAllSPLockInfo()")

        infoList := cmrt.GetAllSPLockInfo()

        var jsonResult struct {
                Result []string `json:"splockinfo"`
        }
        if infoList == nil {
                infoList = []string{}
        }
        jsonResult.Result = infoList
        return c.JSON(http.StatusOK, &jsonResult)
}

//================ Get CSP Resource Name

func GetCSPResourceName(c echo.Context) error {
        cblog.Info("call GetCSPResourceName()")

        var req struct {
                ConnectionName string
                ResourceType string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // To support for Get-Query Param Type API
        if req.ConnectionName == "" {
                req.ConnectionName = c.QueryParam("ConnectionName")
        }
        if req.ResourceType == "" {
                req.ResourceType = c.QueryParam("ResourceType")
        }

        // Call common-runtime API
        result, err := cmrt.GetCSPResourceName(req.ConnectionName, req.ResourceType, c.Param("Name"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        var resultInfo struct {
                Name string
        }
	resultInfo.Name = string(result)

        return c.JSON(http.StatusOK, &resultInfo)
}

//================ Disk Handler

type DiskRegisterReq struct {
        ConnectionName string
        ReqInfo        struct {
                Name           string
                CSPId          string
        }
}

func RegisterDisk(c echo.Context) error {
        cblog.Info("call RegisterDisk()")

        req := DiskRegisterReq{}

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // create UserIID
        userIId := cres.IID{req.ReqInfo.Name, req.ReqInfo.CSPId}

        // Call common-runtime API
        result, err := cmrt.RegisterDisk(req.ConnectionName, userIId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func UnregisterDisk(c echo.Context) error {
        cblog.Info("call UnregisterDisk()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.UnregisterResource(req.ConnectionName, rsDisk, c.Param("Name"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}

type DiskReq struct {
        ConnectionName string
        ReqInfo        struct {
                Name            string

                DiskType        string
                DiskSize        string
        }
}

func CreateDisk(c echo.Context) error {
        cblog.Info("call CreateDisk()")

        req := DiskReq{}

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Rest RegInfo => Driver ReqInfo
        reqInfo := cres.DiskInfo{
                IId:           cres.IID{req.ReqInfo.Name, req.ReqInfo.Name},
                DiskType:           req.ReqInfo.DiskType,
                DiskSize:           req.ReqInfo.DiskSize,
        }

        // Call common-runtime API
        result, err := cmrt.CreateDisk(req.ConnectionName, rsDisk, reqInfo)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

func ListDisk(c echo.Context) error {
        cblog.Info("call ListDisk()")

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
        result, err := cmrt.ListDisk(req.ConnectionName, rsDisk)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        var jsonResult struct {
                Result []*cres.DiskInfo `json:"disk"`
        }
        jsonResult.Result = result
        return c.JSON(http.StatusOK, &jsonResult)
}

// list all Disks for management
// (1) get args from REST Call
// (2) get all Disk List by common-runtime API
// (3) return REST Json Format
func ListAllDisk(c echo.Context) error {
        cblog.Info("call ListAllDisk()")

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
        allResourceList, err := cmrt.ListAllResource(req.ConnectionName, rsDisk)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, &allResourceList)
}

func GetDisk(c echo.Context) error {
        cblog.Info("call GetDisk()")

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
        result, err := cmrt.GetDisk(req.ConnectionName, rsDisk, c.Param("Name"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

func ChangeDiskSize(c echo.Context) error {
        cblog.Info("call ChangeDiskSize()")

        var req struct {
                ConnectionName string
                ReqInfo        struct {
                        Size string
                }
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // To support for Get-Query Param Type API
        if req.ConnectionName == "" {
                req.ConnectionName = c.QueryParam("ConnectionName")
        }

        // Call common-runtime API
        result, err := cmrt.ChangeDiskSize(req.ConnectionName, c.Param("Name"), req.ReqInfo.Size)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func DeleteDisk(c echo.Context) error {
        cblog.Info("call DeleteDisk()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, _, err := cmrt.DeleteResource(req.ConnectionName, rsDisk, c.Param("Name"), c.QueryParam("force"))
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
func DeleteCSPDisk(c echo.Context) error {
        cblog.Info("call DeleteCSPDisk()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, rsDisk, c.Param("Id"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}

func AttachDisk(c echo.Context) error {
        cblog.Info("call AttachDisk()")

        var req struct {
                ConnectionName string
                ReqInfo        struct {
                        VMName string
                }
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.AttachDisk(req.ConnectionName, c.Param("Name"), req.ReqInfo.VMName)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

func DetachDisk(c echo.Context) error {
        cblog.Info("call DetachDisk()")

        var req struct {
                ConnectionName string
                ReqInfo        struct {
                        VMName string
                }
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.DetachDisk(req.ConnectionName, c.Param("Name"), req.ReqInfo.VMName)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}
