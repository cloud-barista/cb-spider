package resources

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"strings"
	"time"
)

type AzureFileSystemHandler struct {
	CredentialInfo  idrv.CredentialInfo
	Region          idrv.RegionInfo
	Ctx             context.Context
	AccountsClient  *armstorage.AccountsClient
	FileShareClient *armstorage.FileSharesClient
	SubnetClient    *armnetwork.SubnetsClient
	VnetClient      *armnetwork.VirtualNetworksClient
}

func (af *AzureFileSystemHandler) AddAccessSubnet(fsIID irs.IID, subnetIID irs.IID) (irs.FileSystemInfo, error) {
	rg := af.Region.Region

	sa, err := af.getOrCreateStorageAccount()
	if err != nil {
		cblogger.Errorf("Failed to get storage account name: %v", err)
		return irs.FileSystemInfo{}, err
	}

	if subnetIID.SystemId == "" {
		cblogger.Infof("SystemId not provided. Finding VNet for subnet '%s'...", subnetIID.NameId)

		vnetPager := af.VnetClient.NewListPager(rg, nil)
		found := false

		for vnetPager.More() {
			page, err := vnetPager.NextPage(af.Ctx)
			if err != nil {
				return irs.FileSystemInfo{}, fmt.Errorf("failed to get next VNet page: %v", err)
			}

			for _, vnet := range page.Value {
				if vnet.Name == nil {
					continue
				}

				vnetName := *vnet.Name
				subnetPager := af.SubnetClient.NewListPager(rg, vnetName, nil)

				for subnetPager.More() {
					subnetPage, err := subnetPager.NextPage(af.Ctx)
					if err != nil {
						return irs.FileSystemInfo{}, fmt.Errorf("failed to get subnets: %v", err)
					}

					for _, subnet := range subnetPage.Value {
						if subnet.Name != nil && *subnet.Name == subnetIID.NameId {
							subnetIID.SystemId = *subnet.ID
							cblogger.Infof("Matched subnet. SystemId: %s", subnetIID.SystemId)
							found = true
							break
						}
					}
					if found {
						break
					}
				}
				if found {
					break
				}
			}
			if found {
				break
			}
		}

		if !found {
			return irs.FileSystemInfo{}, fmt.Errorf("failed to find VNet containing subnet '%s'", subnetIID.NameId)
		}

		cblogger.Infof("Constructed SystemId: %s", subnetIID.SystemId)
	}

	acctProps, err := af.AccountsClient.GetProperties(af.Ctx, rg, sa, nil)
	if err != nil {
		cblogger.Errorf("Failed to get storage account '%s': %v", sa, err)
		return irs.FileSystemInfo{}, err
	}

	nr := acctProps.Properties.NetworkRuleSet
	if nr == nil {
		cblogger.Infof("No existing NetworkRuleSet found. Creating new rule set with default Deny policy.")
		nr = &armstorage.NetworkRuleSet{
			DefaultAction:       to.Ptr(armstorage.DefaultActionDeny),
			IPRules:             []*armstorage.IPRule{},
			VirtualNetworkRules: []*armstorage.VirtualNetworkRule{},
		}
	} else {
		cblogger.Infof("Retrieved existing NetworkRuleSet. DefaultAction: %s", *nr.DefaultAction)
	}

	exists := false
	for _, r := range nr.VirtualNetworkRules {
		if r != nil && r.VirtualNetworkResourceID != nil && *r.VirtualNetworkResourceID == subnetIID.SystemId {
			exists = true
			cblogger.Infof("VNet rule for subnet '%s' already exists. Skipping addition.", subnetIID.SystemId)
			break
		}
	}
	if !exists {
		nr.VirtualNetworkRules = append(nr.VirtualNetworkRules, &armstorage.VirtualNetworkRule{
			VirtualNetworkResourceID: to.Ptr(subnetIID.SystemId),
			Action:                   to.Ptr(string(armstorage.DefaultActionAllow)),
		})
		cblogger.Infof("Added new VNet rule for subnet '%s'.", subnetIID.SystemId)
	}

	_, err = af.AccountsClient.Update(
		af.Ctx,
		rg,
		sa,
		armstorage.AccountUpdateParameters{
			Properties: &armstorage.AccountPropertiesUpdateParameters{
				NetworkRuleSet: nr,
			},
		},
		nil,
	)
	if err != nil {
		cblogger.Errorf("Failed to update storage account '%s': %v", sa, err)
		return irs.FileSystemInfo{}, err
	}
	cblogger.Infof("Successfully updated storage account '%s'.", sa)

	rawFS, err := af.getRawFileSystem(sa, fsIID)
	if err != nil {
		cblogger.Errorf("Failed to get raw file system: %v", err)
		return irs.FileSystemInfo{}, err
	}
	info, err := af.setterFileSystemInfo(&rawFS)
	if err != nil {
		cblogger.Errorf("Failed to set file system info: %v", err)
		return irs.FileSystemInfo{}, err
	}

	cblogger.Infof("Completed AddAccessSubnet operation successfully.")
	return *info, nil
}

func (af *AzureFileSystemHandler) RemoveAccessSubnet(iid irs.IID, subnetIID irs.IID) (bool, error) {
	rg := af.Region.Region

	saName, err := af.getOrCreateStorageAccount()
	if err != nil {
		return false, fmt.Errorf("failed to get storage account name: %v", err)
	}

	acctProps, err := af.AccountsClient.GetProperties(af.Ctx, rg, saName, nil)
	if err != nil {
		return false, fmt.Errorf("failed to get storage account properties: %v", err)
	}

	nr := acctProps.Properties.NetworkRuleSet
	if nr == nil || nr.VirtualNetworkRules == nil {
		return false, fmt.Errorf("no network rule set found for storage account")
	}

	rawFS, err := af.getRawFileSystem(saName, iid)
	if err != nil {
		cblogger.Errorf("Failed to get raw file system: %v", err)
		return false, err
	}
	info, err := af.setterFileSystemInfo(&rawFS)
	if err != nil {
		cblogger.Errorf("Failed to get file system info: %v", err)
		return false, err
	}

	if subnetIID.SystemId == "" {
		subnetIID.SystemId = GetSubnetIdByName(
			af.CredentialInfo,
			rg,
			info.VpcIID.NameId,
			subnetIID.NameId,
		)
		cblogger.Infof("Constructed SystemId for removal: %s", subnetIID.SystemId)
	}

	targetID := subnetIID.SystemId

	var newRules = make([]*armstorage.VirtualNetworkRule, 0)
	removed := false
	for _, rule := range nr.VirtualNetworkRules {
		if rule != nil && rule.VirtualNetworkResourceID != nil && *rule.VirtualNetworkResourceID == targetID {
			removed = true
			continue
		}
		newRules = append(newRules, rule)
	}

	if !removed {
		return false, fmt.Errorf("subnet %s not found in current access list", targetID)
	}

	nr.VirtualNetworkRules = newRules

	_, err = af.AccountsClient.Update(
		af.Ctx,
		rg,
		saName,
		armstorage.AccountUpdateParameters{
			Properties: &armstorage.AccountPropertiesUpdateParameters{
				NetworkRuleSet: nr,
			},
		},
		nil,
	)
	if err != nil {
		return false, fmt.Errorf("failed to update storage account: %v", err)
	}

	cblogger.Infof("Successfully removed subnet %s from access list", targetID)
	return true, nil
}

func (af *AzureFileSystemHandler) ListAccessSubnet(iid irs.IID) ([]irs.IID, error) {
	rg := af.Region.Region

	saName, err := af.getOrCreateStorageAccount()
	if err != nil {
		return nil, fmt.Errorf("failed to get storage account name: %v", err)
	}

	acctProps, err := af.AccountsClient.GetProperties(af.Ctx, rg, saName, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get storage account properties: %v", err)
	}

	nr := acctProps.Properties.NetworkRuleSet
	if nr == nil || nr.VirtualNetworkRules == nil {
		return []irs.IID{}, nil
	}

	var subnetList []irs.IID

	for _, rule := range nr.VirtualNetworkRules {
		if rule != nil && rule.VirtualNetworkResourceID != nil {
			split := strings.Split(*rule.VirtualNetworkResourceID, "/")
			var subnetName string
			if len(split) >= 11 {
				subnetName = split[10]
			}

			subnetList = append(subnetList, irs.IID{
				NameId:   subnetName,
				SystemId: *rule.VirtualNetworkResourceID,
			})
		}
	}

	return subnetList, nil
}

func (af *AzureFileSystemHandler) GetMetaInfo() (irs.FileSystemMetaInfo, error) {
	metaInfo := irs.FileSystemMetaInfo{
		SupportsFileSystemType: map[irs.FileSystemType]bool{
			irs.FileSystemType("RegionType"): true,
		},

		SupportsVPC: map[irs.RSType]bool{
			irs.RSType("VPC"): true,
		},

		SupportsNFSVersion: []string{"4.1"},

		SupportsCapacity: true,

		CapacityGBOptions: map[string]irs.CapacityGBRange{
			"Premium": {
				Min: 100,
				Max: 102400,
			},
		},

		PerformanceOptions: map[string][]string{
			"Tier": {"Premium", "Hot", "Cool", "TransactionOptimized"},
		},
	}

	return metaInfo, nil
}

func (af *AzureFileSystemHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(af.Region, call.FILESYSTEM, "FILESYSTEM", "ListIID()")
	start := call.Start()

	var iidList []*irs.IID

	resourceGroup := af.Region.Region
	saName, err := af.getOrCreateStorageAccount()
	if err != nil {
		return make([]*irs.IID, 0), fmt.Errorf("failed to get storage account name: %v", err)
	}

	sharePager := af.FileShareClient.NewListPager(resourceGroup, saName, nil)

	for sharePager.More() {
		sharePage, err := sharePager.NextPage(af.Ctx)
		if err != nil {
			cblogger.Warnf("Failed to list shares for %s: %v", saName, err)
			continue
		}

		for _, fs := range sharePage.Value {
			iid := &irs.IID{}
			if fs.ID != nil {
				iid.SystemId = *fs.ID
			}
			if fs.Name != nil {
				iid.NameId = *fs.Name
			}
			iidList = append(iidList, iid)
		}
	}

	LoggingInfo(hiscallInfo, start)

	return iidList, nil

}

func (af *AzureFileSystemHandler) getRawFileSystem(saName string, iid irs.IID) (armstorage.FileShare, error) {
	shareName := iid.NameId
	if shareName == "" {
		return armstorage.FileShare{}, fmt.Errorf("invalid IID: NameId is required")
	}

	rg := af.Region.Region
	sharePager := af.FileShareClient.NewListPager(rg, saName, nil)
	for sharePager.More() {
		sp, err := sharePager.NextPage(af.Ctx)
		if err != nil {
			continue
		}
		for _, item := range sp.Value {
			if item.Name != nil && *item.Name == shareName {
				return armstorage.FileShare{
					ID:                  item.ID,
					Name:                item.Name,
					FileShareProperties: item.Properties,
				}, nil
			}
		}
	}

	return armstorage.FileShare{}, fmt.Errorf("file share %q not found in resource group %q", shareName, rg)
}

func (af *AzureFileSystemHandler) GetFileSystem(iid irs.IID) (irs.FileSystemInfo, error) {
	hiscallInfo := GetCallLogScheme(af.Region, call.FILESYSTEM, "FILESYSTEM", "GetFileSystem()")
	start := call.Start()

	saName, err := af.getOrCreateStorageAccount()
	if err != nil {
		return irs.FileSystemInfo{}, fmt.Errorf("failed to get storage account name: %v", err)
	}

	fs, err := af.getRawFileSystem(saName, iid)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, fmt.Errorf("failed to get FileShare: %v", err)
	}

	info, err := af.setterFileSystemInfo(&fs)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, fmt.Errorf("failed to parse FileShare info: %v", err)
	}

	LoggingInfo(hiscallInfo, start)
	return *info, nil
}

func (af *AzureFileSystemHandler) setterFileSystemInfo(
	fs *armstorage.FileShare,
) (*irs.FileSystemInfo, error) {
	if fs == nil || fs.Name == nil || fs.ID == nil {
		return nil, fmt.Errorf("invalid FileShare input")
	}

	info := &irs.FileSystemInfo{
		IId: irs.IID{
			NameId:   *fs.Name,
			SystemId: *fs.ID,
		},
		Region:           af.Region.Region,
		Zone:             "",
		VpcIID:           irs.IID{},
		AccessSubnetList: nil,
		NFSVersion:       "4.1",
		FileSystemType:   irs.FileSystemType("RegionType"),
		Encryption:       true,
		CapacityGB:       0,
		PerformanceInfo:  map[string]string{},
		Status:           irs.FileSystemStatus("Available"),
		CreatedTime:      time.Time{},
	}

	if p := fs.FileShareProperties; p != nil {
		if p.ShareQuota != nil {
			info.CapacityGB = int64(*p.ShareQuota)
		}
		if p.LastModifiedTime != nil {
			info.CreatedTime = (*p.LastModifiedTime).UTC()
		}
		if p.AccessTier != nil {
			info.PerformanceInfo["Tier"] = string(*p.AccessTier)
		}
		if p.ShareUsageBytes != nil {
			info.UsedSizeGB = *p.ShareUsageBytes / (1024 * 1024 * 1024)
		}
	}

	rg := af.Region.Region
	saName, err := af.getOrCreateStorageAccount()
	if err == nil {
		acctProps, err := af.AccountsClient.GetProperties(af.Ctx, rg, saName, nil)
		if err == nil && acctProps.Properties.NetworkRuleSet != nil {
			var subnetList []irs.IID
			for _, rule := range acctProps.Properties.NetworkRuleSet.VirtualNetworkRules {
				if rule != nil && rule.VirtualNetworkResourceID != nil {
					split := strings.Split(*rule.VirtualNetworkResourceID, "/")
					if len(split) >= 11 {
						vnetName := split[8]
						subnetName := split[10]

						if info.VpcIID.NameId == "" {
							info.VpcIID.NameId = vnetName
							info.VpcIID.SystemId = strings.Join(split[:9], "/")
						}

						subnetList = append(subnetList, irs.IID{
							NameId:   subnetName,
							SystemId: *rule.VirtualNetworkResourceID,
						})
					}
				}
			}
			info.AccessSubnetList = subnetList
		}
	}

	// 키밸류
	baseKVs := irs.StructToKeyValueList(fs)
	var kvs []irs.KeyValue
	for _, kv := range baseKVs {
		if kv.Key == "FileShareProperties" {
			continue
		}
		kvs = append(kvs, kv)
	}
	if fs.FileShareProperties != nil {
		propKVs := irs.StructToKeyValueList(fs.FileShareProperties)
		kvs = append(kvs, propKVs...)
	}
	info.KeyValueList = kvs

	return info, nil
}

func (af *AzureFileSystemHandler) DeleteFileSystem(iid irs.IID) (bool, error) {
	rg := af.Region.Region
	fsName := iid.NameId
	if fsName == "" {
		return false, fmt.Errorf("invalid IID: NameId (file system name) is required")
	}

	saName, err := af.getOrCreateStorageAccount()
	if err != nil {
		return false, fmt.Errorf("failed to get storage account name: %v", err)
	}

	_, err = af.FileShareClient.Delete(af.Ctx, rg, saName, fsName, nil)
	if err != nil {
		return false, fmt.Errorf("failed to delete file share '%s' in storage account '%s': %v", fsName, saName, err)
	}

	cblogger.Infof("Successfully deleted file share '%s' in storage account '%s'", fsName, saName)
	return true, nil
}

func (af *AzureFileSystemHandler) createStorageAccount(accountName string) error {
	rg := af.Region.Region

	storageAccountParams := armstorage.AccountCreateParameters{
		Location: to.Ptr(rg),
		SKU: &armstorage.SKU{
			Name: to.Ptr(armstorage.SKUNamePremiumLRS),
		},
		Kind: to.Ptr(armstorage.KindFileStorage),
		Properties: &armstorage.AccountPropertiesCreateParameters{
			AllowBlobPublicAccess:  to.Ptr(false),
			AllowSharedKeyAccess:   to.Ptr(true),
			MinimumTLSVersion:      to.Ptr(armstorage.MinimumTLSVersionTLS12),
			EnableHTTPSTrafficOnly: to.Ptr(true),
			NetworkRuleSet: &armstorage.NetworkRuleSet{
				DefaultAction:       to.Ptr(armstorage.DefaultActionDeny),
				IPRules:             []*armstorage.IPRule{},
				VirtualNetworkRules: []*armstorage.VirtualNetworkRule{},
			},
		},
	}

	pollerResp, err := af.AccountsClient.BeginCreate(
		af.Ctx,
		rg,
		accountName,
		storageAccountParams,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to begin storage account creation: %v", err)
	}

	_, err = pollerResp.PollUntilDone(af.Ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to create storage account: %v", err)
	}

	cblogger.Infof("Successfully created storage account '%s' in resource group '%s'", accountName, rg)
	return nil
}

func (af *AzureFileSystemHandler) getOrCreateStorageAccount() (string, error) {
	rg := af.Region.Region
	accountName := AzureStorageAccountPrefix + af.Region.Region

	_, err := af.AccountsClient.GetProperties(af.Ctx, rg, accountName, nil)
	if err == nil {
		cblogger.Infof("Using existing storage account: %s", accountName)
		return accountName, nil
	}

	cblogger.Infof("Storage account '%s' not found. Creating new account...", accountName)

	err = af.createStorageAccount(accountName)
	if err != nil {
		return "", fmt.Errorf("failed to create storage account: %v", err)
	}

	cblogger.Infof("Storage account '%s' created.", accountName)

	return accountName, nil
}

func (af *AzureFileSystemHandler) CreateFileSystem(reqInfo irs.FileSystemInfo) (irs.FileSystemInfo, error) {

	hiscallInfo := GetCallLogScheme(
		af.Region, call.FILESYSTEM, af.Region.Region, "CreateFileSystem()",
	)
	start := call.Start()

	shareName := reqInfo.IId.NameId
	if shareName == "" {
		err := fmt.Errorf("invalid request: NameId is required")
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, err
	}
	rg := af.Region.Region

	saName, err := af.getOrCreateStorageAccount()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, err
	}

	props := armstorage.FileShareProperties{}

	if strings.EqualFold(reqInfo.NFSVersion, "4.1") {
		props.EnabledProtocols = to.Ptr(armstorage.EnabledProtocolsNFS)
	} else {
		return irs.FileSystemInfo{}, fmt.Errorf("unsupported NFS version: %q", reqInfo.NFSVersion)
	}

	if tier, ok := reqInfo.PerformanceInfo["Tier"]; ok {
		switch strings.ToLower(tier) {
		case "hot":
			props.AccessTier = to.Ptr(armstorage.ShareAccessTierHot)
		case "cool":
			props.AccessTier = to.Ptr(armstorage.ShareAccessTierCool)
		case "transactionoptimized":
			props.AccessTier = to.Ptr(armstorage.ShareAccessTierTransactionOptimized)
		}
	}

	_, err = af.FileShareClient.Create(
		af.Ctx,
		rg,
		saName,
		shareName,
		armstorage.FileShare{FileShareProperties: &props},
		nil,
	)

	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, fmt.Errorf("failed to create file share: %w", err)
	}

	for _, subnetIID := range reqInfo.AccessSubnetList {
		subnetInfo := irs.SubnetInfo{
			IId: subnetIID,
		}

		fsIID := irs.IID{SystemId: fmt.Sprintf(
			"/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Storage/storageAccounts/%s",
			af.CredentialInfo.SubscriptionId, rg, saName,
		)}

		if _, err := af.AddAccessSubnet(fsIID, subnetInfo.IId); err != nil {
			cblogger.Warnf("CreateFileSystem: AddAccessSubnet(%s) failed: %v",
				subnetIID.SystemId, err)
		}

	}

	fileSystemInfo, err := af.GetFileSystem(irs.IID{NameId: shareName})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, fmt.Errorf("created but get failed: %w", err)
	}

	LoggingInfo(hiscallInfo, start)
	return fileSystemInfo, nil
}

func (af *AzureFileSystemHandler) ListFileSystem() ([]*irs.FileSystemInfo, error) {
	hiscallInfo := GetCallLogScheme(
		af.Region, call.FILESYSTEM, af.Region.Region, "ListFileSystem()",
	)
	start := call.Start()

	var list []*irs.FileSystemInfo

	resourceGroup := af.Region.Region
	saName, err := af.getOrCreateStorageAccount()
	if err != nil {
		return make([]*irs.FileSystemInfo, 0), fmt.Errorf("failed to get storage account name: %v", err)
	}

	sharePager := af.FileShareClient.NewListPager(resourceGroup, saName, nil)
	for sharePager.More() {
		sp, err := sharePager.NextPage(af.Ctx)
		if err != nil {
			continue
		}
		for _, share := range sp.Value {
			raw := armstorage.FileShare{
				ID:                  share.ID,
				Name:                share.Name,
				FileShareProperties: share.Properties,
			}
			info, err := af.setterFileSystemInfo(&raw)
			if err != nil {
				cblogger.Warnf("ListFileSystem: setter error for %s/%s/%s: %v",
					resourceGroup, saName, *share.Name, err)
				continue
			}
			list = append(list, info)
		}
	}

	LoggingInfo(hiscallInfo, start)
	return list, nil
}

// Backup
func (af *AzureFileSystemHandler) ScheduleBackup(reqInfo irs.FileSystemBackupInfo) (irs.FileSystemBackupInfo, error) {
	return irs.FileSystemBackupInfo{}, nil
}
func (af *AzureFileSystemHandler) OnDemandBackup(fsIID irs.IID) (irs.FileSystemBackupInfo, error) {
	return irs.FileSystemBackupInfo{}, nil
}
func (af *AzureFileSystemHandler) ListBackup(fsIID irs.IID) ([]irs.FileSystemBackupInfo, error) {
	return nil, nil
}
func (af *AzureFileSystemHandler) GetBackup(fsIID irs.IID, backupID string) (irs.FileSystemBackupInfo, error) {
	return irs.FileSystemBackupInfo{}, nil
}
func (af *AzureFileSystemHandler) DeleteBackup(fsIID irs.IID, backupID string) (bool, error) {
	return false, nil
}
