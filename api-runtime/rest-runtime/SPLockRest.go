// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2022.06.

package restruntime

import (

        cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"

        // REST API (echo)
        "net/http"

        "github.com/labstack/echo/v4"
)

//================ Get All SPLock Infos

func GetAllSPLockInfo(c echo.Context) error {
        cblog.Info("call GetAllSPLockInfo()")

        infoList := cmrt.GetAllSPLockInfo()

        var jsonResult struct {
                Result []string `json:"splockinfo"`
        }
        if infoList == nil {
                infoList = []string{}
        }
        jsonResult.Result = infoList
        return c.JSON(http.StatusOK, &jsonResult)
}

