// Cloud Control Manager's Rest Runtime of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, April 2026.

package restruntime

import (
	cmrt "github.com/cloud-barista/cb-spider/api-runtime/common-runtime"
	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	// REST API (echo)
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"strconv"
)

//================ RDBMS Handler

// RDBMSInfoResponse mirrors cres.RDBMSInfo minus MasterUserName/MasterUserPassword. It is the
// single response shape for every RDBMS read/write endpoint (Create, Register, Get, List) --
// echoing those fields back would be a pointless round-trip of a value the caller already has.
// The driver-facing cres.RDBMSInfo struct itself is untouched, since drivers also use it as
// CreateRDBMS's *request* parameter to receive the master credentials they need to actually
// provision the CSP account -- only this REST-layer response shape is affected.
type RDBMSInfoResponse struct {
	IId    cres.IID `json:"IId"`
	VpcIID cres.IID `json:"VpcIID"`

	DBEngine        string `json:"DBEngine" example:"mysql"`
	DBEngineVersion string `json:"DBEngineVersion" example:"8.0"`

	DBSpec         string `json:"DBSpec" example:"db.t3.medium"`
	DBInstanceType string `json:"DBInstanceType,omitempty" example:"Primary"`

	StorageType string `json:"StorageType,omitempty" example:"gp2"`
	StorageSize string `json:"StorageSize" example:"100"`
	Iops        string `json:"Iops,omitempty" example:"3000"`

	SubnetIIDs        []cres.IID `json:"SubnetIIDs,omitempty"`
	SecurityGroupIIDs []cres.IID `json:"SecurityGroupIIDs,omitempty"`

	HighAvailability bool `json:"HighAvailability,omitempty" default:"false"`

	BackupRetentionDays int    `json:"BackupRetentionDays,omitempty" example:"7"`
	BackupTime          string `json:"BackupTime,omitempty" example:"03:00"`

	PublicAccess bool   `json:"PublicAccess,omitempty" default:"false"`
	Endpoint     string `json:"Endpoint,omitempty"`

	NHNAutoOpenDBSecurityGroup bool `json:"NHNAutoOpenDBSecurityGroup,omitempty" default:"false"`

	Encryption bool `json:"Encryption,omitempty" default:"false"`

	DeletionProtection bool `json:"DeletionProtection,omitempty" default:"false"`

	Status cres.RDBMSStatus `json:"Status,omitempty" example:"Available"`

	CreatedTime  time.Time       `json:"CreatedTime,omitempty"`
	TagList      []cres.KeyValue `json:"TagList,omitempty"`
	KeyValueList []cres.KeyValue `json:"KeyValueList,omitempty"`
}

// toRDBMSInfoResponse copies every field of a driver-returned cres.RDBMSInfo except
// MasterUserName/MasterUserPassword.
func toRDBMSInfoResponse(info *cres.RDBMSInfo) *RDBMSInfoResponse {
	return &RDBMSInfoResponse{
		IId:                        info.IId,
		VpcIID:                     info.VpcIID,
		DBEngine:                   info.DBEngine,
		DBEngineVersion:            info.DBEngineVersion,
		DBSpec:                     info.DBSpec,
		DBInstanceType:             info.DBInstanceType,
		StorageType:                info.StorageType,
		StorageSize:                info.StorageSize,
		Iops:                       info.Iops,
		SubnetIIDs:                 info.SubnetIIDs,
		SecurityGroupIIDs:          info.SecurityGroupIIDs,
		HighAvailability:           info.HighAvailability,
		BackupRetentionDays:        info.BackupRetentionDays,
		BackupTime:                 info.BackupTime,
		PublicAccess:               info.PublicAccess,
		Endpoint:                   info.Endpoint,
		NHNAutoOpenDBSecurityGroup: info.NHNAutoOpenDBSecurityGroup,
		Encryption:                 info.Encryption,
		DeletionProtection:         info.DeletionProtection,
		Status:                     info.Status,
		CreatedTime:                info.CreatedTime,
		TagList:                    info.TagList,
		KeyValueList:               info.KeyValueList,
	}
}

// toRDBMSInfoResponseList converts a []interface{} of *cres.RDBMSInfo (as returned by the
// resource-type-agnostic ListAllResourceInfo's MappedInfoList/OnlyCSPInfoList) into concretely
// typed, credential-stripped entries. Non-*cres.RDBMSInfo elements are skipped defensively; in
// practice every element here is a *cres.RDBMSInfo (see CommonManager.go's fetchResourceInfoList,
// cres.RDBMS case).
func toRDBMSInfoResponseList(items []interface{}) []*RDBMSInfoResponse {
	result := make([]*RDBMSInfoResponse, 0, len(items))
	for _, item := range items {
		if info, ok := item.(*cres.RDBMSInfo); ok {
			result = append(result, toRDBMSInfoResponse(info))
		}
	}
	return result
}

// RDBMSRegisterRequest represents the request body for registering an RDBMS.
type RDBMSRegisterRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	ReqInfo        struct {
		VPCName string `json:"VPCName" validate:"required" example:"vpc-01"`
		Name    string `json:"Name" validate:"required" example:"rdbms-01"`
		CSPId   string `json:"CSPId" validate:"required" example:"csp-rdbms-1234"`
	} `json:"ReqInfo" validate:"required"`
}

// registerRDBMS godoc
// @ID register-rdbms
// @Summary Register RDBMS
// @Description Register a new RDBMS with the specified name and CSP ID.
// @Tags [RDBMS Management]
// @Accept  json
// @Produce  json
// @Param RDBMSRegisterRequest body restruntime.RDBMSRegisterRequest true "Request body for registering an RDBMS"
// @Success 200 {object} RDBMSInfoResponse "Details of the registered RDBMS"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regrdbms [post]
func RegisterRDBMS(c echo.Context) error {
	cblog.Info("call RegisterRDBMS()")

	req := RDBMSRegisterRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// create UserIID
	userIId := cres.IID{req.ReqInfo.Name, req.ReqInfo.CSPId}

	// Call common-runtime API
	result, err := cmrt.RegisterRDBMS(req.ConnectionName, req.ReqInfo.VPCName, userIId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, toRDBMSInfoResponse(result))
}

// unregisterRDBMS godoc
// @ID unregister-rdbms
// @Summary Unregister RDBMS
// @Description Unregister an RDBMS with the specified name.
// @Tags [RDBMS Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for unregistering an RDBMS"
// @Param Name path string true "The name of the RDBMS to unregister"
// @Success 200 {object} BooleanInfo "Result of the unregister operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /regrdbms/{Name} [delete]
func UnregisterRDBMS(c echo.Context) error {
	cblog.Info("call UnregisterRDBMS()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.UnregisterResource(req.ConnectionName, RDBMS, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// RDBMSCreateRequest represents the request body for creating an RDBMS.
type RDBMSCreateRequest struct {
	ConnectionName  string `json:"ConnectionName" validate:"required" example:"aws-connection"`
	IDTransformMode string `json:"IDTransformMode,omitempty" validate:"omitempty" example:"ON"` // ON: transform CSP ID, OFF: no-transform CSP ID
	ReqInfo         struct {
		Name    string `json:"Name" validate:"required" example:"rdbms-01"`
		VPCName string `json:"VPCName" validate:"required" example:"vpc-01"`

		DBEngine        string `json:"DBEngine" validate:"required" example:"mysql"`
		DBEngineVersion string `json:"DBEngineVersion" validate:"required" example:"8.0"`
		DBSpec          string `json:"DBSpec" validate:"required" example:"db.t3.medium"`
		StorageSize     string `json:"StorageSize" validate:"required" example:"100"` // in GB

		// StorageType: storage volume type. Use GetMetaInfo() to discover available options per CSP.
		// OpenStack: configurable at creation time, but Trove API does not return this field in responses (always "NA").
		StorageType string `json:"StorageType,omitempty" validate:"omitempty" example:"gp2"`
		// Iops: Provisioned IOPS for the storage volume.
		// AWS: required for io1/io2 (100-64000).
		// Other CSPs: not used.
		Iops string `json:"Iops,omitempty" validate:"omitempty" example:"3000"`

		SubnetNames        []string `json:"SubnetNames,omitempty" validate:"omitempty" example:"subnet-01"`
		SecurityGroupNames []string `json:"SecurityGroupNames,omitempty" validate:"omitempty" example:"sg-01"`

		MasterUserName     string `json:"MasterUserName" validate:"required" example:"admin"`
		MasterUserPassword string `json:"MasterUserPassword" validate:"required" example:"password123!"`

		HighAvailability    bool `json:"HighAvailability,omitempty" default:"false"`
		BackupRetentionDays int  `json:"BackupRetentionDays,omitempty" example:"7"` // Backup retention days (CSP will auto-assign backup time)

		PublicAccess       bool `json:"PublicAccess,omitempty" default:"false"`
		DeletionProtection bool `json:"DeletionProtection,omitempty" default:"false"`

		// NHNAutoOpenDBSecurityGroup (NHN Cloud only): requires PublicAccess=true.
		// When true, CB-Spider auto-creates and attaches a fully-open (0.0.0.0/0)
		// NHN Cloud RDS DB Security Group, and deletes it automatically when the
		// instance is deleted. Ignored by every other CSP. NHN Cloud RDBMS does
		// not use SecurityGroupNames at all (see NHNAutoOpenDBSecurityGroup
		// instead); if this flag is false (default), create and attach a DB
		// Security Group yourself via the NHN Cloud console/API for external
		// SQL access.
		NHNAutoOpenDBSecurityGroup bool `json:"NHNAutoOpenDBSecurityGroup,omitempty" default:"false"`

		TagList []cres.KeyValue `json:"TagList,omitempty" validate:"omitempty"`
	} `json:"ReqInfo" validate:"required"`
}

// createRDBMS godoc
// @ID create-rdbms
// @Summary Create RDBMS
// @Description Create a new Relational Database (RDBMS) with the specified configuration. <br> The response never echoes back MasterUserName/MasterUserPassword -- you already supplied them in this request, so keep them yourself, e.g. alongside your SSH private key.
// @Tags [RDBMS Management]
// @Accept  json
// @Produce  json
// @Param RDBMSCreateRequest body restruntime.RDBMSCreateRequest true "Request body for creating an RDBMS"
// @Success 200 {object} RDBMSInfoResponse "Details of the created RDBMS"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /rdbms [post]
func CreateRDBMS(c echo.Context) error {
	cblog.Info("call CreateRDBMS()")

	req := RDBMSCreateRequest{}

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if req.ReqInfo.VPCName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "VPCName is required")
	}

	// Build SubnetIIDs from names
	subnetIIDs := []cres.IID{}
	for _, name := range req.ReqInfo.SubnetNames {
		subnetIIDs = append(subnetIIDs, cres.IID{NameId: name, SystemId: ""})
	}

	// Build SecurityGroupIIDs from names
	sgIIDs := []cres.IID{}
	for _, name := range req.ReqInfo.SecurityGroupNames {
		sgIIDs = append(sgIIDs, cres.IID{NameId: name, SystemId: ""})
	}

	// Rest RegInfo => Driver ReqInfo
	reqInfo := cres.RDBMSInfo{
		IId:    cres.IID{NameId: req.ReqInfo.Name, SystemId: req.ReqInfo.Name},
		VpcIID: cres.IID{NameId: req.ReqInfo.VPCName, SystemId: ""},

		DBEngine:        req.ReqInfo.DBEngine,
		DBEngineVersion: req.ReqInfo.DBEngineVersion,
		DBSpec:          req.ReqInfo.DBSpec,
		StorageType:     req.ReqInfo.StorageType,
		StorageSize:     req.ReqInfo.StorageSize,
		Iops:            req.ReqInfo.Iops,

		SubnetIIDs:        subnetIIDs,
		SecurityGroupIIDs: sgIIDs,

		MasterUserName:     req.ReqInfo.MasterUserName,
		MasterUserPassword: req.ReqInfo.MasterUserPassword,

		HighAvailability:    req.ReqInfo.HighAvailability,
		BackupRetentionDays: req.ReqInfo.BackupRetentionDays,
		// BackupTime is not configurable at creation (CSP auto-assigns)

		PublicAccess:       req.ReqInfo.PublicAccess,
		DeletionProtection: req.ReqInfo.DeletionProtection,
		// Encryption is not configurable at creation (CSP default)

		NHNAutoOpenDBSecurityGroup: req.ReqInfo.NHNAutoOpenDBSecurityGroup,

		TagList: req.ReqInfo.TagList,
	}

	// Call common-runtime API
	result, err := cmrt.CreateRDBMS(req.ConnectionName, RDBMS, reqInfo, req.IDTransformMode)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, toRDBMSInfoResponse(result))
}

// RDBMSListResponse represents the response body for listing RDBMS instances.
type RDBMSListResponse struct {
	Result []*RDBMSInfoResponse `json:"rdbms" validate:"required" description:"A list of RDBMS information"`
}

// listRDBMS godoc
// @ID list-rdbms
// @Summary List RDBMS
// @Description Retrieve a list of RDBMS instances associated with a specific connection.
// @Tags [RDBMS Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list RDBMS for"
// @Success 200 {object} RDBMSListResponse "List of RDBMS instances"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /rdbms [get]
func ListRDBMS(c echo.Context) error {
	cblog.Info("call ListRDBMS()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.ListRDBMS(req.ConnectionName, RDBMS)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	infoList := make([]*RDBMSInfoResponse, len(result))
	for i, info := range result {
		infoList[i] = toRDBMSInfoResponse(info)
	}

	jsonResult := RDBMSListResponse{
		Result: infoList,
	}

	return c.JSON(http.StatusOK, &jsonResult)
}

// listAllRDBMS godoc
// @ID list-all-rdbms
// @Summary List All RDBMS in a Connection
// @Description Retrieve a comprehensive list of all RDBMS instances associated with a specific connection, <br> including those mapped between CB-Spider and the CSP, <br> only registered in CB-Spider's metadata, <br> and only existing in the CSP.
// @Tags [RDBMS Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list RDBMS for"
// @Success 200 {object} AllResourceListResponse "List of all RDBMS instances within the specified connection, including RDBMS in CB-Spider only, CSP only, and mapped between both."
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /allrdbms [get]
func ListAllRDBMS(c echo.Context) error {
	cblog.Info("call ListAllRDBMS()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	allResourceList, err := cmrt.ListAllResource(req.ConnectionName, RDBMS)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &allResourceList)
}

// RDBMSAllInfoListResponse represents the response body for ListAllRDBMSInfo. RDBMS gets its own
// type here instead of reusing the resource-type-agnostic AllResourceInfoListResponse (whose
// MappedInfoList/OnlyCSPInfoList are untyped []interface{}, shared across every resource type) so
// that MasterUserName/MasterUserPassword are genuinely absent from the documented schema and the
// actual response, not just emptied out -- consistent with Create/Register/List/Get RDBMS.
type RDBMSAllInfoListResponse struct {
	ResourceType cres.RSType `json:"ResourceType" example:"rdbms" description:"The type of resource"`
	AllListInfo  struct {
		MappedInfoList  []*RDBMSInfoResponse `json:"MappedInfoList" description:"A list of resources that are mapped between CB-Spider and CSP"`
		OnlySpiderList  []*cres.IID          `json:"OnlySpiderList" description:"A list of resources that exist only in CB-Spider"`
		OnlyCSPInfoList []*RDBMSInfoResponse `json:"OnlyCSPInfoList" description:"A list of resources that exist only in the CSP"`
	} `json:"AllListInfo" description:"A list of all resources info with their respective lists"`
}

// listAllRDBMSInfo godoc
// @ID list-all-rdbms-info
// @Summary List All RDBMS Info
// @Description Retrieve a comprehensive list of all RDBMS information associated with a specific connection, <br> including those mapped between CB-Spider and the CSP, <br> only registered in CB-Spider's metadata, <br> and only existing in the CSP.
// @Tags [RDBMS Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list RDBMS information for"
// @Success 200 {object} RDBMSAllInfoListResponse "List of all RDBMS information within the specified connection, including RDBMS in CB-Spider only, CSP only, and mapped between both."
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /allrdbmsinfo [get]
func ListAllRDBMSInfo(c echo.Context) error {
	cblog.Info("call ListAllRDBMSInfo()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	allResourceInfoList, err := cmrt.ListAllResourceInfo(req.ConnectionName, cres.RDBMS)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	var result RDBMSAllInfoListResponse
	result.ResourceType = allResourceInfoList.ResourceType
	result.AllListInfo.OnlySpiderList = allResourceInfoList.AllListInfo.OnlySpiderList
	result.AllListInfo.MappedInfoList = toRDBMSInfoResponseList(allResourceInfoList.AllListInfo.MappedInfoList)
	result.AllListInfo.OnlyCSPInfoList = toRDBMSInfoResponseList(allResourceInfoList.AllListInfo.OnlyCSPInfoList)

	return c.JSON(http.StatusOK, &result)
}

// getRDBMS godoc
// @ID get-rdbms
// @Summary Get RDBMS
// @Description Retrieve details of a specific RDBMS instance.
// @Tags [RDBMS Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to get an RDBMS for"
// @Param Name path string true "The name of the RDBMS to retrieve"
// @Success 200 {object} RDBMSInfoResponse "Details of the RDBMS"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /rdbms/{Name} [get]
func GetRDBMS(c echo.Context) error {
	cblog.Info("call GetRDBMS()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}

	// Call common-runtime API
	result, err := cmrt.GetRDBMS(req.ConnectionName, RDBMS, c.Param("Name"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, toRDBMSInfoResponse(result))
}

// deleteRDBMS godoc
// @ID delete-rdbms
// @Summary Delete RDBMS
// @Description Delete a specified RDBMS instance.
// @Tags [RDBMS Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for deleting an RDBMS"
// @Param Name path string true "The name of the RDBMS to delete"
// @Param force query string false "Force delete the RDBMS. ex) true or false(default: false)"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /rdbms/{Name} [delete]
func DeleteRDBMS(c echo.Context) error {
	cblog.Info("call DeleteRDBMS()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, err := cmrt.DeleteRDBMS(req.ConnectionName, RDBMS, c.Param("Name"), c.QueryParam("force"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// deleteCSPRDBMS godoc
// @ID delete-csp-rdbms
// @Summary Delete CSP RDBMS
// @Description Delete a specified CSP RDBMS.
// @Tags [RDBMS Management]
// @Accept  json
// @Produce  json
// @Param ConnectionRequest body restruntime.ConnectionRequest true "Request body for deleting a CSP RDBMS"
// @Param Id path string true "The CSP RDBMS ID to delete"
// @Success 200 {object} BooleanInfo "Result of the delete operation"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid JSON structure or missing fields"
// @Failure 404 {object} SimpleMsg "Resource Not Found"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /csprdbms/{Id} [delete]
func DeleteCSPRDBMS(c echo.Context) error {
	cblog.Info("call DeleteCSPRDBMS()")

	var req ConnectionRequest

	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	// Call common-runtime API
	result, _, err := cmrt.DeleteCSPResource(req.ConnectionName, RDBMS, c.Param("Id"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	resultInfo := BooleanInfo{
		Result: strconv.FormatBool(result),
	}

	return c.JSON(http.StatusOK, &resultInfo)
}

// getRDBMSMetaInfo godoc
// @ID get-rdbms-metainfo
// @Summary Get RDBMS Meta Information
// @Description Retrieve CSP-specific RDBMS capability information (supported engines, features, storage options).
// @Tags [RDBMS Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection"
// @Param DBEngine query string true "DB engine name: mysql, mariadb, or postgresql"
// @Success 200 {object} cres.RDBMSMetaInfo "RDBMS MetaInfo for the CSP"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /rdbmsmetainfo [get]
func GetRDBMSMetaInfo(c echo.Context) error {
	cblog.Info("call GetRDBMSMetaInfo()")

	connectionName := c.QueryParam("ConnectionName")
	dbEngine := c.QueryParam("DBEngine")

	// Call common-runtime API
	result, err := cmrt.GetRDBMSMetaInfo(connectionName, dbEngine)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// RDBMSEngineListResponse represents the response body structure for the ListRDBMSEngine API.
type RDBMSEngineListResponse struct {
	Result []string `json:"rdbmsengine" validate:"required" description:"A list of RDBMS engines supported by the CSP for this connection"`
}

// listRDBMSEngine godoc
// @ID list-rdbms-engine
// @Summary List RDBMS Engines
// @Description Retrieve the list of RDBMS engines (e.g., mysql, mariadb, postgresql) that the CSP supports for a specific connection, derived from the connection's driver capability information (GET /driver/capability).
// @Tags [RDBMS Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to list supported RDBMS engines for"
// @Success 200 {object} RDBMSEngineListResponse "List of RDBMS engines supported by the CSP"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /rdbmsengine [get]
func ListRDBMSEngine(c echo.Context) error {
	cblog.Info("call ListRDBMSEngine()")

	connectionName := c.QueryParam("ConnectionName")

	result, err := cmrt.ListRDBMSEngine(connectionName)
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}

	jsonResult := RDBMSEngineListResponse{
		Result: result,
	}
	return c.JSON(http.StatusOK, &jsonResult)
}

// getRDBMSOwnerVPC godoc
// @ID get-rdbms-owner-vpc
// @Summary Get RDBMS Owner VPC
// @Description Retrieve the Owner VPC of a given RDBMS CSP ID.
// @Tags [RDBMS Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection"
// @Param CSPId query string true "The CSP RDBMS ID"
// @Success 200 {object} cres.IID "Owner VPC IID"
// @Failure 400 {object} SimpleMsg "Bad Request, possibly due to invalid query parameter"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /getrdbmsowner [get]
func GetRDBMSOwnerVPC(c echo.Context) error {
	cblog.Info("call GetRDBMSOwnerVPC()")

	connectionName := c.QueryParam("ConnectionName")
	cspId := c.QueryParam("CSPId")

	// Call common-runtime API
	result, err := cmrt.GetRDBMSOwnerVPC(connectionName, cspId)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// countAllRDBMS godoc
// @ID count-all-rdbms
// @Summary Count All RDBMS
// @Description Get the total number of RDBMS instances across all connections.
// @Tags [RDBMS Management]
// @Produce  json
// @Success 200 {object} CountResponse "Total count of RDBMS instances"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countrdbms [get]
func CountAllRDBMS(c echo.Context) error {
	count, err := cmrt.CountAllRDBMS()
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	jsonResult := CountResponse{
		Count: int(count),
	}

	return c.JSON(http.StatusOK, jsonResult)
}

// countRDBMSByConnection godoc
// @ID count-rdbms-by-connection
// @Summary Count RDBMS by Connection
// @Description Get the total number of RDBMS instances for a specific connection.
// @Tags [RDBMS Management]
// @Produce  json
// @Param ConnectionName path string true "The name of the Connection"
// @Success 200 {object} CountResponse "Total count of RDBMS instances for the connection"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /countrdbms/{ConnectionName} [get]
func CountRDBMSByConnection(c echo.Context) error {
	count, err := cmrt.CountRDBMSByConnection(c.Param("ConnectionName"))
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	jsonResult := CountResponse{
		Count: int(count),
	}

	return c.JSON(http.StatusOK, jsonResult)
}

//================ RDBMS Database Management (CSP-native API)

// RDBMSDatabaseRequest is used for database create/list/delete via CSP-native API.
type RDBMSDatabaseRequest struct {
	ConnectionName string `json:"ConnectionName" validate:"required" example:"ncp-korea1-config"`
	DatabaseName   string `json:"DatabaseName,omitempty" example:"mydb"` // required only for create/delete
	// MasterUserName/MasterUserPassword: required when the driver uses the SQL fallback (e.g. AWS,
	// IBM). CB-Spider no longer tracks these after creation (see RDBMSManager.go's
	// redactRDBMSMasterCredentials -- it cannot reliably confirm the actual master account for
	// several CSPs), so the caller must supply both, the same credentials set at instance creation.
	MasterUserName     string `json:"MasterUserName,omitempty" example:"myadmin"`
	MasterUserPassword string `json:"MasterUserPassword,omitempty" example:"P@ssw0rd"`
}

// RDBMSDatabaseListResponse wraps the list of databases returned by CSP API.
type RDBMSDatabaseListResponse struct {
	Databases []string `json:"Databases"`
}

// createRDBMSDatabase godoc
// @ID create-rdbms-database
// @Summary Create Database in RDBMS
// @Description Create a database inside an RDBMS instance using the CSP-native API.
// @Tags [RDBMS Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the RDBMS instance"
// @Param RDBMSDatabaseRequest body restruntime.RDBMSDatabaseRequest true "ConnectionName and DatabaseName"
// @Success 200 {object} SimpleMsg "Created"
// @Failure 400 {object} SimpleMsg "Bad Request"
// @Failure 501 {object} SimpleMsg "Not Supported by driver"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /rdbms/{Name}/databases [post]
func CreateRDBMSDatabase(c echo.Context) error {
	cblog.Info("call CreateRDBMSDatabase()")

	var req RDBMSDatabaseRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if req.ConnectionName == "" || req.DatabaseName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "ConnectionName and DatabaseName are required")
	}

	err := cmrt.CreateRDBMSDatabase(req.ConnectionName, c.Param("Name"), req.DatabaseName, req.MasterUserName, req.MasterUserPassword)
	if err != nil {
		if err == cmrt.ErrRDBMSDatabaseMgrNotSupported {
			return echo.NewHTTPError(http.StatusNotImplemented, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &SimpleMsg{Message: "created"})
}

// listRDBMSDatabases godoc
// @ID list-rdbms-databases
// @Summary List Databases in RDBMS
// @Description List databases inside an RDBMS instance using the CSP-native API.
// @Tags [RDBMS Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the RDBMS instance"
// @Param ConnectionName query string true "The name of the Connection"
// @Param X-Master-User-Name header string false "The master user name (required by SQL-based drivers such as AWS and IBM)"
// @Param X-Master-User-Password header string false "The master user password (required by SQL-based drivers such as AWS and IBM)"
// @Success 200 {object} restruntime.RDBMSDatabaseListResponse "List of databases"
// @Failure 400 {object} SimpleMsg "Bad Request"
// @Failure 501 {object} SimpleMsg "Not Supported by driver"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /rdbms/{Name}/databases [get]
func ListRDBMSDatabases(c echo.Context) error {
	cblog.Info("call ListRDBMSDatabases()")

	var req RDBMSDatabaseRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}
	if req.MasterUserName == "" {
		req.MasterUserName = c.Request().Header.Get("X-Master-User-Name")
	}
	if req.MasterUserPassword == "" {
		req.MasterUserPassword = c.Request().Header.Get("X-Master-User-Password")
	}
	if req.ConnectionName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "ConnectionName is required")
	}

	databases, err := cmrt.ListRDBMSDatabases(req.ConnectionName, c.Param("Name"), req.MasterUserName, req.MasterUserPassword)
	if err != nil {
		if err == cmrt.ErrRDBMSDatabaseMgrNotSupported {
			return echo.NewHTTPError(http.StatusNotImplemented, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	if databases == nil {
		databases = []string{}
	}
	return c.JSON(http.StatusOK, &RDBMSDatabaseListResponse{Databases: databases})
}

// deleteRDBMSDatabase godoc
// @ID delete-rdbms-database
// @Summary Delete Database in RDBMS
// @Description Drop a database inside an RDBMS instance using the CSP-native API.
// @Tags [RDBMS Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the RDBMS instance"
// @Param DBName path string true "The name of the database to drop"
// @Param RDBMSDatabaseRequest body restruntime.RDBMSDatabaseRequest true "ConnectionName"
// @Success 200 {object} SimpleMsg "Deleted"
// @Failure 400 {object} SimpleMsg "Bad Request"
// @Failure 501 {object} SimpleMsg "Not Supported by driver"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /rdbms/{Name}/databases/{DBName} [delete]
func DeleteRDBMSDatabase(c echo.Context) error {
	cblog.Info("call DeleteRDBMSDatabase()")

	var req RDBMSDatabaseRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	if req.ConnectionName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "ConnectionName is required")
	}

	err := cmrt.DeleteRDBMSDatabase(req.ConnectionName, c.Param("Name"), c.Param("DBName"), req.MasterUserName, req.MasterUserPassword)
	if err != nil {
		if err == cmrt.ErrRDBMSDatabaseMgrNotSupported {
			return echo.NewHTTPError(http.StatusNotImplemented, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &SimpleMsg{Message: "deleted"})
}

//================ RDBMS Secure Transport Status

// getRDBMSSecureTransport godoc
// @ID get-rdbms-secure-transport
// @Summary Get RDBMS Secure Transport Status
// @Description Report whether an RDBMS instance enforces encrypted (TLS/SSL) client connections, whether TLS is actually available, and its CA certificate. <br> Determined uniformly across every CSP via standard SQL and protocol handshakes against the engine itself, not each CSP's own (inconsistently available) management API: <br> MySQL/MariaDB: `SHOW VARIABLES LIKE 'require_secure_transport'`. <br> PostgreSQL: `pg_hba_file_rules`. <br> TLS availability and the server's certificate are captured live from the connection/handshake itself — see the response fields below.
// @Tags [RDBMS Management]
// @Accept  json
// @Produce  json
// @Param Name path string true "The name of the RDBMS instance"
// @Param ConnectionName query string true "The name of the Connection"
// @Param X-Master-User-Name header string true "The master user name, used to connect and run the SQL check"
// @Param X-Master-User-Password header string true "The master user password, used to connect and run the SQL check"
// @Success 200 {object} cmrt.RDBMSSecureTransportInfo "Secure transport status"
// @Failure 400 {object} SimpleMsg "Bad Request"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /rdbms/{Name}/secure-transport [get]
func GetRDBMSSecureTransport(c echo.Context) error {
	cblog.Info("call GetRDBMSSecureTransport()")

	var req RDBMSDatabaseRequest
	if err := c.Bind(&req); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, err.Error())
	}
	// To support for Get-Query Param Type API
	if req.ConnectionName == "" {
		req.ConnectionName = c.QueryParam("ConnectionName")
	}
	if req.MasterUserName == "" {
		req.MasterUserName = c.Request().Header.Get("X-Master-User-Name")
	}
	if req.MasterUserPassword == "" {
		req.MasterUserPassword = c.Request().Header.Get("X-Master-User-Password")
	}
	if req.ConnectionName == "" || req.MasterUserName == "" || req.MasterUserPassword == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "ConnectionName, MasterUserName and MasterUserPassword are required")
	}

	result, err := cmrt.GetRDBMSSecureTransportStatus(req.ConnectionName, c.Param("Name"), req.MasterUserName, req.MasterUserPassword)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}
