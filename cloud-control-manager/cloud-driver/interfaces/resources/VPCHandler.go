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

type VPCReqInfo struct { 
	IId   IID       // {NameId, SystemId}
	IPv4_CIDR string 
	SubnetInfoList []SubnetInfo 
}

type VPCInfo struct {
	IId   IID       // {NameId, SystemId}
	IPv4_CIDR string 
	SubnetInfoList []SubnetInfo 

	KeyValueList []KeyValue 
}

type SubnetInfo struct {
	IId   IID       // {NameId, SystemId}
	IPv4_CIDR string 

	KeyValueList []KeyValue 
}

type VPCHandler interface {
	CreateVPC(vpcReqInfo VPCReqInfo) (VPCInfo, error)
	ListVPC() ([]*VPCInfo, error)
	GetVPC(vpcIID IID) (VPCInfo, error)
	DeleteVPC(vpcIID IID) (bool, error)
}
