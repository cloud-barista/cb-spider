// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, April 2026.

package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NhnCloudRDBMSHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	DBClient       *nhnsdk.ServiceClient
}

type nhnRDSResponseHeader struct {
	ResultCode    int    `json:"resultCode"`
	ResultMessage string `json:"resultMessage"`
	IsSuccessful  bool   `json:"isSuccessful"`
}

type nhnRDSDBVersionsResponse struct {
	Header     nhnRDSResponseHeader `json:"header"`
	DBVersions []struct {
		DBVersion string `json:"dbVersion"`
	} `json:"dbVersions"`
}

type nhnRDSStorageTypesResponse struct {
	Header       nhnRDSResponseHeader `json:"header"`
	StorageTypes []json.RawMessage    `json:"storageTypes"`
}

// ─── NHN RDS for MySQL v3.0 native-API types ─────────────────────────────────

type nhnRDSEndpoint struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
}

type nhnRDSDBUser struct {
	DBUserId     string `json:"dbUserId"`
	DBUserName   string `json:"dbUserName"`
	DBUserStatus string `json:"dbUserStatus"`
}

type nhnRDSDBUserListResponse struct {
	Header  nhnRDSResponseHeader `json:"header"`
	DBUsers []nhnRDSDBUser       `json:"dbUsers"`
}

type nhnRDSNetworkEndpoint struct {
	Domain       string `json:"domain"`
	IPAddress    string `json:"ipAddress"`
	EndPointType string `json:"endPointType"`
}

type nhnRDSNetworkSubnet struct {
	SubnetId   string `json:"subnetId"`
	SubnetName string `json:"subnetName"`
	SubnetCidr string `json:"subnetCidr"`
}

type nhnRDSNetworkInfoResponse struct {
	Header    nhnRDSResponseHeader    `json:"header"`
	Subnet    nhnRDSNetworkSubnet     `json:"subnet"`
	EndPoints []nhnRDSNetworkEndpoint `json:"endPoints"`
}

type nhnRDSStorageInfoResponse struct {
	Header      nhnRDSResponseHeader `json:"header"`
	StorageType string               `json:"storageType"`
	StorageSize int                  `json:"storageSize"`
}

// nhnRDSEnrichmentData holds data from supplementary NHN RDS APIs.
type nhnRDSEnrichmentData struct {
	MasterUserName  string
	PublicEndpoint  string
	StorageType     string
	StorageSize     int
	SubnetId        string
	SubnetName      string
	SubnetCidr      string
	UsePublicAccess bool
	DBFlavorName    string
	BackupConfig    nhnRDSBackupConfig
}

type nhnRDSNetworkInfo struct {
	SubnetId         string           `json:"subnetId"`
	AvailabilityZone string           `json:"availabilityZone,omitempty"`
	UsePublicAccess  bool             `json:"usePublicAccess"`
	Endpoints        []nhnRDSEndpoint `json:"endpoints,omitempty"`
}

type nhnRDSStorageInfo struct {
	StorageType string `json:"storageType"`
	StorageSize int    `json:"storageSize"`
}

type nhnRDSBackupSchedule struct {
	BackupWndBgnTime  string `json:"backupWndBgnTime"`  // HH:mm:ss format — backup window begin time
	BackupWndDuration string `json:"backupWndDuration"` // "ONE_HOUR", "TWO_HOUR", etc.
}

type nhnRDSBackupConfig struct {
	BackupPeriod     int                    `json:"backupPeriod"`
	BackupRetryCount int                    `json:"backupRetryCount"`
	BackupSchedules  []nhnRDSBackupSchedule `json:"backupSchedules,omitempty"`
}

type nhnRDSDBInstance struct {
	DBInstanceId          string             `json:"dbInstanceId"`
	DBInstanceName        string             `json:"dbInstanceName"`
	DBInstanceStatus      string             `json:"dbInstanceStatus"`
	DBVersion             string             `json:"dbVersion"`
	DBPort                int                `json:"dbPort"`
	DBFlavorId            string             `json:"dbFlavorId"`
	Description           string             `json:"description"`
	Storage               nhnRDSStorageInfo  `json:"storage"`
	Network               nhnRDSNetworkInfo  `json:"network"`
	UseHighAvailability   bool               `json:"useHighAvailability"`
	UseDeletionProtection bool               `json:"useDeletionProtection"`
	Backup                nhnRDSBackupConfig `json:"backup"`
	DBSecurityGroupIds    []string           `json:"dbSecurityGroupIds"`
	CreatedYmdt           string             `json:"createdYmdt"`
}

type nhnRDSListInstancesResponse struct {
	Header      nhnRDSResponseHeader `json:"header"`
	DBInstances []nhnRDSDBInstance   `json:"dbInstances"`
}

// nhnRDSGetInstanceResponse is the flat response from GET /v3.0/db-instances/{id}.
// NHN returns instance fields at the top level alongside the header (not nested under a "dbInstance" key).
type nhnRDSGetInstanceResponse struct {
	nhnRDSDBInstance
	Header nhnRDSResponseHeader `json:"header"`
}

type nhnRDSDBSecurityGroupPort struct {
	PortType string `json:"portType"` // DB_PORT | PORT | PORT_RANGE
	MinPort  int    `json:"minPort,omitempty"`
	MaxPort  int    `json:"maxPort,omitempty"`
}

type nhnRDSDBSecurityGroupRule struct {
	Direction   string                    `json:"direction"` // INGRESS | EGRESS
	EtherType   string                    `json:"etherType"` // IPV4 | IPV6
	Cidr        string                    `json:"cidr"`
	Port        nhnRDSDBSecurityGroupPort `json:"port"`
	Description string                    `json:"description,omitempty"`
}

type nhnRDSCreateDBSecurityGroupRequest struct {
	DBSecurityGroupName string                      `json:"dbSecurityGroupName"`
	Description         string                      `json:"description,omitempty"`
	Rules               []nhnRDSDBSecurityGroupRule `json:"rules"`
}

type nhnRDSCreateDBSecurityGroupResponse struct {
	Header            nhnRDSResponseHeader `json:"header"`
	DBSecurityGroupId string               `json:"dbSecurityGroupId"`
}

type nhnRDSModifyInstanceRequest struct {
	DBSecurityGroupIds []string `json:"dbSecurityGroupIds"`
}

type nhnRDSDBSecurityGroupDetail struct {
	DBSecurityGroupId   string `json:"dbSecurityGroupId"`
	DBSecurityGroupName string `json:"dbSecurityGroupName"`
	Description         string `json:"description"`
}

type nhnRDSDBSecurityGroupDetailResponse struct {
	Header nhnRDSResponseHeader `json:"header"`
	nhnRDSDBSecurityGroupDetail
}

type nhnRDSCreateInstanceRequest struct {
	DBInstanceName        string             `json:"dbInstanceName"`
	DBFlavorId            string             `json:"dbFlavorId"`
	DBVersion             string             `json:"dbVersion"`
	DBPort                int                `json:"dbPort"`
	DBUserName            string             `json:"dbUserName"`
	DBPassword            string             `json:"dbPassword"`
	ParameterGroupId      string             `json:"parameterGroupId"`
	Network               nhnRDSNetworkInfo  `json:"network"`
	Storage               nhnRDSStorageInfo  `json:"storage"`
	Backup                nhnRDSBackupConfig `json:"backup"`
	UseHighAvailability   bool               `json:"useHighAvailability"`
	PingInterval          int                `json:"pingInterval,omitempty"`
	UseDefaultUserGroup   bool               `json:"useDefaultUserGroup"`
	UserGroupIds          []string           `json:"userGroupIds"`
	DBSecurityGroupIds    []string           `json:"dbSecurityGroupIds"`
	UseDeletionProtection bool               `json:"useDeletionProtection"`
}

type nhnRDSDBFlavorInfo struct {
	DBFlavorId   string `json:"dbFlavorId"`
	DBFlavorName string `json:"dbFlavorName"`
}

type nhnRDSFlavorListResponse struct {
	Header    nhnRDSResponseHeader `json:"header"`
	DBFlavors []nhnRDSDBFlavorInfo `json:"dbFlavors"`
}

type nhnRDSParameterGroup struct {
	ParameterGroupId   string `json:"parameterGroupId"`
	ParameterGroupName string `json:"parameterGroupName"`
	DBVersion          string `json:"dbVersion"`
}

type nhnRDSParameterGroupListResponse struct {
	Header          nhnRDSResponseHeader   `json:"header"`
	ParameterGroups []nhnRDSParameterGroup `json:"parameterGroups"`
}

type nhnRDSResourceRelation struct {
	ResourceType string `json:"resourceType"`
	ResourceId   string `json:"resourceId"`
}

type nhnRDSJobResponse struct {
	Header            nhnRDSResponseHeader     `json:"header"`
	JobId             string                   `json:"jobId"`
	JobStatus         string                   `json:"jobStatus"`
	ResourceRelations []nhnRDSResourceRelation `json:"resourceRelations"`
}

// GetMetaInfo returns metadata queried dynamically from NHN Cloud RDS for MySQL/MariaDB API v3.0.
func (handler *NhnCloudRDBMSHandler) GetMetaInfo(dbEngine string) (irs.RDBMSMetaInfo, error) {
	cblogger.Info("NHN Cloud Driver: called GetMetaInfo()")
	callLogInfo := getCallLogScheme(handler.RegionInfo.Region, call.RDBMS, "GetMetaInfo", "GetMetaInfo()")
	start := call.Start()

	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		LoggingError(callLogInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}
	if requestedEngine != "mysql" && requestedEngine != "mariadb" {
		err := fmt.Errorf("NHN Cloud RDS supports mysql and mariadb engines, requested: %s", requestedEngine)
		LoggingError(callLogInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}

	if err := handler.checkRDSCredentials(); err != nil {
		LoggingError(callLogInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	endpointFn := handler.rdsEndpoint
	if requestedEngine == "mariadb" {
		endpointFn = handler.rdsMariaDBEndpoint
	}

	dbVersions, err := handler.listRDSDBVersionsWithEndpoint(ctx, endpointFn)
	if err != nil {
		newErr := fmt.Errorf("failed to list DB versions from NHN Cloud RDS API: %w", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.RDBMSMetaInfo{}, newErr
	}
	dbFlavors, err := handler.listRDSDBFlavorsWithEndpoint(ctx, endpointFn)
	if err != nil {
		newErr := fmt.Errorf("failed to list DB flavors from NHN Cloud RDS API: %w", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.RDBMSMetaInfo{}, newErr
	}
	storageTypes, err := handler.listRDSStorageTypesWithEndpoint(ctx, endpointFn)
	if err != nil {
		newErr := fmt.Errorf("failed to list storage types from NHN Cloud RDS API: %w", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.RDBMSMetaInfo{}, newErr
	}

	if len(dbVersions) == 0 {
		err := fmt.Errorf("NHN Cloud RDS API returned no DB versions for engine: %s", requestedEngine)
		LoggingError(callLogInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}
	if len(dbFlavors) == 0 {
		err := fmt.Errorf("NHN Cloud RDS API returned no DB flavors for engine: %s", requestedEngine)
		LoggingError(callLogInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}
	if len(storageTypes) == 0 {
		err := fmt.Errorf("NHN Cloud RDS API returned no storage types for engine: %s", requestedEngine)
		LoggingError(callLogInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}

	supportedEngines := map[string][]string{
		requestedEngine: dbVersions,
	}
	instanceSpecOptions := map[string][]string{
		requestedEngine: dbFlavors,
	}
	storageTypeOptions := map[string][]string{
		requestedEngine: storageTypes,
	}

	storageSizeRange := irs.StorageSizeRange{Min: 20, Max: 2048}

	metaInfo, err := irs.BuildRDBMSMetaInfo(requestedEngine, supportedEngines, instanceSpecOptions, storageTypeOptions, storageSizeRange, true, true, true, true, false, "1-730", true, false, true, true, false)
	if err != nil {
		LoggingError(callLogInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}
	metaInfo.MarkStatic("StorageSizeRange", "NHN Cloud RDS API does not expose a storage size range; fixed at 20-2048GB.")

	LoggingInfo(callLogInfo, start)
	return metaInfo, nil
}

func (handler *NhnCloudRDBMSHandler) listRDSDBVersions(ctx context.Context) ([]string, error) {
	return handler.listRDSDBVersionsWithEndpoint(ctx, handler.rdsEndpoint)
}

func (handler *NhnCloudRDBMSHandler) listRDSDBVersionsWithEndpoint(ctx context.Context, endpointFn func() (string, error)) ([]string, error) {
	var result nhnRDSDBVersionsResponse
	if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-versions", &result); err != nil {
		return nil, err
	}
	if err := checkRDSResponseHeader(result.Header); err != nil {
		return nil, err
	}

	versions := make([]string, 0, len(result.DBVersions))
	for _, version := range result.DBVersions {
		if version.DBVersion != "" {
			versions = append(versions, version.DBVersion)
		}
	}
	return versions, nil
}

func (handler *NhnCloudRDBMSHandler) listRDSDBFlavors(ctx context.Context) ([]string, error) {
	return handler.listRDSDBFlavorsWithEndpoint(ctx, handler.rdsEndpoint)
}

func (handler *NhnCloudRDBMSHandler) listRDSDBFlavorsWithEndpoint(ctx context.Context, endpointFn func() (string, error)) ([]string, error) {
	var result nhnRDSFlavorListResponse
	if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-flavors", &result); err != nil {
		return nil, err
	}
	if err := checkRDSResponseHeader(result.Header); err != nil {
		return nil, err
	}

	flavors := make([]string, 0, len(result.DBFlavors))
	for _, flavor := range result.DBFlavors {
		if flavor.DBFlavorName != "" {
			flavors = append(flavors, flavor.DBFlavorName)
		}
	}
	return flavors, nil
}

func (handler *NhnCloudRDBMSHandler) listRDSStorageTypes(ctx context.Context) ([]string, error) {
	return handler.listRDSStorageTypesWithEndpoint(ctx, handler.rdsEndpoint)
}

func (handler *NhnCloudRDBMSHandler) listRDSStorageTypesWithEndpoint(ctx context.Context, endpointFn func() (string, error)) ([]string, error) {
	var result nhnRDSStorageTypesResponse
	if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/storage-types", &result); err != nil {
		return nil, err
	}
	if err := checkRDSResponseHeader(result.Header); err != nil {
		return nil, err
	}

	return extractStringValues(result.StorageTypes, "storageType"), nil
}

// rdsEffectiveAppKey returns the App Key to use for the given NHN RDS endpoint URL.
// RDS for MySQL and RDS for MariaDB are separate NHN Cloud services with separate App Keys.
// If the caller has configured RDSMariaDBAppKey and the endpoint targets the MariaDB service,
// that key is used; otherwise RDSMySQLAppKey is returned.
func (handler *NhnCloudRDBMSHandler) rdsEffectiveAppKey(endpoint string) string {
	if strings.Contains(endpoint, "rds-mariadb") {
		if handler.CredentialInfo.RDSMariaDBAppKey != "" {
			return handler.CredentialInfo.RDSMariaDBAppKey
		}
		cblogger.Warnf("[NHN RDS] WARNING: calling MariaDB endpoint but RDSMariaDBAppKey (mariadbAppKey) is not set; " +
			"falling back to RDSMySQLAppKey — this will likely cause Unauthorized or 500 from NHN API. " +
			"Register 'mariadbAppKey' in your NHN credential (NHN Console → RDS for MariaDB → URL & AppKey).")
	}
	return handler.CredentialInfo.RDSMySQLAppKey
}

func (handler *NhnCloudRDBMSHandler) getRDS(ctx context.Context, path string, v interface{}) error {
	return handler.getRDSWithEndpoint(ctx, handler.rdsEndpoint, path, v)
}

// rdsEndpointForEngine returns the appropriate RDS endpoint function for the given DB engine name.
// MariaDB engine (case-insensitive "mariadb") uses rdsMariaDBEndpoint; all others use rdsEndpoint.
func (handler *NhnCloudRDBMSHandler) rdsEndpointForEngine(dbEngine string) func() (string, error) {
	if strings.EqualFold(dbEngine, "mariadb") {
		return handler.rdsMariaDBEndpoint
	}
	return handler.rdsEndpoint
}

// getRDSWithEndpoint sends a GET request to the given NHN RDS endpoint (MySQL or MariaDB).
func (handler *NhnCloudRDBMSHandler) getRDSWithEndpoint(ctx context.Context, endpointFn func() (string, error), path string, v interface{}) error {
	endpoint, err := endpointFn()
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create NHN Cloud RDS request: %w", err)
	}
	req.Header.Set("X-TC-APP-KEY", handler.rdsEffectiveAppKey(endpoint))
	req.Header.Set("X-TC-AUTHENTICATION-ID", handler.CredentialInfo.RDSUserAccessKey)
	req.Header.Set("X-TC-AUTHENTICATION-SECRET", handler.CredentialInfo.RDSSecretAccessKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call NHN Cloud RDS API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("NHN Cloud RDS API returned HTTP status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read NHN Cloud RDS API response: %w", err)
	}

	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("failed to decode NHN Cloud RDS API response: %w", err)
	}
	return nil
}

func (handler *NhnCloudRDBMSHandler) rdsEndpoint() (string, error) {
	region := strings.ToLower(handler.RegionInfo.Region)
	switch region {
	case "kr1", "kr2", "jp1":
		return fmt.Sprintf("https://%s-rds-mysql.api.nhncloudservice.com", region), nil
	default:
		return "", fmt.Errorf("unsupported NHN Cloud RDS for MySQL region: %s", handler.RegionInfo.Region)
	}
}

// rdsMariaDBEndpoint returns the NHN Cloud RDS for MariaDB API base URL for the current region.
func (handler *NhnCloudRDBMSHandler) rdsMariaDBEndpoint() (string, error) {
	region := strings.ToLower(handler.RegionInfo.Region)
	switch region {
	case "kr1", "kr2", "jp1":
		return fmt.Sprintf("https://%s-rds-mariadb.api.nhncloudservice.com", region), nil
	default:
		return "", fmt.Errorf("unsupported NHN Cloud RDS for MariaDB region: %s", handler.RegionInfo.Region)
	}
}

func (handler *NhnCloudRDBMSHandler) checkRDSCredentials() error {
	isUnset := func(v string) bool { return v == "" || v == "Not set" }
	var missing []string
	if isUnset(handler.CredentialInfo.RDSMySQLAppKey) {
		missing = append(missing, "'mysqlAppKey'")
	}
	if isUnset(handler.CredentialInfo.RDSUserAccessKey) {
		missing = append(missing, "'User Access Key'")
	}
	if isUnset(handler.CredentialInfo.RDSSecretAccessKey) {
		missing = append(missing, "'Secret Access Key'")
	}
	if len(missing) > 0 {
		return fmt.Errorf(
			"NHN Cloud RDBMS requires 3 credential keys that are not yet registered: %s.\n"+
				"How to obtain and register them:\n"+
				"  1. mysqlAppKey  : NHN Cloud Console → Database → RDS for MySQL → URL & AppKey\n"+
				"  2. User Access Key  : NHN Cloud Console → My Page → API Security Settings → User Access Key ID\n"+
				"  3. Secret Access Key: same page as above → Secret Access Key\n"+
				"Add the missing key(s) to your CB-Spider credential and try again.",
			strings.Join(missing, ", "),
		)
	}
	return nil
}

func checkRDSResponseHeader(header nhnRDSResponseHeader) error {
	if !header.IsSuccessful || header.ResultCode != 0 {
		return fmt.Errorf("NHN Cloud RDS API error: resultCode=%d resultMessage=%s", header.ResultCode, header.ResultMessage)
	}
	return nil
}

func extractStringValues(rawValues []json.RawMessage, fieldName string) []string {
	values := make([]string, 0, len(rawValues))
	for _, rawValue := range rawValues {
		var value string
		if err := json.Unmarshal(rawValue, &value); err == nil && value != "" {
			values = append(values, value)
			continue
		}

		var objectValue map[string]string
		if err := json.Unmarshal(rawValue, &objectValue); err == nil && objectValue[fieldName] != "" {
			values = append(values, objectValue[fieldName])
		}
	}
	return values
}

// ListIID returns a list of RDBMS instance IIDs using the NHN native RDS API.
func (handler *NhnCloudRDBMSHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("NHN Cloud Driver: called ListIID()")
	callLogInfo := getCallLogScheme(handler.RegionInfo.Region, call.RDBMS, "ListIID", "GET /v3.0/db-instances")
	start := call.Start()

	if err := handler.checkRDSCredentials(); err != nil {
		LoggingError(callLogInfo, err)
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	iidList := make([]*irs.IID, 0)

	// MySQL instances
	var mysqlResult nhnRDSListInstancesResponse
	if err := handler.getRDS(ctx, "/v3.0/db-instances", &mysqlResult); err != nil {
		LoggingError(callLogInfo, err)
		return nil, err
	}
	if err := checkRDSResponseHeader(mysqlResult.Header); err != nil {
		LoggingError(callLogInfo, err)
		return nil, err
	}
	for _, inst := range mysqlResult.DBInstances {
		iidList = append(iidList, &irs.IID{
			NameId:   inst.DBInstanceName,
			SystemId: inst.DBInstanceId,
		})
	}

	// MariaDB instances (best-effort; only if MariaDB endpoint is available)
	var mariaResult nhnRDSListInstancesResponse
	if err := handler.getRDSWithEndpoint(ctx, handler.rdsMariaDBEndpoint, "/v3.0/db-instances", &mariaResult); err != nil {
		cblogger.Warnf("[NHN RDS] ListIID: MariaDB endpoint unavailable (non-fatal): %v", err)
	} else if err := checkRDSResponseHeader(mariaResult.Header); err == nil {
		for _, inst := range mariaResult.DBInstances {
			iidList = append(iidList, &irs.IID{
				NameId:   inst.DBInstanceName,
				SystemId: inst.DBInstanceId,
			})
		}
	}

	LoggingInfo(callLogInfo, start)
	return iidList, nil
}

// CreateRDBMS creates a new DB instance via the NHN Cloud native RDS for MySQL API v3.0.
func (handler *NhnCloudRDBMSHandler) CreateRDBMS(rdbmsReqInfo irs.RDBMSInfo) (irs.RDBMSInfo, error) {
	cblogger.Info("NHN Cloud Driver: called CreateRDBMS()")
	callLogInfo := getCallLogScheme(handler.RegionInfo.Region, call.RDBMS, rdbmsReqInfo.IId.NameId, "POST /v3.0/db-instances")
	start := call.Start()

	if err := handler.checkRDSCredentials(); err != nil {
		LoggingError(callLogInfo, err)
		return irs.RDBMSInfo{}, err
	}

	switch {
	case rdbmsReqInfo.IId.NameId == "":
		return irs.RDBMSInfo{}, errors.New("RDBMS instance name is required")
	case rdbmsReqInfo.DBEngineVersion == "":
		return irs.RDBMSInfo{}, errors.New("DBEngineVersion is required (use the value from MetaInfo, e.g. MYSQL_V8032)")
	case rdbmsReqInfo.DBInstanceSpec == "":
		return irs.RDBMSInfo{}, errors.New("DBInstanceSpec (NHN DB flavor UUID) is required")
	case rdbmsReqInfo.StorageSize == "":
		return irs.RDBMSInfo{}, errors.New("StorageSize is required")
	case rdbmsReqInfo.MasterUserName == "":
		return irs.RDBMSInfo{}, errors.New("MasterUserName is required")
	case rdbmsReqInfo.MasterUserPassword == "":
		return irs.RDBMSInfo{}, errors.New("MasterUserPassword is required")
	}

	storageSize, err := strconv.Atoi(rdbmsReqInfo.StorageSize)
	if err != nil {
		return irs.RDBMSInfo{}, fmt.Errorf("invalid StorageSize '%s': %w", rdbmsReqInfo.StorageSize, err)
	}

	// Subnet UUID required for NHN RDS network placement
	subnetId := ""
	if len(rdbmsReqInfo.SubnetIIDs) > 0 {
		subnetId = rdbmsReqInfo.SubnetIIDs[0].SystemId
		if subnetId == "" {
			subnetId = rdbmsReqInfo.SubnetIIDs[0].NameId
		}
	}
	if subnetId == "" {
		return irs.RDBMSInfo{}, errors.New("SubnetNames[0] is required for NHN Cloud RDS (provide the subnet UUID)")
	}

	storageType := rdbmsReqInfo.StorageType
	if storageType == "" {
		storageType = "General SSD"
	}

	dbPort := 3306 // NHN MySQL default port

	// NHN requires backup config - use 7 days as default if not specified
	backupPeriod := rdbmsReqInfo.BackupRetentionDays
	if backupPeriod <= 0 {
		backupPeriod = 7 // NHN default
	}
	// BackupTime (backupHour/backupMinute) is not configurable at creation via Spider (NHN uses fixed time)
	backupHour, backupMinute := 3, 0

	// Debug: Log backup period being set
	cblogger.Infof("[NHN RDBMS] Creating DB instance with BackupPeriod: %d days", backupPeriod)

	// NHN RDS provisioning is async — allow up to 15 minutes
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Choose endpoint based on DB engine version: MARIADB_V* → MariaDB endpoint
	endpointFn := handler.rdsEndpoint
	if strings.HasPrefix(strings.ToUpper(rdbmsReqInfo.DBEngineVersion), "MARIADB_") {
		endpointFn = handler.rdsMariaDBEndpoint
	}

	// Resolve DB flavor UUID from name or pass-through if already a UUID
	flavorId, err := handler.resolveRDSFlavorIdWithEndpoint(ctx, endpointFn, rdbmsReqInfo.DBInstanceSpec)
	if err != nil {
		newErr := fmt.Errorf("failed to resolve NHN Cloud RDS flavor '%s': %w", rdbmsReqInfo.DBInstanceSpec, err)
		LoggingError(callLogInfo, newErr)
		return irs.RDBMSInfo{}, newErr
	}

	// Fetch a parameter group for the requested DB version
	paramGroupId, err := handler.findDefaultParameterGroupIdWithEndpoint(ctx, endpointFn, rdbmsReqInfo.DBEngineVersion)
	if err != nil {
		newErr := fmt.Errorf("failed to find NHN Cloud RDS parameter group: %w", err)
		LoggingError(callLogInfo, newErr)
		return irs.RDBMSInfo{}, newErr
	}

	// Build DB Security Group IDs.
	// If SecurityGroupIIDs are provided, use their SystemIds (NHN RDS DB SG UUIDs).
	// Otherwise auto-create a permissive SG that allows inbound on DB_PORT from 0.0.0.0/0.
	var dbSGIds []string
	if len(rdbmsReqInfo.SecurityGroupIIDs) > 0 {
		for _, sg := range rdbmsReqInfo.SecurityGroupIIDs {
			if sg.SystemId != "" {
				dbSGIds = append(dbSGIds, sg.SystemId)
			}
		}
	}
	if len(dbSGIds) == 0 {
		// Resolve subnet CIDR for private-access restriction (best-effort)
		subnetCidr := ""
		if !rdbmsReqInfo.PublicAccess && subnetId != "" {
			subnetCidr = handler.fetchSubnetCidr(ctx, subnetId)
		}
		autoSGId, sgErr := handler.createDefaultDBSecurityGroup(ctx, endpointFn, rdbmsReqInfo.IId.NameId, rdbmsReqInfo.PublicAccess, subnetCidr)
		if sgErr != nil {
			LoggingError(callLogInfo, sgErr)
			return irs.RDBMSInfo{}, sgErr
		}
		dbSGIds = []string{autoSGId}
	}

	reqBody := nhnRDSCreateInstanceRequest{
		DBInstanceName:   rdbmsReqInfo.IId.NameId,
		DBFlavorId:       flavorId,
		DBVersion:        rdbmsReqInfo.DBEngineVersion,
		DBPort:           dbPort,
		DBUserName:       rdbmsReqInfo.MasterUserName,
		DBPassword:       rdbmsReqInfo.MasterUserPassword,
		ParameterGroupId: paramGroupId,
		Network: nhnRDSNetworkInfo{
			SubnetId:         subnetId,
			AvailabilityZone: handler.RegionInfo.Zone,
			UsePublicAccess:  rdbmsReqInfo.PublicAccess,
		},
		Storage: nhnRDSStorageInfo{
			StorageType: storageType,
			StorageSize: storageSize,
		},
		Backup: nhnRDSBackupConfig{
			BackupPeriod:     backupPeriod,
			BackupRetryCount: 0,
			BackupSchedules: []nhnRDSBackupSchedule{
				{
					BackupWndBgnTime:  fmt.Sprintf("%02d:%02d", backupHour, backupMinute),
					BackupWndDuration: "ONE_HOUR",
				},
			},
		},
		UseHighAvailability:   rdbmsReqInfo.HighAvailability,
		UseDefaultUserGroup:   true,
		UserGroupIds:          []string{},
		DBSecurityGroupIds:    dbSGIds,
		UseDeletionProtection: rdbmsReqInfo.DeletionProtection,
	}
	if rdbmsReqInfo.HighAvailability {
		reqBody.PingInterval = 3
	}

	var createResp nhnRDSJobResponse
	if err := handler.postRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-instances", reqBody, &createResp); err != nil {
		newErr := fmt.Errorf("failed to submit NHN Cloud RDS create request: %w", err)
		LoggingError(callLogInfo, newErr)
		return irs.RDBMSInfo{}, newErr
	}
	if err := checkRDSResponseHeader(createResp.Header); err != nil {
		LoggingError(callLogInfo, err)
		return irs.RDBMSInfo{}, err
	}

	// Poll until the async job completes and we have the instance UUID
	dbInstanceId, err := handler.pollRDSJobWithEndpoint(ctx, endpointFn, createResp.JobId)
	if err != nil {
		newErr := handler.rollbackCreatedRDBMS(irs.IID{NameId: rdbmsReqInfo.IId.NameId},
			fmt.Errorf("NHN Cloud RDS instance creation job failed: %w", err))
		LoggingError(callLogInfo, newErr)
		return irs.RDBMSInfo{}, newErr
	}

	// Retry the post-create fetches so a transient API error does not destroy a
	// successfully created instance.
	const maxGetAttempts = 3
	var getResp nhnRDSGetInstanceResponse
	for attempt := 1; ; attempt++ {
		getResp = nhnRDSGetInstanceResponse{}
		fetchErr := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-instances/"+dbInstanceId, &getResp)
		if fetchErr == nil {
			fetchErr = checkRDSResponseHeader(getResp.Header)
		}
		if fetchErr == nil {
			break
		}
		if attempt >= maxGetAttempts {
			newErr := handler.rollbackCreatedRDBMS(irs.IID{NameId: rdbmsReqInfo.IId.NameId, SystemId: dbInstanceId},
				fmt.Errorf("failed to fetch newly created NHN Cloud RDS instance after %d attempts: %w", attempt, fetchErr))
			LoggingError(callLogInfo, newErr)
			return irs.RDBMSInfo{}, newErr
		}
		cblogger.Warnf("[NHN RDBMS] fetch created instance failed (attempt %d/%d), retrying in 10s: %v", attempt, maxGetAttempts, fetchErr)
		time.Sleep(10 * time.Second)
	}

	// Debug: Log backup information returned after creation
	cblogger.Infof("[NHN RDBMS] After creation - BackupPeriod from API: %d, BackupSchedules count: %d",
		getResp.Backup.BackupPeriod, len(getResp.Backup.BackupSchedules))

	var enrichment nhnRDSEnrichmentData
	for attempt := 1; ; attempt++ {
		var enrichErr error
		enrichment, enrichErr = handler.fetchRDBMSEnrichmentWithEndpoint(ctx, endpointFn, dbInstanceId, getResp.DBFlavorId)
		if enrichErr == nil {
			break
		}
		if attempt >= maxGetAttempts {
			newErr := handler.rollbackCreatedRDBMS(irs.IID{NameId: rdbmsReqInfo.IId.NameId, SystemId: dbInstanceId},
				fmt.Errorf("failed to fetch RDBMS enrichment data after %d attempts: %w", attempt, enrichErr))
			LoggingError(callLogInfo, newErr)
			return irs.RDBMSInfo{}, newErr
		}
		cblogger.Warnf("[NHN RDBMS] fetch enrichment data failed (attempt %d/%d), retrying in 10s: %v", attempt, maxGetAttempts, enrichErr)
		time.Sleep(10 * time.Second)
	}

	LoggingInfo(callLogInfo, start)
	result := convertNhnRDSInstanceToRDBMSInfo(getResp.nhnRDSDBInstance, rdbmsReqInfo.VpcIID.NameId, enrichment)
	// NHN appends a random suffix to the DB name; restore the user's requested name
	result.IId.NameId = rdbmsReqInfo.IId.NameId
	// Set engine based on DBVersion prefix for MariaDB
	if strings.HasPrefix(strings.ToUpper(rdbmsReqInfo.DBEngineVersion), "MARIADB_") {
		result.DBEngine = "mariadb"
	}
	return result, nil
}

// rollbackCreatedRDBMS best-effort deletes a partially-created instance when a
// post-create step fails, so the CSP is not left with an orphaned RDBMS.
func (handler *NhnCloudRDBMSHandler) rollbackCreatedRDBMS(rdbmsIID irs.IID, cause error) error {
	cblogger.Errorf("CreateRDBMS failed after instance creation (%s/%s); attempting rollback deletion: %v", rdbmsIID.NameId, rdbmsIID.SystemId, cause)
	if _, delErr := handler.DeleteRDBMS(rdbmsIID); delErr != nil {
		cblogger.Errorf("rollback deletion of RDBMS (%s/%s) failed: %v", rdbmsIID.NameId, rdbmsIID.SystemId, delErr)
		return fmt.Errorf("%w (rollback deletion also failed: %v)", cause, delErr)
	}
	cblogger.Infof("rollback deletion of RDBMS (%s/%s) succeeded", rdbmsIID.NameId, rdbmsIID.SystemId)
	return fmt.Errorf("%w (partially-created RDBMS was rolled back and deleted)", cause)
}
func (handler *NhnCloudRDBMSHandler) ListRDBMS() ([]*irs.RDBMSInfo, error) {
	cblogger.Info("NHN Cloud Driver: called ListRDBMS()")
	callLogInfo := getCallLogScheme(handler.RegionInfo.Region, call.RDBMS, "ListRDBMS", "GET /v3.0/db-instances")
	start := call.Start()

	if err := handler.checkRDSCredentials(); err != nil {
		LoggingError(callLogInfo, err)
		return nil, err
	}

	// Use a longer timeout: list call + one GET per instance.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	rdbmsList := make([]*irs.RDBMSInfo, 0)

	for _, endpointFn := range []func() (string, error){handler.rdsEndpoint, handler.rdsMariaDBEndpoint} {
		var result nhnRDSListInstancesResponse
		if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-instances", &result); err != nil {
			cblogger.Warnf("[NHN RDS] ListRDBMS: endpoint unavailable (non-fatal): %v", err)
			continue
		}
		if err := checkRDSResponseHeader(result.Header); err != nil {
			cblogger.Warnf("[NHN RDS] ListRDBMS: response header error (non-fatal): %v", err)
			continue
		}

		// NHN list endpoint omits storage/network/backup details.
		// Fetch each instance individually to get full field data.
		for _, listInst := range result.DBInstances {
			var detail nhnRDSGetInstanceResponse
			if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-instances/"+listInst.DBInstanceId, &detail); err != nil {
				newErr := fmt.Errorf("failed to get details for NHN Cloud RDS instance '%s': %w", listInst.DBInstanceId, err)
				LoggingError(callLogInfo, newErr)
				return nil, newErr
			}
			if err := checkRDSResponseHeader(detail.Header); err != nil {
				LoggingError(callLogInfo, err)
				return nil, err
			}
			enrichment, err := handler.fetchRDBMSEnrichmentWithEndpoint(ctx, endpointFn, listInst.DBInstanceId, detail.DBFlavorId)
			if err != nil {
				LoggingError(callLogInfo, err)
				return nil, err
			}
			info := convertNhnRDSInstanceToRDBMSInfo(detail.nhnRDSDBInstance, "", enrichment)
			// Set engine based on DBVersion prefix for MariaDB
			if strings.HasPrefix(strings.ToUpper(detail.DBVersion), "MARIADB_") {
				info.DBEngine = "mariadb"
			}
			rdbmsList = append(rdbmsList, &info)
		}
	}

	LoggingInfo(callLogInfo, start)
	return rdbmsList, nil
}

// GetRDBMS retrieves a specific RDBMS instance from the NHN native RDS API.
func (handler *NhnCloudRDBMSHandler) GetRDBMS(rdbmsIID irs.IID) (irs.RDBMSInfo, error) {
	cblogger.Info("NHN Cloud Driver: called GetRDBMS()")
	callLogInfo := getCallLogScheme(handler.RegionInfo.Region, call.RDBMS, rdbmsIID.NameId, "GET /v3.0/db-instances/{id}")
	start := call.Start()

	if err := handler.checkRDSCredentials(); err != nil {
		LoggingError(callLogInfo, err)
		return irs.RDBMSInfo{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Try to get instance from MySQL endpoint first, then MariaDB endpoint if not found
	for _, endpointFn := range []func() (string, error){handler.rdsEndpoint, handler.rdsMariaDBEndpoint} {
		dbInstanceId := rdbmsIID.SystemId
		if dbInstanceId == "" {
			foundId, err := handler.findRDSInstanceIDByNameWithEndpoint(ctx, endpointFn, rdbmsIID.NameId)
			if err != nil {
				continue
			}
			dbInstanceId = foundId
		}

		var result nhnRDSGetInstanceResponse
		if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-instances/"+dbInstanceId, &result); err != nil {
			continue
		}
		if err := checkRDSResponseHeader(result.Header); err != nil {
			continue
		}

		cblogger.Infof("[NHN RDBMS] Backup info from API - BackupPeriod: %d, BackupSchedules count: %d",
			result.Backup.BackupPeriod, len(result.Backup.BackupSchedules))

		enrichment, err := handler.fetchRDBMSEnrichmentWithEndpoint(ctx, endpointFn, dbInstanceId, result.DBFlavorId)
		if err != nil {
			LoggingError(callLogInfo, err)
			return irs.RDBMSInfo{}, err
		}

		LoggingInfo(callLogInfo, start)
		info := convertNhnRDSInstanceToRDBMSInfo(result.nhnRDSDBInstance, "", enrichment)
		if strings.HasPrefix(strings.ToUpper(result.DBVersion), "MARIADB_") {
			info.DBEngine = "mariadb"
		}
		return info, nil
	}

	err := fmt.Errorf("NHN Cloud RDS instance not found: %s (name: %s)", rdbmsIID.SystemId, rdbmsIID.NameId)
	LoggingError(callLogInfo, err)
	return irs.RDBMSInfo{}, err
}

func (handler *NhnCloudRDBMSHandler) DeleteRDBMS(rdbmsIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called DeleteRDBMS()")
	callLogInfo := getCallLogScheme(handler.RegionInfo.Region, call.RDBMS, rdbmsIID.NameId, "DELETE /v3.0/db-instances/{id}")
	start := call.Start()

	if err := handler.checkRDSCredentials(); err != nil {
		LoggingError(callLogInfo, err)
		return false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Determine which endpoint the instance belongs to (MySQL or MariaDB)
	endpointFn, dbInstanceId, err := handler.resolveInstanceEndpoint(ctx, rdbmsIID)
	if err != nil {
		LoggingError(callLogInfo, err)
		return false, err
	}

	// Collect DB security group IDs before deleting the instance so we can
	// clean up any SGs that were auto-created by CB-Spider.
	var getInstResp nhnRDSGetInstanceResponse
	if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-instances/"+dbInstanceId, &getInstResp); err == nil {
		_ = checkRDSResponseHeader(getInstResp.Header) // ignore header error here
	}
	autoSGIds := handler.collectAutoCBSpiderSGIdsWithEndpoint(ctx, endpointFn, getInstResp.DBSecurityGroupIds)

	var deleteResp nhnRDSJobResponse
	if err := handler.deleteRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-instances/"+dbInstanceId, &deleteResp); err != nil {
		newErr := fmt.Errorf("failed to delete NHN Cloud RDS instance '%s': %w", dbInstanceId, err)
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}
	if err := checkRDSResponseHeader(deleteResp.Header); err != nil {
		LoggingError(callLogInfo, err)
		return false, err
	}

	// Delete auto-created DB SGs (best-effort; log but do not fail)
	for _, sgId := range autoSGIds {
		if delErr := handler.deleteRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-security-groups/"+sgId, nil); delErr != nil {
			cblogger.Warnf("[NHN RDS] failed to delete auto-created DB security group '%s': %v", sgId, delErr)
		} else {
			cblogger.Infof("[NHN RDS] deleted auto-created DB security group '%s'", sgId)
		}
	}

	LoggingInfo(callLogInfo, start)
	return true, nil
}

// ---- NHN native RDS API helper methods ────────────────────────────────────

// postRDS sends a POST request to the NHN RDS for MySQL API.
func (handler *NhnCloudRDBMSHandler) postRDS(ctx context.Context, path string, body interface{}, v interface{}) error {
	return handler.postRDSWithEndpoint(ctx, handler.rdsEndpoint, path, body, v)
}

// postRDSWithEndpoint sends a POST request to the NHN RDS API at the given endpoint.
func (handler *NhnCloudRDBMSHandler) postRDSWithEndpoint(ctx context.Context, endpointFn func() (string, error), path string, body interface{}, v interface{}) error {
	endpoint, err := endpointFn()
	if err != nil {
		return err
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal NHN Cloud RDS request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create NHN Cloud RDS POST request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TC-APP-KEY", handler.rdsEffectiveAppKey(endpoint))
	req.Header.Set("X-TC-AUTHENTICATION-ID", handler.CredentialInfo.RDSUserAccessKey)
	req.Header.Set("X-TC-AUTHENTICATION-SECRET", handler.CredentialInfo.RDSSecretAccessKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call NHN Cloud RDS API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read NHN Cloud RDS API response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("NHN Cloud RDS API returned HTTP %s: %s", resp.Status, string(respBody))
	}

	if err := json.Unmarshal(respBody, v); err != nil {
		return fmt.Errorf("failed to decode NHN Cloud RDS API response: %w", err)
	}
	return nil
}

// collectAutoCBSpiderSGIds inspects the given DB SG IDs and returns those
// that were auto-created by CB-Spider (identified by description prefix).
func (handler *NhnCloudRDBMSHandler) collectAutoCBSpiderSGIds(ctx context.Context, sgIds []string) []string {
	var auto []string
	for _, id := range sgIds {
		var detail nhnRDSDBSecurityGroupDetailResponse
		if err := handler.getRDS(ctx, "/v3.0/db-security-groups/"+id, &detail); err != nil {
			continue
		}
		if checkRDSResponseHeader(detail.Header) != nil {
			continue
		}
		if strings.HasPrefix(detail.Description, "Auto-created by CB-Spider:") {
			auto = append(auto, id)
		}
	}
	return auto
}

// fetchSubnetCidr retrieves the CIDR of the given subnet ID from the NHN RDS subnet list.
// Returns empty string on any error (best-effort).
func (handler *NhnCloudRDBMSHandler) fetchSubnetCidr(ctx context.Context, subnetId string) string {
	type nhnRDSSubnet struct {
		SubnetId   string `json:"subnetId"`
		SubnetCidr string `json:"subnetCidr"`
	}
	type nhnRDSSubnetListResponse struct {
		Header  nhnRDSResponseHeader `json:"header"`
		Subnets []nhnRDSSubnet       `json:"subnets"`
	}
	var resp nhnRDSSubnetListResponse
	if err := handler.getRDS(ctx, "/v3.0/network/subnets", &resp); err != nil {
		return ""
	}
	for _, s := range resp.Subnets {
		if s.SubnetId == subnetId {
			return s.SubnetCidr
		}
	}
	return ""
}

// putRDS sends a PUT request to the NHN RDS for MySQL API.
func (handler *NhnCloudRDBMSHandler) putRDS(ctx context.Context, path string, body interface{}, v interface{}) error {
	endpoint, err := handler.rdsEndpoint()
	if err != nil {
		return err
	}

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal NHN Cloud RDS request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint+path, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create NHN Cloud RDS PUT request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TC-APP-KEY", handler.CredentialInfo.RDSMySQLAppKey)
	req.Header.Set("X-TC-AUTHENTICATION-ID", handler.CredentialInfo.RDSUserAccessKey)
	req.Header.Set("X-TC-AUTHENTICATION-SECRET", handler.CredentialInfo.RDSSecretAccessKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call NHN Cloud RDS API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read NHN Cloud RDS API response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("NHN Cloud RDS API returned HTTP %s: %s", resp.Status, string(respBody))
	}

	if v != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, v); err != nil {
			return fmt.Errorf("failed to decode NHN Cloud RDS API response: %w", err)
		}
	}
	return nil
}

// createDefaultDBSecurityGroup creates a NHN Cloud RDS DB Security Group and returns its ID.
// endpointFn selects the correct RDS endpoint (MySQL or MariaDB) for the DB instance being created.
// If publicAccess is true, allows inbound DB port from 0.0.0.0/0.
// If publicAccess is false and subnetCidr is non-empty, restricts inbound to the subnet CIDR only.
func (handler *NhnCloudRDBMSHandler) createDefaultDBSecurityGroup(ctx context.Context, endpointFn func() (string, error), name string, publicAccess bool, subnetCidr string) (string, error) {
	cidr := "0.0.0.0/0"
	if !publicAccess && subnetCidr != "" {
		cidr = subnetCidr
	}
	var desc string
	if publicAccess {
		desc = "Auto-created by CB-Spider: allow inbound DB port from 0.0.0.0/0"
	} else {
		desc = "Auto-created by CB-Spider: allow inbound DB port from subnet " + cidr
	}
	reqBody := nhnRDSCreateDBSecurityGroupRequest{
		DBSecurityGroupName: name + "-sg",
		Description:         desc,
		Rules: []nhnRDSDBSecurityGroupRule{
			{
				Direction: "INGRESS",
				EtherType: "IPV4",
				Cidr:      cidr,
				Port:      nhnRDSDBSecurityGroupPort{PortType: "DB_PORT"},
			},
		},
	}
	var resp nhnRDSCreateDBSecurityGroupResponse
	if err := handler.postRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-security-groups", reqBody, &resp); err != nil {
		return "", fmt.Errorf("failed to create NHN Cloud RDS DB security group: %w", err)
	}
	if err := checkRDSResponseHeader(resp.Header); err != nil {
		return "", fmt.Errorf("create DB security group response error: %w", err)
	}
	return resp.DBSecurityGroupId, nil
}

// deleteRDS sends a DELETE request to the NHN RDS for MySQL API.
func (handler *NhnCloudRDBMSHandler) deleteRDS(ctx context.Context, path string, v interface{}) error {
	return handler.deleteRDSWithEndpoint(ctx, handler.rdsEndpoint, path, v)
}

// deleteRDSWithEndpoint sends a DELETE request to the NHN RDS API at the given endpoint.
func (handler *NhnCloudRDBMSHandler) deleteRDSWithEndpoint(ctx context.Context, endpointFn func() (string, error), path string, v interface{}) error {
	endpoint, err := endpointFn()
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create NHN Cloud RDS DELETE request: %w", err)
	}
	req.Header.Set("X-TC-APP-KEY", handler.rdsEffectiveAppKey(endpoint))
	req.Header.Set("X-TC-AUTHENTICATION-ID", handler.CredentialInfo.RDSUserAccessKey)
	req.Header.Set("X-TC-AUTHENTICATION-SECRET", handler.CredentialInfo.RDSSecretAccessKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call NHN Cloud RDS API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read NHN Cloud RDS API response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("NHN Cloud RDS API returned HTTP %s: %s", resp.Status, string(respBody))
	}

	if v != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, v); err != nil {
			return fmt.Errorf("failed to decode NHN Cloud RDS API response: %w", err)
		}
	}
	return nil
}

// pollRDSJob polls until an async NHN RDS job completes and returns the created resourceId.
func (handler *NhnCloudRDBMSHandler) pollRDSJob(ctx context.Context, jobId string) (string, error) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("timed out waiting for NHN Cloud RDS job %s", jobId)
		case <-ticker.C:
			var jobResp nhnRDSJobResponse
			if err := handler.getRDS(ctx, "/v3.0/jobs/"+jobId, &jobResp); err != nil {
				return "", fmt.Errorf("failed to poll NHN Cloud RDS job %s: %w", jobId, err)
			}
			switch jobResp.JobStatus {
			case "COMPLETED":
				for _, rel := range jobResp.ResourceRelations {
					if rel.ResourceType == "DB_INSTANCE" {
						return rel.ResourceId, nil
					}
				}
				return "", fmt.Errorf("NHN Cloud RDS job %s completed but no DB_INSTANCE in resourceRelations", jobId)
			case "FAILED":
				return "", fmt.Errorf("NHN Cloud RDS job %s failed", jobId)
			default:
				cblogger.Infof("[NHN RDS] job %s status: %s (waiting...)", jobId, jobResp.JobStatus)
			}
		}
	}
}

// pollRDSJobWithEndpoint polls a NHN RDS async job at the given endpoint and returns the created resourceId.
func (handler *NhnCloudRDBMSHandler) pollRDSJobWithEndpoint(ctx context.Context, endpointFn func() (string, error), jobId string) (string, error) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("timed out waiting for NHN Cloud RDS job %s", jobId)
		case <-ticker.C:
			var jobResp nhnRDSJobResponse
			if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/jobs/"+jobId, &jobResp); err != nil {
				return "", fmt.Errorf("failed to poll NHN Cloud RDS job %s: %w", jobId, err)
			}
			switch jobResp.JobStatus {
			case "COMPLETED":
				for _, rel := range jobResp.ResourceRelations {
					if rel.ResourceType == "DB_INSTANCE" {
						return rel.ResourceId, nil
					}
				}
				return "", fmt.Errorf("NHN Cloud RDS job %s completed but no DB_INSTANCE in resourceRelations", jobId)
			case "FAILED":
				return "", fmt.Errorf("NHN Cloud RDS job %s failed", jobId)
			default:
				cblogger.Infof("[NHN RDS] job %s status: %s (waiting...)", jobId, jobResp.JobStatus)
			}
		}
	}
}

// findRDSInstanceIDByNameWithEndpoint finds a DB instance UUID by name at the given endpoint.
func (handler *NhnCloudRDBMSHandler) findRDSInstanceIDByNameWithEndpoint(ctx context.Context, endpointFn func() (string, error), name string) (string, error) {
	var result nhnRDSListInstancesResponse
	if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-instances", &result); err != nil {
		return "", fmt.Errorf("failed to list NHN Cloud RDS instances: %w", err)
	}
	if err := checkRDSResponseHeader(result.Header); err != nil {
		return "", err
	}
	for _, inst := range result.DBInstances {
		if inst.DBInstanceName == name || strings.HasPrefix(inst.DBInstanceName, name+"-") {
			return inst.DBInstanceId, nil
		}
	}
	return "", fmt.Errorf("NHN Cloud RDS instance with name '%s' not found", name)
}

// resolveInstanceEndpoint determines which RDS endpoint (MySQL or MariaDB) an instance belongs to.
// Returns the endpoint function and the resolved instance ID.
func (handler *NhnCloudRDBMSHandler) resolveInstanceEndpoint(ctx context.Context, rdbmsIID irs.IID) (func() (string, error), string, error) {
	for _, endpointFn := range []func() (string, error){handler.rdsEndpoint, handler.rdsMariaDBEndpoint} {
		dbInstanceId := rdbmsIID.SystemId
		if dbInstanceId == "" {
			foundId, err := handler.findRDSInstanceIDByNameWithEndpoint(ctx, endpointFn, rdbmsIID.NameId)
			if err != nil {
				continue
			}
			dbInstanceId = foundId
		}
		// Verify the instance exists at this endpoint
		var checkResp nhnRDSGetInstanceResponse
		if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-instances/"+dbInstanceId, &checkResp); err != nil {
			continue
		}
		if checkRDSResponseHeader(checkResp.Header) == nil {
			return endpointFn, dbInstanceId, nil
		}
	}
	return nil, "", fmt.Errorf("NHN Cloud RDS instance not found: %s (name: %s)", rdbmsIID.SystemId, rdbmsIID.NameId)
}

// fetchRDBMSEnrichmentWithEndpoint calls enrichment APIs using the given endpoint function.
func (handler *NhnCloudRDBMSHandler) fetchRDBMSEnrichmentWithEndpoint(ctx context.Context, endpointFn func() (string, error), dbInstanceId string, flavorId string) (nhnRDSEnrichmentData, error) {
	var data nhnRDSEnrichmentData
	data.DBFlavorName = handler.resolveRDSFlavorNameWithEndpoint(ctx, endpointFn, flavorId)

	var usersResp nhnRDSDBUserListResponse
	if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-instances/"+dbInstanceId+"/db-users", &usersResp); err != nil {
		return data, fmt.Errorf("failed to list DB users for NHN Cloud RDS instance '%s': %w", dbInstanceId, err)
	}
	if err := checkRDSResponseHeader(usersResp.Header); err != nil {
		return data, fmt.Errorf("db-users response error for instance '%s': %w", dbInstanceId, err)
	}
	for _, u := range usersResp.DBUsers {
		if u.DBUserStatus == "STABLE" {
			data.MasterUserName = u.DBUserName
			break
		}
	}

	var netResp nhnRDSNetworkInfoResponse
	if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-instances/"+dbInstanceId+"/network-info", &netResp); err != nil {
		return data, fmt.Errorf("failed to get network info for NHN Cloud RDS instance '%s': %w", dbInstanceId, err)
	}
	if err := checkRDSResponseHeader(netResp.Header); err != nil {
		return data, fmt.Errorf("network-info response error for instance '%s': %w", dbInstanceId, err)
	}
	data.SubnetId = netResp.Subnet.SubnetId
	data.SubnetName = netResp.Subnet.SubnetName
	data.SubnetCidr = netResp.Subnet.SubnetCidr
	for _, ep := range netResp.EndPoints {
		if ep.EndPointType == "EXTERNAL" {
			data.UsePublicAccess = true
			if ep.Domain != "" {
				data.PublicEndpoint = ep.Domain
			} else {
				data.PublicEndpoint = ep.IPAddress
			}
			break
		}
	}

	var storResp nhnRDSStorageInfoResponse
	if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-instances/"+dbInstanceId+"/storage-info", &storResp); err != nil {
		return data, fmt.Errorf("failed to get storage info for NHN Cloud RDS instance '%s': %w", dbInstanceId, err)
	}
	if err := checkRDSResponseHeader(storResp.Header); err != nil {
		return data, fmt.Errorf("storage-info response error for instance '%s': %w", dbInstanceId, err)
	}
	data.StorageType = storResp.StorageType
	data.StorageSize = storResp.StorageSize

	var backupResp struct {
		Header nhnRDSResponseHeader `json:"header"`
		nhnRDSBackupConfig
	}
	if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-instances/"+dbInstanceId+"/backup-info", &backupResp); err == nil {
		if checkErr := checkRDSResponseHeader(backupResp.Header); checkErr == nil {
			data.BackupConfig = backupResp.nhnRDSBackupConfig
		}
	}

	return data, nil
}

// collectAutoCBSpiderSGIdsWithEndpoint inspects DB SG IDs using the given endpoint.
func (handler *NhnCloudRDBMSHandler) collectAutoCBSpiderSGIdsWithEndpoint(ctx context.Context, endpointFn func() (string, error), sgIds []string) []string {
	var auto []string
	for _, id := range sgIds {
		var detail nhnRDSDBSecurityGroupDetailResponse
		if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-security-groups/"+id, &detail); err != nil {
			continue
		}
		if checkRDSResponseHeader(detail.Header) != nil {
			continue
		}
		if strings.HasPrefix(detail.Description, "Auto-created by CB-Spider:") {
			auto = append(auto, id)
		}
	}
	return auto
}

// resolveRDSFlavorNameWithEndpoint resolves a DB flavor UUID to its name using the given endpoint.
func (handler *NhnCloudRDBMSHandler) resolveRDSFlavorNameWithEndpoint(ctx context.Context, endpointFn func() (string, error), flavorId string) string {
	var result nhnRDSFlavorListResponse
	if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-flavors", &result); err != nil {
		return flavorId
	}
	if checkRDSResponseHeader(result.Header) != nil {
		return flavorId
	}
	for _, f := range result.DBFlavors {
		if f.DBFlavorId == flavorId {
			return f.DBFlavorName
		}
	}
	return flavorId
}

// findRDSInstanceIDByName finds a DB instance UUID by name using the NHN native API.
func (handler *NhnCloudRDBMSHandler) findRDSInstanceIDByName(ctx context.Context, name string) (string, error) {
	var result nhnRDSListInstancesResponse
	if err := handler.getRDS(ctx, "/v3.0/db-instances", &result); err != nil {
		return "", fmt.Errorf("failed to list NHN Cloud RDS instances: %w", err)
	}
	if err := checkRDSResponseHeader(result.Header); err != nil {
		return "", err
	}
	for _, inst := range result.DBInstances {
		// NHN may append a random suffix (e.g. "my-db-01-<random>"); match exact or by prefix
		if inst.DBInstanceName == name || strings.HasPrefix(inst.DBInstanceName, name+"-") {
			return inst.DBInstanceId, nil
		}
	}
	return "", fmt.Errorf("NHN Cloud RDS instance with name '%s' not found", name)
}

func convertNhnRDSInstanceToRDBMSInfo(inst nhnRDSDBInstance, ownerVPCName string, e nhnRDSEnrichmentData) irs.RDBMSInfo {
	// Prefer publicEndpoint from network-info API; fallback to embedded endpoints
	endpoint := e.PublicEndpoint
	port := inst.DBPort // DBPort from instance
	if endpoint == "" {
		for _, ep := range inst.Network.Endpoints {
			if ep.Address != "" {
				endpoint = ep.Address
				if ep.Port > 0 {
					port = ep.Port
				}
				break
			}
		}
	}
	if endpoint == "" {
		endpoint = "NA"
	} else if port > 0 {
		// Append port to endpoint
		endpoint = fmt.Sprintf("%s:%d", endpoint, port)
	}

	master := e.MasterUserName
	if master == "" {
		master = "NA"
	}

	storageType := e.StorageType
	if storageType == "" {
		storageType = inst.Storage.StorageType
	}
	storageSize := e.StorageSize
	if storageSize == 0 {
		storageSize = inst.Storage.StorageSize
	}

	subnetId := e.SubnetId
	if subnetId == "" {
		subnetId = inst.Network.SubnetId
	}

	var subnetIIDs []irs.IID
	if subnetId != "" {
		subnetIIDs = []irs.IID{{NameId: e.SubnetName, SystemId: subnetId}}
	}

	publicAccess := e.UsePublicAccess

	// Backup info: prefer enrichment data (from /backup-info), fallback to inst.Backup
	backupPeriod := e.BackupConfig.BackupPeriod
	if backupPeriod == 0 {
		backupPeriod = inst.Backup.BackupPeriod
	}
	backupTime := "00:00"
	backupSchedules := e.BackupConfig.BackupSchedules
	if len(backupSchedules) == 0 {
		backupSchedules = inst.Backup.BackupSchedules
	}
	if len(backupSchedules) > 0 {
		// NHN returns "HH:MM:SS" format, convert to "HH:MM" for Spider
		rawTime := backupSchedules[0].BackupWndBgnTime
		if len(rawTime) >= 5 {
			backupTime = rawTime[:5] // "03:00:00" -> "03:00"
		} else {
			backupTime = rawTime
		}
	}

	cblogger.Infof("[NHN RDBMS] Final backup values - Period: %d, Time: %s (enrichment schedules: %d, inst schedules: %d)",
		backupPeriod, backupTime, len(e.BackupConfig.BackupSchedules), len(inst.Backup.BackupSchedules))

	createdTime, _ := time.Parse(time.RFC3339, inst.CreatedYmdt)

	dbEngine := "mysql"
	if strings.HasPrefix(strings.ToUpper(inst.DBVersion), "MARIADB_") {
		dbEngine = "mariadb"
	}

	return irs.RDBMSInfo{
		IId: irs.IID{
			NameId:   inst.DBInstanceName,
			SystemId: inst.DBInstanceId,
		},
		VpcIID: irs.IID{NameId: ownerVPCName, SystemId: "NA"},

		DBEngine:        dbEngine,
		DBEngineVersion: inst.DBVersion,
		DBInstanceSpec:  e.DBFlavorName,
		DBInstanceType:  "NA",

		StorageType: storageType,
		StorageSize: strconv.Itoa(storageSize),

		SubnetIIDs: subnetIIDs,

		Endpoint: endpoint,

		MasterUserName: master,

		HighAvailability: inst.UseHighAvailability,

		BackupRetentionDays: backupPeriod,
		BackupTime:          backupTime,

		PublicAccess:       publicAccess,
		Encryption:         false,
		DeletionProtection: inst.UseDeletionProtection,

		Status:      convertNhnRDSStatusToRDBMSStatus(inst.DBInstanceStatus),
		CreatedTime: createdTime,

		KeyValueList: []irs.KeyValue{
			{Key: "DBInstanceStatus", Value: inst.DBInstanceStatus},
			{Key: "DBFlavorId", Value: inst.DBFlavorId},
			{Key: "SubnetId", Value: subnetId},
		},
	}
}

func convertNhnRDSStatusToRDBMSStatus(status string) irs.RDBMSStatus {
	switch strings.ToUpper(status) {
	case "AVAILABLE":
		return irs.RDBMSAvailable
	case "CREATING":
		return irs.RDBMSCreating
	case "STOPPING", "MAINTENANCE":
		return irs.RDBMSCreating
	case "STOPPED":
		return irs.RDBMSStopped
	case "DELETING", "DELETED":
		return irs.RDBMSDeleting
	case "FAIL_TO_CREATE", "FAIL_TO_DELETE":
		return irs.RDBMSError
	default:
		return irs.RDBMSError
	}
}

// fetchRDBMSEnrichment calls supplementary APIs to get fields not present in the
// main GET /v3.0/db-instances/{id} response:
//   - GET /v3.0/db-instances/{id}/db-users       → MasterUserName (first STABLE user)
//   - GET /v3.0/db-instances/{id}/network-info   → PublicEndpoint (EXTERNAL) + SubnetId
//   - GET /v3.0/db-instances/{id}/storage-info   → StorageType + StorageSize
//   - GET /v3.0/db-flavors                       → DBFlavorName (UUID → name)
func (handler *NhnCloudRDBMSHandler) fetchRDBMSEnrichment(ctx context.Context, dbInstanceId string, flavorId string) (nhnRDSEnrichmentData, error) {
	var data nhnRDSEnrichmentData
	data.DBFlavorName = handler.resolveRDSFlavorName(ctx, flavorId)

	var usersResp nhnRDSDBUserListResponse
	if err := handler.getRDS(ctx, "/v3.0/db-instances/"+dbInstanceId+"/db-users", &usersResp); err != nil {
		return data, fmt.Errorf("failed to list DB users for NHN Cloud RDS instance '%s': %w", dbInstanceId, err)
	}
	if err := checkRDSResponseHeader(usersResp.Header); err != nil {
		return data, fmt.Errorf("db-users response error for instance '%s': %w", dbInstanceId, err)
	}
	for _, u := range usersResp.DBUsers {
		if u.DBUserStatus == "STABLE" {
			data.MasterUserName = u.DBUserName
			break
		}
	}

	var netResp nhnRDSNetworkInfoResponse
	if err := handler.getRDS(ctx, "/v3.0/db-instances/"+dbInstanceId+"/network-info", &netResp); err != nil {
		return data, fmt.Errorf("failed to get network info for NHN Cloud RDS instance '%s': %w", dbInstanceId, err)
	}
	if err := checkRDSResponseHeader(netResp.Header); err != nil {
		return data, fmt.Errorf("network-info response error for instance '%s': %w", dbInstanceId, err)
	}
	data.SubnetId = netResp.Subnet.SubnetId
	data.SubnetName = netResp.Subnet.SubnetName
	data.SubnetCidr = netResp.Subnet.SubnetCidr
	for _, ep := range netResp.EndPoints {
		if ep.EndPointType == "EXTERNAL" {
			data.UsePublicAccess = true
			if ep.Domain != "" {
				data.PublicEndpoint = ep.Domain
			} else {
				data.PublicEndpoint = ep.IPAddress
			}
			break
		}
	}

	var storResp nhnRDSStorageInfoResponse
	if err := handler.getRDS(ctx, "/v3.0/db-instances/"+dbInstanceId+"/storage-info", &storResp); err != nil {
		return data, fmt.Errorf("failed to get storage info for NHN Cloud RDS instance '%s': %w", dbInstanceId, err)
	}
	if err := checkRDSResponseHeader(storResp.Header); err != nil {
		return data, fmt.Errorf("storage-info response error for instance '%s': %w", dbInstanceId, err)
	}
	data.StorageType = storResp.StorageType
	data.StorageSize = storResp.StorageSize

	// Try to fetch backup info (may not be available as separate endpoint)
	var backupResp struct {
		Header nhnRDSResponseHeader `json:"header"`
		nhnRDSBackupConfig
	}
	if err := handler.getRDS(ctx, "/v3.0/db-instances/"+dbInstanceId+"/backup-info", &backupResp); err == nil {
		if checkErr := checkRDSResponseHeader(backupResp.Header); checkErr == nil {
			data.BackupConfig = backupResp.nhnRDSBackupConfig
			cblogger.Infof("[NHN RDBMS] Backup info from /backup-info - BackupPeriod: %d, BackupSchedules: %d",
				data.BackupConfig.BackupPeriod, len(data.BackupConfig.BackupSchedules))
		} else {
			cblogger.Infof("[NHN RDBMS] /backup-info endpoint exists but returned error: %v", checkErr)
		}
	} else {
		cblogger.Infof("[NHN RDBMS] /backup-info endpoint not available: %v", err)
	}

	return data, nil
}

// resolveRDSFlavorName resolves a DB flavor UUID to its name (e.g. "m2.c2m4") via
// GET /v3.0/db-flavors. Returns the UUID unchanged on any error (best-effort).
func (handler *NhnCloudRDBMSHandler) resolveRDSFlavorName(ctx context.Context, flavorId string) string {
	var result nhnRDSFlavorListResponse
	if err := handler.getRDS(ctx, "/v3.0/db-flavors", &result); err != nil {
		return flavorId
	}
	if checkRDSResponseHeader(result.Header) != nil {
		return flavorId
	}
	for _, f := range result.DBFlavors {
		if f.DBFlavorId == flavorId {
			return f.DBFlavorName
		}
	}
	return flavorId
}

// resolveRDSFlavorId resolves a DB flavor name (e.g. "m2.c2m4") to its UUID via
// GET /v3.0/db-flavors. If the input already looks like a UUID it is returned as-is.
func (handler *NhnCloudRDBMSHandler) resolveRDSFlavorId(ctx context.Context, nameOrId string) (string, error) {
	return handler.resolveRDSFlavorIdWithEndpoint(ctx, handler.rdsEndpoint, nameOrId)
}

func (handler *NhnCloudRDBMSHandler) resolveRDSFlavorIdWithEndpoint(ctx context.Context, endpointFn func() (string, error), nameOrId string) (string, error) {
	if isRDSUUID(nameOrId) {
		return nameOrId, nil
	}
	var result nhnRDSFlavorListResponse
	if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/db-flavors", &result); err != nil {
		return "", fmt.Errorf("failed to list NHN Cloud RDS flavors: %w", err)
	}
	if err := checkRDSResponseHeader(result.Header); err != nil {
		return "", err
	}
	for _, f := range result.DBFlavors {
		if strings.EqualFold(f.DBFlavorName, nameOrId) {
			return f.DBFlavorId, nil
		}
	}
	return "", fmt.Errorf("NHN Cloud RDS flavor '%s' not found; check NHN Console → RDS for MySQL → DB Instance Spec", nameOrId)
}

// findDefaultParameterGroupId fetches parameter groups from GET /v3.0/parameter-groups
// and returns one whose dbVersion matches, or the first available group.
func (handler *NhnCloudRDBMSHandler) findDefaultParameterGroupId(ctx context.Context, dbVersion string) (string, error) {
	return handler.findDefaultParameterGroupIdWithEndpoint(ctx, handler.rdsEndpoint, dbVersion)
}

func (handler *NhnCloudRDBMSHandler) findDefaultParameterGroupIdWithEndpoint(ctx context.Context, endpointFn func() (string, error), dbVersion string) (string, error) {
	var result nhnRDSParameterGroupListResponse
	if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/parameter-groups", &result); err != nil {
		return "", fmt.Errorf("failed to list NHN Cloud RDS parameter groups: %w", err)
	}
	if err := checkRDSResponseHeader(result.Header); err != nil {
		return "", err
	}
	if len(result.ParameterGroups) == 0 {
		return "", fmt.Errorf("no NHN Cloud RDS parameter groups found; create one in NHN Console → Database → RDS for MySQL → Parameter Groups")
	}
	for _, pg := range result.ParameterGroups {
		if strings.EqualFold(pg.DBVersion, dbVersion) {
			return pg.ParameterGroupId, nil
		}
	}
	// Fall back to first available parameter group
	return result.ParameterGroups[0].ParameterGroupId, nil
}

// ─── rdbmsDatabaseManager interface implementation ───────────────────────────
// NHN Cloud RDS manages databases ("DB schemas") via its v3.0 REST API.
// SQL-level CREATE/DROP DATABASE is forbidden on NHN RDS instances.

type nhnRDSDBSchema struct {
	DBSchemaId     string `json:"dbSchemaId"`
	DBSchemaName   string `json:"dbSchemaName"`
	DBSchemaStatus string `json:"dbSchemaStatus"`
}

type nhnRDSListSchemasResponse struct {
	Header    nhnRDSResponseHeader `json:"header"`
	DBSchemas []nhnRDSDBSchema     `json:"dbSchemas"`
}

type nhnRDSCreateSchemaRequest struct {
	DBSchemaName string `json:"dbSchemaName"`
}

type nhnRDSSchemaJobResponse struct {
	Header nhnRDSResponseHeader `json:"header"`
	JobId  string               `json:"jobId"`
}

// pollRDSJobSimpleWithEndpoint polls a NHN RDS async job to reach a terminal state using the given endpoint.
func (handler *NhnCloudRDBMSHandler) pollRDSJobSimpleWithEndpoint(ctx context.Context, endpointFn func() (string, error), jobId string) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for NHN Cloud RDS job %s", jobId)
		case <-ticker.C:
			var jobResp nhnRDSJobResponse
			if err := handler.getRDSWithEndpoint(ctx, endpointFn, "/v3.0/jobs/"+jobId, &jobResp); err != nil {
				return fmt.Errorf("failed to poll NHN Cloud RDS job %s: %w", jobId, err)
			}
			switch jobResp.JobStatus {
			case "COMPLETED":
				return nil
			case "FAILED", "ERROR", "CANCELED", "INTERRUPTED", "FAIL_TO_READY", "DELETED":
				return fmt.Errorf("NHN Cloud RDS job %s ended with status: %s", jobId, jobResp.JobStatus)
			default:
				cblogger.Infof("[NHN RDS] job %s status: %s (waiting...)", jobId, jobResp.JobStatus)
			}
		}
	}
}

// CreateDatabase creates a DB schema on the NHN Cloud RDS instance via v3.0 REST API.
func (handler *NhnCloudRDBMSHandler) CreateDatabase(rdbmsSystemId, dbEngine, dbName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	endpointFn := handler.rdsEndpointForEngine(dbEngine)
	var resp nhnRDSSchemaJobResponse
	if err := handler.postRDSWithEndpoint(ctx, endpointFn, fmt.Sprintf("/v3.0/db-instances/%s/db-schemas", rdbmsSystemId),
		nhnRDSCreateSchemaRequest{DBSchemaName: dbName}, &resp); err != nil {
		return fmt.Errorf("NHN RDS CreateDatabase: %w", err)
	}
	if err := checkRDSResponseHeader(resp.Header); err != nil {
		return fmt.Errorf("NHN RDS CreateDatabase: %w", err)
	}
	return handler.pollRDSJobSimpleWithEndpoint(ctx, endpointFn, resp.JobId)
}

// ListDatabases lists DB schemas on the NHN Cloud RDS instance.
func (handler *NhnCloudRDBMSHandler) ListDatabases(rdbmsSystemId, dbEngine string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var resp nhnRDSListSchemasResponse
	if err := handler.getRDSWithEndpoint(ctx, handler.rdsEndpointForEngine(dbEngine), fmt.Sprintf("/v3.0/db-instances/%s/db-schemas", rdbmsSystemId), &resp); err != nil {
		return nil, fmt.Errorf("NHN RDS ListDatabases: %w", err)
	}
	if err := checkRDSResponseHeader(resp.Header); err != nil {
		return nil, fmt.Errorf("NHN RDS ListDatabases: %w", err)
	}
	names := make([]string, 0, len(resp.DBSchemas))
	for _, s := range resp.DBSchemas {
		names = append(names, s.DBSchemaName)
	}
	return names, nil
}

// DeleteDatabase deletes a DB schema from the NHN Cloud RDS instance.
func (handler *NhnCloudRDBMSHandler) DeleteDatabase(rdbmsSystemId, dbEngine, dbName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	endpointFn := handler.rdsEndpointForEngine(dbEngine)
	// Find the schema ID by name.
	var listResp nhnRDSListSchemasResponse
	if err := handler.getRDSWithEndpoint(ctx, endpointFn, fmt.Sprintf("/v3.0/db-instances/%s/db-schemas", rdbmsSystemId), &listResp); err != nil {
		return fmt.Errorf("NHN RDS DeleteDatabase (list): %w", err)
	}
	if err := checkRDSResponseHeader(listResp.Header); err != nil {
		return fmt.Errorf("NHN RDS DeleteDatabase (list): %w", err)
	}
	var schemaId string
	for _, s := range listResp.DBSchemas {
		if strings.EqualFold(s.DBSchemaName, dbName) {
			schemaId = s.DBSchemaId
			break
		}
	}
	if schemaId == "" {
		return fmt.Errorf("NHN RDS DeleteDatabase: schema %q not found in instance %s", dbName, rdbmsSystemId)
	}

	var resp nhnRDSSchemaJobResponse
	if err := handler.deleteRDSWithEndpoint(ctx, endpointFn, fmt.Sprintf("/v3.0/db-instances/%s/db-schemas/%s", rdbmsSystemId, schemaId), &resp); err != nil {
		return fmt.Errorf("NHN RDS DeleteDatabase: %w", err)
	}
	if err := checkRDSResponseHeader(resp.Header); err != nil {
		return fmt.Errorf("NHN RDS DeleteDatabase: %w", err)
	}
	return handler.pollRDSJobSimpleWithEndpoint(ctx, endpointFn, resp.JobId)
}

// isRDSUUID returns true if s is in standard UUID format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx).
func isRDSUUID(s string) bool {
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}
