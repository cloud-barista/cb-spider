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

type AzurePublicIPHandler struct {
	Region idrv.RegionInfo
	Ctx    context.Context
	Client *network.PublicIPAddressesClient
}

func setterIP(address network.PublicIPAddress) *irs.PublicIPInfo {
	publicIP := &irs.PublicIPInfo{
		Name:      *address.Name,
		PublicIP:  *address.IPAddress,
		OwnedVMID: *address.ID,
		//todo: Status(available, unavailable 등) 올바르게 뜬거 맞나 확인, KeyValue도 넣어야하나?
		Status: *address.ProvisioningState,
	}
	return publicIP
}

func (publicIpHandler *AzurePublicIPHandler) CreatePublicIP(publicIPReqInfo irs.PublicIPReqInfo) (irs.PublicIPInfo, error) {

	/*reqInfo := irs.PublicIPReqInfo{
		Name: "basic",
	}*/

	publicIPArr := strings.Split(publicIPReqInfo.Name, ":")

	// Check PublicIP Exists
	publicIP, err := publicIpHandler.Client.Get(publicIpHandler.Ctx, publicIPArr[0], publicIPArr[1], "")
	if publicIP.ID != nil {
		errMsg := fmt.Sprintf("Public IP with name %s already exist", publicIPArr[1])
		createErr := errors.New(errMsg)
		return irs.PublicIPInfo{}, createErr
	}

	createOpts := network.PublicIPAddress{
		Name: to.StringPtr(publicIPReqInfo.Name),
		Sku: &network.PublicIPAddressSku{
			Name: network.PublicIPAddressSkuName("Basic"),
		},
		PublicIPAddressPropertiesFormat: &network.PublicIPAddressPropertiesFormat{
			PublicIPAddressVersion:   network.IPVersion("IPv4"),
			PublicIPAllocationMethod: network.IPAllocationMethod("Static"),
			IdleTimeoutInMinutes:     to.Int32Ptr(4),
		},
		Location: &publicIpHandler.Region.Region,
	}

	future, err := publicIpHandler.Client.CreateOrUpdate(publicIpHandler.Ctx, publicIPArr[0], publicIPArr[1], createOpts)
	if err != nil {
		return irs.PublicIPInfo{}, err
	}
	err = future.WaitForCompletionRef(publicIpHandler.Ctx, publicIpHandler.Client.Client)
	if err != nil {
		return irs.PublicIPInfo{}, err
	}

	// @TODO: 생성된 PublicIP 정보 리턴
	publicIPInfo, err := publicIpHandler.GetPublicIP(publicIPReqInfo.Name)
	if err != nil {
		return irs.PublicIPInfo{}, err
	}
	return publicIPInfo, nil
}

func (publicIpHandler *AzurePublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	//result, err := publicIpHandler.Client.ListAll(publicIpHandler.Ctx)
	result, err := publicIpHandler.Client.List(publicIpHandler.Ctx, publicIpHandler.Region.ResourceGroup)
	if err != nil {
		return nil, err
	}

	var publicIPList []*irs.PublicIPInfo
	for _, publicIP := range result.Values() {
		publicIPInfo := setterIP(publicIP)
		publicIPList = append(publicIPList, publicIPInfo)
	}

	spew.Dump(publicIPList)
	return nil, nil
}

func (publicIpHandler *AzurePublicIPHandler) GetPublicIP(publicIPID string) (irs.PublicIPInfo, error) {
	publicIPArr := strings.Split(publicIPID, ":")
	publicIP, err := publicIpHandler.Client.Get(publicIpHandler.Ctx, publicIPArr[0], publicIPArr[1], "")
	if err != nil {
		return irs.PublicIPInfo{}, err
	}

	publicIPInfo := setterIP(publicIP)

	spew.Dump(publicIPInfo)
	return irs.PublicIPInfo{}, nil
}

func (publicIpHandler *AzurePublicIPHandler) DeletePublicIP(publicIPID string) (bool, error) {
	publicIPArr := strings.Split(publicIPID, ":")
	future, err := publicIpHandler.Client.Delete(publicIpHandler.Ctx, publicIPArr[0], publicIPArr[1])
	if err != nil {
		return false, err
	}
	err = future.WaitForCompletionRef(publicIpHandler.Ctx, publicIpHandler.Client.Client)
	if err != nil {
		return false, err
	}
	return true, nil
}
