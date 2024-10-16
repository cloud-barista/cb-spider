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

	VpcIID IID // {NameId, SystemId}
	//Direction     string // To be deprecated
	SecurityRules *[]SecurityRuleInfo

	TagList []KeyValue
}

type SecurityRuleInfo struct {
	Direction  string `json:"Direction" validate:"required" example:"inbound"`         // inbound or outbound
	IPProtocol string `json:"IPProtocol" validate:"required" example:"TCP"`            // TCP, UDP, ICMP, ALL
	FromPort   string `json:"FromPort" validate:"required" example:"22"`               // TCP, UDP: 1~65535, ICMP, ALL: -1
	ToPort     string `json:"ToPort" validate:"required" example:"22"`                 // TCP, UDP: 1~65535, ICMP, ALL: -1
	CIDR       string `json:"CIDR,omitempty" validate:"omitempty" example:"0.0.0.0/0"` // if not specified, defaults to 0.0.0.0/0
}

type SecurityInfo struct {
	IId IID `json:"IId" validate:"required"` // {NameId, SystemId}

	VpcIID IID `json:"VpcIID" validate:"required"` // {NameId, SystemId}

	SecurityRules *[]SecurityRuleInfo `json:"SecurityRules" validate:"required" description:"A list of security rules applied to this security group"`

	TagList      []KeyValue `json:"TagList,omitempty" validate:"omitempty" description:"A list of tags associated with this security group"`
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty" description:"Additional key-value pairs associated with this security group"`
}

type SecurityHandler interface {
	ListIID() ([]*IID, error)

	CreateSecurity(securityReqInfo SecurityReqInfo) (SecurityInfo, error)
	ListSecurity() ([]*SecurityInfo, error)
	GetSecurity(securityIID IID) (SecurityInfo, error)
	DeleteSecurity(securityIID IID) (bool, error)

	AddRules(sgIID IID, securityRules *[]SecurityRuleInfo) (SecurityInfo, error)
	RemoveRules(sgIID IID, securityRules *[]SecurityRuleInfo) (bool, error)
}
