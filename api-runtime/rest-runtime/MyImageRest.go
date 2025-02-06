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

//================ MyImage Handler

// MyImageRegisterRequest represents the request body for registering a MyImage.
type MyImageRegisterRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		Name  string `json:"Name" validate:"required" example:"myimage-01"`
		CSPId string `json:"CSPId" validate:"required" example:"csp-myimage-1234"`
	} `json:"ReqInfo" validate:"required"`
}

// registerMyImage godoc
// @ID register-myimage
// @Summary Register MyImage
// @Description Register a new MyImage with the specified name and CSP ID.
// @Tags [MyImage Management]
// @Accept  json
// @Produce  json
// @Param MyImageRegisterRequest body restruntime.MyImageRegisterRequest true "Request body for registering a MyImage"
// @Success 200 {object} cres.MyImageInfo "Details of the registered MyImage"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regmyimage [post]
func RegisterMyImage(c echo.Context) error {
	cblog.Info("call RegisterMyImage()")

	req := MyImageRegisterRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// create UserIID
	userIId := cres.IID{req.ReqInfo.Name, req.ReqInfo.CSPId}

	// Call common-runtime API
	result, err := cmrt.RegisterMyImage(req.ConnectionName, userIId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// unregisterMyImage godoc
// @ID unregister-myimage
// @Summary Unregister MyImage
// @Description Unregister a MyImage with the specified name.
// @Tags [MyImage Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for unregistering a MyImage"
// @Param Name path string true "The name of the MyImage to unregister"
// @Success 200 {object} BooleanInfo "Result of the unregister operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regmyimage/{Name} [delete]
func UnregisterMyImage(c echo.Context) error {
	cblog.Info("call UnregisterMyImage()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.UnregisterResource(req.ConnectionName, MYIMAGE, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// MyImageSnapshotRequest represents the request body for creating a MyImage snapshot.
type MyImageSnapshotRequest struct {
	ConnectionName  string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	IDTransformMode string `json:"IDTransformMode,omitempty" validate:"omitempty" example:"ON"` // ON: transform CSP ID, OFF: no-transform CSP ID
	ReqInfo         struct {
		Name     string          `json:"Name" validate:"required" example:"myimage-01"`
		SourceVM string          `json:"SourceVM" validate:"required" example:"vm-01"`
		TagList  []cres.KeyValue `json:"TagList,omitempty" validate:"omitempty"`
	} `json:"ReqInfo" validate:"required"`
}

// snapshotVM godoc
// @ID create-myimage
// @Summary Snapshot VM
// @Description Create a new MyImage snapshot from a specified VM. 🕷️ [[Concept Guide](https://github.com/cloud-barista/cb-spider/wiki/MyImage-and-Driver-API)], [[Snapshot-MyImage,Disk Guide](https://github.com/cloud-barista/cb-spider/wiki/VM-Snapshot,-MyImage-and-Disk-Overview)]
// @Tags [MyImage Management]
// @Accept  json
// @Produce  json
// @Param MyImageSnapshotRequest body restruntime.MyImageSnapshotRequest true "Request body for creating a MyImage snapshot"
// @Success 200 {object} cres.MyImageInfo "Details of the created MyImage snapshot"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /myimage [post]
func SnapshotVM(c echo.Context) error {
	cblog.Info("call SnapshotVM()")

	req := MyImageSnapshotRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Rest RegInfo => Driver ReqInfo
	reqInfo := cres.MyImageInfo{
		IId:      cres.IID{req.ReqInfo.Name, req.ReqInfo.Name},
		SourceVM: cres.IID{req.ReqInfo.SourceVM, req.ReqInfo.SourceVM},
		TagList:  req.ReqInfo.TagList,
	}

	// Call common-runtime API
	result, err := cmrt.SnapshotVM(req.ConnectionName, MYIMAGE, reqInfo, req.IDTransformMode)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// MyImageListResponse represents the response body for listing MyImages.
type MyImageListResponse struct {
	Result []*cres.MyImageInfo `json:"myImage" validate:"required" description:"A list of MyImage information"`
}

// listMyImage godoc
// @ID list-myimage
// @Summary List MyImages
// @Description Retrieve a list of MyImages associated with a specific connection.
// @Tags [MyImage Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list MyImages for"
// @Success 200 {object} MyImageListResponse "List of MyImages"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /myimage [get]
func ListMyImage(c echo.Context) error {
	cblog.Info("call ListMyImage()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.ListMyImage(req.ConnectionName, MYIMAGE)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	jsonResult := MyImageListResponse{
		Result: result,
	}

	return c.JSON(http.StatusOK, &jsonResult)
}

// listAllMyImage godoc
// @ID list-all-myimage
// @Summary List All MyImages in a Connection
// @Description Retrieve a comprehensive list of all MyImages associated with a specific connection, <br> including those mapped between CB-Spider and the CSP, <br> only registered in CB-Spider's metadata, <br> and only existing in the CSP.
// @Tags [MyImage Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list MyImages for"
// @Success 200 {object} AllResourceListResponse "List of all MyImages within the specified connection, including MyImages in CB-Spider only, CSP only, and mapped between both."
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /allmyimage [get]
func ListAllMyImage(c echo.Context) error {
	cblog.Info("call ListAllMyImage()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	allResourceList, err := cmrt.ListAllResource(req.ConnectionName, MYIMAGE)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &allResourceList)
}

// listAllMyImageInfo godoc
// @ID list-all-myimage-info
// @Summary List All MyImage Info
// @Description Retrieve a comprehensive list of all MyImage information associated with a specific connection, <br> including those mapped between CB-Spider and the CSP, <br> only registered in CB-Spider's metadata, <br> and only existing in the CSP.
// @Tags [MyImage Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list MyImage information for"
// @Success 200 {object} AllResourceInfoListResponse "List of all MyImage information within the specified connection, including MyImage information in CB-Spider only, CSP only, and mapped between both."
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /allmyimageinfo [get]
func ListAllMyImageInfo(c echo.Context) error { return listAllResourceInfo(c, cres.MYIMAGE) }

// getMyImage godoc
// @ID get-myimage
// @Summary Get MyImage
// @Description Retrieve details of a specific MyImage.
// @Tags [MyImage Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to get a MyImage for"
// @Param Name path string true "The name of the MyImage to retrieve"
// @Success 200 {object} cres.MyImageInfo "Details of the MyImage"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /myimage/{Name} [get]
func GetMyImage(c echo.Context) error {
	cblog.Info("call GetMyImage()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.GetMyImage(req.ConnectionName, MYIMAGE, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// deleteMyImage godoc
// @ID delete-myimage
// @Summary Delete MyImage
// @Description Delete a specified MyImage.
// @Tags [MyImage Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for deleting a MyImage"
// @Param Name path string true "The name of the MyImage to delete"
// @Param force query string false "Force delete the MyImage. ex) true or false(default: false)"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /myimage/{Name} [delete]
func DeleteMyImage(c echo.Context) error {
	cblog.Info("call DeleteMyImage()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.DeleteMyImage(req.ConnectionName, MYIMAGE, c.Param("Name"), c.QueryParam("force"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// deleteCSPMyImage godoc
// @ID delete-csp-myimage
// @Summary Delete CSP MyImage
// @Description Delete a specified CSP MyImage.
// @Tags [MyImage Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for deleting a CSP MyImage"
// @Param Id path string true "The CSP MyImage ID to delete"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cspmyimage/{Id} [delete]
func DeleteCSPMyImage(c echo.Context) error {
	cblog.Info("call DeleteCSPMyImage()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, MYIMAGE, c.Param("Id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// countAllMyImages godoc
// @ID count-all-myimage
// @Summary Count All MyImages
// @Description Get the total number of MyImages across all connections.
// @Tags [MyImage Management]
// @Produce  json
// @Success 200 {object} CountResponse "Total count of MyImages"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countmyimage [get]
func CountAllMyImages(c echo.Context) error {
	// Call common-runtime API to get count of MyImages
	count, err := cmrt.CountAllMyImages()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Prepare JSON result
	var jsonResult CountResponse
	jsonResult.Count = int(count)

	// Return JSON response
	return c.JSON(http.StatusOK, jsonResult)
}

// countMyImagesByConnection godoc
// @ID count-myimage-by-connection
// @Summary Count MyImages by Connection
// @Description Get the total number of MyImages for a specific connection.
// @Tags [MyImage Management]
// @Produce  json
// @Param ConnectionName path string true "The name of the Connection"
// @Success 200 {object} CountResponse "Total count of MyImages for the connection"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countmyimage/{ConnectionName} [get]
func CountMyImagesByConnection(c echo.Context) error {
	// Call common-runtime API to get count of MyImages
	count, err := cmrt.CountMyImagesByConnection(c.Param("ConnectionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Prepare JSON result
	var jsonResult CountResponse
	jsonResult.Count = int(count)

	// Return JSON response
	return c.JSON(http.StatusOK, jsonResult)
}
