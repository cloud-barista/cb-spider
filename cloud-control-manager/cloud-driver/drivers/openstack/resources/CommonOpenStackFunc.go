package resources

import (
	"errors"
	"fmt"
	_ "fmt"
	_ "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/rackspace/gophercloud/openstack/compute/v2/flavors"
	"github.com/rackspace/gophercloud/openstack/networking/v2/networks"
	"github.com/rackspace/gophercloud/openstack/networking/v2/ports"
	"github.com/rackspace/gophercloud/openstack/networking/v2/subnets"
	"github.com/rackspace/gophercloud/pagination"
	_ "strconv"
	"strings"
	_ "strings"
)

const (
	CBPublicIPPool       = "ext"
	CBGateWayId          = "8c1af031-aad6-4762-ac83-52e09dd82571"
	CBVirutalNetworkName = "CB-VNet"
	DNSNameservers       = "8.8.8.8"
)

// 기본 가상 네트워크(CB-VNet) Id 정보 조회
func GetCBVNetId(client *gophercloud.ServiceClient) (string, error) {
	listOpt := networks.ListOpts{
		Name: CBVirutalNetworkName,
	}

	var vNetworkId string

	pager := networks.List(client, listOpt)
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

func GetFlavor(client *gophercloud.ServiceClient, flavorName string) (*string, error) {
	flavorId, err := flavors.IDFromName(client, flavorName)
	if err != nil {
		return nil, err
	}
	return &flavorId, nil
}

func GetSecurityByName(networkClient *gophercloud.ServiceClient, securityName string) (*secgroups.SecurityGroup, error) {
	pages, err := secgroups.List(networkClient).AllPages()
	if err != nil {
		return nil, err
	}
	secGroupList, err := secgroups.ExtractSecurityGroups(pages)
	if err != nil {
		return nil, err
	}

	for _, s := range secGroupList {
		if strings.EqualFold(s.Name, securityName) {
			return &s, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("could not found SecurityGroups with name %s ", securityName))
}

func GetNetworkByName(networkClient *gophercloud.ServiceClient, networkName string) (*networks.Network, error) {
	pages, err := networks.List(networkClient, networks.ListOpts{Name: networkName}).AllPages()
	if err != nil {
		return nil, err
	}
	netList, err := networks.ExtractNetworks(pages)
	if err != nil {
		return nil, err
	}

	for _, s := range netList {
		if strings.EqualFold(s.Name, networkName) {
			return &s, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("could not found SecurityGroups with name %s ", networkName))
}

func GetSubnetByID(networkClient *gophercloud.ServiceClient, subnetId string) (*subnets.Subnet, error) {
	subnet, err := subnets.Get(networkClient, subnetId).Extract()
	if err != nil {
		return nil, err
	}
	return subnet, nil
}

func GetPortByDeviceID(networkClient *gophercloud.ServiceClient, deviceID string) (*ports.Port, error) {
	pages, err := ports.List(networkClient, ports.ListOpts{}).AllPages()
	if err != nil {
		return nil, err
	}
	portList, err := ports.ExtractPorts(pages)
	if err != nil {
		return nil, err
	}

	for _, s := range portList {
		if strings.EqualFold(s.DeviceID, deviceID) {
			return &s, nil
		}
	}
	return nil, errors.New(fmt.Sprintf("could not found SecurityGroups with name %s ", deviceID))
}
