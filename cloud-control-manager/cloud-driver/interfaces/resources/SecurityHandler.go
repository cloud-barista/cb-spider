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
	Name string
	Id   string

	// @todo
	GroupName   string //AWS
	Description string //AWS
	VpcId       string //AWS

	IPPermissions       []*SecurityRuleInfo //AWS:InBounds
	IPPermissionsEgress []*SecurityRuleInfo //AWS:OutBounds
}

type SecurityInfo struct {
	Name string
	Id   string
	// @todo
	GroupName           string              //AWS
	GroupID             string              //AWS
	IPPermissions       []*SecurityRuleInfo //AWS:InBounds
	IPPermissionsEgress []*SecurityRuleInfo //AWS:OutBounds

	Description string //AWS
	VpcID       string //AWS
	OwnerID     string //AWS, Azure & OpenStack은 TenantId
}

type SecurityRuleInfo struct {
	FromPort   int64
	ToPort     int64
	IPProtocol string
	Cidr       string
}

type SecurityHandler interface {
	CreateSecurity(securityReqInfo SecurityReqInfo) (SecurityInfo, error)
	ListSecurity() ([]*SecurityInfo, error)
	GetSecurity(securityID string) (SecurityInfo, error)
	DeleteSecurity(securityID string) (bool, error)
}
