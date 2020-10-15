package resources

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-04-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	ICMP          = "icmp"
	SecurityGroup = "SECURITYGROUP"
)

type AzureSecurityHandler struct {
	Region idrv.RegionInfo
	Ctx    context.Context
	Client *network.SecurityGroupsClient
}

func (securityHandler *AzureSecurityHandler) setterSec(securityGroup network.SecurityGroup) *irs.SecurityInfo {
	security := &irs.SecurityInfo{
		IId: irs.IID{
			NameId:   *securityGroup.Name,
			SystemId: *securityGroup.ID,
		},
		KeyValueList: []irs.KeyValue{{Key: "ResourceGroup", Value: securityHandler.Region.ResourceGroup}},
	}

	var securityRuleArr []irs.SecurityRuleInfo
	for _, sgRule := range *securityGroup.SecurityRules {

		var fromPort string
		var toPort string

		if strings.Contains(*sgRule.SourcePortRange, "-") {
			sourcePortArr := strings.Split(*sgRule.SourcePortRange, "-")
			fromPort = sourcePortArr[0]
			toPort = sourcePortArr[1]
		} else {
			fromPort = *sgRule.SourcePortRange
			toPort = *sgRule.DestinationPortRange
		}

		ruleInfo := irs.SecurityRuleInfo{
			IPProtocol: strings.ToLower(fmt.Sprint(sgRule.Protocol)),
			Direction:  fmt.Sprint(sgRule.Direction),
		}

		if strings.ToLower(fmt.Sprint(sgRule.Protocol)) == ICMP {
			ruleInfo.FromPort = "-1"
			ruleInfo.ToPort = "-1"
		} else {
			ruleInfo.FromPort = fromPort
			ruleInfo.ToPort = toPort
		}

		securityRuleArr = append(securityRuleArr, ruleInfo)
	}
	security.SecurityRules = &securityRuleArr

	return security
}

func (securityHandler *AzureSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, securityReqInfo.IId.NameId, "CreateSecurity()")

	// Check SecurityGroup Exists
	security, _ := securityHandler.Client.Get(securityHandler.Ctx, securityHandler.Region.ResourceGroup, securityReqInfo.IId.NameId, "")
	if security.ID != nil {
		createErr := errors.New(fmt.Sprintf("Security Group with name %s already exist", securityReqInfo.IId.NameId))
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}

	var sgRuleList []network.SecurityRule
	var priorityNum int32
	for idx, rule := range *securityReqInfo.SecurityRules {
		priorityNum = int32(300 + idx*100)
		sgRuleInfo := network.SecurityRule{
			Name: to.StringPtr(fmt.Sprintf("%s-rules-%d", securityReqInfo.IId.NameId, idx+1)),
			SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
				SourceAddressPrefix:      to.StringPtr("*"),
				DestinationAddressPrefix: to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr("*"),
				Protocol:                 network.SecurityRuleProtocol(strings.ToUpper(rule.IPProtocol)),
				Access:                   network.SecurityRuleAccess("Allow"),
				Priority:                 to.Int32Ptr(priorityNum),
				Direction:                network.SecurityRuleDirection(rule.Direction),
			},
		}

		if strings.ToLower(rule.IPProtocol) == ICMP || (rule.FromPort == "*" && rule.ToPort == "*") {
			sgRuleInfo.SourcePortRange = to.StringPtr("*")
		} else if rule.FromPort == rule.ToPort {
			sgRuleInfo.SourcePortRange = to.StringPtr(rule.FromPort)
		} else {
			sgRuleInfo.SourcePortRange = to.StringPtr(rule.FromPort + "-" + rule.ToPort)
		}

		sgRuleList = append(sgRuleList, sgRuleInfo)
	}

	createOpts := network.SecurityGroup{
		SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
			SecurityRules: &sgRuleList,
		},
		Location: &securityHandler.Region.Region,
	}

	start := call.Start()
	future, err := securityHandler.Client.CreateOrUpdate(securityHandler.Ctx, securityHandler.Region.ResourceGroup, securityReqInfo.IId.NameId, createOpts)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	err = future.WaitForCompletionRef(securityHandler.Ctx, securityHandler.Client.Client)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}

	// 생성된 SecurityGroup 정보 리턴
	securityInfo, err := securityHandler.GetSecurity(securityReqInfo.IId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}

	return securityInfo, nil
}

func (securityHandler *AzureSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, SecurityGroup, "ListSecurity()")

	start := call.Start()
	result, err := securityHandler.Client.List(securityHandler.Ctx, securityHandler.Region.ResourceGroup)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return nil, err
	}
	LoggingInfo(hiscallInfo, start)

	var securityList []*irs.SecurityInfo
	for _, security := range result.Values() {
		securityInfo := securityHandler.setterSec(security)
		securityList = append(securityList, securityInfo)
	}
	return securityList, nil
}

func (securityHandler *AzureSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, securityIID.NameId, "GetSecurity()")

	start := call.Start()
	security, err := securityHandler.Client.Get(securityHandler.Ctx, securityHandler.Region.ResourceGroup, securityIID.NameId, "")
	if err != nil {
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	securityInfo := securityHandler.setterSec(security)
	return *securityInfo, nil
}

func (securityHandler *AzureSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, securityIID.NameId, "DeleteSecurity()")

	start := call.Start()
	future, err := securityHandler.Client.Delete(securityHandler.Ctx, securityHandler.Region.ResourceGroup, securityIID.NameId)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}
	err = future.WaitForCompletionRef(securityHandler.Ctx, securityHandler.Client.Client)
	if err != nil {
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}
