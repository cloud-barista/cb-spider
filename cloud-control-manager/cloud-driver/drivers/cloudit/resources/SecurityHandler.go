package resources

import (
	"errors"
	"fmt"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/iam/securitygroup"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
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
	// 보안그룹 이름 중복 체크
	securityInfo, _ := securityHandler.getSecurityByName(securityReqInfo.Name)
	if securityInfo != nil {
		errMsg := fmt.Sprintf("SecurityGroup with name %s already exist", securityReqInfo.Name)
		createErr := errors.New(errMsg)
		return irs.SecurityInfo{}, createErr
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
			Port:     rule.FromPort + "-" + rule.ToPort,
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

func (securityHandler *ClouditSecurityHandler) GetSecurity(securityNameID string) (irs.SecurityInfo, error) {
	// 이름 기준 보안그룹 조회
	securityInfo, err := securityHandler.getSecurityByName(securityNameID)
	if err != nil {
		cblogger.Error(err)
		return irs.SecurityInfo{}, err
	}

	securityHandler.Client.TokenID = securityHandler.CredentialInfo.AuthToken
	authHeader := securityHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	// SecurityGroup Rule 정보 가져오기
	if sgRules, err := securitygroup.ListRule(securityHandler.Client, securityInfo.ID, &requestOpts); err != nil {
		return irs.SecurityInfo{}, err
	} else {
		(*securityInfo).Rules = *sgRules
		(*securityInfo).RulesCount = len(*sgRules)
	}
	secGroupInfo := setterSecGroup(*securityInfo)
	return *secGroupInfo, nil
}

func (securityHandler *ClouditSecurityHandler) DeleteSecurity(securityNameID string) (bool, error) {
	// 이름 기준 보안그룹 조회
	securityInfo, err := securityHandler.getSecurityByName(securityNameID)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	securityHandler.Client.TokenID = securityHandler.CredentialInfo.AuthToken
	authHeader := securityHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	// 보안그룹 삭제
	if err := securitygroup.Delete(securityHandler.Client, securityInfo.ID, &requestOpts); err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func (securityHandler *ClouditSecurityHandler) getSecurityByName(securityName string) (*securitygroup.SecurityGroupInfo, error) {
	var security *securitygroup.SecurityGroupInfo

	securityHandler.Client.TokenID = securityHandler.CredentialInfo.AuthToken
	authHeader := securityHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	securityList, err := securitygroup.List(securityHandler.Client, &requestOpts)
	if err != nil {
		return nil, err
	}

	for _, s := range *securityList {
		if strings.EqualFold(s.Name, securityName) {
			security = &s
			break
		}
	}

	if security == nil {
		err := errors.New(fmt.Sprintf("failed to find security group with name %s", securityName))
		return nil, err
	}
	return security, nil
}
