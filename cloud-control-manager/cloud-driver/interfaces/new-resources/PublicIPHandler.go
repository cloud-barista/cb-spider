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
}

type PublicIPHandler interface {
	CreatePublicIP(publicIPReqInfo PublicIPReqInfo) (PublicIPInfo, error)
	ListPublicIP() ([]*PublicIPInfo, error)
	GetPublicIP(publicIPID string) (PublicIPInfo, error)
	DeletePublicIP(publicIPID string) (bool, error)
}
