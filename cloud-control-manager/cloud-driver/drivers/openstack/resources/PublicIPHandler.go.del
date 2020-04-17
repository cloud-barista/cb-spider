package resources

import (
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/floatingip"
	"github.com/rackspace/gophercloud/pagination"
)

type OpenStackPublicIPHandler struct {
	Client *gophercloud.ServiceClient
}

func setterPublicIP(publicIP floatingip.FloatingIP) *irs.PublicIPInfo {
	publicIPInfo := &irs.PublicIPInfo{
		Name:      publicIP.ID,
		PublicIP:  publicIP.IP,
		OwnedVMID: publicIP.InstanceID,
	}
	return publicIPInfo
}

func (publicIPHandler *OpenStackPublicIPHandler) CreatePublicIP(publicIPReqInfo irs.PublicIPReqInfo) (irs.PublicIPInfo, error) {

	createOpts := floatingip.CreateOpts{
		Pool: CBPublicIPPool,
	}
	publicIP, err := floatingip.Create(publicIPHandler.Client, createOpts).Extract()
	if err != nil {
		return irs.PublicIPInfo{}, err
	}

	// 생성된 PublicIP 정보 리턴
	publicIPInfo, err := publicIPHandler.GetPublicIP(publicIP.ID)
	if err != nil {
		return irs.PublicIPInfo{}, err
	}
	return publicIPInfo, nil
}

func (publicIPHandler *OpenStackPublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	var publicIPList []*irs.PublicIPInfo

	pager := floatingip.List(publicIPHandler.Client)
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		// Get PublicIP
		list, err := floatingip.ExtractFloatingIPs(page)
		if err != nil {
			return false, err
		}
		// Add to List
		for _, p := range list {
			publicIPInfo := setterPublicIP(p)
			publicIPList = append(publicIPList, publicIPInfo)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return publicIPList, nil
}

func (publicIPHandler *OpenStackPublicIPHandler) GetPublicIP(publicIPID string) (irs.PublicIPInfo, error) {
	floatingIP, err := floatingip.Get(publicIPHandler.Client, publicIPID).Extract()
	if err != nil {
		return irs.PublicIPInfo{}, err
	}

	publicIPInfo := setterPublicIP(*floatingIP)
	return *publicIPInfo, nil
}

func (publicIPHandler *OpenStackPublicIPHandler) DeletePublicIP(publicIPID string) (bool, error) {
	err := floatingip.Delete(publicIPHandler.Client, publicIPID).ExtractErr()
	if err != nil {
		return false, err
	}
	return true, nil
}
