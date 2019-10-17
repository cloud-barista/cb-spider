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
     Name string
}

type VNetworkInfo struct {
     Id   string 
     Name string 
     AddressPrefix string 
     Status string  

     KeyValueList []KeyValue 
}

type VNetworkHandler interface {
	CreateVNetwork(vNetworkReqInfo VNetworkReqInfo) (VNetworkInfo, error)
	ListVNetwork() ([]*VNetworkInfo, error)
	GetVNetwork(vNetworkID string) (VNetworkInfo, error)
	DeleteVNetwork(vNetworkID string) (bool, error)
}
