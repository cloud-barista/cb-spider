package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"net/url"
	"strconv"
	"strings"
)

type IbmSecurityHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	VpcService     *vpcv1.VpcV1
	Ctx            context.Context
}

func (securityHandler *IbmSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, securityReqInfo.IId.NameId, "CreateSecurity()")
	start := call.Start()

	// req 체크
	err := checkSecurityReqInfo(securityReqInfo)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Security. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}
	exist, err := existSecurityGroup(securityReqInfo.IId, securityHandler.VpcService, securityHandler.Ctx)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Security. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	} else if exist {
		createErr := errors.New(fmt.Sprintf("Failed to Create Security. err = The Security name %s already exists", securityReqInfo.IId.NameId))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}
	vpc, err := getRawVPC(securityReqInfo.VpcIID, securityHandler.VpcService, securityHandler.Ctx)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Security. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}
	options := &vpcv1.CreateSecurityGroupOptions{}
	options.SetVPC(&vpcv1.VPCIdentity{
		ID: vpc.ID,
	})
	options.SetName(securityReqInfo.IId.NameId)
	securityGroup, _, err := securityHandler.VpcService.CreateSecurityGroupWithContext(securityHandler.Ctx, options)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Security. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}
	securityGroupRulePrototypes, err := convertCBRuleInfoToIbmRule(*securityReqInfo.SecurityRules)
	_ = addDefaultOutBoundRule(*securityReqInfo.SecurityRules, securityGroupRulePrototypes)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Security. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}
	for _, sgPrototype := range *securityGroupRulePrototypes {
		ruleOptions := &vpcv1.CreateSecurityGroupRuleOptions{}
		ruleOptions.SetSecurityGroupID(*securityGroup.ID)
		ruleOptions.SetSecurityGroupRulePrototype(&sgPrototype)
		_, _, err := securityHandler.VpcService.CreateSecurityGroupRuleWithContext(securityHandler.Ctx, ruleOptions)
		if err != nil {
			options := &vpcv1.DeleteSecurityGroupOptions{}
			options.SetID(*securityGroup.ID)
			_, deleteError := securityHandler.VpcService.DeleteSecurityGroupWithContext(securityHandler.Ctx, options)
			if deleteError != nil {
				err = errors.New(err.Error() + deleteError.Error())
			}
			createErr := errors.New(fmt.Sprintf("Failed to Create Security. err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.SecurityInfo{}, createErr
		}
	}

	rawSecurityGroup, err := getRawSecurityGroup(irs.IID{SystemId: *securityGroup.ID}, securityHandler.VpcService, securityHandler.Ctx)
	if err != nil {
		options := &vpcv1.DeleteSecurityGroupOptions{}
		options.SetID(*securityGroup.ID)
		_, deleteError := securityHandler.VpcService.DeleteSecurityGroupWithContext(securityHandler.Ctx, options)
		if deleteError != nil {
			err = errors.New(err.Error() + deleteError.Error())
		}
		createErr := errors.New(fmt.Sprintf("Failed to Create Security. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}
	securityGroupInfo, err := setSecurityGroupInfo(rawSecurityGroup)
	if err != nil {
		options := &vpcv1.DeleteSecurityGroupOptions{}
		options.SetID(*securityGroup.ID)
		_, deleteError := securityHandler.VpcService.DeleteSecurityGroupWithContext(securityHandler.Ctx, options)
		if deleteError != nil {
			err = errors.New(err.Error() + deleteError.Error())
		}
		createErr := errors.New(fmt.Sprintf("Failed to Create Security. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}
	LoggingInfo(hiscallInfo, start)
	return securityGroupInfo, nil
}

func (securityHandler *IbmSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, "SECURITYGROUP", "ListSecurity()")
	start := call.Start()
	options := &vpcv1.ListSecurityGroupsOptions{}
	securityGroups, _, err := securityHandler.VpcService.ListSecurityGroupsWithContext(securityHandler.Ctx, options)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get SecurityList. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	var securityGroupList []*irs.SecurityInfo
	for {
		for _, securityGroup := range securityGroups.SecurityGroups {
			securityInfo, err := setSecurityGroupInfo(securityGroup)
			if err != nil {
				getErr := errors.New(fmt.Sprintf("Failed to Get SecurityList. err = %s", err.Error()))
				cblogger.Error(getErr.Error())
				LoggingError(hiscallInfo, getErr)
				return nil, getErr
			}
			securityGroupList = append(securityGroupList, &securityInfo)
		}
		nextstr, _ := getSecurityGroupNextHref(securityGroups.Next)
		if nextstr != "" {
			options2 := &vpcv1.ListSecurityGroupsOptions{
				Start: core.StringPtr(nextstr),
			}
			securityGroups, _, err = securityHandler.VpcService.ListSecurityGroupsWithContext(securityHandler.Ctx, options2)
			if err != nil {
				getErr := errors.New(fmt.Sprintf("Failed to Get SecurityList. err = %s", err.Error()))
				cblogger.Error(getErr.Error())
				LoggingError(hiscallInfo, getErr)
				return nil, getErr
			}
		} else {
			break
		}
	}
	LoggingInfo(hiscallInfo, start)
	return securityGroupList, nil
}

func (securityHandler *IbmSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, securityIID.NameId, "GetSecurity()")
	start := call.Start()

	err := checkSecurityGroupIID(securityIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Security. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}
	securityGroup, err := getRawSecurityGroup(securityIID, securityHandler.VpcService, securityHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Security. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}
	securityGroupInfo, err := setSecurityGroupInfo(securityGroup)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Security. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return securityGroupInfo, nil
}

func (securityHandler *IbmSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, securityIID.NameId, "DeleteSecurity()")
	start := call.Start()

	err := checkSecurityGroupIID(securityIID)

	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Security. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	securityGroup, err := getRawSecurityGroup(securityIID, securityHandler.VpcService, securityHandler.Ctx)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Security. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	options := &vpcv1.DeleteSecurityGroupOptions{}
	options.SetID(*securityGroup.ID)
	res, err := securityHandler.VpcService.DeleteSecurityGroupWithContext(securityHandler.Ctx, options)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Security. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	if res.StatusCode == 204 {
		LoggingInfo(hiscallInfo, start)
		return true, nil
	} else {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Security. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
}

func existSecurityGroup(securityIID irs.IID, vpcService *vpcv1.VpcV1, ctx context.Context) (bool, error) {
	if securityIID.NameId != "" {
		options := &vpcv1.ListSecurityGroupsOptions{}
		securityGroups, _, err := vpcService.ListSecurityGroupsWithContext(ctx, options)
		if err != nil {
			return false, err
		}
		for {
			for _, securityGroup := range securityGroups.SecurityGroups {
				if *securityGroup.Name == securityIID.NameId {
					return true, nil
				}
			}
			nextstr, _ := getSecurityGroupNextHref(securityGroups.Next)
			if nextstr != "" {
				options2 := &vpcv1.ListSecurityGroupsOptions{
					Start: core.StringPtr(nextstr),
				}
				securityGroups, _, err = vpcService.ListSecurityGroupsWithContext(ctx, options2)
				if err != nil {
					return false, err
				}
			} else {
				break
			}
		}
		return false, nil
	} else {
		err := errors.New("invalid securityIID")
		return false, err
	}
}

func getRawSecurityGroup(securityIID irs.IID, vpcService *vpcv1.VpcV1, ctx context.Context) (vpcv1.SecurityGroup, error) {
	if securityIID.SystemId == "" {
		options := &vpcv1.ListSecurityGroupsOptions{}
		securityGroups, _, err := vpcService.ListSecurityGroupsWithContext(ctx, options)
		if err != nil {
			return vpcv1.SecurityGroup{}, err
		}
		for {
			for _, securityGroup := range securityGroups.SecurityGroups {
				if *securityGroup.Name == securityIID.NameId {
					return securityGroup, nil
				}
			}
			nextstr, _ := getSecurityGroupNextHref(securityGroups.Next)
			if nextstr != "" {
				options2 := &vpcv1.ListSecurityGroupsOptions{
					Start: core.StringPtr(nextstr),
				}
				securityGroups, _, err = vpcService.ListSecurityGroupsWithContext(ctx, options2)
				if err != nil {
					return vpcv1.SecurityGroup{}, err
				}
			} else {
				break
			}
		}
		return vpcv1.SecurityGroup{}, errors.New(fmt.Sprintf("not found SecurityGroup %s", securityIID.NameId))
	} else {
		options := &vpcv1.GetSecurityGroupOptions{}
		options.SetID(securityIID.SystemId)
		sg, _, err := vpcService.GetSecurityGroupWithContext(ctx, options)
		if err != nil {
			return vpcv1.SecurityGroup{}, err
		}
		return *sg, nil
	}
}

func setSecurityGroupInfo(securityGroup vpcv1.SecurityGroup) (irs.SecurityInfo, error) {
	securityInfo := irs.SecurityInfo{
		IId: irs.IID{
			NameId:   *securityGroup.Name,
			SystemId: *securityGroup.ID,
		},
		VpcIID: irs.IID{
			NameId:   *securityGroup.VPC.Name,
			SystemId: *securityGroup.VPC.ID,
		},
	}
	ruleList, err := setRule(securityGroup)
	if err != nil {
		return irs.SecurityInfo{}, err
	}
	securityInfo.SecurityRules = &ruleList
	return securityInfo, nil
}

func setRule(securityGroup vpcv1.SecurityGroup) ([]irs.SecurityRuleInfo, error) {
	var ruleList []irs.SecurityRuleInfo
	for _, rule := range securityGroup.Rules {
		ruleInfo, err := ConvertIbmRuleToCBRuleInfo(rule)
		if err != nil {
			return nil, err
		}
		ruleList = append(ruleList, *ruleInfo)
	}
	return ruleList, nil
}

func getSecurityGroupNextHref(next *vpcv1.SecurityGroupCollectionNext) (string, error) {
	if next != nil {
		href := *next.Href
		u, err := url.Parse(href)
		if err != nil {
			return "", err
		}
		paramMap, _ := url.ParseQuery(u.RawQuery)
		if paramMap != nil {
			safe := paramMap["start"]
			if safe != nil && len(safe) > 0 {
				return safe[0], nil
			}
		}
	}
	return "", errors.New("NOT NEXT")
}

func checkSecurityGroupIID(securityIID irs.IID) error {
	if securityIID.SystemId == "" && securityIID.NameId == "" {
		err := errors.New("invalid IID")
		return err
	}
	return nil
}

func checkSecurityReqInfo(securityReqInfo irs.SecurityReqInfo) error {
	if securityReqInfo.IId.NameId == "" {
		return errors.New("invalid securityReqInfo IID")
	}
	if securityReqInfo.VpcIID.NameId == "" && securityReqInfo.VpcIID.SystemId == "" {
		return errors.New("invalid securityReqInfo VpcIID")
	}
	return nil
}

func (securityHandler *IbmSecurityHandler) AddRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, sgIID.NameId, "GetSecurity()")
	start := call.Start()

	err := checkSecurityGroupIID(sgIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}
	securityGroup, err := getRawSecurityGroup(sgIID, securityHandler.VpcService, securityHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}
	securityGroupInfo, err := setSecurityGroupInfo(securityGroup)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}
	var updateRules []irs.SecurityRuleInfo
	// 추가될 Rule 판단
	for _, newRule := range *securityRules {
		existCheck := false
		for _, baseRule := range *securityGroupInfo.SecurityRules {
			if equalsRule(newRule, baseRule) {
				existCheck = true
				break
			}
		}
		if existCheck {
			b, err := json.Marshal(newRule)
			err = errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err already Exist Rule : %s", string(b)))
			cblogger.Error(err.Error())
			LoggingError(hiscallInfo, err)
			return irs.SecurityInfo{}, err
		}
		updateRules = append(updateRules, newRule)
	}
	securityGroupRulePrototypes, err := convertCBRuleInfoToIbmRule(updateRules)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}
	for _, sgPrototype := range *securityGroupRulePrototypes {
		ruleOptions := &vpcv1.CreateSecurityGroupRuleOptions{}
		ruleOptions.SetSecurityGroupID(*securityGroup.ID)
		ruleOptions.SetSecurityGroupRulePrototype(&sgPrototype)
		_, _, err := securityHandler.VpcService.CreateSecurityGroupRuleWithContext(securityHandler.Ctx, ruleOptions)
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.SecurityInfo{}, createErr
		}
	}

	newSecurityGroup, err := getRawSecurityGroup(sgIID, securityHandler.VpcService, securityHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}
	newSecurityGroupInfo, err := setSecurityGroupInfo(newSecurityGroup)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)
	return newSecurityGroupInfo, nil
}

func (securityHandler *IbmSecurityHandler) RemoveRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (bool, error) {
	hiscallInfo := GetCallLogScheme(securityHandler.Region, call.SECURITYGROUP, sgIID.NameId, "RemoveRules()")
	start := call.Start()

	err := checkSecurityGroupIID(sgIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Remove SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}
	securityGroup, err := getRawSecurityGroup(sgIID, securityHandler.VpcService, securityHandler.Ctx)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Remove SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}

	ruleWithIds, err := getRuleInfoWithId(&securityGroup.Rules)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Remove SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}
	var deleteRuleIds []string

	for _, delRule := range *securityRules {
		existCheck := false
		for _, baseRuleWithId := range *ruleWithIds {
			if equalsRule(baseRuleWithId.RuleInfo, delRule) {
				existCheck = true
				deleteRuleIds = append(deleteRuleIds, baseRuleWithId.Id)
				break
			}
		}
		if !existCheck {
			b, err := json.Marshal(delRule)
			err = errors.New(fmt.Sprintf("Failed to Remove SecurityGroup Rules. err = not Exist Rule : %s", string(b)))
			cblogger.Error(err.Error())
			LoggingError(hiscallInfo, err)
			return false, err
		}
	}

	for _, deleteRuleId := range deleteRuleIds {
		options := &vpcv1.DeleteSecurityGroupRuleOptions{}
		options.SetSecurityGroupID(*securityGroup.ID)
		options.SetID(deleteRuleId)
		_, err := securityHandler.VpcService.DeleteSecurityGroupRuleWithContext(securityHandler.Ctx, options)
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

type securityRuleInfoWithId struct {
	Id       string
	RuleInfo irs.SecurityRuleInfo
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

func getRuleInfoWithId(rawRules *[]vpcv1.SecurityGroupRuleIntf) (*[]securityRuleInfoWithId, error) {
	var arr []securityRuleInfoWithId
	for _, rule := range *rawRules {
		jsonRuleBytes, err := json.Marshal(rule)
		if err != nil {
			return nil, err
		}
		var ru vpcv1.SecurityGroupRule
		_ = json.Unmarshal(jsonRuleBytes, &ru)
		if ru.ID == nil {
			return nil, errors.New("securityGroup Rule marshal failed")
		}
		ruleInfo, err := ConvertIbmRuleToCBRuleInfo(rule)
		if err != nil {
			return nil, err
		}
		arr = append(arr, securityRuleInfoWithId{Id: *ru.ID, RuleInfo: *ruleInfo})
	}
	return &arr, nil
}

func ConvertIbmRuleToCBRuleInfo(rule vpcv1.SecurityGroupRuleIntf) (*irs.SecurityRuleInfo, error) {
	jsonRuleBytes, err := json.Marshal(rule)
	if err != nil {
		return nil, err
	}
	jsonRuleMap := make(map[string]json.RawMessage)
	unmarshalErr := json.Unmarshal(jsonRuleBytes, &jsonRuleMap)
	if unmarshalErr != nil {
		return nil, err
	}
	remoteJson := jsonRuleMap["remote"]

	var remote vpcv1.SecurityGroupRuleRemote
	unmarshalErr = json.Unmarshal(remoteJson, &remote)
	if unmarshalErr != nil {
		return nil, err
	}
	var ruleProtocolAll vpcv1.SecurityGroupRulePrototypeSecurityGroupRuleProtocolAll
	_ = json.Unmarshal(jsonRuleBytes, &ruleProtocolAll)
	protocol := convertRuleProtocolIBMToCB(*ruleProtocolAll.Protocol)
	cidr := "0.0.0.0/0"
	if remote.CIDRBlock != nil {
		cidr = *remote.CIDRBlock
	}
	if protocol == "tcp" || protocol == "udp" {
		var ruleProtocolTcpUdp vpcv1.SecurityGroupRulePrototypeSecurityGroupRuleProtocolTcpudp
		_ = json.Unmarshal(jsonRuleBytes, &ruleProtocolTcpUdp)
		from, to := convertRulePortRangeIBMToCB(*ruleProtocolTcpUdp.PortMin, *ruleProtocolTcpUdp.PortMax)
		ruleInfo := irs.SecurityRuleInfo{
			IPProtocol: protocol,
			Direction:  *ruleProtocolTcpUdp.Direction,
			FromPort:   from,
			ToPort:     to,
			CIDR:       cidr,
		}
		return &ruleInfo, nil
	} else if protocol == "icmp" {
		var ruleProtocolIcmp vpcv1.SecurityGroupRulePrototypeSecurityGroupRuleProtocolIcmp
		_ = json.Unmarshal(jsonRuleBytes, &ruleProtocolIcmp)
		ruleInfo := irs.SecurityRuleInfo{
			IPProtocol: protocol,
			Direction:  *ruleProtocolIcmp.Direction,
			CIDR:       cidr,
			FromPort:   "-1",
			ToPort:     "-1",
		}
		return &ruleInfo, nil
	} else {
		ruleInfo := irs.SecurityRuleInfo{
			IPProtocol: protocol,
			Direction:  *ruleProtocolAll.Direction,
			CIDR:       cidr,
			FromPort:   "-1",
			ToPort:     "-1",
		}
		return &ruleInfo, nil
	}
}

func ModifyVPCDefaultRule(rules []vpcv1.SecurityGroupRuleIntf, sgIId irs.IID, vpcService *vpcv1.VpcV1, ctx context.Context) error {
	for _, rule := range rules {
		jsonRuleBytes, err := json.Marshal(rule)
		if err != nil {
			return err
		}
		jsonRuleMap := make(map[string]json.RawMessage)
		unmarshalErr := json.Unmarshal(jsonRuleBytes, &jsonRuleMap)
		if unmarshalErr != nil {
			return err
		}
		remoteJson := jsonRuleMap["remote"]

		var remote vpcv1.SecurityGroupRuleRemote
		unmarshalErr = json.Unmarshal(remoteJson, &remote)
		if unmarshalErr != nil {
			return err
		}
		if remote.Name != nil && *remote.Name == sgIId.NameId {
			continue
		}
		var ruleProtocolAll vpcv1.SecurityGroupRule
		_ = json.Unmarshal(jsonRuleBytes, &ruleProtocolAll)
		if ruleProtocolAll.ID != nil && *ruleProtocolAll.ID != "" {
			options := &vpcv1.DeleteSecurityGroupRuleOptions{}
			options.SetSecurityGroupID(sgIId.SystemId)
			options.SetID(*ruleProtocolAll.ID)
			_, err = vpcService.DeleteSecurityGroupRuleWithContext(ctx, options)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func convertRuleProtocolIBMToCB(protocol string) string {
	return strings.ToLower(protocol)
}

func convertRuleProtocolCBToIBM(protocol string) (string, error) {
	switch strings.ToUpper(protocol) {
	case "ALL":
		return strings.ToLower(protocol), nil
	case "ICMP", "TCP", "UDP":
		return strings.ToLower(protocol), nil
	}
	return "", errors.New("invalid Rule Protocol")
}

func convertRulePortRangeIBMToCB(min int64, max int64) (from string, to string) {
	return strconv.FormatInt(min, 10), strconv.FormatInt(max, 10)
}

func convertRulePortRangeCBToIBM(from string, to string) (min int64, max int64, err error) {
	if from == "" || to == "" {
		return 0, 0, errors.New("invalid Rule PortRange")
	}
	fromInt, err := strconv.ParseInt(from, 10, 64)
	if err != nil {
		return 0, 0, errors.New("invalid Rule PortRange")
	}
	toInt, err := strconv.ParseInt(to, 10, 64)
	if err != nil {
		return 0, 0, errors.New("invalid Rule PortRange")
	}
	if fromInt == -1 || toInt == -1 {
		return int64(1), int64(65535), nil
	}
	if fromInt > 65535 || fromInt < -1 || toInt > 65535 || toInt < -1 {
		return 0, 0, errors.New("invalid Rule PortRange")
	}
	if fromInt == toInt {
		return fromInt, fromInt, nil
	} else {
		return fromInt, toInt, nil
	}
}

func convertCBRuleInfoToIbmRule(rules []irs.SecurityRuleInfo) (*[]vpcv1.SecurityGroupRulePrototype, error) {
	var IbmSGRuleList []vpcv1.SecurityGroupRulePrototype
	for _, securityRule := range rules {
		protocol, err := convertRuleProtocolCBToIBM(securityRule.IPProtocol)
		if err != nil {
			return nil, err
		}
		if protocol == "tcp" || protocol == "udp" {
			portMin, portMax, err := convertRulePortRangeCBToIBM(securityRule.FromPort, securityRule.ToPort)
			if err != nil {
				return nil, err
			}
			IbmSGRuleList = append(IbmSGRuleList, vpcv1.SecurityGroupRulePrototype{
				Direction: core.StringPtr(strings.ToLower(securityRule.Direction)),
				Protocol:  core.StringPtr(protocol),
				PortMax:   core.Int64Ptr(portMax),
				PortMin:   core.Int64Ptr(portMin),
				IPVersion: core.StringPtr("ipv4"),
				Remote: &vpcv1.SecurityGroupRuleRemotePrototype{
					CIDRBlock: &securityRule.CIDR,
				},
			})
		} else {
			IbmSGRuleList = append(IbmSGRuleList, vpcv1.SecurityGroupRulePrototype{
				Direction: core.StringPtr(strings.ToLower(securityRule.Direction)),
				Protocol:  core.StringPtr(protocol),
				IPVersion: core.StringPtr("ipv4"),
				Remote: &vpcv1.SecurityGroupRuleRemotePrototype{
					CIDRBlock: &securityRule.CIDR,
				},
			})
		}
	}
	return &IbmSGRuleList, nil
}
func addDefaultOutBoundRule(baseRuleInfos []irs.SecurityRuleInfo, addRules *[]vpcv1.SecurityGroupRulePrototype) error {
	defaultRuleInfo := irs.SecurityRuleInfo{
		CIDR:       "0.0.0.0/0",
		IPProtocol: "all",
		FromPort:   "-1",
		ToPort:     "-1",
		Direction:  "outbound",
	}
	addCheck := true
	for _, rule := range baseRuleInfos {
		if equalsRule(rule, defaultRuleInfo) {
			addCheck = false
		}
	}
	if addCheck {
		*addRules = append(*addRules, vpcv1.SecurityGroupRulePrototype{
			Direction: core.StringPtr(strings.ToLower(defaultRuleInfo.Direction)),
			Protocol:  core.StringPtr(defaultRuleInfo.IPProtocol),
			IPVersion: core.StringPtr("ipv4"),
			Remote: &vpcv1.SecurityGroupRuleRemotePrototype{
				CIDRBlock: &defaultRuleInfo.CIDR,
			},
		})
	}
	return nil
}
