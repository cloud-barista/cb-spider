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

        "github.com/labstack/echo/v4"

        "strconv"
)

//================ Disk Handler

type DiskRegisterReq struct {
        ConnectionName string
        ReqInfo        struct {
                Name           string
                CSPId          string
        }
}

func RegisterDisk(c echo.Context) error {
        cblog.Info("call RegisterDisk()")

        req := DiskRegisterReq{}

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // create UserIID
        userIId := cres.IID{req.ReqInfo.Name, req.ReqInfo.CSPId}

        // Call common-runtime API
        result, err := cmrt.RegisterDisk(req.ConnectionName, userIId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func UnregisterDisk(c echo.Context) error {
        cblog.Info("call UnregisterDisk()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.UnregisterResource(req.ConnectionName, rsDisk, c.Param("Name"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}

type DiskReq struct {
        ConnectionName string
        ReqInfo        struct {
                Name            string

                DiskType        string
                DiskSize        string
        }
}

func CreateDisk(c echo.Context) error {
        cblog.Info("call CreateDisk()")

        req := DiskReq{}

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Rest RegInfo => Driver ReqInfo
        reqInfo := cres.DiskInfo{
                IId:           cres.IID{req.ReqInfo.Name, req.ReqInfo.Name},
                DiskType:           req.ReqInfo.DiskType,
                DiskSize:           req.ReqInfo.DiskSize,
        }

        // Call common-runtime API
        result, err := cmrt.CreateDisk(req.ConnectionName, rsDisk, reqInfo)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

func ListDisk(c echo.Context) error {
        cblog.Info("call ListDisk()")

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
        result, err := cmrt.ListDisk(req.ConnectionName, rsDisk)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        var jsonResult struct {
                Result []*cres.DiskInfo `json:"disk"`
        }
        jsonResult.Result = result
        return c.JSON(http.StatusOK, &jsonResult)
}

// list all Disks for management
// (1) get args from REST Call
// (2) get all Disk List by common-runtime API
// (3) return REST Json Format
func ListAllDisk(c echo.Context) error {
        cblog.Info("call ListAllDisk()")

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
        allResourceList, err := cmrt.ListAllResource(req.ConnectionName, rsDisk)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, &allResourceList)
}

func GetDisk(c echo.Context) error {
        cblog.Info("call GetDisk()")

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
        result, err := cmrt.GetDisk(req.ConnectionName, rsDisk, c.Param("Name"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

func ChangeDiskSize(c echo.Context) error {
        cblog.Info("call ChangeDiskSize()")

        var req struct {
                ConnectionName string
                ReqInfo        struct {
                        Size string
                }
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // To support for Get-Query Param Type API
        if req.ConnectionName == "" {
                req.ConnectionName = c.QueryParam("ConnectionName")
        }

        // Call common-runtime API
        result, err := cmrt.ChangeDiskSize(req.ConnectionName, c.Param("Name"), req.ReqInfo.Size)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func DeleteDisk(c echo.Context) error {
        cblog.Info("call DeleteDisk()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, _, err := cmrt.DeleteResource(req.ConnectionName, rsDisk, c.Param("Name"), c.QueryParam("force"))
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
func DeleteCSPDisk(c echo.Context) error {
        cblog.Info("call DeleteCSPDisk()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, rsDisk, c.Param("Id"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}

func AttachDisk(c echo.Context) error {
        cblog.Info("call AttachDisk()")

        var req struct {
                ConnectionName string
                ReqInfo        struct {
                        VMName string
                }
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.AttachDisk(req.ConnectionName, c.Param("Name"), req.ReqInfo.VMName)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

func DetachDisk(c echo.Context) error {
        cblog.Info("call DetachDisk()")

        var req struct {
                ConnectionName string
                ReqInfo        struct {
                        VMName string
                }
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.DetachDisk(req.ConnectionName, c.Param("Name"), req.ReqInfo.VMName)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}


