// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.

package restruntime

import (
	"encoding/json"
	"net/http"

	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

// ================ VMSpec Handler

// VMSpecListResponse represents the response body structure for the ListVMSpec API.
type VMSpecListResponse struct {
	Result []*cres.VMSpecInfo `json:"vmspec" validate:"required" description:"A list of VM specs"`
}

// OriginalVMSpecListResponse represents the dynamic structure for the Original VM Spec List response.
type OriginalVMSpecListResponse struct {
	VMSpecInfo map[string]interface{} `json:"VMSpecInfo" validate:"required"` // CSP-specific JSON format
}

// listVMSpec godoc
// @ID list-vm-spec
// @Summary List VM Specs
// @Description Retrieve a list of VM specs associated with a specific connection.
// @Tags [Cloud Metadata] VM Spec
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list VM specs for"
// @Success 200 {object} VMSpecListResponse "List of VM specs"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vmspec [get]
func ListVMSpec(c echo.Context) error {
	cblog.Info("call ListVMSpec()")

	req := ConnectionRequest{}

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

	jsonResult := VMSpecListResponse{
		Result: result,
	}
	return c.JSON(http.StatusOK, &jsonResult)
}

// getVMSpec godoc
// @ID get-vm-spec
// @Summary Get VM Spec
// @Description Retrieve details of a specific VM spec.
// @Tags [Cloud Metadata] VM Spec
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to get a specific VM spec for"
// @Param Name path string true "The name of the VM spec to retrieve"
// @Success 200 {object} cres.VMSpecInfo "Details of the VM spec"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vmspec/{Name} [get]
func GetVMSpec(c echo.Context) error {
	cblog.Info("call GetVMSpec()")

	req := ConnectionRequest{}

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

// listOrgVMSpec godoc
// @ID list-org-vm-spec
// @Summary List Original VM Specs
// @Description Retrieve a list of Original VM Specs associated with a specific connection. <br> The response structure may vary depending on the request ConnectionName.
// @Tags [Cloud Metadata] VM Spec
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list Original VM specs for"
// @Success 200 {object} OriginalVMSpecListResponse "Dynamic JSON structure representing the list of Original VM Specs"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vmorgspec [get]
func ListOrgVMSpec(c echo.Context) error {
	cblog.Info("call ListOrgVMSpec()")

	req := ConnectionRequest{}

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

	var resultInterface interface{}
	if err := json.Unmarshal([]byte(result), &resultInterface); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to parse result")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"VMSpecInfo": resultInterface})
}

// getOrgVMSpec godoc
// @ID get-org-vm-spec
// @Summary Get Original VM Spec
// @Description Retrieve details of a specific Original VM Spec.
// @Tags [Cloud Metadata] VM Spec
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to get a specific Original VM spec for"
// @Param Name path string true "The name of the VM spec to retrieve"
// @Success 200 {object} OriginalVMSpecListResponse "Details of the Original VM Spec"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vmorgspec/{Name} [get]
func GetOrgVMSpec(c echo.Context) error {
	cblog.Info("call GetOrgVMSpec()")

	req := ConnectionRequest{}

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

	var resultInterface interface{}
	if err := json.Unmarshal([]byte(result), &resultInterface); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to parse result")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"VMSpecInfo": resultInterface})
}
