// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2023.09.

package restruntime

import (
	"net/http"

	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

// ================ RegionZone Handler
func ListRegionZone(c echo.Context) error {
	cblog.Info("call ListRegionZone()")

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
	result, err := cmrt.ListRegionZone(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result []*cres.RegionZoneInfo `json:"regionzone"`
	}
	jsonResult.Result = result
	return c.JSON(http.StatusOK, &jsonResult)
}

func GetRegionZone(c echo.Context) error {
	cblog.Info("call GetRegionZone()")

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
	result, err := cmrt.GetRegionZone(req.ConnectionName, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

func ListOrgRegion(c echo.Context) error {
	cblog.Info("call ListOrgRegion()")

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
	result, err := cmrt.ListOrgRegion(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.String(http.StatusOK, result)
}

func ListOrgZone(c echo.Context) error {
	cblog.Info("call ListOrgZone()")

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
	result, err := cmrt.ListOrgZone(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.String(http.StatusOK, result)
}

// ================ RegionZone Handler (Pre-Config Version)
func ListRegionZonePreConfig(c echo.Context) error {
	cblog.Info("call ListRegionZonePreConfig()")

	var req struct {
		DriverName     string
		CredentialName string
	}

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

	var jsonResult struct {
		Result []*cres.RegionZoneInfo `json:"regionzone"`
	}
	jsonResult.Result = result
	return c.JSON(http.StatusOK, &jsonResult)
}

func GetRegionZonePreConfig(c echo.Context) error {
	cblog.Info("call GetRegionZonePreConfig()")

	var req struct {
		DriverName     string
		CredentialName string
	}

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

func ListOrgRegionPreConfig(c echo.Context) error {
	cblog.Info("call ListOrgRegionPreConfig()")

	var req struct {
		DriverName     string
		CredentialName string
	}

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

	return c.String(http.StatusOK, result)
}
