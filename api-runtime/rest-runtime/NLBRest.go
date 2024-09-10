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

// NLBGetOwnerVPCRequest represents the request body for retrieving the owner VPC of an NLB.
type NLBGetOwnerVPCRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		CSPId string `json:"CSPId" validate:"required" example:"csp-nlb-1234"`
	} `json:"ReqInfo" validate:"required"`
}

// registerNLB godoc
// @ID get-nlb-owner-vpc
// @Summary Get NLB Owner VPC
// @Description Retrieve the owner VPC of a specified Network Load Balancer (NLB).
// @Tags [NLB Management]
// @Accept  json
// @Produce  json
// @Param NLBGetOwnerVPCRequest body restruntime.NLBGetOwnerVPCRequest true "Request body for getting NLB Owner VPC"
// @Success 200 {object} cres.IID "Details of the owner VPC"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /getnlbowner [post]
func GetNLBOwnerVPC(c echo.Context) error {
	cblog.Info("call GetNLBOwnerVPC()")

	var req NLBGetOwnerVPCRequest

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

// NLBRegisterRequest represents the request body for registering an NLB.
type NLBRegisterRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		VPCName string `json:"VPCName" validate:"required" example:"vpc-01"`
		Name    string `json:"Name" validate:"required" example:"nlb-01"`
		CSPId   string `json:"CSPId" validate:"required" example:"csp-nlb-1234"`
	} `json:"ReqInfo" validate:"required"`
}

// registerNLB godoc
// @ID register-nlb
// @Summary Register NLB
// @Description Register a new Network Load Balancer (NLB) with the specified name and CSP ID.
// @Tags [NLB Management]
// @Accept  json
// @Produce  json
// @Param NLBRegisterRequest body restruntime.NLBRegisterRequest true "Request body for registering an NLB"
// @Success 200 {object} cres.NLBInfo "Details of the registered NLB"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regnlb [post]
func RegisterNLB(c echo.Context) error {
	cblog.Info("call RegisterNLB()")

	req := NLBRegisterRequest{}

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

// unregisterNLB godoc
// @ID unregister-nlb
// @Summary Unregister NLB
// @Description Unregister a Network Load Balancer (NLB) with the specified name.
// @Tags [NLB Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for unregistering an NLB"
// @Param Name path string true "The name of the NLB to unregister"
// @Success 200 {object} BooleanInfo "Result of the unregister operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regnlb/{Name} [delete]
func UnregisterNLB(c echo.Context) error {
	cblog.Info("call UnregisterNLB()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.UnregisterResource(req.ConnectionName, NLB, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// NLBCreateRequest represents the request body for creating an NLB.
type NLBCreateRequest struct {
	ConnectionName  string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	IDTransformMode string `json:"IDTransformMode,omitempty" validate:"omitempty" example:"ON"` // ON: transform CSP ID, OFF: no-transform CSP ID
	ReqInfo         struct {
		Name          string                   `json:"Name" validate:"required" example:"nlb-01"`
		VPCName       string                   `json:"VPCName" validate:"required" example:"vpc-01"`
		Type          string                   `json:"Type" validate:"required" example:"PUBLIC"`  // PUBLIC(V) | INTERNAL
		Scope         string                   `json:"Scope" validate:"required" example:"REGION"` // REGION(V) | GLOBAL
		Listener      NLBListenerCreateRequest `json:"Listener" validate:"required"`
		VMGroup       NLBVMGroupRequest        `json:"VMGroup" validate:"required"`
		HealthChecker NLBHealthCheckerRequest  `json:"HealthChecker" validate:"required"`
		TagList       []cres.KeyValue          `json:"TagList,omitempty" validate:"omitempty"`
	} `json:"ReqInfo" validate:"required"`
}

// NLBListenerCreateRequest represents the request body for the listener configuration in an NLB.
type NLBListenerCreateRequest struct {
	Protocol string `json:"Protocol" validate:"required" example:"TCP"` // TCP|UDP
	Port     string `json:"Port" validate:"required" example:"22"`      // 1-65535
}

// createNLB godoc
// @ID create-nlb
// @Summary Create NLB
// @Description Create a new Network Load Balancer (NLB) with specified configurations. 🕷️ [[Concept Guide](https://github.com/cloud-barista/cb-spider/wiki/Network-Load-Balancer-and-Driver-API)]
// @Tags [NLB Management]
// @Accept  json
// @Produce  json
// @Param NLBCreateRequest body restruntime.NLBCreateRequest true "Request body for creating an NLB"
// @Success 200 {object} cres.NLBInfo "Details of the created NLB"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /nlb [post]
func CreateNLB(c echo.Context) error {
	cblog.Info("call CreateNLB()")

	req := NLBCreateRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Rest RegInfo => Driver ReqInfo
	reqInfo := cres.NLBInfo{
		IId:      cres.IID{req.ReqInfo.Name, req.ReqInfo.Name},
		VpcIID:   cres.IID{req.ReqInfo.VPCName, ""},
		Type:     req.ReqInfo.Type,
		Scope:    req.ReqInfo.Scope,
		Listener: convertListenerInfo(req.ReqInfo.Listener),
		VMGroup:  convertVMGroupInfo(req.ReqInfo.VMGroup),
		TagList:  req.ReqInfo.TagList,
		//HealthChecker: below
	}
	healthChecker, err := convertHealthCheckerInfo(req.ReqInfo.HealthChecker)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	reqInfo.HealthChecker = healthChecker

	// Call common-runtime API
	result, err := cmrt.CreateNLB(req.ConnectionName, NLB, reqInfo, req.IDTransformMode)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// convertListenerInfo converts an NLBListenerCreateRequest to ListenerInfo.
func convertListenerInfo(listenerReq NLBListenerCreateRequest) cres.ListenerInfo {
	return cres.ListenerInfo{
		Protocol: listenerReq.Protocol,
		Port:     listenerReq.Port,
	}
}

// NLBVMGroupRequest represents the request body for VM group configurations in an NLB.
type NLBVMGroupRequest struct {
	Protocol string   `json:"Protocol" validate:"required" example:"TCP"` // TCP|UDP
	Port     string   `json:"Port" validate:"required" example:"22"`      // Listener Port or 1-65535
	VMs      []string `json:"VMs" validate:"required" example:"vm-01", "vm-02"`
}

func convertVMGroupInfo(vgInfo NLBVMGroupRequest) cres.VMGroupInfo {
	vmIIDList := []cres.IID{}
	for _, vm := range vgInfo.VMs {
		vmIIDList = append(vmIIDList, cres.IID{vm, ""})
	}
	return cres.VMGroupInfo{vgInfo.Protocol, vgInfo.Port, &vmIIDList, "", nil}
}

// NLBHealthCheckerRequest represents the request body for health checker configurations in an NLB.
type NLBHealthCheckerRequest struct {
	Protocol  string `json:"Protocol" validate:"required" example:"TCP"`                 // TCP|HTTP
	Port      string `json:"Port" validate:"required" example:"22"`                      // Listener Port or 1-65535
	Interval  string `json:"Interval,omitempty" validate:"omitempty" example:"default"`  // secs, if not specified, treated as "default", determined by CSP
	Timeout   string `json:"Timeout,omitempty" validate:"omitempty" example:"default"`   // secs, if not specified, treated as "default", determined by CSP
	Threshold string `json:"Threshold,omitempty" validate:"omitempty" example:"default"` // num, if not specified, treated as "default", determined by CSP
}

func convertHealthCheckerInfo(hcInfo NLBHealthCheckerRequest) (cres.HealthCheckerInfo, error) {
	// default: "default" or "" or "-1" => -1

	var err error
	// (1) Interval
	interval := -1
	strInterval := strings.ToLower(hcInfo.Interval)
	switch strInterval {
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
	switch strTimeout {
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
	switch strThreshold {
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

// NLBListResponse represents the response body for listing NLBs.
type NLBListResponse struct {
	Result []*cres.NLBInfo `json:"nlb" validate:"required" description:"A list of NLB information"`
}

// listNLB godoc
// @ID list-nlb
// @Summary List NLBs
// @Description Retrieve a list of Network Load Balancers (NLBs) associated with a specific connection.
// @Tags [NLB Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list NLBs for"
// @Success 200 {object} NLBListResponse "List of NLBs"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /nlb [get]
func ListNLB(c echo.Context) error {
	cblog.Info("call ListNLB()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.ListNLB(req.ConnectionName, NLB)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	jsonResult := NLBListResponse{
		Result: result,
	}
	return c.JSON(http.StatusOK, &jsonResult)
}

// listAllNLB godoc
// @ID list-all-nlb
// @Summary List All NLBs in a Connection
// @Description Retrieve a comprehensive list of all Network Load Balancers (NLBs) associated with a specific connection, <br> including those mapped between CB-Spider and the CSP, <br> only registered in CB-Spider's metadata, <br> and only existing in the CSP.
// @Tags [NLB Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list NLBs for"
// @Success 200 {object} AllResourceListResponse "List of all NLBs within the specified connection, including NLBs in CB-Spider only, CSP only, and mapped between both."
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /allnlb [get]
func ListAllNLB(c echo.Context) error {
	cblog.Info("call ListAllNLB()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	allResourceList, err := cmrt.ListAllResource(req.ConnectionName, NLB)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &allResourceList)
}

// getNLB godoc
// @ID get-nlb
// @Summary Get NLB
// @Description Retrieve details of a specific Network Load Balancer (NLB).
// @Tags [NLB Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to get an NLB for"
// @Param Name path string true "The name of the NLB to retrieve"
// @Success 200 {object} cres.NLBInfo "Details of the NLB"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /nlb/{Name} [get]
func GetNLB(c echo.Context) error {
	cblog.Info("call GetNLB()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.GetNLB(req.ConnectionName, NLB, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// NLBAddVMsRequest represents the request body for adding VMs to an NLB.
type NLBAddVMsRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		VMs []string `json:"VMs" validate:"required" example:"vm-01"`
	} `json:"ReqInfo" validate:"required"`
}

// addNLBVMs godoc
// @ID add-nlb-vms
// @Summary Add VMs to NLB
// @Description Add a new set of VMs to an existing Network Load Balancer (NLB).
// @Tags [NLB Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the NLB to add VMs to"
// @Param NLBAddVMsRequest body restruntime.NLBAddVMsRequest true "Request body for adding VMs to an NLB"
// @Success 200 {object} cres.NLBInfo "Details of the NLB including the added VMs"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /nlb/{Name}/vms [post]
func AddNLBVMs(c echo.Context) error {
	cblog.Info("call AddNLBVMs()")

	var req NLBAddVMsRequest

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

// NLBRemoveVMsRequest represents the request body for removing VMs from an NLB.
type NLBRemoveVMsRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		VMs []string `json:"VMs" validate:"required" example:"vm-01"`
	} `json:"ReqInfo" validate:"required"`
}

// removeNLBVMs godoc
// @ID remove-nlb-vms
// @Summary Remove VMs from NLB
// @Description Remove a set of VMs from an existing Network Load Balancer (NLB).
// @Tags [NLB Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the NLB to remove VMs from"
// @Param NLBRemoveVMsRequest body restruntime.NLBRemoveVMsRequest true "Request body for removing VMs from an NLB"
// @Success 200 {object} BooleanInfo "Result of the remove operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /nlb/{Name}/vms [delete]
func RemoveNLBVMs(c echo.Context) error {
	cblog.Info("call RemoveNLBVMs()")

	var req NLBRemoveVMsRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.RemoveNLBVMs(req.ConnectionName, c.Param("Name"), req.ReqInfo.VMs)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// NLBChangeListenerRequest represents the request body for changing the listener of an NLB.
type NLBChangeListenerRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		Protocol string `json:"Protocol" validate:"required" example:"TCP"`
		Port     string `json:"Port" validate:"required" example:"80"`
	} `json:"ReqInfo" validate:"required"`
}

/*
##################  @todo  To support or not will be decided later.
// changeListener godoc
// @ID change-listener
// @Summary Change NLB Listener
// @Description Change the Listener configuration of a specified Network Load Balancer (NLB).
// @Tags [NLB Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the NLB to change the listener for"
// @Param NLBChangeListenerRequest body restruntime.NLBChangeListenerRequest true "Request body for changing the Listener"
// @Success 200 {object} cres.NLBInfo "Details of the NLB including the changed Listener"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /nlb/{Name}/listener [put]
##################
*/
func ChangeListener(c echo.Context) error {
	cblog.Info("call ChangeListener()")

	var req NLBChangeListenerRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	reqInfo := cres.ListenerInfo{
		Protocol: req.ReqInfo.Protocol,
		Port:     req.ReqInfo.Port,
	}

	// Call common-runtime API
	result, err := cmrt.ChangeListener(req.ConnectionName, c.Param("Name"), reqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// NLBChangeVMGroupRequest represents the request body for changing the VM group of an NLB.
type NLBChangeVMGroupRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		Protocol string `json:"Protocol" validate:"required" example:"TCP"`
		Port     string `json:"Port" validate:"required" example:"80"`
	} `json:"ReqInfo" validate:"required"`
}

/*
##################  @todo  To support or not will be decided later.
// changeVMGroup godoc
// @ID change-vmgroup
// @Summary Change NLB VM Group
// @Description Change the VM Group configuration of a specified Network Load Balancer (NLB).
// @Tags [NLB Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the NLB to change the VM Group for"
// @Param NLBChangeVMGroupRequest body restruntime.NLBChangeVMGroupRequest true "Request body for changing the VM Group"
// @Success 200 {object} cres.NLBInfo "Details of the NLB including the changed VM Group"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /nlb/{Name}/vmgroup [put]
##################
*/
func ChangeVMGroup(c echo.Context) error {
	cblog.Info("call ChangeVMGroup()")

	var req NLBChangeVMGroupRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	reqInfo := cres.VMGroupInfo{
		Protocol: req.ReqInfo.Protocol,
		Port:     req.ReqInfo.Port,
	}

	// Call common-runtime API
	result, err := cmrt.ChangeVMGroup(req.ConnectionName, c.Param("Name"), reqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// NLBChangeHealthCheckerRequest represents the request body for changing the health checker of an NLB.
type NLBChangeHealthCheckerRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		Protocol  string `json:"Protocol" validate:"required" example:"TCP"`
		Port      string `json:"Port" validate:"required" example:"80"`
		Interval  string `json:"Interval" validate:"required" example:"30"`
		Timeout   string `json:"Timeout" validate:"required" example:"5"`
		Threshold string `json:"Threshold" validate:"required" example:"3"`
	} `json:"ReqInfo" validate:"required"`
}

/*
##################  @todo  To support or not will be decided later.
// changeHealthChecker godoc
// @ID change-healthchecker
// @Summary Change NLB Health Checker
// @Description Change the Health Checker configuration of a specified Network Load Balancer (NLB).
// @Tags [NLB Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the NLB to change the Health Checker for"
// @Param NLBChangeHealthCheckerRequest body restruntime.NLBChangeHealthCheckerRequest true "Request body for changing the Health Checker"
// @Success 200 {object} cres.NLBInfo "Details of the NLB including the changed Health Checker"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /nlb/{Name}/healthchecker [put]
##################
*/
func ChangeHealthChecker(c echo.Context) error {
	cblog.Info("call ChangeHealthChecker()")

	var req NLBChangeHealthCheckerRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	interval, err := strconv.Atoi(req.ReqInfo.Interval)
	timeout, err := strconv.Atoi(req.ReqInfo.Timeout)
	threshold, err := strconv.Atoi(req.ReqInfo.Threshold)

	reqInfo := cres.HealthCheckerInfo{
		Protocol:  req.ReqInfo.Protocol,
		Port:      req.ReqInfo.Port,
		Interval:  interval,
		Timeout:   timeout,
		Threshold: threshold,
	}

	// Call common-runtime API
	result, err := cmrt.ChangeHealthChecker(req.ConnectionName, c.Param("Name"), reqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// NLBGetVMGroupHealthInfoResponse represents the response body for retrieving the health information of a VM group in an NLB.
type NLBGetVMGroupHealthInfoResponse struct {
	Result cres.HealthInfo `json:"healthinfo" validate:"required" description:"Health information of the VM group"`
}

// getVMGroupHealthInfo godoc
// @ID get-vmgroup-healthinfo
// @Summary Get NLB VM Group Health Info
// @Description Retrieve the health information of the VM group in a specified Network Load Balancer (NLB).
// @Tags [NLB Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the NLB to get the VM Group Health Info for"
// @Param ConnectionName query string true "The name of the Connection"
// @Success 200 {object} NLBGetVMGroupHealthInfoResponse "Health information of the VM group"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /nlb/{Name}/health [get]
func GetVMGroupHealthInfo(c echo.Context) error {
	cblog.Info("call GetVMGroupHealthInfo()")

	var req ConnectionRequest

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

	jsonResult := NLBGetVMGroupHealthInfoResponse{
		Result: *result,
	}

	return c.JSON(http.StatusOK, &jsonResult)
}

// deleteNLB godoc
// @ID delete-nlb
// @Summary Delete NLB
// @Description Delete a specified Network Load Balancer (NLB).
// @Tags [NLB Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for deleting an NLB"
// @Param Name path string true "The name of the NLB to delete"
// @Param force query string false "Force delete the NLB. ex) true or false(default: false)"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /nlb/{Name} [delete]
func DeleteNLB(c echo.Context) error {
	cblog.Info("call DeleteNLB()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.DeleteNLB(req.ConnectionName, NLB, c.Param("Name"), c.QueryParam("force"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// deleteCSPNLB godoc
// @ID delete-csp-nlb
// @Summary Delete CSP NLB
// @Description Delete a specified CSP Network Load Balancer (NLB).
// @Tags [NLB Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for deleting a CSP NLB"
// @Param Id path string true "The CSP NLB ID to delete"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cspnlb/{Id} [delete]
func DeleteCSPNLB(c echo.Context) error {
	cblog.Info("call DeleteCSPNLB()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, NLB, c.Param("Id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// countAllNLBs godoc
// @ID count-all-nlbs
// @Summary Count All NLBs
// @Description Get the total number of Network Load Balancers (NLBs) across all connections.
// @Tags [NLB Management]
// @Produce  json
// @Success 200 {object} CountResponse "Total count of NLBs"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countnlb [get]
func CountAllNLBs(c echo.Context) error {
	// Call common-runtime API to get count of NLBs
	count, err := cmrt.CountAllNLBs()
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

// countNLBsByConnection godoc
// @ID count-nlbs-by-connection
// @Summary Count NLBs by Connection
// @Description Get the total number of Network Load Balancers (NLBs) for a specific connection.
// @Tags [NLB Management]
// @Produce  json
// @Param ConnectionName path string true "The name of the Connection"
// @Success 200 {object} CountResponse "Total count of NLBs for the connection"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countnlb/{ConnectionName} [get]
func CountNLBsByConnection(c echo.Context) error {
	// Call common-runtime API to get count of NLBs
	count, err := cmrt.CountNLBsByConnection(c.Param("ConnectionName"))
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
