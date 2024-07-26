package resources

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
	"github.com/Azure/go-autorest/autorest/to"

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
	Client       *network.VirtualNetworksClient
	SubnetClient *network.SubnetsClient
}

func (vpcHandler *AzureVPCHandler) setterVPC(network network.VirtualNetwork) *irs.VPCInfo {
	vpcInfo := &irs.VPCInfo{
		IId: irs.IID{
			NameId:   *network.Name,
			SystemId: *network.ID,
		},
		IPv4_CIDR:    (*network.AddressSpace.AddressPrefixes)[0],
		KeyValueList: []irs.KeyValue{{Key: "ResourceGroup", Value: vpcHandler.Region.Region}},
	}
	subnetArr := make([]irs.SubnetInfo, len(*network.Subnets))
	for i, subnet := range *network.Subnets {
		subnetArr[i] = *vpcHandler.setterSubnet(subnet)
	}
	vpcInfo.SubnetInfoList = subnetArr

	if network.Tags != nil {
		vpcInfo.TagList = setTagList(network.Tags)
	}
	return vpcInfo
}

func (vpcHandler *AzureVPCHandler) setterSubnet(subnet network.Subnet) *irs.SubnetInfo {
	subnetInfo := &irs.SubnetInfo{
		IId: irs.IID{
			NameId:   *subnet.Name,
			SystemId: *subnet.ID,
		},
		IPv4_CIDR:    *subnet.AddressPrefix,
		KeyValueList: []irs.KeyValue{{Key: "ResourceGroup", Value: vpcHandler.Region.Region}},
	}
	return subnetInfo
}

func (vpcHandler *AzureVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, VPC, "CreateVPC()")

	// Check VPC Exists
	vpc, _ := vpcHandler.Client.Get(vpcHandler.Ctx, vpcHandler.Region.Region, vpcReqInfo.IId.NameId, "")
	if vpc.ID != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = vpc with name %s already exist", vpcReqInfo.IId.NameId))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}
	// Create Tag
	tags := setTags(vpcReqInfo.TagList)

	// Create VPC
	createOpts := network.VirtualNetwork{
		Name: to.StringPtr(vpcReqInfo.IId.NameId),
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{
				AddressPrefixes: &[]string{vpcReqInfo.IPv4_CIDR},
			},
		},
		Location: &vpcHandler.Region.Region,
		Tags: tags,
	}

	start := call.Start()
	future, err := vpcHandler.Client.CreateOrUpdate(vpcHandler.Ctx, vpcHandler.Region.Region, vpcReqInfo.IId.NameId, createOpts)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}
	err = future.WaitForCompletionRef(vpcHandler.Ctx, vpcHandler.Client.Client)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}
	LoggingInfo(hiscallInfo, start)

	// Create Subnet
	var subnetCreateOpts network.Subnet
	for _, subnet := range vpcReqInfo.SubnetInfoList {
		subnetCreateOpts = network.Subnet{
			Name: to.StringPtr(subnet.IId.NameId),
			SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
				AddressPrefix: to.StringPtr(subnet.IPv4_CIDR),
			},
		}
		future, err := vpcHandler.SubnetClient.CreateOrUpdate(vpcHandler.Ctx, vpcHandler.Region.Region, vpcReqInfo.IId.NameId, subnet.IId.NameId, subnetCreateOpts)
		if err != nil {
			cblogger.Error(fmt.Sprintf("failed to create subnet with name %s", subnet.IId.NameId))
			continue
		}
		err = future.WaitForCompletionRef(vpcHandler.Ctx, vpcHandler.Client.Client)
		if err != nil {
			cblogger.Error(fmt.Sprintf("failed to create subnet with name %s", subnet.IId.NameId))
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
	vpcList, err := vpcHandler.Client.List(vpcHandler.Ctx, vpcHandler.Region.Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List VPC err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	LoggingInfo(hiscallInfo, start)

	vpcInfoList := make([]*irs.VPCInfo, len(vpcList.Values()))
	for i, vpc := range vpcList.Values() {
		vpcInfoList[i] = vpcHandler.setterVPC(vpc)
	}
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
	fmt.Printf("vpc : %+v" , vpc)
	vpcInfo := vpcHandler.setterVPC(*vpc)
	return *vpcInfo, nil
}

func (vpcHandler *AzureVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.Region, call.VPCSUBNET, vpcIID.NameId, "DeleteVPC()")

	start := call.Start()
	future, err := vpcHandler.Client.Delete(vpcHandler.Ctx, vpcHandler.Region.Region, vpcIID.NameId)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete VPC err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	err = future.WaitForCompletionRef(vpcHandler.Ctx, vpcHandler.Client.Client)
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
	subnetCreateOpts := network.Subnet{
		Name: to.StringPtr(subnetInfo.IId.NameId),
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			AddressPrefix: to.StringPtr(subnetInfo.IPv4_CIDR),
		},
	}
	start := call.Start()
	future, err := vpcHandler.SubnetClient.CreateOrUpdate(vpcHandler.Ctx, vpcHandler.Region.Region, *vpc.Name, subnetInfo.IId.NameId, subnetCreateOpts)
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to AddSubnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}
	err = future.WaitForCompletionRef(vpcHandler.Ctx, vpcHandler.Client.Client)
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to AddSubnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
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
	future, err := vpcHandler.SubnetClient.Delete(vpcHandler.Ctx, vpcHandler.Region.Region, vpcIID.NameId, subnetIID.NameId)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to RemoveSubnet err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	err = future.WaitForCompletionRef(vpcHandler.Ctx, vpcHandler.Client.Client)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to RemoveSubnet err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}
func (vpcHandler *AzureVPCHandler) getRawVPC(vpcIID irs.IID) (*network.VirtualNetwork, error) {
	if vpcIID.SystemId == "" && vpcIID.NameId == "" {
		return nil, errors.New("invalid IID")
	}
	if vpcIID.NameId == "" {
		result, err := vpcHandler.Client.List(vpcHandler.Ctx, vpcHandler.Region.Region)
		if err != nil {
			return nil, err
		}
		for _, vpc := range result.Values() {
			if *vpc.ID == vpcIID.SystemId {
				return &vpc, nil
			}
		}
		return nil, errors.New("not found SecurityGroup")
	} else {
		vpc, err := vpcHandler.Client.Get(vpcHandler.Ctx, vpcHandler.Region.Region, vpcIID.NameId, "")
		return &vpc, err
	}
}

func getRawVirtualNetwork(vpcIID irs.IID, virtualNetworksClient *network.VirtualNetworksClient, ctx context.Context, resourceGroup string) (*network.VirtualNetwork, error) {
	if vpcIID.SystemId == "" && vpcIID.NameId == "" {
		return nil, errors.New("invalid IID")
	}
	if vpcIID.NameId == "" {
		result, err := virtualNetworksClient.List(ctx, resourceGroup)
		if err != nil {
			return nil, err
		}
		for _, vpc := range result.Values() {
			if *vpc.ID == vpcIID.SystemId {
				return &vpc, nil
			}
		}
		return nil, errors.New("not found SecurityGroup")
	} else {
		vpc, err := virtualNetworksClient.Get(ctx, resourceGroup, vpcIID.NameId, "")
		return &vpc, err
	}
}
