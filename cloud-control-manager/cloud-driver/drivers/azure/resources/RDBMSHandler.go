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
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v9"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/privatedns/armprivatedns"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AzureRDBMSHandler struct {
	CredentialInfo           idrv.CredentialInfo
	Region                   idrv.RegionInfo
	Ctx                      context.Context
	ServersClient            *armmysqlfs.ServersClient
	FirewallRulesClient      *armmysqlfs.FirewallRulesClient
	DatabasesClient          *armmysqlfs.DatabasesClient
	SubnetClient             *armnetwork.SubnetsClient
	PrivateDNSZonesClient    *armprivatedns.PrivateZonesClient
	PrivateDNSVNetLinkClient *armprivatedns.VirtualNetworkLinksClient
	BackupsClient            *armmysqlfs.BackupsClient
}

// Azure MySQL Flexible Server storage type (storageSku) is read-only and set automatically by Azure.
// StorageTypeOptions returns ["NA"] to indicate it is not user-selectable.

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

	versions, skus, storageSizeRange, err := handler.fetchMySQLMetaOptions(handler.Region.Region)
	if err != nil {
		hiscallInfo.ElapsedTime = call.Elapsed(start)
		LoggingError(hiscallInfo, err)
		return irs.RDBMSMetaInfo{}, fmt.Errorf("GetMetaInfo failed: %w", err)
	}

	// Azure MySQL Flexible Server provides SKU list via LocationBasedCapabilitySet API
	instanceSpecOptions := map[string][]string{
		"mysql": skus,
	}
	// Azure storageSku is read-only and set automatically; not user-selectable
	storageTypeOptions := map[string][]string{"mysql": {"NA"}}

	metaInfo, err := irs.BuildRDBMSMetaInfo(requestedEngine, map[string][]string{"mysql": versions}, instanceSpecOptions, storageTypeOptions, storageSizeRange, true, true, true, false, true, "1-35", false, false, false, true)
	if err != nil {
		return irs.RDBMSMetaInfo{}, err
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return metaInfo, nil
}

// fetchMySQLMetaOptions queries the Azure LocationBasedCapabilitySet_Get endpoint
// (api-version 2023-12-30) to get supported MySQL versions, SKUs, and storage size range for the given location.
// StorageType is not returned because Azure sets storageSku automatically (read-only).
func (handler *AzureRDBMSHandler) fetchMySQLMetaOptions(location string) ([]string, []string, irs.StorageSizeRange, error) {
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
	versions, skus, storageSizeRange, err := handler.doCapabilitySetRequest(apiURL, token.Token)
	if err != nil {
		return nil, nil, irs.StorageSizeRange{}, fmt.Errorf("capabilitySets API failed for location %s: %w", location, err)
	}
	return versions, skus, storageSizeRange, nil
}

// doCapabilitySetRequest calls the Azure LocationBasedCapabilitySet_Get endpoint.
// URL: GET .../capabilitySets/default?api-version=2023-12-30
// Response: { "properties": { "supportedServerVersions": [{"name":"..."}], "supportedFlexibleServerEditions": [...] } }
func (handler *AzureRDBMSHandler) doCapabilitySetRequest(apiURL, bearerToken string) ([]string, []string, irs.StorageSizeRange, error) {
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
				Name          string `json:"name"`
				SupportedSkus []struct {
					Name string `json:"name"`
				} `json:"supportedSkus"`
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

	// Parse SKUs from supportedFlexibleServerEditions
	skuSet := map[string]bool{}
	for _, edition := range result.Properties.SupportedFlexibleServerEditions {
		for _, sku := range edition.SupportedSkus {
			if sku.Name != "" {
				skuSet[sku.Name] = true
			}
		}
	}
	skus := make([]string, 0, len(skuSet))
	for s := range skuSet {
		skus = append(skus, s)
	}
	sort.Strings(skus)

	var minStorageGB int64
	var maxStorageGB int64
	for _, edition := range result.Properties.SupportedFlexibleServerEditions {
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
	if minStorageGB == 0 || maxStorageGB == 0 {
		return nil, nil, irs.StorageSizeRange{}, fmt.Errorf("capabilitySets response did not include storage size options")
	}

	storageSizeRange := irs.StorageSizeRange{Min: minStorageGB, Max: maxStorageGB}
	return versions, skus, storageSizeRange, nil
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
	if rdbmsReqInfo.StorageType != "" {
		return irs.RDBMSInfo{}, errors.New("StorageType is not supported for Azure: storage type is set automatically by the CSP. See SupportsStorageTypeSelection in GetMetaInfo")
	}

	// Validate PublicAccess and VPC combination
	// Case 1: PublicAccess=true, VPCName present → OK (public mode; VPCName is recorded as OwnerVPC but not passed to Azure API)
	// Case 2: PublicAccess=true, VPCName absent  → OK (public mode with firewall rules)
	// Case 3: PublicAccess=false, VPCName present → OK (VPC private mode; subnet will be dedicated to MySQL and cannot be used for other resources)
	// Case 4: PublicAccess=false, VPCName absent  → error (no access path available without VPC)
	hasVPC := rdbmsReqInfo.VpcIID.NameId != "" || rdbmsReqInfo.VpcIID.SystemId != ""
	if rdbmsReqInfo.PublicAccess && hasVPC {
		cblogger.Infof("[Azure RDBMS] PublicAccess=true with VPCName='%s': "+
			"VPC integration is not used for public-access Azure MySQL Flexible Server. "+
			"VPCName will be recorded as OwnerVPC but is not passed to the Azure CSP API.",
			rdbmsReqInfo.VpcIID.NameId)
	}
	if !rdbmsReqInfo.PublicAccess && !hasVPC {
		return irs.RDBMSInfo{}, errors.New(
			"PublicAccess=false requires a VPCName: " +
				"Azure MySQL Flexible Server with public access disabled must be deployed in a VPC. " +
				"Please provide VPCName (and optionally SubnetNames) to enable private access.")
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

	// Backup retention: -1 or 0 = not set (use Azure default 7 days), 1-35 = explicit value
	var backupRetention *int32
	if rdbmsReqInfo.BackupRetentionDays > 0 {
		retention := int32(rdbmsReqInfo.BackupRetentionDays)
		backupRetention = &retention
	}

	// Flexible Server create mode
	createMode := armmysqlfs.CreateModeDefault

	// PublicNetworkAccess
	publicAccess := armmysqlfs.EnableStatusEnumDisabled
	if rdbmsReqInfo.PublicAccess {
		publicAccess = armmysqlfs.EnableStatusEnumEnabled
	}

	// VPC private mode: ensure subnet delegation and create Private DNS Zone
	var delegatedSubnetResourceID, privateDNSZoneResourceID string
	if !rdbmsReqInfo.PublicAccess && hasVPC {
		cblogger.Infof("[Azure RDBMS] VPC private mode requested (VPCName=%s). "+
			"The specified subnet will be dedicated exclusively to Azure MySQL Flexible Server "+
			"and cannot be used for VMs or other resources.",
			rdbmsReqInfo.VpcIID.NameId)

		// Use the first subnet if provided, otherwise error
		if len(rdbmsReqInfo.SubnetIIDs) == 0 {
			return irs.RDBMSInfo{}, fmt.Errorf(
				"VPC private mode requires at least one SubnetName: " +
					"please provide a dedicated subnet for Azure MySQL Flexible Server " +
					"(the subnet must not contain VMs or other resources)")
		}

		subnetResourceID := rdbmsReqInfo.SubnetIIDs[0].SystemId
		vnetName := rdbmsReqInfo.VpcIID.NameId

		// Ensure the subnet has delegation to Microsoft.DBforMySQL/flexibleServers
		if err := handler.ensureSubnetDelegation(resourceGroup, vnetName, rdbmsReqInfo.SubnetIIDs[0].NameId, subnetResourceID); err != nil {
			return irs.RDBMSInfo{}, err
		}

		// Create Private DNS Zone: <serverName>.private.mysql.database.azure.com
		// NOTE: Current implementation creates a separate DNS Zone per server (not VPC-level shared).
		// This avoids the concurrency issue that GCP Service Networking Peering has.
		// Future optimization: Use a single VNet-level zone (e.g., "privatelink.mysql.database.azure.com")
		// and add A records per server. If that optimization is implemented, use vpcSharedResourceSPLock
		// to prevent concurrent zone creation/deletion (similar to GCP).
		dnsZoneName := rdbmsReqInfo.IId.NameId + ".private.mysql.database.azure.com"
		dnsZoneID, err := handler.ensurePrivateDNSZone(resourceGroup, dnsZoneName)
		if err != nil {
			return irs.RDBMSInfo{}, err
		}

		// Link Private DNS Zone to VNet
		vnetResourceID := rdbmsReqInfo.VpcIID.SystemId
		if err := handler.ensurePrivateDNSVNetLink(resourceGroup, dnsZoneName, vnetName, vnetResourceID); err != nil {
			// best-effort cleanup
			handler.deletePrivateDNSZone(resourceGroup, dnsZoneName)
			return irs.RDBMSInfo{}, err
		}

		delegatedSubnetResourceID = subnetResourceID
		privateDNSZoneResourceID = dnsZoneID
	}

	network := &armmysqlfs.Network{
		PublicNetworkAccess: &publicAccess,
	}
	if delegatedSubnetResourceID != "" {
		network.DelegatedSubnetResourceID = &delegatedSubnetResourceID
		network.PrivateDNSZoneResourceID = &privateDNSZoneResourceID
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
				BackupRetentionDays: backupRetention,
			},
			HighAvailability: &armmysqlfs.HighAvailability{
				Mode: &haMode,
			},
			Network: network,
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
		if privateDNSZoneResourceID != "" {
			handler.deletePrivateDNSZone(resourceGroup, rdbmsReqInfo.IId.NameId+".private.mysql.database.azure.com")
		}
		return irs.RDBMSInfo{}, err
	}
	calllogger.Info(call.String(hiscallInfo))

	resp, err := poller.PollUntilDone(handler.Ctx, nil)
	if err != nil {
		cblogger.Error(err)
		if privateDNSZoneResourceID != "" {
			handler.deletePrivateDNSZone(resourceGroup, rdbmsReqInfo.IId.NameId+".private.mysql.database.azure.com")
		}
		return irs.RDBMSInfo{}, err
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

	rdbmsInfo := handler.convertToRDBMSInfo(&resp.Server)
	if hasVPC {
		rdbmsInfo.VpcIID = rdbmsReqInfo.VpcIID
	}
	return rdbmsInfo, nil
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

	// NOTE: Private DNS Zone cleanup is not performed here because each server has its own zone
	// (<serverName>.private.mysql.database.azure.com). The zone is automatically removed by Azure
	// or can be manually cleaned up if needed.
	//
	// If future optimization uses a shared VNet-level zone (e.g., "privatelink.mysql.database.azure.com"),
	// implement the following logic with vpcSharedResourceSPLock:
	//   1. Lock vpcSharedResourceSPLock for the VNet
	//   2. Query all MySQL Flexible Servers in the same VNet via handler.listServersInVNet(vnetResourceID)
	//   3. If no other servers exist (last server deleted), delete the shared DNS Zone and VNet Link
	//   4. Unlock vpcSharedResourceSPLock

	return true, nil
}

// ===== Helper Functions =====

// ensureSubnetDelegation checks whether the subnet is delegated to Microsoft.DBforMySQL/flexibleServers.
// If not, it adds the delegation automatically and logs the action.
func (handler *AzureRDBMSHandler) ensureSubnetDelegation(resourceGroup, vnetName, subnetName, subnetResourceID string) error {
	const delegationService = "Microsoft.DBforMySQL/flexibleServers"

	resp, err := handler.SubnetClient.Get(handler.Ctx, resourceGroup, vnetName, subnetName, nil)
	if err != nil {
		return fmt.Errorf("failed to get subnet '%s': %w", subnetName, err)
	}

	// Check existing delegation
	for _, d := range resp.Subnet.Properties.Delegations {
		if d.Properties != nil && d.Properties.ServiceName != nil &&
			*d.Properties.ServiceName == delegationService {
			return nil // already delegated
		}
	}

	// Add delegation
	cblogger.Infof("[Azure RDBMS] Subnet '%s' is not delegated to %s. Adding delegation automatically.", subnetName, delegationService)

	delegationName := "MySQLFlexibleServerDelegation"
	serviceName := delegationService
	resp.Subnet.Properties.Delegations = append(resp.Subnet.Properties.Delegations, &armnetwork.Delegation{
		Name: &delegationName,
		Properties: &armnetwork.ServiceDelegationPropertiesFormat{
			ServiceName: &serviceName,
		},
	})

	poller, err := handler.SubnetClient.BeginCreateOrUpdate(handler.Ctx, resourceGroup, vnetName, subnetName, resp.Subnet, nil)
	if err != nil {
		return fmt.Errorf("failed to add delegation to subnet '%s': %w", subnetName, err)
	}
	if _, err = poller.PollUntilDone(handler.Ctx, nil); err != nil {
		return fmt.Errorf("failed waiting for subnet delegation update on '%s': %w", subnetName, err)
	}

	cblogger.Infof("[Azure RDBMS] Subnet '%s' delegated to %s successfully. "+
		"This subnet is now exclusively dedicated to Azure MySQL Flexible Server and cannot be used for VMs or other resources.",
		subnetName, delegationService)
	return nil
}

// ensurePrivateDNSZone creates the Private DNS Zone if it does not already exist.
// Returns the full resource ID of the zone.
func (handler *AzureRDBMSHandler) ensurePrivateDNSZone(resourceGroup, zoneName string) (string, error) {
	// Check if already exists
	existing, err := handler.PrivateDNSZonesClient.Get(handler.Ctx, resourceGroup, zoneName, nil)
	if err == nil {
		cblogger.Infof("[Azure RDBMS] Private DNS Zone '%s' already exists.", zoneName)
		return *existing.ID, nil
	}

	cblogger.Infof("[Azure RDBMS] Creating Private DNS Zone '%s'.", zoneName)
	location := "global"
	poller, err := handler.PrivateDNSZonesClient.BeginCreateOrUpdate(
		handler.Ctx, resourceGroup, zoneName,
		armprivatedns.PrivateZone{Location: &location},
		nil,
	)
	if err != nil {
		return "", fmt.Errorf("failed to create Private DNS Zone '%s': %w", zoneName, err)
	}
	result, err := poller.PollUntilDone(handler.Ctx, nil)
	if err != nil {
		return "", fmt.Errorf("failed waiting for Private DNS Zone creation '%s': %w", zoneName, err)
	}

	cblogger.Infof("[Azure RDBMS] Private DNS Zone '%s' created (ID: %s).", zoneName, *result.ID)
	return *result.ID, nil
}

// ensurePrivateDNSVNetLink links the Private DNS Zone to the VNet if not already linked.
func (handler *AzureRDBMSHandler) ensurePrivateDNSVNetLink(resourceGroup, zoneName, vnetName, vnetResourceID string) error {
	linkName := "cb-spider-" + vnetName

	// Check if already exists
	_, err := handler.PrivateDNSVNetLinkClient.Get(handler.Ctx, resourceGroup, zoneName, linkName, nil)
	if err == nil {
		cblogger.Infof("[Azure RDBMS] Private DNS VNet link '%s' already exists.", linkName)
		return nil
	}

	cblogger.Infof("[Azure RDBMS] Linking Private DNS Zone '%s' to VNet '%s'.", zoneName, vnetName)
	registrationEnabled := false
	poller, err := handler.PrivateDNSVNetLinkClient.BeginCreateOrUpdate(
		handler.Ctx, resourceGroup, zoneName, linkName,
		armprivatedns.VirtualNetworkLink{
			Location: func() *string { s := "global"; return &s }(),
			Properties: &armprivatedns.VirtualNetworkLinkProperties{
				VirtualNetwork:      &armprivatedns.SubResource{ID: &vnetResourceID},
				RegistrationEnabled: &registrationEnabled,
			},
		},
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to create VNet link for Private DNS Zone '%s': %w", zoneName, err)
	}
	if _, err = poller.PollUntilDone(handler.Ctx, nil); err != nil {
		return fmt.Errorf("failed waiting for VNet link creation for Private DNS Zone '%s': %w", zoneName, err)
	}

	cblogger.Infof("[Azure RDBMS] VNet '%s' linked to Private DNS Zone '%s' successfully.", vnetName, zoneName)
	return nil
}

// deletePrivateDNSZone is a best-effort cleanup helper used on CreateRDBMS failure.
func (handler *AzureRDBMSHandler) deletePrivateDNSZone(resourceGroup, zoneName string) {
	poller, err := handler.PrivateDNSZonesClient.BeginDelete(handler.Ctx, resourceGroup, zoneName, nil)
	if err != nil {
		cblogger.Warnf("[Azure RDBMS] Failed to delete Private DNS Zone '%s' during cleanup: %v", zoneName, err)
		return
	}
	if _, err = poller.PollUntilDone(handler.Ctx, nil); err != nil {
		cblogger.Warnf("[Azure RDBMS] Failed waiting for Private DNS Zone deletion '%s': %v", zoneName, err)
	}
}

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
			if server.Properties.Storage.StorageSKU != nil {
				rdbmsInfo.StorageType = string(*server.Properties.Storage.StorageSKU)
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

	if rdbmsInfo.StorageType == "" {
		rdbmsInfo.StorageType = "NA"
	}
	rdbmsInfo.BackupTime = "AUTO" // Azure manages backup schedule automatically
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

func (handler *AzureRDBMSHandler) enrichBackupTime(rdbmsInfo *irs.RDBMSInfo) {
	if rdbmsInfo.IId.SystemId == "" || handler.BackupsClient == nil {
		return
	}

	resourceGroup := handler.Region.Region
	pager := handler.BackupsClient.NewListByServerPager(resourceGroup, rdbmsInfo.IId.SystemId, nil)

	// Get the first (most recent) backup
	if pager.More() {
		page, err := pager.NextPage(handler.Ctx)
		if err != nil {
			cblogger.Debug("Failed to retrieve backup list: ", err)
			return
		}

		if len(page.Value) > 0 {
			backup := page.Value[0]
			if backup.SystemData != nil && backup.SystemData.CreatedAt != nil {
				rdbmsInfo.BackupTime = backup.SystemData.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
			}
		}
	}
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
