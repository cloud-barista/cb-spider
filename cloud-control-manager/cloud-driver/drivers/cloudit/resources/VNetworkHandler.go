package resources

import (
	"errors"
	"fmt"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/dna/subnet"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"strings"
)

type ClouditVNetworkHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func setterVNet(vNet subnet.SubnetInfo) *irs.VNetworkInfo {
	addrPrefix := vNet.Addr + "/" + vNet.Prefix
	vNetInfo := &irs.VNetworkInfo{
		Id:            vNet.Addr, // Subnet 주소 정보를 Id로 사용
		Name:          vNet.Name,
		AddressPrefix: addrPrefix,
		Status:        vNet.State,
	}
	return vNetInfo
}

func (vNetworkHandler *ClouditVNetworkHandler) CreateVNetwork(vNetReqInfo irs.VNetworkReqInfo) (irs.VNetworkInfo, error) {
	// 서브넷 이름 중복 체크
	vnetwork, _ := vNetworkHandler.getVNetworkByName(vNetReqInfo.Name)
	if vnetwork != nil {
		errMsg := fmt.Sprintf("VirtualNetwork with name %s already exist", vNetReqInfo.Name)
		createErr := errors.New(errMsg)
		return irs.VNetworkInfo{}, createErr
	}

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

	if vNetwork, err := subnet.Create(vNetworkHandler.Client, &createOpts); err != nil {
		return irs.VNetworkInfo{}, err
	} else {
		vNetInfo := setterVNet(*vNetwork)
		return *vNetInfo, nil
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

func (vNetworkHandler *ClouditVNetworkHandler) GetVNetwork(vNetworkNameID string) (irs.VNetworkInfo, error) {
	// 이름 기준 서브넷 조회
	subnetInfo, err := vNetworkHandler.getVNetworkByName(vNetworkNameID)
	if err != nil {
		cblogger.Error(err)
		return irs.VNetworkInfo{}, err
	}

	vNetInfo := setterVNet(*subnetInfo)
	return *vNetInfo, nil
}

func (vNetworkHandler *ClouditVNetworkHandler) DeleteVNetwork(vNetworkNameID string) (bool, error) {
	// 이름 기준 서브넷 조회
	subnetInfo, err := vNetworkHandler.getVNetworkByName(vNetworkNameID)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	vNetworkHandler.Client.TokenID = vNetworkHandler.CredentialInfo.AuthToken
	authHeader := vNetworkHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if err := subnet.Delete(vNetworkHandler.Client, subnetInfo.Addr, &requestOpts); err != nil {
		//panic(err)
		return false, err
	} else {
		return true, nil
	}
}

func (vNetworkHandler *ClouditVNetworkHandler) getVNetworkByName(subnetName string) (*subnet.SubnetInfo, error) {
	var subnetInfo *subnet.SubnetInfo

	vNetworkHandler.Client.TokenID = vNetworkHandler.CredentialInfo.AuthToken
	authHeader := vNetworkHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	subnetList, err := subnet.List(vNetworkHandler.Client, &requestOpts)
	if err != nil {
		return nil, err
	}

	for _, s := range *subnetList {
		if strings.EqualFold(s.Name, subnetName) {
			subnetInfo = &s
			break
		}
	}

	if subnetInfo == nil {
		err := errors.New(fmt.Sprintf("failed to find virtual network with name %s", subnetName))
		return nil, err
	}
	return subnetInfo, nil
}
