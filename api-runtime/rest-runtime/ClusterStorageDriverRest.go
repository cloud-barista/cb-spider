// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// REST API for CSI storage driver management of a Kubernetes Cluster.
//
// by CB-Spider Team, 2026.10.

package restruntime

import (
	"net/http"
	"strconv"

	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"github.com/labstack/echo/v4"
)

// StorageDriverListResponse represents the response body for listing the storage drivers of a Cluster.
type StorageDriverListResponse struct {
	Result []cres.StorageDriverInfo `json:"storagedriver" validate:"required"`
}

// StorageDriverEnableRequest represents the request body for enabling a storage driver of a Cluster.
type StorageDriverEnableRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		Kind    string `json:"Kind" validate:"required" example:"Block"`       // Block or File
		Version string `json:"Version,omitempty" example:"v1.37.0-eksbuild.1"` // CSP default version if empty
	} `json:"ReqInfo" validate:"required"`
}

// listStorageDriver godoc
// @ID list-storage-driver
// @Summary List Storage Drivers of a Cluster
// @Description List the CSP-managed CSI storage drivers (Block and File) of a Cluster, read through the CSP API.
// @Tags [Cluster Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the Cluster"
// @Param ConnectionName query string true "The name of the Connection to use"
// @Success 200 {object} StorageDriverListResponse "Storage drivers of the Cluster"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to missing parameters"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cluster/{Name}/storagedriver [get]
func ListStorageDriver(c echo.Context) error {
	cblog.Info("call ListStorageDriver()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.ListStorageDriver(req.ConnectionName, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	jsonResult := StorageDriverListResponse{
		Result: result,
	}

	return c.JSON(http.StatusOK, &jsonResult)
}

// enableStorageDriver godoc
// @ID enable-storage-driver
// @Summary Enable a Storage Driver of a Cluster
// @Description Enable a CSP-managed CSI storage driver (Block or File) of a Cluster through the CSP API. <br> Enabling installs the driver components and wires the permissions the driver needs (e.g., AWS IAM Role and Pod Identity). <br> Idempotent: an already enabled or BuiltIn driver is returned as-is. <br> Some CSPs apply the change asynchronously and return Status "Updating"; check the result with the GET API.
// @Tags [Cluster Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the Cluster"
// @Param StorageDriverEnableRequest body restruntime.StorageDriverEnableRequest true "Request body for enabling a storage driver"
// @Success 200 {object} cres.StorageDriverInfo "The storage driver after the request"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to an invalid Kind"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cluster/{Name}/storagedriver [put]
func EnableStorageDriver(c echo.Context) error {
	cblog.Info("call EnableStorageDriver()")

	req := StorageDriverEnableRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if _, err := cmrt.ParseStorageDriverKind(req.ReqInfo.Kind); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	result, err := cmrt.EnableStorageDriver(req.ConnectionName, c.Param("Name"), req.ReqInfo.Kind, req.ReqInfo.Version)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// disableStorageDriver godoc
// @ID disable-storage-driver
// @Summary Disable a Storage Driver of a Cluster
// @Description Disable a CSP-managed CSI storage driver (Block or File) of a Cluster through the CSP API. <br> Only the driver components are removed; volumes, their data and cloud resources are kept. <br> WARNING: existing PVs of the driver can no longer be attached or mounted, so Pods using them fail when rescheduled. <br> BuiltIn drivers cannot be disabled.
// @Tags [Cluster Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the Cluster"
// @Param Kind path string true "The kind of the storage driver: Block or File"
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for disabling a storage driver"
// @Success 200 {object} BooleanInfo "Result of the disable operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to an invalid Kind"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cluster/{Name}/storagedriver/{Kind} [delete]
func DisableStorageDriver(c echo.Context) error {
	cblog.Info("call DisableStorageDriver()")

	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}
	if _, err := cmrt.ParseStorageDriverKind(c.Param("Kind")); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	result, err := cmrt.DisableStorageDriver(req.ConnectionName, c.Param("Name"), c.Param("Kind"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &BooleanInfo{Result: strconv.FormatBool(result)})
}
