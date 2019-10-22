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
	Name          string
	Direction     string // GCP 는 하나에 한개의 Direction만 생성/조회 가능
	SecurityRules *[]SecurityRuleInfo
}

type SecurityRuleInfo struct {
	FromPort   string
	ToPort     string
	IPProtocol string
	Direction  string
}

type SecurityInfo struct {
	Id            string
	Name          string
	Direction     string // GCP 는 하나에 한개의 Direction만 생성/조회 가능
	SecurityRules *[]SecurityRuleInfo

	KeyValueList []KeyValue
}

type SecurityHandler interface {
	CreateSecurity(securityReqInfo SecurityReqInfo) (SecurityInfo, error)
	ListSecurity() ([]*SecurityInfo, error)
	GetSecurity(securityID string) (SecurityInfo, error)
	DeleteSecurity(securityID string) (bool, error)
}
