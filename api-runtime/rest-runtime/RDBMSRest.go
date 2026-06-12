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

	"github.com/labstack/echo/v4"

	"strconv"
)

//================ RDBMS Handler

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
// @Success 200 {object} cres.RDBMSInfo "Details of the registered RDBMS"
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

	return c.JSON(http.StatusOK, result)
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
		DBInstanceSpec  string `json:"DBInstanceSpec" validate:"required" example:"db.t3.medium"`
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

		TagList []cres.KeyValue `json:"TagList,omitempty" validate:"omitempty"`
	} `json:"ReqInfo" validate:"required"`
}

// createRDBMS godoc
// @ID create-rdbms
// @Summary Create RDBMS
// @Description Create a new Relational Database (RDBMS) with the specified configuration.
// @Tags [RDBMS Management]
// @Accept  json
// @Produce  json
// @Param RDBMSCreateRequest body restruntime.RDBMSCreateRequest true "Request body for creating an RDBMS"
// @Success 200 {object} cres.RDBMSInfo "Details of the created RDBMS"
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
		DBInstanceSpec:  req.ReqInfo.DBInstanceSpec,
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

		TagList: req.ReqInfo.TagList,
	}

	// Call common-runtime API
	result, err := cmrt.CreateRDBMS(req.ConnectionName, RDBMS, reqInfo, req.IDTransformMode)
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, result)
}

// RDBMSListResponse represents the response body for listing RDBMS instances.
type RDBMSListResponse struct {
	Result []*cres.RDBMSInfo `json:"rdbms" validate:"required" description:"A list of RDBMS information"`
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

	jsonResult := RDBMSListResponse{
		Result: result,
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

// listAllRDBMSInfo godoc
// @ID list-all-rdbms-info
// @Summary List All RDBMS Info
// @Description Retrieve a list of all RDBMS information associated with all connections.
// @Tags [RDBMS Management]
// @Accept  json
// @Produce  json
// @Success 200 {object} AllResourceListResponse "List of all RDBMS information across all connections"
// @Failure 500 {object} SimpleMsg "Internal Server Error"
// @Router /allrdbmsinfo [get]
func ListAllRDBMSInfo(c echo.Context) error { return listAllResourceInfo(c, cres.RDBMS) }

// getRDBMS godoc
// @ID get-rdbms
// @Summary Get RDBMS
// @Description Retrieve details of a specific RDBMS instance.
// @Tags [RDBMS Management]
// @Accept  json
// @Produce  json
// @Param ConnectionName query string true "The name of the Connection to get an RDBMS for"
// @Param Name path string true "The name of the RDBMS to retrieve"
// @Success 200 {object} cres.RDBMSInfo "Details of the RDBMS"
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

	return c.JSON(http.StatusOK, result)
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
	ConnectionName     string `json:"ConnectionName" validate:"required" example:"ncp-korea1-config"`
	DatabaseName       string `json:"DatabaseName,omitempty" example:"mydb"`           // required only for create/delete
	MasterUserPassword string `json:"MasterUserPassword,omitempty" example:"P@ssw0rd"` // required when driver uses SQL (e.g. AWS, IBM)
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

	err := cmrt.CreateRDBMSDatabase(req.ConnectionName, c.Param("Name"), req.DatabaseName, req.MasterUserPassword)
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
// @Param RDBMSDatabaseRequest body restruntime.RDBMSDatabaseRequest true "ConnectionName"
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
	if req.ConnectionName == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "ConnectionName is required")
	}

	databases, err := cmrt.ListRDBMSDatabases(req.ConnectionName, c.Param("Name"), req.MasterUserPassword)
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

	err := cmrt.DeleteRDBMSDatabase(req.ConnectionName, c.Param("Name"), c.Param("DBName"), req.MasterUserPassword)
	if err != nil {
		if err == cmrt.ErrRDBMSDatabaseMgrNotSupported {
			return echo.NewHTTPError(http.StatusNotImplemented, err.Error())
		}
		return echo.NewHTTPError(http.StatusInternalServerError, err.Error())
	}

	return c.JSON(http.StatusOK, &SimpleMsg{Message: "deleted"})
}
