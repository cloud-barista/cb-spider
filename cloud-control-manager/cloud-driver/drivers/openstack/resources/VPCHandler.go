package resources

import (
	"errors"
	"fmt"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/external"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/layer3/routers"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/networks"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/subnets"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	VPC = "VPC"
)

type OpenStackVPCHandler struct {
	Client *gophercloud.ServiceClient
}

type NetworkWithExt struct {
	networks.Network
	external.NetworkExternalExt
}

func (vpcHandler *OpenStackVPCHandler) setterVPC(nvpc NetworkWithExt) *irs.VPCInfo {

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
			cblogger.Error("Failed to Get Subnet with Id %s, err=%s", subnetId, err)
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
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC. The VPC name %s already exists", vpcReqInfo.IId.NameId))
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
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC with name %s err=%s and Finished to rollback deleting", vpcReqInfo.IId.NameId, err.Error()))
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}
	LoggingInfo(hiscallInfo, start)

	// Create Subnet
	for _, subnet := range vpcReqInfo.SubnetInfoList {
		_, err := vpcHandler.CreateSubnet(vpc.ID, subnet)
		if err != nil {
			// CreateSubnet Error => DeleteNetwork
			networkDeleteErr := networks.Delete(vpcHandler.Client, vpc.ID).ExtractErr()
			createErr := errors.New(fmt.Sprintf("Failed to Create VPC with name %s. While Create Failed Subnet with name %s, err=%s and Finished to rollback deleting", vpcReqInfo.IId.NameId, subnet.IId.NameId, err.Error()))
			if networkDeleteErr != nil {
				// CreateSubnet Error => DeleteNetwork Error => return error + error
				createErr = errors.New(fmt.Sprintf("Failed to Create VPC with name %s. While Create Failed subnet with name %s, err=%s and Failed to rollback delete Network with name %s. err=%s", vpcReqInfo.IId.NameId, subnet.IId.NameId, err.Error(), vpcReqInfo.IId.NameId, networkDeleteErr.Error()))
			}
			LoggingError(hiscallInfo, createErr)
			return irs.VPCInfo{}, createErr
		}
	}

	vpcInfo, err := vpcHandler.GetVPC(irs.IID{SystemId: vpc.ID})
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC with name %s, err=%s and Finished to rollback deleting", vpcReqInfo.IId.NameId, err.Error()))
		networkDeleteErr := networks.Delete(vpcHandler.Client, vpc.ID).ExtractErr()
		if networkDeleteErr != nil {
			// CreateSubnet Error => DeleteNetwork Error => return error + error
			createErr = errors.New(fmt.Sprintf("Failed to Create VPC with name %s, err=%s and Failed to rollback delete Network with name %s  err=%s", vpcReqInfo.IId.NameId, err.Error(), vpcReqInfo.IId.NameId, networkDeleteErr.Error()))
		}
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}

	// TODO: nested flow 개선
	// Create Router
	routerId, err := vpcHandler.CreateRouter(vpcReqInfo.IId.NameId)
	if err != nil {
		// CreateRouter Error => DeleteNetwork
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC with name %s, and Finished to rollback deleting", vpcReqInfo.IId.NameId))
		networkDeleteErr := networks.Delete(vpcHandler.Client, vpc.ID).ExtractErr()
		if networkDeleteErr != nil {
			// CreateRouter Error => DeleteNetwork Error => return error + error
			err = errors.New(fmt.Sprintf("Failed to Create VPC with name %s err=%s, and Failed to rollback delete Network with name %s err=%s", vpcReqInfo.IId.NameId, err.Error(), vpcReqInfo.IId.NameId, networkDeleteErr.Error()))
		}
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}

	// TODO: nested flow 개선
	// Create Interface
	for subnetIndex, subnet := range vpcInfo.SubnetInfoList {
		if ok, err := vpcHandler.AddInterface(subnet.IId.SystemId, *routerId); !ok {
			// AddInterface Error => Delete PreInterfaceDelete
			if subnetIndex != 0 {
				preSubnetList := vpcInfo.SubnetInfoList[:subnetIndex]
				for _, deleteSubnet := range preSubnetList {
					if deleteCheck, err := vpcHandler.DeleteInterface(deleteSubnet.IId.SystemId, *routerId); !deleteCheck {
						if err == nil {
							createErr := errors.New(fmt.Sprintf("Failed to Create VPC with name %s and Failed to rollback delete Interface", vpcReqInfo.IId.NameId))
							LoggingError(hiscallInfo, createErr)
							return irs.VPCInfo{}, createErr
						}
						createErr := errors.New(fmt.Sprintf("Failed to Create VPC with name %s and Failed to rollback delete Network with name %s, Router with name %s-Router, Interfaces err=%s", vpcReqInfo.IId.NameId, vpcReqInfo.IId.NameId, vpcReqInfo.IId.NameId, err.Error()))
						LoggingError(hiscallInfo, createErr)
						return irs.VPCInfo{}, createErr
					}
				}
			}
			// AddInterface Error => Delete Router
			if err == nil {
				createErr := errors.New(fmt.Sprintf("Failed to Create VPC with name %s, and Finished to rollback deleting", vpcReqInfo.IId.NameId))
				LoggingError(hiscallInfo, createErr)
				return irs.VPCInfo{}, createErr
			}
			createErr := errors.New(fmt.Sprintf("Failed to Create VPC with name %s, and Finished to rollback deleting", vpcReqInfo.IId.NameId))
			// Delete Router
			routerDeleteErr := routers.Delete(vpcHandler.Client, *routerId).ExtractErr()
			if routerDeleteErr != nil {
				// AddInterface Error => Delete Router Error return error + error
				createErr = errors.New(fmt.Sprintf("Failed to Create VPC with name %s err=%s and Failed to rollback delete Network with name %s, Router with name %s-Router err=%s", vpcReqInfo.IId.NameId, err.Error(), vpcReqInfo.IId.NameId, vpcReqInfo.IId.NameId, routerDeleteErr.Error()))
			} else {
				// AddInterface Error => Delete Router Success => Delete Network
				networkDeleteErr := networks.Delete(vpcHandler.Client, vpc.ID).ExtractErr()
				if networkDeleteErr != nil {
					// AddInterface Error => Delete Router Success => Delete Network Error return error + error
					createErr = errors.New(fmt.Sprintf("Failed to Create VPC with name %s err=%s, and Failed to rollback delete Network with name %s err=%s", vpcReqInfo.IId.NameId, err.Error(), vpcReqInfo.IId.NameId, networkDeleteErr.Error()))
				}
			}
			LoggingError(hiscallInfo, createErr)
			return irs.VPCInfo{}, createErr
		}
	}

	return vpcInfo, nil
}
func (vpcHandler *OpenStackVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.Client.IdentityEndpoint, call.VPCSUBNET, VPC, "ListVPC()")

	listOpts := external.ListOptsExt{
		ListOptsBuilder: networks.ListOpts{},
	}

	start := call.Start()
	page, err := networks.List(vpcHandler.Client, listOpts).AllPages()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VPC List, err=%s", err.Error()))
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	LoggingInfo(hiscallInfo, start)

	var vpcList []NetworkWithExt
	err = networks.ExtractNetworksInto(page, &vpcList)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VPC List, err=%s", err.Error()))
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
	var vpc NetworkWithExt
	start := call.Start()

	if iidCheck := CheckIIDValidation(vpcIID); !iidCheck {
		getErr := errors.New(fmt.Sprintf("Failed to Get VPC err = InValid IID"))
		LoggingError(hiscallInfo, getErr)
		return irs.VPCInfo{}, getErr
	}
	if vpcIID.SystemId == "" {
		listOpts := external.ListOptsExt{
			ListOptsBuilder: networks.ListOpts{
				Name: vpcIID.NameId,
			},
		}
		page, err := networks.List(vpcHandler.Client, listOpts).AllPages()
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to Get VPC with Id %s, err=%s", vpcIID.SystemId, err.Error()))
			LoggingError(hiscallInfo, getErr)
			return irs.VPCInfo{}, getErr
		}
		LoggingInfo(hiscallInfo, start)

		var vpcList []NetworkWithExt
		err = networks.ExtractNetworksInto(page, &vpcList)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to Get VPC with Id %s, err=%s", vpcIID.SystemId, err.Error()))
			LoggingError(hiscallInfo, getErr)
			return irs.VPCInfo{}, getErr
		}

		for _, vpc := range vpcList {
			if vpc.Name == vpcIID.NameId {
				vpcInfo := vpcHandler.setterVPC(vpc)
				return *vpcInfo, nil
			}
		}
		notExistVpcErr := errors.New(fmt.Sprintf("Failed to Get VPC with Id %s, not Exist VPC", vpcIID.SystemId))
		getErr := errors.New(fmt.Sprintf("Failed to Get VPC with Id %s, err=%s", vpcIID.SystemId, notExistVpcErr))
		LoggingError(hiscallInfo, getErr)
		return irs.VPCInfo{}, getErr
	} else {
		err := networks.Get(vpcHandler.Client, vpcIID.SystemId).ExtractInto(&vpc)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to Get VPC with Id %s, err=%s", vpcIID.SystemId, err.Error()))
			LoggingError(hiscallInfo, getErr)
			return irs.VPCInfo{}, getErr
		}
		LoggingInfo(hiscallInfo, start)

		vpcInfo := vpcHandler.setterVPC(vpc)
		return *vpcInfo, nil
	}

}

func (vpcHandler *OpenStackVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.Client.IdentityEndpoint, call.VPCSUBNET, vpcIID.NameId, "DeleteVPC()")

	vpcInfo, err := vpcHandler.GetVPC(vpcIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Delete VPC with name %s err=%s", vpcIID.NameId, err.Error()))
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}

	// Delete Interface
	routerId, err := vpcHandler.GetRouter(vpcIID.NameId)
	if err == nil {
		// Delete Interface
		for _, subnet := range vpcInfo.SubnetInfoList {
			if ok, err := vpcHandler.DeleteInterface(subnet.IId.SystemId, *routerId); !ok {
				if err != nil && err.Error() != ResourceNotFound {
					// DeleteInterface Error except Resource not found
					getErr := errors.New(fmt.Sprintf("Failed to Delete VPC with name %s err=%s", vpcIID.NameId, err.Error()))
					LoggingError(hiscallInfo, getErr)
					return false, getErr
				}
			}
		}
		// Delete Router
		if routerId != nil {
			err = routers.Delete(vpcHandler.Client, *routerId).ExtractErr()
			if err != nil {
				getErr := errors.New(fmt.Sprintf("Failed to Delete VPC with name %s err=%s", vpcIID.NameId, err.Error()))
				LoggingError(hiscallInfo, getErr)
				return false, getErr
			}
		}
	} else if err.Error() != ResourceNotFound {
		getErr := errors.New(fmt.Sprintf("Failed to Delete VPC with name %s err=%s", vpcIID.NameId, err.Error()))
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}
	// TODO: nested flow 개선
	//Delete VPC
	start := call.Start()
	err = networks.Delete(vpcHandler.Client, vpcInfo.IId.SystemId).ExtractErr()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Delete VPC with name %s err=%s", vpcIID.NameId, err.Error()))
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
		IPVersion:      gophercloud.IPv4,
		DNSNameservers: []string{DNSNameservers},
	}
	subnet, err := subnets.Create(vpcHandler.Client, subnetCreateOpts).Extract()
	if err != nil {
		cblogger.Error("Failed to Create Subnet with name %s, err=%s", reqSubnetInfo.IId.NameId, err)
		return irs.SubnetInfo{}, err
	}
	subnetInfo := vpcHandler.setterSubnet(*subnet)
	return *subnetInfo, nil
}

func (vpcHandler *OpenStackVPCHandler) GetSubnet(subnetIId irs.IID) (irs.SubnetInfo, error) {
	subnet, err := subnets.Get(vpcHandler.Client, subnetIId.SystemId).Extract()
	if err != nil {
		cblogger.Error("Failed to Get Subnet with Id %s, err=%s", subnetIId.SystemId, err)
		return irs.SubnetInfo{}, nil
	}
	subnetInfo := vpcHandler.setterSubnet(*subnet)
	return *subnetInfo, nil
}

func (vpcHandler *OpenStackVPCHandler) DeleteSubnet(subnetIId irs.IID) (bool, error) {
	err := subnets.Delete(vpcHandler.Client, subnetIId.SystemId).ExtractErr()
	if err != nil {
		cblogger.Error("Failed to Delete Subnet with Id %s, err=%s", subnetIId.SystemId, err)
		return false, err
	}
	return true, nil
}

func (vpcHandler *OpenStackVPCHandler) CreateRouter(vpcName string) (*string, error) {
	externVPCId, _ := GetPublicVPCInfo(vpcHandler.Client, "ID")
	routerName := vpcName + "-Router"
	AdminStateUp := true
	createOpts := routers.CreateOpts{
		Name:         routerName,
		AdminStateUp: &AdminStateUp,
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
		cblogger.Error("Failed to Get Router List, err=%s", err)
		return nil, err
	}
	routerList, err := routers.ExtractRouters(page)
	if err != nil {
		cblogger.Error("Failed to extract Router, err=%s", err)
		return nil, err
	}
	if len(routerList) != 1 {
		notExistErr := errors.New(ResourceNotFound)
		cblogger.Error("Failed to Get Router with name %s, err=%s", routerName, notExistErr)
		return nil, notExistErr
	}

	routerId := routerList[0].ID
	return &routerId, nil
}

func (vpcHandler *OpenStackVPCHandler) DeleteRouter(vpcName string) (bool, error) {
	// Get Router
	routerId, err := vpcHandler.GetRouter(vpcName)
	if err != nil {
		if err.Error() == ResourceNotFound {
			cblogger.Error("Failed to Delete Router with Id %s, err=%s", routerId, ResourceNotFound)
			return false, err
		}
		cblogger.Error("Failed to Delete Router with Id %s, err=%s", routerId)
		return false, err
	}
	// Delete Router
	err = routers.Delete(vpcHandler.Client, *routerId).ExtractErr()
	if err != nil {
		cblogger.Error("Failed to Delete Router with Id %s, err=%s", routerId)
		return false, err
	}
	return true, nil
}

////////////////////////////////////TEST 진행
func (vpcHandler *OpenStackVPCHandler) AddInterface(subnetId string, routerId string) (bool, error) {
	createOpts := routers.AddInterfaceOpts{
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
	deleteOpts := routers.RemoveInterfaceOpts{
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
		IPVersion:      gophercloud.IPv4,
		DNSNameservers: []string{DNSNameservers},
	}

	start := call.Start()
	_, err := subnets.Create(vpcHandler.Client, subnetCreateOpts).Extract()
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Subnet with name %s, err=%s", subnetCreateOpts.Name, err.Error()))
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
