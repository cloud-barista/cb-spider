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
	"errors"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type AlibabaSecurityHandler struct {
	Region idrv.RegionInfo
	Client *ecs.Client
}

func (securityHandler *AlibabaSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	cblogger.Infof("securityReqInfo : ", securityReqInfo)
	//spew.Dump(securityReqInfo)

	//=======================================
	// 보안 그룹 생성
	//=======================================
	request := ecs.CreateCreateSecurityGroupRequest()
	request.Scheme = "https"

	request.Description = securityReqInfo.IId.NameId
	request.SecurityGroupName = securityReqInfo.IId.NameId
	request.VpcId = securityReqInfo.VpcIID.SystemId
	cblogger.Debugf("보안 그룹 생성 요청 정보", request)

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
	cblogger.Infof("[%s] 보안 그룹 생성완료: SecurityGroupId:[%s]", securityReqInfo.IId.NameId, createRes.SecurityGroupId)
	//spew.Dump(createRes)

	//=======================================
	// 보안 정책 추가
	//=======================================
	cblogger.Infof("보안 그룹[%s]에 인바운드/아웃바운드 보안 정책 처리", createRes.SecurityGroupId)
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
	//// spew.Dump(createRuleRes)
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
//  - vpcId deprecated
//  - function name 변경 : AuthorizeSecurityRules to AddRules
//func (securityHandler *AlibabaSecurityHandler) AuthorizeSecurityRules(securityGroupId string, vpcId string, securityRuleInfos *[]irs.SecurityRuleInfo) (*[]irs.SecurityRuleInfo, error) {
//func (securityHandler *AlibabaSecurityHandler) AuthorizeSecurityRules(securityGroupId string, securityRuleInfos *[]irs.SecurityRuleInfo) (*[]irs.SecurityRuleInfo, error) {
func (securityHandler *AlibabaSecurityHandler) AddRules(securityIID irs.IID, securityRuleInfos *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	securityGroupId := securityIID.SystemId
	cblogger.Infof("securityGroupId : [%s]  / securityRuleInfos : [%v]", securityGroupId, securityRuleInfos)
	//cblogger.Info("AuthorizeSecurityRules ", securityRuleInfos)
	spew.Dump(securityRuleInfos)

	for _, curRule := range *securityRuleInfos {
		if strings.EqualFold(curRule.Direction, "inbound") {
			request := ecs.CreateAuthorizeSecurityGroupRequest()
			request.Scheme = "https"
			request.IpProtocol = curRule.IPProtocol
			request.PortRange = curRule.FromPort + "/" + curRule.ToPort
			request.SecurityGroupId = securityGroupId
			request.SourceCidrIp = curRule.CIDR

			cblogger.Infof("[%s] [%s] inbound rule Request", request.IpProtocol, request.PortRange)
			spew.Dump(request)
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
			spew.Dump(request)
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
//func (securityHandler *AlibabaSecurityHandler) RevokeSecurityRules(securityGroupId string, securityRuleInfos *[]irs.SecurityRuleInfo) (*[]irs.SecurityRuleInfo, error) {
func (securityHandler *AlibabaSecurityHandler) RemoveRules(securityIID irs.IID, securityRuleInfos *[]irs.SecurityRuleInfo) (bool, error) {
	securityGroupId := securityIID.SystemId
	cblogger.Infof("securityGroupId : [%s]  / securityRuleInfos : [%v]", securityGroupId, securityRuleInfos)
	spew.Dump(securityRuleInfos)
	// "cidr": "string",
	// "fromPort": "string",
	// "ipprotocol": "string",
	// "toPort": "string"
	for _, curRule := range *securityRuleInfos {
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
			spew.Dump(request)
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
			spew.Dump(request)
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
	spew.Dump(request)

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
	callogger.Info(call.String(callLogInfo))

	cblogger.Info(result)
	//spew.Dump(result)
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
	//spew.Dump(result)
	//ecs.DescribeSecurityGroupsResponse

	//ecs.DescribeSecurityGroupsResponse.SecurityGroups
	//ecs.SecurityGroups
	//spew.Dump(result)

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
		curSecurityRuleInfo.Direction = curPermission.Direction

		if strings.EqualFold(curPermission.Direction, "ingress") {
			// curSecurityRuleInfo.Direction = "inbound"
			curSecurityRuleInfo.CIDR = curPermission.SourceCidrIp
		} else if strings.EqualFold(curPermission.Direction, "egress") {
			// curSecurityRuleInfo.Direction = "outbound"
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

	cblogger.Info(response)
	cblogger.Infof("Successfully delete security group %q.", securityIID.SystemId)
	return true, nil
}