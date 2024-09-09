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

	"github.com/labstack/echo/v4"
)

//================ AnyCall Handler

// AnyCallRequest represents the request body for executing the AnyCall API.
type AnyCallRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"mock-config01"`
	ReqInfo        struct {
		FID           string          `json:"FID" validate:"required" example:"countAll"`   // Function ID (FID) to call, ex: countAll
		IKeyValueList []cres.KeyValue `json:"IKeyValueList,omitempty" validate:"omitempty"` // Input key-value pairs, ex:[{"Key": "rsType", "Value": "vpc"}]
	} `json:"ReqInfo" validate:"required"`
}

// anyCall godoc
// @ID any-call
// @Summary Execute AnyCall
// @Description Execute a custom function (FID) with key-value parameters through AnyCall. 🕷️ [[Development Guide](https://github.com/cloud-barista/cb-spider/wiki/AnyCall-API-Extension-Guide)]
// @Tags [AnyCall Management]
// @Accept  json
// @Produce  json
// @Param AnyCallRequest body restruntime.AnyCallRequest true "Request body for executing AnyCall"
// @Success 200 {object} cres.AnyCallInfo "Result of the AnyCall operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /anycall [post]
func AnyCall(c echo.Context) error {
	cblog.Info("call AnyCall()")

	req := AnyCallRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	reqInfo := cres.AnyCallInfo{
		FID:           req.ReqInfo.FID,
		IKeyValueList: req.ReqInfo.IKeyValueList,
	}

	// Call common-runtime API
	result, err := cmrt.AnyCall(req.ConnectionName, reqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}
