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

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
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

	// Create the security group with the VPC, name and description.
	createRes, err := securityHandler.Client.CreateSecurityGroup(request)
	if err != nil {
		cblogger.Errorf("Unable to create security group %q, %v", securityReqInfo.IId.NameId, err)
		return irs.SecurityInfo{}, err
	}
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

	for _, curRule := range *securityRuleInfos {
		if curRule.Direction == "inbound" {
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
		} else if curRule.Direction == "outbound" {
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
	result, err := securityHandler.Client.DescribeSecurityGroups(request)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}
	cblogger.Info(result)
	//spew.Dump(result)
	//ecs.DescribeSecurityGroupsResponse

	var securityInfoList []*irs.SecurityInfo
	for _, curSecurityGroup := range result.SecurityGroups.SecurityGroup {
		curSecurityInfo := ExtractSecurityInfo(&curSecurityGroup)
		securityInfoList = append(securityInfoList, &curSecurityInfo)
	}

	return securityInfoList, nil
}

func (securityHandler *AlibabaSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	cblogger.Infof("SecurityGroupId : [%s]", securityIID.SystemId)

	request := ecs.CreateDescribeSecurityGroupsRequest()
	request.Scheme = "https"
	request.SecurityGroupId = securityIID.SystemId

	result, err := securityHandler.Client.DescribeSecurityGroups(request)
	if err != nil {
		cblogger.Error(err)
		return irs.SecurityInfo{}, err
	}

	//ecs.DescribeSecurityGroupsResponse.SecurityGroups
	//ecs.SecurityGroups
	cblogger.Info(result)
	spew.Dump(result)
	//ecs.DescribeSecurityGroupsResponse
	if result.TotalCount < 1 {
		return irs.SecurityInfo{}, errors.New("Notfound: '" + securityIID.SystemId + "' SecurityGroup Not found")
	}

	securityInfo := ExtractSecurityInfo(&result.SecurityGroups.SecurityGroup[0])
	return securityInfo, nil
}

func ExtractSecurityInfo(securityGroupResult *ecs.SecurityGroup) irs.SecurityInfo {
	//securityRules := ExtractIpPermissions(securityGroupResult.SecurityGroups.SecurityGroup)
	securityInfo := irs.SecurityInfo{
		IId: irs.IID{NameId: securityGroupResult.SecurityGroupName, SystemId: securityGroupResult.SecurityGroupId},
		//SecurityRules: &[]irs.SecurityRuleInfo{},
		//SecurityRules: &securityRules,
		VpcIID: irs.IID{SystemId: securityGroupResult.VpcId},

		KeyValueList: []irs.KeyValue{
			{Key: "SecurityGroupName", Value: securityGroupResult.SecurityGroupName},
			{Key: "CreationTime", Value: securityGroupResult.CreationTime},
		},
	}

	return securityInfo
}

func (securityHandler *AlibabaSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	cblogger.Infof("securityID : [%s]", securityIID.SystemId)

	request := ecs.CreateDeleteSecurityGroupRequest()
	request.Scheme = "https"
	request.SecurityGroupId = securityIID.SystemId

	response, err := securityHandler.Client.DeleteSecurityGroup(request)
	if err != nil {
		cblogger.Errorf("Unable to get descriptions for security groups, %v.", err)
		return false, err
	}
	cblogger.Info(response)
	cblogger.Infof("Successfully delete security group %q.", securityIID.SystemId)
	return true, nil
}
