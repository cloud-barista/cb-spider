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
	"reflect"
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"strings"
	"time"
)

type AwsSecurityHandler struct {
	Region idrv.RegionInfo
	Client *ec2.EC2
}

// 2019-11-16부로 CB-Driver 전체 로직이 NameId 기반으로 변경됨. (보안 그룹은 그룹명으로 처리 가능하기 때문에 Name 태깅시 에러는 무시함)
// @TODO : 존재하는 보안 그룹에 정책 추가하는 기능 필요
// VPC 생략 시 활성화된 세션의 기본 VPC를 이용 함.
func (securityHandler *AwsSecurityHandler) CreateSecurity(securityReqInfo irs.SecurityReqInfo) (irs.SecurityInfo, error) {
	cblogger.Debugf("securityReqInfo : ", securityReqInfo)
	//cblogger.Debug(securityReqInfo)

	/*
		//VPC & Subnet을 자동으로 찾아서 처리
		VPCHandler := AwsVPCHandler{Client: securityHandler.Client}
		awsCBNetworkInfo, errAutoCBNetInfo := VPCHandler.GetAutoCBNetworkInfo()
		if errAutoCBNetInfo != nil || awsCBNetworkInfo.VpcId == "" {
			cblogger.Error("VPC 정보 획득 실패")
			return irs.SecurityInfo{}, errors.New("mcloud-barista의 기본 네트워크 정보를 찾을 수 없습니다.")
		}

		cblogger.Infof("==> [%s] CB Default VPC 정보 찾음", awsCBNetworkInfo.VpcId)
		vpcId := awsCBNetworkInfo.VpcId
	*/
	vpcId := securityReqInfo.VpcIID.SystemId

	// Create the security group with the VPC, name and description.
	//createRes, err := securityHandler.Client.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
	input := ec2.CreateSecurityGroupInput{
		//GroupName:   aws.String(securityReqInfo.Name),
		GroupName: aws.String(securityReqInfo.IId.NameId),
		//Description: aws.String(securityReqInfo.Name),
		Description: aws.String(securityReqInfo.IId.NameId),
		//		VpcId:       aws.String(securityReqInfo.VpcId),awsCBNetworkInfo
		VpcId: aws.String(vpcId),
	}
	cblogger.Debugf("Security group creation request information", input)
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: securityReqInfo.IId.NameId,
		CloudOSAPI:   "CreateSecurityGroup()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	createRes, err := securityHandler.Client.CreateSecurityGroup(&input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case "InvalidVpcID.NotFound":
				cblogger.Errorf("Unable to find VPC with ID %q.", vpcId)
				return irs.SecurityInfo{}, err
			case "InvalidGroup.Duplicate":
				cblogger.Errorf("Security group %q already exists.", securityReqInfo.IId.NameId)
				return irs.SecurityInfo{}, err
			}
		}
		cblogger.Errorf("Unable to create security group %q, %v", securityReqInfo.IId.NameId, err)
		return irs.SecurityInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))
	cblogger.Infof("[%s] Security group creation completed", aws.StringValue(createRes.GroupId))
	cblogger.Debug(createRes)
	//cblogger.Debug(createRes)

	//보안 그룹에 룰을 추가 함.
	_, err = securityHandler.ProcessAddRules(createRes.GroupId, securityReqInfo.SecurityRules)
	if err != nil {
		cblogger.Error(err)
		return irs.SecurityInfo{}, err
	}

	/*****
		//newGroupId = *createRes.GroupId

		cblogger.Debug("인바운드 보안 정책 처리")
		//Ingress 처리
		var ipPermissions []*ec2.IpPermission
		for _, ip := range *securityReqInfo.SecurityRules {
			//for _, ip := range securityReqInfo.IPPermissions {
			if ip.Direction != "inbound" {
				cblogger.Debug("==> inbound가 아닌 보안 그룹 Skip : ", ip.Direction)
				continue
			}

			// cblogger.Debug("===>변환중")
			// cblogger.Debug(ip)
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
					SetCidrIp(ip.CIDR),
				//SetCidrIp("0.0.0.0/0"),
			})
			// cblogger.Debug("===>변환완료")
			// cblogger.Debug(ipPermission)

			ipPermissions = append(ipPermissions, ipPermission)
		}

		//인바운드 정책이 있는 경우에만 처리
		if len(ipPermissions) > 0 {
			cblogger.Debug("===>적용할 최종 인바운드 정책")
			cblogger.Debug(ipPermissions)
			// cblogger.Debug(ipPermissions)

			// Add permissions to the security group
			_, err = securityHandler.Client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
				//GroupName:     aws.String(securityReqInfo.Name),
				GroupId:       createRes.GroupId,
				IpPermissions: ipPermissions,
			})
			if err != nil {
				cblogger.Errorf("Unable to set security group %q ingress, %v", securityReqInfo.IId.NameId, err)
				return irs.SecurityInfo{}, err
			}

			cblogger.Info("Successfully set security group ingress")
		}

		cblogger.Debug("아웃바운드 보안 정책 처리")
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
					SetCidrIp(ip.CIDR),
				//SetCidrIp("0.0.0.0/0"),
			})
			//ipPermissions = append(ipPermissions, ipPermission)
			ipPermissionsEgress = append(ipPermissionsEgress, ipPermission)
		}

		//아웃바운드 정책이 있는 경우에만 처리
		if len(ipPermissionsEgress) > 0 {
			cblogger.Debug("===>적용할 최종 아웃바운드 정책")
			cblogger.Debug(ipPermissionsEgress)

			// Add permissions to the security group
			_, err = securityHandler.Client.AuthorizeSecurityGroupEgress(&ec2.AuthorizeSecurityGroupEgressInput{
				GroupId:       createRes.GroupId,
				IpPermissions: ipPermissionsEgress,
			})
			if err != nil {
				cblogger.Errorf("Unable to set security group %q egress, %v", securityReqInfo.IId.NameId, err)
				return irs.SecurityInfo{}, err
			}

			cblogger.Info("Successfully set security group egress")
		}
	***/

	cblogger.Debug("Name Tag processing")
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
				Value: aws.String(securityReqInfo.IId.NameId),
			},
		},
	}
	//cblogger.Debug(tagInput)

	_, errTag := securityHandler.Client.CreateTags(tagInput)
	//Tag 실패 시 별도의 처리 없이 에러 로그만 남겨 놓음.
	if errTag != nil {
		cblogger.Error(errTag)
	}

	//securityInfo, _ := securityHandler.GetSecurity(*createRes.GroupId)
	//securityInfo, _ := securityHandler.GetSecurity(securityReqInfo.IId) //2019-11-16 NameId 기반으로 변경됨
	securityInfo, _ := securityHandler.GetSecurity(irs.IID{SystemId: *createRes.GroupId}) //2020-04-09 SystemId기반으로 변경
	securityInfo.IId.NameId = securityReqInfo.IId.NameId                                  // Name이 필수가 아니므로 혹시 모르니 사용자가 요청한 NameId로 재설정 함.
	securityInfo.VpcIID.NameId = securityReqInfo.VpcIID.NameId                            // Name이 필수가 아니므로 객체에 저장되지 않기 때문에 시스템에서 활용 가능하도록 사용자가 요청한 NameId 값을 그대로 돌려 줌.
	return securityInfo, nil
}

func (securityHandler *AwsSecurityHandler) ListSecurity() ([]*irs.SecurityInfo, error) {
	//VPC ID 조회
	/* 2020-04-13 : 전체 영역에서 조회하도록 변경
	VPCHandler := AwsVPCHandler{Client: securityHandler.Client}
	vpcId := VPCHandler.GetMcloudBaristaDefaultVpcId()
	if vpcId == "" {
		return nil, nil
	}
	*/

	input := &ec2.DescribeSecurityGroupsInput{
		GroupIds: []*string{
			nil,
		},
		/*
			Filters: []*ec2.Filter{
				{
					Name:   aws.String("vpc-id"),
					Values: aws.StringSlice([]string{vpcId}),
				},
			},
		*/
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: "List()",
		CloudOSAPI:   "DescribeSecurityGroups()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := securityHandler.Client.DescribeSecurityGroups(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Info("result : ", result)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

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
	callogger.Info(call.String(callLogInfo))

	var results []*irs.SecurityInfo
	for _, securityGroup := range result.SecurityGroups {
		securityInfo := ExtractSecurityInfo(securityGroup)
		results = append(results, &securityInfo)
	}

	return results, nil
}

// 2019-11-16부로 CB-Driver 전체 로직이 NameId 기반으로 변경됨.
// func (securityHandler *AwsSecurityHandler) GetSecurity(securityNameId string) (irs.SecurityInfo, error) {
func (securityHandler *AwsSecurityHandler) GetSecurity(securityIID irs.IID) (irs.SecurityInfo, error) {
	cblogger.Infof("securityNameId : [%s]", securityIID.SystemId)

	//2020-04-09 Filter 대신 SystemId 기반으로 변경
	input := &ec2.DescribeSecurityGroupsInput{
		GroupIds: []*string{
			aws.String(securityIID.SystemId),
		},
	}
	/* 2020-04-09 Name 기반으로 조회 하지 않기때문에 미사용
	input.Filters = ([]*ec2.Filter{
		&ec2.Filter{
			//Name: aws.String("tag:Name"), // subnet-id
			Name: aws.String("group-name"), // subnet-id
			Values: []*string{
				aws.String(securityIID.SystemId),
			},
		},
	})
	*/
	cblogger.Debug(input)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: securityIID.SystemId,
		CloudOSAPI:   "DescribeSecurityGroups()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := securityHandler.Client.DescribeSecurityGroups(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Debug("result : ", result)
	cblogger.Debug("err : ", err)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

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
	callogger.Info(call.String(callLogInfo))

	if len(result.SecurityGroups) > 0 {
		securityInfo := ExtractSecurityInfo(result.SecurityGroups[0])
		return securityInfo, nil
	} else {
		//return irs.SecurityInfo{}, errors.New("[" + securityNameId + "] 정보를 찾을 수 없습니다.")
		return irs.SecurityInfo{}, errors.New("InvalidSecurityGroup.NotFound: The security group '" + securityIID.SystemId + "' does not exist")
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
	//cblogger.Debug(ipPermissionsEgress)
	securityRules = append(ipPermissions, ipPermissionsEgress...)

	securityInfo := irs.SecurityInfo{
		//Id: *securityGroupResult.GroupId,
		IId: irs.IID{"", *securityGroupResult.GroupId},
		//SecurityRules: &[]irs.SecurityRuleInfo{},
		SecurityRules: &securityRules,
		VpcIID:        irs.IID{"", *securityGroupResult.VpcId},

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
			//securityInfo.Name = *t.Value
			securityInfo.IId.NameId = *t.Value
			cblogger.Debug("Name : ", securityInfo.IId.NameId)
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

	//이슈 #642 처리 - cb보안 그룹 출력 규칙
	//https://github.com/cloud-barista/cb-spider/wiki/Security-Group-Rules-and-Driver-API
	securityRuleInfo.IPProtocol = *ip.IpProtocol
	if securityRuleInfo.IPProtocol == "-1" {
		securityRuleInfo.IPProtocol = "ALL"
		securityRuleInfo.FromPort = "-1"
		securityRuleInfo.ToPort = "-1"
	}
}

func ExtractIpPermissions(ipPermissions []*ec2.IpPermission, direction string) []irs.SecurityRuleInfo {
	var results []irs.SecurityRuleInfo

	for _, ip := range ipPermissions {

		//ipv4 처리
		for _, ipv4 := range ip.IpRanges {
			cblogger.Debug("Inbound/Outbound information retrieval: ", *ip.IpProtocol)
			securityRuleInfo := irs.SecurityRuleInfo{
				Direction: direction, // "inbound | outbound"
				CIDR:      *ipv4.CidrIp,
			}
			cblogger.Debug(*ipv4.CidrIp)

			ExtractIpPermissionCommon(ip, &securityRuleInfo) //IP & Port & Protocol 추출
			results = append(results, securityRuleInfo)
		}

		//ipv6 처리
		for _, ipv6 := range ip.Ipv6Ranges {
			securityRuleInfo := irs.SecurityRuleInfo{
				Direction: direction, // "inbound | outbound"
				CIDR:      *ipv6.CidrIpv6,
			}
			cblogger.Debug(*ipv6.CidrIpv6)

			ExtractIpPermissionCommon(ip, &securityRuleInfo) //IP & Port & Protocol 추출
			results = append(results, securityRuleInfo)
		}

		//ELB나 보안그룹 참조 방식 처리
		for _, userIdGroup := range ip.UserIdGroupPairs {
			securityRuleInfo := irs.SecurityRuleInfo{
				Direction: direction, // "inbound | outbound"
				CIDR:      *userIdGroup.GroupId,
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

// 2019-11-16부로 CB-Driver 전체 로직이 NameId 기반으로 변경됨.
// func (securityHandler *AwsSecurityHandler) DeleteSecurity(securityNameId string) (bool, error) {
func (securityHandler *AwsSecurityHandler) DeleteSecurity(securityIID irs.IID) (bool, error) {
	cblogger.Infof("securityNameId : [%s]", securityIID.SystemId)

	/* //2020-04-09 SystemId 기반으로 변경되어서 필요 없음.
	securityInfo, errsecurityInfo := securityHandler.GetSecurity(securityIID)
	if errsecurityInfo != nil {
		return false, errsecurityInfo
	}
	cblogger.Info(securityInfo)
	*/

	//securityID := securityInfo.Id
	//securityID := securityInfo.IId.SystemId
	securityID := securityIID.SystemId

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   securityHandler.Region.Zone,
		ResourceType: call.SECURITYGROUP,
		ResourceName: securityIID.SystemId,
		CloudOSAPI:   "DeleteSecurityGroup()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	// Delete the security group.
	//_, err := securityHandler.Client.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
	//	GroupId: aws.String(securityID),
	//})
	err := loopDeleteSecurityGroup(securityHandler.Client, &ec2.DeleteSecurityGroupInput{
		GroupId: aws.String(securityID),
	})
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

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
	callogger.Info(call.String(callLogInfo))

	cblogger.Infof("Successfully delete security group %q.", securityID)

	return true, nil
}

// wait to resolve the 'DependencyViolation' error
func loopDeleteSecurityGroup(client *ec2.EC2, input *ec2.DeleteSecurityGroupInput) error {

	var err error

	maxRetryCnt := 40 // retry until 120s
	for i := 0; i < maxRetryCnt; i++ {
		_, err = client.DeleteSecurityGroup(input)
		if err == nil {
			return nil
		}
		if strings.Contains(err.Error(), "DependencyViolation") {
			time.Sleep(time.Second * 3)
		} else {
			return err
		}
	}
	return err
}

func (securityHandler *AwsSecurityHandler) AddRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	cblogger.Debugf("AddRules : SecurityNameId : [%s]", sgIID.SystemId)
	// 존재하는 보안 그룹인지 확인
	ruleInfo, err := securityHandler.GetSecurity(sgIID)
	if err != nil {
		cblogger.Error(err)
		return irs.SecurityInfo{}, err
	}
	cblogger.Debug("GetSecurity Result : ", ruleInfo)

	//보안 그룹에 룰 추가 처리
	securityInfo, err := securityHandler.ProcessAddRules(&sgIID.SystemId, securityRules)
	if err != nil {
		cblogger.Error(err)
		return irs.SecurityInfo{}, err
	}

	//최종 정보 리턴
	return securityInfo, nil

	//return securityHandler.GetSecurity(sgIID)
}

func (securityHandler *AwsSecurityHandler) ProcessAddRules(newGroupId *string, securityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	var err error

	cblogger.Debug("Inbound security policy processing")
	//Ingress 처리
	var ipPermissions []*ec2.IpPermission
	for _, ip := range *securityRules {
		//for _, ip := range securityReqInfo.IPPermissions {
		if ip.Direction != "inbound" {
			cblogger.Debug("==> Skipping security group that is not inbound: ", ip.Direction)
			continue
		}

		// cblogger.Debug("===>변환중")
		// cblogger.Debug(ip)
		ipPermission := new(ec2.IpPermission)
		ipPermission.SetIpProtocol(ip.IPProtocol)

		if ip.FromPort != "" {
			if n, err := strconv.ParseInt(ip.FromPort, 10, 64); err == nil {
				ipPermission.SetFromPort(n)
			} else {
				cblogger.Error(ip.FromPort, "is not number!!")
				return irs.SecurityInfo{}, err
			}
		} else {
			//ipPermission.SetFromPort(0)
		}

		if ip.ToPort != "" {
			if n, err := strconv.ParseInt(ip.ToPort, 10, 64); err == nil {
				ipPermission.SetToPort(n)
			} else {
				cblogger.Error(ip.ToPort, "is not number!!")
				return irs.SecurityInfo{}, err
			}
		} else {
			//ipPermission.SetToPort(0)
		}

		ipPermission.SetIpRanges([]*ec2.IpRange{
			(&ec2.IpRange{}).
				SetCidrIp(ip.CIDR),
			//SetCidrIp("0.0.0.0/0"),
		})
		// cblogger.Debug("===>변환완료")
		// cblogger.Debug(ipPermission)

		ipPermissions = append(ipPermissions, ipPermission)
	}

	//인바운드 정책이 있는 경우에만 처리
	if len(ipPermissions) > 0 {
		cblogger.Debug("===> Final inbound policy to apply")
		cblogger.Debug(ipPermissions)
		// cblogger.Debug(ipPermissions)

		// Add permissions to the security group
		_, err = securityHandler.Client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
			//GroupName:     aws.String(securityReqInfo.Name),
			GroupId:       newGroupId, //createRes.GroupId,
			IpPermissions: ipPermissions,
		})
		if err != nil {
			cblogger.Errorf("Unable to set security group %q ingress, %v", *newGroupId, err)
			return irs.SecurityInfo{}, err
		}

		cblogger.Info("Successfully set security group ingress")
	}

	cblogger.Debug("Outbound security policy processing")
	//Egress 처리
	var ipPermissionsEgress []*ec2.IpPermission
	//for _, ip := range securityReqInfo.IPPermissionsEgress {
	for _, ip := range *securityRules {
		if ip.Direction != "outbound" {
			cblogger.Debug("==> Skipping security group that is not outbound: ", ip.Direction)
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
				cblogger.Error(ip.FromPort, "is not number!!")
				return irs.SecurityInfo{}, err
			}
		} else {
			//ipPermission.SetFromPort(0)
		}

		if ip.ToPort != "" {
			if n, err := strconv.ParseInt(ip.ToPort, 10, 64); err == nil {
				ipPermission.SetToPort(n)
			} else {
				cblogger.Error(ip.ToPort, "is not number!!")
				return irs.SecurityInfo{}, err
			}
		} else {
			//ipPermission.SetToPort(0)
		}

		ipPermission.SetIpRanges([]*ec2.IpRange{
			(&ec2.IpRange{}).
				SetCidrIp(ip.CIDR),
			//SetCidrIp("0.0.0.0/0"),
		})
		//ipPermissions = append(ipPermissions, ipPermission)
		ipPermissionsEgress = append(ipPermissionsEgress, ipPermission)
	}

	//아웃바운드 정책이 있는 경우에만 처리
	if len(ipPermissionsEgress) > 0 {
		cblogger.Debug("===> Final outbound policy to apply")
		cblogger.Debug(ipPermissionsEgress)

		// Add permissions to the security group
		_, err = securityHandler.Client.AuthorizeSecurityGroupEgress(&ec2.AuthorizeSecurityGroupEgressInput{
			GroupId:       newGroupId, //createRes.GroupId,
			IpPermissions: ipPermissionsEgress,
		})
		if err != nil {
			cblogger.Errorf("Unable to set security group %q egress, %v", *newGroupId, err)
			return irs.SecurityInfo{}, err
		}

		cblogger.Info("Successfully set security group egress")
	}

	return securityHandler.GetSecurity(irs.IID{SystemId: *newGroupId})
}

func (securityHandler *AwsSecurityHandler) RemoveRules(sgIID irs.IID, securityRules *[]irs.SecurityRuleInfo) (bool, error) {
	cblogger.Debugf("RemoveRules : SecurityNameId : [%s]", sgIID.SystemId)

	// 존재하는 보안 그룹인지 확인
	ruleInfo, err := securityHandler.GetSecurity(sgIID)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}
	cblogger.Debug("GetSecurity Result : ", ruleInfo)

	//보안 그룹의 룰 삭제 처리
	_, err = securityHandler.ProcessRemoveRules(&sgIID.SystemId, securityRules)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	//최종 정보 리턴
	return true, nil
}

func (securityHandler *AwsSecurityHandler) ProcessRemoveRules(newGroupId *string, securityRules *[]irs.SecurityRuleInfo) (irs.SecurityInfo, error) {
	var err error

	cblogger.Debug("Inbound security policy processing")
	//Ingress 처리
	var ipPermissions []*ec2.IpPermission
	for _, ip := range *securityRules {
		//for _, ip := range securityReqInfo.IPPermissions {
		if ip.Direction != "inbound" {
			cblogger.Debug("==> Skipping security group that is not inbound: ", ip.Direction)
			continue
		}

		// cblogger.Debug("===>변환중")
		// cblogger.Debug(ip)
		ipPermission := new(ec2.IpPermission)
		ipPermission.SetIpProtocol(ip.IPProtocol)

		if ip.FromPort != "" {
			if n, err := strconv.ParseInt(ip.FromPort, 10, 64); err == nil {
				ipPermission.SetFromPort(n)
			} else {
				cblogger.Error(ip.FromPort, "is not number!!")
				return irs.SecurityInfo{}, err
			}
		} else {
			//ipPermission.SetFromPort(0)
		}

		if ip.ToPort != "" {
			if n, err := strconv.ParseInt(ip.ToPort, 10, 64); err == nil {
				ipPermission.SetToPort(n)
			} else {
				cblogger.Error(ip.ToPort, "is not number!!")
				return irs.SecurityInfo{}, err
			}
		} else {
			//ipPermission.SetToPort(0)
		}

		ipPermission.SetIpRanges([]*ec2.IpRange{
			(&ec2.IpRange{}).
				SetCidrIp(ip.CIDR),
			//SetCidrIp("0.0.0.0/0"),
		})
		// cblogger.Debug("===>변환완료")
		// cblogger.Debug(ipPermission)

		ipPermissions = append(ipPermissions, ipPermission)
	}

	//인바운드 정책이 있는 경우에만 처리
	if len(ipPermissions) > 0 {
		cblogger.Debug("===> Final inbound policy to apply")
		cblogger.Debug(ipPermissions)
		// cblogger.Debug(ipPermissions)

		// Add permissions to the security group
		_, err = securityHandler.Client.RevokeSecurityGroupIngress(&ec2.RevokeSecurityGroupIngressInput{
			//GroupName:     aws.String(securityReqInfo.Name),
			GroupId:       newGroupId, //createRes.GroupId,
			IpPermissions: ipPermissions,
		})
		if err != nil {
			cblogger.Errorf("Unable to set security group %q ingress, %v", *newGroupId, err)
			return irs.SecurityInfo{}, err
		}

		cblogger.Info("Successfully set security group ingress")
	}

	cblogger.Debug("Outbound security policy processing")
	//Egress 처리
	var ipPermissionsEgress []*ec2.IpPermission
	//for _, ip := range securityReqInfo.IPPermissionsEgress {
	for _, ip := range *securityRules {
		if ip.Direction != "outbound" {
			cblogger.Debug("==> Skipping security group that is not outbound: ", ip.Direction)
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
				cblogger.Error(ip.FromPort, "is not number!!")
				return irs.SecurityInfo{}, err
			}
		} else {
			//ipPermission.SetFromPort(0)
		}

		if ip.ToPort != "" {
			if n, err := strconv.ParseInt(ip.ToPort, 10, 64); err == nil {
				ipPermission.SetToPort(n)
			} else {
				cblogger.Error(ip.ToPort, "is not number!!")
				return irs.SecurityInfo{}, err
			}
		} else {
			//ipPermission.SetToPort(0)
		}

		ipPermission.SetIpRanges([]*ec2.IpRange{
			(&ec2.IpRange{}).
				SetCidrIp(ip.CIDR),
			//SetCidrIp("0.0.0.0/0"),
		})
		//ipPermissions = append(ipPermissions, ipPermission)
		ipPermissionsEgress = append(ipPermissionsEgress, ipPermission)
	}

	//아웃바운드 정책이 있는 경우에만 처리
	if len(ipPermissionsEgress) > 0 {
		cblogger.Debug("===> Final outbound policy to apply")
		cblogger.Debug(ipPermissionsEgress)

		// Add permissions to the security group
		_, err = securityHandler.Client.RevokeSecurityGroupEgress(&ec2.RevokeSecurityGroupEgressInput{
			GroupId:       newGroupId, //createRes.GroupId,
			IpPermissions: ipPermissionsEgress,
		})
		if err != nil {
			cblogger.Errorf("Unable to set security group %q egress, %v", *newGroupId, err)
			return irs.SecurityInfo{}, err
		}

		cblogger.Info("Successfully set security group egress")
	}

	return securityHandler.GetSecurity(irs.IID{SystemId: *newGroupId})
}
