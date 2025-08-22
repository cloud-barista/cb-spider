package resources

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
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
		SupportsCapacity:   true,                   // Alibaba NAS supports capacity specification for extreme type
		CapacityGBOptions: map[string]irs.CapacityGBRange{
			"standard": {Min: 0, Max: 0},        // Standard NAS: capacity managed automatically by service
			"extreme":  {Min: 100, Max: 262144}, // Extreme NAS: 100 GiB to 256 TiB (262,144 GiB)
		},
		PerformanceOptions: map[string][]string{
			"RequiredFields": {
				"StorageType", // Required: StorageType must be specified (varies by region)
			},
			"OptionalFields": {
				"FileSystemType", // Optional: NAS file system type (standard/extreme/cpfs)
				"ProtocolType",   // Optional: NFS or SMB
				"Capacity",       // Optional: Capacity in GB (only for extreme type)
			},
			"FileSystemType": {
				"standard (default): General-purpose NAS file system - capacity managed automatically",
				"extreme: Extreme NAS file system - capacity must be specified (100-262144 GB)",
				"cpfs: Cloud Parallel File Storage (CPFS) file system",
			},
			"StorageType": {
				"Capacity",    // Capacity-based storage (pay for storage used)
				"Premium",     // Premium storage (pay for performance tier)
				"Performance", // Performance-based storage (pay for performance tier)
			},
			"ProtocolType": {
				"NFS", // NFS protocol (v3.0, v4.0)
				"SMB", // SMB protocol
			},
			"BasicSetup": {
				"FileSystemType:standard", // Default: General-purpose NAS
				"ProtocolType:NFS",        // Default: NFS protocol
				"NFSVersion:4.0",          // Default: NFS 4.0
				"StorageType:Required",    // Required: Must be specified by user
			},
			"AdvancedSetup": {
				"FileSystemType:standard,extreme,cpfs",     // Choose NAS file system type
				"StorageType:Capacity,Premium,Performance", // Choose storage type (varies by region)
				"ProtocolType:NFS,SMB",                     // Choose protocol type
				"Capacity:100-262144GB",                    // Capacity range (only for extreme type)
			},
			"CapacityRules": {
				"standard:auto",        // General-purpose NAS: capacity managed automatically
				"extreme:100-262144GB", // Extreme NAS: capacity must be specified
				"extreme:required",     // Extreme NAS: capacity is required
				"extreme:purchased",    // Extreme NAS: billed based on purchased capacity
			},
			"Examples": {
				"Basic:StorageType:Capacity (FileSystemType defaults to standard)",
				"Advanced:FileSystemType:extreme,StorageType:Capacity,Capacity:1024",
				"Advanced:FileSystemType:standard,StorageType:Premium,ProtocolType:NFS",
			},
			"Notes": {
				"StorageType is required and varies by region - check Alibaba Cloud console",
				"FileSystemType 'standard' (default): General-purpose NAS, capacity auto-managed",
				"FileSystemType 'extreme': Extreme NAS, capacity required (100-262144 GB)",
				"Extreme NAS: billed based on purchased capacity, not actual usage",
				"FileSystemType 'cpfs': Cloud Parallel File Storage (CPFS)",
				"ProtocolType NFS supports v3.0 and v4.0",
				"Zone and VPC are required for all file systems",
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
		cblogger.Info("Using default FileSystemType: ZoneType")
	}

	// ================================
	// Validate and set performance options
	// ================================
	protocolType := "NFS"        // Default protocol type
	storageType := ""            // StorageType is required - no default
	fileSystemType := "standard" // Default NAS file system type (General-purpose NAS)
	capacity := int64(100)       // Default capacity in GB (for extreme type)

	// Get meta info for validation
	metaInfo, err := fileSystemHandler.GetMetaInfo()
	if err != nil {
		cblogger.Errorf("Failed to get meta info for validation: %v", err)
		return irs.FileSystemInfo{}, fmt.Errorf("failed to get meta info: %v", err)
	}

	validOptions := metaInfo.PerformanceOptions

	// ================================
	// Process PerformanceInfo for advanced setup
	// ================================
	if reqInfo.PerformanceInfo != nil && len(reqInfo.PerformanceInfo) > 0 {
		// Validate FileSystemType if provided (NAS file system type)
		if fst, exists := reqInfo.PerformanceInfo["FileSystemType"]; exists {
			// Extract the actual type from the description
			var actualType string
			switch {
			case strings.Contains(fst, "standard"):
				actualType = "standard"
			case strings.Contains(fst, "extreme"):
				actualType = "extreme"
			case strings.Contains(fst, "cpfs"):
				actualType = "cpfs"
			default:
				actualType = fst // Use as-is if not in description format
			}

			// Validate against API-supported types
			validTypes := []string{"standard", "extreme", "cpfs"}
			fileSystemTypeValid := false
			for _, validType := range validTypes {
				if actualType == validType {
					fileSystemTypeValid = true
					fileSystemType = actualType
					break
				}
			}
			if !fileSystemTypeValid {
				return irs.FileSystemInfo{}, fmt.Errorf("invalid FileSystemType '%s'. Valid options: standard, extreme, cpfs", fst)
			}
			cblogger.Infof("Using user-provided FileSystemType: %s", fileSystemType)
		}

		// Validate StorageType (Required field)
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
			cblogger.Infof("Using user-provided StorageType: %s", storageType)
		} else {
			// StorageType is required
			return irs.FileSystemInfo{}, errors.New("StorageType is required. Please specify StorageType in PerformanceInfo (e.g., 'Capacity', 'Premium', or 'Performance')")
		}

		// Validate protocol type if provided
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
		} else {
			cblogger.Info("ProtocolType not provided, using default: NFS")
		}

		// Validate capacity if specified (from PerformanceInfo or CapacityGB)
		var capacitySpecified bool
		var capacityValue string

		// Check PerformanceInfo first
		if cap, exists := reqInfo.PerformanceInfo["Capacity"]; exists {
			capacitySpecified = true
			capacityValue = cap
		} else if reqInfo.CapacityGB > 0 {
			// Check CapacityGB field as fallback
			capacitySpecified = true
			capacityValue = strconv.FormatInt(reqInfo.CapacityGB, 10)
		}

		if capacitySpecified {
			if capInt, err := strconv.ParseInt(capacityValue, 10, 64); err == nil {
				// Capacity validation based on FileSystemType
				if fileSystemType == "extreme" {
					// Extreme NAS: capacity is required and must be 100-262144 GB
					capacityRange := metaInfo.CapacityGBOptions["extreme"]
					if capInt >= capacityRange.Min && capInt <= capacityRange.Max {
						capacity = capInt
						cblogger.Infof("Using user-provided capacity: %d GB for extreme file system", capacity)
					} else {
						return irs.FileSystemInfo{}, fmt.Errorf("capacity for extreme file system must be between %d and %d GB", capacityRange.Min, capacityRange.Max)
					}
				} else {
					// Standard NAS: capacity is not supported (managed automatically)
					cblogger.Warnf("Capacity specification is not supported for standard file system type. Capacity will be managed automatically by the service.")
				}
			} else {
				return irs.FileSystemInfo{}, errors.New("invalid capacity value")
			}
		} else {
			// No capacity specified
			if fileSystemType == "extreme" {
				// Extreme NAS requires capacity
				return irs.FileSystemInfo{}, errors.New("capacity is required for extreme file system type. Please specify Capacity in PerformanceInfo")
			} else {
				// Standard NAS: capacity managed automatically
				cblogger.Info("Capacity not specified for standard file system - will be managed automatically by the service")
			}
		}

		cblogger.Infof("Advanced setup - FileSystemType=%s, StorageType=%s, ProtocolType=%s, Capacity=%dGB", fileSystemType, storageType, protocolType, capacity)
	} else {
		// Basic setup mode - use defaults
		cblogger.Info("Basic setup mode - using default values: FileSystemType=standard, StorageType=Capacity, ProtocolType=NFS")
	}

	// ================================
	// Get default resource group (similar to web console behavior)
	// ================================
	resourceGroupId, err := fileSystemHandler.getDefaultResourceGroupId()
	if err != nil {
		cblogger.Warnf("Failed to get default resource group: %v", err)
		cblogger.Info("Continuing without resource group - Alibaba Cloud will use default")
	} else {
		cblogger.Infof("Using default resource group: %s", resourceGroupId)
	}

	// ================================
	// Create NAS file system
	// ================================
	request := nas.CreateCreateFileSystemRequest()
	request.Scheme = "https"
	request.FileSystemType = fileSystemType
	request.ProtocolType = protocolType
	request.StorageType = storageType
	request.ZoneId = reqInfo.Zone
	request.VpcId = reqInfo.VpcIID.SystemId
	request.Description = reqInfo.IId.NameId

	// Set resource group if available
	if resourceGroupId != "" {
		// Note: Alibaba NAS API might not support ResourceGroupId parameter
		// This is for future compatibility
		cblogger.Infof("Resource group ID: %s (will be used if API supports it)", resourceGroupId)
	}

	// Set capacity only for extreme file system type
	// According to Alibaba Cloud NAS API documentation:
	// - Capacity is only valid and required for extreme file system type
	// - For standard file systems, capacity is managed automatically by the service
	if fileSystemType == "extreme" {
		request.Capacity = requests.NewInteger(int(capacity))
		cblogger.Infof("Setting capacity to %d GB for extreme file system type", capacity)
	} else {
		cblogger.Infof("FileSystemType '%s' selected - capacity will be managed automatically by the service", fileSystemType)
	}

	// Note: Tags will be added after file system creation using TagResources API
	// as CreateFileSystemRequest does not support Tag field directly

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

	cblogger.Infof("Successfully created file system with ID: %s", response.FileSystemId)

	// Wait for file system to be available
	err = fileSystemHandler.waitUntilFileSystemAvailable(response.FileSystemId)
	if err != nil {
		cblogger.Error(err)
		return irs.FileSystemInfo{}, err
	}

	// ================================
	// Add tags if provided using TagHandler
	// ================================
	if reqInfo.TagList != nil && len(reqInfo.TagList) > 0 {
		cblogger.Infof("Adding %d tags to file system %s using TagHandler", len(reqInfo.TagList), response.FileSystemId)

		// Use TagHandler to add tags
		for _, tag := range reqInfo.TagList {
			if tag.Key != "" && tag.Value != "" {
				_, tagErr := fileSystemHandler.TagHandler.AddTag(irs.FILESYSTEM, irs.IID{SystemId: response.FileSystemId}, tag)
				if tagErr != nil {
					cblogger.Errorf("Failed to add tag (Key=%s, Value=%s) to file system %s: %v", tag.Key, tag.Value, response.FileSystemId, tagErr)
					// Don't fail the entire operation for tag errors, just log the warning
					cblogger.Warnf("File system %s created successfully but tag (Key=%s, Value=%s) could not be added", response.FileSystemId, tag.Key, tag.Value)
				} else {
					cblogger.Infof("Successfully added tag (Key=%s, Value=%s) to file system %s", tag.Key, tag.Value, response.FileSystemId)
				}
			}
		}
	}

	// Add VpcId as a tag for future reference
	vpcIdTag := irs.KeyValue{Key: "VpcId", Value: reqInfo.VpcIID.SystemId}
	_, vpcTagErr := fileSystemHandler.TagHandler.AddTag(irs.FILESYSTEM, irs.IID{SystemId: response.FileSystemId}, vpcIdTag)
	if vpcTagErr != nil {
		cblogger.Warnf("Failed to add VpcId tag to file system %s: %v", response.FileSystemId, vpcTagErr)
	} else {
		cblogger.Infof("Successfully added VpcId tag (Value=%s) to file system %s", reqInfo.VpcIID.SystemId, response.FileSystemId)
	}

	// ================================
	// Create mount targets if specified
	// ================================
	if len(reqInfo.AccessSubnetList) > 0 {
		cblogger.Info("Creating mount targets using AccessSubnetList")

		for _, subnetIID := range reqInfo.AccessSubnetList {
			err := fileSystemHandler.createMountTargetWithVPC(
				irs.IID{NameId: reqInfo.IId.NameId, SystemId: response.FileSystemId},
				subnetIID,
				reqInfo.VpcIID.SystemId, // Pass VPC ID from request
			)
			if err != nil {
				cblogger.Errorf("Failed to create mount target for subnet %s: %v", subnetIID.SystemId, err)
				// Mount Target creation is critical - fail the entire operation
				return irs.FileSystemInfo{}, fmt.Errorf("failed to create mount target for subnet %s: %v", subnetIID.SystemId, err)
			} else {
				cblogger.Infof("Successfully created mount target for subnet: %s", subnetIID.SystemId)
			}
		}
	} else {
		cblogger.Info("No AccessSubnetList provided - mount targets will need to be created separately")
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

	// Validate file system IID
	if err := fileSystemHandler.validateFileSystemIID(iid); err != nil {
		return irs.FileSystemInfo{}, err
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

	return fileSystemInfo, nil
}

// extractTagsFromFileSystem extracts tag information directly from the FileSystem object
func extractTagsFromFileSystem(fs *nas.FileSystem) []irs.KeyValue {
	var tagList []irs.KeyValue

	cblogger.Info("=== extractTagsFromFileSystem Debug ===")
	cblogger.Infof("fs.Tags: %+v", fs.Tags)
	cblogger.Infof("fs.Tags.Tag length: %d", len(fs.Tags.Tag))

	// Check if Tags field exists and has content
	if len(fs.Tags.Tag) > 0 {
		for i, tag := range fs.Tags.Tag {
			cblogger.Infof("Tag[%d]: Key='%s', Value='%s'", i, tag.Key, tag.Value)
			if tag.Key != "" && tag.Value != "" {
				tagList = append(tagList, irs.KeyValue{Key: tag.Key, Value: tag.Value})
				cblogger.Infof("Successfully extracted tag: Key=%s, Value=%s", tag.Key, tag.Value)
			} else {
				cblogger.Infof("Skipping tag[%d] - empty key or value", i)
			}
		}
	} else {
		cblogger.Info("No tags found in fs.Tags.Tag")
	}

	cblogger.Infof("Total extracted tags from fs object: %d", len(tagList))
	cblogger.Info("=== End extractTagsFromFileSystem Debug ===")
	return tagList
}

// // extractTagsFromKeyValueList extracts tag information from KeyValueList (fallback method)
// func extractTagsFromKeyValueList(keyValueList []irs.KeyValue) []irs.KeyValue {
// 	var tagList []irs.KeyValue

// 	for _, kv := range keyValueList {
// 		if kv.Key == "Tags" && kv.Value != "" {
// 			// Tags 값에서 태그 정보 파싱
// 			// 예: "{Tag:[{Key:Tag1,Value:tag-test-1,FileSystemIds:{FileSystemId:null}}]}"
// 			cblogger.Debugf("Parsing tags from KeyValueList: %s", kv.Value)

// 			// 더 정확한 파싱을 위해 정규식 사용
// 			tagPattern := regexp.MustCompile(`Key:([^,}]+),Value:([^,}]+)`)
// 			matches := tagPattern.FindAllStringSubmatch(kv.Value, -1)

// 			for _, match := range matches {
// 				if len(match) >= 3 {
// 					key := strings.TrimSpace(match[1])
// 					value := strings.TrimSpace(match[2])

// 					// 빈 키나 값이 아닌 경우만 추가
// 					if key != "" && value != "" {
// 						tagList = append(tagList, irs.KeyValue{Key: key, Value: value})
// 						cblogger.Debugf("Extracted tag from KeyValueList: Key=%s, Value=%s", key, value)
// 					}
// 				}
// 			}

// 			// 정규식으로 파싱할 수 없는 경우 기존 로직 사용
// 			if len(tagList) == 0 {
// 				if strings.Contains(kv.Value, "Key:") && strings.Contains(kv.Value, "Value:") {
// 					// 간단한 파싱 로직
// 					keyStart := strings.Index(kv.Value, "Key:")
// 					valueStart := strings.Index(kv.Value, "Value:")

// 					if keyStart != -1 && valueStart != -1 {
// 						// Key 추출
// 						keyEnd := strings.Index(kv.Value[keyStart:], ",")
// 						if keyEnd == -1 {
// 							keyEnd = strings.Index(kv.Value[keyStart:], "}")
// 						}
// 						if keyEnd != -1 {
// 							key := strings.TrimSpace(kv.Value[keyStart+4 : keyStart+keyEnd])

// 							// Value 추출
// 							valueEnd := strings.Index(kv.Value[valueStart:], ",")
// 							if valueEnd == -1 {
// 								valueEnd = strings.Index(kv.Value[valueStart:], "}")
// 							}
// 							if valueEnd != -1 {
// 								value := strings.TrimSpace(kv.Value[valueStart+6 : valueStart+valueEnd])

// 								if key != "" && value != "" {
// 									tagList = append(tagList, irs.KeyValue{Key: key, Value: value})
// 									cblogger.Debugf("Extracted tag from KeyValueList (fallback): Key=%s, Value=%s", key, value)
// 								}
// 							}
// 						}
// 					}
// 				}
// 			}
// 		}
// 	}

// 	cblogger.Debugf("Total extracted tags from KeyValueList: %d", len(tagList))
// 	return tagList
// }

// DeleteFileSystem deletes the specified file system
func (fileSystemHandler *AlibabaFileSystemHandler) DeleteFileSystem(iid irs.IID) (bool, error) {
	cblogger.Debug("Alibaba Cloud NAS DeleteFileSystem() called")

	if fileSystemHandler.Client == nil {
		return false, errors.New("NAS client is not initialized")
	}

	// Validate file system IID
	if err := fileSystemHandler.validateFileSystemIID(iid); err != nil {
		return false, err
	}

	// ================================
	// Step 1: Delete all mount targets
	// ================================
	// Note: Tags will be automatically deleted when the file system is deleted
	mountTargets, err := fileSystemHandler.listMountTargets(iid.SystemId)
	if err != nil {
		cblogger.Errorf("Failed to list mount targets: %v", err)
		return false, err
	}

	cblogger.Infof("Found %d mount targets to delete", len(mountTargets))
	for _, mt := range mountTargets {
		cblogger.Infof("Deleting mount target: %s", mt.MountTargetDomain)
		err := fileSystemHandler.deleteMountTarget(iid.SystemId, mt.MountTargetDomain)
		if err != nil {
			cblogger.Errorf("Failed to delete mount target %s: %v", mt.MountTargetDomain, err)
			// Continue with other mount targets even if one fails
		} else {
			cblogger.Infof("Successfully deleted mount target: %s", mt.MountTargetDomain)
		}
	}

	// Wait for mount targets to be deleted
	if len(mountTargets) > 0 {
		cblogger.Info("Waiting for mount targets to be deleted...")
		time.Sleep(30 * time.Second)
	}

	// ================================
	// Step 2: Delete the file system
	// ================================
	cblogger.Infof("Deleting file system: %s", iid.SystemId)
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

	cblogger.Infof("Successfully deleted file system: %s", iid.SystemId)
	return true, nil
}

// AddAccessSubnet adds a subnet to the file system access list
func (fileSystemHandler *AlibabaFileSystemHandler) AddAccessSubnet(iid irs.IID, subnetIID irs.IID) (irs.FileSystemInfo, error) {
	cblogger.Debug("Alibaba Cloud NAS AddAccessSubnet() called")

	if fileSystemHandler.Client == nil {
		return irs.FileSystemInfo{}, errors.New("NAS client is not initialized")
	}

	// Validate file system IID
	if err := fileSystemHandler.validateFileSystemIID(iid); err != nil {
		return irs.FileSystemInfo{}, err
	}

	// Validate subnet IID
	if subnetIID.SystemId == "" {
		return irs.FileSystemInfo{}, errors.New("subnet SystemId is required")
	}

	// Validate that the file system exists before proceeding
	_, err := fileSystemHandler.GetFileSystem(iid)
	if err != nil {
		cblogger.Errorf("File system %s does not exist: %v", iid.SystemId, err)
		return irs.FileSystemInfo{}, fmt.Errorf("file system %s does not exist: %w", iid.SystemId, err)
	}

	err = fileSystemHandler.createMountTarget(iid, subnetIID)
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

	// Validate file system IID
	if err := fileSystemHandler.validateFileSystemIID(iid); err != nil {
		return false, err
	}

	// Validate subnet IID
	if subnetIID.SystemId == "" {
		return false, errors.New("subnet SystemId is required")
	}

	// Validate that the file system exists before proceeding
	_, err := fileSystemHandler.GetFileSystem(iid)
	if err != nil {
		cblogger.Errorf("File system %s does not exist: %v", iid.SystemId, err)
		return false, fmt.Errorf("file system %s does not exist: %w", iid.SystemId, err)
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
			err := fileSystemHandler.deleteMountTarget(iid.SystemId, mt.MountTargetDomain)
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

	// Validate file system IID
	if err := fileSystemHandler.validateFileSystemIID(iid); err != nil {
		return nil, err
	}

	// Validate that the file system exists before proceeding
	_, err := fileSystemHandler.GetFileSystem(iid)
	if err != nil {
		cblogger.Errorf("File system %s does not exist: %v", iid.SystemId, err)
		return nil, fmt.Errorf("file system %s does not exist: %w", iid.SystemId, err)
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

// validateFileSystemIID validates that the file system IID is valid
func (fileSystemHandler *AlibabaFileSystemHandler) validateFileSystemIID(iid irs.IID) error {
	if iid.SystemId == "" {
		return errors.New("file system SystemId is required")
	}
	return nil
}

// getVpcIdFromTag gets VpcId from file system tags
func (fileSystemHandler *AlibabaFileSystemHandler) getVpcIdFromTag(fileSystemId string) (string, error) {
	if fileSystemHandler.TagHandler == nil {
		return "", errors.New("TagHandler is not initialized")
	}

	// Get VpcId tag from file system
	vpcIdKeyValue, err := fileSystemHandler.TagHandler.GetTag(irs.FILESYSTEM, irs.IID{SystemId: fileSystemId}, "VpcId")
	if err != nil {
		return "", fmt.Errorf("failed to get VpcId tag: %w", err)
	}

	if vpcIdKeyValue.Value == "" {
		return "", errors.New("VpcId tag not found or empty")
	}

	return vpcIdKeyValue.Value, nil
}

// getAccessGroupFromMountTarget gets Access Group information for a mount target
func (fileSystemHandler *AlibabaFileSystemHandler) getAccessGroupFromMountTarget(mountTargetDomain string) (string, error) {
	if fileSystemHandler.Client == nil {
		return "", errors.New("NAS client is not initialized")
	}

	// Describe mount target to get access group information
	request := nas.CreateDescribeMountTargetsRequest()
	request.Scheme = "https"
	request.MountTargetDomain = mountTargetDomain

	response, err := fileSystemHandler.Client.DescribeMountTargets(request)
	if err != nil {
		return "", fmt.Errorf("failed to describe mount target %s: %w", mountTargetDomain, err)
	}

	if len(response.MountTargets.MountTarget) > 0 {
		mt := response.MountTargets.MountTarget[0]
		return mt.AccessGroup, nil
	}

	return "", errors.New("mount target not found")
}

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

func (fileSystemHandler *AlibabaFileSystemHandler) deleteMountTarget(fileSystemId string, mountTargetDomain string) error {
	if fileSystemHandler.Client == nil {
		return errors.New("NAS client is not initialized")
	}

	request := nas.CreateDeleteMountTargetRequest()
	request.Scheme = "https"
	request.FileSystemId = fileSystemId
	request.MountTargetDomain = mountTargetDomain

	_, err := fileSystemHandler.Client.DeleteMountTarget(request)
	return err
}

func (fileSystemHandler *AlibabaFileSystemHandler) createMountTarget(iid irs.IID, subnetIID irs.IID) error {
	return fileSystemHandler.createMountTargetWithVPC(iid, subnetIID, "")
}

func (fileSystemHandler *AlibabaFileSystemHandler) createMountTargetWithVPC(iid irs.IID, subnetIID irs.IID, vpcId string) error {
	cblogger.Debug("Alibaba Cloud NAS createMountTarget() called")

	if fileSystemHandler.Client == nil {
		return errors.New("NAS client is not initialized")
	}

	// Get or create access group
	accessGroupName, err := fileSystemHandler.getOrCreateAccessGroup(iid.SystemId)
	if err != nil {
		cblogger.Errorf("Failed to get or create access group: %v", err)
		return err
	}

	// If VPC ID is not provided, try to get it from file system tags first, then from subnet
	if vpcId == "" {
		// Try to get VPC ID from file system tags
		tagVpcId, tagErr := fileSystemHandler.getVpcIdFromTag(iid.SystemId)
		if tagErr == nil && tagVpcId != "" {
			vpcId = tagVpcId
			cblogger.Infof("Using VpcId from file system tags: %s", vpcId)
		} else {
			cblogger.Infof("VpcId not found in tags, trying subnet: %v", tagErr)
			// Fallback to subnet-based VPC ID detection (commented out for now)
			// vpcId, err = fileSystemHandler.getVPCFromSubnet(subnetIID.SystemId)
			// if err != nil {
			// 	cblogger.Errorf("Failed to get VPC ID from subnet %s: %v", subnetIID.SystemId, err)
			// 	// Continue without VPC ID - let Alibaba Cloud handle it
			// 	cblogger.Info("Continuing without VPC ID - Alibaba Cloud will determine it automatically")
			// }
		}
	}

	request := nas.CreateCreateMountTargetRequest()
	request.Scheme = "https"
	request.FileSystemId = iid.SystemId
	request.VSwitchId = subnetIID.SystemId
	request.NetworkType = "Vpc" // Alibaba Cloud NAS requires NetworkType to be set to "Vpc"
	request.AccessGroupName = accessGroupName

	// Only set VpcId if we have it
	if vpcId != "" {
		request.VpcId = vpcId
		cblogger.Infof("Creating mount target with AccessGroupName: %s, VpcId: %s", accessGroupName, vpcId)
	} else {
		cblogger.Infof("Creating mount target with AccessGroupName: %s (VpcId will be auto-detected)", accessGroupName)
	}

	hiscallInfo := GetCallLogScheme(fileSystemHandler.Region, call.FILESYSTEM, iid.SystemId, "CreateMountTarget()")
	start := call.Start()

	_, err = fileSystemHandler.Client.CreateMountTarget(request)

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

// getDefaultResourceGroupId gets the default resource group ID
func (fileSystemHandler *AlibabaFileSystemHandler) getDefaultResourceGroupId() (string, error) {
	// For now, we'll use a simple approach
	// In a production environment, you should use Alibaba Cloud Resource Manager API

	// Alibaba Cloud typically has a default resource group with ID like "rg-acfnvol7oa3usoa"
	// Since we don't have direct access to Resource Manager API in this handler,
	// we'll return an empty string and let Alibaba Cloud handle it automatically

	cblogger.Info("Attempting to get default resource group ID")

	// TODO: Implement proper Resource Manager API call
	// This would require:
	// 1. Resource Manager client initialization
	// 2. ListResourceGroups API call
	// 3. Find the default resource group

	// For now, return empty string - Alibaba Cloud will use default
	return "", nil
}

// getVPCFromSubnet gets VPC ID from subnet ID using VPC API
func (fileSystemHandler *AlibabaFileSystemHandler) getVPCFromSubnet(subnetId string) (string, error) {
	// For now, we'll use a simple approach - extract VPC ID from the subnet ID pattern
	// In a production environment, you might want to use the VPC API to get this information

	// Alibaba Cloud subnet IDs typically follow the pattern: vsw-xxxxxxxxx
	// VPC IDs typically follow the pattern: vpc-xxxxxxxxx
	// Since we don't have direct VPC API access in this handler, we'll use a fallback approach

	// Try to get VPC ID from the file system's VPC ID if available
	// This is a simplified approach - in production, you should use proper VPC API calls

	// For now, return an empty string and let the API handle it
	// The VPC ID might be automatically determined by Alibaba Cloud based on the subnet
	cblogger.Infof("Attempting to get VPC ID for subnet: %s", subnetId)

	// If we have access to VPC client, we could use it here
	// For now, we'll let the API handle VPC ID determination
	return "", nil
}

func (fileSystemHandler *AlibabaFileSystemHandler) getOrCreateAccessGroup(fileSystemId string) (string, error) {
	// First, try to list existing access groups
	request := nas.CreateDescribeAccessGroupsRequest()
	request.Scheme = "https"
	request.FileSystemType = "standard" // Alibaba NAS standard file system type

	response, err := fileSystemHandler.Client.DescribeAccessGroups(request)
	if err != nil {
		cblogger.Warnf("Failed to list access groups: %v", err)
		// If listing fails, use a default access group name
		defaultAccessGroupName := fmt.Sprintf("cb-spider-default-%s", fileSystemId)
		cblogger.Infof("Using default access group name: %s", defaultAccessGroupName)
		return defaultAccessGroupName, nil
	}

	// Check if there are existing access groups
	if len(response.AccessGroups.AccessGroup) > 0 {
		// Use the first available access group
		accessGroupName := response.AccessGroups.AccessGroup[0].AccessGroupName
		cblogger.Infof("Using existing access group: %s", accessGroupName)
		return accessGroupName, nil
	}

	// No existing access groups, create a default one
	defaultAccessGroupName := fmt.Sprintf("cb-spider-default-%s", fileSystemId)
	cblogger.Infof("No existing access groups found, creating default: %s", defaultAccessGroupName)

	createRequest := nas.CreateCreateAccessGroupRequest()
	createRequest.Scheme = "https"
	createRequest.AccessGroupName = defaultAccessGroupName
	createRequest.AccessGroupType = "Vpc"
	createRequest.Description = "Default access group created by CB-Spider"

	_, err = fileSystemHandler.Client.CreateAccessGroup(createRequest)
	if err != nil {
		cblogger.Errorf("Failed to create access group: %v", err)
		// Access Group creation is critical for mount target creation
		return "", fmt.Errorf("failed to create access group %s: %v", defaultAccessGroupName, err)
	}

	cblogger.Infof("Successfully created access group: %s", defaultAccessGroupName)
	return defaultAccessGroupName, nil
}

func (fileSystemHandler *AlibabaFileSystemHandler) convertToFileSystemInfo(fs *nas.FileSystem) (irs.FileSystemInfo, error) {
	if fs == nil {
		return irs.FileSystemInfo{}, errors.New("file system is nil")
	}

	// Get mount targets
	mountTargets, err := fileSystemHandler.listMountTargets(fs.FileSystemId)
	if err != nil {
		cblogger.Errorf("Failed to get mount targets: %v", err)
		// Mount targets are critical information - fail the operation
		return irs.FileSystemInfo{}, fmt.Errorf("failed to get mount targets for file system %s: %v", fs.FileSystemId, err)
	}

	var mountTargetList []irs.MountTargetInfo
	var accessSubnetList []irs.IID
	for _, mt := range mountTargets {
		mountTargetInfo := irs.MountTargetInfo{
			SubnetIID: irs.IID{SystemId: mt.VswId},
			Endpoint:  mt.MountTargetDomain,
		}

		// Get Access Group information (Alibaba Cloud NAS uses Access Group instead of Security Groups)
		if mt.AccessGroup != "" {
			// Map Access Group to SecurityGroups field for compatibility
			mountTargetInfo.SecurityGroups = []string{mt.AccessGroup}
		} else {
			// Try to get Access Group from mount target details
			accessGroup, err := fileSystemHandler.getAccessGroupFromMountTarget(mt.MountTargetDomain)
			if err == nil && accessGroup != "" {
				mountTargetInfo.SecurityGroups = []string{accessGroup}
			} else {
				// Set empty slice instead of nil for consistency
				mountTargetInfo.SecurityGroups = []string{}
			}
		}

		// Create mount command example
		if fs.ProtocolType == "NFS" {
			mountTargetInfo.MountCommandExample = fmt.Sprintf("sudo mount -t nfs %s:/ /mnt/nas", mt.MountTargetDomain)
		} else if fs.ProtocolType == "SMB" {
			mountTargetInfo.MountCommandExample = fmt.Sprintf("sudo mount -t cifs //%s /mnt/nas -o username=your_username,password=your_password", mt.MountTargetDomain)
		}

		// Add complete Alibaba NAS mount target object information
		mountTargetInfo.KeyValueList = irs.StructToKeyValueList(mt)

		// Add Access Group information to KeyValueList for clarity
		if len(mountTargetInfo.SecurityGroups) > 0 {
			mountTargetInfo.KeyValueList = append(mountTargetInfo.KeyValueList, irs.KeyValue{
				Key:   "AccessGroup",
				Value: mountTargetInfo.SecurityGroups[0],
			})
		}

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

	// Mount target에서 VPC ID 추출 (fs.VpcId가 비어있는 경우)
	vpcId := fs.VpcId
	if vpcId == "" {
		// Try to get VPC ID from file system tags first
		tagVpcId, tagErr := fileSystemHandler.getVpcIdFromTag(fs.FileSystemId)
		if tagErr == nil && tagVpcId != "" {
			vpcId = tagVpcId
			cblogger.Infof("Using VpcId from file system tags: %s", vpcId)
		} else {
			// Fallback to mount target VpcId (commented out for now)
			// if len(mountTargets) > 0 {
			// 	// Mount target의 VpcId 필드에서 직접 추출
			// 	for _, mt := range mountTargets {
			// 		if mt.VpcId != "" {
			// 			vpcId = mt.VpcId
			// 			break
			// 		}
			// 	}
			// }
			cblogger.Infof("VpcId not found in tags or mount targets: %v", tagErr)
		}
	}

	// 생성 시간 파싱 (KeyValueList에서 CreateTime 찾기)
	createdTime := time.Now()
	for _, kv := range keyValueList {
		if kv.Key == "CreateTime" && kv.Value != "" {
			// "2025-08-20T18:42:44CST" 형식을 파싱
			if parsedTime, err := time.Parse("2006-01-02T15:04:05MST", kv.Value); err == nil {
				createdTime = parsedTime
			}
			break
		}
	}

	fileSystemInfo := irs.FileSystemInfo{
		IId: irs.IID{
			NameId:   fs.Description,
			SystemId: fs.FileSystemId,
		},
		Region:           fileSystemHandler.Region.Region,
		Zone:             fs.ZoneId,
		VpcIID:           irs.IID{SystemId: vpcId},
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
		CreatedTime:      createdTime,
		KeyValueList:     keyValueList,
	}

	// Fallback: If no tags found in fs object, try TagHandler API
	if fileSystemHandler.TagHandler != nil {
		tagList, err := fileSystemHandler.TagHandler.ListTag(irs.FILESYSTEM, fileSystemInfo.IId)
		if err != nil {
			// Tag retrieval is non-critical - log warning and continue
			cblogger.Warnf("Failed to get tags via TagHandler (non-critical): %v", err)
			cblogger.Debug("extracting tags from FileSystem")
			// Extract tags directly from the original fs object
			fileSystemInfo.TagList = extractTagsFromFileSystem(fs)
		} else {
			fileSystemInfo.TagList = tagList
		}
	} else {
		cblogger.Warnf("TagHandler is nil, extracting tags from FileSystem")
		// Extract tags directly from the original fs object
		fileSystemInfo.TagList = extractTagsFromFileSystem(fs)
	}

	return fileSystemInfo, nil
}
