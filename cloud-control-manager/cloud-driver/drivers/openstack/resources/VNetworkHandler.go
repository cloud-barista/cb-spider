package resources

import (
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/networking/v2/networks"
	"github.com/rackspace/gophercloud/openstack/networking/v2/subnets"
	"github.com/rackspace/gophercloud/pagination"
)

/*var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}*/

type OpenStackVNetworkHandler struct {
	Client *gophercloud.ServiceClient
}

type VNetworkInfo struct {
	ID           string
	Name         string
	AdminStateUp bool
	Status       string
	Subnets      []string
	TenantID     string
	Shared       bool
}

func (vNetworkInfo *VNetworkInfo) setter(network networks.Network) *VNetworkInfo {
	vNetworkInfo.ID = network.ID
	vNetworkInfo.Name = network.Name
	vNetworkInfo.AdminStateUp = network.AdminStateUp
	vNetworkInfo.Status = network.Status
	vNetworkInfo.Subnets = network.Subnets
	vNetworkInfo.TenantID = network.TenantID
	vNetworkInfo.Shared = network.Shared

	return vNetworkInfo
}

func (vNetworkHandler *OpenStackVNetworkHandler) CreateVNetwork(vNetworkReqInfo irs.VNetworkReqInfo) (irs.VNetworkInfo, error) {

	// @TODO: vNetwork 생성 요청 파라미터 정의 필요
	type IPPool struct {
		Start string
		End   string
	}
	type VNetworkReqInfo struct {
		Name          string
		AdminStateUp  bool
		CIDR          string
		IPVersion     int
		SubnetName    string
		IPPool        []IPPool
		DNSNameServer []string
	}

	reqInfo := VNetworkReqInfo{
		Name:         vNetworkReqInfo.Name,
		AdminStateUp: *networks.Up,
		CIDR:         "30.0.0.0/24",
		IPVersion:    subnets.IPv4,
		SubnetName:   "default",
		IPPool: []IPPool{
			{
				Start: "30.0.0.2",
				End:   "30.0.0.254",
			},
		},
		DNSNameServer: []string{
			"8.8.8.8",
		},
	}

	// Create vNetwork
	createOpts := networks.CreateOpts{
		Name:         reqInfo.Name,
		AdminStateUp: &reqInfo.AdminStateUp,
	}
	network, err := networks.Create(vNetworkHandler.Client, createOpts).Extract()
	if err != nil {
		return irs.VNetworkInfo{}, err
	}
	spew.Dump(network)

	// Set IPPool
	var AllocationPool []subnets.AllocationPool
	for _, IPPool := range reqInfo.IPPool {
		pool := subnets.AllocationPool{
			Start: IPPool.Start,
			End:   IPPool.End,
		}
		AllocationPool = append(AllocationPool, pool)
	}

	// Create Subnet
	subnetCreateOpts := subnets.CreateOpts{
		NetworkID:       network.ID,
		CIDR:            reqInfo.CIDR,
		IPVersion:       reqInfo.IPVersion,
		Name:            reqInfo.SubnetName,
		AllocationPools: AllocationPool,
		DNSNameservers:  reqInfo.DNSNameServer,
	}

	subnet, err := subnets.Create(vNetworkHandler.Client, subnetCreateOpts).Extract()
	if err != nil {
		return irs.VNetworkInfo{}, err
	}
	spew.Dump(subnet)

	// @TODO: 생성된 vNetwork 정보 리턴
	return irs.VNetworkInfo{Id: network.ID, SubnetId: subnet.ID}, nil
}

func (vNetworkHandler *OpenStackVNetworkHandler) ListVNetwork() ([]*irs.VNetworkInfo, error) {
	var vNetworkIList []*VNetworkInfo

	pager := networks.List(vNetworkHandler.Client, nil)
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		// Get vNetwork
		list, err := networks.ExtractNetworks(page)
		if err != nil {
			return false, err
		}
		// Add to List
		for _, n := range list {
			vNetworkInfo := new(VNetworkInfo).setter(n)
			vNetworkIList = append(vNetworkIList, vNetworkInfo)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	spew.Dump(vNetworkIList)
	return nil, nil
}

func (vNetworkHandler *OpenStackVNetworkHandler) GetVNetwork(vNetworkID string) (irs.VNetworkInfo, error) {
	network, err := networks.Get(vNetworkHandler.Client, vNetworkID).Extract()
	if err != nil {
		return irs.VNetworkInfo{}, err
	}

	if network != nil {
		vNetworkInfo := new(VNetworkInfo).setter(*network)
		spew.Dump(vNetworkInfo)
	}

	return irs.VNetworkInfo{}, nil
}

func (vNetworkHandler *OpenStackVNetworkHandler) DeleteVNetwork(vNetworkID string) (bool, error) {
	err := networks.Delete(vNetworkHandler.Client, vNetworkID).ExtractErr()
	if err != nil {
		return false, err
	}
	return true, nil
}
