package resources

//https://docs.aws.amazon.com/sdk-for-go/api/service/elb

import (
	"errors"
	"fmt"
	"strconv"

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
	spew.Dump(result)

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
	spew.Dump(result)

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
	spew.Dump(result)
	nlbReqInfo.IId.SystemId = *result.LoadBalancers[0].LoadBalancerArn //리스너 생성을 위해 Req에 ARN 정보를 셋팅함.

	//타겟그룹 생성
	targetGroup, errTargetGroup := NLBHandler.CreateTargetGroup(nlbReqInfo)
	if errTargetGroup != nil {
		cblogger.Error(errTargetGroup.Error())
		return irs.NLBInfo{}, err
	}
	spew.Dump(targetGroup)
	nlbReqInfo.VMGroup.CspID = *targetGroup.TargetGroups[0].TargetGroupArn //리스너 생성을 위해 Req에 ARN 정보를 셋팅함.

	//타겟그룹 생성
	listener, errListener := NLBHandler.CreateListener(nlbReqInfo)
	if errListener != nil {
		cblogger.Error(errListener.Error())
		return irs.NLBInfo{}, err
	}
	spew.Dump(listener)

	//가장 최신 정보로 정보를 갱신 함.
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
	spew.Dump(result)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeLoadBalancerNotFoundException:
				fmt.Println(elbv2.ErrCodeLoadBalancerNotFoundException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
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
	spew.Dump(result)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case elbv2.ErrCodeLoadBalancerNotFoundException:
				fmt.Println(elbv2.ErrCodeLoadBalancerNotFoundException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
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

func (NLBHandler *AwsNLBHandler) ExtractNLBInfo(nlbReqInfo *elbv2.LoadBalancer) (irs.NLBInfo, error) {
	retNLBInfo := irs.NLBInfo{
		IId:    irs.IID{NameId: *nlbReqInfo.LoadBalancerName, SystemId: *nlbReqInfo.LoadBalancerArn},
		VpcIID: irs.IID{SystemId: *nlbReqInfo.VpcId},
		Type:   "PUBLIC",
		Scope:  "REGION",

		/*
				Listener: irs.ListenerInfo{
					Protocol: "TCP",
					//IP: "",
					Port: "1234",
				},

			VMGroup: irs.VMGroupInfo{
				Protocol: "TCP",  //TCP|UDP|HTTP|HTTPS
				Port:     "1234", //1-65535
				VMs:      &[]irs.IID{},
			},

			HealthChecker: irs.HealthCheckerInfo{
				Protocol:  "TCP",  // TCP|HTTP|HTTPS
				Port:      "1234", // Listener Port or 1-65535
				Interval:  0,      // secs, Interval time between health checks.
				Timeout:   0,      // secs, Waiting time to decide an unhealthy VM when no response.
				Threshold: 0,      // num, The number of continuous health checks to change the VM status
			},
		*/
	}
	return retNLBInfo, nil
}

func (NLBHandler *AwsNLBHandler) DeleteNLB(nlbIID irs.IID) (bool, error) {
	return false, nil
}

//------ Frontend Control
func (NLBHandler *AwsNLBHandler) ChangeListener(nlbIID irs.IID, listener irs.ListenerInfo) (irs.NLBInfo, error) {
	return irs.NLBInfo{}, nil
}

//------ Backend Control
func (NLBHandler *AwsNLBHandler) ChangeVMGroupInfo(nlbIID irs.IID, vmGroup irs.VMGroupInfo) error {
	return nil
}

func (NLBHandler *AwsNLBHandler) AddVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (irs.NLBInfo, error) {
	return irs.NLBInfo{}, nil
}

func (NLBHandler *AwsNLBHandler) RemoveVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {
	return false, nil
}

func (NLBHandler *AwsNLBHandler) GetVMGroupHealthInfo(nlbIID irs.IID) (irs.HealthInfo, error) {
	return irs.HealthInfo{}, nil
}

func (NLBHandler *AwsNLBHandler) ChangeHealthCheckerInfo(nlbIID irs.IID, healthChecker irs.HealthCheckerInfo) error {
	return nil
}
