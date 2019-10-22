package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	compute "google.golang.org/api/compute/v1"

	idrv "github.com/cloud-barista/poc-cb-spider/cloud-driver/interfaces"
	irs "github.com/cloud-barista/poc-cb-spider/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type GCPVNetworkHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

// @TODO: VNetworkInfo 리소스 프로퍼티 정의 필요
type VNetworkInfo struct {
	Id              string
	Name            string
	AddressPrefixes []string
	Subnets         []SubnetInfo
	Location        string
}

type SubnetInfo struct {
	Id            string
	Name          string
	AddressPrefix string
}

func (vNetInfo *VNetworkInfo) setter(network network.VirtualNetwork) *VNetworkInfo {
	vNetInfo.Id = *network.ID
	vNetInfo.Name = *network.Name
	vNetInfo.AddressPrefixes = *network.AddressSpace.AddressPrefixes
	var subnetArr []SubnetInfo
	for _, subnet := range *network.Subnets {
		subnetInfo := SubnetInfo{
			Id:            *subnet.ID,
			Name:          *subnet.Name,
			AddressPrefix: *subnet.AddressPrefix,
		}
		subnetArr = append(subnetArr, subnetInfo)
	}
	vNetInfo.Subnets = subnetArr

	vNetInfo.Location = *network.Location

	return vNetInfo
}

func (vNetworkHandler *GCPVNetworkHandler) CreateVNetwork(vNetworkReqInfo irs.VNetworkReqInfo) (irs.VNetworkInfo, error) {

	// @TODO: VNicInfo 생성 요청 파라미터 정의 필요
	type VNetworkReqInfo struct {
		Name            string
		AddressPrefixes []string
		Subnets         *[]SubnetInfo
	}

	vNicIdArr := strings.Split(vNetworkReqInfo.Id, ":")

	reqInfo := VNetworkReqInfo{
		Name:            vNicIdArr[1],
		AddressPrefixes: []string{"130.0.0.0/8"},
		Subnets:         &[]SubnetInfo{
			/*{
				Name: "test-subnet1",
				AddressPrefix: "10.0.0.0/16",
			},*/
		},
	}

	var subnetArr []network.Subnet
	for _, subnet := range *reqInfo.Subnets {
		subnetInfo := network.Subnet{
			Name: &subnet.Name,
			SubnetPropertiesFormat: &network.SubnetPropertiesFormat{
				AddressPrefix: &subnet.AddressPrefix,
			},
		}
		subnetArr = append(subnetArr, subnetInfo)
	}

	// Check vNetwork Exists
	vNetwork, err := vNetworkHandler.Client.Get(vNetworkHandler.Ctx, vNicIdArr[0], vNicIdArr[1], "")
	if vNetwork.ID != nil {
		errMsg := fmt.Sprintf("Virtual Network with name %s already exist", vNicIdArr[1])
		createErr := errors.New(errMsg)
		return irs.VNetworkInfo{}, createErr
	}

	createOpts := network.VirtualNetwork{
		Name: &reqInfo.Name,
		VirtualNetworkPropertiesFormat: &network.VirtualNetworkPropertiesFormat{
			AddressSpace: &network.AddressSpace{
				AddressPrefixes: &reqInfo.AddressPrefixes,
			},
			Subnets: &subnetArr,
		},
		Location: &vNetworkHandler.Region.Region,
	}

	future, err := vNetworkHandler.Client.CreateOrUpdate(vNetworkHandler.Ctx, vNicIdArr[0], vNicIdArr[1], createOpts)
	if err != nil {
		return irs.VNetworkInfo{}, err
	}
	err = future.WaitForCompletionRef(vNetworkHandler.Ctx, vNetworkHandler.Client.Client)
	if err != nil {
		return irs.VNetworkInfo{}, err
	}

	return irs.VNetworkInfo{}, nil
}

func (vNetworkHandler *GCPVNetworkHandler) ListVNetwork() ([]*irs.VNetworkInfo, error) {
	//vNetworkList, err := vNetworkHandler.Client.ListAll(vNetworkHandler.Ctx)
	vNetworkList, err := vNetworkHandler.Client.List(vNetworkHandler.Ctx, vNetworkHandler.Region.ResourceGroup)
	if err != nil {
		return nil, err
	}

	var vNetList []*VNetworkInfo
	for _, vNetwork := range vNetworkList.Values() {
		vNetInfo := new(VNetworkInfo).setter(vNetwork)
		vNetList = append(vNetList, vNetInfo)
	}

	spew.Dump(vNetList)
	return nil, nil
}

func (vNetworkHandler *GCPVNetworkHandler) GetVNetwork(vNetworkID string) (irs.VNetworkInfo, error) {
	vNetworkIdArr := strings.Split(vNetworkID, ":")
	vNetwork, err := vNetworkHandler.Client.Get(vNetworkHandler.Ctx, vNetworkIdArr[0], vNetworkIdArr[1], "")
	if err != nil {
		return irs.VNetworkInfo{}, err
	}

	vNetInfo := new(VNetworkInfo).setter(vNetwork)

	spew.Dump(vNetInfo)
	return irs.VNetworkInfo{}, nil
}

func (vNetworkHandler *GCPVNetworkHandler) DeleteVNetwork(vNetworkID string) (bool, error) {
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
