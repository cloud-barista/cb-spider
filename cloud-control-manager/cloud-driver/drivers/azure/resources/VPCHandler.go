package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	VPC = "VPC"
)

type AzureVPCHandler struct {
	Region       idrv.RegionInfo
	Ctx          context.Context
	Client       *armnetwork.VirtualNetworksClient
	SubnetClient *armnetwork.SubnetsClient
}

func (vpcHandler *AzureVPCHandler) setterVPC(network *armnetwork.VirtualNetwork) *irs.VPCInfo {
	vpcInfo := &irs.VPCInfo{
		IId: irs.IID{
			NameId:   *network.Name,
			SystemId: *network.ID,
		},
		IPv4_CIDR:    *(network.Properties.AddressSpace.AddressPrefixes)[0],
		KeyValueList: []irs.KeyValue{{Key: "ResourceGroup", Value: vpcHandler.Region.Region}},
	}
	subnetArr := make([]irs.SubnetInfo, len(network.Properties.Subnets))
	for i, subnet := range network.Properties.Subnets {
		subnetArr[i] = *vpcHandler.setterSubnet(subnet)
	}
	vpcInfo.SubnetInfoList = subnetArr

	if network.Tags != nil {
		vpcInfo.TagList = setTagList(network.Tags)
	}
	return vpcInfo
}

func (vpcHandler *AzureVPCHandler) setterSubnet(subnet *armnetwork.Subnet) *irs.SubnetInfo {
	subnetInfo := &irs.SubnetInfo{
		IId: irs.IID{
			NameId:   *subnet.Name,
			SystemId: *subnet.ID,
		},
		IPv4_CIDR:    *subnet.Properties.AddressPrefix,
		KeyValueList: []irs.KeyValue{{Key: "ResourceGroup", Value: vpcHandler.Region.Region}},
	}
	return subnetInfo
}

func (vpcHandler *AzureVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, VPC, "CreateVPC()")

	// Check VPC Exists
	vpc, _ := vpcHandler.Client.Get(vpcHandler.Ctx, vpcHandler.Region.Region, vpcReqInfo.IId.NameId, nil)
	if vpc.ID != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = vpc with name %s already exist", vpcReqInfo.IId.NameId))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}
	// Create Tag
	tags := setTags(vpcReqInfo.TagList)

	// Create VPC
	createOpts := armnetwork.VirtualNetwork{
		Name: &vpcReqInfo.IId.NameId,
		Properties: &armnetwork.VirtualNetworkPropertiesFormat{
			AddressSpace: &armnetwork.AddressSpace{
				AddressPrefixes: []*string{&vpcReqInfo.IPv4_CIDR},
			},
		},
		Location: toStrPtr(vpcHandler.Region.Region),
		Tags:     tags,
	}

	start := call.Start()
	poller, err := vpcHandler.Client.BeginCreateOrUpdate(vpcHandler.Ctx, vpcHandler.Region.Region, vpcReqInfo.IId.NameId, createOpts, nil)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}
	_, err = poller.PollUntilDone(vpcHandler.Ctx, nil)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
		cblogger.Error(createErr.Error())

	}
	LoggingInfo(hiscallInfo, start)

	// Create Subnet
	var subnetCreateOpts armnetwork.Subnet
	for _, subnet := range vpcReqInfo.SubnetInfoList {
		subnetCreateOpts = armnetwork.Subnet{
			Name: &subnet.IId.NameId,
			Properties: &armnetwork.SubnetPropertiesFormat{
				AddressPrefix: toStrPtr(subnet.IPv4_CIDR),
			},
		}
		poller, err := vpcHandler.SubnetClient.BeginCreateOrUpdate(vpcHandler.Ctx, vpcHandler.Region.Region, vpcReqInfo.IId.NameId, subnet.IId.NameId, subnetCreateOpts, nil)
		if err != nil {
			cblogger.Error(fmt.Sprintf("failed to create subnet with name %s", subnet.IId.NameId))
			continue
		}
		_, err = poller.PollUntilDone(vpcHandler.Ctx, nil)
		if err != nil {
			cblogger.Error(fmt.Sprintf("failed to get subnet with name %s", subnet.IId.NameId))
			continue
		}
	}

	// 생성된 VNetwork 정보 리턴
	vpcInfo, err := vpcHandler.GetVPC(irs.IID{NameId: vpcReqInfo.IId.NameId})
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}
	return vpcInfo, nil
}

func (vpcHandler *AzureVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, VPC, "ListVPC()")

	start := call.Start()

	var networkList []*armnetwork.VirtualNetwork

	pager := vpcHandler.Client.NewListPager(vpcHandler.Region.Region, nil)

	for pager.More() {
		page, err := pager.NextPage(vpcHandler.Ctx)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List VPC err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return nil, getErr
		}

		for _, vpc := range page.Value {
			networkList = append(networkList, vpc)
		}
	}

	vpcInfoList := make([]*irs.VPCInfo, len(networkList))
	for i, vpc := range networkList {
		vpcInfoList[i] = vpcHandler.setterVPC(vpc)
	}
	LoggingInfo(hiscallInfo, start)

	return vpcInfoList, nil
}

func (vpcHandler *AzureVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, vpcIID.NameId, "GetVPC()")

	start := call.Start()
	vpc, err := vpcHandler.getRawVPC(vpcIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VPC err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VPCInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	vpcInfo := vpcHandler.setterVPC(vpc)
	return *vpcInfo, nil
}

func (vpcHandler *AzureVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, vpcIID.NameId, "DeleteVPC()")

	start := call.Start()
	poller, err := vpcHandler.Client.BeginDelete(vpcHandler.Ctx, vpcHandler.Region.Region, vpcIID.NameId, nil)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete VPC err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	_, err = poller.PollUntilDone(vpcHandler.Ctx, nil)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete VPC err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func (vpcHandler *AzureVPCHandler) AddSubnet(vpcIID irs.IID, subnetInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, subnetInfo.IId.NameId, "AddSubnet()")

	vpc, err := vpcHandler.getRawVPC(vpcIID)
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to AddSubnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}
	subnetCreateOpts := armnetwork.Subnet{
		Name: &subnetInfo.IId.NameId,
		Properties: &armnetwork.SubnetPropertiesFormat{
			AddressPrefix: toStrPtr(subnetInfo.IPv4_CIDR),
		},
	}
	start := call.Start()
	poller, err := vpcHandler.SubnetClient.BeginCreateOrUpdate(vpcHandler.Ctx, vpcHandler.Region.Region, *vpc.Name, subnetInfo.IId.NameId, subnetCreateOpts, nil)
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to AddSubnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}
	_, err = poller.PollUntilDone(vpcHandler.Ctx, nil)
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to AddSubnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
	}
	result, err := vpcHandler.GetVPC(irs.IID{NameId: vpcIID.NameId})
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to AddSubnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}
	LoggingInfo(hiscallInfo, start)
	return result, nil
}

func (vpcHandler *AzureVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, subnetIID.NameId, "RemoveSubnet()")
	start := call.Start()
	poller, err := vpcHandler.SubnetClient.BeginDelete(vpcHandler.Ctx, vpcHandler.Region.Region, vpcIID.NameId, subnetIID.NameId, nil)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to RemoveSubnet err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	_, err = poller.PollUntilDone(vpcHandler.Ctx, nil)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to RemoveSubnet err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}
func (vpcHandler *AzureVPCHandler) getRawVPC(vpcIID irs.IID) (*armnetwork.VirtualNetwork, error) {
	if vpcIID.SystemId == "" && vpcIID.NameId == "" {
		return nil, errors.New("invalid IID")
	}
	if vpcIID.NameId == "" {
		var networkList []*armnetwork.VirtualNetwork

		pager := vpcHandler.Client.NewListPager(vpcHandler.Region.Region, nil)

		for pager.More() {
			page, err := pager.NextPage(vpcHandler.Ctx)
			if err != nil {
				return nil, err
			}

			for _, vpc := range page.Value {
				networkList = append(networkList, vpc)
			}
		}

		for _, vpc := range networkList {
			if *vpc.ID == vpcIID.SystemId {
				return vpc, nil
			}
		}
		return nil, errors.New("not found SecurityGroup")
	} else {
		resp, err := vpcHandler.Client.Get(vpcHandler.Ctx, vpcHandler.Region.Region, vpcIID.NameId, nil)
		if err != nil {
			return nil, err
		}

		return &resp.VirtualNetwork, err
	}
}

func getRawVirtualNetwork(vpcIID irs.IID, virtualNetworksClient *armnetwork.VirtualNetworksClient, ctx context.Context, resourceGroup string) (*armnetwork.VirtualNetwork, error) {
	if vpcIID.SystemId == "" && vpcIID.NameId == "" {
		return nil, errors.New("invalid IID")
	}
	if vpcIID.NameId == "" {
		var networkList []*armnetwork.VirtualNetwork

		pager := virtualNetworksClient.NewListPager(resourceGroup, nil)

		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, vpc := range page.Value {
				networkList = append(networkList, vpc)
			}
		}

		for _, vpc := range networkList {
			if *vpc.ID == vpcIID.SystemId {
				return vpc, nil
			}
		}
		return nil, errors.New("not found SecurityGroup")
	} else {
		resp, err := virtualNetworksClient.Get(ctx, resourceGroup, vpcIID.NameId, nil)
		if err != nil {
			return nil, err
		}

		return &resp.VirtualNetwork, err
	}
}

func (vpcHandler *AzureVPCHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	return nil, errors.New("Does not support ListIID() yet!!")
}
