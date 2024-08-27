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
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

// define string of resource types
// redefined for backward compatibility
const (
	IMAGE     string = string(cres.IMAGE)
	VPC       string = string(cres.VPC)
	SUBNET    string = string(cres.SUBNET)
	SG        string = string(cres.SG)
	KEY       string = string(cres.KEY)
	VM        string = string(cres.VM)
	NLB       string = string(cres.NLB)
	DISK      string = string(cres.DISK)
	MYIMAGE   string = string(cres.MYIMAGE)
	CLUSTER   string = string(cres.CLUSTER)
	NODEGROUP string = string(cres.NODEGROUP)
)

//================ Common Request & Response

// ConnectionRequest represents the request body for common use.
type ConnectionRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
}

// REST API Return struct for boolean type
type BooleanInfo struct {
	Result string `json:"Result" validate:"required" example:"true"` // true or false
}

type StatusInfo struct {
	Status string `json:"Status" validate:"required" example:"RUNNING"` // "RUNNING,SUSPENDING,SUSPENDED,REBOOTING,TERMINATING,TERMINATED,NOTEXIST,FAILED"
}

//================ Get CSP Resource Name

func GetCSPResourceName(c echo.Context) error {
	cblog.Info("call GetCSPResourceName()")

	var req struct {
		ConnectionName string
		ResourceType   string
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
		ResourceType   string
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
	case VPC:
		var Result cres.VPCInfo
		json.Unmarshal(result, &Result)
		return c.JSON(http.StatusOK, Result)
	case SG:
		var Result cres.SecurityInfo
		json.Unmarshal(result, &Result)
		return c.JSON(http.StatusOK, Result)
	case KEY:
		var Result cres.KeyPairInfo
		json.Unmarshal(result, &Result)
		return c.JSON(http.StatusOK, Result)
	case VM:
		var Result cres.VMInfo
		json.Unmarshal(result, &Result)
		return c.JSON(http.StatusOK, Result)
	case NLB:
		var Result cres.NLBInfo
		json.Unmarshal(result, &Result)
		return c.JSON(http.StatusOK, Result)
	case DISK:
		var Result cres.DiskInfo
		json.Unmarshal(result, &Result)
		return c.JSON(http.StatusOK, Result)
	case MYIMAGE:
		var Result cres.MyImageInfo
		json.Unmarshal(result, &Result)
		return c.JSON(http.StatusOK, Result)
	case CLUSTER:
		var Result cres.ClusterInfo
		json.Unmarshal(result, &Result)
		return c.JSON(http.StatusOK, Result)
	default:
		return fmt.Errorf(req.ResourceType + " is not supported Resource!!")
	}

	return nil

}

func Destroy(c echo.Context) error {
	cblog.Info("call Destroy()")

	var req struct {
		ConnectionName string
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.Destroy(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &result)
}
