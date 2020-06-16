// Cloud Control Manager's Rest Runtime of CB-Spider.ll
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
	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"

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
	// rsSubnet = SUBNET:{VPC NameID} => cook in code
        rsSG string = "sg"
        rsKey string = "keypair"
        rsVM string = "vm"
)

const rsSubnetPrefix string = "subnet:"
const sgDELIMITER string = "-delimiter-"

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
                ReqInfo struct {
			Name string
		}
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
        bool_ret, err := iidRWLock.IsExistIID(req.ConnectionName, rsType, cres.IID{req.ReqInfo.Name, ""})
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        if bool_ret == true {
                return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf(req.ReqInfo.Name + " already exists!"))
        }

        reqInfo := cres.ImageReqInfo {
                        IId: cres.IID{req.ReqInfo.Name, ""},
                   }

// (2) create Resource
	info, err := handler.CreateImage(reqInfo)
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
				info.IId.NameId = iidInfo.IId.NameId
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
                ReqInfo struct {
                        Name 		string
                        IPv4_CIDR	string
			SubnetInfoList [] struct {
				Name  string
				IPv4_CIDR string
			}
                }
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

	// check the input Name to include the SUBNET: Prefix
	if strings.HasPrefix(req.ReqInfo.Name, rsSubnetPrefix) {
		return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf(rsSubnetPrefix + " cannot be used for VPC name prefix!!"))
	}
        // check the input Name to include the SecurityGroup Delimiter
        if strings.HasPrefix(req.ReqInfo.Name, sgDELIMITER) {
                return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf(sgDELIMITER + " cannot be used in VPC name!!"))
        }



        // Rest RegInfo => Driver ReqInfo
	// (1) create SubnetInfo List
	subnetInfoList := []cres.SubnetInfo{}
	for _, info := range req.ReqInfo.SubnetInfoList {
		subnetInfo := cres.SubnetInfo{IId: cres.IID{info.Name, ""}, IPv4_CIDR: info.IPv4_CIDR}
		subnetInfoList = append(subnetInfoList, subnetInfo)
	}
	// (2) create VPCReqInfo with SubnetInfo List
        reqInfo := cres.VPCReqInfo {
                        IId: cres.IID{req.ReqInfo.Name, ""},
                        IPv4_CIDR: req.ReqInfo.IPv4_CIDR,
                       SubnetInfoList: subnetInfoList,
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
	bool_ret, err := iidRWLock.IsExistIID(req.ConnectionName, rsType, reqInfo.IId)
        if err != nil {
                cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

	if bool_ret == true {
                return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!"))
	}
// (2) create Resource
	info, err := handler.CreateVPC(reqInfo)
	if err != nil {
                cblog.Error(err)
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	info.IId.NameId = req.ReqInfo.Name

// (3) insert IID
	// for VPC
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
	// for Subnet list
	for _, subnetInfo := range info.SubnetInfoList {
		// key-value structure: /{ConnectionName}/{VPC-NameId}/{Subnet-IId}
                _, err := iidRWLock.CreateIID(req.ConnectionName, rsSubnetPrefix + info.IId.NameId, subnetInfo.IId)
                if err != nil {
			cblog.Error(err)
			// rollback
			// (1) for resource
			cblog.Info("<<ROLLBACK:TRY:VPC-CSP>> " + info.IId.SystemId)
			_, err2 := handler.DeleteVPC(iidInfo.IId)
			if err2 != nil {
				cblog.Error(err2)
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error() + ", " + err2.Error())
			}
			// (2) for VPC IID
			cblog.Info("<<ROLLBACK:TRY:VPC-IID>> " + info.IId.NameId)
			_, err := iidRWLock.DeleteIID(req.ConnectionName, rsType, info.IId)
			if err != nil {
				return echo.NewHTTPError(http.StatusInternalServerError, err.Error() + ", " + err.Error())
			}
			// (3) for Subnet IID
			tmpIIdInfoList, err := iidRWLock.ListIID(req.ConnectionName, rsSubnetPrefix + info.IId.NameId)
			for _, subnetInfo := range tmpIIdInfoList {
				_, err := iidRWLock.DeleteIID(req.ConnectionName, rsSubnetPrefix + info.IId.NameId, subnetInfo.IId)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
				}
			}
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
                }
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
				
				//+++++++++++++++++++++++++++++++++++++++++++
				// set ResourceInfo(IID.NameId)
					// set VPC NameId
					IIdInfo, err := iidRWLock.GetIIDbySystemID(req.ConnectionName, rsType, info.IId)
					if err != nil {
						return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
					}
					info.IId.NameId = IIdInfo.IId.NameId
				//+++++++++++++++++++++++++++++++++++++++++++
				// set NameId for SubnetInfo List
					// create new SubnetInfo List
					subnetInfoList := []cres.SubnetInfo{}
					for _, subnetInfo := range info.SubnetInfoList {
						subnetIIDInfo, err := iidRWLock.GetIIDbySystemID(req.ConnectionName, rsSubnetPrefix + info.IId.NameId, subnetInfo.IId)
						if err != nil {
							return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
						}	
						if subnetIIDInfo.IId.NameId != "" { // insert only this user created.
							subnetInfo.IId.NameId = subnetIIDInfo.IId.NameId
							subnetInfoList = append(subnetInfoList, subnetInfo)
						} 
						
					}
					info.SubnetInfoList = subnetInfoList


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

// list all VPCs for management
// (1) get args from REST Call
// (2) get all VPC List by common-runtime API
// (3) return REST Json Format
func listAllVPC(c echo.Context) error {
        cblog.Info("call listAllVPC()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

	// Call common-runtime API
        rsType := rsVPC
	allResourceList, err := cmrt.ListAllResource(req.ConnectionName, rsType)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }	

        return c.JSON(http.StatusOK, &allResourceList)
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

	// set NameId for SubnetInfo List
	// create new SubnetInfo List
	subnetInfoList := []cres.SubnetInfo{}
        for i, subnetInfo := range info.SubnetInfoList {
                subnetIIDInfo, err := iidRWLock.GetIIDbySystemID(req.ConnectionName, rsSubnetPrefix + info.IId.NameId, subnetInfo.IId)
                if err != nil {
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
                }
		if subnetIIDInfo.IId.NameId != "" { // insert only this user created.
			info.SubnetInfoList[i].IId.NameId = subnetIIDInfo.IId.NameId
			subnetInfoList = append(subnetInfoList, info.SubnetInfoList[i])
		}
        }
	info.SubnetInfoList = subnetInfoList
	return c.JSON(http.StatusOK, &info)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func deleteVPC(c echo.Context) error {
	cblog.Info("call deleteVPC()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

	// Call common-runtime API
        result, _, err := cmrt.DeleteResource(req.ConnectionName, rsVPC, c.Param("Name"), c.QueryParam("force"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func deleteCSPVPC(c echo.Context) error {
        cblog.Info("call deleteCSPVPC()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, rsVPC, c.Param("Id"))
        if err != nil {
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
		ReqInfo struct {
			Name string
			VPCName string
			Direction     string
			SecurityRules *[]cres.SecurityRuleInfo
		}
	}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

        // check the input Name to include the SecurityGroup Delimiter
        if strings.HasPrefix(req.ReqInfo.Name, sgDELIMITER) {
                return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf(sgDELIMITER + " cannot be used in SecurityGroup name!!"))
        }

	// Rest RegInfo => Driver ReqInfo
	reqInfo := cres.SecurityReqInfo {
			// SG NameID format => {VPC NameID} + sgDELIMITER + {SG NameID}
			// transform: SG NameID => {VPC NameID} + sgDELIMITER + {SG NameID}
			IId: cres.IID{req.ReqInfo.VPCName + sgDELIMITER + req.ReqInfo.Name, ""},
			VpcIID: cres.IID{req.ReqInfo.VPCName, ""},
			Direction: req.ReqInfo.Direction,
			SecurityRules: req.ReqInfo.SecurityRules,
		   }
//+++++++++++++++++++++++++++++++++++++++++++
        // set VPC SystemId
        vpcIIDInfo, err := iidRWLock.GetIID(req.ConnectionName, rsVPC, reqInfo.VpcIID)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }
        reqInfo.VpcIID.SystemId = vpcIIDInfo.IId.SystemId
//+++++++++++++++++++++++++++++++++++++++++++
	
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
        bool_ret, err := iidRWLock.IsExistIID(req.ConnectionName, rsType, reqInfo.IId)
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        if bool_ret == true {
                return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!"))
        }

// (2) create Resource
	info, err := handler.CreateSecurity(reqInfo)
	if err != nil { return echo.NewHTTPError(http.StatusInternalServerError, err.Error()) }

	// set VPC NameId
	info.VpcIID.NameId = vpcIIDInfo.IId.NameId
	info.VpcIID.SystemId = vpcIIDInfo.IId.SystemId


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

	// set ResourceInfo(IID.NameId)
	// iidInfo.IId.NameID format => {VPC NameID} + sgDELIMITER + {SG NameID}
	vpc_sg_nameid := strings.Split(info.IId.NameId, sgDELIMITER)
	info.IId.NameId = vpc_sg_nameid[1]

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

				// set ResourceInfo(IID.NameId)
				// iidInfo.IId.NameID format => {VPC NameID} + sgDELIMITER + {SG NameID}
				vpc_sg_nameid := strings.Split(iidInfo.IId.NameId, sgDELIMITER)
				info.VpcIID.NameId = vpc_sg_nameid[0]
				info.IId.NameId = vpc_sg_nameid[1]

				// set VPC SystemId
				vpcIIDInfo, err := iidRWLock.GetIID(req.ConnectionName, rsVPC, info.VpcIID)
				if err != nil {
					return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
				}
				info.VpcIID.SystemId = vpcIIDInfo.IId.SystemId

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

// list all SGs for management
// (1) get args from REST Call
// (2) get all SG List by common-runtime API
// (3) return REST Json Format
func listAllSecurity(c echo.Context) error {
        cblog.Info("call listAllSecurity()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        rsType := rsSG
        allResourceList, err := cmrt.ListAllResource(req.ConnectionName, rsType)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, &allResourceList)
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
	// SG NameID format => {VPC NameID} + sgDELIMITER + {SG NameID}
        iidInfo, err := iidRWLock.FindIID(req.ConnectionName, rsType, c.Param("Name"))
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
	// set ResourceInfo(IID.NameId)
	// iidInfo.IId.NameID format => {VPC NameID} + sgDELIMITER + {SG NameID}
	vpc_sg_nameid := strings.Split(iidInfo.IId.NameId, sgDELIMITER)
	info.VpcIID.NameId = vpc_sg_nameid[0]
	info.IId.NameId = vpc_sg_nameid[1]

	// set VPC SystemId
	vpcIIDInfo, err := iidRWLock.GetIID(req.ConnectionName, rsVPC, info.VpcIID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	info.VpcIID.SystemId = vpcIIDInfo.IId.SystemId


	return c.JSON(http.StatusOK, &info)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func deleteSecurity(c echo.Context) error {
	cblog.Info("call deleteSecurity()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, _, err := cmrt.DeleteResource(req.ConnectionName, rsSG, c.Param("Name"), c.QueryParam("force"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func deleteCSPSecurity(c echo.Context) error {
        cblog.Info("call deleteCSPSecurity()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, rsSG, c.Param("Id"))
        if err != nil {
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
                ReqInfo struct {
			Name string
		}
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Rest RegInfo => Driver ReqInfo
        reqInfo := cres.KeyPairReqInfo {
                        IId: cres.IID{req.ReqInfo.Name, ""},
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
        bool_ret, err := iidRWLock.IsExistIID(req.ConnectionName, rsType, reqInfo.IId)
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        if bool_ret == true {
                return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf(reqInfo.IId.NameId + " already exists!"))
        }

// (2) create Resource
	info, err := handler.CreateKey(reqInfo)
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

        jsonResult.Result = infoList2
        return c.JSON(http.StatusOK, &jsonResult)
}

// list all KeyPairs for management
// (1) get args from REST Call
// (2) get all KeyPair List by common-runtime API
// (3) return REST Json Format
func listAllKey(c echo.Context) error {
        cblog.Info("call listAllKey()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        rsType := rsKey
        allResourceList, err := cmrt.ListAllResource(req.ConnectionName, rsType)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, &allResourceList)
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

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func deleteKey(c echo.Context) error {
	cblog.Info("call deleteKey()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, _, err := cmrt.DeleteResource(req.ConnectionName, rsKey, c.Param("Name"), c.QueryParam("force"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

	return c.JSON(http.StatusOK, &resultInfo)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func deleteCSPKey(c echo.Context) error {
        cblog.Info("call deleteCSPKey()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, rsKey, c.Param("Id"))
        if err != nil {
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


func getSetSystemId(ConnectionName string, reqInfo *cres.VMReqInfo) error {

        // set Image SystemId
	// @todo before Image Handling by powerkim
        reqInfo.ImageIID.SystemId = reqInfo.ImageIID.NameId

        // set VPC SystemId
	if reqInfo.VpcIID.NameId != "" {
		IIdInfo, err := iidRWLock.GetIID(ConnectionName, rsVPC, reqInfo.VpcIID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		reqInfo.VpcIID.SystemId = IIdInfo.IId.SystemId
	}

        // set Subnet SystemId
	if reqInfo.SubnetIID.NameId != "" {
		IIdInfo, err := iidRWLock.GetIID(ConnectionName, rsSubnetPrefix + reqInfo.VpcIID.NameId, reqInfo.SubnetIID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		reqInfo.SubnetIID.SystemId = IIdInfo.IId.SystemId
	}

        // set SecurityGroups SystemId
	for i, sgIID := range reqInfo.SecurityGroupIIDs {
		IIdInfo, err := iidRWLock.GetIID(ConnectionName, rsSG, sgIID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		reqInfo.SecurityGroupIIDs[i].SystemId = IIdInfo.IId.SystemId
	}

	// set KeyPair SystemId
	if reqInfo.KeyPairIID.NameId != "" {
		IIdInfo, err := iidRWLock.GetIID(ConnectionName, rsKey, reqInfo.KeyPairIID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		reqInfo.KeyPairIID.SystemId = IIdInfo.IId.SystemId
	}

	return nil
}

//================ VM Handler
// (1) check exist(NameID)
// (2) create Resource
// (3) insert IID
func startVM(c echo.Context) error {
	cblog.Info("call startVM()")

        var req struct {
                ConnectionName string
                ReqInfo struct {
			Name string
			ImageName string
			VPCName string
			SubnetName string
			SecurityGroupNames []string
			VMSpecName string
			KeyPairName string

			VMUserId string
			VMUserPasswd string
		}
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

	// Rest RegInfo => Driver ReqInfo
	// (1) create SecurityGroup IID List
	sgIIDList := []cres.IID{}
	for _, sgName := range req.ReqInfo.SecurityGroupNames {
		// SG NameID format => {VPC NameID} + sgDELIMITER + {SG NameID}
		// transform: SG NameID => {VPC NameID}-{SG NameID}
                sgIID := cres.IID{req.ReqInfo.VPCName + sgDELIMITER + sgName, ""}
                sgIIDList = append(sgIIDList, sgIID)
        }
	// (2) create VMReqInfo with SecurityGroup IID List
	reqInfo := cres.VMReqInfo {
		IId: cres.IID{req.ReqInfo.Name, ""},
		ImageIID: cres.IID{req.ReqInfo.ImageName, ""}, 
		VpcIID: cres.IID{req.ReqInfo.VPCName, ""},
		SubnetIID: cres.IID{req.ReqInfo.SubnetName, ""},
		SecurityGroupIIDs: sgIIDList,

		VMSpecName: req.ReqInfo.VMSpecName,
		KeyPairIID: cres.IID{req.ReqInfo.KeyPairName, ""},

		VMUserId: req.ReqInfo.VMUserId,
		VMUserPasswd: req.ReqInfo.VMUserPasswd,
	   }

	// get & set SystemId
	err := getSetSystemId(req.ConnectionName, &reqInfo)
	if err != nil {
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
// vmRWLock.Lock() @todo undo this until supporting async call. by powerkim, 2020.05.10
// defer vmRWLock.Unlock() @todo undo this until supporting async call. by powerkim, 2020.05.10
// (1) check exist(NameID)
        bool_ret, err := iidRWLock.IsExistIID(req.ConnectionName, rsType, reqInfo.IId)
        if err != nil {
                cblog.Error(err)
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        if bool_ret == true {
                return echo.NewHTTPError(http.StatusInternalServerError, fmt.Errorf(rsType + "-" + reqInfo.IId.NameId + " already exists!"))
        }

// (2) create Resource
	info, err := handler.StartVM(reqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}
	// set NameId for info by reqInfo
	setNameId(req.ConnectionName, &info, &reqInfo)

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

        // set sg NameId from VPCNameId-SecurityGroupNameId
        // IID.NameID format => {VPC NameID} + sgDELIMITER + {SG NameID}
	for i, sgIID := range info.SecurityGroupIIds {
		vpc_sg_nameid := strings.Split(sgIID.NameId, sgDELIMITER)
		info.SecurityGroupIIds[i].NameId = vpc_sg_nameid[1]
	}

	return c.JSON(http.StatusOK, &info)
}

func setNameId(ConnectionName string, vmInfo *cres.VMInfo, reqInfo *cres.VMReqInfo) error {

        // set Image SystemId
        // @todo before Image Handling by powerkim
	if reqInfo.ImageIID.NameId != "" {
		vmInfo.ImageIId.NameId = reqInfo.ImageIID.NameId
	}

        // set VPC SystemId
	if reqInfo.VpcIID.NameId != "" {
		vmInfo.VpcIID.NameId = reqInfo.VpcIID.NameId
	}

	if reqInfo.SubnetIID.NameId != "" {
		// set Subnet SystemId
		vmInfo.SubnetIID.NameId = reqInfo.SubnetIID.NameId
	}

	vmInfo.SecurityGroupIIds = reqInfo.SecurityGroupIIDs

        // set SecurityGroups SystemId
        for i, sgIID := range reqInfo.SecurityGroupIIDs {
                IIdInfo, err := iidRWLock.GetIIDbySystemID(ConnectionName, rsSG, sgIID)
                if err != nil {
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
                }
                reqInfo.SecurityGroupIIDs[i].NameId = IIdInfo.IId.NameId
        }

	if reqInfo.KeyPairIID.NameId != "" {
		// set KeyPair SystemId
		vmInfo.KeyPairIId.NameId = reqInfo.KeyPairIID.NameId
	}

        return nil
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

//+++++++++++++++++++++++++++++++++++++++++++
// set ResourceInfo(IID.NameId)
        // set VPC NameId
        IIdInfo, err := iidRWLock.GetIIDbySystemID(req.ConnectionName, rsType, info.IId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }
        info.IId.NameId = IIdInfo.IId.NameId
//+++++++++++++++++++++++++++++++++++++++++++
	err = getSetNameId(req.ConnectionName, info)
	if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

				// set sg NameId from VPCNameId-SecurityGroupNameId
				// IID.NameID format => {VPC NameID} + sgDELIMITER + {SG NameID}
				for i, sgIID := range info.SecurityGroupIIds {
					vpc_sg_nameid := strings.Split(sgIID.NameId, sgDELIMITER)
					info.SecurityGroupIIds[i].NameId = vpc_sg_nameid[1]
				}

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

func getSetNameId(ConnectionName string, vmInfo *cres.VMInfo) error {

        // set Image NameId
        // @todo before Image Handling by powerkim
        //vmInfo.ImageIId.NameId = vmInfo.ImageIId.SystemId

	if vmInfo.VpcIID.SystemId != "" {
		// set VPC NameId
		IIdInfo, err := iidRWLock.GetIIDbySystemID(ConnectionName, rsVPC, vmInfo.VpcIID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		vmInfo.VpcIID.NameId = IIdInfo.IId.NameId
	}

	if vmInfo.SubnetIID.SystemId != "" {
		// set Subnet NameId
		IIdInfo, err := iidRWLock.GetIIDbySystemID(ConnectionName, rsSubnetPrefix + vmInfo.VpcIID.NameId, vmInfo.SubnetIID)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		vmInfo.SubnetIID.NameId = IIdInfo.IId.NameId
	}

        // set SecurityGroups NameId
        for i, sgIID := range vmInfo.SecurityGroupIIds {
                IIdInfo, err := iidRWLock.GetIIDbySystemID(ConnectionName, rsSG, sgIID)
                if err != nil {
                        return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
                }
                vmInfo.SecurityGroupIIds[i].NameId = IIdInfo.IId.NameId
        }

	if vmInfo.KeyPairIId.SystemId != "" {
		// set KeyPair NameId
		IIdInfo, err := iidRWLock.GetIIDbySystemID(ConnectionName, rsKey, vmInfo.KeyPairIId)
		if err != nil {
			return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
		}
		vmInfo.KeyPairIId.NameId = IIdInfo.IId.NameId
	}

        return nil
}

// list all VMs for management
// (1) get args from REST Call
// (2) get all VM List by common-runtime API
// (3) return REST Json Format
func listAllVM(c echo.Context) error {
        cblog.Info("call listAllVM()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        rsType := rsVM
        allResourceList, err := cmrt.ListAllResource(req.ConnectionName, rsType)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, &allResourceList)
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

	err = getSetNameId(req.ConnectionName, &info)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // set sg NameId from VPCNameId-SecurityGroupNameId
        // IID.NameID format => {VPC NameID} + sgDELIMITER + {SG NameID}
        for i, sgIID := range info.SecurityGroupIIds {
                vpc_sg_nameid := strings.Split(sgIID.NameId, sgDELIMITER)
                info.SecurityGroupIIds[i].NameId = vpc_sg_nameid[1]
        }

	return c.JSON(http.StatusOK, &info)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func terminateVM(c echo.Context) error {
	cblog.Info("call terminateVM()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        _, result, err := cmrt.DeleteResource(req.ConnectionName, rsVM, c.Param("Name"), c.QueryParam("force"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := StatusInfo{
                Status: string(result),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func terminateCSPVM(c echo.Context) error {
        cblog.Info("call terminateCSPVM()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        _, result, err := cmrt.DeleteCSPResource(req.ConnectionName, rsVM, c.Param("Id"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := StatusInfo{
                Status: string(result),
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
				info.IId.NameId = iidInfo.IId.NameId
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
