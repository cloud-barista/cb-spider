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

type PublicIPReqInfo struct {
	Name string
	Id   string
	// @todo
}

type PublicIPInfo struct {
	Name string // AWS
	Id   string
	// @todo

	Domain                  string // AWS
	PublicIp                string // AWS
	PublicIpv4Pool          string // AWS
	AllocationId            string // AWS:할당ID
	AssociationId           string // AWS:연결ID
	InstanceId              string // AWS:연결된 VM
	NetworkInterfaceId      string // AWS:연결된 Nic
	NetworkInterfaceOwnerId string // AWS
	PrivateIpAddress        string // AWS
}

type PublicIPHandler interface {
	CreatePublicIP(publicIPReqInfo PublicIPReqInfo) (PublicIPInfo, error)
	ListPublicIP() ([]*PublicIPInfo, error)
	GetPublicIP(publicIPID string) (PublicIPInfo, error)
	DeletePublicIP(publicIPID string) (bool, error)
}
