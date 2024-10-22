// CB-Spider's Spiderlet Supporter REST Runtime.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista

package restruntime

import (
	"net/http"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	"github.com/labstack/echo/v4"
)

// Critical data for Spiderlet.
// Requires a TLS environment.
func GetCloudDriverAndConnectionInfoTLS(c echo.Context) error {
	cblog.Info("call GetCredential()")

	crdinfo, err := ccm.GetCloudDriverAndConnectionInfo(c.Param("ConnectionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfo)
}
