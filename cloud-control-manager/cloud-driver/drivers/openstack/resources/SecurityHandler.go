package resources

import (
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"
	_ "github.com/davecgh/go-spew/spew"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/secgroups"
	"github.com/rackspace/gophercloud/pagination"
	"strconv"
)

type OpenStackSecurityHandler struct {
	Client *gophercloud.ServiceClient
}

// @TODO: SecurityInfo 리소스 프로퍼티 정의 필요
/*type SecurityInfo struct {
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
}*/

/*func (securityInfo *SecurityInfo) setter(results secgroups.SecurityGroup) *SecurityInfo {
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
*/

func setterSeg(secGroup secgroups.SecurityGroup) *irs.SecurityInfo {
	secInfo := &irs.SecurityInfo{
		Id:   secGroup.ID,
		Name: secGroup.Name,
	}

	// 보안그룹 룰 정보 등록
	secRuleList := make([]irs.SecurityRuleInfo, len(secGroup.Rules))
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

	// @TODO: SecurityGroup 생성 요청 파라미터 정의 필요
	/*type SecurityRuleReqInfo struct {
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
	}*/

	// Create SecurityGroup
	createOpts := secgroups.CreateOpts{
		Name: securityReqInfo.Name,
	}
	group, err := secgroups.Create(securityHandler.Client, createOpts).Extract()
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	/*reqInfo.SecurityRules = &[]SecurityRuleReqInfo{
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
	}*/

	// Create SecurityGroup Rules
	for _, rule := range *securityReqInfo.SecurityRules {
		fromPort, _ := strconv.Atoi(rule.FromPort)
		toPort, _ := strconv.Atoi(rule.ToPort)

		createRuleOpts := secgroups.CreateRuleOpts{
			FromPort:   fromPort,
			ToPort:     toPort,
			IPProtocol: rule.IPProtocol,
			CIDR:       "0.0.0.0/0",
		}

		_, err := secgroups.CreateRule(securityHandler.Client, createRuleOpts).Extract()
		if err != nil {
			return irs.SecurityInfo{}, err
		}
	}

	// 생성된 SecurityGroup
	securityInfo, err := securityHandler.GetSecurity(group.ID)
	if err != nil {
		return irs.SecurityInfo{}, err
	}
	return securityInfo, nil
}

func (securityHandler *OpenStackSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	var securityList []*irs.SecurityInfo

	pager := secgroups.List(securityHandler.Client)
	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		// Get SecurityGroup
		list, err := secgroups.ExtractSecurityGroups(page)
		if err != nil {
			return false, err
		}
		// Add to List
		for _, s := range list {
			securityInfo := setterSeg(s)
			securityList = append(securityList, securityInfo)
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	//spew.Dump(securityList)
	return securityList, nil
}

func (securityHandler *OpenStackSecurityHandler) GetSecurity(securityID string) (irs.SecurityInfo, error) {
	securityGroup, err := secgroups.Get(securityHandler.Client, securityID).Extract()
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	securityInfo := setterSeg(*securityGroup)
	//spew.Dump(securityInfo)
	return *securityInfo, nil
}

func (securityHandler *OpenStackSecurityHandler) DeleteSecurity(securityID string) (bool, error) {
	result := secgroups.Delete(securityHandler.Client, securityID)
	if result.Err != nil {
		return false, result.Err
	}
	return true, nil
}
