// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.04.
// by CB-Spider Team, 2019.10.

package restruntime

import (
	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"

	// REST API (echo)
	"net/http"
	"github.com/labstack/echo/v4"
)

// define string of resource types
const (
	rsImage string = "image"
	rsVPC   string = "vpc"
	rsSubnet string = "subnet"	
	rsSG  string = "sg"
	rsKey string = "keypair"
	rsVM  string = "vm"
	rsNLB  string = "nlb"
	rsDisk  string = "disk"
)


//================ Get CSP Resource Name

func GetCSPResourceName(c echo.Context) error {
        cblog.Info("call GetCSPResourceName()")

        var req struct {
                ConnectionName string
                ResourceType string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // To support for Get-Query Param Type API
        if req.ConnectionName == "" {
                req.ConnectionName = c.QueryParam("ConnectionName")
        }
        if req.ResourceType == "" {
                req.ResourceType = c.QueryParam("ResourceType")
        }

        // Call common-runtime API
        result, err := cmrt.GetCSPResourceName(req.ConnectionName, req.ResourceType, c.Param("Name"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        var resultInfo struct {
                Name string
        }
	resultInfo.Name = string(result)

        return c.JSON(http.StatusOK, &resultInfo)
}
