// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2020.04.
// by CB-Spider Team, 2019.06.

package resources

type SecurityReqInfo struct {
	IId IID // {NameId, SystemId}

	VpcIID        IID    // {NameId, SystemId}
	Direction     string // @todo used??
	SecurityRules *[]SecurityRuleInfo
}

type SecurityRuleInfo struct {
	FromPort   string
	ToPort     string
	IPProtocol string
	Direction  string
}

type SecurityInfo struct {
	IId IID // {NameId, SystemId}

	VpcIID        IID    // {NameId, SystemId}
	Direction     string // @todo userd??
	SecurityRules *[]SecurityRuleInfo

	KeyValueList []KeyValue
}

type SecurityHandler interface {
	CreateSecurity(securityReqInfo SecurityReqInfo) (SecurityInfo, error)
	ListSecurity() ([]*SecurityInfo, error)
	GetSecurity(securityIID IID) (SecurityInfo, error)
	DeleteSecurity(securityIID IID) (bool, error)
}
