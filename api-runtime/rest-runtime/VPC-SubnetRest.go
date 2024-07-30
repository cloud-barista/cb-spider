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

	dri "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"strconv"
)

//================ VPC Handler

type vpcRegisterReq struct {
	ConnectionName string
	ReqInfo        struct {
		Name  string
		CSPId string
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

type subnetRegisterReq struct {
	ConnectionName string
	ReqInfo        struct {
		Name    string
		Zone    string
		VPCName string
		CSPId   string
	}
}

func RegisterSubnet(c echo.Context) error {
	cblog.Info("call RegisterSubnet()")

	req := subnetRegisterReq{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// create UserIID
	userIId := cres.IID{NameId: req.ReqInfo.Name, SystemId: req.ReqInfo.CSPId}

	// Call common-runtime API
	result, err := cmrt.RegisterSubnet(req.ConnectionName, req.ReqInfo.Zone, req.ReqInfo.VPCName, userIId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

type subnetUnregisterReq struct {
	ConnectionName string
	ReqInfo        struct {
		VPCName string
	}
}

func UnregisterSubnet(c echo.Context) error {
	cblog.Info("call UnregisterSubnet()")

	req := subnetUnregisterReq{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.UnregisterSubnet(req.ConnectionName, req.ReqInfo.VPCName, c.Param("Name"))
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
	result, err := cmrt.UnregisterResource(req.ConnectionName, VPC, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

type vpcCreateReq struct {
	ConnectionName  string
	IDTransformMode string // ON | OFF, default is ON
	ReqInfo         struct {
		Name           string
		IPv4_CIDR      string
		SubnetInfoList []struct {
			Name      string
			Zone      string
			IPv4_CIDR string
			TagList   []dri.KeyValue
		}
		TagList []dri.KeyValue
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
		subnetInfo := cres.SubnetInfo{IId: cres.IID{info.Name, ""}, IPv4_CIDR: info.IPv4_CIDR, Zone: info.Zone, TagList: info.TagList}
		subnetInfoList = append(subnetInfoList, subnetInfo)
	}
	// (2) create VPCReqInfo with SubnetInfo List
	reqInfo := cres.VPCReqInfo{
		IId:            cres.IID{req.ReqInfo.Name, ""},
		IPv4_CIDR:      req.ReqInfo.IPv4_CIDR,
		SubnetInfoList: subnetInfoList,
		TagList:        req.ReqInfo.TagList,
	}

	// Call common-runtime API
	result, err := cmrt.CreateVPC(req.ConnectionName, VPC, reqInfo, req.IDTransformMode)
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
	result, err := cmrt.ListVPC(req.ConnectionName, VPC)
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
	allResourceList, err := cmrt.ListAllResource(req.ConnectionName, VPC)
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
	result, err := cmrt.GetVPC(req.ConnectionName, VPC, c.Param("Name"))
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
	result, err := cmrt.DeleteVPC(req.ConnectionName, VPC, c.Param("Name"), c.QueryParam("force"))
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
	result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, VPC, c.Param("Id"))
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
		ConnectionName  string
		IDTransformMode string // ON | OFF, default is ON
		ReqInfo         struct {
			Name      string
			Zone      string
			IPv4_CIDR string
		}
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Rest RegInfo => Driver ReqInfo
	reqSubnetInfo := cres.SubnetInfo{IId: cres.IID{req.ReqInfo.Name, ""}, IPv4_CIDR: req.ReqInfo.IPv4_CIDR, Zone: req.ReqInfo.Zone}

	// Call common-runtime API
	result, err := cmrt.AddSubnet(req.ConnectionName, SUBNET, c.Param("VPCName"), reqSubnetInfo, req.IDTransformMode)
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
			CSPId string
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

func CountAllVPCs(c echo.Context) error {
	cblog.Info("call CountAllVPCs()")

	// Call common-runtime API to get count of VPCs
	count, err := cmrt.CountAllVPCs()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Prepare JSON result
	var jsonResult struct {
		Count int `json:"count"`
	}
	jsonResult.Count = int(count)

	// Return JSON response
	return c.JSON(http.StatusOK, jsonResult)
}

func CountVPCsByConnection(c echo.Context) error {
	// Call common-runtime API to get count of VPCs
	count, err := cmrt.CountVPCsByConnection(c.Param("ConnectionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Prepare JSON result
	var jsonResult struct {
		Count int `json:"count"`
	}
	jsonResult.Count = int(count)

	// Return JSON response
	return c.JSON(http.StatusOK, jsonResult)
}

func CountAllSubnets(c echo.Context) error {
	// Call common-runtime API to get count of Subnets
	count, err := cmrt.CountAllSubnets()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Prepare JSON result
	var jsonResult struct {
		Count int `json:"count"`
	}
	jsonResult.Count = int(count)

	// Return JSON response
	return c.JSON(http.StatusOK, jsonResult)
}

func CountSubnetsByConnection(c echo.Context) error {
	// Call common-runtime API to get count of Subnets
	count, err := cmrt.CountSubnetsByConnection(c.Param("ConnectionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Prepare JSON result
	var jsonResult struct {
		Count int `json:"count"`
	}
	jsonResult.Count = int(count)

	// Return JSON response
	return c.JSON(http.StatusOK, jsonResult)
}
