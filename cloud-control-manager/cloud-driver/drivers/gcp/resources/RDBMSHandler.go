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

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	sqladmin "google.golang.org/api/sqladmin/v1beta4"
)

type GCPRDBMSHandler struct {
	Region     idrv.RegionInfo
	Credential idrv.CredentialInfo
	Client     *sqladmin.Service
}

func (handler *GCPRDBMSHandler) GetMetaInfo() (irs.RDBMSMetaInfo, error) {
	cblogger.Debug("GCP Cloud SQL GetMetaInfo() called")

	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "GetMetaInfo", "GetMetaInfo()")
	start := call.Start()

	metaInfo := irs.RDBMSMetaInfo{
		SupportedEngines: map[string][]string{
			"mysql":      {"5.7", "8.0", "8.4"},
			"postgresql": {"13", "14", "15", "16", "17"},
		},
		SupportsHighAvailability:   true,
		SupportsBackup:             true,
		SupportsPublicAccess:       true,
		SupportsDeletionProtection: true,
		SupportsEncryption:         true, // CMEK supported
		StorageTypeOptions: map[string][]string{
			"mysql":      {"PD_SSD", "PD_HDD"},
			"postgresql": {"PD_SSD", "PD_HDD"},
		},
		StorageSizeRange: irs.StorageSizeRange{
			Min: 10,
			Max: 65536,
		},
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return metaInfo, nil
}

func (handler *GCPRDBMSHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "ListIID", "Instances.List()")
	start := call.Start()

	projectId := handler.Credential.ClientEmail
	// GCP project ID is typically extracted from credential info
	// In CB-Spider, the ProjectID is often stored in the credential
	projectId = handler.getProjectId()

	resp, err := handler.Client.Instances.List(projectId).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	calllogger.Info(call.String(hiscallInfo))

	var iidList []*irs.IID
	for _, instance := range resp.Items {
		iid := &irs.IID{
			NameId:   instance.Name,
			SystemId: instance.Name,
		}
		iidList = append(iidList, iid)
	}

	return iidList, nil
}

func (handler *GCPRDBMSHandler) CreateRDBMS(rdbmsReqInfo irs.RDBMSInfo) (irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, rdbmsReqInfo.IId.NameId, "Instances.Insert()")
	start := call.Start()

	// Validate required fields
	if rdbmsReqInfo.IId.NameId == "" {
		return irs.RDBMSInfo{}, errors.New("RDBMS NameId is required")
	}
	if rdbmsReqInfo.DBEngine == "" {
		return irs.RDBMSInfo{}, errors.New("DBEngine is required")
	}
	if rdbmsReqInfo.DBEngineVersion == "" {
		return irs.RDBMSInfo{}, errors.New("DBEngineVersion is required")
	}
	if rdbmsReqInfo.DBInstanceSpec == "" {
		return irs.RDBMSInfo{}, errors.New("DBInstanceSpec is required")
	}
	if rdbmsReqInfo.MasterUserName == "" {
		return irs.RDBMSInfo{}, errors.New("MasterUserName is required")
	}
	if rdbmsReqInfo.MasterUserPassword == "" {
		return irs.RDBMSInfo{}, errors.New("MasterUserPassword is required")
	}
	if rdbmsReqInfo.StorageSize == "" {
		return irs.RDBMSInfo{}, errors.New("StorageSize is required")
	}

	storageSizeGB, err := strconv.ParseInt(rdbmsReqInfo.StorageSize, 10, 64)
	if err != nil {
		return irs.RDBMSInfo{}, fmt.Errorf("invalid StorageSize: %s", rdbmsReqInfo.StorageSize)
	}

	projectId := handler.getProjectId()

	// Map engine to GCP database version format
	databaseVersion := handler.mapDatabaseVersion(rdbmsReqInfo.DBEngine, rdbmsReqInfo.DBEngineVersion)

	// Build settings
	settings := &sqladmin.Settings{
		Tier:           rdbmsReqInfo.DBInstanceSpec, // e.g., "db-custom-2-7680"
		DataDiskSizeGb: storageSizeGB,
	}

	// Storage Type
	if rdbmsReqInfo.StorageType != "" {
		settings.DataDiskType = rdbmsReqInfo.StorageType
	}

	// High Availability
	if rdbmsReqInfo.HighAvailability {
		settings.AvailabilityType = "REGIONAL"
	} else {
		settings.AvailabilityType = "ZONAL"
	}

	// Backup Configuration
	backupConfig := &sqladmin.BackupConfiguration{
		Enabled: true,
	}
	if rdbmsReqInfo.BackupRetentionDays > 0 {
		backupConfig.TransactionLogRetentionDays = int64(rdbmsReqInfo.BackupRetentionDays)
	}
	if rdbmsReqInfo.BackupTime != "" {
		backupConfig.StartTime = rdbmsReqInfo.BackupTime
	}
	settings.BackupConfiguration = backupConfig

	// IP Configuration (Public Access, Network)
	ipConfig := &sqladmin.IpConfiguration{}
	if rdbmsReqInfo.PublicAccess {
		ipConfig.Ipv4Enabled = true
		// Allow all external access (like Azure's AllowAll firewall rule)
		ipConfig.AuthorizedNetworks = []*sqladmin.AclEntry{
			{
				Value: "0.0.0.0/0",
				Name:  "AllowAll",
			},
		}
	} else {
		ipConfig.Ipv4Enabled = false
	}

	// VPC Network - GCP uses private service connection
	if rdbmsReqInfo.VpcIID.SystemId != "" {
		ipConfig.PrivateNetwork = rdbmsReqInfo.VpcIID.SystemId
	}
	settings.IpConfiguration = ipConfig

	// Deletion Protection
	settings.DeletionProtectionEnabled = rdbmsReqInfo.DeletionProtection

	// Build the instance
	dbInstance := &sqladmin.DatabaseInstance{
		Name:            rdbmsReqInfo.IId.NameId,
		DatabaseVersion: databaseVersion,
		Settings:        settings,
		Region:          handler.Region.Region,
		RootPassword:    rdbmsReqInfo.MasterUserPassword,
	}

	// User labels (tags)
	if len(rdbmsReqInfo.TagList) > 0 {
		labels := make(map[string]string)
		for _, tag := range rdbmsReqInfo.TagList {
			labels[strings.ToLower(tag.Key)] = strings.ToLower(tag.Value)
		}
		settings.UserLabels = labels
	}

	// Create the instance
	op, err := handler.Client.Instances.Insert(projectId, dbInstance).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	// Wait for operation completion
	err = handler.waitForOperation(projectId, op.Name)
	if err != nil {
		return irs.RDBMSInfo{}, fmt.Errorf("failed waiting for instance creation: %w", err)
	}

	// Set master user password via Users API
	user := &sqladmin.User{
		Name:     rdbmsReqInfo.MasterUserName,
		Password: rdbmsReqInfo.MasterUserPassword,
	}
	_, err = handler.Client.Users.Insert(projectId, rdbmsReqInfo.IId.NameId, user).Do()
	if err != nil {
		cblogger.Warnf("Failed to create master user: %v", err)
	}

	// Set initial database if specified
	if rdbmsReqInfo.DatabaseName != "" {
		db := &sqladmin.Database{
			Name: rdbmsReqInfo.DatabaseName,
		}
		_, err = handler.Client.Databases.Insert(projectId, rdbmsReqInfo.IId.NameId, db).Do()
		if err != nil {
			cblogger.Warnf("Failed to create initial database: %v", err)
		}
	}

	// Get the created instance
	return handler.GetRDBMS(irs.IID{NameId: rdbmsReqInfo.IId.NameId, SystemId: rdbmsReqInfo.IId.NameId})
}

func (handler *GCPRDBMSHandler) ListRDBMS() ([]*irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "ListRDBMS", "Instances.List()")
	start := call.Start()

	projectId := handler.getProjectId()
	resp, err := handler.Client.Instances.List(projectId).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	calllogger.Info(call.String(hiscallInfo))

	var rdbmsList []*irs.RDBMSInfo
	for _, instance := range resp.Items {
		rdbmsInfo := handler.convertToRDBMSInfo(instance)
		rdbmsList = append(rdbmsList, &rdbmsInfo)
	}

	return rdbmsList, nil
}

func (handler *GCPRDBMSHandler) GetRDBMS(rdbmsIID irs.IID) (irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, rdbmsIID.NameId, "Instances.Get()")
	start := call.Start()

	projectId := handler.getProjectId()
	instance, err := handler.Client.Instances.Get(projectId, rdbmsIID.SystemId).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	return handler.convertToRDBMSInfo(instance), nil
}

func (handler *GCPRDBMSHandler) DeleteRDBMS(rdbmsIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, rdbmsIID.NameId, "Instances.Delete()")
	start := call.Start()

	projectId := handler.getProjectId()

	// Check and disable deletion protection if needed
	instance, err := handler.Client.Instances.Get(projectId, rdbmsIID.SystemId).Do()
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	if instance.Settings != nil && instance.Settings.DeletionProtectionEnabled {
		instance.Settings.DeletionProtectionEnabled = false
		_, err = handler.Client.Instances.Update(projectId, rdbmsIID.SystemId, instance).Do()
		if err != nil {
			return false, fmt.Errorf("failed to disable deletion protection: %w", err)
		}
	}

	op, err := handler.Client.Instances.Delete(projectId, rdbmsIID.SystemId).Do()
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	calllogger.Info(call.String(hiscallInfo))

	err = handler.waitForOperation(projectId, op.Name)
	if err != nil {
		return false, fmt.Errorf("failed waiting for instance deletion: %w", err)
	}

	return true, nil
}

// ===== Helper Functions =====

func (handler *GCPRDBMSHandler) getProjectId() string {
	return handler.Credential.ProjectID
}

func (handler *GCPRDBMSHandler) mapDatabaseVersion(engine, version string) string {
	switch strings.ToLower(engine) {
	case "mysql":
		return "MYSQL_" + strings.ReplaceAll(version, ".", "_")
	case "postgresql":
		return "POSTGRES_" + strings.Split(version, ".")[0]
	default:
		return strings.ToUpper(engine) + "_" + strings.ReplaceAll(version, ".", "_")
	}
}

func (handler *GCPRDBMSHandler) waitForOperation(projectId, opName string) error {
	maxWait := 30 * time.Minute
	pollInterval := 15 * time.Second
	deadline := time.Now().Add(maxWait)

	for time.Now().Before(deadline) {
		op, err := handler.Client.Operations.Get(projectId, opName).Do()
		if err != nil {
			return err
		}
		if op.Status == "DONE" {
			if op.Error != nil && len(op.Error.Errors) > 0 {
				return fmt.Errorf("operation failed: %s", op.Error.Errors[0].Message)
			}
			return nil
		}
		time.Sleep(pollInterval)
	}
	return fmt.Errorf("timeout waiting for operation %s", opName)
}

func (handler *GCPRDBMSHandler) convertToRDBMSInfo(instance *sqladmin.DatabaseInstance) irs.RDBMSInfo {
	rdbmsInfo := irs.RDBMSInfo{}

	rdbmsInfo.IId = irs.IID{
		NameId:   instance.Name,
		SystemId: instance.Name,
	}

	// Parse engine and version from DatabaseVersion (e.g., "MYSQL_8_0", "POSTGRES_15")
	engine, version := handler.parseDatabaseVersion(instance.DatabaseVersion)
	rdbmsInfo.DBEngine = engine
	rdbmsInfo.DBEngineVersion = version

	// Instance Spec
	if instance.Settings != nil {
		rdbmsInfo.DBInstanceSpec = instance.Settings.Tier

		// Storage
		rdbmsInfo.StorageSize = strconv.FormatInt(instance.Settings.DataDiskSizeGb, 10)
		rdbmsInfo.StorageType = instance.Settings.DataDiskType

		// High Availability
		rdbmsInfo.HighAvailability = (instance.Settings.AvailabilityType == "REGIONAL")
		if rdbmsInfo.HighAvailability {
			rdbmsInfo.DBInstanceType = "REGIONAL"
			rdbmsInfo.ReplicationType = "sync"
		} else {
			rdbmsInfo.DBInstanceType = "ZONAL"
		}

		// Backup
		if instance.Settings.BackupConfiguration != nil {
			if instance.Settings.BackupConfiguration.Enabled {
				rdbmsInfo.BackupRetentionDays = int(instance.Settings.BackupConfiguration.TransactionLogRetentionDays)
				rdbmsInfo.BackupTime = instance.Settings.BackupConfiguration.StartTime
			}
		}

		// IP Configuration
		if instance.Settings.IpConfiguration != nil {
			rdbmsInfo.PublicAccess = instance.Settings.IpConfiguration.Ipv4Enabled
			// VPC
			if instance.Settings.IpConfiguration.PrivateNetwork != "" {
				rdbmsInfo.VpcIID = irs.IID{
					SystemId: instance.Settings.IpConfiguration.PrivateNetwork,
				}
			}
		}

		// Deletion Protection
		rdbmsInfo.DeletionProtection = instance.Settings.DeletionProtectionEnabled

		// Tags (labels)
		for k, v := range instance.Settings.UserLabels {
			rdbmsInfo.TagList = append(rdbmsInfo.TagList, irs.KeyValue{Key: k, Value: v})
		}
	}

	// Authentication - retrieve master user from Users API
	rdbmsInfo.MasterUserName = "" // default empty
	projectId := handler.getProjectId()
	userListResp, userErr := handler.Client.Users.List(projectId, instance.Name).Do()
	if userErr == nil && userListResp != nil {
		for _, u := range userListResp.Items {
			// Skip built-in system accounts
			if u.Name != "" && u.Name != "root" && u.Name != "postgres" &&
				u.Name != "cloudsqlsuperuser" && u.Name != "cloudsqladmin" &&
				u.Name != "cloudsqlreplica" {
				rdbmsInfo.MasterUserName = u.Name
				break
			}
		}
	}

	// Endpoint
	if len(instance.IpAddresses) > 0 {
		for _, ip := range instance.IpAddresses {
			if ip.Type == "PRIMARY" || ip.Type == "PRIVATE" {
				rdbmsInfo.Endpoint = ip.IpAddress
				break
			}
		}
	}

	// Port - GCP uses standard ports
	switch strings.ToLower(engine) {
	case "mysql":
		rdbmsInfo.Port = "3306"
	case "postgresql":
		rdbmsInfo.Port = "5432"
	}

	// Encryption - GCP encrypts by default
	rdbmsInfo.Encryption = true

	// Status
	rdbmsInfo.Status = convertGCPStatusToRDBMSStatus(instance.State)

	// Created Time
	if instance.CreateTime != "" {
		t, err := time.Parse(time.RFC3339, instance.CreateTime)
		if err == nil {
			rdbmsInfo.CreatedTime = t
		}
	}

	// SecurityGroupIIDs - Not applicable for GCP Cloud SQL
	// SubnetIIDs - Not applicable for GCP Cloud SQL (uses VPC peering)

	// KeyValueList
	rdbmsInfo.KeyValueList = irs.StructToKeyValueList(instance)

	return rdbmsInfo
}

func (handler *GCPRDBMSHandler) parseDatabaseVersion(dbVersion string) (string, string) {
	// e.g., "MYSQL_8_0" -> "mysql", "8.0"
	// e.g., "POSTGRES_15" -> "postgresql", "15"
	parts := strings.SplitN(dbVersion, "_", 2)
	if len(parts) < 2 {
		return strings.ToLower(dbVersion), ""
	}
	engine := strings.ToLower(parts[0])
	if engine == "postgres" {
		engine = "postgresql"
	}
	version := strings.ReplaceAll(parts[1], "_", ".")
	return engine, version
}

func convertGCPStatusToRDBMSStatus(state string) irs.RDBMSStatus {
	switch strings.ToUpper(state) {
	case "RUNNABLE":
		return irs.RDBMSAvailable
	case "PENDING_CREATE", "MAINTENANCE":
		return irs.RDBMSCreating
	case "SUSPENDED", "STOPPED":
		return irs.RDBMSStopped
	case "PENDING_DELETE":
		return irs.RDBMSDeleting
	case "FAILED":
		return irs.RDBMSError
	default:
		return irs.RDBMSError
	}
}
