// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2020.08.
// by CB-Spider Team, 2020.04.
// by CB-Spider Team, 2019.06.

package resources

type VPCReqInfo struct {
	IId            IID // {NameId, SystemId}
	IPv4_CIDR      string
	SubnetInfoList []SubnetInfo

	TagList []KeyValue
}

type VPCInfo struct {
	IId            IID          `json:"IId" validate:"required"` // {NameId, SystemId}
	IPv4_CIDR      string       `json:"IPv4_CIDR" validate:"required" example:"10.0.0.0/16" description:"The IPv4 CIDR block for the VPC"`
	SubnetInfoList []SubnetInfo `json:"SubnetInfoList" validate:"required" description:"A list of subnet information associated with this VPC"`

	TagList      []KeyValue `json:"TagList,omitempty" validate:"omitempty" description:"A list of tags associated with this VPC"`
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty" description:"Additional key-value pairs associated with this VPC"`
}

type SubnetInfo struct {
	IId       IID    `json:"IId" validate:"required"` // {NameId, SystemId}
	Zone      string `json:"Zone" validate:"required" example:"us-east-1a"`
	IPv4_CIDR string `json:"IPv4_CIDR" validate:"required" example:"10.0.8.0/22" description:"The IPv4 CIDR block for the subnet"`

	TagList      []KeyValue `json:"TagList,omitempty" validate:"omitempty" description:"A list of tags associated with this subnet"`
	KeyValueList []KeyValue `json:"KeyValueList,omitempty" validate:"omitempty" description:"Additional key-value pairs associated with this subnet"`
}

type VPCHandler interface {
	ListIID() ([]*IID, error)
	CreateVPC(vpcReqInfo VPCReqInfo) (VPCInfo, error)
	ListVPC() ([]*VPCInfo, error)
	GetVPC(vpcIID IID) (VPCInfo, error)
	DeleteVPC(vpcIID IID) (bool, error)

	AddSubnet(vpcIID IID, subnetInfo SubnetInfo) (VPCInfo, error)
	RemoveSubnet(vpcIID IID, subnetIID IID) (bool, error)
}
