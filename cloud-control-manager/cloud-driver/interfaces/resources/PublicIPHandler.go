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
	Name      string // AWS : Name Tag대신 AllocationId를 리턴 함.(편의를 위해 Name 정보는 KeyValueList에 전달 함.), OpenStack : 381a10f8-5831-4822-8388-922673addde4(ID), Cloudit : 182.252.135.44(IP)
	Id        string // AWS : Name에는 NameId가 설정되기 때문에 삭제를 위해 Id 필드 추가
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
