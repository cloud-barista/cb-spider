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

type VNetworkReqInfo struct {
	//2차 인터페이스
	Name string // AWS

	// @todo - 삭제예정(1차 인터페이스 잔여 구조체)
	Id        string
	CidrBlock string // AWS
}

type VNetworkInfo struct {
	//2차 인터페이스
	Id            string
	Name          string
	AddressPrefix string
	Status        string

	KeyValueList []KeyValue

	// @todo - 삭제예정(1차 인터페이스 잔여 구조체)
	SubnetId  string // AWS에서는 이 필드에 Subnet ID할당
	CidrBlock string // AWS
	State     string // AWS

	MapPublicIpOnLaunch     bool   // AWS(향후 Map으로 변환?)
	AvailableIpAddressCount int64  // AWS(향후 Map으로 변환?)
	AvailabilityZone        string // AWS(향후 Map으로 변환?)
}

type VNetworkHandler interface {
	CreateVNetwork(vNetworkReqInfo VNetworkReqInfo) (VNetworkInfo, error)
	ListVNetwork() ([]*VNetworkInfo, error) //@TODO : 여러 VPC에 속한 Subnet 목록을 조회하게되는데... 입력 아규먼트가 없고 맥락상 CB-Vnet의 서브넷만 조회해야할지 결정이 필요함. 현재는 1차 버전의 변경될 I/F 문맥상 CB-Vnet으로 내부적으로 제한해서 구현했음.
	GetVNetwork(vNetworkID string) (VNetworkInfo, error)
	DeleteVNetwork(vNetworkID string) (bool, error)
}
