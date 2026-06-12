// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, April 2026.

package resources

import (
	"fmt"
	"strings"
	"time"
)

// -------- Const
type RDBMSStatus string

const (
	RDBMSCreating  RDBMSStatus = "Creating"
	RDBMSAvailable RDBMSStatus = "Available"
	RDBMSDeleting  RDBMSStatus = "Deleting"
	RDBMSStopped   RDBMSStatus = "Stopped"
	RDBMSError     RDBMSStatus = "Error"
)

// -------- Meta Info Structures

// RDBMSMetaInfo provides CSP-specific capability information for RDBMS provisioning.
// Use GetMetaInfo() to discover what each CSP supports before creating an RDBMS instance.
// @description RDBMS Meta Information for CSP-specific capabilities
type RDBMSMetaInfo struct {
	DBEngine              string           `json:"DBEngine" example:"mysql"`                                    // Requested DB engine name. e.g., mysql, mariadb, postgresql
	SupportedVersions     []string         `json:"SupportedVersions" example:"8.0,8.4"`                         // Supported versions for the requested DB engine
	DBInstanceSpecOptions []string         `json:"DBInstanceSpecOptions,omitempty" example:"db.t3.medium,1000"` // Available DBInstanceSpec values for the requested DB engine. "NA" if CSP does not provide spec list API.
	StorageTypeOptions    []string         `json:"StorageTypeOptions,omitempty" example:"gp2,gp3,io1"`          // Available storage types for the requested DB engine
	StorageSizeRange      StorageSizeRange `json:"StorageSizeRange,omitempty"`                                  // Min/Max storage size in GB for the requested DB engine

	SupportsHighAvailability   bool   `json:"SupportsHighAvailability"`       // true if HA/Multi-AZ can be configured
	SupportsBackup             bool   `json:"SupportsBackup"`                 // true if managed automatic backup is supported
	BackupRetentionRange       string `json:"BackupRetentionRange,omitempty"` // Backup retention range at creation time (e.g., "1-35", "7-730"). "NA" if not configurable at creation.
	SupportsPublicAccess       bool   `json:"SupportsPublicAccess"`           // true if public access can be toggled
	SupportsDeletionProtection bool   `json:"SupportsDeletionProtection"`     // true if deletion protection is available
	SupportsEncryption         bool   `json:"SupportsEncryption"`             // true if storage encryption is available

	SupportsStorageTypeSelection    bool `json:"SupportsStorageTypeSelection"`    // true if user can specify StorageType at creation; false if CSP sets it automatically (e.g., Azure, NCP)
	SupportsStorageSizeConfiguration bool `json:"SupportsStorageSizeConfiguration"` // true if user can specify StorageSize at creation; false if CSP manages size automatically (e.g., NCP)

	RequiresSubnet        bool `json:"RequiresSubnet"`        // true if SubnetNames is required at creation
	RequiresSecurityGroup bool `json:"RequiresSecurityGroup"` // true if SecurityGroupNames is required at creation
}

func NormalizeRDBMSEngine(dbEngine string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(dbEngine))
	switch normalized {
	case "mysql", "mariadb", "postgresql":
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported DBEngine '%s'; valid values are mysql, mariadb, postgresql", dbEngine)
	}
}

func BuildRDBMSMetaInfo(dbEngine string, supportedEngines map[string][]string, dbInstanceSpecOptions map[string][]string, storageTypeOptions map[string][]string, storageSizeRange StorageSizeRange, supportsHighAvailability, supportsBackup, supportsPublicAccess, supportsDeletionProtection, supportsEncryption bool, backupRetentionRange string, requiresSubnet, requiresSecurityGroup, supportsStorageTypeSelection, supportsStorageSizeConfiguration bool) (RDBMSMetaInfo, error) {
	normalizedEngine, err := NormalizeRDBMSEngine(dbEngine)
	if err != nil {
		return RDBMSMetaInfo{}, err
	}

	versions := append([]string(nil), supportedEngines[normalizedEngine]...)
	if len(versions) == 0 {
		return RDBMSMetaInfo{}, fmt.Errorf("DBEngine '%s' is not supported", normalizedEngine)
	}

	instanceSpecs := append([]string(nil), dbInstanceSpecOptions[normalizedEngine]...)
	storageTypes := append([]string(nil), storageTypeOptions[normalizedEngine]...)
	return RDBMSMetaInfo{
		DBEngine:                   normalizedEngine,
		SupportedVersions:          versions,
		DBInstanceSpecOptions:      instanceSpecs,
		StorageTypeOptions:         storageTypes,
		StorageSizeRange:           storageSizeRange,
		SupportsHighAvailability:   supportsHighAvailability,
		SupportsBackup:             supportsBackup,
		BackupRetentionRange:       backupRetentionRange,
		SupportsPublicAccess:       supportsPublicAccess,
		SupportsDeletionProtection: supportsDeletionProtection,
		SupportsEncryption:         supportsEncryption,
		SupportsStorageTypeSelection:    supportsStorageTypeSelection,
		SupportsStorageSizeConfiguration: supportsStorageSizeConfiguration,
		RequiresSubnet:             requiresSubnet,
		RequiresSecurityGroup:      requiresSecurityGroup,
	}, nil
}

// StorageSizeRange represents the minimum and maximum storage size in GB.
type StorageSizeRange struct {
	Min int64 `json:"Min" example:"20"`    // Minimum storage in GB
	Max int64 `json:"Max" example:"65536"` // Maximum storage in GB
}

// -------- Info Structure

// RDBMSInfo represents the details of a Relational Database instance.
// @description Relational Database (RDBMS) Information
type RDBMSInfo struct {
	IId    IID `json:"IId" validate:"required"`    // {NameId, SystemId}
	VpcIID IID `json:"VpcIID" validate:"required"` // Owner VPC IID

	// DB Engine
	DBEngine        string `json:"DBEngine" validate:"required" example:"mysql"`      // mysql | mariadb | postgresql
	DBEngineVersion string `json:"DBEngineVersion" validate:"required" example:"8.0"` // e.g., "8.0", "10.6", "15"

	// Instance Spec
	DBInstanceSpec string `json:"DBInstanceSpec" validate:"required" example:"db.t3.medium"` // CSP instance class/type
	DBInstanceType string `json:"DBInstanceType,omitempty" example:"Primary"`                // Primary | ReadReplica (for response)

	// Storage
	// StorageType: storage volume type for the RDBMS instance.
	// e.g., "gp2", "io1", "SSD", "cloud_essd"
	// OpenStack: configurable at creation time, but Trove API does not return this field in responses (always "NA").
	StorageType string `json:"StorageType,omitempty" example:"gp2"`
	StorageSize string `json:"StorageSize" validate:"required" example:"100"` // in GB
	// Iops: Provisioned IOPS for the storage volume.
	// AWS: required for io1/io2 (100–64000).
	// Other CSPs: not used.
	Iops string `json:"Iops,omitempty" example:"3000"`

	// Network
	SubnetIIDs        []IID `json:"SubnetIIDs,omitempty"`        // Subnet IIDs for DB placement
	SecurityGroupIIDs []IID `json:"SecurityGroupIIDs,omitempty"` // Associated Security Groups

	// Authentication
	MasterUserName     string `json:"MasterUserName" validate:"required" example:"admin"` // Master user name
	MasterUserPassword string `json:"MasterUserPassword,omitempty"`                       // Master user password (for Create request only)

	// High Availability
	HighAvailability bool `json:"HighAvailability,omitempty" default:"false"` // Multi-AZ / HA enabled

	// Backup
	BackupRetentionDays int    `json:"BackupRetentionDays,omitempty" example:"7"` // Automated backup retention period in days (configurable at creation if CSP supports)
	BackupTime          string `json:"BackupTime,omitempty" example:"03:00"`      // Preferred backup time (read-only, CSP-managed. Not configurable at creation via Spider.)

	// Access
	PublicAccess bool   `json:"PublicAccess,omitempty" default:"false"` // Whether publicly accessible
	Endpoint     string `json:"Endpoint,omitempty"`                     // Connection endpoint (for response)

	// Encryption - read-only, CSP-managed. Not configurable at creation via Spider.
	Encryption bool `json:"Encryption,omitempty" default:"false"` // Storage encryption enabled (CSP default)

	// Protection
	DeletionProtection bool `json:"DeletionProtection,omitempty" default:"false"` // Deletion protection enabled

	//**************************************************************************************************
	//** (1) Basic setup: If not set by the user, these fields use CSP default values.
	//** (2) Advanced setup: If set by the user, these fields enable CSP-specific RDBMS features.
	//**    → Use GetMetaInfo() to discover CSP-supported options before setting advanced fields.
	//**    Advanced fields: StorageType, HighAvailability, PublicAccess, DeletionProtection
	//**    Read-only fields (CSP-managed): BackupTime, Encryption
	//**************************************************************************************************

	// Status (for response)
	Status RDBMSStatus `json:"Status,omitempty" validate:"required" example:"Available"`

	CreatedTime  time.Time  `json:"CreatedTime,omitempty"`
	TagList      []KeyValue `json:"TagList,omitempty" validate:"omitempty"`
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"`
}

// -------- RDBMS Handler API
type RDBMSHandler interface {

	// Meta API
	GetMetaInfo(dbEngine string) (RDBMSMetaInfo, error)

	//------ RDBMS Management
	ListIID() ([]*IID, error)
	CreateRDBMS(rdbmsReqInfo RDBMSInfo) (RDBMSInfo, error)
	ListRDBMS() ([]*RDBMSInfo, error)
	GetRDBMS(rdbmsIID IID) (RDBMSInfo, error)
	DeleteRDBMS(rdbmsIID IID) (bool, error)

	//------ Instance Control (TBD: support will be decided later)
	// ChangeSpec(rdbmsIID IID, newSpec string) (RDBMSInfo, error)        // Change instance class/spec
	// ChangeStorageSize(rdbmsIID IID, newSize string) (bool, error)      // Expand storage
}
