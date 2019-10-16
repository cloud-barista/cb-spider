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

type SecurityReqInfo struct {
	//2차 인터페이스
	Name          string
	SecurityRules *[]SecurityRuleInfo

	// @todo - 삭제예정(1차 인터페이스 잔여 필드)
	Id                  string
	GroupName           string              //AWS
	Description         string              //AWS
	VpcId               string              //AWS
	IPPermissions       []*SecurityRuleInfo //AWS:InBounds
	IPPermissionsEgress []*SecurityRuleInfo //AWS:OutBounds
}

type SecurityRuleInfo struct {
	//2차 인터페이스
	FromPort   string
	ToPort     string
	IPProtocol string // tcp | udp | icmp | ...
	Direction  string // inbound | outbound

	// @todo - 삭제예정(1차 인터페이스 잔여 필드)
	Cidr string
}

type SecurityInfo struct {
	//2차 인터페이스
	Id            string
	Name          string
	SecurityRules *[]SecurityRuleInfo

	KeyValueList []KeyValue

	// @todo - 삭제예정(1차 인터페이스 잔여 필드)
	GroupName           string              //AWS
	GroupID             string              //AWS
	IPPermissions       []*SecurityRuleInfo //AWS:InBounds
	IPPermissionsEgress []*SecurityRuleInfo //AWS:OutBounds
	Description         string              //AWS
	VpcID               string              //AWS
	OwnerID             string              //AWS, Azure & OpenStack은 TenantId
}

type SecurityHandler interface {
	CreateSecurity(securityReqInfo SecurityReqInfo) (SecurityInfo, error)
	ListSecurity() ([]*SecurityInfo, error)
	GetSecurity(securityID string) (SecurityInfo, error)
	DeleteSecurity(securityID string) (bool, error)
}
