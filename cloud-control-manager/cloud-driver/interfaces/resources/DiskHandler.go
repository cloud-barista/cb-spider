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


//-------- Const
type DiskStatus string

const (
        DiskCreating 	DiskStatus = "Creating"
        DiskAvailable   DiskStatus = "Available"
        DiskAttached   	DiskStatus = "Attached"
        DiskDeleting   	DiskStatus = "Deleting"
        DiskError   	DiskStatus = "Error"
)

//-------- Info Structure
type DiskInfo struct {
	IId	IID 	// {NameId, SystemId}

        DiskType string  // "", "SSD(gp2)", "Premium SSD", ...
	DiskSize string  // "", "default", "50", "1000"  # (GB)
	
        Status 		DiskStatus	// DiskCreating | DiskAvailable | DiskAttached | DiskDeleting | DiskError
	OwnerVM		IID		// When the Status is DiskAttached

	CreatedTime	time.Time
	KeyValueList []KeyValue
}


//-------- Disk API
type DiskHandler interface {

	//------ Disk Management
	CreateDisk(DiskReqInfo DiskInfo) (DiskInfo, error)
	ListDisk() ([]*DiskInfo, error)
	GetDisk(diskIID IID) (DiskInfo, error)
	ChangeDiskSize(diskIID IID, size string) (bool, error)
	DeleteDisk(diskIID IID) (bool, error)


	//------ Disk Attachment
	AttachDisk(diskIID IID, ownerVM IID) (DiskInfo, error)
	DetachDisk(diskIID IID, ownerVM IID) (bool, error)
}
