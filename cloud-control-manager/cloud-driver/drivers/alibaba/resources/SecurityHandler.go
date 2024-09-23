// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by zephy@mz.co.kr, 2019.09.

package resources

import (
	"encoding/json"
	"errors"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AlibabaSecurityHandler struct {
	Region idrv.RegionInfo
	Client *ecs.Client
}

type RuleAction string

const (
	Add    RuleAction = "Add"
	Remove RuleAction = "Remove"
)

func (securityHandler *AlibabaSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	cblogger.Infof("securityReqInfo : ", securityReqInfo)
	//cblogger.Debug(securityReqInfo)

	//=======================================
	// 보안 그룹 생성
	//=======================================
	request := ecs.CreateCreateSecurityGroupRequest()
	request.Scheme = "https"

	request.Description = securityReqInfo.IId.NameId
	request.SecurityGroupName = securityReqInfo.IId.NameId
	request.VpcId = securityReqInfo.VpcIID.SystemId
	request.SecurityGroupType = "enterprise"

	/// 0717 ///

	if securityReqInfo.TagList != nil && len(securityReqInfo.TagList) > 0 {

		sgTags := []ecs.CreateSecurityGroupTag{}
		for _, sgTag := range securityReqInfo.TagList {
			tag0 := ecs.CreateSecurityGroupTag{
				Key:   sgTag.Key,
				Value: sgTag.Value,
			}
			sgTags = append(sgTags, tag0)

		}
		request.Tag = &sgTags
	}

	/// 0717 ///

	cblogger.Debugf("Security group creation request information", request)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: securityReqInfo.IId.NameId,
		CloudOSAPI:   "CreateSecurityGroup()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	// Create the security group with the VPC, name and description.
	createRes, err := securityHandler.Client.CreateSecurityGroup(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Errorf("Unable to create security group %q, %v", securityReqInfo.IId.NameId, err)
		return irs.SecurityInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))
	cblogger.Infof("[%s] Security group creation complete: SecurityGroupId:[%s]", securityReqInfo.IId.NameId, createRes.SecurityGroupId)
	//cblogger.Debug(createRes)

	//=======================================
	// 보안 정책 추가
	//=======================================
	defaultRuleRequest := ecs.CreateAuthorizeSecurityGroupEgressRequest()
	defaultRuleRequest.Scheme = "https"
	defaultRuleRequest.IpProtocol = "all"
	defaultRuleRequest.PortRange = "-1/-1"
	defaultRuleRequest.SecurityGroupId = createRes.SecurityGroupId
	defaultRuleRequest.DestCidrIp = "0.0.0.0/0"
	defaultRuleRequest.Priority = "100"

	cblogger.Infof("[%s] [%s] outbound rule Request", defaultRuleRequest.IpProtocol, defaultRuleRequest.PortRange)
	cblogger.Debug(request)
	response, err := securityHandler.Client.AuthorizeSecurityGroupEgress(defaultRuleRequest)
	if err != nil {
		cblogger.Errorf("Unable to create security group[%s] outbound rule - [%s] [%s] AuthorizeSecurityGroup Request", defaultRuleRequest.SecurityGroupId, defaultRuleRequest.IpProtocol, defaultRuleRequest.PortRange)
		cblogger.Error(err)
		return irs.SecurityInfo{}, err
	}
	cblogger.Infof("[%s] [%s] AuthorizeSecurityGroup Request success - RequestId:[%s]", defaultRuleRequest.IpProtocol, defaultRuleRequest.PortRange, response)

	cblogger.Infof("Processing inbound/outbound security policies for security group [%s]", defaultRuleRequest.SecurityGroupId)
	return securityHandler.AddRules(irs.IID{SystemId: createRes.SecurityGroupId}, securityReqInfo.SecurityRules)

	//createRuleRes, errRule := securityHandler.AuthorizeSecurityRules(createRes.SecurityGroupId, securityReqInfo.VpcIID.SystemId, securityReqInfo.SecurityRules)
	//createRuleRes, errRule := securityHandler.AuthorizeSecurityRules(createRes.SecurityGroupId, securityReqInfo.SecurityRules)
	//if errRule != nil {
	//	cblogger.Errorf("Unable to create security group rule %q, %v", securityReqInfo.IId.NameId, err)
	//	return irs.SecurityInfo{}, errRule
	//} else {
	//	cblogger.Info("Successfully set security group AuthorizeSecurityRules")
	//}
	//
	//cblogger.Debug("AuthorizeSecurityRules Result")
	//// cblogger.Debug(createRuleRes)
	//cblogger.Debug(createRuleRes)
	//
	//securityInfo, _ := securityHandler.GetSecurity(irs.IID{SystemId: createRes.SecurityGroupId})
	////securityInfo.IId.NameId = securityReqInfo.IId.NameId
	//return securityInfo, nil
}

// SecurityGroup에 Rule 추가
// 공통 함수명은 AddRules 이나 실제 Alibaba는 AuthorizeSecurityRules
// SecurityGroup 생성시에도 호출
// 저장 후 rule 목록 조회하여 return
//   - vpcId deprecated
//   - function name 변경 : AuthorizeSecurityRules to AddRules
//
// func (securityHandler *AlibabaSecurityHandler) AuthorizeSecurityRules(securityGroupId string, vpcId string, securityRuleInfos *[]irs.SecurityRuleInfo) (*[]irs.SecurityRuleInfo, error) {
// func (securityHandler *AlibabaSecurityHandler) AuthorizeSecurityRules(securityGroupId string, securityRuleInfos *[]irs.SecurityRuleInfo) (*[]irs.SecurityRuleInfo, error) {
func (securityHandler *AlibabaSecurityHandler) AddRules(securityIID irs.IID, reqSecurityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	securityGroupId := securityIID.SystemId
	cblogger.Infof("securityGroupId : [%s]  / securityRuleInfos : [%v]", securityGroupId, reqSecurityRules)
	//cblogger.Info("AuthorizeSecurityRules ", securityRuleInfos)
	cblogger.Debug(reqSecurityRules)

	if len(*reqSecurityRules) < 1 {
		return irs.SecurityInfo{}, errors.New("invalid value - The SecurityRules to add is empty")
	}

	presentRules, presentRulesErr := securityHandler.ExtractSecurityRuleInfo(securityGroupId)
	if presentRulesErr != nil {
		cblogger.Error(presentRulesErr)
		return irs.SecurityInfo{}, presentRulesErr
	}

	checkResult := sameRulesCheck(&presentRules, reqSecurityRules, Add)
	cblogger.Infof("checkResult: [%v]", checkResult)
	if checkResult != nil {
		errorMsg := ""
		for _, rule := range *checkResult {
			jsonRule, err := json.Marshal(rule)
			if err != nil {
				cblogger.Error(err)
			}
			errorMsg += string(jsonRule)
		}
		return irs.SecurityInfo{}, errors.New("invalid value - " + errorMsg + " already exists!")
	}

	for _, curRule := range *reqSecurityRules {
		if strings.EqualFold(curRule.Direction, "inbound") {
			request := ecs.CreateAuthorizeSecurityGroupRequest()
			request.Scheme = "https"
			request.IpProtocol = curRule.IPProtocol
			request.PortRange = curRule.FromPort + "/" + curRule.ToPort
			request.SecurityGroupId = securityGroupId
			request.SourceCidrIp = curRule.CIDR

			cblogger.Infof("[%s] [%s] inbound rule Request", request.IpProtocol, request.PortRange)
			cblogger.Debug(request)
			response, err := securityHandler.Client.AuthorizeSecurityGroup(request)
			if err != nil {
				cblogger.Errorf("Unable to create security group[%s] inbound rule - [%s] [%s] AuthorizeSecurityGroup Request", securityGroupId, request.IpProtocol, request.PortRange)
				cblogger.Error(err)
				return irs.SecurityInfo{}, err
			}
			cblogger.Infof("[%s] [%s] AuthorizeSecurityGroup Request success - RequestId:[%s]", request.IpProtocol, request.PortRange, response)
		} else if strings.EqualFold(curRule.Direction, "outbound") {
			request := ecs.CreateAuthorizeSecurityGroupEgressRequest()
			request.Scheme = "https"
			request.IpProtocol = curRule.IPProtocol
			request.PortRange = curRule.FromPort + "/" + curRule.ToPort
			request.SecurityGroupId = securityGroupId
			request.DestCidrIp = curRule.CIDR

			cblogger.Infof("[%s] [%s] outbound rule Request", request.IpProtocol, request.PortRange)
			cblogger.Debug(request)

			response, err := securityHandler.Client.AuthorizeSecurityGroupEgress(request)
			if err != nil {
				cblogger.Errorf("Unable to create security group[%s] outbound rule - [%s] [%s] AuthorizeSecurityGroup Request", securityGroupId, request.IpProtocol, request.PortRange)
				cblogger.Error(err)
				return irs.SecurityInfo{}, err
			}
			cblogger.Infof("[%s] [%s] AuthorizeSecurityGroup Request success - RequestId:[%s]", request.IpProtocol, request.PortRange, response)
		}
	}

	securityInfo, _ := securityHandler.GetSecurity(irs.IID{SystemId: securityGroupId})
	//securityInfo.IId.NameId = securityReqInfo.IId.NameId
	return securityInfo, nil
}

// SecurityGroup의 Rule 제거
// 공통 함수명은 RemoveRules 이나 실제 Alibaba는 RevokeSecurityRules

// If the security group rule to be deleted does not exist, the RevokeSecurityGroup operation succeeds but no rule is deleted.
// func (securityHandler *AlibabaSecurityHandler) RevokeSecurityRules(securityGroupId string, securityRuleInfos *[]irs.SecurityRuleInfo) (*[]irs.SecurityRuleInfo, error) {
func (securityHandler *AlibabaSecurityHandler) RemoveRules(securityIID irs.IID, reqSecurityRules *[]irs.SecurityRuleInfo) (bool, error) {
	securityGroupId := securityIID.SystemId
	cblogger.Infof("securityGroupId : [%s]  / securityRuleInfos : [%v]", securityGroupId, reqSecurityRules)
	cblogger.Debug(reqSecurityRules)

	presentRules, presentRulesErr := securityHandler.ExtractSecurityRuleInfo(securityGroupId)
	if presentRulesErr != nil {
		cblogger.Error(presentRulesErr)
		return false, presentRulesErr
	}

	checkResult := sameRulesCheck(&presentRules, reqSecurityRules, Remove)
	cblogger.Infof("checkResult: [%v]", checkResult)
	if checkResult != nil {
		errorMsg := ""
		for _, rule := range *checkResult {
			jsonRule, err := json.Marshal(rule)
			if err != nil {
				cblogger.Error(err)
			}
			errorMsg += string(jsonRule)
		}
		return false, errors.New("invalid value - " + errorMsg + " does not exist!")
	}

	// "cidr": "string",
	// "fromPort": "string",
	// "ipprotocol": "string",
	// "toPort": "string"
	for _, curRule := range *reqSecurityRules {
		if strings.EqualFold(curRule.Direction, "inbound") {
			// 3가지 case중 1번 사용.
			// SourceGroupId : The ID of the source security group
			// - At least one of SourceGroupId and SourceCidrIp must be specified.
			//   .If SourceGroupId is specified but SourceCidrIp is not, the NicType parameter can be set only to intranet.
			//     -> securityRuleInfo 에는 nicType, policy 가 들어있지 않음.
			//   .If both SourceGroupId and SourceCidrIp are specified, SourceCidrIp takes precedence.
			//nicType := "intranet"

			request := ecs.CreateRevokeSecurityGroupRequest()
			// case 1. 특정 CIDR block 의 인바운드 Rule 삭제 : IpProtocol, PortRange, SourcePortRange (optional), NicType, Policy, DestCidrIp (optional), and SourceCidrIp
			request.IpProtocol = curRule.IPProtocol
			request.PortRange = curRule.FromPort + "/" + curRule.ToPort
			request.SecurityGroupId = securityGroupId
			request.SourceCidrIp = curRule.CIDR

			//// case 2. 특정 securityGroup의 인바운드 Rule 삭제 : IpProtocol, PortRange, SourcePortRange (optional), NicType, Policy, DestCidrIp (optional), and SourceCidrIp
			//request.SecurityGroupId = securityGroupId
			////request.SourceGroupId = SourceGroupId
			//request.IpProtocol = curRule.IPProtocol
			//request.PortRange = curRule.FromPort + "/" + curRule.ToPort
			////request.NicType = nicType
			////request.Policy =
			//
			//// case 3. 접두사(prefix)목록과 연결 된 인바운드 Rule 삭제 : IpProtocol, PortRange, SourcePortRange (optional), NicType, Policy, DestCidrIp (optional), and SourceCidrIp
			//request.SecurityGroupId = securityGroupId
			////request.SourcePrefixListId =
			//request.IpProtocol = curRule.IPProtocol
			//request.PortRange = curRule.FromPort + "/" + curRule.ToPort
			////request.NicType = nicType
			////request.Policy =

			cblogger.Infof("[%s] [%s] inbound rule Request", request.IpProtocol, request.PortRange)
			cblogger.Debug(request)
			response, err := securityHandler.Client.RevokeSecurityGroup(request)
			if err != nil {
				cblogger.Errorf("Unable to revoke security group[%s] inbound rule - [%s] [%s] RevokeSecurityGroup Request", securityGroupId, request.IpProtocol, request.PortRange)
				cblogger.Error(err)
				return false, err
			}
			cblogger.Infof("[%s] [%s] RevokeSecurityGroup Request success - RequestId:[%s]", request.IpProtocol, request.PortRange, response)
		} else if strings.EqualFold(curRule.Direction, "outbound") {
			request := ecs.CreateRevokeSecurityGroupEgressRequest()
			request.Scheme = "https"
			request.IpProtocol = curRule.IPProtocol
			request.PortRange = curRule.FromPort + "/" + curRule.ToPort
			request.SecurityGroupId = securityGroupId
			request.DestCidrIp = curRule.CIDR

			cblogger.Infof("[%s] [%s] outbound rule Request", request.IpProtocol, request.PortRange)
			cblogger.Debug(request)
			response, err := securityHandler.Client.RevokeSecurityGroupEgress(request)
			if err != nil {
				cblogger.Errorf("Unable to revoke security group[%s] outbound rule - [%s] [%s] RevokeSecurityGroupEgress Request", securityGroupId, request.IpProtocol, request.PortRange)
				cblogger.Error(err)
				return false, err
			}
			cblogger.Infof("[%s] [%s] RevokeSecurityGroupEgress Request success - RequestId:[%s]", request.IpProtocol, request.PortRange, response)
		}
	}

	return true, nil
}

func (securityHandler *AlibabaSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	//	return nil, nil

	// get SecurityGroup & SecurityGroupAttribute for Alibaba
	request := ecs.CreateDescribeSecurityGroupsRequest()
	request.Scheme = "https"
	cblogger.Debug(request)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: "ListSecurity()",
		CloudOSAPI:   "DescribeSecurityGroups()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	result, err := securityHandler.Client.DescribeSecurityGroups(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return nil, err
	}
	callogger.Debug(call.String(callLogInfo))

	cblogger.Debug(result)
	//cblogger.Debug(result)
	//ecs.DescribeSecurityGroupsResponse

	var securityInfoList []*irs.SecurityInfo
	for _, curSecurityGroup := range result.SecurityGroups.SecurityGroup {
		curSecurityInfo, errSecurityInfo := securityHandler.ExtractSecurityInfo(&curSecurityGroup)
		if errSecurityInfo != nil {
			cblogger.Error(errSecurityInfo)
			return nil, errSecurityInfo
		}

		securityInfoList = append(securityInfoList, &curSecurityInfo)
	}

	return securityInfoList, nil
}

func (securityHandler *AlibabaSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	cblogger.Infof("SecurityGroupId : [%s]", securityIID.SystemId)

	request := ecs.CreateDescribeSecurityGroupsRequest()
	request.Scheme = "https"
	request.SecurityGroupId = securityIID.SystemId

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: securityIID.SystemId,
		CloudOSAPI:   "DescribeSecurityGroups()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	result, err := securityHandler.Client.DescribeSecurityGroups(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return irs.SecurityInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Debug(result)
	//cblogger.Debug(result)
	//ecs.DescribeSecurityGroupsResponse

	//ecs.DescribeSecurityGroupsResponse.SecurityGroups
	//ecs.SecurityGroups
	//cblogger.Debug(result)

	//ecs.DescribeSecurityGroupsResponse
	if result.TotalCount < 1 {
		return irs.SecurityInfo{}, errors.New("Notfound: '" + securityIID.SystemId + "' SecurityGroup Not found")
	}

	securityInfo, errSecurityInfo := securityHandler.ExtractSecurityInfo(&result.SecurityGroups.SecurityGroup[0])
	if errSecurityInfo != nil {
		cblogger.Error(errSecurityInfo)
		return irs.SecurityInfo{}, errSecurityInfo
	}

	return securityInfo, nil
}

func (securityHandler *AlibabaSecurityHandler) ExtractSecurityInfo(securityGroupResult *ecs.SecurityGroup) (irs.SecurityInfo, error) {
	//securityRules := ExtractIpPermissions(securityGroupResult.SecurityGroups.SecurityGroup)
	var securityRuleInfos []irs.SecurityRuleInfo

	securityRuleInfos, errRuleInfos := securityHandler.ExtractSecurityRuleInfo(securityGroupResult.SecurityGroupId)
	if errRuleInfos != nil {
		cblogger.Error(errRuleInfos)
		return irs.SecurityInfo{}, errRuleInfos
	}

	securityInfo := irs.SecurityInfo{
		IId: irs.IID{NameId: securityGroupResult.SecurityGroupName, SystemId: securityGroupResult.SecurityGroupId},
		//SecurityRules: &[]irs.SecurityRuleInfo{},
		//SecurityRules: &securityRules,
		VpcIID:        irs.IID{SystemId: securityGroupResult.VpcId},
		SecurityRules: &securityRuleInfos,

		KeyValueList: []irs.KeyValue{
			{Key: "SecurityGroupName", Value: securityGroupResult.SecurityGroupName},
			{Key: "CreationTime", Value: securityGroupResult.CreationTime},
		},
	}
	if securityGroupResult.Tags.Tag != nil {
		var tagList []irs.KeyValue
		for _, tag := range securityGroupResult.Tags.Tag {
			tagList = append(tagList, irs.KeyValue{
				Key:   tag.TagKey,
				Value: tag.TagValue,
			})
		}
		securityInfo.TagList = tagList
	}

	return securityInfo, nil
}

// 보안 그룹의 InBound / OutBound 정보를 조회함.
func (securityHandler *AlibabaSecurityHandler) ExtractSecurityRuleInfo(securityGroupId string) ([]irs.SecurityRuleInfo, error) {
	var securityRuleInfos []irs.SecurityRuleInfo

	request := ecs.CreateDescribeSecurityGroupAttributeRequest()
	request.Scheme = "https"
	request.SecurityGroupId = securityGroupId

	response, err := securityHandler.Client.DescribeSecurityGroupAttribute(request)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}
	cblogger.Info(response)

	/*
	   FromPort    string
	   ToPort        string
	*/
	var curSecurityRuleInfo irs.SecurityRuleInfo
	for _, curPermission := range response.Permissions.Permission {
		// curSecurityRuleInfo.Direction = curPermission.Direction

		if strings.EqualFold(curPermission.Direction, "ingress") {
			curSecurityRuleInfo.Direction = "inbound"
			curSecurityRuleInfo.CIDR = curPermission.SourceCidrIp
		} else if strings.EqualFold(curPermission.Direction, "egress") {
			curSecurityRuleInfo.Direction = "outbound"
			curSecurityRuleInfo.CIDR = curPermission.DestCidrIp
		}

		curSecurityRuleInfo.IPProtocol = curPermission.IpProtocol

		portRange := strings.Split(curPermission.PortRange, "/")

		curSecurityRuleInfo.FromPort = portRange[0]
		if len(portRange) > 1 {
			curSecurityRuleInfo.ToPort = portRange[1]
		}
		securityRuleInfos = append(securityRuleInfos, curSecurityRuleInfo)
	}

	return securityRuleInfos, nil
}

func (securityHandler *AlibabaSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	cblogger.Infof("securityID : [%s]", securityIID.SystemId)

	request := ecs.CreateDeleteSecurityGroupRequest()
	request.Scheme = "https"
	request.SecurityGroupId = securityIID.SystemId

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: securityIID.SystemId,
		CloudOSAPI:   "DeleteSecurityGroup()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	callLogStart := call.Start()
	response, err := securityHandler.Client.DeleteSecurityGroup(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Errorf("Unable to get descriptions for security groups, %v.", err)
		return false, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Debug(response)
	cblogger.Infof("Successfully delete security group %q.", securityIID.SystemId)
	return true, nil
}

// 동일한 rule이 있는지 체크
// RuleAction이 Add면 중복인 rule 리턴, Remove면 없는 rule 리턴
func sameRulesCheck(presentSecurityRules *[]irs.SecurityRuleInfo, reqSecurityRules *[]irs.SecurityRuleInfo, action RuleAction) *[]irs.SecurityRuleInfo {
	var checkResult []irs.SecurityRuleInfo
	cblogger.Infof("presentSecurityRules: [%v] / reqSecurityRules: [%v]", presentSecurityRules, reqSecurityRules)
	for _, reqRule := range *reqSecurityRules {
		hasFound := false
		reqRulePort := ""
		if reqRule.FromPort == "" {
			reqRulePort = reqRule.ToPort
		} else if reqRule.ToPort == "" {
			reqRulePort = reqRule.FromPort
		} else if reqRule.FromPort == reqRule.ToPort {
			reqRulePort = reqRule.FromPort
		} else {
			reqRulePort = reqRule.FromPort + "-" + reqRule.ToPort
		}

		for _, present := range *presentSecurityRules {
			presentPort := ""
			if present.FromPort == "" {
				presentPort = present.ToPort
			} else if present.ToPort == "" {
				presentPort = present.FromPort
			} else if present.FromPort == present.ToPort {
				presentPort = present.FromPort
			} else {
				presentPort = present.FromPort + "-" + present.ToPort
			}

			if !strings.EqualFold(reqRule.Direction, present.Direction) {
				continue
			}
			if !strings.EqualFold(reqRule.IPProtocol, present.IPProtocol) {
				continue
			}
			if !strings.EqualFold(reqRulePort, presentPort) {
				continue
			}
			if !strings.EqualFold(reqRule.CIDR, present.CIDR) {
				continue
			}

			if action == Add {
				checkResult = append(checkResult, reqRule)
			}
			hasFound = true
			break
		}

		// Remove일때는 못 찾아야 append
		if action == Remove && !hasFound {
			checkResult = append(checkResult, reqRule)
		}
	}

	if len(checkResult) > 0 {
		return &checkResult
	}

	return nil
}

func (securityHandler *AlibabaSecurityHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("Cloud driver: called ListIID()!!")
	return nil, errors.New("Does not support ListIID() yet!!")
}

