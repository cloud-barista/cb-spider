package resources

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	keypair "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/common"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/dna/subnet"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	defaultVPCCIDR = "10.0.0.0/16"
	VPC            = "VPC"
	VPCProvider    = "CloudITVPC"
)

type ClouditVPCHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func (vpcHandler *ClouditVPCHandler) setterVPC(subnets []subnet.SubnetInfo, vpcName string) *irs.VPCInfo {
	// VPC 정보 맵핑
	vpcInfo := irs.VPCInfo{
		IId: irs.IID{
			NameId:   vpcName,
			SystemId: vpcName,
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
	hashString, err := CreateVPCHashString(vpcHandler.CredentialInfo)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}
	// Check NameId value
	if vpcReqInfo.IId.NameId == "" {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = Invalid IID"))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}
	// Check Exist VPC
	_, getKeyErr := keypair.GetKey(VPCProvider, hashString, ClouditVPCREGISTER)
	if getKeyErr == nil {
		// Exist VPC
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = Cloudit can only create one VPC."))
		cblogger.Error(createErr)
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}

	// Check Creatable Subnets
	start := call.Start()
	var createSubnetList []irs.SubnetInfo
	for _, vpcSubnet := range vpcReqInfo.SubnetInfoList {
		if err := vpcHandler.checkCreatableSubnetCIDR(vpcSubnet.IPv4_CIDR); err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VPCInfo{}, createErr
		}
		createSubnetList = append(createSubnetList, vpcSubnet)
	}
	// Create Subnets
	subnetList := make([]subnet.SubnetInfo, len(createSubnetList))
	for i, vpcSubnet := range vpcReqInfo.SubnetInfoList {
		result, err := vpcHandler.CreateSubnet(vpcSubnet)
		if err != nil {
			for _, newCreateSubnet := range subnetList {
				vpcHandler.DeleteSubnet(irs.IID{NameId: newCreateSubnet.Name})
			}
			createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.VPCInfo{}, createErr
		}
		subnetList[i] = result
	}
	err = keypair.AddKey(VPCProvider, hashString, ClouditVPCREGISTER, assembleVPCRegisterValue(vpcReqInfo.IId.NameId, getSubnetNames(subnetList)))
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create VPC err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.VPCInfo{}, createErr
	}
	LoggingInfo(hiscallInfo, start)

	vpcInfo := vpcHandler.setterVPC(subnetList, vpcReqInfo.IId.NameId)
	return *vpcInfo, nil
}

func (vpcHandler *ClouditVPCHandler) ListVPC() ([]*irs.VPCInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VPCSUBNET, VPC, "ListVPC()")

	start := call.Start()

	hashString, err := CreateVPCHashString(vpcHandler.CredentialInfo)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List VPC err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}

	key, err := keypair.GetKey(VPCProvider, hashString, ClouditVPCREGISTER)
	if err != nil {
		return []*irs.VPCInfo{}, nil
	}
	vpcName, subnetNames, err := disassembleVPCRegisterValue(key.Value)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to List VPC err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	var subnetList []subnet.SubnetInfo
	for _, subnetName := range subnetNames {
		subnetinfo, err := vpcHandler.GetSubnet(irs.IID{NameId: subnetName})
		if err != nil {
			continue
		}
		subnetList = append(subnetList, subnetinfo)
	}
	LoggingInfo(hiscallInfo, start)

	err = keypair.AddKey(VPCProvider, hashString, ClouditVPCREGISTER, assembleVPCRegisterValue(vpcName, getSubnetNames(subnetList)))
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to List VPC err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return nil, createErr
	}
	vpcInfo := vpcHandler.setterVPC(subnetList, vpcName)

	vpcInfoList := []*irs.VPCInfo{vpcInfo}
	return vpcInfoList, nil
}

func (vpcHandler *ClouditVPCHandler) GetVPC(vpcIID irs.IID) (irs.VPCInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.Client.IdentityEndpoint, call.VPCSUBNET, vpcIID.NameId, "GetVPC()")
	start := call.Start()
	hashString, err := CreateVPCHashString(vpcHandler.CredentialInfo)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VPC err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VPCInfo{}, getErr
	}
	// Check NameId value
	if vpcIID.NameId == "" {
		getErr := errors.New(fmt.Sprintf("Failed to Get VPC err = Invalid IID"))
		cblogger.Error(getErr)
		LoggingError(hiscallInfo, getErr)
		return irs.VPCInfo{}, getErr
	}
	// Check Exist VPC
	key, err := keypair.GetKey(VPCProvider, hashString, ClouditVPCREGISTER)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VPC err = Not Exist"))
		cblogger.Error(getErr)
		LoggingError(hiscallInfo, getErr)
		return irs.VPCInfo{}, getErr
	}
	vpcName, subnetNames, err := disassembleVPCRegisterValue(key.Value)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VPC err = Not Exist"))
		cblogger.Error(getErr)
		LoggingError(hiscallInfo, getErr)
		return irs.VPCInfo{}, getErr
	}
	if vpcName != vpcIID.NameId {
		getErr := errors.New(fmt.Sprintf("Failed to Get VPC err = Not Exist VPC : %s", vpcIID.NameId))
		cblogger.Error(getErr)
		LoggingError(hiscallInfo, getErr)
		return irs.VPCInfo{}, getErr
	}
	var subnetList []subnet.SubnetInfo
	for _, subnetName := range subnetNames {
		subnetinfo, err := vpcHandler.GetSubnet(irs.IID{NameId: subnetName})
		if err != nil {
			continue
		}
		subnetList = append(subnetList, subnetinfo)
	}
	LoggingInfo(hiscallInfo, start)

	err = keypair.AddKey(VPCProvider, hashString, ClouditVPCREGISTER, assembleVPCRegisterValue(vpcName, getSubnetNames(subnetList)))
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get VPC err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.VPCInfo{}, getErr
	}
	vpcInfo := vpcHandler.setterVPC(subnetList, vpcName)

	return *vpcInfo, err
}

func (vpcHandler *ClouditVPCHandler) DeleteVPC(vpcIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(vpcHandler.Client.IdentityEndpoint, call.VPCSUBNET, vpcIID.NameId, "DeleteVPC()")
	hashString, err := CreateVPCHashString(vpcHandler.CredentialInfo)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete VPC err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	vpcInfo, err := vpcHandler.GetVPC(vpcIID)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete VPC err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}

	start := call.Start()
	for _, subnetInfo := range vpcInfo.SubnetInfoList {
		subnetList, err := vpcHandler.ListSubnet()
		if err != nil {
			delErr := errors.New(fmt.Sprintf("Failed to Delete VPC err = %s", err.Error()))
			cblogger.Error(delErr.Error())
			LoggingError(hiscallInfo, delErr)
			return false, delErr
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

	err = keypair.DelKey(VPCProvider, hashString, ClouditVPCREGISTER)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete VPC err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
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
	start := call.Start()
	hashString, err := CreateVPCHashString(vpcHandler.CredentialInfo)
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}
	key, err := keypair.GetKey(VPCProvider, hashString, ClouditVPCREGISTER)
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}
	vpcName, subnetNames, err := disassembleVPCRegisterValue(key.Value)
	if vpcName != vpcIID.NameId {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = Not Exist %s", vpcName))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}

	checkSubnet, _ := vpcHandler.getSubnetByName(subnetInfo.IId.NameId)
	if checkSubnet != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = Subnet with name %s already exist", subnetInfo.IId.NameId))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}
	err = vpcHandler.checkCreatableSubnetCIDR(subnetInfo.IPv4_CIDR)
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}
	cidrArrays := strings.Split(subnetInfo.IPv4_CIDR, "/")

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

	newSubnet, err := subnet.Create(vpcHandler.Client, &createOpts)
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}

	subnetNames = append(subnetNames, newSubnet.Name)

	err = keypair.AddKey(VPCProvider, hashString, ClouditVPCREGISTER, assembleVPCRegisterValue(vpcName, subnetNames))
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}
	result, err := vpcHandler.GetVPC(vpcIID)
	if err != nil {
		addSubnetErr := errors.New(fmt.Sprintf("Failed to Add Subnet err = %s", err.Error()))
		cblogger.Error(addSubnetErr.Error())
		LoggingError(hiscallInfo, addSubnetErr)
		return irs.VPCInfo{}, addSubnetErr
	}

	LoggingInfo(hiscallInfo, start)

	return result, nil
}

func (vpcHandler *ClouditVPCHandler) RemoveSubnet(vpcIID irs.IID, subnetIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.VPCSUBNET, subnetIID.NameId, "RemoveSubnet()")
	start := call.Start()

	hashString, err := CreateVPCHashString(vpcHandler.CredentialInfo)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Remove Subnet err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	key, err := keypair.GetKey(VPCProvider, hashString, ClouditVPCREGISTER)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Remove Subnet err = Not Exist VPC"))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	vpcName, subnetNames, err := disassembleVPCRegisterValue(key.Value)
	if vpcName != vpcIID.NameId {
		delErr := errors.New(fmt.Sprintf("Failed to Remove Subnet err = Not Exist %s", vpcName))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}

	subnetInfo, err := vpcHandler.getSubnetByName(subnetIID.NameId)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Remove Subnet err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}

	vpcHandler.Client.TokenID = vpcHandler.CredentialInfo.AuthToken
	authHeader := vpcHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	err = subnet.Delete(vpcHandler.Client, subnetInfo.Addr, &requestOpts)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Remove Subnet err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	for i, subnetname := range subnetNames {
		if subnetname == subnetInfo.Name {
			subnetNames = append(subnetNames[:i], subnetNames[i+1:]...)
			break
		}
	}
	err = keypair.AddKey(VPCProvider, hashString, ClouditVPCREGISTER, assembleVPCRegisterValue(vpcName, subnetNames))
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Remove Subnet err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
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

//func (vpcHandler *ClouditVPCHandler) checkExistSubnet(cidr string) (bool, error) {
//	creatableSubnetCIDRList,err := vpcHandler.creatableSubnetCIDRList()
//	if err != nil{
//		return false, errors.New(fmt.Sprintf("Failed Get Cloudit Creatable Subnet List err = %s",err.Error()))
//	}
//	// Check Valid Cidr
//	err = checkSubnetCIDR(cidr)
//	if err != nil{
//		return false, errors.New(fmt.Sprintf("%s Currently, the list of subnets that can be created is \n[ %s ]",err.Error(),strings.Join(creatableSubnetCIDRList," , ")))
//	}
//	for _, creatableSubnetAddr := range creatableSubnetCIDRList{
//		if creatableSubnetAddr == cidr {
//			return true, errors.New(fmt.Sprintf("already Exist Subnet IPv4_CIDR : %s . Currently, the list of subnets that can be created is \n[ %s ] ",cidr,strings.Join(creatableSubnetCIDRList," , ")))
//		}
//	}
//	return false, nil
//}

func getSubnetNames(subnetList []subnet.SubnetInfo) []string {
	subnetListNames := make([]string, len(subnetList))
	for i, subnetInfo := range subnetList {
		subnetListNames[i] = subnetInfo.Name
	}
	return subnetListNames
}

func assembleVPCRegisterValue(vpcName string, subnetNames []string) string {
	subnetListNamesString := strings.Join(makeSliceUnique(subnetNames), ",")
	return vpcName + "/" + subnetListNamesString
}

func disassembleVPCRegisterValue(registerValue string) (vpcName string, subnetNames []string, err error) {
	vpcRegisterStringSplits := strings.Split(registerValue, "/")
	if len(vpcRegisterStringSplits) != 2 {
		getErr := errors.New(fmt.Sprintf("Invalid VPCRegister"))
		return "", nil, getErr
	}
	vpcSubnetNames := strings.Split(vpcRegisterStringSplits[1], ",")

	return vpcRegisterStringSplits[0], makeSliceUnique(vpcSubnetNames), nil
}

func makeClouditAllSubnetsRange() []string {
	var startSubnetRangeNumber = 0  // 10.0.0.0
	var lastSubnetRangeNumber = 248 // 10.0.248.0
	var addrArray []string
	for {
		if startSubnetRangeNumber > lastSubnetRangeNumber {
			break
		}
		addrArray = append(addrArray, fmt.Sprintf("10.0.%d.0", startSubnetRangeNumber))
		startSubnetRangeNumber = startSubnetRangeNumber + 4
	}
	return addrArray
}

func (vpcHandler *ClouditVPCHandler) creatableSubnetCIDRList() ([]string, error) {
	existAllSubnets, err := vpcHandler.ListSubnet()
	if err != nil {
		return nil, err
	}
	allCreatableSubnetAddrList := makeClouditAllSubnetsRange()
	for _, existSubnet := range existAllSubnets {
		for j, creatableSubnetAddr := range allCreatableSubnetAddrList {
			if existSubnet.Addr == creatableSubnetAddr {
				allCreatableSubnetAddrList = append(allCreatableSubnetAddrList[:j], allCreatableSubnetAddrList[j+1:]...)
				break
			}
		}
	}
	creatableSubnetCIDRList := make([]string, len(allCreatableSubnetAddrList))
	for i, creatableSubnetAddr := range allCreatableSubnetAddrList {
		creatableSubnetCIDRList[i] = fmt.Sprintf("%s/%s", creatableSubnetAddr, "22")
	}
	return creatableSubnetCIDRList, nil
}
func (vpcHandler *ClouditVPCHandler) GetDefaultVPC() (irs.VPCInfo, error) {
	hashString, err := CreateVPCHashString(vpcHandler.CredentialInfo)
	if err != nil {
		return irs.VPCInfo{}, err
	}

	key, err := keypair.GetKey(VPCProvider, hashString, ClouditVPCREGISTER)
	if err != nil {
		return irs.VPCInfo{}, err
	}
	vpcName, subnetNames, err := disassembleVPCRegisterValue(key.Value)
	if err != nil {
		return irs.VPCInfo{}, err
	}
	var subnetList []subnet.SubnetInfo
	for _, subnetName := range subnetNames {
		subnetinfo, err := vpcHandler.GetSubnet(irs.IID{NameId: subnetName})
		if err != nil {
			continue
		}
		subnetList = append(subnetList, subnetinfo)
	}

	err = keypair.AddKey(VPCProvider, hashString, ClouditVPCREGISTER, assembleVPCRegisterValue(vpcName, getSubnetNames(subnetList)))
	if err != nil {
		return irs.VPCInfo{}, err
	}
	vpcInfo := vpcHandler.setterVPC(subnetList, vpcName)
	return *vpcInfo, nil
}

func (vpcHandler *ClouditVPCHandler) checkCreatableSubnetCIDR(checkCIDR string) error {
	creatableSubnetCIDRList, err := vpcHandler.creatableSubnetCIDRList()
	if err != nil {
		return errors.New(fmt.Sprintf("Failed Get Cloudit Creatable Subnet List err = %s", err.Error()))
	}
	// Check Valid Cidr
	err = checkSubnetCIDR(checkCIDR)
	if err != nil {
		return errors.New(fmt.Sprintf("%s Currently, the list of subnets that can be created is \n[ %s ]", err.Error(), strings.Join(creatableSubnetCIDRList, ",")))
	}
	// Check Valid creatable
	creatable := false
	for _, creatableSubnetAddr := range creatableSubnetCIDRList {
		if creatableSubnetAddr == checkCIDR {
			creatable = true
			break
		}
	}
	if creatable {
		return nil
	}
	return errors.New(fmt.Sprintf("already Exist Subnet IPv4_CIDR : %s . Currently, the list of subnets that can be created is \n[ %s ]", checkCIDR, strings.Join(creatableSubnetCIDRList, ",")))
}

func makeSliceUnique(s []string) []string {
	keys := make(map[string]struct{})
	res := make([]string, 0)
	for _, val := range s {
		if _, ok := keys[val]; ok {
			continue
		} else {
			keys[val] = struct{}{}
			res = append(res, val)
		}
	}
	return res
}

func (VPCHandler *ClouditVPCHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	return nil, errors.New("Does not support ListIID() yet!!")
}
