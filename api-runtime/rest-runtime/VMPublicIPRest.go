// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2026.08.

package restruntime

import (
	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	// REST API (echo)
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"
)

//================ VM Default PublicIP Handler

// AssignVMDefaultPublicIP godoc
// @ID assign-vm-default-publicip
// @Summary Assign Default PublicIP to a VM
// @Description Create a new PublicIP and attach it to the VM's default NIC (nic0) - the same effect as creating the VM with AssignPublicIP=true. Fails if the VM already has a default PublicIP. 🕷️
// @Tags [VM Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the VM"
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for assigning a default PublicIP"
// @Success 200 {object} cres.PublicIPInfo "Details of the assigned PublicIP"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vm/{Name}/publicip [post]
func AssignVMDefaultPublicIP(c echo.Context) error {
	cblog.Info("call AssignVMDefaultPublicIP()")

	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var result *cres.PublicIPInfo
	result, err := cmrt.AssignVMDefaultPublicIP(req.ConnectionName, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// UnassignVMDefaultPublicIP godoc
// @ID unassign-vm-default-publicip
// @Summary Unassign Default PublicIP from a VM
// @Description Disassociate and DELETE the PublicIP that was assigned via AssignVMDefaultPublicIP. Fails if the VM has no default PublicIP assigned. 🕷️
// @Tags [VM Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the VM"
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for unassigning the default PublicIP"
// @Success 200 {object} BooleanInfo "Result of the unassign operation"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vm/{Name}/publicip [delete]
func UnassignVMDefaultPublicIP(c echo.Context) error {
	cblog.Info("call UnassignVMDefaultPublicIP()")

	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := cmrt.UnassignVMDefaultPublicIP(req.ConnectionName, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &BooleanInfo{Result: strconv.FormatBool(result)})
}

// AttachVMPublicIP godoc
// @ID attach-vm-publicip
// @Summary Attach an existing PublicIP to a VM
// @Description Attach a PublicIP that was previously created via the PublicIP Manager (POST /publicip) to the VM's default NIC (nic0). Fails if the VM already has a default PublicIP. 🕷️
// @Tags [VM Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the VM"
// @Param PublicIPName path string true "The name of the existing PublicIP to attach"
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for attaching the PublicIP"
// @Success 200 {object} cres.PublicIPInfo "Details of the attached PublicIP"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vm/{Name}/publicip/{PublicIPName} [put]
func AttachVMPublicIP(c echo.Context) error {
	cblog.Info("call AttachVMPublicIP()")

	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var result *cres.PublicIPInfo
	result, err := cmrt.AttachVMPublicIP(req.ConnectionName, c.Param("Name"), c.Param("PublicIPName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// DetachVMPublicIP godoc
// @ID detach-vm-publicip
// @Summary Detach a PublicIP from a VM
// @Description Detach a PublicIP previously attached via AttachVMPublicIP, WITHOUT deleting the PublicIP resource itself (it belongs to the user). Fails if the VM has no default PublicIP assigned. 🕷️
// @Tags [VM Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the VM"
// @Param PublicIPName path string true "The name of the PublicIP to detach"
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for detaching the PublicIP"
// @Success 200 {object} BooleanInfo "Result of the detach operation"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vm/{Name}/publicip/{PublicIPName} [delete]
func DetachVMPublicIP(c echo.Context) error {
	cblog.Info("call DetachVMPublicIP()")

	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := cmrt.DetachVMPublicIP(req.ConnectionName, c.Param("Name"), c.Param("PublicIPName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &BooleanInfo{Result: strconv.FormatBool(result)})
}
