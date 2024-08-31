// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2020.04.
// by CB-Spider Team, 2019.06.

package resources

import (
	"time"
)

type ImageType string

const (
	PublicImage ImageType = "PublicImage"
	MyImage     ImageType = "MyImage"
)

type Platform string

const (
	LINUX_UNIX Platform = "LINUX/UNIX"
	WINDOWS    Platform = "WINDOWS"
)

type VMReqInfo struct {
	IId IID // {NameId, SystemId}

	ImageType         ImageType // PublicImage | MyImage, default: PublicImage
	ImageIID          IID
	VpcIID            IID
	SubnetIID         IID
	SecurityGroupIIDs []IID

	VMSpecName string
	KeyPairIID IID

	RootDiskType string // "", "SSD(gp2)", "Premium SSD", ...
	RootDiskSize string // "", "default", "50", "1000" (unit is GB)

	DataDiskIIDs []IID

	VMUserId     string
	VMUserPasswd string
	WindowsType  bool

	TagList []KeyValue
}

type VMStatusInfo struct {
	IId      IID      `json:"IId" validate:"required" example:"` // {NameId: 'vm-01', SystemId: 'i-12345678'}"
	VmStatus VMStatus `json:"VmStatus" validate:"required" example:"Running"`
}

// VMStatus represents the possible statuses of a VM.
// @description The status of a Virtual Machine (VM).
// @enum string
// @enum values [Creating, Running, Suspending, Suspended, Resuming, Rebooting, Terminating, Terminated, NotExist, Failed]
type VMStatus string

const (
	Creating VMStatus = "Creating" // from launch to running
	Running  VMStatus = "Running"

	Suspending VMStatus = "Suspending" // from running to suspended
	Suspended  VMStatus = "Suspended"
	Resuming   VMStatus = "Resuming" // from suspended to running

	Rebooting VMStatus = "Rebooting" // from running to running

	Terminating VMStatus = "Terminating" // from running, suspended to terminated
	Terminated  VMStatus = "Terminated"
	NotExist    VMStatus = "NotExist" // VM does not exist

	Failed VMStatus = "Failed"
)

type RegionInfo struct {
	Region string `json:"Region" validate:"required" example:"us-east-1"`
	Zone   string `json:"Zone,omitempty" validate:"omitempty" example:"us-east-1a"`
}

type VMInfo struct {
	IId       IID       `json:"IId" validate:"required"`                                      // example:"{NameId: 'vm-01', SystemId: 'i-12345678'}"
	StartTime time.Time `json:"StartTime" validate:"required" example:"2024-08-27T10:00:00Z"` // Timezone: based on cloud-barista server location.

	Region            RegionInfo `json:"Region" validate:"required"`                          // example:"{Region: 'us-east-1', Zone: 'us-east-1a'}"
	ImageType         ImageType  `json:"ImageType" validate:"required" example:"PublicImage"` // PublicImage | MyImage
	ImageIId          IID        `json:"ImageIId" validate:"required"`                        // example:"{NameId: 'ami-12345678', SystemId: 'ami-12345678'}"
	VMSpecName        string     `json:"VMSpecName" validate:"required" example:"t2.micro"`   // instance type or flavour, etc... ex) t2.micro or f1.micro
	VpcIID            IID        `json:"VpcIID" validate:"required"`                          // example:"{NameId: 'vpc-01', SystemId: 'vpc-12345678'}"
	SubnetIID         IID        `json:"SubnetIID" validate:"required"`                       // example:"{NameId: 'subnet-01', SystemId: 'subnet-12345678'}"
	SecurityGroupIIds []IID      `json:"SecurityGroupIIds" validate:"required"`               // example:"[{NameId: 'sg-01', SystemId: 'sg-12345678'}]"

	KeyPairIId IID `json:"KeyPairIId" validate:"required"` // example:"{NameId: 'keypair-01', SystemId: 'keypair-12345678'}"

	RootDiskType   string `json:"RootDiskType" validate:"required" example:"gp2"`         // "gp2", "Premium SSD", ...
	RootDiskSize   string `json:"RootDiskSize" validate:"required" example:"50"`          // "default", "50", "1000" (unit is GB)
	RootDeviceName string `json:"RootDeviceName" validate:"required" example:"/dev/sda1"` // "/dev/sda1", ...

	DataDiskIIDs []IID `json:"DataDiskIIDs,omitempty" validate:"omitempty"` // example:"[{NameId: 'datadisk-01', SystemId: 'datadisk-12345678'}]"

	VMBootDisk  string `json:"VMBootDisk,omitempty" validate:"omitempty" example:"/dev/sda1"`  // Deprecated soon
	VMBlockDisk string `json:"VMBlockDisk,omitempty" validate:"omitempty" example:"/dev/sda2"` // Deprecated soon

	VMUserId     string `json:"VMUserId" validate:"required" example:"cb-user"`                     // cb-user or Administrator
	VMUserPasswd string `json:"VMUserPasswd,omitempty" validate:"omitempty" example:"password1234"` // Only for Windows

	NetworkInterface string `json:"NetworkInterface" validate:"required" example:"eni-12345678"`
	PublicIP         string `json:"PublicIP" validate:"required" example:"1.2.3.4"`
	PublicDNS        string `json:"PublicDNS,omitempty" validate:"omitempty" example:"ec2-1-2-3-4.compute-1.amazonaws.com"`
	PrivateIP        string `json:"PrivateIP" validate:"required" example:"192.168.1.1"`
	PrivateDNS       string `json:"PrivateDNS,omitempty" validate:"omitempty" example:"ip-192-168-1-1.ec2.internal"`

	Platform Platform `json:"Platform" validate:"required" example:"LINUX"` // LINUX | WINDOWS

	SSHAccessPoint string `json:"SSHAccessPoint,omitempty" validate:"omitempty" example:"10.2.3.2:22"` // Deprecated
	AccessPoint    string `json:"AccessPoint" validate:"required" example:"1.2.3.4:22"`                // 10.2.3.2:22, 123.456.789.123:432

	TagList      []KeyValue `json:"TagList,omitempty" validate:"omitempty"`      // example:"[{Key: 'Name', Value: 'MyVM'}]"
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty"` // example:"[{Key: 'Architecture', Value: 'x86_64'}]"
}

type VMHandler interface {
	StartVM(vmReqInfo VMReqInfo) (VMInfo, error)

	SuspendVM(vmIID IID) (VMStatus, error)
	ResumeVM(vmIID IID) (VMStatus, error)
	RebootVM(vmIID IID) (VMStatus, error)
	TerminateVM(vmIID IID) (VMStatus, error)

	ListVMStatus() ([]*VMStatusInfo, error)
	GetVMStatus(vmIID IID) (VMStatus, error)

	ListVM() ([]*VMInfo, error)
	GetVM(vmIID IID) (VMInfo, error)
}
