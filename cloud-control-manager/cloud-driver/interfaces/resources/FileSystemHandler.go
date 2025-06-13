// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// by CB-Spider Team, 2025.06.

package resources

import "time"

// -------- File System Status Constants
type FileSystemStatus string

const (
	FileSystemCreating  FileSystemStatus = "Creating"
	FileSystemAvailable FileSystemStatus = "Available"
	FileSystemDeleting  FileSystemStatus = "Deleting"
	FileSystemError     FileSystemStatus = "Error"
)

// -------- File System Type Constants
type FileSystemType string

const (
	RegionType FileSystemType = "REGION-TYPE"
	ZoneType   FileSystemType = "ZONE-TYPE"
)

const (
	// only for response, not for request
	RegionVPCBasedType  FileSystemType = "REGION-VPC-BASED-TYPE"
	RegionZoneBasedType FileSystemType = "REGION-ZONE-BASED-TYPE"
)

// =============================== Meta API
type FileSystemMetaInfo struct {
	// filled by the cloud driver
	SupportsFileSystemType map[FileSystemType]bool    `json:"SupportsFileSystemType"`       // e.g., {"RegionType": true, "ZoneType": true, "RegionZoneBasedType": true, ...}
	SupportsNFSVersion     []string                   `json:"SupportsNFSVersion"`           // e.g., ["3.0", "4.1"]
	SupportsCapacity       bool                       `json:"SupportsCapacity"`             // true if capacity can be specified
	CapacityGBOptions      map[string]CapacityGBRange `json:"CapacityGBOptions,omitempty"`  // Capacity ranges per file system option (valid only if SupportsCapacity is true). e.g., GCP Filestore: {"Basic": {Min: 1024, Max: 65229}, "Zonal": {Min: 1024, Max: 102400}, "Regional": {Min: 1024, Max: 102400}}
	PerformanceOptions     map[string][]string        `json:"PerformanceOptions,omitempty"` // Available performance settings per file system option. e.g., {"Basic": ["STANDARD"], "Zonal": ["HIGH_SCALE", "EXTREME"]}
}

type CapacityGBRange struct {
	Min int64 `json:"Min" example:"100"`    // Minimum capacity in GB
	Max int64 `json:"Max" example:"102400"` // Maximum capacity in GB
}

type FileSystemMetaApi interface {
	GetMetaInfo() (FileSystemMetaInfo, error)
}

//=============================== Meta API

// -------- File System Info Structures
type FileSystemInfo struct {
	IId    IID    `json:"IId" validate:"required"` // {NameId, SystemId}
	Region string `json:"Region,omitempty" example:"us-east-1"`
	Zone   string `json:"Zone,omitempty" example:"us-east-1a"`
	VpcIID IID    `json:"VpcIID" validate:"required"` // Owner VPC IID

	Encryption     bool                 `json:"Encryption,omitempty" default:"false"` // Encryption enabled or not
	BackupSchedule FileSystemBackupInfo `json:"BackupSchedule,omitempty"`             // Cron schedule for backups, default is "0 5 * * *" (Every day at 5 AM)
	TagList        []KeyValue           `json:"TagList,omitempty" validate:"omitempty"`

	//**************************************************************************************************
	//** (1) Basic setup: If not set by the user, these fields use CSP default values.
	//** (2) Advanced setup: If set by the user, these fields enable CSP-specific file system features.
	//**************************************************************************************************
	FileSystemType FileSystemType `json:"FileSystemType,omitempty" example:"RegionType"` // RegionType, ZoneType; CSP default if omitted
	NFSVersion     string         `json:"NFSVersion" validate:"required" example:"4.1"`  // NFS protocol version, e.g., "3.0", "4.1"; CSP default if omitted
	CapacityGB     int64          `json:"CapacityGB,omitempty" example:"1024"`           // Capacity in GB, -1 when not applicable. Ignored if CSP unsupported.; CSP default if omitted
	// Each key/value must match one of the PerformanceOptions provided by the cloud driver for the selected file system type.
	PerformanceInfo map[string]string `json:"PerformanceInfo,omitempty"` // Performance options, e.g., {"Tier": "STANDARD"}, {"ThroughputMode": "provisioned", "Throughput": "128"}; CSP default if omitted
	//**************************************************************************************************

	// only for response, not for request
	Status          FileSystemStatus  `json:"Status" validate:"required" example:"Available"`
	UsedSizeGB      int64             `json:"UsedSizeGB" validate:"required" example:"256"` // Current used size in GB.
	MountTargetList []MountTargetInfo `json:"MountTargetList,omitempty"`

	CreatedTime  time.Time  `json:"CreatedTime" validate:"required"`
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"` // Additional key-value pairs associated with this File System
}

type MountTargetInfo struct {
	SubnetIID           IID        `json:"SubnetIID,omitempty"`                         // location of the mount target
	SecurityGroups      []string   `json:"SecurityGroups,omitempty"`                    // security groups associated with the mount target
	Endpoint            string     `json:"Endpoint,omitempty"`                          // mount target endpoint (IP, DNS, URL)
	MountCommandExample string     `json:"MountCommandExample,omitempty"`               // Example mount command
	KeyValueList        []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"` // Additional key-value pairs associated with this mount target
}

// -------- Backup Structures
type FileSystemBackupInfo struct {
	FileSystemIID string       `json:"FileSystemIID" validate:"required"` // The File System IID to which this backup belongs
	Schedule      CronSchedule `json:"BackupSchedule,omitempty"`          // Cron schedule for backups, default is "0 5 * * *" (Every day at 5 AM)

	// for response only, not for request
	BackupID     string     `json:"BackupID" validate:"required"`
	CreationTime time.Time  `json:"CreationTime"`
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"`
}

type CronSchedule struct { // default: "0 5 * * *" ## Every day at 5 AM
	Minute     string `json:"Minute" default:"0"`     // 0-59, *
	Hour       string `json:"Hour" default:"5"`       // 0-23, *
	DayOfMonth string `json:"DayOfMonth" default:"*"` // 1-31, *
	Month      string `json:"Month" default:"*"`      // 1-12, *
	DayOfWeek  string `json:"DayOfWeek" default:"*"`  // 0-6 (Sunday=0), *
}

// -------- File System Handler Interface
type FileSystemHandler interface {
	ListIID() ([]*IID, error)

	CreateFileSystem(reqInfo FileSystemInfo) (FileSystemInfo, error)
	ListFileSystem() ([]*FileSystemInfo, error)
	GetFileSystem(iid IID) (FileSystemInfo, error)
	DeleteFileSystem(iid IID) (bool, error)

	// Mount Target Management
	AddMountTarget(iid IID, target MountTargetInfo) (FileSystemInfo, error)
	RemoveMountTarget(iid IID, target MountTargetInfo) (bool, error)
	ListMountTarget(iid IID) ([]MountTargetInfo, error)

	// Backup Management
	ScheduleBackup(reqInfo FileSystemBackupInfo) (FileSystemBackupInfo, error) // Create a backup with the specified schedule
	OnDemandBackup(fsIID IID) (FileSystemBackupInfo, error)                    // Create an on-demand backup for the specified file system
	ListBackups(fsIID IID) ([]FileSystemBackupInfo, error)
	GetBackup(fsIID IID, backupID string) (FileSystemBackupInfo, error)
	DeleteBackup(fsIID IID, backupID string) (bool, error)

	// TBD
	// RestoreBackup(fsIID IID, backupIID IID, newFsIID IID) (FileSystemInfo, error)
}
