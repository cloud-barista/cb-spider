package resources

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"
	"github.com/gophercloud/gophercloud/openstack/sharedfilesystems/v2/sharenetworks"
	"github.com/gophercloud/gophercloud/openstack/sharedfilesystems/v2/shares"
	"github.com/gophercloud/gophercloud/openstack/sharedfilesystems/v2/sharetypes"
	"io"
	"net/http"
	"strings"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type OpenstackFileSystemHandler struct {
	Region                 idrv.RegionInfo
	CredentialInfo         idrv.CredentialInfo
	SharedFileSystemClient *gophercloud.ServiceClient
	NetworkClient          *gophercloud.ServiceClient
	ComputeClient          *gophercloud.ServiceClient
}

const (
	DefaultShareType       = "cb-spider-share-type"
	DefaultShareProtocol   = "NFS"
	DefaultNFSVersion      = "4.1"
	ShareNetworkNamePrefix = "cb-spider-share-network-"
)

type QuotaUsage struct {
	TotalQuota   int64
	UsedCapacity int64
}

func (filesystemHandler *OpenstackFileSystemHandler) getQuotaUsage() (*QuotaUsage, error) {
	url := strings.TrimSuffix(filesystemHandler.SharedFileSystemClient.Endpoint, "/") + "/limits"

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-Auth-Token", filesystemHandler.SharedFileSystemClient.TokenID)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get shared filesystem quota: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var limitsResp struct {
		Limits struct {
			Absolute struct {
				MaxTotalShareGigabytes  int `json:"maxTotalShareGigabytes"`
				TotalShareGigabytesUsed int `json:"totalShareGigabytesUsed"`
			} `json:"absolute"`
		} `json:"limits"`
	}

	err = json.Unmarshal(body, &limitsResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode quota response: %v", err)
	}

	listOpts := shares.ListOpts{}
	allPages, err := shares.ListDetail(filesystemHandler.SharedFileSystemClient, listOpts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("failed to list shares: %v", err)
	}

	shareList, err := shares.ExtractShares(allPages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract shares: %v", err)
	}

	var usedCapacity int64
	for _, share := range shareList {
		usedCapacity += int64(share.Size)
	}

	return &QuotaUsage{
		TotalQuota:   int64(limitsResp.Limits.Absolute.MaxTotalShareGigabytes),
		UsedCapacity: int64(limitsResp.Limits.Absolute.TotalShareGigabytesUsed),
	}, nil
}

func (filesystemHandler *OpenstackFileSystemHandler) GetMetaInfo() (irs.FileSystemMetaInfo, error) {
	availableCapacity := int64(0)

	quotaUsage, err := filesystemHandler.getQuotaUsage()
	if err != nil {
		cblogger.Warnf("Failed to get quota usage %v", err)
	} else {
		totalQuota := quotaUsage.TotalQuota
		usedCapacity := quotaUsage.UsedCapacity

		if totalQuota > 0 {
			remainingCapacity := totalQuota - usedCapacity
			if remainingCapacity > 0 {
				availableCapacity = remainingCapacity
			}
		}
	}

	metaInfo := irs.FileSystemMetaInfo{
		SupportsFileSystemType: map[irs.FileSystemType]bool{
			irs.FileSystemType("nfs"): true,
		},

		SupportsVPC: map[irs.RSType]bool{
			irs.RSType("VPC"): true,
		},

		SupportsNFSVersion: []string{"4.1"},

		SupportsCapacity: true,

		CapacityGBOptions: map[string]irs.CapacityGBRange{
			"Standard": {
				Min: 1,
				Max: availableCapacity,
			},
		},

		PerformanceOptions: map[string][]string{},
	}

	return metaInfo, nil
}

func (filesystemHandler *OpenstackFileSystemHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(filesystemHandler.NetworkClient.IdentityEndpoint, call.FILESYSTEM, "FILESYSTEM", "ListIID()")
	start := call.Start()

	listOpts := shares.ListOpts{}
	allPages, err := shares.ListDetail(filesystemHandler.SharedFileSystemClient, listOpts).AllPages()
	if err != nil {
		return nil, err
	}

	shareList, err := shares.ExtractShares(allPages)
	if err != nil {
		return nil, err
	}

	var iidList []*irs.IID
	for _, share := range shareList {
		if filesystemHandler.Region.Zone != "" && share.AvailabilityZone != filesystemHandler.Region.Zone {
			continue
		}

		iid := &irs.IID{
			NameId:   share.Name,
			SystemId: share.ID,
		}
		iidList = append(iidList, iid)
	}

	LoggingInfo(hiscallInfo, start)
	return iidList, nil
}

func (filesystemHandler *OpenstackFileSystemHandler) findNetworkByName(name string) (string, error) {
	listOpts := networks.ListOpts{
		Name: name,
	}
	allPages, err := networks.List(filesystemHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		return "", err
	}

	networkList, err := networks.ExtractNetworks(allPages)
	if err != nil {
		return "", err
	}

	if len(networkList) == 0 {
		return "", fmt.Errorf("network with name %s not found", name)
	}

	return networkList[0].ID, nil
}

func (filesystemHandler *OpenstackFileSystemHandler) findSubnetByName(name string) (string, error) {
	listOpts := subnets.ListOpts{
		Name: name,
	}
	allPages, err := subnets.List(filesystemHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		return "", err
	}

	subnetList, err := subnets.ExtractSubnets(allPages)
	if err != nil {
		return "", err
	}

	if len(subnetList) == 0 {
		return "", fmt.Errorf("subnet with name %s not found", name)
	}

	return subnetList[0].ID, nil
}

func (filesystemHandler *OpenstackFileSystemHandler) findSubnetsByNetwork(networkIID irs.IID) ([]irs.IID, error) {
	var networkID string

	if networkIID.SystemId != "" {
		networkID = networkIID.SystemId
	} else if networkIID.NameId != "" {
		id, err := filesystemHandler.findNetworkByName(networkIID.NameId)
		if err != nil {
			return nil, err
		}
		networkID = id
	}

	listOpts := subnets.ListOpts{
		NetworkID: networkID,
	}
	allPages, err := subnets.List(filesystemHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		return nil, err
	}

	subnetList, err := subnets.ExtractSubnets(allPages)
	if err != nil {
		return nil, err
	}

	var subnetIIDList []irs.IID
	for _, subnet := range subnetList {
		subnetIID := irs.IID{
			NameId:   subnet.Name,
			SystemId: subnet.ID,
		}
		subnetIIDList = append(subnetIIDList, subnetIID)
	}

	return subnetIIDList, nil
}

func (filesystemHandler *OpenstackFileSystemHandler) getOrCreateShareNetwork(networkID, subnetID string) (string, error) {
	listOpts := sharenetworks.ListOpts{
		NeutronNetID:    networkID,
		NeutronSubnetID: subnetID,
	}

	allPages, err := sharenetworks.ListDetail(filesystemHandler.SharedFileSystemClient, listOpts).AllPages()
	if err != nil {
		return "", err
	}

	shareNetworkList, err := sharenetworks.ExtractShareNetworks(allPages)
	if err != nil {
		return "", err
	}

	if len(shareNetworkList) > 0 {
		return shareNetworkList[0].ID, nil
	}

	createOpts := sharenetworks.CreateOpts{
		Name:            ShareNetworkNamePrefix + networkID + "-" + subnetID,
		Description:     "Share network created by CB-Spider",
		NeutronNetID:    networkID,
		NeutronSubnetID: subnetID,
	}

	shareNetwork, err := sharenetworks.Create(filesystemHandler.SharedFileSystemClient, createOpts).Extract()
	if err != nil {
		return "", err
	}

	return shareNetwork.ID, nil
}

func (filesystemHandler *OpenstackFileSystemHandler) getAvailableBackend() (string, error) {
	url := strings.TrimSuffix(filesystemHandler.SharedFileSystemClient.Endpoint, "/") + "/scheduler-stats/pools"

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Add("X-Auth-Token", filesystemHandler.SharedFileSystemClient.TokenID)

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var poolsResp struct {
		Pools []struct {
			Name         string `json:"name"`
			Host         string `json:"host"`
			Capabilities struct {
				ShareBackendName string `json:"share_backend_name"`
			} `json:"capabilities"`
		} `json:"pools"`
	}

	body, _ := io.ReadAll(resp.Body)
	err = json.Unmarshal(body, &poolsResp)
	if err != nil {
		cblogger.Warnf("Failed to unmarshal pools response: %v", err)
		return "", err
	}

	if len(poolsResp.Pools) > 0 {
		pool := poolsResp.Pools[0]

		if pool.Capabilities.ShareBackendName != "" {
			return pool.Capabilities.ShareBackendName, nil
		}

		if strings.Contains(pool.Name, "#") {
			parts := strings.Split(pool.Name, "#")
			if len(parts) > 1 {
				return parts[1], nil
			}
		}

		if pool.Host != "" {
			return "GENERIC", nil
		}
	}

	return "GENERIC", nil // fallback
}

func (filesystemHandler *OpenstackFileSystemHandler) getOrCreateShareType() (string, error) {
	listOpts := sharetypes.ListOpts{
		IsPublic: "true",
	}

	allPages, err := sharetypes.List(filesystemHandler.SharedFileSystemClient, listOpts).AllPages()
	if err != nil {
		return "", fmt.Errorf("failed to list share types: %v", err)
	}

	shareTypeList, err := sharetypes.ExtractShareTypes(allPages)
	if err != nil {
		return "", fmt.Errorf("failed to extract share types: %v", err)
	}

	for _, shareType := range shareTypeList {
		if shareType.Name == DefaultShareType {
			return shareType.ID, nil
		}
	}

	backendName, err := filesystemHandler.getAvailableBackend()
	if err != nil {
		backendName = "GENERIC" // fallback
	}

	requestBody := map[string]interface{}{
		"share_type": map[string]interface{}{
			"name":                           DefaultShareType,
			"os-share-type-access:is_public": true,
			"extra_specs": map[string]interface{}{
				"share_backend_name":           backendName,
				"driver_handles_share_servers": "True",
				"snapshot_support":             "True",
			},
		},
	}

	var result sharetypes.CreateResult
	resp, err := filesystemHandler.SharedFileSystemClient.Post(
		filesystemHandler.SharedFileSystemClient.Endpoint+"/types",
		requestBody,
		&result.Body,
		&gophercloud.RequestOpts{
			OkCodes: []int{200, 202},
		},
	)
	_, result.Header, result.Err = gophercloud.ParseResponse(resp, err)

	if result.Err != nil {
		return "", fmt.Errorf("failed to create share type: %v", result.Err)
	}

	shareType, err := result.Extract()
	if err != nil {
		return "", fmt.Errorf("failed to extract created share type: %v", err)
	}

	return shareType.ID, nil
}

func (filesystemHandler *OpenstackFileSystemHandler) grantShareAccess(shareID string) error {
	url := strings.TrimSuffix(filesystemHandler.SharedFileSystemClient.Endpoint, "/") + "/shares/" + shareID + "/action"

	requestBody := map[string]interface{}{
		"os-allow_access": map[string]interface{}{
			"access_type":  "ip",
			"access_to":    "0.0.0.0/0",
			"access_level": "rw",
		},
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	jsonData, _ := json.Marshal(requestBody)
	req, err := http.NewRequest("POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return err
	}

	req.Header.Add("X-Auth-Token", filesystemHandler.SharedFileSystemClient.TokenID)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("X-OpenStack-Manila-API-Version", "2.1")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != 200 && resp.StatusCode != 202 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to grant access: %d %s", resp.StatusCode, string(body))
	}

	return nil
}

func (filesystemHandler *OpenstackFileSystemHandler) CreateFileSystem(reqInfo irs.FileSystemInfo) (irs.FileSystemInfo, error) {
	hiscallInfo := GetCallLogScheme(filesystemHandler.NetworkClient.IdentityEndpoint, call.FILESYSTEM, filesystemHandler.Region.Region, "CreateFileSystem()")
	start := call.Start()

	if reqInfo.IId.NameId == "" {
		err := fmt.Errorf("invalid request: NameId is required")
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, err
	}

	if !strings.EqualFold(reqInfo.NFSVersion, "4.1") {
		err := fmt.Errorf("unsupported NFS version: %q", reqInfo.NFSVersion)
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, err
	}

	capacity := reqInfo.CapacityGB
	if capacity <= 0 {
		capacity = 1
	}

	var shareNetworkID string
	if len(reqInfo.AccessSubnetList) > 0 {
		subnetIID := reqInfo.AccessSubnetList[0]
		var subnetID string

		if subnetIID.SystemId != "" {
			subnetID = subnetIID.SystemId
		} else if subnetIID.NameId != "" {
			id, err := filesystemHandler.findSubnetByName(subnetIID.NameId)
			if err != nil {
				LoggingError(hiscallInfo, err)
				return irs.FileSystemInfo{}, fmt.Errorf("failed to find subnet: %v", err)
			}
			subnetID = id
		}

		subnet, err := subnets.Get(filesystemHandler.NetworkClient, subnetID).Extract()
		if err != nil {
			LoggingError(hiscallInfo, err)
			return irs.FileSystemInfo{}, fmt.Errorf("failed to get subnet info: %v", err)
		}

		shareNetworkID, err = filesystemHandler.getOrCreateShareNetwork(subnet.NetworkID, subnetID)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return irs.FileSystemInfo{}, fmt.Errorf("failed to create share network: %v", err)
		}
	} else if reqInfo.VpcIID.NameId != "" || reqInfo.VpcIID.SystemId != "" {
		err := fmt.Errorf("VPC is specified but no subnet provided. Please specify exactly one subnet from the VPC")
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, err
	} else {
		err := fmt.Errorf("either VPC with subnet or subnet must be specified")
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, err
	}

	shareTypeID, err := filesystemHandler.getOrCreateShareType()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, fmt.Errorf("failed to get or create share type: %v", err)
	}

	createOpts := shares.CreateOpts{
		Size:           int(capacity),
		Name:           reqInfo.IId.NameId,
		Description:    "FileSystem created by CB-Spider",
		ShareProto:     DefaultShareProtocol,
		ShareType:      shareTypeID,
		ShareNetworkID: shareNetworkID,
	}

	if reqInfo.Zone != "" {
		createOpts.AvailabilityZone = reqInfo.Zone
	}

	share, err := shares.Create(filesystemHandler.SharedFileSystemClient, createOpts).Extract()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, fmt.Errorf("failed to create file share: %v", err)
	}

	for i := 0; i < 240; i++ {
		currentShare, err := shares.Get(filesystemHandler.SharedFileSystemClient, share.ID).Extract()
		if err != nil {
			LoggingError(hiscallInfo, err)
			return irs.FileSystemInfo{}, fmt.Errorf("failed to check share status: %v", err)
		}

		if currentShare.Status == "available" {
			break
		} else if currentShare.Status == "error" {
			return irs.FileSystemInfo{}, fmt.Errorf("share creation failed with error status")
		}

		time.Sleep(5 * time.Second)
	}

	err = filesystemHandler.grantShareAccess(share.ID)
	if err != nil {
		cblogger.Warnf("Failed to grant access rule: %v", err)
	}

	fileSystemInfo, err := filesystemHandler.GetFileSystem(irs.IID{
		NameId:   share.Name,
		SystemId: share.ID,
	})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, fmt.Errorf("created but get failed: %v", err)
	}

	LoggingInfo(hiscallInfo, start)
	return fileSystemInfo, nil
}

func (filesystemHandler *OpenstackFileSystemHandler) ListFileSystem() ([]*irs.FileSystemInfo, error) {
	hiscallInfo := GetCallLogScheme(filesystemHandler.NetworkClient.IdentityEndpoint, call.FILESYSTEM, filesystemHandler.Region.Region, "ListFileSystem()")
	start := call.Start()

	listOpts := shares.ListOpts{}
	allPages, err := shares.ListDetail(filesystemHandler.SharedFileSystemClient, listOpts).AllPages()
	if err != nil {
		return nil, err
	}

	shareList, err := shares.ExtractShares(allPages)
	if err != nil {
		return nil, err
	}

	var list []*irs.FileSystemInfo
	for _, share := range shareList {
		if filesystemHandler.Region.Zone != "" && share.AvailabilityZone != filesystemHandler.Region.Zone {
			continue
		}

		info, err := filesystemHandler.setterFileSystemInfo(&share)
		if err != nil {
			cblogger.Warnf("ListFileSystem: setter error for %s: %v", share.Name, err)
			continue
		}
		list = append(list, info)
	}

	LoggingInfo(hiscallInfo, start)
	return list, nil
}

func (filesystemHandler *OpenstackFileSystemHandler) GetFileSystem(iid irs.IID) (irs.FileSystemInfo, error) {
	hiscallInfo := GetCallLogScheme(filesystemHandler.NetworkClient.IdentityEndpoint, call.FILESYSTEM, "FILESYSTEM", "GetFileSystem()")
	start := call.Start()

	var shareID string
	if iid.SystemId != "" {
		shareID = iid.SystemId
	} else if iid.NameId != "" {
		listOpts := shares.ListOpts{
			Name: iid.NameId,
		}
		allPages, err := shares.ListDetail(filesystemHandler.SharedFileSystemClient, listOpts).AllPages()
		if err != nil {
			LoggingError(hiscallInfo, err)
			return irs.FileSystemInfo{}, fmt.Errorf("failed to list shares: %v", err)
		}

		shareList, err := shares.ExtractShares(allPages)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return irs.FileSystemInfo{}, fmt.Errorf("failed to extract shares: %v", err)
		}

		if len(shareList) == 0 {
			err := fmt.Errorf("file share with name %s not found", iid.NameId)
			LoggingError(hiscallInfo, err)
			return irs.FileSystemInfo{}, err
		}

		shareID = shareList[0].ID
	} else {
		err := fmt.Errorf("invalid IID: either NameId or SystemId is required")
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, err
	}

	share, err := shares.Get(filesystemHandler.SharedFileSystemClient, shareID).Extract()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, fmt.Errorf("failed to get file share: %v", err)
	}

	info, err := filesystemHandler.setterFileSystemInfo(share)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, fmt.Errorf("failed to parse file share info: %v", err)
	}

	LoggingInfo(hiscallInfo, start)
	return *info, nil
}

func (filesystemHandler *OpenstackFileSystemHandler) findNetworkByID(networkID string) (*networks.Network, error) {
	network, err := networks.Get(filesystemHandler.NetworkClient, networkID).Extract()
	if err != nil {
		return nil, err
	}
	return network, nil
}

func (filesystemHandler *OpenstackFileSystemHandler) findSubnetByID(subnetID string) (*subnets.Subnet, error) {
	subnet, err := subnets.Get(filesystemHandler.NetworkClient, subnetID).Extract()
	if err != nil {
		return nil, err
	}
	return subnet, nil
}

func (filesystemHandler *OpenstackFileSystemHandler) setterFileSystemInfo(share *shares.Share) (*irs.FileSystemInfo, error) {
	if share == nil {
		return nil, fmt.Errorf("invalid Share input")
	}

	info := &irs.FileSystemInfo{
		IId: irs.IID{
			NameId:   share.Name,
			SystemId: share.ID,
		},
		Region:           filesystemHandler.Region.Region,
		Zone:             share.AvailabilityZone,
		VpcIID:           irs.IID{},
		AccessSubnetList: nil,
		NFSVersion:       DefaultNFSVersion,
		FileSystemType:   irs.FileSystemType("nfs"),
		Encryption:       false,
		CapacityGB:       int64(share.Size),
		PerformanceInfo:  map[string]string{},
		Status:           irs.FileSystemStatus("Unknown"),
		CreatedTime:      share.CreatedAt,
	}

	switch share.Status {
	case "available":
		info.Status = irs.FileSystemAvailable
	case "creating":
		info.Status = irs.FileSystemCreating
	case "deleting":
		info.Status = irs.FileSystemDeleting
	case "error":
		info.Status = irs.FileSystemError
	default:
		info.Status = irs.FileSystemStatus(share.Status)
	}

	if share.ShareNetworkID != "" {
		shareNetwork, err := sharenetworks.Get(filesystemHandler.SharedFileSystemClient, share.ShareNetworkID).Extract()
		if err == nil {
			if shareNetwork.NeutronNetID != "" {
				network, err := filesystemHandler.findNetworkByID(shareNetwork.NeutronNetID)
				if err == nil {
					info.VpcIID = irs.IID{
						NameId:   network.Name,
						SystemId: network.ID,
					}
				}
			}

			if shareNetwork.NeutronSubnetID != "" {
				subnet, err := filesystemHandler.findSubnetByID(shareNetwork.NeutronSubnetID)
				if err == nil {
					subnetIID := irs.IID{
						NameId:   subnet.Name,
						SystemId: subnet.ID,
					}
					info.AccessSubnetList = []irs.IID{subnetIID}
				}
			}
		}
	}

	baseKVs := irs.StructToKeyValueList(share)
	info.KeyValueList = baseKVs

	return info, nil
}

func (filesystemHandler *OpenstackFileSystemHandler) DeleteFileSystem(iid irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(filesystemHandler.NetworkClient.IdentityEndpoint, call.FILESYSTEM, "FILESYSTEM", "DeleteFileSystem()")
	start := call.Start()

	shareID := iid.SystemId
	if shareID == "" && iid.NameId != "" {
		listOpts := shares.ListOpts{
			Name: iid.NameId,
		}
		allPages, err := shares.ListDetail(filesystemHandler.SharedFileSystemClient, listOpts).AllPages()
		if err != nil {
			LoggingError(hiscallInfo, err)
			return false, fmt.Errorf("failed to list shares: %v", err)
		}

		shareList, err := shares.ExtractShares(allPages)
		if err != nil {
			LoggingError(hiscallInfo, err)
			return false, fmt.Errorf("failed to extract shares: %v", err)
		}

		if len(shareList) == 0 {
			err := fmt.Errorf("file share with name %s not found", iid.NameId)
			LoggingError(hiscallInfo, err)
			return false, err
		}

		shareID = shareList[0].ID
	}

	if shareID == "" {
		err := fmt.Errorf("invalid IID: filesystem not found")
		LoggingError(hiscallInfo, err)
		return false, err
	}

	filesystemInfo, err := filesystemHandler.GetFileSystem(irs.IID{SystemId: shareID})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}

	err = shares.Delete(filesystemHandler.SharedFileSystemClient, shareID).ExtractErr()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, fmt.Errorf("failed to delete file share: %v", err)
	}

	for i := 0; i < 120; i++ {
		_, err := shares.Get(filesystemHandler.SharedFileSystemClient, shareID).Extract()
		if err != nil {
			break
		}

		time.Sleep(5 * time.Second)
	}

	accessSubnetList := filesystemInfo.AccessSubnetList
	if len(accessSubnetList) == 0 {
		cblogger.Warnf("subnet list is empty")
	} else {
		shareNetworkID, err := filesystemHandler.getOrCreateShareNetwork(filesystemInfo.VpcIID.SystemId, accessSubnetList[0].SystemId)
		if err != nil {
			cblogger.Warnf("failed to get share network: %v", err)
		} else {
			err = sharenetworks.Delete(filesystemHandler.SharedFileSystemClient, shareNetworkID).ExtractErr()
			if err != nil {
				cblogger.Warnf("failed to delete share network: %v", err)
			}
		}
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (filesystemHandler *OpenstackFileSystemHandler) AddAccessSubnet(fsIID irs.IID, subnetIID irs.IID) (irs.FileSystemInfo, error) {
	return irs.FileSystemInfo{}, fmt.Errorf("AddAccessSubnet is not supported in OpenStack - use CreateFileSystem with subnet specification")
}

func (filesystemHandler *OpenstackFileSystemHandler) RemoveAccessSubnet(fsIID irs.IID, subnetIID irs.IID) (bool, error) {
	return false, fmt.Errorf("RemoveAccessSubnet is not supported in OpenStack - recreate filesystem with different subnet")
}

func (filesystemHandler *OpenstackFileSystemHandler) ListAccessSubnet(fsIID irs.IID) ([]irs.IID, error) {
	if fsIID.NameId == "" && fsIID.SystemId == "" {
		return nil, fmt.Errorf("invalid filesystem IID: NameId or SystemId is required")
	}

	shareID := fsIID.SystemId
	if shareID == "" {
		listOpts := shares.ListOpts{
			Name: fsIID.NameId,
		}
		allPages, err := shares.ListDetail(filesystemHandler.SharedFileSystemClient, listOpts).AllPages()
		if err != nil {
			return nil, fmt.Errorf("failed to list shares: %v", err)
		}

		shareList, err := shares.ExtractShares(allPages)
		if err != nil {
			return nil, fmt.Errorf("failed to extract shares: %v", err)
		}

		if len(shareList) == 0 {
			return nil, fmt.Errorf("filesystem not found")
		}

		shareID = shareList[0].ID
	}

	share, err := shares.Get(filesystemHandler.SharedFileSystemClient, shareID).Extract()
	if err != nil {
		return nil, fmt.Errorf("failed to get share: %v", err)
	}

	var subnetList []irs.IID
	if share.ShareNetworkID != "" {
		shareNetwork, err := sharenetworks.Get(filesystemHandler.SharedFileSystemClient, share.ShareNetworkID).Extract()
		if err == nil && shareNetwork.NeutronSubnetID != "" {
			subnet, err := filesystemHandler.findSubnetByID(shareNetwork.NeutronSubnetID)
			if err == nil {
				subnetIID := irs.IID{
					NameId:   subnet.Name,
					SystemId: subnet.ID,
				}
				subnetList = append(subnetList, subnetIID)
			}
		}
	}

	return subnetList, nil
}

func (filesystemHandler *OpenstackFileSystemHandler) ScheduleBackup(reqInfo irs.FileSystemBackupInfo) (irs.FileSystemBackupInfo, error) {
	return irs.FileSystemBackupInfo{}, fmt.Errorf("backup scheduling is not supported in OpenStack Manila")
}

func (filesystemHandler *OpenstackFileSystemHandler) OnDemandBackup(fsIID irs.IID) (irs.FileSystemBackupInfo, error) {
	return irs.FileSystemBackupInfo{}, fmt.Errorf("on-demand backup is not supported in OpenStack Manila")
}

func (filesystemHandler *OpenstackFileSystemHandler) ListBackup(fsIID irs.IID) ([]irs.FileSystemBackupInfo, error) {
	return nil, fmt.Errorf("backup listing is not supported in OpenStack Manila")
}

func (filesystemHandler *OpenstackFileSystemHandler) GetBackup(fsIID irs.IID, backupID string) (irs.FileSystemBackupInfo, error) {
	return irs.FileSystemBackupInfo{}, fmt.Errorf("backup retrieval is not supported in OpenStack Manila")
}

func (filesystemHandler *OpenstackFileSystemHandler) DeleteBackup(fsIID irs.IID, backupID string) (bool, error) {
	return false, fmt.Errorf("backup deletion is not supported in OpenStack Manila")
}
