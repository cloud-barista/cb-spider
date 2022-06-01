package resources

import (
	"context"
	"encoding/json"
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

// GCP는 동일 vpc가 아니어도 LB 생성가능, but Spider는 동일 vpc에 있어야하므로 사용할 instance 들이 동일한 VPC에 있는지 체크 필요
// 대상 풀 기반 외부 TCP/UDP 네트워크 부하 분산
// 아키텍쳐 : 대상 풀 1개, 여러 전달규칙 ( https://cloud.google.com/load-balancing/docs/network/networklb-target-pools?hl=ko )
func (nlbHandler *GCPNLBHandler) CreateNLB(nlbReqInfo irs.NLBInfo) (irs.NLBInfo, error) {
	// GCP
	//		name
	//		region
	//		[] network (vnet) - frontend configuration
	//			ip
	//			protocol
	//			port
	//		backend service - backend configuration {name, backend type(instance group), protocol, port, timeout ...}
	//		routing rules - simple [ {host1, path1, backend1 } ]
	//					  - advanced [ host2, path matches, actions and services ... ]
	fmt.Println("CreateNLB")
	projectID := nlbHandler.Credential.ProjectID
	regionID := nlbHandler.Region.Region
	fmt.Println("start ", projectID)

	// backend service 없음.
	// region forwarding rule, targetpool, targetpool안의 instance에서 사용하는 healthchecker

	fmt.Println("frontend (forwarding rule) 조회")
	//forwardingRules, err := nlbHandler.getForwardingRules("", "lbtest-global01-forwarding-rule")
	//if err != nil {
	//	fmt.Println(err)
	//}
	//fmt.Println(forwardingRules)

	//forwardingRules, err := nlbHandler.listGlobalForwardingRules("")
	//if err != nil {
	//	fmt.Println(err)
	//}
	//fmt.Println(forwardingRules)

	// TODO : 현재 global forwardingRule는 지원하지 않음
	//forwardingRules, err := nlbHandler.getGlobalForwardingRules("lbtest-global01-forwarding-rule")
	//if err != nil {
	//	fmt.Println(err)
	//}
	//fmt.Println(forwardingRules)
	//fmt.Println("frontend (forwarding rule) 조회 완료")
	//
	//fmt.Println("before return")
	//if 1 == 1 {
	//	return irs.NLBInfo{}, nil
	//}

	// InstanceGroup 조회
	// InstanceGroup이 없으면 error

	//type NLBInfo struct {
	//	IId		IID 	// {NameId, SystemId}
	//	VpcIID		IID	// {NameId, SystemId}
	//
	//	Type		string	// PUBLIC(V) | INTERNAL
	//	Scope		string	// REGION(V) | GLOBAL
	//
	//	//------ Frontend
	//	Listener	ListenerInfo
	//
	//	//------ Backend
	//	VMGroup		VMGroupInfo
	//	HealthChecker	HealthCheckerInfo
	//
	//	CreatedTime	time.Time
	//	KeyValueList []KeyValue
	//}
	//
	//type ListenerInfo struct {
	//	Protocol	string	// TCP|UDP
	//	IP		string	// Auto Generated and attached
	//	Port		string	// 1-65535
	//	DNSName		string	// Optional, Auto Generated and attached
	//
	//	CspID		string	// Optional, May be Used by Driver.
	//	KeyValueList []KeyValue
	//}
	//
	//type VMGroupInfo struct {
	//	Protocol        string	// TCP|UDP|HTTP|HTTPS
	//	Port            string	// 1-65535
	//	VMs		*[]IID
	//
	//	CspID		string	// Optional, May be Used by Driver.
	//	KeyValueList []KeyValue
	//}
	//
	//type HealthCheckerInfo struct {
	//	Protocol	string	// TCP|HTTP|HTTPS
	//	Port		string	// Listener Port or 1-65535
	//	Interval	int	// secs, Interval time between health checks.
	//	Timeout		int	// secs, Waiting time to decide an unhealthy VM when no response.
	//	Threshold	int	// num, The number of continuous health checks to change the VM status.
	//
	//	KeyValueList	[]KeyValue
	//}
	//
	//type HealthInfo struct {
	//	AllVMs		*[]IID
	//	HealthyVMs	*[]IID
	//	UnHealthyVMs	*[]IID
	//}

	// FilterLabels: The list of label value pairs that must match labels in
	// the provided metadata based on filterMatchCriteria
	// This list must not be empty and can have at the most 64 entries.
	//FilterLabels []*MetadataFilterLabelMatch `json:"filterLabels,omitempty"`

	//FilterMatchCriteria string `json:"filterMatchCriteria,omitempty"`
	metadataFilterLabelMatch := compute.MetadataFilterLabelMatch{
		Name:  "",
		Value: "",
	}
	var metadataFilterLabelMatchList []*compute.MetadataFilterLabelMatch
	metadataFilterLabelMatchList = append(metadataFilterLabelMatchList, &metadataFilterLabelMatch)

	var metadataFilter = compute.MetadataFilter{
		FilterMatchCriteria: "", // enum MATCH_ANY, MATCH_ALL, NOT_SET
		FilterLabels:        metadataFilterLabelMatchList,
	}
	var metadataFilterList []*compute.MetadataFilter
	metadataFilterList = append(metadataFilterList, &metadataFilter)

	// forwarding rule : forwarding rule은 region, global 동일.
	forwardingRule := compute.ForwardingRule{
		//Kind string `json:"kind,omitempty"`	// output only
		//Id uint64 `json:"id,omitempty,string"`// output only
		//CreationTimestamp string `json:"creationTimestamp,omitempty"`// output only
		//Name string `json:"name,omitempty"`
		Name: nlbReqInfo.IId.NameId,
		//Description string `json:"description,omitempty"`
		//IPAddress string `json:"IPAddress,omitempty"`
		IPAddress: nlbReqInfo.Listener.IP,

		//AllPorts bool `json:"allPorts,omitempty"`
		AllPorts: false, //You can only use one of ports and portRange, or allPorts. The three are mutually exclusive.
		//BackendService string `json:"backendService,omitempty"`

		//IpVersion string `json:"ipVersion,omitempty"`
		IpVersion: "IPV4", // IPV4 or IPV6
		//LoadBalancingScheme string `json:"loadBalancingScheme,omitempty"`
		LoadBalancingScheme: "EXTERNAL_MANAGED",
		//MetadataFilters []*MetadataFilter `json:"metadataFilters,omitempty"`
		MetadataFilters: metadataFilterList,

		//Network string `json:"network,omitempty"`
		//NetworkTier string `json:"networkTier,omitempty"`
		//PortRange string `json:"portRange,omitempty"`
		//Ports []string `json:"ports,omitempty"`
		Region: "asia-northeast3",
		//SelfLink string `json:"selfLink,omitempty"`
		//ServiceLabel string `json:"serviceLabel,omitempty"`
		//ServiceName string `json:"serviceName,omitempty"`
		//Subnetwork string `json:"subnetwork,omitempty"`
		//Target string `json:"target,omitempty"`
		//
		//googleapi.ServerResponse `json:"-"`
		//ForceSendFields []string `json:"-"`
		//NullFields []string `json:"-"`
	}
	fmt.Println("forwardingRule  : ", forwardingRule)
	//result, err := nlbHandler.Client.ForwardingRules.Insert(projectID, regionID, &forwardingRule).Do()
	//if err != nil {
	//	return irs.NLBInfo{}, err
	//}
	//fmt.Println("result  : ", result)
	regionForwardingRuleResult, err := nlbHandler.insertRegionForwardingRules(regionID, &forwardingRule)
	if err != nil {
		return irs.NLBInfo{}, err
	}
	fmt.Println("result  : ", regionForwardingRuleResult)

	// backend rule

	return irs.NLBInfo{}, nil
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
	regionForwardingRuleList, err := nlbHandler.listRegionForwardingRules(regionID, "")
	if err != nil {
		fmt.Println("regionForwardingRule  list: ", err)
	}
	if regionForwardingRuleList != nil { // dial tcp: lookup compute.googleapis.com: no such host 일 때, 	panic: runtime error: invalid memory address or nil pointer dereference
		if len(regionForwardingRuleList.Items) > 0 {
			for _, forwardingRule := range regionForwardingRuleList.Items {
				targetLbIndex := strings.LastIndex(forwardingRule.Target, "/")
				targetLbValue := forwardingRule.Target[(targetLbIndex + 1):]

				targetForwardingRuleIndex := strings.LastIndex(forwardingRule.Name, "/")
				targetForwardingRuleValue := forwardingRule.Name[(targetForwardingRuleIndex + 1):]

				// targetlink에서 lb 추출
				//targetNlbInfo := nlbMap[targetLbValue]
				newNlbInfo, exists := nlbMap[targetLbValue]
				if exists {
					// spider는 1개의 listener(forwardingrule)만 사용하므로 skip
				} else {
					listenerInfo := irs.ListenerInfo{
						Protocol: forwardingRule.IPProtocol,
						IP:       forwardingRule.IPAddress,
						Port:     forwardingRule.PortRange,
						//DNSName:  forwardingRule., // 향후 사용할 때 Adderess에서 가져올 듯
						//CspID: targetLbValue, // LoadBalancer Name ?
						CspID: forwardingRule.Name, // forwarding rule name
						//KeyValueList:
					}

					createdTime, _ := time.Parse(
						time.RFC3339,
						forwardingRule.CreationTimestamp) // RFC3339형태이므로 해당 시간으로 다시 생성

					loadBalancerType := forwardingRule.LoadBalancingScheme
					if strings.EqualFold(loadBalancerType, "EXTERNAL") { // PUBLIC/INTERNAL : extenel -> PUBLIC으로 변경
						loadBalancerType = "PUBLIC"
					}

					newNlbInfo = irs.NLBInfo{
						IId:         irs.IID{NameId: targetLbValue, SystemId: targetForwardingRuleValue}, // NameId :Lb Name, poolName, SystemId : forwardingRule Name
						VpcIID:      irs.IID{NameId: "", SystemId: ""},                                   // VpcIID 는 Pool 안의 instance에 있는 값
						Type:        loadBalancerType,                                                    // PUBLIC/INTERNAL : extenel -> PUBLIC으로 변경하는 로직 적용해야함.
						Scope:       "REGION",
						Listener:    listenerInfo,
						CreatedTime: createdTime, //RFC3339 "creationTimestamp":"2022-05-24T01:20:40.334-07:00"
						//KeyValueList  []KeyValue
					}
				}
				nlbMap[targetLbValue] = newNlbInfo
				printToJson(forwardingRule)
			}
		}
	}

	//fmt.Println("forwardingRules result size  : ", len(regionForwardingRuleList.Items))
	fmt.Println("regionForwardingRule end: ")

	fmt.Println("nlbMap end: ", nlbMap)

	//fmt.Println("global forwardingRules start: ")
	//globalResult, err := nlbHandler.Client.GlobalForwardingRules.List(projectID).Do()
	//if err != nil {
	//	fmt.Println("GlobalForwardingRules  list: ", err)
	//}
	//for _, globalForwardinfRule := range globalResult.Items {
	//	//var nlbInfo irs.NLBInfo
	//	//nlbInfo.FrontendIP = forwardinfRule.IPAddress
	//	//nlbInfo.FrontendDNSName= forwardinfRule.
	//	fmt.Println("GlobalForwardingRules  : ", globalForwardinfRule)
	//}
	//fmt.Println("GlobalForwardingRules  : ", globalResult)
	//fmt.Println("GlobalForwardingRules size  : ", len(globalResult.Items))
	//fmt.Println("globalforwardingRules end: ")

	// gcp console에서 보이는 영역은 forwardingrule임.
	// TODO : urlMapList의 용도 확인필요.
	//fmt.Println("region urlMapping start: ")
	//regionUrlMapList, err := nlbHandler.Client.RegionUrlMaps.List(projectID, regionID).Do()
	//if err != nil {
	//	fmt.Println("regionUrlMapList  list: ", err)
	//}
	//for _, regionUrlMap := range regionUrlMapList.Items {
	//	fmt.Println("regionUrlMap  : ", regionUrlMap)
	//}
	//fmt.Println("regionUrlMapList  : ", regionUrlMapList)
	//fmt.Println("regionUrlMapList size  : ", len(regionUrlMapList.Items))
	//fmt.Println("region urlMapping end: ")
	//
	//fmt.Println("global urlMapping start: ")
	//globalUrlMapList, err := nlbHandler.Client.UrlMaps.List(projectID).Do()
	//if err != nil {
	//	fmt.Println("globalUrlMapList  list: ", err)
	//}
	//for _, globalUrlMap := range globalUrlMapList.Items {
	//	fmt.Println("globalUrlMap  : ", globalUrlMap)
	//}
	//fmt.Println("globalUrlMapList  : ", globalUrlMapList)
	//fmt.Println("globalUrlMapList size  : ", len(globalUrlMapList.Items))
	//fmt.Println("global urlMapping end: ")

	//fmt.Println("region targetHttpProxies start: ")
	//targetHttpProxiesResult, err := nlbHandler.Client.RegionTargetHttpProxies.List(projectID, regionID).Do()
	//if err != nil {
	//	fmt.Println("GlobalForwardingRules  list: ", err)
	//}
	//for _, targetHttpProxiy := range targetHttpProxiesResult.Items {
	//	fmt.Println("targetHttpProxiy  : ", targetHttpProxiy)
	//}
	//fmt.Println("region targetHttpProxies  : ", targetHttpProxiesResult)
	//fmt.Println("region targetHttpProxies size  : ", len(targetHttpProxiesResult.Items))
	//fmt.Println("region targetHttpProxies end: ")

	//req := nlbHandler.Client.ForwardingRules.List(projectID, regionID)
	//if err := req.Pages(ctx, func(page *compute.ForwardingRuleList) error {
	//	for _, forwardingRule := range page.Items {
	//		// TODO: Change code below to process each `forwardingRule` resource:
	//		fmt.Printf("%#v\n", forwardingRule)
	//	}
	//	return nil
	//}); err != nil {
	//	log.Fatal(err)
	//}

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
		targetPoolValue := targetPool.SelfLink[(targetPoolIndex + 1):]
		newNlbInfo.VMGroup.CspID = targetPoolValue

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
			targetHealthCheckerInfo, err := nlbHandler.getGlobalHttpHealthChecks(targetHealthCheckerValue) // healthchecker는 전역
			if err != nil {
				fmt.Println("targetHealthCheckerInfo : ", err)
			}
			if targetHealthCheckerInfo != nil {
				printToJson(targetHealthCheckerInfo)

				healthCheckerInfo := irs.HealthCheckerInfo{
					Protocol:  "TCP",
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

	//fmt.Println("region healthCheck start: ", regionID)
	//regionHealthCheckList, err := nlbHandler.listRegionHealthChecks(regionID, "")
	//if err != nil {
	//	fmt.Println("regionBackendServiceList  list: ", err)
	//}
	//if regionHealthCheckList != nil {
	//	for _, regionHealthCheck := range regionHealthCheckList.Items {
	//		// name = regionHealthCheck.Name
	//		printToJson(regionHealthCheck)
	//	}
	//}
	//fmt.Println("regionHealthCheckList result size  : ", len(regionHealthCheckList.Items))
	//fmt.Println("regionHealthCheckList end: ")
	//printToJson(nlbMap)

	return nlbInfoList, nil
}

// Load balancer 조회
// nlbIID 에서 NameId = lbName, targetPoolName
// nlbIID 에서 SystemId = forwardingRuleId   -> forwardingRule을 찾을 방법이 없어서 systemId 사용
func (nlbHandler *GCPNLBHandler) GetNLB(nlbIID irs.IID) (irs.NLBInfo, error) {
	var nlbInfo irs.NLBInfo

	projectID := nlbHandler.Credential.ProjectID
	regionID := nlbHandler.Region.Region

	// instace에서 vpc의 IID 조회를 위해
	vmHandler := GCPVMHandler{
		Client:     nlbHandler.Client,
		Region:     nlbHandler.Region,
		Ctx:        nlbHandler.Ctx,
		Credential: nlbHandler.Credential,
	}

	fmt.Println("projectID: ", projectID)
	fmt.Println("regionID: ", regionID)

	// region forwarding rule 는 target pool 과 lb이름으로 엮임.
	// map에 nb이름으로 nbInfo를 넣고 해당 값들 추가해서 조합
	fmt.Println("region forwardingRules start: ", regionID)
	regionForwardingRule, err := nlbHandler.getRegionForwardingRules(regionID, nlbIID.SystemId)
	if err != nil {
		fmt.Println("regionForwardingRule  list: ", err)
	}
	if regionForwardingRule != nil { // dial tcp: lookup compute.googleapis.com: no such host 일 때, 	panic: runtime error: invalid memory address or nil pointer dereference

		targetLbIndex := strings.LastIndex(regionForwardingRule.Target, "/")
		targetLbValue := regionForwardingRule.Target[(targetLbIndex + 1):]

		targetForwardingRuleIndex := strings.LastIndex(regionForwardingRule.Name, "/")
		targetForwardingRuleValue := regionForwardingRule.Name[(targetForwardingRuleIndex + 1):]

		// targetlink에서 lb 추출
		//targetNlbInfo := nlbMap[targetLbValue]
		listenerInfo := irs.ListenerInfo{
			Protocol: regionForwardingRule.IPProtocol,
			IP:       regionForwardingRule.IPAddress,
			Port:     regionForwardingRule.PortRange,
			//DNSName:  forwardingRule., // 향후 사용할 때 Adderess에서 가져올 듯
			CspID: regionForwardingRule.Name, // forwarding rule name 전체
			//KeyValueList:
		}

		createdTime, _ := time.Parse(
			time.RFC3339,
			regionForwardingRule.CreationTimestamp) // RFC3339형태이므로 해당 시간으로 다시 생성

		loadBalancerType := regionForwardingRule.LoadBalancingScheme
		if strings.EqualFold(loadBalancerType, "EXTERNAL") {
			loadBalancerType = "PUBLIC"
		}

		nlbInfo = irs.NLBInfo{
			IId:         irs.IID{NameId: targetLbValue, SystemId: targetForwardingRuleValue}, // NameId = Lb Name, SystemId = forwardingRule name
			VpcIID:      irs.IID{NameId: "", SystemId: ""},                                   // VpcIID 는 Pool 안의 instance에 있는 값
			Type:        loadBalancerType,                                                    // PUBLIC/INTERNAL : extenel -> PUBLIC으로 변경하는 로직 적용해야함.
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

	targetPool, err := nlbHandler.getTargetPool(regionID, nlbIID.NameId)
	if err != nil {
		fmt.Println("targetPoolList  list: ", err)
	}
	printToJson(targetPool)

	vpcInstanceName := "" // vpc를 갸져올 instance 이름

	// vmGroup == targetPool
	//"name":"lb-test-seoul-03",
	//"selfLink":"https://www.googleapis.com/compute/v1/projects/yhnoh-335705/regions/asia-northeast3/targetPools/lb-test-seoul-03",
	targetPoolIndex := strings.LastIndex(targetPool.SelfLink, "/")
	targetPoolValue := targetPool.SelfLink[(targetPoolIndex + 1):]
	nlbInfo.VMGroup.CspID = targetPoolValue

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
	nlbInfo.VMGroup.VMs = &instanceIIDs
	fmt.Println("instanceIIDs : ", targetPool.Name)
	fmt.Println("instanceIIDs----- : ", instanceIIDs)
	fmt.Println("vpcInstanceName----- : ", vpcInstanceName)
	fmt.Println("newNlbInfo.VMGroup.VMs----- : ", nlbInfo.VMGroup.VMs)

	// health checker에 대한 ID는 가지고 있으나 내용은 갖고 있지 않아 정보 조회 필요.
	for _, healthChecker := range targetPool.HealthChecks {
		printToJson(healthChecker)
		targetHealthCheckerIndex := strings.LastIndex(healthChecker, "/")
		targetHealthCheckerValue := healthChecker[(targetHealthCheckerIndex + 1):]

		fmt.Println("GlobalHttpHealthChecks start: ", regionID, " : "+targetHealthCheckerValue)
		//targetHealthCheckerInfo, err := nlbHandler.getRegionHealthChecks(regionID, targetHealthCheckerValue)
		targetHealthCheckerInfo, err := nlbHandler.getGlobalHttpHealthChecks(targetHealthCheckerValue) // healthchecker는 전역
		if err != nil {
			fmt.Println("targetHealthCheckerInfo : ", err)
		}
		if targetHealthCheckerInfo != nil {
			printToJson(targetHealthCheckerInfo)

			healthCheckerInfo := irs.HealthCheckerInfo{
				Protocol:  "TCP",
				Port:      strconv.FormatInt(targetHealthCheckerInfo.Port, 10),
				Interval:  int(targetHealthCheckerInfo.CheckIntervalSec),
				Timeout:   int(targetHealthCheckerInfo.TimeoutSec),
				Threshold: int(targetHealthCheckerInfo.HealthyThreshold),
				//KeyValueList[], KeyValue
			}

			//printToJson(healthCheckerInfo)
			nlbInfo.HealthChecker = healthCheckerInfo
		}

		fmt.Println("GlobalHttpHealthChecks end: ")
	}

	// vpcIID 조회
	vNetVmInfo, err := vmHandler.GetVM(irs.IID{SystemId: vpcInstanceName})
	if err != nil {
		fmt.Println("fail to get VPC Info : ", err)
	}
	//printToJson(vNetVmInfo)
	nlbInfo.VpcIID = vNetVmInfo.VpcIID

	fmt.Println("Targetpool end: ")
	printToJson(nlbInfo)

	return nlbInfo, nil
}
func (nlbHandler *GCPNLBHandler) DeleteNLB(nlbIID irs.IID) (bool, error) {
	return false, nil
}

//------ Frontend Control
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
	return irs.ListenerInfo{}, nil
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

	return irs.VMGroupInfo{}, nil
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

	err = WaitUntilComplete(nlbHandler.Client, projectID, "", req.ClientOperationId, true)
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
	//errWait := vVPCHandler.WaitUntilComplete(strconv.FormatUint(req.Id, 10), true)

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

	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.ClientOperationId, true)
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

	err = WaitUntilComplete(nlbHandler.Client, projectID, "", req.ClientOperationId, true)
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
	//errWait := vVPCHandler.WaitUntilComplete(strconv.FormatUint(req.Id, 10), true)

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

	err = WaitUntilComplete(nlbHandler.Client, projectID, "", req.ClientOperationId, true)
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

	err = WaitUntilComplete(nlbHandler.Client, projectID, "", req.ClientOperationId, true)
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

	err = WaitUntilComplete(nlbHandler.Client, projectID, "", req.ClientOperationId, true)
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

	err = WaitUntilComplete(nlbHandler.Client, projectID, "", req.ClientOperationId, true)
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
func (nlbHandler *GCPNLBHandler) insertRegionForwardingRules(regionID string, reqRegionForwardingRule *compute.ForwardingRule) (*compute.ForwardingRule, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID
	//region := nlbHandler.Region.Region

	//reqForwardingRule := &compute.ForwardingRule{}
	req, err := nlbHandler.Client.ForwardingRules.Insert(projectID, regionID, reqRegionForwardingRule).Do()
	if err != nil {
		return &compute.ForwardingRule{}, err
	}

	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.ClientOperationId, true)
	if err != nil {
		return &compute.ForwardingRule{}, err
	}

	id := req.Id
	name := req.Name
	fmt.Println("id = ", id, " : name = ", name)
	forwardingRule, err := nlbHandler.getRegionForwardingRules(regionID, reqRegionForwardingRule.Name)
	if err != nil {
		return &compute.ForwardingRule{}, err
		//return nil, err
	}

	fmt.Println("ForwardingRule ", forwardingRule)
	return forwardingRule, nil
}

// Region ForwardingRule patch
func (nlbHandler *GCPNLBHandler) patchRegionForwardingRules(regionID string, forwardingRuleName string, patchRegionForwardingRule *compute.ForwardingRule) (*compute.ForwardingRule, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID
	//region := nlbHandler.Region.Region

	// 현재 ForwardingRules.Patch 가 안보임. google.golang.org/api v0.50.0
	req, err := nlbHandler.Client.ForwardingRules.Patch(projectID, regionID, forwardingRuleName, patchRegionForwardingRule).Do()

	//// 삭제 후 insert
	//delReq, delErr := nlbHandler.Client.ForwardingRules.Delete(projectID, region, forwardingRuleName).Do()
	//if delErr != nil {
	//	return &compute.ForwardingRule{}, delErr
	//}
	//delErr = WaitUntilComplete(nlbHandler.Client, projectID, region, delReq.ClientOperationId, true)
	//if delErr != nil {
	//	return &compute.ForwardingRule{}, delErr
	//}
	//
	//req, err := nlbHandler.Client.ForwardingRules.Insert(projectID, region, patchRegionForwardingRule).Do()
	//if err != nil {
	//	return &compute.ForwardingRule{}, err
	//}

	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.ClientOperationId, false)
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
func (nlbHandler *GCPNLBHandler) listRegionForwardingRules(regionID string, filter string) (*compute.ForwardingRuleList, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.ForwardingRules.List(projectID, regionID).Do()
	if err != nil {
		return nil, err
	}
	for _, item := range resp.Items {
		fmt.Println(item)
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

	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.ClientOperationId, true)
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

	err = WaitUntilComplete(nlbHandler.Client, projectID, "", req.ClientOperationId, true)
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

	err = WaitUntilComplete(nlbHandler.Client, projectID, "", req.ClientOperationId, true)
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

func (nlbHandler *GCPNLBHandler) insertGlobalHealthChecks(healthCheckerInfo irs.HealthCheckerInfo) {
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

// Inteal Http(S) load balancer : region health check => compute.v1.regionHealthCheck
// Traffic Director : global health check => compute.v1.HealthCheck
func (nlbHandler *GCPNLBHandler) getGlobalHealthChecks(healthCheckName string) (*compute.HealthCheck, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.HealthChecks.Get(projectID, healthCheckName).Do()

	if err != nil {
		return nil, err
	}
	return resp, nil
}
func (nlbHandler *GCPNLBHandler) getGlobalHttpHealthChecks(healthCheckName string) (*compute.HttpHealthCheck, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.HttpHealthChecks.Get(projectID, healthCheckName).Do()

	if err != nil {
		return nil, err
	}
	return resp, nil
}

// Global BackendService 목록 조회
// HealthCheckList 객체를 넘기고 사용은 HealthCheckList.Item에서 꺼내서 사용
func (nlbHandler *GCPNLBHandler) listGlobalHealthChecks(healthCheckName string) (*compute.HealthCheckList, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.HealthChecks.List(projectID).Do()
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

	//healthCheckPort :=	&
	targetPool := &compute.TargetPool{}

	// requestBody
	req, err := nlbHandler.Client.TargetPools.Insert(projectID, regionID, targetPool).Do()
	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.ClientOperationId, true)
	if err != nil {
		return &compute.TargetPool{}, err
	}

	id := req.Id
	name := req.Name
	fmt.Println("id = ", id, " : name = ", name)
	result, err := nlbHandler.getTargetPool(regionID, name)
	if err != nil {
		return &compute.TargetPool{}, err
	}

	fmt.Println("insertTargetPool ", result)
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
	err = WaitUntilComplete(nlbHandler.Client, projectID, regionID, req.ClientOperationId, true)
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

// Target pools are used for network TCP/UDP load balancing. A target pool references member instances, an associated legacy HttpHealthCheck resource, and, optionally, a backup target poo
func (nlbHandler *GCPNLBHandler) addInstance(healthCheckerInfo irs.HealthCheckerInfo) {
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
func (nlbHandler *GCPNLBHandler) removeInstance(healthCheckerInfo irs.HealthCheckerInfo) {
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
func (nlbHandler *GCPNLBHandler) deleteTargetPool(healthCheckName string) (*compute.HealthCheckList, error) {

	// path param
	projectID := nlbHandler.Credential.ProjectID

	resp, err := nlbHandler.Client.HealthChecks.List(projectID).Do()
	if err != nil {
		return nil, err
	}

	for _, item := range resp.Items {
		fmt.Println(item)
	}

	return resp, nil

}

func printToJson(class interface{}) {
	e, err := json.Marshal(class)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(string(e))
}
