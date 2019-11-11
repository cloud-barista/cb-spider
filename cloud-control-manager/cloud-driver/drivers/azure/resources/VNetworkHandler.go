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

func (vNetworkHandler *AzureVNetworkHandler) setterVNet(network network.Subnet) *irs.VNetworkInfo {
	vNetInfo := &irs.VNetworkInfo{
		Id:            *network.ID,
		Name:          *network.Name,
		AddressPrefix: *network.AddressPrefix,
		Status:        *network.ProvisioningState,
		KeyValueList:  []irs.KeyValue{{Key: "ResourceGroup", Value: vNetworkHandler.Region.ResourceGroup}},
	}

	return vNetInfo
}

func (vNetworkHandler *AzureVNetworkHandler) CreateVNetwork(vNetworkReqInfo irs.VNetworkReqInfo) (irs.VNetworkInfo, error) {
	// Check VNet Exists
	// 기본 가상 네트워크가 생성되지 않았을 경우 디폴트 네트워크 생성 (CB-VNet)
	vNetwork, _ := vNetworkHandler.Client.Get(vNetworkHandler.Ctx, vNetworkHandler.Region.ResourceGroup, CBVirutalNetworkName, "")
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

		future, err := vNetworkHandler.Client.CreateOrUpdate(vNetworkHandler.Ctx, vNetworkHandler.Region.ResourceGroup, CBVirutalNetworkName, createOpts)
		if err != nil {
			return irs.VNetworkInfo{}, err
		}
		err = future.WaitForCompletionRef(vNetworkHandler.Ctx, vNetworkHandler.Client.Client)
		if err != nil {
			return irs.VNetworkInfo{}, err
		}
	}

	// Check Subnet Exists
	subnet, _ := vNetworkHandler.SubnetClient.Get(vNetworkHandler.Ctx, vNetworkHandler.Region.ResourceGroup, CBVirutalNetworkName, vNetworkReqInfo.Name, "")
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

	future, err := vNetworkHandler.SubnetClient.CreateOrUpdate(vNetworkHandler.Ctx, vNetworkHandler.Region.ResourceGroup, CBVirutalNetworkName, vNetworkReqInfo.Name, createOpts)
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
	// Check VNet Exists
	vNetwork, _ := vNetworkHandler.Client.Get(vNetworkHandler.Ctx, vNetworkHandler.Region.ResourceGroup, CBVirutalNetworkName, "")
	if vNetwork.ID == nil {
		return nil, nil
	}

	vNetworkList, err := vNetworkHandler.SubnetClient.List(vNetworkHandler.Ctx, vNetworkHandler.Region.ResourceGroup, CBVirutalNetworkName)
	if err != nil {
		return nil, err
	}

	var vNetList []*irs.VNetworkInfo
	for _, vNetwork := range vNetworkList.Values() {
		vNetInfo := vNetworkHandler.setterVNet(vNetwork)
		vNetList = append(vNetList, vNetInfo)
	}
	return vNetList, nil
}

func (vNetworkHandler *AzureVNetworkHandler) GetVNetwork(vNetworkID string) (irs.VNetworkInfo, error) {
	// Check VNet Exists
	vNetwork, err := vNetworkHandler.Client.Get(vNetworkHandler.Ctx, vNetworkHandler.Region.ResourceGroup, CBVirutalNetworkName, "")
	if vNetwork.ID == nil {
		return irs.VNetworkInfo{}, err
	}

	subnet, err := vNetworkHandler.SubnetClient.Get(vNetworkHandler.Ctx, vNetworkHandler.Region.ResourceGroup, CBVirutalNetworkName, vNetworkID, "")
	if err != nil {
		return irs.VNetworkInfo{}, err
	}

	vNetInfo := vNetworkHandler.setterVNet(subnet)
	return *vNetInfo, nil
}

func (vNetworkHandler *AzureVNetworkHandler) DeleteVNetwork(vNetworkID string) (bool, error) {
	future, err := vNetworkHandler.SubnetClient.Delete(vNetworkHandler.Ctx, vNetworkHandler.Region.ResourceGroup, CBVirutalNetworkName, vNetworkID)
	if err != nil {
		return false, err
	}
	err = future.WaitForCompletionRef(vNetworkHandler.Ctx, vNetworkHandler.Client.Client)
	if err != nil {
		return false, err
	}

	// 서브넷이 없을 경우 해당 VNetwork도 함께 삭제 처리
	vNetworkList, err := vNetworkHandler.ListVNetwork()
	if err != nil {
		return false, err
	}

	if len(vNetworkList) == 0 {
		future, err := vNetworkHandler.Client.Delete(vNetworkHandler.Ctx, vNetworkHandler.Region.ResourceGroup, CBVirutalNetworkName)
		if err != nil {
			return false, err
		}
		err = future.WaitForCompletionRef(vNetworkHandler.Ctx, vNetworkHandler.Client.Client)
		if err != nil {
			return false, err
		}
	}

	return true, nil
}
