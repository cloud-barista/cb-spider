package resources

//https://docs.aws.amazon.com/sdk-for-go/api/service/elb

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/davecgh/go-spew/spew"

	//"github.com/aws/aws-sdk-go/service/elb"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/elbv2"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsNLBHandler struct {
	Region idrv.RegionInfo
	//Client *elb.ELB
	Client   *elbv2.ELBV2 //elbV2
	VMClient *ec2.EC2
}

type TargetGroupInfo struct {
	VMGroup       irs.VMGroupInfo
	HealthChecker irs.HealthCheckerInfo
	/*
		IId		IID 	// {NameId, SystemId}
		VpcIID		IID	// {NameId, SystemId}

		Type		string	// PUBLIC(V) | INTERNAL
		Scope		string	// REGION(V) | GLOBAL

		//------ Frontend
		Listener	ListenerInfo

		//------ Backend
	*/
}

func (NLBHandler *AwsNLBHandler) CreateListener(nlbReqInfo irs.NLBInfo) (*elbv2.CreateListenerOutput, error) {
	input := &elbv2.CreateListenerInput{
		DefaultActions: []*elbv2.Action{
			{
				TargetGroupArn: aws.String(nlbReqInfo.VMGroup.CspID), //생성된 VMGroup(타겟그룹)의 ARN
				Type:           aws.String("forward"),
			},
		},
		LoadBalancerArn: aws.String(nlbReqInfo.IId.SystemId), //생성된 NLB의 ARN 값
		//Port:            aws.Int64(80), //숫자 값 검증 후 적용
		Protocol: aws.String(nlbReqInfo.Listener.Protocol), // AWS NLB : TCP, TLS, UDP, or TCP_UDP
	}

	//리스너 포트 포메팅 검증 및 셋팅
	if nlbReqInfo.Listener.Port != "" {
		if n, err := strconv.ParseInt(nlbReqInfo.Listener.Port, 10, 64); err == nil {
			input.SetPort(n)
		} else {
			cblogger.Error(nlbReqInfo.Listener.Port, "은 숫자가 아님!!")
			return nil, err
		}
	} else {
		return nil, errors.New("InvalidNumberFormat : Listener.Port is null")
	}

	result, err := NLBHandler.Client.CreateListener(input)
	cblogger.Debug(result)

	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeDuplicateListenerException:
				cblogger.Error(elbv2.ErrCodeDuplicateListenerException, aerr.Error())
			case elbv2.ErrCodeTooManyListenersException:
				cblogger.Error(elbv2.ErrCodeTooManyListenersException, aerr.Error())
			case elbv2.ErrCodeTooManyCertificatesException:
				cblogger.Error(elbv2.ErrCodeTooManyCertificatesException, aerr.Error())
			case elbv2.ErrCodeLoadBalancerNotFoundException:
				cblogger.Error(elbv2.ErrCodeLoadBalancerNotFoundException, aerr.Error())
			case elbv2.ErrCodeTargetGroupNotFoundException:
				cblogger.Error(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
			case elbv2.ErrCodeTargetGroupAssociationLimitException:
				cblogger.Error(elbv2.ErrCodeTargetGroupAssociationLimitException, aerr.Error())
			case elbv2.ErrCodeInvalidConfigurationRequestException:
				cblogger.Error(elbv2.ErrCodeInvalidConfigurationRequestException, aerr.Error())
			case elbv2.ErrCodeIncompatibleProtocolsException:
				cblogger.Error(elbv2.ErrCodeIncompatibleProtocolsException, aerr.Error())
			case elbv2.ErrCodeSSLPolicyNotFoundException:
				cblogger.Error(elbv2.ErrCodeSSLPolicyNotFoundException, aerr.Error())
			case elbv2.ErrCodeCertificateNotFoundException:
				cblogger.Error(elbv2.ErrCodeCertificateNotFoundException, aerr.Error())
			case elbv2.ErrCodeUnsupportedProtocolException:
				cblogger.Error(elbv2.ErrCodeUnsupportedProtocolException, aerr.Error())
			case elbv2.ErrCodeTooManyRegistrationsForTargetIdException:
				cblogger.Error(elbv2.ErrCodeTooManyRegistrationsForTargetIdException, aerr.Error())
			case elbv2.ErrCodeTooManyTargetsException:
				cblogger.Error(elbv2.ErrCodeTooManyTargetsException, aerr.Error())
			case elbv2.ErrCodeTooManyActionsException:
				cblogger.Error(elbv2.ErrCodeTooManyActionsException, aerr.Error())
			case elbv2.ErrCodeInvalidLoadBalancerActionException:
				cblogger.Error(elbv2.ErrCodeInvalidLoadBalancerActionException, aerr.Error())
			case elbv2.ErrCodeTooManyUniqueTargetGroupsPerLoadBalancerException:
				cblogger.Error(elbv2.ErrCodeTooManyUniqueTargetGroupsPerLoadBalancerException, aerr.Error())
			case elbv2.ErrCodeALPNPolicyNotSupportedException:
				cblogger.Error(elbv2.ErrCodeALPNPolicyNotSupportedException, aerr.Error())
			case elbv2.ErrCodeTooManyTagsException:
				cblogger.Error(elbv2.ErrCodeTooManyTagsException, aerr.Error())
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

	cblogger.Debug("Listener 생성 결과")
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	return result, nil
}

func (NLBHandler *AwsNLBHandler) CreateTargetGroup(nlbReqInfo irs.NLBInfo) (*elbv2.CreateTargetGroupOutput, error) {
	input := &elbv2.CreateTargetGroupInput{
		Name:       aws.String(nlbReqInfo.IId.NameId),
		TargetType: aws.String("instance"), // instance , ip, lambda
		//Port:     aws.Int64(80),	//숫자 값 검증 후 적용

		Protocol: aws.String(nlbReqInfo.VMGroup.Protocol),
		VpcId:    aws.String(nlbReqInfo.VpcIID.SystemId),

		//헬스체크
		HealthCheckProtocol: aws.String(nlbReqInfo.HealthChecker.Protocol),
		HealthCheckPort:     aws.String(nlbReqInfo.HealthChecker.Port),
		//HealthCheckIntervalSeconds: aws.Int64(int64(nlbReqInfo.HealthChecker.Interval)), // 5초이상	// 0 이상의 값이 있을 때만 설정하도록 변경
		//HealthCheckTimeoutSeconds: aws.Int64(int64(nlbReqInfo.HealthChecker.Timeout)), // 0 이상의 값이 있을 때만 설정하도록 변경
	}

	//AWS TargetGroup 포트 포메팅 검증 및 셋팅
	if nlbReqInfo.VMGroup.Port != "" {
		if n, err := strconv.ParseInt(nlbReqInfo.VMGroup.Port, 10, 64); err == nil {
			input.SetPort(n)
		} else {
			cblogger.Error(nlbReqInfo.VMGroup.Port, "은 숫자가 아님!!")
			return nil, err
		}
	} else {
		return nil, errors.New("InvalidNumberFormat : VMGroup.Port is null")
	}

	//============
	//헬스체크
	//============
	// 인터벌 설정
	// Health check interval '60' not supported for target groups with the TCP protocol. Must be one of the following values '[10, 30]'.
	if nlbReqInfo.HealthChecker.Interval > 0 {
		input.HealthCheckIntervalSeconds = aws.Int64(int64(nlbReqInfo.HealthChecker.Interval))
	}

	// 타임아웃 설정 - TCP는 타임아웃 설정 기능 미지원. (HTTP는 설정 가능 하지만 NLB라서 TCP 외에는 셋팅 불가)
	if nlbReqInfo.HealthChecker.Timeout > 0 {
		input.HealthCheckTimeoutSeconds = aws.Int64(int64(nlbReqInfo.HealthChecker.Timeout))
	}

	// Threshold 설정
	if nlbReqInfo.HealthChecker.Threshold > 0 {
		input.HealthyThresholdCount = aws.Int64(int64(nlbReqInfo.HealthChecker.Threshold))

		//TCP는 HealthyThresholdCount와 UnhealthyThresholdCount 값을 동일하게 설정해야 함.
		if strings.EqualFold(nlbReqInfo.HealthChecker.Protocol, "TCP") {
			input.UnhealthyThresholdCount = aws.Int64(int64(nlbReqInfo.HealthChecker.Threshold))
		}
	}

	result, err := NLBHandler.Client.CreateTargetGroup(input)
	cblogger.Debug(result)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeDuplicateTargetGroupNameException:
				cblogger.Error(elbv2.ErrCodeDuplicateTargetGroupNameException, aerr.Error())
			case elbv2.ErrCodeTooManyTargetGroupsException:
				cblogger.Error(elbv2.ErrCodeTooManyTargetGroupsException, aerr.Error())
			case elbv2.ErrCodeInvalidConfigurationRequestException:
				cblogger.Error(elbv2.ErrCodeInvalidConfigurationRequestException, aerr.Error())
			case elbv2.ErrCodeTooManyTagsException:
				cblogger.Error(elbv2.ErrCodeTooManyTagsException, aerr.Error())
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

	cblogger.Debug("TargetGroup 생성 결과")
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	return result, nil
}

func (NLBHandler *AwsNLBHandler) ExtractVmSubnets(VMs *[]irs.IID) ([]*string, error) {
	cblogger.Debug(VMs)
	if len(*VMs) == 0 {
		return nil, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "No VM information to query.", nil)
	}

	input := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{},
	}

	for _, cur := range *VMs {
		input.InstanceIds = append(input.InstanceIds, aws.String(cur.SystemId))
	}

	result, err := NLBHandler.VMClient.DescribeInstances(input)
	if err != nil {
		cblogger.Error(err.Error())
	}

	cblogger.Debug(result)
	cblogger.Infof("조회된 VM 정보 수 : [%d]", len(result.Reservations))
	if len(result.Reservations) == 0 {
		cblogger.Error("조회된 VM 정보가 없습니다.")
		return nil, awserr.New(CUSTOM_ERR_CODE_NOTFOUND, "VM information was not found.", nil)
	}

	if len(*VMs) != len(result.Reservations) {
		cblogger.Errorf("요청된 VM 수[%d]와 조회된 VM 수[%d]가 일치하지 않습니다.", len(*VMs), len(result.Reservations))
		return nil, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, fmt.Sprintf("Requested number of VMs [%d] and queried number of VMs [%d] do not match.", len(*VMs), len(result.Reservations)), nil)
	}

	//VM의 서브넷 정보 추출
	subnetList := []*string{}
	for _, i := range result.Reservations {
		for _, vm := range i.Instances {
			cblogger.Debugf("[%s] EC2 Subnet : [%s]", *vm.InstanceId, *vm.SubnetId)
			subnetList = append(subnetList, vm.SubnetId)
		}
	}

	/*
		VmHandler := AwsVMHandler{Client: NLBHandler.VMClient}
		for _, cur := range *VMs {
			cblogger.Debugf("======>VM ID : [%s]", cur.SystemId)
			vmInfo, errGetVM := VmHandler.GetVM(cur)
			if errGetVM != nil {
				return nil, errGetVM
			}
			cblogger.Debugf("=====> 서브넷 정보 : [%s]", vmInfo.SubnetIID.SystemId)
			subnetList = append(subnetList, aws.String(vmInfo.SubnetIID.SystemId))
		}
	*/

	return subnetList, nil
}

//&elbv2.CreateTargetGroupInput
func (NLBHandler *AwsNLBHandler) CheckCreateValidation(nlbReqInfo irs.NLBInfo) error {
	//&elbv2.CreateTargetGroupInput
	if nlbReqInfo.HealthChecker.Interval > 0 {
		//TCP, TLS, UDP, or TCP_UDP의 경우 Health check interval은 10이나 30만 가능함.
		// The approximate amount of time, in seconds, between health checks of an individual target.
		// If the target group protocol is TCP, TLS, UDP, or TCP_UDP, the supported values are 10 and 30 seconds.
		// If the target group protocol is HTTP or HTTPS, the default is 30 seconds.
		// If the target group protocol is GENEVE, the default is 10 seconds.
		// If the target type is lambda, the default is 35 seconds
		if strings.EqualFold(nlbReqInfo.HealthChecker.Protocol, "TCP") || strings.EqualFold(nlbReqInfo.HealthChecker.Protocol, "TLS") || strings.EqualFold(nlbReqInfo.HealthChecker.Protocol, "UDP") || strings.EqualFold(nlbReqInfo.HealthChecker.Protocol, "TCP_UDP") {
			//헬스 체크 인터벌 값 검증
			if nlbReqInfo.HealthChecker.Interval == 10 || nlbReqInfo.HealthChecker.Interval == 30 {
				cblogger.Debugf("===================> 헬스 체크 인터벌 값 검증 : 통과 : [%d]", nlbReqInfo.HealthChecker.Interval)
			} else {
				cblogger.Errorf("===================> 헬스 체크 인터벌 값 검증 : 실패 - 입력 값 : [%d]", nlbReqInfo.HealthChecker.Interval)
				cblogger.Error("TCP 프로토콜의 헬스 체크 인터벌은 10 또는 30만 가능 함.")
				return awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "The health check interval for TCP protocol can only be 10 or 30.", nil)
			}

		}
	}

	//타임 아웃 값 검증
	//InvalidConfigurationRequest: Custom health check timeouts are not supported for health checks for target groups with the TCP protocol
	// The amount of time, in seconds, during which no response from a target means a failed health check.
	// For target groups with a protocol of HTTP, HTTPS, or GENEVE, the default is 5 seconds.
	// For target groups with a protocol of TCP or TLS, this value must be 6 seconds for HTTP health checks and 10 seconds for TCP and HTTPS health checks.
	// If the target type is lambda, the default is 30 seconds.
	if nlbReqInfo.HealthChecker.Timeout > 0 {
		if strings.EqualFold(nlbReqInfo.HealthChecker.Protocol, "TCP") {
			cblogger.Errorf("===================> TCP 프로토콜은 헬스 체크 타임아웃 값 설정을 지원하지 않음")
			return awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "Custom health check timeouts are not supported for health checks for target groups with the TCP protocol.", nil)
		} else {
			cblogger.Debugf("===================> 헬스 체크 타임아웃 값 검증 : 통과 : [%d](TCP프로토콜 아님)", nlbReqInfo.HealthChecker.Timeout)
		}
	}

	return nil
}

//------ NLB Management
func (NLBHandler *AwsNLBHandler) CreateNLB(nlbReqInfo irs.NLBInfo) (irs.NLBInfo, error) {
	cblogger.Debug(nlbReqInfo)

	// 동일 네임 NLB가 이미 존재하는지 체크 해야 함. (현재 API는 에러가 발생하는 경우도 있지만 대부분 에러 없이 Skip됨)

	//최대한 삭제 로직을 태우지 않기 위해 NLB 생성에 문제가 없는지 사전에 검증한다.
	errValidation := NLBHandler.CheckCreateValidation(nlbReqInfo)
	if errValidation != nil {
		cblogger.Error(errValidation)
		return irs.NLBInfo{}, errValidation
	}

	//==================
	//서브넷 정보 추출
	//==================
	vmSubnets, errVmInfo := NLBHandler.ExtractVmSubnets(nlbReqInfo.VMGroup.VMs)
	if errVmInfo != nil {
		cblogger.Error(errVmInfo)
		return irs.NLBInfo{}, errVmInfo
	}

	input := &elbv2.CreateLoadBalancerInput{
		Name:    aws.String(nlbReqInfo.IId.NameId),
		Type:    aws.String("network"), //NLB 생성
		Subnets: vmSubnets,
		//Scheme: aws.String("internal"),	// private IP 이용
		//Scheme: aws.String("Internet-facing"),	//Default - 퍼블릭 서브넷 필요(public subnet)
		/*
			Subnets: []*string{
				//aws.String(vmInfo.SubnetIID.SystemId),
				//aws.String("subnet-0d30ee6b367974a39"), //New-CB-Subnet-NLB-1a1
				//aws.String("subnet-07a53d994a52abfe1"), //New-CB-Subnet-NLB-1c2
				//aws.String("subnet-0cf7417f83fd0fd47"), //New-CB-Subnet-NLB-1d1
			},
		*/
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: nlbReqInfo.IId.NameId,
		CloudOSAPI:   "CreateLoadBalancer()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := NLBHandler.Client.CreateLoadBalancer(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeDuplicateLoadBalancerNameException:
				cblogger.Error(elbv2.ErrCodeDuplicateLoadBalancerNameException, aerr.Error())
			case elbv2.ErrCodeTooManyLoadBalancersException:
				cblogger.Error(elbv2.ErrCodeTooManyLoadBalancersException, aerr.Error())
			case elbv2.ErrCodeInvalidConfigurationRequestException:
				cblogger.Error(elbv2.ErrCodeInvalidConfigurationRequestException, aerr.Error())
			case elbv2.ErrCodeSubnetNotFoundException:
				cblogger.Error(elbv2.ErrCodeSubnetNotFoundException, aerr.Error())
			case elbv2.ErrCodeInvalidSubnetException:
				cblogger.Error(elbv2.ErrCodeInvalidSubnetException, aerr.Error())
			case elbv2.ErrCodeInvalidSecurityGroupException:
				cblogger.Error(elbv2.ErrCodeInvalidSecurityGroupException, aerr.Error())
			case elbv2.ErrCodeInvalidSchemeException:
				cblogger.Error(elbv2.ErrCodeInvalidSchemeException, aerr.Error())
			case elbv2.ErrCodeTooManyTagsException:
				cblogger.Error(elbv2.ErrCodeTooManyTagsException, aerr.Error())
			case elbv2.ErrCodeDuplicateTagKeysException:
				cblogger.Error(elbv2.ErrCodeDuplicateTagKeysException, aerr.Error())
			case elbv2.ErrCodeResourceInUseException:
				cblogger.Error(elbv2.ErrCodeResourceInUseException, aerr.Error())
			case elbv2.ErrCodeAllocationIdNotFoundException:
				cblogger.Error(elbv2.ErrCodeAllocationIdNotFoundException, aerr.Error())
			case elbv2.ErrCodeAvailabilityZoneNotSupportedException:
				cblogger.Error(elbv2.ErrCodeAvailabilityZoneNotSupportedException, aerr.Error())
			case elbv2.ErrCodeOperationNotPermittedException:
				cblogger.Error(elbv2.ErrCodeOperationNotPermittedException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}

		return irs.NLBInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Infof("[%s] NLB 생성 완료 - LoadBalancerArn : [%s]", nlbReqInfo.IId.NameId, *result.LoadBalancers[0].LoadBalancerArn)
	cblogger.Debug(result)
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	nlbReqInfo.IId.SystemId = *result.LoadBalancers[0].LoadBalancerArn //리스너 생성및 장애시 삭제 처리를 위해 Req에 LoadBalancerArn 정보를 셋팅함.

	//================
	// 타겟그룹 생성
	//================
	targetGroup, errTargetGroup := NLBHandler.CreateTargetGroup(nlbReqInfo)
	if errTargetGroup != nil {
		cblogger.Error(errTargetGroup.Error())

		//생성된 NLB 포함 리소스들 삭제
		cblogger.Infof("VM 그룹 생성 실패에 따른 NLB[%s]및 관련 리소스 삭제 작업 시작!!", nlbReqInfo.IId.NameId)
		_, errNlbInfo := NLBHandler.DeleteNLB(nlbReqInfo.IId)
		if errNlbInfo != nil {
			cblogger.Errorf("VM 그룹 생성 실패에 따른 NLB[%s]및 관련 리소스 삭제 작업 실패!!", nlbReqInfo.IId.NameId)
			cblogger.Error(errNlbInfo.Error())
			//만약 NLB 포함 관련 리소스 정보 제거에 실패해도 생성 에러 메시지 유지를 위해 다른 작업은 진행하지 않음.
		} else {
			cblogger.Infof("VM 그룹 생성 실패에 따른 NLB[%s]및 관련 리소스 삭제 작업 완료!!", nlbReqInfo.IId.NameId)
		}

		return irs.NLBInfo{}, errTargetGroup
	}

	if cblogger.Level.String() == "debug" {
		spew.Dump(targetGroup)
	}

	//===================
	// 타겟그룹에 VM 추가
	//===================
	_, errAddVms := NLBHandler.AddVMs(nlbReqInfo.IId, nlbReqInfo.VMGroup.VMs)
	if errAddVms != nil {
		cblogger.Error(errAddVms.Error())

		//생성된 NLB 포함 리소스들 삭제
		cblogger.Infof("생성된 VM 그룹에 인스턴스 추가 실패에 따른 NLB[%s]및 관련 리소스 삭제 작업 시작!!", nlbReqInfo.IId.NameId)
		_, errNlbInfo := NLBHandler.DeleteNLB(nlbReqInfo.IId)
		if errNlbInfo != nil {
			cblogger.Errorf("생성된 VM 그룹에 인스턴스 추가 실패에 따른 NLB[%s]및 관련 리소스 삭제 작업 실패!!", nlbReqInfo.IId.NameId)
			cblogger.Error(errNlbInfo.Error())
			//만약 NLB 포함 관련 리소스 정보 제거에 실패해도 생성 에러 메시지 유지를 위해 다른 작업은 진행하지 않음.
		} else {
			cblogger.Infof("생성된 VM 그룹에 인스턴스 추가 실패에 따른 NLB[%s]및 관련 리소스 삭제 작업 완료!!", nlbReqInfo.IId.NameId)
		}

		return irs.NLBInfo{}, errAddVms
	}

	//================
	// 리스너 생성
	//================
	nlbReqInfo.VMGroup.CspID = *targetGroup.TargetGroups[0].TargetGroupArn //리스너 생성을 위해 Req에 TargetGroupArn 정보를 셋팅함.
	listener, errListener := NLBHandler.CreateListener(nlbReqInfo)
	if errListener != nil {
		cblogger.Error(errListener.Error())

		//생성된 NLB 포함 리소스들 삭제
		cblogger.Infof("리스너 생성 실패에 따른 NLB[%s]및 관련 리소스 삭제 작업 시작!!", nlbReqInfo.IId.NameId)
		_, errNlbInfo := NLBHandler.DeleteNLB(nlbReqInfo.IId)
		if errNlbInfo != nil {
			cblogger.Errorf("리스너 생성 실패에 따른 NLB[%s]및 관련 리소스 삭제 작업 실패!!", nlbReqInfo.IId.NameId)
			cblogger.Error(errNlbInfo.Error())
			//만약 NLB 포함 관련 리소스 정보 제거에 실패해도 생성 에러 메시지 유지를 위해 다른 작업은 진행하지 않음.
		} else {
			cblogger.Infof("리스너 생성 실패에 따른 NLB[%s]및 관련 리소스 삭제 작업 완료!!", nlbReqInfo.IId.NameId)
		}

		return irs.NLBInfo{}, errListener
	}

	if cblogger.Level.String() == "debug" {
		spew.Dump(listener)
	}

	//================================
	// 가장 최신 정보로 정보를 갱신 함.
	//================================
	nlbInfo, errNLBInfo := NLBHandler.GetNLB(nlbReqInfo.IId)
	if errNLBInfo != nil {
		cblogger.Error(errNLBInfo.Error())
		return irs.NLBInfo{}, errNLBInfo
	}

	//Name이 필수라서 GetNLB에서 NameId 값을 채워서 리턴하기 때문에 강제로 설정할 필요 없음.
	//nlbInfo.IId.NameId = nlbReqInfo.IId.NameId // cb-spider를 위해 NameId 설정

	return nlbInfo, nil
}

func (NLBHandler *AwsNLBHandler) ListNLB() ([]*irs.NLBInfo, error) {
	input := &elbv2.DescribeLoadBalancersInput{
		//LoadBalancerArns: []*string{
		//	aws.String("arn:aws:elasticloadbalancing:us-west-2:123456789012:loadbalancer/app/my-load-balancer/50dc6c495c0c9188"),
		//},
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "LIST()",
		CloudOSAPI:   "DescribeLoadBalancers()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := NLBHandler.Client.DescribeLoadBalancers(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	cblogger.Debug(result)
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeLoadBalancerNotFoundException:
				cblogger.Error(elbv2.ErrCodeLoadBalancerNotFoundException, aerr.Error())
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

	var results []*irs.NLBInfo
	for _, curNLB := range result.LoadBalancers {
		nlbInfo, errNLBInfo := NLBHandler.GetNLB(irs.IID{SystemId: *curNLB.LoadBalancerArn})
		if errNLBInfo != nil {
			cblogger.Error(errNLBInfo.Error())
			return nil, err
		}

		results = append(results, &nlbInfo)
	}

	return results, nil
}

func (NLBHandler *AwsNLBHandler) GetNLB(nlbIID irs.IID) (irs.NLBInfo, error) {
	cblogger.Info("NLB IID : ", nlbIID.SystemId)
	if nlbIID.SystemId == "" {
		cblogger.Error("IID 값이 Null임.")
		return irs.NLBInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "nlbIID.systemId value of the input parameter is empty.", nil)
	}

	input := &elbv2.DescribeLoadBalancersInput{
		LoadBalancerArns: []*string{
			aws.String(nlbIID.SystemId),
		},
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: nlbIID.SystemId,
		CloudOSAPI:   "DescribeLoadBalancers()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := NLBHandler.Client.DescribeLoadBalancers(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	cblogger.Debug(result)
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeLoadBalancerNotFoundException:
				cblogger.Error(elbv2.ErrCodeLoadBalancerNotFoundException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return irs.NLBInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	if len(result.LoadBalancers) > 0 {
		nlbInfo, errInfo := NLBHandler.ExtractNLBInfo(result.LoadBalancers[0])
		if errInfo != nil {
			return irs.NLBInfo{}, errInfo
		}
		return nlbInfo, nil
	} else {
		return irs.NLBInfo{}, errors.New("InvalidNLBArn.NotFound: The NLB Arn '" + nlbIID.SystemId + "' does not exist")
	}
}

func (NLBHandler *AwsNLBHandler) ExtractListenerInfo(nlbIID irs.IID) (irs.ListenerInfo, error) {
	//리스너는 NLB와 연결되어야만 생성 가능하기에 Arn으로 조회 함.
	inputListener := &elbv2.DescribeListenersInput{
		LoadBalancerArn: aws.String(nlbIID.SystemId),
	}

	resListener, err := NLBHandler.Client.DescribeListeners(inputListener)
	if err != nil {
		cblogger.Error(err)
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeListenerNotFoundException:
				cblogger.Error(elbv2.ErrCodeListenerNotFoundException, aerr.Error())
				//Test 결과 NLB에 리스너가 할당되지 않아도 지금은 ErrCodeListenerNotFoundException 예외는 발생하지 않고 정상 결과로 처리되지만
				//만약을 위해서 TargetGroup처럼 NotFound의 경우 정상 처리 함.
				cblogger.Info("조회 및 삭제 로직을 위해 리스너 Not Found는 에러로 처리하지 않음.")
				return irs.ListenerInfo{}, nil
			case elbv2.ErrCodeLoadBalancerNotFoundException:
				cblogger.Error(elbv2.ErrCodeLoadBalancerNotFoundException, aerr.Error())
			case elbv2.ErrCodeUnsupportedProtocolException:
				cblogger.Error(elbv2.ErrCodeUnsupportedProtocolException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return irs.ListenerInfo{}, err
	}

	cblogger.Debug(resListener)
	if cblogger.Level.String() == "debug" {
		spew.Dump(resListener)
	}

	if len(resListener.Listeners) > 0 {
		retListenerInfo := irs.ListenerInfo{
			CspID:    *resListener.Listeners[0].ListenerArn,
			Protocol: *resListener.Listeners[0].Protocol, // TCP|UDP
			//IP       string // Auto Generated and attached
			//DNSName  string // Optional, Auto Generated and attached
		}
		retListenerInfo.Port = strconv.FormatInt(*resListener.Listeners[0].Port, 10)

		//Key Value 처리
		keyValueList, _ := ConvertKeyValueList(resListener.Listeners[0])
		retListenerInfo.KeyValueList = keyValueList

		return retListenerInfo, nil
	} else {
		return irs.ListenerInfo{}, nil
	}
}

func (NLBHandler *AwsNLBHandler) ExtractVMGroupInfo(nlbIID irs.IID) (TargetGroupInfo, error) {
	targetGroupInfo := TargetGroupInfo{}
	input := &elbv2.DescribeTargetGroupsInput{
		//LoadBalancerArn: aws.String(nlbIID.SystemId),
		Names: []*string{aws.String(nlbIID.NameId)}, //TargetGroup과 연결이 끊기는 경우가 있어서 NLB 이름으로 검색함.
	}

	result, err := NLBHandler.Client.DescribeTargetGroups(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeLoadBalancerNotFoundException:
				cblogger.Error(elbv2.ErrCodeLoadBalancerNotFoundException, aerr.Error())
			case elbv2.ErrCodeTargetGroupNotFoundException:
				cblogger.Error(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())

				//TargetGroup 정보가 없는 경우 생성 도중 실패나 AWS 콘솔 등에서 삭제된 경우를 감안해서 List및 삭제 작업 시 발생할 에러를 방지하기 위해 아무런 처리도 하지 않음.
				cblogger.Info("조회 및 삭제 로직을 위해 타겟그룹 Not Found는 에러로 처리하지 않음.")
				return TargetGroupInfo{}, nil
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return TargetGroupInfo{}, err
	}
	cblogger.Debug(result)
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	if len(result.TargetGroups) > 0 {
		retVMGroupInfo := irs.VMGroupInfo{
			CspID:    *result.TargetGroups[0].TargetGroupArn,
			Protocol: *result.TargetGroups[0].Protocol, // TCP|UDP
			//VMs[]
		}
		retVMGroupInfo.Port = strconv.FormatInt(*result.TargetGroups[0].Port, 10)
		targetGroupInfo.VMGroup = retVMGroupInfo

		//=================================
		//HealthCheckerInfo 정보도 함께 처리
		//=================================
		targetGroupInfo.HealthChecker = irs.HealthCheckerInfo{
			CspID:     *result.TargetGroups[0].TargetGroupArn,
			Protocol:  *result.TargetGroups[0].Protocol,
			Interval:  int(*result.TargetGroups[0].HealthCheckIntervalSeconds),
			Timeout:   int(*result.TargetGroups[0].HealthCheckTimeoutSeconds),
			Threshold: int(*result.TargetGroups[0].HealthyThresholdCount),
		}
		targetGroupInfo.HealthChecker.Port = strconv.FormatInt(*result.TargetGroups[0].Port, 10)

		//================
		//Key Value 처리
		//================
		keyValueList, _ := ConvertKeyValueList(result.TargetGroups[0])
		targetGroupInfo.VMGroup.KeyValueList = keyValueList

		//================
		//VM 정보 처리
		//================
		targetHealthInfo, errHealthInfo := NLBHandler.ExtractVMGroupHealthInfo(*result.TargetGroups[0].TargetGroupArn)
		if err != nil {
			return TargetGroupInfo{}, errHealthInfo
		}
		targetGroupInfo.VMGroup.VMs = targetHealthInfo.AllVMs

		return targetGroupInfo, nil
	} else {
		return TargetGroupInfo{}, nil
	}
}

func (NLBHandler *AwsNLBHandler) ExtractHealthCheckerInfo(targetGroupArn string) (irs.HealthCheckerInfo, error) {
	input := &elbv2.DescribeTargetHealthInput{
		//TargetGroupArn : aws.String(nlbIID.SystemId),
		TargetGroupArn: aws.String(targetGroupArn),
	}

	result, err := NLBHandler.Client.DescribeTargetHealth(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeInvalidTargetException:
				cblogger.Error(elbv2.ErrCodeInvalidTargetException, aerr.Error())
			case elbv2.ErrCodeTargetGroupNotFoundException:
				cblogger.Error(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
			case elbv2.ErrCodeHealthUnavailableException:
				cblogger.Error(elbv2.ErrCodeHealthUnavailableException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return irs.HealthCheckerInfo{}, err
	}

	cblogger.Debug(result)
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	if len(result.TargetHealthDescriptions) > 0 {
		retHealthCheckerInfo := irs.HealthCheckerInfo{
			//Protocol: *result.TargetHealthDescriptions[0].Target.
			Port: *result.TargetHealthDescriptions[0].HealthCheckPort,
			//Interval: 정보 없음
			//Timeout: 정보 없음
			//Threshold: 정보 없음
			//VMs[]: 정보 없음
		}
		//retHealthCheckerInfo.Port = strconv.FormatInt(*result.TargetHealthDescriptions[0].Target.Port, 10)
		return retHealthCheckerInfo, nil
	} else {
		return irs.HealthCheckerInfo{}, nil
	}
}

func (NLBHandler *AwsNLBHandler) ExtractNLBInfo(nlbReqInfo *elbv2.LoadBalancer) (irs.NLBInfo, error) {
	retNLBInfo := irs.NLBInfo{
		IId:         irs.IID{NameId: *nlbReqInfo.LoadBalancerName, SystemId: *nlbReqInfo.LoadBalancerArn},
		VpcIID:      irs.IID{SystemId: *nlbReqInfo.VpcId},
		Type:        "PUBLIC",
		Scope:       "REGION",
		CreatedTime: *nlbReqInfo.CreatedTime,
	}

	/*
		//AZ 정보 등 누락되는 정보가 많아서 KeyValueList는 일일이 직접 대입 대신에 ConvertKeyValueList() 유틸 함수를 사용함.
		keyValueList := []irs.KeyValue{
			//{Key: "LoadBalancerArn", Value: *nlbReqInfo.LoadBalancerArn},
		}
		if !reflect.ValueOf(nlbReqInfo.State).IsNil() {
			keyValueList = append(keyValueList, irs.KeyValue{Key: "State", Value: *nlbReqInfo.State.Code}) //Code: "provisioning"
		}

		if !reflect.ValueOf(nlbReqInfo.LoadBalancerArn).IsNil() {
			keyValueList = append(keyValueList, irs.KeyValue{Key: "LoadBalancerArn", Value: *nlbReqInfo.LoadBalancerArn})
		}
	*/

	keyValueList, _ := ConvertKeyValueList(nlbReqInfo)
	retNLBInfo.KeyValueList = keyValueList

	//==================
	// VM Group 처리
	//==================
	cblogger.Info("VM Group 정보 조회 시작")
	retTargetGroupInfo, errVMGroupInfo := NLBHandler.ExtractVMGroupInfo(retNLBInfo.IId) //NLB Name으로 검색함.
	//NLB에 연결되지 않았거나 아직 생성되지 않은 TargetGroup을 감안해서 404 Notfound는 에러처리 하지 않음
	if errVMGroupInfo != nil {
		cblogger.Error(errVMGroupInfo.Error())
		return irs.NLBInfo{}, errVMGroupInfo
	}
	retNLBInfo.VMGroup = retTargetGroupInfo.VMGroup
	retNLBInfo.HealthChecker = retTargetGroupInfo.HealthChecker

	//==================
	// 리스너 처리
	//==================
	cblogger.Info("Listener 정보 조회 시작")
	retListenerInfo, errListener := NLBHandler.ExtractListenerInfo(retNLBInfo.IId) //NLB Arn으로 검색 함.
	if errListener != nil {
		cblogger.Error(errListener.Error())
		return irs.NLBInfo{}, errListener
	}
	retListenerInfo.DNSName = *nlbReqInfo.DNSName
	retNLBInfo.Listener = retListenerInfo

	/*
		//==================
		// HealthChecker 처리
		//==================
		retHealthCheckerInfo, errHealthCheckerInfo := NLBHandler.ExtractHealthCheckerInfo(retVMGroupInfo.CspID)
		if errHealthCheckerInfo != nil {
			cblogger.Error(errHealthCheckerInfo.Error())
			return irs.NLBInfo{}, errHealthCheckerInfo
		}
		retNLBInfo.HealthChecker = retHealthCheckerInfo
	*/

	return retNLBInfo, nil
}

func (NLBHandler *AwsNLBHandler) DeleteListener(listenerArn *string) (bool, error) {
	input := &elbv2.DeleteListenerInput{
		ListenerArn: listenerArn,
	}

	result, err := NLBHandler.Client.DeleteListener(input)
	if err != nil {
		cblogger.Errorf("Listener[%s] 삭제 실패", listenerArn)
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeListenerNotFoundException:
				cblogger.Error(elbv2.ErrCodeListenerNotFoundException, aerr.Error())
			case elbv2.ErrCodeResourceInUseException:
				cblogger.Error(elbv2.ErrCodeResourceInUseException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return false, err
	}

	cblogger.Infof("Listener[%s] 삭제 완료", listenerArn)
	cblogger.Debug(result)
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	return true, nil
}

func (NLBHandler *AwsNLBHandler) DeleteTargetGroup(targetGroupArn *string) (bool, error) {
	input := &elbv2.DeleteTargetGroupInput{
		TargetGroupArn: targetGroupArn,
	}

	result, err := NLBHandler.Client.DeleteTargetGroup(input)
	if err != nil {
		cblogger.Errorf("TargetGroup[%s] 삭제 실패", targetGroupArn)
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeResourceInUseException:
				cblogger.Error(elbv2.ErrCodeResourceInUseException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return false, err
	}

	cblogger.Infof("TargetGroup[%s] 삭제 완료", targetGroupArn)
	cblogger.Debug(result)
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	return true, nil
}

// @TODO : 상황봐서 TargetGroup과 Listener 삭제 시 발생하는 오류는 무시하고 NLB 삭제까지 진행할 것
func (NLBHandler *AwsNLBHandler) DeleteNLB(nlbIID irs.IID) (bool, error) {
	//타겟 그룹과 리스너가 존재할 경우 함께 삭제하기 위해 삭제할 NLB의 정보를 조회 함.
	nlbInfo, errNlbInfo := NLBHandler.GetNLB(nlbIID)
	if errNlbInfo != nil {
		cblogger.Error(errNlbInfo.Error())
		return false, errNlbInfo
	}

	cblogger.Info("삭제할 NLB 정보")
	cblogger.Info(nlbInfo)
	if cblogger.Level.String() == "debug" {
		spew.Dump(nlbInfo)
	}

	//=========================
	// Listener 삭제
	//=========================
	// Listener 정보가 있는 경우에만 진행
	//nlbInfo.Listener = irs.ListenerInfo{}
	if nlbInfo.Listener.CspID != "" {
		cblogger.Infof("[%s] Listener 삭제 시작", nlbInfo.Listener.CspID)

		_, errDeleteListener := NLBHandler.DeleteListener(&nlbInfo.Listener.CspID)
		if errDeleteListener != nil {
			cblogger.Error(errDeleteListener.Error())
			return false, errDeleteListener
		}
	}

	//=========================
	// TargetGroup 삭제
	//=========================
	// TargetGroup 정보가 있는 경우에만 진행
	// TargetGroup 삭제 전에 연결된 Listener부터 삭제 해야 함.
	// 	ResourceInUse: Target group 'arn:aws:elasticloadbalancing:ap-northeast-1:050864702683:targetgroup/cb-nlb-test01/013fca42c7472109' is currently in use by a listener or a rule
	//if !reflect.ValueOf(nlbInfo.VMGroup).IsNil() && nlbInfo.VMGroup.CspID != "" {
	//nlbInfo.VMGroup = irs.VMGroupInfo{}
	if nlbInfo.VMGroup.CspID != "" {
		cblogger.Infof("[%s] TargetGroup 삭제 시작", nlbInfo.VMGroup.CspID)
		_, errDeleteTargetGroup := NLBHandler.DeleteTargetGroup(&nlbInfo.VMGroup.CspID)
		if errDeleteTargetGroup != nil {
			cblogger.Error(errDeleteTargetGroup.Error())
			return false, errDeleteTargetGroup
		}
	}

	//=========================
	// NLB 삭제
	//=========================
	input := &elbv2.DeleteLoadBalancerInput{
		LoadBalancerArn: aws.String(nlbInfo.IId.SystemId),
	}

	result, err := NLBHandler.Client.DeleteLoadBalancer(input)
	if err != nil {
		cblogger.Errorf("NLB[%s] 삭제 실패", nlbIID.SystemId)
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeLoadBalancerNotFoundException:
				cblogger.Error(elbv2.ErrCodeLoadBalancerNotFoundException, aerr.Error())
			case elbv2.ErrCodeOperationNotPermittedException:
				cblogger.Error(elbv2.ErrCodeOperationNotPermittedException, aerr.Error())
			case elbv2.ErrCodeResourceInUseException:
				cblogger.Error(elbv2.ErrCodeResourceInUseException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return false, err
	}
	cblogger.Infof("NLB[%s] 삭제 완료", nlbIID.SystemId)
	cblogger.Debug(result)
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	return true, nil
}

//------ Frontend Control
// Protocol 하고 Port 정보만 변경 가능
func (NLBHandler *AwsNLBHandler) ChangeListener(nlbIID irs.IID, listener irs.ListenerInfo) (irs.ListenerInfo, error) {
	if nlbIID.SystemId == "" {
		cblogger.Error("IID 값이 Null임.")
		return irs.ListenerInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "nlbIID.systemId value of the input parameter is empty.", nil)
	}

	//리스너의 ARN 값을 조회함.
	listenerInfo, errListener := NLBHandler.ExtractListenerInfo(nlbIID)
	if errListener != nil {
		cblogger.Error(errListener.Error())
		return irs.ListenerInfo{}, errListener
	}

	if listenerInfo.CspID == "" {
		cblogger.Error("NLB와 연결된 리스너의 ARN 값을 찾을 수 없음")
		return irs.ListenerInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "Listener associated with NLB does not exist.", nil)
	}

	input := &elbv2.ModifyListenerInput{
		/*
			DefaultActions: []*elbv2.Action{
				{
					TargetGroupArn: aws.String(""),	//cb-spider는 타겟그룹 변경 기능이 없음.
					Type:           aws.String("forward"),
				},
			},
		*/
		ListenerArn: aws.String(listenerInfo.CspID),
		//Protocol: aws.String(nlbReqInfo.Listener.Protocol), // AWS NLB : TCP, TLS, UDP, or TCP_UDP
	}

	//리스너 프로토콜 변경
	if listener.Protocol != "" {
		input.Protocol = aws.String(listener.Protocol)
	}

	//리스너 포트 변경
	if listener.Port != "" {
		if n, err := strconv.ParseInt(listener.Port, 10, 64); err == nil {
			input.SetPort(n)
		} else {
			cblogger.Error(listener.Port, "은 숫자가 아님!!")
			return irs.ListenerInfo{}, err
		}
	}

	cblogger.Info("리스너 정보 변경 시작")
	cblogger.Info(input)

	result, err := NLBHandler.Client.ModifyListener(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeDuplicateListenerException:
				cblogger.Error(elbv2.ErrCodeDuplicateListenerException, aerr.Error())
			case elbv2.ErrCodeTooManyListenersException:
				cblogger.Error(elbv2.ErrCodeTooManyListenersException, aerr.Error())
			case elbv2.ErrCodeTooManyCertificatesException:
				cblogger.Error(elbv2.ErrCodeTooManyCertificatesException, aerr.Error())
			case elbv2.ErrCodeListenerNotFoundException:
				cblogger.Error(elbv2.ErrCodeListenerNotFoundException, aerr.Error())
			case elbv2.ErrCodeTargetGroupNotFoundException:
				cblogger.Error(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
			case elbv2.ErrCodeTargetGroupAssociationLimitException:
				cblogger.Error(elbv2.ErrCodeTargetGroupAssociationLimitException, aerr.Error())
			case elbv2.ErrCodeIncompatibleProtocolsException:
				cblogger.Error(elbv2.ErrCodeIncompatibleProtocolsException, aerr.Error())
			case elbv2.ErrCodeSSLPolicyNotFoundException:
				cblogger.Error(elbv2.ErrCodeSSLPolicyNotFoundException, aerr.Error())
			case elbv2.ErrCodeCertificateNotFoundException:
				cblogger.Error(elbv2.ErrCodeCertificateNotFoundException, aerr.Error())
			case elbv2.ErrCodeInvalidConfigurationRequestException:
				cblogger.Error(elbv2.ErrCodeInvalidConfigurationRequestException, aerr.Error())
			case elbv2.ErrCodeUnsupportedProtocolException:
				cblogger.Error(elbv2.ErrCodeUnsupportedProtocolException, aerr.Error())
			case elbv2.ErrCodeTooManyRegistrationsForTargetIdException:
				cblogger.Error(elbv2.ErrCodeTooManyRegistrationsForTargetIdException, aerr.Error())
			case elbv2.ErrCodeTooManyTargetsException:
				cblogger.Error(elbv2.ErrCodeTooManyTargetsException, aerr.Error())
			case elbv2.ErrCodeTooManyActionsException:
				cblogger.Error(elbv2.ErrCodeTooManyActionsException, aerr.Error())
			case elbv2.ErrCodeInvalidLoadBalancerActionException:
				cblogger.Error(elbv2.ErrCodeInvalidLoadBalancerActionException, aerr.Error())
			case elbv2.ErrCodeTooManyUniqueTargetGroupsPerLoadBalancerException:
				cblogger.Error(elbv2.ErrCodeTooManyUniqueTargetGroupsPerLoadBalancerException, aerr.Error())
			case elbv2.ErrCodeALPNPolicyNotSupportedException:
				cblogger.Error(elbv2.ErrCodeALPNPolicyNotSupportedException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}

		return irs.ListenerInfo{}, err
	}

	cblogger.Infof("리스너 정보 변경 완료")
	cblogger.Debug(result)
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	//변경된 최종 리스너 정보를 리턴 함.
	/*
		listenerInfo, errListener = NLBHandler.ExtractListenerInfo(nlbIID)
		if errListener != nil {
			cblogger.Error(errListener.Error())
			return irs.ListenerInfo{}, errListener
		}
		return listenerInfo, nil
	*/

	//ListenerInfo의 DNS 등의 정보 때문에 NLB 정보 조회후 리턴 함.
	nlbInfo, errNLBInfo := NLBHandler.GetNLB(nlbIID)
	if errNLBInfo != nil {
		cblogger.Error(errNLBInfo.Error())
		return irs.ListenerInfo{}, errNLBInfo
	}
	return nlbInfo.Listener, nil

}

//------ Backend Control
func (NLBHandler *AwsNLBHandler) ChangeVMGroupInfo(nlbIID irs.IID, vmGroup irs.VMGroupInfo) (irs.VMGroupInfo, error) {
	if nlbIID.SystemId == "" {
		cblogger.Error("IID 값이 Null임.")
		return irs.VMGroupInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "nlbIID.systemId value of the input parameter is empty.", nil)
	}

	return irs.VMGroupInfo{}, nil
}

// @TODO : VM 추가 시 NLB에 등록되지 않은 서브넷의 경우 추가 및 검증 로직 필요
// @TODO : 이미 등록된 AZ의 다른 서브넷을 사용하는 Instance 처리 필요
func (NLBHandler *AwsNLBHandler) AddVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (irs.VMGroupInfo, error) {
	if nlbIID.NameId == "" || nlbIID.SystemId == "" {
		cblogger.Error("IID 값이 Null임.")
		return irs.VMGroupInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "nlbIID value of the input parameter is empty.", nil)
	}

	//TargetGroup ARN및 사용할 Port 정보를 조회하기 위해 VM그룹 정보를 조회 함.
	retTargetGroupInfo, errVMGroupInfo := NLBHandler.ExtractVMGroupInfo(nlbIID)
	if errVMGroupInfo != nil {
		cblogger.Error(errVMGroupInfo.Error())
		return irs.VMGroupInfo{}, errVMGroupInfo
	}

	//NLB과 관련된 Target 그룹이 존재하지 않을 경우
	if retTargetGroupInfo.VMGroup.Port == "" {
		cblogger.Errorf("[%s] NLB와 연결된 VM Group이 존재하지 않아서 요청된 Instance를 추가할 수 없음", nlbIID.NameId)
		return irs.VMGroupInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "VM Group does not exist to add the instance.", nil)
	}

	// Port 정보 추출
	targetPort, _ := strconv.ParseInt(retTargetGroupInfo.VMGroup.Port, 10, 64)
	iTagetPort := aws.Int64(targetPort)

	input := &elbv2.RegisterTargetsInput{
		TargetGroupArn: aws.String(retTargetGroupInfo.VMGroup.CspID),
		/*
			Targets: []*elbv2.TargetDescription{
				{
					//Id: aws.String("i-008778f60fd7ae3fa"),
					//Port: aws.Int64(1234),
					//Port: iTagetPort,
				},
			},
		*/
	}

	// 추가할 VM 인스턴스 처리
	//targetList := []elbv2.TargetDescription{}
	for _, curVM := range *vmIIDs {
		//targetList = append(targetList, elbv2.TargetDescription{Id: aws.String(curVM.SystemId), Port: iTagetPort})
		input.Targets = append(input.Targets, &elbv2.TargetDescription{Id: aws.String(curVM.SystemId), Port: iTagetPort})
	}

	cblogger.Infof("VM 그룹(%s)에 추가 예정 인스턴스 정보들", retTargetGroupInfo.VMGroup.CspID)
	cblogger.Info(input)

	if cblogger.Level.String() == "debug" {
		spew.Dump(input)
	}

	//input.Targets = &targetList
	result, err := NLBHandler.Client.RegisterTargets(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeTargetGroupNotFoundException:
				cblogger.Error(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
			case elbv2.ErrCodeTooManyTargetsException:
				cblogger.Error(elbv2.ErrCodeTooManyTargetsException, aerr.Error())
			case elbv2.ErrCodeInvalidTargetException:
				cblogger.Error(elbv2.ErrCodeInvalidTargetException, aerr.Error())
			case elbv2.ErrCodeTooManyRegistrationsForTargetIdException:
				cblogger.Error(elbv2.ErrCodeTooManyRegistrationsForTargetIdException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return irs.VMGroupInfo{}, err
	}

	cblogger.Infof("VM 그룹(%s)에 인스턴스 추가 완료", retTargetGroupInfo.VMGroup.CspID)
	cblogger.Debug(result)
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	//최신 정보 전달을 위해 다시 호출함.
	retTargetGroupInfo, errVMGroupInfo = NLBHandler.ExtractVMGroupInfo(nlbIID)
	if errVMGroupInfo != nil {
		cblogger.Error(errVMGroupInfo.Error())
		return irs.VMGroupInfo{}, errVMGroupInfo
	}

	return retTargetGroupInfo.VMGroup, nil
}

func (NLBHandler *AwsNLBHandler) RemoveVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {
	if nlbIID.NameId == "" || nlbIID.SystemId == "" {
		cblogger.Error("IID 값이 Null임.")
		return false, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "nlbIID value of the input parameter is empty.", nil)
	}

	//TargetGroup ARN을 조회하기 위해 VM그룹 정보를 조회 함.
	retTargetGroupInfo, errVMGroupInfo := NLBHandler.ExtractVMGroupInfo(nlbIID)
	if errVMGroupInfo != nil {
		cblogger.Error(errVMGroupInfo.Error())
		return false, errVMGroupInfo
	}

	//NLB과 관련된 Target 그룹이 존재하지 않을 경우
	if retTargetGroupInfo.VMGroup.Port == "" {
		cblogger.Errorf("[%s] NLB와 연결된 VM Group이 존재하지 않아서 요청된 Instance를 제거할 수 없음", nlbIID.NameId)
		return false, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "VM Group does not exist to remove the instance.", nil)
	}

	input := &elbv2.DeregisterTargetsInput{
		TargetGroupArn: aws.String(retTargetGroupInfo.VMGroup.CspID),
		/*
			Targets: []*elbv2.TargetDescription{
				{
					//Id: aws.String("i-008778f60fd7ae3fa"),
				},
			},
		*/
	}

	// 삭제할 VM 인스턴스 처리
	for _, curVM := range *vmIIDs {
		input.Targets = append(input.Targets, &elbv2.TargetDescription{Id: aws.String(curVM.SystemId)})
	}

	cblogger.Infof("VM 그룹(%s)에서 삭제 예정 인스턴스 정보들", retTargetGroupInfo.VMGroup.CspID)
	cblogger.Info(input)
	if cblogger.Level.String() == "debug" {
		spew.Dump(input)
	}

	result, err := NLBHandler.Client.DeregisterTargets(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeTargetGroupNotFoundException:
				cblogger.Error(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
			case elbv2.ErrCodeInvalidTargetException:
				cblogger.Error(elbv2.ErrCodeInvalidTargetException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and Message from an error.
			cblogger.Error(err.Error())
		}
		return false, err
	}

	cblogger.Infof("VM 그룹(%s)에서 인스턴스 삭제 성공", retTargetGroupInfo.VMGroup.CspID)
	cblogger.Debug(result)
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	return true, nil
}

//ExtractVMGroupInfo에서 GetVMGroupHealthInfo를 호출하는 형태로 사용되면 발생할 무한 루프 방지를 위해 별도의 함수로 분리 함.
//https://docs.aws.amazon.com/elasticloadbalancing/latest/APIReference/API_TargetHealth.html
func (NLBHandler *AwsNLBHandler) ExtractVMGroupHealthInfo(targetGroupArn string) (irs.HealthInfo, error) {
	input := &elbv2.DescribeTargetHealthInput{
		TargetGroupArn: aws.String(targetGroupArn),
	}

	result, err := NLBHandler.Client.DescribeTargetHealth(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeInvalidTargetException:
				cblogger.Error(elbv2.ErrCodeInvalidTargetException, aerr.Error())
			case elbv2.ErrCodeTargetGroupNotFoundException:
				cblogger.Error(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
			case elbv2.ErrCodeHealthUnavailableException:
				cblogger.Error(elbv2.ErrCodeHealthUnavailableException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return irs.HealthInfo{}, err
	}

	cblogger.Debug(result)
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	retHealthInfo := irs.HealthInfo{}
	allVMs := []irs.IID{}
	healthyVMs := []irs.IID{}
	unHealthyVMs := []irs.IID{}
	for _, cur := range result.TargetHealthDescriptions {
		allVMs = append(allVMs, irs.IID{SystemId: *cur.Target.Id})
		cblogger.Debug(cur.TargetHealth.State)

		//상태별 결과 처리
		//initial | healthy | unhealthy | unused | draining | unavailable
		if strings.EqualFold(*cur.TargetHealth.State, "healthy") {
			healthyVMs = append(healthyVMs, irs.IID{SystemId: *cur.Target.Id})
			/*
				} else if strings.EqualFold(*cur.TargetHealth.State, "unhealthy") {
					unHealthyVMs = append(unHealthyVMs, irs.IID{SystemId: *cur.Target.Id})
				} else if strings.EqualFold(*cur.TargetHealth.State, "initial") || strings.EqualFold(*cur.TargetHealth.State, "unused") || strings.EqualFold(*cur.TargetHealth.State, "draining") || strings.EqualFold(*cur.TargetHealth.State, "unavailable") {
					cblogger.Infof("[%s] Instance는 [%s] 상태라서 Skip함.", *cur.Target.Id, *cur.TargetHealth.State)
					continue
			*/
		} else { //다른 CSP에 맞추기 위해 healthy 외의 상태를 모두 unhealthy로 처리 함.
			//cblogger.Errorf("미정의 VM Health 상태[%s]", cur.TargetHealth.State)
			unHealthyVMs = append(unHealthyVMs, irs.IID{SystemId: *cur.Target.Id})
		}
	}

	retHealthInfo.AllVMs = &allVMs
	retHealthInfo.HealthyVMs = &healthyVMs
	retHealthInfo.UnHealthyVMs = &unHealthyVMs
	return retHealthInfo, nil
}

// @TODO : 5가지의 상태(Healthy / Unhealthy / Unused / Initial / Draining)가 존재 하기 때문에 리턴 객체에 담을 Unhealthy의 범위 확정이 필요 함.
func (NLBHandler *AwsNLBHandler) GetVMGroupHealthInfo(nlbIID irs.IID) (irs.HealthInfo, error) {
	if nlbIID.SystemId == "" {
		cblogger.Error("IID 값이 Null임.")
		return irs.HealthInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "nlbIID.systemId value of the input parameter is empty.", nil)
	}

	//TargetGroup ARN을 조회하기 위해 VM그룹 정보를 조회 함.
	retTargetGroupInfo, errVMGroupInfo := NLBHandler.ExtractVMGroupInfo(nlbIID)
	if errVMGroupInfo != nil {
		cblogger.Error(errVMGroupInfo.Error())
		return irs.HealthInfo{}, errVMGroupInfo
	}

	result, err := NLBHandler.ExtractVMGroupHealthInfo(retTargetGroupInfo.VMGroup.CspID)
	if err != nil {
		return irs.HealthInfo{}, err
	}

	// @TODO : 삭제할 것
	//==================
	//서브넷 정보 추출
	//==================
	_, errVmInfo := NLBHandler.ExtractVmSubnets(retTargetGroupInfo.VMGroup.VMs)
	if errVmInfo != nil {
		cblogger.Error(errVmInfo)
		return irs.HealthInfo{}, errVmInfo
	}

	return result, nil
}

func (NLBHandler *AwsNLBHandler) ChangeHealthCheckerInfo(nlbIID irs.IID, healthChecker irs.HealthCheckerInfo) (irs.HealthCheckerInfo, error) {
	if nlbIID.SystemId == "" {
		cblogger.Error("IID 값이 Null임.")
		return irs.HealthCheckerInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "nlbIID.systemId value of the input parameter is empty.", nil)
	}

	return irs.HealthCheckerInfo{}, nil
}
