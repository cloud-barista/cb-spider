package resources

import (
	"errors"
	"fmt"

	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
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
	CredentialInfo idrv.CredentialInfo
	IdentityClient *gophercloud.ServiceClient
	ComputeClient  *gophercloud.ServiceClient
	NetworkClient  *gophercloud.ServiceClient
	NLBClient      *gophercloud.ServiceClient
}

type NetworkWithExt struct {
	networks.Network
	external.NetworkExternalExt
}

func (vpcHandler *OpenStackVPCHandler) setterVPC(nvpc NetworkWithExt) *irs.VPCInfo {
	var tags []irs.KeyValue

	for _, tag := range nvpc.Tags {
		tags = append(tags, tagsToKeyValue(tag))
	}

	// VPC 정보 맵핑
	vpcInfo := irs.VPCInfo{
		IId: irs.IID{
			NameId:   nvpc.Name,
			SystemId: nvpc.ID,
		},
		IPv4_CIDR: "OpenStack VPC does not support IPv4_CIDR",
		TagList:   tags,
	}
	//var External string
	//if nvpc.External == true {
	//	External = "Yes"
	//} else if nvpc.External == false {
	//	External = "No"
	//}
	//keyValueList := []irs.KeyValue{
	//	{Key: "External Network", Value: External},
	//}
	//vpcInfo.KeyValueList = keyValueList

	vpcInfo.KeyValueList = irs.StructToKeyValueList(nvpc)

	// 서브넷 정보 조회
	subnetInfoList := make([]irs.SubnetInfo, len(nvpc.Subnets))

	for i, subnetId := range nvpc.Subnets {
		subnet, err := vpcHandler.GetSubnet(irs.IID{SystemId: subnetId})
		if err != nil {
			cblogger.Error("Failed to Get Subnet with Id %s, err=%s", subnetId, err)
			continue
		}
		subnetInfoList[i] = subnet
		subnet.KeyValueList = irs.StructToKeyValueList(subnet)
		//subnetInfoList[i] = subnetInfo
	}
	vpcInfo.SubnetInfoList = subnetInfoList

	return &vpcInfo
}

func (vpcHandler *OpenStackVPCHandler) setterSubnet(subnet subnets.Subnet) *irs.SubnetInfo {
	var tags []irs.KeyValue

	for _, tag := range subnet.Tags {
		tags = append(tags, tagsToKeyValue(tag))
	}

	subnetInfo := irs.SubnetInfo{
		IId: irs.IID{
			NameId:   subnet.Name,
			SystemId: subnet.ID,
		},
		IPv4_CIDR: subnet.CIDR,
		//TagList:   tags,
	}

	subnetInfo.KeyValueList = irs.StructToKeyValueList(subnet)

	return &subnetInfo
}

func (vpcHandler *OpenStackVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (createdVPC irs.VPCInfo, createErr error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.NetworkClient.IdentityEndpoint, call.VPCSUBNET, vpcReqInfo.IId.NameId, "CreateVPC()")
	start := call.Start()

	// Check VPC Exists
	listOpts := networks.ListOpts{Name: vpcReqInfo.IId.NameId}
	page, err := networks.List(vpcHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}
	vpcList, err := networks.ExtractNetworks(page)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}
	if len(vpcList) != 0 {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC. err = The VPC name %s already exists", vpcReqInfo.IId.NameId))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}

	// Create - Network
	createOpts := networks.CreateOpts{
		Name: vpcReqInfo.IId.NameId,
	}

	vpc, err := networks.Create(vpcHandler.NetworkClient, createOpts).Extract()
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC with name %s err=%s and Finished to rollback deleting", vpcReqInfo.IId.NameId, err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}
	LoggingInfo(hiscallInfo, start)

	defer func() {
		if createErr != nil {
			cleanVPCIId := irs.IID{
				SystemId: vpc.ID,
				NameId:   vpc.Name,
			}
			cleanErr := vpcHandler.vpcCleaner(cleanVPCIId)
			if cleanErr != nil {
				createErr = errors.New(fmt.Sprintf("%s Failed to rollback deleting err = %s", createErr, cleanErr))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
			}
		}
	}()

	// Create Router
	routerId, err := vpcHandler.CreateRouter(vpcReqInfo.IId.NameId)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}

	// Create Subnet
	for _, subnet := range vpcReqInfo.SubnetInfoList {
		_, err := vpcHandler.CreateSubnet(vpc.ID, subnet)
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VPCInfo{}, createErr
		}
	}

	// Tagging
	tagHandler := OpenStackTagHandler{
		CredentialInfo: vpcHandler.CredentialInfo,
		IdentityClient: vpcHandler.IdentityClient,
		ComputeClient:  vpcHandler.ComputeClient,
		NetworkClient:  vpcHandler.NetworkClient,
		NLBClient:      vpcHandler.NLBClient,
	}

	var errTags []irs.KeyValue
	var errMsg string
	for _, tag := range vpcReqInfo.TagList {
		_, err = tagHandler.AddTag(irs.VPC, irs.IID{SystemId: vpc.ID}, tag)
		if err != nil {
			cblogger.Error(err)
			errTags = append(errTags, tag)
			errMsg += err.Error() + ", "
		}
	}
	if len(errTags) > 0 {
		return irs.VPCInfo{}, returnTaggingError(errTags, errMsg[:len(errMsg)-2])
	}

	vpcInfo, err := vpcHandler.GetVPC(irs.IID{SystemId: vpc.ID})
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}

	// Create Interface
	for _, subnet := range vpcInfo.SubnetInfoList {
		if ok, err := vpcHandler.AddInterface(subnet.IId.SystemId, *routerId); !ok {
			if err != nil {
				createErr = errors.New(fmt.Sprintf("Failed to Create VPC with name %s err=%s", vpcReqInfo.IId.NameId, err.Error()))
			} else {
				createErr = errors.New(fmt.Sprintf("Failed to Create VPC with name %s", vpcReqInfo.IId.NameId))
			}
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VPCInfo{}, createErr
		}
	}

	return vpcInfo, nil
}
func (vpcHandler *OpenStackVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.NetworkClient.IdentityEndpoint, call.VPCSUBNET, VPC, "ListVPC()")

	listOpts := external.ListOptsExt{
		ListOptsBuilder: networks.ListOpts{},
	}

	start := call.Start()
	page, err := networks.List(vpcHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List VPC err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	LoggingInfo(hiscallInfo, start)

	var vpcList []NetworkWithExt
	err = networks.ExtractNetworksInto(page, &vpcList)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List VPC err = %s", err.Error()))
		cblogger.Error(getErr.Error())
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
	hiscallInfo := GetCallLogScheme(vpcHandler.NetworkClient.IdentityEndpoint, call.VPCSUBNET, vpcIID.NameId, "GetVPC()")
	//var vpc NetworkWithExt
	start := call.Start()
	vpc, err := vpcHandler.getRawVPC(vpcIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VPC err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VPCInfo{}, getErr
	}
	vpcInfo := vpcHandler.setterVPC(*vpc)
	LoggingInfo(hiscallInfo, start)
	return *vpcInfo, nil
}

func (vpcHandler *OpenStackVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.NetworkClient.IdentityEndpoint, call.VPCSUBNET, vpcIID.NameId, "DeleteVPC()")

	start := call.Start()
	err := vpcHandler.vpcCleaner(vpcIID)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete VPC err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
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
	subnet, err := subnets.Create(vpcHandler.NetworkClient, subnetCreateOpts).Extract()
	if err != nil {
		cblogger.Error("Failed to Create Subnet with name %s, err=%s", reqSubnetInfo.IId.NameId, err)
		return irs.SubnetInfo{}, err
	}

	// Tagging
	tagHandler := OpenStackTagHandler{
		CredentialInfo: vpcHandler.CredentialInfo,
		IdentityClient: vpcHandler.IdentityClient,
		ComputeClient:  vpcHandler.ComputeClient,
		NetworkClient:  vpcHandler.NetworkClient,
		NLBClient:      vpcHandler.NLBClient,
	}

	var errTags []irs.KeyValue
	var errMsg string
	for _, tag := range reqSubnetInfo.TagList {
		_, err = tagHandler.AddTag(irs.SUBNET, irs.IID{SystemId: subnet.ID}, tag)
		if err != nil {
			cblogger.Error(err)
			errTags = append(errTags, tag)
			errMsg += err.Error() + ", "
		}
	}
	if len(errTags) > 0 {
		return irs.SubnetInfo{}, returnTaggingError(errTags, errMsg[:len(errMsg)-2])
	}

	subnetInfo := vpcHandler.setterSubnet(*subnet)

	return *subnetInfo, nil
}

func (vpcHandler *OpenStackVPCHandler) GetSubnet(subnetIId irs.IID) (irs.SubnetInfo, error) {
	subnet, err := subnets.Get(vpcHandler.NetworkClient, subnetIId.SystemId).Extract()
	if err != nil {
		cblogger.Error("Failed to Get Subnet with Id %s, err=%s", subnetIId.SystemId, err)
		return irs.SubnetInfo{}, nil
	}
	subnetInfo := vpcHandler.setterSubnet(*subnet)
	return *subnetInfo, nil
}

func (vpcHandler *OpenStackVPCHandler) DeleteSubnet(subnetIId irs.IID) (bool, error) {
	err := subnets.Delete(vpcHandler.NetworkClient, subnetIId.SystemId).ExtractErr()
	if err != nil {
		cblogger.Error("Failed to Delete Subnet with Id %s, err=%s", subnetIId.SystemId, err)
		return false, err
	}
	return true, nil
}

func (vpcHandler *OpenStackVPCHandler) CreateRouter(vpcName string) (*string, error) {
	externVPCId, _ := GetPublicVPCInfo(vpcHandler.NetworkClient, "ID")
	routerName := vpcName
	AdminStateUp := true
	createOpts := routers.CreateOpts{
		Name:         routerName,
		AdminStateUp: &AdminStateUp,
		GatewayInfo: &routers.GatewayInfo{
			NetworkID: externVPCId,
		},
	}

	// Create Router
	router, err := routers.Create(vpcHandler.NetworkClient, createOpts).Extract()
	if err != nil {
		return nil, err
	}
	return &router.ID, nil
}

func (vpcHandler *OpenStackVPCHandler) GetRouter(vpcName string) (*string, error) {
	// Get Router Info
	routerName := vpcName
	listOpts := routers.ListOpts{Name: routerName}
	page, err := routers.List(vpcHandler.NetworkClient, listOpts).AllPages()
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
	err = routers.Delete(vpcHandler.NetworkClient, *routerId).ExtractErr()
	if err != nil {
		cblogger.Error("Failed to Delete Router with Id %s, err=%s", routerId)
		return false, err
	}
	return true, nil
}

func (vpcHandler *OpenStackVPCHandler) AddInterface(subnetId string, routerId string) (bool, error) {
	createOpts := routers.AddInterfaceOpts{
		SubnetID: subnetId,
	}

	// Add Interface
	_, err := routers.AddInterface(vpcHandler.NetworkClient, routerId, createOpts).Extract()
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
	_, err := routers.RemoveInterface(vpcHandler.NetworkClient, routerId, deleteOpts).Extract()
	if err != nil {
		return false, err
	}
	return true, nil
}

func (vpcHandler *OpenStackVPCHandler) AddSubnetInterfaceToVPCRouter(vpcName string, subnetID string) error {
	routerId, err := vpcHandler.GetRouter(vpcName)
	if err != nil {
		return err
	}

	addOpts := routers.AddInterfaceOpts{
		SubnetID: subnetID,
	}

	_, err = routers.AddInterface(vpcHandler.NetworkClient, *routerId, addOpts).Extract()
	if err != nil {
		return fmt.Errorf("failed to add interface to router: %v", err)
	}

	return nil
}

func (vpcHandler *OpenStackVPCHandler) AddSubnet(vpcIID irs.IID, subnetInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.NetworkClient.IdentityEndpoint, call.VPCSUBNET, subnetInfo.IId.NameId, "AddSubnet()")

	vpc, err := vpcHandler.GetVPC(vpcIID)
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}

	subnetCreateOpts := subnets.CreateOpts{
		NetworkID:      vpc.IId.SystemId,
		Name:           subnetInfo.IId.NameId,
		CIDR:           subnetInfo.IPv4_CIDR,
		IPVersion:      gophercloud.IPv4,
		DNSNameservers: []string{DNSNameservers},
	}

	start := call.Start()
	subnet, err := subnets.Create(vpcHandler.NetworkClient, subnetCreateOpts).Extract()
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}

	err = vpcHandler.AddSubnetInterfaceToVPCRouter(vpc.IId.NameId, subnet.ID)
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}

	result, err := vpcHandler.GetVPC(vpcIID)
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}

	LoggingInfo(hiscallInfo, start)

	return result, nil
}

func (vpcHandler *OpenStackVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.NetworkClient.IdentityEndpoint, call.VPCSUBNET, subnetIID.NameId, "RemoveSubnet()")

	start := call.Start()
	err := subnets.Delete(vpcHandler.NetworkClient, subnetIID.SystemId).ExtractErr()
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Remove Subnet err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func (vpcHandler *OpenStackVPCHandler) getRawRouter(vpcName string) (router routers.Router, err error) {
	routerName := vpcName
	listOpts := routers.ListOpts{Name: routerName}
	page, err := routers.List(vpcHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		cblogger.Error("Failed to Get Router List, err=%s", err)
		return routers.Router{}, err
	}
	routerList, err := routers.ExtractRouters(page)
	if err != nil {
		cblogger.Error("Failed to extract Router, err=%s", err)
		return routers.Router{}, err
	}
	if len(routerList) != 1 {
		notExistErr := errors.New(ResourceNotFound)
		cblogger.Error("Failed to Get Router with name %s, err=%s", routerName, notExistErr)
		return routers.Router{}, notExistErr
	}
	return routerList[0], nil
}

func (vpcHandler *OpenStackVPCHandler) vpcCleaner(vpcIId irs.IID) error {
	// VPC
	vpc, err := vpcHandler.GetVPC(vpcIId)
	if err != nil {
		return err
	}
	pager, err := servers.List(vpcHandler.ComputeClient, nil).AllPages()
	if err != nil {
		return err
	}

	serverList, err := servers.ExtractServers(pager)
	if err != nil {
		return err
	}
	for _, server := range serverList {
		for k, _ := range server.Addresses {
			if k == vpc.IId.NameId {
				return errors.New("vm exists on this VPC.")
			}
		}
	}
	listOpts := routers.ListOpts{Name: vpc.IId.NameId}
	page, err := routers.List(vpcHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		return err
	}
	routerList, err := routers.ExtractRouters(page)
	if err != nil {
		return err
	}
	if len(routerList) == 0 {
		// Not Exist Route Only VPC Delete
		err = networks.Delete(vpcHandler.NetworkClient, vpc.IId.SystemId).ExtractErr()
		if err != nil {
			return err
		}
		return nil
	}
	if len(routerList) == 1 {
		// Exist Route
		router := routerList[0]
		for _, subnet := range vpc.SubnetInfoList {
			vpcHandler.DeleteInterface(subnet.IId.SystemId, router.ID)
		}
		err = routers.Delete(vpcHandler.NetworkClient, router.ID).ExtractErr()
		if err != nil {
			return err
		}
		err = networks.Delete(vpcHandler.NetworkClient, vpc.IId.SystemId).ExtractErr()
		if err != nil {
			return err
		}
		return nil
	}
	return errors.New("unexpected error")
}

func (vpcHandler *OpenStackVPCHandler) getRawVPC(vpcIID irs.IID) (*NetworkWithExt, error) {
	if !CheckIIDValidation(vpcIID) {
		return nil, errors.New("invalid IID")
	}
	var vpc NetworkWithExt
	if vpcIID.SystemId == "" {
		listOpts := external.ListOptsExt{
			ListOptsBuilder: networks.ListOpts{
				Name: vpcIID.NameId,
			},
		}
		page, err := networks.List(vpcHandler.NetworkClient, listOpts).AllPages()
		if err != nil {
			return nil, err
		}

		var vpcList []NetworkWithExt
		err = networks.ExtractNetworksInto(page, &vpcList)
		if err != nil {
			return nil, err
		}

		for _, vpc := range vpcList {
			if vpc.Name == vpcIID.NameId {
				return &vpc, nil
			}
		}
		return nil, errors.New("not found vpc")
	} else {
		err := networks.Get(vpcHandler.NetworkClient, vpcIID.SystemId).ExtractInto(&vpc)
		if err != nil {
			return nil, err
		}
		return &vpc, nil
	}
}

func (vpcHandler *OpenStackVPCHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(vpcHandler.NetworkClient.IdentityEndpoint, call.VPCSUBNET, "VPC", "ListIID()")

	start := call.Start()

	var iidList []*irs.IID

	listOpts := networks.ListOpts{}

	allPages, err := networks.List(vpcHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VPC information from Openstack!! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(hiscallInfo, newErr)
		return make([]*irs.IID, 0), newErr
	}

	allNetworks, err := networks.ExtractNetworks(allPages)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get VPC List from Openstack! : [%v] ", err)
		cblogger.Error(newErr.Error())
		LoggingError(hiscallInfo, newErr)
		return make([]*irs.IID, 0), newErr
	}

	for _, vpc := range allNetworks {
		var iid irs.IID
		iid.SystemId = vpc.ID
		iid.NameId = vpc.Name

		iidList = append(iidList, &iid)
	}

	LoggingInfo(hiscallInfo, start)

	return iidList, nil
}
