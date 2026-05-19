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

import "time"

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
	// filled by the cloud driver
	SupportedEngines map[string][]string `json:"SupportedEngines"` // Supported DB engine names → versions. e.g., {"mysql": ["8.0", "8.4"], "postgresql": ["15", "16"]}

	SupportsHighAvailability   bool `json:"SupportsHighAvailability"`   // true if HA/Multi-AZ can be configured
	SupportsBackup             bool `json:"SupportsBackup"`             // true if managed automatic backup is supported
	SupportsPublicAccess       bool `json:"SupportsPublicAccess"`       // true if public access can be toggled
	SupportsDeletionProtection bool `json:"SupportsDeletionProtection"` // true if deletion protection is available
	SupportsEncryption         bool `json:"SupportsEncryption"`         // true if storage encryption is available

	StorageTypeOptions map[string][]string `json:"StorageTypeOptions,omitempty"` // Available storage types per engine. e.g., {"mysql": ["gp2", "gp3", "io1"]}
	StorageSizeRange   StorageSizeRange    `json:"StorageSizeRange,omitempty"`   // Min/Max storage size in GB
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
	StorageType string `json:"StorageType,omitempty" example:"gp2"`           // e.g., "gp2", "io1", "SSD", "cloud_essd"
	StorageSize string `json:"StorageSize" validate:"required" example:"100"` // in GB

	// Network
	SubnetIIDs        []IID  `json:"SubnetIIDs,omitempty"`          // Subnet IIDs for DB placement
	SecurityGroupIIDs []IID  `json:"SecurityGroupIIDs,omitempty"`   // Associated Security Groups
	Port              string `json:"Port,omitempty" example:"3306"` // DB listen port

	// Authentication
	MasterUserName     string `json:"MasterUserName" validate:"required" example:"admin"` // Master user name
	MasterUserPassword string `json:"MasterUserPassword,omitempty"`                       // Master user password (for Create request only)

	// Database
	DatabaseName string `json:"DatabaseName,omitempty" example:"mydb"` // Initial database name

	// High Availability & Replication
	HighAvailability bool   `json:"HighAvailability,omitempty" default:"false"` // Multi-AZ / HA enabled
	ReplicationType  string `json:"ReplicationType,omitempty" example:"async"`  // async | semi-sync | sync (for response)

	// Backup
	BackupRetentionDays int    `json:"BackupRetentionDays,omitempty" example:"7"` // Automated backup retention period in days
	BackupTime          string `json:"BackupTime,omitempty" example:"03:00"`      // Preferred backup time (HH:MM in UTC)

	// Access
	PublicAccess bool   `json:"PublicAccess,omitempty" default:"false"` // Whether publicly accessible
	Endpoint     string `json:"Endpoint,omitempty"`                     // Connection endpoint (for response)

	// Encryption
	Encryption bool `json:"Encryption,omitempty" default:"false"` // Storage encryption enabled

	// Protection
	DeletionProtection bool `json:"DeletionProtection,omitempty" default:"false"` // Deletion protection enabled

	//**************************************************************************************************
	//** (1) Basic setup: If not set by the user, these fields use CSP default values.
	//** (2) Advanced setup: If set by the user, these fields enable CSP-specific RDBMS features.
	//**    → Use GetMetaInfo() to discover CSP-supported options before setting advanced fields.
	//**    Advanced fields: StorageType, HighAvailability, PublicAccess, Encryption, DeletionProtection
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
	GetMetaInfo() (RDBMSMetaInfo, error)

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
