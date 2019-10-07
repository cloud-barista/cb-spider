package resources

import (
	"github.com/Azure/go-autorest/autorest/to"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/networking/v2/ports"
	"github.com/rackspace/gophercloud/pagination"
)

/*var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}*/

type OpenStackVNicworkHandler struct {
	Client *gophercloud.ServiceClient
}

// @TODO: KeyPairInfo 리소스 프로퍼티 정의 필요
type FixedIPInfo struct {
	SubnetId  string
	IPAddress string
}
type AddressPairInfo struct {
	IPAddress  string
	MACAddress string
}
type PortInfo struct {
	Id                  string
	NetworkId           string
	Name                string
	AdminStateUp        bool
	Status              string
	MACAddress          string
	FixedIPs            []FixedIPInfo
	TenantID            string
	DeviceOwner         string
	SecurityGroups      []string
	DeviceID            string
	AllowedAddressPairs []AddressPairInfo
}

func (portInfo *PortInfo) setter(port ports.Port) *PortInfo {
	portInfo.Id = port.ID
	portInfo.NetworkId = port.NetworkID
	portInfo.Name = port.Name
	portInfo.AdminStateUp = port.AdminStateUp
	portInfo.MACAddress = port.MACAddress
	portInfo.TenantID = port.TenantID
	portInfo.DeviceOwner = port.DeviceOwner
	portInfo.SecurityGroups = port.SecurityGroups
	portInfo.DeviceID = port.DeviceID

	var fixedIPArr []FixedIPInfo
	var allowedAddressPairArr []AddressPairInfo

	for _, ip := range port.FixedIPs {
		IPInfo := FixedIPInfo{
			SubnetId:  ip.SubnetID,
			IPAddress: ip.IPAddress,
		}
		fixedIPArr = append(fixedIPArr, IPInfo)
	}

	for _, addressPair := range port.AllowedAddressPairs {
		addressPairInfo := AddressPairInfo{
			IPAddress:  addressPair.IPAddress,
			MACAddress: addressPair.MACAddress,
		}
		allowedAddressPairArr = append(allowedAddressPairArr, addressPairInfo)
	}

	portInfo.FixedIPs = fixedIPArr
	portInfo.AllowedAddressPairs = allowedAddressPairArr

	return portInfo
}

func (vNicHandler *OpenStackVNicworkHandler) CreateVNic(vNicReqInfo irs.VNicReqInfo) (irs.VNicInfo, error) {

	// @TODO: Port 생성 요청 파라미터 정의 필요
	type PortReqInfo struct {
		NetworkId    string
		Name         string
		AdminStateUp bool
		SubnetId     string
	}

	reqInfo := PortReqInfo{
		NetworkId:    "ccaec0ad-f187-4c41-b26d-23bde011795f",
		AdminStateUp: true,
		SubnetId:     "171c1c68-4ab1-4185-87f4-941262b9ff5e",
	}

	createOpts := ports.CreateOpts{
		NetworkID:    reqInfo.NetworkId,
		AdminStateUp: to.BoolPtr(true),
		FixedIPs: []ports.IP{
			{SubnetID: reqInfo.SubnetId},
		},
	}
	port, err := ports.Create(vNicHandler.Client, createOpts).Extract()
	if err != nil {
		return irs.VNicInfo{}, err
	}

	spew.Dump(port)
	return irs.VNicInfo{Id: port.ID, Name: port.Name}, nil
}

func (vNicHandler *OpenStackVNicworkHandler) ListVNic() ([]*irs.VNicInfo, error) {
	var portList []PortInfo

	pager := ports.List(vNicHandler.Client, nil)
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		// Get Port
		list, err := ports.ExtractPorts(page)
		if err != nil {
			return false, err
		}
		// Add to Port
		for _, p := range list {
			PortInfo := new(PortInfo).setter(p)
			portList = append(portList, *PortInfo)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	spew.Dump(portList)
	return nil, nil
}

func (vNicHandler *OpenStackVNicworkHandler) GetVNic(vNicID string) (irs.VNicInfo, error) {
	port, err := ports.Get(vNicHandler.Client, vNicID).Extract()
	if err != nil {
		return irs.VNicInfo{}, err
	}

	portInfo := new(PortInfo).setter(*port)

	spew.Dump(portInfo)
	return irs.VNicInfo{}, nil
}

func (vNicHandler *OpenStackVNicworkHandler) DeleteVNic(vNicID string) (bool, error) {
	err := ports.Delete(vNicHandler.Client, vNicID).ExtractErr()
	if err != nil {
		return false, err
	}
	return true, nil
}
