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

func AnyCall(c echo.Context) error {
        cblog.Info("call AnyCall()")

        var req struct {
                ConnectionName string
		ReqInfo	struct {
			FID string
			IKeyValueList []cres.KeyValue
		}
        }


        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

	reqInfo := cres.AnyCallInfo {
		FID:	req.ReqInfo.FID,
		IKeyValueList: req.ReqInfo.IKeyValueList,
	}

        // Call common-runtime API
        result, err := cmrt.AnyCall(req.ConnectionName, reqInfo)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

