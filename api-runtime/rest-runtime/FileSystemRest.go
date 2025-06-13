// Cloud Control Manager's Rest Runtime of CB-Spider.
// REST API implementation for FileSystemHandler interface
// by CB-Spider Team

package restruntime

import (
	"net/http"
	"strconv"

	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

// FileSystemCreateRequest represents the request body for creating a FileSystem.
type FileSystemCreateRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		Name                  string          `json:"Name" validate:"required" example:"efs-01"`
		Zone                  string          `json:"Zone" validate:"required" example:"us-east-1a"`
		FileSystemType        string          `json:"FileSystemType" example:"efs"`
		PerformanceMode       string          `json:"PerformanceMode" example:"generalPurpose"`
		ThroughputMode        string          `json:"ThroughputMode" example:"bursting"`
		ProvisionedThroughput string          `json:"ProvisionedThroughput" example:"0"`
		Encrypted             bool            `json:"Encrypted" example:"false"`
		KmsKeyId              string          `json:"KmsKeyId,omitempty"`
		TagList               []cres.KeyValue `json:"TagList,omitempty"`
	} `json:"ReqInfo" validate:"required"`
}

// CreateFileSystem godoc
// @ID create-filesystem
// @Summary Create FileSystem
// @Description Create a new FileSystem with the specified configuration.
// @Tags [FileSystem Management]
// @Accept  json
// @Produce  json
// @Param FileSystemCreateRequest body restruntime.FileSystemCreateRequest true "Request body for creating a FileSystem"
// @Success 200 {object} cres.FileSystemInfo "Details of the created FileSystem"
// @Failure 400 {object} SimpleMsg "Bad Request"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /filesystem [post]
func CreateFileSystem(c echo.Context) error {
	cblog.Info("call CreateFileSystem()")

	req := FileSystemCreateRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	reqInfo := cres.FileSystemInfo{
		IId:            cres.IID{NameId: req.ReqInfo.Name, SystemId: ""},
		Zone:           req.ReqInfo.Zone,
		FileSystemType: cres.FileSystemType(req.ReqInfo.FileSystemType),
		TagList:        req.ReqInfo.TagList,
	}

	result, err := cmrt.CreateFileSystem(req.ConnectionName, reqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// ListFileSystem godoc
// @Summary List FileSystems
// @Tags [FileSystem Management]
// @Produce json
// @Param ConnectionName query string true "Connection Name"
// @Success 200 {array} cres.FileSystemInfo
// @Router /filesystem [get]
func ListFileSystem(c echo.Context) error {
	conn := c.QueryParam("ConnectionName")
	result, err := cmrt.ListFileSystem(conn)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, result)
}

// GetFileSystem godoc
// @Summary Get FileSystem
// @Tags [FileSystem Management]
// @Produce json
// @Param ConnectionName query string true "Connection Name"
// @Param Name path string true "FileSystem Name"
// @Success 200 {object} cres.FileSystemInfo
// @Router /filesystem/{Name} [get]
func GetFileSystem(c echo.Context) error {
	conn := c.QueryParam("ConnectionName")
	name := c.Param("Name")
	result, err := cmrt.GetFileSystem(conn, name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, result)
}

// DeleteFileSystem godoc
// @Summary Delete FileSystem
// @Tags [FileSystem Management]
// @Accept json
// @Produce json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Connection Name"
// @Param Name path string true "FileSystem Name"
// @Success 200 {object} BooleanInfo
// @Router /filesystem/{Name} [delete]
func DeleteFileSystem(c echo.Context) error {
	var req ConnectionRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	name := c.Param("Name")
	result, err := cmrt.DeleteFileSystem(req.ConnectionName, name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, BooleanInfo{Result: strconv.FormatBool(result)})
}

type AddMountTargetRequest struct {
	ConnectionName string               `json:"ConnectionName" validate:"required" example:"aws-connection"`
	Target         cres.MountTargetInfo `json:"Target" validate:"required"`
}

// AddMountTarget godoc
// @Summary Add Mount Target to FileSystem
// @Tags [FileSystem Management]
// @Accept json
// @Produce json
// @Param Name path string true "FileSystem Name"
// @Param AddRequest body restruntime.AddMountTargetRequest true "Add MountTarget Info"
//
// @Success 200 {object} cres.FileSystemInfo
// @Router /filesystem/{Name}/mount-target [post]
func AddMountTarget(c echo.Context) error {
	name := c.Param("Name")
	var req struct {
		ConnectionName string
		Target         cres.MountTargetInfo
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	result, err := cmrt.AddMountTarget(req.ConnectionName, name, req.Target)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, result)
}

type RemoveMountTargetRequest struct {
	ConnectionName string               `json:"ConnectionName" validate:"required" example:"aws-connection"`
	Target         cres.MountTargetInfo `json:"Target" validate:"required"`
}

// RemoveMountTarget godoc
// @Summary Remove Mount Target from FileSystem
// @Tags [FileSystem Management]
// @Accept json
// @Produce json
// @Param Name path string true "FileSystem Name"
// @Param RemoveRequest body restruntime.RemoveMountTargetRequest true "Remove MountTarget Info"
//
// @Success 200 {object} BooleanInfo
// @Router /filesystem/{Name}/mount-target [delete]
func RemoveMountTarget(c echo.Context) error {
	name := c.Param("Name")
	var req struct {
		ConnectionName string
		Target         cres.MountTargetInfo
	}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	result, err := cmrt.RemoveMountTarget(req.ConnectionName, name, req.Target)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, BooleanInfo{Result: strconv.FormatBool(result)})
}

// ListMountTarget godoc
// @Summary List Mount Targets of FileSystem
// @Tags [FileSystem Management]
// @Produce json
// @Param ConnectionName query string true "Connection Name"
// @Param Name path string true "FileSystem Name"
// @Success 200 {array} cres.MountTargetInfo
// @Router /filesystem/{Name}/mount-target [get]
func ListMountTarget(c echo.Context) error {
	conn := c.QueryParam("ConnectionName")
	name := c.Param("Name")
	result, err := cmrt.ListMountTarget(conn, name)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, result)
}
