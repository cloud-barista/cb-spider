package resources

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
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
	"gopkg.in/yaml.v2"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type NcpVpcClusterHandler struct {
	CredentialInfo idrv.CredentialInfo
	RegionInfo     idrv.RegionInfo
	Ctx            context.Context
	VMClient       *vserver.APIClient
	VPCClient      *vpc.APIClient
	ClusterClient  *vnks.APIClient
	ASClient       *vas.APIClient
}

const (
	// XEN is the default value for NCP VPC
	ClusterTypeXen = "SVR.VNKS.STAND.C002.M008.NET.SSD.B050.G002"

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

// NCP Cluster Status Constants (from NCP NKS API)
// Reference: https://api.ncloud-docs.com/docs/nks
const (
	ncpClusterStatusCreating = "CREATING"
	ncpClusterStatusRunning  = "RUNNING"
	ncpClusterStatusDeleting = "DELETING"
	ncpClusterStatusReturned = "RETURNED"
)

// NCP NodePool Status Constants (from NCP NKS API)
const (
	ncpNodePoolStatusCreating = "CREATING"
	ncpNodePoolStatusRun      = "RUN"
	ncpNodePoolStatusDeleting = "DELETING"
)

// convertNCPClusterStatus converts NCP cluster status to cb-spider normalized ClusterStatus
func convertNCPClusterStatus(ncpStatus string) irs.ClusterStatus {
	switch ncpStatus {
	case ncpClusterStatusCreating:
		return irs.ClusterCreating
	case ncpClusterStatusRunning:
		return irs.ClusterActive
	case ncpClusterStatusDeleting:
		return irs.ClusterDeleting
	case ncpClusterStatusReturned:
		return irs.ClusterInactive
	default:
		return irs.ClusterInactive
	}
}

// convertNCPNodePoolStatus converts NCP nodepool status to cb-spider normalized NodeGroupStatus
func convertNCPNodePoolStatus(ncpStatus string) irs.NodeGroupStatus {
	switch ncpStatus {
	case ncpNodePoolStatusCreating:
		return irs.NodeGroupCreating
	case ncpNodePoolStatusRun:
		return irs.NodeGroupActive
	case ncpNodePoolStatusDeleting:
		return irs.NodeGroupDeleting
	default:
		return irs.NodeGroupInactive
	}
}

// ------ Cluster Management
func (nvch *NcpVpcClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (irs.ClusterInfo, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("NCP Cloud Driver: called CreateCluster()")
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
	versions, err := ncpOptionVersionGet(nvch.ClusterClient, nvch.Ctx, hypervisorCodeXen)
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
	clusterType := ClusterTypeXen
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
		// VPC 재조회
		vpc, err = vpcHandler.GetVPC(vpcIID)
		if err != nil {
			return "", fmt.Errorf("failed to get VPC after private LB subnet creation: %v", err)
		}
		availSubnets = availSubnets[1:]
	}
	if !existPublicLbSubnet {
		err = nvch.addSubnetAndWait(vpc.IId.SystemId, defaultPublicLbSubnetForK8s, availSubnets[0], subnetTypeCodePublic, usageTypeCodeLoadb)
		if err != nil {
			return "", fmt.Errorf("failed to create public LB subnet: %v", err)
		}
		// VPC 재조회
		vpc, err = vpcHandler.GetVPC(vpcIID)
		if err != nil {
			return "", fmt.Errorf("failed to get VPC after public LB subnet creation: %v", err)
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

		// Get NKS-specific ProductCode
		productCode, err := nvch.getNKSProductCode(ng.VMSpecName, softwareCode)
		if err != nil {
			return "", fmt.Errorf("failed to get NKS ProductCode for VMSpec %s: %v", ng.VMSpecName, err)
		}

		nodePool := &vnks.NodePoolDto{
			Name:           ncloud.String(ng.IId.NameId),
			NodeCount:      ncloud.Int32(int32(ng.DesiredNodeSize)),
			SoftwareCode:   ncloud.String(softwareCode),
			ServerSpecCode: ncloud.String(ng.VMSpecName),
			ProductCode:    ncloud.String(productCode),
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
	networkAclNo, err := ncpGetDefaultNetworkAclNo(nvch.VPCClient, nvch.RegionInfo.Region, vpcNo)
	if err != nil {
		err := fmt.Errorf("failed to add subnet(%s, %s): %v", subnetName, subnetRange, err)
		return err
	}

	subnet, err := ncpCreateSubnet(nvch.VPCClient, nvch.RegionInfo.Region,
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
			   cblogger.Debug("=========WaitUntilNodegroupActive() ended")
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

	cblogger.Debug("NCP Cloud Driver: called GetCluster()")
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

	clusterList, err := ncpClustersGet(nvch.ClusterClient, nvch.Ctx)
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
		Status:      convertNCPClusterStatus(ncloud.StringValue(targetCluster.Status)),
		AccessInfo: irs.AccessInfo{
			Endpoint:   ncloud.StringValue(targetCluster.Endpoint),
			Kubeconfig: nvch.getKubeConfig(targetCluster),
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
			Status:          convertNCPNodePoolStatus(ncloud.StringValue(np.Status)),
			KeyValueList:    nodeGroupKeyValueList,
		}
		clusterInfo.NodeGroupList = append(clusterInfo.NodeGroupList, nodeGroupInfo)
	}

	// NCP 정책상 NodeGroup의 실제 노드 목록 및 컨테이너(파드) 목록 반환은 미지원

	LoggingInfo(hiscallInfo, start)
	cblogger.Debug(clusterInfo)
	return clusterInfo, nil
}

// GenerateClusterToken generates a token for cluster authentication
// For NCP, returns kubeconfig with OIDC authentication configuration
// NCP NKS supports OIDC (OpenID Connect) authentication for kubectl access
func (nvch *NcpVpcClusterHandler) GenerateClusterToken(clusterIID irs.IID) (string, error) {
	cblogger.Info("call GenerateClusterToken()")

	// Get cluster info first
	clusterInfo, err := nvch.GetCluster(clusterIID)
	if err != nil {
		cblogger.Errorf("Failed to get cluster info: %v", err)
		return "", fmt.Errorf("failed to get cluster info: %w", err)
	}

	// Get cluster object for UUID
	clusterList, err := ncpClustersGet(nvch.ClusterClient, nvch.Ctx)
	if err != nil {
		cblogger.Errorf("Failed to get cluster list: %v", err)
		return "", fmt.Errorf("failed to get cluster list: %w", err)
	}

	var targetCluster *vnks.Cluster
	for _, cluster := range clusterList {
		if ncloud.StringValue(cluster.Name) == clusterInfo.IId.NameId ||
			ncloud.StringValue(cluster.Uuid) == clusterInfo.IId.SystemId {
			targetCluster = cluster
			break
		}
	}

	if targetCluster == nil || targetCluster.Uuid == nil {
		return "", fmt.Errorf("cluster not found or UUID is missing")
	}

	// Get kubeconfig with OIDC authentication
	kubeconfigWithOIDC, err := nvch.getKubeConfigWithOIDC(targetCluster)
	if err != nil {
		cblogger.Errorf("Failed to generate kubeconfig with OIDC: %v", err)
		return "", fmt.Errorf("failed to generate kubeconfig with OIDC: %w", err)
	}

	return kubeconfigWithOIDC, nil
}

// getKubeConfig retrieves the kubeconfig from NCP NKS API with OIDC authentication
// Returns kubeconfig with complete user authentication section
func (nvch *NcpVpcClusterHandler) getKubeConfig(cluster *vnks.Cluster) string {
	if cluster.Uuid == nil {
		cblogger.Warn("Cluster UUID is nil, cannot retrieve kubeconfig")
		return "Kubeconfig is not available: Cluster UUID is missing"
	}

	// Get kubeconfig with OIDC authentication
	kubeconfigWithOIDC, err := nvch.getKubeConfigWithOIDC(cluster)
	if err != nil {
		// Check if cluster is still in progress (not an error, just not ready yet)
		if strings.Contains(err.Error(), "cluster is not ready yet") {
			// Don't log as error - cluster is simply not ready yet
			cblogger.Debugf("Cluster is still being created, kubeconfig not yet available: %v", err)
			return "Kubeconfig will be available after cluster reaches RUNNING status"
		}
		// Actual error
		cblogger.Errorf("Failed to generate kubeconfig with OIDC: %v", err)
		return fmt.Sprintf("Kubeconfig is not available: %v", err)
	}

	return kubeconfigWithOIDC
}

// getKubeConfigWithOIDC generates a complete kubeconfig with IAM authentication
// NCP NKS API returns incomplete kubeconfig without 'users' section by default
// This function generates NCP IAM tokens using CB-Spider's credentials and embeds them in kubeconfig
// No external CLI tool (ncp-iam-authenticator) installation required
func (nvch *NcpVpcClusterHandler) getKubeConfigWithOIDC(cluster *vnks.Cluster) (string, error) {
	if cluster.Uuid == nil {
		return "", fmt.Errorf("cluster UUID is nil")
	}

	// 1. Get base kubeconfig from NCP
	kubeconfigRes, err := nvch.ClusterClient.V2Api.ClustersUuidKubeconfigGet(nvch.Ctx, cluster.Uuid)
	if err != nil {
		// Check if cluster is still in progress
		if strings.Contains(err.Error(), "409") || strings.Contains(err.Error(), "in progress") {
			return "", fmt.Errorf("cluster is not ready yet, please wait until cluster status is RUNNING: %w", err)
		}
		return "", fmt.Errorf("failed to get kubeconfig from NCP: %w", err)
	}

	if kubeconfigRes == nil || kubeconfigRes.Kubeconfig == nil {
		return "", fmt.Errorf("kubeconfig response is empty")
	}

	baseKubeconfig := ncloud.StringValue(kubeconfigRes.Kubeconfig)

	// 2. Add NCP IAM token-based authentication using CB-Spider's credentials
	// Token is generated directly and embedded in kubeconfig
	// Reference: https://github.com/NaverCloudPlatform/ncp-iam-authenticator
	return nvch.addIAMAuthentication(baseKubeconfig, cluster)
}

// addIAMAuthentication adds NCP IAM token-based authentication to kubeconfig
// This uses CB-Spider's existing NCP credentials to generate IAM tokens directly
// No external CLI tool (ncp-iam-authenticator) installation required
func (nvch *NcpVpcClusterHandler) addIAMAuthentication(baseKubeconfig string, cluster *vnks.Cluster) (string, error) {
	// Parse base kubeconfig YAML
	var kubeconfigMap map[string]interface{}
	if err := yaml.Unmarshal([]byte(baseKubeconfig), &kubeconfigMap); err != nil {
		return "", fmt.Errorf("failed to parse base kubeconfig: %w", err)
	}

	// Get region code from RegionInfo
	regionCode := nvch.RegionInfo.Region

	// Map region code to NCP region format (KR, SGN, JPN)
	ncpRegion := regionCode
	if strings.Contains(strings.ToLower(regionCode), "korea") || strings.Contains(strings.ToLower(regionCode), "kr") {
		ncpRegion = "KR"
	} else if strings.Contains(strings.ToLower(regionCode), "singapore") || strings.Contains(strings.ToLower(regionCode), "sgn") {
		ncpRegion = "SGN"
	} else if strings.Contains(strings.ToLower(regionCode), "japan") || strings.Contains(strings.ToLower(regionCode), "jpn") {
		ncpRegion = "JPN"
	}

	// Get cluster name for user name (use cluster name from kubeconfig clusters section)
	clusterName := ""
	if clusters, ok := kubeconfigMap["clusters"].([]interface{}); ok && len(clusters) > 0 {
		if clusterMap, ok := clusters[0].(map[string]interface{}); ok {
			if name, ok := clusterMap["name"].(string); ok {
				clusterName = name
			}
		}
	}

	// Fallback to cluster name from NCP API
	if clusterName == "" && cluster.Name != nil {
		clusterName = ncloud.StringValue(cluster.Name)
	}

	// Default user name following NCP convention
	userName := fmt.Sprintf("nks_%s_%s_%s", ncpRegion, clusterName, ncloud.StringValue(cluster.Uuid))

	// Generate NCP IAM token using CB-Spider's credentials
	accessKey := nvch.CredentialInfo.ClientId
	secretKey := nvch.CredentialInfo.ClientSecret
	clusterUUID := ncloud.StringValue(cluster.Uuid)

	token, err := generateNCPIAMToken(accessKey, secretKey, clusterUUID, ncpRegion)
	if err != nil {
		return "", fmt.Errorf("failed to generate NCP IAM token: %w", err)
	}

	cblogger.Infof("[NCP-IAM] Successfully generated IAM token for cluster %s", clusterUUID)

	// Add token-based user configuration
	iamUser := map[string]interface{}{
		"name": userName,
		"user": map[string]interface{}{
			"token": token,
		},
	}

	// Add users section to kubeconfig
	users := []interface{}{iamUser}
	kubeconfigMap["users"] = users

	// Update all contexts to use the IAM user
	// Note: yaml.v2 may create map[interface{}]interface{} instead of map[string]interface{}
	contextUpdated := false
	if contexts, ok := kubeconfigMap["contexts"].([]interface{}); ok {
		cblogger.Infof("[NCP-IAM] Found %d context(s) in kubeconfig", len(contexts))
		for i, ctx := range contexts {
			// Try map[interface{}]interface{} first (yaml.v2 default)
			if contextMap, ok := ctx.(map[interface{}]interface{}); ok {
				if contextData, ok := contextMap["context"].(map[interface{}]interface{}); ok {
					oldUser := contextData["user"]
					contextData["user"] = userName
					contextUpdated = true
					cblogger.Infof("[NCP-IAM] Updated context[%d] user from '%v' to '%s'", i, oldUser, userName)
				} else {
					cblogger.Warnf("[NCP-IAM] Context[%d] does not have 'context' field", i)
				}
			} else if contextMap, ok := ctx.(map[string]interface{}); ok {
				// Fallback to map[string]interface{}
				if contextData, ok := contextMap["context"].(map[string]interface{}); ok {
					oldUser := contextData["user"]
					contextData["user"] = userName
					contextUpdated = true
					cblogger.Infof("[NCP-IAM] Updated context[%d] user from '%v' to '%s'", i, oldUser, userName)
				}
			} else {
				cblogger.Warnf("[NCP-IAM] Context[%d] has unexpected type: %T", i, ctx)
			}
		}
	} else {
		cblogger.Warnf("[NCP-IAM] Contexts field is not an array or does not exist")
	}

	if !contextUpdated {
		cblogger.Error("[NCP-IAM] Failed to update any context user, kubeconfig may not work properly")
	}

	// Marshal back to YAML
	modifiedKubeconfig, err := yaml.Marshal(kubeconfigMap)
	if err != nil {
		return "", fmt.Errorf("failed to marshal modified kubeconfig: %w", err)
	}

	cblogger.Infof("Successfully generated kubeconfig with NCP IAM token authentication (region: %s, cluster: %s, user: %s)", ncpRegion, clusterUUID, userName)
	cblogger.Info("Note: Token is embedded in kubeconfig - no external CLI tool required")

	return string(modifiedKubeconfig), nil
}

func (nvch *NcpVpcClusterHandler) DeleteCluster(clusterIID irs.IID) (bool, error) {
	cblogger.Infof("NCP Cloud Driver: called DeleteCluster()")

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

// AddNodeGroup adds a new node group to an existing cluster.
// The node group uses the same subnet configuration as the cluster's NetworkInfo.
// When adding a node group, it retrieves the target cluster information to configure the subnet.
// Note: If different subnet settings are required for node groups, this should be discussed further.
// Reference: https://github.com/cloud-barista/cb-spider/wiki/Provider-Managed-Kubernetes-and-Driver-API
func (nvch *NcpVpcClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (irs.NodeGroupInfo, error) {
	cblogger.Infof("Cluster SystemId: [%s] / NodeGroup Name: [%s]", clusterIID.SystemId, nodeGroupReqInfo.IId.NameId)

	hiscallInfo := GetCallLogScheme(nvch.RegionInfo.Region, call.CLUSTER, nodeGroupReqInfo.IId.NameId, "AddNodeGroup()")
	start := call.Start()

	// Validation
	if nodeGroupReqInfo.IId.NameId == "" {
		addErr := fmt.Errorf("NodeGroup name is required")
		cblogger.Error(addErr)
		LoggingError(hiscallInfo, addErr)
		return irs.NodeGroupInfo{}, addErr
	}

	if nodeGroupReqInfo.DesiredNodeSize < 1 {
		addErr := fmt.Errorf("DesiredNodeSize must be greater than or equal to 1")
		cblogger.Error(addErr)
		LoggingError(hiscallInfo, addErr)
		return irs.NodeGroupInfo{}, addErr
	}

	// Get cluster info to retrieve subnet information
	clusterInfo, err := nvch.GetCluster(clusterIID)
	if err != nil {
		addErr := fmt.Errorf("failed to get cluster info: %w", err)
		cblogger.Error(addErr)
		LoggingError(hiscallInfo, addErr)
		return irs.NodeGroupInfo{}, addErr
	}

	// Use first subnet from cluster if no subnet specified
	var subnetNo int32
	if len(clusterInfo.Network.SubnetIIDs) > 0 {
		subnetNoStr := clusterInfo.Network.SubnetIIDs[0].SystemId
		subnetNoInt, err := strconv.ParseInt(subnetNoStr, 10, 32)
		if err != nil {
			addErr := fmt.Errorf("failed to parse subnet no: %w", err)
			cblogger.Error(addErr)
			LoggingError(hiscallInfo, addErr)
			return irs.NodeGroupInfo{}, addErr
		}
		subnetNo = int32(subnetNoInt)
	} else {
		addErr := fmt.Errorf("no subnets found in cluster")
		cblogger.Error(addErr)
		LoggingError(hiscallInfo, addErr)
		return irs.NodeGroupInfo{}, addErr
	}

	// Build NodePoolCreationBody
	nodePoolBody := &vnks.NodePoolCreationBody{
		Name:      ncloud.String(nodeGroupReqInfo.IId.NameId),
		NodeCount: ncloud.Int32(int32(nodeGroupReqInfo.DesiredNodeSize)),
		SubnetNo:  ncloud.Int32(subnetNo),
	}

	// Set StorageSize (required: 50~2000 GB)
	// Use RootDiskSize if provided, otherwise default to 100GB
	storageSize := int32(100) // default 100GB
	if nodeGroupReqInfo.RootDiskSize != "" && nodeGroupReqInfo.RootDiskSize != "0" {
		if parsedSize, err := strconv.ParseInt(nodeGroupReqInfo.RootDiskSize, 10, 32); err == nil {
			if parsedSize >= 50 && parsedSize <= 2000 {
				storageSize = int32(parsedSize)
			} else {
				cblogger.Warnf("RootDiskSize %d out of range (50~2000), using default 100GB", parsedSize)
			}
		}
	}
	nodePoolBody.StorageSize = ncloud.Int32(storageSize)

	// Set ServerSpecCode and ProductCode if VMSpecName is provided
	if nodeGroupReqInfo.VMSpecName != "" {
		nodePoolBody.ServerSpecCode = ncloud.String(nodeGroupReqInfo.VMSpecName)

		// Get SoftwareCode (image code) for NKS ProductCode lookup
		imageName := nodeGroupReqInfo.ImageIID.NameId
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
				addErr := fmt.Errorf("failed to get default server image: %w", err)
				cblogger.Error(addErr)
				LoggingError(hiscallInfo, addErr)
				return irs.NodeGroupInfo{}, addErr
			}
		} else {
			softwareCode, err = nvch.getServerImageByNamePrefix(imageName)
			if err != nil {
				addErr := fmt.Errorf("failed to get server image: %w", err)
				cblogger.Error(addErr)
				LoggingError(hiscallInfo, addErr)
				return irs.NodeGroupInfo{}, addErr
			}
		}

		// Get NKS-specific ProductCode
		productCode, err := nvch.getNKSProductCode(nodeGroupReqInfo.VMSpecName, softwareCode)
		if err != nil {
			addErr := fmt.Errorf("failed to get NKS ProductCode for VMSpec %s: %w", nodeGroupReqInfo.VMSpecName, err)
			cblogger.Error(addErr)
			LoggingError(hiscallInfo, addErr)
			return irs.NodeGroupInfo{}, addErr
		}
		nodePoolBody.ProductCode = ncloud.String(productCode)
		nodePoolBody.SoftwareCode = ncloud.String(softwareCode)
		cblogger.Debugf("Resolved NKS ProductCode for VMSpec %s: %s (SoftwareCode: %s)", nodeGroupReqInfo.VMSpecName, productCode, softwareCode)
	}

	// Override ProductCode/SoftwareCode if explicitly provided in KeyValueList
	for _, kv := range nodeGroupReqInfo.KeyValueList {
		if kv.Key == "ProductCode" {
			nodePoolBody.ProductCode = ncloud.String(kv.Value)
			cblogger.Debugf("ProductCode overridden from KeyValueList: %s", kv.Value)
		}
		if kv.Key == "SoftwareCode" {
			nodePoolBody.SoftwareCode = ncloud.String(kv.Value)
		}
	}

	// Set AutoScaling if enabled
	if nodeGroupReqInfo.OnAutoScaling {
		autoscaleOption := &vnks.AutoscalerUpdate{
			Enabled: ncloud.Bool(true),
			Min:     ncloud.Int32(int32(nodeGroupReqInfo.MinNodeSize)),
			Max:     ncloud.Int32(int32(nodeGroupReqInfo.MaxNodeSize)),
		}
		nodePoolBody.Autoscale = autoscaleOption
	}

	cblogger.Debug("NodePoolCreationBody: ", nodePoolBody)

	// Call NCP API to add node pool (async operation)
	result, err := nvch.ClusterClient.V2Api.ClustersUuidNodePoolPost(nvch.Ctx, nodePoolBody, ncloud.String(clusterIID.SystemId))
	if err != nil {
		addErr := fmt.Errorf("failed to add node pool: %w", err)
		cblogger.Error(addErr)
		LoggingError(hiscallInfo, addErr)
		return irs.NodeGroupInfo{}, addErr
	}

	cblogger.Debug("AddNodePool Result: ", result)
	cblogger.Infof("NodePool [%s] creation initiated (async). Use GetCluster() to check status.", nodeGroupReqInfo.IId.NameId)

	// Retrieve cluster info to get the newly created node pool
	// NCP creates node pool asynchronously, so status will be "Creating" initially
	clusterInfo, err = nvch.GetCluster(clusterIID)
	if err != nil {
		addErr := fmt.Errorf("failed to get cluster info after adding node pool: %w", err)
		cblogger.Error(addErr)
		LoggingError(hiscallInfo, addErr)
		return irs.NodeGroupInfo{}, addErr
	}

	// Find the newly created node pool from cluster info
	var nodeGroupInfo *irs.NodeGroupInfo
	for _, ng := range clusterInfo.NodeGroupList {
		if ng.IId.NameId == nodeGroupReqInfo.IId.NameId {
			nodeGroupInfo = &ng
			break
		}
	}

	if nodeGroupInfo == nil {
		addErr := fmt.Errorf("node pool [%s] not found in cluster after creation", nodeGroupReqInfo.IId.NameId)
		cblogger.Error(addErr)
		LoggingError(hiscallInfo, addErr)
		return irs.NodeGroupInfo{}, addErr
	}

	LoggingInfo(hiscallInfo, start)
	cblogger.Infof("Added NodeGroup(name=%s, id=%s) to Cluster(%s)", nodeGroupInfo.IId.NameId, nodeGroupInfo.IId.SystemId, clusterIID.SystemId)
	return *nodeGroupInfo, nil
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
	cblogger.Infof("Cluster SystemId: [%s] / NodeGroup SystemId: [%s]", clusterIID.SystemId, nodeGroupIID.SystemId)

	hiscallInfo := GetCallLogScheme(nvch.RegionInfo.Region, call.CLUSTER, nodeGroupIID.SystemId, "GetNodeGroup()")
	start := call.Start()

	// Get NodePool list for the cluster
	nodePoolRes, err := nvch.ClusterClient.V2Api.ClustersUuidNodePoolGet(nvch.Ctx, ncloud.String(clusterIID.SystemId))
	if err != nil {
		getErr := fmt.Errorf("failed to get node pool list: %w", err)
		cblogger.Error(getErr)
		LoggingError(hiscallInfo, getErr)
		return irs.NodeGroupInfo{}, getErr
	}

	if nodePoolRes.NodePool == nil || len(nodePoolRes.NodePool) == 0 {
		getErr := fmt.Errorf("no node pools found for cluster %s", clusterIID.SystemId)
		cblogger.Error(getErr)
		LoggingError(hiscallInfo, getErr)
		return irs.NodeGroupInfo{}, getErr
	}

	// Find the target node pool by comparing SystemId (InstanceNo)
	var targetNodePool *vnks.NodePool
	for _, np := range nodePoolRes.NodePool {
		if np.InstanceNo != nil && fmt.Sprintf("%d", ncloud.Int32Value(np.InstanceNo)) == nodeGroupIID.SystemId {
			targetNodePool = np
			break
		}
	}

	if targetNodePool == nil {
		getErr := fmt.Errorf("node pool with SystemId %s not found in cluster %s", nodeGroupIID.SystemId, clusterIID.SystemId)
		cblogger.Error(getErr)
		LoggingError(hiscallInfo, getErr)
		return irs.NodeGroupInfo{}, getErr
	}

	// Get cluster info for KeyPairIID
	clusterList, err := ncpClustersGet(nvch.ClusterClient, nvch.Ctx)
	if err != nil {
		getErr := fmt.Errorf("failed to list clusters: %w", err)
		cblogger.Error(getErr)
		LoggingError(hiscallInfo, getErr)
		return irs.NodeGroupInfo{}, getErr
	}

	var targetCluster *vnks.Cluster
	for _, c := range clusterList {
		if ncloud.StringValue(c.Uuid) == clusterIID.SystemId {
			targetCluster = c
			break
		}
	}
	if targetCluster == nil {
		getErr := fmt.Errorf("cluster with SystemId %s not found", clusterIID.SystemId)
		cblogger.Error(getErr)
		LoggingError(hiscallInfo, getErr)
		return irs.NodeGroupInfo{}, getErr
	}

	// Build NodeGroupInfo
	onAutoScaling := false
	minNodeSize := 0
	maxNodeSize := 0
	if targetNodePool.Autoscale != nil {
		onAutoScaling = ncloud.BoolValue(targetNodePool.Autoscale.Enabled)
		minNodeSize = int(ncloud.Int32Value(targetNodePool.Autoscale.Min))
		maxNodeSize = int(ncloud.Int32Value(targetNodePool.Autoscale.Max))
	}

	nodeGroupKeyValueList := []irs.KeyValue{
		{Key: "InstanceNo", Value: fmt.Sprintf("%d", ncloud.Int32Value(targetNodePool.InstanceNo))},
		{Key: "Status", Value: ncloud.StringValue(targetNodePool.Status)},
		{Key: "ServerSpecCode", Value: ncloud.StringValue(targetNodePool.ServerSpecCode)},
		{Key: "SoftwareCode", Value: ncloud.StringValue(targetNodePool.SoftwareCode)},
	}
	if targetNodePool.Autoscale != nil {
		nodeGroupKeyValueList = append(nodeGroupKeyValueList,
			irs.KeyValue{Key: "AutoScalingEnabled", Value: fmt.Sprintf("%v", ncloud.BoolValue(targetNodePool.Autoscale.Enabled))},
			irs.KeyValue{Key: "AutoScalingMin", Value: fmt.Sprintf("%d", ncloud.Int32Value(targetNodePool.Autoscale.Min))},
			irs.KeyValue{Key: "AutoScalingMax", Value: fmt.Sprintf("%d", ncloud.Int32Value(targetNodePool.Autoscale.Max))},
		)
	}

	nodeGroupInfo := irs.NodeGroupInfo{
		IId: irs.IID{
			NameId:   ncloud.StringValue(targetNodePool.Name),
			SystemId: fmt.Sprintf("%d", ncloud.Int32Value(targetNodePool.InstanceNo)),
		},
		DesiredNodeSize: int(ncloud.Int32Value(targetNodePool.NodeCount)),
		MinNodeSize:     minNodeSize,
		MaxNodeSize:     maxNodeSize,
		KeyPairIID:      irs.IID{NameId: ncloud.StringValue(targetCluster.LoginKeyName)},
		OnAutoScaling:   onAutoScaling,
		Status:          convertNCPNodePoolStatus(ncloud.StringValue(targetNodePool.Status)),
		KeyValueList:    nodeGroupKeyValueList,
	}

	LoggingInfo(hiscallInfo, start)
	cblogger.Debug("NodeGroupInfo: ", nodeGroupInfo)
	return nodeGroupInfo, nil
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
	cblogger.Infof("Cluster SystemId: [%s] / NodeGroup SystemId: [%s]", clusterIID.SystemId, nodeGroupIID.SystemId)

	hiscallInfo := GetCallLogScheme(nvch.RegionInfo.Region, call.CLUSTER, nodeGroupIID.SystemId, "RemoveNodeGroup()")
	start := call.Start()

	// Check if this is the default node pool (cannot be deleted)
	nodePoolRes, err := nvch.ClusterClient.V2Api.ClustersUuidNodePoolGet(nvch.Ctx, ncloud.String(clusterIID.SystemId))
	if err != nil {
		removeErr := fmt.Errorf("failed to get node pool list: %w", err)
		cblogger.Error(removeErr)
		LoggingError(hiscallInfo, removeErr)
		return false, removeErr
	}

	// Find the target node pool and check if it's default
	var targetNodePool *vnks.NodePool
	for _, np := range nodePoolRes.NodePool {
		if fmt.Sprintf("%d", ncloud.Int32Value(np.InstanceNo)) == nodeGroupIID.SystemId {
			targetNodePool = np
			break
		}
	}

	if targetNodePool == nil {
		removeErr := fmt.Errorf("node pool with SystemId %s not found in cluster %s", nodeGroupIID.SystemId, clusterIID.SystemId)
		cblogger.Error(removeErr)
		LoggingError(hiscallInfo, removeErr)
		return false, removeErr
	}

	// Check if it's a default node pool
	if targetNodePool.IsDefault != nil && ncloud.BoolValue(targetNodePool.IsDefault) {
		removeErr := fmt.Errorf("cannot delete default node pool '%s'. NCP does not allow deleting the default node pool", ncloud.StringValue(targetNodePool.Name))
		cblogger.Error(removeErr)
		LoggingError(hiscallInfo, removeErr)
		return false, removeErr
	}

	// Convert SystemId (string) to InstanceNo (int32)
	instanceNo, err := strconv.ParseInt(nodeGroupIID.SystemId, 10, 32)
	if err != nil {
		removeErr := fmt.Errorf("invalid NodeGroup SystemId (InstanceNo): %w", err)
		cblogger.Error(removeErr)
		LoggingError(hiscallInfo, removeErr)
		return false, removeErr
	}

	// Call NCP API to delete node pool
	err = nvch.ClusterClient.V2Api.ClustersUuidNodePoolInstanceNoDelete(
		nvch.Ctx,
		ncloud.String(clusterIID.SystemId),
		ncloud.String(fmt.Sprintf("%d", instanceNo)),
	)
	if err != nil {
		removeErr := fmt.Errorf("failed to remove node pool: %w", err)
		cblogger.Error(removeErr)
		LoggingError(hiscallInfo, removeErr)
		return false, removeErr
	}

	cblogger.Info("NodePool deletion initiated successfully")

	// Wait for node pool to be deleted (verify deletion)
	maxRetries := 30
	retryInterval := 10 * time.Second
	deleted := false

	for i := 0; i < maxRetries; i++ {
		time.Sleep(retryInterval)

		nodePoolRes, err := nvch.ClusterClient.V2Api.ClustersUuidNodePoolGet(nvch.Ctx, ncloud.String(clusterIID.SystemId))
		if err != nil {
			cblogger.Warnf("Failed to get node pool list (retry %d/%d): %v", i+1, maxRetries, err)
			continue
		}

		// Check if node pool still exists
		found := false
		for _, np := range nodePoolRes.NodePool {
			if fmt.Sprintf("%d", ncloud.Int32Value(np.InstanceNo)) == nodeGroupIID.SystemId {
				status := ncloud.StringValue(np.Status)
				cblogger.Infof("NodePool [%s] status: %s (retry %d/%d)", nodeGroupIID.SystemId, status, i+1, maxRetries)
				found = true
				break
			}
		}

		if !found {
			deleted = true
			cblogger.Info("NodePool successfully deleted")
			break
		}
	}

	if !deleted {
		cblogger.Warn("NodePool deletion may still be in progress (timeout)")
		// Don't return error as deletion was initiated successfully
	}

	LoggingInfo(hiscallInfo, start)
	return true, nil
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
//return irs.NodeGroupInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "Extraction error", nil)
		return nodeGroupInfo, nil
}
*/

func (nvch *NcpVpcClusterHandler) isValidServerImageName(imageName string) (bool, error) {
	optionsRes, err := ncpOptionServerImageGet(nvch.ClusterClient, nvch.Ctx, hypervisorCodeDefault)
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
	optionsRes, err := ncpOptionServerImageGet(nvch.ClusterClient, nvch.Ctx, hypervisorCodeDefault)
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
	optionsRes, err := ncpOptionServerImageGet(nvch.ClusterClient, nvch.Ctx, hypervisorCodeDefault)
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
	optionsRes, err := ncpOptionServerImageGet(nvch.ClusterClient, nvch.Ctx, hypervisorCodeDefault)
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

	clusterList, err := ncpClustersGet(nvch.ClusterClient, nvch.Ctx)
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

func ncpGetVpcListByName(vpcClient *vpc.APIClient, regionCode, vpcName string) ([]*vpc.Vpc, error) {
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

func ncpGetSubnet(vpcClient *vpc.APIClient, regionCode, zoneCode, vpcNo, subnetTypeCode, usageTypeCode string) ([]*vpc.Subnet, error) {
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

func ncpGetSubnetDetail(vpcClient *vpc.APIClient, regionCode, subnetNo string) (*vpc.Subnet, error) {
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

func ncpGetDefaultNetworkAclNo(vpcClient *vpc.APIClient, regionCode, vpcNo string) (string, error) {
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
func ncpCreateSubnet(vpcClient *vpc.APIClient, regionCode, zoneCode, vpcNo, subnetName, subnetRange, networkAclNo, subnetTypeCode, usageTypeCode string) (*vpc.Subnet, error) {
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
		subnet, err := ncpGetSubnetDetail(vpcClient, regionCode, subnetNo)
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

func ncpClustersPost(acCluster *vnks.APIClient, ctx context.Context, clusterName, clusterType, k8sVersion, loginKeyName, regionCode, zoneCode string, vpcNo int32, subnetNoList []int32, lbPrivateSubnetNo, lbPublicSubnetNo int32) (string, error) {
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

func ncpOptionVersionGet(acCluster *vnks.APIClient, ctx context.Context, hypervisorCode string) (vnks.OptionsRes, error) {
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

func ncpOptionServerImageGet(acCluster *vnks.APIClient, ctx context.Context, hypervisorCode string) (vnks.OptionsRes, error) {
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

func ncpOptionServerProductCodeGet(acCluster *vnks.APIClient, ctx context.Context, hypervisorCode, softwareCode, zoneCode, zoneNo string) (vnks.OptionsResForServerProduct, error) {
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

func ncpClustersGet(acCluster *vnks.APIClient, ctx context.Context) ([]*vnks.Cluster, error) {
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
		return fmt.Errorf("%s", "Unsupported K8s version. (Available version: "+strings.Join(supportedK8sVersions[:], ", ")+")")
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

// getNKSProductCode retrieves the NCP NKS-specific ProductCode for a given VMSpec and SoftwareCode
// NKS requires a different ProductCode format than regular VM creation.
// This function queries the NCP API to get the list of available ProductCodes for NKS clusters.
func (nvch *NcpVpcClusterHandler) getNKSProductCode(specName string, softwareCode string) (string, error) {
	if specName == "" {
		return "", fmt.Errorf("invalid specName: empty string")
	}
	if softwareCode == "" {
		return "", fmt.Errorf("invalid softwareCode: empty string")
	}

	// Get VMSpec details (CPU, Memory)
	specReq := vserver.GetServerSpecListRequest{
		RegionCode:         &nvch.RegionInfo.Region,
		ZoneCode:           &nvch.RegionInfo.Zone,
		ServerSpecCodeList: []*string{ncloud.String(specName)},
	}

	specResult, err := nvch.VMClient.V2Api.GetServerSpecList(&specReq)
	if err != nil {
		return "", fmt.Errorf("failed to get VMSpec from NCP: %w", err)
	}

	if len(specResult.ServerSpecList) < 1 {
		return "", fmt.Errorf("VMSpec %s does not exist", specName)
	}

	vmSpec := specResult.ServerSpecList[0]
	if vmSpec.CpuCount == nil || vmSpec.MemorySize == nil {
		return "", fmt.Errorf("VMSpec %s has no CPU or Memory info", specName)
	}

	cpuCount := *vmSpec.CpuCount
	memorySize := int32(*vmSpec.MemorySize / (1024 * 1024 * 1024)) // Convert bytes to GB (int32)

	// Query NCP API for available ProductCodes for NKS
	optionsRes, err := ncpOptionServerProductCodeGet(
		nvch.ClusterClient,
		nvch.Ctx,
		hypervisorCodeDefault,
		softwareCode,
		nvch.RegionInfo.Zone,
		"", // zoneNo can be empty
	)
	if err != nil {
		return "", fmt.Errorf("failed to get NKS ProductCode options: %w", err)
	}

	// Find matching ProductCode based on CPU and Memory
	for _, option := range optionsRes {
		// Check if Detail field has the ServerProduct info
		if option.Detail != nil {
			detail := option.Detail
			if detail.CpuCount != nil && detail.MemorySizeGb != nil {
				optionCpu := *detail.CpuCount
				optionMemory := *detail.MemorySizeGb

				// Match based on CPU and Memory
				if optionCpu == cpuCount && optionMemory == memorySize {
					// Use Value field which contains the ProductCode
					if option.Value != nil && *option.Value != "" {
						cblogger.Debugf("Matched NKS ProductCode for VMSpec %s (CPU:%d, Mem:%dGB): %s",
							specName, cpuCount, memorySize, *option.Value)
						return *option.Value, nil
					}
				}
			}
		}
	}

	// If no exact match found, return error with available options
	availableOptions := make([]string, 0)
	for _, option := range optionsRes {
		if option.Value != nil && *option.Value != "" && option.Detail != nil {
			detail := option.Detail
			cpu := "?"
			mem := "?"
			if detail.CpuCount != nil {
				cpu = fmt.Sprintf("%d", *detail.CpuCount)
			}
			if detail.MemorySizeGb != nil {
				mem = fmt.Sprintf("%d", *detail.MemorySizeGb)
			}
			availableOptions = append(availableOptions, fmt.Sprintf("%s (CPU:%s, Mem:%sGB)", *option.Value, cpu, mem))
		}
	}

	return "", fmt.Errorf("no matching NKS ProductCode found for VMSpec %s (CPU:%d, Mem:%dGB). Available options: %s",
		specName, cpuCount, memorySize, strings.Join(availableOptions, ", "))
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

// ============================================================================
// NCP IAM Token Generation Functions
// Based on: https://github.com/NaverCloudPlatform/ncp-iam-authenticator/blob/main/pkg/token/token.go
// ============================================================================

const (
	ncpTokenPrefix = "k8s-ncp-v1."
)

// ncpTokenClaim represents the token claim structure for NCP IAM authentication
type ncpTokenClaim struct {
	Timestamp string `json:"timestamp"`
	AccessKey string `json:"accessKey"`
	Signature string `json:"signature"`
	Path      string `json:"path"`
}

// generateNCPIAMToken generates a presigned NCP IAM token for Kubernetes authentication
// This token is used by Kubernetes API server to authenticate kubectl requests
// Reference: https://github.com/NaverCloudPlatform/ncp-iam-authenticator
func generateNCPIAMToken(accessKey, secretKey, clusterUUID, region string) (string, error) {
	// 1. Generate timestamp (milliseconds since epoch)
	timestamp := strconv.FormatInt(time.Now().UnixNano()/int64(time.Millisecond), 10)

	// 2. Build IAM path with cluster UUID
	path := getNCPIAMPath(clusterUUID, region)

	// 3. Create HMAC-SHA256 signature
	signature := makeNCPSignature("GET", path, accessKey, secretKey, timestamp)

	// 4. Build token claim
	claim := ncpTokenClaim{
		Timestamp: timestamp,
		AccessKey: accessKey,
		Signature: signature,
		Path:      path,
	}

	// 5. Marshal claim to JSON
	claimJSON, err := json.Marshal(claim)
	if err != nil {
		return "", fmt.Errorf("failed to marshal token claim: %w", err)
	}

	// 6. Base64 encode and add prefix
	token := ncpTokenPrefix + base64.StdEncoding.EncodeToString(claimJSON)

	return token, nil
}

// getNCPIAMPath constructs the IAM API path with cluster UUID parameter
// Path format: /iam/{stage}/user?clusterUuid={uuid}
// Stage is determined by region (v1 for KR, sgn-v1 for SGN, etc.)
func getNCPIAMPath(clusterUUID, region string) string {
	stage := getNCPRegionStage(region)
	return fmt.Sprintf("/iam/%s/user?clusterUuid=%s", stage, clusterUUID)
}

// getNCPRegionStage returns the API stage for the given region
// - KR, FKR, PCS*, FCS*, GCS*: v1
// - SGN: sgn-v1
// - KRS: krs-v1
// - JPN: jpn-v1
func getNCPRegionStage(region string) string {
	upperRegion := strings.ToUpper(region)

	// Default v1 for Korea regions and cloud stack regions
	if upperRegion == "" || upperRegion == "KR" {
		return "v1"
	}
	if strings.HasPrefix(upperRegion, "F") { // FKR, FCS01, etc.
		return "v1"
	}
	if strings.Contains(upperRegion, "CS") { // PCS01, FCS01, GCS01
		return "v1"
	}

	// Region-specific stages
	switch upperRegion {
	case "SGN":
		return "sgn-v1"
	case "KRS":
		return "krs-v1"
	case "JPN":
		return "jpn-v1"
	default:
		// For unknown regions, use lowercase-v1 format
		return strings.ToLower(upperRegion) + "-v1"
	}
}

// makeNCPSignature creates HMAC-SHA256 signature for NCP IAM authentication
// Signature format: HMAC-SHA256(secretKey, "METHOD SPACE URI NEWLINE TIMESTAMP NEWLINE ACCESSKEY")
func makeNCPSignature(method, uri, accessKey, secretKey, timestamp string) string {
	// Build message to sign
	// Format: "GET /iam/v1/user?clusterUuid=xxx\n1234567890\naccessKey"
	message := fmt.Sprintf("%s %s\n%s\n%s", method, uri, timestamp, accessKey)

	// Create HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(secretKey))
	mac.Write([]byte(message))

	// Return base64-encoded signature
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
