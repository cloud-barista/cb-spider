package resources

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	idrv "../../../interfaces"
	nirs "../../../interfaces/new-resources"
	irs "../../../interfaces/resources"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/davecgh/go-spew/spew"
	compute "google.golang.org/api/compute/v1"
)

type GCPSecurityHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

// @TODO: SecurityInfo 리소스 프로퍼티 정의 필요
type SecurityInfo struct {
	Id                   string
	Name                 string
	Location             string
	SecurityRules        []SecurityRuleInfo
	DefaultSecurityRules []SecurityRuleInfo
}

type SecurityRuleInfo struct {
	Name                     string
	SourceAddressPrefix      string
	SourcePortRange          string
	DestinationAddressPrefix string
	DestinationPortRange     string
	Protocol                 string
	Access                   string
	Priority                 int32
	Direction                string
}

func (security *SecurityInfo) setter(securityGroup network.SecurityGroup) *SecurityInfo {
	security.Id = *securityGroup.ID
	security.Name = *securityGroup.Name
	security.Location = *securityGroup.Location
	var securityRuleArr []SecurityRuleInfo
	var defaultSecurityRuleArr []SecurityRuleInfo

	for _, sgRule := range *securityGroup.SecurityRules {
		ruleInfo := SecurityRuleInfo{
			Name:                     *sgRule.Name,
			SourceAddressPrefix:      *sgRule.SourceAddressPrefix,
			SourcePortRange:          *sgRule.SourcePortRange,
			DestinationAddressPrefix: *sgRule.DestinationAddressPrefix,
			DestinationPortRange:     *sgRule.DestinationPortRange,
			Protocol:                 fmt.Sprint(sgRule.Protocol),
			Access:                   fmt.Sprint(sgRule.Access),
			Priority:                 *sgRule.Priority,
			Direction:                fmt.Sprint(sgRule.Direction),
		}
		//fmt.Println(ruleInfo)
		securityRuleArr = append(securityRuleArr, ruleInfo)
	}

	for _, sgRule := range *securityGroup.DefaultSecurityRules {
		ruleInfo := SecurityRuleInfo{
			Name:                     *sgRule.Name,
			SourceAddressPrefix:      *sgRule.SourceAddressPrefix,
			SourcePortRange:          *sgRule.SourcePortRange,
			DestinationAddressPrefix: *sgRule.DestinationAddressPrefix,
			DestinationPortRange:     *sgRule.DestinationPortRange,
			Protocol:                 fmt.Sprint(sgRule.Protocol),
			Access:                   fmt.Sprint(sgRule.Access),
			Priority:                 *sgRule.Priority,
			Direction:                fmt.Sprint(sgRule.Direction),
		}
		//fmt.Println(ruleInfo)
		defaultSecurityRuleArr = append(defaultSecurityRuleArr, ruleInfo)
	}

	security.SecurityRules = securityRuleArr
	security.DefaultSecurityRules = defaultSecurityRuleArr

	return security
}

func (securityHandler *GCPSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {

	// @TODO: SecurityGroup 생성 요청 파라미터 정의 필요
	type SecurityReqInfo struct {
		SecurityRules *[]SecurityRuleInfo
	}

	reqInfo := SecurityReqInfo{
		SecurityRules: &[]SecurityRuleInfo{
			{
				Name:                     "HTTP",
				SourceAddressPrefix:      "*",
				SourcePortRange:          "*",
				DestinationAddressPrefix: "*",
				DestinationPortRange:     "80",
				Protocol:                 "TCP",
				Access:                   "Allow",
				Priority:                 300,
				Direction:                "Inbound",
			},
			{
				Name:                     "SSH",
				SourceAddressPrefix:      "*",
				SourcePortRange:          "*",
				DestinationAddressPrefix: "*",
				DestinationPortRange:     "22",
				Protocol:                 "TCP",
				Access:                   "Allow",
				Priority:                 320,
				Direction:                "Inbound",
			},
		},
	}

	var sgRuleList []network.SecurityRule
	for _, rule := range *reqInfo.SecurityRules {
		sgRuleInfo := network.SecurityRule{
			Name: to.StringPtr(rule.Name),
			SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
				SourceAddressPrefix:      to.StringPtr(rule.SourceAddressPrefix),
				SourcePortRange:          to.StringPtr(rule.SourcePortRange),
				DestinationAddressPrefix: to.StringPtr(rule.DestinationAddressPrefix),
				DestinationPortRange:     to.StringPtr(rule.DestinationPortRange),
				Protocol:                 network.SecurityRuleProtocol(rule.Protocol),
				Access:                   network.SecurityRuleAccess(rule.Access),
				Priority:                 to.Int32Ptr(rule.Priority),
				Direction:                network.SecurityRuleDirection(rule.Direction),
			},
		}
		sgRuleList = append(sgRuleList, sgRuleInfo)
	}

	createOpts := network.SecurityGroup{
		SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
			SecurityRules: &sgRuleList,
		},
		Location: &securityHandler.Region.Region,
	}

	securityIdArr := strings.Split(securityReqInfo.Id, ":")

	// Check SecurityGroup Exists
	security, err := securityHandler.Client.Get(securityHandler.Ctx, securityIdArr[0], securityIdArr[1], "")
	if security.ID != nil {
		errMsg := fmt.Sprintf("Security Group with name %s already exist", securityIdArr[1])
		createErr := errors.New(errMsg)
		return irs.SecurityInfo{}, createErr
	}

	future, err := securityHandler.Client.CreateOrUpdate(securityHandler.Ctx, securityIdArr[0], securityIdArr[1], createOpts)
	if err != nil {
		log.Fatal(err)
		return irs.SecurityInfo{}, err
	}
	err = future.WaitForCompletionRef(securityHandler.Ctx, securityHandler.Client.Client)
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	// @TODO: 생성된 SecurityGroup 정보 리턴
	publicIPInfo, err := securityHandler.GetSecurity(securityReqInfo.Id)
	if err != nil {
		return irs.SecurityInfo{}, err
	}
	return publicIPInfo, nil
}

func (securityHandler *GCPSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	//result, err := securityHandler.Client.ListAll(securityHandler.Ctx)
	result, err := securityHandler.Client.List(securityHandler.Ctx, securityHandler.Region.ResourceGroup)
	if err != nil {
		return nil, err
	}

	var securityList []*SecurityInfo
	for _, security := range result.Values() {
		securityInfo := new(SecurityInfo).setter(security)
		securityList = append(securityList, securityInfo)
	}

	spew.Dump(securityList)
	return nil, nil
}

func (securityHandler *GCPSecurityHandler) GetSecurity(securityID string) (irs.SecurityInfo, error) {
	securityIdArr := strings.Split(securityID, ":")
	security, err := securityHandler.Client.Get(securityHandler.Ctx, securityIdArr[0], securityIdArr[1], "")
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	securityInfo := new(SecurityInfo).setter(security)

	spew.Dump(securityInfo)
	return nirs.SecurityInfo{}, nil
}

func (securityHandler *GCPSecurityHandler) DeleteSecurity(securityID string) (bool, error) {
	securityIDArr := strings.Split(securityID, ":")
	future, err := securityHandler.Client.Delete(securityHandler.Ctx, securityIDArr[0], securityIDArr[1])
	if err != nil {
		return false, err
	}
	err = future.WaitForCompletionRef(securityHandler.Ctx, securityHandler.Client.Client)
	if err != nil {
		return false, err
	}
	return true, nil
}
