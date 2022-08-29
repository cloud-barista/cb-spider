package resources

import (
	"encoding/json"
	"errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/slb"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"

	"strconv"
	"strings"
	"time"
)

type AlibabaNLBHandler struct {
	Region idrv.RegionInfo
	//Client *ecs.Client
	Client    *slb.Client
	VMClient  *ecs.Client
	VpcClient *vpc.Client
}

type AlibabaNLBBackendServer struct {
	ServerId    string
	Port        int
	Weight      int
	Description string
	Type        string
	ServerIp    string
}

const (
	ListenerProtocol_TCP string = "tcp"
	ListenerProtocol_UDP string = "udp"

	BackendServerType_ECS       string = "ecs"
	LoadBalancerSpec_SLBS1SMALL string = "slb.s1.small"

	ALI_LoadBalancerAddressType_INTERNET = "internet"
	ALI_LoadBalancerAddressType_INTRANET = "intranet"
	SPIDER_LoadBalancerType_PUBLIC       = "PUBLIC"
	SPIDER_LoadBalancerType_PRIVATE      = "PRIVATE"

	SCOPE_REGION = "REGION"
	SCOPE_GLOBAL = "GLOBAL"

	ServerHealthStatus_NORMAL      string = "normal"
	ServerHealthStatus_ABNORMAL    string = "abnormal"
	ServerHealthStatus_unavailable string = "unavailable"
)

/*
https://www.alibabacloud.com/help/en/server-load-balancer/latest/createloadbalancer-2

	같은이름의 NLB생성가능. ID가 다름.
*/
func (NLBHandler *AlibabaNLBHandler) CreateNLB(nlbReqInfo irs.NLBInfo) (irs.NLBInfo, error) {
	// validation Check
	//// validation check area
	err := NLBHandler.validateCreateNLB(nlbReqInfo)
	if err != nil {
		cblogger.Info("validateCreateNLB ", err)
		return irs.NLBInfo{}, err
	}

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: nlbReqInfo.IId.NameId,
		CloudOSAPI:   "CreateLoadBalancer()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	// loadbalancer 생성

	// add Listener + health checker

	// add vms

	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	loadBalancerRequest := slb.CreateCreateLoadBalancerRequest()
	loadBalancerRequest.LoadBalancerName = nlbReqInfo.IId.NameId

	if strings.EqualFold(nlbReqInfo.Type, SPIDER_LoadBalancerType_PUBLIC) { // ali : internet/intranet spider : PUBLIC/PRIVATE
		loadBalancerRequest.AddressType = ALI_LoadBalancerAddressType_INTERNET
	} else {
		loadBalancerRequest.AddressType = ALI_LoadBalancerAddressType_INTRANET
	}

	// The maximum bandwidth of the listener. Unit: Mbit/s.
	// Valid values: 1 to 5120. For a pay-by-bandwidth Internet-facing CLB instance, you can specify the maximum bandwidth of each listener. The sum of maximum bandwidth of all listeners cannot exceed the maximum bandwidth of the CLB instance.
	//loadBalancerRequest.Bandwidth = 10

	loadBalancerRequest.VpcId = nlbReqInfo.VpcIID.SystemId

	//loadBalancerRequest.VSwitchId = ""

	// masterzone, slavezone not required
	//loadBalancerRequest.MasterZoneId = NLBHandler.Region.Zone
	//loadBalancerRequest.SlaveZoneId

	//loadBalancerRequest.InternetChargeType = "" // paybytraffic (default): pay-by-data-transfer , paybybandwidth: pay-by-bandwidth
	loadBalancerRequest.PayType = "PayOnDemand"
	// slb.s1.small, slb.s2.small, slb.s2.medium, slb.s3.small, slb.s3.medium, slb.s3.large
	// If InstanceChargeType is set to PayByCLCU, the LoadBalancerSpec parameter is invalid and you do not need to set this parameter.
	loadBalancerRequest.LoadBalancerSpec = "slb.s1.small" // required

	response, err := NLBHandler.Client.CreateLoadBalancer(loadBalancerRequest)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		return irs.NLBInfo{}, err
	}
	cblogger.Info(response)

	nlbIID := irs.IID{NameId: nlbReqInfo.IId.NameId, SystemId: response.LoadBalancerId}

	////// add Listener /////
	nlbListener, err := NLBHandler.AddLoadBalancerListener(nlbIID, nlbReqInfo)
	if err != nil {
		// 자원 회수
		deleteRequest := slb.CreateDeleteLoadBalancerRequest()
		deleteRequest.LoadBalancerId = response.LoadBalancerId
		_, delerr := NLBHandler.Client.DeleteLoadBalancer(deleteRequest)
		if delerr != nil {
			callLogInfo.ErrorMSG = delerr.Error()
			callogger.Info(call.String(callLogInfo))

			return irs.NLBInfo{}, errors.New(err.Error() + " recalled of resource " + delerr.Error())
		}
		return irs.NLBInfo{}, errors.New(err.Error() + " recalled of resource ")
	}
	// Listener 생성 후에는 start를 시켜줘야 Active가 됨. 안하면 Inactive상태
	cblogger.Debug(nlbListener)

	////// add VM Group /////
	nlbVmGroup, vmGroupErr := NLBHandler.addBackendServer(nlbIID, nlbReqInfo)
	//nlbVmGroup, err := NLBHandler.addVMGroupInfo(nlbIID, nlbReqInfo)// VServerGroup 으로 만들 때
	if vmGroupErr != nil {
		// 자원 회수 : Listener
		delListenerResult, delListenerErr := NLBHandler.deleteLoadBalancerListener(nlbIID, nlbReqInfo)
		if delListenerErr != nil {
			cblogger.Info("deleteLoadBalancerListener err ", delListenerErr)
		}
		cblogger.Info("deleteLoadBalancerListener result ", delListenerResult)

		// 자원 회수 : LB 껍데기
		deleteRequest := slb.CreateDeleteLoadBalancerRequest()
		deleteRequest.LoadBalancerId = response.LoadBalancerId
		_, delLBerr := NLBHandler.Client.DeleteLoadBalancer(deleteRequest)
		if delLBerr != nil {
			callLogInfo.ErrorMSG = delLBerr.Error()
			callogger.Info(call.String(callLogInfo))

			return irs.NLBInfo{}, errors.New(" recalled of resource " + delLBerr.Error())
		}
		return irs.NLBInfo{}, errors.New(" recalled of resource " + vmGroupErr.Error())
	}
	cblogger.Debug(nlbVmGroup)

	//nlbReqInfo.IId.SystemId = response.LoadBalancerId
	//returnNlbInfo := irs.NLBInfo{}
	//returnNlbInfo.IId.SystemId = response.LoadBalancerId
	returnNlbInfo, err := NLBHandler.GetNLB(irs.IID{SystemId: response.LoadBalancerId})
	if err != nil {
		return irs.NLBInfo{}, errors.New("Load balancer created successfully. However, the inquiry failed for the following reasons:" + err.Error())
	}

	return returnNlbInfo, err
}

/*
	Load balancer 전체 목록 보기
*/
func (NLBHandler *AlibabaNLBHandler) ListNLB() ([]*irs.NLBInfo, error) {

	//DescribeLoadBalancers
	cblogger.Info("Start")

	request := slb.CreateDescribeLoadBalancersRequest()
	request.RegionId = NLBHandler.Region.Region

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "List()",
		CloudOSAPI:   "DescribeLoadBalancers()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := NLBHandler.Client.DescribeLoadBalancers(request)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	//spew.Dump(result)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		return nil, err
	}

	callogger.Info("result count ", result.TotalCount)
	callogger.Info(result)
	var nlbInfoList []*irs.NLBInfo
	for _, curLB := range result.LoadBalancers.LoadBalancer { // LB 목록 조회시 가져오는 값들이 많지 않음. 상세정보는 GetNL로로
		nlbInfo, nlbErr := NLBHandler.GetNLB(irs.IID{SystemId: curLB.LoadBalancerId})

		if nlbErr != nil {
			return nil, nlbErr
		}
		nlbInfoList = append(nlbInfoList, &nlbInfo)
	}

	cblogger.Debug(result)
	//spew.Dump(vpcInfoList)
	return nlbInfoList, nil
}
func (NLBHandler *AlibabaNLBHandler) GetNLB(nlbIID irs.IID) (irs.NLBInfo, error) {
	var nlbInfo irs.NLBInfo
	vmGroup := irs.VMGroupInfo{}
	healthChecker := irs.HealthCheckerInfo{}

	request := slb.CreateDescribeLoadBalancerAttributeRequest()
	request.LoadBalancerId = nlbIID.SystemId

	lbAttributeResponse, err := NLBHandler.Client.DescribeLoadBalancerAttribute(request)
	if err != nil {
		cblogger.Info(err.Error())
		return irs.NLBInfo{}, err
	}
	cblogger.Info(lbAttributeResponse)

	nlbIID.NameId = lbAttributeResponse.LoadBalancerName
	nlbInfo.IId = nlbIID

	if strings.EqualFold(ALI_LoadBalancerAddressType_INTERNET, lbAttributeResponse.AddressType) {
		nlbInfo.Type = SPIDER_LoadBalancerType_PUBLIC
	} else {
		nlbInfo.Type = SPIDER_LoadBalancerType_PRIVATE
	}

	nlbInfo.Scope = SCOPE_REGION

	// Listener 는 여려개가능하나 CB-SP에서 1개로 fixed.
	listener := irs.ListenerInfo{}
	listenerProtocolAndPortList := lbAttributeResponse.ListenerPortsAndProtocol.ListenerPortAndProtocol // 이중으로 되어 있음.
	cblogger.Info("listenerProtocolAndPortList")
	cblogger.Info(listenerProtocolAndPortList)
	for _, listenerProtocolAndPort := range listenerProtocolAndPortList {
		cblogger.Info(listenerProtocolAndPort)
		listener.Protocol = listenerProtocolAndPort.ListenerProtocol
		listener.IP = lbAttributeResponse.Address
		listener.Port = strconv.Itoa(listenerProtocolAndPort.ListenerPort)
		//DNSName		string	// Optional, Auto Generated and attached
		listener.CspID = lbAttributeResponse.ResourceGroupId
		//KeyValueList []KeyValue
		break
	}
	nlbInfo.Listener = listener

	// 상세 listener 추가 정보 : listener가 여려개면 for문 안으로 넣어야 함. protocol에 따라 가져오는 상세정보가 다름
	if strings.EqualFold(listener.Protocol, ListenerProtocol_TCP) {
		responseNlbInfo, err := NLBHandler.describeLoadBalancerTcpListenerAttribute(nlbIID, listener)
		if err != nil {
			return irs.NLBInfo{}, err
		}

		vmGroup = responseNlbInfo.VMGroup
		healthChecker = responseNlbInfo.HealthChecker
	} else if strings.EqualFold(listener.Protocol, ListenerProtocol_UDP) {
		responseNlbInfo, err := NLBHandler.describeLoadBalancerUdpListenerAttribute(nlbIID, listener)
		if err != nil {
			return irs.NLBInfo{}, err
		}

		vmGroup = responseNlbInfo.VMGroup
		healthChecker = responseNlbInfo.HealthChecker
	}

	var vms []irs.IID
	backendServerList := lbAttributeResponse.BackendServers.BackendServer
	for _, backendServer := range backendServerList {
		// vm 이름 때문에 조회 해야하나...
		vmHandler := AlibabaVMHandler{Client: NLBHandler.VMClient}
		vmIID := irs.IID{SystemId: backendServer.ServerId}
		vmInfo, err := vmHandler.GetVM(vmIID)
		if err != nil {
			cblogger.Info(err.Error())
			// vm 정보 조회 실패
			var inKeyValueList []irs.KeyValue
			keyValue := irs.KeyValue{"reason", err.Error()}
			inKeyValueList = append(inKeyValueList, keyValue)
			vmGroup.KeyValueList = inKeyValueList
		} else {
			vmIID = vmInfo.IId
			nlbInfo.VpcIID = vmInfo.VpcIID
		}

		vms = append(vms, vmIID)
	}

	//VpcIID : vm 조회하면서 set함.
	//vpcHandler := AlibabaVPCHandler{Client: NLBHandler.VpcClient}
	//vpcInfo, err := vpcHandler.GetVPC(nlbInfo.VpcIID)
	//if err != nil {
	//
	//}
	//nlbInfo.VpcIID = vpcInfo.IId

	// VMGroup
	vmGroup.VMs = &vms
	nlbInfo.VMGroup = vmGroup

	// Health checker
	nlbInfo.HealthChecker = healthChecker

	createdTime, _ := time.Parse(
		time.RFC3339,
		lbAttributeResponse.CreateTime) // RFC3339형태이므로 해당 시간으로 다시 생성. "CreateTime": "2022-07-05T07:54:37Z",
	nlbInfo.CreatedTime = createdTime

	return nlbInfo, nil
}

/*
	NLB 삭제
	After you delete an SLB instance, the listeners and tags added to the SLB instance are deleted.
*/
func (NLBHandler *AlibabaNLBHandler) DeleteNLB(nlbIID irs.IID) (bool, error) {
	request := slb.CreateDeleteLoadBalancerRequest()

	request.LoadBalancerId = nlbIID.SystemId

	response, err := NLBHandler.Client.DeleteLoadBalancer(request)
	if err != nil {
		cblogger.Info(err.Error())
		return false, err
	}
	cblogger.Info(response)
	return true, nil
}

//------ Frontend Control

/*
	Spider에서 원하는 listener의 변경은 protocol, ip, port 의 변경아나
	ALIBABA에서는 listener의 key 가 loadbalancerId, port 이므로 실제로 변경할 수 있는 parameter가 없음.

	수정 가능한 항목은 healthcheck 부분으로 현재버전에서는 error로 return
	향후 필요시 삭제 후 추가 하는 방법 고려.
*/
func (NLBHandler *AlibabaNLBHandler) ChangeListener(nlbIID irs.IID, listener irs.ListenerInfo) (irs.ListenerInfo, error) {
	return irs.ListenerInfo{}, errors.New("ALIBABA_CANNOT_CHANGE_LISTENER")
}

//------ Backend Control
func (NLBHandler *AlibabaNLBHandler) ChangeVMGroupInfo(nlbIID irs.IID, vmGroup irs.VMGroupInfo) (irs.VMGroupInfo, error) {
	return irs.VMGroupInfo{}, errors.New("ALIBABA_CANNOT_CHANGE_VMGROUP")
}

/*
	loadBalancer에 VM추가
	vm만 추가
*/
func (NLBHandler *AlibabaNLBHandler) AddVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (irs.VMGroupInfo, error) {
	nlbReqInfo := irs.NLBInfo{}
	vmGroup := irs.VMGroupInfo{}
	vmGroup.VMs = vmIIDs

	nlbReqInfo.VMGroup = vmGroup
	returnVmGroup, err := NLBHandler.addBackendServer(nlbIID, nlbReqInfo)

	return returnVmGroup, err
}

/*
	loadBalancer에 VM제거
	vm만 제거
*/
func (NLBHandler *AlibabaNLBHandler) RemoveVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {
	nlbReqInfo := irs.NLBInfo{}
	vmGroup := irs.VMGroupInfo{}
	vmGroup.VMs = vmIIDs

	nlbReqInfo.VMGroup = vmGroup
	result, err := NLBHandler.removeBackendServer(nlbIID, nlbReqInfo)
	return result, err
}

func (NLBHandler *AlibabaNLBHandler) GetVMGroupHealthInfo(nlbIID irs.IID) (irs.HealthInfo, error) {
	returnHealthInfo := irs.HealthInfo{}
	allVMs := []irs.IID{}
	healthyVMs := []irs.IID{}
	unHealthyVMs := []irs.IID{}
	//AllVMs       *[]IID nameId, systemId
	//HealthyVMs   *[]IID
	//UnHealthyVMs *[]IID

	request := slb.CreateDescribeHealthStatusRequest()
	request.LoadBalancerId = nlbIID.SystemId

	response, err := NLBHandler.Client.DescribeHealthStatus(request)
	if err != nil {
		cblogger.Info(err.Error())
		return returnHealthInfo, err
	}
	cblogger.Info(response)

	for _, backendServer := range response.BackendServers.BackendServer {
		if strings.EqualFold(backendServer.ServerHealthStatus, ServerHealthStatus_NORMAL) {
			healthyVMs = append(healthyVMs, irs.IID{SystemId: backendServer.ServerId})
		} else { // abnomal or unavailable
			unHealthyVMs = append(unHealthyVMs, irs.IID{SystemId: backendServer.ServerId})
		}
		allVMs = append(allVMs, irs.IID{SystemId: backendServer.ServerId})
	}

	returnHealthInfo.HealthyVMs = &healthyVMs
	returnHealthInfo.UnHealthyVMs = &unHealthyVMs
	returnHealthInfo.AllVMs = &allVMs
	printToJson(returnHealthInfo)
	return returnHealthInfo, nil
}

/*
	HealthChecker 의 정보가 실제로는 listener에 들어있음.
	따라서 nblId, port 에 해당하는 listener를 찾고
	nlbInfo에 모든정보를 set(lb ID, listener protocol,port, healthchecker info)하여 healthchecker정보를 수정

*/
func (NLBHandler *AlibabaNLBHandler) ChangeHealthCheckerInfo(nlbIID irs.IID, healthChecker irs.HealthCheckerInfo) (irs.HealthCheckerInfo, error) {
	returnHealthChecker := irs.HealthCheckerInfo{}

	// loadbalancer 조회
	request := slb.CreateDescribeLoadBalancerAttributeRequest()
	request.LoadBalancerId = nlbIID.SystemId

	lbAttributeResponse, err := NLBHandler.Client.DescribeLoadBalancerAttribute(request)
	if err != nil {
		cblogger.Info(err.Error())
		return returnHealthChecker, err
	}
	cblogger.Info(lbAttributeResponse)

	// listener 정보 추출
	listener := irs.ListenerInfo{}
	listenerProtocolAndPortList := lbAttributeResponse.ListenerPortsAndProtocol.ListenerPortAndProtocol // 이중으로 되어 있음.
	for _, listenerProtocolAndPort := range listenerProtocolAndPortList {
		cblogger.Info(listenerProtocolAndPort)
		listener.Protocol = listenerProtocolAndPort.ListenerProtocol
		listener.IP = lbAttributeResponse.Address
		listener.Port = strconv.Itoa(listenerProtocolAndPort.ListenerPort)
		break
	}

	// health checker 수정할 정보 set
	nlbReqInfo := irs.NLBInfo{}
	nlbReqInfo.IId = nlbIID
	nlbReqInfo.Listener = listener
	nlbReqInfo.HealthChecker = healthChecker

	// listener 안의 healthchecker 정보 수정
	if strings.EqualFold(listener.Protocol, ListenerProtocol_TCP) {
		responseHealthChecker, err := NLBHandler.modifyLoadBalancerTcpHealthChecker(nlbIID, nlbReqInfo)
		if err != nil {
			return returnHealthChecker, err
		}
		returnHealthChecker = responseHealthChecker
	} else if strings.EqualFold(listener.Protocol, ListenerProtocol_UDP) {
		responseHealthChecker, err := NLBHandler.modifyLoadBalancerUdpHealthChecker(nlbIID, nlbReqInfo)
		if err != nil {
			return returnHealthChecker, err
		}
		returnHealthChecker = responseHealthChecker
	} else {
		return returnHealthChecker, errors.New("Invalid protocol " + listener.Protocol)
	}

	// healthchecker info return
	return returnHealthChecker, nil
}

//////////
/*
//리스너는 Client의 요청 및 입력 스트림을 수신하여 백엔드 영역의 VM그룹으로 전달한다.
//	하나의 NLB는 하나의 리스너를 포함하며, 수신 프로토콜, IP 및 수신 포트로 구성된다.
//	선택 가능한 수신 프로토콜은 TCP 및 UDP이며, IP는 CSP 또는 대상 Driver에서 자동 생성 및 관리된다.
//	수신 포트는 1-65535 범위의 값으로 설정이 가능하다.
//※ 리스너 IP에 매핑 되는 DNS-Name 지원: 추후 고려
//https://www.alibabacloud.com/help/en/server-load-balancer/latest/createloadbalancertcplistener

Newly created listeners are in the stopped state. After a listener is created, you must call the StartLoadBalancerListener operation to start the listener. This way, the listener can forward network traffic.

리스너를 생성하면 중단 상태임. startListener 호출하여 동작시켜야 함
*/
func (NLBHandler *AlibabaNLBHandler) AddLoadBalancerListener(nlbIID irs.IID, nlbReqInfo irs.NLBInfo) (irs.ListenerInfo, error) {
	listener := nlbReqInfo.Listener
	healthChecker := nlbReqInfo.HealthChecker

	listenerProtocol := listener.Protocol
	healthCheckerProtocol := healthChecker.Protocol
	if listenerProtocol != healthCheckerProtocol {
		return irs.ListenerInfo{}, errors.New("ALIBABA_HEALTHCHECK_PROTOCOL_NOT_SAME_LISTENER_PROTOCOL. " + listener.Protocol + ":" + healthCheckerProtocol)
	}

	var returnListener irs.ListenerInfo
	if listener.Protocol == "TCP" {

		responseListener, err := NLBHandler.addLoadBalancerTcpListener(nlbIID, nlbReqInfo)
		if err != nil {
			return irs.ListenerInfo{}, err
		}
		returnListener = responseListener
	} else if listener.Protocol == "UDP" {
		responseListener, err := NLBHandler.addLoadBalancerUdpListener(nlbIID, nlbReqInfo)
		if err != nil {
			return irs.ListenerInfo{}, err
		}
		returnListener = responseListener
	} else {
		return irs.ListenerInfo{}, errors.New("Invalid protocol " + listener.Protocol)
	}

	// Listener start
	responseListener, err := NLBHandler.startLoadBalancerListener(nlbIID, nlbReqInfo)
	if err != nil {
		return irs.ListenerInfo{}, err
	}
	returnListener = responseListener
	return returnListener, nil
}

/*
	중단된 Listener를 시작
	stop 상태인 경우에만 호출가능
	Listener를 생성하면 중단된 상태이므로 start를 시켜야 함.
*/
func (NLBHandler *AlibabaNLBHandler) startLoadBalancerListener(nlbIID irs.IID, nlbReqInfo irs.NLBInfo) (irs.ListenerInfo, error) {
	listener := nlbReqInfo.Listener

	listenerRequest := slb.CreateStartLoadBalancerListenerRequest()

	listenerRequest.LoadBalancerId = nlbIID.SystemId
	listenerRequest.ListenerProtocol = listener.Protocol
	listenerRequest.ListenerPort = requests.Integer(listener.Port)
	//listenerRequest.RegionId = NLBHandler.Region.Region

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "List()",
		CloudOSAPI:   "startLoadBalancerListener()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	response, err := NLBHandler.Client.StartLoadBalancerListener(listenerRequest)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		return irs.ListenerInfo{}, err
	}

	requestId := response.RequestId
	cblogger.Debug(response)
	cblogger.Debug(requestId)
	return irs.ListenerInfo{}, nil
}

/*
// Loadbalancer에서 사용할 TCP Listener 등록
	mandatory : loadBalancerId, Bandwidth, ListenerPort, RegionID
	BackendServerport : vm 추가방식은 필수. vServer Group 방식은 해당 VServerGroupId set
*/
func (NLBHandler *AlibabaNLBHandler) addLoadBalancerTcpListener(nlbIID irs.IID, nlbReqInfo irs.NLBInfo) (irs.ListenerInfo, error) {
	listener := nlbReqInfo.Listener
	healthChecker := nlbReqInfo.HealthChecker
	vmGroup := nlbReqInfo.VMGroup

	cblogger.Info(listener)
	listenerRequest := slb.CreateCreateLoadBalancerTCPListenerRequest()

	//// set listener area
	listenerRequest.LoadBalancerId = nlbIID.SystemId
	listenerRequest.Bandwidth = requests.NewInteger(-1) //For a pay-by-data-transfer Internet-facing Classic Load Balancer (CLB) instance, set the value to -1. This indicates that the maximum bandwidth value is unlimited.
	listenerRequest.ListenerPort = requests.Integer(listener.Port)
	listenerRequest.RegionId = NLBHandler.Region.Region // 일단은 동일 region으로 set.

	// The BackendServerPort or VServerGroupId is required at lease one
	listenerRequest.BackendServerPort = requests.Integer(vmGroup.Port) // 1 to 65535.	If the VServerGroupId parameter is not set, this parameter is required.

	//// set health checker area
	listenerRequest.HealthCheckType = "tcp"
	listenerRequest.HealthCheckConnectPort = requests.Integer(healthChecker.Port)
	listenerRequest.HealthyThreshold = requests.NewInteger(healthChecker.Threshold)
	listenerRequest.UnhealthyThreshold = requests.NewInteger(healthChecker.Threshold)
	listenerRequest.HealthCheckConnectTimeout = requests.NewInteger(healthChecker.Timeout)
	listenerRequest.HealthCheckInterval = requests.NewInteger(healthChecker.Interval)

	// 필수만 set
	////listenerRequest.Scheduler = "wrr"
	//listenerRequest.PersistenceTimeout = requests.Integer(500) // 0 to 3600.  0 : session persistence is disabled
	//listenerRequest.EstablishedTimeout = requests.Integer(500) // 100 to 900.
	//
	//listenerRequest.HealthyThreshold = requests.Integer(4)            // 2 to 10.
	//listenerRequest.UnhealthyThreshold = requests.Integer(4)          // 2 to 10.
	//listenerRequest.HealthCheckConnectTimeout = requests.Integer(100) // 1 to 300.
	//listenerRequest.HealthCheckConnectPort = requests.Integer(80)     // 1 to 65535. If this parameter is not set, the port specified by BackendServerPort is used for health checks.
	//listenerRequest.HealthCheckInterval = requests.Integer(3)         // 1 to 50
	//listenerRequest.HealthCheckDomain = listener.DNSName
	////listenerRequest.HealthCheckURI = ""
	//listenerRequest.HealthCheckHttpCode = "http_2xx" // http_2xx, http_3xx, http_4xx, http_5xx
	//listenerRequest.HealthCheckType = "tcp"          // tcp(default), http
	//
	//listenerRequest.VServerGroupId = ""
	////listenerRequest.MasterSlaveServerGroupId = "" //You cannot specify the vServer group ID and primary/secondary server group ID at the same time.
	//
	////listenerRequest.AclId = ""// access control list
	////listenerRequest.AclType = "black" // white/black
	////listenerRequest.AclStatus = "off" // on/off
	////listenerRequest.Description = ""
	//
	//listenerRequest.ConnectionDrain = "off" // on/off
	//listenerRequest.ConnectionDrainTimeout = requests.Integer(300)

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "List()",
		CloudOSAPI:   "DescribeLoadBalancers()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	cblogger.Info(listenerRequest)
	response, err := NLBHandler.Client.CreateLoadBalancerTCPListener(listenerRequest)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		return irs.ListenerInfo{}, err
	}
	cblogger.Debug(response)
	requestId := response.RequestId
	cblogger.Debug(response)
	cblogger.Debug(requestId)
	return irs.ListenerInfo{}, nil
}

func (NLBHandler *AlibabaNLBHandler) addLoadBalancerUdpListener(nlbIID irs.IID, nlbReqInfo irs.NLBInfo) (irs.ListenerInfo, error) {
	listener := nlbReqInfo.Listener
	healthChecker := nlbReqInfo.HealthChecker
	vmGroup := nlbReqInfo.VMGroup

	listenerRequest := slb.CreateCreateLoadBalancerUDPListenerRequest()

	//// set listener area
	listenerRequest.LoadBalancerId = nlbIID.SystemId
	listenerRequest.Bandwidth = requests.NewInteger(-1) //For a pay-by-data-transfer Internet-facing Classic Load Balancer (CLB) instance, set the value to -1. This indicates that the maximum bandwidth value is unlimited.
	listenerRequest.ListenerPort = requests.Integer(listener.Port)
	listenerRequest.RegionId = NLBHandler.Region.Region                // 일단은 동일 region으로 set.
	listenerRequest.BackendServerPort = requests.Integer(vmGroup.Port) // 1 to 65535.	If the VServerGroupId parameter is not set, this parameter is required.

	//listenerRequest.Scheduler = "wrr"

	listenerRequest.HealthCheckType = "udp"
	listenerRequest.HealthCheckConnectPort = requests.Integer(healthChecker.Port)          // 1 to 65535. If this parameter is not set, the port specified by BackendServerPort is used for health checks.
	listenerRequest.HealthyThreshold = requests.NewInteger(healthChecker.Threshold)        // 2 to 10.
	listenerRequest.UnhealthyThreshold = requests.NewInteger(healthChecker.Threshold)      // 2 to 10.
	listenerRequest.HealthCheckConnectTimeout = requests.NewInteger(healthChecker.Timeout) // 1 to 300.
	listenerRequest.HealthCheckInterval = requests.NewInteger(healthChecker.Interval)      // 1 to 50

	//listenerRequest.HealthCheckReq = "" // 1 to 64
	//listenerRequest.HealthCheckExp = "" // 1 to 64

	//listenerRequest.VServerGroupId = ""//
	//listenerRequest.MasterSlaveServerGroupId = "" //You cannot specify the vServer group ID and primary/secondary server group ID at the same time.

	//listenerRequest.AclId = ""// access control list
	//listenerRequest.AclType = "black" // white/black
	//listenerRequest.AclStatus = "off" // on/off
	//listenerRequest.Description = ""

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "List()",
		CloudOSAPI:   "AddLoadBalancerUdpListener()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	response, err := NLBHandler.Client.CreateLoadBalancerUDPListener(listenerRequest)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		return irs.ListenerInfo{}, err
	}

	requestId := response.RequestId
	cblogger.Debug(response)
	cblogger.Debug(requestId)
	return irs.ListenerInfo{}, nil
}

/*
// Listener 수정
	: Protocol, Port는 수정 불가, IP는 set하지 않음.
	즉, 현재 cb-spider에서 alibaba listener는 수정 불가

    Protocol     string
    IP           string
    Port         string
    DNSName      string
    CspID        string
    KeyValueList []KeyValue

	ps : sting to Integer 는 requests.Integer가 되나, int to Integer는 requests.NewInteger로
*/
func (NLBHandler *AlibabaNLBHandler) modifyLoadBalancerTcpListener(nlbIID irs.IID, nlbReqInfo irs.NLBInfo) (irs.ListenerInfo, error) {
	listener := nlbReqInfo.Listener
	cblogger.Info(listener)
	listenerRequest := slb.CreateSetLoadBalancerTCPListenerAttributeRequest()
	listenerRequest.LoadBalancerId = nlbIID.SystemId
	listenerRequest.Bandwidth = requests.NewInteger(-1) //For a pay-by-data-transfer Internet-facing Classic Load Balancer (CLB) instance, set the value to -1. This indicates that the maximum bandwidth value is unlimited.
	listenerRequest.ListenerPort = requests.Integer(listener.Port)
	listenerRequest.RegionId = NLBHandler.Region.Region // 일단은 동일 region으로 set.

	// 필수만 set
	////listenerRequest.Scheduler = "wrr"
	//listenerRequest.PersistenceTimeout = requests.Integer(500) // 0 to 3600.  0 : session persistence is disabled
	//listenerRequest.EstablishedTimeout = requests.Integer(500) // 100 to 900.
	//
	//listenerRequest.HealthyThreshold = requests.Integer(4)            // 2 to 10.
	//listenerRequest.UnhealthyThreshold = requests.Integer(4)          // 2 to 10.
	//listenerRequest.HealthCheckConnectTimeout = requests.Integer(100) // 1 to 300.
	//listenerRequest.HealthCheckConnectPort = requests.Integer(80)     // 1 to 65535. If this parameter is not set, the port specified by BackendServerPort is used for health checks.
	//listenerRequest.HealthCheckInterval = requests.Integer(3)         // 1 to 50
	//listenerRequest.HealthCheckDomain = listener.DNSName
	////listenerRequest.HealthCheckURI = ""
	//listenerRequest.HealthCheckHttpCode = "http_2xx" // http_2xx, http_3xx, http_4xx, http_5xx
	//listenerRequest.HealthCheckType = "tcp"          // tcp(default), http
	//
	//listenerRequest.VServerGroupId = ""
	////listenerRequest.MasterSlaveServerGroupId = "" //You cannot specify the vServer group ID and primary/secondary server group ID at the same time.
	//
	////listenerRequest.AclId = ""// access control list
	////listenerRequest.AclType = "black" // white/black
	////listenerRequest.AclStatus = "off" // on/off
	////listenerRequest.Description = ""
	//
	//listenerRequest.ConnectionDrain = "off" // on/off
	//listenerRequest.ConnectionDrainTimeout = requests.Integer(300)

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "List()",
		CloudOSAPI:   "DescribeLoadBalancers()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	cblogger.Info(listenerRequest)
	response, err := NLBHandler.Client.SetLoadBalancerTCPListenerAttribute(listenerRequest)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		return irs.ListenerInfo{}, err
	}
	cblogger.Debug(response)
	requestId := response.RequestId
	cblogger.Debug(response)
	cblogger.Debug(requestId)
	return irs.ListenerInfo{}, nil
}

/*
//modifyLoadBalancerUdpListener
	TCP Listener와 호출하는 객체와 function 이름만 다르고 다른 항목이 같음
	HTTP.. 등 다른 Listener도 추가될 수 있어 따로 뺌.
*/
func (NLBHandler *AlibabaNLBHandler) modifyLoadBalancerUdpListener(nlbIID irs.IID, nlbReqInfo irs.NLBInfo) (irs.ListenerInfo, error) {
	listener := nlbReqInfo.Listener
	cblogger.Info(listener)
	listenerRequest := slb.CreateSetLoadBalancerUDPListenerAttributeRequest()
	listenerRequest.LoadBalancerId = nlbIID.SystemId
	listenerRequest.Bandwidth = requests.NewInteger(-1) //For a pay-by-data-transfer Internet-facing Classic Load Balancer (CLB) instance, set the value to -1. This indicates that the maximum bandwidth value is unlimited.
	listenerRequest.ListenerPort = requests.Integer(listener.Port)
	listenerRequest.RegionId = NLBHandler.Region.Region // 일단은 동일 region으로 set.

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "List()",
		CloudOSAPI:   "DescribeLoadBalancers()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	cblogger.Info(listenerRequest)
	response, err := NLBHandler.Client.SetLoadBalancerUDPListenerAttribute(listenerRequest)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		return irs.ListenerInfo{}, err
	}
	cblogger.Debug(response)
	requestId := response.RequestId
	cblogger.Debug(response)
	cblogger.Debug(requestId)
	return irs.ListenerInfo{}, nil
}

/*
	listenerPort : required
	listenerProtocol : not required 이나, 동일한 포트를 쓰는 리스너가 여러개면 required
*/
func (NLBHandler *AlibabaNLBHandler) deleteLoadBalancerListener(nlbIID irs.IID, nlbReqInfo irs.NLBInfo) (bool, error) {
	listener := nlbReqInfo.Listener
	cblogger.Info(listener)
	listenerRequest := slb.CreateDeleteLoadBalancerListenerRequest()
	listenerRequest.LoadBalancerId = nlbIID.SystemId
	listenerRequest.ListenerProtocol = listener.Protocol
	listenerRequest.ListenerPort = requests.Integer(listener.Port)
	listenerRequest.RegionId = NLBHandler.Region.Region // 일단은 동일 region으로 set.

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "List()",
		CloudOSAPI:   "deleteLoadBalancers()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	cblogger.Info(listenerRequest)
	response, err := NLBHandler.Client.DeleteLoadBalancerListener(listenerRequest)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		return false, err
	}
	cblogger.Debug(response)
	requestId := response.RequestId
	cblogger.Debug(response)
	cblogger.Debug(requestId)
	return true, nil
}

/*
	현재는 Default ServerGroup 을 사용. VMGroup은 쓰지 않음.
	// CreateVServerGroup
Examples:
    ECS instance: [{ "ServerId": "i-xxxxxxxxx", "Weight": "100", "Type": "ecs", "Port": "80", "Description": "test-112" }].
    ENI: [{ "ServerId": "eni-xxxxxxxxx", "Weight": "100", "Type": "eni", "ServerIp": "192.168.**.**", "Port":"80","Description":"test-112" }]
    ENI with multiple IP addresses: [{ "ServerId": "eni-xxxxxxxxx", "Weight": "100", "Type": "eni", "ServerIp": "192.168.**.**", "Port":"80","Description":"test-112" },{ "ServerId": "eni-xxxxxxxxx", "Weight": "100", "Type": "eni", "ServerIp": "172.166.**.**", "Port":"80","Description":"test-113" }]

You can add at most 20 backend servers to a CLB instance in each request.
*/
func (NLBHandler *AlibabaNLBHandler) addVMGroupInfo(nlbIID irs.IID, nlbReqInfo irs.NLBInfo) (irs.VMGroupInfo, error) {
	vmGroup := nlbReqInfo.VMGroup

	vmGroupRequest := slb.CreateCreateVServerGroupRequest()
	vmGroupRequest.LoadBalancerId = nlbIID.SystemId
	vmGroupRequest.VServerGroupName = nlbIID.NameId // LB와 똑같은 이름 상관없음

	vmHandler := AlibabaVMHandler{Client: NLBHandler.VMClient}

	maxVmCount := 20
	vmCount := len(*vmGroup.VMs)
	if maxVmCount < vmCount {
		return irs.VMGroupInfo{}, errors.New("You can add at most 20 backend servers to a CLB instance in each request " + strconv.Itoa(vmCount))
	}

	remainingWeight := 100 // 전체 가중치
	weight := 100 / vmCount

	var vms []string
	for vmIndex, vmIId := range *vmGroup.VMs {

		backendServer := AlibabaNLBBackendServer{}
		backendServer.ServerId = vmIId.SystemId
		// vm 정보 조회해서 backendServer정보 set
		vmInfo, _ := vmHandler.GetVM(nlbIID)
		// IP
		backendServer.ServerIp = vmInfo.PublicIP

		// Port
		backendServer.Port, _ = strconv.Atoi(vmGroup.Port)

		// Weight
		if vmIndex == vmCount-1 {
			backendServer.Weight = remainingWeight
		} else {
			backendServer.Weight = weight
			remainingWeight -= weight
		}

		// type( ecs : elastic compute service instance / eni : elastic network interface )
		backendServer.Type = "ecs"
		backendServerJson, err := json.Marshal(backendServer)
		if err != nil {
			return irs.VMGroupInfo{}, err
		}
		vms = append(vms, string(backendServerJson))
	}
	vmGroupRequest.VServerGroupName = "[" + strings.Join(vms, ",") + "]" //
	// The value of this parameter must be a STRING list in the JSON format. You can specify up to 20 elements in each request.
	//vmGroupRequest.BackendServers

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "List()",
		CloudOSAPI:   "AddLoadBalancerUdpListener()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	response, err := NLBHandler.Client.CreateVServerGroup(vmGroupRequest)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		return irs.VMGroupInfo{}, err
	}

	requestId := response.RequestId
	cblogger.Debug(response)
	cblogger.Debug(requestId)

	vmGroup.CspID = response.VServerGroupId
	return vmGroup, nil
}

/*
// Default ServerGroup 으로 Vserver Group 없이 instance만 추가
Examples:
    ECS instance: [{ "ServerId": "i-xxxxxxxxx", "Weight": "100", "Type": "ecs", "Port":"80","Description":"test-112" }]
    ENI: [{ "ServerId": "eni-xxxxxxxxx", "Weight": "100", "Type": "eni", "ServerIp": "192.168.**.**", "Port":"80","Description":"test-112" }]
    ENI with multiple IP addresses: [{ "ServerId": "eni-xxxxxxxxx", "Weight": "100", "Type": "eni", "ServerIp": "192.168.**.**", "Port":"80","Description":"test-113" },{ "ServerId": "eni-xxxxxxxxx", "Weight": "100", "Type": "eni", "ServerIp": "172.166.**.**", "Port":"80","Description":"test-113" }]
    Elastic container instance: [{ "ServerId": "eci-xxxxxxxxx", "Weight": "100", "Type": "eci", "ServerIp": "192.168.**.**", "Port":"80","Description":"test-114" }]

*/
func (NLBHandler *AlibabaNLBHandler) addBackendServer(nlbIID irs.IID, nlbReqInfo irs.NLBInfo) (irs.VMGroupInfo, error) {
	vmGroup := nlbReqInfo.VMGroup

	backendServersRequest := slb.CreateAddBackendServersRequest()
	backendServersRequest.LoadBalancerId = nlbIID.SystemId

	vmHandler := AlibabaVMHandler{Client: NLBHandler.VMClient}

	maxVmCount := 20
	vmCount := len(*vmGroup.VMs)
	if maxVmCount < vmCount {
		return irs.VMGroupInfo{}, errors.New("You can add at most 20 backend servers to a CLB instance in each request " + strconv.Itoa(vmCount))
	}

	remainingWeight := 100 // 전체 가중치
	weight := 100 / vmCount

	var vms []string
	var returnVms []irs.IID
	for vmIndex, vmIId := range *vmGroup.VMs {

		backendServer := AlibabaNLBBackendServer{ServerId: vmIId.SystemId}

		// vm 정보 조회해서 backendServer정보 set
		vmInfo, err := vmHandler.GetVM(vmIId)
		if err != nil {
			return irs.VMGroupInfo{}, err
		}
		printToJson(vmInfo)
		// IP
		backendServer.ServerIp = vmInfo.PublicIP

		// Port
		backendServer.Port, _ = strconv.Atoi(vmGroup.Port)

		// Weight
		if vmIndex == vmCount-1 {
			backendServer.Weight = remainingWeight
		} else {
			backendServer.Weight = weight
			remainingWeight -= weight
		}

		// type( ecs : elastic compute service instance / eni : elastic network interface / eci : elastic container instance )
		backendServer.Type = "ecs"

		backendServerJson, err := json.Marshal(backendServer)
		if err != nil {
			return irs.VMGroupInfo{}, err
		}
		vms = append(vms, string(backendServerJson))

		// vmGroup 정보 갱신
		returnVms = append(returnVms, vmInfo.IId)
		printToJson(vmInfo.IId)
		printToJson(returnVms)
		//vmIId.NameId = vmInfo.IId.NameId
	}
	backendServersRequest.BackendServers = "[" + strings.Join(vms, ",") + "]"
	cblogger.Info("backendServersRequest---")
	cblogger.Info(backendServersRequest)
	// The value of this parameter must be a STRING list in the JSON format. You can specify up to 20 elements in each request.
	//vmGroupRequest.BackendServers

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "List()",
		CloudOSAPI:   "addBackendServer()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	response, err := NLBHandler.Client.AddBackendServers(backendServersRequest)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		return irs.VMGroupInfo{}, err
	}

	requestId := response.RequestId
	cblogger.Debug(response)
	cblogger.Debug(requestId)

	vmGroup.VMs = &returnVms
	printToJson(vmGroup)
	return vmGroup, nil
}

/*
	ex) 현재는 ecs만 사용
    Remove an ECS instance: [{"ServerId":"i-bp1fq61enf4loa5i****", "Type": "ecs","Weight":"100"}]
    Remove an ENI: [{"ServerId":"eni-2ze1sdp5****","Type": "eni","Weight":"100"}]
*/
func (NLBHandler *AlibabaNLBHandler) removeBackendServer(nlbIID irs.IID, nlbReqInfo irs.NLBInfo) (bool, error) {
	vmGroup := nlbReqInfo.VMGroup

	backendServersRequest := slb.CreateRemoveBackendServersRequest()
	backendServersRequest.LoadBalancerId = nlbIID.SystemId

	var vms []string
	for _, vmIId := range *vmGroup.VMs {
		backendServer := AlibabaNLBBackendServer{ServerId: vmIId.SystemId}

		// type( ecs : elastic compute service instance / eni : elastic network interface / eci : elastic container instance )
		backendServer.Type = "ecs" // 현재는 ecs로 fixed

		backendServerJson, err := json.Marshal(backendServer)
		if err != nil {
			return false, err
		}
		vms = append(vms, string(backendServerJson))

	}
	backendServersRequest.BackendServers = "[" + strings.Join(vms, ",") + "]"

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "List()",
		CloudOSAPI:   "addBackendServer()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	response, err := NLBHandler.Client.RemoveBackendServers(backendServersRequest)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		return false, err
	}

	//LoadBalancerId string                               `json:"LoadBalancerId" xml:"LoadBalancerId"`
	//RequestId      string                               `json:"RequestId" xml:"RequestId"`
	//BackendServers BackendServersInRemoveBackendServers `json:"BackendServers" xml:"BackendServers"
	printToJson(response)
	return true, nil
}

/*
	실제로는 Listener 수정이나, 항목이 healthchecker 부분이라 modify health checker라고 함
	TCP는 protocol 변경 가능( http, tcp 중 택1), UDP는 protocol변경 불가능
*/
func (NLBHandler *AlibabaNLBHandler) modifyLoadBalancerTcpHealthChecker(nlbIID irs.IID, nlbReqInfo irs.NLBInfo) (irs.HealthCheckerInfo, error) {
	listener := nlbReqInfo.Listener
	healthChecker := nlbReqInfo.HealthChecker
	cblogger.Info(listener)
	listenerRequest := slb.CreateSetLoadBalancerTCPListenerAttributeRequest()

	// required key
	listenerRequest.LoadBalancerId = nlbIID.SystemId
	listenerRequest.ListenerPort = requests.Integer(listener.Port)

	// healthchecker에 해당하는 값들만 set
	if !strings.EqualFold(healthChecker.Protocol, "") { // healthchecker port변경 가능
		if strings.EqualFold(healthChecker.Protocol, "tcp") || strings.EqualFold(healthChecker.Protocol, "http") {
			listenerRequest.HealthCheckType = healthChecker.Protocol
		} else {
			return irs.HealthCheckerInfo{}, errors.New("The appropriate value for the healthcheck protocol is tcp and http ")
		}
	}
	if !strings.EqualFold(healthChecker.Port, "") { // healthchecker port변경 가능
		listenerRequest.HealthCheckConnectPort = requests.Integer(healthChecker.Port)
	}
	if healthChecker.Threshold >= 2 && healthChecker.Threshold <= 10 {
		listenerRequest.HealthyThreshold = requests.NewInteger(healthChecker.Threshold)
		listenerRequest.UnhealthyThreshold = requests.NewInteger(healthChecker.Threshold)
	} else {
		return irs.HealthCheckerInfo{}, errors.New("The appropriate value for the threshold is 2 to 10")
	}
	if healthChecker.Timeout >= 1 && healthChecker.Timeout <= 300 {
		listenerRequest.HealthCheckConnectTimeout = requests.NewInteger(healthChecker.Timeout)
	} else {
		return irs.HealthCheckerInfo{}, errors.New("The appropriate value for the timeout is 1 to 300")
	}
	if healthChecker.Interval >= 1 && healthChecker.Interval <= 50 {
		listenerRequest.HealthCheckInterval = requests.NewInteger(healthChecker.Interval)
	} else {
		return irs.HealthCheckerInfo{}, errors.New("The appropriate value for the interval is 1 to 50")
	}

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "List()",
		CloudOSAPI:   "SetLoadBalancerTCPListenerAttribute()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	printToJson(listenerRequest)
	response, err := NLBHandler.Client.SetLoadBalancerTCPListenerAttribute(listenerRequest)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		return irs.HealthCheckerInfo{}, err
	}
	cblogger.Debug(response)
	requestId := response.RequestId
	cblogger.Debug(response)
	cblogger.Debug(requestId)

	responseNlbInfo, err := NLBHandler.describeLoadBalancerTcpListenerAttribute(nlbIID, listener)
	if err != nil {
		return irs.HealthCheckerInfo{}, err
	}
	return responseNlbInfo.HealthChecker, nil
}

/*
	실제로는 Listener 수정이나, 항목이 healthchecker 부분이라 modify health checker라고 함.
	UDP는 Protocol변경 불가. port만 변경 가능
*/
func (NLBHandler *AlibabaNLBHandler) modifyLoadBalancerUdpHealthChecker(nlbIID irs.IID, nlbReqInfo irs.NLBInfo) (irs.HealthCheckerInfo, error) {
	listener := nlbReqInfo.Listener
	healthChecker := nlbReqInfo.HealthChecker
	cblogger.Info(listener)
	listenerRequest := slb.CreateSetLoadBalancerUDPListenerAttributeRequest()

	// required key
	listenerRequest.LoadBalancerId = nlbIID.SystemId
	listenerRequest.ListenerPort = requests.Integer(listener.Port)

	// healthchecker에 해당하는 값들만 set
	if !strings.EqualFold(healthChecker.Port, "") { // healthchecker port변경 가능
		listenerRequest.HealthCheckConnectPort = requests.Integer(healthChecker.Port)
	}
	if healthChecker.Threshold >= 2 && healthChecker.Threshold <= 10 {
		listenerRequest.HealthyThreshold = requests.NewInteger(healthChecker.Threshold)
		listenerRequest.UnhealthyThreshold = requests.NewInteger(healthChecker.Threshold)
	} else {
		return irs.HealthCheckerInfo{}, errors.New("The appropriate value for the threshold is 2 to 10")
	}
	if healthChecker.Timeout >= 1 && healthChecker.Timeout <= 300 {
		listenerRequest.HealthCheckConnectTimeout = requests.NewInteger(healthChecker.Timeout)
	} else {
		return irs.HealthCheckerInfo{}, errors.New("The appropriate value for the timeout is 1 to 300")
	}
	if healthChecker.Interval >= 1 && healthChecker.Interval <= 50 {
		listenerRequest.HealthCheckInterval = requests.NewInteger(healthChecker.Interval)
	} else {
		return irs.HealthCheckerInfo{}, errors.New("The appropriate value for the interval is 1 to 50")
	}

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.ALIBABA,
		RegionZone:   NLBHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "List()",
		CloudOSAPI:   "SetLoadBalancerUDPListenerAttribute()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	printToJson(listenerRequest)
	response, err := NLBHandler.Client.SetLoadBalancerUDPListenerAttribute(listenerRequest)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		return irs.HealthCheckerInfo{}, err
	}
	cblogger.Debug(response)
	requestId := response.RequestId
	cblogger.Debug(response)
	cblogger.Debug(requestId)

	responseNlbInfo, err := NLBHandler.describeLoadBalancerUdpListenerAttribute(nlbIID, listener)
	if err != nil {
		return irs.HealthCheckerInfo{}, err
	}
	return responseNlbInfo.HealthChecker, nil
}

/*
	TCPListener를 조회하여 CB-Spider의 객체에 맞게 set하여 return
	호출하는 곳에서 원하는 값을 추출하여 사용

*/
func (NLBHandler *AlibabaNLBHandler) describeLoadBalancerTcpListenerAttribute(nlbIID irs.IID, listener irs.ListenerInfo) (irs.NLBInfo, error) {
	nlbInfo := irs.NLBInfo{}
	vmGroup := irs.VMGroupInfo{}
	healthChecker := irs.HealthCheckerInfo{}

	// 조회
	listenerAttributeRequest := slb.CreateDescribeLoadBalancerTCPListenerAttributeRequest()
	listenerAttributeRequest.LoadBalancerId = nlbIID.SystemId
	listenerAttributeRequest.ListenerPort = requests.Integer(listener.Port)
	printToJson(listener)
	printToJson(listenerAttributeRequest)
	listenerAttributeResponse, err := NLBHandler.Client.DescribeLoadBalancerTCPListenerAttribute(listenerAttributeRequest)
	if err != nil {
		cblogger.Info(err.Error())
		return irs.NLBInfo{}, err
	}

	// backend 정보가 여기에 있어서 먼저 set.
	vmGroup.Protocol = listener.Protocol
	vmGroup.Port = strconv.Itoa(listenerAttributeResponse.BackendServerPort)

	// health checker 정보가 여기에 있어서 set.
	healthChecker.Protocol = listener.Protocol
	healthChecker.Port = strconv.Itoa(listenerAttributeResponse.HealthCheckConnectPort)
	healthChecker.Threshold = listenerAttributeResponse.HealthyThreshold
	healthChecker.Timeout = listenerAttributeResponse.HealthCheckConnectTimeout
	healthChecker.Interval = listenerAttributeResponse.HealthCheckInterval

	nlbInfo.IId = nlbIID
	//nlbInfo.Listener = listener// listener자체는 변경안됨( protocol, port 변경 불가함.)
	nlbInfo.VMGroup = vmGroup
	nlbInfo.HealthChecker = healthChecker
	return nlbInfo, nil
}

/*
	UDP Listener를 조회하여 CB-Spider의 객체에 맞게 set하여 return
	호출하는 곳에서 원하는 값을 추출하여 사용

*/
func (NLBHandler *AlibabaNLBHandler) describeLoadBalancerUdpListenerAttribute(nlbIID irs.IID, listener irs.ListenerInfo) (irs.NLBInfo, error) {
	nlbInfo := irs.NLBInfo{}
	vmGroup := irs.VMGroupInfo{}
	healthChecker := irs.HealthCheckerInfo{}

	// 조회
	listenerAttributeRequest := slb.CreateDescribeLoadBalancerUDPListenerAttributeRequest()
	listenerAttributeRequest.LoadBalancerId = nlbIID.SystemId
	listenerAttributeRequest.ListenerPort = requests.Integer(listener.Port)
	printToJson(listener)
	printToJson(listenerAttributeRequest)
	listenerAttributeResponse, err := NLBHandler.Client.DescribeLoadBalancerUDPListenerAttribute(listenerAttributeRequest)
	if err != nil {
		cblogger.Info(err.Error())
		return irs.NLBInfo{}, err
	}

	// backend 정보가 여기에 있어서 먼저 set.
	vmGroup.Protocol = listener.Protocol
	vmGroup.Port = strconv.Itoa(listenerAttributeResponse.BackendServerPort)

	// health checker 정보가 여기에 있어서 set.
	healthChecker.Protocol = listener.Protocol
	healthChecker.Port = strconv.Itoa(listenerAttributeResponse.HealthCheckConnectPort)
	healthChecker.Threshold = listenerAttributeResponse.HealthyThreshold
	healthChecker.Timeout = listenerAttributeResponse.HealthCheckConnectTimeout
	healthChecker.Interval = listenerAttributeResponse.HealthCheckInterval

	nlbInfo.IId = nlbIID
	//nlbInfo.Listener = listener// listener자체는 변경안됨( protocol, port 변경 불가함.)
	nlbInfo.VMGroup = vmGroup
	nlbInfo.HealthChecker = healthChecker
	return nlbInfo, nil
}

/*
	LB 생성 시 validation check

	udplistener : You cannot specify ports 250, 4789, or 4790 for UDP listeners. They are system reserved ports.
*/
func (NLBHandler *AlibabaNLBHandler) validateCreateNLB(nlbReqInfo irs.NLBInfo) error {
	// lb

	// listener
	listener := nlbReqInfo.Listener

	// listener port : 1 to 65535
	portVal, err := strconv.Atoi(listener.Port)
	if err != nil {
		return errors.New("The appropriate value for the listener port is 1 to 65535. " + listener.Port)
	}
	if portVal < 1 || portVal > 66535 {
		return errors.New("The appropriate value for the listener port is 1 to 65535. " + listener.Port)
	}

	// vm
	vmGroup := nlbReqInfo.VMGroup
	maxVmCount := 20
	vmCount := len(*vmGroup.VMs)
	if maxVmCount < vmCount {
		return errors.New("You can add at most 20 backend servers to a CLB instance in each request " + strconv.Itoa(vmCount))
	}

	// health check
	healthChecker := nlbReqInfo.HealthChecker
	if healthChecker.Threshold >= 2 && healthChecker.Threshold <= 10 {
	} else {
		return errors.New("The appropriate value for the threshold is 2 to 10")
	}
	if healthChecker.Timeout >= 1 && healthChecker.Timeout <= 300 {
	} else {
		return errors.New("The appropriate value for the timeout is 1 to 300")
	}
	if healthChecker.Interval >= 1 && healthChecker.Interval <= 50 {
	} else {
		return errors.New("The appropriate value for the interval is 1 to 50")
	}

	return nil
}
