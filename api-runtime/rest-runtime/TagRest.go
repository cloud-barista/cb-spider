// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista

package restruntime

import (
	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	// REST API (echo)
	"net/http"

	"strconv"

	"github.com/labstack/echo/v4"
)

//================ Tag Handler

type TagAddRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		ResourceType cres.RSType   `json:"ResourceType" validate:"required" example:"VPC"`
		ResourceName string        `json:"ResourceName" validate:"required" example:"vpc-01"`
		Tag          cres.KeyValue `json:"Tag" validate:"required"`
	} `json:"ReqInfo" validate:"required"`
}

// addTag godoc
// @ID add-tag
// @Summary Add Tag
// @Description Add a tag to a specified resource.
// @Tags [Tag Management]
// @Accept  json
// @Produce  json
// @Param TagAddRequest body restruntime.TagAddRequest true "Request body for adding a tag"
// @Success 200 {object} cres.KeyValue "Details of the added tag"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /tag [post]
func AddTag(c echo.Context) error {
	cblog.Info("call AddTag()")

	req := TagAddRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.AddTag(req.ConnectionName, req.ReqInfo.ResourceType, req.ReqInfo.ResourceName, req.ReqInfo.Tag)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// listTag godoc
// @ID list-tag
// @Summary List Tags
// @Description Retrieve a list of tags for a specified resource.
// @Tags [Tag Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "Connection Name. ex) aws-connection"
// @Param ResourceType query string true "Resource Type. ex) VPC"
// @Param ResourceName query string true "Resource Name. ex) vpc-01"
// @Success 200 {object} []cres.KeyValue "List of tags"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /tag [get]
func ListTag(c echo.Context) error {
	cblog.Info("call ListTag()")

	// Retrieve query parameters
	connectionName := c.QueryParam("ConnectionName")
	resourceType := c.QueryParam("ResourceType")
	resourceName := c.QueryParam("ResourceName")

	if connectionName == "" || resourceType == "" || resourceName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required query parameters")
	}

	// Convert resourceType to cres.RSType
	rType, err := cres.StringToRSType(resourceType)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	// Log the resource type using RSTypeString()
	cblog.Infof("Listing tags for resource type: %s", cres.RSTypeString(rType))

	// Call common-runtime API
	result, err := cmrt.ListTag(connectionName, rType, resourceName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result       []cres.KeyValue `json:"tag"`
		ResourceType string          `json:"resourceType"`
	}
	jsonResult.Result = result
	jsonResult.ResourceType = cres.RSTypeString(rType) // Include the resource type in a human-readable format

	return c.JSON(http.StatusOK, &jsonResult)
}

// getTag godoc
// @ID get-tag
// @Summary Get Tag
// @Description Retrieve a specific tag for a specified resource.
// @Tags [Tag Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "Connection Name. ex) aws-connection"
// @Param ResourceType query string true "Resource Type. ex) VPC"
// @Param ResourceName query string true "Resource Name. ex) vpc-01"
// @Param Key path string true "The key of the tag to retrieve"
// @Success 200 {object} cres.KeyValue "Details of the tag"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameters"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /tag/{Key} [get]
func GetTag(c echo.Context) error {
	cblog.Info("call GetTag()")

	// Retrieve query parameters
	connectionName := c.QueryParam("ConnectionName")
	resourceType := c.QueryParam("ResourceType")
	resourceName := c.QueryParam("ResourceName")
	tagKey := c.Param("Key")

	if connectionName == "" || resourceType == "" || resourceName == "" || tagKey == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "Missing required query parameters")
	}

	// Convert resourceType to cres.RSType
	rType, err := cres.StringToRSType(resourceType)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid resource type")
	}

	// Call common-runtime API
	result, err := cmrt.GetTag(connectionName, rType, resourceName, tagKey)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// tagRemoveReq represents the request body for removing a Tag from a Resource.
type TagRemoveRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		ResourceType cres.RSType `json:"ResourceType" validate:"required" example:"VPC"`
		ResourceName string      `json:"ResourceName" validate:"required" example:"vpc-01"`
	} `json:"ReqInfo" validate:"required"`
}

// removeTag godoc
// @ID remove-tag
// @Summary Remove Tag
// @Description Remove a specific tag from a specified resource.
// @Tags [Tag Management]
// @Accept  json
// @Produce  json
// @Param TagRemoveRequest body restruntime.TagRemoveRequest true "Request body for removing a specific tag"
// @Param Key path string true "The key of the tag to remove"
// @Success 200 {object} BooleanInfo "Result of the remove operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /tag/{Key} [delete]
func RemoveTag(c echo.Context) error {
	cblog.Info("call RemoveTag()")

	req := TagRemoveRequest{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.RemoveTag(req.ConnectionName, req.ReqInfo.ResourceType, req.ReqInfo.ResourceName, c.Param("Key"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

type tagFindReq struct {
	ConnectionName string
	ReqInfo        struct {
		ResourceType cres.RSType
		Keyword      string
	}
}

func FindTag(c echo.Context) error {
	cblog.Info("call FindTag()")

	req := tagFindReq{}
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.FindTag(req.ConnectionName, req.ReqInfo.ResourceType, req.ReqInfo.Keyword)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result []*cres.TagInfo `json:"tag"`
	}
	jsonResult.Result = result
	return c.JSON(http.StatusOK, &jsonResult)
}
