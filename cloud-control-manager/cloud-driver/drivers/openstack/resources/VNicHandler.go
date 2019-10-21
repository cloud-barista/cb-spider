package resources

import (
	"github.com/Azure/go-autorest/autorest/to"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"
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
		MacAdress:        port.MACAddress,
		OwnedVMID:        port.DeviceID,
		SecurityGroupIds: port.SecurityGroups,
		Status:           port.Status,
	}

	return portInfo
}

func (vNicHandler *OpenStackVNicworkHandler) CreateVNic(vNicReqInfo irs.VNicReqInfo) (irs.VNicInfo, error) {

	createOpts := ports.CreateOpts{
		NetworkID:    vNicReqInfo.VNetId,
		AdminStateUp: to.BoolPtr(true),
		FixedIPs: []ports.IP{
			{SubnetID: vNicReqInfo.SubnetId},
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

	listOpts := ports.ListOpts{
		NetworkID: "0013efbf-9e64-476b-a09c-9e4f5c0c8bed", //Network ID
	}

	pager := ports.List(vNicHandler.Client, listOpts)
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
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
