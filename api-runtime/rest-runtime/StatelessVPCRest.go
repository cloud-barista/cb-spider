// Stateless API mode Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This file provides the "Stateless Mode" counterpart of VPC-SubnetRest.go.
// Unlike the default (managed) mode, Stateless Mode:
//   - does not use a ConnectionName; the caller supplies ProviderName + CSP-native
//     CredentialInfo/RegionInfo inline on every call,
//   - does not persist any IID mapping to the MetaDB,
//   - returns the CSP driver's IId (NameId/SystemId) unmodified, so SystemId is always
//     the raw CSP-native resource ID.
//
// It calls the cloud-driver interfaces directly (cloud-control-manager + cloud-driver/interfaces)
// and does not go through api-runtime/common-runtime, which owns the managed-mode ConnectionName
// resolution and IID persistence that Stateless Mode intentionally skips.
//
// by CB-Spider Team, 2026.11.

package restruntime

import (
	"net/http"
	"strconv"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	icon "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/connect"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"github.com/labstack/echo/v4"
)

// StatelessConnectionRequest is the common request envelope for every Stateless Mode API call:
// the CSP Provider name plus CSP-native Credential/Region info, supplied inline instead of
// being resolved from a registered ConnectionName.
type StatelessConnectionRequest struct {
	ProviderName   string              `json:"ProviderName" validate:"required" example:"AWS"`
	CredentialInfo idrv.CredentialInfo `json:"CredentialInfo" validate:"required"`
	RegionInfo     idrv.RegionInfo     `json:"RegionInfo" validate:"required"`
}

func (req StatelessConnectionRequest) connect() (icon.CloudConnection, error) {
	return ccm.ConnectCloudByProvider(req.ProviderName, req.CredentialInfo, req.RegionInfo)
}

//================ Stateless VPC Handler

// StatelessVPCCreateRequest represents the request body for creating a VPC in Stateless Mode.
type StatelessVPCCreateRequest struct {
	StatelessConnectionRequest
	ReqInfo cres.VPCReqInfo `json:"ReqInfo" validate:"required"`
}

// statelessCreateVPC godoc
// @ID stateless-create-vpc
// @Summary [Stateless Mode] Create VPC
// @Description Create a VPC directly through the CSP driver using inline Provider/Credential/Region info. No ConnectionName is used and no IID mapping is stored; the returned IId.SystemId is the raw CSP-native resource ID.
// @Tags [Stateless Mode] VPC
// @Accept  json
// @Produce  json
// @Param StatelessVPCCreateRequest body restruntime.StatelessVPCCreateRequest true "Request body for creating a VPC in Stateless Mode"
// @Success 200 {object} cres.VPCInfo "Details of the created VPC"
// @Failure 400 {object} SimpleMsg "Bad Request"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /stateless/vpc [post]
func StatelessCreateVPC(c echo.Context) error {
	cblog.Info("call StatelessCreateVPC()")

	req := StatelessVPCCreateRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	conn, err := req.connect()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := conn.CreateVPCHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := handler.CreateVPC(req.ReqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// StatelessVPCListResponse represents the response body for listing VPCs in Stateless Mode.
type StatelessVPCListResponse struct {
	Result []*cres.VPCInfo `json:"vpc" validate:"required" description:"A list of VPC information, as returned by the CSP"`
}

// statelessListVPC godoc
// @ID stateless-list-vpc
// @Summary [Stateless Mode] List VPCs
// @Description List every VPC visible to the given Provider/Credential/Region, read directly from the CSP (no MetaDB involved).
// @Tags [Stateless Mode] VPC
// @Accept  json
// @Produce  json
// @Param StatelessConnectionRequest body restruntime.StatelessConnectionRequest true "Provider/Credential/Region info"
// @Success 200 {object} StatelessVPCListResponse "List of VPCs"
// @Failure 400 {object} SimpleMsg "Bad Request"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /stateless/vpc [get]
func StatelessListVPC(c echo.Context) error {
	cblog.Info("call StatelessListVPC()")

	req := StatelessConnectionRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	conn, err := req.connect()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := conn.CreateVPCHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := handler.ListVPC()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &StatelessVPCListResponse{Result: result})
}

// statelessGetVPC godoc
// @ID stateless-get-vpc
// @Summary [Stateless Mode] Get VPC
// @Description Get a single VPC by its CSP-native SystemId, read directly from the CSP (no MetaDB involved).
// @Tags [Stateless Mode] VPC
// @Accept  json
// @Produce  json
// @Param SystemId path string true "The CSP-native SystemId of the VPC"
// @Param StatelessConnectionRequest body restruntime.StatelessConnectionRequest true "Provider/Credential/Region info"
// @Success 200 {object} cres.VPCInfo "Details of the VPC"
// @Failure 400 {object} SimpleMsg "Bad Request"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /stateless/vpc/{SystemId} [get]
func StatelessGetVPC(c echo.Context) error {
	cblog.Info("call StatelessGetVPC()")

	req := StatelessConnectionRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	conn, err := req.connect()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := conn.CreateVPCHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := handler.GetVPC(cres.IID{SystemId: c.Param("SystemId")})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// statelessDeleteVPC godoc
// @ID stateless-delete-vpc
// @Summary [Stateless Mode] Delete VPC
// @Description Delete a VPC by its CSP-native SystemId. No MetaDB cleanup is needed since nothing was stored.
// @Tags [Stateless Mode] VPC
// @Accept  json
// @Produce  json
// @Param SystemId path string true "The CSP-native SystemId of the VPC"
// @Param StatelessConnectionRequest body restruntime.StatelessConnectionRequest true "Provider/Credential/Region info"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 400 {object} SimpleMsg "Bad Request"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /stateless/vpc/{SystemId} [delete]
func StatelessDeleteVPC(c echo.Context) error {
	cblog.Info("call StatelessDeleteVPC()")

	req := StatelessConnectionRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	conn, err := req.connect()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := conn.CreateVPCHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := handler.DeleteVPC(cres.IID{SystemId: c.Param("SystemId")})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &BooleanInfo{Result: strconv.FormatBool(result)})
}

//================ Stateless Subnet Handler

// StatelessSubnetAddRequest represents the request body for adding a Subnet to a VPC in Stateless Mode.
type StatelessSubnetAddRequest struct {
	StatelessConnectionRequest
	ReqInfo cres.SubnetInfo `json:"ReqInfo" validate:"required"`
}

// statelessAddSubnet godoc
// @ID stateless-add-subnet
// @Summary [Stateless Mode] Add Subnet
// @Description Add a Subnet to an existing VPC, identified by its CSP-native SystemId.
// @Tags [Stateless Mode] VPC
// @Accept  json
// @Produce  json
// @Param SystemId path string true "The CSP-native SystemId of the owning VPC"
// @Param StatelessSubnetAddRequest body restruntime.StatelessSubnetAddRequest true "Request body for adding a Subnet in Stateless Mode"
// @Success 200 {object} cres.VPCInfo "Details of the VPC including the added Subnet"
// @Failure 400 {object} SimpleMsg "Bad Request"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /stateless/vpc/{SystemId}/subnet [post]
func StatelessAddSubnet(c echo.Context) error {
	cblog.Info("call StatelessAddSubnet()")

	req := StatelessSubnetAddRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	conn, err := req.connect()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := conn.CreateVPCHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := handler.AddSubnet(cres.IID{SystemId: c.Param("SystemId")}, req.ReqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// statelessRemoveSubnet godoc
// @ID stateless-remove-subnet
// @Summary [Stateless Mode] Remove Subnet
// @Description Remove a Subnet from a VPC, both identified by their CSP-native SystemId.
// @Tags [Stateless Mode] VPC
// @Accept  json
// @Produce  json
// @Param SystemId path string true "The CSP-native SystemId of the owning VPC"
// @Param SubnetSystemId path string true "The CSP-native SystemId of the Subnet to remove"
// @Param StatelessConnectionRequest body restruntime.StatelessConnectionRequest true "Provider/Credential/Region info"
// @Success 200 {object} BooleanInfo "Result of the remove operation"
// @Failure 400 {object} SimpleMsg "Bad Request"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /stateless/vpc/{SystemId}/subnet/{SubnetSystemId} [delete]
func StatelessRemoveSubnet(c echo.Context) error {
	cblog.Info("call StatelessRemoveSubnet()")

	req := StatelessConnectionRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	conn, err := req.connect()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := conn.CreateVPCHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := handler.RemoveSubnet(cres.IID{SystemId: c.Param("SystemId")}, cres.IID{SystemId: c.Param("SubnetSystemId")})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &BooleanInfo{Result: strconv.FormatBool(result)})
}
