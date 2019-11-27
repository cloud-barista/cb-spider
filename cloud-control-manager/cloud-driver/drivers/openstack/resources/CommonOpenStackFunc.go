package resources

import (
	"fmt"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/flavors"
	"github.com/rackspace/gophercloud/openstack/networking/v2/networks"
	"github.com/rackspace/gophercloud/pagination"
	"strconv"
	"strings"
)

const (
	CBPublicIPPool       = "ext"
	CBGateWayId          = "8c1af031-aad6-4762-ac83-52e09dd82571"
	CBVirutalNetworkName = "CB-VNet"
	CBVnetDefaultCidr    = "130.0.0.0/16"
	//CBVMUser             = "cb-user"
	DNSNameservers = "8.8.8.8"
)

// 서브넷 CIDR 생성 (CIDR C class 기준 생성)
func CreateSubnetCIDR(subnetList []*irs.VNetworkInfo) (*string, error) {

	// CIDR C class 최대값 찾기
	maxClassNum := 0
	for _, subnet := range subnetList {
		addressArr := strings.Split(subnet.AddressPrefix, ".")
		if curClassNum, err := strconv.Atoi(addressArr[2]); err != nil {
			return nil, err
		} else {
			if curClassNum > maxClassNum {
				maxClassNum = curClassNum
			}
		}
	}

	if len(subnetList) == 0 {
		maxClassNum = 0
	} else {
		maxClassNum = maxClassNum + 1
	}

	// 서브넷 CIDR 할당
	vNetIP := strings.Split(CBVnetDefaultCidr, "/")
	vNetIPClass := strings.Split(vNetIP[0], ".")
	subnetCIDR := fmt.Sprintf("%s.%s.%d.0/24", vNetIPClass[0], vNetIPClass[1], maxClassNum)
	return &subnetCIDR, nil
}

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

// 외부 네트워크(Public Network) 정보 조회
/*func GetExternalNetwork(client *gophercloud.ServiceClient) (string, error) {

	listOpts := external.ListOptsExt{
		External: to.BoolPtr(true),
	}
}*/
/*func GetPublicGatewayId() (*string, error) {
	//CBPublicIPPool       = "ext"
	//CBGateWayId          = "8c1af031-aad6-4762-ac83-52e09dd82571"

	listOpts := external.ListOptsExt{
		External: to.BoolPtr(true),
	}

	query, err := listOpts.ToNetworkListQuery()
	if err != nil {
		panic(err)
		return nil, err
	}

	fmt.Println(query)

	return nil, nil
}*/
