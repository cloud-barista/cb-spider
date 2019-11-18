// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by powerkim@etri.re.kr, 2019.06.

package resources

import (
	"time"
)

type VMReqInfo struct {
	VMName string

	ImageId            string
	VirtualNetworkId   string
	NetworkInterfaceId string
	PublicIPId         string
	SecurityGroupIds   []string

	VMSpecId string

	KeyPairName  string
	VMUserId     string
	VMUserPasswd string
}

type VMStatusInfo struct {
	VmId     string
	VmName   string
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
	NotExist  VMStatus = "NotExist" // VM does not exist

	Failed VMStatus = "Failed"
)

type RegionInfo struct {
	Region string
	Zone   string
}

type VMInfo struct {
	Name      string    // AWS,
	Id        string    // AWS,
	StartTime time.Time // Timezone: based on cloud-barista server location.

	Region           RegionInfo // AWS, ex) {us-east1, us-east1-c} or {ap-northeast-2}
	ImageId          string
	VMSpecId         string   // AWS, instance type or flavour, etc... ex) t2.micro or f1.micro
	VirtualNetworkId string   // AWS, ex) subnet-8c4a53e4
	SecurityGroupIds []string // AWS, ex) sg-0b7452563e1121bb6

	NetworkInterfaceId string // ex) eth0
	PublicIP           string // ex) AWS, 13.125.43.21
	PublicDNS          string // ex) AWS, ec2-13-125-43-0.ap-northeast-2.compute.amazonaws.com
	PrivateIP          string // ex) AWS, ip-172-31-4-60.ap-northeast-2.compute.internal
	PrivateDNS         string // ex) AWS, 172.31.4.60

	KeyPairName  string // ex) AWS, powerkimKeyPair
	VMUserId     string // ex) user1
	VMUserPasswd string

	VMBootDisk  string // ex) /dev/sda1
	VMBlockDisk string // ex)

	KeyValueList []KeyValue
}

type VMHandler interface {
	StartVM(vmReqInfo VMReqInfo) (VMInfo, error)

	SuspendVM(vmID string) (VMStatus, error)
	ResumeVM(vmID string) (VMStatus, error)
	RebootVM(vmID string) (VMStatus, error)
	TerminateVM(vmID string) (VMStatus, error)

	ListVMStatus() ([]*VMStatusInfo, error)
	GetVMStatus(vmID string) (VMStatus, error)

	ListVM() ([]*VMInfo, error)
	GetVM(vmID string) (VMInfo, error)
}
