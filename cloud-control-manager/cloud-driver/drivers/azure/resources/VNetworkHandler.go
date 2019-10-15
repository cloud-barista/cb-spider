package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"
	"github.com/davecgh/go-spew/spew"
	"strings"
)

type AzureVNetworkHandler struct {
	Region idrv.RegionInfo
	Ctx    context.Context
	Client *network.VirtualNetworksClient
}

// @TODO: VNetworkInfo 리소스 프로퍼티 정의 필요
//type VNetworkInfo struct {
//	Id              string
//	Name            string
//	AddressPrefixes []string
//	Subnets         []SubnetInfo
//	Location        string
//}
//
//type SubnetInfo struct {
//	Id            string
//	Name          string
//	AddressPrefix string
//}

func setterVNet(network network.VirtualNetwork) *irs.VNetworkInfo {
	vNetInfo := &irs.VNetworkInfo{
		Id:   *network.ID,
		Name: *network.Name,
		//AddressPrefix: &network.AddressSpace.AddressPrefixes,
		Status: *network.ProvisioningState,
	}

	return vNetInfo
}

func (vNetworkHandler *AzureVNetworkHandler) CreateVNetwork(vNetworkReqInfo irs.VNetworkReqInfo) (irs.VNetworkInfo, error) {

	//reqInfo := irs.VNetworkReqInfo{
	//	Name:            vNicIdArr[1],
	//	AddressPrefixes: []string{"130.0.0.0/8"},
	//	Subnets: &[]SubnetInfo{
	//		{
	//			Name:          "default",
	//			AddressPrefix: "130.1.0.0/16",
	//		},
	//	},
	//}

	//var subnetArr []network.Subnet
	//	subnetInfo := network.Subnet{
	//		Name: &vNetworkReqInfo.Name,
	//	}
	//	subnetArr = append(subnetArr, subnetInfo)
	vNetworkReqInfo.Name = "inno-platform1-rsrc-grup:Test-mcb-test-vnet"
	vNetworkIdArr := strings.Split(vNetworkReqInfo.Name, ":")

	// Check vNetwork Exists
	vNetwork, _ := vNetworkHandler.Client.Get(vNetworkHandler.Ctx, vNetworkIdArr[0], vNetworkIdArr[1], "")
	if vNetwork.ID != nil {
		errMsg := fmt.Sprintf("Virtual Network with name %s already exist", vNetworkIdArr[1])
		createErr := errors.New(errMsg)
		return irs.VNetworkInfo{}, createErr
	}

	createOpts := network.VirtualNetwork{
		Name: to.StringPtr("default"),
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{
				AddressPrefixes: &[]string{"130.0.0.0/8"},
			},
		},
		Location: &vNetworkHandler.Region.Region,
	}

	future, err := vNetworkHandler.Client.CreateOrUpdate(vNetworkHandler.Ctx, vNetworkIdArr[0], vNetworkIdArr[1], createOpts)
	if err != nil {
		return irs.VNetworkInfo{}, err
	}
	err = future.WaitForCompletionRef(vNetworkHandler.Ctx, vNetworkHandler.Client.Client)
	if err != nil {
		return irs.VNetworkInfo{}, err
	}

	return irs.VNetworkInfo{}, nil
}

func (vNetworkHandler *AzureVNetworkHandler) ListVNetwork() ([]*irs.VNetworkInfo, error) {
	//vNetworkList, err := vNetworkHandler.Client.ListAll(vNetworkHandler.Ctx)
	vNetworkList, err := vNetworkHandler.Client.List(vNetworkHandler.Ctx, vNetworkHandler.Region.ResourceGroup)
	if err != nil {
		return nil, err
	}

	var vNetList []*irs.VNetworkInfo
	for _, vNetwork := range vNetworkList.Values() {
		vNetInfo := setterVNet(vNetwork)
		vNetList = append(vNetList, vNetInfo)
	}

	spew.Dump(vNetList)
	return nil, nil
}

func (vNetworkHandler *AzureVNetworkHandler) GetVNetwork(vNetworkID string) (irs.VNetworkInfo, error) {
	vNetworkIdArr := strings.Split(vNetworkID, ":")
	vNetwork, err := vNetworkHandler.Client.Get(vNetworkHandler.Ctx, vNetworkIdArr[0], vNetworkIdArr[1], "")
	if err != nil {
		return irs.VNetworkInfo{}, err
	}

	vNetInfo := setterVNet(vNetwork)

	spew.Dump(vNetInfo)
	return irs.VNetworkInfo{}, nil
}

func (vNetworkHandler *AzureVNetworkHandler) DeleteVNetwork(vNetworkID string) (bool, error) {
	vNetworkIdArr := strings.Split(vNetworkID, ":")
	future, err := vNetworkHandler.Client.Delete(vNetworkHandler.Ctx, vNetworkIdArr[0], vNetworkIdArr[1])
	if err != nil {
		return false, err
	}
	err = future.WaitForCompletionRef(vNetworkHandler.Ctx, vNetworkHandler.Client.Client)
	if err != nil {
		return false, err
	}
	return true, nil
}
