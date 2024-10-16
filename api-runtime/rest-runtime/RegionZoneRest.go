// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2023.09.

package restruntime

import (
	"encoding/json"
	"net/http"

	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

// ================ RegionZone Handler

// RegionZoneListResponse represents the response body structure for the ListRegionZone API.
type RegionZoneListResponse struct {
	Result []*cres.RegionZoneInfo `json:"regionzone" validate:"required" description:"A list of region zones"`
}

// listRegionZone godoc
// @ID list-region-zone
// @Summary List Region Zones
// @Description Retrieve a list of Region Zones associated with a specific connection. 🕷️ [[User Guide](https://github.com/cloud-barista/cb-spider/wiki/REST-API-Region-Zone-Information-Guide)]
// @Tags [Cloud Metadata] Region/Zone
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list Region and Zones for"
// @Success 200 {object} RegionZoneListResponse "List of Region Zones"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regionzone [get]
func ListRegionZone(c echo.Context) error {
	cblog.Info("call ListRegionZone()")

	req := ConnectionRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.ListRegionZone(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	jsonResult := RegionZoneListResponse{
		Result: result,
	}
	return c.JSON(http.StatusOK, &jsonResult)
}

// getRegionZone godoc
// @ID get-region-zone
// @Summary Get Region Zone
// @Description Retrieve details of a specific Region Zone. 🕷️ [[User Guide](https://github.com/cloud-barista/cb-spider/wiki/REST-API-Region-Zone-Information-Guide)]
// @Tags [Cloud Metadata] Region/Zone
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to get a specific Region and Zones for"
// @Param Name path string true "The name of the Region to retrieve"
// @Success 200 {object} cres.RegionZoneInfo "Details of the Region Zone"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regionzone/{Name} [get]
func GetRegionZone(c echo.Context) error {
	cblog.Info("call GetRegionZone()")

	req := ConnectionRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.GetRegionZone(req.ConnectionName, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// OriginalRegionListResponse represents the dynamic structure for the Original Region List response.
type OriginalRegionListResponse struct {
	RegionInfo map[string]interface{} `json:"RegionInfo" validate:"required"` // CSP-specific JSON format
}

// listOrgRegion godoc
// @ID list-org-region
// @Summary List Original Regions
// @Description Retrieve a list of Original Regions associated with a specific connection. <br> The response structure may vary depending on the request ConnectionName.
// @Tags [Cloud Metadata] Region/Zone
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list Original regions for"
// @Success 200 {object} OriginalRegionListResponse "Dynamic JSON structure representing the list of Original Regions"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /orgregion [get]
func ListOrgRegion(c echo.Context) error {
	cblog.Info("call ListOrgRegion()")

	req := ConnectionRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.ListOrgRegion(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var resultInterface interface{}
	if err := json.Unmarshal([]byte(result), &resultInterface); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to parse result")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"RegionInfo": resultInterface})
}

// OriginalZoneListResponse represents the dynamic structure for the Original Zone List response.
type OriginalZoneListResponse struct {
	ZoneInfo map[string]interface{} `json:"ZoneInfo" validate:"required"` // CSP-specific JSON format
}

// listOrgZone godoc
// @ID list-org-zone
// @Summary List Original Zones
// @Description Retrieve a list of Original Zones associated with a specific connection. <br> The response structure may vary depending on the request ConnectionName.
// @Tags [Cloud Metadata] Region/Zone
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list Original zones for"
// @Success 200 {object} OriginalZoneListResponse "Dynamic JSON structure representing the list of Original Zones"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /orgzone [get]
func ListOrgZone(c echo.Context) error {
	cblog.Info("call ListOrgZone()")

	req := ConnectionRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.ListOrgZone(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var resultInterface interface{}
	if err := json.Unmarshal([]byte(result), &resultInterface); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to parse result")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"ZoneInfo": resultInterface})
}

// ================ RegionZone Handler (Pre-Config Version)
// PreConfigRegionZoneListRequest represents the request body for listing region zones with pre-config.
type PreConfigRegionZoneListRequest struct {
	DriverName     string `json:"DriverName" validate:"required" query:"DriverName"`         //  example:"aws-driver"
	CredentialName string `json:"CredentialName" validate:"required" query:"CredentialName"` // example:"aws-credential"
}

// listRegionZonePreConfig godoc
// @ID list-region-zone-preconfig
// @Summary List Pre-configured Region Zones
// @Description Retrieve a list of pre-configured Region Zones based on driver and credential names. 🕷️ [[User Guide](https://github.com/cloud-barista/cb-spider/wiki/REST-API-Region-Zone-Information-Guide)]
// @Tags [Cloud Metadata] Region/Zone
// @Accept  json
// @Produce  json
// @Param PreConfigRegionZoneListRequest query restruntime.PreConfigRegionZoneListRequest true "Query parameters for listing pre-configured region zones"
// @Success 200 {object} RegionZoneListResponse "List of Pre-configured Region Zones"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /preconfig/regionzone [get]
func ListRegionZonePreConfig(c echo.Context) error {
	cblog.Info("call ListRegionZonePreConfig()")

	req := PreConfigRegionZoneListRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.DriverName == "" {
		req.DriverName = c.QueryParam("DriverName")
	}
	if req.CredentialName == "" {
		req.CredentialName = c.QueryParam("CredentialName")
	}

	// Call common-runtime API
	result, err := cmrt.ListRegionZonePreConfig(req.DriverName, req.CredentialName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	jsonResult := RegionZoneListResponse{
		Result: result,
	}
	return c.JSON(http.StatusOK, &jsonResult)
}

// PreConfigRegionZoneGetRequest represents the request body for getting a specific region zone with pre-config.
type PreConfigRegionZoneGetRequest struct {
	DriverName     string `json:"DriverName" validate:"required" query:"DriverName"`         // example:"aws-driver"
	CredentialName string `json:"CredentialName" validate:"required" query:"CredentialName"` // example:"aws-credential"
}

// getRegionZonePreConfig godoc
// @ID get-region-zone-preconfig
// @Summary Get Pre-configured Region Zone
// @Description Retrieve details of a specific pre-configured Region Zone based on driver and credential names. 🕷️ [[User Guide](https://github.com/cloud-barista/cb-spider/wiki/REST-API-Region-Zone-Information-Guide)]
// @Tags [Cloud Metadata] Region/Zone
// @Accept  json
// @Produce  json
// @Param PreConfigRegionZoneGetRequest query restruntime.PreConfigRegionZoneGetRequest true "Query parameters for getting a specific pre-configured region zone"
// @Param Name path string true "The name of the Region to retrieve"
// @Success 200 {object} cres.RegionZoneInfo "Details of the Pre-configured Region Zone"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /preconfig/regionzone/{Name} [get]
func GetRegionZonePreConfig(c echo.Context) error {
	cblog.Info("call GetRegionZonePreConfig()")

	req := PreConfigRegionZoneGetRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.DriverName == "" {
		req.DriverName = c.QueryParam("DriverName")
	}
	if req.CredentialName == "" {
		req.CredentialName = c.QueryParam("CredentialName")
	}

	// Call common-runtime API
	result, err := cmrt.GetRegionZonePreConfig(req.DriverName, req.CredentialName, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// PreConfigOriginalRegionListRequest represents the request body for listing Original regions with pre-configuration.
type PreConfigOriginalRegionListRequest struct {
	DriverName     string `json:"DriverName" validate:"required" query:"DriverName"`         // example:"aws-driver"
	CredentialName string `json:"CredentialName" validate:"required" query:"CredentialName"` // example:"aws-credential"
}

// ListOrgRegionPreConfig godoc
// @ID list-preconfigured-original-org-region
// @Summary List Pre-configured Original Regions
// @Description Retrieve a list of pre-configured Original Regions based on driver and credential names. <br> The response structure may vary depending on the request DriverName and CredentialName.
// @Tags [Cloud Metadata] Region/Zone
// @Accept  json
// @Produce  json
// @Param PreConfigOriginalRegionListRequest query restruntime.PreConfigOriginalRegionListRequest true "Query parameters for listing pre-configured Original regions"
// @Success 200 {object} OriginalRegionListResponse "List of Pre-configured Original Regions"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /preconfig/orgregion [get]
func ListOrgRegionPreConfig(c echo.Context) error {
	cblog.Info("call ListPreConfiguredOriginalOrgRegion()")

	req := PreConfigOriginalRegionListRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.DriverName == "" {
		req.DriverName = c.QueryParam("DriverName")
	}
	if req.CredentialName == "" {
		req.CredentialName = c.QueryParam("CredentialName")
	}

	// Call common-runtime API
	result, err := cmrt.ListOrgRegionPreConfig(req.DriverName, req.CredentialName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var resultInterface interface{}
	if err := json.Unmarshal([]byte(result), &resultInterface); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to parse result")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"RegionInfo": resultInterface})
}
