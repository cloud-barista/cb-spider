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

//================ VPC Handler

// VPCRegisterRequest represents the request body for registering a VPC.
type VPCRegisterRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		Name  string `json:"Name" validate:"required" example:"vpc-01"`
		CSPId string `json:"CSPId" validate:"required" example:"csp-vpc-1234"`
	} `json:"ReqInfo" validate:"required"`
}

// registerVPC godoc
// @ID register-vpc
// @Summary Register VPC
// @Description Register a new Virtual Private Cloud (VPC) with the specified name and CSP ID.
// @Tags [VPC Management]
// @Accept  json
// @Produce  json
// @Param VPCRegisterRequest body restruntime.VPCRegisterRequest true "Request body for registering a VPC"
// @Success 200 {object} cres.VPCInfo "Details of the registered VPC"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regvpc [post]
func RegisterVPC(c echo.Context) error {
	cblog.Info("call RegisterVPC()")

	req := VPCRegisterRequest{}

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

// SubnetRegisterRequest represents the request body for registering a subnet.
type SubnetRegisterRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		Name    string `json:"Name" validate:"required" example:"subnet-01"`
		Zone    string `json:"Zone,omitempty" validate:"omitempty" example:"us-east-1a"`
		VPCName string `json:"VPCName" validate:"required" example:"vpc-01"`
		CSPId   string `json:"CSPId" validate:"required" example:"csp-subnet-1234"`
	} `json:"ReqInfo" validate:"required"`
}

// registerSubnet godoc
// @ID register-subnet
// @Summary Register Subnet
// @Description Register a new Subnet within a specified VPC.
// @Tags [VPC Management]
// @Accept  json
// @Produce  json
// @Param SubnetRegisterRequest body restruntime.SubnetRegisterRequest true "Request body for registering a Subnet"
// @Success 200 {object} cres.VPCInfo "Details of the VPC including the registered Subnet"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regsubnet [post]
func RegisterSubnet(c echo.Context) error {
	cblog.Info("call RegisterSubnet()")

	req := SubnetRegisterRequest{}

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

// SubnetUnregisterRequest represents the request body for unregistering a subnet.
type SubnetUnregisterRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		VPCName string `json:"VPCName" validate:"required" example:"vpc-01"`
	} `json:"ReqInfo" validate:"required"`
}

// unregisterSubnet godoc
// @ID unregister-subnet
// @Summary Unregister Subnet
// @Description Unregister a Subnet from a specified VPC.
// @Tags [VPC Management]
// @Accept  json
// @Produce  json
// @Param SubnetUnregisterRequest body restruntime.SubnetUnregisterRequest true "Request body for unregistering a Subnet"
// @Param Name path string true "The name of the Subnet to unregister"
// @Success 200 {object} BooleanInfo "Result of the unregister operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regsubnet/{Name} [delete]
func UnregisterSubnet(c echo.Context) error {
	cblog.Info("call UnregisterSubnet()")

	req := SubnetUnregisterRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.UnregisterSubnet(req.ConnectionName, req.ReqInfo.VPCName, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// unregisterVPC godoc
// @ID unregister-vpc
// @Summary Unregister VPC
// @Description Unregister a VPC with the specified name.
// @Tags [VPC Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for unregistering a VPC"
// @Param Name path string true "The name of the VPC to unregister"
// @Success 200 {object} BooleanInfo "Result of the unregister operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regvpc/{Name} [delete]
func UnregisterVPC(c echo.Context) error {
	cblog.Info("call UnregisterVPC()")

	var req ConnectionRequest

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

// VPCCreateRequest represents the request body for creating a VPC.
type VPCCreateRequest struct {
	ConnectionName  string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	IDTransformMode string `json:"IDTransformMode,omitempty" validate:"omitempty" example:"ON"` // ON: transform CSP ID, OFF: no-transform CSP ID
	ReqInfo         struct {
		Name           string `json:"Name" validate:"required" example:"vpc-01"`
		IPv4_CIDR      string `json:"IPv4_CIDR" validate:"omitempty"` // Some CSPs unsupported VPC CIDR
		SubnetInfoList []struct {
			Name      string          `json:"Name" validate:"required" example:"subnet-01"`
			Zone      string          `json:"Zone,omitempty" validate:"omitempty" example:"us-east-1b"` // target zone for the subnet, if not specified, it will be created in the same zone as the Connection.
			IPv4_CIDR string          `json:"IPv4_CIDR" validate:"required" example:"10.0.8.0/22"`
			TagList   []cres.KeyValue `json:"TagList,omitempty" validate:"omitempty"`
		} `json:"SubnetInfoList" validate:"required"`
		TagList []cres.KeyValue `json:"TagList,omitempty" validate:"omitempty"`
	} `json:"ReqInfo" validate:"required"`
}

// createVPC godoc
// @ID create-vpc
// @Summary Create VPC
// @Description Create a new Virtual Private Cloud (VPC) with specified subnet configurations.
// @Tags [VPC Management]
// @Accept  json
// @Produce  json
// @Param VPCCreateRequest body restruntime.VPCCreateRequest true "Request body for creating a VPC"
// @Success 200 {object} cres.VPCInfo "Details of the created VPC"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vpc [post]
func CreateVPC(c echo.Context) error {
	cblog.Info("call CreateVPC()")

	req := VPCCreateRequest{}

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

type VPCListResponse struct {
	Result []*cres.VPCInfo `json:"vpc" validate:"required" description:"A list of VPC information"`
}

// listVPC godoc
// @ID list-vpc
// @Summary List VPCs
// @Description Retrieve a list of Virtual Private Clouds (VPCs) associated with a specific connection.
// @Tags [VPC Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list VPCs for"
// @Success 200 {object} VPCListResponse "List of VPCs"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vpc [get]
func ListVPC(c echo.Context) error {
	cblog.Info("call ListVPC()")

	var req ConnectionRequest

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

	jsonResult := VPCListResponse{
		Result: result,
	}

	return c.JSON(http.StatusOK, &jsonResult)
}

// listAllVPC godoc
// @ID list-all-vpc
// @Summary List All VPCs in a Connection
// @Description Retrieve a comprehensive list of all Virtual Private Clouds (VPCs) associated with a specific connection, <br> including those mapped between CB-Spider and the CSP, <br> only registered in CB-Spider's metadata, <br> and only existing in the CSP.
// @Tags [VPC Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list VPCs for"
// @Success 200 {object} AllResourceListResponse "List of all VPCs within the specified connection, including VPCs in CB-Spider only, CSP only, and mapped between both."
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /allvpc [get]
func ListAllVPC(c echo.Context) error {
	cblog.Info("call ListAllVPC()")

	var req ConnectionRequest

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

// SubnetAddRequest represents the request body for adding a subnet to a VPC.
type SubnetAddRequest struct {
	ConnectionName  string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	IDTransformMode string `json:"IDTransformMode,omitempty" validate:"omitempty" example:"ON"` // ON: transform CSP ID, OFF: no-transform CSP ID
	ReqInfo         struct {
		Name      string          `json:"Name" validate:"required" example:"subnet-01"`
		Zone      string          `json:"Zone,omitempty" validate:"omitempty" example:"us-east-1b"` // target zone for the subnet, if not specified, it will be created in the same zone as the Connection.
		IPv4_CIDR string          `json:"IPv4_CIDR" validate:"required" example:"10.0.12.0/22"`
		TagList   []cres.KeyValue `json:"TagList,omitempty" validate:"omitempty"`
	} `json:"ReqInfo" validate:"required"`
}

// addSubnet godoc
// @ID add-subnet
// @Summary Add Subnet
// @Description Add a new Subnet to an existing VPC.
// @Tags [VPC Management]
// @Accept  json
// @Produce  json
// @Param VPCName path string true "The name of the VPC to add the Subnet to"
// @Param SubnetAddRequest body restruntime.SubnetAddRequest true "Request body for adding a Subnet"
// @Success 200 {object} cres.VPCInfo "Details of the VPC including the added Subnet"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vpc/{VPCName}/subnet [post]
func AddSubnet(c echo.Context) error {
	cblog.Info("call AddSubnet()")

	var req SubnetAddRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Rest RegInfo => Driver ReqInfo
	reqSubnetInfo := cres.SubnetInfo{IId: cres.IID{req.ReqInfo.Name, ""}, IPv4_CIDR: req.ReqInfo.IPv4_CIDR, Zone: req.ReqInfo.Zone, TagList: req.ReqInfo.TagList}

	// Call common-runtime API
	result, err := cmrt.AddSubnet(req.ConnectionName, SUBNET, c.Param("VPCName"), reqSubnetInfo, req.IDTransformMode)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// removeSubnet godoc
// @ID remove-subnet
// @Summary Remove Subnet
// @Description Remove an existing Subnet from a VPC.
// @Tags [VPC Management]
// @Accept  json
// @Produce  json
// @Param VPCName path string true "The name of the VPC"
// @Param SubnetName path string true "The name of the Subnet to remove"
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for removing a Subnet"
// @Success 200 {object} BooleanInfo "Result of the remove operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vpc/{VPCName}/subnet/{SubnetName} [delete]
func RemoveSubnet(c echo.Context) error {
	cblog.Info("call RemoveSubnet()")

	var req ConnectionRequest

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

// removeCSPSubnet godoc
// @ID remove-csp-subnet
// @Summary Remove CSP Subnet
// @Description Remove an existing CSP Subnet from a VPC.
// @Tags [VPC Management]
// @Accept  json
// @Produce  json
// @Param VPCName path string true "The name of the VPC"
// @Param Id path string true "The CSP Subnet ID to remove"
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for removing a CSP Subnet"
// @Success 200 {object} BooleanInfo "Result of the remove operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vpc/{VPCName}/cspsubnet/{Id} [delete]
func RemoveCSPSubnet(c echo.Context) error {
	cblog.Info("call RemoveCSPSubnet()")

	var req ConnectionRequest

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

// getVPC godoc
// @ID get-vpc
// @Summary Get VPC
// @Description Retrieve details of a specific Virtual Private Cloud (VPC).
// @Tags [VPC Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to get a VPC for"
// @Param Name path string true "The name of the VPC to retrieve"
// @Success 200 {object} cres.VPCInfo "Details of the VPC"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vpc/{Name} [get]
func GetVPC(c echo.Context) error {
	cblog.Info("call GetVPC()")

	var req ConnectionRequest

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

// deleteVPC godoc
// @ID delete-vpc
// @Summary Delete VPC
// @Description Delete a specified Virtual Private Cloud (VPC).
// @Tags [VPC Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for deleting a VPC"
// @Param Name path string true "The name of the VPC to delete"
// @Param force query string false "Force delete the VPC"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vpc/{Name} [delete]
func DeleteVPC(c echo.Context) error {
	cblog.Info("call DeleteVPC()")

	var req ConnectionRequest

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

// deleteCSPVPC godoc
// @ID delete-csp-vpc
// @Summary Delete CSP VPC
// @Description Delete a specified CSP Virtual Private Cloud (VPC).
// @Tags [VPC Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for deleting a CSP VPC"
// @Param Id path string true "The CSP VPC ID to delete"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cspvpc/{Id} [delete]
func DeleteCSPVPC(c echo.Context) error {
	cblog.Info("call DeleteCSPVPC()")

	var req ConnectionRequest

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

// VPCGetSecurityGroupOwnerRequest represents the request body for retrieving the owner VPC of a Security Group.
type VPCGetSecurityGroupOwnerRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		CSPId string `json:"CSPId" validate:"required" example:"csp-sg-1234"`
	} `json:"ReqInfo" validate:"required"`
}

// getSGOwnerVPC godoc
// @ID get-sg-owner-vpc
// @Summary Get Security Group Owner VPC
// @Description Retrieve the owner VPC of a specified Security Group.
// @Tags [VPC Management]
// @Accept  json
// @Produce  json
// @Param VPCGetSecurityGroupOwnerRequest body restruntime.VPCGetSecurityGroupOwnerRequest true "Request body for getting Security Group Owner VPC"
// @Success 200 {object} cres.IID "Details of the owner VPC"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /getsecuritygroupowner [post]
func GetSGOwnerVPC(c echo.Context) error {
	cblog.Info("call GetSGOwnerVPC()")

	var req VPCGetSecurityGroupOwnerRequest

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

// countAllVPCs godoc
// @ID count-all-vpcs
// @Summary Count All VPCs
// @Description Get the total number of VPCs across all connections.
// @Tags [VPC Management]
// @Produce  json
// @Success 200 {object} CountResponse "Total count of VPCs"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countvpc [get]
func CountAllVPCs(c echo.Context) error {
	cblog.Info("call CountAllVPCs()")

	// Call common-runtime API to get count of VPCs
	count, err := cmrt.CountAllVPCs()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Prepare JSON result
	jsonResult := CountResponse{
		Count: int(count),
	}

	// Return JSON response
	return c.JSON(http.StatusOK, jsonResult)
}

// countVPCsByConnection godoc
// @ID count-vpcs-by-connection
// @Summary Count VPCs by Connection
// @Description Get the total number of VPCs for a specific connection.
// @Tags [VPC Management]
// @Produce  json
// @Param ConnectionName path string true "The name of the Connection"
// @Success 200 {object} CountResponse "Total count of VPCs for the connection"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countvpc/{ConnectionName} [get]
func CountVPCsByConnection(c echo.Context) error {
	cblog.Info("call CountVPCsByConnection()")

	// Call common-runtime API to get count of VPCs
	count, err := cmrt.CountVPCsByConnection(c.Param("ConnectionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Prepare JSON result
	jsonResult := CountResponse{
		Count: int(count),
	}

	// Return JSON response
	return c.JSON(http.StatusOK, jsonResult)
}

// countAllSubnets godoc
// @ID count-all-subnets
// @Summary Count All Subnets
// @Description Get the total number of Subnets across all connections.
// @Tags [VPC Management]
// @Produce  json
// @Success 200 {object} CountResponse "Total count of Subnets"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countsubnet [get]
func CountAllSubnets(c echo.Context) error {
	// Call common-runtime API to get count of Subnets
	count, err := cmrt.CountAllSubnets()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Prepare JSON result
	jsonResult := CountResponse{
		Count: int(count),
	}

	// Return JSON response
	return c.JSON(http.StatusOK, jsonResult)
}

// countSubnetsByConnection godoc
// @ID count-subnets-by-connection
// @Summary Count Subnets by Connection
// @Description Get the total number of Subnets for a specific connection.
// @Tags [VPC Management]
// @Produce  json
// @Param ConnectionName path string true "The name of the Connection"
// @Success 200 {object} CountResponse "Total count of Subnets for the connection"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countsubnet/{ConnectionName} [get]
func CountSubnetsByConnection(c echo.Context) error {
	// Call common-runtime API to get count of Subnets
	count, err := cmrt.CountSubnetsByConnection(c.Param("ConnectionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Prepare JSON result
	jsonResult := CountResponse{
		Count: int(count),
	}

	// Return JSON response
	return c.JSON(http.StatusOK, jsonResult)
}
