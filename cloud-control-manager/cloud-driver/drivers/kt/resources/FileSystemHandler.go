package resources

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	ktvpcsdk "github.com/cloud-barista/ktcloudvpc-sdk-go"
	networks "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/networks"
	subnets "github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/networking/v2/subnets"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/sharedfilesystems/v2/sharenetworks"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/openstack/sharedfilesystems/v2/shares"
	"github.com/cloud-barista/ktcloudvpc-sdk-go/pagination"
	// "github.com/davecgh/go-spew/spew"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// Inferface definition : https://github.com/cloud-barista/cb-spider/blob/master/cloud-control-manager/cloud-driver/interfaces/resources/FileSystemHandler.go
// KT Cloud NAS API manual : https://cloud.kt.com/docs/open-api-guide/d/storage/nas

type KTVpcFileSystemHandler struct {
	RegionInfo    idrv.RegionInfo
	NetworkClient *ktvpcsdk.ServiceClient
	NASClient     *ktvpcsdk.ServiceClient
}

const (
	DefaultZoneName 	 = "DX-M1" 	// Currently, KT Cloud shared Filesystem is available only on the 'DX-M1' by default.
	DefaultShareProtocol = "NFS" 	// Currently, cb-spider supports only the NFS protocol for shared file system volumes.
	DefaultNFSVersion    = "3.0"
	DefaultVolumeSizeGB  = 500
	DefaultShareType     = "HDD" 	// KT Cloud currently supports only “HDD” as the NAS volume type.
)

type QuotaUsage struct {
	TotalQuota   int64
	UsedCapacity int64
}

func (filesystemHandler *KTVpcFileSystemHandler) GetMetaInfo() (irs.FileSystemMetaInfo, error) {
	cblogger.Info("KT Cloud Driver: called GetMetaInfo()")

	metaInfo := irs.FileSystemMetaInfo{
		SupportsFileSystemType: map[irs.FileSystemType]bool{
			irs.RegionType:          false,
			irs.ZoneType:            true,
			irs.RegionVPCBasedType:  false,
			irs.RegionZoneBasedType: true,
		},
		SupportsVPC: map[irs.RSType]bool{
			irs.VPC: false,
		},
		SupportsNFSVersion: []string{DefaultNFSVersion},
		SupportsCapacity:   true,
		CapacityGBOptions: map[string]irs.CapacityGBRange{
			"default": {
				Min: 500,
				Max: 10000,
			},
		},
	}

	return metaInfo, nil
}

func (filesystemHandler *KTVpcFileSystemHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("KT Cloud Driver: called ListIID()")

	listOpts := shares.ListOpts{}
	allPages, err := shares.ListDetail(filesystemHandler.NASClient, listOpts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("failed to list NAS shares: %w", err)
	}

	shareList, err := shares.ExtractShares(allPages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract NAS shares: %w", err)
	}

	iidList := make([]*irs.IID, 0, len(shareList))
	for _, share := range shareList {
		if filesystemHandler.RegionInfo.Zone != "" && share.AvailabilityZone != "" &&
			!strings.EqualFold(filesystemHandler.RegionInfo.Zone, share.AvailabilityZone) {
			continue
		}

		iid := &irs.IID{
			NameId:   share.Name,
			SystemId: share.ID,
		}
		iidList = append(iidList, iid)
	}

	return iidList, nil
}

func (fileSystemHandler *KTVpcFileSystemHandler) CreateFileSystem(reqInfo irs.FileSystemInfo) (irs.FileSystemInfo, error) {
	cblogger.Info("KT Cloud Driver: called CreateFileSystem()")

	if reqInfo.IId.NameId == "" {
		err := fmt.Errorf("Invalid request: NameId is required")
		return irs.FileSystemInfo{}, err
	}

	if reqInfo.Zone == "" {
		err := fmt.Errorf("Invalid request: Zone Name is required")
		return irs.FileSystemInfo{}, err
	}

	if reqInfo.Zone != DefaultZoneName {
		err := fmt.Errorf("Invalid request: Currently, KT Cloud shared Filesystem is available only on the 'DX-M1' zone by default.")
		return irs.FileSystemInfo{}, err
	}
	
	if !regexp.MustCompile(`^[A-Za-z0-9_]+$`).MatchString(reqInfo.IId.NameId) {
		err := fmt.Errorf("invalid request: NameId cannot contain special characters other than underscore (_)")
		return irs.FileSystemInfo{}, err
	}

	// # Note) FileSystemType : If not specified, it defaults to 'ZONE-TYPE'.
	if reqInfo.FileSystemType == irs.RegionType {
		err := fmt.Errorf("Unsupported FileSystemType on KT Cloud : %s (only 'ZONE-TYPE' is supported)", reqInfo.FileSystemType)
		return irs.FileSystemInfo{}, err
	}

	if reqInfo.CapacityGB == 0 {
		reqInfo.CapacityGB = DefaultVolumeSizeGB
	}

	// Note) KT NAS volume size must be between 500GB and 10,000GB
	if reqInfo.CapacityGB < 500 || reqInfo.CapacityGB > 10000 {
		err := fmt.Errorf("Invalid volume size: %d GB. KT NAS volume size must be between 500GB and 10,000GB", reqInfo.CapacityGB)
		return irs.FileSystemInfo{}, err
	}

	vpcSystemID := strings.TrimSpace(reqInfo.VpcIID.SystemId)
	if vpcSystemID == "" {
		err := fmt.Errorf("invalid request: VpcIID.SystemId is required")
		return irs.FileSystemInfo{}, err
	}

	vpcExists, err := fileSystemHandler.existsVPCBySystemID(vpcSystemID)
	if err != nil {
		return irs.FileSystemInfo{}, fmt.Errorf("failed to validate VPC SystemID %s: %w", vpcSystemID, err)
	}
	if !vpcExists {
		return irs.FileSystemInfo{}, fmt.Errorf("invalid request: VPC SystemID %s not found", vpcSystemID)
	}

	if len(reqInfo.AccessSubnetList) == 0 {
		err := fmt.Errorf("invalid request: AccessSubnetList is required")
		return irs.FileSystemInfo{}, err
	}

	subnetNameID := strings.TrimSpace(reqInfo.AccessSubnetList[0].NameId)
	if subnetNameID == "" {
		err := fmt.Errorf("invalid request: AccessSubnetList[0].NameId (Subnet NameID) is required")
		return irs.FileSystemInfo{}, err
	}

	shareNetworkID, err := fileSystemHandler.createSharenetwork(subnetNameID)
	if err != nil {
		return irs.FileSystemInfo{}, fmt.Errorf("failed to create share network with Subnet NameID %s: %w", subnetNameID, err)
	}

	// Note) KT Cloud supports: 'NFS' (default) for Linux, 'CIFS' for MS Windows
	// Currently, cb-spider supports only the NFS protocol for shared file 
	if reqInfo.NFSVersion != "" {
		if reqInfo.NFSVersion == "4.1" {
			err := fmt.Errorf("KT Cloud FileSystem supports NFS version '3.0'")
			return irs.FileSystemInfo{}, err
		}		
	}

	cblogger.Info("### Creating NAS volume instance...")
	createOpts := shares.CreateOpts{
		ShareProto:     	DefaultShareProtocol,
		ShareNetworkID: 	shareNetworkID,
		Name:           	reqInfo.IId.NameId,
		Size:           	int(reqInfo.CapacityGB),
		AvailabilityZone:  	reqInfo.Zone,
		ShareType:      	DefaultShareType, // KT Cloud currently supports only “HDD” as the NAS volume type.
	}

	createdShare, err := shares.Create(fileSystemHandler.NASClient, createOpts).Extract()
	if err != nil {
		return irs.FileSystemInfo{}, fmt.Errorf("failed to create NAS share: %w", err)
	}

	if createdShare == nil || createdShare.ID == "" {
		err := fmt.Errorf("failed to create NAS share: empty share result")
		return irs.FileSystemInfo{}, err
	}

	// Wait for the volume to become 'available'
	cblogger.Info("### Waiting for NAS volume to be ready...")
	availableShare, err := fileSystemHandler.waitForShareAvailable(createdShare.ID, 10*time.Minute)
	if err != nil {
		return irs.FileSystemInfo{}, fmt.Errorf("NAS share %s did not become available: %w", createdShare.ID, err)
	}

	fileSystemInfo, err := fileSystemHandler.mapShareToFileSystemInfo(availableShare)
	if err != nil {
		return irs.FileSystemInfo{}, fmt.Errorf("failed to map created NAS share %s: %w", createdShare.ID, err)
	}
	return *fileSystemInfo, nil

}

func (filesystemHandler *KTVpcFileSystemHandler) ListFileSystem() ([]*irs.FileSystemInfo, error) {
	cblogger.Info("KT Cloud Driver: called ListFileSystem()")

	listOpts := shares.ListOpts{}
	allPages, err := shares.ListDetail(filesystemHandler.NASClient, listOpts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("failed to list NAS shares: %w", err)
	}

	shareList, err := shares.ExtractShares(allPages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract NAS shares: %w", err)
	}

	fileSystemList := make([]*irs.FileSystemInfo, 0, len(shareList))
	for _, share := range shareList {
		if filesystemHandler.RegionInfo.Zone != "" && share.AvailabilityZone != "" &&
			!strings.EqualFold(filesystemHandler.RegionInfo.Zone, share.AvailabilityZone) {
			continue
		}

		fileSystemInfo, err := filesystemHandler.mapShareToFileSystemInfo(&share)
		if err != nil {
			return nil, fmt.Errorf("failed to map NAS share %s: %w", share.ID, err)
		}

		fileSystemList = append(fileSystemList, fileSystemInfo)
	}

	return fileSystemList, nil
}

func (filesystemHandler *KTVpcFileSystemHandler) mapShareToFileSystemInfo(share *shares.Share) (*irs.FileSystemInfo, error) {
	cblogger.Info("KT Cloud Driver: called mapShareToFileSystemInfo()")

	if share == nil {
		return nil, fmt.Errorf("File system does not exist.!!")
	}

	accessSubnetList := []irs.IID{}
	vpcIID := irs.IID{
		NameId:   "NA",
		SystemId: "NA",
	}
	if strings.TrimSpace(share.ShareNetworkID) != "" {
		resolvedAccessSubnetList, err := filesystemHandler.getAccessSubnetListByShareNetworkID(share.ShareNetworkID)
		if err != nil {
			cblogger.Warnf("failed to resolve AccessSubnetList from share network %s: %v", share.ShareNetworkID, err)
		} else {
			accessSubnetList = resolvedAccessSubnetList
		}

		resolvedVpcIID, err := filesystemHandler.getVpcIIDByShareNetworkID(share.ShareNetworkID)
		if err != nil {
			cblogger.Warnf("failed to resolve VPC from share network %s: %v", share.ShareNetworkID, err)
		} else if resolvedVpcIID != nil {
			vpcIID = *resolvedVpcIID
		}
	}

	fileSystemInfo := &irs.FileSystemInfo{
		IId: irs.IID{
			NameId:   share.Name,
			SystemId: share.ID,
		},
		Region:           filesystemHandler.RegionInfo.Region,
		Zone:             share.AvailabilityZone,
		VpcIID:           vpcIID,
		AccessSubnetList: accessSubnetList,
		FileSystemType:   irs.ZoneType,
		CapacityGB:       int64(share.Size),
		Status:           mapKTNasStatusToFileSystemStatus(share.Status),
		CreatedTime:      share.CreatedAt,
		KeyValueList:     irs.StructToKeyValueList(share),
	}

	if fileSystemInfo.Zone == "" {
		fileSystemInfo.Zone = filesystemHandler.RegionInfo.Zone
	}
	if fileSystemInfo.Zone == "" {
		fileSystemInfo.FileSystemType = irs.RegionType
	}

	if strings.EqualFold(share.ShareProto, "NFS") {
		fileSystemInfo.NFSVersion = DefaultNFSVersion
	}

	for _, exportLocation := range share.ExportLocations {
		if exportLocation == "" {
			continue
		}
		fileSystemInfo.MountTargetList = append(fileSystemInfo.MountTargetList, irs.MountTargetInfo{
			Endpoint: exportLocation,
		})
	}
	if len(fileSystemInfo.MountTargetList) == 0 && share.ExportLocation != "" {
		fileSystemInfo.MountTargetList = append(fileSystemInfo.MountTargetList, irs.MountTargetInfo{
			Endpoint: share.ExportLocation,
		})
	}

	return fileSystemInfo, nil
}

func (filesystemHandler *KTVpcFileSystemHandler) getAccessSubnetListByShareNetworkID(shareNetworkID string) ([]irs.IID, error) {
	shareNetworkID = strings.TrimSpace(shareNetworkID)
	if shareNetworkID == "" {
		return nil, fmt.Errorf("invalid share network ID: empty value")
	}

	shareNetwork, err := sharenetworks.Get(filesystemHandler.NASClient, shareNetworkID).Extract()
	if err != nil {
		return nil, fmt.Errorf("failed to get share network %s: %w", shareNetworkID, err)
	}
	if shareNetwork == nil {
		return nil, fmt.Errorf("share network %s not found", shareNetworkID)
	}

	subnetIID := irs.IID{
		NameId:   strings.TrimSpace(shareNetwork.Name),
		SystemId: strings.TrimSpace(shareNetwork.NeutronNetID),
	}

	if subnetIID.SystemId == "" {
		subnetIID.SystemId = strings.TrimSpace(shareNetwork.ID)
	}
	if subnetIID.NameId == "" {
		subnetIID.NameId = subnetIID.SystemId
	}

	return []irs.IID{subnetIID}, nil
}

func (filesystemHandler *KTVpcFileSystemHandler) getVpcIIDByShareNetworkID(shareNetworkID string) (*irs.IID, error) {
	shareNetworkID = strings.TrimSpace(shareNetworkID)
	if shareNetworkID == "" {
		return nil, fmt.Errorf("invalid share network ID: empty value")
	}

	shareNetwork, err := sharenetworks.Get(filesystemHandler.NASClient, shareNetworkID).Extract()
	if err != nil {
		return nil, fmt.Errorf("failed to get share network %s: %w", shareNetworkID, err)
	}
	if shareNetwork == nil {
		return nil, fmt.Errorf("share network %s not found", shareNetworkID)
	}

	tierID := strings.TrimSpace(shareNetwork.NeutronNetID)
	if tierID == "" {
		return nil, fmt.Errorf("share network %s has empty Tier ID(neutron_net_id)", shareNetworkID)
	}

	subnetList, err := filesystemHandler.listKTSubnet()
	if err != nil {
		return nil, fmt.Errorf("failed to list KT subnets: %w", err)
	}

	vpcID := ""
	for _, subnet := range subnetList {
		if subnet == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(subnet.RefID), tierID) {
			vpcID = strings.TrimSpace(subnet.VpcID)
			break
		}
	}
	if vpcID == "" {
		return nil, fmt.Errorf("failed to resolve VPC ID from Tier ID %s", tierID)
	}

	vpc, err := networks.Get(filesystemHandler.NetworkClient, vpcID).ExtractVPC()
	if err != nil {
		return nil, fmt.Errorf("failed to get VPC %s: %w", vpcID, err)
	}
	if vpc == nil {
		return nil, fmt.Errorf("VPC %s not found", vpcID)
	}

	return &irs.IID{
		NameId:   strings.TrimSpace(vpc.Name),
		SystemId: strings.TrimSpace(vpc.VpcID),
	}, nil
}

func mapKTNasStatusToFileSystemStatus(status string) irs.FileSystemStatus {
	switch strings.ToLower(status) {
	case "creating", "creating_from_snapshot", "manage_starting", "unmanage_starting", "extending", "shrinking", "migrating", "replication_change", "reverting":
		return irs.FileSystemCreating
	case "deleting", "deleted":
		return irs.FileSystemDeleting
	case "error", "error_deleting", "manage_error", "unmanage_error", "extending_error", "shrinking_error", "shrinking_possible_data_loss_error", "reverting_error":
		return irs.FileSystemError
	case "available", "inactive", "migrating_to", "unmanaged":
		return irs.FileSystemAvailable
	default:
		return irs.FileSystemCreating
	}
}

func (filesystemHandler *KTVpcFileSystemHandler) GetFileSystem(iid irs.IID) (irs.FileSystemInfo, error) {
	cblogger.Info("KT Cloud Driver: called GetFileSystem()")

	shareID := iid.SystemId
	if shareID == "" && iid.NameId != "" {
		listOpts := shares.ListOpts{}
		allPages, err := shares.ListDetail(filesystemHandler.NASClient, listOpts).AllPages()
		if err != nil {
			return irs.FileSystemInfo{}, fmt.Errorf("failed to list NAS file system for NameId lookup: %w", err)
		}

		shareList, err := shares.ExtractShares(allPages)
		if err != nil {
			return irs.FileSystemInfo{}, fmt.Errorf("failed to extract NAS file system for NameId lookup: %w", err)
		}

		for _, share := range shareList {
			if strings.EqualFold(share.Name, iid.NameId) {
				shareID = share.ID
				break
			}
		}
	}

	if shareID == "" {
		newErr := fmt.Errorf("invalid IID - SystemId or resolvable NameId is required")
		cblogger.Error(newErr)
		return irs.FileSystemInfo{}, newErr
	}

	share, err := shares.Get(filesystemHandler.NASClient, shareID).Extract()
	if err != nil {
		return irs.FileSystemInfo{}, fmt.Errorf("failed to get NAS file system %s: %w", shareID, err)
	}

	fileSystemInfo, err := filesystemHandler.mapShareToFileSystemInfo(share)
	if err != nil {
		return irs.FileSystemInfo{}, fmt.Errorf("failed to map NAS file system %s: %w", shareID, err)
	}
	return *fileSystemInfo, nil
}

func (filesystemHandler *KTVpcFileSystemHandler) DeleteFileSystem(iid irs.IID) (bool, error) {
	cblogger.Info("KT Cloud Driver: called DeleteFileSystem()")

	shareID := strings.TrimSpace(iid.SystemId)
	if shareID == "" && strings.TrimSpace(iid.NameId) != "" {
		listOpts := shares.ListOpts{}
		allPages, err := shares.ListDetail(filesystemHandler.NASClient, listOpts).AllPages()
		if err != nil {
			return false, fmt.Errorf("failed to list NAS shares for NameId lookup: %w", err)
		}

		shareList, err := shares.ExtractShares(allPages)
		if err != nil {
			return false, fmt.Errorf("failed to extract NAS shares for NameId lookup: %w", err)
		}

		for _, share := range shareList {
			if strings.EqualFold(share.Name, iid.NameId) {
				shareID = share.ID
				break
			}
		}
	}

	if shareID == "" {
		return false, fmt.Errorf("invalid IID - SystemId or resolvable NameId is required")
	}

	share, err := shares.Get(filesystemHandler.NASClient, shareID).Extract()
	if err != nil {
		return false, fmt.Errorf("failed to get NAS share %s: %w", shareID, err)
	}

	cblogger.Info("### Waiting for the NAS volume to be deleted...")
	shareNetworkID := ""
	if share != nil {
		shareNetworkID = strings.TrimSpace(share.ShareNetworkID)
	}

	err = shares.Delete(filesystemHandler.NASClient, shareID).ExtractErr()
	if err != nil {
		return false, fmt.Errorf("failed to delete NAS share %s: %w", shareID, err)
	}

	cblogger.Info("### Waiting for the NAS volume to be completely deleted...")
	err = filesystemHandler.waitForShareDeleted(shareID, 10*time.Minute)
	if err != nil {
		return false, fmt.Errorf("NAS share %s did not complete deletion: %w", shareID, err)
	}

	cblogger.Info("### Waiting for the shared network to be deleted...")
	if shareNetworkID != "" {
		err = filesystemHandler.deleteSharenetwork(shareNetworkID)
		if err != nil {
			return false, fmt.Errorf("failed to delete shared network %s for NAS share %s: %w", shareNetworkID, shareID, err)
		}
	}

	return true, nil
}

// #############

func (filesystemHandler *KTVpcFileSystemHandler) AddAccessSubnet(fsIID irs.IID, subnetIID irs.IID) (irs.FileSystemInfo, error) {
	cblogger.Info("KT Cloud Driver: called AddAccessSubnet()")

	return irs.FileSystemInfo{}, fmt.Errorf("AddAccessSubnet is not supported in OpenStack - use CreateFileSystem with subnet specification")
}

func (filesystemHandler *KTVpcFileSystemHandler) RemoveAccessSubnet(fsIID irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger.Info("KT Cloud Driver: called RemoveAccessSubnet()")

	return false, fmt.Errorf("RemoveAccessSubnet is not supported in OpenStack - recreate filesystem with different subnet")
}

func (filesystemHandler *KTVpcFileSystemHandler) ListAccessSubnet(fsIID irs.IID) ([]irs.IID, error) {
	cblogger.Info("KT Cloud Driver: called ListAccessSubnet()")

	return nil, nil
}

func (filesystemHandler *KTVpcFileSystemHandler) ScheduleBackup(reqInfo irs.FileSystemBackupInfo) (irs.FileSystemBackupInfo, error) {
	cblogger.Info("KT Cloud Driver: called ScheduleBackup()")

	return irs.FileSystemBackupInfo{}, fmt.Errorf("backup scheduling is not supported in OpenStack Manila")
}

func (filesystemHandler *KTVpcFileSystemHandler) OnDemandBackup(fsIID irs.IID) (irs.FileSystemBackupInfo, error) {
	cblogger.Info("KT Cloud Driver: called OnDemandBackup()")

	return irs.FileSystemBackupInfo{}, fmt.Errorf("on-demand backup is not supported in OpenStack Manila")
}

func (filesystemHandler *KTVpcFileSystemHandler) ListBackup(fsIID irs.IID) ([]irs.FileSystemBackupInfo, error) {
	cblogger.Info("KT Cloud Driver: called ListBackup()")

	return nil, fmt.Errorf("backup listing is not supported in OpenStack Manila")
}

func (filesystemHandler *KTVpcFileSystemHandler) GetBackup(fsIID irs.IID, backupID string) (irs.FileSystemBackupInfo, error) {
	cblogger.Info("KT Cloud Driver: called GetBackup()")

	return irs.FileSystemBackupInfo{}, fmt.Errorf("backup retrieval is not supported in OpenStack Manila")
}

func (filesystemHandler *KTVpcFileSystemHandler) DeleteBackup(fsIID irs.IID, backupID string) (bool, error) {
	cblogger.Info("KT Cloud Driver: called DeleteBackup()")

	return false, fmt.Errorf("backup deletion is not supported in OpenStack Manila")
}

func (filesystemHandler *KTVpcFileSystemHandler) waitForShareAvailable(shareID string, timeout time.Duration) (*shares.Share, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		share, err := shares.Get(filesystemHandler.NASClient, shareID).Extract()
		if err != nil {
			return nil, fmt.Errorf("failed to get NAS volume status for %s: %w", shareID, err)
		}
		cblogger.Infof("### NAS volume %s status: %s", shareID, share.Status)
		switch strings.ToLower(share.Status) {
		case "available":
			return share, nil
		case "error", "error_deleting", "manage_error", "unmanage_error":
			return nil, fmt.Errorf("NAS share %s entered error state: %s", shareID, share.Status)
		}
		time.Sleep(3 * time.Second)
	}
	return nil, fmt.Errorf("timed out waiting for NAS volume %s to become available", shareID)
}

func (filesystemHandler *KTVpcFileSystemHandler) waitForShareDeleted(shareID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		share, err := shares.Get(filesystemHandler.NASClient, shareID).Extract()
		if err != nil {
			// If we get an error (e.g., 404), the share is deleted
			cblogger.Infof("### NAS share %s has been completely deleted (status check returned error as expected)", shareID)
			return nil
		}
		if share == nil {
			cblogger.Infof("### NAS share %s has been completely deleted", shareID)
			return nil
		}
		cblogger.Infof("### NAS share %s deletion status: %s", shareID, share.Status)
		if strings.EqualFold(strings.TrimSpace(share.Status), "deleted") {
			cblogger.Infof("### NAS share %s has been marked as deleted", shareID)
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("timed out waiting for NAS share %s to be completely deleted", shareID)
}

func (filesystemHandler *KTVpcFileSystemHandler) createSharenetwork(subnetNameID string) (string, error) {
	cblogger.Info("KT Cloud Driver: called createSharenetwork()")
	
	if subnetNameID == "" {
		return "", fmt.Errorf("invalid Subnet NameID: empty value")
	}

	subnetNameID = strings.TrimSpace(subnetNameID)
	subnetNameID = strings.ToLower(subnetNameID)
	
	cblogger.Infof("### Requested subnet NameID : %s", subnetNameID)

	subnetList, err := filesystemHandler.listKTSubnet()
	if err != nil {
		return "", fmt.Errorf("failed to list KT subnets for Subnet NameID validation: %w", err)
	}

	tierID := ""
	for _, subnet := range subnetList {
		if subnet == nil {
			continue
		}
		if strings.EqualFold(subnet.RefName, subnetNameID) {
			tierID = strings.TrimSpace(subnet.RefID)
			break
		}
	}

	if tierID == "" {
		return "", fmt.Errorf("invalid Subnet NameID: %s (not found in listKTSubnet using subnet.RefName)", subnetNameID)
	}
	
	cblogger.Infof("### Subnet(Tier) ID : %s", tierID)

	cblogger.Info("### Creating Shared Network...")
	createOpts := sharenetworks.CreateOpts{
		NeutronNetID: tierID,
		Name:         subnetNameID, // Like web console.
	}

	createdShareNetwork, err := sharenetworks.Create(filesystemHandler.NASClient, createOpts).Extract()
	if err != nil {
		return "", fmt.Errorf("failed to create share network for Subnet NameID %s (Tier ID %s): %w", subnetNameID, tierID, err)
	}

	if createdShareNetwork == nil || createdShareNetwork.ID == "" {
		return "", fmt.Errorf("failed to create share network for Subnet NameID %s (Tier ID %s): empty share network ID", subnetNameID, tierID)
	}
	cblogger.Infof("### createdShareNetwork.ID : %s", createdShareNetwork.ID)

	return createdShareNetwork.ID, nil
}

func (filesystemHandler *KTVpcFileSystemHandler) deleteSharenetwork(sharedNetworkID string) error {
	cblogger.Info("KT Cloud Driver: called deleteSharenetwork()")

	sharedNetworkID = strings.TrimSpace(sharedNetworkID)
	if sharedNetworkID == "" {
		return fmt.Errorf("invalid shared network ID: empty value")
	}

	// shareNetwork, err := sharenetworks.Get(filesystemHandler.NASClient, sharedNetworkID).Extract()
	// if err != nil {
	// 	return fmt.Errorf("failed to get shared network %s: %w", sharedNetworkID, err)
	// }
	// if shareNetwork != nil && strings.TrimSpace(shareNetwork.NeutronNetID) != "" {
	// 	cblogger.Infof("### Skipping deletion of Shared Network %s: The Shared Network (%s) is still in use", sharedNetworkID, shareNetwork.NeutronNetID)
	// 	return nil
	// }

	cblogger.Infof("### Deleting Shared Network... ID: %s", sharedNetworkID)
	err := sharenetworks.Delete(filesystemHandler.NASClient, sharedNetworkID).ExtractErr()
	if err != nil {
		return fmt.Errorf("failed to delete shared network %s: %w", sharedNetworkID, err)
	}

	return nil
}

func (filesystemHandler *KTVpcFileSystemHandler) listKTSubnet() ([]*subnets.Subnet, error) {
	if filesystemHandler.NetworkClient == nil {
		return nil, fmt.Errorf("network client is nil")
	}

	listOpts := subnets.ListOpts{
		Page:        1,
		Size:        20,
		NetworkType: "ALL",
	}

	pager := subnets.List(filesystemHandler.NetworkClient, listOpts)

	var subnetAdrsList []*subnets.Subnet
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		subnetList, err := subnets.ExtractSubnets(page)
		if err != nil {
			return false, fmt.Errorf("failed to extract subnet list: %w", err)
		}
		for _, subnet := range subnetList {
			subnetAdrsList = append(subnetAdrsList, &subnet)
		}
		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list KT subnets: %w", err)
	}

	return subnetAdrsList, nil
}

func (filesystemHandler *KTVpcFileSystemHandler) listKTVPC() ([]*networks.VPC, error) {
	if filesystemHandler.NetworkClient == nil {
		return nil, fmt.Errorf("network client is nil")
	}

	listOpts := networks.ListOpts{
		Page: 1,
		Size: 20,
	}

	pager := networks.List(filesystemHandler.NetworkClient, listOpts)

	var vpcAdrsList []*networks.VPC
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		vpcList, err := networks.ExtractVPCs(page)
		if err != nil {
			return false, fmt.Errorf("failed to extract VPC list: %w", err)
		}
		for _, vpc := range vpcList {
			vpcAdrsList = append(vpcAdrsList, &vpc)
		}
		return true, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list VPCs: %w", err)
	}

	return vpcAdrsList, nil
}

func (filesystemHandler *KTVpcFileSystemHandler) existsVPCBySystemID(vpcSystemID string) (bool, error) {
	vpcSystemID = strings.TrimSpace(vpcSystemID)
	if vpcSystemID == "" {
		return false, fmt.Errorf("invalid VPC SystemID: empty value")
	}

	vpcList, err := filesystemHandler.listKTVPC()
	if err != nil {
		return false, err
	}

	for _, vpc := range vpcList {
		if vpc == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(vpc.VpcID), vpcSystemID) {
			return true, nil
		}
	}

	return false, nil
}
