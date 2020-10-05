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
	cblogger.Info("Call Azure ListSecurity()")
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, securityReqInfo.IId.NameId, "CreateSecurity()")

	// Check SecurityGroup Exists
	security, _ := securityHandler.Client.Get(securityHandler.Ctx, securityHandler.Region.ResourceGroup, securityReqInfo.IId.NameId, "")
	if security.ID != nil {
		errMsg := fmt.Sprintf("Security Group with name %s already exist", securityReqInfo.IId.NameId)
		createErr := errors.New(errMsg)
		cblogger.Error(createErr.Error())
		hiscallInfo.ErrorMSG = createErr.Error()
		calllogger.Info(call.String(hiscallInfo))
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
		cblogger.Error(err.Error())
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Info(call.String(hiscallInfo))
		return irs.SecurityInfo{}, err
	}
	err = future.WaitForCompletionRef(securityHandler.Ctx, securityHandler.Client.Client)
	if err != nil {
		cblogger.Error(err.Error())
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Info(call.String(hiscallInfo))
		return irs.SecurityInfo{}, err
	}

	// 생성된 SecurityGroup 정보 리턴
	securityInfo, err := securityHandler.GetSecurity(securityReqInfo.IId)
	if err != nil {
		cblogger.Error(err.Error())
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Info(call.String(hiscallInfo))
		return irs.SecurityInfo{}, err
	}

	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	return securityInfo, nil
}

func (securityHandler *AzureSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	// log HisCall
	cblogger.Info("Call Azure ListSecurity()")
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, SecurityGroup, "ListSecurity()")

	start := call.Start()
	result, err := securityHandler.Client.List(securityHandler.Ctx, securityHandler.Region.ResourceGroup)
	if err != nil {
		cblogger.Error(err.Error())
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Info(call.String(hiscallInfo))
		return nil, err
	}
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	var securityList []*irs.SecurityInfo
	for _, security := range result.Values() {
		securityInfo := securityHandler.setterSec(security)
		securityList = append(securityList, securityInfo)
	}
	return securityList, nil
}

func (securityHandler *AzureSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	// log HisCall
	cblogger.Info("Call Azure GetSecurity()")
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, securityIID.NameId, "GetSecurity()")

	start := call.Start()
	security, err := securityHandler.Client.Get(securityHandler.Ctx, securityHandler.Region.ResourceGroup, securityIID.NameId, "")
	if err != nil {
		cblogger.Error(err.Error())
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Info(call.String(hiscallInfo))
		return irs.SecurityInfo{}, err
	}
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))

	securityInfo := securityHandler.setterSec(security)
	return *securityInfo, nil
}

func (securityHandler *AzureSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	// log HisCall
	cblogger.Info("Call Azure GetSecurity()")
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, securityIID.NameId, "DeleteSecurity()")

	start := call.Start()
	future, err := securityHandler.Client.Delete(securityHandler.Ctx, securityHandler.Region.ResourceGroup, securityIID.NameId)
	if err != nil {
		cblogger.Error(err.Error())
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Info(call.String(hiscallInfo))
		return false, err
	}
	err = future.WaitForCompletionRef(securityHandler.Ctx, securityHandler.Client.Client)
	if err != nil {
		cblogger.Error(err.Error())
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Info(call.String(hiscallInfo))
		return false, err
	}
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	calllogger.Info(call.String(hiscallInfo))
	return true, nil
}
