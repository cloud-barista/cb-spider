// Cloud Info Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2019.09.

package restruntime

import (
	"strconv"

	im "github.com/cloud-barista/cb-spider/cloud-info-manager"
	ccim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/credential-info-manager"
	dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"
	rim "github.com/cloud-barista/cb-spider/cloud-info-manager/region-info-manager"

	// REST API (echo)
	"net/http"

	"github.com/labstack/echo/v4"

	"fmt"
	"io"
	"os"
)

// ListCloudOSResponse represents the response body for listing Cloud OS.
type ListCloudOSResponse struct {
	Result []string `json:"cloudos" validate:"required" example:"[\"AWS\", \"GCP\"]"`
}

// ================ List of support CloudOS

// listCloudOS godoc
// @ID list-cloudos
// @Summary List Cloud OS
// @Description Retrieve a list of supported Cloud OS.
// @Tags [Cloud Info Management] CloudOS Info
// @Produce  json
// @Success 200 {object} ListCloudOSResponse "List of supported Cloud OS"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cloudos [get]
func ListCloudOS(c echo.Context) error {
	// cblog.Info("call ListCloudOS()")

	infoList := im.ListCloudOS()

	var jsonResult ListCloudOSResponse
	if infoList == nil {
		infoList = []string{}
	}
	jsonResult.Result = infoList
	return c.JSON(http.StatusOK, &jsonResult)
}

// ================ CloudOS Metainfo

// getCloudOSMetaInfo godoc
// @ID get-cloudos-metainfo
// @Summary Get Cloud OS Meta Info
// @Description Retrieve metadata information for a specific Cloud OS.
// @Tags [Cloud Info Management] CloudOS Info
// @Produce  json
// @Param CloudOSName path string true "The name of the Cloud OS"
// @Success 200 {object} im.CloudOSMetaInfo "Cloud OS Meta Info"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cloudos/metainfo/{CloudOSName} [get]
func GetCloudOSMetaInfo(c echo.Context) error {
	cblog.Info("call GetCloudOSMetaInfo()")

	cldMetainfo, err := im.GetCloudOSMetaInfo(c.Param("CloudOSName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &cldMetainfo)
}

// ================ CloudDriver Handler

// registerCloudDriver godoc
// @ID register-driver
// @Summary Register Cloud Driver
// @Description Register a new Cloud Driver. 🕷️ [[User Guide](https://github.com/cloud-barista/cb-spider/wiki/features-and-usages#1-cloud-driver-%EC%A0%95%EB%B3%B4-%EB%93%B1%EB%A1%9D-%EB%B0%8F-%EA%B4%80%EB%A6%AC)]
// @Tags [Cloud Info Management] Driver Info
// @Accept  json
// @Produce  json
// @Param CloudDriverInfo body dim.CloudDriverInfo true "Request body for registering a Cloud Driver"
// @Success 200 {object} dim.CloudDriverInfo "Details of the registered Cloud Driver"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /driver [post]
func RegisterCloudDriver(c echo.Context) error {
	cblog.Info("call RegisterCloudDriver()")
	req := &dim.CloudDriverInfo{}
	if err := c.Bind(req); err != nil {
		cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	cldinfoList, err := dim.RegisterCloudDriverInfo(*req)
	if err != nil {
		cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &cldinfoList)
}

// ListCloudDriverResponse represents the response body for listing Cloud Drivers.
type ListCloudDriverResponse struct {
	Result []*dim.CloudDriverInfo `json:"driver" validate:"required"`
}

// listCloudDriver godoc
// @ID list-driver
// @Summary List Cloud Drivers
// @Description Retrieve a list of registered Cloud Drivers.
// @Tags [Cloud Info Management] Driver Info
// @Produce  json
// @Param provider query string false "The name of the provider to filter the Cloud Drivers by"
// @Success 200 {object} ListCloudDriverResponse "List of Cloud Drivers"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /driver [get]
func ListCloudDriver(c echo.Context) error {
	cblog.Info("call ListCloudDriver()")

	var providerName string
	providerName = c.QueryParam("provider")
	if providerName == "" {
		providerName = c.QueryParam("ProviderName")
	}

	infoList := []*dim.CloudDriverInfo{}
	var err error
	if providerName != "" {
		infoList, err = dim.ListCloudDriverByProvider(providerName)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	} else {
		infoList, err = dim.ListCloudDriver()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	var jsonResult ListCloudDriverResponse
	if infoList == nil {
		infoList = []*dim.CloudDriverInfo{}
	}
	jsonResult.Result = infoList
	return c.JSON(http.StatusOK, &jsonResult)
}

// getCloudDriver godoc
// @ID get-driver
// @Summary Get Cloud Driver
// @Description Retrieve details of a specific Cloud Driver.
// @Tags [Cloud Info Management] Driver Info
// @Produce  json
// @Param DriverName path string true "The name of the Cloud Driver"
// @Success 200 {object} dim.CloudDriverInfo "Details of the Cloud Driver"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /driver/{DriverName} [get]
func GetCloudDriver(c echo.Context) error {
	cblog.Info("call GetCloudDriver()")

	cldinfo, err := dim.GetCloudDriver(c.Param("DriverName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &cldinfo)
}

// unregisterCloudDriver godoc
// @ID unregister-driver
// @Summary Unregister Cloud Driver
// @Description Unregister a specific Cloud Driver.
// @Tags [Cloud Info Management] Driver Info
// @Produce  json
// @Param DriverName path string true "The name of the Cloud Driver"
// @Success 200 {object} BooleanInfo "Result of the unregister operation"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /driver/{DriverName} [delete]
func UnRegisterCloudDriver(c echo.Context) error {
	cblog.Info("call UnRegisterCloudDriver()")

	result, err := dim.UnRegisterCloudDriver(c.Param("DriverName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// uploadCloudDriver godoc
// @ID upload-driver
// @Summary Upload Cloud Driver
// @Description Upload a Cloud Driver library file.
// @Tags [Cloud Info Management] Driver Info
// @Accept mpfd
// @Produce html
// @Param file formData file true "Cloud Driver Library File"
// @Success 200 {string} string "File uploaded successfully"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /driver/upload [post]
func UploadCloudDriver(c echo.Context) error {
	// Source
	file, err := c.FormFile("file")
	if err != nil {
		return err
	}
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	// Destination
	cbspiderRoot := os.Getenv("CBSPIDER_ROOT")
	dst, err := os.Create(cbspiderRoot + "/cloud-driver-libs/" + file.Filename)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Copy
	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	return c.HTML(http.StatusOK, fmt.Sprintf("<p>File %s uploaded successfully.</p>", file.Filename))
}

// ================ Credential Handler

// registerCredential godoc
// @ID register-credential
// @Summary Register Credential
// @Description Register a new Credential. 🕷️ [[User Guide](https://github.com/cloud-barista/cb-spider/wiki/features-and-usages#2-cloud-credential-%EC%A0%95%EB%B3%B4-%EB%93%B1%EB%A1%9D-%EB%B0%8F-%EA%B4%80%EB%A6%AC)]
// @Tags [Cloud Info Management] Credential Info
// @Accept  json
// @Produce  json
// @Param CredentialInfo body cim.CredentialInfo true "Request body for registering a Credential"
// @Success 200 {object} cim.CredentialInfo "Details of the registered Credential"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /credential [post]
func RegisterCredential(c echo.Context) error {
	cblog.Info("call RegisterCredential()")

	req := &cim.CredentialInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	crdinfoList, err := cim.RegisterCredentialInfo(*req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfoList)
}

// ListCredentialResponse represents the response body for listing Credentials.
type ListCredentialResponse struct {
	Result []*cim.CredentialInfo `json:"credential" validate:"required"`
}

// listCredential godoc
// @ID list-credential
// @Summary List Credentials
// @Description Retrieve a list of registered Credentials.
// @Tags [Cloud Info Management] Credential Info
// @Produce  json
// @Param provider query string false "The name of the provider to filter the Credentials by"
// @Success 200 {object} ListCredentialResponse "List of Credentials"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /credential [get]
func ListCredential(c echo.Context) error {
	cblog.Info("call ListCredential()")

	var providerName string
	providerName = c.QueryParam("provider")
	if providerName == "" {
		providerName = c.QueryParam("ProviderName")
	}

	infoList := []*cim.CredentialInfo{}
	var err error
	if providerName != "" {
		infoList, err = cim.ListCredentialByProvider(providerName)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	} else {
		infoList, err = cim.ListCredential()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	var jsonResult ListCredentialResponse
	if infoList == nil {
		infoList = []*cim.CredentialInfo{}
	}
	jsonResult.Result = infoList
	return c.JSON(http.StatusOK, &jsonResult)
}

// getCredential godoc
// @ID get-credential
// @Summary Get Credential
// @Description Retrieve details of a specific Credential.
// @Tags [Cloud Info Management] Credential Info
// @Produce  json
// @Param CredentialName path string true "The name of the Credential"
// @Success 200 {object} cim.CredentialInfo "Details of the Credential"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /credential/{CredentialName} [get]
func GetCredential(c echo.Context) error {
	cblog.Info("call GetCredential()")

	crdinfo, err := cim.GetCredential(c.Param("CredentialName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfo)
}

// unregisterCredential godoc
// @ID unregister-credential
// @Summary Unregister Credential
// @Description Unregister a specific Credential.
// @Tags [Cloud Info Management] Credential Info
// @Produce  json
// @Param CredentialName path string true "The name of the Credential"
// @Success 200 {object} BooleanInfo "Result of the unregister operation"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /credential/{CredentialName} [delete]
func UnRegisterCredential(c echo.Context) error {
	cblog.Info("call UnRegisterCredential()")

	result, err := cim.UnRegisterCredential(c.Param("CredentialName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// ================ Region Handler

// registerRegion godoc
// @ID register-region
// @Summary Register Region
// @Description Register a new Region. 🕷️ [[User Guide](https://github.com/cloud-barista/cb-spider/wiki/features-and-usages#3-cloud-regionzone-%EC%A0%95%EB%B3%B4-%EB%93%B1%EB%A1%9D-%EB%B0%8F-%EA%B4%80%EB%A6%AC)]
// @Tags [Cloud Info Management] Region Info
// @Accept  json
// @Produce  json
// @Param RegionInfo body rim.RegionInfo true "Request body for registering a Region"
// @Success 200 {object} rim.RegionInfo "Details of the registered Region"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /region [post]
func RegisterRegion(c echo.Context) error {
	cblog.Info("call RegisterRegion()")

	req := &rim.RegionInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	crdinfoList, err := rim.RegisterRegionInfo(*req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfoList)
}

// ListRegionResponse represents the response body for listing Regions.
type ListRegionResponse struct {
	Result []*rim.RegionInfo `json:"region" validate:"required"`
}

// listRegion godoc
// @ID list-region
// @Summary List Regions
// @Description Retrieve a list of registered Regions.
// @Tags [Cloud Info Management] Region Info
// @Produce  json
// @Param provider query string false "The name of the provider to filter the Regions by"
// @Success 200 {object} ListRegionResponse "List of Regions"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /region [get]
func ListRegion(c echo.Context) error {
	// cblog.Info("call ListRegion()")

	var providerName string
	providerName = c.QueryParam("provider")
	if providerName == "" {
		providerName = c.QueryParam("ProviderName")
	}

	infoList := []*rim.RegionInfo{}
	var err error
	if providerName != "" {
		infoList, err = rim.ListRegionByProvider(providerName)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	} else {
		infoList, err = rim.ListRegion()
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
	}

	var jsonResult ListRegionResponse
	if infoList == nil {
		infoList = []*rim.RegionInfo{}
	}
	jsonResult.Result = infoList
	return c.JSON(http.StatusOK, &jsonResult)
}

// getRegion godoc
// @ID get-region
// @Summary Get Region
// @Description Retrieve details of a specific Region.
// @Tags [Cloud Info Management] Region Info
// @Produce  json
// @Param RegionName path string true "The name of the Region"
// @Success 200 {object} rim.RegionInfo "Details of the Region"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /region/{RegionName} [get]
func GetRegion(c echo.Context) error {
	cblog.Info("call GetRegion()")

	crdinfo, err := rim.GetRegion(c.Param("RegionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfo)
}

// unregisterRegion godoc
// @ID unregister-region
// @Summary Unregister Region
// @Description Unregister a specific Region.
// @Tags [Cloud Info Management] Region Info
// @Produce  json
// @Param RegionName path string true "The name of the Region"
// @Success 200 {object} BooleanInfo "Result of the unregister operation"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /region/{RegionName} [delete]
func UnRegisterRegion(c echo.Context) error {
	cblog.Info("call UnRegisterRegion()")

	result, err := rim.UnRegisterRegion(c.Param("RegionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// ================ ConnectionConfig Handler

// createConnectionConfig godoc
// @ID create-connection-config
// @Summary Create Connection Config
// @Description Create a new Connection Config. 🕷️ [[User Guide](https://github.com/cloud-barista/cb-spider/wiki/features-and-usages#4-cloud-connection-configuration-%EC%A0%95%EB%B3%B4-%EB%93%B1%EB%A1%9D-%EB%B0%8F-%EA%B4%80%EB%A6%AC)]
// @Tags [Cloud Info Management] Connection Info
// @Accept  json
// @Produce  json
// @Param ConnectionConfigInfo body ccim.ConnectionConfigInfo true "Request body for creating a Connection Config"
// @Success 200 {object} ccim.ConnectionConfigInfo "Details of the created Connection Config"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /connectionconfig [post]
func CreateConnectionConfig(c echo.Context) error {
	cblog.Info("call CreateConnectionConfig()")

	req := &ccim.ConnectionConfigInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	crdinfoList, err := ccim.CreateConnectionConfigInfo(*req)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfoList)
}

// ListConnectionConfigResponse represents the response body for listing Connection Configurations.
type ListConnectionConfigResponse struct {
	Result []*ccim.ConnectionConfigInfo `json:"connectionconfig" validate:"required"`
}

// listConnectionConfig godoc
// @ID list-connection-config
// @Summary List Connection Configs
// @Description Retrieve a list of registered Connection Configs.
// @Tags [Cloud Info Management] Connection Info
// @Produce  json
// @Success 200 {object} ListConnectionConfigResponse "List of Connection Configs"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /connectionconfig [get]
func ListConnectionConfig(c echo.Context) error {
	// cblog.Info("call ListConnectionConfig()")

	infoList, err := ccim.ListConnectionConfig()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult ListConnectionConfigResponse
	if infoList == nil {
		infoList = []*ccim.ConnectionConfigInfo{}
	}
	jsonResult.Result = infoList
	return c.JSON(http.StatusOK, &jsonResult)
}

// getConnectionConfig godoc
// @ID get-connection-config
// @Summary Get Connection Config
// @Description Retrieve details of a specific Connection Config.
// @Tags [Cloud Info Management] Connection Info
// @Produce  json
// @Param ConfigName path string true "The name of the Connection Config"
// @Success 200 {object} ccim.ConnectionConfigInfo "Details of the Connection Config"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /connectionconfig/{ConfigName} [get]
func GetConnectionConfig(c echo.Context) error {
	cblog.Info("call GetConnectionConfig()")

	crdinfo, err := ccim.GetConnectionConfig(c.Param("ConfigName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfo)
}

// deleteConnectionConfig godoc
// @ID delete-connection-config
// @Summary Delete Connection Config
// @Description Delete a specific Connection Config.
// @Tags [Cloud Info Management] Connection Info
// @Produce  json
// @Param ConfigName path string true "The name of the Connection Config"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /connectionconfig/{ConfigName} [delete]
func DeleteConnectionConfig(c echo.Context) error {
	cblog.Info("call DeleteConnectionConfig()")

	result, err := ccim.DeleteConnectionConfig(c.Param("ConfigName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// countAllConnections godoc
// @ID count-all-connection
// @Summary Count All Connections
// @Description Get the total number of connections.
// @Tags [Cloud Info Management] Connection Info
// @Produce  json
// @Success 200 {object} CountResponse "Total count of connections"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countconnectionconfig [get]
func CountAllConnections(c echo.Context) error {
	count, err := ccim.CountAllConnections()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Prepare JSON result
	var jsonResult CountResponse
	jsonResult.Count = int(count)

	// Return JSON response
	return c.JSON(http.StatusOK, jsonResult)
}

// countConnectionsByProvider godoc
// @ID count-connection-by-provider
// @Summary Count Connections by Provider
// @Description Get the total number of connections for a specific provider.
// @Tags [Cloud Info Management] Connection Info
// @Produce  json
// @Param ProviderName path string true "The name of the provider"
// @Success 200 {object} CountResponse "Total count of connections for the provider"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countconnectionconfig/{ProviderName} [get]
func CountConnectionsByProvider(c echo.Context) error {
	count, err := ccim.CountConnectionsByProvider(c.Param("ProviderName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Prepare JSON result
	var jsonResult CountResponse
	jsonResult.Count = int(count)

	// Return JSON response
	return c.JSON(http.StatusOK, jsonResult)
}
