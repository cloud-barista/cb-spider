// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2025.02.

package restruntime

import (
	"net/http"

	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	ifs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	"github.com/labstack/echo/v4"
)

// DriverCapabilityResponse represents the response body structure for the GetDriverCapability API.
type DriverCapabilityResponse struct {
	ifs.DriverCapabilityInfo `json:",inline" description:"Driver capability information details"`
}

// getDriverCapability godoc
// @ID get-driver-capability
// @Summary Get Driver Capability Information
// @Description Retrieve capability information of the cloud driver.
// @Tags [Cloud Driver] Capability
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "Name of connection to retrieve driver capability for"
// @Success 200 {object} DriverCapabilityResponse "Driver Capability Information Details"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to empty or invalid connection name"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /driver/capability [get]
func GetDriverCapability(c echo.Context) error {
	cblog.Info("call GetDriverCapability()")

	// Get connection name from query parameter
	connectionName := c.QueryParam("ConnectionName")

	// Call common-runtime API
	result, err := cmrt.GetDriverCapabilityInfo(connectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	response := DriverCapabilityResponse{result}
	return c.JSON(http.StatusOK, response)
}
