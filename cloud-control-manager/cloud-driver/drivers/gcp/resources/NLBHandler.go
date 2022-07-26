package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"github.com/davecgh/go-spew/spew"
	compute "google.golang.org/api/compute/v1"
	"google.golang.org/api/googleapi"
	"strconv"
	"strings"
	"time"
)

/**
Adderess(LB) -> pool(backend) -> firewallrule(Listener)
*/
type GCPNLBHandler struct {
	Region     idrv.RegionInfo
	Ctx        context.Context
	Client     *compute.Service
	Credential idrv.CredentialInfo
}

type Tag struct {
	Items []string
}

type AccessConfig struct {
	Kind                     string
	Type                     string //enum
	Name                     string
	NatIP                    string
	ExternalIpv6             string
	ExternalIpv6PrefixLength int
	SetPublicPtr             bool
	PublicPtrDomainName      string
	NetworkTier              string // enum
}
type AliasIpRange struct {
	IpCidrRange         string
	SubnetworkRangeName string
}
type NetworkInterface struct {
	Kind                     string
	Network                  string
	Subnetwork               string
	NetworkIP                string
	Ipv6Address              string
	InternalIpv6PrefixLength int
	Name                     string
	AccessConfigs            []AccessConfig
	Ipv6AccessConfigs        []AccessConfig
	AliasIpRanges            []AliasIpRange
	Fingerprint              string
	StackType                string // enum,
	Ipv6AccessType           string // enum,
	QueueCount               int
	NicType                  string // enum
}
type Label struct {
	String string
}
type SourceEncryptionKey struct {
	Sha256               string
	MmsKeyServiceAccount string

	RawKey          string
	RsaEncryptedKey string
	KmsKeyName      string
}
type GuestOsFeature struct {
	Type string // enum
}
type DiskEncryptionKey struct {
	RawKey               string
	RsaEncryptedKey      string
	KmsKeyName           string
	Sha256               string
	KmsKeyServiceAccount string
}
type ContentAndFileType struct {
	Content  string
	FileType string //enum
}
type ShieldedInstanceInitialState struct {
	Pk   ContentAndFileType
	Keys []ContentAndFileType
	Dbs  []ContentAndFileType
	Dbxs []ContentAndFileType
}
type InitializeParam struct {
	DiskName                    string
	SourceImage                 string
	DiskSizeGb                  string
	DiskType                    string
	SourceImageEncryptionKey    SourceEncryptionKey
	Labels                      Label
	SourceSnapshot              string
	SourceSnapshotEncryptionKey SourceEncryptionKey
	Description                 string
	ResourcePolicies            []string
	OnUpdateAction              string // enum
	ProvisionedIops             string
	Licenses                    []string
}

type Disk struct {
	Kind                         string
	Type                         string //enum,
	Mode                         string //": enum,
	Source                       string
	DeviceName                   string
	Index                        int
	Boot                         bool
	InitializeParams             InitializeParam
	AutoDelete                   bool
	Licenses                     []string
	Interface                    string //enum
	GuestOsFeatures              []GuestOsFeature
	DiskEncryptionKey            DiskEncryptionKey
	DiskSizeGb                   string
	ShieldedInstanceInitialState ShieldedInstanceInitialState
}
type KeyValue struct {
	Key   string
	Value string
}
type Metadata struct {
	Kind        string
	Fingerprint string
	Items       []KeyValue
}
type ServiceAccount struct {
	Email  string
	Scopes []string
}
type NodeAffinity struct {
	Key      string
	Operator string //enum
	Values   []string
}
type Scheduling struct {
	OnHostMaintenance         string // enum,
	AutomaticRestart          bool
	Preemptible               bool
	NodeAffinities            []NodeAffinity
	MinNodeCpus               int
	LocationHint              string
	ProvisioningModel         string // enum,
	InstanceTerminationAction string // enum
}
type GuestAccelerator struct {
	AcceleratorType  string
	AcceleratorCount int
}
type RevervationAffinity struct {
	ConsumeReservationType string // enum,
	Key                    string
	Values                 []string
}
type ShieldedInstanceConfig struct {
	EnableSecureBoot          bool
	EnableVtpm                bool
	EnableIntegrityMonitoring bool
}
type ConfidentialInstanceConfig struct {
	EnableConfidentialCompute bool
}
type AdvancedMachineFeatures struct {
	EnableNestedVirtualization bool
	ThreadsPerCore             int
	EnableUefiNetworking       bool
}
type NetworkPerformanceConfig struct {
	TotalEgressBandwidthTier string //enum
}
type Property struct {
	Description                string
	Tags                       Tag
	Fingerprint                string
	ResourceManagerTags        string // struct인가?
	MachineType                string
	CanIpForward               bool
	NetworkInterfaces          []NetworkInterface
	Disks                      []Disk
	Metadata                   Metadata
	ServiceAccounts            []ServiceAccount
	Scheduling                 Scheduling
	Label                      Label
	GuestAccelerators          []GuestAccelerator
	MinCpuPlatform             string
	RevervationAffinity        RevervationAffinity
	ShieldedInstanceConfig     ShieldedInstanceConfig
	ResourcePolicies           []string
	ConfidentialInstanceConfig ConfidentialInstanceConfig
	PrivateIpv6GoogleAccess    string // enum
	AdvancedMachineFeatures    AdvancedMachineFeatures
	NetworkPerformanceConfig   NetworkPerformanceConfig
}
type DiskConfig struct {
	DeviceName      string
	InstantiateFrom string // enum,
	AutoDelete      bool
	CustomImage     string
}
type SourceInstanceParam struct {
	DiskConfigs []DiskConfig
}
type InstanceTemplateInfo struct {
	Kind                 string
	Id                   string
	CreationTimestamp    string
	Name                 string
	Description          string
	Properties           Property
	SelfLink             string
	SourceInstance       string
	SourceInstanceParams SourceInstanceParam
}

const (
	//HealthCheck_types : TCP, SSL, HTTP, HTTPS
	HealthCheck_Http  string = "HTTP"
	HealthCheck_Https string = "HTTPS"
	HealthCheck_Http2 string = "HTTP2"
	HealthCheck_TCP   string = "TCP"
	HealthCheck_SSL   string = "SSL"

	HealthState_UNHEALTHY string = "UNHEALTHY"
	HealthState_HEALTHY   string = "HEALTHY"

	GCP_ForwardingRuleScheme_EXTERNAL = "EXTERNAL"
	SPIDER_LoadBalancerType_PUBLIC    = "PUBLIC"

	SCOPE_REGION = "REGION"
	SCOPE_GLOBAL = "GLOBAL"

	ErrorCode_NotFound = 404

	RequestStatus_DONE string = "DONE"

	StringSeperator_Slash string = "/"
	StringSeperator_Hypen string = "-"
	String_Empty          string = ""

	NLB_Component_HEALTHCHECKER  string = "HEALTHCHECKER"
	NLB_Component_TARGETPOOL     string = "TARGETPOOL"
	NLB_Component_FORWARDINGRULE string = "FORWARDINGRULE"
)

/*
// GCP는 동일 vpc가 아니어도 LB 생성가능, but Spider는 동일 vpc에 있어야하므로 사용할 instance 들이 동일한 VPC에 있는지 체크 필요
// 대상 풀 기반 외부 TCP/UDP 네트워크 부하 분산
// 아키텍쳐 : 대상 풀 1개, 여러 전달규칙 ( https://cloud.google.com/load-balancing/docs/network/networklb-target-pools?hl=ko )
// 1LNB = 1 Listener , 1 backend, 1 health checker

	// 방법 1. nameId = targetPoolName, systemId = forwardingRulename
	// * 방법 2. nameId = targetPoolName, systemId = targetPoolUrl
	//	targetPoolName = forwardingRule name 이므로 적당. 단, front-end 와 back-end가 1:1 이어야 함.
	// 방법 3. nameId = targetPoolUrl, systemId = forwardingRule name
	// 방법 4. nameId = targetPoolName, systemId = forwardingRule

	// url 형태가 필요한 resource에 대하여. 조회시에는 끝의 id만 , 실제 사용시에는 id를 바탕으로 url을 만들어 사용
	// url set이 가능한 parma은 cspID임.
*/
func (nlbHandler *GCPNLBHandler) CreateNLB(nlbReqInfo irs.NLBInfo) (irs.NLBInfo, error) {
	cblogger.Info("CreateNLB")
	projectID := nlbHandler.Credential.ProjectID
	regionID := nlbHandler.Region.Region

	nlbName := nlbReqInfo.IId.NameId
	cblogger.Info("start ", projectID)

	resultMap := make(map[string]string) // 에러 발생 시 자원 회수용

	//// validation check area
	err := nlbHandler.validateCreateNLB(nlbReqInfo)
	if err != nil {
		// 404면 없는게 맞으므로 진행
		is404, checkErr := checkErrorCode(ErrorCode_NotFound, err) // 404 : not found면 pass
		if is404 && checkErr {                                     // 하나라도 false 면 error return
			cblogger.Info("existsTargetPoolChecks : ", err)
		} else {
			cblogger.Info("validateCreateNLB ", err)
			return irs.NLBInfo{}, err
		}
	}

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   nlbHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: nlbName,
		CloudOSAPI:   "CreateNLB()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	// backend service == target pool
	// region forwarding rule, targetpool, targetpool안의 instance에서 사용하는 healthchecker

	cblogger.Info("backend TargetPool 생성 ")

	// 새로운 health checker 등록
	healthCheckerInfo := nlbReqInfo.HealthChecker
	err = nlbHandler.insertHealthCheck(regionID, nlbName, &healthCheckerInfo)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.NLBInfo{}, err // 첫 단계에서 에러 발생. return error
	}
	resultMap[NLB_Component_HEALTHCHECKER] = healthCheckerInfo.CspID //TargetLink
	cblogger.Info("insertTargetPoolHealthCheck -----")
	printToJson(healthCheckerInfo)
	nlbReqInfo.HealthChecker.CspID = healthCheckerInfo.CspID

	// targetPool 안에 healthCheckID가 들어 감.
	newTargetPool, err := nlbHandler.convertNlbInfoToTargetPool(&nlbReqInfo)
	if err != nil {
		cblogger.Info("targetPoolList convert err: ", err)
		// 이전 step 자원 회수 후 return
		resultMsg := nlbHandler.rollbackCreatedNlbResources(regionID, resultMap)
		resultMsg += resultMsg + "(2) TargetPool " + err.Error()
		return irs.NLBInfo{}, errors.New(resultMsg)
	}
	printToJson(newTargetPool)

	targetPool, err := nlbHandler.insertTargetPool(regionID, newTargetPool)
	if err != nil {
		cblogger.Info("targetPoolList  err: ", err)
		// 이전 step 자원 회수 후 return
		resultMsg := nlbHandler.rollbackCreatedNlbResources(regionID, resultMap)
		resultMsg += resultMsg + "(2) TargetPool " + err.Error()
		return irs.NLBInfo{}, errors.New(resultMsg)
	}
	resultMap[NLB_Component_TARGETPOOL] = targetPool.SelfLink
	printToJson(targetPool)
	cblogger.Info("backend TargetPool 생성 완료 ")

	cblogger.Info("frontend (forwarding rule) 생성 ")
	// targetPool 생성 후 selfLink 를 forwardingRule의 target으로 set.
	newForwardingRule := convertNlbInfoToForwardingRule(nlbReqInfo.Listener, targetPool)
	err = nlbHandler.insertRegionForwardingRules(regionID, &newForwardingRule)
	if err != nil {
		cblogger.Info("forwardingRule err  : ", err)
		// 이전 step 자원 회수 후 return
		resultMsg := nlbHandler.rollbackCreatedNlbResources(regionID, resultMap)
		resultMsg += resultMsg + "(3) Forwarding Rule " + err.Error()
		//return irs.NLBInfo{}, err
		return irs.NLBInfo{}, errors.New(resultMsg)
	}
	cblogger.Info("forwardingRule result  : ")

	//IId:         irs.IID{NameId: targetLbValue, SystemId: targetForwardingRuleValue}, // NameId = Lb Name, SystemId = forwardingRule name
	nlbIID := irs.IID{
		NameId:   nlbName,         // lb Name != targetPool name
		SystemId: targetPool.Name, // targetPool
	}
	nlbInfo, err := nlbHandler.GetNLB(nlbIID)
	if err != nil {
		resultMsg := "Successfully created NLB, but " + err.Error()
		return irs.NLBInfo{}, errors.New(resultMsg)
	}
	//
	//cblogger.Info("Targetpool end: ")
	printToJson(nlbInfo)
	return nlbInfo, nil
}

/*
 At the API level, there is no Load Balancer,
 only the components that make it up.
 Your best bet to get a view similar to the UI is to list forwarding rules (global and regional).
You can use gcloud compute forwarding-rules list which will show you all the forwarding rules in use (similar to the UI view),
along with the IPs of each and the target (which may be a backend service or a target pool).

 load balancer => GCP forwardingrules
 listener => GCP frontend
 vmGroup => GCP backend. vm instances target pull or instance group list
 healthchecker => GCP Healthchecker

- backend service 없음.
- region forwarding rule, targetpool, targetpool안의 instance에서 사용하는 healthchecker

 NLBInfo의 IID 에서 NameId = targetPool name, SystemId = targetPool Url

- VPC정보조회를 위해 INSTANCE 정보 조회 시 같은 region의 다른 zone은 가져오지 못함. getVPCInfoFromVM 으로 가져오도록 함.

*/
func (nlbHandler *GCPNLBHandler) ListNLB() ([]*irs.NLBInfo, error) {
	var nlbInfoList []*irs.NLBInfo

	//projectID := nlbHandler.Credential.ProjectID
	regionID := nlbHandler.Region.Region

	//nlbMap := make(map[string]irs.NLBInfo)

	// 외부 dns주소가 있는 경우 region/global에서 해당 주소를 가져온다.
	//cblogger.Info("region address start: ", regionID)
	//regionAddressList, err := nlbHandler.listRegionAddresses(regionID, "")
	//if err != nil {
	//	cblogger.Info("region address  list: ", err)
	//}
	//printToJson(regionAddressList)
	//cblogger.Info("region address end: ")
	//cblogger.Info("global address start: ")
	//globalAddressList, err := nlbHandler.listGlobalAddresses("")
	//if err != nil {
	//	cblogger.Info("globalAddressList  list: ", err)
	//}
	//printToJson(globalAddressList)
	//cblogger.Info("global address end: ")

	// region forwarding rule 는 target pool 과 lb이름으로 엮임.
	// map에 nb이름으로 nbInfo를 넣고 해당 값들 추가해서 조합

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   nlbHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: "ListNLB",
		CloudOSAPI:   "ListNLB()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	cblogger.Info("region forwardingRules start: ", regionID)
	regionForwardingRuleList, err := nlbHandler.listRegionForwardingRules(regionID, String_Empty, String_Empty)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		cblogger.Info("regionForwardingRule  list: ", err)
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return nil, err
	}

	if regionForwardingRuleList != nil {
		for _, forwardingRule := range regionForwardingRuleList.Items {
			targetPoolUrl := forwardingRule.Target
			targetLbIndex := strings.LastIndex(targetPoolUrl, StringSeperator_Slash)
			targetLbValue := forwardingRule.Target[(targetLbIndex + 1):]
			nlbInfo, err := nlbHandler.GetNLB(irs.IID{NameId: "", SystemId: targetLbValue})
			if err != nil {
				// 에러가 났어도 다음 nlb 조회
				cblogger.Info("getNLB error. targetPoolUrl = " + targetPoolUrl)
				cblogger.Error(err)
			} else {
				nlbInfoList = append(nlbInfoList, &nlbInfo)
			}
		}
	}

	//if regionForwardingRuleList != nil { // dial tcp: lookup compute.googleapis.com: no such host 일 때, 	panic: runtime error: invalid memory address or nil pointer dereference
	//	if len(regionForwardingRuleList.Items) > 0 {
	//		for _, forwardingRule := range regionForwardingRuleList.Items {
	//			targetPoolUrl := forwardingRule.Target
	//			targetLbIndex := strings.LastIndex(targetPoolUrl, StringSeperator_Slash)
	//			targetLbValue := forwardingRule.Target[(targetLbIndex + 1):]
	//
	//			// targetlink에서 lb 추출
	//			//targetNlbInfo := nlbMap[targetLbValue]
	//			newNlbInfo, exists := nlbMap[targetLbValue]
	//			if exists {
	//				// spider는 1개의 listener(forwardingrule)만 사용하므로 skip
	//			} else {
	//				listenerInfo := convertRegionForwardingRuleToNlbListener(forwardingRule)
	//
	//				createdTime, _ := time.Parse(
	//					time.RFC3339,
	//					forwardingRule.CreationTimestamp) // RFC3339형태이므로 해당 시간으로 다시 생성
	//
	//				loadBalancerType := forwardingRule.LoadBalancingScheme
	//				if strings.EqualFold(loadBalancerType, GCP_ForwardingRuleScheme_EXTERNAL) { // PUBLIC/INTERNAL : extenel -> PUBLIC으로 변경
	//					loadBalancerType = SPIDER_LoadBalancerType_PUBLIC
	//				}
	//
	//				newNlbInfo = irs.NLBInfo{
	//					//IId:         irs.IID{NameId: targetLbValue, SystemId: targetPoolUrl}, // NameId :Lb Name, poolName, SystemId : targetPool Url
	//					IId:         irs.IID{NameId: targetLbValue, SystemId: targetLbValue},
	//					VpcIID:      irs.IID{NameId: String_Empty, SystemId: String_Empty}, // VpcIID 는 Pool 안의 instance에 있는 값
	//					Type:        loadBalancerType,                                      // PUBLIC/INTERNAL : extenel -> PUBLIC으로 변경하는 로직 적용해야함.
	//					Scope:       SCOPE_REGION,
	//					Listener:    listenerInfo,
	//					CreatedTime: createdTime, //RFC3339 "creationTimestamp":"2022-05-24T01:20:40.334-07:00"
	//					//KeyValueList  []KeyValue
	//				}
	//
	//				newNlbInfo.VMGroup.Protocol = forwardingRule.IPProtocol
	//				newNlbInfo.VMGroup.Port = forwardingRule.PortRange
	//			}
	//			nlbMap[targetLbValue] = newNlbInfo
	//			printToJson(forwardingRule)
	//		}
	//	}
	//}
	//cblogger.Info("regionForwardingRule end: ")
	//cblogger.Info("nlbMap end: ", nlbMap)
	//
	//cblogger.Info("Targetpool start: ")
	//
	//targetPoolList, err := nlbHandler.listTargetPools(regionID, String_Empty)
	//if err != nil {
	//	cblogger.Info("targetPoolList  list: ", err)
	//	return nil, err
	//}
	//printToJson(targetPoolList)
	//
	////vpcInstanceName := "" // vpc를 갸져올 instance 이름
	////vpcInstanceZone := "" // vpc를 가져올 instance zone
	//
	//for _, targetPool := range targetPoolList.Items {
	//	//printToJson(targetPool)
	//	newNlbInfo, exists := nlbMap[targetPool.Name] // lb name
	//	if !exists {
	//		// 없으면 안됨.
	//		cblogger.Info("targetPool.Name does not exist in nlbMap ", targetPool.Name)
	//		continue
	//	}
	//	err = nlbHandler.convertTargetPoolToNLBInfo(targetPool, &newNlbInfo)
	//
	//	if err != nil {
	//		return nil, err
	//	}
	//	nlbMap[targetPool.Name] = newNlbInfo
	//}
	////printToJson(targetPoolList)
	//
	//cblogger.Info("Targetpool end: ")
	//printToJson(nlbMap)
	//
	//for _, nlbInfo := range nlbMap {
	//	cblogger.Info(nlbInfo)
	//	nlbInfoList = append(nlbInfoList, &nlbInfo)
	//}
	return nlbInfoList, nil
}

// Load balancer 조회
// nlbIID 에서 NameId = lbName, targetPoolName, forwardingRuleName
func (nlbHandler *GCPNLBHandler) GetNLB(nlbIID irs.IID) (irs.NLBInfo, error) {
	nlbInfo := irs.NLBInfo{}

	projectID := nlbHandler.Credential.ProjectID
	regionID := nlbHandler.Region.Region

	cblogger.Info("projectID: ", projectID)
	cblogger.Info("regionID: ", regionID)
	printToJson(nlbInfo)
	// region forwarding rule 는 target pool 과 lb이름으로 엮임.
	// map에 nb이름으로 nbInfo를 넣고 해당 값들 추가해서 조합
	//nlbID := nlbIID.NameId
	nlbID := nlbIID.SystemId

	// forwardingRule 조회
	listenerInfo, err := nlbHandler.getListenerByNlbSystemID(nlbIID)
	if err != nil {
		cblogger.Error(err)
		return irs.NLBInfo{}, err
	}

	nlbInfo.IId = nlbIID
	nlbInfo.VpcIID = irs.IID{NameId: String_Empty, SystemId: String_Empty} // VpcIID 는 Pool 안의 instance에 있는 값
	nlbInfo.Scope = SCOPE_REGION
	nlbInfo.Listener = listenerInfo

	for _, keyValue := range listenerInfo.KeyValueList {
		if strings.EqualFold(keyValue.Key, "createdTime") {
			createdTime, _ := time.Parse(
				time.RFC3339,
				keyValue.Value) // RFC3339형태이므로 해당 시간으로 다시 생성
			nlbInfo.CreatedTime = createdTime //RFC3339 "creationTimestamp":"2022-05-24T01:20:40.334-07:00"
		}
		if strings.EqualFold(keyValue.Key, "loadBalancerType") {
			nlbInfo.Type = keyValue.Value
		}
	}

	//cblogger.Info("region forwardingRules start: ", regionID)
	//callogger := call.GetLogger("HISCALL")
	//callLogInfo := call.CLOUDLOGSCHEMA{
	//	CloudOS:      call.GCP,
	//	RegionZone:   nlbHandler.Region.Zone,
	//	ResourceType: call.NLB,
	//	ResourceName: nlbID,
	//	CloudOSAPI:   "GetNLB()",
	//	ElapsedTime:  "",
	//	ErrorMSG:     "",
	//}
	//callLogStart := call.Start()
	//regionForwardingRule, err := nlbHandler.getRegionForwardingRules(regionID, nlbID)
	//callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	//if err != nil {
	//	cblogger.Info("regionForwardingRule  list: ", err)
	//	callLogInfo.ErrorMSG = err.Error()
	//	callogger.Info(call.String(callLogInfo))
	//	cblogger.Error(err)
	//	return irs.NLBInfo{}, err
	//}
	//if regionForwardingRule != nil { // dial tcp: lookup compute.googleapis.com: no such host 일 때, 	panic: runtime error: invalid memory address or nil pointer dereference
	//
	//	listenerInfo := convertRegionForwardingRuleToNlbListener(regionForwardingRule)
	//
	//	createdTime, _ := time.Parse(
	//		time.RFC3339,
	//		regionForwardingRule.CreationTimestamp) // RFC3339형태이므로 해당 시간으로 다시 생성
	//
	//	loadBalancerType := regionForwardingRule.LoadBalancingScheme
	//	if strings.EqualFold(loadBalancerType, GCP_ForwardingRuleScheme_EXTERNAL) {
	//		loadBalancerType = SPIDER_LoadBalancerType_PUBLIC
	//	}
	//
	//	nlbInfo = irs.NLBInfo{
	//		IId:         nlbIID,
	//		VpcIID:      irs.IID{NameId: String_Empty, SystemId: String_Empty}, // VpcIID 는 Pool 안의 instance에 있는 값
	//		Type:        loadBalancerType,                                      // PUBLIC/INTERNAL : extenel -> PUBLIC으로 변경하는 로직 적용해야함.
	//		Scope:       SCOPE_REGION,
	//		Listener:    listenerInfo,
	//		CreatedTime: createdTime, //RFC3339 "creationTimestamp":"2022-05-24T01:20:40.334-07:00"
	//		//KeyValueList  []KeyValue
	//	}
	//	printToJson(regionForwardingRule)
	//}

	//cblogger.Info("forwardingRules result size  : ", len(regionForwardingRuleList.Items))
	cblogger.Info("regionForwardingRule end: ")

	cblogger.Info("Targetpool start: ")

	targetPool, err := nlbHandler.getTargetPool(regionID, nlbID)
	if err != nil {
		cblogger.Info("targetPoolList  list: ", err)
		return irs.NLBInfo{}, err
	}

	// vms, health checker, vpc,
	err = nlbHandler.convertTargetPoolToNLBInfo(targetPool, &nlbInfo)
	if err != nil {
		return irs.NLBInfo{}, err
	}

	cblogger.Info("Targetpool end: ")

	nlbInfo.VMGroup.Protocol = listenerInfo.Protocol
	nlbInfo.VMGroup.Port = listenerInfo.Port
	printToJson(nlbInfo)

	return nlbInfo, nil
}

/*
// NLB 삭제.
// delete 는 forwardingRule -> targetPool순으로 삭제. (healthchecker는 어디에 있어도 상관없음.)
// targetPool을 먼저 삭제하면 Error 400: The target_pool resource 'xxx' is already being used by 'yyy', resourceInUseByAnotherResource
// 두 개가 transaction으로 묶이지 않기 때문에 비정상적인 상태로 존재 가능
// 이 경우에 다시 삭제 요청이 들어 왔을 때 기존에 지워진 것은 skip하고 있는 것만 삭제

// ex) frontend는 삭제되고 targetPool이 어떤이유에서 삭제가 되지 않았을 때,
// 다음 시도에서 삭제

	삭제 시도 시 404 Error인 경우는 이미 지워진 것일 수 있음.

	3가지 resource가 모두 없으면 404 Error
    1가지라도 있어서 삭제하면 삭제처리.
*/
func (nlbHandler *GCPNLBHandler) DeleteNLB(nlbIID irs.IID) (bool, error) {
	deleteResultMap := make(map[string]error) // 삭제가 정상이면 error == nil

	//projectID := nlbHandler.Credential.ProjectID
	regionID := nlbHandler.Region.Region
	targetPoolName := nlbIID.SystemId

	allDeleted := false

	forwardingRuleDeleteResult, err := nlbHandler.deleteRegionForwardingRules(regionID, nlbIID)
	if err != nil {
		cblogger.Info("DeleteNLB forwardingRule ", forwardingRuleDeleteResult, err)
		deleteResultMap[NLB_Component_FORWARDINGRULE] = err
	}
	cblogger.Info("DeleteNLB forwardingRuleDeleteResult ", forwardingRuleDeleteResult)

	// backend

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   nlbHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: targetPoolName,
		CloudOSAPI:   "DeleteNLB()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	err = nlbHandler.removeTargetPool(regionID, targetPoolName)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		cblogger.Info("DeleteNLB removeTargetPool  err: ", err)
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		deleteResultMap[NLB_Component_TARGETPOOL] = err
		//return false, err
	}

	// health checker
	err = nlbHandler.removeHttpHealthCheck(targetPoolName, String_Empty)
	if err != nil {
		cblogger.Info("DeleteNLB removeHttpHealthCheck  err: ", err)
		deleteResultMap[NLB_Component_HEALTHCHECKER] = err
		//return false, err
	}

	// 삭제 결과 return
	returnMsg := String_Empty
	resourceIdx := 1

	resourceCountTotal := 0
	resourceCount404 := 0
	for errKey, errMsg := range deleteResultMap {
		if errMsg != nil {
			isValidCode, isValidErrorFormat := checkErrorCode(ErrorCode_NotFound, errMsg)
			cblogger.Info("DeleteNLB checkErrorCode: ", errKey, errMsg, isValidCode, isValidErrorFormat)
			if !isValidCode && isValidErrorFormat {
				returnMsg += "(" + strconv.Itoa(resourceIdx) + ") " + errKey + " " + errMsg.Error()
				resourceIdx++
			} else {
				// 404 면 이미 지워진 것일 수 있음
				resourceCount404++
			}
			resourceCountTotal++
		}
	}

	if resourceCountTotal > 0 && resourceCountTotal == resourceCount404 {
		return allDeleted, errors.New("The resource NLB " + targetPoolName + " was not found")
	}
	if resourceIdx == 1 { // error 없으면
		allDeleted = true
		return allDeleted, nil
	} else {
		return allDeleted, errors.New(returnMsg)
	}

	return allDeleted, nil
}

//------ Frontend Control

/*
	Listener 정보 변경 -> 수정기능이 없으므로 Error return

		수정 가능한 항목은 Protocol, IP, Port, DNSName(현재 버전에서는 사용x. 향후 사용가능)
		: patch function이 있으나 현재는 NetworkTier만 수정가능하여 해당 function사용 못함

		부하 분산기를 전환하려면 다음 단계를 따르세요.

		프리미엄 등급 IP 주소를 사용하는 새로운 부하 분산기 전달 규칙을 만듭니다.
		현재 표준 등급 IP 주소에서 새로운 프리미엄 등급 IP 주소로 트래픽을 천천히 마이그레이션하려면 DNS를 사용합니다.
		마이그레이션이 완료되면 표준 등급 IP 주소 및 이와 연결된 리전 부하 분산기를 해제할 수 있습니다.
		여러 부하 분산기가 동일한 백엔드를 가리키도록 할 수 있으므로 백엔드를 변경할 필요는 없습니다.

		(참고) patch 사용하려던 로직
		if !strings.EqualFold(listener.Protocol, "") {
			patchRegionForwardingRule.IPProtocol = listener.Protocol
		}

		if !strings.EqualFold(listener.IP, "") {
			patchRegionForwardingRule.IPAddress = listener.IP
		}

		if !strings.EqualFold(listener.Port, "") {
			patchRegionForwardingRule.PortRange = listener.Port
		}

		patchRegionForwardingRule.NetworkTier = "STANDARD"
		//networkTier :
		//	. If this field is not specified, it is assumed to be PREMIUM.
		//	. If IPAddress is specified, this value must be equal to the networkTier of the Address.
		//	- Region forwording rule : PREMIUM and STANDARD
		//	- Global forwording rule : PREMIUM only

		nlbHandler.patchRegionForwardingRules(regionID, forwardingRuleName, &patchRegionForwardingRule)

*/
func (nlbHandler *GCPNLBHandler) ChangeListener(nlbIID irs.IID, listener irs.ListenerInfo) (irs.ListenerInfo, error) {

	return irs.ListenerInfo{}, errors.New("GCP_CANNOT_CHANGE_LISTENER")

	//regionID := nlbHandler.Region.Region
	//targetPoolName := nlbIID.NameId
	////targetPoolUrl := nlbIID.SystemId
	//
	//// targetPool url이 forwarding rule에 필요하여 targetPool 조회
	//targetPool, err := nlbHandler.getTargetPool(regionID, targetPoolName)
	//if err != nil {
	//	cblogger.Info("cannot find Backend Service : ", targetPoolName)
	//	return irs.ListenerInfo{}, errors.New("cannot find Backend Service")
	//}
	//
	//// 기존 forwardingRule 삭제
	//err = nlbHandler.deleteRegionForwardingRule(regionID, targetPoolName)
	//if err != nil {
	//	return irs.ListenerInfo{}, err
	//}
	//
	//// 새로운 forwarding Rule 등록
	//regRegionForwardingRule := convertNlbInfoToForwardingRule(listener, targetPool)
	//
	//callogger := call.GetLogger("HISCALL")
	//callLogInfo := call.CLOUDLOGSCHEMA{
	//	CloudOS:      call.GCP,
	//	RegionZone:   nlbHandler.Region.Zone,
	//	ResourceType: call.NLB,
	//	ResourceName: targetPoolName,
	//	CloudOSAPI:   "ChangeListener()",
	//	ElapsedTime:  "",
	//	ErrorMSG:     "",
	//}
	//callLogStart := call.Start()
	//err = nlbHandler.insertRegionForwardingRules(regionID, &regRegionForwardingRule)
	//callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	//if err != nil {
	//	// 등록 실패
	//	callLogInfo.ErrorMSG = err.Error()
	//	callogger.Info(call.String(callLogInfo))
	//	cblogger.Error(err)
	//	return irs.ListenerInfo{}, err
	//}
	//
	//forwardingRule, err := nlbHandler.getRegionForwardingRules(regionID, targetPoolName)
	//
	//// set ListenerInfo
	//listenerInfo := irs.ListenerInfo{
	//	Protocol: forwardingRule.IPProtocol,
	//	IP:       forwardingRule.IPAddress,
	//	Port:     forwardingRule.PortRange,
	//	//DNSName:  forwardingRule., // 향후 사용할 때 Adderess에서 가져올 듯
	//	CspID: forwardingRule.Name, // forwarding rule name 전체
	//	//KeyValueList:
	//}
	//
	//return listenerInfo, nil
}

/*
	VM Group 변경에서는 VMs 는 제외임.
	GCP의 경우 frontend와 backend를 protocol, ip로 연결하지 않으므로 해당 기능은 제외한다.
*/
func (nlbHandler *GCPNLBHandler) ChangeVMGroupInfo(nlbIID irs.IID, vmGroup irs.VMGroupInfo) (irs.VMGroupInfo, error) {

	return irs.VMGroupInfo{}, errors.New("GCP_CANNOT_CHANGE_VMGroupINFO")
}

/*
	targetPool에 vm 추가
    필요한 parameter는 instanceUrl이며 vmIID.SystemID에서 vm을 조회하여 사용해야 함.
	수정 후 해당 vmGroupInfo(instance 들) return
*/
func (nlbHandler *GCPNLBHandler) AddVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (irs.VMGroupInfo, error) {
	//
	regionID := nlbHandler.Region.Region
	targetPoolName := nlbIID.NameId

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   nlbHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: targetPoolName,
		CloudOSAPI:   "AddVMs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	err := nlbHandler.addTargetPoolInstance(regionID, targetPoolName, vmIIDs)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.VMGroupInfo{}, err
	}

	targetPool, err := nlbHandler.getTargetPool(regionID, targetPoolName)
	if err != nil {
		cblogger.Info("targetPoolList  list: ", err)
		return irs.VMGroupInfo{}, err
	}

	nlbInfo := irs.NLBInfo{}

	// protocol, port는 listener 에 있는 값으로 set
	listenerInfo, err := nlbHandler.getListenerByNlbSystemID(nlbIID)
	nlbInfo.VMGroup.Protocol = listenerInfo.Protocol
	nlbInfo.VMGroup.Port = listenerInfo.Port
	err = nlbHandler.convertTargetPoolToNLBInfo(targetPool, &nlbInfo)
	if err != nil {
		return irs.VMGroupInfo{}, err
	}
	//vmGroup := extractVmGroup(targetPool)
	//printToJson(vmGroup)
	//return vmGroup, nil
	return nlbInfo.VMGroup, nil
}

/*
	targetPool에 vm 삭제
    필요한 parameter는 instanceUrl이며 vmIID.SystemID에 들어있음.
	수정 성공여부 return
*/
func (nlbHandler *GCPNLBHandler) RemoveVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {
	//
	regionID := nlbHandler.Region.Region
	targetPoolName := nlbIID.NameId

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   nlbHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: targetPoolName,
		CloudOSAPI:   "RemoveVMs()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	err := nlbHandler.removeTargetPoolInstances(regionID, targetPoolName, vmIIDs)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)

		return false, err
	}
	return true, nil
}

// get HealthCheckerInfo
// VMGroup의 healthcheckResult
func (nlbHandler *GCPNLBHandler) GetVMGroupHealthInfo(nlbIID irs.IID) (irs.HealthInfo, error) {
	var returnHealthInfo irs.HealthInfo
	// vmgroup의 instance 목록 조회
	regionID := nlbHandler.Region.Region
	targetPoolName := nlbIID.NameId

	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   nlbHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: targetPoolName,
		CloudOSAPI:   "GetVMGroupHealthInfo()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	targetPool, err := nlbHandler.getTargetPool(regionID, targetPoolName)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		cblogger.Info("targetPoolList  list: ", err)
	}

	//vmGroup := extractVmGroup(targetPool)

	allVmIIDs := []irs.IID{}
	healthyVmIIDs := []irs.IID{}
	unHealthyVmIIDs := []irs.IID{}

	for _, instanceUrl := range targetPool.Instances {
		//for _, instance := range *vmGroup.VMs {
		//instanceUrl := instance.SystemId

		instanceHealthStatusList, err := nlbHandler.getTargetPoolHealth(regionID, targetPoolName, instanceUrl)
		if err != nil {
			cblogger.Info("targetPool HealthList  list: ", err)
			return irs.HealthInfo{}, err
		}

		healthStatusInfo := instanceHealthStatusList.HealthStatus

		targetPoolInstanceArr := strings.Split(instanceUrl, StringSeperator_Slash)
		instanceID := targetPoolInstanceArr[len(targetPoolInstanceArr)-1]
		instanceIID := irs.IID{SystemId: instanceID}
		allVmIIDs = append(allVmIIDs, instanceIID)

		// healthStatus 가 배열형태이고 0번째만 취함.
		if strings.EqualFold(healthStatusInfo[0].HealthState, HealthState_UNHEALTHY) {
			unHealthyVmIIDs = append(unHealthyVmIIDs, instanceIID)
		}

		if strings.EqualFold(healthStatusInfo[0].HealthState, HealthState_HEALTHY) {
			healthyVmIIDs = append(healthyVmIIDs, instanceIID)
		}
	}

	returnHealthInfo.UnHealthyVMs = &unHealthyVmIIDs
	returnHealthInfo.HealthyVMs = &healthyVmIIDs
	returnHealthInfo.AllVMs = &allVmIIDs
	printToJson(returnHealthInfo)
	return returnHealthInfo, nil
}

/*
// HealthCheckerInfo 변경
	cspId = selfLink
	healthCheckerName = nbl name

	다른 health checker로 변경은 기존 health checker 삭제 후 추가 됨.
*/
func (nlbHandler *GCPNLBHandler) ChangeHealthCheckerInfo(nlbIID irs.IID, healthChecker irs.HealthCheckerInfo) (irs.HealthCheckerInfo, error) {
	regionID := nlbHandler.Region.Region
	targetPoolName := nlbIID.NameId

	// health checker 수정
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   nlbHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: targetPoolName,
		CloudOSAPI:   "ChangeHealthCheckerInfo()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	err := nlbHandler.patchHealthCheck(regionID, targetPoolName, &healthChecker)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.HealthCheckerInfo{}, err
	}
	cblogger.Info("patchTargetPoolHealthCheck -----")
	printToJson(healthChecker)

	targetPool, err := nlbHandler.getTargetPool(regionID, targetPoolName)
	if err != nil {
		cblogger.Info("targetPoolList  list: ", err)
	}
	returnHealthChecker, err := nlbHandler.extractHealthChecker(regionID, targetPool)

	if err != nil {
		cblogger.Info("get targetpool err ", err)
	}
	return returnHealthChecker, nil
}

////// private area ////////////
// region and global methods
// - GCP API-               - SPIDER API -
// backendService        -> globalBackendService
// regionBackendService  -> regionBackendService
// globalForwardingRule  -> globalForwardingRule
// forwardingRule        -> regionForwardingRule
// healthCheck           -> globalHealthCheck
// regionHealthCheck     -> regionHealthCheck
// targetPools           -> ... : AddVms, RemoveVms

// instance template 등록 : 미사용
//func (nlbHandler *GCPNLBHandler) insertInstanceTemplate(instanceTemplateReq compute.InstanceTemplate) error {
//	//POST https://compute.googleapis.com/compute/v1/projects/PROJECT_ID/global/instanceTemplates
//	//{
//	//	"name": "INSTANCE_TEMPLATE_NAME",
//	//	"sourceInstance": "zones/SOURCE_INSTANCE_ZONE/instances/SOURCE_INSTANCE",
//	//	"sourceInstanceParams": {
//	//		"diskConfigs": [
//	//			{
//	//			"deviceName": "SOURCE_DISK",
//	//			"instantiateFrom": "INSTANTIATE_OPTIONS",
//	//			"autoDelete": false
//	//			}
//	//		]
//	//	}
//	//}
//
//	// path param
//	projectID := nlbHandler.Credential.ProjectID
//
//	//instanceTemplate := compute.InstanceTemplate{}
//	req, err := nlbHandler.Client.InstanceTemplates.Insert(projectID, &instanceTemplateReq).Do()
//	if err != nil {
//
//	}
//
//	err = WaitUntilComplete(nlbHandler.Client, projectID, "", req.Name, true)
//	if err != nil {
//
//	}
//
//	id := req.Id
//	name := req.Name
//	cblogger.Info("id = ", id, " : name = ", name)
//	instanceTemplate, err := nlbHandler.getInstanceTemplate(name)
//	if err != nil {
//		//return nil, err
//	}
//
//	cblogger.Info("instanceTemplate ", instanceTemplate)
//	return nil
//	//if err != nil {
//	//	return irs.VPCInfo{}, err
//	//}
//	//errWait := vVPCHandler.WaitUntilComplete(req.Name, true)
//
//	//compute.NewInstanceTemplatesService()
//	//result, err := nlbHandler.Client.
//	//fireWall := compute.Firewall{
//	//	Name:      firewallName,
//	//	Allowed:   firewallAllowed,
//	//	Denied:    firewallDenied,
//	//	Direction: firewallDirection,
//	//	Network:   networkURL,
//	//	TargetTags: []string{
//	//		securityGroupName,
//	//	},
//	//}
//	//type InstanceTemplatesInsertCall struct {
//	//	s                *Service
//	//	project          string
//	//	instancetemplate *InstanceTemplate
//	//	urlParams_       gensupport.URLParams
//	//	ctx_             context.Context
//	//	header_          http.Header
//	//}
//}

// instanceTemplate 조회 : 미사용
//func (nlbHandler *GCPNLBHandler) getInstanceTemplate(resourceId string) (*compute.InstanceTemplate, error) {
//	projectID := nlbHandler.Credential.ProjectID
//
//	instanceTemplateInfo, err := nlbHandler.Client.InstanceTemplates.Get(projectID, resourceId).Do()
//	if err != nil {
//		return &compute.InstanceTemplate{}, err
//	}
//
//	//
//	cblogger.Info(instanceTemplateInfo)
//	return instanceTemplateInfo, nil
//}

// instanceTemplate 목록 조회 : 미사용
// InstanceTemplateList 객체를 넘기고 사용은 InstanceTemplateList.Item에서 꺼내서 사용
//func (nlbHandler *GCPNLBHandler) listInstanceTemplate(filter string) (*compute.InstanceTemplateList, error) {
//	projectID := nlbHandler.Credential.ProjectID
//
//	fmt.Printf(filter)
//	//if strings.EqualFold(filter, "") {
//	//	req := nlbHandler.Client.InstanceTemplates.List(projectID)
//	//	//req.Filter()
//	//	if err := req.Pages(nlbHandler.Ctx, func(page *compute.InstanceTemplateList) error {
//	//		for _, instanceTemplate := range page.Items {
//	//			fmt.Printf("%#v\n", instanceTemplate)
//	//		}
//	//		return nil
//	//	}); err != nil {
//	//		//log.Fatal(err)
//	//	}
//	//}
//	result, err := nlbHandler.Client.InstanceTemplates.List(projectID).Do()
//	if err != nil {
//		return nil, err
//	}
//
//	//
//
//	cblogger.Info(result)
//	cblogger.Info(" len ", len(result.Items))
//	return result, nil
//}

// Region instance group 등록 : 미사용
//func (nlbHandler *GCPNLBHandler) insertRegionInstanceGroup(regionID string, reqInstanceGroupManager compute.InstanceGroupManager) (*compute.InstanceGroupManager, error) {
//	// path param
//	projectID := nlbHandler.Credential.ProjectID
//
//	req, err := nlbHandler.Client.RegionInstanceGroupManagers.Insert(projectID, regionID, &reqInstanceGroupManager).Do()
//	if err != nil {
//
//	}
//
//	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.Name, true)
//	if err != nil {
//
//	}
//
//	id := req.Id
//	name := req.Name
//	cblogger.Info("id = ", id, " : name = ", name)
//	result, err := nlbHandler.getRegionInstanceGroupManager(regionID, name)
//	if err != nil {
//		//return nil, err
//	}
//
//	cblogger.Info("RegionInstanceGroup ", result)
//	return result, nil
//}

// Region InstanceGroup 조회 : 미사용
//func (nlbHandler *GCPNLBHandler) getRegionInstanceGroupManager(regionID string, resourceId string) (*compute.InstanceGroupManager, error) {
//	projectID := nlbHandler.Credential.ProjectID
//
//	result, err := nlbHandler.Client.RegionInstanceGroupManagers.Get(projectID, regionID, resourceId).Do()
//	if err != nil {
//		return &compute.InstanceGroupManager{}, err
//	}
//
//	//
//	cblogger.Info(result)
//	return result, nil
//}

// Region InstanceGroup 목록 조회 : 미사용
// InstanceGroupList 객체를 넘기고 사용은 InstanceGroupList.Item에서 꺼내서 사용
// return 객체가 RegionInstanceGroupManagerList 임. 다른것들은 Region 구분 없는 객체로 return
//func (nlbHandler *GCPNLBHandler) listRegionInstanceGroupManager(regionID string, filter string) (*compute.RegionInstanceGroupManagerList, error) {
//	projectID := nlbHandler.Credential.ProjectID
//
//	fmt.Printf(filter)
//	result, err := nlbHandler.Client.RegionInstanceGroupManagers.List(projectID, regionID).Do()
//	if err != nil {
//		return nil, err
//	}
//
//	//
//
//	cblogger.Info(result)
//	cblogger.Info(" len ", len(result.Items))
//	return result, nil
//}

// regionInstanceGroups는 이기종 또는 직접관리하는 경우 사용 but, get, list, listInstances, setNamedPoers  만 있음. insert없음
// InstanceGroup 조회 : 미사용
//func (nlbHandler *GCPNLBHandler) getRegionInstanceGroup(regionID string, resourceId string) (*compute.InstanceGroup, error) {
//	projectID := nlbHandler.Credential.ProjectID
//
//	result, err := nlbHandler.Client.RegionInstanceGroups.Get(projectID, regionID, resourceId).Do()
//	if err != nil {
//		return &compute.InstanceGroup{}, err
//	}
//
//	//
//	cblogger.Info(result)
//	return result, nil
//}

// InstanceGroup 목록 조회 : 미사용
// regionInstanceGroups는 이기종 또는 직접관리하는 경우 사용 but, get, list, listInstances, setNamedPoers  만 있음. insert없음
// RegionInstanceGroupList 객체를 넘기고 사용은 RegionInstanceGroupList.Item에서 꺼내서 사용
//func (nlbHandler *GCPNLBHandler) listRegionInstanceGroups(regionID string, filter string) (*compute.RegionInstanceGroupList, error) {
//	projectID := nlbHandler.Credential.ProjectID
//
//	fmt.Printf(filter)
//	result, err := nlbHandler.Client.RegionInstanceGroups.List(projectID, regionID).Do()
//	if err != nil {
//		return nil, err
//	}
//
//	//
//
//	cblogger.Info(result)
//	cblogger.Info(" len ", len(result.Items))
//	return result, nil
//}

// 호출하는 api가 listInstances 여서 listInstances + RegionInstanceGroups : 미사용
// RegionInstanceGroupsListInstances 객체를 넘기고 사용은 RegionInstanceGroupsListInstances.Item에서 꺼내서 사용
//func (nlbHandler *GCPNLBHandler) listInstancesRegionInstanceGroups(regionID string, regionInstanceGroupName string, reqListInstance compute.RegionInstanceGroupsListInstancesRequest) (*compute.RegionInstanceGroupsListInstances, error) {
//	projectID := nlbHandler.Credential.ProjectID
//
//	fmt.Printf(regionInstanceGroupName)
//	result, err := nlbHandler.Client.RegionInstanceGroups.ListInstances(projectID, regionID, regionInstanceGroupName, &reqListInstance).Do()
//	if err != nil {
//		return nil, err
//	}
//
//	//
//
//	cblogger.Info(result)
//	cblogger.Info(" len ", len(result.Items))
//	return result, nil
//}

// global instance group 등록 : 미사용
//func (nlbHandler *GCPNLBHandler) insertGlobalInstanceGroup(zoneID string, reqInstanceGroup compute.InstanceGroup) error {
//	// path param
//	projectID := nlbHandler.Credential.ProjectID
//
//	//instanceTemplate := compute.InstanceTemplate{}
//	req, err := nlbHandler.Client.InstanceGroups.Insert(projectID, zoneID, &reqInstanceGroup).Do()
//	if err != nil {
//
//	}
//
//	err = WaitUntilComplete(nlbHandler.Client, projectID, "", req.Name, true)
//	if err != nil {
//
//	}
//
//	id := req.Id
//	name := req.Name
//	cblogger.Info("id = ", id, " : name = ", name)
//	instanceTemplate, err := nlbHandler.getInstanceTemplate(name)
//	if err != nil {
//		//return nil, err
//	}
//
//	cblogger.Info("instanceTemplate ", instanceTemplate)
//	return nil
//}

// global InstanceGroup 조회 : 미사용
//func (nlbHandler *GCPNLBHandler) getGlobalInstanceGroup(zoneID string, instanceGroupName string) (*compute.InstanceGroup, error) {
//	projectID := nlbHandler.Credential.ProjectID
//
//	instanceGroupInfo, err := nlbHandler.Client.InstanceGroups.Get(projectID, zoneID, instanceGroupName).Do()
//	if err != nil {
//		return &compute.InstanceGroup{}, err
//	}
//
//	//
//	cblogger.Info(instanceGroupInfo)
//	return instanceGroupInfo, nil
//}

// global InstanceGroup 목록 조회 : 미사용
// In용tanceGroupList 객체를 넘기고 사용은 InstanceGroupList.Item에서 꺼내서 사용
//func (nlbHandler *GCPNLBHandler) listGlobalInstanceGroup(zoneID string, filter string) (*compute.InstanceGroupList, error) {
//	projectID := nlbHandler.Credential.ProjectID
//
//	fmt.Printf(filter)
//	result, err := nlbHandler.Client.InstanceGroups.List(projectID, zoneID).Do()
//	if err != nil {
//		return nil, err
//	}
//
//	cblogger.Info(result)
//	cblogger.Info(" len ", len(result.Items))
//	return result, nil
//}

// Address 등록
func (nlbHandler *GCPNLBHandler) insertRegionAddresses(regionID string, reqAddress compute.Address) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	//instanceTemplate := compute.InstanceTemplate{}
	req, err := nlbHandler.Client.Addresses.Insert(projectID, regionID, &reqAddress).Do()
	if err != nil {

	}

	err = WaitUntilComplete(nlbHandler.Client, projectID, String_Empty, req.Name, true)
	if err != nil {
		return err
	}

	// TODO : 조회로직을 넣어야하나?
	//id := req.Id
	//name := req.Name
	//cblogger.Info("id = ", id, " : name = ", name)
	//addressInfo, err := nlbHandler.getAddresses(regionID, name)
	//if err != nil {
	//	return err
	//}
	//cblogger.Info("addressInfo ", addressInfo)
	return nil
}

// Address 삭제
func (nlbHandler *GCPNLBHandler) removeRegionAddresses(regionID string, addressName string) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	req, err := nlbHandler.Client.Addresses.Delete(projectID, regionID, addressName).Do()
	if err != nil {

	}

	err = WaitUntilComplete(nlbHandler.Client, projectID, String_Empty, req.Name, true)
	if err != nil {
		return err
	}

	return nil
}

// Address 수집목록
func (nlbHandler *GCPNLBHandler) aggregatedListRegionAddresses(filter string) (*compute.AddressAggregatedList, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.Addresses.AggregatedList(projectID).Do()
	if err != nil {
		return nil, err
	}
	for _, item := range resp.Items {
		cblogger.Info(item)
	}
	return resp, nil
}

// Address 조회
func (nlbHandler *GCPNLBHandler) getRegionAddresses(regionID string, addressName string) (*compute.Address, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID
	//region := nlbHandler.Region.Region

	addressInfo, err := nlbHandler.Client.Addresses.Get(projectID, regionID, addressName).Do()
	if err != nil {
		return nil, err
	}
	return addressInfo, nil
}

// Address 목록조회
func (nlbHandler *GCPNLBHandler) listRegionAddresses(regionID string, filter string) (*compute.AddressList, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.Addresses.List(projectID, regionID).Do()
	if err != nil {
		return nil, err
	}
	for _, item := range resp.Items {
		cblogger.Info(item)
	}
	return resp, nil
}

// Address 등록 : LB의 시작점
func (nlbHandler *GCPNLBHandler) insertGlobalAddresses(reqAddress compute.Address) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	//instanceTemplate := compute.InstanceTemplate{}
	req, err := nlbHandler.Client.GlobalAddresses.Insert(projectID, &reqAddress).Do()
	if err != nil {

	}

	err = WaitUntilComplete(nlbHandler.Client, projectID, String_Empty, req.Name, true)
	if err != nil {
		return err
	}

	// TODO : 조회로직을 넣어야하나?
	//id := req.Id
	//name := req.Name
	//cblogger.Info("id = ", id, " : name = ", name)
	//addressInfo, err := nlbHandler.getAddresses(regionID, name)
	//if err != nil {
	//	return err
	//}
	//cblogger.Info("addressInfo ", addressInfo)
	return nil
}

// Address 삭제
func (nlbHandler *GCPNLBHandler) removeGlobalAddresses(addressName string) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	req, err := nlbHandler.Client.GlobalAddresses.Delete(projectID, addressName).Do()
	if err != nil {
		return err
	}

	err = WaitUntilComplete(nlbHandler.Client, projectID, String_Empty, req.Name, true)
	if err != nil {
		return err
	}

	return nil
}

// Address 조회
func (nlbHandler *GCPNLBHandler) getGlobalAddresses(addressName string) (*compute.Address, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID
	//region := nlbHandler.Region.Region

	addressInfo, err := nlbHandler.Client.GlobalAddresses.Get(projectID, addressName).Do()
	if err != nil {
		return nil, err
	}
	return addressInfo, nil
}

// Address 목록조회
func (nlbHandler *GCPNLBHandler) listGlobalAddresses(filter string) (*compute.AddressList, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.GlobalAddresses.List(projectID).Do()
	if err != nil {
		return nil, err
	}
	for _, item := range resp.Items {
		cblogger.Info(item)
	}
	return resp, nil
}

// Region ForwardingRule 등록
func (nlbHandler *GCPNLBHandler) insertRegionForwardingRules(regionID string, reqRegionForwardingRule *compute.ForwardingRule) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID
	//region := nlbHandler.Region.Region

	//reqForwardingRule := &compute.ForwardingRule{}
	req, err := nlbHandler.Client.ForwardingRules.Insert(projectID, regionID, reqRegionForwardingRule).Do()
	if err != nil {
		//return &compute.ForwardingRule{}, err
		return err
	}

	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.Name, false)
	if err != nil {
		//	return &compute.ForwardingRule{}, err
		return err
	}

	return nil
}

//deleteRegionForwardingRule
// 특정 forwarding rule만 삭제
func (nlbHandler *GCPNLBHandler) deleteRegionForwardingRule(regionID string, forwardingRuleName string) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	req, err := nlbHandler.Client.ForwardingRules.Delete(projectID, regionID, forwardingRuleName).Do()
	if err != nil {
		cblogger.Info("deleteRegionForwardingRule ", err)
		return err
	}
	cblogger.Info("req ", req)
	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.Name, false)
	if err != nil {
		cblogger.Info("WaitUntilComplete ", err)
		return err
	}
	return nil
}

// nlb에 종속 된 forwarding rule 모두 삭제 : nlb 삭제시 사용
// return : 삭제 총 갯수 / 전체 갯수, error
func (nlbHandler *GCPNLBHandler) deleteRegionForwardingRules(regionID string, nlbIID irs.IID) (string, error) {
	// path param
	//targetPoolUrl := nlbIID.SystemId
	targetPoolId := nlbIID.SystemId

	forwardingRuleList, err := nlbHandler.listRegionForwardingRules(regionID, String_Empty, targetPoolId)
	if err != nil {
		cblogger.Info("DeleteNLB forwardingRule  err: ", err)
		return String_Empty, err
	}
	deleteCount := 0
	itemLength := len(forwardingRuleList.Items)
	if itemLength == 0 {
		return String_Empty, errors.New("Error 404: The Forwarding Rule resource of " + targetPoolId + " was not found")
	}
	for _, forwardingRule := range forwardingRuleList.Items {
		err := nlbHandler.deleteRegionForwardingRule(regionID, forwardingRule.Name)
		if err != nil {
			cblogger.Info("deleteRegionForwardingRule ", err)
			return String_Empty, err
		}
	}
	return strconv.Itoa(deleteCount) + StringSeperator_Slash + strconv.Itoa(itemLength), nil
}

// Region ForwardingRule patch
func (nlbHandler *GCPNLBHandler) patchRegionForwardingRules(regionID string, forwardingRuleName string, patchRegionForwardingRule *compute.ForwardingRule) (*compute.ForwardingRule, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID
	//region := nlbHandler.Region.Region

	req, err := nlbHandler.Client.ForwardingRules.Patch(projectID, regionID, forwardingRuleName, patchRegionForwardingRule).Do()
	if err != nil {
		cblogger.Info("patchRegionForwardingRules ", err)
		return &compute.ForwardingRule{}, err
	}

	printToJson(req)
	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.Name, false)
	if err != nil {
		return &compute.ForwardingRule{}, err
	}

	id := req.Id
	name := req.Name
	cblogger.Info("id = ", id, " : name = ", name)
	forwardingRule, err := nlbHandler.getRegionForwardingRules(regionID, patchRegionForwardingRule.Name)
	if err != nil {
		return &compute.ForwardingRule{}, err
		//return nil, err
	}

	cblogger.Info("ForwardingRule ", forwardingRule)
	return forwardingRule, nil
}

// Region ForwardingRule 조회
func (nlbHandler *GCPNLBHandler) getRegionForwardingRules(regionID string, regionForwardingRuleName string) (*compute.ForwardingRule, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID
	//region := nlbHandler.Region.Region
	regionForwardingRule, err := nlbHandler.Client.ForwardingRules.Get(projectID, regionID, regionForwardingRuleName).Do()
	if err != nil {
		return nil, err
	}
	return regionForwardingRule, nil
}

// Region ForwardingRule 목록 조회
// FordingRuleList 객체를 넘기고 사용은 fordingRuleList.Item에서 꺼내서 사용
// 특정 targetPoolName을 넘겨주면 해당 targetPool내 forwardingRule목록을 넘김
func (nlbHandler *GCPNLBHandler) listRegionForwardingRules(regionID string, filter string, forwardingRuleName string) (*compute.ForwardingRuleList, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.ForwardingRules.List(projectID, regionID).Do()
	if err != nil {
		cblogger.Info(err)
		return nil, err
	}

	if !strings.EqualFold(forwardingRuleName, String_Empty) {
		cblogger.Info("listRegionForwardingRules")

		responseForwardingRule := compute.ForwardingRuleList{}
		forwardingRuleList := []*compute.ForwardingRule{}
		for _, item := range resp.Items {
			forwardingRuleUrlArr := strings.Split(item.SelfLink, StringSeperator_Slash)

			itemForwardingRule := forwardingRuleUrlArr[len(forwardingRuleUrlArr)-1]

			if strings.EqualFold(itemForwardingRule, forwardingRuleName) {
				forwardingRuleList = append(forwardingRuleList, item)
				cblogger.Info(item)
			}
		}
		responseForwardingRule.Items = forwardingRuleList
		return &responseForwardingRule, nil
	}
	return resp, nil

}

// Global ForwardingRule 등록
func (nlbHandler *GCPNLBHandler) insertGlobalForwardingRules(regionID string, reqGlobalForwardingRule *compute.ForwardingRule) (*compute.ForwardingRule, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID
	//region := nlbHandler.Region.Region

	//reqForwardingRule := &compute.ForwardingRule{}
	req, err := nlbHandler.Client.GlobalForwardingRules.Insert(projectID, reqGlobalForwardingRule).Do()
	if err != nil {
		return &compute.ForwardingRule{}, err
	}

	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.Name, true)
	if err != nil {
		return &compute.ForwardingRule{}, err
	}

	id := req.Id
	name := req.Name
	cblogger.Info("id = ", id, " : name = ", name)
	globalForwardingRule, err := nlbHandler.getGlobalForwardingRules(reqGlobalForwardingRule.Name)
	if err != nil {
		return &compute.ForwardingRule{}, err
		//return nil, err
	}

	cblogger.Info("backendService ", globalForwardingRule)
	return globalForwardingRule, nil
}

// Global ForwardingRule 조회
func (nlbHandler *GCPNLBHandler) getGlobalForwardingRules(forwardingRuleName string) (*compute.ForwardingRule, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID
	//region := nlbHandler.Region.Region

	forwardingRule, err := nlbHandler.Client.GlobalForwardingRules.Get(projectID, forwardingRuleName).Do()
	if err != nil {
		return nil, err
	}
	return forwardingRule, nil
}

// Global ForwardingRule 목록 조회
// FordingRuleList 객체를 넘기고 사용은 fordingRuleList.Item에서 꺼내서 사용
func (nlbHandler *GCPNLBHandler) listGlobalForwardingRules(filter string) (*compute.ForwardingRuleList, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.GlobalForwardingRules.List(projectID).Do()
	if err != nil {
		return nil, err
	}
	for _, item := range resp.Items {
		cblogger.Info(item)
	}

	return resp, nil
}

// Region BackendService 등록
func (nlbHandler *GCPNLBHandler) insertRegionBackendServices(regionID string, reqRegionBackendService compute.BackendService) (*compute.BackendService, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	//instanceTemplate := compute.InstanceTemplate{}
	req, err := nlbHandler.Client.RegionBackendServices.Insert(projectID, regionID, &reqRegionBackendService).Do()
	if err != nil {

	}

	err = WaitUntilComplete(nlbHandler.Client, projectID, String_Empty, req.Name, true)
	if err != nil {
		return &compute.BackendService{}, err
	}

	id := req.Id
	name := req.Name
	cblogger.Info("id = ", id, " : name = ", name)
	backendService, err := nlbHandler.getRegionBackendServices(regionID, reqRegionBackendService.Name)
	if err != nil {
		return &compute.BackendService{}, err
		//return nil, err
	}

	cblogger.Info("backendService ", backendService)
	return backendService, nil
}

// Region BackendService 조회
func (nlbHandler *GCPNLBHandler) getRegionBackendServices(region string, regionBackendServiceName string) (*compute.BackendService, error) {
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.RegionBackendServices.Get(projectID, region, regionBackendServiceName).Do()
	if err != nil {
		return nil, err
	}
	//backend service name : lb-seoul-backendservice

	//
	for _, item := range resp.Backends {
		cblogger.Info(item)
		//if strings.EqualFold(item.Group,   // instance group or network endpoint group(NEG)
	}
	//// item.group // instance group or network endpoint group
	//cblogger.Info(backServices)
	//return backServices, nil
	return resp, nil
}

// Region BackendService 목록 조회
// FordingRuleList 객체를 넘기고 사용은 fordingRuleList.Item에서 꺼내서 사용
func (nlbHandler *GCPNLBHandler) listRegionBackendServices(region string, filter string) (*compute.BackendServiceList, error) {
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.RegionBackendServices.List(projectID, region).Do()
	if err != nil {
		return nil, err
	}

	//cblogger.Info(resp)
	printToJson(resp)
	for _, item := range resp.Items {
		cblogger.Info(item)
		//if strings.EqualFold(item.Group,   // instance group or network endpoint group(NEG)
	}
	//// item.group // instance group or network endpoint group
	//cblogger.Info(backServices)
	//return backServices, nil
	return resp, nil
}

// Global BackendService 등록
func (nlbHandler *GCPNLBHandler) insertGlobalBackendServices(reqBackendService compute.BackendService) (*compute.BackendService, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	//instanceTemplate := compute.InstanceTemplate{}
	req, err := nlbHandler.Client.BackendServices.Insert(projectID, &reqBackendService).Do()
	if err != nil {

	}

	err = WaitUntilComplete(nlbHandler.Client, projectID, String_Empty, req.Name, true)
	if err != nil {
		return &compute.BackendService{}, err
	}

	id := req.Id
	name := req.Name
	cblogger.Info("id = ", id, " : name = ", name)
	backendService, err := nlbHandler.getGlobalBackendServices(reqBackendService.Name)
	if err != nil {
		return &compute.BackendService{}, err
		//return nil, err
	}

	cblogger.Info("backendService ", backendService)
	return backendService, nil
}

// Global BackendService 조회
//func (nlbHandler *GCPNLBHandler) getBackendServices(resourceId string) (*compute.InstanceTemplate, error) {
func (nlbHandler *GCPNLBHandler) getGlobalBackendServices(backendServiceName string) (*compute.BackendService, error) {
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.BackendServices.Get(projectID, backendServiceName).Do()
	if err != nil {
		return nil, err
	}
	//backend service name : lb-seoul-backendservice

	//
	backServices := resp.Backends
	for _, item := range backServices {
		cblogger.Info(item)
		//if strings.EqualFold(item.Group,   // instance group or network endpoint group(NEG)
	}

	return resp, nil
}

// Global BackendService 목록 조회
// BackendServiceList 객체를 넘기고 사용은 BackendServiceList.Item에서 꺼내서 사용
func (nlbHandler *GCPNLBHandler) listGlobalBackendServices(filter string) (*compute.BackendServiceList, error) {
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.BackendServices.List(projectID).Do()
	if err != nil {
		return nil, err
	}

	for _, item := range resp.Items {
		cblogger.Info(item)
		//if strings.EqualFold(item.Group,   // instance group or network endpoint group(NEG)
	}
	//// item.group // instance group or network endpoint group
	//cblogger.Info(backServices)
	//return backServices, nil
	return resp, nil
}

// RegionalHealthCheck 등록 : 미사용
//func (nlbHandler *GCPNLBHandler) insertRegionHealthChecks(region string, healthCheckerInfo irs.HealthCheckerInfo) {
//	// path param
//	projectID := nlbHandler.Credential.ProjectID
//
//	// queryParam
//	// 4개 중 1개 : TCP, SSL, HTTP, HTTP2
//	//tcpHealthCheck := &compute.TCPHealthCheck{
//	//	Port:              80, // default value:80, 1~65535
//	//	PortName:          "", // InstanceGroup#NamedPort#name
//	//	PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
//	//	Request:           "", // default value is empty
//	//	Response:          "", // default value is empty
//	//	ProxyHeader:       "", // NONE, PROXY_V1
//	//}
//	//sslHealthCheck := &compute.SSLHealthCheck{
//	//	Port:              443, // default value is 443n 1~65535
//	//	PortName:          "",  // InstanceGroup#NamedPort#name
//	//	PortSpecification: "",  //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
//	//	Request:           "",  // default value is empty
//	//	Response:          "",  // default value is empty
//	//	ProxyHeader:       "",  // NONE, PROXY_V1
//	//}
//	//httpHealthCheck := &compute.HTTPHealthCheck{
//	//	Port:              80, // default value:80, 1~65535
//	//	PortName:          "", // InstanceGroup#NamedPort#name
//	//	PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
//	//	Host:              "", // default value is empty. IP
//	//	RequestPath:       "", // default value is "/"
//	//	Response:          "", // default value is empty
//	//	ProxyHeader:       "", // NONE, PROXY_V1
//	//}
//	//
//	//http2HealthCheck := &compute.HTTP2HealthCheck{
//	//	Port:              443, // default value is 443n 1~65535
//	//	PortName:          "",  // InstanceGroup#NamedPort#name
//	//	PortSpecification: "",  //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
//	//	Host:              "",  // default value is empty
//	//	RequestPath:       "",  // default value is "/"
//	//	Response:          "",  // default value is empty
//	//	ProxyHeader:       "",  // default value is NONE,  NONE, PROXY_V1
//	//}
//
//	//healthCheckPort :=	&
//	regionHealthCheck := &compute.HealthCheck{
//
//	}
//
//	// requestBody
//	nlbHandler.Client.RegionHealthChecks.Insert(projectID, region, regionHealthCheck)
//
//}

// RegionHealthCheck 조회 : 미사용
// Inteal Http(S) load balancer : region health check => compute.v1.regionHealthCheck
// Traffic Director : global health check => compute.v1.HealthCheck
//func (nlbHandler *GCPNLBHandler) getRegionHealthChecks(region string, regionHealthCheckName string) (*compute.HealthCheck, error) {
//
//	// path param
//	projectID := nlbHandler.Credential.ProjectID
//
//	resp, err := nlbHandler.Client.RegionHealthChecks.Get(projectID, region, regionHealthCheckName).Do()
//	if err != nil {
//		return nil, err
//	}
//	return resp, nil
//}

// Global BackendService 목록 조회
// HealthCheckList 객체를 넘기고 사용은 HealthCheckList.Item에서 꺼내서 사용
//func (nlbHandler *GCPNLBHandler) listRegionHealthChecks(region string, filter string) (*compute.HealthCheckList, error) {
//
//	// path param
//	projectID := nlbHandler.Credential.ProjectID
//
//	resp, err := nlbHandler.Client.RegionHealthChecks.List(projectID, region).Do()
//	if err != nil {
//		return nil, err
//	}
//
//	for _, item := range resp.Items {
//		cblogger.Info(item)
//	}
//
//	return resp, nil
//
//}

// HttpHealthChecker Insert
// cspId는 url형태임( url끝에 healthChecker Name 있음) . 생성시에는 url이 없으므로 name을 그대로 사용
// httpHealthcheck insert : LB의 healthcheck는 이거 사용
func (nlbHandler *GCPNLBHandler) insertHttpHealthChecks(healthCheckType string, healthCheckerInfo irs.HealthCheckerInfo) (*compute.HttpHealthCheck, error) {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	port, err := strconv.ParseInt(healthCheckerInfo.Port, 10, 64)
	if err != nil {
		return &compute.HttpHealthCheck{}, err
	}
	checkIntervalSec := int64(healthCheckerInfo.Interval)
	if err != nil {
		return &compute.HttpHealthCheck{}, err
	}
	healthyThreshold := int64(healthCheckerInfo.Threshold)
	if err != nil {
		return &compute.HttpHealthCheck{}, err
	}
	timeoutSec := int64(healthCheckerInfo.Timeout)
	if err != nil {
		return &compute.HttpHealthCheck{}, err
	}

	// cspId는 url형태임( url끝에 healthChecker Name 있음) . 생성시에는 url이 없으므로 name을 그대로 사용
	httpHealthCheck := &compute.HttpHealthCheck{
		Name: healthCheckerInfo.CspID,
		//Description              string `json:"description,omitempty"`
		//Host : The value of the host header in the HTTP health check request. If left empty (default value), the public IP on behalf of which this health check is performed will be used.
		Port:             port, // default value:80, 1~65535
		CheckIntervalSec: checkIntervalSec,
		TimeoutSec:       timeoutSec,
		//UnhealthyThreshold : A so-far healthy instance will be marked unhealthy after this many consecutive failures. The default value is 2.
		UnhealthyThreshold: healthyThreshold,
		HealthyThreshold:   healthyThreshold,

		//RequestPath, string `json:"requestPath,omitempty"`
	}
	printToJson(httpHealthCheck)
	req, err := nlbHandler.Client.HttpHealthChecks.Insert(projectID, httpHealthCheck).Do()
	if err != nil {
		cblogger.Error(err)
		return &compute.HttpHealthCheck{}, err
	}

	err = WaitUntilComplete(nlbHandler.Client, projectID, String_Empty, req.Name, true)
	if err != nil {
		//	return &compute.ForwardingRule{}, err
		return &compute.HttpHealthCheck{}, err
	}

	id := req.Id
	name := req.Name
	cblogger.Info("id = ", id, " : name = ", name, " cspId = ", healthCheckerInfo.CspID)
	healthCheck, err := nlbHandler.getHttpHealthChecks(healthCheckerInfo.CspID)
	if err != nil {
		return healthCheck, err
		//return nil, err
	}

	cblogger.Info("healthCheck ", healthCheck)
	return nil, nil
}

// HttpHealthcheck 조회
func (nlbHandler *GCPNLBHandler) getHttpHealthChecks(healthCheckName string) (*compute.HttpHealthCheck, error) {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.HttpHealthChecks.Get(projectID, healthCheckName).Do()

	if err != nil {
		return nil, err
	}
	return resp, nil
}

// HttpHealthcheck 목록 조회
// HttpHealthCheckList 객체를 넘기고 사용은 HealthCheckList.Item에서 꺼내서 사용
func (nlbHandler *GCPNLBHandler) listHttpHealthChecks(filter string) (*compute.HttpHealthCheckList, error) {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.HttpHealthChecks.List(projectID).Do()
	if err != nil {
		return nil, err
	}

	for _, item := range resp.Items {
		cblogger.Info(item)
	}

	return resp, nil
}

// TargetPool Insert
// Target pools are used for network TCP/UDP load balancing. A target pool references member instances, an associated legacy HttpHealthCheck resource, and, optionally, a backup target poo
// 넘어온 값으로 덮었는지. update가 되는지 확인
// AddVMs, RemoveVMs 에서 사용 예정
func (nlbHandler *GCPNLBHandler) insertTargetPool(regionID string, reqTargetPool compute.TargetPool) (*compute.TargetPool, error) {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	// requestBody
	req, err := nlbHandler.Client.TargetPools.Insert(projectID, regionID, &reqTargetPool).Do()
	if err != nil {
		return &compute.TargetPool{}, err
	}
	printToJson(req)
	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.Name, false)
	if err != nil {
		return &compute.TargetPool{}, err
	}

	id := req.Id
	name := req.Name
	cblogger.Info("id = ", id, " : name = ", name)
	result, err := nlbHandler.getTargetPool(regionID, reqTargetPool.Name)
	if err != nil {
		return &compute.TargetPool{}, err
	}

	cblogger.Info("insertTargetPool return targetpool ", result)
	return result, nil
}

// TargetPool 조회
// Target pools are used for network TCP/UDP load balancing. A target pool references member instances, an associated legacy HttpHealthCheck resource, and, optionally, a backup target poo
func (nlbHandler *GCPNLBHandler) getTargetPool(regionID string, targetPoolName string) (*compute.TargetPool, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.TargetPools.Get(projectID, regionID, targetPoolName).Do()
	if err != nil {
		cblogger.Error("TargetPools.Get ", err)
		return nil, err
	}
	return resp, nil
}

// TargetPool 목록 조회
// Target pools are used for network TCP/UDP load balancing. A target pool references member instances, an associated legacy HttpHealthCheck resource, and, optionally, a backup target poo
func (nlbHandler *GCPNLBHandler) listTargetPools(regionID string, filter string) (*compute.TargetPoolList, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.TargetPools.List(projectID, regionID).Do()
	if err != nil {
		cblogger.Error("TargetPools.List ", err)
		return &compute.TargetPoolList{}, err
	}
	printToJson(resp)
	for _, item := range resp.Items {
		cblogger.Info(item)
	}

	return resp, nil
}

// nlbHandler.Client.TargetPools.AggregatedList(projectID) : 해당 project의 모든 region 에 대해 region별  target pool 목록

/*
	getHealth : Instance의 가장 최근의 health check result
// instanceReference 는 instarce의 url을 인자로 갖는다.
// targetPools.get(targetPoolName)  을 통해 instalces[]을 알 수 있음. 배열에서 하나씩 꺼내서 instanceReference에 넣고 사용.
// Target pools are used for network TCP/UDP load balancing. A target pool references member instances, an associated legacy HttpHealthCheck resource, and, optionally, a backup target poo

	//https://www.googleapis.com/compute/v1/projects/myproject/zones/zoneName/instances/lbname

*/
func (nlbHandler *GCPNLBHandler) getTargetPoolHealth(regionID string, targetPoolName string, instanceUrl string) (*compute.TargetPoolInstanceHealth, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	instanceReference := &compute.InstanceReference{Instance: instanceUrl}

	// requestBody
	resp, err := nlbHandler.Client.TargetPools.GetHealth(projectID, regionID, targetPoolName, instanceReference).Do()
	if err != nil {
		cblogger.Error("TargetPools.GetHealth ", err)
		return nil, err
	}
	return resp, nil
}

/*
	health checker 생성
	targetPoolName = health checker name
	health checker는 독립적임. HttpHealthChecks의 insert
	TimeoutSec should be less than checkIntervalSec
*/
func (nlbHandler *GCPNLBHandler) insertHealthCheck(regionID string, targetPoolName string, healthCheckerInfo *irs.HealthCheckerInfo) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	printToJson(healthCheckerInfo)
	port, err := strconv.ParseInt(healthCheckerInfo.Port, 10, 64)
	if err != nil {
		return err
	}
	// queryParam
	reqHealthCheck := compute.HttpHealthCheck{}
	reqHealthCheck.Name = targetPoolName
	reqHealthCheck.HealthyThreshold = int64(healthCheckerInfo.Threshold)
	reqHealthCheck.UnhealthyThreshold = int64(healthCheckerInfo.Threshold)
	reqHealthCheck.CheckIntervalSec = int64(healthCheckerInfo.Interval)
	reqHealthCheck.TimeoutSec = int64(healthCheckerInfo.Timeout)
	reqHealthCheck.Port = port
	//reqHealthCheck.RequestPath = healthChecker.
	reqHealthCheck.TimeoutSec = int64(healthCheckerInfo.Timeout)
	printToJson(reqHealthCheck)
	// requestBody
	req, err := nlbHandler.Client.HttpHealthChecks.Insert(projectID, &reqHealthCheck).Do()
	if err != nil {
		return err
	}
	printToJson(req)
	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.Name, true)
	if err != nil {
		return err
	}

	healthCheckerInfo.CspID = req.TargetLink
	printToJson(req)
	return nil
}

/*
	등록된 healthchecker 수정
	health checker는 독립적임. HttpHealthChecks의 patch
	patch 가 맞는지, update 가 맞는지 확인 필요
Updates a HttpHealthCheck resource in the specified project using the data included in the request.
This method supports PATCH semantics and uses the JSON merge patch format and processing rules.

	default 설정 : targetPoolName = healthCheckerName
	healthCheckInfo.CspID = healthChecker의 URL
		- URL의 마지막이 healthCheckerName임.

port : default 80, (0은 수정안함)
checkIntervalSec : default 5, (0은 수정안함)
timeoutSec : default 5 . checkIntervalSec보다 커야 함. 같아도 됨. (0은 수정안함)
unhealthyThreshold : default 2. (0은 수정안함)
healthThreshold : default 2. (0은 수정안함)
*/
func (nlbHandler *GCPNLBHandler) patchHealthCheck(regionID string, targetPoolName string, healthCheckerInfo *irs.HealthCheckerInfo) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	// queryParam
	printToJson(healthCheckerInfo)
	port, err := strconv.ParseInt(healthCheckerInfo.Port, 10, 64)
	if err != nil {
		return err
	}
	// queryParam
	reqHealthCheck := compute.HttpHealthCheck{}

	healthCheckerUrl := healthCheckerInfo.CspID
	if strings.EqualFold(healthCheckerUrl, String_Empty) {
		reqHealthCheck.Name = targetPoolName
	} else {
		healthCheckNameIndex := strings.LastIndex(healthCheckerUrl, StringSeperator_Slash)
		realHealthCheckName := healthCheckerUrl[(healthCheckNameIndex + 1):]
		reqHealthCheck.Name = realHealthCheckName
	}

	reqHealthCheck.HealthyThreshold = int64(healthCheckerInfo.Threshold)
	reqHealthCheck.UnhealthyThreshold = int64(healthCheckerInfo.Threshold)
	reqHealthCheck.CheckIntervalSec = int64(healthCheckerInfo.Interval)
	reqHealthCheck.TimeoutSec = int64(healthCheckerInfo.Timeout)
	reqHealthCheck.Port = port
	printToJson(reqHealthCheck)
	// requestBody
	req, err := nlbHandler.Client.HttpHealthChecks.Patch(projectID, targetPoolName, &reqHealthCheck).Do()
	if err != nil {
		return err
	}
	printToJson(req)
	if !strings.EqualFold(req.Status, RequestStatus_DONE) {
		err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.Name, true)
		if err != nil {
			return err
		}
	} // 이미 있는 경우 : Invalid value for field 'resource.healthCheck': 'https://www.googleapis.com/compute/v1/projects/yhnoh-335705/global/httpHealthChecks/test-lb-seoul-healthchecker'. Target pools have a healthCheck limit of 1

	return nil
}

/*
	등록된 healthchecker 삭제
	health checker는 독립적임.

	health checker Url 이 있으면 해당값이 제일 정확하므로 url에서 이름을 추충하여 사용
*/
func (nlbHandler *GCPNLBHandler) removeHttpHealthCheck(targetPoolName string, healthCheckUrl string) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID
	regionID := nlbHandler.Region.Region

	// TargetPool 조회하여 그 안에 있는 값 사용
	// TargetPool이 없으면 targetPool이름으로 삭제 시도
	targetPool, err := nlbHandler.getTargetPool(regionID, targetPoolName)
	if err != nil {
		cblogger.Info("targetPoolList  list for removeHttpHealthCheck: ", err)
	}
	// queryParam

	// requestBody
	healthCheckerName := targetPoolName
	if targetPool != nil {
		healthCheckIIDs := targetPool.HealthChecks
		cblogger.Info("removeHttpHealthCheck : ", healthCheckIIDs)
		for _, healthCheckUrl := range healthCheckIIDs {
			healthCheckerIndex := strings.LastIndex(healthCheckUrl, StringSeperator_Slash)
			healthCheckerName = healthCheckUrl[(healthCheckerIndex + 1):]
			cblogger.Info("removeHttpHealthCheck by targetPool ", healthCheckerName)
			req, err := nlbHandler.Client.HttpHealthChecks.Delete(projectID, healthCheckerName).Do()
			if err != nil {
				cblogger.Info("HttpHealthChecks.Delete : ", err)
				return err
			}
			err = WaitUntilComplete(nlbHandler.Client, projectID, String_Empty, req.Name, true)
			if err != nil {
				cblogger.Info("HttpHealthChecks.Delete WaitUntilComplete : ", err)
				return err
			}
		}
	} else {
		cblogger.Info("removeHttpHealthCheck by ", healthCheckerName)
		req, err := nlbHandler.Client.HttpHealthChecks.Delete(projectID, healthCheckerName).Do()
		if err != nil {
			cblogger.Info("2HttpHealthChecks.Delete : ", err)
			return err
		}
		err = WaitUntilComplete(nlbHandler.Client, projectID, String_Empty, req.Name, true)
		if err != nil {
			cblogger.Info("HttpHealthChecks.Delete WaitUntilComplete : ", err)
			return err
		}
	}
	return nil
}

/*
// Target Pool에 health check 추가 : url만 주면 됨.
// health check는 instance url이 있어야 하므로 갖고 있는 곳에서 목록조회
// add는 성공여부만
// Target pools are used for network TCP/UDP load balancing. A target pool references member instances, an associated legacy HttpHealthCheck resource, and, optionally, a backup target poo

	health check를 추가하면 해당 health checker가 선택되는가
*/
func (nlbHandler *GCPNLBHandler) addTargetPoolHealthCheck(regionID string, targetPoolName string, healthCheckerName string) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	// queryParam
	reqHealthCheck := compute.TargetPoolsAddHealthCheckRequest{}
	healthCheckReferenceList := []*compute.HealthCheckReference{}
	healthCheckReference := compute.HealthCheckReference{}

	healthCheckReference.HealthCheck = healthCheckerName
	healthCheckReferenceList = append(healthCheckReferenceList, &healthCheckReference)
	// requestBody
	req, err := nlbHandler.Client.TargetPools.AddHealthCheck(projectID, regionID, targetPoolName, &reqHealthCheck).Do()
	printToJson(req)
	if !strings.EqualFold(req.Status, RequestStatus_DONE) {
		err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.Name, true)
		if err != nil {
			return err
		}
	} // 이미 있는 경우 : Invalid value for field 'resource.healthCheck': 'https://www.googleapis.com/compute/v1/projects/yhnoh-335705/global/httpHealthChecks/test-lb-seoul-healthchecker'. Target pools have a healthCheck limit of 1

	return nil
}

/*
 TargetPool에 등록되어 있는 health checker 제거(링크만 제거)
	TargetPool에 healthChecker는 없을 수도 있음. 없어도 오류가 아님.
	실제 healthCheck는 살아있음. httpHealthCheck 제거는 nlbHandler.removeHealthCheck
*/
func (nlbHandler *GCPNLBHandler) removeTargetPoolHealthCheck(regionID string, targetPoolName string, healthCheckerName string) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	// queryParam
	reqHealthCheck := compute.TargetPoolsRemoveHealthCheckRequest{}
	healthCheckReferenceList := []*compute.HealthCheckReference{}
	healthCheckReference := compute.HealthCheckReference{}

	healthCheckReference.HealthCheck = healthCheckerName
	healthCheckReferenceList = append(healthCheckReferenceList, &healthCheckReference)
	// requestBody
	req, err := nlbHandler.Client.TargetPools.RemoveHealthCheck(projectID, regionID, targetPoolName, &reqHealthCheck).Do()
	printToJson(req)
	if !strings.EqualFold(req.Status, RequestStatus_DONE) {
		err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.Name, true)
		if err != nil {
			return err
		}
	}
	return nil
}

/*
// TargetPool 에 Instance bind추가
	parameter instance = The URL for a specific instance

// Target pools are used for network TCP/UDP load balancing. A target pool references member instances, an associated legacy HttpHealthCheck resource, and, optionally, a backup target poo
*/
func (nlbHandler *GCPNLBHandler) addTargetPoolInstance(regionID string, targetPoolName string, instanceIIDs *[]irs.IID) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID
	zoneID := nlbHandler.Region.Zone

	// TODO : 해당 region 아래의 모든 zone을 검색하여 조회해야 할 듯. 특정 zone으로만 조회해서는 vm을 제대로 찾을 수 없음.

	if instanceIIDs != nil {
		// queryParam
		instanceRequest := compute.TargetPoolsAddInstanceRequest{}
		instanceReferenceList := []*compute.InstanceReference{}
		for _, instance := range *instanceIIDs {
			//instanceUrl := instance.SystemId
			instanceUrl, err := nlbHandler.getVmUrl(zoneID, instance)
			if err != nil {
				return err
			}
			instanceReference := &compute.InstanceReference{Instance: instanceUrl}
			instanceReferenceList = append(instanceReferenceList, instanceReference)
		}
		instanceRequest.Instances = instanceReferenceList

		// requestBody
		res, err := nlbHandler.Client.TargetPools.AddInstance(projectID, regionID, targetPoolName, &instanceRequest).Do()
		if err != nil {
			return err
		}

		printToJson(res)
		err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, res.Name, false)
		if err != nil {
			return err
		}
		cblogger.Info("Done")
		return nil
	}
	return errors.New("instanceIIDs are empty.)")
}

/*
	TargetPool에서 instance bind 삭제
	parameter instance = The URL for a specific instance
*/
func (nlbHandler *GCPNLBHandler) removeTargetPoolInstances(regionID string, targetPoolName string, deleteInstanceIIDs *[]irs.IID) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID
	zoneID := nlbHandler.Region.Zone

	if deleteInstanceIIDs != nil {
		instanceRequest := compute.TargetPoolsRemoveInstanceRequest{}
		instanceReferenceList := []*compute.InstanceReference{}
		for _, instance := range *deleteInstanceIIDs {
			//instanceUrl := instance.SystemId
			instanceUrl, err := nlbHandler.getVmUrl(zoneID, instance)
			if err != nil {
				return err
			}
			instanceReference := &compute.InstanceReference{Instance: instanceUrl}
			instanceReferenceList = append(instanceReferenceList, instanceReference)
		}
		instanceRequest.Instances = instanceReferenceList

		// requestBody
		res, err := nlbHandler.Client.TargetPools.RemoveInstance(projectID, regionID, targetPoolName, &instanceRequest).Do()
		if err != nil {
			return err
		}

		printToJson(res)
		err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, res.Name, false)
		if err != nil {
			return err
		}
		cblogger.Info("Done")
		return nil
	}
	return errors.New("instanceIIDs are empty.)")
}

// slice 제거
func removeElementByIndex(slice []string, index int) []string {
	//func removeElementByIndex[T string](slice []T, index int) []T {
	sliceLen := len(slice)
	sliceLastIndex := sliceLen - 1

	if index != sliceLastIndex {
		slice[index] = slice[sliceLastIndex]
	}

	return slice[:sliceLastIndex]
}

/*
	TargetPool 삭제
	: 꼬였을 때 강제 삭제용
	ex) Targetpool 생성 후 forwardingRule 생성중 오류발생 시, Targetpool을 console에서 삭제가 안됨.

	remove : 일부 삭제
	delete : 전체 삭제

// Target pools are used for network TCP/UDP load balancing. A target pool references member instances, an associated legacy HttpHealthCheck resource, and, optionally, a backup target poo
*/
func (nlbHandler *GCPNLBHandler) removeTargetPool(regionID string, targetPoolName string) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	// requestBody
	res, err := nlbHandler.Client.TargetPools.Delete(projectID, regionID, targetPoolName).Do()
	if err != nil {
		cblogger.Error(err)
		return err
	}

	printToJson(res)
	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, res.Name, false)
	if err != nil {
		cblogger.Error(err)
		return err
	}
	cblogger.Info("Done")

	return nil
}

// Target pools are used for network TCP/UDP load balancing. A target pool references member instances, an associated legacy HttpHealthCheck resource, and, optionally, a backup target poo
//func (nlbHandler *GCPNLBHandler) aggregatedTargetPoolsList() (*compute.TargetPoolList, error) {
//
//}

// Target pools are used for network TCP/UDP load balancing. A target pool references member instances, an associated legacy HttpHealthCheck resource, and, optionally, a backup target poo
//func (nlbHandler *GCPNLBHandler) setTargetpoolBackup(healthCheckerInfo irs.HealthCheckerInfo) {
//	// path param
//	projectID := nlbHandler.Credential.ProjectID
//}

// toString 용
func printToJson(class interface{}) {
	e, err := json.Marshal(class)
	if err != nil {
		cblogger.Info(err)
	}
	cblogger.Info(string(e))
}

/*
// lister 정보를 gcp의 forwardingRule에 맞게 변경
	ipProtocol // TCP, UDP, ESP, AH, SCTP, ICMP and L3_DEFAULT
	allPorts : ports/portRange/allPorts 세가지 중 1가지만 사용가능
	  - TCP, UDP and SCTP traffic, packets addressed to any ports will be forwarded to the target or backendService.
	portRange
      - load balancing scheme 가 EXTERNAL, INTERNAL_SELF_MANAGED or INTERNAL_MANAGED 일 때,
      - IPProtocol 가 TCP, UDP or SCTP 일 떄 사용가능
      - targetPool이나 backendService로 전달
	ports[] : backendService로 직접전달 시 사용
	allowGlobalAccess :
	  - If the field is set to TRUE, clients can access ILB from all regions.
	  - Otherwise only allows access from clients in the same region as the internal load balancer.

	target
	subnetwork : custom mode 또는 ipv6로 external forwarding rule 생성 시 필요
	network : external load balancing 에서 사용 안함.
	networkTier :
	  . If this field is not specified, it is assumed to be PREMIUM.
	  . If IPAddress is specified, this value must be equal to the networkTier of the Address.
	  - Region forwording rule : PREMIUM and STANDARD
	  - Global forwording rule : PREMIUM only
	backendService : Required for Internal TCP/UDP Load Balancing and Network Load Balancing; must be omitted for all other load balancer types.
	ipVersion : IPV4 or IPV6. This can only be specified for an external global forwarding rule.

*/
func convertNlbInfoToForwardingRule(nlbListener irs.ListenerInfo, targetPool *compute.TargetPool) compute.ForwardingRule {
	//ipProtocol // TCP, UDP, ESP, AH, SCTP, ICMP and L3_DEFAULT
	//portRange
	loadBalancerScheme := GCP_ForwardingRuleScheme_EXTERNAL

	//{
	//	"IPAddress":"34.64.173.241",
	//	"IPProtocol":"TCP",
	//	"creationTimestamp":"2022-05-23T23:13:06.742-07:00",
	//	"fingerprint":"7ciRJbD4Ht8=","id":"2343947829076150685",
	//	"kind":"compute#forwardingRule","loadBalancingScheme":"EXTERNAL",
	//	"name":"test-lb-seoul-frontend",
	//	"networkTier":"PREMIUM",
	//	"portRange":"80-80",
	//	"region":"https://www.googleapis.com/compute/v1/projects/yhnoh-335705/regions/asia-northeast3",
	//	"selfLink":"https://www.googleapis.com/compute/v1/projects/yhnoh-335705/regions/asia-northeast3/forwardingRules/test-lb-seoul-frontend",
	//	"target":"https://www.googleapis.com/compute/v1/projects/yhnoh-335705/regions/asia-northeast3/targetPools/test-lb-seoul"
	//	}

	//nlbListener := nlbInfo.Listener
	newForwardingRule := compute.ForwardingRule{
		Name:                targetPool.Name, // forwardingRule Name == targetPool Name
		LoadBalancingScheme: loadBalancerScheme,
		IPProtocol:          nlbListener.Protocol,
		IPAddress:           nlbListener.IP,
		PortRange:           nlbListener.Port,

		Target: targetPool.SelfLink, //Must be either a valid In-Project Forwarding Rule Target URL, a valid Service Attachment URL, or a supported Google API bundle
		/// from listener
		//DNSName      string
		//KeyValueList []KeyValue

		/// to forwardingRule
		//Description              string            `json:"description,omitempty"`

	}
	return newForwardingRule
}

/*
//
	NLB 생성을 위해 요청받은 nlbInfo 정보를 gcp의 TargetPool에 맞게 변경
	FailoverRatio : 설정 시 backupPool도 설정해야 함.
	vmID 는 url형태가 아니므로 vm을 조회하여 selflink를 set
	Instances[] : resource URLs
	HealthChecks[] : resource URLs

  vmGroup = TargetPool
  vmGroup.cspId = targetPoolName, lbName

	ex)
	//"healthChecks":[
	//					"https://www.googleapis.com/compute/v1/projects/myproject/global/httpHealthChecks/test-lb-seoul-healthchecker"
	//					],
	//"instances":[
	//					"https://www.googleapis.com/compute/v1/projects/myproject/zones/asia-northeast3-a/instances/test-lb-seoul-01"
	//					]
*/
func (nlbHandler *GCPNLBHandler) convertNlbInfoToTargetPool(nlbInfo *irs.NLBInfo) (compute.TargetPool, error) {
	vmList := nlbInfo.VMGroup.VMs

	projectID := nlbHandler.Credential.ProjectID
	//regionID := nlbHandler.Region.Region
	zoneID := nlbHandler.Region.Zone

	instancesUrl := []string{}
	for _, instance := range *vmList {
		// get instance url from instance id
		//instanceUrl := "https://www.googleapis.com/compute/v1/projects/" + projectID + "/zones/" + zoneID + "/instances/" + instance.SystemId
		//instancesUrl = append(instancesUrl, instanceUrl) // URL

		vm, err := nlbHandler.Client.Instances.Get(projectID, zoneID, instance.SystemId).Do()
		if err != nil {
			cblogger.Error(err)
			return compute.TargetPool{}, err
		}
		instancesUrl = append(instancesUrl, vm.SelfLink) // URL

		printToJson(instancesUrl)
	}

	healthChecks := []string{nlbInfo.HealthChecker.CspID} // url

	targetPool := compute.TargetPool{
		Name:         nlbInfo.IId.NameId,
		Instances:    instancesUrl,
		HealthChecks: healthChecks,
	}
	return targetPool, nil
}

/*
	가져온 TargetPool정보를 Spider 의 NLBInfo로 변환
	extractVmGroup 은 추출만 하면 됨.
	extractHealthChecker는 health checker 정보를 조회해야 하므로 nlbHandler 필요
*/
func (nlbHandler *GCPNLBHandler) convertTargetPoolToNLBInfo(targetPool *compute.TargetPool, nlbInfo *irs.NLBInfo) error {
	regionID := nlbHandler.Region.Region

	// VM Group 정보 추출
	nlbInfo.VMGroup = extractVmGroup(targetPool, nlbInfo)

	// health checker 정보 추출
	// health checker에 대한 ID는 가지고 있으나 내용은 갖고 있지 않아 정보 조회 필요.
	healthChecker, err := nlbHandler.extractHealthChecker(regionID, targetPool)
	if err != nil {
		return err
	}
	nlbInfo.HealthChecker = healthChecker

	// vpc 정보 추출
	for _, instanceUrl := range targetPool.Instances {
		targetPoolInstanceArr := strings.Split(instanceUrl, StringSeperator_Slash)

		instanceName := targetPoolInstanceArr[len(targetPoolInstanceArr)-1]
		instanceZone := targetPoolInstanceArr[len(targetPoolInstanceArr)-3]
		vpcIID, err := nlbHandler.getVPCInfoFromVM(instanceZone, irs.IID{SystemId: instanceName})
		if err != nil {
			return err
		}
		nlbInfo.VpcIID = vpcIID
		break
	}

	return nil
}

/*
	forwarding rule의 Port가 Range 이나 Spider에서는 1개만 사용하므로 첫번째 값만 사용
*/
func convertRegionForwardingRuleToNlbListener(forwardingRule *compute.ForwardingRule) irs.ListenerInfo {
	portRange := forwardingRule.PortRange
	portArr := strings.Split(portRange, StringSeperator_Hypen)
	listenerInfo := irs.ListenerInfo{
		Protocol: forwardingRule.IPProtocol,
		IP:       forwardingRule.IPAddress,
		Port:     portArr[0],
		//DNSName:  forwardingRule., // 향후 사용할 때 Adderess에서 가져올 듯
		CspID: forwardingRule.Name, // forwarding rule name 전체
		//KeyValueList:
	}
	return listenerInfo
}

// Listener 만 조회
func (nlbHandler *GCPNLBHandler) getListenerByNlbSystemID(nlbIID irs.IID) (irs.ListenerInfo, error) {
	listenerInfo := irs.ListenerInfo{}
	regionID := nlbHandler.Region.Region

	// region forwarding rule 는 target pool 과 lb이름으로 엮임.
	// map에 nb이름으로 nbInfo를 넣고 해당 값들 추가해서 조합
	//nlbID := nlbIID.NameId
	nlbID := nlbIID.SystemId
	// forwardingRule 조회

	cblogger.Info("region forwardingRules start: ", regionID)
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   nlbHandler.Region.Zone,
		ResourceType: call.NLB,
		ResourceName: nlbID,
		CloudOSAPI:   "GetNLB()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	regionForwardingRule, err := nlbHandler.getRegionForwardingRules(regionID, nlbID)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		cblogger.Info("regionForwardingRule  list: ", err)
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.ListenerInfo{}, err
	}
	if regionForwardingRule != nil { // dial tcp: lookup compute.googleapis.com: no such host 일 때, 	panic: runtime error: invalid memory address or nil pointer dereference
		listenerInfo = convertRegionForwardingRuleToNlbListener(regionForwardingRule)

		loadBalancerType := regionForwardingRule.LoadBalancingScheme
		if strings.EqualFold(loadBalancerType, GCP_ForwardingRuleScheme_EXTERNAL) {
			loadBalancerType = SPIDER_LoadBalancerType_PUBLIC
		}

		createTimeKeyValue := irs.KeyValue{Key: "createdTime", Value: regionForwardingRule.CreationTimestamp}
		loadBalancerTypeKeyValue := irs.KeyValue{Key: "loadBalancerType", Value: loadBalancerType}
		keyValueList := []irs.KeyValue{}
		keyValueList = append(keyValueList, createTimeKeyValue)
		keyValueList = append(keyValueList, loadBalancerTypeKeyValue)
		listenerInfo.KeyValueList = keyValueList
	}

	return listenerInfo, nil
}

/*
// TargetPool = backend = vmGroup 이며
	가져온 targetPool을 spider에서 사용하는 vmGroup으로 변환하여 return

	ex) vmGroup
	//"name":"lb-test-seoul-03",
	//"selfLink":"https://www.googleapis.com/compute/v1/projects/yhnoh-335705/regions/asia-northeast3/targetPools/lb-test-seoul-03",
	//targetPoolIndex := strings.LastIndex(targetPool.SelfLink, "/")
	//targetPoolValue := targetPool.SelfLink[(targetPoolIndex + 1):]
*/
func extractVmGroup(targetPool *compute.TargetPool, nlbInfo *irs.NLBInfo) irs.VMGroupInfo {
	//vmGroup := irs.VMGroupInfo{}
	vmGroup := nlbInfo.VMGroup

	//targetPool, err := nlbHandler.getTargetPool(regionID, targetPoolName)
	//if err != nil {
	//	cblogger.Info("targetPoolList  list: ", err)
	//}
	if targetPool.Instances != nil {
		printToJson(targetPool)

		// instances iid set
		instanceIIDs := []irs.IID{}

		for _, instanceUrl := range targetPool.Instances {
			targetPoolInstanceIndex := strings.LastIndex(instanceUrl, StringSeperator_Slash)
			targetPoolInstanceValue := instanceUrl[(targetPoolInstanceIndex + 1):]

			instanceIID := irs.IID{SystemId: targetPoolInstanceValue}
			//instanceIID := irs.IID{NameId: targetPoolInstanceValue, SystemId: instanceUrl}
			instanceIIDs = append(instanceIIDs, instanceIID)
		}

		//네트워크 부하 분산기는 주소, 포트, 프로토콜 유형과 같은 수신 IP 프로토콜 데이터를 기준으로 부하를 시스템에 분산합니다.
		//네트워크 부하 분산기는 패스스루 부하 분산기이므로 백엔드는 원래 클라이언트 요청을 수신합니다.
		//네트워크 부하 분산기는 전송 계층 보안(TLS) 오프로드 또는 프록시를 수행하지 않습니다. 트래픽은 VM으로 직접 라우팅됩니다.
		vmGroup.CspID = targetPool.Name
		vmGroup.VMs = &instanceIIDs
	}
	return vmGroup
}

/*
	targetPool에서 healthcheker를 가져와서 spider의 HealthCheckerInfo 로 return
*/
func (nlbHandler *GCPNLBHandler) extractHealthChecker(regionID string, targetPool *compute.TargetPool) (irs.HealthCheckerInfo, error) {
	returnHealthChecker := irs.HealthCheckerInfo{}
	//targetPoolName := targetPool.Name
	//targetPool, err := nlbHandler.getTargetPool(regionID, targetPoolName)
	//if err != nil {
	//	cblogger.Info("targetPoolList  list: ", err)
	//}

	if targetPool.Instances != nil {
		printToJson(targetPool)

		// health checker에 대한 ID는 가지고 있으나 내용은 갖고 있지 않아 정보 조회 필요.
		for _, healthChecker := range targetPool.HealthChecks {
			printToJson(healthChecker)
			targetHealthCheckerIndex := strings.LastIndex(healthChecker, StringSeperator_Slash)
			targetHealthCheckerValue := healthChecker[(targetHealthCheckerIndex + 1):]

			//cblogger.Info("GlobalHttpHealthChecks start: ", regionID, " : "+targetHealthCheckerValue)
			//targetHealthCheckerInfo, err := nlbHandler.getRegionHealthChecks(regionID, targetHealthCheckerValue)
			targetHealthCheckerInfo, err := nlbHandler.getHttpHealthChecks(targetHealthCheckerValue)
			//targetHealthCheckerInfo, err := nlbHandler.getHttpHealthChecks(targetPoolName) // healthchecker는 전역
			if err != nil {
				cblogger.Info("targetHealthCheckerInfo : ", err)
				return returnHealthChecker, err
			}
			if targetHealthCheckerInfo != nil {
				printToJson(targetHealthCheckerInfo)

				returnHealthChecker.CspID = targetHealthCheckerInfo.SelfLink
				returnHealthChecker.Protocol = HealthCheck_Http // 고정
				returnHealthChecker.Port = strconv.FormatInt(targetHealthCheckerInfo.Port, 10)
				returnHealthChecker.Interval = int(targetHealthCheckerInfo.CheckIntervalSec)
				returnHealthChecker.Timeout = int(targetHealthCheckerInfo.TimeoutSec)
				returnHealthChecker.Threshold = int(targetHealthCheckerInfo.HealthyThreshold)
				//healthChecker.KeyValueList[], KeyValue
			}
			cblogger.Info("GlobalHttpHealthChecks end: ")
		}

	}
	return returnHealthChecker, nil
}

/*
	vpc를 가져오기 위해 vm 정보를 조회.
	zone은 다를 수 있으므로 VMHandler의 GetVM을 사용하지 않고 zone을 parameter로 받는 function을 따로 만듬
*/
func (nlbHandler *GCPNLBHandler) getVPCInfoFromVM(zoneID string, vmID irs.IID) (irs.IID, error) {
	projectID := nlbHandler.Credential.ProjectID

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   zoneID,
		ResourceType: call.NLB,
		ResourceName: vmID.SystemId,
		CloudOSAPI:   "getVM()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	vm, err := nlbHandler.Client.Instances.Get(projectID, zoneID, vmID.SystemId).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return irs.IID{}, err
	}
	//callogger.Info(call.String(callLogInfo))
	//spew.Dump(vm)

	////Network: (string) (len=87) "https://www.googleapis.com/compute/v1/projects/[projectID]/global/networks/[vpcName]",
	////NetworkIP: (string) (len=8) "10.0.0.6",
	////Subnetwork: (string) (len=110) "https://www.googleapis.com/compute/v1/projects/[projectID]/regions/[regionID]/subnetworks/[subnetName]",
	vpcUrl := vm.NetworkInterfaces[0].Network
	////subnetUrl := vm.NetworkInterfaces[0].Subnetwork
	vpcArr := strings.Split(vpcUrl, StringSeperator_Slash)
	////subnetArr := strings.Split(subnetUrl, StringSeperator_Slash)
	vpcName := vpcArr[len(vpcArr)-1]
	////subnetName := subnetArr[len(subnetArr)-1]
	//vpcIID := irs.IID{NameId: vpcName, SystemId: vpcUrl}
	vpcIID := irs.IID{NameId: vpcName, SystemId: vpcName}

	//infoVPC, err := nlbHandler.Client.Networks.Get(projectID, vm.Name).Do()
	//if err != nil {
	//	cblogger.Error(err)
	//	return irs.IID{}, err
	//}
	//return irs.IID{NameId: infoVPC.Name, SystemId: infoVPC.Name	}

	return vpcIID, nil
}

/*
	vm의 url 조회
	zone은 다를 수 있으므로 VMHandler의 GetVM을 사용하지 않고 zone을 parameter로 받는 function을 따로 만듬
*/
func (nlbHandler *GCPNLBHandler) getVmUrl(zoneID string, vmID irs.IID) (string, error) {
	projectID := nlbHandler.Credential.ProjectID

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.GCP,
		RegionZone:   zoneID,
		ResourceType: call.NLB,
		ResourceName: vmID.SystemId,
		CloudOSAPI:   "getVM()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()
	vm, err := nlbHandler.Client.Instances.Get(projectID, zoneID, vmID.SystemId).Do()
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		cblogger.Error(err)
		return "", err
	}
	callogger.Info(call.String(callLogInfo))
	spew.Dump(vm)

	return vm.SelfLink, nil
}

/*
// CreateNLB validation check
	nlb이름이 같은 resource가 있는지 check
*/
func (nlbHandler *GCPNLBHandler) validateCreateNLB(nlbReqInfo irs.NLBInfo) error {
	//// validation check area
	cblogger.Info("validateCreateNLB")
	regionID := nlbHandler.Region.Region

	nlbName := nlbReqInfo.IId.NameId

	// targetPool
	_, err := nlbHandler.getTargetPool(regionID, nlbName) // 없으면 error
	if err != nil {
		is404, checkErr := checkErrorCode(ErrorCode_NotFound, err) // 404 : not found면 pass
		if !is404 || !checkErr {                                   // 하나라도 false 면 error return
			cblogger.Info("existsTargetPoolChecks : ", err)
			return err
		}

	} else {
		// 이미 있으므로 Error
		return errors.New("Load Balancer already exists ")
	}

	// healthCheck
	_, err = nlbHandler.getHttpHealthChecks(nlbName)
	if err != nil {
		is404, checkErr := checkErrorCode(ErrorCode_NotFound, err) // 404 : not found면 pass
		if !is404 || !checkErr {                                   // 하나라도 false 면 error return
			cblogger.Info("existsHealthChecks : ", err)
			return err
		}
	} else {
		// 이미 있으므로 Error
		return errors.New("Load Balancer health checker already exists ")
	}

	// forwarding rule
	_, err = nlbHandler.getRegionForwardingRules(regionID, nlbName)
	if err != nil {
		is404, checkErr := checkErrorCode(ErrorCode_NotFound, err) // 404 : not found면 pass
		if !is404 || !checkErr {                                   // 하나라도 false 면 error return
			cblogger.Info("existsListener : ", err)
			return err
		}
	} else {
		// 이미 있으므로 Error
		return errors.New("Load Balancer listener already exists ")
	}

	checkIntervalSec := int64(nlbReqInfo.HealthChecker.Interval)
	if err != nil {
		return err
	}
	timeoutSec := int64(nlbReqInfo.HealthChecker.Timeout)
	if err != nil {
		return err
	}

	if checkIntervalSec < timeoutSec {
		return errors.New("The healthchecker's TimeoutSec should be less than IntervalSec ")
	}

	return nil
}

/*
// DeleteNLB validation check
	체크하면서 validation이 실패하면 errors.New("message ")로 map에 추가

	삭제 전 대상 resource가 없는지 checkeck. 셋 다 있으면 OK.
	없으면 없는 항목 return
	모두 없으면 error
*/
func (nlbHandler *GCPNLBHandler) validateDeleteNLB(nlbIID irs.IID) (map[string]error, error) {
	//// validation check area

	validationMap := make(map[string]error)

	cblogger.Info("validateDeleteNLB")
	regionID := nlbHandler.Region.Region

	nlbName := nlbIID.NameId

	//existTargetPool := false
	//existForwardingRule := false
	//existHealthChecker := false

	// targetPool
	_, targetPoolErr := nlbHandler.getTargetPool(regionID, nlbName) // 없으면 not found error
	validationMap[NLB_Component_TARGETPOOL] = targetPoolErr
	//if targetPoolErr != nil {
	//
	//	// notFound 든 뭐든 따질 필요 없이 Error면 Error이네.
	//	//is404, checkErr := checkErrorCode(ErrorCode_NotFound, targetPoolErr) // 404 : not found면 안됨
	//	//if is404 && checkErr {
	//	//	cblogger.Info("existsListener : ", targetPoolErr)
	//	//}
	//	//cblogger.Info("TargetPool not found : ", targetPoolErr)
	//	// 없네?
	//} else {
	//	existTargetPool = true
	//}

	// healthCheck
	_, healthCheckErr := nlbHandler.getHttpHealthChecks(nlbName)
	validationMap[NLB_Component_FORWARDINGRULE] = healthCheckErr
	//if healthCheckErr != nil {
	//	cblogger.Info("HealthChecks not found : ", healthCheckErr)
	//} else {
	//	existHealthChecker = true
	//}

	// forwarding rule
	_, forwardingRuleErr := nlbHandler.getRegionForwardingRules(regionID, nlbName)
	validationMap[NLB_Component_HEALTHCHECKER] = forwardingRuleErr
	//if forwardingRuleErr != nil {
	//	cblogger.Info("Listener not found : ", forwardingRuleErr)
	//} else {
	//	existForwardingRule = true
	//}

	//for key, errResult := range validationMap {
	//	fmt.Println(key, errResult)
	//	if errResult != nil {
	//		if strings.EqualFold(NLB_Component_TARGETPOOL, key){
	//
	//		}
	//		if strings.EqualFold(NLB_Component_TARGETPOOL, key){
	//
	//		}
	//		if strings.EqualFold(NLB_Component_TARGETPOOL, key){
	//
	//		}
	//	}
	//}

	// 셋 다 없으면 없는 resource 삭제 요청임.
	countMap := 0
	count404 := 0
	for validKey, validErr := range validationMap {
		cblogger.Info("validMap "+validKey, validErr)
		isValidCode, isValidErrorFormat := checkErrorCode(ErrorCode_NotFound, validErr)
		cblogger.Info("validMap isValidCode ", isValidCode, isValidErrorFormat)
		if isValidCode && isValidErrorFormat {
			count404++
		}
		countMap++
	}
	cblogger.Info(countMap, count404)
	if countMap == count404 {
		cblogger.Info("all not found : ")
		return nil, nil
	}
	return validationMap, nil
}

/*
	GCP의 Error는 String이기 때문에 안에 있는 Code, Message를 겍체로 변환한 후 비교하는 function
	예상하는 ErrorCode면 true, 아니면 false
	return param1 : 같은지 비교
	return param2 : Error 변환 에러 여부

		if err != nil {
				if ee, ok := err.(*googleapi.Error); ok {
						fmt.Printf("Error Code %v", ee.Code)
						fmt.Printf("Error Message %v", ee.Message)
						fmt.Printf("Error Details %v", ee.Details)
						fmt.Printf("Error Body %v", ee.Body)
				}
		}

	return 결과 : 비교결과, 비교성공 여부.   비교성공여부가 false면 error자체가 GCP의 에러가 아님.
	같으면 true, true
	다르면 false, true
	예외면 false, false
*/
func checkErrorCode(expectErrorCode int, err error) (bool, bool) {
	var errorCode int
	//err = errors.New("TestTest ")

	//var errorMessage string
	errorDetail, ok := err.(*googleapi.Error) // casting이 정상이면 ok=true, 비정상이면 ok=false
	fmt.Printf("errorDetail", errorDetail)
	fmt.Printf("ok", ok)
	if ok {
		fmt.Printf("Error Code %v", errorDetail.Code)
		fmt.Printf("Error Message %v", errorDetail.Message)
		errorCode = errorDetail.Code
		//errorMessage = errorDetail.Message
	}

	return errorCode == expectErrorCode, ok
}

/*
	에러 발생 시 자원 회수(삭제)용
	map에 해당 자원의 경로가 들어 있음.
	healthChecker = url
	targetPool = url

	오류 메시지 예시
	(1) XXX deleted
	(2) YYY deleted
	(3) ~~~~~ error.... (= CSP 반환 메시지)
*/
func (nlbHandler *GCPNLBHandler) rollbackCreatedNlbResources(regionID string, resourceMap map[string]string) string {
	rollbackResult := String_Empty

	// health checker
	if strings.EqualFold(resourceMap[NLB_Component_HEALTHCHECKER], String_Empty) {
		healthCheckerUrl := resourceMap[NLB_Component_HEALTHCHECKER]
		healthCheckerIndex := strings.LastIndex(healthCheckerUrl, StringSeperator_Slash)
		healthCheckerName := healthCheckerUrl[(healthCheckerIndex + 1):]

		err := nlbHandler.removeHttpHealthCheck(healthCheckerName, healthCheckerUrl)
		if err != nil {
			cblogger.Info("rollbackCreatedNlbResources removeHealthCheck  err: ", err)
			rollbackResult = "(1) HealthChecker delete error : " + err.Error()
			//return false, err
		} else {
			rollbackResult = "(1) HealthChecker deleted"
		}
	}

	// backend
	if strings.EqualFold(resourceMap[NLB_Component_TARGETPOOL], String_Empty) {
		targetPoolUrl := resourceMap[NLB_Component_TARGETPOOL]
		targetPoolIndex := strings.LastIndex(targetPoolUrl, StringSeperator_Slash)
		targetPoolName := targetPoolUrl[(targetPoolIndex + 1):]
		err := nlbHandler.removeTargetPool(regionID, targetPoolName)
		if err != nil {
			cblogger.Info("rollbackCreatedNlbResources removeTargetPool  err: ", err)
			cblogger.Error(err)
			rollbackResult = "(2) Targetpool delete error : " + err.Error()
			//return false, err
		} else {
			rollbackResult = "(2) Targetpool deleted"
		}
	}

	// forwarding rule 이 마지막이라. 굳이 삭제로직 태울 필요 없음.

	return rollbackResult
}
