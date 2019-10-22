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

type VNicReqInfo struct {
	Name             string
	VNetName         string
	VNetId           string // ex) OpenStack, 0b108f81-f2a7-4fb7-bae3-fe5544b0b1d0
	SecurityGroupIds []string
	PublicIPid       string
}

type VNicInfo struct {
	Id               string
	Name             string
	PublicIP         string
	MacAddress       string
	OwnedVMID        string
	SecurityGroupIds []string
	Status           string

	KeyValueList []KeyValue
}

type VNicHandler interface {
	CreateVNic(vNicReqInfo VNicReqInfo) (VNicInfo, error)
	ListVNic() ([]*VNicInfo, error)
	GetVNic(vNicID string) (VNicInfo, error)
	DeleteVNic(vNicID string) (bool, error)
}
