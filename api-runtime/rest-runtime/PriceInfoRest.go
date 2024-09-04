// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2023.09.

package restruntime

import (
	"encoding/json"
	"net/http"

	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

// ================ PriceInfo Handler

// ProductFamilyListResponse represents the response body structure for the ListProductFamily API.
type ProductFamilyListResponse struct {
	Result []string `json:"productfamily" validate:"required"`
}

// listProductFamily godoc
// @ID list-product-family
// @Summary List Product Families
// @Description Retrieve a list of Product Families associated with a specific connection and region.
// @Tags [Cloud Metadata] Price
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list Product Families for"
// @Param RegionName path string true "The name of the Region to list Product Families for"
// @Success 200 {object} ProductFamilyListResponse "List of Product Families"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /productfamily/{RegionName} [get]
func ListProductFamily(c echo.Context) error {
	cblog.Info("call ListProductFamily()")

	req := ConnectionRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.ListProductFamily(req.ConnectionName, c.Param("RegionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult ProductFamilyListResponse
	jsonResult.Result = result
	return c.JSON(http.StatusOK, &jsonResult)
}

// PriceInfoRequest represents the request body structure for the GetPriceInfo API.
type PriceInfoRequest struct {
	ConnectionName string          `json:"connectionName" validate:"required" description:"The name of the Connection to get Price Information for"`
	FilterList     []cres.KeyValue `json:"filterList" description:"A list of filters to apply to the price information request"`
}

// PriceInfoResponse represents the response body structure for the GetPriceInfo API.
type PriceInfoResponse struct {
	cres.CloudPriceData `json:",inline" description:"Price information details"`
}

// getPriceInfo godoc
// @ID get-price-info
// @Summary Get Price Information
// @Description Retrieve price details of a specific Product Family in a specific Region. <br> * example body: {"connectionName":"aws-connection","FilterList":[{"Key":"instanceType","Value":"t2.micro"}]}
// @Tags [Cloud Metadata] Price
// @Accept  json
// @Produce  json
// @Param ProductFamily path string true "The name of the Product Family to retrieve price information for"
// @Param RegionName path string true "The name of the Region to retrieve price information for"
// @Param PriceInfoRequest body PriceInfoRequest false "The request body containing additional filters for price information"
// @Success 200 {object} PriceInfoResponse "Price Information Details"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /priceinfo/{ProductFamily}/{RegionName} [post]
func GetPriceInfo(c echo.Context) error {
	cblog.Info("call GetPriceInfo()")

	var req PriceInfoRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.GetPriceInfo(req.ConnectionName, c.Param("ProductFamily"), c.Param("RegionName"), req.FilterList)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var response PriceInfoResponse
	json.Unmarshal([]byte(result), &response)
	return c.JSON(http.StatusOK, response)
}
