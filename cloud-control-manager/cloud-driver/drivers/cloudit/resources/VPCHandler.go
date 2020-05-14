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

const (
	defaultVPCName    = "Default-VPC"
	defaultVPCCIDR    = "10.0.0.0/16"
	defaultSubnetName = "Default Network"
)

type ClouditVPCHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func (vpcHandler *ClouditVPCHandler) setterVPC(subnets []subnet.SubnetInfo) *irs.VPCInfo {
	// VPC 정보 맵핑
	vpcInfo := irs.VPCInfo{
		IId: irs.IID{
			NameId:   defaultVPCName,
			SystemId: defaultVPCName,
		},
		IPv4_CIDR: defaultVPCCIDR,
	}
	// 서브넷 정보 조회
	subnetInfoList := make([]irs.SubnetInfo, len(subnets))
	for i, s := range subnets {
		subnetInfo := vpcHandler.setterSubnet(s)
		subnetInfoList[i] = *subnetInfo
	}
	vpcInfo.SubnetInfoList = subnetInfoList

	return &vpcInfo
}

func (vpcHandler *ClouditVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {
	// Create Subnet
	subnetList := make([]subnet.SubnetInfo, len(vpcReqInfo.SubnetInfoList))
	for i, subnet := range vpcReqInfo.SubnetInfoList {
		result, err := vpcHandler.CreateSubnet(subnet)
		if err != nil {
			return irs.VPCInfo{}, err
		}
		subnetList[i] = result
	}
	vpcInfo := vpcHandler.setterVPC(subnetList)
	return *vpcInfo, nil
}

func (vpcHandler *ClouditVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	subnetList, err := vpcHandler.ListSubnet()
	if err != nil {
		return nil, err
	}
	vpcInfo := vpcHandler.setterVPC(subnetList)
	vpcInfoList := []*irs.VPCInfo{vpcInfo}
	return vpcInfoList, nil
}

func (vpcHandler *ClouditVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	vpcInfo, err := vpcHandler.ListVPC()
	if err != nil {
		return irs.VPCInfo{}, err
	}
	return *vpcInfo[0], err
}

func (vpcHandler *ClouditVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	vpcInfo, err := vpcHandler.GetVPC(vpcIID)
	if err != nil {
		return false, err
	}

	for _, subnetInfo := range vpcInfo.SubnetInfoList {
		// 기본 서브넷의 경우 삭제 예외처리
		if strings.EqualFold(subnetInfo.IId.NameId, defaultSubnetName) {
			continue
		}
		if ok, err := vpcHandler.DeleteSubnet(subnetInfo.IId); !ok {
			return false, err
		}
	}
	return true, nil
}

func (vpcHandler *ClouditVPCHandler) setterSubnet(subnet subnet.SubnetInfo) *irs.SubnetInfo {
	subnetInfo := irs.SubnetInfo{
		IId: irs.IID{
			NameId:   subnet.Name,
			SystemId: subnet.ID,
		},
		IPv4_CIDR: subnet.Addr + "/" + subnet.Prefix,
	}
	return &subnetInfo
}

func (vpcHandler *ClouditVPCHandler) CreateSubnet(subnetReqInfo irs.SubnetInfo) (subnet.SubnetInfo, error) {
	// 서브넷 이름 중복 체크
	checkSubnet, _ := vpcHandler.getSubnetByName(subnetReqInfo.IId.NameId)
	if checkSubnet != nil {
		errMsg := fmt.Sprintf("VirtualNetwork with name %s already exist", subnetReqInfo.IId.NameId)
		createErr := errors.New(errMsg)
		return subnet.SubnetInfo{}, createErr
	}

	vpcHandler.Client.TokenID = vpcHandler.CredentialInfo.AuthToken
	authHeader := vpcHandler.Client.AuthenticatedHeaders()

	var creatableSubnet subnet.SubnetInfo

	// 1. 사용 가능한 Subnet 목록 가져오기
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	if creatableSubnetList, err := subnet.ListCreatableSubnet(vpcHandler.Client, &requestOpts); err != nil {
		return subnet.SubnetInfo{}, err
	} else {
		if len(*creatableSubnetList) == 0 {
			allocateErr := errors.New(fmt.Sprintf("There is no PublicIPs to allocate"))
			return subnet.SubnetInfo{}, allocateErr
		} else {
			creatableSubnet = (*creatableSubnetList)[0]
		}
	}

	// 2. Subnet 생성
	reqInfo := subnet.VNetworkReqInfo{
		Name:   subnetReqInfo.IId.NameId,
		Addr:   creatableSubnet.Addr,
		Prefix: creatableSubnet.Prefix,
	}

	createOpts := client.RequestOpts{
		JSONBody:    reqInfo,
		MoreHeaders: authHeader,
	}

	subnetInfo, err := subnet.Create(vpcHandler.Client, &createOpts)
	if err != nil {
		return subnet.SubnetInfo{}, err
	}
	return *subnetInfo, nil
}

func (vpcHandler *ClouditVPCHandler) ListSubnet() ([]subnet.SubnetInfo, error) {
	vpcHandler.Client.TokenID = vpcHandler.CredentialInfo.AuthToken
	authHeader := vpcHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	subnetList, err := subnet.List(vpcHandler.Client, &requestOpts)
	if err != nil {
		return nil, err
	}
	return *subnetList, err
}

func (vpcHandler *ClouditVPCHandler) GetSubnet(subnetIId irs.IID) (subnet.SubnetInfo, error) {
	// 이름 기준 서브넷 조회
	subnetInfo, err := vpcHandler.getSubnetByName(subnetIId.NameId)
	if err != nil {
		cblogger.Error(err)
		return subnet.SubnetInfo{}, err
	}
	return *subnetInfo, nil
}

func (vpcHandler *ClouditVPCHandler) DeleteSubnet(subnetIId irs.IID) (bool, error) {
	// 이름 기준 서브넷 조회
	subnetInfo, err := vpcHandler.getSubnetByName(subnetIId.NameId)
	if err != nil {
		return false, err
	}

	vpcHandler.Client.TokenID = vpcHandler.CredentialInfo.AuthToken
	authHeader := vpcHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if err := subnet.Delete(vpcHandler.Client, subnetInfo.Addr, &requestOpts); err != nil {
		return false, err
	}
	return true, nil
}

func (vpcHandler *ClouditVPCHandler) getSubnetByName(subnetName string) (*subnet.SubnetInfo, error) {
	var subnetInfo *subnet.SubnetInfo

	vpcHandler.Client.TokenID = vpcHandler.CredentialInfo.AuthToken
	authHeader := vpcHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	subnetList, err := subnet.List(vpcHandler.Client, &requestOpts)
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
		err := errors.New(fmt.Sprintf("failed to find subnet with name %s", subnetName))
		return nil, err
	}
	return subnetInfo, nil
}
