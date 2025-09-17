// Proof of Concepts of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is a Cloud Driver Example for PoC Test.
//
// by ETRI Team, 2022.07.
// by ETRI Team, 2024.04.

package resources

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	calllog "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/vpcs"
	"github.com/goccy/go-json"

	nhnsdk "github.com/cloud-barista/nhncloud-sdk-go"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/compute/v2/servers"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/loadbalancer/v2/listeners"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/loadbalancer/v2/loadbalancers"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/loadbalancer/v2/monitors"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/loadbalancer/v2/pools"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/loadbalancer/v2/providers"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/extensions/layer3/floatingips"
	"github.com/cloud-barista/nhncloud-sdk-go/openstack/networking/v2/networks"
)

type NhnCloudNLBHandler struct {
	RegionInfo    idrv.RegionInfo
	VMClient      *nhnsdk.ServiceClient
	NetworkClient *nhnsdk.ServiceClient
}

const (
	PublicType          string = "shared"
	InternalType        string = "dedicated"
	DefaultWeight       int    = 1
	DefaultAdminStateUp bool   = true

	// NHN Cloud default value for Listener and Health Monitor
	DefaultConnectionLimit        int = 2000 // NHN Cloud Listener ConnectionLimit range : 1 ~ 60000 (Dedicated LB : 1 ~ 480000)
	DefaultKeepAliveTimeout       int = 300  // NHN Cloud Listener KeepAliveTimeout range : 0 ~ 3600
	DefaultHealthCheckerInterval  int = 30
	DefaultHealthCheckerTimeout   int = 5
	DefaultHealthCheckerThreshold int = 2
)

func (nlbHandler *NhnCloudNLBHandler) checkNLBClient() error {
	if nlbHandler.NetworkClient == nil {
		return errors.New("this NHNCloud environment cannot provide LoadBalancer. Please check if LoadBalancer service is enabled")
	}
	return nil
}

func (nlbHandler *NhnCloudNLBHandler) CreateNLB(nlbReqInfo irs.NLBInfo) (createNLB irs.NLBInfo, createError error) {
	cblogger.Info("NHN Cloud Driver: called CreateNLB()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbReqInfo.IId.NameId, "CreateNLB()")
	callLogStart := calllog.Start()

	if err := nlbHandler.checkNLBClient(); err != nil {
		newErr := fmt.Errorf("failed to create NLB: %w", err)
		cblogger.Error(newErr.Error())
		callLogInfo.ErrorMSG = newErr.Error()
		calllogger := calllog.GetLogger("NHN-CALLLOG")
		calllogger.Error(calllog.String(callLogInfo))
		return irs.NLBInfo{}, newErr
	}

	lb, err := nlbHandler.createLoadBalancer(nlbReqInfo)
	if err != nil {
		newErr := fmt.Errorf("failed to create LoadBalancer: %w", err)
		cblogger.Error(newErr.Error())
		callLogInfo.ErrorMSG = newErr.Error()
		calllogger := calllog.GetLogger("NHN-CALLLOG")
		calllogger.Error(calllog.String(callLogInfo))
		return irs.NLBInfo{}, newErr
	}

	listener, err := nlbHandler.createListener(lb.ID, nlbReqInfo)
	if err != nil {
		newErr := fmt.Errorf("failed to create Listener: %w", err)
		cblogger.Error(newErr.Error())
		callLogInfo.ErrorMSG = newErr.Error()
		calllogger := calllog.GetLogger("NHN-CALLLOG")
		calllogger.Error(calllog.String(callLogInfo))
		return irs.NLBInfo{}, newErr
	}
	defer func() {
		if createError != nil {
			_ = nlbHandler.cleanUpNLB(irs.IID{SystemId: lb.ID})
		}
	}()

	pool, err := nlbHandler.createPool(listener.ID, nlbReqInfo)
	if err != nil {
		newErr := fmt.Errorf("failed to create Pool: %w", err)
		cblogger.Error(newErr.Error())
		callLogInfo.ErrorMSG = newErr.Error()
		calllogger := calllog.GetLogger("NHN-CALLLOG")
		calllogger.Error(calllog.String(callLogInfo))
		return irs.NLBInfo{}, newErr
	}
	cblogger.Infof("============================= Created Pool: ID=%s, Protocol=%s, ListenerID=%s ============================= ", pool.ID, pool.Protocol, listener.ID)

	_, err = nlbHandler.createHealthMonitor(pool.ID, nlbReqInfo)
	if err != nil {
		newErr := fmt.Errorf("failed to create HealthMonitor: %w", err)
		cblogger.Error(newErr.Error())
		callLogInfo.ErrorMSG = newErr.Error()
		calllogger := calllog.GetLogger("NHN-CALLLOG")
		calllogger.Error(calllog.String(callLogInfo))
		return irs.NLBInfo{}, newErr
	}

	publicIP, err := nlbHandler.createPublicIP(lb.VipPortID)
	if err != nil {
		newErr := fmt.Errorf("failed to create Public IP: %w", err)
		cblogger.Error(newErr.Error())
		callLogInfo.ErrorMSG = newErr.Error()
		calllogger := calllog.GetLogger("NHN-CALLLOG")
		calllogger.Error(calllog.String(callLogInfo))
		return irs.NLBInfo{}, newErr
	}

	rawnlb, err := nlbHandler.getRawNLB(irs.IID{SystemId: lb.ID})
	if err != nil {
		newErr := fmt.Errorf("failed to get RawNLB: %w", err)
		cblogger.Error(newErr.Error())
		callLogInfo.ErrorMSG = newErr.Error()
		calllogger := calllog.GetLogger("NHN-CALLLOG")
		calllogger.Error(calllog.String(callLogInfo))
		return irs.NLBInfo{}, newErr
	}

	if listeners, ok := rawnlb["listeners"].([]interface{}); ok && len(listeners) > 0 {
		for _, l := range listeners {
			if lm, ok := l.(map[string]interface{}); ok {
				cblogger.Infof("=============================  Listener Found: ID=%s, DefaultPoolID=%v", lm["id"], lm["default_pool_id ============================= "])
			}
		}
	}

	if pools, ok := rawnlb["pools"].([]interface{}); ok && len(pools) > 0 {
		for _, p := range pools {
			if pm, ok := p.(map[string]interface{}); ok {
				cblogger.Infof("=============================  Pool Found: ID=%s ============================= ", pm["id"])
			}
		}
	}

	createNLB, err = nlbHandler.setterNLB(rawnlb)
	if err != nil {
		newErr := fmt.Errorf("failed to set NLB Info: %w", err)
		cblogger.Error(newErr.Error())
		callLogInfo.ErrorMSG = newErr.Error()
		calllogger := calllog.GetLogger("NHN-CALLLOG")
		calllogger.Error(calllog.String(callLogInfo))
		return irs.NLBInfo{}, newErr
	}
	createNLB.Listener.IP = publicIP

	callLogInfo.ElapsedTime = calllog.Elapsed(callLogStart)
	callLogInfo.ErrorMSG = ""
	calllogger := calllog.GetLogger("NHN-CALLLOG")
	calllogger.Info(calllog.String(callLogInfo))

	return createNLB, nil
}

func (nlbHandler *NhnCloudNLBHandler) createLoadBalancer(nlbReqInfo irs.NLBInfo) (*loadbalancers.LoadBalancer, error) {
	listOpts := loadbalancers.ListOpts{Name: nlbReqInfo.IId.NameId}
	allPages, err := loadbalancers.List(nlbHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		return nil, err
	}
	existList, _ := loadbalancers.ExtractLoadBalancers(allPages)
	if len(existList) > 0 {
		return nil, fmt.Errorf("already exist NLB Name %s", nlbReqInfo.IId.NameId)
	}

	subnetID, _, err := nlbHandler.getFirstSubnetAndNetworkId(nlbReqInfo.VpcIID.NameId)
	if err != nil {
		return nil, err
	}

	createOpts := loadbalancers.CreateOpts{
		Name:         nlbReqInfo.IId.NameId,
		AdminStateUp: DefaultAdminStateUp,
		VipSubnetID:  subnetID,
		// VipAddress:   networkID,  // ❌ 제거
	}

	newLB, err := loadbalancers.Create(nlbHandler.NetworkClient, createOpts).Extract()
	if err != nil {
		return nil, fmt.Errorf("failed to create loadbalancer: %s", err)
	}

	_, err = nlbHandler.waitToGetNLBInfo(irs.IID{SystemId: newLB.ID})
	if err != nil {
		return nil, err
	}

	return newLB, nil
}

func (nlbHandler *NhnCloudNLBHandler) createHealthMonitor(poolID string, nlbReqInfo irs.NLBInfo) (*monitors.Monitor, error) {
	if poolID == "" {
		return nil, fmt.Errorf("invalid pool ID")
	}

	hc := nlbReqInfo.HealthChecker
	portNum, err := strconv.Atoi(hc.Port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return nil, fmt.Errorf("invalid health check port: %s", hc.Port)
	}

	protocol := strings.ToUpper(hc.Protocol)
	if protocol != "TCP" && protocol != "HTTP" && protocol != "HTTPS" {
		return nil, fmt.Errorf("unsupported health check protocol: %s", hc.Protocol)
	}

	interval := hc.Interval
	if interval <= 0 {
		interval = DefaultHealthCheckerInterval
	}
	timeout := hc.Timeout
	if timeout <= 0 {
		timeout = DefaultHealthCheckerTimeout
	}
	threshold := hc.Threshold
	if threshold <= 0 {
		threshold = DefaultHealthCheckerThreshold
	}

	createOpts := monitors.CreateOpts{
		PoolID:          poolID,
		Type:            protocol,
		Delay:           interval,
		Timeout:         timeout,
		MaxRetries:      threshold,
		HealthCheckPort: portNum,
	}

	monitor, err := monitors.Create(nlbHandler.NetworkClient, createOpts).Extract()
	if err != nil {
		return nil, fmt.Errorf("failed to create health monitor: %s", err)
	}
	return monitor, nil
}

func (nlbHandler *NhnCloudNLBHandler) getRawNLB(iid irs.IID) (map[string]interface{}, error) {
	var body []byte

	if iid.SystemId != "" {
		resp, err := loadbalancers.Get(nlbHandler.NetworkClient, iid.SystemId).Extract()
		if err != nil {
			return nil, err
		}
		body, _ = json.Marshal(resp)
	} else if iid.NameId != "" {
		listOpts := loadbalancers.ListOpts{Name: iid.NameId}
		rawListAllPage, err := loadbalancers.List(nlbHandler.NetworkClient, listOpts).AllPages()
		if err != nil {
			return nil, err
		}
		list, err := loadbalancers.ExtractLoadBalancers(rawListAllPage)
		if err != nil {
			return nil, err
		}
		if len(list) != 1 {
			return nil, fmt.Errorf("not Exist NLB: %s", iid.NameId)
		}
		body, _ = json.Marshal(list[0])
	} else {
		return nil, fmt.Errorf("invalid IID: %+v", iid)
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse raw NLB: %v", err)
	}
	return raw, nil
}

func (nlbHandler *NhnCloudNLBHandler) getRawNLBList() ([]loadbalancers.LoadBalancer, error) {
	listOpts := loadbalancers.ListOpts{}
	rawListAllPage, err := loadbalancers.List(nlbHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		return nil, err
	}
	list, err := loadbalancers.ExtractLoadBalancers(rawListAllPage)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (nlbHandler *NhnCloudNLBHandler) ListNLB() ([]*irs.NLBInfo, error) {
	cblogger.Info("NHN Cloud Driver: called ListNLB()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", "ListNLB()", "ListNLB()")
	callLogStart := calllog.Start()

	var nlbList []*irs.NLBInfo

	allPages, err := loadbalancers.List(nlbHandler.NetworkClient, nil).AllPages()
	if err != nil {
		newErr := fmt.Errorf("failed to list NLBs: %w", err)
		cblogger.Error(newErr.Error())
		callLogInfo.ErrorMSG = newErr.Error()
		calllogger := calllog.GetLogger("NHN-CALLLOG")
		calllogger.Error(calllog.String(callLogInfo))
		return nil, newErr
	}

	var rawBody map[string]interface{}
	if err := allPages.(loadbalancers.LoadBalancerPage).ExtractInto(&rawBody); err != nil {
		newErr := fmt.Errorf("failed to extract NLBs into raw map: %w", err)
		cblogger.Error(newErr.Error())
		callLogInfo.ErrorMSG = newErr.Error()
		calllogger := calllog.GetLogger("NHN-CALLLOG")
		calllogger.Error(calllog.String(callLogInfo))
		return nil, newErr
	}

	lbList, ok := rawBody["loadbalancers"].([]interface{})
	if !ok {
		newErr := fmt.Errorf("invalid response format: no loadbalancers field")
		cblogger.Error(newErr.Error())
		callLogInfo.ErrorMSG = newErr.Error()
		calllogger := calllog.GetLogger("NHN-CALLLOG")
		calllogger.Error(calllog.String(callLogInfo))
		return nil, newErr
	}

	for _, lb := range lbList {
		lbMap, ok := lb.(map[string]interface{})
		if !ok {
			cblogger.Warn("skip invalid loadbalancer entry")
			continue
		}

		nlbInfo, err := nlbHandler.setterNLB(lbMap)
		if err != nil {
			cblogger.Error(fmt.Sprintf("failed to set NLB Info: %v", err))
			continue
		}
		nlbList = append(nlbList, &nlbInfo)
	}

	callLogInfo.ElapsedTime = calllog.Elapsed(callLogStart)
	callLogInfo.ErrorMSG = ""
	calllogger := calllog.GetLogger("NHN-CALLLOG")
	calllogger.Info(calllog.String(callLogInfo))

	return nlbList, nil
}

func (nlbHandler *NhnCloudNLBHandler) getRawNLBProviderList() ([]providers.Provider, error) {
	listOpts := providers.ListOpts{}

	rawListAllPage, err := providers.List(nlbHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		return nil, err
	}

	list, err := providers.ExtractProviders(rawListAllPage)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (nlbHandler *NhnCloudNLBHandler) getRawNLBProvider() (string, error) {
	list, err := nlbHandler.getRawNLBProviderList()
	if err != nil {
		return "", err
	}

	for _, provider := range list {
		if provider.Name == "amphora" {
			return provider.Name, nil
		}
	}
	return "", errors.New("no Exist NHNCloud LoadBalancer Provider amphora")
}

func (nlbHandler *NhnCloudNLBHandler) createListener(nlbID string, nlbReqInfo irs.NLBInfo) (*listeners.Listener, error) {
	if nlbID == "" {
		return nil, fmt.Errorf("invalid NLB ID")
	}

	portNum, err := strconv.Atoi(nlbReqInfo.Listener.Port)
	if err != nil || portNum < 1 || portNum > 65535 {
		return nil, fmt.Errorf("invalid listener port: %s", nlbReqInfo.Listener.Port)
	}

	protocol, err := getListenerProtocol(nlbReqInfo.Listener.Protocol)
	if err != nil {
		return nil, err
	}

	createOpts := listeners.CreateOpts{
		Name:             nlbReqInfo.IId.NameId,
		LoadbalancerID:   nlbID,
		Protocol:         protocol,
		ProtocolPort:     portNum,
		AdminStateUp:     DefaultAdminStateUp,
		ConnLimit:        DefaultConnectionLimit,
		KeepAliveTimeout: DefaultKeepAliveTimeout,
	}

	listener, err := listeners.Create(nlbHandler.NetworkClient, createOpts).Extract()
	if err != nil {
		return nil, fmt.Errorf("failed to create listener: %s", err)
	}

	if err := nlbHandler.waitForListenerActive(listener.ID); err != nil {
		return nil, fmt.Errorf("listener %s did not become ACTIVE: %w", listener.ID, err)
	}

	if _, err := nlbHandler.waitToGetNLBInfo(irs.IID{SystemId: nlbID}); err != nil {
		return nil, fmt.Errorf("loadbalancer %s did not become ACTIVE after listener attach: %w", nlbID, err)
	}

	return listener, nil
}

func (nlbHandler *NhnCloudNLBHandler) waitForListenerActive(listenerID string) error {
	maxRetry := 60
	for i := 0; i < maxRetry; i++ {
		listener, err := nlbHandler.getRawListenerById(listenerID)
		if err != nil {
			return fmt.Errorf("failed to get listener %s: %w", listenerID, err)
		}
		if len(listener.Loadbalancers) < 1 {
			return fmt.Errorf("listener %s has no associated loadbalancer", listenerID)
		}

		lbID := listener.Loadbalancers[0].ID
		lb, err := nlbHandler.getRawNLB(irs.IID{SystemId: lbID})
		if err != nil {
			return fmt.Errorf("failed to get loadbalancer %s: %w", lbID, err)
		}

		switch strings.ToUpper(lb["provisioning_status"].(string)) {
		case "ACTIVE":
			return nil
		case "ERROR":
			return fmt.Errorf("loadbalancer %s provisioning ERROR", lbID)
		}

		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("timeout: listener %s not ACTIVE", listenerID)
}

func (nlbHandler *NhnCloudNLBHandler) getListener(listenerID string) (*listeners.Listener, error) {
	if listenerID == "" {
		return nil, fmt.Errorf("invalid listener ID")
	}

	listOpts := listeners.ListOpts{ID: listenerID}
	allPages, err := listeners.List(nlbHandler.NetworkClient, &listOpts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("failed to list listeners: %s", err)
	}

	listenerList, err := listeners.ExtractListeners(allPages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract listeners: %s", err)
	}

	if len(listenerList) < 1 {
		return nil, fmt.Errorf("listener not found: %s", listenerID)
	}

	return &listenerList[0], nil
}

func getListenerProtocol(protocol string) (listeners.Protocol, error) {
	switch strings.ToUpper(protocol) {
	case "TCP":
		return listeners.ProtocolTCP, nil
	case "HTTP":
		return listeners.ProtocolHTTP, nil
	case "HTTPS":
		return listeners.ProtocolHTTPS, nil
	case "TERMINATED_HTTPS":
		return listeners.ProtocolTerminatedHTTPS, nil
	}
	return "", fmt.Errorf("unsupported listener protocol: %s", protocol)
}

func (nlbHandler *NhnCloudNLBHandler) getPoolCreateOpt(nlbReqInfo irs.NLBInfo, listenerID string) (pools.CreateOpts, error) {
	poolProtocol, err := getPoolProtocol(nlbReqInfo.VMGroup.Protocol)
	if err != nil {
		return pools.CreateOpts{}, err
	}

	portInt, err := strconv.Atoi(nlbReqInfo.VMGroup.Port)
	if err != nil || portInt < 1 || portInt > 65535 {
		return pools.CreateOpts{}, fmt.Errorf("invalid VMGroup Port: %s", nlbReqInfo.VMGroup.Port)
	}

	return pools.CreateOpts{
		ListenerID:   listenerID,
		Name:         nlbReqInfo.IId.NameId,
		LBMethod:     pools.LBMethodRoundRobin,
		Protocol:     poolProtocol,
		AdminStateUp: true,
		MemberPort:   portInt,
		Description:  fmt.Sprintf("Pool for NLB %s", nlbReqInfo.IId.NameId),
	}, nil
}

func (nlbHandler *NhnCloudNLBHandler) createPool(listenerID string, nlbReqInfo irs.NLBInfo) (*pools.Pool, error) {
	cblogger.Info("NHN Cloud Driver: called createPool()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCER", "createPool()", "createPool()")
	callLogStart := calllog.Start()

	if listenerID == "" {
		newErr := fmt.Errorf("invalid listener ID")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	poolCreateOpts, err := nlbHandler.getPoolCreateOpt(nlbReqInfo, listenerID)
	if err != nil {
		newErr := fmt.Errorf("failed to get PoolCreateOpt: %s", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	pool, err := pools.Create(nlbHandler.NetworkClient, poolCreateOpts).Extract()
	if err != nil {
		newErr := fmt.Errorf("failed to create pool: %s", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	listener, err := nlbHandler.getRawListenerById(listenerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get listener %s after pool creation: %w", listenerID, err)
	}
	if len(listener.Loadbalancers) < 1 {
		return nil, fmt.Errorf("listener %s has no associated loadbalancer", listenerID)
	}
	lbID := listener.Loadbalancers[0].ID

	_, err = nlbHandler.waitToGetNLBInfo(irs.IID{SystemId: lbID})
	if err != nil {
		return nil, fmt.Errorf("loadbalancer %s did not become ACTIVE after pool creation: %s", lbID, err)
	}

	updateMap := map[string]interface{}{
		"listener": map[string]interface{}{
			"default_pool_id": pool.ID,
		},
	}

	resp, err := nlbHandler.NetworkClient.Put(
		nlbHandler.NetworkClient.ServiceURL("lbaas", "listeners", listenerID),
		updateMap,
		nil,
		nil,
	)
	if err != nil {

		if strings.Contains(err.Error(), "but got 200 instead") {
			cblogger.Infof("listener %s updated successfully with pool %s (HTTP 200 via SDK error)", listenerID, pool.ID)
		} else {
			return nil, fmt.Errorf("failed to update listener with default pool: %s", err)
		}
	} else {
		defer resp.Body.Close()
		if resp.StatusCode != 200 && resp.StatusCode != 201 && resp.StatusCode != 202 {
			return nil, fmt.Errorf("unexpected HTTP status code when updating listener: %d", resp.StatusCode)
		}
		cblogger.Infof("listener %s updated successfully with pool %s (HTTP %d)", listenerID, pool.ID, resp.StatusCode)
	}

	_, err = nlbHandler.waitingNLBPoolActive(pool.ID)
	if err != nil {
		newErr := fmt.Errorf("pool did not become active: %s", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	LoggingInfo(callLogInfo, callLogStart)
	return pool, nil
}

func (nlbHandler *NhnCloudNLBHandler) setPoolDescription(des string, poolId string) (*pools.Pool, error) {
	updateOpts := pools.UpdateOpts{
		Description: &des,
	}
	pool, err := pools.Update(nlbHandler.NetworkClient, poolId, updateOpts).Extract()
	if err != nil {
		return nil, err
	}

	_, err = nlbHandler.waitingNLBPoolActive(pool.ID)
	if err != nil {
		return nil, err
	}

	return pool, nil
}

func (nlbHandler *NhnCloudNLBHandler) getPool(poolID string) (*pools.Pool, error) {
	if poolID == "" {
		return nil, fmt.Errorf("invalid pool ID")
	}

	listOpts := pools.ListOpts{ID: poolID}
	allPages, err := pools.List(nlbHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("failed to list pools: %s", err)
	}

	poolList, err := pools.ExtractPools(allPages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract pools: %s", err)
	}

	if len(poolList) < 1 {
		return nil, fmt.Errorf("pool not found: %s", poolID)
	}
	return &poolList[0], nil
}

func getPoolProtocol(protocol string) (pools.Protocol, error) {
	switch strings.ToUpper(protocol) {
	case "TCP":
		return pools.ProtocolTCP, nil
	case "HTTP":
		return pools.ProtocolHTTP, nil
	case "HTTPS":
		return pools.ProtocolHTTPS, nil
	}
	return "", fmt.Errorf("unsupported pool protocol: %s", protocol)
}

func (nlbHandler *NhnCloudNLBHandler) waitingNLBPoolActive(poolID string) (bool, error) {
	pool, err := nlbHandler.getRawPoolById(poolID)
	if err != nil {
		return false, err
	}

	if len(pool.Listeners) < 1 {
		return false, fmt.Errorf("pool %s has no associated listener", poolID)
	}

	listenerID := pool.Listeners[0].ID
	listener, err := nlbHandler.getRawListenerById(listenerID)
	if err != nil {
		return false, err
	}

	if len(listener.Loadbalancers) < 1 {
		return false, fmt.Errorf("listener %s has no associated loadbalancer", listenerID)
	}
	lbID := listener.Loadbalancers[0].ID

	curRetry := 0
	maxRetry := 360
	for {
		curRetry++

		listener, err := nlbHandler.getRawListenerById(listenerID)
		if err == nil {
			if listener.DefaultPoolID == poolID {

				rawLB, err := nlbHandler.getRawNLB(irs.IID{SystemId: lbID})
				if err == nil {
					status := fmt.Sprintf("%v", rawLB["provisioning_status"])
					switch strings.ToUpper(status) {
					case "ACTIVE":
						cblogger.Infof("Listener %s successfully bound with pool %s and LB %s is ACTIVE",
							listenerID, poolID, lbID)
						return true, nil
					case "ERROR":
						return false, fmt.Errorf("loadbalancer %s provisioning ERROR", lbID)
					}
				}
			} else {
				cblogger.Infof("Listener %s still has no default_pool_id (expected %s), retry %d/%d",
					listenerID, poolID, curRetry, maxRetry)
			}
		}

		if curRetry > maxRetry {
			return false, fmt.Errorf("timeout: listener %s default_pool_id not set to %s after %d retries",
				listenerID, poolID, maxRetry)
		}
		time.Sleep(1 * time.Second)
	}
}

func (nlbHandler *NhnCloudNLBHandler) getRawMonitorById(monitorID string) (*monitors.Monitor, error) {
	if monitorID == "" {
		return nil, fmt.Errorf("invalid monitor ID")
	}

	return monitors.Get(nlbHandler.NetworkClient, monitorID).Extract()
}

func (nlbHandler *NhnCloudNLBHandler) getRawListenerById(listenerID string) (*listeners.Listener, error) {
	if listenerID == "" {
		return nil, fmt.Errorf("invalid listener ID")
	}

	return listeners.Get(nlbHandler.NetworkClient, listenerID).Extract()
}

func (nlbHandler *NhnCloudNLBHandler) getRawPoolById(poolID string) (*pools.Pool, error) {
	if poolID == "" {
		return nil, fmt.Errorf("invalid pool ID")
	}

	listOpts := pools.ListOpts{
		ID: poolID,
	}

	allPages, err := pools.List(nlbHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		return nil, err
	}

	list, err := pools.ExtractPools(allPages)
	if err != nil {
		return nil, err
	}

	if len(list) != 1 {
		return nil, fmt.Errorf("not Exist Pool with ID %s", poolID)
	}

	return &list[0], nil
}

func (nlbHandler *NhnCloudNLBHandler) getRawPoolMembersById(poolID string) (*[]pools.Member, error) {
	if poolID == "" {
		return nil, fmt.Errorf("invalid pool ID")
	}

	pages, err := pools.ListMembers(nlbHandler.NetworkClient, poolID, pools.ListMembersOpts{}).AllPages()
	if err != nil {
		return nil, err
	}
	list, err := pools.ExtractMembers(pages)
	if err != nil {
		return nil, err
	}
	return &list, nil
}

func (nlbHandler *NhnCloudNLBHandler) getRawPoolMemberById(poolID, memberID string) (*pools.Member, error) {
	if poolID == "" || memberID == "" {
		return nil, fmt.Errorf("invalid pool/member ID")
	}

	pages, err := pools.ListMembers(nlbHandler.NetworkClient, poolID, pools.ListMembersOpts{ID: memberID}).AllPages()
	if err != nil {
		return nil, err
	}
	list, err := pools.ExtractMembers(pages)
	if err != nil {
		return nil, err
	}
	if len(list) != 1 {
		return nil, fmt.Errorf("not found PoolMember %s", memberID)
	}
	return &list[0], nil
}

func (nlbHandler *NhnCloudNLBHandler) ChangeVMGroupInfo(nlbIID irs.IID, vmGroup irs.VMGroupInfo) (irs.VMGroupInfo, error) {
	cblogger.Info("NHN Cloud Driver: called ChangeVMGroupInfo()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCER", nlbIID.NameId, "ChangeVMGroupInfo()")
	callLogStart := calllog.Start()

	if err := nlbHandler.checkNLBClient(); err != nil {
		newErr := fmt.Errorf("failed to change VMGroupInfo: %w", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	rawNLB, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("failed to get NLB: %w", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	poolsIfc, ok := rawNLB["pools"].([]interface{})
	if !ok || len(poolsIfc) < 1 {
		newErr := fmt.Errorf("no pool found in NLB %s", nlbIID.SystemId)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	poolMap, ok := poolsIfc[0].(map[string]interface{})
	if !ok {
		newErr := fmt.Errorf("invalid pool format in NLB %s", nlbIID.SystemId)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}
	poolId, _ := poolMap["id"].(string)

	oldPool, err := nlbHandler.getRawPoolById(poolId)
	if err != nil {
		newErr := fmt.Errorf("failed to get pool %s: %w", poolId, err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	portInt, err := strconv.Atoi(vmGroup.Port)
	if err != nil || portInt < 1 || portInt > 65535 {
		newErr := fmt.Errorf("invalid VMGroup port: %s", vmGroup.Port)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	if vmGroup.Protocol != "" && !strings.EqualFold(vmGroup.Protocol, oldPool.Protocol) {
		newErr := fmt.Errorf("changing VMGroup protocol is not supported")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	oldVMGroup, err := nlbHandler.getVMGroup(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("failed to get old VMGroup: %w", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	if strings.EqualFold(vmGroup.Port, oldVMGroup.Port) {
		return oldVMGroup, nil
	}

	if err := nlbHandler.detachPoolMembers(poolId, *oldVMGroup.VMs); err != nil {
		newErr := fmt.Errorf("failed to detach old pool members: %w", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	if _, err := nlbHandler.attachPoolMembers(*oldVMGroup.VMs, vmGroup.Port, poolId); err != nil {
		newErr := fmt.Errorf("failed to re-attach pool members: %w", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	desc := fmt.Sprintf("cb-vmgroup-port-%d", portInt)
	if _, err := nlbHandler.setPoolDescription(desc, poolId); err != nil {
		newErr := fmt.Errorf("failed to update pool description: %w", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	rawNLB, err = nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("failed to refresh NLB: %w", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}
	newVMGroup, err := nlbHandler.getVMGroup(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("failed to get new VMGroup: %w", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.VMGroupInfo{}, newErr
	}

	LoggingInfo(callLogInfo, callLogStart)
	return newVMGroup, nil
}

func (nlbHandler *NhnCloudNLBHandler) ChangeListener(nlbIID irs.IID, listenerInfo irs.ListenerInfo) (irs.ListenerInfo, error) {

	rawLB, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		return irs.ListenerInfo{}, fmt.Errorf("failed to get NLB: %s", err)
	}

	listenersArr, ok := rawLB["listeners"].([]interface{})
	if !ok || len(listenersArr) < 1 {
		return irs.ListenerInfo{}, fmt.Errorf("no listener found for NLB %s", nlbIID.SystemId)
	}

	listener := listenersArr[0].(map[string]interface{})
	listenerID := fmt.Sprintf("%v", listener["id"])
	listenerProtocol := fmt.Sprintf("%v", listener["protocol"])
	listenerPort := fmt.Sprintf("%v", listener["protocol_port"])

	updateOpts := listeners.UpdateOpts{}

	if listenerInfo.Protocol != "" && !strings.EqualFold(listenerInfo.Protocol, listenerProtocol) {
		return irs.ListenerInfo{}, fmt.Errorf("protocol change is not supported")
	}

	if listenerInfo.DNSName != "" {
		updateOpts.Name = &listenerInfo.DNSName
	}

	connLimit := DefaultConnectionLimit
	updateOpts.ConnLimit = &connLimit
	keepAlive := DefaultKeepAliveTimeout
	updateOpts.KeepAliveTimeout = &keepAlive

	_, err = listeners.Update(nlbHandler.NetworkClient, listenerID, updateOpts).Extract()
	if err != nil {
		return irs.ListenerInfo{}, fmt.Errorf("failed to update listener: %s", err)
	}

	updated := irs.ListenerInfo{
		Protocol: listenerProtocol,
		Port:     listenerPort,
		CspID:    listenerID,
	}

	return updated, nil
}

func (nlbHandler *NhnCloudNLBHandler) ChangeHealthCheckerInfo(nlbIID irs.IID, healthChecker irs.HealthCheckerInfo) (irs.HealthCheckerInfo, error) {
	rawLB, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		return irs.HealthCheckerInfo{}, err
	}

	poolID, err := nlbHandler.getDefaultPoolIDFromNLB(rawLB)
	if err != nil {
		return irs.HealthCheckerInfo{}, err
	}

	pool, err := nlbHandler.getDefaultPool(poolID)
	if err != nil {
		return irs.HealthCheckerInfo{}, err
	}
	if pool.MonitorID == "" {
		return irs.HealthCheckerInfo{}, fmt.Errorf("no monitor attached to pool %s", pool.ID)
	}

	updateOpts := monitors.UpdateOpts{
		Timeout: healthChecker.Timeout,
	}

	_, err = monitors.Update(nlbHandler.NetworkClient, pool.MonitorID, updateOpts).Extract()
	if err != nil {
		return irs.HealthCheckerInfo{}, err
	}

	return healthChecker, nil
}

func (nlbHandler *NhnCloudNLBHandler) createPublicIP(portID string) (string, error) {
	if portID == "" {
		return "", fmt.Errorf("invalid VIP port ID")
	}

	externVPCID, err := getPublicVPCInfo(nlbHandler.NetworkClient, "ID")
	if err != nil {
		return "", fmt.Errorf("failed to get public VPC info: %s", err)
	}

	createOpts := floatingips.CreateOpts{
		FloatingNetworkID: externVPCID,
		PortID:            portID,
	}
	fip, err := floatingips.Create(nlbHandler.NetworkClient, createOpts).Extract()
	if err != nil {
		return "", fmt.Errorf("failed to create floating IP: %s", err)
	}
	return fip.FloatingIP, nil
}

func (nlbHandler *NhnCloudNLBHandler) getPublicIP(privateIP string) (string, error) {
	if privateIP == "" {
		return "", fmt.Errorf("invalid private IP")
	}

	listOpts := floatingips.ListOpts{FixedIP: privateIP}
	allPages, err := floatingips.List(nlbHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		return "", fmt.Errorf("failed to list floating IPs: %s", err)
	}

	fipList, err := floatingips.ExtractFloatingIPs(allPages)
	if err != nil {
		return "", fmt.Errorf("failed to extract floating IPs: %s", err)
	}

	if len(fipList) < 1 {
		return "", fmt.Errorf("no floating IP found for private IP %s", privateIP)
	}
	return fipList[0].FloatingIP, nil
}

func (nlbHandler *NhnCloudNLBHandler) deletePublicIP(portID string) error {
	if portID == "" {
		return fmt.Errorf("invalid VIP port ID")
	}

	listOpts := floatingips.ListOpts{PortID: portID}
	allPages, err := floatingips.List(nlbHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		return fmt.Errorf("failed to list floating IPs: %s", err)
	}

	fipList, err := floatingips.ExtractFloatingIPs(allPages)
	if err != nil {
		return fmt.Errorf("failed to extract floating IPs: %s", err)
	}

	if len(fipList) < 1 {
		return fmt.Errorf("no floating IP found for port %s", portID)
	}

	err = floatingips.Delete(nlbHandler.NetworkClient, fipList[0].ID).ExtractErr()
	if err != nil {
		return fmt.Errorf("failed to delete floating IP: %s", err)
	}
	return nil
}

func (nlbHandler *NhnCloudNLBHandler) getListenerInfo(raw map[string]interface{}) (irs.ListenerInfo, error) {
	listeners, ok := raw["listeners"].([]interface{})
	if !ok || len(listeners) < 1 {
		return irs.ListenerInfo{}, fmt.Errorf("not Exist Listener")
	}

	listener := listeners[0].(map[string]interface{})

	return irs.ListenerInfo{
		Protocol: fmt.Sprintf("%v", listener["protocol"]),
		Port:     fmt.Sprintf("%v", listener["protocol_port"]),
		IP:       fmt.Sprintf("%v", raw["vip_address"]),
		CspID:    fmt.Sprintf("%v", listener["id"]),
	}, nil
}

func (nlbHandler *NhnCloudNLBHandler) getHealthCheckerInfo(raw map[string]interface{}) (irs.HealthCheckerInfo, error) {
	poolID, err := nlbHandler.getDefaultPoolIDFromNLB(raw)
	if err != nil {
		return irs.HealthCheckerInfo{}, err
	}

	pool, err := nlbHandler.getDefaultPool(poolID)
	if err != nil {
		return irs.HealthCheckerInfo{}, err
	}

	if pool.MonitorID == "" {
		return irs.HealthCheckerInfo{}, fmt.Errorf("no monitor attached to pool %s", pool.ID)
	}

	monitor, err := nlbHandler.getRawMonitorById(pool.MonitorID)
	if err != nil {
		return irs.HealthCheckerInfo{}, err
	}

	return irs.HealthCheckerInfo{
		Protocol:  monitor.Type,
		Interval:  monitor.Delay,
		Timeout:   monitor.Timeout,
		Threshold: monitor.MaxRetries,
		Port:      strconv.Itoa(pool.MemberPort),
		CspID:     monitor.ID,
	}, nil
}

func (nlbHandler *NhnCloudNLBHandler) setterNLB(raw map[string]interface{}) (irs.NLBInfo, error) {
	var nlbInfo irs.NLBInfo

	nlbInfo.IId = irs.IID{
		NameId:   fmt.Sprintf("%v", raw["name"]),
		SystemId: fmt.Sprintf("%v", raw["id"]),
	}
	nlbInfo.VpcIID = irs.IID{SystemId: fmt.Sprintf("%v", raw["vip_subnet_id"])}
	nlbInfo.Type = fmt.Sprintf("%v", raw["loadbalancer_type"])
	nlbInfo.Scope = "REGION"
	nlbInfo.CreatedTime = time.Now()

	listenInfo, err := nlbHandler.getListenerInfo(raw)
	if err == nil {
		nlbInfo.Listener = listenInfo
	}

	vmGroup, err := nlbHandler.getVMGroup(nlbInfo.IId)
	if err == nil {
		nlbInfo.VMGroup = vmGroup
	}

	healthInfo, err := nlbHandler.getHealthCheckerInfo(raw)
	if err == nil {
		nlbInfo.HealthChecker = healthInfo
	}

	var lb loadbalancers.LoadBalancer
	jsonBytes, _ := json.Marshal(raw)
	_ = json.Unmarshal(jsonBytes, &lb)

	nlbInfo.KeyValueList = irs.StructToKeyValueList(lb)

	return nlbInfo, nil
}

func (nlbHandler *NhnCloudNLBHandler) attachPoolMembers(vmIIDs []irs.IID, port string, poolID string) ([]pools.Member, error) {
	portNum, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("invalid port %s", port)
	}

	var members []pools.Member
	for _, vm := range vmIIDs {
		privateIP, subnetID, err := nlbHandler.getNetInfoWithVMName(vm.NameId)
		if err != nil {
			return nil, fmt.Errorf("failed to get net info for VM %s: %v", vm.NameId, err)
		}

		createOpts := pools.CreateMemberOpts{
			SubnetID:     subnetID,
			Address:      privateIP,
			ProtocolPort: portNum,
			Weight:       DefaultWeight,
			AdminStateUp: DefaultAdminStateUp,
		}

		member, err := pools.CreateMember(nlbHandler.NetworkClient, poolID, createOpts).Extract()
		if err != nil {
			return nil, fmt.Errorf("failed to attach member for VM %s: %v", vm.NameId, err)
		}
		members = append(members, *member)
	}
	return members, nil
}

func (nlbHandler *NhnCloudNLBHandler) detachPoolMembers(poolID string, vmIIDs []irs.IID) error {
	members, err := nlbHandler.getRawPoolMembersById(poolID)
	if err != nil {
		return err
	}

	for _, vm := range vmIIDs {
		for _, m := range *members {
			if strings.EqualFold(m.Address, vm.NameId) {
				if err := pools.DeleteMember(nlbHandler.NetworkClient, poolID, m.ID).ExtractErr(); err != nil {
					return fmt.Errorf("failed to detach member %s: %v", m.ID, err)
				}
			}
		}
	}
	return nil
}

func (nlbHandler *NhnCloudNLBHandler) AddVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (irs.VMGroupInfo, error) {
	vmGroup, err := nlbHandler.getVMGroup(nlbIID)
	if err != nil {
		return irs.VMGroupInfo{}, err
	}

	for _, cur := range *vmGroup.VMs {
		for _, add := range *vmIIDs {
			if strings.EqualFold(cur.NameId, add.NameId) {
				return irs.VMGroupInfo{}, fmt.Errorf("VM %s already exists in pool", add.NameId)
			}
		}
	}

	if _, err := nlbHandler.attachPoolMembers(*vmIIDs, vmGroup.Port, vmGroup.CspID); err != nil {
		return irs.VMGroupInfo{}, err
	}

	return nlbHandler.getVMGroup(nlbIID)
}

func (nlbHandler *NhnCloudNLBHandler) RemoveVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {
	vmGroup, err := nlbHandler.getVMGroup(nlbIID)
	if err != nil {
		return false, err
	}

	if err := nlbHandler.detachPoolMembers(vmGroup.CspID, *vmIIDs); err != nil {
		return false, err
	}
	return true, nil
}

func (nlbHandler *NhnCloudNLBHandler) GetNLB(iid irs.IID) (irs.NLBInfo, error) {
	cblogger.Info("NHN Cloud Driver: called GetNLB()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", iid.SystemId, "GetNLB()")

	callLogStart := calllog.Start()
	rawNLB, err := nlbHandler.getRawNLB(iid)
	if err != nil {
		newErr := fmt.Errorf("Failed to Get NLB. : [%v]", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return irs.NLBInfo{}, newErr
	}

	LoggingInfo(callLogInfo, callLogStart)
	return nlbHandler.setterNLB(rawNLB)
}

func (nlbHandler *NhnCloudNLBHandler) getFirstSubnetAndNetworkId(vpcName string) (string, string, error) {
	listOpts := networks.ListOpts{Name: vpcName}
	allPages, err := networks.List(nlbHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		return "", "", fmt.Errorf("failed to list networks: %v", err)
	}

	var rawNetworks []map[string]interface{}
	if err := networks.ExtractNetworksInto(allPages, &rawNetworks); err != nil {
		return "", "", fmt.Errorf("failed to extract raw networks: %v", err)
	}
	if len(rawNetworks) < 1 {
		return "", "", fmt.Errorf("not found vpc: %s", vpcName)
	}

	first := rawNetworks[0]
	networkID := fmt.Sprintf("%v", first["id"])

	subnets, ok := first["subnets"].([]interface{})
	if !ok || len(subnets) < 1 {
		return "", "", fmt.Errorf("no subnet found in vpc: %s", vpcName)
	}

	subnetID := fmt.Sprintf("%v", subnets[0])
	return subnetID, networkID, nil
}

func (nlbHandler *NhnCloudNLBHandler) getNetInfoWithVMName(vmName string) (string, string, error) {
	listOpts := servers.ListOpts{Name: vmName}
	allPages, err := servers.List(nlbHandler.VMClient, listOpts).AllPages()
	if err != nil {
		return "", "", fmt.Errorf("failed to list servers: %v", err)
	}

	serverList, err := servers.ExtractServers(allPages)
	if err != nil {
		return "", "", fmt.Errorf("failed to extract servers: %v", err)
	}
	if len(serverList) < 1 {
		return "", "", fmt.Errorf("vm not found: %s", vmName)
	}
	server := serverList[0]

	var privateIP string
	for _, addrSet := range server.Addresses {
		for _, addr := range addrSet.([]interface{}) {
			addrMap := addr.(map[string]interface{})
			if addrMap["OS-EXT-IPS:type"] == "fixed" {
				privateIP = addrMap["addr"].(string)
				break
			}
		}
	}

	if privateIP == "" {
		return "", "", fmt.Errorf("failed to get private ip for vm: %s", vmName)
	}

	port, err := getPortWithDeviceId(nlbHandler.NetworkClient, server.ID)
	if err != nil {
		return "", "", fmt.Errorf("failed to get port info: %v", err)
	}
	if len(port.FixedIPs) < 1 {
		return "", "", fmt.Errorf("no fixed ip for vm: %s", vmName)
	}
	return privateIP, port.FixedIPs[0].SubnetID, nil
}

func (nlbHandler *NhnCloudNLBHandler) waitToGetNLBInfo(nlbIID irs.IID) (bool, error) {
	maxRetry := 240
	for i := 0; i < maxRetry; i++ {
		raw, err := nlbHandler.getRawNLB(nlbIID)
		if err != nil {
			return false, err
		}

		status := fmt.Sprintf("%v", raw["provisioning_status"])
		if strings.ToUpper(status) == "ACTIVE" {
			return true, nil
		}
		if strings.ToUpper(status) == "ERROR" {
			return false, fmt.Errorf("failed to create NLB %s (ProvisioningStatus=ERROR)", nlbIID.SystemId)
		}
		time.Sleep(3 * time.Second)
	}
	return false, fmt.Errorf("timeout waiting for NLB %s to be ACTIVE", nlbIID.SystemId)
}

func (nlbHandler *NhnCloudNLBHandler) DeleteNLB(nlbIID irs.IID) (bool, error) {
	cblogger.Info("NHN Cloud Driver: called DeleteNLB()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.NameId, "DeleteNLB()")
	callLogStart := calllog.Start()

	if err := nlbHandler.checkNLBClient(); err != nil {
		newErr := fmt.Errorf("failed to delete NLB: %s", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	if err := nlbHandler.cleanUpNLB(nlbIID); err != nil {
		newErr := fmt.Errorf("failed to delete NLB: %s", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return false, newErr
	}

	LoggingInfo(callLogInfo, callLogStart)
	return true, nil
}

func (nlbHandler *NhnCloudNLBHandler) checkDeletable(nlbIID irs.IID) (bool, error) {
	raw, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		return false, err
	}

	status := fmt.Sprintf("%v", raw["provisioning_status"])
	if strings.ToUpper(status) == "PENDING_CREATE" {
		return false, errors.New("cannot delete NLB while ProvisioningStatus is PENDING_CREATE")
	}
	return true, nil
}

func (nlbHandler *NhnCloudNLBHandler) cleanUpNLB(nlbIID irs.IID) error {
	cblogger.Info("NHN Cloud Driver: called cleanUpNLB()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", nlbIID.SystemId, "cleanUpNLB()")
	callLogStart := calllog.Start()

	rawnlb, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		newErr := fmt.Errorf("failed to get RawNLB: %s", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return newErr
	}

	lbID, ok := rawnlb["id"].(string)
	if !ok || lbID == "" {
		newErr := fmt.Errorf("invalid NLB id in raw data")
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return newErr
	}

	if _, err := nlbHandler.checkDeletable(nlbIID); err != nil {
		newErr := fmt.Errorf("NLB is not deletable: %s", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return newErr
	}

	if floatingIP, err := nlbHandler.getNLBRawPublicIP(nlbIID); err == nil {
		if delErr := floatingips.Delete(nlbHandler.NetworkClient, floatingIP.ID).ExtractErr(); delErr != nil {
			newErr := fmt.Errorf("failed to delete floating IP %s: %s", floatingIP.ID, delErr)
			cblogger.Error(newErr.Error())
			LoggingError(callLogInfo, newErr)
			return newErr
		}
	}

	if err := loadbalancers.Delete(
		nlbHandler.NetworkClient,
		lbID,
		loadbalancers.DeleteOpts{Cascade: true},
	).ExtractErr(); err != nil {
		newErr := fmt.Errorf("failed to delete loadbalancer %s: %s", lbID, err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return newErr
	}

	LoggingInfo(callLogInfo, callLogStart)
	return nil
}

func (nlbHandler *NhnCloudNLBHandler) getNLBRawPublicIP(nlbIID irs.IID) (floatingips.FloatingIP, error) {
	raw, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		return floatingips.FloatingIP{}, err
	}

	externVPCID, err := getPublicVPCInfo(nlbHandler.NetworkClient, "ID")
	if err != nil {
		return floatingips.FloatingIP{}, err
	}

	listOpt := floatingips.ListOpts{
		FloatingNetworkID: externVPCID,
		PortID:            fmt.Sprintf("%v", raw["vip_port_id"]),
	}
	pager, err := floatingips.List(nlbHandler.NetworkClient, listOpt).AllPages()
	if err != nil {
		return floatingips.FloatingIP{}, err
	}

	all, err := floatingips.ExtractFloatingIPs(pager)
	if err != nil {
		return floatingips.FloatingIP{}, err
	}

	if len(all) > 0 {
		if strings.EqualFold(all[0].PortID, fmt.Sprintf("%v", raw["vip_port_id"])) {
			return all[0], nil
		}
	}
	return floatingips.FloatingIP{}, errors.New("not found floatingIP")
}

func (nlbHandler *NhnCloudNLBHandler) getRawVPCByName(vpcName string) (*vpcs.VPC, error) {
	listOpts := vpcs.ListOpts{
		Name: vpcName,
	}

	allPages, err := vpcs.List(nlbHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		return nil, fmt.Errorf("failed to list VPCs: %w", err)
	}

	vpcList, err := vpcs.ExtractVPCs(allPages)
	if err != nil {
		return nil, fmt.Errorf("failed to extract VPCs: %w", err)
	}

	for _, vpc := range vpcList {
		if vpc.Name == vpcName {
			return &vpc, nil
		}
	}
	return nil, fmt.Errorf("not found vpc with name %s", vpcName)
}

func (nlbHandler *NhnCloudNLBHandler) ListIID() ([]*irs.IID, error) {
	cblogger.Info("NHN Cloud Driver: called ListIID()")
	callLogInfo := getCallLogScheme(nlbHandler.RegionInfo.Region, "NETWORKLOADBALANCE", "ListIID()", "ListIID()")
	callLogStart := calllog.Start()

	listOpts := loadbalancers.ListOpts{}
	allPages, err := loadbalancers.List(nlbHandler.NetworkClient, listOpts).AllPages()
	if err != nil {
		newErr := fmt.Errorf("failed to list NLBs: %v", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	nlbList, err := loadbalancers.ExtractLoadBalancers(allPages)
	if err != nil {
		newErr := fmt.Errorf("failed to extract NLB list: %v", err)
		cblogger.Error(newErr.Error())
		LoggingError(callLogInfo, newErr)
		return nil, newErr
	}

	var iidList []*irs.IID
	for _, lb := range nlbList {
		iidList = append(iidList, &irs.IID{
			NameId:   lb.Name,
			SystemId: lb.ID,
		})
	}

	LoggingInfo(callLogInfo, callLogStart)
	return iidList, nil
}

func (nlbHandler *NhnCloudNLBHandler) getVMGroup(nlbIID irs.IID) (irs.VMGroupInfo, error) {
	rawLB, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		return irs.VMGroupInfo{}, fmt.Errorf("failed to get raw NLB: %v", err)
	}

	listeners, ok := rawLB["listeners"].([]interface{})
	if !ok || len(listeners) < 1 {
		return irs.VMGroupInfo{}, fmt.Errorf("no listener found for NLB %s", nlbIID.SystemId)
	}

	listenerID := listeners[0].(map[string]interface{})["id"].(string)

	listener, err := nlbHandler.getRawListenerById(listenerID)
	if err != nil {
		return irs.VMGroupInfo{}, fmt.Errorf("failed to get listener %s: %v", listenerID, err)
	}

	if listener.DefaultPoolID == "" {
		return irs.VMGroupInfo{}, fmt.Errorf("no default pool for listener %s", listenerID)
	}

	pool, err := nlbHandler.getRawPoolById(listener.DefaultPoolID)
	if err != nil {
		return irs.VMGroupInfo{}, err
	}

	members, err := nlbHandler.getRawPoolMembersById(pool.ID)
	if err != nil {
		return irs.VMGroupInfo{}, err
	}

	var vmIIDs []irs.IID
	for _, m := range *members {
		vmIIDs = append(vmIIDs, irs.IID{
			SystemId: m.ID,
			NameId:   m.Address,
		})
	}

	return irs.VMGroupInfo{
		Protocol: pool.Protocol,
		Port:     strconv.Itoa(pool.MemberPort),
		VMs:      &vmIIDs,
		CspID:    pool.ID,
	}, nil
}

func (nlbHandler *NhnCloudNLBHandler) getDefaultPoolIDFromNLB(raw map[string]interface{}) (string, error) {
	listeners, ok := raw["listeners"].([]interface{})
	if !ok || len(listeners) < 1 {
		return "", fmt.Errorf("no listener found for NLB %v", raw["id"])
	}

	listener := listeners[0].(map[string]interface{})
	poolID, ok := listener["default_pool_id"].(string)
	if !ok || poolID == "" {
		return "", fmt.Errorf("no default pool for listener %v", listener["id"])
	}
	return poolID, nil
}

func (nlbHandler *NhnCloudNLBHandler) getDefaultPool(poolID string) (*pools.Pool, error) {
	if poolID == "" {
		return nil, fmt.Errorf("invalid pool ID")
	}
	return nlbHandler.getRawPoolById(poolID)
}

func (nlbHandler *NhnCloudNLBHandler) GetVMGroupHealthInfo(nlbIID irs.IID) (irs.HealthInfo, error) {
	cblogger.Info("NHN Cloud Driver: called GetVMGroupHealthInfo()")

	var healthInfo irs.HealthInfo

	rawLB, err := nlbHandler.getRawNLB(nlbIID)
	if err != nil {
		return healthInfo, fmt.Errorf("failed to get NLB: %v", err)
	}

	listeners, ok := rawLB["listeners"].([]interface{})
	if !ok || len(listeners) < 1 {
		return healthInfo, fmt.Errorf("no listener found for NLB %s", nlbIID.SystemId)
	}

	listenerID := listeners[0].(map[string]interface{})["id"].(string)

	listener, err := nlbHandler.getRawListenerById(listenerID)
	if err != nil {
		return healthInfo, fmt.Errorf("failed to get listener %s: %v", listenerID, err)
	}

	if listener.DefaultPoolID == "" {
		return healthInfo, fmt.Errorf("no default pool found for listener %s", listenerID)
	}

	members, err := nlbHandler.getRawPoolMembersById(listener.DefaultPoolID)
	if err != nil {
		return healthInfo, fmt.Errorf("failed to get pool members: %v", err)
	}

	var allVMs, healthyVMs, unHealthyVMs []irs.IID
	for _, m := range *members {
		vmIID := irs.IID{SystemId: m.ID, NameId: m.Address}
		allVMs = append(allVMs, vmIID)

		if strings.ToUpper(m.OperatingStatus) == "ONLINE" {
			healthyVMs = append(healthyVMs, vmIID)
		} else {
			unHealthyVMs = append(unHealthyVMs, vmIID)
		}
	}

	healthInfo.AllVMs = &allVMs
	healthInfo.HealthyVMs = &healthyVMs
	healthInfo.UnHealthyVMs = &unHealthyVMs

	return healthInfo, nil
}
