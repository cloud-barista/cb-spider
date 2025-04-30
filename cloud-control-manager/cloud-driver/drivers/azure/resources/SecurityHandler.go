package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v6"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"math/rand"
	"strconv"
	"strings"
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
	Client     *armnetwork.SecurityGroupsClient
	RuleClient *armnetwork.SecurityRulesClient
}

func (securityHandler *AzureSecurityHandler) setterSec(securityGroup *armnetwork.SecurityGroup) *irs.SecurityInfo {
	//keyValues := []irs.KeyValue{{Key: "ResourceGroup", Value: securityHandler.Region.Region}}
	security := &irs.SecurityInfo{
		IId: irs.IID{
			NameId:   *securityGroup.Name,
			SystemId: *securityGroup.ID,
		},
	}

	var securityRuleArr []irs.SecurityRuleInfo
	for _, sgRule := range securityGroup.Properties.SecurityRules {
		if *sgRule.Properties.Access == armnetwork.SecurityRuleAccessAllow {
			ruleInfo, _ := convertRuleInfoAZToCB(sgRule)
			securityRuleArr = append(securityRuleArr, ruleInfo)
		} // else {
		//unControlledRule := unControlledRule{
		//	Name:        *sgRule.Name,
		//	Port:        *sgRule.Properties.DestinationPortRange,
		//	Protocol:    fmt.Sprint(*sgRule.Properties.Protocol),
		//	source:      *sgRule.Properties.SourceAddressPrefix,
		//	Destination: *sgRule.Properties.DestinationAddressPrefix,
		//	Action:      string(armnetwork.SecurityRuleAccessDeny),
		//}
		//b, err := json.Marshal(unControlledRule)
		//if err == nil {
		//	keyValues = append(keyValues, irs.KeyValue{
		//		Key:   *sgRule.Name,
		//		Value: string(b),
		//	})
		//}
		//}
	}
	//for _, sgRule := range securityGroup.Properties.DefaultSecurityRules {
	//	action := string(armnetwork.SecurityRuleAccessDeny)
	//	if *sgRule.Properties.Access == armnetwork.SecurityRuleAccessAllow {
	//		action = string(armnetwork.SecurityRuleAccessAllow)
	//	}
	//	unControlledRule := unControlledRule{
	//		Name:        *sgRule.Name,
	//		Port:        *sgRule.Properties.DestinationPortRange,
	//		Protocol:    fmt.Sprint(*sgRule.Properties.Protocol),
	//		source:      *sgRule.Properties.SourceAddressPrefix,
	//		Destination: *sgRule.Properties.DestinationAddressPrefix,
	//		Action:      action,
	//	}
	//	b, err := json.Marshal(unControlledRule)
	//	if err == nil {
	//		keyValues = append(keyValues, irs.KeyValue{
	//			Key:   *sgRule.Name,
	//			Value: string(b),
	//		})
	//	}
	//}

	if securityGroup.Tags != nil {
		security.TagList = setTagList(securityGroup.Tags)
	}

	//security.KeyValueList = keyValues
	security.KeyValueList = irs.StructToKeyValueList(securityGroup)
	security.SecurityRules = &securityRuleArr

	return security
}

func (securityHandler *AzureSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, securityReqInfo.IId.NameId, "CreateSecurity()")

	// Check SecurityGroup Exists
	resp, _ := securityHandler.Client.Get(securityHandler.Ctx, securityHandler.Region.Region, securityReqInfo.IId.NameId, nil)
	if resp.SecurityGroup.ID != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Security. err = Security Group with name %s already exist", securityReqInfo.IId.NameId))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}
	// Create Tag
	tags := setTags(securityReqInfo.TagList)
	sgRuleList, err := convertRuleInfoListCBToAZ(*securityReqInfo.SecurityRules)

	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Security. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}

	sgRuleList, err = addCBDefaultRule(sgRuleList)

	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Security. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}

	createOpts := armnetwork.SecurityGroup{
		Properties: &armnetwork.SecurityGroupPropertiesFormat{
			SecurityRules: sgRuleList,
		},
		Location: &securityHandler.Region.Region,
		Tags:     tags,
	}

	start := call.Start()
	poller, err := securityHandler.Client.BeginCreateOrUpdate(securityHandler.Ctx, securityHandler.Region.Region, securityReqInfo.IId.NameId, createOpts, nil)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Security. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}
	_, err = poller.PollUntilDone(securityHandler.Ctx, nil)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Security. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}
	LoggingInfo(hiscallInfo, start)

	// 생성된 SecurityGroup 정보 리턴
	securityInfo, err := securityHandler.GetSecurity(securityReqInfo.IId)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Security. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}

	return securityInfo, nil
}

func (securityHandler *AzureSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, SecurityGroup, "ListSecurity()")
	start := call.Start()

	var securityGroupList []*armnetwork.SecurityGroup

	pager := securityHandler.Client.NewListPager(securityHandler.Region.Region, nil)

	for pager.More() {
		page, err := pager.NextPage(securityHandler.Ctx)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to List Security. err = %s", err))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return []*irs.SecurityInfo{}, getErr
		}

		for _, securityGroup := range page.Value {
			securityGroupList = append(securityGroupList, securityGroup)
		}
	}

	var securityList []*irs.SecurityInfo
	for _, security := range securityGroupList {
		securityInfo := securityHandler.setterSec(security)
		securityList = append(securityList, securityInfo)
	}
	LoggingInfo(hiscallInfo, start)
	return securityList, nil
}

func (securityHandler *AzureSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, securityIID.NameId, "GetSecurity()")

	start := call.Start()
	rawSecurityGroup, err := getRawSecurityGroup(securityIID, securityHandler.Client, securityHandler.Ctx, securityHandler.Region.Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Security. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)

	securityInfo := securityHandler.setterSec(rawSecurityGroup)
	return *securityInfo, nil
}

func (securityHandler *AzureSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, securityIID.NameId, "DeleteSecurity()")

	start := call.Start()
	poller, err := securityHandler.Client.BeginDelete(securityHandler.Ctx, securityHandler.Region.Region, securityIID.NameId, nil)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Security. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	_, err = poller.PollUntilDone(securityHandler.Ctx, nil)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Security. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)
	return true, nil
}

func (securityHandler *AzureSecurityHandler) AddRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, sgIID.NameId, "AddRules()")

	start := call.Start()
	security, err := getRawSecurityGroup(sgIID, securityHandler.Client, securityHandler.Ctx, securityHandler.Region.Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}

	baseRuleWithNames, err := getRuleInfoWithNames(security.Properties.SecurityRules)

	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
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
			getErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return irs.SecurityInfo{}, getErr
		}
		addRuleInfos = append(addRuleInfos, addRule)
	}

	addAZRule, err := getAddAzureRules(security.Properties.SecurityRules, &addRuleInfos)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}

	for _, ru := range *addAZRule {
		poller, err := securityHandler.RuleClient.BeginCreateOrUpdate(securityHandler.Ctx, securityHandler.Region.Region, sgIID.NameId, *ru.Name, ru, nil)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return irs.SecurityInfo{}, getErr
		}
		_, err = poller.PollUntilDone(securityHandler.Ctx, nil)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return irs.SecurityInfo{}, getErr
		}
	}

	// 변된 SecurityGroup 정보 리턴
	updatedSecurity, err := getRawSecurityGroup(sgIID, securityHandler.Client, securityHandler.Ctx, securityHandler.Region.Region)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}

	updatedSecurityInfo := securityHandler.setterSec(updatedSecurity)
	LoggingInfo(hiscallInfo, start)

	return *updatedSecurityInfo, nil
}

func (securityHandler *AzureSecurityHandler) RemoveRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (bool, error) {
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, sgIID.NameId, "RemoveRules()")

	start := call.Start()

	security, err := getRawSecurityGroup(sgIID, securityHandler.Client, securityHandler.Ctx, securityHandler.Region.Region)

	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Remove SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}

	baseRuleWithNames, err := getRuleInfoWithNames(security.Properties.SecurityRules)

	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Remove SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
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
			getErr := errors.New(fmt.Sprintf("Failed to Remove SecurityGroup Rules. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return false, getErr
		}
	}

	for _, deleteRuleName := range deleteRuleNames {
		poller, err := securityHandler.RuleClient.BeginDelete(securityHandler.Ctx, securityHandler.Region.Region, sgIID.NameId, deleteRuleName, nil)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to Remove SecurityGroup Rules. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return false, getErr
		}
		_, err = poller.PollUntilDone(securityHandler.Ctx, nil)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to Remove SecurityGroup Rules. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return false, getErr
		}
	}

	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func getRawSecurityGroup(sgIID irs.IID, client *armnetwork.SecurityGroupsClient, ctx context.Context, resourceGroup string) (*armnetwork.SecurityGroup, error) {
	if sgIID.SystemId == "" && sgIID.NameId == "" {
		return nil, errors.New("invalid IID")
	}
	if sgIID.NameId == "" {
		var securityGroupList []*armnetwork.SecurityGroup

		pager := client.NewListPager(resourceGroup, nil)

		for pager.More() {
			page, err := pager.NextPage(ctx)
			if err != nil {
				return nil, err
			}

			for _, securityGroup := range page.Value {
				securityGroupList = append(securityGroupList, securityGroup)
			}
		}

		for _, sg := range securityGroupList {
			if *sg.ID == sgIID.SystemId {
				return sg, nil
			}
		}
		return nil, errors.New("not found SecurityGroup")
	} else {
		resp, err := client.Get(ctx, resourceGroup, sgIID.NameId, nil)
		if err != nil {
			return nil, err
		}

		return &resp.SecurityGroup, err
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

func convertRuleProtocolCBToAZ(protocol string) (armnetwork.SecurityRuleProtocol, error) {
	switch strings.ToUpper(protocol) {
	case "ALL":
		return armnetwork.SecurityRuleProtocolAsterisk, nil
	case "ICMP":
		return armnetwork.SecurityRuleProtocolIcmp, nil
	case "TCP":
		return armnetwork.SecurityRuleProtocolTCP, nil
	case "UDP":
		return armnetwork.SecurityRuleProtocolUDP, nil
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

func convertRuleDirectionCBToAZ(direction string) (armnetwork.SecurityRuleDirection, error) {
	if strings.ToLower(direction) == "inbound" {
		return armnetwork.SecurityRuleDirectionInbound, nil
	}
	if strings.ToLower(direction) == "outbound" {
		return armnetwork.SecurityRuleDirectionOutbound, nil
	}
	return "", errors.New("invalid rule Direction")
}

func convertRuleDirectionAZToCB(direction armnetwork.SecurityRuleDirection) (string, error) {
	if direction == armnetwork.SecurityRuleDirectionInbound {
		return "inbound", nil
	}
	if direction == armnetwork.SecurityRuleDirectionOutbound {
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

func addCBDefaultRule(azureSGRuleList []*armnetwork.SecurityRule) ([]*armnetwork.SecurityRule, error) {
	outboundPriority := initPriority
	var addCBDefaultRuleList []*armnetwork.SecurityRule
	for _, sgRule := range azureSGRuleList {
		if *sgRule.Properties.Access == armnetwork.SecurityRuleAccessAllow {
			if *sgRule.Properties.Direction == armnetwork.SecurityRuleDirectionOutbound && outboundPriority < int(*sgRule.Properties.Priority) {
				outboundPriority = int(*sgRule.Properties.Priority)
			}
		}
	}

	cbDefaultDenySGRule := &armnetwork.SecurityRule{
		Name: toStrPtr("deny-outbound"),
		Properties: &armnetwork.SecurityRulePropertiesFormat{
			SourceAddressPrefix:      toStrPtr("*"),
			SourcePortRange:          toStrPtr("*"),
			DestinationAddressPrefix: toStrPtr("0.0.0.0/0"),
			DestinationPortRange:     toStrPtr("*"),
			Protocol:                 (*armnetwork.SecurityRuleProtocol)(toStrPtr(string(armnetwork.SecurityRuleProtocolAsterisk))),
			Access:                   (*armnetwork.SecurityRuleAccess)(toStrPtr(string(armnetwork.SecurityRuleAccessDeny))),
			Priority:                 toInt32Ptr(maxPriority),
			Direction:                (*armnetwork.SecurityRuleDirection)(toStrPtr(string(armnetwork.SecurityRuleDirectionOutbound))),
		},
	}
	addCBDefaultRuleList = append(azureSGRuleList, cbDefaultDenySGRule)

	cbDefaultAllowSGRule := &armnetwork.SecurityRule{
		Name: toStrPtr("allow-outbound"),
		Properties: &armnetwork.SecurityRulePropertiesFormat{
			SourceAddressPrefix:      toStrPtr("*"),
			SourcePortRange:          toStrPtr("*"),
			DestinationAddressPrefix: toStrPtr("0.0.0.0/0"),
			DestinationPortRange:     toStrPtr("*"),
			Protocol:                 (*armnetwork.SecurityRuleProtocol)(toStrPtr(string(armnetwork.SecurityRuleProtocolAsterisk))),
			Access:                   (*armnetwork.SecurityRuleAccess)(toStrPtr(string(armnetwork.SecurityRuleAccessAllow))),
			Priority:                 toInt32Ptr(outboundPriority + 1),
			Direction:                (*armnetwork.SecurityRuleDirection)(toStrPtr(string(armnetwork.SecurityRuleDirectionOutbound))),
		},
	}
	protocols := convertRuleProtocolAZToCB(fmt.Sprint(*cbDefaultAllowSGRule.Properties.Protocol))
	fromPort, toPort := convertRulePortRangeAZToCB(*cbDefaultAllowSGRule.Properties.DestinationPortRange, protocols)
	cbDirection, err := convertRuleDirectionAZToCB(*cbDefaultAllowSGRule.Properties.Direction)
	if err != nil {
		return nil, err
	}
	cidr := convertRuleCIDRAZToCB(*cbDefaultAllowSGRule.Properties.SourceAddressPrefix)
	cbDefaultAllowSGRuleInfo := irs.SecurityRuleInfo{
		IPProtocol: protocols,
		Direction:  cbDirection,
		CIDR:       cidr,
		FromPort:   fromPort,
		ToPort:     toPort,
	}
	addAllowDefaultRule := false
	for _, sgRule := range azureSGRuleList {
		if *sgRule.Properties.Access == armnetwork.SecurityRuleAccessAllow {
			protocols := convertRuleProtocolAZToCB(fmt.Sprint(*sgRule.Properties.Protocol))
			fromPort, toPort := convertRulePortRangeAZToCB(*sgRule.Properties.DestinationPortRange, protocols)
			direction, err := convertRuleDirectionAZToCB(*sgRule.Properties.Direction)
			if err != nil {
				return nil, err
			}
			cidr := convertRuleCIDRAZToCB(*sgRule.Properties.SourceAddressPrefix)
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

	return addCBDefaultRuleList, nil
}

func generateRuleName(direct string) string {
	return fmt.Sprintf("%s-rules-%s", direct, strconv.FormatInt(rand.Int63n(100000), 10))
}

func convertRuleInfoAZToCB(rawRule *armnetwork.SecurityRule) (irs.SecurityRuleInfo, error) {
	protocols := convertRuleProtocolAZToCB(fmt.Sprint(*rawRule.Properties.Protocol))
	fromPort, toPort := convertRulePortRangeAZToCB(*rawRule.Properties.DestinationPortRange, protocols)
	direction, err := convertRuleDirectionAZToCB(*rawRule.Properties.Direction)
	if err != nil {
		return irs.SecurityRuleInfo{}, err
	}
	cidr := convertRuleCIDRAZToCB(*rawRule.Properties.SourceAddressPrefix)
	if *rawRule.Properties.Direction == armnetwork.SecurityRuleDirectionInbound {
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

func convertRuleInfoCBToAZ(rule irs.SecurityRuleInfo, priority int) (armnetwork.SecurityRule, error) {
	protocol, err := convertRuleProtocolCBToAZ(rule.IPProtocol)
	if err != nil {
		return armnetwork.SecurityRule{}, err
	}
	portRange, err := convertRulePortRangeCBToAZ(rule.FromPort, rule.ToPort, rule.IPProtocol)
	if err != nil {
		return armnetwork.SecurityRule{}, err
	}
	direction, err := convertRuleDirectionCBToAZ(rule.Direction)
	if err != nil {
		return armnetwork.SecurityRule{}, err
	}

	if direction == armnetwork.SecurityRuleDirectionInbound {
		sgRuleInfo := armnetwork.SecurityRule{
			Name: toStrPtr(fmt.Sprintf("%s-%s-%s-%s", generateRuleName(rule.Direction), rule.FromPort, rule.ToPort, rule.IPProtocol)),
			Properties: &armnetwork.SecurityRulePropertiesFormat{
				SourceAddressPrefix:      toStrPtr(rule.CIDR),
				SourcePortRange:          toStrPtr("*"),
				DestinationAddressPrefix: toStrPtr("*"),
				DestinationPortRange:     toStrPtr(portRange),
				Protocol:                 &protocol,
				Access:                   (*armnetwork.SecurityRuleAccess)(toStrPtr(string(armnetwork.SecurityRuleAccessAllow))),
				Priority:                 toInt32Ptr(priority),
				Direction:                &direction,
			},
		}
		return sgRuleInfo, nil
	} else {
		sgRuleInfo := armnetwork.SecurityRule{
			Name: toStrPtr(fmt.Sprintf("%s-%s-%s-%s", generateRuleName(rule.Direction), rule.FromPort, rule.ToPort, rule.IPProtocol)),
			Properties: &armnetwork.SecurityRulePropertiesFormat{
				SourceAddressPrefix:      toStrPtr("*"),
				SourcePortRange:          toStrPtr("*"),
				DestinationAddressPrefix: toStrPtr(rule.CIDR),
				DestinationPortRange:     toStrPtr(portRange),
				Protocol:                 &protocol,
				Access:                   (*armnetwork.SecurityRuleAccess)(toStrPtr(string(armnetwork.SecurityRuleAccessAllow))),
				Priority:                 toInt32Ptr(priority),
				Direction:                &direction,
			},
		}
		return sgRuleInfo, nil
	}

}

func convertRuleInfoListCBToAZ(rules []irs.SecurityRuleInfo) ([]*armnetwork.SecurityRule, error) {
	var azureSGRuleList []*armnetwork.SecurityRule
	for idx, rule := range rules {
		priorityNum := initPriority + idx
		sgRuleInfo, err := convertRuleInfoCBToAZ(rule, priorityNum)
		if err != nil {
			return nil, err
		}
		azureSGRuleList = append(azureSGRuleList, &sgRuleInfo)
	}
	return azureSGRuleList, nil
}

type securityRuleInfoWithName struct {
	Name     string
	RuleInfo irs.SecurityRuleInfo
}

func getRuleInfoWithNames(rawRules []*armnetwork.SecurityRule) (*[]securityRuleInfoWithName, error) {
	var ruleInfoWithNames []securityRuleInfoWithName
	for _, sgRule := range rawRules {
		if *sgRule.Properties.Access == armnetwork.SecurityRuleAccessAllow {
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

func getAddAzureRules(baseRawRules []*armnetwork.SecurityRule, addRuleInfo *[]irs.SecurityRuleInfo) (*[]armnetwork.SecurityRule, error) {
	inboundPriority := initPriority
	outboundPriority := initPriority
	for _, sgRule := range baseRawRules {
		if *sgRule.Properties.Access == armnetwork.SecurityRuleAccessAllow {
			if *sgRule.Properties.Direction == armnetwork.SecurityRuleDirectionInbound && inboundPriority < int(*sgRule.Properties.Priority) {
				inboundPriority = int(*sgRule.Properties.Priority)
			}
			if *sgRule.Properties.Direction == armnetwork.SecurityRuleDirectionOutbound && outboundPriority < int(*sgRule.Properties.Priority) {
				outboundPriority = int(*sgRule.Properties.Priority)
			}
		}
	}
	var azureSGRuleList []armnetwork.SecurityRule

	for idx, rule := range *addRuleInfo {
		priorityNum := initPriority + 1 + idx
		if strings.ToLower(rule.Direction) == "inbound" {
			priorityNum = inboundPriority + 1 + idx
			inboundPriority = inboundPriority + 1 + idx
		}
		if strings.ToLower(rule.Direction) == "outbound" {
			priorityNum = outboundPriority + 1 + idx
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

func (securityHandler *AzureSecurityHandler) ListIID() ([]*irs.IID, error) {
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, SecurityGroup, "ListIID()")
	start := call.Start()

	var iidList []*irs.IID

	pager := securityHandler.Client.NewListPager(securityHandler.Region.Region, nil)

	for pager.More() {
		page, err := pager.NextPage(securityHandler.Ctx)
		if err != nil {
			err = errors.New(fmt.Sprintf("Failed to List Security. err = %s", err))
			cblogger.Error(err.Error())
			LoggingError(hiscallInfo, err)
			return make([]*irs.IID, 0), err
		}

		for _, securityGroup := range page.Value {
			var iid irs.IID

			if securityGroup.ID != nil {
				iid.SystemId = *securityGroup.ID
			}
			if securityGroup.Name != nil {
				iid.NameId = *securityGroup.Name
			}

			iidList = append(iidList, &iid)
		}
	}

	LoggingInfo(hiscallInfo, start)

	return iidList, nil
}
