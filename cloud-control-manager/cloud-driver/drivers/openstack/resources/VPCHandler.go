package resources

import (
	"errors"
	"fmt"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/external"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/rackspace/gophercloud/openstack/networking/v2/networks"
	"github.com/rackspace/gophercloud/openstack/networking/v2/subnets"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	VPC = "VPC"
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
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.Client.IdentityEndpoint, call.VPCSUBNET, vpcReqInfo.IId.NameId, "CreateVPC()")

	// Check VPC Exists
	listOpts := networks.ListOpts{Name: vpcReqInfo.IId.NameId}
	page, err := networks.List(vpcHandler.Client, listOpts).AllPages()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{}, err
	}
	vpcList, err := networks.ExtractNetworks(page)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{}, err
	}
	if len(vpcList) != 0 {
		createErr := errors.New(fmt.Sprintf("VPC with name %s already exist", vpcReqInfo.IId.NameId))
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}

	// Create VPC
	createOpts := networks.CreateOpts{
		Name: vpcReqInfo.IId.NameId,
	}

	start := call.Start()
	vpc, err := networks.Create(vpcHandler.Client, createOpts).Extract()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	// Create Subnet
	for _, subnet := range vpcReqInfo.SubnetInfoList {
		_, err := vpcHandler.CreateSubnet(vpc.ID, subnet)
		if err != nil {
			// TODO: VPC 삭제 처리 (rollback)
			/*if ok, err := vpcHandler.DeleteVPC(irs.IID{SystemId: vpc.ID}); !ok {
				cblogger.Error("Failed to delete vpc with Id %s, err=%s", vpc.ID, err)
				return irs.VPCInfo{}, err
			}*/
			LoggingError(hiscallInfo, err)
			return irs.VPCInfo{}, err
		}
	}

	vpcInfo, err := vpcHandler.GetVPC(irs.IID{SystemId: vpc.ID})
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to get vpc with Id %s, err=%s", vpc.ID, err.Error()))
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}

	// TODO: nested flow 개선
	// Create Router
	routerId, err := vpcHandler.CreateRouter(vpcReqInfo.IId.NameId)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to get create router, err=%s", err.Error()))
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}

	// TODO: nested flow 개선
	// Create Interface
	for _, subnet := range vpcInfo.SubnetInfoList {
		if ok, err := vpcHandler.AddInterface(subnet.IId.SystemId, *routerId); !ok {
			createErr := errors.New(fmt.Sprintf("Failed to get create router interface, err=%s", err.Error()))
			LoggingError(hiscallInfo, createErr)
			return irs.VPCInfo{}, createErr
		}
	}

	return vpcInfo, nil
}
func (vpcHandler *OpenStackVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.Client.IdentityEndpoint, call.VPCSUBNET, VPC, "ListVPC()")

	start := call.Start()
	page, err := networks.List(vpcHandler.Client, nil).AllPages()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get vpc list, err=%s", err.Error()))
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	LoggingInfo(hiscallInfo, start)

	vpcList, err := external.ExtractList(page)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get vpc list, err=%s", err.Error()))
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	// Get VPC List
	vpcInfoList := make([]*irs.VPCInfo, len(vpcList))
	for i, vpc := range vpcList {
		vpcInfo := vpcHandler.setterVPC(vpc)
		vpcInfoList[i] = vpcInfo
	}
	return vpcInfoList, nil
}

func (vpcHandler *OpenStackVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.Client.IdentityEndpoint, call.VPCSUBNET, vpcIID.NameId, "GetVPC()")

	start := call.Start()
	vpcResult := networks.Get(vpcHandler.Client, vpcIID.SystemId)
	externalVpc, err := external.ExtractGet(vpcResult)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get vpc with Id %s, err=%s", vpcIID.SystemId, err.Error()))
		LoggingError(hiscallInfo, getErr)
		return irs.VPCInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)

	vpcInfo := vpcHandler.setterVPC(*externalVpc)
	return *vpcInfo, nil
}

func (vpcHandler *OpenStackVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.Client.IdentityEndpoint, call.VPCSUBNET, vpcIID.NameId, "DeleteVPC()")

	vpcInfo, err := vpcHandler.GetVPC(vpcIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get vpc with Id %s, err=%s", vpcIID.SystemId, err.Error()))
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}

	// TODO: nested flow 개선
	// Delete Interface
	routerId, err := vpcHandler.GetRouter(vpcIID.NameId)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to get router, err=%s", err.Error()))
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}
	for _, subnet := range vpcInfo.SubnetInfoList {
		if ok, err := vpcHandler.DeleteInterface(subnet.IId.SystemId, *routerId); !ok {
			getErr := errors.New(fmt.Sprintf("Failed to delete router interface, err=%s", err.Error()))
			LoggingError(hiscallInfo, getErr)
			return false, getErr
		}
	}

	// TODO: nested flow 개선
	// Delete Router
	err = routers.Delete(vpcHandler.Client, *routerId).ExtractErr()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to delete router, err=%s", err.Error()))
		LoggingError(hiscallInfo, getErr)
		return false, getErr
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
	start := call.Start()
	err = networks.Delete(vpcHandler.Client, vpcInfo.IId.SystemId).ExtractErr()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to delete vpc, err=%s", err.Error()))
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}
	LoggingInfo(hiscallInfo, start)

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

func (vpcHandler *OpenStackVPCHandler) AddSubnet(vpcIID irs.IID, subnetInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.Client.IdentityEndpoint, call.VPCSUBNET, subnetInfo.IId.NameId, "AddSubnet()")

	subnetCreateOpts := subnets.CreateOpts{
		NetworkID:      vpcIID.SystemId,
		Name:           subnetInfo.IId.NameId,
		CIDR:           subnetInfo.IPv4_CIDR,
		IPVersion:      subnets.IPv4,
		DNSNameservers: []string{DNSNameservers},
	}

	start := call.Start()
	_, err := subnets.Create(vpcHandler.Client, subnetCreateOpts).Extract()
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to create Subnet with name %s, err=%s", subnetCreateOpts.Name, err.Error()))
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}
	LoggingInfo(hiscallInfo, start)

	result, err := vpcHandler.GetVPC(vpcIID)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.VPCInfo{}, err
	}
	return result, nil
}

func (vpcHandler *OpenStackVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.Client.IdentityEndpoint, call.VPCSUBNET, subnetIID.NameId, "RemoveSubnet()")

	start := call.Start()
	err := subnets.Delete(vpcHandler.Client, subnetIID.SystemId).ExtractErr()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}
