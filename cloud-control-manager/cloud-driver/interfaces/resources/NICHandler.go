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
type NICStatus string

const (
	NICAvailable NICStatus = "Available" // Created, not attached to any VM
	NICAttached  NICStatus = "Attached"  // Attached to a VM
	NICDeleting  NICStatus = "Deleting"
	NICError     NICStatus = "Error"
	NICNotFound  NICStatus = "NotFound" // Registered in Spider but not found in CSP
)

// -------- Info Structure
// NICInfo represents the information of a Network Interface Card resource.
type NICInfo struct {
	IId IID `json:"IId" validate:"required"` // {NameId, SystemId}

	VpcIID    IID `json:"VpcIID" validate:"required"`    // VPC this NIC belongs to
	SubnetIID IID `json:"SubnetIID" validate:"required"` // Subnet this NIC belongs to

	SecurityGroupIIDs []IID `json:"SecurityGroupIIDs,omitempty" validate:"omitempty"` // Attached security groups

	PrivateIP  string   `json:"PrivateIP" validate:"required"`            // Primary private IP (convenience)
	PrivateIPs []string `json:"PrivateIPs,omitempty" validate:"omitempty"` // All private IPs (primary first, then secondary)
	PublicIPs  []string `json:"PublicIPs,omitempty" validate:"omitempty"`  // Public IPs index-aligned to PrivateIPs ("" = no EIP for that IP)

	PublicIP string `json:"PublicIP,omitempty" validate:"omitempty"` // Primary public IP (convenience)

	OwnerVM     IID       `json:"OwnerVM,omitempty" validate:"omitempty"` // VM this NIC is attached to
	DeviceIndex int       `json:"DeviceIndex,omitempty"`                  // 0=primary, 1,2,...=secondary
	MACAddress  string    `json:"MACAddress,omitempty" validate:"omitempty"`
	Status      NICStatus `json:"Status" validate:"required"`

	CreatedTime  time.Time  `json:"CreatedTime" validate:"required"`
	TagList      []KeyValue `json:"TagList,omitempty" validate:"omitempty"`
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"`
}

// NICReqInfo represents the request information for creating a NIC.
type NICReqInfo struct {
	IId IID `json:"IId" validate:"required"`

	VpcIID    IID `json:"VpcIID" validate:"required"`
	SubnetIID IID `json:"SubnetIID" validate:"required"`

	SecurityGroupIIDs []IID `json:"SecurityGroupIIDs,omitempty" validate:"omitempty"`

	TagList []KeyValue `json:"TagList,omitempty" validate:"omitempty"`
}

// -------- NIC API
type NICHandler interface {

	//------ NIC Management
	ListIID() ([]*IID, error)
	CreateNIC(nicReqInfo NICReqInfo) (NICInfo, error)
	ListNIC() ([]*NICInfo, error)
	GetNIC(nicIID IID) (NICInfo, error)
	DeleteNIC(nicIID IID) (bool, error)

	//------ VM Attachment
	AttachNIC(nicIID IID, vmIID IID) (NICInfo, error)
	DetachNIC(nicIID IID) (bool, error)

	//------ Private IP Management
	// AddPrivateIP adds a secondary private IP to the NIC. If privateIP is empty, CSP auto-assigns one.
	AddPrivateIP(nicIID IID, privateIP string) (NICInfo, error)
	// RemovePrivateIP removes a secondary private IP from the NIC.
	RemovePrivateIP(nicIID IID, privateIP string) (bool, error)

	//------ OS Configuration
	// GetNICOSConfigScript returns a shell script that must be run inside the VM OS
	// after a secondary NIC (or additional private/public IP) is attached via the cloud API.
	// AWS uses DHCP and requires no OS-level configuration; its implementation returns an empty string.
	// All other CSPs (Azure, Alibaba, Tencent, IBM, OpenStack) require explicit routing table
	// and interface bring-up commands inside the guest OS.
	// The returned script is a self-contained bash script ready to execute as root.
	GetNICOSConfigScript(nicIID IID) (string, error)
}
