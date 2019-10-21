package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AzureVNetworkHandler struct {
	Region       idrv.RegionInfo
	Ctx          context.Context
	Client       *network.VirtualNetworksClient
	SubnetClient *network.SubnetsClient
}

func setterVNet(network network.Subnet) *irs.VNetworkInfo {
	vNetInfo := &irs.VNetworkInfo{
		Id:            *network.ID,
		Name:          *network.Name,
		AddressPrefix: *network.AddressPrefix,
		Status:        *network.ProvisioningState,
		KeyValueList:  []irs.KeyValue{{Key: "ResourceGroup", Value: CBResourceGroupName}},
	}

	return vNetInfo
}

func (vNetworkHandler *AzureVNetworkHandler) CreateVNetwork(vNetworkReqInfo irs.VNetworkReqInfo) (irs.VNetworkInfo, error) {
	// 기본 가상 네트워크가 생성되지 않았을 경우 디폴트 네트워크 생성 (CB-VNet)
	vNetwork, _ := vNetworkHandler.Client.Get(vNetworkHandler.Ctx, CBResourceGroupName, CBVirutalNetworkName, "")
	if vNetwork.ID == nil {
		createOpts := network.VirtualNetwork{
			Name: to.StringPtr(CBVirutalNetworkName),
			VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
				AddressSpace: &network.AddressSpace{
					AddressPrefixes: &[]string{CBVnetDefaultCidr},
				},
			},
			Location: &vNetworkHandler.Region.Region,
		}

		future, err := vNetworkHandler.Client.CreateOrUpdate(vNetworkHandler.Ctx, CBResourceGroupName, CBVirutalNetworkName, createOpts)
		if err != nil {
			return irs.VNetworkInfo{}, err
		}
		err = future.WaitForCompletionRef(vNetworkHandler.Ctx, vNetworkHandler.Client.Client)
		if err != nil {
			return irs.VNetworkInfo{}, err
		}
	}

	// Check Subnet Exists
	subnet, _ := vNetworkHandler.SubnetClient.Get(vNetworkHandler.Ctx, CBResourceGroupName, CBVirutalNetworkName, vNetworkReqInfo.Name, "")
	if subnet.ID != nil {
		errMsg := fmt.Sprintf("Virtual Network with name %s already exist", vNetworkReqInfo.Name)
		createErr := errors.New(errMsg)
		return irs.VNetworkInfo{}, createErr
	}

	// 서브넷 CIDR 할당
	list, err := vNetworkHandler.ListVNetwork()
	if err != nil {
		return irs.VNetworkInfo{}, err
	}
	subnetCIDR, err := CreateSubnetCIDR(list)

	// 서브넷 생성
	createOpts := network.Subnet{
		Name: to.StringPtr(vNetworkReqInfo.Name),
		SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
			AddressPrefix: subnetCIDR,
		},
	}

	future, err := vNetworkHandler.SubnetClient.CreateOrUpdate(vNetworkHandler.Ctx, CBResourceGroupName, CBVirutalNetworkName, vNetworkReqInfo.Name, createOpts)
	if err != nil {
		return irs.VNetworkInfo{}, err
	}
	err = future.WaitForCompletionRef(vNetworkHandler.Ctx, vNetworkHandler.Client.Client)
	if err != nil {
		return irs.VNetworkInfo{}, err
	}

	// 생성된 VNetwork 정보 리턴
	vNetworkInfo, err := vNetworkHandler.GetVNetwork(vNetworkReqInfo.Name)
	if err != nil {
		return irs.VNetworkInfo{}, err
	}
	return vNetworkInfo, nil
}

func (vNetworkHandler *AzureVNetworkHandler) ListVNetwork() ([]*irs.VNetworkInfo, error) {
	//vNetworkList, err := vNetworkHandler.Client.List(vNetworkHandler.Ctx, CBResourceGroupName)
	vNetworkList, err := vNetworkHandler.SubnetClient.List(vNetworkHandler.Ctx, CBResourceGroupName, CBVirutalNetworkName)
	if err != nil {
		return nil, err
	}

	var vNetList []*irs.VNetworkInfo
	for _, vNetwork := range vNetworkList.Values() {
		vNetInfo := setterVNet(vNetwork)
		vNetList = append(vNetList, vNetInfo)
	}
	//spew.Dump(vNetList)
	return vNetList, nil
}

func (vNetworkHandler *AzureVNetworkHandler) GetVNetwork(vNetworkID string) (irs.VNetworkInfo, error) {
	//vNetwork, err := vNetworkHandler.Client.Get(vNetworkHandler.Ctx, CBResourceGroupName, vNetworkID, "")
	vNetwork, err := vNetworkHandler.SubnetClient.Get(vNetworkHandler.Ctx, CBResourceGroupName, CBVirutalNetworkName, vNetworkID, "")
	if err != nil {
		return irs.VNetworkInfo{}, err
	}

	vNetInfo := setterVNet(vNetwork)
	//spew.Dump(vNetInfo)
	return *vNetInfo, nil
}

func (vNetworkHandler *AzureVNetworkHandler) DeleteVNetwork(vNetworkID string) (bool, error) {
	//future, err := vNetworkHandler.Client.Delete(vNetworkHandler.Ctx, CBResourceGroupName, vNetworkID)
	future, err := vNetworkHandler.SubnetClient.Delete(vNetworkHandler.Ctx, CBResourceGroupName, CBVirutalNetworkName, vNetworkID)
	if err != nil {
		return false, err
	}
	err = future.WaitForCompletionRef(vNetworkHandler.Ctx, vNetworkHandler.Client.Client)
	if err != nil {
		return false, err
	}
	return true, nil
}
