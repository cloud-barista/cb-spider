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
	BackupWndBgnTime  string `json:"backupWndBgnTime"`  // HH:mm format — backup window begin time
	BackupWndDuration int    `json:"backupWndDuration"` // hours
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

// GetMetaInfo returns metadata queried dynamically from NHN Cloud RDS for MySQL API v3.0.
func (handler *NhnCloudRDBMSHandler) GetMetaInfo(dbEngine string) (irs.RDBMSMetaInfo, error) {
	cblogger.Info("NHN Cloud Driver: called GetMetaInfo()")
	callLogInfo := getCallLogScheme(handler.RegionInfo.Region, call.RDBMS, "GetMetaInfo", "GetMetaInfo()")
	start := call.Start()

	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		LoggingError(callLogInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}
	if requestedEngine != "mysql" {
		err := fmt.Errorf("NHN Cloud RDS native metadata API supports only mysql, requested: %s", requestedEngine)
		LoggingError(callLogInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}

	if err := handler.checkRDSCredentials(); err != nil {
		LoggingError(callLogInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	dbVersions, err := handler.listRDSDBVersions(ctx)
	if err != nil {
		newErr := fmt.Errorf("failed to list DB versions from NHN Cloud RDS API: %w", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.RDBMSMetaInfo{}, newErr
	}
	storageTypes, err := handler.listRDSStorageTypes(ctx)
	if err != nil {
		newErr := fmt.Errorf("failed to list storage types from NHN Cloud RDS API: %w", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.RDBMSMetaInfo{}, newErr
	}

	if len(dbVersions) == 0 {
		err := errors.New("NHN Cloud RDS API returned no DB versions")
		LoggingError(callLogInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}
	if len(storageTypes) == 0 {
		err := errors.New("NHN Cloud RDS API returned no storage types")
		LoggingError(callLogInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}

	supportedEngines := map[string][]string{
		"mysql": dbVersions,
	}
	storageTypeOptions := map[string][]string{
		"mysql": storageTypes,
	}

	storageSizeRange := irs.StorageSizeRange{Min: 20, Max: 2048}

	metaInfo, err := irs.BuildRDBMSMetaInfo(requestedEngine, supportedEngines, storageTypeOptions, storageSizeRange, true, true, true, true, false)
	if err != nil {
		LoggingError(callLogInfo, err)
		return irs.RDBMSMetaInfo{}, err
	}

	LoggingInfo(callLogInfo, start)
	return metaInfo, nil
}

func (handler *NhnCloudRDBMSHandler) listRDSDBVersions(ctx context.Context) ([]string, error) {
	var result nhnRDSDBVersionsResponse
	if err := handler.getRDS(ctx, "/v3.0/db-versions", &result); err != nil {
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

func (handler *NhnCloudRDBMSHandler) listRDSStorageTypes(ctx context.Context) ([]string, error) {
	var result nhnRDSStorageTypesResponse
	if err := handler.getRDS(ctx, "/v3.0/storage-types", &result); err != nil {
		return nil, err
	}
	if err := checkRDSResponseHeader(result.Header); err != nil {
		return nil, err
	}

	return extractStringValues(result.StorageTypes, "storageType"), nil
}

func (handler *NhnCloudRDBMSHandler) getRDS(ctx context.Context, path string, v interface{}) error {
	endpoint, err := handler.rdsEndpoint()
	if err != nil {
		return err
	}
	cblogger.Infof("[NHN RDS] calling %s%s (appKey len=%d, accessKeyID len=%d, secretKey len=%d)",
		endpoint, path,
		len(handler.CredentialInfo.RDSAppKey),
		len(handler.CredentialInfo.RDSUserAccessKey),
		len(handler.CredentialInfo.RDSSecretAccessKey),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create NHN Cloud RDS request: %w", err)
	}
	req.Header.Set("X-TC-APP-KEY", handler.CredentialInfo.RDSAppKey)
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
	cblogger.Infof("[NHN RDS] raw response: %s", string(body))

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

func (handler *NhnCloudRDBMSHandler) checkRDSCredentials() error {
	isUnset := func(v string) bool { return v == "" || v == "Not set" }
	var missing []string
	if isUnset(handler.CredentialInfo.RDSAppKey) {
		missing = append(missing, "'appKey'")
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
				"  1. appKey       : NHN Cloud Console → Database → RDS for MySQL → URL & AppKey\n"+
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

	var result nhnRDSListInstancesResponse
	if err := handler.getRDS(ctx, "/v3.0/db-instances", &result); err != nil {
		LoggingError(callLogInfo, err)
		return nil, err
	}
	if err := checkRDSResponseHeader(result.Header); err != nil {
		LoggingError(callLogInfo, err)
		return nil, err
	}

	iidList := make([]*irs.IID, 0, len(result.DBInstances))
	for _, inst := range result.DBInstances {
		iidList = append(iidList, &irs.IID{
			NameId:   inst.DBInstanceName,
			SystemId: inst.DBInstanceId,
		})
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

	dbPort := 3306
	if rdbmsReqInfo.Port != "" {
		if p, convErr := strconv.Atoi(rdbmsReqInfo.Port); convErr == nil {
			dbPort = p
		}
	}

	backupPeriod := rdbmsReqInfo.BackupRetentionDays
	if backupPeriod <= 0 {
		backupPeriod = 1
	}
	backupHour, backupMinute := 3, 0
	if rdbmsReqInfo.BackupTime != "" {
		parts := strings.SplitN(rdbmsReqInfo.BackupTime, ":", 2)
		if len(parts) == 2 {
			if h, parseErr := strconv.Atoi(parts[0]); parseErr == nil {
				backupHour = h
			}
			if m, parseErr := strconv.Atoi(parts[1]); parseErr == nil {
				backupMinute = m
			}
		}
	}

	// NHN RDS provisioning is async — allow up to 15 minutes
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Resolve DB flavor UUID from name or pass-through if already a UUID
	flavorId, err := handler.resolveRDSFlavorId(ctx, rdbmsReqInfo.DBInstanceSpec)
	if err != nil {
		newErr := fmt.Errorf("failed to resolve NHN Cloud RDS flavor '%s': %w", rdbmsReqInfo.DBInstanceSpec, err)
		LoggingError(callLogInfo, newErr)
		return irs.RDBMSInfo{}, newErr
	}

	// Fetch a parameter group for the requested DB version
	paramGroupId, err := handler.findDefaultParameterGroupId(ctx, rdbmsReqInfo.DBEngineVersion)
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
		autoSGId, sgErr := handler.createDefaultDBSecurityGroup(ctx, rdbmsReqInfo.IId.NameId, rdbmsReqInfo.PublicAccess, subnetCidr)
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
					BackupWndDuration: 1,
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
	if err := handler.postRDS(ctx, "/v3.0/db-instances", reqBody, &createResp); err != nil {
		newErr := fmt.Errorf("failed to submit NHN Cloud RDS create request: %w", err)
		LoggingError(callLogInfo, newErr)
		return irs.RDBMSInfo{}, newErr
	}
	if err := checkRDSResponseHeader(createResp.Header); err != nil {
		LoggingError(callLogInfo, err)
		return irs.RDBMSInfo{}, err
	}

	// Poll until the async job completes and we have the instance UUID
	dbInstanceId, err := handler.pollRDSJob(ctx, createResp.JobId)
	if err != nil {
		newErr := fmt.Errorf("NHN Cloud RDS instance creation job failed: %w", err)
		LoggingError(callLogInfo, newErr)
		return irs.RDBMSInfo{}, newErr
	}

	var getResp nhnRDSGetInstanceResponse
	if err := handler.getRDS(ctx, "/v3.0/db-instances/"+dbInstanceId, &getResp); err != nil {
		newErr := fmt.Errorf("failed to fetch newly created NHN Cloud RDS instance: %w", err)
		LoggingError(callLogInfo, newErr)
		return irs.RDBMSInfo{}, newErr
	}
	if err := checkRDSResponseHeader(getResp.Header); err != nil {
		LoggingError(callLogInfo, err)
		return irs.RDBMSInfo{}, err
	}

	enrichment, err := handler.fetchRDBMSEnrichment(ctx, dbInstanceId, getResp.DBFlavorId)
	if err != nil {
		LoggingError(callLogInfo, err)
		return irs.RDBMSInfo{}, err
	}

	LoggingInfo(callLogInfo, start)
	result := convertNhnRDSInstanceToRDBMSInfo(getResp.nhnRDSDBInstance, rdbmsReqInfo.VpcIID.NameId, enrichment)
	// NHN appends a random suffix to the DB name; restore the user's requested name
	result.IId.NameId = rdbmsReqInfo.IId.NameId
	return result, nil
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

	var result nhnRDSListInstancesResponse
	if err := handler.getRDS(ctx, "/v3.0/db-instances", &result); err != nil {
		newErr := fmt.Errorf("failed to list NHN Cloud RDS instances: %w", err)
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}
	if err := checkRDSResponseHeader(result.Header); err != nil {
		LoggingError(callLogInfo, err)
		return nil, err
	}

	// NHN list endpoint omits storage/network/backup details.
	// Fetch each instance individually to get full field data.
	rdbmsList := make([]*irs.RDBMSInfo, 0, len(result.DBInstances))
	for _, listInst := range result.DBInstances {
		var detail nhnRDSGetInstanceResponse
		if err := handler.getRDS(ctx, "/v3.0/db-instances/"+listInst.DBInstanceId, &detail); err != nil {
			newErr := fmt.Errorf("failed to get details for NHN Cloud RDS instance '%s': %w", listInst.DBInstanceId, err)
			LoggingError(callLogInfo, newErr)
			return nil, newErr
		}
		if err := checkRDSResponseHeader(detail.Header); err != nil {
			LoggingError(callLogInfo, err)
			return nil, err
		}
		enrichment, err := handler.fetchRDBMSEnrichment(ctx, listInst.DBInstanceId, detail.DBFlavorId)
		if err != nil {
			LoggingError(callLogInfo, err)
			return nil, err
		}
		info := convertNhnRDSInstanceToRDBMSInfo(detail.nhnRDSDBInstance, "", enrichment)
		rdbmsList = append(rdbmsList, &info)
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

	dbInstanceId := rdbmsIID.SystemId
	if dbInstanceId == "" {
		foundId, err := handler.findRDSInstanceIDByName(ctx, rdbmsIID.NameId)
		if err != nil {
			LoggingError(callLogInfo, err)
			return irs.RDBMSInfo{}, err
		}
		dbInstanceId = foundId
	}

	var result nhnRDSGetInstanceResponse
	if err := handler.getRDS(ctx, "/v3.0/db-instances/"+dbInstanceId, &result); err != nil {
		newErr := fmt.Errorf("failed to get NHN Cloud RDS instance '%s': %w", dbInstanceId, err)
		LoggingError(callLogInfo, newErr)
		return irs.RDBMSInfo{}, newErr
	}
	if err := checkRDSResponseHeader(result.Header); err != nil {
		LoggingError(callLogInfo, err)
		return irs.RDBMSInfo{}, err
	}

	enrichment, err := handler.fetchRDBMSEnrichment(ctx, dbInstanceId, result.DBFlavorId)
	if err != nil {
		LoggingError(callLogInfo, err)
		return irs.RDBMSInfo{}, err
	}

	LoggingInfo(callLogInfo, start)
	return convertNhnRDSInstanceToRDBMSInfo(result.nhnRDSDBInstance, "", enrichment), nil
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

	dbInstanceId := rdbmsIID.SystemId
	if dbInstanceId == "" {
		foundId, err := handler.findRDSInstanceIDByName(ctx, rdbmsIID.NameId)
		if err != nil {
			LoggingError(callLogInfo, err)
			return false, err
		}
		dbInstanceId = foundId
	}

	// Collect DB security group IDs before deleting the instance so we can
	// clean up any SGs that were auto-created by CB-Spider.
	var getInstResp nhnRDSGetInstanceResponse
	if err := handler.getRDS(ctx, "/v3.0/db-instances/"+dbInstanceId, &getInstResp); err == nil {
		_ = checkRDSResponseHeader(getInstResp.Header) // ignore header error here
	}
	autoSGIds := handler.collectAutoCBSpiderSGIds(ctx, getInstResp.DBSecurityGroupIds)

	var deleteResp nhnRDSJobResponse
	if err := handler.deleteRDS(ctx, "/v3.0/db-instances/"+dbInstanceId, &deleteResp); err != nil {
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
		if delErr := handler.deleteRDS(ctx, "/v3.0/db-security-groups/"+sgId, nil); delErr != nil {
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
	endpoint, err := handler.rdsEndpoint()
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
	req.Header.Set("X-TC-APP-KEY", handler.CredentialInfo.RDSAppKey)
	req.Header.Set("X-TC-AUTHENTICATION-ID", handler.CredentialInfo.RDSUserAccessKey)
	req.Header.Set("X-TC-AUTHENTICATION-SECRET", handler.CredentialInfo.RDSSecretAccessKey)

	cblogger.Infof("[NHN RDS] POST %s%s", endpoint, path)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to call NHN Cloud RDS API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read NHN Cloud RDS API response: %w", err)
	}
	cblogger.Infof("[NHN RDS] POST %s%s response(%d): %s", endpoint, path, resp.StatusCode, string(respBody))

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
	req.Header.Set("X-TC-APP-KEY", handler.CredentialInfo.RDSAppKey)
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
	cblogger.Infof("[NHN RDS] PUT %s%s response(%d): %s", endpoint, path, resp.StatusCode, string(respBody))

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
// If publicAccess is true, allows inbound DB port from 0.0.0.0/0.
// If publicAccess is false and subnetCidr is non-empty, restricts inbound to the subnet CIDR only.
func (handler *NhnCloudRDBMSHandler) createDefaultDBSecurityGroup(ctx context.Context, name string, publicAccess bool, subnetCidr string) (string, error) {
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
	if err := handler.postRDS(ctx, "/v3.0/db-security-groups", reqBody, &resp); err != nil {
		return "", fmt.Errorf("failed to create NHN Cloud RDS DB security group: %w", err)
	}
	if err := checkRDSResponseHeader(resp.Header); err != nil {
		return "", fmt.Errorf("create DB security group response error: %w", err)
	}
	return resp.DBSecurityGroupId, nil
}

// deleteRDS sends a DELETE request to the NHN RDS for MySQL API.
func (handler *NhnCloudRDBMSHandler) deleteRDS(ctx context.Context, path string, v interface{}) error {
	endpoint, err := handler.rdsEndpoint()
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create NHN Cloud RDS DELETE request: %w", err)
	}
	req.Header.Set("X-TC-APP-KEY", handler.CredentialInfo.RDSAppKey)
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
	cblogger.Infof("[NHN RDS] DELETE %s%s response(%d): %s", endpoint, path, resp.StatusCode, string(respBody))

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
	if endpoint == "" {
		for _, ep := range inst.Network.Endpoints {
			if ep.Address != "" {
				endpoint = ep.Address
				break
			}
		}
	}
	if endpoint == "" {
		endpoint = "NA"
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

	backupTime := "00:00"
	if len(inst.Backup.BackupSchedules) > 0 {
		backupTime = inst.Backup.BackupSchedules[0].BackupWndBgnTime
	}

	createdTime, _ := time.Parse(time.RFC3339, inst.CreatedYmdt)

	return irs.RDBMSInfo{
		IId: irs.IID{
			NameId:   inst.DBInstanceName,
			SystemId: inst.DBInstanceId,
		},
		VpcIID: irs.IID{NameId: ownerVPCName, SystemId: "NA"},

		DBEngine:        "mysql",
		DBEngineVersion: inst.DBVersion,
		DBInstanceSpec:  e.DBFlavorName,
		DBInstanceType:  "Primary",

		StorageType: storageType,
		StorageSize: strconv.Itoa(storageSize),

		SubnetIIDs: subnetIIDs,

		Port:     strconv.Itoa(inst.DBPort),
		Endpoint: endpoint,

		MasterUserName: master,
		DatabaseName:   "NA",

		HighAvailability: inst.UseHighAvailability,
		ReplicationType:  "NA",

		BackupRetentionDays: inst.Backup.BackupPeriod,
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
	if isRDSUUID(nameOrId) {
		return nameOrId, nil
	}
	var result nhnRDSFlavorListResponse
	if err := handler.getRDS(ctx, "/v3.0/db-flavors", &result); err != nil {
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
	var result nhnRDSParameterGroupListResponse
	if err := handler.getRDS(ctx, "/v3.0/parameter-groups", &result); err != nil {
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

// pollRDSJobSimple waits for a NHN RDS async job to reach a terminal state (no resource extraction).
func (handler *NhnCloudRDBMSHandler) pollRDSJobSimple(ctx context.Context, jobId string) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timed out waiting for NHN Cloud RDS job %s", jobId)
		case <-ticker.C:
			var jobResp nhnRDSJobResponse
			if err := handler.getRDS(ctx, "/v3.0/jobs/"+jobId, &jobResp); err != nil {
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

	var resp nhnRDSSchemaJobResponse
	if err := handler.postRDS(ctx, fmt.Sprintf("/v3.0/db-instances/%s/db-schemas", rdbmsSystemId),
		nhnRDSCreateSchemaRequest{DBSchemaName: dbName}, &resp); err != nil {
		return fmt.Errorf("NHN RDS CreateDatabase: %w", err)
	}
	if err := checkRDSResponseHeader(resp.Header); err != nil {
		return fmt.Errorf("NHN RDS CreateDatabase: %w", err)
	}
	return handler.pollRDSJobSimple(ctx, resp.JobId)
}

// ListDatabases lists DB schemas on the NHN Cloud RDS instance.
func (handler *NhnCloudRDBMSHandler) ListDatabases(rdbmsSystemId, dbEngine string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var resp nhnRDSListSchemasResponse
	if err := handler.getRDS(ctx, fmt.Sprintf("/v3.0/db-instances/%s/db-schemas", rdbmsSystemId), &resp); err != nil {
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

	// Find the schema ID by name.
	var listResp nhnRDSListSchemasResponse
	if err := handler.getRDS(ctx, fmt.Sprintf("/v3.0/db-instances/%s/db-schemas", rdbmsSystemId), &listResp); err != nil {
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
	if err := handler.deleteRDS(ctx, fmt.Sprintf("/v3.0/db-instances/%s/db-schemas/%s", rdbmsSystemId, schemaId), &resp); err != nil {
		return fmt.Errorf("NHN RDS DeleteDatabase: %w", err)
	}
	if err := checkRDSResponseHeader(resp.Header); err != nil {
		return fmt.Errorf("NHN RDS DeleteDatabase: %w", err)
	}
	return handler.pollRDSJobSimple(ctx, resp.JobId)
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
