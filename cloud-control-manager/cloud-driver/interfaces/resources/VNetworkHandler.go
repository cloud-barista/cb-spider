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

// @todo add subnets by powerkim, 2020.04
type VNetworkReqInfo struct { 
	IId   IID       // {NameId, SystemId}
}

type VNetworkInfo struct {
	IId   IID       // {NameId, SystemId}

	AddressPrefix string 
	Status string  

	KeyValueList []KeyValue 
}

type VNetworkHandler interface {
	CreateVNetwork(vNetworkReqInfo VNetworkReqInfo) (VNetworkInfo, error)
	ListVNetwork() ([]*VNetworkInfo, error)
	GetVNetwork(vNetworkIID IID) (VNetworkInfo, error)
	DeleteVNetwork(vNetworkIID IID) (bool, error)
}
