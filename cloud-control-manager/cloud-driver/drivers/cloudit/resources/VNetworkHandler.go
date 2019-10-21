package resources

import (
	"errors"
	"fmt"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/dna/subnet"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type ClouditVNetworkHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func setterVNet(vNet subnet.SubnetInfo) *irs.VNetworkInfo {
	vNetInfo := &irs.VNetworkInfo{
		Id:            vNet.ID,
		Name:          vNet.Name,
		AddressPrefix: vNet.Prefix,
		Status:        vNet.State,
	}
	return vNetInfo
}

func (vNetworkHandler *ClouditVNetworkHandler) CreateVNetwork(vNetReqInfo irs.VNetworkReqInfo) (irs.VNetworkInfo, error) {
	vNetworkHandler.Client.TokenID = vNetworkHandler.CredentialInfo.AuthToken
	authHeader := vNetworkHandler.Client.AuthenticatedHeaders()

	var creatableSubnet subnet.SubnetInfo

	// 1. 사용 가능한 Subnet 목록 가져오기
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	if creatableSubnetList, err := subnet.ListCreatableSubnet(vNetworkHandler.Client, &requestOpts); err != nil {
		return irs.VNetworkInfo{}, err
	} else {
		if len(*creatableSubnetList) == 0 {
			allocateErr := errors.New(fmt.Sprintf("There is no PublicIPs to allocate"))
			return irs.VNetworkInfo{}, allocateErr
		} else {
			creatableSubnet = (*creatableSubnetList)[0]
		}
	}

	// 2. Subnet 생성
	reqInfo := subnet.VNetworkReqInfo{
		Name:   vNetReqInfo.Name,
		Addr:   creatableSubnet.Addr,
		Prefix: creatableSubnet.Prefix,
	}

	createOpts := client.RequestOpts{
		JSONBody:    reqInfo,
		MoreHeaders: authHeader,
	}

	if subnet, err := subnet.Create(vNetworkHandler.Client, &createOpts); err != nil {
		return irs.VNetworkInfo{}, err
	} else {
		spew.Dump(subnet)
		return irs.VNetworkInfo{Id: subnet.Addr, Name: subnet.Name}, nil
	}
}

func (vNetworkHandler *ClouditVNetworkHandler) ListVNetwork() ([]*irs.VNetworkInfo, error) {
	vNetworkHandler.Client.TokenID = vNetworkHandler.CredentialInfo.AuthToken
	authHeader := vNetworkHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if vNetList, err := subnet.List(vNetworkHandler.Client, &requestOpts); err != nil {
		return nil, err
	} else {
		var resultList []*irs.VNetworkInfo

		for _, vNet := range *vNetList {
			vNetInfo := setterVNet(vNet)
			resultList = append(resultList, vNetInfo)
		}
		return resultList, nil
	}
}

func (vNetworkHandler *ClouditVNetworkHandler) GetVNetwork(vNetworkID string) (irs.VNetworkInfo, error) {
	vNetworkHandler.Client.TokenID = vNetworkHandler.CredentialInfo.AuthToken
	authHeader := vNetworkHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if vNetwork, err := subnet.Get(vNetworkHandler.Client, vNetworkID, &requestOpts); err != nil {
		return irs.VNetworkInfo{}, err
	} else {
		spew.Dump(vNetwork)
		return irs.VNetworkInfo{Id: vNetwork.ID, Name: vNetwork.Name}, nil
	}
}

func (vNetworkHandler *ClouditVNetworkHandler) DeleteVNetwork(vNetworkID string) (bool, error) {
	vNetworkHandler.Client.TokenID = vNetworkHandler.CredentialInfo.AuthToken
	authHeader := vNetworkHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if err := subnet.Delete(vNetworkHandler.Client, vNetworkID, &requestOpts); err != nil {
		return false, err
	} else {
		return true, nil
	}
}
