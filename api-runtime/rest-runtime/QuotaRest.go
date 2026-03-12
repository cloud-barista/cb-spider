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

// ================ Quota Handler

// QuotaResponse represents the response body structure for the GetQuota API.
type QuotaResponse struct {
	cres.QuotaInfo `json:",inline"`
}

// getQuota godoc
// @ID get-quota
// @Summary Get Resource Quota
// @Description Retrieve resource quota limits and current usage for CSP resources (VM, vCPU, VPC, Subnet, SecurityGroup, Disk, NLB, PublicIP, KeyPair, etc.). 🕷️ Supported CSPs: AWS, Azure, GCP, Alibaba, IBM.
// @Tags [Cloud Metadata] Resource Quota
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to retrieve resource quota for"
// @Success 200 {object} QuotaResponse "Resource quota limits and current usage"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /quota [get]
func GetQuota(c echo.Context) error {
	cblog.Info("call GetQuota()")

	req := ConnectionRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.GetQuota(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &result)
}
