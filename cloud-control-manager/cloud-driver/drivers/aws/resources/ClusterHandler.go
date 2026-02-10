package resources

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsClusterHandler struct {
	CredentialInfo idrv.CredentialInfo
	Region         idrv.RegionInfo
	Client         *eks.EKS
	EC2Client      *ec2.EC2
	Iam            *iam.IAM
	StsClient      *sts.STS
	AutoScaling    *autoscaling.AutoScaling
	TagHandler     *AwsTagHandler // 2024-07-18 TagHandler add
}

const (
	NODEGROUP_TAG string = "nodegroup"

	clusterStatusCreating = "CREATING"
	clusterStatusActive   = "ACTIVE"
	clusterStatusDeleting = "DELETING"
	clusterStatusFailed   = "FAILED"
	clusterStatusUpdating = "UPDATING"
	clusterStatusPending  = "PENDING"

	nodeGroupStatusCreating     = "CREATING"
	nodeGroupStatusActive       = "ACTIVE"
	nodeGroupStatusUpdating     = "UPDATING"
	nodeGroupStatusDeleting     = "DELETING"
	nodeGroupStatusCreateFailed = "CREATE_FAILED"
	nodeGroupStatusDeleteFailed = "DELETE_FAILED"
	nodeGroupStatusDegraded     = "DEGRADED"
)

const (
	// set up EKS token duration: 15 minutes (max 1 hour)
	EKS_TOKEN_DURATION = 15 * time.Minute

	httpPutResponseHopLimitIs2 = 2
)

// AMIInfo holds AMI metadata for EKS AMI Type mapping
type AMIInfo struct {
	ImageId         string
	Name            string
	Description     string
	Architecture    string // "x86_64" or "arm64"
	PlatformDetails string // "Linux/UNIX", "Windows", etc.
	OSType          string // "amazon-linux-2", "amazon-linux-2023", "bottlerocket", "windows"
}

// getServerAddress returns the Spider server address from SERVER_ADDRESS env variable
func getServerAddress() string {
	hostEnv := os.Getenv("SERVER_ADDRESS")
	if hostEnv == "" {
		return "localhost:1024"
	}

	// Replace 0.0.0.0 with localhost (0.0.0.0 is for server binding, not client connection)
	if strings.HasPrefix(hostEnv, "0.0.0.0:") {
		return "localhost" + strings.TrimPrefix(hostEnv, "0.0.0.0")
	}
	if hostEnv == "0.0.0.0" {
		return "localhost:1024"
	}

	// "1.2.3.4" or "localhost"
	if !strings.Contains(hostEnv, ":") {
		return hostEnv + ":1024"
	}

	// ":31024" => "localhost:31024"
	if strings.HasPrefix(hostEnv, ":") {
		return "localhost" + hostEnv
	}

	// "1.2.3.4:31024" or "localhost:31024"
	return hostEnv
}

// getSpiderAuthUsername returns the Spider authentication username from SPIDER_AUTH_USERNAME env variable
func getSpiderAuthUsername() string {
	username := os.Getenv("SPIDER_AUTH_USERNAME")
	if username == "" {
		return "default"
	}
	return username
}

// getSpiderAuthPassword returns the Spider authentication password from SPIDER_AUTH_PASSWORD env variable
func getSpiderAuthPassword() string {
	password := os.Getenv("SPIDER_AUTH_PASSWORD")
	if password == "" {
		return "$2a$10$4PKzCuJ6fPYsbCF.HR//ieLjaCzBAdwORchx62F2JRXQsuR3d9T0q" // default bcrypt hash
	}
	return password
}

//------ Cluster Management

/*
	AWS Cluster는 Role이 필수임.
	우선, roleName=spider-eks-role로 설정, 생성 시 Role의 ARN을 조회하여 사용
*/

//------ AMI Analysis Helper Functions

// analyzeAMI analyzes received AMI ID and extracts metadata
func (h *AwsClusterHandler) analyzeAMI(amiID string) (*AMIInfo, error) {
	// Skip if empty (use default)
	if amiID == "" {
		cblogger.Info("No AMI specified, will use default")
		return nil, nil
	}

	cblogger.Infof("Analyzing received AMI: %s", amiID)

	input := &ec2.DescribeImagesInput{
		ImageIds: []*string{aws.String(amiID)},
	}

	result, err := h.EC2Client.DescribeImages(input)
	if err != nil {
		return nil, fmt.Errorf("failed to describe AMI %s: %w", amiID, err)
	}

	if len(result.Images) == 0 {
		return nil, fmt.Errorf("AMI %s not found", amiID)
	}

	image := result.Images[0]

	amiInfo := &AMIInfo{
		ImageId:         aws.StringValue(image.ImageId),
		Name:            aws.StringValue(image.Name),
		Description:     aws.StringValue(image.Description),
		Architecture:    aws.StringValue(image.Architecture),
		PlatformDetails: aws.StringValue(image.PlatformDetails),
	}

	// Detect OS type from name/description
	amiInfo.OSType = amiInfo.detectOSType()

	cblogger.Infof("AMI Analysis: arch=%s, platform=%s, osType=%s",
		amiInfo.Architecture, amiInfo.PlatformDetails, amiInfo.OSType)

	return amiInfo, nil
}

// detectOSType detects OS type from AMI name and description
func (info *AMIInfo) detectOSType() string {
	combined := strings.ToLower(info.Name + " " + info.Description)

	// Priority order matters!

	// 1. Windows (highest priority - check platform)
	if strings.Contains(strings.ToLower(info.PlatformDetails), "windows") {
		return "windows"
	}

	// 2. Bottlerocket
	if strings.Contains(combined, "bottlerocket") {
		return "bottlerocket"
	}

	// 3. Amazon Linux 2023
	if strings.Contains(combined, "2023") || strings.Contains(combined, "al2023") {
		return "amazon-linux-2023"
	}

	// 4. Amazon Linux 2 (default)
	if strings.Contains(combined, "amzn") ||
		strings.Contains(combined, "amazon linux") ||
		strings.Contains(combined, "al2") {
		return "amazon-linux-2"
	}

	// 5. Others (Ubuntu, etc.)
	if strings.Contains(combined, "ubuntu") {
		return "ubuntu"
	}

	// Default: assume Amazon Linux 2
	return "amazon-linux-2"
}

// isGPUInstanceType checks if instance type requires GPU support
func (h *AwsClusterHandler) isGPUInstanceType(instanceType string) bool {
	gpuPrefixes := []string{"p2.", "p3.", "p4.", "p5.", "g3.", "g4.", "g5."}

	for _, prefix := range gpuPrefixes {
		if strings.HasPrefix(instanceType, prefix) {
			return true
		}
	}

	return false
}

// mapToEKSAMIType maps AMI info to EKS AMI Type
func (h *AwsClusterHandler) mapToEKSAMIType(amiInfo *AMIInfo, instanceType string) string {
	// Default: AL2023 x86_64 (K8s 1.30+ compatible)
	if amiInfo == nil {
		cblogger.Info("No AMI info, using default: AL2023_x86_64_STANDARD")
		return "AL2023_x86_64_STANDARD"
	}

	osType := amiInfo.OSType
	isARM := amiInfo.Architecture == "arm64"
	isGPU := h.isGPUInstanceType(instanceType)

	cblogger.Infof("Mapping: osType=%s, arch=%s, gpu=%v", osType, amiInfo.Architecture, isGPU)

	// Priority 1: Windows
	if osType == "windows" {
		return h.selectWindowsAMIType(amiInfo.Name)
	}

	// Priority 2: Bottlerocket
	if osType == "bottlerocket" {
		if isGPU && isARM {
			cblogger.Info("Mapped to: BOTTLEROCKET_ARM_64_NVIDIA")
			return "BOTTLEROCKET_ARM_64_NVIDIA"
		}
		if isGPU {
			cblogger.Info("Mapped to: BOTTLEROCKET_x86_64_NVIDIA")
			return "BOTTLEROCKET_x86_64_NVIDIA"
		}
		if isARM {
			cblogger.Info("Mapped to: BOTTLEROCKET_ARM_64")
			return "BOTTLEROCKET_ARM_64"
		}
		cblogger.Info("Mapped to: BOTTLEROCKET_x86_64")
		return "BOTTLEROCKET_x86_64"
	}

	// Priority 3: Amazon Linux 2023
	if osType == "amazon-linux-2023" {
		if isARM {
			cblogger.Info("Mapped to: AL2023_ARM_64_STANDARD")
			return "AL2023_ARM_64_STANDARD"
		}
		cblogger.Info("Mapped to: AL2023_x86_64_STANDARD")
		return "AL2023_x86_64_STANDARD"
	}

	// Priority 4: GPU instances (use Bottlerocket for GPU - AL2023 has no GPU variant yet)
	if isGPU {
		if isARM {
			cblogger.Info("Mapped to: BOTTLEROCKET_ARM_64_NVIDIA (GPU + ARM64)")
			return "BOTTLEROCKET_ARM_64_NVIDIA"
		}
		cblogger.Info("Mapped to: BOTTLEROCKET_x86_64_NVIDIA (GPU)")
		return "BOTTLEROCKET_x86_64_NVIDIA"
	}

	// Priority 5: ARM64 (use AL2023 for K8s 1.30+ compatibility)
	if isARM {
		cblogger.Info("Mapped to: AL2023_ARM_64_STANDARD")
		return "AL2023_ARM_64_STANDARD"
	}

	// Priority 6: Default x86_64 (use AL2023 for K8s 1.30+ compatibility)
	cblogger.Info("Mapped to: AL2023_x86_64_STANDARD (default)")
	return "AL2023_x86_64_STANDARD"
}

// selectWindowsAMIType selects Windows AMI type based on version and edition
func (h *AwsClusterHandler) selectWindowsAMIType(amiName string) string {
	nameLower := strings.ToLower(amiName)

	// Detect Windows Server version and edition
	is2022 := strings.Contains(nameLower, "2022")
	isCore := strings.Contains(nameLower, "core")

	if is2022 {
		if isCore {
			cblogger.Info("Mapped to: WINDOWS_CORE_2022_x86_64")
			return "WINDOWS_CORE_2022_x86_64"
		}
		cblogger.Info("Mapped to: WINDOWS_FULL_2022_x86_64")
		return "WINDOWS_FULL_2022_x86_64"
	}

	// Default to 2019
	if isCore {
		cblogger.Info("Mapped to: WINDOWS_CORE_2019_x86_64")
		return "WINDOWS_CORE_2019_x86_64"
	}
	cblogger.Info("Mapped to: WINDOWS_FULL_2019_x86_64")
	return "WINDOWS_FULL_2019_x86_64"
}

// ------ Cluster Management
func (ClusterHandler *AwsClusterHandler) CreateCluster(clusterReqInfo irs.ClusterInfo) (irs.ClusterInfo, error) {
	// validation check

	reqSecurityGroupIds := clusterReqInfo.Network.SecurityGroupIIDs
	var securityGroupIds []*string
	for _, securityGroupIID := range reqSecurityGroupIds {
		securityGroupIds = append(securityGroupIds, aws.String(securityGroupIID.SystemId))
	}

	reqSubnetIds := clusterReqInfo.Network.SubnetIIDs
	var subnetIds []*string
	for _, subnetIID := range reqSubnetIds {
		subnetIds = append(subnetIds, aws.String(subnetIID.SystemId))
	}

	//AWS의 경우 사전에 Role의 생성이 필요하며, 현재는 role 이름을 다음 이름으로 일치 시킨다.(추후 개선)
	//예시) cluster : cloud-barista-eks-cluster-role, Node : cloud-barista-eks-nodegroup-role
	eksRoleName := "cloud-barista-eks-cluster-role"
	// get or create Role Arn
	eksRole, err := ClusterHandler.getOrCreateEKSClusterRole(eksRoleName)
	if err != nil {
		cblogger.Error(err)
		// role 은 required 임.
		return irs.ClusterInfo{}, fmt.Errorf("failed to get or create EKS cluster role: %w", err)
	}
	roleArn := eksRole.Role.Arn

	reqK8sVersion := clusterReqInfo.Version

	tagsMap, err := ConvertTagListToTagsMap(clusterReqInfo.TagList, clusterReqInfo.IId.NameId)
	if err != nil {
		return irs.ClusterInfo{}, fmt.Errorf("failed to convert tags map: %w", err)
	}

	// create cluster
	input := &eks.CreateClusterInput{
		Name: aws.String(clusterReqInfo.IId.NameId),
		ResourcesVpcConfig: &eks.VpcConfigRequest{
			SecurityGroupIds: securityGroupIds,
			SubnetIds:        subnetIds,
		},
		//RoleArn: aws.String("arn:aws:iam::012345678910:role/eks-service-role-AWSServiceRoleForAmazonEKS-J7ONKE3BQ4PI"),
		//RoleArn: aws.String(roleArn),
		RoleArn: roleArn,
		Tags:    tagsMap,
	}

	//EKS버전 처리(Spider 입력 값 형태 : "1.23.4" / AWS 버전 형태 : "1.23")
	if reqK8sVersion != "" {
		arrVer := strings.Split(reqK8sVersion, ".")
		switch len(arrVer) {
		case 2: // 그대로 적용
			input.Version = aws.String(reqK8sVersion)
		case 3: // 앞의 2자리만 취함. (정상적인 입력 형태)
			input.Version = aws.String(arrVer[0] + "." + arrVer[1])
		default: // 위 2가지 외에는 CSP의 기본값(최신버전)을 적용 함.
		}
	}

	cblogger.Debug(input)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   ClusterHandler.Region.Zone,
		ResourceType: call.CLUSTER,
		ResourceName: clusterReqInfo.IId.NameId,
		CloudOSAPI:   "CreateCluster()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := ClusterHandler.Client.CreateCluster(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeResourceInUseException:
				cblogger.Error(eks.ErrCodeResourceInUseException, aerr.Error())
			case eks.ErrCodeResourceLimitExceededException:
				cblogger.Error(eks.ErrCodeResourceLimitExceededException, aerr.Error())
			case eks.ErrCodeInvalidParameterException:
				cblogger.Error(eks.ErrCodeInvalidParameterException, aerr.Error())
			case eks.ErrCodeClientException:
				cblogger.Error(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeServerException:
				cblogger.Error(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeServiceUnavailableException:
				cblogger.Error(eks.ErrCodeServiceUnavailableException, aerr.Error())
			case eks.ErrCodeUnsupportedAvailabilityZoneException:
				cblogger.Error(eks.ErrCodeUnsupportedAvailabilityZoneException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			cblogger.Error(err.Error())
		}
		return irs.ClusterInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Debug(result)

	/*// Sync Call에서 Async Call로 변경 - 이슈:#716
	//----- wait until Status=COMPLETE -----//  :  cluster describe .status 로 확인
	errWait := ClusterHandler.WaitUntilClusterActive(result.Cluster.Identity.String())
	if errWait != nil {
		cblogger.Error(errWait)
		return irs.ClusterInfo{}, errWait
	}
	*/

	/*
		//노드그룹 추가
		clusterIID := irs.IID{NameId: clusterReqInfo.IId.NameId, SystemId: result.Cluster.Identity.String()}
		nodeGroupInfoList := clusterReqInfo.NodeGroupList
		for _, nodeGroupInfo := range nodeGroupInfoList {
			resultNodeGroupInfo, nodeGroupErr := ClusterHandler.AddNodeGroup(clusterIID, nodeGroupInfo)
			if nodeGroupErr != nil {
				cblogger.Error(err.Error())
			}
			cblogger.Debug(resultNodeGroupInfo)

		//----- wait until Status=COMPLETE -----//  :  Nodegroup이 모두 생성되면 조회
	*/

	clusterReqInfo.IId.SystemId = *result.Cluster.Name
	clusterInfo, errClusterInfo := ClusterHandler.GetCluster(clusterReqInfo.IId)
	if errClusterInfo != nil {
		cblogger.Error(errClusterInfo.Error())
		return irs.ClusterInfo{}, errClusterInfo
	}
	clusterInfo.IId.NameId = clusterReqInfo.IId.NameId
	clusterInfo.TagList, _ = ClusterHandler.TagHandler.ListTag(irs.CLUSTER, clusterInfo.IId)

	//--- install CSI driver and pod identity agent for EBS
	csiagenterr := ClusterHandler.InstallEBSCSIDriverAndPodIdentityAgentIfNotExists(clusterInfo.IId.SystemId)
	if csiagenterr != nil {
		cblogger.Errorf("Failed to install EBS CSI Driver and pod identity agent: %w", csiagenterr)
		return irs.ClusterInfo{}, csiagenterr
	}
	//--- end of install CSI driver for EBS

	return clusterInfo, nil
}

// Nodegroup이 Activty 상태일때까지 대기함.
func (ClusterHandler *AwsClusterHandler) WaitUntilNodegroupActive(clusterName string, nodegroupName string) error {
	cblogger.Debugf("Cluster Name : [%s] / NodegroupName : [%s]", clusterName, nodegroupName)
	input := &eks.DescribeNodegroupInput{
		ClusterName:   aws.String(clusterName),
		NodegroupName: aws.String(nodegroupName),
	}

	err := ClusterHandler.Client.WaitUntilNodegroupActive(input)
	if err != nil {
		cblogger.Errorf("failed to wait until Nodegroup Active : %v", err)
		return err
	}
	cblogger.Debug("=========WaitUntilNodegroupActive() 종료")
	return nil
}

// Cluster가 Activty 상태일때까지 대기함.
func (ClusterHandler *AwsClusterHandler) WaitUntilClusterActive(clusterName string) error {
	cblogger.Debugf("Cluster Name : [%s]", clusterName)
	input := &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	}

	// AWS SDK의 WaitUntilClusterActive 함수는 내부적으로 폴링(polling) 메커니즘을 구현
	err := ClusterHandler.Client.WaitUntilClusterActive(input)
	if err != nil {
		cblogger.Errorf("failed to wait until cluster Active: %v", err)
		return err
	}
	cblogger.Debug("=========WaitUntilClusterActive() ended")
	return nil
}

func (ClusterHandler *AwsClusterHandler) ListCluster() ([]*irs.ClusterInfo, error) {
	//return irs.ClusterInfo{}, nil
	if ClusterHandler == nil {
		cblogger.Error("ClusterHandlerIs nil")
		return nil, errors.New("ClusterHandler is nil")

	}

	cblogger.Debug(ClusterHandler)
	if ClusterHandler.Client == nil {
		cblogger.Error(" ClusterHandler.Client Is nil")
		return nil, errors.New("ClusterHandler is nil")
	}

	input := &eks.ListClustersInput{}
	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   ClusterHandler.Region.Zone,
		ResourceType: call.CLUSTER,
		ResourceName: "List()",
		CloudOSAPI:   "ListClusters()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := ClusterHandler.Client.ListClusters(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeInvalidParameterException:
				cblogger.Error(eks.ErrCodeInvalidParameterException, aerr.Error())
			case eks.ErrCodeClientException:
				cblogger.Error(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeServerException:
				cblogger.Error(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeServiceUnavailableException:
				cblogger.Error(eks.ErrCodeServiceUnavailableException, aerr.Error())
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
	callogger.Info(call.String(callLogInfo))

	cblogger.Debug(result)

	clusterList := []*irs.ClusterInfo{}
	for _, clusterName := range result.Clusters {

		clusterInfo, err := ClusterHandler.GetCluster(irs.IID{SystemId: *clusterName})
		if err != nil {
			cblogger.Error(err)
			continue //	에러가 나면 일단 skip시킴.
		}
		clusterList = append(clusterList, &clusterInfo)

	}
	return clusterList, nil

}

func (ClusterHandler *AwsClusterHandler) GetCluster(clusterIID irs.IID) (irs.ClusterInfo, error) {
	input := &eks.DescribeClusterInput{
		Name: aws.String(clusterIID.SystemId),
	}

	cblogger.Debug(input)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   ClusterHandler.Region.Zone,
		ResourceType: call.CLUSTER,
		ResourceName: clusterIID.SystemId,
		CloudOSAPI:   "DescribeCluster()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := ClusterHandler.Client.DescribeCluster(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeResourceNotFoundException:
				cblogger.Error(eks.ErrCodeResourceNotFoundException, aerr.Error())
			case eks.ErrCodeClientException:
				cblogger.Error(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeServerException:
				cblogger.Error(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeServiceUnavailableException:
				cblogger.Error(eks.ErrCodeServiceUnavailableException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			cblogger.Error(err.Error())
		}
		return irs.ClusterInfo{}, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Debug(result)

	clusterInfo := irs.ClusterInfo{
		IId:         irs.IID{NameId: "", SystemId: *result.Cluster.Name},
		Version:     *result.Cluster.Version,
		CreatedTime: *result.Cluster.CreatedAt,
		Status:      convertClusterStatusToClusterInfoStatus(aws.StringValue(result.Cluster.Status)),
		//AccessInfo:  irs.AccessInfo{Endpoint: *result.Cluster.Endpoint},
		AccessInfo: irs.AccessInfo{},
	}
	/*
		NodeGroupList []NodeGroupInfo
		Addons        AddonsInfo
	*/

	if !reflect.ValueOf(result.Cluster.Endpoint).IsNil() {
		clusterInfo.AccessInfo.Endpoint = *result.Cluster.Endpoint
	}
	if !reflect.ValueOf(result.Cluster.CertificateAuthority.Data).IsNil() {
		clusterInfo.AccessInfo.Kubeconfig = ClusterHandler.getKubeConfig(result)
	}

	if !reflect.ValueOf(result.Cluster.ResourcesVpcConfig).IsNil() {
		clusterInfo.Network.VpcIID = irs.IID{SystemId: *result.Cluster.ResourcesVpcConfig.VpcId}

		//서브넷 처리
		//SubnetIds: ["subnet-0d30ee6b367974a39","subnet-06d5c04b32019b81f","subnet-05c5d26bd2f014591"],
		if len(result.Cluster.ResourcesVpcConfig.SubnetIds) > 0 {
			for _, curSubnetId := range result.Cluster.ResourcesVpcConfig.SubnetIds {
				clusterInfo.Network.SubnetIIDs = append(clusterInfo.Network.SubnetIIDs, irs.IID{SystemId: *curSubnetId})
			}
		}

		//클러스터 보안그룹 처리
		// ClusterSecurityGroupId: "sg-0bb02bf07fe5f42f0",
		//@TODO - 클러스터 생성시 자동으로 추가되는 보안 그룹이라서 일단 CB보안그룹 목록에 포함은 시키지 않았음.
		if !reflect.ValueOf(result.Cluster.ResourcesVpcConfig.ClusterSecurityGroupId).IsNil() {
			//if *result.Cluster.ResourcesVpcConfig.ClusterSecurityGroupId != "" {
			/*
				for _, curSecurityGroupId := range result.Cluster.ResourcesVpcConfig.SecurityGroupIds {
					clusterInfo.Network.SecurityGroupIIDs = append(clusterInfo.Network.SecurityGroupIIDs, irs.IID{SystemId: *curSecurityGroupId})
				}
			*/
		}

		//보안그룹 처리 : "추가 보안 그룹"에 해당하는 듯
		if len(result.Cluster.ResourcesVpcConfig.SecurityGroupIds) > 0 {
			for _, curSecurityGroupId := range result.Cluster.ResourcesVpcConfig.SecurityGroupIds {
				clusterInfo.Network.SecurityGroupIIDs = append(clusterInfo.Network.SecurityGroupIIDs, irs.IID{SystemId: *curSecurityGroupId})
			}
		}
	}

	// keyValueList := []irs.KeyValue{
	// 	{Key: "Status", Value: *result.Cluster.Status},
	// 	{Key: "Arn", Value: *result.Cluster.Arn},
	// 	{Key: "RoleArn", Value: *result.Cluster.RoleArn},
	// }
	// clusterInfo.KeyValueList = keyValueList
	// irs.StructToKeyValueList() 함수 사용
	clusterInfo.KeyValueList = irs.StructToKeyValueList(result.Cluster)
	clusterInfo.Network.KeyValueList = irs.StructToKeyValueList(result.Cluster.ResourcesVpcConfig)

	clusterInfo.TagList, _ = ClusterHandler.TagHandler.ListTag(irs.CLUSTER, clusterInfo.IId)

	//노드 그룹 처리
	resNodeGroupList, errNodeGroup := ClusterHandler.ListNodeGroup(clusterInfo.IId)
	if errNodeGroup != nil {
		cblogger.Error(errNodeGroup)
		return irs.ClusterInfo{}, errNodeGroup
	}

	cblogger.Debug(resNodeGroupList)

	//노드 그룹 타입 변환
	for _, curNodeGroup := range resNodeGroupList {
		cblogger.Debugf("Node Group : [%s]", curNodeGroup.IId.NameId)
		curNodeGroup.KeyValueList = irs.StructToKeyValueList(curNodeGroup)
		clusterInfo.NodeGroupList = append(clusterInfo.NodeGroupList, *curNodeGroup)
	}

	// Addons 처리
	addons, err := ClusterHandler.ListAddons(clusterIID)
	if err != nil {
		cblogger.Error(err)
		return irs.ClusterInfo{}, err
	}
	clusterInfo.Addons = addons

	cblogger.Debug(clusterInfo)

	return clusterInfo, nil
}

func (ClusterHandler *AwsClusterHandler) ListAddons(clusterIID irs.IID) (irs.AddonsInfo, error) {
	input := &eks.ListAddonsInput{
		ClusterName: aws.String(clusterIID.SystemId),
	}

	result, err := ClusterHandler.Client.ListAddons(input)
	if err != nil {
		cblogger.Error(err)
		return irs.AddonsInfo{}, err
	}

	addonsInfo := irs.AddonsInfo{}
	for _, addonName := range result.Addons {
		addonInfo, err := ClusterHandler.GetAddon(clusterIID, *addonName)
		if err != nil {
			cblogger.Error(err)
			continue
		}
		addonsInfo.KeyValueList = append(addonsInfo.KeyValueList, addonInfo.KeyValueList...)
	}

	return addonsInfo, nil
}

func (ClusterHandler *AwsClusterHandler) GetAddon(clusterIID irs.IID, addonName string) (irs.AddonsInfo, error) {
	input := &eks.DescribeAddonInput{
		ClusterName: aws.String(clusterIID.SystemId),
		AddonName:   aws.String(addonName),
	}

	result, err := ClusterHandler.Client.DescribeAddon(input)
	if err != nil {
		cblogger.Error(err)
		return irs.AddonsInfo{}, err
	}

	addonInfo := irs.AddonsInfo{
		KeyValueList: irs.StructToKeyValueList(result.Addon),
	}

	return addonInfo, nil
}

func (ClusterHandler *AwsClusterHandler) getKubeConfig(clusterDesc *eks.DescribeClusterOutput) string {
	// Return dynamic token kubeconfig instead of static token
	return ClusterHandler.getDynamicKubeConfig(clusterDesc)
}

// getStaticKubeConfig generates kubeconfig with embedded static token
func (ClusterHandler *AwsClusterHandler) getStaticKubeConfig(clusterDesc *eks.DescribeClusterOutput) string {

	cluster := clusterDesc.Cluster

	// create kubeconfig with EKS token
	token, err := ClusterHandler.generateEKSToken(*cluster.Name)
	if err != nil {
		cblogger.Errorf("Failed to generate EKS token: %v", err)
		// empty token when error occurs
		token = ""
	}

	// Generate kubeconfig content with the token
	kubeconfigContent := fmt.Sprintf(`apiVersion: v1
clusters:
- cluster:
    server: %s
    certificate-authority-data: %s
  name: %s
contexts:
- context:
    cluster: %s
    user: aws-token
  name: %s
current-context: %s
kind: Config
preferences: {}
users:
- name: aws-token
  user:
    token: %s
`, *cluster.Endpoint, *cluster.CertificateAuthority.Data, *cluster.Name, *cluster.Name, *cluster.Name, *cluster.Name, token)

	return kubeconfigContent
}

// getDynamicKubeConfig generates kubeconfig with exec-based dynamic token
func (ClusterHandler *AwsClusterHandler) getDynamicKubeConfig(clusterDesc *eks.DescribeClusterOutput) string {

	cluster := clusterDesc.Cluster

	// Get Spider server address from environment variable
	serverAddr := getServerAddress()

	// Get Spider authentication credentials from environment variables
	authUsername := getSpiderAuthUsername()
	authPassword := getSpiderAuthPassword()

	// Get ConnectionName from CredentialInfo
	connectionName := ClusterHandler.CredentialInfo.ConnectionName
	if connectionName == "" {
		// Fallback: use provider-region format if ConnectionName is not set
		connectionName = "aws-" + ClusterHandler.Region.Region
	}

	// Generate kubeconfig content with exec-based dynamic token using cluster NameId instead of SystemId
	kubeconfigContent := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: %s
    certificate-authority-data: %s
  name: %s
contexts:
- context:
    cluster: %s
    user: aws-dynamic-token
  name: %s
current-context: %s
users:
- name: aws-dynamic-token
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1
      interactiveMode: Never
      command: curl
      args:
      - -u
      - %s:%s
      - -s
      - "http://%s/spider/cluster/%s/token?ConnectionName=%s"
`, *cluster.Endpoint, *cluster.CertificateAuthority.Data, *cluster.Name, *cluster.Name, *cluster.Name, *cluster.Name, authUsername, authPassword, serverAddr, *cluster.Name, connectionName)

	return kubeconfigContent
}

// create EKS token using AWS STS
func (ClusterHandler *AwsClusterHandler) generateEKSToken(clusterName string) (string, error) {
	if ClusterHandler.StsClient == nil {
		return "", fmt.Errorf("STS client not available")
	}

	// create request for GetCallerIdentity
	input := &sts.GetCallerIdentityInput{}
	req, _ := ClusterHandler.StsClient.GetCallerIdentityRequest(input)

	req.HTTPRequest.Header.Set("x-k8s-aws-id", clusterName)

	// set 15 minutes expiration for presigned URL(max is 1 hour)
	duration := EKS_TOKEN_DURATION
	presignedURL, err := req.Presign(duration)
	if err != nil {
		cblogger.Errorf("Failed to create presigned URL: %v", err)
		return "", fmt.Errorf("failed to create presigned URL: %w", err)
	}

	encodedURL := base64.RawURLEncoding.EncodeToString([]byte(presignedURL))

	// prepend "k8s-aws-v1." to the encoded URL
	token := "k8s-aws-v1." + encodedURL

	return token, nil
}

// GenerateClusterToken generates a token for cluster authentication
// This implements the ClusterHandler interface
func (ClusterHandler *AwsClusterHandler) GenerateClusterToken(clusterIID irs.IID) (string, error) {
	cblogger.Info("call GenerateClusterToken()")

	// For EKS, we need the cluster name to generate token
	clusterName := clusterIID.SystemId
	if clusterName == "" {
		clusterName = clusterIID.NameId
	}

	if clusterName == "" {
		return "", fmt.Errorf("cluster name is required for token generation")
	}

	// Generate EKS token using existing function
	token, err := ClusterHandler.generateEKSToken(clusterName)
	if err != nil {
		cblogger.Errorf("Failed to generate cluster token: %v", err)
		return "", fmt.Errorf("failed to generate cluster token: %w", err)
	}

	return token, nil
}

func (ClusterHandler *AwsClusterHandler) DeleteCluster(clusterIID irs.IID) (bool, error) {
	cblogger.Infof("Cluster Name : %s", clusterIID.SystemId)
	input := &eks.DeleteClusterInput{
		Name: aws.String(clusterIID.SystemId),
	}

	cblogger.Debug(input)

	// logger for HisCall
	callogger := call.GetLogger("HISCALL")
	callLogInfo := call.CLOUDLOGSCHEMA{
		CloudOS:      call.AWS,
		RegionZone:   ClusterHandler.Region.Zone,
		ResourceType: call.CLUSTER,
		ResourceName: clusterIID.SystemId,
		CloudOSAPI:   "DeleteCluster()",
		ElapsedTime:  "",
		ErrorMSG:     "",
	}
	callLogStart := call.Start()

	result, err := ClusterHandler.Client.DeleteCluster(input)
	callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		callogger.Info(call.String(callLogInfo))

		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case eks.ErrCodeResourceInUseException:
				cblogger.Error(eks.ErrCodeResourceInUseException, aerr.Error())
			case eks.ErrCodeResourceNotFoundException:
				cblogger.Error(eks.ErrCodeResourceNotFoundException, aerr.Error())
			case eks.ErrCodeClientException:
				cblogger.Error(eks.ErrCodeClientException, aerr.Error())
			case eks.ErrCodeServerException:
				cblogger.Error(eks.ErrCodeServerException, aerr.Error())
			case eks.ErrCodeServiceUnavailableException:
				cblogger.Error(eks.ErrCodeServiceUnavailableException, aerr.Error())
			default:
				cblogger.Error(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			cblogger.Error(err.Error())
		}
		return false, err
	}
	callogger.Info(call.String(callLogInfo))

	cblogger.Debug(result)
	/*
		waitInput := &eks.DescribeClusterInput{
			Name: aws.String(clusterIID.SystemId),
		}
		err = ClusterHandler.Client.WaitUntilClusterDeleted(waitInput)
		if err != nil {
			return false, err
		}
	*/

	return true, nil
}

// ------ NodeGroup Management

/*
Cluster.NetworkInfo 설정과 동일 서브넷으로 설정
NodeGroup 추가시에는 대상 Cluster 정보 획득하여 설정
NodeGroup에 다른 Subnet 설정이 꼭 필요시 추후 재논의
//https://github.com/cloud-barista/cb-spider/wiki/Provider-Managed-Kubernetes-and-Driver-API
*/
func (ClusterHandler *AwsClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (irs.NodeGroupInfo, error) {
	// validation check
	if nodeGroupReqInfo.MaxNodeSize < 1 { // nodeGroupReqInfo.MaxNodeSize 는 최소가 1이다.
		return irs.NodeGroupInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "The MaxNodeSize value must be greater than or equal to 1.", nil)
	}

	// get or create Role Arn
	eksRoleName := "cloud-barista-eks-nodegroup-role"
	eksRole, err := ClusterHandler.getOrCreateEKSNodeGroupRole(eksRoleName)
	if err != nil {
		cblogger.Error(err)
		// role 은 required 임.
		return irs.NodeGroupInfo{}, fmt.Errorf("failed to get or create EKS node group role: %w", err)
	}
	roleArn := eksRole.Role.Arn

	clusterInfo, err := ClusterHandler.GetCluster(clusterIID)
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
		input := &ec2.ModifySubnetAttributeInput{
			MapPublicIpOnLaunch: &ec2.AttributeBooleanValue{
				Value: aws.Bool(true),
			},
			SubnetId: subnetIdPtr,
		}
		_, err := ClusterHandler.EC2Client.ModifySubnetAttribute(input)
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

	// ============================================================
	// AMI Analysis & EKS AMI Type Mapping
	// ============================================================

	// Received from Tumblebug (CspImageName from PostgreSQL)
	receivedImageName := nodeGroupReqInfo.ImageIID.SystemId
	cblogger.Infof("Received ImageName from Tumblebug: '%s'", receivedImageName)

	var amiType string

	if receivedImageName == "" {
		// No AMI specified, use default (K8s 1.30+ compatible)
		cblogger.Info("No ImageName provided, using default: AL2023_x86_64_STANDARD")
		amiType = "AL2023_x86_64_STANDARD"
	} else {
		// Step 1: Analyze received AMI
		cblogger.Infof("Analyzing AMI: %s", receivedImageName)
		amiInfo, err := ClusterHandler.analyzeAMI(receivedImageName)
		if err != nil {
			cblogger.Warnf("Failed to analyze AMI %s: %v. Using default (K8s 1.30+ compatible).", receivedImageName, err)
			amiType = "AL2023_x86_64_STANDARD"
		} else {
			// Step 2: Map to EKS AMI Type
			instanceType := nodeGroupReqInfo.VMSpecName
			if instanceType == "" {
				instanceType = "t3.medium" // default
			}

			amiType = ClusterHandler.mapToEKSAMIType(amiInfo, instanceType)
			cblogger.Infof("Final EKS AMI Type: %s", amiType)
		}
	}

	cblogger.Infof("Creating NodeGroup with AmiType: %s", amiType)

	// ============================================================

	tags := map[string]string{}
	tags["key"] = NODEGROUP_TAG
	tags["value"] = nodeGroupReqInfo.IId.NameId

	input := &eks.CreateNodegroupInput{
		// Set mapped AMI type - AWS EKS automatically uses the latest EKS-optimized AMI for this type
		AmiType: aws.String(amiType),
		//CapacityType: aws.String("ON_DEMAND"),//Valid Values: ON_DEMAND | SPOT, Required: No

		//ClusterName:   aws.String("cb-eks-cluster"),              //uri, required
		ClusterName:   aws.String(clusterIID.SystemId),         //uri, required
		NodegroupName: aws.String(nodeGroupReqInfo.IId.NameId), // required
		Tags:          aws.StringMap(tags),
		//NodeRole:      aws.String(eksRoleName), // roleName, required
		NodeRole: roleArn,
		ScalingConfig: &eks.NodegroupScalingConfig{
			DesiredSize: aws.Int64(int64(nodeGroupReqInfo.DesiredNodeSize)),
			MaxSize:     aws.Int64(int64(nodeGroupReqInfo.MaxNodeSize)),
			MinSize:     aws.Int64(int64(nodeGroupReqInfo.MinNodeSize)),
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
		RemoteAccess: &eks.RemoteAccessConfig{
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
		input.DiskSize = aws.Int64(rootDiskSize)
	}

	if !strings.EqualFold(nodeGroupReqInfo.VMSpecName, "") {
		var nodeSpec []string
		nodeSpec = append(nodeSpec, nodeGroupReqInfo.VMSpecName) //"p2.xlarge"
		input.InstanceTypes = aws.StringSlice(nodeSpec)
	}

	cblogger.Debug(input)

	result, err := ClusterHandler.Client.CreateNodegroup(input) // 비동기
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}

	cblogger.Debug(result)

	nodegroupName := result.Nodegroup.NodegroupName

	// To modify instances' metadata options, wait until nodegroup is active: #1487
	errWait := ClusterHandler.WaitUntilNodegroupActive(clusterIID.SystemId, *nodegroupName)
	if errWait != nil {
		cblogger.Error(errWait)
		return irs.NodeGroupInfo{}, errWait
	}

	nodeGroup, err := ClusterHandler.GetNodeGroup(clusterIID, irs.IID{NameId: nodeGroupReqInfo.IId.NameId, SystemId: *nodegroupName})
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}
	nodeGroup.IId.NameId = nodeGroupReqInfo.IId.NameId

	// Update HttpPutReponseHopLimit as 2: #1487
	for _, node := range nodeGroup.Nodes {
		err = ModifyInstanceMetadataOptionsHttpPutResponseHopLimit(ClusterHandler.EC2Client, node.SystemId, httpPutResponseHopLimitIs2)
		if err != nil {
			cblogger.Error(err)
		}
	}

	return nodeGroup, nil
}

// installs the EBS CSI driver and pod identity agent only if they doesn't exist
func (ClusterHandler *AwsClusterHandler) InstallEBSCSIDriverAndPodIdentityAgentIfNotExists(clusterID string) error {
	// Check if EBS CSI driver already exists
	addonListParams := &eks.ListAddonsInput{
		ClusterName: aws.String(clusterID),
	}

	addonList, err := ClusterHandler.Client.ListAddons(addonListParams)
	if err != nil {
		cblogger.Errorf("Failed to list addons: %v", err)
		return err
	}

	// Check if EBS CSI driver is already installed
	csiDriverExists := false
	podIdentityAgentExists := false
	for _, addon := range addonList.Addons {
		if aws.StringValue(addon) == "aws-ebs-csi-driver" {
			csiDriverExists = true
		} else if aws.StringValue(addon) == "eks-pod-identity-agent" {
			podIdentityAgentExists = true
		}
	}

	// Install CSI driver and pod identity agent only if it doesn't exist
	if csiDriverExists && podIdentityAgentExists {
		cblogger.Infof("EBS CSI driver and pod identity agent already exist in cluster %s", clusterID)
		return nil
	}

	var errCsiDriver error
	if csiDriverExists == false {
		errCsiDriver = ClusterHandler.installEBSCSIDriver(clusterID)
		if errCsiDriver != nil {
			cblogger.Errorf("Failed to install EBS CSI driver: %w", errCsiDriver)
		}
	}

	var errPodIdentityAgent error
	if podIdentityAgentExists == false {
		errPodIdentityAgent = ClusterHandler.installPodIdentityAgent(clusterID)
		if errPodIdentityAgent != nil {
			cblogger.Errorf("Failed to install pod identity agent: %w", errPodIdentityAgent)
		}
	}

	if errCsiDriver != nil || errPodIdentityAgent != nil {
		return errors.Join(errCsiDriver, errPodIdentityAgent)
	}

	return nil
}

func (ClusterHandler *AwsClusterHandler) installEBSCSIDriver(clusterID string) error {
	addonName := "aws-ebs-csi-driver" // EBS CSI Driver, don't change

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	input := &eks.CreateAddonInput{
		ClusterName:      aws.String(clusterID),
		AddonName:        aws.String(addonName),
		ResolveConflicts: aws.String("OVERWRITE"),
	}

	_, err := ClusterHandler.Client.CreateAddonWithContext(ctx, input)
	if err != nil {
		cblogger.Errorf("Failed to install EBS CSI Driver: %w", err)
		return err
	}
	return nil
}

func (ClusterHandler *AwsClusterHandler) installPodIdentityAgent(clusterID string) error {
	addonName := "eks-pod-identity-agent" // Eks pod identity agent, don't change

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	input := &eks.CreateAddonInput{
		ClusterName:      aws.String(clusterID),
		AddonName:        aws.String(addonName),
		ResolveConflicts: aws.String("OVERWRITE"),
	}

	_, err := ClusterHandler.Client.CreateAddonWithContext(ctx, input)
	if err != nil {
		cblogger.Errorf("Failed to install pod identity agent: %w", err)
		return err
	}
	return nil
}

func (ClusterHandler *AwsClusterHandler) ListNodeGroup(clusterIID irs.IID) ([]*irs.NodeGroupInfo, error) {
	input := &eks.ListNodegroupsInput{
		ClusterName: aws.String(clusterIID.SystemId),
	}
	cblogger.Debug(input)

	result, err := ClusterHandler.Client.ListNodegroups(input)
	if err != nil {
		cblogger.Error(err)
		return nil, err
	}
	cblogger.Debug(result)
	nodeGroupInfoList := []*irs.NodeGroupInfo{}
	for _, nodeGroupName := range result.Nodegroups {
		nodeGroupInfo, err := ClusterHandler.GetNodeGroup(clusterIID, irs.IID{SystemId: *nodeGroupName})
		if err != nil {
			cblogger.Error(err)
			//return nil, err
			continue
		}
		nodeGroupInfoList = append(nodeGroupInfoList, &nodeGroupInfo)
	}
	return nodeGroupInfoList, nil
}

func (ClusterHandler *AwsClusterHandler) GetNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (irs.NodeGroupInfo, error) {
	cblogger.Debugf("Cluster SystemId : [%s] / NodeGroup SystemId : [%s]", clusterIID.SystemId, nodeGroupIID.SystemId)

	input := &eks.DescribeNodegroupInput{
		//AmiType: "", // Valid Values: AL2_x86_64 | AL2_x86_64_GPU | AL2_ARM_64 | CUSTOM | BOTTLEROCKET_ARM_64 | BOTTLEROCKET_x86_64, Required: No
		//CapacityType: aws.String("ON_DEMAND"),//Valid Values: ON_DEMAND | SPOT, Required: No
		ClusterName:   aws.String(clusterIID.SystemId),   //required
		NodegroupName: aws.String(nodeGroupIID.SystemId), // required
	}

	result, err := ClusterHandler.Client.DescribeNodegroup(input)
	cblogger.Debug("===> Node Group Invocation Result")
	cblogger.Debug(result)
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}

	nodeGroupInfo, err := ClusterHandler.convertNodeGroup(result)
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}
	return nodeGroupInfo, nil
}

func (ClusterHandler *AwsClusterHandler) GetAutoScalingGroups(autoScalingGroupName string) ([]irs.IID, error) {
	input := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{
			aws.String(autoScalingGroupName),
		},
	}

	result, err := ClusterHandler.AutoScaling.DescribeAutoScalingGroups(input)
	cblogger.Debug(result)

	if err != nil {
		cblogger.Error(err)
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case autoscaling.ErrCodeInvalidNextToken:
				cblogger.Error(autoscaling.ErrCodeInvalidNextToken, aerr.Error())
			case autoscaling.ErrCodeResourceContentionFault:
				cblogger.Error(autoscaling.ErrCodeResourceContentionFault, aerr.Error())
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
}

/*
AutoScaling 이라는 별도의 메뉴가 있음.
*/
func (ClusterHandler *AwsClusterHandler) SetNodeGroupAutoScaling(clusterIID irs.IID, nodeGroupIID irs.IID, on bool) (bool, error) {
	return false, nil
}

func (ClusterHandler *AwsClusterHandler) ChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID,
	DesiredNodeSize int, MinNodeSize int, MaxNodeSize int) (irs.NodeGroupInfo, error) {
	cblogger.Infof("Cluster SystemId : [%s] / NodeGroup SystemId : [%s] / DesiredNodeSize : [%d] / MinNodeSize : [%d] / MaxNodeSize : [%d]", clusterIID.SystemId, nodeGroupIID.SystemId, DesiredNodeSize, MinNodeSize, MaxNodeSize)

	// clusterIID로 cluster 정보를 조회
	// nodeGroupIID로 nodeGroup 정보를 조회
	// 		nodeGroup에 AutoScaling 그룹 이름이 있음.

	// TODO : 공통으로 뺄 것
	input := &eks.DescribeNodegroupInput{
		ClusterName:   aws.String(clusterIID.SystemId),   //required
		NodegroupName: aws.String(nodeGroupIID.SystemId), // required
	}

	result, err := ClusterHandler.Client.DescribeNodegroup(input)
	cblogger.Debug(result.Nodegroup)
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}

	nodeGroupName := result.Nodegroup.NodegroupName
	nodeGroupResources := result.Nodegroup.Resources.AutoScalingGroups
	for _, autoScalingGroup := range nodeGroupResources {
		input := &autoscaling.UpdateAutoScalingGroupInput{
			AutoScalingGroupName: aws.String(*autoScalingGroup.Name),

			MaxSize:         aws.Int64(int64(MaxNodeSize)),
			MinSize:         aws.Int64(int64(MinNodeSize)),
			DesiredCapacity: aws.Int64(int64(DesiredNodeSize)),
		}

		updateResult, err := ClusterHandler.AutoScaling.UpdateAutoScalingGroup(input)
		if err != nil {
			cblogger.Error(err)
			return irs.NodeGroupInfo{}, err
		}
		cblogger.Debug(updateResult)
	}

	nodeGroupInfo, err := ClusterHandler.GetNodeGroup(clusterIID, irs.IID{SystemId: *nodeGroupName})
	if err != nil {
		cblogger.Error(err)
		return irs.NodeGroupInfo{}, err
	}
	return nodeGroupInfo, nil
}

func (ClusterHandler *AwsClusterHandler) RemoveNodeGroup(clusterIID irs.IID, nodeGroupIID irs.IID) (bool, error) {
	cblogger.Infof("Cluster SystemId : [%s] / NodeGroup SystemId : [%s]", clusterIID.SystemId, nodeGroupIID.SystemId)

	input := &eks.DeleteNodegroupInput{
		ClusterName:   aws.String(clusterIID.SystemId),   //required
		NodegroupName: aws.String(nodeGroupIID.SystemId), // required
	}

	result, err := ClusterHandler.Client.DeleteNodegroup(input)
	if err != nil {
		cblogger.Error(err)
		return false, err
	}

	cblogger.Debug(result)

	return true, nil
}

// Control Plane 과 Node Group(worked node)의 K8s 버전 업그레이드드
func (ClusterHandler *AwsClusterHandler) UpgradeCluster(clusterIID irs.IID, newVersion string) (irs.ClusterInfo, error) {
	cblogger.Infof("Cluster SystemId : [%s] / Request New Version : [%s]", clusterIID.SystemId, newVersion)

	// 현재 클러스터 정보 조회 및 버전 비교
	currentClusterInfo, err := ClusterHandler.GetCluster(clusterIID)
	if err != nil {
		cblogger.Errorf("Failed to get current cluster info: %v", err)
		return irs.ClusterInfo{}, err
	}

	currentVersion := currentClusterInfo.Version
	cblogger.Infof("Current cluster version: %s, Target version: %s", currentVersion, newVersion)

	// Control Plane이 업그레이드 필요한지 확인
	needsControlPlaneUpgrade := currentVersion != newVersion

	// 업그레이드가 필요한 Node Group이 있는지 확인하기 위해 모든 Node Group 조회
	nodeGroups, err := ClusterHandler.ListNodeGroup(clusterIID)
	if err != nil {
		cblogger.Errorf("Failed to list node groups: %v", err)
		return irs.ClusterInfo{}, err
	}

	// Node Group 업그레이드 필요 여부 확인
	needsNodeGroupUpgrade := false
	for _, nodeGroup := range nodeGroups {
		// Node Group 버전을 확인
		nodeGroupVersion := ""
		for _, kv := range nodeGroup.KeyValueList {
			if kv.Key == "Version" {
				nodeGroupVersion = kv.Value
				break
			}
		}

		// 버전을 결정할 수 없거나 대상 버전과 다른 경우
		if nodeGroupVersion == "" || nodeGroupVersion != newVersion {
			needsNodeGroupUpgrade = true
			cblogger.Infof("Node group %s needs upgrade from version %s to %s",
				nodeGroup.IId.SystemId, nodeGroupVersion, newVersion)
		}
	}

	// 업그레이드가 필요 없는 경우 조기 반환
	if !needsControlPlaneUpgrade && !needsNodeGroupUpgrade {
		cblogger.Info("Both control plane and all node groups are already at target version. No upgrade needed.")
		return currentClusterInfo, nil
	}

	if needsControlPlaneUpgrade {
		cblogger.Infof("Control plane needs upgrade from version %s to %s", currentVersion, newVersion)
	} else {
		cblogger.Info("Control plane is already at target version. Only node groups will be upgraded.")
	}

	// Control Plane 업그레이드
	if needsControlPlaneUpgrade {
		input := &eks.UpdateClusterVersionInput{
			Name:    aws.String(clusterIID.SystemId),
			Version: aws.String(newVersion),
		}

		cblogger.Debug("Upgrading control plane with input:", input)

		// logger for HisCall
		callogger := call.GetLogger("HISCALL")
		callLogInfo := call.CLOUDLOGSCHEMA{
			CloudOS:      call.AWS,
			RegionZone:   ClusterHandler.Region.Zone,
			ResourceType: call.CLUSTER,
			ResourceName: clusterIID.SystemId,
			CloudOSAPI:   "UpdateClusterVersion()",
			ElapsedTime:  "",
			ErrorMSG:     "",
		}
		callLogStart := call.Start()

		result, err := ClusterHandler.Client.UpdateClusterVersion(input)
		callLogInfo.ElapsedTime = call.Elapsed(callLogStart)

		if err != nil {
			callLogInfo.ErrorMSG = err.Error()
			callogger.Info(call.String(callLogInfo))

			if aerr, ok := err.(awserr.Error); ok {
				switch aerr.Code() {
				case eks.ErrCodeInvalidParameterException:
					cblogger.Error(eks.ErrCodeInvalidParameterException, aerr.Error())
				case eks.ErrCodeClientException:
					cblogger.Error(eks.ErrCodeClientException, aerr.Error())
				case eks.ErrCodeResourceNotFoundException:
					cblogger.Error(eks.ErrCodeResourceNotFoundException, aerr.Error())
				case eks.ErrCodeServerException:
					cblogger.Error(eks.ErrCodeServerException, aerr.Error())
				case eks.ErrCodeInvalidRequestException:
					cblogger.Error(eks.ErrCodeInvalidRequestException, aerr.Error())
				default:
					cblogger.Error(aerr.Error())
				}
			} else {
				cblogger.Error(err.Error())
			}
			return irs.ClusterInfo{}, err
		}
		callogger.Info(call.String(callLogInfo))
		cblogger.Infof("Control plane upgrade initiated: %s", *result.Update.Id)

		// Control Plane 업그레이드 완료 대기
		cblogger.Info("Waiting for control plane upgrade to complete...")
		errWait := ClusterHandler.WaitUntilClusterActive(clusterIID.SystemId)
		if errWait != nil {
			cblogger.Errorf("Failed to wait for cluster to become active after control plane upgrade: %v", errWait)
			// Control Plane 대기 실패해도 Node Group 업그레이드 계속 진행
		} else {
			cblogger.Info("Control plane upgrade completed successfully")
		}

		// [추가] Control Plane 업그레이드 후 추가 대기 시간
		cblogger.Info("Control plane marked as ACTIVE, waiting additional time for version propagation...")
		time.Sleep(120 * time.Second) // 2분 대기

		// [추가] Control Plane 버전 재확인
		checkInput := &eks.DescribeClusterInput{
			Name: aws.String(clusterIID.SystemId),
		}
		checkResult, checkErr := ClusterHandler.Client.DescribeCluster(checkInput)
		if checkErr != nil {
			cblogger.Errorf("Failed to verify cluster version: %v", checkErr)
		} else {
			cblogger.Infof("Verified control plane version: %s", *checkResult.Cluster.Version)
			if *checkResult.Cluster.Version != newVersion {
				cblogger.Warnf("Control plane version mismatch: expected %s, found %s",
					newVersion, *checkResult.Cluster.Version)
				// 버전이 맞지 않더라도 진행은 함
			}
		}
	}

	// Node Group 업그레이드 전 활성 작업 체크 및 백오프 재시도 로직 추가
	maxRetries := 10
	retryInterval := 60 // seconds
	backoffFactor := 1.5

	currentInterval := retryInterval
	for i := 0; i < maxRetries; i++ {
		// 클러스터가 Node Group 업그레이드를 할 수 있는 상태인지 확인
		clusterStatus, err := ClusterHandler.checkClusterOperationStatus(clusterIID)
		if err != nil {
			cblogger.Errorf("Error checking cluster status: %v", err)
			break // 상태 확인 실패해도 Node Group 업그레이드 계속 진행
		}

		if clusterStatus == "ACTIVE" {
			break // 노드 그룹 업그레이드 진행
		}

		if i == maxRetries-1 {
			cblogger.Warnf("Cluster not in ACTIVE state after %d retries. Proceeding with node group upgrades anyway.", maxRetries)
			break
		}

		cblogger.Infof("Cluster is in %s state, waiting %d seconds before retrying (%d/%d)",
			clusterStatus, currentInterval, i+1, maxRetries)
		time.Sleep(time.Duration(currentInterval) * time.Second)

		// 지수 백오프 적용
		currentInterval = int(float64(currentInterval) * backoffFactor)
	}

	// Node Group 업그레이드
	if needsNodeGroupUpgrade {
		cblogger.Info("Starting node group upgrades...")

		// Node Group 업그레이드 실패 추적을 목록
		upgradeFailures := []string{}

		for _, nodeGroup := range nodeGroups {
			cblogger.Infof("Upgrading node group: %s to version: %s", nodeGroup.IId.SystemId, newVersion)

			// [추가] Node Group 업그레이드에 재시도 로직 추가
			maxNodeGroupRetries := 3
			var updateErr error
			var nodeGroupUpdateResult *eks.UpdateNodegroupVersionOutput

			for retry := 0; retry < maxNodeGroupRetries; retry++ {
				updateNodeGroupInput := &eks.UpdateNodegroupVersionInput{
					ClusterName:   aws.String(clusterIID.SystemId),
					NodegroupName: aws.String(nodeGroup.IId.SystemId),
					Version:       aws.String(newVersion),
				}

				callogger := call.GetLogger("HISCALL")
				nodeGroupCallLogInfo := call.CLOUDLOGSCHEMA{
					CloudOS:      call.AWS,
					RegionZone:   ClusterHandler.Region.Zone,
					ResourceType: call.CLUSTER,
					ResourceName: nodeGroup.IId.SystemId,
					CloudOSAPI:   "UpdateNodegroupVersion()",
					ElapsedTime:  "",
					ErrorMSG:     "",
				}
				nodeGroupCallLogStart := call.Start()

				nodeGroupUpdateResult, updateErr = ClusterHandler.Client.UpdateNodegroupVersion(updateNodeGroupInput)
				nodeGroupCallLogInfo.ElapsedTime = call.Elapsed(nodeGroupCallLogStart)

				if updateErr == nil {
					// 업그레이드 요청 성공
					callogger.Info(call.String(nodeGroupCallLogInfo))
					cblogger.Infof("Node group update initiated for %s: %s",
						nodeGroup.IId.SystemId, *nodeGroupUpdateResult.Update.Id)
					break
				}

				// 에러 발생 시 로그 기록
				nodeGroupCallLogInfo.ErrorMSG = updateErr.Error()
				callogger.Info(call.String(nodeGroupCallLogInfo))
				cblogger.Errorf("Attempt %d: Failed to upgrade node group %s: %v",
					retry+1, nodeGroup.IId.SystemId, updateErr)

				if retry < maxNodeGroupRetries-1 {
					retryDelay := (retry + 1) * 60 // 점진적 증가: 60초, 120초, ...
					cblogger.Infof("Retrying node group upgrade in %d seconds (%d/%d)",
						retryDelay, retry+1, maxNodeGroupRetries)
					time.Sleep(time.Duration(retryDelay) * time.Second)
				}
			}

			if updateErr != nil {
				// 모든 재시도 후에도 실패
				cblogger.Errorf("Failed to upgrade node group %s after %d attempts: %v",
					nodeGroup.IId.SystemId, maxNodeGroupRetries, updateErr)
				upgradeFailures = append(upgradeFailures,
					fmt.Sprintf("NodeGroup %s: %v", nodeGroup.IId.SystemId, updateErr))
				continue // 다른 Node Group은 업그레이드를 계속 진행
			}

			// 하나씩 순차적으로 Node Group 업그레이드를 진행 (병렬 처리 X)
			// Control Plane보다 짧은 타임아웃을 설정하여 Node Group 업그레이드 처리
			cblogger.Infof("Waiting for node group %s upgrade to complete...", nodeGroup.IId.SystemId)
			errWaitNodeGroup := ClusterHandler.WaitUntilNodegroupActive(clusterIID.SystemId, nodeGroup.IId.SystemId)
			if errWaitNodeGroup != nil {
				cblogger.Errorf("Failed to wait for node group to become active: %v", errWaitNodeGroup)
				upgradeFailures = append(upgradeFailures,
					fmt.Sprintf("Waiting for NodeGroup %s: %v", nodeGroup.IId.SystemId, errWaitNodeGroup))
			} else {
				cblogger.Infof("Node group %s upgraded successfully", nodeGroup.IId.SystemId)
			}
		}

		// 실패 정보 종합 로깅
		if len(upgradeFailures) > 0 {
			cblogger.Warnf("Some node groups failed to upgrade: %s", strings.Join(upgradeFailures, "; "))
		} else {
			cblogger.Info("All node group upgrades completed successfully")
		}
	} else {
		cblogger.Info("No node groups need upgrading")
	}

	// 최종 상태를 확인하고 업데이트된 클러스터 정보를 반환
	cblogger.Info("Fetching updated cluster information")
	updatedClusterInfo, err := ClusterHandler.GetCluster(clusterIID)
	if err != nil {
		cblogger.Errorf("Failed to get updated cluster info: %v", err)
		return irs.ClusterInfo{}, err
	}

	// 업그레이드 결과 확인
	if needsControlPlaneUpgrade && updatedClusterInfo.Version != newVersion {
		cblogger.Warnf("Control plane upgrade may not have completed. Expected version: %s, Actual version: %s",
			newVersion, updatedClusterInfo.Version)
	} else {
		cblogger.Info("Cluster upgrade completed successfully")
	}

	return updatedClusterInfo, nil
}

// Helper function to check if cluster has active operations
func (ClusterHandler *AwsClusterHandler) checkClusterOperationStatus(clusterIID irs.IID) (string, error) {
	input := &eks.DescribeClusterInput{
		Name: aws.String(clusterIID.SystemId),
	}

	result, err := ClusterHandler.Client.DescribeCluster(input)
	if err != nil {
		return "", err
	}

	return *result.Cluster.Status, nil
}

func (ClusterHandler *AwsClusterHandler) getRole(role irs.IID) (*iam.GetRoleOutput, error) {
	input := &iam.GetRoleInput{
		RoleName: aws.String(role.SystemId),
	}

	result, err := ClusterHandler.Iam.GetRole(input)
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

// getOrCreateEKSClusterRole gets or creates EKS cluster role if it doesn't exist
func (ClusterHandler *AwsClusterHandler) getOrCreateEKSClusterRole(roleName string) (*iam.GetRoleOutput, error) {
	cblogger.Infof("Getting or creating EKS cluster role: %s", roleName)

	// Try to get existing role (without logging error if not found)
	input := &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	}

	result, err := ClusterHandler.Iam.GetRole(input)
	if err == nil {
		cblogger.Infof("EKS cluster role already exists: %s", roleName)
		return result, nil
	}

	// Check if error is NoSuchEntityException
	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() != iam.ErrCodeNoSuchEntityException {
			// Log error only if it's not NoSuchEntityException
			cblogger.Error(aerr.Error())
			return nil, fmt.Errorf("failed to get role: %w", err)
		}
		// NoSuchEntityException is expected, don't log as error
		cblogger.Debugf("Role %s does not exist, will create it", roleName)
	} else {
		cblogger.Error(err.Error())
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	// Role doesn't exist, create it
	cblogger.Infof("Creating EKS cluster role: %s", roleName)
	err = ClusterHandler.createEKSClusterRole(roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to create EKS cluster role: %w", err)
	}

	// Get the newly created role
	result, err = ClusterHandler.Iam.GetRole(input)
	if err != nil {
		return nil, fmt.Errorf("failed to get newly created role: %w", err)
	}

	cblogger.Infof("Successfully created and retrieved EKS cluster role: %s", roleName)
	return result, nil
}

// getOrCreateEKSNodeGroupRole gets or creates EKS node group role if it doesn't exist
func (ClusterHandler *AwsClusterHandler) getOrCreateEKSNodeGroupRole(roleName string) (*iam.GetRoleOutput, error) {
	cblogger.Infof("Getting or creating EKS node group role: %s", roleName)

	// Try to get existing role (without logging error if not found)
	input := &iam.GetRoleInput{
		RoleName: aws.String(roleName),
	}

	result, err := ClusterHandler.Iam.GetRole(input)
	if err == nil {
		cblogger.Infof("EKS node group role already exists: %s", roleName)
		return result, nil
	}

	// Check if error is NoSuchEntityException
	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() != iam.ErrCodeNoSuchEntityException {
			// Log error only if it's not NoSuchEntityException
			cblogger.Error(aerr.Error())
			return nil, fmt.Errorf("failed to get role: %w", err)
		}
		// NoSuchEntityException is expected, don't log as error
		cblogger.Debugf("Role %s does not exist, will create it", roleName)
	} else {
		cblogger.Error(err.Error())
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	// Role doesn't exist, create it
	cblogger.Infof("Creating EKS node group role: %s", roleName)
	err = ClusterHandler.createEKSNodeGroupRole(roleName)
	if err != nil {
		return nil, fmt.Errorf("failed to create EKS node group role: %w", err)
	}

	// Get the newly created role
	result, err = ClusterHandler.Iam.GetRole(input)
	if err != nil {
		return nil, fmt.Errorf("failed to get newly created role: %w", err)
	}

	cblogger.Infof("Successfully created and retrieved EKS node group role: %s", roleName)
	return result, nil
}

// createEKSClusterRole creates IAM role for EKS cluster with required policies
func (ClusterHandler *AwsClusterHandler) createEKSClusterRole(roleName string) error {
	cblogger.Infof("Creating EKS cluster role: %s", roleName)

	// Trust policy for EKS service
	trustPolicy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"Service": "eks.amazonaws.com"
				},
				"Action": "sts:AssumeRole"
			}
		]
	}`

	// Create role
	createRoleInput := &iam.CreateRoleInput{
		RoleName:                 aws.String(roleName),
		AssumeRolePolicyDocument: aws.String(trustPolicy),
		Description:              aws.String("IAM role for Cloud-Barista EKS cluster"),
		Tags: []*iam.Tag{
			{
				Key:   aws.String("ManagedBy"),
				Value: aws.String("Cloud-Barista-Spider"),
			},
		},
	}

	_, err := ClusterHandler.Iam.CreateRole(createRoleInput)
	if err != nil {
		cblogger.Errorf("Failed to create role: %v", err)
		return fmt.Errorf("failed to create role: %w", err)
	}

	cblogger.Infof("Successfully created role: %s", roleName)

	// Attach required policy: AmazonEKSClusterPolicy
	policyArn := "arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"
	attachPolicyInput := &iam.AttachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String(policyArn),
	}

	_, err = ClusterHandler.Iam.AttachRolePolicy(attachPolicyInput)
	if err != nil {
		cblogger.Errorf("Failed to attach policy %s: %v", policyArn, err)
		return fmt.Errorf("failed to attach policy %s: %w", policyArn, err)
	}

	cblogger.Infof("Successfully attached policy %s to role %s", policyArn, roleName)

	// Wait a bit for IAM changes to propagate
	cblogger.Info("Waiting for IAM changes to propagate...")
	time.Sleep(10 * time.Second)

	return nil
}

// createEKSNodeGroupRole creates IAM role for EKS node group with required policies
func (ClusterHandler *AwsClusterHandler) createEKSNodeGroupRole(roleName string) error {
	cblogger.Infof("Creating EKS node group role: %s", roleName)

	// Trust policy for EC2 service
	trustPolicy := `{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"Service": "ec2.amazonaws.com"
				},
				"Action": "sts:AssumeRole"
			}
		]
	}`

	// Create role
	createRoleInput := &iam.CreateRoleInput{
		RoleName:                 aws.String(roleName),
		AssumeRolePolicyDocument: aws.String(trustPolicy),
		Description:              aws.String("IAM role for Cloud-Barista EKS node group"),
		Tags: []*iam.Tag{
			{
				Key:   aws.String("ManagedBy"),
				Value: aws.String("Cloud-Barista-Spider"),
			},
		},
	}

	_, err := ClusterHandler.Iam.CreateRole(createRoleInput)
	if err != nil {
		cblogger.Errorf("Failed to create role: %v", err)
		return fmt.Errorf("failed to create role: %w", err)
	}

	cblogger.Infof("Successfully created role: %s", roleName)

	// Attach required policies
	requiredPolicies := []string{
		"arn:aws:iam::aws:policy/AmazonEKSWorkerNodePolicy",
		"arn:aws:iam::aws:policy/AmazonEKS_CNI_Policy",
		"arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryPullOnly",
		"arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy",
	}

	for _, policyArn := range requiredPolicies {
		attachPolicyInput := &iam.AttachRolePolicyInput{
			RoleName:  aws.String(roleName),
			PolicyArn: aws.String(policyArn),
		}

		_, err = ClusterHandler.Iam.AttachRolePolicy(attachPolicyInput)
		if err != nil {
			cblogger.Errorf("Failed to attach policy %s: %v", policyArn, err)
			return fmt.Errorf("failed to attach policy %s: %w", policyArn, err)
		}

		cblogger.Infof("Successfully attached policy %s to role %s", policyArn, roleName)
	}

	// Wait a bit for IAM changes to propagate
	cblogger.Info("Waiting for IAM changes to propagate...")
	time.Sleep(10 * time.Second)

	return nil
}

/*
EKS의 NodeGroup정보를 Spider의 NodeGroup으로 변경
*/
func (NodeGroupHandler *AwsClusterHandler) convertNodeGroup(nodeGroupOutput *eks.DescribeNodegroupOutput) (irs.NodeGroupInfo, error) {
	nodeGroupInfo := irs.NodeGroupInfo{}
	PrintToJson(nodeGroupOutput)

	nodeGroup := nodeGroupOutput.Nodegroup
	//nodeRole := nodeGroup.NodeRole
	//version := nodeGroup.Version
	//releaseVersion := nodeGroup.ReleaseVersion

	//subnetList := nodeGroup.Subnets
	//nodeGroupStatus := nodeGroup.Status
	nodeGroupInfo.Status = convertNodeGroupStatusToNodeGroupInfoStatus(aws.StringValue(nodeGroup.Status))
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

	/*
		nodes := []irs.IID{}
		for _, issue := range nodeGroup.Health.Issues {
			resourceIds := issue.ResourceIds
			for _, resourceId := range resourceIds {
				nodes = append(nodes, irs.IID{SystemId: *resourceId})
			}
		}
		nodeGroupInfo.NodeList = nodes
	*/

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
				nodeList, errNodeList := NodeGroupHandler.GetAutoScalingGroups(autoscalingGroupName)
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
		nodeGroupTagList = make(map[string]*string)     // nil 체크 후 초기화
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

	nodeGroupInfo.RootDiskSize = strconv.FormatInt(aws.Int64Value(rootDiskSize), 10)

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

	// irs.StructToKeyValueList() 함수 사용
	nodeGroupInfo.KeyValueList = irs.StructToKeyValueList(nodeGroup)

	PrintToJson(nodeGroupInfo)
	//return irs.NodeGroupInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "추출 오류", nil)
	return nodeGroupInfo, nil
}

func convertNodeGroupStatusToNodeGroupInfoStatus(nodeGroupStatus string) irs.NodeGroupStatus {
	status := irs.NodeGroupInactive
	if strings.EqualFold(nodeGroupStatus, nodeGroupStatusCreating) {
		status = irs.NodeGroupCreating
	} else if strings.EqualFold(nodeGroupStatus, nodeGroupStatusUpdating) {
		status = irs.NodeGroupUpdating
	} else if strings.EqualFold(nodeGroupStatus, nodeGroupStatusCreateFailed) {
		status = irs.NodeGroupInactive
	} else if strings.EqualFold(nodeGroupStatus, nodeGroupStatusDeleteFailed) {
		status = irs.NodeGroupInactive
	} else if strings.EqualFold(nodeGroupStatus, nodeGroupStatusDegraded) {
		status = irs.NodeGroupInactive
	} else if strings.EqualFold(nodeGroupStatus, nodeGroupStatusDeleting) {
		status = irs.NodeGroupDeleting
	} else if strings.EqualFold(nodeGroupStatus, nodeGroupStatusActive) {
		status = irs.NodeGroupActive
	}

	return status
}

func convertClusterStatusToClusterInfoStatus(clusterStatus string) irs.ClusterStatus {
	status := irs.ClusterInactive
	if strings.EqualFold(clusterStatus, clusterStatusCreating) {
		status = irs.ClusterCreating
	} else if strings.EqualFold(clusterStatus, clusterStatusUpdating) {
		status = irs.ClusterUpdating
	} else if strings.EqualFold(clusterStatus, clusterStatusFailed) {
		status = irs.ClusterInactive
	} else if strings.EqualFold(clusterStatus, clusterStatusPending) {
		status = irs.ClusterInactive
	} else if strings.EqualFold(clusterStatus, clusterStatusDeleting) {
		status = irs.ClusterDeleting
	} else if strings.EqualFold(clusterStatus, clusterStatusActive) {
		status = irs.ClusterActive
	}

	return status
}

func (ClusterHandler *AwsClusterHandler) ListIID() ([]*irs.IID, error) {
	var iidList []*irs.IID
	input := &eks.ListClustersInput{}

	callLogInfo := GetCallLogScheme(ClusterHandler.Region, call.CLUSTER, "ListIID", "ListClusters()")
	start := call.Start()

	result, err := ClusterHandler.Client.ListClusters(input)
	callLogInfo.ElapsedTime = call.Elapsed(start)
	if err != nil {
		callLogInfo.ErrorMSG = err.Error()
		calllogger.Error(call.String(callLogInfo))
		cblogger.Error(err)
		return nil, err
	}
	calllogger.Info(call.String(callLogInfo))

	for _, clusterName := range result.Clusters {
		iid := irs.IID{SystemId: *clusterName}
		iidList = append(iidList, &iid)
	}

	return iidList, nil
}
