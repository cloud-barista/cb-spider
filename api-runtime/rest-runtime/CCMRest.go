// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by powerkim@etri.re.kr, 2019.10.

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

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateImageHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	req := &cres.ImageReqInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.CreateImage(*req)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func listImage(c echo.Context) error {
	cblog.Info("call listImage()")

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	return c.JSON(http.StatusOK, &infoList)
}

func getImage(c echo.Context) error {
	cblog.Info("call getImage()")

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

//================ VNetwork Handler
func createVNetwork(c echo.Context) error {
	cblog.Info("call createVNetwork()")

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateVNetworkHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	req := &cres.VNetworkReqInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.CreateVNetwork(*req)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func listVNetwork(c echo.Context) error {
	cblog.Info("call listVNetwork()")

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	return c.JSON(http.StatusOK, &infoList)
}

func getVNetwork(c echo.Context) error {
	cblog.Info("call getVNetwork()")

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	req := &cres.SecurityReqInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.CreateSecurity(*req)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func listSecurity(c echo.Context) error {
	cblog.Info("call listSecurity()")

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	return c.JSON(http.StatusOK, &infoList)
}

func getSecurity(c echo.Context) error {
	cblog.Info("call getSecurity()")

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateKeyPairHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	req := &cres.KeyPairReqInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.CreateKey(*req)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func listKey(c echo.Context) error {
	cblog.Info("call listKey()")

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	return c.JSON(http.StatusOK, &infoList)
}

func getKey(c echo.Context) error {
	cblog.Info("call getKey()")

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateVNicHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	req := &cres.VNicReqInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.CreateVNic(*req)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func listVNic(c echo.Context) error {
	cblog.Info("call listVNic()")

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	return c.JSON(http.StatusOK, &infoList)
}

func getVNic(c echo.Context) error {
	cblog.Info("call getVNic()")

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	req := &cres.PublicIPReqInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.CreatePublicIP(*req)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func listPublicIP(c echo.Context) error {
	cblog.Info("call listPublicIP()")

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	return c.JSON(http.StatusOK, &infoList)
}

func getPublicIP(c echo.Context) error {
	cblog.Info("call getPublicIP()")

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	req := &cres.VMReqInfo{}
	if err := c.Bind(req); err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	info, err := handler.StartVM(*req)
	if err != nil {
		return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func listVM(c echo.Context) error {
	cblog.Info("call listVM()")

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	return c.JSON(http.StatusOK, &infoList)
}

func getVM(c echo.Context) error {
	cblog.Info("call getVM()")

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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


	return c.JSON(http.StatusOK, &infoList)
}

func getVMStatus(c echo.Context) error {
	cblog.Info("call getVMStatus()")

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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

	cldConn, err := ccm.GetCloudConnection(c.QueryParam("connection_name"))
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
