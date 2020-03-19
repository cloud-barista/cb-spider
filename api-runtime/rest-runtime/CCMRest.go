// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2019.10.

package main

import (
	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	// REST API (echo)
	"github.com/labstack/echo"
	"net/http"

	"strings"
	"strconv"

	"time"
)

//================ Image Handler
// @todo
func createImage(c echo.Context) error {
	cblog.Info("call createImage()")

        var req struct {
                ConnectionName string
                ReqInfo cres.ImageReqInfo
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateImageHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.CreateImage(req.ReqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func listImage(c echo.Context) error {
	cblog.Info("call listImage()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateImageHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	infoList, err := handler.ListImage()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

        var jsonResult struct {
                Result []*cres.ImageInfo `json:"image"`
        }
        if infoList == nil {
                infoList = []*cres.ImageInfo{}
        }
        jsonResult.Result = infoList
        return c.JSON(http.StatusOK, &jsonResult)
}

func getImage(c echo.Context) error {
	cblog.Info("call getImage()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateImageHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}
start := time.Now()
	info, err := handler.GetImage(c.Param("ImageName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}
elapsed := time.Since(start)
cblog.Infof(c.QueryParam("connection_name") + " : elapsed %d", elapsed.Nanoseconds()/1000000) // msec
	return c.JSON(http.StatusOK, &info)
}

func deleteImage(c echo.Context) error {
	cblog.Info("call deleteImage()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateImageHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	result, err := handler.DeleteImage(c.Param("ImageId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

//================ VMSpec Handler
func listVMSpec(c echo.Context) error {
        cblog.Info("call listVMSpec()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        handler, err := cldConn.CreateVMSpecHandler()
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        infoList, err := handler.ListVMSpec(c.Param("RegionName"))
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        var jsonResult struct {
                Result []*cres.VMSpecInfo `json:"vmspec"`
        }
        if infoList == nil {
                infoList = []*cres.VMSpecInfo{}
        }
        jsonResult.Result = infoList
        return c.JSON(http.StatusOK, &jsonResult)
}

func getVMSpec(c echo.Context) error {
        cblog.Info("call getVMSpec()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        handler, err := cldConn.CreateVMSpecHandler()
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }
        info, err := handler.GetVMSpec(c.Param("RegionName"), c.Param("VMSpecName"))
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }
        return c.JSON(http.StatusOK, &info)
}

func listOrgVMSpec(c echo.Context) error {
        cblog.Info("call listOrgVMSpec()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        handler, err := cldConn.CreateVMSpecHandler()
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        infoList, err := handler.ListOrgVMSpec(c.Param("RegionName"))
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        return c.JSON(http.StatusOK, &infoList)
}

func getOrgVMSpec(c echo.Context) error {
        cblog.Info("call getOrgVMSpec()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        handler, err := cldConn.CreateVMSpecHandler()
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }
        info, err := handler.GetOrgVMSpec(c.Param("RegionName"), c.Param("VMSpecName"))
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }
        return c.JSON(http.StatusOK, &info)
}

//================ VNetwork Handler
func createVNetwork(c echo.Context) error {
	cblog.Info("call createVNetwork()")

        var req struct {
                ConnectionName string
                ReqInfo cres.VNetworkReqInfo
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateVNetworkHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.CreateVNetwork(req.ReqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func listVNetwork(c echo.Context) error {
	cblog.Info("call listVNetwork()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateVNetworkHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	infoList, err := handler.ListVNetwork()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

        var jsonResult struct {
                Result []*cres.VNetworkInfo `json:"vnetwork"`
        }
	if infoList == nil {
		infoList = []*cres.VNetworkInfo{}
	}
        jsonResult.Result = infoList
        return c.JSON(http.StatusOK, &jsonResult)
}

func getVNetwork(c echo.Context) error {
	cblog.Info("call getVNetwork()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateVNetworkHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.GetVNetwork(c.Param("VNetId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func deleteVNetwork(c echo.Context) error {
	cblog.Info("call deleteVNetwork()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateVNetworkHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	result, err := handler.DeleteVNetwork(c.Param("VNetId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}

//================ SecurityGroup Handler
func createSecurity(c echo.Context) error {
	cblog.Info("call createSecurity()")

	var req struct {
		ConnectionName string
		ReqInfo cres.SecurityReqInfo
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.CreateSecurity(req.ReqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func listSecurity(c echo.Context) error {
	cblog.Info("call listSecurity()")

	var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	infoList, err := handler.ListSecurity()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

        var jsonResult struct {
                Result []*cres.SecurityInfo `json:"securitygroup"`
        }
        if infoList == nil {
                infoList = []*cres.SecurityInfo{}
        }
        jsonResult.Result = infoList
        return c.JSON(http.StatusOK, &jsonResult)
}

func getSecurity(c echo.Context) error {
	cblog.Info("call getSecurity()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.GetSecurity(c.Param("SecurityGroupId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func deleteSecurity(c echo.Context) error {
	cblog.Info("call deleteSecurity()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	result, err := handler.DeleteSecurity(c.Param("SecurityGroupId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}

//================ KeyPair Handler
func createKey(c echo.Context) error {
	cblog.Info("call createKey()")

        var req struct {
                ConnectionName string
                ReqInfo cres.KeyPairReqInfo
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateKeyPairHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.CreateKey(req.ReqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func listKey(c echo.Context) error {
	cblog.Info("call listKey()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateKeyPairHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	infoList, err := handler.ListKey()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

        var jsonResult struct {
                Result []*cres.KeyPairInfo `json:"keypair"`
        }
        if infoList == nil {
                infoList = []*cres.KeyPairInfo{}
        }
        jsonResult.Result = infoList
        return c.JSON(http.StatusOK, &jsonResult)
}

func getKey(c echo.Context) error {
	cblog.Info("call getKey()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateKeyPairHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.GetKey(c.Param("KeyPairId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func deleteKey(c echo.Context) error {
	cblog.Info("call deleteKey()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateKeyPairHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	result, err := handler.DeleteKey(c.Param("KeyPairId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}

//================ VNic Handler
func createVNic(c echo.Context) error {
	cblog.Info("call createVNic()")

        var req struct {
                ConnectionName string
                ReqInfo cres.VNicReqInfo
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateVNicHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.CreateVNic(req.ReqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func listVNic(c echo.Context) error {
	cblog.Info("call listVNic()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateVNicHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	infoList, err := handler.ListVNic()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

        var jsonResult struct {
                Result []*cres.VNicInfo `json:"vnic"`
        }
        if infoList == nil {
                infoList = []*cres.VNicInfo{}
        }
        jsonResult.Result = infoList
        return c.JSON(http.StatusOK, &jsonResult)
}

func getVNic(c echo.Context) error {
	cblog.Info("call getVNic()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateVNicHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.GetVNic(c.Param("VNicId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func deleteVNic(c echo.Context) error {
	cblog.Info("call deleteVNic()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateVNicHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	result, err := handler.DeleteVNic(c.Param("VNicId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}

//================ PublicIP Handler
func createPublicIP(c echo.Context) error {
	cblog.Info("call createPublicIP()")

        var req struct {
                ConnectionName string
                ReqInfo cres.PublicIPReqInfo
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.CreatePublicIP(req.ReqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func listPublicIP(c echo.Context) error {
	cblog.Info("call listPublicIP()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	infoList, err := handler.ListPublicIP()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

        var jsonResult struct {
                Result []*cres.PublicIPInfo `json:"publicip"`
        }
        if infoList == nil {
                infoList = []*cres.PublicIPInfo{}
        }
        jsonResult.Result = infoList
        return c.JSON(http.StatusOK, &jsonResult)
}

func getPublicIP(c echo.Context) error {
	cblog.Info("call getPublicIP()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.GetPublicIP(c.Param("PublicIPId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func deletePublicIP(c echo.Context) error {
	cblog.Info("call deletePublicIP()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	result, err := handler.DeletePublicIP(c.Param("PublicIPId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}

//================ VM Handler
func startVM(c echo.Context) error {
	cblog.Info("call startVM()")

        var req struct {
                ConnectionName string
                ReqInfo cres.VMReqInfo
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.StartVM(req.ReqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func listVM(c echo.Context) error {
	cblog.Info("call listVM()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	infoList, err := handler.ListVM()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

        var jsonResult struct {
                Result []*cres.VMInfo `json:"vm"`
        }
        if infoList == nil {
                infoList = []*cres.VMInfo{}
        }
        jsonResult.Result = infoList
        return c.JSON(http.StatusOK, &jsonResult)
}

func getVM(c echo.Context) error {
	cblog.Info("call getVM()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.GetVM(c.Param("VmId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func terminateVM(c echo.Context) error {
	cblog.Info("call terminateVM()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.TerminateVM(c.Param("VmId"))
        if err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        resultInfo := StatusInfo{
                Status: string(info),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}

func listVMStatus(c echo.Context) error {
	cblog.Info("call listVMStatus()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	infoList, err := handler.ListVMStatus()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

        var jsonResult struct {
                Result []*cres.VMStatusInfo `json:"vmstatus"`
        }
        if infoList == nil {
                infoList = []*cres.VMStatusInfo{}
        }
        jsonResult.Result = infoList
        return c.JSON(http.StatusOK, &jsonResult)
}

func getVMStatus(c echo.Context) error {
	cblog.Info("call getVMStatus()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.GetVMStatus(c.Param("VmId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

        resultInfo := StatusInfo{
                Status: string(info),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}

func controlVM(c echo.Context) error {
	cblog.Info("call controlVM()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	vmID := c.Param("VmId")
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	action := c.QueryParam("action")
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}


	var info cres.VMStatus

	switch strings.ToLower(action) {
	case "suspend":
		info, err = handler.SuspendVM(vmID)
	case "resume":
		info, err = handler.ResumeVM(vmID)
	case "reboot":
		info, err = handler.RebootVM(vmID)
	default:
		errmsg := action + " is not a valid action!!"
		return echo.NewHTTPError(http.StatusInternalServerError, errmsg)

	}
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
        resultInfo := StatusInfo{
                Status: string(info),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}
