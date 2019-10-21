package resources

import (
	"errors"
	"fmt"
	"github.com/Azure/go-autorest/autorest/to"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/networking/v2/ports"
	"github.com/rackspace/gophercloud/pagination"
)

type OpenStackVNicworkHandler struct {
	Client *gophercloud.ServiceClient
}

func setterVNic(port ports.Port) *irs.VNicInfo {
	portInfo := &irs.VNicInfo{
		Id:               port.ID,
		Name:             port.Name,
		MacAddress:       port.MACAddress,
		OwnedVMID:        port.DeviceID,
		SecurityGroupIds: port.SecurityGroups,
		Status:           port.Status,
	}

	return portInfo
}

func (vNicHandler *OpenStackVNicworkHandler) CreateVNic(vNicReqInfo irs.VNicReqInfo) (irs.VNicInfo, error) {

	// 기본 가상 네트워크 아이디 정보 가져오기
	networkId, err := GetCBVNetId(vNicHandler.Client)
	if networkId == "" {
		return irs.VNicInfo{}, errors.New(fmt.Sprintf("failed to get virtual network by name, name: %s", CBVirutalNetworkName))
	}

	createOpts := ports.CreateOpts{
		NetworkID:    networkId,
		AdminStateUp: to.BoolPtr(true),
		FixedIPs: []ports.IP{
			{SubnetID: vNicReqInfo.VNetId},
		},
		SecurityGroups: vNicReqInfo.SecurityGroupIds,
		Name:           vNicReqInfo.Name,
	}

	port, err := ports.Create(vNicHandler.Client, createOpts).Extract()
	if err != nil {
		return irs.VNicInfo{}, err
	}

	portInfo, err := vNicHandler.GetVNic(port.ID)
	if err != nil {
		return irs.VNicInfo{}, nil
	}

	return portInfo, nil
}

func (vNicHandler *OpenStackVNicworkHandler) ListVNic() ([]*irs.VNicInfo, error) {
	var portList []*irs.VNicInfo

	// 기본 가상 네트워크 아이디 정보 가져오기
	networkId, err := GetCBVNetId(vNicHandler.Client)
	if networkId == "" {
		return nil, errors.New(fmt.Sprintf("failed to get virtual network by name, name: %s", CBVirutalNetworkName))
	}

	listOpts := ports.ListOpts{
		NetworkID: networkId, //Network ID
	}

	pager := ports.List(vNicHandler.Client, listOpts)
	err = pager.EachPage(func(page pagination.Page) (bool, error) {
		// Get Port
		list, err := ports.ExtractPorts(page)
		if err != nil {
			return false, err
		}
		// Add to Port
		for _, p := range list {
			PortInfo := setterVNic(p)
			portList = append(portList, PortInfo)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return portList, nil
}

func (vNicHandler *OpenStackVNicworkHandler) GetVNic(vNicID string) (irs.VNicInfo, error) {
	port, err := ports.Get(vNicHandler.Client, vNicID).Extract()
	if err != nil {
		return irs.VNicInfo{}, err
	}

	portInfo := setterVNic(*port)

	return *portInfo, nil
}

func (vNicHandler *OpenStackVNicworkHandler) DeleteVNic(vNicID string) (bool, error) {
	err := ports.Delete(vNicHandler.Client, vNicID).ExtractErr()
	if err != nil {
		return false, err
	}
	return true, nil
}
