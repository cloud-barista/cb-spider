// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2025.06.

package resources

import (
	"errors"
	"fmt"
	"strings"
	"time"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	cfs "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cfs/v20190719"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"

	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

type TencentFileSystemHandler struct {
	Region    idrv.RegionInfo
	CFSClient *cfs.Client
	VPCClient *vpc.Client
}

// Storage type of the file system.
// Valid values: SD (Standard), HP (High-Performance),
// TB (Standard Turbo), and TP (High-Performance Turbo). Default value: SD.
type TfsStorageType string

const TfsStorageTypeStandard TfsStorageType = "SD"
const TfsStorageTypeHighPerformance TfsStorageType = "HP"
const TfsStorageTypeStandardTurbo TfsStorageType = "TB"
const TfsStorageTypeHighPerformanceTurbo TfsStorageType = "TP"

type TfsNetInterfaceType string

const TfsNetInterfaceTypeVPC TfsNetInterfaceType = "VPC" //for a Standard or High-Performance file system
const TfsNetInterfaceTypeCCN TfsNetInterfaceType = "CCN" //for a Standard Turbo or High-Performance Turbo one.

type TfsProtocol string

const TfsProtocolNFS TfsProtocol = "NFS" // default
const TfsProtocolCIFS TfsProtocol = "CIFS"
const TfsProtocolTURBO TfsProtocol = "TURBO"

type TfsLifeCycleState string

const TfsLifeCycleStateCreating TfsLifeCycleState = "creating"
const TfsLifeCycleStateCreateFailed TfsLifeCycleState = "create_failed"
const TfsLifeCycleStateAvailable TfsLifeCycleState = "available"
const TfsLifeCycleStateMounting TfsLifeCycleState = "mounting"
const TfsLifeCycleStateMountFailed TfsLifeCycleState = "unserviced"
const TfsLifeCycleStateUnmounting TfsLifeCycleState = "upgrading"
const TfsLifeCycleStateDeleting TfsLifeCycleState = "deleting" // actualy it is not a tencent file system state. just for CB-Spider

// Performance specifications for different storage types
type TfsPerformanceSpec struct {
	Tier                           string
	MaxSystemCapacity              string
	MinSystemCapacity              string
	MaxSystemBandwidth             string
	MaxSystemFiles                 string
	MaxSystemDirectories           string
	MaxFilenameLength              string
	MaxAbsolutePathLength          string
	MaxDirectoryLevels             string
	MaxFilesPerDirectory           string
	MaxConcurrentlyOpenedFiles     string
	MaxLocksPerFile                string
	MaxClients                     string
	MaxBandwidthPerClient          string
	MaxMountedFileSystemsPerClient string
	Billing                        string
	SupportedProtocol              string
	SupportedOS                    string
	Notes                          string
}

// Performance specifications map for each storage type
var performanceSpecs = map[TfsStorageType]TfsPerformanceSpec{
	TfsStorageTypeStandard: {
		Tier:                           "STANDARD",
		MaxSystemCapacity:              "160 TiB",
		MaxSystemBandwidth:             "300 MiB/s",
		MaxSystemFiles:                 "Min[15,000 x used capacity (GiB), 1 billion]",
		MaxSystemDirectories:           "10 million",
		MaxFilenameLength:              "255 bytes",
		MaxAbsolutePathLength:          "4096 bytes",
		MaxDirectoryLevels:             "1000",
		MaxFilesPerDirectory:           "1 million",
		MaxConcurrentlyOpenedFiles:     "65536",
		MaxLocksPerFile:                "512",
		MaxClients:                     "1000",
		MaxBandwidthPerClient:          "300 MiB/s",
		MaxMountedFileSystemsPerClient: "1000",
		Billing:                        "Billed by the actual usage (excluding prepaid)",
		SupportedProtocol:              "NFS/SMB",
		SupportedOS:                    "Linux/Windows",
		Notes:                          "Recommended for general data storage, log storage, and data backup",
	},
	TfsStorageTypeHighPerformance: {
		Tier:                           "HIGH_PERFORMANCE",
		MaxSystemCapacity:              "32 TiB",
		MinSystemCapacity:              "40 TiB",
		MaxSystemBandwidth:             "1 GiB/s",
		MaxSystemFiles:                 "Min[20,000 x used capacity (GiB), 15 billion]",
		MaxSystemDirectories:           "15 million",
		MaxFilenameLength:              "255 bytes",
		MaxAbsolutePathLength:          "4096 bytes",
		MaxDirectoryLevels:             "1000",
		MaxFilesPerDirectory:           "1 million",
		MaxConcurrentlyOpenedFiles:     "65536",
		MaxLocksPerFile:                "512",
		MaxClients:                     "1000",
		MaxBandwidthPerClient:          "500 MiB/s",
		MaxMountedFileSystemsPerClient: "1000",
		Billing:                        "Billed by the purchased capacity",
		SupportedProtocol:              "NFS",
		SupportedOS:                    "Linux",
		Notes:                          "High performance and low latency - Suitable for latency-sensitive core businesses such as DevOps, website application source code, and cloud desktop",
	},
	TfsStorageTypeStandardTurbo: {
		Tier:                           "STANDARD_TURBO",
		MaxSystemCapacity:              "100 PiB",
		MaxSystemBandwidth:             "100 GiB/s",
		MaxSystemFiles:                 "Min[15,000 x deployed capacity (GiB), 1 billion]",
		MaxSystemDirectories:           "10 million",
		MaxFilenameLength:              "255 bytes",
		MaxAbsolutePathLength:          "4096 bytes",
		MaxDirectoryLevels:             "1000",
		MaxFilesPerDirectory:           "1 million",
		MaxConcurrentlyOpenedFiles:     "65536",
		MaxLocksPerFile:                "512",
		MaxClients:                     "2000",
		MaxBandwidthPerClient:          "10 GiB/s",
		MaxMountedFileSystemsPerClient: "16",
		Billing:                        "Billed by the purchased capacity",
		SupportedProtocol:              "POSIX/MPI",
		SupportedOS:                    "Linux",
		Notes:                          "Turbo series - Mounted using client, billed by purchased capacity, cannot be scaled down, initial creation takes ~20 minutes",
	},
	TfsStorageTypeHighPerformanceTurbo: {
		Tier:                           "HIGH_PERFORMANCE_TURBO",
		MaxSystemCapacity:              "100 PiB",
		MaxSystemBandwidth:             "100 GiB/s",
		MaxSystemFiles:                 "Min[30,000 x deployed capacity (GiB), 1.5 billion]",
		MaxSystemDirectories:           "15 million",
		MaxFilenameLength:              "255 bytes",
		MaxAbsolutePathLength:          "4096 bytes",
		MaxDirectoryLevels:             "1000",
		MaxFilesPerDirectory:           "1 million",
		MaxConcurrentlyOpenedFiles:     "65536",
		MaxLocksPerFile:                "512",
		MaxClients:                     "2000",
		MaxBandwidthPerClient:          "10 GiB/s",
		MaxMountedFileSystemsPerClient: "16",
		Billing:                        "Billed by the purchased capacity",
		SupportedProtocol:              "POSIX/MPI",
		SupportedOS:                    "Linux",
		Notes:                          "High-Performance Turbo series - Highest performance tier, mounted using client, billed by purchased capacity, cannot be scaled down",
	},
}

// GetMetaInfo returns metadata about file system capabilities
func (fsHandler *TencentFileSystemHandler) GetMetaInfo() (irs.FileSystemMetaInfo, error) {
	cblogger.Info("Start GetMetaInfo()")

	metaInfo := irs.FileSystemMetaInfo{
		// Tencent Cloud CFS supports both Region and Zone level file systems
		SupportsFileSystemType: map[irs.FileSystemType]bool{
			irs.RegionType: true, // CFS Turbo supports cross-zone deployment
			irs.ZoneType:   true, // Standard CFS (SD, HP) is zone-specific
		},
		// Tencent Cloud CFS requires VPC for network connectivity
		SupportsVPC: map[irs.RSType]bool{
			irs.VPC: true,
		},
		// Tencent Cloud CFS supports NFS 3.0 and 4.0 protocols
		SupportsNFSVersion: []string{"3.0", "4.0"},
		// Tencent Cloud CFS supports capacity specification
		SupportsCapacity: true,
		// Capacity options based on storage type
		// Standard (SD, HP): 10GB to 32TB
		// Turbo (TB, TP): 20TB to 1PB
		CapacityGBOptions: map[string]irs.CapacityGBRange{
			// SD Capacity:0-160TiB
			string(TfsStorageTypeStandard):             {Min: 4000, Max: 163840},      // SD, HP: 4GB to 160TiB
			string(TfsStorageTypeStandardTurbo):        {Min: 40960, Max: 160 * 1024}, // TB, TP: 20TB to 1PB
			string(TfsStorageTypeHighPerformance):      {Min: 10, Max: 32768},         // SD, HP: 10GB to 32TB
			string(TfsStorageTypeHighPerformanceTurbo): {Min: 10, Max: 32768},         // SD, HP: 10GB to 32TB
			// HP Capacity:0-32TiB

		},
		// SD default 160TiB
		// File system capacity, in GiB (required for the Turbo series).
		// For Standard Turbo, the minimum purchase required is 40,960 GiB (40 TiB)
		// and the expansion increment is 20,480 GiB (20 TiB).
		// For High-Performance Turbo, the minimum purchase required is 20,480 GiB (20 TiB)
		// and the expansion increment is 10,240 GiB (10 TiB).

		// Performance options based on storage type
		// Standard: SD (Standard), HP (High-Performance)
		// Turbo: TB (Turbo), TP (High-Performance Turbo)
		PerformanceOptions: map[string][]string{
			"StorageType": {string(TfsStorageTypeStandard), string(TfsStorageTypeStandardTurbo), string(TfsStorageTypeHighPerformance), string(TfsStorageTypeHighPerformanceTurbo)},
		},
	}

	return metaInfo, nil
}

// ListIID returns list of file system IDs
func (fsHandler *TencentFileSystemHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Start ListIID()")

	// 공통 함수를 사용하여 모든 파일시스템 조회 (필터 없음)
	fileSystems, err := fsHandler.describeCfsFileSystem(nil, nil, nil)
	if err != nil {
		return nil, err
	}

	// IID 목록으로 변환
	var allIidList []*irs.IID
	for _, fs := range fileSystems {
		iid := &irs.IID{
			//NameId:   *fs.FsName, // NameId 는 나중에 설정 됨.
			SystemId: *fs.FileSystemId,
		}
		allIidList = append(allIidList, iid)
	}

	cblogger.Infof("Total file system IDs extracted: %d", len(allIidList))
	return allIidList, nil
}

// CreateFileSystem creates a new file system
func (fsHandler *TencentFileSystemHandler) CreateFileSystem(reqInfo irs.FileSystemInfo) (irs.FileSystemInfo, error) {
	cblogger.Info("Start CreateFileSystem()")

	// Check if file system already exists :
	// exist, err := fsHandler.fileSystemExist(reqInfo.IId)
	// if err != nil {
	// 	return irs.FileSystemInfo{}, err
	// }

	// if exist {
	// 	return irs.FileSystemInfo{}, errors.New("file system already exists")
	// }

	// Get meta info for validation and default values
	metaInfo, err := fsHandler.GetMetaInfo()
	if err != nil {
		cblogger.Error(err)
		return irs.FileSystemInfo{}, err
	}

	ccnId := ""
	cidrBlock := ""
	pGroupId := "pgroupbasic" // Default permission group

	if reqInfo.KeyValueList != nil {
		for _, kv := range reqInfo.KeyValueList {
			switch kv.Key {
			case "CcnId":
				ccnId = kv.Value
			case "CidrBlock":
				cidrBlock = kv.Value
			case "PGroupId":
				pGroupId = kv.Value
			}
		}
	}

	request := cfs.NewCreateCfsFileSystemRequest()
	request.Zone = common.StringPtr(reqInfo.Zone)
	request.FsName = common.StringPtr(reqInfo.IId.NameId)

	// FileSystemType FileSystemType `json:"FileSystemType,omitempty" example:"RegionType"` // RegionType, ZoneType; CSP default if omitted
	// NFSVersion     string         `json:"NFSVersion" validate:"required" example:"4.1"`  // NFS protocol version, e.g., "3.0", "4.1"; CSP default if omitted
	// CapacityGB     int64          `json:"CapacityGB,omitempty" example:"1024"`           // Capacity in GB, -1 when not applicable. Ignored if CSP unsupported.; CSP default if omitted

	// Set storage type based on performance info or use default

	storageType := string(TfsStorageTypeStandard) // Default to Standard

	if reqInfo.PerformanceInfo != nil {
		// FileSystemType를 Tier로 보면 되나?
		// if fileSystemType, exists := reqInfo.PerformanceInfo["FileSystemType"]; exists {
		// 	switch fileSystemType {
		// 	case "Standard":
		// 		storageType = "SD"
		// 		// Throughput:Max 300 MiB/s
		// 		// IOPS:Max 15,000
		// 		// Latency:Milliseconds
		// 		// Capacity:0-160TiB

		// 		// available Region check
		// 	case "High-Performance":
		// 		storageType = "HP"
		// 		// Throughput:Max 1 GiB/s
		// 		// IOPS:Max 30,000
		// 		// Latency:Submillisecond
		// 		// Capacity:0-32TiB

		// 		// available Region check
		// 	default:
		// 		cblogger.Warnf("Unknown file system type: %s, using default SD", fileSystemType)
		// 	}
		// }

		if tier, exists := reqInfo.PerformanceInfo["Tier"]; exists {
			// Map performance tier to storage type
			switch tier {
			case string(TfsStorageTypeStandard):
				storageType = string(TfsStorageTypeStandard)
			case string(TfsStorageTypeHighPerformance):
				storageType = string(TfsStorageTypeHighPerformance)
			case string(TfsStorageTypeStandardTurbo):
				storageType = string(TfsStorageTypeStandardTurbo)
			case string(TfsStorageTypeHighPerformanceTurbo):
				storageType = string(TfsStorageTypeHighPerformanceTurbo)
			default:
				cblogger.Warnf("Unknown performance tier: %s, using default SD", tier)
			}
		}
	}
	request.StorageType = common.StringPtr(storageType)

	// Set network interface based on storage type
	// VPC for Standard/High-Performance, CCN for Turbo series
	if storageType == string(TfsStorageTypeStandardTurbo) || storageType == string(TfsStorageTypeHighPerformanceTurbo) {
		// For CCN, we need CcnId and CidrBlock (required)
		// These should be provided in reqInfo.KeyValueList

		// Validate required CCN parameters
		if ccnId == "" {
			return irs.FileSystemInfo{}, errors.New("CcnId is required for Turbo file systems (CCN network interface)")
		}
		if cidrBlock == "" {
			return irs.FileSystemInfo{}, errors.New("CidrBlock is required for Turbo file systems (CCN network interface)")
		}

		request.NetInterface = common.StringPtr(string(TfsNetInterfaceTypeCCN))
		request.CcnId = common.StringPtr(ccnId)
		request.CidrBlock = common.StringPtr(cidrBlock)
	} else {

		if reqInfo.VpcIID.SystemId == "" {
			return irs.FileSystemInfo{}, errors.New("VPC ID is required")
		}
		if len(reqInfo.AccessSubnetList) == 0 {
			return irs.FileSystemInfo{}, errors.New("AccessSubnetList is required")
		}

		request.NetInterface = common.StringPtr(string(TfsNetInterfaceTypeVPC))
		// For VPC, we need VpcId and SubnetId
		request.VpcId = common.StringPtr(reqInfo.VpcIID.SystemId)
		request.SubnetId = common.StringPtr(reqInfo.AccessSubnetList[0].SystemId) // 1개의 subnet만 허용
	}

	// Set protocol based on storage type and NFS version( NFS, CIFS, TURBO)
	if storageType == string(TfsStorageTypeStandardTurbo) || storageType == string(TfsStorageTypeHighPerformanceTurbo) {
		// Turbo series must use TURBO protocol
		request.Protocol = common.StringPtr(string(TfsProtocolTURBO))
	} else {
		// Standard/High-Performance can use NFS or CIFS
		if reqInfo.NFSVersion != "" {
			request.Protocol = common.StringPtr(string(TfsProtocolNFS))
		}
	}

	// Validate capacity against storage type limits
	if reqInfo.CapacityGB > 0 {
		// Determine capacity range based on storage type
		var capacityRange irs.CapacityGBRange
		switch storageType {
		case string(TfsStorageTypeStandardTurbo):
			capacityRange = metaInfo.CapacityGBOptions[string(TfsStorageTypeStandardTurbo)]
		case string(TfsStorageTypeHighPerformanceTurbo):
			capacityRange = metaInfo.CapacityGBOptions[string(TfsStorageTypeHighPerformanceTurbo)]
		case string(TfsStorageTypeStandard):
			capacityRange = metaInfo.CapacityGBOptions[string(TfsStorageTypeStandard)]
		case string(TfsStorageTypeHighPerformance):
			capacityRange = metaInfo.CapacityGBOptions[string(TfsStorageTypeHighPerformance)]
		default:
			return irs.FileSystemInfo{}, fmt.Errorf("unknown storage type: %s", storageType)
		}

		if reqInfo.CapacityGB < capacityRange.Min || reqInfo.CapacityGB > capacityRange.Max {
			return irs.FileSystemInfo{}, fmt.Errorf("capacity %dGB is out of range for storage type %s. Valid range: %dGB to %dGB",
				reqInfo.CapacityGB, storageType, capacityRange.Min, capacityRange.Max)
		}

		request.Capacity = common.Uint64Ptr(uint64(reqInfo.CapacityGB))
	} else {
		// Use minimum capacity for selected storage type
		var minCapacity int64
		switch storageType {
		case string(TfsStorageTypeStandardTurbo):
			minCapacity = metaInfo.CapacityGBOptions[string(TfsStorageTypeStandardTurbo)].Min
		case string(TfsStorageTypeHighPerformanceTurbo):
			minCapacity = metaInfo.CapacityGBOptions[string(TfsStorageTypeHighPerformanceTurbo)].Min
		case string(TfsStorageTypeStandard):
			minCapacity = metaInfo.CapacityGBOptions[string(TfsStorageTypeStandard)].Min
		case string(TfsStorageTypeHighPerformance):
			minCapacity = metaInfo.CapacityGBOptions[string(TfsStorageTypeHighPerformance)].Min
		default:
			return irs.FileSystemInfo{}, fmt.Errorf("unknown storage type: %s", storageType)
		}
		request.Capacity = common.Uint64Ptr(uint64(minCapacity))
	}

	// Set permission group ID (PGroupId)

	request.PGroupId = common.StringPtr(pGroupId)
	//request.ClientToken = common.StringPtr(fmt.Sprintf("cb-%d", time.Now().Unix()))// valid for 2 hours

	// Tag 추가
	var tags []*cfs.TagInfo
	for _, tag := range reqInfo.TagList {
		tags = append(tags, &cfs.TagInfo{
			TagKey:   common.StringPtr(tag.Key),
			TagValue: common.StringPtr(tag.Value),
		})
	}
	request.ResourceTags = tags

	response, err := fsHandler.CFSClient.CreateCfsFileSystem(request)
	if err != nil {
		cblogger.Error(err)
		return irs.FileSystemInfo{}, err
	}

	//////// 생성 후 조회 //////////////////////////
	// Wait for file system to be available
	fileSystemId := *response.Response.FileSystemId
	availableFileSystem, err := fsHandler.waitForFileSystemAvailable(fileSystemId) //wait 안에 describeCfsFileSystem 호출
	if err != nil {
		return irs.FileSystemInfo{}, err
	}

	// 이미 조회된 파일시스템 정보를 CB-Spider 형식으로 변환하여 반환
	resultFileSystemInfo, err := fsHandler.convertToFileSystemInfo(availableFileSystem)
	if err != nil {
		return resultFileSystemInfo, err
	}

	////////////////
	// FileSystemType FileSystemType `json:"FileSystemType,omitempty" example:"RegionType"` // RegionType, ZoneType; CSP default if omitted
	// NFSVersion     string         `json:"NFSVersion" validate:"required" example:"4.1"`  // NFS protocol version, e.g., "3.0", "4.1"; CSP default if omitted
	// CapacityGB     int64          `json:"CapacityGB,omitempty" example:"1024"`           // Capacity in GB, -1 when not applicable. Ignored if CSP unsupported.; CSP default if omitted
	// // Each key/value must match one of the PerformanceOptions provided by the cloud driver for the selected file system type.
	// PerformanceInfo map[string]string `json:"PerformanceInfo,omitempty"` // Performance options, e.g., {"Tier": "STANDARD"}, {"ThroughputMode": "provisioned", "Throughput": "128"}; CSP default if omitted
	// //**************************************************************************************************

	// // only for response, not for request
	// Status          FileSystemStatus  `json:"Status" validate:"required" example:"Available"`
	// UsedSizeGB      int64             `json:"UsedSizeGB" validate:"required" example:"256"` // Current used size in GB.
	// MountTargetList []MountTargetInfo `json:"MountTargetList,omitempty"`

	// CreatedTime  time.Time  `json:"CreatedTime" validate:"required"`
	// KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"` // Additional key-value pairs associated with this File System

	/////////////////

	return resultFileSystemInfo, nil
}

// ListFileSystem returns list of all file systems
func (fsHandler *TencentFileSystemHandler) ListFileSystem() ([]*irs.FileSystemInfo, error) {
	cblogger.Info("Start ListFileSystem()")

	// 공통 함수를 사용하여 모든 파일시스템 조회 (필터 없음)
	fileSystems, err := fsHandler.describeCfsFileSystem(nil, nil, nil)
	if err != nil {
		return nil, err
	}

	///// Response 예시
	// {
	// 	"Response": {
	// 	  "FileSystems": [
	// 		{
	// 		  "AppId": 111111,
	// 		  "AutoScaleUpRule": null,
	// 		  "AutoSnapshotPolicyId": "",
	// 		  "BandwidthLimit": 100,
	// 		  "BandwidthResourcePkg": "",
	// 		  "Capacity": 163840,
	// 		  "CreationTime": "2025-08-11 09:39:57",
	// 		  "CreationToken": "cbspider-file-system01",
	// 		  "Encrypted": false,
	// 		  "FileSystemId": "cfs-l7gz8wif",
	// 		  "FsName": "cbspider-file-system01",
	// 		  "KmsKeyId": "",
	// 		  "LifeCycleState": "available",
	// 		  "PGroup": {
	// 			"Name": "default",
	// 			"PGroupId": "pgroupbasic"
	// 		  },
	// 		  "Protocol": "NFS",
	// 		  "SizeByte": 1048576,
	// 		  "SizeLimit": 0,
	// 		  "SnapStatus": "normal",
	// 		  "StorageResourcePkg": "",
	// 		  "StorageType": "SD",
	// 		  "Tags": [],
	// 		  "TieringDetail": {
	// 			"SecondaryTieringSizeInBytes": 0,
	// 			"TieringSizeInBytes": 0
	// 		  },
	// 		  "TieringState": "NotAvailable",
	// 		  "Version": "v1.5",
	// 		  "Zone": "ap-seoul-1",
	// 		  "ZoneId": 180001
	// 		}
	// 	  ],
	// 	  "RequestId": "d489d0ae-9719-40d4-84a4-b5dab387be84",
	// 	  "TotalCount": 1
	// 	}
	//   }

	// 파일시스템들을 CB-Spider 형식으로 변환
	var allFileSystemList []*irs.FileSystemInfo
	for _, fs := range fileSystems {
		fileSystemInfo, err := fsHandler.convertToFileSystemInfo(fs)
		if err != nil {
			cblogger.Error(err)
			continue
		}
		allFileSystemList = append(allFileSystemList, &fileSystemInfo)
	}

	cblogger.Infof("Total file systems converted: %d", len(allFileSystemList))
	return allFileSystemList, nil
}

// GetFileSystem returns file system information by IID
func (fsHandler *TencentFileSystemHandler) GetFileSystem(iid irs.IID) (irs.FileSystemInfo, error) {
	cblogger.Info("Start GetFileSystem()")

	// IID check
	if iid.SystemId == "" {
		return irs.FileSystemInfo{}, errors.New("file system ID is required")
	}

	// 공통 함수를 사용하여 특정 파일시스템 ID로 조회
	fileSystems, err := fsHandler.describeCfsFileSystem(&iid.SystemId, nil, nil)
	if err != nil {
		cblogger.Error(err)
		return irs.FileSystemInfo{}, err
	}

	if len(fileSystems) == 0 {
		return irs.FileSystemInfo{}, errors.New("file system not found")
	}

	return fsHandler.convertToFileSystemInfo(fileSystems[0])
}

// DeleteFileSystem deletes a file system
func (fsHandler *TencentFileSystemHandler) DeleteFileSystem(iid irs.IID) (bool, error) {
	cblogger.Info("Start DeleteFileSystem()")

	// IID check
	if iid.SystemId == "" {
		return false, errors.New("file system ID is required")
	}

	request := cfs.NewDeleteCfsFileSystemRequest()
	request.FileSystemId = common.StringPtr(iid.SystemId)

	_, err := fsHandler.CFSClient.DeleteCfsFileSystem(request)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	return true, nil
}

// AddAccessSubnet adds a subnet to the file system access list
func (fsHandler *TencentFileSystemHandler) AddAccessSubnet(iid irs.IID, subnetIID irs.IID) (irs.FileSystemInfo, error) {

	// Tencent filesystem 은 생성시 subnet을 지정하고 추가/제거, 수정이 불가하다.
	return irs.FileSystemInfo{}, errors.New("only a single network is supported")

	// cblogger.Info("Start AddAccessSubnet()")

	// // IID check
	// if iid.SystemId == "" {
	// 	return irs.FileSystemInfo{}, errors.New("file system ID is required")
	// }

	// if subnetIID.SystemId == "" {
	// 	return irs.FileSystemInfo{}, errors.New("subnet ID is required")
	// }

	// // subnetIID가 존재하는지 확인
	// subnetRequest := vpc.NewDescribeSubnetsRequest()
	// subnetRequest.SubnetIds = []*string{&subnetIID.SystemId}
	// subnetResponse, err := fsHandler.VPCClient.DescribeSubnets(subnetRequest)
	// if err != nil {
	// 	return irs.FileSystemInfo{}, err
	// }
	// if len(subnetResponse.Response.SubnetSet) == 0 {
	// 	return irs.FileSystemInfo{}, errors.New("subnet not found")
	// }

	// // In Tencent Cloud, file systems are created with a specific subnet
	// // Adding additional subnets requires creating mount targets
	// // For now, we'll return the current file system info
	// return fsHandler.GetFileSystem(iid)
}

// RemoveAccessSubnet removes a subnet from the file system access list
func (fsHandler *TencentFileSystemHandler) RemoveAccessSubnet(iid irs.IID, subnetIID irs.IID) (bool, error) {
	//cblogger.Info("Start RemoveAccessSubnet()")

	// Tencent filesystem 은 생성시 subnet을 지정하고 추가/제거, 수정이 불가하다.
	return false, errors.New("only a single network is supported")
}

// ListAccessSubnet returns list of subnets that can access the file system
func (fsHandler *TencentFileSystemHandler) ListAccessSubnet(iid irs.IID) ([]irs.IID, error) {
	cblogger.Info("Start ListAccessSubnet()")

	fileSystemInfo, err := fsHandler.GetFileSystem(iid)
	if err != nil {
		return nil, err
	}

	accessSubnetList := fileSystemInfo.AccessSubnetList

	// FileSystemInfo 에서 조회하는 방법이 없다.(create객체에는 있으나 response객체에는 없다. GetFileSystem에서 해결하자.)
	//return nil, errors.New("only a single network is supported")
	return accessSubnetList, nil
}

// ScheduleBackup schedules a backup for the file system
func (fsHandler *TencentFileSystemHandler) ScheduleBackup(reqInfo irs.FileSystemBackupInfo) (irs.FileSystemBackupInfo, error) {
	cblogger.Info("Start ScheduleBackup()")

	// Tencent Cloud CFS doesn't support scheduled backups through API
	// This would need to be implemented using Cloud Functions or other services
	return irs.FileSystemBackupInfo{}, errors.New("scheduled backups not supported by Tencent Cloud CFS")
}

// OnDemandBackup creates an on-demand backup
func (fsHandler *TencentFileSystemHandler) OnDemandBackup(fsIID irs.IID) (irs.FileSystemBackupInfo, error) {
	cblogger.Info("Start OnDemandBackup()")

	// Tencent Cloud CFS doesn't support on-demand backups through API
	return irs.FileSystemBackupInfo{}, errors.New("on-demand backups not supported by Tencent Cloud CFS")
}

// ListBackup returns list of backups for the file system
func (fsHandler *TencentFileSystemHandler) ListBackup(fsIID irs.IID) ([]irs.FileSystemBackupInfo, error) {
	cblogger.Info("Start ListBackup()")

	// Tencent Cloud CFS doesn't support backups through API
	return []irs.FileSystemBackupInfo{}, nil
}

// GetBackup returns backup information
func (fsHandler *TencentFileSystemHandler) GetBackup(fsIID irs.IID, backupID string) (irs.FileSystemBackupInfo, error) {
	cblogger.Info("Start GetBackup()")

	// Tencent Cloud CFS doesn't support backups through API
	return irs.FileSystemBackupInfo{}, errors.New("backups not supported by Tencent Cloud CFS")
}

// DeleteBackup deletes a backup
func (fsHandler *TencentFileSystemHandler) DeleteBackup(fsIID irs.IID, backupID string) (bool, error) {
	cblogger.Info("Start DeleteBackup()")

	// Tencent Cloud CFS doesn't support backups through API
	return false, errors.New("backups not supported by Tencent Cloud CFS")
}

// Helper functions

// describeCfsFileSystem retrieves file systems using pagination with optional filters
func (fsHandler *TencentFileSystemHandler) describeCfsFileSystem(fileSystemId *string, vpcId *string, subnetId *string) ([]*cfs.FileSystemInfo, error) {
	// Check if current region is supported by CFS API
	if err := fsHandler.checkSupportedRegion(); err != nil {
		cblogger.Error(err)
		return nil, err
	}

	var fileSystems []*cfs.FileSystemInfo
	offset := uint64(0)
	limit := uint64(100) // 한 번에 최대 100개씩 조회

	for {
		request := cfs.NewDescribeCfsFileSystemsRequest()
		// Note: Tencent CFS API는 클라이언트 생성 시 Region이 설정되어 있음
		// CFS API는 특정 region들만 지원: ap-bangkok, ap-beijing, ap-chengdu, ap-chongqing,
		// ap-guangzhou, ap-hongkong, ap-jakarta, ap-mumbai, ap-nanjing, ap-seoul,
		// ap-shanghai, ap-shanghai-fsi, ap-shenzhen-fsi, ap-singapore, ap-tokyo,
		// eu-frankfurt, na-ashburn, na-siliconvalley
		request.Offset = common.Uint64Ptr(offset)
		request.Limit = common.Uint64Ptr(limit)

		// 선택적 필터 적용
		if fileSystemId != nil {
			request.FileSystemId = fileSystemId
		}
		if vpcId != nil {
			request.VpcId = vpcId
		}
		if subnetId != nil {
			request.SubnetId = subnetId
		}

		response, err := fsHandler.CFSClient.DescribeCfsFileSystems(request)
		if err != nil {
			cblogger.Error(err)
			return nil, err
		}

		// 응답이 없거나 파일시스템이 없으면 종료
		if response.Response.FileSystems == nil || len(response.Response.FileSystems) == 0 {
			break
		}

		// 현재 페이지의 파일시스템들을 추가
		fileSystems = append(fileSystems, response.Response.FileSystems...)

		// 특정 파일시스템 ID로 조회한 경우는 페이징 불필요
		if fileSystemId != nil {
			break
		}

		// 다음 페이지가 있는지 확인
		if len(response.Response.FileSystems) < int(limit) {
			break
		}

		// 다음 페이지로 이동
		offset += limit
	}

	cblogger.Infof("Total file systems retrieved: %d", len(fileSystems))
	return fileSystems, nil
}

// fileSystemExist checks if a file system with the given ID exists
func (fsHandler *TencentFileSystemHandler) fileSystemExist(fsIID irs.IID) (bool, error) {
	// 공통 함수를 사용하여 특정 파일시스템 ID로 조회
	fileSystems, err := fsHandler.describeCfsFileSystem(&fsIID.SystemId, nil, nil)
	if err != nil {
		return false, err
	}

	return len(fileSystems) > 0, nil
}

// waitForFileSystemAvailable waits for the file system to become available
func (fsHandler *TencentFileSystemHandler) waitForFileSystemAvailable(fileSystemId string) (*cfs.FileSystemInfo, error) {
	cblogger.Info("Waiting for file system to become available...")

	maxRetryCnt := 60
	for i := 0; i < maxRetryCnt; i++ {
		// 공통 함수를 사용하여 특정 파일시스템 ID로 조회
		fileSystems, err := fsHandler.describeCfsFileSystem(&fileSystemId, nil, nil)
		if err != nil {
			cblogger.Error(err)
			time.Sleep(10 * time.Second)
			continue
		}

		if len(fileSystems) > 0 {
			status := *fileSystems[0].LifeCycleState
			cblogger.Infof("File system status: %s", status)

			if status == "available" {
				return fileSystems[0], nil
			}
		}

		time.Sleep(10 * time.Second)
	}

	return nil, errors.New("timeout waiting for file system to become available")
}

// checkSupportedRegion checks if the current region is supported by CFS API
func (fsHandler *TencentFileSystemHandler) checkSupportedRegion() error {
	supportedRegions := []string{
		"ap-bangkok", "ap-beijing", "ap-chengdu", "ap-chongqing", "ap-guangzhou",
		"ap-hongkong", "ap-jakarta", "ap-mumbai", "ap-nanjing", "ap-seoul",
		"ap-shanghai", "ap-shanghai-fsi", "ap-shenzhen-fsi", "ap-singapore",
		"ap-tokyo", "eu-frankfurt", "na-ashburn", "na-siliconvalley",
	}

	currentRegion := fsHandler.Region.Region
	for _, region := range supportedRegions {
		if region == currentRegion {
			return nil
		}
	}

	return fmt.Errorf("CFS API does not support region: %s. Supported regions: %v", currentRegion, supportedRegions)
}

// convertToFileSystemInfo converts Tencent Cloud CFS file system to CB-Spider format
func (fsHandler *TencentFileSystemHandler) convertToFileSystemInfo(fs *cfs.FileSystemInfo) (irs.FileSystemInfo, error) {

	// Parse capacity from Capacity field (in GB)
	capacityGB := int64(-1)
	if fs.Capacity != nil {
		// Capacity is in GB, convert to int64
		capacityGB = int64(*fs.Capacity)
	}

	// Parse NFS version from protocol
	nfsVersion := "4.0" // 기본값
	if fs.Protocol != nil {
		protocol := *fs.Protocol
		// Protocol이 "NFS"인 경우 기본값 4.0 사용
		// 만약 "NFS3", "NFS4" 등으로 명시되어 있다면 해당 버전 사용
		if strings.HasPrefix(protocol, "NFS") {
			version := strings.TrimPrefix(protocol, "NFS")
			if version != "" {
				nfsVersion = version
			}
		}
	}

	// Convert status
	status := fsHandler.convertStatus(*fs.LifeCycleState)

	// Parse creation time from CreationTime field
	createdTime := time.Now() // 기본값
	if fs.CreationTime != nil {
		// CreationTime format: "2025-08-11 09:39:57"
		parsedTime, err := time.Parse("2006-01-02 15:04:05", *fs.CreationTime)
		if err == nil {
			createdTime = parsedTime
		} else {
			cblogger.Warnf("Failed to parse creation time: %s, using current time", *fs.CreationTime)
		}
	}

	// Parse used size from SizeByte field (if available)
	usedSizeGB := int64(0) // 기본값
	if fs.SizeByte != nil {
		// SizeByte is in bytes, convert to GB
		usedSizeGB = int64(*fs.SizeByte) / (1024 * 1024 * 1024)
	}

	///////////////////// 마운트 타겟 정보 설정 ///////////////////////
	// DescribeMountTargets API를 사용하여 실제 마운트 타겟 정보 가져오기
	mountTargets, err := fsHandler.getMountTargets(*fs.FileSystemId)
	if err != nil {
		cblogger.Errorf("Failed to get mount targets: %v", err)
	}
	mountTargetList, vpcID, accessSubnetList := fsHandler.convertMountTargetsToInfo(mountTargets, nfsVersion)

	///////////////////// PerformanceInfo 추가 - 성능 관련 정보 ///////////////////////
	performanceInfo := make(map[string]string)

	// StorageType 매핑 및 상세 성능 정보 설정
	if fs.StorageType != nil {
		storageType := TfsStorageType(*fs.StorageType)
		if spec, exists := performanceSpecs[storageType]; exists {
			// 상수에서 정의된 성능 정보를 사용
			performanceInfo["Tier"] = spec.Tier
			performanceInfo["MaxSystemCapacity"] = spec.MaxSystemCapacity
			if spec.MinSystemCapacity != "" {
				performanceInfo["MinSystemCapacity"] = spec.MinSystemCapacity
			}
			performanceInfo["MaxSystemBandwidth"] = spec.MaxSystemBandwidth
			performanceInfo["MaxSystemFiles"] = spec.MaxSystemFiles
			performanceInfo["MaxSystemDirectories"] = spec.MaxSystemDirectories
			performanceInfo["MaxFilenameLength"] = spec.MaxFilenameLength
			performanceInfo["MaxAbsolutePathLength"] = spec.MaxAbsolutePathLength
			performanceInfo["MaxDirectoryLevels"] = spec.MaxDirectoryLevels
			performanceInfo["MaxFilesPerDirectory"] = spec.MaxFilesPerDirectory
			performanceInfo["MaxConcurrentlyOpenedFiles"] = spec.MaxConcurrentlyOpenedFiles
			performanceInfo["MaxLocksPerFile"] = spec.MaxLocksPerFile
			performanceInfo["MaxClients"] = spec.MaxClients
			performanceInfo["MaxBandwidthPerClient"] = spec.MaxBandwidthPerClient
			performanceInfo["MaxMountedFileSystemsPerClient"] = spec.MaxMountedFileSystemsPerClient
			performanceInfo["Billing"] = spec.Billing
			performanceInfo["SupportedProtocol"] = spec.SupportedProtocol
			performanceInfo["SupportedOS"] = spec.SupportedOS
			performanceInfo["Notes"] = spec.Notes
		} else {
			// 기본값으로 STANDARD 사용
			defaultSpec := performanceSpecs[TfsStorageTypeStandard]
			performanceInfo["Tier"] = defaultSpec.Tier
			performanceInfo["MaxSystemCapacity"] = defaultSpec.MaxSystemCapacity
			performanceInfo["MaxSystemBandwidth"] = defaultSpec.MaxSystemBandwidth
			performanceInfo["MaxSystemFiles"] = defaultSpec.MaxSystemFiles
			performanceInfo["MaxSystemDirectories"] = defaultSpec.MaxSystemDirectories
			performanceInfo["MaxFilenameLength"] = defaultSpec.MaxFilenameLength
			performanceInfo["MaxAbsolutePathLength"] = defaultSpec.MaxAbsolutePathLength
			performanceInfo["MaxDirectoryLevels"] = defaultSpec.MaxDirectoryLevels
			performanceInfo["MaxFilesPerDirectory"] = defaultSpec.MaxFilesPerDirectory
			performanceInfo["MaxConcurrentlyOpenedFiles"] = defaultSpec.MaxConcurrentlyOpenedFiles
			performanceInfo["MaxLocksPerFile"] = defaultSpec.MaxLocksPerFile
			performanceInfo["MaxClients"] = defaultSpec.MaxClients
			performanceInfo["MaxBandwidthPerClient"] = defaultSpec.MaxBandwidthPerClient
			performanceInfo["MaxMountedFileSystemsPerClient"] = defaultSpec.MaxMountedFileSystemsPerClient
			performanceInfo["Billing"] = defaultSpec.Billing
			performanceInfo["SupportedProtocol"] = defaultSpec.SupportedProtocol
			performanceInfo["SupportedOS"] = defaultSpec.SupportedOS
			performanceInfo["Notes"] = defaultSpec.Notes
		}
	}

	// BandwidthLimit 설정
	if fs.BandwidthLimit != nil {
		performanceInfo["BandwidthLimit"] = fmt.Sprintf("%d", *fs.BandwidthLimit)
	}

	// TieringState 설정
	if fs.TieringState != nil {
		performanceInfo["TieringState"] = *fs.TieringState
	}

	// TieringDetail 설정
	if fs.TieringDetail != nil {
		if fs.TieringDetail.TieringSizeInBytes != nil {
			performanceInfo["TieringSizeInBytes"] = fmt.Sprintf("%d", *fs.TieringDetail.TieringSizeInBytes)
		}
		if fs.TieringDetail.SecondaryTieringSizeInBytes != nil {
			performanceInfo["SecondaryTieringSizeInBytes"] = fmt.Sprintf("%d", *fs.TieringDetail.SecondaryTieringSizeInBytes)
		}
	}

	// SizeLimit 설정
	if fs.SizeLimit != nil {
		performanceInfo["SizeLimit"] = fmt.Sprintf("%d", *fs.SizeLimit)
	}

	// SnapStatus 설정
	if fs.SnapStatus != nil {
		performanceInfo["SnapStatus"] = *fs.SnapStatus
	}

	// AutoSnapshotPolicyId 설정
	if fs.AutoSnapshotPolicyId != nil && *fs.AutoSnapshotPolicyId != "" {
		performanceInfo["AutoSnapshotPolicyId"] = *fs.AutoSnapshotPolicyId
	}

	// Encrypted 설정
	if fs.Encrypted != nil {
		performanceInfo["Encrypted"] = fmt.Sprintf("%t", *fs.Encrypted)
	}

	// KmsKeyId 설정 (암호화된 경우)
	if fs.KmsKeyId != nil && *fs.KmsKeyId != "" {
		performanceInfo["KmsKeyId"] = *fs.KmsKeyId
	}

	///////////////////// Tag 추가 ///////////////////////
	var tagList []irs.KeyValue
	if fs.Tags != nil {
		for _, tag := range fs.Tags {
			tagList = append(tagList, irs.KeyValue{
				Key:   *tag.TagKey,
				Value: *tag.TagValue,
			})
		}
	}

	// KeyValueList 추가
	var keyValueList []irs.KeyValue
	pgroup := fs.PGroup
	if pgroup != nil {
		keyValueList = append(keyValueList, irs.KeyValue{
			Key:   "PGroupId",
			Value: *pgroup.PGroupId,
		})
	}

	fileSystemInfo := irs.FileSystemInfo{
		IId: irs.IID{
			NameId:   *fs.FsName,
			SystemId: *fs.FileSystemId,
		},
		Region:           fsHandler.Region.Region,
		Zone:             *fs.Zone, // JSON에서 Zone 필드 사용
		VpcIID:           irs.IID{SystemId: vpcID},
		AccessSubnetList: accessSubnetList,
		FileSystemType:   irs.ZoneType,
		NFSVersion:       nfsVersion,
		CapacityGB:       capacityGB,
		Status:           status,
		UsedSizeGB:       usedSizeGB,
		CreatedTime:      createdTime,
		MountTargetList:  mountTargetList,
		TagList:          tagList,
		PerformanceInfo:  performanceInfo,
		KeyValueList:     keyValueList,
	}

	return fileSystemInfo, nil
}

// convertMountTargetsToInfo converts Tencent Cloud CFS mount targets to CB-Spider MountTargetInfo format
// Returns MountTargetInfo list and extracted VPC/Subnet information
func (fsHandler *TencentFileSystemHandler) convertMountTargetsToInfo(mountTargets []*cfs.MountInfo, nfsVersion string) ([]irs.MountTargetInfo, string, []irs.IID) {
	var mountTargetInfoList []irs.MountTargetInfo
	var vpcID string
	var accessSubnetList []irs.IID

	if mountTargets == nil || len(mountTargets) == 0 {
		cblogger.Infof("No mount targets provided for conversion")
		return mountTargetInfoList, vpcID, accessSubnetList
	}

	// 한 번의 순회로 모든 작업 처리
	for i, mountTarget := range mountTargets {
		// 첫 번째 마운트 타겟에서 VPC ID 추출
		if i == 0 {
			vpcID = *mountTarget.VpcId
			// 서브넷 정보를 AccessSubnetList에 추가
			accessSubnetList = append(accessSubnetList, irs.IID{
				SystemId: *mountTarget.SubnetId,
			})
		}

		// 마운트 타겟 정보를 MountTargetInfo로 변환
		mountTargetInfo := irs.MountTargetInfo{
			SubnetIID:           irs.IID{SystemId: *mountTarget.SubnetId},
			Endpoint:            *mountTarget.IpAddress,
			MountCommandExample: fsHandler.generateMountCommandExample(*mountTarget.IpAddress, nfsVersion),
			// KeyValueList에 추가 마운트 타겟 정보 저장
			KeyValueList: []irs.KeyValue{
				{Key: "MountTargetId", Value: *mountTarget.MountTargetId},
				{Key: "FileSystemId", Value: *mountTarget.FileSystemId},
				{Key: "FSID", Value: *mountTarget.FSID},
				{Key: "LifeCycleState", Value: *mountTarget.LifeCycleState},
				{Key: "NetworkInterface", Value: *mountTarget.NetworkInterface},
				{Key: "VpcId", Value: *mountTarget.VpcId},
				{Key: "VpcName", Value: *mountTarget.VpcName},
				{Key: "SubnetName", Value: *mountTarget.SubnetName},
				{Key: "CidrBlock", Value: *mountTarget.CidrBlock},
			},
		}

		// CCN ID가 유효한 경우에만 추가 (CCN은 CFS Turbo에서 사용)
		if mountTarget.CcnID != nil && *mountTarget.CcnID != "-" {
			mountTargetInfo.KeyValueList = append(mountTargetInfo.KeyValueList, irs.KeyValue{
				Key: "CcnID", Value: *mountTarget.CcnID,
			})
		}

		mountTargetInfoList = append(mountTargetInfoList, mountTargetInfo)
	}

	cblogger.Infof("Successfully converted %d mount targets to MountTargetInfo", len(mountTargetInfoList))
	return mountTargetInfoList, vpcID, accessSubnetList
}

// convertFileSystemClientsToMountInfo converts Tencent Cloud CFS file system clients to CB-Spider MountTargetInfo format
func (fsHandler *TencentFileSystemHandler) convertFileSystemClientsToMountInfo(clients []*cfs.FileSystemClient, nfsVersion string) []irs.MountTargetInfo {
	var mountTargetInfoList []irs.MountTargetInfo

	if clients == nil || len(clients) == 0 {
		cblogger.Infof("No clients provided for conversion")
		return mountTargetInfoList
	}

	// 각 클라이언트 정보를 MountTargetInfo로 변환
	for _, client := range clients {
		mountTargetInfo := irs.MountTargetInfo{
			// CFS 클라이언트 정보를 기반으로 설정
			Endpoint:            *client.CfsVip, // CFS VIP 주소 사용
			MountCommandExample: fsHandler.generateMountCommandExample(*client.CfsVip, nfsVersion),
			// KeyValueList에 추가 클라이언트 정보 저장
			KeyValueList: []irs.KeyValue{
				{Key: "ClientIp", Value: *client.ClientIp},
				{Key: "MountDirectory", Value: *client.MountDirectory},
				{Key: "VpcId", Value: *client.VpcId},
				{Key: "Zone", Value: *client.Zone},
				{Key: "ZoneName", Value: *client.ZoneName},
			},
		}
		mountTargetInfoList = append(mountTargetInfoList, mountTargetInfo)
	}

	cblogger.Infof("Successfully converted %d CFS clients to MountTargetInfo", len(mountTargetInfoList))
	return mountTargetInfoList
}

// generateMountCommandExample generates mount command examples for specific NFS version and platform
func (fsHandler *TencentFileSystemHandler) generateMountCommandExample(ipAddress string, nfsVersion string) string {
	// Linux 명령어를 저장할 문자열 초기화
	linuxCommands := ""

	// Windows 마운트 명령어는 NFS 버전과 관계없이 동일
	windowsMountCommand := `Mount under Windows:
mount -o nolock mtype=hard ` + ipAddress + `:/1age0tne x:`

	linuxCommandsNFS3 := `Mount under Linux (NFS 3.0):
sudo mount -t nfs -o vers=3,nolock,proto=tcp,noresvport ` + ipAddress + `:/1age0tne/ /localfolder

Mount subdirectory (NFS 3.0):
sudo mount -t nfs -o vers=3,nolock,proto=tcp,noresvport ` + ipAddress + `:/1age0tne/subfolder /localfolder`

	linuxCommandsNFS4 := `Mount under Linux (NFS 4.0):
sudo mount -t nfs -o vers=4.0,noresvport ` + ipAddress + `:/ /localfolder

Mount subdirectory (NFS 4.0):
sudo mount -t nfs -o vers=4.0,noresvport ` + ipAddress + `:/subfolder /localfolder

NFS 4.0 Credential Mounting:
sudo mount -t cfs -o vers=4.0,noresvport,tls,cert=/path/to/pem ` + ipAddress + `:/ /localfolder`

	switch nfsVersion {
	case "3.0":
		linuxCommands = linuxCommandsNFS3
	case "4.0":
		linuxCommands = linuxCommandsNFS4
	default:
		linuxCommands = linuxCommandsNFS3 + linuxCommandsNFS4
	}

	// Note 정보 추가
	noteInfo := `

Note:
    "localfolder" refers to the local directory you create, and "subfolder" is the subdirectory created in the CFS instance.
    You are advised to mount using the NFSv3 protocol for better performance. If your application requires file locking, that is, multiple CVM instances need to edit one single file, use NFSv4.`

	// Linux 명령어, Windows 명령어, Note 정보를 조합하여 반환
	return linuxCommands + windowsMountCommand + noteInfo
}

// getCfsFileSystemClients gets CFS file system clients using DescribeCfsFileSystemClients API
func (fsHandler *TencentFileSystemHandler) getCfsFileSystemClients(fileSystemId string) ([]*cfs.FileSystemClient, error) {
	request := cfs.NewDescribeCfsFileSystemClientsRequest()
	request.FileSystemId = common.StringPtr(fileSystemId)

	response, err := fsHandler.CFSClient.DescribeCfsFileSystemClients(request)
	if err != nil {
		cblogger.Errorf("Failed to describe CFS file system clients: %v", err)
		return nil, err
	}

	if response.Response.ClientList == nil {
		return []*cfs.FileSystemClient{}, nil
	}

	return response.Response.ClientList, nil
}

// getMountTargets gets mount targets for a file system (kept for backward compatibility)
func (fsHandler *TencentFileSystemHandler) getMountTargets(fileSystemId string) ([]*cfs.MountInfo, error) {
	request := cfs.NewDescribeMountTargetsRequest()
	request.FileSystemId = common.StringPtr(fileSystemId)

	response, err := fsHandler.CFSClient.DescribeMountTargets(request)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	if response.Response.MountTargets == nil {
		return []*cfs.MountInfo{}, nil
	}

	return response.Response.MountTargets, nil
}

// convertStatus converts Tencent Cloud status to CB-Spider status
func (fsHandler *TencentFileSystemHandler) convertStatus(status string) irs.FileSystemStatus {
	switch status {
	case string(TfsLifeCycleStateCreating):
		return irs.FileSystemCreating
	case string(TfsLifeCycleStateAvailable):
		return irs.FileSystemAvailable
	case string(TfsLifeCycleStateDeleting):
		return irs.FileSystemDeleting
	case string(TfsLifeCycleStateCreateFailed):
		return irs.FileSystemError
	default:
		return irs.FileSystemError
	}
}
