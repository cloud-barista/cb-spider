// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2022.08.

package resources

import "time"

// -------- Const
type DiskStatus string

const (
	DiskCreating  DiskStatus = "Creating"
	DiskAvailable DiskStatus = "Available"
	DiskAttached  DiskStatus = "Attached"
	DiskDeleting  DiskStatus = "Deleting"
	DiskError     DiskStatus = "Error"
)

// -------- Info Structure
// DiskInfo represents the information of a Disk resource.
type DiskInfo struct {
	IId  IID    `json:"IId" validate:"required"`                       // {NameId, SystemId}
	Zone string `json:"Zone" validate:"required" example:"us-east-1a"` // Target Zone Name

	DiskType string `json:"DiskType" validate:"required" example:"gp2"` // "gp2", "Premium SSD", ...
	DiskSize string `json:"DiskSize" validate:"required" example:"100"` // "default", "50", "1000" (unit is GB)

	Status  DiskStatus `json:"Status" validate:"required" example:"Available"`
	OwnerVM IID        `json:"OwnerVM" validate:"omitempty"` // When the Status is DiskAttached

	CreatedTime  time.Time  `json:"CreatedTime" validate:"required"`             // The time when the disk was created
	TagList      []KeyValue `json:"TagList,omitempty" validate:"omitempty"`      // A list of tags associated with this disk
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"` // Additional key-value pairs associated with this disk
}

// -------- Disk API
type DiskHandler interface {

	//------ Disk Management
        ListIID() ([]*IID, error)
	CreateDisk(DiskReqInfo DiskInfo) (DiskInfo, error)
	ListDisk() ([]*DiskInfo, error)
	GetDisk(diskIID IID) (DiskInfo, error)
	ChangeDiskSize(diskIID IID, size string) (bool, error)
	DeleteDisk(diskIID IID) (bool, error)

	//------ Disk Attachment
	AttachDisk(diskIID IID, ownerVM IID) (DiskInfo, error)
	DetachDisk(diskIID IID, ownerVM IID) (bool, error)
}
