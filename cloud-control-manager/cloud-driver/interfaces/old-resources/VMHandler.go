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
	//2차 인터페이스
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

	// @todo - 삭제예정(1차 인터페이스 잔여 필드)
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

//2차 인터페이스
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

//2차 인터페이스
type RegionInfo struct {
	Region string
	Zone   string
}

type VMInfo struct {
	//2차 인터페이스
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

	// @todo - 삭제예정(1차 인터페이스 잔여 필드)
	ImageID      string // AWS, ex) ami-047f7b46bd6dd5d84 or projects/gce-uefi-images/global/images/centos-7-v20190326
	SpecID       string // AWS, instance type or flavour, etc... ex) t2.micro or f1-micro
	VNetworkID   string // AWS, ex) vpc-23ed0a4b
	SubNetworkID string // AWS, ex) subnet-8c4a53e4
	SecurityID   string // AWS, ex) sg-0b7452563e1121bb6 - @todo AWS는 배열임

	VNIC         string // ex) eth0
	KeyPairID    string // ex) AWS, powerkimKeyPair
	GuestUserID  string // ex) user1
	GuestUserPwd string

	GuestBootDisk  string // ex) /dev/sda1
	GuestBlockDisk string // ex)

	AdditionalInfo string // Any information to be good for users and developers.
}

// @todo - 삭제예정(1차 인터페이스 잔여 구조체)
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
