package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
	"github.com/Azure/go-autorest/autorest/to"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	ICMP          = "icmp"
	SecurityGroup = "SECURITYGROUP"
	initPriority  = 100
	maxPriority   = 4096
)

type AzureSecurityHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *network.SecurityGroupsClient
	RuleClient *network.SecurityRulesClient
}

func (securityHandler *AzureSecurityHandler) setterSec(securityGroup network.SecurityGroup) *irs.SecurityInfo {
	keyValues := []irs.KeyValue{{Key: "ResourceGroup", Value: securityHandler.Region.ResourceGroup}}
	security := &irs.SecurityInfo{
		IId: irs.IID{
			NameId:   *securityGroup.Name,
			SystemId: *securityGroup.ID,
		},
	}

	var securityRuleArr []irs.SecurityRuleInfo
	for _, sgRule := range *securityGroup.SecurityRules {
		if sgRule.Access == network.SecurityRuleAccessAllow {
			ruleInfo, _ := convertRuleInfoAZToCB(sgRule)
			securityRuleArr = append(securityRuleArr, ruleInfo)
		} else {
			unControlledRule := unControlledRule{
				Name:        *sgRule.Name,
				Port:        *sgRule.DestinationPortRange,
				Protocol:    fmt.Sprint(sgRule.Protocol),
				source:      *sgRule.SourceAddressPrefix,
				Destination: *sgRule.DestinationAddressPrefix,
				Action:      "Deny",
			}
			b, err := json.Marshal(unControlledRule)
			if err == nil {
				keyValues = append(keyValues, irs.KeyValue{
					Key:   *sgRule.Name,
					Value: string(b),
				})
			}
		}
	}
	for _, sgRule := range *securityGroup.DefaultSecurityRules {
		action := "Deny"
		if sgRule.Access == network.SecurityRuleAccessAllow {
			action = "Allow"
		}
		unControlledRule := unControlledRule{
			Name:        *sgRule.Name,
			Port:        *sgRule.DestinationPortRange,
			Protocol:    fmt.Sprint(sgRule.Protocol),
			source:      *sgRule.SourceAddressPrefix,
			Destination: *sgRule.DestinationAddressPrefix,
			Action:      action,
		}
		b, err := json.Marshal(unControlledRule)
		if err == nil {
			keyValues = append(keyValues, irs.KeyValue{
				Key:   *sgRule.Name,
				Value: string(b),
			})
		}
	}
	security.KeyValueList = keyValues
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
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}

	sgRuleList, err := convertRuleInfoListCBToAZ(*securityReqInfo.SecurityRules)

	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}

	sgRuleList, err = addCBDefaultRule(sgRuleList)

	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}

	createOpts := network.SecurityGroup{
		SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
			SecurityRules: sgRuleList,
		},
		Location: &securityHandler.Region.Region,
	}

	start := call.Start()
	future, err := securityHandler.Client.CreateOrUpdate(securityHandler.Ctx, securityHandler.Region.ResourceGroup, securityReqInfo.IId.NameId, createOpts)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	err = future.WaitForCompletionRef(securityHandler.Ctx, securityHandler.Client.Client)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}

	// 생성된 SecurityGroup 정보 리턴
	securityInfo, err := securityHandler.GetSecurity(securityReqInfo.IId)
	if err != nil {
		cblogger.Error(err.Error())
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
		cblogger.Error(err.Error())
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
	rawSecurityGroup, err := getRawSecurityGroup(securityIID, securityHandler.Client, securityHandler.Ctx, securityHandler.Region.ResourceGroup)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}
	LoggingInfo(hiscallInfo, start)

	securityInfo := securityHandler.setterSec(*rawSecurityGroup)
	return *securityInfo, nil
}

func (securityHandler *AzureSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, securityIID.NameId, "DeleteSecurity()")

	start := call.Start()
	future, err := securityHandler.Client.Delete(securityHandler.Ctx, securityHandler.Region.ResourceGroup, securityIID.NameId)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return false, err
	}
	err = future.WaitForCompletionRef(securityHandler.Ctx, securityHandler.Client.Client)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return false, err
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (securityHandler *AzureSecurityHandler) AddRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, sgIID.NameId, "AddRules()")

	start := call.Start()
	security, err := getRawSecurityGroup(sgIID, securityHandler.Client, securityHandler.Ctx, securityHandler.Region.ResourceGroup)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}

	baseRuleWithNames, err := getRuleInfoWithNames(security.SecurityRules)

	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}

	var addRuleInfos []irs.SecurityRuleInfo

	for _, addRule := range *securityRules {
		existCheck := false
		for _, baseRule := range *baseRuleWithNames {
			if equalsRule(baseRule.RuleInfo, addRule) {
				existCheck = true
				break
			}
		}
		if existCheck {
			b, err := json.Marshal(addRule)
			err = errors.New(fmt.Sprintf("already Exist Rule : %s", string(b)))
			cblogger.Error(err.Error())
			LoggingError(hiscallInfo, err)
			return irs.SecurityInfo{}, err
		}
		addRuleInfos = append(addRuleInfos, addRule)
	}

	addAZRule, err := getAddAzureRules(security.SecurityRules, &addRuleInfos)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}

	for _, ru := range *addAZRule {
		future, err := securityHandler.RuleClient.CreateOrUpdate(securityHandler.Ctx, securityHandler.Region.ResourceGroup, sgIID.NameId, *ru.Name, ru)
		if err != nil {
			cblogger.Error(err.Error())
			LoggingError(hiscallInfo, err)
			return irs.SecurityInfo{}, err
		}
		err = future.WaitForCompletionRef(securityHandler.Ctx, securityHandler.RuleClient.Client)
		if err != nil {
			cblogger.Error(err.Error())
			LoggingError(hiscallInfo, err)
			return irs.SecurityInfo{}, err
		}
	}

	// 변된 SecurityGroup 정보 리턴
	updatedSecurity, err := getRawSecurityGroup(sgIID, securityHandler.Client, securityHandler.Ctx, securityHandler.Region.ResourceGroup)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}

	updatedSecurityInfo := securityHandler.setterSec(*updatedSecurity)
	LoggingInfo(hiscallInfo, start)

	return *updatedSecurityInfo, nil
}

func (securityHandler *AzureSecurityHandler) RemoveRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (bool, error) {
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, sgIID.NameId, "RemoveRules()")

	start := call.Start()

	security, err := getRawSecurityGroup(sgIID, securityHandler.Client, securityHandler.Ctx, securityHandler.Region.ResourceGroup)

	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return false, err
	}

	baseRuleWithNames, err := getRuleInfoWithNames(security.SecurityRules)

	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return false, err
	}

	var deleteRuleNames []string

	for _, delRule := range *securityRules {
		existCheck := false
		for _, baseRule := range *baseRuleWithNames {
			if equalsRule(baseRule.RuleInfo, delRule) {
				existCheck = true
				deleteRuleNames = append(deleteRuleNames, baseRule.Name)
				break
			}
		}
		if !existCheck {
			b, err := json.Marshal(delRule)
			err = errors.New(fmt.Sprintf("not Exist Rule : %s", string(b)))
			cblogger.Error(err.Error())
			LoggingError(hiscallInfo, err)
			return false, err
		}
	}

	for _, deleteRuleName := range deleteRuleNames {
		future, err := securityHandler.RuleClient.Delete(securityHandler.Ctx, securityHandler.Region.ResourceGroup, sgIID.NameId, deleteRuleName)
		if err != nil {
			cblogger.Error(err.Error())
			LoggingError(hiscallInfo, err)
			return false, err
		}
		err = future.WaitForCompletionRef(securityHandler.Ctx, securityHandler.RuleClient.Client)
		if err != nil {
			cblogger.Error(err.Error())
			LoggingError(hiscallInfo, err)
			return false, err
		}
	}

	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func getRawSecurityGroup(sgIID irs.IID, client *network.SecurityGroupsClient, ctx context.Context, resourceGroup string) (*network.SecurityGroup, error) {
	if sgIID.SystemId == "" && sgIID.NameId == "" {
		return nil, errors.New("invalid IID")
	}
	if sgIID.NameId == "" {
		result, err := client.List(ctx, resourceGroup)
		if err != nil {
			return nil, err
		}
		for _, sg := range result.Values() {
			if *sg.ID == sgIID.SystemId {
				return &sg, nil
			}
		}
		return nil, errors.New("not found SecurityGroup")
	} else {
		security, err := client.Get(ctx, resourceGroup, sgIID.NameId, "")
		return &security, err
	}
}

func convertRuleProtocolAZToCB(protocol string) string {
	switch strings.ToUpper(protocol) {
	case "*":
		return strings.ToLower("all")
	default:
		return strings.ToLower(protocol)
	}
}

func convertRuleProtocolCBToAZ(protocol string) (string, error) {
	switch strings.ToUpper(protocol) {
	case "ALL":
		return strings.ToUpper("*"), nil
	case "ICMP", "TCP", "UDP":
		return strings.ToUpper(protocol), nil
	}
	return "", errors.New("invalid Rule Protocol")
}

func convertRulePortRangeAZToCB(portRange string, protocol string) (from string, to string) {
	if strings.ToUpper(protocol) == "ICMP" || strings.ToUpper(protocol) == "ALL" {
		return "-1", "-1"
	}
	portRangeArr := strings.Split(portRange, "-")
	if len(portRangeArr) != 2 {
		if len(portRangeArr) == 1 && portRange != "*" {
			return portRangeArr[0], portRangeArr[0]
		}
		return "1", "65535"
	}
	return portRangeArr[0], portRangeArr[1]
}

func equalsRule(pre irs.SecurityRuleInfo, post irs.SecurityRuleInfo) bool {
	if pre.ToPort == "-1" || pre.FromPort == "-1" {
		pre.FromPort = "1"
		pre.ToPort = "65535"
	}
	if post.ToPort == "-1" || post.FromPort == "-1" {
		post.FromPort = "1"
		post.ToPort = "65535"
	}
	return strings.ToLower(fmt.Sprintf("%#v", pre)) == strings.ToLower(fmt.Sprintf("%#v", post))
}

func convertRuleDirectionCBToAZ(direction string) (network.SecurityRuleDirection, error) {
	if strings.ToLower(direction) == "inbound" {
		return network.SecurityRuleDirectionInbound, nil
	}
	if strings.ToLower(direction) == "outbound" {
		return network.SecurityRuleDirectionOutbound, nil
	}
	return "", errors.New("invalid rule Direction")
}

func convertRuleDirectionAZToCB(direction network.SecurityRuleDirection) (string, error) {
	if direction == network.SecurityRuleDirectionInbound {
		return "inbound", nil
	}
	if direction == network.SecurityRuleDirectionOutbound {
		return "outbound", nil
	}
	return "", errors.New("invalid rule Direction")
}

func convertRuleCIDRAZToCB(cidr string) string {
	if cidr == "*" {
		return "0.0.0.0/0"
	}
	return cidr
}

func convertRulePortRangeCBToAZ(from string, to string, protocol string) (string, error) {
	if strings.ToUpper(protocol) == "ICMP" {
		return "*", nil
	}
	if from == "" || to == "" {
		return "", errors.New("invalid Rule PortRange")
	}
	fromInt, err := strconv.Atoi(from)
	if err != nil {
		return "", errors.New("invalid Rule PortRange")
	}
	toInt, err := strconv.Atoi(to)
	if err != nil {
		return "", errors.New("invalid Rule PortRange")
	}
	if fromInt == -1 || toInt == -1 {
		return "*", nil
	}
	if fromInt > 65535 || fromInt < -1 || toInt > 65535 || toInt < -1 {
		return "", errors.New("invalid Rule PortRange")
	}
	if fromInt == toInt {
		return strconv.Itoa(fromInt), nil
	} else {
		return fmt.Sprintf("%d-%d", fromInt, toInt), nil
	}
}

func addCBDefaultRule(azureSGRuleList *[]network.SecurityRule) (*[]network.SecurityRule, error) {
	outboundPriority := initPriority
	var addCBDefaultRuleList []network.SecurityRule
	for _, sgRule := range *azureSGRuleList {
		if sgRule.Access == network.SecurityRuleAccessAllow {
			if sgRule.Direction == network.SecurityRuleDirectionOutbound && outboundPriority < int(*sgRule.Priority) {
				outboundPriority = int(*sgRule.Priority)
			}
		}
	}

	cbDefaultDenySGRule := network.SecurityRule{
		Name: to.StringPtr("deny-outbound"),
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			SourceAddressPrefix:      to.StringPtr("*"),
			SourcePortRange:          to.StringPtr("*"),
			DestinationAddressPrefix: to.StringPtr("0.0.0.0/0"),
			DestinationPortRange:     to.StringPtr("*"),
			Protocol:                 network.SecurityRuleProtocol("*"),
			Access:                   network.SecurityRuleAccessDeny,
			Priority:                 to.Int32Ptr(maxPriority),
			Direction:                network.SecurityRuleDirectionOutbound,
		},
	}
	addCBDefaultRuleList = append(*azureSGRuleList, cbDefaultDenySGRule)

	cbDefaultAllowSGRule := network.SecurityRule{
		Name: to.StringPtr("allow-outbound"),
		SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
			SourceAddressPrefix:      to.StringPtr("*"),
			SourcePortRange:          to.StringPtr("*"),
			DestinationAddressPrefix: to.StringPtr("0.0.0.0/0"),
			DestinationPortRange:     to.StringPtr("*"),
			Protocol:                 network.SecurityRuleProtocol("*"),
			Access:                   network.SecurityRuleAccessAllow,
			Priority:                 to.Int32Ptr(int32(outboundPriority) + 1),
			Direction:                network.SecurityRuleDirectionOutbound,
		},
	}
	protocols := convertRuleProtocolAZToCB(fmt.Sprint(cbDefaultAllowSGRule.Protocol))
	fromPort, toPort := convertRulePortRangeAZToCB(*cbDefaultAllowSGRule.DestinationPortRange, protocols)
	direction, err := convertRuleDirectionAZToCB(cbDefaultAllowSGRule.Direction)
	if err != nil {
		return nil, err
	}
	cidr := convertRuleCIDRAZToCB(*cbDefaultAllowSGRule.SourceAddressPrefix)
	cbDefaultAllowSGRuleInfo := irs.SecurityRuleInfo{
		IPProtocol: protocols,
		Direction:  direction,
		CIDR:       cidr,
		FromPort:   fromPort,
		ToPort:     toPort,
	}
	addAllowDefaultRule := false
	for _, sgRule := range *azureSGRuleList {
		if sgRule.Access == network.SecurityRuleAccessAllow {
			protocols := convertRuleProtocolAZToCB(fmt.Sprint(sgRule.Protocol))
			fromPort, toPort := convertRulePortRangeAZToCB(*sgRule.DestinationPortRange, protocols)
			direction, err := convertRuleDirectionAZToCB(sgRule.Direction)
			if err != nil {
				return nil, err
			}
			cidr := convertRuleCIDRAZToCB(*sgRule.SourceAddressPrefix)
			ruleInfo := irs.SecurityRuleInfo{
				IPProtocol: protocols,
				Direction:  direction,
				CIDR:       cidr,
				FromPort:   fromPort,
				ToPort:     toPort,
			}
			if equalsRule(ruleInfo, cbDefaultAllowSGRuleInfo) {
				addAllowDefaultRule = true
			}
		}
	}
	if !addAllowDefaultRule {
		addCBDefaultRuleList = append(addCBDefaultRuleList, cbDefaultAllowSGRule)
	}

	return &addCBDefaultRuleList, nil
}

func generateRuleName(direct string) string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%s-rules-%s", direct, strconv.FormatInt(rand.Int63n(100000), 10))
}

func convertRuleInfoAZToCB(rawRule network.SecurityRule) (irs.SecurityRuleInfo, error) {
	protocols := convertRuleProtocolAZToCB(fmt.Sprint(rawRule.Protocol))
	fromPort, toPort := convertRulePortRangeAZToCB(*rawRule.DestinationPortRange, protocols)
	direction, err := convertRuleDirectionAZToCB(rawRule.Direction)
	if err != nil {
		return irs.SecurityRuleInfo{}, err
	}
	cidr := convertRuleCIDRAZToCB(*rawRule.SourceAddressPrefix)
	if rawRule.Direction == network.SecurityRuleDirectionInbound {
		RuleInfo := irs.SecurityRuleInfo{
			IPProtocol: protocols,
			Direction:  direction,
			CIDR:       cidr,
			FromPort:   fromPort,
			ToPort:     toPort,
		}
		return RuleInfo, nil
	} else {
		RuleInfo := irs.SecurityRuleInfo{
			IPProtocol: protocols,
			Direction:  direction,
			CIDR:       cidr,
			FromPort:   fromPort,
			ToPort:     toPort,
		}
		return RuleInfo, nil
	}

}

func convertRuleInfoCBToAZ(rule irs.SecurityRuleInfo, priority int32) (network.SecurityRule, error) {
	protocol, err := convertRuleProtocolCBToAZ(rule.IPProtocol)
	if err != nil {
		return network.SecurityRule{}, err
	}
	portRange, err := convertRulePortRangeCBToAZ(rule.FromPort, rule.ToPort, rule.IPProtocol)
	if err != nil {
		return network.SecurityRule{}, err
	}
	direction, err := convertRuleDirectionCBToAZ(rule.Direction)
	if err != nil {
		return network.SecurityRule{}, err
	}
	if direction == network.SecurityRuleDirectionInbound {
		sgRuleInfo := network.SecurityRule{
			Name: to.StringPtr(fmt.Sprintf("%s-%s-%s-%s", generateRuleName(rule.Direction), rule.FromPort, rule.ToPort, rule.IPProtocol)),
			SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
				SourceAddressPrefix:      to.StringPtr(rule.CIDR),
				SourcePortRange:          to.StringPtr("*"),
				DestinationAddressPrefix: to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr(portRange),
				Protocol:                 network.SecurityRuleProtocol(protocol),
				Access:                   network.SecurityRuleAccessAllow,
				Priority:                 to.Int32Ptr(priority),
				Direction:                direction,
			},
		}
		return sgRuleInfo, nil
	} else {
		sgRuleInfo := network.SecurityRule{
			Name: to.StringPtr(fmt.Sprintf("%s-%s-%s-%s", generateRuleName(rule.Direction), rule.FromPort, rule.ToPort, rule.IPProtocol)),
			SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
				SourceAddressPrefix:      to.StringPtr("*"),
				SourcePortRange:          to.StringPtr("*"),
				DestinationAddressPrefix: to.StringPtr(rule.CIDR),
				DestinationPortRange:     to.StringPtr(portRange),
				Protocol:                 network.SecurityRuleProtocol(protocol),
				Access:                   network.SecurityRuleAccessAllow,
				Priority:                 to.Int32Ptr(priority),
				Direction:                direction,
			},
		}
		return sgRuleInfo, nil
	}

}

func convertRuleInfoListCBToAZ(rules []irs.SecurityRuleInfo) (*[]network.SecurityRule, error) {
	var azureSGRuleList []network.SecurityRule
	var priorityNum int32
	for idx, rule := range rules {
		priorityNum = int32(initPriority + idx)
		sgRuleInfo, err := convertRuleInfoCBToAZ(rule, priorityNum)
		if err != nil {
			return nil, err
		}
		azureSGRuleList = append(azureSGRuleList, sgRuleInfo)
	}
	return &azureSGRuleList, nil
}

type securityRuleInfoWithName struct {
	Name     string
	RuleInfo irs.SecurityRuleInfo
}

func getRuleInfoWithNames(rawRules *[]network.SecurityRule) (*[]securityRuleInfoWithName, error) {
	var ruleInfoWithNames []securityRuleInfoWithName
	for _, sgRule := range *rawRules {
		if sgRule.Access == network.SecurityRuleAccessAllow {
			ruleInfo, err := convertRuleInfoAZToCB(sgRule)
			if err != nil {
				return nil, err
			}
			securityRuleInfoName := securityRuleInfoWithName{
				Name:     *sgRule.Name,
				RuleInfo: ruleInfo,
			}
			ruleInfoWithNames = append(ruleInfoWithNames, securityRuleInfoName)
		}
	}
	return &ruleInfoWithNames, nil
}

type unControlledRule struct {
	Name        string
	Port        string
	Protocol    string
	source      string
	Destination string
	Action      string
}

func getAddAzureRules(baseRawRules *[]network.SecurityRule, addRuleInfo *[]irs.SecurityRuleInfo) (*[]network.SecurityRule, error) {
	inboundPriority := initPriority
	outboundPriority := initPriority
	for _, sgRule := range *baseRawRules {
		if sgRule.Access == network.SecurityRuleAccessAllow {
			if sgRule.Direction == network.SecurityRuleDirectionInbound && inboundPriority < int(*sgRule.Priority) {
				inboundPriority = int(*sgRule.Priority)
			}
			if sgRule.Direction == network.SecurityRuleDirectionOutbound && outboundPriority < int(*sgRule.Priority) {
				outboundPriority = int(*sgRule.Priority)
			}
		}
	}
	var azureSGRuleList []network.SecurityRule

	for idx, rule := range *addRuleInfo {
		priorityNum := int32(initPriority + 1 + idx)
		if strings.ToLower(rule.Direction) == "inbound" {
			priorityNum = int32(inboundPriority + 1 + idx)
			inboundPriority = inboundPriority + 1 + idx
		}
		if strings.ToLower(rule.Direction) == "outbound" {
			priorityNum = int32(outboundPriority + 1 + idx)
			outboundPriority = outboundPriority + 1 + idx
		}
		sgRuleInfo, err := convertRuleInfoCBToAZ(rule, priorityNum)
		if err != nil {
			return nil, err
		}
		azureSGRuleList = append(azureSGRuleList, sgRuleInfo)
	}
	return &azureSGRuleList, nil
}
