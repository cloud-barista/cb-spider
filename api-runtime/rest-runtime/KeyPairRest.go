// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.

package restruntime

import (
	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	// REST API (echo)
	"net/http"

	"github.com/labstack/echo/v4"

	"strconv"
)

//================ KeyPair Handler

// KeyRegisterRequest represents the request body for registering a KeyPair.
type KeyRegisterRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		Name  string `json:"Name" validate:"required" example:"keypair-01"`
		CSPId string `json:"CSPId" validate:"required" example:"csp-key-1234"`
	} `json:"ReqInfo" validate:"required"`
}

// registerKey godoc
// @ID register-key
// @Summary Register KeyPair
// @Description Register a new KeyPair with the specified name and CSP ID.
// @Tags [KeyPair Management]
// @Accept  json
// @Produce  json
// @Param KeyRegisterRequest body restruntime.KeyRegisterRequest true "Request body for registering a KeyPair"
// @Success 200 {object} cres.KeyPairInfo "Details of the registered KeyPair"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regkeypair [post]
func RegisterKey(c echo.Context) error {
	cblog.Info("call RegisterKey()")

	req := KeyRegisterRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// create UserIID
	userIId := cres.IID{NameId: req.ReqInfo.Name, SystemId: req.ReqInfo.CSPId}

	// Call common-runtime API
	result, err := cmrt.RegisterKey(req.ConnectionName, userIId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// unregisterKey godoc
// @ID unregister-key
// @Summary Unregister KeyPair
// @Description Unregister a KeyPair with the specified name.
// @Tags [KeyPair Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for unregistering a KeyPair"
// @Param Name path string true "The name of the KeyPair to unregister"
// @Success 200 {object} BooleanInfo "Result of the unregister operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regkeypair/{Name} [delete]
func UnregisterKey(c echo.Context) error {
	cblog.Info("call UnregisterKey()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.UnregisterResource(req.ConnectionName, KEY, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// KeyCreateRequest represents the request body for creating a KeyPair.
type KeyCreateRequest struct {
	ConnectionName  string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	IDTransformMode string `json:"IDTransformMode,omitempty" validate:"omitempty" example:"ON"` // ON: transform CSP ID, OFF: no-transform CSP ID
	ReqInfo         struct {
		Name    string          `json:"Name" validate:"required" example:"keypair-01"`
		TagList []cres.KeyValue `json:"TagList,omitempty" validate:"omitempty"`
	} `json:"ReqInfo" validate:"required"`
}

// createKey godoc
// @ID create-key
// @Summary Create KeyPair
// @Description Create a new KeyPair with the specified configurations.
// @Tags [KeyPair Management]
// @Accept  json
// @Produce  json
// @Param KeyCreateRequest body restruntime.KeyCreateRequest true "Request body for creating a KeyPair"
// @Success 200 {object} cres.KeyPairInfo "Details of the created KeyPair"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /keypair [post]
func CreateKey(c echo.Context) error {
	cblog.Info("call CreateKey()")

	var req KeyCreateRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Rest RegInfo => Driver ReqInfo
	reqInfo := cres.KeyPairReqInfo{
		IId:     cres.IID{NameId: req.ReqInfo.Name, SystemId: ""},
		TagList: req.ReqInfo.TagList,
	}

	// Call common-runtime API
	result, err := cmrt.CreateKey(req.ConnectionName, KEY, reqInfo, req.IDTransformMode)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// KeyListResponse represents the response body for listing KeyPairs.
type KeyListResponse struct {
	Result []*cres.KeyPairInfo `json:"keypair"`
}

// listKey godoc
// @ID list-key
// @Summary List KeyPairs
// @Description Retrieve a list of KeyPairs associated with a specific connection.
// @Tags [KeyPair Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list KeyPairs for"
// @Success 200 {object} restruntime.KeyListResponse "List of KeyPairs"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /keypair [get]
func ListKey(c echo.Context) error {
	cblog.Info("call ListKey()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.ListKey(req.ConnectionName, KEY)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult KeyListResponse
	jsonResult.Result = result
	return c.JSON(http.StatusOK, &jsonResult)
}

// listAllKey godoc
// @ID list-all-key
// @Summary List All KeyPairs
// @Description Retrieve a list of all KeyPairs across all connections.
// @Tags [KeyPair Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection"
// @Success 200 {object} AllResourceListResponse "List of all KeyPairs with their respective lists"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /allkeypair [get]
func ListAllKey(c echo.Context) error {
	cblog.Info("call ListAllKey()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	allResourceList, err := cmrt.ListAllResource(req.ConnectionName, KEY)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &allResourceList)
}

// getKey godoc
// @ID get-key
// @Summary Get KeyPair
// @Description Retrieve details of a specific KeyPair.
// @Tags [KeyPair Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to get a KeyPair for"
// @Param Name path string true "The name of the KeyPair to retrieve"
// @Success 200 {object} cres.KeyPairInfo "Details of the KeyPair"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /keypair/{Name} [get]
func GetKey(c echo.Context) error {
	cblog.Info("call GetKey()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.GetKey(req.ConnectionName, KEY, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// deleteKey godoc
// @ID delete-key
// @Summary Delete KeyPair
// @Description Delete a specified KeyPair.
// @Tags [KeyPair Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for deleting a KeyPair"
// @Param Name path string true "The name of the KeyPair to delete"
// @Param force query string false "Force delete the KeyPair"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /keypair/{Name} [delete]
func DeleteKey(c echo.Context) error {
	cblog.Info("call DeleteKey()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.DeleteKey(req.ConnectionName, KEY, c.Param("Name"), c.QueryParam("force"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// deleteCSPKey godoc
// @ID delete-csp-key
// @Summary Delete CSP KeyPair
// @Description Delete a specified CSP KeyPair.
// @Tags [KeyPair Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for deleting a CSP KeyPair"
// @Param Id path string true "The CSP KeyPair ID to delete"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cspkeypair/{Id} [delete]
func DeleteCSPKey(c echo.Context) error {
	cblog.Info("call DeleteCSPKey()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, KEY, c.Param("Id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// countAllKeys godoc
// @ID count-all-keys
// @Summary Count All KeyPairs
// @Description Get the total number of KeyPairs across all connections.
// @Tags [KeyPair Management]
// @Produce  json
// @Success 200 {object} CountResponse "Total count of KeyPairs"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countkeypair [get]
func CountAllKeys(c echo.Context) error {
	// Call common-runtime API to get count of Keys
	count, err := cmrt.CountAllKeys()
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

// countKeysByConnection godoc
// @ID count-keys-by-connection
// @Summary Count KeyPairs by Connection
// @Description Get the total number of KeyPairs for a specific connection.
// @Tags [KeyPair Management]
// @Produce  json
// @Param ConnectionName path string true "The name of the Connection"
// @Success 200 {object} CountResponse "Total count of KeyPairs for the connection"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countkeypair/{ConnectionName} [get]
func CountKeysByConnection(c echo.Context) error {
	// Call common-runtime API to get count of Keys
	count, err := cmrt.CountKeysByConnection(c.Param("ConnectionName"))
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
