// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2025.07.

package restruntime

import (
	"net/http"

	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

// ================ Quota Info Handler

// QuotaServiceTypeResponse represents the response body for the ListQuotaServiceType API.
type QuotaServiceTypeResponse struct {
	ServiceTypes []string `json:"ServiceTypes" validate:"required" example:"[\"ec2\",\"vpc\",\"ebs\"]"`
}

// QuotaResponse represents the response body structure for the GetQuotaInfo API.
type QuotaResponse struct {
	cres.QuotaInfo `json:",inline"`
}

// listQuotaServiceType godoc
// @ID list-quota-service-type
// @Summary List Quota Service Types
// @Description Retrieve the list of service type names for which quota information is available. Use a returned service type as input to the GetQuotaInfo API. 🕷️ Supported CSPs: AWS, Azure, GCP, Alibaba.
// @Tags [Cloud Metadata] Quota Info
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to retrieve service types for"
// @Success 200 {object} QuotaServiceTypeResponse "List of available service type names"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /quotaservicetype [get]
func ListQuotaServiceType(c echo.Context) error {
	cblog.Info("call ListQuotaServiceType()")

	req := ConnectionRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.ListQuotaServiceType(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, QuotaServiceTypeResponse{ServiceTypes: result})
}

// getQuotaInfo godoc
// @ID get-quota-info
// @Summary Get Quota Info
// @Description Retrieve all quota limits and current usage for a given service type. No filtering is applied; all CSP-original quota items for the service type are returned. 🕷️ Supported CSPs: AWS, Azure, GCP, Alibaba.
// @Tags [Cloud Metadata] Quota Info
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to retrieve quota info for"
// @Param ServiceType query string true "The service type name (obtained from ListQuotaServiceType)"
// @Success 200 {object} QuotaResponse "Quota limits and current usage"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /quotainfo [get]
func GetQuotaInfo(c echo.Context) error {
	cblog.Info("call GetQuotaInfo()")

	req := ConnectionRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	serviceType := c.QueryParam("ServiceType")

	// Call common-runtime API
	result, err := cmrt.GetQuotaInfo(req.ConnectionName, serviceType)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &result)
}
