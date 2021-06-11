package resources

import (
	"errors"
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/softlayer/softlayer-go/datatypes"
	"github.com/softlayer/softlayer-go/filter"
	"github.com/softlayer/softlayer-go/services"
	"github.com/softlayer/softlayer-go/sl"
	"strconv"
	"strings"
)

type IbmSecurityHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	AccountClient  *services.Account
	SecurityGroupClient *services.Network_SecurityGroup
}

func setterSecurityInfo(securityInfo *irs.SecurityInfo, securityItem *datatypes.Network_SecurityGroup) error {
	var vmSecurityInfo irs.SecurityInfo
	var result error
	defer func() {
		v := recover()
		if v != nil{
			securityInfo = nil
		}else {
			result = nil
			*securityInfo = vmSecurityInfo
		}
	}()
	var securityRules []irs.SecurityRuleInfo
	for _, rule := range securityItem.Rules{
		var vmSecurityRuleInfo irs.SecurityRuleInfo
		err := setterSecurityGroupRuleInfo(&vmSecurityRuleInfo,&rule)
		if err == nil {
			securityRules = append(securityRules, vmSecurityRuleInfo)
		}
	}
	var securityInfoIId irs.IID
	err := setSecurityInfoIId(&securityInfoIId,securityItem)
	if err == nil{
		vmSecurityInfo = irs.SecurityInfo{
			IId: irs.IID{
				NameId: *securityItem.Name,
				SystemId: strconv.Itoa(*securityItem.Id),
			},
			SecurityRules: &securityRules,
		}
	} else {
		result = errors.New("invalid SecurityGroupName or SecurityGroupId")
	}
	return result
}

func (securityHandler *IbmSecurityHandler) existCheckSecurityGroupByName(securityGroupName string) (bool,error){
	existFilter := filter.Path("securityGroups.name").Eq(securityGroupName).Build()
	filterObjects,err := securityHandler.AccountClient.Filter(existFilter).GetSecurityGroups()
	if err != nil{
		return true, err
	}
	if len(filterObjects) > 0 {
		return true, errors.New(fmt.Sprintf("Security Group with name %s already exist", securityGroupName))
	}
	return false, nil
}

func (securityHandler *IbmSecurityHandler) getterSecurityGroupByName(securityGroupName string) (datatypes.Network_SecurityGroup, error){
	if securityGroupName != ""{
		sgMask := "id;rules;name"
		existFilter := filter.Path("securityGroups.name").Eq(securityGroupName).Build()
		filterObjects, err := securityHandler.AccountClient.Mask(sgMask).Filter(existFilter).GetSecurityGroups()
		if err != nil{
			return datatypes.Network_SecurityGroup{}, err
		}
		if len(filterObjects) > 0 {
			return filterObjects[0], nil
		} else {
			return datatypes.Network_SecurityGroup{}, errors.New(fmt.Sprintf("Security Group with name %s Not exist", securityGroupName))
		}
	}
	return datatypes.Network_SecurityGroup{}, errors.New(fmt.Sprintf("Security Group name invalid"))
}

func (securityHandler *IbmSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error){
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, securityReqInfo.IId.NameId, "CreateSecurity()")

	sgName := securityReqInfo.IId.NameId
	exist, err := securityHandler.existCheckSecurityGroupByName(sgName)
	if exist {
		if err == nil{
			err = errors.New(fmt.Sprintf("Security Group with name %s already exist",sgName))
		}
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	requestSecurityRules := securityReqInfo.SecurityRules
	var createSecurityRules []datatypes.Network_SecurityGroup_Rule
	for _, requestSecurityRule := range *requestSecurityRules{
		var securityRule datatypes.Network_SecurityGroup_Rule
		err := setterSecurityGroupRule(&securityRule,&requestSecurityRule)
		if err == nil {
			createSecurityRules = append(createSecurityRules,securityRule)
		}else {
			LoggingError(hiscallInfo, err)
			return irs.SecurityInfo{}, errors.New("invalid SecurityReqInfo SecurityRules")
		}
	}
	start := call.Start()
	newSgObject, err:= securityHandler.SecurityGroupClient.CreateObject(&datatypes.Network_SecurityGroup{
		Name: &sgName,
	})

	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	_, err = securityHandler.SecurityGroupClient.Id(*newSgObject.Id).AddRules(createSecurityRules)
	if err != nil{
		LoggingError(hiscallInfo, err)
		errObjectIId :=irs.IID{SystemId: strconv.Itoa(*newSgObject.Id) }
		_, _ = securityHandler.DeleteSecurity(errObjectIId)
		return irs.SecurityInfo{}, err
	}

	createSGObject ,err := securityHandler.GetSecurity(irs.IID{SystemId: strconv.Itoa(*newSgObject.Id)})
	if err != nil{
		LoggingError(hiscallInfo, err)
		errObjectIId :=irs.IID{SystemId: strconv.Itoa(*newSgObject.Id) }
		_, _ = securityHandler.DeleteSecurity(errObjectIId)
		return irs.SecurityInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)
	return createSGObject, nil
}

func (securityHandler *IbmSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error){
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, "SECURITYGROUP", "ListSecurity()")
	sgMask := "id;rules;name"
	start := call.Start()
	allSGList ,err := securityHandler.SecurityGroupClient.Mask(sgMask).GetAllObjects()
	if err != nil{
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	var securityInfos []*irs.SecurityInfo
	for _, sgItem := range allSGList{
		var securityInfo irs.SecurityInfo
		err = setterSecurityInfo(&securityInfo,&sgItem)
		if err != nil{
			LoggingError(hiscallInfo, err)
			return nil, err
		} else {
			securityInfos = append(securityInfos,&securityInfo)
		}
	}
	LoggingInfo(hiscallInfo, start)
	return securityInfos, nil
}

func (securityHandler *IbmSecurityHandler) GetSecurity(keyIID irs.IID) (irs.SecurityInfo, error){
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, keyIID.NameId, "GetSecurity()")
	sgMask := "id;rules;name"
	// systemId == "" 이면 Name 검색
	var sgObject datatypes.Network_SecurityGroup
	start := call.Start()
	numSystemId , err := strconv.Atoi(keyIID.SystemId)
	if err != nil{
		sgObject ,err = securityHandler.getterSecurityGroupByName(keyIID.NameId)
		if err != nil{
			LoggingError(hiscallInfo, err)
			return irs.SecurityInfo{}, err
		}
	} else {
		sgObject ,err = securityHandler.SecurityGroupClient.Id(numSystemId).Mask(sgMask).GetObject()
		if err != nil{
			LoggingError(hiscallInfo, err)
			return irs.SecurityInfo{}, err
		}
	}
	var securityInfo irs.SecurityInfo
	err = setterSecurityInfo(&securityInfo,&sgObject)
	if err != nil{
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	} else {
		LoggingInfo(hiscallInfo, start)
		return securityInfo, nil
	}
}

func (securityHandler *IbmSecurityHandler) DeleteSecurity(keyIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, keyIID.NameId, "DeleteSecurity()")
	sgMask := "id;rules;name"
	var sgObject datatypes.Network_SecurityGroup
	numSystemId , err := strconv.Atoi(keyIID.SystemId)
	if err != nil{
		sgObject ,err = securityHandler.getterSecurityGroupByName(keyIID.NameId)
		if err != nil{
			LoggingError(hiscallInfo, err)
			return false, err
		}
	} else {
		sgObject ,err = securityHandler.SecurityGroupClient.Id(numSystemId).Mask(sgMask).GetObject()
		if err != nil {
			LoggingError(hiscallInfo, err)
			return false, err
		}
	}
	start := call.Start()
	result, err :=  securityHandler.SecurityGroupClient.Id(*sgObject.Id).DeleteObject()
	LoggingInfo(hiscallInfo, start)
	return result, err
}

func setSecurityInfoIId(securityInfoIId *irs.IID ,securityItem *datatypes.Network_SecurityGroup) error {
	var vmSecurityInfoIId irs.IID
	var result error
	defer func() {
		v := recover()
		if v != nil{
			result = errors.New("invalid Network_SecurityGroup")
		}else {
			*securityInfoIId = vmSecurityInfoIId
		}
	}()
	vmSecurityInfoIId = irs.IID{
		NameId: *securityItem.Name,
		SystemId: strconv.Itoa(*securityItem.Id),
	}
	return result
}

// softlayer SGRule => spider SecurityRuleInfo
func setterSecurityGroupRuleInfo(securityRuleInfo *irs.SecurityRuleInfo,rule *datatypes.Network_SecurityGroup_Rule) error {
	vmSecurityRuleInfo := irs.SecurityRuleInfo{}
	defer func() {
		v := recover()
		if v == nil{
			*securityRuleInfo = vmSecurityRuleInfo
		}
	}()
	if rule != nil && *rule.Ethertype == "IPv4"{
		var fromPort string
		var toPort string
		var ipProtocol string
		var direction string
		if rule.Protocol == nil {
			ipProtocol = "ALL"
		} else {
			ipProtocol = strings.ToUpper(*rule.Protocol)
		}
		switch ipProtocol {
		case "ALL": {
			fromPort = "0"
			toPort = "65535"
		}
		case "ICMP": {
			if rule.PortRangeMin == nil && rule.PortRangeMax == nil{
				fromPort = "0"
				toPort = "255"
			} else {
				if rule.PortRangeMin == nil{
					fromPort = strconv.Itoa(*rule.PortRangeMax)
					toPort = strconv.Itoa(*rule.PortRangeMax)
				}else if rule.PortRangeMax == nil{
					fromPort = strconv.Itoa(*rule.PortRangeMin)
					toPort = strconv.Itoa(*rule.PortRangeMin)
				}else{
					if *rule.PortRangeMin == 0 && *rule.PortRangeMax == 0 {
						fromPort = "0"
						toPort = "255"
					}else{
						fromPort = strconv.Itoa(*rule.PortRangeMin)
						toPort = strconv.Itoa(*rule.PortRangeMax)
					}
				}
			}
		}
		default: {
			//"TCP", "UDP"
			if rule.PortRangeMin == nil && rule.PortRangeMax == nil{
				fromPort = "0"
				toPort = "65535"
			} else {
				if rule.PortRangeMin == nil{
					fromPort = strconv.Itoa(*rule.PortRangeMax)
					toPort = strconv.Itoa(*rule.PortRangeMax)
				}else if rule.PortRangeMax == nil{
					fromPort = strconv.Itoa(*rule.PortRangeMin)
					toPort = strconv.Itoa(*rule.PortRangeMin)
				}else{
					fromPort = strconv.Itoa(*rule.PortRangeMin)
					toPort = strconv.Itoa(*rule.PortRangeMax)
				}
			}
		}
		}
		switch *rule.Direction {
		case "ingress" :
			direction = "inbound"
		case "egress" :
			direction = "outbound"
		}
		cidr := "0.0.0.0/0"
		if rule.RemoteIp != nil{
			cidr = *rule.RemoteIp
		}
		vmSecurityRuleInfo = irs.SecurityRuleInfo {
			FromPort: fromPort,
			ToPort: toPort,
			Direction: direction,
			IPProtocol: ipProtocol,
			CIDR:cidr,
		}
		return nil
	}
	return errors.New("invalid Rule")
}

func setterSecurityGroupRule(rule *datatypes.Network_SecurityGroup_Rule, securityRuleInfo *irs.SecurityRuleInfo) error {
	var vmSecurityRule datatypes.Network_SecurityGroup_Rule
	var result error
	defer func() {
		v := recover()
		if v != nil{
			result = errors.New("invalid Rule")
		}else {
			*rule = vmSecurityRule
		}
	}()
	var PortRangeMax *int
	var PortRangeMin *int
	var Direction string
	RemoteIp := securityRuleInfo.CIDR
	if securityRuleInfo.CIDR == "" {
		RemoteIp = "0.0.0.0/0"
	}
	Protocol := strings.ToLower(securityRuleInfo.IPProtocol)
	switch Protocol {
	case "icmp":{
		checkPortRangeMax, err := strconv.Atoi(securityRuleInfo.ToPort)
		if err != nil {
			result = errors.New("invalid Port Number")
		}
		checkPortRangeMin, err := strconv.Atoi(securityRuleInfo.FromPort)
		if err != nil {
			result = errors.New("invalid Port Number")
		}
		if checkPortRangeMax >= 0 && checkPortRangeMax <= 255  &&  checkPortRangeMin >= 0 && checkPortRangeMin <= 255  && checkPortRangeMin <= checkPortRangeMax {
			PortRangeMax = sl.Int(checkPortRangeMax)
			PortRangeMin =  sl.Int(checkPortRangeMin)
		}else{
			if checkPortRangeMax == -1 || checkPortRangeMin == -1 {
				PortRangeMax = nil
				PortRangeMin =  nil
			}
			result = errors.New("invalid Port Number")
		}
	}
	case "udp", "tcp" :{
		checkPortRangeMax, err := strconv.Atoi(securityRuleInfo.ToPort)
		if err != nil {
			result = errors.New("invalid Port Number")
		}
		checkPortRangeMin, err := strconv.Atoi(securityRuleInfo.FromPort)
		if err != nil {
			result = errors.New("invalid Port Number")
		}
		if checkPortRangeMax >= 1 && checkPortRangeMax <= 65535  &&  checkPortRangeMin >= 1 && checkPortRangeMin <= 65535  && checkPortRangeMin <= checkPortRangeMax {
			PortRangeMax = sl.Int(checkPortRangeMax)
			PortRangeMin =  sl.Int(checkPortRangeMin)
		}else{
			if checkPortRangeMax == -1 || checkPortRangeMin == -1 {
				PortRangeMax = nil
				PortRangeMin =  nil
			}
			result = errors.New("invalid Port Number")
		}
	}
	default:
		result = errors.New("invalid Port Number")
	}
	switch securityRuleInfo.Direction {
	case "inbound":{
		Direction = "ingress"
	}
	case "outbound":{
		Direction = "egress"
	}
	default:
		result = errors.New("invalid Direction")
	}
	vmSecurityRule = datatypes.Network_SecurityGroup_Rule{
		Direction: sl.String(Direction), //ingress, egress
		Protocol: sl.String(Protocol), // tcp, udp, icmp
		PortRangeMin: PortRangeMin,
		PortRangeMax: PortRangeMax,
		RemoteIp: sl.String(RemoteIp),
	}
	return result
}
