// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2022.09.

package restruntime

import (
	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	// REST API (echo)
	"net/http"

	"github.com/labstack/echo/v4"

	"strconv"
)

//================ Disk Handler

// DiskRegisterRequest represents the request body for registering a Disk.
type DiskRegisterRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		Name  string `json:"Name" validate:"required" example:"disk-01"`
		Zone  string `json:"Zone" validate:"required" example:"us-east-1b"`
		CSPId string `json:"CSPId" validate:"required" example:"csp-disk-1234"`
	} `json:"ReqInfo" validate:"required"`
}

// registerDisk godoc
// @ID register-disk
// @Summary Register Disk
// @Description Register a new Disk with the specified name, zone, and CSP ID.
// @Tags [Disk Management]
// @Accept  json
// @Produce  json
// @Param DiskRegisterRequest body restruntime.DiskRegisterRequest true "Request body for registering a Disk"
// @Success 200 {object} cres.DiskInfo "Details of the registered Disk"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regdisk [post]
func RegisterDisk(c echo.Context) error {
	cblog.Info("call RegisterDisk()")

	req := DiskRegisterRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// create UserIID
	userIId := cres.IID{req.ReqInfo.Name, req.ReqInfo.CSPId}

	// Call common-runtime API
	result, err := cmrt.RegisterDisk(req.ConnectionName, req.ReqInfo.Zone, userIId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// unregisterDisk godoc
// @ID unregister-disk
// @Summary Unregister Disk
// @Description Unregister a Disk with the specified name.
// @Tags [Disk Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for unregistering a Disk"
// @Param Name path string true "The name of the Disk to unregister"
// @Success 200 {object} BooleanInfo "Result of the unregister operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regdisk/{Name} [delete]
func UnregisterDisk(c echo.Context) error {
	cblog.Info("call UnregisterDisk()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.UnregisterResource(req.ConnectionName, DISK, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// DiskCreateRequest represents the request body for creating a Disk.
type DiskCreateRequest struct {
	ConnectionName  string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	IDTransformMode string `json:"IDTransformMode,omitempty" validate:"omitempty" example:"ON"` // ON: transform CSP ID, OFF: no-transform CSP ID
	ReqInfo         struct {
		Name     string          `json:"Name" validate:"required" example:"disk-01"`
		Zone     string          `json:"Zone,omitempty" validate:"omitempty" example:"us-east-1b"` // target zone for the disk, if not specified, it will be created in the same zone as the Connection.
		DiskType string          `json:"DiskType" validate:"required" example:"gp2"`               // gp2 or default, if not specified, default is used
		DiskSize string          `json:"DiskSize" validate:"required" example:"100"`               // 100 or default, if not specified, default is used (unit is GB)
		TagList  []cres.KeyValue `json:"TagList,omitempty" validate:"omitempty"`
	} `json:"ReqInfo" validate:"required"`
}

// createDisk godoc
// @ID create-disk
// @Summary Create Disk
// @Description Create a new Disk with the specified configuration. 🕷️ [[Concept Guide](https://github.com/cloud-barista/cb-spider/wiki/Disk-and-Driver-API)], [[Snapshot-MyImage,Disk Guide](https://github.com/cloud-barista/cb-spider/wiki/VM-Snapshot,-MyImage-and-Disk-Overview)]
// @Tags [Disk Management]
// @Accept  json
// @Produce  json
// @Param DiskCreateRequest body restruntime.DiskCreateRequest true "Request body for creating a Disk"
// @Success 200 {object} cres.DiskInfo "Details of the created Disk"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /disk [post]
func CreateDisk(c echo.Context) error {
	cblog.Info("call CreateDisk()")

	req := DiskCreateRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Rest RegInfo => Driver ReqInfo
	reqInfo := cres.DiskInfo{
		IId:      cres.IID{req.ReqInfo.Name, req.ReqInfo.Name},
		Zone:     req.ReqInfo.Zone,
		DiskType: req.ReqInfo.DiskType,
		DiskSize: req.ReqInfo.DiskSize,
		TagList:  req.ReqInfo.TagList,
	}

	// Call common-runtime API
	result, err := cmrt.CreateDisk(req.ConnectionName, DISK, reqInfo, req.IDTransformMode)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// DiskListResponse represents the response body for listing Disks.
type DiskListResponse struct {
	Result []*cres.DiskInfo `json:"disk" validate:"required" description:"A list of Disk information"`
}

// listDisk godoc
// @ID list-disk
// @Summary List Disks
// @Description Retrieve a list of Disks associated with a specific connection.
// @Tags [Disk Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list Disks for"
// @Success 200 {object} DiskListResponse "List of Disks"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /disk [get]
func ListDisk(c echo.Context) error {
	cblog.Info("call ListDisk()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.ListDisk(req.ConnectionName, DISK)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	jsonResult := DiskListResponse{
		Result: result,
	}

	return c.JSON(http.StatusOK, &jsonResult)
}

// listAllDisk godoc
// @ID list-all-disk
// @Summary List All Disks in a Connection
// @Description Retrieve a comprehensive list of all Disks associated with a specific connection, <br> including those mapped between CB-Spider and the CSP, <br> only registered in CB-Spider's metadata, <br> and only existing in the CSP.
// @Tags [Disk Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list Disks for"
// @Success 200 {object} AllResourceListResponse "List of all Disks within the specified connection, including Disks in CB-Spider only, CSP only, and mapped between both."
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /alldisk [get]
func ListAllDisk(c echo.Context) error {
	cblog.Info("call ListAllDisk()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	allResourceList, err := cmrt.ListAllResource(req.ConnectionName, DISK)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &allResourceList)
}

// getDisk godoc
// @ID get-disk
// @Summary Get Disk
// @Description Retrieve details of a specific Disk.
// @Tags [Disk Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to get a Disk for"
// @Param Name path string true "The name of the Disk to retrieve"
// @Success 200 {object} cres.DiskInfo "Details of the Disk"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /disk/{Name} [get]
func GetDisk(c echo.Context) error {
	cblog.Info("call GetDisk()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.GetDisk(req.ConnectionName, DISK, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// DiskSizeIncreaseRequest represents the request body for changing the size of a Disk.
type DiskSizeIncreaseRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		Size string `json:"Size" validate:"required" example:"150"`
	} `json:"ReqInfo" validate:"required"`
}

// increaseDiskSize godoc
// @ID increase-disk-size
// @Summary Increase Disk Size
// @Description Increase the size of an existing disk.
// @Tags [Disk Management]
// @Accept  json
// @Produce  json
// @Param DiskSizeIncreaseRequest body restruntime.DiskSizeIncreaseRequest true "Request body for increasing the Disk size"
// @Param Name path string true "The name of the Disk to increase the size for"
// @Success 200 {object} BooleanInfo "Result of the size increase operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /disk/{Name}/size [put]
func IncreaseDiskSize(c echo.Context) error {
	cblog.Info("call IncreaseDiskSize()")

	var req DiskSizeIncreaseRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.ChangeDiskSize(req.ConnectionName, c.Param("Name"), req.ReqInfo.Size)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// deleteDisk godoc
// @ID delete-disk
// @Summary Delete Disk
// @Description Delete a specified Disk.
// @Tags [Disk Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for deleting a Disk"
// @Param Name path string true "The name of the Disk to delete"
// @Param force query string false "Force delete the Disk. ex) true or false(default: false)"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /disk/{Name} [delete]
func DeleteDisk(c echo.Context) error {
	cblog.Info("call DeleteDisk()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.DeleteDisk(req.ConnectionName, DISK, c.Param("Name"), c.QueryParam("force"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// deleteCSPDisk godoc
// @ID delete-csp-disk
// @Summary Delete CSP Disk
// @Description Delete a specified CSP Disk.
// @Tags [Disk Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for deleting a CSP Disk"
// @Param Id path string true "The CSP Disk ID to delete"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cspdisk/{Id} [delete]
func DeleteCSPDisk(c echo.Context) error {
	cblog.Info("call DeleteCSPDisk()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, DISK, c.Param("Id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// DiskAttachRequest represents the request body for attaching a Disk to a VM.
type DiskAttachRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		VMName string `json:"VMName" validate:"required" example:"vm-01"`
	} `json:"ReqInfo" validate:"required"`
}

// attachDisk godoc
// @ID attach-disk
// @Summary Attach Disk
// @Description Attach an existing Disk to a VM.
// @Tags [Disk Management]
// @Accept  json
// @Produce  json
// @Param DiskAttachRequest body restruntime.DiskAttachRequest true "Request body for attaching a Disk to a VM"
// @Param Name path string true "The name of the Disk to attach"
// @Success 200 {object} cres.DiskInfo "Details of the attached Disk"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /disk/{Name}/attach [put]
func AttachDisk(c echo.Context) error {
	cblog.Info("call AttachDisk()")

	var req DiskAttachRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.AttachDisk(req.ConnectionName, c.Param("Name"), req.ReqInfo.VMName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// DiskDetachRequest represents the request body for detaching a Disk from a VM.
type DiskDetachRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		VMName string `json:"VMName" validate:"required" example:"vm-01"`
	} `json:"ReqInfo" validate:"required"`
}

// detachDisk godoc
// @ID detach-disk
// @Summary Detach Disk
// @Description Detach an existing Disk from a VM.
// @Tags [Disk Management]
// @Accept  json
// @Produce  json
// @Param DiskDetachRequest body restruntime.DiskDetachRequest true "Request body for detaching a Disk from a VM"
// @Param Name path string true "The name of the Disk to detach"
// @Success 200 {object} BooleanInfo "Result of the detach operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /disk/{Name}/detach [put]
func DetachDisk(c echo.Context) error {
	cblog.Info("call DetachDisk()")

	var req DiskDetachRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.DetachDisk(req.ConnectionName, c.Param("Name"), req.ReqInfo.VMName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// countAllDisks godoc
// @ID count-all-disks
// @Summary Count All Disks
// @Description Get the total number of Disks across all connections.
// @Tags [Disk Management]
// @Produce  json
// @Success 200 {object} CountResponse "Total count of Disks"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countdisk [get]
func CountAllDisks(c echo.Context) error {
	// Call common-runtime API to get count of Disks
	count, err := cmrt.CountAllDisks()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Prepare JSON result
	jsonResult := CountResponse{
		Count: int(count),
	}

	// Return JSON response
	return c.JSON(http.StatusOK, jsonResult)
}

// countDisksByConnection godoc
// @ID count-disks-by-connection
// @Summary Count Disks by Connection
// @Description Get the total number of Disks for a specific connection.
// @Tags [Disk Management]
// @Produce  json
// @Param ConnectionName path string true "The name of the Connection"
// @Success 200 {object} CountResponse "Total count of Disks for the connection"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countdisk/{ConnectionName} [get]
func CountDisksByConnection(c echo.Context) error {
	// Call common-runtime API to get count of Disks
	count, err := cmrt.CountDisksByConnection(c.Param("ConnectionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Prepare JSON result
	jsonResult := CountResponse{
		Count: int(count),
	}

	// Return JSON response
	return c.JSON(http.StatusOK, jsonResult)
}
