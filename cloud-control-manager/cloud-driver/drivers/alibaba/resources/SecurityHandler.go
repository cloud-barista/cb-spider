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

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	/*
		"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
		"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
		idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
		irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
		"github.com/davecgh/go-spew/spew"
	*/)

type AlibabaSecurityHandler struct {
	Region idrv.RegionInfo
	Client *ecs.Client
}

//@TODO : 존재하는 보안 그룹에 정책 추가하는 기능 필요
//VPC 생략 시 활성화된 세션의 기본 VPC를 이용 함.
func (securityHandler *AlibabaSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	return irs.SecurityInfo{}, errors.New("mcloud-barista의 기본 네트워크 정보를 찾을 수 없습니다.")
	/*
			cblogger.Infof("securityReqInfo : ", securityReqInfo)
			spew.Dump(securityReqInfo)

			request := ecs.CreateCreateSecurityGroupRequest()
		    request.Scheme = "https"

			//=========> 제거 시작 ========
			//VPC & Subnet을 자동으로 찾아서 처리
			vNetworkHandler := AlibabaVNetworkHandler{Client: securityHandler.Client}
			alibabaCBNetworkInfo, errAutoCBNetInfo := vNetworkHandler.GetAutoCBNetworkInfo()
			if errAutoCBNetInfo != nil || alibabaCBNetworkInfo.VpcId == "" {
				cblogger.Error("VPC 정보 획득 실패")
				return irs.SecurityInfo{}, errors.New("mcloud-barista의 기본 네트워크 정보를 찾을 수 없습니다.")
			}

			cblogger.Infof("==> [%s] CB Default VPC 정보 찾음", alibabaCBNetworkInfo.VpcId)
			vpcId := alibabaCBNetworkInfo.VpcId

		    vpcId := "" //@TODO : 구현해야 함.
			//=========> 제거 종료 ========

			request.Description = securityReqInfo.Name
		 	request.SecurityGroupName = securityReqInfo.Name
			request.VpcId = vpcId // "vpc-t4nevljk4rfqxa9n92vh7"

			cblogger.Debugf("보안 그룹 생성 요청 정보", request)

			// Create the security group with the VPC, name and description.
			createRes, err := securityHandler.Client.CreateSecurityGroup(request)
			if err != nil {
				cblogger.Errorf("Unable to create security group %q, %v", securityReqInfo.Name, err)
				return irs.SecurityInfo{}, err
			}
			cblogger.Debug("%s] 보안 그룹 생성완료", createRes.SecurityGroupId)
			spew.Dump(createRes)

			//newGroupId = *createRes.GroupId

			cblogger.Infof("인바운드 보안 정책 처리")
			//Security Group Rule 처리
			createRuleRes, err := securityHandler.AuthorizeSecurityRules(createRes.SecurityGroupId, securityReqInfo.SecurityRules)
			if err != nil {
				cblogger.Errorf("Unable to create security group rule %q, %v", securityReqInfo.Name, err)
				// return irs.SecurityRuleInfo{}, err
			} else {
				cblogger.Info("Successfully set security group egress")
			}

			//return securityInfo, nil
			securityInfo, _ := securityHandler.GetSecurity(createRes.SecurityGroupId)
			return securityInfo, nil
	*/
}

func (securityHandler *AlibabaSecurityHandler) AuthorizeSecurityRules(securityGroupId string, securityRuleInfos []*irs.SecurityRuleInfo) ([]*irs.SecurityRuleInfo, error) {
	return nil, nil
	/*
		cblogger.Infof("AuthorizeSecurityRules : ", securityRuleInfos)
		spew.Dump(securityRuleInfos)

		//var ipPermissionsEgress []*ec2.IpPermission
		//ecs.CreateDescribeSecurityGroupAttributeRequest
		var ipPermissionsEgress []ecs.Permission
		for _, securityRule := range securityRuleInfos {

			createRes, err := securityHandler.AuthorizeSecurityRule(securityGroupId, securityRule)
			if err != nil {
				cblogger.Errorf("Unable to create security group[%s] rule, %v", securityGroupId, err)
				return irs.SecurityRuleInfo{}, err
			}
		}
		return securityRuleInfo, nil
	*/
}

func (securityHandler *AlibabaSecurityHandler) AuthorizeSecurityRule(securityGroupId string, securityRuleReqInfo irs.SecurityRuleInfo) (irs.SecurityRuleInfo, error) {
	return irs.SecurityRuleInfo{}, nil
	/*
		cblogger.Infof("securityRuleReqInfo : ", securityRuleReqInfo)
		spew.Dump(securityRuleReqInfo)

		if securityRuleReqInfo.Direction != "ingress" { // egress
			request := ecs.CreateAuthorizeSecurityGroupEgressRequest()
			// request.DestCidrIp = "0.0.0.0/0"
		} else { // ingress
			request := ecs.CreateAuthorizeSecurityGroupRequest()
			// request.SourceCidrIp = "0.0.0.0/0"
		}
		request.Scheme = "https"

		request.SecurityGroupId = securityGroupId // "sg-t4n5d7znfsqs69xer1w2"
		request.IpProtocol = securityRuleReqInfo.IPProtocol // "tcp", "udp", "icmp", "gre", "all"
		request.PortRange = securityRuleReqInfo.FromPort + "/" + securityRuleReqInfo.ToPort // "1/200"
		// request.Policy = "accept"
		// request.NicType = "intranet"

		securityRuleReqInfoName = securityRuleReqInfo.Direction + " " + securityRuleReqInfo.IPProtocol + " " + securityRuleReqInfo.FromPort + "/" + securityRuleReqInfo.ToPort

		// Create the security group rule
		if securityRuleReqInfo.Direction != "ingress" { // egress
			createRes, err := securityHandler.Client.AuthorizeSecurityGroup(request)
			if err != nil {
				cblogger.Errorf("Unable to create security group rule egress %q, %v", securityRuleReqInfoName, err)
				return irs.SecurityRuleInfo{}, err
			}
		} else { // ingress
			createRes, err := securityHandler.Client.AuthorizeSecurityGroup(request)
			if err != nil {
				cblogger.Errorf("Unable to create security group rule ingress %q, %v", securityRuleReqInfoName, err)
				return irs.SecurityRuleInfo{}, err
			}
		}

		// createRes, err := securityHandler.Client.AuthorizeSecurityGroup(request)
		// if err != nil {
		// 	// if aerr, ok := err.(error.Error); ok {
		// 	// 	switch aerr.Code() {
		// 	// 	case "InvalidVpcID.NotFound":
		// 	// 		cblogger.Errorf("Unable to find VPC with ID %q.", VpcId)
		// 	// 		return irs.SecurityInfo{}, err
		// 	// 	case "InvalidGroup.Duplicate":
		// 	// 		cblogger.Errorf("Security group %q already exists.", securityRuleReqInfoName)
		// 	// 		return irs.SecurityInfo{}, err
		// 	// 	}
		// 	// }
		// 	cblogger.Errorf("Unable to create security group rule %q, %v", securityRuleReqInfoName, err)
		// 	return irs.SecurityRuleInfo{}, err
		// }

		securityRuleInfo := irs.SecurityRuleInfo{
			FromPort: *securityRuleReqInfo.FromPort,
			ToPort:   *securityRuleReqInfo.ToPort,
			IPProtocol: *securityRuleReqInfo.IPProtocol,
			Direction: *securityRuleReqInfo.Direction,
		}

		cblogger.Debug("%s] 보안 그룹 Rule 생성완료", securityRuleReqInfoName)
		spew.Dump(createRes)
		spew.Dump(securityRuleInfo)

		return securityRuleInfo, nil
	*/
}

func (securityHandler *AlibabaSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	//	return nil, nil

	// get SecurityGroup & SecurityGroupAttribute for Alibaba

	request := ecs.CreateDescribeSecurityGroupsRequest()
	request.Scheme = "https"

	request.VpcId = "vpc-t4nokxb60pv7ejm9ebsjr"
	request.PageNumber = requests.NewInteger(1)
	request.PageSize = requests.NewInteger(10)
	request.SecurityGroupIds = "[\"sg-t4n5d7znfsqs69xer1w2\"]"
	request.SecurityGroupId = "sg-t4n5d7znfsqs69xer1w2"
	request.SecurityGroupName = "sg-20191010"

	result, err := securityHandler.Client.DescribeSecurityGroups(request)
	spew.Dump(result)
	//cblogger.Info("result : ", result)
	if err != nil {
		return nil, err
	}

	var results []*irs.SecurityInfo
	for _, securityGroup := range result.SecurityGroups.SecurityGroup {
		securityInfo := ExtractSecurityInfo(&securityGroup)
		results = append(results, &securityInfo)
	}

	return results, nil

}

func (securityHandler *AlibabaSecurityHandler) GetSecurity(securityID string) (irs.SecurityInfo, error) {
	return irs.SecurityInfo{}, nil
	/*
		cblogger.Infof("securityID : [%s]", securityID)


		input := &ec2.DescribeSecurityGroupsInput{
			GroupIds: []*string{
				aws.String(securityID),
			},
		}

		result, err := securityHandler.Client.DescribeSecurityGroups(input)
		cblogger.Info("result : ", result)
		cblogger.Info("err : ", err)
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				default:
					cblogger.Error(aerr.Error())
				}
			} else {
				// Print the error, cast err to awserr.Error to get the Code and
				// Message from an error.
				cblogger.Error(err.Error())
			}
			return irs.SecurityInfo{}, err
		}

		securityInfo := ExtractSecurityInfo(result.SecurityGroups[0])
		return securityInfo, nil
	*/
}

func (securityHandler *AlibabaSecurityHandler) GetPermissions(securityID string) (ecs.Permissions, error) {
	cblogger.Infof("securityID : [%s]", securityID)
	return ecs.Permissions{}, nil
	/*
		request := ecs.CreateDescribeSecurityGroupAttributeRequest()
		request.Scheme = "https"

		request.SecurityGroupId = securityID // "sg-t4n5d7znfsqs69xer1w2"

		result, err := securityHandler.Client.DescribeSecurityGroupAttribute(request)
		cblogger.Info("result : ", result)
		cblogger.Info("err : ", err)
		if err != nil {
			return ecs.Permissions{}, err
		}
		securityPermissionInfo := ExtractPermissions(result.Permissions)
		return securityPermissionInfo, nil
	*/
}

func ExtractSecurityInfo(securityGroupResult *ecs.SecurityGroup) irs.SecurityInfo {
	return irs.SecurityInfo{}
	/*
		var ipPermissions []*irs.SecurityRuleInfo
		var ipPermissionsEgress []*irs.SecurityRuleInfo

		cblogger.Info("===[그룹아이디:%s]===", securityGroupResult.SecurityGroupId)
		ipPermissions = ExtractIpPermissions(securityGroupResult.IpPermissions)
		cblogger.Info("InBouds : ", ipPermissions)
		ipPermissionsEgress = ExtractIpPermissions(securityGroupResult.IpPermissionsEgress)
		cblogger.Info("OutBounds : ", ipPermissionsEgress)
		//spew.Dump(ipPermissionsEgress)

		securityInfo := irs.SecurityInfo{
			GroupName: *securityGroupResult.GroupName,
			GroupID:   *securityGroupResult.GroupId,

			IPPermissions:       ipPermissions,       //AWS:InBounds
			IPPermissionsEgress: ipPermissionsEgress, //AWS:OutBounds

			Description: *securityGroupResult.Description,
			VpcID:       *securityGroupResult.VpcId,
			OwnerID:     *securityGroupResult.OwnerId,
		}

		// get SecurityGroupAttribute
		permissions = GetPermissions(securityGroupResult.GroupId)

		//Name은 Tag의 "Name" 속성에만 저장됨
		cblogger.Debug("Name Tag 찾기")
		for _, t := range securityGroupResult.Tags {
			if *t.Key == "Name" {
				securityInfo.Name = *t.Value
				cblogger.Debug("Name : ", securityInfo.Name)
				break
			}
		}

		return securityInfo
	*/
}

// IpPermission에서 공통정보 추출
func ExtractIpPermissionCommon(ip *ecs.Permission, securityRuleInfo *irs.SecurityRuleInfo) {
	/*
		//공통 정보
		if !reflect.ValueOf(ip.FromPort).IsNil() {
			securityRuleInfo.FromPort = *ip.FromPort
		}

		if !reflect.ValueOf(ip.ToPort).IsNil() {
			securityRuleInfo.ToPort = *ip.ToPort
		}

		securityRuleInfo.IPProtocol = *ip.IpProtocol
	*/
}

func ExtractIpPermissions(ipPermissions []*ecs.Permission) []*irs.SecurityRuleInfo {
	return nil
	/*
		var results []*irs.SecurityRuleInfo

		for _, ip := range ipPermissions {

			//ipv4 처리
			for _, ipv4 := range ip.IpRanges {
				cblogger.Info("Inbound/Outbound 정보 조회 : ", *ip.IpProtocol)
				securityRuleInfo := new(irs.SecurityRuleInfo)
				securityRuleInfo.Cidr = *ipv4.CidrIp

				ExtractIpPermissionCommon(ip, securityRuleInfo)
				results = append(results, securityRuleInfo)
			}

			//ipv6 처리
			for _, ipv6 := range ip.Ipv6Ranges {
				securityRuleInfo := new(irs.SecurityRuleInfo)
				securityRuleInfo.Cidr = *ipv6.CidrIpv6

				ExtractIpPermissionCommon(ip, securityRuleInfo)
				results = append(results, securityRuleInfo)
			}

			//ELB나 보안그룹 참조 방식 처리
			for _, userIdGroup := range ip.UserIdGroupPairs {
				securityRuleInfo := new(irs.SecurityRuleInfo)
				securityRuleInfo.Cidr = *userIdGroup.GroupId
				// *userIdGroup.GroupName / *userIdGroup.UserId

				ExtractIpPermissionCommon(ip, securityRuleInfo)
				results = append(results, securityRuleInfo)
			}
		}

		return results
	*/
}

//@TODO : CIDR이 없는 경우 구조처 처리해야 함.(예: 타겟이 ELB거나 다른 보안 그룹일 경우))
//@TODO : InBound / OutBound의 배열 처리및 테스트해야 함.
func _ExtractIpPermissions(ipPermissions []*ecs.Permission) []*irs.SecurityRuleInfo {
	return nil
	/*
		var results []*irs.SecurityRuleInfo

		for _, ip := range ipPermissions {
			cblogger.Info("Inbound/Outbound 정보 조회 : ", *ip.IpProtocol)
			securityRuleInfo := new(irs.SecurityRuleInfo)

			if !reflect.ValueOf(ip.FromPort).IsNil() {
				securityRuleInfo.FromPort = *ip.FromPort
			}

			if !reflect.ValueOf(ip.ToPort).IsNil() {
				securityRuleInfo.ToPort = *ip.ToPort
			}

			//IpRanges가 없고 UserIdGroupPairs가 있는 경우가 있음(ELB / 보안 그룹 참조 등)
			securityRuleInfo.IPProtocol = *ip.IpProtocol

			if !reflect.ValueOf(ip.IpRanges).IsNil() {
				securityRuleInfo.Cidr = *ip.IpRanges[0].CidrIp
			} else {
				//ELB나 다른 보안그룹 참조처럼 IpRanges가 없고 UserIdGroupPairs가 있는 경우 처리
				//https://docs.aws.amazon.com/ko_kr/elasticloadbalancing/latest/classic/elb-security-groups.html
				if !reflect.ValueOf(ip.UserIdGroupPairs).IsNil() {
					securityRuleInfo.Cidr = *ip.UserIdGroupPairs[0].GroupId
				} else {
					cblogger.Error("미지원 보안 그룹 형태 발견 - 구조 파악 필요 ", ip)
				}
			}

			results = append(results, securityRuleInfo)
		}

		return results
	*/
}

func (securityHandler *AlibabaSecurityHandler) DeleteSecurity(securityID string) (bool, error) {
	cblogger.Infof("securityID : [%s]", securityID)
	/*
		// Delete the security group.
		_, err := securityHandler.Client.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(securityID),
		})
		if err != nil {
			cblogger.Errorf("Unable to get descriptions for security groups, %v.", err)
			return false, err
		}

		cblogger.Infof("Successfully delete security group %q.", securityID)
	*/
	return true, nil
}
