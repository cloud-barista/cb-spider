// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.04.
// by CB-Spider Team, 2019.10.

package main

import (
	"fmt"
	"sync"

	ccm "github.com/cloud-barista/cb-spider/cloud-control-manager"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	iidm "github.com/cloud-barista/cb-spider/cloud-control-manager/iid-manager"

	// REST API (echo)
	"github.com/labstack/echo"
	"net/http"

	"strings"
	"strconv"

	"time"
)


// define string of resource types
const (
        rsImage string = "image"
        rsVPC string = "vpc"
        rsSubnet string = "subnet"
        rsSG string = "sg"
        rsKey string = "keypair"
        rsVM string = "vm"
)

// definition of RWLock for each Resource Ops
var imgRWLock = new(sync.RWMutex)
var vpcRWLock = new(sync.RWMutex)
var sgRWLock = new(sync.RWMutex)
var keyRWLock = new(sync.RWMutex)
var vmRWLock = new(sync.RWMutex)


// definition of IIDManager RWLock
var iidRWLock = new(iidm.IIDRWLOCK)

//================ Image Handler
// @todo
// (1) check exist(NameID)
// (2) create Resource
// (3) insert IID
func createImage(c echo.Context) error {
	cblog.Info("call createImage()")

        var req struct {
                ConnectionName string
                ReqInfo cres.ImageReqInfo
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateImageHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	rsType := rsImage
imgRWLock.Lock()
defer imgRWLock.Unlock()
// (1) check exist(NameID)
        bool_ret, err := iidRWLock.IsExistIID(req.ConnectionName, rsType, req.ReqInfo.IId)
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        if bool_ret == true {
                return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf(req.ReqInfo.IId.NameId + " already exists!"))
        }

// (2) create Resource
	info, err := handler.CreateImage(req.ReqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

// (3) insert IID
        iidInfo, err := iidRWLock.CreateIID(req.ConnectionName, rsType, info.IId)
        if err != nil {
                cblog.Error(err)
                // rollback
                _, err2 := handler.DeleteImage(iidInfo.IId)
                if err2 != nil {
                        cblog.Error(err2)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error() + ", " + err2.Error())
                }
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

	return c.JSON(http.StatusOK, &info)
}

// (1) get IID:list
// (2) get CSP:list
// (3) filtering CSP-list by IID-list
func listImage(c echo.Context) error {
	cblog.Info("call listImage()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateImageHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        rsType := rsImage
imgRWLock.RLock()
defer imgRWLock.RUnlock()
// (1) get IID:list
        iidInfoList, err := iidRWLock.ListIID(req.ConnectionName, rsType)
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        var jsonResult struct {
                Result []*cres.ImageInfo `json:"image"`
        }
	var infoList []*cres.ImageInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
                infoList = []*cres.ImageInfo{}
                jsonResult.Result = infoList
                return c.JSON(http.StatusOK, &jsonResult)
        }

// (2) get CSP:list
	infoList, err = handler.ListImage()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
        if infoList == nil { // if iidInfoList not null, then infoList has any list.
                return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("<IID-CSP mismatch> " + rsType + " IID List has " + strconv.Itoa(len(iidInfoList)) + ", but " + req.ConnectionName + " Resource list has nothing!"))
        }

// (3) filtering CSP-list by IID-list
        infoList2 := []*cres.ImageInfo{}
        for _, iidInfo := range iidInfoList {
                exist := false
                for _, info := range infoList {
                        if iidInfo.IId.SystemId == info.IId.SystemId {
                                infoList2 = append(infoList2, info)
                                exist = true
                        }
                }
                if exist == false {
                        return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + req.ConnectionName + " does not have!"))
                }
        }

        jsonResult.Result = infoList2
        return c.JSON(http.StatusOK, &jsonResult)
}

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func getImage(c echo.Context) error {
	cblog.Info("call getImage()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateImageHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        rsType := rsImage
imgRWLock.RLock()
defer imgRWLock.RUnlock()
// (1) get IID(NameId)
        iidInfo, err := iidRWLock.GetIID(req.ConnectionName, rsType, cres.IID{c.Param("Name"), ""})
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

// (2) get resource(SystemId)
start := time.Now()
	info, err := handler.GetImage(iidInfo.IId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
elapsed := time.Since(start)
cblog.Infof(req.ConnectionName + " : elapsed %d", elapsed.Nanoseconds()/1000000) // msec

// (3) set ResourceInfo(IID.NameId)
        info.IId.NameId = iidInfo.IId.NameId

	return c.JSON(http.StatusOK, &info)
}

// (1) get IID(NameId)
// (2) delete Resource(SystemId)
// (3) delete IID
func deleteImage(c echo.Context) error {
	cblog.Info("call deleteImage()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateImageHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        rsType := rsImage
imgRWLock.Lock()
defer imgRWLock.Unlock()
// (1) get IID(NameId)
        iidInfo, err := iidRWLock.GetIID(req.ConnectionName, rsType, cres.IID{c.Param("Name"), ""})
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // keeping for rollback
        info, err := handler.GetImage(iidInfo.IId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

// (2) delete Resource(SystemId)
	result, err := handler.DeleteImage(iidInfo.IId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if result == false {
	        resultInfo := BooleanInfo{
			Result: strconv.FormatBool(result),
		}
		return c.JSON(http.StatusOK, &resultInfo)
	}

// (3) delete IID
        _, err = iidRWLock.DeleteIID(req.ConnectionName, rsType, iidInfo.IId)
        if err != nil {
                cblog.Error(err)
                // rollback
                reqInfo := cres.ImageReqInfo{info.IId} // @todo
                _, err2 := handler.CreateImage(reqInfo)
                if err2 != nil {
                        cblog.Error(err2)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error() + ", " + err2.Error())
                }
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
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
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

	regionName, _, err := ccm.GetRegionNameByConnectionName(req.ConnectionName)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        handler, err := cldConn.CreateVMSpecHandler()
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        infoList, err := handler.ListVMSpec(regionName)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        var jsonResult struct {
                Result []*cres.VMSpecInfo `json:"vmspec"`
        }
	if infoList == nil || len(infoList) <= 0 {
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
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

	regionName, _, err := ccm.GetRegionNameByConnectionName(req.ConnectionName)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        handler, err := cldConn.CreateVMSpecHandler()
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }
        info, err := handler.GetVMSpec(regionName, c.Param("Name"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }
        return c.JSON(http.StatusOK, &info)
}

func listOrgVMSpec(c echo.Context) error {
        cblog.Info("call listOrgVMSpec()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

	regionName, _, err := ccm.GetRegionNameByConnectionName(req.ConnectionName)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        handler, err := cldConn.CreateVMSpecHandler()
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        infoList, err := handler.ListOrgVMSpec(regionName)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.String(http.StatusOK, infoList)
}

func getOrgVMSpec(c echo.Context) error {
        cblog.Info("call getOrgVMSpec()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

	regionName, _, err := ccm.GetRegionNameByConnectionName(req.ConnectionName)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        handler, err := cldConn.CreateVMSpecHandler()
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }
        info, err := handler.GetOrgVMSpec(regionName, c.Param("Name"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }
        return c.String(http.StatusOK, info)
}

//================ VPC Handler
// (1) check exist(NameID)
// (2) create Resource
// (3) insert IID
func createVPC(c echo.Context) error {
	cblog.Info("call createVPC()")

        var req struct {
                ConnectionName string
                ReqInfo cres.VPCReqInfo
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateVPCHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	rsType := rsVPC
vpcRWLock.Lock()
defer vpcRWLock.Unlock()
// (1) check exist(NameID)
	bool_ret, err := iidRWLock.IsExistIID(req.ConnectionName, rsType, req.ReqInfo.IId)
        if err != nil {
                cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

	if bool_ret == true {
                return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf(rsType + "-" + req.ReqInfo.IId.NameId + " already exists!"))
	}
fmt.Printf("req.ReqInfo=============== %#v\n", req.ReqInfo)
// (2) create Resource
	info, err := handler.CreateVPC(req.ReqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
fmt.Printf("info =============== %#v\n", info)

// (3) insert IID
        iidInfo, err := iidRWLock.CreateIID(req.ConnectionName, rsType, info.IId)
        if err != nil {
                cblog.Error(err)
		// rollback
		_, err2 := handler.DeleteVPC(iidInfo.IId)
		if err2 != nil {
			cblog.Error(err2)
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error() + ", " + err2.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

	return c.JSON(http.StatusOK, &info)
}

// (1) get IID:list
// (2) get CSP:list
// (3) filtering CSP-list by IID-list
func listVPC(c echo.Context) error {
	cblog.Info("call listVPC()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateVPCHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	rsType := rsVPC
vpcRWLock.RLock()
defer vpcRWLock.RUnlock()
// (1) get IID:list
        iidInfoList, err := iidRWLock.ListIID(req.ConnectionName, rsType)
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        var jsonResult struct {
                Result []*cres.VPCInfo `json:"vpc"`
        }
	var infoList []*cres.VPCInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
		infoList = []*cres.VPCInfo{}
		jsonResult.Result = infoList
		return c.JSON(http.StatusOK, &jsonResult)
	}

// (2) get CSP:list
	infoList, err = handler.ListVPC()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	if infoList == nil { // if iidInfoList not null, then infoList has any list.
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("<IID-CSP mismatch> " + rsType + " IID List has " + strconv.Itoa(len(iidInfoList)) + ", but " + req.ConnectionName + " Resource list has nothing!"))
	}

// (3) filtering CSP-list by IID-list
	infoList2 := []*cres.VPCInfo{}
        for _, iidInfo := range iidInfoList {
		exist := false
		for _, info := range infoList {
			if iidInfo.IId.SystemId == info.IId.SystemId {
				infoList2 = append(infoList2, info)
				exist = true
			}
		}
		if exist == false {
			return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + req.ConnectionName + " does not have!"))
		}
        }

        jsonResult.Result = infoList2
        return c.JSON(http.StatusOK, &jsonResult)
}

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func getVPC(c echo.Context) error {
	cblog.Info("call getVPC()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateVPCHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	rsType := rsVPC
vpcRWLock.RLock()
defer vpcRWLock.RUnlock()
// (1) get IID(NameId)
        iidInfo, err := iidRWLock.GetIID(req.ConnectionName, rsType, cres.IID{c.Param("Name"), ""})
        if err != nil {
                cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

// (2) get resource(SystemId)
	info, err := handler.GetVPC(iidInfo.IId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
// (3) set ResourceInfo(IID.NameId)
	info.IId.NameId = iidInfo.IId.NameId

	return c.JSON(http.StatusOK, &info)
}

// (1) get IID(NameId)
// (2) delete Resource(SystemId)
// (3) delete IID
func deleteVPC(c echo.Context) error {
	cblog.Info("call deleteVPC()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateVPCHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	rsType := rsVPC
vpcRWLock.Lock()
defer vpcRWLock.Unlock()
// (1) get IID(NameId)
        iidInfo, err := iidRWLock.GetIID(req.ConnectionName, rsType, cres.IID{c.Param("Name"), ""})
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

	// keeping for rollback
        info, err := handler.GetVPC(iidInfo.IId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }


// (2) delete Resource(SystemId)
	result, err := handler.DeleteVPC(iidInfo.IId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
        if result == false {
                resultInfo := BooleanInfo{
                        Result: strconv.FormatBool(result),
                }
                return c.JSON(http.StatusOK, &resultInfo)
        }

// (3) delete IID
        _, err = iidRWLock.DeleteIID(req.ConnectionName, rsType, iidInfo.IId)
        if err != nil {
                cblog.Error(err)
                // rollback
		reqInfo := cres.VPCReqInfo{info.IId, info.IPv4_CIDR, info.SubnetInfoList } 	
                _, err2 := handler.CreateVPC(reqInfo)
                if err2 != nil {
                        cblog.Error(err2)
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error() + ", " + err2.Error())
                }
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}

//================ SecurityGroup Handler
// (1) check exist(NameID)
// (2) create Resource
// (3) insert IID
func createSecurity(c echo.Context) error {
	cblog.Info("call createSecurity()")

	var req struct {
		ConnectionName string
		ReqInfo cres.SecurityReqInfo
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        rsType := rsSG
sgRWLock.Lock()
defer sgRWLock.Unlock()
// (1) check exist(NameID)
        bool_ret, err := iidRWLock.IsExistIID(req.ConnectionName, rsType, req.ReqInfo.IId)
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        if bool_ret == true {
                return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf(rsType + "-" + req.ReqInfo.IId.NameId + " already exists!"))
        }

// (2) create Resource
	info, err := handler.CreateSecurity(req.ReqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

// (3) insert IID
        iidInfo, err := iidRWLock.CreateIID(req.ConnectionName, rsType, info.IId)
        if err != nil {
                cblog.Error(err)
                // rollback
                _, err2 := handler.DeleteSecurity(iidInfo.IId)
                if err2 != nil {
                        cblog.Error(err2)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error() + ", " + err2.Error())
                }
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

	return c.JSON(http.StatusOK, &info)
}

// (1) get IID:list
// (2) get CSP:list
// (3) filtering CSP-list by IID-list
func listSecurity(c echo.Context) error {
	cblog.Info("call listSecurity()")

	var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        rsType := rsSG
sgRWLock.RLock()
defer sgRWLock.RUnlock()
// (1) get IID:list
        iidInfoList, err := iidRWLock.ListIID(req.ConnectionName, rsType)
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        var jsonResult struct {
                Result []*cres.SecurityInfo `json:"securitygroup"`
        }
	var infoList []*cres.SecurityInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
                infoList = []*cres.SecurityInfo{}
                jsonResult.Result = infoList
                return c.JSON(http.StatusOK, &jsonResult)
        }

// (2) get CSP:list
	infoList, err = handler.ListSecurity()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
        if infoList == nil { // if iidInfoList not null, then infoList has any list.
                return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("<IID-CSP mismatch> " + rsType + " IID List has " + strconv.Itoa(len(iidInfoList)) + ", but " + req.ConnectionName + " Resource list has nothing!"))
        }

// (3) filtering CSP-list by IID-list
        infoList2 := []*cres.SecurityInfo{}
        for _, iidInfo := range iidInfoList {
                exist := false
                for _, info := range infoList {
                        if iidInfo.IId.SystemId == info.IId.SystemId {
				infoList2 = append(infoList2, info)
                                exist = true
                        }
                }
                if exist == false {
                        return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + req.ConnectionName + " does not have!"))
                }
        }

        jsonResult.Result = infoList2
        return c.JSON(http.StatusOK, &jsonResult)
}

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func getSecurity(c echo.Context) error {
	cblog.Info("call getSecurity()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        rsType := rsSG
sgRWLock.RLock()
defer sgRWLock.RUnlock()
// (1) get IID(NameId)
        iidInfo, err := iidRWLock.GetIID(req.ConnectionName, rsType, cres.IID{c.Param("Name"), ""})
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

// (2) get resource(SystemId)
	info, err := handler.GetSecurity(iidInfo.IId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

// (3) set ResourceInfo(IID.NameId)
        info.IId.NameId = iidInfo.IId.NameId

	return c.JSON(http.StatusOK, &info)
}

// (1) get IID(NameId)
// (2) delete Resource(SystemId)
// (3) delete IID
func deleteSecurity(c echo.Context) error {
	cblog.Info("call deleteSecurity()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateSecurityHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        rsType := rsSG
sgRWLock.Lock()
defer sgRWLock.Unlock()
// (1) get IID(NameId)
        iidInfo, err := iidRWLock.GetIID(req.ConnectionName, rsType, cres.IID{c.Param("Name"), ""})
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // keeping for rollback
        info, err := handler.GetSecurity(iidInfo.IId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

// (2) delete Resource(SystemId)
	result, err := handler.DeleteSecurity(iidInfo.IId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
        if result == false {
                resultInfo := BooleanInfo{
                        Result: strconv.FormatBool(result),
                }
                return c.JSON(http.StatusOK, &resultInfo)
        }

// (3) delete IID
        _, err = iidRWLock.DeleteIID(req.ConnectionName, rsType, iidInfo.IId)
        if err != nil {
                cblog.Error(err)
                // rollback
                reqInfo := cres.SecurityReqInfo{info.IId, info.VpcIID, info.Direction, info.SecurityRules}
                _, err2 := handler.CreateSecurity(reqInfo)
                if err2 != nil {
                        cblog.Error(err2)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error() + ", " + err2.Error())
                }
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }
        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}

//================ KeyPair Handler
// (1) check exist(NameID)
// (2) create Resource
// (3) insert IID
func createKey(c echo.Context) error {
	cblog.Info("call createKey()")

        var req struct {
                ConnectionName string
                ReqInfo cres.KeyPairReqInfo
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateKeyPairHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        rsType := rsKey
keyRWLock.Lock()
defer keyRWLock.Unlock()
// (1) check exist(NameID)
        bool_ret, err := iidRWLock.IsExistIID(req.ConnectionName, rsType, req.ReqInfo.IId)
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        if bool_ret == true {
                return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf(req.ReqInfo.IId.NameId + " already exists!"))
        }

// (2) create Resource
	info, err := handler.CreateKey(req.ReqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

// (3) insert IID
        iidInfo, err := iidRWLock.CreateIID(req.ConnectionName, rsType, info.IId)
        if err != nil {
                cblog.Error(err)
                // rollback
                _, err2 := handler.DeleteKey(iidInfo.IId)
                if err2 != nil {
                        cblog.Error(err2)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error() + ", " + err2.Error())
                }
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

	return c.JSON(http.StatusOK, &info)
}

// (1) get IID:list
// (2) get CSP:list
// (3) filtering CSP-list by IID-list
func listKey(c echo.Context) error {
	cblog.Info("call listKey()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateKeyPairHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        rsType := rsKey
keyRWLock.RLock()
defer keyRWLock.RUnlock()
// (1) get IID:list
        iidInfoList, err := iidRWLock.ListIID(req.ConnectionName, rsType)
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        var jsonResult struct {
                Result []*cres.KeyPairInfo `json:"keypair"`
        }
        var infoList []*cres.KeyPairInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
                infoList = []*cres.KeyPairInfo{}
                jsonResult.Result = infoList
                return c.JSON(http.StatusOK, &jsonResult)
        }

// (2) get CSP:list
	infoList, err = handler.ListKey()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
        if infoList == nil { // if iidInfoList not null, then infoList has any list.
                return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("<IID-CSP mismatch> " + rsType + " IID List has " + strconv.Itoa(len(iidInfoList)) + ", but " + req.ConnectionName + " Resource list has nothing!"))
        }

// (3) filtering CSP-list by IID-list
        infoList2 := []*cres.KeyPairInfo{}
        for _, iidInfo := range iidInfoList {
                exist := false
                for _, info := range infoList {
                        if iidInfo.IId.SystemId == info.IId.SystemId {
                                infoList2 = append(infoList2, info)
                                exist = true
                        }
                }
                if exist == false {
                        return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + req.ConnectionName + " does not have!"))
                }
        }

        jsonResult.Result = infoList
        return c.JSON(http.StatusOK, &jsonResult)
}

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func getKey(c echo.Context) error {
	cblog.Info("call getKey()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateKeyPairHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        rsType := rsKey
keyRWLock.RLock()
defer keyRWLock.RUnlock()
// (1) get IID(NameId)
        iidInfo, err := iidRWLock.GetIID(req.ConnectionName, rsType, cres.IID{c.Param("Name"), ""})
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

// (2) get resource(SystemId)
	info, err := handler.GetKey(iidInfo.IId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

// (3) set ResourceInfo(IID.NameId)
        info.IId.NameId = iidInfo.IId.NameId

	return c.JSON(http.StatusOK, &info)
}

// (1) get IID(NameId)
// (2) delete Resource(SystemId)
// (3) delete IID
func deleteKey(c echo.Context) error {
	cblog.Info("call deleteKey()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateKeyPairHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        rsType := rsKey
keyRWLock.Lock()
defer keyRWLock.Unlock()
// (1) get IID(NameId)
        iidInfo, err := iidRWLock.GetIID(req.ConnectionName, rsType, cres.IID{c.Param("Name"), ""})
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // keeping for rollback
        info, err := handler.GetKey(iidInfo.IId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

// (2) delete Resource(SystemId)
	result, err := handler.DeleteKey(iidInfo.IId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
        if result == false {
                resultInfo := BooleanInfo{
                        Result: strconv.FormatBool(result),
                }
                return c.JSON(http.StatusOK, &resultInfo)
        }

// (3) delete IID
        _, err = iidRWLock.DeleteIID(req.ConnectionName, rsType, iidInfo.IId)
        if err != nil {
                cblog.Error(err)
                // rollback
                reqInfo := cres.KeyPairReqInfo{info.IId}
                _, err2 := handler.CreateKey(reqInfo) // @todo check local key files
                if err2 != nil {
                        cblog.Error(err2)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error() + ", " + err2.Error())
                }
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}

/****************************
//================ VNic Handler
func createVNic(c echo.Context) error {
	cblog.Info("call createVNic()")

        var req struct {
                ConnectionName string
                ReqInfo cres.VNicReqInfo
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateVNicHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	info, err := handler.CreateVNic(req.ReqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func listVNic(c echo.Context) error {
	cblog.Info("call listVNic()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateVNicHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	infoList, err := handler.ListVNic()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
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
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateVNicHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	info, err := handler.GetVNic(c.Param("VNicId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func deleteVNic(c echo.Context) error {
	cblog.Info("call deleteVNic()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateVNicHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := handler.DeleteVNic(c.Param("VNicId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
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
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	info, err := handler.CreatePublicIP(req.ReqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func listPublicIP(c echo.Context) error {
	cblog.Info("call listPublicIP()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	infoList, err := handler.ListPublicIP()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
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
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	info, err := handler.GetPublicIP(c.Param("PublicIPId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &info)
}

func deletePublicIP(c echo.Context) error {
	cblog.Info("call deletePublicIP()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreatePublicIPHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	result, err := handler.DeletePublicIP(c.Param("PublicIPId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}
****************************/

//================ VM Handler
// (1) check exist(NameID)
// (2) create Resource
// (3) insert IID
func startVM(c echo.Context) error {
	cblog.Info("call startVM()")

        var req struct {
                ConnectionName string
                ReqInfo cres.VMReqInfo
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        rsType := rsVM
vmRWLock.Lock()
defer vmRWLock.Unlock()
// (1) check exist(NameID)
        bool_ret, err := iidRWLock.IsExistIID(req.ConnectionName, rsType, req.ReqInfo.IId)
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        if bool_ret == true {
                return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf(rsType + "-" + req.ReqInfo.IId.NameId + " already exists!"))
        }

// (2) create Resource
	info, err := handler.StartVM(req.ReqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

// (3) insert IID
        iidInfo, err := iidRWLock.CreateIID(req.ConnectionName, rsType, info.IId)
        if err != nil {
                cblog.Error(err)
                // rollback
                _, err2 := handler.TerminateVM(iidInfo.IId) // @todo check validation
                if err2 != nil {
                        cblog.Error(err2)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error() + ", " + err2.Error())
                }
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

	return c.JSON(http.StatusOK, &info)
}

// (1) get IID:list
// (2) get CSP:list
// (3) filtering CSP-list by IID-list
func listVM(c echo.Context) error {
	cblog.Info("call listVM()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        rsType := rsVM
vmRWLock.RLock()
defer vmRWLock.RUnlock()
// (1) get IID:list
        iidInfoList, err := iidRWLock.ListIID(req.ConnectionName, rsType)
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        var jsonResult struct {
                Result []*cres.VMInfo `json:"vm"`
        }
        var infoList []*cres.VMInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
                infoList = []*cres.VMInfo{}
                jsonResult.Result = infoList
                return c.JSON(http.StatusOK, &jsonResult)
        }

// (2) get CSP:list
	infoList, err = handler.ListVM()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
        if infoList == nil { // if iidInfoList not null, then infoList has any list.
                return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("<IID-CSP mismatch> " + rsType + " IID List has " + strconv.Itoa(len(iidInfoList)) + ", but " + req.ConnectionName + " Resource list has nothing!"))
        }

// (3) filtering CSP-list by IID-list
        infoList2 := []*cres.VMInfo{}
        for _, iidInfo := range iidInfoList {
                exist := false
                for _, info := range infoList {
                        if iidInfo.IId.SystemId == info.IId.SystemId {
                                infoList2 = append(infoList2, info)
                                exist = true
                        }
                }
                if exist == false {
                        return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + req.ConnectionName + " does not have!"))
                }
        }

        jsonResult.Result = infoList2
        return c.JSON(http.StatusOK, &jsonResult)
}

// (1) get IID(NameId)
// (2) get resource(SystemId)
// (3) set ResourceInfo(IID.NameId)
func getVM(c echo.Context) error {
	cblog.Info("call getVM()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        rsType := rsVM
vmRWLock.RLock()
defer vmRWLock.RUnlock()
// (1) get IID(NameId)
        iidInfo, err := iidRWLock.GetIID(req.ConnectionName, rsType, cres.IID{c.Param("Name"), ""})
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

// (2) get resource(SystemId)
	info, err := handler.GetVM(iidInfo.IId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

// (3) set ResourceInfo(IID.NameId)
        info.IId.NameId = iidInfo.IId.NameId

	return c.JSON(http.StatusOK, &info)
}

// (1) get IID(NameId)
// (2) delete Resource(SystemId)
// (3) delete IID
func terminateVM(c echo.Context) error {
	cblog.Info("call terminateVM()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        rsType := rsVM
vmRWLock.Lock()
defer vmRWLock.Unlock()
// (1) get IID(NameId)
        iidInfo, err := iidRWLock.GetIID(req.ConnectionName, rsType, cres.IID{c.Param("Name"), ""})
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // keeping for rollback
        info, err := handler.GetVM(iidInfo.IId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

// (2) delete Resource(SystemId)
	info2, err := handler.TerminateVM(iidInfo.IId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

// (3) delete IID
        _, err = iidRWLock.DeleteIID(req.ConnectionName, rsType, iidInfo.IId)
        if err != nil {
                cblog.Error(err)
                // rollback
                reqInfo := cres.VMReqInfo{info.IId, info.ImageIId, info.VpcIID, info.SubnetIID, info.SecurityGroupIIds, info.VMSpecName, info.KeyPairIId, info.VMUserId, info.VMUserPasswd}
                _, err2 := handler.StartVM(reqInfo)
                if err2 != nil {
                        cblog.Error(err2)
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error() + ", " + err2.Error())
                }
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := StatusInfo{
                Status: string(info2),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}

// (1) get IID:list
// (2) get CSP:VMStatuslist
// (3) filtering CSP-VMStatuslist by IID-list
func listVMStatus(c echo.Context) error {
	cblog.Info("call listVMStatus()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        rsType := rsVM
vmRWLock.RLock()
defer vmRWLock.RUnlock()
// (1) get IID:list
        iidInfoList, err := iidRWLock.ListIID(req.ConnectionName, rsType)
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        var jsonResult struct {
                Result []*cres.VMStatusInfo `json:"vmstatus"`
        }
        var infoList []*cres.VMStatusInfo
	if iidInfoList == nil || len(iidInfoList) <= 0 {
                infoList = []*cres.VMStatusInfo{}
                jsonResult.Result = infoList
                return c.JSON(http.StatusOK, &jsonResult)
        }

// (2) get CSP:VMStatuslist
	infoList, err = handler.ListVMStatus()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
        if infoList == nil { // if iidInfoList not null, then infoList has any list.
                return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("<IID-CSP mismatch> " + rsType + " IID List has " + strconv.Itoa(len(iidInfoList)) + ", but " + req.ConnectionName + " Resource list has nothing!"))
        }

// (3) filtering CSP-VMStatuslist by IID-list
        infoList2 := []*cres.VMStatusInfo{}
        for _, iidInfo := range iidInfoList {
                exist := false
                for _, info := range infoList {
                        if iidInfo.IId.SystemId == info.IId.SystemId {
                                infoList2 = append(infoList2, info)
                                exist = true
                        }
                }
                if exist == false {
                        return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf("<IID-CSP mismatch> " + rsType + "-" + iidInfo.IId.NameId + ":" + iidInfo.IId.SystemId + " exsits. but " + req.ConnectionName + " does not have!"))
                }
        }

        jsonResult.Result = infoList2
        return c.JSON(http.StatusOK, &jsonResult)
}

// (1) get IID(NameId)
// (2) get CSP:VMStatus(SystemId)
func getVMStatus(c echo.Context) error {
	cblog.Info("call getVMStatus()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        rsType := rsVM
vmRWLock.RLock()
defer vmRWLock.RUnlock()
// (1) get IID(NameId)
        iidInfo, err := iidRWLock.GetIID(req.ConnectionName, rsType, cres.IID{c.Param("Name"), ""})
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

// (2) get CSP:VMStatus(SystemId)
	info, err := handler.GetVMStatus(iidInfo.IId)  // type of info => string
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        resultInfo := StatusInfo{
                Status: string(info),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}

// (1) get IID(NameId)
// (2) control CSP:VM(SystemId)
func controlVM(c echo.Context) error {
	cblog.Info("call controlVM()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        cldConn, err := ccm.GetCloudConnection(req.ConnectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	handler, err := cldConn.CreateVMHandler()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        rsType := rsVM
vmRWLock.RLock()
defer vmRWLock.RUnlock()
// (1) get IID(NameId)
        iidInfo, err := iidRWLock.GetIID(req.ConnectionName, rsType, cres.IID{c.Param("Name"), ""})
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

// (2) control CSP:VM(SystemId)
	vmIID := iidInfo.IId
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	action := c.QueryParam("action")
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}


	var info cres.VMStatus

	switch strings.ToLower(action) {
	case "suspend":
		info, err = handler.SuspendVM(vmIID)
	case "resume":
		info, err = handler.ResumeVM(vmIID)
	case "reboot":
		info, err = handler.RebootVM(vmIID)
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
