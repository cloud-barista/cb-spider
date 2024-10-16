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
	"net/url"

	"github.com/labstack/echo/v4"
)

// ================ Image Handler

// ImageListResponse represents the response body structure for the ListImage API.
type ImageListResponse struct {
	Result []*cres.ImageInfo `json:"image" validate:"required" description:"A list of public images"`
}

// listImage godoc
// @ID list-image
// @Summary List Public Images
// @Description Retrieve a list of Public Images associated with a specific connection. 🕷️ [[User Guide](https://github.com/cloud-barista/cb-spider/wiki/How-to-get-Image-List-with-REST-API)]
// @Tags [Cloud Metadata] Public VM Image
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list Public Images for"
// @Success 200 {object} ImageListResponse "List of Public Images"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vmimage [get]
func ListImage(c echo.Context) error {
	cblog.Info("call ListImage()")

	var req struct {
		ConnectionName string
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.ListImage(req.ConnectionName, IMAGE)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult ImageListResponse
	jsonResult.Result = result
	return c.JSON(http.StatusOK, &jsonResult)
}

// getImage godoc
// @ID get-image
// @Summary Get Public Image
// @Description Retrieve details of a specific Public Image. 🕷️ [[User Guide](https://github.com/cloud-barista/cb-spider/wiki/How-to-get-Image-List-with-REST-API)]
// @Tags [Cloud Metadata] Public VM Image
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to get a specific Public Image for"
// @Param Name path string true "The name of the Public Image to retrieve"
// @Success 200 {object} cres.ImageInfo "Details of the Public Image"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vmimage/{Name} [get]
func GetImage(c echo.Context) error {
	cblog.Info("call GetImage()")

	var req struct {
		ConnectionName string
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	encodedImageName := c.Param("Name")
	decodedImageName, err := url.QueryUnescape(encodedImageName)
	if err != nil {
		cblog.Fatal(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := cmrt.GetImage(req.ConnectionName, IMAGE, decodedImageName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}
