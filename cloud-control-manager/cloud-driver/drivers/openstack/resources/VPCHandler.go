package resources

import (
	"errors"
	"fmt"
	"github.com/Azure/go-autorest/autorest/to"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	//"github.com/davecgh/go-spew/spew"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/external"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/rackspace/gophercloud/openstack/networking/v2/networks"
	"github.com/rackspace/gophercloud/openstack/networking/v2/subnets"
)

type OpenStackVPCHandler struct {
	Client *gophercloud.ServiceClient
}

func (vpcHandler *OpenStackVPCHandler) setterVPC(nvpc external.NetworkExternal) *irs.VPCInfo {
	// VPC 정보 맵핑
	vpcInfo := irs.VPCInfo{
		IId: irs.IID{
			NameId:   nvpc.Name,
			SystemId: nvpc.ID,
		},
	}
	var External string
	if nvpc.External == true {
		External = "Yes"
	} else if nvpc.External == false {
		External = "No"
	}
	keyValueList := []irs.KeyValue{
		{Key: "External Network", Value: External},
	}
	vpcInfo.KeyValueList = keyValueList
	// 서브넷 정보 조회
	subnetInfoList := make([]irs.SubnetInfo, len(nvpc.Subnets))

	for i, subnetId := range nvpc.Subnets {
		subnetInfo, err := vpcHandler.GetSubnet(irs.IID{SystemId: subnetId})
		if err != nil {
			cblogger.Error("Failed to get subnet with Id %s, err=%s", subnetId, err)
			continue
		}
		subnetInfoList[i] = subnetInfo
	}
	vpcInfo.SubnetInfoList = subnetInfoList

	return &vpcInfo
}

func (vpcHandler *OpenStackVPCHandler) setterSubnet(subnet subnets.Subnet) *irs.SubnetInfo {
	subnetInfo := irs.SubnetInfo{
		IId: irs.IID{
			NameId:   subnet.Name,
			SystemId: subnet.ID,
		},
		IPv4_CIDR: subnet.CIDR,
	}
	return &subnetInfo
}

func (vpcHandler *OpenStackVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	// Check VPC Exists
	listOpts := networks.ListOpts{Name: vpcReqInfo.IId.NameId}
	page, err := networks.List(vpcHandler.Client, listOpts).AllPages()
	if err != nil {
		return irs.VPCInfo{}, err
	}
	vpcList, err := networks.ExtractNetworks(page)
	if err != nil {
		return irs.VPCInfo{}, err
	}
	if len(vpcList) != 0 {
		createErr := errors.New(fmt.Sprintf("VPC with name %s already exist", vpcReqInfo.IId.NameId))
		return irs.VPCInfo{}, createErr
	}

	// Create VPC
	createOpts := networks.CreateOpts{
		Name: vpcReqInfo.IId.NameId,
	}
	vpc, err := networks.Create(vpcHandler.Client, createOpts).Extract()
	if err != nil {
		return irs.VPCInfo{}, err
	}

	// Create Subnet
	for _, subnet := range vpcReqInfo.SubnetInfoList {
		_, err := vpcHandler.CreateSubnet(vpc.ID, subnet)
		if err != nil {
			// TODO: VPC 삭제 처리 (rollback)
			/*if ok, err := vpcHandler.DeleteVPC(irs.IID{SystemId: vpc.ID}); !ok {
				cblogger.Error("Failed to delete vpc with Id %s, err=%s", vpc.ID, err)
				return irs.VPCInfo{}, err
			}*/
			return irs.VPCInfo{}, err
		}
	}

	vpcInfo, err := vpcHandler.GetVPC(irs.IID{SystemId: vpc.ID})
	if err != nil {
		cblogger.Error("Failed to get vpc with Id %s, err=%s", vpc.ID, err)
		return irs.VPCInfo{}, err
	}

	// TODO: nested flow 개선
	// Create Router
	routerId, err := vpcHandler.CreateRouter(vpcReqInfo.IId.NameId)
	if err != nil {
		cblogger.Error("Failed to get create router, err=%s", err)
		return irs.VPCInfo{}, err
	}

	// TODO: nested flow 개선
	// Create Interface
	for _, subnet := range vpcInfo.SubnetInfoList {
		if ok, err := vpcHandler.AddInterface(subnet.IId.SystemId, *routerId); !ok {
			cblogger.Error("Failed to get create router, err=%s", err)
			return irs.VPCInfo{}, err
		}
	}

	return vpcInfo, nil
}
func (vpcHandler *OpenStackVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {

	page, err := networks.List(vpcHandler.Client, nil).AllPages()
	if err != nil {
		cblogger.Error("Failed to get vpc list, err=%s", err)
		return nil, err
	}

	nvpcList, err := external.ExtractList(page)
	if err != nil {
		cblogger.Error("Failed to get vpc list, err=%s", err)
		return nil, err
	}

	//	keyValue := make([]*irs.KeyValue, len(vpcInfo.KeyValueList))

	// Get VPC List
	vpcInfoList := make([]*irs.VPCInfo, len(nvpcList))
	for i, vpc := range nvpcList {
		vpcInfo := vpcHandler.setterVPC(vpc)
		vpcInfoList[i] = vpcInfo
	}
	return vpcInfoList, nil
}

func (vpcHandler *OpenStackVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	vpc := networks.Get(vpcHandler.Client, vpcIID.SystemId)
	//var vpc networks.GetResult
	//nvpc ,err :=
	nvpc, err := external.ExtractGet(vpc)
	if err != nil {
		cblogger.Error("Failed to get vpc with Id %s, err=%s", vpcIID.SystemId, err)
		return irs.VPCInfo{}, err
	}
	vpcInfo := vpcHandler.setterVPC(*nvpc)

	return *vpcInfo, nil
}

func (vpcHandler *OpenStackVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	vpcInfo, err := vpcHandler.GetVPC(vpcIID)
	if err != nil {
		cblogger.Error("Failed to get vpc with Id %s, err=%s", vpcIID.SystemId, err)
		return false, err
	}

	// TODO: nested flow 개선
	// Delete Interface
	routerId, err := vpcHandler.GetRouter(vpcIID.NameId)
	if err != nil {
		cblogger.Error("Failed to get router, err=%s", err)
		return false, err
	}
	for _, subnet := range vpcInfo.SubnetInfoList {
		if ok, err := vpcHandler.DeleteInterface(subnet.IId.SystemId, *routerId); !ok {
			cblogger.Error("Failed to delete router interface, err=%s", err)
			return false, err
		}
	}

	// TODO: nested flow 개선
	// Delete Router
	err = routers.Delete(vpcHandler.Client, *routerId).ExtractErr()
	if err != nil {
		cblogger.Error("Failed to delete router, err=%s", err)
		return false, err
	}

	// TODO: nested flow 개선
	// Delete Subnet
	/*for _, subnet := range vpcInfo.SubnetInfoList {
		if ok, err:= vpcHandler.DeleteSubnet(irs.IID{SystemId: subnet.IId.SystemId}); !ok {
			cblogger.Error("Failed to delete subnet, err=%s", err)
			return false, err
		}
	}*/

	// TODO: nested flow 개선
	//Delete VPC
	err = networks.Delete(vpcHandler.Client, vpcInfo.IId.SystemId).ExtractErr()
	if err != nil {
		cblogger.Error("Failed to delete vpc, err=%s", err)
		return false, err
	}

	return true, nil
}

func (vpcHandler *OpenStackVPCHandler) CreateSubnet(vpcId string, reqSubnetInfo irs.SubnetInfo) (irs.SubnetInfo, error) {
	subnetCreateOpts := subnets.CreateOpts{
		NetworkID:      vpcId,
		Name:           reqSubnetInfo.IId.NameId,
		CIDR:           reqSubnetInfo.IPv4_CIDR,
		IPVersion:      subnets.IPv4,
		DNSNameservers: []string{DNSNameservers},
	}
	subnet, err := subnets.Create(vpcHandler.Client, subnetCreateOpts).Extract()
	if err != nil {
		cblogger.Error("Failed to create Subnet with name %s, err=%s", reqSubnetInfo.IId.NameId, err)
		return irs.SubnetInfo{}, err
	}
	subnetInfo := vpcHandler.setterSubnet(*subnet)
	return *subnetInfo, nil
}

func (vpcHandler *OpenStackVPCHandler) GetSubnet(subnetIId irs.IID) (irs.SubnetInfo, error) {
	subnet, err := subnets.Get(vpcHandler.Client, subnetIId.SystemId).Extract()
	if err != nil {
		cblogger.Error("Failed to get Subnet with Id %s, err=%s", subnetIId.SystemId, err)
		return irs.SubnetInfo{}, nil
	}
	subnetInfo := vpcHandler.setterSubnet(*subnet)
	return *subnetInfo, nil
}

func (vpcHandler *OpenStackVPCHandler) DeleteSubnet(subnetIId irs.IID) (bool, error) {
	err := subnets.Delete(vpcHandler.Client, subnetIId.SystemId).ExtractErr()
	if err != nil {
		cblogger.Error("Failed to delete Subnet with Id %s, err=%s", subnetIId.SystemId, err)
		return false, err
	}
	return true, nil
}

func (vpcHandler *OpenStackVPCHandler) CreateRouter(vpcName string) (*string, error) {
	externVPCId, _ := GetPublicVPCInfo(vpcHandler.Client, "ID")
	routerName := vpcName + "-Router"
	createOpts := routers.CreateOpts{
		Name:         routerName,
		AdminStateUp: to.BoolPtr(true),
		GatewayInfo: &routers.GatewayInfo{
			NetworkID: externVPCId,
		},
	}

	// Create Router
	router, err := routers.Create(vpcHandler.Client, createOpts).Extract()
	if err != nil {
		return nil, err
	}
	return &router.ID, nil
}

func (vpcHandler *OpenStackVPCHandler) GetRouter(vpcName string) (*string, error) {
	// Get Router Info
	routerName := vpcName + "-Router"
	listOpts := routers.ListOpts{Name: routerName}
	page, err := routers.List(vpcHandler.Client, listOpts).AllPages()
	if err != nil {
		cblogger.Error("Failed to list router, err=%s", err)
		return nil, err
	}
	routerList, err := routers.ExtractRouters(page)
	if err != nil {
		cblogger.Error("Failed to extract router, err=%s", err)
		return nil, err
	}
	if len(routerList) != 1 {
		cblogger.Error("Failed to get router with name %s, err=%s", routerName)
		return nil, err
	}

	routerId := routerList[0].ID
	return &routerId, nil
}

func (vpcHandler *OpenStackVPCHandler) DeleteRouter(vpcName string) (bool, error) {
	// Get Router
	routerId, err := vpcHandler.GetRouter(vpcName)
	if err != nil {
		cblogger.Error("Failed to delete router with Id %s, err=%s", routerId)
		return false, err
	}
	// Delete Router
	err = routers.Delete(vpcHandler.Client, *routerId).ExtractErr()
	if err != nil {
		cblogger.Error("Failed to delete router with Id %s, err=%s", routerId)
		return false, err
	}
	return true, nil
}

func (vpcHandler *OpenStackVPCHandler) AddInterface(subnetId string, routerId string) (bool, error) {
	createOpts := routers.InterfaceOpts{
		SubnetID: subnetId,
	}

	// Add Interface
	_, err := routers.AddInterface(vpcHandler.Client, routerId, createOpts).Extract()
	if err != nil {
		return false, err
	}
	return true, nil
}

func (vpcHandler *OpenStackVPCHandler) DeleteInterface(subnetId string, routerId string) (bool, error) {
	deleteOpts := routers.InterfaceOpts{
		SubnetID: subnetId,
	}

	// Delete Interface
	_, err := routers.RemoveInterface(vpcHandler.Client, routerId, deleteOpts).Extract()
	if err != nil {
		return false, err
	}
	return true, nil
}
