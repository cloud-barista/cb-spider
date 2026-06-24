// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2025.06.

package restruntime

import (
	"net/http"

	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	"github.com/labstack/echo/v4"
)

// SaveTopologyLayoutRequest is the request body for saving a topology layout version.
type SaveTopologyLayoutRequest struct {
	ConnectionName  string `json:"ConnectionName"  validate:"required"`
	VersionName     string `json:"VersionName"     validate:"required"` // user label
	LayoutJSON      string `json:"LayoutJSON"      validate:"required"` // positions JSON
	ThumbnailBase64 string `json:"ThumbnailBase64" validate:"omitempty"` // base64 PNG
}

// ListTopologyLayouts godoc
// @ID list-topology-layouts
// @Summary List all saved topology layout versions
// @Tags [Topology]
// @Produce json
// @Param ConnectionName query string true "Connection config name"
// @Success 200 {array} cmrt.TopologyLayoutInfo "List of saved versions"
// @Router /topology/layout [get]
func ListTopologyLayouts(c echo.Context) error {
	cblog.Info("call ListTopologyLayouts()")
	connName := c.QueryParam("ConnectionName")
	if connName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "ConnectionName query parameter is required")
	}
	list, err := cmrt.ListTopologyLayouts(connName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, list)
}

// GetTopologyLayout godoc
// @ID get-topology-layout
// @Summary Get one saved topology layout version (full LayoutJSON)
// @Tags [Topology]
// @Produce json
// @Param ConnectionName query string true "Connection config name"
// @Param VersionName    query string true "Version label"
// @Success 200 {object} cmrt.TopologyLayoutInfo "Full layout info"
// @Router /topology/layout/version [get]
func GetTopologyLayout(c echo.Context) error {
	cblog.Info("call GetTopologyLayout()")
	connName := c.QueryParam("ConnectionName")
	version  := c.QueryParam("VersionName")
	if connName == "" || version == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "ConnectionName and VersionName are required")
	}
	info, err := cmrt.GetTopologyLayout(connName, version)
	if err != nil {
		return echo.NewHTTPError(http.StatusNotFound, err.Error())
	}
	return c.JSON(http.StatusOK, info)
}

// SaveTopologyLayout godoc
// @ID save-topology-layout
// @Summary Save (upsert) a named topology layout version
// @Tags [Topology]
// @Accept  json
// @Produce json
// @Param SaveTopologyLayoutRequest body restruntime.SaveTopologyLayoutRequest true "Layout to save"
// @Success 200 {object} BooleanInfo "Result"
// @Router /topology/layout [post]
func SaveTopologyLayout(c echo.Context) error {
	cblog.Info("call SaveTopologyLayout()")
	req := SaveTopologyLayoutRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if err := cmrt.SaveTopologyLayout(req.ConnectionName, req.VersionName, req.LayoutJSON, req.ThumbnailBase64); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, &BooleanInfo{Result: "true"})
}

// DeleteTopologyLayout godoc
// @ID delete-topology-layout
// @Summary Delete one named topology layout version
// @Tags [Topology]
// @Produce json
// @Param ConnectionName query string true "Connection config name"
// @Param VersionName    query string true "Version label to delete"
// @Success 200 {object} BooleanInfo "Result"
// @Router /topology/layout [delete]
func DeleteTopologyLayout(c echo.Context) error {
	cblog.Info("call DeleteTopologyLayout()")
	connName := c.QueryParam("ConnectionName")
	version  := c.QueryParam("VersionName")
	if connName == "" || version == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "ConnectionName and VersionName are required")
	}
	if err := cmrt.DeleteTopologyLayout(connName, version); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, &BooleanInfo{Result: "true"})
}
