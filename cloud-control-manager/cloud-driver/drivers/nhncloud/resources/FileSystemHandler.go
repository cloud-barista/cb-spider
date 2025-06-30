package resources

import (
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/sharedfilesystems/v2/shares"
)

type NhnCloudFileSystemHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	FSClient       *nhnsdk.ServiceClient
}

func (nf *NhnCloudFileSystemHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	callLogInfo := getCallLogScheme(nf.RegionInfo.Zone, call.FILESYSTEM, "fileSystemId", "ListIID()")
	start := call.Start()

	var iidList []*irs.IID

	pager := shares.ListDetail(nf.FSClient, shares.ListOpts{}) // 여기만 사용하면 됨

	allPages, err := pager.AllPages()
	if err != nil {
		return nil, fmt.Errorf("NAS 목록 조회 실패: %v", err)
	}

	allShares, err := shares.ExtractShares(allPages)
	if err != nil {
		return nil, fmt.Errorf("NAS 응답 파싱 실패: %v", err)
	}

	for _, sh := range allShares {
		iidList = append(iidList, &irs.IID{
			NameId:   sh.Name,
			SystemId: sh.ID,
		})
	}

	LoggingInfo(callLogInfo, start)
	return iidList, nil
}

func (nf *NhnCloudFileSystemHandler) GetMetaInfo() (irs.FileSystemMetaInfo, error) {
	return irs.FileSystemMetaInfo{}, nil
}

// File System Management
func (nf *NhnCloudFileSystemHandler) CreateFileSystem(reqInfo irs.FileSystemInfo) (irs.FileSystemInfo, error) {
	return irs.FileSystemInfo{}, nil
}
func (nf *NhnCloudFileSystemHandler) ListFileSystem() ([]*irs.FileSystemInfo, error) {
	return []*irs.FileSystemInfo{}, nil
}

func (nf *NhnCloudFileSystemHandler) GetFileSystem(iid irs.IID) (irs.FileSystemInfo, error) {
	return irs.FileSystemInfo{}, nil
}

func (nf *NhnCloudFileSystemHandler) DeleteFileSystem(iid irs.IID) (bool, error) {
	return false, nil
}

// Access Subnet Management
func (nf *NhnCloudFileSystemHandler) AddAccessSubnet(iid irs.IID, subnetIID irs.IID) (irs.FileSystemInfo, error) {
	return irs.FileSystemInfo{}, nil
} // Add a subnet to the file system for access; creates a mount target in the driver if needed
func (nf *NhnCloudFileSystemHandler) RemoveAccessSubnet(id irs.IID, subnetIID irs.IID) (bool, error) {
	return false, nil
} // Remove a subnet from the file system access list; deletes the mount target if needed
func (nf *NhnCloudFileSystemHandler) ListAccessSubnet(iid irs.IID) ([]irs.IID, error) {
	return []irs.IID{}, nil
} // List of subnets whose VMs can use this file system

// Backup Management
func (nf *NhnCloudFileSystemHandler) ScheduleBackup(reqInfo irs.FileSystemBackupInfo) (irs.FileSystemBackupInfo, error) {
	return irs.FileSystemBackupInfo{}, nil
} // Create a backup with the specified schedule
func (nf *NhnCloudFileSystemHandler) OnDemandBackup(fsIID irs.IID) (irs.FileSystemBackupInfo, error) {
	return irs.FileSystemBackupInfo{}, nil
} // Create an on-demand backup for the specified file system
func (nf *NhnCloudFileSystemHandler) ListBackup(fsIID irs.IID) ([]irs.FileSystemBackupInfo, error) {
	return []irs.FileSystemBackupInfo{}, nil
}
func (nf *NhnCloudFileSystemHandler) GetBackup(fsIID irs.IID, backupID string) (irs.FileSystemBackupInfo, error) {
	return irs.FileSystemBackupInfo{}, nil
}
func (nf *NhnCloudFileSystemHandler) DeleteBackup(fsIID irs.IID, backupID string) (bool, error) {
	return false, nil
}
