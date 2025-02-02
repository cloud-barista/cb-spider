// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// CB-Spider Team

package restruntime

import (
	"net"
	"strconv"
	"time"

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

// CountResponse represents the response body for counting all VPCs.
type CountResponse struct {
	Count int `json:"count" validate:"required" example:"5" description:"The total number of resources counted"`
}

// AllResourceListResponse represents the response body structure for the ListAllVPC API.
type AllResourceListResponse struct {
	AllList struct {
		MappedList     []*cres.IID `json:"MappedList" validate:"required" description:"A list of resources that are mapped between CB-Spider and CSP"`
		OnlySpiderList []*cres.IID `json:"OnlySpiderList" validate:"required" description:"A list of resources that exist only in CB-Spider"`
		OnlyCSPList    []*cres.IID `json:"OnlyCSPList" validate:"required" description:"A list of resources that exist only in the CSP"`
	} `json:"AllList" validate:"required" description:"A list of all resources with their respective lists"`
}

// AllResourceInfoListResponse represents the response body structure for the ListAllVPCInfo API.
type AllResourceInfoListResponse struct {
	ResourceType cres.RSType `json:"ResourceType" validate:"required" example:"vpc" description:"The type of resource"`
	AllListInfo  struct {
		MappedInfoList  []interface{} `json:"MappedInfoList" validate:"required" description:"A list of resources that are mapped between CB-Spider and CSP"`
		OnlySpiderList  []*cres.IID   `json:"OnlySpiderList" validate:"required" description:"A list of resources that exist only in CB-Spider"`
		OnlyCSPInfoList []interface{} `json:"OnlyCSPInfoList" validate:"required" description:"A list of resources that exist only in the CSP"`
	} `json:"AllListInfo" validate:"required" description:"A list of all resources info with their respective lists"`
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

// Destroy godoc
// @ID destroy-all-resource
// @Summary Destroy all resources in a connection
// @Description Deletes all resources associated with a specific cloud connection. This action is irreversible.
// @Tags [Utility]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for deleting all resources"
// @Success 200 {object} cmrt.DestroyedInfo "Details of the destroyed resources"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to missing parameters"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /destroy [delete]
func Destroy(c echo.Context) error {
	cblog.Info("call Destroy()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.Destroy(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &result)
}

// CheckTCPPort godoc
// @ID check-tcp-port
// @Summary Check if a specific TCP port is open
// @Description Verifies whether a given TCP port is open on the specified host.
// @Tags [Utility]
// @Accept  json
// @Produce  json
// @Param HostName query string true "The hostname or IP address to check"
// @Param Port query int true "The TCP port to check"
// @Success 200 {object} SimpleMsg "Success message with port status"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid parameters"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /check/tcp [get]
func CheckTCPPort(c echo.Context) error {
	cblog.Info("call CheckTCPPortHandler()")

	// Fetching hostname and port from query parameters
	hostname := c.QueryParam("HostName")
	port := c.QueryParam("Port")

	if hostname == "" {
		cblog.Error("Hostname parameter is missing")
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required query parameter: Hostname")
	}

	if port == "" {
		cblog.Error("Port parameter is missing")
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required query parameter: Port")
	}

	// Convert port to integer
	portInt, err := strconv.Atoi(port)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid port value")
	}

	// Call the port check function
	err = checkTCPPort(hostname, portInt)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Error: TCP port %d is not open on %s: %v", portInt, hostname, err))
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Success: TCP port %d is open on %s", portInt, hostname),
	})
}

// CheckUDPPort godoc
// @ID check-udp-port
// @Summary Check if a specific UDP port is open
// @Description Verifies whether a given UDP port is open on the specified host.
// @Description ※ Note: As UDP is connectionless, this check mainly performs a lookup and may not confirm if the server is working.
// @Tags [Utility]
// @Accept  json
// @Produce  json
// @Param HostName query string true "The hostname or IP address to check"
// @Param Port query int true "The UDP port to check"
// @Success 200 {object} SimpleMsg "Success message with port status"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid parameters"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /check/udp [get]
func CheckUDPPort(c echo.Context) error {
	cblog.Info("call CheckUDPPortHandler()")

	// Fetching hostname and port from query parameters
	hostname := c.QueryParam("HostName")
	port := c.QueryParam("Port")

	if hostname == "" {
		cblog.Error("Hostname parameter is missing")
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required query parameter: Hostname")
	}

	if port == "" {
		cblog.Error("Port parameter is missing")
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required query parameter: Port")
	}

	// Convert port to integer
	portInt, err := strconv.Atoi(port)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid port value")
	}

	// Call the port check function
	err = checkUDPPort(hostname, portInt)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Sprintf("Error: UDP port %d is not open on %s: %v", portInt, hostname, err))
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Success: UDP port %d is open on %s", portInt, hostname),
	})
}

//================ Port Checking Functions (TCP/UDP)

// checkTCPPort checks if a TCP port is open
func checkTCPPort(hostname string, port int) error {
	address := fmt.Sprintf("%s:%d", hostname, port)
	timeout := 3 * time.Second

	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		cblog.Errorf("TCP Port %d closed on %s: %v\n", port, hostname, err)
		return err
	}
	defer conn.Close()
	return nil
}

// checkUDPPort checks if a UDP port is open
func checkUDPPort(hostname string, port int) error {
	address := fmt.Sprintf("%s:%d", hostname, port)
	timeout := 3 * time.Second

	conn, err := net.DialTimeout("udp", address, timeout)
	if err != nil {
		cblog.Errorf("UDP Port %d closed on %s: %v\n", port, hostname, err)
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte{})
	if err != nil {
		cblog.Errorf("UDP Port %d closed on %s: %v\n", port, hostname, err)
		return err
	}

	return nil
}

func listAllResourceInfo(c echo.Context, rsType cres.RSType) error {
	cblog.Info("call listAllResourceInfo()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	allResourceInfoList, err := cmrt.ListAllResourceInfo(req.ConnectionName, rsType)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &allResourceInfoList)
}
