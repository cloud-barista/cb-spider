package resources

import (
	"bytes"
	"encoding/json"
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	ostack "github.com/cloud-barista/nhncloud-sdk-go/openstack"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/vpcs"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/vpcsubnets"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type NhnCloudFileSystemHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	FSClient       *nhnsdk.ServiceClient
	NetworkClient  *nhnsdk.ServiceClient
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
		cblogger.Errorf("Failed to issue authentication token: %v", err)
		return "", fmt.Errorf("failed to issue authentication token: %w", err)
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
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"volumes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		cblogger.Errorf("Failed to parse NAS response: %v", err)
		return nil, fmt.Errorf("failed to parse NAS response: %w", err)
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
	metaInfo := irs.FileSystemMetaInfo{
		SupportsFileSystemType: map[irs.FileSystemType]bool{
			irs.FileSystemType("RegionType"): true,
			irs.FileSystemType("ZoneType"):   false,
		},

		SupportsVPC: map[irs.RSType]bool{
			irs.RSType("VPC"): true,
		},

		SupportsNFSVersion: []string{"4.1"},

		SupportsCapacity: true,
		CapacityGBOptions: map[string]irs.CapacityGBRange{
			"STANDARD": {
				Min: 300,
				Max: 10240,
			},
		},

		PerformanceOptions: map[string][]string{
			"STANDARD": {"Default"},
		},
	}

	return metaInfo, nil
}

func (nf *NhnCloudFileSystemHandler) getRawFileSystem(nameId string) (map[string]interface{}, error) {
	cblogger.Infof("Getting raw FileSystem for NameId: %s", nameId)
	if nameId == "" {
		cblogger.Error("NameId is required.")
		return nil, fmt.Errorf("nameId is required")
	}

	authToken, err := getTokenFromCredential(nf.CredentialInfo)
	if err != nil {
		cblogger.Errorf("Failed to get authentication token: %v", err)
		return nil, fmt.Errorf("failed to get authentication token: %w", err)
	}

	region := strings.ToLower(nf.RegionInfo.Region)
	url := fmt.Sprintf("https://%s-api-nas-infrastructure.nhncloudservice.com/v1/volumes", region)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		cblogger.Errorf("Failed to create HTTP request: %v", err)
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Add("X-Auth-Token", authToken)

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		cblogger.Errorf("Failed to query NAS list: %v", err)
		return nil, fmt.Errorf("failed to query NAS list: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			cblogger.Errorf("Error closing response body: %v", closeErr)
		}
	}()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		cblogger.Errorf("Failed to read NAS list response body: %v", readErr)
		return nil, fmt.Errorf("failed to read NAS list response body: %w", readErr)
	}

	if resp.StatusCode != http.StatusOK {
		cblogger.Errorf("NAS list API failed with status [%d]: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("NAS list API failed with status [%d]: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Volumes []map[string]interface{} `json:"volumes"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		cblogger.Errorf("Failed to parse JSON response: %v", err)
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	for _, vol := range result.Volumes {
		if name, ok := vol["name"].(string); ok && name == nameId {
			cblogger.Infof("Found raw FileSystem for NameId '%s'.", nameId)
			return vol, nil
		}
	}

	cblogger.Errorf("NAS not found for NameId '%s'.", nameId)
	return nil, fmt.Errorf("NAS not found for NameId '%s'", nameId)
}

func (nf *NhnCloudFileSystemHandler) setterFileSystemInfo(raw map[string]interface{}) (*irs.FileSystemInfo, error) {
	cblogger.Info("Setting FileSystem information from raw data.")
	if raw == nil {
		cblogger.Error("Invalid NAS raw input: nil.")
		return nil, fmt.Errorf("invalid NAS raw input: nil")
	}

	nameId, _ := raw["name"].(string)
	systemId, _ := raw["id"].(string)
	statusStr, _ := raw["status"].(string)

	var sizeGB int64
	if val, ok := raw["size"]; ok {
		switch v := val.(type) {
		case float64:
			sizeGB = int64(v)
		case int:
			sizeGB = int64(v)
		case string:
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				sizeGB = parsed
			} else {
				cblogger.Warnf("Failed to parse 'size' string to int64: %v", err)
			}
		default:
			cblogger.Warnf("Unknown type for 'size' field: %T", v)
		}
	}

	var accessSubnetList []irs.IID
	var vpcIID irs.IID

	if rawIfaces, ok := raw["interfaces"].([]interface{}); ok {
		for _, ifaceRaw := range rawIfaces {
			iface, ok := ifaceRaw.(map[string]interface{})
			if !ok {
				cblogger.Warn("Skipping non-map interface entry in raw data.")
				continue
			}
			subnetId, _ := iface["subnetId"].(string)
			if subnetId == "" {
				cblogger.Warn("Skipping interface with empty subnetId.")
				continue
			}

			cblogger.Infof("Querying subnet with ID: %s", subnetId)
			subnet, err := vpcsubnets.Get(nf.NetworkClient, subnetId).Extract()
			if err != nil {
				cblogger.Errorf("Failed to query subnet '%s': %v", subnetId, err)
				continue
			}

			accessSubnetList = append(accessSubnetList, irs.IID{
				NameId:   subnet.Name,
				SystemId: subnet.ID,
			})

			// 🔍 Query VPC
			if vpcIID.SystemId == "" && subnet.VPCID != "" {
				cblogger.Infof("Querying VPC with ID: %s", subnet.VPCID)
				vpc, err := vpcs.Get(nf.NetworkClient, subnet.VPCID).Extract()
				if err == nil {
					vpcIID = irs.IID{
						NameId:   vpc.Name,
						SystemId: vpc.ID,
					}
					cblogger.Infof("VPC '%s' found for subnet '%s'.", vpc.ID, subnetId)
				} else {
					cblogger.Errorf("Failed to query VPC '%s' for subnet '%s': %v", subnet.VPCID, subnetId, err)
				}
			}
		}
	}

	var kvs []irs.KeyValue
	for k, v := range raw {
		kvs = append(kvs, irs.KeyValue{
			Key:   k,
			Value: fmt.Sprintf("%v", v),
		})
	}

	fs := &irs.FileSystemInfo{
		IId:              irs.IID{NameId: nameId, SystemId: systemId},
		Region:           nf.RegionInfo.Region,
		Zone:             "",
		VpcIID:           vpcIID,
		AccessSubnetList: accessSubnetList,

		Encryption: func() bool {
			if enc, ok := raw["encryption"].(map[string]interface{}); ok {
				if enabled, ok := enc["enabled"].(bool); ok {
					return enabled
				}
			}
			return false
		}(),
		BackupSchedule:  irs.FileSystemBackupInfo{},
		TagList:         nil,
		FileSystemType:  irs.FileSystemType("RegionType"),
		NFSVersion:      "4.1",
		CapacityGB:      sizeGB,
		PerformanceInfo: map[string]string{"Tier": "STANDARD"},
		Status:          irs.FileSystemStatus(statusStr),
		UsedSizeGB:      0,
		MountTargetList: nil,
		CreatedTime:     irs.FileSystemInfo{}.CreatedTime,

		KeyValueList: kvs,
	}

	cblogger.Infof("Successfully set FileSystemInfo for '%s'.", nameId)
	return fs, nil
}

func (nf *NhnCloudFileSystemHandler) GetFileSystem(iid irs.IID) (irs.FileSystemInfo, error) {
	hiscallInfo := getCallLogScheme(nf.RegionInfo.Region, call.FILESYSTEM, "FILESYSTEM", "GetFileSystem()")
	start := call.Start()
	if iid.NameId == "" {
		return irs.FileSystemInfo{}, fmt.Errorf("NameId is required")
	}

	raw, err := nf.getRawFileSystem(iid.NameId)
	if err != nil {
		return irs.FileSystemInfo{}, err
	}

	fs, err := nf.setterFileSystemInfo(raw)
	if err != nil {
		return irs.FileSystemInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)
	return *fs, nil
}

func (nf *NhnCloudFileSystemHandler) ListFileSystem() ([]*irs.FileSystemInfo, error) {
	hiscallInfo := getCallLogScheme(nf.RegionInfo.Region, call.FILESYSTEM, "FILESYSTEM", "GetFileSystem()")
	start := call.Start()

	authToken, err := getTokenFromCredential(nf.CredentialInfo)
	if err != nil {
		cblogger.Errorf("Failed to get token: %v", err)
		return nil, fmt.Errorf("failed to get token: %v", err)
	}

	region := strings.ToLower(nf.RegionInfo.Region)
	url := fmt.Sprintf("https://%s-api-nas-infrastructure.nhncloudservice.com/v1/volumes", region)

	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("X-Auth-Token", authToken)

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		cblogger.Errorf("Failed to request NAS list: %v", err)
		return nil, fmt.Errorf("failed to request NAS list: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		cblogger.Errorf("Failed to get NAS list: %s", string(body))
		return nil, fmt.Errorf("failed to get NAS list (status: %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Volumes []map[string]interface{} `json:"volumes"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		cblogger.Errorf("Failed to parse NAS list response: %v", err)
		return nil, fmt.Errorf("failed to parse NAS list response: %v", err)
	}

	var fsList []*irs.FileSystemInfo
	for _, raw := range result.Volumes {
		fs, err := nf.setterFileSystemInfo(raw)
		if err != nil {
			cblogger.Warnf("Failed to convert NAS entry (skipped): %v", err)
			continue
		}
		fsList = append(fsList, fs)
	}

	LoggingInfo(hiscallInfo, start)
	return fsList, nil
}

func (nf *NhnCloudFileSystemHandler) CreateFileSystem(reqInfo irs.FileSystemInfo) (irs.FileSystemInfo, error) {
	hiscallInfo := getCallLogScheme(nf.RegionInfo.Region, call.FILESYSTEM, "FILESYSTEM", "CreateFileSystem()")
	start := call.Start()

	authToken, err := getTokenFromCredential(nf.CredentialInfo)
	if err != nil {
		cblogger.Errorf("Failed to get token: %v", err)
		return irs.FileSystemInfo{}, fmt.Errorf("failed to get token: %v", err)
	}

	if len(reqInfo.AccessSubnetList) == 0 {
		return irs.FileSystemInfo{}, fmt.Errorf("AccessSubnetList is empty")
	}

	subnetName := reqInfo.AccessSubnetList[0].NameId
	subnetInfo, err := GetVpcsubnetWithName(nf.NetworkClient, subnetName)
	if err != nil {
		return irs.FileSystemInfo{}, fmt.Errorf("failed to get subnet ID: %v", err)
	}

	subnetID := subnetInfo.ID

	region := strings.ToLower(nf.RegionInfo.Region)
	url := fmt.Sprintf("https://%s-api-nas-infrastructure.nhncloudservice.com/v1/volumes", region)

	createReq := map[string]interface{}{
		"volume": map[string]interface{}{
			"name":        reqInfo.IId.NameId,
			"description": "NAS created by cb-spider",
			"sizeGb":      reqInfo.CapacityGB,
			"encryption": map[string]bool{
				"enabled": reqInfo.Encryption,
			},
			"interfaces": []map[string]string{
				{
					"subnetId": subnetID,
				},
			},
			"mountProtocol": map[string]string{
				"protocol": "nfs",
			},
		},
	}

	jsonBody, err := json.Marshal(createReq)
	if err != nil {
		return irs.FileSystemInfo{}, fmt.Errorf("failed to marshal NAS create body: %v", err)
	}

	httpReq, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	httpReq.Header.Add("X-Auth-Token", authToken)
	httpReq.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(httpReq)
	if err != nil {
		return irs.FileSystemInfo{}, fmt.Errorf("failed to send NAS create request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return irs.FileSystemInfo{}, fmt.Errorf("NAS creation failed: %s", string(bodyBytes))
	}

	createdIID := irs.IID{NameId: reqInfo.IId.NameId}

	var fsInfo irs.FileSystemInfo
	var getErr error
	maxRetry := 10

	for i := 0; i < maxRetry; i++ {
		time.Sleep(1 * time.Second)

		fsInfo, getErr = nf.GetFileSystem(createdIID)
		if getErr == nil {
			return fsInfo, nil
		}

		cblogger.Infof("Waiting for NAS to become available... (%d/%d)", i+1, maxRetry)
	}

	LoggingInfo(hiscallInfo, start)
	return irs.FileSystemInfo{}, fmt.Errorf("NAS created but not found after retries: %v", getErr)
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
