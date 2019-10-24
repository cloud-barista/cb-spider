package resources

import (
	"errors"
	"fmt"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/iam/securitygroup"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"strings"
)

type ClouditSecurityHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func setterSecGroup(secGroup securitygroup.SecurityGroupInfo) *irs.SecurityInfo {
	secInfo := &irs.SecurityInfo{
		Id:            secGroup.ID,
		Name:          secGroup.Name,
		SecurityRules: nil,
	}

	var secRuleArr []irs.SecurityRuleInfo
	for _, sgRule := range secGroup.Rules {
		secRuleInfo := irs.SecurityRuleInfo{
			FromPort:   sgRule.Port,
			ToPort:     sgRule.Port,
			IPProtocol: sgRule.Protocol,
			Direction:  sgRule.Type,
		}
		secRuleArr = append(secRuleArr, secRuleInfo)
	}
	secInfo.SecurityRules = &secRuleArr

	return secInfo
}

func (securityHandler *ClouditSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	// Check SecurityGroup Exists
	if securityId, err := securityHandler.CheckSecurityExist(securityReqInfo.Name); err != nil {
		return irs.SecurityInfo{}, err
	} else {
		if *securityId != "" {
			errMsg := fmt.Sprintf("Security Group with name %s already exist", securityReqInfo.Name)
			createErr := errors.New(errMsg)
			return irs.SecurityInfo{}, createErr
		}
	}

	securityHandler.Client.TokenID = securityHandler.CredentialInfo.AuthToken
	authHeader := securityHandler.Client.AuthenticatedHeaders()

	reqInfo := securitygroup.SecurityReqInfo{
		Name: securityReqInfo.Name,
	}

	// SecurityGroup Rule 설정
	ruleList := []securitygroup.SecurityGroupRules{}
	for idx, rule := range *securityReqInfo.SecurityRules {
		secRuleInfo := securitygroup.SecurityGroupRules{
			Name:     fmt.Sprintf("%s-rules-%d", securityReqInfo.Name, idx+1),
			Type:     rule.Direction,
			Port:     rule.ToPort,
			Target:   "0.0.0.0/0",
			Protocol: strings.ToLower(rule.IPProtocol),
		}
		ruleList = append(ruleList, secRuleInfo)
	}
	reqInfo.Rules = ruleList

	createOpts := client.RequestOpts{
		JSONBody:    reqInfo,
		MoreHeaders: authHeader,
	}

	if securityGroup, err := securitygroup.Create(securityHandler.Client, &createOpts); err != nil {
		return irs.SecurityInfo{}, err
	} else {
		//spew.Dump(securityGroup)
		secGroupInfo := setterSecGroup(*securityGroup)
		return *secGroupInfo, nil
	}
}

func (securityHandler *ClouditSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	securityHandler.Client.TokenID = securityHandler.CredentialInfo.AuthToken
	authHeader := securityHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if securityList, err := securitygroup.List(securityHandler.Client, &requestOpts); err != nil {
		return nil, err
	} else {
		// SecurityGroup Rule 정보 가져오기
		for i, sg := range *securityList {
			if sgRules, err := securitygroup.ListRule(securityHandler.Client, sg.ID, &requestOpts); err != nil {
				return nil, err
			} else {
				(*securityList)[i].Rules = *sgRules
				(*securityList)[i].RulesCount = len(*sgRules)
			}
		}
		var resultList []*irs.SecurityInfo
		for _, security := range *securityList {
			secInfo := setterSecGroup(security)
			resultList = append(resultList, secInfo)
		}
		return resultList, nil
	}
}

func (securityHandler *ClouditSecurityHandler) GetSecurity(securityID string) (irs.SecurityInfo, error) {
	securityHandler.Client.TokenID = securityHandler.CredentialInfo.AuthToken
	authHeader := securityHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if securityInfo, err := securitygroup.Get(securityHandler.Client, securityID, &requestOpts); err != nil {
		return irs.SecurityInfo{}, err
	} else {
		// SecurityGroup Rule 정보 가져오기
		if sgRules, err := securitygroup.ListRule(securityHandler.Client, securityInfo.ID, &requestOpts); err != nil {
			return irs.SecurityInfo{}, err
		} else {
			(*securityInfo).Rules = *sgRules
			(*securityInfo).RulesCount = len(*sgRules)
		}
		spew.Dump(securityInfo)
		secGroupInfo := setterSecGroup(*securityInfo)
		return *secGroupInfo, nil
	}
}

func (securityHandler *ClouditSecurityHandler) DeleteSecurity(securityID string) (bool, error) {
	securityHandler.Client.TokenID = securityHandler.CredentialInfo.AuthToken
	authHeader := securityHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	if err := securitygroup.Delete(securityHandler.Client, securityID, &requestOpts); err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func (securityHandler *ClouditSecurityHandler) CheckSecurityExist(securityName string) (*string, error) {
	var securityId string

	securityList, err := securityHandler.ListSecurity()
	if err != nil {
		return nil, err
	}

	for _, sec := range securityList {
		if sec.Name == securityName {
			securityId = sec.Id
		}
	}
	return &securityId, nil
}
