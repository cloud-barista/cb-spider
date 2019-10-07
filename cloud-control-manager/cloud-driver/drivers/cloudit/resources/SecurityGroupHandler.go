package resources

import (
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/iam/securitygroup"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"strconv"
)

/*var cblogger *logrus.Logger

func init() {
	// cblog is a global variable.
	cblogger = cblog.GetLogger("CB-SPIDER")
}*/

type ClouditSecurityHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func (securityHandler *ClouditSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	securityHandler.Client.TokenID = securityHandler.CredentialInfo.AuthToken
	authHeader := securityHandler.Client.AuthenticatedHeaders()

	// @TODO: SecurityGroup 생성 요청 파라미터 정의 필요
	type SecurityReqInfo struct {
		Name       string                             `json:"name" required:"true"`
		Rules      []securitygroup.SecurityGroupRules `json:"rules" required:"false"`
		Protection int                                `json:"protection" required:"false"`
	}

	reqInfo := SecurityReqInfo{
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
		for i, security := range *securityList {
			cblogger.Info("[" + strconv.Itoa(i) + "]")
			spew.Dump(security)
		}
		return nil, nil
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
