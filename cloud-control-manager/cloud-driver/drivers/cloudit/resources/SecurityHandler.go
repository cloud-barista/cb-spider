package resources

import (
	"errors"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client"
	"github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/drivers/cloudit/client/iam/securitygroup"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

const (
	SecurityGroup = "SECURITYGROUP"
	NULL          = ""
	DefaultCIDR   = "0.0.0.0/0"
	DefaultPort   = "0"
)

type ClouditSecurityHandler struct {
	CredentialInfo idrv.CredentialInfo
	Client         *client.RestClient
}

func (securityHandler *ClouditSecurityHandler) setterSecGroup(secGroup securitygroup.SecurityGroupInfo) (irs.SecurityInfo, error) {

	secInfo := irs.SecurityInfo{
		IId: irs.IID{
			NameId:   secGroup.Name,
			SystemId: secGroup.ID,
		},
		SecurityRules: nil,
	}

	secRuleArr := make([]irs.SecurityRuleInfo, len(secGroup.Rules))
	for i, sgRule := range secGroup.Rules {
		secRuleArr[i] = convertRuleInfoCloudItToCB(sgRule)
	}
	secInfo.SecurityRules = &secRuleArr
	VPCHandler := ClouditVPCHandler{
		Client:         securityHandler.Client,
		CredentialInfo: securityHandler.CredentialInfo,
	}
	defaultVPC, err := VPCHandler.GetDefaultVPC()
	if err == nil {
		secInfo.VpcIID = defaultVPC.IId
	}
	return secInfo, nil
}

func (securityHandler *ClouditSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.SECURITYGROUP, securityReqInfo.IId.NameId, "CreateSecurity()")

	// 보안그룹 이름 중복 체크
	securityInfo, _ := securityHandler.getRawSecurityGroup(securityReqInfo.IId)
	if securityInfo != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Security. err = SecurityGroup with name %s already exist", securityReqInfo.IId.NameId))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
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
		createRule, err := convertRuleInfoCBToCloudIt(rule)
		if err != nil {
			createErr := errors.New(fmt.Sprintf("Failed to Create Security. err = %s", err.Error()))
			cblogger.Error(createErr.Error())
			LoggingError(hiscallInfo, createErr)
			return irs.SecurityInfo{}, createErr
		}
		ruleList[i] = createRule
	}
	reqInfo.Rules = ruleList

	createOpts := client.RequestOpts{
		JSONBody:    reqInfo,
		MoreHeaders: authHeader,
	}

	start := call.Start()
	securityGroup, err := securitygroup.Create(securityHandler.Client, &createOpts)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Security. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}
	LoggingInfo(hiscallInfo, start)

	secGroupInfo, err := securityHandler.setterSecGroup(*securityGroup)
	if err != nil {
		createErr := errors.New(fmt.Sprintf("Failed to Create Security. err = %s", err.Error()))
		cblogger.Error(createErr.Error())
		LoggingError(hiscallInfo, createErr)
		return irs.SecurityInfo{}, createErr
	}
	return secGroupInfo, nil
}

func (securityHandler *ClouditSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.SECURITYGROUP, SecurityGroup, "ListSecurity()")

	securityHandler.Client.TokenID = securityHandler.CredentialInfo.AuthToken
	authHeader := securityHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	start := call.Start()
	securityList, err := securitygroup.List(securityHandler.Client, &requestOpts)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get SecurityList. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return nil, getErr
	}
	LoggingInfo(hiscallInfo, start)

	// SecurityGroup Rule 정보 가져오기
	for i, sg := range *securityList {
		sgRules, err := securitygroup.ListRule(securityHandler.Client, sg.ID, &requestOpts)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to Get SecurityList. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return nil, getErr
		}
		(*securityList)[i].Rules = *sgRules
		(*securityList)[i].RulesCount = len(*sgRules)
	}

	resultList := make([]*irs.SecurityInfo, len(*securityList))
	for i, security := range *securityList {
		secInfo, err := securityHandler.setterSecGroup(security)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to Get SecurityList. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return nil, getErr
		}
		resultList[i] = &secInfo
	}
	return resultList, nil
}

func (securityHandler *ClouditSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.SECURITYGROUP, securityIID.NameId, "GetSecurity()")

	// 이름 기준 보안그룹 조회
	start := call.Start()
	securityInfo, err := securityHandler.getRawSecurityGroup(securityIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Security. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)

	securityHandler.Client.TokenID = securityHandler.CredentialInfo.AuthToken
	authHeader := securityHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	// SecurityGroup Rule 정보 가져오기
	sgRules, err := securitygroup.ListRule(securityHandler.Client, securityInfo.ID, &requestOpts)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Security. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}

	(*securityInfo).Rules = *sgRules
	(*securityInfo).RulesCount = len(*sgRules)
	secGroupInfo, err := securityHandler.setterSecGroup(*securityInfo)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Get Security. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}
	return secGroupInfo, nil
}

func (securityHandler *ClouditSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	// log HisCall
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.SECURITYGROUP, securityIID.NameId, "DeleteSecurity()")

	// 이름 기준 보안그룹 조회
	securityInfo, err := securityHandler.getRawSecurityGroup(securityIID)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Security. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}

	securityHandler.Client.TokenID = securityHandler.CredentialInfo.AuthToken
	authHeader := securityHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	// 보안그룹 삭제
	start := call.Start()
	err = securitygroup.Delete(securityHandler.Client, securityInfo.ID, &requestOpts)
	if err != nil {
		delErr := errors.New(fmt.Sprintf("Failed to Delete Security. err = %s", err.Error()))
		cblogger.Error(delErr.Error())
		LoggingError(hiscallInfo, delErr)
		return false, delErr
	}
	LoggingInfo(hiscallInfo, start)

	return true, nil
}

func (securityHandler *ClouditSecurityHandler) listRulesInSG(securityID string) (*[]securitygroup.SecurityGroupRules, error) {
	securityHandler.Client.TokenID = securityHandler.CredentialInfo.AuthToken
	authHeader := securityHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	securityList, err := securitygroup.ListRulesinSG(securityHandler.Client, securityID, &requestOpts)
	if err != nil {
		return nil, err
	}

	return securityList, nil
}

func (securityHandler *ClouditSecurityHandler) AddRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.SECURITYGROUP, sgIID.NameId, "AddRules()")

	// 이름 기준 보안그룹 조회
	start := call.Start()
	securityInfo, err := securityHandler.getRawSecurityGroup(sgIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}
	LoggingInfo(hiscallInfo, start)

	securityHandler.Client.TokenID = securityHandler.CredentialInfo.AuthToken
	authHeader := securityHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	// SecurityGroup Rule 정보 가져오기
	sgRules, err := securitygroup.ListRule(securityHandler.Client, securityInfo.ID, &requestOpts)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}

	(*securityInfo).Rules = *sgRules
	(*securityInfo).RulesCount = len(*sgRules)

	secGroupInfo, err := securityHandler.setterSecGroup(*securityInfo)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}

	var updateRules []irs.SecurityRuleInfo
	for _, newRule := range *securityRules {
		chk := true
		for _, baseRule := range *secGroupInfo.SecurityRules {
			if equalsRule(newRule, baseRule) {
				chk = false
				break
			}
		}
		if chk {
			updateRules = append(updateRules, newRule)
		}
	}
	ruleList := make([]securitygroup.SecurityGroupRules, len(updateRules))
	for i, rule := range updateRules {
		secRuleInfo, err := convertRuleInfoCBToCloudIt(rule)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return irs.SecurityInfo{}, getErr
		}
		ruleList[i] = secRuleInfo
	}
	for _, reqRule := range ruleList {
		createOpts := client.RequestOpts{
			JSONBody:    reqRule,
			MoreHeaders: authHeader,
		}
		_, err := securitygroup.AddRule(securityHandler.Client, secGroupInfo.IId.SystemId, &createOpts, reqRule.Type)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return irs.SecurityInfo{}, getErr
		}
	}
	newSGRules, err := securitygroup.ListRule(securityHandler.Client, securityInfo.ID, &requestOpts)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}

	(*securityInfo).Rules = *newSGRules
	(*securityInfo).RulesCount = len(*newSGRules)

	newSecGroupInfo, err := securityHandler.setterSecGroup(*securityInfo)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Add SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return irs.SecurityInfo{}, getErr
	}
	return newSecGroupInfo, nil
}

func (securityHandler *ClouditSecurityHandler) RemoveRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (bool, error) {
	hiscallInfo := GetCallLogScheme(ClouditRegion, call.SECURITYGROUP, sgIID.NameId, "RemoveRules()")

	// 이름 기준 보안그룹 조회
	start := call.Start()
	securityInfo, err := securityHandler.getRawSecurityGroup(sgIID)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Remove SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}
	LoggingInfo(hiscallInfo, start)

	securityHandler.Client.TokenID = securityHandler.CredentialInfo.AuthToken
	authHeader := securityHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}

	// SecurityGroup Rule 정보 가져오기
	sgRules, err := securitygroup.ListRule(securityHandler.Client, securityInfo.ID, &requestOpts)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Remove SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}

	ruleWithIds, err := getRuleInfoWithIds(sgRules)
	if err != nil {
		getErr := errors.New(fmt.Sprintf("Failed to Remove SecurityGroup Rules. err = %s", err.Error()))
		cblogger.Error(getErr.Error())
		LoggingError(hiscallInfo, getErr)
		return false, getErr
	}

	var deleteRuleIds []string

	for _, newRule := range *securityRules {
		for _, baseRuleWithId := range *ruleWithIds {
			if equalsRule(newRule, baseRuleWithId.RuleInfo) {
				deleteRuleIds = append(deleteRuleIds, baseRuleWithId.Id)
				break
			}
		}
	}
	for _, ruleId := range deleteRuleIds {
		createOpts := client.RequestOpts{
			MoreHeaders: authHeader,
		}
		err := securitygroup.DeleteRule(securityHandler.Client, securityInfo.ID, &createOpts, ruleId)
		if err != nil {
			getErr := errors.New(fmt.Sprintf("Failed to Remove SecurityGroup Rules. err = %s", err.Error()))
			cblogger.Error(getErr.Error())
			LoggingError(hiscallInfo, getErr)
			return false, getErr
		}
	}
	return true, nil
}

func (securityHandler *ClouditSecurityHandler) getRawSecurityGroup(sgIID irs.IID) (*securitygroup.SecurityGroupInfo, error) {
	if sgIID.SystemId == "" && sgIID.NameId == ""{
		return nil, errors.New("invalid IID")
	}
	securityHandler.Client.TokenID = securityHandler.CredentialInfo.AuthToken
	authHeader := securityHandler.Client.AuthenticatedHeaders()

	requestOpts := client.RequestOpts{
		MoreHeaders: authHeader,
	}
	securityList, err := securitygroup.List(securityHandler.Client, &requestOpts)
	if err != nil {
		return nil, err
	}
	if sgIID.SystemId == "" {
		for _, s := range *securityList {
			if strings.EqualFold(s.Name,sgIID.NameId) {
				return &s,nil
			}
		}
	}else{
		for _, s := range *securityList {
			if strings.EqualFold(s.ID,sgIID.SystemId) {
				return &s,nil
			}
		}
	}
	return nil, errors.New("not found SecurityGroup")
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

func convertRuleProtocolCloudItToCB(protocol string) string {
	return strings.ToLower(protocol)
}

func convertRuleProtocolCBToCloudIt(protocol string) (string, error) {
	switch strings.ToUpper(protocol) {
	case "ALL":
		return strings.ToLower("all"), nil
	case "TCP", "UDP":
		return strings.ToLower(protocol), nil
	}
	return "", errors.New("invalid Rule Protocol CloudIt only offers tcp, udp. ")
}

func convertRulePortRangeCloudItToCB(portRange string) (from string, to string) {
	portRangeArr := strings.Split(portRange, "-")
	if len(portRangeArr) != 2 {
		if len(portRangeArr) == 1 && portRange != "*" {
			return portRangeArr[0], portRangeArr[0]
		}
		return "1", "65535"
	}
	return portRangeArr[0], portRangeArr[1]
}

func convertRulePortRangeCBToCloudIt(from string, to string) (string, error) {
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
		return "1-65535", nil
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

func convertRuleInfoCloudItToCB(sgRule securitygroup.SecurityGroupRules) irs.SecurityRuleInfo {
	protocol := convertRuleProtocolCloudItToCB(sgRule.Protocol)
	fromPort, toPort := convertRulePortRangeCloudItToCB(sgRule.Port)
	return irs.SecurityRuleInfo{
		IPProtocol: protocol,
		Direction:  sgRule.Type,
		CIDR:       sgRule.Target,
		FromPort:   fromPort,
		ToPort:     toPort,
	}
}

func convertRuleInfoCBToCloudIt(sgRuleInfo irs.SecurityRuleInfo) (securitygroup.SecurityGroupRules, error) {
	if sgRuleInfo.CIDR == NULL {
		sgRuleInfo.CIDR = DefaultCIDR
	}
	portRange, err := convertRulePortRangeCBToCloudIt(sgRuleInfo.FromPort, sgRuleInfo.ToPort)
	if err != nil {
		return securitygroup.SecurityGroupRules{}, err
	}
	protocol, err := convertRuleProtocolCBToCloudIt(sgRuleInfo.IPProtocol)
	if err != nil {
		return securitygroup.SecurityGroupRules{}, err
	}
	return securitygroup.SecurityGroupRules{
		Name:     generateRuleName(sgRuleInfo.Direction),
		Type:     sgRuleInfo.Direction,
		Port:     portRange,
		Target:   sgRuleInfo.CIDR,
		Protocol: protocol,
	}, nil
}
func generateRuleName(direct string) string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%s-rules-%s", direct, strconv.FormatInt(rand.Int63n(100000), 10))
}

func getRuleInfoWithIds(rawRules *[]securitygroup.SecurityGroupRules) (*[]securityRuleInfoWithId, error) {
	secRuleArrIds := make([]securityRuleInfoWithId, len(*rawRules))
	for i, sgRule := range *rawRules {
		secRuleInfo := convertRuleInfoCloudItToCB(sgRule)
		secRuleArrIds[i] = securityRuleInfoWithId{
			Id:       sgRule.ID,
			RuleInfo: secRuleInfo,
		}
	}
	return &secRuleArrIds, nil
}


