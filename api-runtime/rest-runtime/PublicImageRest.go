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
	"net/url"

	"github.com/labstack/echo/v4"
)

// ================ Image Handler
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
	result, err := cmrt.ListImage(req.ConnectionName, IMAGE)
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

	result, err := cmrt.GetImage(req.ConnectionName, IMAGE, decodedImageName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}
