package resources

import (
	"context"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/new-resources"
	"github.com/davecgh/go-spew/spew"
	"strings"
)

type AzureSecurityHandler struct {
	Region idrv.RegionInfo
	Ctx    context.Context
	Client *network.SecurityGroupsClient
}

func setterSec(securityGroup network.SecurityGroup) *irs.SecurityInfo {
	security := &irs.SecurityInfo{
		Id:            *securityGroup.ID,
		Name:          *securityGroup.Name,
		SecurityRules: nil,
	}

	var securityRuleArr []irs.SecurityRuleInfo

	for _, sgRule := range *securityGroup.SecurityRules {
		ruleInfo := irs.SecurityRuleInfo{
			FromPort:   *sgRule.SourcePortRange,
			ToPort:     *sgRule.DestinationPortRange,
			IPProtocol: fmt.Sprint(sgRule.Protocol),
			Direction:  fmt.Sprint(sgRule.Direction),
		}

		securityRuleArr = append(securityRuleArr, ruleInfo)
	}

	security.SecurityRules = &securityRuleArr

	return security
}

func (securityHandler *AzureSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {

	var sgRuleList []network.SecurityRule
	for _, rule := range *securityReqInfo.SecurityRules {
		sgRuleInfo := network.SecurityRule{
			Name: to.StringPtr(securityReqInfo.Name),
			SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
				SourceAddressPrefix:      to.StringPtr("*"),
				SourcePortRange:          to.StringPtr(rule.FromPort),
				DestinationAddressPrefix: to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr(rule.ToPort),
				Protocol:                 network.SecurityRuleProtocol(rule.IPProtocol),
				Access:                   network.SecurityRuleAccess("Allow"),
				Priority:                 to.Int32Ptr(300),
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

	securityIdArr := strings.Split(securityReqInfo.Name, ":")

	// Check SecurityGroup Exists
	security, err := securityHandler.Client.Get(securityHandler.Ctx, securityIdArr[0], securityIdArr[1], "")
	if security.ID != nil {
		errMsg := fmt.Sprintf("Security Group with name %s already exist", securityIdArr[1])
		createErr := errors.New(errMsg)
		return irs.SecurityInfo{}, createErr
	}

	future, err := securityHandler.Client.CreateOrUpdate(securityHandler.Ctx, securityIdArr[0], securityIdArr[1], createOpts)
	if err != nil {
		return irs.SecurityInfo{}, err
	}
	err = future.WaitForCompletionRef(securityHandler.Ctx, securityHandler.Client.Client)
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	// @TODO: 생성된 SecurityGroup 정보 리턴
	securityInfo, err := securityHandler.GetSecurity(securityReqInfo.Name)
	if err != nil {
		return irs.SecurityInfo{}, err
	}
	return securityInfo, nil
}

func (securityHandler *AzureSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	//result, err := securityHandler.Client.ListAll(securityHandler.Ctx)
	result, err := securityHandler.Client.List(securityHandler.Ctx, securityHandler.Region.ResourceGroup)
	if err != nil {
		return nil, err
	}

	var securityList []*irs.SecurityInfo
	for _, security := range result.Values() {
		securityInfo := setterSec(security)
		securityList = append(securityList, securityInfo)
	}

	spew.Dump(securityList)
	return nil, nil
}

func (securityHandler *AzureSecurityHandler) GetSecurity(securityID string) (irs.SecurityInfo, error) {
	securityIdArr := strings.Split(securityID, ":")
	security, err := securityHandler.Client.Get(securityHandler.Ctx, securityIdArr[0], securityIdArr[1], "")
	if err != nil {
		return irs.SecurityInfo{}, err
	}

	securityInfo := setterSec(security)

	spew.Dump(securityInfo)
	return irs.SecurityInfo{}, nil
}

func (securityHandler *AzureSecurityHandler) DeleteSecurity(securityID string) (bool, error) {
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
