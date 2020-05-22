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

const (
	defaultSecGroupCIDR = "0.0.0.0/0"
)

type ClouditSecurityHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func setterSecGroup(secGroup securitygroup.SecurityGroupInfo) *irs.SecurityInfo {

	secInfo := &irs.SecurityInfo{
		IId: irs.IID{
			NameId:   secGroup.Name,
			SystemId: secGroup.ID,
		},
		VpcIID: irs.IID{
			NameId:   defaultVPCName,
			SystemId: defaultVPCName,
		},
		SecurityRules: nil,
	}

	secRuleArr := make([]irs.SecurityRuleInfo, len(secGroup.Rules))
	for i, sgRule := range secGroup.Rules {
		secRuleInfo := irs.SecurityRuleInfo{
			IPProtocol: sgRule.Protocol,
			Direction:  sgRule.Type,
		}
		if strings.Contains(sgRule.Port, "-") {
			portArr := strings.Split(sgRule.Port, "-")
			secRuleInfo.FromPort = portArr[0]
			secRuleInfo.ToPort = portArr[1]
		} else {
			secRuleInfo.FromPort = sgRule.Port
			secRuleInfo.ToPort = sgRule.Port
		}
		secRuleArr[i] = secRuleInfo
	}
	secInfo.SecurityRules = &secRuleArr

	return secInfo
}

func (securityHandler *ClouditSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	// 보안그룹 이름 중복 체크
	securityInfo, _ := securityHandler.getSecurityByName(securityReqInfo.IId.NameId)
	if securityInfo != nil {
		errMsg := fmt.Sprintf("SecurityGroup with name %s already exist", securityReqInfo.IId.NameId)
		createErr := errors.New(errMsg)
		return irs.SecurityInfo{}, createErr
	}

	securityHandler.Client.TokenID = securityHandler.CredentialInfo.AuthToken
	authHeader := securityHandler.Client.AuthenticatedHeaders()

	reqInfo := securitygroup.SecurityReqInfo{
		Name: securityReqInfo.IId.NameId,
	}

	// SecurityGroup Rule 설정
	ruleList := make([]securitygroup.SecurityGroupRules, len(*securityReqInfo.SecurityRules))
	for i, rule := range *securityReqInfo.SecurityRules {
		secRuleInfo := securitygroup.SecurityGroupRules{
			Name:     fmt.Sprintf("%s-rules-%d", securityReqInfo.IId.NameId, i+1),
			Type:     rule.Direction,
			Port:     rule.FromPort + "-" + rule.ToPort,
			Target:   defaultSecGroupCIDR,
			Protocol: strings.ToLower(rule.IPProtocol),
		}
		ruleList[i] = secRuleInfo
	}
	reqInfo.Rules = ruleList

	createOpts := client.RequestOpts{
		JSONBody:    reqInfo,
		MoreHeaders: authHeader,
	}

	securityGroup, err := securitygroup.Create(securityHandler.Client, &createOpts)
	if err != nil {
		return irs.SecurityInfo{}, err
	}
	secGroupInfo := setterSecGroup(*securityGroup)
	return *secGroupInfo, nil
}

func (securityHandler *ClouditSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	securityHandler.Client.TokenID = securityHandler.CredentialInfo.AuthToken
	authHeader := securityHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	securityList, err := securitygroup.List(securityHandler.Client, &requestOpts)
	if err != nil {
		return nil, err
	}

	// SecurityGroup Rule 정보 가져오기
	for i, sg := range *securityList {
		sgRules, err := securitygroup.ListRule(securityHandler.Client, sg.ID, &requestOpts)
		if err != nil {
			cblogger.Error(err)
			continue
		}
		(*securityList)[i].Rules = *sgRules
		(*securityList)[i].RulesCount = len(*sgRules)
	}

	resultList := make([]*irs.SecurityInfo, len(*securityList))
	for i, security := range *securityList {
		secInfo := setterSecGroup(security)
		resultList[i] = secInfo
	}
	return resultList, nil
}

func (securityHandler *ClouditSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	// 이름 기준 보안그룹 조회
	securityInfo, err := securityHandler.getSecurityByName(securityIID.NameId)
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
	sgRules, err := securitygroup.ListRule(securityHandler.Client, securityInfo.ID, &requestOpts)
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	(*securityInfo).Rules = *sgRules
	(*securityInfo).RulesCount = len(*sgRules)
	secGroupInfo := setterSecGroup(*securityInfo)
	return *secGroupInfo, nil
}

func (securityHandler *ClouditSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	// 이름 기준 보안그룹 조회
	securityInfo, err := securityHandler.getSecurityByName(securityIID.NameId)
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
	}
	return true, nil
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
		if strings.EqualFold(s.ID, securityName) {
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
