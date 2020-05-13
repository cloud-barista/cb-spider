package resources

import (
	"errors"
	"fmt"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/rackspace/gophercloud/openstack/networking/v2/extensions/security/rules"
	"strconv"
	"strings"
)

const (
	Inbound  = "inbound"
	Outbound = "outbound"
	ICMP     = "icmp"
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
		if strings.EqualFold(rule.Direction, rules.DirIngress) {
			direction = Inbound
		} else {
			direction = Outbound
		}

		ruleInfo := irs.SecurityRuleInfo{
			Direction:  direction,
			IPProtocol: rule.Protocol,
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
	// Check SecurityGroup Exists
	secGroupList, err := securityHandler.ListSecurity()
	if err != nil {
		return irs.SecurityInfo{}, err
	}
	for _, sg := range secGroupList {
		if sg.IId.NameId == securityReqInfo.IId.NameId {
			errMsg := fmt.Sprintf("Security Group with name %s already exist", securityReqInfo.IId.NameId)
			createErr := errors.New(errMsg)
			return irs.SecurityInfo{}, createErr
		}
	}

	// Create SecurityGroup
	createOpts := secgroups.CreateOpts{
		Name:        securityReqInfo.IId.NameId,
		Description: securityReqInfo.IId.NameId,
	}
	group, err := secgroups.Create(securityHandler.Client, createOpts).Extract()
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	// Create SecurityGroup Rules
	for _, rule := range *securityReqInfo.SecurityRules {

		var direction string
		if strings.EqualFold(strings.ToLower(rule.Direction), Inbound) {
			direction = rules.DirIngress
		} else {
			direction = rules.DirEgress
		}

		var createRuleOpts rules.CreateOpts

		if strings.ToLower(rule.IPProtocol) == ICMP {
			createRuleOpts = rules.CreateOpts{
				Direction:      direction,
				EtherType:      rules.Ether4,
				SecGroupID:     group.ID,
				Protocol:       strings.ToLower(rule.IPProtocol),
				RemoteIPPrefix: "0.0.0.0/0",
			}
		} else {
			fromPort, _ := strconv.Atoi(rule.FromPort)
			toPort, _ := strconv.Atoi(rule.ToPort)
			createRuleOpts = rules.CreateOpts{
				Direction:      direction,
				EtherType:      rules.Ether4,
				SecGroupID:     group.ID,
				PortRangeMin:   fromPort,
				PortRangeMax:   toPort,
				Protocol:       strings.ToLower(rule.IPProtocol),
				RemoteIPPrefix: "0.0.0.0/0",
			}
		}

		_, err := rules.Create(securityHandler.NetworkClient, createRuleOpts).Extract()
		if err != nil {
			return irs.SecurityInfo{}, err
		}
	}

	// 생성된 SecurityGroup 정보 리턴
	securityInfo, err := securityHandler.GetSecurity(irs.IID{SystemId: group.ID})
	if err != nil {
		return irs.SecurityInfo{}, err
	}
	return securityInfo, nil
}

func (securityHandler *OpenStackSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {

	// 보안그룹 목록 조회
	pager, err := secgroups.List(securityHandler.Client).AllPages()
	if err != nil {
		return nil, err
	}
	security, err := secgroups.ExtractSecurityGroups(pager)
	if err != nil {
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
	securityGroup, err := secgroups.Get(securityHandler.Client, securityIID.SystemId).Extract()
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	securityInfo := securityHandler.setterSeg(*securityGroup)
	return *securityInfo, nil
}

func (securityHandler *OpenStackSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	result := secgroups.Delete(securityHandler.Client, securityIID.SystemId)
	if result.Err != nil {
		return false, result.Err
	}
	return true, nil
}
