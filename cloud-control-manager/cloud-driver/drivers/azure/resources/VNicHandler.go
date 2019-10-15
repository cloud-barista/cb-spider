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

func setterVNic(ni network.Interface) *irs.VNicInfo {
	nic := &irs.VNicInfo{
		Name:     *ni.Name,
		PublicIP: "",
		//MacAdress:        *ni.MacAddress,
		OwnedVMID:        *ni.ID,
		SecurityGroupIds: nil,
		Status:           *ni.ProvisioningState,
	}
	return nic
}

func (vNicHandler *AzureVNicHandler) CreateVNic(vNicReqInfo irs.VNicReqInfo) (irs.VNicInfo, error) {

	securityGroupId := "/subscriptions/cb592624-b77b-4a8f-bb13-0e5a48cae40f/resourceGroups/inno-platform1-rsrc-grup/providers/Microsoft.Network/networkSecurityGroups/inno-test-vm-nsg"
	publicIpId := "/subscriptions/cb592624-b77b-4a8f-bb13-0e5a48cae40f/resourceGroups/inno-platform1-rsrc-grup/providers/Microsoft.Network/publicIPAddresses/mcb-test-publicip"

	// TODO: Test_Resource에서 파라미터로 넘어옴
	vNicReqInfo.VNetName = "inno-platform1-rsrc-grup-vnet"
	vNicReqInfo.Name = "inno-platform1-rsrc-grup:Test-mcb-test-vnic"
	vNicReqInfo.PublicIPid = publicIpId
	vNicReqInfo.SecurityGroupIds = []string{securityGroupId}

	vNicIdArr := strings.Split(vNicReqInfo.Name, ":")

	// Check vNic Exists
	//vNic, err := vNicHandler.NicClient.Get(vNicHandler.Ctx, vNicIdArr[0], vNicIdArr[1], "")
	vNic, _ := vNicHandler.NicClient.Get(vNicHandler.Ctx, vNicIdArr[0], vNicIdArr[1], "")
	if vNic.ID != nil {
		errMsg := fmt.Sprintf("Virtual Network Interface with name %s already exist", vNicIdArr[1])
		createErr := errors.New(errMsg)
		return irs.VNicInfo{}, createErr
	}

	subnet, err := vNicHandler.getSubnet(vNicIdArr[0], vNicReqInfo.VNetName, "default")
	// TODO: 추후 VNet 생성 API 기준 테스트
	//subnet, err := vNicHandler.getSubnet(vNicIdArr[0], "CB-VNet", vNicReqInfo.VNetName)

	// TODO: PublicIP Id 값 등록 후 테스트
	var ipConfigArr []network.InterfaceIPConfiguration
	ipConfig := network.InterfaceIPConfiguration{
		Name: to.StringPtr("ipConfig1"),
		InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
			Subnet:                    &subnet,
			PrivateIPAllocationMethod: "Dynamic",
			PublicIPAddress: &network.PublicIPAddress{
				ID: &vNicReqInfo.PublicIPid,
			},
		},
	}
	ipConfigArr = append(ipConfigArr, ipConfig)

	createOpts := network.Interface{
		InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
			IPConfigurations: &ipConfigArr,
			NetworkSecurityGroup: &network.SecurityGroup{
				ID: &vNicReqInfo.SecurityGroupIds[0],
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

	var vNicList []*irs.VNicInfo
	for _, vNic := range result.Values() {
		vNicInfo := setterVNic(vNic)
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

	vNicInfo := setterVNic(vNic)

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
