package resources

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	filestore "cloud.google.com/go/filestore/apiv1"
	filestorepb "cloud.google.com/go/filestore/apiv1/filestorepb"

	monitoring "cloud.google.com/go/monitoring/apiv3/v2"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	compute "google.golang.org/api/compute/v1"
	container "google.golang.org/api/container/v1"
)

type GCPFileSystemHandler struct {
	Region           idrv.RegionInfo
	Ctx              context.Context
	Client           *compute.Service
	FilestoreClient  *filestore.CloudFilestoreManagerClient
	ContainerClient  *container.Service
	MonitoringClient *monitoring.MetricClient
	Credential       idrv.CredentialInfo
}

type GCPNFSVersion string

const (
	NFS_V3   GCPNFSVersion = "3.0"
	NFS_V4_1 GCPNFSVersion = "4.1"
)

// FileSystem Meta Info
// 지원하는 타입은 현재 RegionType, ZoneType, DefaultType이 있음. 그중 DefaultType은 ZoneType으로 처리함.
func (fsHandler *GCPFileSystemHandler) GetMetaInfo() (irs.FileSystemMetaInfo, error) {
	metaInfo := irs.FileSystemMetaInfo{
		SupportsFileSystemType: map[irs.FileSystemType]bool{
			irs.RegionType: true,
			irs.ZoneType:   true,
		},
		SupportsVPC: map[irs.RSType]bool{
			irs.VPC: true,
		},
		SupportsNFSVersion: []string{"3.0", "4.1"},
		SupportsCapacity:   true,
		CapacityGBOptions: map[string]irs.CapacityGBRange{
			"BASIC_HDD":                     {Min: 1024, Max: 65433},   // 1~63.9T (65433.6GB)
			"BASIC_SSD":                     {Min: 2560, Max: 65433},   // 2.5~63.9T (65433.6GB)
			"ZONAL_BETWEEN_1_TO_9.75TiB":    {Min: 1024, Max: 9984},    // 1~9.75TiB, increse 0.25TiB
			"ZONAL_BETWEEN_10_TO_100TiB":    {Min: 10240, Max: 102400}, // increse 2.5TiB
			"REGIONAL_BETWEEN_1_TO_9.75TiB": {Min: 1024, Max: 9984},    // increse 0.25TiB
			"REGIONAL_BETWEEN_10_TO_100TiB": {Min: 10240, Max: 102400}, // increse 2.5TiB
		},
		PerformanceOptions: map[string][]string{
			"BASIC_HDD":                     {"DEFAULT"},                                                                           // Basic은 선택불가
			"BASIC_SSD":                     {"DEFAULT"},                                                                           // Basic은 선택불가
			"ZONAL_BETWEEN_1_TO_9.75TiB":    {"Default_fixed_9000", "Custom_Default_12000", "Custom_Min_4000", "Custom_Max_17000"}, // default는 9200 인데, 9000으로 변경
			"ZONAL_BETWEEN_10_TO_100TiB":    {"Default_fixed_92000", "Custom_Default_75000", "Custom_Min_30000", "Custom_Max_75000"},
			"REGIONAL_BETWEEN_1_TO_9.75TiB": {"Default_fixed_12000", "Custom_Default_12000", "Custom_Min_4000", "Custom_Max_17000"},
			"REGIONAL_BETWEEN_10_TO_100TiB": {"Default_fixed_92000", "Custom_Default_75000", "Custom_Min_30000", "Custom_Max_75000"},
			"IS_CUSTOM_PERFORMANCE_MODE":    {"true", "false"},
		},
		// Performance 에 설정을 하지 않으면 Default_fixed 값으로 설정되고 CustomPerformanceSupported 는 false
		// CustomPerformanceSupported 가 true 이면 PerformanceConfig 에 설정된 값으로 설정된다.
		//   이때 Custom의 Min, Max 값 않에서 설정해야 하는데 기본값이 Custom_Default 이다.
	}

	// 	//CapacityOption = Tier at GCP
	// TIER_UNSPECIFIED 	Not set.
	// STANDARD 	STANDARD tier. BASIC_HDD is the preferred term for this tier.
	// PREMIUM 	PREMIUM tier. BASIC_SSD is the preferred term for this tier.
	// BASIC_HDD 	BASIC instances offer a maximum capacity of 63.9 TB. BASIC_HDD is an alias for STANDARD Tier, offering economical performance backed by HDD.
	// BASIC_SSD 	BASIC instances offer a maximum capacity of 63.9 TB. BASIC_SSD is an alias for PREMIUM Tier, and offers improved performance backed by SSD.
	// HIGH_SCALE_SSD 	HIGH_SCALE instances offer expanded capacity and performance scaling capabilities.
	// ENTERPRISE 	ENTERPRISE instances offer the features and availability needed for mission-critical workloads.
	// ZONAL 	ZONAL instances offer expanded capacity and performance scaling capabilities.
	// REGIONAL 	REGIONAL instances offer the features and availability needed for mission-critical workloads.

	// SupportsFileSystemType map[FileSystemType]bool    `json:"SupportsFileSystemType"`       // e.g., {"RegionType": true, "ZoneType": true, "RegionZoneBasedType": true, ...}
	// SupportsVPC            map[RSType]bool            `json:"SupportsVPC"`                  // e.g., {"VPC": true} or {"VPC": false} (if not supported)
	// SupportsNFSVersion     []string                   `json:"SupportsNFSVersion"`           // e.g., ["3.0", "4.1"]
	// SupportsCapacity       bool                       `json:"SupportsCapacity"`             // true if capacity can be specified
	// CapacityGBOptions      map[string]CapacityGBRange `json:"CapacityGBOptions,omitempty"`  // Capacity ranges per file system option (valid only if SupportsCapacity is true). e.g., GCP Filestore: {"Basic": {Min: 1024, Max: 65229}, "Zonal": {Min: 1024, Max: 102400}, "Regional": {Min: 1024, Max: 102400}}
	// PerformanceOptions     map[string][]string        `json:"PerformanceOptions,omitempty"` // Available performance settings per file system option. e.g., {"Basic": ["STANDARD"], "Zonal": ["HIGH_SCALE", "EXTREME"]}

	return metaInfo, nil
}

// List all file system IDs, not detailed info
// connection의 Region과 zone들에 대한 결과 return
func (fsHandler *GCPFileSystemHandler) ListIID() ([]*irs.IID, error) {

	instances, err := ListFilestoreInstancesByRegionAndZones(fsHandler.FilestoreClient, fsHandler.Client, fsHandler.Credential, fsHandler.Region, fsHandler.Ctx)
	if err != nil {
		cblogger.Error("error while listing file system instances")
		return nil, err
	}

	fsIIDList := make([]*irs.IID, 0)
	for _, instance := range instances {
		cblogger.Debug("instance", instance.Name)
		// "projects/%s/locations/%s/instances/%s" 이 형태에서 마지막 부분을 추출
		fileStoreName := ExtractFilestoreName(instance.Name)
		cblogger.Debug("fileStoreName", fileStoreName)
		fsIIDList = append(fsIIDList, &irs.IID{SystemId: fileStoreName})
	}
	return fsIIDList, nil
}

// File System Management
func (fsHandler *GCPFileSystemHandler) CreateFileSystem(reqInfo irs.FileSystemInfo) (irs.FileSystemInfo, error) {
	cblogger.Info("CreateFileSystem")
	cblogger.Info(fsHandler.Region)
	cblogger.Info(fsHandler.Credential.ProjectID)
	cblogger.Info("context region : ", fsHandler.Region.Region, ", zone : ", fsHandler.Region.Zone, ", targetZone : ", fsHandler.Region.TargetZone)
	regionZone := ""
	tier := ""
	cblogger.Info("Check tier : ", tier, ",region : ", fsHandler.Region.Region, ",zone : ", reqInfo.Zone)
	if reqInfo.Zone != "" {
		tier = "ZONAL"
		regionZone = reqInfo.Zone
	} else {
		tier = "REGIONAL"                    // 기본이 regional. 따로 설정하지 않으면 regional으로 처리
		regionZone = fsHandler.Region.Region // zone 없으면 region으로 처리
	}

	// 그 외 Tier 처리는 보류. 설정할 값들은 더 받아야 가능. HDD 여부 등.

	parent := FormatParentPath(fsHandler.Credential.ProjectID, regionZone)
	cblogger.Info("parent : ", parent)

	instanceName := reqInfo.IId.NameId
	// name check : Must be a match of regex "^(?:[a-z](?:[-a-z0-9]{0,61}[a-z0-9])?)$", (ID must start with a lowercase letter followed by up to 62 lowercase letters, numbers, or hyphens, and cannot end with a hyphen).
	if !regexp.MustCompile(`^(?:[a-z](?:[-a-z0-9]{0,61}[a-z0-9])?)$`).MatchString(instanceName) {
		return irs.FileSystemInfo{}, errors.New("instance name must start with a lowercase letter followed by up to 62 lowercase letters, numbers, or hyphens, and cannot end with a hyphen")
	}

	volumeName := reqInfo.IId.NameId
	// name check : must start with a letter followed by letters, numbers, or underscores, and cannot end with an underscore.
	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`).MatchString(volumeName) {
		return irs.FileSystemInfo{}, errors.New("file system name must start with a letter followed by letters, numbers, or underscores, and cannot end with an underscore")
	}

	// IId              IID    `json:"IId" validate:"required"` // {NameId, SystemId}
	// Region           string `json:"Region,omitempty" example:"us-east-1"`
	// Zone             string `json:"Zone,omitempty" example:"us-east-1a"`
	// VpcIID           IID    `json:"VpcIID" validate:"required"` // Owner VPC IID
	// AccessSubnetList []IID  `json:"AccessSubnetList,omitempty"` // List of subnets whose VMs can use this file system

	// Encryption     bool                 `json:"Encryption,omitempty" default:"false"` // Encryption enabled or not
	// BackupSchedule FileSystemBackupInfo `json:"BackupSchedule,omitempty"`             // Cron schedule for backups, default is "0 5 * * *" (Every day at 5 AM)
	// TagList        []KeyValue           `json:"TagList,omitempty" validate:"omitempty"`

	// //**************************************************************************************************
	// //** (1) Basic setup: If not set by the user, these fields use CSP default values.
	// //** (2) Advanced setup: If set by the user, these fields enable CSP-specific file system features.
	// //**************************************************************************************************
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

	cblogger.Info("Check capacity")
	// capacity는 Tier의 용량 이내에있어야 한다.
	capacity := reqInfo.CapacityGB
	metaInfo, err := fsHandler.GetMetaInfo()
	if err != nil {
		return irs.FileSystemInfo{}, err
	}

	if capacity > 0 {
		err = validateInstanceCapacityRange(metaInfo, tier, capacity)
		if err != nil {
			return irs.FileSystemInfo{}, err
		}
	}
	cblogger.Info("Check performance")
	customPerformanceSupported := false
	performanceConfig := &filestorepb.Instance_PerformanceConfig{}
	if reqInfo.PerformanceInfo != nil {
		if reqInfo.PerformanceInfo["IS_CUSTOM_PERFORMANCE_MODE"] == "true" {
			// 성능 정보가 있으면 설정
			if reqInfo.PerformanceInfo["Throughput"] != "" {
				performanceConfig, err = extractInstancePerformanceRange(metaInfo, tier, capacity, reqInfo.PerformanceInfo)
				if err != nil {
					return irs.FileSystemInfo{}, err
				}
				customPerformanceSupported = true
			}
		}
	}
	cblogger.Info("Check squash mode")
	squashMode := getSquashMode("")
	anonUid := 0 //An integer representing the anonymous user id with a default value of 65534. Anon_uid may only be set with squashMode of ROOT_SQUASH. An error will be returned if this field is specified for other squashMode settings.
	anonGid := 0 //An integer representing the anonymous group id with a default value of 65534. Anon_gid may only be set with squashMode of ROOT_SQUASH. An error will be returned if this field is specified for other squashMode settings.
	if squashMode == filestorepb.NfsExportOptions_ROOT_SQUASH {
		anonUid = 65534
		anonGid = 65534
	}

	// if reqInfo.Encryption {
	// 	encryptionConfig := &filestorepb.Instance_EncryptionConfig{
	// 		KmsKeyName: "",// kms key 이름
	// 	}
	// }

	//PerformanceConfig는 validateInstancePerformanceRange에서 설정됨

	// performanceLimit는 설정해야하나??
	// performanceLimits := &filestorepb.Instance_PerformanceLimits{
	// 	MaxIops:               0,
	// 	MaxReadIops:           0,
	// 	MaxWriteIops:          0,
	// 	MaxReadThroughputBps:  0,
	// 	MaxWriteThroughputBps: 0,
	// }

	cblogger.Info("Check network")
	// 	"selfLink": "https://www.googleapis.com/compute/v1/projects/csta-349809/global/networks/cmig",
	//   "selfLinkWithId": "https://www.googleapis.com/compute/v1/projects/csta-349809/global/networks/8773273090939442472",

	// 	"selfLink": "https://www.googleapis.com/compute/v1/projects/yhnoh-335705/global/networks/default",
	//   "selfLinkWithId": "https://www.googleapis.com/compute/v1/projects/yhnoh-335705/global/networks/6251234682456612724",

	// TODO: 파라미터 처리
	vpcID := reqInfo.VpcIID.SystemId
	network := fmt.Sprintf("projects/%s/global/networks/%s", fsHandler.Credential.ProjectID, vpcID)
	modes := []filestorepb.NetworkConfig_AddressMode{filestorepb.NetworkConfig_MODE_IPV4}

	cblogger.Info("Check file protocol")
	fileProtocol := filestorepb.Instance_FILE_PROTOCOL_UNSPECIFIED // default value는 "NFS_V3"
	if reqInfo.NFSVersion == string(NFS_V4_1) {
		fileProtocol = filestorepb.Instance_NFS_V4_1
	} else if reqInfo.NFSVersion == string(NFS_V3) {
		fileProtocol = filestorepb.Instance_NFS_V3
	} else {
		//return irs.FileSystemInfo{}, errors.New("invalid NFS version")
		fileProtocol = filestorepb.Instance_NFS_V3 // default value는 "NFS_V3"
	}

	// IpCidrRange: "10.0.0.0/29", // 선택 사항: 특정 IP 범위 지정
	cblogger.Info("Check file shares") // 1개만 가능.
	fileShares := []*filestorepb.FileShareConfig{
		{
			CapacityGb: capacity,
			Name:       volumeName,
			NfsExportOptions: []*filestorepb.NfsExportOptions{
				{
					IpRanges:   []string{"0.0.0.0/0"},
					AccessMode: filestorepb.NfsExportOptions_READ_WRITE,
					SquashMode: squashMode,
					AnonUid:    int64(anonUid),
					AnonGid:    int64(anonGid),
				},
			},
		},
	}

	cblogger.Info("Check network configs")
	networkConfigs := []*filestorepb.NetworkConfig{
		{
			Network: network,
			Modes:   modes,
		},
	}

	cblogger.Info("Check tags")
	tagList := make(map[string]string)
	for _, tag := range reqInfo.TagList {
		tagList[tag.Key] = tag.Value
	}

	cblogger.Info("set filestore instance")
	filestoreInstance := &filestorepb.Instance{
		//Name:              reqInfo.IId.NameId,// output only
		Tier:       getTier(tier),
		FileShares: fileShares,
		Networks:   networkConfigs,
		// PerformanceLimits: performanceLimits,
		Protocol: fileProtocol,
		//Description:       reqInfo.Description,
		Labels: tagList,
		//KmsKeyName:  reqInfo.KmsKeyName,
		//Replication: replication,
		//Tags: tagList,// tag는 특정 형태가 지정되어 있음. key-value 형태인 label로 처리함.
	}
	filestoreInstance.CustomPerformanceSupported = customPerformanceSupported
	if customPerformanceSupported {
		filestoreInstance.PerformanceConfig = performanceConfig
	}

	cblogger.Info("set create instance request")
	req := &filestorepb.CreateInstanceRequest{
		Parent:     parent,
		InstanceId: instanceName,
		Instance:   filestoreInstance,
	}

	cblogger.Info("create filestore instance", req)
	// complete 될 때까지 기다린다
	instance, err := CreateFilestoreInstance(fsHandler.FilestoreClient, fsHandler.Ctx, req)
	if err != nil {
		return irs.FileSystemInfo{}, err
	}

	// // vpc 정보조회
	// vNetworkHandler := GCPVPCHandler{
	// 	Client:     fsHandler.Client,
	// 	Region:     fsHandler.Region,
	// 	Ctx:        fsHandler.Ctx,
	// 	Credential: fsHandler.Credential,
	// }

	cblogger.Info("get filestore instance")
	instance, err2 := GetFilestoreInstance(fsHandler.FilestoreClient, fsHandler.Ctx, instance.Name)
	if err2 != nil {
		cblogger.Error(err2)
		return irs.FileSystemInfo{}, err2
	}

	// vpcInfo, errVnet := vNetworkHandler.GetVPC(irs.IID{SystemId: instance.Networks[0].Network})
	// cblogger.Debug(vpcInfo)
	// if errVnet != nil {
	// 	cblogger.Error(errVnet)
	// 	return irs.FileSystemInfo{}, errVnet
	// }
	// fsInfo := extractFileSystemInfo(instance, &vpcInfo)

	// 인스턴스 정보 조회
	instances := []*filestorepb.Instance{instance}
	// 파일 시스템 정보 조회 ( extract 조건이 모두 array라서 array 담아서 처리)
	fsInfoList, err := extractFileSystemInfo(fsHandler.Client, fsHandler.FilestoreClient, fsHandler.MonitoringClient, fsHandler.Credential, fsHandler.Region, instances)
	if err != nil {
		cblogger.Error("error while extracting file system info")
		return irs.FileSystemInfo{}, err
	}

	if len(fsInfoList) == 0 {
		return irs.FileSystemInfo{}, errors.New("file system not found")
	}

	fsInfo := fsInfoList[0]

	return *fsInfo, nil

}
func (fsHandler *GCPFileSystemHandler) ListFileSystem() ([]*irs.FileSystemInfo, error) {
	// instances := []*filestorepb.Instance{} // region + zones

	// regionParent := FormatParentPath(fsHandler.Credential.ProjectID, fsHandler.Region.Region)
	// regionInstances, err := ListFilestoreInstances(fsHandler.FilestoreClient, fsHandler.Ctx, regionParent)
	// if err != nil {
	// 	cblogger.Error("error while listing file system instances")
	// 	return nil, err
	// }
	// cblogger.Debug("regionInstances", regionInstances)
	// instances = append(instances, regionInstances...)

	// // 해당 connection에 대한 region
	// zonesByRegion, err := GetZonesByRegion(fsHandler.Client, fsHandler.Credential.ProjectID, fsHandler.Region.Region)
	// if err != nil {
	// 	cblogger.Error("error while listing zones by region")
	// 	return nil, err
	// }
	// cblogger.Debug("zonesByRegion", zonesByRegion)

	// for _, zone := range zonesByRegion.Items {
	// 	zoneParent := FormatParentPath(fsHandler.Credential.ProjectID, zone.Name)
	// 	zoneInstances, err := ListFilestoreInstances(fsHandler.FilestoreClient, fsHandler.Ctx, zoneParent)
	// 	if err != nil {
	// 		cblogger.Error("failed to listing file system instances")
	// 		continue
	// 	}
	// 	instances = append(instances, zoneInstances...)
	// }
	// zoneParent := FormatParentPath(fsHandler.Credential.ProjectID, fsHandler.Region.Zone)
	// zoneInstances, err := ListFilestoreInstances(fsHandler.FilestoreClient, fsHandler.Ctx, zoneParent)
	// if err != nil {
	// 	cblogger.Error("error while listing file system instances")
	// 	return nil, err
	// }
	// cblogger.Debug("zoneInstances", zoneInstances)
	// instances := append(regionInstances, zoneInstances...)

	instances, err := ListFilestoreInstancesByRegionAndZones(fsHandler.FilestoreClient, fsHandler.Client, fsHandler.Credential, fsHandler.Region, fsHandler.Ctx)
	if err != nil {
		cblogger.Error("error while listing file system instances")
		return nil, err
	}
	cblogger.Debug("instances", instances)

	fsInfoList, err := extractFileSystemInfo(fsHandler.Client, fsHandler.FilestoreClient, fsHandler.MonitoringClient, fsHandler.Credential, fsHandler.Region, instances)
	if err != nil {
		cblogger.Error("error while extracting file system info")
		return nil, err
	}

	// fsInfoList := make([]*irs.FileSystemInfo, 0)

	// // vpc 정보조회 : vpc client만 따로 받을까?
	// vNetworkHandler := GCPVPCHandler{
	// 	Client:     fsHandler.Client,
	// 	Region:     fsHandler.Region,
	// 	Ctx:        fsHandler.Ctx,
	// 	Credential: fsHandler.Credential,
	// }
	// for _, instance := range instances {
	// 	vpcInfo, errVnet := vNetworkHandler.GetVPC(irs.IID{SystemId: instance.Networks[0].Network})
	// 	cblogger.Debug(vpcInfo)

	// 	if errVnet != nil {
	// 		cblogger.Error(errVnet)
	// 		return fsInfoList, errVnet
	// 	}
	// 	fsInfo := extractFileSystemInfo(instance, &vpcInfo)

	// 	// mount 된 게 있는지 확인
	// 	if len(instance.FileShares) > 0 {
	// 		// fsInfo.MountTargetList = make([]irs.MountTargetInfo, 0)
	// 		// for _, fileShare := range instance.FileShares {
	// 		// 	// state         protoimpl.MessageState
	// 		// 	// sizeCache     protoimpl.SizeCache
	// 		// 	// unknownFields protoimpl.UnknownFields

	// 		// 	// // Required. The name of the file share. Must use 1-16 characters for the
	// 		// 	// // basic service tier and 1-63 characters for all other service tiers.
	// 		// 	// // Must use lowercase letters, numbers, or underscores `[a-z0-9_]`. Must
	// 		// 	// // start with a letter. Immutable.
	// 		// 	// Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
	// 		// 	// // File share capacity in gigabytes (GB).
	// 		// 	// // Filestore defines 1 GB as 1024^3 bytes.
	// 		// 	// CapacityGb int64 `protobuf:"varint,2,opt,name=capacity_gb,json=capacityGb,proto3" json:"capacity_gb,omitempty"`
	// 		// 	// // The source that this file share has been restored from. Empty if the file
	// 		// 	// // share is created from scratch.
	// 		// 	// //
	// 		// 	// // Types that are assignable to Source:
	// 		// 	// //
	// 		// 	// //	*FileShareConfig_SourceBackup
	// 		// 	// Source isFileShareConfig_Source `protobuf_oneof:"source"`
	// 		// 	// // Nfs Export Options.
	// 		// 	// // There is a limit of 10 export options per file share.
	// 		// 	// NfsExportOptions []*NfsExportOptions `protobuf:"bytes,7,rep,name=nfs_export_options,json=nfsExportOptions,proto3" json:"nfs_export_options,omitempty"`

	// 		// 	//fsInfo.MountTargetList = append(fsInfo.MountTargetList, irs.MountTargetInfo{
	// 		// 	//MountTargetIID: irs.IID{SystemId: fileShare.Name},
	// 		// 	// })
	// 		// }
	// 	}

	// 	fsInfoList = append(fsInfoList, fsInfo)
	// }
	return fsInfoList, nil
}
func (fsHandler *GCPFileSystemHandler) GetFileSystem(iid irs.IID) (irs.FileSystemInfo, error) {
	instances := []*filestorepb.Instance{}
	// region 인스턴스 조회
	instanceName := FormatFilestoreInstanceName(fsHandler.Credential.ProjectID, fsHandler.Region.Region, iid.SystemId)

	instance, err := GetFilestoreInstance(fsHandler.FilestoreClient, fsHandler.Ctx, instanceName)
	if err != nil {
		return irs.FileSystemInfo{}, err
	}

	if instance == nil {
		// zone 인스턴스 조회
		zone := fsHandler.Region.Zone
		if fsHandler.Region.TargetZone != "" {
			zone = fsHandler.Region.TargetZone
		}
		instanceName := FormatFilestoreInstanceName(fsHandler.Credential.ProjectID, zone, iid.SystemId)

		instance, err := GetFilestoreInstance(fsHandler.FilestoreClient, fsHandler.Ctx, instanceName)
		if err != nil {
			return irs.FileSystemInfo{}, err
		}
		instances = append(instances, instance)
	} else {
		instances = append(instances, instance)
	}

	// 파일 시스템 정보 조회 ( extract 조건이 모두 array라서 array 담아서 처리)
	fsInfoList, err := extractFileSystemInfo(fsHandler.Client, fsHandler.FilestoreClient, fsHandler.MonitoringClient, fsHandler.Credential, fsHandler.Region, instances)
	if err != nil {
		cblogger.Error("error while extracting file system info")
		return irs.FileSystemInfo{}, err
	}

	if len(fsInfoList) == 0 {
		return irs.FileSystemInfo{}, errors.New("file system not found")
	}

	fsInfo := fsInfoList[0]

	// // vpc 정보조회
	// vNetworkHandler := GCPVPCHandler{
	// 	Client:     fsHandler.Client,
	// 	Region:     fsHandler.Region,
	// 	Ctx:        fsHandler.Ctx,
	// 	Credential: fsHandler.Credential,
	// }

	// vpcInfo, errVnet := vNetworkHandler.GetVPC(irs.IID{SystemId: instance.Networks[0].Network})
	// cblogger.Debug(vpcInfo)
	// if errVnet != nil {
	// 	cblogger.Error(errVnet)
	// 	return irs.FileSystemInfo{}, errVnet
	// }
	// fsInfo := extractFileSystemInfo(instance, &vpcInfo)

	return *fsInfo, nil
}
func (fsHandler *GCPFileSystemHandler) DeleteFileSystem(iid irs.IID) (bool, error) {

	// connection의 region 기준으로 조회
	instanceName := FormatFilestoreInstanceName(fsHandler.Credential.ProjectID, fsHandler.Region.Region, iid.SystemId)

	// region 인스턴스 조회
	instance, err := GetFilestoreInstance(fsHandler.FilestoreClient, fsHandler.Ctx, instanceName)
	if err != nil {
		//return false, err
		// 없을 수 있음
	}

	if instance == nil {
		zone := fsHandler.Region.Zone
		if fsHandler.Region.TargetZone != "" {
			zone = fsHandler.Region.TargetZone
		}
		instanceName = FormatFilestoreInstanceName(fsHandler.Credential.ProjectID, zone, iid.SystemId)
		// 조회
		instance, err = GetFilestoreInstance(fsHandler.FilestoreClient, fsHandler.Ctx, instanceName)
		if err != nil {
			return false, err
		}

		if instance == nil {
			return false, errors.New("file system not found")
		}
	}

	// instanceName := instance.Name
	// cblogger.Debug("instanceName: ", instanceName)

	err = DeleteFilestoreInstance(fsHandler.FilestoreClient, fsHandler.Ctx, instanceName)
	if err != nil {
		return false, err
	}
	return true, nil
}

// Access Subnet Management
// VPC networks to which the instance is connected. For this version, only a single network is supported.
func (fsHandler *GCPFileSystemHandler) AddAccessSubnet(iid irs.IID, subnetIID irs.IID) (irs.FileSystemInfo, error) {
	return irs.FileSystemInfo{}, errors.New("only a single network is supported.")
}
func (fsHandler *GCPFileSystemHandler) RemoveAccessSubnet(iid irs.IID, subnetIID irs.IID) (bool, error) {
	return false, errors.New("only a single network is supported.")
}
func (fsHandler *GCPFileSystemHandler) ListAccessSubnet(iid irs.IID) ([]irs.IID, error) {
	return nil, errors.New("only a single network is supported.")
}

// Backup Management
func (fsHandler *GCPFileSystemHandler) ScheduleBackup(reqInfo irs.FileSystemBackupInfo) (irs.FileSystemBackupInfo, error) {
	return irs.FileSystemBackupInfo{}, nil
}
func (fsHandler *GCPFileSystemHandler) OnDemandBackup(fsIID irs.IID) (irs.FileSystemBackupInfo, error) {
	return irs.FileSystemBackupInfo{}, nil
}
func (fsHandler *GCPFileSystemHandler) ListBackup(fsIID irs.IID) ([]irs.FileSystemBackupInfo, error) {
	return nil, nil
}
func (fsHandler *GCPFileSystemHandler) GetBackup(fsIID irs.IID, backupID string) (irs.FileSystemBackupInfo, error) {
	return irs.FileSystemBackupInfo{}, nil
}
func (fsHandler *GCPFileSystemHandler) DeleteBackup(fsIID irs.IID, backupID string) (bool, error) {
	return false, nil
}

func getTier(tier string) filestorepb.Instance_Tier {
	switch tier {
	case "BASIC_HDD":
		return filestorepb.Instance_STANDARD
	case "BASIC_SSD":
		return filestorepb.Instance_PREMIUM
	case "HIGH_SCALE_SSD":
		return filestorepb.Instance_HIGH_SCALE_SSD
	case "ENTERPRISE":
		return filestorepb.Instance_ENTERPRISE
	case "ZONAL":
		return filestorepb.Instance_ZONAL
	case "REGIONAL":
		return filestorepb.Instance_REGIONAL
	default:
		return filestorepb.Instance_REGIONAL
	}
}

// user, group, other 권한 설정
func getSquashMode(squashMode string) filestorepb.NfsExportOptions_SquashMode {
	switch squashMode {
	case "SQUASH_MODE_UNSPECIFIED": //SquashMode not set.
		return filestorepb.NfsExportOptions_SQUASH_MODE_UNSPECIFIED
	case "NO_ROOT_SQUASH": //The Root user has root access to the file share
		return filestorepb.NfsExportOptions_NO_ROOT_SQUASH
	case "ROOT_SQUASH": //The Root user has squashed access to the anonymous uid/gid.
		return filestorepb.NfsExportOptions_ROOT_SQUASH
	default: //The Root user has root access to the file share (default).
		return filestorepb.NfsExportOptions_NO_ROOT_SQUASH
	}
}

// 범위 내에 있으면 nil을 반환한다.
// TODO :  BASIC Tial 처리여부 확인 필요
func validateInstanceCapacityRange(metaInfo irs.FileSystemMetaInfo, tier string, capacity int64) error {
	capacityGbRange := irs.CapacityGBRange{}
	switch tier {
	case "ZONAL":
		if capacity >= 1024 && capacity <= 9984 {
			capacityGbRange = metaInfo.CapacityGBOptions["ZONAL_BETWEEN_1_TO_9.75TiB"]
			if capacity >= capacityGbRange.Min && capacity <= capacityGbRange.Max {
				return nil
			}
		} else if capacity >= 10240 && capacity <= 102400 {
			capacityGbRange = metaInfo.CapacityGBOptions["ZONAL_BETWEEN_10_TO_100TiB"]
			if capacity >= capacityGbRange.Min && capacity <= capacityGbRange.Max {
				return nil
			}
		}
	case "REGIONAL":
		if capacity >= 1024 && capacity <= 9984 {
			capacityGbRange = metaInfo.CapacityGBOptions["REGIONAL_BETWEEN_1_TO_9.75TiB"]
			if capacity >= capacityGbRange.Min && capacity <= capacityGbRange.Max {
				return nil
			}
		} else if capacity >= 10240 && capacity <= 102400 {
			capacityGbRange = metaInfo.CapacityGBOptions["REGIONAL_BETWEEN_10_TO_100TiB"]
			if capacity >= capacityGbRange.Min && capacity <= capacityGbRange.Max {
				return nil
			}
		}
	}
	return errors.New("capacity is out of range. min: " + strconv.FormatInt(capacityGbRange.Min, 10) + " max: " + strconv.FormatInt(capacityGbRange.Max, 10))
}

// capacity에 따라 performance capacity 범위도 달라진다. 범위 내에 있으면 성능 설정을 반환한다.
func extractInstancePerformanceRange(metaInfo irs.FileSystemMetaInfo, tier string, capacityGiB int64, performanceInfo map[string]string) (*filestorepb.Instance_PerformanceConfig, error) {
	performanceConfig := &filestorepb.Instance_PerformanceConfig{}
	// PerformanceOptions: map[string][]string{
	// 	"BASIC_HDD":                     {"NONE"}, // Basic은 선택불가
	// 	"BASIC_SSD":                     {"NONE"}, // Basic은 선택불가
	// 	"ZONAL_BETWEEN_1_TO_9.75TiB":    {"Default_fixed_9200", "Custom_Min_4000", "Custom_Max_17000"},
	// 	"ZONAL_BETWEEN_10_TO_100TiB":    {"Default_fixed_75000", "Custom_Min_3000", "Custom_Max_7500"},
	// 	"REGIONAL_BETWEEN_1_TO_9.75TiB": {"Default_fixed_12000", "Custom_Min_4000", "Custom_Max_17000"},
	// 	"REGIONAL_BETWEEN_10_TO_100TiB": {"Default_fixed_75000", "Custom_Min_30000", "Custom_Max_75000"},
	// },

	// tier와 capacityGiB를 기반으로 실제 tier 키를 결정
	var actualTierKey string
	GiBPerTiB := int64(1024)

	if tier == "ZONAL" {
		if capacityGiB >= 1*GiBPerTiB && capacityGiB <= int64(9.75*float64(GiBPerTiB)) {
			actualTierKey = "ZONAL_BETWEEN_1_TO_9.75TiB"
		} else if capacityGiB >= 10*GiBPerTiB && capacityGiB <= 100*GiBPerTiB {
			actualTierKey = "ZONAL_BETWEEN_10_TO_100TiB"
		} else {
			return nil, fmt.Errorf("ZONAL tier에서 지원하지 않는 용량: %d GiB", capacityGiB)
		}
	} else if tier == "REGIONAL" {
		if capacityGiB >= 1*GiBPerTiB && capacityGiB <= int64(9.75*float64(GiBPerTiB)) {
			actualTierKey = "REGIONAL_BETWEEN_1_TO_9.75TiB"
		} else if capacityGiB >= 10*GiBPerTiB && capacityGiB <= 100*GiBPerTiB {
			actualTierKey = "REGIONAL_BETWEEN_10_TO_100TiB"
		} else {
			return nil, fmt.Errorf("REGIONAL tier에서 지원하지 않는 용량: %d GiB", capacityGiB)
		}
	} else {
		// BASIC_HDD, BASIC_SSD 등의 경우
		actualTierKey = tier
	}

	// performanceInfo에서 해당 tier 키의 값 확인
	performanceTier := performanceInfo["PerformanceTier"]
	if performanceTier == "" {
		performanceTier = tier
	}

	// if _, exists := performanceInfo[actualTierKey]; !exists {
	// 	// performanceInfo에 해당 키가 없으면 기본 tier 키로 시도
	// 	if _, exists := performanceInfo[tier]; !exists {
	// 		return nil, fmt.Errorf("performanceInfo에서 tier '%s' 또는 '%s'를 찾을 수 없습니다", actualTierKey, tier)
	// 	}
	// }
	performanceThroughput := performanceInfo["Throughput"]
	var requestedIops int64
	var err error

	cblogger.Debug("performanceThroughput ", performanceThroughput)
	if performanceThroughput != "" {
		requestedIops, err = strconv.ParseInt(performanceThroughput, 10, 64)
		if err != nil {
			return nil, errors.New("performance throughput is not a number")
		}

		if requestedIops == 0 { // 받은게 없으면 default로 설정하도록
			return nil, nil
		}
	}

	if tier == "BASIC_HDD" || tier == "BASIC_SSD" {
		// BASIC 티어는 일반적으로 커스텀 IOPS/Throughput을 설정할 수 없습니다.
		// PerformanceOptions에 "NONE" 또는 "DEFAULT"로 정의되어 있다면,
		// requestedIops가 0이 아니거나 (즉, 사용자가 값을 지정하려 했다면) 에러를 반환합니다.
		if requestedIops != 0 {
			return nil, fmt.Errorf("%s 티어는 사용자 정의 IOPS를 지원하지 않습니다. 요청된 IOPS: %d", tier, requestedIops)
		}
		// Basic 티어는 특별한 검증 없이 nil 반환 (항상 DEFAULT 모드)
		return nil, nil
	}

	// 3. ZONAL 또는 REGIONAL 티어 처리

	// 해당 용량 범위에 대한 성능 규칙 가져오기
	performanceCapacityRange, ok := metaInfo.PerformanceOptions[actualTierKey]
	if !ok {
		return nil, fmt.Errorf("Tier '%s' 용량 %d GiB에 대한 성능 규칙을 찾을 수 없습니다 (키: %s).", tier, capacityGiB, actualTierKey)
	}

	// 4. Min, Max, Default IOPS 값 파싱
	var defaultIops, customMin, customMax int64
	for _, valStr := range performanceCapacityRange {
		if strings.HasPrefix(valStr, "Default_fixed_") { // 사용하지 않을 듯.
			defaultIops, err = extractPerformanceIOPSValue(valStr)
			if err != nil {
				return nil, fmt.Errorf("기본 IOPS '%s' 파싱 오류: %w", valStr, err)
			}
		} else if strings.HasPrefix(valStr, "Custom_Default_") {
			defaultIops, err = extractPerformanceIOPSValue(valStr)
			if err != nil {
				return nil, fmt.Errorf("기본 IOPS '%s' 파싱 오류: %w", valStr, err)
			}
		} else if strings.HasPrefix(valStr, "Custom_Min_") {
			customMin, err = extractPerformanceIOPSValue(valStr)
			if err != nil {
				return nil, fmt.Errorf("최소 IOPS '%s' 파싱 오류: %w", valStr, err)
			}
		} else if strings.HasPrefix(valStr, "Custom_Max_") {
			customMax, err = extractPerformanceIOPSValue(valStr)
			if err != nil {
				return nil, fmt.Errorf("최대 IOPS '%s' 파싱 오류: %w", valStr, err)
			}
		}
	}

	// 5. 요청된 IOPS 값 유효성 검사 및 Default/Custom 결정
	if requestedIops != 0 { // 사용자가 명시적으로 IOPS를 요청한 경우
		// 요청된 값이 default 값과 동일한지 확인
		if requestedIops == defaultIops {
			// default 값을 요청한 경우 (CUSTOM으로 처리해도 무방하지만, 여기서는 DEFAULT로 간주)
			// 특별한 에러 없이 nil 반환
			fmt.Printf("Tier %s, 용량 %d GiB: Default IOPS %d가 명시적으로 요청됨.\n", tier, capacityGiB, requestedIops)
			performanceConfig.Mode = &filestorepb.Instance_PerformanceConfig_FixedIops{
				FixedIops: &filestorepb.Instance_FixedIOPS{
					MaxIops: defaultIops,
				},
			}
		} else {
			// Custom 값 요청 시 Min/Max 범위 검증
			if requestedIops < customMin || requestedIops > customMax {
				return nil, fmt.Errorf("요청된 IOPS %d는 Tier '%s' (%s)의 허용 범위 %d~%d IOPS를 벗어납니다.",
					requestedIops, tier, formatGiB(capacityGiB), customMin, customMax)
			}
			fmt.Printf("Tier %s, 용량 %d GiB: Custom IOPS %d가 유효함.\n", tier, capacityGiB, requestedIops)
			performanceConfig.Mode = &filestorepb.Instance_PerformanceConfig_FixedIops{
				FixedIops: &filestorepb.Instance_FixedIOPS{
					MaxIops: requestedIops,
				},
			}
		}

		//supplied IOPS 4100 must be a multiple of 1000.
		if requestedIops%1000 != 0 {
			return nil, fmt.Errorf("supplied IOPS %d must be a multiple of 1000.", requestedIops)
		}
	} else { // 사용자가 IOPS를 요청하지 않은 경우 (Default 값 적용)
		fmt.Printf("Tier %s, 용량 %d GiB: IOPS가 요청되지 않아 기본 IOPS %d가 적용됩니다.\n", tier, capacityGiB, defaultIops)
		performanceConfig.Mode = &filestorepb.Instance_PerformanceConfig_FixedIops{
			FixedIops: &filestorepb.Instance_FixedIOPS{
				MaxIops: defaultIops,
			},
		}
	}

	return performanceConfig, nil
}

// PerformanceConfig에서 IOPS 값을 추출한다.
func extractPerformanceIOPSValue(s string) (int64, error) {
	valueNumRegex := regexp.MustCompile(`_(\d+)$`)
	matches := valueNumRegex.FindStringSubmatch(s)
	if len(matches) < 2 {
		return 0, fmt.Errorf("형식에 맞는 숫자를 찾을 수 없음: %s", s)
	}
	return strconv.ParseInt(matches[1], 10, 64)
}

// 헬퍼 함수 (GiB를 TiB/GiB 형식으로 출력)
func formatGiB(gib int64) string {
	GiBPerTiB := int64(1024)
	if gib%GiBPerTiB == 0 {
		return fmt.Sprintf("%dTiB", gib/GiBPerTiB)
	}
	return fmt.Sprintf("%dGiB", gib)
}
