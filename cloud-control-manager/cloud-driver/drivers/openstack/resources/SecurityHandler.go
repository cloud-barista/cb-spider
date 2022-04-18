package resources

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/gophercloud/gophercloud/openstack/networking/v2/extensions/security/rules"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	Inbound       = "inbound"
	Outbound      = "outbound"
	ICMP          = "icmp"
	SecurityGroup = "SECURITYGROUP"
)

type OpenStackSecurityHandler struct {
	Client        *gophercloud.ServiceClient
	NetworkClient *gophercloud.ServiceClient
}

func (securityHandler *OpenStackSecurityHandler) setterSeg(secGroup secgroups.SecurityGroup) *irs.SecurityInfo {
	secInfo := &irs.SecurityInfo{
		IId: irs.IID{
			NameId:   secGroup.Name,
			SystemId: secGroup.ID,
		},
	}

	listOpts := rules.ListOpts{
		SecGroupID: secGroup.ID,
	}
	pager, err := rules.List(securityHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		return nil
	}
	secList, err := rules.ExtractRules(pager)
	if err != nil {
		return nil
	}

	// 보안그룹 룰 정보 등록
	var secRuleList []irs.SecurityRuleInfo
	for _, rule := range secList {
		if rule.EtherType == string(rules.EtherType4) {
			ruleInfo := convertOpenStackRuleToCBRuleInfo(&rule)
			secRuleList = append(secRuleList, ruleInfo)
		}
	}
	secInfo.SecurityRules = &secRuleList

	return secInfo
}

func (securityHandler *OpenStackSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (createdSG irs.SecurityInfo, creteErr error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(securityHandler.Client.IdentityEndpoint, call.SECURITYGROUP, securityReqInfo.IId.NameId, "CreateSecurity()")

	// Check SecurityGroup Exists
	secGroupList, err := securityHandler.ListSecurity()
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	for _, sg := range secGroupList {
		if sg.IId.NameId == securityReqInfo.IId.NameId {
			createErr := errors.New(fmt.Sprintf("Security Group with name %s already exist", securityReqInfo.IId.NameId))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.SecurityInfo{}, createErr
		}
	}

	// Create SecurityGroup
	createOpts := secgroups.CreateOpts{
		Name:        securityReqInfo.IId.NameId,
		Description: securityReqInfo.IId.NameId,
	}

	start := call.Start()
	group, err := secgroups.Create(securityHandler.Client, createOpts).Extract()
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}

	defer func() {
		if creteErr != nil {
			secgroups.Delete(securityHandler.Client, group.ID)
		}
	}()

	securityDefaultInfo := securityHandler.setterSeg(*group)

	var updateRules []irs.SecurityRuleInfo

	for _, newRule := range *securityReqInfo.SecurityRules {
		chk := true
		for _, baseRule := range *securityDefaultInfo.SecurityRules {
			if equalsRule(newRule, baseRule) {
				chk = false
				break
			}
		}
		if chk {
			updateRules = append(updateRules, newRule)
		}
	}

	createRuleOpts, err := convertCBRuleInfosToOpenStackRules(group.ID, &updateRules)
	if err != nil {
		creteErr = err
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, creteErr
	}
	// Create SecurityGroup Rules
	for _, createRuleOpt := range *createRuleOpts {
		_, err := rules.Create(securityHandler.NetworkClient, createRuleOpt).Extract()
		if err != nil {
			creteErr = err
			cblogger.Error(err.Error())
			LoggingError(hiscallInfo, err)
			return irs.SecurityInfo{}, creteErr
		}
	}

	// 생성된 SecurityGroup 정보 리턴
	securityInfo, err := securityHandler.GetSecurity(irs.IID{SystemId: group.ID})
	if err != nil {
		creteErr = err
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, creteErr
	}
	LoggingInfo(hiscallInfo, start)
	return securityInfo, creteErr
}

func (securityHandler *OpenStackSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(securityHandler.Client.IdentityEndpoint, call.SECURITYGROUP, SecurityGroup, "ListSecurity()")

	// 보안그룹 목록 조회
	start := call.Start()
	pager, err := secgroups.List(securityHandler.Client).AllPages()
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	security, err := secgroups.ExtractSecurityGroups(pager)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	// 보안그룹 목록 정보 매핑
	var securityList []*irs.SecurityInfo
	securityList = make([]*irs.SecurityInfo, len(security))
	for i, v := range security {
		securityList[i] = securityHandler.setterSeg(v)
	}
	return securityList, nil
}

func (securityHandler *OpenStackSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(securityHandler.Client.IdentityEndpoint, call.SECURITYGROUP, securityIID.NameId, "GetSecurity()")

	start := call.Start()
	securityGroup, err := securityHandler.getRawSecurity(securityIID)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	securityInfo := securityHandler.setterSeg(*securityGroup)
	return *securityInfo, nil
}

func (securityHandler *OpenStackSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(securityHandler.Client.IdentityEndpoint, call.SECURITYGROUP, securityIID.NameId, "DeleteSecurity()")

	start := call.Start()
	result := secgroups.Delete(securityHandler.Client, securityIID.SystemId)
	if result.Err != nil {
		cblogger.Error(result.Err.Error())
		LoggingError(hiscallInfo, result.Err)
		return false, result.Err
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (securityHandler *OpenStackSecurityHandler) AddRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	hiscallInfo := GetCallLogScheme(securityHandler.Client.IdentityEndpoint, call.SECURITYGROUP, sgIID.NameId, "AddRules()")

	start := call.Start()
	securityGroup, err := securityHandler.getRawSecurity(sgIID)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}

	securityInfo := securityHandler.setterSeg(*securityGroup)

	var updateRules []irs.SecurityRuleInfo
	for _, newRule := range *securityRules {
		chk := true
		for _, baseRule := range *securityInfo.SecurityRules {
			if equalsRule(newRule, baseRule) {
				chk = false
				break
			}
		}
		if chk {
			updateRules = append(updateRules, newRule)
		}
	}
	createRuleOpts, err := convertCBRuleInfosToOpenStackRules(securityGroup.ID, &updateRules)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	for _, createRuleOpt := range *createRuleOpts {
		_, err := rules.Create(securityHandler.NetworkClient, createRuleOpt).Extract()
		if err != nil {
			cblogger.Error(err.Error())
			LoggingError(hiscallInfo, err)
			return irs.SecurityInfo{}, err
		}
	}

	//  SecurityGroup 정보 리턴
	updatedSecurity, err := securityHandler.getRawSecurity(sgIID)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	updatedSecurityInfo := securityHandler.setterSeg(*updatedSecurity)

	return *updatedSecurityInfo, nil
}

func (securityHandler *OpenStackSecurityHandler) RemoveRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (bool, error) {
	hiscallInfo := GetCallLogScheme(securityHandler.Client.IdentityEndpoint, call.SECURITYGROUP, sgIID.NameId, "AddRules()")

	start := call.Start()

	listOpts := rules.ListOpts{
		SecGroupID: sgIID.SystemId,
	}
	pager, err := rules.List(securityHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		return false, err
	}
	secList, err := rules.ExtractRules(pager)
	if err != nil {
		return false, err
	}
	var deleteRuleIds []string
	ruleWithIds, err := getRuleInfoWithIds(&secList)
	if err != nil {
		return false, err
	}
	for _, newRule := range *securityRules {
		for _, baseRuleWithId := range *ruleWithIds {
			if equalsRule(newRule, baseRuleWithId.RuleInfo) {
				deleteRuleIds = append(deleteRuleIds, baseRuleWithId.Id)
				break
			}
		}
	}
	for _, deleteRuleId := range deleteRuleIds {
		err := rules.Delete(securityHandler.NetworkClient, deleteRuleId).ExtractErr()
		if err != nil {
			return false, err
		}
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}
func convertRuleProtocolOPToCB(protocol string) string {
	switch strings.ToUpper(protocol) {
	case "":
		return "all"
	default:
		return strings.ToLower(protocol)
	}
}

func convertRuleProtocolCBToOP(protocol string) (string, error) {
	switch strings.ToUpper(protocol) {
	case "ALL":
		return "", nil
	case "ICMP", "TCP", "UDP":
		return strings.ToLower(protocol), nil
	}
	return "", errors.New("invalid Rule Protocol. The rule protocol of OpenStack must be specified accurately tcp, udp, icmp")
}

func convertRulePortRangeOPToCB(min int, max int, protocol string) (from string, to string) {
	if strings.ToLower(protocol) == ICMP {
		return "-1", "-1"
	} else {
		if min == 0 && max == 0 {
			return "1", "65535"
		}
		return strconv.Itoa(min), strconv.Itoa(max)
	}
}

func convertRulePortRangeCBToOP(from string, to string) (min int, max int, err error) {
	if from == "" || to == "" {
		return 0, 0, errors.New("invalid Rule PortRange")
	}
	fromInt, err := strconv.Atoi(from)
	if err != nil {
		return 0, 0, errors.New("invalid Rule PortRange")
	}
	toInt, err := strconv.Atoi(to)
	if err != nil {
		return 0, 0, errors.New("invalid Rule PortRange")
	}
	if fromInt == -1 || toInt == -1 {
		return 1, 65535, nil
	}
	if fromInt > 65535 || fromInt < -1 || toInt > 65535 || toInt < -1 {
		return 0, 0, errors.New("invalid Rule PortRange")
	}
	if fromInt == toInt {
		return fromInt, fromInt, nil
	} else {
		return fromInt, toInt, nil
	}
}

func equalsRule(pre irs.SecurityRuleInfo, post irs.SecurityRuleInfo) bool {
	if pre.ToPort == "-1" || pre.FromPort == "-1" {
		pre.FromPort = "1"
		pre.ToPort = "65535"
	}
	if post.ToPort == "-1" || post.FromPort == "-1" {
		post.FromPort = "1"
		post.ToPort = "65535"
	}
	return strings.ToLower(fmt.Sprintf("%#v", pre)) == strings.ToLower(fmt.Sprintf("%#v", post))
}

func convertCBRuleInfosToOpenStackRules(sgId string, sgRules *[]irs.SecurityRuleInfo) (*[]rules.CreateOpts, error) {
	openStackRuleCreateOpts := make([]rules.CreateOpts, len(*sgRules))
	for i, rule := range *sgRules {
		var direction string
		if strings.EqualFold(strings.ToLower(rule.Direction), Inbound) {
			direction = string(rules.DirIngress)
		} else {
			direction = string(rules.DirEgress)
		}

		var createRuleOpts rules.CreateOpts
		protocol, err := convertRuleProtocolCBToOP(rule.IPProtocol)
		if err != nil {
			return nil, err
		}
		if strings.ToLower(rule.IPProtocol) == ICMP {
			createRuleOpts = rules.CreateOpts{
				Direction:      rules.RuleDirection(direction),
				EtherType:      rules.EtherType4,
				SecGroupID:     sgId,
				Protocol:       rules.RuleProtocol(protocol),
				RemoteIPPrefix: rule.CIDR,
			}
		} else {
			min, max, err := convertRulePortRangeCBToOP(rule.FromPort, rule.ToPort)
			if err != nil {
				return nil, err
			}
			createRuleOpts = rules.CreateOpts{
				Direction:      rules.RuleDirection(direction),
				EtherType:      rules.EtherType4,
				SecGroupID:     sgId,
				PortRangeMin:   min,
				PortRangeMax:   max,
				Protocol:       rules.RuleProtocol(protocol),
				RemoteIPPrefix: rule.CIDR,
			}
		}
		openStackRuleCreateOpts[i] = createRuleOpts
	}
	return &openStackRuleCreateOpts, nil
}

type securityRuleInfoWithId struct {
	Id       string
	RuleInfo irs.SecurityRuleInfo
}

func convertOpenStackRuleToCBRuleInfo(rawRules *rules.SecGroupRule) irs.SecurityRuleInfo {
	var direction string
	if strings.EqualFold(rawRules.Direction, string(rules.DirIngress)) {
		direction = Inbound
	} else {
		direction = Outbound
	}
	cidr := rawRules.RemoteIPPrefix
	if cidr == "" {
		cidr = "0.0.0.0/0"
	}
	ruleInfo := irs.SecurityRuleInfo{
		Direction:  direction,
		IPProtocol: convertRuleProtocolOPToCB(rawRules.Protocol),
		CIDR:       cidr,
	}

	if strings.ToLower(rawRules.Protocol) == ICMP {
		ruleInfo.FromPort = "-1"
		ruleInfo.ToPort = "-1"
	} else {
		min, max := convertRulePortRangeOPToCB(rawRules.PortRangeMin, rawRules.PortRangeMax, rawRules.Protocol)
		ruleInfo.FromPort = min
		ruleInfo.ToPort = max
	}

	return ruleInfo
}

func getRuleInfoWithIds(rawRules *[]rules.SecGroupRule) (*[]securityRuleInfoWithId, error) {
	var secRuleArrIds []securityRuleInfoWithId
	for _, rawRule := range *rawRules {
		if rawRule.EtherType == string(rules.EtherType4) {
			ruleInfo := convertOpenStackRuleToCBRuleInfo(&rawRule)
			secRuleArrIds = append(secRuleArrIds, securityRuleInfoWithId{
				Id:       rawRule.ID,
				RuleInfo: ruleInfo,
			})
		}
	}
	return &secRuleArrIds, nil
}


func (securityHandler *OpenStackSecurityHandler) getRawSecurity(securityIID irs.IID) (*secgroups.SecurityGroup, error) {
	if securityIID.SystemId == "" && securityIID.NameId == "" {
		return nil, errors.New("invalid IID")
	}
	if securityIID.SystemId != ""{
		return secgroups.Get(securityHandler.Client, securityIID.SystemId).Extract()
	} else {
		pager, err := secgroups.List(securityHandler.Client).AllPages()
		if err != nil {
			return nil, err
		}
		rawSecurityGroups, err := secgroups.ExtractSecurityGroups(pager)
		for _, rawSeg := range rawSecurityGroups {
			if  securityIID.NameId == rawSeg.Name {
				return &rawSeg, nil
			}
		}
		return nil, errors.New("not found SecurityGroup")
	}
}

