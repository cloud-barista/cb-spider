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
	secRuleList := make([]irs.SecurityRuleInfo, len(secList))
	for i, rule := range secList {
		var direction string
		if strings.EqualFold(rule.Direction, string(rules.DirIngress)) {
			direction = Inbound
		} else {
			direction = Outbound
		}

		ruleInfo := irs.SecurityRuleInfo{
			Direction:  direction,
			IPProtocol: strings.ToLower(rule.Protocol),
		}

		if strings.ToLower(rule.Protocol) == ICMP {
			ruleInfo.FromPort = "-1"
			ruleInfo.ToPort = "-1"
		} else {
			ruleInfo.FromPort = strconv.Itoa(rule.PortRangeMin)
			ruleInfo.ToPort = strconv.Itoa(rule.PortRangeMax)
		}

		secRuleList[i] = ruleInfo
	}
	secInfo.SecurityRules = &secRuleList

	return secInfo
}

func (securityHandler *OpenStackSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(securityHandler.Client.IdentityEndpoint, call.SECURITYGROUP, securityReqInfo.IId.NameId, "CreateSecurity()")

	// Check SecurityGroup Exists
	secGroupList, err := securityHandler.ListSecurity()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	for _, sg := range secGroupList {
		if sg.IId.NameId == securityReqInfo.IId.NameId {
			createErr := errors.New(fmt.Sprintf("Security Group with name %s already exist", securityReqInfo.IId.NameId))
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
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	// Create SecurityGroup Rules
	for _, rule := range *securityReqInfo.SecurityRules {

		var direction string
		if strings.EqualFold(strings.ToLower(rule.Direction), Inbound) {
			direction = string(rules.DirIngress)
		} else {
			direction = string(rules.DirEgress)
		}

		var createRuleOpts rules.CreateOpts

		if strings.ToLower(rule.IPProtocol) == ICMP {
			createRuleOpts = rules.CreateOpts{
				Direction:      rules.RuleDirection(direction),
				EtherType:      rules.EtherType4,
				SecGroupID:     group.ID,
				Protocol:       rules.RuleProtocol(strings.ToLower(rule.IPProtocol)),
				RemoteIPPrefix: "0.0.0.0/0",
			}
		} else {
			fromPort, _ := strconv.Atoi(rule.FromPort)
			toPort, _ := strconv.Atoi(rule.ToPort)
			createRuleOpts = rules.CreateOpts{
				Direction:      rules.RuleDirection(direction),
				EtherType:      rules.EtherType4,
				SecGroupID:     group.ID,
				PortRangeMin:   fromPort,
				PortRangeMax:   toPort,
				Protocol:       rules.RuleProtocol(strings.ToLower(rule.IPProtocol)),
				RemoteIPPrefix: "0.0.0.0/0",
			}
		}

		_, err := rules.Create(securityHandler.NetworkClient, createRuleOpts).Extract()
		if err != nil {
			LoggingError(hiscallInfo, err)
			return irs.SecurityInfo{}, err
		}
	}

	// 생성된 SecurityGroup 정보 리턴
	securityInfo, err := securityHandler.GetSecurity(irs.IID{SystemId: group.ID})
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	return securityInfo, nil
}

func (securityHandler *OpenStackSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(securityHandler.Client.IdentityEndpoint, call.SECURITYGROUP, SecurityGroup, "ListSecurity()")

	// 보안그룹 목록 조회
	start := call.Start()
	pager, err := secgroups.List(securityHandler.Client).AllPages()
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	security, err := secgroups.ExtractSecurityGroups(pager)
	if err != nil {
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
	securityGroup, err := secgroups.Get(securityHandler.Client, securityIID.SystemId).Extract()
	if err != nil {
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
		LoggingError(hiscallInfo, result.Err)
		return false, result.Err
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}
