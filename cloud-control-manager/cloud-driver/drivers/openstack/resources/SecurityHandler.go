package resources

import (
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/rackspace/gophercloud/pagination"
)

type OpenStackSecurityHandler struct {
	Client *gophercloud.ServiceClient
}

// @TODO: SecurityInfo 리소스 프로퍼티 정의 필요
type SecurityInfo struct {
	ID          string
	Name        string
	Description string
	Rules       []SecurityRuleInfo
	TenantID    string
}

type SecurityRuleInfo struct {
	ID            string
	FromPort      int
	ToPort        int
	IPProtocol    string
	ParentGroupID string
	CIDR          string
	Group         Group
}

type Group struct {
	TenantId string
	Name     string
}

func (securityInfo *SecurityInfo) setter(results secgroups.SecurityGroup) *SecurityInfo {
	securityInfo.ID = results.ID
	securityInfo.Name = results.Name
	securityInfo.Description = results.Description

	var securityRuleArr []SecurityRuleInfo

	for _, sgRule := range results.Rules {
		ruleInfo := SecurityRuleInfo{
			ID:            sgRule.ID,
			FromPort:      sgRule.FromPort,
			ToPort:        sgRule.ToPort,
			IPProtocol:    sgRule.IPProtocol,
			CIDR:          sgRule.IPRange.CIDR,
			ParentGroupID: sgRule.ParentGroupID,
			Group: Group{
				TenantId: sgRule.Group.TenantID,
				Name:     sgRule.Group.Name,
			},
		}

		securityRuleArr = append(securityRuleArr, ruleInfo)
	}

	securityInfo.Rules = securityRuleArr
	securityInfo.TenantID = results.TenantID

	return securityInfo
}

func (securityHandler *OpenStackSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {

	// @TODO: SecurityGroup 생성 요청 파라미터 정의 필요
	type SecurityRuleReqInfo struct {
		ParentGroupID string
		FromPort      int
		ToPort        int
		IPProtocol    string
		CIDR          string // 방식 1) 특정 CIDR 범위의 IP 기준으로 룰 적용
		FromGroupID   string // 방식 2) SecurityGroup 기준으로 룰 적용
	}
	type SecurityReqInfo struct {
		Name          string
		Description   string
		SecurityRules *[]SecurityRuleReqInfo
	}

	reqInfo := SecurityReqInfo{
		Name:        securityReqInfo.Name,
		Description: "Temp securityGroup for test",
	}

	// Create SecurityGroup
	createOpts := secgroups.CreateOpts{
		Name:        reqInfo.Name,
		Description: reqInfo.Description,
	}
	group, err := secgroups.Create(securityHandler.Client, createOpts).Extract()
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	reqInfo.SecurityRules = &[]SecurityRuleReqInfo{
		{
			//ParentGroupID: group.ID,
			FromPort:   22,
			ToPort:     22,
			IPProtocol: "TCP",
			CIDR:       "0.0.0.0/0", // 방식 1) CIDR 기준 룰 적용
		},
		{
			//ParentGroupID: group.ID,
			FromPort:    3306,
			ToPort:      3306,
			IPProtocol:  "TCP",
			FromGroupID: group.ID, // 방식 2) 보안그룹 기준 룰 적용
		},
		{
			//ParentGroupID: group.ID,
			FromPort:   -1,
			ToPort:     -1,
			IPProtocol: "ICMP",
			CIDR:       "0.0.0.0/0",
		},
	}

	// Create SecurityGroup Rules
	for _, rule := range *reqInfo.SecurityRules {
		createRuleOpts := secgroups.CreateRuleOpts{
			ParentGroupID: group.ID,
			FromPort:      rule.FromPort,
			ToPort:        rule.ToPort,
			IPProtocol:    rule.IPProtocol,
		}

		if rule.CIDR != "" {
			createRuleOpts.CIDR = rule.CIDR
		} else {
			createRuleOpts.FromGroupID = group.ID
		}

		_, err := secgroups.CreateRule(securityHandler.Client, createRuleOpts).Extract()
		if err != nil {
			return irs.SecurityInfo{}, err
		}
	}

	securityInfo, err := securityHandler.GetSecurity(group.ID)
	if err != nil {
		return irs.SecurityInfo{}, nil
	}

	spew.Dump(securityInfo)
	return irs.SecurityInfo{Id: group.ID, Name: group.Name}, nil
}

func (securityHandler *OpenStackSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	var securityList []*SecurityInfo

	pager := secgroups.List(securityHandler.Client)
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		// Get SecurityGroup
		list, err := secgroups.ExtractSecurityGroups(page)
		if err != nil {
			return false, err
		}
		// Add to List
		for _, s := range list {
			securityInfo := new(SecurityInfo).setter(s)
			securityList = append(securityList, securityInfo)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}

	spew.Dump(securityList)
	return nil, nil
}

func (securityHandler *OpenStackSecurityHandler) GetSecurity(securityID string) (irs.SecurityInfo, error) {
	securityGroup, err := secgroups.Get(securityHandler.Client, securityID).Extract()
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	securityInfo := new(SecurityInfo).setter(*securityGroup)

	spew.Dump(securityInfo)
	return irs.SecurityInfo{}, nil
}

func (securityHandler *OpenStackSecurityHandler) DeleteSecurity(securityID string) (bool, error) {
	result := secgroups.Delete(securityHandler.Client, securityID)
	if result.Err != nil {
		return false, result.Err
	}
	return true, nil
}
