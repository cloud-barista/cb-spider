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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
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

const gcpCloudSQLDiscoveryURL = "https://sqladmin.googleapis.com/$discovery/rest?version=v1beta4"

func (handler *GCPRDBMSHandler) GetMetaInfo(dbEngine string) (irs.RDBMSMetaInfo, error) {
	cblogger.Debug("GCP Cloud SQL GetMetaInfo() called")

	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "GetMetaInfo", "CloudSQLDiscovery.DatabaseInstance.databaseVersion,Tiers.List()")
	start := call.Start()
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return irs.RDBMSMetaInfo{}, err
	}

	supportedEngines, storageTypeOptions, err := handler.fetchCloudSQLMetaOptions()
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSMetaInfo{}, fmt.Errorf("fetch Cloud SQL meta options failed: %w", err)
	}
	instanceSpecOptions, storageSizeRange, err := handler.fetchCloudSQLInstanceOptions()
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSMetaInfo{}, fmt.Errorf("fetch Cloud SQL instance options failed: %w", err)
	}

	metaInfo, err := irs.BuildRDBMSMetaInfo(requestedEngine, supportedEngines, instanceSpecOptions, storageTypeOptions, storageSizeRange, true, true, true, true, true, "1-7", false, false)
	if err != nil {
		return irs.RDBMSMetaInfo{}, err
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return metaInfo, nil
}

func (handler *GCPRDBMSHandler) fetchCloudSQLMetaOptions() (map[string][]string, map[string][]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, gcpCloudSQLDiscoveryURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build Cloud SQL discovery request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("Cloud SQL discovery request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("Cloud SQL discovery API returned status %d", resp.StatusCode)
	}

	var discovery struct {
		Schemas struct {
			DatabaseInstance struct {
				Properties struct {
					DatabaseVersion struct {
						Enum []string `json:"enum"`
					} `json:"databaseVersion"`
				} `json:"properties"`
			} `json:"DatabaseInstance"`
			Settings struct {
				Properties struct {
					DataDiskType struct {
						Enum           []string `json:"enum"`
						EnumDeprecated []bool   `json:"enumDeprecated"`
					} `json:"dataDiskType"`
				} `json:"properties"`
			} `json:"Settings"`
		} `json:"schemas"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&discovery); err != nil {
		return nil, nil, fmt.Errorf("failed to parse Cloud SQL discovery response: %w", err)
	}

	supportedEngines := map[string][]string{
		"mysql":      {},
		"postgresql": {},
	}
	versionSets := map[string]map[string]bool{
		"mysql":      {},
		"postgresql": {},
	}

	for _, dbVersion := range discovery.Schemas.DatabaseInstance.Properties.DatabaseVersion.Enum {
		engine, version := cloudSQLDatabaseVersionToEngineVersion(dbVersion)
		if engine == "" || version == "" {
			continue
		}
		versionSets[engine][version] = true
	}

	for engine, versionSet := range versionSets {
		for version := range versionSet {
			supportedEngines[engine] = append(supportedEngines[engine], version)
		}
		sort.Slice(supportedEngines[engine], func(leftIndex, rightIndex int) bool {
			return compareVersionStrings(supportedEngines[engine][leftIndex], supportedEngines[engine][rightIndex]) < 0
		})
	}

	if len(supportedEngines["mysql"]) == 0 || len(supportedEngines["postgresql"]) == 0 {
		return nil, nil, fmt.Errorf("Cloud SQL discovery response did not include required MySQL/PostgreSQL versions")
	}

	storageTypes := cloudSQLStorageTypesFromDiscovery(discovery.Schemas.Settings.Properties.DataDiskType.Enum, discovery.Schemas.Settings.Properties.DataDiskType.EnumDeprecated)
	if len(storageTypes) == 0 {
		return nil, nil, fmt.Errorf("Cloud SQL discovery response did not include supported storage types")
	}

	storageTypeOptions := map[string][]string{
		"mysql":      append([]string(nil), storageTypes...),
		"postgresql": append([]string(nil), storageTypes...),
	}

	return supportedEngines, storageTypeOptions, nil
}

func (handler *GCPRDBMSHandler) fetchCloudSQLInstanceOptions() (map[string][]string, irs.StorageSizeRange, error) {
	projectID := handler.getProjectId()
	if projectID == "" {
		return nil, irs.StorageSizeRange{}, fmt.Errorf("GCP project ID is empty")
	}

	resp, err := handler.Client.Tiers.List(projectID).Do()
	if err != nil {
		return nil, irs.StorageSizeRange{}, fmt.Errorf("Cloud SQL Tiers.List failed: %w", err)
	}

	const bytesPerGiB = int64(1024 * 1024 * 1024)
	maxStorageGB := int64(0)
	tierSet := map[string]bool{}
	for _, tier := range resp.Items {
		if tier == nil || tier.DiskQuota <= 0 {
			continue
		}
		if !cloudSQLTierSupportsRegion(tier, handler.Region.Region) {
			continue
		}
		if tier.Tier != "" {
			tierSet[tier.Tier] = true
		}
		storageGB := tier.DiskQuota / bytesPerGiB
		if storageGB > maxStorageGB {
			maxStorageGB = storageGB
		}
	}
	if maxStorageGB == 0 {
		return nil, irs.StorageSizeRange{}, fmt.Errorf("Cloud SQL Tiers.List returned no disk quota for region %s", handler.Region.Region)
	}

	tierList := make([]string, 0, len(tierSet))
	for tier := range tierSet {
		tierList = append(tierList, tier)
	}
	sort.Strings(tierList)

	instanceSpecOptions := map[string][]string{
		"mysql":      append([]string(nil), tierList...),
		"postgresql": append([]string(nil), tierList...),
	}

	return instanceSpecOptions, irs.StorageSizeRange{Min: 10, Max: maxStorageGB}, nil
}

func cloudSQLTierSupportsRegion(tier *sqladmin.Tier, region string) bool {
	if region == "" || len(tier.Region) == 0 {
		return true
	}
	for _, tierRegion := range tier.Region {
		if tierRegion == region {
			return true
		}
	}
	return false
}

func cloudSQLDatabaseVersionToEngineVersion(dbVersion string) (string, string) {
	switch {
	case strings.HasPrefix(dbVersion, "MYSQL_"):
		return "mysql", strings.ReplaceAll(strings.TrimPrefix(dbVersion, "MYSQL_"), "_", ".")
	case strings.HasPrefix(dbVersion, "POSTGRES_"):
		return "postgresql", strings.ReplaceAll(strings.TrimPrefix(dbVersion, "POSTGRES_"), "_", ".")
	default:
		return "", ""
	}
}

func cloudSQLStorageTypesFromDiscovery(enumValues []string, enumDeprecated []bool) []string {
	storageTypes := make([]string, 0, len(enumValues))
	for index, storageType := range enumValues {
		if storageType == "" || storageType == "SQL_DATA_DISK_TYPE_UNSPECIFIED" {
			continue
		}
		if strings.HasPrefix(storageType, "OBSOLETE_") {
			continue
		}
		if index < len(enumDeprecated) && enumDeprecated[index] {
			continue
		}
		storageTypes = append(storageTypes, storageType)
	}
	sort.Strings(storageTypes)
	return storageTypes
}

func compareVersionStrings(leftVersion, rightVersion string) int {
	leftParts := strings.Split(leftVersion, ".")
	rightParts := strings.Split(rightVersion, ".")
	maxLen := len(leftParts)
	if len(rightParts) > maxLen {
		maxLen = len(rightParts)
	}

	for index := 0; index < maxLen; index++ {
		leftValue := 0
		if index < len(leftParts) {
			leftValue, _ = strconv.Atoi(leftParts[index])
		}
		rightValue := 0
		if index < len(rightParts) {
			rightValue, _ = strconv.Atoi(rightParts[index])
		}
		if leftValue < rightValue {
			return -1
		}
		if leftValue > rightValue {
			return 1
		}
	}
	return 0
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
	// BackupTime (StartTime) is not configurable at creation via Spider (GCP auto-assigns)
	settings.BackupConfiguration = backupConfig

	// IP Configuration (Public Access, Network)
	// GCP Cloud SQL supports three networking modes:
	//   1. Public IP only: Ipv4Enabled=true, PrivateNetwork not set
	//   2. Private IP only: Ipv4Enabled=false, PrivateNetwork set (requires Service Networking Peering)
	//   3. Hybrid (both): Ipv4Enabled=true, PrivateNetwork set (requires Service Networking Peering)
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

	// VPC Network - GCP uses Service Networking Peering (private service connection)
	// Only set PrivateNetwork when PublicAccess=false (Private IP mode)
	// PublicAccess=true uses public IP only (VPC is for SecurityGroup reference, not for private connectivity)
	if !rdbmsReqInfo.PublicAccess && rdbmsReqInfo.VpcIID.SystemId != "" {
		// TODO: Implement Service Networking Peering auto-creation with vpcSharedResourceSPLock.
		// Current behavior: User must manually create peering via gcloud:
		//   1. gcloud services enable servicenetworking.googleapis.com
		//   2. gcloud compute addresses create google-managed-services-{vpc} --global --purpose=VPC_PEERING --prefix-length=16 --network={vpc}
		//   3. gcloud services vpc-peerings connect --service=servicenetworking.googleapis.com --ranges=google-managed-services-{vpc} --network={vpc}
		//
		// Required implementation:
		//   - Use vpcSharedResourceSPLock.Lock(connectionName, vpcName) to prevent concurrent peering creation
		//   - Check if peering exists via compute API (addresses.list with purpose=VPC_PEERING)
		//   - If not exists, create IP range and service connection via servicenetworking API
		//   - Reference: https://cloud.google.com/sql/docs/mysql/configure-private-services-access
		cblogger.Warnf("[GCP RDBMS] Private network mode detected (PublicAccess=false). Ensure Service Networking Peering is manually created for VPC '%s'.", rdbmsReqInfo.VpcIID.SystemId)
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

	// Get instance info to extract VPC (for Service Networking Peering cleanup)
	instance, err := handler.Client.Instances.Get(projectId, rdbmsIID.SystemId).Do()
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	// Extract VPC for peering cleanup check
	var vpcNetwork string
	if instance.Settings != nil && instance.Settings.IpConfiguration != nil {
		vpcNetwork = instance.Settings.IpConfiguration.PrivateNetwork
	}

	// Disable deletion protection if needed
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

	// TODO: Implement Service Networking Peering auto-deletion with vpcSharedResourceSPLock.
	// Current behavior: Peering remains after RDBMS deletion (user must manually delete).
	//
	// Required implementation:
	//   - If vpcNetwork is not empty (private network was used):
	//     1. Use vpcSharedResourceSPLock.Lock(connectionName, vpcName) for concurrency control
	//     2. Query all Cloud SQL instances in the same VPC via handler.listInstancesInVPC(vpcNetwork)
	//     3. If no other instances exist (last instance deleted), delete Service Networking Peering:
	//        a. gcloud services vpc-peerings delete --service=servicenetworking.googleapis.com --network={vpc}
	//        b. gcloud compute addresses delete google-managed-services-{vpc} --global
	//     4. Unlock vpcSharedResourceSPLock
	//
	// Race condition without lock:
	//   - Thread A deletes db-1, checks "db-2 exists" → keeps peering
	//   - Thread B deletes db-2, checks "db-1 exists" → keeps peering
	//   - Result: Both deleted but peering remains, VPC deletion fails
	if vpcNetwork != "" {
		cblogger.Warnf("[GCP RDBMS] Instance with private network deleted. Service Networking Peering for VPC '%s' remains (manual cleanup required if this was the last instance).", vpcNetwork)
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

	// Endpoint with port
	if len(instance.IpAddresses) > 0 {
		for _, ip := range instance.IpAddresses {
			if ip.Type == "PRIMARY" || ip.Type == "PRIVATE" {
				// Add default port based on engine
				port := "3306" // MySQL default
				if engine == "postgresql" {
					port = "5432"
				}
				rdbmsInfo.Endpoint = fmt.Sprintf("%s:%s", ip.IpAddress, port)
				break
			}
		}
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

// ─── rdbmsDatabaseManager interface implementation ───────────────────────────

// CreateDatabase creates a database in a GCP Cloud SQL instance.
func (handler *GCPRDBMSHandler) CreateDatabase(rdbmsSystemId, dbEngine, dbName string) error {
	ctx := context.Background()
	projectId := handler.getProjectId()
	db := &sqladmin.Database{Name: dbName}
	op, err := handler.Client.Databases.Insert(projectId, rdbmsSystemId, db).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("GCP CreateDatabase: %w", err)
	}
	return handler.waitForOperation(projectId, op.Name)
}

// ListDatabases lists all user-created databases in a GCP Cloud SQL instance.
func (handler *GCPRDBMSHandler) ListDatabases(rdbmsSystemId, dbEngine string) ([]string, error) {
	ctx := context.Background()
	projectId := handler.getProjectId()
	resp, err := handler.Client.Databases.List(projectId, rdbmsSystemId).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("GCP ListDatabases: %w", err)
	}
	var names []string
	for _, db := range resp.Items {
		names = append(names, db.Name)
	}
	return names, nil
}

// DeleteDatabase deletes a database from a GCP Cloud SQL instance.
func (handler *GCPRDBMSHandler) DeleteDatabase(rdbmsSystemId, dbEngine, dbName string) error {
	ctx := context.Background()
	projectId := handler.getProjectId()
	op, err := handler.Client.Databases.Delete(projectId, rdbmsSystemId, dbName).Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("GCP DeleteDatabase: %w", err)
	}
	return handler.waitForOperation(projectId, op.Name)
}

// ===== Service Networking Peering Helpers =====

// listInstancesInVPC returns all Cloud SQL instances that use the specified VPC private network.
// This is used to determine if a Service Networking Peering can be safely deleted (only when no instances remain).
func (handler *GCPRDBMSHandler) listInstancesInVPC(vpcNetwork string) ([]*sqladmin.DatabaseInstance, error) {
	projectId := handler.getProjectId()
	resp, err := handler.Client.Instances.List(projectId).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list Cloud SQL instances: %w", err)
	}

	var instances []*sqladmin.DatabaseInstance
	for _, instance := range resp.Items {
		if instance.Settings != nil && instance.Settings.IpConfiguration != nil {
			if instance.Settings.IpConfiguration.PrivateNetwork == vpcNetwork {
				instances = append(instances, instance)
			}
		}
	}
	return instances, nil
}
