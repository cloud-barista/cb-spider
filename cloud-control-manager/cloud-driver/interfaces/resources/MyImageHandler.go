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
type MyImageStatus string

const (
	MyImageAvailable   MyImageStatus = "Available"
	MyImageUnavailable MyImageStatus = "Unavailable"
)

// -------- Info Structure
// MyImageInfo represents the information of a MyImage resource.
type MyImageInfo struct {
	IId      IID `json:"IId" validate:"required"` // {NameId, SystemId}
	SourceVM IID `json:"SourceVM" validate:"omitempty"`

	Status MyImageStatus `json:"Status" validate:"required" example:"Available"` // Available | Unavailable

	CreatedTime  time.Time  `json:"CreatedTime" validate:"required"`
	TagList      []KeyValue `json:"TagList,omitempty" validate:"omitempty"`
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"`
}

// -------- MyImage API
type MyImageHandler interface {

	//------ Snapshot to create a MyImage
	SnapshotVM(snapshotReqInfo MyImageInfo) (MyImageInfo, error)

	//------ MyImage Management
	ListIID() ([]*IID, error)
	ListMyImage() ([]*MyImageInfo, error)
	GetMyImage(myImageIID IID) (MyImageInfo, error)
	CheckWindowsImage(myImageIID IID) (bool, error)
	DeleteMyImage(myImageIID IID) (bool, error)
}
