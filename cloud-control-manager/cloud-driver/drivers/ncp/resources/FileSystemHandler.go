package resources

import (
	"fmt"
	"strings"
	"time"

	"github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	vnas "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vnas"
	// "github.com/davecgh/go-spew/spew"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

// Inferface definition : https://github.com/cloud-barista/cb-spider/blob/master/cloud-control-manager/cloud-driver/interfaces/resources/FileSystemHandler.go

type NcpVpcFileSystemHandler struct {
	CredentialInfo         	idrv.CredentialInfo
	RegionInfo              idrv.RegionInfo
	VNASClient       		*vnas.APIClient
}

const (
	DefaultShareType       = "cb-spider-share-type"
	DefaultShareProtocol   = "NFS" // Currently, cb-spider supports only the NFS protocol for shared file system volumes.
	DefaultNFSVersion      = "3.0"
	DefaultVolumeSizeGB    = 500
	ShareNetworkNamePrefix = "cb-spider-share-network-"
)

type QuotaUsage struct {
	TotalQuota   int64
	UsedCapacity int64
}

func (filesystemHandler *NcpVpcFileSystemHandler) GetMetaInfo() (irs.FileSystemMetaInfo, error) {
	cblogger.Info("NCP VPC Driver: called GetMetaInfo()")

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
				Max: 20000,
			},
		},
	}

	return metaInfo, nil
}

func (filesystemHandler *NcpVpcFileSystemHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("NCP VPC Driver: called ListIID()")

	req := vnas.GetNasVolumeInstanceListRequest{
		RegionCode: ncloud.String(filesystemHandler.RegionInfo.Region),
		ZoneCode:   ncloud.String(filesystemHandler.RegionInfo.Zone),
	}

	result, err := filesystemHandler.VNASClient.V2Api.GetNasVolumeInstanceList(&req)
	if err != nil {
		cblogger.Errorf("Failed to get NAS volume instance list : %v", err)
		return nil, err
	}

	var iidList []*irs.IID
	for _, nasVolume := range result.NasVolumeInstanceList {
		if nasVolume == nil {
			continue
		}

		var nameId, systemId string

		if nasVolume.VolumeName != nil {
			nameId = *nasVolume.VolumeName
		}

		if nasVolume.NasVolumeInstanceNo != nil {
			systemId = *nasVolume.NasVolumeInstanceNo
		}

		iid := irs.IID{
			NameId:   nameId,
			SystemId: systemId,
		}
		iidList = append(iidList, &iid)
	}

	cblogger.Infof("Successfully retrieved %d NAS volume IIDs", len(iidList))
	return iidList, nil
}

func (fileSystemHandler *NcpVpcFileSystemHandler) CreateFileSystem(reqInfo irs.FileSystemInfo) (irs.FileSystemInfo, error) {
	cblogger.Info("NCP VPC Driver: called CreateFileSystem()")

	if reqInfo.IId.NameId == "" {
		err := fmt.Errorf("Invalid request: NameId is required")
		return irs.FileSystemInfo{}, err
	}

	// # Note) NAS volume name :
	// 3 to 20 characters; only English letter and numbers can be entered
	// (Entered name is prefixed with "nMemberID_" in NCP for customer identification)
	for _, r := range reqInfo.IId.NameId {
		isUpper := r >= 'A' && r <= 'Z'
		isLower := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if !isUpper && !isLower && !isDigit {
			err := fmt.Errorf("NameId cannot contain special characters")
			return irs.FileSystemInfo{}, err
		}
	}

	// # Note) FileSystemType : If not specified, it defaults to 'ZONE-TYPE' in mapNasVolumeToFileSystemInfo().
	if reqInfo.FileSystemType == irs.RegionType {
		err := fmt.Errorf("Unsupported FileSystemType on NCP: %s (only 'ZONE-TYPE' is supported)", reqInfo.FileSystemType)
		return irs.FileSystemInfo{}, err
	}

	if reqInfo.CapacityGB == 0 {
		reqInfo.CapacityGB = DefaultVolumeSizeGB
	}

	// Note) NCP NAS volume size must be between 500GB and 20,000GB
	if reqInfo.CapacityGB < 500 || reqInfo.CapacityGB > 20000 {
		err := fmt.Errorf("Invalid volume size: %d GB. NCP NAS volume size must be between 500GB and 20,000GB", reqInfo.CapacityGB)
		return irs.FileSystemInfo{}, err
	}	
	
	req := vnas.CreateNasVolumeInstanceRequest{
		RegionCode: ncloud.String(fileSystemHandler.RegionInfo.Region),
		ZoneCode:   ncloud.String(fileSystemHandler.RegionInfo.Zone),
		VolumeName: ncloud.String(reqInfo.IId.NameId),
	}

	// Convert GB to bytes - NCP expects size in GB as int32
	req.VolumeSize = ncloud.Int32(int32(reqInfo.CapacityGB))

	// Note) NCP supports: 'NFS' (default) for Linux, 'CIFS' for MS Windows
	if reqInfo.NFSVersion != "" {
		if reqInfo.NFSVersion == "4.1" {
			err := fmt.Errorf("NCP FileSystem supports NFS version '3.0'")
			return irs.FileSystemInfo{}, err
		}
		if strings.HasPrefix(strings.ToUpper(reqInfo.NFSVersion), "NFS") || 
		   reqInfo.NFSVersion == "3.0" {
			req.VolumeAllotmentProtocolTypeCode = ncloud.String("NFS")
		} else {
			req.VolumeAllotmentProtocolTypeCode = ncloud.String("NFS") // Default to NFS
		}
	} else {
		req.VolumeAllotmentProtocolTypeCode = ncloud.String("NFS") // Default protocol
	}

	if reqInfo.Encryption {
		req.IsEncryptedVolume = ncloud.Bool(true)
	}

	description := "NAS Volume created by CB-Spider"
	if len(reqInfo.KeyValueList) > 0 {
		for _, kv := range reqInfo.KeyValueList {
			if kv.Key == "NasVolumeDescription" {
				description = kv.Value
				break
			}
		}
	}
	req.NasVolumeDescription = ncloud.String(description)

	// Set return protection (default: false)
	isReturnProtection := false
	if len(reqInfo.KeyValueList) > 0 {
		for _, kv := range reqInfo.KeyValueList {
			if kv.Key == "IsReturnProtection" && kv.Value == "true" {
				isReturnProtection = true
				break
			}
		}
	}
	req.IsReturnProtection = ncloud.Bool(isReturnProtection)

	cblogger.Info("### Creating NAS volume instance...")
	result, err := fileSystemHandler.VNASClient.V2Api.CreateNasVolumeInstance(&req)
	if err != nil {
		cblogger.Errorf("Failed to create NAS volume instance: %v", err)
		return irs.FileSystemInfo{}, err
	}

	if result == nil || len(result.NasVolumeInstanceList) == 0 {
		err := fmt.Errorf("Failed to create NAS volume: empty response")
		return irs.FileSystemInfo{}, err
	}

	nasVolume := result.NasVolumeInstanceList[0]
	if nasVolume == nil {
		err := fmt.Errorf("failed to create NAS volume: nil volume in response")
		return irs.FileSystemInfo{}, err
	}

	// Wait for the volume to be in 'creating' or 'available' state
	// NCP NAS creation is usually fast, but we'll add a simple wait
	cblogger.Info("### Waiting for NAS volume to be ready...")
	time.Sleep(5 * time.Second)

	// Get the created file system info
	systemId := ncloud.StringValue(nasVolume.NasVolumeInstanceNo)
	fileSystemInfo, err := fileSystemHandler.GetFileSystem(irs.IID{SystemId: systemId})
	if err != nil {
		cblogger.Warnf("Created NAS volume but failed to get details: %v", err)
		fileSystemInfo, err = fileSystemHandler.mapNasVolumeToFileSystemInfo(nasVolume)
		if err != nil {
			return irs.FileSystemInfo{}, fmt.Errorf("Failed to convert created NAS volume info: %v", err)
		}
	}
	return fileSystemInfo, nil
}

func (filesystemHandler *NcpVpcFileSystemHandler) ListFileSystem() ([]*irs.FileSystemInfo, error) {
	cblogger.Info("NCP VPC Driver: called ListFileSystem()")

	req := vnas.GetNasVolumeInstanceListRequest{
		RegionCode: ncloud.String(filesystemHandler.RegionInfo.Region),
		ZoneCode: 	ncloud.String(filesystemHandler.RegionInfo.Zone),
	}

	result, err := filesystemHandler.VNASClient.V2Api.GetNasVolumeInstanceList(&req)
	if err != nil {
		cblogger.Errorf("Failed to get NAS volume instance list : %v", err)
		return nil, err
	}

	if len(result.NasVolumeInstanceList) == 0 {
		cblogger.Info("No NAS volume instances found")
		return []*irs.FileSystemInfo{}, nil
	}

	var fileSystemList []*irs.FileSystemInfo
	for _, nasVolume := range result.NasVolumeInstanceList {
		if nasVolume == nil {
			continue
		}

		fileSystemInfo, err := filesystemHandler.mapNasVolumeToFileSystemInfo(nasVolume)
		if err != nil {
			cblogger.Errorf("Failed to convert NAS volume instance info [%s]: %v", 
				ncloud.StringValue(nasVolume.NasVolumeInstanceNo), err)
			continue
		}
		
		fileSystemList = append(fileSystemList, &fileSystemInfo)
	}

	cblogger.Infof("Successfully retrieved %d NAS volume instances", len(fileSystemList))
	return fileSystemList, nil
}

func (filesystemHandler *NcpVpcFileSystemHandler) mapNasVolumeToFileSystemInfo(nasVolume *vnas.NasVolumeInstance) (irs.FileSystemInfo, error) {
	cblogger.Info("NCP VPC Driver: called mapNasVolumeToFileSystemInfo()")

	if nasVolume == nil {
		return irs.FileSystemInfo{}, fmt.Errorf("nasVolume is nil")
	}

	fileSystemInfo := irs.FileSystemInfo{
		IId: irs.IID{
			NameId:   ncloud.StringValue(nasVolume.VolumeName),
			SystemId: ncloud.StringValue(nasVolume.NasVolumeInstanceNo),
		},
		Region: filesystemHandler.RegionInfo.Region,
		VpcIID: irs.IID{
			NameId:   "NA",
			SystemId: "NA",
		},
	}

	if nasVolume.ZoneCode != nil {
		fileSystemInfo.Zone = ncloud.StringValue(nasVolume.ZoneCode)
	}

	// Default : irs.ZoneType
	if nasVolume.ZoneCode != nil && *nasVolume.ZoneCode != "" {
		fileSystemInfo.FileSystemType = irs.ZoneType
	} else {
		fileSystemInfo.FileSystemType = irs.RegionType
	}

	if nasVolume.NasVolumeInstanceStatus != nil && nasVolume.NasVolumeInstanceStatus.Code != nil {
		if strings.EqualFold(*nasVolume.NasVolumeInstanceStatusName, "created") {
			fileSystemInfo.Status = irs.FileSystemStatus(irs.FileSystemAvailable)
		} else if strings.EqualFold(*nasVolume.NasVolumeInstanceStatusName, "creating") {
			fileSystemInfo.Status = irs.FileSystemStatus(irs.FileSystemCreating)
		}
	}

	// Set Capacity (NCP returns size in bytes, convert to GB.)
	if nasVolume.VolumeSize != nil {
		fileSystemInfo.CapacityGB = int64(*nasVolume.VolumeSize) / (1024 * 1024 * 1024)
	}

	// NCP 'NFS' FileSystem supports NFS version '3.0
	if nasVolume.VolumeAllotmentProtocolType != nil && nasVolume.VolumeAllotmentProtocolType.Code != nil {
		protocol := *nasVolume.VolumeAllotmentProtocolType.Code
		if protocol == "NFS" {
			fileSystemInfo.NFSVersion = DefaultNFSVersion
		}
	}

	if nasVolume.IsEncryptedVolume != nil {
		fileSystemInfo.Encryption = *nasVolume.IsEncryptedVolume
	}
	
	if nasVolume.CreateDate != nil {
		createdTime, err := time.Parse("2006-01-02T15:04:05-0700", *nasVolume.CreateDate)
		if err != nil {
			cblogger.Warnf("Failed to parse CreateDate [%s]: %v", *nasVolume.CreateDate, err)
		} else {
			fileSystemInfo.CreatedTime = createdTime
		}
	}

	fileSystemInfo.KeyValueList = irs.StructToKeyValueList(nasVolume)

	return fileSystemInfo, nil
}

func (filesystemHandler *NcpVpcFileSystemHandler) GetFileSystem(iid irs.IID) (irs.FileSystemInfo, error) {
	cblogger.Info("NCP VPC Driver: called GetFileSystem()")

	if iid.SystemId == "" {
		newErr := fmt.Errorf("Invalid IID - SystemId is required")
		cblogger.Error(newErr)
		return irs.FileSystemInfo{}, newErr
	}

	req := vnas.GetNasVolumeInstanceDetailRequest{
		RegionCode:          ncloud.String(filesystemHandler.RegionInfo.Region),
		NasVolumeInstanceNo: ncloud.String(iid.SystemId),
	}

	result, err := filesystemHandler.VNASClient.V2Api.GetNasVolumeInstanceDetail(&req)
	if err != nil {
		cblogger.Errorf("Failed to get NAS volume instance detail [%s]: %v", iid.SystemId, err)
		return irs.FileSystemInfo{}, err
	}

	if len(result.NasVolumeInstanceList) == 0 {
		newErr := fmt.Errorf("NAS volume instance not found: %s", iid.SystemId)
		cblogger.Error(newErr)
		return irs.FileSystemInfo{}, newErr
	}

	nasVolume := result.NasVolumeInstanceList[0]
	if nasVolume == nil {
		newErr := fmt.Errorf("NAS volume instance is nil: %s", iid.SystemId)
		cblogger.Error(newErr)
		return irs.FileSystemInfo{}, newErr
	}

	fileSystemInfo, err := filesystemHandler.mapNasVolumeToFileSystemInfo(nasVolume)
	if err != nil {
		cblogger.Errorf("Failed to convert NAS volume instance to FileSystemInfo: %v", err)
		return irs.FileSystemInfo{}, err
	}

	cblogger.Infof("Successfully retrieved NAS volume instance [%s]", iid.SystemId)
	return fileSystemInfo, nil
}

func (filesystemHandler *NcpVpcFileSystemHandler) DeleteFileSystem(iid irs.IID) (bool, error) {
	cblogger.Info("NCP VPC Driver: called DeleteFileSystem()")

	if iid.SystemId == "" {
		newErr := fmt.Errorf("Invalid IID - SystemId is required")
		cblogger.Error(newErr)
		return false, newErr
	}

	deleteReq := vnas.DeleteNasVolumeInstancesRequest{
		RegionCode: ncloud.String(filesystemHandler.RegionInfo.Region),
		NasVolumeInstanceNoList: []*string{
			ncloud.String(iid.SystemId),
		},
		IsAsync: ncloud.Bool(false),
	}

	result, err := filesystemHandler.VNASClient.V2Api.DeleteNasVolumeInstances(&deleteReq)
	if err != nil {
		wrappedErr := fmt.Errorf("Failed to delete NAS volume instance [%s]: %w", iid.SystemId, err)
		cblogger.Error(wrappedErr)
		return false, wrappedErr
	}

	if result == nil {
		newErr := fmt.Errorf("Delete NAS volume instance returned empty response [%s]", iid.SystemId)
		cblogger.Error(newErr)
		return false, newErr
	}

	if result.ReturnCode != nil && *result.ReturnCode != "0" {
		newErr := fmt.Errorf("Delete NAS volume instance failed [%s]: returnCode=%s, returnMessage=%s", iid.SystemId, ncloud.StringValue(result.ReturnCode), ncloud.StringValue(result.ReturnMessage))
		cblogger.Error(newErr)
		return false, newErr
	}

	cblogger.Infof("Successfully deleted NAS volume instance [%s]", iid.SystemId)
	return true, nil
}

func (filesystemHandler *NcpVpcFileSystemHandler) AddAccessSubnet(fsIID irs.IID, subnetIID irs.IID) (irs.FileSystemInfo, error) {
	cblogger.Info("NCP VPC Driver: called AddAccessSubnet()")

	return irs.FileSystemInfo{}, fmt.Errorf("AddAccessSubnet is not supported in OpenStack - use CreateFileSystem with subnet specification")
}

func (filesystemHandler *NcpVpcFileSystemHandler) RemoveAccessSubnet(fsIID irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger.Info("NCP VPC Driver: called RemoveAccessSubnet()")

	return false, fmt.Errorf("RemoveAccessSubnet is not supported in OpenStack - recreate filesystem with different subnet")
}

func (filesystemHandler *NcpVpcFileSystemHandler) ListAccessSubnet(fsIID irs.IID) ([]irs.IID, error) {
	cblogger.Info("NCP VPC Driver: called ListAccessSubnet()")

	// var subnetList []irs.IID
	// if share.ShareNetworkID != "" {
	// 	shareNetwork, err := sharenetworks.Get(filesystemHandler.SharedFileSystemClient, share.ShareNetworkID).Extract()
	// 	if err == nil && shareNetwork.NeutronSubnetID != "" {
	// 		subnet, err := filesystemHandler.findSubnetByID(shareNetwork.NeutronSubnetID)
	// 		if err == nil {
	// 			subnetIID := irs.IID{
	// 				NameId:   subnet.Name,
	// 				SystemId: subnet.ID,
	// 			}
	// 			subnetList = append(subnetList, subnetIID)
	// 		}
	// 	}
	// }

	// return subnetList, nil

	return nil, nil
}

func (filesystemHandler *NcpVpcFileSystemHandler) ScheduleBackup(reqInfo irs.FileSystemBackupInfo) (irs.FileSystemBackupInfo, error) {
	cblogger.Info("NCP VPC Driver: called ScheduleBackup()")

	return irs.FileSystemBackupInfo{}, fmt.Errorf("backup scheduling is not supported in OpenStack Manila")
}

func (filesystemHandler *NcpVpcFileSystemHandler) OnDemandBackup(fsIID irs.IID) (irs.FileSystemBackupInfo, error) {
	cblogger.Info("NCP VPC Driver: called OnDemandBackup()")

	return irs.FileSystemBackupInfo{}, fmt.Errorf("on-demand backup is not supported in OpenStack Manila")
}

func (filesystemHandler *NcpVpcFileSystemHandler) ListBackup(fsIID irs.IID) ([]irs.FileSystemBackupInfo, error) {
	cblogger.Info("NCP VPC Driver: called ListBackup()")

	return nil, fmt.Errorf("backup listing is not supported in OpenStack Manila")
}

func (filesystemHandler *NcpVpcFileSystemHandler) GetBackup(fsIID irs.IID, backupID string) (irs.FileSystemBackupInfo, error) {
	cblogger.Info("NCP VPC Driver: called GetBackup()")

	return irs.FileSystemBackupInfo{}, fmt.Errorf("backup retrieval is not supported in OpenStack Manila")
}

func (filesystemHandler *NcpVpcFileSystemHandler) DeleteBackup(fsIID irs.IID, backupID string) (bool, error) {
	cblogger.Info("NCP VPC Driver: called DeleteBackup()")

	return false, fmt.Errorf("backup deletion is not supported in OpenStack Manila")
}
