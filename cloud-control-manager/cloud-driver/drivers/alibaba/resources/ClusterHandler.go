// Alibaba Driver of CB-Spider.
// The CB-Spider is a sub-Framework of the Cloud-Barista Multi-Cloud Project.
// The CB-Spider Mission is to connect all the clouds with a single interface.
//
//      * Cloud-Barista: https://github.com/cloud-barista
//
// This is Alibaba Driver.
//
// by CB-Spider Team, 2022.09.

package resources

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
	"gopkg.in/yaml.v2"

	cs2015 "github.com/alibabacloud-go/cs-20151215/v4/client"
	ecs2014 "github.com/alibabacloud-go/ecs-20140526/v4/client"
	"github.com/alibabacloud-go/tea/tea"
	vpc2016 "github.com/alibabacloud-go/vpc-20160428/v6/client"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

	"github.com/Masterminds/semver/v3"
)

// calllogger
// 공통로거 만들기 이전까지 사용
// var once sync.Once
// var calllogger *logrus.Logger

// func init() {
// 	once.Do(func() {
// 		calllogger = call.GetLogger("HISCALL")
// 	})
// }

const (
	defaultClusterType        = "ManagedKubernetes"
	defaultClusterSpec        = "ack.pro.small" // Pro cluster: no quota limit, ~$65/month, 99.95% SLA
	defaultClusterRuntimeName = "containerd"
	defaultNodePoolImageType  = "AliyunLinux3ContainerOptimized" // ACK-optimized image (required for node pools)

	tagKeyAckAliyunCom        = "ack.aliyun.com"
	tagKeyCbSpiderPmksCluster = "CB-SPIDER:PMKS:CLUSTER"
	tagValueOwned             = "owned"

	clusterStateInitial        = "initial"
	clusterStateFailed         = "failed"
	clusterStateRunning        = "running"
	clusterStateUpdating       = "updating"
	clusterStateUpdatingFailed = "updating_failed"
	clusterStateScaling        = "scaling"
	clusterStateWaiting        = "waiting"
	clusterStateDisconnected   = "disconnected"
	clusterStateStopped        = "stopped"
	clusterStateDeleting       = "deleting"
	clusterStateDeleted        = "deleted"
	clusterStateDeletedFailed  = "deleted_failed"

	nodepoolStatusActive   = "active"
	nodepoolStatusScaling  = "scaling"
	nodepoolStatusRemoving = "removing"
	nodepoolStatusDeleting = "deleting"
	nodepoolStatusUpdating = "updating"

	eipStatusAvailable = "Available"
)

type AlibabaClusterHandler struct {
	RegionInfo     idrv.RegionInfo
	CredentialInfo idrv.CredentialInfo
	VpcClient      *vpc2016.Client
	CsClient       *cs2015.Client
	EcsClient      *ecs2014.Client
}

func (ach *AlibabaClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (irs.ClusterInfo, error) {
	cblogger.Debug("Alibaba Cloud Driver: called CreateCluster()")
	emptyClusterInfo := irs.ClusterInfo{}
	hiscallInfo := GetCallLogScheme(ach.RegionInfo, call.CLUSTER, "CreateCluster()", "CreateCluster()")
	start := call.Start()

	cblogger.Info("Create Cluster")

	//
	// Validation
	//
	err := validateAtCreateCluster(clusterReqInfo)
	if err != nil {
		err = fmt.Errorf("Failed to Create Cluster: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyClusterInfo, err
	}

	regionId := ach.RegionInfo.Region
	vpcId := clusterReqInfo.Network.VpcIID.SystemId

	//
	// Get all VSwitches in VPC
	//
	vswitches, err := aliDescribeVSwitches(ach.VpcClient, regionId, vpcId)
	if err != nil {
		err = fmt.Errorf("Failed to Create Cluster: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyClusterInfo, err
	}

	if len(vswitches) == 0 {
		err = fmt.Errorf("No VSwitch in VPC(ID=%s)", vpcId)
		err = fmt.Errorf("Failed to Create Cluster: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyClusterInfo, err
	}

	var vswitchIds []string
	for _, vs := range vswitches {
		vswitchIds = append(vswitchIds, tea.StringValue(vs.VSwitchId))
	}
	cblogger.Debugf("VSwiches in VPC(%s): %v", vpcId, vswitchIds)

	cidrList, err := ach.getAvailableCidrList()
	if err != nil {
		err = fmt.Errorf("Failed to Create Cluster: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyClusterInfo, err
	}

	if len(cidrList) < 2 {
		err = fmt.Errorf("insufficient CIDRs")
		err = fmt.Errorf("Failed to Create Cluster: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyClusterInfo, err

	}
	cblogger.Debugf("Available CIDR: ContainerCidr(%s), ServiceCidr(%s)", cidrList[0], cidrList[1])

	//
	// Create a Cluster
	//
	clusterName := clusterReqInfo.IId.NameId
	k8sVersion := clusterReqInfo.Version
	clusterType := defaultClusterType
	clusterSpec := defaultClusterSpec
	containerCidr := cidrList[0]
	serviceCidr := cidrList[1]
	secGroupId := clusterReqInfo.Network.SecurityGroupIIDs[0].SystemId
	snatEntry := true
	epPublicAccess := true

	var tagList *[]cs2015.Tag
	if clusterReqInfo.TagList != nil && len(clusterReqInfo.TagList) > 0 {

		clusterTags := []cs2015.Tag{}
		for _, clusterTag := range clusterReqInfo.TagList {
			tag0 := cs2015.Tag{
				Key:   &clusterTag.Key,
				Value: &clusterTag.Value,
			}
			clusterTags = append(clusterTags, tag0)

		}
		tagList = &clusterTags
	}

	runtimeName, runtimeVersion, err := getLatestRuntime(ach.CsClient, regionId, clusterType, k8sVersion)
	if err != nil {
		err := fmt.Errorf("Failed to Create Cluster: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyClusterInfo, err
	}
	cblogger.Debugf("Selected Runtime (Name=%s, Version=%s)", runtimeName, runtimeVersion)

	nodepools := getNodepoolsFromNodeGroupList(clusterReqInfo.NodeGroupList, runtimeName, runtimeVersion, vswitchIds)

	clusterId, err := aliCreateCluster(ach.CsClient, clusterName, regionId, clusterType, clusterSpec, k8sVersion, runtimeName, runtimeVersion, vpcId, containerCidr, serviceCidr, secGroupId, snatEntry, epPublicAccess, vswitchIds, tagKeyCbSpiderPmksCluster, tagValueOwned, tagList, nodepools)
	if err != nil {
		err := fmt.Errorf("Failed to Create Cluster: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyClusterInfo, err
	}
	cblogger.Debugf("To Create Cluster is In Progress.")

	var createErr error = nil
	defer func() {
		if createErr != nil {
			cblogger.Error(createErr)
			LoggingError(hiscallInfo, createErr)

			cleanCluster(ach.CsClient, tea.StringValue(clusterId))
			cblogger.Infof("Cluster(%s) will be Deleted.", clusterName)
		}
	}()

	//
	// Get ClusterInfo
	//
	clusterInfo, err := ach.getClusterInfo(regionId, tea.StringValue(clusterId))
	if err != nil {
		createErr = fmt.Errorf("Failed to Create Cluster: %v", err)
		return emptyClusterInfo, createErr
	}

	LoggingInfo(hiscallInfo, start)

	cblogger.Infof("Creating Cluster(Name=%s, ID=%s).", clusterInfo.IId.NameId, clusterInfo.IId.SystemId)

	return *clusterInfo, nil
}

func (ach *AlibabaClusterHandler) ListCluster() ([]*irs.ClusterInfo, error) {
	cblogger.Debug("Alibaba Cloud Driver: called ListCluster()")
	hiscallInfo := GetCallLogScheme(ach.RegionInfo, call.CLUSTER, "ListCluster()", "ListCluster()")
	start := call.Start()

	cblogger.Infof("Get Cluster List")

	//
	// Get Cluster List
	//
	regionId := ach.RegionInfo.Region
	clusters, err := aliDescribeClustersV1(ach.CsClient, regionId)
	if err != nil {
		err := fmt.Errorf("Failed to List Cluster: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return nil, err
	}

	//
	// Get ClusterInfo List
	//
	var clusterInfoList []*irs.ClusterInfo
	for _, cluster := range clusters {
		clusterInfo, err := ach.getClusterInfo(regionId, tea.StringValue(cluster.ClusterId))
		if err != nil {
			err := fmt.Errorf("Failed to List Cluster: %v", err)
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return nil, err
		}

		clusterInfoList = append(clusterInfoList, clusterInfo)
	}

	LoggingInfo(hiscallInfo, start)

	return clusterInfoList, nil
}

func (ach *AlibabaClusterHandler) GetCluster(clusterIID irs.IID) (irs.ClusterInfo, error) {
	cblogger.Debug("Alibaba Cloud Driver: called GetCluster()")
	emptyClusterInfo := irs.ClusterInfo{}
	hiscallInfo := GetCallLogScheme(ach.RegionInfo, call.CLUSTER, clusterIID.NameId, "GetCluster()")
	start := call.Start()

	cblogger.Infof("Get Cluster")

	//
	// Get ClusterInfo
	//
	regionId := ach.RegionInfo.Region
	clusterInfo, err := ach.getClusterInfo(regionId, clusterIID.SystemId)
	if err != nil {
		err := fmt.Errorf("Failed to Get Cluster: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyClusterInfo, err
	}

	LoggingInfo(hiscallInfo, start)

	cblogger.Infof("Get Cluster(Name=%s, ID=%s)", clusterInfo.IId.NameId, clusterInfo.IId.SystemId)

	return *clusterInfo, nil
}

// GenerateClusterToken generates a token for cluster authentication
// Alibaba Cloud does not support dynamic token generation yet
func (ach *AlibabaClusterHandler) GenerateClusterToken(clusterIID irs.IID) (string, error) {
	return "", fmt.Errorf("GenerateClusterToken is not supported for Alibaba Cloud clusters yet")
}

func (ach *AlibabaClusterHandler) DeleteCluster(clusterIID irs.IID) (bool, error) {
	cblogger.Debug("Alibaba Cloud Driver: called DeleteCluster()")
	hiscallInfo := GetCallLogScheme(ach.RegionInfo, call.CLUSTER, clusterIID.NameId, "DeleteCluster()")
	start := call.Start()

	cblogger.Infof("Delete Cluster")

	//
	// Get Cluster Detailed Information
	//
	cluster, err := aliDescribeClusterDetail(ach.CsClient, clusterIID.SystemId)
	if err != nil {
		err := fmt.Errorf("Failed to Delete Cluster:  %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	//
	// Check if there is a nat gateway automatically created with the cluster
	//
	cblogger.Debugf("Check if NAT Gatway Automatically Created with Cluster(%s)", clusterIID.NameId)

	regionId := ach.RegionInfo.Region
	vpcId := tea.StringValue(cluster.VpcId)
	tagKey := tagKeyAckAliyunCom
	tagValue := clusterIID.SystemId
	ngwsRetaining, err := getInternetNatGatewaysWithTagInVpc(ach.VpcClient, regionId, vpcId, tagKey, tagValue)
	if err != nil {
		err = fmt.Errorf("Failed to Delete Cluster: %v", err)
		cblogger.Error(err)
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(hiscallInfo))
		return false, err
	}

	var retainResources []string
	if len(ngwsRetaining) > 0 {
		retainResources = append(retainResources, tea.StringValue(ngwsRetaining[0].NatGatewayId))
	}
	if len(retainResources) > 0 {
		cblogger.Debugf("The NAT Gateway(IDs=%v) is retained.", retainResources)
	}

	//
	// Delete a Cluster without NAT Gateway
	//
	_, err = aliDeleteCluster(ach.CsClient, tea.StringValue(cluster.ClusterId), retainResources)
	if err != nil {
		err := fmt.Errorf("Failed to Delete Cluster: %v", err)
		cblogger.Error(err)
		hiscallInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(hiscallInfo))
		return false, err
	}
	cblogger.Debugf("To Delete Cluster is In Progress.")

	//
	// Cleanup NAT Gateway if there is no more cluster created by CB-SPIDER
	//
	cblogger.Debugf("Check if Cluster Created By CB-SPIDER Exists.")

	exist, err := existNotDeletedClusterWithTagInVpc(ach.CsClient, tea.StringValue(cluster.RegionId), tea.StringValue(cluster.VpcId), tagKeyCbSpiderPmksCluster, tagValueOwned)
	if err != nil {
		err := fmt.Errorf("Failed to Delete Cluster: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}
	if exist == false {
		cblogger.Debugf("No More Cluster Created By CB-SPIDER.")

		tagKey = tagKeyCbSpiderPmksCluster
		tagValue = tagValueOwned
		ngwsRetained, err := getInternetNatGatewaysWithTagInVpc(ach.VpcClient, regionId, vpcId, tagKey, tagValue)
		if err != nil {
			err = fmt.Errorf("Failed to Delete Cluster: %v", err)
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return false, err
		}

		for _, ngw := range ngwsRetained {
			cleanNatGatewayWithEip(ach.VpcClient, tea.StringValue(cluster.RegionId), tea.StringValue(ngw.NatGatewayId))
			cblogger.Infof("Internet NAT Gateway(ID=%s) will be deleted.", tea.StringValue(ngw.NatGatewayId))
		}
	} else {
		cblogger.Debugf("Cluster Created By CB-SPIDER Exists.")
	}

	LoggingInfo(hiscallInfo, start)

	cblogger.Info(fmt.Sprintf("Deleting Cluster(Name=%s, ID=%s).", clusterIID.NameId, clusterIID.SystemId))

	return true, nil
}

func (ach *AlibabaClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (irs.NodeGroupInfo, error) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("PANIC!!\n%v\n%v", r, string(debug.Stack()))
			cblogger.Error(err)
		}
	}()

	cblogger.Debug("Alibaba Cloud Driver: called AddNodeGroup()")
	emptyNodeGroupInfo := irs.NodeGroupInfo{}
	hiscallInfo := GetCallLogScheme(ach.RegionInfo, call.CLUSTER, clusterIID.NameId, "AddNodeGroup()")
	start := call.Start()

	cblogger.Infof("Add NodeGroup")

	//
	// Validation
	//
	err := validateAtAddNodeGroup(clusterIID, nodeGroupReqInfo)
	if err != nil {
		err = fmt.Errorf("Failed to Add Node Group: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyNodeGroupInfo, err
	}

	//
	// Get Cluster Detailed Information
	//
	regionId := ach.RegionInfo.Region
	clusterId := clusterIID.SystemId

	cluster, err := aliDescribeClusterDetail(ach.CsClient, clusterId)
	if err != nil {
		err = fmt.Errorf("Failed to Add NodeGroup: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyNodeGroupInfo, err
	}

	//
	// Get all VSwitches in VPC
	//
	vswitches, err := aliDescribeVSwitches(ach.VpcClient, regionId, tea.StringValue(cluster.VpcId))
	if err != nil {
		err = fmt.Errorf("Failed to Add NodeGroup: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyNodeGroupInfo, err
	}

	if len(vswitches) == 0 {
		err = fmt.Errorf("No VSwitch in VPC(ID=%s)", tea.StringValue(cluster.VpcId))
		err = fmt.Errorf("Failed to Add NodeGroup: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyNodeGroupInfo, err
	}

	var vswitchIds []string
	for _, vs := range vswitches {
		vswitchIds = append(vswitchIds, tea.StringValue(vs.VSwitchId))
	}
	cblogger.Debugf("VSwiches in VPC(%s): %v", tea.StringValue(cluster.VpcId), vswitchIds)

	//
	// Check availability of Instance Type
	//
	instanceType := nodeGroupReqInfo.VMSpecName
	isAvail, err := ach.isAvailableInstanceType(tea.StringValue(cluster.RegionId), tea.StringValue(cluster.ZoneId), instanceType)
	if err != nil {
		err = fmt.Errorf("Failed to Add NodeGroup: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyNodeGroupInfo, err
	}
	if isAvail == false {
		err = fmt.Errorf("InstanceType(%s) is not availale", instanceType)
		err = fmt.Errorf("Failed to Add NodeGroup: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyNodeGroupInfo, err
	}

	//
	// Validate RootDiskType availability
	//
	if nodeGroupReqInfo.RootDiskType != "" {
		// Create ECS client for disk type validation
		ecsClient, err := ecs.NewClientWithAccessKey(
			tea.StringValue(cluster.RegionId),
			ach.CredentialInfo.ClientId,
			ach.CredentialInfo.ClientSecret,
		)
		if err != nil {
			err = fmt.Errorf("Failed to create ECS client for disk validation: %v", err)
			cblogger.Error(err)
			LoggingError(hiscallInfo, err)
			return emptyNodeGroupInfo, err
		}

		// Query available disk types
		availableDiskTypes, err := GetAvailableSystemDiskTypesForCluster(
			ecsClient,
			tea.StringValue(cluster.RegionId),
			tea.StringValue(cluster.ZoneId),
			instanceType,
		)
		if err != nil {
			cblogger.Warnf("Failed to query available disk types: %v", err)
			// Continue without validation if query fails
		} else {
			// Check if requested disk type is available
			diskTypeAvailable := false
			for _, availType := range availableDiskTypes {
				if availType == nodeGroupReqInfo.RootDiskType {
					diskTypeAvailable = true
					break
				}
			}

			if !diskTypeAvailable {
				err = fmt.Errorf(
					"RootDiskType '%s' is not supported for ACK node pools in zone '%s' with instance type '%s'.\n"+
						"Note: Only the following disk types are available in this zone: %v",
					nodeGroupReqInfo.RootDiskType,
					tea.StringValue(cluster.ZoneId),
					instanceType,
					availableDiskTypes,
				)
				cblogger.Error(err)
				LoggingError(hiscallInfo, err)
				return emptyNodeGroupInfo, err
			}

			cblogger.Infof("RootDiskType '%s' is available and supported", nodeGroupReqInfo.RootDiskType)
		}
	}

	//
	// Create Node Group
	//
	name := nodeGroupReqInfo.IId.NameId
	autoScalingEnable := nodeGroupReqInfo.OnAutoScaling
	maxInstances := int64(nodeGroupReqInfo.MaxNodeSize)
	minInstances := int64(nodeGroupReqInfo.MinNodeSize)
	instanceTypes := []string{nodeGroupReqInfo.VMSpecName}
	systemDiskCategory := nodeGroupReqInfo.RootDiskType
	systemDiskSize, _ := strconv.ParseInt(nodeGroupReqInfo.RootDiskSize, 10, 64)

	// KeyPair: Alibaba uses KeyPairName as both NameId and SystemId, so use SystemId
	keyPair := nodeGroupReqInfo.KeyPairIID.SystemId

	// Image: Alibaba uses ImageId as SystemId
	imageId := nodeGroupReqInfo.ImageIID.SystemId

	imageType := ""
	if strings.EqualFold(imageId, "") || strings.EqualFold(imageId, "default") {
		imageId = ""
		imageType = defaultNodePoolImageType
		cblogger.Debugf("Using default image type: %s", imageType)
	} else {
		cblogger.Debugf("Using specified image ID: %s", imageId)
	}
	desiredSize := int64(nodeGroupReqInfo.DesiredNodeSize)

	nodepoolId, err := aliCreateClusterNodePool(ach.CsClient, clusterId, name,
		autoScalingEnable, maxInstances, minInstances, vswitchIds, instanceTypes, systemDiskCategory, systemDiskSize, keyPair, imageId, imageType, desiredSize)
	if err != nil {
		err = fmt.Errorf("Failed to Add NodeGroup: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyNodeGroupInfo, err
	}
	cblogger.Debugf("To Create NodePool is In Progress.")

	//
	// Get NodeGroupInfo
	//
	nodeGroupInfo, err := ach.getNodeGroupInfo(clusterId, tea.StringValue(nodepoolId))
	if err != nil {
		err = fmt.Errorf("Failed to Add NodeGroup: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyNodeGroupInfo, err
	}

	LoggingInfo(hiscallInfo, start)

	cblogger.Infof("Adding NodeGroup(Name=%s, ID=%s) to Cluster(%s).", nodeGroupInfo.IId.NameId, nodeGroupInfo.IId.SystemId, clusterId)

	// Log detailed nodepool status for debugging
	nodepool, err := aliDescribeClusterNodePoolDetail(ach.CsClient, clusterId, tea.StringValue(nodepoolId))
	if err == nil && nodepool != nil && nodepool.Status != nil {
		cblogger.Infof("NodePool Status: State=%s, TotalNodes=%d, FailedNodes=%d, HealthyNodes=%d",
			tea.StringValue(nodepool.Status.State),
			tea.Int64Value(nodepool.Status.TotalNodes),
			tea.Int64Value(nodepool.Status.FailedNodes),
			tea.Int64Value(nodepool.Status.HealthyNodes))

		// If there are failed nodes, log a warning
		if tea.Int64Value(nodepool.Status.FailedNodes) > 0 {
			cblogger.Warnf("⚠️  NodePool has %d failed nodes. Check Alibaba Cloud Console for detailed error messages.",
				tea.Int64Value(nodepool.Status.FailedNodes))
			cblogger.Warn("Common causes: 1) Image not available in the zone, 2) Instance type not available, 3) Insufficient quota")
		}
	}

	return *nodeGroupInfo, nil
}

func (ach *AlibabaClusterHandler) SetNodeGroupAutoScaling(clusterIID irs.IID, nodeGroupIID irs.IID, on bool) (bool, error) {
	cblogger.Debug("Alibaba Cloud Driver: called SetNodeGroupAutoScaling()")
	hiscallInfo := GetCallLogScheme(ach.RegionInfo, call.CLUSTER, clusterIID.NameId, "SetNodeGroupAutoScaling()")
	start := call.Start()

	cblogger.Infof("Set NodeGroup AutoScaling")

	//
	// Set NodeGroup AutoScaling
	//
	clusterId := clusterIID.SystemId
	ngId := nodeGroupIID.SystemId
	_, err := aliModifyClusterNodePoolAutoScalingEnable(ach.CsClient, clusterId, ngId, on)
	if err != nil {
		err := fmt.Errorf("Failed to Set NodeGroup AutoScaling: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	LoggingInfo(hiscallInfo, start)

	cblogger.Infof("Modifying AutoScaling of NodeGroup(Name=%s, ID=%s) in Cluster(%s).", nodeGroupIID.NameId, nodeGroupIID.SystemId, clusterIID.NameId)

	return true, nil
}

func (ach *AlibabaClusterHandler) ChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID, desiredNodeSize int, minNodeSize int, maxNodeSize int) (irs.NodeGroupInfo, error) {
	cblogger.Debug("Alibaba Cloud Driver: called ChangeNodeGroupScaling()")
	emptyNodeGroupInfo := irs.NodeGroupInfo{}
	hiscallInfo := GetCallLogScheme(ach.RegionInfo, call.CLUSTER, clusterIID.NameId, "ChangeNodeGroupScaling()")
	start := call.Start()

	cblogger.Infof("Change NodeGroup Scaling")

	//
	// Validation
	//
	err := validateAtChangeNodeGroupScaling(clusterIID, nodeGroupIID, minNodeSize, maxNodeSize)
	if err != nil {
		err = fmt.Errorf("Failed to Change Node Group Scaling: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyNodeGroupInfo, err
	}

	//
	// Change NodeGroup's Scaling Size
	//
	clusterId := clusterIID.SystemId
	ngId := nodeGroupIID.SystemId
	autoScalingEnable := true

	// CAUTION: desiredNodeSize cannot be applied in alibaba with auto scaling mode
	_, err = aliModifyClusterNodePoolScalingSize(ach.CsClient, clusterId, ngId, autoScalingEnable, int64(maxNodeSize), int64(minNodeSize), int64(desiredNodeSize))
	if err != nil {
		err = fmt.Errorf("Failed to Change NodeGroup Scaling: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyNodeGroupInfo, err
	}

	//
	// Get NodeGroupInfo
	//
	nodeGroupInfo, err := ach.getNodeGroupInfo(clusterId, ngId)
	if err != nil {
		err = fmt.Errorf("Failed to Change NodeGroup Scaling: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyNodeGroupInfo, err
	}

	LoggingInfo(hiscallInfo, start)

	cblogger.Infof("Modifying Scaling of NodeGroup(Name=%s, ID=%s) in Cluster(%s).", nodeGroupInfo.IId.NameId, nodeGroupInfo.IId.SystemId, clusterIID.NameId)

	return *nodeGroupInfo, nil
}

func (ach *AlibabaClusterHandler) RemoveNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (bool, error) {
	cblogger.Debug("Alibaba Cloud Driver: called RemoveNodeGroup()")
	hiscallInfo := GetCallLogScheme(ach.RegionInfo, call.CLUSTER, clusterIID.NameId, "RemoveNodeGroup()")
	start := call.Start()

	cblogger.Infof("Remove NodeGroup")

	//
	// Remove NodeGroup
	//
	clusterId := clusterIID.SystemId
	ngId := nodeGroupIID.SystemId

	_, err := aliDeleteClusterNodepool(ach.CsClient, clusterId, ngId, true)
	hiscallInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		err := fmt.Errorf("Failed to Remove NodeGroup: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return false, err
	}

	LoggingInfo(hiscallInfo, start)

	cblogger.Infof("Removing NodeGroup(Name=%s, ID=%s) to Cluster(%s).", nodeGroupIID.NameId, nodeGroupIID.SystemId, clusterIID.NameId)

	return true, nil
}

func (ach *AlibabaClusterHandler) UpgradeCluster(clusterIID irs.IID, newVersion string) (irs.ClusterInfo, error) {
	cblogger.Debug("Alibaba Cloud Driver: called UpgradeCluster()")
	emptyClusterInfo := irs.ClusterInfo{}
	hiscallInfo := GetCallLogScheme(ach.RegionInfo, call.CLUSTER, clusterIID.NameId, "UpgradeCluster()")
	start := call.Start()

	cblogger.Infof("Upgrade Cluster")

	//
	// Upgrade Cluster
	//
	regionId := ach.RegionInfo.Region
	clusterId := clusterIID.SystemId

	_, err := aliUpgradeCluster(ach.CsClient, clusterId, newVersion)
	if err != nil {
		err := fmt.Errorf("Failed to Upgrade Cluster: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyClusterInfo, err
	}

	//
	// Get ClusterInfo
	//
	clusterInfo, err := ach.getClusterInfo(regionId, clusterId)
	if err != nil {
		err = fmt.Errorf("Failed to Upgrade Cluster: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return emptyClusterInfo, err
	}

	LoggingInfo(hiscallInfo, start)

	cblogger.Infof("Upgrading Cluster(Name=%s, ID=%s)", clusterInfo.IId.NameId, clusterInfo.IId.SystemId)

	return *clusterInfo, nil
}

func (ach *AlibabaClusterHandler) getClusterInfoListWithoutNodeGroupList(regionId string) ([]*irs.ClusterInfo, error) {
	clusters, err := aliDescribeClustersV1(ach.CsClient, regionId)
	if err != nil {
		err = fmt.Errorf("failed to get ClusterInfoList: %v", err)
		return nil, err
	}

	var clusterInfoList []*irs.ClusterInfo
	for _, cluster := range clusters {
		clusterInfo, err := ach.getClusterInfoWithoutNodeGroupList(regionId, tea.StringValue(cluster.ClusterId))
		if err != nil {
			err = fmt.Errorf("failed to get ClusterInfoList: %v", err)
			return nil, err
		}

		clusterInfoList = append(clusterInfoList, clusterInfo)
	}

	return clusterInfoList, nil
}

func (ach *AlibabaClusterHandler) getClusterInfoWithoutNodeGroupList(regionId, clusterId string) (*irs.ClusterInfo, error) {
	//
	// Fill clusterInfo
	//
	cluster, err := aliDescribeClusterDetail(ach.CsClient, clusterId)
	if err != nil {
		err = fmt.Errorf("failed to get ClusterInfo: %v", err)
		return nil, err
	}

	clusterStatus := irs.ClusterInactive
	if strings.EqualFold(tea.StringValue(cluster.State), clusterStateInitial) {
		clusterStatus = irs.ClusterCreating
	} else if strings.EqualFold(tea.StringValue(cluster.State), clusterStateUpdating) {
		clusterStatus = irs.ClusterUpdating
	} else if strings.EqualFold(tea.StringValue(cluster.State), clusterStateFailed) {
		clusterStatus = irs.ClusterInactive
	} else if strings.EqualFold(tea.StringValue(cluster.State), clusterStateDeleting) {
		clusterStatus = irs.ClusterDeleting
	} else if strings.EqualFold(tea.StringValue(cluster.State), clusterStateRunning) {
		clusterStatus = irs.ClusterActive
	}

	createdTime, err := time.Parse(time.RFC3339, tea.StringValue(cluster.Created)) // 2022-09-08T09:02:16+08:00,

	// "{\"api_server_endpoint\":\"https://111.222.333.444:6443\",\"intranet_api_server_endpoint\":\"https://10.2.1.1:6443\"}"
	// if api_server_endpoint is not exist, it'll throw error
	endpoint := "Endpoint is not ready yet!"
	if cluster.MasterUrl != nil && !strings.EqualFold(tea.StringValue(cluster.MasterUrl), "") {
		jsonMasterUrl := make(map[string]interface{})
		err = json.Unmarshal([]byte(tea.StringValue(cluster.MasterUrl)), &jsonMasterUrl)
		if err != nil {
			err = fmt.Errorf("failed to get ClusterInfo: %v", err)
			return nil, err
		}

		if jsonMasterUrl["api_server_endpoint"] != nil {
			endpoint = jsonMasterUrl["api_server_endpoint"].(string)
		}
	}

	kubeconfig := "Kubeconfig is not ready yet!"
	userKubeconfig, err := aliDescribeClusterUserKubeconfig(ach.CsClient, clusterId)
	if err != nil {
		cblogger.Info("Kubeconfig is not ready yet!")
	} else {
		kubeconfig = userKubeconfig
	}

	vpcAttr, err := aliDescribeVpcAttribute(ach.VpcClient, regionId, tea.StringValue(cluster.VpcId))
	if err != nil {
		err = fmt.Errorf("failed to get ClusterInfo: %v", err)
		return nil, err
	}

	vswitchAttr, err := aliDescribeVSwitchAttributes(ach.VpcClient, regionId, tea.StringValue(cluster.VswitchId))
	if err != nil {
		err = fmt.Errorf("failed to get ClusterInfo: %v", err)
		return nil, err
	}

	secGroupAttr, err := aliDescribeSecurityGroupAttribute(ach.EcsClient, regionId, tea.StringValue(cluster.SecurityGroupId))
	if err != nil {
		err = fmt.Errorf("failed to get ClusterInfo: %v", err)
		return nil, err
	}

	clusterInfo := &irs.ClusterInfo{
		IId: irs.IID{
			NameId:   tea.StringValue(cluster.Name),
			SystemId: tea.StringValue(cluster.ClusterId),
		},
		Version: tea.StringValue(cluster.CurrentVersion),
		Network: irs.NetworkInfo{
			VpcIID: irs.IID{
				NameId:   tea.StringValue(vpcAttr.VpcName),
				SystemId: tea.StringValue(cluster.VpcId),
			},
			SubnetIIDs: []irs.IID{
				{
					NameId:   tea.StringValue(vswitchAttr.VSwitchName),
					SystemId: tea.StringValue(cluster.VswitchId),
				},
			},
			SecurityGroupIIDs: []irs.IID{
				{
					NameId:   tea.StringValue(secGroupAttr.SecurityGroupName),
					SystemId: tea.StringValue(cluster.SecurityGroupId),
				},
			},
		},
		Status:      clusterStatus,
		CreatedTime: createdTime,
		AccessInfo: irs.AccessInfo{
			Endpoint:   endpoint,
			Kubeconfig: kubeconfig,
		},
		//KeyValueList: []irs.KeyValue{},
	}

	//
	// 2025-03-13 StructToKeyValueList 사용으로 변경
	clusterInfo.KeyValueList = irs.StructToKeyValueList(cluster)
	// Fill clusterInfo.KeyValueList
	//

	// jsonCluster, err := json.Marshal(cluster)
	// if err != nil {
	// 	err = fmt.Errorf("failed to marshal cluster: %v", err)
	// 	err = fmt.Errorf("failed to get ClusterInfo: %v", err)
	// 	return nil, err
	// }

	// var mapCluster map[string]interface{}
	// err = json.Unmarshal(jsonCluster, &mapCluster)
	// if err != nil {
	// 	err = fmt.Errorf("failed to unmarshal cluster: %v", err)
	// 	err = fmt.Errorf("failed to get ClusterInfo: %v", err)
	// 	return nil, err
	// }

	// flat, err := flatten.Flatten(mapCluster, "", flatten.DotStyle)
	// if err != nil {
	// 	err = fmt.Errorf("failed to flatten cluster: %v", err)
	// 	err = fmt.Errorf("failed to get ClusterInfo: %v", err)
	// 	return nil, err
	// }
	// delete(flat, "meta_data")
	// for k, v := range flat {
	// 	clusterInfo.KeyValueList = append(clusterInfo.KeyValueList, irs.KeyValue{Key: k, Value: fmt.Sprintf("%v", v)})
	// }

	clusterInfo.KeyValueList = irs.StructToKeyValueList(cluster)

	return clusterInfo, nil
}

func (ach *AlibabaClusterHandler) getClusterInfo(regionId, clusterId string) (*irs.ClusterInfo, error) {
	//
	// Fill clusterInfo
	//
	clusterInfo, err := ach.getClusterInfoWithoutNodeGroupList(regionId, clusterId)
	if err != nil {
		err = fmt.Errorf("failed to get ClusterInfo: %v", err)
		return nil, err
	}

	//
	// Fill clusterInfo.NodeGroupList
	//
	nodepools, err := aliDescribeClusterNodePools(ach.CsClient, clusterId)
	if err != nil {
		err = fmt.Errorf("failed to get ClusterInfo: %v", err)
		return nil, err
	}

	for _, np := range nodepools {
		ngId := tea.StringValue(np.NodepoolInfo.NodepoolId)
		nodeGroupInfo, err := ach.getNodeGroupInfo(clusterId, ngId)
		if err != nil {
			err = fmt.Errorf("failed to get ClusterInfo: %v", err)
			return nil, err
		}

		clusterInfo.NodeGroupList = append(clusterInfo.NodeGroupList, *nodeGroupInfo)
	}

	return clusterInfo, nil
}

func (ach *AlibabaClusterHandler) getNodeGroupInfo(clusterId, nodeGroupId string) (*irs.NodeGroupInfo, error) {
	//
	// Fill nodeGroupInfo
	//
	nodepool, err := aliDescribeClusterNodePoolDetail(ach.CsClient, clusterId, nodeGroupId)
	if err != nil {
		err = fmt.Errorf("failed to get NodeGroupInfo: %v", err)
		return nil, err
	}
	if nodepool.Status == nil ||
		nodepool.ScalingGroup == nil ||
		nodepool.AutoScaling == nil ||
		nodepool.NodepoolInfo == nil {
		err = fmt.Errorf("invalid nodepool's information")
		err = fmt.Errorf("failed to get NodeGroupInfo: %v", err)
		return nil, err
	}

	ngiStatus := irs.NodeGroupInactive
	if strings.EqualFold(tea.StringValue(nodepool.Status.State), nodepoolStatusActive) {
		ngiStatus = irs.NodeGroupActive
	} else if strings.EqualFold(tea.StringValue(nodepool.Status.State), nodepoolStatusScaling) {
		ngiStatus = irs.NodeGroupUpdating
	} else if strings.EqualFold(tea.StringValue(nodepool.Status.State), nodepoolStatusRemoving) {
		ngiStatus = irs.NodeGroupUpdating // removing is a kind of updating?
	} else if strings.EqualFold(tea.StringValue(nodepool.Status.State), nodepoolStatusDeleting) {
		ngiStatus = irs.NodeGroupDeleting
	} else if strings.EqualFold(tea.StringValue(nodepool.Status.State), nodepoolStatusUpdating) {
		ngiStatus = irs.NodeGroupUpdating
	}

	// Handle case where InstanceTypes might be empty (e.g., failed node pools)
	vmSpecName := ""
	if len(nodepool.ScalingGroup.InstanceTypes) > 0 {
		vmSpecName = tea.StringValue(nodepool.ScalingGroup.InstanceTypes[0])
	} else {
		cblogger.Warnf("NodePool %s has empty InstanceTypes (State: %s)",
			tea.StringValue(nodepool.NodepoolInfo.Name),
			tea.StringValue(nodepool.Status.State))
		// Log the raw nodepool data for debugging
		cblogger.Debugf("Raw NodePool Data: %+v", nodepool)
	}

	nodeGroupInfo := &irs.NodeGroupInfo{
		IId: irs.IID{
			NameId:   tea.StringValue(nodepool.NodepoolInfo.Name),
			SystemId: tea.StringValue(nodepool.NodepoolInfo.NodepoolId),
		},
		ImageIID: irs.IID{
			NameId:   tea.StringValue(nodepool.ScalingGroup.ImageType),
			SystemId: tea.StringValue(nodepool.ScalingGroup.ImageId),
		},
		VMSpecName:   vmSpecName,
		RootDiskType: tea.StringValue(nodepool.ScalingGroup.SystemDiskCategory),
		RootDiskSize: strconv.FormatInt(tea.Int64Value(nodepool.ScalingGroup.SystemDiskSize), 10),
		KeyPairIID: irs.IID{
			NameId:   tea.StringValue(nodepool.ScalingGroup.KeyPair),
			SystemId: tea.StringValue(nodepool.ScalingGroup.KeyPair),
		},
		Status:          ngiStatus,
		OnAutoScaling:   tea.BoolValue(nodepool.AutoScaling.Enable),
		MinNodeSize:     int(tea.Int64Value(nodepool.AutoScaling.MinInstances)),
		MaxNodeSize:     int(tea.Int64Value(nodepool.AutoScaling.MaxInstances)),
		DesiredNodeSize: int(tea.Int64Value(nodepool.ScalingGroup.DesiredSize)),
	}

	//
	// Fill nodeGroupInfo.Nodes
	//
	nodes, err := aliDescribeClusterNodes(ach.CsClient, clusterId, tea.StringValue(nodepool.NodepoolInfo.NodepoolId))
	if err != nil {
		err = fmt.Errorf("failed to get NodeGroupInfo: %v", err)
		return nil, err
	}

	for _, n := range nodes {
		nId := tea.StringValue(n.InstanceId)
		if nId != "" {
			node := irs.IID{
				NameId:   tea.StringValue(n.InstanceName),
				SystemId: tea.StringValue(n.InstanceId),
			}
			nodeGroupInfo.Nodes = append(nodeGroupInfo.Nodes, node)
		}
	}

	//
	// 2025-03-13 StructToKeyValueList 사용으로 변경
	nodeGroupInfo.KeyValueList = irs.StructToKeyValueList(nodepool)
	// Fill nodeGroupInfo.KeyValueList
	//

	// jsonNodepool, err := json.Marshal(nodepool)
	// if err != nil {
	// 	err = fmt.Errorf("failed to marshal nodepool: %v", err)
	// 	err = fmt.Errorf("failed to get NodeGroupInfo: %v", err)
	// 	return nil, err
	// }

	// var mapNodepool map[string]interface{}
	// err = json.Unmarshal(jsonNodepool, &mapNodepool)
	// if err != nil {
	// 	err = fmt.Errorf("failed to unmarshal nodepool: %v", err)
	// 	err = fmt.Errorf("failed to get NodeGroupInfo: %v", err)
	// 	return nil, err
	// }

	// flat, err := flatten.Flatten(mapNodepool, "", flatten.DotStyle)
	// if err != nil {
	// 	err = fmt.Errorf("failed to flatten nodepool: %v", err)
	// 	err = fmt.Errorf("failed to get NodeGroupInfo: %v", err)
	// 	return nil, err
	// }
	// delete(flat, "meta_data")
	// for k, v := range flat {
	// 	nodeGroupInfo.KeyValueList = append(nodeGroupInfo.KeyValueList, irs.KeyValue{Key: k, Value: fmt.Sprintf("%v", v)})
	// }

	nodeGroupInfo.KeyValueList = irs.StructToKeyValueList(nodepool)

	return nodeGroupInfo, err
}

func aliDescribeNatGatewaysWithTagInVpc(vpcClient *vpc2016.Client, regionId, vpcId, networkType, tagKey, tagValue string) ([]*vpc2016.DescribeNatGatewaysResponseBodyNatGatewaysNatGateway, error) {
	tags := []*vpc2016.DescribeNatGatewaysRequestTag{
		&vpc2016.DescribeNatGatewaysRequestTag{
			Key:   tea.String(tagKey),
			Value: tea.String(tagValue),
		},
	}

	describeNatGatewaysRequest := &vpc2016.DescribeNatGatewaysRequest{
		RegionId:    tea.String(regionId),
		VpcId:       tea.String(vpcId),
		NetworkType: tea.String(networkType),
		Tag:         tags,
	}
	describeNatGatewaysResponse, err := vpcClient.DescribeNatGateways(describeNatGatewaysRequest)
	if err != nil {
		return make([]*vpc2016.DescribeNatGatewaysResponseBodyNatGatewaysNatGateway, 0), err
	}

	return describeNatGatewaysResponse.Body.NatGateways.NatGateway, nil
}

func getInternetNatGatewaysWithTagInVpc(vpcClient *vpc2016.Client, regionId, vpcId, tagKey, tagValue string) ([]*vpc2016.DescribeNatGatewaysResponseBodyNatGatewaysNatGateway, error) {
	natGatewayList, err := aliDescribeNatGatewaysWithTagInVpc(vpcClient, regionId, vpcId, "internet", tagKey, tagValue)
	if err != nil {
		return nil, err
	}

	return natGatewayList, nil
}

func aliTagNatGateway(vpcClient *vpc2016.Client, regionId, natGatewayId, tagKey, tagValue string) error {
	tags := []*vpc2016.TagResourcesRequestTag{
		&vpc2016.TagResourcesRequestTag{
			Key:   tea.String(tagKey),
			Value: tea.String(tagValue),
		},
	}

	tagResourcesRequest := &vpc2016.TagResourcesRequest{
		RegionId:     tea.String(regionId),
		ResourceId:   tea.StringSlice([]string{natGatewayId}),
		ResourceType: tea.String("NATGATEWAY"),
		Tag:          tags,
	}
	//cblogger.Debug(tagResourcesRequest)
	_, err := vpcClient.TagResources(tagResourcesRequest)
	if err != nil {
		return err
	}
	//cblogger.Debug(tagResourcesResponse.Body)

	return nil
}

func aliDeleteNatGateway(vpcClient *vpc2016.Client, regionId, natGatewayId string) error {
	deleteNatGatewayRequest := &vpc2016.DeleteNatGatewayRequest{
		RegionId:     tea.String(regionId),
		Force:        tea.Bool(true), // When deleting the NAT Gateway, the Snat Entry will be deleted
		NatGatewayId: tea.String(natGatewayId),
	}
	//cblogger.Debug(deleteNatGatewayRequest)
	_, err := vpcClient.DeleteNatGateway(deleteNatGatewayRequest)
	if err != nil {
		return err
	}
	//cblogger.Debug(deleteNatGatewayResponse)

	return nil
}

func aliDescribeClustersV1(csClient *cs2015.Client, regionId string) ([]*cs2015.DescribeClustersV1ResponseBodyClusters, error) {
	describeClustersV1Request := &cs2015.DescribeClustersV1Request{
		ClusterType: tea.String("ManagedKubernetes"),
		RegionId:    tea.String(regionId),
		//RegionId: tea.String("ap-northeast-1"),
	}
	cblogger.Debug(describeClustersV1Request)
	describeClustersV1Response, err := csClient.DescribeClustersV1(describeClustersV1Request)
	if err != nil {
		return make([]*cs2015.DescribeClustersV1ResponseBodyClusters, 0), err
	}
	cblogger.Debug(describeClustersV1Response.Body)

	return describeClustersV1Response.Body.Clusters, nil
}

func aliDescribeClusterDetail(csClient *cs2015.Client, clusterId string) (*cs2015.DescribeClusterDetailResponseBody, error) {
	describeClusterDetailResponse, err := csClient.DescribeClusterDetail(tea.String(clusterId))
	if err != nil {
		return nil, err
	}
	//cblogger.Debug(describeClusterDetailResponse.Body)

	return describeClusterDetailResponse.Body, nil
}

func existNotDeletedClusterWithTagInVpc(csClient *cs2015.Client, regionId, vpcId, tagKey, tagValue string) (bool, error) {
	clusterList, err := aliDescribeClustersV1(csClient, regionId)
	if err != nil {
		return false, err
	}

	var clusterListWithTagInVpc []*cs2015.DescribeClustersV1ResponseBodyClusters
	for _, cluster := range clusterList {
		if strings.EqualFold(*cluster.State, "deleting") ||
			strings.EqualFold(*cluster.State, "deleted") {
			continue
		}
		if strings.EqualFold(*cluster.VpcId, vpcId) {
			for _, tag := range cluster.Tags {
				if strings.EqualFold(*tag.Key, tagKey) &&
					strings.EqualFold(*tag.Value, tagValue) {
					clusterListWithTagInVpc = append(clusterListWithTagInVpc, cluster)
					break
				}
			}
		}
	}

	if len(clusterListWithTagInVpc) > 0 {
		return true, nil
	} else {
		return false, nil
	}
}

func aliDescribeVSwitches(vpcClient *vpc2016.Client, regionId, vpcId string) ([]*vpc2016.DescribeVSwitchesResponseBodyVSwitchesVSwitch, error) {
	describeVSwitchesRequest := &vpc2016.DescribeVSwitchesRequest{
		RegionId: tea.String(regionId),
		VpcId:    tea.String(vpcId),
	}
	//cblogger.Debug(describeVSwitchesRequest)
	describeVSwitchesResponse, err := vpcClient.DescribeVSwitches(describeVSwitchesRequest)
	if err != nil {
		return make([]*vpc2016.DescribeVSwitchesResponseBodyVSwitchesVSwitch, 0), err
	}
	//cblogger.Debug(describeVSwitchesResponse.Body.VSwitches.VSwitch)

	return describeVSwitchesResponse.Body.VSwitches.VSwitch, nil
}

// normalizeVersion converts version strings like "2.1.4.1" to Semantic Version format "2.1.4"
func normalizeVersion(version string) string {
	parts := strings.Split(version, ".")
	if len(parts) >= 3 {
		// Take only first 3 parts (major.minor.patch) for Semantic Version
		return strings.Join(parts[:3], ".")
	}
	return version
}

func getLatestRuntime(csClient *cs2015.Client, regionId, clusterType, k8sVersion string) (string, string, error) {
	metadata, err := aliDescribeKubernetesVersionMetadata(csClient, regionId, clusterType, k8sVersion)
	if err != nil {
		err = fmt.Errorf("failed to get latest runtime name and version: %v", err)
		return "", "", err
	}
	if len(metadata) == 0 {
		err = fmt.Errorf("failed to get kubernetes version metadata")
		return "", "", err
	}

	runtimeName := defaultClusterRuntimeName
	invalidVersion, _ := semver.NewVersion("0.0.0")
	latestVersion := invalidVersion
	var latestVersionString string

	// Debug: Log all available runtimes
	cblogger.Debugf("Available runtimes for K8s %s:", k8sVersion)
	for _, rt := range metadata[0].Runtimes {
		rtName := tea.StringValue(rt.Name)
		rtVersionStr := tea.StringValue(rt.Version)
		cblogger.Debugf("  - Runtime: %s, Version: %s", rtName, rtVersionStr)
		if strings.EqualFold(rtName, runtimeName) {
			// Try to parse as-is first
			rtVersion, err := semver.NewVersion(rtVersionStr)
			if err != nil {
				// If parsing fails, try to normalize the version (e.g., "2.1.4.1" -> "2.1.4")
				normalizedVersion := normalizeVersion(rtVersionStr)
				cblogger.Debugf("  - Normalizing version %s to %s", rtVersionStr, normalizedVersion)
				rtVersion, err = semver.NewVersion(normalizedVersion)
				if err != nil {
					cblogger.Warnf("  - Failed to parse version %s (normalized: %s): %v", rtVersionStr, normalizedVersion, err)
					// If still fails, use the original version string as fallback
					if latestVersion.Equal(invalidVersion) {
						latestVersionString = rtVersionStr
					}
					continue
				}
			}
			if latestVersion.Equal(invalidVersion) || latestVersion.LessThan(rtVersion) {
				latestVersion = rtVersion
				latestVersionString = rtVersionStr // Keep original version string
				cblogger.Debugf("  - New latest version: %s (parsed: %s)", latestVersionString, rtVersion.String())
			}
		}
	}

	if latestVersion.Equal(invalidVersion) {
		if latestVersionString == "" {
			err = fmt.Errorf("failed to get valid runtime version")
			return "", "", err
		}
		// Use the fallback version string if we have one
		cblogger.Infof("Selected latest runtime: %s version %s (using fallback)", runtimeName, latestVersionString)
		return runtimeName, latestVersionString, nil
	}
	runtimeVersion := latestVersionString

	cblogger.Infof("Selected latest runtime: %s version %s", runtimeName, runtimeVersion)

	return runtimeName, runtimeVersion, nil
}

func getNodepoolsFromNodeGroupList(nodeGroupInfoList []irs.NodeGroupInfo, runtimeName, runtimeVersion string, vswitchIds []string) []*cs2015.Nodepool {
	var nodepools []*cs2015.Nodepool
	for _, ngInfo := range nodeGroupInfoList {
		name := ngInfo.IId.NameId
		autoScalingEnable := ngInfo.OnAutoScaling
		maxInstances := ngInfo.MaxNodeSize
		minInstances := ngInfo.MinNodeSize
		instanceTypes := []string{ngInfo.VMSpecName}
		systemDiskCategory := ngInfo.RootDiskType
		systemDiskSize, _ := strconv.ParseInt(ngInfo.RootDiskSize, 10, 64)
		keyPair := ngInfo.KeyPairIID.NameId
		imageId := ngInfo.ImageIID.NameId
		if strings.EqualFold(imageId, "") || strings.EqualFold(imageId, "default") {
			imageId = ""
		}

		nodepool := cs2015.Nodepool{
			NodepoolInfo: &cs2015.NodepoolNodepoolInfo{
				Name: tea.String(name),
			},
			AutoScaling: &cs2015.NodepoolAutoScaling{
				Enable:       tea.Bool(autoScalingEnable),
				MaxInstances: tea.Int64(int64(maxInstances)),
				MinInstances: tea.Int64(int64(minInstances)),
			},
			KubernetesConfig: &cs2015.NodepoolKubernetesConfig{
				Runtime:        tea.String(runtimeName),
				RuntimeVersion: tea.String(runtimeVersion),
			},
			ScalingGroup: &cs2015.NodepoolScalingGroup{
				VswitchIds:         tea.StringSlice(vswitchIds),
				InstanceTypes:      tea.StringSlice(instanceTypes),
				SystemDiskCategory: tea.String(systemDiskCategory),
				SystemDiskSize:     tea.Int64(systemDiskSize),
				KeyPair:            tea.String(keyPair),
				ImageId:            tea.String(imageId),
				//DesiredSize:        tea.Int64(desiredSize),
			},
			Management: &cs2015.NodepoolManagement{
				Enable: tea.Bool(true),
			},
		}

		// CAUTION: if DesiredSize is set when AutoScaling is enabled, Alibaba reject the request
		if autoScalingEnable == false {
			nodepool.ScalingGroup.DesiredSize = tea.Int64(int64(ngInfo.DesiredNodeSize))
		}

		nodepools = append(nodepools, &nodepool)
	}

	return nodepools
}

func aliCreateCluster(csClient *cs2015.Client, name, regionId, clusterType, clusterSpec, k8sVersion, runtimeName, runtimeVersion, vpcId, containerCidr, serviceCidr, secGroupId string, snatEntry, endpointPublicAccess bool, masterVswitchIds []string, tagKey, tagValue string, tagList *[]cs2015.Tag, nodepools []*cs2015.Nodepool) (*string, error) {
	tags := []*cs2015.Tag{
		{
			Key:   tea.String(tagKey),
			Value: tea.String(tagValue),
		},
	}

	// tagList가 nil이 아니고 요소가 있으면 tags에 추가
	if tagList != nil && len(*tagList) > 0 {
		for _, tag := range *tagList {
			newTag := &cs2015.Tag{
				Key:   tag.Key,
				Value: tag.Value,
			}
			tags = append(tags, newTag)
		}
	}

	createClusterRequest := &cs2015.CreateClusterRequest{
		Name:              tea.String(name),
		RegionId:          tea.String(regionId),
		ClusterType:       tea.String(clusterType),
		ClusterSpec:       tea.String(clusterSpec),
		KubernetesVersion: tea.String(k8sVersion),
		Runtime: &cs2015.Runtime{
			Name:    tea.String(runtimeName),
			Version: tea.String(runtimeVersion),
		},
		Vpcid:                tea.String(vpcId),
		ContainerCidr:        tea.String(containerCidr),
		ServiceCidr:          tea.String(serviceCidr),
		MasterVswitchIds:     tea.StringSlice(masterVswitchIds),
		SecurityGroupId:      tea.String(secGroupId),
		SnatEntry:            tea.Bool(snatEntry),
		EndpointPublicAccess: tea.Bool(endpointPublicAccess),
		Tags:                 tags,
		//Nodepools:        nodepools,
	}
	if len(nodepools) > 0 {
		createClusterRequest.Nodepools = nodepools
	}
	//cblogger.Debug(createClusterRequest)
	createClusterResponse, err := csClient.CreateCluster(createClusterRequest)
	if err != nil {
		return nil, err
	}
	//cblogger.Debug(createClusterResponse.Body)

	return createClusterResponse.Body.ClusterId, nil
}

func aliDescribeVpcAttribute(vpcClient *vpc2016.Client, regionId, vpcId string) (*vpc2016.DescribeVpcAttributeResponseBody, error) {
	describeVpcAttributeRequest := &vpc2016.DescribeVpcAttributeRequest{
		RegionId: tea.String(regionId),
		VpcId:    tea.String(vpcId),
	}
	//cblogger.Debug(describeVpcAttributeRequest)
	describeVpcAttributeResponse, err := vpcClient.DescribeVpcAttribute(describeVpcAttributeRequest)
	if err != nil {
		return nil, err
	}
	//cblogger.Debug(describeVpcAttributeResponse.Body)

	return describeVpcAttributeResponse.Body, nil
}

func aliDescribeVSwitchAttributes(vpcClient *vpc2016.Client, regionId, vswitchId string) (*vpc2016.DescribeVSwitchAttributesResponseBody, error) {
	describeVSwitchAttributesRequest := &vpc2016.DescribeVSwitchAttributesRequest{
		RegionId:  tea.String(regionId),
		VSwitchId: tea.String(vswitchId),
	}
	//cblogger.Debug(describeVSwitchAttributesRequest)
	describeVSwitchAttributesResponse, err := vpcClient.DescribeVSwitchAttributes(describeVSwitchAttributesRequest)
	if err != nil {
		return nil, err
	}
	//cblogger.Debug(describeVSwitchAttributesResponse.Body)

	return describeVSwitchAttributesResponse.Body, nil
}

func aliDescribeSecurityGroupAttribute(ecsClient *ecs2014.Client, regionId, securityGroupId string) (*ecs2014.DescribeSecurityGroupAttributeResponseBody, error) {
	describeSecurityGroupAttributeRequest := &ecs2014.DescribeSecurityGroupAttributeRequest{
		RegionId:        tea.String(regionId),
		SecurityGroupId: tea.String(securityGroupId),
	}
	//cblogger.Debug(describeSecurityGroupAttributeRequest)
	describeSecurityGroupAttributeResponse, err := ecsClient.DescribeSecurityGroupAttribute(describeSecurityGroupAttributeRequest)
	if err != nil {
		return nil, err
	}
	//cblogger.Debug(describeSecurityGroupAttributeResponse.Body)

	return describeSecurityGroupAttributeResponse.Body, nil
}

func aliDeleteCluster(csClient *cs2015.Client, clusterId string, retainResources []string) (*cs2015.DeleteClusterResponseBody, error) {
	deleteClusterRequest := &cs2015.DeleteClusterRequest{
		RetainResources: tea.StringSlice(retainResources),
	}
	//cblogger.Debug(deleteClusterRequest)
	deleteClusterResponse, err := csClient.DeleteCluster(tea.String(clusterId), deleteClusterRequest)
	if err != nil {
		return nil, err
	}
	//cblogger.Debug(deleteClusterResponse.Body)

	return deleteClusterResponse.Body, nil
}

func aliDescribeKubernetesVersionMetadata(csClient *cs2015.Client, regionId, clusterType, k8sVersion string) ([]*cs2015.DescribeKubernetesVersionMetadataResponseBody, error) {
	describeKubernetesVersionMetadataRequest := &cs2015.DescribeKubernetesVersionMetadataRequest{
		Region:            tea.String(regionId),
		ClusterType:       tea.String(clusterType),
		KubernetesVersion: tea.String(k8sVersion),
	}
	//cblogger.Debug(describeKubernetesVersionMetadataRequest)
	describeKubernetesVersionMetadataResponse, err := csClient.DescribeKubernetesVersionMetadata(describeKubernetesVersionMetadataRequest)
	if err != nil {
		return nil, err
	}
	//cblogger.Debug(describeKubernetesVersionMetadataResponse.Body)

	return describeKubernetesVersionMetadataResponse.Body, nil
}

// extractRuntimeFromMetadata extracts runtime name and version from cluster metadata
func extractRuntimeFromMetadata(metaData string) (string, string, error) {
	// Parse metadata JSON string
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(metaData), &metadata); err != nil {
		return "", "", fmt.Errorf("failed to parse metadata JSON: %v", err)
	}

	// Extract Runtime and RuntimeVersion
	runtime, ok := metadata["Runtime"].(string)
	if !ok || runtime == "" {
		return "", "", fmt.Errorf("Runtime not found in metadata")
	}

	runtimeVersion, ok := metadata["RuntimeVersion"].(string)
	if !ok || runtimeVersion == "" {
		return "", "", fmt.Errorf("RuntimeVersion not found in metadata")
	}

	return runtime, runtimeVersion, nil
}

func aliCreateClusterNodePool(csClient *cs2015.Client, clusterId, name string, autoScalingEnable bool, maxInstances, minInstances int64, vswitchIds, instanceTypes []string, systemDiskCategory string, systemDiskSize int64, keyPair, imageId, imageType string, desiredSize int64) (*string, error) {
	// Note: KubernetesConfig is optional - Alibaba automatically uses cluster's runtime if not specified
	// The code below can be uncommented if you need to explicitly set runtime version

	// Get cluster information to extract runtime configuration
	// cluster, err := aliDescribeClusterDetail(csClient, clusterId)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to get cluster details: %v", err)
	// }

	// Extract runtime information from cluster's metadata
	// metaData := tea.StringValue(cluster.MetaData)
	// runtimeName, runtimeVersion, err := extractRuntimeFromMetadata(metaData)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to extract runtime from cluster metadata: %v", err)
	// }
	// cblogger.Debugf("Using cluster's existing runtime: %s version %s", runtimeName, runtimeVersion)

	createClusterNodePoolRequest := &cs2015.CreateClusterNodePoolRequest{
		NodepoolInfo: &cs2015.CreateClusterNodePoolRequestNodepoolInfo{
			Name: tea.String(name),
		},
		AutoScaling: &cs2015.CreateClusterNodePoolRequestAutoScaling{
			Enable:       tea.Bool(autoScalingEnable),
			MaxInstances: tea.Int64(maxInstances),
			MinInstances: tea.Int64(minInstances),
		},
		// KubernetesConfig is optional - uncomment if you need to explicitly set runtime
		// KubernetesConfig: &cs2015.CreateClusterNodePoolRequestKubernetesConfig{
		// 	Runtime:        tea.String(runtimeName),
		// 	RuntimeVersion: tea.String(runtimeVersion),
		// },
		ScalingGroup: &cs2015.CreateClusterNodePoolRequestScalingGroup{
			VswitchIds:         tea.StringSlice(vswitchIds),
			InstanceTypes:      tea.StringSlice(instanceTypes),
			SystemDiskCategory: tea.String(systemDiskCategory),
			SystemDiskSize:     tea.Int64(systemDiskSize),
			KeyPair:            tea.String(keyPair),
			ImageId:            tea.String(imageId),
			ImageType:          tea.String(imageType),
			//DesiredSize:        tea.Int64(desiredSize),
		},
		Management: &cs2015.CreateClusterNodePoolRequestManagement{
			Enable: tea.Bool(true),
		},
	}

	// CAUTION: if DesiredSize is set when AutoScaling is enabled, Alibaba reject the request
	if autoScalingEnable == false {
		createClusterNodePoolRequest.ScalingGroup.DesiredSize = tea.Int64(desiredSize)
	}

	//cblogger.Debug(createClusterNodePoolRequest)
	createClusterNodePoolResponse, err := csClient.CreateClusterNodePool(tea.String(clusterId), createClusterNodePoolRequest)
	if err != nil {
		return nil, err
	}
	//cblogger.Debug(createClusterNodePoolResponse.Body)

	return createClusterNodePoolResponse.Body.NodepoolId, nil
}

func aliDeleteClusterNodepool(csClient *cs2015.Client, clusterId, nodepoolId string, force bool) (*cs2015.DeleteClusterNodepoolResponseBody, error) {
	deleteClusterNodepoolRequest := &cs2015.DeleteClusterNodepoolRequest{
		Force: tea.Bool(force),
	}
	//cblogger.Debug(deleteClusterNodepoolRequest)
	deleteClusterNodepoolResponse, err := csClient.DeleteClusterNodepool(tea.String(clusterId), tea.String(nodepoolId), deleteClusterNodepoolRequest)
	if err != nil {
		return nil, err
	}
	//cblogger.Debug(deleteClusterNodepoolResponse.Body)

	return deleteClusterNodepoolResponse.Body, nil
}

// GetAvailableSystemDiskTypesForCluster queries available system disk types for a specific zone and instance type
// This is useful for ACK node pool creation as disk type availability varies by region/zone
func GetAvailableSystemDiskTypesForCluster(ecsClient *ecs.Client, regionId, zoneId, instanceType string) ([]string, error) {
	request := ecs.CreateDescribeAvailableResourceRequest()
	request.Scheme = "https"
	request.RegionId = regionId
	request.ZoneId = zoneId
	request.DestinationResource = "SystemDisk"
	request.InstanceType = instanceType
	request.InstanceChargeType = "PostPaid"

	response, err := ecsClient.DescribeAvailableResource(request)
	if err != nil {
		return nil, fmt.Errorf("failed to query available system disk types: %v", err)
	}

	var diskTypes []string
	for _, zone := range response.AvailableZones.AvailableZone {
		if zone.ZoneId == zoneId {
			for _, resource := range zone.AvailableResources.AvailableResource {
				if resource.Type == "SystemDisk" {
					for _, disk := range resource.SupportedResources.SupportedResource {
						if disk.Status == "Available" {
							diskTypes = append(diskTypes, disk.Value)
						}
					}
				}
			}
		}
	}

	if len(diskTypes) == 0 {
		return nil, fmt.Errorf("no available system disk types found for zone %s with instance type %s", zoneId, instanceType)
	}

	return diskTypes, nil
}

func aliDescribeClusterUserKubeconfig(csClient *cs2015.Client, clusterId string) (string, error) {
	request := &cs2015.DescribeClusterUserKubeconfigRequest{}
	response, err := csClient.DescribeClusterUserKubeconfig(tea.String(clusterId), request)
	if err != nil {
		return "", err
	}

	kubeconfig := tea.StringValue(response.Body.Config)

	var parsedConfig map[string]interface{}
	err = yaml.Unmarshal([]byte(kubeconfig), &parsedConfig)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal kubeconfig: %v", err)
	}

	modifyUserFields(parsedConfig)

	modifiedKubeconfig, err := yaml.Marshal(parsedConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal modified kubeconfig: %v", err)
	}

	return string(modifiedKubeconfig), nil
}

func modifyUserFields(config map[string]interface{}) {
	if contexts, ok := config["contexts"].([]interface{}); ok {
		for _, context := range contexts {
			if ctxMap, ok := context.(map[interface{}]interface{}); ok {
				if ctx, ok := ctxMap["context"].(map[interface{}]interface{}); ok {
					if user, ok := ctx["user"]; ok {
						ctx["user"] = fmt.Sprintf("%q", user)
					}
				}
				if name, ok := ctxMap["name"]; ok {
					ctxMap["name"] = fmt.Sprintf("%q", name)
				}
			}
		}
	}

	if users, ok := config["users"].([]interface{}); ok {
		for _, user := range users {
			if userMap, ok := user.(map[interface{}]interface{}); ok {
				if name, ok := userMap["name"]; ok {
					userMap["name"] = fmt.Sprintf("%q", name)
				}
			}
		}
	}
}

func aliDescribeClusterNodePools(csClient *cs2015.Client, clusterId string) ([]*cs2015.DescribeClusterNodePoolsResponseBodyNodepools, error) {
	describeClusterNodePoolsResponse, err := csClient.DescribeClusterNodePools(tea.String(clusterId))
	if err != nil {
		return make([]*cs2015.DescribeClusterNodePoolsResponseBodyNodepools, 0), err
	}
	//cblogger.Debug(describeClusterNodePoolsResponse.Body)

	return describeClusterNodePoolsResponse.Body.Nodepools, nil
}

func aliDescribeClusterNodePoolDetail(csClient *cs2015.Client, clusterId, nodepoolId string) (*cs2015.DescribeClusterNodePoolDetailResponseBody, error) {
	describeClusterNodePoolDetailResponse, err := csClient.DescribeClusterNodePoolDetail(tea.String(clusterId), tea.String(nodepoolId))
	if err != nil {
		return nil, err
	}
	//cblogger.Debug(describeClusterNodePoolDetailResponse.Body)

	return describeClusterNodePoolDetailResponse.Body, nil
}

func aliDescribeClusterNodes(csClient *cs2015.Client, clusterId, nodepoolId string) ([]*cs2015.DescribeClusterNodesResponseBodyNodes, error) {
	describeClusterNodesRequest := &cs2015.DescribeClusterNodesRequest{
		NodepoolId: tea.String(nodepoolId),
	}
	//cblogger.Debug(describeClusterNodesRequest)
	describeClusterNodesResponse, err := csClient.DescribeClusterNodes(tea.String(clusterId), describeClusterNodesRequest)
	if err != nil {
		return nil, err
	}
	//cblogger.Debug(describeClusterNodesResponse.Body)

	return describeClusterNodesResponse.Body.Nodes, nil
}

func aliModifyClusterNodePoolAutoScalingEnable(csClient *cs2015.Client, clusterId, nodepoolId string, enable bool) (*cs2015.ModifyClusterNodePoolResponseBody, error) {
	modifyClusterNodePoolRequest := &cs2015.ModifyClusterNodePoolRequest{
		AutoScaling: &cs2015.ModifyClusterNodePoolRequestAutoScaling{
			Enable: tea.Bool(enable),
		},
	}
	modifyClusterNodePoolResponse, err := csClient.ModifyClusterNodePool(tea.String(clusterId), tea.String(nodepoolId), modifyClusterNodePoolRequest)
	if err != nil {
		return nil, err
	}
	//cblogger.Debug(modifyClusterNodePoolResponse.Body)

	return modifyClusterNodePoolResponse.Body, nil
}

func aliModifyClusterNodePoolScalingSize(csClient *cs2015.Client, clusterId, nodepoolId string, autoScalingEnable bool, maxInstances, minInstances, desiredSize int64) (*cs2015.ModifyClusterNodePoolResponseBody, error) {
	modifyClusterNodePoolRequest := &cs2015.ModifyClusterNodePoolRequest{
		AutoScaling: &cs2015.ModifyClusterNodePoolRequestAutoScaling{
			Enable:       tea.Bool(autoScalingEnable),
			MaxInstances: tea.Int64(maxInstances),
			MinInstances: tea.Int64(minInstances),
		},
	}

	// CAUTION: if DesiredSize is set when AutoScaling is enabled, Alibaba reject the request
	if autoScalingEnable == false {
		modifyClusterNodePoolRequest.ScalingGroup.DesiredSize = tea.Int64(desiredSize)
	}
	//cblogger.Debug(modifyClusterNodePoolRequest)

	modifyClusterNodePoolResponse, err := csClient.ModifyClusterNodePool(tea.String(clusterId), tea.String(nodepoolId), modifyClusterNodePoolRequest)
	//cblogger.Debug("aliModifyClusterNodePoolScalingSize")
	if err != nil {
		return nil, err
	}
	//cblogger.Debug(modifyClusterNodePoolResponse.Body)

	return modifyClusterNodePoolResponse.Body, nil
}

func aliUpgradeCluster(csClient *cs2015.Client, clusterId, nextVersion string) (*cs2015.UpgradeClusterResponseBody, error) {
	upgradeClusterRequest := &cs2015.UpgradeClusterRequest{
		NextVersion: tea.String(nextVersion),
	}
	//cblogger.Debug(upgradeClusterRequest)
	upgradeClusterResponse, err := csClient.UpgradeCluster(tea.String(clusterId), upgradeClusterRequest)
	if err != nil {
		return nil, err
	}
	//cblogger.Debug(upgradeClusterResponse.Body)

	return upgradeClusterResponse.Body, nil
}

func aliDescribeSnatTableEntriesWithNatGateway(vpcClient *vpc2016.Client, regionId, natGatewayId string) ([]*vpc2016.DescribeSnatTableEntriesResponseBodySnatTableEntriesSnatTableEntry, error) {
	describeSnatTableEntriesRequest := &vpc2016.DescribeSnatTableEntriesRequest{
		RegionId:     tea.String(regionId),
		NatGatewayId: tea.String(natGatewayId),
	}
	//cblogger.Debug(describeSnatTableEntriesRequest)
	describeSnatTableEntriesResponse, err := vpcClient.DescribeSnatTableEntries(describeSnatTableEntriesRequest)
	if err != nil {
		return nil, err
	}
	//cblogger.Debug(describeSnatTableEntriesResponse.Body)

	return describeSnatTableEntriesResponse.Body.SnatTableEntries.SnatTableEntry, err
}

func aliDeleteSnatEntry(vpcClient *vpc2016.Client, regionId, snatTableId, snatEntryId string) error {
	deleteSnatEntryRequest := &vpc2016.DeleteSnatEntryRequest{
		RegionId:    tea.String(regionId),
		SnatTableId: tea.String(snatTableId),
		SnatEntryId: tea.String(snatEntryId),
	}
	//cblogger.Debug(deleteSnatEntryRequest)
	_, err := vpcClient.DeleteSnatEntry(deleteSnatEntryRequest)
	if err != nil {
		return err
	}
	//cblogger.Debug(deleteSnatEntryResponse.Body)

	return nil
}

func waitUntilSnatTableEntriesWithNatGatewayIsEmpty(vpcClient *vpc2016.Client, regionId, natGatewayId string) error {
	apiCallCount := 0
	maxAPICallCount := 20

	var waitingErr error
	for {
		snatEntries, err := aliDescribeSnatTableEntriesWithNatGateway(vpcClient, regionId, natGatewayId)
		if err != nil {
			maxAPICallCount = maxAPICallCount / 2
		}
		if len(snatEntries) == 0 {
			break
		}
		apiCallCount++
		if apiCallCount >= maxAPICallCount {
			waitingErr = fmt.Errorf("failed to get SNAT table entries: The maximum number of verification requests has been exceeded while waiting for availability of that resource")
			break
		}
		time.Sleep(5 * time.Second)
	}

	return waitingErr
}

func aliReleaseEipAddress(vpcClient *vpc2016.Client, regionId, eipId string) error {
	releaseEipAddressRequest := &vpc2016.ReleaseEipAddressRequest{
		RegionId:     tea.String(regionId),
		AllocationId: tea.String(eipId),
	}
	//cblogger.Debug(releaseEipAddressRequest)
	_, err := vpcClient.ReleaseEipAddress(releaseEipAddressRequest)
	if err != nil {
		return err
	}
	//cblogger.Debug(releaseEipAddressResponse.Body)

	return nil
}

func aliUnassociateEipAddressFromNatGateway(vpcClient *vpc2016.Client, regionId, eipId, natGatewayId string) error {
	unassociateEipAddressRequest := &vpc2016.UnassociateEipAddressRequest{
		RegionId:     tea.String(regionId),
		AllocationId: tea.String(eipId),
		InstanceId:   tea.String(natGatewayId),
		InstanceType: tea.String("Nat"),
	}
	//cblogger.Debug(unassociateEipAddressRequest)
	_, err := vpcClient.UnassociateEipAddress(unassociateEipAddressRequest)
	if err != nil {
		return err
	}
	//cblogger.Debug(unassociateEipAddressResponse.Body)

	return nil
}

func aliDescribeEipAddressesWithNatGateway(vpcClient *vpc2016.Client, regionId, natGatewayId string) (eipAddress []*vpc2016.DescribeEipAddressesResponseBodyEipAddressesEipAddress, err error) {
	describeEipAddressesRequest := &vpc2016.DescribeEipAddressesRequest{
		RegionId:               tea.String(regionId),
		AssociatedInstanceType: tea.String("Nat"),
		AssociatedInstanceId:   tea.String(natGatewayId),
	}
	//cblogger.Debug(describeEipAddressesRequest)
	describeEipAddressesResponse, err := vpcClient.DescribeEipAddresses(describeEipAddressesRequest)
	if err != nil {
		return nil, err
	}
	//cblogger.Debug(describeEipAddressesResponse.Body)

	return describeEipAddressesResponse.Body.EipAddresses.EipAddress, err
}

func waitUntilEipAddressesWithNatGatewayIsStatus(vpcClient *vpc2016.Client, regionId, natGatewayId, status string) error {
	apiCallCount := 0
	maxAPICallCount := 20

	var waitingErr error
	for {
		eipAddrs, err := aliDescribeEipAddressesWithNatGateway(vpcClient, regionId, natGatewayId)
		if err != nil {
			maxAPICallCount = maxAPICallCount / 2
		}
		equalAll := true
		for _, eip := range eipAddrs {
			if !strings.EqualFold(tea.StringValue(eip.Status), status) {
				equalAll = false
				break
			}
		}
		if equalAll == true {
			break
		}
		apiCallCount++
		if apiCallCount >= maxAPICallCount {
			waitingErr = fmt.Errorf("failed to get eip addresses: The maximum number of verification requests has been exceeded while waiting for availability of that resource")
			break
		}
		time.Sleep(5 * time.Second)
	}

	return waitingErr
}

func cleanCluster(csClient *cs2015.Client, clusterId string) error {
	_, err := aliDeleteCluster(csClient, clusterId, []string{})
	if err != nil {
		return err
	}

	return nil
}

func cleanNatGatewayWithEip(vpcClient *vpc2016.Client, regionId, natGatewayId string) error {
	snatEntries, err := aliDescribeSnatTableEntriesWithNatGateway(vpcClient, regionId, natGatewayId)
	if err != nil {
		err = fmt.Errorf("failed to get SNAT entries with NAT Gateway(ID=%s): %v", natGatewayId, err)
		cblogger.Error(err)
	} else {
		for _, snat := range snatEntries {
			aliDeleteSnatEntry(vpcClient, regionId, tea.StringValue(snat.SnatTableId), tea.StringValue(snat.SnatEntryId))
		}

		waitUntilSnatTableEntriesWithNatGatewayIsEmpty(vpcClient, regionId, natGatewayId)
	}

	eipAddrs, err := aliDescribeEipAddressesWithNatGateway(vpcClient, regionId, natGatewayId)
	if err != nil {
		err = fmt.Errorf("failed to get eip addresses with NAT Gateway(ID=%s): %v", natGatewayId, err)
		cblogger.Error(err)
	}

	for _, eipAddr := range eipAddrs {
		err = aliUnassociateEipAddressFromNatGateway(vpcClient, regionId, tea.StringValue(eipAddr.AllocationId), natGatewayId)
		if err != nil {
			err = fmt.Errorf("failed to unassociate eip address(ID=%s, IP=%s), it should be manually unassociated: %v", tea.StringValue(eipAddr.AllocationId), tea.StringValue(eipAddr.IpAddress), err)
			cblogger.Error(err)
		} else {
			cblogger.Infof("EIP Address(ID=%s) from NAT Gateway(ID=%s) is Unassociating.", tea.StringValue(eipAddr.AllocationId), natGatewayId)
		}
	}

	err = waitUntilEipAddressesWithNatGatewayIsStatus(vpcClient, regionId, natGatewayId, eipStatusAvailable)
	if err != nil {
		err = fmt.Errorf("failed to wait until eip addresses with NAT Gateway(ID=%s) is %s: %v", natGatewayId, eipStatusAvailable, err)
		cblogger.Error(err)
	}

	for _, eipAddr := range eipAddrs {
		err = aliReleaseEipAddress(vpcClient, regionId, tea.StringValue(eipAddr.AllocationId))
		if err != nil {
			err = fmt.Errorf("failed to release eip address(ID=%s, IP=%s), it should be manually unassociated: %v", tea.StringValue(eipAddr.AllocationId), tea.StringValue(eipAddr.IpAddress), err)
			cblogger.Error(err)
		} else {
			cblogger.Infof("EIP Address(ID=%s) associated with Internet NAT Gateway(ID=%s) will be released.", tea.StringValue(eipAddr.AllocationId), natGatewayId)
		}
	}

	aliDeleteNatGateway(vpcClient, regionId, natGatewayId)

	return nil
}

func validateAtCreateCluster(clusterInfo irs.ClusterInfo) error {
	if clusterInfo.IId.NameId == "" {
		return fmt.Errorf("Cluster name is required")
	}
	if clusterInfo.Network.VpcIID.SystemId == "" && clusterInfo.Network.VpcIID.NameId == "" {
		return fmt.Errorf("Cannot identify VPC(IID=%s)", clusterInfo.Network.VpcIID)
	}
	if len(clusterInfo.Network.SubnetIIDs) < 1 {
		return fmt.Errorf("At least one Subnet must be specified")
	}
	if len(clusterInfo.Network.SecurityGroupIIDs) < 1 {
		return fmt.Errorf("At least one Subnet must be specified")
	}
	// CAUTION: Currently CB-Spider's Alibaba PMKS Drivers does not support to create a cluster with nodegroups
	if len(clusterInfo.NodeGroupList) > 0 {
		return fmt.Errorf("Node Group cannot be specified")
	}

	return nil
}

func validateAtAddNodeGroup(clusterIID irs.IID, nodeGroupInfo irs.NodeGroupInfo) error {
	if clusterIID.SystemId == "" && clusterIID.NameId == "" {
		return fmt.Errorf("Invalid Cluster IID")
	}
	if nodeGroupInfo.IId.NameId == "" {
		return fmt.Errorf("Node Group's name is required")
	}

	// Alibaba API behavior:
	// - OnAutoScaling = true:  Uses MinNodeSize/MaxNodeSize (DesiredNodeSize is ignored)
	// - OnAutoScaling = false: Uses DesiredNodeSize (MinNodeSize/MaxNodeSize are ignored)

	if nodeGroupInfo.OnAutoScaling {
		// When auto-scaling is enabled, MinNodeSize and MaxNodeSize are required
		if nodeGroupInfo.MaxNodeSize < 1 {
			return fmt.Errorf("MaxNodeSize cannot be smaller than 1 when auto-scaling is enabled")
		}
		if nodeGroupInfo.MinNodeSize < 1 {
			return fmt.Errorf("MinNodeSize cannot be smaller than 1 when auto-scaling is enabled")
		}
		if nodeGroupInfo.MinNodeSize > nodeGroupInfo.MaxNodeSize {
			return fmt.Errorf("MinNodeSize cannot be greater than MaxNodeSize")
		}
		// Note: DesiredNodeSize is ignored when auto-scaling is enabled
	} else {
		// When auto-scaling is disabled, only DesiredNodeSize is used
		if nodeGroupInfo.DesiredNodeSize < 0 {
			return fmt.Errorf("DesiredNodeSize cannot be negative when auto-scaling is disabled")
		}
		// Note: MinNodeSize and MaxNodeSize are ignored when auto-scaling is disabled
		// They can be 0 or any value - Alibaba API will ignore them
	}

	if nodeGroupInfo.VMSpecName == "" {
		return fmt.Errorf("VM Spec Name is required")
	}

	return nil
}

func validateAtChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID, minNodeSize int, maxNodeSize int) error {
	if clusterIID.SystemId == "" && clusterIID.NameId == "" {
		return fmt.Errorf("Invalid Cluster IID")
	}
	if nodeGroupIID.SystemId == "" && nodeGroupIID.NameId == "" {
		return fmt.Errorf("Invalid Node Group IID")
	}
	if minNodeSize < 1 {
		return fmt.Errorf("MaxNodeSize cannot be smaller than 1")
	}
	if maxNodeSize < 1 {
		return fmt.Errorf("MaxNodeSize cannot be smaller than 1")
	}

	return nil
}

func (ach *AlibabaClusterHandler) getAvailableCidrList() ([]string, error) {
	//
	// Valid CIDR: 10.0.0.0/16-24, 172.16-31.0.0/16-24, and 192.168.0.0/16-24.
	//
	mapCidr := make(map[string]bool)
	for i := 16; i < 32; i++ {
		mapCidr[fmt.Sprintf("172.%v.0.0/16", i)] = true
	}

	clusterInfoList, err := ach.getClusterInfoListWithoutNodeGroupList(ach.RegionInfo.Region)
	if err != nil {
		err = fmt.Errorf("failed to get available CIDR list: %v", err)
		return []string{}, err
	}

	for _, clusterInfo := range clusterInfoList {
		for _, v := range clusterInfo.KeyValueList {
			if v.Key == "parameters.ServiceCIDR" || v.Key == "subnet_cidr" {
				delete(mapCidr, v.Value)
			}
		}
	}

	cidrList := []string{}
	for k := range mapCidr {
		cidrList = append(cidrList, k)
	}

	return cidrList, nil
}

func aliDescribeAvailableResourceWithInstanceType(ecsClient *ecs2014.Client, regionId, zoneId, instanceType string) ([]*ecs2014.DescribeAvailableResourceResponseBodyAvailableZonesAvailableZone, error) {
	emptyAz := make([]*ecs2014.DescribeAvailableResourceResponseBodyAvailableZonesAvailableZone, 0)
	describeAvailableResourceRequest := &ecs2014.DescribeAvailableResourceRequest{
		RegionId:            tea.String(regionId),
		ZoneId:              tea.String(zoneId),
		DestinationResource: tea.String("InstanceType"),
		InstanceType:        tea.String(instanceType),
	}
	//cblogger.Debug(describeAvailableResourceRequest)
	describeAvailableResourceResponse, err := ecsClient.DescribeAvailableResource(describeAvailableResourceRequest)
	if err != nil {
		return emptyAz, err
	}
	//cblogger.Debug(describeAvailableResourceResponse.Body)

	if describeAvailableResourceResponse.Body.AvailableZones == nil {
		// in case of invalid instanceType
		err = fmt.Errorf("no available zone")
		return emptyAz, err
	}

	return describeAvailableResourceResponse.Body.AvailableZones.AvailableZone, nil
}

func (ach *AlibabaClusterHandler) isAvailableInstanceType(regionId, zoneId, instanceType string) (bool, error) {
	availableZones, err := aliDescribeAvailableResourceWithInstanceType(ach.EcsClient, regionId, zoneId, instanceType)
	if err != nil {
		err = fmt.Errorf("failed to describe available resource with instance type(%s): %v", instanceType, err)
		return false, err
	}

	isAvailable := false
	for _, az := range availableZones {
		if strings.EqualFold(tea.StringValue(az.Status), "Available") {
			isAvailable = true
		}
	}

	return isAvailable, nil
}

/*
func waitUntilNodepoolIsState(csClient *cs2015.Client, clusterId, nodepoolId, state string) error {
	apiCallCount := 0
	maxAPICallCount := 20

	var waitingErr error
	for {
		nodepool, err := aliDescribeClusterNodePoolDetail(csClient, clusterId, nodepoolId)
		if err != nil {
			maxAPICallCount = maxAPICallCount / 2
		}
		if nodepool.Status != nil && strings.EqualFold(tea.StringValue(nodepool.Status.State), state) {
			return nil
		}
		apiCallCount++
		if apiCallCount >= maxAPICallCount {
			waitingErr = fmt.Errorf("failed to get nodepool: The maximum number of verification requests has been exceeded while waiting for availability of that resource")
			break
		}
		time.Sleep(5 * time.Second)
	}

	return waitingErr
}

// Check whether Internet NAT Gateway is avaiable or not
func isExistInternetNatGatewayInVpc(vpcClient *vpc2016.Client, regionId, vpcId, vSwitchId string) (bool, error) {
	natGatewayList, err := aliDescribeNatGateways(vpcClient, regionId, vpcId, vSwitchId, "internet")
	if err != nil {
		return false, err
	}

	if len(natGatewayList) > 0 {
		return true, nil
	}

	return false, nil
}

func aliAllocateEipAddress(vpcClient *vpc2016.Client, regionId, vpcId string) (*string, *string, error) {
	description := delimiterVpcId + vpcId

	allocateEipAddressRequest := &vpc2016.AllocateEipAddressRequest{
		RegionId:    tea.String(regionId),
		Description: tea.String(description),
	}
	allocateEipAddressResponse, err := vpcClient.AllocateEipAddress(allocateEipAddressRequest)
	if err != nil {
		return nil, nil, err
	}

	return allocateEipAddressResponse.Body.EipAddress, allocateEipAddressResponse.Body.AllocationId, nil
}

func aliDescribeEipAddressWithIdAndNat(vpcClient *vpc2016.Client, regionId, eipId, natGatewayId string) (eipAddress *vpc2016.DescribeEipAddressesResponseBodyEipAddressesEipAddress, err error) {
	describeEipAddressesRequest := &vpc2016.DescribeEipAddressesRequest{
		RegionId:               tea.String(regionId),
		AllocationId:           tea.String(eipId),
		AssociatedInstanceType: tea.String("Nat"),
		AssociatedInstanceId:   tea.String(natGatewayId),
	}
	//cblogger.Debug(describeEipAddressesRequest)
	describeEipAddressesResponse, err := vpcClient.DescribeEipAddresses(describeEipAddressesRequest)
	if err != nil {
		return nil, err
	}
	//cblogger.Debug(describeEipAddressesResponse.Body)

	eipCount := len(describeEipAddressesResponse.Body.EipAddresses.EipAddress)
	if eipCount == 1 {
		eipAddress = describeEipAddressesResponse.Body.EipAddresses.EipAddress[0]
		err = nil
	} else if eipCount == 0 {
		eipAddress = nil
		err = fmt.Errorf("no eip address(ID=%s)", eipId)
	} else {
		eipAddress = nil
		err = fmt.Errorf("more than one eip address(ID=%s)", eipId)
	}

	return eipAddress, err
}

func aliDescribeEipAddressesWithNat(vpcClient *vpc2016.Client, regionId, natGatewayId string) ([]*vpc2016.DescribeEipAddressesResponseBodyEipAddressesEipAddress, error) {
	describeEipAddressesRequest := &vpc2016.DescribeEipAddressesRequest{
		RegionId:               tea.String(regionId),
		AssociatedInstanceType: tea.String("Nat"),
		AssociatedInstanceId:   tea.String(natGatewayId),
	}
	//cblogger.Debug(describeEipAddressesRequest)
	describeEipAddressesResponse, err := vpcClient.DescribeEipAddresses(describeEipAddressesRequest)
	if err != nil {
		return make([]*vpc2016.DescribeEipAddressesResponseBodyEipAddressesEipAddress, 0), err
	}
	//cblogger.Debug(describeEipAddressesResponse.Body)

	return describeEipAddressesResponse.Body.EipAddresses.EipAddress, nil
}

func aliCreateNatGateway(vpcClient *vpc2016.Client, regionId, vpcId, vSwitchId string) (*string, []*string, error) {
	description := delimiterVpcId + vpcId

	createNatGatewayRequest := &vpc2016.CreateNatGatewayRequest{
		RegionId:    tea.String(regionId),
		VpcId:       tea.String(vpcId),
		VSwitchId:   tea.String(vSwitchId),
		NatType:     tea.String("Enhanced"),
		NetworkType: tea.String("internet"),
		Description: tea.String(description),
		EipBindMode: tea.String("NAT"),
	}
	//cblogger.Debug(createNatGatewayRequest)
	createNatGatewayResponse, err := vpcClient.CreateNatGateway(createNatGatewayRequest)
	if err != nil {
		return nil, make([]*string, 0), err
	}
	//cblogger.Debug(createNatGatewayResponse.Body)

	return createNatGatewayResponse.Body.NatGatewayId, createNatGatewayResponse.Body.SnatTableIds.SnatTableId, nil
}

func aliGetNatGatewayAttribute(vpcClient *vpc2016.Client, regionId, natGatewayId string) (*vpc2016.GetNatGatewayAttributeResponseBody, error) {
	getNatGatewayAttributeRequest := &vpc2016.GetNatGatewayAttributeRequest{
		RegionId:     tea.String(regionId),
		NatGatewayId: tea.String(natGatewayId),
	}
	//cblogger.Debug(getNatGatewayAttributeRequest)
	getNatGatewayAttributeResponse, err := vpcClient.GetNatGatewayAttribute(getNatGatewayAttributeRequest)
	if err != nil {
		return nil, err
	}
	//cblogger.Debug(getNatGatewayAttributeResponse.Body)

	return getNatGatewayAttributeResponse.Body, nil
}

func waitUntilNatGatewayIsAvailable(vpcClient *vpc2016.Client, regionId, natGatewayId string) error {
	apiCallCount := 0
	maxAPICallCount := 20

	var waitingErr error
	for {
		ngw, err := aliGetNatGatewayAttribute(vpcClient, regionId, natGatewayId)
		if err != nil {
			maxAPICallCount = maxAPICallCount / 2
		}
		if ngw != nil && strings.EqualFold(*ngw.Status, "Available") {
			return nil
		}
		apiCallCount++
		if apiCallCount >= maxAPICallCount {
			waitingErr = fmt.Errorf("failed to get NAT Gateway: The maximum number of verification requests has been exceeded while waiting for the creation of that resource")
			break
		}
		time.Sleep(10 * time.Second)
	}

	return waitingErr
}

func aliCreateSnatEntryForVpc(vpcClient *vpc2016.Client, regionId, snatTableId, snatIp, srcCidr string) error {
	createSnatEntryRequest := &vpc2016.CreateSnatEntryRequest{
		RegionId:    tea.String(regionId),
		SnatIp:      tea.String(snatIp),
		SnatTableId: tea.String(snatTableId),
		SourceCIDR:  tea.String(srcCidr),
	}
	//cblogger.Debug(createSnatEntryRequest)
	_, err := vpcClient.CreateSnatEntry(createSnatEntryRequest)
	if err != nil {
		return err
	}
	//cblogger.Debug(createSnatEntryResponse.Body)

	return nil
}

func waitUntilEipIsStatus(vpcClient *vpc2016.Client, regionId, eipId, natGatewayId, status string) error {
	apiCallCount := 0
	maxAPICallCount := 20

	var waitingErr error
	for {
		eipAddress, err := aliDescribeEipAddressWithIdAndNat(vpcClient, regionId, eipId, natGatewayId)
		if err != nil {
			maxAPICallCount = maxAPICallCount / 2
		}
		if eipAddress != nil && strings.EqualFold(*eipAddress.Status, status) {
			return nil
		}
		apiCallCount++
		if apiCallCount >= maxAPICallCount {
			waitingErr = fmt.Errorf("failed to get eip address: The maximum number of verification requests has been exceeded while waiting for availability of that resource")
			break
		}
		time.Sleep(5 * time.Second)
	}

	return waitingErr
}

func aliAssociateEipAddressToNatGateway(vpcClient *vpc2016.Client, regionId, eipId, natGatewayId, vpcId string) error {
	associateEipAddressRequest := &vpc2016.AssociateEipAddressRequest{
		RegionId:     tea.String(regionId),
		AllocationId: tea.String(eipId),
		InstanceId:   tea.String(natGatewayId),
		InstanceType: tea.String("Nat"),
		VpcId:        tea.String(vpcId),
	}
	//cblogger.Debug(associateEipAddressRequest)
	_, err := vpcClient.AssociateEipAddress(associateEipAddressRequest)
	if err != nil {
		return err
	}
	//cblogger.Debug(associateEipAddressResponse.Body)

	return nil
}

func createNatGatewayWithEip(vpcClient *vpc2016.Client, regionId, vpcId, vSwitchId string) (waitErr error) {
	var err error

	vpcAttribute, vpcErr := aliDescribeVpcAttribute(vpcClient, regionId, vpcId)
	if vpcErr != nil {
		vpcErr = fmt.Errorf("failed to get VPC Attribute: %v", vpcErr)
		return vpcErr
	}

	cblogger.Debug("Request to allocate EIP")
	eipAddress, eipId, allocateErr := aliAllocateEipAddress(vpcClient, regionId, vpcId)
	if allocateErr != nil {
		cblogger.Debug("Failed to allocate EIP: ", allocateErr)
		allocateErr = fmt.Errorf("failed to allocate an EIP: %v", allocateErr)
		return allocateErr
	}
	cblogger.Debug("Successfully allocated EIP: IP=", *eipAddress)

	cblogger.Debug("Request to create NAT Gateway")
	natGatewayId, snatTableIds, createNatGatewayErr := aliCreateNatGateway(vpcClient, regionId, vpcId, vSwitchId)
	if createNatGatewayErr != nil || len(snatTableIds) == 0 {
		cblogger.Debug("Failed to create NAT Gateway: ", createNatGatewayErr)
		err = aliReleaseEipAddress(vpcClient, regionId, *eipId)
		if err != nil {
			createNatGatewayErr = fmt.Errorf("failed to release EIP(ID=%s), it should be manually released: %v: %v", eipId, err, createNatGatewayErr)
		}

		createNatGatewayErr = fmt.Errorf("failed to create NAT Gateway: %v", createNatGatewayErr)
		return createNatGatewayErr
	}
	cblogger.Debug("Successfully created NAT Gateway: ID=", *natGatewayId)

	cblogger.Debug("Wait until NAT Gateway is Available: ID=", *natGatewayId)
	waitErr = waitUntilNatGatewayIsAvailable(vpcClient, regionId, *natGatewayId)
	if waitErr != nil {
		cblogger.Debug("Failed to wait until NAT Gateway is Available: ID=", *natGatewayId, ": ", waitErr)

		err = aliDeleteNatGateway(vpcClient, regionId, *natGatewayId)
		if err != nil {
			waitErr = fmt.Errorf("failed to delete NAT Gateway(ID=%s), it should be manually deleted: %v: %v", natGatewayId, err, waitErr)
		}

		err = aliReleaseEipAddress(vpcClient, regionId, *eipId)
		if err != nil {
			waitErr = fmt.Errorf("failed to release EIP(ID=%s), it should be manually released: %v: %v", eipId, err, waitErr)
		}

		waitErr = fmt.Errorf("failed to wait until NAT Gateway is available: %v", waitErr)
		return waitErr
	}

	cblogger.Debug("Request to associate EIP to NAT Gateway: IP=", *eipAddress, " NAT Gateway ID=", *natGatewayId)
	associateErr := aliAssociateEipAddressToNatGateway(vpcClient, regionId, *eipId, *natGatewayId, vpcId)
	if associateErr != nil {
		cblogger.Debug("Failed to associate EIP to NAT Gateway: IP=", *eipAddress, " NAT Gateway ID=", *natGatewayId, ": ", associateErr)

		err = aliDeleteNatGateway(vpcClient, regionId, *natGatewayId)
		if err != nil {
			associateErr = fmt.Errorf("failed to delete NAT Gateway(ID=%s), it should be manually deleted: %v: %v", natGatewayId, err, associateErr)
		}

		err = aliReleaseEipAddress(vpcClient, regionId, *eipId)
		if err != nil {
			associateErr = fmt.Errorf("failed to release EIP(ID=%s), it should be manually released: %v: %v", eipId, err, associateErr)
		}

		associateErr = fmt.Errorf("failed to associate EIP(ID=%s): %v", natGatewayId, associateErr)
		return associateErr
	}
	cblogger.Debug("Successfully associated EIP to NAT Gateway: IP=", *eipAddress, " NAT Gateway ID=", *natGatewayId)

	cblogger.Debug("Wait until EIP is InUse: IP=", *eipAddress)
	waitErr = waitUntilEipIsStatus(vpcClient, regionId, *eipId, *natGatewayId, "InUse")
	if waitErr != nil {
		cblogger.Debug("Failed to wait until EIP is InUse: IP=", *eipAddress, ": ", waitErr)

		err = aliDeleteNatGateway(vpcClient, regionId, *natGatewayId)
		if err != nil {
			waitErr = fmt.Errorf("failed to delete NAT Gateway(ID=%s), it should be manually deleted: %v: %v", natGatewayId, err, waitErr)
		}

		err = aliReleaseEipAddress(vpcClient, regionId, *eipId)
		if err != nil {
			waitErr = fmt.Errorf("failed to release EIP(ID=%s), it should be manually released: %v: %v", eipId, err, waitErr)
		}

		waitErr = fmt.Errorf("failed to wait until NAT Gateway is available: %v", waitErr)
		return waitErr
	}

	cblogger.Debug("Request to create SNAT Entry in NAT Gateway for VPC: NAT Gateway ID=", *natGatewayId, ", VPC ID=", vpcId)
	createSnatEntryErr := aliCreateSnatEntryForVpc(vpcClient, regionId, *snatTableIds[0], *eipAddress, *vpcAttribute.CidrBlock)
	if createSnatEntryErr != nil {
		cblogger.Debug("Failed to create SNAT Entry in NAT Gateway for VPC: NAT Gateway ID=", *natGatewayId, ", VPC ID=", vpcId, ": ", createSnatEntryErr)

		err = aliDeleteNatGateway(vpcClient, regionId, *natGatewayId)
		if err != nil {
			createSnatEntryErr = fmt.Errorf("failed to delete NAT Gateway(ID=%s), it should be manually deleted: %v: %v", natGatewayId, err, createSnatEntryErr)
		}

		err = aliReleaseEipAddress(vpcClient, regionId, *eipId)
		if err != nil {
			createSnatEntryErr = fmt.Errorf("failed to release EIP(ID=%s), it should be manually released: %v: %v", eipId, err, createSnatEntryErr)
		}

		createSnatEntryErr = fmt.Errorf("failed to create a SNAT etnry: %v", createSnatEntryErr)
		return createSnatEntryErr
	}
	cblogger.Debug("Successfully created SNAT Entry in NAT Gateway: ID=", *natGatewayId)

	return nil
}

func deleteNatGatewayWithEip(vpcClient *vpc2016.Client, regionId, vpcId, vSwitchId string) (resultErr error) {
	natGatewayList, err := aliDescribeNatGateways(vpcClient, regionId, vpcId, vSwitchId, "internet")
	if err != nil {
		resultErr = fmt.Errorf("failed to get NAT Gateways: %v", err)
		return resultErr
	}

	var ngwForCluster *vpc2016.DescribeNatGatewaysResponseBodyNatGatewaysNatGateway = nil
	for _, ngw := range natGatewayList {
		ngwVpcId := ""
		re := regexp.MustCompile(`\S*` + delimiterVpcId + `\S*`)
		found := re.FindString(*ngw.Description)
		if found != "" {
			split := strings.Split(found, delimiterVpcId)
			ngwVpcId = split[1]
			if strings.EqualFold(ngwVpcId, vpcId) {
				ngwForCluster = ngw
				break
			}
		}
	}

	if ngwForCluster == nil {
		resultErr = fmt.Errorf("no NAT Gateway in %s%s", delimiterVpcId, vpcId)
		return resultErr
	}

	cblogger.Debug("Request to delete NAT Gateway: ID=", *ngwForCluster.NatGatewayId)
	err = aliDeleteNatGateway(vpcClient, regionId, *ngwForCluster.NatGatewayId)
	if err != nil {
		cblogger.Debug("Failed to delete NAT Gateway: ID=", *ngwForCluster.NatGatewayId, ": ", err)
		resultErr = fmt.Errorf("failed to delete NAT Gateway(ID=%s): %v", *ngwForCluster.NatGatewayId, err)
	}
	cblogger.Debug("Successfully deleted NAT Gateway: ID=", *ngwForCluster.NatGatewayId)

	resultErr = nil
	eipAddressList, err := aliDescribeEipAddressesWithNat(vpcClient, regionId, *ngwForCluster.NatGatewayId)
	if err != nil {
		if resultErr != nil {
			resultErr = fmt.Errorf("%v - no EIP with NAT Gateway(ID=%s): %v", resultErr, *ngwForCluster.NatGatewayId, err)
		} else {
			resultErr = fmt.Errorf("no EIP with NAT Gateway(ID=%s): %v", *ngwForCluster.NatGatewayId, err)
		}
	} else {
		for _, eipAddr := range eipAddressList {
			eipVpcId := ""
			re := regexp.MustCompile(`\S*` + delimiterVpcId + `\S*`)
			found := re.FindString(*eipAddr.Description)
			if found != "" {
				split := strings.Split(found, delimiterVpcId)
				eipVpcId = split[1]
				if strings.EqualFold(eipVpcId, vpcId) {
					cblogger.Debug("Wait until EIP is Available: IP=", eipAddr.IpAddress)
					waitUntilEipIsStatus(vpcClient, regionId, *eipAddr.AllocationId, "", "Available")

					cblogger.Debug("Request to release EIP: IP=", eipAddr.IpAddress)
					err = aliReleaseEipAddress(vpcClient, regionId, *eipAddr.AllocationId)
					if err != nil {
						cblogger.Debug("Failed to release EIP: IP=", eipAddr.IpAddress, ": ", err)
					}
				}
			}
		}
	}

	return resultErr
}

func waitUntilClusterSecurityGroupIdIsExist(csClient *cs2015.Client, clusterId string) error {
	apiCallCount := 0
	maxAPICallCount := 20

	var waitingErr error
	for {
		cluster, err := aliDescribeClusterDetail(csClient, clusterId)
		if err != nil {
			maxAPICallCount = maxAPICallCount / 2
		}
		if !strings.EqualFold(tea.StringValue(cluster.SecurityGroupId), "") {
			return nil
		}
		apiCallCount++
		if apiCallCount >= maxAPICallCount {
			waitingErr = fmt.Errorf("failed to get cluster's security group id: The maximum number of verification requests has been exceeded while waiting for availability of that resource")
			break
		}
		time.Sleep(5 * time.Second)
		cblogger.Info("Wait until cluster's security group id is exist")
	}

	return waitingErr
}
*/
/*
	//
	// Check whether if a nat gateway is created with the cluster or not
	//
	cblogger.Debug(fmt.Sprintf("Check if NAT Gateway is Automatically Created."))

	tagKey := tagKeyAckAliyunCom
	tagValue := tea.StringValue(clusterId)
	ngwsWithTag, err := getInternetNatGatewaysWithTagInVpc(ach.VpcClient, regionId, vpcId, tagKey, tagValue)
	if err != nil {
		createErr = fmt.Errorf("Failed to Create Cluster: %v", err)
		cblogger.Error(createErr)
		LoggingError(hiscallInfo, err)
		return emptyClusterInfo, createErr
	}
	if len(ngwsWithTag) > 0 {
		cblogger.Debug(fmt.Sprintf("NAT Gateway(%s) is Automatically Created.", tea.StringValue(ngwsWithTag[0].NatGatewayId)))
		err = aliTagNatGateway(ach.VpcClient, regionId, tea.StringValue(ngwsWithTag[0].NatGatewayId), tagKeyCbSpiderPmksNatGateway, tagValueOwned)
		if err != nil {
			createErr = fmt.Errorf("Failed to Create Cluster: %v", err)
			cblogger.Error(createErr)
			LoggingError(hiscallInfo, createErr)
			return emptyClusterInfo, createErr
		}
	} else {
		cblogger.Debug(fmt.Sprintf("No Created NAT Gateway."))
	}
*/

func (alibabaClusterHandler *AlibabaClusterHandler) ListIID() ([]*irs.IID, error) {
	var iidList []*irs.IID

	cblogger.Debug("Alibaba Cloud Driver: called ListCluster()")
	hiscallInfo := GetCallLogScheme(alibabaClusterHandler.RegionInfo, call.CLUSTER, "ListCluster()", "ListCluster()")
	start := call.Start()

	cblogger.Infof("Get Cluster List")
	regionId := alibabaClusterHandler.RegionInfo.Region
	clusters, err := aliDescribeClustersV1(alibabaClusterHandler.CsClient, regionId)
	if err != nil {
		err := fmt.Errorf("Failed to List Cluster: %v", err)
		cblogger.Error(err)
		LoggingError(hiscallInfo, err)
		return iidList, err
	}

	for _, cluster := range clusters {
		iid := irs.IID{SystemId: tea.StringValue(cluster.ClusterId)}
		iidList = append(iidList, &iid)

	}
	LoggingInfo(hiscallInfo, start)
	return iidList, nil

}
