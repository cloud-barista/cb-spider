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

        "strconv"
)

//================ Image Handler
func CreateImage(c echo.Context) error {
	cblog.Info("call CreateImage()")

	var req struct {
		ConnectionName string
		ReqInfo        struct {
			Name string
		}
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	reqInfo := cres.ImageReqInfo{
		IId: cres.IID{req.ReqInfo.Name, ""},
	}

	// Call common-runtime API
	result, err := cmrt.CreateImage(req.ConnectionName, rsImage, reqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

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
	result, err := cmrt.ListImage(req.ConnectionName, rsImage)
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

	result, err := cmrt.GetImage(req.ConnectionName, rsImage, decodedImageName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

func DeleteImage(c echo.Context) error {
	cblog.Info("call DeleteImage()")

	var req struct {
		ConnectionName string
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.DeleteImage(req.ConnectionName, rsImage, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}
