// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, August 2026.

package restruntime

import (
	"encoding/json"
	"net/http"

	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

// ================ DBSpec Handler

// DBSpecListResponse represents the response body structure for the ListDBSpec API.
type DBSpecListResponse struct {
	Result []*cres.DBSpecInfo `json:"dbspec" validate:"required" description:"A list of Database instance specs"`
}

// OriginalDBSpecListResponse represents the dynamic structure for the Original DBSpec List response.
type OriginalDBSpecListResponse struct {
	DBSpecInfo map[string]interface{} `json:"DBSpecInfo" validate:"required"` // CSP-specific JSON format
}

// listDBSpec godoc
// @ID list-db-spec
// @Summary List Database Instance Specs
// @Description Retrieve a list of Database instance specs (vCPU/Memory/StorageSize details) for a specific DB engine, associated with a specific connection.
// @Tags [Cloud Metadata] DB Spec
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list DB instance specs for"
// @Param DBEngine query string true "DB engine name: mysql, mariadb, or postgresql"
// @Success 200 {object} DBSpecListResponse "List of Database instance specs"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /dbspec [get]
func ListDBSpec(c echo.Context) error {
	cblog.Info("call ListDBSpec()")

	connectionName := c.QueryParam("ConnectionName")
	dbEngine := c.QueryParam("DBEngine")

	result, err := cmrt.ListDBSpec(connectionName, dbEngine)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	jsonResult := DBSpecListResponse{
		Result: result,
	}
	return c.JSON(http.StatusOK, &jsonResult)
}

// getDBSpec godoc
// @ID get-db-spec
// @Summary Get Database Instance Spec
// @Description Retrieve details (vCPU/Memory/StorageSize) of a specific Database instance spec.
// @Tags [Cloud Metadata] DB Spec
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection"
// @Param DBEngine query string true "DB engine name: mysql, mariadb, or postgresql"
// @Param Name path string true "The name of the DB instance spec to retrieve"
// @Success 200 {object} cres.DBSpecInfo "Details of the Database instance spec"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /dbspec/{Name} [get]
func GetDBSpec(c echo.Context) error {
	cblog.Info("call GetDBSpec()")

	connectionName := c.QueryParam("ConnectionName")
	dbEngine := c.QueryParam("DBEngine")

	result, err := cmrt.GetDBSpec(connectionName, dbEngine, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// listOrgDBSpec godoc
// @ID list-org-db-spec
// @Summary List Original Database Instance Specs
// @Description Retrieve a list of Original Database instance specs for a specific DB engine, associated with a specific connection. <br> The response structure may vary depending on the requested CSP.
// @Tags [Cloud Metadata] DB Spec
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection"
// @Param DBEngine query string true "DB engine name: mysql, mariadb, or postgresql"
// @Success 200 {object} OriginalDBSpecListResponse "Dynamic JSON structure representing the list of Original Database instance specs"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /dborgspec [get]
func ListOrgDBSpec(c echo.Context) error {
	cblog.Info("call ListOrgDBSpec()")

	connectionName := c.QueryParam("ConnectionName")
	dbEngine := c.QueryParam("DBEngine")

	result, err := cmrt.ListOrgDBSpec(connectionName, dbEngine)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var resultInterface interface{}
	if err := json.Unmarshal([]byte(result), &resultInterface); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to parse result")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"DBSpecInfo": resultInterface})
}

// getOrgDBSpec godoc
// @ID get-org-db-spec
// @Summary Get Original Database Instance Spec
// @Description Retrieve details of a specific Original Database instance spec.
// @Tags [Cloud Metadata] DB Spec
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection"
// @Param DBEngine query string true "DB engine name: mysql, mariadb, or postgresql"
// @Param Name path string true "The name of the DB instance spec to retrieve"
// @Success 200 {object} OriginalDBSpecListResponse "Details of the Original Database instance spec"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /dborgspec/{Name} [get]
func GetOrgDBSpec(c echo.Context) error {
	cblog.Info("call GetOrgDBSpec()")

	connectionName := c.QueryParam("ConnectionName")
	dbEngine := c.QueryParam("DBEngine")

	result, err := cmrt.GetOrgDBSpec(connectionName, dbEngine, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	var resultInterface interface{}
	if err := json.Unmarshal([]byte(result), &resultInterface); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, "Failed to parse result")
	}

	return c.JSON(http.StatusOK, map[string]interface{}{"DBSpecInfo": resultInterface})
}
