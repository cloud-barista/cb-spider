package resources

import (
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/iam/securitygroup"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"
	"github.com/davecgh/go-spew/spew"
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
			ToPort:     sgRule.Target, //todo:  toport, Direction에 가져올 데이터????
			IPProtocol: sgRule.Protocol,
			Direction:  sgRule.Target,
		}

		secRuleArr = append(secRuleArr, secRuleInfo)
	}
	secInfo.SecurityRules = &secRuleArr

	return secInfo
}

func (securityHandler *ClouditSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	securityHandler.Client.TokenID = securityHandler.CredentialInfo.AuthToken
	authHeader := securityHandler.Client.AuthenticatedHeaders()

	reqInfo := securitygroup.SecurityReqInfo{
		Name: securityReqInfo.Name,
		Rules: []securitygroup.SecurityGroupRules{
			{
				Name:     "SSH Inbound",
				Protocol: "tcp",
				Port:     "22",
				Target:   "0.0.0.0/0",
				Type:     "inbound",
			},
			{
				Name:     "Default Outbound",
				Protocol: "all",
				Port:     "0",
				Target:   "0.0.0.0/0",
				Type:     "outbound",
			},
		},
	}

	createOpts := client.RequestOpts{
		JSONBody:    reqInfo,
		MoreHeaders: authHeader,
	}

	if securityGroup, err := securitygroup.Create(securityHandler.Client, &createOpts); err != nil {
		return irs.SecurityInfo{}, err
	} else {
		spew.Dump(securityGroup)
		return irs.SecurityInfo{Id: securityGroup.ID, Name: securityGroup.Name}, nil
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
		return irs.SecurityInfo{Id: securityInfo.ID, Name: securityInfo.Name}, nil
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
