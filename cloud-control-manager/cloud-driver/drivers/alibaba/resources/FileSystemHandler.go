package resources

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/nas"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AlibabaFileSystemHandler struct {
	Region     idrv.RegionInfo
	Client     *nas.Client
	TagHandler *AlibabaTagHandler
}

// GetMetaInfo returns metadata about the file system capabilities
func (fileSystemHandler *AlibabaFileSystemHandler) GetMetaInfo() (irs.FileSystemMetaInfo, error) {
	cblogger.Debug("Alibaba Cloud NAS GetMetaInfo() called")

	hiscallInfo := GetCallLogScheme(fileSystemHandler.Region, call.FILESYSTEM, "GetMetaInfo", "GetMetaInfo()")
	start := call.Start()

	metaInfo := irs.FileSystemMetaInfo{
		SupportsFileSystemType: map[irs.FileSystemType]bool{
			irs.RegionType:          false, // Alibaba NAS is zone-based
			irs.ZoneType:            true,  // Alibaba NAS supports zone-based deployment
			irs.RegionVPCBasedType:  false, // Not supported
			irs.RegionZoneBasedType: true,  // Alibaba NAS is zone-based
		},
		SupportsVPC: map[irs.RSType]bool{
			irs.VPC: true,
		},
		SupportsNFSVersion: []string{"3.0", "4.0"}, // Alibaba NAS supports NFS v3 and v4
		SupportsCapacity:   true,                   // Alibaba NAS supports capacity specification
		CapacityGBOptions: map[string]irs.CapacityGBRange{
			"GeneralPurpose": {Min: 100, Max: 102400}, // General Purpose NAS capacity range
			"Extreme":        {Min: 100, Max: 102400}, // Extreme NAS capacity range
		},
		PerformanceOptions: map[string][]string{
			"RequiredFields": {
				"StorageType",  // Required field: StorageType must be specified
				"ProtocolType", // Required field: ProtocolType must be specified
			},
			"OptionalFields": {
				"Capacity", // Optional: Capacity in GB
			},
			"StorageType": {
				"Capacity",    // Capacity-based storage (pay for storage used)
				"Performance", // Performance-based storage (pay for performance)
			},
			"ProtocolType": {
				"NFS", // NFS protocol
				"SMB", // SMB protocol
			},
			"Capacity": {
				"Min:100",    // Minimum capacity (GB)
				"Max:102400", // Maximum capacity (GB)
			},
			"Constraints": {
				"ZoneType:Required",               // Zone is required for Alibaba NAS
				"VPC:Required",                    // VPC is required
				"StorageType:Capacity:Min:100",    // Minimum capacity for Capacity storage
				"StorageType:Performance:Min:100", // Minimum capacity for Performance storage
			},
			"Examples": {
				"GeneralPurpose+Capacity+NFS:StorageType:Capacity,ProtocolType:NFS,Capacity:1024",
				"GeneralPurpose+Performance+NFS:StorageType:Performance,ProtocolType:NFS,Capacity:1024",
				"Extreme+Capacity+NFS:StorageType:Capacity,ProtocolType:NFS,Capacity:2048",
				"Extreme+Performance+NFS:StorageType:Performance,ProtocolType:NFS,Capacity:2048",
			},
		},
	}

	LoggingInfo(hiscallInfo, start)
	return metaInfo, nil
}

// ListIID returns list of file system IDs
func (fileSystemHandler *AlibabaFileSystemHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Debug("Alibaba Cloud NAS ListIID() called")

	if fileSystemHandler.Client == nil {
		return nil, errors.New("NAS client is not initialized")
	}

	request := nas.CreateDescribeFileSystemsRequest()
	request.Scheme = "https"

	hiscallInfo := GetCallLogScheme(fileSystemHandler.Region, call.FILESYSTEM, "ListIID", "DescribeFileSystems()")
	start := call.Start()

	response, err := fileSystemHandler.Client.DescribeFileSystems(request)

	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	var iidList []*irs.IID
	for _, fs := range response.FileSystems.FileSystem {
		iid := irs.IID{
			NameId:   fs.Description, // Alibaba NAS uses Description as name
			SystemId: fs.FileSystemId,
		}
		iidList = append(iidList, &iid)
	}

	return iidList, nil
}

// CreateFileSystem creates a new Alibaba Cloud NAS file system
func (fileSystemHandler *AlibabaFileSystemHandler) CreateFileSystem(reqInfo irs.FileSystemInfo) (irs.FileSystemInfo, error) {
	cblogger.Debug("Alibaba Cloud NAS CreateFileSystem() called")

	if fileSystemHandler.Client == nil {
		return irs.FileSystemInfo{}, errors.New("NAS client is not initialized")
	}

	// ================================
	// Validate VPC requirement
	// ================================
	if reqInfo.VpcIID.SystemId == "" {
		return irs.FileSystemInfo{}, errors.New("VPC is required for Alibaba Cloud NAS file system creation")
	}

	// ================================
	// Validate Zone requirement for Alibaba NAS
	// ================================
	if reqInfo.Zone == "" {
		return irs.FileSystemInfo{}, errors.New("Zone is required for Alibaba Cloud NAS file system creation")
	}

	// ================================
	// Validate NFS version if provided
	// ================================
	if reqInfo.NFSVersion != "" {
		metaInfo, err := fileSystemHandler.GetMetaInfo()
		if err != nil {
			cblogger.Errorf("Failed to get meta info for NFS version validation: %v", err)
			return irs.FileSystemInfo{}, fmt.Errorf("failed to validate NFS version: %v", err)
		}

		supported := false
		for _, supportedVersion := range metaInfo.SupportsNFSVersion {
			if reqInfo.NFSVersion == supportedVersion {
				supported = true
				break
			}
		}

		if !supported {
			return irs.FileSystemInfo{}, fmt.Errorf("Alibaba Cloud NAS only supports NFS versions: %v", metaInfo.SupportsNFSVersion)
		}
	} else {
		reqInfo.NFSVersion = "4.0" // Alibaba NAS default
		cblogger.Info("Using default NFS version: 4.0")
	}

	// ================================
	// Set default values for basic setup
	// ================================
	if reqInfo.FileSystemType == "" {
		reqInfo.FileSystemType = irs.ZoneType // Alibaba NAS is zone-based
	}

	// ================================
	// Validate and set performance options
	// ================================
	storageType := "Capacity" // Default storage type
	protocolType := "NFS"     // Default protocol type
	capacity := int64(100)    // Default capacity in GB

	if reqInfo.PerformanceInfo != nil {
		metaInfo, err := fileSystemHandler.GetMetaInfo()
		if err != nil {
			cblogger.Errorf("Failed to get meta info for performance validation: %v", err)
			return irs.FileSystemInfo{}, fmt.Errorf("failed to validate performance options: %v", err)
		}

		validOptions := metaInfo.PerformanceOptions

		// Validate required fields
		requiredFields := validOptions["RequiredFields"]
		for _, field := range requiredFields {
			if _, exists := reqInfo.PerformanceInfo[field]; !exists {
				return irs.FileSystemInfo{}, fmt.Errorf("required field '%s' is missing in PerformanceInfo", field)
			}
		}

		// Validate storage type
		if st, exists := reqInfo.PerformanceInfo["StorageType"]; exists {
			validStorageTypes := validOptions["StorageType"]
			storageTypeValid := false
			for _, validType := range validStorageTypes {
				if validType == st {
					storageTypeValid = true
					storageType = st
					break
				}
			}
			if !storageTypeValid {
				return irs.FileSystemInfo{}, fmt.Errorf("invalid StorageType '%s'. Valid options: %v", st, validStorageTypes)
			}
		}

		// Validate protocol type
		if pt, exists := reqInfo.PerformanceInfo["ProtocolType"]; exists {
			validProtocolTypes := validOptions["ProtocolType"]
			protocolTypeValid := false
			for _, validType := range validProtocolTypes {
				if validType == pt {
					protocolTypeValid = true
					protocolType = pt
					break
				}
			}
			if !protocolTypeValid {
				return irs.FileSystemInfo{}, fmt.Errorf("invalid ProtocolType '%s'. Valid options: %v", pt, validProtocolTypes)
			}
		}

		// Validate capacity if specified
		if cap, exists := reqInfo.PerformanceInfo["Capacity"]; exists {
			if capInt, err := strconv.ParseInt(cap, 10, 64); err == nil {
				capacityRange := metaInfo.CapacityGBOptions["GeneralPurpose"] // Use GeneralPurpose range
				if capInt >= capacityRange.Min && capInt <= capacityRange.Max {
					capacity = capInt
				} else {
					return irs.FileSystemInfo{}, fmt.Errorf("capacity must be between %d and %d GB", capacityRange.Min, capacityRange.Max)
				}
			} else {
				return irs.FileSystemInfo{}, errors.New("invalid capacity value")
			}
		}

		cblogger.Infof("Performance settings: StorageType=%s, ProtocolType=%s, Capacity=%dGB", storageType, protocolType, capacity)
	}

	// ================================
	// Create NAS file system
	// ================================
	request := nas.CreateCreateFileSystemRequest()
	request.Scheme = "https"
	request.ProtocolType = protocolType
	request.StorageType = storageType
	request.ZoneId = reqInfo.Zone
	request.VpcId = reqInfo.VpcIID.SystemId
	request.Description = reqInfo.IId.NameId

	// Set capacity based on storage type
	if storageType == "Capacity" {
		request.Capacity = requests.NewInteger(int(capacity))
	}

	hiscallInfo := GetCallLogScheme(fileSystemHandler.Region, call.FILESYSTEM, reqInfo.IId.NameId, "CreateFileSystem()")
	start := call.Start()

	response, err := fileSystemHandler.Client.CreateFileSystem(request)

	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	if response.FileSystemId == "" {
		return irs.FileSystemInfo{}, errors.New("failed to create file system: invalid response")
	}

	// Wait for file system to be available
	err = fileSystemHandler.waitUntilFileSystemAvailable(response.FileSystemId)
	if err != nil {
		cblogger.Error(err)
		return irs.FileSystemInfo{}, err
	}

	// ================================
	// Create mount targets if specified
	// ================================
	if len(reqInfo.AccessSubnetList) > 0 {
		cblogger.Info("Creating mount targets using AccessSubnetList")

		for _, subnetIID := range reqInfo.AccessSubnetList {
			err := fileSystemHandler.createMountTarget(
				irs.IID{NameId: reqInfo.IId.NameId, SystemId: response.FileSystemId},
				subnetIID,
			)
			if err != nil {
				cblogger.Errorf("Failed to create mount target for subnet %s: %v", subnetIID.SystemId, err)
			}
		}
	}

	// Get the created file system info
	fileSystemInfo, err := fileSystemHandler.GetFileSystem(irs.IID{NameId: reqInfo.IId.NameId, SystemId: response.FileSystemId})
	if err != nil {
		return irs.FileSystemInfo{}, err
	}

	return fileSystemInfo, nil
}

// ListFileSystem returns list of all file systems
func (fileSystemHandler *AlibabaFileSystemHandler) ListFileSystem() ([]*irs.FileSystemInfo, error) {
	cblogger.Debug("Alibaba Cloud NAS ListFileSystem() called")

	hiscallInfo := GetCallLogScheme(fileSystemHandler.Region, call.FILESYSTEM, "ListFileSystem", "ListFileSystem()")
	start := call.Start()

	iidList, err := fileSystemHandler.ListIID()
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	var fileSystemList []*irs.FileSystemInfo
	for _, iid := range iidList {
		fileSystemInfo, err := fileSystemHandler.GetFileSystem(*iid)
		if err != nil {
			cblogger.Errorf("Failed to get file system info for %s: %v", iid.SystemId, err)
			// Continue with other file systems instead of failing completely
			continue
		}
		fileSystemList = append(fileSystemList, &fileSystemInfo)
	}

	LoggingInfo(hiscallInfo, start)
	return fileSystemList, nil
}

// GetFileSystem returns specific file system info
func (fileSystemHandler *AlibabaFileSystemHandler) GetFileSystem(iid irs.IID) (irs.FileSystemInfo, error) {
	cblogger.Debug("Alibaba Cloud NAS GetFileSystem() called")

	if fileSystemHandler.Client == nil {
		return irs.FileSystemInfo{}, errors.New("NAS client is not initialized")
	}

	request := nas.CreateDescribeFileSystemsRequest()
	request.Scheme = "https"
	request.FileSystemId = iid.SystemId

	hiscallInfo := GetCallLogScheme(fileSystemHandler.Region, call.FILESYSTEM, iid.SystemId, "DescribeFileSystems()")
	start := call.Start()

	response, err := fileSystemHandler.Client.DescribeFileSystems(request)

	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	if len(response.FileSystems.FileSystem) == 0 {
		return irs.FileSystemInfo{}, errors.New("file system not found")
	}

	fileSystemInfo, err := fileSystemHandler.convertToFileSystemInfo(&response.FileSystems.FileSystem[0])
	if err != nil {
		return irs.FileSystemInfo{}, err
	}

	// Get tags using TagHandler
	if fileSystemHandler.TagHandler != nil {
		tagList, err := fileSystemHandler.TagHandler.ListTag(irs.FILESYSTEM, fileSystemInfo.IId)
		if err != nil {
			cblogger.Errorf("Failed to get tags: %v", err)
			fileSystemInfo.TagList = []irs.KeyValue{}
		} else {
			fileSystemInfo.TagList = tagList
		}
	} else {
		cblogger.Warn("TagHandler is nil, skipping tag retrieval")
		fileSystemInfo.TagList = []irs.KeyValue{}
	}

	return fileSystemInfo, nil
}

// DeleteFileSystem deletes the specified file system
func (fileSystemHandler *AlibabaFileSystemHandler) DeleteFileSystem(iid irs.IID) (bool, error) {
	cblogger.Debug("Alibaba Cloud NAS DeleteFileSystem() called")

	if fileSystemHandler.Client == nil {
		return false, errors.New("NAS client is not initialized")
	}

	// First, delete all mount targets
	mountTargets, err := fileSystemHandler.listMountTargets(iid.SystemId)
	if err != nil {
		cblogger.Errorf("Failed to list mount targets: %v", err)
		return false, err
	}

	for _, mt := range mountTargets {
		err := fileSystemHandler.deleteMountTarget(mt.MountTargetDomain)
		if err != nil {
			cblogger.Errorf("Failed to delete mount target %s: %v", mt.MountTargetDomain, err)
		}
	}

	// Wait for mount targets to be deleted
	if len(mountTargets) > 0 {
		time.Sleep(30 * time.Second)
	}

	request := nas.CreateDeleteFileSystemRequest()
	request.Scheme = "https"
	request.FileSystemId = iid.SystemId

	hiscallInfo := GetCallLogScheme(fileSystemHandler.Region, call.FILESYSTEM, iid.SystemId, "DeleteFileSystem()")
	start := call.Start()

	_, err = fileSystemHandler.Client.DeleteFileSystem(request)

	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

// AddAccessSubnet adds a subnet to the file system access list
func (fileSystemHandler *AlibabaFileSystemHandler) AddAccessSubnet(iid irs.IID, subnetIID irs.IID) (irs.FileSystemInfo, error) {
	cblogger.Debug("Alibaba Cloud NAS AddAccessSubnet() called")

	if fileSystemHandler.Client == nil {
		return irs.FileSystemInfo{}, errors.New("NAS client is not initialized")
	}

	err := fileSystemHandler.createMountTarget(iid, subnetIID)
	if err != nil {
		return irs.FileSystemInfo{}, err
	}

	return fileSystemHandler.GetFileSystem(iid)
}

// RemoveAccessSubnet removes a subnet from the file system access list
func (fileSystemHandler *AlibabaFileSystemHandler) RemoveAccessSubnet(iid irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger.Debug("Alibaba Cloud NAS RemoveAccessSubnet() called")

	if fileSystemHandler.Client == nil {
		return false, errors.New("NAS client is not initialized")
	}

	hiscallInfo := GetCallLogScheme(fileSystemHandler.Region, call.FILESYSTEM, iid.SystemId, "RemoveAccessSubnet()")
	start := call.Start()

	mountTargets, err := fileSystemHandler.listMountTargets(iid.SystemId)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	for _, mt := range mountTargets {
		if mt.VswId == subnetIID.SystemId {
			err := fileSystemHandler.deleteMountTarget(mt.MountTargetDomain)
			if err != nil {
				cblogger.Error(err)
				LoggingError(hiscallInfo, err)
				return false, err
			}
			LoggingInfo(hiscallInfo, start)
			return true, nil
		}
	}

	LoggingError(hiscallInfo, errors.New("mount target not found for the specified subnet"))
	return false, errors.New("mount target not found for the specified subnet")
}

// ListAccessSubnet returns list of subnets that can access the file system
func (fileSystemHandler *AlibabaFileSystemHandler) ListAccessSubnet(iid irs.IID) ([]irs.IID, error) {
	cblogger.Debug("Alibaba Cloud NAS ListAccessSubnet() called")

	if fileSystemHandler.Client == nil {
		return nil, errors.New("NAS client is not initialized")
	}

	hiscallInfo := GetCallLogScheme(fileSystemHandler.Region, call.FILESYSTEM, iid.SystemId, "ListAccessSubnet()")
	start := call.Start()

	mountTargets, err := fileSystemHandler.listMountTargets(iid.SystemId)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	var subnetList []irs.IID
	for _, mt := range mountTargets {
		subnetIID := irs.IID{
			SystemId: mt.VswId,
		}
		subnetList = append(subnetList, subnetIID)
	}

	LoggingInfo(hiscallInfo, start)
	return subnetList, nil
}

// Backup management methods (not implemented for Alibaba Cloud NAS)
func (fileSystemHandler *AlibabaFileSystemHandler) ScheduleBackup(reqInfo irs.FileSystemBackupInfo) (irs.FileSystemBackupInfo, error) {
	return irs.FileSystemBackupInfo{}, errors.New("scheduled backups are not supported in Alibaba Cloud NAS")
}

func (fileSystemHandler *AlibabaFileSystemHandler) OnDemandBackup(fsIID irs.IID) (irs.FileSystemBackupInfo, error) {
	return irs.FileSystemBackupInfo{}, errors.New("on-demand backups are not supported in Alibaba Cloud NAS")
}

func (fileSystemHandler *AlibabaFileSystemHandler) ListBackup(fsIID irs.IID) ([]irs.FileSystemBackupInfo, error) {
	return nil, errors.New("backup listing is not supported in Alibaba Cloud NAS")
}

func (fileSystemHandler *AlibabaFileSystemHandler) GetBackup(fsIID irs.IID, backupID string) (irs.FileSystemBackupInfo, error) {
	return irs.FileSystemBackupInfo{}, errors.New("backup retrieval is not supported in Alibaba Cloud NAS")
}

func (fileSystemHandler *AlibabaFileSystemHandler) DeleteBackup(fsIID irs.IID, backupID string) (bool, error) {
	return false, errors.New("backup deletion is not supported in Alibaba Cloud NAS")
}

// Helper functions

func (fileSystemHandler *AlibabaFileSystemHandler) waitUntilFileSystemAvailable(fileSystemId string) error {
	cblogger.Info("Waiting for file system to be available...")

	if fileSystemHandler.Client == nil {
		return errors.New("NAS client is not initialized")
	}

	request := nas.CreateDescribeFileSystemsRequest()
	request.Scheme = "https"
	request.FileSystemId = fileSystemId

	for {
		response, err := fileSystemHandler.Client.DescribeFileSystems(request)
		if err != nil {
			return err
		}

		if len(response.FileSystems.FileSystem) > 0 {
			fs := response.FileSystems.FileSystem[0]
			if fs.Status == "Running" {
				cblogger.Info("File system is now available")
				return nil
			} else if fs.Status == "Stopped" || fs.Status == "Error" {
				return errors.New("file system creation failed")
			}
		}

		time.Sleep(10 * time.Second)
	}
}

func (fileSystemHandler *AlibabaFileSystemHandler) listMountTargets(fileSystemId string) ([]nas.MountTarget, error) {
	if fileSystemHandler.Client == nil {
		return nil, errors.New("NAS client is not initialized")
	}

	request := nas.CreateDescribeMountTargetsRequest()
	request.Scheme = "https"
	request.FileSystemId = fileSystemId

	response, err := fileSystemHandler.Client.DescribeMountTargets(request)
	if err != nil {
		return nil, err
	}

	return response.MountTargets.MountTarget, nil
}

func (fileSystemHandler *AlibabaFileSystemHandler) deleteMountTarget(mountTargetDomain string) error {
	if fileSystemHandler.Client == nil {
		return errors.New("NAS client is not initialized")
	}

	request := nas.CreateDeleteMountTargetRequest()
	request.Scheme = "https"
	request.MountTargetDomain = mountTargetDomain

	_, err := fileSystemHandler.Client.DeleteMountTarget(request)
	return err
}

func (fileSystemHandler *AlibabaFileSystemHandler) createMountTarget(iid irs.IID, subnetIID irs.IID) error {
	cblogger.Debug("Alibaba Cloud NAS createMountTarget() called")

	if fileSystemHandler.Client == nil {
		return errors.New("NAS client is not initialized")
	}

	request := nas.CreateCreateMountTargetRequest()
	request.Scheme = "https"
	request.FileSystemId = iid.SystemId
	request.VSwitchId = subnetIID.SystemId

	hiscallInfo := GetCallLogScheme(fileSystemHandler.Region, call.FILESYSTEM, iid.SystemId, "CreateMountTarget()")
	start := call.Start()

	_, err := fileSystemHandler.Client.CreateMountTarget(request)

	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return err
	}
	LoggingInfo(hiscallInfo, start)

	// Wait for mount target to be available
	time.Sleep(10 * time.Second)

	return nil
}

func (fileSystemHandler *AlibabaFileSystemHandler) convertToFileSystemInfo(fs *nas.FileSystem) (irs.FileSystemInfo, error) {
	if fs == nil {
		return irs.FileSystemInfo{}, errors.New("file system is nil")
	}

	// Get mount targets
	mountTargets, err := fileSystemHandler.listMountTargets(fs.FileSystemId)
	if err != nil {
		cblogger.Errorf("Failed to get mount targets: %v", err)
		// Continue with empty mount targets instead of failing
		mountTargets = []nas.MountTarget{}
	}

	var mountTargetList []irs.MountTargetInfo
	var accessSubnetList []irs.IID
	for _, mt := range mountTargets {
		mountTargetInfo := irs.MountTargetInfo{
			SubnetIID: irs.IID{SystemId: mt.VswId},
			Endpoint:  mt.MountTargetDomain,
		}

		// Create mount command example
		if fs.ProtocolType == "NFS" {
			mountTargetInfo.MountCommandExample = fmt.Sprintf("sudo mount -t nfs %s:/ /mnt/nas", mt.MountTargetDomain)
		} else if fs.ProtocolType == "SMB" {
			mountTargetInfo.MountCommandExample = fmt.Sprintf("sudo mount -t cifs //%s /mnt/nas -o username=your_username,password=your_password", mt.MountTargetDomain)
		}

		// Add complete Alibaba NAS mount target object information
		mountTargetInfo.KeyValueList = irs.StructToKeyValueList(mt)

		mountTargetList = append(mountTargetList, mountTargetInfo)
		accessSubnetList = append(accessSubnetList, irs.IID{SystemId: mt.VswId})
	}

	// Convert performance info
	performanceInfo := make(map[string]string)
	performanceInfo["StorageType"] = fs.StorageType
	performanceInfo["ProtocolType"] = fs.ProtocolType
	if fs.Capacity != 0 {
		performanceInfo["Capacity"] = fmt.Sprintf("%d", fs.Capacity)
	}

	// Convert status
	var status irs.FileSystemStatus
	switch fs.Status {
	case "Creating":
		status = irs.FileSystemCreating
	case "Running":
		status = irs.FileSystemAvailable
	case "Stopped":
		status = irs.FileSystemError
	default:
		status = irs.FileSystemError
	}

	// Create additional key-value list
	var keyValueList []irs.KeyValue
	keyValueList = append(keyValueList, irs.KeyValue{Key: "ZoneId", Value: fs.ZoneId})
	keyValueList = append(keyValueList, irs.KeyValue{Key: "VpcId", Value: fs.VpcId})
	keyValueList = append(keyValueList, irs.KeyValue{Key: "ChargeType", Value: fs.ChargeType})

	// Add complete Alibaba NAS object information
	alibabaFileSystemKeyValueList := irs.StructToKeyValueList(fs)
	keyValueList = append(keyValueList, alibabaFileSystemKeyValueList...)

	fileSystemInfo := irs.FileSystemInfo{
		IId: irs.IID{
			NameId:   fs.Description,
			SystemId: fs.FileSystemId,
		},
		Region:           fileSystemHandler.Region.Region,
		Zone:             fs.ZoneId,
		VpcIID:           irs.IID{SystemId: fs.VpcId},
		AccessSubnetList: accessSubnetList,
		Encryption:       false, // Alibaba NAS doesn't support encryption in basic setup
		TagList:          []irs.KeyValue{},
		FileSystemType:   irs.ZoneType, // Alibaba NAS is always zone-based
		NFSVersion:       "4.0",        // Default NFS version
		CapacityGB:       int64(fs.Capacity),
		PerformanceInfo:  performanceInfo,
		Status:           status,
		UsedSizeGB:       0, // Alibaba NAS doesn't provide used size
		MountTargetList:  mountTargetList,
		CreatedTime:      time.Now(), // Alibaba NAS doesn't provide creation time
		KeyValueList:     keyValueList,
	}

	return fileSystemInfo, nil
}
