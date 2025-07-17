package resources

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/efs"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsFileSystemHandler struct {
	Region    idrv.RegionInfo
	Client    *efs.EFS
	EC2Client *ec2.EC2
}

// GetMetaInfo returns metadata about the file system capabilities
func (fileSystemHandler *AwsFileSystemHandler) GetMetaInfo() (irs.FileSystemMetaInfo, error) {
	cblogger.Debug("AWS EFS GetMetaInfo() called")

	hiscallInfo := GetCallLogScheme(fileSystemHandler.Region, call.FILESYSTEM, "GetMetaInfo", "GetMetaInfo()")
	start := call.Start()

	metaInfo := irs.FileSystemMetaInfo{
		SupportsFileSystemType: map[irs.FileSystemType]bool{
			irs.RegionType:          true,  // AWS EFS Regional (Multi-AZ)
			irs.ZoneType:            true,  // AWS EFS One Zone
			irs.RegionVPCBasedType:  false, // AWS EFS Regional (Multi-AZ)
			irs.RegionZoneBasedType: true,  // AWS EFS One Zone - Zone based
		},
		SupportsVPC: map[irs.RSType]bool{
			irs.VPC: true,
		},
		SupportsNFSVersion: []string{"4.0", "4.1"},           // AWS EFS supports NFS 4.0 and 4.1
		SupportsCapacity:   false,                            // AWS EFS uses elastic scaling
		CapacityGBOptions:  map[string]irs.CapacityGBRange{}, // Empty map since AWS EFS doesn't support capacity specification
		PerformanceOptions: map[string][]string{
			"RequiredFields": {
				"ThroughputMode",  // Required field: ThroughputMode must be specified
				"PerformanceMode", // Required field: PerformanceMode must be specified
			},
			"OptionalFields": {
				"ProvisionedThroughput", // Optional: Required only for Provisioned mode
			},
			"ProvisionedThroughput": {
				"Min:1",    // Minimum provisioned throughput (MiB/s)
				"Max:1024", // Maximum provisioned throughput (MiB/s)
			},
			"ThroughputMode": {
				"Elastic",     // AWS EFS Elastic throughput mode (recommended)
				"Bursting",    // AWS EFS Bursting throughput mode (API name)
				"Provisioned", // AWS EFS provisioned mode
			},
			"PerformanceMode": {
				"GeneralPurpose", // General Purpose (recommended)
				"MaxIO",          // Max I/O (One Zone excluded)
			},
			"Constraints": {
				"Elastic:MaxIO:NotSupported",                 // Max I/O not supported with Elastic
				"OneZone:MaxIO:NotSupported",                 // Max I/O not supported for One Zone
				"Provisioned:ProvisionedThroughput:Required", // ProvisionedThroughput required for Provisioned mode
			},
			"Examples": {
				"Elastic+GeneralPurpose:ThroughputMode:Elastic,PerformanceMode:GeneralPurpose",
				"Bursting+MaxIO:ThroughputMode:Bursting,PerformanceMode:MaxIO",
				"Provisioned+GeneralPurpose:ThroughputMode:Provisioned,PerformanceMode:GeneralPurpose,ProvisionedThroughput:128",
				"Provisioned+MaxIO:ThroughputMode:Provisioned,PerformanceMode:MaxIO,ProvisionedThroughput:256",
				"OneZone+Provisioned:ThroughputMode:Provisioned,PerformanceMode:GeneralPurpose,ProvisionedThroughput:64",
			},
		},
	}

	LoggingInfo(hiscallInfo, start)
	return metaInfo, nil
}

// ListIID returns list of file system IDs
func (fileSystemHandler *AwsFileSystemHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Debug("AWS EFS ListIID() called")

	input := &efs.DescribeFileSystemsInput{}

	hiscallInfo := GetCallLogScheme(fileSystemHandler.Region, call.FILESYSTEM, "ListIID", "DescribeFileSystems()")
	start := call.Start()

	result, err := fileSystemHandler.Client.DescribeFileSystems(input)

	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	var iidList []*irs.IID
	for _, fs := range result.FileSystems {
		if fs == nil {
			continue
		}

		var nameId, systemId string

		// Safely handle Name
		if fs.Name != nil {
			nameId = *fs.Name
		}

		// Safely handle FileSystemId
		if fs.FileSystemId != nil {
			systemId = *fs.FileSystemId
		}

		iid := irs.IID{
			NameId:   nameId,
			SystemId: systemId,
		}
		iidList = append(iidList, &iid)
	}

	return iidList, nil
}

// CreateFileSystem creates a new EFS file system
func (fileSystemHandler *AwsFileSystemHandler) CreateFileSystem(reqInfo irs.FileSystemInfo) (irs.FileSystemInfo, error) {
	cblogger.Debug("AWS EFS CreateFileSystem() called")

	// ================================
	// Validate NFS version if provided
	// ================================
	if reqInfo.NFSVersion != "" {
		// Get supported NFS versions from GetMetaInfo to ensure consistency
		metaInfo, err := fileSystemHandler.GetMetaInfo()
		if err != nil {
			cblogger.Errorf("Failed to get meta info for NFS version validation: %v", err)
			return irs.FileSystemInfo{}, fmt.Errorf("failed to validate NFS version: %v", err)
		}

		// Check if the requested NFS version is supported
		supported := false
		for _, supportedVersion := range metaInfo.SupportsNFSVersion {
			if reqInfo.NFSVersion == supportedVersion {
				supported = true
				break
			}
		}

		if !supported {
			return irs.FileSystemInfo{}, fmt.Errorf("AWS EFS only supports NFS versions: %v", metaInfo.SupportsNFSVersion)
		}
		cblogger.Infof("Requested NFS version: %s (AWS EFS will use 4.1 for file system, but %s can be used for mounting)", reqInfo.NFSVersion, reqInfo.NFSVersion)
	} else {
		reqInfo.NFSVersion = "4.1" // AWS EFS default
		cblogger.Info("Using default NFS version: 4.1")
	}

	// =====================================
	// Validate VPC requirement for AWS EFS
	// =====================================
	if reqInfo.VpcIID.SystemId == "" {
		return irs.FileSystemInfo{}, errors.New("VPC is required for AWS EFS file system creation")
	}

	input := &efs.CreateFileSystemInput{
		CreationToken: aws.String(reqInfo.IId.NameId),
	}

	// ==============
	// Prepare tags
	// ==============
	var tags []*efs.Tag
	if reqInfo.IId.NameId != "" {
		// Add default Name tag
		tags = append(tags, &efs.Tag{
			Key:   aws.String("Name"),
			Value: aws.String(reqInfo.IId.NameId),
		})
	}

	// Add user-provided tags
	if reqInfo.TagList != nil {
		for _, tag := range reqInfo.TagList {
			tags = append(tags, &efs.Tag{
				Key:   aws.String(tag.Key),
				Value: aws.String(tag.Value),
			})
		}
	}

	// Only set Tags if there are tags to add
	if len(tags) > 0 {
		input.Tags = tags
	}

	// ================================================
	// Check if the user is using basic setup mode
	// ================================================
	isDefaultLifecyclePolicy := true
	if reqInfo.FileSystemType == "" {
		reqInfo.FileSystemType = irs.RegionType // AWS EFS default
		reqInfo.Encryption = true               // AWS EFS requires encryption by default

		input.Backup = aws.Bool(true) // automatic backup enabled by default
		// Performance mode and throughput mode will use AWS EFS defaults
		// - Performance Mode: generalPurpose (default)
		// - Throughput Mode: bursting (default)
		isDefaultLifecyclePolicy = true
	} else {
		// =======================================================================
		// Set backup policy
		// =======================================================================
		// [TODO] Since reqInfo.Backup(bool) field doesn't exist, we need to decide whether to set Backup parameter to true only when BackupSchedule is configured or always set it as default
		input.Backup = aws.Bool(true) // automatic backup enabled by default
	}

	// =======================================================================
	// Determine and set performance mode and throughput mode if specified
	// =======================================================================
	if reqInfo.PerformanceInfo != nil {
		isDefaultLifecyclePolicy = false

		// Get meta info to validate performance options
		metaInfo, err := fileSystemHandler.GetMetaInfo()
		if err != nil {
			cblogger.Errorf("Failed to get meta info for performance validation: %v", err)
			return irs.FileSystemInfo{}, fmt.Errorf("failed to validate performance options: %v", err)
		}

		// Get valid options from meta info
		validOptions := metaInfo.PerformanceOptions

		// Validate required fields
		requiredFields := validOptions["RequiredFields"]
		for _, field := range requiredFields {
			if _, exists := reqInfo.PerformanceInfo[field]; !exists {
				return irs.FileSystemInfo{}, fmt.Errorf("required field '%s' is missing in PerformanceInfo", field)
			}
		}

		// Validate throughput mode
		throughputMode := reqInfo.PerformanceInfo["ThroughputMode"]
		validThroughputModes := validOptions["ThroughputMode"]
		throughputModeValid := false
		for _, validMode := range validThroughputModes {
			if validMode == throughputMode {
				throughputModeValid = true
				break
			}
		}
		if !throughputModeValid {
			return irs.FileSystemInfo{}, fmt.Errorf("invalid ThroughputMode '%s'. Valid options: %v", throughputMode, validThroughputModes)
		}

		// Validate performance mode
		performanceMode := reqInfo.PerformanceInfo["PerformanceMode"]
		validPerformanceModes := validOptions["PerformanceMode"]
		performanceModeValid := false
		for _, validMode := range validPerformanceModes {
			if validMode == performanceMode {
				performanceModeValid = true
				break
			}
		}
		if !performanceModeValid {
			return irs.FileSystemInfo{}, fmt.Errorf("invalid PerformanceMode '%s'. Valid options: %v", performanceMode, validPerformanceModes)
		}

		// Check constraints
		constraints := validOptions["Constraints"]
		for _, constraint := range constraints {
			parts := strings.Split(constraint, ":")
			if len(parts) >= 3 {
				switch parts[0] {
				case "Elastic":
					if parts[1] == "MaxIO" && parts[2] == "NotSupported" && throughputMode == "Elastic" && performanceMode == "MaxIO" {
						return irs.FileSystemInfo{}, fmt.Errorf("MaxIO performance mode is not supported with Elastic throughput mode")
					}
				case "OneZone":
					if parts[1] == "MaxIO" && parts[2] == "NotSupported" && reqInfo.FileSystemType == irs.ZoneType && performanceMode == "MaxIO" {
						cblogger.Warn("MaxIO performance mode is not supported for One Zone EFS, using GeneralPurpose")
						performanceMode = "GeneralPurpose"
					}
				case "Provisioned":
					if parts[1] == "ProvisionedThroughput" && parts[2] == "Required" && throughputMode == "Provisioned" {
						if _, exists := reqInfo.PerformanceInfo["ProvisionedThroughput"]; !exists {
							return irs.FileSystemInfo{}, fmt.Errorf("ProvisionedThroughput is required when ThroughputMode is Provisioned")
						}
					}
				}
			}
		}

		// Validate provisioned throughput if specified
		if throughputMode == "Provisioned" {
			if throughput, exists := reqInfo.PerformanceInfo["ProvisionedThroughput"]; exists {
				if throughputFloat, err := strconv.ParseFloat(throughput, 64); err == nil {
					// Get throughput range from meta info
					throughputRange := validOptions["ProvisionedThroughput"]
					var minThroughput, maxThroughput float64
					for _, rangeOption := range throughputRange {
						if strings.HasPrefix(rangeOption, "Min:") {
							minStr := strings.TrimPrefix(rangeOption, "Min:")
							if min, err := strconv.ParseFloat(minStr, 64); err == nil {
								minThroughput = min
							}
						} else if strings.HasPrefix(rangeOption, "Max:") {
							maxStr := strings.TrimPrefix(rangeOption, "Max:")
							if max, err := strconv.ParseFloat(maxStr, 64); err == nil {
								maxThroughput = max
							}
						}
					}

					if throughputFloat >= minThroughput && throughputFloat <= maxThroughput {
						input.ProvisionedThroughputInMibps = &throughputFloat
					} else {
						return irs.FileSystemInfo{}, fmt.Errorf("provisioned throughput must be between %.0f and %.0f MiB/s", minThroughput, maxThroughput)
					}
				} else {
					return irs.FileSystemInfo{}, errors.New("invalid provisioned throughput value")
				}
			} else {
				return irs.FileSystemInfo{}, errors.New("provisioned throughput value is required when ThroughputMode is Provisioned")
			}
		}

		// Set AWS API parameters based on validated values
		switch throughputMode {
		case "Elastic", "Bursting":
			// Both Elastic and Bursting map to AWS EFS bursting mode
			input.ThroughputMode = aws.String(efs.ThroughputModeBursting)
		case "Provisioned":
			input.ThroughputMode = aws.String(efs.ThroughputModeProvisioned)
		}

		// Set performance mode
		switch performanceMode {
		case "GeneralPurpose":
			input.PerformanceMode = aws.String(efs.PerformanceModeGeneralPurpose)
		case "MaxIO":
			input.PerformanceMode = aws.String(efs.PerformanceModeMaxIo)
		}

		cblogger.Infof("Performance settings: ThroughputMode=%s, PerformanceMode=%s", throughputMode, performanceMode)
	}

	// =======================================================================
	// Handle encryption according to user preference
	// AWS EFS allows users to choose encryption settings
	// =======================================================================
	if reqInfo.Encryption {
		input.Encrypted = aws.Bool(true)
		cblogger.Info("User requested encryption - enabling with default AWS EFS KMS key")
	} else {
		cblogger.Info("User requested no encryption - creating unencrypted file system")
	}

	// =======================================================================
	// Process ZoneType and determine target zone
	// =======================================================================
	// AWS EFS is created in the region where the EFS client is configured
	targetRegion := fileSystemHandler.Region.Region
	cblogger.Infof("Creating EFS in region: %s", targetRegion)

	// Determine target zone for One Zone EFS
	var targetZoneId string
	if reqInfo.FileSystemType == irs.ZoneType {
		cblogger.Info("Creating One Zone EFS as specified by user")
		// For One Zone EFS, we need to specify the availability zone
		targetZoneId = reqInfo.Zone
		if targetZoneId == "" {
			// If no zone specified, use the handler's zone or get available zones
			targetZoneId = fileSystemHandler.Region.Zone
			if targetZoneId == "" {
				// Get available zones from the current region
				availableZones, err := fileSystemHandler.getAvailableZonesInRegion(targetRegion)
				if err != nil {
					cblogger.Errorf("Failed to get available zones for region %s: %v", targetRegion, err)
					return irs.FileSystemInfo{}, fmt.Errorf("failed to get available zones for region %s: %v", targetRegion, err)
				}
				if len(availableZones) > 0 {
					targetZoneId = availableZones[0] // Use the first available zone
					cblogger.Infof("Auto-selected zone: %s in region: %s", targetZoneId, targetRegion)
				} else {
					return irs.FileSystemInfo{}, fmt.Errorf("no available zones found in region %s", targetRegion)
				}
			}
		}

		// For One Zone EFS, specify the availability zone name
		// AWS EFS One Zone uses AvailabilityZoneName parameter (e.g., us-east-1a)
		input.AvailabilityZoneName = aws.String(targetZoneId)
		cblogger.Infof("Creating One Zone EFS in zone: %s (region: %s)", targetZoneId, targetRegion)
	} else {
		// Default behavior: Create Regional EFS (Multi-AZ)
		cblogger.Infof("Creating Regional EFS (Multi-AZ) in region: %s - AWS EFS default", targetRegion)
	}

	hiscallInfo := GetCallLogScheme(fileSystemHandler.Region, call.FILESYSTEM, reqInfo.IId.NameId, "CreateFileSystem()")
	start := call.Start()

	result, err := fileSystemHandler.Client.CreateFileSystem(input)

	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	// Check if result is valid
	if result == nil || result.FileSystemId == nil {
		return irs.FileSystemInfo{}, errors.New("failed to create file system: invalid response")
	}

	// Wait for file system to be available
	err = fileSystemHandler.waitUntilFileSystemAvailable(*result.FileSystemId)
	if err != nil {
		cblogger.Error(err)
		return irs.FileSystemInfo{}, err
	}

	// Set lifecycle management policy according to AWS Console defaults
	if isDefaultLifecyclePolicy {
		err = fileSystemHandler.setDefaultLifecyclePolicy(*result.FileSystemId)
		if err != nil {
			cblogger.Errorf("Failed to set lifecycle policy: %v", err)
			// Don't fail the entire operation, just log the error
		}
	}

	// =======================================================================
	// Create mount targets
	// =======================================================================

	// Determine mount target creation strategy based on user input
	if len(reqInfo.MountTargetList) > 0 {
		// User provided MountTargetList with security groups
		cblogger.Info("Creating mount targets using MountTargetList with security groups")

		// Validate One Zone EFS constraint: only 1 mount target allowed
		if reqInfo.FileSystemType == irs.ZoneType && len(reqInfo.MountTargetList) > 1 {
			return irs.FileSystemInfo{}, fmt.Errorf("One Zone EFS can only have 1 mount target, but %d were specified", len(reqInfo.MountTargetList))
		}

		// Validate that all subnets are in the correct zone for One Zone EFS
		if reqInfo.FileSystemType == irs.ZoneType && targetZoneId != "" {
			cblogger.Infof("Validating that all mount targets are in zone: %s for One Zone EFS", targetZoneId)
			for _, mtInfo := range reqInfo.MountTargetList {
				subnetZone, err := fileSystemHandler.getSubnetZone(mtInfo.SubnetIID.SystemId)
				if err != nil {
					cblogger.Errorf("Failed to get zone for subnet %s: %v", mtInfo.SubnetIID.SystemId, err)
					continue
				}
				if subnetZone != targetZoneId {
					cblogger.Errorf("Mount target subnet %s is in zone %s, but One Zone EFS is in zone %s",
						mtInfo.SubnetIID.SystemId, subnetZone, targetZoneId)
					return irs.FileSystemInfo{}, fmt.Errorf("mount target subnet %s is not in the correct zone for One Zone EFS", mtInfo.SubnetIID.SystemId)
				}
			}
		}

		// Create mount targets with specified security groups
		for _, mtInfo := range reqInfo.MountTargetList {
			err := fileSystemHandler.createMountTargetWithSecurityGroups(
				irs.IID{NameId: reqInfo.IId.NameId, SystemId: *result.FileSystemId},
				mtInfo.SubnetIID,
				mtInfo.SecurityGroups,
			)
			if err != nil {
				cblogger.Errorf("Failed to create mount target for subnet %s: %v", mtInfo.SubnetIID.SystemId, err)
			}
		}

	} else if len(reqInfo.AccessSubnetList) > 0 {
		// User provided AccessSubnetList only (default security groups will be used)
		cblogger.Info("Creating mount targets using AccessSubnetList with default security groups")

		// Validate One Zone EFS constraint: only 1 subnet allowed
		if reqInfo.FileSystemType == irs.ZoneType && len(reqInfo.AccessSubnetList) > 1 {
			return irs.FileSystemInfo{}, fmt.Errorf("One Zone EFS can only have 1 mount target, but %d subnets were specified", len(reqInfo.AccessSubnetList))
		}

		// Validate that all subnets are in the correct zone for One Zone EFS
		if reqInfo.FileSystemType == irs.ZoneType && targetZoneId != "" {
			cblogger.Infof("Validating that all subnets are in zone: %s for One Zone EFS", targetZoneId)
			for _, subnetIID := range reqInfo.AccessSubnetList {
				subnetZone, err := fileSystemHandler.getSubnetZone(subnetIID.SystemId)
				if err != nil {
					cblogger.Errorf("Failed to get zone for subnet %s: %v", subnetIID.SystemId, err)
					continue
				}
				if subnetZone != targetZoneId {
					cblogger.Errorf("Subnet %s is in zone %s, but One Zone EFS is in zone %s",
						subnetIID.SystemId, subnetZone, targetZoneId)
					return irs.FileSystemInfo{}, fmt.Errorf("subnet %s is not in the correct zone for One Zone EFS", subnetIID.SystemId)
				}
			}
		}

		// Create mount targets with default security groups
		for _, subnetIID := range reqInfo.AccessSubnetList {
			err := fileSystemHandler.createMountTargetWithSecurityGroups(
				irs.IID{NameId: reqInfo.IId.NameId, SystemId: *result.FileSystemId},
				subnetIID,
				nil, // nil means default security groups will be used
			)
			if err != nil {
				cblogger.Errorf("Failed to create mount target for subnet %s: %v", subnetIID.SystemId, err)
			}
		}

	} else {
		// No user specification - use AWS console default behavior
		cblogger.Info("No mount target specification provided - using AWS console default behavior")
		err = fileSystemHandler.createDefaultMountTargets(*result.FileSystemId, reqInfo, targetZoneId)
		if err != nil {
			cblogger.Errorf("Failed to create default mount targets: %v", err)
			// Don't fail the entire operation, just log the error
		}
	}

	// Get the created file system info
	fileSystemInfo, err := fileSystemHandler.GetFileSystem(irs.IID{NameId: reqInfo.IId.NameId, SystemId: *result.FileSystemId})
	if err != nil {
		return irs.FileSystemInfo{}, err
	}

	return fileSystemInfo, nil
}

// ListFileSystem returns list of all file systems
func (fileSystemHandler *AwsFileSystemHandler) ListFileSystem() ([]*irs.FileSystemInfo, error) {
	cblogger.Debug("AWS EFS ListFileSystem() called")

	// Measure total ListFileSystem operation time
	hiscallInfo := GetCallLogScheme(fileSystemHandler.Region, call.FILESYSTEM, "ListFileSystem", "ListFileSystem()")
	start := call.Start()

	// Get list of file system IIDs first
	iidList, err := fileSystemHandler.ListIID()
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	// Get detailed information for each file system using GetFileSystem
	var fileSystemList []*irs.FileSystemInfo
	for _, iid := range iidList {
		fileSystemInfo, err := fileSystemHandler.GetFileSystem(*iid)
		if err != nil {
			cblogger.Errorf("Failed to get file system info for %s: %v", iid.SystemId, err)
			continue
		}
		fileSystemList = append(fileSystemList, &fileSystemInfo)
	}

	LoggingInfo(hiscallInfo, start)
	return fileSystemList, nil
}

// GetFileSystem returns specific file system info
func (fileSystemHandler *AwsFileSystemHandler) GetFileSystem(iid irs.IID) (irs.FileSystemInfo, error) {
	cblogger.Debug("AWS EFS GetFileSystem() called")

	input := &efs.DescribeFileSystemsInput{
		FileSystemId: aws.String(iid.SystemId),
	}

	hiscallInfo := GetCallLogScheme(fileSystemHandler.Region, call.FILESYSTEM, iid.SystemId, "DescribeFileSystems()")
	start := call.Start()

	result, err := fileSystemHandler.Client.DescribeFileSystems(input)

	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return irs.FileSystemInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	if len(result.FileSystems) == 0 {
		return irs.FileSystemInfo{}, errors.New("file system not found")
	}

	return fileSystemHandler.convertToFileSystemInfo(result.FileSystems[0])
}

// DeleteFileSystem deletes the specified file system
func (fileSystemHandler *AwsFileSystemHandler) DeleteFileSystem(iid irs.IID) (bool, error) {
	cblogger.Debug("AWS EFS DeleteFileSystem() called")

	// First, delete all mount targets
	mountTargets, err := fileSystemHandler.listMountTargets(iid.SystemId)
	if err != nil {
		cblogger.Errorf("Failed to list mount targets: %v", err)
		return false, err
	}

	for _, mt := range mountTargets {
		err := fileSystemHandler.deleteMountTarget(*mt.MountTargetId)
		if err != nil {
			cblogger.Errorf("Failed to delete mount target %s: %v", *mt.MountTargetId, err)
		}
	}

	// Wait for mount targets to be deleted
	if len(mountTargets) > 0 {
		time.Sleep(30 * time.Second) // Wait for mount targets to be fully deleted
	}

	input := &efs.DeleteFileSystemInput{
		FileSystemId: aws.String(iid.SystemId),
	}

	hiscallInfo := GetCallLogScheme(fileSystemHandler.Region, call.FILESYSTEM, iid.SystemId, "DeleteFileSystem()")
	start := call.Start()

	_, err = fileSystemHandler.Client.DeleteFileSystem(input)

	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

// AddAccessSubnet adds a subnet to the file system access list
func (fileSystemHandler *AwsFileSystemHandler) AddAccessSubnet(iid irs.IID, subnetIID irs.IID) (irs.FileSystemInfo, error) {
	cblogger.Debug("AWS EFS AddAccessSubnet() called")

	// Create mount target
	input := &efs.CreateMountTargetInput{
		FileSystemId: aws.String(iid.SystemId),
		SubnetId:     aws.String(subnetIID.SystemId),
	}

	hiscallInfo := GetCallLogScheme(fileSystemHandler.Region, call.FILESYSTEM, iid.SystemId, "CreateMountTarget()")
	start := call.Start()

	result, err := fileSystemHandler.Client.CreateMountTarget(input)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "MountTargetConflict" {
				cblogger.Info("Mount target already exists for this subnet")
				LoggingInfo(hiscallInfo, start)
			} else {
				cblogger.Error(err)
				LoggingError(hiscallInfo, err)
				return irs.FileSystemInfo{}, err
			}
		} else {
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return irs.FileSystemInfo{}, err
		}
	} else {
		cblogger.Infof("Created mount target %s for file system %s in subnet %s",
			*result.MountTargetId, iid.SystemId, subnetIID.SystemId)
		LoggingInfo(hiscallInfo, start)
	}

	// Wait for mount target to be available
	time.Sleep(10 * time.Second)

	return fileSystemHandler.GetFileSystem(iid)
}

// RemoveAccessSubnet removes a subnet from the file system access list
func (fileSystemHandler *AwsFileSystemHandler) RemoveAccessSubnet(iid irs.IID, subnetIID irs.IID) (bool, error) {
	cblogger.Debug("AWS EFS RemoveAccessSubnet() called")

	hiscallInfo := GetCallLogScheme(fileSystemHandler.Region, call.FILESYSTEM, iid.SystemId, "RemoveAccessSubnet()")
	start := call.Start()

	// Find mount target for this subnet
	mountTargets, err := fileSystemHandler.listMountTargets(iid.SystemId)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	for _, mt := range mountTargets {
		if mt == nil {
			continue
		}

		if mt.SubnetId != nil && *mt.SubnetId == subnetIID.SystemId {
			if mt.MountTargetId != nil {
				err := fileSystemHandler.deleteMountTarget(*mt.MountTargetId)
				if err != nil {
					cblogger.Error(err)
					LoggingError(hiscallInfo, err)
					return false, err
				}
				LoggingInfo(hiscallInfo, start)
				return true, nil
			}
		}
	}

	LoggingError(hiscallInfo, errors.New("mount target not found for the specified subnet"))
	return false, errors.New("mount target not found for the specified subnet")
}

// ListAccessSubnet returns list of subnets that can access the file system
func (fileSystemHandler *AwsFileSystemHandler) ListAccessSubnet(iid irs.IID) ([]irs.IID, error) {
	cblogger.Debug("AWS EFS ListAccessSubnet() called")

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
		if mt == nil {
			continue
		}

		if mt.SubnetId != nil {
			subnetIID := irs.IID{
				SystemId: *mt.SubnetId,
			}
			subnetList = append(subnetList, subnetIID)
		}
	}

	LoggingInfo(hiscallInfo, start)
	return subnetList, nil
}

// ScheduleBackup creates a backup with the specified schedule
func (fileSystemHandler *AwsFileSystemHandler) ScheduleBackup(reqInfo irs.FileSystemBackupInfo) (irs.FileSystemBackupInfo, error) {
	cblogger.Debug("AWS EFS ScheduleBackup() called")

	// TODO: AWS EFS scheduled backup implementation
	// According to the specification document, backup functionality will be developed after basic FS create/delete functionality
	// This would need to be implemented using AWS Lambda + EventBridge for scheduled backups
	// For now, return an error indicating this feature is planned for future development
	return irs.FileSystemBackupInfo{}, errors.New("scheduled backups are planned for future development in AWS EFS driver")
}

// OnDemandBackup creates an on-demand backup
func (fileSystemHandler *AwsFileSystemHandler) OnDemandBackup(fsIID irs.IID) (irs.FileSystemBackupInfo, error) {
	cblogger.Debug("AWS EFS OnDemandBackup() called")

	// TODO: AWS EFS on-demand backup implementation
	// According to the specification document, backup functionality will be developed after basic FS create/delete functionality
	// AWS EFS doesn't support traditional backups or snapshots, but could use EFS-to-EFS replication or external backup solutions
	// For now, return an error indicating this feature is planned for future development
	return irs.FileSystemBackupInfo{}, errors.New("on-demand backups are planned for future development in AWS EFS driver")
}

// ListBackup returns list of backups for the file system
func (fileSystemHandler *AwsFileSystemHandler) ListBackup(fsIID irs.IID) ([]irs.FileSystemBackupInfo, error) {
	cblogger.Debug("AWS EFS ListBackup() called")

	// TODO: AWS EFS backup listing implementation
	// According to the specification document, backup functionality will be developed after basic FS create/delete functionality
	// For now, return an error indicating this feature is planned for future development
	return nil, errors.New("backup listing is planned for future development in AWS EFS driver")
}

// GetBackup returns specific backup info
func (fileSystemHandler *AwsFileSystemHandler) GetBackup(fsIID irs.IID, backupID string) (irs.FileSystemBackupInfo, error) {
	cblogger.Debug("AWS EFS GetBackup() called")

	// TODO: AWS EFS backup retrieval implementation
	// According to the specification document, backup functionality will be developed after basic FS create/delete functionality
	// For now, return an error indicating this feature is planned for future development
	return irs.FileSystemBackupInfo{}, errors.New("backup retrieval is planned for future development in AWS EFS driver")
}

// DeleteBackup deletes the specified backup
func (fileSystemHandler *AwsFileSystemHandler) DeleteBackup(fsIID irs.IID, backupID string) (bool, error) {
	cblogger.Debug("AWS EFS DeleteBackup() called")

	// TODO: AWS EFS backup deletion implementation
	// According to the specification document, backup functionality will be developed after basic FS create/delete functionality
	// For now, return an error indicating this feature is planned for future development
	return false, errors.New("backup deletion is planned for future development in AWS EFS driver")
}

// Helper functions

func (fileSystemHandler *AwsFileSystemHandler) waitUntilFileSystemAvailable(fileSystemId string) error {
	cblogger.Info("Waiting for file system to be available...")

	input := &efs.DescribeFileSystemsInput{
		FileSystemId: aws.String(fileSystemId),
	}

	for {
		result, err := fileSystemHandler.Client.DescribeFileSystems(input)
		if err != nil {
			return err
		}

		if len(result.FileSystems) > 0 && result.FileSystems[0] != nil {
			if result.FileSystems[0].LifeCycleState != nil {
				status := *result.FileSystems[0].LifeCycleState
				if status == "available" {
					cblogger.Info("File system is now available")
					return nil
				} else if status == "error" {
					return errors.New("file system creation failed")
				}
			}
		}

		time.Sleep(10 * time.Second)
	}
}

func (fileSystemHandler *AwsFileSystemHandler) listMountTargets(fileSystemId string) ([]*efs.MountTargetDescription, error) {
	input := &efs.DescribeMountTargetsInput{
		FileSystemId: aws.String(fileSystemId),
	}

	result, err := fileSystemHandler.Client.DescribeMountTargets(input)
	if err != nil {
		// Handle common AWS errors gracefully
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "FileSystemNotFound":
				cblogger.Warnf("File system %s not found", fileSystemId)
				return []*efs.MountTargetDescription{}, nil // Return empty slice instead of error
			case "InvalidFileSystemId":
				cblogger.Warnf("Invalid file system ID: %s", fileSystemId)
				return []*efs.MountTargetDescription{}, nil // Return empty slice instead of error
			}
		}
		return nil, err
	}

	if result.MountTargets == nil {
		return []*efs.MountTargetDescription{}, nil
	}

	return result.MountTargets, nil
}

func (fileSystemHandler *AwsFileSystemHandler) deleteMountTarget(mountTargetId string) error {
	input := &efs.DeleteMountTargetInput{
		MountTargetId: aws.String(mountTargetId),
	}

	_, err := fileSystemHandler.Client.DeleteMountTarget(input)
	return err
}

func (fileSystemHandler *AwsFileSystemHandler) convertToFileSystemInfo(fs *efs.FileSystemDescription) (irs.FileSystemInfo, error) {
	// Validate input
	if fs == nil {
		return irs.FileSystemInfo{}, errors.New("file system description is nil")
	}

	// Get mount targets
	var mountTargets []*efs.MountTargetDescription
	if fs.FileSystemId != nil {
		var err error
		mountTargets, err = fileSystemHandler.listMountTargets(*fs.FileSystemId)
		if err != nil {
			cblogger.Errorf("Failed to get mount targets: %v", err)
			// Continue without mount targets
		}
	}

	var mountTargetList []irs.MountTargetInfo
	var accessSubnetList []irs.IID
	for _, mt := range mountTargets {
		if mt == nil {
			continue
		}

		mountTargetInfo := irs.MountTargetInfo{}

		// Safely handle SubnetId
		if mt.SubnetId != nil {
			mountTargetInfo.SubnetIID = irs.IID{SystemId: *mt.SubnetId}
			// Add to AccessSubnetList
			accessSubnetList = append(accessSubnetList, irs.IID{SystemId: *mt.SubnetId})
		}

		// Safely handle IpAddress
		if mt.IpAddress != nil {
			mountTargetInfo.Endpoint = *mt.IpAddress
			mountTargetInfo.MountCommandExample = fmt.Sprintf("sudo mount -t nfs4 %s:/ /mnt/efs", *mt.IpAddress)
		}

		// Get security groups for mount target
		if mt.MountTargetId != nil {
			securityGroups, err := fileSystemHandler.getMountTargetSecurityGroups(*mt.MountTargetId)
			if err != nil {
				cblogger.Errorf("Failed to get security groups for mount target %s: %v", *mt.MountTargetId, err)
			} else {
				mountTargetInfo.SecurityGroups = securityGroups
			}
		}

		// Add complete AWS mount target object information using StructToKeyValueList
		mountTargetInfo.KeyValueList = irs.StructToKeyValueList(mt)

		mountTargetList = append(mountTargetList, mountTargetInfo)
	}

	// Convert performance mode and throughput mode
	performanceInfo := make(map[string]string)

	// Convert throughput mode (AWS API -> CB-Spider format)
	if fs.ThroughputMode != nil {
		switch *fs.ThroughputMode {
		case "bursting":
			performanceInfo["ThroughputMode"] = "Elastic" // AWS Console shows "Elastic" for bursting mode
		case "provisioned":
			performanceInfo["ThroughputMode"] = "Provisioned"
		}
	}

	// Convert performance mode (AWS API -> CB-Spider format)
	if fs.PerformanceMode != nil {
		switch *fs.PerformanceMode {
		case "generalPurpose":
			performanceInfo["PerformanceMode"] = "GeneralPurpose"
		case "maxIO":
			performanceInfo["PerformanceMode"] = "MaxIO"
		}
	}

	// Add provisioned throughput if available
	if fs.ProvisionedThroughputInMibps != nil {
		performanceInfo["ProvisionedThroughput"] = fmt.Sprintf("%.0f", *fs.ProvisionedThroughputInMibps)
	}

	// Get VPC ID from mount targets if available
	vpcIID := irs.IID{}
	if len(mountTargets) > 0 {
		// Get VPC ID from the first mount target's subnet
		for _, mt := range mountTargets {
			if mt != nil && mt.SubnetId != nil {
				vpcId, err := fileSystemHandler.getVPCFromSubnet(*mt.SubnetId)
				if err != nil {
					cblogger.Errorf("Failed to get VPC ID from subnet %s: %v", *mt.SubnetId, err)
					continue
				}
				if vpcId != "" {
					vpcIID = irs.IID{SystemId: vpcId}
					break
				}
			}
		}
	}

	// Safely handle Name
	var nameId string
	if fs.Name != nil {
		nameId = *fs.Name
	}

	// Safely handle FileSystemId
	var systemId string
	if fs.FileSystemId != nil {
		systemId = *fs.FileSystemId
	}

	// Safely handle Encryption
	var encryption bool
	if fs.Encrypted != nil {
		encryption = *fs.Encrypted
	}

	// Safely handle LifeCycleState
	var status irs.FileSystemStatus
	if fs.LifeCycleState != nil {
		status = fileSystemHandler.convertStatus(*fs.LifeCycleState)
	} else {
		status = irs.FileSystemError
	}

	// Safely handle CreationTime
	var createdTime time.Time
	if fs.CreationTime != nil {
		createdTime = *fs.CreationTime
	}

	// Get backup policy information
	backupPolicy, err := fileSystemHandler.getBackupPolicy(systemId)
	if err != nil {
		cblogger.Errorf("Failed to get backup policy: %v", err)
		backupPolicy = "UNKNOWN"
	}

	// Get complete backup policy object for KeyValueList
	backupPolicyObject, err := fileSystemHandler.getBackupPolicyObject(systemId)
	if err != nil {
		cblogger.Errorf("Failed to get backup policy object: %v", err)
		backupPolicyObject = nil
	} else if backupPolicyObject == nil {
		// PolicyNotFound case - backup is disabled
		cblogger.Infof("Backup policy not found for file system %s (backup is disabled)", systemId)
	}

	// Convert AWS EFS backup policy to CB-Spider BackupSchedule format
	var backupSchedule irs.FileSystemBackupInfo
	if backupPolicy == "ENABLED" {
		// AWS EFS automatic backup is enabled - map to CB-Spider format
		backupSchedule = irs.FileSystemBackupInfo{
			FileSystemIID: systemId,
			Schedule: irs.CronSchedule{
				Minute:     "0", // AWS EFS runs daily at midnight UTC
				Hour:       "0", // AWS EFS runs daily at midnight UTC
				DayOfMonth: "*", // Every day
				Month:      "*", // Every month
				DayOfWeek:  "*", // Every day of week
			},
			BackupID:     "aws-efs-automatic-backup", // AWS EFS doesn't provide individual backup IDs
			CreationTime: createdTime,
			KeyValueList: []irs.KeyValue{
				{Key: "BackupType", Value: "AWS EFS Automatic Backup"},
				{Key: "BackupPolicy", Value: backupPolicy},
				{Key: "Note", Value: "AWS EFS automatic backup runs daily at midnight UTC"},
			},
		}

		// Add complete AWS backup policy object information using StructToKeyValueList
		if backupPolicyObject != nil {
			awsBackupPolicyKeyValueList := irs.StructToKeyValueList(backupPolicyObject)
			backupSchedule.KeyValueList = append(backupSchedule.KeyValueList, awsBackupPolicyKeyValueList...)
		}
	}
	// If backup is disabled, backupSchedule remains zero value (empty struct)

	// Convert tags
	var tagList []irs.KeyValue
	if fs.Tags != nil {
		for _, tag := range fs.Tags {
			if tag.Key != nil && tag.Value != nil {
				tagList = append(tagList, irs.KeyValue{
					Key:   *tag.Key,
					Value: *tag.Value,
				})
			}
		}
	}

	// Determine file system type and zone based on availability zone
	fileSystemType := irs.RegionType // Default to Regional
	var zone string
	if fs.AvailabilityZoneId != nil && *fs.AvailabilityZoneId != "" {
		fileSystemType = irs.ZoneType // One Zone
		zone = *fs.AvailabilityZoneId
	}

	// Create additional key-value list for extra information
	var keyValueList []irs.KeyValue
	keyValueList = append(keyValueList, irs.KeyValue{Key: "AvailabilityZoneId", Value: zone})
	if fs.AvailabilityZoneName != nil {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "AvailabilityZoneName", Value: *fs.AvailabilityZoneName})
	}
	if fs.KmsKeyId != nil {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "KmsKeyId", Value: *fs.KmsKeyId})
	}
	if fs.OwnerId != nil {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "OwnerId", Value: *fs.OwnerId})
	}
	if fs.SizeInBytes != nil && fs.SizeInBytes.Value != nil {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "SizeInBytes", Value: fmt.Sprintf("%d", *fs.SizeInBytes.Value)})
	}

	// Add complete AWS EFS object information using StructToKeyValueList
	awsFileSystemKeyValueList := irs.StructToKeyValueList(fs)
	keyValueList = append(keyValueList, awsFileSystemKeyValueList...)

	fileSystemInfo := irs.FileSystemInfo{
		IId: irs.IID{
			NameId:   nameId,
			SystemId: systemId,
		},
		Region:           fileSystemHandler.Region.Region,
		Zone:             zone,
		VpcIID:           vpcIID,
		AccessSubnetList: accessSubnetList,
		Encryption:       encryption,
		TagList:          tagList,
		FileSystemType:   fileSystemType,
		NFSVersion:       "4.1", // AWS EFS uses NFS 4.1 by default, but supports 4.0 for mounting
		CapacityGB:       0,     // AWS EFS doesn't support capacity specification
		PerformanceInfo:  performanceInfo,
		Status:           status,
		UsedSizeGB:       0, // AWS EFS doesn't provide used size
		MountTargetList:  mountTargetList,
		CreatedTime:      createdTime,
		BackupSchedule:   backupSchedule,
		KeyValueList:     keyValueList,
	}

	return fileSystemInfo, nil
}

func (fileSystemHandler *AwsFileSystemHandler) convertStatus(awsStatus string) irs.FileSystemStatus {
	switch awsStatus {
	case "creating":
		return irs.FileSystemCreating
	case "available":
		return irs.FileSystemAvailable
	case "deleting":
		return irs.FileSystemDeleting
	case "error":
		return irs.FileSystemError
	default:
		return irs.FileSystemError
	}
}

// convertZoneNameToZoneId converts zone name to zone ID format for AWS EFS One Zone
// e.g., us-east-1a -> use1-az1, ap-northeast-2a -> apne2-az1
func convertZoneNameToZoneId(zoneName string) string {
	// This is a simplified mapping. In production, you might want to use AWS API
	// to get the actual zone ID mapping for each region.

	// Common zone name to zone ID mappings
	zoneMapping := map[string]string{
		// US East (N. Virginia)
		"us-east-1a": "use1-az1",
		"us-east-1b": "use1-az2",
		"us-east-1c": "use1-az3",
		"us-east-1d": "use1-az4",
		"us-east-1e": "use1-az5",
		"us-east-1f": "use1-az6",

		// US West (Oregon)
		"us-west-2a": "usw2-az1",
		"us-west-2b": "usw2-az2",
		"us-west-2c": "usw2-az3",
		"us-west-2d": "usw2-az4",

		// Asia Pacific (Seoul)
		"ap-northeast-2a": "apne2-az1",
		"ap-northeast-2b": "apne2-az2",
		"ap-northeast-2c": "apne2-az3",
		"ap-northeast-2d": "apne2-az4",

		// Asia Pacific (Tokyo)
		"ap-northeast-1a": "apne1-az1",
		"ap-northeast-1b": "apne1-az2",
		"ap-northeast-1c": "apne1-az3",
		"ap-northeast-1d": "apne1-az4",

		// Europe (Frankfurt)
		"eu-central-1a": "euc1-az1",
		"eu-central-1b": "euc1-az2",
		"eu-central-1c": "euc1-az3",
	}

	if zoneId, exists := zoneMapping[zoneName]; exists {
		return zoneId
	}

	// If mapping not found, return the original zone name
	// This will cause an error, but it's better than silent failure
	cblogger.Warnf("Zone mapping not found for %s, using original name", zoneName)
	return zoneName
}

// getAvailableZones gets available zones in the current region
func (fileSystemHandler *AwsFileSystemHandler) getAvailableZones() ([]string, error) {
	return fileSystemHandler.getAvailableZonesInRegion(fileSystemHandler.Region.Region)
}

// getAvailableZonesInRegion gets available zones in the specified region
func (fileSystemHandler *AwsFileSystemHandler) getAvailableZonesInRegion(region string) ([]string, error) {
	if fileSystemHandler.EC2Client == nil {
		return nil, errors.New("EC2 client not initialized")
	}

	// If the requested region is the same as the current client's region, use the current client
	if region == fileSystemHandler.Region.Region {
		input := &ec2.DescribeAvailabilityZonesInput{
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("state"),
					Values: []*string{aws.String("available")},
				},
			},
		}

		result, err := fileSystemHandler.EC2Client.DescribeAvailabilityZones(input)
		if err != nil {
			return nil, err
		}

		var zones []string
		for _, zone := range result.AvailabilityZones {
			if zone.ZoneName != nil {
				zones = append(zones, *zone.ZoneName)
			}
		}

		return zones, nil
	}

	// For different regions, we need to create a new EC2 client for that region
	// This requires the session and credentials from the current client
	// For now, we'll use the current client and let AWS handle the region mismatch
	// In a production environment, you might want to create a new client for the target region

	cblogger.Warnf("Requested region %s differs from current client region %s. Using current client.", region, fileSystemHandler.Region.Region)

	input := &ec2.DescribeAvailabilityZonesInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("state"),
				Values: []*string{aws.String("available")},
			},
		},
	}

	result, err := fileSystemHandler.EC2Client.DescribeAvailabilityZones(input)
	if err != nil {
		return nil, err
	}

	var zones []string
	for _, zone := range result.AvailabilityZones {
		if zone.ZoneName != nil {
			zones = append(zones, *zone.ZoneName)
		}
	}

	return zones, nil
}

// createDefaultMountTargets creates mount targets based on file system type
func (fileSystemHandler *AwsFileSystemHandler) createDefaultMountTargets(fileSystemId string, reqInfo irs.FileSystemInfo, targetZoneId string) error {
	if fileSystemHandler.EC2Client == nil {
		return errors.New("EC2 client not initialized")
	}

	// AWS EFS is created in the region where the EFS client is configured
	targetRegion := fileSystemHandler.Region.Region
	cblogger.Infof("Using handler's region: %s", targetRegion)

	// Get available zones for the current region
	availableZones, err := fileSystemHandler.getAvailableZonesInRegion(targetRegion)
	if err != nil {
		return fmt.Errorf("failed to get available zones for region %s: %v", targetRegion, err)
	}

	if len(availableZones) == 0 {
		return errors.New("no available zones found")
	}

	if reqInfo.FileSystemType == irs.ZoneType {
		// For One Zone EFS: Create mount target in the specified zone
		zoneId := targetZoneId
		if zoneId == "" {
			// Fallback to original logic if targetZoneId is not provided
			zoneId = reqInfo.Zone
			if zoneId == "" {
				zoneId = fileSystemHandler.Region.Zone
				if zoneId == "" {
					zoneId = availableZones[0] // Use first available zone
				}
			}
		}

		// Find a subnet in the specified zone
		subnetId, err := fileSystemHandler.findSubnetInZone(zoneId, reqInfo.VpcIID.SystemId)
		if err != nil {
			return fmt.Errorf("failed to find subnet in zone %s: %v", zoneId, err)
		}

		// Find security groups for this subnet from MountTargetList
		var securityGroups []string
		for _, mtInfo := range reqInfo.MountTargetList {
			if mtInfo.SubnetIID.SystemId == subnetId {
				securityGroups = mtInfo.SecurityGroups
				break
			}
		}

		// Create mount target with security groups
		input := &efs.CreateMountTargetInput{
			FileSystemId: aws.String(fileSystemId),
			SubnetId:     aws.String(subnetId),
		}
		if len(securityGroups) > 0 {
			input.SecurityGroups = aws.StringSlice(securityGroups)
			cblogger.Infof("Creating mount target with security groups: %v", securityGroups)
		}

		_, err = fileSystemHandler.Client.CreateMountTarget(input)
		if err != nil {
			return fmt.Errorf("failed to create mount target in zone %s: %v", zoneId, err)
		}

		cblogger.Infof("Created mount target for One Zone EFS in zone: %s", zoneId)
	} else {
		// For Regional EFS: Create mount targets in ALL available AZs (like AWS Console)
		// This provides maximum availability and follows AWS Console behavior
		cblogger.Infof("Creating mount targets in all %d available zones for Regional EFS", len(availableZones))

		for _, zoneId := range availableZones {
			// Find a subnet in this zone
			subnetId, err := fileSystemHandler.findSubnetInZone(zoneId, reqInfo.VpcIID.SystemId)
			if err != nil {
				cblogger.Errorf("Failed to find subnet in zone %s: %v", zoneId, err)
				continue
			}

			// Find security groups for this subnet from MountTargetList
			var securityGroups []string
			for _, mtInfo := range reqInfo.MountTargetList {
				if mtInfo.SubnetIID.SystemId == subnetId {
					securityGroups = mtInfo.SecurityGroups
					break
				}
			}

			// Create mount target with security groups
			input := &efs.CreateMountTargetInput{
				FileSystemId: aws.String(fileSystemId),
				SubnetId:     aws.String(subnetId),
			}
			if len(securityGroups) > 0 {
				input.SecurityGroups = aws.StringSlice(securityGroups)
				cblogger.Infof("Creating mount target with security groups: %v", securityGroups)
			}

			_, err = fileSystemHandler.Client.CreateMountTarget(input)
			if err != nil {
				if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "MountTargetConflict" {
					cblogger.Infof("Mount target already exists in zone %s", zoneId)
				} else {
					cblogger.Errorf("Failed to create mount target in zone %s: %v", zoneId, err)
				}
			} else {
				cblogger.Infof("Created mount target for Regional EFS in zone: %s", zoneId)
			}
		}
	}

	return nil
}

// setDefaultLifecyclePolicy sets the default lifecycle policy according to AWS Console behavior
func (fileSystemHandler *AwsFileSystemHandler) setDefaultLifecyclePolicy(fileSystemId string) error {
	cblogger.Info("Setting default lifecycle policy for EFS file system")

	// AWS Console default lifecycle policy:
	// - Infrequent Access: 30 days after last access
	// - Archive: 90 days after last access
	// - Standard: No automatic transition (files stay in Standard)

	lifecyclePolicies := []*efs.LifecyclePolicy{
		{
			TransitionToIA: aws.String("AFTER_30_DAYS"), // 30 days after last access
		},
		// {
		// 	TransitionToArchive: aws.String("AFTER_90_DAYS"), // 90 days after last access (Archive) - Support in aws v2 version
		// },
	}

	input := &efs.PutLifecycleConfigurationInput{
		FileSystemId:      aws.String(fileSystemId),
		LifecyclePolicies: lifecyclePolicies,
	}

	_, err := fileSystemHandler.Client.PutLifecycleConfiguration(input)
	if err != nil {
		return fmt.Errorf("failed to set lifecycle policy: %v", err)
	}

	cblogger.Info("Successfully set default lifecycle policy")
	return nil
}

// getBackupPolicy gets the backup policy for the specified file system
func (fileSystemHandler *AwsFileSystemHandler) getBackupPolicy(fileSystemId string) (string, error) {
	input := &efs.DescribeBackupPolicyInput{
		FileSystemId: aws.String(fileSystemId),
	}

	result, err := fileSystemHandler.Client.DescribeBackupPolicy(input)
	if err != nil {
		// Handle PolicyNotFound error (404) - this means backup policy is not set
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "PolicyNotFound" {
				return "DISABLED", nil // Backup policy not found means it's disabled
			}
		}
		return "", err
	}

	if result.BackupPolicy == nil || result.BackupPolicy.Status == nil {
		return "UNKNOWN", nil
	}

	return *result.BackupPolicy.Status, nil
}

// getBackupPolicyObject gets the complete backup policy object for the specified file system
func (fileSystemHandler *AwsFileSystemHandler) getBackupPolicyObject(fileSystemId string) (*efs.BackupPolicy, error) {
	input := &efs.DescribeBackupPolicyInput{
		FileSystemId: aws.String(fileSystemId),
	}

	result, err := fileSystemHandler.Client.DescribeBackupPolicy(input)
	if err != nil {
		// Handle PolicyNotFound error (404) - this means backup policy is not set
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "PolicyNotFound" {
				return nil, nil // Return nil instead of error for PolicyNotFound
			}
		}
		return nil, err
	}

	return result.BackupPolicy, nil
}

// updateBackupPolicy updates the backup policy for the specified file system
func (fileSystemHandler *AwsFileSystemHandler) updateBackupPolicy(fileSystemId string, enabled bool) error {
	status := "DISABLED"
	if enabled {
		status = "ENABLED"
	}

	backupPolicy := &efs.BackupPolicy{
		Status: aws.String(status),
	}

	input := &efs.PutBackupPolicyInput{
		FileSystemId: aws.String(fileSystemId),
		BackupPolicy: backupPolicy,
	}

	_, err := fileSystemHandler.Client.PutBackupPolicy(input)
	return err
}

// getMountTargetSecurityGroups gets security groups for a mount target
func (fileSystemHandler *AwsFileSystemHandler) getMountTargetSecurityGroups(mountTargetId string) ([]string, error) {
	input := &efs.DescribeMountTargetSecurityGroupsInput{
		MountTargetId: aws.String(mountTargetId),
	}

	result, err := fileSystemHandler.Client.DescribeMountTargetSecurityGroups(input)
	if err != nil {
		// Handle common AWS errors gracefully
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "MountTargetNotFound":
				cblogger.Warnf("Mount target %s not found", mountTargetId)
				return []string{}, nil // Return empty slice instead of error
			case "InvalidMountTargetId":
				cblogger.Warnf("Invalid mount target ID: %s", mountTargetId)
				return []string{}, nil // Return empty slice instead of error
			}
		}
		return nil, err
	}

	if result.SecurityGroups == nil {
		return []string{}, nil
	}

	return aws.StringValueSlice(result.SecurityGroups), nil
}

// findSubnetInZone finds a subnet in the specified zone and VPC
func (fileSystemHandler *AwsFileSystemHandler) findSubnetInZone(zoneId, vpcId string) (string, error) {
	if fileSystemHandler.EC2Client == nil {
		return "", errors.New("EC2 client not initialized")
	}

	input := &ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("availability-zone"),
				Values: []*string{aws.String(zoneId)},
			},
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(vpcId)},
			},
			{
				Name:   aws.String("state"),
				Values: []*string{aws.String("available")},
			},
		},
	}

	result, err := fileSystemHandler.EC2Client.DescribeSubnets(input)
	if err != nil {
		return "", err
	}

	if len(result.Subnets) == 0 {
		return "", fmt.Errorf("no available subnets found in zone %s for VPC %s", zoneId, vpcId)
	}

	// Return the first available subnet
	return *result.Subnets[0].SubnetId, nil
}

// getVPCFromSubnet gets VPC ID from subnet ID using EC2 API
func (fileSystemHandler *AwsFileSystemHandler) getVPCFromSubnet(subnetId string) (string, error) {
	if fileSystemHandler.EC2Client == nil {
		return "", errors.New("EC2 client not initialized")
	}

	input := &ec2.DescribeSubnetsInput{
		SubnetIds: []*string{aws.String(subnetId)},
	}

	result, err := fileSystemHandler.EC2Client.DescribeSubnets(input)
	if err != nil {
		return "", err
	}

	if len(result.Subnets) == 0 {
		return "", errors.New("subnet not found")
	}

	if result.Subnets[0].VpcId == nil {
		return "", errors.New("VPC ID not found in subnet")
	}

	return *result.Subnets[0].VpcId, nil
}

// createMountTargetWithSecurityGroups creates a mount target with specified security groups
func (fileSystemHandler *AwsFileSystemHandler) createMountTargetWithSecurityGroups(iid irs.IID, subnetIID irs.IID, securityGroups []string) error {
	cblogger.Debug("AWS EFS createMountTargetWithSecurityGroups() called")

	// Create mount target
	input := &efs.CreateMountTargetInput{
		FileSystemId: aws.String(iid.SystemId),
		SubnetId:     aws.String(subnetIID.SystemId),
	}

	// Add security groups if specified
	if len(securityGroups) > 0 {
		input.SecurityGroups = aws.StringSlice(securityGroups)
		cblogger.Infof("Creating mount target with security groups: %v", securityGroups)
	}

	hiscallInfo := GetCallLogScheme(fileSystemHandler.Region, call.FILESYSTEM, iid.SystemId, "CreateMountTarget()")
	start := call.Start()

	result, err := fileSystemHandler.Client.CreateMountTarget(input)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == "MountTargetConflict" {
				cblogger.Info("Mount target already exists for this subnet")
				LoggingInfo(hiscallInfo, start)
			} else {
				cblogger.Error(err)
				LoggingError(hiscallInfo, err)
				return err
			}
		} else {
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return err
		}
	} else {
		cblogger.Infof("Created mount target %s for file system %s in subnet %s",
			*result.MountTargetId, iid.SystemId, subnetIID.SystemId)
		LoggingInfo(hiscallInfo, start)
	}

	// Wait for mount target to be available
	time.Sleep(10 * time.Second)

	return nil
}

// getSubnetZone gets the availability zone of a subnet using EC2 API
func (fileSystemHandler *AwsFileSystemHandler) getSubnetZone(subnetId string) (string, error) {
	if fileSystemHandler.EC2Client == nil {
		return "", errors.New("EC2 client not initialized")
	}

	input := &ec2.DescribeSubnetsInput{
		SubnetIds: []*string{aws.String(subnetId)},
	}

	result, err := fileSystemHandler.EC2Client.DescribeSubnets(input)
	if err != nil {
		return "", err
	}

	if len(result.Subnets) == 0 {
		return "", errors.New("subnet not found")
	}

	if result.Subnets[0].AvailabilityZone == nil {
		return "", errors.New("availability zone not found in subnet")
	}

	return *result.Subnets[0].AvailabilityZone, nil
}
