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
	"errors"
	"fmt"
	"strconv"
	"strings"

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
}

func (handler *AzureRDBMSHandler) GetMetaInfo() (irs.RDBMSMetaInfo, error) {
	cblogger.Debug("Azure MySQL Flexible Server GetMetaInfo() called")

	hiscallInfo := GetCallLogScheme(handler.Region, call.RDBMS, "GetMetaInfo", "GetMetaInfo()")
	start := call.Start()

	metaInfo := irs.RDBMSMetaInfo{
		SupportedEngines: map[string][]string{
			"mysql": {"5.7", "8.0.21"},
		},
		SupportsHighAvailability:   true,
		SupportsBackup:             true,
		SupportsPublicAccess:       true,
		SupportsDeletionProtection: false,
		SupportsEncryption:         true,
		StorageTypeOptions: map[string][]string{
			"mysql": {"GeneralPurpose", "BusinessCritical", "Burstable"},
		},
		StorageSizeRange: irs.StorageSizeRange{
			Min: 20,
			Max: 16384,
		},
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return metaInfo, nil
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
