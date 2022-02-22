package resources

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/dna/subnet"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	defaultVPCName    = "Default-VPC"
	defaultVPCCIDR    = "10.0.0.0/16"
	defaultSubnetCIDR = "10.0.0.0/22"
	VPC               = "VPC"
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
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VPCSUBNET, VPC, "CreateVPC()")

	// Create Subnet
	start := call.Start()
	var createSubnetList []irs.SubnetInfo
	for _, vpcSubnet := range vpcReqInfo.SubnetInfoList {
		if err := checkSubnetCIDR(vpcSubnet.IPv4_CIDR); err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VPCInfo{}, createErr
		}
		if !strings.EqualFold(vpcSubnet.IPv4_CIDR, defaultSubnetCIDR) {
			exist, err := vpcHandler.checkExistSubnet(vpcSubnet.IPv4_CIDR)
			if exist {
				createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = Subnet IPv4_CIDR %s already exist", vpcSubnet.IPv4_CIDR))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VPCInfo{}, createErr
			}
			if err != nil {
				createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
				cblogger.Error(createErr.Error())
				LoggingError(hiscallInfo, createErr)
				return irs.VPCInfo{}, createErr
			}
			createSubnetList = append(createSubnetList, vpcSubnet)
		}
	}
	subnetList := make([]subnet.SubnetInfo, len(createSubnetList))
	for i, vpcSubnet := range vpcReqInfo.SubnetInfoList {
		result, err := vpcHandler.CreateSubnet(vpcSubnet)
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VPCInfo{}, createErr
		}
		subnetList[i] = result
	}
	LoggingInfo(hiscallInfo, start)

	vpcInfo := vpcHandler.setterVPC(subnetList)
	return *vpcInfo, nil
}

func (vpcHandler *ClouditVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VPCSUBNET, VPC, "ListVPC()")

	start := call.Start()
	subnetList, err := vpcHandler.ListSubnet()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VPCList err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	LoggingInfo(hiscallInfo, start)

	vpcInfo := vpcHandler.setterVPC(subnetList)
	vpcInfoList := []*irs.VPCInfo{vpcInfo}
	return vpcInfoList, nil
}

func (vpcHandler *ClouditVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.Client.IdentityEndpoint, call.VPCSUBNET, vpcIID.NameId, "GetVPC()")
	start := call.Start()

	subnetList, err := vpcHandler.ListSubnet()
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VPC err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VPCInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)

	vpcInfo := vpcHandler.setterVPC(subnetList)

	return *vpcInfo, err
}

func (vpcHandler *ClouditVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.Client.IdentityEndpoint, call.VPCSUBNET, vpcIID.NameId, "DeleteVPC()")

	vpcInfo, err := vpcHandler.GetVPC(vpcIID)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return false, err
	}

	start := call.Start()
	for _, subnetInfo := range vpcInfo.SubnetInfoList {
		subnetList, err := vpcHandler.ListSubnet()
		if err != nil {
			cblogger.Error(err.Error())
			LoggingError(hiscallInfo, err)
			return false, err
		}
		for _, value := range subnetList {
			if value.ID == subnetInfo.IId.SystemId {
				if value.Protection == 0 {
					if ok, _ := vpcHandler.DeleteSubnet(subnetInfo.IId); ok {
						break
					}
				}
			}
		}
	}
	LoggingInfo(hiscallInfo, start)

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
		errMsg := fmt.Sprintf("subnet with name %s already exist", subnetReqInfo.IId.NameId)
		createErr := errors.New(errMsg)
		return subnet.SubnetInfo{}, createErr
	}

	vpcHandler.Client.TokenID = vpcHandler.CredentialInfo.AuthToken
	authHeader := vpcHandler.Client.AuthenticatedHeaders()

	cidrArrays := strings.Split(subnetReqInfo.IPv4_CIDR, "/")

	// 2. Subnet 생성
	reqInfo := subnet.VNetworkReqInfo{
		Name:       subnetReqInfo.IId.NameId,
		Addr:       cidrArrays[0],
		Prefix:     cidrArrays[1],
		Protection: 0,
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

func (vpcHandler *ClouditVPCHandler) AddSubnet(vpcIID irs.IID, subnetInfo irs.SubnetInfo) (irs.VPCInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VPCSUBNET, vpcIID.NameId, "AddSubnet()")

	checkSubnet, _ := vpcHandler.getSubnetByName(subnetInfo.IId.NameId)
	if checkSubnet != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = Subnet with name %s already exist", subnetInfo.IId.NameId))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}
	exist, err := vpcHandler.checkExistSubnet(subnetInfo.IPv4_CIDR)

	if exist {
		createErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = Subnet IPv4_CIDR %s already exist", subnetInfo.IPv4_CIDR))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}
	cidrArrays := strings.Split(subnetInfo.IPv4_CIDR, "/")
	if strings.EqualFold(cidrArrays[0], strings.Split(defaultSubnetCIDR, "/")[0]) {
		createErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = 10.0.0.0/22 is created as a default subnet"))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}

	vpcHandler.Client.TokenID = vpcHandler.CredentialInfo.AuthToken
	authHeader := vpcHandler.Client.AuthenticatedHeaders()

	// 2. Subnet 생성
	reqInfo := subnet.VNetworkReqInfo{
		Name:       subnetInfo.IId.NameId,
		Addr:       cidrArrays[0],
		Prefix:     cidrArrays[1],
		Protection: 0,
	}

	createOpts := client.RequestOpts{
		JSONBody:    reqInfo,
		MoreHeaders: authHeader,
	}

	start := call.Start()
	_, err = subnet.Create(vpcHandler.Client, &createOpts)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}

	result, err := vpcHandler.GetVPC(vpcIID)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}

	LoggingInfo(hiscallInfo, start)

	return result, nil
}

func (vpcHandler *ClouditVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VPCSUBNET, subnetIID.NameId, "RemoveSubnet()")

	subnetInfo, err := vpcHandler.getSubnetByName(subnetIID.NameId)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Remove Subnet err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return false, err
	}

	vpcHandler.Client.TokenID = vpcHandler.CredentialInfo.AuthToken
	authHeader := vpcHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	start := call.Start()
	err = subnet.Delete(vpcHandler.Client, subnetInfo.Addr, &requestOpts)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Remove Subnet err = %s", err))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func checkSubnetCIDR(cidr string) error {
	cidrArrays := strings.Split(cidr, "/")
	defaultVPCAddrstrings := strings.Split(strings.Split(defaultVPCCIDR, "/")[0], ".")
	cidrAddrStrings := strings.Split(cidrArrays[0], ".")
	if len(cidrArrays) != 2 {
		return errors.New("invalid IPv4_CIDR")
	}
	// addr Checked
	if net.ParseIP(cidrArrays[0]) == nil {
		return errors.New("invalid IPv4_CIDR")
	} else {
		defaultVPCAddrRange := fmt.Sprintf("%s.%s", defaultVPCAddrstrings[0], defaultVPCAddrstrings[1])
		checkedCidrAddrRange := fmt.Sprintf("%s.%s", cidrAddrStrings[0], cidrAddrStrings[1])
		num, err := strconv.Atoi(cidrAddrStrings[2])
		num2, err := strconv.Atoi(cidrAddrStrings[3])
		if err != nil {
			return errors.New("invalid IPv4_CIDR, Cloudit provides subnets in units of 22 blocks, starting from 10.0.0.0 addresses to 10.0.252.0")
		}
		if num%4 != 0 || num > 248 || num2 != 0 || !strings.EqualFold(defaultVPCAddrRange, checkedCidrAddrRange) {
			return errors.New("invalid IPv4_CIDR, Cloudit provides subnets in units of 22 blocks, starting from 10.0.0.0 addresses to 10.0.252.0")
		}
	}
	// block Checked
	if cidrArrays[1] != "22" {
		return errors.New("invalid IPv4_CIDR, Cloudit provides subnets in units of 22 blocks, starting from 10.0.0.0 addresses to 10.0.252.0")
	}
	return nil
}

func (vpcHandler *ClouditVPCHandler) checkExistSubnet(cidr string) (bool, error) {
	if err := checkSubnetCIDR(cidr); err != nil {
		return false, err
	}
	subnetList, err := vpcHandler.ListSubnet()
	if err != nil {
		return false, err
	}
	cidrArrays := strings.Split(cidr, "/")
	for _, subnetItem := range subnetList {
		if strings.EqualFold(subnetItem.Addr, cidrArrays[0]) {
			return true, errors.New("already Exist Subnet IPv4_CIDR")
		}
	}
	return false, nil
}
