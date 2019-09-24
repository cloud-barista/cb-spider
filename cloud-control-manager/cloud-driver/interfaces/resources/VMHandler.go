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
	// region/zone: Do not specify, this driver already knew these in Connection.

	Name string

	ImageInfo    ImageInfo
	VNetworkInfo VNetworkInfo
	SecurityInfo SecurityInfo
	KeyPairInfo  KeyPairInfo
	SpecID       string // instance type or flavour, etc...
	vNicInfo     VNicInfo
	PublicIPInfo PublicIPInfo
	LoginInfo    LoginInfo
}

type VMStatusInfo struct {
	VmId     string
	VmStatus VMStatus
}

// GO do not support Enum. So, define like this.
type VMStatus string

const (
	pending VMStatus = "PENDING" // from launch, suspended to running
	running VMStatus = "RUNNING"

	suspending VMStatus = "SUSPENDING" // from running to suspended
	suspended  VMStatus = "SUSPENDED"

	rebooting VMStatus = "REBOOTING" // from running to running

	termiating VMStatus = "TERMINATING" // from running, suspended to terminated
	termiated  VMStatus = "TERMINATED"
)

type RegionInfo struct {
	Region string
	Zone   string
}

type VMInfo struct {
	Name      string    // AWS,
	Id        string    // AWS,
	StartTime time.Time // Timezone: based on cloud-barista server location.

	Region       RegionInfo // AWS, ex) {us-east1, us-east1-c} or {ap-northeast-2}
	ImageID      string     // AWS, ex) ami-047f7b46bd6dd5d84 or projects/gce-uefi-images/global/images/centos-7-v20190326
	SpecID       string     // AWS, instance type or flavour, etc... ex) t2.micro or f1-micro
	VNetworkID   string     // AWS, ex) vpc-23ed0a4b
	SubNetworkID string     // AWS, ex) subnet-8c4a53e4
	SecurityID   string     // AWS, ex) sg-0b7452563e1121bb6 - @todo AWS는 배열임

	VNIC       string // ex) eth0
	PublicIP   string // ex) AWS, 13.125.43.21
	PublicDNS  string // ex) AWS, ec2-13-125-43-0.ap-northeast-2.compute.amazonaws.com
	PrivateIP  string // ex) AWS, ip-172-31-4-60.ap-northeast-2.compute.internal
	PrivateDNS string // ex) AWS, 172.31.4.60

	KeyPairID    string // ex) AWS, powerkimKeyPair
	GuestUserID  string // ex) user1
	GuestUserPwd string

	GuestBootDisk  string // ex) /dev/sda1
	GuestBlockDisk string // ex)

	AdditionalInfo string // Any information to be good for users and developers.
}

type LoginInfo struct {
	AdminUsername string
	AdminPassword string
}

type VMHandler interface {
	StartVM(vmReqInfo VMReqInfo) (VMInfo, error)
	SuspendVM(vmID string)
	ResumeVM(vmID string)
	RebootVM(vmID string)
	TerminateVM(vmID string)

	ListVMStatus() []*VMStatusInfo
	GetVMStatus(vmID string) VMStatus

	ListVM() []*VMInfo
	GetVM(vmID string) VMInfo
}
