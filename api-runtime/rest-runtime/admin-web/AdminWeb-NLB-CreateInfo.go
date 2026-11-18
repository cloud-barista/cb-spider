// Proxy for AdminWeb(Multi-Connection NLB) of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista

package adminweb

import (
	"sync"

	cres "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/labstack/echo/v4"
)

//====================================== Multi-Connection NLB: data for the Add NLB dialog
//
// The Add NLB dialog needs, for every connection, whether NLB is supported and its VPCs and VMs.
// They are fetched on the server side and streamed per connection (see streamPerConnection).

// nlbCreateInfoVPC is the part of a VPC the dialog uses (same field names as cres.VPCInfo).
type nlbCreateInfoVPC struct {
	IId       cres.IID `json:"IId"`
	IPv4_CIDR string   `json:"IPv4_CIDR"`
}

// nlbCreateInfoVM is the part of a VM the dialog uses (same field names as cres.VMInfo).
type nlbCreateInfoVM struct {
	IId       cres.IID `json:"IId"`
	VpcIID    cres.IID `json:"VpcIID"`
	PrivateIP string   `json:"PrivateIP"`
}

// nlbCreateInfo is one streamed line: the Add NLB dialog data of one connection.
type nlbCreateInfo struct {
	Connection string             `json:"connection"`
	Supported  bool               `json:"supported"`       // false: the CSP does not support NLB
	VPCs       []nlbCreateInfoVPC `json:"vpcs"`            // VPCs of the connection
	VMs        []nlbCreateInfoVM  `json:"vms"`             // VMs of the connection (empty when they cannot be listed)
	Error      string             `json:"error,omitempty"` // VPC list error: the connection is skipped
}

// fetchNLBCreateInfo gets the Add NLB dialog data of one connection.
// VPCs and VMs are fetched at the same time; VMs are optional, so a VM list error leaves them empty.
func fetchNLBCreateInfo(connName string) nlbCreateInfo {
	info := nlbCreateInfo{Connection: connName, VPCs: []nlbCreateInfoVPC{}, VMs: []nlbCreateInfoVM{}}
	if fetchUnsupportedResources(connName)["nlb"] {
		return info
	}
	info.Supported = true

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		vpcs, err := fetchVPCs(connName)
		if err != nil {
			info.Error = err.Error()
			return
		}
		for _, vpc := range vpcs {
			if vpc != nil {
				info.VPCs = append(info.VPCs, nlbCreateInfoVPC{IId: vpc.IId, IPv4_CIDR: vpc.IPv4_CIDR})
			}
		}
	}()
	go func() {
		defer wg.Done()
		vms, err := fetchVMs(connName)
		if err != nil {
			cblog.Info("NLB create info: VMs of " + connName + " are not listed: " + err.Error())
			return
		}
		for _, vm := range vms {
			if vm != nil {
				info.VMs = append(info.VMs, nlbCreateInfoVM{IId: vm.IId, VpcIID: vm.VpcIID, PrivateIP: vm.PrivateIP})
			}
		}
	}()
	wg.Wait()
	return info
}

// NLBCreateInfo streams the Add NLB dialog data of the given connections, one JSON line each,
// in the order the connections finish.
// GET /spider/adminweb/nlb-createinfo?ids=conn1,conn2
func NLBCreateInfo(c echo.Context) error {
	return streamPerConnection(c, connectionIDs(c), func(connName string) any { return fetchNLBCreateInfo(connName) })
}
