package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	compute "google.golang.org/api/compute/v1"
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
*/
func (nlbHandler *GCPNLBHandler) CreateNLB(nlbReqInfo irs.NLBInfo) (irs.NLBInfo, error) {
	fmt.Println("CreateNLB")
	projectID := nlbHandler.Credential.ProjectID
	regionID := nlbHandler.Region.Region

	nlbName := nlbReqInfo.IId.NameId
	fmt.Println("start ", projectID)

	// backend service 없음.
	// region forwarding rule, targetpool, targetpool안의 instance에서 사용하는 healthchecker

	fmt.Println("backend TargetPool 생성 ")

	// healthChecker는 있는 것을 선택
	healthCheckId := nlbReqInfo.HealthChecker.CspID // url
	if strings.EqualFold(healthCheckId, "") {
		return irs.NLBInfo{}, errors.New("Health Checker doesn't exist")
	} else {
		// 존재여부 확인. url형태이므로 마지막에 있는 healthName만 추출
		healthCheckIndex := strings.LastIndex(nlbReqInfo.HealthChecker.CspID, "/")
		healthCheckValue := nlbReqInfo.HealthChecker.CspID[(healthCheckIndex + 1):]

		existsHealthCheck, err := nlbHandler.getHttpHealthChecks(healthCheckValue)
		if err != nil {
			fmt.Println("existsHealthCheck err ", err)
			return irs.NLBInfo{}, err
		}
		printToJson(existsHealthCheck)
	}

	// targetPool 안에 healthCheckID가 들어 감.
	newTargetPool := convertNlbInfoToTargetPool(nlbReqInfo)
	printToJson(newTargetPool)
	targetPool, err := nlbHandler.insertTargetPool(regionID, newTargetPool)
	if err != nil {
		fmt.Println("targetPoolList  err: ", err)
		return irs.NLBInfo{}, err
	}

	printToJson(targetPool)
	fmt.Println("backend TargetPool 생성 완료 ")

	fmt.Println("frontend (forwarding rule) 생성 ")
	// targetPool 생성 후 selfLink 를 forwardingRule의 target으로 set.
	newForwardingRule := convertNlbInfoToForwardingRule(nlbReqInfo.Listener, targetPool)
	err = nlbHandler.insertRegionForwardingRules(regionID, &newForwardingRule)
	if err != nil {
		fmt.Println("forwardingRule err  : ", err)
		return irs.NLBInfo{}, err
	}
	fmt.Println("forwardingRule result  : ")

	//IId:         irs.IID{NameId: targetLbValue, SystemId: targetForwardingRuleValue}, // NameId = Lb Name, SystemId = forwardingRule name
	nlbIID := irs.IID{
		NameId:   nlbName,             // lb Name = targetPool name
		SystemId: targetPool.SelfLink, // targetPool url
	}
	nlbInfo, err := nlbHandler.GetNLB(nlbIID)
	if err != nil {
		return irs.NLBInfo{}, err
	}
	//
	//fmt.Println("Targetpool end: ")
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


*/
func (nlbHandler *GCPNLBHandler) ListNLB() ([]*irs.NLBInfo, error) {
	var nlbInfoList []*irs.NLBInfo

	projectID := nlbHandler.Credential.ProjectID
	regionID := nlbHandler.Region.Region

	nlbMap := make(map[string]irs.NLBInfo)

	// instace에서 vpc의 IID 조회를 위해
	vmHandler := GCPVMHandler{
		Client:     nlbHandler.Client,
		Region:     nlbHandler.Region,
		Ctx:        nlbHandler.Ctx,
		Credential: nlbHandler.Credential,
	}

	// call logger

	// lb name, 			lb type, 					protocol, 	region, 		backend type
	// testlb-frontendlist	network(target pool-based) 	tcp			asia-northeast3	1target pool(1 instance)

	fmt.Println("projectID: ", projectID)
	fmt.Println("regionID: ", regionID)

	// 외부 dns주소가 있는 경우 region/global에서 해당 주소를 가져온다.
	//fmt.Println("region address start: ", regionID)
	//regionAddressList, err := nlbHandler.listRegionAddresses(regionID, "")
	//if err != nil {
	//	fmt.Println("region address  list: ", err)
	//}
	//printToJson(regionAddressList)
	//fmt.Println("region address end: ")
	//fmt.Println("global address start: ")
	//globalAddressList, err := nlbHandler.listGlobalAddresses("")
	//if err != nil {
	//	fmt.Println("globalAddressList  list: ", err)
	//}
	//printToJson(globalAddressList)
	//fmt.Println("global address end: ")

	// region forwarding rule 는 target pool 과 lb이름으로 엮임.
	// map에 nb이름으로 nbInfo를 넣고 해당 값들 추가해서 조합
	fmt.Println("region forwardingRules start: ", regionID)
	regionForwardingRuleList, err := nlbHandler.listRegionForwardingRules(regionID, "", "")
	if err != nil {
		fmt.Println("regionForwardingRule  list: ", err)
	}
	if regionForwardingRuleList != nil { // dial tcp: lookup compute.googleapis.com: no such host 일 때, 	panic: runtime error: invalid memory address or nil pointer dereference
		if len(regionForwardingRuleList.Items) > 0 {
			for _, forwardingRule := range regionForwardingRuleList.Items {
				targetPoolUrl := forwardingRule.Target
				targetLbIndex := strings.LastIndex(targetPoolUrl, "/")
				targetLbValue := forwardingRule.Target[(targetLbIndex + 1):]

				// targetlink에서 lb 추출
				//targetNlbInfo := nlbMap[targetLbValue]
				newNlbInfo, exists := nlbMap[targetLbValue]
				if exists {
					// spider는 1개의 listener(forwardingrule)만 사용하므로 skip
				} else {
					listenerInfo := convertRegionForwardingRuleToNlbListener(forwardingRule)

					createdTime, _ := time.Parse(
						time.RFC3339,
						forwardingRule.CreationTimestamp) // RFC3339형태이므로 해당 시간으로 다시 생성

					loadBalancerType := forwardingRule.LoadBalancingScheme
					if strings.EqualFold(loadBalancerType, "EXTERNAL") { // PUBLIC/INTERNAL : extenel -> PUBLIC으로 변경
						loadBalancerType = "PUBLIC"
					}

					newNlbInfo = irs.NLBInfo{
						IId:         irs.IID{NameId: targetLbValue, SystemId: targetPoolUrl}, // NameId :Lb Name, poolName, SystemId : targetPool Url
						VpcIID:      irs.IID{NameId: "", SystemId: ""},                       // VpcIID 는 Pool 안의 instance에 있는 값
						Type:        loadBalancerType,                                        // PUBLIC/INTERNAL : extenel -> PUBLIC으로 변경하는 로직 적용해야함.
						Scope:       "REGION",
						Listener:    listenerInfo,
						CreatedTime: createdTime, //RFC3339 "creationTimestamp":"2022-05-24T01:20:40.334-07:00"
						//KeyValueList  []KeyValue
					}

					newNlbInfo.VMGroup.Protocol = forwardingRule.IPProtocol
					newNlbInfo.VMGroup.Port = forwardingRule.PortRange
				}
				nlbMap[targetLbValue] = newNlbInfo
				printToJson(forwardingRule)
			}
		}
	}
	fmt.Println("regionForwardingRule end: ")
	fmt.Println("nlbMap end: ", nlbMap)

	fmt.Println("Targetpool start: ")

	targetPoolList, err := nlbHandler.listTargetPools(regionID, "")
	if err != nil {
		fmt.Println("targetPoolList  list: ", err)
	}
	printToJson(targetPoolList)

	vpcInstanceName := "" // vpc를 갸져올 instance 이름

	for _, targetPool := range targetPoolList.Items {
		//printToJson(targetPool)
		newNlbInfo, exists := nlbMap[targetPool.Name] // lb name
		if !exists {
			// 없으면 안됨.
			fmt.Println("targetPool.Name does not exist in nlbMap ", targetPool.Name)
			continue
		}

		// vmGroup == targetPool
		//"name":"lb-test-seoul-03",
		//"selfLink":"https://www.googleapis.com/compute/v1/projects/yhnoh-335705/regions/asia-northeast3/targetPools/lb-test-seoul-03",
		targetPoolIndex := strings.LastIndex(targetPool.SelfLink, "/")
		targetPoolName := targetPool.SelfLink[(targetPoolIndex + 1):]
		newNlbInfo.VMGroup.CspID = targetPoolName

		// instances iid set
		instanceIIDs := []irs.IID{}
		for _, instanceId := range targetPool.Instances {
			targetPoolInstanceIndex := strings.LastIndex(instanceId, "/")
			targetPoolInstanceValue := instanceId[(targetPoolInstanceIndex + 1):]

			//instanceIID := irs.IID{SystemId: instanceId}
			instanceIID := irs.IID{NameId: targetPoolInstanceValue, SystemId: instanceId}
			instanceIIDs = append(instanceIIDs, instanceIID)
			vpcInstanceName = targetPoolInstanceValue
		}
		//네트워크 부하 분산기는 주소, 포트, 프로토콜 유형과 같은 수신 IP 프로토콜 데이터를 기준으로 부하를 시스템에 분산합니다.
		//네트워크 부하 분산기는 패스스루 부하 분산기이므로 백엔드는 원래 클라이언트 요청을 수신합니다.
		//네트워크 부하 분산기는 전송 계층 보안(TLS) 오프로드 또는 프록시를 수행하지 않습니다. 트래픽은 VM으로 직접 라우팅됩니다.
		newNlbInfo.VMGroup.VMs = &instanceIIDs
		fmt.Println("instanceIIDs : ", targetPool.Name)
		fmt.Println("instanceIIDs----- : ", instanceIIDs)
		fmt.Println("vpcInstanceName----- : ", vpcInstanceName)
		fmt.Println("newNlbInfo.VMGroup.VMs----- : ", newNlbInfo.VMGroup.VMs)

		// health checker에 대한 ID는 가지고 있으나 내용은 갖고 있지 않아 정보 조회 필요.
		for _, healthChecker := range targetPool.HealthChecks {
			printToJson(healthChecker)
			targetHealthCheckerIndex := strings.LastIndex(healthChecker, "/")
			targetHealthCheckerValue := healthChecker[(targetHealthCheckerIndex + 1):]

			fmt.Println("GlobalHttpHealthChecks start: ", regionID, " : "+targetHealthCheckerValue)
			//targetHealthCheckerInfo, err := nlbHandler.getRegionHealthChecks(regionID, targetHealthCheckerValue)
			targetHealthCheckerInfo, err := nlbHandler.getHttpHealthChecks(targetHealthCheckerValue) // healthchecker는 전역
			if err != nil {
				fmt.Println("targetHealthCheckerInfo : ", err)
			}
			if targetHealthCheckerInfo != nil {
				printToJson(targetHealthCheckerInfo)

				healthCheckerInfo := irs.HealthCheckerInfo{
					CspID:     targetHealthCheckerInfo.SelfLink, // health checker는 url필요.
					Protocol:  HealthCheck_Http,                 // GlobalHttpHealthChecks 이므로
					Port:      strconv.FormatInt(targetHealthCheckerInfo.Port, 10),
					Interval:  int(targetHealthCheckerInfo.CheckIntervalSec),
					Timeout:   int(targetHealthCheckerInfo.TimeoutSec),
					Threshold: int(targetHealthCheckerInfo.HealthyThreshold),
					//KeyValueList[], KeyValue
				}

				//printToJson(healthCheckerInfo)
				newNlbInfo.HealthChecker = healthCheckerInfo
			}

			fmt.Println("GlobalHttpHealthChecks end: ")
		}

		// vpcIID 조회
		// TODO : VPC정보조회를 위해 INSTANCE 정보 조회 시 같은 region의 다른 zone은 가져오지 못함. GetVM에서 GetVMByZone 같은거 추가해야 할 듯.
		vNetVmInfo, err := vmHandler.GetVM(irs.IID{SystemId: vpcInstanceName})
		if err != nil {
			fmt.Println("fail to get VPC Info : ", err)
		}
		//printToJson(vNetVmInfo)
		newNlbInfo.VpcIID = vNetVmInfo.VpcIID

		nlbMap[targetPool.Name] = newNlbInfo
	}
	//printToJson(targetPoolList)

	fmt.Println("Targetpool end: ")
	printToJson(nlbMap)

	return nlbInfoList, nil
}

// Load balancer 조회
// nlbIID 에서 NameId = lbName, targetPoolName, forwardingRuleName
func (nlbHandler *GCPNLBHandler) GetNLB(nlbIID irs.IID) (irs.NLBInfo, error) {
	var nlbInfo irs.NLBInfo

	projectID := nlbHandler.Credential.ProjectID
	regionID := nlbHandler.Region.Region

	fmt.Println("projectID: ", projectID)
	fmt.Println("regionID: ", regionID)

	// region forwarding rule 는 target pool 과 lb이름으로 엮임.
	// map에 nb이름으로 nbInfo를 넣고 해당 값들 추가해서 조합
	targetPoolName := nlbIID.NameId

	// forwardingRule 조회

	fmt.Println("region forwardingRules start: ", regionID)
	regionForwardingRule, err := nlbHandler.getRegionForwardingRules(regionID, targetPoolName)
	if err != nil {
		fmt.Println("regionForwardingRule  list: ", err)
	}
	if regionForwardingRule != nil { // dial tcp: lookup compute.googleapis.com: no such host 일 때, 	panic: runtime error: invalid memory address or nil pointer dereference

		//targetLbIndex := strings.LastIndex(regionForwardingRule.Target, "/")
		//targetLbName := regionForwardingRule.Target[(targetLbIndex + 1):]
		//
		//targetForwardingRuleIndex := strings.LastIndex(regionForwardingRule.Name, "/")
		//targetForwardingRuleValue := regionForwardingRule.Name[(targetForwardingRuleIndex + 1):]

		// targetlink에서 lb 추출
		//targetNlbInfo := nlbMap[targetLbValue]
		listenerInfo := convertRegionForwardingRuleToNlbListener(regionForwardingRule)

		createdTime, _ := time.Parse(
			time.RFC3339,
			regionForwardingRule.CreationTimestamp) // RFC3339형태이므로 해당 시간으로 다시 생성

		loadBalancerType := regionForwardingRule.LoadBalancingScheme
		if strings.EqualFold(loadBalancerType, "EXTERNAL") {
			loadBalancerType = "PUBLIC"
		}

		nlbInfo = irs.NLBInfo{
			//IId:         irs.IID{NameId: targetPoolName, SystemId: targetForwardingRuleValue}, // NameId = Lb Name, SystemId = forwardingRule name
			IId:         nlbIID,
			VpcIID:      irs.IID{NameId: "", SystemId: ""}, // VpcIID 는 Pool 안의 instance에 있는 값
			Type:        loadBalancerType,                  // PUBLIC/INTERNAL : extenel -> PUBLIC으로 변경하는 로직 적용해야함.
			Scope:       "REGION",
			Listener:    listenerInfo,
			CreatedTime: createdTime, //RFC3339 "creationTimestamp":"2022-05-24T01:20:40.334-07:00"
			//KeyValueList  []KeyValue
		}
		printToJson(regionForwardingRule)
	}

	//fmt.Println("forwardingRules result size  : ", len(regionForwardingRuleList.Items))
	fmt.Println("regionForwardingRule end: ")

	fmt.Println("Targetpool start: ")

	targetPool, err := nlbHandler.getTargetPool(regionID, targetPoolName)
	if err != nil {
		fmt.Println("targetPoolList  list: ", err)
	}

	nlbInfo.VMGroup = nlbHandler.extractVmGroup(targetPool)

	// health checker에 대한 ID는 가지고 있으나 내용은 갖고 있지 않아 정보 조회 필요.
	healthChecker, err := nlbHandler.extractHealthChecker(regionID, targetPool)
	nlbInfo.HealthChecker = healthChecker

	// TODO : vpc는 별도 조회 : instance의 vpc 조회하도록. 문제 instance가 region의 zone c에 있는 경우 vmHandler.GetVM에서 instance 를 못찾는 경우가 있음.

	//if targetPool.Instances != nil {
	//	printToJson(targetPool)
	//
	//	vpcInstanceName := "" // vpc를 갸져올 instance 이름
	//
	//	// vmGroup == targetPool
	//	//"name":"lb-test-seoul-03",
	//	//"selfLink":"https://www.googleapis.com/compute/v1/projects/yhnoh-335705/regions/asia-northeast3/targetPools/lb-test-seoul-03",
	//	//targetPoolIndex := strings.LastIndex(targetPool.SelfLink, "/")
	//	//targetPoolValue := targetPool.SelfLink[(targetPoolIndex + 1):]
	//
	//	// instances iid set
	//	instanceIIDs := []irs.IID{}
	//
	//	for _, instanceId := range targetPool.Instances {
	//		targetPoolInstanceIndex := strings.LastIndex(instanceId, "/")
	//		targetPoolInstanceValue := instanceId[(targetPoolInstanceIndex + 1):]
	//
	//		//instanceIID := irs.IID{SystemId: instanceId}
	//		instanceIID := irs.IID{NameId: targetPoolInstanceValue, SystemId: instanceId}
	//		instanceIIDs = append(instanceIIDs, instanceIID)
	//		vpcInstanceName = targetPoolInstanceValue
	//	}
	//
	//	//네트워크 부하 분산기는 주소, 포트, 프로토콜 유형과 같은 수신 IP 프로토콜 데이터를 기준으로 부하를 시스템에 분산합니다.
	//	//네트워크 부하 분산기는 패스스루 부하 분산기이므로 백엔드는 원래 클라이언트 요청을 수신합니다.
	//	//네트워크 부하 분산기는 전송 계층 보안(TLS) 오프로드 또는 프록시를 수행하지 않습니다. 트래픽은 VM으로 직접 라우팅됩니다.
	//	nlbInfo.VMGroup.CspID = targetPoolName
	//	nlbInfo.VMGroup.Protocol = regionForwardingRule.IPProtocol
	//	nlbInfo.VMGroup.Port = regionForwardingRule.PortRange
	//	nlbInfo.VMGroup.VMs = &instanceIIDs
	//
	//	// health checker에 대한 ID는 가지고 있으나 내용은 갖고 있지 않아 정보 조회 필요.
	//	for _, healthChecker := range targetPool.HealthChecks {
	//		printToJson(healthChecker)
	//		targetHealthCheckerIndex := strings.LastIndex(healthChecker, "/")
	//		targetHealthCheckerValue := healthChecker[(targetHealthCheckerIndex + 1):]
	//
	//		fmt.Println("GlobalHttpHealthChecks start: ", regionID, " : "+targetHealthCheckerValue)
	//		//targetHealthCheckerInfo, err := nlbHandler.getRegionHealthChecks(regionID, targetHealthCheckerValue)
	//		targetHealthCheckerInfo, err := nlbHandler.getHttpHealthChecks(targetHealthCheckerValue) // healthchecker는 전역
	//		if err != nil {
	//			fmt.Println("targetHealthCheckerInfo : ", err)
	//		}
	//		if targetHealthCheckerInfo != nil {
	//			printToJson(targetHealthCheckerInfo)
	//
	//			healthCheckerInfo := irs.HealthCheckerInfo{
	//				CspID:     targetHealthCheckerInfo.SelfLink,
	//				Protocol:  HealthCheck_Http,
	//				Port:      strconv.FormatInt(targetHealthCheckerInfo.Port, 10),
	//				Interval:  int(targetHealthCheckerInfo.CheckIntervalSec),
	//				Timeout:   int(targetHealthCheckerInfo.TimeoutSec),
	//				Threshold: int(targetHealthCheckerInfo.HealthyThreshold),
	//				//KeyValueList[], KeyValue
	//			}
	//
	//			//printToJson(healthCheckerInfo)
	//			nlbInfo.HealthChecker = healthCheckerInfo
	//		}
	//
	//		fmt.Println("GlobalHttpHealthChecks end: ")
	//	}
	//
	//	// vpcIID 조회
	// instace에서 vpc의 IID 조회를 위해
	//vmHandler := GCPVMHandler{
	//	Client:     nlbHandler.Client,
	//	Region:     nlbHandler.Region,
	//	Ctx:        nlbHandler.Ctx,
	//	Credential: nlbHandler.Credential,
	//}
	//	vNetVmInfo, err := vmHandler.GetVM(irs.IID{SystemId: vpcInstanceName})
	//	if err != nil {
	//		fmt.Println("fail to get VPC Info : ", err)
	//	}
	//	//printToJson(vNetVmInfo)
	//	nlbInfo.VpcIID = vNetVmInfo.VpcIID
	//}
	fmt.Println("Targetpool end: ")
	printToJson(nlbInfo)

	return nlbInfo, nil
}

/*
// NLB 삭제. healthchecker는 삭제하지 않음.
// delete 는 forwardingRule -> targetPool순으로 삭제
// targetPool을 먼저 삭제하면 Error 400: The target_pool resource 'xxx' is already being used by 'yyy', resourceInUseByAnotherResource
// 두 개가 transaction으로 묶이지 않기 때문에
// frontend는 삭제되고 targetPool이 어떤이유에서 삭제가 되지 않았을 때,
// 다음 시도에서 삭제할 방법 찾아야 함.( frontend에서 오류 발생시 (404) -> targetpool 삭제 )

	ex) CreateNLB에서 TargetPool 생성직후 forwardingRule생성을 호출하면 "not ready"로 에러 발생 -> 리소스 회수로직이 필요할까?
        이 때, DeleteNLB로는 삭제 불가...
forwardingRule err  :  googleapi: Error 400: The resource 'projects/yhnoh-335705/regions/asia-northeast3/targetPools/lb-tcptest-be-01' is not ready, resourceNotReady

*/
func (nlbHandler *GCPNLBHandler) DeleteNLB(nlbIID irs.IID) (bool, error) {
	//projectID := nlbHandler.Credential.ProjectID
	regionID := nlbHandler.Region.Region

	allDeleted := false

	// frontend

	// forwarding rule을 모두 조회하여 target이 nlbIID.SystemId 인 forwarding rule 모두 조회
	// (사용자가 추가 했을 수도 있으므로)
	// for loop로 돌면서 forwarding rule 이름으로 삭제
	targetPoolName := nlbIID.NameId
	targetPoolUrl := nlbIID.SystemId

	forwardingRuleList, err := nlbHandler.listRegionForwardingRules(regionID, "", targetPoolUrl)
	if err != nil {
		fmt.Println("DeleteNLB forwardingRule  err: ", err)
		return false, err
	}

	deleteCount := 0
	itemLength := len(forwardingRuleList.Items)
	for idx, forwardingRule := range forwardingRuleList.Items {
		err := nlbHandler.deleteRegionForwardingRule(regionID, forwardingRule.Name)
		if err != nil {
			fmt.Println("DeleteNLB forwardingRule  err: ", err)
			return false, err
		}
		fmt.Println("DeleteNLB forwardingRule  delete: index= ", idx, " / ", itemLength)
		deleteCount++
	}
	if deleteCount > 0 && deleteCount == itemLength {
		fmt.Println("DeleteNLB forwardingRule  deleted: ", deleteCount, " / ", itemLength)
	} else {
		// 삭제 중 오류가 있다는 것인데 중간에 error나면 return하고 있어서...
	}

	// backend
	targetPoolDeleteResult, err := nlbHandler.deleteTargetPool(regionID, targetPoolName)
	if err != nil {
		fmt.Println("DeleteNLB targetPool  err: ", err)
		return false, err
	}

	if !targetPoolDeleteResult {
		fmt.Println("DeleteNLB targetPool result : ", targetPoolDeleteResult)
		return false, err
	}

	//if forwardingRuleDeleteResult && targetPoolDeleteResult {
	if targetPoolDeleteResult {
		allDeleted = true
	}

	return allDeleted, nil
}

//------ Frontend Control

/*
	Listener 정보 변경
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

	// forwardingRule => regionForwardingRule
	//type ListenerInfo struct {
	//	Protocol	string	// TCP|UDP
	//	IP		string	// Auto Generated and attached
	//	Port		string	// 1-65535
	//	DNSName		string	// Optional, Auto Generated and attached
	//
	//	CspID		string	// Optional, May be Used by Driver.
	//	KeyValueList []KeyValue
	//}
	// 수정 가능한 항목은 Protocol, IP, Port, DNSName

	regionID := nlbHandler.Region.Region
	targetPoolName := nlbIID.NameId
	//targetPoolUrl := nlbIID.SystemId
	// targetPool url
	targetPool, err := nlbHandler.getTargetPool(regionID, targetPoolName)
	if err != nil {
		fmt.Println("cannot find Backend Service : ", targetPoolName)
		return irs.ListenerInfo{}, errors.New("cannot find Backend Service")
	}

	// 기존 forwardingRule 삭제
	err = nlbHandler.deleteRegionForwardingRule(regionID, targetPoolName)
	if err != nil {
		return irs.ListenerInfo{}, err
	}

	// 새로운 forwarding Rule 등록
	regRegionForwardingRule := convertNlbInfoToForwardingRule(listener, targetPool)
	err = nlbHandler.insertRegionForwardingRules(regionID, &regRegionForwardingRule)
	if err != nil {
		// 등록 실패
		return irs.ListenerInfo{}, err
	}

	forwardingRule, err := nlbHandler.getRegionForwardingRules(regionID, targetPoolName)
	listenerInfo := irs.ListenerInfo{
		Protocol: forwardingRule.IPProtocol,
		IP:       forwardingRule.IPAddress,
		Port:     forwardingRule.PortRange,
		//DNSName:  forwardingRule., // 향후 사용할 때 Adderess에서 가져올 듯
		CspID: forwardingRule.Name, // forwarding rule name 전체
		//KeyValueList:
	}

	return listenerInfo, nil
}

//func (nlbHandler *GCPNLBHandler) ChangeListener(nlbIID irs.IID, listeners *[]irs.ListenerInfo) (irs.NLBInfo, error) {
//	return irs.NLBInfo{}, nil
//}

//------ Backend Control
// VMGroup 정보 수정
// VM의 변경이 없는 경우 VMGroupInfo.VMs 는 빈 값으로 하여 vm수정로직 탈 필요없도록
// VM의 변경이 있는 경우는
func (nlbHandler *GCPNLBHandler) ChangeVMGroupInfo(nlbIID irs.IID, vmGroup irs.VMGroupInfo) (irs.VMGroupInfo, error) {

	//type VMGroupInfo struct {
	//	Protocol     string
	//	Port         string
	//	VMs          *[]IID
	//	CspID        string
	//	KeyValueList []KeyValue
	//}

	// backend 에 연동된 instance가 instanceGroup인지 instancePool인지 확인. 둘 중 한가지만 가능
	//vmGroup.CspID 를 key 롤 instanceGroup/instancePool 조회.
	// 우선은 instancePool로 작업. instanceGroup 에 대한 사용은 고민해 봐야 함.
	// 넘어온 vm정보가 추가 되는지 덮어쓰는지 확인 필요.

	//projectID : csta-349809
	//regionID : asia-northeast3
	//zoneID : asia-northeast3-a
	// poolName : lbtest-seoul-region-01
	// targetPools.list -> items.name == vmGroup.CspID
	//"instances": [
	//"https://www.googleapis.com/compute/v1/projects/csta-349809/zones/asia-northeast3-a/instances/lb-test-instance-seoul01",
	//"https://www.googleapis.com/compute/v1/projects/csta-349809/zones/asia-northeast3-a/instances/lb-test-instance-seoul02"
	//],

	// 1. targetPool 존재하는지 확인 by cspID
	// 2. targetPool에서 vm목록 추출
	// 3. for에서 해당 instance이름으로 instance url 추충
	// 3-1 instance가 존재하지 않으면 error
	// 3-2. 존재하면 targetPool의 vm목록에.IID = instance url 있는지 비교
	// 3-2-1     있으면 continue
	// 3-2-2	 없으면 addInstance
	// 3-2-3     vm목록에 없는 targetPool의 vm은 삭제

	// instances.get(projectID, zoneID, nameofInstance) -> instance url 추출
	// 		"selfLink": "https://www.googleapis.com/compute/v1/projects/csta-349809/zones/asia-northeast3-a/instances/lb-test-instance-seoul02",
	//		"ID"
	//		"Name"

	// addInstance/ removeInstance 시 instance url  사용
	// type VMGroupInfo struct {
	//	Protocol        string	// TCP|UDP|HTTP|HTTPS
	//	Port            string	// 1-65535
	//	VMs		*[]IID
	//
	//	CspID		string	// Optional, May be Used by Driver.
	//	KeyValueList []KeyValue
	//}

	//
	return irs.VMGroupInfo{}, nil
}

// targetPool에 vm 추가 by instanceUrl
func (nlbHandler *GCPNLBHandler) AddVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (irs.VMGroupInfo, error) {
	//
	regionID := nlbHandler.Region.Region
	targetPoolName := nlbIID.NameId

	err := nlbHandler.addTargetPoolInstance(regionID, targetPoolName, vmIIDs)
	if err != nil {
		return irs.VMGroupInfo{}, err
	}

	targetPool, err := nlbHandler.getTargetPool(regionID, targetPoolName)
	if err != nil {
		fmt.Println("targetPoolList  list: ", err)
	}

	vmGroup := nlbHandler.extractVmGroup(targetPool)
	return vmGroup, nil
}

// targetPool에서 vm 삭제 by instanceUrl
func (nlbHandler *GCPNLBHandler) RemoveVMs(nlbIID irs.IID, vmIIDs *[]irs.IID) (bool, error) {
	return false, nil
}

// get HealthCheckerInfo
func (nlbHandler *GCPNLBHandler) GetVMGroupHealthInfo(nlbIID irs.IID) (irs.HealthInfo, error) {
	return irs.HealthInfo{}, nil
}

// HealthCheckerInfo 변경
func (nlbHandler *GCPNLBHandler) ChangeHealthCheckerInfo(nlbIID irs.IID, healthChecker irs.HealthCheckerInfo) (irs.HealthCheckerInfo, error) {

	return irs.HealthCheckerInfo{}, nil
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

// instance template 등록
func (nlbHandler *GCPNLBHandler) insertInstanceTemplate(instanceTemplateReq compute.InstanceTemplate) error {
	//POST https://compute.googleapis.com/compute/v1/projects/PROJECT_ID/global/instanceTemplates
	//{
	//	"name": "INSTANCE_TEMPLATE_NAME",
	//	"sourceInstance": "zones/SOURCE_INSTANCE_ZONE/instances/SOURCE_INSTANCE",
	//	"sourceInstanceParams": {
	//		"diskConfigs": [
	//			{
	//			"deviceName": "SOURCE_DISK",
	//			"instantiateFrom": "INSTANTIATE_OPTIONS",
	//			"autoDelete": false
	//			}
	//		]
	//	}
	//}

	// path param
	projectID := nlbHandler.Credential.ProjectID

	//instanceTemplate := compute.InstanceTemplate{}
	req, err := nlbHandler.Client.InstanceTemplates.Insert(projectID, &instanceTemplateReq).Do()
	if err != nil {

	}

	err = WaitUntilComplete(nlbHandler.Client, projectID, "", req.Name, true)
	if err != nil {

	}

	id := req.Id
	name := req.Name
	fmt.Println("id = ", id, " : name = ", name)
	instanceTemplate, err := nlbHandler.getInstanceTemplate(name)
	if err != nil {
		//return nil, err
	}

	fmt.Println("instanceTemplate ", instanceTemplate)
	return nil
	//if err != nil {
	//	return irs.VPCInfo{}, err
	//}
	//errWait := vVPCHandler.WaitUntilComplete(req.Name, true)

	//compute.NewInstanceTemplatesService()
	//result, err := nlbHandler.Client.
	//fireWall := compute.Firewall{
	//	Name:      firewallName,
	//	Allowed:   firewallAllowed,
	//	Denied:    firewallDenied,
	//	Direction: firewallDirection,
	//	Network:   networkURL,
	//	TargetTags: []string{
	//		securityGroupName,
	//	},
	//}
	//type InstanceTemplatesInsertCall struct {
	//	s                *Service
	//	project          string
	//	instancetemplate *InstanceTemplate
	//	urlParams_       gensupport.URLParams
	//	ctx_             context.Context
	//	header_          http.Header
	//}
}

// instanceTemplate 조회
func (nlbHandler *GCPNLBHandler) getInstanceTemplate(resourceId string) (*compute.InstanceTemplate, error) {
	projectID := nlbHandler.Credential.ProjectID

	instanceTemplateInfo, err := nlbHandler.Client.InstanceTemplates.Get(projectID, resourceId).Do()
	if err != nil {
		return &compute.InstanceTemplate{}, err
	}

	//
	fmt.Println(instanceTemplateInfo)
	return instanceTemplateInfo, nil
}

// instanceTemplate 목록 조회
// InstanceTemplateList 객체를 넘기고 사용은 InstanceTemplateList.Item에서 꺼내서 사용
func (nlbHandler *GCPNLBHandler) listInstanceTemplate(filter string) (*compute.InstanceTemplateList, error) {
	projectID := nlbHandler.Credential.ProjectID

	fmt.Printf(filter)
	//if strings.EqualFold(filter, "") {
	//	req := nlbHandler.Client.InstanceTemplates.List(projectID)
	//	//req.Filter()
	//	if err := req.Pages(nlbHandler.Ctx, func(page *compute.InstanceTemplateList) error {
	//		for _, instanceTemplate := range page.Items {
	//			fmt.Printf("%#v\n", instanceTemplate)
	//		}
	//		return nil
	//	}); err != nil {
	//		//log.Fatal(err)
	//	}
	//}
	result, err := nlbHandler.Client.InstanceTemplates.List(projectID).Do()
	if err != nil {
		return nil, err
	}

	//

	fmt.Println(result)
	fmt.Println(" len ", len(result.Items))
	return result, nil
}

// Region instance group 등록
func (nlbHandler *GCPNLBHandler) insertRegionInstanceGroup(regionID string, reqInstanceGroupManager compute.InstanceGroupManager) (*compute.InstanceGroupManager, error) {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	req, err := nlbHandler.Client.RegionInstanceGroupManagers.Insert(projectID, regionID, &reqInstanceGroupManager).Do()
	if err != nil {

	}

	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.Name, true)
	if err != nil {

	}

	id := req.Id
	name := req.Name
	fmt.Println("id = ", id, " : name = ", name)
	result, err := nlbHandler.getRegionInstanceGroupManager(regionID, name)
	if err != nil {
		//return nil, err
	}

	fmt.Println("RegionInstanceGroup ", result)
	return result, nil
}

// Region InstanceGroup 조회
func (nlbHandler *GCPNLBHandler) getRegionInstanceGroupManager(regionID string, resourceId string) (*compute.InstanceGroupManager, error) {
	projectID := nlbHandler.Credential.ProjectID

	result, err := nlbHandler.Client.RegionInstanceGroupManagers.Get(projectID, regionID, resourceId).Do()
	if err != nil {
		return &compute.InstanceGroupManager{}, err
	}

	//
	fmt.Println(result)
	return result, nil
}

// Region InstanceGroup 목록 조회
// InstanceGroupList 객체를 넘기고 사용은 InstanceGroupList.Item에서 꺼내서 사용
// return 객체가 RegionInstanceGroupManagerList 임. 다른것들은 Region 구분 없는 객체로 return
func (nlbHandler *GCPNLBHandler) listRegionInstanceGroupManager(regionID string, filter string) (*compute.RegionInstanceGroupManagerList, error) {
	projectID := nlbHandler.Credential.ProjectID

	fmt.Printf(filter)
	result, err := nlbHandler.Client.RegionInstanceGroupManagers.List(projectID, regionID).Do()
	if err != nil {
		return nil, err
	}

	//

	fmt.Println(result)
	fmt.Println(" len ", len(result.Items))
	return result, nil
}

// regionInstanceGroups는 이기종 또는 직접관리하는 경우 사용 but, get, list, listInstances, setNamedPoers  만 있음. insert없음
// InstanceGroup 조회
func (nlbHandler *GCPNLBHandler) getRegionInstanceGroup(regionID string, resourceId string) (*compute.InstanceGroup, error) {
	projectID := nlbHandler.Credential.ProjectID

	result, err := nlbHandler.Client.RegionInstanceGroups.Get(projectID, regionID, resourceId).Do()
	if err != nil {
		return &compute.InstanceGroup{}, err
	}

	//
	fmt.Println(result)
	return result, nil
}

// regionInstanceGroups는 이기종 또는 직접관리하는 경우 사용 but, get, list, listInstances, setNamedPoers  만 있음. insert없음
// RegionInstanceGroupList 객체를 넘기고 사용은 RegionInstanceGroupList.Item에서 꺼내서 사용
func (nlbHandler *GCPNLBHandler) listRegionInstanceGroups(regionID string, filter string) (*compute.RegionInstanceGroupList, error) {
	projectID := nlbHandler.Credential.ProjectID

	fmt.Printf(filter)
	result, err := nlbHandler.Client.RegionInstanceGroups.List(projectID, regionID).Do()
	if err != nil {
		return nil, err
	}

	//

	fmt.Println(result)
	fmt.Println(" len ", len(result.Items))
	return result, nil
}

// 호출하는 api가 listInstances 여서 listInstances + RegionInstanceGroups
// RegionInstanceGroupsListInstances 객체를 넘기고 사용은 RegionInstanceGroupsListInstances.Item에서 꺼내서 사용
func (nlbHandler *GCPNLBHandler) listInstancesRegionInstanceGroups(regionID string, regionInstanceGroupName string, reqListInstance compute.RegionInstanceGroupsListInstancesRequest) (*compute.RegionInstanceGroupsListInstances, error) {
	projectID := nlbHandler.Credential.ProjectID

	fmt.Printf(regionInstanceGroupName)
	result, err := nlbHandler.Client.RegionInstanceGroups.ListInstances(projectID, regionID, regionInstanceGroupName, &reqListInstance).Do()
	if err != nil {
		return nil, err
	}

	//

	fmt.Println(result)
	fmt.Println(" len ", len(result.Items))
	return result, nil
}

// global instance group 등록
func (nlbHandler *GCPNLBHandler) insertGlobalInstanceGroup(zoneID string, reqInstanceGroup compute.InstanceGroup) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	//instanceTemplate := compute.InstanceTemplate{}
	req, err := nlbHandler.Client.InstanceGroups.Insert(projectID, zoneID, &reqInstanceGroup).Do()
	if err != nil {

	}

	err = WaitUntilComplete(nlbHandler.Client, projectID, "", req.Name, true)
	if err != nil {

	}

	id := req.Id
	name := req.Name
	fmt.Println("id = ", id, " : name = ", name)
	instanceTemplate, err := nlbHandler.getInstanceTemplate(name)
	if err != nil {
		//return nil, err
	}

	fmt.Println("instanceTemplate ", instanceTemplate)
	return nil
	//if err != nil {
	//	return irs.VPCInfo{}, err
	//}
	//errWait := vVPCHandler.WaitUntilComplete(req.Name, true)

	//compute.NewInstanceTemplatesService()
	//result, err := nlbHandler.Client.
	//fireWall := compute.Firewall{
	//	Name:      firewallName,
	//	Allowed:   firewallAllowed,
	//	Denied:    firewallDenied,
	//	Direction: firewallDirection,
	//	Network:   networkURL,
	//	TargetTags: []string{
	//		securityGroupName,
	//	},
	//}
	//type InstanceTemplatesInsertCall struct {
	//	s                *Service
	//	project          string
	//	instancetemplate *InstanceTemplate
	//	urlParams_       gensupport.URLParams
	//	ctx_             context.Context
	//	header_          http.Header
	//}
}

// global InstanceGroup 조회
func (nlbHandler *GCPNLBHandler) getGlobalInstanceGroup(zoneID string, instanceGroupName string) (*compute.InstanceGroup, error) {
	projectID := nlbHandler.Credential.ProjectID

	instanceGroupInfo, err := nlbHandler.Client.InstanceGroups.Get(projectID, zoneID, instanceGroupName).Do()
	if err != nil {
		return &compute.InstanceGroup{}, err
	}

	//
	fmt.Println(instanceGroupInfo)
	return instanceGroupInfo, nil
}

// global InstanceGroup 목록 조회
// InstanceGroupList 객체를 넘기고 사용은 InstanceGroupList.Item에서 꺼내서 사용
func (nlbHandler *GCPNLBHandler) listGlobalInstanceGroup(zoneID string, filter string) (*compute.InstanceGroupList, error) {
	projectID := nlbHandler.Credential.ProjectID

	fmt.Printf(filter)
	//if strings.EqualFold(filter, "") {
	//	req := nlbHandler.Client.InstanceTemplates.List(projectID)
	//	//req.Filter()
	//	if err := req.Pages(nlbHandler.Ctx, func(page *compute.InstanceTemplateList) error {
	//		for _, instanceTemplate := range page.Items {
	//			fmt.Printf("%#v\n", instanceTemplate)
	//		}
	//		return nil
	//	}); err != nil {
	//		//log.Fatal(err)
	//	}
	//}
	result, err := nlbHandler.Client.InstanceGroups.List(projectID, zoneID).Do()
	if err != nil {
		return nil, err
	}

	//

	fmt.Println(result)
	fmt.Println(" len ", len(result.Items))
	return result, nil
}

// Address 등록 : LB의 시작점
func (nlbHandler *GCPNLBHandler) insertRegionAddresses(regionID string, reqAddress compute.Address) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	//instanceTemplate := compute.InstanceTemplate{}
	req, err := nlbHandler.Client.Addresses.Insert(projectID, regionID, &reqAddress).Do()
	if err != nil {

	}

	err = WaitUntilComplete(nlbHandler.Client, projectID, "", req.Name, true)
	if err != nil {
		return err
	}

	// TODO : 조회로직을 넣어야하나?
	//id := req.Id
	//name := req.Name
	//fmt.Println("id = ", id, " : name = ", name)
	//addressInfo, err := nlbHandler.getAddresses(regionID, name)
	//if err != nil {
	//	return err
	//}
	//fmt.Println("addressInfo ", addressInfo)
	return nil
}

// Address 삭제
func (nlbHandler *GCPNLBHandler) removeRegionAddresses(regionID string, addressName string) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	req, err := nlbHandler.Client.Addresses.Delete(projectID, regionID, addressName).Do()
	if err != nil {

	}

	err = WaitUntilComplete(nlbHandler.Client, projectID, "", req.Name, true)
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
		fmt.Println(item)
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
		fmt.Println(item)
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

	err = WaitUntilComplete(nlbHandler.Client, projectID, "", req.Name, true)
	if err != nil {
		return err
	}

	// TODO : 조회로직을 넣어야하나?
	//id := req.Id
	//name := req.Name
	//fmt.Println("id = ", id, " : name = ", name)
	//addressInfo, err := nlbHandler.getAddresses(regionID, name)
	//if err != nil {
	//	return err
	//}
	//fmt.Println("addressInfo ", addressInfo)
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

	err = WaitUntilComplete(nlbHandler.Client, projectID, "", req.Name, true)
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
		fmt.Println(item)
	}
	return resp, nil
}

// Region ForwardingRule 등록
// 생성한 forwarding rule turn이 필요한가?
//func (nlbHandler *GCPNLBHandler) insertRegionForwardingRules(regionID string, reqRegionForwardingRule *compute.ForwardingRule) (*compute.ForwardingRule, error) {
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

	//id := req.Id
	//name := req.Name
	//fmt.Println("id = ", id, " : name = ", name)
	//forwardingRule, err := nlbHandler.getRegionForwardingRules(regionID, reqRegionForwardingRule.Name)
	//if err != nil {
	//	//	return &compute.ForwardingRule{}, err
	//	//	//return nil, err
	//	return err
	//}
	//
	//fmt.Println("ForwardingRule ", forwardingRule)
	//return forwardingRule, nil
	return nil
}

//deleteRegionForwardingRule
func (nlbHandler *GCPNLBHandler) deleteRegionForwardingRule(regionID string, forwardingRuleName string) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	req, err := nlbHandler.Client.ForwardingRules.Delete(projectID, regionID, forwardingRuleName).Do()
	if err != nil {
		fmt.Println("deleteRegionForwardingRule ", err)
		return err
	}
	fmt.Println("req ", req)
	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.Name, false)
	if err != nil {
		fmt.Println("WaitUntilComplete ", err)
		return err
	}

	return nil
}

// Region ForwardingRule patch
func (nlbHandler *GCPNLBHandler) patchRegionForwardingRules(regionID string, forwardingRuleName string, patchRegionForwardingRule *compute.ForwardingRule) (*compute.ForwardingRule, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID
	//region := nlbHandler.Region.Region

	req, err := nlbHandler.Client.ForwardingRules.Patch(projectID, regionID, forwardingRuleName, patchRegionForwardingRule).Do()
	if err != nil {
		fmt.Println("patchRegionForwardingRules ", err)
		return &compute.ForwardingRule{}, err
	}
	//// 삭제 후 insert
	//delReq, delErr := nlbHandler.Client.ForwardingRules.Delete(projectID, region, forwardingRuleName).Do()
	//if delErr != nil {
	//	return &compute.ForwardingRule{}, delErr
	//}
	//delErr = WaitUntilComplete(nlbHandler.Client, projectID, region, delreq.Name, true)
	//if delErr != nil {
	//	return &compute.ForwardingRule{}, delErr
	//}
	//
	//req, err := nlbHandler.Client.ForwardingRules.Insert(projectID, region, patchRegionForwardingRule).Do()
	//if err != nil {
	//	return &compute.ForwardingRule{}, err
	//}
	printToJson(req)
	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.Name, false)
	if err != nil {
		return &compute.ForwardingRule{}, err
	}

	id := req.Id
	name := req.Name
	fmt.Println("id = ", id, " : name = ", name)
	forwardingRule, err := nlbHandler.getRegionForwardingRules(regionID, patchRegionForwardingRule.Name)
	if err != nil {
		return &compute.ForwardingRule{}, err
		//return nil, err
	}

	fmt.Println("ForwardingRule ", forwardingRule)
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
func (nlbHandler *GCPNLBHandler) listRegionForwardingRules(regionID string, filter string, targetPoolUrl string) (*compute.ForwardingRuleList, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.ForwardingRules.List(projectID, regionID).Do()
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(targetPoolUrl, "") {
		responseForwardingRule := compute.ForwardingRuleList{}
		forwardingRuleList := []*compute.ForwardingRule{}
		for _, item := range resp.Items {
			if strings.EqualFold(item.Target, targetPoolUrl) {
				forwardingRuleList = append(forwardingRuleList, item)
				fmt.Println(item)
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

	}

	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.Name, true)
	if err != nil {
		return &compute.ForwardingRule{}, err
	}

	id := req.Id
	name := req.Name
	fmt.Println("id = ", id, " : name = ", name)
	globalForwardingRule, err := nlbHandler.getGlobalForwardingRules(reqGlobalForwardingRule.Name)
	if err != nil {
		return &compute.ForwardingRule{}, err
		//return nil, err
	}

	fmt.Println("backendService ", globalForwardingRule)
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

	//req := nlbHandler.Client.ForwardingRules.List(projectID, region)
	//if err := req.Pages(nlbHandler.Ctx, func(page *compute.ForwardingRuleList) error {
	//	for _, forwardingRule := range page.Items {
	//		fmt.Printf("%#v\n", forwardingRule)
	//	}
	//	return nil
	//}); err != nil {
	//	return nil, err
	//}

	resp, err := nlbHandler.Client.GlobalForwardingRules.List(projectID).Do()
	if err != nil {
		return nil, err
	}
	for _, item := range resp.Items {
		fmt.Println(item)
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

	err = WaitUntilComplete(nlbHandler.Client, projectID, "", req.Name, true)
	if err != nil {
		return &compute.BackendService{}, err
	}

	id := req.Id
	name := req.Name
	fmt.Println("id = ", id, " : name = ", name)
	backendService, err := nlbHandler.getRegionBackendServices(regionID, reqRegionBackendService.Name)
	if err != nil {
		return &compute.BackendService{}, err
		//return nil, err
	}

	fmt.Println("backendService ", backendService)
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
		fmt.Println(item)
		//if strings.EqualFold(item.Group,   // instance group or network endpoint group(NEG)
	}
	//// item.group // instance group or network endpoint group
	//fmt.Println(backServices)
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

	//fmt.Println(resp)
	printToJson(resp)
	for _, item := range resp.Items {
		fmt.Println(item)
		//if strings.EqualFold(item.Group,   // instance group or network endpoint group(NEG)
	}
	//// item.group // instance group or network endpoint group
	//fmt.Println(backServices)
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

	err = WaitUntilComplete(nlbHandler.Client, projectID, "", req.Name, true)
	if err != nil {
		return &compute.BackendService{}, err
	}

	id := req.Id
	name := req.Name
	fmt.Println("id = ", id, " : name = ", name)
	backendService, err := nlbHandler.getGlobalBackendServices(reqBackendService.Name)
	if err != nil {
		return &compute.BackendService{}, err
		//return nil, err
	}

	fmt.Println("backendService ", backendService)
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
		fmt.Println(item)
		//if strings.EqualFold(item.Group,   // instance group or network endpoint group(NEG)
	}
	//// item.group // instance group or network endpoint group
	//fmt.Println(backServices)
	//return backServices, nil
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
		fmt.Println(item)
		//if strings.EqualFold(item.Group,   // instance group or network endpoint group(NEG)
	}
	//// item.group // instance group or network endpoint group
	//fmt.Println(backServices)
	//return backServices, nil
	return resp, nil
}

func (nlbHandler *GCPNLBHandler) insertRegionHealthChecks(region string, healthCheckerInfo irs.HealthCheckerInfo) {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	// queryParam
	// 4개 중 1개 : TCP, SSL, HTTP, HTTP2
	tcpHealthCheck := &compute.TCPHealthCheck{
		Port:              80, // default value:80, 1~65535
		PortName:          "", // InstanceGroup#NamedPort#name
		PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		Request:           "", // default value is empty
		Response:          "", // default value is empty
		ProxyHeader:       "", // NONE, PROXY_V1
	}
	sslHealthCheck := &compute.SSLHealthCheck{
		Port:              443, // default value is 443n 1~65535
		PortName:          "",  // InstanceGroup#NamedPort#name
		PortSpecification: "",  //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		Request:           "",  // default value is empty
		Response:          "",  // default value is empty
		ProxyHeader:       "",  // NONE, PROXY_V1
	}
	httpHealthCheck := &compute.HTTPHealthCheck{
		Port:              80, // default value:80, 1~65535
		PortName:          "", // InstanceGroup#NamedPort#name
		PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		Host:              "", // default value is empty. IP
		RequestPath:       "", // default value is "/"
		Response:          "", // default value is empty
		ProxyHeader:       "", // NONE, PROXY_V1
	}

	http2HealthCheck := &compute.HTTP2HealthCheck{
		Port:              443, // default value is 443n 1~65535
		PortName:          "",  // InstanceGroup#NamedPort#name
		PortSpecification: "",  //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		Host:              "",  // default value is empty
		RequestPath:       "",  // default value is "/"
		Response:          "",  // default value is empty
		ProxyHeader:       "",  // default value is NONE,  NONE, PROXY_V1
	}

	//healthCheckPort :=	&
	regionHealthCheck := &compute.HealthCheck{
		//////
		//CheckIntervalSec int64 `json:"checkIntervalSec,omitempty"`
		//CreationTimestamp string `json:"creationTimestamp,omitempty"`
		//Description string `json:"description,omitempty"`
		//HealthyThreshold int64 `json:"healthyThreshold,omitempty"`
		//Http2HealthCheck *HTTP2HealthCheck `json:"http2HealthCheck,omitempty"`
		//HttpHealthCheck *HTTPHealthCheck `json:"httpHealthCheck,omitempty"`
		//HttpsHealthCheck *HTTPSHealthCheck `json:"httpsHealthCheck,omitempty"`
		//Id uint64 `json:"id,omitempty,string"`
		//Kind string `json:"kind,omitempty"`
		//Name string `json:"name,omitempty"`
		//Region string `json:"region,omitempty"`
		//SelfLink string `json:"selfLink,omitempty"`
		//SslHealthCheck *SSLHealthCheck `json:"sslHealthCheck,omitempty"`
		//TcpHealthCheck *TCPHealthCheck `json:"tcpHealthCheck,omitempty"`
		//TimeoutSec int64 `json:"timeoutSec,omitempty"`
		//Type string `json:"type,omitempty"`
		//UnhealthyThreshold int64 `json:"unhealthyThreshold,omitempty"`
		//googleapi.ServerResponse `json:"-"`
		//ForceSendFields []string `json:"-"`
		//NullFields []string `json:"-"`
		///////
		Kind: "", // type of resource
		//Id: //output only
		//CreationTimestamp: //  output only
		Name:               "", //
		Description:        "",
		CheckIntervalSec:   10,
		TimeoutSec:         10,
		UnhealthyThreshold: 3,  // default value:2
		HealthyThreshold:   3,  // default value:2
		Type:               "", // enum : TCP, SSL,HTTP, HTTPS, HTTP2

		TcpHealthCheck:   tcpHealthCheck,
		SslHealthCheck:   sslHealthCheck,
		HttpHealthCheck:  httpHealthCheck,
		Http2HealthCheck: http2HealthCheck,

		//HttpHealthCheck: { //&compute.HTTPHealthCheck
		//	Port:              80,
		//	PortName:          "", // 해당 객체에는 Name으로 정의되어 있음.
		//	PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		//	RequestPath:       "",
		//	Response:          "",
		//	ProxyHeader:       "", // NONE, PROXY_V1
		//},
		//HttpsHealthCheck: { //&compute.HttpsHealthCheck
		//	Port:              80,
		//	PortName:          "",
		//	PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		//	RequestPath:       "",
		//	Response:          "",
		//	ProxyHeader:       "", // NONE, PROXY_V1
		//},
		//Http2HealthCheck: { //&compute.HTTP2HealthCheck
		//	Port:              80,
		//	PortName:          "",
		//	PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		//	Host:              "",
		//	RequestPath:       "",
		//	Response:          "",
		//	ProxyHeader:       "", // NONE, PROXY_V1
		//},
		//GrpcHealthCheck: { //&compute.   // grpcHealthcheck 없음
		//	Port:              80,
		//	PortName:          "",
		//	PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		//	GrpcServiceName:   "",
		//},
		//SelfLink: //output only
		//Region: //output only

		//LogConfig: {
		//	Enable: false,
		//},
	}

	// requestBody
	nlbHandler.Client.RegionHealthChecks.Insert(projectID, region, regionHealthCheck)
	//{
	//	"kind": string,
	//	"id": string,
	//	"creationTimestamp": string,
	//	"name": string,
	//	"description": string,
	//	"checkIntervalSec": integer,
	//	"timeoutSec": integer,
	//	"unhealthyThreshold": integer,
	//	"healthyThreshold": integer,
	//	"type": enum,
	//	"tcpHealthCheck": {
	//	"port": integer,
	//		"portName": string,
	//		"portSpecification": enum,
	//		"request": string,
	//		"response": string,
	//		"proxyHeader": enum
	//},
	//	"sslHealthCheck": {
	//	"port": integer,
	//		"portName": string,
	//		"portSpecification": enum,
	//		"request": string,
	//		"response": string,
	//		"proxyHeader": enum
	//},
	//	"httpHealthCheck": {
	//	"port": integer,
	//		"portName": string,
	//		"portSpecification": enum,
	//		"host": string,
	//		"requestPath": string,
	//		"proxyHeader": enum,
	//		"response": string
	//},
	//	"httpsHealthCheck": {
	//	"port": integer,
	//		"portName": string,
	//		"portSpecification": enum,
	//		"host": string,
	//		"requestPath": string,
	//		"proxyHeader": enum,
	//		"response": string
	//},
	//	"http2HealthCheck": {
	//	"port": integer,
	//		"portName": string,
	//		"portSpecification": enum,
	//		"host": string,
	//		"requestPath": string,
	//		"proxyHeader": enum,
	//		"response": string
	//},
	//	"grpcHealthCheck": {
	//	"port": integer,
	//		"portName": string,
	//		"portSpecification": enum,
	//		"grpcServiceName": string
	//},
	//	"selfLink": string,
	//	"region": string,
	//	"logConfig": {
	//	"enable": boolean
	//}
	//}
	// 다른 handler data 호출 시
	//vNetworkHandler := GCPVPCHandler{
	//	Client:     securityHandler.Client,
	//	Region:     securityHandler.Region,
	//	Ctx:        securityHandler.Ctx,
	//	Credential: securityHandler.Credential,
	//}
	//vNetInfo, errVnet := vNetworkHandler.GetVPC(securityReqInfo.VpcIID)
	//spew.Dump(vNetInfo)
	//if errVnet != nil {
	//	cblogger.Error(errVnet)
	//	return irs.SecurityInfo{}, errVnet
	//}
}

// Inteal Http(S) load balancer : region health check => compute.v1.regionHealthCheck
// Traffic Director : global health check => compute.v1.HealthCheck
func (nlbHandler *GCPNLBHandler) getRegionHealthChecks(region string, regionHealthCheckName string) (*compute.HealthCheck, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.RegionHealthChecks.Get(projectID, region, regionHealthCheckName).Do()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Global BackendService 목록 조회
// HealthCheckList 객체를 넘기고 사용은 HealthCheckList.Item에서 꺼내서 사용
func (nlbHandler *GCPNLBHandler) listRegionHealthChecks(region string, filter string) (*compute.HealthCheckList, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.RegionHealthChecks.List(projectID, region).Do()
	if err != nil {
		return nil, err
	}

	for _, item := range resp.Items {
		fmt.Println(item)
	}

	return resp, nil

}

// GlobalHealthChecker Insert
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
		HealthyThreshold: healthyThreshold,

		//RequestPath, string `json:"requestPath,omitempty"`
	}
	printToJson(httpHealthCheck)
	req, err := nlbHandler.Client.HttpHealthChecks.Insert(projectID, httpHealthCheck).Do()
	if err != nil {
		cblogger.Error(err)
		return &compute.HttpHealthCheck{}, err
	}

	err = WaitUntilComplete(nlbHandler.Client, projectID, "", req.Name, true)
	if err != nil {
		//	return &compute.ForwardingRule{}, err
		return &compute.HttpHealthCheck{}, err
	}

	id := req.Id
	name := req.Name
	fmt.Println("id = ", id, " : name = ", name, " cspId = ", healthCheckerInfo.CspID)
	healthCheck, err := nlbHandler.getHttpHealthChecks(healthCheckerInfo.CspID)
	if err != nil {
		return healthCheck, err
		//return nil, err
	}

	fmt.Println("healthCheck ", healthCheck)
	return nil, nil
}

func (nlbHandler *GCPNLBHandler) getHttpHealthChecks(healthCheckName string) (*compute.HttpHealthCheck, error) {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.HttpHealthChecks.Get(projectID, healthCheckName).Do()

	if err != nil {
		return nil, err
	}
	return resp, nil
}

// HttpHealthCheckList 객체를 넘기고 사용은 HealthCheckList.Item에서 꺼내서 사용
func (nlbHandler *GCPNLBHandler) listHttpHealthChecks(filter string) (*compute.HttpHealthCheckList, error) {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.HttpHealthChecks.List(projectID).Do()
	if err != nil {
		return nil, err
	}

	for _, item := range resp.Items {
		fmt.Println(item)
	}

	return resp, nil
}

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
	fmt.Println("id = ", id, " : name = ", name)
	result, err := nlbHandler.getTargetPool(regionID, reqTargetPool.Name)
	if err != nil {
		return &compute.TargetPool{}, err
	}

	fmt.Println("insertTargetPool return targetpool ", result)
	return result, nil
}

// Target pools are used for network TCP/UDP load balancing. A target pool references member instances, an associated legacy HttpHealthCheck resource, and, optionally, a backup target poo
func (nlbHandler *GCPNLBHandler) getTargetPool(regionID string, targetPoolName string) (*compute.TargetPool, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.TargetPools.Get(projectID, regionID, targetPoolName).Do()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Target pools are used for network TCP/UDP load balancing. A target pool references member instances, an associated legacy HttpHealthCheck resource, and, optionally, a backup target poo
func (nlbHandler *GCPNLBHandler) listTargetPools(regionID string, filter string) (*compute.TargetPoolList, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.TargetPools.List(projectID, regionID).Do()
	if err != nil {
		return &compute.TargetPoolList{}, err
	}

	for _, item := range resp.Items {
		fmt.Println(item)
	}

	return resp, nil

}

// nlbHandler.Client.TargetPools.AggregatedList(projectID) : 해당 project의 모든 region 에 대해 region별  target pool 목록

// instanceReference 는 instarce의 url을 인자로 갖는다.
// targetPools.get(targetPoolName)  을 통해 instalces[]을 알 수 있음. 배열에서 하나씩 꺼내서 instanceReference에 넣고 사용.
// Target pools are used for network TCP/UDP load balancing. A target pool references member instances, an associated legacy HttpHealthCheck resource, and, optionally, a backup target poo
func (nlbHandler *GCPNLBHandler) getTargetPoolHealth(regionID string, targetPoolName string, instanceReference *compute.InstanceReference) (*compute.TargetPoolInstanceHealth, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID
	//https://www.googleapis.com/compute/v1/projects/csta-349809/zones/asia-northeast3-a/instances/lb-test-instance-seoul01
	resp, err := nlbHandler.Client.TargetPools.GetHealth(projectID, regionID, targetPoolName, instanceReference).Do()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Target Pool에 health check 추가
// health check는 instance url이 있어야 하므로 갖고 있는 곳에서 목록조회
// add는 성공여부만
// Target pools are used for network TCP/UDP load balancing. A target pool references member instances, an associated legacy HttpHealthCheck resource, and, optionally, a backup target poo
func (nlbHandler *GCPNLBHandler) addTargetPoolHealthCheck(regionID string, targetPoolName string, reqHealthCheck compute.TargetPoolsAddHealthCheckRequest) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	// queryParam

	// requestBody
	req, err := nlbHandler.Client.TargetPools.AddHealthCheck(projectID, regionID, targetPoolName, &reqHealthCheck).Do()
	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.Name, true)
	if err != nil {
		return err
	}

	id := req.Id
	name := req.Name
	fmt.Println("id = ", id, " : name = ", name)

	for _, item := range reqHealthCheck.HealthChecks {
		fmt.Println("item = ", item)
	}
	fmt.Println("addTargetPoolHealthCheck ")
	return nil
}

func (nlbHandler *GCPNLBHandler) removeHealthCheck(healthCheckerInfo irs.HealthCheckerInfo) {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	// queryParam
	// 4개 중 1개 : TCP, SSL, HTTP, HTTP2
	tcpHealthCheck := &compute.TCPHealthCheck{
		Port:              80, // default value:80, 1~65535
		PortName:          "", // InstanceGroup#NamedPort#name
		PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		Request:           "", // default value is empty
		Response:          "", // default value is empty
		ProxyHeader:       "", // NONE, PROXY_V1
	}
	sslHealthCheck := &compute.SSLHealthCheck{
		Port:              443, // default value is 443n 1~65535
		PortName:          "",  // InstanceGroup#NamedPort#name
		PortSpecification: "",  //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		Request:           "",  // default value is empty
		Response:          "",  // default value is empty
		ProxyHeader:       "",  // NONE, PROXY_V1
	}
	httpHealthCheck := &compute.HTTPHealthCheck{
		Port:              80, // default value:80, 1~65535
		PortName:          "", // InstanceGroup#NamedPort#name
		PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		Host:              "", // default value is empty. IP
		RequestPath:       "", // default value is "/"
		Response:          "", // default value is empty
		ProxyHeader:       "", // NONE, PROXY_V1
	}

	http2HealthCheck := &compute.HTTP2HealthCheck{
		Port:              443, // default value is 443n 1~65535
		PortName:          "",  // InstanceGroup#NamedPort#name
		PortSpecification: "",  //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		Host:              "",  // default value is empty
		RequestPath:       "",  // default value is "/"
		Response:          "",  // default value is empty
		ProxyHeader:       "",  // default value is NONE,  NONE, PROXY_V1
	}

	//healthCheckPort :=	&
	rb := &compute.HealthCheck{
		//////
		//CheckIntervalSec int64 `json:"checkIntervalSec,omitempty"`
		//CreationTimestamp string `json:"creationTimestamp,omitempty"`
		//Description string `json:"description,omitempty"`
		//HealthyThreshold int64 `json:"healthyThreshold,omitempty"`
		//Http2HealthCheck *HTTP2HealthCheck `json:"http2HealthCheck,omitempty"`
		//HttpHealthCheck *HTTPHealthCheck `json:"httpHealthCheck,omitempty"`
		//HttpsHealthCheck *HTTPSHealthCheck `json:"httpsHealthCheck,omitempty"`
		//Id uint64 `json:"id,omitempty,string"`
		//Kind string `json:"kind,omitempty"`
		//Name string `json:"name,omitempty"`
		//Region string `json:"region,omitempty"`
		//SelfLink string `json:"selfLink,omitempty"`
		//SslHealthCheck *SSLHealthCheck `json:"sslHealthCheck,omitempty"`
		//TcpHealthCheck *TCPHealthCheck `json:"tcpHealthCheck,omitempty"`
		//TimeoutSec int64 `json:"timeoutSec,omitempty"`
		//Type string `json:"type,omitempty"`
		//UnhealthyThreshold int64 `json:"unhealthyThreshold,omitempty"`
		//googleapi.ServerResponse `json:"-"`
		//ForceSendFields []string `json:"-"`
		//NullFields []string `json:"-"`
		///////
		Kind: "", // type of resource
		//Id: //output only
		//CreationTimestamp: //  output only
		Name:               "", //
		Description:        "",
		CheckIntervalSec:   10,
		TimeoutSec:         10,
		UnhealthyThreshold: 3,  // default value:2
		HealthyThreshold:   3,  // default value:2
		Type:               "", // enum : TCP, SSL,HTTP, HTTPS, HTTP2

		TcpHealthCheck:   tcpHealthCheck,
		SslHealthCheck:   sslHealthCheck,
		HttpHealthCheck:  httpHealthCheck,
		Http2HealthCheck: http2HealthCheck,

		//HttpHealthCheck: { //&compute.HTTPHealthCheck
		//	Port:              80,
		//	PortName:          "", // 해당 객체에는 Name으로 정의되어 있음.
		//	PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		//	RequestPath:       "",
		//	Response:          "",
		//	ProxyHeader:       "", // NONE, PROXY_V1
		//},
		//HttpsHealthCheck: { //&compute.HttpsHealthCheck
		//	Port:              80,
		//	PortName:          "",
		//	PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		//	RequestPath:       "",
		//	Response:          "",
		//	ProxyHeader:       "", // NONE, PROXY_V1
		//},
		//Http2HealthCheck: { //&compute.HTTP2HealthCheck
		//	Port:              80,
		//	PortName:          "",
		//	PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		//	Host:              "",
		//	RequestPath:       "",
		//	Response:          "",
		//	ProxyHeader:       "", // NONE, PROXY_V1
		//},
		//GrpcHealthCheck: { //&compute.   // grpcHealthcheck 없음
		//	Port:              80,
		//	PortName:          "",
		//	PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		//	GrpcServiceName:   "",
		//},
		//SelfLink: //output only
		//Region: //output only

		//LogConfig: {
		//	Enable: false,
		//},
	}

	// requestBody
	nlbHandler.Client.HealthChecks.Insert(projectID, rb)
	//{
	//	"kind": string,
	//	"id": string,
	//	"creationTimestamp": string,
	//	"name": string,
	//	"description": string,
	//	"checkIntervalSec": integer,
	//	"timeoutSec": integer,
	//	"unhealthyThreshold": integer,
	//	"healthyThreshold": integer,
	//	"type": enum,
	//	"tcpHealthCheck": {
	//	"port": integer,
	//		"portName": string,
	//		"portSpecification": enum,
	//		"request": string,
	//		"response": string,
	//		"proxyHeader": enum
	//},
	//	"sslHealthCheck": {
	//	"port": integer,
	//		"portName": string,
	//		"portSpecification": enum,
	//		"request": string,
	//		"response": string,
	//		"proxyHeader": enum
	//},
	//	"httpHealthCheck": {
	//	"port": integer,
	//		"portName": string,
	//		"portSpecification": enum,
	//		"host": string,
	//		"requestPath": string,
	//		"proxyHeader": enum,
	//		"response": string
	//},
	//	"httpsHealthCheck": {
	//	"port": integer,
	//		"portName": string,
	//		"portSpecification": enum,
	//		"host": string,
	//		"requestPath": string,
	//		"proxyHeader": enum,
	//		"response": string
	//},
	//	"http2HealthCheck": {
	//	"port": integer,
	//		"portName": string,
	//		"portSpecification": enum,
	//		"host": string,
	//		"requestPath": string,
	//		"proxyHeader": enum,
	//		"response": string
	//},
	//	"grpcHealthCheck": {
	//	"port": integer,
	//		"portName": string,
	//		"portSpecification": enum,
	//		"grpcServiceName": string
	//},
	//	"selfLink": string,
	//	"region": string,
	//	"logConfig": {
	//	"enable": boolean
	//}
	//}
	// 다른 handler data 호출 시
	//vNetworkHandler := GCPVPCHandler{
	//	Client:     securityHandler.Client,
	//	Region:     securityHandler.Region,
	//	Ctx:        securityHandler.Ctx,
	//	Credential: securityHandler.Credential,
	//}
	//vNetInfo, errVnet := vNetworkHandler.GetVPC(securityReqInfo.VpcIID)
	//spew.Dump(vNetInfo)
	//if errVnet != nil {
	//	cblogger.Error(errVnet)
	//	return irs.SecurityInfo{}, errVnet
	//}
}

/*
// TargetPool 에 Instance 추가
	넘겨받은 instanceIID 들로 저장되기 때문에
	AddVMs/RemoveVms 가 같은 로직으로 사용 함.
// Target pools are used for network TCP/UDP load balancing. A target pool references member instances, an associated legacy HttpHealthCheck resource, and, optionally, a backup target poo

*/
func (nlbHandler *GCPNLBHandler) addTargetPoolInstance(regionID string, targetPoolName string, instanceIIDs *[]irs.IID) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	if instanceIIDs != nil {
		// queryParam
		instanceRequest := compute.TargetPoolsAddInstanceRequest{}
		instanceReferenceList := []*compute.InstanceReference{}
		for _, instance := range *instanceIIDs {
			instanceReference := &compute.InstanceReference{Instance: instance.SystemId}
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
		fmt.Println("Done")
		return nil
	}
	return errors.New("instanceIIDs are empty.)")
}

/*
	GCP는 삭제없이 넘져지는 iid가 최종형태이므로
	전체 instance목록에서 넘겨받은 iid를 제외하고 addTargetPoolInstance 호출
	instanceIID는 url형태임.
*/
func (nlbHandler *GCPNLBHandler) removeTargetPoolInstance(regionID string, targetPoolName string, deleteInstanceIIDs *[]irs.IID) error {
	// path param

	targetPool, err := nlbHandler.getTargetPool(regionID, targetPoolName)
	if err != nil {
		fmt.Println("cannot find Backend Service : ", targetPoolName)
		return err
	}

	// 전체목록에서 deleteIID 들 뺌
	instanceIIDs := targetPool.Instances
	for _, deleteInstance := range *deleteInstanceIIDs {
		for idx, instance := range instanceIIDs {
			if strings.EqualFold(deleteInstance.SystemId, instance) {
				instanceIIDs = removeElementByIndex(instanceIIDs, idx)
			}
		}
	}

	// addTargetPoolInstance를 호출하기 위해 변환
	targetInstanceIIDs := []irs.IID{}
	for _, targetInstance := range instanceIIDs {
		targetInstanceIID := irs.IID{NameId: "", SystemId: targetInstance}
		targetInstanceIIDs = append(targetInstanceIIDs, targetInstanceIID)
	}

	err = nlbHandler.addTargetPoolInstance(regionID, targetPoolName, &targetInstanceIIDs)
	if err != nil {
		return err
	}
	return nil
	//addTargetPoolInstance(regionID string, targetPoolName string, instanceIIDs *[]irs.IID) error
	//if instanceIIDs != nil {
	//	// queryParam
	//	instanceRequest := compute.TargetPoolsAddInstanceRequest{}
	//	instanceReferenceList := []*compute.InstanceReference{}
	//	for _, instance := range instanceIIDs {
	//		instanceReference := &compute.InstanceReference{Instance: instance}
	//		instanceReferenceList = append(instanceReferenceList, instanceReference)
	//	}
	//	instanceRequest.Instances = instanceReferenceList
	//
	//	// requestBody
	//	res, err := nlbHandler.Client.TargetPools.AddInstance(projectID, regionID, targetPoolName, &instanceRequest).Do()
	//	if err != nil {
	//		return err
	//	}
	//
	//	printToJson(res)
	//	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, res.Name, false)
	//	if err != nil {
	//		return err
	//	}
	//	fmt.Println("Done")
	//	return nil
	//}
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
*/
// Target pools are used for network TCP/UDP load balancing. A target pool references member instances, an associated legacy HttpHealthCheck resource, and, optionally, a backup target poo
func (nlbHandler *GCPNLBHandler) removeTargetPool(regionID string, targetPoolName string) error {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	// requestBody
	res, err := nlbHandler.Client.TargetPools.Delete(projectID, regionID, targetPoolName).Do()
	if err != nil {
		return err
	}

	printToJson(res)
	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, res.Name, false)
	if err != nil {
		return err
	}
	fmt.Println("Done")

	return nil
}

// Target pools are used for network TCP/UDP load balancing. A target pool references member instances, an associated legacy HttpHealthCheck resource, and, optionally, a backup target poo
func (nlbHandler *GCPNLBHandler) aggregatedTargetPoolsList(healthCheckName string) (*compute.HealthCheck, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.HealthChecks.Get(projectID, healthCheckName).Do()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Target pools are used for network TCP/UDP load balancing. A target pool references member instances, an associated legacy HttpHealthCheck resource, and, optionally, a backup target poo
func (nlbHandler *GCPNLBHandler) setTargetpoolBackup(healthCheckerInfo irs.HealthCheckerInfo) {
	// path param
	projectID := nlbHandler.Credential.ProjectID

	// queryParam
	// 4개 중 1개 : TCP, SSL, HTTP, HTTP2
	tcpHealthCheck := &compute.TCPHealthCheck{
		Port:              80, // default value:80, 1~65535
		PortName:          "", // InstanceGroup#NamedPort#name
		PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		Request:           "", // default value is empty
		Response:          "", // default value is empty
		ProxyHeader:       "", // NONE, PROXY_V1
	}
	sslHealthCheck := &compute.SSLHealthCheck{
		Port:              443, // default value is 443n 1~65535
		PortName:          "",  // InstanceGroup#NamedPort#name
		PortSpecification: "",  //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		Request:           "",  // default value is empty
		Response:          "",  // default value is empty
		ProxyHeader:       "",  // NONE, PROXY_V1
	}
	httpHealthCheck := &compute.HTTPHealthCheck{
		Port:              80, // default value:80, 1~65535
		PortName:          "", // InstanceGroup#NamedPort#name
		PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		Host:              "", // default value is empty. IP
		RequestPath:       "", // default value is "/"
		Response:          "", // default value is empty
		ProxyHeader:       "", // NONE, PROXY_V1
	}

	http2HealthCheck := &compute.HTTP2HealthCheck{
		Port:              443, // default value is 443n 1~65535
		PortName:          "",  // InstanceGroup#NamedPort#name
		PortSpecification: "",  //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		Host:              "",  // default value is empty
		RequestPath:       "",  // default value is "/"
		Response:          "",  // default value is empty
		ProxyHeader:       "",  // default value is NONE,  NONE, PROXY_V1
	}

	//healthCheckPort :=	&
	rb := &compute.HealthCheck{
		//////
		//CheckIntervalSec int64 `json:"checkIntervalSec,omitempty"`
		//CreationTimestamp string `json:"creationTimestamp,omitempty"`
		//Description string `json:"description,omitempty"`
		//HealthyThreshold int64 `json:"healthyThreshold,omitempty"`
		//Http2HealthCheck *HTTP2HealthCheck `json:"http2HealthCheck,omitempty"`
		//HttpHealthCheck *HTTPHealthCheck `json:"httpHealthCheck,omitempty"`
		//HttpsHealthCheck *HTTPSHealthCheck `json:"httpsHealthCheck,omitempty"`
		//Id uint64 `json:"id,omitempty,string"`
		//Kind string `json:"kind,omitempty"`
		//Name string `json:"name,omitempty"`
		//Region string `json:"region,omitempty"`
		//SelfLink string `json:"selfLink,omitempty"`
		//SslHealthCheck *SSLHealthCheck `json:"sslHealthCheck,omitempty"`
		//TcpHealthCheck *TCPHealthCheck `json:"tcpHealthCheck,omitempty"`
		//TimeoutSec int64 `json:"timeoutSec,omitempty"`
		//Type string `json:"type,omitempty"`
		//UnhealthyThreshold int64 `json:"unhealthyThreshold,omitempty"`
		//googleapi.ServerResponse `json:"-"`
		//ForceSendFields []string `json:"-"`
		//NullFields []string `json:"-"`
		///////
		Kind: "", // type of resource
		//Id: //output only
		//CreationTimestamp: //  output only
		Name:               "", //
		Description:        "",
		CheckIntervalSec:   10,
		TimeoutSec:         10,
		UnhealthyThreshold: 3,  // default value:2
		HealthyThreshold:   3,  // default value:2
		Type:               "", // enum : TCP, SSL,HTTP, HTTPS, HTTP2

		TcpHealthCheck:   tcpHealthCheck,
		SslHealthCheck:   sslHealthCheck,
		HttpHealthCheck:  httpHealthCheck,
		Http2HealthCheck: http2HealthCheck,

		//HttpHealthCheck: { //&compute.HTTPHealthCheck
		//	Port:              80,
		//	PortName:          "", // 해당 객체에는 Name으로 정의되어 있음.
		//	PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		//	RequestPath:       "",
		//	Response:          "",
		//	ProxyHeader:       "", // NONE, PROXY_V1
		//},
		//HttpsHealthCheck: { //&compute.HttpsHealthCheck
		//	Port:              80,
		//	PortName:          "",
		//	PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		//	RequestPath:       "",
		//	Response:          "",
		//	ProxyHeader:       "", // NONE, PROXY_V1
		//},
		//Http2HealthCheck: { //&compute.HTTP2HealthCheck
		//	Port:              80,
		//	PortName:          "",
		//	PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		//	Host:              "",
		//	RequestPath:       "",
		//	Response:          "",
		//	ProxyHeader:       "", // NONE, PROXY_V1
		//},
		//GrpcHealthCheck: { //&compute.   // grpcHealthcheck 없음
		//	Port:              80,
		//	PortName:          "",
		//	PortSpecification: "", //USE_FIXED_PORT, USE_NAMED_PORT, USE_SERVING_PORT
		//	GrpcServiceName:   "",
		//},
		//SelfLink: //output only
		//Region: //output only

		//LogConfig: {
		//	Enable: false,
		//},
	}

	// requestBody
	nlbHandler.Client.HealthChecks.Insert(projectID, rb)
	//{
	//	"kind": string,
	//	"id": string,
	//	"creationTimestamp": string,
	//	"name": string,
	//	"description": string,
	//	"checkIntervalSec": integer,
	//	"timeoutSec": integer,
	//	"unhealthyThreshold": integer,
	//	"healthyThreshold": integer,
	//	"type": enum,
	//	"tcpHealthCheck": {
	//	"port": integer,
	//		"portName": string,
	//		"portSpecification": enum,
	//		"request": string,
	//		"response": string,
	//		"proxyHeader": enum
	//},
	//	"sslHealthCheck": {
	//	"port": integer,
	//		"portName": string,
	//		"portSpecification": enum,
	//		"request": string,
	//		"response": string,
	//		"proxyHeader": enum
	//},
	//	"httpHealthCheck": {
	//	"port": integer,
	//		"portName": string,
	//		"portSpecification": enum,
	//		"host": string,
	//		"requestPath": string,
	//		"proxyHeader": enum,
	//		"response": string
	//},
	//	"httpsHealthCheck": {
	//	"port": integer,
	//		"portName": string,
	//		"portSpecification": enum,
	//		"host": string,
	//		"requestPath": string,
	//		"proxyHeader": enum,
	//		"response": string
	//},
	//	"http2HealthCheck": {
	//	"port": integer,
	//		"portName": string,
	//		"portSpecification": enum,
	//		"host": string,
	//		"requestPath": string,
	//		"proxyHeader": enum,
	//		"response": string
	//},
	//	"grpcHealthCheck": {
	//	"port": integer,
	//		"portName": string,
	//		"portSpecification": enum,
	//		"grpcServiceName": string
	//},
	//	"selfLink": string,
	//	"region": string,
	//	"logConfig": {
	//	"enable": boolean
	//}
	//}
	// 다른 handler data 호출 시
	//vNetworkHandler := GCPVPCHandler{
	//	Client:     securityHandler.Client,
	//	Region:     securityHandler.Region,
	//	Ctx:        securityHandler.Ctx,
	//	Credential: securityHandler.Credential,
	//}
	//vNetInfo, errVnet := vNetworkHandler.GetVPC(securityReqInfo.VpcIID)
	//spew.Dump(vNetInfo)
	//if errVnet != nil {
	//	cblogger.Error(errVnet)
	//	return irs.SecurityInfo{}, errVnet
	//}
}

// Target pools are used for network TCP/UDP load balancing. A target pool references member instances, an associated legacy HttpHealthCheck resource, and, optionally, a backup target poo
func (nlbHandler *GCPNLBHandler) deleteTargetPool(regionID string, targetPoolName string) (bool, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID
	fmt.Println("begin ", targetPoolName)
	req, err := nlbHandler.Client.TargetPools.Delete(projectID, regionID, targetPoolName).Do()
	if err != nil {
		return false, err
	}
	fmt.Println("req", req)
	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.Name, false)
	if err != nil {
		return false, err
	}
	fmt.Println("result")
	return true, nil

}

// toString 용
func printToJson(class interface{}) {
	e, err := json.Marshal(class)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(e))
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
	loadBalancerScheme := "EXTERNAL"

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
// nlbInfo 정보를 gcp의 TargetPool에 맞게 변경
	FailoverRatio : 설정 시 backupPool도 설정해야 함.
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
func convertNlbInfoToTargetPool(nlbInfo irs.NLBInfo) compute.TargetPool {
	vmList := nlbInfo.VMGroup.VMs

	instances := []string{}
	for _, instance := range *vmList {
		instances = append(instances, instance.SystemId) // URL
		printToJson(instance)
	}

	healthChecks := []string{nlbInfo.HealthChecker.CspID} // url

	targetPool := compute.TargetPool{
		Name:         nlbInfo.IId.NameId,
		Instances:    instances,
		HealthChecks: healthChecks,
	}
	return targetPool
}

func convertRegionForwardingRuleToNlbListener(forwardingRule *compute.ForwardingRule) irs.ListenerInfo {
	listenerInfo := irs.ListenerInfo{
		Protocol: forwardingRule.IPProtocol,
		IP:       forwardingRule.IPAddress,
		Port:     forwardingRule.PortRange,
		//DNSName:  forwardingRule., // 향후 사용할 때 Adderess에서 가져올 듯
		CspID: forwardingRule.Name, // forwarding rule name 전체
		//KeyValueList:
	}
	return listenerInfo
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
func (nlbHandler *GCPNLBHandler) extractVmGroup(targetPool *compute.TargetPool) irs.VMGroupInfo {
	vmGroup := irs.VMGroupInfo{}

	//targetPool, err := nlbHandler.getTargetPool(regionID, targetPoolName)
	//if err != nil {
	//	fmt.Println("targetPoolList  list: ", err)
	//}
	if targetPool.Instances != nil {
		printToJson(targetPool)

		// instances iid set
		instanceIIDs := []irs.IID{}

		for _, instanceId := range targetPool.Instances {
			targetPoolInstanceIndex := strings.LastIndex(instanceId, "/")
			targetPoolInstanceValue := instanceId[(targetPoolInstanceIndex + 1):]

			//instanceIID := irs.IID{SystemId: instanceId}
			instanceIID := irs.IID{NameId: targetPoolInstanceValue, SystemId: instanceId}
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

	//targetPool, err := nlbHandler.getTargetPool(regionID, targetPoolName)
	//if err != nil {
	//	fmt.Println("targetPoolList  list: ", err)
	//}
	if targetPool.Instances != nil {
		printToJson(targetPool)

		// health checker에 대한 ID는 가지고 있으나 내용은 갖고 있지 않아 정보 조회 필요.
		for _, healthChecker := range targetPool.HealthChecks {
			printToJson(healthChecker)
			targetHealthCheckerIndex := strings.LastIndex(healthChecker, "/")
			targetHealthCheckerValue := healthChecker[(targetHealthCheckerIndex + 1):]

			fmt.Println("GlobalHttpHealthChecks start: ", regionID, " : "+targetHealthCheckerValue)
			//targetHealthCheckerInfo, err := nlbHandler.getRegionHealthChecks(regionID, targetHealthCheckerValue)
			targetHealthCheckerInfo, err := nlbHandler.getHttpHealthChecks(targetHealthCheckerValue) // healthchecker는 전역
			if err != nil {
				fmt.Println("targetHealthCheckerInfo : ", err)
				return returnHealthChecker, err
			}
			if targetHealthCheckerInfo != nil {
				printToJson(targetHealthCheckerInfo)

				returnHealthChecker.CspID = targetHealthCheckerInfo.SelfLink
				returnHealthChecker.Protocol = HealthCheck_Http
				returnHealthChecker.Port = strconv.FormatInt(targetHealthCheckerInfo.Port, 10)
				returnHealthChecker.Interval = int(targetHealthCheckerInfo.CheckIntervalSec)
				returnHealthChecker.Timeout = int(targetHealthCheckerInfo.TimeoutSec)
				returnHealthChecker.Threshold = int(targetHealthCheckerInfo.HealthyThreshold)
				//healthChecker.KeyValueList[], KeyValue
			}
			fmt.Println("GlobalHttpHealthChecks end: ")
		}

	}
	return returnHealthChecker, nil
}
