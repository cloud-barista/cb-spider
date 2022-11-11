// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2020.

package restruntime

import (

        cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
        cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

        // REST API (echo)
        "net/http"

        "github.com/labstack/echo/v4"

        "strconv"
)

//================ VM Handler


func GetVMUsingRS(c echo.Context) error {
        cblog.Info("call GetVMUsingRS()")

        var req struct {
                ConnectionName string
                ReqInfo        struct {
                        CSPId          string
                }
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.GetVMUsingRS(req.ConnectionName, req.ReqInfo.CSPId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

type vmRegisterReq struct {
        ConnectionName string
        ReqInfo        struct {
                Name           string
                CSPId          string
        }
}

func RegisterVM(c echo.Context) error {
        cblog.Info("call RegisterVM()")

        req := vmRegisterReq{}

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // create UserIID
        userIId := cres.IID{req.ReqInfo.Name, req.ReqInfo.CSPId}

        // Call common-runtime API
        result, err := cmrt.RegisterVM(req.ConnectionName, userIId)
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func UnregisterVM(c echo.Context) error {
        cblog.Info("call UnregisterVM()")

        var req struct {
                ConnectionName string
        }

        if err := c.Bind(&req); err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        // Call common-runtime API
        result, err := cmrt.UnregisterResource(req.ConnectionName, rsVM, c.Param("Name"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        resultInfo := BooleanInfo{
                Result: strconv.FormatBool(result),
        }

        return c.JSON(http.StatusOK, &resultInfo)
}


func StartVM(c echo.Context) error {
	cblog.Info("call StartVM()")

	var req struct {
		ConnectionName string
		ReqInfo        struct {
			Name               string
			ImageType          string
			ImageName          string
			VPCName            string
			SubnetName         string
			SecurityGroupNames []string
			VMSpecName         string
			KeyPairName        string

			RootDiskType       string
			RootDiskSize       string

			DataDiskNames      []string

			VMUserId     string
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
		// SG NameID format => {VPC NameID} + cm.SG_DELIMITER + {SG NameID}
		// transform: SG NameID => {VPC NameID}-{SG NameID}
		//sgIID := cres.IID{req.ReqInfo.VPCName + cm.SG_DELIMITER + sgName, ""}
		sgIID := cres.IID{sgName, ""}
		sgIIDList = append(sgIIDList, sgIID)
	}

	// (2) create DataDisk IID List
        diskIIDList := []cres.IID{}
        for _, diskName := range req.ReqInfo.DataDiskNames {
                diskIID := cres.IID{diskName, ""}
                diskIIDList = append(diskIIDList, diskIID)
        }	

	// (3) create VMReqInfo with SecurityGroup & diskIID IID List
	reqInfo := cres.VMReqInfo{
		IId:               cres.IID{req.ReqInfo.Name, ""},
		ImageType:         cres.ImageType(req.ReqInfo.ImageType),
		ImageIID:          cres.IID{req.ReqInfo.ImageName, ""},
		VpcIID:            cres.IID{req.ReqInfo.VPCName, ""},
		SubnetIID:         cres.IID{req.ReqInfo.SubnetName, ""},
		SecurityGroupIIDs: sgIIDList,

		VMSpecName: req.ReqInfo.VMSpecName,
		KeyPairIID: cres.IID{req.ReqInfo.KeyPairName, ""},

		RootDiskType: req.ReqInfo.RootDiskType,
		RootDiskSize: req.ReqInfo.RootDiskSize,

		DataDiskIIDs: diskIIDList,

		VMUserId:     req.ReqInfo.VMUserId,
		VMUserPasswd: req.ReqInfo.VMUserPasswd,
	}

	// Call common-runtime API
	result, err := cmrt.StartVM(req.ConnectionName, rsVM, reqInfo)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

func ListVM(c echo.Context) error {
	cblog.Info("call ListVM()")

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
	result, err := cmrt.ListVM(req.ConnectionName, rsVM)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result []*cres.VMInfo `json:"vm"`
	}
	jsonResult.Result = result

	return c.JSON(http.StatusOK, &jsonResult)
}

// list all VMs for management
// (1) get args from REST Call
// (2) get all VM List by common-runtime API
// (3) return REST Json Format
func ListAllVM(c echo.Context) error {
	cblog.Info("call ListAllVM()")

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
	allResourceList, err := cmrt.ListAllResource(req.ConnectionName, rsVM)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &allResourceList)
}

func GetVM(c echo.Context) error {
	cblog.Info("call GetVM()")

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
	result, err := cmrt.GetVM(req.ConnectionName, rsVM, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

func GetCSPVM(c echo.Context) error {
        cblog.Info("call GetCSPVM()")

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
        result, err := cmrt.GetCSPVM(req.ConnectionName, rsVM, c.Param("Id"))
        if err != nil {
                return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
        }

        return c.JSON(http.StatusOK, result)
}

// (1) get args from REST Call
// (2) call common-runtime API
// (3) return REST Json Format
func TerminateVM(c echo.Context) error {
	cblog.Info("call TerminateVM()")

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
func TerminateCSPVM(c echo.Context) error {
	cblog.Info("call TerminateCSPVM()")

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

func ListVMStatus(c echo.Context) error {
	cblog.Info("call ListVMStatus()")

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
	result, err := cmrt.ListVMStatus(req.ConnectionName, rsVM)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult struct {
		Result []*cres.VMStatusInfo `json:"vmstatus"`
	}
	jsonResult.Result = result

	return c.JSON(http.StatusOK, &jsonResult)
}

func GetVMStatus(c echo.Context) error {
	cblog.Info("call GetVMStatus()")

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
	result, err := cmrt.GetVMStatus(req.ConnectionName, rsVM, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := StatusInfo{
		Status: string(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

func ControlVM(c echo.Context) error {
	cblog.Info("call ControlVM()")

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
	result, err := cmrt.ControlVM(req.ConnectionName, rsVM, c.Param("Name"), c.QueryParam("action"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := StatusInfo{
		Status: string(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}
