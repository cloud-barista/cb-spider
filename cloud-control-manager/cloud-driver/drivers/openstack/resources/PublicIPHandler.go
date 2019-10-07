package resources

import (
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/floatingip"
	"github.com/rackspace/gophercloud/pagination"
)

type OpenStackPublicIPHandler struct {
	Client *gophercloud.ServiceClient
}

// @TODO: PublicIP 리소스 프로퍼티 정의 필요
type PublicIPInfo struct {
	ID         string
	FixedIP    string
	InstanceID string
	IP         string
	Pool       string
}

func (publicIPInfo *PublicIPInfo) setter(floatingIp floatingip.FloatingIP) *PublicIPInfo {
	publicIPInfo.ID = floatingIp.ID
	publicIPInfo.FixedIP = floatingIp.FixedIP
	publicIPInfo.InstanceID = floatingIp.InstanceID
	publicIPInfo.IP = floatingIp.IP
	publicIPInfo.Pool = floatingIp.Pool

	return publicIPInfo
}

func (publicIPHandler *OpenStackPublicIPHandler) CreatePublicIP(publicIPReqInfo irs.PublicIPReqInfo) (irs.PublicIPInfo, error) {

	// @TODO: PublicIP 생성 요청 파라미터 정의 필요
	type PublicIPReqInfo struct {
		Pool string
	}
	reqInfo := PublicIPReqInfo{
		Pool: "public1", // Floating IP가 할당되는 IP Pool 정보
	}

	createOpts := floatingip.CreateOpts{
		Pool: reqInfo.Pool,
	}
	publicIPInfo, err := floatingip.Create(publicIPHandler.Client, createOpts).Extract()
	if err != nil {
		return irs.PublicIPInfo{}, err
	}

	spew.Dump(publicIPInfo)
	return irs.PublicIPInfo{Id: publicIPInfo.ID}, nil
}

func (publicIPHandler *OpenStackPublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	var publicIPList []*PublicIPInfo

	pager := floatingip.List(publicIPHandler.Client)
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		// Get PublicIP
		list, err := floatingip.ExtractFloatingIPs(page)
		if err != nil {
			return false, err
		}
		// Add to List
		for _, p := range list {
			publicIPInfo := new(PublicIPInfo).setter(p)
			publicIPList = append(publicIPList, publicIPInfo)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	spew.Dump(publicIPList)
	return nil, nil
}

func (publicIPHandler *OpenStackPublicIPHandler) GetPublicIP(publicIPID string) (irs.PublicIPInfo, error) {
	floatingIP, err := floatingip.Get(publicIPHandler.Client, publicIPID).Extract()
	if err != nil {
		return irs.PublicIPInfo{}, err
	}

	publicIPInfo := new(PublicIPInfo).setter(*floatingIP)

	spew.Dump(publicIPInfo)
	return irs.PublicIPInfo{Id: publicIPInfo.IP}, nil
}

func (publicIPHandler *OpenStackPublicIPHandler) DeletePublicIP(publicIPID string) (bool, error) {
	err := floatingip.Delete(publicIPHandler.Client, publicIPID).ExtractErr()
	if err != nil {
		return false, err
	}
	return true, nil
}

func (publicIPHandler *OpenStackPublicIPHandler) AssociatePublicIP(serverID string, publicIPID string) (bool, error) {
	associateOpts := floatingip.AssociateOpts{
		ServerID:   serverID,
		FloatingIP: publicIPID,
	}
	err := floatingip.AssociateInstance(publicIPHandler.Client, associateOpts).ExtractErr()
	if err != nil {
		return false, err
	}
	return true, nil
}
