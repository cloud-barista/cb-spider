// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2024.

package restruntime

import (
	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	// REST API (echo)
	"net/http"

	"strconv"

	"github.com/labstack/echo/v4"
)

//================ Tag Handler

type tagAddReq struct {
	ConnectionName string
	ReqInfo        struct {
		ResourceType cres.RSType
		ResourceName string
		Tag          cres.KeyValue
	}
}

func AddTag(c echo.Context) error {
	cblog.Info("call AddTag()")

	req := tagAddReq{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.AddTag(req.ConnectionName, req.ReqInfo.ResourceType, req.ReqInfo.ResourceName, req.ReqInfo.Tag)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

type tagListReq struct {
	ConnectionName string
	ReqInfo        struct {
		ResourceType cres.RSType
		ResourceName string
	}
}

func ListTag(c echo.Context) error {
	cblog.Info("call ListTag()")

	req := tagListReq{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.ListTag(req.ConnectionName, req.ReqInfo.ResourceType, req.ReqInfo.ResourceName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result []cres.KeyValue `json:"tag"`
	}
	jsonResult.Result = result
	return c.JSON(http.StatusOK, &jsonResult)
}

type tagGetReq struct {
	ConnectionName string
	ReqInfo        struct {
		ResourceType cres.RSType
		ResourceName string
	}
}

func GetTag(c echo.Context) error {
	cblog.Info("call GetTag()")

	req := tagGetReq{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.GetTag(req.ConnectionName, req.ReqInfo.ResourceType, req.ReqInfo.ResourceName, c.Param("Key"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

type tagRemoveReq struct {
	ConnectionName string
	ReqInfo        struct {
		ResourceType cres.RSType
		ResourceName string
	}
}

func RemoveTag(c echo.Context) error {
	cblog.Info("call RemoveTag()")

	req := tagRemoveReq{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.RemoveTag(req.ConnectionName, req.ReqInfo.ResourceType, req.ReqInfo.ResourceName, c.Param("Key"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

type tagFindReq struct {
	ConnectionName string
	ReqInfo        struct {
		ResourceType cres.RSType
		Keyword      string
	}
}

func FindTag(c echo.Context) error {
	cblog.Info("call FindTag()")

	req := tagFindReq{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.FindTag(req.ConnectionName, req.ReqInfo.ResourceType, req.ReqInfo.Keyword)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result []*cres.TagInfo `json:"tag"`
	}
	jsonResult.Result = result
	return c.JSON(http.StatusOK, &jsonResult)
}
