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
	spew.Dump(securityReqInfo)

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
	createRuleRes, errRule := securityHandler.AuthorizeSecurityRules(createRes.SecurityGroupId, securityReqInfo.VpcIID.SystemId, securityReqInfo.SecurityRules)
	if errRule != nil {
		cblogger.Errorf("Unable to create security group rule %q, %v", securityReqInfo.IId.NameId, err)
		return irs.SecurityInfo{}, errRule
	} else {
		cblogger.Info("Successfully set security group AuthorizeSecurityRules")
	}

	cblogger.Info("AuthorizeSecurityRules Result")
	spew.Dump(createRuleRes)

	securityInfo, _ := securityHandler.GetSecurity(irs.IID{SystemId: createRes.SecurityGroupId})
	//securityInfo.IId.NameId = securityReqInfo.IId.NameId
	return securityInfo, nil
}

func (securityHandler *AlibabaSecurityHandler) AuthorizeSecurityRules(securityGroupId string, vpcId string, securityRuleInfos *[]irs.SecurityRuleInfo) (*[]irs.SecurityRuleInfo, error) {
	cblogger.Infof("securityGroupId : [%s] / vpcId : [%s] / securityRuleInfos : [%v]", securityGroupId, vpcId, securityRuleInfos)
	//cblogger.Info("AuthorizeSecurityRules ", securityRuleInfos)
	spew.Dump(securityRuleInfos)

	/*
		if strings.EqualFold(curRule.Direction, "inbound") {
		} else if strings.EqualFold(curRule.Direction, "outbound") {
		}
	*/

	for _, curRule := range *securityRuleInfos {
		//if curRule.Direction == "inbound" {
		if strings.EqualFold(curRule.Direction, "inbound") {
			request := ecs.CreateAuthorizeSecurityGroupRequest()
			request.Scheme = "https"
			request.IpProtocol = curRule.IPProtocol
			request.PortRange = curRule.FromPort + "/" + curRule.ToPort
			request.SecurityGroupId = securityGroupId
			request.SourceCidrIp = "0.0.0.0/0"

			cblogger.Infof("[%s] [%s] inbound rule Request", request.IpProtocol, request.PortRange)
			spew.Dump(request)
			response, err := securityHandler.Client.AuthorizeSecurityGroup(request)
			if err != nil {
				cblogger.Errorf("Unable to create security group[%s] inbound rule - [%s] [%s] AuthorizeSecurityGroup Request", securityGroupId, request.IpProtocol, request.PortRange)
				cblogger.Error(err)
				return nil, err
			}
			cblogger.Infof("[%s] [%s] AuthorizeSecurityGroup Request success - RequestId:[%s]", request.IpProtocol, request.PortRange, response)
			//} else if curRule.Direction == "outbound" {
		} else if strings.EqualFold(curRule.Direction, "outbound") {
			request := ecs.CreateAuthorizeSecurityGroupEgressRequest()
			request.Scheme = "https"
			request.IpProtocol = curRule.IPProtocol
			request.PortRange = curRule.FromPort + "/" + curRule.ToPort
			request.SecurityGroupId = securityGroupId
			//request.SourceCidrIp = "0.0.0.0/0"
			request.DestCidrIp = "0.0.0.0/0"

			cblogger.Infof("[%s] [%s] outbound rule Request", request.IpProtocol, request.PortRange)
			spew.Dump(request)
			response, err := securityHandler.Client.AuthorizeSecurityGroupEgress(request)
			if err != nil {
				cblogger.Errorf("Unable to create security group[%s] outbound rule - [%s] [%s] AuthorizeSecurityGroup Request", securityGroupId, request.IpProtocol, request.PortRange)
				cblogger.Error(err)
				return nil, err
			}
			cblogger.Infof("[%s] [%s] AuthorizeSecurityGroup Request success - RequestId:[%s]", request.IpProtocol, request.PortRange, response)
		}
	}

	return nil, nil
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

	cblogger.Info(result)
	//spew.Dump(result)
	//ecs.DescribeSecurityGroupsResponse

	//ecs.DescribeSecurityGroupsResponse.SecurityGroups
	//ecs.SecurityGroups
	cblogger.Info(result)
	spew.Dump(result)
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

		/*
			if strings.EqualFold(curPermission.Direction, "ingress") {
				curSecurityRuleInfo.Direction = "inbound"
			} else if strings.EqualFold(curPermission.Direction, "egress") {
				curSecurityRuleInfo.Direction = "outbound"
			}
		*/
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
