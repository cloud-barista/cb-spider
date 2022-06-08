package resources

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	//"github.com/davecgh/go-spew/spew"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
)

type TencentNLBHandler struct {
	Region idrv.RegionInfo
	Client *clb.Client
}

const (
	LoadBalancerSet_Status_Creating uint64 = 0
	LoadBalancerSet_Status_Running  uint64 = 1
)

const (
	Request_Status_Succeeded int64 = 0
	Request_Status_Failed    int64 = 1
	Request_Status_Progress  int64 = 2
)

const (
	Request_Status_Running string = "Running"
	Request_Status_Done    string = "Done"
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

	nlbResult := irs.NLBInfo{}

	// NLB 생성
	nlbRequest := clb.NewCreateLoadBalancerRequest()

	nlbRequest.LoadBalancerName = common.StringPtr(nlbReqInfo.IId.NameId)
	if strings.EqualFold(nlbReqInfo.Type, "") || strings.EqualFold(nlbReqInfo.Type, "PUBLIC") {
		nlbRequest.LoadBalancerType = common.StringPtr("OPEN")
	} else {
		nlbRequest.LoadBalancerType = common.StringPtr("INTERNAL")
	}
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
	if curStatus == Request_Status_Running {

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
		listenerRequest.Ports = common.Int64Ptrs([]int64{listenerPort})
		listenerRequest.Protocol = common.StringPtr(nlbReqInfo.Listener.Protocol)
		listenerRequest.HealthCheck = &clb.HealthCheck{
			TimeOut:      common.Int64Ptr(int64(nlbReqInfo.HealthChecker.Timeout)),
			IntervalTime: common.Int64Ptr(int64(nlbReqInfo.HealthChecker.Interval)),
			HealthNum:    common.Int64Ptr(int64(nlbReqInfo.HealthChecker.Threshold)),
			CheckPort:    common.Int64Ptr(healthPort),
			CheckType:    common.StringPtr(nlbReqInfo.HealthChecker.Protocol),
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
		listStatus, listStatErr := NLBHandler.WaitForDone(*listenerResponse.Response.RequestId)
		if listStatErr != nil {
			return irs.NLBInfo{}, listStatErr
		}

		// VM 연결
		if listStatus == Request_Status_Done {
			targetRequest := clb.NewRegisterTargetsRequest()

			targetRequest.LoadBalancerId = common.StringPtr(newNLBId)
			targetRequest.ListenerId = common.StringPtr(newListenerId)
			targetRequest.Targets = []*clb.Target{}
			for _, target := range *nlbReqInfo.VMGroup.VMs {
				targetRequest.Targets = append(targetRequest.Targets, &clb.Target{
					InstanceId: common.StringPtr(target.SystemId),
					Port:       common.Int64Ptr(backendPort),
				})
			}

			cblogger.Debug(targetRequest.ToJsonString())

			targetResponse, targetErr := NLBHandler.Client.RegisterTargets(targetRequest)
			if targetErr != nil {
				return irs.NLBInfo{}, targetErr
			}
			fmt.Printf("%s", targetResponse.ToJsonString())

			// VM 연결되길 기다림
			targetStatus, targetStatErr := NLBHandler.WaitForDone(*targetResponse.Response.RequestId)
			if targetStatErr != nil {
				return irs.NLBInfo{}, targetStatErr
			}

			if targetStatus == Request_Status_Done {
				nlbInfo, nlbInfoErr := NLBHandler.GetNLB(irs.IID{SystemId: newNLBId})
				if nlbInfoErr != nil {
					return irs.NLBInfo{}, nlbInfoErr
				}

				nlbResult = nlbInfo
			}
		}
	}

	callogger.Info(call.String(callLogInfo))

	fmt.Printf("%s", nlbResponse.ToJsonString())

	return nlbResult, nil
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
	request.LoadBalancerIds = common.StringPtrs([]string{nlbIID.SystemId})
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
	listener, listenerErr := NLBHandler.ExtractListenerInfo(nlbIID)
	if listenerErr != nil {
		return irs.NLBInfo{}, listenerErr
	}
	healthChecker, healthCheckerErr := NLBHandler.ExtractHealthCheckerInfo(nlbIID)
	if healthCheckerErr != nil {
		return irs.NLBInfo{}, healthCheckerErr
	}
	vmGroup, vmGroupErr := NLBHandler.ExtractVMGroupInfo(nlbIID)
	if vmGroupErr != nil {
		return irs.NLBInfo{}, vmGroupErr
	}
	nlbInfo.Listener = listener
	nlbInfo.HealthChecker = healthChecker
	nlbInfo.VMGroup = vmGroup

	cblogger.Debug(nlbInfo)

	return nlbInfo, nil
}

func ExtractNLBDescribeInfo(nlbInfo *clb.LoadBalancer) irs.NLBInfo {

	createTime, _ := time.Parse("2006-01-02 15:04:05", *nlbInfo.CreateTime)
	nlbType := ""
	if strings.EqualFold(*nlbInfo.LoadBalancerType, "OPEN") {
		nlbType = "PUBLIC"
	} else {
		nlbType = "INTERNAL"
	}

	resNLBInfo := irs.NLBInfo{

		IId:         irs.IID{SystemId: *nlbInfo.LoadBalancerId, NameId: *nlbInfo.LoadBalancerName},
		VpcIID:      irs.IID{SystemId: *nlbInfo.VpcId},
		CreatedTime: createTime,
		Type:        nlbType,
		Scope:       "REGION",
	}

	return resNLBInfo
}

func (NLBHandler *TencentNLBHandler) ExtractListenerInfo(nlbIID irs.IID) (irs.ListenerInfo, error) {
	cblogger.Info("NLB IID : ", nlbIID.SystemId)

	request := clb.NewDescribeListenersRequest()
	request.LoadBalancerId = common.StringPtr(nlbIID.SystemId)
	response, err := NLBHandler.Client.DescribeListeners(request)
	if err != nil {
		cblogger.Errorf("An API error has returned: %s", err.Error())
		return irs.ListenerInfo{}, err
	}

	resListenerInfo := irs.ListenerInfo{
		Protocol: *response.Response.Listeners[0].Protocol,
		Port:     strconv.FormatInt(*response.Response.Listeners[0].Port, 10),
	}

	ipRequest := clb.NewDescribeLoadBalancersRequest()
	ipRequest.LoadBalancerIds = common.StringPtrs([]string{nlbIID.SystemId})
	ipResponse, ipErr := NLBHandler.Client.DescribeLoadBalancers(ipRequest)
	if ipErr != nil {
		cblogger.Errorf("An API error has returned: %s", err.Error())
		return irs.ListenerInfo{}, ipErr
	}
	resListenerInfo.IP = *ipResponse.Response.LoadBalancerSet[0].LoadBalancerVips[0]

	return resListenerInfo, nil
}

func (NLBHandler *TencentNLBHandler) ExtractVMGroupInfo(nlbIID irs.IID) (irs.VMGroupInfo, error) {
	cblogger.Info("NLB IID : ", nlbIID.SystemId)

	request := clb.NewDescribeTargetsRequest()
	request.LoadBalancerId = common.StringPtr(nlbIID.SystemId)
	response, err := NLBHandler.Client.DescribeTargets(request)

	if err != nil {
		cblogger.Errorf("An API error has returned: %s", err.Error())
		return irs.VMGroupInfo{}, err
	}

	cblogger.Debug(response.Response.Listeners[0].Targets)

	if len(response.Response.Listeners[0].Targets) == 0 {
		return irs.VMGroupInfo{}, errors.New("Target VM does not exist!")
	}

	resVmInfo := irs.VMGroupInfo{
		Protocol: "TCP",
		Port:     strconv.FormatInt(*response.Response.Listeners[0].Targets[0].Port, 10),
	}

	vms := []irs.IID{}
	for _, target := range response.Response.Listeners[0].Targets {
		vms = append(vms, irs.IID{SystemId: *target.InstanceId})
	}

	resVmInfo.VMs = &vms

	return resVmInfo, nil
}

func (NLBHandler *TencentNLBHandler) ExtractHealthCheckerInfo(nlbIID irs.IID) (irs.HealthCheckerInfo, error) {
	cblogger.Info("NLB IID : ", nlbIID.SystemId)

	request := clb.NewDescribeListenersRequest()
	request.LoadBalancerId = common.StringPtr(nlbIID.SystemId)
	response, err := NLBHandler.Client.DescribeListeners(request)

	if err != nil {
		cblogger.Errorf("An API error has returned: %s", err.Error())
		return irs.HealthCheckerInfo{}, err
	}

	resHealthCheckerInfo := irs.HealthCheckerInfo{
		Protocol:  *response.Response.Listeners[0].HealthCheck.CheckType,
		Port:      strconv.FormatInt(*response.Response.Listeners[0].HealthCheck.CheckPort, 10),
		Interval:  int(*response.Response.Listeners[0].HealthCheck.IntervalTime),
		Timeout:   int(*response.Response.Listeners[0].HealthCheck.TimeOut),
		Threshold: int(*response.Response.Listeners[0].HealthCheck.HealthNum),
	}

	return resHealthCheckerInfo, nil
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
	cblogger.Info("TENCENT_CANNOT_CHANGE_LISTENER")
	return irs.ListenerInfo{}, nil
}

func (NLBHandler *TencentNLBHandler) ChangeVMGroupInfo(nlbIID irs.IID, vmGroup irs.VMGroupInfo) (irs.VMGroupInfo, error) {
	vmGroupInfo, vmGroupInfoErr := NLBHandler.ExtractVMGroupInfo(nlbIID)
	if vmGroupInfoErr != nil {
		return irs.VMGroupInfo{}, vmGroupInfoErr
	}

	newNLBId := nlbIID.SystemId
	vmGroupInfoResult := irs.VMGroupInfo{}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "ChangeVMGroupInfo",
		CloudOSAPI:   "ModifyTargetPort()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := clb.NewDescribeListenersRequest()
	request.LoadBalancerId = common.StringPtr(newNLBId)
	response, err := NLBHandler.Client.DescribeListeners(request)
	if err != nil {
		return irs.VMGroupInfo{}, err
	}

	newListenerId := *response.Response.Listeners[0].ListenerId
	port, portErr := strconv.ParseInt(vmGroupInfo.Port, 10, 64)
	if portErr != nil {
		return irs.VMGroupInfo{}, portErr
	}
	newPort, newPortErr := strconv.ParseInt(vmGroup.Port, 10, 64)
	if newPortErr != nil {
		return irs.VMGroupInfo{}, newPortErr
	}

	modifyTargetRequest := clb.NewModifyTargetPortRequest()

	modifyTargetRequest.LoadBalancerId = common.StringPtr(newNLBId)
	modifyTargetRequest.ListenerId = common.StringPtr(newListenerId)
	modifyTargetRequest.Targets = []*clb.Target{}
	for _, target := range *vmGroupInfo.VMs {
		modifyTargetRequest.Targets = append(modifyTargetRequest.Targets, &clb.Target{
			InstanceId: common.StringPtr(target.SystemId),
			Port:       common.Int64Ptr(port),
		})
	}

	modifyTargetRequest.NewPort = common.Int64Ptr(newPort)

	modifyTargetResponse, modifyTargetErr := NLBHandler.Client.ModifyTargetPort(modifyTargetRequest)
	if modifyTargetErr != nil {
		return irs.VMGroupInfo{}, modifyTargetErr
	}

	callogger.Info(call.String(callLogInfo))

	targetStatus, targetStatErr := NLBHandler.WaitForDone(*modifyTargetResponse.Response.RequestId)
	if targetStatErr != nil {
		return irs.VMGroupInfo{}, targetStatErr
	}

	if targetStatus == Request_Status_Done {
		result, resultErr := NLBHandler.ExtractVMGroupInfo(nlbIID)
		if resultErr != nil {
			return irs.VMGroupInfo{}, resultErr
		}
		vmGroupInfoResult = result
	}

	return vmGroupInfoResult, nil
}

func (NLBHandler *TencentNLBHandler) AddVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (irs.VMGroupInfo, error) {

	newNLBId := nlbIID.SystemId
	vmGroupInfoResult := irs.VMGroupInfo{}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "AddVMs",
		CloudOSAPI:   "RegisterTargets()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := clb.NewDescribeListenersRequest()
	request.LoadBalancerId = common.StringPtr(newNLBId)
	response, err := NLBHandler.Client.DescribeListeners(request)
	if err != nil {
		return irs.VMGroupInfo{}, err
	}

	newListenerId := *response.Response.Listeners[0].ListenerId
	backendPort := int64(0)
	vmGroupInfo, vmGroupInfoErr := NLBHandler.ExtractVMGroupInfo(nlbIID)
	if vmGroupInfoErr != nil {
		backendPort = *response.Response.Listeners[0].Port
	} else {
		port, portErr := strconv.ParseInt(vmGroupInfo.Port, 10, 64)
		if portErr != nil {
			return irs.VMGroupInfo{}, portErr
		}
		backendPort = port

	}

	targetRequest := clb.NewRegisterTargetsRequest()

	targetRequest.LoadBalancerId = common.StringPtr(newNLBId)
	targetRequest.ListenerId = common.StringPtr(newListenerId)
	targetRequest.Targets = []*clb.Target{}
	for _, target := range *vmIIDs {
		targetRequest.Targets = append(targetRequest.Targets, &clb.Target{
			InstanceId: common.StringPtr(target.SystemId),
			Port:       common.Int64Ptr(backendPort),
		})
	}

	cblogger.Debug(targetRequest.ToJsonString())

	targetResponse, targetErr := NLBHandler.Client.RegisterTargets(targetRequest)
	if targetErr != nil {
		return irs.VMGroupInfo{}, targetErr
	}
	fmt.Printf("%s", targetResponse.ToJsonString())

	callogger.Info(call.String(callLogInfo))

	// VM 연결되길 기다림
	targetStatus, targetStatErr := NLBHandler.WaitForDone(*targetResponse.Response.RequestId)
	if targetStatErr != nil {
		return irs.VMGroupInfo{}, targetStatErr
	}

	if targetStatus == Request_Status_Done {
		vmGroupInfo, vmGroupInfoErr := NLBHandler.ExtractVMGroupInfo(nlbIID)
		if vmGroupInfoErr != nil {
			return irs.VMGroupInfo{}, vmGroupInfoErr
		}
		vmGroupInfoResult = vmGroupInfo
	}

	return vmGroupInfoResult, nil
}

func (NLBHandler *TencentNLBHandler) RemoveVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {
	newNLBId := nlbIID.SystemId
	//vmGroupInfoResult := irs.VMGroupInfo{}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "RemoveVMs",
		CloudOSAPI:   "DeregisterTargets()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := clb.NewDescribeListenersRequest()
	request.LoadBalancerId = common.StringPtr(newNLBId)
	response, err := NLBHandler.Client.DescribeListeners(request)
	if err != nil {
		return false, err
	}

	newListenerId := *response.Response.Listeners[0].ListenerId
	backendPort := int64(0)
	vmGroupInfo, vmGroupInfoErr := NLBHandler.ExtractVMGroupInfo(nlbIID)
	if vmGroupInfoErr != nil {
		return false, vmGroupInfoErr
	} else {
		port, portErr := strconv.ParseInt(vmGroupInfo.Port, 10, 64)
		if portErr != nil {
			return false, portErr
		}
		backendPort = port

	}

	targetRequest := clb.NewDeregisterTargetsRequest()

	targetRequest.LoadBalancerId = common.StringPtr(newNLBId)
	targetRequest.ListenerId = common.StringPtr(newListenerId)
	targetRequest.Targets = []*clb.Target{}
	for _, target := range *vmIIDs {
		targetRequest.Targets = append(targetRequest.Targets, &clb.Target{
			InstanceId: common.StringPtr(target.SystemId),
			Port:       common.Int64Ptr(backendPort),
		})
	}

	cblogger.Debug(targetRequest.ToJsonString())

	targetResponse, targetErr := NLBHandler.Client.DeregisterTargets(targetRequest)
	if targetErr != nil {
		return false, targetErr
	}
	fmt.Printf("%s", targetResponse.ToJsonString())

	callogger.Info(call.String(callLogInfo))

	return true, nil
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

	callogger.Info(call.String(callLogInfo))

	vmGroup := response.Response.LoadBalancers[0].Listeners[0].Rules[0].Targets

	allVMs := []irs.IID{}
	healthyVMs := []irs.IID{}
	unHealthyVMs := []irs.IID{}

	for _, vm := range vmGroup {
		allVMs = append(allVMs, irs.IID{SystemId: *vm.TargetId})
		if *vm.HealthStatus {
			healthyVMs = append(healthyVMs, irs.IID{SystemId: *vm.TargetId})
		} else {
			unHealthyVMs = append(unHealthyVMs, irs.IID{SystemId: *vm.TargetId})
		}
	}

	healthInfo := irs.HealthInfo{}
	healthInfo.AllVMs = &allVMs
	healthInfo.HealthyVMs = &healthyVMs
	healthInfo.UnHealthyVMs = &unHealthyVMs

	return healthInfo, nil
}

func (NLBHandler *TencentNLBHandler) ChangeHealthCheckerInfo(nlbIID irs.IID, healthChecker irs.HealthCheckerInfo) (irs.HealthCheckerInfo, error) {
	newNLBId := nlbIID.SystemId
	healthCheckerResult := irs.HealthCheckerInfo{}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.TENCENT,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "ChangeHealthCheckerInfo",
		CloudOSAPI:   "ModifyListener()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}

	request := clb.NewDescribeListenersRequest()
	request.LoadBalancerId = common.StringPtr(newNLBId)
	response, err := NLBHandler.Client.DescribeListeners(request)
	if err != nil {
		return irs.HealthCheckerInfo{}, err
	}

	newListenerId := *response.Response.Listeners[0].ListenerId

	healthPort, healthPortErr := strconv.ParseInt(healthChecker.Port, 10, 64)
	if healthPortErr != nil {
		return irs.HealthCheckerInfo{}, healthPortErr
	}

	changeHealthCheckerRequest := clb.NewModifyListenerRequest()

	changeHealthCheckerRequest.LoadBalancerId = common.StringPtr(newNLBId)
	changeHealthCheckerRequest.ListenerId = common.StringPtr(newListenerId)
	changeHealthCheckerRequest.HealthCheck = &clb.HealthCheck{
		TimeOut:      common.Int64Ptr(int64(healthChecker.Timeout)),
		IntervalTime: common.Int64Ptr(int64(healthChecker.Interval)),
		HealthNum:    common.Int64Ptr(int64(healthChecker.Threshold)),
		CheckPort:    common.Int64Ptr(healthPort),
		CheckType:    common.StringPtr(healthChecker.Protocol),
	}

	changeHealthCheckerResponse, err := NLBHandler.Client.ModifyListener(changeHealthCheckerRequest)

	callogger.Info(call.String(callLogInfo))

	// VM 연결되길 기다림
	changeStatus, changeStatErr := NLBHandler.WaitForDone(*changeHealthCheckerResponse.Response.RequestId)
	if changeStatErr != nil {
		return irs.HealthCheckerInfo{}, changeStatErr
	}

	if changeStatus == Request_Status_Done {
		healthCheckerInfo, healthErr := NLBHandler.ExtractHealthCheckerInfo(nlbIID)
		if healthErr != nil {
			return irs.HealthCheckerInfo{}, healthErr
		}
		healthCheckerResult = healthCheckerInfo
	}

	return healthCheckerResult, nil
}

func (NLBHandler *TencentNLBHandler) WaitForRun(nlbIID irs.IID) (string, error) {

	waitStatus := "Running"

	curRetryCnt := 0
	maxRetryCnt := 120
	for {
		request := clb.NewDescribeLoadBalancersRequest()

		request.LoadBalancerIds = common.StringPtrs([]string{nlbIID.SystemId})
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

func (NLBHandler *TencentNLBHandler) WaitForDone(requestId string) (string, error) {

	waitStatus := "Done"

	curRetryCnt := 0
	maxRetryCnt := 120
	for {
		request := clb.NewDescribeTaskStatusRequest()

		request.TaskId = common.StringPtr(requestId)

		response, err := NLBHandler.Client.DescribeTaskStatus(request)
		if err != nil {
			cblogger.Error(err.Error())
		}

		requestStatus := *response.Response.Status

		cblogger.Info("===>request status : ", requestStatus)

		if requestStatus == Request_Status_Succeeded {
			cblogger.Infof("===>request 상태가 [%s]라서 대기를 중단합니다.", waitStatus)
			break
		}

		curRetryCnt++
		cblogger.Errorf("request 상태가 [%s]이 아니라서 1초 대기후 조회합니다.", waitStatus)
		time.Sleep(time.Second * 1)
		if curRetryCnt > maxRetryCnt {
			cblogger.Errorf("장시간(%d 초) 대기해도 request Status 값이 [%s]으로 변경되지 않아서 강제로 중단합니다.", maxRetryCnt, waitStatus)
			return "Failed", errors.New("장시간 기다렸으나 생성된 request 상태가 [" + waitStatus + "]으로 바뀌지 않아서 중단 합니다.")
		}
	}

	return waitStatus, nil
}
