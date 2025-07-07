package resources

import (
	"context"
	"fmt"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	ncloud "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/ncloud"
	vas "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vautoscaling"
	vnks "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vnks"
	vpc "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vpc"
	vserver "github.com/NaverCloudPlatform/ncloud-sdk-go-v2/services/vserver"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpVpcClusterHandler struct {
	RegionInfo    idrv.RegionInfo
	Ctx           context.Context
	VMClient      *vserver.APIClient
	VPCClient     *vpc.APIClient
	ClusterClient *vnks.APIClient
	ASClient      *vas.APIClient
}

const (
	clusterTypeAAA = "AAA.VNKS.STAND.C002.M008.G000"

	hypervisorCodeXen     = "XEN"
	hypervisorCodeKvm     = "KVM"
	hypervisorCodeDefault = hypervisorCodeXen

	defaultServerImageNamePrefixForXen = "ubuntu-20"
	defaultServerImageNamePrefixForKvm = "ubuntu-22.04-nks"

	searchColumnRoleName = "roleName"

	// https://guide.ncloud-docs.com/docs/k8s-k8sprep
	lbSubnetPrefixLengthForK8s          = 26
	defaultPrivateLbSubnetForK8s string = "cb-private-lb-subnet-for-k8s"
	defaultPublicLbSubnetForK8s  string = "cb-public-lb-subnet-for-k8s"

	defaultNetworkAclName = "default-network-acl"
)

const (
	NODEGROUP_TAG string = "nodegroup"
)

// ------ Cluster Management
func (nvch *NcpVpcClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (irs.ClusterInfo, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NCPVPC Cloud Driver: called CreateCluster()")
	emptyClusterInfo := irs.ClusterInfo{}
	hiscallInfo := GetCallLogScheme(nvch.RegionInfo.Region, call.CLUSTER, clusterReqInfo.IId.NameId, "CreateCluster()")
	start := call.Start()

	cblogger.Info("Create Cluster")

	var clusterId string
	var createErr error
	defer func() {
		if createErr != nil {
			cblogger.Error(createErr)
			LoggingError(hiscallInfo, createErr)

			if clusterId != "" {
				_ = nvch.deleteCluster(clusterId)
				cblogger.Infof("Cluster(Name=%s) will be Deleted.", clusterReqInfo.IId.NameId)
			}
		}
	}()

	//
	// Validation
	//
	supportedK8sVersions, err := nvch.getSupportedK8sVersions()
	if err != nil {
		createErr = fmt.Errorf("Failed to Create Cluster: %v", err)
		return emptyClusterInfo, createErr
	}

	err = validateAtCreateCluster(clusterReqInfo, supportedK8sVersions)
	if err != nil {
		createErr = fmt.Errorf("Failed to Create Cluster: %v", err)
		return emptyClusterInfo, createErr
	}

	//
	// Create Cluster
	//
	clusterId, err = nvch.createCluster(&clusterReqInfo)
	if err != nil {
		createErr = fmt.Errorf("Failed to Create Cluster: %v", err)
		return emptyClusterInfo, createErr
	}
	cblogger.Debug("To Create a Cluster is In Progress.")

	/*
		err = nvch.waitUntilClusterSecGroupIsCreated(clusterId)
		if err != nil {
			createErr = fmt.Errorf("Failed to Create Cluster: %v", err)
			return emptyClusterInfo, createErr
		}
	*/

	//
	// Get ClusterInfo
	//

	// Get ClusterInfo (생성된 클러스터의 uuid로 정보 조회)
	clusterInfo, err := nvch.GetCluster(irs.IID{SystemId: clusterId})
	if err != nil {
		createErr = fmt.Errorf("Failed to Get Created Cluster Info: %v", err)
		return emptyClusterInfo, createErr
	}

	LoggingInfo(hiscallInfo, start)
	cblogger.Infof("Creating Cluster(name=%s, id=%s)", clusterInfo.IId.NameId, clusterInfo.IId.SystemId)
	return clusterInfo, nil
}

func (nvch *NcpVpcClusterHandler) getSupportedK8sVersions() ([]string, error) {
	versions, err := ncpvpcOptionVersionGet(nvch.ClusterClient, nvch.Ctx, hypervisorCodeXen)
	if err != nil {
		return make([]string, 0), err
	}

	supportedK8sVersions := make([]string, 0)
	for _, version := range versions {
		value := ncloud.StringValue(version.Value)
		if strings.EqualFold(value, "") == false {
			supportedK8sVersions = append(supportedK8sVersions, value)
		}
	}

	return supportedK8sVersions, nil
}

func (nvch *NcpVpcClusterHandler) createCluster(clusterReqInfo *irs.ClusterInfo) (string, error) {
	clusterName := clusterReqInfo.IId.NameId

	// 1. 필수 파라미터 검증
	if clusterName == "" || clusterReqInfo.Version == "" || clusterReqInfo.Network.VpcIID.SystemId == "" {
		return "", fmt.Errorf("required parameter missing (name, version, vpcNo)")
	}

	// 2. NCP 특화 옵션(KeyValueList) 파싱 및 기본값 처리
	// productCode는 "hypervisorCode"가 XEN인 경우만 사용하는 optional 변수라고 되어있으나 지정이 꼭 필요.
	// clusterType := "SVR.VNKS.STAND.C002.M008.NET.SSD.B050.G002"  // XEN
	// clusterType := SVR.VNKS.STAND.C004.M016.NET.SSD.B050.G002  	// XEN
	clusterType := "SVR.VNKS.STAND.C002.M008.NET.SSD.B050.G002" // KVM
	// clusterType := "SVR.VNKS.STAND.C004.M016.NET.SSD.B050.G002"	// KVM
	// clusterType := "SVR.VNKS.STAND.C004.M016.G003" // API 문서상의 KVM, 하지만 실제로는 지원하지 않음
	publicNetwork := true // 기본값
	for _, kv := range clusterReqInfo.KeyValueList {
		if kv.Key == "ClusterType" && kv.Value != "" {
			clusterType = kv.Value
		}
		if kv.Key == "PublicNetwork" && kv.Value != "" {
			publicNetwork = (kv.Value == "true")
		}
	}

	// 3. VPC 및 LB 서브넷 준비
	vpcHandler := NcpVpcVPCHandler{
		RegionInfo: nvch.RegionInfo,
		VPCClient:  nvch.VPCClient,
	}
	vpcIID := clusterReqInfo.Network.VpcIID
	vpc, err := vpcHandler.GetVPC(vpcIID)
	if err != nil {
		return "", fmt.Errorf("failed to get VPC: %v", err)
	}

	existPrivateLbSubnet := false
	existPublicLbSubnet := false
	existingSubnets := []string{}
	for _, subnetInfo := range vpc.SubnetInfoList {
		if strings.EqualFold(subnetInfo.IId.NameId, defaultPrivateLbSubnetForK8s) {
			existPrivateLbSubnet = true
		} else if strings.EqualFold(subnetInfo.IId.NameId, defaultPublicLbSubnetForK8s) {
			existPublicLbSubnet = true
		} else {
			existingSubnets = append(existingSubnets, subnetInfo.IPv4_CIDR)
		}
		if existPrivateLbSubnet && existPublicLbSubnet {
			break
		}
	}

	availSubnets, err := GetReverseSubnetCidrs(vpc.IPv4_CIDR, existingSubnets, lbSubnetPrefixLengthForK8s, 2)
	if err != nil {
		return "", fmt.Errorf("failed to create a cluster(%s): %v", clusterName, err)
	}
	if len(availSubnets) < 2 {
		return "", fmt.Errorf("no available subnet range for LB")
	}

	if !existPrivateLbSubnet {
		err = nvch.addSubnetAndWait(vpc.IId.SystemId, defaultPrivateLbSubnetForK8s, availSubnets[0], subnetTypeCodePrivate, usageTypeCodeLoadb)
		if err != nil {
			return "", fmt.Errorf("failed to create private LB subnet: %v", err)
		}
		availSubnets = availSubnets[1:]
	}
	if !existPublicLbSubnet {
		err = nvch.addSubnetAndWait(vpc.IId.SystemId, defaultPublicLbSubnetForK8s, availSubnets[0], subnetTypeCodePublic, usageTypeCodeLoadb)
		if err != nil {
			return "", fmt.Errorf("failed to create public LB subnet: %v", err)
		}
	}

	// 4. NodePool(nodeGroup) 매핑 및 이미지 코드 추출
	var nodePools []*vnks.NodePoolDto
	for _, ng := range clusterReqInfo.NodeGroupList {
		if ng.IId.NameId == "" || ng.DesiredNodeSize < 1 || ng.VMSpecName == "" || ng.KeyPairIID.NameId == "" {
			return "", fmt.Errorf("required node group parameter missing")
		}

		// 이미지 코드 추출
		imageName := ng.ImageIID.NameId
		var softwareCode string
		var err error
		if imageName == "" || strings.EqualFold(imageName, "default") {
			var defaultServerImageNamePrefix string
			if hypervisorCodeDefault == hypervisorCodeXen {
				defaultServerImageNamePrefix = defaultServerImageNamePrefixForXen
			} else {
				defaultServerImageNamePrefix = defaultServerImageNamePrefixForKvm
			}
			softwareCode, err = nvch.getServerImageByNamePrefix(defaultServerImageNamePrefix)
			if err != nil {
				return "", fmt.Errorf("failed to get default server image: %v", err)
			}
		} else {
			softwareCode, err = nvch.getServerImageByNamePrefix(imageName)
			if err != nil {
				serverList, err2 := nvch.getAvailableServerImageList()
				if err2 != nil {
					return "", fmt.Errorf("failed to create a cluster(%s): %v", clusterName, err2)
				}
				return "", fmt.Errorf("failed to create a cluster(%s): %v (available server images: %s)", clusterName, err, strings.Join(serverList, ", "))
			}
		}

		nodePool := &vnks.NodePoolDto{
			Name:           ncloud.String(ng.IId.NameId),
			NodeCount:      ncloud.Int32(int32(ng.DesiredNodeSize)),
			SoftwareCode:   ncloud.String(softwareCode),
			ServerSpecCode: ncloud.String(ng.VMSpecName),
			ProductCode:    ncloud.String("SVR.VSVR.STAND.C002.M008.NET.SSD.B050.G002"),
			// 필요시 SubnetNo, Labels, Taints, StorageSize 등 추가
		}
		nodePools = append(nodePools, nodePool)
	}

	// 5. Subnet, LB 서브넷 등 네트워크 정보 준비
	var subnetNoList []int32
	var lbPrivateSubnetNo, lbPublicSubnetNo int32
	for _, subnet := range vpc.SubnetInfoList {
		if subnet.IId.NameId == defaultPrivateLbSubnetForK8s {
			no, _ := strconv.ParseInt(subnet.IId.SystemId, 10, 32)
			lbPrivateSubnetNo = int32(no)
		} else if subnet.IId.NameId == defaultPublicLbSubnetForK8s {
			no, _ := strconv.ParseInt(subnet.IId.SystemId, 10, 32)
			lbPublicSubnetNo = int32(no)
		} else {
			no, _ := strconv.ParseInt(subnet.IId.SystemId, 10, 32)
			subnetNoList = append(subnetNoList, int32(no))
		}
	}

	// 6. NCP API 호출 파라미터 구성
	vpcNoInt64, err := strconv.ParseInt(vpc.IId.SystemId, 10, 32)
	if err != nil {
		return "", fmt.Errorf("failed to parse VPC SystemId '%s' to int32: %v", vpc.IId.SystemId, err)
	}
	clusterInputBody := &vnks.ClusterInputBody{
		Name:              ncloud.String(clusterName),
		ClusterType:       ncloud.String(clusterType),
		K8sVersion:        ncloud.String(clusterReqInfo.Version),
		LoginKeyName:      ncloud.String(clusterReqInfo.NodeGroupList[0].KeyPairIID.NameId),
		RegionCode:        ncloud.String(nvch.RegionInfo.Region),
		ZoneCode:          ncloud.String(nvch.RegionInfo.Zone),
		PublicNetwork:     ncloud.Bool(publicNetwork),
		VpcNo:             ncloud.Int32(int32(vpcNoInt64)),
		SubnetNoList:      int32List(subnetNoList), // []int32 → []*int32 변환 함수 사용
		LbPrivateSubnetNo: ncloud.Int32(int32(lbPrivateSubnetNo)),
		LbPublicSubnetNo:  ncloud.Int32(int32(lbPublicSubnetNo)),
		NodePool:          nodePools, // []*vnks.NodePoolDto 타입으로 전달
	}

	// 7. NCP API 호출 및 에러 발생 시 롤백 처리
	createClusterRes, err := nvch.ClusterClient.V2Api.ClustersPost(nvch.Ctx, clusterInputBody)
	if err != nil {
		// TODO: 에러 발생 시 롤백(클러스터 삭제) 구현 필요
		return "", fmt.Errorf("failed to create cluster: %v", err)
	}

	return ncloud.StringValue(createClusterRes.Uuid), nil
}

func (nvch *NcpVpcClusterHandler) addSubnetAndWait(vpcNo, subnetName, subnetRange, subnetTypeCode, usageTypeCode string) error {
	networkAclNo, err := ncpvpcGetDefaultNetworkAclNo(nvch.VPCClient, nvch.RegionInfo.Region, vpcNo)
	if err != nil {
		err := fmt.Errorf("failed to add subnet(%s, %s): %v", subnetName, subnetRange, err)
		return err
	}

	subnet, err := ncpvpcCreateSubnet(nvch.VPCClient, nvch.RegionInfo.Region,
		nvch.RegionInfo.Zone, vpcNo, subnetName, subnetRange,
		networkAclNo, subnetTypeCode, usageTypeCode)
	if err != nil {
		err := fmt.Errorf("failed to add subnet(%s, %s): %v", subnetName, subnetRange, err)
		return err
	}

	err = waitUntilSubnetIsStatus(nvch.VPCClient, nvch.RegionInfo.Region, ncloud.StringValue(subnet.SubnetNo), subnetStatusRun)
	if err != nil {
		err := fmt.Errorf("failed to add subnet(%s, %s): %v", subnetName, subnetRange, err)
		return err
	}
	return nil
}

/*
// Nodegroup이 Activty 상태일때까지 대기함.

	func (nvch *NcpVpcClusterHandler) WaitUntilNodegroupActive(clusterName string, nodegroupName string) error {
		cblogger.Debugf("Cluster Name : [%s] / NodegroupName : [%s]", clusterName, nodegroupName)
		input := &vnks.DescribeNodegroupInput{
			ClusterName:   ncloud.String(clusterName),
			NodegroupName: ncloud.String(nodegroupName),
		}

		err := nvch.ClusterClient.WaitUntilNodegroupActive(input)
		if err != nil {
			cblogger.Errorf("failed to wait until Nodegroup Active : %v", err)
			return err
		}
		cblogger.Debug("=========WaitUntilNodegroupActive() 종료")
		return nil
	}

// Cluster가 Activty 상태일때까지 대기함.

	func (nvch *NcpVpcClusterHandler) WaitUntilClusterActive(clusterName string) error {
		cblogger.Debugf("Cluster Name : [%s]", clusterName)
		input := &vnks.DescribeClusterInput{
			Name: ncloud.String(clusterName),
		}

		err := nvch.ClusterClient.WaitUntilClusterActive(input)
		if err != nil {
			cblogger.Errorf("failed to wait until cluster Active: %v", err)
			return err
		}
		cblogger.Debug("=========WaitUntilClusterActive() ended")
		return nil
	}
*/
func (nvch *NcpVpcClusterHandler) ListCluster() ([]*irs.ClusterInfo, error) {
	return nil, nil
	/*
	   	if ClusterHandler == nil {
	   		cblogger.Error("ClusterHandlerIs nil")
	   		return nil, errors.New("ClusterHandler is nil")

	   }

	   cblogger.Debug(ClusterHandler)

	   	if nvch.Client == nil {
	   		cblogger.Error(" nvch.Client Is nil")
	   		return nil, errors.New("ClusterHandler is nil")
	   	}

	   input := &vnks.ListClustersInput{}
	   // logger for HisCall
	   callogger := call.GetLogger("HISCALL")

	   	callLogInfo := call.CLOUDLOGSCHEMA{
	   		CloudOS:      call.AWS,
	   		RegionZone:   nvch.Region.Zone,
	   		ResourceType: call.CLUSTER,
	   		ResourceName: "List()",
	   		CloudOSAPI:   "ListClusters()",
	   		ElapsedTime:  "",
	   		ErrorMSG:     "",
	   	}

	   callLogStart := call.Start()

	   result, err := nvch.ClusterClient.ListClusters(input)
	   callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	   	if err != nil {
	   		callLogInfo.ErrorMSG = err.Error()
	   		callogger.Info(call.String(callLogInfo))
	   		cblogger.Error(err.Error())
	   		return nil, err
	   	}

	   callogger.Info(call.String(callLogInfo))

	   cblogger.Debug(result)

	   clusterList := []*irs.ClusterInfo{}
	   for _, clusterName := range result.Clusters {

	   	clusterInfo, err := nvch.GetCluster(irs.IID{SystemId: *clusterName})
	   	if err != nil {
	   		cblogger.Error(err)
	   		continue //	에러가 나면 일단 skip시킴.
	   	}
	   	clusterList = append(clusterList, &clusterInfo)

	   }
	   return clusterList, nil
	*/
}

func (nvch *NcpVpcClusterHandler) GetCluster(clusterIID irs.IID) (irs.ClusterInfo, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NCPVPC Cloud Driver: called GetCluster()")
	emptyClusterInfo := irs.ClusterInfo{}
	hiscallInfo := GetCallLogScheme(nvch.RegionInfo.Region, call.CLUSTER, clusterIID.SystemId, "GetCluster()")
	start := call.Start()

	var getErr error
	defer func() {
		if getErr != nil {
			cblogger.Error(getErr)
			LoggingError(hiscallInfo, getErr)
		}
	}()

	clusterList, err := ncpvpcClustersGet(nvch.ClusterClient, nvch.Ctx)
	if err != nil {
		getErr = fmt.Errorf("failed to list clusters: %v", err)
		return emptyClusterInfo, getErr
	}

	var targetCluster *vnks.Cluster
	for _, c := range clusterList {
		if ncloud.StringValue(c.Uuid) == clusterIID.SystemId {
			targetCluster = c
			break
		}
	}
	if targetCluster == nil {
		getErr = fmt.Errorf("cluster with SystemId %s not found", clusterIID.SystemId)
		return emptyClusterInfo, getErr
	}

	var createdTime time.Time
	if targetCluster.CreatedAt != nil && *targetCluster.CreatedAt != "" {
		parsedTime, err := time.Parse(time.RFC3339, *targetCluster.CreatedAt)
		if err != nil {
			getErr = fmt.Errorf("failed to parse CreatedAt (%s): %v", *targetCluster.CreatedAt, err)
			return emptyClusterInfo, getErr
		}
		createdTime = parsedTime
	}

	// ACG(Access Control Group)를 SecurityGroupIIDs로 매핑
	securityGroupIIDs := []irs.IID{}
	if targetCluster.AcgNo != nil {
		securityGroupIIDs = append(securityGroupIIDs, irs.IID{
			SystemId: fmt.Sprintf("%d", ncloud.Int32Value(targetCluster.AcgNo)),
			NameId:   ncloud.StringValue(targetCluster.AcgName),
		})
	}

	// KeyValueList에 부가 정보 추가
	keyValueList := []irs.KeyValue{
		{Key: "Status", Value: ncloud.StringValue(targetCluster.Status)},
		{Key: "Uuid", Value: ncloud.StringValue(targetCluster.Uuid)},
		{Key: "VpcNo", Value: fmt.Sprintf("%d", ncloud.Int32Value(targetCluster.VpcNo))},
	}
	if targetCluster.Endpoint != nil {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "Endpoint", Value: ncloud.StringValue(targetCluster.Endpoint)})
	}
	if targetCluster.K8sVersion != nil {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "K8sVersion", Value: ncloud.StringValue(targetCluster.K8sVersion)})
	}
	if targetCluster.AcgName != nil {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "AcgName", Value: ncloud.StringValue(targetCluster.AcgName)})
	}
	if targetCluster.AcgNo != nil {
		keyValueList = append(keyValueList, irs.KeyValue{Key: "AcgNo", Value: fmt.Sprintf("%d", ncloud.Int32Value(targetCluster.AcgNo))})
	}

	clusterInfo := irs.ClusterInfo{
		IId: irs.IID{
			NameId:   ncloud.StringValue(targetCluster.Name),
			SystemId: ncloud.StringValue(targetCluster.Uuid),
		},
		Version:     ncloud.StringValue(targetCluster.K8sVersion),
		CreatedTime: createdTime,
		Status:      irs.ClusterStatus(ncloud.StringValue(targetCluster.Status)),
		AccessInfo: irs.AccessInfo{
			Endpoint:   ncloud.StringValue(targetCluster.Endpoint),
			Kubeconfig: "Kubeconfig is not provided for NCP.", // NCP는 kubeconfig 미제공
		},
		Network: irs.NetworkInfo{
			VpcIID: irs.IID{
				NameId:   ncloud.StringValue(targetCluster.VpcName),
				SystemId: fmt.Sprintf("%d", ncloud.Int32Value(targetCluster.VpcNo)),
			},
			SubnetIIDs: func() []irs.IID {
				var list []irs.IID
				for _, sn := range targetCluster.SubnetNoList {
					list = append(list, irs.IID{SystemId: fmt.Sprintf("%d", ncloud.Int32Value(sn))})
				}
				return list
			}(),
			SecurityGroupIIDs: securityGroupIIDs,
		},
		KeyValueList: keyValueList,
	}

	// NodePool 상세 정보 KeyValueList 포함 (NHN/GCP 스타일)
	for _, np := range targetCluster.NodePool {
		onAutoScaling := false
		minNodeSize := 0
		maxNodeSize := 0
		if np.Autoscale != nil {
			onAutoScaling = ncloud.BoolValue(np.Autoscale.Enabled)
			minNodeSize = int(ncloud.Int32Value(np.Autoscale.Min))
			maxNodeSize = int(ncloud.Int32Value(np.Autoscale.Max))
		}

		nodeGroupKeyValueList := []irs.KeyValue{
			{Key: "InstanceNo", Value: fmt.Sprintf("%d", ncloud.Int32Value(np.InstanceNo))},
			{Key: "Status", Value: ncloud.StringValue(np.Status)},
			{Key: "ServerSpecCode", Value: ncloud.StringValue(np.ServerSpecCode)},
			{Key: "SoftwareCode", Value: ncloud.StringValue(np.SoftwareCode)},
		}
		if np.Autoscale != nil {
			nodeGroupKeyValueList = append(nodeGroupKeyValueList,
				irs.KeyValue{Key: "AutoScalingEnabled", Value: fmt.Sprintf("%v", ncloud.BoolValue(np.Autoscale.Enabled))},
				irs.KeyValue{Key: "AutoScalingMin", Value: fmt.Sprintf("%d", ncloud.Int32Value(np.Autoscale.Min))},
				irs.KeyValue{Key: "AutoScalingMax", Value: fmt.Sprintf("%d", ncloud.Int32Value(np.Autoscale.Max))},
			)
		}

		nodeGroupInfo := irs.NodeGroupInfo{
			IId: irs.IID{
				NameId:   ncloud.StringValue(np.Name),
				SystemId: fmt.Sprintf("%d", ncloud.Int32Value(np.InstanceNo)),
			},
			VMSpecName:      ncloud.StringValue(np.ServerSpecCode),
			ImageIID:        irs.IID{NameId: ncloud.StringValue(np.SoftwareCode)},
			DesiredNodeSize: int(ncloud.Int32Value(np.NodeCount)),
			MinNodeSize:     minNodeSize,
			MaxNodeSize:     maxNodeSize,
			RootDiskSize:    fmt.Sprintf("%d", ncloud.Int32Value(np.StorageSize)),
			KeyPairIID:      irs.IID{NameId: ncloud.StringValue(targetCluster.LoginKeyName)},
			OnAutoScaling:   onAutoScaling,
			Status:          irs.NodeGroupStatus(ncloud.StringValue(np.Status)),
			KeyValueList:    nodeGroupKeyValueList,
		}
		clusterInfo.NodeGroupList = append(clusterInfo.NodeGroupList, nodeGroupInfo)
	}

	// NCP 정책상 NodeGroup의 실제 노드 목록 및 컨테이너(파드) 목록 반환은 미지원

	LoggingInfo(hiscallInfo, start)
	cblogger.Debug(clusterInfo)
	return clusterInfo, nil
}

/*
func getKubeConfig(clusterDesc *vnks.DescribeClusterOutput) string {

	cluster := clusterDesc.Cluster

	kubeconfigContent := fmt.Sprintf(`apiVersion: v1
clusters:
- cluster:
    server: %s
    certificate-authority-data: %s
  name: kubernetes
contexts:
- context:
    cluster: kubernetes
    user: aws
  name: aws
current-context: aws
kind: Config
preferences: {}
users:
- name: aws
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1beta1
      command: aws
      args:
        - "eks"
        - "get-token"
        - "--cluster-name"
        - "%s"
`, *cluster.Endpoint, *cluster.CertificateAuthority.Data, *cluster.Name)

	return kubeconfigContent
}
*/

func (nvch *NcpVpcClusterHandler) DeleteCluster(clusterIID irs.IID) (bool, error) {
	cblogger.Infof("NCPVPC Cloud Driver: called DeleteCluster()")

	if clusterIID.SystemId == "" {
		return false, fmt.Errorf("DeleteCluster: SystemId is required")
	}

	hiscallInfo := GetCallLogScheme(nvch.RegionInfo.Region, call.CLUSTER, clusterIID.SystemId, "DeleteCluster()")
	start := call.Start()

	err := nvch.deleteCluster(clusterIID.SystemId)
	if err != nil {
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, fmt.Errorf("DeleteCluster: failed to delete cluster (SystemId=%s): %v", clusterIID.SystemId, err)
	}

	LoggingInfo(hiscallInfo, start)
	cblogger.Infof("Cluster(SystemId=%s) deleted successfully.", clusterIID.SystemId)
	return true, nil
}

// ------ NodeGroup Management

/*
Cluster.NetworkInfo 설정과 동일 서브넷으로 설정
NodeGroup 추가시에는 대상 Cluster 정보 획득하여 설정
NodeGroup에 다른 Subnet 설정이 꼭 필요시 추후 재논의
//https://github.com/cloud-barista/cb-spider/wiki/Provider-Managed-Kubernetes-and-Driver-API
*/
func (nvch *NcpVpcClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (irs.NodeGroupInfo, error) {
	return irs.NodeGroupInfo{}, fmt.Errorf("the MaxNodeSize value must be greater than or equal to 1")
	/*
		// validation check
		if nodeGroupReqInfo.MaxNodeSize < 1 { // nodeGroupReqInfo.MaxNodeSize 는 최소가 1이다.
			return irs.NodeGroupInfo{}, fmt.Errorf("The MaxNodeSize value must be greater than or equal to 1.")
		}

		clusterInfo, err := nvch.GetCluster(clusterIID)
		if err != nil {
			cblogger.Error(err)
			return irs.NodeGroupInfo{}, err
		}

		networkInfo := clusterInfo.Network
		var subnetList []*string
		for _, subnet := range networkInfo.SubnetIIDs {
			subnetId := subnet.SystemId // 포인터라서 subnet.SystemId를 직접 Append하면 안 됨.
			subnetList = append(subnetList, &subnetId)
		}

		cblogger.Debug("Final Subnet List")
		// 이 부분에서 VPC subnet ID 를 바탕으로 리스트 순회하며 ModifySubnetAttribute 를 통해 Auto-assign public IPv4 address를 활성화
		for _, subnetIdPtr := range subnetList {
			input := &vserver.ModifySubnetAttributeInput{
				MapPublicIpOnLaunch: &vserver.AttributeBooleanValue{
					Value: ncloud.Bool(true),
				},
				SubnetId: subnetIdPtr,
			}
			_, err := nvch.VMClient.ModifySubnetAttribute(input)
			if err != nil {
				errmsg := "error during ModifySubnetAttribute to MapPublicIpOnLaunch=TRUE on subnet : " + *subnetIdPtr
				cblogger.Error(err)
				cblogger.Error(errmsg)
				// return irs.NodeGroupInfo{}, errors.New(errmsg) // 서브넷 순회 이므로 나머지 서브넷은 진행하도록 주석처리함.
			}
		}

		cblogger.Debug("Subnet list")
		cblogger.Debug(subnetList)

		var nodeSecurityGroupList []*string
		for _, securityGroup := range networkInfo.SecurityGroupIIDs {
			nodeSecurityGroupList = append(nodeSecurityGroupList, &securityGroup.SystemId)
		}

		tags := map[string]string{}
		tags["key"] = NODEGROUP_TAG
		tags["value"] = nodeGroupReqInfo.IId.NameId

		input := &vnks.CreateNodegroupInput{
			//AmiType: "", // Valid Values: AL2_x86_64 | AL2_x86_64_GPU | AL2_ARM_64 | CUSTOM | BOTTLEROCKET_ARM_64 | BOTTLEROCKET_x86_64, Required: No
			//CapacityType: ncloud.String("ON_DEMAND"),//Valid Values: ON_DEMAND | SPOT, Required: No

			//ClusterName:   ncloud.String("cb-eks-cluster"),              //uri, required
			ClusterName:   ncloud.String(clusterIID.SystemId),         //uri, required
			NodegroupName: ncloud.String(nodeGroupReqInfo.IId.NameId), // required
			Tags:          ncloud.StringMap(tags),
			//NodeRole:      ncloud.String(eksRoleName), // roleName, required
			//NodeRole: roleArn,
			ScalingConfig: &vnks.NodegroupScalingConfig{
				DesiredSize: ncloud.Int64(int64(nodeGroupReqInfo.DesiredNodeSize)),
				MaxSize:     ncloud.Int64(int64(nodeGroupReqInfo.MaxNodeSize)),
				MinSize:     ncloud.Int64(int64(nodeGroupReqInfo.MinNodeSize)),
			},
			Subnets: subnetList,

			//DiskSize: 0,
			//InstanceTypes: ["",""],
			//Labels : {"key": "value"},
			//LaunchTemplate: {
			//	Id: "",
			//	Name: "",
			//	Version: ""
			//},

			//ReleaseVersion: "",
			RemoteAccess: &vnks.RemoteAccessConfig{
				Ec2SshKey:            &nodeGroupReqInfo.KeyPairIID.SystemId,
				SourceSecurityGroups: nodeSecurityGroupList,
			},

			//Taints: [{
			//	Effect:"",
			//	Key : "",
			//	Value :""
			//}],
			//UpdateConfig: {
			//	MaxUnavailable: 0,
			//	MaxUnavailablePercentage: 0
			//},
			//Version: ""
		}

		// 필수 외에 넣을 항목들 set
		rootDiskSize, _ := strconv.ParseInt(nodeGroupReqInfo.RootDiskSize, 10, 64)
		if rootDiskSize > 0 {
			input.DiskSize = ncloud.Int64(rootDiskSize)
		}

		if !strings.EqualFold(nodeGroupReqInfo.VMSpecName, "") {
			var nodeSpec []string
			nodeSpec = append(nodeSpec, nodeGroupReqInfo.VMSpecName) //"p2.xlarge"
			input.InstanceTypes = ncloud.StringSlice(nodeSpec)
		}

		cblogger.Debug(input)

		result, err := nvch.ClusterClient.CreateNodegroup(input) // 비동기
		if err != nil {
			cblogger.Error(err)
			return irs.NodeGroupInfo{}, err
		}

		cblogger.Debug(result)

		nodegroupName := result.Nodegroup.NodegroupName

		// Sync Call에서 Async Call로 변경 - 이슈:#716
		//노드 그룹이 활성화될 때까지 대기
		errWait := nvch.WaitUntilNodegroupActive(clusterIID.SystemId, *nodegroupName)
		if errWait != nil {
			cblogger.Error(errWait)
			return irs.NodeGroupInfo{}, errWait
		}

		nodeGroup, err := nvch.GetNodeGroup(clusterIID, irs.IID{NameId: nodeGroupReqInfo.IId.NameId, SystemId: *nodegroupName})
		if err != nil {
			cblogger.Error(err)
			return irs.NodeGroupInfo{}, err
		}
		nodeGroup.IId.NameId = nodeGroupReqInfo.IId.NameId
		return nodeGroup, nil
	*/
}

func (nvch *NcpVpcClusterHandler) ListNodeGroup(clusterIID irs.IID) ([]*irs.NodeGroupInfo, error) {
	return nil, nil
	/*
		input := &vnks.ListNodegroupsInput{
			ClusterName: ncloud.String(clusterIID.SystemId),
		}
		cblogger.Debug(input)

		result, err := nvch.ClusterClient.ListNodegroups(input)
		if err != nil {
			cblogger.Error(err)
			return nil, err
		}
		cblogger.Debug(result)
		nodeGroupInfoList := []*irs.NodeGroupInfo{}
		for _, nodeGroupName := range result.Nodegroups {
			nodeGroupInfo, err := nvch.GetNodeGroup(clusterIID, irs.IID{SystemId: *nodeGroupName})
			if err != nil {
				cblogger.Error(err)
				//return nil, err
				continue
			}
			nodeGroupInfoList = append(nodeGroupInfoList, &nodeGroupInfo)
		}
		return nodeGroupInfoList, nil
	*/
}

func (nvch *NcpVpcClusterHandler) GetNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (irs.NodeGroupInfo, error) {
	cblogger.Debugf("Cluster SystemId : [%s] / NodeGroup SystemId : [%s]", clusterIID.SystemId, nodeGroupIID.SystemId)

	return irs.NodeGroupInfo{}, nil
	/*
		input := &vnks.DescribeNodegroupInput{
			//AmiType: "", // Valid Values: AL2_x86_64 | AL2_x86_64_GPU | AL2_ARM_64 | CUSTOM | BOTTLEROCKET_ARM_64 | BOTTLEROCKET_x86_64, Required: No
			//CapacityType: ncloud.String("ON_DEMAND"),//Valid Values: ON_DEMAND | SPOT, Required: No
			ClusterName:   ncloud.String(clusterIID.SystemId),   //required
			NodegroupName: ncloud.String(nodeGroupIID.SystemId), // required
		}

		result, err := nvch.ClusterClient.DescribeNodegroup(input)
		cblogger.Debug("===> Node Group Invocation Result")
		cblogger.Debug(result)
		if err != nil {
			cblogger.Error(err)
			return irs.NodeGroupInfo{}, err
		}

		nodeGroupInfo, err := nvch.convertNodeGroup(result)
		if err != nil {
			cblogger.Error(err)
			return irs.NodeGroupInfo{}, err
		}
		return nodeGroupInfo, nil
	*/
}

func (nvch *NcpVpcClusterHandler) GetAutoScalingGroups(autoScalingGroupName string) ([]irs.IID, error) {
	return nil, nil
	/*
		input := &vautoscaling.DescribeAutoScalingGroupsInput{
			AutoScalingGroupNames: []*string{
				ncloud.String(autoScalingGroupName),
			},
		}

		result, err := nvch.ASClient.DescribeAutoScalingGroups(input)
		cblogger.Debug(result)

		if err != nil {
			cblogger.Error(err)
			cblogger.Error(err.Error())
			return nil, err
		}

		//nodeList := []*irs.IID{}
		nodeList := []irs.IID{}
		//AutoScalingGroups
		if len(result.AutoScalingGroups) > 0 {
			for _, curGroup := range result.AutoScalingGroups[0].Instances {
				cblogger.Debugf("   ====> [%s]", *curGroup.InstanceId)
				nodeList = append(nodeList, irs.IID{SystemId: *curGroup.InstanceId})
			}
		}

		cblogger.Debug("**VM Instance List**")
		cblogger.Debug(nodeList)
		return nodeList, nil
	*/
}

// AutoScaling 이라는 별도의 메뉴가 있음.
func (nvch *NcpVpcClusterHandler) SetNodeGroupAutoScaling(clusterIID irs.IID, nodeGroupIID irs.IID, on bool) (bool, error) {
	return false, nil
}

func (nvch *NcpVpcClusterHandler) ChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID,
	DesiredNodeSize int, MinNodeSize int, MaxNodeSize int) (irs.NodeGroupInfo, error) {
	cblogger.Infof("Cluster SystemId : [%s] / NodeGroup SystemId : [%s] / DesiredNodeSize : [%d] / MinNodeSize : [%d] / MaxNodeSize : [%d]", clusterIID.SystemId, nodeGroupIID.SystemId, DesiredNodeSize, MinNodeSize, MaxNodeSize)

	return irs.NodeGroupInfo{}, nil
	/*
		// clusterIID로 cluster 정보를 조회
		// nodeGroupIID로 nodeGroup 정보를 조회
		// 		nodeGroup에 AutoScaling 그룹 이름이 있음.

		// TODO : 공통으로 뺄 것
		input := &vnks.DescribeNodegroupInput{
			ClusterName:   ncloud.String(clusterIID.SystemId),   //required
			NodegroupName: ncloud.String(nodeGroupIID.SystemId), // required
		}

		result, err := nvch.ClusterClient.DescribeNodegroup(input)
		cblogger.Debug(result.Nodegroup)
		if err != nil {
			cblogger.Error(err)
			return irs.NodeGroupInfo{}, err
		}

		nodeGroupName := result.Nodegroup.NodegroupName
		nodeGroupResources := result.Nodegroup.Resources.AutoScalingGroups
		for _, autoScalingGroup := range nodeGroupResources {
			input := &vautoscaling.UpdateAutoScalingGroupInput{
				AutoScalingGroupName: ncloud.String(*autoScalingGroup.Name),

				MaxSize:         ncloud.Int64(int64(MaxNodeSize)),
				MinSize:         ncloud.Int64(int64(MinNodeSize)),
				DesiredCapacity: ncloud.Int64(int64(DesiredNodeSize)),
			}

			updateResult, err := nvch.ASClient.UpdateAutoScalingGroup(input)
			if err != nil {
				cblogger.Error(err)
				return irs.NodeGroupInfo{}, err
			}
			cblogger.Debug(updateResult)
		}

		nodeGroupInfo, err := nvch.GetNodeGroup(clusterIID, irs.IID{SystemId: *nodeGroupName})
		if err != nil {
			cblogger.Error(err)
			return irs.NodeGroupInfo{}, err
		}
		return nodeGroupInfo, nil
	*/
}

func (nvch *NcpVpcClusterHandler) RemoveNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (bool, error) {
	cblogger.Infof("Cluster SystemId : [%s] / NodeGroup SystemId : [%s]", clusterIID.SystemId, nodeGroupIID.SystemId)
	return false, nil
	/*
		input := &vnks.DeleteNodegroupInput{
			ClusterName:   ncloud.String(clusterIID.SystemId),   //required
			NodegroupName: ncloud.String(nodeGroupIID.SystemId), // required
		}

		result, err := nvch.ClusterClient.DeleteNodegroup(input)
		if err != nil {
			cblogger.Error(err)
			return false, err
		}

		cblogger.Debug(result)

		return true, nil
	*/
}

// ------ Upgrade K8S
func (nvch *NcpVpcClusterHandler) UpgradeCluster(clusterIID irs.IID, newVersion string) (irs.ClusterInfo, error) {
	cblogger.Infof("Cluster SystemId : [%s] / Request New Version : [%s]", clusterIID.SystemId, newVersion)
	return irs.ClusterInfo{}, nil

	/*
		// -- version 만 update인 경우
		input := &vnks.UpdateClusterVersionInput{
			Name:    ncloud.String(clusterIID.SystemId),
			Version: ncloud.String(newVersion),
		}

		result, err := nvch.ClusterClient.UpdateClusterVersion(input)
		if err != nil {
			cblogger.Error(err)
		}
		cblogger.Debug(result)
		// getClusterInfo
		return irs.ClusterInfo{}, nil
	*/
}

/*
func (nvch *NcpVpcClusterHandler) getRole(role irs.IID) (*iam.GetRoleOutput, error) {
	input := &iam.GetRoleInput{
		RoleName: ncloud.String(role.SystemId),
	}

	result, err := nvch.Iam.GetRole(input)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case iam.ErrCodeNoSuchEntityException:
				cblogger.Error(iam.ErrCodeNoSuchEntityException, aerr.Error())
			case iam.ErrCodeServiceFailureException:
				cblogger.Error(iam.ErrCodeServiceFailureException, aerr.Error())
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

	return result, nil
}
*/

/*
// EKS의 NodeGroup정보를 Spider의 NodeGroup으로 변경
func (nvch *NcpVpcClusterHandler) convertNodeGroup(nodeGroupOutput *vnks.DescribeNodegroupOutput) (irs.NodeGroupInfo, error) {
		nodeGroup := nodeGroupOutput.Nodegroup
		//nodeRole := nodeGroup.NodeRole
		//version := nodeGroup.Version
		//releaseVersion := nodeGroup.ReleaseVersion

		//subnetList := nodeGroup.Subnets
		//nodeGroupStatus := nodeGroup.Status
		nodeGroupInfo.Status = irs.NodeGroupStatus(*nodeGroup.Status)
		instanceTypeList := nodeGroup.InstanceTypes // spec

		//nodes := nodeGroup.Health.Issues[0].ResourceIds // 문제 있는 node들만 있는것이 아닌지..
		rootDiskSize := nodeGroup.DiskSize
		//nodeGroup.Taints// 미사용
		nodeGroupTagList := nodeGroup.Tags
		scalingConfig := nodeGroup.ScalingConfig
		//nodeGroup.RemoteAccess
		nodeGroupName := nodeGroup.NodegroupName

		//nodeGroup.LaunchTemplate //미사용
		//clusterName := nodeGroup.ClusterName
		//capacityType := nodeGroup.CapacityType // "ON_DEMAND"
		nodeGroupInfo.ImageIID.NameId = *nodeGroup.AmiType // AL2_x86_64"
		//createTime := nodeGroup.CreatedAt
		//health := nodeGroup.Health // Code, Message, ResourceIds	// ,"Health":{"Issues":[{"Code":"NodeCreationFailure","Message":"Unhealthy nodes in the kubernetes cluster",
		//labelList := nodeGroup.Labels
		//nodeGroupArn := nodeGroup.NodegroupArn
		//nodeGroupResources := nodeGroup.Resources
		//nodeGroupResources.AutoScalingGroups// 미사용
		//nodeGroupResources.RemoteAccessSecurityGroup// 미사용

		//=============
		//VM 노드 조회
		//=============
		// 오토스케일링 그룹 목록에서 VM 목록 정보 추출
		//"Resources":{"AutoScalingGroups":[{"Name":"eks-cb-eks-nodegroup-test-fec135d9-c812-8862-e3b0-7b773ce70d2e"}],

		if !reflect.ValueOf(nodeGroup.Resources).IsNil() {
			if !reflect.ValueOf(nodeGroup.Resources.AutoScalingGroups).IsNil() {
				autoscalingGroupName := *nodeGroup.Resources.AutoScalingGroups[0].Name //"eks-cb-eks-node-test02a-aws-9cc2876a-d3cb-2c25-55a8-9a19c431e716"
				cblogger.Debugf("autoscalingGroupName : [%s]", autoscalingGroupName)

				if autoscalingGroupName != "" {
					nodeList, errNodeList := nvch.GetAutoScalingGroups(autoscalingGroupName)
					if errNodeList != nil {
						return irs.NodeGroupInfo{}, errNodeList
					}

					nodeGroupInfo.Nodes = nodeList
				}
			}
		}

		nodeGroupInfo.DesiredNodeSize = int(*scalingConfig.DesiredSize)
		nodeGroupInfo.MinNodeSize = int(*scalingConfig.MinSize)
		nodeGroupInfo.MaxNodeSize = int(*scalingConfig.MaxSize)

		if nodeGroupTagList == nil {
			nodeGroupTagList[NODEGROUP_TAG] = nodeGroupName // 값이없으면 nodeGroupName이랑 같은값으로 set.
		}
		nodeGroupTag := ""
		for key, val := range nodeGroupTagList {
			if strings.EqualFold("key", NODEGROUP_TAG) {
				nodeGroupTag = *val
				break
			}
			cblogger.Debug(key, *val)
		}
		//printToJson(nodeGroupTagList)
		cblogger.Debug("nodeGroupName=", *nodeGroupName)
		cblogger.Debug("tag=", nodeGroupTagList[NODEGROUP_TAG])
		nodeGroupInfo.IId = irs.IID{
			NameId:   nodeGroupTag, // TAG에 이름
			SystemId: *nodeGroupName,
		}
		nodeGroupInfo.VMSpecName = *instanceTypeList[0]
		//nodeGroupInfo.ImageIID

		if !reflect.ValueOf(nodeGroup.RemoteAccess).IsNil() {
			if !reflect.ValueOf(nodeGroup.RemoteAccess.Ec2SshKey).IsNil() {
				nodeGroupInfo.KeyPairIID = irs.IID{
					SystemId: *nodeGroup.RemoteAccess.Ec2SshKey,
				}
			}
		}

		//nodeGroupInfo.RootDiskSize = strconv.FormatInt(*nodeGroup.DiskSize, 10)
		nodeGroupInfo.RootDiskSize = strconv.FormatInt(*rootDiskSize, 10)

		// TODO : node 목록 NodegroupArn 으로 조회해야하나??
		//nodeList := []irs.IID{}
		//if nodeList != nil {
		//	for _, nodeId := range nodes {
		//		nodeList = append(nodeList, irs.IID{NameId: "", SystemId: *nodeId})
		//	}
		//}
		//nodeGroupInfo.NodeList = nodeList
		//cblogger.Info("NodeGroup")
		//	{"Nodegroup":
		//		{"AmiType":"AL2_x86_64"
		//		,"CapacityType":"ON_DEMAND"
		//		,"ClusterName":"cb-eks-cluster"
		//		,"CreatedAt":"2022-08-05T01:51:49.673Z"
		//		,"DiskSize":20
		//		,"Health":{
		//					"Issues":[
		//							{"Code":"NodeCreationFailure"
		//							,"Message":"Unhealthy nodes in the kubernetes cluster"
		//							,"ResourceIds":["i-06ee95583f3f7de5c","i-0a283a92dcce27aa8"]}]},
		//		"InstanceTypes":["t3.medium"],
		//		"Labels":{},
		//		"LaunchTemplate":null,
		//		"ModifiedAt":"2022-08-05T02:15:14.308Z",
		//		"NodeRole":"arn:aws:iam::050864702683:role/cb-eks-nodegroup-role",
		//		"NodegroupArn":"arn:aws:eks:ap-northeast-2:050864702683:nodegroup/cb-eks-cluster/cb-eks-nodegroup-test/fec135d9-c812-8862-e3b0-7b773ce70d2e","NodegroupName":"cb-eks-nodegro
		//up-test",
		//		"ReleaseVersion":"1.22.9-20220725",
		//		"RemoteAccess":{"Ec2SshKey":"cb-webtool","SourceSecurityGroups":["sg-04607666"]},
		//		"Resources":{"AutoScalingGroups":[{"Name":"eks-cb-eks-nodegroup-test-fec135d9-c812-8862-e3b0-7b773ce70d2e"}],
		//		"RemoteAccessSecurityGroup":null},
		//		"ScalingConfig":{"DesiredSize":2,"MaxSize":2,"MinSize":2},
		//		"Status":"CREATE_FAILED",
		//		"Subnets":["subnet-262d6d7a","subnet-d0ee6fab","subnet-875a62cb","subnet-e08f5b8b"],
		//		"Tags":{},
		//		"Taints":null,
		//		"UpdateConfig":{"MaxUnavailable":1,"MaxUnavailablePercentage":null},
		//		"Version":"1.22"}}

		//nodeGroupArn
		// arn format
		//arn:partition:service:region:account-id:resource-id
		//arn:partition:service:region:account-id:resource-type/resource-id
		//arn:partition:service:region:account-id:resource-type:resource-id

		PrintToJson(nodeGroupInfo)
		//return irs.NodeGroupInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "추출 오류", nil)
		return nodeGroupInfo, nil
}
*/

func (nvch *NcpVpcClusterHandler) isValidServerImageName(imageName string) (bool, error) {
	optionsRes, err := ncpvpcOptionServerImageGet(nvch.ClusterClient, nvch.Ctx, hypervisorCodeDefault)
	if err != nil {
		return false, err
	}

	for _, optionRes := range optionsRes {
		label := ncloud.StringValue(optionRes.Label)
		if strings.EqualFold(strings.ToLower(label), strings.ToLower(imageName)) {
			return true, nil
		}
	}

	return false, fmt.Errorf("no server image with name prefix(%s)", imageName)
}

func (nvch *NcpVpcClusterHandler) getServerImageByNamePrefix(imageNamePrefix string) (string, error) {
	optionsRes, err := ncpvpcOptionServerImageGet(nvch.ClusterClient, nvch.Ctx, hypervisorCodeDefault)
	if err != nil {
		return "", err
	}

	for _, optionRes := range optionsRes {
		label := ncloud.StringValue(optionRes.Label)
		if strings.Contains(strings.ToLower(label), strings.ToLower(imageNamePrefix)) {
			return ncloud.StringValue(optionRes.Value), nil
		}
	}

	return "", fmt.Errorf("no server image with name prefix(%s)", imageNamePrefix)
}

func (nvch *NcpVpcClusterHandler) getAvailableServerImageList() ([]string, error) {
	var serverImageList []string
	optionsRes, err := ncpvpcOptionServerImageGet(nvch.ClusterClient, nvch.Ctx, hypervisorCodeDefault)
	if err != nil {
		return []string{}, err
	}

	for _, optionRes := range optionsRes {
		nameAndId := fmt.Sprintf("%s[Code=%s]", ncloud.StringValue(optionRes.Label), ncloud.StringValue(optionRes.Value))
		serverImageList = append(serverImageList, nameAndId)
	}

	return serverImageList, nil
}

/*
func (nvch *NcpVpcClusterHandler) getRoleNo(roleName string) (string, error) {
	optionsRes, err := ncpvpcOptionServerImageGet(nvch.ClusterClient, nvch.Ctx, hypervisorCodeDefault)
	if err != nil {
		return []string{}, err
	}

	for _, optionRes := range optionsRes {
		nameAndId := fmt.Sprintf("%s[Code=%s]", ncloud.StringValue(optionRes.Label), ncloud.StringValue(optionRes.Value))
		serverImageList = append(serverImageList, nameAndId)
	}

	return serverImageList, nil
}
*/

// deleteCluster: 실제 NCP 클러스터 삭제 로직만 담당하는 헬퍼 함수
func (nvch *NcpVpcClusterHandler) deleteCluster(clusterId string) error {
	err := nvch.ClusterClient.V2Api.ClustersUuidDelete(nvch.Ctx, &clusterId)
	if err != nil {
		return fmt.Errorf("failed to delete a cluster(id=%s): %v", clusterId, err)
	}
	return nil
}

func (nvch *NcpVpcClusterHandler) ListIID() ([]*irs.IID, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NHN Cloud Driver: called ListCluster()")
	hiscallInfo := GetCallLogScheme(nvch.RegionInfo.Region, call.CLUSTER, "ListCluster()", "ListIID()") // HisCall logging

	start := call.Start()

	var iidList []*irs.IID

	var listErr error
	defer func() {
		if listErr != nil {
			cblogger.Error(listErr)
			LoggingError(hiscallInfo, listErr)
		}
	}()

	clusterList, err := ncpvpcClustersGet(nvch.ClusterClient, nvch.Ctx)
	if err != nil {
		listErr = fmt.Errorf("Failed to List Cluster: %v", err)
		return nil, listErr
	}

	for _, cluster := range clusterList {
		var iid irs.IID
		iid.SystemId = ncloud.StringValue(cluster.Uuid)
		iid.NameId = ncloud.StringValue(cluster.Name)

		iidList = append(iidList, &iid)
	}

	LoggingInfo(hiscallInfo, start)

	return iidList, nil
}

func ncpvpcGetVpcListByName(vpcClient *vpc.APIClient, regionCode, vpcName string) ([]*vpc.Vpc, error) {
	emptyVpcList := make([]*vpc.Vpc, 0)
	getVpcListRequest := &vpc.GetVpcListRequest{
		RegionCode: ncloud.String(regionCode),
		VpcName:    ncloud.String(vpcName),
	}

	getVpcListResponse, err := vpcClient.V2Api.GetVpcList(getVpcListRequest)
	if err != nil {
		return emptyVpcList, err
	}

	return getVpcListResponse.VpcList, nil
}

func ncpvpcGetSubnet(vpcClient *vpc.APIClient, regionCode, zoneCode, vpcNo, subnetTypeCode, usageTypeCode string) ([]*vpc.Subnet, error) {
	emptySubnetList := make([]*vpc.Subnet, 0)
	getSubnetListRequest := &vpc.GetSubnetListRequest{
		RegionCode:     ncloud.String(regionCode),
		ZoneCode:       ncloud.String(zoneCode),
		VpcNo:          ncloud.String(vpcNo),
		SubnetTypeCode: ncloud.String(subnetTypeCode),
		UsageTypeCode:  ncloud.String(usageTypeCode),
	}

	getSubnetListResponse, err := vpcClient.V2Api.GetSubnetList(getSubnetListRequest)
	if err != nil {
		return emptySubnetList, err
	}

	return getSubnetListResponse.SubnetList, nil
}

func ncpvpcGetSubnetDetail(vpcClient *vpc.APIClient, regionCode, subnetNo string) (*vpc.Subnet, error) {
	getSubnetDetailRequest := &vpc.GetSubnetDetailRequest{
		RegionCode: ncloud.String(regionCode),
		SubnetNo:   ncloud.String(subnetNo),
	}

	getSubnetDetailResponse, err := vpcClient.V2Api.GetSubnetDetail(getSubnetDetailRequest)
	if err != nil {
		return nil, err
	}

	if len(getSubnetDetailResponse.SubnetList) < 1 {
		return nil, fmt.Errorf("no subnet(no: %s)", subnetNo)
	}

	return getSubnetDetailResponse.SubnetList[0], nil
}

func ncpvpcGetDefaultNetworkAclNo(vpcClient *vpc.APIClient, regionCode, vpcNo string) (string, error) {
	getNetworkAclListRequest := &vpc.GetNetworkAclListRequest{
		RegionCode:     ncloud.String(regionCode),
		NetworkAclName: ncloud.String(defaultNetworkAclName),
		VpcNo:          ncloud.String(vpcNo),
	}

	getNetworkAclListResponse, err := vpcClient.V2Api.GetNetworkAclList(getNetworkAclListRequest)
	if err != nil {
		return "", err
	}

	if len(getNetworkAclListResponse.NetworkAclList) < 1 {
		return "", fmt.Errorf("no network acl with %s", defaultNetworkAclName)
	}

	if strings.EqualFold(ncloud.StringValue(getNetworkAclListResponse.NetworkAclList[0].NetworkAclNo), "") {
		return "", fmt.Errorf("invalid network acl no with %s", defaultNetworkAclName)
	}

	return ncloud.StringValue(getNetworkAclListResponse.NetworkAclList[0].NetworkAclNo), nil
}

// subnetName: Allows only lowercase letters, numbers or special character "-". Start with an alphabet character. Length 3~30.
func ncpvpcCreateSubnet(vpcClient *vpc.APIClient, regionCode, zoneCode, vpcNo, subnetName, subnetRange, networkAclNo, subnetTypeCode, usageTypeCode string) (*vpc.Subnet, error) {
	createSubnetListRequest := &vpc.CreateSubnetRequest{
		RegionCode:     ncloud.String(regionCode),
		ZoneCode:       ncloud.String(zoneCode),
		VpcNo:          ncloud.String(vpcNo),
		SubnetName:     ncloud.String(subnetName),
		Subnet:         ncloud.String(subnetRange),
		NetworkAclNo:   ncloud.String(networkAclNo),
		SubnetTypeCode: ncloud.String(subnetTypeCode),
		UsageTypeCode:  ncloud.String(usageTypeCode),
	}

	createSubnetResponse, err := vpcClient.V2Api.CreateSubnet(createSubnetListRequest)
	if err != nil {
		return nil, err
	}

	if len(createSubnetResponse.SubnetList) < 1 {
		return nil, fmt.Errorf("failed to create a subnet(name: %s, range: %s)", subnetName, subnetRange)
	}

	return createSubnetResponse.SubnetList[0], nil
}

func waitUntilSubnetIsStatus(vpcClient *vpc.APIClient, regionCode, subnetNo, status string) error {
	apiCallCount := 0
	maxAPICallCount := 20

	var waitingErr error
	for {
		subnet, err := ncpvpcGetSubnetDetail(vpcClient, regionCode, subnetNo)
		if err != nil {
			maxAPICallCount = maxAPICallCount / 2
		}
		if strings.EqualFold(ncloud.StringValue(subnet.SubnetStatus.Code), status) {
			return nil
		}
		apiCallCount++
		if apiCallCount >= maxAPICallCount {
			waitingErr = fmt.Errorf("failed to get cluster: The maximum number of verification requests has been exceeded while waiting for availability of that resource")
			break
		}
		time.Sleep(5 * time.Second)
		cblogger.Infof("Wait until subnet's status is %s", status)
	}

	return waitingErr
}

func int32List(s []int32) []*int32 {
	vs := make([]*int32, 0, len(s))
	for _, v := range s {
		value := v
		vs = append(vs, &value)
	}
	return vs
}

func ncpvpcClustersPost(acCluster *vnks.APIClient, ctx context.Context, clusterName, clusterType, k8sVersion, loginKeyName, regionCode, zoneCode string, vpcNo int32, subnetNoList []int32, lbPrivateSubnetNo, lbPublicSubnetNo int32) (string, error) {
	publicNetwork := true
	clusterInputBody := &vnks.ClusterInputBody{
		Name:              ncloud.String(clusterName),
		ClusterType:       ncloud.String(clusterType),
		K8sVersion:        ncloud.String(k8sVersion),
		LoginKeyName:      ncloud.String(loginKeyName),
		RegionCode:        ncloud.String(regionCode),
		ZoneCode:          ncloud.String(zoneCode),
		PublicNetwork:     ncloud.Bool(publicNetwork),
		VpcNo:             ncloud.Int32(vpcNo),
		SubnetNoList:      int32List(subnetNoList),
		LbPrivateSubnetNo: ncloud.Int32(lbPrivateSubnetNo),
		LbPublicSubnetNo:  ncloud.Int32(lbPublicSubnetNo),
	}

	createClusterRes, err := acCluster.V2Api.ClustersPost(ctx, clusterInputBody)
	if err != nil {
		return "", err
	}

	return ncloud.StringValue(createClusterRes.Uuid), nil
}

func ncpvpcOptionVersionGet(acCluster *vnks.APIClient, ctx context.Context, hypervisorCode string) (vnks.OptionsRes, error) {
	emptyOptionsRes := make(vnks.OptionsRes, 0)
	queryParam := map[string]interface{}{
		"hypervisorCode": ncloud.String(hypervisorCode),
	}
	optionsRes, err := acCluster.V2Api.OptionVersionGet(ctx, queryParam)
	if err != nil {
		return emptyOptionsRes, err
	}

	return *optionsRes, nil
}

func ncpvpcOptionServerImageGet(acCluster *vnks.APIClient, ctx context.Context, hypervisorCode string) (vnks.OptionsRes, error) {
	emptyOptionsRes := make(vnks.OptionsRes, 0)
	queryParam := map[string]interface{}{
		"hypervisorCode": ncloud.String(hypervisorCode),
	}
	optionsRes, err := acCluster.V2Api.OptionServerImageGet(ctx, queryParam)
	if err != nil {
		return emptyOptionsRes, err
	}

	return *optionsRes, nil
}

func ncpvpcOptionServerProductCodeGet(acCluster *vnks.APIClient, ctx context.Context, hypervisorCode, softwareCode, zoneCode, zoneNo string) (vnks.OptionsResForServerProduct, error) {
	emptyOptionsResForServerProduct := make(vnks.OptionsResForServerProduct, 0)
	queryParam := map[string]interface{}{
		"hypervisorCode": ncloud.String(hypervisorCode),
		"zoneCode":       ncloud.String(zoneCode),
		"zoneNo":         ncloud.String(zoneNo),
	}
	optionsResForServerProduct, err := acCluster.V2Api.OptionServerProductCodeGet(ctx, ncloud.String(softwareCode), queryParam)
	if err != nil {
		return emptyOptionsResForServerProduct, err
	}

	return *optionsResForServerProduct, nil
}

func ncpvpcClustersGet(acCluster *vnks.APIClient, ctx context.Context) ([]*vnks.Cluster, error) {
	emptyClusterList := make([]*vnks.Cluster, 0)
	clustersRes, err := acCluster.V2Api.ClustersGet(ctx)
	if err != nil {
		return emptyClusterList, err
	}

	return clustersRes.Clusters, nil
}

type ncpRoleInput struct {
	Page         *int32  `json:"page"`
	Size         *int32  `json:"size"`
	SearchColumn *string `json:"search_column"`
	SearchWord   *string `json:"search_word"`
}

func getRoleNoWithRoleName(roleName string) string {
	roleInput := ncpRoleInput{
		SearchColumn: ncloud.String(searchColumnRoleName),
		SearchWord:   ncloud.String(roleName),
	}

	roleInput = roleInput

	// for testing
	roleNo := "99e185a0-2731-11f0-9327-246e9659184c"

	return roleNo
}

func isValidKeyOrValue(keyOrValue string) bool {
	// https://api.ncloud-docs.com/docs/nks-createcluster
	const pattern = `^[a-zA-Z0-9](?:[a-zA-Z0-9._-]*[a-zA-Z0-9])?$`
	matched, err := regexp.MatchString(pattern, keyOrValue)
	if err != nil {
		return false
	}
	return matched
}

func validateNodeGroupInfoList(nodeGroupInfoList []irs.NodeGroupInfo) error {
	if len(nodeGroupInfoList) == 0 {
		return fmt.Errorf("Node Group must be specified")
	}

	// NCP VPC의 KeyPair는 클러스터 의존, NodeGroup에 의존하지 않음
	var firstKeypairId *irs.IID
	for i, nodeGroupInfo := range nodeGroupInfoList {
		if nodeGroupInfo.IId.NameId == "" {
			return fmt.Errorf("Node Group's name is required")
		}
		if nodeGroupInfo.VMSpecName == "" {
			return fmt.Errorf("Node Group's vm spec name is required")
		}
		if i == 0 {
			if nodeGroupInfo.KeyPairIID.NameId == "" && nodeGroupInfo.KeyPairIID.SystemId == "" {
				return fmt.Errorf("Node Group's keypair is required")
			}
			firstKeypairId = &nodeGroupInfo.KeyPairIID
		} else {
			// NameId, SystemId 둘다 값이 있음
			if nodeGroupInfo.KeyPairIID.NameId != "" && nodeGroupInfo.KeyPairIID.SystemId != "" {
				if nodeGroupInfo.KeyPairIID.NameId != firstKeypairId.NameId || nodeGroupInfo.KeyPairIID.SystemId != firstKeypairId.SystemId {
					return fmt.Errorf("Node Group's keypair must all be the same")
				}
			} else if nodeGroupInfo.KeyPairIID.NameId != "" {
				if nodeGroupInfo.KeyPairIID.NameId != firstKeypairId.NameId {
					return fmt.Errorf("Node Group's keypair must all be the same")
				}
			} else if nodeGroupInfo.KeyPairIID.SystemId != "" {
				if nodeGroupInfo.KeyPairIID.SystemId != firstKeypairId.SystemId {
					return fmt.Errorf("Node Group's keypair must all be the same")
				}
			} else {
				return fmt.Errorf("Node Group's keypair must all be the same")
			}
		}

		// OnAutoScaling + MinNodeSize
		// MaxNodeSize
		// DesiredNodeSize
		if nodeGroupInfo.OnAutoScaling && nodeGroupInfo.MinNodeSize < 1 {
			return fmt.Errorf("MinNodeSize must be greater than 0 when OnAutoScaling is enabled.")
		}
		if nodeGroupInfo.MinNodeSize > 0 && !nodeGroupInfo.OnAutoScaling {
			return fmt.Errorf("If MinNodeSize is specified, OnAutoScaling must be enabled.")
		}
		if nodeGroupInfo.MinNodeSize > 0 && (nodeGroupInfo.MinNodeSize > nodeGroupInfo.MaxNodeSize) {
			return fmt.Errorf("MaxNodeSize must be greater than MinNodeSize.")
		}
		if nodeGroupInfo.MinNodeSize > 0 && (nodeGroupInfo.DesiredNodeSize < nodeGroupInfo.MinNodeSize) {
			return fmt.Errorf("DesiredNodeSize must be greater than or equal to MinNodeSize.")
		}
	}

	return nil
}

func validateAtCreateCluster(clusterInfo irs.ClusterInfo, supportedK8sVersions []string) error {
	// Check clusterInfo.IId.NameId
	if clusterInfo.IId.NameId == "" {
		return fmt.Errorf("Cluster name is required")
	}

	// Check clusterInfo.Network
	if len(clusterInfo.Network.SubnetIIDs) < 1 {
		return fmt.Errorf("At least one Subnet must be specified")
	}
	if len(clusterInfo.Network.SecurityGroupIIDs) < 1 {
		return fmt.Errorf("At least one Subnet must be specified")
	}

	// Check clusterInfo.Version
	var supported = false
	for _, version := range supportedK8sVersions {
		if strings.EqualFold(clusterInfo.Version, version) {
			supported = true
			break
		}
	}
	if supported == false {
		return fmt.Errorf("Unsupported K8s version. (Available version: " + strings.Join(supportedK8sVersions[:], ", ") + ")")
	}

	// Check clusterInfo.NodeGroupList
	err := validateNodeGroupInfoList(clusterInfo.NodeGroupList)
	if err != nil {
		return err
	}

	return nil
}

func validateAtAddNodeGroup(clusterIID irs.IID, nodeGroupInfo irs.NodeGroupInfo) error {
	//
	// Check nodeGroupInfo
	//
	err := validateNodeGroupInfoList([]irs.NodeGroupInfo{nodeGroupInfo})
	if err != nil {
		return err
	}

	return nil
}

func validateAtChangeNodeGroupScaling(minNodeSize int, maxNodeSize int) error {
	if minNodeSize < 1 {
		return fmt.Errorf("MaxNodeSize cannot be smaller than 1")
	}
	if maxNodeSize < 1 {
		return fmt.Errorf("MaxNodeSize cannot be smaller than 1")
	}

	return nil
}
