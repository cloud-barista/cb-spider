// Cloud Info Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by powerkim@etri.re.kr, 2019.09.

package main

import (
	"fmt"
	"strconv"

	ccim "github.com/cloud-barista/cb-spider/cloud-info-manager/connection-config-info-manager"
	cim "github.com/cloud-barista/cb-spider/cloud-info-manager/credential-info-manager"
	dim "github.com/cloud-barista/cb-spider/cloud-info-manager/driver-info-manager"
	rim "github.com/cloud-barista/cb-spider/cloud-info-manager/region-info-manager"

	// ccim "../../cloud-info-manager/connection-config-info-manager"
	// cim "../../cloud-info-manager/credential-info-manager"
	// dim "../../cloud-info-manager/driver-info-manager"
	// rim "../../cloud-info-manager/region-info-manager"

	// REST API (echo)
	"net/http"

	"github.com/labstack/echo"
)

//================ CloudDriver Handler
func registerCloudDriver(c echo.Context) error {
	cblog.Info("call registerCloudDriver()")
	fmt.Println("###############호출 했음. 왜 안되는가 봅시다.###############")
	req := &dim.CloudDriverInfo{}
	if err := c.Bind(req); err != nil {
		fmt.Println("Binding error!!!")
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	cldinfoList, err := dim.RegisterCloudDriverInfo(*req)
	if err != nil {
		fmt.Println("Register error!!!")
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &cldinfoList)
}

func listCloudDriver(c echo.Context) error {
	cblog.Info("call listCloudDriver()")

	cldinfoList, err := dim.ListCloudDriver()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &cldinfoList)
}

func getCloudDriver(c echo.Context) error {
	cblog.Info("call getCloudDriver()")

	cldinfo, err := dim.GetCloudDriver(c.Param("DriverName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &cldinfo)
}

func unRegisterCloudDriver(c echo.Context) error {
	cblog.Info("call unRegisterCloudDriver()")

	result, err := dim.UnRegisterCloudDriver(c.Param("DriverName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}

//================ Credential Handler
func registerCredential(c echo.Context) error {
	cblog.Info("call registerCredential()")

	req := &cim.CredentialInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	crdinfoList, err := cim.RegisterCredentialInfo(*req)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfoList)
}

func listCredential(c echo.Context) error {
	cblog.Info("call listCredential()")

	crdinfoList, err := cim.ListCredential()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfoList)
}

func getCredential(c echo.Context) error {
	cblog.Info("call getCredential()")

	crdinfo, err := cim.GetCredential(c.Param("CredentialName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfo)
}

func unRegisterCredential(c echo.Context) error {
	cblog.Info("call unRegisterCredential()")

	result, err := cim.UnRegisterCredential(c.Param("CredentialName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}

//================ Region Handler
func registerRegion(c echo.Context) error {
	cblog.Info("call registerRegion()")

	req := &rim.RegionInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	crdinfoList, err := rim.RegisterRegionInfo(*req)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfoList)
}

func listRegion(c echo.Context) error {
	cblog.Info("call listRegion()")

	crdinfoList, err := rim.ListRegion()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfoList)
}

func getRegion(c echo.Context) error {
	cblog.Info("call getRegion()")

	crdinfo, err := rim.GetRegion(c.Param("RegionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfo)
}

func unRegisterRegion(c echo.Context) error {
	cblog.Info("call unRegisterRegion()")

	result, err := rim.UnRegisterRegion(c.Param("RegionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}

//================ ConnectionConfig Handler
func createConnectionConfig(c echo.Context) error {
	cblog.Info("call registerConnectionConfig()")

	req := &ccim.ConnectionConfigInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	crdinfoList, err := ccim.CreateConnectionConfigInfo(*req)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfoList)
}

func listConnectionConfig(c echo.Context) error {
	cblog.Info("call listConnectionConfig()")

	crdinfoList, err := ccim.ListConnectionConfig()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfoList)
}

func getConnectionConfig(c echo.Context) error {
	cblog.Info("call getConnectionConfig()")

	crdinfo, err := ccim.GetConnectionConfig(c.Param("ConfigName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &crdinfo)
}

func deleteConnectionConfig(c echo.Context) error {
	cblog.Info("call deleteConnectionConfig()")

	result, err := ccim.DeleteConnectionConfig(c.Param("ConfigName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}
