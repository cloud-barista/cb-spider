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
	"encoding/json"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
	cblog "github.com/cloud-barista/cb-log"
)

type TencentSecurityHandler struct {
	Region idrv.RegionInfo
	Client *vpc.Client
}

type RuleAction string

const (
	Add RuleAction = "Add"
	Remove RuleAction = "Remove"
)

//https://intl.cloud.tencent.com/document/product/213/34272
//https://intl.cloud.tencent.com/ko/document/api/215/36083
/*
@TODO 포트 다양하게 처리 가능해야 함. - 현재는 콤머는 에러 처리
  사용가능 포트 규칙 : 콤머(,) / 대쉬(-) / ALL(전체)
Port: A single port number, or a port range in the format of “8000-8010”. The Port field is accepted only if the value of the Protocol field is TCP or UDP. Otherwise Protocol and Port are mutually exclusive.
Action : ACCEPT or DROP
*/
// Tencent의 경우 : If no rules are set, all traffic is rejected by default
// CB Spider의 outbound default는 All Open이므로 기본 Egress는 모두 open : CreateSecurityGroupWithPolicies
// 사용자의 policy를 추가로 적용 : CreateSecurityGroupPolicies
// 1번의 request는 한반향만 가능(두가지 동시에 불가)
func (securityHandler *TencentSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	cblogger.Infof("securityReqInfo : ", securityReqInfo)
	cblog.SetLevel("debug")
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

	defaultEgressRequest := vpc.NewCreateSecurityGroupWithPoliciesRequest()
	defaultEgressRequest.GroupName = common.StringPtr(securityReqInfo.IId.NameId)
	defaultEgressRequest.GroupDescription = common.StringPtr(securityReqInfo.IId.NameId) //설명 없으면 에러

	// default outbound는 All open 인데 tencent는 All block이므로 포트 열어 줌.
	egressSecurityGroupPolicySet := &vpc.SecurityGroupPolicySet{}
	egressSecurityGroupPolicy := new(vpc.SecurityGroupPolicy)
	egressSecurityGroupPolicy.Protocol = common.StringPtr("ALL")
	egressSecurityGroupPolicy.CidrBlock = common.StringPtr("0.0.0.0/0") // TODO : 넣어줘야 할 지 확인
	egressSecurityGroupPolicy.Action = common.StringPtr("accept")
	egressSecurityGroupPolicy.Port = common.StringPtr("ALL")
	egressSecurityGroupPolicySet.Egress = append(egressSecurityGroupPolicySet.Egress, egressSecurityGroupPolicy)

	defaultEgressRequest.SecurityGroupPolicySet = egressSecurityGroupPolicySet

	callLogStart := call.Start()
	defaultEgressResponse, err := securityHandler.Client.CreateSecurityGroupWithPolicies(defaultEgressRequest)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))

		cblogger.Error(err)
		spew.Dump(defaultEgressRequest)
		return irs.SecurityInfo{}, err
	}
	
	//spew.Dump(defaultEgressResponse)
	cblogger.Debug(defaultEgressResponse.ToJsonString())
	callogger.Info(call.String(callLogInfo))

	cblogger.Debug("보안 정책 처리")
	securityGroupPolicySet := &vpc.SecurityGroupPolicySet{}
	request := vpc.NewCreateSecurityGroupPoliciesRequest()
	request.SecurityGroupId = common.StringPtr(*defaultEgressResponse.Response.SecurityGroup.SecurityGroupId)

	for _, curPolicy := range *securityReqInfo.SecurityRules {
		securityGroupPolicy := new(vpc.SecurityGroupPolicy)
		securityGroupPolicy.Protocol = common.StringPtr(curPolicy.IPProtocol)
		//securityGroupPolicy.CidrBlock = common.StringPtr("0.0.0.0/0")
		securityGroupPolicy.CidrBlock = common.StringPtr(curPolicy.CIDR)
		securityGroupPolicy.Action = common.StringPtr("accept")

		// 포트 번호에 "-"가 오면 모든 포트로 설정
		if curPolicy.FromPort == "-1" || curPolicy.ToPort == "-1" {
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

	//callLogStart := call.Start()
	//response, err := securityHandler.Client.CreateSecurityGroupWithPolicies(request)

	response, err := securityHandler.Client.CreateSecurityGroupPolicies(request)
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

	securityInfo, errSecurity := securityHandler.GetSecurity(irs.IID{SystemId: *defaultEgressResponse.Response.SecurityGroup.SecurityGroupId})
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

// SecurityGroupRule추가
// 추가 후 SecurityGroup return
// CreateSecurityGroupPolicies inbound, outbound 동시 호출 불가 > 각각 호출
// ModifySecurityGroupPolicies Version을 0으로 set하면 초기화(모든 룰 사라짐), 설정하지 않으면 모두 삭제 후 insert(기존 값 사라짐, 넘어온 값만 사용)
func (securityHandler *TencentSecurityHandler) AddRules(securityIID irs.IID, reqSecurityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	////////
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: "AddRules()",
		CloudOSAPI:   "CreateSecurityGroupPolicies()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	if len(*reqSecurityRules) < 1 {
		return irs.SecurityInfo{}, errors.New("invalid value - The SecurityRules to add is empty")
	}

	presentRules, presentRulesErr := securityHandler.GetSecurityRuleInfo(securityIID)
	if presentRulesErr != nil {
		cblogger.Error(presentRulesErr)
		return irs.SecurityInfo{}, presentRulesErr
	}

	checkResult := sameRulesCheck(presentRules, reqSecurityRules, Add)
	if checkResult != nil {
		errorMsg := ""
		for _, rule := range *checkResult {
			jsonRule, err := json.Marshal(rule)
			if err != nil {
				cblogger.Error(err)
			}
			errorMsg += string(jsonRule)
		}
		return irs.SecurityInfo{}, errors.New("invalid value - "+ errorMsg +" already exists!")
	}

	securityGroupPolicyIngressSet := &vpc.SecurityGroupPolicySet{}
	securityGroupPolicyEgressSet := &vpc.SecurityGroupPolicySet{}

	// rule 생성 시에는 ingress와 egress가 동시에 생성되지 않기 때문에 ingress, egress 따로 호촐함
	for _, curPolicy := range *reqSecurityRules {
	
		// if !strings.EqualFold(curPolicy.Direction, commonDirection) {
		// 	return irs.SecurityInfo{}, errors.New("invalid - The parameter `Egress and Ingress` cannot be imported at the same time in the request.")
		// }
		

		securityGroupPolicy := new(vpc.SecurityGroupPolicy)
		securityGroupPolicy.Protocol = common.StringPtr(curPolicy.IPProtocol)
		//securityGroupPolicy.CidrBlock = common.StringPtr("0.0.0.0/0")
		securityGroupPolicy.CidrBlock = common.StringPtr(curPolicy.CIDR)
		securityGroupPolicy.Action = common.StringPtr("accept") // 하드코딩으로 Set되고 있음.

		// 포트 번호에 "-"가 오면 모든 포트로 설정
		if curPolicy.FromPort == "-1" || curPolicy.ToPort == "-1" {
			securityGroupPolicy.Port = common.StringPtr("ALL")
		} else if curPolicy.ToPort != "" && curPolicy.ToPort != curPolicy.FromPort {
			securityGroupPolicy.Port = common.StringPtr(curPolicy.FromPort + "-" + curPolicy.ToPort)
		} else {
			securityGroupPolicy.Port = common.StringPtr(curPolicy.FromPort)
		}

		if strings.EqualFold(curPolicy.Direction, "inbound") {
			securityGroupPolicyIngressSet.Ingress = append(securityGroupPolicyIngressSet.Ingress, securityGroupPolicy)
		} else {
			securityGroupPolicyEgressSet.Egress = append(securityGroupPolicyEgressSet.Egress, securityGroupPolicy)
		}
	}


	// Ingress request
	if len(securityGroupPolicyIngressSet.Ingress) > 0 {
		ingressRequest := vpc.NewCreateSecurityGroupPoliciesRequest()
		ingressRequest.SecurityGroupId = common.StringPtr(securityIID.SystemId)
		ingressRequest.SecurityGroupPolicySet = securityGroupPolicyIngressSet

		callLogStart := call.Start()
		ingressResponse, err := securityHandler.Client.CreateSecurityGroupPolicies(ingressRequest)
		callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

		if err != nil {
			callLogInfo.ErrorMSG = err.Error()
			callogger.Error(call.String(callLogInfo))

			cblogger.Error(err)
			return irs.SecurityInfo{}, err
		}
		//spew.Dump(response)
		cblogger.Debug(ingressResponse.ToJsonString())
		callogger.Info(call.String(callLogInfo))
	}

	// Egress request
	if len(securityGroupPolicyEgressSet.Egress) > 0 {
		egressRequest := vpc.NewCreateSecurityGroupPoliciesRequest()
		egressRequest.SecurityGroupId = common.StringPtr(securityIID.SystemId)
		egressRequest.SecurityGroupPolicySet = securityGroupPolicyEgressSet

		callLogStart := call.Start()
		egressResponse, err := securityHandler.Client.CreateSecurityGroupPolicies(egressRequest)
		callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

		if err != nil {
			callLogInfo.ErrorMSG = err.Error()
			callogger.Error(call.String(callLogInfo))

			cblogger.Error(err)
			return irs.SecurityInfo{}, err
		}
		//spew.Dump(response)
		cblogger.Debug(egressResponse.ToJsonString())
		callogger.Info(call.String(callLogInfo))
	}

	securityInfo, errSecurity := securityHandler.GetSecurity(securityIID)
	if errSecurity != nil {
		cblogger.Error(errSecurity)
		return irs.SecurityInfo{}, errSecurity
	}
	return securityInfo, errSecurity
}


// DeleteSecurityGroupPolicies inbound, outbound 동시 호출 불가 > 각각 호출
func (securityHandler *TencentSecurityHandler) RemoveRules(securityIID irs.IID, reqSecurityRules *[]irs.SecurityRuleInfo) (bool, error) {
	////////
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: "RemoveRules()",
		CloudOSAPI:   "DeleteSecurityGroupPolicies()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	presentRules, presentRulesErr := securityHandler.GetSecurityRuleInfo(securityIID)
	if presentRulesErr != nil {
		cblogger.Error(presentRulesErr)
		return false, presentRulesErr
	}

	checkResult := sameRulesCheck(presentRules, reqSecurityRules, Remove)
	if checkResult != nil {
		errorMsg := ""
		for _, rule := range *checkResult {
			jsonRule, err := json.Marshal(rule)
			if err != nil {
				cblogger.Error(err)
			}
			errorMsg += string(jsonRule)
		}
		return false, errors.New("invalid value - "+ errorMsg +" does not exist!")
	}

	securityGroupPolicyIngressSet := &vpc.SecurityGroupPolicySet{}
	securityGroupPolicyEgressSet := &vpc.SecurityGroupPolicySet{}

	for _, curPolicy := range *reqSecurityRules {
		securityGroupPolicy := new(vpc.SecurityGroupPolicy)
		securityGroupPolicy.Protocol = common.StringPtr(curPolicy.IPProtocol)
		//securityGroupPolicy.CidrBlock = common.StringPtr("0.0.0.0/0")
		securityGroupPolicy.CidrBlock = common.StringPtr(curPolicy.CIDR)
		securityGroupPolicy.Action = common.StringPtr("accept") // 하드코딩으로 Set되고 있음.

		// 포트 번호에 "-"가 오면 모든 포트로 설정
		if curPolicy.FromPort == "-1" || curPolicy.ToPort == "-1" {
			securityGroupPolicy.Port = common.StringPtr("ALL")
		} else if curPolicy.ToPort != "" && curPolicy.ToPort != curPolicy.FromPort {
			securityGroupPolicy.Port = common.StringPtr(curPolicy.FromPort + "-" + curPolicy.ToPort)
		} else {
			securityGroupPolicy.Port = common.StringPtr(curPolicy.FromPort)
		}

		if strings.EqualFold(curPolicy.Direction, "inbound") {
			securityGroupPolicyIngressSet.Ingress = append(securityGroupPolicyIngressSet.Ingress, securityGroupPolicy)
		} else {
			securityGroupPolicyEgressSet.Egress = append(securityGroupPolicyEgressSet.Egress, securityGroupPolicy)
		}
	}

	if len(securityGroupPolicyIngressSet.Ingress) > 0 {
		ingressRequest := vpc.NewDeleteSecurityGroupPoliciesRequest()
		ingressRequest.SecurityGroupId = common.StringPtr(securityIID.SystemId)
		ingressRequest.SecurityGroupPolicySet = securityGroupPolicyIngressSet

		callLogStart := call.Start()
		ingressResponse, err := securityHandler.Client.DeleteSecurityGroupPolicies(ingressRequest)
		callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

		if err != nil {
			callLogInfo.ErrorMSG = err.Error()
			callogger.Error(call.String(callLogInfo))

			cblogger.Error(err)
			return false, err
		}
		//spew.Dump(response)
		cblogger.Debug(ingressResponse.ToJsonString())
		callogger.Info(call.String(callLogInfo))
	}

	if len(securityGroupPolicyEgressSet.Egress) > 0 {
		egressRequest := vpc.NewDeleteSecurityGroupPoliciesRequest()
		egressRequest.SecurityGroupId = common.StringPtr(securityIID.SystemId)
		egressRequest.SecurityGroupPolicySet = securityGroupPolicyEgressSet

		callLogStart := call.Start()
		egressResponse, err := securityHandler.Client.DeleteSecurityGroupPolicies(egressRequest)
		callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

		if err != nil {
			callLogInfo.ErrorMSG = err.Error()
			callogger.Error(call.String(callLogInfo))

			cblogger.Error(err)
			return false, err
		}
		//spew.Dump(response)
		cblogger.Debug(egressResponse.ToJsonString())
		callogger.Info(call.String(callLogInfo))
	}

	securityInfo, errSecurity := securityHandler.GetSecurity(securityIID)
	cblogger.Debug(securityInfo)
	if errSecurity != nil {
		cblogger.Error(errSecurity)
		return false, errSecurity
	}
	return true, nil
}

// 동일한 rule이 있는지 체크
// RuleAction이 Add면 중복인 rule 리턴, Remove면 없는 rule 리턴
func sameRulesCheck(presentSecurityRules *[]irs.SecurityRuleInfo, reqSecurityRules *[]irs.SecurityRuleInfo, action RuleAction) (*[]irs.SecurityRuleInfo) {
	var checkResult []irs.SecurityRuleInfo
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
				cblogger.Info("add")
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


