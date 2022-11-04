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
	RootDiskSize string // "", "default", "50", "1000" (GB)

	DataDiskIIDs []IID

	VMUserId     string
	VMUserPasswd string
	WindowsType  bool
}

type VMStatusInfo struct {
	IId      IID // {NameId, SystemId}
	VmStatus VMStatus
}

// GO do not support Enum. So, define like this.
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
	Region string
	Zone   string
}

type VMInfo struct {
	IId       IID       // {NameId, SystemId}
	StartTime time.Time // Timezone: based on cloud-barista server location.

	Region            RegionInfo //  ex) {us-east1, us-east1-c} or {ap-northeast-2}
	ImageType         ImageType  // PublicImage | MyImage
	ImageIId          IID
	VMSpecName        string //  instance type or flavour, etc... ex) t2.micro or f1.micro
	VpcIID            IID
	SubnetIID         IID   // AWS, ex) subnet-8c4a53e4
	SecurityGroupIIds []IID // AWS, ex) sg-0b7452563e1121bb6

	KeyPairIId IID

	RootDiskType   string // "SSD(gp2)", "Premium SSD", ...
	RootDiskSize   string // "default", "50", "1000" (GB)
	RootDeviceName string // "/dev/sda1", ...

	DataDiskIIDs []IID

	VMBootDisk  string // Deprecated soon
	VMBlockDisk string // Deprecated soon

	VMUserId     string // ex) user1
	VMUserPasswd string

	NetworkInterface string // ex) eth0
	PublicIP         string
	PublicDNS        string
	PrivateIP        string
	PrivateDNS       string

	SSHAccessPoint string // ex) 10.2.3.2:22, 123.456.789.123:4321

	KeyValueList []KeyValue
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
