package resources

import (
	"errors"
	"strconv"

	"strings"

	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	//"github.com/davecgh/go-spew/spew"

	clb "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/clb/v20180317"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	tencentvpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

type TencentNLBHandler struct {
	Region    idrv.RegionInfo
	Client    *clb.Client
	VpcClient *tencentvpc.Client
}

const (
	//
	LoadBalancerSet_Status_Creating uint64 = 0
	LoadBalancerSet_Status_Running  uint64 = 1

	Tencent_LoadBalancerType_Open     string = "OPEN" // to Spider "PUBLIC
	Tencent_LoadBalancerType_INTERNAL string = "INTERNAL"
	Spider_LoadBalancerType_PUBLIC    string = "PUBLIC" // to Tencent "OPEN"
	Spider_LoadBalancerType_INTERNAL  string = "INTERNAL"

	// Request Status : Succeeded, Failed, Progress
	Request_Status_Succeeded int64 = 0
	Request_Status_Failed    int64 = 1
	Request_Status_Progress  int64 = 2

	// Request Status : Running, Done
	Request_Status_Running string = "Running"
	Request_Status_Done    string = "Done"

	Protocol_TCP string = "TCP"
	Protocol_UDP string = "UDP"
)

/*
NLB 생성
vpc required
*/
func (NLBHandler *TencentNLBHandler) CreateNLB(nlbReqInfo irs.NLBInfo) (irs.NLBInfo, error) {
	////// validation check area //////
	// NLB 이름 중복 체크
	existName, errExist := NLBHandler.nlbExist(nlbReqInfo.IId.NameId)
	if errExist != nil {
		cblogger.Error(errExist)
		return irs.NLBInfo{}, errExist
	}
	if existName {
		return irs.NLBInfo{}, errors.New("A NLB with the name " + nlbReqInfo.IId.NameId + " already exists.")
	}

	healthCheckerProtocol := nlbReqInfo.HealthChecker.Protocol
	healthCheckerTimeOut := int64(nlbReqInfo.HealthChecker.Timeout)
	healthCheckerInterval := int64(nlbReqInfo.HealthChecker.Interval)
	healthCheckerThreshold := int64(nlbReqInfo.HealthChecker.Threshold)
	if healthCheckerTimeOut > 0 && healthCheckerInterval > 0 && healthCheckerTimeOut > healthCheckerInterval {
		return irs.NLBInfo{}, errors.New("HealthCheck.IntervalTime should not be less than HealthCheck.TimeOut")
	}

	listenerProtocol := nlbReqInfo.Listener.Protocol
	listenerPort, portErr := strconv.ParseInt(nlbReqInfo.Listener.Port, 10, 64)
	if portErr != nil {
		return irs.NLBInfo{}, portErr
	}

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

	// NLB request생성
	nlbRequest := clb.NewCreateLoadBalancerRequest()

	nlbRequest.LoadBalancerName = common.StringPtr(nlbReqInfo.IId.NameId)

	if strings.EqualFold(nlbReqInfo.Type, "") || strings.EqualFold(nlbReqInfo.Type, Spider_LoadBalancerType_PUBLIC) {
		nlbRequest.LoadBalancerType = common.StringPtr(Tencent_LoadBalancerType_Open)
	} else {
		nlbRequest.LoadBalancerType = common.StringPtr(Tencent_LoadBalancerType_INTERNAL)
	}

	nlbRequest.VpcId = common.StringPtr(nlbReqInfo.VpcIID.SystemId)

	var tags []*clb.TagInfo
	for _, inputTag := range nlbReqInfo.TagList {
		tags = append(tags, &clb.TagInfo{
			TagKey:   common.StringPtr(inputTag.Key),
			TagValue: common.StringPtr(inputTag.Value),
		})
	}
	nlbRequest.Tags = tags

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

	// Listener 생성( listener + healthchecker)
	if curStatus == Request_Status_Running {

		listenerRequest := clb.NewCreateListenerRequest()

		listenerRequest.HealthCheck = &clb.HealthCheck{}
		listenerRequest.LoadBalancerId = common.StringPtr(newNLBId)
		listenerRequest.Ports = common.Int64Ptrs([]int64{listenerPort})
		listenerRequest.Protocol = common.StringPtr(listenerProtocol)
		listenerRequest.ListenerNames = common.StringPtrs([]string{nlbReqInfo.IId.NameId})

		// health Checker port 값이 있을 때만 setting
		if !strings.EqualFold(nlbReqInfo.HealthChecker.Port, "") {
			healthPort, healthErr := strconv.ParseInt(nlbReqInfo.HealthChecker.Port, 10, 64)
			if healthErr != nil {
				return irs.NLBInfo{}, healthErr
			}
			listenerRequest.HealthCheck.CheckPort = common.Int64Ptr(healthPort)
		}

		if healthCheckerTimeOut > 0 {
			listenerRequest.HealthCheck.TimeOut = common.Int64Ptr(healthCheckerTimeOut)
		}
		if healthCheckerInterval > 0 {
			listenerRequest.HealthCheck.IntervalTime = common.Int64Ptr(healthCheckerInterval)
		}
		if healthCheckerThreshold > 0 {
			listenerRequest.HealthCheck.HealthNum = common.Int64Ptr(healthCheckerThreshold)
		}
		if !strings.EqualFold(nlbReqInfo.HealthChecker.Protocol, "") {
			listenerRequest.HealthCheck.CheckType = common.StringPtr(nlbReqInfo.HealthChecker.Protocol)
		}

		// Listener의 protocol이 UDP일 때 HealthChecker의 CheckType, ContextType은 고정
		if strings.EqualFold(listenerProtocol, "UDP") {
			listenerRequest.HealthCheck.CheckType = common.StringPtr("CUSTOM")
			listenerRequest.HealthCheck.ContextType = common.StringPtr("TEXT")
		}

		// HealthChecker protocol이 HTTP일 때 Domain Set
		if strings.EqualFold(healthCheckerProtocol, "HTTP") {
			listenerRequest.HealthCheck.HttpCheckDomain = common.StringPtr("")
		}

		listenerResponse, listenerErr := NLBHandler.Client.CreateListener(listenerRequest)
		if listenerErr != nil {
			cblogger.Errorf("NLB CreateListner err: %s", listenerErr.Error())
			cblogger.Errorf("delete abnormal nlb")
			_, err := NLBHandler.DeleteNLB(irs.IID{SystemId: newNLBId})
			if err != nil {
				return irs.NLBInfo{}, err
			}
			return irs.NLBInfo{}, listenerErr
		}

		cblogger.Info("%s", listenerResponse.ToJsonString())

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
				cblogger.Errorf("NLB RegisterTargets err: %s", listenerErr.Error())
				cblogger.Errorf("delete abnormal nlb")
				_, err := NLBHandler.DeleteNLB(irs.IID{SystemId: newNLBId})
				if err != nil {
					return irs.NLBInfo{}, err
				}
				return irs.NLBInfo{}, targetErr
			}
			cblogger.Info("%s", targetResponse.ToJsonString())

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

	cblogger.Info("%s", nlbResponse.ToJsonString())

	return nlbResult, nil
}

/*
NLB 모든 목록 조회 : TCP/UDP
*/
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
			cblogger.Debugf("[%s] NLB information retrieval - [%s]", *curNLB.LoadBalancerId, *curNLB.LoadBalancerName)
			nlbInfo, nlbErr := NLBHandler.GetNLB(irs.IID{SystemId: *curNLB.LoadBalancerId})

			if nlbErr != nil {
				cblogger.Error(nlbErr)
				return nil, nlbErr
			}
			nlbInfoList = append(nlbInfoList, &nlbInfo)
		}
	}

	cblogger.Debugf("Number of returned result items: [%d]", len(nlbInfoList))

	return nlbInfoList, nil
}

/*
NLB 조회
*/
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

	cblogger.Debug("NLB Count : ", *response.Response.TotalCount)
	if *response.Response.TotalCount < 1 {
		return irs.NLBInfo{}, errors.New("Notfound: '" + nlbIID.SystemId + "' NLB Not found")
	}

	nlbInfo, nlbErr := NLBHandler.ExtractNLBDescribeInfo(response.Response.LoadBalancerSet[0])
	if nlbErr != nil {
		return irs.NLBInfo{}, nlbErr
	}
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

	return irs.ListenerInfo{}, errors.New("TENCENT_CANNOT_CHANGE_LISTENER")
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
	cblogger.Info("%s", targetResponse.ToJsonString())

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
	cblogger.Info("%s", targetResponse.ToJsonString())

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
	listenerProtocol := *response.Response.Listeners[0].Protocol
	healthCheckerProtocol := healthChecker.Protocol
	healthCheckerTimeOut := int64(healthChecker.Timeout)
	healthCheckerInterval := int64(healthChecker.Interval)
	healthCheckerThreshold := int64(healthChecker.Threshold)
	if healthCheckerTimeOut > 0 && healthCheckerInterval > 0 && healthCheckerTimeOut > healthCheckerInterval {
		return irs.HealthCheckerInfo{}, errors.New("HealthCheck.IntervalTime should not be less than HealthCheck.TimeOut")
	}

	changeHealthCheckerRequest := clb.NewModifyListenerRequest()

	changeHealthCheckerRequest.HealthCheck = &clb.HealthCheck{}
	changeHealthCheckerRequest.LoadBalancerId = common.StringPtr(newNLBId)
	changeHealthCheckerRequest.ListenerId = common.StringPtr(newListenerId)

	// health Checker port 값이 있을 때만 setting
	if !strings.EqualFold(healthChecker.Port, "") {
		healthPort, healthErr := strconv.ParseInt(healthChecker.Port, 10, 64)
		if healthErr != nil {
			return irs.HealthCheckerInfo{}, healthErr
		}
		changeHealthCheckerRequest.HealthCheck.CheckPort = common.Int64Ptr(healthPort)
	}

	if healthCheckerTimeOut > 0 {
		changeHealthCheckerRequest.HealthCheck.TimeOut = common.Int64Ptr(healthCheckerTimeOut)
	}
	if healthCheckerInterval > 0 {
		changeHealthCheckerRequest.HealthCheck.IntervalTime = common.Int64Ptr(healthCheckerInterval)
	}
	if healthCheckerThreshold > 0 {
		changeHealthCheckerRequest.HealthCheck.HealthNum = common.Int64Ptr(healthCheckerThreshold)
	}
	if !strings.EqualFold(healthCheckerProtocol, "") {
		changeHealthCheckerRequest.HealthCheck.CheckType = common.StringPtr(healthCheckerProtocol)
	}

	// Listener의 protocol이 UDP일 때 HealthChecker의 CheckType, ContextType은 고정
	if strings.EqualFold(listenerProtocol, "UDP") {
		changeHealthCheckerRequest.HealthCheck.CheckType = common.StringPtr("CUSTOM")
		changeHealthCheckerRequest.HealthCheck.ContextType = common.StringPtr("TEXT")
	}

	// HealthChecker protocol이 HTTP일 때 Domain,Version Set
	if strings.EqualFold(healthCheckerProtocol, "HTTP") {
		changeHealthCheckerRequest.HealthCheck.HttpCheckDomain = common.StringPtr("")
		changeHealthCheckerRequest.HealthCheck.HttpVersion = common.StringPtr("HTTP/1.1")
	}

	changeHealthCheckerResponse, err := NLBHandler.Client.ModifyListener(changeHealthCheckerRequest)
	if err != nil {
		return irs.HealthCheckerInfo{}, err
	}

	callogger.Info(call.String(callLogInfo))

	// Listener 변경을 기다림
	changeStatus, changeStatErr := NLBHandler.WaitForDone(*changeHealthCheckerResponse.Response.RequestId)
	if changeStatErr != nil {
		return irs.HealthCheckerInfo{}, changeStatErr
	}
	cblogger.Debug(changeStatus)

	healthCheckerResult, healthErr := NLBHandler.ExtractHealthCheckerInfo(nlbIID)
	if healthErr != nil {
		return irs.HealthCheckerInfo{}, healthErr
	}

	return healthCheckerResult, nil

}

// CLB instance status (creating, running)
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
			cblogger.Infof("===>The NLB state is [%d] so it stops waiting.", curStatus)
			break
		}

		curRetryCnt++
		cblogger.Infof("NLB status is not [%s] so I'm checking the climate for a second.", waitStatus)
		time.Sleep(time.Second * 1)
		if curRetryCnt > maxRetryCnt {
			cblogger.Errorf("The NLB Status value does not change to [%s] even after waiting for a long time (%d seconds), so it is forced to stop.", maxRetryCnt, waitStatus)
			return "Failed", errors.New("I waited for a long time, but the generated NLB status did not change to [" + waitStatus + "] so I will stop.")
		}
	}

	return waitStatus, nil
}

// Current status of a task (succeeded==Done, failed, in progress)
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
			cblogger.Infof("===>The request state is [%s] and will stop waiting.", waitStatus)
			break
		}

		curRetryCnt++
		cblogger.Infof("The request status is not [%s], so I'm checking the climate for a second.", waitStatus)
		time.Sleep(time.Second * 1)
		if curRetryCnt > maxRetryCnt {
			cblogger.Errorf("Waiting for a long time (%d seconds) does not change the request status value to [%s] and forces it to stop.", maxRetryCnt, waitStatus)
			return "Failed", errors.New("I waited for a long time, but the generated request status did not change to [" + waitStatus + "] so I will stop.")
		}
	}

	return waitStatus, nil
}

/*
nlb가 존재하는지 check
동일이름이 없으면 false, 있으면 true
*/
func (NLBHandler *TencentNLBHandler) nlbExist(chkName string) (bool, error) {
	cblogger.Debugf("chkName : %s", chkName)

	request := clb.NewDescribeLoadBalancersRequest()
	request.LoadBalancerName = common.StringPtr(chkName)

	response, err := NLBHandler.Client.DescribeLoadBalancers(request)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	if *response.Response.TotalCount < 1 {
		return false, nil
	}

	cblogger.Infof("NLB 정보 찾음 - NLBId:[%s] / NLBName:[%s]", *response.Response.LoadBalancerSet[0].LoadBalancerId, *response.Response.LoadBalancerSet[0].LoadBalancerName)
	return true, nil
}

/*
조회한 결과에서 Spider의 NLBInfo 값으로 변환
*/
func (NLBHandler *TencentNLBHandler) ExtractNLBDescribeInfo(nlbInfo *clb.LoadBalancer) (irs.NLBInfo, error) {

	// vpc name 구하기
	VPCHandler := TencentVPCHandler{
		Region: NLBHandler.Region,
		Client: NLBHandler.VpcClient,
	}
	cblogger.Debug(VPCHandler)

	retVpcInfo, errVpcInfo := VPCHandler.GetVPC(irs.IID{SystemId: *nlbInfo.VpcId})
	if errVpcInfo != nil {
		cblogger.Error(errVpcInfo)
		return irs.NLBInfo{}, errVpcInfo
	}

	cblogger.Debug(retVpcInfo)

	vpcId := retVpcInfo.IId.SystemId
	vpcName := retVpcInfo.IId.NameId

	// 생성 시간 정보 포맷 변환
	createTime, _ := time.Parse("2006-01-02 15:04:05", *nlbInfo.CreateTime)

	// NLB Type을 NLBInfo에 맞는 값으로 변환
	nlbType := ""
	if strings.EqualFold(*nlbInfo.LoadBalancerType, Tencent_LoadBalancerType_Open) {
		nlbType = Spider_LoadBalancerType_PUBLIC
	} else {
		nlbType = Spider_LoadBalancerType_INTERNAL
	}

	// NLBInfo 채워넣기
	resNLBInfo := irs.NLBInfo{

		IId:         irs.IID{SystemId: *nlbInfo.LoadBalancerId, NameId: *nlbInfo.LoadBalancerName},
		VpcIID:      irs.IID{SystemId: vpcId, NameId: vpcName},
		CreatedTime: createTime,
		Type:        nlbType,
		Scope:       "REGION",
	}

	return resNLBInfo, nil
}

/*
NLB Name으로 Listener를 조회하여 NLBInfo.Listener 값으로 변환
NLB 를 조회하여 Listener에 사용할 IP인 VIP 추출
*/
func (NLBHandler *TencentNLBHandler) ExtractListenerInfo(nlbIID irs.IID) (irs.ListenerInfo, error) {
	cblogger.Info("NLB IID : ", nlbIID.SystemId)

	// Listener정보 조회
	request := clb.NewDescribeListenersRequest()
	request.LoadBalancerId = common.StringPtr(nlbIID.SystemId)
	response, err := NLBHandler.Client.DescribeListeners(request)
	if err != nil {
		cblogger.Errorf("An API error has returned: %s", err.Error())
		return irs.ListenerInfo{}, err
	}

	// protocol port set
	resListenerInfo := irs.ListenerInfo{
		Protocol: *response.Response.Listeners[0].Protocol,
		Port:     strconv.FormatInt(*response.Response.Listeners[0].Port, 10),
	}

	// vip 정보 조회 : listener IP
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

/*
VM Group 정보 조회
*/
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

	// https://intl.cloud.tencent.com/ko/document/product/214/6151
	// If you use a layer-4 listener (i.e., layer-4 protocol forwarding),
	// the CLB instance will establish a TCP connection with the real server on the listening port,
	// and directly forward requests to the real server.
	// TCP, UDP Listener일 때 Real Server Protocol을 TCP로 설정
	resVmInfo := irs.VMGroupInfo{
		Protocol: Protocol_TCP,
		Port:     strconv.FormatInt(*response.Response.Listeners[0].Targets[0].Port, 10),
	}

	vms := []irs.IID{}
	for _, target := range response.Response.Listeners[0].Targets {
		vms = append(vms, irs.IID{SystemId: *target.InstanceId, NameId: *target.InstanceName})
	}

	resVmInfo.VMs = &vms

	return resVmInfo, nil
}

/*
Health Checker 정보 조회
*/
func (NLBHandler *TencentNLBHandler) ExtractHealthCheckerInfo(nlbIID irs.IID) (irs.HealthCheckerInfo, error) {
	cblogger.Info("NLB IID : ", nlbIID.SystemId)

	request := clb.NewDescribeListenersRequest()
	request.LoadBalancerId = common.StringPtr(nlbIID.SystemId)
	response, err := NLBHandler.Client.DescribeListeners(request)

	if err != nil {
		cblogger.Errorf("An API error has returned: %s", err.Error())
		return irs.HealthCheckerInfo{}, err
	}

	cblogger.Debug(response.ToJsonString())

	tencentHealthCheck := response.Response.Listeners[0].HealthCheck

	resHealthCheckerInfo := irs.HealthCheckerInfo{
		Protocol:  *tencentHealthCheck.CheckType,
		Interval:  int(*tencentHealthCheck.IntervalTime),
		Timeout:   int(*tencentHealthCheck.TimeOut),
		Threshold: int(*tencentHealthCheck.HealthNum),
	}

	// checkPort는 ommitEmpty로 값이 없으면 안들어 옴
	if tencentHealthCheck.CheckPort != nil {
		resHealthCheckerInfo.Port = strconv.FormatInt(*tencentHealthCheck.CheckPort, 10)
	}
	cblogger.Debug("after")

	return resHealthCheckerInfo, nil

}
