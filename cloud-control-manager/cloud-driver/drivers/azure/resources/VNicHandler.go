package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"strings"
)

type AzureVNicHandler struct {
	Region       idrv.RegionInfo
	Ctx          context.Context
	NicClient    *network.InterfacesClient
	SubnetClient *network.SubnetsClient
}

// @TODO: VNicInfo 리소스 프로퍼티 정의 필요
type VNicInfo struct {
	Id            string
	Name          string
	Location      string
	Primary       bool
	MacAddress    string
	IP            []VNicIPConfig
	SecurityGroup string
}

type VNicIPConfig struct {
	Primary                   bool
	PrivateIPAddress          string
	PrivateIPAddressVersion   string
	PrivateIPAllocationMethod string
	PublicIP                  string
	PublicIPAddressVersion    string
	PublicIPAllocationMethod  string
}

func (nic *VNicInfo) setter(ni network.Interface) *VNicInfo {
	nic.Id = *ni.ID
	nic.Name = *ni.Name
	nic.Location = *ni.Location

	if ni.NetworkSecurityGroup != nil {
		nic.SecurityGroup = *ni.NetworkSecurityGroup.ID
	}

	var IPArr []VNicIPConfig
	for _, ip := range *ni.IPConfigurations {
		ipConfigInfo := VNicIPConfig{
			Primary:                   *ip.Primary,
			PrivateIPAddress:          *ip.PrivateIPAddress,
			PrivateIPAddressVersion:   fmt.Sprint(ip.PrivateIPAddressVersion),
			PrivateIPAllocationMethod: fmt.Sprint(ip.PrivateIPAllocationMethod),
		}

		if ip.PublicIPAddress != nil {
			ipConfigInfo.PublicIP = *ip.PublicIPAddress.ID
		}

		IPArr = append(IPArr, ipConfigInfo)
	}
	nic.IP = IPArr

	return nic
}

func (vNicHandler *AzureVNicHandler) CreateVNic(vNicReqInfo irs.VNicReqInfo) (irs.VNicInfo, error) {

	// @TODO: VNicInfo 생성 요청 파라미터 정의 필요
	type VNicIPReqInfo struct {
		Name                      string
		PrivateIPAllocationMethod string
		PublicIPId                string
	}
	type VNicReqInfo struct {
		Id              string
		VNetworkName    string
		SubnetName      string
		SecurityGroupId string
		IP              []VNicIPReqInfo
	}

	reqInfo := VNicReqInfo{
		//VNetworkName: "mcb-test-vnet",
		// edited by powerkim for test, 2019.08.13
		VNetworkName: "cb-vnet",
		SubnetName:   "default",
		IP: []VNicIPReqInfo{
			{
				Name:                      "ipConfig1",
				PrivateIPAllocationMethod: "Dynamic",
				//PublicIPId:                "/subscriptions/cb592624-b77b-4a8f-bb13-0e5a48cae40f/resourceGroups/inno-platform1-rsrc-grup/providers/Microsoft.Network/publicIPAddresses/mcb-test-publicip", // @todo
				//changed by powerkim for test, 2019.09.02
				PublicIPId: "/subscriptions/f1548292-2be3-4acd-84a4-6df079160846/resourceGroups/cb-resource-group/providers/Microsoft.Network/publicIPAddresses/powerkim-test1-ip",
			},
		},
		//SecurityGroupId: "/subscriptions/cb592624-b77b-4a8f-bb13-0e5a48cae40f/resourceGroups/inno-platform1-rsrc-grup/providers/Microsoft.Network/networkSecurityGroups/mcb-test-sg", // @todo
		//edited by powerkim for test, 2019.08.13
		//SecurityGroupId: "cb-security-group", // changed by powerkim, 2019.09.02
		SecurityGroupId: "/subscriptions/f1548292-2be3-4acd-84a4-6df079160846/resourceGroups/cb-resource-group/providers/Microsoft.Network/networkSecurityGroups/cb-security-group",
	}

	vNicIdArr := strings.Split(vNicReqInfo.Id, ":")

	// Check vNic Exists
	vNic, err := vNicHandler.NicClient.Get(vNicHandler.Ctx, vNicIdArr[0], vNicIdArr[1], "")
	if vNic.ID != nil {
		errMsg := fmt.Sprintf("Virtual Network Interface with name %s already exist", vNicIdArr[1])
		createErr := errors.New(errMsg)
		return irs.VNicInfo{}, createErr
	}

	subnet, err := vNicHandler.getSubnet(vNicIdArr[0], reqInfo.VNetworkName, reqInfo.SubnetName)

	var ipConfigArr []network.InterfaceIPConfiguration
	for _, ipReqInfo := range reqInfo.IP {
		ipConfig := network.InterfaceIPConfiguration{
			Name: &ipReqInfo.Name,
			InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
				Subnet:                    &subnet,
				PrivateIPAllocationMethod: network.IPAllocationMethod(ipReqInfo.PrivateIPAllocationMethod),
				PublicIPAddress: &network.PublicIPAddress{
					ID: &ipReqInfo.PublicIPId,
				},
			},
		}
		ipConfigArr = append(ipConfigArr, ipConfig)
	}

	createOpts := network.Interface{
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &ipConfigArr,
			NetworkSecurityGroup: &network.SecurityGroup{
				ID: &reqInfo.SecurityGroupId,
			},
		},
		Location: &vNicHandler.Region.Region,
		//NetworkSecurityGroup:
	}

	future, err := vNicHandler.NicClient.CreateOrUpdate(vNicHandler.Ctx, vNicIdArr[0], vNicIdArr[1], createOpts)
	if err != nil {
		return irs.VNicInfo{}, err
	}
	err = future.WaitForCompletionRef(vNicHandler.Ctx, vNicHandler.NicClient.Client)
	if err != nil {
		return irs.VNicInfo{}, err
	}

	return irs.VNicInfo{}, nil
}

func (vNicHandler *AzureVNicHandler) ListVNic() ([]*irs.VNicInfo, error) {
	//result, err := vNicHandler.NicClient.ListAll(vNicHandler.Ctx)
	result, err := vNicHandler.NicClient.List(vNicHandler.Ctx, vNicHandler.Region.ResourceGroup)
	if err != nil {
		return nil, err
	}

	var vNicList []*VNicInfo
	for _, vNic := range result.Values() {
		vNicInfo := new(VNicInfo).setter(vNic)
		vNicList = append(vNicList, vNicInfo)
	}

	spew.Dump(vNicList)
	return nil, nil
}

func (vNicHandler *AzureVNicHandler) GetVNic(vNicID string) (irs.VNicInfo, error) {
	vNicIDArr := strings.Split(vNicID, ":")
	vNic, err := vNicHandler.NicClient.Get(vNicHandler.Ctx, vNicIDArr[0], vNicIDArr[1], "")
	if err != nil {
		return irs.VNicInfo{}, err
	}

	vNicInfo := new(VNicInfo).setter(vNic)

	spew.Dump(vNicInfo)
	return irs.VNicInfo{}, nil
}

func (vNicHandler *AzureVNicHandler) DeleteVNic(vNicID string) (bool, error) {
	vNicIDArr := strings.Split(vNicID, ":")
	future, err := vNicHandler.NicClient.Delete(vNicHandler.Ctx, vNicIDArr[0], vNicIDArr[1])
	if err != nil {
		return false, err
	}
	err = future.WaitForCompletionRef(vNicHandler.Ctx, vNicHandler.NicClient.Client)
	if err != nil {
		return false, err
	}
	return true, err
}

func (vNicHandler *AzureVNicHandler) getSubnet(rsgName string, vNetName string, subnetName string) (network.Subnet, error) {
	return vNicHandler.SubnetClient.Get(vNicHandler.Ctx, rsgName, vNetName, subnetName, "")
}
