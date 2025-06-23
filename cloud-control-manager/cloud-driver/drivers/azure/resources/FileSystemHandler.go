package resources

import (
	"context"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"strings"
)

type AzureFileSystemHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	Ctx            context.Context
	//SubnetInfo     *armnetwork.SubnetsClient
	//ResourceGroup string
	//ARMClient       *armstorage.AccountsClient
	//StorageAccount  string
	FileShareClient *armstorage.FileSharesClient
	//ServiceClient   *service.Client
	//ShareClient     *share.Client
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

// ResourceGroup 추출
func extractResourceGroupFromID(id string) (string, error) {
	parts := strings.Split(id, "/")
	for i, part := range parts {
		if strings.EqualFold(part, "resourceGroups") && i+1 < len(parts) {
			return parts[i+1], nil
		}
	}
	return "", fmt.Errorf("resourceGroup not found in ID: %s", id)
}

// storageAcoount Name 추출
func extractStorageAccountNameFromID(id string) (string, error) {
	parts := strings.Split(id, "/")
	for i, part := range parts {
		if strings.EqualFold(part, "storageAccounts") && i+1 < len(parts) {
			return parts[i+1], nil
		}
	}
	return "", fmt.Errorf("storageAccount name not found in ID: %s", id)
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

			// 리소스 그룹 추출
			resourceGroup, err := extractResourceGroupFromID(*acct.ID)
			if err != nil {
				fmt.Printf("[WARN] Failed to extract resource group from ID: %s\n", *acct.ID)
				continue
			}

			fmt.Printf("[INFO] StorageAccount Name: %s | ResourceGroup: %s\n", *acct.Name, resourceGroup)

			// 해당 스토리지 계정의 파일 공유 목록 조회
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

func (af *AzureFileSystemHandler) setterFileSystemInfo(
	share *armstorage.FileShareItem,
	fsClient *armstorage.FileSharesClient,
	region idrv.RegionInfo,
	vpcIID irs.IID,
	ctx context.Context,
) (info irs.FileSystemInfo, err error) {

	//resp, err := fsClient.Get(
	//	ctx,
	//	af.ResourceGroup,
	//	af.StorageAccount,
	//	*share.Name,
	//	nil,
	//)
	//if err != nil {
	//	return info, err
	//}
	//full := resp.FileShare
	//
	//// 2) 필수 필드 채우기
	//info.IId = irs.IID{
	//	NameId:   *full.Name,
	//	SystemId: *full.ID,
	//}
	//info.VpcIID = vpcIID
	//info.NFSVersion = "4.1"
	//if full.FileShareProperties.Deleted != nil && *full.FileShareProperties.Deleted {
	//	info.Status = irs.FileSystemStatus("Deleted")
	//} else {
	//	info.Status = irs.FileSystemStatus("Available")
	//}
	//info.UsedSizeGB = *full.FileShareProperties.ShareUsageBytes
	//info.CreatedTime = full.FileShareProperties.LastModifiedTime.UTC()
	//
	//info.Region = region.Region

	return info, nil
}

func (af *AzureFileSystemHandler) GetFileSystem(iid irs.IID) (irs.FileSystemInfo, error) {

	return irs.FileSystemInfo{}, nil
}

func (af *AzureFileSystemHandler) DeleteFileSystem(iid irs.IID) (bool, error) {
	return false, nil
}

func (af *AzureFileSystemHandler) CreateFileSystem(reqInfo irs.FileSystemInfo) (irs.FileSystemInfo, error) {
	return irs.FileSystemInfo{}, nil
}

func (af *AzureFileSystemHandler) ListFileSystem() (listInfo []*irs.FileSystemInfo, getErr error) {
	//hiscallInfo := GetCallLogScheme(af.Region, call.FILESYSTEM, af.StorageAccount, "ListFileSystem()")
	//start := call.Start()
	//
	//var fsList []*armstorage.FileShareItem
	//
	//pager := af.FileShareClient.NewListPager(
	//	af.ResourceGroup,
	//	af.StorageAccount,
	//	nil, // *armstorage.FileSharesClientListOptions
	//)
	//
	//for pager.More() {
	//	page, err := pager.NextPage(af.Ctx)
	//	if err != nil {
	//		getErr = fmt.Errorf("Failed to List FileSystems: %w", err)
	//		cblogger.Error(getErr.Error())
	//		LoggingError(hiscallInfo, getErr)
	//		return nil, getErr
	//	}
	//	for _, fs := range page.Value {
	//		fsList = append(fsList, fs)
	//	}
	//}
	//
	//for _, fs := range fsList {
	//	fsInfo, err := af.setterFileSystemInfo(
	//		fs,
	//		af.FileShareClient,
	//		af.Region,
	//		af.VpcInfo.IId,
	//		af.Ctx,
	//	)
	//	if err != nil {
	//		getErr = fmt.Errorf("Failed to build FileSystemInfo: %w", err)
	//		cblogger.Error(getErr.Error())
	//		LoggingError(hiscallInfo, getErr)
	//		return nil, getErr
	//	}
	//	listInfo = append(listInfo, &fsInfo)
	//}
	//
	//LoggingInfo(hiscallInfo, start)
	return listInfo, nil
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
