package resources

import (
	"encoding/json"
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	ostack "github.com/cloud-barista/nhncloud-sdk-go/openstack"
	"io"
	"net/http"
	"strings"
)

type NhnCloudFileSystemHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	FSClient       *nhnsdk.ServiceClient
}

func getTokenFromCredential(cred idrv.CredentialInfo) (string, error) {
	authOpts := nhnsdk.AuthOptions{
		IdentityEndpoint: cred.IdentityEndpoint,
		Username:         cred.Username,
		Password:         cred.Password,
		DomainName:       cred.DomainName,
		TenantID:         cred.TenantId,
	}

	provider, err := ostack.AuthenticatedClient(authOpts)
	if err != nil {
		return "", fmt.Errorf("인증 토큰 발급 실패: %v", err)
	}

	return provider.TokenID, nil
}

func (nf *NhnCloudFileSystemHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()")

	callLogInfo := getCallLogScheme(nf.RegionInfo.Zone, call.FILESYSTEM, "volumeId", "ListIID()")
	start := call.Start()

	var iidList []*irs.IID

	authToken, err := getTokenFromCredential(nf.CredentialInfo)
	if err != nil {
		return nil, fmt.Errorf("토큰 발급 실패: %v", err)
	}

	region := strings.ToLower(nf.RegionInfo.Region)
	url := fmt.Sprintf("https://%s-api-nas-infrastructure.nhncloudservice.com/v1/volumes", region)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("요청 생성 실패: %v", err)
	}
	req.Header.Add("X-Auth-Token", authToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("NAS 목록 조회 실패: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("NAS 목록 응답 오류 [%d]: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Volumes []struct {
			ID   string `json:"volumeId"`
			Name string `json:"volumeName"`
		} `json:"volumes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("NAS 응답 파싱 실패: %v", err)
	}

	for _, vol := range result.Volumes {
		iidList = append(iidList, &irs.IID{
			NameId:   vol.Name,
			SystemId: vol.ID,
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
