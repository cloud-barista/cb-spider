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

		listenerResponse, err := NLBHandler.Client.CreateListener(listenerRequest)
		if err != nil {
			return irs.NLBInfo{}, err
		}
		cblogger.Debug(listenerResponse.ToJsonString())
	}
	

	callogger.Info(call.String(callLogInfo))
	fmt.Printf("%s", nlbResponse.ToJsonString())

	return irs.NLBInfo{}, nlbErr
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

		if curStatus == 1 { 
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