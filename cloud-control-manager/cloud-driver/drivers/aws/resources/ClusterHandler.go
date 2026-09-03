package resources

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
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
	"github.com/aws/aws-sdk-go/service/elb"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/sts"
	call "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/call-log"
	idrv "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces"
	irs "github.com/cloud-barista/cb-spider/cloud-control-manager/cloud-driver/interfaces/resources"
)

type AwsClusterHandler struct {
	Region      idrv.RegionInfo
	Client      *eks.EKS
	EC2Client   *ec2.EC2
	ELBClient   *elb.ELB // Classic ELB client, used to clean up Load Balancers that EKS clusters auto-create (ref: cloud-barista/cb-spider#1208)
	Iam         *iam.IAM
	StsClient   *sts.STS
	AutoScaling *autoscaling.AutoScaling
	TagHandler  *AwsTagHandler // 2024-07-18 TagHandler add
}

const (
	NODEGROUP_TAG string = "nodegroup"

	// Tag that the Kubernetes in-tree AWS cloud provider attaches to a Classic Load Balancer
	// it auto-creates for a type=LoadBalancer Service. EKS does not clean these up on cluster
	// deletion, so cb-spider has to find and remove them itself. (ref: cloud-barista/cb-spider#1208)
	k8sClusterOwnershipTagPrefix = "kubernetes.io/cluster/"
	k8sClusterOwnershipTagValue  = "owned"

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

// getServerAddress returns the Spider server address from SERVER_ADDRESS env variable
func getServerAddress() string {
	hostEnv := os.Getenv("SERVER_ADDRESS")
	if hostEnv == "" {
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

//------ Cluster Management

/*
	AWS Cluster requires a Role as a prerequisite.
	For now, roleName is set to spider-eks-role; the Role's ARN is looked up at creation time.
*/

//------ AMI Analysis Helper Functions

// eksAMITypes is the single source of truth for supported EKS AMI Type strings.
// Hardcoded due to AWS SDK v1.39.4 (2021) lacking recent AMI type constants.
// TODO: After upgrading to AWS SDK v1.50+, replace with eks.AMITypes_Values()
// Excludes deprecated AL2 types (K8s ≤1.32 only) and CUSTOM type.
var eksAMITypes = []string{
	"AL2023_x86_64_STANDARD",
	"AL2023_ARM_64_STANDARD",
	"AL2023_x86_64_NVIDIA",
	"AL2023_ARM_64_NVIDIA",
	"AL2023_x86_64_NEURON",
	"BOTTLEROCKET_ARM_64",
	"BOTTLEROCKET_x86_64",
	"BOTTLEROCKET_ARM_64_NVIDIA",
	"BOTTLEROCKET_x86_64_NVIDIA",
	"WINDOWS_CORE_2019_x86_64",
	"WINDOWS_FULL_2019_x86_64",
	"WINDOWS_CORE_2022_x86_64",
	"WINDOWS_FULL_2022_x86_64",
	"WINDOWS_CORE_2025_x86_64",
	"WINDOWS_FULL_2025_x86_64",
}

// isValidEKSAMIType reports whether s is a known EKS AMI Type string.
// EKS AMI Types are predefined constants (e.g. AL2023_x86_64_STANDARD) and must
// NOT be confused with EC2 AMI IDs (ami-xxxxxxxxxxxxxxxxx).
// When a caller passes an EKS AMI Type directly, it is forwarded to AWS EKS as-is.
func isValidEKSAMIType(s string) bool {
	for _, t := range eksAMITypes {
		if t == s {
			return true
		}
	}
	return false
}

// isEC2LaunchTemplateID reports whether s is an EC2 Launch Template ID (lt-[0-9a-f]{17}).
func isEC2LaunchTemplateID(s string) bool {
	if !strings.HasPrefix(s, "lt-") {
		return false
	}
	suffix := s[3:]
	if len(suffix) != 17 {
		return false
	}
	for _, c := range suffix {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

// getAvailableAMITypesMessage returns a human-readable list of supported EKS AMI Types.
func getAvailableAMITypesMessage() string {
	msg := "Available EKS AMI Types:\n- " + strings.Join(eksAMITypes, "\n- ")
	msg += "\n\n⚠️  Important Notes:"
	msg += "\n- AL2023 types require Kubernetes ≥ 1.30"
	msg += "\n- Bottlerocket types support all Kubernetes versions"
	msg += "\n- For Windows workloads, use WINDOWS_* types"
	msg += "\n\nRefer to AWS EKS documentation for detailed compatibility."
	return msg
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

	// AWS requires a pre-created Role. The role name is currently fixed as follows (to be improved later).
	// Example) cluster: cloud-barista-eks-cluster-role, Node: cloud-barista-eks-nodegroup-role
	eksRoleName := "cloud-barista-eks-cluster-role"
	// get or create Role Arn
	eksRole, err := ClusterHandler.getOrCreateEKSClusterRole(eksRoleName)
	if err != nil {
		cblogger.Error(err)
		// role is required.
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

	// EKS version handling (Spider input format: "1.23.4" / AWS format: "1.23")
	if reqK8sVersion != "" {
		arrVer := strings.Split(reqK8sVersion, ".")
		switch len(arrVer) {
		case 2: // use as-is
			input.Version = aws.String(reqK8sVersion)
		case 3: // take only the first two parts (normal input format)
			input.Version = aws.String(arrVer[0] + "." + arrVer[1])
		default: // for other cases, the CSP default (latest version) is applied
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

	/*// Changed from Sync Call to Async Call - Issue:#716
	//----- wait until Status=COMPLETE -----//  :  check via cluster describe .status
	errWait := ClusterHandler.WaitUntilClusterActive(result.Cluster.Identity.String())
	if errWait != nil {
		cblogger.Error(errWait)
		return irs.ClusterInfo{}, errWait
	}
	*/

	/*
		// add node groups
		clusterIID := irs.IID{NameId: clusterReqInfo.IId.NameId, SystemId: result.Cluster.Identity.String()}
		nodeGroupInfoList := clusterReqInfo.NodeGroupList
		for _, nodeGroupInfo := range nodeGroupInfoList {
			resultNodeGroupInfo, nodeGroupErr := ClusterHandler.AddNodeGroup(clusterIID, nodeGroupInfo)
			if nodeGroupErr != nil {
				cblogger.Error(err.Error())
			}
			cblogger.Debug(resultNodeGroupInfo)

		//----- wait until Status=COMPLETE -----//  :  query after all Nodegroups are created
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
		cblogger.Errorf("Failed to install EBS CSI Driver and pod identity agent: %v", csiagenterr)
		return irs.ClusterInfo{}, csiagenterr
	}
	//--- end of install CSI driver for EBS

	return clusterInfo, nil
}

// WaitUntilNodegroupActive waits until the Nodegroup reaches Active state.
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
	cblogger.Debug("=========WaitUntilNodegroupActive() done")
	return nil
}

// WaitUntilClusterActive waits until the Cluster reaches Active state.
func (ClusterHandler *AwsClusterHandler) WaitUntilClusterActive(clusterName string) error {
	cblogger.Debugf("Cluster Name : [%s]", clusterName)
	input := &eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	}

	// The AWS SDK's WaitUntilClusterActive internally implements a polling mechanism.
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
			continue // skip on error and proceed with remaining clusters
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

		// Subnet handling
		//SubnetIds: ["subnet-0d30ee6b367974a39","subnet-06d5c04b32019b81f","subnet-05c5d26bd2f014591"],
		if len(result.Cluster.ResourcesVpcConfig.SubnetIds) > 0 {
			for _, curSubnetId := range result.Cluster.ResourcesVpcConfig.SubnetIds {
				clusterInfo.Network.SubnetIIDs = append(clusterInfo.Network.SubnetIIDs, irs.IID{SystemId: *curSubnetId})
			}
		}

		// Cluster security group handling
		// ClusterSecurityGroupId: "sg-0bb02bf07fe5f42f0",
		//@TODO - This security group is auto-created at cluster creation time, so it is intentionally excluded from the CB security group list for now.
		if !reflect.ValueOf(result.Cluster.ResourcesVpcConfig.ClusterSecurityGroupId).IsNil() {
			//if *result.Cluster.ResourcesVpcConfig.ClusterSecurityGroupId != "" {
			/*
				for _, curSecurityGroupId := range result.Cluster.ResourcesVpcConfig.SecurityGroupIds {
					clusterInfo.Network.SecurityGroupIIDs = append(clusterInfo.Network.SecurityGroupIIDs, irs.IID{SystemId: *curSecurityGroupId})
				}
			*/
		}

		// Security group handling: corresponds to "additional security groups"
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
	// Using irs.StructToKeyValueList()
	clusterInfo.KeyValueList = irs.StructToKeyValueList(result.Cluster)
	clusterInfo.Network.KeyValueList = irs.StructToKeyValueList(result.Cluster.ResourcesVpcConfig)

	clusterInfo.TagList, _ = ClusterHandler.TagHandler.ListTag(irs.CLUSTER, clusterInfo.IId)

	// Node group handling
	resNodeGroupList, errNodeGroup := ClusterHandler.ListNodeGroup(clusterInfo.IId)
	if errNodeGroup != nil {
		cblogger.Error(errNodeGroup)
		return irs.ClusterInfo{}, errNodeGroup
	}

	cblogger.Debug(resNodeGroupList)

	// Node group type conversion
	for _, curNodeGroup := range resNodeGroupList {
		cblogger.Debugf("Node Group : [%s]", curNodeGroup.IId.NameId)
		curNodeGroup.KeyValueList = irs.StructToKeyValueList(curNodeGroup)
		clusterInfo.NodeGroupList = append(clusterInfo.NodeGroupList, *curNodeGroup)
	}

	// Addons handling
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
	token, _, err := ClusterHandler.generateEKSToken(*cluster.Name)
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
// Credentials are read from ~/.cb-spider/.spider-credential file at kubectl execution time
// The credentials file should contain: SPIDER_USERNAME=<username> and SPIDER_PASSWORD=<password>
func (ClusterHandler *AwsClusterHandler) getDynamicKubeConfig(clusterDesc *eks.DescribeClusterOutput) string {

	cluster := clusterDesc.Cluster

	// Get Spider server address from environment variable
	serverAddr := getServerAddress()

	// Generate kubeconfig content with exec-based dynamic token
	// Credentials are sourced from ~/.cb-spider/.spider-credential at runtime (not embedded in kubeconfig)
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
      command: sh
      args:
      - -c
      - ". ~/.cb-spider/.spider-credential && curl -s -u \"$SPIDER_USERNAME:$SPIDER_PASSWORD\" \"http://%s/spider/cluster/CLUSTER_NAME_PLACEHOLDER/token?ConnectionName=CONNECTION_NAME_PLACEHOLDER\""
`, *cluster.Endpoint, *cluster.CertificateAuthority.Data, *cluster.Name, *cluster.Name, *cluster.Name, *cluster.Name, serverAddr)

	return kubeconfigContent
}

// create EKS token using AWS STS
// generateEKSToken returns the EKS bearer token and the time it stops being accepted.
//
// The expiry is derived from the presigned URL itself (X-Amz-Date + X-Amz-Expires) rather
// than from time.Now(), so it tracks the value STS actually enforces even if the duration
// constant changes. One minute is subtracted for cushion, mirroring the official
// aws-iam-authenticator, so that a request started just before the boundary does not
// arrive after it (clock skew, network latency, signing-time drift).
// Ref: kubernetes-sigs/aws-iam-authenticator pkg/token/token.go
//
// A zero expiry is returned when it cannot be determined; callers must treat that as
// "unknown" rather than "expired".
func (ClusterHandler *AwsClusterHandler) generateEKSToken(clusterName string) (string, time.Time, error) {
	if ClusterHandler.StsClient == nil {
		return "", time.Time{}, fmt.Errorf("STS client not available")
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
		return "", time.Time{}, fmt.Errorf("failed to create presigned URL: %w", err)
	}

	encodedURL := base64.RawURLEncoding.EncodeToString([]byte(presignedURL))

	// prepend "k8s-aws-v1." to the encoded URL
	token := "k8s-aws-v1." + encodedURL

	return token, eksTokenExpiry(presignedURL), nil
}

// eksTokenExpiryCushion is subtracted from the presigned URL expiry so that a request
// started just before the boundary is not rejected on arrival. The official
// aws-iam-authenticator uses the same one-minute cushion.
const eksTokenExpiryCushion = 1 * time.Minute

// eksTokenExpiry reads X-Amz-Date and X-Amz-Expires from a presigned STS URL and returns
// when the derived token stops being usable. It returns the zero time when the URL cannot
// be parsed, which callers treat as "expiry unknown".
func eksTokenExpiry(presignedURL string) time.Time {
	u, err := url.Parse(presignedURL)
	if err != nil {
		cblogger.Errorf("Failed to parse presigned URL for token expiry: %v", err)
		return time.Time{}
	}

	q := u.Query()
	// X-Amz-Date is the signing time in the SigV4 basic format (e.g. 20260902T051500Z).
	signedAt, err := time.Parse("20060102T150405Z", q.Get("X-Amz-Date"))
	if err != nil {
		cblogger.Errorf("Failed to parse X-Amz-Date for token expiry: %v", err)
		return time.Time{}
	}

	secs, err := strconv.ParseInt(q.Get("X-Amz-Expires"), 10, 64)
	if err != nil || secs <= 0 {
		cblogger.Errorf("Failed to parse X-Amz-Expires for token expiry: %v", err)
		return time.Time{}
	}

	expiresAt := signedAt.Add(time.Duration(secs) * time.Second).Add(-eksTokenExpiryCushion)
	if !expiresAt.After(signedAt) {
		// Duration shorter than the cushion; reporting a past time would be worse than
		// reporting nothing, so treat it as unknown.
		return time.Time{}
	}
	return expiresAt
}

// GenerateClusterToken generates a token for cluster authentication
// This implements the ClusterHandler interface
func (ClusterHandler *AwsClusterHandler) GenerateClusterToken(clusterIID irs.IID) (irs.ClusterToken, error) {
	cblogger.Info("call GenerateClusterToken()")

	// For EKS, we need the cluster name to generate token
	clusterName := clusterIID.SystemId
	if clusterName == "" {
		clusterName = clusterIID.NameId
	}

	if clusterName == "" {
		return irs.ClusterToken{}, fmt.Errorf("cluster name is required for token generation")
	}

	// Generate EKS token using existing function
	token, expiresAt, err := ClusterHandler.generateEKSToken(clusterName)
	if err != nil {
		cblogger.Errorf("Failed to generate cluster token: %v", err)
		return irs.ClusterToken{}, fmt.Errorf("failed to generate cluster token: %w", err)
	}

	return irs.ClusterToken{Token: token, ExpiresAt: expiresAt}, nil
}

func (ClusterHandler *AwsClusterHandler) DeleteCluster(clusterIID irs.IID) (bool, error) {
	cblogger.Infof("Cluster Name : %s", clusterIID.SystemId)

	// Get the cluster's VPC before deletion so that, once the cluster is gone, we can look for
	// any Load Balancer(and its dedicated Security Group) that the Kubernetes in-tree AWS cloud
	// provider auto-created for a type=LoadBalancer Service. EKS never cleans these up itself.
	// (ref: https://github.com/cloud-barista/cb-spider/issues/1208)
	clusterVpcId := ClusterHandler.getClusterVpcId(clusterIID.SystemId)

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

	ClusterHandler.cleanupOwnedLoadBalancers(clusterIID.SystemId, clusterVpcId)

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

// getClusterVpcId looks up the VPC that a cluster runs in. Returns "" (not an error) if the
// cluster can't be described anymore, since DeleteCluster must still proceed in that case.
func (ClusterHandler *AwsClusterHandler) getClusterVpcId(clusterName string) string {
	descResult, err := ClusterHandler.Client.DescribeCluster(&eks.DescribeClusterInput{
		Name: aws.String(clusterName),
	})
	if err != nil {
		cblogger.Warnf("Failed to describe Cluster(%s) before deletion: %v", clusterName, err)
		return ""
	}
	if descResult.Cluster == nil || descResult.Cluster.ResourcesVpcConfig == nil || descResult.Cluster.ResourcesVpcConfig.VpcId == nil {
		return ""
	}
	return *descResult.Cluster.ResourcesVpcConfig.VpcId
}

// cleanupOwnedLoadBalancers removes any Classic Load Balancer(and its dedicated Security Group)
// that the Kubernetes in-tree AWS cloud provider auto-created in clusterVpcId for a
// type=LoadBalancer Service on this cluster. This is a best-effort cleanup that runs after the
// EKS cluster is already gone: failures are logged but never turn a successful cluster deletion
// into an error. (ref: https://github.com/cloud-barista/cb-spider/issues/1208)
func (ClusterHandler *AwsClusterHandler) cleanupOwnedLoadBalancers(clusterName string, clusterVpcId string) {
	if clusterVpcId == "" || ClusterHandler.ELBClient == nil {
		return
	}

	ownedLBNames, ownedSecurityGroupIds, err := ClusterHandler.findOwnedLoadBalancers(clusterName, clusterVpcId)
	if err != nil {
		cblogger.Errorf("Failed to search Load Balancers auto-created by Cluster(%s): %v", clusterName, err)
		return
	}

	for _, lbName := range ownedLBNames {
		if _, err := ClusterHandler.ELBClient.DeleteLoadBalancer(&elb.DeleteLoadBalancerInput{
			LoadBalancerName: aws.String(lbName),
		}); err != nil {
			cblogger.Errorf("Failed to delete Load Balancer(%s) auto-created by Cluster(%s): %v", lbName, clusterName, err)
		} else {
			cblogger.Infof("Deleted Load Balancer(%s) auto-created by Cluster(%s)", lbName, clusterName)
		}
	}

	if ClusterHandler.EC2Client == nil {
		return
	}
	for _, sgId := range ownedSecurityGroupIds {
		if err := loopDeleteSecurityGroup(ClusterHandler.EC2Client, &ec2.DeleteSecurityGroupInput{
			GroupId: aws.String(sgId),
		}); err != nil {
			cblogger.Errorf("Failed to delete Security Group(%s) used by Cluster(%s)'s Load Balancer: %v", sgId, clusterName, err)
		} else {
			cblogger.Infof("Deleted Security Group(%s) used by Cluster(%s)'s Load Balancer", sgId, clusterName)
		}
	}
}

// findOwnedLoadBalancers returns the names and Security Group IDs of every Classic Load Balancer
// in clusterVpcId that is tagged "kubernetes.io/cluster/<clusterName>: owned" — the tag the
// Kubernetes in-tree AWS cloud provider sets on Load Balancers it creates for that cluster.
func (ClusterHandler *AwsClusterHandler) findOwnedLoadBalancers(clusterName string, clusterVpcId string) ([]string, []string, error) {
	securityGroupsByLBName := map[string][]string{}

	input := &elb.DescribeLoadBalancersInput{}
	for {
		result, err := ClusterHandler.ELBClient.DescribeLoadBalancers(input)
		if err != nil {
			return nil, nil, err
		}
		for _, lb := range result.LoadBalancerDescriptions {
			if lb.LoadBalancerName == nil || lb.VPCId == nil || *lb.VPCId != clusterVpcId {
				continue
			}
			var sgIds []string
			for _, sgId := range lb.SecurityGroups {
				if sgId != nil {
					sgIds = append(sgIds, *sgId)
				}
			}
			securityGroupsByLBName[*lb.LoadBalancerName] = sgIds
		}
		if result.NextMarker == nil || *result.NextMarker == "" {
			break
		}
		input.Marker = result.NextMarker
	}

	if len(securityGroupsByLBName) == 0 {
		return nil, nil, nil
	}

	candidateNames := make([]string, 0, len(securityGroupsByLBName))
	for name := range securityGroupsByLBName {
		candidateNames = append(candidateNames, name)
	}

	ownershipTagKey := k8sClusterOwnershipTagPrefix + clusterName

	var ownedNames []string
	var ownedSecurityGroupIds []string
	seenSecurityGroupIds := map[string]bool{}

	// DescribeTags accepts at most 20 Load Balancer names per call.
	const describeTagsBatchSize = 20
	for i := 0; i < len(candidateNames); i += describeTagsBatchSize {
		end := i + describeTagsBatchSize
		if end > len(candidateNames) {
			end = len(candidateNames)
		}
		batch := candidateNames[i:end]
		names := make([]*string, len(batch))
		for j, name := range batch {
			names[j] = aws.String(name)
		}

		tagResult, err := ClusterHandler.ELBClient.DescribeTags(&elb.DescribeTagsInput{LoadBalancerNames: names})
		if err != nil {
			return nil, nil, err
		}

		for _, tagDesc := range tagResult.TagDescriptions {
			if tagDesc.LoadBalancerName == nil {
				continue
			}
			for _, tag := range tagDesc.Tags {
				if tag.Key == nil || *tag.Key != ownershipTagKey {
					continue
				}
				if tag.Value == nil || *tag.Value != k8sClusterOwnershipTagValue {
					continue
				}
				ownedNames = append(ownedNames, *tagDesc.LoadBalancerName)
				for _, sgId := range securityGroupsByLBName[*tagDesc.LoadBalancerName] {
					if !seenSecurityGroupIds[sgId] {
						seenSecurityGroupIds[sgId] = true
						ownedSecurityGroupIds = append(ownedSecurityGroupIds, sgId)
					}
				}
				break
			}
		}
	}

	return ownedNames, ownedSecurityGroupIds, nil
}

// ------ NodeGroup Management

/*
Uses the same subnets as Cluster.NetworkInfo.
On AddNodeGroup, the target Cluster info is retrieved and applied.
If a different Subnet is required for a NodeGroup, discuss separately.
//https://github.com/cloud-barista/cb-spider/wiki/Provider-Managed-Kubernetes-and-Driver-API
*/
func (ClusterHandler *AwsClusterHandler) AddNodeGroup(clusterIID irs.IID, nodeGroupReqInfo irs.NodeGroupInfo) (irs.NodeGroupInfo, error) {
	// validation check
	if nodeGroupReqInfo.MaxNodeSize < 1 { // MaxNodeSize must be at least 1
		return irs.NodeGroupInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "The MaxNodeSize value must be greater than or equal to 1.", nil)
	}
	// AWS EKS cannot create a managed node group with autoscaling disabled.
	// Reject it explicitly rather than silently ignoring the requested value.
	// https://github.com/cloud-barista/cb-spider/issues/1496
	if !nodeGroupReqInfo.OnAutoScaling {
		return irs.NodeGroupInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST,
			"OnAutoScaling=false is not supported in AWS EKS. "+
				"A managed node group is always backed by an Auto Scaling group. "+
				"Please set OnAutoScaling=true and use Min/Max/Desired node size to control the node count.", nil)
	}

	// get or create Role Arn
	eksRoleName := "cloud-barista-eks-nodegroup-role"
	eksRole, err := ClusterHandler.getOrCreateEKSNodeGroupRole(eksRoleName)
	if err != nil {
		cblogger.Error(err)
		// role is required.
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
		subnetId := subnet.SystemId // copy to a local variable; appending subnet.SystemId directly would capture the loop pointer
		subnetList = append(subnetList, &subnetId)
	}

	cblogger.Debug("Final Subnet List")
	// Iterate over subnet IDs and enable Auto-assign public IPv4 address via ModifySubnetAttribute.
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
			// return irs.NodeGroupInfo{}, errors.New(errmsg) // commented out to continue processing remaining subnets in the loop
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

	receivedImageName := nodeGroupReqInfo.ImageIID.SystemId
	cblogger.Infof("Received ImageIID.SystemId: '%s'", receivedImageName)

	var amiType string
	var useLaunchTemplate bool
	var launchTemplateID *string
	var launchTemplateVersion *string

	if receivedImageName == "" || strings.EqualFold(receivedImageName, "default") {
		// Preserve existing behavior.
		cblogger.Info("No ImageName provided (empty or 'default'), using default: AL2023_x86_64_STANDARD")
		amiType = "AL2023_x86_64_STANDARD"
	} else if isValidEKSAMIType(receivedImageName) {
		// Preserve existing behavior.
		cblogger.Infof("EKS AMI Type directly specified: %s", receivedImageName)
		amiType = receivedImageName
	} else if isEC2LaunchTemplateID(receivedImageName) {
		// Launch Template ID provided directly — reference as-is without creating a new one.
		// The Launch Template must include bootstrap UserData; cb-spider does not create or modify it.
		cblogger.Infof("Launch Template ID detected: %s — referencing directly", receivedImageName)
		amiType = "CUSTOM"
		useLaunchTemplate = true
		launchTemplateID = aws.String(receivedImageName)
		launchTemplateVersion = aws.String("$Latest")
	} else {
		// Unsupported format: raw AMI ID (ami-xxx) or unrecognized string.
		// Automatic mapping from AMI ID to EKS AMI type has been removed.
		// Direct AMI ID usage via Launch Template auto-creation is planned as a separate issue.
		return irs.NodeGroupInfo{}, fmt.Errorf(
			"unsupported ImageIID value: '%s'.\n\n"+
				"Supported formats:\n"+
				"  1. EKS AMI Type identifier (e.g. AL2023_x86_64_STANDARD)\n"+
				"  2. EC2 Launch Template ID (e.g. lt-0123456789abcdef0)\n"+
				"  3. Empty string or 'default' (uses AL2023_x86_64_STANDARD)\n\n"+
				"Raw EC2 AMI IDs (ami-xxx) are not directly supported.\n"+
				"To use a custom AMI, create an EC2 Launch Template with the AMI ID\n"+
				"and bootstrap UserData, then pass its ID as ImageIID.SystemId.\n\n%s",
			receivedImageName, getAvailableAMITypesMessage())
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

	// Set optional fields beyond the required ones
	rootDiskSize, _ := strconv.ParseInt(nodeGroupReqInfo.RootDiskSize, 10, 64)
	if rootDiskSize > 0 {
		input.DiskSize = aws.Int64(rootDiskSize)
	}

	if useLaunchTemplate {
		input.LaunchTemplate = &eks.LaunchTemplateSpecification{
			Id:      launchTemplateID,
			Version: launchTemplateVersion,
		}
		// DiskSize cannot be combined with LaunchTemplate.
		input.DiskSize = nil
	}

	if !strings.EqualFold(nodeGroupReqInfo.VMSpecName, "") {
		var nodeSpec []string
		nodeSpec = append(nodeSpec, nodeGroupReqInfo.VMSpecName) //"p2.xlarge"
		input.InstanceTypes = aws.StringSlice(nodeSpec)
	}

	cblogger.Debug(input)

	result, err := ClusterHandler.Client.CreateNodegroup(input) // async
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
			cblogger.Errorf("Failed to install EBS CSI driver: %v", errCsiDriver)
		}
	}

	var errPodIdentityAgent error
	if podIdentityAgentExists == false {
		errPodIdentityAgent = ClusterHandler.installPodIdentityAgent(clusterID)
		if errPodIdentityAgent != nil {
			cblogger.Errorf("Failed to install pod identity agent: %v", errPodIdentityAgent)
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
		cblogger.Errorf("Failed to install EBS CSI Driver: %v", err)
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
		cblogger.Errorf("Failed to install pod identity agent: %v", err)
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
AWS EKS has no API to enable/disable autoscaling for a managed node group:
a managed node group is always backed by an Auto Scaling group, and
NodegroupScalingConfig carries only desiredSize/minSize/maxSize.

Therefore an "on" request is a no-op success, and an "off" request is rejected
with an explicit error so that the caller is made aware of the AWS behavior
instead of receiving a silent (false, nil).
https://github.com/cloud-barista/cb-spider/issues/1496
*/
func (ClusterHandler *AwsClusterHandler) SetNodeGroupAutoScaling(clusterIID irs.IID, nodeGroupIID irs.IID, on bool) (bool, error) {
	cblogger.Infof("Cluster SystemId : [%s] / NodeGroup SystemId : [%s] / OnAutoScaling : [%t]", clusterIID.SystemId, nodeGroupIID.SystemId, on)

	if !on {
		err := awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST,
			"Disabling autoscaling is not supported in AWS EKS. "+
				"A managed node group is always backed by an Auto Scaling group. "+
				"Please use the ChangeNodeGroupScaling API to adjust Min/Max/Desired node size instead.", nil)
		cblogger.Error(err)
		return false, err
	}

	// Already on; nothing to change.
	return true, nil
}

func (ClusterHandler *AwsClusterHandler) ChangeNodeGroupScaling(clusterIID irs.IID, nodeGroupIID irs.IID,
	DesiredNodeSize int, MinNodeSize int, MaxNodeSize int) (irs.NodeGroupInfo, error) {
	cblogger.Infof("Cluster SystemId : [%s] / NodeGroup SystemId : [%s] / DesiredNodeSize : [%d] / MinNodeSize : [%d] / MaxNodeSize : [%d]", clusterIID.SystemId, nodeGroupIID.SystemId, DesiredNodeSize, MinNodeSize, MaxNodeSize)

	// Retrieve cluster info by clusterIID
	// Retrieve nodeGroup info by nodeGroupIID
	// 		nodeGroup contains the AutoScaling group name

	// TODO : refactor into a shared helper
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

// UpgradeCluster upgrades the K8s version of both the Control Plane and Node Groups (worker nodes).
func (ClusterHandler *AwsClusterHandler) UpgradeCluster(clusterIID irs.IID, newVersion string) (irs.ClusterInfo, error) {
	cblogger.Infof("Cluster SystemId : [%s] / Request New Version : [%s]", clusterIID.SystemId, newVersion)

	// Retrieve current cluster info and compare versions
	currentClusterInfo, err := ClusterHandler.GetCluster(clusterIID)
	if err != nil {
		cblogger.Errorf("Failed to get current cluster info: %v", err)
		return irs.ClusterInfo{}, err
	}

	currentVersion := currentClusterInfo.Version
	cblogger.Infof("Current cluster version: %s, Target version: %s", currentVersion, newVersion)

	// Check whether the Control Plane needs an upgrade
	needsControlPlaneUpgrade := currentVersion != newVersion

	// List all Node Groups to determine which ones need an upgrade
	nodeGroups, err := ClusterHandler.ListNodeGroup(clusterIID)
	if err != nil {
		cblogger.Errorf("Failed to list node groups: %v", err)
		return irs.ClusterInfo{}, err
	}

	// Determine whether each Node Group needs an upgrade
	needsNodeGroupUpgrade := false
	for _, nodeGroup := range nodeGroups {
		// Check the Node Group version
		nodeGroupVersion := ""
		for _, kv := range nodeGroup.KeyValueList {
			if kv.Key == "Version" {
				nodeGroupVersion = kv.Value
				break
			}
		}

		// If the version cannot be determined or differs from the target version
		if nodeGroupVersion == "" || nodeGroupVersion != newVersion {
			needsNodeGroupUpgrade = true
			cblogger.Infof("Node group %s needs upgrade from version %s to %s",
				nodeGroup.IId.SystemId, nodeGroupVersion, newVersion)
		}
	}

	// Early return if no upgrade is needed
	if !needsControlPlaneUpgrade && !needsNodeGroupUpgrade {
		cblogger.Info("Both control plane and all node groups are already at target version. No upgrade needed.")
		return currentClusterInfo, nil
	}

	if needsControlPlaneUpgrade {
		cblogger.Infof("Control plane needs upgrade from version %s to %s", currentVersion, newVersion)
	} else {
		cblogger.Info("Control plane is already at target version. Only node groups will be upgraded.")
	}

	// Upgrade the Control Plane
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

		// Wait for Control Plane upgrade to complete
		cblogger.Info("Waiting for control plane upgrade to complete...")
		errWait := ClusterHandler.WaitUntilClusterActive(clusterIID.SystemId)
		if errWait != nil {
			cblogger.Errorf("Failed to wait for cluster to become active after control plane upgrade: %v", errWait)
			// Continue with Node Group upgrades even if waiting for Control Plane fails
		} else {
			cblogger.Info("Control plane upgrade completed successfully")
		}

		// Additional wait time after Control Plane upgrade
		cblogger.Info("Control plane marked as ACTIVE, waiting additional time for version propagation...")
		time.Sleep(120 * time.Second) // wait 2 minutes

		// Re-verify Control Plane version
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
				// Proceed even if the version does not match
			}
		}
	}

	// Check for active operations before Node Group upgrade and add exponential backoff retry logic
	maxRetries := 10
	retryInterval := 60 // seconds
	backoffFactor := 1.5

	currentInterval := retryInterval
	for i := 0; i < maxRetries; i++ {
		// Check if the cluster is in a state ready for Node Group upgrade
		clusterStatus, err := ClusterHandler.checkClusterOperationStatus(clusterIID)
		if err != nil {
			cblogger.Errorf("Error checking cluster status: %v", err)
			break // continue with Node Group upgrade even if status check fails
		}

		if clusterStatus == "ACTIVE" {
			break // proceed with node group upgrade
		}

		if i == maxRetries-1 {
			cblogger.Warnf("Cluster not in ACTIVE state after %d retries. Proceeding with node group upgrades anyway.", maxRetries)
			break
		}

		cblogger.Infof("Cluster is in %s state, waiting %d seconds before retrying (%d/%d)",
			clusterStatus, currentInterval, i+1, maxRetries)
		time.Sleep(time.Duration(currentInterval) * time.Second)

		// Apply exponential backoff
		currentInterval = int(float64(currentInterval) * backoffFactor)
	}

	// Upgrade Node Groups
	if needsNodeGroupUpgrade {
		cblogger.Info("Starting node group upgrades...")

		// Track failed Node Group upgrades
		upgradeFailures := []string{}

		for _, nodeGroup := range nodeGroups {
			cblogger.Infof("Upgrading node group: %s to version: %s", nodeGroup.IId.SystemId, newVersion)

			// Retry logic for Node Group upgrade
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
					// Upgrade request succeeded
					callogger.Info(call.String(nodeGroupCallLogInfo))
					cblogger.Infof("Node group update initiated for %s: %s",
						nodeGroup.IId.SystemId, *nodeGroupUpdateResult.Update.Id)
					break
				}

				// Log on error
				nodeGroupCallLogInfo.ErrorMSG = updateErr.Error()
				callogger.Info(call.String(nodeGroupCallLogInfo))
				cblogger.Errorf("Attempt %d: Failed to upgrade node group %s: %v",
					retry+1, nodeGroup.IId.SystemId, updateErr)

				if retry < maxNodeGroupRetries-1 {
					retryDelay := (retry + 1) * 60 // progressive increase: 60s, 120s, ...
					cblogger.Infof("Retrying node group upgrade in %d seconds (%d/%d)",
						retryDelay, retry+1, maxNodeGroupRetries)
					time.Sleep(time.Duration(retryDelay) * time.Second)
				}
			}

			if updateErr != nil {
				// Failed after all retries
				cblogger.Errorf("Failed to upgrade node group %s after %d attempts: %v",
					nodeGroup.IId.SystemId, maxNodeGroupRetries, updateErr)
				upgradeFailures = append(upgradeFailures,
					fmt.Sprintf("NodeGroup %s: %v", nodeGroup.IId.SystemId, updateErr))
				continue // continue upgrading remaining node groups
			}

			// Upgrade node groups sequentially one by one (no parallel processing)
			// Use a shorter timeout than the Control Plane for Node Group upgrade
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

		// Aggregate and log upgrade failures
		if len(upgradeFailures) > 0 {
			cblogger.Warnf("Some node groups failed to upgrade: %s", strings.Join(upgradeFailures, "; "))
		} else {
			cblogger.Info("All node group upgrades completed successfully")
		}
	} else {
		cblogger.Info("No node groups need upgrading")
	}

	// Verify final state and return updated cluster info
	cblogger.Info("Fetching updated cluster information")
	updatedClusterInfo, err := ClusterHandler.GetCluster(clusterIID)
	if err != nil {
		cblogger.Errorf("Failed to get updated cluster info: %v", err)
		return irs.ClusterInfo{}, err
	}

	// Verify upgrade result
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
Convert EKS NodeGroup info to Spider NodeGroup
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

	//nodes := nodeGroup.Health.Issues[0].ResourceIds // may only contain nodes with issues
	rootDiskSize := nodeGroup.DiskSize
	//nodeGroup.Taints // unused
	nodeGroupTagList := nodeGroup.Tags
	scalingConfig := nodeGroup.ScalingConfig
	//nodeGroup.RemoteAccess
	nodeGroupName := nodeGroup.NodegroupName

	//nodeGroup.LaunchTemplate // unused
	//clusterName := nodeGroup.ClusterName
	//capacityType := nodeGroup.CapacityType // "ON_DEMAND"
	nodeGroupInfo.ImageIID.NameId = *nodeGroup.AmiType // AL2_x86_64"
	//createTime := nodeGroup.CreatedAt
	//health := nodeGroup.Health // Code, Message, ResourceIds	// ,"Health":{"Issues":[{"Code":"NodeCreationFailure","Message":"Unhealthy nodes in the kubernetes cluster",
	//labelList := nodeGroup.Labels
	//nodeGroupArn := nodeGroup.NodegroupArn
	//nodeGroupResources := nodeGroup.Resources
	//nodeGroupResources.AutoScalingGroups // unused
	//nodeGroupResources.RemoteAccessSecurityGroup // unused

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
	// Query VM nodes
	//=============
	// Extract VM list from Auto Scaling group list
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

	// AWS EKS: a managed node group is always backed by an Auto Scaling group.
	// EKS exposes no API to toggle autoscaling on/off -- NodegroupScalingConfig
	// carries only desiredSize/minSize/maxSize -- so this is always reported as true.
	// https://github.com/cloud-barista/cb-spider/issues/1496
	//
	// Note: "true" means the node group is ASG-backed and scalable within Min/Max,
	// not that a Cluster Autoscaler is currently running in the cluster.
	nodeGroupInfo.OnAutoScaling = true

	if nodeGroupTagList == nil {
		nodeGroupTagList = make(map[string]*string)     // initialize after nil check
		nodeGroupTagList[NODEGROUP_TAG] = nodeGroupName // if no value, set to nodeGroupName
	}
	nodeGroupTag := ""
	for key, val := range nodeGroupTagList {
		if val == nil {
			continue
		}
		if strings.EqualFold(key, "value") || strings.EqualFold(key, NODEGROUP_TAG) {
			nodeGroupTag = *val
			break
		}
		cblogger.Debug(key, *val)
	}
	//printToJson(nodeGroupTagList)
	cblogger.Debug("nodeGroupName=", *nodeGroupName)
	cblogger.Debug("tag=", nodeGroupTagList[NODEGROUP_TAG])
	nodeGroupInfo.IId = irs.IID{
		NameId:   nodeGroupTag, // name from TAG
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

	// TODO : should node list be queried by NodegroupArn?
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

	// Using irs.StructToKeyValueList()
	nodeGroupInfo.KeyValueList = irs.StructToKeyValueList(nodeGroup)

	PrintToJson(nodeGroupInfo)
	//return irs.NodeGroupInfo{}, awserr.New(CUSTOM_ERR_CODE_BAD_REQUEST, "extraction error", nil)
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
