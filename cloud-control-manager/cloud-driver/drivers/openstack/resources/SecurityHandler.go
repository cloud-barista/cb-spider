package resources

import (
	"errors"
	"fmt"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/secgroups"
	"strconv"
	"strings"
)

type OpenStackSecurityHandler struct {
	Client *gophercloud.ServiceClient
}

func setterSeg(secGroup secgroups.SecurityGroup) *irs.SecurityInfo {
	secInfo := &irs.SecurityInfo{
		Id:   secGroup.ID,
		Name: secGroup.Name,
	}

	// 보안그룹 룰 정보 등록
	secRuleList := []irs.SecurityRuleInfo{}
	for _, rule := range secGroup.Rules {
		ruleInfo := irs.SecurityRuleInfo{
			FromPort:   strconv.Itoa(rule.FromPort),
			ToPort:     strconv.Itoa(rule.ToPort),
			IPProtocol: rule.IPProtocol,
		}
		secRuleList = append(secRuleList, ruleInfo)
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
		if sg.Name == securityReqInfo.Name {
			errMsg := fmt.Sprintf("Security Group with name %s already exist", securityReqInfo.Name)
			createErr := errors.New(errMsg)
			return irs.SecurityInfo{}, createErr
		}
	}

	// Create SecurityGroup
	createOpts := secgroups.CreateOpts{
		Name:        securityReqInfo.Name,
		Description: securityReqInfo.Name,
	}
	group, err := secgroups.Create(securityHandler.Client, createOpts).Extract()
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	// Create SecurityGroup Rules
	for _, rule := range *securityReqInfo.SecurityRules {
		fromPort, _ := strconv.Atoi(rule.FromPort)
		toPort, _ := strconv.Atoi(rule.ToPort)

		createRuleOpts := secgroups.CreateRuleOpts{
			FromPort:      fromPort,
			ToPort:        toPort,
			IPProtocol:    rule.IPProtocol,
			CIDR:          "0.0.0.0/0",
			ParentGroupID: group.ID,
		}

		_, err := secgroups.CreateRule(securityHandler.Client, createRuleOpts).Extract()
		if err != nil {
			return irs.SecurityInfo{}, err
		}
	}

	// 생성된 SecurityGroup 정보 리턴
	securityInfo, err := securityHandler.getSecurityById(group.ID)
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
		securityList[i] = setterSeg(v)
	}
	return securityList, nil
}

func (securityHandler *OpenStackSecurityHandler) GetSecurity(securityNameId string) (irs.SecurityInfo, error) {
	securityId, err := securityHandler.getSecurityIdByName(securityNameId)
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	securityGroup, err := securityHandler.getSecurityById(securityId)
	if err != nil {
		return irs.SecurityInfo{}, err
	}
	return securityGroup, nil
}

func (securityHandler *OpenStackSecurityHandler) DeleteSecurity(securityNameId string) (bool, error) {
	securityId, err := securityHandler.getSecurityIdByName(securityNameId)
	if err != nil {
		return false, err
	}

	result := secgroups.Delete(securityHandler.Client, securityId)
	if result.Err != nil {
		return false, result.Err
	}
	return true, nil
}

func (securityHandler *OpenStackSecurityHandler) getSecurityById(securityID string) (irs.SecurityInfo, error) {
	securityGroup, err := secgroups.Get(securityHandler.Client, securityID).Extract()
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	securityInfo := setterSeg(*securityGroup)
	return *securityInfo, nil
}

func (securityHandler *OpenStackSecurityHandler) getSecurityIdByName(securityName string) (string, error) {
	var securityId string

	securityList, err := securityHandler.ListSecurity()
	if err != nil {
		return "", err
	}
	for _, s := range securityList {
		if strings.EqualFold(s.Name, securityName) {
			securityId = s.Id
			break
		}
	}

	return securityId, nil
}
