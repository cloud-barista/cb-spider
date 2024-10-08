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

// VMUsingResources represents the structure of the resources associated with a VM.
type VMUsingResources struct {
	Resources struct {
		VPC    *cres.IID   `json:"VPC" validate:"required"`              // example:"{NameId: 'vpc-01', SystemId: 'vpc-12345678'}"
		SGList []*cres.IID `json:"SGList" validate:"required"`           // example:"[{NameId: 'sg-01', SystemId: 'sg-12345678'}]"
		VMKey  *cres.IID   `json:"VMKey,omitempty" validate:"omitempty"` // example:"{NameId: 'keypair-01', SystemId: 'keypair-12345678'}"
	} `json:"Resources" validate:"required"`
}

// getVMUsingRS godoc
// @ID get-vm-using-rs
// @Summary Get VM Using Resource
// @Description Retrieve details of a VM using resource ID.
// @Tags [VM Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "Connection name for the VM"
// @Param CSPId query string true "CSP ID of the VM"
// @Success 200 {object} restruntime.VMUsingResources "Details of the VM using resource"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameters"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /getvmusingresources [post]
func GetVMUsingRS(c echo.Context) error {
	cblog.Info("call GetVMUsingRS()")

	// Parse query parameters
	connectionName := c.QueryParam("ConnectionName")
	cspID := c.QueryParam("CSPId")

	// Validate required parameters
	if connectionName == "" || cspID == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "ConnectionName and CSPId are required")
	}

	// Call common-runtime API
	result, err := cmrt.GetVMUsingRS(connectionName, cspID)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// VMRegisterRequest represents the request body for registering a VM.
type VMRegisterRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		Name  string `json:"Name" validate:"required" example:"vm-01"`
		CSPId string `json:"CSPId" validate:"required" example:"csp-vm-1234"`
	} `json:"ReqInfo" validate:"required"`
}

// registerVM godoc
// @ID register-vm
// @Summary Register VM
// @Description Register a new Virtual Machine (VM) with the specified name and CSP ID.
// @Tags [VM Management]
// @Accept  json
// @Produce  json
// @Param VMRegisterRequest body restruntime.VMRegisterRequest true "Request body for registering a VM"
// @Success 200 {object} cres.VMInfo "Details of the registered VM"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regvm [post]
func RegisterVM(c echo.Context) error {
	cblog.Info("call RegisterVM()")

	req := VMRegisterRequest{}

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

// unregisterVM godoc
// @ID unregister-vm
// @Summary Unregister VM
// @Description Unregister a Virtual Machine (VM) with the specified name.
// @Tags [VM Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for unregistering a VM"
// @Param Name path string true "The name of the VM to unregister"
// @Success 200 {object} BooleanInfo "Result of the unregister operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regvm/{Name} [delete]
func UnregisterVM(c echo.Context) error {
	cblog.Info("call UnregisterVM()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.UnregisterResource(req.ConnectionName, VM, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// VMStartRequest represents the request body for starting a VM.
type VMStartRequest struct {
	ConnectionName  string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	IDTransformMode string `json:"IDTransformMode,omitempty" validate:"omitempty" example:"ON"` // ON: transform CSP ID, OFF: no-transform CSP ID
	ReqInfo         struct {
		Name       string `json:"Name" validate:"required" example:"vm-01"`
		ImageType  string `json:"ImageType" validate:"required" example:"PublicImage"` // PublicImage or MyImage
		ImageName  string `json:"ImageName" validate:"required" example:"ami-12345678"`
		VMSpecName string `json:"VMSpecName" validate:"required" example:"t2.micro"`

		VPCName            string   `json:"VPCName" validate:"required" example:"vpc-01"`
		SubnetName         string   `json:"SubnetName" validate:"required" example:"subnet-01"`
		SecurityGroupNames []string `json:"SecurityGroupNames" validate:"required" example:"sg-01,sg-02"`
		KeyPairName        string   `json:"KeyPairName" validate:"required" example:"keypair-01"`

		RootDiskType  string   `json:"RootDiskType,omitempty" validate:"omitempty" example:"gp2"`                         // gp2 or default, if not specified, default is used
		RootDiskSize  string   `json:"RootDiskSize,omitempty" validate:"omitempty" example:"30"`                          // 100 or default, if not specified, default is used (unit is GB)
		DataDiskNames []string `json:"DataDiskNames,omitempty" validate:"omitempty" example:"data-disk-01, data-disk-02"` // Data disks in the same zone as this VM

		VMUserId     string `json:"VMUserId,omitempty" validate:"omitempty" example:"Administrator"`    // Administrator, Windows Only
		VMUserPasswd string `json:"VMUserPasswd,omitempty" validate:"omitempty" example:"password1234"` // Windows Only

		TagList []cres.KeyValue `json:"TagList,omitempty" validate:"omitempty"`
	} `json:"ReqInfo" validate:"required"`
}

// startVM godoc
// @ID start-vm
// @Summary Start VM
// @Description Start a new Virtual Machine (VM) with specified configurations. 🕷️ [[User Guide](https://github.com/cloud-barista/cb-spider/wiki/features-and-usages#2-%EB%A9%80%ED%8B%B0%ED%81%B4%EB%9D%BC%EC%9A%B0%EB%93%9C-vm-%EC%9D%B8%ED%94%84%EB%9D%BC-%EC%9E%90%EC%9B%90-%EC%A0%9C%EC%96%B4multi-cloud-vm-infra-resource-control)], [[Snapshot-MyImage,Disk Guide](https://github.com/cloud-barista/cb-spider/wiki/VM-Snapshot,-MyImage-and-Disk-Overview)]
// @Tags [VM Management]
// @Accept  json
// @Produce  json
// @Param VMStartRequest body restruntime.VMStartRequest true "Request body for starting a VM"
// @Success 200 {object} cres.VMInfo "Details of the started VM"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vm [post]
func StartVM(c echo.Context) error {
	cblog.Info("call StartVM()")

	req := VMStartRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Rest RegInfo => Driver ReqInfo
	// (1) create SecurityGroup IID List
	sgIIDList := []cres.IID{}
	for _, sgName := range req.ReqInfo.SecurityGroupNames {
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
		ImageIID:          cres.IID{req.ReqInfo.ImageName, req.ReqInfo.ImageName},
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

		TagList: req.ReqInfo.TagList,
	}

	// Call common-runtime API
	result, err := cmrt.StartVM(req.ConnectionName, VM, reqInfo, req.IDTransformMode)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// VMListResponse represents the response body structure for listing VMs.
type VMListResponse struct {
	VMs []*cres.VMInfo `json:"vm" validate:"required"`
}

// listVM godoc
// @ID list-vm
// @Summary List VMs
// @Description Retrieve a list of Virtual Machines (VMs) associated with a specific connection.
// @Tags [VM Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list VMs for"
// @Success 200 {object} VMListResponse "List of VMs"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vm [get]
func ListVM(c echo.Context) error {
	cblog.Info("call ListVM()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.ListVM(req.ConnectionName, VM)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	jsonResult := VMListResponse{
		VMs: result,
	}

	return c.JSON(http.StatusOK, &jsonResult)
}

// listAllVM godoc
// @ID list-all-vm
// @Summary List All VMs in a Connection
// @Description Retrieve a comprehensive list of all Virtual Machines (VMs) associated with a specific connection, <br> including those mapped between CB-Spider and the CSP, <br> only registered in CB-Spider's metadata, <br> and only existing in the CSP.
// @Tags [VM Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list VMs for"
// @Success 200 {object} AllResourceListResponse "List of all VMs within the specified connection, including VMs in CB-Spider only, CSP only, and mapped between both."
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /allvm [get]
func ListAllVM(c echo.Context) error {
	cblog.Info("call ListAllVM()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	allResourceList, err := cmrt.ListAllResource(req.ConnectionName, VM)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &allResourceList)
}

// getVM godoc
// @ID get-vm
// @Summary Get VM
// @Description Retrieve details of a specific Virtual Machine (VM).
// @Tags [VM Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to get a VM for"
// @Param Name path string true "The name of the VM to retrieve"
// @Success 200 {object} cres.VMInfo "Details of the VM"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vm/{Name} [get]
func GetVM(c echo.Context) error {
	cblog.Info("call GetVM()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.GetVM(req.ConnectionName, VM, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// getCSPVM godoc
// @ID get-csp-vm
// @Summary Get CSP VM
// @Description Retrieve details of a specific CSP Virtual Machine (VM).
// @Tags [VM Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to get a CSP VM for"
// @Param Id path string true "The CSP VM ID to retrieve"
// @Success 200 {object} cres.VMInfo "Details of the CSP VM"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cspvm/{Id} [get]
func GetCSPVM(c echo.Context) error {
	cblog.Info("call GetCSPVM()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.GetCSPVM(req.ConnectionName, VM, c.Param("Id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// terminateVM godoc
// @ID terminate-vm
// @Summary Terminate VM
// @Description Terminate a specified Virtual Machine (VM).
// @Tags [VM Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for terminating a VM"
// @Param Name path string true "The name of the VM to terminate"
// @Param force query string false "Force terminate the VM. ex) true or false(default: false)"
// @Success 200 {object} VMStatusResponse "Result of the terminate operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vm/{Name} [delete]
func TerminateVM(c echo.Context) error {
	cblog.Info("call TerminateVM()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	_, result, err := cmrt.DeleteVM(req.ConnectionName, VM, c.Param("Name"), c.QueryParam("force"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := VMStatusResponse{
		Status: result,
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// terminateCSPVM godoc
// @ID terminate-csp-vm
// @Summary Terminate CSP VM
// @Description Terminate a specified CSP Virtual Machine (VM).
// @Tags [VM Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for terminating a CSP VM"
// @Param Id path string true "The CSP VM ID to terminate"
// @Success 200 {object} VMStatusResponse "Result of the terminate operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /cspvm/{Id} [delete]
func TerminateCSPVM(c echo.Context) error {
	cblog.Info("call TerminateCSPVM()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	_, result, err := cmrt.DeleteCSPResource(req.ConnectionName, VM, c.Param("Id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := VMStatusResponse{
		Status: result,
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// VMListStatusResponse represents the response structure for listing VM statuses.
type VMListStatusResponse struct {
	Result []*cres.VMStatusInfo `json:"vmstatus" validate:"required"`
}

// listVMStatus godoc
// @ID list-vm-status
// @Summary List VM Statuses
// @Description Retrieve a list of statuses for Virtual Machines (VMs) associated with a specific connection.
// @Tags [VM Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list VM statuses for"
// @Success 200 {object} VMListStatusResponse "List of VM statuses"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vmstatus [get]
func ListVMStatus(c echo.Context) error {
	cblog.Info("call ListVMStatus()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.ListVMStatus(req.ConnectionName, VM)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var jsonResult VMListStatusResponse
	jsonResult.Result = result

	return c.JSON(http.StatusOK, &jsonResult)
}

// VMStatusResponse represents the response body structure for VM status APIs.
type VMStatusResponse struct {
	Status cres.VMStatus `json:"Status" validate:"required" example:"Running"`
}

// getVMStatus godoc
// @ID get-vm-status
// @Summary Get VM Status
// @Description Retrieve the status of a specific Virtual Machine (VM).
// @Tags [VM Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to get a VM status for"
// @Param Name path string true "The name of the VM to retrieve the status of"
// @Success 200 {object} VMStatusResponse "Details of the VM status"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /vmstatus/{Name} [get]
func GetVMStatus(c echo.Context) error {
	cblog.Info("call GetVMStatus()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.GetVMStatus(req.ConnectionName, VM, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := VMStatusResponse{
		Status: result,
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// controlVM godoc
// @ID control-vm
// @Summary Control VM
// @Description Control the state of a Virtual Machine (VM) such as suspend, resume, or reboot.
// @Tags [VM Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for controlling a VM"
// @Param Name path string true "The name of the VM to control"
// @Param action query string true "The action to perform on the VM (suspend, resume, reboot)"
// @Success 200 {object} VMStatusResponse "Result of the control operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /controlvm/{Name} [put]
func ControlVM(c echo.Context) error {
	cblog.Info("call ControlVM()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.ControlVM(req.ConnectionName, VM, c.Param("Name"), c.QueryParam("action"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := VMStatusResponse{
		Status: result,
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// countAllVMs godoc
// @ID count-all-vm
// @Summary Count All VMs
// @Description Get the total number of Virtual Machines (VMs) across all connections.
// @Tags [VM Management]
// @Produce  json
// @Success 200 {object} CountResponse "Total count of VMs"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countvm [get]
func CountAllVMs(c echo.Context) error {
	// Call common-runtime API to get count of VMs
	count, err := cmrt.CountAllVMs()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Prepare JSON result
	jsonResult := CountResponse{
		Count: int(count),
	}

	// Return JSON response
	return c.JSON(http.StatusOK, jsonResult)
}

// countVMsByConnection godoc
// @ID count-vm-by-connection
// @Summary Count VMs by Connection
// @Description Get the total number of Virtual Machines (VMs) for a specific connection.
// @Tags [VM Management]
// @Produce  json
// @Param ConnectionName path string true "The name of the Connection"
// @Success 200 {object} CountResponse "Total count of VMs for the connection"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countvm/{ConnectionName} [get]
func CountVMsByConnection(c echo.Context) error {
	// Call common-runtime API to get count of VMs
	count, err := cmrt.CountVMsByConnection(c.Param("ConnectionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Prepare JSON result
	jsonResult := CountResponse{
		Count: int(count),
	}

	// Return JSON response
	return c.JSON(http.StatusOK, jsonResult)
}
