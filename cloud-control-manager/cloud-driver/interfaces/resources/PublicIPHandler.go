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
	Name         string
	KeyValueList []KeyValue
}

type PublicIPInfo struct {
	Name      string
	PublicIP  string
	OwnedVMID string
	Status    string

	KeyValueList []KeyValue
<<<<<<< HEAD

	// @todo - 삭제예정(1차 인터페이스 잔여 필드)
	Id                      string
	Domain                  string // AWS
	PublicIp                string // AWS
	PublicIpv4Pool          string // AWS
	AllocationId            string // AWS:할당ID
	AssociationId           string // AWS:연결ID
	InstanceId              string // AWS:연결된 VM
	NetworkInterfaceId      string // AWS:연결된 Nic
	NetworkInterfaceOwnerId string // AWS
	PrivateIpAddress        string // AWS

	Region            string // GCP
	CreationTimestamp string // GCP
	Address           string // GCP
	NetworkTier       string // GCP : PREMIUM, STANDARD
	AddressType       string // GCP : External, INTERNAL, UNSPECIFIED_TYPE
	Status            string // GCP : IN_USE, RESERVED, RESERVING
=======
>>>>>>> 1cefbbb819ec4faf9ba803e90a1199db8bc27f6c
}

type PublicIPHandler interface {
	CreatePublicIP(publicIPReqInfo PublicIPReqInfo) (PublicIPInfo, error)
	ListPublicIP() ([]*PublicIPInfo, error)
	GetPublicIP(publicIPID string) (PublicIPInfo, error)
	DeletePublicIP(publicIPID string) (bool, error)
}
