package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"
)

type AzurePublicIPHandler struct {
	Region idrv.RegionInfo
	Ctx    context.Context
	Client *network.PublicIPAddressesClient
}

func setterIP(address network.PublicIPAddress) *irs.PublicIPInfo {
	publicIP := &irs.PublicIPInfo{
		Name:         *address.Name,
		PublicIP:     *address.IPAddress,
		Status:       *address.ProvisioningState,
		KeyValueList: []irs.KeyValue{{Key: "ResourceGroup", Value: CB_GROUP}},
	}

	return publicIP
}

func (publicIpHandler *AzurePublicIPHandler) CreatePublicIP(publicIPReqInfo irs.PublicIPReqInfo) (irs.PublicIPInfo, error) {
	// Check PublicIP Exists
	publicIP, err := publicIpHandler.Client.Get(publicIpHandler.Ctx, CB_GROUP, publicIPReqInfo.Name, "")
	if publicIP.ID != nil {
		errMsg := fmt.Sprintf("Public IP with name %s already exist", publicIPReqInfo.Name)
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

	future, err := publicIpHandler.Client.CreateOrUpdate(publicIpHandler.Ctx, CB_GROUP, publicIPReqInfo.Name, createOpts)
	if err != nil {
		return irs.PublicIPInfo{}, err
	}
	err = future.WaitForCompletionRef(publicIpHandler.Ctx, publicIpHandler.Client.Client)
	if err != nil {
		return irs.PublicIPInfo{}, err
	}

	// 생성된 PublicIP 정보 리턴
	publicIPInfo, err := publicIpHandler.GetPublicIP(publicIPReqInfo.Name)
	if err != nil {
		return irs.PublicIPInfo{}, err
	}
	return publicIPInfo, nil
}

func (publicIpHandler *AzurePublicIPHandler) ListPublicIP() ([]*irs.PublicIPInfo, error) {
	result, err := publicIpHandler.Client.List(publicIpHandler.Ctx, CB_GROUP)
	if err != nil {
		return nil, err
	}

	var publicIPList []*irs.PublicIPInfo
	for _, publicIP := range result.Values() {
		publicIPInfo := setterIP(publicIP)
		publicIPList = append(publicIPList, publicIPInfo)
	}
	//spew.Dump(publicIPList)
	return publicIPList, nil
}

func (publicIpHandler *AzurePublicIPHandler) GetPublicIP(publicIPID string) (irs.PublicIPInfo, error) {
	publicIP, err := publicIpHandler.Client.Get(publicIpHandler.Ctx, CB_GROUP, publicIPID, "")
	if err != nil {
		return irs.PublicIPInfo{}, err
	}

	publicIPInfo := setterIP(publicIP)
	//spew.Dump(publicIPInfo)
	return *publicIPInfo, nil
}

func (publicIpHandler *AzurePublicIPHandler) DeletePublicIP(publicIPID string) (bool, error) {
	future, err := publicIpHandler.Client.Delete(publicIpHandler.Ctx, CB_GROUP, publicIPID)
	if err != nil {
		return false, err
	}
	err = future.WaitForCompletionRef(publicIpHandler.Ctx, publicIpHandler.Client.Client)
	if err != nil {
		return false, err
	}
	return true, nil
}
