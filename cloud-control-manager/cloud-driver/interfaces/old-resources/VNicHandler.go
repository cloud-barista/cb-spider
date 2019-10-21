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
	//2차 인터페이스
	Name             string
	VNetName         string
	SecurityGroupIds []string
	PublicIPid       string

	// @todo - 삭제예정(1차 인터페이스 잔여 구조체)
	Id string
}

//2차 인터페이스 필드 반영
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
