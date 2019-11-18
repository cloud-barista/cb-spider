// Cloud Driver Interface of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Resouces interfaces of Cloud Driver.
//
// by powerkim@etri.re.kr, 2019.06.

package resources

import (
	"errors"
	"reflect"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type AwsSecurityHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

//2019-11-16부로 CB-Driver 전체 로직이 NameId 기반으로 변경됨. (보안 그룹은 그룹명으로 처리 가능하기 때문에 Name 태깅시 에러는 무시함)
//@TODO : 존재하는 보안 그룹에 정책 추가하는 기능 필요
//VPC 생략 시 활성화된 세션의 기본 VPC를 이용 함.
func (securityHandler *AwsSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	cblogger.Infof("securityReqInfo : ", securityReqInfo)
	spew.Dump(securityReqInfo)

	//VPC & Subnet을 자동으로 찾아서 처리
	vNetworkHandler := AwsVNetworkHandler{Client: securityHandler.Client}
	awsCBNetworkInfo, errAutoCBNetInfo := vNetworkHandler.GetAutoCBNetworkInfo()
	if errAutoCBNetInfo != nil || awsCBNetworkInfo.VpcId == "" {
		cblogger.Error("VPC 정보 획득 실패")
		return irs.SecurityInfo{}, errors.New("mcloud-barista의 기본 네트워크 정보를 찾을 수 없습니다.")
	}

	cblogger.Infof("==> [%s] CB Default VPC 정보 찾음", awsCBNetworkInfo.VpcId)
	vpcId := awsCBNetworkInfo.VpcId

	// Create the security group with the VPC, name and description.
	//createRes, err := securityHandler.Client.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
	input := ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(securityReqInfo.Name),
		Description: aws.String(securityReqInfo.Name),
		//		VpcId:       aws.String(securityReqInfo.VpcId),awsCBNetworkInfo
		VpcId: aws.String(vpcId),
	}
	cblogger.Debugf("보안 그룹 생성 요청 정보", input)
	createRes, err := securityHandler.Client.CreateSecurityGroup(&input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidVpcID.NotFound":
				cblogger.Errorf("Unable to find VPC with ID %q.", vpcId)
				return irs.SecurityInfo{}, err
			case "InvalidGroup.Duplicate":
				cblogger.Errorf("Security group %q already exists.", securityReqInfo.Name)
				return irs.SecurityInfo{}, err
			}
		}
		cblogger.Errorf("Unable to create security group %q, %v", securityReqInfo.Name, err)
		return irs.SecurityInfo{}, err
	}
	cblogger.Infof("[%s] 보안 그룹 생성완료", aws.StringValue(createRes.GroupId))
	spew.Dump(createRes)

	//newGroupId = *createRes.GroupId

	cblogger.Infof("인바운드 보안 정책 처리")
	//Ingress 처리
	var ipPermissions []*ec2.IpPermission
	for _, ip := range *securityReqInfo.SecurityRules {
		//for _, ip := range securityReqInfo.IPPermissions {
		if ip.Direction != "inbound" {
			cblogger.Debug("==> inbound가 아닌 보안 그룹 Skip : ", ip.Direction)
			continue
		}

		ipPermission := new(ec2.IpPermission)
		ipPermission.SetIpProtocol(ip.IPProtocol)

		if ip.FromPort != "" {
			if n, err := strconv.ParseInt(ip.FromPort, 10, 64); err == nil {
				ipPermission.SetFromPort(n)
			} else {
				cblogger.Error(ip.FromPort, "은 숫자가 아님!!")
				return irs.SecurityInfo{}, err
			}
		} else {
			//ipPermission.SetFromPort(0)
		}

		if ip.ToPort != "" {
			if n, err := strconv.ParseInt(ip.ToPort, 10, 64); err == nil {
				ipPermission.SetToPort(n)
			} else {
				cblogger.Error(ip.ToPort, "은 숫자가 아님!!")
				return irs.SecurityInfo{}, err
			}
		} else {
			//ipPermission.SetToPort(0)
		}

		ipPermission.SetIpRanges([]*ec2.IpRange{
			(&ec2.IpRange{}).
				//SetCidrIp(ip.Cidr),
				SetCidrIp("0.0.0.0/0"),
		})
		ipPermissions = append(ipPermissions, ipPermission)
	}

	//인바운드 정책이 있는 경우에만 처리
	if len(ipPermissions) > 0 {
		// Add permissions to the security group
		_, err = securityHandler.Client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
			//GroupName:     aws.String(securityReqInfo.Name),
			GroupId:       createRes.GroupId,
			IpPermissions: ipPermissions,
		})
		if err != nil {
			cblogger.Errorf("Unable to set security group %q ingress, %v", securityReqInfo.Name, err)
			return irs.SecurityInfo{}, err
		}

		cblogger.Info("Successfully set security group ingress")
	}

	cblogger.Infof("아웃바운드 보안 정책 처리")
	//Egress 처리
	var ipPermissionsEgress []*ec2.IpPermission
	//for _, ip := range securityReqInfo.IPPermissionsEgress {
	for _, ip := range *securityReqInfo.SecurityRules {
		if ip.Direction != "outbound" {
			cblogger.Debug("==> outbound가 아닌 보안 그룹 Skip : ", ip.Direction)
			continue
		}

		ipPermission := new(ec2.IpPermission)
		ipPermission.SetIpProtocol(ip.IPProtocol)
		//ipPermission.SetFromPort(ip.FromPort)
		//ipPermission.SetToPort(ip.ToPort)
		if ip.FromPort != "" {
			if n, err := strconv.ParseInt(ip.FromPort, 10, 64); err == nil {
				ipPermission.SetFromPort(n)
			} else {
				cblogger.Error(ip.FromPort, "은 숫자가 아님!!")
				return irs.SecurityInfo{}, err
			}
		} else {
			//ipPermission.SetFromPort(0)
		}

		if ip.ToPort != "" {
			if n, err := strconv.ParseInt(ip.ToPort, 10, 64); err == nil {
				ipPermission.SetToPort(n)
			} else {
				cblogger.Error(ip.ToPort, "은 숫자가 아님!!")
				return irs.SecurityInfo{}, err
			}
		} else {
			//ipPermission.SetToPort(0)
		}

		ipPermission.SetIpRanges([]*ec2.IpRange{
			(&ec2.IpRange{}).
				//SetCidrIp(ip.Cidr),
				SetCidrIp("0.0.0.0/0"),
		})
		//ipPermissions = append(ipPermissions, ipPermission)
		ipPermissionsEgress = append(ipPermissionsEgress, ipPermission)
	}

	//아웃바운드 정책이 있는 경우에만 처리
	if len(ipPermissionsEgress) > 0 {

		// Add permissions to the security group
		_, err = securityHandler.Client.AuthorizeSecurityGroupEgress(&ec2.AuthorizeSecurityGroupEgressInput{
			GroupId:       createRes.GroupId,
			IpPermissions: ipPermissionsEgress,
		})
		if err != nil {
			cblogger.Errorf("Unable to set security group %q egress, %v", securityReqInfo.Name, err)
			return irs.SecurityInfo{}, err
		}

		cblogger.Info("Successfully set security group egress")
	}

	cblogger.Info("Name Tag 처리")
	//======================
	// Name 태그 처리
	//======================
	//VPC Name 태깅
	tagInput := &ec2.CreateTagsInput{
		Resources: []*string{
			aws.String(*createRes.GroupId),
		},
		Tags: []*ec2.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String(securityReqInfo.Name),
			},
		},
	}
	//spew.Dump(tagInput)

	_, errTag := securityHandler.Client.CreateTags(tagInput)
	//Tag 실패 시 별도의 처리 없이 에러 로그만 남겨 놓음.
	if errTag != nil {
		cblogger.Error(errTag)
	}

	//securityInfo, _ := securityHandler.GetSecurity(*createRes.GroupId)
	securityInfo, _ := securityHandler.GetSecurity(securityReqInfo.Name) //2019-11-16 NameId 기반으로 변경됨
	return securityInfo, nil
}

func (securityHandler *AwsSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	//VPC ID 조회
	vNetworkHandler := AwsVNetworkHandler{Client: securityHandler.Client}
	vpcId := vNetworkHandler.GetMcloudBaristaDefaultVpcId()
	if vpcId == "" {
		return nil, nil
	}

	input := &ec2.DescribeSecurityGroupsInput{
		GroupIds: []*string{
			nil,
		},
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: aws.StringSlice([]string{vpcId}),
			},
		},
	}

	result, err := securityHandler.Client.DescribeSecurityGroups(input)
	//cblogger.Info("result : ", result)
	if err != nil {
		cblogger.Info("err : ", err)
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
		return nil, err
	}

	var results []*irs.SecurityInfo
	for _, securityGroup := range result.SecurityGroups {
		securityInfo := ExtractSecurityInfo(securityGroup)
		results = append(results, &securityInfo)
	}

	return results, nil
}

//2019-11-16부로 CB-Driver 전체 로직이 NameId 기반으로 변경됨.
func (securityHandler *AwsSecurityHandler) GetSecurity(securityNameId string) (irs.SecurityInfo, error) {
	cblogger.Infof("securityNameId : [%s]", securityNameId)
	input := &ec2.DescribeSecurityGroupsInput{
		/*
			GroupIds: []*string{
				aws.String(securityID),
			},
		*/
	}
	input.Filters = ([]*ec2.Filter{
		&ec2.Filter{
			//Name: aws.String("tag:Name"), // subnet-id
			Name: aws.String("group-name"), // subnet-id
			Values: []*string{
				aws.String(securityNameId),
			},
		},
	})
	cblogger.Info(input)

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

	if len(result.SecurityGroups) > 0 {
		securityInfo := ExtractSecurityInfo(result.SecurityGroups[0])
		return securityInfo, nil
	} else {
		//return irs.SecurityInfo{}, errors.New("[" + securityNameId + "] 정보를 찾을 수 없습니다.")
		return irs.SecurityInfo{}, errors.New("InvalidSecurityGroup.NotFound: The security group '" + securityNameId + "' does not exist")
	}
}

func ExtractSecurityInfo(securityGroupResult *ec2.SecurityGroup) irs.SecurityInfo {
	var ipPermissions []irs.SecurityRuleInfo
	var ipPermissionsEgress []irs.SecurityRuleInfo
	var securityRules []irs.SecurityRuleInfo

	cblogger.Debugf("===[그룹아이디:%s]===", *securityGroupResult.GroupId)
	ipPermissions = ExtractIpPermissions(securityGroupResult.IpPermissions, "inbound")
	cblogger.Debug("InBouds : ", ipPermissions)
	ipPermissionsEgress = ExtractIpPermissions(securityGroupResult.IpPermissionsEgress, "outbound")
	cblogger.Debug("OutBounds : ", ipPermissionsEgress)
	//spew.Dump(ipPermissionsEgress)
	securityRules = append(ipPermissions, ipPermissionsEgress...)

	securityInfo := irs.SecurityInfo{
		Id: *securityGroupResult.GroupId,
		//SecurityRules: &[]irs.SecurityRuleInfo{},
		SecurityRules: &securityRules,

		KeyValueList: []irs.KeyValue{
			{Key: "GroupName", Value: *securityGroupResult.GroupName},
			{Key: "VpcID", Value: *securityGroupResult.VpcId},
			{Key: "OwnerID", Value: *securityGroupResult.OwnerId},
			{Key: "Description", Value: *securityGroupResult.Description},
		},
	}

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
}

// IpPermission에서 공통정보 추출
func ExtractIpPermissionCommon(ip *ec2.IpPermission, securityRuleInfo *irs.SecurityRuleInfo) {
	//공통 정보
	if !reflect.ValueOf(ip.FromPort).IsNil() {
		//securityRuleInfo.FromPort = *ip.FromPort
		securityRuleInfo.FromPort = strconv.FormatInt(*ip.FromPort, 10)
	}

	if !reflect.ValueOf(ip.ToPort).IsNil() {
		//securityRuleInfo.ToPort = *ip.ToPort
		securityRuleInfo.ToPort = strconv.FormatInt(*ip.ToPort, 10)
	}

	securityRuleInfo.IPProtocol = *ip.IpProtocol
}

func ExtractIpPermissions(ipPermissions []*ec2.IpPermission, direction string) []irs.SecurityRuleInfo {
	var results []irs.SecurityRuleInfo

	for _, ip := range ipPermissions {

		//ipv4 처리
		for _, ipv4 := range ip.IpRanges {
			cblogger.Debug("Inbound/Outbound 정보 조회 : ", *ip.IpProtocol)
			securityRuleInfo := irs.SecurityRuleInfo{
				Direction: direction, // "inbound | outbound"
				//Cidr: *ipv4.CidrIp,
			}
			cblogger.Debug(*ipv4.CidrIp)

			ExtractIpPermissionCommon(ip, &securityRuleInfo) //IP & Port & Protocol 추출
			results = append(results, securityRuleInfo)
		}

		//ipv6 처리
		for _, ipv6 := range ip.Ipv6Ranges {
			securityRuleInfo := irs.SecurityRuleInfo{
				Direction: direction, // "inbound | outbound"
				//Cidr: *ipv6.CidrIpv6,
			}
			cblogger.Debug(*ipv6.CidrIpv6)

			ExtractIpPermissionCommon(ip, &securityRuleInfo) //IP & Port & Protocol 추출
			results = append(results, securityRuleInfo)
		}

		//ELB나 보안그룹 참조 방식 처리
		for _, userIdGroup := range ip.UserIdGroupPairs {
			securityRuleInfo := irs.SecurityRuleInfo{
				Direction: direction, // "inbound | outbound"
				//Cidr: *userIdGroup.GroupId,
			}
			cblogger.Debug(*userIdGroup.UserId)

			ExtractIpPermissionCommon(ip, &securityRuleInfo) //IP & Port & Protocol 추출
			results = append(results, securityRuleInfo)
		}

		/*  @TODO : 미지원 방식 체크 로직 추가 여부 결정해야 함.
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
		*/
	}

	return results
}

//2019-11-16부로 CB-Driver 전체 로직이 NameId 기반으로 변경됨.
func (securityHandler *AwsSecurityHandler) DeleteSecurity(securityNameId string) (bool, error) {
	cblogger.Infof("securityNameId : [%s]", securityNameId)

	securityInfo, errsecurityInfo := securityHandler.GetSecurity(securityNameId)
	if errsecurityInfo != nil {
		return false, errsecurityInfo
	}
	cblogger.Info(securityInfo)

	securityID := securityInfo.Id

	// Delete the security group.
	_, err := securityHandler.Client.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
		GroupId: aws.String(securityID),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidGroupId.Malformed":
				fallthrough
			case "InvalidGroup.NotFound":
				cblogger.Errorf("%s.", aerr.Message())
				return false, err
			}
		}
		cblogger.Errorf("Unable to get descriptions for security groups, %v.", err)
		return false, err
	}

	cblogger.Infof("Successfully delete security group %q.", securityID)

	return true, nil
}
