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
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	// REST API (echo)
	"net/http"
	"github.com/labstack/echo/v4"
	"fmt"
	"encoding/json"
)

// define string of resource types
const (
	rsImage 	string = "image"
	rsVPC   	string = "vpc"
	rsSubnet 	string = "subnet"	
	rsSG  		string = "sg"
	rsKey 		string = "keypair"
	rsVM  		string = "vm"
	rsNLB  		string = "nlb"
	rsDisk  	string = "disk"
	rsMyImage 	string = "myimage"
	rsCluster 	string = "cluster"
	rsNodeGroup 	string = "nodegroup"
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

//================ Get Json string of CSP's Resource Info

func GetCSPResourceInfo(c echo.Context) error {
        cblog.Info("call GetCSPResourceInfo()")

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
        result, err := cmrt.GetCSPResourceInfo(req.ConnectionName, req.ResourceType, c.Param("Name"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

	switch req.ResourceType {
	case rsVPC:
		var Result cres.VPCInfo
		json.Unmarshal(result, &Result)
		return c.JSON(http.StatusOK, Result)
	case rsSG:
		var Result cres.SecurityInfo
		json.Unmarshal(result, &Result)
		return c.JSON(http.StatusOK, Result)
	case rsKey:
		var Result cres.KeyPairInfo
		json.Unmarshal(result, &Result)
		return c.JSON(http.StatusOK, Result)
	case rsVM:
		var Result cres.VMInfo
		json.Unmarshal(result, &Result)
		return c.JSON(http.StatusOK, Result)
	case rsNLB:
		var Result cres.NLBInfo
		json.Unmarshal(result, &Result)
		return c.JSON(http.StatusOK, Result)
	case rsDisk:
		var Result cres.DiskInfo
		json.Unmarshal(result, &Result)
		return c.JSON(http.StatusOK, Result)
	case rsMyImage:
		var Result cres.MyImageInfo
		json.Unmarshal(result, &Result)
		return c.JSON(http.StatusOK, Result)
	case rsCluster:
		var Result cres.ClusterInfo
		json.Unmarshal(result, &Result)
		return c.JSON(http.StatusOK, Result)
	default:
		return fmt.Errorf(req.ResourceType + " is not supported Resource!!")
	}

	return nil

}
