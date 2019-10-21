package resources

import (
	"errors"
	"fmt"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/networking/v2/networks"
	"github.com/rackspace/gophercloud/openstack/networking/v2/subnets"
	"github.com/rackspace/gophercloud/pagination"
)

type OpenStackVNetworkHandler struct {
	Client *gophercloud.ServiceClient
}

func setterVNet(vNet subnets.Subnet) *irs.VNetworkInfo {
	vNetWorkInfo := &irs.VNetworkInfo{
		Id:            vNet.ID,
		Name:          vNet.Name,
		AddressPrefix: fmt.Sprintf("%v", vNet.CIDR),
		//Status:       ,
	}
	return vNetWorkInfo
}

func (vNetworkHandler *OpenStackVNetworkHandler) CreateVNetwork(vNetworkReqInfo irs.VNetworkReqInfo) (irs.VNetworkInfo, error) {

	// Check VNet Exists
	// 기본 가상 네트워크가 생성되지 않았을 경우 디폴트 네트워크 생성 (CB-VNet)
	networkId, err := vNetworkHandler.GetCBVNetId()
	if err != nil {
		return irs.VNetworkInfo{}, err
	}

	if networkId == "" {
		// Create vNetwork
		createOpts := networks.CreateOpts{
			Name: vNetworkReqInfo.Name,
		}

		network, err := networks.Create(vNetworkHandler.Client, createOpts).Extract()
		if err != nil {
			return irs.VNetworkInfo{}, err
		}
		networkId = network.ID
	}

	// 서브넷 CIDR 할당
	list, err := vNetworkHandler.ListVNetwork()
	if err != nil {
		return irs.VNetworkInfo{}, err
	}
	subnetCIDR, err := CreateSubnetCIDR(list)

	// Create Subnet
	subnetCreateOpts := subnets.CreateOpts{
		NetworkID:      networkId,
		CIDR:           *subnetCIDR,
		IPVersion:      subnets.IPv4,
		Name:           vNetworkReqInfo.Name,
		DNSNameservers: []string{DNSNameservers},
	}

	subnet, err := subnets.Create(vNetworkHandler.Client, subnetCreateOpts).Extract()
	if err != nil {
		return irs.VNetworkInfo{}, err
	}

	vNetIPInfo, err := vNetworkHandler.GetVNetwork(subnet.ID)
	if err != nil {
		return irs.VNetworkInfo{}, err
	}
	return vNetIPInfo, nil
}

func (vNetworkHandler *OpenStackVNetworkHandler) ListVNetwork() ([]*irs.VNetworkInfo, error) {
	// 기본 가상 네트워크 아이디 정보 가져오기
	networkId, err := vNetworkHandler.GetCBVNetId()
	if networkId == "" {
		return nil, errors.New(fmt.Sprintf("failed to get virtual network by name, name: %s", CBVirutalNetworkName))
	}

	var vNetworkIList []*irs.VNetworkInfo

	listOpts := subnets.ListOpts{
		NetworkID: networkId,
	}
	pager := subnets.List(vNetworkHandler.Client, listOpts)
	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		// Get vNetwork
		list, err := subnets.ExtractSubnets(page)
		if err != nil {
			return false, err
		}
		// Add to List
		for _, n := range list {
			vNetworkInfo := setterVNet(n)
			vNetworkIList = append(vNetworkIList, vNetworkInfo)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return vNetworkIList, nil
}

func (vNetworkHandler *OpenStackVNetworkHandler) GetVNetwork(vNetworkID string) (irs.VNetworkInfo, error) {
	networkId, _ := vNetworkHandler.GetCBVNetId()
	if networkId == "" {
		return irs.VNetworkInfo{}, errors.New(fmt.Sprintf("failed to get virtual network by name, name: %s", CBVirutalNetworkName))
	}

	network, err := subnets.Get(vNetworkHandler.Client, vNetworkID).Extract()
	if err != nil {
		return irs.VNetworkInfo{}, err
	}

	vNetworkInfo := setterVNet(*network)
	return *vNetworkInfo, nil
}

func (vNetworkHandler *OpenStackVNetworkHandler) DeleteVNetwork(vNetworkID string) (bool, error) {
	networkId, _ := vNetworkHandler.GetCBVNetId()
	if networkId == "" {
		return false, errors.New(fmt.Sprintf("failed to get virtual network by name, name: %s", CBVirutalNetworkName))
	}

	err := subnets.Delete(vNetworkHandler.Client, vNetworkID).ExtractErr()
	if err != nil {
		return false, err
	}
	return true, nil
}

// 기본 가상 네트워크(CB-VNet) Id 정보 조회
func (vNetworkHandler *OpenStackVNetworkHandler) GetCBVNetId() (string, error) {
	listOpt := networks.ListOpts{
		Name: CBVirutalNetworkName,
	}

	var vNetworkId string

	pager := networks.List(vNetworkHandler.Client, listOpt)
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		// Get vNetwork
		list, err := networks.ExtractNetworks(page)
		if err != nil {
			return false, err
		}
		// Add to List
		for _, n := range list {
			vNetworkId = n.ID
		}
		return true, nil
	})
	if err != nil {
		return "", err
	}

	return vNetworkId, nil
}
