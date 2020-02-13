package resources

import (
	"errors"
	"fmt"
	"github.com/Azure/go-autorest/autorest/to"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/rackspace/gophercloud/openstack/networking/v2/networks"
	"github.com/rackspace/gophercloud/openstack/networking/v2/subnets"
)

type OpenStackVNetworkHandler struct {
	Client *gophercloud.ServiceClient
}

func setterVNet(vNet subnets.Subnet) *irs.VNetworkInfo {
	vNetWorkInfo := &irs.VNetworkInfo{
		Id:            vNet.ID,
		Name:          vNet.Name,
		AddressPrefix: fmt.Sprintf("%v", vNet.CIDR),
	}
	return vNetWorkInfo
}

func (vNetworkHandler *OpenStackVNetworkHandler) CreateVNetwork(vNetworkReqInfo irs.VNetworkReqInfo) (irs.VNetworkInfo, error) {
	// Check VNet Exists
	// 기본 가상 네트워크가 생성되지 않았을 경우 디폴트 네트워크 생성 (CB-VNet)
	var isVNetCreated = true
	networkId, _ := GetCBVNetId(vNetworkHandler.Client)
	if networkId == "" {
		isVNetCreated = false

		// Create vNetwork
		createOpts := networks.CreateOpts{
			Name: CBVirutalNetworkName,
		}

		network, err := networks.Create(vNetworkHandler.Client, createOpts).Extract()
		if err != nil {
			return irs.VNetworkInfo{}, err
		}
		networkId = network.ID
	}

	// Check Subnet Exists
	subnetId, _ := subnets.IDFromName(vNetworkHandler.Client, vNetworkReqInfo.Name)
	if subnetId != "" {
		errMsg := fmt.Sprintf("Virtual Network with name %s already exist", vNetworkReqInfo.Name)
		createErr := errors.New(errMsg)
		return irs.VNetworkInfo{}, createErr
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

	// Create Router (Internet Gateway Router)
	var routerId string
	if !isVNetCreated {
		if router, err := vNetworkHandler.CreateRouter(vNetworkReqInfo.Name); err != nil {
			return irs.VNetworkInfo{}, err
		} else {
			routerId = *router
		}
	} else {
		if router, err := vNetworkHandler.GetRouterID(); err != nil {
			return irs.VNetworkInfo{}, err
		} else {
			routerId = *router
		}
	}

	// Create Router Interface
	if ok, err := vNetworkHandler.AddInterface(subnet.ID, routerId); !ok {
		return irs.VNetworkInfo{}, err
	}

	vNetIPInfo, err := vNetworkHandler.getVNetworkById(subnet.ID)
	if err != nil {
		return irs.VNetworkInfo{}, err
	}
	return vNetIPInfo, nil
}

func (vNetworkHandler *OpenStackVNetworkHandler) ListVNetwork() ([]*irs.VNetworkInfo, error) {
	// 기본 가상 네트워크 아이디 정보 가져오기
	networkId, err := GetCBVNetId(vNetworkHandler.Client)
	if networkId == "" {
		return nil, errors.New(fmt.Sprintf("failed to get virtual network by name, name: %s", CBVirutalNetworkName))
	}

	// 서브넷 목록 조회
	listOpts := subnets.ListOpts{
		NetworkID: networkId,
	}
	pager, err := subnets.List(vNetworkHandler.Client, listOpts).AllPages()
	if err != nil {
		return nil, err
	}
	network, err := subnets.ExtractSubnets(pager)
	if err != nil {
		return nil, err
	}

	// 서브넷 목록 정보 매핑
	vNetworkIList := make([]*irs.VNetworkInfo, len(network))
	for i, n := range network {
		vNetworkIList[i] = setterVNet(n)
	}
	return vNetworkIList, nil
}

func (vNetworkHandler *OpenStackVNetworkHandler) GetVNetwork(vNetworkNameId string) (irs.VNetworkInfo, error) {
	// 기본 가상 네트워크 아이디 정보 가져오기
	networkId, _ := GetCBVNetId(vNetworkHandler.Client)
	if networkId == "" {
		return irs.VNetworkInfo{}, errors.New(fmt.Sprintf("failed to get virtual network by name, name: %s", CBVirutalNetworkName))
	}

	subnetId, err := vNetworkHandler.getVNetworkIdByName(vNetworkNameId)
	if err != nil {
		return irs.VNetworkInfo{}, err
	}

	subnet, err := vNetworkHandler.getVNetworkById(subnetId)
	if err != nil {
		return irs.VNetworkInfo{}, err
	}

	return subnet, nil
}

func (vNetworkHandler *OpenStackVNetworkHandler) DeleteVNetwork(vNetworkNameId string) (bool, error) {
	// Get Subnet Info
	subnet, err := vNetworkHandler.GetVNetwork(vNetworkNameId)
	if err != nil {
		return false, err
	}

	// Get Router Info
	routerId, err := vNetworkHandler.GetRouterID()
	if err != nil {
		return false, err
	}

	// Delete Interface
	vNetworkHandler.DeleteInterface(subnet.Id, *routerId)

	// Delete Subnet
	networkId, _ := GetCBVNetId(vNetworkHandler.Client)
	if networkId == "" {
		return false, errors.New(fmt.Sprintf("failed to get virtual network by name, name: %s", CBVirutalNetworkName))
	}
	err = subnets.Delete(vNetworkHandler.Client, subnet.Id).ExtractErr()
	if err != nil {
		return false, err
	}

	// 마지막 서브넷을 삭제할 경우 VNetwork, Router 삭제
	list, err := vNetworkHandler.ListVNetwork()
	if err != nil {
		return false, err
	}
	if len(list) == 0 {

		// Delete Router
		if ok, err := vNetworkHandler.DeleteRouter(subnet.Name); !ok {
			return false, err
		}

		// Delete VNetwork
		networkId, err := GetCBVNetId(vNetworkHandler.Client)
		if err != nil {
			return false, err
		}
		err = networks.Delete(vNetworkHandler.Client, networkId).ExtractErr()
		if err != nil {
			return false, err
		}
	}

	return true, nil
}

// Router 이름 기준 ID 정보 조회
func (vNetworkHandler *OpenStackVNetworkHandler) GetRouterID() (*string, error) {
	var routerID string

	routerName := CBVirutalNetworkName + "-Router"
	listOpts := routers.ListOpts{
		Name: routerName,
	}
	pager, err := routers.List(vNetworkHandler.Client, listOpts).AllPages()
	if err != nil {
		return nil, err
	}
	routerList, err := routers.ExtractRouters(pager)
	if err != nil {
		return nil, err
	}

	if len(routerList) == 1 {
		routerID = routerList[0].ID
	}
	return &routerID, nil
}

func (vNetworkHandler *OpenStackVNetworkHandler) CreateRouter(subnetName string) (*string, error) {
	routerName := CBVirutalNetworkName + "-Router"
	createOpts := routers.CreateOpts{
		Name:         routerName,
		AdminStateUp: to.BoolPtr(true),
		GatewayInfo: &routers.GatewayInfo{
			NetworkID: CBGateWayId,
		},
	}

	// Create Router
	router, err := routers.Create(vNetworkHandler.Client, createOpts).Extract()
	if err != nil {
		return nil, err
	}
	spew.Dump(router)
	return &router.ID, nil
}

func (vNetworkHandler *OpenStackVNetworkHandler) DeleteRouter(subnetName string) (bool, error) {
	// Get Router Info
	routerId, err := vNetworkHandler.GetRouterID()
	if err != nil {
		return false, err
	}

	// Delete Router
	err = routers.Delete(vNetworkHandler.Client, *routerId).ExtractErr()
	if err != nil {
		return false, err
	}
	return true, nil
}

func (vNetworkHandler *OpenStackVNetworkHandler) AddInterface(subnetId string, routerId string) (bool, error) {
	createOpts := routers.InterfaceOpts{
		SubnetID: subnetId,
	}

	// Add Interface
	_, err := routers.AddInterface(vNetworkHandler.Client, routerId, createOpts).Extract()
	if err != nil {
		return false, err
	}
	return true, nil
}

func (vNetworkHandler *OpenStackVNetworkHandler) DeleteInterface(subnetID string, routerID string) (bool, error) {
	deleteOpts := routers.InterfaceOpts{
		SubnetID: subnetID,
	}

	// Delete Interface
	_, err := routers.RemoveInterface(vNetworkHandler.Client, routerID, deleteOpts).Extract()
	if err != nil {
		return false, err
	}
	return true, nil
}

func (vNetworkHandler *OpenStackVNetworkHandler) getVNetworkById(vNetworkID string) (irs.VNetworkInfo, error) {
	network, err := subnets.Get(vNetworkHandler.Client, vNetworkID).Extract()
	if err != nil {
		return irs.VNetworkInfo{}, err
	}

	vNetworkInfo := setterVNet(*network)
	return *vNetworkInfo, nil
}

func (vNetworkHandler *OpenStackVNetworkHandler) getVNetworkIdByName(vNetworkName string) (string, error) {
	var vNetworkId string

	// Name 기준으로 조회
	listOpts := subnets.ListOpts{
		Name: vNetworkName,
	}
	pager, err := subnets.List(vNetworkHandler.Client, listOpts).AllPages()
	if err != nil {
		return "", err
	}
	network, err := subnets.ExtractSubnets(pager)
	if err != nil {
		return "", err
	}

	// 1개 이상의 서브넷이 중복 조회될 경우 에러 처리
	if len(network) == 0 {
		err := errors.New(fmt.Sprintf("failed to search vm with name %s", vNetworkName))
		return "", err
	} else if len(network) > 1 {
		err := errors.New(fmt.Sprintf("failed to search subnet, duplicate nameId exists, %s", vNetworkName))
		return "", err
	} else {
		vNetworkId = network[0].ID
	}

	return vNetworkId, nil
}
