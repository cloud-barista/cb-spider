package resources

//https://docs.aws.amazon.com/sdk-for-go/api/service/elb

import (
	"errors"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/davecgh/go-spew/spew"

	//"github.com/aws/aws-sdk-go/service/elb"

	"github.com/aws/aws-sdk-go/service/elbv2"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsNLBHandler struct {
	Region idrv.RegionInfo
	//Client *elb.ELB
	Client *elbv2.ELBV2 //elbV2
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
	if nlbReqInfo.HealthChecker.Interval > 0 {
		input.HealthCheckIntervalSeconds = aws.Int64(int64(nlbReqInfo.HealthChecker.Interval))
	}

	// 타임아웃 설정
	if nlbReqInfo.HealthChecker.Timeout > 0 {
		input.HealthCheckTimeoutSeconds = aws.Int64(int64(nlbReqInfo.HealthChecker.Timeout))
	}

	// Threshold 설정
	if nlbReqInfo.HealthChecker.Threshold > 0 {
		input.HealthyThresholdCount = aws.Int64(int64(nlbReqInfo.HealthChecker.Threshold))
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

//------ NLB Management
func (NLBHandler *AwsNLBHandler) CreateNLB(nlbReqInfo irs.NLBInfo) (irs.NLBInfo, error) {
	cblogger.Debug(nlbReqInfo)

	input := &elbv2.CreateLoadBalancerInput{
		Name: aws.String(nlbReqInfo.IId.NameId),
		//Scheme: aws.String("internal"),	// private IP 이용
		//Scheme: aws.String("Internet-facing"),	//Default - 퍼블릭 서브넷 필요(public subnet)
		Type: aws.String("network"),

		Subnets: []*string{
			aws.String("subnet-0d30ee6b367974a39"), //New-CB-Subnet-NLB-1a1
			aws.String("subnet-07a53d994a52abfe1"), //New-CB-Subnet-NLB-1c2
			aws.String("subnet-0cf7417f83fd0fd47"), //New-CB-Subnet-NLB-1d1
		},
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

	cblogger.Debug(result)
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

	cblogger.Debug("NLB 생성 결과")
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	nlbReqInfo.IId.SystemId = *result.LoadBalancers[0].LoadBalancerArn //리스너 생성을 위해 Req에 ARN 정보를 셋팅함.

	//================
	// 타겟그룹 생성
	//================
	targetGroup, errTargetGroup := NLBHandler.CreateTargetGroup(nlbReqInfo)
	if errTargetGroup != nil {
		cblogger.Error(errTargetGroup.Error())
		return irs.NLBInfo{}, err
	}

	if cblogger.Level.String() == "debug" {
		spew.Dump(targetGroup)
	}

	nlbReqInfo.VMGroup.CspID = *targetGroup.TargetGroups[0].TargetGroupArn //리스너 생성을 위해 Req에 ARN 정보를 셋팅함.

	//================
	// 리스너 생성
	//================
	listener, errListener := NLBHandler.CreateListener(nlbReqInfo)
	if errListener != nil {
		cblogger.Error(errListener.Error())
		return irs.NLBInfo{}, err
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
		return irs.NLBInfo{}, err
	}

	nlbInfo.IId.NameId = nlbReqInfo.IId.NameId
	//nlbInfo.VMGroup.CspID = *targetGroup.TargetGroups[0].TargetGroupArn,

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
	inputListener := &elbv2.DescribeListenersInput{
		LoadBalancerArn: aws.String(nlbIID.SystemId),
	}

	resListener, err := NLBHandler.Client.DescribeListeners(inputListener)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeListenerNotFoundException:
				cblogger.Error(elbv2.ErrCodeListenerNotFoundException, aerr.Error())
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
		LoadBalancerArn: aws.String(nlbIID.SystemId),
	}

	result, err := NLBHandler.Client.DescribeTargetGroups(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeLoadBalancerNotFoundException:
				cblogger.Error(elbv2.ErrCodeLoadBalancerNotFoundException, aerr.Error())
			case elbv2.ErrCodeTargetGroupNotFoundException:
				cblogger.Error(elbv2.ErrCodeTargetGroupNotFoundException, aerr.Error())
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
	// 리스너 처리
	//==================
	retListenerInfo, errListener := NLBHandler.ExtractListenerInfo(retNLBInfo.IId)
	if errListener != nil {
		cblogger.Error(errListener.Error())
		return irs.NLBInfo{}, errListener
	}
	retListenerInfo.DNSName = *nlbReqInfo.DNSName
	retNLBInfo.Listener = retListenerInfo

	//==================
	// VM Group 처리
	//==================
	retTargetGroupInfo, errVMGroupInfo := NLBHandler.ExtractVMGroupInfo(retNLBInfo.IId)
	if errVMGroupInfo != nil {
		cblogger.Error(errVMGroupInfo.Error())
		return irs.NLBInfo{}, errVMGroupInfo
	}
	retNLBInfo.VMGroup = retTargetGroupInfo.VMGroup
	retNLBInfo.HealthChecker = retTargetGroupInfo.HealthChecker

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

func (NLBHandler *AwsNLBHandler) DeleteNLB(nlbIID irs.IID) (bool, error) {
	return false, nil
}

//------ Frontend Control
func (NLBHandler *AwsNLBHandler) ChangeListener(nlbIID irs.IID, listener irs.ListenerInfo) (irs.ListenerInfo, error) {
	return irs.ListenerInfo{}, nil
}

//------ Backend Control
func (NLBHandler *AwsNLBHandler) ChangeVMGroupInfo(nlbIID irs.IID, vmGroup irs.VMGroupInfo) (irs.VMGroupInfo, error) {
	return irs.VMGroupInfo{}, nil
}

// @TODO : VM 추가 시 NLB에 등록되지 않은 서브넷의 경우 추가 및 검증 로직 필요
// @TODO : 이미 등록된 AZ의 다른 서브넷을 사용하는 Instance 처리 필요
func (NLBHandler *AwsNLBHandler) AddVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (irs.VMGroupInfo, error) {
	//TargetGroup ARN및 사용할 Port 정보를 조회하기 위해 VM그룹 정보를 조회 함.
	retTargetGroupInfo, errVMGroupInfo := NLBHandler.ExtractVMGroupInfo(nlbIID)
	if errVMGroupInfo != nil {
		cblogger.Error(errVMGroupInfo.Error())
		return irs.VMGroupInfo{}, errVMGroupInfo
	}

	// Port 정보 추출
	targetPort, _ := strconv.ParseInt(retTargetGroupInfo.VMGroup.Port, 10, 64)
	iTagetPort := aws.Int64(targetPort)

	input := &elbv2.RegisterTargetsInput{
		TargetGroupArn: aws.String(retTargetGroupInfo.VMGroup.CspID),

		Targets: []*elbv2.TargetDescription{
			{
				//Id: aws.String("i-008778f60fd7ae3fa"),
				//Port: aws.Int64(1234),
				//Port: iTagetPort,
			},
		},
	}

	// 추가할 VM 인스턴스 처리
	//targetList := []elbv2.TargetDescription{}
	for _, curVM := range *vmIIDs {
		//targetList = append(targetList, elbv2.TargetDescription{Id: aws.String(curVM.SystemId), Port: iTagetPort})
		input.Targets = append(input.Targets, &elbv2.TargetDescription{Id: aws.String(curVM.SystemId), Port: iTagetPort})
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
	//TargetGroup ARN을 조회하기 위해 VM그룹 정보를 조회 함.
	retTargetGroupInfo, errVMGroupInfo := NLBHandler.ExtractVMGroupInfo(nlbIID)
	if errVMGroupInfo != nil {
		cblogger.Error(errVMGroupInfo.Error())
		return false, errVMGroupInfo
	}

	input := &elbv2.DeregisterTargetsInput{
		TargetGroupArn: aws.String(retTargetGroupInfo.VMGroup.CspID),
		Targets: []*elbv2.TargetDescription{
			{
				//Id: aws.String("i-008778f60fd7ae3fa"),
			},
		},
	}

	// 삭제할 VM 인스턴스 처리
	for _, curVM := range *vmIIDs {
		input.Targets = append(input.Targets, &elbv2.TargetDescription{Id: aws.String(curVM.SystemId)})
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

	cblogger.Debug(result)
	if cblogger.Level.String() == "debug" {
		spew.Dump(result)
	}

	return false, nil
}

//ExtractVMGroupInfo에서 GetVMGroupHealthInfo를 호출하는 형태로 사용되면 발생할 무한 루프 방지를 위해 별도의 함수로 분리 함.
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

		if strings.EqualFold(*cur.TargetHealth.State, "healthy") {
			healthyVMs = append(healthyVMs, irs.IID{SystemId: *cur.Target.Id})
		} else if strings.EqualFold(*cur.TargetHealth.State, "unhealthy") {
			unHealthyVMs = append(unHealthyVMs, irs.IID{SystemId: *cur.Target.Id})
		} else if strings.EqualFold(*cur.TargetHealth.State, "initial") {
			cblogger.Infof("[%s] Instance는 initial 상태라서 Skip함.", *cur.Target.Id)
			continue
		} else {
			cblogger.Errorf("미정의 VM Health 상태[%s]", cur.TargetHealth.State)
		}
	}

	retHealthInfo.AllVMs = &allVMs
	retHealthInfo.HealthyVMs = &healthyVMs
	retHealthInfo.UnHealthyVMs = &unHealthyVMs
	return retHealthInfo, nil
}

func (NLBHandler *AwsNLBHandler) GetVMGroupHealthInfo(nlbIID irs.IID) (irs.HealthInfo, error) {
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

	return result, nil
}

func (NLBHandler *AwsNLBHandler) ChangeHealthCheckerInfo(nlbIID irs.IID, healthChecker irs.HealthCheckerInfo) (irs.HealthCheckerInfo, error) {
	return irs.HealthCheckerInfo{}, nil
}
