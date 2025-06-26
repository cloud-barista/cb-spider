package resources

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"time"
)

type AzureFileSystemHandler struct {
	CredentialInfo  idrv.CredentialInfo
	Region          idrv.RegionInfo
	Ctx             context.Context
	AccountsClient  *armstorage.AccountsClient
	FileShareClient *armstorage.FileSharesClient
}

// Access Subnet
func (af *AzureFileSystemHandler) AddAccessSubnet(iid irs.IID, subnetIID irs.IID) (irs.FileSystemInfo, error) {

	return irs.FileSystemInfo{}, nil
}

func (af *AzureFileSystemHandler) RemoveAccessSubnet(iid irs.IID, subnetIID irs.IID) (bool, error) {
	return false, nil
}

func (af *AzureFileSystemHandler) ListAccessSubnet(iid irs.IID) ([]irs.IID, error) {
	return nil, nil
}

// File System
func (af *AzureFileSystemHandler) GetMetaInfo() (irs.FileSystemMetaInfo, error) {
	return irs.FileSystemMetaInfo{}, nil
}

func (af *AzureFileSystemHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(af.Region, call.FILESYSTEM, "FILESYSTEM", "ListIID()")
	start := call.Start()

	var iidList []*irs.IID

	cred, err := azidentity.NewClientSecretCredential(af.CredentialInfo.TenantId, af.CredentialInfo.ClientId, af.CredentialInfo.ClientSecret, nil)

	if err != nil {
		return nil, fmt.Errorf("failed to create credential: %w", err)
	}

	storageClient, err := armstorage.NewAccountsClient(af.CredentialInfo.SubscriptionId, cred, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage account client: %w", err)
	}

	pager := storageClient.NewListPager(nil)

	for pager.More() {
		page, err := pager.NextPage(af.Ctx)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return nil, fmt.Errorf("failed to list storage accounts: %w", err)
		}

		for _, acct := range page.Value {
			if acct.ID == nil || acct.Name == nil {
				continue
			}

			resourceGroup := af.Region.Region

			fmt.Printf("[INFO] StorageAccount Name: %s | ResourceGroup: %s\n", *acct.Name, resourceGroup)

			sharePager := af.FileShareClient.NewListPager(resourceGroup, *acct.Name, nil)

			for sharePager.More() {
				sharePage, err := sharePager.NextPage(af.Ctx)
				if err != nil {
					cblogger.Warnf("Failed to list shares for %s: %v", *acct.Name, err)
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
		}
	}
	LoggingInfo(hiscallInfo, start)
	return iidList, nil

}

func (af *AzureFileSystemHandler) getRawFileSystem(iid irs.IID) (armstorage.FileShare, error) {

	shareName := iid.NameId
	if shareName == "" {
		return armstorage.FileShare{}, fmt.Errorf("invalid IID: NameId is required")
	}

	rg := af.Region.Region

	acctPager := af.AccountsClient.NewListByResourceGroupPager(rg, nil)
	for acctPager.More() {
		page, err := acctPager.NextPage(af.Ctx)
		if err != nil {
			return armstorage.FileShare{}, fmt.Errorf("listing storage accounts failed: %w", err)
		}
		for _, acct := range page.Value {
			if acct.Name == nil {
				continue
			}
			sa := *acct.Name

			sharePager := af.FileShareClient.NewListPager(rg, sa, nil)
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
		}
	}

	return armstorage.FileShare{}, fmt.Errorf("file share %q not found in resource group %q", shareName, rg)
}

func (af *AzureFileSystemHandler) GetFileSystem(iid irs.IID) (irs.FileSystemInfo, error) {
	hiscallInfo := GetCallLogScheme(af.Region, call.FILESYSTEM, "FILESYSTEM", "GetFileSystem()")
	start := call.Start()

	fs, err := af.getRawFileSystem(iid)
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
	return false, nil
}

func (af *AzureFileSystemHandler) CreateFileSystem(reqInfo irs.FileSystemInfo) (irs.FileSystemInfo, error) {
	return irs.FileSystemInfo{}, nil
}

func (af *AzureFileSystemHandler) ListFileSystem() ([]*irs.FileSystemInfo, error) {
	hiscallInfo := GetCallLogScheme(
		af.Region, call.FILESYSTEM, af.Region.Region, "ListFileSystem()",
	)
	start := call.Start()

	rg := af.Region.Region
	var list []*irs.FileSystemInfo

	acctPager := af.AccountsClient.NewListByResourceGroupPager(rg, nil)
	for acctPager.More() {
		page, err := acctPager.NextPage(af.Ctx)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return nil, fmt.Errorf("ListFileSystem: list accounts failed: %w", err)
		}
		for _, acct := range page.Value {
			if acct.Name == nil {
				continue
			}
			sa := *acct.Name

			sharePager := af.FileShareClient.NewListPager(rg, sa, nil)
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
							rg, sa, *share.Name, err)
						continue
					}
					list = append(list, info)
				}
			}
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
