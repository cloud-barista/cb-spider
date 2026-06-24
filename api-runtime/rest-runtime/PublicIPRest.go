// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2025.06.

package restruntime

import (
	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	// REST API (echo)
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

//================ PublicIP Handler

// PublicIPRegisterRequest represents the request body for registering a Public IP.
type PublicIPRegisterRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		Name  string `json:"Name" validate:"required" example:"publicip-01"`
		CSPId string `json:"CSPId" validate:"required" example:"eipalloc-0abc1234"`
	} `json:"ReqInfo" validate:"required"`
}

// RegisterPublicIP godoc
// @ID register-publicip
// @Summary Register Public IP
// @Description Register a new Public IP with the specified name and CSP ID.
// @Tags [PublicIP Management]
// @Accept  json
// @Produce  json
// @Param PublicIPRegisterRequest body restruntime.PublicIPRegisterRequest true "Request body for registering a Public IP"
// @Success 200 {object} cres.PublicIPInfo "Details of the registered Public IP"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regpublicip [post]
func RegisterPublicIP(c echo.Context) error {
	cblog.Info("call RegisterPublicIP()")

	req := PublicIPRegisterRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	userIId := cres.IID{req.ReqInfo.Name, req.ReqInfo.CSPId}
	result, err := cmrt.RegisterPublicIP(req.ConnectionName, userIId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// UnregisterPublicIP godoc
// @ID unregister-publicip
// @Summary Unregister Public IP
// @Description Unregister a Public IP with the specified name.
// @Tags [PublicIP Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for unregistering a Public IP"
// @Param Name path string true "The name of the Public IP to unregister"
// @Success 200 {object} BooleanInfo "Result of the unregister operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regpublicip/{Name} [delete]
func UnregisterPublicIP(c echo.Context) error {
	cblog.Info("call UnregisterPublicIP()")

	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := cmrt.UnregisterResource(req.ConnectionName, PUBLICIP, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &BooleanInfo{Result: strconv.FormatBool(result)})
}

// PublicIPListResponse is the response body for listing Public IPs.
type PublicIPListResponse struct {
	Result []*cres.PublicIPInfo `json:"publicip"`
}

// PublicIPCreateRequest represents the request body for creating a Public IP.
type PublicIPCreateRequest struct {
	ConnectionName  string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	IDTransformMode string `json:"IDTransformMode,omitempty" validate:"omitempty" example:"ON"`
	ReqInfo         struct {
		Name    string          `json:"Name" validate:"required" example:"publicip-01"`
		TagList []cres.KeyValue `json:"TagList,omitempty" validate:"omitempty"`
	} `json:"ReqInfo" validate:"required"`
}

// CreatePublicIP godoc
// @ID create-publicip
// @Summary Create Public IP
// @Description Allocate a new Public IP address.
// @Tags [PublicIP Management]
// @Accept  json
// @Produce  json
// @Param PublicIPCreateRequest body restruntime.PublicIPCreateRequest true "Request body for creating a Public IP"
// @Success 200 {object} cres.PublicIPInfo "Details of the created Public IP"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /publicip [post]
func CreatePublicIP(c echo.Context) error {
	cblog.Info("call CreatePublicIP()")

	req := PublicIPCreateRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	reqInfo := cres.PublicIPInfo{
		IId:     cres.IID{NameId: req.ReqInfo.Name},
		TagList: req.ReqInfo.TagList,
	}

	result, err := cmrt.CreatePublicIP(req.ConnectionName, PUBLICIP, reqInfo, req.IDTransformMode)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// ListPublicIP godoc
// @ID list-publicip
// @Summary List Public IPs
// @Description Retrieve a list of Public IP addresses.
// @Tags [PublicIP Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body containing the Connection Name"
// @Success 200 {object} restruntime.PublicIPListResponse "List of Public IPs"
// @Failure 400 {object} SimpleMsg "Bad Request"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /publicip [get]
func ListPublicIP(c echo.Context) error {
	cblog.Info("call ListPublicIP()")

	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	infoList, err := cmrt.ListPublicIP(req.ConnectionName, PUBLICIP)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result []*cres.PublicIPInfo `json:"publicip"`
	}
	if infoList == nil {
		infoList = []*cres.PublicIPInfo{}
	}
	jsonResult.Result = infoList

	return c.JSON(http.StatusOK, &jsonResult)
}

// GetPublicIP godoc
// @ID get-publicip
// @Summary Get Public IP
// @Description Retrieve details of a specific Public IP.
// @Tags [PublicIP Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body containing the Connection Name"
// @Param Name path string true "The name of the Public IP"
// @Success 200 {object} cres.PublicIPInfo "Details of the Public IP"
// @Failure 400 {object} SimpleMsg "Bad Request"
// @Failure 404 {object} SimpleMsg "Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /publicip/{Name} [get]
func GetPublicIP(c echo.Context) error {
	cblog.Info("call GetPublicIP()")

	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := cmrt.GetPublicIP(req.ConnectionName, PUBLICIP, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// PublicIPAssociateRequest represents the request body for associating a Public IP.
// Flow A (Non-NCP/GCP): provide NICName (+ optional PrivateIP).
// Flow B (GCP): provide NICName as "{vmName}/nic{n}" (e.g. "my-vm/nic0").
// Flow C (NCP): provide VMName only.
type PublicIPAssociateRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		VMName    string `json:"VMName,omitempty" validate:"omitempty" example:"my-vm-01"`       // NCP: required
		NICName   string `json:"NICName,omitempty" validate:"omitempty" example:"nic-01"`       // Non-NCP: required
		PrivateIP string `json:"PrivateIP,omitempty" validate:"omitempty" example:"10.0.1.11"` // Optional: specific private IP on NIC
	} `json:"ReqInfo" validate:"required"`
}

// AssociatePublicIP godoc
// @ID associate-publicip
// @Summary Associate Public IP with VM
// @Description Associate a Public IP address with a VM.
// @Tags [PublicIP Management]
// @Accept  json
// @Produce  json
// @Param PublicIPAssociateRequest body restruntime.PublicIPAssociateRequest true "Request body for associating a Public IP with a VM"
// @Param Name path string true "The name of the Public IP"
// @Success 200 {object} cres.PublicIPInfo "Updated Public IP info"
// @Failure 400 {object} SimpleMsg "Bad Request"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /publicip/{Name}/associate [put]
func AssociatePublicIP(c echo.Context) error {
	cblog.Info("call AssociatePublicIP()")

	req := PublicIPAssociateRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := cmrt.AssociatePublicIP(req.ConnectionName, c.Param("Name"), req.ReqInfo.VMName, req.ReqInfo.NICName, req.ReqInfo.PrivateIP)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// DisassociatePublicIP godoc
// @ID disassociate-publicip
// @Summary Disassociate Public IP from VM
// @Description Remove the VM association from a Public IP address.
// @Tags [PublicIP Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body containing the Connection Name"
// @Param Name path string true "The name of the Public IP"
// @Success 200 {object} BooleanInfo "Result of the disassociate operation"
// @Failure 400 {object} SimpleMsg "Bad Request"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /publicip/{Name}/disassociate [put]
func DisassociatePublicIP(c echo.Context) error {
	cblog.Info("call DisassociatePublicIP()")

	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := cmrt.DisassociatePublicIP(req.ConnectionName, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &BooleanInfo{Result: strconv.FormatBool(result)})
}

// CountAllPublicIPs godoc
// @ID count-all-publicips
// @Summary Count All Public IPs
// @Description Get the total number of Public IPs registered across all connections.
// @Tags [PublicIP Management]
// @Produce  json
// @Success 200 {object} CountResponse "Total count of Public IPs"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countpublicip [get]
func CountAllPublicIPs(c echo.Context) error {
	count, err := cmrt.CountAllPublicIPs()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, CountResponse{Count: int(count)})
}

// CountPublicIPsByConnection godoc
// @ID count-publicip-by-connection
// @Summary Count Public IPs by Connection
// @Description Get the total number of Public IPs for a specific connection.
// @Tags [PublicIP Management]
// @Produce  json
// @Param ConnectionName path string true "The name of the Connection"
// @Success 200 {object} CountResponse "Total count of Public IPs for the connection"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countpublicip/{ConnectionName} [get]
func CountPublicIPsByConnection(c echo.Context) error {
	count, err := cmrt.CountPublicIPsByConnection(c.Param("ConnectionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, CountResponse{Count: int(count)})
}

// DeletePublicIP godoc
// @ID delete-publicip
// @Summary Delete Public IP
// @Description Release (delete) a Public IP address.
// @Tags [PublicIP Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body containing the Connection Name"
// @Param Name path string true "The name of the Public IP to delete"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 400 {object} SimpleMsg "Bad Request"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /publicip/{Name} [delete]
func DeletePublicIP(c echo.Context) error {
	cblog.Info("call DeletePublicIP()")

	var req struct {
		ConnectionName string
		Force          string
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := cmrt.DeletePublicIP(req.ConnectionName, PUBLICIP, c.Param("Name"), req.Force)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &BooleanInfo{Result: strconv.FormatBool(result)})
}
