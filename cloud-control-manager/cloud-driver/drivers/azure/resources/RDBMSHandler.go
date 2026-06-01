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
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	armmysqlfs "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/mysql/armmysqlflexibleservers"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AzureRDBMSHandler struct {
	CredentialInfo      idrv.CredentialInfo
	Region              idrv.RegionInfo
	Ctx                 context.Context
	ServersClient       *armmysqlfs.ServersClient
	FirewallRulesClient *armmysqlfs.FirewallRulesClient
	DatabasesClient     *armmysqlfs.DatabasesClient
}

var azureMySQLFallbackVersions = []string{"5.7", "8.0.21", "8.4", "9.5"}

var azureMySQLFallbackStorageTypeOptions = map[string][]string{
	"mysql": {"GeneralPurpose", "BusinessCritical", "Burstable"},
}

var azureMySQLFallbackStorageSizeRange = irs.StorageSizeRange{Min: 20, Max: 16384}

// GetMetaInfo returns metadata about Azure Database for MySQL Flexible Server capabilities.
// Supported engine versions are fetched dynamically via the Azure REST API (api-version 2023-12-30).
func (handler *AzureRDBMSHandler) GetMetaInfo(dbEngine string) (irs.RDBMSMetaInfo, error) {
	cblogger.Debug("Azure MySQL Flexible Server GetMetaInfo() called")

	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "GetMetaInfo", "Microsoft.DBforMySQL/locations/capabilitySets/default")
	start := call.Start()
	requestedEngine, err := irs.NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return irs.RDBMSMetaInfo{}, err
	}

	versions, storageTypeOptions, storageSizeRange, err := handler.fetchMySQLMetaOptions(handler.Region.Region)
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSMetaInfo{}, fmt.Errorf("GetMetaInfo failed: %w", err)
	}

	metaInfo, err := irs.BuildRDBMSMetaInfo(requestedEngine, map[string][]string{"mysql": versions}, storageTypeOptions, storageSizeRange, true, true, true, false, true)
	if err != nil {
		return irs.RDBMSMetaInfo{}, err
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return metaInfo, nil
}

// fetchMySQLMetaOptions queries the Azure LocationBasedCapabilitySet_Get endpoint
// (api-version 2023-12-30) to get supported MySQL versions and storage options for the given location.
// This endpoint supersedes the legacy /capabilities endpoint and is stable across all regions.
func (handler *AzureRDBMSHandler) fetchMySQLMetaOptions(location string) ([]string, map[string][]string, irs.StorageSizeRange, error) {
	cred, err := azidentity.NewClientSecretCredential(
		handler.CredentialInfo.TenantId,
		handler.CredentialInfo.ClientId,
		handler.CredentialInfo.ClientSecret,
		nil,
	)
	if err != nil {
		return nil, nil, irs.StorageSizeRange{}, fmt.Errorf("failed to create Azure credential: %w", err)
	}

	token, err := cred.GetToken(handler.Ctx, policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		return nil, nil, irs.StorageSizeRange{}, fmt.Errorf("failed to get Azure token: %w", err)
	}

	apiURL := fmt.Sprintf(
		"https://management.azure.com/subscriptions/%s/providers/Microsoft.DBforMySQL/locations/%s/capabilitySets/default?api-version=2023-12-30",
		handler.CredentialInfo.SubscriptionId, location,
	)
	versions, storageTypeOptions, storageSizeRange, err := handler.doCapabilitySetRequest(apiURL, token.Token)
	if err != nil {
		fallbackVersions := append([]string(nil), azureMySQLFallbackVersions...)
		fallbackStorageTypeOptions := map[string][]string{
			"mysql": append([]string(nil), azureMySQLFallbackStorageTypeOptions["mysql"]...),
		}
		cblogger.Infof("Azure MySQL capabilitySets API failed for location %s. Using fallback versions %v and storage metadata %v/%+v: %v", location, fallbackVersions, fallbackStorageTypeOptions, azureMySQLFallbackStorageSizeRange, err)
		return fallbackVersions, fallbackStorageTypeOptions, azureMySQLFallbackStorageSizeRange, nil
	}
	return versions, storageTypeOptions, storageSizeRange, nil
}

// doCapabilitySetRequest calls the Azure LocationBasedCapabilitySet_Get endpoint.
// URL: GET .../capabilitySets/default?api-version=2023-12-30
// Response: { "properties": { "supportedServerVersions": [{"name":"..."}] } }
func (handler *AzureRDBMSHandler) doCapabilitySetRequest(apiURL, bearerToken string) ([]string, map[string][]string, irs.StorageSizeRange, error) {
	req, err := http.NewRequestWithContext(handler.Ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, nil, irs.StorageSizeRange{}, fmt.Errorf("failed to build capabilitySets request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, nil, irs.StorageSizeRange{}, fmt.Errorf("capabilitySets request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, irs.StorageSizeRange{}, fmt.Errorf("failed to read capabilitySets response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, irs.StorageSizeRange{}, fmt.Errorf("capabilitySets API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Properties struct {
			SupportedFlexibleServerEditions []struct {
				Name                     string `json:"name"`
				SupportedStorageEditions []struct {
					Name           string `json:"name"`
					MinStorageSize int64  `json:"minStorageSize"`
					MaxStorageSize int64  `json:"maxStorageSize"`
				} `json:"supportedStorageEditions"`
			} `json:"supportedFlexibleServerEditions"`
			SupportedServerVersions []struct {
				Name string `json:"name"`
			} `json:"supportedServerVersions"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, nil, irs.StorageSizeRange{}, fmt.Errorf("failed to parse capabilitySets response: %w", err)
	}

	versionSet := map[string]bool{}
	for _, sv := range result.Properties.SupportedServerVersions {
		if sv.Name != "" {
			versionSet[sv.Name] = true
		}
	}

	versions := make([]string, 0, len(versionSet))
	for v := range versionSet {
		versions = append(versions, v)
	}
	sort.Strings(versions)

	storageTypeSet := map[string]bool{}
	var minStorageGB int64
	var maxStorageGB int64
	for _, edition := range result.Properties.SupportedFlexibleServerEditions {
		if edition.Name != "" {
			storageTypeSet[edition.Name] = true
		}
		for _, storageEdition := range edition.SupportedStorageEditions {
			if storageEdition.MinStorageSize > 0 {
				minGB := azureStorageMBToGB(storageEdition.MinStorageSize)
				if minStorageGB == 0 || minGB < minStorageGB {
					minStorageGB = minGB
				}
			}
			if storageEdition.MaxStorageSize > 0 {
				maxGB := azureStorageMBToGB(storageEdition.MaxStorageSize)
				if maxGB > maxStorageGB {
					maxStorageGB = maxGB
				}
			}
		}
	}
	storageTypes := make([]string, 0, len(storageTypeSet))
	for storageType := range storageTypeSet {
		storageTypes = append(storageTypes, storageType)
	}
	sort.Strings(storageTypes)
	if len(storageTypes) == 0 || minStorageGB == 0 || maxStorageGB == 0 {
		return nil, nil, irs.StorageSizeRange{}, fmt.Errorf("capabilitySets response did not include storage options")
	}

	storageTypeOptions := map[string][]string{"mysql": storageTypes}
	storageSizeRange := irs.StorageSizeRange{Min: minStorageGB, Max: maxStorageGB}
	return versions, storageTypeOptions, storageSizeRange, nil
}

func azureStorageMBToGB(storageMB int64) int64 {
	return storageMB / 1024
}

func (handler *AzureRDBMSHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "ListIID", "Servers.NewListByResourceGroupPager()")
	start := call.Start()

	resourceGroup := handler.Region.Region
	var iidList []*irs.IID

	pager := handler.ServersClient.NewListByResourceGroupPager(resourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(handler.Ctx)
		if err != nil {
			hiscallInfo.ElapsedTime = call.Elapsed(start)
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return nil, err
		}
		for _, server := range page.Value {
			name := ""
			if server.Name != nil {
				name = *server.Name
			}
			iidList = append(iidList, &irs.IID{NameId: name, SystemId: name})
		}
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return iidList, nil
}

func (handler *AzureRDBMSHandler) CreateRDBMS(rdbmsReqInfo irs.RDBMSInfo) (irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, rdbmsReqInfo.IId.NameId, "Servers.BeginCreate()")
	start := call.Start()

	// Validate required fields
	if rdbmsReqInfo.IId.NameId == "" {
		return irs.RDBMSInfo{}, errors.New("RDBMS NameId is required")
	}
	if rdbmsReqInfo.DBEngine == "" {
		return irs.RDBMSInfo{}, errors.New("DBEngine is required")
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

	resourceGroup := handler.Region.Region
	location := handler.Region.Region

	storageSizeGB, err := strconv.ParseInt(rdbmsReqInfo.StorageSize, 10, 32)
	if err != nil {
		return irs.RDBMSInfo{}, fmt.Errorf("invalid StorageSize: %s", rdbmsReqInfo.StorageSize)
	}
	storageSizeGB32 := int32(storageSizeGB)

	// Map engine version
	version := armmysqlfs.ServerVersionEight021
	switch rdbmsReqInfo.DBEngineVersion {
	case "5.7":
		version = armmysqlfs.ServerVersionFive7
	case "8.0.21", "8.0":
		version = armmysqlfs.ServerVersionEight021
	}

	// High Availability
	haMode := armmysqlfs.HighAvailabilityModeDisabled
	if rdbmsReqInfo.HighAvailability {
		haMode = armmysqlfs.HighAvailabilityModeZoneRedundant
	}

	// Backup retention
	backupRetention := int32(7)
	if rdbmsReqInfo.BackupRetentionDays > 0 {
		backupRetention = int32(rdbmsReqInfo.BackupRetentionDays)
	}

	// Flexible Server create mode
	createMode := armmysqlfs.CreateModeDefault

	// PublicNetworkAccess
	publicAccess := armmysqlfs.EnableStatusEnumDisabled
	if rdbmsReqInfo.PublicAccess {
		publicAccess = armmysqlfs.EnableStatusEnumEnabled
	}

	parameters := armmysqlfs.Server{
		Location: &location,
		Properties: &armmysqlfs.ServerProperties{
			AdministratorLogin:         &rdbmsReqInfo.MasterUserName,
			AdministratorLoginPassword: &rdbmsReqInfo.MasterUserPassword,
			Version:                    &version,
			CreateMode:                 &createMode,
			Storage: &armmysqlfs.Storage{
				StorageSizeGB: &storageSizeGB32,
			},
			Backup: &armmysqlfs.Backup{
				BackupRetentionDays: &backupRetention,
			},
			HighAvailability: &armmysqlfs.HighAvailability{
				Mode: &haMode,
			},
			Network: &armmysqlfs.Network{
				PublicNetworkAccess: &publicAccess,
			},
		},
		SKU: &armmysqlfs.SKU{
			Name: &rdbmsReqInfo.DBInstanceSpec,
			Tier: skuTierFromSpec(rdbmsReqInfo.DBInstanceSpec),
		},
	}

	// Tags
	if len(rdbmsReqInfo.TagList) > 0 {
		tags := make(map[string]*string)
		for _, tag := range rdbmsReqInfo.TagList {
			v := tag.Value
			tags[tag.Key] = &v
		}
		parameters.Tags = tags
	}

	poller, err := handler.ServersClient.BeginCreate(handler.Ctx, resourceGroup, rdbmsReqInfo.IId.NameId, parameters, nil)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	resp, err := poller.PollUntilDone(handler.Ctx, nil)
	if err != nil {
		return irs.RDBMSInfo{}, fmt.Errorf("failed waiting for server creation: %w", err)
	}

	// Add firewall rule to allow all IPs when PublicAccess is enabled
	if rdbmsReqInfo.PublicAccess && handler.FirewallRulesClient != nil {
		serverName := rdbmsReqInfo.IId.NameId
		if resp.Server.Name != nil {
			serverName = *resp.Server.Name
		}
		startIP := "0.0.0.0"
		endIP := "255.255.255.255"
		fwRule := armmysqlfs.FirewallRule{
			Properties: &armmysqlfs.FirewallRuleProperties{
				StartIPAddress: &startIP,
				EndIPAddress:   &endIP,
			},
		}
		fwPoller, fwErr := handler.FirewallRulesClient.BeginCreateOrUpdate(
			handler.Ctx, resourceGroup, serverName, "AllowAll", fwRule, nil)
		if fwErr != nil {
			cblogger.Warn("failed to create firewall rule: ", fwErr)
		} else {
			_, fwErr = fwPoller.PollUntilDone(handler.Ctx, nil)
			if fwErr != nil {
				cblogger.Warn("failed waiting for firewall rule creation: ", fwErr)
			}
		}
	}

	return handler.convertToRDBMSInfo(&resp.Server), nil
}

func (handler *AzureRDBMSHandler) ListRDBMS() ([]*irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "ListRDBMS", "Servers.NewListByResourceGroupPager()")
	start := call.Start()

	resourceGroup := handler.Region.Region
	var rdbmsList []*irs.RDBMSInfo

	pager := handler.ServersClient.NewListByResourceGroupPager(resourceGroup, nil)
	for pager.More() {
		page, err := pager.NextPage(handler.Ctx)
		if err != nil {
			hiscallInfo.ElapsedTime = call.Elapsed(start)
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return nil, err
		}
		for _, server := range page.Value {
			rdbmsInfo := handler.convertToRDBMSInfo(server)
			rdbmsList = append(rdbmsList, &rdbmsInfo)
		}
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return rdbmsList, nil
}

func (handler *AzureRDBMSHandler) GetRDBMS(rdbmsIID irs.IID) (irs.RDBMSInfo, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, rdbmsIID.NameId, "Servers.Get()")
	start := call.Start()

	resourceGroup := handler.Region.Region
	resp, err := handler.ServersClient.Get(handler.Ctx, resourceGroup, rdbmsIID.SystemId, nil)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	return handler.convertToRDBMSInfo(&resp.Server), nil
}

func (handler *AzureRDBMSHandler) DeleteRDBMS(rdbmsIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, rdbmsIID.NameId, "Servers.BeginDelete()")
	start := call.Start()

	resourceGroup := handler.Region.Region

	poller, err := handler.ServersClient.BeginDelete(handler.Ctx, resourceGroup, rdbmsIID.SystemId, nil)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	calllogger.Info(call.String(hiscallInfo))

	_, err = poller.PollUntilDone(handler.Ctx, nil)
	if err != nil {
		return false, fmt.Errorf("failed waiting for server deletion: %w", err)
	}

	return true, nil
}

// ===== Helper Functions =====

func skuTierFromSpec(spec string) *armmysqlfs.SKUTier {
	tierBurstable := armmysqlfs.SKUTierBurstable
	tierGP := armmysqlfs.SKUTierGeneralPurpose
	tierMO := armmysqlfs.SKUTierMemoryOptimized

	specLower := strings.ToLower(spec)
	switch {
	case strings.HasPrefix(specLower, "standard_b"):
		return &tierBurstable
	case strings.HasPrefix(specLower, "standard_e"):
		return &tierMO
	default:
		return &tierGP
	}
}

func (handler *AzureRDBMSHandler) convertToRDBMSInfo(server *armmysqlfs.Server) irs.RDBMSInfo {
	rdbmsInfo := irs.RDBMSInfo{}

	if server.Name != nil {
		rdbmsInfo.IId = irs.IID{
			NameId:   *server.Name,
			SystemId: *server.Name,
		}
	}

	// VPC - Azure MySQL Flexible Server networking is managed separately
	rdbmsInfo.VpcIID = irs.IID{SystemId: "NA"}

	// Engine
	rdbmsInfo.DBEngine = "mysql"
	if server.Properties != nil {
		if server.Properties.Version != nil {
			rdbmsInfo.DBEngineVersion = string(*server.Properties.Version)
		}

		// FQDN as endpoint
		if server.Properties.FullyQualifiedDomainName != nil {
			rdbmsInfo.Endpoint = *server.Properties.FullyQualifiedDomainName + ":3306"
		}

		// Admin user
		if server.Properties.AdministratorLogin != nil {
			rdbmsInfo.MasterUserName = *server.Properties.AdministratorLogin
		}

		// Status
		if server.Properties.State != nil {
			rdbmsInfo.Status = convertFlexibleServerStatus(string(*server.Properties.State))
		}

		// Storage
		if server.Properties.Storage != nil {
			if server.Properties.Storage.StorageSizeGB != nil {
				rdbmsInfo.StorageSize = strconv.FormatInt(int64(*server.Properties.Storage.StorageSizeGB), 10)
			}
		}

		// Backup
		if server.Properties.Backup != nil {
			if server.Properties.Backup.BackupRetentionDays != nil {
				rdbmsInfo.BackupRetentionDays = int(*server.Properties.Backup.BackupRetentionDays)
			}
		}

		// High Availability
		if server.Properties.HighAvailability != nil && server.Properties.HighAvailability.Mode != nil {
			rdbmsInfo.HighAvailability = (*server.Properties.HighAvailability.Mode != armmysqlfs.HighAvailabilityModeDisabled)
		}

		// Replication
		if server.Properties.ReplicationRole != nil {
			rdbmsInfo.ReplicationType = string(*server.Properties.ReplicationRole)
		}

		// PublicAccess
		if server.Properties.Network != nil && server.Properties.Network.PublicNetworkAccess != nil {
			rdbmsInfo.PublicAccess = (*server.Properties.Network.PublicNetworkAccess == armmysqlfs.EnableStatusEnumEnabled)
		}
	}

	// SKU
	if server.SKU != nil {
		if server.SKU.Name != nil {
			rdbmsInfo.DBInstanceSpec = *server.SKU.Name
		}
		if server.SKU.Tier != nil {
			rdbmsInfo.DBInstanceType = string(*server.SKU.Tier)
		}
	}

	// Port
	rdbmsInfo.Port = "3306"
	rdbmsInfo.StorageType = "NA"
	rdbmsInfo.DatabaseName = ""
	rdbmsInfo.BackupTime = "NA"
	rdbmsInfo.DeletionProtection = false

	// CreatedTime from SystemData
	if server.SystemData != nil && server.SystemData.CreatedAt != nil {
		rdbmsInfo.CreatedTime = *server.SystemData.CreatedAt
	}

	// Tags
	for k, v := range server.Tags {
		val := ""
		if v != nil {
			val = *v
		}
		rdbmsInfo.TagList = append(rdbmsInfo.TagList, irs.KeyValue{Key: k, Value: val})
	}

	// KeyValueList
	rdbmsInfo.KeyValueList = irs.StructToKeyValueList(server)

	return rdbmsInfo
}

func convertFlexibleServerStatus(state string) irs.RDBMSStatus {
	switch strings.ToLower(state) {
	case "ready":
		return irs.RDBMSAvailable
	case "stopped", "disabled", "dropping":
		return irs.RDBMSStopped
	case "starting", "updating":
		return irs.RDBMSCreating
	default:
		return irs.RDBMSError
	}
}

// ─── rdbmsDatabaseManager interface implementation ───────────────────────────
// Azure MySQL Flexible Server supports native database management via the ARM Databases API.

// CreateDatabase creates a database in an Azure MySQL Flexible Server instance.
func (handler *AzureRDBMSHandler) CreateDatabase(rdbmsSystemId, dbEngine, dbName string) error {
	if handler.DatabasesClient == nil {
		return fmt.Errorf("Azure CreateDatabase: DatabasesClient is not initialized")
	}
	resourceGroup := handler.Region.Region
	poller, err := handler.DatabasesClient.BeginCreateOrUpdate(
		handler.Ctx, resourceGroup, rdbmsSystemId, dbName,
		armmysqlfs.Database{}, nil,
	)
	if err != nil {
		return fmt.Errorf("Azure CreateDatabase: %w", err)
	}
	if _, err := poller.PollUntilDone(handler.Ctx, nil); err != nil {
		return fmt.Errorf("Azure CreateDatabase (poll): %w", err)
	}
	return nil
}

// ListDatabases lists all databases in an Azure MySQL Flexible Server instance.
func (handler *AzureRDBMSHandler) ListDatabases(rdbmsSystemId, dbEngine string) ([]string, error) {
	if handler.DatabasesClient == nil {
		return nil, fmt.Errorf("Azure ListDatabases: DatabasesClient is not initialized")
	}
	resourceGroup := handler.Region.Region
	pager := handler.DatabasesClient.NewListByServerPager(resourceGroup, rdbmsSystemId, nil)
	var names []string
	for pager.More() {
		page, err := pager.NextPage(handler.Ctx)
		if err != nil {
			return nil, fmt.Errorf("Azure ListDatabases: %w", err)
		}
		for _, db := range page.Value {
			if db.Name != nil {
				names = append(names, *db.Name)
			}
		}
	}
	return names, nil
}

// DeleteDatabase deletes a database from an Azure MySQL Flexible Server instance.
func (handler *AzureRDBMSHandler) DeleteDatabase(rdbmsSystemId, dbEngine, dbName string) error {
	if handler.DatabasesClient == nil {
		return fmt.Errorf("Azure DeleteDatabase: DatabasesClient is not initialized")
	}
	resourceGroup := handler.Region.Region
	poller, err := handler.DatabasesClient.BeginDelete(handler.Ctx, resourceGroup, rdbmsSystemId, dbName, nil)
	if err != nil {
		return fmt.Errorf("Azure DeleteDatabase: %w", err)
	}
	if _, err := poller.PollUntilDone(handler.Ctx, nil); err != nil {
		return fmt.Errorf("Azure DeleteDatabase (poll): %w", err)
	}
	return nil
}
