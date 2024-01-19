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

// ================ PriceInfo Handler
func ListProductFamily(c echo.Context) error {
	cblog.Info("call ListProductFamily()")

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
	result, err := cmrt.ListProductFamily(req.ConnectionName, c.Param("RegionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result []string `json:"productfamily"`
	}
	jsonResult.Result = result
	return c.JSON(http.StatusOK, &jsonResult)
}

func GetPriceInfo(c echo.Context) error {
	cblog.Info("call GetPriceInfo()")

	var req struct {
		ConnectionName string
		FilterList     []cres.KeyValue
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.GetPriceInfo(req.ConnectionName, c.Param("ProductFamily"), c.Param("RegionName"), req.FilterList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var Result cres.CloudPriceData
	json.Unmarshal([]byte(result), &Result)
	return c.JSON(http.StatusOK, Result)

}
