package resources

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-02-01/network"
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
		protocols := convertRuleProtocolAZToCB(fmt.Sprint(sgRule.Protocol))
		fromPort, toPort := convertRulePortRangeAZToCB(*sgRule.DestinationPortRange, protocols)
		ruleInfo := irs.SecurityRuleInfo{
			IPProtocol: protocols,
			Direction:  strings.ToLower(fmt.Sprint(sgRule.Direction)),
			CIDR:       *sgRule.SourceAddressPrefix,
			FromPort:   fromPort,
			ToPort:     toPort,
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
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}

	sgRuleList, err := convertRuleInfoCBToAZ(*securityReqInfo.SecurityRules)

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
	rawSecurityGroup, err := securityHandler.getRawSecurityGroup(securityIID)
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
	security, err := securityHandler.getRawSecurityGroup(sgIID)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}

	securityInfo := securityHandler.setterSec(*security)

	var updateRules []irs.SecurityRuleInfo
	for _, baseRule := range *securityInfo.SecurityRules {
		updateRules = append(updateRules, baseRule)
	}

	for _, newRule := range *securityRules {
		chk := true
		for _, baseRule := range *securityInfo.SecurityRules {
			if equalsRule(newRule, baseRule) {
				chk = false
				break
			}
		}
		if chk {
			updateRules = append(updateRules, newRule)
		}
	}

	addSGRuleList, err := convertRuleInfoCBToAZ(updateRules)

	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}

	updateOpts := network.SecurityGroup{
		SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
			SecurityRules: addSGRuleList,
		},
		Location: &securityHandler.Region.Region,
	}

	future, err := securityHandler.Client.CreateOrUpdate(securityHandler.Ctx, securityHandler.Region.ResourceGroup, sgIID.NameId, updateOpts)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}

	err = future.WaitForCompletionRef(securityHandler.Ctx, securityHandler.Client.Client)
	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return irs.SecurityInfo{}, err
	}

	// 변된 SecurityGroup 정보 리턴
	updatedSecurity, err := securityHandler.getRawSecurityGroup(sgIID)
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

	security, err := securityHandler.getRawSecurityGroup(sgIID)

	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return false, err
	}
	securityInfo := securityHandler.setterSec(*security)

	var updateRules []irs.SecurityRuleInfo

	for _, baseRule := range *securityInfo.SecurityRules {
		chk := true
		for _, delRule := range *securityRules {
			if equalsRule(baseRule, delRule) {
				chk = false
				break
			}
		}
		if chk {
			updateRules = append(updateRules, baseRule)
		}
	}

	addSGRuleList, err := convertRuleInfoCBToAZ(updateRules)

	if err != nil {
		cblogger.Error(err.Error())
		LoggingError(hiscallInfo, err)
		return false, err
	}

	updateOpts := network.SecurityGroup{
		SecurityGroupPropertiesFormat: &network.SecurityGroupPropertiesFormat{
			SecurityRules: addSGRuleList,
		},
		Location: &securityHandler.Region.Region,
	}

	future, err := securityHandler.Client.CreateOrUpdate(securityHandler.Ctx, securityHandler.Region.ResourceGroup, sgIID.NameId, updateOpts)
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

func (securityHandler *AzureSecurityHandler) getRawSecurityGroup(sgIID irs.IID) (*network.SecurityGroup, error) {
	if sgIID.SystemId == "" && sgIID.NameId == "" {
		return nil, errors.New("invalid IID")
	}
	if sgIID.NameId == "" {
		result, err := securityHandler.Client.List(securityHandler.Ctx, securityHandler.Region.ResourceGroup)
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
		security, err := securityHandler.Client.Get(securityHandler.Ctx, securityHandler.Region.ResourceGroup, sgIID.NameId, "")
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
	if strings.ToUpper(protocol) == "ICMP" {
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

func convertRuleInfoCBToAZ(rules []irs.SecurityRuleInfo) (*[]network.SecurityRule, error) {
	var azureSGRuleList []network.SecurityRule
	var priorityNum int32
	for idx, rule := range rules {
		protocol, err := convertRuleProtocolCBToAZ(rule.IPProtocol)
		if err != nil {
			return nil, err
		}
		portRange, err := convertRulePortRangeCBToAZ(rule.FromPort, rule.ToPort, rule.IPProtocol)
		if err != nil {
			return nil, err
		}
		priorityNum = int32(300 + idx*100)
		sgRuleInfo := network.SecurityRule{
			Name: to.StringPtr(fmt.Sprintf("%s-rules-%d", rule.Direction, idx+1)),
			SecurityRulePropertiesFormat: &network.SecurityRulePropertiesFormat{
				SourceAddressPrefix:      to.StringPtr(rule.CIDR),
				SourcePortRange:          to.StringPtr("*"),
				DestinationAddressPrefix: to.StringPtr("*"),
				DestinationPortRange:     to.StringPtr(portRange),
				Protocol:                 network.SecurityRuleProtocol(protocol),
				Access:                   network.SecurityRuleAccess("Allow"),
				Priority:                 to.Int32Ptr(priorityNum),
				Direction:                network.SecurityRuleDirection(rule.Direction),
			},
		}
		azureSGRuleList = append(azureSGRuleList, sgRuleInfo)
	}
	return &azureSGRuleList, nil
}
