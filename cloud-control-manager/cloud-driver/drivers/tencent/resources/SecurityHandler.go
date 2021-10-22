// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by CB-Spider Team, 2019.06.

package resources

import (
	"errors"
	"strings"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

type TencentSecurityHandler struct {
	Region idrv.RegionInfo
	Client *vpc.Client
}

//https://intl.cloud.tencent.com/document/product/213/34272
//https://intl.cloud.tencent.com/ko/document/api/215/36083
/*
@TODO 포트 다양하게 처리 가능해야 함. - 현재는 콤머는 에러 처리
  사용가능 포트 규칙 : 콤머(,) / 대쉬(-) / ALL(전체)
Port: A single port number, or a port range in the format of “8000-8010”. The Port field is accepted only if the value of the Protocol field is TCP or UDP. Otherwise Protocol and Port are mutually exclusive.
Action : ACCEPT or DROP
*/
func (securityHandler *TencentSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	cblogger.Infof("securityReqInfo : ", securityReqInfo)

	//=================================================
	// 동일 이름 생성 방지 추가(cb-spider 요청 필수 기능)
	//=================================================
	isExist, errExist := securityHandler.isExist(securityReqInfo.IId.NameId)
	if errExist != nil {
		cblogger.Error(errExist)
		return irs.SecurityInfo{}, errExist
	}
	if isExist {
		return irs.SecurityInfo{}, errors.New("A SecurityGroup with the name " + securityReqInfo.IId.NameId + " already exists.")
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: securityReqInfo.IId.NameId,
		CloudOSAPI:   "CreateSecurityGroupWithPolicies()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := vpc.NewCreateSecurityGroupWithPoliciesRequest()
	request.GroupName = common.StringPtr(securityReqInfo.IId.NameId)
	request.GroupDescription = common.StringPtr(securityReqInfo.IId.NameId) //설명 없으면 에러

	cblogger.Debug("보안 정책 처리")
	securityGroupPolicySet := &vpc.SecurityGroupPolicySet{}
	for _, curPolicy := range *securityReqInfo.SecurityRules {
		securityGroupPolicy := new(vpc.SecurityGroupPolicy)
		securityGroupPolicy.Protocol = common.StringPtr(curPolicy.IPProtocol)
		//securityGroupPolicy.CidrBlock = common.StringPtr("0.0.0.0/0")
		securityGroupPolicy.CidrBlock = common.StringPtr(curPolicy.CIDR)
		securityGroupPolicy.Action = common.StringPtr("accept")

		// 포트 번호에 "-"가 오면 모든 포트로 설정
		if curPolicy.FromPort == "-" || curPolicy.ToPort == "-" {
			securityGroupPolicy.Port = common.StringPtr("ALL")
		} else if curPolicy.ToPort != "" && curPolicy.ToPort != curPolicy.FromPort {
			securityGroupPolicy.Port = common.StringPtr(curPolicy.FromPort + "-" + curPolicy.ToPort)
		} else {
			securityGroupPolicy.Port = common.StringPtr(curPolicy.FromPort)
		}

		if strings.EqualFold(curPolicy.Direction, "inbound") {
			securityGroupPolicySet.Ingress = append(securityGroupPolicySet.Ingress, securityGroupPolicy)
		} else {
			securityGroupPolicySet.Egress = append(securityGroupPolicySet.Egress, securityGroupPolicy)
		}
	}

	request.SecurityGroupPolicySet = securityGroupPolicySet

	callLogStart := call.Start()
	response, err := securityHandler.Client.CreateSecurityGroupWithPolicies(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		spew.Dump(request)
		return irs.SecurityInfo{}, err
	}
	//spew.Dump(response)
	cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	securityInfo, errSecurity := securityHandler.GetSecurity(irs.IID{SystemId: *response.Response.SecurityGroup.SecurityGroupId})
	if errSecurity != nil {
		cblogger.Error(errSecurity)
		return irs.SecurityInfo{}, errSecurity
	}

	securityInfo.IId.NameId = securityReqInfo.IId.NameId
	return securityInfo, nil
}

func (securityHandler *TencentSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: "ListSecurity()",
		CloudOSAPI:   "DescribeSecurityGroups()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := vpc.NewDescribeSecurityGroupsRequest()
	request.Limit = common.StringPtr("100") //default : 20 / max : 100

	callLogStart := call.Start()
	response, err := securityHandler.Client.DescribeSecurityGroups(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return nil, err
	}
	//spew.Dump(response)
	cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	var results []*irs.SecurityInfo
	for _, securityGroup := range response.Response.SecurityGroupSet {
		// 	securityInfo := ExtractSecurityInfo(securityGroup)
		securityInfo, errSecurity := securityHandler.GetSecurity(irs.IID{NameId: *securityGroup.SecurityGroupName, SystemId: *securityGroup.SecurityGroupId})
		if errSecurity != nil {
			cblogger.Error(errSecurity)
			return nil, errSecurity
		}
		results = append(results, &securityInfo)
	}

	return results, nil
}

// cb-spider 정책상 이름 기반으로 중복 생성을 막아야 함.
func (securityHandler *TencentSecurityHandler) isExist(chkName string) (bool, error) {
	cblogger.Debugf("chkName : %s", chkName)

	request := vpc.NewDescribeSecurityGroupsRequest()
	request.Filters = []*vpc.Filter{
		&vpc.Filter{
			Name:   common.StringPtr("security-group-name"),
			Values: common.StringPtrs([]string{chkName}),
		},
	}

	response, err := securityHandler.Client.DescribeSecurityGroups(request)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	if *response.Response.TotalCount < 1 {
		return false, nil
	}

	cblogger.Infof("보안그룹 정보 찾음 - VpcId:[%s] / VpcName:[%s]", *response.Response.SecurityGroupSet[0].SecurityGroupId, *response.Response.SecurityGroupSet[0].SecurityGroupName)
	return true, nil
}

func (securityHandler *TencentSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	cblogger.Infof("securitySystemId : [%s]", securityIID.SystemId)
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: securityIID.SystemId,
		CloudOSAPI:   "DescribeSecurityGroupPolicies()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := vpc.NewDescribeSecurityGroupsRequest()
	request.SecurityGroupIds = common.StringPtrs([]string{securityIID.SystemId})

	callLogStart := call.Start()
	response, err := securityHandler.Client.DescribeSecurityGroups(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return irs.SecurityInfo{}, err
	}
	//spew.Dump(response)
	cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	if *response.Response.TotalCount > 0 {
		securityInfo := irs.SecurityInfo{}
		securityInfo.VpcIID = irs.IID{NameId: "N/A", SystemId: "N/A"}
		securityInfo.IId = irs.IID{NameId: *response.Response.SecurityGroupSet[0].SecurityGroupName, SystemId: *response.Response.SecurityGroupSet[0].SecurityGroupId}

		securityInfo.SecurityRules, err = securityHandler.GetSecurityRuleInfo(securityIID)
		if err != nil {
			cblogger.Error(err)
			return irs.SecurityInfo{}, err
		}
		return securityInfo, nil
	} else {
		return irs.SecurityInfo{}, errors.New("InvalidSecurityGroupId.NotFound: The SecurityGroup " + securityIID.SystemId + " does not exist")
	}
}

func (securityHandler *TencentSecurityHandler) GetSecurityRuleInfo(securityIID irs.IID) (*[]irs.SecurityRuleInfo, error) {
	cblogger.Infof("securitySystemId : [%s]", securityIID.SystemId)

	request := vpc.NewDescribeSecurityGroupPoliciesRequest()
	request.SecurityGroupId = common.StringPtr(securityIID.SystemId)

	response, err := securityHandler.Client.DescribeSecurityGroupPolicies(request)

	if err != nil {
		cblogger.Error(err)
		return nil, err
	}
	//spew.Dump(response)
	cblogger.Debug(response.ToJsonString())

	var securityRuleInfos []irs.SecurityRuleInfo
	var ingress []irs.SecurityRuleInfo
	var egress []irs.SecurityRuleInfo
	ingress, err = securityHandler.ExtractPolicyGroups(response.Response.SecurityGroupPolicySet.Ingress, "inbound")
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	egress, err = securityHandler.ExtractPolicyGroups(response.Response.SecurityGroupPolicySet.Egress, "outbound")
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}

	securityRuleInfos = append(ingress, egress...)

	return &securityRuleInfos, nil
}

//@TODO Port에 콤머가 사용된 정책 처리해야 함.
//direction : inbound / outbound
func (securityHandler *TencentSecurityHandler) ExtractPolicyGroups(policyGroups []*vpc.SecurityGroupPolicy, direction string) ([]irs.SecurityRuleInfo, error) {
	var results []irs.SecurityRuleInfo

	var fromPort string
	var toPort string

	/*
		var newDirection string
		//ingress -> inbound
		if strings.EqualFold(direction, "ingress") {
			newDirection = "inbound"
		} else if strings.EqualFold(direction, "egress") {
			newDirection = "outbound"
		} else { //UnKnown
			newDirection = direction
		}
	*/

	for _, curPolicy := range policyGroups {
		if len(*curPolicy.Port) > 0 {

			//WEB UI에서는 입력 자체가 불 가능한 것 같지만 혹시 몰라서 콤머 기반으로 파싱 후 대쉬(-)를 처리함.
			portArr := strings.Split(*curPolicy.Port, ",")
			for _, curPort := range portArr {
				portRange := strings.Split(curPort, "-")
				fromPort = portRange[0]
				if len(portRange) > 1 {
					toPort = portRange[len(portRange)-1]
				} else {
					toPort = ""
				}

				securityRuleInfo := irs.SecurityRuleInfo{
					Direction:  direction, // "inbound | outbound"
					CIDR:       *curPolicy.CidrBlock,
					IPProtocol: *curPolicy.Protocol,
					FromPort:   fromPort,
					ToPort:     toPort,
				}
				results = append(results, securityRuleInfo)
			}
		}
	}

	return results, nil
}

func (securityHandler *TencentSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	cblogger.Infof("securityNameId : [%s]", securityIID.SystemId)

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: securityIID.SystemId,
		CloudOSAPI:   "DeleteSecurityGroup()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := vpc.NewDeleteSecurityGroupRequest()
	request.SecurityGroupId = common.StringPtr(securityIID.SystemId)

	callLogStart := call.Start()
	response, err := securityHandler.Client.DeleteSecurityGroup(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		return false, err
	}
	//spew.Dump(response)
	cblogger.Debug(response.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	return true, nil
}
