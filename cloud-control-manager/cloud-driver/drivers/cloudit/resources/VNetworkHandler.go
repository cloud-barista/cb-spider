package resources

import (
	"errors"
	"fmt"
	//cblog "github.com/cloud-barista/cb-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/dna/subnet"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	//"github.com/sirupsen/logrus"
	"strconv"
)

type ClouditVNetworkHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
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
	// @TODO: Subnet 생성 요청 파라미터 정의 필요
	type VNetworkReqInfo struct {
		Name       string `json:"name" required:"true"`
		Addr       string `json:"addr" required:"true"`
		Prefix     string `json:"prefix" required:"true"`
		Gateway    string `json:"gateway" required:"false"`
		Protection int    `json:"protection" required:"false"`
	}
	reqInfo := VNetworkReqInfo{
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
		for i, vNet := range *vNetList {
			cblogger.Info("[" + strconv.Itoa(i) + "]")
			spew.Dump(vNet)
		}
		return nil, nil
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
