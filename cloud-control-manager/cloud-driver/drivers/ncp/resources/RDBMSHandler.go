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
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vmysql"
	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vpostgresql"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpVpcRDBMSHandler struct {
	RegionInfo     idrv.RegionInfo
	MysqlClient    *vmysql.APIClient
	PostgresClient *vpostgresql.APIClient
}

// GetMetaInfo returns metadata about NCP RDBMS capabilities.
func (handler *NcpVpcRDBMSHandler) GetMetaInfo() (irs.RDBMSMetaInfo, error) {
	cblogger.Debug("NCP VPC RDBMSHandler GetMetaInfo() called")

	hiscallInfo := GetCallLogScheme(handler.RegionInfo.Region, call.RDBMS, "GetMetaInfo", "GetMetaInfo()")
	start := call.Start()

	metaInfo := irs.RDBMSMetaInfo{
		SupportedEngines: map[string][]string{
			"mysql":      {"8.0"},
			"postgresql": {"13", "14", "15"},
		},

		SupportsHighAvailability:   true,
		SupportsBackup:             true,
		SupportsPublicAccess:       false, // NCP DB instances are VPC-only
		SupportsDeletionProtection: true,  // MySQL only (IsDeleteProtection)
		SupportsEncryption:         true,

		StorageTypeOptions: map[string][]string{
			"mysql":      {"SSD"},
			"postgresql": {"SSD"},
		},
		StorageSizeRange: irs.StorageSizeRange{
			Min: 10,
			Max: 6000,
		},
	}

	LoggingInfo(hiscallInfo, start)
	return metaInfo, nil
}

// ListIID returns a list of all RDBMS instance IIDs (MySQL + PostgreSQL).
func (handler *NcpVpcRDBMSHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(handler.RegionInfo.Region, call.RDBMS, "ListIID", "GetCloudMysqlInstanceList()+GetCloudPostgresqlInstanceList()")
	start := call.Start()

	var iidList []*irs.IID

	// List MySQL instances
	mysqlResp, err := handler.MysqlClient.V2Api.GetCloudMysqlInstanceList(&vmysql.GetCloudMysqlInstanceListRequest{
		RegionCode: &handler.RegionInfo.Region,
	})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, fmt.Errorf("failed to list MySQL instances: %w", err)
	}
	for _, inst := range mysqlResp.CloudMysqlInstanceList {
		iidList = append(iidList, &irs.IID{
			NameId:   derefStr(inst.CloudMysqlServiceName),
			SystemId: derefStr(inst.CloudMysqlInstanceNo),
		})
	}

	// List PostgreSQL instances
	pgResp, err := handler.PostgresClient.V2Api.GetCloudPostgresqlInstanceList(&vpostgresql.GetCloudPostgresqlInstanceListRequest{
		RegionCode: &handler.RegionInfo.Region,
	})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, fmt.Errorf("failed to list PostgreSQL instances: %w", err)
	}
	for _, inst := range pgResp.CloudPostgresqlInstanceList {
		iidList = append(iidList, &irs.IID{
			NameId:   derefStr(inst.CloudPostgresqlServiceName),
			SystemId: derefStr(inst.CloudPostgresqlInstanceNo),
		})
	}

	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

// CreateRDBMS creates a new NCP managed database instance.
func (handler *NcpVpcRDBMSHandler) CreateRDBMS(rdbmsReqInfo irs.RDBMSInfo) (irs.RDBMSInfo, error) {
	engineType := strings.ToLower(rdbmsReqInfo.DBEngine)

	hiscallInfo := GetCallLogScheme(handler.RegionInfo.Region, call.RDBMS, rdbmsReqInfo.IId.NameId, "CreateRDBMS()")
	start := call.Start()

	// Validate required fields
	if rdbmsReqInfo.IId.NameId == "" {
		return irs.RDBMSInfo{}, errors.New("RDBMS instance name is required")
	}
	if rdbmsReqInfo.VpcIID.SystemId == "" {
		return irs.RDBMSInfo{}, errors.New("VpcIID.SystemId (VpcNo) is required")
	}
	if len(rdbmsReqInfo.SubnetIIDs) == 0 || rdbmsReqInfo.SubnetIIDs[0].SystemId == "" {
		return irs.RDBMSInfo{}, errors.New("at least one SubnetIID.SystemId (SubnetNo) is required")
	}
	if rdbmsReqInfo.MasterUserName == "" {
		return irs.RDBMSInfo{}, errors.New("MasterUserName is required")
	}
	if rdbmsReqInfo.MasterUserPassword == "" {
		return irs.RDBMSInfo{}, errors.New("MasterUserPassword is required")
	}

	switch engineType {
	case "mysql":
		return handler.createMysqlInstance(rdbmsReqInfo, hiscallInfo, start)
	case "postgresql":
		return handler.createPostgresqlInstance(rdbmsReqInfo, hiscallInfo, start)
	default:
		return irs.RDBMSInfo{}, fmt.Errorf("unsupported DBEngine '%s': NCP supports mysql and postgresql", rdbmsReqInfo.DBEngine)
	}
}

func (handler *NcpVpcRDBMSHandler) createMysqlInstance(reqInfo irs.RDBMSInfo, hiscallInfo call.CLOUDLOGSCHEMA, start time.Time) (irs.RDBMSInfo, error) {
	isHa := reqInfo.HighAvailability
	isBackup := true
	hostIp := "%"
	dbName := reqInfo.DatabaseName
	if dbName == "" {
		dbName = "mydb"
	}

	createReq := &vmysql.CreateCloudMysqlInstanceRequest{
		RegionCode:                 &handler.RegionInfo.Region,
		VpcNo:                      &reqInfo.VpcIID.SystemId,
		SubnetNo:                   &reqInfo.SubnetIIDs[0].SystemId,
		CloudMysqlServiceName:      &reqInfo.IId.NameId,
		CloudMysqlServerNamePrefix: &reqInfo.IId.NameId,
		CloudMysqlUserName:         &reqInfo.MasterUserName,
		CloudMysqlUserPassword:     &reqInfo.MasterUserPassword,
		HostIp:                     &hostIp,
		CloudMysqlDatabaseName:     &dbName,
		IsHa:                       &isHa,
		IsBackup:                   &isBackup,
	}

	if reqInfo.DBInstanceSpec != "" {
		createReq.CloudMysqlProductCode = &reqInfo.DBInstanceSpec
	}
	if reqInfo.DBEngineVersion != "" {
		createReq.EngineVersionCode = &reqInfo.DBEngineVersion
	}
	if reqInfo.StorageType != "" {
		createReq.DataStorageTypeCode = &reqInfo.StorageType
	}
	if reqInfo.Encryption {
		createReq.IsStorageEncryption = &reqInfo.Encryption
	}
	if reqInfo.DeletionProtection {
		createReq.IsDeleteProtection = &reqInfo.DeletionProtection
	}
	if reqInfo.BackupRetentionDays > 0 {
		period := int32(reqInfo.BackupRetentionDays)
		createReq.BackupFileRetentionPeriod = &period
	}
	if reqInfo.BackupTime != "" {
		createReq.BackupTime = &reqInfo.BackupTime
	}
	if reqInfo.Port != "" {
		port, err := strconv.Atoi(reqInfo.Port)
		if err == nil {
			p := int32(port)
			createReq.CloudMysqlPort = &p
		}
	}

	resp, err := handler.MysqlClient.V2Api.CreateCloudMysqlInstance(createReq)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, fmt.Errorf("failed to create MySQL instance: %w", err)
	}

	LoggingInfo(hiscallInfo, start)

	if len(resp.CloudMysqlInstanceList) == 0 {
		return irs.RDBMSInfo{}, errors.New("MySQL instance created but no instance returned in response")
	}

	return handler.convertMysqlInstanceToRDBMSInfo(resp.CloudMysqlInstanceList[0]), nil
}

func (handler *NcpVpcRDBMSHandler) createPostgresqlInstance(reqInfo irs.RDBMSInfo, hiscallInfo call.CLOUDLOGSCHEMA, start time.Time) (irs.RDBMSInfo, error) {
	isHa := reqInfo.HighAvailability
	isBackup := true
	clientCidr := "0.0.0.0/0"
	dbName := reqInfo.DatabaseName
	if dbName == "" {
		dbName = "mydb"
	}

	createReq := &vpostgresql.CreateCloudPostgresqlInstanceRequest{
		RegionCode:                      &handler.RegionInfo.Region,
		VpcNo:                           &reqInfo.VpcIID.SystemId,
		SubnetNo:                        &reqInfo.SubnetIIDs[0].SystemId,
		CloudPostgresqlServiceName:      &reqInfo.IId.NameId,
		CloudPostgresqlServerNamePrefix: &reqInfo.IId.NameId,
		CloudPostgresqlUserName:         &reqInfo.MasterUserName,
		CloudPostgresqlUserPassword:     &reqInfo.MasterUserPassword,
		ClientCidr:                      &clientCidr,
		CloudPostgresqlDatabaseName:     &dbName,
		IsHa:                            &isHa,
		IsBackup:                        &isBackup,
	}

	if reqInfo.DBInstanceSpec != "" {
		createReq.CloudPostgresqlProductCode = &reqInfo.DBInstanceSpec
	}
	if reqInfo.DBEngineVersion != "" {
		createReq.EngineVersionCode = &reqInfo.DBEngineVersion
	}
	if reqInfo.StorageType != "" {
		createReq.DataStorageTypeCode = &reqInfo.StorageType
	}
	if reqInfo.Encryption {
		createReq.IsStorageEncryption = &reqInfo.Encryption
	}
	if reqInfo.BackupRetentionDays > 0 {
		period := int32(reqInfo.BackupRetentionDays)
		createReq.BackupFileRetentionPeriod = &period
	}
	if reqInfo.BackupTime != "" {
		createReq.BackupTime = &reqInfo.BackupTime
	}
	if reqInfo.Port != "" {
		port, err := strconv.Atoi(reqInfo.Port)
		if err == nil {
			p := int32(port)
			createReq.CloudPostgresqlPort = &p
		}
	}

	resp, err := handler.PostgresClient.V2Api.CreateCloudPostgresqlInstance(createReq)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, fmt.Errorf("failed to create PostgreSQL instance: %w", err)
	}

	LoggingInfo(hiscallInfo, start)

	if len(resp.CloudPostgresqlInstanceList) == 0 {
		return irs.RDBMSInfo{}, errors.New("PostgreSQL instance created but no instance returned in response")
	}

	return handler.convertPostgresqlInstanceToRDBMSInfo(resp.CloudPostgresqlInstanceList[0]), nil
}

// ListRDBMS returns a list of all RDBMS instances (MySQL + PostgreSQL).
func (handler *NcpVpcRDBMSHandler) ListRDBMS() ([]*irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.RegionInfo.Region, call.RDBMS, "ListRDBMS", "GetCloudMysqlInstanceList()+GetCloudPostgresqlInstanceList()")
	start := call.Start()

	var rdbmsList []*irs.RDBMSInfo

	// List MySQL instances
	mysqlResp, err := handler.MysqlClient.V2Api.GetCloudMysqlInstanceList(&vmysql.GetCloudMysqlInstanceListRequest{
		RegionCode: &handler.RegionInfo.Region,
	})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, fmt.Errorf("failed to list MySQL instances: %w", err)
	}
	for _, inst := range mysqlResp.CloudMysqlInstanceList {
		info := handler.convertMysqlInstanceToRDBMSInfo(inst)
		rdbmsList = append(rdbmsList, &info)
	}

	// List PostgreSQL instances
	pgResp, err := handler.PostgresClient.V2Api.GetCloudPostgresqlInstanceList(&vpostgresql.GetCloudPostgresqlInstanceListRequest{
		RegionCode: &handler.RegionInfo.Region,
	})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, fmt.Errorf("failed to list PostgreSQL instances: %w", err)
	}
	for _, inst := range pgResp.CloudPostgresqlInstanceList {
		info := handler.convertPostgresqlInstanceToRDBMSInfo(inst)
		rdbmsList = append(rdbmsList, &info)
	}

	LoggingInfo(hiscallInfo, start)
	return rdbmsList, nil
}

// GetRDBMS retrieves a specific RDBMS instance by IID.
func (handler *NcpVpcRDBMSHandler) GetRDBMS(rdbmsIID irs.IID) (irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.RegionInfo.Region, call.RDBMS, rdbmsIID.NameId, "GetRDBMS()")
	start := call.Start()

	// Try MySQL first
	if rdbmsIID.SystemId != "" {
		mysqlResp, err := handler.MysqlClient.V2Api.GetCloudMysqlInstanceDetail(&vmysql.GetCloudMysqlInstanceDetailRequest{
			RegionCode:           &handler.RegionInfo.Region,
			CloudMysqlInstanceNo: &rdbmsIID.SystemId,
		})
		if err == nil && len(mysqlResp.CloudMysqlInstanceList) > 0 {
			LoggingInfo(hiscallInfo, start)
			return handler.convertMysqlInstanceToRDBMSInfo(mysqlResp.CloudMysqlInstanceList[0]), nil
		}

		// Try PostgreSQL
		pgResp, err := handler.PostgresClient.V2Api.GetCloudPostgresqlInstanceDetail(&vpostgresql.GetCloudPostgresqlInstanceDetailRequest{
			RegionCode:                &handler.RegionInfo.Region,
			CloudPostgresqlInstanceNo: &rdbmsIID.SystemId,
		})
		if err == nil && len(pgResp.CloudPostgresqlInstanceList) > 0 {
			LoggingInfo(hiscallInfo, start)
			return handler.convertPostgresqlInstanceToRDBMSInfo(pgResp.CloudPostgresqlInstanceList[0]), nil
		}

		notFoundErr := fmt.Errorf("RDBMS instance with SystemId '%s' not found", rdbmsIID.SystemId)
		LoggingError(hiscallInfo, notFoundErr)
		return irs.RDBMSInfo{}, notFoundErr
	}

	// Search by NameId
	if rdbmsIID.NameId != "" {
		// Search MySQL
		mysqlResp, err := handler.MysqlClient.V2Api.GetCloudMysqlInstanceList(&vmysql.GetCloudMysqlInstanceListRequest{
			RegionCode:            &handler.RegionInfo.Region,
			CloudMysqlServiceName: &rdbmsIID.NameId,
		})
		if err == nil && len(mysqlResp.CloudMysqlInstanceList) > 0 {
			LoggingInfo(hiscallInfo, start)
			return handler.convertMysqlInstanceToRDBMSInfo(mysqlResp.CloudMysqlInstanceList[0]), nil
		}

		// Search PostgreSQL
		pgResp, err := handler.PostgresClient.V2Api.GetCloudPostgresqlInstanceList(&vpostgresql.GetCloudPostgresqlInstanceListRequest{
			RegionCode:                 &handler.RegionInfo.Region,
			CloudPostgresqlServiceName: &rdbmsIID.NameId,
		})
		if err == nil && len(pgResp.CloudPostgresqlInstanceList) > 0 {
			LoggingInfo(hiscallInfo, start)
			return handler.convertPostgresqlInstanceToRDBMSInfo(pgResp.CloudPostgresqlInstanceList[0]), nil
		}
	}

	notFoundErr := fmt.Errorf("RDBMS instance '%s/%s' not found", rdbmsIID.NameId, rdbmsIID.SystemId)
	LoggingError(hiscallInfo, notFoundErr)
	return irs.RDBMSInfo{}, notFoundErr
}

// DeleteRDBMS deletes an NCP managed database instance.
func (handler *NcpVpcRDBMSHandler) DeleteRDBMS(rdbmsIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(handler.RegionInfo.Region, call.RDBMS, rdbmsIID.NameId, "DeleteRDBMS()")
	start := call.Start()

	// Need to find the instance first to determine engine type
	info, err := handler.GetRDBMS(rdbmsIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, fmt.Errorf("failed to find RDBMS instance for deletion: %w", err)
	}

	engineType := strings.ToLower(info.DBEngine)
	systemID := info.IId.SystemId

	switch {
	case strings.Contains(engineType, "mysql"):
		_, err = handler.MysqlClient.V2Api.DeleteCloudMysqlInstance(&vmysql.DeleteCloudMysqlInstanceRequest{
			RegionCode:           &handler.RegionInfo.Region,
			CloudMysqlInstanceNo: &systemID,
		})
	case strings.Contains(engineType, "postgresql"):
		_, err = handler.PostgresClient.V2Api.DeleteCloudPostgresqlInstance(&vpostgresql.DeleteCloudPostgresqlInstanceRequest{
			RegionCode:                &handler.RegionInfo.Region,
			CloudPostgresqlInstanceNo: &systemID,
		})
	default:
		return false, fmt.Errorf("unknown engine type '%s' for instance '%s'", engineType, systemID)
	}

	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, fmt.Errorf("failed to delete RDBMS instance '%s': %w", systemID, err)
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

// ---- Helper functions ----

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefBool(b *bool) bool {
	if b == nil {
		return false
	}
	return *b
}

func derefInt32(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}

func derefInt64(i *int64) int64 {
	if i == nil {
		return 0
	}
	return *i
}

func (handler *NcpVpcRDBMSHandler) convertMysqlInstanceToRDBMSInfo(inst *vmysql.CloudMysqlInstance) irs.RDBMSInfo {
	info := irs.RDBMSInfo{
		IId: irs.IID{
			NameId:   derefStr(inst.CloudMysqlServiceName),
			SystemId: derefStr(inst.CloudMysqlInstanceNo),
		},

		DBEngine:        "mysql",
		DBEngineVersion: derefStr(inst.EngineVersion),

		HighAvailability: derefBool(inst.IsHa),
		Encryption:       false,

		BackupRetentionDays: int(derefInt32(inst.BackupFileRetentionPeriod)),
		BackupTime:          derefStr(inst.BackupTime),

		Port:               strconv.Itoa(int(derefInt32(inst.CloudMysqlPort))),
		MasterUserName:     "NA", // Not exposed in list/detail API
		DatabaseName:       "NA", // Not exposed in list/detail API
		DeletionProtection: false,
		PublicAccess:       false,
		ReplicationType:    "async",

		Status: convertNcpMysqlStatusToRDBMSStatus(inst.CloudMysqlInstanceStatusName),
	}

	if inst.CreateDate != nil {
		t, err := time.Parse("2006-01-02T15:04:05+0900", *inst.CreateDate)
		if err == nil {
			info.CreatedTime = t
		}
	}

	// Extract details from first server instance
	if len(inst.CloudMysqlServerInstanceList) > 0 {
		serverInst := inst.CloudMysqlServerInstanceList[0]
		info.VpcIID = irs.IID{NameId: "NA", SystemId: derefStr(serverInst.VpcNo)}
		info.DBInstanceSpec = derefStr(serverInst.CloudMysqlProductCode)
		info.Endpoint = derefStr(serverInst.PrivateDomain)
		info.Encryption = derefBool(serverInst.IsStorageEncryption)

		storageSizeBytes := derefInt64(serverInst.DataStorageSize)
		info.StorageSize = strconv.FormatInt(storageSizeBytes/(1024*1024*1024), 10)

		if serverInst.DataStorageType != nil && serverInst.DataStorageType.CodeName != nil {
			info.StorageType = *serverInst.DataStorageType.CodeName
		} else {
			info.StorageType = "NA"
		}

		if len(inst.CloudMysqlServerInstanceList) > 0 {
			var subnetIIDs []irs.IID
			for _, si := range inst.CloudMysqlServerInstanceList {
				subnetIIDs = append(subnetIIDs, irs.IID{NameId: "NA", SystemId: derefStr(si.SubnetNo)})
			}
			info.SubnetIIDs = subnetIIDs
		}

		if serverInst.CloudMysqlServerRole != nil && serverInst.CloudMysqlServerRole.CodeName != nil {
			info.DBInstanceType = *serverInst.CloudMysqlServerRole.CodeName
		}
	}

	return info
}

func (handler *NcpVpcRDBMSHandler) convertPostgresqlInstanceToRDBMSInfo(inst *vpostgresql.CloudPostgresqlInstance) irs.RDBMSInfo {
	info := irs.RDBMSInfo{
		IId: irs.IID{
			NameId:   derefStr(inst.CloudPostgresqlServiceName),
			SystemId: derefStr(inst.CloudPostgresqlInstanceNo),
		},

		DBEngine:        "postgresql",
		DBEngineVersion: derefStr(inst.EngineVersion),

		HighAvailability: derefBool(inst.IsHa),
		Encryption:       false,

		BackupRetentionDays: int(derefInt32(inst.BackupFileRetentionPeriod)),
		BackupTime:          derefStr(inst.BackupTime),

		Port:               strconv.Itoa(int(derefInt32(inst.CloudPostgresqlPort))),
		MasterUserName:     "NA",
		DatabaseName:       "NA",
		DeletionProtection: false, // PostgreSQL does not support deletion protection
		PublicAccess:       false,
		ReplicationType:    "async",

		Status: convertNcpPostgresqlStatusToRDBMSStatus(inst.CloudPostgresqlInstanceStatusName),
	}

	if inst.CreateDate != nil {
		t, err := time.Parse("2006-01-02T15:04:05+0900", *inst.CreateDate)
		if err == nil {
			info.CreatedTime = t
		}
	}

	// Extract details from first server instance
	if len(inst.CloudPostgresqlServerInstanceList) > 0 {
		serverInst := inst.CloudPostgresqlServerInstanceList[0]
		info.VpcIID = irs.IID{NameId: "NA", SystemId: derefStr(serverInst.VpcNo)}
		info.DBInstanceSpec = derefStr(serverInst.CloudPostgresqlProductCode)
		info.Endpoint = derefStr(serverInst.PrivateDomain)
		info.Encryption = derefBool(serverInst.IsStorageEncryption)

		storageSizeBytes := derefInt64(serverInst.DataStorageSize)
		info.StorageSize = strconv.FormatInt(storageSizeBytes/(1024*1024*1024), 10)

		if serverInst.DataStorageType != nil && serverInst.DataStorageType.CodeName != nil {
			info.StorageType = *serverInst.DataStorageType.CodeName
		} else {
			info.StorageType = "NA"
		}

		if len(inst.CloudPostgresqlServerInstanceList) > 0 {
			var subnetIIDs []irs.IID
			for _, si := range inst.CloudPostgresqlServerInstanceList {
				subnetIIDs = append(subnetIIDs, irs.IID{NameId: "NA", SystemId: derefStr(si.SubnetNo)})
			}
			info.SubnetIIDs = subnetIIDs
		}

		if serverInst.CloudPostgresqlServerRole != nil && serverInst.CloudPostgresqlServerRole.CodeName != nil {
			info.DBInstanceType = *serverInst.CloudPostgresqlServerRole.CodeName
		}
	}

	return info
}

func convertNcpMysqlStatusToRDBMSStatus(statusName *string) irs.RDBMSStatus {
	if statusName == nil {
		return irs.RDBMSError
	}
	switch strings.ToLower(*statusName) {
	case "creating", "setting up", "configuring":
		return irs.RDBMSCreating
	case "running":
		return irs.RDBMSAvailable
	case "deleting":
		return irs.RDBMSDeleting
	case "stopped", "shutting down":
		return irs.RDBMSStopped
	default:
		return irs.RDBMSError
	}
}

func convertNcpPostgresqlStatusToRDBMSStatus(statusName *string) irs.RDBMSStatus {
	if statusName == nil {
		return irs.RDBMSError
	}
	switch strings.ToLower(*statusName) {
	case "creating", "setting up", "configuring":
		return irs.RDBMSCreating
	case "running":
		return irs.RDBMSAvailable
	case "deleting":
		return irs.RDBMSDeleting
	case "stopped", "shutting down":
		return irs.RDBMSStopped
	default:
		return irs.RDBMSError
	}
}
