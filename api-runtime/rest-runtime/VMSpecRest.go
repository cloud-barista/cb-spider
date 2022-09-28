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
)

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
