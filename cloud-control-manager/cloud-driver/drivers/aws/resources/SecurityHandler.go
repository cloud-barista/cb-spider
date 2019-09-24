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
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	idrv "github.com/cloud-barista/cb-spider/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
)

type AwsSecurityHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

//@TODO : 존재하는 보안 그룹에 정책 추가하는 기능 필요
//VPC 생략 시 활성화된 세션의 기본 VPC를 이용 함.
func (securityHandler *AwsSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	cblogger.Infof("securityReqInfo : ", securityReqInfo)
	spew.Dump(securityReqInfo)

	// Create the security group with the VPC, name and description.
	createRes, err := securityHandler.Client.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(securityReqInfo.GroupName),
		Description: aws.String(securityReqInfo.Description),
		VpcId:       aws.String(securityReqInfo.VpcId),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidVpcID.NotFound":
				cblogger.Errorf("Unable to find VPC with ID %q.", securityReqInfo.VpcId)
				return irs.SecurityInfo{}, err
			case "InvalidGroup.Duplicate":
				cblogger.Errorf("Security group %q already exists.", securityReqInfo.GroupName)
				return irs.SecurityInfo{}, err
			}
		}
		cblogger.Errorf("Unable to create security group %q, %v", securityReqInfo.GroupName, err)
		return irs.SecurityInfo{}, err
	}
	cblogger.Debug("보안 그룹 생성완료")
	spew.Dump(createRes)

	cblogger.Infof("Created security group %s with VPC %s.\n",
		aws.StringValue(createRes.GroupId), securityReqInfo.VpcId)

	//newGroupId = *createRes.GroupId

	//Ingress 처리
	var ipPermissions []*ec2.IpPermission
	for _, ip := range securityReqInfo.IPPermissions {
		ipPermission := new(ec2.IpPermission)
		ipPermission.SetIpProtocol(ip.IPProtocol)
		ipPermission.SetFromPort(ip.FromPort)
		ipPermission.SetToPort(ip.ToPort)
		ipPermission.SetIpRanges([]*ec2.IpRange{
			(&ec2.IpRange{}).
				SetCidrIp(ip.Cidr),
		})
		ipPermissions = append(ipPermissions, ipPermission)
	}

	// Add permissions to the security group
	_, err = securityHandler.Client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupName:     aws.String(securityReqInfo.GroupName),
		IpPermissions: ipPermissions,
	})
	if err != nil {
		cblogger.Errorf("Unable to set security group %q ingress, %v", securityReqInfo.GroupName, err)
		return irs.SecurityInfo{}, err
	}

	cblogger.Info("Successfully set security group ingress")

	//Egress 처리
	var ipPermissionsEgress []*ec2.IpPermission
	for _, ip := range securityReqInfo.IPPermissionsEgress {
		ipPermission := new(ec2.IpPermission)
		ipPermission.SetIpProtocol(ip.IPProtocol)
		ipPermission.SetFromPort(ip.FromPort)
		ipPermission.SetToPort(ip.ToPort)
		ipPermission.SetIpRanges([]*ec2.IpRange{
			(&ec2.IpRange{}).
				SetCidrIp(ip.Cidr),
		})
		ipPermissionsEgress = append(ipPermissionsEgress, ipPermission)
	}

	// Add permissions to the security group
	_, err = securityHandler.Client.AuthorizeSecurityGroupEgress(&ec2.AuthorizeSecurityGroupEgressInput{
		GroupId:       createRes.GroupId,
		IpPermissions: ipPermissionsEgress,
	})
	if err != nil {
		cblogger.Errorf("Unable to set security group %q egress, %v", securityReqInfo.GroupName, err)
		return irs.SecurityInfo{}, err
	}

	cblogger.Info("Successfully set security group egress")

	//return securityInfo, nil
	securityInfo, _ := securityHandler.GetSecurity(*createRes.GroupId)
	return securityInfo, nil
}

func (securityHandler *AwsSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	input := &ec2.DescribeSecurityGroupsInput{
		GroupIds: []*string{
			nil,
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

func (securityHandler *AwsSecurityHandler) GetSecurity(securityID string) (irs.SecurityInfo, error) {
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
}

func ExtractSecurityInfo(securityGroupResult *ec2.SecurityGroup) irs.SecurityInfo {
	var ipPermissions []*irs.SecurityRuleInfo
	var ipPermissionsEgress []*irs.SecurityRuleInfo

	cblogger.Info("===[그룹아이디:%s]===", *securityGroupResult.GroupId)
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
		securityRuleInfo.FromPort = *ip.FromPort
	}

	if !reflect.ValueOf(ip.ToPort).IsNil() {
		securityRuleInfo.ToPort = *ip.ToPort
	}

	securityRuleInfo.IPProtocol = *ip.IpProtocol
}

func ExtractIpPermissions(ipPermissions []*ec2.IpPermission) []*irs.SecurityRuleInfo {

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

//@TODO : CIDR이 없는 경우 구조처 처리해야 함.(예: 타겟이 ELB거나 다른 보안 그룹일 경우))
//@TODO : InBound / OutBound의 배열 처리및 테스트해야 함.
func _ExtractIpPermissions(ipPermissions []*ec2.IpPermission) []*irs.SecurityRuleInfo {

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
}

func (securityHandler *AwsSecurityHandler) DeleteSecurity(securityID string) (bool, error) {
	cblogger.Infof("securityID : [%s]", securityID)

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
