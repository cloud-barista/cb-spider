package resources

import (
	"fmt"
	"strconv"
	"time"
	"errors"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	//"github.com/davecgh/go-spew/spew"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
)

type TencentNLBHandler struct {
	Region idrv.RegionInfo
	Client *clb.Client
}

const (
	LoadBalancerSet_Status_Creating  int = 0
	LoadBalancerSet_Status_Running int = 1
)

func (NLBHandler *TencentNLBHandler) CreateNLB(nlbReqInfo irs.NLBInfo) (irs.NLBInfo, error) { 
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: nlbReqInfo.IId.NameId,
		CloudOSAPI:   "CreateLoadBalancer()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	// NLB 생성
	nlbRequest := clb.NewCreateLoadBalancerRequest()

	nlbRequest.LoadBalancerName = common.StringPtr(nlbReqInfo.IId.NameId)
	nlbRequest.LoadBalancerType = common.StringPtr("OPEN")
	nlbRequest.VpcId = common.StringPtr(nlbReqInfo.VpcIID.SystemId)

	nlbResponse, nlbErr := NLBHandler.Client.CreateLoadBalancer(nlbRequest)
	if nlbErr != nil {
		return irs.NLBInfo{}, nlbErr
	}

	cblogger.Debug(nlbResponse.ToJsonString())
	newNLBId := *nlbResponse.Response.LoadBalancerIds[0]
	cblogger.Debug(newNLBId)

	// nlb가 running 상태가 되길 기다림
	curStatus, statusErr := NLBHandler.WaitForRun(irs.IID{SystemId: newNLBId})
	if statusErr != nil {
		return irs.NLBInfo{}, statusErr
	}


	// Listener 생성
	if curStatus == "Running" {
		
		listenerPort, portErr := strconv.ParseInt(nlbReqInfo.Listener.Port, 10, 64)
		if portErr != nil {
			return irs.NLBInfo{}, portErr
		}

		healthPort, healthErr := strconv.ParseInt(nlbReqInfo.HealthChecker.Port, 10, 64)
		if healthErr != nil {
			return irs.NLBInfo{}, healthErr
		}

		listenerRequest := clb.NewCreateListenerRequest()

		listenerRequest.LoadBalancerId = common.StringPtr(newNLBId)
		listenerRequest.Ports = common.Int64Ptrs([]int64{ listenerPort })
		listenerRequest.Protocol = common.StringPtr(nlbReqInfo.Listener.Protocol)
		listenerRequest.HealthCheck = &clb.HealthCheck {
			TimeOut: common.Int64Ptr(int64(nlbReqInfo.HealthChecker.Timeout)),
			IntervalTime: common.Int64Ptr(int64(nlbReqInfo.HealthChecker.Interval)),
			HealthNum: common.Int64Ptr(int64(nlbReqInfo.HealthChecker.Threshold)),
			CheckPort: common.Int64Ptr(healthPort),
			CheckType: common.StringPtr(nlbReqInfo.HealthChecker.Protocol),
		}

		listenerResponse, listenerErr := NLBHandler.Client.CreateListener(listenerRequest)
		if listenerErr != nil {
			return irs.NLBInfo{}, listenerErr
		}

		fmt.Printf("%s", listenerResponse.ToJsonString())

		newListenerId := *listenerResponse.Response.ListenerIds[0]
		backendPort, backendErr := strconv.ParseInt(nlbReqInfo.VMGroup.Port, 10, 64)
		if backendErr != nil {
			return irs.NLBInfo{}, backendErr
		}

		// Listener가 생성되길 기다림
	    listStatus, listStatErr := NLBHandler.WaitForDone(newNLBId, newListenerId)
		if listStatErr != nil {
			return irs.NLBInfo{}, listStatErr
		}

		// VM 연결
		if listStatus == "Done" {
			targetRequest := clb.NewRegisterTargetsRequest()
			
			targetRequest.LoadBalancerId = common.StringPtr(newNLBId)
			targetRequest.ListenerId = common.StringPtr(newListenerId)
			targetRequest.Targets = []*clb.Target {}
			for _, target := range *nlbReqInfo.VMGroup.VMs {
				targetRequest.Targets = append(targetRequest.Targets, &clb.Target {
					InstanceId: common.StringPtr(target.SystemId),
					Port: common.Int64Ptr(backendPort),
				})
			}

			cblogger.Debug(targetRequest.ToJsonString())

			targetResponse, targetErr := NLBHandler.Client.RegisterTargets(targetRequest)
			if targetErr != nil {
				return irs.NLBInfo{}, targetErr
			}
			fmt.Printf("%s", targetResponse.ToJsonString())
		}
		
	}
	

	callogger.Info(call.String(callLogInfo))
	
	fmt.Printf("%s", nlbResponse.ToJsonString())

	

	nlbInfo, nlbInfoErr := NLBHandler.GetNLB(irs.IID{SystemId: newNLBId})
	if nlbInfoErr != nil {
		return irs.NLBInfo{}, nlbInfoErr
	}

	return nlbInfo, nlbInfoErr
}


func (NLBHandler *TencentNLBHandler) ListNLB() ([]*irs.NLBInfo, error) { 
	cblogger.Info("Start")

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "ListNLB",
		CloudOSAPI:   "DescribeLoadBalancers()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := clb.NewDescribeLoadBalancersRequest()
	callLogStart := call.Start()
	response, err := NLBHandler.Client.DescribeLoadBalancers(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	cblogger.Debug(response.ToJsonString())
	
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		cblogger.Error(err)
		return nil, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Info("NLB 개수 : ", *response.Response.TotalCount)

	var nlbInfoList []*irs.NLBInfo
	if *response.Response.TotalCount > 0 {
		for _, curNLB := range response.Response.LoadBalancerSet {
			cblogger.Debugf("[%s] NLB 정보 조회 - [%s]", *curNLB.LoadBalancerId, *curNLB.LoadBalancerName)
			nlbInfo, nlbErr := NLBHandler.GetNLB(irs.IID{SystemId: *curNLB.LoadBalancerId})
			
			if nlbErr != nil {
				cblogger.Error(nlbErr)
				return nil, nlbErr
			}
			nlbInfoList = append(nlbInfoList, &nlbInfo)
		}
	}

	cblogger.Debugf("리턴 결과 목록 수 : [%d]", len(nlbInfoList))
	
	return nlbInfoList, nil
}

func (NLBHandler *TencentNLBHandler) GetNLB(nlbIID irs.IID) (irs.NLBInfo, error) {
	cblogger.Info("NLB IID : ", nlbIID.SystemId)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "GetNLB",
		CloudOSAPI:   "DescribeLoadBalancers()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := clb.NewDescribeLoadBalancersRequest()
	request.LoadBalancerIds = common.StringPtrs([]string{ nlbIID.SystemId })
	callLogStart := call.Start()
	response, err := NLBHandler.Client.DescribeLoadBalancers(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		cblogger.Errorf("An API error has returned: %s", err.Error())
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		return irs.NLBInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Debug("NLB 개수 : ", *response.Response.TotalCount)
	if *response.Response.TotalCount < 1 {
		return irs.NLBInfo{}, errors.New("Notfound: '" + nlbIID.SystemId + "' NLB Not found")
	}

	nlbInfo := ExtractNLBDescribeInfo(response.Response.LoadBalancerSet[0])
	nlbInfo.Listener = NLBHandler.ExtractListenerInfo(nlbIID)
	nlbInfo.VMGroup = NLBHandler.ExtractVMGroupInfo(nlbIID)
	nlbInfo.HealthChecker = NLBHandler.ExtractHealthCheckerInfo(nlbIID)

	cblogger.Debug(nlbInfo)

	return nlbInfo, nil
}

func ExtractNLBDescribeInfo(nlbInfo *clb.LoadBalancer) irs.NLBInfo {
	
	resNLBInfo := irs.NLBInfo{
		
		IId:       irs.IID{SystemId: *nlbInfo.LoadBalancerId, NameId: *nlbInfo.LoadBalancerName},
	    VpcIID:    irs.IID{SystemId: *nlbInfo.VpcId},

	}

	return resNLBInfo
}

func (NLBHandler *TencentNLBHandler) ExtractListenerInfo(nlbIID irs.IID) irs.ListenerInfo {
	cblogger.Info("NLB IID : ", nlbIID.SystemId)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "GetNLB",
		CloudOSAPI:   "DescribeListeners()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := clb.NewDescribeListenersRequest()
	request.LoadBalancerId = common.StringPtr(nlbIID.SystemId)
	callLogStart := call.Start()
	response, err := NLBHandler.Client.DescribeListeners(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		cblogger.Errorf("An API error has returned: %s", err.Error())
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		return irs.ListenerInfo{}
	}
	callogger.Info(call.String(callLogInfo))

	resListenerInfo := irs.ListenerInfo{
		Protocol: *response.Response.Listeners[0].Protocol,
		Port: strconv.FormatInt(*response.Response.Listeners[0].Port,10),
	}

	return resListenerInfo
}

func (NLBHandler *TencentNLBHandler) ExtractVMGroupInfo(nlbIID irs.IID) irs.VMGroupInfo {
	cblogger.Info("NLB IID : ", nlbIID.SystemId)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "GetNLB",
		CloudOSAPI:   "DescribeTargets()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := clb.NewDescribeTargetsRequest()
	request.LoadBalancerId = common.StringPtr(nlbIID.SystemId)
	callLogStart := call.Start()
	response, err := NLBHandler.Client.DescribeTargets(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		cblogger.Errorf("An API error has returned: %s", err.Error())
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		return irs.VMGroupInfo{}
	}
	callogger.Info(call.String(callLogInfo))

	resVmInfo := irs.VMGroupInfo{
		Protocol: "TCP",
		Port: strconv.FormatInt(*response.Response.Listeners[0].Targets[0].Port,10),
	}

	vms := []irs.IID{}
	for _, target := range response.Response.Listeners[0].Targets{
		vms = append(vms, irs.IID{SystemId: *target.InstanceId})
	}

	resVmInfo.VMs = &vms

	return resVmInfo
}

func (NLBHandler *TencentNLBHandler) ExtractHealthCheckerInfo(nlbIID irs.IID) irs.HealthCheckerInfo {
	cblogger.Info("NLB IID : ", nlbIID.SystemId)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "GetNLB",
		CloudOSAPI:   "DescribeListeners()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := clb.NewDescribeListenersRequest()
	request.LoadBalancerId = common.StringPtr(nlbIID.SystemId)
	callLogStart := call.Start()
	response, err := NLBHandler.Client.DescribeListeners(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		cblogger.Errorf("An API error has returned: %s", err.Error())
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		return irs.HealthCheckerInfo{}
	}
	callogger.Info(call.String(callLogInfo))

	resHealthCheckerInfo := irs.HealthCheckerInfo{
		Protocol: *response.Response.Listeners[0].HealthCheck.CheckType,
		Port: strconv.FormatInt(*response.Response.Listeners[0].HealthCheck.CheckPort,10),
		Interval: int(*response.Response.Listeners[0].HealthCheck.IntervalTime),
		Timeout: int(*response.Response.Listeners[0].HealthCheck.TimeOut),
		Threshold: int(*response.Response.Listeners[0].HealthCheck.HealthNum),
	}

	return resHealthCheckerInfo
}

func (NLBHandler *TencentNLBHandler) DeleteNLB(nlbIID irs.IID) (bool, error) {
	cblogger.Info("NLB IID : ", nlbIID.SystemId)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "DeleteNLB",
		CloudOSAPI:   "DeleteLoadBalancer()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := clb.NewDeleteLoadBalancerRequest()
	request.LoadBalancerIds = common.StringPtrs([]string{nlbIID.SystemId})
	callLogStart := call.Start()
	_, err := NLBHandler.Client.DeleteLoadBalancer(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		cblogger.Errorf("An API error has returned: %s", err.Error())
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		return false, err
	}

	callogger.Info(call.String(callLogInfo))

	return true, nil
}

func (NLBHandler *TencentNLBHandler) ChangeListener(nlbIID irs.IID, listener irs.ListenerInfo) (irs.ListenerInfo, error) {
	return irs.ListenerInfo{}, nil
}

func (NLBHandler *TencentNLBHandler) ChangeVMGroupInfo(nlbIID irs.IID, vmGroup irs.VMGroupInfo) (irs.VMGroupInfo, error) {
	return irs.VMGroupInfo{}, nil
}

func (NLBHandler *TencentNLBHandler) AddVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (irs.VMGroupInfo, error) {
	return irs.VMGroupInfo{}, nil
}

func (NLBHandler *TencentNLBHandler) RemoveVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {
	return false, nil
}

func (NLBHandler *TencentNLBHandler) GetVMGroupHealthInfo(nlbIID irs.IID) (irs.HealthInfo, error) {
	cblogger.Info("NLB IID : ", nlbIID.SystemId)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "GetVMGroupHealthInfo",
		CloudOSAPI:   "DescribeTargetHealth()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := clb.NewDescribeTargetHealthRequest()
	request.LoadBalancerIds = common.StringPtrs([]string{nlbIID.SystemId})
	response, err := NLBHandler.Client.DescribeTargetHealth(request)
	if err != nil {
		cblogger.Errorf("An API error has returned: %s", err.Error())
		callLogInfo.ErrorMSG = err.Error()
		callogger.Error(call.String(callLogInfo))
		return irs.HealthInfo{}, err
	}

	vmGroup := response.Response.LoadBalancers[0].Listeners[0].Rules[0].Targets

	allVMs := []irs.IID{}
	healthyVMs := []irs.IID{}
	unHealthyVMs := []irs.IID{}

	for _, vm := range vmGroup {
		allVMs = append(allVMs, irs.IID{SystemId:*vm.TargetId})
		if *vm.HealthStatus {
			healthyVMs = append(healthyVMs, irs.IID{SystemId:*vm.TargetId})
		} else {
			unHealthyVMs = append(unHealthyVMs, irs.IID{SystemId:*vm.TargetId})
		}
	}

	healthInfo := irs.HealthInfo{}
	healthInfo.AllVMs = &allVMs
	healthInfo.HealthyVMs = &healthyVMs
	healthInfo.UnHealthyVMs = &unHealthyVMs

	return healthInfo, nil
}

func (NLBHandler *TencentNLBHandler) ChangeHealthCheckerInfo(nlbIID irs.IID, healthChecker irs.HealthCheckerInfo) (irs.HealthCheckerInfo, error) {
	return irs.HealthCheckerInfo{}, nil
}

func (NLBHandler *TencentNLBHandler) WaitForRun(nlbIID irs.IID) (string, error) {

	waitStatus := "Running"

	curRetryCnt := 0
	maxRetryCnt := 120
	for {
		request := clb.NewDescribeLoadBalancersRequest()
        
        request.LoadBalancerIds = common.StringPtrs([]string{ nlbIID.SystemId })
        response, errStatus := NLBHandler.Client.DescribeLoadBalancers(request)
		if errStatus != nil {
			cblogger.Error(errStatus.Error())
		}

		curStatus := *response.Response.LoadBalancerSet[0].Status

		cblogger.Info("===>NLB Status : ", curStatus)

		if curStatus == LoadBalancerSet_Status_Running { 
			cblogger.Infof("===>NLB 상태가 [%d]라서 대기를 중단합니다.", curStatus)
			break
		}

		curRetryCnt++
		cblogger.Errorf("NLB 상태가 [%s]이 아니라서 1초 대기후 조회합니다.", waitStatus)
		time.Sleep(time.Second * 1)
		if curRetryCnt > maxRetryCnt {
			cblogger.Errorf("장시간(%d 초) 대기해도 NLB Status 값이 [%s]으로 변경되지 않아서 강제로 중단합니다.", maxRetryCnt, waitStatus)
			return "Failed", errors.New("장시간 기다렸으나 생성된 NLB의 상태가 [" + waitStatus + "]으로 바뀌지 않아서 중단 합니다.")
		}
	}

	return waitStatus, nil
}


func (NLBHandler *TencentNLBHandler) WaitForDone(nlbId string, listenerId string) (string, error) {

	waitStatus := "Done"

	curRetryCnt := 0
	maxRetryCnt := 120
	for {
		request := clb.NewDescribeListenersRequest()
        
        request.LoadBalancerId = common.StringPtr(nlbId)
        request.ListenerIds = common.StringPtrs([]string{ listenerId })

        response, err := NLBHandler.Client.DescribeListeners(request)
		if err != nil {
			cblogger.Error(err.Error())
		}

		listenerInfo := response.Response.Listeners

		cblogger.Info("===>listener info : ", listenerInfo)

		if len(listenerInfo) > 0 { 
			cblogger.Infof("===>listener 상태가 [%s]라서 대기를 중단합니다.", waitStatus)
			break
		}

		curRetryCnt++
		cblogger.Errorf("listener 상태가 [%s]이 아니라서 1초 대기후 조회합니다.", waitStatus)
		time.Sleep(time.Second * 1)
		if curRetryCnt > maxRetryCnt {
			cblogger.Errorf("장시간(%d 초) 대기해도 listener Status 값이 [%s]으로 변경되지 않아서 강제로 중단합니다.", maxRetryCnt, waitStatus)
			return "Failed", errors.New("장시간 기다렸으나 생성된 listener 상태가 [" + waitStatus + "]으로 바뀌지 않아서 중단 합니다.")
		}
	}

	return waitStatus, nil
}