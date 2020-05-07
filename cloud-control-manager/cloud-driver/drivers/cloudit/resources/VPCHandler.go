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
	defaultVPCName = "Default-VPC"
	defaultVPCCIDR = "10.0.0.0/16"
)

type ClouditVPCHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

/*
func (vpcHandler *ClouditVPCHandler) setterVPC(subnets []subnet.SubnetInfo) *irs.VPCInfo{
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
	for i, subnet := range subnets {
		subnetInfo, err := vpcHandler.GetSubnet(irs.IID{SystemId: subnet.ID})
		if err != nil {
			cblogger.Error("Failed to get subnet with Id %s, err=%s", subnet.ID, err)
			continue
		}
		subnetInfoList[i] = *subnetInfo
	}

	vpcInfo.SubnetInfoList = subnetInfoList

	return &vpcInfo
}*/

func (vpcHandler *ClouditVPCHandler) CreateVPC(vpcReqInfo irs.VPCReqInfo) (irs.VPCInfo, error) {

	//fmt.Println(vpcReqInfo)
	// VPC creation
	vpcInfo := irs.VPCInfo{
		IId: irs.IID{
			NameId:   defaultVPCName,
			SystemId: defaultVPCName,
		},
		IPv4_CIDR: defaultVPCCIDR,
	}

	// 2. Subnet creation
	subnetInfo := make([]irs.SubnetInfo, len(vpcReqInfo.SubnetInfoList))
	for i, subnet := range vpcReqInfo.SubnetInfoList {
		result, err := vpcHandler.CreateSubnet(subnet)
		if err != nil {
			return irs.VPCInfo{}, err
		}
		subnetInfo[i] = result
	}
	vpcInfo.SubnetInfoList = subnetInfo
	/*subnetInfo :=irs.SubnetInfo{

			IId: irs.IID{
				NameId:  defaultVPCName + "-subnet-1",
			},
			IPv4_CIDR: "180.0.10.0/24",
	}
	vpcInfo.SubnetInfoList = append(vpcInfo.SubnetInfoList,subnetInfo)*/

	return vpcInfo, nil
}

func (vpcHandler *ClouditVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {

	// set default VPC info
	vpcInfo := irs.VPCInfo{
		IId: irs.IID{
			NameId:   defaultVPCName,
			SystemId: defaultVPCName,
		},
		IPv4_CIDR: defaultVPCCIDR,
	}

	// set Subnet info
	subnetList, err := vpcHandler.ListSubnet()
	if err != nil {
		return nil, err
	}

	subnetInfoList := make([]irs.SubnetInfo, len(subnetList))
	for i, subnet := range subnetList {
		subnetInfoList[i] = *vpcHandler.setterSubnet(subnet)
	}
	vpcInfo.SubnetInfoList = subnetInfoList
	vpcInfoList := []*irs.VPCInfo{&vpcInfo}

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

	for _, irsSubnet := range vpcInfo.SubnetInfoList {

		cloutItSubnet, err := vpcHandler.GetSubnet(irsSubnet.IId)
		if err != nil {
			return false, err
		}
		vpcHandler.DeleteSubnet(cloutItSubnet.IId)
	}

	return true, nil
}

func (vpcHandler *ClouditVPCHandler) setterSubnet(subnet subnet.SubnetInfo) *irs.SubnetInfo {
	subnetInfo := irs.SubnetInfo{
		IId: irs.IID{
			NameId:   subnet.Name,
			SystemId: subnet.ID,
		},
		IPv4_CIDR: defaultVPCCIDR,
	}
	return &subnetInfo
}

func (vpcHandler *ClouditVPCHandler) CreateSubnet(subnetInfo irs.SubnetInfo) (irs.SubnetInfo, error) {
	// 서브넷 이름 중복 체크
	checkSubnet, _ := vpcHandler.getSubnetByName(subnetInfo.IId.NameId)
	if checkSubnet != nil {
		errMsg := fmt.Sprintf("VirtualNetwork with name %s already exist", subnetInfo.IId.NameId)
		createErr := errors.New(errMsg)
		return irs.SubnetInfo{}, createErr
	}

	vpcHandler.Client.TokenID = vpcHandler.CredentialInfo.AuthToken
	authHeader := vpcHandler.Client.AuthenticatedHeaders()

	var creatableSubnet subnet.SubnetInfo

	// 1. 사용 가능한 Subnet 목록 가져오기
	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	// VPC
	if creatableSubnetList, err := subnet.ListCreatableSubnet(vpcHandler.Client, &requestOpts); err != nil {
		return irs.SubnetInfo{}, err
	} else {
		if len(*creatableSubnetList) == 0 {
			allocateErr := errors.New(fmt.Sprintf("There is no PublicIPs to allocate"))
			return irs.SubnetInfo{}, allocateErr
		} else {
			creatableSubnet = (*creatableSubnetList)[0]
		}
	}

	// 2. Subnet 생성
	reqInfo := subnet.VNetworkReqInfo{
		Name:   subnetInfo.IId.NameId,
		Addr:   creatableSubnet.Addr,
		Prefix: creatableSubnet.Prefix,
	}

	createOpts := client.RequestOpts{
		JSONBody:    reqInfo,
		MoreHeaders: authHeader,
	}

	if cSubnet, err := subnet.Create(vpcHandler.Client, &createOpts); err != nil {
		return irs.SubnetInfo{}, err
	} else {
		cSubnet := vpcHandler.setterSubnet(*cSubnet)
		return *cSubnet, nil
	}
}

func (vpcHandler *ClouditVPCHandler) GetSubnet(subnetIId irs.IID) (*irs.SubnetInfo, error) {
	// 이름 기준 서브넷 조회
	subnet, err := vpcHandler.getSubnetByName(subnetIId.NameId)
	if err != nil {
		cblogger.Error(err)
		return &irs.SubnetInfo{}, err
	}

	subnetInfo := vpcHandler.setterSubnet(*subnet)

	return subnetInfo, nil
}

func (vpcHandler *ClouditVPCHandler) DeleteSubnet(subnetIId irs.IID) (bool, error) {
	// 이름 기준 서브넷 조회
	subnetInfo, err := vpcHandler.getSubnetByName(subnetIId.NameId)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	vpcHandler.Client.TokenID = vpcHandler.CredentialInfo.AuthToken
	authHeader := vpcHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if err := subnet.Delete(vpcHandler.Client, subnetInfo.Addr, &requestOpts); err != nil {
		//panic(err)
		return false, err
	} else {
		return true, nil
	}
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
		err := errors.New(fmt.Sprintf("failed to find virtual network with name %s", subnetName))
		return nil, err
	}

	return subnetInfo, nil
}
