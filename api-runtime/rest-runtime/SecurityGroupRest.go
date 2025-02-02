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
	dri "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	// REST API (echo)
	"net/http"

	"github.com/labstack/echo/v4"

	"strconv"
)

//================ SecurityGroup Handler

// SecurityGroupRegisterRequest represents the request body for registering a SecurityGroup.
type SecurityGroupRegisterRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		VPCName string `json:"VPCName" validate:"required" example:"vpc-01"`
		Name    string `json:"Name" validate:"required" example:"sg-01"`
		CSPId   string `json:"CSPId" validate:"required" example:"csp-sg-1234"`
	} `json:"ReqInfo" validate:"required"`
}

// registerSecurity godoc
// @ID register-securitygroup
// @Summary Register SecurityGroup
// @Description Register a new Security Group with the specified name and CSP ID.
// @Tags [SecurityGroup Management]
// @Accept  json
// @Produce  json
// @Param SecurityGroupRegisterRequest body restruntime.SecurityGroupRegisterRequest true "Request body for registering a SecurityGroup"
// @Success 200 {object} cres.SecurityInfo "Details of the registered SecurityGroup"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regsecuritygroup [post]
func RegisterSecurity(c echo.Context) error {
	cblog.Info("call RegisterSecurity()")

	req := SecurityGroupRegisterRequest{}

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

// unregisterSecurity godoc
// @ID unregister-securitygroup
// @Summary Unregister SecurityGroup
// @Description Unregister a Security Group with the specified name.
// @Tags [SecurityGroup Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for unregistering a SecurityGroup"
// @Param Name path string true "The name of the SecurityGroup to unregister"
// @Success 200 {object} BooleanInfo "Result of the unregister operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regsecuritygroup/{Name} [delete]
func UnregisterSecurity(c echo.Context) error {
	cblog.Info("call UnregisterSecurity()")

	req := ConnectionRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.UnregisterResource(req.ConnectionName, SG, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// SecurityGroupCreateRequest represents the request body for creating a SecurityGroup.
type SecurityGroupCreateRequest struct {
	ConnectionName  string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	IDTransformMode string `json:"IDTransformMode,omitempty" validate:"omitempty" example:"ON"` // ON: transform CSP ID, OFF: no-transform CSP ID
	ReqInfo         struct {
		Name          string                   `json:"Name" validate:"required" example:"sg-01"`
		VPCName       string                   `json:"VPCName" validate:"required" example:"vpc-01"`
		SecurityRules *[]cres.SecurityRuleInfo `json:"SecurityRules" validate:"required"`
		TagList       []dri.KeyValue           `json:"TagList,omitempty" validate:"omitempty"`
	} `json:"ReqInfo" validate:"required"`
}

// createSecurity godoc
// @ID create-securitygroup
// @Summary Create SecurityGroup
// @Description Create a new Security Group with specified rules and tags. 🕷️ [[Concept Guide](https://github.com/cloud-barista/cb-spider/wiki/Security-Group-Rules-and-Driver-API)], 🕷️ [[User Guide](https://github.com/cloud-barista/cb-spider/wiki/features-and-usages#4-securitygroup-%EC%83%9D%EC%84%B1-%EB%B0%8F-%EC%A0%9C%EC%96%B4)]
// @Tags [SecurityGroup Management]
// @Accept  json
// @Produce  json
// @Param SecurityGroupCreateRequest body restruntime.SecurityGroupCreateRequest true "Request body for creating a SecurityGroup"
// @Success 200 {object} cres.SecurityInfo "Details of the created SecurityGroup"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /securitygroup [post]
func CreateSecurity(c echo.Context) error {
	cblog.Info("call CreateSecurity()")

	req := SecurityGroupCreateRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Rest RegInfo => Driver ReqInfo
	reqInfo := cres.SecurityReqInfo{
		IId:           cres.IID{req.ReqInfo.Name, req.ReqInfo.Name},
		VpcIID:        cres.IID{req.ReqInfo.VPCName, ""},
		SecurityRules: req.ReqInfo.SecurityRules,
		TagList:       req.ReqInfo.TagList,
	}

	// Call common-runtime API
	result, err := cmrt.CreateSecurity(req.ConnectionName, SG, reqInfo, req.IDTransformMode)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// SecurityGroupListResponse represents the response body for listing SecurityGroups.
type SecurityGroupListResponse struct {
	Result []*cres.SecurityInfo `json:"securitygroup" validate:"required" description:"A list of security group information"`
}

// listSecurity godoc
// @ID list-securitygroup
// @Summary List SecurityGroups
// @Description Retrieve a list of Security Groups associated with a specific connection.
// @Tags [SecurityGroup Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list SecurityGroups for"
// @Success 200 {object} restruntime.SecurityGroupListResponse "List of SecurityGroups"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /securitygroup [get]
func ListSecurity(c echo.Context) error {
	cblog.Info("call ListSecurity()")

	req := ConnectionRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	result, err := cmrt.ListSecurity(req.ConnectionName, SG)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	jsonResult := SecurityGroupListResponse{
		Result: result,
	}

	return c.JSON(http.StatusOK, &jsonResult)
}

// listAllSecurityGroups godoc
// @ID list-all-securitygroup
// @Summary List All Security Groups in a Connection
// @Description Retrieve a comprehensive list of all Security Groups associated with a specific connection, <br> including those mapped between CB-Spider and the CSP, <br> only registered in CB-Spider's metadata, <br> and only existing in the CSP.
// @Tags [SecurityGroup Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list Security Groups for"
// @Success 200 {object} AllResourceListResponse "List of all Security Groups within the specified connection, including Security Groups in CB-Spider only, CSP only, and mapped between both."
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /allsecuritygroup [get]
func ListAllSecurity(c echo.Context) error {
	cblog.Info("call ListAllSecurity()")

	req := ConnectionRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	allResourceList, err := cmrt.ListAllResource(req.ConnectionName, SG)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &allResourceList)
}

// listAllSecurityGroupInfo godoc
// @ID list-all-securitygroup-info
// @Summary List All SecurityGroup Info
// @Description Retrieve a list of Security Group information associated with all connections.
// @Tags [SecurityGroup Management]
// @Accept  json
// @Produce  json
// @Success 200 {object} AllResourceInfoListResponse "List of all Security Group information"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /allsecuritygroupinfo [get]
func ListAllSecurityGroupInfo(c echo.Context) error { return listAllResourceInfo(c, cres.SG) }

// getSecurity godoc
// @ID get-securitygroup
// @Summary Get SecurityGroup
// @Description Retrieve details of a specific Security Group.
// @Tags [SecurityGroup Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to get a SecurityGroup for"
// @Param Name path string true "The name of the SecurityGroup to retrieve"
// @Success 200 {object} cres.SecurityInfo "Details of the SecurityGroup"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /securitygroup/{Name} [get]
func GetSecurity(c echo.Context) error {
	cblog.Info("call GetSecurity()")

	req := ConnectionRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	result, err := cmrt.GetSecurity(req.ConnectionName, SG, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// deleteSecurity godoc
// @ID delete-securitygroup
// @Summary Delete SecurityGroup
// @Description Delete a specified Security Group.
// @Tags [SecurityGroup Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for deleting a SecurityGroup"
// @Param Name path string true "The name of the SecurityGroup to delete"
// @Param force query string false "Force delete the SecurityGroup. ex) true or false(default: false)"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /securitygroup/{Name} [delete]
func DeleteSecurity(c echo.Context) error {
	cblog.Info("call DeleteSecurity()")

	req := ConnectionRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := cmrt.DeleteSecurity(req.ConnectionName, SG, c.Param("Name"), c.QueryParam("force"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// deleteCSPSecurity godoc
// @ID delete-csp-securitygroup
// @Summary Delete CSP SecurityGroup
// @Description Delete a specified CSP Security Group.
// @Tags [SecurityGroup Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for deleting a CSP SecurityGroup"
// @Param Id path string true "The CSP SecurityGroup ID to delete"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cspsecuritygroup/{Id} [delete]
func DeleteCSPSecurity(c echo.Context) error {
	cblog.Info("call DeleteCSPSecurity()")

	req := ConnectionRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, SG, c.Param("Id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// RuleControlRequest represents the request body for controlling rules in a SecurityGroup.
type RuleControlRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		RuleInfoList []struct {
			Direction  string `json:"Direction" validate:"required" example:"inbound"`
			IPProtocol string `json:"IPProtocol" validate:"required" example:"TCP"`
			FromPort   string `json:"FromPort" validate:"required" example:"22"`
			ToPort     string `json:"ToPort" validate:"required" example:"22"`
			CIDR       string `json:"CIDR,omitempty" validate:"omitempty" example:"0.0.0.0/0(default)"`
		} `json:"RuleInfoList" validate:"required"`
	} `json:"ReqInfo" validate:"required"`
}

// addRules godoc
// @ID add-rule
// @Summary Add Rules to SecurityGroup
// @Description Add new rules to a Security Group.
// @Tags [SecurityGroup Management]
// @Accept  json
// @Produce  json
// @Param SGName path string true "The name of the SecurityGroup to add rules to"
// @Param RuleControlRequest body restruntime.RuleControlRequest true "Request body for adding rules"
// @Success 200 {object} cres.SecurityInfo "Details of the SecurityGroup after adding rules"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /securitygroup/{SGName}/rules [post]
func AddRules(c echo.Context) error {
	cblog.Info("call AddRules()")

	req := RuleControlRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	reqRuleInfoList := []cres.SecurityRuleInfo{}
	for _, info := range req.ReqInfo.RuleInfoList {
		ruleInfo := cres.SecurityRuleInfo{
			Direction:  info.Direction,
			IPProtocol: info.IPProtocol,
			FromPort:   info.FromPort,
			ToPort:     info.ToPort,
			CIDR:       info.CIDR,
		}
		reqRuleInfoList = append(reqRuleInfoList, ruleInfo)
	}

	result, err := cmrt.AddRules(req.ConnectionName, c.Param("SGName"), reqRuleInfoList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// removeRules godoc
// @ID remove-rule
// @Summary Remove Rules from SecurityGroup
// @Description Remove existing rules from a Security Group.
// @Tags [SecurityGroup Management]
// @Accept  json
// @Produce  json
// @Param SGName path string true "The name of the SecurityGroup to remove rules from"
// @Param RuleControlRequest body restruntime.RuleControlRequest true "Request body for removing rules"
// @Success 200 {object} BooleanInfo "Result of the remove operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /securitygroup/{SGName}/rules [delete]
func RemoveRules(c echo.Context) error {
	cblog.Info("call RemoveRules()")

	req := RuleControlRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	reqRuleInfoList := []cres.SecurityRuleInfo{}
	for _, info := range req.ReqInfo.RuleInfoList {
		ruleInfo := cres.SecurityRuleInfo{
			Direction:  info.Direction,
			IPProtocol: info.IPProtocol,
			FromPort:   info.FromPort,
			ToPort:     info.ToPort,
			CIDR:       info.CIDR,
		}
		reqRuleInfoList = append(reqRuleInfoList, ruleInfo)
	}

	result, err := cmrt.RemoveRules(req.ConnectionName, c.Param("SGName"), reqRuleInfoList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// countAllSecurityGroups godoc
// @ID count-all-securitygroup
// @Summary Count All SecurityGroups
// @Description Get the total number of Security Groups across all connections.
// @Tags [SecurityGroup Management]
// @Produce  json
// @Success 200 {object} CountResponse "Total count of SecurityGroups"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countsecuritygroup [get]
func CountAllSecurityGroups(c echo.Context) error {
	cblog.Info("call CountAllSecurityGroups()")

	count, err := cmrt.CountAllSecurityGroups()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	jsonResult := CountResponse{
		Count: int(count),
	}

	return c.JSON(http.StatusOK, jsonResult)
}

// countSecurityGroupsByConnection godoc
// @ID count-securitygroup-by-connection
// @Summary Count SecurityGroups by Connection
// @Description Get the total number of Security Groups for a specific connection.
// @Tags [SecurityGroup Management]
// @Produce  json
// @Param ConnectionName path string true "The name of the Connection"
// @Success 200 {object} CountResponse "Total count of SecurityGroups for the connection"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countsecuritygroup/{ConnectionName} [get]
func CountSecurityGroupsByConnection(c echo.Context) error {
	cblog.Info("call CountSecurityGroupsByConnection()")

	count, err := cmrt.CountSecurityGroupsByConnection(c.Param("ConnectionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	jsonResult := CountResponse{
		Count: int(count),
	}

	return c.JSON(http.StatusOK, jsonResult)
}
