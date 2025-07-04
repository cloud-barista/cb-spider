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
	var mountTargetList []irs.MountTargetInfo

	if rawIfaces, ok := raw["interfaces"].([]interface{}); ok {
		for _, ifaceRaw := range rawIfaces {
			iface, ok := ifaceRaw.(map[string]interface{})
			if !ok {
				continue
			}

			subnetId, _ := iface["subnetId"].(string)
			interfaceId, _ := iface["id"].(string)

			if subnetId == "" || interfaceId == "" {
				continue
			}

			// Get subnet name
			subnet, err := vpcsubnets.Get(nf.NetworkClient, subnetId).Extract()
			if err != nil {
				continue
			}

			accessSubnetList = append(accessSubnetList, irs.IID{
				NameId:   subnet.Name,
				SystemId: subnet.ID,
			})

			mountTargetList = append(mountTargetList, irs.MountTargetInfo{
				SubnetIID: irs.IID{
					NameId:   subnet.Name,
					SystemId: subnet.ID,
				},
				KeyValueList: []irs.KeyValue{
					{Key: "InterfaceId", Value: interfaceId},
				},
			})

			// Set VPC info only once
			if vpcIID.SystemId == "" && subnet.VPCID != "" {
				vpc, err := vpcs.Get(nf.NetworkClient, subnet.VPCID).Extract()
				if err == nil {
					vpcIID = irs.IID{
						NameId:   vpc.Name,
						SystemId: vpc.ID,
					}
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
		MountTargetList: mountTargetList,
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
	if iid.NameId == "" {
		return false, fmt.Errorf("NameId is required")
	}

	authToken, err := getTokenFromCredential(nf.CredentialInfo)
	if err != nil {
		return false, fmt.Errorf("failed to get token: %v", err)
	}

	region := strings.ToLower(nf.RegionInfo.Region)
	url := fmt.Sprintf("https://%s-api-nas-infrastructure.nhncloudservice.com/v1/volumes", region)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create list request: %v", err)
	}
	req.Header.Add("X-Auth-Token", authToken)
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to send list request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("failed to list NAS volumes: %s", string(body))
	}

	var result struct {
		Volumes []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"volumes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode NAS list response: %v", err)
	}

	var volumeID string
	for _, v := range result.Volumes {
		if v.Name == iid.NameId {
			volumeID = v.ID
			break
		}
	}
	if volumeID == "" {
		return false, fmt.Errorf("NAS not found with name: %s", iid.NameId)
	}

	deleteURL := fmt.Sprintf("https://%s-api-nas-infrastructure.nhncloudservice.com/v1/volumes/%s", region, volumeID)
	deleteReq, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create delete request: %v", err)
	}
	deleteReq.Header.Add("X-Auth-Token", authToken)
	deleteReq.Header.Add("Content-Type", "application/json")

	deleteResp, err := client.Do(deleteReq)
	if err != nil {
		return false, fmt.Errorf("failed to send delete request: %v", err)
	}
	defer deleteResp.Body.Close()

	if deleteResp.StatusCode >= 200 && deleteResp.StatusCode < 300 {
		cblogger.Infof("Successfully deleted file share '%s' in storage account '%s'", iid.NameId, volumeID)
		return true, nil
	}

	return false, fmt.Errorf("failed to delete NAS: status=%d, body=%s", deleteResp.StatusCode)
}

// Access Subnet Management
func (nf *NhnCloudFileSystemHandler) AddAccessSubnet(iid irs.IID, subnetIID irs.IID) (irs.FileSystemInfo, error) {

	cblogger.Info("Cloud driver: called AddAccessSubnet()")

	authToken, err := getTokenFromCredential(nf.CredentialInfo)
	if err != nil {
		cblogger.Errorf("Failed to get auth token: %v", err)
		return irs.FileSystemInfo{}, fmt.Errorf("failed to get auth token: %v", err)
	}

	subnetName := subnetIID.NameId
	subnet, err := GetVpcsubnetWithName(nf.NetworkClient, subnetName)
	if err != nil {
		return irs.FileSystemInfo{}, fmt.Errorf("failed to get subnet ID: %v", err)
	}

	region := strings.ToLower(nf.RegionInfo.Region)
	url := fmt.Sprintf("https://%s-api-nas-infrastructure.nhncloudservice.com/v1/volumes/%s/interfaces", region, iid.SystemId)

	cblogger.Infof("DEBUG: volumeId = %s", iid.SystemId)
	cblogger.Infof("DEBUG: subnetName = %s", subnetName)
	cblogger.Infof("DEBUG: subnetId = %s", subnet.ID)

	addAccessReq := map[string]interface{}{
		"interface": map[string]interface{}{
			"subnetId": subnet.ID,
			"mountProtocol": map[string]string{
				"protocol": "nfs",
			},
		},
	}

	jsonBody, err := json.Marshal(addAccessReq)
	if err != nil {
		cblogger.Errorf("Failed to marshal JSON: %v", err)
		return irs.FileSystemInfo{}, fmt.Errorf("failed to marshal JSON: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		cblogger.Errorf("Failed to create HTTP request: %v", err)
		return irs.FileSystemInfo{}, fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Auth-Token", authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		cblogger.Errorf("Failed to send HTTP request: %v", err)
		return irs.FileSystemInfo{}, fmt.Errorf("failed to send HTTP request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusCreated {
		bodyBytes, _ := io.ReadAll(resp.Body)
		cblogger.Errorf("AddAccessSubnet request failed: %s", string(bodyBytes))
		return irs.FileSystemInfo{}, fmt.Errorf("add AccessSubnet failed: %s", string(bodyBytes))
	}

	cblogger.Info("Successfully added access subnet to the volume")
	updatedFsInfo, err := nf.GetFileSystem(iid)
	if err != nil {
		cblogger.Warnf("Subnet added but failed to retrieve updated NAS info: %v", err)
		return irs.FileSystemInfo{IId: iid}, nil
	}
	return updatedFsInfo, nil
}

func (nf *NhnCloudFileSystemHandler) RemoveAccessSubnet(id irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger.Info("Cloud driver: called RemoveAccessSubnet()")

	callLogInfo := getCallLogScheme(nf.RegionInfo.Zone, call.FILESYSTEM, id.SystemId, "RemoveAccessSubnet()")
	start := call.Start()

	if id.SystemId == "" {
		cblogger.Error("FileSystem IID SystemId is empty")
		return false, fmt.Errorf("FileSystem IID SystemId is empty")
	}

	fsInfo, err := nf.GetFileSystem(id)
	if err != nil {
		cblogger.Errorf("Failed to get FileSystem info: %v", err)
		return false, err
	}

	var interfaceID string
	for _, mt := range fsInfo.MountTargetList {
		if mt.SubnetIID.NameId == subnetIID.NameId {
			for _, kv := range mt.KeyValueList {
				if kv.Key == "InterfaceId" || kv.Key == "interfaceId" {
					interfaceID = kv.Value
					break
				}
			}
		}
	}
	if interfaceID == "" {
		return false, fmt.Errorf("interface ID not found for subnet %s", subnetIID.NameId)
	}

	authToken, err := getTokenFromCredential(nf.CredentialInfo)
	if err != nil {
		cblogger.Errorf("Failed to get auth token: %v", err)
		return false, fmt.Errorf("failed to get auth token: %v", err)
	}

	region := strings.ToLower(nf.RegionInfo.Region)
	url := fmt.Sprintf("https://%s-api-nas-infrastructure.nhncloudservice.com/v1/volumes/%s/interfaces/%s", region, id.SystemId, interfaceID)
	cblogger.Infof("Request DELETE URL: %s", url)

	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		cblogger.Errorf("Failed to create DELETE request: %v", err)
		return false, err
	}
	req.Header.Set("X-Auth-Token", authToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		cblogger.Errorf("Failed to send DELETE request: %v", err)
		return false, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted && resp.StatusCode != http.StatusNoContent {
		cblogger.Errorf("RemoveAccessSubnet request failed: %s", string(bodyBytes))
		return false, fmt.Errorf("remove AccessSubnet failed: %s", string(bodyBytes))
	}

	cblogger.Infof("Successfully removed access subnet from the NAS volume")
	LoggingInfo(callLogInfo, start)
	return true, nil
}

func (nf *NhnCloudFileSystemHandler) ListAccessSubnet(iid irs.IID) ([]irs.IID, error) {
	cblogger.Info("Called ListAccessSubnet()")

	fsInfo, err := nf.GetFileSystem(iid)
	if err != nil {
		cblogger.Errorf("Failed to get file system info: %v", err)
		return nil, fmt.Errorf("failed to get file system info: %v", err)
	}

	return fsInfo.AccessSubnetList, nil
}

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
