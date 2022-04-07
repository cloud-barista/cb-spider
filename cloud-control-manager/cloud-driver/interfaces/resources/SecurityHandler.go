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
	//Direction     string // To be deprecated
	SecurityRules *[]SecurityRuleInfo
}

// @definitionAlias cres.SecurityRuleInfo
type SecurityRuleInfo struct {
	Direction  string
	IPProtocol string
	FromPort   string
	ToPort     string
	CIDR       string
}

type SecurityInfo struct {
	IId IID // {NameId, SystemId}

	VpcIID        IID    // {NameId, SystemId}
	//Direction     string // @todo userd??
	SecurityRules *[]SecurityRuleInfo

	KeyValueList []KeyValue
}

type SecurityHandler interface {
	CreateSecurity(securityReqInfo SecurityReqInfo) (SecurityInfo, error)
	ListSecurity() ([]*SecurityInfo, error)
	GetSecurity(securityIID IID) (SecurityInfo, error)
	DeleteSecurity(securityIID IID) (bool, error)

	//AddRules(sgIID IID, securityRules *[]SecurityRuleInfo) (SecurityInfo, error)
	//RemoveRules(sgIID IID, securityRules *[]SecurityRuleInfo) (bool, error)
}
