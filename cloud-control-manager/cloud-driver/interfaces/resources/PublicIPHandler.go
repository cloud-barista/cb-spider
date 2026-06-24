// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resources interfaces of Cloud Driver.
//
// by CB-Spider Team, 2025.06.

package resources

import "time"

// -------- Const
type PublicIPStatus string

const (
	PublicIPAvailable  PublicIPStatus = "Available"
	PublicIPAssociated PublicIPStatus = "Associated"
	PublicIPDeleting   PublicIPStatus = "Deleting"
	PublicIPError      PublicIPStatus = "Error"
	PublicIPNotFound   PublicIPStatus = "NotFound" // Registered in Spider but not found in CSP
)

// -------- Info Structure
// PublicIPInfo represents the information of a Public IP resource.
type PublicIPInfo struct {
	IId             IID            `json:"IId" validate:"required"`                                      // {NameId, SystemId}
	PublicIPAddress string         `json:"PublicIPAddress" validate:"required" example:"52.10.20.30"`    // Allocated public IP address
	Status          PublicIPStatus `json:"Status" validate:"required" example:"Available"`               // Current status
	OwnedVM         IID            `json:"OwnedVM,omitempty" validate:"omitempty"`                        // Associated VM (when Status is Associated)
	OwnedNIC        IID            `json:"OwnedNIC,omitempty" validate:"omitempty"`                       // Associated NIC (NameId=Spider NIC name, SystemId=CSP NIC ID)
	OwnedPrivateIP  string         `json:"OwnedPrivateIP,omitempty" validate:"omitempty"`                 // The private IP on the NIC that this Public IP is mapped to
	CreatedTime     time.Time      `json:"CreatedTime" validate:"required"`                               // Allocation time
	TagList         []KeyValue     `json:"TagList,omitempty" validate:"omitempty"`                        // Tags
	KeyValueList    []KeyValue     `json:"KeyValueList,omitempty" validate:"omitempty"`                   // CSP-specific additional info
}

// -------- Public IP API
type PublicIPHandler interface {

	//------ PublicIP Management
	ListIID() ([]*IID, error)
	CreatePublicIP(publicIPReqInfo PublicIPInfo) (PublicIPInfo, error)
	ListPublicIP() ([]*PublicIPInfo, error)
	GetPublicIP(publicIPIID IID) (PublicIPInfo, error)
	DeletePublicIP(publicIPIID IID) (bool, error)

	//------ NIC/VM Association
	// vmIID: used by NCP (VM-level NAT). nicIID+privateIP: used by other CSPs.
	// GCP: nicIID holds the NIC name (e.g. "nic0", "nic1"), privateIP is ignored.
	AssociatePublicIP(publicIPIID IID, vmIID IID, nicIID IID, privateIP string) (PublicIPInfo, error)
	DisassociatePublicIP(publicIPIID IID) (bool, error)
}
