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

	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

//================ NIC Handler

// NICRegisterRequest represents the request body for registering a NIC.
type NICRegisterRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		Name  string `json:"Name" validate:"required" example:"nic-01"`
		CSPId string `json:"CSPId" validate:"required" example:"eni-0abc1234"`
	} `json:"ReqInfo" validate:"required"`
}

// RegisterNIC godoc
// @ID register-nic
// @Summary Register NIC
// @Description Register an existing NIC with the specified name and CSP ID.
// @Tags [NIC Management]
// @Accept  json
// @Produce  json
// @Param NICRegisterRequest body restruntime.NICRegisterRequest true "Request body for registering a NIC"
// @Success 200 {object} cres.NICInfo "Details of the registered NIC"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regnic [post]
func RegisterNIC(c echo.Context) error {
	cblog.Info("call RegisterNIC()")
	req := NICRegisterRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	userIId := cres.IID{req.ReqInfo.Name, req.ReqInfo.CSPId}
	result, err := cmrt.RegisterNIC(req.ConnectionName, userIId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, result)
}

// UnregisterNIC godoc
// @ID unregister-nic
// @Summary Unregister NIC
// @Description Unregister a NIC with the specified name.
// @Tags [NIC Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for unregistering a NIC"
// @Param Name path string true "The name of the NIC to unregister"
// @Success 200 {object} BooleanInfo "Result of the unregister operation"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regnic/{Name} [delete]
func UnregisterNIC(c echo.Context) error {
	cblog.Info("call UnregisterNIC()")
	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	result, err := cmrt.UnregisterResource(req.ConnectionName, NIC, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, &BooleanInfo{Result: strconv.FormatBool(result)})
}

// NICCreateRequest represents the request body for creating a NIC.
type NICCreateRequest struct {
	ConnectionName  string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	IDTransformMode string `json:"IDTransformMode,omitempty" validate:"omitempty" example:"ON"`
	ReqInfo         struct {
		Name              string          `json:"Name" validate:"required" example:"nic-01"`
		VPCName           string          `json:"VPCName" validate:"required" example:"vpc-01"`
		SubnetName        string          `json:"SubnetName" validate:"required" example:"subnet-01"`
		SecurityGroupNames []string       `json:"SecurityGroupNames,omitempty" validate:"omitempty"`
		TagList           []cres.KeyValue `json:"TagList,omitempty" validate:"omitempty"`
	} `json:"ReqInfo" validate:"required"`
}

// CreateNIC godoc
// @ID create-nic
// @Summary Create NIC
// @Description Create a new Network Interface Card (NIC).
// @Tags [NIC Management]
// @Accept  json
// @Produce  json
// @Param NICCreateRequest body restruntime.NICCreateRequest true "Request body for creating a NIC"
// @Success 200 {object} cres.NICInfo "Details of the created NIC"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /nic [post]
func CreateNIC(c echo.Context) error {
	cblog.Info("call CreateNIC()")
	req := NICCreateRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Build SecurityGroupIIDs
	var sgIIDs []cres.IID
	for _, name := range req.ReqInfo.SecurityGroupNames {
		sgIIDs = append(sgIIDs, cres.IID{NameId: name})
	}

	reqInfo := cres.NICReqInfo{
		IId:               cres.IID{NameId: req.ReqInfo.Name},
		VpcIID:            cres.IID{NameId: req.ReqInfo.VPCName},
		SubnetIID:         cres.IID{NameId: req.ReqInfo.SubnetName},
		SecurityGroupIIDs: sgIIDs,
		TagList:           req.ReqInfo.TagList,
	}

	result, err := cmrt.CreateNIC(req.ConnectionName, NIC, reqInfo, req.IDTransformMode)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, result)
}

// NICListResponse is the response body for listing NICs.
type NICListResponse struct {
	Result []*cres.NICInfo `json:"nic"`
}

// ListNIC godoc
// @ID list-nic
// @Summary List NICs
// @Description Retrieve a list of Network Interface Cards.
// @Tags [NIC Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body containing the Connection Name"
// @Success 200 {object} restruntime.NICListResponse "List of NICs"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /nic [get]
func ListNIC(c echo.Context) error {
	cblog.Info("call ListNIC()")
	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	infoList, err := cmrt.ListNIC(req.ConnectionName, NIC)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if infoList == nil {
		infoList = []*cres.NICInfo{}
	}
	return c.JSON(http.StatusOK, &NICListResponse{Result: infoList})
}

// GetNIC godoc
// @ID get-nic
// @Summary Get NIC
// @Description Retrieve details of a specific NIC.
// @Tags [NIC Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body containing the Connection Name"
// @Param Name path string true "The name of the NIC"
// @Success 200 {object} cres.NICInfo "Details of the NIC"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /nic/{Name} [get]
func GetNIC(c echo.Context) error {
	cblog.Info("call GetNIC()")
	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	result, err := cmrt.GetNIC(req.ConnectionName, NIC, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, result)
}

// DeleteNIC godoc
// @ID delete-nic
// @Summary Delete NIC
// @Description Delete a NIC.
// @Tags [NIC Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body containing the Connection Name"
// @Param Name path string true "The name of the NIC to delete"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /nic/{Name} [delete]
func DeleteNIC(c echo.Context) error {
	cblog.Info("call DeleteNIC()")
	var req struct {
		ConnectionName string
		Force          string
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	result, err := cmrt.DeleteNIC(req.ConnectionName, NIC, c.Param("Name"), req.Force)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, &BooleanInfo{Result: strconv.FormatBool(result)})
}

// NICAttachRequest represents the request body for attaching a NIC to a VM.
type NICAttachRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		VMName string `json:"VMName" validate:"required" example:"my-vm-01"`
	} `json:"ReqInfo" validate:"required"`
}

// AttachNIC godoc
// @ID attach-nic
// @Summary Attach NIC to VM
// @Description Attach a NIC to a VM instance.
// @Tags [NIC Management]
// @Accept  json
// @Produce  json
// @Param NICAttachRequest body restruntime.NICAttachRequest true "Request body for attaching a NIC to a VM"
// @Param Name path string true "The name of the NIC"
// @Success 200 {object} cres.NICInfo "Updated NIC info"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /nic/{Name}/attach [put]
func AttachNIC(c echo.Context) error {
	cblog.Info("call AttachNIC()")
	req := NICAttachRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	result, err := cmrt.AttachNIC(req.ConnectionName, c.Param("Name"), req.ReqInfo.VMName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, result)
}

// DetachNIC godoc
// @ID detach-nic
// @Summary Detach NIC from VM
// @Description Detach a NIC from its VM instance.
// @Tags [NIC Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body containing the Connection Name"
// @Param Name path string true "The name of the NIC"
// @Success 200 {object} BooleanInfo "Result of the detach operation"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /nic/{Name}/detach [put]
func DetachNIC(c echo.Context) error {
	cblog.Info("call DetachNIC()")
	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	result, err := cmrt.DetachNIC(req.ConnectionName, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, &BooleanInfo{Result: strconv.FormatBool(result)})
}

// NICPrivateIPRequest represents the request body for adding a private IP to a NIC.
type NICPrivateIPRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		PrivateIP string `json:"PrivateIP,omitempty" validate:"omitempty" example:"10.0.1.50"` // Leave empty for auto-assign
	} `json:"ReqInfo" validate:"required"`
}

// AddNICPrivateIP godoc
// @ID add-nic-privateip
// @Summary Add Private IP to NIC
// @Description Add a secondary private IP to a NIC. Leave PrivateIP empty for CSP auto-assignment.
// @Tags [NIC Management]
// @Accept  json
// @Produce  json
// @Param NICPrivateIPRequest body restruntime.NICPrivateIPRequest true "Request body"
// @Param Name path string true "The name of the NIC"
// @Success 200 {object} cres.NICInfo "Updated NIC info"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /nic/{Name}/privateip [post]
func AddNICPrivateIP(c echo.Context) error {
	cblog.Info("call AddNICPrivateIP()")
	req := NICPrivateIPRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	result, err := cmrt.AddNICPrivateIP(req.ConnectionName, c.Param("Name"), req.ReqInfo.PrivateIP)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, result)
}

// RemoveNICPrivateIP godoc
// @ID remove-nic-privateip
// @Summary Remove Private IP from NIC
// @Description Remove a secondary private IP from a NIC.
// @Tags [NIC Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body containing the Connection Name"
// @Param Name path string true "The name of the NIC"
// @Param IP path string true "The private IP address to remove"
// @Success 200 {object} BooleanInfo "Result of the remove operation"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /nic/{Name}/privateip/{IP} [delete]
func RemoveNICPrivateIP(c echo.Context) error {
	cblog.Info("call RemoveNICPrivateIP()")
	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	result, err := cmrt.RemoveNICPrivateIP(req.ConnectionName, c.Param("Name"), c.Param("IP"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, &BooleanInfo{Result: strconv.FormatBool(result)})
}

// NICOSConfigScriptResponse is the response body for GetNICOSConfigScript.
type NICOSConfigScriptResponse struct {
	// ConnectionName used to generate this script.
	ConnectionName string `json:"ConnectionName" example:"azure-koreacentral"`
	// NIC name the script applies to.
	NICName string `json:"NICName" example:"my-nic-01"`
	// Shell script to run inside the VM OS as root after AttachNIC.
	// Empty string means no OS configuration is required (e.g. AWS).
	Script string `json:"Script" example:"#!/bin/bash\n..."`
}

// GetNICOSConfigScript godoc
// @ID get-nic-osconfigscript
// @Summary Get NIC OS Configuration Script
// @Description Return a bash script to be executed inside the VM OS after a secondary NIC is attached.
// @Description AWS uses DHCP-based routing and returns an empty script.
// @Description Azure, Alibaba, Tencent, IBM, and OpenStack require manual OS-level interface
// @Description and routing table configuration; this endpoint returns the ready-to-run script.
// @Tags [NIC Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body containing the Connection Name"
// @Param Name path string true "The name of the NIC"
// @Success 200 {object} restruntime.NICOSConfigScriptResponse "OS configuration script"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /nic/{Name}/osconfigscript [get]
func GetNICOSConfigScript(c echo.Context) error {
	cblog.Info("call GetNICOSConfigScript()")
	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}
	nicName := c.Param("Name")
	script, err := cmrt.GetNICOSConfigScript(req.ConnectionName, nicName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, NICOSConfigScriptResponse{
		ConnectionName: req.ConnectionName,
		NICName:        nicName,
		Script:         script,
	})
}

// CountAllNICs godoc
// @ID count-all-nics
// @Summary Count All NICs
// @Description Get the total number of NICs registered across all connections.
// @Tags [NIC Management]
// @Produce  json
// @Success 200 {object} CountResponse "Total count of NICs"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countnic [get]
func CountAllNICs(c echo.Context) error {
	count, err := cmrt.CountAllNICs()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, CountResponse{Count: int(count)})
}

// CountNICsByConnection godoc
// @ID count-nic-by-connection
// @Summary Count NICs by Connection
// @Description Get the total number of NICs for a specific connection.
// @Tags [NIC Management]
// @Produce  json
// @Param ConnectionName path string true "The name of the Connection"
// @Success 200 {object} CountResponse "Total count of NICs for the connection"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countnic/{ConnectionName} [get]
func CountNICsByConnection(c echo.Context) error {
	count, err := cmrt.CountNICsByConnection(c.Param("ConnectionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, CountResponse{Count: int(count)})
}
