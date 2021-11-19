// Cloud Info Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2019.09.

package restruntime

import (
	"strconv"

	im "github.com/cloud-barista/cb-spider/cloud-info-manager"
	ccim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/credential-info-manager"
	dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"
	rim "github.com/cloud-barista/cb-spider/cloud-info-manager/region-info-manager"

	// REST API (echo)
	"net/http"

	"github.com/labstack/echo/v4"
)

//================ List of support CloudOS
func ListCloudOS(c echo.Context) error {
	cblog.Info("call ListCloudOS()")

	infoList := im.ListCloudOS()

	var jsonResult struct {
		Result []string `json:"cloudos"`
	}
	if infoList == nil {
		infoList = []string{}
	}
	jsonResult.Result = infoList
	return c.JSON(http.StatusOK, &jsonResult)
}

//================ CloudOS Metainfo
func GetCloudOSMetaInfo(c echo.Context) error {
        cblog.Info("call GetCloudOSMetaInfo()")

	cldMetainfo, err := im.GetCloudOSMetaInfo(c.Param("CloudOSName"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, &cldMetainfo)
}

//================ CloudDriver Handler
func RegisterCloudDriver(c echo.Context) error {
	cblog.Info("call RegisterCloudDriver()")
	req := &dim.CloudDriverInfo{}
	if err := c.Bind(req); err != nil {
		cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	cldinfoList, err := dim.RegisterCloudDriverInfo(*req)
	if err != nil {
		cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &cldinfoList)
}

func ListCloudDriver(c echo.Context) error {
	cblog.Info("call ListCloudDriver()")

	infoList, err := dim.ListCloudDriver()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result []*dim.CloudDriverInfo `json:"driver"`
	}
	if infoList == nil {
		infoList = []*dim.CloudDriverInfo{}
	}
	jsonResult.Result = infoList
	return c.JSON(http.StatusOK, &jsonResult)

}

func GetCloudDriver(c echo.Context) error {
	cblog.Info("call GetCloudDriver()")

	cldinfo, err := dim.GetCloudDriver(c.Param("DriverName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &cldinfo)
}

func UnRegisterCloudDriver(c echo.Context) error {
	cblog.Info("call UnRegisterCloudDriver()")

	result, err := dim.UnRegisterCloudDriver(c.Param("DriverName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

//================ Credential Handler
func RegisterCredential(c echo.Context) error {
	cblog.Info("call RegisterCredential()")

	req := &cim.CredentialInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	crdinfoList, err := cim.RegisterCredentialInfo(*req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfoList)
}

func ListCredential(c echo.Context) error {
	cblog.Info("call ListCredential()")

	infoList, err := cim.ListCredential()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result []*cim.CredentialInfo `json:"credential"`
	}
	if infoList == nil {
		infoList = []*cim.CredentialInfo{}
	}
	jsonResult.Result = infoList
	return c.JSON(http.StatusOK, &jsonResult)
}

func GetCredential(c echo.Context) error {
	cblog.Info("call GetCredential()")

	crdinfo, err := cim.GetCredential(c.Param("CredentialName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfo)
}

func UnRegisterCredential(c echo.Context) error {
	cblog.Info("call UnRegisterCredential()")

	result, err := cim.UnRegisterCredential(c.Param("CredentialName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

//================ Region Handler
func RegisterRegion(c echo.Context) error {
	cblog.Info("call RegisterRegion()")

	req := &rim.RegionInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	crdinfoList, err := rim.RegisterRegionInfo(*req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfoList)
}

func ListRegion(c echo.Context) error {
	cblog.Info("call ListRegion()")

	infoList, err := rim.ListRegion()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result []*rim.RegionInfo `json:"region"`
	}
	if infoList == nil {
		infoList = []*rim.RegionInfo{}
	}
	jsonResult.Result = infoList
	return c.JSON(http.StatusOK, &jsonResult)
}

func GetRegion(c echo.Context) error {
	cblog.Info("call GetRegion()")

	crdinfo, err := rim.GetRegion(c.Param("RegionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfo)
}

func UnRegisterRegion(c echo.Context) error {
	cblog.Info("call UnRegisterRegion()")

	result, err := rim.UnRegisterRegion(c.Param("RegionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

//================ ConnectionConfig Handler
func CreateConnectionConfig(c echo.Context) error {
	cblog.Info("call CreateConnectionConfig()")

	req := &ccim.ConnectionConfigInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	crdinfoList, err := ccim.CreateConnectionConfigInfo(*req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfoList)
}

func ListConnectionConfig(c echo.Context) error {
	cblog.Info("call ListConnectionConfig()")

	infoList, err := ccim.ListConnectionConfig()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result []*ccim.ConnectionConfigInfo `json:"connectionconfig"`
	}
	if infoList == nil {
		infoList = []*ccim.ConnectionConfigInfo{}
	}
	jsonResult.Result = infoList
	return c.JSON(http.StatusOK, &jsonResult)
}

func GetConnectionConfig(c echo.Context) error {
	cblog.Info("call GetConnectionConfig()")

	crdinfo, err := ccim.GetConnectionConfig(c.Param("ConfigName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfo)
}

func DeleteConnectionConfig(c echo.Context) error {
	cblog.Info("call DeleteConnectionConfig()")

	result, err := ccim.DeleteConnectionConfig(c.Param("ConfigName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}
